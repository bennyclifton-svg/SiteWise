# Next wave: agent work packages

Status: revised plan, 5 October 2026. **Documentation-only; do not start or resume implementation.** The implementation plan §0.2 is the current evidence summary and §5 is the current decision table. Historical package states are not fresh verification. M1/M2/M3 replace the original stage-only execution order.

Read with:

- **Requirements**: `docs/plans/2026-10-04-next-wave-requirements-register.md` (`NW-REQ-###`).
- **Architecture, schema, decisions, budgets, scenarios**: `docs/plans/2026-10-04-next-wave-implementation-plan.md`. The ownership model is §3, the logical schema §4, decisions `D-##` §5, budgets §8.3 and acceptance scenarios `AT-##` §8.4.
- **Source PRG** (original line-numbered baseline plus review amendment): `docs/plans/2026-10-04-next-wave-architecture-schema.md`.
- **Binding repo rules**: `AGENTS.md` and `docs/developing.md`.

Shared schemas and contracts are **designed in the plan, not by implementing agents**. An agent implements the plan's §4 and §3.4-3.6 for its tables. If the design does not work, the agent stops and records a question (§4.3). It does not redesign.

## 1. Roles

| Role | Who | Does |
| - | - | - |
| Owner | Benny | Decisions `D-##`, knowledge review (`status: reviewed`), answer-key review, live Jev spend, VPS measurement sign-off |
| Integration lead | One lead session per stage | Assigns packages, holds the migration-number and file-ownership tables (§2), merges in order, runs the full gate after each merge, checks traceability both ways (§3) |
| Implementing agent | Any coding agent (Claude, Codex) | One package at a time, in a worktree branch `nw/<wp-id>-<slug>`; ends with a handoff (§4) |
| Reviewer | A different agent or person from the implementer | Reviews against this document and the register before merge; Lane C packages always get a security review |
| Research agent (K1-K4) | Separate from implementers | Writes draft YAML only; never touches Go, migrations or web |

## 2. Coordination

### 2.1 Migration numbers

Migrations live in `internal/db/migrations`. They are applied in name order, in one transaction (`internal/db/migrate.go`).

| No. | File | Package | Contents (plan section) |
| - | - | - | - |
| 011 | `011_sites.sql` | WP-11 | §4.2 |
| 012 | `012_value_provenance.sql` | WP-12 | §4.3 user values and rows |
| 013 | `013_planning_values.sql` | WP-13 | §4.3 planning values |
| 014 | `014_revisions.sql` | WP-14 | §4.4 |
| 015 | `015_work_items.sql` | WP-20 | §4.5 plus the scope backfill (§4.12) |
| 016 | `016_proposals.sql` | WP-26 | §4.7 |
| 017 | `017_packages.sql` | WP-30 | §4.8 `packages`, `package_stages` |
| 018 | `018_package_scope.sql` | WP-31 | §4.8 `package_scope_items` |
| 019 | `019_delivery.sql` | WP-35a | §4.9; final schema, minimal controls at M1 |
| 020 | `020_reports.sql` | WP-41a | §4.11 (`report_references` included); M1 draft, issue disabled |
| 021 | `021_cost_plan.sql` | WP-33 | §4.10; M2, after M1 review |
| 022 | `022_adoption_events.sql` | WP-45 | D-21 |

Rules:

- A number is used only by its package. A package never edits another package's migration after merge; a fix gets the next free number, assigned by the integration lead.
- Migrations still merge in number order. Reservations 019–021 were reordered for M1; none existed at this planning revision. Recheck before coding and never rename an applied migration. WP-33 waits for report migration 020 even though its cost FKs require only 017/018.

### 2.2 File ownership

The owner of a file is the only package that may restructure it. Other packages may add only the items listed in the "Shared append" column.

| File or area | Owner | Shared append (others may add) |
| - | - | - |
| `internal/store/sites.go` | WP-11 | none |
| `internal/store/profile.go` | WP-12 | WP-13 (planning reads), WP-14 (build revision), WP-15, WP-20 (rebuild hook) |
| `internal/profile/*.go` | WP-15 | WP-20, 21, 22, 23 (separate files `works.go`, `location.go`) |
| `internal/jobs/worker.go`, `source_reading.go` | WP-22 | WP-23 (location question), WP-27 (signals); these three run **sequentially** |
| `internal/knowledge/load.go` | WP-25 | WP-14 (hash function in a new `hash.go`) |
| `internal/works/*` | WP-20 (create), WP-26 | WP-24 |
| `internal/procurement/*` | WP-30 | WP-31, 32 |
| `internal/costs/*` | WP-33 | WP-34 |
| `internal/delivery/*` | WP-35 | WP-60 |
| `internal/reports/*` | WP-41 | WP-42, 43, 44, 45, 50, 51, 60 (templates in `internal/reports/templates/`) |
| `internal/httpapi/projects.go` route table | integration lead | every package adds its routes (one line each) |
| `internal/httpapi/<domain>.go` | the domain's package | none |
| `bench/budgets.json`, `internal/latency/gate_test.go`, `cmd/intake-bench/main.go` | integration lead | each package appends its own path entries |
| `data/profile/thresholds.json` | WP-22 | WP-23 (`location.n*`) |
| `knowledge/SCHEMA.md`, `tools/check_knowledge.py` | WP-K0 (all shapes and checker rules, including `key_scope.yaml` and `planning_keys.yaml`) | WP-K4 merge pass (no shape changes) |
| `knowledge/clusters/<c>/*.yaml` | WP-K1, K2, K4 per cluster | WP-K3 (`rules.yaml`, `consequences.yaml` in its clusters) |
| `data/eval/profile/*` | WP-28 | WP-00 (recordings) |
| `web/src/Profile.tsx`, `ScopePicker.tsx` | WP-70 | WP-20 (no UI change expected) |
| `web/src/Report*.tsx` | WP-45 | WP-50, 51 |

### 2.3 Shared contracts

Defined once in the plan:

- write-path contract: §3.4
- events: §3.5
- staleness: §3.6
- `SourceRef` and the provenance columns: §4.1
- error codes 404, 409 and 422: §3.4

A package that changes a shared contract is out of scope. It must stop and ask (§4.3).

### 2.4 Milestones, phase ownership and merge order

The owner has requested plan edits only. This is the future sequence, not a direction to run agents now. Full gates remain mandatory before each future merge; M1 is an additional usefulness gate, not a waiver.

| Order | Work | Gate / deferred work |
| - | - | - |
| M0 | WP-00 follow-up | Whole-file replay restored; component and end-to-end gates reported; AT-35. |
| M1 foundations | WP-K0; WP-11 → 12 → 13 → 14 → 15; WP-25 after K0 | Same final tables and scope registry; relevant K1–K3 records and WP-40a may proceed alongside. |
| M1 scope | WP-20 → 21 → 22 → 23; WP-26 after 14/20/25; WP-28a from 20 | 0991 frozen owner-reviewed work-item and critical-obligation keys before scoring. WP-24 subdivision and WP-27 signals deferred. |
| M1 responsibilities | WP-30 → 31 → 32; WP-35a after 30; WP-40a before 31 | WP-31 no longer waits for split operations in WP-24; it supports existing coarse items. Final delivery schema migration 019, minimal date/risk/approval entry only. |
| M1 draft | WP-41a → 42a → 45a, with WP-K6a and WP-X1 | WP-41a needs 14/26/30–32/35a/40a. Migration 020 is the final report schema. No cost query or issue/export endpoint until M2. §8.6 and AT-31–34 gate M1. |
| M2 | WP-24, 27, remaining 28/35/40; WP-33 → 34; WP-41b → 42b → 43 → 44 → 45b | Migration 021 adds costs after reports; 022 analytics is optional, not needed for stopwatch measurement. M2 issue/export and full RFP gates pass. |
| M3 | WP-K6b on 0777 → WP-50 → 51 → 60 → 70 → 71 | 0777 quality gate before broader reports; target-VPS, isolation and restore evidence before deployment. WP-X1 after M1, M2 and before release. |

An a/b phase is a bounded part of the existing package under the same file owner. Its handoff lists the requirements actually met and the remainder, never marks the whole package Verified from a partial delivery. K4–K5 and wider K1–K3 research can continue independently; unreviewed data cannot become approved obligations.

- **WP-28a / K6a:** 0991 keys and relevant knowledge walkthrough. **WP-28b / K6b:** 0777 scored case and broader regression completion.
- **WP-35a:** final schema and minimal manual dates/risks/approvals used by the RFP; remaining delivery views/dependencies in M2.
- **WP-40a:** only 0991 RFP clauses/template; b completes RFT/PMP catalogues.
- **WP-41a/42a:** final report store, shared draft assembler, citations, protected edits and stale/conflict handling. Optional cost data is explicitly unavailable; never invent money or import a second ledger. b adds cost-backed sections, pricing and remaining report support.
- **WP-45a:** minimal work-item/proposal/responsibility review controls plus RFP draft/edit/refresh; show “Draft — not for issue”. Measure with a stopwatch and evidence note, not a new analytics dependency. **WP-45b:** full issue/export workflow and original issuable-RFP adoption measurement.

AT-03–06 and AT-27 remain standing Lane A gates. The full check is `pwsh tools/check.ps1` with the test database up; any platform-specific equivalent must run all the same steps and record why. The lead owns shared routes, budgets and migration reservations, runs the full check after every merge and resolves traceability both ways. No merge with a red required gate; prior merges while gates were incomplete are history, not a precedent.

### 2.5 K-track separation

| Step | Research output | Review | Integration |
| - | - | - | - |
| K1-K4 | Research agent writes draft YAML and a merge note in `knowledge/clusters/<c>/REPORT.md` | A different agent runs `check_knowledge.py --strict` and spot-checks 10% of new records against their cited source (seed anchor or dataset row) | Integration lead merges per cluster; owner reviews later; nothing becomes `reviewed` without the owner |
| K3 | Strongest model; primary instruments only | Owner | As above; `clause_verified` stays `false` until the instrument is read |
| K5 | Owner interview notes | Owner | Lead session writes draft records |
| K6 | Lead session walk-through report | Owner sets the kept share | Misses become records through K1-K4 shapes |

## 3. Traceability check at each merge

The integration lead confirms, for each package:

1. Every `NW-REQ` listed in the package appears in the register with that package in its WP column, and the reverse holds.
2. The handoff (§4) gives a status for each listed requirement: Implemented or Verified, with evidence.
3. The register's Disp. / State cells for those requirements are updated **by the integration lead**, not the implementing agent. This is the only edit to the register allowed during implementation.

## 4. Handoff and stop rules

### 4.1 States

Packages and requirements move **Not started → In progress → Implemented → Verified**.

- **Implemented**: merged with tests written.
- **Verified**: §4.2 evidence recorded and accepted by the reviewer.
- Code written, or a test written, is not verification.

