# Single-transaction proposal rebuild diagnostic

Lane A, WP-26 step 0. The outcome is a faithful measurement of profile and
proposal projection in one transaction, using the same snapshot and reconciled
work items. This removes duplicate reads from the previous two-transaction
diagnostic without enabling automatic generation. Budgets remain edit p50/p90
50/150 ms and rebuild 100/300 ms. No new dependency, service or Jev call.

`Store.RebuildProfileWithProposals` validates the configured catalogue and
evaluator, acquires the existing project lock, and runs the existing profile
projection with proposal evaluation and persistence before its build/event
commit. Ordinary profile edits still use the original path without an evaluator.
`intake-bench -measure-proposals -integrated-proposals` measures this complete
transaction. It rejects missing diagnostic opt-in and release mode. Its edit
requests remain ordinary; their numbers cannot establish integrated-edit speed.

The PostgreSQL-backed integration test compares complete proposals, profile
rows and fingerprints against separate rebuilds. It checks revision advancement,
wrong-organisation rejection and a missing evaluator. A proposal persistence
failure after profile row replacement rolls back both projections, the build
revision/timestamp and events. Focused tests passed before measurement.

The uninstrumented result is
`bench/results/2026-10-06-proposal-integrated-rebuild.json`, using 20 files per
round, two rounds and 40 API samples. The combined rebuild measured p50 95.414 ms
and p90 708.451 ms: **failed**. Its first five samples were 708–744 ms. Ordinary
Spec Home edits measured 47.262/53.819 ms, with one 690.605 ms maximum. This
recurrence means the earlier ownership lookup change did not prove the tail
anomaly fixed.

Temporary stage timing distinguished snapshot reads, fingerprint inputs,
computation, work reconciliation, profile projection/hash, proposal evaluation
and proposal persistence. The first instrumented repeat passed rebuild at
96.0/102.6 ms; none of its stages exceeded 100 ms. A diagnostic connection using
`plan_cache_mode=force_custom_plan` slowed rebuild to 115.872/122.0 ms and did not
reproduce the 700 ms tail. This setting was not retained. A second ordinary-plan
instrumented repeat measured rebuild 97.2/101.7 ms but failed ordinary Spec Home
edit median at 50.746 ms. No slow stage was captured. These experiments do not
identify the cause, and passing repeats do not supersede the failing sample.

Temporary tracing is removed. Automatic proposal generation remains off.
Integrated edit timing, 0991 quality review, missing explicit CQ/UC semantics,
and live/release gates remain outstanding; this diagnostic is not step-0
completion or release evidence.

After removing instrumentation, the complete PostgreSQL-backed Go suite passes
with `go test -p 1 ./...` (log: `.tools/integrated-proposals-all-tests.log`).
`git diff --check` passes with the existing line-ending warnings. No claim of
race-detector or target-VPS validation is made by this local run.
