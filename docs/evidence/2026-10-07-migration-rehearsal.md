# Populated pre-M2 migration rehearsal

This is the implementation plan §7 upgrade dry-run, separate from the
already-final-schema restore rehearsal. It uses synthetic local data, not a
production backup or a claim about deployment readiness.

The opt-in `TestPopulatedPreM2MigrationRehearsal` creates two uniquely named,
owned databases on the dedicated test PostgreSQL server. The first receives
all baseline migrations through `020d`, then representative synthetic records
across two tenants: users/sites/projects/parts, documents/files/passages and
source outcomes, profile rows with assumption provenance, user-owned works,
packages/stages/responsibilities, delivery records/dependency, proposals with
triggers and dismissed decisions, and saved report drafts.

It takes a custom-format `pg_dump`, restores into the second newly created
database, compares old-table counts and canonical row SHA-256 digests, applies
the normal embedded migration runner, and compares again. Only explicitly
changed schema fields are normalized: new layout/hash/source-document columns
and the removed cached rank. Separate assertions verify layout defaults,
authoritative source-document backfills, blank hash compatibility, identical
proposal rank semantics per tenant, empty initial costs and validated
constraints. A second migration run must leave the same digests. Cleanup drops
only names whose creation succeeded in this run; caller-provided databases
are never restored over or dropped.

`internal/db/migrations/checks/pre_m2_preservation.sql` is the reusable read-only
psql inventory beside the migrations. Its subdirectory is excluded from the
migration runner's `migrations/*.sql` pattern. Run before and after an upgrade,
compare pre-existing table counts/digests, and review expected new tables
separately. The test also executes this script on the upgraded scratch database.
The script aggregates rows and may need substantial memory for large databases;
the Go fixture comparison streams sorted rows. Neither prints document content.

Run from the repository with `SITEWISE_TEST_DATABASE_URL` pointing to the
dedicated `sitewise_test` database and `SITEWISE_RESTORE_PG_BIN` pointing to
PostgreSQL 17 binaries:

```powershell
.tools/go/bin/go.exe test ./internal/db -run '^TestPopulatedPreM2MigrationRehearsal$' -count=1 -v
```

Validation passes through `021k`: 31 baseline migrations and 12 new migrations, all 42
pre-existing tables preserve row counts and normalized SHA-256 digests; all
backfill/default/rank/constraint assertions pass, the read-only psql inventory
executes successfully, and rerunning the migration runner changes no records.
The baseline also includes three evidence-call records; their new document
references backfill correctly and original fingerprints/results remain in the
normalized digest comparison. Test time 3.55 s (package 3.639 s). Ignored log:
`tmp/populated-upgrade-rehearsal.txt`. Both owned scratch databases were cleaned
up without error.

The final-schema restore suite after 021k also passes (4 tests, 5.811 s): the
independent dump/restore rehearsal takes 3.78 s and preserves all 96 table/tenant
digests present in the final test database, 115 validated foreign keys, the
immutable issued snapshot and its PDF blob.
Ignored log: `tmp/final-restore-rehearsal.txt`.

The production pre-deployment backup, populated production clone comparison,
restore evidence and VPS timing remain required independently; no production
deployment occurred.