### 4.2 Handoff (required at the end of every package)

```text
WP: <id>   Branch: <name>   Base commit: <sha>   Head commit: <sha>
Contracts: plan §<n> version as of commit <sha of plans/2026-10-04-next-wave-implementation-plan.md>
Requirements: NW-REQ-xxx … each with Implemented | Verified | Not done (+ reason)
Changes: plain language, what calls what, what could fail in production
Migrations: <file> (assertions: …)
Dependencies added: none | <name, version, why the stack cannot do it, who else watches it>
Verification run (commands verbatim, with result lines):
  python tools/check_knowledge.py --strict  → …
  go test ./internal/... (SITEWISE_TEST_DATABASE_URL set) → …
  go run ./cmd/profile-eval → …     (and the Hale replay)
  bench paths → p50/p90 against budget (local replay = timing model, not release evidence)
Not run and why: …
Acceptance scenarios touched: AT-xx → pass | fail | not run
Deviations from the plan: … (each needs integration-lead or owner approval)
Open questions: Q-<WP>-n …
Next agent needs to know: …
```

### 4.3 Stop rules

- On a material ambiguity, or a conflict with the PRG, plan or register, **stop only the affected step**. Record it as `Q-<WP>-n` in the handoff and continue the unaffected steps. Never silently change a requirement, a decision or a shared contract.
- If a speed budget cannot be met, stop and report what would have to be traded (`docs/developing.md`). Do not add a round trip, retry or heavier library.
- **Recheck HEAD** at start (L399). If `main` moved past the package's base commit in a file you touch, rebase first and re-run the gate.
- Never commit private-corpus content or personal details. Never mark knowledge or answer keys `reviewed`.

## 5. Packages

Every brief below is meant to be pasted into a fresh agent session. Each one assumes the agent first reads `AGENTS.md`, `docs/developing.md`, this package section, the plan sections named in it, and the listed register rows.

---

### Review amendment applied to package briefs

These obligations override old stage labels and copied run-sheet prerequisites; use §2.4 for phases. No package starts in this documentation-only turn.

- WP-00 owns a separately scoped prerequisite repair: record every sheet the app files, prove replay completeness (AT-35), retain unreadable/failure cases, and report title quality and component failures explicitly. Do not accept a new accuracy baseline to hide a regression. Broader title improvement remains a separate defect; source metadata in the RFP must be reviewed/corrected before M1/M2 acceptance.
- WP-15/42/45 own D-32/AT-31; change presentation expectations only, never weaken factual answer keys. Unknown or unverified values cannot appear verified through colour or prose.
- WP-31/32 own D-33/AT-32. One explicit design role may be held by services or works; test both owner-PM and D&C cases, no/duplicate designer and supply-only rejection. WP-45/50 wording follows those assignments.
- WP-22/23/27 own D-36/AT-33: measured request preflight and oversized-call failure handling, atomic questions, per-question/per-option-count calibration, zero new hot-path calls; cite TypeSafe fan-out, confidence and model/API limits. No new service or AI provider.
- WP-26/28/45 and WP-K6a/b own §8.6/AT-34 usefulness and critical-recall gates. WP-K6 also serves NW-REQ-385. Critical proposals remain visible even beyond the default ten. Scored keys are frozen before tuning; owner review is mandatory.
- WP-71 starts representative target-VPS measurement before any release candidate is called ready, and retains the final full-load run. WP-X1 now sweeps M1 routes, M2 and release. Neither is deferred merely because a draft demo passed.
- Full package completion still includes original requirements: the phase split does not remove split/retirement, costing, pricing, export, immutable issue or failure handling.

### WP-00: Restore the profile regression gate and measure the evidence workload

- **Lane / stage / state:** A / gate before Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-118, 131, 272, 314, 330, 368, 369, 381, 388.
- **Source wording:** "the existing answer keys must still pass" (L152); "record call count, tokens and elapsed time" (L171); "reserve foreground filing capacity" (L355); "Preserve existing filing p50 ≤1 s/p90 ≤2 s" (L409).
- **Behaviour:** both profile replays pass at HEAD; the evidence-call size is known; filing is unaffected under background reading.
- **Prerequisites:** D-01. D-17 for step 3; owner consent for any live Jev recording.
- **Current evidence:**
  - Executed: `go run ./cmd/profile-eval` → `bankstown-page9: evidence recording is stale`. The Hale replay → `hale-description: evidence recording is stale`. At `02094db` the source replay gives 15/15 (plan F01).
  - Static: detector loading in `internal/knowledge/load.go:311-330`; call building in `internal/jobs/worker.go:500-545`; fingerprint in `internal/jobs/source_reading.go` `cachedAsk`.
- **Files:** `cmd/profile-eval/main.go` (diagnostic flag only, if needed); `docs/evidence/2026-10-xx-evidence-workload.md` (new). Depending on D-17: `data/eval/profile/private/*-recording.json` (re-record, private, never committed if gitignored; check), or `internal/jobs/worker.go`, `internal/knowledge/load.go` (routing).
- **Contracts:** none new.
- **Steps:**
  1. Reproduce F01. Diff the evidence `jev.Call` for `bankstown-page9` and `hale-description` between `02094db` and HEAD (`git worktree add` in a temporary directory, or `git archive`). List the added question IDs.
  2. Measure questions per evidence call, input tokens (from the recordings) and estimated tokens at HEAD, over the source and Hale cases. Write the evidence note.
  3. Apply D-17:
     - (a) re-record live with owner consent; or
     - (b) move failure-mode detector questions into a separate cached stage so the profile evidence call regains its prior questions, with a test proving the fingerprints equal `02094db` for those cases.
  4. Run the bench with background reading active and confirm `whole_intake` p50/p90.
- **Edge cases:** a recording containing private text must stay in the private path. Jev 402 or timeout during re-recording → stop and report. Detectors moved under (b) must still be asked for every labelled passage (no silent loss): count test.
- **Acceptance:** both replay commands exit 0 with readings correct and applied, and 0 forbidden. Evidence note committed. `whole_intake` within 1,000/2,000 ms on the local replay (timing model).
- **Commands:**
  - `go run ./cmd/profile-eval`
  - `go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json`
  - `go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json`
- **Non-goals:** changing any question wording, thresholds or knowledge content; calibrating thresholds.
- **Completion evidence:** handoff with the diff of question IDs, the measurement table and replay outputs.

```text
Feature: Make the profile regression gate pass again at HEAD and measure what the K4 knowledge merge did to the evidence Jev call.
Why: Answer keys are the trust gate for every later Lane A change (NW-REQ-118); it is red on main today.
Lane: A. Package WP-00 in docs/plans/2026-10-04-next-wave-agent-work-packages.md. Decision D-17 in docs/plans/2026-10-04-next-wave-implementation-plan.md §5 must be recorded before step 3.
Out of scope: no new service, dependency or AI; no question wording or threshold change; no knowledge edits.
Speed: filing whole_intake p50 ≤1 s, p90 ≤2 s must hold with background reading running.
Correctness: done = both profile-eval replays green, evidence note written with questions/call, tokens, elapsed. Edge cases: private text stays private; no detector silently dropped.
Security: org scoping unchanged.
After: follow §4.2 handoff; report numbers against budget; stop per §4.3 on ambiguity.
```

---

### WP-11: Sites and site-owned parts

- **Lane / stage / state:** A / Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-025, 026, 042, 043, 046, 065, 086, 096, 097, 098, 099, 102, 230, 232, 233, 314.
- **Source wording:** "A site is one address or campus. Parts belong to the site … gain `roof` and `plant_area`" (L134); "Version 1 creates one site per project, including existing projects, so nothing visible changes" (L138); "Use composite foreign keys … service checks alone are insufficient" (L322).
- **Behaviour:** every project has a site. Parts are site-owned. The UI and API are visibly unchanged. The database rejects cross-org and cross-site parts.
- **Prerequisites:** D-01, WP-00 green.
- **Current evidence:** `project_parts` in `008_project_profile.sql`; `ensureWholePart`, `CreatePart` and `UpdatePart` in `internal/store/profile.go`; part routes in `internal/httpapi/profile.go:629-720`; project creation in `internal/httpapi/projects.go` `createProject`.
- **Files:** `internal/db/migrations/011_sites.sql`; `internal/store/sites.go` (new); `internal/store/profile.go` (part queries move to `site_id`); `internal/httpapi/projects.go` (create site with project); `internal/httpapi/profile.go` (kinds); tests in `internal/store/sites_test.go` and `internal/httpapi/profile_test.go`.
- **Contracts introduced:** plan §4.2. `GET /api/projects/{id}` adds `site_id`; `PATCH /api/sites/{site}` (label, address, lot, version) is added with the §3.4 rules.
- **Steps:**
  1. Write failing DB tests: cross-site part rejected; backfill gives one site per project; whole part unique per site; kinds `roof` and `plant_area` accepted.
  2. Migration per §4.2 with assertion block.
  3. Store: `CreateProject` creates the site in the same transaction; part queries are keyed by site; `ensureWholePart` per site.
  4. API: kinds; `PATCH /sites/{site}` with version check and `projects` revision.
  5. Run the full gate.
- **Edge cases:** a project with no parts yet; duplicate part label on the site → 409; wrong-org site ID → 404; migration on an empty database; rerunning the migration on a restored database (deterministic site ids).
- **Acceptance:** AT-22 (site variant); existing profile tests unchanged; `project_profile_read` and `profile_edit` 50/150.
- **Commands:** DB tests with `SITEWISE_TEST_DATABASE_URL`; `pwsh tools/check.ps1`.
- **Non-goals:** moving values to site scope (WP-12); a multi-project site UI.
- **Completion:** handoff plus migration assertion output.

```text
Feature: Every project gets a site; parts belong to the site; nothing visible changes.
Why: The site is the lasting building record (NW-REQ-025); later facts key to it.
Lane: A. WP-11, schema docs/plans/2026-10-04-next-wave-implementation-plan.md §4.2, write contract §3.4.
Out of scope: value re-scoping, UI changes, new dependencies.
Speed: project_profile_read and profile_edit stay p50 50 / p90 150 ms; PATCH /sites 100/250 ms.
Correctness: done = migration 011 with assertions, composite FKs reject cross-site/org parts, existing profile tests unchanged. Edge cases: empty project, duplicate label, wrong org, rerun on restored DB.
Security: every row org-scoped; 404 never leaks existence.
After: §4.2 handoff, timings vs budget; stop per §4.3.
```

---

### WP-12: Value scope, provenance columns and version checks

