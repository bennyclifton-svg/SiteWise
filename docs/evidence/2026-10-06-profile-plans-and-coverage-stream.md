# Profile plans and rejected coverage stream

WP-26 speed diagnosis, lane A. Existing budgets and the full remaining work
package scope are unchanged. No runtime change, migration or dependency from
this experiment is retained.

With the full 20-file/two-round upload history, temporary `EXPLAIN (ANALYZE,
BUFFERS)` captured the frozen fixture's snapshot query at 4.662 ms and source
coverage at 8.551 ms. Neither plan reported JIT. Coverage's partial index scan
performed 3,859 heap fetches across 38 document lookups, accounting for about
6 ms. The rest of coverage took approximately 2.5 ms. These normal plans do
not explain the previous 700 ms tail. Raw plans are in the ignored file
`.tools/profile-full-query-plans.log`.

A temporary session-only PostgreSQL `auto_explain` probe used a 100 ms
threshold, actual row counts, timing disabled and parameter logging disabled.
It was attached only to benchmark pool connections, with no persistent server
setting. Twelve full-upload/profile diagnostic repetitions failed to reproduce
the tail: rebuild p90 stayed approximately 29–35 ms. The slow plans captured
were benchmark organisation cleanup. The probe and benchmark early-return
branch are removed. These diagnostic loops omit later route benchmarks and
are not complete or release gate runs.

The coverage experiment read document, passage, outcome and evidence streams
in one statement, then counted by passage/document identity in Go. It preserved
one SQL snapshot, tenant scope, supersession exclusion and database filename
ordering. Existing coverage/tenant/stale-statistics tests passed. Its cost was
more rows transferred and more Go allocation/decoding.

`bench/results/2026-10-06-coverage-stream-experiment.json` records the full
unchanged workload with integrated proposal diagnostics: edit 52.755/58.4 ms
(median fails), combined rebuild 87.5/92.3 ms. A controlled component comparison
then measured the stream at 16.487/17.323 ms versus the restored existing query
at 8.623/9.206 ms, using the same stale-statistics coverage fixture and 20 samples.
Logs: `.tools/coverage-stream-component.log` and
`.tools/coverage-restored-component.log`.

The stream implementation is rejected and fully reverted. The restored coverage
test passes. Existing tested concurrency and proposal-computation fixes remain.
Temporary plan capture and notices are absent from runtime code, and
`git diff --check` passes with existing line-ending warnings. No passing repeat
supersedes the earlier tail failures; automatic proposal generation remains off.
