# Profile slowdown: plan-only and held-statistics probes

WP-26 diagnosis, with unchanged edit 50/150 ms and rebuild 100/300 ms budgets.
No runtime fix or new dependency is retained from this turn.

The previous full run failed at edit 679.626/685.590 ms and combined rebuild
713.996/724.880 ms. A session-only `auto_explain` probe with execution analysis
disabled did not reproduce that delay and logged only slow benchmark cleanup.
It was removed, restoring the original store file byte-for-byte.

A following ordinary full run was observed through a separate PostgreSQL
connection querying active statements older than 150 ms. It passed locally:
edit 49.8/52.2 ms and combined rebuild 85.9/93.2 ms. The observer logged later
autovacuum activity, not a slow profile statement. This demonstrates that
application instrumentation is not necessary for a fast repeat; it does not
prove what caused earlier slow runs or establish reliable latency.

The bounded statistics experiment then analysed all 43 public tables in the
dedicated `sitewise_test` database while the benchmark organisation was absent,
temporarily disabled automatic vacuum/analyse, and ran the unchanged full
20-file/two-round/40-API-sample workload with integrated proposal diagnostics.
Session-preloaded `auto_explain` recorded plans over 100 ms without execution
analysis or parameter values. The only slow plan captured was final benchmark
organisation cleanup (252.640 ms). The large profile delay did not recur.

`bench/results/2026-10-06-profile-statistics-held.json` preserves this diagnostic
result. It fails the ordinary edit median at 50.504 ms. It is not a normal-state
passing gate or release evidence, and it does not supersede the earlier failures.

Before mutation, a restoration script was generated from each table's actual
`autovacuum_enabled` option, preserving unset versus explicit values. A finally
block applied it after the benchmark; regenerating the same script from the
catalogue produced identical contents for all 43 tables. No persistent database
or server setting remains changed. Logs and restoration SQL are in ignored
`.tools/all-stats-*`, `.tools/all-held-stats-bench.*`,
`.tools/plan-only-probe.*`, and `.tools/profile-active-statements.log`.

Holding small-table estimates fixed is insufficient by itself to reproduce this
tail. Its cause remains unproven. Automatic proposals, owner-scored quality
gates, M2/M3 work and live/VPS release gates remain outstanding. No code change
was retained, so the preceding correctness results are not replaced by a new
test claim. `git diff --check` passes with the existing line-ending warnings.
