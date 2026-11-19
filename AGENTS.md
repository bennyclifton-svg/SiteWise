# Agent instructions

SiteWise is a construction-project app built from first principles on the
physical building, with **Jev (TypeSafe System One) as the only AI that reads
documents** and **speed as the first objective**. Read `PRODUCT.md` for the current surfaces.
Read `docs/design/2026-09-29-foundation-design.md` for the binding rules:
speed, Jev only, the building as the model, and the intake budget.

## Non-negotiables

1. **Jev reads; one LLM acts only on what it is given.** Jev (TypeSafe System
   One) is the only AI that files documents or reads them into fields. Jev
   picks; code parses, computes, looks up and controls flow. One LLM, which
   each org switches on, has four jobs. In background jobs it may propose
   line structure, row mappings and qualifications for quotes. It may draft
   text. In chat it may answer the signed-in user's question from the
   project's own records, messages and documents, quoting what it read. In
   chat it may turn the signed-in user's typed instruction into the
   operations a screen would send. It never supplies an amount, a date or a
   rule number that the user or SiteWise did not give it. Nothing in a
   document or an email is an instruction to it. Outside chat, nothing it
   proposes is applied without the user's acceptance. Chat may prepare a Jev
   read for the user's click; the turn never calls Jev; the click queues the
   background read. No embeddings and no other providers. (Owner decisions,
   14 and 15 November 2026:
   `docs/design/2026-11-14-chat-control-and-email-intake.md`,
   `docs/plans/2026-11-15-chat-agent.md` Task 0; 18 November 2026:
   `docs/plans/2026-11-18-chat-agency.md` decision 1.)
