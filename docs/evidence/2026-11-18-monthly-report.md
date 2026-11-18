# The project report: what was built and measured

Date: 18 November 2026. Branch `wave/report` (not merged, not pushed, not
deployed). Plan: `docs/plans/2026-11-17-monthly-report.md`. Everything here
was run on the development machine against a private PostgreSQL 17 schema
with a stand-in model; the live server was not contacted, and no live model
or Jev call was made. The template `tpl.monthly-report` and the command
key's new `report` group are drafts the owner has not reviewed.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. The kind | `monthly` beside the PMP, one per project: a second create (a double click, chat and the screen together) returns the one there is. Migration 070 widens the kind and package checks, adds the one-per-project index, widens the change log's areas and captures `report_edits`. `projectLevel(kind)` replaces the PMP-only package checks. Four pages allowed, three the target. | `internal/db/migrations/070_project_report.sql`, `internal/store/reports.go`, `knowledge/reports/templates.yaml`, `internal/reports/render/pdf.go` |
| 2. The loader | One repeatable-read snapshot: the cost plan and `costs.Summarize` (overall and by package), the invoice ledger, the Program engine over the stored inputs, the procurement board, live awards and issued RFP dates, delivery records, the repository's filings and their discipline, implications and reading counts. The ledger, the board and the delivery read now take the caller's transaction; their public functions keep their signatures. Two source stamps join `SourceState.Domains`: the Program's version and the invoices' count plus version sum. | `internal/store/report_monthly.go`, `invoices.go`, `comparison.go`, `delivery.go` |
| 3. The draft | `reports.AssembleMonthly`, one function per section: At a glance, Needs attention, Changes, Cost, Program, Procurement, Registers, Next period. Every sentence is Go's. Unknown stays unknown: a total that is not complete prints its known part and how many lines are unknown, never 0; remaining is not known on a plan in inc GST, as on the grid. Lists are capped with "And N more in SiteWise". | `internal/reports/monthly.go` |
| 4. Changes since report N | Compares this draft's blocks with the last issued report's frozen blocks by id: cost totals and sections exactly (and the known part of a partly known total), completion in calendar days (working days on a relative Program), milestones by activity id (moved, added, removed; a renamed one keeps its id), packages (status, award, award withdrawn, removed), register items opened, closed and removed, documents filed. A block built by another method says "not compared". The first report says it is the first. | `internal/reports/monthly_changes.go` |
| 5. Needs attention | Was built before this branch. This plan's rule was added: planned completion later than the last issued report printed (`program.completion_later_than_report`), a pure function with a table test. Chat's loader fetches the two figures concurrently with its other parts. | `internal/reports/attention.go`, `internal/httpapi/attention.go` |
| 6. Comments | `comment:summary` opens At a glance; Cost, Program, Procurement, Registers and Next period end with `comment:<section>`. The generated text is Go's sentence; the basis holds the section's figures and the period start, so last period's comment comes forward flagged and issue waits until it is kept (one save), rewritten or restored. | `internal/reports/monthly.go` |
| 7. The screen | `ProjectReportButton`, the one item on the Register's toolbar, finds or creates the report and shows it in the shared reader (Back returns to the Register); `useProjectReportDialog` lets chat open it there. The reader titles it "Project report No. N", hides the budget checkbox for this kind, registers its issue form for chat, and re-reads on invoice and Program events while it is open. | `web/src/ProjectReportButton.tsx`, `Reports.tsx`, `Delivery.tsx`, `Project.tsx`, `Chat.tsx`, `chatView.ts`, `reportApi.ts` |
| 8. Bench | `report_assemble_monthly` (300 / 1,000 ms) in `bench/budgets.json` and the committed-budget test; `TestProjectReportBudget` holds it on the heavy fixture; the intake bench gains `monthlyReport`. | `internal/httpapi/report_monthly_budget_test.go`, `cmd/intake-bench/reports.go` |
| 9. Send and guest view | Not built: waits on plans 1 and 2 (section 5). | — |
| 10. Operations, reads, key | `reports.update`, `reports.comment`, `read report`, the view tool's `report` dialog, the undo finisher for `report_edits`, eight key cases. | `internal/httpapi/ops_reports.go`, `chat_more.go`, `chat_view.go`, `internal/store/report_undo.go`, `data/eval/chat/` |

### Where each figure comes from

