# WP-26 proposal rank-write experiment

Date: 6 October 2026. Lane A speed investigation; automatic generation remains off.

The preceding local integrated rebuild passed 100/300 ms at 87.337/92.143 ms.
That is still above the edit median budget of 50 ms before response reading,
so it cannot justify wiring proposals into edits. Edit budget stays 50/150 ms;
filing stays 1000/2000 ms. No dependency, schema or Jev call is added.

## Evidence

An exact-match prefilter read stored proposals under the project lock, compared
them after applying decisions, and sent changed rows through the existing full
upsert. Store/API suites passed (28.599/13.255 s). An aggregate field trace logged
only counts, never record IDs, source text or values. The 20-file/two-round/
40-sample run is `bench/results/2026-10-06-proposal-field-diff.json`.

On repeated subclass changes, 381 of 388 proposals differed: rank changed on
377, reasons and input fingerprints on 24, specificity on one, and four were new.
Counts overlap. The initial repeated unchanged input had zero differences.
This explains why the earlier selective-projection experiment found few exact
matches. The prefilter alone measured rebuild 99.122/105.123 ms and ordinary
edit 50.509/51.998 ms; the combined gate failed. It does not establish a speedup.

## Rank-only candidate

The next candidate separates rows for which rank is the sole difference. Those
rows use `UPDATE ... SET rank`, preserving their stored reason value instead of
resending it through the full upsert. Everything else retains the original
complete update. Trigger reconciliation and obsolete-proposal removal still run
independently, including when the projection content is unchanged.

A typed-struct comparison initially lost unknown JSON reason fields. A new
regression reproduced that stale-data bug. The candidate now compares complete
JSON objects after removing only the scoped org/project/site columns and
normalizing rank for comparison. JSON numbers use `UseNumber`, preventing float
rounding from equating different provenance values. Unknown fields remain part
of the comparison. Missing/current reasons, critical flags and rank changes
cannot take the rank-only shortcut unless all other fields match exactly.

Expanded projection regressions cover simultaneous rank/reason and rank/critical
changes, an unknown reason field, stale labels, missing trigger repair, child
trigger replacement with the same semantic fingerprint, and unchanged-row
retention. Integrated rebuild tests cover complete output and atomic rollback.
The focused corrected comparison passed in 1.047 s. Temporary aggregate tracing
was removed before final validation and measurement.

## Outcome: candidate rejected

The complete-JSON candidate passed store/API suites in 29.544/13.179 s.
The final uninstrumented artifact,
`bench/results/2026-10-06-proposal-rank-writes.json`, failed substantially:
ordinary edits measured 679.671/683.898 ms and integrated rebuilds
726.002/736.301 ms. Both paths show the previously observed large slowdown;
this experiment does not identify its cause or establish a rank-write speedup.

Both prefilter and rank-only runtime changes were removed. `proposals.go` was
restored byte-for-byte from its pre-experiment copy, checked by SHA-256. The
expanded reason/critical/unknown-JSON projection tests remain; they protect the
original full comparison too. Focused projection and integrated rollback tests
pass again on the restored implementation. No new runtime dependency, setting,
schema, cache or instrumentation remains. Automatic generation stays off.

The useful new evidence is the field-level distinction: broad rank churn is
different from reason/fingerprint churn. Any next optimization must avoid
paying for a complete stored projection read merely to rediscover that fact,
while still protecting full reason contents and actual trigger identities.
