# Cost plan: direct editing, compact rows and contractor categories

Date: 11 October 2026. Status: approved and implemented locally.

**Outcome:** a compact ledger where users enter contract values directly and
read Category as who delivers the work, and Item as what is being priced.
This reduces clicks and improves time-to-decision. **Lane A:** monetary edits
and package assignments must preserve totals, scope and historical records.

## Implementation

1. **Edit Approved Contract in the cell.** Generalise the existing Budget
   editor to also save `commitment` through the existing value API. Click to
   type; Enter or blur saves once; Escape cancels. Blank means unknown, not
   zero. Preserve typed input on failure and refresh forecast, variance and
   totals from the saved response. Existing range values retain the detailed
   editor so a click cannot silently replace a range with one number.

2. **Make the grid compact.** Reduce cell padding, row height, input width and
   excessive column minimums. Aim for approximately 28–32 px ordinary desktop
   rows; retain readable figures, visible focus and usable touch controls.
   Wrap column headings and expose truncated labels on hover/focus. Keep the
   sticky identity columns, four section headings and far-right bin. The
   drawer remains for subdivision, ranges and other detailed settings.

3. **Correct construction Category editing.** The current editor changes the
   building system; replace its options with active construction procurement
   packages and save `package_id`. Preserve the line's system/work-item link.
   Update the add-line flow to distinguish contractor assignment from physical
   scope instead of presenting systems as contractor categories.

4. **Apply project procurement assignments consistently.** Use explicit line
   assignment first, otherwise an applicable trade package, then the project's
   head-contractor/D&C package. Display the actual package title. Offer a clear
   “Use automatic assignment” option. With no matching package, display
   Head contractor as the default planning role with no assigned package ID.
   Ambiguous matches and missing explicit assignments remain Unassigned.
   Never infer
   the arrangement from the user's role or invent a contractor package.
   Resolve the package ID and label together; keep draft category display and
   package totals consistent without rewriting frozen baselines.

5. **Preserve the two-level meaning.** Under Consultants, Category remains the
   consultant package and Item starts as its fee; subdivision creates the
   project's chosen stage fees. Under Construction, Category is the head
   contractor or trade, while Item is the system/works allowance (for example,
   Electrical works). Subdivision creates detailed scope items under that
   package. Children inherit the package unless explicitly reassigned; amounts
   remain on posting children and count once. Do not merge, reprice or rename
   existing user-edited lines when procurement changes.

## Verification and boundaries

- Test direct contract entry, clearing, invalid input, failed-save retry,
  Enter/blur deduplication, stale versions and read-only baselines. Confirm
  forecast and variance update correctly without opening the drawer.
- Test single-head-contract, multiple-trade, explicit reassignment, ambiguous
  and retired-package cases. Confirm changing Category preserves physical
  scope and that consultant stages/trade subdivisions retain correct totals.
- Inspect desktop and 390 px layouts, keyboard editing and horizontal scrolling;
  run the cost browser tests, relevant Go tests, access checks and frontend build.
- Retain p50/p90 budgets: reads and edits **100/250 ms**, automatic preparation
  **250/500 ms**, and filing **1,000/2,000 ms**. Extend the existing benchmark to
  measure contract-cell writes and package reassignment; report actual timings.
- Scope is Budget/Approved Contract direct editing. Claimed-to-date date rules,
  Excel-style paste/fill/formulas, new procurement modes, automatic prices and
  deployment are outside this change. No new dependency, service or AI call.

Primary files: `web/src/CostGrid.tsx`, `Costs.tsx`, `costs.css`, `costApi.ts`;
`internal/costs/suggest.go`, `totals.go`, relevant store validation/snapshot
code, and existing cost tests/benchmarks. Reuse the current model and APIs;
any need for a schema change must be justified before implementation.

## Implementation record

- Approved Contract uses the shared inline money editor: cancel, clear,
  validation, failed-save retry, single-save Enter/blur and range protection.
