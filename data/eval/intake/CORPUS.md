# Real-PDF intake regression loop

The corpus runner inventories every PDF in `corpus-roots.json`, hashes its
contents, records duplicate locations, excludes AppleDouble `._` metadata,
and uses the production PDF reader, candidate harvester, rules, question
builder and confidence routing. It does not copy PDF data into Git.

Run from the repository root with the local Go toolchain. All inventories,
labels, recordings and traces below are in the existing git-ignored private
directory. Keep them there: filenames and extracted text are private.

```powershell
# All documents: offline extraction and rule baseline.
go run ./cmd/corpus-eval -mode rules -per-folder 0 -out data/eval/intake/private/experiment-01

# Stable small sample: up to two unique PDFs per leaf folder.
go run ./cmd/corpus-eval -mode scan -per-folder 2 -out data/eval/intake/private/sample

# Live calls use the configured Jev key; this sends identity text to TypeSafe.
# Use a NEW output/recording path for each experiment. Existing logs are protected.
go run ./cmd/corpus-eval -mode live -per-folder 2 -gold data/eval/intake/private/corpus-gold.json -out data/eval/intake/private/experiment-02

# Experimental confidence calibration, using ONLY calibration-family labels.
go run ./cmd/corpus-eval -mode replay -per-folder 2 -gold data/eval/intake/private/corpus-gold.json -recordings data/eval/intake/private/experiment-02/recordings.jsonl -fit -out data/eval/intake/private/fit-02
# Replay with the experimental config; this does not overwrite production.
go run ./cmd/corpus-eval -mode replay -per-folder 2 -data data/eval/intake/private/fit-02/calibrated -gold data/eval/intake/private/corpus-gold.json -recordings data/eval/intake/private/experiment-02/recordings.jsonl -out data/eval/intake/private/fitted-02 -gate

# Replay EXACT requests. Changed code/state requires a fresh live recording.
go run ./cmd/corpus-eval -mode replay -per-folder 2 -gold data/eval/intake/private/corpus-gold.json -recordings data/eval/intake/private/experiment-02/recordings.jsonl -out data/eval/intake/private/replay-02 -gate

# Score a saved run without another provider call or PDF extraction.
go run ./cmd/corpus-eval -mode score -per-folder 0 -traces data/eval/intake/private/experiment-01/traces -gold data/eval/intake/private/corpus-gold.json -out data/eval/intake/private/score-01

# Actual local app: creates fresh QA projects, uploads the selected files,
# polls persisted document fields, and saves the exact results and project IDs.
# This invokes the app's configured provider, so live permission applies here too.
python tools/corpus-upload.py --selected data/eval/intake/private/sample/selected.json --out data/eval/intake/private/app-01 --allow-live-jev
# Use --resume with the same selection/output to continue interrupted uploads.
# HTTP 413 is recorded and the next file proceeds. Other HTTP failures stop.

# Compare persisted app fields with gold using the same field normalisation.
go run ./cmd/corpus-eval -mode app-score -per-folder 2 -app-results data/eval/intake/private/app-01/results.json -gold data/eval/intake/private/corpus-gold.json -out data/eval/intake/private/app-score-01 -gate

# Combined offline check: both gates run and any failure returns nonzero.
./tools/check-corpus.ps1 -Gold data/eval/intake/private/corpus-gold.json -Recordings data/eval/intake/private/experiment-02/recordings.jsonl -AppResults data/eval/intake/private/app-01/results.json -Out data/eval/intake/private/release-check
```

The draft runner isolates classification from persistence and does not exercise
supersession. The upload runner exercises the actual app and its database;
the existing `intake-bench` and Playwright suite cover complete-path latency,
SSE, org isolation and supersession. Neither draft latency nor replay latency
is proof of end-to-end production performance.

Run an updated server against a separate QA database and port (for example
8081) using `--base http://127.0.0.1:8081`. Two builds sharing a database can
consume each other's intake jobs and invalidate a comparison. An app run
records polling-observed filing time; it does not instrument extraction.

## Independent expectations

Gold is a JSON array keyed by the PDF's SHA256, not by its filename. Example:

```json
[
  {
    "SHA256": "<64 hexadecimal characters>",
    "Fields": {
      "number": {"Value": "A-101", "Evidence": "page 1, title block, Drawing Number"},
      "revision": {"Value": "A", "Evidence": "page 1, current issue row"}
    }
  }
]
```

Write expected values from independently inspected PDF pages or verified
register/transmittal entries **before inspecting predictions**. Never copy the
model output into gold. `Value: ""` explicitly means a field is absent;
an omitted field is unknown and is not scored. Use catalog IDs for kind,
discipline and lifecycle. Do not infer lifecycle from the storage folder:
document purpose and issue status need a written adjudication policy.
The owner has decided that technical drawings remain **Design**, including
drawings issued for construction. Issue status is not lifecycle filing purpose.

Scans or unreadable files can have `NotFiled` equal to the expected extraction
failure and an `Evidence` explanation, with no `Fields`. They are checked as
expected abstentions instead of requiring invented metadata. For example,
`"NotFiled": "no text layer on identity page"`. This does not claim OCR support.