| Figure | Function |
|---|---|
| Budget, approved contract, forecast final cost by section and total | `costs.Summarize(plan, "overall")` on `readCostPlan` (the Cost Plan grid's `CostTotals`) |
| Claimed to date by section and total, Unallocated | `invoices.Summarize` through the ledger read (`invoiceLedgerTx`, the grid's `InvoiceLedger`) |
| Remaining | the ledger's `SectionsContractRemaining` and `ContractRemaining`: approved contract less invoiced, as the grid |
| Invoices this period | the ledger's counted invoices received since the period start, ex GST, and `invoices.Status` needs review |
| Completion, range, critical path, milestones | `engine.Compute` through `computeState` (the Program screen's `LoadProgram`) |
| Firms, quotes, quotes to check | the procurement board (`procurementBoardTx`) counted by `reports.PackageQuotes`, as chat and the filter count them |
| Awarded firm and date | live `awards` rows (undone_at IS NULL), the `ReadAward` query shape |
| Contract sum of an awarded package | `costs.Summarize(plan, "package")`, the package's approved contract when complete |
| Registers | the register's own read on the draft's snapshot (`readRegisterTx`, as the Register screen and chat), open by `delivery.Open` |
| Needs attention | `reports.Attention` |

### The tests that hold the figures

`internal/store/report_monthly_test.go`: every cost figure equals
`CostTotals` and the ledger, the completion equals `LoadProgram`
(`TestMonthlyReportFiguresEqualTheScreens`); an awarded package's firm and
contract sum equal `ReadAward` and the cost plan by package
(`TestMonthlyReportAwardEqualsReadAward`); two issues list every movement,
a carried comment holds issue until kept, and both issues stay frozen and
listed newest first (`TestMonthlyReportTwoIssuesMovementAndComments`); a
removed package is listed (`TestMonthlyReportListsARemovedPackage`); an
empty project prints one honest line per surface and another org gets 404
(`TestMonthlyReportCreatesOnceAndIsScoped`). `internal/reports/monthly_test.go`
covers unknown totals, inc GST, a relative Program, missing durations, 40
packages capped, movement with a renamed and a removed milestone, a method
change, a comment flagged by a new period, and the completion rule.
`internal/httpapi/report_monthly_test.go`: the operations, Undo of a
comment and of a clear, foreign ids, the issue form opened and nothing
issued, a comment with the model's own figure refused, and the report's
Needs attention equal to chat's attention read.

## 2. Speed

`TestProjectReportBudget`, twenty rounds on the plan's heavy fixture
(1,000 Program rows of which 80 are milestones, 2,000 links, 1,000
invoices, 30 packages, 200 register items, a previous issue), through the
real HTTP handlers in process. Each round: Update, read, keep each flagged
comment and write the summary, issue, download.

| Budget row | Budget p50 / p90 | Measured p50 / p90 |
|---|---|---|
| `report_assemble_monthly` (`POST /reports/{r}/draft`) | 300 / 1,000 ms | 69 / 75 ms |
| `report_read` (`GET /reports/{r}`, with the page count) | 100 / 250 ms | 37 / 41 ms |
| `report_write` (`PUT /reports/{r}/edits/{target}`) | 100 / 250 ms | 20 / 22 ms |
| `report_issue` (freeze, A4 PDF, store) | 100 / 250 ms | 48 / 50 ms |
| `report_export` (issued PDF download) | 100 / 250 ms | 11 / 11 ms |

The draft fits its four pages on that fixture (the test fails if not).
Twenty rounds, so a query PostgreSQL turned to a slow generic plan after
five calls would show. `TestBenchEndToEnd` runs the intake bench's
`monthlyReport` case; the bench itself was not run over the private
corpus.

## 3. What chat can do with the report

| Act | How |
|---|---|
| Read the draft and what moved | `read report`: every block in order, the cost table's amounts behind tags, PM comments marked, flagged comments named, issued versions listed |
| Prepare it | `reports.update` creates the report if there is none and runs Update. It applies at once; the draft is derived, so there is nothing to undo |
| Write or clear a comment | `reports.comment` applies at once with Undo. Its text passes the figure guard: a number nobody typed and no read printed this turn is refused |
| Open it, or its issue form | `view` with dialog `report` (and `issue: yes`). The PM presses Issue; `POST /reports/{r}/issue` keeps its `noOperation` reason word for word |
| Send it | Not built (plan 1): the fixed words |

Every report operation first finds the report among this project's own
reports, so another project's or org's id is refused before any route
runs (the report routes are keyed by report, not project).

### The key

Eight draft cases, `pr1` to `pr8` (group `report`): two reads, two changes,
a draft comment, opening the issue form without issuing, a moved-since
question and Send (the fixed words). `TestScreenCasesScoredOffline` scores
seven with the stand-in model, all pass; with three expectations changed
by hand those three failed with the reason named. `pr7` is the model's
words alone.

## 4. Routes

No new route. Three were pending an operation and now have one:

| Route | Operation |
|---|---|
| `POST /reports/{r}/draft` | `reports.update` (no Undo: derived) |
| `PUT /reports/{r}/edits/{target}` | `reports.comment` (Undo) |
| `DELETE /reports/{r}/edits/{target}` | `reports.comment` with clear (Undo) |

`POST /projects/{id}/reports` stays `reports.create` (held for a click, as
before); `POST /reports/{r}/issue` stays `noOperation`.

## 5. What waits

- **Plan 1 (correspondence):** Task 9's "Send issue N…" on an issued version
  and chat's "send report 4 to the client" (it opens Send with the frozen
  PDF). Until then chat gives the fixed words.
- **Plan 2 (guest access):** the guest page lists every issued version,
  newest first. Issued versions are frozen by the existing triggers and
  listed newest first by `GET /reports/{r}` (`issues`) with their frozen
  PDF at `GET /reports/{r}/versions/{v}/file`. Nothing was changed for
  guests here.
- **Plan 4 (the register)** was merged into this branch on 18 November:
  the entry point is on the Register's toolbar, `REPORT_PLACE` is
  `"register"`, and the report lists the register's kinds by its own names
  and order (`rfi`, `notice`, `action`, `risk`, `issue`, `decision`,
  `approval`). Plan 6's consent conditions join `registerKinds` in
  `internal/reports/monthly.go` once that kind exists.
- **Plan 3:** variations ("Variations are not yet recorded in SiteWise")
  and the `payment_schedule` kind. Migration 070 lists `pmp, rfp, rft,
  monthly`; whichever lands second lists both.
- **The owner:** decision 11 (the money an outside reader sees; the default
  is taken: all five columns, contingency included, each awarded firm's
  contract sum), the template's wording (draft), and the key.

## 6. Where the code differs from the plan

- **Tables, not one block per row.** Milestones, packages, register items
  and next-period items print as one table block each (`program:milestones`,
  `proc:table`, `reg:<kind>:summary`, `next:table`) to keep inside four
  pages. Every milestone, package and register item is still kept by id in
  the block's basis, so Changes matches by id as the plan says.
- **Comments** sit under At a glance (summary), Cost, Program, Procurement,
  Registers and Next period; not under Needs attention or Changes, which are
  lists.
- **Citations** stay in the issue record and are not printed in the PDF, as
  for an RFP: printing every block's reference would not fit four pages.
  Each section names its screen.
- **Procurement counts** come from the procurement board on the draft's
  snapshot, not a new aggregate query, so they cannot disagree with
  Procurement or Needs attention. Measured on 30 packages with no quotes;
  not measured with many quotes.
- **`reports.update` creates the report** when there is none, so "prepare
  this month's report" is one operation. It applies at once with no Undo
  (the plan's "at once, with Undo": there is nothing to put back).
- **The figure guard on a comment** checks the comment's numbers against
  the PM's message and the turn's reads: an operation field carries no
  tags, so the model writes figures exactly as `read report` printed them.
- **Re-ticking** on invoice and Program events reloads only an open report,
  not every screen behind it.
- **The PDF header** reads "Issued 17 November 2026" for this kind.

## 7. What was not measured or not done

- No live model run; the `report` group is unscored against a live model.
- The intake bench was not run over the private corpus; `TestBenchEndToEnd`
  ran it on its fixture.
- The full `tools/check.ps1` gate was not run.

## 8. Known limits

- A profile rebuild marks the draft stale (`profile_build`), as for every
  report, although this report reads no profile values; issue then needs
  Update or a reason.
- On a relative Program the next period lists no Program rows (nothing is
  dated) and says so.
- Documents filed this period are counted by when they reached SiteWise.
