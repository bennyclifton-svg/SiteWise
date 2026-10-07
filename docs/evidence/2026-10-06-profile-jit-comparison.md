# Profile latency: session JIT comparison

Date: 6 October 2026. Diagnostic evidence, not a fix or passing gate.

The earlier full-fixture run reported approximately 680 ms edits and 714 ms
integrated rebuilds. Test whether query compilation contributes to that
intermittent delay before changing runtime policy. Alternative hypotheses are
plan selection and application/host stalls; neither is resolved here.

Ran the same 20-file, two-round, 40-API-sample replay benchmark serially, with
`-measure-proposals -integrated-proposals`. First used the connection-string
parameter `jit=off`; then used the ordinary connection string. No application
instrumentation, source edits, concurrent tests, or persistent database-setting
changes. A separate ordinary connection reported `jit=on`, with cost thresholds
100000 / 500000 / 500000 for compilation, inlining and optimisation.

| Session | Edit p50/p90 ms | Integrated rebuild p50/p90 ms |
| --- | --- | --- |
| JIT off | 50.438 / 52.883 | 86.242 / 92.644 |
| Ordinary setting | 50.879 / 52.837 | 88.260 / 93.162 |

Both runs fail the 50 ms edit median budget; rebuilds meet 100/300 ms. Neither
reproduces the large delay, and their similar results do not establish that JIT
caused it. Do not retain a JIT configuration change on this evidence. A useful
next comparison needs the slow state or a captured slow query/plan; another
fast pair cannot prove the original defect fixed.

Results are `bench/results/2026-10-06-profile-jit-off.json` and
`bench/results/2026-10-06-profile-jit-control.json`; raw samples and logs remain
in ignored `.tools/profile-jit-*`. Each loads 571 proposal records and produces
388 proposals on the final rebuild. Profile edits remain ordinary; their timing
does not establish an integrated-proposal edit budget. Jev latencies are replayed,
not live. Component extraction is reported as over budget and remains subject to
target-VPS judgment. No release, owner-quality or automatic-proposal gate is
cleared. No new dependency or runtime change is retained.
