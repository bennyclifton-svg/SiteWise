# F31: profile edits after an upload with stale statistics

Date: 5 October 2026. Lane A. Branch `codex/f31-profile-snapshot`.
Base: `6098fa5`. This is the prerequisite defect repair before WP-14, not
completion of WP-14 or WP-15.

## Outcome and scope

A profile edit must keep the existing 50/150 ms p50/p90 budget when a large
upload has made PostgreSQL's table statistics stale. Filing keeps its
1,000/2,000 ms budget. No budget, filing baseline, migration, dependency,
Jev question or knowledge content changes.

The unchanged code reproduced F31 on the full replay benchmark:
`profile_edit` p50 193.081 / p90 303.796 ms with stale statistics (160 edits).
A normal-statistics baseline also missed p50: 56.132 / 68.784 ms.

## Change

`RebuildProfile` still reads facts and their source metadata in one SQL
statement. Its document, file and passage-source lookups now use lateral
subqueries bounded by org and exact ID. `OFFSET 0` prevents the planner
from flattening them into joins that multiply scans under stale statistics.

The profile returned after an edit also counted jobs and skipped kinds with
plan-sensitive joins. Query plans measured roughly 13 ms for the status
query and 29 ms for the skipped-kind query. These now resolve the project's
document IDs before reading jobs or decisions. Counts, reading filters,
source defaults, row order and org scoping retain their existing meanings.

An intermediate split-query implementation was rejected in independent
review: concurrent reprocessing could remove a source between the fact and
metadata reads. The final single statement preserves its database snapshot.
A deterministic interleaving test models replacement immediately after the
fact cursor closes and checks that the prior provenance remains complete.

The new store fixture uses temporary copies of the tables, analyzes them
before inserting 1,000 documents and 1,000 facts, and checks source fields,
supersessions, skipped readings, missing metadata, absent passage IDs and
foreign-org source IDs. It checks snapshot p50/p90 against 50/150 ms; the
full HTTP benchmark remains the end-to-end edit gate. The focused fixture
alone did not reproduce the original end-to-end failure; the full stale
benchmark did.

## Verification

Full gate passed on the final implementation. The final test-only additions
(database-name guard, migration setup and the reduced 1,000-document fixture)
also passed a fresh `go test ./internal/store -count=1`.

- Strict knowledge validation: 0 errors, 3 existing warnings.
- npm clean install and production build: passed.
- `go test ./...`: passed.
- OCR filing p50/p90 3,268/3,301 ms; detail recovery 3,266/3,285 ms,
  both against 10,000/20,000 ms.
- Browser tests: 11/11 passed. Windows left the owned test server alive
  during teardown; stopping that server and its `go run` parent let the
  runner exit successfully. No test assertions were skipped.
- Intake replay: passed, 218 recorded calls, 0 replay misses, 0 provider errors.
- Source and Hale replays: 15/15 and 12/12, 0 forbidden readings applied.
- Independent final review: no remaining actionable findings.

| Final stale-statistics path | p50 / p90 (ms) | Budget (ms) |
| - | - | - |
| `profile_edit` | 23.716 / 77.272 | 50 / 150 |
| `project_profile_read` | 2.120 / 17.704 | 50 / 150 |
| `whole_intake` | 398.257 / 1232.835 | 1,000 / 2,000 |

Stale-statistics aggregate: `bench/results/2026-10-05-f31-stale.json`; accuracy aggregate:
`data/eval/intake/results/replay-latest.json`. Temporary table options and
all three `auto_explain` settings were restored.

The benchmark uses the
existing 184-file corpus, two rounds, four upload workers, two simulated
background Jev callers and 10% injected stalls. It is local replay timing,
not target-VPS release evidence. Component overages remain reported under
D-37; every user path must pass locally.

Commands use Git Bash with the main checkout's Go runtime/cache and the
dedicated `sitewise_test` database. A worktree-local `.tools` junction and
private-data junctions provide ignored runtime inputs; read-only reference
paths are available through sibling junctions. npm runs through its Node CLI
because this host's npm shim can invoke the wrong shell (F27).

```sh
export PATH="/d/AI Projects/sitewise/.tools/go/bin:/c/Program Files/Git/bin:$PATH"
export GOCACHE='D:/AI Projects/sitewise/.tools/go-cache'
export GOPATH='D:/AI Projects/sitewise/.tools/go-path' GOTOOLCHAIN=local
export SITEWISE_TEST_DATABASE_URL='postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
python tools/check_knowledge.py --strict
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web ci
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run build
go test ./...
pwsh -NoProfile -File tools/check-ocr.ps1
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run test:e2e
go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-bench -out bench/results/latest.json -samples-out .tools/f31-final-stale-samples.json
```

Stale-statistics reproduction, on the test database after a bench cleans up
its org: `ANALYZE` documents, decisions, jobs and document_sources; set
`autovacuum_enabled=false` on those four tables, then run the benchmark.
Temporary `auto_explain` instrumentation identified the remaining read
queries and was removed before the final timing. Restore the four table
options after the run. No production database is involved.

## Review and remaining work

Independent reviewer: `/root/review_f31`. The split-query consistency finding
was fixed; the final review reported no remaining actionable findings.
The reviewer did not run concurrent database checks during measurement.

WP-14 remains next: revision counters, fingerprints, startup knowledge hash,
staleness and the `profile_rebuild` benchmark. Its write/rebuild transaction
contract still needs to be closed. WP-15 follows WP-14. No new package or
release requirement is marked Verified by this prerequisite fix.

## Post-merge check

Implementation commit `ceb6751`, merged to local main as `b1bb85b`.
The complete gate above was repeated on the merged checkout and passed:
knowledge, clean npm install/build, all Go tests, installed OCR runtime,
11/11 browser tests, intake replay (0 misses), source 15/15 and Hale 12/12
(0 forbidden), and the benchmark. The owned browser-test server again needed
manual termination after every browser test had passed so teardown could
finish. No application source changed after review.

With normal database options restored, the benchmark wrote
`bench/results/latest.json` (command used `-out .tools/f31-main-bench.json`
and `-samples-out .tools/f31-main-bench-samples.json`, then copied the aggregate).

| Post-merge path | p50 / p90 (ms) | Budget (ms) |
| - | - | - |
| `profile_edit` | 19.576 / 70.309 | 50 / 150 |
| `project_profile_read` | 1.185 / 13.542 | 50 / 150 |
| `whole_intake` | 400.975 / 1088.050 | 1,000 / 2,000 |

The final intake replay aggregate records the main checkout's raw input
hashes. Its manifest and thresholds JSON equal the worktree's exactly;
the raw hashes differ because main uses CRLF and the worktree uses LF.
Accuracy metrics are unchanged.
