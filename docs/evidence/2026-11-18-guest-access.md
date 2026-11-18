# Guest access by link: what was built and measured

Date: 18 November 2026. Branch `wave/guests`, from local `main` at 158326f
(not merged, not deployed). Plan: `docs/plans/2026-11-17-guest-access.md`.
Lane C. Everything here was run on the development machine against a
private PostgreSQL 17 schema (`wave_guests` in `sitewise_test`); the live
server was not contacted and no live model or Jev call was made. The
chat key's new `shares` group is draft and not owner-reviewed. The plan
asks for a security review before this ships; none has been done.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. Migration | `067_guest_grants.sql`: one link to one thing (`documents`, `report`, `package`), token kept only as its SHA-256 (unique), no `respond` column, no item grants. Five composite keys; a package link cascades with its invitation and follows a corrected firm (`ON UPDATE CASCADE`). The change log accepts `via 'guest'`. The FK sweep inventories the five keys, and its fixtures now insert `package_invitations` (with a spare firm for that table's own control). | `internal/db/migrations/067_guest_grants.sql`, `internal/db/migrate_fk_sweep_test.go`, `internal/db/testdata/next_wave_foreign_keys.json` |
| 2. Store: grants | `CreateGuestGrant` in one transaction under the project lock: project live and not archived; 1 to 200 documents all filed in this project; a report with an issue; a live package with the firm invited and an issued RFP or RFT; life 1 to 180 days (default 30, a package 21); `Replaces` revokes an older link in the same step. Anything outside the project is `ErrNotFound` without saying which. `GuestGrantByToken` is one query; unknown, revoked, expired, archived project and retired package are all `ErrGuestGone`. `ListGuestGrants`, `RevokeGuestGrant` (row lock), `TouchGuestGrant`, `ShareContext`. | `internal/store/guest.go` |
| 3. Store: what a grant may fetch | `GuestPage` in one read-only transaction; `GuestDocumentFile` admits a documents link's ids, or a package link's transmittal ids from any issue of its RFP or RFT; `GuestIssueFile` admits only an issued version. | same |
| 4. Store: the quote | `OpenSubmissionFor` split into a wrapper and `openSubmissionTx` (no behaviour change). `OpenGuestQuote` rechecks the grant `FOR SHARE` under the project lock, refuses an awarded package, never re-points a document that existed before unless it is already this firm's quote here, places it with the firm preset as a PM drop on that cell, and writes one `project_changes` row (via `guest`, actor the grant id, area `procurement`, op `guest_quote`). | `internal/store/submissions.go`, `internal/store/guest.go` |
| 5. HTTP: guest routes | Five routes under `/guest`. `guestGrant` is the one check: it reads only `sitewise_guest`, hashes it, finds the live grant on every request, and takes org, project and target from the grant. Exchange `POST /guest/session`; page `GET /guest`; downloads through one `serveFile` writer; `POST /guest/quote` (type by name before a byte, 50 MB, 20 a link an hour, `orgUploadAllowed`, upload, `OpenGuestQuote`, delete before filing if the link went, then `Filer.Start` or `ProposalReader.Start` exactly as `uploadSubmission`). | `internal/httpapi/guest.go`, `internal/httpapi/billing.go`, `internal/httpapi/accounts.go` |
| 6. HTTP: share routes and the board | `POST`, `GET`, `DELETE /projects/{id}/shares[/{share}]` through `deliverySession`, timed as `team`. Create answers `{share, link}` once; 100 creates per org an hour. `DELETE` is the operation `shares.revoke`, `Click`. The board's cards gain `linked`, `link_opened_on` and `tender_issued`; link changes and a guest's first open write a procurement event so the PM's cell moves live. | `internal/httpapi/shares.go`, `internal/httpapi/ops_shares.go`, `internal/store/comparison.go` |
| 7. Isolation and behaviour tests | `TestGuestReachesOnlyItsGrant` (every route pattern, via `export_test.go`), `TestGuestGrantScope`, `TestGuestGrantLife`, `TestGuestFetchRules`, `TestOpenGuestQuote`, `TestGuestRevokedMidUpload`, `TestGuestQuoteIsAPMDrop`, `TestGuestUploadLimits`, `TestGuestsAreNotSeatsAndCountTowardTheCap`; the share routes join `TestNextWaveRouteAccessSweep`. | `internal/httpapi/guest_*_test.go`, `access_sweep_test.go` |
| 8. Web: Share | `ShareDialog.tsx` on `ConfirmDialog` (plain tone): Shared with, Link works until, Create link, then the link and Copy; a warning when the newest issue prints the budget; everyone with a link (by whom, opens, last opened, works until, Revoke). One `ShareHost` per project registers it with `chatView.ts` as `share`; the register's right-click menu (Share…), Issued reports (Share…) and an invited firm's cell open it the same way chat does. The cell reads "Invited · link opened 14 Nov". | `web/src/ShareDialog.tsx`, `Register.tsx`, `Reports.tsx`, `Procurement.tsx`, `Project.tsx`, `Chat.tsx` |
| 9. Web: the guest page | `GuestPage.tsx`, its own chunk, rendered by `main.tsx` at `/g` before `App` mounts: posts the fragment token once, clears it from the address bar, then shows the band (project · from · to · works until), issues newest first, documents, and for a firm "Your quote" with a drop zone and "Received" rows. Trace sheet, Wordmark only: no nav, no account menu, no chat strip, no sign-up. | `web/src/GuestPage.tsx`, `web/src/main.tsx`, `trace-home.css` |
| 10. Bench and docs | `guest_open` 100 / 250 ms in `bench/budgets.json` and the gate test; `TestGuestOpenBudget`; the intake bench's `guestLinks` step; PRODUCT.md, AGENTS.md, the plan's status line. | `bench/budgets.json`, `internal/latency/gate_test.go`, `internal/httpapi/guest_budget_test.go`, `cmd/intake-bench/guest.go` |
| From chat | `read(shares)` (Go prints who holds a link to what, by whom, opens, last opened, end, state; never a token); `read(procurement)` adds "link opened 14 Nov" or "link not opened" on an invited firm; the named filter `link_not_opened` on Procurement; Share opened filled in through `view` (`dialog: share`), every id checked by Go first, anything but documents, an issued report or an issued RFP or RFT refused; `shares.revoke` prepared for the click. Eight draft key cases (`ga1` to `ga8`); five scored offline. The prompt gains one line. | `internal/httpapi/chat_shares.go`, `chat_view.go`, `chat_more.go`, `chat_read.go`, `knowledge/chat/system.md`, `data/eval/chat/`, `cmd/chat-eval` |

## 2. Speed

`TestGuestOpenBudget`: 20 rounds on a package link whose issued RFT
transmittal holds 50 documents; each round makes a link, exchanges it in a
fresh browser, reads the page, reads the board, lists and revokes. Local
test database, two runs.

| Budget row | Budget p50 / p90 | Run 1 p50 / p90 | Run 2 p50 / p90 |
|---|---|---|---|
| `guest_open` (exchange and page) | 100 / 250 ms | 1.8 / 3.4 ms | 2.7 / 4.5 ms |
| `team` (share create, list, revoke) | 100 / 250 ms | 1.3 / 1.8 ms | 1.6 / 2.4 ms |
| `procurement_board` (with link state) | 150 / 300 ms | 2.2 / 4.1 ms | 3.0 / 4.1 ms |

The intake bench (`cmd/intake-bench`, `guestLinks`) samples the same paths
through the real server; `TestBenchEndToEnd` passed with the step in
place, but the bench itself was not run over the private corpus. A guest
quote takes the unchanged `whole_intake` and `proposal_read` paths; they
were not re-measured. Downloads stream and are not timed (their grant
check is `guest_open`'s).

## 3. What was not measured

- No live model run; the `shares` group's model-only cases (`ga1`, `ga3`,
  `ga8`) are unscored.
- No owner review of the key or of the page's words.
- No browser check by the owner. The Playwright spec ran headless.
- No security review (the index asks for one before plan 2 ships).
- The full `tools/check.ps1` gate was not run.

## 4. Known limits

- A link is a bearer secret. Scope, expiry, revoke and the open count
  contain it; they do not prove who holds it (plan, risk 1).
- The rate limits are in memory, per process, as the account limits are:
  a restart clears them. Exchanges and failed link checks share one
  30-per-address window; a quote and a page read have their own per-link
  windows.
- A guest's quote is placed after its bytes land; a link revoked in that
  window places nothing and its new document is deleted before filing. A
  background worker that polls pending filings could, in principle, pick
  the document up in the milliseconds between upload and the recheck; the
  dev server runs with the backlog worker off and nothing was seen.
- "Shared with" is a label the PM types; SiteWise sends no mail (plan 1
  owns sending). "Remind the firms" answers with the fixed words.
- The 402 message tells the guest to let "the person who sent this link"
  know, not the sharer's address.

## 5. Threats

| Who | Can | Cannot | Held by |
|---|---|---|---|
| Holder of a leaked link | See and download the one shared thing until it ends or is withdrawn; a firm's link can drop a quote on that firm's own cell only | Reach any other document, an issue's draft, another firm's quote or price, another project or org, any member route, chat or Jev | `TestGuestReachesOnlyItsGrant` (a), (b), (c); `TestOpenGuestQuote` (another firm's bytes 409, B untouched) |
| Someone guessing tokens | 30 tries per address per 15 minutes across the exchange and the cookie; 256-bit tokens | Learn whether a token was ever valid: wrong, expired, revoked and paused give one answer | `TestGuestReachesOnlyItsGrant` (d); `authLimits.blocked` |
| Holder of a revoked or expired link | Nothing: the next request is 401 and the cookie is cleared; an upload in flight places nothing | Re-exchange the token; keep a cached page (no-store) | `TestGuestReachesOnlyItsGrant` (d), `TestGuestRevokedMidUpload`, `TestGuestGrantLife` (replaced link) |
| Signed-in user of another org | Nothing on this project: share list, create and revoke give 404 | Make a link to another org's records | `TestGuestGrantScope`, `TestNextWaveRouteAccessSweep` |
| A PM's own browser with a guest cookie | Two cookie names; member routes read only `sitewise_session`, guest routes only `sitewise_guest` (Path=/api/guest) | Gain either side's access from the other | (a) of the sweep sends the guest cookie to every member route |
| A cross-site page | Nothing: guest writes check Origin, the cookie is SameSite=Strict, pages cannot be framed | Post a quote or exchange a token from another origin | `TestGuestReachesOnlyItsGrant` (e) and its header checks |
