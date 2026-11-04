# Invoice ledger: compact rows and inline discipline

25 October 2026. Owner-requested layout and assignment editing. Lane A for
saved discipline edits; the column and density changes are Lane B.

Columns: invoice date, invoice number, discipline, activity line, month,
ex-GST (AUD), status. Discipline is the existing procurement package label.
Its dropdown saves through the existing invoice PATCH endpoint, returns the
recomputed ledger, and never calls Jev. Failed saves keep the saved value and
show a retry message. Saves are serialised to avoid competing ledger responses.

Each allocation stays on one visual line. Provenance chips and check details
remain in the invoice panel, opened through the invoice number. Status remains
visible; review reasons are also available on hover. GST and inclusive total
remain in details and exports. Amount calculation, filing, split allocations,
unknown values and export schemas are unchanged. No new dependency.

Verification: frontend build and invoice browser scenario passed, including
inline assignment, clearing, failed-save retry, re-mapping, totals and export.
Desktop, mobile and print inspected. Local 20-sample discipline writes on the
10-invoice fixture measured p50 10.4 / p90 11.7 ms against 50 / 100 ms.
The 1,000-invoice read benchmark and full filing benchmark were not rerun.
Read budget remains 150 / 300 ms; filing remains 1,000 / 2,000 ms.
Not deployed.
