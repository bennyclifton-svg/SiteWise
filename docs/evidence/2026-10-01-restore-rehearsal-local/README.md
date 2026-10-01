# Local restore rehearsal, 1 October 2026

A rehearsal of the restore *checks* on the development machine (Windows 10,
PostgreSQL 17 from `.tools/`). It is **not** the release gate. That gate is
the VPS rehearsal in [operations.md section 5](../../operations.md#5-restore-and-rehearsal),
with pgBackRest, the off-site bucket and an isolated Linux host. Those have
not run.

## What was done

1. A fresh PostgreSQL 17 cluster on port 5435 stood in for production. It
   shared nothing with the development or test databases.
2. `sitewise serve` (the Task 12 build) ran against it **offline**: a
   placeholder Jev key, and `HTTPS_PROXY` pointed at a closed local port. No
   request left the machine, and every filing went grey. Health reported
   `degraded` for `jev not reached since start` and `jev circuit open`.
3. Two orgs were seeded through the real API: invite sign-in, a project
   each, and the public fixtures in `testdata/identity`. Org B also got a
   unique text upload. Three further empty orgs came from aborted sign-in
   attempts and stayed in as empty-org cases. Totals: 5 orgs, 15 documents
   (6 filed and 1 not filed per org, plus org B's unsupported text file),
   8 distinct blobs.
4. With the app stopped: `sitewise restore-check -out source-facts.json`.
5. Physical backup: `pg_basebackup -X stream -c fast` into a new data
   directory, then `pg_verifybackup` reported "backup successfully verified".
   Blobs were copied to a separate directory. The source cluster was stopped.
6. The copy was started on port 5434 as the "isolated host". WAL replay
   reached a consistent state.
7. `sitewise restore-check -expect source-facts.json -out restored-facts.json`.

## Results

| Check | Result |
|---|---|
| Restored vs source: per-org counts, 17 foreign keys (all validated), 5 migrations, blob digest | **passed**, identical ([source](source-facts.json), [restored](restored-facts.json)) |
| Every referenced blob present and hashing to its name | 8/8 |
| Cross-org references (`events.document_id`, documents→projects/files, decisions→documents, sessions/memberships→users) | 0 rows |
| Control: one blob deleted | **failed** as expected: `blob missing 0ce4f9f7…`, exit 1 |
| Control: same blob overwritten | **failed** as expected: `blob hash mismatch 0ce4f9f7…` |
| Control: extra event in org A naming org B's document | **failed** as expected: `events.document_id: 1 rows reference another org or nothing`, plus `events 8, source 7` |
| After undoing the controls | passed |
| App on the restored data: org A session → own project documents | 200 |
| org A → org B's project documents / org B's document | 404 / 404 |
| org B → own document; org B → org A's project | 200 / 404 |

## What this shows and what it does not

- `restore-check` detects missing and corrupted blobs, count drift and
  cross-org references, and passes a faithful physical restore.
- The tenant boundary holds through the application on restored data.
- **Restored sessions are live.** Org A's cookie from the source still signed
  in on the restored host. A rehearsal host must be unreachable by users.
- **Not shown:** pgBackRest restore from the S3 repository, WAL archive
  replay over a time range, rclone crypt round trip, Linux/systemd
  behaviour, restore timings at production size. All of these belong to the
  VPS rehearsal.
