# Hale BCA comparison review and implementation

19 October 2026. Lane A. See the task boundary in
`docs/plans/2026-10-19-comparison-alignment.md`.

## Review

Read all four supplied PDFs and visually checked the fee schedules: BM+G
physical page 5, Certatude (filename Certitude) page 7, Group DLA page 15,
and MBC page 7. Private extracted text, page images and the original saved
comparison are under ignored `tmp/bca-review/`.

The material difference is scope, not just price. BM+G offers advisory work
through design and tender. The certification offers cover different combinations
of assessment, approvals, inspections and occupation. DLA separates these;
Certatude combines construction and occupation; MBC provides one certification
bundle plus disbursements. A total cannot establish equivalent scope.

The saved Hale table also contains concrete reading errors: a boilerplate
sentence as a firm name, missing or incorrectly allocated fees, and insurance
limits and penalties in offered extras. A generated comparison must remain a
reviewable draft. Lowering confidence thresholds to claim 50% or 80% accuracy
would conceal this distinction. Measure capture coverage, mapping accuracy,
and the time needed to finish review separately on a labelled corpus.

Recommended comparison structure:

- Separate advisory and statutory certification services in the descriptions.
  Select the intended common base scope explicitly; keep options below it.
- Keep original consultant wording and page evidence alongside common rows.
  Combine a firm's split fees when justified; count a bundled fee once and
  mark its other covered rows Included with a bundle reference.
- Record inspection/meeting allowances, approval counts and scope exclusions
  beside prices. Keep per-event and hourly rates outside lump-sum totals.
- Retain an explicit human review step for ambiguous scope and firm identity.
  Missing fees and excluded required scope must not produce a lowest complete
  price. Do not spread an undivided fee across stages by assumption.

## Delivered

Arrange comparison adds/renames common rows and moves fee rows above/below
the base total. Comparison overrides are separate from the RFP. Custom rows
and overrides survive subsequent RFP synchronisation; issued wording is intact.
Unit-rate rows support hours, meetings, inspections, approvals, reports,
reviews and revisions, with units also carried to Excel.

For each firm, select source prices and a destination to move or combine.
Go sums integer cents, rejects duplicate sources, refuses to overwrite a priced
destination unless selected, and prevents known rates becoming lump-sum fees.
Source rows become Included, or Excluded when moved out of required base scope.
Set selected aside handles non-fees and items outside the comparison. The last
alignment can be undone, including after reload, until another submission edit.
Snapshots and source passages preserve the prior values; later background price
reads leave an aligned submission's prices intact.

Each cell has an editable scope/allowance note. Notes appear in the comparison,
base-fee report rows and Excel. All unpriced fee/rate rows remain editable.
Explicit policy values, imprisonment penalties and no-invoice liability limits
are filtered from future harvested fee candidates. Existing stored proposal
readings are not silently rerun or accepted by this change.

Award and comparison writes share a package lock. No edits after award, no
cross-organisation or cross-project access, stale submission versions rejected.
Rates cannot be awarded as lump sums. Excluded base scope and an empty base
schedule no longer produce a complete zero fee. No dependency added.

Program integration is deliberately through the existing RFP boundary: its
planned Update/Keep mechanism can refresh source stages while preserving
comparison overrides. No second stage catalogue or premature Program API was
introduced. The Program module itself remains separate ongoing work.

## Verification

- Proposal, RFP, store and HTTP suites passed with the previously documented
  unrelated `TestProfilePlacesInputsInBuildOrder` excluded.
- New regressions cover non-fee amounts, excluded/no-base totals, alignment,
  duplicate counting, stale versions, undo, unit rates, notes, set-aside,
  organisation isolation, award locks and RFP override persistence.
- The new comparison browser test and both existing procurement award tests
  passed. Desktop and narrow-screen layouts inspected.
- TypeScript/Vite build and diff whitespace checks passed.
- Local app rebuilt/restarted and Hale's four-column price table verified
  through its HTTP API and browser. No supplier award or live fee allocation
  was made on the user's behalf.

20-sample development timings (not release/provider benchmarks):

| Path | p50 ms | p90 ms | Budget p50/p90 ms |
|---|---:|---:|---:|
| Actual Hale price-table HTTP read | 8.56 | 29.82 | 150 / 300 |
| Store row write | 1.08 | 1.66 | 100 / 250 |
| Store move/combine | 3.00 | 3.36 | 100 / 250 |

Store timing tests fail when these budgets are exceeded. No additional Jev
request was introduced. Candidate filtering follows the existing code-parses,
Jev-picks pattern in TypeSafe's [pre-parsed extraction guide](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and retains the existing [single fan-out](https://docs.typesafe.ai/patterns/fan-out).

## Limits

This supplies a practical correction workflow; it does not certify that all
four historical readings are complete. Firm-name extraction, comprehensive
conditional-charge capture, and automatic reconciliation of bundled coverage
still need corpus-based evaluation. Users can correct the saved table now.
Undo is one alignment deep and refuses to overwrite a subsequent cell edit.
RFP/program changes do not silently reapprove scope equivalence.

## Direct table interaction refinement

Replaced the Arrange comparison disclosure with row-end up/down arrows, clickable
service names, Add row, and fee-level move/combine buttons. Selecting a fee exposes
destination buttons in that firm's column; confirmation includes an existing
priced destination automatically. Undo remains in the firm header. Set aside is
available in the contextual action bar. Rates have no base/option arrow and cannot
be combined with another priced rate. Existing Go validation, integer arithmetic,
version checks and award locks remain authoritative.

The item column is 22% instead of 34%; firm columns share remaining space. Review
triangles replace repeated Check/Jev chips. Source wording and scope notes are
available by opening the fee; full notes also remain in the hover description.
The separate arrangement form and its CSS were removed. No new dependency.

Validation: web build passed; direct-controls browser test passed, including
placement persistence, rename, combine, undo, add unit rate and set aside. Both
existing procurement/award browser tests passed after updating the header assertion
to count firms separately from the new action column. Desktop Hale and mobile test
screenshots inspected. Store regression and speed gates passed: read p50/p90
1.05/1.20 ms (150/300 budget), row write 1.07/1.13 ms and alignment 2.30/2.82 ms
(100/250 budget). These are local database operation samples, not end-to-end click
latency. Rebuilt frontend and restarted local server; actual Hale prices unchanged.
