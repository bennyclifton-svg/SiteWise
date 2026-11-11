# Cost Plan: right-click menu and bulk actions

Owner request, 11 Nov 2026, after the Accommodation schedule: give the Cost
Plan the same selection behaviour. Shift-click selected the range but also
highlighted the page's text; there was no right-click menu and no way to
delete several lines at once. Brief agreed the same day ("This looks please
proceed"), with Duplicate and a bulk category/package set.

## Outcome

- Shift-click and Ctrl-click select lines without selecting text (the
  Accommodation schedule too).
- Right-click on a line opens the menu the document register uses; on a
  selected line it acts on the whole selection. One line: View line details,
  Subdivide allowance, Set category or package, Exclude or Include in
  totals, Duplicate, Delete line. Several lines: Set category or package for
  N lines (when they are one kind of line), Exclude N / Include N, Duplicate
  N lines, Delete N lines.
- Bulk delete from the menu, the Delete key, or the bin with the count in the
  table head while lines are selected. One confirmation names how many lines
  go, how many more their group lines take, and how many hold a contract
  amount or claimed invoices.
- Duplicate copies label (marked "(copy)"), category or package, quantities
  and the budget allowance, straight after the original; never the contract
  amount or what has been claimed. A group line is not copied; its lines are.
- A frozen baseline or the history view offers View line details only.

Serves time-to-decision: tidying a prepared plan was one line at a time.

## Lane

A (cost figures). Each action is one write in one transaction and one plan
version; deletion follows the one-line delete (a group line takes its
children, frozen history stays, a removed default stays dismissed). A stale
plan is refused (409) and reloaded, as today. No Jev.

## Budget

New path `costs_bulk`, the cost write budget: p50 ≤ 100 ms, p90 ≤ 250 ms,
benchmarked in `cmd/intake-bench` (exclude five lines, include them, copy
them, delete the copies).

## Out of scope

Undo, copy and paste between cells, editing an amount across several lines.

## Result (11 Nov 2026, built, uncommitted on main)

- `internal/store/cost_bulk.go` (`BulkCostItems`), route
  `POST /api/projects/{id}/cost-plan/items/bulk`, grid menu, head bin,
  Delete key and category/package picker in `web/src/CostGrid.tsx`.
- Tests: store and API tests for each action, a stale plan (409), another
  org (404) and bad input (422); browser test `web/tests/cost-bulk.spec.ts`
  (Shift-click selects no text, menu items, exclude, set category, duplicate,
  Delete key with one confirmation). The existing Cost Plan browser tests
  pass except `cost-disclosure.spec.ts`, which also fails without this
  change (one run in three): an invoice event lands during its timed loop
  and the ledger is read again.
- Timing (replayed bench): `costs_bulk` p50 8.1 ms, p90 10.6 ms (budget
  100/250); `costs_delete` p50 5.4 ms; `costs_write` p50 3.4 ms.
