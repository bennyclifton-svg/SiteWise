# Whole-project part creation under concurrency

Lane A. Preserve a reliable first profile view/edit when two requests create
the same site's whole-project part. Existing edit/rebuild budgets remain
p50/p90 50/150 ms and 100/300 ms. No new dependency, schema, service or Jev call.

The former insert-and-read CTE could observe a uniqueness conflict with a row
committed after its statement snapshot, then return no row because its SELECT
used that earlier snapshot. The caller consequently received `ErrNotFound`
despite a valid project and a committed whole-project part.

`TestEnsureWholePartConcurrentCreation` holds an uncommitted competing insert,
starts the real store method, and observes PostgreSQL's blocking PID relation
before committing. It asserts the actual winner's ID and label are returned,
and exactly one whole part exists. This is a database-observed wait, not a
timing-only sleep. The test failed on the old method
(`.tools/whole-part-race-red.log`).

The helper now reads an existing part first. Missing parts use the same
tenant/site uniqueness constraint with `ON CONFLICT DO NOTHING`. If that insert
returns no row, a new statement reads the committed winner. A genuinely absent
or foreign project still returns not found. Existing edit and rebuild calls
avoid a needless insert attempt; no part label or class is overwritten.

Store, HTTP and worker suites pass serially against PostgreSQL after the fix
(`.tools/whole-part-race-green.log`: 28.533/13.606/9.864 seconds), including the
new concurrency case and existing tenant/site isolation cases.

## Related timing investigation

The ordinary edit response trace measured source coverage around 9 ms and
the complete response read around 13 ms. The same run reproduced the separate
tail anomaly: rebuild p90 671.070 ms and edit p90 686.682 ms. Thus response
coverage alone does not explain that delay. These are diagnostic observations
in `.tools/profile-read-stages.log`, not release evidence.

A subsequent store-test setup and query-traced run measured edit median
92.913 ms with rebuild near 33 ms, but did not reproduce the 700 ms tail. The
following query trace returned edit median 50.223 ms. Extending the tracer to
pgx batches and COPY measured edit 49.4/50.8 ms and rebuild 29.6/32.0 ms, with
no slow reconciliation batch. All temporary response/query/batch/COPY tracing
is removed. The source of the intermittent tail remains unproven; this
concurrency fix does not claim to resolve it.

The uninstrumented full workload after the fix is
`bench/results/2026-10-06-whole-part-concurrency.json` (20 files per round,
two rounds, 40 API samples, integrated proposal diagnostic). It **fails**:
ordinary edit 686.523/690.273 ms and combined rebuild 720.677/730.058 ms;
filing passes at 469.630/652.891 ms. The slowdown affected most samples, not
merely the earlier first-five pattern. This is no speed-improvement claim.

An immediate repeat with temporary rebuild-stage timers did not reproduce
the delay; ordinary edit median was 50.036 ms, still narrowly failing. No
stage exceeded 100 ms. That repeat is diagnostic only
(`.tools/race-tail-stages.log`); it does not supersede the failed uninstrumented
measurement. Those timers are removed, preserving the tested concurrency fix.
`git diff --check` passes with existing line-ending warnings. Automatic proposal
generation remains disabled, with the speed gate still open.
