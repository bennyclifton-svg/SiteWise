# Golden consultant RFPs: peer review

Date: 9 October 2026. Status: draft recommendations for owner review.
Scope: the five completed discipline drafts, their ledgers, the shared skeleton,
the golden-set plan, relevant local seed data, and the current SiteWise repository.
This is a content and implementation-readiness review, not implementation or sign-off.

## Assessment

Keep the shared skeleton, discipline-specific stages, explicit allowances,
itemised departures and standard fee returns. These are a strong foundation for
comparable consultant proposals. Keep structural and civil as separate RFPs, as
the owner decided.

The drafts are not yet ready to become general-purpose selection rules. Their
main weakness is not the quality of their wording: it is when each piece of
wording applies. Much of the evidence describes one warehouse extension and one
camp redevelopment. The drafts sometimes turn those project-specific scopes,
exclusions and commercial positions into universal requirements.

The best next pass is a library of concise, approved scope modules selected by
appointment boundary, affected building parts and systems, work type, pathway,
procurement and site conditions. Class is one input. The resulting RFP should
contain only the selected position for this project, not both simple/complex
columns and instructions for the user to resolve them manually.

## What was checked

- Read the [plan](../plans/2026-10-09-consultant-rfp-golden-set-plan.md),
  [skeleton](skeleton.md), all five `rfp.md` files and all five ledgers.
- Read the foundation design and development rules; inspected report assembly,
  snapshots, edits, issue disclosure, PDF limits, package/stage/scope structures,
  profile catalogues and existing report speed budgets.
- Read the local architecture, town-planning, traffic, structural-commercial and
  civil-commercial seeds; relevant residential structural/civil guidance;
  remediation and advisory guidance; mixed-use and selected renovation,
  industrial, institutional and procurement sections. These are guidance,
  not project evidence or verified legal rules.
- Spot-checked original PDFs: PACE architecture printed pages 5-6 visually;
  TTW civil pages 2-5 as text and page 3 visually; Willow Tree fee page 5 and
  Engine Room's structural/civil comparison page 5 as text. This was not a
  complete re-extraction or page-by-page audit of the original corpus.
- Checked TypeSafe documentation and current NSW government guidance for the
  regulated-design trigger and planning-pathway distinctions. The legislation
  endpoint returned 403, so no statutory rule is marked verified by this review.

Services, BCA/certifier, fire, access, geotech, survey, QS, landscape, waste and
arborist are later tranches in the plan, not omissions from the five drafts.
Their responsibilities still need explicit owners at the interfaces below.

## Corrections before encoding the drafts

