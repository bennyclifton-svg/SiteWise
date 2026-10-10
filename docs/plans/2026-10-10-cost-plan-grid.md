# Cost Plan Grid Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Costs list-and-reader page with a full-width cost plan grid
(one row per line, columns across the top, subtotals by group), retire the
`estimate` metric, and keep every other cost mechanic as it is.

**Architecture:** The ledger, endpoints and version pins stay. The backend gains
per-line figures (forecast, budget variance, remaining) on the existing totals
response so the browser never does money arithmetic. `estimate` is no longer
writable; draft estimates fold into budget in one migration; frozen baselines are
untouched. The React page becomes a grid plus a right-hand drawer that holds the
existing line editor, value editor, subdivision, benchmarks and scope links.

**Tech Stack:** Go (internal/costs, internal/store, internal/httpapi), PostgreSQL
migration, React + TypeScript (web/src), Playwright (web/tests).

---

## 0. Task statement (AGENTS.md)

- **User outcome:** open Costs and see every line's budget, contract, forecast,
  variance, claimed and remaining at once, as in a QS cost report; type a budget
  straight into the grid.
- **Why:** time-to-decision. Today one line's figures are visible at a time and
  totals are a paragraph of text.
- **Lanes:** Part 1 (retire estimate) and Part 2 (per-line figures) are **lane A**:
  they change numbers a user trusts. Part 3 (grid) is **lane B**. Each part is its
  own commit series; Part 3 must not land before Part 2.
- **Budgets:** `costs_read` and `costs_write` stay p50 ≤ 100 ms, p90 ≤ 250 ms
  (bench/budgets.json). The page does not make more requests than it does today;
  opening Costs makes fewer (package scopes load only when the drawer opens).
- **Out of scope:** invoices, variations, monthly claims, the Process invoices
  button, export/download, the Invoices/Variations tabs, sorting by system/part/
  package in the grid (the `?by=` API stays for reports).
- **Done:** Go tests, store tests, migration test, `npm run build`, the rewritten
  `costs.spec.ts` and the costs speed bench all pass; Petersham shows the grid in
  the browser.

### Owner decisions (2026-10-10)

1. **Estimate is retired.** Budget is the one planning figure. On fast projects
   the budget is often skipped and the first contract award sets the number;
   larger projects set a budget first. Either way an estimate column adds nothing.
2. **Variations and This Month columns are shown as placeholders** (disabled, "—")
   until those features exist.
3. **Only Budget is edited in the grid.** Approved Contract and Claimed to Date
   remain editable in the drawer (nothing else writes them yet; removing that
   would remove a mechanic). Ranges, unknown, as-of and rationale are drawer-only.

### Columns

| Column | Source | Editable in grid |
|---|---|---|
| Code | `code` | no |
| Category | fee/works → package title (or "Unassigned"); project-wide → category | no |
| Item | `label` (children indented under a group line) | no |
| Budget | value `budget` | **yes** (posting, not excluded, not frozen, not a range) |
| Approved Contract | value `commitment` | no (drawer) |
| Forecast Variations | placeholder | no |
| Approved Variations | placeholder | no |
| Forecast Final Cost | `items[id].forecast` (commitment, else budget) | no |
| Budget Variance | `items[id].budget_variance` = budget − forecast | no |
| Claimed to Date | value `claimed_to_date` | no (drawer) |
| This Month | placeholder | no |
| Remaining | `items[id].remaining` = forecast − claimed | no |

Display rules (trust, lane A rule carried into lane B):
- Unknown is "—", never 0.00. A range shows `low–high`.
- A subtotal with unknown lines shows its known subtotal plus "+ n unknown".
- Budget Variance is positive when under budget (QS convention, matches the
  reference screenshot). The existing `totals.variance` (forecast − budget) used by
  CostSummary and PMP reports is not changed.
- Remaining is unknown until Claimed to Date is entered; nothing is assumed zero.

---

## Part 1 — Retire `estimate` (lane A)

### Task 1: Estimate is no longer a writable metric

**Files:**
- Modify: `internal/costs/plan.go:9`
- Modify: `internal/costs/totals.go:27-35` (comment only; order stays)
- Test: `internal/costs/costs_test.go`

**Step 1: Failing test** — add to `costs_test.go`:

```go
func TestEstimateIsNotWritable(t *testing.T) {
	v := val("10.00")
	if ValidateValue("estimate", &v) == nil {
		t.Fatal("estimate accepted")
	}
}
```

**Step 2:** `go test ./internal/costs/ -run TestEstimateIsNotWritable` → FAIL.

