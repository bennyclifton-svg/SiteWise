# K4 review preparation and held-statistics check

WP-K4 can proceed independently of M1 in the package plan. Added a read-only
inventory tool, its reproducible JSON report and a current supplement to the
owner packet. The supplement corrects the obsolete work-type implementation
warning, enumerates current attachment scope, and retains unresolved merge,
detector, source-verification and owner-review decisions. No knowledge data or
review status changed, and no entire package or milestone is marked complete.

The audit covers all knowledge YAML, rejects duplicate record IDs, records
content hashes and compares existing failure modes to exact pre-K4 commit
`02094dbde0d9bd90f02d82c6ccc5bea2933a620c`. UTF-8 decoding is explicit. A repeat
produces byte-identical JSON. The strict knowledge checker reports zero errors,
zero pending ledger entries and three existing package-default warnings; all
30 tools tests pass. The targeted code-fed work-type evaluator test also passes.
The tool reuses existing PyYAML and Git; no dependency or runtime path changes.

Before the review refresh, the handoff's F30 reproduction recipe was run on
the dedicated `sitewise_test` database: analyse documents, decisions, jobs and
document sources while small, then temporarily disable automatic vacuum/analyse
on those four tables. A session-only `auto_explain` probe captured statements
over 100 ms with parameter logging disabled. The full workload did not reproduce
the 700 ms tail; the edit median still failed at 51.203 ms. The run is diagnostic
only (`.tools/held-stats-bench.log`), not release evidence.

The command's finally block restored all four table options. A subsequent
catalogue query confirmed their `reloptions` are empty, matching the initial
state. The temporary connection probe was removed and `git diff --check` passes.
The speed, live-call, owner-review and M1 gates remain open. General optimistic
UI polish is WP-70 in M3; it was not brought forward during this audit.
