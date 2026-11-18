# Monthly project report: implementation plan

Date 17 November 2026; revised 18 November. Lane A (it restates cost figures, awards and dates a
client relies on). Status: plan only, nothing built.

This turns the requirements framework `docs/design/2026-11-12-project-report.md`
into a build. It does not rewrite it. Its 12 open decisions are answered
below with a recommended default each; the owner decides. Sibling plans:
sending is plan 1 (`2026-11-17-correspondence-transmittals.md`), the client's
view is plan 2 (`2026-11-17-guest-access.md`), variations and claims plan 3
(`2026-11-17-variations-progress-claims.md`), RFIs plan 4
(`2026-11-17-rfis-review-comments.md`), consent conditions plan 6
(`2026-11-17-consent-conditions.md`).

**Revised 18 November** after the review in `2026-11-17-next-wave-index.md`.
The report opens from the register's toolbar and from chat, not from
Procurement's menu. `Attention` (Task 5) is built first, as its own change.
The LLM summary button is gone: the PM asks chat for a comment ("From
chat"). The earlier text is at git ref
`refs/backup/next-wave-plans-2026-11-18`.

## In one page (product requirements)

**Outcome.** On the Register the PM presses "Project report", or asks chat
to prepare it. In under a second the draft is on screen: cost by section
(budget, approved contract, forecast final cost, claimed to date,
remaining), Program completion and the critical path, each package's
procurement status, the open register items, what needs attention, and what
is due next. Every figure is the one the Cost Plan, Program or Procurement
screen shows. The PM types a comment under any section, presses Issue, and
SiteWise freezes the report and makes an A4 PDF to send. Next month Update
builds the next draft, and the report says what moved since the last one:
forecast up $40,000, completion 9 days later, two packages awarded.

**Why.** Time-to-decision: the client decides from three or four pages, not
five screens, and the PM writes the report in minutes, not a day. Answer
trust: no figure is retyped, so the report cannot disagree with the
workbench, and every line carries its source.

**Where it lives.** No new screen and no new left-nav entry. The report
opens in the shared reader (`web/src/Reports.tsx`), from one item, "Project
report", on the register's toolbar (plan 4 switches the register on in the
idle nav slot and lands before this plan). A project report is about the
whole project, so it does not sit in Procurement's menu. Chat opens it too.

**What reads what.**
- **Jev: none.** All data is already structured. Update, open, edit, issue
  and export are Go over saved records (framework rule 1). A consultant's or
  builder's own monthly report dropped into the repository is filed as now
  (one Jev fan-out, unchanged) and only counted under "documents filed this
  period" by its filing fields. Reading its contents into this report is not
  needed for v1; if the owner wants it later it is its own plan (one
  background fan-out, a proposal).
- **Go:** every figure, read at Update from the existing rollups:
  `costs.Summarize` (`internal/costs/totals.go:139`) for budget, approved
  contract and forecast; `invoices.Summarize` (`internal/invoices/ledger.go:144`)
  for claimed to date, remaining and the Unallocated row, exactly as the
  Cost Plan grid does (`web/src/CostGrid.tsx:43-47`); `engine.Compute` via
  `computeState` (`internal/store/program.go:308`) for completion range,
  critical path and milestone dates; packages, awards and invitations for
  procurement; `project_delivery_items` for registers. Movement since the
  last report compares this draft with the last issued report's frozen
  blocks. Plain sentences ("Forecast final cost is $4,215,000 ex GST,
  $85,000 under budget") are Go templates filled from those figures.
- **LLM: none on any screen path.** Commentary is the PM's own words over
  Go's sentences. The model's one door is the strip: asked in chat, it
  drafts a section's comment, with every figure printed by Go ("From
  chat"). There is no button for it on the report.

**Speed budgets** (all existing rows except one; no path waits on a model).

| Path | Route | Bench row | p50 / p90 ms |
|---|---|---|---|
| Open the report (includes the page count) | `GET /reports/{r}` | `report_read` | 100 / 250 |
| Create (first open only) | `POST /projects/{id}/reports` | `report_write` | 100 / 250 |
| Update (assemble the draft) | `POST /reports/{r}/draft` | `report_assemble` | 300 / 1,000 |
| Update on the heavy fixture: 1,000 program activities and 2,000 links, 1,000 invoices, 30 packages, 200 register items, a previous issue | same | **new** `report_assemble_monthly` | 300 / 1,000 |
| Comment, keep or restore wording | `PUT`/`DELETE /reports/{r}/edits/{target}` | `report_write` | 100 / 250 |
| Issue: freeze, A4 PDF render, store | `POST /reports/{r}/issue` | `report_issue` | 100 / 250 |
| Download the issued PDF; draft PDF or Word | `GET /reports/{r}/versions/{v}/file`, `GET /reports/{r}/export` | `report_export` | 100 / 250 |

The monthly bench case samples into `report_read`, `report_write`,
`report_issue` and `report_export` as well, so those gates now cover it.

**Done means.**
1. From the register's toolbar the PM opens one project report (a
   double click never makes two) and Update assembles it inside budget.