**Step 3:** In `plan.go` set
`var Metrics = []string{"budget", "commitment", "claimed_to_date"}`.
In `totals.go` keep the forecast order `commitment, estimate, budget` and add the
comment: `// estimate survives only on frozen baselines written before it was
retired; reading it keeps their forecast unchanged.`

**Step 4:** `go test ./internal/costs/` → PASS. `TestForecastNullRangeAndExclusion`
still passes (it builds an item directly; forecast is commitment).

**Step 5:** Commit `Retire the estimate cost metric (lane A)`.

### Task 2: Calculate and benchmarks write budget

**Files:**
- Modify: `internal/store/cost_calculate.go:14` → `if metric != "budget" {`
- Modify: `internal/store/cost_benchmarks.go:145-152` → write `"budget"`, update
  `metric='budget'` in the provenance UPDATE; error text "enter a budget or leave
  unknown".
- Modify: `internal/store/report_costs.go:38` → drop `"estimate"` from the list.
- Test: `internal/store/costs_test.go:99-103,186-190,395-411,487`,
  `internal/store/report_cost_summary_test.go:22`

**Step 1:** Change those tests to use `"budget"` where they wrote `"estimate"`;
rename `TestCostMissingBenchmarkDoesNotInventEstimate` →
`...DoesNotInventBudget`. Add one assertion that
`PutCostValue(..., "estimate", ...)` returns `costs.ErrInvalid`.

**Step 2:** `SITEWISE_TEST_DATABASE_URL=... go test ./internal/store/ -run Cost`
→ FAIL on benchmark/calculate tests.

**Step 3:** Make the three code edits above.

**Step 4:** Re-run → PASS. Also `go test ./internal/store/ -run Report`.

**Step 5:** Commit `Benchmarks and quantity × rate write budget (lane A)`.

### Task 3: Fold draft estimates into budget

**Files:**
- Create: `internal/db/migrations/021o_cost_retire_estimate.sql`
- Test: `internal/db/migrate_cost_estimate_test.go` (pattern: `migrate_packages_test.go`)

Rules: drafts only (frozen versions are protected by trigger and must not change).
Where a draft line has a known estimate and its budget is missing or unknown, the
estimate becomes the budget with provenance recording the move. Then every draft
estimate row is deleted. The CHECK constraint keeps `'estimate'` so baselines
stay valid.

```sql
-- Estimate is retired: budget is the one planning figure. Drafts move a known
-- estimate into an empty budget; baselines are frozen and keep theirs.
INSERT INTO cost_values (org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,review_status,meaning,rationale,provenance)
SELECT e.org_id,e.project_id,e.plan_version_id,e.cost_item_id,'budget',e.value_state,e.amount,e.low,e.high,e.as_of,e.origin,e.review_status,e.meaning,e.rationale,
       e.provenance||jsonb_build_object('moved_from','estimate','migration','021o')
FROM cost_values e JOIN cost_plan_versions v ON v.org_id=e.org_id AND v.id=e.plan_version_id AND v.status='draft'
WHERE e.metric='estimate' AND e.value_state='known'
ON CONFLICT (org_id,plan_version_id,cost_item_id,metric) DO UPDATE
SET value_state=EXCLUDED.value_state,amount=EXCLUDED.amount,low=EXCLUDED.low,high=EXCLUDED.high,as_of=EXCLUDED.as_of,
    origin=EXCLUDED.origin,review_status=EXCLUDED.review_status,meaning=EXCLUDED.meaning,rationale=EXCLUDED.rationale,
    provenance=EXCLUDED.provenance,version=cost_values.version+1
WHERE cost_values.value_state='unknown';
DELETE FROM cost_values e USING cost_plan_versions v
WHERE v.org_id=e.org_id AND v.id=e.plan_version_id AND v.status='draft' AND e.metric='estimate';
```

**Step 1: Failing test** — in one transaction, seed two orgs; draft line A
(estimate 120 known, budget unknown), draft line B (estimate 120, budget 100),
draft line C (estimate unknown), and a baseline line with estimate 120. Run the
migration file. Assert: A budget = 120.00 with `moved_from`; B budget still 100.00;
no draft estimate rows remain; the baseline estimate row is unchanged; org B rows
untouched.

**Step 2:** Run with the dedicated test DB → FAIL (file missing).

**Step 3:** Add the migration.

**Step 4:** Test → PASS. Run `go test ./internal/db/` (fk sweep seeds an
`estimate` row — it still satisfies the CHECK).

