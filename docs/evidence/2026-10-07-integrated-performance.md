# Integrated proposal and M2/M3 performance work — 7 October 2026

This records local Windows/PostgreSQL measurements. It is not target-VPS release
proof or a fresh live Jev measurement. Original budgets are unchanged.

## Final measured status

The complete prescribed developer benchmark **passes**, with automatic proposals
integrated into ordinary edits. The final unaltered result is
[`2026-10-07-m2-m3-final.json`](../../bench/results/2026-10-07-m2-m3-final.json).
It used 184 files � two rounds, concurrency four, two background callers
(130 calls), 10% degraded requests (55 stalled), and forty API samples.

| Path | p50 / p90 ms | Budget ms |
|---|---:|---:|
| Spec Home profile edit | 47.6 / 49.9 | 50 / 150 |
| Ordinary profile edit | 33.3 / 100.8 | 50 / 150 |
| Profile rebuild | 22.3 / 24.9 | 100 / 300 |
| Work write, including layout corrections | 39.2 / 81.6 | 100 / 250 |
| Split work | 13.5 / 16.6 | 100 / 250 |
| Retire work | 9.9 / 13.1 | 100 / 250 |
| Cost read | 2.7 / 3.9 | 100 / 250 |
| Cost write | 5.4 / 7.5 | 100 / 250 |
| Issue report | 17.4 / 19.3 | 100 / 250 |
| Download issued report | 1.0 / 1.1 | 100 / 250 |
| Whole filing, round one | 334.4 / 720.5 | 1,000 / 2,000 |
| Whole filing, round two | 350.2 / 691.8 | 1,000 / 2,000 |

Identity extraction p90 (301.1 ms) and deterministic field-rule p90 (4.0 ms)
remain reported developer-host components judged on the target VPS under the
existing benchmark policy. This pass does not close target-host or fresh live
Jev accuracy gates. Earlier failed results below are retained as diagnostic
history, not the status of the final implementation.

## Reproduction and diagnosis

The isolated complete benchmark used the committed 184-file manifest, two
rounds, concurrency four, two background workers, degradation every tenth model
request and forty API samples. Its recorded model latency is replayed.
`bench/results/2026-10-07-m2-m3-isolated-before.json` records the result before
automatic proposal integration: whole filing 351.3/942.9 ms p50/p90, ordinary
profile edits 32.3/55.0 ms, and Spec Home edits 52.935/60.0 ms. The Spec Home
median failed its unchanged 50 ms budget. Work split 7.3/8.8 ms, retire 6.9/8.1 ms,
report issue 15.9/17.9 ms and export 1.0/1.4 ms passed 100/250 ms.

A CPU sample of automatic integration identified repeated ancestry checks in
proposal evaluation. The immutable loaded catalogue now indexes ancestor pairs;
an all-pairs regression compares that index with the previous tree traversal.
The subsequent reduced filing diagnostic, with the full frozen Spec Home fixture
and twenty API samples, still failed: ordinary edit median 56.878 ms and Spec
Home median 103.820 ms. This is recorded in
`bench/results/2026-10-07-m2-m3-ancestry-diagnostic.json`. It established that
enabling automatic evaluation without further work was not acceptable.

## Changes and correctness

Profile writes now compile an immutable evaluator at process wiring and rebuild
proposals in the same transaction. A persisted hash of complete evaluator inputs
permits unchanged inputs to reuse the saved proposal projection. The hash includes
all work records except edit clocks, effective part determinants/work types,
signal states and citations, existing/present tri-states for every loaded system,
and the evaluator's catalogue/record version. It retains no project data in a
process-wide cache. Project locking and transactional stamp writes prevent a
failed rebuild from advancing it. Explicit diagnostic rebuilds and decision undo
invalidate the stamp. Undo immediately restores the evidence-addressed base state.

Pure tests cover semantic invalidation and audit-only reuse. Database tests prove
unchanged inputs issue no proposal projection statements, explicit rebuilds
invalidate reuse, changed actions refresh proposals, and failed evaluation or
persistence rolls back both projections and events. All passed in the targeted
7 October run. The complete works/profile/knowledge suites also passed.

Benchmark coverage adds separate split, retirement, cost read/write, report issue
and saved PDF download samples. Each has 100/250 ms budgets and sample-count
regressions; fast creates cannot hide slow subdivision. The benchmark regression
and exact committed-budget inventory tests passed after these additions.

