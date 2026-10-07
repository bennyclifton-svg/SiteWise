# Remaining profile and proposal timing

The profile-status experiment combined job and readable-document counts into
one statement with a shared document CTE. Store and HTTP suites passed
(`.tools/profile-status-batch-tests.log`, 26.860/13.310 seconds). The unchanged
40-sample workload measured edit 57.032/60.5 ms and rebuild 34.0/35.9 ms in
`bench/results/2026-10-06-profile-status-batch.json`. The unchanged rebuild
also slowed, so this run does not isolate the query's contribution; it does
not establish an improvement. The query change was reverted. The stronger
status test remains: multiple queued stages on one document must still count
as one pending document. It passes independently.

## WP-26 diagnostic localization

The existing compiled synthetic evaluator benchmark measured 2.727 ms/op,
1,113,510 bytes/op and 13,692 allocations/op. Its CPU profile is diagnostic
only: `.tools/proposal-evaluator.cpu`, with text output in
`.tools/proposal-evaluator-profile.log` and `.tools/proposal-evaluator-top.log`.
It is not representative of the full Spec Home input size.

An unchanged full workload with `-measure-proposals` and temporary stage
timing measured the explicit operation on real Spec Home inputs. Typical
stage observations were input reads 11–12 ms, preparation 2–3 ms, evaluation
24–30 ms, projection writes 37–42 ms, and commit 11–18 ms. The raw trace is
`.tools/proposal-stages.log`. The 40-sample diagnostic p50/p90 values are:

| Diagnostic | p50 ms | p90 ms |
|---|---:|---:|
| Proposal input preparation and evaluation | 26.177 | 32.351 |
| Rebuild plus dry evaluation | 69.663 | 76.222 |
| Rebuild plus explicit persisted proposals | 127.172 | 135.349 |

The persisted measurement still repeats snapshot/reconciliation and uses a
second transaction; it is not integrated-edit timing. The command's ordinary
path gate passed in this diagnostic run (edit 49.9/52.0 ms, rebuild 29.4/31.3
ms), but proposal measurements are explicitly diagnostic and do not make its
127 ms persisted median pass the 100 ms budget. Instrumentation and run
variation also mean this is not a substitute for an uninstrumented baseline.

The next targeted opportunity is avoiding unnecessary projection replacement
when proposal content and trigger relationships are unchanged. Any change must
preserve all fields, decisions, fingerprints, removals, atomic rollback and
org/project isolation; comparing semantic fingerprints alone is insufficient
because split work items can retain those fingerprints while trigger IDs
change. No such persistence change is implemented in this evidence record.

All temporary tracing has been removed. Ordinary profile edits still do not
invoke proposal generation. Missing explicit CQ/UC target/action semantics,
owner-reviewed quality keys and the remaining live/release gates stay open.
No dependency, schema, runtime AI call or wider feature was added in this
iteration.
