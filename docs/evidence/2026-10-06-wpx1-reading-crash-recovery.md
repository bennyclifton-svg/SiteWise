# WP-X1 / AT-21: reading process recovery and stable evidence retries

Lane A defect found while completing the Lane C recovery sweep. Outcome:
recover interrupted reading without losing the last completed profile or
repeating completed judgments. Budgets remain filing p50/p90 1000/2000 ms,
profile edit 50/150 ms and profile rebuild 100/300 ms. No new service or
dependency; no live model call. Scope is label/evidence reading recovery, not
new questions, model calibration or issue/export.

The process test in `internal/jobs/reading_crash_test.go` covers both label
and evidence stages, with two source passages. Evidence starts with the real
label-stage implementation and its durable cache. The owned child runs
`Worker.Once`; a database lock stops it at profile-row replacement, after
the transaction has written facts and synchronized proposed works. PostgreSQL
activity confirms that exact boundary before the parent kills the child.

The test checks saved facts, profile rows/build fingerprint, work items,
domain revisions and events both while the transaction is blocked and after
the crashed backend disappears. They must equal the pre-crash snapshot.
The separately committed passage cache survives; the interrupted document
remains visible as pending. After natural lease expiry, a fresh process must
complete the same job on attempt two. Its judgment provider deliberately
returns an error if called, so success proves actual cache reuse. The cache
rows remain byte-for-byte unchanged, facts are complete without duplicates,
the profile advances once and one completion event is published.

## Defect and fix

The first test failed for evidence reading. Diagnostic probes showed that
the first request used the hydraulic family (97 questions), while retry used
the family plus the gas leaf added by the evidence results (114 questions).
The worker was feeding its own partially persisted outputs back into the
request, changing the fingerprint. Repeating the test with an actual label
stage ruled out an artificial leaf-only fixture as the cause.

Evidence reading already loads the fingerprint-matched label result to reuse
its location. It now also derives the request's labels from that same result.
No extra query or model request is introduced. Genuine changed label inputs
still invalidate that lookup; the existing fallback applies when no matching
label result exists. Stored evidence labels remain available to other views.

This follows TypeSafe's [speculative fan-out pattern](https://docs.typesafe.ai/patterns/fan-out):
one request per state, with code applying the results. The fix preserves the
original evidence request on retry rather than using its output to create a
second state. Diagnostic logging was removed after the failing process test
passed. Keeping the originating label input distinct from evidence outputs
would have prevented this cache invalidation.

Validation on 6 October 2026: the complete serial Go suite passes
(`.tools/wpx1-reading-crash-all-tests.log`). After strengthening the projection
assertions, the focused extraction/reading process suite passes in 4.922
seconds: recovery publishes the new work type and updates the existing work
item without changing its ID. `git diff --check` passes; no diagnostic logs
remain.

The unchanged local workload (20 files, two rounds, 40 samples per API path,
saved Spec Home fixture, recorded Jev latency) is saved in
`bench/results/2026-10-06-reading-recovery-diagnostic.json`. Filing p50/p90 is
478.1/678.5 ms against 1000/2000 ms; profile rebuild 29.3/33.6 ms against
100/300 ms; report assembly 7.6/8.6 ms against 300/1000 ms. The overall gate
still fails: Spec Home edit is 52.496/58.9 ms against 50/150 ms. Local component
overruns also remain recorded for target-VPS judgment. Budgets were unchanged.

Issue recovery awaits the M2 issue endpoint. WP-X1 still requires the semantic
relationship review; this evidence does not claim M1, production or release
completion.
