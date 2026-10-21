# Proposal fee extraction and repository recovery

19 October 2026. Lane A, draft evidence. The outcome is to show supported fees
from scanned and mixed PDFs in the existing comparison, leave unresolved scope
unknown, and show a proposal upload in Documents without a reload. This improves
answer trust and time to a procurement decision. No new AI, service, dependency,
award, budget or profile workflow is introduced.

## Cause and change

The identity reader's first-page OCR could finish filing without invoking the
proposal reader. The proposal reader then reopened native text only, so an image
appendix in an otherwise readable PDF was also lost. The worker now invokes the
same post-filing hook, including a terminal failed filing, and the proposal reader
supplements pages with missing text using the existing PDFium/Tesseract runtime.
It retains physical page numbers and the full bounded OCR text for fee extraction.

OCR is limited to four missing pages, 32 MiB input, 12 million rendered pixels
per page, two OCR workers, and bounded output. Successful extraction is cached
beside the content-addressed source. Remaining unread pages and OCR failures
produce an incomplete-text flag. OCR prices require review; handwriting still
needs visual checking. The original PDF is never rewritten.

Money parsing now handles space-grouped thousands and GST columns. Column-local
stage headings prevent facing-page prose, insurance limits and hourly-rate
headings from contaminating fee rows. A single quoted fee selected for multiple
schedule rows is counted once, with the other coverage left unresolved. Missing
base prices no longer produce a complete zero total. Prices and the comparison
report show the known subtotal and an incomplete-fee label.

An empty existing firm column can receive its proposal without creating another
column. The server validates the organisation, project, package and target before
storing bytes, then attaches the source transactionally. User prices and the
chosen firm survive a machine read; a new machine ABN cannot move that column to
another firm. An already attached source cannot be overwritten through this path.

Proposal uploads already used the document repository, but the open Project
view ignored events for documents it had not uploaded itself. It now fetches an
unseen document after its filing event, checks its project, and merges the latest
buffered event into the repository view. A browser regression checks this without
reloading the page.

## Jev contract

Code extracts candidates and computes money; Jev only selects from those
candidates in the existing single proposal fan-out. Question version, model and
confidence thresholds are unchanged. This follows TypeSafe's
[pre-parsed value extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[fan-out](https://docs.typesafe.ai/patterns/fan-out),
[confidence](https://docs.typesafe.ai/confidence) and
[jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13) guidance.

## Validation

- Real private PDFs reproduced missing scanned fees, space-grouped money and
  facing-column stage confusion. The repaired live comparison and repository
  were checked against the source pages. Source documents, amounts and recovery
  snapshots remain outside tracked evidence.
- Proposal, identity, intake, store, HTTP API and RFP Go suites pass with
  `TestProfilePlacesInputsInBuildOrder` excluded. Running without that exclusion
  exposes an existing unrelated Profile failure: profile inputs have no home.
- Both procurement browser tests pass, including upload into an existing firm,
  immediate appearance in Documents, and the award/report/undo flow.
- TypeScript checking, Vite build and `git diff --check` pass.
- The complete `cmd/intake-bench` Go suite passes, including its database-backed
  end-to-end replay and the new event-stream deadline regression.
- Regressions cover dense OCR, physical page provenance, cached OCR, post-OCR
  hooks, money/GST parsing, stage headings, insurance/rate filtering, duplicate
  fee selection, unknown totals, organisation isolation and retained corrections.

## Timing and limits

The recorded budgets remain unchanged: proposal read p50/p90 3,000/6,000 ms,
price table 150/300 ms, submission write 100/250 ms, and filing 1,000/2,000 ms.

The three cold private recovery runs took 2,922, 4,984 and 7,375 ms from upload
to a non-reading column, measured by polling. This is only three observations,
includes polling overhead, and does **not** demonstrate the proposal-read budget.
The slowest scanned document exceeded six seconds. Cached OCR avoids repeating
that extraction work, but this is not a claim that cold scan latency is solved.

The replay benchmark previously lost its long-lived SSE connection at the HTTP
client's 60-second request deadline, then reported a proposal timeout. Its event
request now follows the caller's cancellation lifetime; normal HTTP requests keep
their timeout. A regression reproduces the old deadline and passes after the fix.

The development replay gate passed with 20 files, one round and 20 API samples,
using the dedicated benchmark database. The proposal fixture uses the existing
1,500 ms fallback replay delay when a private recording is absent. These results
measure the deterministic application paths and replayed provider latency, not
live Jev or cold scanned-PDF performance. This is not release evidence.

| Path | Samples | p50 ms | p90 ms | Budget p50/p90 ms |
|---|---:|---:|---:|---:|
| Proposal read | 40 | 1,793.7 | 1,969.7 | 3,000 / 6,000 |
| Price table | 20 | 6.0 | 6.4 | 150 / 300 |
| Submission write | 20 | 7.5 | 8.0 | 100 / 250 |
| Whole intake | 20 | 502.6 | 727.8 | 1,000 / 2,000 |

Command: `go run ./cmd/intake-bench -files 20 -rounds 1 -api-samples 20` with
output and raw samples saved under the ignored `tmp/hale-fee-diagnosis/` directory.
