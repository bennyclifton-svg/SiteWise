# Next wave: implementation plan

Status: **planning draft for owner review. No implementation is authorised by this document.**
Companion documents:

- `docs/plans/2026-10-04-next-wave-requirements-register.md`: every PRG statement, requirement IDs (`NW-REQ-###`), the coverage ledger, traceability and status.
- `docs/plans/2026-10-04-next-wave-agent-work-packages.md`: execution packages (`WP-##`), copy-ready briefs, file ownership, merge order and handoff rules.

Authoritative source (PRG): `docs/plans/2026-10-04-next-wave-architecture-schema.md`, left unchanged.
Decisions are recorded **once**, in §5 of this document (`D-##`). The register and packages refer to them by ID.

## 0. Planning baseline

| Item | Value |
| - | - |
| Branch | `main` |
| HEAD at planning | `f0c170a9bfefd49d749af507d57b1912689006d8` ("Merge K4 Pass B batch 2 services-wet-air…") |
| PRG commit reviewed by its author | `02094dbde0d9bd90f02d82c6ccc5bea2933a620c` |
| PRG file SHA-256 at planning | `6c1a03a7fd6a8b300079147fbd7b9e5fc864230bb1a755d929780d67e9a72cab` (435 lines; line numbers `L###` in the register refer to this hash) |
| PRG edits after 02094db | `6798122` replaced `construction_year` with `existing_building_year` at L209, L218, L345 and L387. The PRG preamble (L1) still says it was reviewed against 02094db. |
| Commits since 02094db | 36. Knowledge only: K0 shapes (`65b51cd`, `c2ed60d`), the building-age merge (`6798122`), K4 Pass A for batches 1 and 2, and K4 Pass B per cluster for batches 1 and 2. **No Go, migration or web change.** |
| Working tree | Untracked `.claude/` (agent worktrees) and `docs/unforeseen/pass-b-reports.md` (someone else's report). Left untouched. |
| Planning date | 5 October 2026 |

## 0.1 Standing owner authorisations (5 October 2026)

The owner (Benny Clifton) granted these in conversation on 5 October 2026, while away from the keyboard for most of the implementation. Agents working this plan **do not need to ask again** for anything in this list. Each use is recorded in the package handoff (work-package document §4.2).

| # | Authorised without asking | Limits |
| - | - | - |
| A1 | **Jev (TypeSafe) calls, live, at any time and in any volume needed by this plan.** This covers re-recording `data/eval/profile/private/*` and intake recordings, live evaluations, live benches, and measuring call counts, tokens and elapsed time. The key comes from `.env` (`SITEWISE_JEV_API_KEY`). | Never print or commit the key. Recordings stay in git-ignored private paths. |
| A2 | Commit all work, including previously uncommitted files in the working tree, on feature branches or worktrees, and **merge to the local `main` branch** once the package's gate passes. | Do not commit `.claude/` (agent worktrees and tool state), `.env` or private corpus content. Do not push to the remote unless the owner asks: the repository is public. |
| A3 | Start and stop the local PostgreSQL on port 5433 (`tools/dev.ps1`); create and reset the `sitewise_test` and `sitewise_dev` databases; apply migrations to them. | Local databases only. No VPS or production database. |
| A4 | Run every build, test, evaluation and bench command in this plan (`tools/check.ps1`, `go test`, `npm ci`, Playwright, `cmd/*`), and start the local server. | none |
| A5 | Create branches and worktrees, and use subagents for implementation, review and research. | Each package still gets a reviewer other than its implementer (work-package document §1). |
| A6 | Write and edit draft knowledge (`status: draft`), draft answer keys (`reviewed: false`) and evidence notes under `docs/evidence/`. | Never mark knowledge `reviewed` or answer keys reviewed. |
| A7 | Read private-corpus documents under `D:/AI Projects/Test Data` where a package needs them, for example WP-28 key drafting and re-recording. | Never copy their text or personal details into the repository. |
| A8 | Approval to commence (D-01, partial): the owner instructed implementation to start on 5 October 2026 with the first logical step (WP-00). | This approves starting work. It does not answer the Open product decisions in §5 (D-02 to D-07, D-09, D-11 to D-13, D-15, D-16, D-21 to D-24, D-29, D-30). Packages blocked on those stay blocked. Unblocked packages may proceed in plan order. |
| A9 | D-17 is delegated to the implementing agent, because it turns on measured cost and the owner has authorised Jev spend (A1). Apply the recommendation in §5: measure first; choose (a) re-record if the enlarged call stays within TypeSafe's documented limits; otherwise (b), recorded with its calls-per-passage count. | The choice and its evidence are recorded in §5 and in the WP-00 handoff. |

**Still owner-only:** product decisions marked Open in §5; marking anything `reviewed`; any VPS or production change; pushing to the remote; new third-party dependencies beyond what an approved decision names (`AGENTS.md`).

## 1. Owner overview

**What this wave builds.** It builds one shared record behind four outputs: the consultant RFP, the works RFT, the PMP and the cost plan.

- **Site and project.** The site is the lasting building record. The project is an intervention on it.
- **Work items.** Each work item is an action on a system at a part. It joins the physical building, the delivery dates and the money.
- **Proposals.** Code-only rules raise proposals, each with its reason. You accept or dismiss them.
- **Packages and costs.** Packages and cost lines reference work items by ID, so totals reconcile without counting money twice.
- **One assembler.** A single assembler builds the RFP (0991 first), then the RFT, then the PMP, from consistent saved revisions.

The stack stays as it is: Go, PostgreSQL, Jev as the only AI, local files, the existing jobs and SSE.

**What must happen before any coding:**

1. **The profile regression gate is broken on `main` today.** This is executed evidence: `go run ./cmd/profile-eval` and the Hale replay both report "evidence recording is stale" at HEAD, while the reviewed commit passes 15/15.
   - The probable cause is the K4 knowledge merges. They grew failure-mode detectors, which ride in every evidence Jev call, from 23 in the fire cluster (and about 172 overall on 29 September) to 859.
   - Stage 1's gate ("answer keys still pass") cannot be met until this is resolved. WP-00 and D-17 cover it.
2. **Your decisions in §5.** Eight are blocking and need an answer before their packages start:
   - D-03: sequencing of the gap check and delivery records.
   - D-04: site and project value split.
   - D-07: mixed interventions.
   - D-09: responsibility rules.
   - D-15: export renderer.
   - D-18: speed budgets.
   - D-05: existing-condition ownership; blocks WP-20.
   - D-06: eligibility of user inputs for derivations; blocks WP-15.
3. **Approval of the K0 shapes, which are already merged.** The PRG gate says "owner approves shapes". I found no record of that approval.

**The sequence I recommend:**

- WP-00 restores the gate.
- Stage 1 (sites and provenance) and the K-track run in parallel.
- Stage 2 builds work items and proposals.
- Stage 3 builds packages, costs and a minimal set of delivery records. The minimal delivery records are pulled forward from Stage 6 (D-03).
- Stage 4 delivers the RFP on 0991, then the RFT, then the PMP, then live reporting, then interaction polish.

**Recommended first package after approval: WP-00.** If you approve D-17 option (b) instead, start WP-00 and WP-11 together.

**Not ready to claim:** this plan is not "implementation-ready" in full. §5 lists open decisions, and the packages that depend on them are marked **Blocked** in the work-package document.

## 2. Repository findings

The evidence labels mean:

- **S**: static inspection.
- **X**: executed here on 5 October 2026.
- **D**: from repo documents, not re-verified.

### 2.1 Current behaviour relevant to the PRG

| # | Finding | Evidence | Kind |
| - | - | - | - |
| F01 | Profile replays fail at HEAD: `bankstown-page9: evidence recording is stale`; the Hale replay reports `hale-description: evidence recording is stale`. At 02094db (exported to the scratchpad with the same private recordings), the source replay gives 15/15 correct, 15/15 applied, 0 forbidden. | `cmd/profile-eval`, `data/eval/profile/private/*-recording.json` | X |
| F02 | Failure-mode detectors (`noul`) from **every** `failure_modes.yaml` are loaded into `Catalog.evidence` and asked in the evidence fan-out of any passage whose labels match `runs_on`. There is no cap on questions per call. Counts at HEAD: electrical-comms 143, envelope 226, fire 95, services-wet-air 208, structure 187, so 859 in all. The fire cluster had 23 at 02094db. Call size, tokens and elapsed time since K4 are **unmeasured**. | `internal/knowledge/load.go:311-330, 436-446`; `internal/jobs/worker.go:500-545` | S |
| F03 | The evidence cache key is the SHA-256 of the whole marshalled call, questions included. Any new or changed question re-reads every selected document on the next update. | `internal/jobs/source_reading.go` `cachedAsk` | S |
| F04 | The works layer (`knowledge/works/*`, `uc.*`, `cq.*`, `sig.*`) is validated by `tools/check_knowledge.py` but **not loaded by the Go runtime**. The loader reads only systems, rules, interfaces, failure modes, tables, determinants and profile. | `internal/knowledge/load.go:100-150`; commit `98ca0b2` | S |
| F05 | Checker at HEAD: 151 systems, 74 determinants, 227 rules, 144 interfaces, 859 failure modes, 1 consequence, 560 unforeseen conditions, 9 interface consequences, 331 signals, 8 actions, 2,000 dataset rows, 0 pending. `check_triage`: 0 errors. | `python tools/check_knowledge.py`, `python tools/check_triage.py` | X |
| F06 | `existing_building_year` exists (`knowledge/determinants.yaml:1507`) but has no `profile_group`, so `Harvest` never triggers it and Jev never reads it. The same applies to `existing_building`. The PRG's gap "No determinant records building age" (L110) is stale; the runtime gap remains. | `internal/knowledge/profile.go:216`; `internal/profile/harvest.go:80` | S |
| F07 | `profile_facts.part_label` has no producer. Jobs never set `PartLabel`, so every fact reconciles onto the whole part. Exact-label matching (L175) exists in `Reconcile` but is never exercised by real readings. | `internal/store/profile.go:154`; `internal/profile/reconcile.go` `partIndex` | S |
| F08 | `source.scope` has a fixed residential option list and is accepted at a hard-coded 0.6, not through `data/profile/thresholds.json`. It is stored in `passage_sources.scope` and not used to place facts. | `internal/jobs/source_reading.go` | S |
| F09 | `profile_user_values` and `profile_rows` have a foreign key on `(org_id, part_id)` only. The project that owns a part is checked in SQL `WHERE EXISTS`, not by a constraint, so the database does not enforce same-project. | `internal/db/migrations/008_project_profile.sql` | S |
| F10 | `profile_user_values.version` is incremented, but no write checks an expected version: last write wins. | `internal/store/profile.go` `SetUserValue`, `SetScope`; `internal/httpapi/profile.go` `putProfileValue` | S |
| F11 | Scope is stored as `scope.<leaf>` = `in`/`out` user values on the whole part. `Build` derives suggested scope from `ScopeDefaults` and evidence-included presence (amber, green or user bands). Refurb, remediation and advisory start empty. | `internal/profile/build.go`; `internal/knowledge/profile.go` `ScopeDefaults` | S |
| F12 | `hdr.work_type` is a single choice with options `new`, `refurb`, `extend`, `remediation` and `advisory`. These match `knowledge/works/actions.yaml` `work_type_defaults`. A determinant `work_type` ("Nature of the building works", `knowledge/determinants.yaml:485`) also exists, but it has **no options** and no `profile_group`. Several K4 `uc.*` records use `{det: work_type, any_of: …}` against it, so they "check nothing yet" (Pass B report). | `knowledge/profile/taxonomy.yaml:9-20`; `knowledge/determinants.yaml:485`; `docs/unforeseen/pass-b-reports.md` | S, D |
| F13 | `ReplaceSource` deletes and re-creates passages with new UUIDs and deletes the document's `profile_facts`. **Passage IDs are not stable across reprocessing.** | `internal/store/sources.go:49-66` | S |
| F14 | Existing budgets (`bench/budgets.json`) include `project_profile_read` 50/150 ms and `profile_edit` 50/150 ms. **There is no `profile_rebuild` path.** It was folded into `profile_edit` (stated in `docs/evidence/2026-10-03-profile-scope.md`, "Deviations"), although the design note lists it at 100/300 ms on Spec Home. The PRG (L409) says "must keep `profile_rebuild` (p50 100 ms, p90 300 ms on Spec Home)". | `bench/budgets.json`; `docs/design/2026-10-03-profile-scope.md` §7 | S, D |
| F15 | Rebuild: one transaction under a per-project advisory lock. It deletes and re-inserts `profile_rows`, upserts `profile_builds (built_at, thresholds_version)` and appends a `profile` event. There is no revision, input fingerprint or knowledge version. | `internal/store/profile.go` `RebuildProfile` | S |
| F16 | Profile edits in the UI are **not optimistic**: `await`, then re-render, with `catch` showing an error. Bands in use: user, green, amber, red, suggested, unchecked, blank. | `web/src/Profile.tsx:89-130, 279-325` | S |
| F17 | Thresholds: `profile-3`, `approved_by_owner: false`. Every shape has `green: null`, so green is withheld and amber floors are 0.6. There is no `action` or location shape. | `data/profile/thresholds.json` | S |
| F18 | Jobs: one durable job per document and kind (`jobs_document_kind_uq`). Kinds are `full_text`, `label` and `evidence`, with priority and leases. A profile update queues only readable documents. Failed work retries on an explicit click, and HTTP 402 is surfaced as `payment_required`. | migration 003 (`jobs_document_kind_uq`), migration 004 (priority, leases); `internal/store/profile.go` `RequestProfileRead`, `ReadProfile` | S |
| F19 | No tables exist for sites, work items, packages, costs, delivery, reports or proposals. There are no routes for them. | `internal/db/migrations/001-010`; `internal/httpapi/projects.go` routes | S |
| F20 | Pure Go tests pass: `internal/profile`, `internal/knowledge`, `internal/latency`. **Database tests were not run**: the test DB at `127.0.0.1:5433` was not listening, so `internal/store`, `internal/jobs` and `internal/httpapi` fail with `SITEWISE_TEST_DATABASE_URL is required`. | `go test` | X |
| F21 | The profile eval manifest holds Newham, Hale, Petersham, Rutherford and Mornington, with answer keys `reviewed: false`. 0991 and 0777 are not in it. The private corpus folders `0991-hydrant-sprinkler-head-upgrade` and `0777-city-tower-fitout` exist under `D:/AI Projects/Test Data`. I listed folder names only and read no content. | `data/eval/profile/manifest.json` | S |
| F22 | The Clerk reference data is present: `../clerk/data/taxonomy/{asset-register,consultant-rosters,disciplines,complexity-dimensions,…}.json` and `../clerk/data/seed/*.md`. | filesystem | S |
| F23 | The K4 owner lists have been processed: batches 1 and 2, 1,000 rows each, with triage and Pass B records plus ledger overlays (`knowledge/works/coverage/<batch>/<cluster>.yaml`). The checker reports 0 pending. Owner review is still pending. The cross-cluster merge pass and the open items in `docs/unforeseen/pass-b-reports.md` are not done. | commits `d250243`…`f0c170a` | S, X |
| F24 | The deploy package ships neither `knowledge/` nor `data/profile/`. This is a pre-existing gap noted in `docs/evidence/2026-10-03-profile-scope.md`. Catalogues "loaded at startup" (L347) depend on it. | `deploy/package.sh` (D) | D |

### 2.2 Limitations

- The private corpus is present, but I read no document in it. 0991 and 0777 content, budgets and appointed consultants are **unverified**. Packages WP-28, WP-K6 and WP-45 depend on them.
- Recording a live Jev run needs `SITEWISE_JEV_API_KEY` and owner consent to spend. I recorded nothing.
- The target VPS has not been measured. Every new budget is a target (§8).
- `docs/unforeseen/batch2` has been processed. The PRG's "batch 2 in progress" (L391) is stale.
- I did not read TypeSafe's live documentation pages during planning. Packages that use Jev must read and cite them at implementation time (`AGENTS.md` rule 2).

## 3. Architecture

### 3.1 Boundaries (unchanged)

- **One Go binary**: API, workers, the embedded SPA, SSE and PostgreSQL 17. Files sit on local disk by content hash.
- **Jev is the only runtime AI.** It is asked only by background jobs (`label`, `evidence`) through `cachedAsk`.
- **Rebuilds, reads, edits, proposals, gap checks, totals and report assembly are code-only.**
- **Ruled out:** no LLM, embeddings, agent harness, vector or graph database, additional service, workflow platform, universal dependency engine or untyped entity table.
- **Research agents are outside the runtime.** K1 to K6 produce draft YAML in `knowledge/` only.

### 3.2 Ownership model: one authoritative store per fact

| Fact | Authoritative store | Projections (rebuilt, never input) |
| - | - | - |
| What exists on a site: parts, class, determinants, scale, existing systems and condition | `sites`, `project_parts` (site-owned), site-scope `profile_user_values` and `profile_planning_values`; document readings in `profile_facts` | `profile_rows` (per project view) |
| What a document says | `profile_facts` (project and document owned), `passage_sources`, `document_sources` | `profile_rows` |
| User word on a profile key | `profile_user_values` | `profile_rows` |
| Assumptions and calculated planning values | `profile_planning_values` | `profile_rows`, report citations |
| Scope of works | `work_items` (D-08) | `profile_rows` `scope.<leaf>` (compatibility only, D-08) |
| Proposal outcomes | `proposal_decisions`, plus the records created on accept | `proposals` (rebuilt list with reasons and rank) |
| Contract boundaries | `packages`, `package_stages`, `package_scope_items` | gap check (computed on read) |
| Dates, risks, approvals, decisions, actions | `project_delivery_items`, `delivery_dependencies` | report sections |
| Money | `cost_plan_versions`, `cost_item_revisions`, `cost_values` | totals (computed on read), profile summary, report tables |
| Report content | `report_versions` (draft sections, protected edits); issued snapshot (immutable) | export file (content-addressed blob) |
| Library knowledge | `knowledge/**` YAML, loaded at startup, content-hashed (WP-14) | none |

**No competing stores.**

- The profile shows cost summaries read from the cost tables (L316).
- Scope chips read `work_items`.
- Report sections copy values only into the issued snapshot, which is an immutable historical record, not a working store.

### 3.3 Internal Go packages

Following PRG L128 ("separate internal packages … are sufficient"):

| Package | Owns |
| - | - |
| `internal/profile` (existing) | Pure reconciliation; adds site/project routing, eligibility, work-item reconciliation |
| `internal/works` (new) | Pure work-item rules (default action, split invariants), predicate evaluation over work items, proposal engine (ranking, fingerprints) |
| `internal/procurement` (new) | Packages, responsibilities, gap check (pure) |
| `internal/costs` (new) | Money arithmetic and central rounding, subdivision, rollups, reconciliation (pure) |
| `internal/delivery` (new) | Delivery item validation by kind, dependency cycle check (pure) |
| `internal/reports` (new) | Assembler, citations, edit merge, snapshot, page-fit model (pure, except the renderer adapter) |
| `internal/store` (existing) | All SQL; one file per domain (`sites.go`, `works.go`, `proposals.go`, `packages.go`, `costs.go`, `delivery.go`, `reports.go`) |
| `internal/httpapi` (existing) | Routes; one file per domain |
| `internal/knowledge` (existing) | Loads the works layer, clause and benchmark catalogues; content hash |

### 3.4 Write-path contract (all new mutable records)

1. **Authenticate.** `originOK`, then `memberSession`. The org comes from the session only. Path IDs are UUID-validated.
2. **Validate at the boundary.** Vocabularies come from the loaded catalogue. Return 422 with a short message.
3. **One transaction**, starting with `pg_advisory_xact_lock(hashtextextended(org||'/'||project,0))`, the same lock as `RebuildProfile`.
4. **Check the expected version.** On a mismatch, return `409` with JSON `{"error":"version_conflict","current_version":n}`. A wrong org, project or site returns `404`, with no existence leak.
5. **Write the authoritative rows.** Increment the relevant `project_revisions` counter (WP-14). If profile inputs changed, run the code-only rebuild in the same transaction.
6. **Append an event in the same transaction** (`appendEvent`). Commit, then respond with the updated read model.

### 3.5 Events (SSE)

Use the existing `events` table, per-org counters and `/events` stream. The payload is JSON text.

| kind | payload | emitted by |
| - | - | - |
| `profile` (existing) | `{"project_id","revision"}` (adds revision) | rebuild (now also covers work-item reconciliation and the proposal list) |
| `works` | `{"project_id","revision"}` | work-item writes, proposal decisions |
| `packages` | `{"project_id","revision"}` | package, stage and scope-item writes |
| `costs` | `{"project_id","plan_version_id","revision"}` | cost writes, baseline |
| `delivery` | `{"project_id","revision"}` | delivery writes |
| `report` | `{"project_id","report_id","version_id","state"}` (`state`: drafting, ready, stale, issued, failed) | assembler, issue |

The client refetches on an event and never trusts payload data as content.

### 3.6 Refresh and staleness (PRG L353-363)

- **Counters.** `project_revisions` holds monotonic per-domain counters, incremented inside each write transaction. These are the "explicit domain revision dependencies" of L353.
- **Freshness.** Each projection or report records the counters it read. "Stale" means a dependency counter has moved. There is no general dependency engine; a static Go map lists each report kind's dependencies.
- **Evidence changes.** These include reading choice, a new document, reprocessing or a Jev result. They increment `profile_inputs` through the existing rebuild. Knowledge changes appear as a new `knowledge_version` (content hash), compared at read.
- **Jev work.** Only `RequestProfileRead` and existing jobs queue Jev work, and only for readable documents (F18). Repeated requests coalesce on `jobs_document_kind_uq`. Filing keeps its priority lane; the new evidence load is verified against `whole_intake` (WP-00, WP-27).

## 4. Logical schema

These are names and types for implementation, not final DDL. Every table below gets these unless stated:

- `org_id uuid NOT NULL` and a primary key starting with `org_id`.
- Composite foreign keys that carry `org_id` and, where relevant, `project_id` or `site_id`. This is how the database enforces same-tenant, same-project and same-site ownership (PRG L322).
- `version bigint NOT NULL DEFAULT 1` on mutable rows.
- `created_at` and `updated_at`.

Controlled vocabularies are `CHECK` constraints. Migration numbers are reserved in the work-package document §2.

### 4.0 Why each new table is needed

This answers the peer-review question "Is the proposed schema the smallest adequate extension?" (L421). Tables the PRG names are not repeated here. Each table below goes beyond the PRG's entity list and is an **INT addition needing owner approval** (NW-REQ-380):

| Table | Why the PRG's tables are not enough |
| - | - |
| `proposals` (projection) | The ranked list with reasons must be readable without recomputing on every GET. It is rebuilt with the profile, like `profile_rows`, and is never input. |
| `project_revisions` | One row of counters is the "explicit domain revision dependencies" (L353), without a dependency engine. |
| `report_edits` | Protected edits need their own version and base-content hash (L361). They can be folded into `report_versions.sections` JSON if the owner prefers, at the cost of per-edit versioning. |
| `package_stages` | Fee lines name "a package and stage" (L308), and novation "splits its stages" (L296). A stage needs an ID for both. |
| `cost_item_revisions` | Implements L338's "stable item identity; versioned parent, code, label…": the identity row and the per-version attributes are separate, so a baseline freezes attributes. |
| `adoption_events` | Only for the L411 measure (D-21). It can be dropped if the owner times the run by hand. |

Columns that are interpretations: `work_items.inclusion` (`excluded` keeps the user's "out" final, L144), `user_touched`, `coarse_key`, and the `PATCH /api/sites/{site}` route (site label, address, lot). All are INT.

### 4.1 Provenance columns (shared shape, D-06)

These columns are added wherever the PRG asks for "origin, review status, meaning, provenance":

| Column | Type | Rule |
| - | - | - |
| `origin` | text NOT NULL | `document` \| `user` \| `calculation` \| `assumption` |
| `review_status` | text NOT NULL | `proposed` \| `accepted_for_planning` \| `verified` \| `superseded` |
| `meaning` | text NOT NULL | `stated` \| `requirement` \| `allowance` \| `forecast` |
| `provenance` | jsonb NOT NULL DEFAULT `'{}'` | `{sources:[SourceRef], actor, at, rationale, method:{id, knowledge_version}, inputs:[{ref, origin, review_status}], limitations}` |
| `verified_by` / `verified_at` / `verification_basis` | uuid, timestamptz, text NULL | `CHECK (review_status <> 'verified' OR (verified_by IS NOT NULL AND verification_basis <> ''))` |

**`SourceRef`** is a snapshot. It never relies on the passage ID alone:

```
{document_id, file_sha256, filename, document_number, revision, page, location, section, start_offset, end_offset, excerpt, passage_id?}
```

### 4.2 Sites and parts (WP-11)

**`sites`**

- Columns: `org_id`, `id`, `label text NOT NULL CHECK 1..120`, `address text`, `lot text`, `version`.
- `PK (org_id,id)`; FK `org_id → orgs ON DELETE CASCADE`.
- Rows are never deleted while any project references them.

**`projects`**

- Adds `site_id uuid NOT NULL` (after backfill), with FK `(org_id,site_id) → sites`.
- Adds `UNIQUE (org_id,id,site_id)` as the composite-FK anchor.
- Adds `UNIQUE (org_id,site_id)`, named `projects_one_per_site_v1`, which enforces "Version 1 has one project per site" (L21). Drop it when multi-project sites arrive.

**`project_parts`**

The name is kept for compatibility; the table becomes site-owned.

- Adds `site_id NOT NULL`, with FK `(org_id,site_id) → sites ON DELETE CASCADE`.
- `project_id` becomes `created_by_project_id uuid NULL`, with FK `ON DELETE SET NULL`. Parts no longer cascade from a project.
- `UNIQUE (org_id,site_id,label)` replaces `(org_id,project_id,label)`. `UNIQUE (org_id,site_id,id)` is added as the FK anchor. A partial unique index allows one `kind='whole'` part per site.
- The `kind` CHECK adds `roof` and `plant_area`.
- The whole part keeps the label "Whole project" in v1, so nothing visible changes.

**Backfill, in one migration transaction:**

1. For each project, insert a site with `label = project.name`, using a deterministic `id`: uuid v5 of `"site:"||project.id`. This makes reruns safe.
2. Set `projects.site_id`, then `project_parts.site_id` from each part's project, and `created_by_project_id = project_id`.
3. Validate that the counts match, and that no part's site differs from its project's site. Fail the migration otherwise.

### 4.3 Profile values (WP-12, WP-13)

**Key-scope registry (D-04)**

The registry lives in `knowledge/profile/key_scope.yaml`, which is data, `status: draft` and owner-reviewed.

- It maps each key family (`hdr.building_class`, `hdr.subclass`, `hdr.scale.*`, `det.*`, `fact.*`, `hdr.work_type`, `hdr.cond.*`, `sys.*.presence|provider|note|action|existing`, `scope.*`) to `site` or `project`, with explicit per-key overrides.
- Unclassified keys fail the checker and fail startup.

**`profile_user_values`** (authoritative)

- Adds `site_id NOT NULL` and `scope text CHECK (site, project)`. `project_id` becomes nullable: NULL iff `scope='site'`.
- Adds the provenance columns (§4.1), with `origin` limited to `user` or `assumption`.
- Adds `value_state text NOT NULL CHECK (set, cleared, unknown)`. The migration maps existing `value IS NULL` to `cleared`, which keeps today's "cleared by user" meaning; `unknown` is the new explicit unresolved value (L274).
- Uniqueness becomes two partial unique indexes, one per scope, replacing the PK. A surrogate `id` becomes the PK.
- Composite FKs: `(org_id,site_id,part_id) → project_parts(org_id,site_id,id)` and `(org_id,project_id,site_id) → projects(org_id,id,site_id)`. The second is MATCH SIMPLE, so it is skipped when `project_id` is NULL.
- Reset still deletes the row.
- Writes check the expected version (F10).

**`profile_rows`** (projection)

- Adds `site_id`, `scope`, `origin`, `review_status`, `meaning` and `value_state`. The `sources` JSON elements become `SourceRef`.
- The PK is unchanged, so each project keeps its own view, which includes its site's values.
- Existing bands are preserved and stay as the screen mark (L375: "show one mark per value on screen").

**`profile_planning_values`** (new, authoritative)

- Columns: `id`, `site_id`, `project_id NULL`, `scope`, `part_id`, `key` (registered in `knowledge/profile/planning_keys.yaml`), `value_state CHECK (set, unknown)`, `value_text`, `value_numeric numeric`, `value_bool`, `range_low numeric`, `range_high numeric`, `unit`, the provenance columns (`origin` IN user, assumption, calculation), `superseded_by uuid NULL` and `version`.
- `CHECK` that exactly one typed value is set when `value_state='set'`.
- One live row per (scope, part, key): partial unique index `WHERE superseded_by IS NULL`.
- The planning-key registry has no money-total keys (L329: "No duplicate ledger totals"). The checker and the boundary enforce this.

**`profile_facts`** is unchanged. It stays document-owned and project-scoped. During reconciliation, a site-scope key read from a project's document becomes a site row carrying that document's `SourceRef` (L138).

### 4.4 Revisions and fingerprints (WP-14)

**`project_revisions`**

- Columns: `org_id`, `project_id` (PK), and the counters `profile_inputs`, `works`, `packages`, `delivery`, `costs`, `reports`, all `bigint NOT NULL DEFAULT 0`.
- Writes update counters with `UPDATE … SET x = x+1 RETURNING x` under the project lock.

**`profile_builds`** adds:

- `revision bigint NOT NULL`, which is monotonic.
- `input_fingerprint text`: the SHA-256 of the sorted fact IDs and values, read settings, user and planning values, the work-item set, the thresholds version, the question version and the knowledge version.
- `knowledge_version text`: the SHA-256 of the loaded catalogue file contents in path order, computed at startup.
- `question_version text`.
- `inputs jsonb`: the component counters.

`profile_builds` remains the latest build per project. It is not a snapshot; issued reports hold their own snapshot (L92).

### 4.5 Work items (WP-20, WP-24)

**`work_items`**

Columns:

- Identity and placement: `id`, `project_id`, `site_id`, `part_id`, `system_id text NOT NULL` (a leaf or top-level system, validated at the boundary against the catalogue).
- Action: `action CHECK (new, replace, upgrade, alter, repair, remove, retain, investigate)` and `inclusion CHECK (included, excluded)`.
- Hierarchy: `parent_id NULL` and `is_group bool NOT NULL DEFAULT false`.
- Description: `title text NOT NULL`, `existing_condition text NULL CHECK (serviceable, nearing_end_of_life, end_of_life, beyond_economical_repair, failed, defective, unknown)`. **This column's existence depends on D-05**: under option (a) it is replaced by a read-through of the site key `sys.<leaf>.condition` plus `existing_condition_note`. WP-20 is blocked until D-05 is decided., `existing_condition_note text CHECK ≤120`, and `target jsonb NOT NULL DEFAULT '{}'`, shaped `{values:[{key,value,unit}], clause_refs:[{id,version}], text}`.
- Quantity: `quantity numeric NULL` and `unit text NULL`, both set or both null.
- Provenance: the provenance columns, plus `user_touched bool NOT NULL DEFAULT false` and `coarse_key text NULL`.
- Lifecycle: `source_proposal_key text NULL`, `retired_at`, `retired_by` and `version`.

Constraints:

- FKs: `(org_id,project_id,site_id) → projects(org_id,id,site_id)`; `(org_id,site_id,part_id) → project_parts(org_id,site_id,id)`; `(org_id,project_id,parent_id) → work_items(org_id,project_id,id)`, which requires `UNIQUE (org_id,project_id,id)`.
- `UNIQUE (org_id,project_id,coarse_key) WHERE parent_id IS NULL AND retired_at IS NULL`, where `coarse_key = part_id||'|'||system_id`. This prevents duplicate coarse scope (acceptance scenario AT-16).
- `UNIQUE (org_id,project_id,source_proposal_key) WHERE source_proposal_key IS NOT NULL`. This makes proposal acceptance idempotent.
- `CHECK (inclusion='included' OR origin='user')`: only the user excludes.
- Index on `(org_id,project_id,system_id)`.

**Coarse rows (D-08).** The code-only rebuild creates and maintains coarse rows for defaults and evidence:

- IDs are deterministic: uuid v5 of `project|part|system`.
- The rebuild only touches rows that are `review_status='proposed' AND user_touched=false`.
- A proposed row that loses its support is retired by the rebuild.
- A user-touched or accepted row is never changed by the rebuild. Disagreeing evidence shows as a conflict, read from `profile_rows` key `sys.<leaf>.action` (§4.6).

**Groups.** A group (`is_group`) is not physical work for the gap check or for proposal predicates. Its children are (D-09, D-19).

**Deletion.** Rows are never hard-deleted once any package scope item, cost revision, delivery item, proposal decision or report snapshot refers to them. Retirement instead sets `retired_at`. Retiring is refused with 409 and a list of blockers while live posting cost values or live responsibilities reference the row (D-19).

### 4.6 Readings for works (WP-22, WP-23)

- **Action fact.** `sys.<leaf>.action` facts reconcile like presence. Use a new threshold shape, `action`, with `green: null` until owner calibration. The row value is an action, `several`, or empty.
- **`several`.** Shows in Needs mapping (`passage_sources.outcome = needs_mapping`, `unresolved` includes the key). It is never applied as an action.
- **Location answers** become `profile_facts.part_label` for the readings of that passage, but only when the location answer is applied: amber-only until calibrated (D-26).
- **Location thresholds** are keyed `location.n<option-count>` in `thresholds.json`.

### 4.7 Proposals (WP-26)

**`proposals`** (projection, rebuilt in the profile rebuild transaction)

Columns:

- Key and source: `project_id`, `key text`, `record_kind CHECK (ic, cq, uc)`, `record_id`, `interface_id NULL` and `proposal_index int`.
- Target: `target_system_id NULL`, `target_part_id NULL`, `kind CHECK (investigation, work_item, discipline, approval, hold_point, obligation)`, `label` and `action NULL`.
- Reason: `trigger_work_item_ids uuid[]` and `reason jsonb` (`{record, interface|rules, triggers:[{work_item_id, action, system, part}], determinants:[{key,value,origin}], signals:[{id,state,sources}]}`).
- Ranking: `severity`, `specificity int` and `rank int`.
- State: `inputs_fingerprint`, `knowledge_version` and `state CHECK (open, dismissed, accepted, reopened, addressed_by_evidence)`.
- PK `(org_id,project_id,key)`.

**Key.** The key is `record_id|interface_id|target_system|target_part|proposal_index`. It deliberately excludes work-item IDs, so splitting an item into children with the same action, system and part keeps the key and its decision (D-19).

**Fingerprint.** `inputs_fingerprint` is the SHA-256 of the record's content hash and the sorted semantic trigger inputs: action, system and part per trigger; other-side existence; the determinant values the predicate read, with their origin; and the signal states.

**`proposal_decisions`** (authoritative)

- Columns: `id`, `project_id`, `proposal_key`, `record_id`, `trigger_work_item_id` (the first trigger, as listed in the PRG), `decision CHECK (accepted, dismissed)`, `inputs_fingerprint`, `rationale ≤200`, `actor`, `decided_at`, `created_record_type NULL CHECK (work_item, package, package_scope_item, delivery_item)`, `created_record_id NULL` and `version`.
- `UNIQUE (org_id,project_id,proposal_key)`.

**Display rules:**

- If the current fingerprint differs from the decision's, a dismissed proposal shows as `reopened`, with "inputs changed since dismissed" and the diff.
- Accepting writes the decision and the created record in one transaction.
- Repeating an accept returns the existing record (idempotent through the unique `source_proposal_key` on the target table).

### 4.8 Packages and responsibilities (WP-30, WP-31)

**`packages`**

- Columns: `id`, `project_id`, `kind CHECK (services, works, supply)`, `works_scope NULL CHECK (head_contract, trade)` (required iff works), `discipline_id text`, `title`, `novation bool NOT NULL DEFAULT false` with `CHECK (NOT novation OR kind='services')`, `lifecycle_status CHECK (proposed, planned, procuring, appointed, closed, cancelled)` (draft vocabulary, D-22), the provenance columns, `source_proposal_key`, `retired_at` and `version`.
- FK `(org_id,project_id) → projects`; `UNIQUE (org_id,project_id,id)`.

**`package_stages`**

- Columns: `id`, `project_id`, `package_id`, `stage_id text` (from `knowledge/works/stages.yaml`), `label` (editable), `ordinal`, `novation_phase CHECK (none, pre, post)`, `origin`, `retired_at` and `version`.
- `UNIQUE (org_id,package_id,label) WHERE retired_at IS NULL`.

**`package_scope_items`**

Columns:

- Placement: `id`, `project_id`, `package_id`, `item_kind CHECK (responsibility, obligation)`, `work_item_id NULL` and `role NULL CHECK (design, document, supply, install, test, certify, inspect, maintain_operation, protect)`.
- Wording: `clause_id NULL`, `clause_version NULL` and `user_text NULL`, with `CHECK (exactly one of clause_id or user_text)`.
- Detail: `stage_id NULL` (FK `package_stages`), `inclusion CHECK (included, excluded)`, `deliverable text NULL`, `interface_ids text[]` and `source_refs jsonb` (`SourceRef[]`: explicit links to interfaces and existing source records, L334).
- Provenance and lifecycle: the provenance columns, `source_proposal_key`, `retired_at` and `version`.

Constraints:

- `CHECK (item_kind='obligation' OR (work_item_id IS NOT NULL AND role IS NOT NULL))`.
- `UNIQUE (org_id,package_id,work_item_id,role) WHERE retired_at IS NULL AND inclusion='included'`. This prevents a duplicate row within a package. Duplicates across packages are allowed and reported as overlaps (L343).
- Composite FKs to `packages` and to `work_items` within the same project.

**Gap check (WP-32).** This is a pure function over live, included, non-group, accepted work items and live included responsibilities. Rules per D-09.

### 4.9 Delivery (WP-35)

**`project_delivery_items`**

- Columns: `id`, `project_id`, `kind CHECK (activity, milestone, action, risk, issue, decision, approval)`, `title`, `owner_text`, `owner_user_id NULL`, `baseline_date`, `target_date`, `forecast_date`, `actual_date` (all date NULL), `status text NOT NULL`, `as_of date NULL`, `package_id NULL`, `work_item_id NULL`, `stage_id NULL`, `details jsonb`, the provenance columns, `source_proposal_key`, `retired_at` and `version`.
- `status` and `details` are validated per kind in Go (L335). The CHECK covers the union of status values.
- Kind-specific details: risk `{likelihood, consequence, uc_record_id?}`; approval `{authority, reference, submitted_on, determined_on}`; decision `{options, chosen}`.

**`delivery_dependencies`**

- Columns: `project_id`, `predecessor_id`, `successor_id`, `type CHECK (finish_to_start)` (D-25) and `lag_days int NOT NULL DEFAULT 0`.
- `PK (org_id,project_id,predecessor_id,successor_id)` and `CHECK (predecessor_id <> successor_id)`.
- Go rejects cycles under the project lock.

### 4.10 Costs (WP-33, WP-34)

**`cost_plans`**

- Columns: `project_id` (PK, one per project), `draft_version_id` and `baseline_version_id NULL`.

**`cost_plan_versions`**

- Columns: `id`, `project_id`, `revision int` (unique per project), `status CHECK (draft, baseline, superseded)`, `currency char(3) NOT NULL DEFAULT 'AUD'`, `tax_basis CHECK (ex_tax, inc_tax)`, `tax_rate numeric(5,4) NULL`, `price_date date NULL`, `coverage text`, `funding_target numeric(18,2) NULL`, `funding_target_provenance jsonb`, `frozen_at NULL` and `version`.
- One draft per project: partial unique index on `status='draft'`.
- A trigger rejects UPDATE or DELETE on a frozen (baseline or superseded) version and its revisions and values. Frozen baselines keep approved budgets (L337).

**`cost_items`** (stable identity)

- Columns: `id`, `project_id` and `created_in_version_id`. The ID is never reused.

**`cost_item_revisions`**

- Columns: `plan_version_id`, `cost_item_id`, `parent_item_id NULL`, `code`, `label`, `line_kind CHECK (works, fee, project_wide)`, `category NULL`, `work_item_id NULL`, `package_id NULL`, `package_stage_id NULL`, `posting bool NOT NULL`, `quantity numeric NULL`, `unit NULL`, `rate numeric(18,4) NULL`, `rate_basis NULL`, `excluded bool NOT NULL DEFAULT false`, the provenance columns and `version`.
- `PK (org_id,plan_version_id,cost_item_id)`.
- Per-kind checks:
  - `works` → `work_item_id NOT NULL`.
  - `fee` → `package_id` and `package_stage_id NOT NULL`.
  - `project_wide` → `category NOT NULL`, from a catalogue that includes contingency, escalation, authority fees and exclusions.
- The parent is in the same version.

**`cost_values`**

- Columns: `plan_version_id`, `cost_item_id`, `metric CHECK (budget, estimate, commitment, claimed_to_date)`, `value_state CHECK (known, unknown)`, `amount numeric(18,2) NULL`, `low numeric(18,2) NULL`, `high numeric(18,2) NULL`, `as_of date NULL` with `CHECK (metric <> 'claimed_to_date' OR as_of IS NOT NULL)`, the provenance columns and `version`.
- `PK (org_id,plan_version_id,cost_item_id,metric)`: one value per version, item and metric (L338).
- A trigger, or the store guard plus a test, rejects values on non-posting items. Group values are always calculated.

**`scope_cost_links`**

- Columns: `project_id`, `package_scope_item_id` and `cost_item_id`. PK on all three. **It holds no money columns** (L339).

**Rule enforced in the store transaction and tested (L308):** when a works revision's `package_id` is set, that package must hold a live responsibility for the line's work item.

### 4.11 Reports (WP-41 to WP-43)

**`reports`**

- Columns: `id`, `project_id`, `kind CHECK (pmp, rfp, rft)`, `package_id NULL` with `CHECK (kind='pmp' OR package_id IS NOT NULL)`, `title`, `current_draft_version_id NULL` and `version`.

**`report_versions`**

- Columns: `id`, `report_id`, `project_id`, `number int`, `status CHECK (draft, issued)`, `reporting_date date`, `previous_issue_id NULL`, `source_revisions jsonb` (project_revisions counters, profile build revision and fingerprint, cost plan version id, knowledge, thresholds and question versions, app build commit), `template_id`, `template_version`, `sections jsonb` (stable section and item IDs), `budget_disclosed bool NOT NULL DEFAULT false`, `snapshot jsonb NULL`, `snapshot_sha256 text NULL`, `export_file_sha256 bytea NULL`, `issued_at`, `issued_by` and `version`.
- A trigger rejects any UPDATE or DELETE of a row whose status is already `issued` (L363).

**`report_edits`**

- Columns: `report_version_id`, `target_id`, `text`, `base_content_sha256` (the generated content when edited), `user_id` and `version`.
- `PK (org_id,report_version_id,target_id)`.

**`report_references`**

- Columns: `report_version_id`, `citation_id` (`E1`, `U2`, `C3`, `A4` …), `label CHECK (E, U, C, A)`, `anchor_id` (section, row or value) and `basis jsonb` (`SourceRef`, or user, actor and time, or a calculation method with inputs, or an assumption with its rationale).
- `PK (org_id,report_version_id,citation_id)`.

### 4.12 Migration order, compatibility and recovery

**Order:** 011 sites → 012 values → 013 planning → 014 revisions → 015 work items → 016 proposals → 017 packages → 018 scope items → 019 costs → 020 delivery → 021 reports → 022 adoption events.

Each migration is additive, then backfill, then tighten (NOT NULL or new constraint), inside one transaction, with an assertion block that raises on any count or ownership mismatch.

**Compatibility:**

- `PUT /projects/{id}/profile/scope` keeps its request and response shape. From 015, it writes work items. `profile_rows` `scope.<leaf>` projection rows keep the profile UI and tests unchanged until WP-70.
- The existing answer keys read the same keys.

**Backfill for scope (015).** Each `scope.<leaf>` user value on the whole part becomes a `work_items` row:

- `in` → `inclusion='included'`, origin `user`, `accepted_for_planning`, `user_touched=true`, with the action from the project's work-type default (D-07).
- `out` → `excluded`.

The old `scope.*` user values are then deleted in the same transaction, so there is one store. The migration's assertion compares the counts before and after.

**Backfill for values (012):** each user value is classified by the registry. Site-scope values move to `project_id NULL, scope='site'`.

**Recovery and backups:**

- Every migration gets a dry run before merge. Restore the latest backup into a scratch database (`deploy/restore.sh` procedure), apply the new migration, and confirm its assertion block passes. Then compare row counts of untouched tables before and after with a SQL script kept beside the migration. `restore-check` (`cmd/sitewise/restore.go`) compares a source host with a restored host; it is used to prove the backup restores, **not** to test migrations. Migration 015 deletes `scope.*` user values, so its dry run is mandatory and a fresh backup is taken immediately before production apply.
- Rollback is by restore, not down-migrations; the repo has none.
- Storage replication is a separate prerequisite and is not bundled (L417).

## 4.13 Decisions resolved during implementation

| Decision | Resolved | By | Outcome | Evidence |
| - | - | - | - | - |
| D-17 | 5 October 2026 | implementing agent under A9 | **(a) re-record.** Enlarged evidence calls stay well within jev-1.13's 64k-token request limit (largest real case about 22k); one call per passage is kept. Recordings were re-recorded live: source 15/15, Hale 12/12, 0 forbidden. | `docs/evidence/2026-10-05-evidence-workload.md` |

Findings added during WP-00:

- **F25.** The all-systems worst-case evidence call exceeds 64k tokens: about 144k at `02094db`, about 210k now. No real passage measured comes close, but WP-22 and WP-27 must measure their worst real passage against the limit.
- **F26.** `TestRuleBudget` (`internal/intake`, 1,000 µs p90) failed once at 1,021 µs during a full parallel `go test ./...`, then passed 3 out of 3 in isolation. Treat it as a timing flake on this host, not a regression.
- **F27.** `tools/check.ps1` run with `pwsh -File` from this agent shell fails at `npm --prefix web ci` (`Unknown command: "pm"`). The PowerShell npm shim mangles the arguments here. The same steps pass when run from bash. The gate script was left unchanged.
- **F28. The intake gate was already red before this wave.** At `02094db`, `intake-eval -replay` failed with 129 unrecorded Jev requests: the recordings dated from 1 October, before later filing changes. They were re-recorded live (A1): 129 calls, 0 errors, 0 replay misses; aggregate metrics are in `data/eval/intake/results/live-latest.json`.
  - The gate still stops at "no accepted baseline: review this report, then run with -accept".
  - Accepting a baseline approves filing accuracy, which is an **owner** decision. It is not covered by §0.1, and I did not accept one.
  - Note for the owner: held-out title accuracy on this run is 0.29, with 15 false-confident titles.
- **F29. The latency bench also failed before this wave.** `cmd/intake-bench` refuses to run when requests have no recorded document: 178 before the re-recording, 52 after. The remainder are probably the 55 documents with no text layer (OCR path), which the eval run does not record.
  - The latency gate is therefore not runnable locally, and the `whole_intake` timing under background load (WP-00 step 4) is **not verified**.
  - Owner decision: re-record the OCR path or exclude those documents from the bench.

## 5. Decisions

The status values mean:

- **Open**: needs the owner.
- **Recommended**: technical default, proceed unless overruled.
- **Blocking**: dependent packages wait.

| ID | Decision (sources) | Options | Recommendation and consequences | Status | Affects |
| - | - | - | - | - | - |
| D-01 | Approve the PRG direction and this plan (L1, L399) | approve / amend | Required before any code. "Recheck HEAD before coding." | Open, blocking all | all WP |
| D-02 | Primary user is the owner-side PM (L27, "assumed; owner to confirm") | confirm / change | Confirm. It changes report wording and default packages, not the schema. | Open | WP-40, WP-45, WP-30 |
| D-03 | **Sequencing.** Stage 2 includes the gap check (L401), but packages arrive in Stage 3 (L403). First reports need dates, risks and approvals (L404-405) before Stage 6 (L406). | (a) Move the gap check and the accept path for `discipline`, `obligation`, `approval` and `hold_point` proposals into Stage 3, and add a minimal delivery-records package (WP-35) to Stage 3. (b) Create a skeleton packages table in Stage 2. | **(a).** The smallest change, and it adds no throwaway schema. In Stage 2, proposals of those kinds can be dismissed but not yet accepted ("accept after packages exist"). Stage 6 keeps progress, changes-since-issue and live summaries. | **Open, blocking** WP-32, WP-35 | WP-26, WP-30-32, WP-35, WP-45 |
| D-04 | **Site and project value split and storage** (L136, L328; peer question L421) | (a) One `profile_user_values` table with `scope` and nullable `project_id`, plus a key-scope registry. (b) Separate `site_values` tables. | **(a).** One store, one reset path, unchanged keys. Proposed classification: header class, subclass and scale, `det.*` and existing-system condition → site; `hdr.work_type`, `hdr.cond.*`, `fact.*`, `sys.*.presence/provider/note/action` and scope → project. Determinants that describe the works rather than the building, such as `existing_building` ("Work to an existing building"), are listed for owner classification. | **Open, blocking** WP-12 | WP-12, WP-15, WP-21 |
| D-05 | **Existing condition ownership.** The PRG puts it on the work item (L148) and among site values (L136). | (a) Authoritative on the site (key `sys.<leaf>.condition` on the part); the work item shows it read-through, plus a project note. (b) A snapshot field on the work item. (c) Both, with the site value updated on accept. | **(a).** One store, and it survives the next project (L21). In v1 (one project per site) the user sees the same thing. This deviates from the literal field list at L142, so owner approval is needed. | Open | WP-20, WP-24 |
| D-06 | **Eligibility for derivations** (L278: "accepting a planning assumption must not make it eligible…") | (a) User `stated` values stay eligible, as today in `usableFacts`; `origin=assumption`, `meaning` allowance or forecast, and planning values are never eligible. A derived value whose inputs include unverified user values is labelled `accepted_for_planning`, not `verified`. (b) Only `verified` inputs are eligible. | **(a).** It preserves current derivations and answer keys and adds the bar. (b) would blank most derived rows today. Band outcome: under (a), a derivation from user-stated inputs keeps today's green band (`derive` in `reconcile.go`), with `review_status=accepted_for_planning`. Showing amber instead is a further option; it would change displayed bands. | **Open, blocking** WP-15 (answers peer-review question L421) | WP-13, WP-15 |
| D-07 | **Mixed interventions** (L106, Hale) | (a) The project `hdr.work_type` stays singular (the primary type); an optional part-level `hdr.work_type` value overrides it for default actions; a `work_type` predicate (any part or project value) is added for knowledge. (b) Make `hdr.work_type` multi-choice. (c) Derive the work type from work items. | **(a).** The answer keys for the singular header are unchanged, and Hale gets per-part defaults (extension part → new, existing part → alter). It also makes the K4 `{det: work_type}` records evaluable (F12). To avoid a name collision, give the existing `work_type` determinant the five taxonomy options, with its value supplied by code from the part or project work type (never asked of Jev), rather than adding a new operator. (b) changes a Jev question and the answer keys. | **Open, blocking** WP-21 | WP-20, WP-21, WP-25, WP-K0 |
| D-08 | **Where coarse proposed work items live** (L144, L150, L328 "rebuild projections rather than … authoritative") | (a) Rows in `work_items` with deterministic IDs, owned by the rebuild only while `proposed` and untouched. (b) A separate projection, materialised on accept. | **(a).** Stable IDs exist before acceptance (cost lines and reports can reference them), there is one table, and the ownership rule is explicit and tested. | Recommended | WP-20 |
| D-09 | **Responsibility semantics and gap rules** (L343, user gap 2) | See the rule set below this table. | Recommend the rule set below. It adds `needs_design` per action to `actions.yaml` (knowledge, draft, owner review). | **Open, blocking** WP-32 | WP-31, WP-32, WP-K0 |
| D-10 | **Which work items predicates see** (L262, L183) | (a) In-scope = `inclusion='included'`, not retired, not a group, any review status; a proposal from an unaccepted trigger is labelled. (b) Accepted only. | **(a).** Proposals appear on the first run (the adoption measure, L411); the labels keep them honest. `system_existing` is true iff the site records the system as existing **or** any live work item on it has an action other than `new`, **and** no live work item replaces or removes it. Treating `remove` as "not existing" is an interpretation. | Recommended | WP-25, WP-26 |
| D-11 | How many proposals show (L179, "owner sets … after the first run") | owner value | The interim default shows the top 10 by rank, with the rest one click away. The value lives in `data/profile/proposals.json` and the owner sets it after the 0991 run. | Open, non-blocking | WP-26 |
| D-12 | **Signals and the code-only rebuild** (L245 vs L179; user gap 6) | (a) Ask signals in the existing evidence fan-out by `runs_on` labels, like failure-mode detectors. Store the answers as facts `sig.<id>`. The rebuild reads them in code and marks a proposal `addressed_by_evidence` (still shown, never auto-dismissed). (b) Ask signals only for live proposals. | **(a).** It is deterministic and cache-friendly, and the rebuild stays Jev-free. (b) would make reading depend on work items and cause re-reads. This depends on the workload outcome of D-17. | Open | WP-27 |
| D-13 | **Forecast** (L316 "budget/forecast variance", L338 metrics exclude forecast; user gap 7) | (a) Compute the forecast: for each posting leaf, commitment if known, else estimate, else budget, with the basis shown; null if all are null. (b) Add a stored `forecast` metric. | **(a).** It adds no ledger value and is computed on read. The PMP shows variance = forecast − budget, with its basis. | Open | WP-34, WP-51 |
| D-14 | **Money** (L306) | numeric precision and rounding | Amounts `numeric(18,2)`, rates `numeric(18,4)`. A single Go function `costs.Round` rounds half away from zero to cents, applied once at the line amount (quantity × rate). Totals are SQL `numeric` sums of rounded lines. Go carries money as `big.Rat` or strings (no new dependency). Currency is AUD by default and explicit; GST basis is per plan version, with a per-value override not supported in v1. | Recommended | WP-33, WP-34 |
| D-15 | **Export renderer** (L409 "after choosing its renderer"; L377) | (a) A pure-Go PDF library (a new dependency, Lane C, needs justification). (b) Headless Chromium (a heavy binary; arguably an "additional service"). (c) An external CLI typesetter binary. | No recommendation until a one-day spike measures fidelity, size and time against a fixed fixture. Report assembly does not depend on the renderer; only WP-44 is blocked. | **Open, blocking** WP-44 | WP-44, WP-45 |
| D-16 | Minimum font size and page geometry (L15 "at a readable size", L377 "Set a minimum font size") | owner value | Propose A4, 10 pt body, 8.5 pt minimum in tables, 15 mm margins. | Open, blocking WP-44 | WP-44 |
| D-17 | **Restore the regression gate** (F01, F02) | (a) Re-record the source and Hale recordings live against HEAD, with owner consent and spend, after measuring the new call size. (b) Route failure-mode detectors and signals out of the profile evidence call, into a separate background stage or cache stage, so presence and provider calls regain their fingerprints. (c) Cap questions per call. | Measure first (WP-00 step 1). Calls per labelled passage: (a) 1 evidence call (larger); (b) 2 (profile evidence + detector stage); (c) 1 or more, depending on the cap. Option (b) adds a second Jev call per passage in a **background** state. `AGENTS.md` rule 3 ("one Jev fan-out per state; never serial round trips on a hot path") permits that only if it is a separate state off any hot path, so the owner must accept that reading. Read the TypeSafe [API](https://docs.typesafe.ai/api) and [jev-1.13](https://docs.typesafe.ai/model-jaggedness/jev-1.13) pages for limits on questions per call before choosing. The 331 signals (WP-27) land in the same call, so this decision covers them too. Recommendation: (a) if the measured call stays within documented limits and acceptable tokens, otherwise (b). | **Open, blocking** Stage 1 gate | WP-00, WP-22, WP-27 |
| D-18 | **Budget conflicts** (L409 vs `bench/budgets.json`) | The PRG proposes "saved reads/writes p50 ≤100, p90 ≤250 ms"; existing `project_profile_read` and `profile_edit` are 50/150. `profile_rebuild` 100/300 is cited but not gated (F14). | Existing paths keep 50/150 (no loosening). New CRUD paths use 100/250. Add a `profile_rebuild` bench path at 100/300 on the larger Spec Home and 0991 fixtures. `profile_edit` keeps 50/150 on its existing bench project. These two can conflict: an edit on Spec Home includes a rebuild allowed 100 ms p50. The owner must choose one rule: (i) `profile_edit` 50/150 applies on every project, so the rebuild must be well under 50 ms; or (ii) `profile_edit` is measured on the bench project only, and large projects are governed by `profile_rebuild`. Recommend (ii), stated explicitly. **WP-26 step 0** measures evaluating the full ic/cq/uc set (~570 records) inside a rebuild before proposals join the edit path. If either budget breaks, stop and report the trade-off (`AGENTS.md`). | **Open, blocking** WP-14 bench | WP-14, WP-26, WP-71 |
| D-19 | **Split, retire and change effects** (user gap 9) | See the policy below this table. | Recommend the policy. | Recommended | WP-24, WP-31, WP-34, WP-41 |
| D-20 | **Issue reproducibility** (L363, L421; user gap 10) | See the policy below this table. | Recommend storing the canonical snapshot and the export blob. | Recommended | WP-43 |
| D-21 | Measuring "active minutes" (L411) | (a) Server log of user-initiated requests and SPA focus heartbeats, counting gaps under 120 s as active. (b) The owner's stopwatch. | Both on the first run, compared, as one measurement. The owner sets the target afterwards. The stored events are minimal (`adoption_events`: org, project, user, at, kind), with no content. | Open | WP-45 |
| D-22 | Default packages and lifecycle vocabulary (L296) | Data copied from Clerk `consultant-rosters.json` and `complexity-dimensions.json` into `knowledge/works/package_defaults.yaml` (draft) | Copy as data, never code. Owner review before suggestions show as anything other than draft. | Open | WP-30, WP-K0 |
| D-23 | Authorship of the 0991 and 0777 work-item keys (L401 "owner-drafted … `reviewed: false`"), and the meaning of "by ID only" (L413) | (a) The agent drafts and the owner reviews. (b) The owner drafts, as L401 literally says. Manifest: (i) ID, relative path and hash, the existing shape (`data/eval/profile/manifest.json`); (ii) ID only, with paths kept outside the repo. | No settled recommendation: L401 literally says owner-drafted. If the owner agrees, (a) with (i) matches existing practice. Both are interpretations until decided. | Open | WP-28 |
| D-24 | `uc.attaches_to` targets. The PRG says "a system, interface or action" (L222); K0 `SCHEMA.md` allows system, interface, stage or package kind, with no `action`. | accept K0 / add action | Accept K0 (stage and package-kind attachments are used by 49 delivery records). An action attachment is expressed through `when.works.action`. Owner approval. | Open | WP-K0, WP-26 |
| D-25 | Dependency types (L336 "supported dependency type") | finish-to-start only, plus lag | FS only, with integer lag days and cycle rejection. | Recommended | WP-35 |
| D-26 | Location options (L175) | Option IDs `p1…pn` mapped in code to part IDs; criteria text = part label and kind; plus `whole_project`, `specific`, `multiple`, `not_stated`. Threshold key `location.n<count>`. | Recommended. The fingerprint changes when parts change, which re-reads that project's evidence. This is acceptable because parts change rarely; record it. | Recommended | WP-23 |
| D-27 | **Existing-building presence** (L114, L152) | For `refurb`, `remediation` and `advisory` (project or part type), evidence presence with no action fact or `not_stated` writes a site row `sys.<leaf>.existing = present` (site scope), not a work item. `new` and `extend` keep today's behaviour. | Recommended. Hale's existing part follows its part-level type (D-07). Answer keys score `presence`, which is unchanged. Only scope inclusion moves. | Recommended | WP-21 |
| D-29 | **Tenant fit-out against base-building capacity** (L197, L413 AT-15) | `if.tenant-fitout-base-building-hvac` is type `loads`. `ic.loads-investigate-supported` fires only for new, replace or upgrade on `from`, while refurb defaults to `alter`, and its label speaks of "structure or ground". Options: (a) K1 re-types or adds the edge as `supplies` (base plant supplies the tenancy), so `ic.supplies-investigate-supply` fires for new, upgrade or alter; (b) add an interface-type-specific ic label or record. | (a) plus a reviewed label. AT-15 cannot pass on today's knowledge. | Open | WP-K1, WP-26, WP-K6 |
| D-30 | `ic.controls-test-link` has kind `investigation`, so accepting it creates an `investigate` work item. The `investigate` boundary (L169) says commissioning tests belong to that work. | (a) Change the kind to `obligation` (a `test` role on the package doing the work). (b) Keep it. | (a). Owner review in WP-K0. | Open | WP-K0, WP-26 |
| D-28 | Ranking specificity (L179) | `specificity` = 3 for a match on the exact leaf system and action, 2 for a parent system, 1 for an action-only match, 0 otherwise. Severity order: life-safety > other values in `SCHEMA.md` order. | Recommended | WP-26 |

**D-09 rule set (recommended):**

- **Physical actions are** `new`, `replace`, `upgrade`, `alter`, `repair` and `remove`. Each needs exactly one live **works** package holding `install`. For removal, `install` means "carry out the works"; the UI label reads "install / carry out".
  - "Supply and install" means that the same works package also holds `supply`.
  - An owner `supply` package holding `supply` does not count as `install`.
  - More than one package holding `install` is an overlap.
  - More than one holding `supply` is an overlap.
- **Design** is needed where `actions.yaml` marks `needs_design: true` (proposed: `new`, `replace`, `upgrade`, `alter`, and `repair` when the item is not a group). Such items need exactly one **services** package holding `design`.
- **`investigate`** needs exactly one package of any kind holding `inspect` or `test`, and needs no installer.
- **`retain`** needs at least one live works package holding `maintain_operation` or `protect`. This is an obligation and not physical work; duplicates are not overlaps.
- **Group parents** are not checked. Each child inherits the parent's responsibilities unless the child has its own for that role. A child's own responsibility replaces the inherited one for that role, so an inherited and an own responsibility for the same role is not an overlap.
- **Proposed items** are listed as "not yet accepted", not as gaps. Excluded and retired items are ignored.

**D-19 policy (recommended):**

- **Split.** The parent becomes a group (`is_group=true`), as "allowance subdivision does" (L146).
  - Children default to the parent's action, system and part, and are editable.
  - Responsibilities stay on the parent and are inherited (D-09).
  - Cost lines stay on the parent until the user moves them. A group's lines still roll up once, by the parent's system and part.
  - Proposal keys exclude work-item IDs, so decisions survive a split unless the semantic inputs change.
  - Draft reports flag protected edits that cite the parent. Issued reports are untouched.
- **Retire.** Retiring is refused with 409, listing blockers, while live posting cost values, live responsibilities or open delivery items reference the item. Once the user resolves them, retiring sets `retired_at`. Decisions and history remain, and drafts become stale.
- **Change of action, system or part.** The version increments.
  - Responsibilities stay, and the gap check re-evaluates.
  - Cost lines move with the item, so by-system and by-part totals stay exact.
  - Proposals recompute. Any decision whose fingerprint changed reopens.
  - Protected edits sourced from the item are flagged as conflicts.

**D-20 policy (recommended).** Issuing stores, inside one transaction:

- **The snapshot.** A canonical JSON snapshot holds:
  - every rendered section and value;
  - every citation's full `SourceRef` (file SHA-256, revision, page, location, offsets, excerpt);
  - calculation methods and inputs, assumptions and their rationale;
  - work-item, package-scope and cost-item IDs and versions, and the cost plan version id;
  - knowledge, template, clause, thresholds and question versions, and the app commit.
- **Its hash**, `snapshot_sha256`.
- **The export bytes** as a content-addressed file in the existing files store.

Reproduction reads the snapshot, never live tables. Reprocessing or a knowledge change therefore cannot alter an issue. Re-rendering from the snapshot with the same renderer version must give the same text.

## 6. Planning gaps resolved

These are the ten questions in the brief. The source statements and the affected requirements are in the register, under the requirement IDs given here.

| # | Question | Resolution | Decision | Requirements |
| - | - | - | - | - |
| 1 | Gap check in Stage 2, packages in Stage 3 | Move the gap check to Stage 3; Stage 2 proposals of package kinds are dismiss-only | D-03 | NW-REQ-259, 260, 162, 315, 317 |
| 2 | Responsibility for investigate, retain, remove, groups, supply-only; define "supply and install" | Rule set | D-09 | NW-REQ-246, 259, 260, 119, 120, 377 |
| 3 | Mixed interventions with a singular work type | Part-level work type override plus a `work_type` predicate | D-07 | NW-REQ-080, 108, 376 |
| 4 | Migrating every site fact, document fact, user value, planning value, key and projection | Registry-based classification and inventory | D-04, §4.12 | NW-REQ-100, 101, 104, 232-242, 370 |
| 5 | Building age missing (L110) vs the existing determinant (L218) | Determinant present since `6798122`; not read at runtime (F06). WP-15 adds a `profile_group` (site, classification) so it is harvested, with owner review. | none | NW-REQ-082, 378 |
| 6 | Jev signals alongside a code-only rebuild | Signals are background evidence questions; the rebuild reads stored answers | D-12 | NW-REQ-136, 168 |
| 7 | Forecast variance without a forecast metric | Computed forecast | D-13 | NW-REQ-224, 254 |
| 8 | Dates, risks and approvals before Stage 6 | Minimal delivery records in Stage 3 | D-03 | NW-REQ-248-250, 283, 320 |
| 9 | Effects of split, retire and change | Policy | D-19 | NW-REQ-110, 215, 277, 278, 280 |
| 10 | Reproducing an issue | Snapshot plus blob | D-20 | NW-REQ-279, 280, 184 |

**Additional tensions found during planning:**

- **F01, F02:** the regression gate and the evidence workload (D-17).
- **F14:** the `profile_rebuild` budget is absent (D-18).
- **F09, F10:** the same-project constraint and the version check are absent. Fixed in WP-11 and WP-12; these are defects within those packages' scope, not widening.
- **F12:** the `work_type` predicate (D-07).
- **D-05:** the duplicated existing-condition field.
- **D-24:** the uc attachment targets.

## 7. Dependency order

```
G0 owner approval (D-01) ──► WP-00 gate restore (D-17)
                                │
        ┌───────────────────────┼───────────────────────────┐
        ▼                       ▼                           ▼
   WP-11 sites ─► WP-12 values ─► WP-13 planning     WP-K0 closure ─► WP-K1, K2, K3 (parallel by cluster)
        │              │                                  WP-K4 merge pass + review (parallel)
        └─► WP-14 revisions/fingerprints ◄─ WP-25 loader (parallel after WP-K0)
                       │
                       ▼
   WP-15 site routing and eligibility ─► WP-20 work items ─► WP-21 existing presence and part types
                                             │                 WP-22 action Q ─► WP-23 location Q
                                             │                 WP-24 edit/split/retire
                                             ▼
                                   WP-26 proposals (needs WP-25) ─► WP-27 signals ─► WP-K6 walk-through
                                             │                     WP-28 eval keys/bench (from WP-20 on)
                                             ▼
   Stage 3: WP-30 packages ─► WP-31 responsibilities ─► WP-32 gap check
            WP-33 cost schema (after WP-31: FKs to package_stages and package_scope_items) ─► WP-34 cost ops
            WP-35 delivery minimal (parallel to WP-30)
                                             ▼
   WP-40 clause catalogue (knowledge, after WP-K0, before WP-31)
   Stage 4: WP-41 assembler ─► WP-42 citations ─► WP-43 issue ─► WP-44 export (D-15/16) ─► WP-45 RFP 0991
   Stage 5: WP-50 RFT ─► WP-51 PMP     Stage 6: WP-60 live reporting     Stage 7: WP-70 tabs/optimistic ─► WP-71 VPS verification
   WP-X1 access/recovery sweep: runs at the end of Stages 3, 5 and 7.
```

There are no cycles. WP-25 (loader and evaluator) depends only on WP-K0. WP-26 needs WP-14's knowledge version and WP-25. WP-12 and WP-13 need WP-K0's `key_scope` and `planning_keys` shapes. WP-33 follows WP-31, because migration 019 references 017 and 018. WP-40 (clause catalogue) precedes WP-31, which validates clause references.

## 8. Verification strategy

### 8.1 States

Each requirement and package moves through **Not started → In progress → Implemented → Verified**.

- **Implemented** means the code is merged with its tests written.
- **Verified** needs recorded evidence: the commands run, their outputs and the commit, in the handoff (work-package document §4). A written test is not verification.

### 8.2 Test layers

| Layer | Runs with | Use for |
| - | - | - |
| Pure Go unit | `go test ./internal/<pkg>` | reconciliation, predicates, proposal ranking and fingerprints, gap rules, money and rounding, page-fit model; table tests over behaviour, not constants |
| DB integration | `SITEWISE_TEST_DATABASE_URL=postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable go test ./internal/store/...` | composite FK rejection (wrong org, project, site), version conflicts, idempotent accept, triggers on frozen and issued rows, migrations with backfill assertions |
| API | `go test ./internal/httpapi/...` | 404 and no existence leak, 409, 422, events in the same transaction, latency tests via `internal/latency` |
| UI (Playwright) | `npm --prefix web run test:e2e` | scope picker unchanged; later, optimistic rollback |
| Knowledge | `python tools/check_knowledge.py --strict`; `python tools/check_triage.py` | every K package |
| Profile accuracy | `go run ./cmd/profile-eval` (source suite) and `go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json`; answer-key scoring (P16 harness) | every Lane A package touching reading or reconciliation |
| Intake | `go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay` | filing unchanged |
| Speed | `cmd/intake-bench` (local replay is a timing model; release uses `-live` on the VPS); `bench/budgets.json` | every new path |
| Rendered export | PDF page count, minimum font and text extraction against fixtures | WP-44 onward |
| Full gate | `pwsh tools/check.ps1` | before every merge to main |

### 8.3 Speed budgets

Status values:

- **E**: existing and gated.
- **P**: proposed target in the PRG; approve, then measure on the VPS.
- **R**: recommended by this plan.

None is measured on the VPS yet.

| Path (bench name) | Metric | p50 | p90 | Fixture | Environment | Status | Package |
| - | - | - | - | - | - | - | - |
| `whole_intake` (filing) | server time per filed document | 1,000 ms | 2,000 ms | intake corpus | VPS for release | E (L409 preserve) | WP-00, WP-27 (under added background load) |
| `project_profile_read` | GET profile | 50 ms | 150 ms | bench project | VPS | E (keep, D-18) | WP-12, WP-20 |
| `profile_edit` (includes rebuild) | PUT value, scope or reading | 50 ms | 150 ms | bench project | VPS | E (keep, D-18) | WP-12, WP-20, WP-26 |
| `profile_rebuild` | code-only rebuild including work items and proposals | 100 ms | 300 ms | Spec Home **and 0991** | VPS | P (L409) / not gated today (F14) | WP-14 adds the path; WP-26 must pass |
| new saved reads and writes (works, packages, costs, delivery, report edits) | request | 100 ms | 250 ms | 0991, Petersham | VPS | P | WP-20, 24, 26, 30-35 |
| `gap_check` | GET gaps | 50 ms | 150 ms | 0991, Petersham | VPS | P | WP-32 |
| `report_assemble` | deterministic assembly from current state | 300 ms | 1,000 ms | 0991 RFP | VPS | P | WP-41, 45, 50, 51 |
| local edit feedback | input to visible change (client) | under 100 ms | n/a | Playwright | target browser and VPS | P | WP-70 |
| `report_export` | render | set after D-15 | set after D-15 | 0991 RFP | VPS | P (L409 "measure separately") | WP-44 |
| evidence reading | calls, tokens and elapsed time per selected document after a question change | record, no gate | n/a | Spec Home, Hale, 0991 | VPS, live | P (L171 "record") | WP-00, 22, 23, 27 |

**Behavioural gates:**

- Normal reads and edits make **zero** Jev calls. A fake Jev client asserts this (AT-25).
- Background reading reports actual counts, never an ETA (AT-26).

**Adoption measure:** active minutes from dropping the 0991 documents to an issuable RFP draft, on the first run (D-21). Recorded, not gated; the owner sets the target afterwards.

### 8.4 Acceptance scenarios

Each scenario maps to the packages and the verification method.

| ID | Scenario (L413 unless stated) | Packages | Method |
| - | - | - | - |
| AT-01 | 0991 capex fire services: coarse items, the flow-and-pressure-test proposal (L197), an issuable RFP draft, adoption minutes | WP-20-28, 30-35, 40-45 | eval keys (reviewed: false until the owner reviews) + manual owner run |
| AT-02 | 0777 city tower fit-out: work items, packages, and base-building capacity proposals | WP-28 (keys), WP-26, WP-K1, WP-K6 (walk-through) | eval keys + walk-through |
| AT-03 | Mornington new house: existing answer keys unchanged (tenders differ) | all Lane A | answer-key harness + replay |
| AT-04 | Petersham multi-part new mixed use: keys unchanged, parts kept | all Lane A | answer-key harness |
| AT-05 | Hale extension plus refurbishment: replay 12/12, keys unchanged; per-part work types (D-07) | WP-00, 15, 21 | replay + keys |
| AT-06 | Newham Kit and Rutherford keys unchanged (manifest regression; not in L413) | all Lane A | answer-key harness |
| AT-07 | Sparse evidence: unknowns explicit; nothing defaulted favourable | WP-13, 20 | unit + API |
| AT-08 | Conflicting, superseded and skipped sources | WP-12, 15, 20 | unit + DB |
| AT-09 | Assumption accepted then challenged; not eligible for derivation | WP-13, 15 | unit + API |
| AT-10 | Mixed physical parts; facts and items on parts; totals by part | WP-11, 23, 34 | DB + unit |
| AT-11 | Retained live systems → obligation in the RFT | WP-31, 32, 50 | API + report fixture |
| AT-12 | Splitting a coarse work item | WP-24, 34 | DB + API |
| AT-13 | A dismissed proposal stays dismissed across rebuild, reprocessing and restart; it reopens only on an input change | WP-26 | DB + API |
| AT-14 | A work item with no installer → gap; with two → overlap | WP-32 | unit + API |
| AT-15 | Tenant fit-out raises base-building capacity. **Cannot pass on current knowledge** (D-29). | WP-K1 (edge), WP-26 (fixture), WP-K6 | fixture + walk-through |
| AT-16 | Duplicate scope: re-add, concurrent picker writes, double accept | WP-20, 26 | DB concurrency |
| AT-17 | Allowance split and double-count prevention | WP-34 | unit + DB |
| AT-18 | Tax basis, rounding and null amounts | WP-33, 34 | unit + DB |
| AT-19 | Concurrent edits: stale version → 409; rebuild and edit serialise | WP-12, 20, 24, 30-35, 41 | DB + API |
| AT-20 | Jev failure (timeout, 402): pending or failed shown; reads, edits and assembly unaffected; draft offered from the last completed state | WP-14, 41 | API with a fake Jev |
| AT-21 | Restart recovery mid-reading, mid-assembly and mid-issue: no partial state | WP-14, 41, 43 | DB + process test |
| AT-22 | Wrong org, wrong project and wrong site, on every new route and every composite FK | each WP + WP-X1 | DB + API |
| AT-23 | Issued snapshot preserved after reprocessing and a knowledge change; reproduction identical | WP-43 | DB + export |
| AT-24 | Two-page export at the agreed minimum font; overflow → prompt or identified attachment; cost plan exempt | WP-44 | rendered-export check |
| AT-25 | Normal reads and edits make no Jev call | WP-14 onward | fake Jev counter |
| AT-26 | Background progress is real, with no fabricated ETA | WP-14 | API |
| AT-27 | All speed budgets in §8.3 | each WP, WP-71 | bench |

AT-03 to AT-06 (answer keys and replays) and AT-27 are standing gates for **every** Lane A package. Each package handoff reports them (work-package document §4.2).
| AT-28 | Small projects feel small (L29): 0991 shows a handful of items and packages; nothing required that is not needed | WP-20, 30, 45 | owner observation + count fixture |
| AT-29 | Existing-system presence does not imply scope (L94, L152) | WP-21 | unit + keys |
| AT-30 | New rooftop plant on an existing building → structural assessment proposal (L197) | WP-26, K1 | fixture |

Private-corpus rule (L413): 0991 and 0777 are added to `data/eval/profile/manifest.json` "by ID only". Whether that means ID alone or the existing ID, path and hash shape is an interpretation pending D-23. Documents and personal details never enter the repo. Keys carry `reviewed: false` until the owner reviews them. L401 says "owner-drafted", so who drafts them is pending D-23.

### 8.5 Handoff and stop rules

These are defined once, in the work-package document §4. An agent that meets a material ambiguity stops only the affected work and records it as `Q-<WP>-n` in its handoff. It does not rewrite a requirement.

## 9. Audit record

Reviewer: an independent agent in a separate context. It worked read-only and audited against the PRG on 5 October 2026. Its findings and their resolution:

| Severity | Finding | Resolution |
| - | - | - |
| High | AT-02 and AT-15 had no package. AT-15 cannot pass on the current knowledge: the fit-out edge is type `loads`, refurb defaults to `alter`, and the label reads "structure or ground". | Added D-29. AT-15 is assigned to WP-K1, WP-26 and WP-K6; AT-02 to WP-28 and WP-K6. NW-REQ-158 and 344 are Blocked(D-29). |
| High | Packages depended on later work: migration 019 needs 018; WP-31 needs the clause catalogue; WP-25 and WP-14 were inconsistent; WP-12 and WP-13 need WP-K0's shapes. | Reordered: WP-31 → WP-33; WP-40 moved to W1; §7 corrected; WP-K0 is a prerequisite of WP-12 and WP-13 and owns the checker rules. |
| High | D-06 answered a peer-review question by default. | D-06 is now Open and blocking WP-15, with the band outcome stated. |
| Medium | The K0 approval was treated as done; L401 and L413 interpretations were stated as settled; the schema conflicted with D-05; D-17 (b) was not weighed against `AGENTS.md` rule 3; the rebuild and edit budgets conflicted; `ic.controls-test-link` conflicts with the `investigate` boundary; `restore-check` was misdescribed. | NW-REQ-304 is PART; D-23 is widened; WP-20 is Blocked(D-05); D-17 now gives calls per passage and the API limits to check; D-18 states the choice, with WP-26 step 0; D-30 added; the migration dry-run procedure is rewritten. |
| Low | F12, F18 and D-16 citations were wrong; the status board omitted prerequisites; NW-REQ-059's migration list was incomplete; some INT additions were unlabelled; uc `effect` and "before tender" had no check; NW-REQ-169 was overstated. | All corrected. §4.0 now justifies each extra table, and the Proposals UI was removed ahead of WP-70. |

**Not re-verified by the reviewer:** F01. Its sandbox blocked `go run`. F01 was executed during planning (§2.1).

**After the fixes:** the scripted traceability check (package lists ⇄ register WP column) reports no mismatch, and the PRG SHA-256 is unchanged.
