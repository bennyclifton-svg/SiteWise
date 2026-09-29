# Handoff — 29 September 2026, knowledge merge complete

Read `AGENTS.md`, `docs/design/2026-09-29-foundation-design.md`,
`knowledge/SCHEMA.md`, then `docs/plans/2026-09-29-instant-intake.md`.

## Completed

- Recovered the interrupted extraction and completed the relevant seed sweeps.
  All six clusters have reports with full-file inventories, exclusions,
  contradictions and residual technical gaps.
- Merged **143 systems, 66 determinants, 226 rules, 140 interfaces and
  172 failure modes**. All records remain draft; only the owner marks reviewed.
- Folded proposals into `knowledge/determinants.yaml`, reconciled duplicate
  interfaces and resolved references. Canonical IDs: `knowledge/MERGE.md`.
- Read primary NCC 2022 adopted Volume One C2/C3/C4. Corrected construction-type
  and compartment-limit seed errors, rejected the generic sprinkler downgrade,
  and corrected the service-opening clause locator.
- Wrote **2 primary-verified baseline tables**: `type_of_construction` and
  `compartment_limits`. Their scope/exclusions are essential. See
  `knowledge/clusters/determinants/VERIFICATION.md`.
- Marked **9 distinct missing tables / 11 derivations pending**. No seed-only
  numeric table is executable. Kept provided fixture count separate from the
  computed required count. Explicit unknown evidence never becomes false.
- Strengthened the interim checker and added 6 regression tests.
- Installed checksum-verified **Go 1.27.1** locally in `.tools/go/`; version
  and compile/run smoke test passed. The installation is ignored by Git.
- Selected **fire** as the background pilot, including water, power, structure
  and smoke-control interfaces. Wrote the intake implementation plan using the
  locally available `superpowers:writing-plans` skill.

## Verified at handoff

```powershell
python tools/check_knowledge.py
python tools/check_knowledge.py --strict
python -m unittest discover -s tools -p 'test_*.py' -v
```

Both knowledge checks: **0 errors, 0 warnings**. All six tests pass.
`git diff --check` passed. Changes were committed per domain cluster, with lead
validation/reconciliation and planning commits. No application runtime or
deployment has been implemented in this continuation.

## Next work

Execute `docs/plans/2026-09-29-instant-intake.md` from Task 1. It specifies
files, acceptance tests, Jev citations, latency gates, tenant isolation,
durable upload/SSE/jobs, parser spike, calibration and operational readiness.

For PowerShell, add the repo-local compiler to this shell only:

```powershell
$env:PATH = "$PWD/.tools/go/bin;$env:PATH"
$env:GOCACHE = "$PWD/.tools/go-cache"
$env:GOPATH = "$PWD/.tools/go-path"
$env:GOTOOLCHAIN = 'local'
go version
```

Confirm PostgreSQL 17 and Node availability before installing them. Corpus
answer keys and the 57-discipline data are located in the frozen `../clerk`
reference; actual corpus paths and hashes still need verification. Some labels
are unreviewed/model-derived and some register revisions post-date the files;
the plan accounts for this. The exact 16-kind taxonomy needs reconciliation
from the reference data before calibration.

No live Jev evaluation, confidence thresholds, Go app tests, end-to-end intake
test, live latency result or restore rehearsal exists yet. These are explicit
implementation gates, not implied by the knowledge checker passing.

## Boundaries to preserve

- Jev is the only runtime AI; pin `jev-1.13.0`, never an alias. Code controls
  flow, parses candidates, compares revisions and computes lookups.
- One fan-out per state. Intake gate: p50 ≤ 1 s, p90 ≤ 2 s after upload.
- Go single binary, embedded React, PostgreSQL 17, local hash-addressed files,
  Caddy/systemd. No stack change without a stated reason.
- No old-app benchmark and no Supabase backup. Future clean VPS replacement
  carries over only secrets/keys; this continuation did not rebuild the VPS.
- Primary Standard tables, state adoption and specialist engineering details
  remain gaps. `pending` means unknown; a draft table is not owner review or
  project compliance. Never use a baseline table outside its recorded scope.
- The owner stopped the earlier Claude extraction workers after overlapping
  writes were detected. This continuation's agents finished and stopped writing.
- Copy only data from `../clerk`, never code or PM doctrine.
