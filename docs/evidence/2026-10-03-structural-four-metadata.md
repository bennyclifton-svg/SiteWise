# Bankstown S0001–S0010 missing metadata

Lane A: investigate four uploads from 3 October 2026 and repair only their
missing title/discipline. Preserve existing fields, tenant boundaries, the
1000/2000 ms foreground budget and one Jev fan-out per state.

## Reproduction and cause

S0001, S0002 and S0003 had blank disciplines at confidence 0.56, 0.54 and
0.39 respectively; S0010 was Structural. These were withheld answers, not
three saved incorrect disciplines. The PDF text reader returns the architect's
contact panel, but the engineer's name and Consulting Engineers have broken
character mappings. Other printed information is absent from native extraction.
The short filename title Notes is separately rejected by the generic caption
filter, leaving S0001 with no title candidate.

A failing regression reproduced the filename omission. The parser now accepts
Notes in the title slot of the already recognised complete
job_sheet_title-(numeric issue) convention. Bare Notes page captions and other
filename captions remain excluded. This source change is not deployed.

## Recovery and saved result

Read-only local OCR recovered STRUCTURAL and Consulting Engineers on all
three affected sheets. With explicit owner approval, the existing Jev question
selected Structural at confidence 0.94, 0.85 and 0.98. No prompt, threshold,
provider or runtime dependency changed. The request retains TypeSafe's
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [single fan-out](https://docs.typesafe.ai/patterns/fan-out) design.

The existing missing-details endpoint returned 409 for all three sheets: it
only admits documents already marked as OCR-processed. Native text, however
incomplete, currently prevents automatic OCR. No recovery eligibility or
automatic OCR policy was broadened in this task.

The printed S0001 title block was independently rendered and visually checked:
GENERAL & CONSTRUCTION NOTES, S0001, issue 03, discipline STRUCTURAL.
Through the authenticated field-edit API, the three blank disciplines were
corrected to Structural and S0001's blank title to that printed title. These
are recorded as user corrections, not automatic Jev results. S0010 was left
unchanged. Before/after API comparisons confirm every previously populated
field retained its value, band and provenance. Original files are unchanged.
Private snapshots and read-only probes remain in ignored .tools/.

## Verification

- The Notes regression fails before the parser change and passes afterward.
- Full Go suite passes with the dedicated test database, including isolation
  and correction protection. Diff whitespace check passes.
- Harvest p50/p90: 4.149/4.519 ms, within 5/10 ms.
- Rules p50/p90: 0.343/0.810 ms, within 1/1 ms.
- Three live OCR-plus-Jev diagnostic samples: 3162, 3414 and 2146 ms.
  These are not percentile or foreground filing measurements. OCR remains
  background work with its existing 10000/20000 ms budget.

The stored records are repaired. Automatic discipline recovery for PDFs with
partially broken native text remains a follow-up; this investigation does not
claim that class of upload is fixed or establish broader classifier accuracy.
- Full foreground replay gate could not produce a valid timing verdict:
  215 Jev requests have no matching recorded document. It fails closed and
  requires refreshed recordings. No full-path 1000/2000 ms pass is claimed.

## Deployment

Deployed to the existing local port-8080 app with owner approval on 3 October
2026 at 21:30 Sydney time. The development launcher rebuilt and restarted the
binary. Health returned HTTP 200 and /dev/build reported server and web current.
Authenticated reads confirmed the three repaired documents retained exactly
the same saved field values and provenance after restart. This supersedes the
source-only deployment status above; automatic sparse-text OCR is still unchanged.
