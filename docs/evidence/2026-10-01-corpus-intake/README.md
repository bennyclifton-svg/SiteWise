# Corpus intake evaluation — 1 October 2026

**The testing regime is implemented and exercised. The 95% accuracy release gate is still failing.** These are diagnostic results, not a release claim.

## Scope and execution

Inventoried 544 PDFs across the three requested roots: 538 unique PDFs after six duplicate copies. Excluded 179 macOS AppleDouble sidecars. Ran offline extraction over every unique PDF and four live iterations over the same deterministic 153-file sample: 116 Petersham, 33 Bankstown and four Newham. The sample takes up to two documents per leaf folder and retains fixed calibration/held-out splits. It is not a random estimate of corpus accuracy.

The final sample was also uploaded through the actual SiteWise HTTP API to fresh QA projects in a separate local database: 133 filed, 19 without identity-page text and one upload rejected for size. All persisted field values on the 133 filed documents matched the draft evaluation. Polling-observed filing latency was 329 ms p50 / 532 ms p90; upload time is separate and this is not a VPS release benchmark.

The older server's baseline upload had a timeout, and one pending job was resumed during initial QA startup before database isolation. That run is retained for diagnosis, **not** used as a clean before/after comparison. The comparison below uses frozen draft runs with the same gold labels.

## Changes and measured effects

- Preserve both ends of the identity page within the 400-run budget.
- Reconnect touching PDFium glyph fragments without joining separate cells.
- Bind drawing-number captions to nearby cells using PDF coordinates.
- Settle a number when filename and unique explicit sheet-number evidence agree, even when the page references other drawings.
- Recognise underscore-separated sheet numbers and hyphenated issue suffixes; avoid treating browser duplicate suffixes such as ` (2)` as revisions.
- Ask Jev to select the explicit issue date in the existing single fan-out.
- Preserve bounded author cues and separate author discipline from subject matter or other consultants listed on the sheet.
- Restrict vocabulary rules so incidental prose and contact-panel captions cannot confidently identify the document's author.
- Keep technical drawings under Design, including construction-issued sets, as explicitly directed by the owner.

On independently annotated fields in the 153-file sample:

| Field | Correct before | Correct after | Wrong green before → after |
|---|---:|---:|---:|
| Drawing number | 0/29 | 29/29 | 0 → 0 |
| Lifecycle | 1/56 | 42/56 | 4 → 0 |
| Discipline | 9/65 | 0/65 | 1 → 0 |
| Kind | 30/119 | 15/119 | 11 → 2 |
| Revision | 22/26 | 22/26 | 0 → 0 |
| Title | 0/26 | 0/26 | 0 → 0 |
| Issue date | 17/26 | 17/26 | 0 → 0 |

More conservative rules reduced kind/discipline population as well as false positives. That is an explicit coverage regression. On the wider offline labelled subset, all 74 known drawing numbers now settle correctly. This does not establish accuracy on unlabelled PDFs.

Raw Jev choice or rule output matched 61/65 discipline labels and 26/26 titles on the final sample. These are diagnostics only: without calibrated confidence they do not authorise filing.

## Remaining blockers

All production Jev threshold lists were already empty before this work. They remain disabled for the changed `intake-6` state. Experimental fitting used only calibration-family labels and produced no supported green cutoffs: 27 discipline examples versus a 30-example floor, 42 kind examples without a safe cutoff, and smaller identity/lifecycle groups. No thresholds were invented or relaxed to increase the apparent strike rate.

Gold consists of independent draft page annotations plus 50 existing, unreviewed register/transmittal transcriptions. Many fields remain unlabelled; mixed reports/declarations and specification sheets need kind adjudication. The two remaining kind disagreements must be resolved against the documents, not by copying model output into gold. Families need review for related revisions stored across folders before any held-out release claim.

Across all unique PDFs, 83 lack first-page text and three exceed the 32 MiB identity limit. Some engineering title-block values are missing from the text layer even in an independent PDF reader. This implementation does not add OCR. Full-corpus extraction measured 72.7 ms p50 / 385.5 ms p90, failing the 250 ms p90 budget. The sample's faster latency cannot hide that failure.

At the end of this historical run, full-corpus live approval was pending. The owner subsequently authorised extended testing, and the full 538-file runs are recorded in [2 October evidence](../2026-10-02-corpus-intake/README.md). Only extracted identity text was sent to TypeSafe; full PDFs stayed local.

## Reproduce and inspect

See [CORPUS.md](../../../data/eval/intake/CORPUS.md) for commands and label format. `tools/check-corpus.ps1` runs exact-request replay and persisted-app scoring without live calls, returning nonzero when either gate fails. Public CI exercises the Go harness and Python upload/resume tests; private documents and recordings never go to hosted CI.

Private evidence under `data/eval/intake/private/`:

- `corpus-gold-v3.json`: draft evidence-backed labels.
- `score-live-baseline/`: initial live run rescored against the same labels.
- `approved-live-04/`: final live traces, recordings and experimental fitting.
- `final-replay-04/`: exact replay after the separator geometry optimisation; no request misses, accuracy gate correctly fails.
- `full-final-offline/`: final extraction/rules over all 538 unique PDFs.
- `app-fixed-04/`: persisted app results, QA project IDs and zero draft-value differences. `app-score-04/` contains the failing app accuracy report.

QA is available at <http://127.0.0.1:8081/dev/login> in database `sitewise_qa_20261001`. The user's original development server is preserved. The QA build precedes only the final separator-call optimisation; exact replay confirmed that optimisation did not change any Jev request.

Validation: full Go suite and `go vet ./...`, plus Python HTTP tests covering upload, polling, saved fields, oversized-file rejection and resume. Regression coverage includes geometry, candidate extraction, lifecycle policy, incidental vocabulary, sample determinism, duplicate exclusion, calibration split isolation, missing candidates, false greens and baseline regressions.

[summary.json](summary.json) contains aggregate metrics only. Detailed PDF text, filenames, labels and provider recordings remain git-ignored.
