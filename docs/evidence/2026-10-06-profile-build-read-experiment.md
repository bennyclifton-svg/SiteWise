# Profile build-state read experiment

Date: 6 October 2026. Reverted experiment; latency cause unresolved.

Lane A diagnosis under unchanged edit 50/150 ms and rebuild 100/300 ms budgets.
ReadProfile currently obtains build timestamp/thresholds in two scalar lookups,
then reads the remaining build metadata and current revisions separately.
An experiment combined this state into one scoped query in the same
repeatable-read transaction. Store and HTTP API tests passed (28.810 and
13.071 seconds), including revision freshness and rollback checks.

The subsequent full 20-file/two-round/40-sample integrated-proposal diagnostic
failed: edit p50/p90 683.015/687.851 ms; rebuild 717.743/725.357 ms.
`bench/results/2026-10-06-profile-build-read.json` preserves this result.
The direct rebuild does not call the changed response-read code. The run
therefore does not isolate any improvement or regression from consolidation;
the shared intermittent slowdown dominates it.

Restored both `internal/store/profile.go` and `internal/store/revisions.go`
byte-for-byte from pre-experiment copies. No source change is retained.

Then ran the same workload on restored code with session-only PostgreSQL
`log_min_duration_statement=100` and `log_parameter_max_length=0`. No application
instrumentation or persistent server configuration was added. This repeat
passed locally: edit 49.027/51.101 ms; rebuild 86.416/90.639 ms. PostgreSQL's only
logged statement above 100 ms was benchmark DeleteOrg cleanup (231.047 ms).
Result: `bench/results/2026-10-06-profile-statement-timing.json`. Server-log
extract, offset and samples remain in ignored `.tools/profile-statement-*`.

This is another non-reproduction under observation, not proof of reliable
latency. No cause has been established and no budget or release gate is cleared
by selecting the faster repeat. The contrast between the post-test slow run and
the following fast run suggests checking setup state before another query
rewrite; it does not establish that tests cause the delay. Prior full-suite
correctness evidence applies to the restored code. Automatic proposals, owner
quality review and subsequent milestone gates remain unchanged.

## Test-prepared reproduction check

Reran the unchanged store/API suites with `-count=1 -p 1`, then immediately
ran the original full benchmark with the same session-only 100 ms statement
logging and parameter-value logging disabled. Both suites passed (28.168 and
13.499 seconds). The following benchmark did not reproduce the large delay:
edit 51.532/52.729 ms, integrated rebuild 86.851/95.481 ms. The edit median
still fails. The only logged slow statement was DeleteOrg cleanup (224.501 ms).

`bench/results/2026-10-06-profile-post-test.json` records the run; ignored
`.tools/profile-post-test-*` contains samples, the log offset and server-log
extract. Running these tests first is insufficient to reproduce the slow
state under this observation. It does not eliminate other setup-state factors
or prove that statement logging is neutral. No source, persistent server
configuration or budget change was made. The large-delay cause remains open;
do not treat repeated fast diagnostic runs as proof that it is fixed.
