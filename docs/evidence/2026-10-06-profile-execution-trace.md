# Profile execution trace

Diagnostic only, 6 October 2026. No production behaviour or dependency changed.

The frozen spec-home workload ran with Go execution tracing around its 40
profile rebuilds and edits. This uses runtime execution events, not the CPU
sampler that previously crashed on this Windows host. Temporary instrumentation
was compiled into a local diagnostic binary; the benchmark source was restored
byte-for-byte before executing it. Trace tooling was compiled from the existing
bundled Go source without installing a dependency.

Result: `bench/results/2026-10-06-profile-execution-trace.json` passes the local
gate. Spec-home edit p50/p90 was 49.817/52.143 ms against 50/150 ms; integrated
proposal rebuild was 87.204/93.274 ms against 100/300 ms. The edit remains the
ordinary edit path, not automatic integrated proposal generation. Maximum
rebuild was 114.774 ms and maximum spec-home edit was 53.080 ms.

The intermittent roughly 680 ms edit / 730 ms rebuild did not reproduce.
Trace-derived scheduler delay totalled 86.26 ms across concurrent goroutines
over the entire capture. Network and synchronization profiles include idle
server and background goroutines; their aggregate durations must not be read
as sequential request costs. The PostgreSQL log segment contained one duration
above the configured 100 ms threshold (215.124 ms). These observations do not
identify the cause of a failing run or justify a runtime configuration change.

Local diagnostic artifacts remain ignored under `.tools`: execution trace,
parsed events, four delay profiles, and the instrumented binary/source copy.
The restored benchmark source hash matches the saved pre-instrumentation
source. No source edits or test builds ran during measurement.

This establishes a working capture method for a future occurrence, not a fix.
Automatic edit-triggered proposal generation and the owner-reviewed M1 quality
gate remain outstanding. A passing local replay is not live Jev accuracy,
owner acceptance, or target-host release evidence.
