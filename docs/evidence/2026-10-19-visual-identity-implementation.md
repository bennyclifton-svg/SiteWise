# Image-assisted filing review

Implemented locally and exercised on 15 existing Hale sheets plus the Hale
Design Brief. New PDF filings queue the same durable background job. Existing
PDFs offer Read document images in the expanded register row. Review has source
image tabs, enlargement, literal suggestions and explicit Use actions through
the existing correction endpoint. No suggestions were applied in the pilot.

## Measured results

The implemented three-field fan-out returned 12/15 correct drawing titles
(80%), two wrong view-caption choices (Sheets 6 and 7), and one unresolved
multiline title (Sheet 5). This is below the earlier best experiment (13/15),
but above the same corpus's existing raw native reader (7/15) and forced OCR
(0/15). These are small-sample, draft-key results, not calibrated acceptance
rates. Number/revision accuracy across the drawings has not been scored.

Live background preparation plus Jev: p50 3064 ms, nearest-rank p90 3699 ms,
15 sheets; queue waiting excluded. The 20-run installed-runtime gate on a real
Hale sheet passed: filing p50 392 ms / p90 399 ms (1000/2000 budget), visual
p50 2880 ms / p90 3161 ms (10000/20000 budget). This gate uses real extraction,
rendering, OCR and persistence with replayed provider delays of 350/600 ms.
tools/check-visual.ps1 is required by tools/check.ps1; the foreground benchmark
excludes this separately gated background path.

The Design Brief returned title Design Brief and project revision P1 in 907 ms.
Its control-page image was inspected: P1 is the project revision, while V1.2 is
the separate base-document revision. One report is a smoke test, not evidence
of general report/specification accuracy. Only the first three pages are read.

## Validation and limits

Focused Go tests pass for image opt-in/model isolation, invalid assets, durable
review, organisation isolation, replay caching, unchanged owner corrections,
the review endpoint, existing OCR filing, list/correction budgets and committed
budget definitions. Web build and focused Playwright acceptance/mobile test
pass. The design detector has no findings. Manual browser inspection verified
the real Hale suggestion beside the source; enlargement was added after the
bottom strip proved too small at fit width.

The broader Go run was interrupted during an unrelated store test that stopped
making progress; this is not a claim that the entire repository suite passed.
No new dependencies or service. Existing PDFium/Pillow/Tesseract are reused.
Evidence is bounded and stored with the document, cascades on deletion and is
available only through authenticated, organisation-scoped access.

This improves the mechanics of checking a filing, but human checking time has
not been measured. Unknown layouts, late control pages and ambiguous view
captions still need manual review. Box remains excluded following its failed
trial. See the [implementation scope](../plans/2026-10-19-visual-document-identity.md)
and [experiment](2026-10-19-titleblock-vision-experiment.md).
