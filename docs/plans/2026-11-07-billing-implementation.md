# Billing, plan limits, email verification, team invites and legal pages

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Free / Starter / Team plans paid through Stripe's hosted pages, with
limits enforced in Go, verified sign-up email, Team invites, and public legal
pages, per `docs/design/2026-11-07-billing.md`.

**Architecture:** The org row carries its plan and Stripe state. Stripe is the
source of truth and reaches us only by signed webhook. Checkout and the
Customer Portal are Stripe-hosted, so no card data touches this server. Limit
checks are local SQL reads inside existing requests. Nothing here calls Jev.

**Tech stack:** Go (net/http, pgx), PostgreSQL migration, React/TypeScript,
static HTML legal pages. Stripe is called over plain HTTPS with a hand-written
client (two form POSTs and HMAC-SHA256 webhook verification) instead of
`stripe-go`: the alternative is about 150 lines, so the dependency is not
justified (AGENTS.md Style).

Lane C (payments, sign-up, access). Branch `billing/v1`, worktree
`../sitewise-billing`.

## Decisions taken in this plan

- **Billing off when no Stripe key** in development: limits still apply
  (Free unless an operator sets a plan), Upgrade reports "billing is not set
  up". Production refuses to start without the Stripe settings.
- **Effective plan** = stored plan when status is `active`, `trialing` or
  `past_due` (Stripe is still retrying); otherwise Free. An operator command
  `sitewise set-plan` exists for comped accounts and tests.
- **Documents read** counts every upload that creates a new document or
  invoice (re-dropping the same file does not count). Counter table per org
  per month, so deleting does not refund the count.
- **Storage** = sum of `files.byte_size` for the org. Checked before an
  upload; one upload may cross the cap (its size is unknown until stored).
- **Concurrency:** two parallel uploads may both pass a check at the cap.
  Accepted: overshoot is at most the number of parallel uploads.
- **Unverified email:** can sign in, browse and create a project, but cannot
  upload until verified. Existing users are marked verified by the migration.
- **Terms:** sign-up requires ticking acceptance; version and time stored.
- **Team:** owner invites by email; invitee sets a password from the link and
  joins the owner's org as `member`. An email that already has a SiteWise
  account cannot be invited (one person, one org in v1). Seats = members +
  open invites ≤ plan cap. Only owners manage billing and team.
- **Over limit:** HTTP 402 with a plain sentence; the web shows it with an
  "See plans" link.

## Tasks

### Task 1: Migration 045_billing.sql
Files: create `internal/db/migrations/045_billing.sql`.
- `orgs`: `plan text NOT NULL DEFAULT 'free' CHECK (plan IN ('free','starter','team'))`,
  `billing_status text NOT NULL DEFAULT ''`, `billing_interval text NOT NULL DEFAULT ''`,
  `paid_until timestamptz`, `cancel_at_period_end boolean NOT NULL DEFAULT false`,
  `stripe_customer_id text UNIQUE`, `stripe_subscription_id text`,
  `billing_event_at timestamptz`.
- `users`: `email_verified_at timestamptz`, `terms_accepted_at timestamptz`,
  `terms_version text NOT NULL DEFAULT ''`; set `email_verified_at = created_at`
  for existing rows.
- `email_verifications` (org_id, id, user_id, token_hash unique, expires_at,
  used_at) like `password_resets`.
- `org_usage (org_id, month date, documents int, PK (org_id, month))`.
- `stripe_events (id text PRIMARY KEY, type text, received_at timestamptz)`.
Test: `go test ./internal/db/...` migrations apply on `sitewise_test`.

### Task 2: `internal/billing` plans
Files: create `internal/billing/plans.go`, `plans_test.go`.
`Plan{ID, Name, MonthlyCents, AnnualCents, Projects, DocumentsPerMonth, StorageBytes, Users}`
for free/starter/team (0 = unlimited for projects only). `Lookup(id)`,
`Effective(plan, status) string`. Tests: unknown plan → free; past_due keeps
plan; canceled/unpaid/incomplete → free.

### Task 3: `internal/billing` Stripe client
Files: create `internal/billing/stripe.go`, `stripe_test.go`.
- `Client{Key, BaseURL, HTTP}`; `Checkout(ctx, CheckoutRequest) (url, error)`
  posts `/v1/checkout/sessions` (mode=subscription, price, quantity 1,
  client_reference_id=org, subscription_data[metadata][org_id], customer or
  customer_email, tax rate, billing_address_collection=required,
  tax_id_collection[enabled]=true, success/cancel URLs).
  `Portal(ctx, customer, returnURL)` posts `/v1/billing_portal/sessions`.
- `VerifySignature(payload, header, secret, now, tolerance)`: parse `t=`,
  `v1=`; HMAC-SHA256 of `t.payload`; constant-time compare; reject outside 5 min.
- `ParseEvent(payload) (Event, error)` for `checkout.session.completed` and
  `customer.subscription.{created,updated,deleted}`.
Tests: httptest fake Stripe checks form fields and auth header; signature
valid / wrong secret / stale / malformed.

