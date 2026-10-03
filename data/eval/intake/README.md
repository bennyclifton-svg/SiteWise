# Intake evaluation

For the complete Petersham, Newham and Bankstown PDF census, deterministic
sampling, extraction traces and actual-app upload loop, see [CORPUS.md](CORPUS.md).
Intake-12 requires fresh recordings and calibration; earlier reports remain
historical evidence, not a passing baseline for the current code.

`manifest.json` is the evaluation contract for intake filing. It pins the
Hale, Petersham and Newham answer keys (read from the frozen `../clerk`
checkout) by SHA-256 and names the corpus folders under `../Test Data`. It
holds no labels, titles or document text: this repository is public.

What never enters Git (`private/` is ignored):

- `private/recordings.jsonl` — every exact Jev request and response from a
  live run. Requests contain identity text from private documents.
- `private/cases-*.json` — per-case decisions and labels.

What is committed: `results/*.json` (counts, hashes and aggregate metrics),
`results/baseline.json` once accepted, and `data/intake/thresholds.json`.

## Order of work

```powershell
# 1. Verify corpus and key hashes; print label counts per split. No Jev call.
go run ./cmd/intake-eval -check

# 2. One live pass over the corpus, recording every exchange (paid).
$env:SITEWISE_JEV_API_KEY = '...'
go run ./cmd/intake-eval -live

# 3. Fit cut-offs on the calibration split and write data/intake/thresholds.json.
go run ./cmd/intake-eval -replay -fit

# 4. Review results/replay-latest.json, then accept the held-out baseline.
go run ./cmd/intake-eval -replay -accept

# 5. From then on, the deterministic gate (also in tools/check.ps1):
go run ./cmd/intake-eval -replay

# Latency, replaying recorded provider latency per document:
go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json
# Release evidence, on the intended VPS:
go run ./cmd/intake-bench -live -target vps -manifest data/eval/intake/manifest.json -budgets bench/budgets.json
```

`-rules-only` files the corpus with Jev unavailable: every question goes grey
and only deterministic rules decide. It measures rule accuracy and false
confident rule actions; it is diagnostic, not the gate.

## How it is scored

- Only the identity page is filed, as in production, so page-level key
  entries after page 1 are excluded. Byte-identical files are one filing.
- Splits are by family: cases sharing a folder or a document number series
  are joined, then the family is assigned by salted hash. Revisions of one
  drawing never straddle calibration and held-out.
- Authoritative labels are `award-doc`/`owner` number, revision, title and
  date. Discipline and kind are model or folder judgements in every key and
  are reported as diagnostic only. All keys are `adjudication: unreviewed`.
- Each key flag removes the fields it makes unreliable (see `flags`); an
  unclassified flag fails the run.
- Supersession truth is derived from the keys: the latest earlier revision of
  the same number in the same corpus and split, or new. Unknown when code
  cannot order the revisions.
- Cut-offs are fitted per question and power-of-two option shape, on
  calibration answers only, using the Wilson lower bounds in `fit`. A shape
  without enough answers stays unknown, which disables automatic action.
  Supersession needs zero errors above its cut-off.

The gate fails on a missing corpus, key or recording, a replay miss, too few
held-out samples, any incorrect or unverifiable automatic supersession, a
field that lost correct values or gained false confident ones against the
baseline, or no accepted baseline.
