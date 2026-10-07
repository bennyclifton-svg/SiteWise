# Next wave: requirements register

> **Current progress:** [NEXT-WAVE-STATUS.md](NEXT-WAVE-STATUS.md). On 7 October the owner authorized completing M2/M3 unattended; earlier sequencing/permission holds are historical. Acceptance evidence remains explicit.

> 7 October owner update: publish the current implementation checkpoint to main/GitHub; move broader acceptance testing beyond the thin 0991 project. See implementation plan "Current owner direction". Outstanding gates remain open; this is not acceptance or deployment approval.

Status: historical planning baseline from 5 October 2026, with dated execution updates below. The owner authorized full M2/M3 implementation on 7 October. Original rows preserve baseline traceability; section R overrides affected requirements. Read NEXT-WAVE-STATUS.md for progress and implementation plan §5 for decisions; historical dispositions are not a live task list.

| Item | Value |
| - | - |
| Source (PRG) | `docs/plans/2026-10-04-next-wave-architecture-schema.md`, SHA-256 `6c1a03a7fd6a8b300079147fbd7b9e5fc864230bb1a755d929780d67e9a72cab`, 435 lines |
| Locator | `L###` is a line of the original 435-line baseline at that hash, excluding the appended review amendment. Tables/code cite those baseline lines. Section R cites the amendment and plan sections directly. |
| HEAD at planning | `f0c170a` (see the implementation plan §0 for drift from `02094db`) |
| Decisions, findings and schema | `docs/plans/2026-10-04-next-wave-implementation-plan.md` (`D-##`, `F##`, `AT-##`, §4) |
| Packages | `docs/plans/2026-10-04-next-wave-agent-work-packages.md` (`WP-##`) |

## How to read this register

**Type** says what kind of statement it is:

| Code | Meaning |
| - | - |
| SR | Stated requirement |
| CN | Constraint |
| PD | Proposed design awaiting approval |
| DEC | Decision captured in the PRG |
| DX | Draft or illustrative example; never approved knowledge |
| TGT | Proposed target |
| EX | Exclusion or later wave |
| CTX | Context; maps to other requirements |
| INT | Planner's interpretation or recommended addition; needs owner approval |

**Current** in the original tables means the historical planning baseline at HEAD `f0c170a`, not current HEAD:

| Code | Meaning |
| - | - |
| SAT | Already satisfied |
| PART | Partially satisfied |
| MISS | Missing |
| CONF | In conflict with current behaviour |
| N/A | Not assessable with available evidence |

Evidence refers to findings `F##` in the plan §2, or to files. The basis is static inspection unless marked (X), which means executed.

**Disposition**:

| Code | Meaning |
| - | - |
| Planned | Has a package |
| Blocked(D-xx) | Planned, but waiting for a decision |
| Keep | Already satisfied; packages must not regress it, and acceptance re-checks it |
| Mapped→ | Context covered by the requirements named |
| Excluded | Recorded exclusion; guarded by a check |

**State** follows Not started → In progress → Implemented → Verified. Every requirement is **Not started** unless the State column says otherwise. "Implemented" items are knowledge files already merged; none is **Verified**, because owner review has not been evidenced.

## A. Purpose and boundaries (L1-L5)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-001 | No implementation starts until the owner approves the direction (document is a draft for owner and peer review) | L1 "No implementation is authorised by this document" | CN | N/A | G0 | D-01 recorded in the handoff log before the first WP merges | D-01 | Planned |
| NW-REQ-002 | One shared project foundation supports a live PMP, consultant RFPs, trade RFTs and a progressively detailed cost plan | L3 "one shared project foundation supporting a live PMP…" | SR | MISS (F19) | WP-33, 34, 41, 45, 50, 51 | AT-01: all four outputs on 0991 read the same work-item, package and cost IDs | D-03 | Planned |
| NW-REQ-003 | Retain Go, PostgreSQL, Jev, local document storage, existing jobs and SSE | L3 "Retain Go, PostgreSQL, Jev, local document storage, existing jobs and SSE" | CN | SAT (`go.mod`, `internal/jobs`, `internal/events`) | all | Every handoff lists its `go.mod` and `web/package.json` diff; no new service unit in `deploy/` | none | Keep |
| NW-REQ-004 | No LLM, agent harness, vector database or additional service in this wave | L3 "No LLM, agent harness, vector database or additional service" | CN / EX | SAT | all | Dependency diff review; AT-25 (Jev only via jobs) | none | Keep |
| NW-REQ-005 | One model serves the full spectrum through design and delivery: single-system capex in occupied buildings (fire pumps, chillers, roof membranes, facades), fit-outs, remediation, extensions, and new houses, warehouses, commercial buildings and residential towers | L5 "One model must serve the full spectrum…" | SR | PART: the profile serves new-build scope; refurb scope starts empty (F11) | WP-20, 21, 26, 28 | AT-01 to AT-06 pass with one schema; no kind-specific tables | D-07 | Planned |

## B. Decisions captured (L7-L31)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-006 | The profiler establishes the reviewable project definition: scope, quality, time, cost, constraints, evidence and unresolved decisions | L9 "scope, quality, time, cost, constraints, evidence and unresolved decisions" | DEC | PART: scope, constraints (conditions) and evidence exist; quality, time, cost and unresolved decisions do not | WP-13, 20, 33, 35, 60 | The profile read model exposes each of the seven areas from its owning record (work-item targets, delivery dates, cost summary, `value_state=unknown` list) | D-03 | Planned |
| NW-REQ-007 | The profiler supports incomplete projects | L9 "It supports incomplete projects" | DEC | PART (blank bands) | WP-13 | AT-07 | none | Planned |
| NW-REQ-008 | The profiler offers useful starting assumptions | L9 "offers useful starting assumptions" | DEC | MISS | WP-13, 34 | Starting values are proposed only from approved libraries; with no approved library, nothing is proposed and the input is asked (see NW-REQ-185) | approved libraries (none exist) | Blocked (libraries) |
| NW-REQ-009 | Jev selects among bounded candidates | L11 "Jev selects among bounded candidates" | CN | SAT (`internal/profile/questions.go`: choice and noul with enumerated criteria) | WP-22, 23, 27 | Every new question is choice or noul with closed criteria; unit test lists criteria keys | none | Keep |
| NW-REQ-010 | Code parses, calculates, applies rules and assembles approved wording | L11 "Code parses, calculates, applies rules and assembles approved wording" | CN | PART | WP-26, 34, 41 | No generated prose in reports; every sentence comes from a template, an approved clause or a user edit (WP-41 test) | none | Planned |
| NW-REQ-011 | Users can edit content directly | L11 "Users can edit content directly" | DEC | PART (profile values only) | WP-24, 31, 34, 35, 41 | Each new record type has PATCH; report edits are protected (NW-REQ-277) | none | Planned |
| NW-REQ-012 | PMP combines delivery plan and progress report | L13 "PMP combines delivery plan and progress report" | DEC | MISS | WP-51, 60 | PMP fixture has both plan and progress sections | WP-35 | Planned |
| NW-REQ-013 | RFP covers a consultant appointment | L13 "RFP covers a consultant appointment" | DEC | MISS | WP-45 | RFP requires a services package | WP-30 | Planned |
| NW-REQ-014 | RFT covers a construction package | L13 "RFT covers a construction package" | DEC | MISS | WP-50 | RFT requires a works package | WP-30 | Planned |
| NW-REQ-015 | All outputs consume shared state independently | L13 "All consume shared state independently" | DEC | MISS | WP-41 | The assembler reads domain stores only; one report never reads another (static check plus test) | none | Planned |
| NW-REQ-016 | Generating a PMP is not a prerequisite for an RFP or RFT | L13 "Generating a PMP is not a prerequisite" | DEC | MISS | WP-41, 45 | Issue an RFP on a project with no PMP | none | Planned |
| NW-REQ-017 | Reports target roughly 1½ pages | L15 "target roughly 1½ pages" | TGT | MISS | WP-44 | Page-fit report states the target and the actual page count; it does not fail on overshoot below 2 pages | D-16 | Blocked(D-16) |
| NW-REQ-018 | Reports must fit within two pages at a readable size | L15 "must fit within two pages at a readable size" | SR | MISS | WP-44 | AT-24 | D-15, D-16 | Blocked(D-15, D-16) |
| NW-REQ-019 | Detailed pricing schedules and technical references may be separate, clearly identified attachments | L15 "may be separate, clearly identified attachments" | SR | MISS | WP-43, 44 | Each attachment is listed by title and ID in the report body | none | Planned |
| NW-REQ-020 | The cost plan is not limited to two pages | L15 "The cost plan itself is not limited to two pages" | SR | MISS | WP-44 | Page-limit test is skipped for the cost-plan export and asserted skipped | none | Planned |
| NW-REQ-021 | Readable evidence means eligible for consideration, not automatically correct | L17 "eligible for consideration, not automatically correct" | CN | SAT (thresholds, green withheld: F17) | WP-12, 15 | Bands unchanged on the answer-key harness | none | Keep |
| NW-REQ-022 | Preserve reading choices, revisions, conflicts, physical scope and user overrides | L17 "Preserve reading choices, revisions, conflicts, physical scope and user overrides" | CN | SAT (`documents.profile_read`, `supersessions`, red band, user band) | all Lane A | Existing store and httpapi tests plus AT-08 | WP-00 | Keep |
| NW-REQ-023 | Detail the backend first | L19 "Detail the backend first" | DEC | N/A | sequence (§7) | Stage 7 UI work starts after Stages 1-5 are verified | none | Planned |
| NW-REQ-024 | Summary and Systems tabs are a later presentation of the same records, not separate stores | L19 "not separate stores" | CN | N/A | WP-70 | Tabs read existing endpoints only; no new table | none | Planned |
| NW-REQ-025 | The site is the lasting record of what exists; a project is an intervention on a site | L21 "The site, not the project, is the lasting record" | DEC | MISS (F19) | WP-11 | `projects.site_id` NOT NULL; site rows survive project deletion paths | none | Planned |
| NW-REQ-026 | Version 1 has one project per site | L21 "Version 1 has one project per site" | DEC | MISS | WP-11 | Unique index `projects_one_per_site_v1` | none | Planned |
| NW-REQ-027 | Building facts are keyed to the site, so a later project on the same building starts from what is known | L21 "building facts are keyed to the site" | DEC | MISS | WP-12, 15 | DB test: site-scope values have `project_id` NULL and are read by any project on that site | D-04 | Blocked(D-04) |
| NW-REQ-028 | Scope is a set of work items: an action on a system at a part | L23 "Scope is a set of work items" | DEC | MISS (F11) | WP-20 | `work_items` holds all scope; no `scope.*` user values remain after migration 015 | none | Planned |
| NW-REQ-029 | The scope picker keeps working; its choices become coarse work items | L23 "The scope picker keeps working" | DEC | MISS | WP-20 | Playwright scope test unchanged; DB shows the matching work items | none | Planned |
| NW-REQ-030 | The building model says what works trigger: physical consequences through the interface graph, regulatory consequences through rules, and common unforeseen conditions | L25 "physical consequences through the interface graph, regulatory consequences through rules, and the unforeseen conditions" | DEC | PART: drafts exist (F05) but are not loaded at runtime (F04) | WP-25, 26 | Proposals of all three record kinds appear on fixtures | WP-K0 | Planned |
| NW-REQ-031 | Code evaluates the works triggers | L25 "Code evaluates them" | DEC | MISS | WP-26 | AT-25: rebuild makes zero Jev calls | none | Planned |
| NW-REQ-032 | Triggers appear as proposals with their reason; the user accepts or dismisses | L25 "appear as proposals with their reason" | DEC | MISS | WP-26 | API returns `reason`; accept and dismiss endpoints exist | none | Planned |
| NW-REQ-033 | Primary user is the owner-side project manager who appoints consultants and contractors (assumed; owner to confirm) | L27 "assumed; owner to confirm" | DEC (pending) | N/A | G0 | D-02 recorded | D-02 | Blocked(D-02) |
| NW-REQ-034 | Builder-side subcontract procurement comes later and reuses the same records | L27 "Builder-side subcontract procurement comes later" | EX + CN | N/A | WP-30, 31 | No builder-side fields; package, scope and cost IDs are stable (NW-REQ-267) | none | Excluded (guarded) |
| NW-REQ-035 | Small projects feel small: a capex job shows a handful of work items and packages | L29 "Small projects must feel small" | SR | N/A | WP-20, 30, 45 | AT-28 | none | Planned |
| NW-REQ-036 | Nothing is required that the job does not need | L29 "nothing is required that the job does not need" | SR | N/A | WP-X1 (sweep checklist; every WP) | API accepts minimal bodies; a 0991 RFP is issuable with only the inputs its template marks essential | none | Planned |
| NW-REQ-037 | First report: a consultant RFP on a real capex project | L31 "First report: a consultant RFP on a real capex project" | DEC | MISS | WP-45 | 0991 RFP issued (AT-01) | Stage 3 | Planned |
| NW-REQ-038 | Then the works RFT, then the PMP | L31 "then the works RFT, then the PMP" | DEC | MISS | WP-50, 51 | Merge order WP-45 → WP-50 → WP-51 | none | Planned |
| NW-REQ-039 | All reports come from one assembler | L31 "all from one assembler" | DEC | MISS | WP-41 | One `internal/reports.Assemble` entry; kinds differ only in template | none | Planned |