## Remaining measurement

The input-reuse diagnostic is recorded in
`bench/results/2026-10-07-m2-m3-input-reuse-diagnostic.json`: ordinary edits
26.9/92.7 ms and unchanged Spec Home rebuilds 34.8/37.8 ms pass. Changed-subclass
Spec Home edits still fail at 114.367/122.1 ms, because these genuinely change
the proposal inputs. Split 15.6/17.0 ms, retirement 10.4/15.1 ms, report issue
17.0/19.2 ms and saved export 1.0/1.2 ms pass. This twenty-file, one-round run is
a diagnostic, not the complete filing-under-load gate.

Automatic integration is still undergoing timing acceptance. The complete gate
must pass after the reuse change and final catalogue changes. Component limits
remain reported locally and judged on the target VPS under D-37. No reported
component failure, live-evidence gap or failed edit median is treated as a pass.

## Further measured reductions (not final acceptance)

Stage timings identified large persisted global-rank updates and repeated trigger
membership writes. Migration 021i replaces stored rank with the project-scoped
`ranked_proposals` view, using the exact severity/specificity/key ordering and C
collation. Database tests compare it with Go ranking across Unicode ties,
state changes, removal, and separate projects. Every read and saved snapshot
uses the view. Migration 021g stores a complete projection payload hash; unchanged
rows and unchanged trigger membership avoid writes, while removed keys still
retire transactionally. Decision state is compared separately because acceptance
and dismissal update it directly. No decision or provenance field is discarded.

Materialized evaluator inputs share resolved state with evaluation. Decision and
projection indexes are read on the transaction goroutine while pure evaluation
runs; no transaction is used concurrently. Primitive predicate matches are reused only within one immutable evaluation. Catalogue
children and ancestry indexes retain the original ordering and matching semantics.
String predicate inputs bypass repeated general-purpose formatting. Equivalence
and unknown/layout-input tests pass.

Migration 021h's validated passage-source document foreign key allows snapshot
provenance to read its document identity directly. Stage snapshot time falls from
about 11 ms to about 9 ms; source metadata remains in the same snapshot statement.
The separate source-coverage evidence documents its index change. The normal
20-sample diagnostic before string fast paths still failed Spec Home at 67.306 ms;
all other developer-gated paths passed. Instrumented runs are attribution only,
not gate results. Temporary instrumentation has been removed. Final integrated
measurement remains outstanding at this checkpoint.

The interface-match map and pre-read work-sync experiments were rejected: the
former did not show a timing gain, while the latter doubled work reads on genuine
subclass changes (about 5.5 to 8 ms). They are not retained. Materialized inputs
now allow independent interface and consequence evaluation to overlap; arbitrary
caller callbacks remain sequential. Eight concurrent evaluations match complete
sequential output. Pure works/knowledge/profile tests pass. `go test -race` could
not run on this Windows host: cgo is disabled, and enabling it reports missing
`gcc`; no compiler or dependency was installed.

The profile delta projection introduced separately in migration 021j lowers
profile SQL from about 11–13.5 ms to about 9.3–10.4 ms on changed inputs. Combined
with parallel pure evaluation (about 5.8–8.8 ms), a normal twenty-sample diagnostic
now records Spec Home at 61.841 ms p50: improved, but still failing 50 ms. This is
not acceptance. All other developer-gated paths passed that diagnostic. Further
attribution and the full 184-file/two-round gate remain outstanding.

Per-call relevance membership reuse reduced the next normal twenty-sample median
to 55.326 ms. Subsequent combined build-state lookup, exact removed-row deletion
and typed proposal encoding measured 60.767 ms on another run, demonstrating
remaining host/workload variation rather than an accepted gate. No result was
selected or relabelled as passing. Rebuilds now reuse snapshot parts rather than
issuing a duplicate whole-part lookup; tests retain whole-first initialization,
existing part identities, and wrong-org rejection. CPU and exact batched proposal
phase attribution continue before final acceptance.

## Final implementation and acceptance run

Migration 021j hashes every complete persisted profile row and upserts only changed
rows, deleting exact removed keys. Independent review and tests cover provenance,
org isolation, constraint failure rollback, and legacy empty hashes. The real
reader subprocess-crash recovery test passes (4.135 s). Build state and current
revision counters now use one query in the same read snapshot; initialization,
concurrency, freshness, roundtrip, org isolation and rollback tests pass (1.613 s).