- **Lane / stage / state:** A / Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-021, 027, 045, 058, 065, 069, 086, 100, 101, 175, 178, 179, 180, 181, 182, 183, 184, 230, 231, 234, 235, 236, 237, 291, 314, 370.
- **Source wording:** "Origin, review status, meaning and structured provenance; preserve existing keys, bands and null/reset behaviour. Site-level keys … against the site part; project-level keys against the project" (L328); "Preserve document revision/hash and source location, not merely a mutable passage ID" (L274).
- **Behaviour:** user values carry origin, status and meaning, and are site- or project-scoped. Stale versions are rejected with 409. Row sources carry `SourceRef`. The page looks the same.
- **Prerequisites:** WP-11; WP-K0 (the `key_scope.yaml` shape and checker rule, documented in `SCHEMA.md` first: NW-REQ-262); D-04 (classification approved); D-06 for the default origin of user writes.
- **Current evidence:** `profile_user_values` and `profile_rows` (migration 008); `SetUserValue`, `SetScope`, `readSnapshot` and `RebuildProfile` (`internal/store/profile.go`); `putProfileValue` (`internal/httpapi/profile.go:411-478`); F09, F10.
- **Files:** `internal/db/migrations/012_value_provenance.sql`; `knowledge/profile/key_scope.yaml` (new data, draft; shape and checker rule come from WP-K0); `internal/knowledge/profile.go` (registry load); `internal/store/profile.go`; `internal/profile/reconcile.go` (`Source` → `SourceRef` fields); `internal/httpapi/profile.go` (`version`, `origin`, `meaning` in the body; 409).
- **Contracts introduced:** plan §4.1 and §4.3. `PUT /projects/{id}/profile/{key}` body gains optional `version`, `origin` (`user` | `assumption`) and `meaning`; the response is unchanged plus new cell fields.
- **Steps:**
  1. Tests first: registry covers every key the catalogue can emit; stale version → 409; null maps to `cleared`; reset deletes; site value visible to the project; wrong-site part rejected by the FK.
  2. Write the registry YAML (data only).
  3. Migration with backfill and assertions.
  4. Store and API.
  5. `SourceRef` enrichment in `readSnapshot` (join documents, files, passage_sources).
  6. Answer-key and replay gate.
- **Edge cases:** an unclassified key at startup → fail fast with the key name; a client without `version` → accepted (back-compatible) but logged (INT; Q to owner if they want it mandatory); a document deleted while a value cites it.
- **Acceptance:** AT-08, AT-19, AT-22; both replays and answer keys unchanged; `profile_edit` 50/150.
- **Non-goals:** planning values (WP-13); eligibility (WP-15); UI marks (WP-70).

```text
Feature: Profile values record origin, review status, meaning and site/project scope; stale edits are refused.
Why: Trustworthy provenance and the lasting site record (NW-REQ-234-236); fixes last-write-wins (F10).
Lane: A. WP-12; schema plan §4.1, §4.3; D-04 must be recorded (registry classification), D-06 for defaults.
Out of scope: planning values, eligibility rules, UI redesign, dependencies.
Speed: profile_edit and project_profile_read p50 50 / p90 150 ms.
Correctness: done = migration 012 + registry + 409 on stale version + SourceRef in sources; answer keys and both replays unchanged. Edge cases: unclassified key, missing version, deleted source document, wrong site.
Security: composite FKs; org from session only.
After: §4.2 handoff; stop per §4.3.
```

---

### WP-13: Planning values, explicit unknowns and starting assumptions

- **Lane / stage / state:** A / Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-006, 007, 008, 065, 086, 175, 176, 178, 182, 183, 185, 186, 187, 188, 189, 190, 238, 239, 314.
- **Source wording:** "`profile_planning_values` … stores assumptions and calculated values … No duplicate ledger totals" (L329); "Accepted assumptions remain assumptions. Structural adequacy, ground conditions and compliance must not be defaulted as favourable" (L276).
- **Behaviour:** the user, or code, records an assumption or a calculated value with range and limitations; it shows as an assumption. Unknown is explicit and listed under "unresolved decisions".
- **Prerequisites:** WP-12; WP-K0 (`planning_keys.yaml` shape); D-06.
- **Files:** `013_planning_values.sql`; `knowledge/profile/planning_keys.yaml` (shape from WP-K0); `internal/store/planning.go`; `internal/httpapi/planning.go`; `internal/profile/reconcile.go` (planning values become rows with origin `assumption` or `calculation`).
- **Contracts introduced:**
  - `GET /api/projects/{id}/planning`
  - `PUT /api/projects/{id}/planning/{key}` with body `{part_id, value|null, value_state, unit, range_low, range_high, origin, meaning, rationale, version}`
  - `DELETE` the same path (supersede, never hard delete)
  - Errors per §3.4.
- **Steps:** tests (typed-value CHECK; a money-total key rejected; a forbidden favourable default rejected); migration; store and API; reconciliation; a suggestion engine stub that proposes nothing until a library row is `reviewed`.
- **Edge cases:** a planning value and a user value on the same key → the user value wins (existing precedence); superseding keeps history; a key not in the registry → 422.
- **Acceptance:** AT-07, AT-09 (assumption half), AT-22.
- **Non-goals:** cost estimates (WP-34); eligibility (WP-15).

```text
Feature: Record assumptions and calculated planning values with range, limitations and explicit unknown.
Why: Incomplete projects need honest starting values (NW-REQ-007, 189, 190).
Lane: A. WP-13; plan §4.3; D-06.
Out of scope: suggesting values without an approved library (stay unknown), costs, eligibility.
Speed: new endpoints p50 100 / p90 250 ms (D-18); profile_edit unchanged 50/150.
Correctness: done = migration 013, registry, API, reconciliation shows them as assumptions. Edge cases: same key as user value, unregistered key, favourable default for structure/ground/compliance refused.
Security: org/site/project composite FKs.
After: §4.2 handoff.
```

---

### WP-14: Revisions, input fingerprints, knowledge version and staleness

- **Lane / stage / state:** A / Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-065, 068, 086, 177, 240, 268, 269, 274, 314, 326, 332, 371, 372.
- **Source wording:** "Monotonic revision and input fingerprint including reading selection, evidence, user/planning values, work items and knowledge versions" (L330); "Use explicit domain revision dependencies first; avoid a universal dependency engine" (L353); "Coalesce repeat update requests" (L357).
- **Behaviour:** each rebuild records a revision and fingerprint. Every domain write bumps its counter. Reads expose `revision` and `stale_for` per output. A `profile_rebuild` bench path exists.
- **Prerequisites:** WP-12, WP-13.
- **Files:** `014_revisions.sql`; `internal/knowledge/hash.go` (new); `internal/store/profile.go` (`RebuildProfile`); `internal/store/revisions.go`; `internal/httpapi/profile.go` (`revision` in JSON and SSE payload); `bench/budgets.json`, `cmd/intake-bench/main.go` (`profile_rebuild` on Spec Home; the 0991 fixture is added by WP-28).
- **Contracts introduced:** plan §4.4, §3.5 (`revision` added to the `profile` event), §3.6.
- **Steps:** tests (the fingerprint changes on each input class and only then; counters are monotonic under concurrent writes; two update requests → one job set); migration; implementation; bench path.
- **Edge cases:** knowledge files changed on disk while running (the hash is fixed at startup); rebuild failure → no counter bump (same transaction).
- **Acceptance:** AT-20 (pending/failed visible), AT-21 (restart), AT-25, AT-26; `profile_rebuild` within D-18 numbers.
- **Non-goals:** report staleness UI (WP-41).

```text
Feature: Rebuilds and domain writes record revisions and fingerprints so outputs know when they are stale.
Why: Reports must never silently claim freshness (NW-REQ-268, 276).
Lane: A. WP-14; plan §4.4, §3.5, §3.6; D-18 for the profile_rebuild budget.
Out of scope: a general dependency engine; UI.
Speed: profile_edit 50/150 unchanged; add profile_rebuild path (Spec Home) at 100/300 ms.
Correctness: done = migration 014, hash at startup, fingerprint table tests, coalescing test. Edge cases: rebuild failure, concurrent writes, knowledge changed on disk.
Security: counters per org/project.
After: §4.2 handoff with bench numbers.
```

---

### WP-15: Site routing of readings and derivation eligibility

- **Lane / stage / state:** A / Stage 1 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-021, 027, 067, 082, 086, 100, 104, 176, 191, 192, 237, 273, 314, 378, 383.
- **Source wording:** "Facts from a project's documents about the building become site facts with their document provenance" (L138); "accepting a planning assumption must not make it eligible for verified regulatory derivations: extend the current user-precedence rules with explicit eligibility checks" (L278).
- **Behaviour:** site-scope readings land on site rows with document provenance. `usableFacts` excludes assumptions, allowances and forecasts. A derived value from unverified inputs is amber, labelled “Planning only — inputs not verified”, and never shown as verified/compliant (D-32). Building age is read.
- **Prerequisites:** WP-12, WP-13, WP-14.
- **Current evidence:** `usableFacts`, `derive` (`internal/profile/reconcile.go`); `Harvest` uses only `ProfileDeterminants` (`harvest.go:80`); F06.
- **Files:** `internal/profile/reconcile.go`, `build.go`; `knowledge/determinants.yaml` (add `profile_group` to `existing_building_year` and `existing_building`; draft, owner review); tests.
- **Steps:** table tests for eligibility per D-06; site routing; the determinant change; measure the label-call fingerprint change (it re-reads label stages) and record it in the handoff.
- **Edge cases:** a user value with `meaning=allowance` → ineligible; a derived row whose input is later reset → recomputed; a year outside 1800 to the current year → candidate rejected by the existing parser.
- **Acceptance:** AT-09, AT-05 (Hale unchanged); both replays; answer keys.
- **Non-goals:** threshold calibration; K3 year thresholds.

```text
Feature: Building facts read from project documents become site facts with provenance; assumptions can't feed verified regulatory derivations; building age is read.
Why: NW-REQ-104, 192, 082.
Lane: A. WP-15; D-04, D-06 in plan §5.
Out of scope: thresholds, legal year thresholds (K3), UI.
Speed: profile_edit 50/150; profile_rebuild per WP-14.
Correctness: done = eligibility table tests, site routing test, replays/answer keys green, label re-read cost recorded. Edge cases: allowance meaning, reset input, implausible year.
Security: unchanged scoping.
After: §4.2 handoff.
```

---

### WP-20: Work items store and scope-picker cut-over

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-005, 006, 028, 029, 035, 046, 048, 067, 069, 070, 084, 085, 089, 099, 105, 106, 107, 108, 109, 112, 113, 114, 115, 119, 129, 191, 237, 241, 242, 273, 315.
- **Source wording:** "Coarse items come from the scope picker and document evidence: one per part and in-scope system … User choices remain final" (L144); "Replaces scope keys as the store of scope; the scope picker writes here" (L331).
- **Behaviour:** scope lives in `work_items`. The picker's contract and UI are unchanged. Defaults and evidence create proposed coarse items with deterministic IDs. User items are final.
- **Prerequisites:** WP-15. D-08 (recommended).
- **Current evidence:** `Build` and `withScope` (`internal/profile/build.go`); `SetScope` (`internal/store/profile.go`); `setScope` (`internal/httpapi/profile.go:776`); `ScopePicker.tsx`; F11.
- **Files:**
  - Create: `015_work_items.sql`, `internal/works/workitems.go`, `internal/store/works.go`, `internal/httpapi/works.go`.
  - Modify: `internal/profile/build.go` (scope rows projected from work items), `internal/store/profile.go` (rebuild maintains proposed coarse rows in the same transaction).
