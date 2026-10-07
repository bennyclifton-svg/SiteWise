# WP-14: revisions, fingerprints and staleness

Date: 5 October 2026. Lane A. Implementation in the supplied `main` working
tree, based on `80a5a4244623109b5da70b14859c303a46e056b8`. No commit, merge or
deployment was made. Independent review and integration-lead acceptance remain
before merge. This note does not mark the wider wave Verified.

Contracts: implementation plan §3.5, §3.6 and §4.4, last changed at
`ceb67518a03701a7df948fe2f384cce816e0e404`. Migration reservation: **014**.
The owner's request to implement WP-14 supersedes the plan's historical
documentation-only instruction for this package.

## Outcome and implementation

Profiles expose their completed build revision, input fingerprint, knowledge
and question versions, and a `stale_for` array. The existing pending, active,
unread and failed document counts remain available; no ETA is invented.
The durable `profile` SSE event carries `{project_id, revision}`.

- `014_revisions.sql` adds the tenant/project-scoped domain counters and extends
  the existing `profile_builds`. Existing builds have revision zero and are
  explicitly stale until rebuilt. New projects create their counter row.
- `knowledge.Load` hashes the exact file bytes it parses, in relative path order.
  The version stays fixed on the loaded catalogue. Disk edits affect the next
  load, never an already-running instance.
- Configured API and worker stores reconcile in the authoritative write
  transaction. A value, planning, scope, reading, part, site or evidence change
  commits its counter, computed rows, build metadata and event together. A
  rebuild error rolls them back. Filing and extraction invalidate counters
  without adding a rebuild or model call to foreground filing.
- Rebuild fingerprints include fact IDs/values/provenance, documents and reading
  settings, parts/site values, user/planning values, reading policy, thresholds,
  questions and knowledge versions. Row ordering, job progress, timestamps and
  optimistic-lock version increments alone do not change the fingerprint.
  Repeated builds still advance the build revision.
- Profile reads use one repeatable-read transaction for their rows, metadata,
  counters and coverage. Freshness depends explicitly on `profile_inputs`,
  `works`, and the loaded configuration versions; unrelated domain counters
  do not invalidate this projection. There is no dependency engine.
- Update requests retain the durable jobs and their unique document/kind key,
  with a project lock to serialize concurrent requests.

No service, library, runtime dependency, model call, question, threshold or
knowledge record was added. Rule-number verification and knowledge review
statuses are unchanged.

## Performance findings

The new benchmark imports a private, saved Spec Home state into the dedicated
test database: **38 documents, 1,153 facts and 6,742 passages**. It times the
complete store rebuild, including input reads, hashing, reconciliation, row
writes, metadata and event commit. It also records full HTTP edit timings as
`spec_home_profile_edit`, separately from the existing bench-project edit gate.
Raw fixture contents remain under ignored `data/eval/profile/private/`.

The first loaded run exposed a 60-second source-coverage query timeout. A
focused reproduction located repeated passage scans under stale statistics.
Coverage now aggregates per document with bounded source lookups. Fact
snapshots also resolve document metadata once per document, within the same
SQL statement as the facts and passage provenance. This preserves the F31
concurrent-source-replacement guarantee.

Regression coverage uses temporary tables analyzed before inserting a large
upload, including missing source metadata, supersessions and foreign-org
records. No global planner setting or production database setting was changed.
Temporary per-session `auto_explain` and application timing traces were removed.

Final aggregate: `bench/results/2026-10-05-wp14.json`. All user-path gates
passed on the final implementation. Local replay timings are a timing model,
not target-VPS release evidence. Component overages remain reported under
D-37 without loosening any budget.

| Path | Samples | p50 / p90 ms | Budget ms |
| - | - | - | - |
| `whole_intake` | 368 | 338.238 / 883.582 | 1,000 / 2,000 |
| `project_profile_read` | 80 | 1.606 / 19.332 | 50 / 150 |
| `profile_edit` | 160 | 32.773 / 82.690 | 50 / 150 |
| `profile_rebuild` (Spec Home) | 40 | 53.039 / 57.606 | 100 / 300 |
| `spec_home_profile_edit` | 40 | 95.850 / 106.721 | reported separately per D-18 |

Fixture SHA-256:
`858ce589eeea6cf3dde011d9e393f4cb2428e1dc8f512bc15fed0c67800ee61e`.
The run used 184 input files, two rounds, four upload workers, two background
callers (175 calls), and 10% injected stalls (55 stalled requests).

