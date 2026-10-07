# WP-20: work items and scope-picker cut-over

Date: 5 October 2026. Lane A. Code implemented in the supplied `main` working
tree, base/head `80a5a4244623109b5da70b14859c303a46e056b8`. No merge or deployment.
The current owner request authorizes implementation beyond the historical
documentation-only planning instruction. Independent review and integration
acceptance remain; this is not a claim that the whole wave is Verified.

Contracts: implementation plan §4.5, §4.12 and D-05/D-07/D-08/D-18, at plan
commit `ceb67518a03701a7df948fe2f384cce816e0e404`.

## Outcome

`work_items` is now the sole authority for scope. The existing picker writes
user-owned, accepted items; the profile response projects its existing
`scope.<system>` rows from these items. Evidence and scope defaults maintain
only untouched proposals. Unsupported proposals retire without losing their
deterministic UUIDv5 identity; returning support revives the same item. A user
exclusion survives evidence, retries and concurrent picker writes.

The profile rebuild computes desired proposals from authoritative profile
inputs, updates work items, then fingerprints and projects the actual items,
all in the same transaction. It never reads profile rows to infer scope.
GET/POST project works endpoints use the existing authentication, origin
checks, project locks and org/site/part ownership checks. Writes emit a
revisioned `works` event and rebuild the profile before committing.

Targets accept registered typed keys and exact reviewed clause versions only.
The optional clause catalogue loader establishes that validation boundary;
this package does not author or approve report clauses. Existing condition is
stored as the site value `sys.<system>.condition`, read through by work items.
A conflicting existing user value must be edited through its versioned profile
endpoint; work-item creation cannot silently replace it. The item retains only
its short condition note. Unrecognized/deprecated systems cannot be newly
created; existing items with removed/deprecated IDs remain visible and flagged.

There is no new service, dependency, model call, threshold, rule number or UI.
Part work type overrides the project default action. Unknown work type defaults
conservatively to investigation. Existing-building presence routing is WP-21;
action questions, split/edit/retire and proposal acceptance are later packages.

## Migration

Migration 015 creates the table, ownership FKs, coarse/proposal uniqueness,
action/inclusion/provenance constraints, quantity/unit pairing and explicit
verification requirements. The Go backfill runs inside the migration's same
transaction, using only the standard library for UUIDv5. It maps legacy `in`
and `out` choices, checks counts, owners and inclusions, deletes those scope
values, and adds a constraint preventing a second scope authority.

An isolated-schema test restores the pre-015 schema, seeds legacy choices and
other profile values, applies 015, and checks exact preservation. An unexpected
legacy value causes rollback with all old choices intact. The test schema is
itself rolled back. This is not the required production-backup dry run: a
restored-backup migration check remains a pre-merge gate, and a fresh production
backup and production authorization remain required before deployment.

## Verification

- Full Go suite with the dedicated test database: passed.
- Strict knowledge checker: 0 errors, 3 existing warnings.
- Clean npm install and production build: passed.
- Playwright: 11/11 passed, including the unchanged scope picker. The exact
  owned Windows e2e server was stopped after all assertions to finish teardown;
  the runner exited zero.
- Source replay 15/15; Hale 12/12; no forbidden applied readings. No factual
  answer key changed. Intake replay: 218 calls, 0 errors and replay misses;
  standing accuracy gate passed. Existing title-quality limitations remain.
- Installed OCR: filing p50/p90 3,235/3,385 ms and detail recovery
  3,232/3,382 ms, against 10,000/20,000 ms.
- Migration rollback, duplicate conflict, cross-org/project ownership,
  transactional rebuild rollback, concurrent exclusions, stable revival,
  user precedence, site condition read-through, target type validation and
  exact reviewed clause validation have automated coverage.
- Full workload benchmark passed. p50/p90 in milliseconds: profile edit
  34.5/113.1 (50/150 budget), profile rebuild 66.3/72.3 (100/300), works read
  1.1/1.7 (100/250), works write 8.0/10.8 (100/250). Filing round 1 was
  327.9/767.8 and round 2 323.0/628.4 (1,000/2,000). Exact measurements,
  sample counts and every component gate are in
  `bench/results/2026-10-05-wp20.json`. This is local replay timing, not
  target-VPS or live-provider release evidence. Benchmark ran the compiled
  WP-20 code before WP-21 source edits.

Two existing development-host component budgets remain over: identity text
extraction p90 314.2 ms versus 250 ms, and deterministic field rules p90
3.0 ms versus 1 ms. The benchmark reports these separately for target-VPS
judgement; the endpoint/end-to-end development gate passes. This is not a
claim that every release component budget passes.

Commands use the same checks as `tools/check.ps1`, with npm's Node entry point
for the host shim and `-p 1` to avoid concurrent wall-clock Go test contention:

```powershell
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web ci
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run build
go test -p 1 ./...
python tools/check_knowledge.py --strict
pwsh -NoProfile -File tools/check-ocr.ps1
node 'C:/Program Files/nodejs/node_modules/npm/bin/npm-cli.js' --prefix web run test:e2e
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
go run ./cmd/intake-bench -out bench/results/2026-10-05-wp20.json -samples-out .tools/wp20-samples.json
```

## Traceability and remaining gates

Implemented in code, awaiting integration/review: NW-REQ-028, 029, 035, 046,
048 (creation/ownership portion), 067, 069, 070 (storage portion), 084, 085,
089, 099, 105, 106, 107, 108, 109, 112, 113, 114, 115 (storage/default portion),
119, 129 (user precedence), 191, 237, 241, 242, 273.

Partial, with later packages still required: NW-REQ-005, 006 and 315 (full
project spectrum and Stage 2). AT-16/AT-22 scope and migration coverage passes;
AT-28 is partial. No register row is marked Verified by this handoff.

Production failure modes are fail-closed: invalid legacy choices abort migration;
unknown input or target types return validation errors; duplicate coarse items
return 409; wrong owners return 404; rebuild failure rolls back the write and
event. Live provider and target-VPS timings are not measured by this package.