A successful single CPU-profile collection found GC in 31% of samples and the
profile fingerprint in 12%. The new `profile-input-v2` representation hashes each
complete canonical input row and sorts those digests before hashing the complete
snapshot. Duplicates remain represented; no evidence, source field, decision input
or dependency is removed. It avoids quoting and copying complete row JSON a second
time. Input-class, ordering, duplicates, invalid JSON, atomicity and rollback
regressions pass (1.512 s). The fingerprint format changes once on the next rebuild.

The final reduced diagnostic passes every developer gate: Spec Home 49.6/55.1 ms
against 50/150 ms, ordinary edits 22.8/88.4 ms, and rebuilds 24.4/25.2 ms.
`bench/results/2026-10-07-m2-m3-preflight-pass.json` preserves this twenty-file,
one-round, twenty-API-sample result. The complete original workload is now being
run separately; this diagnostic is not substituted for it.

The first complete prescribed workload **failed** Spec Home p50 at 52.565 ms
(p90 56.8 ms), preserving the strict 50/150 ms budget. Its unaltered result is
`bench/results/2026-10-07-m2-m3-full-failed-1.json`. Whole filing passed in both
184-file rounds (331.5/682.3 and 356.9/963.0 ms). Split passed 14.4/16.8 ms,
retirement 10.8/13.9 ms, report issue 15.8/16.7 ms, and saved export 0.7/1.1 ms.
Identity extraction p90 and deterministic-rule components remain target-VPS
reports, as prescribed. No selective rerun or relaxed budget is used to convert
the failed profile median into acceptance; further implementation work is required.

After migration 021k replaced passage-ID array probes with validated, indexed
call document identity, coverage measured 4.180/4.791 ms (previously
6.308/6.810 ms). Backfill, compatibility, foreign-document/org rejection,
label-stage exclusion, stale statistics and populated upgrade tests passed.
User-value cleanup/upsert now share a batch after unchanged lock/version checks;
trigger-link deletion narrows candidates to its exact changed proposal keys.

The second complete prescribed workload still **failed**: Spec Home
50.089/55.4 ms against 50/150 ms. Its result is preserved as
`bench/results/2026-10-07-m2-m3-full-failed-2.json`. Ordinary edits passed
30.9/95.6 ms and rebuilds 24.1/26.0 ms. Split passed 13.8/16.2 ms,
retirement 11.0/14.6 ms, report issue 15.0/16.9 ms, and export 1.1/1.1 ms.
Whole filing passed both full rounds at 325.7/716.1 and 358.0/805.0 ms.
The 89-microsecond median overrun is still a failure. No budget tolerance or
selective rerun was applied. Independent response metadata SELECTs are now
batched within the same repeatable-read snapshot; correctness and a new complete
measurement remain required for that further implementation change.

The response-metadata batching experiment regressed the complete workload to
Spec Home p50 64.470 ms. It is preserved in
`bench/results/2026-10-07-m2-m3-full-failed-3.json` and has been reverted.
No transaction or response semantic change from that experiment remains.

Audited private helpers now reuse the transaction's existing project advisory
lock for user-value edits, profile revision changes, projection rebuild and its
work revision bump. Standalone/public entry points still acquire the lock.
Atomicity, concurrency, org isolation and rollback tests pass. A normal
20-sample diagnostic still fails at 51.196 ms; this is not acceptance.
Bounded attribution identified response readability at 4.4–4.7 ms and duplicated
snapshot/fingerprint document metadata as remaining targets. These are being
consolidated without dropping input or relaxing the gate.


Final gains retain exact behavior: readability evaluates the existing predicate
once per scoped document and reuses it for counts and skipped-kind selection;
policy, missing-kind, explicit override, tie ordering, wrong-org, status and
stale-statistics tests pass. Snapshot/fingerprint document metadata is collected
once, including no-fact, skipped and superseded documents; the old-query oracle
confirms byte-for-byte document JSON and unchanged complete fingerprints.
The first profile reconciliation now computes only headers needed to choose
scope; the final full reconciliation remains unchanged. Fifty oracle cases and
the entire profile test package pass. The final reduced check was 49.7/52.9 ms;
the subsequent complete run above passed 47.6/49.9 ms. Temporary instrumentation
and the rejected response batching experiment are removed. No new dependency
or budget change was introduced.
