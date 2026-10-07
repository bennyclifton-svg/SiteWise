# WP-X1 composite foreign-key sweep

Lane C. Outcome: enforce tenant and project/site ownership in the database,
independently of API validation. No production code, service, dependency or
latency path changed. Existing user-facing budgets remain in force.

`internal/db/migrate_fk_sweep_test.go` applies the actual migration files in a
temporary schema on the dedicated `sitewise_test` database. It inventories
composite foreign keys before migration 011 and after the current final
migration, identifying all 51 new or changed keys in this wave. The complete
public schema currently has 72 composite keys; older unchanged filing/auth
keys are outside this new-wave delta, rather than silently counted as tested.

For every key, a valid insertion is first checked as a positive control. The
test then constructs invalid associations using existing parent records from
another tenant, and from another project/site in the same tenant for keys
with additional ownership components. A PostgreSQL `23503` is accepted only
when its reported constraint name is exactly the one under test. A check,
uniqueness violation or failure of a neighbouring foreign key does not count.
Deferred constraints are explicitly made immediate before judging the result.
Generated proposal-decision target columns are exercised through their actual
record type and record ID inputs.

The expected definitions are pinned in
`internal/db/testdata/next_wave_foreign_keys.json`. A migration which removes,
weakens, renames or adds a constraint therefore requires an explicit contract
and coverage review; discovering the remaining keys alone cannot hide a lost
constraint. Fixtures include sites, parts, planning/profile values, works,
proposals/decisions, packages/stages/scope, delivery/dependencies and report
versions/edits/references. Each attempted insertion rolls back to its savepoint,
and the outer schema transaction is rolled back after the test.

Validation: all 51 targeted constraints pass (`.tools/wpx1-fk-sweep.log`), and
the full database suite passes (`.tools/wpx1-db-tests.log`). A subsequent schema
query finds no retained `fk_sweep_*` schemas. `git diff --check` passes. No
runtime AI call or speed-budget relaxation was introduced.

This proves enforcement of the current new-wave composite-key contract. It
does not replace semantic review of fields without a foreign key, the route
matrix, or the remaining process-restart tests. WP-X1 is not yet complete:
issue recovery follows the M2 issue endpoint. Mid-reading recovery now passes
in `2026-10-06-wpx1-reading-crash-recovery.md`. Report-assembly recovery passes; see
`2026-10-06-wpx1-report-crash-recovery.md`. The profile-edit speed gate and
owner/live/VPS gates also remain open.

Extraction worker recovery now passes in
`2026-10-06-wpx1-worker-crash-recovery.md`; that is separate from mid-reading.

The later actor review added five missing composite constraints. The current
pinned inventory is 56; see `2026-10-06-wpx1-actor-scope.md`. The 51 count above
records the original sweep, before those missing relationships were identified.

Subsequent source and lineage checks strengthen the planning successor key
and add the project-scoped fact/document key. The current inventory is 57;
see `2026-10-06-wpx1-planning-lineage.md` and
`2026-10-06-wpx1-fact-source-scope.md`.
