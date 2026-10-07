# Profile latency: parallel-worker session comparison

Date: 6 October 2026. WP-26 step-0 diagnosis; no runtime change retained.

The preceding rank-write experiment reproduced ordinary edit/rebuild medians
near 680/726 ms. Its runtime changes were reverted byte-for-byte. This test
examines a distinct hypothesis: parallel-worker startup on the Windows host.
Competing explanations remain a slow non-parallel plan or delay outside the
database. This does not alter the 50/150 ms edit or 100/300 ms rebuild budgets.

Both runs used the restored implementation, 20 files, two rounds, 40 API
samples, replayed Jev latency and integrated proposal rebuild diagnostics.
Session-level statement logging was enabled at 100 ms, with parameter logging
disabled. No source changes, tests or other compilation ran during measurement.

| Session | Edit p50/p90 ms | Rebuild p50/p90 ms | Result |
| --- | --- | --- | --- |
| Default parallel workers (2) | 50.128 / 52.257 | 87.425 / 92.557 | Edit median fails |
| Parallel workers disabled (0) | 50.241 / 52.793 | 87.329 / 93.033 | Edit median fails |

Artifacts: `bench/results/2026-10-06-parallel-control.json` and
`bench/results/2026-10-06-parallel-disabled.json`. Each new PostgreSQL log segment
contained only one statement over 100 ms: organization cleanup, at 287.892 ms
and 258.014 ms respectively. Parameter values were not printed or collected by
these session logs. The control activity snapshot occurred after completion and
had no active benchmark backend; it is not evidence about worker participation
during a request.

Neither run reproduced the large delay. Disabling workers did not improve the
ordinary median and does not establish the cause of the intermittent problem.
It is neither a fix nor grounds to rule out parallelism during a failing run.
No setting is retained: a fresh ordinary connection confirms workers remain
2, statement logging is -1, and parameter-length logging is its original -1.
All overrides were limited to the benchmark connections.

Automatic proposal generation remains off. These runs are not integrated-edit,
live-provider, target-VPS or release evidence. A next diagnostic should capture
where a delayed request waits without the CPU sampler that previously crashed;
runtime execution tracing is a candidate, subject to its own overhead check.
