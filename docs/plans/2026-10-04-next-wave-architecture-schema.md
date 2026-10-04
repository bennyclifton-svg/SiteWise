Draft for owner review and a later independent peer review. No implementation is authorised by this document. Reviewed 4 October 2026 against SiteWise main commit `02094dbde0d9bd90f02d82c6ccc5bea2933a620c`; static source inspection only. Revised the same day after owner review: adds core concepts, the works model (sites, work items, actions), building logic for works (interface consequences, consequences and unforeseen conditions), knowledge research, three package kinds, a full-spectrum acceptance set and a report-first sequence.

Build one shared project foundation supporting a live PMP, consultant RFPs, trade RFTs and a progressively detailed cost plan. Retain Go, PostgreSQL, Jev, local document storage, existing jobs and SSE. No LLM, agent harness, vector database or additional service in this wave.

One model must serve the full spectrum, through design and delivery: single-system capex work in occupied buildings (fire pumps, chillers, roof membranes, facades), fit-outs, remediation, extensions, and new houses, warehouses, commercial buildings and residential towers.

## Decisions captured

- The profiler establishes the reviewable project definition: scope, quality, time, cost, constraints, evidence and unresolved decisions. It supports incomplete projects and offers useful starting assumptions.

- Jev selects among bounded candidates. Code parses, calculates, applies rules and assembles approved wording. Users can edit content directly.

- PMP combines delivery plan and progress report; RFP covers a consultant appointment; RFT covers a construction package. All consume shared state independently. Generating a PMP is not a prerequisite for an RFP or RFT.

- Reports target roughly 1½ pages and must fit within two pages at a readable size. Detailed pricing schedules and technical references may be separate, clearly identified attachments. The cost plan itself is not limited to two pages.

- Readable evidence means eligible for consideration, not automatically correct. Preserve reading choices, revisions, conflicts, physical scope and user overrides.

- Detail the backend first. Summary and Systems tabs are a sensible later presentation of the same records, not separate stores.

- The site, not the project, is the lasting record of what exists. A project is an intervention on a site. Version 1 has one project per site, but building facts are keyed to the site so a later project on the same building starts from what is known (owner direction, 4 October 2026).

- Scope is a set of work items: an action on a system at a part. The scope picker keeps working; its choices become coarse work items.

- The building model says what works trigger: physical consequences through the interface graph, regulatory consequences through rules, and the unforeseen conditions that commonly arise. Code evaluates them. They appear as proposals with their reason, which the user accepts or dismisses.

- Primary user: the owner-side project manager who appoints consultants and contractors (assumed; owner to confirm). Builder-side subcontract procurement comes later and reuses the same records.

- Small projects must feel small. A capex job shows a handful of work items and packages; nothing is required that the job does not need.

- First report: a consultant RFP on a real capex project, then the works RFT, then the PMP, all from one assembler.

## Core concepts

An ontology is the agreed list of the kinds of things in the domain, what each means and how they relate. `knowledge/SCHEMA.md` is the ontology of the building (systems, determinants, rules, interfaces, failure modes). This section adds the ontology of the works and the concepts that join the two. Adopt the discipline, not the technology: no OWL, RDF store, graph database or reasoner. Postgres rows, YAML catalogues and code are sufficient and keep the speed budgets.

| Concept | Meaning | Held in |
| - | - | - |
| Site | One address or campus and everything built on it. Persists across projects. | `sites` (new) |
| Part | A building, storey, tenancy, roof, plant area, compartment, outbuilding or zone of a site. NCC class attaches here. | parts, moved from project to site |
| System | A kind of physical system. | `knowledge/` systems |
| Determinant | A fact about a site or part that decides which rules apply. | `knowledge/` determinants; profile |
| Project | A bounded intervention on one site, with scope, quality, time and cost objectives. | `projects` |
| Action | What the works do to a system: new, replace, upgrade, alter, repair, remove, retain, investigate. | `knowledge/works/actions.yaml` (new) |
| Work item | An action on a system at a part, with existing condition and target. Coarse first, split when needed. | `work_items` (new) |
| Package | A contract: services (consultant), works (head contract or trade) or supply. | `packages` |
| Responsibility | A package's role for a work item (design, supply, install, test, certify and so on), or a general obligation. | `package_scope_items` |
| Stage | A delivery phase: investigation, design stages, approvals, procurement, construction, completion, defects. | catalogue; delivery items |
| Cost line | Money on a work item, on a package stage, or on a project-wide category. | `cost_items`, `cost_values` |
| Delivery item | Milestone, activity, approval, risk, issue, decision or action. | `project_delivery_items` |
| Rule | A provision of an instrument; applies when its conditions hold. | `knowledge/` rules |
| Interface | A typed physical relation between systems. | `knowledge/` interfaces |
| Consequence | What a kind of work on a system requires, by rule or by interface. | `knowledge/` (new) |
| Unforeseen condition | A state of the site or building commonly discovered during works that a kind of work makes likely. | `knowledge/` (new) |
| Evidence | A source record with document revision, hash and location. | `document_sources`, `passage_sources` |

