# Project report: requirements framework

Date: 12 November 2026. Status: a skeleton for owner review. Nothing is
built and nothing here is decided. Each section says what has to be
settled, what the tree already holds, and what is missing. A design note
and a build plan follow once the decisions in section 12 are answered.

Read first: `PRODUCT.md`; `docs/plans/2026-10-04-next-wave-architecture-schema.md`
("Reports and eventual profile tabs": the PMP's content, and stage 6 "Live
reporting", which is this module and was never built);
`docs/evidence/2026-10-05-wp35-delivery.md` (risks, approvals, milestones);
`docs/design/2026-11-10-run-time-feedback.md` (changed marks, the receipt,
"what this changes elsewhere").

## 1. Outcome

A project manager opens Report and presses Update. In about a second they
have a short, current account of the project: what it is, where each part
stands, what has moved since the last report, and what needs attention.
They add a comment where one is wanted, issue it, and send the PDF. A week
later, or two days, or a month, they do the same, and the report says what
changed since the one before.

Why it earns its place: time-to-decision (the client and the PM decide from
a few pages instead of five screens) and answer trust (every figure is the
workbench's own figure, with its source).

Lane A. The report restates cost figures, awards, approvals and issued
records, so it must never disagree with them. It reads; it owns no project
fact.

## 2. Who uses it, and when

| Use | Who | Rhythm | What they need |
|---|---|---|---|
| The issued report | The PM writes it; a client, owner or financier reads the PDF | Weekly in delivery, fortnightly or monthly in design | A record that stands on its own and can be held to |
| The catch-up | The PM, privately | Any morning | "What changed since I last looked, and what is waiting on me" |

Both are one mechanism. The draft on screen is the catch-up. Issuing it
makes the record. Readers never open the app (there is no sharing yet).

## 3. Rules it must hold

These follow from the binding rules; they are the frame for every later
choice.

1. **No Jev on the report path.** Update, read, edit, issue and export are
   code over saved records. Whatever Jev read, it read at filing, profile
   update, invoice or proposal time.
2. **No machine-written narrative.** Jev only picks and there is no other
   AI, so every sentence is one of three things: fixed wording filled from
   records by code, a table, or the PM's own words.
3. **One figure, one owner.** Money comes from the Cost Plan's Go rollups
   and dates from Program's Go compute. The report never keeps its own
   arithmetic.
4. **Unknown stays unknown, and nothing is inferred.** A filed consent does
   not make an approval "approved"; a paid invoice does not make an
   activity "complete". Status is what a person recorded.
5. **Issue freezes.** An issued report keeps its wording, figures and
   sources whatever happens later.
6. **The PM's word is final.** Commentary survives every Update.
7. **Every statement carries its source.**
8. **Short.** A page limit, measured before issue, as the RFP has.

## 4. What the report says

Proposed order: the project's own order (what it is, what is permitted, who
is appointed, when, how much, to what standard, what is happening on site),
with the summary and the changes in front. Lists of building things follow
building order, as the Profile does.

| # | Section | Answers | Drawn from (already saved) | Missing today |
|---|---|---|---|---|
| 0 | Title block | Which project, which report, what period | Project facts (`proj.*`), RFP page-one title block, issue number and reporting date on `report_versions` | A rule for "current stage" |
| 1 | At a glance | Where each area stands in one line, and the PM's summary | The sections below; the PM's wording | The status words (decision 7) |
| 2 | Needs attention | What is late, over, blocked or waiting on someone | Computed from explicit states only: open records past their date, an approval submitted and not determined, forecast over budget, unallocated or failed-check invoices, a closed RFP with no quote, unreviewed readings, failed reads | The rule for "high" (decision 6) |
| 3 | Changes since the last report | What moved, in date order | The block diff against the last issue; timestamps on the records | A "since" filter on some reads |
| 4 | Scope | What the building is and what is in scope | Profile summary sentence and key fields, work items, Accommodation schedule, documents filed and superseded | — |
| 5 | Planning and authorities | What controls apply, what is lodged, what is determined | Planning determinants on the Profile (LGA, zone, heritage, flood, BASIX, pathway); delivery records of kind approval (authority, reference, submitted, determined, status); authority documents in the register | No conditions-of-consent register; approvals have no screen (section 8) |
| 6 | Procurement | For each consultant and trade: prepared, issued, invited, quoted, awarded, to whom, for how much | Packages, issued RFP/RFT versions, invited firms, submissions, awards, comparison reports | — |
| 7 | Program (time) | Current stage, completion, critical path, what is due next, what moved | Program's computed dates, float and critical path; program changes; delivery milestones | No saved baseline; no progress on activities |
| 8 | Cost | Budget, contract, claimed, remaining, forecast by section; movement; invoices in the period | Cost Plan rollups, frozen baseline, invoice ledger and its checks | Variations (a placeholder today) |
| 9 | Quality | Specification coverage and issues; hold points; readings awaiting review | Specification coverage, issues and history; hold-point evidence readings; proposals | Inspections, defects, non-conformances |
| 10 | Construction status | What is happening on site | Delivery activities and milestones; hold points released; claimed against contract per trade | Almost everything live: progress, site diary, RFIs, variations, extensions of time |
| 11 | Risks, issues, decisions | What could go wrong, what has, what is undecided | Delivery records (risk, issue, decision); unforeseen-condition proposals not dismissed; procurement gaps | No ranking; no screen (section 8) |
| 12 | Next period | What is due before the next report, and from whom | Open actions and decisions; Program look-ahead | — |
| 13 | Sources and reading status | Where each statement came from; what is unread | Citations (evidence, user, calculation, assumption); pending, failed and unread counts | — |

A section with nothing recorded prints one honest line ("No approvals are
recorded"), never a blank and never a guess.

## 5. How a reporting cycle works

1. **Update** assembles the draft from the project as saved now. One
   request, one consistent read.
2. **The period is "since the last issued report"**, not a calendar setting.
   Weekly, every other day and monthly are then the same feature.
3. **Changes** come from comparing this draft with the last issue, block by
   block, by stable identity. A removed item is listed as removed.
4. **The PM comments** in place. Their wording is protected; if its source
   later changes, the report says so and keeps their words.
5. **Issue** freezes the snapshot, numbers it and renders the PDF.
6. The first issued report is also the reference the next one is measured
   against. Program has no saved baseline, yet "completion moved 12 days
   since report 3" falls out of two frozen snapshots.

## 6. Approach

Three ways to build it:

- **A. A new kind on the existing report engine (recommended).** One
  project-level report with numbered issues. The unissued draft doubles as
  the live view. Freeze, diff, commentary, citations, PDF and Word are
  already there.
- **B. A live dashboard page** reading each surface's API from the browser.
  Quick, but there is no record, no "since last report", and it costs
  several requests per open.
- **C. Revive the PMP as it stands.** It is already project-level, but its
  content is a plan, not a status, and it has no period.

A, taking the PMP's project-level shape from C.

## 7. What is already in the tree

| Need | Already built | Where |
|---|---|---|
| Sections and blocks assembled from saved records, no Jev | The report engine | `internal/reports/assemble.go`, `draft.go` |
| A report with no package | The PMP kind | `internal/reports/kinds.go` |
| Section sets per kind | Templates | `knowledge/reports/templates.yaml` |
| PM wording kept across Update, conflict and restore | Protected wording | `internal/reports/edits.go`, `reset.go`, `web/src/InlineText.tsx` |
| Freeze, number, integrity hash | Issue snapshot | `internal/reports/snapshot.go`, `report_versions` |
| What changed since the last issue | `ChangesSinceIssue` | `internal/reports/kinds.go` |
| A source on every statement | Citations | `internal/reports/citations.go` |
| "Sources changed" notice, issue with a reason | Stale list, source revisions | `project_revisions`, the issue form |
| Review lists that are never issued | Internal sections | `Section.Internal` |
| Hide internal money from an outside reader | Budget disclosure | `internal/reports/disclosure.go` |
| PDF, Word, print, page fit | Export and render | `internal/export`, `internal/reports/render` |
| The screen: reader, section bands, Update, three-dot menu, title block | The RFP reader | `web/src/Reports.tsx`, `RfpEditor.tsx`, `PageActions.tsx` |
| Refresh when another surface changes | Events by kind | `internal/events`, `web/src/Project.tsx` |
| Risks, issues, decisions, approvals, milestones, actions, with status history | Delivery records and API | `internal/delivery`, `project_delivery_items` |
| An editor for those records | Built, not mounted anywhere | `web/src/Delivery.tsx` |
| Counts of work waiting elsewhere | The profile's next read | `GET /projects/{id}/profile/next` |
| Changed marks | The revision triangle | run-time feedback |
| Budgets | `report_read`, `report_assemble`, `report_issue`, `report_export` | `bench/budgets.json` |

## 8. What is new, and what is missing

New, and small:

1. A report kind and its template (`status: draft` until the owner reviews).
2. One assemble function for that kind. The shared one expects the seven
   RFP section ids, so this kind needs its own, over the same block and
   section types. Each block needs a stable id (`package:<id>`,
   `approval:<id>`, `cost:section:<name>`) or the diff means nothing.
3. A snapshot loader that calls each surface's existing read once.
4. The dated change list.
5. The nav item switched on, and a route.
6. A bench case under the existing `report_assemble` budget.

Missing data, which the report will expose rather than hide:

- **Risks, approvals, actions and milestones have no screen.** The records,
  API and editor exist; nothing mounts the editor. Without a way in, three
  sections stay empty. The report is the natural place to enter them, as an
  RFP writes project facts through to the Profile.
- **Program holds a plan, not progress.** No actual dates, no percent
  complete, no saved baseline.
- **Construction status has almost no source.** No site diary, RFIs,
  variations, extensions of time, defects or inspections.
- **No conditions-of-consent register.**
- **No risk ranking.** Likelihood and consequence are free text by design.

Each of these is its own decision. None blocks a useful first report.

## 9. Speed

No Jev call. Use the budgets already recorded: read p50 ≤ 100 ms and p90 ≤
250 ms; Update (assemble) p50 ≤ 300 ms and p90 ≤ 1 s; issue and export p50
≤ 100 ms and p90 ≤ 250 ms. Typing commentary is a report write (100 / 250).
The assemble reads six surfaces, so it is the path to watch: measure it on
a project with 1,000 program activities and 1,000 invoices.

## 10. Out of scope

AI-written prose. Scheduled or emailed reports. Sharing or a client login.
A multi-project (portfolio) report. A report builder or per-client
templates. New registers (site diary, RFIs, variations, extensions of
time, defects). A risk-scoring matrix. Charts in the first version.

## 11. Slices

| # | Slice | Done when |
|---|---|---|
| 1 | The draft: title block, Scope, Procurement, Cost, Program from existing reads; Update; nav on | Every figure equals its owning surface's figure in a test; assemble inside budget |
| 2 | Issue, numbered versions, PDF and Word; changes since the last report | A two-issue fixture lists every change and removal; the first report says it is the first |
| 3 | Needs attention and Next period | Each line traces to a recorded state; nothing is listed that cannot be counted |
| 4 | Risks, approvals, actions and milestones entered from the report; Planning and authorities; Risks | A record added in the report appears in the next Update and in the issue |
| 5 | Quality and Construction status, as far as records go | The owner decides which new inputs are worth adding |

## 12. Owner decisions

1. **Name.** The nav says "Report". Is the document "Project report No. 4",
   "Status report", or something else?
2. **First reader.** Build first for the issued client report, or for the
   PM's own catch-up? This sets tone, length and what money is shown.
3. **Length.** Two pages, four, or a one-page summary with detail behind?
4. **Order.** Project order as in section 4? Changes oldest first or newest
   first?
5. **Period.** "Since the last issued report" (proposed), or dates the PM
   picks?
6. **"High risk."** The PM marks it, a fixed likelihood and consequence
   scale, or computed signals only?
7. **Status words at a glance.** Computed only where the rule is exact
   (forecast against budget), set by the PM, or none in the first version?
8. **Where risks, approvals and actions are entered.** In the report, in
   place (proposed), or bring Delivery back to the nav?
9. **Planning authorities.** The approvals register only, or a conditions
   register as well (new data)?
10. **Construction progress.** None in the first version (proposed), or
    actual dates and percent complete on Program activities (a Lane A change
    to Program)?
11. **Money an outside reader sees.** Budget, contract and claimed in full?
    Each firm's contract sum?
12. **The PMP.** Does this report replace it, or do both stay?

## 13. Edge cases to carry into the design

The first report (nothing to compare). An empty project. A surface never
used (no Program yet). An Update while documents are still being read. A
record deleted, or an award undone, between two reports. Commentary whose
source has changed. Two tabs editing one draft. A draft too long to issue.
An archived project. Every row scoped to `org_id`.
