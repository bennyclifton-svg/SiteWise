# Variations, progress claims and payment schedules: what was measured

Date: 19 November 2026. Branch `wave/claims` (from local `main` at
`6f563f2`; not merged, not pushed, not deployed). Plan:
`docs/plans/2026-11-17-variations-progress-claims.md`. Everything here was
run on the development machine against a private PostgreSQL 17 schema
(`wave_claims`, and `wave_claims_pw` for the browser test), with stand-ins
for Jev and the model. The live server was not contacted, and no live Jev
or model call was made. No Security of Payment row is verified, the claim
answer key is empty, and the chat key's new `claims` group is a draft the
owner has not reviewed.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. Act rows and the checker | `knowledge/contracts/security_of_payment.yaml`: NSW and QLD rows, `status: draft`, `verified: false`, counts quoted. NSW's figures are from the NSW Government guidance table, named as such, not the Act; the owner's different figures of 19 November are recorded as an open question. The Go loader and `tools/check_knowledge.py` refuse `verified: true` without the Act's https URL and a checked date, an unquoted count, an unknown state and a duplicate state. The file is not hashed into the catalogue version (no profile, report or Program reads it). `SCHEMA.md` gains "Contracts". | `internal/knowledge/contracts.go`, `tools/check_knowledge.py`, `tools/test_check_knowledge.py` |
| 2. Migration and report guards | Migration 073 (below). Three captured tables with a `claims` finisher (cost revision, `costs` and `invoices` events). A package's report lookup takes only `rfp`/`rft`; `ListReports` leaves payment schedules out; the PDF head says "Payment schedule". | `internal/db/migrations/073_variations_claims.sql`, `internal/store/change_undo.go`, `reports.go`, `report_list.go`, `internal/reports/render/pdf.go` |
| 3. Variation register | Add and edit (number V01 onwards or the PM's; the contract's only line or the PM's pick; approve needs an amount, zero or negative allowed; leaving approved clears it). Undo of an award with a variation or a claim is refused. | `internal/store/variations.go`, `internal/httpapi/variations.go`, `awards.go`, `group_award.go` |
| 4. Variations in the figures | The draft plan carries `approved_variations` and `forecast_variations` on each line with variations; `costs.Forecast` adds them to a contract line (an open variation with no figure leaves the line unknown); the grid's two columns show them. The ledger's Remaining is contract sum to date (contract plus approved) less claimed. A frozen baseline reads the same before and after. The project report reads the same plan. | `internal/costs/totals.go`, `internal/store/variations.go` (`attachVariations`), `internal/invoices/ledger.go`, `groups.go`, `web/src/CostGrid.tsx` |
| 5. Reading a claim | Go lays the claim out as a grid (PDF positions, spreadsheet and Word cells, Markdown bars), finds the header naming to date, previous and this claim, and reads every row under it with two or more amounts until a total row. Rules map a row by its exact line label, a printed variation number, or the word retention; the claim's number by a "Claim no" cell. Jev is asked, in the claim's one `claim-1` fan-out, only which line an unplaced row charges for (and any invoice field rules left open). No table: the contract's lines with amounts unknown. | `internal/invoices/claim.go`, `claim_read.go`, `pipeline.go` |
| 6. Intake and the ledger | `POST /invoices?kind=progress_claim&package=` (only a live works contract); the words-plus-ABN rule; rows written once in the read's transaction. A claim counts once its schedule is issued, at the scheduled amounts, or at the claimed amounts once its due date passes unserved. A tax invoice whose total is the payable to the cent links by rule; others are flagged and the PM links one with `action: settles`. | `internal/store/claims.go`, `invoices.go`, `internal/httpapi/invoices.go` |
| 7. Assessment | `PUT /invoices/{inv}/claim-lines`: rows (assessed to date, reason, line, variation or retention) or Accept claimed. Scheduled is assessed to date less SiteWise's certified before; gross, retention, net, GST (`GSTOf`, zero when the claim printed zero) and payable are Go's. Issue is refused while a row is unassessed or cut without a reason. | `internal/invoices/claim_figures.go`, `internal/store/claims.go` |
| 8. Due dates and the banner | `invoices.ScheduleDue` and `PaymentDue` (section 3). The one banner above the Cost Plan's tabs, a count on the tab, a mark on the Cost Plan's nav entry from any screen, and `reports.Attention`'s `claims.schedule_overdue`, `claims.schedule_due` and `variations.undecided` all read the same alert. | `internal/invoices/sopa.go`, `internal/reports/attention.go`, `internal/store/profile_run.go`, `web/src/Costs.tsx`, `Project.tsx` |
| 9. Issue | One transaction: the claim's report (made on first issue), an issued version with its snapshot, the PDF and its file row; a later issue is version 2 linked to the first. The rows are frozen by a trigger. The PDF names the Act, prints a section only from a verified row, and says when it is issued late. | `internal/store/claims_issue.go`, `internal/reports/payment_schedule.go` |
| 10. The tab | Contract picker (two or more contracts), header figures and the contract's periods, the variation register (grey row to add, inline edits, status menu, Undo), the claims table (drop a file on it), and the claim panel (rows, assessed to date, reasons, totals, the two dates with how each was worked out and "to check", Accept claimed, Issue, Download PDF, Served on). | `web/src/Variations.tsx`, `invoiceApi.ts`, `Costs.tsx`, `invoices.css` |
| 11. Operations, chat, key | Section 4. | `internal/httpapi/ops_claims.go`, `ops_claims_assess.go`, `chat_claims.go`, `chat_view.go`, `chat_screen.go`, `data/eval/chat/` |
| 12. Benches and the key | Five budget rows; `TestVariationsBudget`, `TestClaimIntakeBudget`; `invoice-eval -claims` checks the private claim key's shape; `data/eval/claim/README.md` gives its format. | `bench/budgets.json`, `internal/latency/gate_test.go`, `internal/httpapi/claims_test.go`, `cmd/invoice-eval/claims.go` |

**Migration 073** widens `invoices_kind_check` (adds `progress_claim`),
`reports_kind_check` (keeps `pmp`, `rfp`, `rft`, `monthly`; adds
`payment_schedule`) and `project_changes_area_check` (keeps the five
areas; adds `variations` and `claims`); adds `invoices.settles_claim_id`,
the contract's `schedule_days`, `schedule_basis`, `payment_days` and
`payment_basis` on `awards`, `reports.invoice_id` with one schedule per
claim, the tables `variations`, `variation_documents` and
`progress_claim_lines`, their capture triggers, and the trigger that
freezes an issued claim's rows. The new foreign keys to cost lines, awards,
packages, variations and invoices are deferred, so a project's or org's
deletion removes them in whatever order the cascade takes. The FK sweep
inventory gained twelve keys and fixtures for the three tables.

## 2. Speed

Development machine, private schema, stand-ins; twenty samples per path,
each well past five calls on a connection.

| Budget row | Budget p50 / p90 | Measured p50 / p90 | Held by |
|---|---|---|---|
| `variations_read`: GET /invoices with 200 variations, 24 claims of 40 rows, 324 invoices | 100 / 250 ms | 32 / 34 ms | `TestVariationsBudget` |
| `variation_write`: PATCH a variation, ledger returned | 100 / 250 ms | 26 / 29 ms | `TestVariationsBudget` |
| `claim_assess`: Accept claimed on a 40-row claim, ledger returned | 100 / 250 ms | 48 / 54 ms | `TestVariationsBudget` |
| `payment_schedule_issue`: a 40-row claim's schedule, PDF and all | 100 / 250 ms | 29 / 34 ms | `TestVariationsBudget` |
| `claim_intake`: a Markdown claim dropped to its ledger row, test Jev | 1,000 / 2,000 ms | 81 / 93 ms | `TestClaimIntakeBudget` |
| `invoices_read` (1,000 invoices) | 150 / 300 ms | not re-measured | the read now also reads variations and claim rows |

The intake bench was not run over the private corpus, and no 40-row claim
was read with a live Jev call, so `claim_intake` with Jev's real latency is
unmeasured (plan risk 4).

## 3. Security of Payment dates

Nobody on the project has read the NSW Act. The knowledge file records:

| State | Payment schedule | Payment (head contract) | Source | Verified |
|---|---|---|---|---|
| NSW | 10 business days after service, s 14(4) | 15 business days, s 11(1A)(a) | NSW Government guidance table ("Making a payment claim") and commentary; not the Act. The subcontract rows (20 days, s 11(1B)(a); 10 days residential, s 11(1C)(b)) are recorded and not used. | false |
| QLD | 15 business days, s 76(1) | 10 business days, s 73(1) | The plan's reading of the Act on 17 November 2026; not re-read | false |
| Others | no row | no row | to read | — |

The owner said on 19 November that payment is 15 business days head
contractor to subcontractor and 10 business days principal to head
contractor. That does not match the guidance table; the file notes it as an
open question (the owner's own contract terms, or a slip).

How a date is worked out (`invoices.ScheduleDue`, `PaymentDue`): counting
starts the business day after service (a weekend service starts Monday), on
`program.BusinessDayAfter` with weekends and the state's reviewed public
holidays only (`knowledge/program/calendars.yaml`, empty today), never a
project's rostered days off. The date is the earlier of the contract's own
period (typed per contract, business or calendar days) and the Act row's;
a longer contract period is noted as one the contract cannot extend. With
no contract period the words say the Act's figure was used, with its
section. An Act row's excluded spans (27 to 31 December in NSW) are applied
only when the row is verified, so every fallback errs early.

A date is marked "to check", with its notes, whenever the Act row is
unverified, the served date is the drop day rather than the PM's, or the
state's holidays for a year it spans are not loaded ("serve by this date to
be safe"). No state on the Profile, or neither period known, is "Due date
unknown" with what to enter. Today every date in the product is "to check".

## 4. Chat and the tab

| | From chat |
|---|---|
| Reads | `read(variations)`: per contract the sum, approved variations, sum to date, certified to date, retention held and periods; each variation; each claim with its two dates and, for each, the words and notes it was worked out from; each row's claimed to date, certified before, assessed and this schedule. Every amount is tagged. |
| At once, with Undo | `variations.add`, `variations.edit`, `claims.assess` (one row per operation; amounts and days are figures the PM must have given). |
| Waits for a click | `variations.decide` (approve or reject) and `claims.accept_claimed` wait for the click from chat and apply at once, with Undo, from the screen's own Apply; `claims.contract_periods` (awards carry no capture trigger). |
| Opened filled in | The `claim` dialog: the claim's panel with its assessment; the PM presses Issue payment schedule there. `POST /invoices/{inv}/schedule` is `noOperation`. Send is plan 1's. |
| Screen | Tables `variations`, `claims` and `claim_rows`; named filters `undecided`, `schedule_due` and `unassessed`. |

**The key.** `claims` group, eight draft cases (`cl1` to `cl8`): two reads,
recording a variation, typing an assessment, the unassessed filter,
approving (prepared), opening claim 4 to issue (nothing issued), and a
claimed amount the PM never gave (refused by Go's figure check).
`TestScreenCasesScoredOffline` scores all eight with the stand-in model;
all pass, as do the existing groups. The house fixture's contract, V01 and
claim 4 are seeded by `cmd/chat-eval/claims.go`.

## 5. What was not measured

- No live Jev call: the claim-1 questions' accuracy is unknown; the key is
  empty and every Jev pick on a claim is stored blank.
- No live model run of the `claims` group.
- No date checked against the Acts. No public holiday is loaded.
- No owner review of the Act rows, the key, the screen or the wording.
- The screen was checked by Playwright only, not by eye in a browser.

## 6. Where the build differs from the plan

- **Unverified rows are used, marked "to check".** The plan used an Act row
  only when verified; the task said to show a date from an unverified
  figure with "to check" and its section. QLD stays `verified: false` (the
  plan marked it true) because nothing may be marked verified. VIC, SA, TAS
  and ACT figures in the plan's table are not recorded: their source was not
  named.
- **A claim's shares are worked out at read** from its rows, rather than
  written to `invoice_allocations` on issue, so the claim's certified
  amounts have one source. The tax-invoice link rule is also read-time; the
  PM's own link is stored.
- **Decide and Accept claimed are Hold operations, not Click.** In this
  codebase a Click operation has no Undo; the plan's done means 1 needs
  Undo on approval. From chat both wait for the click; from the screen they
  apply with Undo.
- **New change areas** `variations` and `claims`, not `cost_plan`.
- **No header-cell questions.** When the header words do not place the
  columns, the claim gets its contract's lines with amounts unknown.
- **Not built:** the QLD "supporting statement" check (s 75(7)); filing a
  file dropped on a variation row (documents link by id through
  `variations.edit`); the `-claims` live run, option shuffle and fit in
  `invoice-eval` (it checks the key's shape only).
- **Issued in time, not yet recorded as served** shows "to check", not past
  due, so the claim keeps counting at its scheduled amounts.
- **The PDF is not page-gated** (`render.PDFUnlimited`): an 80-row claim
  never loses rows to the page limit.
- **Two reading fixes:** a Markdown row's table bars count as text in the
  readable check, and a "Claim no" cell gives the claim's number by rule.
- **The plan's chat case "three days"** is written "3 days" in the key so
  the figure check compares digits.
- **The nav mark** comes from the profile run's next counts (`ProfileNext`),
  which reads the same claims as `Attention`.

## 7. What waits

- **Plan 1 (correspondence):** Send with the schedule attached, and
  `schedule_served_on` set by the mail service's acceptance (today the PM
  types it). The message chip's "Read as progress claim" calls
  `store.CreateInvoice` with `InvoiceUpload.ClaimPackage` and the
  attachment's hash; "Add variation" calls `store.AddVariation` with the
  filed message in `VariationInput.Link`. Nothing sends email.
- **The owner:** reading NSW ss 4, 11, 13 and 14 (decision 1) and the
  figures of 19 November; confirming QLD; naming the public holiday source
  (decision 2); 20 or more real claims for `data/eval/claim/private/`
  (decision 7); a review of the chat key and the screen.
