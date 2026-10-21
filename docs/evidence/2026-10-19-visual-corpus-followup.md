# Hale visual identity follow-up

Lane A: make the title review usable in the register, test the existing reader
against other disciplines/reports, and fix demonstrated failures. No automatic
acceptance, new AI provider, dependency, or changes to source tender documents.

## Scope and method

27 title checks from 13 source PDFs in the owner's Hale tender pack: all 15
architectural pages; one page each from structural, civil, mechanical, hydraulic,
electrical, fire and scanned existing-hydraulic drawings; BCA, accessibility,
geotechnical, infrastructure and contractor-scope documents. Reports use their
first three pages. Drawing samples outside architecture are not whole-set tests.

Each provider request ran sequentially. The answer key is draft, manually
transcribed from rendered pages and never sent to Jev. Exact title scoring ignores
case and whitespace only; extra address words, missing words and punctuation
errors count as failures. The printed accessibility-title typo is retained.
These are development examples used for iteration, not held-out validation or
calibration of automatic acceptance. Number/revision accuracy is not represented
by the title score.

Private evidence: `tmp/visual-corpus/manifest.json`, `answer-key.json`,
`results.json`, `metrics.json`, `review.html`, exported requests and raw responses
with request fingerprints. Source files were read only; copies stayed in tmp.

| Title reader | Exact matches | p50 | p90 |
|---|---:|---:|---:|
| Existing native-text candidates / raw picks | 9/27 | 287 ms | 339 ms |
| Forced existing OCR / raw picks | 6/27 | 2,290 ms | 3,438 ms |
| Previous visual reader | 20/27 | 3,155 ms | 4,466 ms |
| Revised visual reader | 23/27 | 2,820 ms | 3,737 ms |

Times include extraction/preparation plus live provider wall time, exclude queue
wait, and are over successful attempts (27 native, 25 OCR, 25 prior visual, 27
revised visual). Failures remain in accuracy denominators. Raw native/OCR picks
are not the number of values actually accepted under current blank thresholds.

Architecture improved from 12/15 in the prior visual sample to 14/15. Remaining
sample failures: architectural sheet 6 chooses a view caption; the structural
sketch is unresolved; the geotechnical title includes an address; the scanned
hydraulic title has stray punctuation. Numbers/revisions remain review-only;
examples still include a view reference as a number and an issue description as
a revision. Do not infer safe automatic filing from the title match rate.

## Confirmed causes and fixes

- Visual proposals were available only after expanding a row. The list now
  fetches small provisional title labels in one org-scoped bulk query; images
  remain lazy. Saved fields, including intentional user blanks, take precedence.
- A durable visual-completion event updates the existing row without a reload.
  Split parents display their filename and “Original drawing set”, instead of
  presenting the retained container as another missing sheet title.
- Repeated candidate lists exhausted Jev's context on the mechanical drawing.
  Only title context is repeated, keeping number/revision candidates in their
  questions. One background fan-out remains the limit.
- The accessibility report is 40,470,600 bytes, exceeding the old 32 MiB cutoff.
  The visual path now permits up to 128 MiB while retaining page, pixel, image,
  output and time limits. It does not render the whole report.
- Bottom-edge halves are enlarged and stacked; the right edge and overview remain
  available. Region OCR precedes whole-sheet noise in the bounded candidate list.
  Adjacent literal title lines may have different font sizes.
- Most importantly, model results depended on evidence order. The separately
  exported benchmark and live worker initially diverged (14/15 vs 11/15).
  Comparing all 15 showed identical images and candidate values. Controlled
  repeat probes on sheets 7, 9 and 10 isolated candidate-first vs candidate-last
  ordering. The final v4 request explicitly places title candidates after image
  evidence, preserved by both the exporter and runtime JSON serialization.
  A regression test locks that order down; this is not a claim of model determinism.
- Cache versions allow an explicit review request to refresh an older result
  without resetting a live lease or modifying a user decision. The existing Hale
  sheet records were requeued, preserving their IDs and accepted fields.

## Box

Seven additional Box/Noul probes were made on enlarged drawing regions and report
covers. Boxes selected neighbouring project cells or clipped title lines; even
the BCA box clipped its subtitle. Raw outlined images are in the private report.
These results do not justify deploying automatic Box cropping. The application
uses the useful image path and preserves broad evidence for human review.

Followed TypeSafe's [Images](https://docs.typesafe.ai/images) and
[Box](https://docs.typesafe.ai/primitives/box) contracts: pinned Jev 1.14, opt-in
header, referenced assets, normalized box edges and a presence question. Both
capabilities are described by TypeSafe as experimental. Confidence was not
converted into automatic acceptance.

## Validation

- Go visual/image tests across intake, HTTP API and Jev pass, including org
  isolation, user corrections/blanks, bounded candidate context, mixed-size
  title lines, ordering and the actual project-list summary.
- Playwright passes: a background completion updates a collapsed row, acceptance
  is explicit, edited revision is protected, mobile panel fits, split parent
  remains identifiable. Frontend build and UI detector pass.
- Installed 20-document gate on a real Hale sheet: filing p50 399 ms / p90 411 ms
  (budget 1,000 / 2,000); visual p50 3,107 / p90 3,193 ms (budget 10,000 / 20,000).
  This gate replays 350 ms / 600 ms provider latency; live provider measurements
  are listed separately above. Queue time is excluded from visual measurements.
- Live 41-document register, 25 HTTP reads: p50 23.3 ms / p90 26.8 ms against the
  50 / 150 ms budget. This is the local project, not a maximum-size load test.
- Final live v4 recheck: all 15 jobs ready and all 15 sheets have a saved or
  provisional title; 14/15 match the answer key. Sheet 6 still selects the view
  caption. Sheet 15's existing saved title remains authoritative. The retained
  source set and sheet IDs are unchanged.
- No new dependencies. The whole repository test suite was not claimed as run.

Reproduce requests with `cmd/titleblock-experiment -visual` (or `-visual-only`
after recording the legacy baseline), then run `tools/visual_corpus_benchmark.py`
with `--manifest`, `--exports`, `--output`, and `--mode visual|native|ocr`.
The runner checks source hashes and refuses mismatched cached requests. Keep
answer keys separate and use a new output directory for each changed request.