| Priority | Finding and evidence | Recommended correction |
|---|---|---|
| High | Appointment boundary is missing as a controlling input. Architecture DD and for-construction are `Always`; structural DD/construction documents and civil stages 1-3 are also `Always`. The [architecture seed](../../data/reference/clerk/data/seed/architectural-services-guide.md#appointment-boundary-and-procurement-route) expressly distinguishes planning-only, traditional, D&C with/without novation and ECI. | Record start/end stages, already-completed work, deliverables being bought now and optional later stages per package. Intersect every stage with that boundary. A due-diligence appointment must not acquire construction documentation. |
| High | D&C tender scope falls through a gap. Architecture and structural show tender documents only for Traditional, despite requiring performance/design-intent information for D&C. Architecture combines for-construction output and partial novation documentation in the same row. Civil assumes a 70% tender stage for every route. | Separate traditional tender documentation, D&C performance/design-intent tender documentation, and post-novation completion. Define deliverables and design responsibility at each handover; percentages alone do not define a priceable deliverable. |
| High | `NSW and Class 2, 3 or 9c in any part` is not a sufficient regulated-design rule in architecture/structural, and civil has the same class shortcut. | Use work type, new/existing status, affected part, dates and applicable exclusions, supported by verified rules or recorded professional advice. Current [NSW guidance](https://www.nsw.gov.au/housing-and-construction/compliance-and-regulation/professionals-working-on-regulated-buildings/building-classes-and-roles) says existing Class 3/9c alteration, repair and renovation enters this regime from 1 July 2028 and describes exemptions. Do not turn every Class 3 into the same automatic obligation. |
| High | Simple/complex controls technical responsibilities. Structural shoring, services fixings and miscellaneous metalwork are excluded on simple jobs; a small project can still need them. Civil's complex flag makes a flood-flagged simple branch effectively unreachable. | Select technical items from actual affected systems and risks. Use complexity only to suggest quantities or review depth. An exclusion needs a named alternative responsible party, or an unresolved allocation. |
| High | Planning lists modification and planning-proposal pathways but has no corresponding report variant; CDC is mentioned in the evidence limit but absent from the pathway table. Report, lodgement and determination are `Always`, including advice-only work. | Define complete stage bundles per supported pathway. REF needs its own determining-authority workflow, not automatic DA lodgement wording; see [NSW development without consent](https://planning.nsw.gov.au/the-planning-system/development-assessment/planning-approval-pathways/development-without-consent). Keep pathways visibly unsupported until their bundle is authored and reviewed. |
| High | Missing information has no defined effect on selection. Unknown flood/asset/OSD status could silently remove important scope; empty `[n]` fields cannot support comparable prices. | Distinguish known yes, known no, unknown and conflicting evidence. Unknown material inputs create focused review questions or explicit pricing assumptions; never silently become false, zero, excluded or verified. |
| Medium | Architecture offers tenant fitout/full FF&E in section 3 but excludes tenant-specific interiors and loose furniture in both section-5 columns. Civil includes pavement profiles in B but makes detailed pavement design optional without fixing who owns it. | One canonical position per scope item feeds stages, exclusions and returnables. Resolve design versus documentation versus coordination separately. No item may be simultaneously included and excluded. |
| Medium | The skeleton requires lump sum to a cap; architecture still says hourly capped, followed by rate-times-cap. Across the drafts, exceeding an allowance automatically attracts rates. | Pick an explicit fee basis per row: fixed fee for defined allowance, actual time to a monetary ceiling, or separately priced option. State who approves extra work and when. Correcting a consultant's own error should not be described as an automatic paid variation. Have the client's agreement govern that distinction. |
| Medium | Civil makes MUSIC/NorBE universal. Traffic puts SIDRA and its output in every report, though the survey stage is conditional on modelling. | Separate drainage modelling, water-quality modelling and catchment-specific assessments. Separate parking/access advice, traffic report, surveys and intersection modelling. Trigger each from the brief/authority requirement; do not require a particular model on every small job. |
| Medium | Shared terms are incomplete as a standalone skeleton: several drafts say only `Shared`. Architecture and planning restrict R6 to sections 3-5 even though terms require liability/agreement departures there. | Put complete shared wording in the skeleton. R6 must cover scope, allowances, program, fees and the attached agreement, with clause/item ID, proposed change and cost/time effect. Explain which responses are departures versus mandatory non-compliance. |
| Medium | Insurance limits, ISO certifications, evaluation weights, validity periods, meeting counts and stage durations come from particular proposals or new policy choices. | Label these as editable client procurement defaults with sources. Do not imply that a historical consultant policy is a universal eligibility requirement, market rate or statutory threshold. Distinguish required certification from acceptable equivalent management systems. |
| Medium | The drafts specify three pages; current PDF issue rejects more than two. | Decide the RFP page policy before build. Recommended: short invitation/project summary plus readable scope and returnable schedules with natural pagination. Change RFP-specific limits deliberately; preserve other report kinds' policy. Do not shrink text or drop scope to fit. |

### Specific provenance repairs

1. **Architecture PACE fee transcription:** the ledger says $45,500 excl. GST,
   $50,050 incl. GST and Stage 1 $25,500. The original `Architecture/Arch RFP3.pdf`,
   PDF page 1 / printed page 6, shows $25,000 + $20,000 = $45,000 excl. GST and
   $49,500 incl. GST. Correct this before using the ledger as an answer key.
2. **Planning modification references:** the draft cites #62-63 for modification
   applications, but those rows describe detailed design being N/A and major
   design changes. The post-approval row cites #54, which is expert-witness work.
   Link modifications to the planning seed/practice source rather than these rows.
3. **Civil design-criteria reference:** the draft cites `S p1` without its available
   canonical row #65. Use stable discipline-qualified IDs for every selection.
4. **Inherited source mistakes:** the original Engine Room comparison's page 5
   labels cumulative figures `Total Stage 2`. Preserve original labels and
   anomalies in provenance, but compute normalised totals independently. Do not
   reproduce a source's arithmetic as an application calculation.
5. **Deliberate changes need their own provenance:** architecture includes
   certifier liaison and document-system uploads where cited proposals excluded
   or charged for them; it changes sustainability documentation from exclusions
   to an inclusion for target projects. These can be sensible client decisions,
   but a source exclusion does not itself establish an approved default inclusion.
6. **Scope beyond the evidence:** architecture construction administration and
   post-novation services need explicit seed/practice provenance just as the
   structural draft labels its construction stage. Original precedents did not
   price the full service offered by those rows.

## Seed-backed additions worth making

These are proposed content modules, all draft. Each has an activation condition
and a concrete deliverable so it can be rendered without generated prose.
The cited seeds support scope discovery, not the verification of code numbers.

### Architecture

| Activate when | Add to the selected scope | Source |
|---|---|---|
| Every appointment, scaled to its boundary | Brief gaps and required-input schedule: document, responsible provider, needed-by date and consequence if unavailable. Maintain a design responsibility matrix and decisions/change register. | [Architecture: inception, DD, coordination](../../data/reference/clerk/data/seed/architectural-services-guide.md) |
| D&C tender, with or without novation | Design-intent/performance brief, retained versus contractor design duties, departures review and a handover deliverables register. If novated, identify deed/timing and pre/post fees. | Same seed, appointment boundary and documentation/tender |
| Existing building, occupied work or staged handover | Retain/demolish/alter drawings; investigations required; old-to-new weatherproofing and fire/access interfaces; staged access, temporary service and occupation design input with the operator. | [Renovation](../../data/reference/clerk/data/seed/renovation-guide.md#live-occupancy-and-staging), [remediation](../../data/reference/clerk/data/seed/building-remediation-rectification-guide.md) |
| Mixed-use or base-building/tenant split | Class/use and scope map by part; shared entry/egress, riser, loading, acoustic and fire interface schedule; shell versus fitout deliverables and handover by use. | [Mixed-use](../../data/reference/clerk/data/seed/mixed-use-construction-guide.md) |
| Relevant energy, access, fire or façade specialist input | Incorporate the named specialist's requirements in drawings/specification; record the designer/reviewer/certifier for each interface. Specialist certification can remain excluded while coordination remains included. | Architecture conditional services and common failures |
| Apartment planning design requirements | Price the applicable architectural design-quality/verification submission and review-panel response where confirmed by current planning advice. Do not assume CC certification covers the planning submission. | Architecture pre-DA section |
| Client actually commissions contract administration | Distinguish design support/site observation from Superintendent/contract administration; define claims/variations, PC and defects duties and inspection allowances. | Architecture construction/completion |

Suggested fixed clause: “Coordinate the retained and proposed building at the
interfaces listed in the responsibility schedule. Identify missing investigations
and consultant inputs before the relevant design sign-off. Record unresolved
interfaces and the decision required.”

### Structural

| Activate when | Add to the selected scope | Source |
|---|---|---|
| Structural design commissioned | Design-basis schedule: geotechnical reliance, design life, importance, loads, wind/earthquake inputs, durability and fire-resistance coordination; missing values referred to the responsible engineer. | [Commercial structural seed](../../data/reference/clerk/data/seed/structural-commercial-industrial-guide.md) |
| Class 1/10 footing or framing work | Residential footing/framing/bracing and tie-down module; review site/wind classification and relevant supplier truss/frame design. Avoid warehouse-specific deliverables. | [Residential structural seed, services by stage](../../data/reference/clerk/data/seed/structural-residential.md#services-by-stage) |
| Alteration, extension, remediation or change of load/use | Record adequacy assessment extent, missing drawings, testing/opening-up, limitations and strengthening option. Separate investigation from design where extent is unknown. | Structural evidence sweep; remediation investigation/design basis |
| Excavation, basement, adjoining structures or temporary support | Assign retention, underpinning, temporary works, monitoring and permanent/temporary load interfaces; identify designer and reviewer. Architect demolition drawings do not replace engineering demolition/propping design. | Structural concept and common failures; renovation structural intervention |
| Specialist contractor designs an element | Element-by-element delegated-design schedule, performance criteria, submission/review/acceptance responsibilities and certificate interfaces. | Structural documentation and construction sections |
| Construction services bought or completion evidence needed | Inspection schedule by element/hold point and stage, notification lead time, reinspection basis, certificates and as-built review. A simple-class flag must not silently remove necessary inspection/certification responsibility. | Structural construction/completion and minimum returnables |

Do not convert a count of pours or loads into a Jev arithmetic question. Code
can aggregate known quantities; the engineer confirms the proposed inspection
regime. The RFP must not guarantee certification regardless of inspection evidence.

### Civil

| Activate when | Add to the selected scope | Source |
|---|---|---|
| Drainage work | Confirm discharge point, downstream capacity, easement/access constraints and required approvals. Distinguish retaining existing OSD, modifying it, replacing it and unresolved feasibility. | [Commercial civil evidence/common failures](../../data/reference/clerk/data/seed/civil-commercial-industrial-guide.md), civil ledger #15-17 and #63 |
| Existing drainage/services retained | Brief for CCTV, survey/potholing or testing where needed; identify who commissions and performs investigation, and how findings alter design. | Civil inception; [residential civil](../../data/reference/clerk/data/seed/civil-residential.md) |
| Earthworks with contamination, acid-sulfate soil, groundwater or uncertain reuse | Coordinate geotech/environmental advice, reuse/disposal assumptions and design changes. Do not promise balanced cut/fill independently of material suitability. | Commercial civil evidence sweep; residential civil low-confidence flags |
| Pavement or retaining works | Name pavement designer and geotech input; separate wall geometry/levels from structural design. Define hydraulic-to-civil connection location, levels and design responsibility. | Civil/structural seeds and civil ledger #32, #39, #42, #99, #101, #141 |
| Public road or authority infrastructure affected | Approval/submission schedule, authority designer/coordinator, review rounds and asset handover requirements. SEA is not a substitute for every water/utility approval. | Commercial civil DD/completion; civil ledger #43, #92, #137 |
| Civil completion commissioned | Works-as-executed review, detention certification, maintenance information and support for title/covenant/public-asset handover requirements where applicable. | Commercial and residential civil completion sections |

Keep the stage-by-element fee grid, but omit inactive elements and define whether
later stages use element subtotals or one stage total. The present prose says the
element split applies to stages 1-2 while the grid repeats A-D through stage 5.

### Town planning

| Activate when | Add to the selected scope | Source |
|---|---|---|
| Advice/due diligence only | Decision-focused controls/pathway/risk report, information limitations and recommended investigations; no automatic lodgement or determination stage. | [Advisory guide](../../data/reference/clerk/data/seed/advisory-services-guide.md), [planning inception](../../data/reference/clerk/data/seed/town-planning-guide.md) |
| Design is being tested | Controls compliance matrix with departures and needed decisions; specialist report schedule with owner and required date. | Planning concept and common failures |
| Consent issued or post-approval support bought | Conditions register separating pre-construction, pre-occupation and ongoing actions, responsible consultant and evidence required; consistency review for later design changes. | Planning conditions/DD, tender and completion sections |
| Modification/review, CDC advice, SSD, REF or planning proposal | Separate reviewed deliverable/stage bundle, appropriate authority, inputs and assessment support. Unsupported pathway prompts a focused review; it must not silently fall back to SEE. | Planning activation plus authoritative pathway verification needed |

Panel attendance, submissions response and development-standard variations should
each have their own trigger/count. Heritage alone does not establish that every
panel or submissions service will be required.

### Traffic

| Activate when | Add to the selected scope | Source |
|---|---|---|
| Loading/access assessment | Vehicle schedule including waste, emergency, service and relevant operator vehicles; ramps, gradients, headroom, sight lines and pedestrian conflicts. Coordinate against current architect/civil layouts. | [Traffic evidence sweep/common failures](../../data/reference/clerk/data/seed/traffic-transport-guide.md), [industrial operational brief](../../data/reference/clerk/data/seed/industrial-construction-guide.md) |
| Counts or modelling needed | Survey locations, dates/periods, source-data acceptance, scenarios and modelling outputs. Select representative periods for the use; school-term weekday peaks are not sufficient for every use. | Traffic surveys; industrial/institutional use context; proposed authoring refinement |
| Approved design is developed or changed by contractor | Review ramps/parking/loading against conditions; define relevant road-reserve signage/committee submissions and changes allowed. | Traffic DD/post-contract sections |
| Construction or completion services commissioned | CTMP preparation versus review, traffic-control/road-occupancy responsibilities, final inspection and completed-car-park certification where needed. | Traffic tender and completion sections |

Require a green travel plan from a confirmed requirement, not just a client opt-in
that can accidentally omit a known condition. Road-safety audit commissioning
must identify its scope and suitable independence from the design role.

## Shared modules for work type and use

These extend the five disciplines without multiplying complete templates.

| Driver | Module |
|---|---|
| `new` | Design and approvals only for the systems and appointment stages actually commissioned. |
| `refurb` / `extend` | Existing records, investigations, retain/demolish interfaces, capacity/adequacy checks and temporary arrangements; no assumption that existing records prove current adequacy. |
| `remediation` | Evidence preservation, diagnosis and testing before permanent solution, access/opening-up/reinstatement, trial details, verification/monitoring and residual limitations. Building defects and contaminated-land remediation remain separate scopes. |
| `advisory` | Decision to support, reliance, evidence cutoff, inspection limits, findings/options and next-step recommendations; downstream design services only as explicit options. |
| Mixed uses | Per-part classes/uses/quantities and shared-system interfaces, with deduplication of genuinely shared work. |
| Live institutional or industrial site | Operator-approved access, possessions/shutdowns, temporary services, staging and handback design inputs. Use-specific needs come from the operational brief; hospital, school, warehouse and laboratory are not interchangeable. |

Sources: [remediation](../../data/reference/clerk/data/seed/building-remediation-rectification-guide.md),
[advisory](../../data/reference/clerk/data/seed/advisory-services-guide.md),
[mixed-use](../../data/reference/clerk/data/seed/mixed-use-construction-guide.md),
[institution](../../data/reference/clerk/data/seed/institution-construction-guide.md),
[industrial](../../data/reference/clerk/data/seed/industrial-construction-guide.md).

## Preliminary drivers table against today's repository

This advances the plan's driver inventory for tranche 1. It is not a final schema
or a claim that every existing profile field is populated on every project.

| Driver | Existing basis | Gap or necessary treatment |
|---|---|---|
| Building category, subtype and work type | `knowledge/profile/taxonomy.yaml`: `hdr.building_class`, `hdr.subclass`, `hdr.work_type`; five work types already exist | Building category such as `industrial` is distinct from NCC class. Add explicit mapping from work type to RFP modules. |
| NCC class and project parts | `knowledge/determinants.yaml` `ncc_class`; profile parts; report snapshots retain part IDs/labels | Evaluate affected parts and shared systems; do not choose one dominant class or infer legal classification from a marketing label. |
| Planning pathway | `hdr.cond.planning`: exempt, CDC, DA, SSD, planning permit | REF, modification and planning proposal are not these existing choices. Need reviewed support or explicit unsupported state. |
| Procurement route | `hdr.cond.procurement_route`: traditional, D&C, ECI, managing contractor, alliance, PPP | RFP's CM wording is not an exact existing enum. Do not conflate routes. Label a proposed D&C default as an assumption until confirmed for issue. |
| Novation and stage position | Package `Novation`, stage `NovationPhase` already exist | Need appointment boundary, handover milestone, client-side versus contractor-side outputs, optional stages and already-completed stages. |
| Stage taxonomy | `knowledge/works/stages.yaml` contains a short general stage tree | Golden discipline stages need explicit stable keys/mapping. Current `DefaultStages` applies catalogue substages to service packages generally. |
| Occupancy, access and stakeholders | Taxonomy operational/access/stakeholder conditions | Enough to trigger a review/module; not enough to invent shutdown dates, access windows or stage counts. |
| Site/geotechnical/environmental facts | Determinants include heritage, groundwater, excavation, flood planning level, contamination, soil, wind and related facts | Site flags are not uniformly available as the draft's desired booleans. A stated flood level is not proof of a complete flood-scope decision. |
| Asset proximity, road authority, catchment, survey/modelling locations, design vehicles, OSD strategy | Some relevant work/system evidence may exist | No complete typed RFP driver set found in inspected profile/package catalogues. Capture only the missing inputs that affect selected scope. |
| Quantities | Subtype scale fields and profile/planning values | Preserve part/use, existing/proposed basis, units, revision and source. Do not sum incompatible areas or reuse disputed quantities. |
| Program | Delivery items/dependencies; included in report snapshot | Map RFP stages to program items; distinguish consultant effort, elapsed authority time and required-input dates. Historical SEA durations remain labelled estimates, not current service promises. |
| Budget | Commercial snapshot and existing disclosure control | Separate internal selection input from publication. A complexity threshold must not force disclosure in the project table. Existing redaction targets `cost:line:` blocks; a new brief budget field requires explicit handling. |
| Other consultants | Packages have lifecycle status and scope | A proposed/planned package is not an appointment. Print `appointed`, `to be appointed` or `unallocated` accurately; route exclusions to actual responsibility. |
| Allowances and fee basis | Free wording/deliverable and cost mechanisms exist | No typed general RFP allowance model in `ScopeContent`. Need units, quantity, fee basis, approval rule and stable pricing-row identity. |
| I / O / X | `ValidateScope` currently accepts only included/excluded | Optional must not be counted as accepted responsibility or rolled into the base fee. Keep unknown and not applicable distinct internally. |
| Selected wording and user edits | Versioned clauses; report block IDs, hashes and protected text edits | Retain this mechanism. Structured positions/counts/stages need protection too; current report edits are text edits, while tables are separate saved data. |

## How to make it dynamic with Jev only

1. **Author once.** Give each scope module a stable ID, version, fixed wording,
   inputs, affected systems/interfaces, stage, output, applicability conditions,
   default position, allowance/fee basis, sources and owner-review status.
   Keep `status: draft` for seed-derived additions. Source lineage and project
   evidence are separate: “this clause came from a proposal” does not mean it
   applies to this project.
2. **Use saved project facts first.** Code resolves exact known drivers and
   approved lookups. Jev interprets relevant document evidence or selects among
   code-parsed candidates only when judgement is needed. Keep the state small;
   do not send the entire seed library on each RFP request. This follows the
   repository foundation and [pre-parsed extraction pattern](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
3. **One fan-out for a state, preferably before opening the RFP.** Ask atomic
   questions together, including relevant speculative questions; code chooses
   applicable results. No serial model chain or prose generation during assembly.
   See [TypeSafe fan-out](https://docs.typesafe.ai/patterns/fan-out).
4. **Treat uncertainty explicitly.** Use per-question thresholds evaluated on
   answer keys. Noul supplies probability, not a separate confidence value;
   Choice has confidence that must be interpreted for that question. A timeout,
   unknown or conflict becomes a review item, not an exclusion. See
   [confidence](https://docs.typesafe.ai/confidence) and
   [confidence routing](https://docs.typesafe.ai/patterns/confidence-routing).
5. **Code assembles and checks consistency.** Compose selected fixed clauses;
   substitute typed values; compute counts, prices and dates; enforce dependencies;
   deduplicate shared scope; compare exclusions to other package responsibilities.
   Jev does not calculate thresholds or determine engineering adequacy. See
   [Jev limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
6. **Make review proportional to uncertainty.** Show selected scope immediately;
   group the few unresolved high-impact decisions first. Each decision explains
   which clauses change. Keep detail expandable. Accepted source changes produce
   a diff; user text and structured choices survive regeneration, with conflicts
   shown if no longer applicable.
7. **Freeze the issued basis.** Preserve the report's existing immutable issue
   model with exact clause, driver, allowance, document and agreement revisions.
   Pin the model version through the existing client; do not add another AI
   provider or client. [TypeSafe API](https://docs.typesafe.ai/api).

Example fixed wording for an unknown drainage strategy:

> Assess the feasibility of retaining the existing detention system using the
> listed records and agreed investigations. Report capacity, access and approval
> constraints. Price replacement design separately as an option pending that
> assessment.

The user then confirms the investigation scope and allowance instead of writing
a drainage brief from scratch. Unknown conditions remain visible and priceable.
This also avoids interpreting the skeleton's “no provisional sums” as permission
to invent a fixed quantity for genuinely uninvestigated work.

## Shared returnable refinements

- R1: stable stage/item IDs; base fee, options and approved allowances separated;
  GST basis; disbursements; no double-counting an option appearing in two sections.
  Internal IDs need not be printed as technical codes.
- R2: deliverable, format, stage, review rounds, responsible preparer and recipient;
  BIM/model exchange requirements based on appointment needs rather than class
  alone. Clarify whether editable source files are included and permitted reliance.
- R4/R7: named lead and availability by stage; subconsultant roles; input
  dependencies and earliest start. Match resources to the promised program.
- R6: both technical and commercial/agreement departures, with cost/time effects;
  missing agreement attachment is an unresolved input, not implicit acceptance.
- Add an acknowledgement inside the existing returnables for issued document
  revisions/addenda and authorised offer acceptance. The procurement seeds
  expressly describe numbered addenda and acknowledgement; no new tenth schedule
  is necessary.
- Define written authority for scope changes, allowance exceedance and fee
  changes through the client's agreement. Stage sign-off is a baseline, not a
  blanket entitlement to charge for all subsequent design development.

## Repository fit and future validation

Build on `internal/reports`, `internal/procurement`, existing `knowledge/reports`
and the profile/works/delivery sources. The current assembler is deterministic
and does not call Jev. `internal/reports/edits.go` already preserves text edits
and surfaces changed/missing source conflicts. `internal/store/report_snapshot.go`
already reads scoped project/package records and source revisions. Preserve
these behaviours instead of introducing a separate document generator.

Current gates in `bench/budgets.json` are:

| Path | p50 | p90 |
|---|---:|---:|
| Report read/write | 100 ms | 250 ms |
| Assemble draft | 300 ms | 1,000 ms |
| Issue / export | 100 ms | 250 ms |
| Filing (foundation requirement) | 1,000 ms | 2,000 ms |

This review measured no runtime timings and changed no runtime code. These are
existing budgets, not results for the proposed generator. Future benchmarks must
cover populated, large and mixed-use RFPs, including PDF rendering and concurrent
filing, under the repository's existing gate/target-host policy. Do not infer
performance from short empty templates.

Before implementation, author expected selections for at least:

| Scenario | Expected proof |
|---|---|
| Arthur St extension | Existing OSD/asset interface and disputed GFA surfaced; no unsupported strengthening assumption. |
| Yarramundi staged redevelopment | Active civil elements, public road/flood/catchment requirements and quantified meeting/inspection scope. |
| Small Class 1 extension | Appropriate footing/framing and existing-work scope; no automatic warehouse BIM/SIDRA/campus allowances. |
| Warehouse D&C without novation | Client-side tender/performance brief; contractor completion not sold as client consultant firm scope. |
| Planning-only appointment | No automatic DD/construction or unrelated engineering deliverables. |
| Mixed Class 2/6/7a building | Part-specific clauses plus shared interfaces; no duplicate projectwide fees. |
| Existing Class 3/9c repair | Regime determined from verified applicability, not class alone. |
| Live hospital/school work | Correct operational module and operator inputs, without transposing one use's requirements to the other. |
| Advisory or remedial investigation | Explicit limitations/testing/next decision; permanent design not assumed complete. |
| Unknown pathway, missing survey or contradictory facts | Focused review/assumption; no fabricated certainty or blank pricing allowance. |
| User changes count/position/text then regenerates | Every override preserved; affected output and fees consistent; conflicts visible. |
| Wrong organisation, retired source, Jev timeout, oversized report | Existing access/isolation, versioning and failure guarantees retained. |

The answer key should record expected inclusions, exclusions, options, unresolved
decisions, allocation gaps and sources. Judge success by avoidable edits and
material omissions as well as rendering correctness. The plan's “80% right” is
an aspiration until tested on this broader set; agree a review-time target with
the owner using actual example RFPs.

## Addendum: implementation plan discovered during review

The separate [implementation plan](../plans/2026-10-09-consultant-rfp-build.md)
appeared during this review. Its goal, task statement, design decisions and file
map were also read. This is a review of those proposed decisions, not an audit
of every implementation task and not authorization to execute them.

Its pure resolver, saved drivers, structured settings, reuse of report records,
zero Jev calls on RFP paths, draft-content gate and existing speed budgets are
well aligned with SiteWise. It already recognises several repository gaps above.
Amend these decisions before executing it:

- **Section 2.3, unknown flags off:** absence of evidence must remain unknown.
  Unknown asset/flood/OSD facts cannot safely select an exclusion. A checkbox
  alone cannot represent unknown, confirmed absent and conflicting evidence.
- **O3, issue blank allowances:** reject this default for material quantities,
  appointment boundaries and fee assumptions. Confirm a common pricing basis,
  request a separately priced investigation, or explicitly define a consultant
  returnable. A blank consultant fee cell is appropriate; a blank client-defined
  meeting/inspection allowance is not an equivalent kind of blank.
- **Section 2.3, flattened class list:** preserve part and work association.
  A projectwide list of classes plus `top_storey_class` cannot explain which
  class belongs to the affected structure or a shared interface.
- **Section 2.4, entire flag-list override:** changing one flag must not freeze
  the complete flag set and hide later evidence for unrelated flags. Track
  independent overrides and their source changes.
- **Section 2.3, DA default and consultants list:** a default pathway remains
  an assumption; require a decision before issuing material pathway-specific
  scope. Live package records include unappointed packages: do not print all
  of them as directly engaged consultants.
- **Sections 2.1-2.2, thresholds:** distinguish client preferences ($20M scale,
  allowance defaults) from statutory applicability. `verified: false` is useful
  provenance but does not make a faulty class-only DBP predicate fit to issue.
  The example `fitout` value also needs mapping: existing work type is `refurb`.
- **O4, title matching:** use a stable discipline ID. A title fallback can
  suggest a discipline for confirmation; a free-text rename should not silently
  change the selected contract scope.
- **O1, three-page limit:** render the populated complex cases before fixing a
  cap. Three Markdown page headings are not proof that the content and fee grids
  fit three readable PDF pages.
- **Task statement, two fixtures:** retain source-fidelity snapshots, but add
  independent selection answer keys from the scenario matrix above. Matching
  today's golden prose would otherwise faithfully preserve today's errors.
- **Task statement, source history:** the proposals are evidence for the new
  drafts, not proof that three firms have priced these exact newly authored
  RFPs. Describe the lineage accurately.

These corrections fit the proposed deterministic architecture. Missing fields
can initially be RFP settings as that plan proposes; they do not require a new
profile feature, service or AI provider. Shared project facts should eventually
have one confirmed source rather than conflicting copies across five RFPs.

## Recommended next content pass

1. Repair the contradictions and provenance errors above.
2. Make appointment boundary, per-item applicability and unknown handling explicit
   in the shared skeleton; complete the shared terms/returnables there.
3. Add the targeted discipline and work-type modules, keeping each short and
   conditional. Record every proposed default inclusion separately from evidence.
4. Complete the driver table and selection answer keys, then review rendered
   examples with the owner discipline by discipline as the plan requires.
5. Write the implementation plan after that content review. The Reports-to-
   Procurement navigation choice can follow separately; Reports currently also
   hosts PMP, which needs an intentional home if navigation changes.

No application code, catalogue status, original RFP, plan or ledger was changed
by this review. No service, runtime dependency or AI provider was added.
