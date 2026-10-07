# Publication checkpoint — 7 October 2026

The owner explicitly requested committing the current publishable changes on
main and pushing to GitHub with the known incomplete gates. This checkpoint is
not M1 acceptance or deployment approval. No new dependency was added by the
publication work. Private corpus/recordings, secrets, local agent worktrees and
raw machine-local timing samples remain excluded.

The owner selected Petersham `03 ANX Q PPR` for broader acceptance testing.
See the implementation plan's current owner direction and Petersham scope.
0991 remains a bounded regression fixture, not the principal acceptance case.

## Validation

- `python tools/check_knowledge.py --strict`: 0 errors, 2 warnings about unmapped
  contamination_level and flood_exposure defaults.
- `npm.cmd --prefix web ci` and `npm.cmd --prefix web run build`: passed.
- `tools/check.ps1` was invoked with npm forwarded to npm.cmd because this
  Windows environment's npm.ps1 invocation failed. Go tests then passed except
  `TestWorkCorpusFailsClosed/valid` (temporary corpus root unavailable).
  `go test ./internal/eval -count=1` passed with TEMP/TMP set to workspace tmp.
  Thus no single full-gate pass is claimed.
- Intake replay: passed, 218 calls, zero replay misses; this does not establish
  satisfactory overall title quality.
- Source and Hale profile replays: fail, stale bankstown-page9 and
  hale-description recordings. No private passages were transmitted to refresh
  them in this task.
- Playwright: all 22 test cases reported passing. The runner did not exit after
  the final case and was interrupted during teardown; no clean suite-exit pass
  is claimed.
- OCR installed-runtime gate: passed; filing p50/p90 3265/4282 ms and detail
  recovery 3251/4074 ms against 10000/20000 ms, including polling allowance.
- Intake benchmark: filing p50/p90 420.1/1022.1 ms against 1000/2000 ms.
  Profile edit 50.446/114.4 ms against 50/150 ms and Spec Home edit
  67.960/103.7 ms against 50/150 ms: median budgets failed. Rebuild
  38.5/71.6 ms against 100/300 ms passed. Intake component overruns were
  reported under the existing target-VPS-only judgement policy.
  These are local diagnostic timings, with other validation work overlapping;
  they do not resolve the existing intermittent latency or prove release speed.
- Pattern-based secret scan found no candidate credentials in publishable files;
  outgoing historical filenames contained no private/secret path matches.
- Two web files had malformed CR-CR-LF line endings normalized; no runtime
  behavior changed by that cleanup.

Automatic proposal generation remains off. Product decisions, reviewed expected
outcomes and knowledge, fresh approved Jev evidence, consistent performance,
target-host checks and release approval remain outstanding.