Discipline:

1. Each table or knowledge file represents one concept above. A new concept is added here before any schema.

2. Relations are explicit IDs or foreign keys, never inferred from text at read time.

3. Every vocabulary Jev reads has literal `describes` and `excludes` text with boundary cases ([jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13)). The precision of these definitions is Jev's accuracy.

4. What is generally true lives in `knowledge/`; what is true of one project lives in Postgres.

5. Concepts align with the ISO 12006-2 construction classification framework, on which Uniclass and OmniClass are built. Codes from those, NATSPEC worksections or AIQS elements may later be added as mapping fields. They are not structure.

| ISO 12006-2 class | SiteWise concept |
| - | - |
| Construction complex, construction entity | Site; part of kind building |
| Built space | Part (storey, tenancy, roof, plant area, zone) |
| Construction element | System |
| Work result | Work item |
| Construction process (management, stage) | Stage; delivery item |
| Construction agent (role) | Package discipline and responsibility role |
| Property and characteristic | Determinant; work item target |

## Existing foundation and gaps

| Existing code or data | Retain and extend |
| - | - |
| `knowledge/SCHEMA.md` | Systems, determinants, rules, interfaces and failure modes remain the physical building model. Discipline and trade are views over that model. |
| Migration 008 | Reuse `project_parts`, `profile_facts`, protected `profile_user_values`, precomputed `profile_rows` and `profile_builds`. |
| Migrations 009 and 010 | Reuse document sources, passage locations, cached passage calls and `documents.profile_read = auto/read/skip`. |
| `internal/profile/build.go` and `reconcile.go` | Preserve pure reconciliation, supersession handling, user precedence and scope suggestions. |
| `internal/store/profile.go` | Retain transactional rebuilds and events; add revision tracking rather than replacing the pipeline. |
| `Profile.tsx` and `httpapi/profile.go` | Current UI exposes category, subclass, work type, dynamic scale fields, conditions, approvals/contract facts, scope checklist, systems and compliance. |

Current conditions include planning pathway, procurement route, access, occupation, stakeholders, environmental sensitivity and industrial conditions. Facts include consent number/date/lapse, contract form/basis, defects period and design life. Systems expose inclusion, provider and notes; determinants are grouped as classification, site, services and fire.

No dedicated cost ledger, project milestone register, report-version model or general quality specification exists in the reviewed migrations/UI. Existing grades and other subtype fields should feed quality, not be duplicated. `profile_builds` records time and thresholds, but is not an immutable project snapshot.

Current colour bands combine confidence, conflicts, suggestions and user authorship. They cannot alone express the proposed provenance model. Existing taxonomy “typical” descriptions are not approved cost or programme benchmarks. Scope checklist rows currently aggregate at whole-project level; preserve part-level evidence when packages need finer boundaries. A system being present in an existing building does not by itself establish that the works include it.

Gaps for the full spectrum:

- Scope records only in or out per system. It cannot say replace, upgrade, alter, repair, remove, retain or investigate.

- The presence question asks whether the completed project will have a system. In an existing building that is true of nearly every system, whether or not the works touch it.

- Parts and building facts belong to a project, so the next project on the same building starts from nothing.

- The passage location question (`source.scope` in `internal/jobs/source_reading.go`) offers a fixed residential list (`apartments`, `basement`, `common_areas`, `roof`, `site`) rather than the project's parts.

- Work type is one value per project, though Hale is both an extension and a refurbishment.

- The interface graph (140 draft interfaces) describes new-design coordination. It has no notion of an existing side, and no edges for a tenant fit-out acting on base-building plant, air distribution or thermal performance.

- No determinant records building age, which decides hazardous-materials obligations.

- The profile evaluation set holds no capex, fit-out or remediation project, although the private corpus does (`0991-hydrant-sprinkler-head-upgrade`, `0777-city-tower-fitout`).

- Clerk's asset register (`data/taxonomy/asset-register.json`) recorded existing condition and proposed action because, in its words, services-dominant work otherwise has nowhere for make, capacity, age and zones to land. This plan carries that lesson into the work item.

## Shared architecture

Maintain three linked dimensions, joined by the work item:

| Dimension | Example | Ownership |
| - | - | - |
| Physical building | Site, parts, systems; relevant determinants and interfaces | Existing knowledge model, site and project profile |
| Delivery | Investigation, design, authorities, procurement, construction; owners and dates | Small project delivery records |
| Commercial | Consultant, works and supply packages and staged pricing lines | Package records and cost plan |

A work item is where the dimensions meet: it names a system and part (physical), is designed, supplied and installed by packages (commercial) and is scheduled through stages and delivery items (delivery). A package can address several work items; a work item can involve several packages in different roles. Use stable identifiers and explicit links. General fees, approvals and contingency can be project-wide without inventing physical systems for them.

One Go application assembles each report from a consistent revision of profile, works, package, delivery and cost data. Separate internal packages for planning, procurement, costs and reports are sufficient. Do not introduce a generic workflow platform or move all domain records into one untyped entity table.