## C. Core concepts (L33-L77)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-040 | Adopt the ontology discipline, not the technology: no OWL, RDF store, graph database or reasoner | L35 "Adopt the discipline, not the technology" | CN / EX | SAT | all | Dependency review | none | Keep |
| NW-REQ-041 | Postgres rows, YAML catalogues and code are sufficient and keep the speed budgets | L35 "Postgres rows, YAML catalogues and code are sufficient" | CN | SAT | all | §8.3 budgets | none | Keep |
| NW-REQ-042 | Concept **Site**: one address or campus and everything built on it; persists across projects; held in `sites` | L39 | PD | MISS | WP-11 | Schema test | none | Planned |
| NW-REQ-043 | Concept **Part**: building, storey, tenancy, roof, plant area, compartment, outbuilding or zone of a site; NCC class attaches here; parts move from project to site | L40 | PD | PART (`project_parts.ncc_class`; no `roof` or `plant_area` kind; project-owned) | WP-11 | Kinds CHECK; a "zone" uses kind `part` (INT: L40 lists zone, but L134's kind list does not) | none | Planned |
| NW-REQ-044 | Concept **System** stays in `knowledge/` systems | L41 | SR | SAT (151 systems) | none | none | none | Keep |
| NW-REQ-045 | Concept **Determinant**: a fact about a site or part that decides which rules apply; `knowledge/` determinants and the profile | L42 | SR | SAT | WP-12 (site scope) | Determinants are site-scoped per the registry | D-04 | Keep |
| NW-REQ-046 | Concept **Project**: a bounded intervention on one site, with scope, quality, time and cost objectives | L43 | PD | PART (projects exist; no site; objectives not held) | WP-11, 20, 33, 35 | Scope in work items, quality in targets, time in milestones, cost in `funding_target` | none | Planned |
| NW-REQ-047 | Concept **Action** in `knowledge/works/actions.yaml` (new) | L44 | PD | SAT in knowledge (F05) | WP-K0 | Owner approves the file | D-01 | Implemented (knowledge, draft) |
| NW-REQ-048 | Concept **Work item**: an action on a system at a part, with existing condition and target; coarse first, split when needed; `work_items` | L45 | PD | MISS | WP-20, 24 | Schema test | none | Planned |
| NW-REQ-049 | Concept **Package**: a contract, services (consultant), works (head contract or trade) or supply; `packages` | L46 | PD | MISS | WP-30 | Schema test | none | Planned |
| NW-REQ-050 | Concept **Responsibility**: a package's role for a work item (design, supply, install, test, certify and so on), or a general obligation; `package_scope_items` | L47 | PD | MISS | WP-31 | Schema test | none | Planned |
| NW-REQ-051 | Concept **Stage**: investigation, design stages, approvals, procurement, construction, completion, defects; a catalogue plus delivery items | L48 | PD | PART (`SCHEMA.md` uc stage targets) | WP-K0, 30, 35 | `knowledge/works/stages.yaml` validated by the checker | none | Planned |
| NW-REQ-052 | Concept **Cost line**: money on a work item, a package stage or a project-wide category; `cost_items`, `cost_values` | L49 | PD | MISS | WP-33 | Schema test | none | Planned |
| NW-REQ-053 | Concept **Delivery item**: milestone, activity, approval, risk, issue, decision or action; `project_delivery_items` | L50 | PD | MISS | WP-35 | Schema test | D-03 | Planned |
| NW-REQ-054 | Concept **Rule** stays in `knowledge/` rules | L51 | SR | SAT (227) | none | none | none | Keep |
| NW-REQ-055 | Concept **Interface** stays in `knowledge/` interfaces | L52 | SR | SAT (144) | none | none | none | Keep |
| NW-REQ-056 | Concept **Consequence**: what a kind of work on a system requires, by rule or interface; `knowledge/` (new) | L53 | PD | PART (shape plus 1 record; not loaded: F04) | WP-K0, K2, K3, WP-25 | Loader test | none | Planned |
| NW-REQ-057 | Concept **Unforeseen condition**: a state commonly discovered during works that a kind of work makes likely; `knowledge/` (new) | L54 | PD | PART (560 drafts; not loaded) | WP-K4, WP-25 | Loader test | none | Planned |
| NW-REQ-058 | Concept **Evidence**: a source record with document revision, hash and location; `document_sources`, `passage_sources` | L55 | PD | PART (passage IDs not stable: F13; `sources` JSON lacks hash and revision) | WP-12 | `SourceRef` carries `file_sha256`, revision, page, location and offsets | none | Planned |
| NW-REQ-059 | Each table or knowledge file represents one concept in the table; a new concept is added to the table before any schema | L59 "A new concept is added here before any schema" | CN | N/A | G0 | The owner amends the PRG concept table (or approves the NW-REQ-380 additions) before migrations 012-017, 019, 021 and 022 | D-01 | Blocked(D-01) |
| NW-REQ-060 | Relations are explicit IDs or foreign keys, never inferred from text at read time | L61 | CN | PART (fact-to-part by label at rebuild, F07) | WP-X1 (sweep checklist; every WP) | Review checklist; label mapping happens only at rebuild (L175 keeps exact-label matching) | none | Planned |
| NW-REQ-061 | Every vocabulary Jev reads has literal `describes` and `excludes` text with boundary cases | L63 | CN | PART (actions yes; location options have none) | WP-22, 23, 27 | Each new option has describes and boundary text; location criteria come from part label and kind (D-26) | D-26 | Planned |
| NW-REQ-062 | What is generally true lives in `knowledge/`; what is true of one project lives in Postgres | L65 | CN | SAT | all | Review | none | Keep |
| NW-REQ-063 | Concepts align with ISO 12006-2; Uniclass, OmniClass, NATSPEC and AIQS codes may later be mapping fields, not structure | L67, L69-L77 (mapping table: complex/entity → site/part; built space → part; element → system; work result → work item; process → stage/delivery item; agent → discipline/role; property → determinant/target) | CN (codes later: EX) | SAT | none | No classification code is a key or FK | none | Keep / Excluded (codes) |

## D. Existing foundation and gaps (L79-L114)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-064 | `knowledge/SCHEMA.md` remains the physical building model; discipline and trade are views over it | L83 | CN | SAT | WP-30 | Disciplines are IDs referencing knowledge, not a new hierarchy | none | Keep |
| NW-REQ-065 | Reuse `project_parts`, `profile_facts`, protected `profile_user_values`, precomputed `profile_rows` and `profile_builds` | L84 | CN | SAT | WP-11-14 | Tables extended, not replaced (migration review) | none | Keep |
| NW-REQ-066 | Reuse document sources, passage locations, cached passage calls and `documents.profile_read` | L85 | CN | SAT | WP-22, 23, 27 | `cachedAsk` unchanged | none | Keep |
| NW-REQ-067 | Preserve pure reconciliation, supersession handling, user precedence and scope suggestions | L86 | CN | SAT (`internal/profile/reconcile.go`, `build.go`) | WP-15, 20, 21 | `go test ./internal/profile` stays green; `Reconcile` stays I/O-free | none | Keep |
| NW-REQ-068 | Retain transactional rebuilds and events; add revision tracking rather than replacing the pipeline | L87 | CN | PART (no revision: F15) | WP-14 | `RebuildProfile` keeps its lock and event; adds revision and fingerprint | none | Planned |
| NW-REQ-069 | Keep what the current UI exposes: category, subclass, work type, dynamic scale fields, conditions (planning pathway, procurement route, access, occupation, stakeholders, environmental sensitivity, industrial conditions), facts (consent number, date and lapse; contract form and basis; defects period; design life), scope checklist, systems (inclusion, provider, notes) and compliance determinants (classification, site, services, fire) | L88, L90 | CTX→CN | SAT | WP-12, 20 | Playwright and `TestProfileReadEditAndSuggestions` unchanged | none | Keep |
| NW-REQ-070 | Existing grades and other subtype fields feed quality and are not duplicated | L92 "should feed quality, not be duplicated" | CN | N/A | WP-20, 24 | A work-item target references existing keys by ID; no copy | none | Planned |
| NW-REQ-071 | `profile_builds` is not an immutable snapshot (context for reports) | L92 | CTX | SAT (F15) | Mapped→NW-REQ-279 | none | none | Mapped |
| NW-REQ-072 | No cost ledger, milestone register, report-version model or quality specification exists (context) | L92 | CTX | SAT (verified: F19) | Mapped→NW-REQ-251-258 | none | none | Mapped |
| NW-REQ-073 | Taxonomy "typical" descriptions are not approved cost or programme benchmarks | L94 | CN | N/A | WP-34 | Estimates read only approved benchmark records (NW-REQ-226) | none | Planned |
| NW-REQ-074 | Preserve part-level evidence when packages need finer boundaries | L94 | SR | MISS (F07) | WP-23 | Facts carry `part_label` when a location answer is applied | D-26 | Planned |
| NW-REQ-075 | A system present in an existing building does not by itself establish that the works include it | L94 | SR | CONF (F11: evidence presence adds scope for any work type) | WP-21 | AT-29 | D-27 | Planned |
| NW-REQ-076 | Gap: scope cannot say replace, upgrade, alter, repair, remove, retain or investigate | L98 | CTX | SAT (gap confirmed) | Mapped→NW-REQ-028, 122-131 | none | none | Mapped |
| NW-REQ-077 | Gap: the presence question asks about the completed project; in an existing building that is true of nearly every system | L100 | CTX | SAT (gap confirmed, `EvidenceQuestions`) | Mapped→NW-REQ-116, 117 | none | none | Mapped |
| NW-REQ-078 | Gap: parts and building facts belong to the project | L102 | CTX | SAT (gap confirmed) | Mapped→NW-REQ-025, 027 | none | none | Mapped |
| NW-REQ-079 | Gap: `source.scope` offers a fixed residential list | L104 | CTX | SAT (gap confirmed, F08) | Mapped→NW-REQ-132 | none | none | Mapped |
| NW-REQ-080 | Represent mixed interventions: work type is one value per project, though Hale is both an extension and a refurbishment | L106 | SR (by implication) | CONF (single `hdr.work_type`, F12) | WP-21 | AT-05 | D-07 | Blocked(D-07) |
| NW-REQ-081 | Gap: the interface graph has no existing side and no fit-out edges | L108 | CTX | PART (K4 added `if.tenant-fitout-base-building-hvac`) | Mapped→NW-REQ-306 | none | none | Mapped |
| NW-REQ-082 | Building age must be recorded; it decides hazardous-materials obligations. **Stale gap**: `existing_building_year` exists since `6798122` but is not read at runtime (F06) | L110 vs L218 | SR + CTX | PART (F06) | WP-15 (adds `profile_group` so it is harvested); WP-K3 (thresholds) | Fixture passage "built in 1985" yields `det.existing_building_year=1985` on the site part | label-call fingerprint change (WP-00 measurement) | Planned |
| NW-REQ-083 | Gap: the profile evaluation set holds no capex, fit-out or remediation project; the private corpus has 0991 and 0777 | L112 | CTX | SAT (F21) | Mapped→NW-REQ-334, 335 | none | none | Mapped |
| NW-REQ-084 | Carry Clerk's asset-register lesson (existing condition and proposed action) into the work item | L114 | SR | PART (conditions copied into `actions.yaml`) | WP-20 | Work item has `existing_condition` from that list | D-05 | Planned |

## E. Shared architecture (L116-L128)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-085 | Maintain three linked dimensions (physical, delivery, commercial) joined by the work item | L118 | PD | MISS | WP-20, 31, 33, 35 | FKs from scope items, cost revisions and delivery items to `work_items` | none | Planned |
| NW-REQ-086 | The physical dimension is owned by the knowledge model, site and project profile | L122 | CN | SAT | WP-11-15 | none | none | Keep |
| NW-REQ-087 | The delivery dimension is owned by small project delivery records | L123 | PD | MISS | WP-35 | Schema test | D-03 | Planned |
| NW-REQ-088 | The commercial dimension is owned by package records and the cost plan | L124 | PD | MISS | WP-30, 33 | Schema test | none | Planned |
| NW-REQ-089 | A work item names a system and part; packages design, supply and install it; stages and delivery items schedule it | L126 | PD | MISS | WP-20, 31, 35 | FK paths exist | none | Planned |
| NW-REQ-090 | A package can address several work items, and a work item can involve several packages in different roles | L126 | PD | MISS | WP-31 | DB test: many-to-many through `package_scope_items` | none | Planned |
| NW-REQ-091 | Use stable identifiers and explicit links | L126 | CN | PART | WP-X1 (sweep checklist; every WP) | IDs never reused; no text joins | none | Planned |
| NW-REQ-092 | General fees, approvals and contingency can be project-wide without inventing physical systems | L126 | SR | MISS | WP-31, 33, 35 | `fee` and `project_wide` cost lines and obligations need no work item | none | Planned |
| NW-REQ-093 | One Go application assembles each report from a consistent revision of profile, works, package, delivery and cost data | L128 | SR | MISS | WP-41 | Assembly reads under one repeatable-read transaction; `source_revisions` recorded | none | Planned |
| NW-REQ-094 | Separate internal packages for planning, procurement, costs and reports are sufficient | L128 | CN | N/A | §3.3 | Package layout review | none | Planned |
| NW-REQ-095 | No generic workflow platform; do not move domain records into one untyped entity table | L128 | EX | SAT | all | Schema review | none | Excluded (guarded) |

## F. Works model: sites, parts and work items (L130-L154)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-096 | A site is one address or campus | L134 | PD | MISS | WP-11 | Schema test | none | Planned |
| NW-REQ-097 | Parts belong to the site | L134 | PD | MISS | WP-11 | FK `(org_id,site_id)` | none | Planned |
| NW-REQ-098 | Part kinds `whole, building, part, storey, compartment, tenancy, outbuilding` gain `roof` and `plant_area` | L134 | PD | PART | WP-11 | CHECK accepts both; API 422 on unknown kinds | none | Planned |
| NW-REQ-099 | A tenancy is a part; base-building systems serving it are site systems | L134 | PD | MISS | WP-11, 20 | 0777 fixture: base-building plant items sit on a building or whole part, not the tenancy | none | Planned |
| NW-REQ-100 | Values describing what exists belong to the site part: determinants, building category and class, scale fields, existing systems and their condition | L136 | PD | MISS | WP-12, 15 | Registry classification test | D-04, D-05 | Blocked(D-04) |
| NW-REQ-101 | Values describing the works belong to the project: work type, conditions (procurement route, occupation), project facts (consent, contract) and work items | L136 | PD | MISS | WP-12 | Registry classification test | D-04 | Blocked(D-04) |
| NW-REQ-102 | Version 1 creates one site per project, including existing projects, so nothing visible changes | L138 | PD | MISS | WP-11 | Migration backfill assertions; UI and API snapshot unchanged | none | Planned |
| NW-REQ-103 | Attaching a second project to a site, and a screen for it, come later | L138 | EX | N/A | none | `projects_one_per_site_v1` present | none | Excluded |
| NW-REQ-104 | Facts from a project's documents about the building become site facts with their document provenance | L138 | PD | MISS | WP-15 | Site row `sources` holds the document `SourceRef` | D-04 | Blocked(D-04) |
| NW-REQ-105 | A work item is an action on a system at a part (example: replace · fire water (pump set) · plant room) | L142 | PD | MISS | WP-20 | Schema test | none | Planned |
| NW-REQ-106 | Work-item fields: part, system (leaf or top-level), action, title, existing condition, target, optional quantity and unit, parent, origin, review status, provenance and version | L142 | PD | MISS | WP-20 | Schema test per field | D-05 | Planned |
| NW-REQ-107 | Coarse items come from the scope picker and document evidence: one per part and in-scope system | L144 | PD | MISS | WP-20 | AT-16; unique `coarse_key` | none | Planned |
| NW-REQ-108 | The default action follows the work type: `new` and `extend` → new; `refurb` → alter; `remediation` → repair; `advisory` → investigate | L144 | PD | PART (in `actions.yaml`; not runtime) | WP-20, 21 | Table test over the five work types plus the part-level override | D-07 | Planned |
| NW-REQ-109 | User choices remain final, as scope choices are today | L144 | PD | SAT for scope today | WP-20 | Rebuild never changes a `user_touched` or accepted row (test) | none | Planned |
| NW-REQ-110 | Splitting a work item makes the parent a group, as allowance subdivision does | L146 | PD | MISS | WP-24 | AT-12 | D-19 | Planned |
| NW-REQ-111 | Small projects never need to split | L146 | PD | N/A | WP-24, 32, 34 | Gap check and totals work on unsplit items | none | Planned |
| NW-REQ-112 | Existing condition uses Clerk's conditions (serviceable, nearing end of life, end of life, beyond economical repair, failed, defective or non-compliant, unknown) plus a short note | L148 | PD | PART (`actions.yaml` `existing_conditions`) | WP-20 | CHECK uses those seven IDs; note ≤120 characters | D-05 | Planned |
| NW-REQ-113 | Target holds typed values or approved text; quality lives here (grades, ratings, performance) | L148 | PD | MISS | WP-20, 24 | 422 on an untyped target value; a `clause_refs` entry must exist in the catalogue | none | Planned |
| NW-REQ-114 | Proposed items follow the plan's review statuses | L150 | PD | MISS | WP-20 | `review_status` CHECK | none | Planned |
| NW-REQ-115 | New evidence can propose or conflict but never silently overwrite an accepted item | L150 | PD | MISS | WP-20, 22 | AT-09 variant on work items | none | Planned |
| NW-REQ-116 | A system stated present in an existing building, with no works action, is recorded on the site, not in the works | L152 | PD | CONF (F11) | WP-21 | AT-29 | D-27, D-04 | Planned |
| NW-REQ-117 | For `new` and `extend` projects today's behaviour holds: presence adds the system to the works as new | L152 | PD | SAT (behaviour) | WP-21 | `scope_test.go` cases for new and extend unchanged | none | Keep |
| NW-REQ-118 | The existing answer keys must still pass | L152 | SR | **CONF (F01, X: replays stale at HEAD)** | WP-00, all Lane A | AT-03 to AT-06; both replays green | D-17 | Blocked(D-17) |
| NW-REQ-119 | `retain` records an existing system that stays and must be protected or kept operating; it is not physical work | L154 | PD | MISS | WP-20, 32 | Gap check treats retain per D-09 | D-09 | Planned |
| NW-REQ-120 | `retain` reaches packages as an obligation (for example, keep the hydrant system live throughout) | L154 | PD | MISS | WP-31, 50 | AT-11 | D-09 | Planned |

## G. Actions and locations (L156-L175)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-121 | Jev reads actions, so each has literal boundaries; the wording is a draft for owner review | L158 | SR | PART (draft file) | WP-K0 | Owner review recorded | D-01 | Implemented (knowledge, draft) |
| NW-REQ-122 | The eight actions with exactly the tabled describes and boundary text (new, replace, upgrade, alter, repair, remove, retain, investigate) | L160-L169 (8 rows) | PD | SAT in knowledge (text matches, checked) | WP-K0 | Test asserts `actions.yaml` text equals the PRG table, or an owner-approved revision | D-01 | Implemented (knowledge, draft) |
| NW-REQ-123 | Jev question `sys.<leaf>.action` is asked in the existing evidence fan-out, alongside presence and provider, for each labelled leaf; no new round trip | L171, [fan-out](https://docs.typesafe.ai/patterns/fan-out) | PD | MISS | WP-22 | Unit test: `EvidenceCall` includes the action question in the same `jev.Call` | D-17 | Blocked(D-17) |
| NW-REQ-124 | One choice: the eight actions plus `several` and `not_stated` | L171 | PD | MISS | WP-22 | Criteria keys are exactly those 10 | none | Planned |
| NW-REQ-125 | The action question is separate and atomic from presence and provider | L171, [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one) | PD | MISS | WP-22 | Distinct question ID | none | Planned |
| NW-REQ-126 | The action question has its own threshold, because choice confidence depends on option count | L171, [confidence](https://docs.typesafe.ai/confidence) | PD | MISS (F17: no `action` shape) | WP-22 | `thresholds.json` shape `action` with `options: 10`; green withheld until owner calibration (INT, consistent with F17) | none | Planned |
| NW-REQ-127 | Instructions read "Using `text`, what does the passage say the works do to the <label>?". INT: the evidence call names the field `excerpt` (`valueQuestion` rewrites `text`→`excerpt`), so the literal wording must match the call's field; raise Q-WP-22-1 if the owner wants a literal copy | L171 | PD | MISS | WP-22 | Instruction string test | none | Planned |
| NW-REQ-128 | `several` goes to Needs mapping for the user | L171 | PD | MISS | WP-22 | Passage outcome `needs_mapping`; the action is not applied | none | Planned |
| NW-REQ-129 | `not_stated` falls back to the work-type default | L171 | PD | MISS | WP-20, 22 | Default action applied; band `suggested` | D-07 | Planned |
| NW-REQ-130 | Adding the question changes the evidence-call fingerprint, so the next update re-reads evidence for selected documents | L171 | PD | SAT (mechanism, F03) | WP-22 | Recorded in the handoff with document counts | none | Planned |
| NW-REQ-131 | Record call count, tokens and elapsed time for that re-read | L171 | PD | MISS | WP-22, WP-00 | Measurement table in `docs/evidence/` | live Jev consent | Planned |
| NW-REQ-132 | Replace the fixed `source.scope` options with options built by code from the site's parts, plus `whole_project`, `specific`, `multiple` and `not_stated` | L175 | PD | MISS (F08) | WP-23 | Option builder unit test | D-26 | Planned |
| NW-REQ-133 | Location thresholds are set per option count | L175, [confidence](https://docs.typesafe.ai/confidence) | PD | MISS | WP-23 | Threshold key `location.n<k>` | D-26 | Planned |
| NW-REQ-134 | Until calibrated, applied location answers are amber only | L175 | PD | MISS | WP-23 | Never green; test | none | Planned |
| NW-REQ-135 | Fact-to-part matching by exact label stays | L175 | PD | PART (mechanism exists; no producer, F07) | WP-23 | Facts get `part_label`; `Reconcile` places them | none | Planned |

## H. Building logic for works (L177-L262)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-136 | Interface consequences, consequences and unforeseen conditions are evaluated by code during the existing code-only rebuild; none calls Jev | L179 | PD | MISS | WP-26 | AT-25 | WP-25 | Planned |
| NW-REQ-137 | Each result is a proposal carrying its reason: the work item and the interface, rule or record that raised it | L179 | PD | MISS | WP-26 | `reason` JSON test | none | Planned |
| NW-REQ-138 | The user accepts or dismisses each proposal | L179 | PD | MISS | WP-26 | API test | D-03 | Planned |
| NW-REQ-139 | A dismissal is final until its inputs change | L179 | PD | MISS | WP-26 | AT-13 | none | Planned |
| NW-REQ-140 | A large library still gives a short list: rank by severity (life safety first) and by how specifically the work items match | L179 | PD | MISS | WP-26 | Ranking unit test | D-28 | Planned |
| NW-REQ-141 | The owner sets how many proposals show after the first run | L179 | PD | MISS | WP-26 | Config read from `data/profile/proposals.json` | D-11 | Planned (value open) |
| NW-REQ-142 | The rest are one click away | L179 | PD | MISS | WP-26 | `?show=all` returns the full list | none | Planned |
| NW-REQ-143 | One small table per interface type, owner-reviewed, in `knowledge/works/interface_consequences.yaml` | L183 | PD | PART (9 draft entries; no owner review) | WP-K0 | Owner review | D-01 | Implemented (knowledge, draft) |
| NW-REQ-144 | An interface consequence applies when the works touch one side and the other side exists on the site and is not being replaced | L183 | PD | MISS | WP-25, 26 | Predicate tests for `system_existing` | D-10 | Planned |
| NW-REQ-145 | Interface direction follows `SCHEMA.md`: `from` acts on `to` | L183 | CN | SAT (knowledge) | WP-25 | Evaluator test for both sides | none | Keep |
| NW-REQ-146 | Over-proposal is fixed in the YAML, not in code | L183 | CN | N/A | WP-26 | No record IDs hard-coded in Go (grep check) | none | Planned |
| NW-REQ-147 | `loads`, works touch `from` with new, replace or upgrade → investigate `to`: capacity of the existing structure or ground for the new load | L187 | PD | SAT (knowledge `ic.loads-investigate-supported`) | WP-26 | Row test | none | Planned |
| NW-REQ-148 | `supplies`, works touch `to` with new, upgrade or alter → investigate `from`: capacity of the existing supply (flow and pressure, electrical, cooling) | L188 | PD | SAT (knowledge) | WP-26 | Row test | none | Planned |
| NW-REQ-149 | `supplies`, works touch `from` with replace, upgrade, alter or remove → keep `to` working: changeover, temporary supply or shutdown window; retain obligation | L189 | PD | SAT (knowledge, kind obligation) | WP-26, 31 | Row test | D-03 | Planned |
| NW-REQ-150 | `penetrates`, works touch `from` with new, alter or replace → work item: make good `to` at penetrations (for example fire-stopping) | L190 | PD | SAT (knowledge, kind `work_item`) | WP-26, K0 (action of created item) | Row test | none | Planned |
| NW-REQ-151 | `controls`, either side → test the control link at commissioning | L191 | PD | CONF (knowledge kind `investigation` contradicts the `investigate` boundary at L169) | WP-26, WP-K0 | Row test after D-30 | D-30 | Blocked(D-30) |
| NW-REQ-152 | `depends_on`, works touch `to` with alter, replace or remove → investigate `from`: whether what relies on it still holds (for example FRL concessions relying on sprinklers) | L192 | PD | SAT (knowledge) | WP-26 | Row test | none | Planned |
| NW-REQ-153 | `sequences`, either side → hold point in the package doing the earlier work | L193 | PD | SAT (knowledge) | WP-26, 35 | Row test | D-03 | Planned |
| NW-REQ-154 | `shares_space`, works touch `from` with new or alter → coordinate set-out with the existing system in that space | L194 | PD | SAT (knowledge) | WP-26, 31 | Row test | none | Planned |
| NW-REQ-155 | `boundary`, either side → the boundary must be assigned to a package (gap check) | L195 | PD | SAT (knowledge) | WP-26, 32 | Row test | D-03 | Planned |
| NW-REQ-156 | Example: new rooftop plant on an existing building reaches `if.plant-loads-structure` and proposes a structural assessment | L197 | DX | PART (interface exists, `services-wet-air/interfaces.yaml:83`) | WP-26 | AT-30 | none | Planned (fixture) |
| NW-REQ-157 | Example: new sprinklers or hydrants reach the fire-water supply and propose a flow and pressure test, which 0991 budgeted for | L197 | DX | N/A (0991 claim unverified) | WP-26, K6 | AT-01 | corpus | Planned (fixture) |
| NW-REQ-158 | A tenant fit-out should reach base-building central plant and air distribution; the graph has no such edges yet (K1) | L197 | SR | PART (K4 added `if.tenant-fitout-base-building-hvac`, type `loads`; with the refurb default `alter` it raises nothing, and the ic label reads "structure or ground") | WP-K1 | AT-15 | D-29 | Blocked(D-29) |
| NW-REQ-159 | Research and draft fit-out knowledge (unverified until checked): tenant heat loads against base-building plant and air distribution; partitions and ceilings changing air paths, zoning, return air and smoke control; supplementary cooling on condenser water; BMS and fire-mode integration; glazing films and blinds (solar gain, glass thermal stress); energy-efficiency provisions for new work; base-building energy ratings and lease obligations; landlord approvals and nominated contractors | L199 | SR (research) | MISS | WP-K1 (physical), WP-K3 (energy provisions, lease, landlord) | Each topic has at least one draft record or a recorded "no source found" | none | Planned |
| NW-REQ-160 | Consequences (`cq.*`) cover what interfaces cannot express, chiefly regulatory triggers on existing buildings | L203 | PD | PART (1 record) | WP-K2, K3 | Checker counts | none | Planned |
| NW-REQ-161 | Illustrative `cq` shape (when works and determinant; propose; `governed_by`; severity; status draft; sources) | L205-L216 | DX | SAT as K0 shape (deviation: `works` moved inside `when.all`) | WP-K0 | Shape approved; record stays draft with `clause_verified: false` | D-01 | Implemented (knowledge, draft) |
| NW-REQ-162 | Proposal kinds: investigation (a work item), discipline (a package suggestion), approval or hold point (a delivery item), obligation (package scope) | L218 | PD | PART (K0 adds `work_item`) | WP-26, 30, 31, 35 | Accept path per kind | D-03 | Blocked(D-03) for non-investigation kinds |
| NW-REQ-163 | Research drafts, all `clause_verified: false` until read in the instrument: NSW alteration and fire-safety upgrade; hazardous-materials surveys of older fabric (reads `existing_building_year`); NSW Class 2 registered-practitioner and regulated-design regime (waterproofing, fire safety, structure, enclosure, services); impairment management and certification when a fire-safety system changes | L218 | SR (research) | MISS | WP-K3 | Records exist with `clause_verified: false` | none | Planned |
| NW-REQ-164 | Unforeseen conditions cover latent and concealed conditions, failed tests of existing systems, and authority and utility surprises, on new builds and existing buildings alike | L222 | SR | PART (560 drafts) | WP-K4 | Checker plus owner review | none | Implemented (knowledge, draft) |
| NW-REQ-165 | Each unforeseen condition attaches to the graph (a system, interface or action) | L222 | PD | CONF (K0 allows system, interface, stage or package kind, not action) | WP-K0 | D-24 recorded | D-24 | Blocked(D-24) |
| NW-REQ-166 | Unforeseen conditions are raised by the work items that make them likely, before tender | L222 | PD | MISS | WP-26 | AT-01; proposals depend only on work items and site facts, never on package, procurement or report state, so they exist before any RFT (test) | none | Planned |
| NW-REQ-167 | Nine categories: ground and site; existing structure; hazardous materials; concealed services and earlier work; existing systems on test; authorities and utilities; third parties and occupation; design and scope; supply and site operations (with the listed examples) | L224-L234 | PD | SAT (checker enumerates all nine) | WP-K4 | Checker | none | Implemented (knowledge, draft) |
| NW-REQ-168 | Illustrative `uc` shape: signals are Jev nouls on labelled passages, as failure-mode detectors are; `de_risk`, `contract`, `effect`, severity | L238-L258 | DX | SAT as shape; signals not run (F04) | WP-27 | Signals answered in the evidence fan-out | D-12, D-17 | Blocked(D-12, D-17) |
| NW-REQ-169 | No invented frequencies or cost percentages; a frequency is recorded only where a source states it, with `verified: false` | L260 | CN | N/A: claimed by the Pass B report, not independently audited | WP-K4 | Reviewer samples records for numbers without a stated source | none | Planned |
| NW-REQ-170 | Unforeseen conditions differ from failure modes: a failure mode is a pitfall a document can show; an unforeseen condition is discovered during the works | L260 | CN | SAT (`SCHEMA.md`) | WP-K4 | Kind field review | none | Keep |
| NW-REQ-171 | Existing `fm.*` records stay as they are | L260 | CN | **CONF (possible)**: K4 extended existing failure modes (sources appended; `fm.integrated-fire-test-failed` gained a `runs_on`, which changes detector routing) | WP-K4 | Owner reviews the extended records; each `runs_on` or wording change is listed | D-01 | Planned |
| NW-REQ-172 | Reports use unforeseen conditions: the PMP risk section, RFT latent-condition handling and provisional sums, and investigate-before-tender proposals | L260 | PD | MISS | WP-26, 50, 51 | Report fixtures show each uc's `effect` (cost, programme, safety, compliance, quality, from L254), `de_risk` and `contract` fields | none | Planned |
| NW-REQ-173 | New predicate operators `works` and `system_existing` for `applies_when`, consequences and unforeseen conditions | L262 | PD | PART (checker only, F04) | WP-25 | Go evaluator tests | D-10 | Planned |
| NW-REQ-174 | `system_present` keeps its meaning: in the completed building | L262 | CN | SAT | WP-25 | Regression test | none | Keep |

## I. Provenance and assumptions (L264-L278)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-175 | Origin: document, user, calculation or assumption | L268 | PD | MISS (bands only) | WP-12, 13 and every new table | CHECK | none | Planned |
| NW-REQ-176 | Calculations retain their input origins | L268 | PD | MISS | WP-13, 15 | `provenance.inputs[].origin` populated for derived rows | none | Planned |
| NW-REQ-177 | A knowledge-raised proposal is a calculation whose method is the record ID and knowledge version | L268 | PD | MISS | WP-26, 14 | Created records carry `method:{id, knowledge_version}` | none | Planned |
| NW-REQ-178 | Review status: proposed, accepted for planning, verified or superseded | L270 | PD | MISS | WP-12, 13 | CHECK | none | Planned |
| NW-REQ-179 | Verification requires an actor and supporting basis; user input is not automatically verified | L270 | PD | MISS | WP-12 | CHECK; user writes default to `accepted_for_planning` | none | Planned |
| NW-REQ-180 | Meaning: stated condition, requirement or target, allowance or forecast | L272 | PD | PART (assertion: stated, required, allowance) | WP-12 | CHECK; mapping test from assertion | none | Planned |
| NW-REQ-181 | Keep existing assertion semantics distinct from approval status | L272 | CN | SAT | WP-12 | `assertion` column kept | none | Keep |
| NW-REQ-182 | Unknown is an explicit unresolved value, not zero or false | L274 | PD | MISS | WP-12, 13, 33 | `value_state='unknown'`; null amounts | none | Planned |
| NW-REQ-183 | Retain source references, actor, timestamp, rationale, method or library version and dependencies as appropriate | L274 | PD | PART | WP-12, 13 | `provenance` JSON test | none | Planned |
| NW-REQ-184 | Preserve document revision or hash and source location, not merely a mutable passage ID | L274 | PD | MISS (F13) | WP-12, 43 | `SourceRef` survives `ReplaceSource` (DB test) | none | Planned |
| NW-REQ-185 | "Suggest starting values" uses approved scope, quality, cost and duration libraries | L276 | PD | MISS (only draft `scope_defaults.yaml`) | WP-13, 34 | Suggestions read only `status: reviewed` library rows | approved libraries | Blocked (libraries) |
| NW-REQ-186 | Suggestions ask only consequential missing questions | L276 | PD | MISS | WP-13 | INT: consequential = an input to an applicable rule, proposal predicate or estimate; test | none | Planned |
| NW-REQ-187 | Suggestions are individually editable proposals | L276 | PD | MISS | WP-13 | Each is its own planning value row | none | Planned |
| NW-REQ-188 | Calculated estimates may depend on assumptions and expose ranges and limitations | L276 | PD | MISS | WP-13, 34 | `range_low`, `range_high`, `limitations` populated | none | Planned |
| NW-REQ-189 | Accepted assumptions remain assumptions | L276 | PD | MISS | WP-13 | AT-09 | none | Planned |
| NW-REQ-190 | Structural adequacy, ground conditions and compliance are never defaulted as favourable; propose investigations or allowances instead | L276 | CN | N/A | WP-13, 26 | Forbidden-default key list test | none | Planned |
| NW-REQ-191 | New evidence can create a change proposal or conflict but never silently overwrites a human decision | L278 | CN | SAT for profile values (user band final) | WP-15, 20 | AT-09 | none | Keep / extend |
| NW-REQ-192 | Accepting a planning assumption must not make it eligible for verified regulatory derivations; extend user-precedence rules with explicit eligibility checks | L278 | PD | CONF (`usableFacts` admits every user value) | WP-15 | AT-09 | D-06 | Blocked(D-06) |

## J. Cost plan through procurement (L280-L318)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-193 | A stable cost item links early allowance, detailed scope and pricing schedule | L282 | PD | MISS | WP-33 | ID unchanged through subdivision and issue | none | Planned |
| NW-REQ-194 | Example: a structural consultant with four staged fee lines; a concrete trade with three works lines on work items for Building A | L284-L292 | DX | N/A | WP-34 | Used as a fixture only | none | Planned (fixture) |
| NW-REQ-195 | Stages and items are editable suggestions, not universal requirements | L294 | PD | MISS | WP-30, 33 | Delete and rename allowed; nothing mandatory | none | Planned |
| NW-REQ-196 | Scope items and cost items are distinct: several obligations can sit in one priced item, and not every obligation needs a separate fee | L294 | PD | MISS | WP-33 | `scope_cost_links` many-to-many | none | Planned |
| NW-REQ-197 | Package kinds: services (consultant), works (head contract covering the works, or trade package covering part) or supply (owner-procured plant or FF&E) | L296 | PD | MISS | WP-30 | CHECK | none | Planned |
| NW-REQ-198 | A services package can be marked for novation, which splits its stages before and after novation | L296 | PD | MISS | WP-30 | `novation_phase` test | none | Planned |
| NW-REQ-199 | Procurement route and work type suggest default packages from Clerk's consultant rosters and complexity additions, copied as data and draft | L296 | PD | MISS | WP-30, K0 | Suggestions are `proposed` and cite `package_defaults.yaml` | D-22 | Planned |
| NW-REQ-200 | Start with an overall funding target and broad allowance groups | L298 | PD | MISS | WP-33, 34 | Fixture | none | Planned |
| NW-REQ-201 | As scope develops, subdivide allowances into package and stage lines | L298 | PD | MISS | WP-34 | Fixture | none | Planned |
| NW-REQ-202 | Subdivision converts the parent to a non-posting group and allocates its amount to children in one transaction | L298 | PD | MISS | WP-34 | AT-17 | none | Planned |
| NW-REQ-203 | Subdivision leaves an explicit unallocated residual or shows the changed total | L298 | PD | MISS | WP-34 | AT-17 | none | Planned |
| NW-REQ-204 | Never add the original allowance and its replacement breakdown together | L298 | CN | MISS | WP-34 | AT-17 | none | Planned |
| NW-REQ-205 | Funding target, approved budget, current estimate, tender offer and commitment are different measures; a quote is not an award | L302 | CN | MISS | WP-33 | Distinct metrics; no offer metric | none | Planned |
| NW-REQ-206 | Sum posting leaf items only | L304 | CN | MISS | WP-34 | AT-17 | none | Planned |
| NW-REQ-207 | Keep contingency, fees, escalation and exclusions explicit | L304 | CN | MISS | WP-33 | `project_wide` categories include them; `excluded` flag | none | Planned |
| NW-REQ-208 | Unknown amounts stay null and produce an incomplete-total warning | L304 | CN | MISS | WP-34 | AT-18 | none | Planned |
| NW-REQ-209 | PostgreSQL decimal amounts, explicit currency, tax basis, quantity units, rate basis and price date | L306 | CN | MISS | WP-33 | Schema test | D-14 | Planned |
| NW-REQ-210 | Rounding is defined centrally; no binary floating point for money | L306 | CN | MISS | WP-33 | `costs.Round` only; vet check for `float64` in `internal/costs` | D-14 | Planned |
| NW-REQ-211 | Works lines name one work item and, through it, one system and part, so by-system and by-part totals reconcile exactly | L308 | CN | MISS | WP-33, 34 | Reconciliation property test | none | Planned |
| NW-REQ-212 | Fee lines name a package and stage | L308 | CN | MISS | WP-33 | CHECK | none | Planned |
| NW-REQ-213 | Project-wide lines name a category | L308 | CN | MISS | WP-33 | CHECK | none | Planned |
| NW-REQ-214 | A systems total covers works lines and shows fees and project-wide lines below it, as an elemental cost plan does | L308 | PD | MISS | WP-34 | Totals API shape | none | Planned |
| NW-REQ-215 | A works line's package is optional until packaging; once set, that package must hold a responsibility for the work item | L308 | CN | MISS | WP-34 | DB test: 422 without a responsibility | WP-31 | Planned |
| NW-REQ-216 | RFP and RFT pricing lines reference stable cost-item IDs | L310 | CN | MISS | WP-43, 45, 50 | Schedule rows carry `cost_item_id` | none | Planned |
| NW-REQ-217 | Issued schedules freeze labels, stages and IDs | L310 | CN | MISS | WP-43 | AT-23 | none | Planned |
| NW-REQ-218 | Later returned prices remain separate from internal budgets | L310 | CN | MISS | WP-43 | No write path from a schedule to `cost_values` | none | Planned |
| NW-REQ-219 | Budget disclosure is opt-in | L310 | CN | MISS | WP-43 | `budget_disclosed=false` by default; test | none | Planned |
| NW-REQ-220 | Returned tender offers need a tenderer dimension and are not stored as cost values; a later offers table keys package, tenderer and cost item | L312 | EX / CN | N/A | none | `metric` CHECK excludes offers | none | Excluded (guarded) |
| NW-REQ-221 | Claimed to date is a user-entered value with an as-of date | L314 | PD | MISS | WP-33, 34 | CHECK `as_of` NOT NULL | none | Planned |
| NW-REQ-222 | Invoices remain out of this wave | L314 | EX | N/A | none | No invoice table | none | Excluded |
| NW-REQ-223 | The profile displays cost-plan summaries and maintains no competing budget or estimate | L316 | CN | N/A | WP-34 | No cost key in profile values or the planning-key registry | none | Planned |
| NW-REQ-224 | The PMP shows material budget/forecast variance and coverage | L316 | PD | MISS | WP-51, 34 | PMP fixture | D-13 | Blocked(D-13) |
| NW-REQ-225 | Unavailable commitments or actuals remain unavailable | L316 | CN | MISS | WP-34, 51 | Rendered "not available", never 0 | none | Planned |
| NW-REQ-226 | Estimates use approved benchmark records with geography, date, quality, inclusions, exclusions and applicability | L318 | PD | MISS (no benchmarks) | WP-K0 (shape), WP-34 | Only `status: reviewed` benchmarks are used | approved benchmarks | Blocked (benchmarks) |
| NW-REQ-227 | With no suitable basis, ask for an input or keep the value unknown | L318 | CN | MISS | WP-34 | Test | none | Planned |

## K. Proposed schema (L320-L349)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-228 | Schema names are provisional for peer review, not DDL | L322 | CTX | none | Mapped→plan §4 | none | none | Mapped |
| NW-REQ-229 | Project-owned rows carry `org_id` and `project_id`; site-owned rows carry `org_id` and `site_id` | L322 | CN | PART | WP-X1 (sweep checklist; every WP) | Schema review | none | Planned |
| NW-REQ-230 | Composite FKs enforce tenant and same-project or same-site ownership; service checks alone are insufficient | L322 | CN | CONF (F09) | WP-11, 12, every schema WP, X1 | AT-22 (DB rejects a cross-project or cross-site row) | none | Planned |
| NW-REQ-231 | Mutable records have a version for concurrent-edit protection | L322 | CN | PART (F10: version not checked) | WP-12 and every schema WP | AT-19 | none | Planned |
| NW-REQ-232 | `sites`: label, address, lot, version; a project references one site; migration creates one site per existing project | L326 | PD | MISS | WP-11 | Migration test | none | Planned |
| NW-REQ-233 | `project_parts` gains `site_id` and becomes site-owned; kinds add `roof` and `plant_area` | L327 | PD | MISS | WP-11 | Migration test | none | Planned |
| NW-REQ-234 | Extend `profile_user_values` and `profile_rows` with origin, review status, meaning and structured provenance | L328 | PD | MISS | WP-12 | Schema test | none | Planned |
| NW-REQ-235 | Preserve existing keys, bands and null/reset behaviour | L328 | CN | SAT | WP-12 | Existing httpapi tests unchanged; null maps to `cleared` | none | Keep |
| NW-REQ-236 | Site-level keys (determinants, class, scale, existing systems) are stored against the site part; project-level keys against the project | L328 | PD | MISS | WP-12 | Registry plus DB test | D-04 | Blocked(D-04) |
| NW-REQ-237 | Rebuild projections rather than treating them as authoritative input | L328 | CN | SAT (`profile_rows` deleted and rebuilt) | WP-12, 15, 20 | No code reads `profile_rows` as input to a write | none | Keep |
| NW-REQ-238 | `profile_planning_values`: part, registered key, typed value and unit, origin, status, meaning, provenance and version; stores assumptions and calculated values that cannot belong in document-only `profile_facts` | L329 | PD | MISS | WP-13 | Schema test | none | Planned |
| NW-REQ-239 | No duplicate ledger totals in planning values | L329 | CN | N/A | WP-13 | Registry has no money-total keys (checker) | none | Planned |
| NW-REQ-240 | `profile_builds` gains a monotonic revision and an input fingerprint covering reading selection, evidence, user and planning values, work items and knowledge versions | L330 | PD | MISS (F15) | WP-14 | Fingerprint changes on each input class (table test) | none | Planned |
| NW-REQ-241 | `work_items`: site part, system ID, action, parent, title, existing condition, target, optional quantity and unit, origin, review status, provenance, version, stable ID | L331 | PD | MISS | WP-20 | Schema test | D-05 | Planned |
| NW-REQ-242 | `work_items` replaces scope keys as the store of scope; the scope picker writes here | L331 | PD | MISS | WP-20 | Migration 015 assertions | none | Planned |
| NW-REQ-243 | `proposal_decisions`: project, knowledge record ID (ic, cq, uc), triggering work item, accepted or dismissed, actor, inputs fingerprint | L332 | PD | MISS | WP-26 | Schema test | none | Planned |
| NW-REQ-244 | Accepting creates an ordinary record (work item, package suggestion, delivery item or obligation) that keeps the record ID | L332 | PD | MISS | WP-26, 30, 31, 35 | `source_proposal_key` and `provenance.method.id` set | D-03 | Planned |
| NW-REQ-245 | `packages`: kind services, works or supply; discipline or trade ID; novation flag; title; lifecycle status; version; scope and stage boundaries belong to the package | L333 | PD | MISS | WP-30 | Schema test | D-22 | Planned |
| NW-REQ-246 | `package_scope_items`: package, stable ID, optional work item (none for general obligations such as meetings or WHS), role (design, document, supply, install, test, certify, inspect, maintain operation, protect), approved clause ID and version or user text, stage, inclusion or exclusion, deliverable, provenance | L334 | PD | MISS | WP-31 | Schema test | D-09 | Planned |
| NW-REQ-247 | Scope items carry explicit links to interfaces and existing source records | L334 | PD | MISS | WP-31 | `interface_ids` validated against the catalogue; `source_refs` are `SourceRef` | none | Planned |
| NW-REQ-248 | `project_delivery_items`: kind (activity, milestone, action, risk, issue, decision, approval), title, owner, baseline, target, forecast and actual dates as applicable, status, as-of date, provenance; optional package and work-item links; kind-specific fields validated in Go | L335 | PD | MISS | WP-35 | Schema test plus a Go validation table | D-03 | Blocked(D-03) |
| NW-REQ-249 | `delivery_dependencies`: predecessor and successor IDs, supported dependency type and lag; reject cycles | L336 | PD | MISS | WP-35 | Cycle test | D-25 | Planned |
| NW-REQ-250 | Start with milestone dependencies, not a full scheduling engine | L336 | CN | N/A | WP-35 | No CPM computation | none | Planned |
| NW-REQ-251 | `cost_plans` and `cost_plan_versions`: revision, draft or baseline status, currency, price date, cost coverage, funding target and its provenance | L337 | PD | MISS | WP-33 | Schema test | none | Planned |
| NW-REQ-252 | Frozen baselines retain approved budgets | L337 | CN | MISS | WP-33, 34 | Trigger test | none | Planned |
| NW-REQ-253 | `cost_items` and `cost_values`: stable item identity; versioned parent, code, label, category, work item (works lines), package and stage, posting flag, quantity and unit, rate, amount or range, metric and provenance | L338 | PD | MISS | WP-33 | Schema test | none | Planned |
| NW-REQ-254 | Metrics this wave: budget, estimate, commitment, claimed to date | L338 | PD | MISS | WP-33 | CHECK | D-13 | Planned |
| NW-REQ-255 | One value per version, item and metric; group values are calculated | L338 | CN | MISS | WP-33, 34 | PK plus a store guard test | none | Planned |
| NW-REQ-256 | An explicit scope-item to cost-item join; reference links to existing physical IDs; no duplicated money on links | L339 | PD | MISS | WP-33 | Schema has no amount column | none | Planned |
| NW-REQ-257 | `reports` and `report_versions`: kind PMP, RFP or RFT; optional package; draft or issued status; reporting date; previous issue; source revisions; template version; structured sections; protected user edits; immutable issue snapshot | L340 | PD | MISS | WP-41, 43 | Schema test | none | Planned |
| NW-REQ-258 | `report_references`: version-scoped citation ID, basis, source revision and location or calculation and assumption details; linked at section, row or value level | L341 | PD | MISS | WP-42 | Schema test | none | Planned |
| NW-REQ-259 | The gap check (a read): every accepted physical work item has exactly one works package with install (or supply and install), and, where its action needs design, one services package with design | L343 | PD | MISS | WP-32 | AT-14 | D-03, D-09 | Blocked(D-03, D-09) |
| NW-REQ-260 | A missing or duplicated role shows as a gap or overlap | L343 | PD | MISS | WP-32 | AT-14 | D-09 | Blocked(D-09) |
| NW-REQ-261 | Knowledge additions: `actions.yaml`, `interface_consequences.yaml`, `consequences.yaml` and `unforeseen.yaml` per cluster, the `works` and `system_existing` operators, and checker support for all of them | L345 | PD | SAT in checker (F05, X) | WP-K0, WP-25 (runtime) | Checker 0 errors; loader tests | none | Implemented (knowledge) |
| NW-REQ-262 | `knowledge/SCHEMA.md` documents each knowledge addition before any is written | L345 | CN | SAT (works-layer section) | WP-K0 | New shapes (stages, package defaults, clauses, benchmarks, key scope, planning keys, `needs_design`) documented before their data | none | Keep / extend |
| NW-REQ-263 | Reviewed clauses and benchmarks are versioned catalogues loaded at startup, following knowledge conventions | L347 | PD | MISS (F24 deploy gap) | WP-K0, 40, 34 | Startup loads them; a version appears in the report snapshot | none | Planned |
| NW-REQ-264 | Extend validation for references, supported predicates and required provenance | L347 | PD | PART (checker) | WP-K0, 25 | Checker and loader reject unknown references | none | Planned |
| NW-REQ-265 | Only reviewed content may establish standard obligations; draft content remains visibly provisional | L347 | CN | N/A | WP-31, 41, 42 | Draft clause rendered with a "draft" mark; scope items from draft clauses are `proposed` | none | Planned |
| NW-REQ-266 | No tender comparison, contract, invoice or payment tables in this wave | L349 | EX | N/A | none | Migration review | none | Excluded (guarded) |
| NW-REQ-267 | Preserve stable package, scope and cost IDs so those features can later link without replacing the foundation | L349 | CN | N/A | WP-30, 31, 33 | IDs never reused; retire, not delete | none | Planned |

## L. Refresh and report behaviour (L351-L365)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-268 | A relevant evidence, reading-choice, override, library or project-record change marks affected projections stale | L353 | PD | MISS | WP-14, 41 | Report shows stale after each change class (table test) | none | Planned |
| NW-REQ-269 | Use explicit domain revision dependencies first; avoid a universal dependency engine | L353 | CN | N/A | WP-14 | Static dependency map per report kind | none | Planned |
| NW-REQ-270 | Reuse passage caches | L355 | CN | SAT (`passage_calls`) | WP-22, 23, 27 | none | none | Keep |
| NW-REQ-271 | Queue only necessary Jev work through existing jobs | L355 | CN | SAT (F18) | all | No new job kind without a stated reason | none | Keep |
| NW-REQ-272 | Reserve foreground filing capacity | L355 | CN | PART (priority and lease lanes; untested under the larger evidence load) | WP-00, 27 | `whole_intake` 1/2 s while background reading runs (bench scenario) | D-17 | Planned |
| NW-REQ-273 | Reconcile profile changes, apply permitted updates and surface conflicts | L357 | CN | SAT (profile) | WP-15, 20 | Extended to work items | none | Keep / extend |
| NW-REQ-274 | Coalesce repeat update requests | L357 | CN | PART (`jobs_document_kind_uq`) | WP-14 | Test: two update clicks produce one set of jobs | none | Planned |
| NW-REQ-275 | Assemble the requested report from consistent saved revisions | L359 | PD | MISS | WP-41 | Single snapshot transaction | none | Planned |
| NW-REQ-276 | If reading is pending or failed, offer a draft from the last completed state or wait for refresh; never silently claim freshness | L359 | PD | MISS | WP-41 | AT-20 | none | Planned |
| NW-REQ-277 | Protect manual edits by stable section and item IDs | L361 | PD | MISS | WP-41 | Edit survives re-assembly | none | Planned |
| NW-REQ-278 | Refresh untouched content; present conflicts where the source of a protected edit changed | L361 | PD | MISS | WP-41 | `base_content_sha256` mismatch → conflict | D-19 | Planned |
| NW-REQ-279 | Issuing records an immutable snapshot with as-of date, sources, calculations, assumptions, line IDs and library versions | L363 | PD | MISS | WP-43 | AT-23 | D-20 | Planned |
| NW-REQ-280 | Later updates flag drafts and never rewrite issued documents | L363 | PD | MISS | WP-43 | Trigger test; AT-23 | none | Planned |
| NW-REQ-281 | Uploaded evidence of submission is not evidence of approval; a drawing received is not a completed design stage | L365 | CN | SAT (nothing infers it today) | WP-35, 60 | No code path sets an approval or stage from a document | none | Keep |
| NW-REQ-282 | Progress requires an explicit record supported by a user update or appropriate evidence | L365 | PD | MISS | WP-35, 60 | Status changes need an actor or a `SourceRef` | none | Planned |
| NW-REQ-283 | Store minimal delivery records now rather than inferring progress from upload volume | L365 | PD | MISS | WP-35 | Schema exists before Stage 4 | D-03 | Blocked(D-03) |

## M. Reports and eventual profile tabs (L367-L379)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-284 | PMP content: definition and quality; delivery and appointments; authorities and approvals; time and cost; material risks (including unforeseen conditions), changes, decisions and next actions | L371 | PD | MISS | WP-51 | Template section list test | WP-35 | Planned |
| NW-REQ-285 | RFP content: shared brief; consultant stages, services and deliverables; investigations required; interfaces; dates; fee return and proposal requirements | L372 | PD | MISS | WP-45 | Template section list test | WP-30, 31, 35 | Planned |
| NW-REQ-286 | RFT content: shared brief; package inclusions and exclusions, supply/install boundaries and retain obligations; access and sequencing; latent-condition handling and provisional sums; dates; tender return and exact document revisions | L373 | PD | MISS | WP-50 | Template section list test | WP-31 | Planned |
| NW-REQ-287 | Use tables, target/current/variance comparisons, short action statements and approved prose fragments | L375 | PD | MISS | WP-40, 41 | Template review | none | Planned |
| NW-REQ-288 | Colour-labelled citations: E blue evidence, U teal user input, C purple calculation, A amber assumption | L375 | PD | MISS | WP-42 | Render test | none | Planned |
| NW-REQ-289 | Citation text labels remain legible without colour | L375 | CN | MISS | WP-42 | Greyscale render test | none | Planned |
| NW-REQ-290 | Delivery status is separate from provenance | L375 | CN | N/A | WP-35, 42 | Distinct fields and marks | none | Planned |
| NW-REQ-291 | Keep origin, review status and meaning in the database; show one mark per value on screen | L375 | PD | PART (one band mark today) | WP-12, 42, 70 | UI shows one mark | none | Planned |
| NW-REQ-292 | Show material assumptions in the report itself | L375 | PD | MISS | WP-42 | Fixture | none | Planned |
| NW-REQ-293 | Opening a citation reveals detail | L375 | PD | MISS | WP-42, 45 | UI test | none | Planned |
| NW-REQ-294 | PDF references remain meaningful without an app session | L375 | CN | MISS | WP-42, 44 | PDF reference text names document, revision, page and location | D-15 | Planned |
| NW-REQ-295 | Set a minimum font size and render-test the page limit | L377 | PD | MISS | WP-44 | AT-24 | D-16 | Blocked(D-16) |
| NW-REQ-296 | Essential overflow produces a review prompt or an identified attachment, never silent omission | L377 | CN | MISS | WP-44 | AT-24 | none | Planned |
| NW-REQ-297 | Detailed trade requirements and conditions remain referenced parts of the tender package | L377 | PD | MISS | WP-50 | RFT lists the referenced documents by revision | none | Planned |
| NW-REQ-298 | Later UI grouping: Summary; Systems and scope; Time and cost; Delivery and authorities; Evidence and assumptions | L379 | PD | MISS | WP-70 | Owner review | none | Planned |
| NW-REQ-299 | Begin with Summary and Systems only, if sufficient | L379 | PD | N/A | WP-70 | Owner decides "sufficient" at WP-70 | none | Planned |
| NW-REQ-300 | Tabs are projections of shared records; tab design and visual polish follow backend validation | L379 | CN | N/A | WP-70 | Starts after Stage 5 is verified | none | Planned |

## N. Knowledge research (L381-L393)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-301 | Context: the physical graph exists as drafts (140 interfaces, 172 failure modes, 227 rules, 4 with verified clauses) | L383 | CTX (counts stale: F05) | SAT | Mapped→NW-REQ-306-310 | none | none | Mapped |
| NW-REQ-302 | Research is build-time only: it produces draft records in `knowledge/` that the checker validates and the owner reviews | L383 | CN | SAT | WP-K* | Checker; `status: draft` | none | Keep |
| NW-REQ-303 | At runtime Jev remains the only AI | L383 | CN | SAT | = NW-REQ-004 | none | none | Keep |
| NW-REQ-304 | Parallel agents may do K1 to K4 only after K0 fixes the record shapes | L383 | CN | PART: K0 merged before K4, but its gate ("owner approves shapes") is not evidenced, and K4 wrote 560 uc records on unapproved shapes | WP-K1-K4 | K packages require WP-K0 closure; K4 records are in the K0 approval packet | D-01 | Planned |
| NW-REQ-305 | K0: record shapes (actions, interface consequences, `cq.*`, `uc.*`, predicate operators), `SCHEMA.md` and checker; gate: checker passes and the owner approves the shapes | L387 | PD | PART: implemented and checker passes (X); owner approval **not evidenced** | WP-K0 | Owner approval line in the handoff log | D-01 | Implemented (approval pending) |
| NW-REQ-306 | K1: interface audit per cluster for existing buildings and fit-outs (direction and endpoints when one side exists; add missing edges: base-building plant to tenancy air conditioning; fit-out heat loads and partitions to air distribution, zoning, smoke control and thermal performance; new-to-existing tie-ins); mid-tier model, seeds first; gate: checker and merge note | L388 | PD | PART (K4 added 4 interfaces, including the tenant fit-out HVAC one) | WP-K1 | Checker; merge note per cluster | WP-K0 | Planned |
| NW-REQ-307 | K2: consequences from seeds (renovation, remediation and rectification, remediation due diligence, fire and life safety in existing buildings, setup and commissioning, services guides); a smaller model suffices; gate: checker and anchors resolve | L389 | PD | MISS (1 cq) | WP-K2 | Checker strict | WP-K0 | Planned |
| NW-REQ-308 | K3: existing-building law, NSW first (alterations and fire-safety upgrade, essential fire-safety measures, Class 2 practitioner regime, hazardous materials in older buildings, home-building thresholds, heritage); strongest model; primary sources only; `clause_verified: false` until the instrument is read; Australian Standards stay unverified without licensed access; owner review | L390 | PD | MISS | WP-K3 | Each record cites a primary instrument or stays `clause_verified: false` | WP-K0 | Planned |
| NW-REQ-309 | K4: unforeseen-conditions library across the nine categories from the owner lists (batch 1: 1,000 rows; batch 2); sort each row by kind, map to system IDs (Jev labelling, reviewed), de-duplicate against the other batch and existing `fm.*`/`if.*`, enrich into records; the coverage ledger gives every row record IDs or a rejection reason; the checker fails on an unaccounted row; further sources | L391 | PD | PART (Pass A and B done for both batches, 0 pending (X); "Jev labelling" method not verified; cross-cluster merge pass and owner review pending: F23) | WP-K4 | Merge-pass items closed; owner review log | none | Implemented (Pass A/B); merge pass Planned |
| NW-REQ-310 | K4 gate: every entry cites a source; no invented frequencies; checker; owner review | L391 | CN | PART | WP-K4 | Checker plus owner review | none | Planned |
| NW-REQ-311 | K5: owner lessons, structured interviews (what was found, when, what it cost in time, what would have revealed it earlier), recorded as draft records | L392 | PD | MISS | WP-K5 | Interview notes become draft records | owner time | Planned |
| NW-REQ-312 | K6: walk 0991 and 0777 work items through the knowledge; compare proposals with the consultants, investigations and tests those projects actually appointed or budgeted; misses become records and noise is fixed in the knowledge; gate: owner-set share of proposals kept | L393 | PD | MISS | WP-K6 | Walk-through report with the kept share against the owner's threshold | WP-26, corpus | Planned |

## O. Implementation sequence, budgets, acceptance and exclusions (L395-L417)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-313 | Stage 0: the owner approves the direction; an independent reviewer challenges schema, counting rules and scope; recheck HEAD before coding | L399 | PD | N/A | G0 | D-01; review record; HEAD check in every brief | D-01 | Planned |
| NW-REQ-314 | Stage 1: provenance, assumptions, typed values, revision tracking, sites, parts on the site, site- and project-level values; gate: existing profile behaviour, user overrides, reading controls, corpus checks and answer keys pass; one site per project; no visible change | L400 | PD | gate currently fails (F01) | WP-00, 11-15 | `tools/check.ps1` green including both replays | D-17, D-04 | Blocked(D-17) |
| NW-REQ-315 | Stage 2: work items from scope, the action question, part-based locations, the gap check, proposals from interface consequences and K0 to K2; gate: new-build answer keys unchanged; **owner-drafted** work-item keys for 0991 and 0777 (`reviewed: false` until owner review) | L401 | PD | MISS | WP-20-28 (gap check moved to WP-32) | Keys plus replay | D-03, D-23 | Planned (sequencing per D-03) |
| NW-REQ-316 | K research runs in parallel with Stages 1 and 2; K3 to K5 continue through later stages | L402 | PD | PART | WP-K* | Schedule | none | Planned |
| NW-REQ-317 | Stage 3: services, works and supply packages; responsibilities; cost lines on work items; allowance subdivision, baseline, claimed to date and sum reconciliation; no procurement pricing copied into a second ledger | L403 | PD | MISS | WP-30-35 | AT-14, 17, 18 | D-03 | Planned |
| NW-REQ-318 | Stage 4: consultant RFP on 0991 from one shared assembler, with approved clauses, stable edits, citations, immutable issue and two-page export | L404 | PD | MISS | WP-40-45 | AT-01, 23, 24 | D-15, D-16 | Blocked(D-15, D-16) for export |
| NW-REQ-319 | Stage 5: works RFT, then PMP, from the same assembler; the RFT carries responsibilities, retain obligations and latent-condition handling; the PMP carries risks including unforeseen conditions | L405 | PD | MISS | WP-50, 51 | AT-11 | none | Planned |
| NW-REQ-320 | Stage 6: minimal progress, authorities, dates, risks and actions, and changes since issue; cost and time summaries reference their owning records | L406 | PD | MISS | WP-60 (base records pulled into WP-35) | Fixture | D-03 | Planned |
| NW-REQ-321 | Stage 7: simple tabs, optimistic edits with rollback, export layout and final latency verification | L407 | PD | MISS (F16) | WP-70, 71 | Playwright rollback test; VPS bench | none | Planned |
| NW-REQ-322 | New budgets are approved and measured on the target VPS | L409 | TGT | N/A (VPS not measured) | WP-71 | VPS bench report | VPS | Planned |
| NW-REQ-323 | Local edit feedback under 100 ms | L409 | TGT | N/A | WP-70 | Playwright timing | none | Planned |
| NW-REQ-324 | Saved reads and writes p50 ≤100 ms, p90 ≤250 ms | L409 | TGT | **CONF** with existing 50/150 for profile paths (F14) | WP-71 (VPS); every CRUD WP adds its bench path | §8.3 | D-18 | Blocked(D-18) |
| NW-REQ-325 | Deterministic assembly from current state p50 ≤300 ms, p90 ≤1 s | L409 | TGT | N/A | WP-41 | Bench `report_assemble` | none | Planned |
| NW-REQ-326 | Work items and proposals are computed inside the existing code-only rebuild and keep `profile_rebuild` (p50 100 ms, p90 300 ms on Spec Home), with a 0991 fixture added | L409 | TGT | PART (no such bench path, F14) | WP-14, 26, 28 | Bench `profile_rebuild` on both fixtures | D-18 | Planned |
| NW-REQ-327 | The gap check is a read: p50 ≤50 ms, p90 ≤150 ms | L409 | TGT | N/A | WP-32 | Bench `gap_check` | none | Planned |
| NW-REQ-328 | Measure export separately after choosing its renderer | L409 | TGT | N/A | WP-44 | Bench `report_export` | D-15 | Blocked(D-15) |
| NW-REQ-329 | These are targets, not measured results | L409 | CTX | none | Mapped→plan §8.3 status column | none | none | Mapped |
| NW-REQ-330 | Preserve existing filing p50 ≤1 s, p90 ≤2 s | L409 | CN | SAT (gated in `bench/budgets.json`) | WP-00, 27 | `whole_intake` | none | Keep |
| NW-REQ-331 | Normal reads and edits make no Jev call | L409 | CN | SAT (`internal/httpapi/profile.go` header comment; routes) | all | AT-25 | none | Keep |
| NW-REQ-332 | Background reading reports actual progress, not a fabricated ETA | L409 | CN | SAT (pending, active and failed counts in `ReadProfile`) | WP-14 | AT-26 | none | Keep |
| NW-REQ-333 | Adoption measure: the user's active minutes from dropping a capex project's documents to an issuable RFP draft, recorded on the first run of 0991; the owner sets the target from that run | L411 | PD | MISS | WP-45 | Recorded minutes in `docs/evidence/` | D-21 | Planned |
| NW-REQ-334 | Acceptance projects: 0991 (capex fire services in an existing building), 0777 (city tower fit-out), Mornington (new house), Petersham (multi-part new mixed use) | L413 | SR | PART (Mornington and Petersham keyed; 0991 and 0777 not) | WP-28 | AT-01 to AT-04 | D-23 | Planned |
| NW-REQ-335 | Add 0991 and 0777 to `data/eval/profile/manifest.json` by ID only; the corpus stays private; no personal details are recorded | L413 | CN | MISS | WP-28 | Manifest diff holds ID, relative path and hash (the existing manifest shape; INT for "by ID only"); stop and ask if a path holds personal details | D-23 | Planned |
| NW-REQ-336 | Verify: sparse evidence | L413 | SR | none | see AT-07 | AT-07 | none | Planned |
| NW-REQ-337 | Verify: conflicting, superseded and skipped sources | L413 | SR | PART (profile tests exist) | AT-08 | AT-08 | none | Planned |
| NW-REQ-338 | Verify: assumptions accepted then challenged | L413 | SR | none | AT-09 | AT-09 | D-06 | Planned |
| NW-REQ-339 | Verify: mixed physical parts | L413 | SR | none | AT-10 | AT-10 | none | Planned |
| NW-REQ-340 | Verify: retained live systems | L413 | SR | none | AT-11 | AT-11 | D-09 | Planned |
| NW-REQ-341 | Verify: splitting a coarse work item | L413 | SR | none | AT-12 | AT-12 | D-19 | Planned |
| NW-REQ-342 | Verify: a dismissed proposal staying dismissed | L413 | SR | none | AT-13 | AT-13 | none | Planned |
| NW-REQ-343 | Verify: a work item with no installer and one with two | L413 | SR | none | AT-14 | AT-14 | D-09 | Planned |
| NW-REQ-344 | Verify: a tenant fit-out raising base-building capacity | L413 | SR | none | AT-15 | AT-15 | D-29 | Blocked(D-29) |
| NW-REQ-345 | Verify: duplicate scope | L413 | SR | none | AT-16 | AT-16 | none | Planned |
| NW-REQ-346 | Verify: allowance split and double-count prevention | L413 | SR | none | AT-17 | AT-17 | none | Planned |
| NW-REQ-347 | Verify: tax, rounding and null amounts | L413 | SR | none | AT-18 | AT-18 | D-14 | Planned |
| NW-REQ-348 | Verify: concurrent edits | L413 | SR | CONF today (F10) | AT-19 | AT-19 | none | Planned |
| NW-REQ-349 | Verify: Jev failure | L413 | SR | PART (402 surfaced) | AT-20 | AT-20 | none | Planned |
| NW-REQ-350 | Verify: restart recovery | L413 | SR | PART (job leases) | AT-21 | AT-21 | none | Planned |
| NW-REQ-351 | Verify: wrong-org, wrong-project and wrong-site access | L413 | SR | PART (org isolation tests exist; no site) | AT-22 | AT-22 | none | Planned |
| NW-REQ-352 | Verify: preserved issued snapshots | L413 | SR | none | AT-23 | AT-23 | D-20 | Planned |
| NW-REQ-353 | Verify: two-page exports | L413 | SR | none | AT-24 | AT-24 | D-15, D-16 | Blocked(D-15, D-16) |
| NW-REQ-354 | Outside this wave: future accounting | L415 | EX | none | none | No accounting tables | none | Excluded |
| NW-REQ-355 | Outside this wave: complex scheduling | L415 | EX | none | none | FS-only dependencies (D-25) | none | Excluded |
| NW-REQ-356 | Outside this wave: automatic tender issue | L415 | EX | none | none | Issue is a manual user action | none | Excluded |
| NW-REQ-357 | Outside this wave: tender offers and comparison | L415 | EX | none | none | = NW-REQ-220 | none | Excluded |
| NW-REQ-358 | Outside this wave: invoices | L415 | EX | none | none | = NW-REQ-222 | none | Excluded |
| NW-REQ-359 | Outside this wave: Jev allocation of requirements to packages | L415 | EX | none | none | No Jev question assigns packages (AT-25, question inventory) | none | Excluded |
| NW-REQ-360 | Outside this wave: a screen for several projects on one site | L415 | EX | none | none | = NW-REQ-103 | none | Excluded |
| NW-REQ-361 | Outside this wave: builder-side subcontracting | L415 | EX | none | none | = NW-REQ-034 | none | Excluded |
| NW-REQ-362 | Outside this wave: LLMs | L415 | EX | none | none | = NW-REQ-004 | none | Excluded |
| NW-REQ-363 | Outside this wave: major UI redesign | L415 | EX | none | WP-70 limited | Tabs are a regrouping only | none | Excluded |
| NW-REQ-364 | The storage prerequisite (files outside releases, prompt off-site replication, aligned DB and file recovery, rehearsed restoration) stays a separate deployment prerequisite, not bundled into this wave | L417 | EX / CN | N/A | none | No storage migration in any WP; WP-43 export blobs use the existing files store | none | Excluded (guarded) |

## P. Peer review brief and source map (L419-L435)

| ID | Requirement (baseline; amendments in R) | Source | Type | Baseline finding | WP | Acceptance check | Deps | Historical disposition |
| - | - | - | - | - | - | - | - | - |
| NW-REQ-365 | Peer review answers: smallest adequate schema (plan §4); actions complete and literal (NW-REQ-122, WP-K0); site/project split (D-04); assumed inputs barred from verified compliance (D-06); rollups reconcile after subdivision (AT-17); scope, package and systems views avoid double counting (NW-REQ-211, 256); interface consequences over-propose and short-listing (D-11, D-28, WP-K6); reproducibility after reprocessing (D-20); approval, quotation and commitment distinct (NW-REQ-205); realistic update boundaries and speed gates (D-18, §8.3) | L421 | PD | N/A | G0 | Independent review record answers each question | D-01 | Planned |
| NW-REQ-366 | Clerk reference data (asset register, consultant rosters, seed guides) is copied as data, never code | L433 | CN | SAT (K0 copied conditions as data) | WP-30, K0, K2 | Review: no Clerk code | none | Keep |
| NW-REQ-367 | Every Jev use follows and cites TypeSafe's pages (fan-out, how to build, confidence, jev-1.13) | L435; `AGENTS.md` rule 2 | CN | SAT (existing code cites them) | WP-22, 23, 27 | Comments cite the page | none | Keep |

## Q. Planner additions (INT, need owner approval)

| ID | Addition | Reason | WP | Acceptance check | Approval | Disp. / State |
| - | - | - | - | - | - | - |
| NW-REQ-368 | Restore the profile regression gate at HEAD before Stage 1 | F01 (X): replays stale; blocks NW-REQ-118 and NW-REQ-314 | WP-00 | Both replays green; no change to answer keys | D-17 | Planned |
| NW-REQ-369 | Measure the evidence fan-out workload (questions per call, tokens, elapsed) before and after K4 | F02; NW-REQ-131, 272 | WP-00 | Evidence note in `docs/evidence/` | D-17 | Planned |
| NW-REQ-370 | Key-scope registry `knowledge/profile/key_scope.yaml` | D-04; NW-REQ-236 | WP-12 | Checker fails on an unclassified key | D-04 | Blocked(D-04) |
| NW-REQ-371 | `project_revisions` counters | §4.4; NW-REQ-268, 269 | WP-14 | Counter test | D-01 | Planned |
| NW-REQ-372 | Knowledge content hash at startup | NW-REQ-177, 240 | WP-14 | Hash changes when any loaded file changes | D-01 | Planned |
| NW-REQ-373 | `proposals` projection table | NW-REQ-137-142 | WP-26 | Rebuilt per build | D-01 | Planned |
| NW-REQ-374 | `report_edits` table | NW-REQ-277, 278 | WP-41 | Schema test | D-01 | Planned |
| NW-REQ-375 | `adoption_events` table | NW-REQ-333 | WP-45 | Minimal fields only | D-21 | Planned |
| NW-REQ-376 | Part-level work type override and `work_type` predicate | D-07; NW-REQ-080 | WP-21, 25, K0 | Hale fixture | D-07 | Blocked(D-07) |
| NW-REQ-377 | `needs_design` per action in `actions.yaml` | D-09; NW-REQ-259 | WP-K0 | Checker | D-09 | Blocked(D-09) |
| NW-REQ-378 | Make `existing_building_year` and `existing_building` profile determinants (site, classification) so they are read | F06; NW-REQ-082 | WP-15 | Harvest test; label fingerprint change measured | D-01, D-17 | Planned |
| NW-REQ-379 | New knowledge shapes: `stages.yaml`, `package_defaults.yaml`, clause catalogue, benchmark catalogue, `planning_keys.yaml`, proposal `action` for `work_item` kinds | NW-REQ-051, 199, 226, 238, 150, 263 | WP-K0 | Documented in `SCHEMA.md` first; checker | D-01 | Planned |
| NW-REQ-380 | Amend the PRG concept table (L37-L55) for new concepts: proposal (projection), revision counter, report edit, planning value, adoption event, scope-cost link, package stage, cost item revision (justified in plan §4.0) | L59 discipline | G0 (owner edits the PRG) | PRG updated by the owner | D-01 | Blocked(D-01) |

## R. Review amendment — current requirements (5 October 2026)

The owner requested incorporation of the review recommendations, with no implementation. These rows supersede conflicting baseline wording; all are **Planned / not verified**. They do not mark any knowledge or answer key reviewed. Original requirement IDs remain valid.

| ID | Requirement / baseline affected | Source | Packages | Acceptance | State |
| - | - | - | - | - | - |
| NW-REQ-381 | One current decision table and evidence summary; distinguish historical results, planning defaults and explicit owner approval. Restore complete all-sheet replay; no green gate inferred from a baseline or local timing alone. Updates 313, 314, 368, 369. | Amendment; plan §0.2, D-36 | Integration lead; WP-00, 22, 23, 27, 71 | AT-35; current evidence links and no stale blocker treated as current | Planned |
| NW-REQ-382 | M1 0991 draft before full costing, signals, subdivision, issue/export; final shared records/assembler, minimal review UI, no competing ledger. M2 preserves issuable RFP. Updates 015–020, 093, 318, 321 and original sequence. | D-31; plan §1.1; packages §2.4 | WP-28, 35, 40, 41, 42, 45 (a/b phases) | AT-34, then AT-01/23/24 at M2; no issue endpoint at M1 | Planned |
| NW-REQ-383 | Unverified-input derivations display amber and planning-only wording, never verified/compliant. Assumptions/allowances/forecasts/planning values remain ineligible. Updates D-06 and provenance requirements 175–192. | D-32 | WP-15, 42, 45 | AT-31 across profile, draft refresh and export; factual answer keys preserved | Planned |
| NW-REQ-384 | Exactly one accountable design role on a services OR explicit D&C works package; supply-only cannot discharge it. Do not require a second consultant package. Updates 259/260/343 and D-09. | D-33 | WP-31, 32, 45, 50 | AT-32 owner-PM/D&C/no/duplicate/supply-only cases | Planned |
| NW-REQ-385 | Freeze owner-reviewed keys; zero missed critical obligations, ≥90% work-item precision and recall, ≥80% useful top-list proposals; critical recall evaluated across complete output and critical items never hidden by ten-item cap. Updates 312, 333, 334. | D-34; plan §8.6 | WP-22, 23, 26, 28, 45, K6 | AT-34 on 0991 before M2; 0777 before RFT/PMP; counts and false positives reported | Planned |
| NW-REQ-386 | Same-corpus manual baseline before scored run, ≥30% active-minute reduction; wall time separate. M1 reviewable draft and M2 issuable RFP measured separately. Stopwatch sufficient; analytics optional. Updates 333/375. | D-35 | WP-45 | AT-34; frozen task/completeness checklist, assistance and correction time recorded | Planned |
| NW-REQ-387 | Actual action/location/signal workloads and request limits measured; preflight oversized calls to visible failure, never silent truncation or serial hot-path calls. Updates 131/272/369. | D-36; plan §8.6 | WP-22, 23, 27 | AT-33, AT-25/27 under representative load | Planned |
| NW-REQ-388 | Local whole-intake pass does not prove component, replay or VPS gates; required red gates block merges/release. Existing title baseline is a regression floor, not quality approval. Updates 322/330/368. | D-36; plan §0.2/§8.6 | WP-00, 71 | AT-27/35; full-sheet coverage, explicit title metrics, target-VPS evidence | Planned |
| NW-REQ-389 | Review/correct RFP source metadata; only reviewed clauses establish obligations; M1 is not issue/release approval. Link independent storage/restore prerequisite before deployment. Updates 304/318/364. | Amendment; plan §8.6 | WP-45, 71 | AT-31/34 at M1, AT-23/24 at M2; restore evidence at release | Planned |

Current disposition precedence: section R for changed behaviour → plan §5 decisions → latest dated requirement evidence below → historical baseline cells. Only evidence can advance Implemented/Verified; editing this register cannot do so. States in package history and the old traceability counts are not alternative authorities.

## Source coverage ledger

Every line range of the PRG at the hash above maps to a disposition.

| PRG lines | Content | Disposition |
| - | - | - |
| L1 | Status, reviewed commit, revision note | NW-REQ-001; revision note is CTX (PRG history) |
| L3 | Foundation, retained stack, exclusions | NW-REQ-002, 003, 004 |
| L5 | Full spectrum | NW-REQ-005 |
| L7 | Heading | none |
| L9-L31 | Twelve captured decisions | NW-REQ-006-039 (each sentence split) |
| L33-L35 | Ontology discipline | NW-REQ-040, 041 |
| L37-L55 | Concept table (17 rows) | NW-REQ-042-058, one row each |
| L57-L67 | Five discipline rules | NW-REQ-059-063 |
| L69-L77 | ISO 12006-2 mapping table (7 rows) | NW-REQ-063 (each row is a mapping, recorded in that requirement's source cell) |
| L79-L88 | Existing-foundation table (6 rows) | NW-REQ-064-069 |
| L90 | Current conditions and facts | NW-REQ-069 |
| L92 | Missing ledgers; grades feed quality; builds not snapshots | NW-REQ-070, 071, 072 |
| L94 | Bands vs provenance; typical ≠ benchmark; part-level evidence; presence ≠ scope | NW-REQ-072 (bands context → 175-181), 073, 074, 075 |
| L96-L114 | Nine gaps | NW-REQ-076-084 |
| L116-L124 | Three dimensions table | NW-REQ-085-088 |
| L126 | Work item as junction; M:N; stable IDs; project-wide money | NW-REQ-089-092 |
| L128 | One assembler; internal packages; no workflow or untyped table | NW-REQ-093-095 |
| L130-L138 | Sites and parts | NW-REQ-096-104 |
| L140-L154 | Work items | NW-REQ-105-120 |
| L156-L169 | Actions table (8 rows) | NW-REQ-121, 122 (all 8 rows) |
| L171 | Action question | NW-REQ-123-131 |
| L173-L175 | Locations | NW-REQ-132-135 |
| L177-L179 | Building logic rules | NW-REQ-136-142 |
| L181-L183 | Interface-consequence rules | NW-REQ-143-146 |
| L185-L195 | Interface-consequence table (9 rows) | NW-REQ-147-155 |
| L197 | Examples | NW-REQ-156, 157, 158 |
| L199 | Fit-out research list | NW-REQ-159 |
| L201-L216 | Consequences and illustrative YAML | NW-REQ-160, 161 (DX: not approved knowledge) |
| L218 | Proposal kinds; K3 research drafts | NW-REQ-162, 163 |
| L220-L222 | Unforeseen conditions definition | NW-REQ-164, 165, 166 |
| L224-L234 | Nine categories table | NW-REQ-167 |
| L236-L258 | Illustrative uc YAML including signals | NW-REQ-168 (DX) |
| L260 | Frequencies; uc vs fm; fm unchanged; report uses | NW-REQ-169-172 |
| L262 | Predicate operators | NW-REQ-173, 174 |
| L264-L278 | Provenance and assumptions | NW-REQ-175-192 |
| L280-L294 | Cost item link, example table, editable suggestions | NW-REQ-193-196 |
| L296-L298 | Package kinds, novation, defaults, subdivision | NW-REQ-197-204 |
| L300-L318 | Cost rules (9 bullets) | NW-REQ-205-227 |
| L320-L322 | Schema preamble | NW-REQ-228-231 |
| L324-L341 | Schema table (16 rows) | NW-REQ-232-258 |
| L343 | Gap check | NW-REQ-259, 260 |
| L345-L349 | Knowledge additions, catalogues, exclusions | NW-REQ-261-267 |
| L351-L363 | Refresh steps 1-6 | NW-REQ-268-280 |
| L365 | Submission ≠ approval; minimal delivery records | NW-REQ-281-283 |
| L367-L373 | Report content table (3 rows) | NW-REQ-284-286 |
| L375-L379 | Presentation, citations, page limit, tabs | NW-REQ-287-300 |
| L381-L383 | Research preamble | NW-REQ-301-304 |
| L385-L393 | K0-K6 table (7 rows) | NW-REQ-305-312 |
| L395-L407 | Stage table (8 rows) | NW-REQ-313-321 |
| L409 | Budgets | NW-REQ-322-332 |
| L411 | Adoption measure | NW-REQ-333 |
| L413 | Acceptance projects and scenarios | NW-REQ-334-353 |
| L415 | Exclusions (10) | NW-REQ-354-363 |
| L417 | Storage prerequisite | NW-REQ-364 |
| L419-L421 | Peer review brief | NW-REQ-365 |
| L423-L435 | Source map links | CTX (links), plus NW-REQ-366 (Clerk as data) and NW-REQ-367 (TypeSafe citations) |

## Historical traceability summary (original 380 rows)

These figures describe the original planning baseline only. They exclude the nine review-amendment requirements (381–389) and later state updates; they are not current completion totals.

| | Count |
| - | - |
| Requirements (NW-REQ-001-380) | 380 |
| … of which planner additions (INT, section Q) | 13 |
| Disposition Planned | 266 |
| Disposition Keep (already satisfied; regression-checked) | 39 |
| Disposition Blocked on a decision, approval or library | 37 |
| Disposition Excluded (guarded) | 17 |
| Disposition Mapped (context only) | 11 |
| Disposition Implemented (knowledge drafts, unverified) | 10 |
| Verified | **0** |

Retain these counts as history. The revised register has 389 unique requirement IDs, including nine new Planned requirements; no new verification is claimed here.

Every requirement with disposition Planned, Keep or Blocked names at least one package or AT and one acceptance check. Each WP in the packages document lists the requirement IDs it serves; the integration lead checks this both ways at each merge (packages document §3).

## State updates

### 7 October M2/M3 implementation (current)

The owner authorized unattended completion and selected Petersham PPR as the
principal residential acceptance set. M1 checkpoint commits `d97e6ee`/`2edc077`
are on GitHub. M2/M3 product behavior is **Implemented** locally, including
split/retire, automatic proposals and action choice, evidence signals, costs and
reviewed-benchmark selection, delivery dependencies, frozen issue/export,
RFP/RFT/PMP, changes since issue and the corresponding UI. This dated update
supersedes older planned/partial implementation states for those delivered
behaviors; it does not promote their quality/release acceptance to Verified.

[Checkpoint validation](../evidence/2026-10-07-m2-m3-validation.md) records
passing full Go, all 29 browser cases, OCR, intake replay, cost/report,
populated upgrade, crash recovery and local dump/restore checks. Final full
regression passed Go, all 29 browser cases, OCR and intake replay; source/Hale replay remains stale and blocks the full gate. The complete
local 184×2 intake / 40-sample timing run passes every developer user-path
budget, including Spec Home p50/p90 47.6/49.9 ms against 50/150 ms; this does not
certify target-VPS timing. No standing budget or acceptance threshold changed.

The [final knowledge audit](../evidence/2026-10-07-final-knowledge-audit.md)
records **Implemented** bounded K1–K4 source-backed draft coverage, explicit
layout input, six NSW review families, and K4 merge/contract/detector corrections.
It supersedes earlier missing-jurisdiction/layout and bounded implementation-gap
claims. All new knowledge remains draft; source/project-data limitations and
owner review are distinct from implementation completion.

Private source/Hale profile accuracy is stale and blocked: the live refresh was
rejected by automatic approval review and is not bypassed. Offline size/routing
preflights are not accuracy evidence. Owner content, usefulness and measured
time-saving judgments, knowledge review and target-VPS/release acceptance
(including NW-REQ-381–389) remain open. The [package table](2026-10-04-next-wave-agent-work-packages.md#6-status-and-evidence)
and [progress page](NEXT-WAVE-STATUS.md) record publication and final gate status;
this update grants no blanket Verified state.

### 6 October local implementation reconciliation

The plan §0.2 and package status table now distinguish the local WP-14/15,
WP-20–23, WP-25/26/28, WP-30–32/35a, WP-40a–42a/45a and WP-X1 work from the
older planning baseline below. Named package handoffs in `docs/evidence/`
record implementation and bounded validation. This update does not mark all
requirements assigned to those packages Implemented or Verified: partial
phases, changed-question live evaluation, automatic proposal generation,
owner-reviewed quality, later cost/issue/export/delivery work and release
evidence remain outstanding. No merge or full green gate is claimed.

The intermittent profile latency failure remains open, despite passing
diagnostic runs. The owner's confirmed proposal-undo behaviour includes
historical-use references; its narrow store/API and endpoint timing evidence
is `docs/evidence/2026-10-06-proposal-undo-history.md`. Current K4 source and
attachment corrections are indexed in
`docs/unforeseen/k4-current-review-status.md`; none grants owner review.

Historical state updates follow unchanged. Their pre-change replay results
must not be treated as evidence for newly changed action/location questions.

The integration lead appends evidence updates here; the newest dated evidence overrides historical baseline cells. The current cross-package summary is plan §0.2. This review amendment leaves historical evidence intact.

| Date | Requirements | New state | Evidence |
| - | - | - | - |
| 2026-10-05 | NW-REQ-368, 369 | Verified | Replays green at HEAD (source 15/15, Hale 12/12, 0 forbidden); `docs/evidence/2026-10-05-evidence-workload.md`; commits `6362148`, `836055d` |
| 2026-10-05 | NW-REQ-118 | Implemented | Both profile replays green. The answer-key harness (Mornington, Petersham, Newham, Rutherford) has not been run, so this is not Verified. |
| 2026-10-05 | NW-REQ-131 | Implemented | Call counts, tokens and elapsed time recorded for the K4 fingerprint change. The action-question re-read (WP-22) is still to come. |
| 2026-10-05 | NW-REQ-272, 330 | Not verified | The latency bench cannot run (F29). |
| 2026-10-05 | NW-REQ-025, 026, 042, 096, 097, 098, 102, 232, 233 | Implemented | WP-11 `23d89a5`: store and httpapi tests; migration dry run on a dev copy; independent review |
| 2026-10-05 | NW-REQ-230 | Implemented for parts and sites | Composite FKs for parts (site, creating project on the same site). Profile values come in WP-12. |
| 2026-10-05 | NW-REQ-099 | In progress | Parts exist; base-building items on site parts come with WP-20 |
| 2026-10-05 | NW-REQ-173, 174, 144 (evaluator half), 261 (runtime half), 264 (loader half) | Implemented | WP-25 `17aeb0b`, `6524887`; knowledge tests; replays unchanged; independent review |

| 2026-10-05 (review amendment) | NW-REQ-381–389 | Planned, not implemented | Documentation-only request; plan §1.1 and §8.6. No application tests run. |
| 2026-10-05 (evidence reconciliation) | NW-REQ-272, 330 | Local whole-intake evidence recorded; not release-verified | Latest follow-up in `docs/evidence/2026-10-05-evidence-workload.md`: live p50 377 / p90 1,245 ms; replay bench/component and VPS gates remain unresolved. Supersedes the earlier “bench cannot run” statement only for the local live result. |
| 2026-10-05 (M0) | NW-REQ-381 (all-sheet replay half), 272, 330 | Implemented (local); not release-verified | M0 `e2eaa6d`: the filing eval splits drawing sets into sheets as the app does (AT-35); live re-recording 218 calls, metrics unchanged (title 0.29, 15 confident wrong; no new baseline); replayed bench 0 unrecorded requests. `docs/evidence/2026-10-05-evidence-workload.md`. |
| 2026-10-05 (D-37) | NW-REQ-272, 330 | Gate rule recorded | `31681f9`: owner chose (b); five component budgets are `where: release`, reported but not gated off the target VPS; user paths gate everywhere. Release evidence (WP-71) still outstanding. |
| 2026-10-05 (WP-12) | NW-REQ-027, 045, 058, 100, 101, 175, 178, 179, 180, 181, 182, 230 (profile values), 231 (profile values), 234, 235, 236, 291 (database half), 370 | Implemented | WP-12 merged `ece0086` (commits `4d563bf`, `dc6a903`): migration 012 with assertions, `knowledge/profile/key_scope.yaml` (D-04), store, httpapi and reconcile tests; independent review, findings fixed; full gate green on the rebased branch (knowledge, Go, e2e 11/11, intake replay, profile replays 15/15 and 12/12, bench). |
| 2026-10-05 (WP-12) | NW-REQ-021, 184, 237 | In progress | 021: replays unchanged; the answer-key harness was not run. 184: rows carry file hash, revision, page and offsets read at rebuild; frozen snapshots for issued reports come with WP-43. 237: no write reads `profile_rows`; WP-15 and WP-20 extend this. |
| 2026-10-05 (F30) | NW-REQ-272 (document list path) | Defect fixed | `3e55ca1`: intermittent ~200 ms document list after bulk filing (stale-statistics join); decisions read by document id, documents indexed by project; independent review; full gate green. F31 (profile edit under stale statistics) recorded in plan §4. |
| 2026-10-05 (WP-13) | NW-REQ-007, 175 (calculation origin in the planning table), 182, 187, 188, 189, 190, 238, 239 | Implemented | WP-13 (commits `4059d48`, `59a658f`): migration 013 (typed value, explicit unknown, composite FKs, one live value per key, history kept, never verified); draft `knowledge/profile/planning_keys.yaml` with favourable values refused (L276) and no money keys; planning API with 422/409/404 per plan §3.4; reconciliation shows `plan.<key>` rows as assumptions, the user's stated word wins, no derivation reads them. Store, httpapi, profile and knowledge tests; independent review (one important finding fixed: an edit omitting origin could make an assumption favourable); full gate green. |
| 2026-10-05 (WP-13) | NW-REQ-006, 176, 183, 186 | In progress | 006: unknowns are listed as unresolved decisions; quality, time and cost definition come later. 176: the `provenance` column exists, but no calculation engine writes input origins yet. 183: rationale, limitations, actor and time recorded; method and library version come with calculations (WP-34). 186: suggestions are limited to keys without a value; "consequential" (an input to an applicable rule, predicate or estimate) is not yet computed. |
| 2026-10-05 (WP-13) | NW-REQ-008, 185 | Blocked (libraries) | Starting values are offered only from a `reviewed` registry; `planning_keys.yaml` is draft, so none are proposed. |

| 2026-10-05 (F31) | NW-REQ-272 (profile edit path), 314 (standing regression gate) | Prerequisite defect fixed; WP-14 not implemented | Exact-ID snapshot lookups and bounded profile response queries; single-statement provenance preserved under reprocessing. Full local gate passed and independent review resolved; see `docs/evidence/2026-10-05-f31-profile-snapshot.md` and `bench/results/latest.json`. No release verification claimed. |