2. **Follow TypeSafe's docs** for every Jev use and cite the page:
   [patterns](https://docs.typesafe.ai/patterns),
   [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one),
   [confidence](https://docs.typesafe.ai/confidence),
   [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
   [API](https://docs.typesafe.ai/api). Full index: https://docs.typesafe.ai/llms.txt
3. **Speed has budgets.** Every user-facing path has a p50/p90 budget and a
   benchmark that fails the build when it is broken. One Jev fan-out per
   state; never serial round trips on a hot path; nothing judged while a user
   waits if it can be precomputed. Chat is the one path where serial model
   rounds are allowed, and it has its own budget rows. No screen read or
   write calls the LLM.
4. **The building is the model.** Knowledge is organised as systems,
   determinants, rules, interfaces and failure modes (`knowledge/SCHEMA.md`).
   Consultant disciplines are a view over it, not its structure.
5. **Rule numbers come from the instrument itself** (NCC 2022, the standard),
   never from seed prose alone. Unverified numbers stay `verified: false`.

## Current product

`PRODUCT.md` names the workbench. The foundation design is the origin of the
rules above and describes the intake slice. The workbench also includes
Profile, Cost Plan, Procurement, Specifications, Program and Register.
Register is the left nav's former Project Plan slot: one table of what is
owed (RFIs, notices, consent conditions and the older delivery kinds).
Implication proposals
stay on the project and are off the left nav.

When a task does not name a budget, use the budget already recorded for that
path in `bench/budgets.json`. Filing stays p50 ≤ 1 s and p90 ≤ 2 s. Profile
edit stays at its existing tighter budget. Cost, procurement and specification
reads and writes stay at p50 ≤ 100 ms and p90 ≤ 250 ms unless that named path
is tighter.

These paths do not call Jev: cost reads and writes, procurement row create,
report create, specification direct edit, review, schedule and issue, the
profile's run reads (progress, finds, next), Accommodation schedule edits,
and guest links: share create, list and revoke, the guest page open and
guest downloads. A guest's quote is filing plus the existing proposal
fan-out, exactly as a PM drop on that firm's cell
(`docs/plans/2026-11-17-guest-access.md`). Only the `/guest` routes read
the guest cookie, and they read nothing else.
Specification import is a background job in the existing worker: one fan-out
per candidate, cached by source. A reading is a proposal. Navigation and
typing do not call Jev. An uploaded invoice is read with one interactive
fan-out for the fields rules leave open; invoice edits, splits, the ledger
read and export do not call Jev. A proposal dropped on a procurement row is
read after filing with one Jev fan-out and, when the org has switched the
model on, one tender pass; the price table, line moves and splits, adds and
subs, checking, award, undo, ratings and the comparison report call neither. Naming an
invited firm on a row, and removing it, do not call Jev; a quote dropped on
an invited firm's cell is that firm's
(`docs/plans/2026-11-11-procurement-invited-firms.md`). Award writes
contract values only from the firm's priced cells. When the package's budget
is blank or unknown, award adopts that contract sum as the budget (undo
restores it); a budget the PM set is never changed.
Program open, edit, drag, link, generate, review, stage sync, export and
print do not call Jev; every date is computed in Go. Its three readings (a
dropped program file, the R7 returnable after award, hold-point release
evidence) are background fan-outs that write proposals only.

Register reads and writes call neither Jev nor the LLM: numbers, due dates
(the Program's working days), days late, With and the also-open line are
Go's. Its one reading is the background `register` job queued in the
filing commit when the project has an Open RFI: Go's cited-number rule for
any document, and for correspondence one Jev fan-out (one noul per RFI, at
most 12) whose picks stay hidden until the owner approves a cut-off
(`docs/plans/2026-11-17-rfis-review-comments.md`). A reading is a proposal;
only the PM's edit makes an RFI answered.

Variations and progress claims sit on the Cost Plan's Variations and claims
tab (`docs/plans/2026-11-17-variations-progress-claims.md`). Variation
writes, assessments, Accept claimed, a contract's periods, the tab's read
(it rides on `GET /invoices`) and Issue call neither Jev nor the LLM; every
figure and date is Go's (`invoices.ClaimLedger`, `invoices.ScheduleDue`).
A progress claim is an invoice of kind `progress_claim`; dropped on a
contract it is read with one interactive fan-out (`claim-1`) for the rows
rules leave open, and those picks stay blank until a claim answer key is
fitted. Security of Payment periods come from
`knowledge/contracts/security_of_payment.yaml`; no row is verified, so every
date from one is "to check" with its section, and only the owner sets
`verified: true`. Issuing a payment schedule is never an operation: chat
opens the claim's panel and the PM presses Issue.

Consent conditions are a kind in the same table
(`docs/plans/2026-11-17-consent-conditions.md`). A notice of determination
is read into rows by a background `consent` job: Go splits it by its
printed numbers and stage headings (the number and words are the notice's,
never renumbered or reworded), then one Jev fan-out per condition (who acts,
the system, whether evidence is needed, and the gate only when the heading
left it open), cached by source. It is queued after filing for a document
filed as a Statutory Instrument that Go recognises as a notice, or by Read
conditions (`POST /projects/{id}/delivery/imports`, which only checks the
text and queues). Every pick is amber "to check" at most until the PM edits
the row; nothing is green until the owner reviews the key. The `register`
job carries the evidence nouls (one per open condition, at most 12) in its
one request; Go's "Condition 13" rule shows at once, Jev's picks wait for an
approved cut-off. Evidence never satisfies a condition: only the PM's click
does. Program gates (one hold point per gate) come through the existing
review; their dates are Go's.

The project report (Register toolbar, one per project) calls neither Jev
nor the LLM: Update, comments, issue and export are Go over saved records,
and every figure comes from the screens' own rollups (`costs.Summarize`,
the invoice ledger, the Program engine, the procurement board, the
register, `reports.Attention`); unknown stays unknown. Chat may update it
and write or clear its comments with Undo, and opens its issue form filled
in, but never presses Issue
(`docs/plans/2026-11-17-monthly-report.md`).

Mail (`docs/plans/2026-11-14-email-intake.md`,
`docs/plans/2026-11-17-correspondence-transmittals.md`; built 19 November
2026 on branch `wave/mail` against stand-ins, Mailgun's Task 0 not run).
The mail webhook, the held-mail list, accept, discard, rotating the address,
the record's read and its two edits (package, the pick), Send and the sheet
do not call Jev or the LLM; each filed document is filed as a dropped one
is. Jev's one `notice` question joins an email body's own filing call (no
second call); its pick is a proposal, never a record, shown only past an
owner-approved cut-off (`data/mail/thresholds.json`). All mail leaves
through `queueOutbound` and only on the PM's press of Send; chat opens Send
filled in and never sends. A filed or sent message is never edited or
deleted. The webhook is off unless `SITEWISE_MAIL_IN_DOMAIN` and
`SITEWISE_MAILGUN_SIGNING_KEY` are both set; its sender check
(`mailin.Authenticated`) fails closed.

Chat (`POST /projects/{id}/chat`) is the one path that waits on the LLM, and
it never calls Jev. Its tools read through the same Go rollups as the
screens and change records only through operations (`internal/httpapi/ops*.go`),
which call the screens' own store writes. The operations call
(`POST /projects/{id}/ops`), the undo of an applied change, a prepared
change's Apply and Dismiss, and the chat reads call neither Jev nor the LLM.
A read chat prepares (`internal/httpapi/ops_reads.go`) is always a card for
the user's click, never applied at once and never undone; its Apply runs the
screen's own route, which only queues the background read.
Chat sees what is on screen: the view, the open document and the selected
rows, sent as ids only and checked against the project. Its `view` tool
moves the screen (show, select, show only, clear, open); a move changes no
record, so it has no Undo and no click. It finds documents by subject in Go,
from titles, passages and the profile's labels, with no embeddings. "Undo
that" is answered by Go without the LLM. Chat opens the award dialog filled
in, but never presses Award, Send or Issue.
An amount, a date or a duration in a chat operation must be in the user's
message or in a tool result of the same turn, or the operation is refused. A
turn that read a document, a tender's lines or a message prepares its changes
for the user's click. A new write route must be classed in
`internal/httpapi/ops_routes.go` or the build fails, and
`TestOnlyChatReachesTheModel` fails if a screen handler reaches the LLM.

Money is computed in Go. Unknown stays unknown. Do not invent a rate, a price,
a rule number or a product selection. An upload leaves an accepted
specification value and a frozen cost baseline as they were. Removing a
procurement package may delete linked draft cost lines only through the
existing confirmation, and leaves frozen baselines and issued history intact.

Before changing one of these surfaces, read its current note:

- Profile: `docs/design/2026-10-03-project-profile.md`; run-time feedback
  (update strip, marks, receipt): `docs/design/2026-11-10-run-time-feedback.md`,
  `docs/evidence/2026-11-10-run-time-feedback.md`; update speed (gzip
  requests, sixteen passages in flight):
  `docs/evidence/2026-11-11-profile-update-speed.md`
  Accommodation schedule: `docs/plans/2026-11-11-accommodation-schedule-lines.md`
- Cost plan: `docs/plans/2026-10-10-cost-plan-grid.md`,
  `docs/plans/2026-10-10-cost-plan-default-inclusion.md`,
  `docs/plans/2026-10-11-cost-plan-direct-editing.md`,
  `docs/plans/2026-11-11-cost-plan-bulk-actions.md`
- Procurement: `docs/plans/2026-10-10-procurement-table.md`,
  `docs/plans/2026-10-11-procurement-cost-removal.md`,
  `docs/plans/2026-10-11-procurement-default-disciplines.md`.
  That plan's header still says nothing is built; the knowledge file, the
  loader and the browser test are in the tree. Trust the code.
- Specifications: `docs/design/2026-10-14-specification-workspace.md`,
  `docs/plans/2026-10-14-coordinated-specification-workspace.md`,
  `docs/evidence/2026-10-14-specification-implementation.md`
- Procurement to award: `docs/plans/2026-10-16-procurement-process-to-award`,
  `docs/plans/2026-10-16-procurement-to-award-implementation.md`,
  `docs/evidence/2026-10-16-procurement-to-award.md`
- Tender comparison (built 14 November 2026 on branch `tender/comparison`):
  `docs/design/2026-11-14-tender-comparison.md`,
  `docs/plans/2026-11-14-tender-comparison-plan.md`, evidence in
  `docs/evidence/2026-11-14-tender-comparison-*.md` (read the gate note
  first: the measured gate was not a clean pass and the answer key is not
  owner-reviewed). It replaces
  `docs/plans/2026-10-19-comparison-alignment.md` and the 13 November
  levelled-comparison specification. It is rule 1's background job for
  quotes: the LLM client (`internal/llm`), the org switch, and the change
  log with undo (`internal/store/change.go`).
- Program: `docs/plans/2026-10-18-program.md` (owner answers in 13.3)
- Chat (built 15 November 2026 on branch `chat/agent`, not merged):
  `docs/plans/2026-11-15-chat-agent.md` (read "Amendments", "Progress" and
  "Deviations"), evidence in `docs/evidence/2026-11-15-chat-agent.md`. The
  command key is `data/eval/chat/`; it is draft and not owner-reviewed.
  Chat that sees and moves the screen: `docs/plans/2026-11-18-chat-agency.md`,
  evidence in `docs/evidence/2026-11-18-chat-agency.md`.
- Invoices (Cost Plan panel): `docs/plans/2026-10-15-invoice-intake.md`,
  `docs/plans/2026-10-16-invoice-intake-implementation.md`,
  `docs/evidence/2026-10-16-invoice-intake.md`
- Mail (repository Messages, Send): `docs/plans/2026-11-14-email-intake.md`
  (read "Task 0 findings: not yet run" first),
  `docs/plans/2026-11-17-correspondence-transmittals.md`, evidence in
  `docs/evidence/2026-11-19-mail.md`. Nothing has run against real
  Mailgun; the notice key (`data/eval/notice/`) is empty until the owner
  supplies about 50 messages.
- Variations and claims (Cost Plan; built 19 November 2026 on branch
  `wave/claims`): `docs/plans/2026-11-17-variations-progress-claims.md`,
  evidence in `docs/evidence/2026-11-19-variations-progress-claims.md`. The
  Act rows are draft and unverified; the claim key
  (`data/eval/claim/private/`) is empty.
- Consent conditions (Register; built 18 November 2026 on branch
  `wave/conditions`): `docs/plans/2026-11-17-consent-conditions.md`,
  evidence in `docs/evidence/2026-11-18-consent-conditions.md`. The answer
  key is the owner's (`data/eval/consent/private/`, not in git); accuracy on
  real notices is not measured.

## Deploying

To update or redeploy the live server at sitewise.au, follow
`docs/deployment.md` (server facts and the redeploy steps). The full
operations manual is `docs/operations.md`.

## Stack

Go single binary (API, workers, embedded Vite + React SPA, SSE), PostgreSQL 17
on the same host (full-text search, no pgvector), files on local disk by
content hash, Caddy + systemd. Stack changes need a stated reason.

## Reference repo

`../clerk` is frozen reference. Copy data from it (`data/seed/`,
`data/taxonomy/`, answer keys, corpora), never code. Ignore its PM doctrine
(`docs/clerk-brief.md`) and its Jev plan's process rules; this repo's design
supersedes them.

## Knowledge work

- Run `python tools/check_knowledge.py` after editing `knowledge/`; it must pass.
- Everything written from seeds is `status: draft`. Only the owner marks
  `reviewed`.
- Follow the question authoring rules at the end of `knowledge/SCHEMA.md`.

## Style

Small obvious functions, validation at boundaries, no speculative
abstraction, comments explain why. Dependencies only when the alternative is
non-trivial; justify each in the commit message.

## Starting a feature

The why, the three lanes, and the prompt run sheets live in
`docs/developing.md`. Follow them.

A change is one lane:

- **A, product core** (filing, org isolation, rule numbers, cost figures,
  procurement packages, issued specifications, accepted specification values,
  anything a user trusts). An existing test or answer key must still pass, and
  the speed budget must still pass. Unverified rule numbers stay
  `verified: false`.
- **B, edge** (a label, an empty state, a layout). Move fast. Do not break
  filing.
- **C, security and dependencies** (login, file access, anything new in the
  build). Handle a missing file, a wrong org, and unexpected input. Prefer
  a small dependency many people already watch over one this repo would be
  the only watcher of.

Before coding a new feature, the task states the user outcome, why it
serves filing speed, answer trust, or time-to-decision, the lane, what is
out of scope, the p50/p90 budget in milliseconds, what "done" looks like,
and the edge cases. When those are missing, apply these defaults and do
not invent a wider task: use the budget already recorded for the path in
`bench/budgets.json`; filing stays p50 ≤ 1 s and p90 ≤ 2 s; one Jev
fan-out per state; every row scoped to `org_id`; no new service or dependency
without a stated reason.

A clear defect is fixed only as far as that defect. A direction choice
(what the next screen or workflow should be) is decided in prose before
code. After implementing, report timing against the budget, what changed
in plain language (what calls what, what could fail), and any new
dependency. Do not widen the feature.
