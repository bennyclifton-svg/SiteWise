# Profile edit: source coverage cost

Lane A/C: reduce profile-edit response time while preserving source coverage,
tenant isolation and the initial whole-project part. Scope is the measured
read path, with no new service, dependency, runtime AI call or precomputed
counter. The unchanged full-path budgets are edit p50/p90 50/150 ms, rebuild
100/300 ms and filing 1000/2000 ms.

The preceding uninstrumented frozen Spec Home run measured edit 52.9/56.7 ms
(`bench/results/2026-10-06-fact-scope-repeat.json`). Temporary stage timing
showed coverage taking approximately 11–12 ms of the response read; row
decoding took approximately 2 ms. A captured query plan measured 10.704 ms
for coverage, approximately 8 ms of which read passage outcomes. Evidence
call counting took approximately 0.2 ms. Traces remain in ignored diagnostic
files `.tools/edit-read-trace.log` and `.tools/coverage-query-plan.log`; all
temporary instrumentation has been removed from source.

Migration `020d_source_coverage_index.sql` indexes non-pending source outcomes
by organisation and passage, including the outcome itself. The query excludes
pending rows explicitly because they contribute to none of its outcome
counts. Total passage units still include every passage, and evidence counts
are independent. This adds index storage and maintenance on reading-state
writes; it does not change authoritative reading state or require a cache.

Profile responses now read the existing parts first. Only a view missing its
whole-project part creates it and rereads. Ordinary views and edit responses
avoid the previous unconditional insert-on-conflict call. Missing-project
and read-error handling remain in the existing response path.

The broader query rewrites were rejected. A direct project-wide join timed
out under stale statistics; grouping all passage states twice measured a
59.2 ms edit median. A bounded full-join variant measured 53.5 ms. A second
covering index on passages did not improve the full path and was removed.
The retained index prototype alone measured 50.322 ms, still failing 50 ms.
The checked-in diagnostic results preserve these failed experiments; they
are not release or passing-gate claims.

Database, store and HTTP suites pass serially
(`.tools/coverage-index-tests.log`: 2.309/27.127/13.489 seconds). The expanded
coverage test checks matching passage IDs in another tenant and transitions
from background to pending, mapped and background again. All outcome totals,
units and independent evidence counts remain correct. Its component timing
is 8.381/9.147 ms (`.tools/coverage-index-transition-tests.log`). The API suite
covers initial profile creation and subsequent edited responses.

The final unchanged 40-sample full workload is
`bench/results/2026-10-06-profile-coverage-index.json`. Spec Home edit measured
50.603/52.5 ms, rebuild 29.9/32.1 ms, filing 471.8/687.2 ms and report assembly
9.2/10.1 ms. The edit median remains above 50 ms, so the gate correctly exits
1. The combined changes improve the measured path but do not close the gate;
the small lazy-initialization saving is not independently established beyond
run variation. Local extraction/admission overruns still require target-VPS
measurement. No release claim is made.

The test database contains the migration's final index and no experimental
coverage indexes. `git diff --check` passes, and no temporary tracing remains.
The separate earlier 664 ms rebuild tail anomaly did not recur in this run;
its cause remains unproven as recorded in the fact-source-scope evidence.
