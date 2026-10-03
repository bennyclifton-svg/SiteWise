# Automatic OCR for textless uploads

Lane A. Outcome: a newly uploaded PDF with no readable identity text receives
one automatic background OCR pass, then the existing filing decision policy.
The owner approved background processing and explicit visual progress on
3 October 2026. Ordinary filing keeps its p50 1000 ms / p90 2000 ms budget.

## Flow and boundaries

The upload runner tries native text extraction first. Only a PDF whose identity
text has `TextLayer=false` enters the durable `ocr` job queue. Text PDFs, DOCX,
XLSX, malformed containers and unsupported files do not invoke OCR. The upload
request does not wait. The queue has its own single worker, separate from text
preparation and project-profile reading. Leases recover interrupted work after
a restart. An ordinary failure finishes the pass with a specific reason;
there are no automatic repeated OCR attempts on an unreadable file.

Progress is persisted and sent over SSE: queued, reading lettering, classifying.
Completion says that recovered fields need checking; no estimated percentage.
Errors preserve the original and expose Retry OCR. Retry is explicit, scoped to
the current org, and also supports older `no_text_layer` uploads. Restarting does
not bulk-process old failures. The database keeps user corrections throughout.

Native PDFium renders at 150 DPI and Tesseract transcribes English text. Small
pages are read whole; pages longer than 1000 points use generic top, right and
bottom identity bands. This is first-page identity recovery, not full-document
searchable OCR. A title in an unexamined region can remain unresolved. Scanned
multi-page drawing packs remain marked for sheet-processing review. Neither
original PDFs nor their text layers are modified.

Code preserves OCR word coordinates and harvests candidates. Jev remains the
classifier, using [one fan-out](https://docs.typesafe.ai/patterns/fan-out),
[pre-parsed candidate selection](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and existing [confidence thresholds](https://docs.typesafe.ai/confidence).
All automatic OCR fields are capped at amber, including rule-derived fields.
OCR never creates supersession links. Missing/uncertain fields stay blank or
grey; code does not repair misread letters into expected drawing numbers.

## Runtime dependencies and limits

No new Go dependency, service or AI provider. OCR additionally requires:

- Tesseract with English language data (`SITEWISE_TESSERACT`, default `tesseract`).
- Python with pypdfium2 and Pillow (`SITEWISE_OCR_PYTHON`, default `python`).
- Optional `SITEWISE_TESSDATA` for a selected language-data directory.

The renderer is embedded in the Go binary. Native PDF rendering and word
geometry are non-trivial; reuse the already installed trial dependencies.
`tools/dev.ps1` discovers the existing Windows Tesseract install, trial fast
English data and bundled Python. Other deployments must configure the paths.
Missing dependencies leave text PDFs working and show OCR-unavailable on scans.
Validated locally: Tesseract 5.4.0, pypdfium2 5.13.0, Pillow 12.3.0.

Bounds: 32 MiB source, 25 million first-page pixels, 30-second combined subprocess
deadline, 2 MiB helper output, at most 400 retained identity runs. Temporary
images are removed on every return. Child processes use argument arrays, never
a shell; file paths come from the org-scoped content-addressed store.

## Speed gate

Background OCR filing target: p50 <= 10000 ms, p90 <= 20000 ms per file on an
otherwise idle queue. Burst waiting is shown as queued and can exceed this;
one OCR worker protects interactive filing capacity. `tools/check-ocr.ps1`
fails for missing dependencies, failed recovery or exceeded timing budgets.
It is invoked by `tools/check.ps1` and uses a public image-only drawing fixture.
Pass `-Pdf <path>` to measure a private drawing. No fixture is sent to a live
provider by this gate.

The gate runs 20 real upload-runner/queue/OCR/filing/database cycles. It replays
350 ms of Jev latency and includes a 2000 ms worker-poll allowance. It excludes
upload transfer, browser paint and burst queue waiting. It is not evidence of
live-provider latency. On the reported CC-06 drawing: p50 **6235 ms**, p90
**6512 ms**, against 10000/20000 ms. The OCR extraction smoke took 3333 ms.

Regression checks cover skipping OCR for readable PDFs, org isolation, durable
lease recovery, user corrections, review-only decisions, missing tools, empty
OCR, bounded inputs, explicit retry and live progress presentation.

`go test ./...` passes. The web build and OCR desktop/mobile browser test pass.
The full existing intake replay and speed gates could not complete: the current
corpus has 129 unmatched accuracy requests and 40 unmatched speed requests in
its stored Jev recordings. No recordings or thresholds were fabricated or
changed for this feature. Those full-corpus checks remain unverified.
