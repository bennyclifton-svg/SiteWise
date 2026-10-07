# WP-26: reuse identical predicate traces within a part

Lane A: reduce code-only proposal evaluation cost without changing eligibility,
reasons, ranking or fingerprints. Scope is identical catalogue predicates over
one part's fixed inputs. No new dependency, schema, runtime AI call or automatic
proposal wiring. Budgets remain edit 50/150 ms, rebuild 100/300 ms and filing
1000/2000 ms (p50/p90).

A callback-answer cache showed no clear benefit: proposal computation measured
22.937 ms versus 23.139 ms previously. It was removed. Its diagnostic remains
in `bench/results/2026-10-06-proposal-evidence-reuse.json`; passing tests alone
were not treated as sufficient reason to retain the extra state.

The retained evaluator computes immutable predicate identities when loading
the catalogue, using the existing canonical record-hash helper. Only identities
used by multiple CQ/UC records are retained. During an evaluation, identical
conditions reuse their complete trace within the same part. Another part or
another evaluation starts with an empty trace map. Record-specific reasons,
signals, severity and fingerprint construction still run separately. Unique
conditions follow the existing path, avoiding unnecessary trace retention.

The uncached evaluation entry point remains the output reference. Tests compare
complete results across work-item changes, different parts and evidence changing
between unknown, true and false. They also verify the caller's evidence callback
remains unchanged. Existing direction, relevance, ranking and fingerprint tests
pass. The synthetic compiled benchmark measured 2.496 ms/op, 1,142,042 bytes/op
and 10,667 allocations/op. Caching every trace had used 1,396,513 bytes/op;
retaining only shared conditions removes most of that extra memory.

The startup benchmark, including catalogue load and evaluator construction,
averaged 180.317 ms over six iterations. This is a local mean, not target-VPS
or release evidence (`.tools/proposal-shared-predicate-tests.log`).

## Frozen workload

`bench/results/2026-10-06-proposal-shared-predicates.json` uses the unchanged
20-files/two-rounds/40-API-samples workload with explicit proposal diagnostics:

| Path | p50 ms | p90 ms |
|---|---:|---:|
| Spec Home edit, without automatic proposals | 49.6 | 51.4 |
| Profile rebuild, without automatic proposals | 28.8 | 30.4 |
| Proposal preparation/evaluation | 20.353 | 21.429 |
| Rebuild plus explicit persisted proposals | 107.272 | 116.640 |
| Filing | 471.9 | 643.0 |

The ordinary-path gate passes. The persisted proposal median still exceeds
100 ms; it is a separate two-transaction diagnostic, not integrated-edit timing.
Runtime generation remains disabled. Missing CQ/UC target/action semantics,
owner-reviewed quality keys and live/release gates remain open. No earlier
tail anomaly is claimed resolved by this computation change.

Works, knowledge, store and HTTP suites pass serially
(`.tools/proposal-shared-predicate-broad-tests.log`): 1.668, 3.418, 27.606 and
13.587 seconds respectively. These include proposal projection retention,
trigger replacement, decision reopening and rollback coverage. `git diff --check`
passes. The rejected callback-answer cache is absent from runtime code.