2. A test proves every cost, program and procurement figure in the draft
   equals the owning screen's figure (`CostTotals`, `InvoiceLedger`,
   `LoadProgram`, `ReadAward`) on one fixture.
3. A two-issue fixture: report 2 lists every movement and every removal;
   report 1 says it is the first.
4. A PM's comment survives Update; when its figures or period change it is
   flagged for review and issue waits until it is kept, rewritten or cleared.
5. Issue freezes the wording, figures and sources; the A4 PDF is at most
   four pages or issue is refused with the page count; the download returns
   the frozen bytes.
6. A surface never used (no Program, no cost plan, no RFIs) prints one honest
   line, never a blank or a guess.
7. No Jev call and no model call on any path above (assemble with nil Jev and
   LLM clients; `TestOnlyChatReachesTheModel` still passes). Bench rows pass.

**Out of scope.** Sending mail (plan 1); client links (plan 2); new
registers (plans 3, 4, 6; read once built); scheduled reports; charts;
portfolio or per-client templates; Program progress; entering records from
the report; the framework's Scope, Planning, Quality and Construction-status
sections (its slices 4 and 5); an LLM button on the report (chat drafts a
comment when asked).

**Owner decisions** (1 to 12 are the framework's; each has a default). The
plan takes its defaults; the owner decides 11 only, because it is about what
an outside reader sees.

1. **Name.** Default: "Project report No. 4" with the period printed under
   it ("17 October to 17 November 2026"). Not "Monthly", because the period
   is whatever lies between issues. The code kind is `monthly`; only the
   printed title follows this decision.
2. **First reader.** Default: the client (the issued PDF). The draft on
   screen doubles as the PM's own catch-up.
3. **Length.** Default: four pages allowed, three the target, like an RFP.
   Lists are capped in Go with an honest "and N more in SiteWise".
4. **Order.** Default: At a glance, Needs attention, Changes since report N,
   Cost, Program, Procurement, Registers, Next period. Changes are grouped
   by area, largest movement first.
5. **Period.** Default: since the last issued report. The first report is
   "to date".
6. **"High risk".** Default: no ranking. Needs attention lists explicit
   states only (the rules chat already uses). Risks are listed oldest first.
7. **Status words.** Default: computed only where the rule is exact: "under
   budget by $X", "over budget by $X", "not yet known"; completion "later",
   "earlier" or "unchanged since report N". No PM traffic lights in v1.
8. **Where risks, approvals and actions are entered.** Default: on the shared
   register view plan 4 brings back (`web/src/Delivery.tsx`). The report
   only reads.
9. **Planning authorities.** Default: approvals from the register now; plan
   6's consent conditions join the Registers section once they exist.
10. **Construction progress.** Default: none in v1. The Program section says
    its dates are planned dates computed from durations and links.
11. **Money an outside reader sees.** Default: the client's own figures in
    full: all five columns by section and total, contingency included, and
    each awarded firm's contract sum. Never an unawarded firm's price. The
    "Include internal budget amounts" checkbox is hidden for this kind.
12. **The PMP.** Default: both stay; the PMP is untouched. Note: no screen
    opens it today (`web/src/Procurement.tsx:317` filters it out). Revisit
    after three issued project reports.
13. **Where it opens (settled 18 November).** The register's toolbar, and
    chat. The left nav's idle slot is the register's (plan 4).
14. **Next period window (new).** Default: 28 days from the as-at date, since
    the next issue date is not known in advance.
15. **LLM drafting (settled 18 November).** No button. The PM asks chat
    for a comment and the pencil guard applies ("From chat").
16. **Last month's comments (new).** Default: they come forward into the new
    draft flagged "review", and must be kept (one save), rewritten or cleared
    before issue. Alternative: start each period with no comments.

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| A report with versions, draft, Update, issue, numbering | report engine; `assembleDraftTx`, `reportSectionsTx` | `internal/store/reports.go:317`, `:430` |
| Freeze, integrity hash, A4 PDF with a page gate | `IssueReport`, `CanonicalSnapshot`, `render.PDF`, `PageLimit` | `internal/store/report_issue.go:37`, `internal/reports/snapshot.go`, `internal/reports/render/pdf.go:33` |
| The last issue's frozen blocks (the month-on-month history) | `reportHistory` returns the last issued sections | `internal/store/report_history.go:18` |
| PM wording kept across Update; flagged when its source changes | `ApplyEdits`, `EditReport`, `ResetReportEdit` | `internal/reports/edits.go:51`, `internal/store/report_edits.go:18` |
| A source on every block | `Cite` | `internal/reports/citations.go:43` |
| Stale notice and "issue the saved state with a reason" | `StaleReasons`, issue form | `internal/reports/deps.go:38`, `web/src/Reports.tsx` |
| Screen-diff list of changes | `ChangesSinceIssue` (kept for the screen) | `internal/reports/kinds.go:264` |
| Draft PDF and Word, print | `getReportExport`, `internal/export` | `internal/httpapi/report_settings.go:211` |
| Cost figures | `readCostPlan`, `costs.Summarize` | `internal/store/costs.go:51`, `internal/costs/totals.go:139` |
| Claimed to date, remaining, Unallocated | `invoices.Summarize`, `invoices.Status`, `DisplayCents` | `internal/invoices/ledger.go:144`, `internal/invoices/money.go:72` |
| Program dates, range, critical path | `readProgramInputs` (takes a tx), `computeState` | `internal/store/program.go:265`, `:308` |
| Award firm and date | `awards` rows (undone_at IS NULL), `ReadAward` query shape | `internal/store/awards.go:794` |
| "Needs attention" rules | chat's `readAttention` (explicit states only) | `internal/httpapi/chat_more.go:165` |
| Registers | `project_delivery_items`, `delivery.Statuses` | `internal/delivery/items.go`, migration `019_delivery.sql` |
| The reader screen | `Reports` component and its issue form | `web/src/Reports.tsx` |
| Showing the reader inside a panel | Procurement's view state, copied by the register; `data.reports` already lists the project's reports | `web/src/Procurement.tsx:310`, `web/src/reportApi.ts:31-33` |
| Live refresh | report/costs/packages/delivery events re-tick reports | `web/src/Project.tsx:385` |
| Benches and fixtures | `reports`, `programBenchCreate`, `seedLedger`, `TestCommittedBudgets` | `cmd/intake-bench/reports.go`, `program.go:141`, `invoices.go:308`, `internal/latency/gate_test.go:124` |
| Chat creating a report | `reports.create` operation (already exists) | `internal/httpapi/ops_more.go:374` |

## What is new

| Thing | Why an existing one cannot carry it | Est. lines |
|---|---|---|
| Report kind `monthly` and template `tpl.monthly-report` (`status: draft`) | Existing kinds are a package's RFP/RFT or the PMP plan; a status report has its own sections and a period | Go 40, SQL 10, YAML 15 |
| `reports.AssembleMonthly` (pure) | `Assemble` requires the seven RFP section ids (`internal/reports/assemble.go:116`) and is package-scoped throughout | Go 220 |
| `monthlyInputTx` loader | `reportSnapshot` (`internal/store/report_snapshot.go:18`) reads works, scope and gaps this report never prints, and none of Program, the ledger or awards | Go 130 |
| One aggregate procurement query | `ProcurementBoard` (`internal/store/comparison.go:437`) builds every firm's price columns in its own transaction; too heavy for the assemble budget and the report shows no unawarded prices | inside the 130 |
| Movement since the last issue | `ChangesSinceIssue` compares text; the report needs amounts and days (forecast up $40,000, 9 days later) | Go 80 |
| `reports.Attention` pure rules | Moved out of chat's `readAttention` so chat and the report cannot disagree; a few rules added | Go 40 new, 70 moved |
| One toolbar item on the register that shows the reader; hide one checkbox; two event kinds | — | TSX 50 |
| Operations `reports.update` and `reports.comment`; `read(report)`; the capture trigger on `report_edits` | Report routes are keyed by report, so chat's forwarder cannot reach them today (chat plan, second leg: 8 report routes pending) | Go 60, SQL 3 |
| Bench case and one budget row | Existing PMP case has no program, ledger or previous issue | Go 60, JSON 1 |

Totals: no new table, endpoint or screen; about 650 new lines of Go and 50
of TSX, plus tests.

## Data

No new table. History is the previous issued `report_versions.sections`
(frozen, with each block's figures in its basis). Comments are
`report_edits`. One migration at the next free number:

```sql
-- One project-level report kind beside the PMP; one per project.
ALTER TABLE reports DROP CONSTRAINT reports_kind_check;   -- generated name; confirm in pg_constraint
ALTER TABLE reports ADD CONSTRAINT reports_kind_check CHECK (kind IN ('pmp','rfp','rft','monthly'));
ALTER TABLE reports DROP CONSTRAINT reports_check;        -- the (kind='pmp' OR package_id IS NOT NULL) check
ALTER TABLE reports ADD CONSTRAINT reports_package_check CHECK (kind IN ('pmp','monthly') OR package_id IS NOT NULL);
CREATE UNIQUE INDEX reports_one_monthly ON reports(org_id, project_id) WHERE kind = 'monthly';
-- A comment written from chat has Undo.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON report_edits FOR EACH ROW EXECUTE FUNCTION capture_change();
```

Staleness needs two more source stamps, read in the same transaction and
kept in `SourceState.Domains` (a JSON map, no column change): `program` (the
`programs.version`) and `invoices` (count and sum of `invoices.version` for
the project). No new revision domain.

Stable block ids (the movement and the comment flags depend on them):
`cost:table`, `cost:invoices`, `program:completion`, `program:critical`,
`program:ms:<activity id>`, `proc:<package id>`, `reg:<kind>:summary`,
`reg:item:<id>`, `next:<source>:<id>`, `attention:<rule>:<subject id>`,
`moved:<block id>`, `comment:<section>`. Figures sit in each block's basis as
exact decimals or ISO dates, so the next report can compare them.

## Tasks

**Task 1. The kind.** Files: the migration; `knowledge/reports/templates.yaml`
(`tpl.monthly-report`, sections `glance, attention, changes, cost, program,
procurement, registers, next`); `internal/knowledge/reports.go:75` (accept
the kind); `internal/store/reports.go` (template map at `:122`; `CreateReport`
returns the existing monthly report on conflict; `insertReport` at `:179`
titles it "Project report"); one helper `projectLevel(kind)` replacing the
`kind == "pmp"` package checks at `reports.go:136,151,321`,
`report_edits.go:134,207`, `report_issue.go:67,87`;
`internal/reports/deps.go:32` (monthly deps `profile_inputs, packages,
delivery, costs, program, invoices`); `render.PageLimit` (4, target 3).
Test: create twice returns one report; another org gets 404; every existing
PMP, RFP and RFT test still passes. Est. Go 40, SQL 10, YAML 15.

**Task 2. The loader.** Files: `internal/store/report_monthly.go` (new);
`internal/store/invoices.go` (split `InvoiceLedger` `:628` and
`invoiceContext` `:273` so their bodies take the caller's tx; the public
functions keep their signatures). `monthlyInputTx` reads, on the draft's
repeatable-read snapshot: project facts, the cost plan and its totals, the
ledger and its summary, Program inputs plus `computeState`, live packages
with one aggregate query (invitations, quotes received, quotes needing
review, award firm and date, RFP/RFT issued date), live delivery items of
every kind, documents filed since the period start by discipline, and
reading counts. Every query filters on `org_id`. Test (`report_monthly_test.go`):
on one fixture every figure equals `CostTotals`, `InvoiceLedger`,
`LoadProgram` and `ReadAward`; invoices tests unchanged. Est. Go 130.

**Task 3. The draft.** Files: `internal/reports/monthly.go` (new),
`monthly_test.go`. `AssembleMonthly(in MonthlyInput, prev []Section,
edits []Edit) ([]Section, error)`, one small function per section:
- Cost: one table (section | budget | approved contract | forecast final cost
  | claimed to date | remaining) with Total and Unallocated rows, amounts ex
  or inc GST as the plan states; "Incomplete: known subtotal $X, N lines
  unknown" where a total is not complete; invoices this period (count, ex
  GST total, how many need review); "Variations are not yet recorded in
  SiteWise" until plan 3 lands.
- Program: completion range and typical date (or "at least …" when
  durations are missing; week numbers for a relative program); the stages
  the critical path runs through; milestones and hold points that moved or
  fall due in the window (max 10); the planned-dates honesty line.
- Procurement: one row per live package: status words, firms invited,
  quotes, awarded to, contract sum from the cost plan's approved contract.
- Registers: per kind present (risk, issue, approval, decision, and plan 4's
  and plan 6's kinds once they exist) open and overdue counts plus up to 8
  items; "No RFIs are recorded in SiteWise" when a kind has none.
- Next period: Program rows starting or finishing in the window, register
  items due in it, packages awaiting quotes (max 10).
Test: empty project; cost plan with unknown lines; inc-GST plan (remaining
unknown, as on the grid); relative program; 40 packages capped honestly.
Est. Go 220.

**Task 4. Changes since report N.** File: `internal/reports/monthly.go`.
`movement(prev, now []Section) []Block` compares figures by block id: cost
totals and sections (Money via `costs.Sub`), completion dates (calendar
days), milestones by activity id (moved, added, removed), package status
(e.g. "Invited → awarded to Northline, $1,240,000"), register items opened
and closed, documents filed. A block whose basis method differs from the
previous one says "not compared". The first report prints "This is the
first issued report; changes are measured from the next one." Test: the
two-issue fixture lists every movement and removal; a renamed activity is
matched by id. Est. Go 80.

**Task 5. Needs attention (built first, as its own change).** Plans 1, 3,
4 and 6 each add rules to it; the left nav's marks and plan 0's named
filters read it. Files: `internal/reports/attention.go` (new),
`internal/httpapi/chat_more.go` (`readAttention` calls it). Move chat's rules
into a pure `Attention(in AttentionInput) []AttentionItem`: contract over
budget by section, lines with no budget, invited with no quote, quotes with
readings to check, documents not filed, Program rows with no duration. Add:
forecast over budget (both totals complete), invoices needing review or
unallocated, completion later than the last report, approvals submitted and
undetermined past their target date, open RFIs past due and conditions due
in the window (when those kinds exist). Max 10 with "and N more". Test:
chat's existing tests pass unchanged; for one fixture the report's list
equals chat's. Est. Go 40 new, 70 moved.

**Task 6. Comments.** File: `internal/reports/monthly.go`. Each printed
section ends with `comment:<section>`, and At a glance opens with
`comment:summary`. Its generated text is Go's sentences for that section;
its basis holds the section's figures and the period start, so a carried
or stale comment is flagged by the existing conflict rule and blocks issue
until kept (one save), rewritten or restored. No new route or table. Test:
a comment survives Update; a new period's draft flags it; one save clears
it; issue refuses while flagged. Est. Go 25.

**Task 7. The screen.** Files: `web/src/Delivery.tsx` (toolbar item
"Project report": find the `monthly` report in the project's reports, else
`reportApi.create(project, "", "monthly")`, then show `Reports` in the
register's panel, as `web/src/Procurement.tsx:310` does with its own view
state); `web/src/reportApi.ts:33` (kind other than rfp/rft sends no
package); `web/src/Reports.tsx:240` (hide the budget checkbox for
`monthly` as for `pmp`); `web/src/Project.tsx:385` (re-tick on invoice
and program events too). The reader registers with plan 0's `chatView.ts`
so chat can open it. Update, Edit wording, the issue form, the issued list
and the three-dot menu are reused unchanged. Test: browser spec
`web/tests/project-report.spec.ts`: open from the register, Update,
comment, issue, download. Est. TSX 50.

**Task 8. Bench.** Files: `cmd/intake-bench/reports.go` (a `monthlyReport`
case on a project seeded with `programBenchCreate` and `seedLedger(…, 1000)`,
30 packages, 200 delivery items and a previous issue; times create, assemble,
read, comment, issue, download); `bench/budgets.json` and
`internal/latency/gate_test.go` (`report_assemble_monthly`, 300,000 /
1,000,000 µs). Run it before Task 3 is finished so a slow read shows early.
If a gate fails, stop and tell the owner what would have to give; do not
raise a budget. Est. Go 60, JSON 1.

**Task 9. Send and guest view (after plans 1 and 2 land).** Files:
`web/src/Reports.tsx` (three-dot item "Send issue N…" on an issued version
opens plan 1's send dialog with that version's frozen PDF attached and the
client contacts filled). Plan 2's guest link shows every issued version of the
report, newest first (plan 2 decision 4; index agreement 5), never the draft, comments in progress or the screen's change list.
No route here. Est. TSX 15.

**Task 10. Operations, chat reads and the key.** Files:
`internal/httpapi/ops_more.go`, `ops_forward.go` (a report operation
resolves the report's project before the project check, which is why these
routes were left pending), `ops_routes.go` (`POST /reports/{r}/draft` and
the two edit routes leave `pendingOperation`), `chat_more.go`
(`read(report)`), `internal/store/change_undo.go` (a `captured` entry for
`report_edits`, area `reports`), `data/eval/chat/gen_cases.py`. Test: a
comment set from chat is undone; a comment carrying a figure the model
typed is refused; `TestOnlyChatReachesTheModel` still passes, because no
report route reaches the model. Est. Go 60.

## From chat

Plan 0's rule table governs (`2026-11-18-chat-agency.md`). No screen path
here calls the model.

- **Reads.** `read(report)`: the draft's blocks, each figure behind its
  tag, and the movements since the last issue.
- **At once, with Undo.** `reports.create` (exists). `reports.update`
  (assemble the draft: it is derived from the screens' records, so there is
  nothing to put back). `reports.comment` (write or clear a section's
  comment). A drafted comment follows the pencil guard
  (`docs/design/2026-11-13-pencil.md` 5.1, `internal/agent/guard.go`): the
  model writes figures as tags, Go prints them, and a comment with a
  figure, date or rule number the model typed itself is refused.
- **Opened filled in.** The reader, for the PM to press Issue; then Send
  (Task 9) with the issued PDF. Chat never issues.
- **Key cases.** "What moved since the last report"; "prepare this month's
  report" (create, Update, open); "draft the cost comment" (every figure is
  Go's); "clear the program comment"; "issue it" (opens the reader, issues
  nothing); "send report 4 to the client" (opens Send, after plan 1).

## Edge cases

| Case | What happens |
|---|---|
| First report | Period "to date"; Changes says it is the first; nothing compared |
| Empty project, or a surface never used | One honest line per section ("No Program yet") |
| Cost line with no budget or forecast | Total shows the known subtotal and "N lines unknown"; status words say "not yet known" |
| Plan in inc-GST | Remaining and percent stay unknown, as the ledger and grid do |
| Program with no start date | Completion in working weeks; no calendar dates invented |
| Durations missing | Completion is "at least …", with the count of unknown durations |
| Award undone between reports | Package shows its current status; Changes lists "award withdrawn" |
| Register item deleted or closed | Listed under Changes as closed or removed |
| Activity renamed or moved | Matched by id; the current label is printed |
| Previous issue made under an older template | Blocks whose method changed say "not compared" |
| Documents still being read | Title block says how many; issue needs the existing "issue the saved state" reason |
| Figures change after the PM comments | The comment is flagged; issue waits for keep, rewrite or restore |
| Issue date later than the Update | Title block prints "Figures as at" the Update date; the stale check catches source changes |
| Draft over four pages | Issue refused with the page count (existing overflow rule); lists are already capped |
| Delivery milestone records and Program milestones both exist | Program section shows Program's computed milestones; recorded ones appear under Registers with their stated dates |
| Plan 4 or 6 kinds not built | "No RFIs are recorded in SiteWise"; nothing errors on an unknown kind |
| Wrong org, archived or deleted project | 404 through the existing report session and org filters |

## Risks

- **Assemble budget.** Program compute, the ledger and citations on a large
  project share 1 s at p90. The bench runs early (Task 8); list caps keep the
  block count down.
- **One figure, two sources.** The grid's Remaining is contract less
  invoiced (ledger), while `costs.Derived.Remaining` is forecast less
  claimed. The report must use the ledger, as the grid does; the parity
  test in Task 2 holds it. If plan 3 changes where claimed comes from, the
  report follows the Cost Plan's function, not its own.
- **Issue render time.** A four-page PDF renders inside `report_issue`
  (100 / 250 ms). Measured in Task 8; if it fails, the owner decides.
- **Monthly review of carried comments** may feel like a chore (decision 16).
- **Template is `status: draft`** until the owner reviews the wording.
- **Shared migration.** Plan 3 adds a report kind (`payment_schedule`); both migrations
  rewrite `reports_kind_check`; the second to land must list both kinds.
