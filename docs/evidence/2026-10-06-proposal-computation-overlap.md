# Proposal computation overlaps profile persistence

WP-26, lane A. Preserve a single atomic rebuild while reducing time to a saved
proposal projection. The existing budgets remain profile rebuild p50/p90
100/300 ms and edit 50/150 ms. No new dependencies, services or Jev calls.

The diagnostic integrated rebuild now evaluates proposals from completed,
immutable inputs while the calling goroutine persists profile rows. Fingerprint
calculation already overlapped these writes. A shared wait group joins both
computations on every exit; only the calling goroutine accesses the transaction.
Computation errors return before proposal persistence or build/event commit.
Ordinary edits still do not evaluate proposals.

The integration test retains complete-output equivalence and database-failure
rollback coverage. It additionally forces an evaluator error by changing record
identities after compilation. That failure must preserve both saved projections,
the profile revision/timestamp and the event count. Runtime catalogues remain
immutable; the test deliberately violates that boundary to exercise error
handling.

Before this change, a temporary pgx query tracer captured statements exceeding
100 ms without logging their parameter values. It reproduced neither the
700 ms rebuild tail nor a slow rebuild statement. Its only slow SQL was the
benchmark's final organisation cleanup (392 ms). The run measured integrated
rebuild 97.2/103.1 ms and ordinary edit 51.333 ms median. The tracer and its pool
configuration are removed; this observation does not identify the tail's cause.

The uninstrumented result after overlap is
`bench/results/2026-10-06-proposal-integrated-overlap.json`: 20 files per round,
two rounds, 40 API samples, full loaded catalogue.

| Path | p50 ms | p90 ms | Result |
|---|---:|---:|---|
| Combined profile/proposal rebuild | 88.862 | 96.072 | Pass |
| Ordinary Spec Home edit | 51.290 | 53.472 | Median fails |
| Filing | 461.320 | 629.025 | Pass |

The benchmark as a whole fails. Neither integrated-edit speed nor reliable tail
behaviour is established. Automatic proposal generation remains disabled and
WP-26 step 0 remains open. No race-detector, owner-reviewed accuracy, live-call
or VPS/release result is claimed.

Validation: store, works, HTTP and jobs suites pass serially against PostgreSQL
(`.tools/integrated-overlap-broad-tests.log`), including the evaluator failure,
projection equivalence and rollback checks. `git diff --check` passes with the
existing line-ending warnings. Temporary SQL tracing is absent.