**Step 5:** Before committing, on the dev DB count what will move:
`SELECT count(*) FILTER (WHERE value_state='known') FROM cost_values e JOIN cost_plan_versions v ON v.id=e.plan_version_id AND v.status='draft' WHERE metric='estimate';`
Record the number in the commit message. Where both budget and estimate were
known and commitment unknown, the draft forecast moves from estimate to budget;
list those lines too.

Commit `Fold draft estimates into budget (lane A)`.

---

## Part 2 — Per-line figures on the totals response (lane A)

### Task 4: Line and bucket figures in Go

**Files:**
- Modify: `internal/costs/totals.go`
- Test: `internal/costs/costs_test.go`

Add:

```go
// Figures are derived for the grid so money is never summed in the browser.
// A figure is nil when either input is unknown or only a range.
type Figures struct {
	Forecast       Value  `json:"forecast"`
	Basis          string `json:"basis"`
	BudgetVariance *Money `json:"budget_variance"` // budget − forecast; positive is under budget
	Remaining      *Money `json:"remaining"`       // forecast − claimed to date
}
```

On `Totals` add `Items map[string]Figures \`json:"items"\`` and
`Buckets map[string]Figures \`json:"buckets"\`` keyed `overall`, `works`, `fees`,
`project_wide`. `diff(a, b *Money) *Money` returns `Sub(*a,*b)` only when both
are non-nil. In `Summarize`, fill `Items[item.ID]` for every posting item
(excluded lines included, so their row still reads; they stay out of totals);
fill `Buckets` after the loop from each bucket's `budget`, `forecast`,
`claimed_to_date` `Amount`.

**Step 1: Failing tests:**
- budget 100, commitment 90, claimed 40 → forecast 90 basis commitment, variance
  10.00, remaining 50.00.
- budget 100 only → forecast 100, variance 0.00, remaining nil (claimed unknown).
- budget range 80–120 → variance nil.
- bucket `works` with one unknown line → variance nil.
- excluded line has `Items` entry but does not change `Buckets["overall"]`.

**Step 2–4:** red → implement → `go test ./internal/costs/` green.

**Step 5:** Commit `Per-line forecast, budget variance and remaining (lane A)`.

### Task 5: Speed and API check

**Files:** `internal/httpapi/costs_test.go`

**Step 1:** Add an assertion to the existing totals handler test that
`items[<id>].forecast` and `buckets.overall` are present and the org-B request
for the same project still 404s.

**Step 2:** Run `go test ./internal/httpapi/ -run Cost` and the speed gate for
`costs_read` (same command the committed-budgets gate uses). Record p50/p90 in
the commit message. Expect no change: the work is O(lines).

**Step 3:** Commit `Totals carry grid figures; costs_read within budget (lane A)`.

---

## Part 3 — The grid (lane B)

### Task 6: API types

**Files:** `web/src/costApi.ts`

Add `Figures` (`forecast: CostValue; basis: string; budget_variance: string|null;
remaining: string|null`); extend `CostTotals` with `works`, `fees`,
`project_wide: Record<string, CostTotal>`, `items: Record<string, Figures>`,
`buckets: Record<string, Figures>`. No new calls.

### Task 7: Grid component

**Files:**
- Create: `web/src/CostGrid.tsx` (grid + inline budget cell, ~150 lines)
- Create: `web/src/costs.css`
- Modify: `web/src/Costs.tsx` (page shell, toolbar, drawer; drop the list layout
  and the "Group totals by" select)

Layout, top to bottom:
1. **Toolbar:** Add cost line · Cost settings · Freeze baseline · View frozen
   baseline / Return to current draft; right side a chip
   `Cost Plan r{revision} · {status}`.
2. **Totals strip:** Funding target · Budget · Forecast final cost · Budget
   variance · `AUD · ex tax · price date …` · coverage. Values from
   `totals.overall` and `totals.buckets.overall`.
3. **Grid** inside a wrapper with `overflow-x: auto` (the page itself must not
   scroll sideways — `costs.spec.ts` asserts this at 390 px). Sticky header;
   Code/Category/Item sticky left; numbers right-aligned, tabular figures.
4. **Groups:** Fees and charges (`fee`), Construction (`works`), Project-wide
   (`project_wide`), each with a heading row and a subtotal row from
   `totals.fees` / `works` / `project_wide` and the matching `buckets` entry.
   Empty groups are not shown. A non-posting line is an indented group row with
   no figures; its children follow, indented.
5. **Sort:** clicking Code, Category or Item sorts within each group (client
   only, no persistence). Children stay under their parent.