## Works model

### Sites and parts

A site is one address or campus. Parts belong to the site. Existing part kinds (`whole`, `building`, `part`, `storey`, `compartment`, `tenancy`, `outbuilding`) gain `roof` and `plant_area`. A tenancy is a part; base-building systems serving it are site systems.

Values that describe what exists belong to the site part: determinants, building category and class, scale fields, existing systems and their condition. Values that describe the works belong to the project: work type, conditions such as procurement route and occupation, project facts such as consent and contract, and work items.

Version 1 creates one site per project, including existing projects, so nothing visible changes. Attaching a second project to a site, and a screen for it, come later. Facts from a project's documents about the building become site facts with their document provenance.

### Work items

A work item is an action on a system at a part: for example replace · fire water (pump set) · plant room. Fields: part, system (a leaf or a top-level system), action, title, existing condition, target, optional quantity and unit, parent, origin, review status, provenance and version.

- Coarse items come from the scope picker and document evidence: one per part and in-scope system. The default action follows the work type: `new` and `extend` give new; `refurb` gives alter; `remediation` gives repair; `advisory` gives investigate. User choices remain final, as scope choices are today.

- Splitting a work item makes the parent a group, as allowance subdivision does. Small projects never need to split.

- Existing condition uses Clerk's conditions (serviceable, nearing end of life, end of life, beyond economical repair, failed, defective or non-compliant, unknown) plus a short note. Target holds typed values or approved text; quality lives here (grades, ratings, performance).

- Proposed items follow the plan's review statuses. New evidence can propose or conflict but never silently overwrite an accepted item.

- A system stated present in an existing building, with no works action, is recorded on the site, not in the works. For `new` and `extend` projects today's behaviour holds: presence adds the system to the works as new, and the existing answer keys must still pass.

- `retain` records an existing system that stays and must be protected or kept operating. It is not physical work, but it reaches packages as an obligation (for example keep the hydrant system live throughout).

### Actions

Jev reads these, so each has literal boundaries. Draft wording for owner review:

| Action | Describes | Boundary |
| - | - | - |
| new | The works install it where none exists now, or in a new building or extension. | Adding capacity to an existing system is upgrade. |
| replace | The works remove an existing one and install an equivalent of similar capacity or performance. | A stated increase in capacity or performance is upgrade. |
| upgrade | The works increase the capacity or performance of an existing one, or bring it to a newer standard, including by adding components. | Like-for-like renewal is replace. |
| alter | The works relocate, extend or modify an existing one to suit a new layout, without a stated change in capacity or performance. | Restoring a defect is repair. |
| repair | The works repair, remediate or rectify a defective or deteriorated existing one so it performs as intended. | Renewing the whole of it is replace. |
| remove | The works remove or decommission an existing one without replacing it. | Removal followed by an equivalent is replace. |
| retain | An existing one stays in place unchanged, or must stay in operation or be protected during the works. | Any physical change is another action. |
| investigate | The works inspect, test, survey or audit an existing one to establish condition or compliance, without stated physical work. | Testing and commissioning new or replaced work belongs to that work. |