- **Contracts introduced:**
  - plan §4.5.
  - `GET /api/projects/{id}/works`.
  - `POST /api/projects/{id}/works` with body `{part_id, system_id, action, title?, existing_condition?, target?, quantity?, unit?}`.
  - `PUT /profile/scope` unchanged.
  - Event `works`.
- **Steps:**
  1. Tests: deterministic ID; duplicate coarse → unique violation mapped to 409; rebuild leaves user-touched rows alone; `out` persists; backfill counts.
  2. Migration with backfill and deletion of `scope.*` user values, with assertions.
  3. Pure rules (default action per work type and part type).
  4. Store and API.
  5. Projection of `scope.<leaf>` rows for compatibility.
  6. Gate: Playwright `profile.spec.ts`, answer keys.
- **Edge cases:** concurrent picker writes (advisory lock); a system ID unknown to the catalogue → 422; a deprecated system on an existing row (keep it, flag it); a project with no whole part (`ensureWholePart`).
- **Acceptance:** AT-16, AT-28 (partial), AT-22; `profile_edit` 50/150; new endpoints 100/250 (D-18).
- **Non-goals:** split and retire (WP-24); the action question (WP-22); existing-building routing (WP-21).

```text
Feature: Scope becomes a set of work items (action on a system at a part); the scope picker keeps working unchanged.
Why: Works model core (NW-REQ-028, 242); everything later links to work-item IDs.
Lane: A. WP-20; plan §4.5, §4.12 (backfill), D-08; D-05/D-07 as data defaults.
Out of scope: split/retire, action question, UI changes, dependencies.
Speed: profile_edit 50/150 (includes rebuild); new works endpoints 100/250.
Correctness: done = migration 015 with backfill assertions; scope.* user values gone; picker Playwright test unchanged; answer keys unchanged. Edge cases: concurrent picker writes, unknown/deprecated system, duplicate coarse item.
Security: composite FKs (project, site, part); 404 on wrong org/project.
After: §4.2 handoff.
```

---

