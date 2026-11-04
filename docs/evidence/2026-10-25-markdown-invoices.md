# Markdown invoice recovery

Lane A defect: invoice intake rejected `.md` and `.markdown` although the
document register already supported them. No extraction or mapping ran.

Invoice intake now routes these extensions through the existing bounded UTF-8
Markdown extractor. The invoice harvester removes paired bold markers and
treats Markdown lines as self-contained, preventing the invoice-date label
from also labelling the following due date. The picker accepts both extensions
and unsupported/unreadable messages no longer assume the original is a PDF.
Re-dropping a previously unsupported file resumes its original ledger row,
provided nobody has corrected or split it. Filed invoices remain deduplicated.

No dependency, model, prompt, threshold or budget change. Jev still picks the
open fields in one fan-out; Go parses dates and computes money, following
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [fan-out](https://docs.typesafe.ai/patterns/fan-out).

## Verification

- Reproduced the format rejection with the uploaded synthetic Quoin fixture.
- Enabling the extension alone exposed missing invoice-number parsing and
  ambiguous issue/due dates; the same regression test now passes for both
  extensions, including mixed case, all money fields and supplier/line mapping.
- Invoice HTTP/store tests pass: upload, recovery, deduplication, ledger,
  corrections, export, org isolation and file preservation.
- Full `internal/invoices` and `internal/identity` tests pass; web build passes.
- Twenty HTTP upload-to-ledger samples with test Jev: p50 82.4 ms, p90 100.6 ms
  against 1,000 / 2,000 ms. This is local test-provider evidence, not a live
  provider percentile or VPS measurement.

## Local recovery

Rebuilt/restarted the local app and re-dropped the original bytes. Invoice
`9716eee1-1021-4536-9158-cefe96b3132a` in Kavanagh Residence 2 was reused
(`created: false`). Network sandbox restrictions initially left Jev fields grey;
after explicit owner approval, the server was restarted with network access.
The existing background recheck completed in 220 ms of Jev request time.

Verified through the ledger API: QUA-2601, 18 March 2026, Architect package,
1. Schematic Design activity, AUD 24,000 ex GST + 2,400 GST = 26,400 total.
Jev fields remain amber under the existing calibration; status is Needs review.
The March invoice date versus October upload/certification period is an
informational flag. No values were manually confirmed on the owner's behalf.

Unchanged limitation: unreadable files still need manual entry, and Jev failures
keep rule-derived fields while mapping awaits the existing background recheck.