- Reduced row padding, column/input widths and desktop bin size. Touch controls
  remain larger. Desktop and 390 px layouts were inspected.
- Construction Category edits the procurement package; physical scope stays
  intact and is selected separately when adding a construction line.
- One resolver provides package ID and label for category display and cost-plan
  package totals. It handles explicit ancestor inheritance, trade precedence,
  ambiguous matches and retired explicit assignments.
- Baselines capture resolved package IDs in existing revision provenance beside
  the category snapshot. No migration is needed. Old baselines keep their former
  assignments; current drafts resolve against current procurement.
- Construction subdivision children and residuals inherit the parent package
  unless explicitly assigned. Legacy work-item design-responsibility links
  remain supported. Tender schedules retain their explicit scope rules.

Frontend build, cost/store/API and report regression tests, route-access sweep,
package snapshot/inheritance tests and both cost browser scenarios pass. The
browser tests cover contract entry, failed-save retry, invalid input, clearing,
Enter/blur deduplication, frozen controls and reassignment without scope changes.
The design scan reports only the existing 8 px sort-arrow type-ramp finding.
Diff whitespace checks pass.

Dedicated local endpoint benchmark (three samples per new path):

| Path | p50 / p90 | Budget p50 / p90 |
|---|---|---|
| Contract cell edit | 6.2 / 6.8 ms | 100 / 250 ms |
| Contractor reassignment | 6.8 / 7.3 ms | 100 / 250 ms |
| Automatic preparation | 19.1 / 23.3 ms | 250 / 500 ms |
| Cost reads | 1.9 / 6.2 ms | 100 / 250 ms |

The endpoint benchmark gate passed. Replayed filing measured 63.0 / 1,218.7 ms,
within the 1,000 / 2,000 ms filing budget. These are local measurements, not VPS
results. The previously recorded unrelated full-suite profile-layout failure
was not changed; this turn ran focused regression coverage.

No new dependency, service, AI call or automatic monetary value. Not deployed.

## Follow-up: default construction role

Approved 11 October: unmatched construction lines display **Head contractor**.
Explicit assignments (including inherited assignments) and unique trade/head
packages still take precedence. Multiple matching packages remain Unassigned;
retired packages are ignored for automatic selection but retained when explicit.
The fallback is a planning label only: costs stay unallocated in package totals
until an actual package resolves. Current drafts update on read; frozen snapshots
retain their saved labels and assignments. No schema change or deployment.

Lane A; outcome is a usable starting plan without manual category entry.
Existing reads/edits retain p50/p90 100/250 ms and preparation 250/500 ms.
Package creation, role inference and monetary changes are out of scope.

Verification: cost domain, store and HTTP cost regression tests pass, including
ambiguous matches, unallocated totals and frozen-label preservation after adding
a contractor package. The endpoint benchmark passes: reads 1.8/2.3 ms,
preparation 18.0/18.3 ms, contract edits 6.2/6.7 ms and package assignment
6.0/7.3 ms (p50/p90, local). No new dependency.

## Follow-up: automatic ledger numbering

Replace editable Code with read-only No.: 01, 02, 03 across all four sections,
including group and child lines but excluding headings and subtotals. Number
the displayed rows consecutively after sorting, adding or removing lines.
Remove Code from the detailed editor too. Existing stored codes remain intact
for compatibility; stable internal item IDs continue to identify linked records.
The display number is not an invoice/variation matching key. Future linking
should present item names (with category context) while storing the item ID.

Lane B: remove an unnecessary user entry and reduce time-to-decision. No new
API calls, schema changes or dependencies; existing read/edit budgets remain
100/250 ms p50/p90. Invoice and variation workflows are outside this change.

Frontend build and both existing cost browser scenarios passed (9.7 seconds
total). Desktop and mobile screenshots checked. No server path changed; endpoint
timings above remain the latest measurements. Not deployed.