6. **Placeholder columns:** header `aria-disabled="true"` and
   `title="Not tracked yet"`; cells "—" in the muted colour.

Formatting helper (display only, no arithmetic):

```ts
// Money arrives as exact decimal strings; group digits without parsing to float.
export const fmt = (m: string | null | undefined) => {
  if (m == null) return "—";
  const [whole, cents = "00"] = m.replace("-", "").split(".");
  return `${m.startsWith("-") ? "−" : ""}${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}.${cents}`;
};
const cell = (v?: CostValue) => v?.value_state !== "known" ? "—" : v.amount != null ? fmt(v.amount) : `${fmt(v.low)}–${fmt(v.high)}`;
const subtotal = (t?: CostTotal) => !t || t.lines === 0 ? "—" : t.amount != null ? fmt(t.amount) : `${fmt(t.known_subtotal)} + ${t.unknown} unknown`;
```

### Task 8: Inline budget edit

**Files:** `web/src/CostGrid.tsx`

- The Budget cell is a button (`aria-label="Budget for {label}"`). Click or Enter
  turns it into an input; Enter or blur saves, Esc cancels.
- Editable only when the line is posting, not excluded, the plan is not frozen,
  not viewing the baseline, and the budget is not a range (a range opens the
  drawer instead, so it is never overwritten by a single figure).
- Input accepts digits, one `.`, up to 2 decimals, commas stripped. Empty saves
  as unknown. Anything else shows the cell in error and does not send.
- Save calls the existing `costApi.value(projectId, plan, id, "budget",
  {...current, amount, low: null, high: null, value_state, origin: "user",
  meaning: "allowance"})` through the page's `save()`, so 409/422/500 handling,
  kept input and reload behave exactly as today. One request per save.

### Task 9: Drawer

**Files:** `web/src/Costs.tsx`

- Clicking a row (outside the budget cell) opens a right-hand drawer (fixed,
  ~420 px; full width under 900 px). Close button and Esc close it.
- Contents are the existing sections moved unchanged: line header, Edit cost
  line, value editors for **budget, commitment (labelled "Approved contract"),
  claimed to date**, Subdivide allowance (metric choices budget/commitment),
  Reviewed benchmarks (now applies to budget), Scope references.
- Add cost line and Cost settings open in the same drawer.
- Package scope lists (`packageApi.scope` per package) load when the drawer
  first opens the scope section, not on page load. Packages and works still load
  on page load (Category column and line form need them).

### Task 10: Playwright

**Files:** `web/tests/costs.spec.ts`

Rewrite the journey for the grid, keeping every existing guarantee:
1. Add line "Unallocated allowance" from the toolbar drawer; its row shows "—" in
   Budget, Forecast Final Cost, Budget Variance and Remaining.
2. Type 1000 into its Budget cell with the save route failing → alert "Could not
   save", the input still holds 1000; unfail → saved; row shows 1,000.00,
   Forecast 1,000.00, Variance 0.00, Remaining "—".
3. Placeholder headers Forecast Variations, Approved Variations, This Month are
   `aria-disabled`.
4. Open the row drawer → benchmarks, no apply button without a match.
5. Subdivide 300/400 from the drawer → children indented under the group row,
   explicit residual row present; API overall budget still 1000.00.
6. Freeze baseline → chip "Cost Plan r2 · draft"; View frozen baseline → chip
   "r1 · baseline", budget cells not editable, Add cost line disabled.
7. Screenshots at 1360 and 390 px; `scrollWidth <= innerWidth` still holds.

Run: `cd web && npm run build && npx playwright test costs.spec.ts`.

### Task 11: Commit and report

Commit Part 3 as `Cost plan grid with inline budget and line drawer (lane B)`.
Report: costs_read/costs_write p50/p90 against 100/250 ms, request count on
opening Costs before and after, the migration move count from Task 3, and a
browser check of Petersham.

---

## Edge cases

- Plan not yet started (`status: unavailable`): grid shows an empty state with
  Add cost line; no groups.
- Frozen baseline view: all editing off, drawer read-only.
- Version conflict (409) during an inline save: existing banner, cell keeps the
  typed value, Reload resets.
- Excluded line: row shown greyed with "excluded" tag; not in subtotals.
- Group line whose children are all unknown: subtotal shows "— " not 0.00.
- Wrong org: endpoints already 404; covered by Task 5.
- Legacy estimate on a baseline: not shown in the grid; it still feeds that
  baseline's forecast so its frozen totals do not move.
