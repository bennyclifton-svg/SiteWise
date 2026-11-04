# Keep the workbench location on refresh

25 October 2026. Lane B defect fix: refreshing Cost Plan / Invoices returned
to Profile because workbench visibility and cost-tab state started from fixed
defaults on every mount.

The current workbench view and cost tab now live in URL query parameters
(`view` and `costTab`). Choosing a page or tab replaces the current history
entry; it does not reload the page. Startup validates those values against
known choices before rendering. Missing or unknown choices fall back to
Profile / Cost plan. Existing project paths and access checks stay unchanged.
Invite-token removal preserves the query string. No storage, API call, Jev
call or dependency is added. This preserves navigation, not unsaved forms,
filters, open documents or dialogs.

The real-browser regression test failed before the fix at the missing
Invoices tab after refresh. It passes afterward with the browser cache
disabled, covering all three cost tabs, Procurement, Specifications, Program,
Profile, and invalid query values. The invoice regression and frontend build
also pass. Existing invoice-write timing: p50 11.1 / p90 12.6 ms on the local
10-invoice fixture (20 samples), against 50 / 100 ms. No server read or filing
path changed; their full benchmarks were not rerun. Ledger read budget stays
150 / 300 ms; filing stays 1,000 / 2,000 ms. No deployment performed.
