# Cost plan group disclosure

Owner approved 28 October 2026. Lane A: displayed money must remain trustworthy.

Outcome: switch between a compact package summary and individual activity fees,
and follow an invoice to its exact cost line. This reduces time-to-decision.

- Add per-group and Expand all / Collapse all controls. Start expanded and
  remember choices per project in optional browser storage.
- Expanded group headings suppress repeated amounts; a budget held only on
  the group remains visible. Collapsed headings show Go-computed summaries.
- Keep section and overall totals, allocations, accepted fees and baselines
  unchanged. Nested groups, unknown/range values, excluded lines, missing
  storage and removed invoice destinations must remain understandable.
- Invoice activity labels carry category context. Following a live activity
  opens the cost plan, expands ancestors and selects the line. Group spend
  links filter to the group's descendant activities, including in exports.
- No new dependencies, services, migrations or Jev calls. Existing read/write
  budgets remain 100/250 ms p50/p90; invoice reads 150/300 ms. Disclosure is
  local, with a browser gate of 100/250 ms p50/p90 and no network requests.

Implementation: add display summaries without changing posting totals; wire
disclosure and invoice navigation; run domain/API/browser regressions, a
frontend build, speed checks and desktop/mobile visual inspection. Exported
cost sheets retain their full detail; browser print follows the visible view.

## Implementation and verification

Implemented locally. Cost totals now carry separate `rollups`; invoice reads
carry separate `groups`. Both are computed in Go from the existing records.
Neither enters posting totals. Approved contract and forecast remain separate,
unknowns and ranges survive aggregation, and a split invoice counts once per
group. Group invoice filters also apply to CSV/Excel exports.

The grid starts expanded, stores collapsed IDs per project, supports keyboard
chevrons and both bulk actions, and retains a parent-held budget. Invoice
activity links switch tabs, expand ancestors, scroll to and focus the selected
line. Allocation choices name category and activity and omit summary groups.
Unavailable/removed activities have no navigation link. Storage failure leaves
disclosure usable for the current visit. No monetary writes occur on disclosure.

Validation:

- Frontend build passes. Existing bundle-size advisory remains.
- Cost, invoice, store and HTTP focused regressions pass (`Test(Group|Subdivision|
  Cost|Invoice|Award)`), including unknown/range rollups, nested groups,
  budget ownership, credits, distinct split-invoice counts and wrong-org reads.
- Existing cost-grid and invoice-ledger browser journeys pass, including editing,
  baseline read-only behaviour, invoice remapping and workbook reconciliation.
- New browser journey reproduces the six fees totalling 96,500; checks keyboard
  disclosure, bulk actions, persistence after refresh, unchanged API totals,
  group invoice filtering/export and return to the focused Concept activity.
- Desktop and 390px screenshots inspected; page stays within viewport, table
  scrolls horizontally. Final alignment/scroll adjustment confirmed once.
- Mechanical design scan finds only existing 8px sort-arrow typography and 4px
  menu-radius advisories. Diff whitespace check passes.

Local p50 / p90 measurements (20 samples for the new checks):

| Path | Measured ms | Budget ms |
|---|---:|---:|
| Collapse all, click through paint | 27.8 / 31.9 | 100 / 250 |
| Cost totals with groups, HTTP | 2.17 / 2.33 | 100 / 250 |
| Invoice ledger with groups, HTTP | 2.16 / 2.97 | 150 / 300 |
| Existing cost write regression | 2.84 / 3.29 | 100 / 250 |
| Invoice discipline write, browser fixture | 9.2 / 9.9 | 50 / 100 |

Disclosure made zero API requests. No live Jev call, new dependency, migration,
deployment or production data change. The shared Windows Go cache denied some
test-server reads; browser verification used a temporary isolated cache and
temporary Playwright config, removed afterwards. Existing unrelated checkout
edits were preserved. These are focused local checks, not a full-suite run or
VPS measurements.
