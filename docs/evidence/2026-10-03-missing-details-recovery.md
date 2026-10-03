# Missing-details recovery verification

Implemented the [recovery flow](../design/2026-10-03-reprocess-missing-details.md)
and restarted the local server on 3 October 2026 at 20:49 Sydney time.
`/dev/build` reports server and web current.

## Live Bankstown result

The new endpoint queued one pass for each of the two existing documents. Both
remained filed throughout. Comparison with the before snapshot confirmed every
previously populated field retained its value, band and provenance.

| Document | Recovered number | Recovered title | Retained revision |
| --- | --- | --- | --- |
| 1115 CC-06 LEVEL 02 F.pdf | CC-06 | LEVEL 2 | F |
| 1115 CC-07 LEVEL 03 E.pdf | CC-07 | LEVEL 3 | E |

Both recovered number/title pairs are amber rule values, to be checked against
the scan. The ambiguous date was not guessed. CC-07's existing Architectural
discipline was retained.

**Remaining classification limitation:** the live CC-06 pass selected Structural
at confidence 0.63, admitted as amber by the existing discipline threshold. This
is inconsistent with the user's identification of the drawings as architectural;
the page includes a large consultant directory. After reporting that result, the
newly added discipline was corrected to Architectural through the field-edit API
using the user's supplied identification, recorded as a user value. It is not
evidence that automatic discipline classification was fixed. No classifier
threshold was changed.

## Checks

- Regression first reproduced missing titles/numbers for both measured title
  blocks; both now pass, as do full captured OCR-text replays.
- `go test -p 1 ./...` passed. The preceding parallel run tripped the existing
  5ms candidate-harvesting p50 gate (5.46ms) under concurrent package load.
- Web build passed. Existing OCR stages test and new recovery browser test
  passed. Desktop/mobile screenshots were inspected.
- New queue endpoint, 25 real authenticated HTTP requests: p50 3.05ms,
  p90 7.89ms, within 50/150ms.
- Installed OCR gate passed across 20 uploads and 20 recoveries of CC-06:
  upload p50 6159ms / p90 6201ms; recovery p50 5584ms / p90 5642ms,
  within the 10000/20000ms background budget.
- Recovery tests cover preserved values, deliberate blanks, concurrent edits,
  duplicate requests, expired leases, wrong org, malformed IDs, no new details,
  and OCR failure without unfiling or clearing fields.

Real OCR timing output is saved locally in
`.tools/tesseract/details-recovery-timing.json`. This gate uses CC-06's actual
PDF, the database queue and commit, and a replayed 350ms provider delay, plus a
2000ms polling allowance. It does not claim live-provider latency guarantees.
The earlier missing corpus/recording limitation on the full intake replay remains.