The extended local gold has 332 cases: 246 with partial field annotations and
86 expected extraction abstentions. It includes 50 original register/transmittal
transcriptions plus independent page-text and visual checks across all three
corpora. Labels remain draft, not owner-reviewed. Another 206 PDFs have no gold
labels. Field coverage is reported separately; a partially labelled document
is not a completely verified document. A title mismatch can be an
answer-key defect: inspect the page before changing code to match an abbreviated
register title. Record corrections with page evidence, preserving original gold.

## Sampling and iteration

1. Run the complete offline census. Classify failures as unreadable/scanned,
   extraction truncation, absent candidate, incorrect rule, wrong Jev selection,
   rejected confidence, or provider/deadline failure.
2. Fix the seed and sample count for an experiment. Each leaf folder gets a
   deterministic hash-ranked sample, preventing large drawing packages from
   dominating. Increase `-per-folder` to broaden coverage; zero traverses every
   unique PDF. Audit `selected.json` for required disciplines and layout types.
3. Folder families stay entirely within calibration or held-out splits. Exact
   duplicates are never counted twice. This is a conservative split, not a
   guarantee against revisions stored in different folders: review family
   assignments before using the held-out set for a release claim.
4. Minimise one failure, add a test, watch it fail, change one cause, and rerun
   that test plus the frozen sample. Record hashes and preserve earlier runs.
5. Fit confidence thresholds only from independent calibration labels, separately
   for each question and option shape using `-fit` and `internal/eval.Fit`. The
   experimental config is written to `<out>/calibrated`, never production. Unknown shapes
   stay disabled. Never lower thresholds merely to make blanks disappear.
6. Evaluate the held-out families only after fitting. Check every field and
   corpus, plus wrong green decisions, and retain the full coverage denominator.
   A small calibration success is not evidence of general accuracy.
7. Upload the same sample to isolated QA projects. Compare persisted fields
   against gold; inspect differences from the draft trace (timeouts, stale
   server build, persistence or worker behaviour). Keep QA project IDs for review.
8. Before release, expand to every unique PDF and run the latency gates.

`summary.json` reports correct, blank, wrong, false-green and absent-candidate
counts per field, corpus, expected discipline and split. Accuracy gate defaults to at least 95%
correct/scored per field, complete labels on the selected set, zero wrong green
values, and no unexpected errors. It also enforces extraction p50/p90 of
80/250 ms and live draft p50/p90 of 1000/2000 ms. Unlabelled cases never count as
correct. `-gate` exits nonzero on failure; rules-only is always diagnostic.
Reports without `-gate` are experiments, not passing release evidence.
Add `-baseline <previous-summary.json>` to reject a drop in correct values,
an increase in false-green values, or a changed scoring population. Per-corpus
and per-split success rates must also meet the configured minimum; aggregate
success cannot hide a weak discipline folder's corpus.

## Jev contract

Code finds candidate values; Jev selects a literal or none, following
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
All questions, including an ambiguous issue date, share one
[fan-out](https://docs.typesafe.ai/patterns/fan-out).
The date question asks for the explicitly identified issue date, not date
ordering; that respects [Jev's limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
Confidence is calibrated per question and option shape per
[confidence](https://docs.typesafe.ai/confidence) and
[confidence routing](https://docs.typesafe.ai/patterns/confidence-routing).
Intake-12 includes spatial number/revision/title evidence, joined title lines,
bounded author context, explicit vocabulary rules, revision-paired dates and
the agreed lifecycle policy. Each state change receives a new question version
and fresh live recordings; previous-version confidence is revalidated rather
than silently transferred. Experimental fitting uses a 95% green lower bound,
80% amber lower bound, at least 30 answers per shape and 20 per band.

See [extended evidence](../../../docs/evidence/2026-10-02-corpus-intake/README.md)
for the validated cutoffs, measured population coverage and remaining failures.
A supported field cutoff does not imply the whole release gate passes.

## Audit details

Every scoring run writes a private `review.json` queue of blanks, wrong values,
wrong green values, status mismatches and completely unlabelled documents.
Each item points to its SHA256 trace; known reference alternatives are respected.

Every run freezes `gold-snapshot.json` and `run.json`, including the runner's
question version, model, seed, thresholds and SHA256 values for labels and
selection. In score mode the runner version is not proof of the historical
trace version; keep the source run's metadata alongside the score.

`Alternatives` can contain additional literal values for a labelled field,
each with its own source evidence. This handles a register abbreviation versus
a full printed sheet title; it is not fuzzy matching or a way to approve a
prediction. Preserve prior gold versions and record source conflicts. Do not
label an ambiguous source field as known merely to increase coverage.

Raw Jev-choice counts are diagnostic and separate from applied-field accuracy.
Provider failures block calibration. Expected textless/oversized files are
excluded from calibration but remain in status accounting. The app's HTTP 413
upload rejection is distinct from the identity reader's 32 MiB limit; the
current status gate reports that mismatch rather than hiding it.

Repeatedly inspecting validation-family failures makes these development
validation sets, not an untouched release holdout. Acquire a new independently
reviewed holdout before a general accuracy claim. Sparse Newham/Bankstown
labels and related drawing templates limit the current estimates.

The local combined gate accepts `-Data <config-directory>` and `-PerFolder 0`
for a full-corpus candidate check. Kind can have an evidence-backed amber
cutoff with `green: null`; even confidence 1 then stays a review suggestion.
An empty/uncalibrated cutoff or unknown option shape still cannot apply values.
