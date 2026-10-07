# M2/M3 integrated validation — 7 October 2026

This is the final implementation validation record. Read
[NEXT-WAVE-STATUS](../plans/NEXT-WAVE-STATUS.md) for the owner-facing summary.
The original plan's acceptance criteria and budgets were retained. All private
corpora, extracted passages, recordings and machine-local logs remain ignored.

## Regression

- `tools/check.ps1` was run against the implementation. The first run found an
  old-schema migration fixture missing its required stored rank. Restoring that
  historical fixture value fixed it; the final schema still derives live rank.
- The subsequent complete `go test -p 1 ./...` passed, including API, tenant/FK,
  concurrency, projection rollback, crash recovery and report/export tests.
- SPA install/type-check/build passed. Strict knowledge validation passed with
  zero errors and the two existing unmapped-default warnings.
- All 33 Python tool tests passed. Both batch1 and batch2 triage checks passed.
- Installed OCR gate passed: 20 filings p50/p90 3192/3223 ms and 20 detail
  recoveries 3189/3196 ms, against 10000/20000 ms. These include polling allowance
  and recorded provider timing, as prescribed by that gate.
- The first final browser run passed 28/29. Its layout test expected an
  inapplicable field after changing work to repair. The corrected assertion
  checks the saved action, absence of that field and the stored unknown value.
  The complete browser rerun passed all 29 cases in 1.2 minutes with clean exit.
- Intake replay passed: 184 cases, 218 Jev calls, zero errors and zero replay
  misses. `data/eval/intake/results/replay-latest.json` records the run.
- Source and Hale replay were both run separately after the browser correction.
  Both fail closed on stale recordings (`bankstown-page9`, `hale-description`).
  The earlier automatic approval rejections prevented refreshing private source
  payloads; no bypass was attempted. No overall green full gate is claimed.
- Final-schema restore passed all four checks: 96 table/tenant digests, 115
  validated FKs and immutable issued snapshot/PDF links preserved. Populated
  pre-M2 migration rehearsal passed separately: 31 baseline plus 12 new
  migrations, with all 42 original table counts and normalized digests retained.

Logs are local: `tmp/m2-m3-check-first.log`, `tmp/m2-m3-check.log`,
`tmp/m2-m3-browser-final.log`, `tmp/m2-m3-intake-replay.log`,
`tmp/m2-m3-source-replay.log` and `tmp/m2-m3-hale-replay.log`.

The full command is fail-closed. A passing subset is not a passing full gate.
Current-question source/Hale replay and live accuracy remain separate from
deterministic application tests; see [evaluation limitations](2026-10-07-evaluation.md).

## Implementation audit and publication boundaries

An independent agent reread the original implementation plan, requirements and
final costs, delivery, reports, UI and knowledge changes. It found no additional
confirmed M2/M3 implementation omission. This does not supply owner-reviewed
keys/clauses, K5 experience, K6 usefulness/manual-time evidence, statutory project
facts, target-VPS timing or off-site restore proof.

Publishable paths contain no private/local credential paths; a token/private-key
pattern scan found no candidates. The new direct dependencies are the pure-Go
PDF writer and bundled Go fonts, preserving deterministic offline rendering in
the existing single binary without a browser service or host font dependency.

Final frozen-code tools/check.ps1 rerun: all Go packages, OCR, all 29 browser cases (clean exit), and intake replay pass; the command exits 1 at stale source profile replay. Hale separately exits 1 stale. The complete prescribed benchmark passes all local user paths; see [performance evidence](2026-10-07-integrated-performance.md). Final restore recheck passes in 5.864 s (dump/restore 4.17 s), preserving the same 96 digests and 115 FKs. No code changed afterward; two test files were gofmt-normalized only.
