# Profile observability comparison

6 October 2026. Diagnostic only; no production change or new dependency.

The previous undo-history benchmark reproduced slow spec-home edits
(682.045/688.426 ms p50/p90) and rebuilds (719.407/726.238 ms) with ordinary
connection settings. The preceding execution-trace run had been fast with
session slow-query logging. These observations suggested checking whether
observability itself changed the measured behaviour.

Two sequential runs used the same frozen spec-home workload, 40 API samples,
20 filing inputs per round, two rounds, and integrated proposal rebuilding:

| Diagnostic | Edit p50/p90 ms | Rebuild p50/p90 ms | Result |
|---|---:|---:|---|
| Execution trace, ordinary connection settings | 48.142/50.440 | 87.752/94.515 | Local gate passed |
| Trace only after a rebuild exceeds 300 ms | 49.971/52.166 | 84.613/92.607 | Local gate passed; trace never started |

Results are in `bench/results/2026-10-06-profile-trace-ordinary-connection.json`
and `bench/results/2026-10-06-profile-delayed-trace.json`. The delayed run's
maximum rebuild was 104.856 ms; its trace output was zero bytes and the trace
activation message was absent. Thus a fast run does not require either logging
or tracing. This does not explain the intermittent failing runs or prove that
instrumentation cannot affect them.

The first run used the earlier diagnostic executable, built before the
undo-history fix; that fix is on the undo endpoint, not profile rebuilding.
The delayed-trace binary was built from current source with temporary benchmark
instrumentation, restored byte-for-byte before execution. No tests or builds
ran during either measurement. No database/role overrides were found in
`pg_db_role_setting`. Connection settings were ordinary local settings, without
the logging parameters. No live Jev call or private payload transmission occurred.

Do not enable tracing or logging as a purported performance repair. These
runs narrow the observation but leave the root cause and integrated edit
budget unresolved. M1 owner-reviewed quality and later milestone gates remain
open. The delayed capture binary remains a local ignored diagnostic for a
future failing occurrence; it is not shipped as product behaviour.

## Activity-sampled ordinary run

`bench/results/2026-10-06-profile-activity-sampled.json` used the ordinary
benchmark launch/settings with external `pg_stat_activity` sampling for 55
seconds. Only hashes, query categories, wait categories and elapsed times
were retained; no query parameters or private source values were collected.
The only sampled application query over 100 ms was identified by exact SQL
hash as `DeleteOrg` cleanup (maximum observed 259 ms). Other long observations
were autovacuum delay waits. The sampler can miss brief queries and does not
establish that all database work is fast.

The roughly 680 ms anomaly did not recur. Spec-home edit was 50.129/51.761 ms
and rebuild 83.946/93.482 ms; the overall local gate correctly failed because
the edit median exceeded 50 ms. This result neither identifies the anomaly's
cause nor proves the tiny median overrun is a regression. No configuration or
production source change was retained. A sampled fast run cannot explain the
unobserved failing run, so further identical runs without a new capture
strategy would add little evidence.

## Bounded delayed-capture attempt

Three subsequent ordinary-connection runs of the delayed-trace diagnostic
completed without a rebuild exceeding 300 ms. No trace was activated. Results
are `bench/results/2026-10-06-delayed-capture-{1,2,3}.json`:

| Attempt | Edit p50/p90 ms | Rebuild p50/p90 ms |
|---|---:|---:|
| 1 | 50.814/52.789 | 87.636/93.236 |
| 2 | 50.738/52.705 | 86.338/93.311 |
| 3 | 51.339/54.555 | 84.394/91.962 |

All three fail the edit median budget; none reproduces the large anomaly.
The diagnostic executable predates the latest knowledge/test changes but
loads the current catalogue. No source/configuration change was made. This
bounded attempt yielded no root-cause evidence or fix; do not continue
identical runs as a substitute for a new diagnostic hypothesis.
