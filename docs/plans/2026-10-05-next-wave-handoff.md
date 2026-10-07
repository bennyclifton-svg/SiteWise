# Next wave: handoff, 5 October 2026

For the next implementing agent (Claude or Codex). Read `AGENTS.md`, then the plan's §0.2, §1.1, §5 and §8.6, and work packages §2.4. This note only covers what those documents do not.

## Continuation update

6 October: the local continuation has implemented substantial M1 code beyond
WP-14, without committing or merging it. The current package/phase summary is
now reconciled in the implementation plan §0.2 and package status table. Read
those and the named handoffs before using the historical baseline below.
Automatic proposal generation remains off; intermittent profile latency,
changed-question evaluation and owner-reviewed 0991 quality remain open.
M2/M3 have not been cleared. Preserve the current dirty worktree.

The following 5 October continuation note is historical:

F31 is fixed in `codex/f31-profile-snapshot`, based on `6098fa5`, with the
full local gate and independent review passed. See
`docs/evidence/2026-10-05-f31-profile-snapshot.md`. Profile edits with stale
statistics are back inside 50/150 ms. WP-14 remains next; its revision,
fingerprint, staleness and atomic write/rebuild work has not started.
The original baseline below is retained as history.

## Where things stand

`main` is at `77ced4b`. Nothing is in progress, and every item below is merged with a green full gate and an independent review.

| Item | Merge | Note |
| - | - | - |
| WP-11 sites and parts | `0a13afd` | |
| WP-25 works evaluator | `b3a1048` | |
| M0, whole-file replay (AT-35) | `e2eaa6d` | |
| D-37, component budgets judged on the VPS | `31681f9` | |
| Clerk source archive (owner) | `24411b7` | |
| WP-12 value provenance and scope | `ece0086` | |
| F30, document list stale-plan fix | `3e55ca1` | |
| WP-13 planning values | `77ced4b` | |

Requirement states are recorded in the register's "State updates"; findings F30 and F31 are in plan §4.

**Next:** WP-14 (revisions, input fingerprints, knowledge version, staleness, and the `profile_rebuild` bench path). Do F31 first (below). Then do WP-15, as work packages §2.4 orders.

## Open items

- **F31.** With statistics held stale, `profile_edit` measured p50 251 ms against a 50 ms budget; under normal conditions it is about 20 ms. Before adding the `profile_rebuild` path, give the rebuild's snapshot queries the F30 treatment: no joins whose cost multiplies under stale statistics (`readSnapshot` in `internal/store/profile.go`).
- **Deviation from plan §3.4 (WP-12 and WP-13).** The profile rebuild runs in its own transaction after the write, not in the same one. WP-14 adds revision counters to the same path, so close it there or record it as a decision.
- **Awaiting owner review:** `knowledge/profile/planning_keys.yaml`, which is a draft with its `favourable` values chosen by the agent. The owner has accepted its overlap with `det.site_class`, `det.potentially_contaminated_land` and the GFA and site-area scale fields as it stands.
- **Never do the following:**
  - push to the remote (the repository is public);
  - mark anything `reviewed`;
  - accept a new filing baseline to hide a regression;
  - change a budget value (that is the owner's call);
  - commit `.env`, `.claude/` or private corpus content.

## Running the full gate on this Windows host

`tools/check.ps1` fails from an agent's PowerShell, because the npm shim mangles its arguments (F27). Run the same steps from bash instead:

```bash
T="/d/AI Projects/sitewise/.tools"
export PATH="$T/go/bin:$PATH" GOCACHE="$T/go-cache" GOPATH="$T/go-path" GOTOOLCHAIN=local
export SITEWISE_TEST_DATABASE_URL='postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
P="/d/AI Projects/sitewise/data/eval/profile/private"
python tools/check_knowledge.py --strict
npm --prefix web ci && npm --prefix web run build
go test ./...
npm --prefix web run test:e2e
go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
go run ./cmd/profile-eval -cases "$P/source-cases.json" -recording "$P/source-recording.json"   # expect 15/15, 0 forbidden
go run ./cmd/profile-eval -cases "$P/hale-cases.json" -recording "$P/hale-recording.json"       # expect 12/12, 0 forbidden
go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json
```

Gotchas:

- **Postgres.** Postgres 17 runs on port 5433 from `.tools/pgsql-dist`, with its data in `.tools/pgdata`. Start it with `pg_ctl -D .tools/pgdata -o "-p 5433" start`.
- **Worktrees** (one per package, `../sitewise-<wp>`) lack git-ignored inputs. Copy `data/eval/intake/private/` and `.tools/dev-files/` from the main checkout before running the replays (about two minutes).
- **Test database.** When you switch to a branch with different migrations, drop and recreate `sitewise_test`. Otherwise you get 500 errors or failed tests from schema drift.
- **Migration numbers.** 011–022 are reserved per package (plan). A fix outside a package uses a suffix, such as `012a_…`; the runner applies unapplied files in name order.
- **Bench results files.** The bench and replay rewrite `bench/results/latest.json` and `data/eval/intake/results/replay-latest.json`. Commit them as gate evidence, or restore them with `git checkout`.
- **Locked worktree folders.** `git worktree remove` often leaves the folder locked ("Permission denied"). That is harmless, because git no longer tracks it.
- **Component budgets** (`where: release`) are reported, not gated, off the VPS (D-37); user paths gate everywhere.

## Before blaming a package for a speed regression

An intermittent bench failure cost hours on 5 October (F30). It was a query plan that went bad on stale statistics, not the package under test. Slow-query logging alone made it disappear. To reproduce it on purpose:

```sql
-- on sitewise_test, after a bench run (its org is deleted, so the tables are nearly empty)
ANALYZE documents; ANALYZE decisions; ANALYZE jobs; ANALYZE document_sources;
ALTER TABLE documents SET (autovacuum_enabled = false);  -- likewise for the other three
-- run the bench; capture plans with:
ALTER DATABASE sitewise_test SET session_preload_libraries = 'auto_explain';
ALTER DATABASE sitewise_test SET auto_explain.log_min_duration = 100;   -- plans land in .tools/pg.log
-- afterwards: ALTER TABLE ... RESET (autovacuum_enabled); ALTER DATABASE sitewise_test RESET ALL;
```

## Process each package followed

1. Branch `nw/<wp>-<slug>` in a worktree, then implement with tests.
2. Run the full gate.
3. Get an independent review (a fresh agent with the spec and the diff), fix the findings, then run the gate again.
4. Commit the gate evidence and merge with `--no-ff`.
5. Update all three status records: the work packages §6 table, plan §0.2 and the register state updates. Plan §0.2 is where to check where things are up to.
