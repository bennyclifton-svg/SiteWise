# Proposal SQL rank-only experiment

6 October 2026. Rejected experiment; no production change retained.

The execution trace attributed 1,550.97 ms aggregate network wait to
`writeProposals` over the captured workload. This is cumulative wait, not
per-request CPU time. Inspection confirmed persistence already uses a bulk
JSON recordset, not one insert per proposal. Earlier field comparisons found
many rank-only changes. Hypothesis: preserving the stored large reason value
when only rank changes could reduce write cost.

The candidate prepended a SQL rank-only update. It compared every other
mutable column with `IS NOT DISTINCT FROM`, including complete reason JSON,
state, provenance-related fingerprints and display flags. The existing full
upsert then handled remaining changes. Unlike the earlier Go prefilter, this
added no projection read and did not discard unknown JSON fields.

Projection/rollback checks passed, then store (29.208 s) and API (13.309 s)
suites passed. However, `bench/results/2026-10-06-sql-rank-update.json` failed:
spec-home edit 678.407/684.595 ms and rebuild 727.140/735.679 ms. The ordinary
edit path does not execute the modified proposal write, so this run cannot
attribute the common slowdown to it. It also provides no evidence of benefit.

The candidate was removed and `internal/store/proposals.go` restored from its
exact pre-experiment bytes. Restored projection/integrated checks pass
(1.039 s). The candidate remains ignored under `.tools` for reference only.
No migration, dependency, database configuration or live Jev request was
introduced. Do not treat this as a fix or as permission to enable automatic
proposal generation. Integrated edit timing and owner quality remain open.