## Verification

- Strict knowledge check: **0 errors, 3 existing warnings**.
- Clean npm install and production build: passed.
- Full Go suite, `go test -p 1 ./...`: passed.
- Installed OCR gate: filing **3,201/3,223 ms**, detail recovery
  **3,196/3,216 ms**, both against 10,000/20,000 ms.
- Browser tests: **11/11 passed**. As with F31, Windows teardown left the
  owned test server and its `go run` parent alive. They were identified by
  executable and start time and stopped after all assertions passed; the
  runner then exited successfully.
- Intake replay: **218 recorded calls, 0 provider errors, 0 replay misses**;
  existing accuracy gate passed. No answer key or baseline changed.
- Source/Hale replays: **15/15 and 12/12**, zero forbidden readings applied.
- Loaded benchmark: all user paths passed; table above.


The Windows equivalent of `tools/check.ps1` uses npm's Node entry point because
the npm shim can select the wrong shell on this host. The full Go suite uses
`-p 1`: concurrent package tests contend with its wall-clock assertions (one
parallel run failed the F31 timing assertion; the focused run was 11.3/12.3 ms).
Budgets and assertions are unchanged. The full benchmark retains its prescribed
four upload workers, two background callers and injected provider stalls.

```powershell
$env:PATH = (Join-Path (Get-Location) '.tools/go/bin') + [IO.Path]::PathSeparator + $env:PATH
$env:GOCACHE = 'D:\AI Projects\sitewise\.tools\go-cache'
$env:GOPATH = 'D:\AI Projects\sitewise\.tools\go-path'
$env:GOTOOLCHAIN = 'local'
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
python tools/check_knowledge.py --strict
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web ci
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run build
go test -p 1 ./...
pwsh -NoProfile -File tools/check-ocr.ps1
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run test:e2e
go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-bench -out bench/results/2026-10-05-wp14.json -samples-out .tools/wp14-final-samples.json
```

New tests cover rollback after a rebuild failure, concurrent writes, every
domain counter, tenant/FK isolation, build persistence after reopening the
store, concurrent refresh coalescing, fingerprint input classes and ordering,
startup hash immutability, JSON/SSE revision agreement and zero Jev calls on
normal profile reads/edits. Existing failure/progress and lease-recovery tests
remain part of the full suite.

## Requirement handoff and boundaries

| Requirements | Package result |
| - | - |
| NW-REQ-065, 068, 086, 240, 269, 274, 371, 372 | Implemented for the existing profile pipeline; local tests above. Existing stores extended. |
| NW-REQ-268 | Profile staleness implemented. Report projection dependencies and report UI belong to WP-41. |
| NW-REQ-177 | Startup knowledge version available; proposal method provenance remains WP-26. |
| NW-REQ-326 | Spec Home rebuild path and 100/300 ms gate added. 0991 fixture is WP-28; proposals remain WP-26. |
| NW-REQ-314, 332 | Existing profile behavior/progress retained and tested; the full Stage 1 gate still includes WP-15. |

AT-25: the API test checks no additional Jev hits. AT-26 and the profile part of
AT-20: existing pending/failed/active counts and failure/retry tests retained.
AT-21: atomic rollback, reopening the store and existing lease recovery tested;
assembly/issue recovery remains with the later report packages. No live Jev
calls, VPS measurements, issue/export tests or independent review were run.
There are no open WP-14 design questions.

`ProfileSnapshot.WorkItems` is the fingerprint input seam for WP-20; there is
no work-item table at this stage. WP-20 must populate it from its authoritative
rows before rebuilding. Future domain stores call `BumpRevision` inside their
write transaction, acquiring the project lock before other write locks. Report
assemblers must declare their own explicit dependency lists.

To recreate the private fixture, use `tools/export_profile_bench.py --database
<local-read-source-DSN> --project <Spec-Home-project-UUID> --psql <psql-path>`.
It runs one read-only snapshot query and writes only to the ignored private
directory. The benchmark never connects to the source database.

Workflow deviation: work was implemented in the supplied checkout rather than
a new worktree. Register/status acceptance remains for the integration lead at
merge. Unrelated existing `HANDOFF.md` deletion and `.claude/` files, and separate
Jev cancellation/retry/backoff edits arriving during this task, were preserved.
The final working tree and benchmark also contain those independent changes;
their behavior is not attributed to WP-14. The latter
share `internal/jobs/worker.go`; WP-14 changes there are limited to transactional
evidence persistence and its code-only rebuild.
