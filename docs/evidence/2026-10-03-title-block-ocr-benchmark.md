# Bankstown title-block OCR benchmark

## Result

150 DPI is the fastest tested setting that retains the four checked identity
values on all five drawings. Render plus OCR p50/p90 is **625/723 ms**. This is
an offline feasibility result, not a passing SiteWise filing latency gate.

| DPI | PDF render + PNG p50 / p90 | OCR p50 / p90 | Combined p50 / p90 | Drawings with all four checks passing |
|---|---|---|---|---|
| 150 | 199 / 286 ms | 417 / 432 ms | 625 / 723 ms | 5/5 |
| 200 | 203 / 293 ms | 494 / 516 ms | 697 / 792 ms | 5/5 |
| 300 | 235 / 307 ms | 585 / 616 ms | 824 / 920 ms | 5/5 |

Each setting has 15 timed samples: five documents, three repeats. Percentiles
use nearest rank on per-sample durations; combined percentiles are measured,
not sums of stage percentiles. OCR text was identical across repeats at each
resolution. Repetition measures timing stability, not 15 independent documents.

## Method and scope

- Windows, AMD Ryzen 5 3600 six-core CPU. Sequential execution, one OCR thread
  (`OMP_THREAD_LIMIT=1`), Tesseract 5.4.0.20240606, English `tessdata_fast`.
- Native PDFium 153.0.7999.0 through pypdfium2 5.13.0; Pillow combines two
  grayscale crops. These are existing local evaluation dependencies, not new
  SiteWise runtime dependencies. SiteWise uses a different WebAssembly PDFium
  binding, so its production rendering cost still needs measuring.
- Original PDFs in `Bankstown Final Contract Set/07 Architecture`, CC-01 to
  CC-05, byte-identical to the reported uploads. Input and model hashes are
  retained in the accompanying results JSON.
- Two fixed, manually inspected regions per sheet: the upper-right revision
  table and lower-right title block. Both are rendered directly as crops,
  joined with white gutters, and recognised in one Tesseract invocation.
  These coordinates are template-specific; automatic crop discovery is not
  implemented or timed. The lower crop includes a partial preceding consultant
  row, but all checked title-block cells and date rows are fully contained.
- Tesseract LSTM mode (`--oem 1`), uniform-block segmentation (`--psm 6`).
  Each sample reopens the PDF, renders and saves PNG, starts Tesseract, and
  captures its text. Startup and image encoding are included. No warm-up is
  discarded; filesystem caches may already be warm. Resolution order rotates
  between repeats. This is an ordinary development machine, not isolated load.
- Excludes upload, crop detection, SiteWise's initial text extraction, Jev,
  candidate harvesting, database writes and concurrent production workloads.

An exploratory sparse-text pass (`--psm 11`) read the dates but dropped the
isolated revision letters from the history table. This is why reading a date
alone is not counted as successful current-date evidence. After inspecting
the initial crops, the lower region was enlarged to include the original-design
line and the upper edge adjusted. The final timings above use those revised
regions. The two passes are not a controlled segmentation-only comparison.

## Accuracy checks

Expected values were read visually from the rendered original title blocks:

| Drawing | Printed drawing title | Revision | Printed date on matching revision row |
|---|---|---|---|
| CC-01 | SITE SETOUT PLAN | D | 09/07/2015 |
| CC-02 | BASEMENT 2 | F | 09/07/2015 |
| CC-03 | BASEMENT 1 | F | 09/07/2015 |
| CC-04 | GROUND FLOOR PLAN | J | 14/07/2015 |
| CC-05 | LEVEL 1 | H | 07/09/2015 |

CC-05's printed September date is retained literally; it was not corrected to
July based on neighbouring sheets. The title does not contain the filename's
leading zero in `LEVEL 01`.

All 45 final samples contain the title and drawing number, a revision label
with the correct value, and the current revision letter followed by its date
on the same OCR line. Number spacing and revision-label case are normalised
for these checks. This measures presence of usable evidence, not successful
production field selection: noisy captions, punctuation and non-target text
remain. In particular, PSM 6 can omit prominent issue/issuer text. No claim of
correct discipline/kind classification, perfect transcription, calibrated
confidence, or general drawing-template accuracy is made.

## Implication for SiteWise

Use 150 DPI / PSM 6 as a candidate for the next integration experiment on this
template. Higher resolutions did not improve these four checks. The combined
625/723 ms exceeds the existing identity-read budget of 80/250 ms; it leaves
little headroom for the remaining work within the one-second p50 filing goal.
OCR should not be added to every upload. A background fallback for textless
files remains the safer speed design until a complete-path benchmark proves
otherwise. The server and existing classifications were not changed by this
benchmark.

## Reproduce

Run with Python containing `pypdfium2` and `Pillow` from the repository root:

```powershell
python tools/bench_ocr_titleblocks.py `
  --corpus '../Test Data/Bankstown Final Contract Set/07 Architecture' `
  --tesseract 'C:/Program Files/Tesseract-OCR/tesseract.exe' `
  --tessdata .tools/tesseract/tessdata `
  --output .tools/tesseract/crop-bench-psm6 `
  --dpi 150 200 300 --psm 6 --repeats 3
```

The script saves each crop, raw OCR text, timings and evidence checks locally.
The repository file `2026-10-03-title-block-ocr-results.json` retains configuration,
hashes, all timings and checks without the private OCR text or rendered images.
