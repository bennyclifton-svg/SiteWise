# Cost-plan controls and shared row gestures

24 October 2026.

Outcome: a quieter cost plan with direct cell edits, a slender navigation rail,
and the same selection and row-drag interaction as Program. This reduces the
clicks needed to curate and arrange a plan. Layout is Lane B; persisted ordering
and inline claims are Lane A. Cost reads and writes retain p50/p90 100/250 ms;
filing retains 1,000/2,000 ms. No filing, pricing or grouping changes are in scope.

- Desktop navigation is 200–236 px, down from 220–268 px, including the shell's
  inline layout and document-rail width calculations.
- Item names select on first click and edit on a second click, double-click,
  Enter or F2. The separate Edit link is removed. Budget and contract amounts,
  existing ranges, and claimed amounts with their required dates edit in cells.
- A three-dot control immediately before the bin opens line details,
  subdivision and include/exclude actions. Redundant drawer editors, reviewed
  benchmarks, geography/quality inputs and package-scope reference controls
  are removed. Existing stored scope references and benchmarks are retained.
  Opening a row no longer fetches every package's scope.
- All bin icons use the shared, larger plain outline without interior strokes.
- Cost Plan imports the existing table selection module used by Program and
  Register: click, Ctrl/Cmd toggle, Shift range, Ctrl/Cmd+Shift additive range,
  Ctrl/Cmd+A and Escape. Program and Cost Plan both use the extracted RowHandle
  component, its CSS, and its pointer threshold/listener/Escape lifecycle.
- Dragging selected sibling lines saves one atomic order update. Parent groups
  travel with their children; lines stay within their section and parent.
  Sorting temporarily disables dragging. Ordering is revision metadata, leaving
  codes, money, assignments and scope unchanged; baseline cloning preserves it.
  Stale writes, frozen revisions, wrong organisations, missing or repeated IDs,
  incomplete sibling sets and cross-section moves are rejected.
- Sticky identities have opaque surfaces so horizontally scrolling figures do
  not show through. Native popovers stay above the scroll container and close
  on outside click, Escape, scrolling or resize.

Verification: frontend build and 234 unit tests passed. Cost domain, store and
API tests and the route-access sweep passed. Local 20-sample reorder endpoint
p50/p90 was 5.0/6.0 ms against 100/250 ms. The same suite measured cost reads at
1.36/1.82 ms and writes at 3.58/5.10 ms. The existing endpoint benchmark now also
samples order writes under the costs_write budget.

The cost-selection and ledger browser scenarios and all 17 Program editing
scenarios passed using installed Chrome. Program's 2,000-activity drag check
measured p95 6.6 ms against 16 ms. Desktop and mobile screenshots were inspected;
the final pass also checks the shell-width and sticky-background corrections.
No new dependency, service, AI call or migration. Not deployed.
