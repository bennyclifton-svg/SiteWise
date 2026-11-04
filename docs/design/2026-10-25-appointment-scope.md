# One appointment schedule

25 October 2026. Lane A. Owner-approved direction: two scope schedules,
one total, row arrows, collapsed details; selected fees must carry through
to award and the cost plan.

The consultant review shows Included in appointment, Total excluding GST,
and Not included in appointment. A source fee appears once. A reconciled
source breakdown replaces the repeated professional-fee total; its amount
is never also added as a package total. Distinct alternative totals remain
available below the line. Scope, Exclusions and Conditions start collapsed;
rates remain at the bottom.

The RFP schedule provides stage names, including the package's default
schedule when no RFP exists. Existing coverage readings suggest mappings;
the UI labels suggestions and allows the user to correct covered stages.
One fee may cover several stages without inventing a monetary split. Its
source wording stays in Details. Mapped stages are not repeated elsewhere.
Source positions across firms never imply equivalent scope.

Go projects each firm's source fees and schedule into appointment rows.
The same projection computes the displayed total and resolves the award.
Row inclusion and mapping are user decisions in the existing
submission_decisions table, scoped to organisation and submission. They
do not change the RFP, another firm's inclusion, or source prices.
No schema migration, service, dependency or Jev call is added.

The row update locks the package and submission, checks the submission
version, validates live stage keys, saves the user decision and returns the
refreshed price table. Award resolves only included priced rows. Every
such row must occur exactly once, separately or in a combined line. An
excluded row, duplicate, stale version, mismatched total or legacy request
that bypasses an explicit selection is refused. An included but unpriced
row blocks award; scope explicitly included in another fee creates no
separate monetary cost line. Source bundle fees move as a whole.

Award uses the existing cost transaction and records quoted-fee and stage
references on the child lines. Budgets and frozen baselines are unchanged.
Undo restores the prior cost plan while retaining the chosen appointment
scope. Report and export total summaries use the selected appointment.

Budgets remain price_table p50/p90 150/300 ms, submission_write 100/250 ms,
award_write 200/500 ms; filing remains 1000/2000 ms. The focused regression
tests cover source-to-schedule mapping, bundle deduplication, unknown tax
basis, persisted selection, stale edits and awards, wrong organisation,
duplicate/excluded award lines, combined cost values and undo. The existing
award, frozen-baseline, comparison-alignment and HTTP isolation tests remain
applicable. Browser tests cover added fees, stage/total modes, row movement,
mapping, reload, combined award, the cost plan, undo and responsive layout.

Local measurements (20 samples): appointment read 2.2/3.0 ms, selection write
including refreshed table 5.3/5.7 ms, award 10.9/12.7 ms. Existing five-firm
HTTP measurements: read 8.4/8.7 ms, write 16.1/16.9 ms (including the new
row-selection endpoint). These are development
machine results, not release-host measurements.

Verification passed: targeted Go award, scope, comparison, quote-reading,
HTTP isolation and cost tests; four procurement browser flows; TypeScript
and the production web build. No live project was awarded during testing.

## Comparison slots — 28 October 2026

Lane B: keep a single consultant's prices near the scope labels, with a bounded
scope column and narrow vacant columns up to three consultants. Vacant headers
open the existing Add firm control; they contain no prices or totals. Two firms
leave one slot, and three or more leave none. Small screens scroll the table
within its panel. No API, computation, dependency or budget changes.

Build and the existing single/multiple-firm fee-editing and award-preview browser
flow passed; desktop and mobile screenshots inspected. The separate scope-arrow
test stopped at a mapping-label assertion (expected the stage label, received
the source fee label); that behaviour was not changed here. The price_table
budget remains 150/300 ms; latency was not remeasured for this presentation-only
change.
