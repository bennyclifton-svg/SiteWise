# Proposal existence reuse and bounded source ownership lookup

Lane A/C, continuing the existing timing and provenance work. No change to
proposal eligibility, ranking, fingerprints, source ownership, dependencies
or runtime AI calls. Budgets remain edit 50/150 ms, rebuild 100/300 ms and
filing 1000/2000 ms (p50/p90).

Interface evaluation now shares the D-10 existence result for the same part
and target system within one call. The work items and evidence callbacks are
fixed inputs during that call. The map starts empty for every evaluation and
is not retained on the evaluator or catalogue, so edits and different projects
cannot inherit cached state. Existing interface-direction, unknown/absent,
replacement, different-part and compiled-output tests pass.

The synthetic compiled benchmark remained approximately 2.74 ms/op. The full
Spec Home diagnostic measured proposal preparation/evaluation at 23.549/26.887
ms, compared with 26.182/29.974 ms in the preceding upsert run. However, the
earlier profile-tail anomaly recurred: rebuild p90 reached 664.761 ms and
persisted proposal diagnostic p90 reached 759.708 ms. This failed run is kept
in `bench/results/2026-10-06-proposal-existence-reuse.json`.

An instrumented repeat, including refreshed statistics on the empty dedicated
test tables, did not reproduce the slow snapshot. No slow plan was captured
(`.tools/tail-plan.log`); the precise cause remains unproven. Tracing was removed.

An experiment returned passage ownership separately from location metadata
in the same SQL statement. It preserved correctness but measured edit
52.279/54.4 ms and persisted proposals 119.080/123.249 ms. It was rejected;
the result is retained in
`bench/results/2026-10-06-proposal-reuse-owner-streams.json`.

The retained snapshot query uses an exact-ID lateral ownership lookup with
`OFFSET 0`, matching its existing document/source lookup pattern. This keeps
the lookup parameterized by the selected passage rather than flattening it
into a broader join. It adds no result rows or database round trip. The Go
map remains keyed by both document and passage ID, and all source rows still
share one SQL snapshot. Missing and mismatched historical locators retain
their facts with blank source locations.

Store, works and HTTP suites passed for the separate-stream experiment
(`.tools/proposal-reuse-owner-stream-tests.log`: 28.301/1.399/13.040 seconds),
including provenance, stale-statistics and source-replacement coverage.
The final exact-ID lookup also passes the focused ownership and stale-statistics
tests (`.tools/proposal-reuse-owner-lookup-tests.log`). The structural query
change does not by itself prove the tail anomaly fixed; the original failure
and its uncertainty remain recorded.

The final uninstrumented full workload is
`bench/results/2026-10-06-proposal-reuse-owner-lookup.json` (20 files per round,
two rounds, 40 API samples, explicit proposal diagnostic). It measured:

| Path | p50 ms | p90 ms |
|---|---:|---:|
| Spec Home edit, without automatic proposals | 48.0 | 50.4 |
| Profile rebuild, without automatic proposals | 28.9 | 30.1 |
| Proposal preparation/evaluation | 23.139 | 25.510 |
| Rebuild plus explicit persisted proposals | 110.580 | 119.754 |
| Filing | 454.6 | 608.0 |

The command's ordinary-path gate passes. The persisted proposal median still
exceeds 100 ms and remains diagnostic, with repeated reads and a separate
transaction. Runtime generation is still disabled; neither integrated-edit
timing nor release readiness is established. The tail anomaly did not recur
in this run, but its exact prior cause remains unproven.

The final store, jobs and HTTP suites pass serially
(`.tools/proposal-reuse-owner-final-tests.log`), including reading crash
recovery and source-ownership checks. The works suite also passes with the
per-evaluation existence map. `git diff --check` passes; temporary tracing
and the rejected source-stream implementation are absent from runtime code.
