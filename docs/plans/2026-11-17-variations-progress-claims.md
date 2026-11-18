# Variations, progress claims and payment schedules: implementation plan

Date 17 November 2026; revised 18 November. Lane A (cost figures, and dates a PM relies on).
Status: plan only, nothing built.

Money after award. It fills the Cost Plan's Variations tab
(`web/src/Invoices.tsx:293`, a placeholder) and two placeholder grid columns
(`web/src/CostGrid.tsx:48`). It sends through plan 1
(`docs/plans/2026-11-17-correspondence-transmittals.md`) and feeds plan 5
(`docs/plans/2026-11-17-monthly-report.md`).

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
The scope stands. Chat opens the claim to Issue but never issues ("From
chat"). A message can become a claim or a variation in one click. Due dates
feed the one `Attention` function. This plan now creates
`knowledge/contracts/` (plan 1's time bars are deferred). The earlier text
is at git ref `refs/backup/next-wave-plans-2026-11-18`.

## In one page (product requirements)

**Outcome.** The Cost Plan tab "Variations and claims" shows each live
contract: contract sum, approved variations, contract sum to date, certified
to date, retention held. Below are two tables. **Variations**: number,
description, status (proposed, submitted, approved, rejected, withdrawn),
claimed, PM's estimate, approved, time claimed (days), linked documents.
**Progress claims**: number, served, reference date, claimed, scheduled,
schedule due, payment due, status. The PM drops the builder's claim on its
contract; it is read in about a second (number, dates, retention, GST, and
each row of the claim table against the contract's lines and variations).
The PM opens it, sets the assessed amount per row or clicks Accept claimed,
types a reason where it is less, and clicks Issue payment schedule. SiteWise
freezes a one-page PDF and opens plan 1's Send. A schedule not served by its
due date is a red banner across the Cost Plan and a mark on the Cost Plan's
left-nav entry, seen from any screen. On the grid, Approved
Variations and Forecast Variations fill in, Forecast Final Cost includes
them, Claimed to Date takes certified claims, and Remaining is contract sum
to date less claimed.

**Why.** Answer trust and time-to-decision. A payment schedule served a day
late can make the client liable for the whole claimed amount. Today the PM
keeps that date, the register and the certified totals in a spreadsheet.

**Where it lives.** The existing Variations tab (renamed, decision 6), a
side panel for one claim (the Invoices tab's `PageSidePanel` pattern), a
banner above the Cost Plan's tabs, and two grid columns that already exist.
New screens: none. New left-nav entries: none.

**A progress claim is an invoice with kind `progress_claim`.** It arrives
the same way (upload, content hash, one row per file, "Already filed"), is
read by the same reader and re-check loop, keeps its fields in
`invoice_decisions` with the same chips, matches its supplier by the same
ABN rule, and sits in the same ledger, filters and export. A parallel
pipeline would copy most of `internal/invoices/` and
`internal/store/invoices.go`. A claim adds its rows (claimed to date,
previous, this claim, the PM's assessment) in one child table, and its own
question version `claim-1`, so the calibrated `invoice-1` questions
(`data/invoice/thresholds.json`) are untouched.

**How a claim is recognised.** Dropped on a contract on the tab, the drop
sets kind and package (user decisions, as a proposal dropped on a
Procurement row). From an email, plan 1's chip offers "Read as progress
claim" (the same `CreateInvoice`, with the attachment's hash) and "Add
variation" (Go fills the firm, the package and the linked message; amounts
stay unknown). On the Invoices tab, a rule makes it a claim when
"progress claim" or "payment claim" is printed and the ABN rule has settled
the supplier to a package with a live award. Both are known before the
fan-out. Otherwise it reads as an invoice and offers "Read as progress
claim" (one click, a new state, one fan-out).

**What counts, and no double counting.** A claim counts in Claimed to Date
when its schedule is issued, at the scheduled amount per contract line, ex
GST, before retention (decision 3). Until then it is "Assessing" and counts
nowhere. If its due date passes with no schedule served, it counts at the
claimed amounts, in red. The builder's tax invoice for the certified money
is linked to the claim (`settles_claim_id`) and not counted, because the
claim already stands for it. Go links it when the package matches and its
total equals the schedule's payable amount to the cent; otherwise the ledger
flags each unlinked invoice from a package with claims ("Is this the tax
invoice for a progress claim?") and the PM links it in one click. A
retention release is a negative retention row, so it changes what is paid,
never the certified cost.

**What reads what.** Jev asks only what rules leave open, all in one
interactive fan-out per claim, question version `claim-1`:

| Jev question | Options | Asked when |
|---|---|---|
| Claim number; claim date; reference date | numbers or dates Go harvested, none | the labelled-candidate rule leaves it open |
| Total payable; GST; retention this claim | amounts Go harvested, none | the amount rule leaves it open |
| "Which line of the contract does this row charge for?" (the row's label and amounts in the question) | the contract's lines in contract order, its variations, none | the exact-label and variation-number rules leave the row open (80 rows at most) |
| Which header cell is claimed to date; previous; this claim | the table's header cells | the header-word rule cannot place the columns |

Jev picks and Go copies ([pre-parsed values](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[choice](https://docs.typesafe.ai/primitives/choice)), in one request
([fan-out](https://docs.typesafe.ai/patterns/fan-out),
[how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)).
Each question names its role and what it is not
([advanced](https://docs.typesafe.ai/primitives/advanced)); at most 254
options plus none ([API](https://docs.typesafe.ai/api)). Bands come only from
calibrated cut-offs ([confidence](https://docs.typesafe.ai/confidence),
[routing](https://docs.typesafe.ai/patterns/confidence-routing)); with no
claim answer key yet, Jev's picks stay blank until one is fitted (decision
7). [jev-1.13](https://docs.typesafe.ai/model-jaggedness/jev-1.13): "Jev is
not a calculator", so all arithmetic is Go; Jev leans to the first option,
so the eval shuffles options; a larger state lowers accuracy, so the state
is the claim excerpt. Go does the rest: the claim table, row rules, every
amount, check and date, contract sums, ledger shares, the schedule. The
LLM: none; nothing is drafted and the reasons are the PM's.

**Security of Payment dates.** Per claim, Go computes when the payment
schedule is due (the earlier of the contract's period and the Act's, in
business days after service) and when payment is due. Inputs: the project's
`state` determinant (`knowledge/determinants.yaml:187`, as Program's calendar,
`internal/store/program_prepare.go:238`); the served date (the drop day by
rule, amber "confirm"; an emailed claim's received day; the PM's word wins);
the contract's own periods, typed once per contract; one Act row per state in
`knowledge/contracts/security_of_payment.yaml` (new); and the state's
`public_holiday` rows in `knowledge/program/calendars.yaml`, which Program
already reads, owner-maintained and empty today (decision 2). Go uses an Act
row only when `verified: true`. Each fallback errs early, never late:

| What is known | What the PM sees |
|---|---|
| Act verified, holidays for the year loaded | The due date (green) |
| Holidays for that state and year missing | "Serve by <date> to be safe": weekends and the Act's own excluded days only, so the true date is never earlier (amber) |
| Act not verified, contract period typed | The contract's date, "the Act may set an earlier date" (amber) |
| Neither, or no state on the Profile | "Due date unknown" and what to enter (red) |

Act rows as they stand (decision 1):

| State | Business day excludes | Schedule due | Payment due if the contract is silent | Status |
|---|---|---|---|---|
| QLD, Building Industry Fairness (Security of Payment) Act 2017 | Sat, Sun; public, special or bank holiday in the place; 22–24 Dec, 27–31 Dec, 2–10 Jan (sch 2) | earlier of the contract or 15 business days after the claim is given (s 76(1)); reasons if less (s 69(c)); none given: liable for the claimed amount (s 77) | the contract's date, else 10 business days after the claim (s 73(1)); a commercial building contract paying later than 15 business days falls back to 10 (s 73(4)(c) and note 3; QBCC Act s 67W itself not read) | `verified: true`, read 17 Nov 2026, https://www.legislation.qld.gov.au/view/whole/html/inforce/current/act-2017-043 |
| NSW, Building and Construction Industry Security of Payment Act 1999 | draft: Sat, Sun, public holiday, 27–31 Dec (s 4) | draft: earlier of the contract or 10 business days after service (s 14(4)); reasons (s 14(3)) | draft: head contractor, earlier of the contract or 15 business days (s 11(1A)) | `verified: false`: legislation.nsw.gov.au served a browser check page, not the Act, on 17 Nov 2026 |
| VIC, SA, TAS, ACT | to read | draft: VIC 10 (s 15(4)), SA 15 (s 14(4)), TAS 10 (s 19), ACT 10 (s 16) | to read | `verified: false` |
| WA, NT | to read | to read | to read | `verified: false` |

**Speed budgets** (p50 / p90 ms). No screen read or write waits on Jev or
the LLM. Due dates are computed at read time, never stored.

| Path | Budget | Bench row |
|---|---|---|
| Claim dropped to its ledger row read (40-row claim) | 1,000 / 2,000 | `claim_intake` (new; the `invoice_intake` budget) |
| Tab read, riding on `GET /projects/{id}/invoices` (200 variations, 24 claims, 300 invoices) | 100 / 250 | `variations_read` (new; the `costs_read` budget); 1,000 invoices keep `invoices_read` 150 / 300 |
| Add or edit a variation; contract periods | 100 / 250 | `variation_write` (new; `costs_write` budget) |
| Save an assessment or Accept claimed | 100 / 250 | `claim_assess` (new; `costs_write` budget) |
| Issue: sections, snapshot, one-page PDF, allocations | 100 / 250 | `payment_schedule_issue` (new; `report_issue` budget) |
| Undo; chat operation; chat read | as recorded | `change_undo`, `ops_apply`, `chat_tool_read` |

**Done means.**
1. V01 (submitted, $12,400 claimed, 3 days) approved at $11,000: contract
   sum to date, Approved Variations, Forecast Final Cost and Remaining move
   by $11,000; Undo puts it all back.
2. A real claim dropped on its contract is read inside budget; rows matching
   the award's lines map by rule; a "previous" unlike SiteWise's is flagged.
3. Accept claimed, cut one row with a reason, Issue: a frozen PDF; Claimed to
   Date rises by the scheduled amounts; the tax invoice links by rule; the
   ledger total does not double.
4. QLD with 2026 holidays dates correctly across Christmas; NSW shows the
   contract's date with its label; no state shows "Due date unknown".
5. Two business days out: amber banner; past due: red, claimed counts.
6. New routes 404 to another org; five bench rows pass; existing invoice,
   award, cost and ledger tests pass unchanged.

**Out of scope.** Extension-of-time assessment and moving Program dates
(time claimed is recorded only). Adjudication, interest, retention trusts.
Set-offs and liquidated damages as rows (the PM assesses less, with the
reason). Subcontractor-to-builder claims. Consultant invoices as Security of
Payment claims (risk 6; the first follow-up, since before construction they
are most of the claims a client-side PM receives). Jev reading a variation claim letter (a follow-up;
the `claim-1` header questions fit). Sending (plan 1). The builder as a
guest (plan 2 has three kinds of link and none is a contract).

**Owner decisions needed** (recommended default first).
1. **NSW Act.** The owner, or an agent with a browser, reads ss 4, 11, 13,
   14 on legislation.nsw.gov.au; we set `verified: true` with URL and date.
   Until then NSW shows contract dates and "serve by" dates.
2. **Public holidays.** Owner answer Q6 keeps `calendars.yaml` empty until a
   source is named. Default: the NSW and QLD governments' published lists,
   2026–2028, entered `reviewed` and shared with Program. A day in doubt
   (bank holiday, local show day) is left out, so dates err early.
3. **When a claim counts.** On issue, certified, before retention; after an
   unmet due date, the claimed amount, in red.
4. **Forecast Variations.** A PM's estimate per variation; blank falls back
   to the claimed amount, labelled "claimed".
5. **Forecast Final Cost** on an awarded line becomes contract plus approved
   plus forecast variations. Frozen baselines keep what they froze.
6. **Tab label** "Variations and claims".
7. **Answer key.** The owner supplies 20 or more real head-contract claims
   (private, `data/eval/claim/`); Jev's claim picks stay withheld until fitted.
8. **Which contracts.** Any live award carries variations (consultant fee
   variations too); claims and schedules only on works packages, since
   client-to-builder and client-to-trade contracts are both head contracts.
9. **Chat.** Records and edits variations and typed assessments at once,
   with Undo. Approve or reject a variation, and Accept claimed, wait for a
   click. Chat never issues: it opens the claim's panel with the assessment
   in place and the PM presses Issue there ("From chat").

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Upload, row, background read, re-check | `uploadInvoice`, `invoiceReader.outcome`, `CreateInvoice`, `CommitInvoiceRead` | `internal/httpapi/invoices.go:158`, `:102`; `internal/store/invoices.go:61`, `:408` |
| Rows, cells, amounts, dates from the page | `Harvest`, `rowsOf` | `internal/invoices/harvest.go:153`, `:77` |
| Rules, one fan-out, bands, decisions | `NewDraft`, `question`, `Apply`, `Resolve`; `Thresholds.Band` | `internal/invoices/read.go:179`, `:569`, `:717`, `:788`; `internal/invoices/thresholds.go:84` |
| Money, GST, ABN | `ParseCents`, `GSTOf`, `DisplayCents`; `ValidABN` | `internal/invoices/money.go`, `internal/invoices/abn.go` |
| Ledger shares, Unallocated, checks | `Counted`, `Shares`, `Summarize`, `Check`, `Status` | `internal/invoices/ledger.go:78`, `:88`, `:144`, `:274`, `:420` |
| Claim fields typed by the PM (served, reference date, schedule served) | `PATCH /projects/{id}/invoices/{inv}`, `EditInvoice`, `cleanInvoiceValue` | `internal/store/invoices.go:738`, `:856` |
| The contract: award, lines, sources, lock | `Award`, `ReadAward`, `ErrAwardLocked`, `cost_line_sources`; works lines naming a package | `internal/store/awards.go:475`, `:794`, `:22`; `internal/db/migrations/032_procurement_award.sql`, `internal/db/migrations/059_works_contract_lines.sql` |
| Forecast and totals | `Forecast`, `Metrics` | `internal/costs/totals.go:71`, `internal/costs/plan.go:11` |
| Frozen document and download | `IssueSnapshot`, `CanonicalSnapshot`, `render.PDF`, issued-version triggers, `GET /reports/{r}/versions/{version}/file` | `internal/reports/snapshot.go`, `internal/reports/render/pdf.go`, `internal/store/report_issue.go:37`, `internal/db/migrations/020_reports.sql` |
| Business days and holidays | `program.NewCalendar`, `IndexOnOrAfter`, `DateOf`; the shared `BusinessDayAfter` (index, agreement 1); `libraryCalendar` | `internal/program/calendar.go:142`, `:344`, `:297`; `internal/store/program_prepare.go:318` |
| Undo, operations, route classes | capture trigger, `captured`, forwarded ops, `noOperation` | `internal/db/migrations/056_project_changes.sql`, `internal/store/change_undo.go:23`, `internal/httpapi/ops_more.go:338`, `internal/httpapi/ops_routes.go:9` |
| Tables, side panel, chips | `InlineCellEditor`, `RowMenu`, `PageSidePanel`, `Chip`, cost table styles | `web/src/table/`, `web/src/PageActions.tsx`, `web/src/Invoices.tsx:41`, `web/src/cost-tables.css` |
| Bench names | `routePaths` | `internal/httpapi/projects.go:329` |

## What is new

| Thing | Why an existing one cannot carry it | Est. lines |
|---|---|---|
| `variations` | `project_delivery_items` holds dates and text; a variation holds cents that flow into the cost plan, a contract and a line, and needs the capture trigger beside the cost rows | 30 SQL |
| `variation_documents` | links with a foreign key; an emailed message is filed as a document, so messages link the same way | 10 SQL |
| `progress_claim_lines` | a claim row has four amounts, a mapping and a reason; `invoice_allocations` holds one amount and is what counts | 30 SQL |
| Kind `progress_claim`, `invoices.settles_claim_id`, `awards.schedule_days`/`payment_days`, `reports.invoice_id`, kind `payment_schedule` | columns and values on existing rows | 30 SQL |
| `internal/invoices/claim.go`, `sopa.go` (new) | claim rows, `claim-1` questions, checks, figures; due dates | 410 Go |
| `internal/store/variations.go`, `claims.go` (new) | register writes, claim lines, issue | 450 Go |
| `internal/reports/payment_schedule.go` (new) | sections from figures; no template, no profile read | 90 Go |
| `internal/httpapi/variations.go` (new): `POST /projects/{id}/variations`, `PATCH /projects/{id}/variations/{v}`, `PUT /projects/{id}/invoices/{inv}/claim-lines`, `POST /projects/{id}/invoices/{inv}/schedule`, `PATCH /projects/{id}/procurement/packages/{pkg}/award` | nothing writes a variation, a claim row, a schedule or a contract's periods; the tab's read rides on the ledger | 160 Go |
| `knowledge/contracts/security_of_payment.yaml`, `internal/knowledge/contracts.go` (new) | Act periods per state; no file or loader holds contract knowledge today | 80 YAML, 30 SCHEMA, 45 Py |
| `web/src/Variations.tsx` (new) | the tab's panel; `VariationsTab` moves out of `Invoices.tsx`, which stays the ledger | 380 TSX |

Across all tasks (edits included, tests extra): about 1,700 Go, 500 TSX.
New: 3 tables, 5 write routes, 0 read routes, 0 screens, 1 component file.

## Data

Migration, next free number. Money is integer cents ex GST; null is unknown;
every key is composite with `org_id`.

```sql
ALTER TABLE invoices DROP CONSTRAINT invoices_kind_check, ADD CONSTRAINT invoices_kind_check
 CHECK (kind IN ('invoice','credit_note','not_invoice','progress_claim')),
 ADD COLUMN settles_claim_id uuid,  -- a tax invoice for certified money; not counted
 ADD FOREIGN KEY (org_id,project_id,settles_claim_id) REFERENCES invoices(org_id,project_id,id);
ALTER TABLE awards ADD COLUMN schedule_days smallint CHECK (schedule_days BETWEEN 0 AND 60),
 ADD COLUMN payment_days smallint CHECK (payment_days BETWEEN 0 AND 120);  -- the contract's; null not entered
CREATE TABLE variations (org_id, id, project_id, package_id, award_id, cost_item_id NOT NULL,
 number text UNIQUE per package, title, details, status IN (proposed,submitted,approved,rejected,withdrawn),
 claimed_cents, estimate_cents, approved_cents bigint, time_claimed_days int >= 0,
 submitted_on date, decided_on date, created_by, updated_at, version,
 CHECK ((status='approved') = (approved_cents IS NOT NULL)));
CREATE TABLE variation_documents (org_id, variation_id, document_id; PK all three; both ON DELETE CASCADE);
CREATE TABLE progress_claim_lines (org_id, id, project_id, invoice_id ON DELETE CASCADE, ordinal,
 kind IN (contract,variation,retention), label, page, cost_item_id, variation_id,
 mapped_by IN (rule,jev,user), band, to_date_cents, previous_cents, this_claim_cents,  -- as printed
 assessed_to_date_cents,  -- the PM's; null is not assessed
 reason text <= 1000, version);
ALTER TABLE reports DROP CONSTRAINT reports_kind_check, ADD CONSTRAINT reports_kind_check
 CHECK (kind IN ('pmp','rfp','rft','payment_schedule')), ADD COLUMN invoice_id uuid (FK invoices),
 ADD CHECK ((kind='payment_schedule') = (invoice_id IS NOT NULL));  -- one schedule report per claim
CREATE TRIGGER capture_change ON variations, variation_documents, progress_claim_lines;
CREATE TRIGGER claim_lines_issued BEFORE UPDATE OR DELETE ON progress_claim_lines;  -- refused once issued
```

Claim-level fields are rows in `invoice_decisions` (no new columns):
`reference_date`, `served_on`, `retention`, `schedule_served_on`, beside the
existing number, date, total, GST and ex GST. Variation sums, contract sum
to date, certified to date and due dates are computed at read, never stored.

## Tasks

**1. Act rows and the checker** (80 YAML, 30 SCHEMA, 45 Py, 70 Go, 80 test).
`knowledge/contracts/security_of_payment.yaml`, `knowledge/SCHEMA.md`,
`tools/check_knowledge.py`, `internal/knowledge/contracts.go` (new). Row: `id: sop.<state>`, `act`,
`business_day_excludes`, `schedule {section, business_days}`, `payment
{section, business_days, rule}`, `verified`, `primary_source`, `checked_on`,
`status: draft`. Test: the checker refuses `verified: true` without a
source, an unquoted count, an unknown state; the check passes.

**2. Migration and report guards** (110 SQL, 30 Go, 60 test).
The migration; three `captured` rows (area `costs`) in
`internal/store/change_undo.go:23`; `kind IN ('rfp','rft')` where a
package's report is found without a kind: `internal/store/reports.go:227`,
`internal/store/report_list.go:12`, `internal/store/procurement_schedule.go:33`,
`internal/store/profile_run.go:216`; `render.PDF` prints "Payment schedule"
(`internal/reports/render/pdf.go:154`). Test: `TestCapturedTablesMatchDatabase`;
a package with only a schedule still offers Create RFT; the trigger refuses.

**3. Variation register** (220 + 120 Go, 200 test).
`internal/store/variations.go`, `internal/httpapi/variations.go`,
`internal/store/awards.go` (`UndoAward` refused while a variation or claim
names the award). Number: the PM's or the next `V01`. Line: the award's only
line by rule, else the PM's pick. Approved needs an amount (zero or negative
allowed). Each write is a marked change (area `cost_plan`), bumps the cost
revision, wakes SSE. Test: 404 across orgs, duplicate number 409, approve
without amount 422, contract sum to date, undo restores.

**4. Variations in the figures** (70 Go, 25 TSX, 120 test).
`internal/invoices/ledger.go` (line approved and forecast variations;
contract remaining uses contract sum to date), `internal/store/invoices.go:628`
(sums per line), `internal/costs/totals.go:71` (draft items only: a contract
line's forecast adds approved and forecast variations; an unknown estimate
gives a partial total, as other metrics), `web/src/CostGrid.tsx:48` (two
columns leave `PLACEHOLDER`). Test: existing cost and ledger tests pass; a
frozen baseline reads the same before and after; one variation moves three
columns by the same amount.

**5. Reading a claim** (300 Go, 250 test). `internal/invoices/claim.go`.
`ClaimRows`: rows with two or more amounts under a header naming to date,
previous and this claim (or this period). Rules, then one `jev.Call` as in
the table above. Checks (flags only): to date −
previous ≠ this claim; rows ≠ printed gross; gross − retention + GST ≠
total; GST ≠ 10 % of net; builder's previous ≠ SiteWise's certified; claimed
to date over the line's contract sum to date; a claim on an unapproved
variation; two claims for one reference date (QLD s 75(4)); no "supporting
statement" on a head-contract claim (QLD s 75(7)). Test: synthetic PDF and
XLSX claims in `internal/invoices/testdata/`; a fake asker shows one call,
none when rules settle all; grey keeps rule values.

**6. Intake and the ledger** (180 Go, 150 test).
`internal/httpapi/invoices.go:158` (`?kind=progress_claim&package=` on the
existing upload), `internal/invoices/read.go` (the words-plus-ABN rule picks
the claim draft before any call), `internal/store/invoices.go` (claim
context; commit writes `progress_claim_lines` and a retention row),
`internal/invoices/ledger.go` (`Counted`, `Shares` per decision 3;
`settles_claim_id`; the link rule and flag). No text: rows from the
contract's lines, amounts unknown, for the PM. Test: drop to row within
`claim_intake`; re-drop "Already filed"; lines plus Unallocated equal the
total with a scheduled claim and its linked invoice.

**7. Assessment** (150 Go, 120 test). `internal/store/claims.go`.
`PUT .../claim-lines` takes rows `{id, assessed_to_date, reason, mapping}` or
`{accept_claimed: true}`. Per row, scheduled = assessed to date − SiteWise's
certified to date. Totals: gross, retention withheld, net, GST by `GSTOf`
(zero when the claim printed zero, flagged), payable. Issue blocked while a
row is unassessed, or below claimed with no reason (QLD s 69(c); NSW s 14(3)
draft). Test: a retention release, unmapped rows to Unallocated, a write
after issue refused.

**8. Due dates and the banner** (110 Go, 30 TSX, 150 test).
`internal/invoices/sopa.go`, `internal/store/invoices.go` (a due record per
open claim in the ledger view; the same record is a rule in
`reports.Attention`, built first under plan 5, so the left-nav mark, chat
and the report show it), `web/src/Costs.tsx` (the one banner in the
product: red past due and
unserved; amber within two business days or any "serve by" or unknown; a
count on the tab). Business days: `program.NewCalendar` over the state's
reviewed `public_holiday` days and the Act's excluded spans, never the
project calendar's RDOs or shutdowns (they would make a date late); the
shared `program.BusinessDayAfter`, written here if plan 4 has not landed. Test: QLD across 22 Dec–10 Jan, weekend service, missing
year, unverified Act with and without a contract period, no state, a
contract period longer than the Act's.

**9. Issue the schedule** (90 + 120 Go, 120 test).
`internal/reports/payment_schedule.go`, `internal/store/claims.go`
(`IssuePaymentSchedule`, one transaction: the claim's report, an issued
version with snapshot, `render.PDF`, the file row as `IssueReport` writes it,
then the claim's `invoice_allocations` from the scheduled amounts),
`POST .../schedule` in `internal/httpapi/variations.go`. Sections: parties
(client from Profile, firm from the award), the claim answered, the Act by
name with a section only from a verified row, the table (row, claimed,
scheduled, reason), totals, reasons for withholding, issued by and date.
Late issue is allowed and says so. Send opens plan 1's dialog to the firm's
email; the mail service's acceptance sets `schedule_served_on` by rule;
before plan 1, the PM types it. Test: stable snapshot hash; re-issue is a
new version linked to the first.

**10. The tab** (380 + 60 TSX, 80 test). `web/src/Variations.tsx`,
`web/src/invoiceApi.ts`, `web/src/Costs.tsx:175`. Contract picker (two or
more contracts only), header figures, variations table (ghost row to add,
inline edits, status menu, drop a file on a row to file and link it), claims
table, claim panel (chips; rows: line, claimed to date, certified before,
this claim, assessed to date, this schedule, reason; Accept claimed; Issue;
Send; Served on). Every figure from the server. Test: e2e add, approve,
drop, accept, issue, banner clears.

**11. Operations, chat reads and the key** (80 Go, 80 test). See "From
chat". `internal/httpapi/ops_more.go`:
`variations.add`, `variations.edit`, `claims.assess` (Undo, `Figures`
money); `variations.decide`, `claims.accept_claimed` (Click);
`claims.contract_periods` (prepared: `awards` has no capture trigger);
served dates through the existing `invoices.edit`.
`internal/httpapi/ops_routes.go`: `POST .../schedule` in `noOperation`
("issuing leaves the building; the PM clicks Issue").
`internal/httpapi/chat_read.go`: `read(variations)`, printed by Go.
`web/src/Variations.tsx` registers its two tables and the claim panel with
plan 0's `chatView.ts`. Test: `TestWriteRoutesHaveOperations` with no route
of this plan pending, `TestOnlyChatReachesTheModel`, a figure not in the
message is refused, the six key cases.

**12. Benches and the answer key** (80 + 100 Go). `bench/budgets.json`
(five rows), `internal/httpapi/projects.go:329`, fixtures;
`cmd/invoice-eval/main.go` gains `-claims` (reads `data/eval/claim/`,
shuffles row options, fits cut-offs for `claim-1`).

Order: 1–2; 3–4 (variations stand alone); 5–9; 10–12.

## From chat

Plan 0's rule table governs (`2026-11-18-chat-agency.md`).

- **Reads.** `read(variations)`: per contract, the contract sum, approved
  variations, the sum to date, certified to date and retention held; the
  variation register; each claim with its schedule date and payment date
  and, for each date, the inputs it was worked out from.
- **At once, with Undo.** `variations.add`, `variations.edit`,
  `claims.assess`. Amounts and days are `Figures`: they must be in the PM's
  words or a read of this turn.
- **Waits for a click.** `variations.decide` (approve or reject),
  `claims.accept_claimed`, `claims.contract_periods`.
- **Opened filled in.** The claim's panel, for the PM to press Issue; then
  Send (plan 1) with the schedule attached. The schedule route stays
  `noOperation`.
- **Screen.** Select and filter variations and claim rows; open a claim.
- **Named filters.** Variations awaiting a decision; claim rows not yet
  assessed; claims with a schedule due.
- **Key cases.** "What is the contract sum to date on Main Works"; "when is
  the payment schedule due on claim 4 and how was that worked out"; "add a
  variation for rock excavation, $12,400 claimed, three days"; "show the
  rows I have not assessed"; "approve V03 at $11,000" (prepared); "issue
  the schedule for claim 4" (opens the panel, issues nothing).

## Edge cases

| Case | What happens |
|---|---|
| Scanned claim, no text | Stored, "Not read"; rows from the contract's lines, amounts unknown; the PM types |
| No live award on the package | No drop zone; on the Invoices tab it stays an invoice |
| Two live works contracts | Drop on the right one; the words rule needs the ABN rule first |
| A row matches two lines; more than 254 lines or 80 rows | No rule; Jev (blank until calibrated) or not asked; the PM picks |
| Row arithmetic fails; builder's previous differs | Flagged; amounts as printed; the schedule uses SiteWise's certified |
| Revised claim, same number | The existing "replaces"; refused once the old one has an issued schedule |
| Served date unconfirmed; served on a weekend | Drop day by rule, amber; counting starts on the next business day |
| Contract period longer than the Act's | The Act's date, flagged "the contract cannot extend this" |
| Issued after the due date | Allowed; red; the PDF says so |
| Undo after issue; award undo with a variation | Refused, with the reason |
| Tax invoice differs from payable | Not linked by rule; flagged; the PM decides |
| Cost plan in GST-inclusive figures | Remaining unknown, as today |
| Jev down | Grey; rules stand; background re-check, as invoices |
| Wrong org or bad id | 404 |

## Risks

1. **NSW unverified.** Most projects are NSW; until decision 1, NSW shows
   contract and "serve by" dates only.
2. **No holidays.** Every date is "serve by" until decision 2. Local show
   days are never loaded, so a date can only fall early.
3. **Reports reuse.** A lookup that ignores kind would take a schedule for
   the package's RFT; task 2 fixes four queries and a test holds them.
4. **Fan-out size.** 40 rows and six header questions may miss 1 s at p50;
   `claim_intake` measures it. The fall-back is a lower row cap, never a
   second call.
5. **Forecast Final Cost changes meaning** after award (decision 5).
6. **Consultant invoices** can be payment claims under the Acts; they stay
   invoices without dates.
7. **Not legal advice.** Dates are arithmetic on the Act rows and the PM's
   inputs, and the screen names the inputs used.

## Conflicts with sibling plans

- **Plan 1** carries the schedule out (its Send) and offers "Read as
  progress claim" and "Add variation" on an email; it computes no dates.
  Its contract time bars are deferred, so this plan creates
  `knowledge/contracts/` and `internal/knowledge/contracts.go`.
- **Plan 4** counts RFI days on the project calendar (RDOs, shutdowns);
  Security of Payment counts public holidays and the Act's excluded days
  only. One helper, `program.BusinessDayAfter`, two inputs; whichever plan
  lands first writes it.
- **Plan 5** reads Claimed to Date from `invoices.Summarize`, which now
  includes certified claims and variation columns; its "Variations are not
  yet recorded" note goes.
- **The register** (plan 4) is `delivery_items`. Variations are their own
  table (money, the capture trigger) and are not listed there.
- **Migrations.** Plan 5 also changes `reports_kind_check`; the later one
  keeps every kind.