### Task 4: Store
Files: create `internal/store/billing.go`, `billing_test.go`.
`OrgBilling(org)`, `Usage(org, now)` (documents this month, storage bytes,
projects, seats incl. open invites), `CountDocument(org, now)`,
`SetPlan(org, plan)`, `ApplyStripe(evt)` (idempotent by event id; ignores
events older than `billing_event_at`; finds org by metadata org id then
customer id), `Role(org, user)`. Tests on `sitewise_test`: isolation between
orgs, duplicate event ignored, older event ignored, deleted sub → free.

### Task 5: Config
Files: modify `internal/config/config.go`, `config_test.go`.
`SITEWISE_STRIPE_SECRET_KEY`, `SITEWISE_STRIPE_WEBHOOK_SECRET`,
`SITEWISE_STRIPE_TAX_RATE`, `SITEWISE_STRIPE_PRICE_STARTER_MONTHLY`,
`..._STARTER_ANNUAL`, `..._TEAM_MONTHLY`, `..._TEAM_ANNUAL`. All required in
production; none in development. Secrets never printed.

### Task 6: HTTP billing routes
Files: create `internal/httpapi/billing.go`, `billing_test.go`; modify
`projects.go` (Deps, routes, routePaths), `bench/budgets.json`.
- `GET /billing` (member): plan, status, paid until, limits, usage, role,
  billing enabled.
- `POST /billing/checkout` (owner, origin): `{plan, interval}` → `{url}`.
- `POST /billing/portal` (owner, origin) → `{url}`; 409 when no customer.
- `POST /stripe/webhook`: no origin or session check; signature required;
  64 KiB cap; 400 on bad signature; 200 on duplicates.
- Helper `checkUpload(ctx, session)` → 403 unverified, 402 over documents or
  storage; `checkProject` → 402 over projects.
Budgets: `billing_read` and `stripe_webhook` p50 100 / p90 250 ms;
`billing_session` p50 800 / p90 1500 ms (calls Stripe, not a hot path).

### Task 7: Enforce limits
Files: modify `documents.go` (uploadFile), `submissions.go`
(uploadSubmission), `program_import.go` (postProgramImport), `invoices.go`
(uploadInvoice), `projects.go` (createProject). Check before reading the
body; count after a created document. Tests: Free org's 51st upload → 402;
re-drop of same file not counted; unverified → 403; second project on Free →
402; another org unaffected.

### Task 8: Email verification and terms
Files: modify `internal/auth/handlers.go` (VerifyMailer + SMTP/Sink/Dev),
`internal/store/accounts.go`, `internal/httpapi/accounts.go`.
Sign-up requires `accept_terms: true`, stores version `2026-11-07`, sends a
verify link `/#verify=<token>` (24 h). `POST /email/verify {token}`,
`POST /email/verify/resend` (session; 3/hour). `GET /session` reports
`email_verified`. Tests in `internal/auth/accounts_test.go`.

### Task 9: Team invites
Files: create `internal/httpapi/team.go`, `internal/store/team.go`, tests.
`GET /team`, `POST /team/invites {email}`, `DELETE /team/invites/{id}`,
`DELETE /team/members/{user}` (owner only; not self), `POST /invites/accept
{token, password, accept_terms}` (public, origin, rate-limited). Link
`/#invite=<token>`, 7 days. Seat cap from plan. Tests: non-owner 403; seat
cap 402; email with an account → 409; accepted user lands in owner's org;
removed member's sessions end.

### Task 10: Wiring and operator command
Files: modify `cmd/sitewise/main.go`. Build billing client from config; pass
mailers; add `set-plan -org <id> -plan <free|starter|team>`.

### Task 11: Web
Files: modify `web/src/api.ts`, `SignIn.tsx`, `App.tsx`; create
`web/src/Account.tsx`, `web/src/account.css`.
- Sign-up: terms checkbox with links; `#verify=` and `#invite=` handling.
- `/account`: plan card, usage bars, Upgrade (monthly/annual) for owners,
  Manage billing, Team section (Team plan), verify banner + resend.
- Upload 402/403 messages show the server's sentence and a link to `/account`.

### Task 12: Legal and pricing pages
Files: create `web/public/legal/{terms,privacy,refunds,pricing}.html`,
`web/public/legal/legal.css`. Plain HTML served as static files (no script;
fits the CSP). Australian law: ACL consumer guarantees, Privacy Act 1988 and
APPs incl. APP 8 overseas disclosure, Notifiable Data Breaches, Spam Act,
NSW governing law, professional-advice disclaimer. Placeholders in
`[BRACKETS]` for entity, ABN, address and contact. Lawyer review required
before launch (listed in the evidence note).

### Task 13: Docs and checks
Files: modify `docs/operations.md` (Stripe setup, webhook, env vars),
`deploy/sitewise.env.example`, design note status; create
`docs/evidence/2026-11-07-billing.md`. Run `go vet ./...`, the touched
packages' tests with `-p 1`, `npm --prefix web run build`, and the knowledge
check if touched (it is not).

## Done
Test mode: Free user verifies email, uploads up to the cap, gets a 402,
upgrades through Checkout, webhook raises limits; failed renewal → Free with
data readable; forged webhook refused; Team owner invites a colleague who joins
the org; legal pages load at `/legal/*.html`.
