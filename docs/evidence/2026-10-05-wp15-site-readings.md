# WP-15: site readings and derivation eligibility

Date: 5 October 2026. Lane A. Implemented in the supplied `main` working tree
on base/head `80a5a4244623109b5da70b14859c303a46e056b8`; no commit, merge or
deployment. The owner's request authorizes this package beyond the historical
documentation-only instruction. Independent review and integration acceptance
remain; this note does not mark the package or wider wave Verified.

Contracts: implementation plan §4.1, §4.3 and D-04/D-06/D-32, at plan commit
`ceb67518a03701a7df948fe2f384cce816e0e404`.

## Outcome

Reconciliation now annotates scope and provenance itself. Document readings
use the key-scope registry and the existing site-owned parts. The existing
store writes their site ID, scope and document SourceRefs to `profile_rows`.
Facts remain document-owned; projections are never authoritative input.
The existing one-project-per-site restriction is unchanged.

Code still admits stated user values for table lookups. Assumptions,
requirements, allowances, forecasts, planning values, cleared/unknown values
and superseded values cannot feed them. A determined result is amber with
“Planning only — inputs not verified” unless every required input is explicitly
verified. The status propagates through chained calculations. High model
confidence is not verification. The existing catalogue review and verified
table checks still apply before any result is determined.

The existing `derived` JSON now carries `provenance.inputs[]`, containing each
available rule input's part/key `ref`, `origin` and `review_status`. This survives
database and API serialization, including intermediate calculation references.
Only the rule's declared inputs affect verification; unrelated unverified
values do not. These are projection snapshots, not a new dependency engine or
a user verification endpoint.

`existing_building_year` joins the site profile group and relevant existing
systems; `existing_building` joins classification. Their knowledge status stays
draft. Building year belongs to the site; “work to an existing building” retains
the registry's project classification and its existing owner-review caveat.
Code harvests candidate years from 1800 through the current year and Jev picks
a verbatim candidate or none in the existing label fan-out. There was no year
parser in this checkout despite the brief describing one: the generic numeric
path only harvested areas, so a bounded year parser was necessary.

No new migration, service, library, model, threshold or rule number. No UI,
hazardous-material year threshold, report refresh or export work. User values
remain final; contrary document sources remain available alongside them.