### WP-21: Existing-building presence and part-level work type

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here. D-27 is recommended.
- **Requirements:** NW-REQ-005, 067, 075, 080, 108, 116, 117, 315, 376.
- **Source wording:** "A system stated present in an existing building, with no works action, is recorded on the site, not in the works. For `new` and `extend` projects today's behaviour holds" (L152); "Work type is one value per project, though Hale is both an extension and a refurbishment" (L106).
- **Behaviour:** for refurb, remediation and advisory, document presence without an action records an existing system on the site, not a work item. New and extend are unchanged. A part may carry its own work type.
- **Prerequisites:** WP-20; D-07.
- **Files:** `internal/profile/build.go`, `internal/works/workitems.go`; `knowledge/profile/key_scope.yaml` (`sys.<leaf>.existing` and part-level `hdr.work_type`); `internal/knowledge` (`work_type` predicate support is in WP-25).
- **Steps:** table tests over work type × presence × action × part type; implement; Hale fixture.
- **Edge cases:** part type `new` on a refurb project (Hale extension); user scope `in` on an existing system (the user wins: work item, default action from the part's work type).
- **Acceptance:** AT-29, AT-05; `scope_test.go` new and extend cases unchanged.
- **Non-goals:** a multi-choice work-type question (rejected option D-07 b).

```text
Feature: In existing-building projects, a system the documents say is present is recorded on the site, not as works; parts can carry their own work type (Hale).
Why: Presence ≠ scope (NW-REQ-075, 116); mixed interventions (NW-REQ-080).
Lane: A. WP-21; D-07, D-27.
Out of scope: changing hdr.work_type's Jev question; UI.
Speed: profile_edit 50/150.
Correctness: done = table tests; Hale replay + keys; new/extend unchanged. Edge cases: Hale extension part; user override.
After: §4.2 handoff.
```

---

### WP-22: The action question in the evidence fan-out

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-009, 061, 066, 115, 123, 124, 125, 126, 127, 128, 129, 130, 131, 270, 315, 367, 381, 385, 387.
- **Source wording:** all of L171, including "asked in the existing evidence fan-out alongside presence and provider … no new round trip", "its own threshold", "`several` goes to Needs mapping", and "record call count, tokens and elapsed time".
- **Behaviour:** the evidence call asks `sys.<leaf>.action` per labelled leaf. Applied answers set proposed work-item actions. `several` goes to Needs mapping. `not_stated` falls back to the default.
- **Prerequisites:** WP-20, WP-00 with D-17 resolved. Read [fan-out](https://docs.typesafe.ai/patterns/fan-out), [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one), [confidence](https://docs.typesafe.ai/confidence) and [jev-1.13](https://docs.typesafe.ai/model-jaggedness/jev-1.13), and cite them in comments.
- **Files:** `internal/profile/questions.go` (`EvidenceQuestions`), `internal/jobs/worker.go`, `source_reading.go`, `internal/profile/reconcile.go` (`shapeOf` gets `action`), `data/profile/thresholds.json` (`action` shape, `options: 10`, `green: null`), `internal/profile/build.go`. `QuestionVersion` bump only if `profile-3` semantics change (they do: a new key) — record it.
- **Steps:**
  1. Test the call contents and criteria keys.
  2. Implement the question with describes and boundary text from `actions.yaml`, so the wording is not duplicated in Go.
  3. Reconcile and apply to proposed coarse items only.
  4. Handle Needs mapping.
  5. Re-record the private replays (owner consent) and measure calls, tokens and elapsed time.
- **Edge cases:** an action for a leaf not in scope → proposes nothing (evidence only); a conflict with an accepted item → conflict shown; Jev omits the answer → unresolved, as today; Q-WP-22-1 on `text` vs `excerpt` (NW-REQ-127).
- **Acceptance:** AT-09 (work-item variant), AT-25; replays green after re-record; measurement note.
- **Non-goals:** location (WP-23); signals (WP-27).

```text
Feature: Ask what the works do to each labelled system (sys.<leaf>.action) in the same evidence call as presence and provider.
Why: Turns document evidence into actions on work items (NW-REQ-123-131).
Lane: A. WP-22. Cite TypeSafe fan-out, how-to-build, confidence and jev-1.13 pages. D-17 resolved first.
Out of scope: new round trips, new job kinds, threshold calibration, location question.
Speed: no change to filing (1 s/2 s); profile_edit 50/150; record evidence call count, tokens and elapsed time after the fingerprint change.
Correctness: done = call test, reconcile test, several→Needs mapping, not_stated→default, accepted items never overwritten, replays green, measurement recorded. Edge cases: out-of-scope leaf, conflict, omitted answer.
Security: unchanged.
After: §4.2 handoff with the measurement.
```

---

### WP-23: Part-based location question and fact placement

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-009, 061, 066, 074, 132, 133, 134, 135, 270, 315, 367, 381, 385, 387.
- **Source wording:** all of L175.
- **Behaviour:** the location options come from the site's parts. An applied answer sets `part_label` on that passage's facts (amber only), so readings land on parts.
- **Prerequisites:** WP-22, WP-11; D-26.
- **Files:** `internal/jobs/source_reading.go` (`sourceQuestions` builds options per project), `internal/jobs/worker.go` (`storedFacts` sets `PartLabel`), `data/profile/thresholds.json` (`location.n<k>`), `internal/profile/reconcile.go` (amber cap for located facts).
- **Steps:** tests (option builder; fingerprint changes when parts change; amber only; exact-label placement); implement; record the re-read cost.
- **Edge cases:** a project with only the whole part (the options reduce to generic ones); two parts with similar labels (D-26 criteria include kind); a part renamed after reading (the label no longer matches → the fact falls back to the whole part; recorded as a known limitation).
- **Acceptance:** AT-10.
- **Non-goals:** threshold calibration.

```text
Feature: The passage location question offers this site's parts instead of a fixed residential list; applied answers place facts on parts (amber only).
Why: Part-level evidence for packages (NW-REQ-074, 132-135).
Lane: A. WP-23; D-26; cite TypeSafe confidence page for per-option-count thresholds.
Out of scope: calibration, UI.
Speed: filing unchanged; profile_edit 50/150.
Correctness: done = option builder test, placement test, amber-only test, re-read cost recorded. Edge cases: whole-part-only project, similar labels, renamed part.
After: §4.2 handoff.
```

---

### WP-24: Edit, split and retire work items

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-011, 048, 070, 110, 111, 113, 315.
- **Source wording:** "Splitting a work item makes the parent a group, as allowance subdivision does. Small projects never need to split" (L146).
- **Behaviour:**
  - The user edits action, condition, target, quantity and title.
  - Splitting turns the parent into a group, with children defaulting to the parent's values.
  - Retiring is refused while the item is referenced (D-19), and allowed otherwise.
- **Prerequisites:** WP-20.
- **Files:** `internal/works/split.go`, `internal/store/works.go`, `internal/httpapi/works.go`.
- **Contracts introduced:**
  - `PATCH /api/projects/{id}/works/{wi}` with body `{version, …fields}`.
  - `POST …/works/{wi}/split` with body `{version, children:[{part_id?, system_id?, action?, title}]}` (at least 2 children).
  - `POST …/works/{wi}/retire` with body `{version}`, returning 409 `{error:"referenced", blockers:[…]}` while references exist.
- **Edge cases:** splitting a group again (allowed, nested); a child on another site's part → 404 or FK error; a stale version → 409; retiring a coarse proposed item → it stays retired across rebuilds (`user_touched`).
- **Acceptance:** AT-12 (work-item half), AT-19.
- **Non-goals:** cost and responsibility effects (WP-31, WP-34 implement their side of D-19).

```text
Feature: Edit, split (parent becomes group) and retire work items safely.
Why: Coarse first, finer when packages need it (NW-REQ-110, 111).
Lane: A. WP-24; plan §4.5; D-19.
Out of scope: cost/responsibility moves (later packages), UI polish.
Speed: 100/250 ms per write (D-18).
Correctness: done = split invariants, retire blockers, version conflicts. Edge cases: nested split, cross-site child, retire then rebuild.
After: §4.2 handoff.
```

---

### WP-25: Load the works knowledge at runtime and evaluate predicates

- **Lane / stage / state:** A / Stage 2 (can start in W1 after WP-K0) / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-030, 056, 057, 144, 145, 173, 174, 261, 264, 315, 376.
- **Source wording:** "New predicate operators … `works` … and `system_existing` … `system_present` keeps its meaning" (L262).
- **Behaviour:** the catalogue loads actions, interface consequences, consequences, unforeseen conditions and signals. A pure evaluator answers predicates over work items and site facts.
- **Prerequisites:** WP-K0. D-10.
- **Current evidence:** F04; `internal/knowledge/evaluate.go` (existing predicate evaluator); `98ca0b2` (the checker skips `knowledge/works/` in the general loader).
- **Files:** `internal/knowledge/load.go`, `works.go` (new), `evaluate.go`; tests with small fixture YAML in `internal/knowledge/testdata/`.
- **Steps:** loader tests (unknown reference → load error naming the file); evaluator tables for `works`, `system_existing` (D-10, including `remove`) and `system_present` unchanged; startup time check.
- **Edge cases:** unknown determinant in `when` → load error (the checker should already have caught it); `{det: work_type}` records → evaluate via the `work_type` operator after D-07; until then, unknown (never true).
- **Acceptance:** loader and evaluator unit tests; existing knowledge tests green; startup adds under 200 ms (record it).
- **Non-goals:** proposal ranking (WP-26).

```text
Feature: Load the works-layer knowledge (actions, ic, cq, uc, signals) and evaluate works/system_existing predicates in code.
Why: Proposals need it (NW-REQ-173); today the runtime ignores knowledge/works (F04).
Lane: A. WP-25; D-10; D-07 for the work_type operator.
Out of scope: proposal engine, Jev questions.
Speed: startup cost recorded; evaluation is pure.
Correctness: done = loader + evaluator tests; system_present unchanged. Edge cases: unknown refs, work_type records before D-07.
After: §4.2 handoff.
```

---

### WP-26: Proposal engine, ranking and decisions

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-005, 010, 030, 031, 032, 136, 137, 138, 139, 140, 141, 142, 144, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 162, 166, 172, 177, 190, 243, 244, 315, 326, 373, 385.
- **Source wording:** L179 in full; the interface-consequence table L185-L195; "Proposal kinds…" (L218); `proposal_decisions` (L332).
- **Behaviour:**
  - The code-only rebuild computes ranked proposals with reasons.
  - The user accepts or dismisses each one. A dismissal stays until its inputs change.
  - Accepting an investigation or `work_item` kind creates a work item idempotently.
- **Prerequisites:** WP-20, WP-25, WP-14. D-10, D-11 (interim 10), D-28.
- **Files:** `016_proposals.sql`; `internal/works/proposals.go` (pure: evaluate, key, fingerprint, rank); `internal/store/proposals.go` (projection written in the rebuild transaction; decisions); `internal/httpapi/proposals.go`; `data/profile/proposals.json` (show count). API only: no UI before WP-70, so the backend comes first (L19).
- **Contracts introduced:**
  - plan §4.7.
  - `GET /api/projects/{id}/proposals[?show=all]`.
  - `POST …/proposals/{key}/accept` with body `{inputs_fingerprint}`, which returns the created record.
  - `POST …/proposals/{key}/dismiss` with body `{inputs_fingerprint, rationale?}`.
  - `DELETE …/proposals/{key}/decision` with body `{version}` (undo).
  - A stale fingerprint returns 409.
- **Steps:**
  0. Measure evaluating the full loaded ic/cq/uc set (~570 records) inside a Spec Home rebuild and a 0991 rebuild. If `profile_edit` or `profile_rebuild` (D-18) would break, stop and report before wiring proposals into the edit path.
  1. Pure tests for each interface-consequence row (NW-REQ-147-155), the rooftop example (AT-30), the 0991 example (AT-01 synthetic) and a tenant fit-out fixture (AT-15, passes only after D-29 knowledge lands).
  2. Key stability across a split (D-19).
  3. Ranking.
  4. Migration, then store and API.
  5. Bench `profile_rebuild` with proposals.
- **Edge cases:** double accept → same record (AT-16); an accept racing a rebuild → serialised by the lock; a knowledge record removed after a decision → the decision is kept and the proposal disappears; `addressed_by_evidence` (WP-27); a draft record → the proposal is labelled draft.
- **Acceptance:** AT-13, AT-16, AT-25, AT-30, AT-15 (after D-29); `profile_edit` and `profile_rebuild` per D-18; proposals never depend on package or report state (NW-REQ-166).
- **Non-goals:** signals (WP-27); package, delivery and obligation accept targets (WP-30, 31, 35 add them).

```text
Feature: Code evaluates interface consequences, consequences and unforeseen conditions against work items during the rebuild, ranks them, and the user accepts or dismisses each.
Why: The building model raises what the works touch (NW-REQ-030, 136-142).
Lane: A. WP-26; plan §4.7; D-03, D-10, D-11, D-19, D-28.
Out of scope: Jev calls of any kind; accept paths for discipline/obligation/approval/hold_point until their tables exist.
Speed: rebuild including proposals within profile_edit 50/150 and profile_rebuild 100/300 (Spec Home, 0991).
Correctness: done = per-row tests, key/fingerprint tests, idempotent accept, dismissal persistence. Edge cases: double accept, removed record, draft record.
Security: composite FKs; no record IDs hard-coded in Go.
After: §4.2 handoff.
```

---

### WP-27: Signals in the evidence fan-out

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-009, 061, 066, 168, 270, 272, 315, 330, 367, 381, 387.
- **Source wording:** "signals: Jev nouls on labelled passages, as failure-mode detectors are" (L245); "none calls Jev" (L179).
- **Behaviour:** signal answers are stored as facts (`sig.<id>`). The rebuild marks a proposal `addressed_by_evidence` and shows the citation. It never auto-dismisses.
- **Prerequisites:** WP-26, WP-22; D-12; the D-17 workload decision.
- **Files:** `internal/jobs/worker.go` (`EvidenceCall`), `internal/knowledge/works.go`, `internal/works/proposals.go`, `internal/store/profile.go` (prefix `sig.` in `ReplaceDocumentFacts`).
- **Steps:** tests; implement; re-record the replays (owner consent); measure; filing bench under load.
- **Edge cases:** a signal with no matching labelled passage → state unknown; conflicting signal answers → unknown with both sources.
- **Acceptance:** AT-25; filing 1/2 s; measurement note.
- **Non-goals:** a second "does the spec require prevention" question (Pass B open item 1; owner decision, not this wave unless approved).

```text
Feature: Ask the knowledge signals as background evidence questions and let proposals show "already addressed by evidence".
Why: Unforeseen conditions become foreseen without a Jev call on a click (NW-REQ-168, 136).
Lane: A. WP-27; D-12, D-17; cite TypeSafe fan-out and confidence pages.
Out of scope: auto-dismissal, new job kinds, new questions beyond signals.
Speed: filing 1 s/2 s under background load; record calls, tokens and elapsed time.
Correctness: done = signal facts stored, proposal state test, replays green, measurement recorded.
After: §4.2 handoff.
```

---

### WP-28: Evaluation for works: 0991 and 0777 keys and the rebuild fixture

- **Lane / stage / state:** A / Stage 2 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-005, 315, 326, 334, 335, 382, 385.
- **Source wording:** "Add 0991 and 0777 to `data/eval/profile/manifest.json` by ID only; the corpus stays private and no personal details are recorded" (L413); "owner-drafted work-item keys for 0991 and 0777 (`reviewed: false` until owner review)" (L401).
- **Behaviour:** both projects are in the manifest. Draft work-item answer keys exist, and the harness scores work items. A `profile_rebuild` bench fixture for 0991 exists.
- **Prerequisites:** WP-20; D-23; corpus access on the machine.
- **Files:** `data/eval/profile/manifest.json`, `data/eval/profile/answer-keys/0991.yaml`, `0777.yaml`, `data/eval/profile/README.md` (work-item scoring rules), the harness (`internal/eval` or `cmd/profile-eval`), `cmd/intake-bench/main.go` (fixture).
- **Steps:** select documents; hash them; draft keys from the documents (`reviewed: false`, agent drafts) **without copying text**; scoring code; fixture.
- **Edge cases:** a path segment containing personal details → stop and ask (NW-REQ-335); a document absent → the harness fails closed.
- **Acceptance:** the harness runs and reports on 0991 (AT-01 keys) and 0777 (AT-02 keys); it does not gate until the owner reviews.
- **Non-goals:** marking keys reviewed.

```text
Feature: Add 0991 (capex fire services) and 0777 (tower fit-out) to the profile evaluation with draft work-item keys and a rebuild fixture.
Why: The eval set has no capex/fit-out project (NW-REQ-334).
Lane: A. WP-28; D-23.
Out of scope: copying any corpus text or personal details; marking keys reviewed.
Correctness: manifest entries (id, relative path, sha256), keys reviewed:false, harness scores work items, 0991 bench fixture. Edge cases: personal details in path, missing file.
After: §4.2 handoff.
```

---

### WP-30: Packages and stages

- **Lane / stage / state:** A / Stage 3 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-034, 035, 049, 051, 064, 088, 162, 195, 197, 198, 199, 244, 245, 267, 317, 366.
- **Source wording:** L296 in full; `packages` row (L333).
- **Behaviour:**
  - The user creates services, works and supply packages with editable stages.
  - Novation splits stages before and after.
  - Defaults are suggested as `proposed` from copied Clerk data.
  - A `discipline` proposal can now be accepted.
- **Prerequisites:** WP-20; WP-K0 (`stages.yaml`, `package_defaults.yaml`).
- **Files:** `017_packages.sql`; `internal/procurement/packages.go`; `internal/store/packages.go`; `internal/httpapi/packages.go`.
- **Contracts introduced:** plan §4.8.
  - `GET/POST /api/projects/{id}/packages`
  - `PATCH /api/projects/{id}/packages/{pkg}`
  - `POST/PATCH …/packages/{pkg}/stages`
  - Event `packages`.
- **Edge cases:** novation on a works package → 422; renaming a stage referenced by an issued schedule (the issued snapshot is unaffected; the draft is stale); retiring a package with responsibilities → 409 with blockers.
- **Acceptance:** AT-22, AT-28; 100/250 ms.
- **Non-goals:** responsibilities (WP-31); tender issue.

```text
Feature: Services, works and supply packages with editable stages and novation, plus draft default suggestions.
Why: Commercial spine (NW-REQ-197-199).
Lane: A. WP-30; plan §4.8; D-22.
Out of scope: responsibilities, pricing, tender flows, builder-side records.
Speed: 100/250 ms.
Correctness: CHECKs, novation split, idempotent accept of discipline proposals. Edge cases: novation on works, retire with references.
After: §4.2 handoff.
```

---

### WP-31: Responsibilities and obligations

- **Lane / stage / state:** A / Stage 3 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-011, 034, 050, 085, 089, 090, 092, 120, 149, 154, 162, 244, 246, 247, 265, 267, 317, 384.
- **Source wording:** `package_scope_items` row (L334); `retain` "reaches packages as an obligation" (L154).
- **Behaviour:**
  - Packages hold roles for work items, or general obligations.
  - Clauses come from the catalogue or user text; draft clauses stay provisional.
  - `obligation` and retain proposals can be accepted.
  - D-19 effects apply: inheritance from a group, and retire blockers.
- **Prerequisites:** WP-30, WP-40a (WP-24 only for M2 split/inheritance integration) (clause catalogue loaded, so `clause_id` and `clause_version` can be validated).
- **Files:** `018_package_scope.sql`; `internal/procurement/scope.go`; `internal/store/packages.go`; `internal/httpapi/packages.go`.
- **Contracts introduced:** plan §4.8. `POST/PATCH /api/projects/{id}/packages/{pkg}/scope` with body `{item_kind, work_item_id?, role?, clause_id?, clause_version?, user_text?, stage_id?, inclusion, deliverable?, interface_ids?, source_refs?, version}`.
- **Edge cases:** a role on a supply package other than `supply` → 422 (D-09); an `interface_ids` entry not in the catalogue → 422; a work item from another project → FK error or 404.
- **Acceptance:** AT-11 (record half), AT-22.

```text
Feature: Packages hold responsibilities (design, supply, install, test, certify, inspect, maintain operation, protect) for work items, and general obligations.
Why: Who does what, without gaps or overlaps (NW-REQ-246, 120).
Lane: A. WP-31; plan §4.8; D-09, D-19.
Out of scope: gap check UI (WP-32), pricing.
Speed: 100/250 ms.
Correctness: CHECKs, clause/user-text exclusivity, retain obligations, idempotent accept. Edge cases: wrong role for kind, unknown interface, cross-project item.
After: §4.2 handoff.
```

---

### WP-32: Gap check

- **Lane / stage / state:** A / Stage 3 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-111, 119, 155, 259, 260, 317, 327, 384.
- **Source wording:** "Gap check (a read): every accepted physical work item has exactly one works package with install (or supply and install), and, where its action needs design, one services package with design. A missing or duplicated role shows as a gap or overlap" (L343).
- **Behaviour:** `GET /api/projects/{id}/gaps` lists gaps, overlaps and items not yet accepted. The rules are D-09.
- **Prerequisites:** WP-31; WP-K0 (`needs_design`).
- **Files:** `internal/procurement/gaps.go` (pure); `internal/httpapi/packages.go`; bench path `gap_check`.
- **Edge cases:** groups and inheritance; supply-only owner package; retain; investigate; excluded and retired items.
- **Acceptance:** AT-14, AT-11 (gap half); `gap_check` 50/150.

```text
Feature: A read that lists work items missing an installer or designer, or with two.
Why: Package boundaries without gaps (NW-REQ-259, 260).
Lane: A. WP-32; D-03, D-09.
Out of scope: writes, Jev, UI beyond JSON.
Speed: gap_check p50 50 / p90 150 ms.
Correctness: table tests per D-09 rule incl. groups, supply-only, retain, investigate.
After: §4.2 handoff.
```

---

### WP-33: Cost plan schema and money core

- **Lane / stage / state:** A / Stage 3 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-002, 006, 046, 052, 085, 088, 092, 182, 193, 195, 196, 200, 205, 207, 209, 210, 211, 212, 213, 221, 251, 252, 253, 254, 255, 256, 267, 317.
- **Source wording:** L298-L314 and the `cost_*` rows (L337-L339).
- **Behaviour:** versioned cost plans with stable items, per-metric values, decimal money, central rounding and a frozen baseline.
- **Prerequisites:** WP-20 (works lines), WP-30 (`package_stages` for fee lines), WP-31 (`package_scope_items` for `scope_cost_links`). Migration 021 references 017 and 018; this package starts after M1 passes and report migration 020 merges. It must not block the M1 draft.
- **Files:** `021_cost_plan.sql`; `internal/costs/money.go` (`Round`, `big.Rat` helpers); `internal/costs/plan.go`; `internal/store/costs.go`; `internal/httpapi/costs.go`.
- **Contracts introduced:** plan §4.10.
  - `GET /api/projects/{id}/cost-plan[?version=]`
  - `POST …/cost-plan/items`
  - `PATCH …/cost-plan/items/{item}`
  - `PUT …/cost-plan/items/{item}/values/{metric}` with body `{amount|null, low?, high?, value_state, as_of?, origin, meaning, rationale?, version}`
  - Event `costs`.
- **Edge cases:** a value on a non-posting item → 422; claimed-to-date without `as_of` → 422; editing a frozen version → 409 `{error:"frozen"}`; GST basis mismatch is not mixable in one version.
- **Acceptance:** AT-18, AT-22; no `float64` in `internal/costs` (a vet-style test).

```text
Feature: Versioned cost plan: stable cost items, budget/estimate/commitment/claimed-to-date values in decimal money with central rounding, frozen baselines.
Why: One money ledger linking allowance to schedule (NW-REQ-193, 209, 210).
Lane: A. WP-33; plan §4.10; D-13 (no stored forecast), D-14.
Out of scope: tender offers, invoices, forecasts as stored values, dependencies (no decimal library: use numeric + big.Rat).
Speed: 100/250 ms.
Correctness: CHECKs, triggers on frozen versions, rounding tests (half away from zero at line level), null amounts stay null. Edge cases: non-posting value, missing as_of, frozen edit.
After: §4.2 handoff.
```

---

### WP-34: Cost operations, reconciliation and summaries

- **Lane / stage / state:** A / Stage 3 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-002, 008, 010, 011, 073, 111, 185, 188, 194, 200, 201, 202, 203, 204, 206, 208, 211, 214, 215, 221, 223, 224, 225, 226, 227, 252, 255, 263, 317.
- **Source wording:** "Convert the parent to a non-posting group … Never add the original allowance and its replacement breakdown together" (L298); "Works lines name one work item … so by-system and by-part totals reconcile exactly" (L308).
- **Content gate:** estimates require owner-reviewed benchmark records; draft benchmarks cannot establish amounts. Unknown stays unknown.
- **Behaviour:**
  - Subdivide an allowance atomically.
  - Baseline the plan.
  - Totals by system, part, package and overall all reconcile, and incomplete totals are flagged.
  - The profile shows a summary read from the cost tables.
  - A works line's package requires a responsibility.
- **Prerequisites:** WP-33, WP-31.
- **Files:** `internal/costs/subdivide.go`, `totals.go`; `internal/store/costs.go`; `internal/httpapi/costs.go`, `profile.go` (summary block).
- **Contracts introduced:**
  - `POST …/cost-plan/items/{item}/subdivide` with body `{version, children:[{label, line_kind, …, amount}], residual: "explicit"|"change_total"}`
  - `POST …/cost-plan/baseline` with body `{version}`
  - `GET …/cost-plan/totals?by=system|part|package|overall`
- **Edge cases:** children summing above the parent with `explicit` residual → 422; a null child amount → incomplete; D-19 retire blockers; splitting the work item behind a works line (the line stays on the group).
- **Acceptance:** AT-17, AT-18, AT-12 (cost half); a property test that the sum over systems equals the sum over parts equals the works total.

```text
Feature: Subdivide allowances safely, baseline, and reconcile totals by system, part, package and overall; show a cost summary in the profile from the cost records.
Why: No double counting (NW-REQ-202-206, 211).
Lane: A. WP-34; D-13, D-14, D-19.
Out of scope: benchmark estimates until benchmarks are reviewed; offers; invoices.
Speed: totals read 100/250 ms.
Correctness: atomic subdivision, residual handling, property test for reconciliation, package-responsibility rule. Edge cases: over-allocation, null child, split work item.
After: §4.2 handoff.
```

---

### WP-35: Minimal delivery records

- **Lane / stage / state:** A / Stage 3 (moved forward per D-03) / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-006, 011, 046, 051, 053, 085, 087, 089, 092, 153, 162, 244, 248, 249, 250, 281, 282, 283, 290, 317, 382.
- **Source wording:** `project_delivery_items` and `delivery_dependencies` (L335-L336); "Store minimal delivery records now rather than inferring progress from upload volume" (L365).
- **Behaviour:** milestones, approvals, risks, issues, decisions, actions and activities with dates and status; finish-to-start dependencies; `approval` and `hold_point` proposals can be accepted.
- **Prerequisites:** WP-20, WP-30 (optional package link).
- **Files:** `019_delivery.sql`; `internal/delivery/*.go`; `internal/store/delivery.go`; `internal/httpapi/delivery.go`.
- **Contracts introduced:**
  - `GET/POST /api/projects/{id}/delivery`
  - `PATCH …/delivery/{item}`
  - `POST/DELETE …/delivery/dependencies`
  - Event `delivery`.
- **Edge cases:** a cycle → 422 naming the cycle; a status invalid for the kind → 422; marking an approval approved needs an actor (always present) and optionally a `SourceRef`; never set from an upload.
- **Acceptance:** AT-22; cycle tests; 100/250 ms.
- **Non-goals:** CPM, Gantt.

```text
Feature: Minimal dates, approvals, risks, decisions and actions with finish-to-start dependencies, before the first report.
Why: RFP/RFT/PMP need dates and risks (D-03; NW-REQ-283).
Lane: A. WP-35; plan §4.9; D-25.
Out of scope: scheduling engine, progress inference from uploads.
Speed: 100/250 ms.
Correctness: per-kind validation, cycle rejection, explicit progress only.
After: §4.2 handoff.
```

---

### WP-40: Report clause and template catalogues (knowledge)

- **Lane / stage / state:** A / runs in W1 after WP-K0 (needed by WP-31) / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-263, 287, 318, 382.
- **Source wording:** "Keep reviewed clauses and benchmarks as versioned catalogues loaded at startup … Only reviewed content may establish standard obligations" (L347); "approved prose fragments" (L375).
- **Behaviour:** `knowledge/reports/clauses.yaml` and the RFP, RFT and PMP templates (section IDs, essential flags, fragments) exist as drafts. The loader and checker validate them.
- **Prerequisites:** WP-K0 (shape).
- **Files:** `knowledge/reports/*.yaml`; `internal/knowledge/reports.go`; checker rules.
- **Acceptance:** checker strict; owner review before any clause is `reviewed`.
- **Non-goals:** writing legal or contract wording as approved; drafts only.

```text
Feature: Versioned clause and report-template catalogues (draft) for RFP, RFT and PMP.
Why: Reports assemble approved wording, never generated prose (NW-REQ-010, 263, 287).
Lane: A (knowledge). WP-40.
Out of scope: marking anything reviewed; generated text.
Correctness: checker strict passes; every clause has id, version, status, source.
After: §4.2 handoff.
```

---

### WP-41: Report store and assembler core

- **Lane / stage / state:** A / Stage 4 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-002, 010, 011, 015, 016, 039, 093, 257, 265, 268, 275, 276, 277, 278, 287, 318, 325, 374, 382.
- **Source wording:** L353-L361; `reports` row (L340).
- **Behaviour:**
  - One assembler builds a draft from a consistent read of saved revisions.
  - Protected edits survive re-assembly. Changed sources under an edit show as conflicts.
  - Pending or failed reading is shown honestly.
- **Prerequisites:** M1/a: WP-14, WP-26, WP-30–32, WP-35a, WP-40a. M2/b: WP-33–35 and remaining WP-40; see §2.4. No cost-table access in phase a.
- **Files:** `020_reports.sql`; `internal/reports/assemble.go`, `edits.go`, `deps.go` (static dependency map); `internal/store/reports.go`; `internal/httpapi/reports.go`.
- **Contracts introduced:**
  - `POST /api/projects/{id}/reports` with body `{kind, package_id?}`
  - `GET /api/reports/{r}`
  - `POST /api/reports/{r}/draft` with body `{use_last_completed?: bool}`
  - `PUT /api/reports/{r}/edits/{target}` with body `{text, version}`
  - Event `report`.
- **Edge cases:** reading pending → the response offers `use_last_completed` or wait (AT-20); a package retired → 409 on draft; template version changed → the draft is stale.
- **Acceptance:** AT-20, AT-21, AT-25; `report_assemble` 300/1,000 ms.

```text
Feature: One deterministic assembler producing report drafts from consistent saved revisions, with protected edits and honest freshness.
Why: Reports trusted and reproducible (NW-REQ-093, 275-278).
Lane: A. WP-41; plan §4.11, §3.6.
Out of scope: export rendering, citations UI, generated prose.
Speed: report_assemble p50 300 / p90 1000 ms.
Correctness: repeatable-read assembly, edit protection + conflicts, stale/pending/failed states. Edge cases: pending reading, retired package, template change.
After: §4.2 handoff.
```

---

### WP-42: Citations and assumptions in reports

- **Lane / stage / state:** A / Stage 4 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-258, 265, 288, 289, 290, 291, 292, 293, 294, 318, 382, 383.
- **Source wording:** "Colour-labelled citations: E blue evidence, U teal user input, C purple calculation, A amber assumption. Text labels remain legible without colour … PDF references must remain meaningful without an app session" (L375).
- **Behaviour:** every value in a draft carries a citation (E, U, C or A). Material assumptions are listed. A citation opens its detail. The PDF reference text is self-contained.
- **Prerequisites:** WP-41.
- **Files:** `internal/reports/citations.go`; `internal/store/reports.go`; a web citation component (WP-45 owns the page).
- **Acceptance:** greyscale-legible labels; the reference text includes filename, document number, revision, page and location.

```text
Feature: Every report value carries an E/U/C/A citation; material assumptions are listed; references make sense on paper.
Why: Trust without the app (NW-REQ-288-294).
Lane: A. WP-42.
Out of scope: export engine.
Correctness: label/colour mapping, greyscale legibility, self-contained reference text.
After: §4.2 handoff.
```

---

### WP-43: Issue, immutable snapshots and frozen pricing schedules

- **Lane / stage / state:** A / Stage 4 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-019, 184, 216, 217, 218, 219, 257, 279, 280, 318.
- **Source wording:** L363; L310.
- **Behaviour:**
  - Issuing freezes a canonical snapshot, its hash, the pricing schedule (cost-item IDs, labels, stages) and, after WP-44, the export blob.
  - Later changes flag drafts only.
  - Budget disclosure is opt-in.
- **Prerequisites:** WP-41b, WP-42b, WP-33–34; M1 passed. Issue is unavailable before M2.
- **Files:** `internal/reports/snapshot.go`; `internal/store/reports.go` (trigger already in 020); `internal/httpapi/reports.go`.
- **Contracts introduced:** `POST /api/reports/{r}/issue` with body `{version, reporting_date, budget_disclosed}`.
- **Edge cases:** issuing while stale → 409 unless `{accept_stale:true}` with the reason recorded; a crash mid-issue → nothing is issued (one transaction; the blob is written before commit and content-addressed, so an orphan is harmless).
- **Acceptance:** AT-23, AT-21.

```text
Feature: Issuing records an immutable, hashed snapshot and frozen pricing schedule; later updates only flag drafts.
Why: Reproducible issues after reprocessing or knowledge changes (NW-REQ-279, 280, 217).
Lane: A. WP-43; D-20.
Out of scope: tender returns, renderer choice.
Correctness: DB trigger forbids editing issued versions; reproduction from snapshot only; opt-in budget disclosure. Edge cases: stale at issue, crash mid-issue.
After: §4.2 handoff.
```

---

### WP-44: Export renderer and the two-page gate

- **Lane / stage / state:** C (new dependency or binary) plus A / Stage 4 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-017, 018, 019, 020, 294, 295, 296, 318, 328.
- **Source wording:** L15; L377; "Measure export separately after choosing its renderer" (L409).
- **Behaviour:** reports export to PDF at the agreed minimum font size within two pages, targeting about 1½. Overflow of essential content produces a review prompt or an identified attachment. The cost plan is exempt.
- **Prerequisites:** WP-43; D-15 spike result; D-16.
- **Files:** `internal/reports/render/*`; `bench/budgets.json` (`report_export` after measuring); a dependency justification in the commit message.
- **Acceptance:** AT-24; `report_export` measured and a budget proposed.

```text
Feature: Render reports to PDF within two pages at the agreed minimum font; never drop essential content silently.
Why: NW-REQ-018, 295, 296.
Lane: C + A. WP-44; D-15 (renderer) and D-16 (font/page) must be recorded.
Out of scope: layout redesign of the app.
Speed: measure report_export and propose a budget.
Correctness: page-count + min-font render tests, overflow prompt/attachment, cost-plan exemption. Dependency justified (who else watches it).
After: §4.2 handoff.
```

---

### WP-45: The first report: consultant RFP on 0991

- **Lane / stage / state:** A / Stage 4 / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:** NW-REQ-002, 013, 016, 035, 037, 216, 285, 293, 318, 333, 375, 382, 383, 384, 385, 386, 389.
- **Source wording:** "Consultant RFP on 0991 from one shared assembler, with approved clauses, stable edits, citations, immutable issue and two-page export" (L404); the adoption measure (L411).
- **Behaviour:** the owner drops the 0991 documents, reviews work items and proposals, sets up the services package, and drafts, edits and issues the RFP. Active minutes are recorded.
- **Prerequisites:** M1/a: WP-41a/42a, WP-28a/K6a and M1 route checks; M2/b: WP-41b–44 and remaining WP-28. D-31–35 govern both phases.
- **Files:** `web/src/Report.tsx` (new), `web/src/reportApi.ts`; `022_adoption_events.sql`; `internal/httpapi/adoption.go`; `docs/evidence/2026-xx-0991-first-run.md`.
- **Acceptance:** M1/a AT-31–34 and plan §8.6 quality/time gates; M2/b AT-01/23/24/28 plus issuable-RFP measurement. Freeze targets before scoring; a draft demo is not an issued report.

```text
Feature: WP-45a: 0991 draft/edit/refresh plus minimum review controls and predefined quality/time gates. WP-45b, only after M1: issue/export and issuable-RFP measurement. Run only the authorised phase.
Why: Proves the wave on a real capex job (NW-REQ-037, 333).
Lane: A. WP-45; D-02, D-21.
Out of scope: RFT/PMP, tabs redesign.
Speed: report_assemble 300/1000; edits 100/250.
Correctness: AT-01 and AT-28; adoption minutes recorded without content.
After: §4.2 handoff.
```

---

### WP-50: Works RFT

- **Lane / stage / state:** A / Stage 5.
- **Requirements:** NW-REQ-002, 014, 038, 120, 172, 216, 286, 297, 319, 384.
- **Behaviour:** an RFT for a works package with inclusions and exclusions, supply/install boundaries, retain obligations, access and sequencing, latent-condition handling and provisional sums (from unforeseen conditions), dates, the tender return, and exact document revisions.
- **Prerequisites:** WP-45.
- **Acceptance:** AT-11; template section test.

```text
Feature: Works RFT from the same assembler, carrying responsibilities, retain obligations and latent-condition handling.
Lane: A. WP-50. Requirements NW-REQ-286, 297, 319.
Out of scope: tender issue automation, offers.
Correctness: AT-11; document revisions exact.
After: §4.2 handoff.
```

---

### WP-51: PMP

- **Lane / stage / state:** A / Stage 5. The variance section waits on D-13.
- **Requirements:** NW-REQ-002, 012, 038, 172, 224, 225, 284, 319.
- **Behaviour:** the PMP has definition and quality; delivery and appointments; authorities and approvals; time and cost (budget/forecast variance, coverage, unavailable values shown as unavailable); material risks including unforeseen conditions; changes, decisions and next actions.
- **Prerequisites:** WP-50, WP-35, WP-34.

```text
Feature: PMP (delivery plan + progress report) from the same assembler.
Lane: A. WP-51. D-13 for forecast.
Correctness: section test; unavailable commitments shown as unavailable, never 0.
After: §4.2 handoff.
```

---

### WP-60: Live reporting

- **Lane / stage / state:** A / Stage 6.
- **Requirements:** NW-REQ-006, 012, 281, 282, 320.
- **Behaviour:** minimal progress updates, authority status, dates, risks and actions, and "changes since issue" (a diff of the current draft against the last issued snapshot). Cost and time summaries reference their owning records.
- **Prerequisites:** WP-51.

```text
Feature: Live PMP reporting: progress, authorities, dates, risks/actions and changes since the last issue.
Lane: A. WP-60. Progress only from explicit records (NW-REQ-281, 282).
After: §4.2 handoff.
```

---

### WP-70: Tabs and optimistic edits

- **Lane / stage / state:** B (layout) plus A (rollback correctness) / Stage 7.
- **Requirements:** NW-REQ-024, 291, 298, 299, 300, 321, 323, 363.
- **Behaviour:** Summary and Systems tabs first (owner decides whether the others are needed), reading the existing endpoints. Edits are optimistic with rollback on 409 or an error. One mark per value.
- **Acceptance:** Playwright rollback test; local edit feedback under 100 ms.

```text
Feature: Summary and Systems tabs over existing endpoints; optimistic edits with rollback.
Lane: B + A. WP-70. No new store; no major redesign (NW-REQ-363).
Speed: local feedback < 100 ms.
After: §4.2 handoff.
```

---

### WP-71: Final latency verification on the VPS

- **Lane / stage / state:** release / Stage 7.
- **Requirements:** NW-REQ-321, 322, 324, 381, 388, 389.
- **Behaviour:** every §8.3 path is measured with `-live` on the target VPS and recorded. Budgets are approved or revised by the owner.
- **Prerequisites:** a provisioned target VPS (not yet measured: plan §2.2), all packages.

```text
Feature: Measure every new and existing budget path live on the target VPS; owner approves the numbers.
Lane: release. WP-71. Plan §8.3.
After: §4.2 handoff with the full table.
```

---

### WP-X1: Access-control and recovery sweep

- **Lane / stage / state:** C / end of W4, W6, W7.
- **Requirements:** NW-REQ-036, 060, 091, 229, 230, plus scenarios AT-19, AT-21, AT-22 across all new routes.
- **Behaviour:** a review checklist over every merged package (nothing required beyond need; no text-inferred relations; stable IDs never reused; `org_id` plus `project_id` or `site_id` on every row), and a table-driven test over every route added in this wave (wrong org, wrong project, wrong site → 404, with no leak), every composite FK (DB rejects), and a restart mid-job, mid-assembly and mid-issue.

```text
Feature: One table-driven access and restart test over every route and FK added in the wave.
Lane: C. WP-X1. AT-19, AT-21, AT-22.
After: §4.2 handoff.
```

---

### WP-K0: K0 closure and new knowledge shapes

- **Lane / stage / state:** knowledge / W1 / K0 implemented (`c2ed60d`); **owner approval not evidenced**. New shapes Not started.
- **Requirements:** NW-REQ-047, 051, 056, 121, 122, 143, 150, 151, 161, 165, 199, 226, 261, 262, 263, 264, 302, 305, 316, 366, 376, 377, 379.
- **Behaviour:**
  - The owner approves (or amends) the existing shapes: actions, interface consequences, cq, uc, signals and the ledger. The approval packet includes the 560 K4 `uc.*` records already written on those shapes (NW-REQ-304) and decision D-30 (`ic.controls-test-link` kind).
  - `SCHEMA.md` documents the new shapes **before** any data: `stages.yaml`; `package_defaults.yaml` (Clerk copy as data); clause and benchmark catalogues; `key_scope.yaml`; `planning_keys.yaml`; `needs_design` on actions; `action` on `work_item` proposals; the `work_type` operator (after D-07); the D-24 decision.
  - The checker validates them.
- **Commands:** `python tools/check_knowledge.py --strict`; `python -m unittest discover -s tools -p 'test_*.py' -v`.

```text
Feature: Close K0 (owner approval) and add the knowledge shapes later packages need, documented in SCHEMA.md first.
Lane: knowledge. WP-K0. Decisions D-07, D-09, D-22, D-24, D-30.
Out of scope: Go code; marking content reviewed.
Correctness: checker strict 0 errors; tools tests pass.
After: §4.2 handoff.
```

---

### WP-K1 / WP-K2 / WP-K3: Interface audit, consequences from seeds, existing-building law

- **Lane / stage / state:** knowledge (research) / W1 onward / Current evidence: plan §0.2; decisions: plan §5; future phase: §2.4 here.
- **Requirements:**
  - K1: NW-REQ-158, 159, 302, 304, 306, 316.
  - K2: NW-REQ-056, 160, 302, 304, 307, 316, 366.
  - K3: NW-REQ-056, 082, 159, 160, 163, 302, 304, 308, 316.
- **Source wording:** the PRG K1-K3 rows (L388-L390) in full, with their gates.
- **Rules:**
  - Seeds first: `../clerk/data/seed/*.md`, copied as data with anchors.
  - Primary instruments for K3.
  - `clause_verified: false` until the instrument is read.
  - Australian Standards stay unverified without licensed access.
  - No invented numbers; `status: draft`.
- **Acceptance:** checker strict; anchors resolve; a merge note per cluster; reviewer spot-check (§2.5). K1 (services-wet-air cluster) must also make AT-15 passable per D-29: base-building plant to tenancy edges typed so an `alter` fit-out raises a capacity investigation with a correct label.

```text
Feature: [K1] Audit interfaces for existing buildings and fit-outs in cluster <c>, adding the missing edges listed in the PRG K1 row. | [K2] Draft cq.* consequences for cluster <c> from the named seed guides. | [K3] Draft NSW existing-building law records from primary instruments.
Lane: knowledge research only (no Go, no migrations, no web). WP-K1/K2/K3 in docs/plans/2026-10-04-next-wave-agent-work-packages.md; shapes in knowledge/SCHEMA.md.
Rules: every record cites a source that resolves; clause_verified:false until read in the instrument; no invented numbers; status: draft.
Done: python tools/check_knowledge.py --strict → 0 errors; REPORT.md merge note; §4.2 handoff.
```

---

### WP-K4: K4 merge pass and owner review support

- **Lane / stage / state:** knowledge / W1 / Pass A and B **Implemented** for batches 1 and 2 (F23); merge pass Not started.
- **Requirements:** NW-REQ-057, 164, 167, 169, 170, 171, 302, 304, 309, 310, 316.
- **Behaviour:** close the open items in `docs/unforeseen/pass-b-reports.md`:
  - cross-cluster overlaps;
  - misplaced records;
  - loose merges;
  - `work_type` conditions (via D-07);
  - generated contract wording and severities.
  
  Also list every existing `fm.*` whose `runs_on` or wording changed (NW-REQ-171), and prepare the owner review packet. Do not re-run Pass A or B.
- **Note:** `docs/unforeseen/pass-b-reports.md` is currently untracked; it belongs to another session. Ask the owner before committing it.

```text
Feature: Finish K4: cross-cluster merge pass and an owner review packet; list every changed existing fm.* record.
Lane: knowledge. WP-K4. NW-REQ-171 (existing fm unchanged) is the key check.
Out of scope: new rows, Go code.
Done: checker strict 0 errors, 0 pending; review packet; §4.2 handoff.
```

---

### WP-K5: Owner lessons

- **Requirements:** NW-REQ-302, 311, 316.
- **Behaviour:** structured interviews (what was found, when, what it cost in time, what would have revealed it earlier), recorded as draft `uc.*` or `fm.*` records.
- **Owner time required.**

### WP-K6: Walk-through of 0991 and 0777

- **Requirements:** NW-REQ-157, 302, 312, 316, 385.
- **Behaviour:** run the 0991 and 0777 work items through the proposal engine. Compare with the consultants, investigations and tests those projects actually appointed or budgeted. Record misses as new records and fix noise in the knowledge. The plan §8.6 sets the scoring target before the run; the owner reviews the key and scores usefulness.
- **Prerequisites:** a: WP-26, WP-28a and reviewed 0991-relevant K1–K3 records. b: WP-28b and reviewed 0777-relevant records. Broad catalogue completion is not a prerequisite.
- **Acceptance:** §8.6/AT-34 scored 0991 gate before M2; same quality gates plus AT-02/15 on 0777 before WP-50. Freeze keys first; record critical misses and useful/total counts, not only the kept share.

## 6. Status and evidence

The implementation plan §0.2 is the summary of where things are up to. §5 there owns decisions, §2.4 here owns future order, and the register state-update log owns detailed requirement evidence. The table below is the per-package status. **At every merge the integration lead updates all three: this table, plan §0.2 and the register state updates.** M1/M2/M3 phases and NW-REQ-381–389 are Planned, not implemented or verified.

### Package status (kept current at every merge)

Blockers are as first planned; for future order use §2.4. Last updated 5 October 2026 at `8b96f33`.

| WP | Stage | Lane | Blocked by | State |
| - | - | - | - | - |
| WP-00 | gate | A | none (D-17 resolved under A9) | **Implemented**, merged `a123dec`, `c0cb356`; profile replays green. Intake and bench gates restored by M0 (`e2eaa6d`, AT-35) and D-37 (`31681f9`) |
| WP-11 | 1 | A | none | **Implemented**, merged `0a13afd`; reviewed; site-edit lock, revision and event deferred to WP-14 |
| WP-12 | 1 | A | WP-11, WP-K0, D-04, D-06 | **Implemented**, merged `ece0086`; reviewed; full gate green |
| WP-13 | 1 | A | WP-12, WP-K0, D-06 | **Implemented**, merged `77ced4b`; reviewed; full gate green. `planning_keys.yaml` awaits owner review |
| WP-14 | 1 | A | WP-12, WP-13, D-18 (bench) | **Next.** Fix F31 first (plan §4) |
| WP-15 | 1 | A | WP-14, D-04, D-06 | Not started |
| WP-20 | 2 | A | WP-15, D-05 | Not started |
| WP-21 | 2 | A | D-07 | Not started |
| WP-22 | 2 | A | D-17 | Not started |
| WP-23 | 2 | A | WP-22 | Not started |
| WP-24 | 2 | A | WP-20 | Not started |
| WP-25 | 2 | A | D-07 for the work_type operator | **Implemented**, merged `b3a1048`; reviewed; findings fixed |
| WP-26 | 2 | A | WP-20, WP-25, WP-14 (D-03 for some accept kinds; D-29 for AT-15) | Not started |
| WP-27 | 2 | A | D-12, D-17 | Not started |
| WP-28 | 2 | A | D-23 | Not started |
| WP-30 | 3 | A | WP-20 (D-22 for defaults) | Not started |
| WP-31 | 3 | A | WP-30, WP-24, WP-40 | Not started |
| WP-32 | 3 | A | D-03, D-09 | Not started |
| WP-33 | 3 | A | WP-31 | Not started |
| WP-34 | 3 | A | WP-33, WP-31 (D-13; benchmarks) | Not started |
| WP-35 | 3 | A | WP-20, WP-30, D-03 | Not started |
| WP-40 | W1 | A | WP-K0 | Not started |
| WP-41 | 4 | A | WP-14, WP-30-35, WP-40 | Not started |
| WP-42 | 4 | A | WP-41 | Not started |
| WP-43 | 4 | A | WP-42 | Not started |
| WP-44 | 4 | C+A | D-15, D-16 | Not started |
| WP-45 | 4 | A | WP-44, WP-28, D-02, D-21 | Not started |
| WP-50 | 5 | A | WP-45 | Not started |
| WP-51 | 5 | A | WP-50, D-13 | Not started |
| WP-60 | 6 | A | WP-51 | Not started |
| WP-70 | 7 | B+A | WP-60 | Not started |
| WP-71 | 7 | release | VPS | Not started |
| WP-X1 | 3/5/7 | C | stage ends | Not started |
| WP-K0 | K | knowledge | owner approval of shapes; D-07, D-09, D-24, D-30 for the remaining shapes | K0 shapes implemented (approval pending); new catalogue shapes **Implemented**, merged `6e47383` (stages, package defaults, planning keys, key scope, clauses, benchmarks) |
| WP-K1-K3 | K | knowledge | WP-K0 | Not started |
| WP-K4 | K | knowledge | owner review | Pass A/B implemented; owner review packet merged `f665501` (`docs/unforeseen/k4-owner-review-packet.md`); merge-pass edits not started |
| WP-K5 | K | owner | owner time | Not started |
| WP-K6 | K | knowledge | WP-26, WP-28, WP-K1-K3 | Not started |
