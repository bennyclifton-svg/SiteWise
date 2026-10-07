# CPU profiling probe and recovery

Date: 6 October 2026. Diagnostic limitation, not a latency fix.

Built a temporary benchmark binary using the standard Go CPU profiler, with
profile files restricted to ignored `.tools/` paths. Planned up to three serial
runs, stopping on a slow result. The first process crashed before producing
benchmark results: Windows access violation `0xc0000005`, address `0x150`, in
`runtime.(*unwinder).initAt`, called by `runtime.sigprof` / `runtime.profilem`.
The stack was in the intake workload; extraction stacks include PDFium's
WebAssembly implementation. No conclusion about a native PDFium defect follows.

Tried a narrower binary starting CPU profiling only at the saved-profile
rebuild/edit loop, after intake and fixture import. It also crashed in the same
profiler unwinder. No valid latency result or usable CPU attribution was obtained
from either probe. Stop this measurement approach on the present Windows
Go 1.27.1 toolchain rather than repeatedly crashing it. This is not proof that
CPU profiling fails in every program or that it explains ordinary latency.

Both source files (`cmd/intake-bench/main.go` and `profile.go`) were restored
byte-for-byte against their pre-probe copies before executing the diagnostic
binaries. No profiling code, dependency or persistent runtime setting remains.
Crash logs are ignored `.tools/profile-cpu-1.log` and
`.tools/profile-cpu-loop-1.log`; neither probe produced a benchmark result JSON.

The ordinary restored benchmark subsequently completed its full workload and
cleanup. `bench/results/2026-10-06-profile-profiler-recovery.json` records edit
p50/p90 54.577/59.844 ms and integrated rebuild 90.912/112.110 ms. The edit median
still fails 50 ms; the large delay did not recur. This recovery run is not a
release pass and does not supersede earlier intermittent failures. Filing,
edit and rebuild budgets remain unchanged; the goal remains incomplete.