TypeSafe references followed:
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [one fan-out per state](https://docs.typesafe.ai/patterns/fan-out).

## Label cache impact

Compared marshalled `jobs.LabelCall` requests with the pre-WP-15 catalogue
against the final catalogue, using the private frozen Spec Home fixture's
6,742 passages and stored source context/section and document decisions.
There were **zero changed fingerprints, zero added questions and zero added
request bytes** on that fixture. Both standing replay sets also retain their
fingerprints and answer keys. This is not evidence that other projects have
no affected passages.

The targeted synthetic passage “The existing building was built in 1985.”
(report, ordinal 30) adds four questions (two values and their assertions)
to one label request: 10,413 → 12,886 JSON bytes (+2,473 bytes).
Before SHA-256:
`10e49d7edea82f7463ef8ee273cda9cc728614a9398a6726578f6aad0d098424`.
After SHA-256:
`4542aee35d3cb164625ed392b5a9086f442ac3f04bc2934579d1825335ad9629`.
An affected passage therefore needs one label cache miss on the next selected
document update. No second request is added to that state. No live Jev call
was made; token cost and live provider latency for these new questions are
unmeasured. The local comparison helper and before-catalogue copy remain in
ignored `.tools/wp15-measure/` and `.tools/wp15-before-knowledge/`.

## Verification

- Strict knowledge checker: 0 errors, the same 3 existing warnings.
- Full Go suite with the dedicated test database: passed, including the final
  rerun after adding input provenance.
- Profile/source replay: 15/15; Hale replay: 12/12. Zero forbidden readings.
- Intake replay: 218 calls, zero provider errors, zero replay misses; existing
  accuracy gate passed. No answer key or baseline changed. Existing title
  accuracy limitations remain visible in the evaluator output.
- Clean npm install and production build: passed.
- Browser tests: 11/11 passed. Windows left the owned test server running
  after all assertions; stopping that exact executable allowed teardown to
  finish with exit code zero. No unrelated server was stopped.
- Installed OCR gate: filing p50/p90 3,288/3,445 ms; detail recovery
  3,288/3,479 ms, against 10,000/20,000 ms.
- New tests cover chained verification, unrelated inputs, high-confidence
  evidence, clearing/unknown inputs, year boundaries and verbatim copying,
  site/project routing, SourceRefs, input provenance persistence, user
  precedence and wrong-org access. Existing eligibility table tests, source
  replacement, supersession, reading policy and scope tests remain green.

Commands (Windows equivalent of `tools/check.ps1`; npm's Node entry point
avoids the host shim issue; serial Go package tests avoid wall-clock test
contention):

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
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
go run ./.tools/wp15-measure
go run ./cmd/intake-bench -out bench/results/2026-10-05-wp15.json -samples-out .tools/wp15-final-samples.json
```

## Performance

Budgets unchanged: filing 1,000/2,000 ms; profile edit 50/150 ms; full Spec Home
rebuild 100/300 ms. Local provider replay is a timing model, not VPS release
evidence.

The first run is retained as `bench/results/2026-10-05-wp15-first.json`:
filing 345.1/935.4 ms, profile edits 23.0/77.1 ms, rebuild 51.4/341.0 ms.
The rebuild gate failed. Four startup rebuild samples were 722–778 ms and
one later sample was 341 ms; most were 48–57 ms. The cause of those outliers
was not established. No budget, sample filter or database setting was changed.
The final-code rerun passed every enforced user-path gate:

| Path | Samples | p50 / p90 ms | Budget ms |
| - | - | - | - |
| Filing | 368 | 401.154 / 1,212.977 | 1,000 / 2,000 |
| Profile read | 80 | 1.049 / 14.890 | 50 / 150 |
| Profile edit | 160 | 23.515 / 74.980 | 50 / 150 |
| Spec Home rebuild | 40 | 52.910 / 58.489 | 100 / 300 |
| Spec Home HTTP edit | 40 | 91.289 / 98.742 | separately reported under D-18 |

Final aggregate: `bench/results/2026-10-05-wp15.json`. Workload: 184 files,
two rounds, four upload workers, two background callers (220 calls), 10%
injected stalls (62 stalled requests). Spec Home fixture: 38 documents,
1,153 facts and 6,742 passages; SHA-256
`858ce589eeea6cf3dde011d9e393f4cb2428e1dc8f512bc15fed0c67800ee61e`.

Startup outliers remain: final rebuild maximum 796 ms and Spec Home HTTP edit
maximum 4,738 ms. A passing p90 rerun does not establish their cause or fix
them. Identity extraction, deterministic field rules and Jev admission have
component overages reported under the existing D-37 target-VPS policy; no
component result was hidden or budget relaxed. Target-VPS measurements and
investigation of the startup variance remain release/integration follow-up.

## Requirement and acceptance handoff

The following are implemented with local test evidence; integration-lead
acceptance and register state changes remain at merge.

| Requirements | Package evidence and boundary |
| - | - |
| NW-REQ-021, 067 | Replays and existing reconciliation tests preserve factual answers, supersession, user precedence and suggestions; reconciliation remains I/O-free. |
| NW-REQ-027, 086, 100, 104 | Registry scope is applied in reconciliation; database test proves site readings retain document provenance. Existing site-value ownership and isolation tests pass. Multi-project sites remain deferred by the v1 constraint. |
| NW-REQ-082, 378 | Building-year and existing-building harvesting, bounded candidate tests, site row provenance and measured label-call impact. K3 legal thresholds remain out of scope. |
| NW-REQ-176 | `derived.provenance.inputs[]` snapshots input origins and review states; chain and database round-trip tests. |
| NW-REQ-191, 192, 273 | Human decisions remain final; accepted assumptions cannot feed derivations; resets/unknown states recompute. Work-item proposals are WP-20. |
| NW-REQ-237 | Existing authoritative snapshot → rebuild transaction retained; no read of `profile_rows` added to writes. |
| NW-REQ-314 | Standing local checks exercised; performance evidence above. Wider Stage 1 acceptance remains with the integration lead. |
| NW-REQ-383 | Profile values are amber/planning-only for unverified inputs. Draft refresh and export remain WP-42/45. |

AT-05: Hale replay passes; per-part work types belong to WP-21.
AT-09: eligibility, user precedence and existing profile API checks pass.
AT-31: profile projection/API serialization covered; UI, report refresh and
export acceptance remain with their owning packages.

Existing WP-14 and independent Jev/worker changes in this dirty checkout were
preserved, including the pre-existing HANDOFF deletion and `.claude/` files.
This task does not claim them. No clean-worktree integration or live/VPS run
was performed. Future report code should consume `derived.provenance` and
the row's trust state rather than inferring verification from confidence.