Jev question `sys.<leaf>.action`, asked in the existing evidence fan-out alongside presence and provider for each labelled leaf, so no new round trip ([fan-out](https://docs.typesafe.ai/patterns/fan-out)). One choice: the eight actions plus `several` and `not_stated`. It is a separate atomic question from presence and provider ([how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)) and has its own threshold, because choice confidence depends on option count ([confidence](https://docs.typesafe.ai/confidence)). Instructions: ``Using `text`, what does the passage say the works do to the <label>?`` `several` goes to Needs mapping for the user; `not_stated` falls back to the work-type default. Adding the question changes the evidence-call fingerprint, so the next update re-reads evidence for selected documents; record call count, tokens and elapsed time.

### Locations

Replace the fixed `source.scope` options with options built by code from the site's parts, plus `whole_project`, `specific`, `multiple` and `not_stated`. The option count varies by project, so thresholds are set per option count ([confidence](https://docs.typesafe.ai/confidence)); until calibrated, applied answers are amber only. Fact-to-part matching by exact label stays.

## Building logic for works

The building model earns its place when doing something to one system raises what that touches. Three kinds of knowledge do this. All are evaluated by code during the existing code-only profile rebuild; none calls Jev. Each result is a proposal carrying its reason: the work item and the interface, rule or record that raised it. The user accepts or dismisses it. A dismissal is final until its inputs change. A large library must still give each project a short list: proposals are ranked by severity (life safety first) and by how specifically the work items match; the owner sets how many show after the first run, and the rest are one click away.

### Interface consequences

One small table per interface type, owner-reviewed, in `knowledge/works/interface_consequences.yaml`. It applies when the works touch one side of an interface and the other side exists on the site and is not being replaced. Interface direction follows `knowledge/SCHEMA.md`: `from` acts on `to`. Over-proposal is fixed in this file, not in code.

| Interface type | Works touch | Proposal for the existing side |
| - | - | - |
| loads | `from`: new, replace, upgrade | Investigate `to`: capacity of the existing structure or ground for the new load |
| supplies | `to`: new, upgrade, alter | Investigate `from`: capacity of the existing supply (flow and pressure, electrical, cooling) |
| supplies | `from`: replace, upgrade, alter, remove | Keep `to` working: changeover, temporary supply or shutdown window; retain obligation |
| penetrates | `from`: new, alter, replace | Work item: make good `to` at penetrations (for example fire-stopping) |
| controls | either side | Test the control link at commissioning |
| depends_on | `to`: alter, replace, remove | Investigate `from`: whether what relies on it still holds (for example FRL concessions relying on sprinklers) |
| sequences | either side | Hold point in the package doing the earlier work |
| shares_space | `from`: new, alter | Coordinate set-out with the existing system in that space |
| boundary | either side | The boundary must be assigned to a package (gap check) |

Examples. New rooftop plant on an existing building reaches `if.plant-loads-structure` and proposes a structural assessment. New sprinklers or hydrants reach the fire-water supply and propose a flow and pressure test, which the 0991 project budgeted for. A tenant fit-out should reach base-building central plant and air distribution, but the graph has no such edges yet (see research K1).

Fit-out knowledge to research and draft (unverified until checked): tenant heat loads (occupant density, equipment, lighting) against base-building plant and air-distribution capacity; partitions and ceilings changing air paths, zoning, return air and smoke control; tenant supplementary cooling on the base-building condenser water; BMS and fire-mode integration; glazing films and blinds affecting solar gain and glass thermal stress; energy-efficiency provisions that apply to the new work; base-building energy ratings and lease obligations; landlord approvals and nominated contractors for base-building systems.

### Consequences (`cq.*`)

For what interfaces cannot express, chiefly regulatory triggers on existing buildings. Illustrative shape:

```yaml
- id: cq.pre-2004-fabric-hazardous-materials-survey
  when:
    works: {action: [alter, replace, upgrade, repair, remove]}
    all: [{det: construction_year, lt: 2004}]
  propose:
    - {kind: investigation, label: Hazardous materials survey of areas affected by the works}
  governed_by: []          # rule written and verified in K3
  severity: life-safety
  status: draft
  sources: [...]
```

Proposal kinds: investigation (a work item), discipline (a package suggestion), approval or hold point (a delivery item), obligation (package scope). Drafts to research, all `clause_verified: false` until read in the instrument: whether an alteration in NSW lets the consent authority require the existing building's fire safety to be upgraded; hazardous materials surveys before disturbing fabric of older buildings (needs a new determinant, `construction_year`); the NSW Class 2 registered practitioner and regulated design regime for waterproofing, fire safety, structure, enclosure and services work; impairment management and certification when a fire safety system changes.

### Unforeseen conditions (`uc.*`)

The conditions that emerge mid-project on new builds and existing buildings alike: latent and concealed conditions, failed tests of existing systems, authority and utility surprises. Each attaches to the graph (a system, interface or action) and is raised by the work items that make it likely, before tender, so the unforeseen becomes foreseen.

| Category | Examples |
| - | - |
| Ground and site | Rock, groundwater, contaminated or uncontrolled fill, old footings, unknown buried services, archaeological or Aboriginal heritage finds |
| Existing structure | Not as drawn or no drawings, capacity below assumption, concrete spalling, post-tensioned tendons where coring, rot or termite damage |
| Hazardous materials | Asbestos, lead paint, PCBs, synthetic mineral fibre |
| Concealed services and earlier work | Services not on drawings, undersized or corroded pipework, non-compliant earlier work, missing fire separation above ceilings |
| Existing systems on test | Fire, hydraulic or electrical systems failing flow, pressure or insulation tests when connected or altered; legacy BMS or fire panel protocols |
| Authorities and utilities | Network capacity needing a substation or larger connection, changed authority requirements, upgrade triggers, fire brigade requirements |
| Third parties and occupation | Neighbours (dilapidation, access, crane oversail), tenants in a live building, landlord or strata approvals |
| Design and scope | Coordination clashes, gaps between packages, documents behind the works |
| Supply and site operations | Long-lead equipment (switchboards, transformers, lifts, generators, chillers), insolvency, weather, restricted hours |

Illustrative shape:

```yaml
- id: uc.existing-fire-water-fails-flow-test
  category: existing_systems_on_test
  attaches_to: {system: fire-active.fire-water}
  when:
    works: {action: [new, upgrade, alter], system: [fire-active.sprinklers, fire-active.hydrants]}
    all: [{det: existing_building, is: true}]
  signals:                 # Jev nouls on labelled passages, as failure-mode detectors are
    - type: noul
      instructions: Using `text`, does the passage state results of a flow and pressure test of the existing fire water supply?
      criteria:
        "true": It states measured flow or pressure results for the existing supply.
        "false": It does not state test results; a requirement to test is not a result.
      runs_on: [fire-active.fire-water]
  de_risk: {kind: investigation, label: Flow and pressure test of the existing fire water supply before design}
  contract: Provisional sum or separable portion for any supply upgrade.
  effect: [cost, programme, compliance]
  severity: life-safety
  status: draft
  sources: [...]
```

No invented frequencies or cost percentages: a frequency is recorded only where a source states it, with `verified: false`. Unforeseen conditions differ from failure modes: a failure mode is a design or installation pitfall a document can show; an unforeseen condition is a state of the site or building discovered during the works. Existing `fm.*` records stay as they are. Reports use them: the PMP risk section, RFT latent-condition handling and provisional sums, and investigate-before-tender proposals.

New predicate operators for `applies_when`, consequences and unforeseen conditions: `works` (action and system of an in-scope work item) and `system_existing` (on the site and not being replaced). `system_present` keeps its meaning: in the completed building.

## Provenance and assumptions

Record three independent concepts:

- **Origin:** document, user, calculation or assumption. Calculations retain their input origins. A proposal raised by knowledge is a calculation whose method is the record ID and knowledge version.

- **Review status:** proposed, accepted for planning, verified or superseded. Verification requires an actor and supporting basis; user input is not automatically verified.

- **Meaning:** stated condition, requirement/target, allowance or forecast. Keep existing assertion semantics distinct from approval status.

Unknown is an explicit unresolved value, not zero or false. Retain source references, actor, timestamp, rationale, method/library version and dependencies as appropriate. Preserve document revision/hash and source location, not merely a mutable passage ID.

“Suggest starting values” uses approved scope, quality, cost and duration libraries, asks only consequential missing questions, and offers individually editable proposals. Calculated estimates may depend on assumptions and should expose ranges and limitations. Accepted assumptions remain assumptions. Structural adequacy, ground conditions and compliance must not be defaulted as favourable; propose investigations or allowances instead.

New evidence can create a change proposal or conflict but never silently overwrite a human decision. In particular, accepting a planning assumption must not make it eligible for verified regulatory derivations: extend the current user-precedence rules with explicit eligibility checks.

## Cost plan through procurement

Use a stable cost item as the link between early allowance, detailed scope and pricing schedule. Example:

| Package | Cost item | Work item |
| - | - | - |
| Structural consultant | Concept design | (fee) |
| Structural consultant | Approval documentation | (fee) |
| Structural consultant | Construction documentation | (fee) |
| Structural consultant | Construction-stage inspections | (fee) |
| Concrete trade | Footings | new · substructure.footings · Building A |
| Concrete trade | Ground slab | new · substructure.slab-on-ground · Building A |
| Concrete trade | Suspended slabs | new · structure.concrete · Building A |

Stages and items are editable suggestions, not universal requirements. Scope items and cost items are distinct: several obligations can be included in one priced item; not every obligation needs a separate fee.

Packages are services (consultant), works (a head contract covering the works, or a trade package covering part) or supply (owner-procured plant or FF&E). A services package can be marked for novation, which splits its stages before and after novation. The procurement route and work type suggest default packages from Clerk's consultant rosters and complexity additions, copied as data and draft.

Start with an overall funding target and broad allowance groups. As scope develops, subdivide allowances into package/stage lines. Convert the parent to a non-posting group and allocate its amount to children in one transaction; leave an explicit unallocated residual or show the changed total. Never add the original allowance and its replacement breakdown together.

Rules:

- Funding target, approved budget, current estimate, tender offer and commitment are different measures. A quote is not an award.

- Sum posting leaf items only. Keep contingency, fees, escalation and exclusions explicit. Unknown amounts stay null and produce an incomplete-total warning.

- Use PostgreSQL decimal amounts, explicit currency, tax basis, quantity units, rate basis and price date. Define rounding centrally; do not use binary floating point for money.

- Works lines name one work item, and through it one system and part, so by-system and by-part totals reconcile exactly. Fee lines name a package and stage; project-wide lines name a category. A systems total covers works lines and shows fees and project-wide lines below it, as an elemental cost plan does. A works line's package is optional until packaging; once set, that package must hold a responsibility for the work item.

- RFP/RFT pricing lines reference stable cost-item IDs. Issued schedules freeze labels, stages and IDs; later returned prices remain separate from internal budgets. Budget disclosure is opt-in.

- Returned tender offers need a tenderer dimension and are not stored as cost values; a later offers table keys package, tenderer and cost item.

- Claimed to date is a user-entered value with an as-of date, because capex cost reports track it monthly. Invoices remain out of this wave.

- The profile displays cost-plan summaries. It does not maintain a competing budget or estimate. PMP shows material budget/forecast variance and coverage; unavailable commitments/actuals remain unavailable.

- Estimates use approved benchmark records with geography, date, quality, inclusions, exclusions and applicability. No suitable basis means ask for an input or retain unknown.

## Proposed schema

Names are provisional for peer review, not migration-ready DDL. All project-owned rows carry `org_id` and `project_id`; site-owned rows carry `org_id` and `site_id`. Use composite foreign keys to enforce both tenant and same-project or same-site ownership; service checks alone are insufficient. Mutable records have a version for concurrent-edit protection.

| Entity | Essential fields and relationships |
| - | - |
| New `sites` | Label, address, lot, version. A project references one site. Migration creates one site per existing project. |
| Parts move to the site | `project_parts` gains `site_id` and becomes site-owned; kinds add `roof` and `plant_area`. |
| Extend `profile_user_values` and `profile_rows` | Origin, review status, meaning and structured provenance; preserve existing keys, bands and null/reset behaviour. Site-level keys (determinants, class, scale, existing systems) are stored against the site part; project-level keys against the project. Rebuild projections rather than treating them as authoritative input. |
| New `profile_planning_values` | Part, registered key, typed value/unit, origin, status, meaning, provenance and version; stores assumptions and calculated values that cannot belong in document-only `profile_facts`. No duplicate ledger totals. |
| Extend `profile_builds` | Monotonic revision and input fingerprint including reading selection, evidence, user/planning values, work items and knowledge versions. |
| New `work_items` | Site part, system ID, action, parent, title, existing condition, target, optional quantity/unit, origin, review status, provenance, version. Stable ID. Replaces scope keys as the store of scope; the scope picker writes here. |
| New `proposal_decisions` | Project, knowledge record ID (interface consequence, `cq.*`, `uc.*`), triggering work item, accepted or dismissed, actor, inputs fingerprint. Accepting creates an ordinary record (work item, package suggestion, delivery item or obligation) that keeps the record ID. |
| `packages` | Kind services/works/supply, discipline/trade ID, novation flag, title, lifecycle status, version; scope and stage boundaries belong to the package. |
| `package_scope_items` | Package, stable ID, optional work item (none for general obligations such as meetings or WHS), role (design, document, supply, install, test, certify, inspect, maintain operation, protect), approved clause ID/version or user text, stage, inclusion/exclusion, deliverable, provenance; explicit links to interfaces and existing source records. |
| `project_delivery_items` | Kind activity/milestone/action/risk/issue/decision/approval, title, owner, baseline/target/forecast/actual dates as applicable, status, as-of date, provenance; optional package and work item links. Kind-specific fields validated in Go. |
| `delivery_dependencies` | Predecessor and successor IDs, supported dependency type and lag; reject cycles. Start with milestone dependencies, not a full scheduling engine. |
| `cost_plans` and `cost_plan_versions` | Project plan; revision, draft/baseline status, currency, price date, cost coverage, funding target and its provenance. Frozen baselines retain approved budgets. |
| `cost_items` and `cost_values` | Stable item identity; versioned parent, code, label, category, work item (works lines), package/stage, posting flag, quantity/unit, rate, amount/range, metric and provenance. Metrics this wave: budget, estimate, commitment, claimed to date. One value per version/item/metric; group values are calculated. |
| Scope and cost links | Explicit scope-item to cost-item join; reference links to existing physical IDs. No duplicated money on links. |
| `reports` and `report_versions` | Kind PMP/RFP/RFT, optional package, draft/issued status, reporting date, previous issue, source revisions, template version, structured sections, protected user edits and immutable issue snapshot. |
| `report_references` | Version-scoped citation ID, basis, source revision/location or calculation/assumption details; linked at section/row/value level. |

Gap check (a read): every accepted physical work item has exactly one works package with install (or supply and install), and, where its action needs design, one services package with design. A missing or duplicated role shows as a gap or overlap.

Knowledge additions: `knowledge/works/actions.yaml`, `knowledge/works/interface_consequences.yaml`, `consequences.yaml` and `unforeseen.yaml` per cluster, the `works` and `system_existing` predicate operators, determinant `construction_year`, and checker support for all of them. `knowledge/SCHEMA.md` documents each before any is written.

Keep reviewed clauses and benchmarks as versioned catalogues loaded at startup, following existing knowledge conventions. Extend validation for references, supported predicates and required provenance. Only reviewed content may establish standard obligations; draft content remains visibly provisional.

Do not create tender comparison, contract, invoice or payment tables in this wave. Preserve stable package, scope and cost IDs so those features can later link without replacing the foundation.

## Refresh and report behaviour

1. A relevant evidence, reading-choice, override, library or project-record change marks affected projections stale. Use explicit domain revision dependencies first; avoid a universal dependency engine.

2. Reuse passage caches. Queue only necessary Jev work through existing jobs; reserve foreground filing capacity.

3. Reconcile profile changes, apply permitted updates and surface conflicts. Coalesce repeat update requests.

4. Assemble the requested report from consistent saved revisions. If reading is pending or failed, clearly offer a draft from the last completed state or wait for refresh; never silently claim freshness.

5. Protect manual edits by stable section/item IDs. Refresh untouched content and present conflicts where the underlying source of a protected edit changed.

6. Issuing records an immutable snapshot with as-of date, sources, calculations, assumptions, line IDs and library versions. Later updates flag drafts, never rewrite issued documents.

Uploaded evidence of submission is not evidence of approval; a drawing received is not a completed design stage. Progress requires an explicit record supported by a user update or appropriate evidence. Store minimal delivery records now rather than inferring progress from upload volume.

## Reports and eventual profile tabs

| Output | Compact content |
| - | - |
| PMP | Definition and quality; delivery/appointments; authorities/approvals; time and cost; material risks (including unforeseen conditions), changes, decisions and next actions |
| RFP | Shared brief; consultant stages/services/deliverables; investigations required; interfaces; dates; fee return and proposal requirements |
| RFT | Shared brief; package inclusions/exclusions, supply/install boundaries and retain obligations; access/sequencing; latent-condition handling and provisional sums; dates; tender return and exact document revisions |

Use tables, target/current/variance comparisons, short action statements and approved prose fragments. Colour-labelled citations: E blue evidence, U teal user input, C purple calculation, A amber assumption. Text labels remain legible without colour. Delivery status is separate from provenance. Keep origin, review status and meaning in the database; show one mark per value on screen. Show material assumptions in the report itself; opening a citation reveals detail. PDF references must remain meaningful without an app session.

Set a minimum font size and render-test the page limit. Essential overflow produces a review prompt or an identified attachment, never silent omission. Detailed trade requirements and conditions remain referenced parts of the tender package.

Later UI grouping: Summary; Systems and scope; Time and cost; Delivery and authorities; Evidence and assumptions. Begin with Summary and Systems only if sufficient. These are projections of shared records; tab design and visual polish follow backend validation.

## Knowledge research

The physical graph largely exists as drafts from the 29 September parallel cluster run (`knowledge/MERGE.md`): 140 interfaces, 172 failure modes and 227 rules, 4 with verified clauses. The works layer does not. Research is build-time only: it produces draft records in `knowledge/` that the checker validates and the owner reviews. At runtime Jev remains the only AI. Parallel agents may do K1 to K4, but only after K0 fixes the record shapes, so their output merges.

| Step | Work | Who | Gate |
| - | - | - | - |
| K0 | Record shapes: actions, interface consequences, `cq.*`, `uc.*`, predicate operators, `construction_year`; `SCHEMA.md` and checker. | Lead session | Checker passes; owner approves shapes |
| K1 | Interface audit for existing buildings and fit-outs, per cluster: confirm direction and endpoints when one side exists; add missing edges (base-building plant to tenancy air conditioning; fit-out heat loads and partitions to air distribution, zoning, smoke control and thermal performance; new-to-existing tie-ins). | Parallel agents, mid-tier model, seeds first | Checker; merge note |
| K2 | Consequences from seeds: renovation, remediation and rectification, remediation due diligence, fire and life safety (existing buildings), setup and commissioning, services guides. | Parallel agents; a smaller model suffices because every record carries a checked seed anchor | Checker; anchors resolve |
| K3 | Existing-building law, NSW first: alterations and fire safety upgrade, essential fire safety measures, Class 2 practitioner regime, hazardous materials in older buildings, home building thresholds, heritage. | Strongest model; primary sources only | `clause_verified: false` until the instrument is read; Australian Standards stay unverified without licensed access; owner review |
| K4 | Unforeseen conditions library across the nine categories, starting from the owner-supplied lists in `docs/unforeseen/` (batch 1: 1,000 rows; batch 2 in progress). Each row is sorted by kind (unforeseen condition, design or coordination error, workmanship defect, process/authority/supply/weather), mapped to system IDs (Jev labelling, reviewed), de-duplicated against the other batch and existing `fm.*`/`if.*` records, then enriched into full records. A coverage ledger gives every row record IDs or a rejection reason, and the checker fails on any unaccounted row. Further sources: seeds, public inquiry and investigation reports, industry and insurer guidance, Australian contract practice on latent conditions. | Parallel agents by category, then one merge | Every entry cites a source; no invented frequencies; checker; owner review |
| K5 | Owner lessons: structured interviews on projects the owner ran (what was found, when, what it cost in time, what would have revealed it earlier). The private corpus holds little delivery history. | Owner with lead session | Recorded as draft records |
| K6 | Walk-through: run 0991 and 0777 work items through the knowledge; compare proposals with the consultants, investigations and tests those projects actually appointed or budgeted. Misses become records; noise is fixed in the knowledge. | Lead session | Owner-set share of proposals kept |

## Implementation sequence and acceptance

| Stage | Deliverable and gate |
| - | - |
| 0 Review | Owner approves this direction; independent reviewer challenges schema, counting rules and scope. Recheck HEAD before coding. |
| 1 Shared facts and sites | Add provenance, assumptions, typed values and revision tracking; sites, parts on the site, site-level and project-level values. Existing profile behaviour, user overrides, reading controls, corpus checks and answer keys still pass. One site per project; no visible change. |
| 2 Works model | Work items from scope, the action question, part-based locations, the gap check, and proposals from interface consequences and K0 to K2 records. New-build answer keys unchanged; owner-drafted work-item keys for 0991 and 0777 (`reviewed: false` until owner review). |
| K Knowledge research | In parallel with stages 1 and 2; K3 to K5 continue through later stages. |
| 3 Commercial spine | Services, works and supply packages; responsibilities; cost lines on work items; allowance subdivision, baseline, claimed to date and sum reconciliation. No procurement pricing copied into a second ledger. |
| 4 First report | Consultant RFP on 0991 from one shared assembler, with approved clauses, stable edits, citations, immutable issue and two-page export. |
| 5 Works RFT, then PMP | Same assembler. RFT carries responsibilities, retain obligations and latent-condition handling; PMP carries risks including unforeseen conditions. |
| 6 Live reporting | Minimal progress, authorities, dates, risks/actions and changes since issue. Cost and time summaries reference their owning records. |
| 7 Interaction refinement | Simple tabs, optimistic edits with rollback, export layout and final latency verification. |

Proposed new budgets, to approve and measure on the target VPS: local edit feedback under 100 ms; saved reads/writes p50 ≤100 ms, p90 ≤250 ms; deterministic assembly from current state p50 ≤300 ms, p90 ≤1 s. Work items and proposals are computed inside the existing code-only rebuild and must keep `profile_rebuild` (p50 100 ms, p90 300 ms on Spec Home), with a 0991 fixture added. The gap check is a read: p50 ≤50 ms, p90 ≤150 ms. Measure export separately after choosing its renderer. These are targets, not measured results. Preserve existing filing p50 ≤1 s/p90 ≤2 s. Normal reads and edits make no Jev call; background reading reports actual progress, not a fabricated ETA.

Adoption measure, beside latency: the user's active minutes from dropping a capex project's documents to an issuable RFP draft, recorded on the first run of 0991. The owner sets the target from that run.

Acceptance projects: 0991 (capex fire services upgrade in an existing building), 0777 (city tower fit-out), Mornington (new house) and Petersham (multi-part new mixed use). Add 0991 and 0777 to `data/eval/profile/manifest.json` by ID only; the corpus stays private and no personal details are recorded. Verify: sparse evidence, conflicting/superseded/skipped sources, assumptions accepted then challenged, mixed physical parts, retained live systems, splitting a coarse work item, a dismissed proposal staying dismissed, a work item with no installer and one with two, tenant fit-out raising base-building capacity, duplicate scope, allowance split and double-count prevention, tax/rounding/null amounts, concurrent edits, Jev failure, restart recovery, wrong-org, wrong-project and wrong-site access, preserved issued snapshots and two-page exports.

Outside this wave: future accounting, complex scheduling, automatic tender issue, tender offers and comparison, invoices, Jev allocation of requirements to packages, a screen for several projects on one site, builder-side subcontracting, LLMs and major UI redesign.

Earlier storage discussion remains a separate deployment prerequisite: keep files outside releases, replicate off-site promptly, align database/file recovery and rehearse restoration. Do not bundle a storage migration into this report wave.

## Peer review brief and source map

Review before implementation: Is the proposed schema the smallest adequate extension? Are the actions complete and their boundaries literal? Is the split between site-level and project-level values right? Are assumed inputs barred from verified compliance conclusions? Do budget and estimate rollups reconcile after subdivision? Can scope, package and systems views avoid double counting? Does the interface-consequence table over-propose, and can unforeseen conditions be ranked so a project sees a short list? Can a report be reproduced after evidence reprocessing? Are approval status, quotation and commitment kept distinct? Are update boundaries and speed gates realistic?

Sources at the reviewed commit:

- [Existing profile schema](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/db/migrations/008_project_profile.sql), [source records](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/db/migrations/009_profile_sources.sql), [reading controls](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/db/migrations/010_profile_reading.sql).

- [Knowledge model](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/knowledge/SCHEMA.md), [knowledge merge](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/knowledge/MERGE.md), [taxonomy](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/knowledge/profile/taxonomy.yaml), [project facts](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/knowledge/profile/project_facts.yaml), [scope defaults](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/knowledge/profile/scope_defaults.yaml).

- [Profile UI](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/web/src/Profile.tsx), [field assembly](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/httpapi/profile.go), [reconciliation](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/profile/reconcile.go), [scope assembly](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/profile/build.go).

- [Transactional profile storage](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/store/profile.go), [cached Jev reading and location question](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/internal/jobs/source_reading.go), [development constraints](https://github.com/bennyclifton-svg/SiteWise/blob/02094db/docs/developing.md).

- Clerk reference data (copy as data, never code): `data/taxonomy/asset-register.json` (existing condition and proposed action), `data/taxonomy/consultant-rosters.json` (default consultants by class, work type and complexity), `data/seed/` guides for renovation, remediation and rectification, remediation due diligence, fire and life safety, and trade interfaces.

- TypeSafe: [fan-out](https://docs.typesafe.ai/patterns/fan-out), [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one), [confidence](https://docs.typesafe.ai/confidence), [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
