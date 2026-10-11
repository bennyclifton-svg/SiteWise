# Engineering services RFP gap analysis

Date: 11 October 2026. Status: draft recommendations for owner review.

The useful contribution of the NSW Health Engineering Services Guideline is stronger attention to how a building is operated, maintained, expanded and handed over. SiteWise already covers most of the general design and procurement process. Improve selected deliverables within the existing services RFPs; do not introduce a healthcare workflow or reproduce the guide as a specification.

This review proposes wording and integration points. It changes no runtime knowledge, RFP templates, project records or appointments.

## Source and comparison basis

Source: [NSW Health Engineering Services Guideline, dated 12 August 2026](<D:/AI Projects/Test Data/Golden Consultant RFP/Sample RFP Submissions/engineering-services-guide.pdf>), 195 PDF pages. Page references below use **printed page numbers**; add five for the PDF viewer page number. The contents and selected substantive sections were reviewed across general principles, electrical, fire, HVAC, hydraulics, ICT, lighting, security, acoustics, controls and lifts. Clinical chapters and hospital performance tables were screened out. Printed page 15 was also visually checked.

The comparison uses the current working-tree versions of:

- [Mechanical services](../../knowledge/rfp/mechanical-services.yaml), [electrical services](../../knowledge/rfp/electrical-services.yaml), [hydraulic services](../../knowledge/rfp/hydraulic-services.yaml) and [fire services](../../knowledge/rfp/fire-services.yaml), with their corresponding prose drafts in `docs/rfp-golden/`.
- [Shared RFP fields](../../knowledge/rfp/skeleton.yaml), [conditional modules](../../knowledge/rfp/modules.yaml) and the [cost-planning RFP](../../knowledge/rfp/cost-planning.yaml).
- [Project details](../../knowledge/profile/project_details.yaml), [project facts](../../knowledge/profile/project_facts.yaml), [works actions](../../knowledge/works/actions.yaml), [interface consequences](../../knowledge/works/interface_consequences.yaml) and relevant building-system knowledge.
- The [knowledge schema](../../knowledge/SCHEMA.md) and [electrical RFP simplification direction](../plans/2026-10-10-electrical-rfp-simplification.md).

“Gap” below means a missing or insufficiently explicit deliverable in these RFPs, not proof that the concept is absent from all SiteWise knowledge. The guide supplies transferable design considerations, not universally applicable legal requirements. Proposed wording is an adaptation for SiteWise, not a quotation or a verified standard.

## Eight worthwhile improvements

### 1 Maintenance and replacement access

**Source:** sections 2.5 and 3.2, pp. 11 and 13–15; lighting maintenance, p. 110.

**Current coverage:** mechanical `concept` already names replacement access. Hydraulic design names access; fire design names maintenance coordination. Electrical building knowledge already has a switchroom replacement-access interface and failure mode. This is a consistency and deliverable gap.

**Add:** a simple maintenance and replacement access drawing or schedule for major plant, coordinated with the architect and structural engineer. Include the removal route, access openings, lifting provision, isolation and operational disruption. Require proposed equipment substitutions to preserve the agreed access outcome.

**Suggested wording:** “For major equipment within the appointment, document routine maintenance access and the replacement route, including openings, lifting provisions and required shutdowns. Coordinate building and structural provisions and identify unresolved constraints before equipment selection is finalised.”

**Home:** concept/design deliverables and substitution review in all four services RFPs. Use existing maintenance-access and plant interfaces. Request additional physical works only where an access problem is actually identified; do not automatically add hatches or lifting beams to every project.

### 2 Future provision separated from installed capacity

**Source:** principles, pp. 5–6; electrical, pp. 20–22; mechanical planning, pp. 41–42; hydraulic future fit-out, p. 70.

**Current coverage:** electrical already rejects assumed spare capacity, mechanical includes loads/hours/reserve, and staged work is supported. An explicit account of what is installed now, reserved spatially, or deferred is weaker.

**Add:** a brief future-provision schedule tied to an actual client expansion plan. Distinguish present demand, evidenced existing capacity, agreed design allowance, reserved routes/space and later plant. Carry accepted provisions into drawings and handover records.

**Suggested wording:** “Identify the agreed future stage and distinguish capacity installed now from space, routes and connection provisions reserved for later work. State the basis of allowances and show how future connections can be constructed.”

**Home:** relevant RFP concept/design deliverables; a short future-works brief in Profile. Do not import the guide's spare-capacity percentages or buy extra installed capacity merely because future expansion is possible.

### 3 Operating cost and practical support

**Source:** pp. 5, 9–11; security recurring costs, p. 122; BMCS upgrade/replacement and support, pp. 146–147 and 152.

**Current coverage:** services RFPs request options and cost inputs; cost planning already has an optional `lifecycle` position. The missing link is a defined set of engineering inputs for comparing those options.

**Add:** for a material system choice, compare capital cost inputs, energy/water use, maintenance, replacement, licences and local support. Disclose proprietary restrictions and dependence on a single supplier. Engineers provide technical assumptions; the appointed cost planner performs the selected financial comparison.

**Suggested wording:** “For the agreed options, provide operating and maintenance assumptions, replacement implications, recurring licence costs and local support constraints. Identify the preferred option and its trade-offs for the selected assessment period.”

**Home:** services concept/options deliverables and cost-planning `lifecycle`. Keep detailed discounted-cost or carbon studies separately appointed. Do not create a new costing engine or mandatory study for small work.

### 4 Continuity requirements across connected services

**Source:** pp. 7–8; HVAC reliability, pp. 44 and 48–50; water supply and isolation, pp. 67–72.

**Current coverage:** `live_operations`, staging, temporary services, electrical critical loads/autonomy, and failure/recovery tests already exist. The gap is a shared operational brief across disciplines, rather than another generic “resilience” checkbox.

**Add:** record what must continue operating, acceptable planned outages, recovery expectations and operator approval arrangements. Ask designers to identify dependent services and common failure points. A backed-up item may also need cooling, controls, communications or water.

**Suggested wording:** “Confirm the functions required during planned shutdowns and credible service failures. Identify dependent supplies and controls, proposed isolation or temporary arrangements, and the agreed recovery sequence.”

**Home:** Profile operational constraints; relevant electrical, mechanical, hydraulic and fire design/interface rows. Use the existing live-site module as a relevance trigger, while recognising that future operational continuity also matters on new buildings. Redundancy and backup duration remain project decisions, never hospital defaults.

### 5 Controls interoperability and usable handover

**Source:** functional integration, p. 6; lighting controls, p. 111; BMCS, pp. 146–152.

**Current coverage:** mechanical `controls` names sequences, points, alarms, metering, programming and licences. Electrical `controls` already names cybersecurity ownership. Handover already includes settings and training. Strengthen compatibility evidence and owner access rather than adding another controls package.

**Add:** define the existing/new system boundary, interface responsibility and compatibility testing; disclose licence/support dependencies; specify the project configuration, backups, exportable data and access rights the operator will receive. Allocate remote access and security arrangements to the appropriate specialist.

**Suggested wording:** “Document existing-system compatibility, interface and commissioning responsibilities, recurring charges and operator access. Specify delivery of the agreed project configurations, backups, editable records and data exports, together with training and support arrangements.”

**Home:** mechanical/electrical controls and selected handover deliverables, with specialist ICT/security boundaries retained. Prefer demonstrable interoperability over a blanket protocol prescription. Do not import hospital networks, monitoring platforms, response times or support periods.

### 6 Asset information developed during design

**Source:** section 3.2.4, p. 15.

**Current coverage:** the RFPs include equipment/system registers and handover information, but a progressive, operator-agreed asset schedule is not consistently an explicit staged deliverable.

**Add:** for projects with meaningful maintainable plant, develop a proportionate asset schedule during design and reconcile it with contractor records at handover. Agree identifiers, locations, system relationships, relevant maintenance information, format and information owner.

**Suggested wording:** “Develop the agreed equipment and asset schedule with design submissions. At the appointed handover stage, review its reconciliation with installed asset records and identify outstanding information.”

**Home:** design and handover deliverables in the services RFPs. The consultant's design schedule and the contractor's installed serial-number/warranty records are different responsibilities. Start with a spreadsheet deliverable; this does not justify building an asset-management product or requiring BIM on every job.

### 7 Commissioning designed in early

**Source:** sections 3.1 and 3.3, pp. 13 and 15; BMCS planning, p. 148.

**Current coverage:** integrated tests, expected/actual results, defects, retests and handover are already strong. Mechanical `seasonal` already addresses deferred tuning; its prose draft also mentions independent commissioning in disbursements. These are not new features.

**Add:** make the design-stage output explicitly address testability: test points, isolation, instrumentation and a commissioning outline. On complex connected systems, identify who coordinates commissioning across packages. Price independent verification where selected.

**Suggested wording:** “Include the test points, access and controls needed to verify design performance. Develop the commissioning outline during design and identify integrated-test leadership, acceptance evidence and deferred tests.”

**Home:** existing design/tender deliverables and selected commissioning stage. Designing for testing does not purchase construction attendance, test execution, witnessing or seasonal visits. Avoid a universal independent-commissioning appointment or fixed post-occupancy period.

### 8 One coordinated penetration record

**Source:** fire barrier penetrations, section 3.2.2, p. 14.

**Current coverage:** SiteWise already models penetrations and making good; hydraulic construction support includes fire-stop hold points. A common cross-trade identification and handover record is the more useful addition.

**Add:** where services cross rated construction, name the party coordinating the penetration register, the trades supplying installation records and the appointed reviewer. Agree record format, locations, system references, photographs and unresolved items.

**Suggested wording:** “Coordinate a common identification and record format for service penetrations through rated construction. Name the register coordinator, installation-evidence providers and appointed reviewers; include the completed records and outstanding items in handover.”

**Home:** services interface/construction clauses, coordinated with architecture, fire strategy and the building contractor's scope. Do not assign the fire-services consultant responsibility for every passive-fire installation or imply that document review certifies unseen work.

## Smaller conditional refinements

| Theme and source | Proportionate use in SiteWise |
|---|---|
| Actual operating schedules and demand; pp. 20–21, 45 and 62–63 | Strengthen the mechanical/electrical design basis with occupancy, equipment and operating-hours assumptions. Distinguish client growth provision from repeated conservative allowances. Existing load-calculation scope largely covers this. |
| Passive measures, zoning and part-load operation; pp. 9–10, 43–44 and 62–63 | Use the existing sustainability target and energy-assessor boundary. Ask for coordinated options where relevant; do not impose the guide's energy target or thermal modelling on all work. |
| Water quality, leakage and wet services above sensitive equipment; pp. 67–71 and 84 | Refine source-quality investigation, material suitability, isolation and leakage interfaces where applicable. Existing source/treatment and hydraulic interface scopes provide the home. Do not copy treatment thresholds, dosing or hospital storage assumptions. |
| ICT coverage and migration; pp. 87 and 94–95 | When active ICT/Wi-Fi design is actually selected, define coverage areas, survey/verification and retained-system compatibility. Electrical currently separates passive cabling from specialist active design; preserve that distinction. |
| Lighting quality and acoustic coordination; pp. 106, 110–111 and 125 | Clarify glare, user control, maintenance access and plant-noise interfaces in existing design deliverables. Use project-specific criteria from the appropriate designer; exclude clinical lighting/acoustic tables. |
| Lift operation and maintenance; pp. 155 and 167 | Retain traffic/use briefing, maintenance response and spares as future lift-procurement source material. SiteWise models lifts but has no dedicated lift RFP in the current library; this review does not justify creating one or hiding lift design inside electrical power scope. |

## Keep the project Profile small

Reuse existing building/work type, systems, existing condition, live operations, staged delivery, program and sustainability information. The existing building `design_life` fact is not the design life of every component.

Consider only three additional optional briefing prompts, shown when relevant:

1. **Operation and shutdown constraints:** affected area/function, operating hours, what must continue, acceptable outage and who agrees it.
2. **Future works to allow for:** the intended later use or stage, affected systems/areas and any stated allowance. “Not known” must remain possible.
3. **Operator and handover requirements:** existing controls platform, support constraints and required asset/data formats. Record a brief or reference document rather than demand a full technical schedule up front.

These are candidate additions, not approved fields. Keep their entries scoped to the affected building, area or system. Present an existing project brief where it already answers them. Only structure individual values later if they actually drive scope selection or validation.

Keep physical scope items in the existing action/system model. For example:

- **Replace** the existing hot-water plant with equivalent capacity; **upgrade** it only if increased performance/capacity is intended.
- **Alter** controls to connect new plant to the existing BMS.
- **Retain** the existing supply in operation during staged works.
- **Investigate** the existing switchboard capacity before deciding on an upgrade.
- **New** access opening or lifting provision only when the accepted design requires it.

“Provide an asset schedule”, “compare operating costs” and “witness commissioning” are consultant deliverables, not physical work items. Future provision may be a brief requirement without any current physical work. Do not automatically turn this guide into project scope.

## Integration and priority

Start with maintenance/replacement access, future provision and progressive asset information. Then strengthen controls handover, early commissioning/testability and the shared penetration record. Operational continuity and life-cycle comparison deserve conditional clauses for projects that need them.

Update the affected discipline's source ledger and prose RFP first, then the corresponding `stages`, `positions`, `scope_schedule` or `returnable_notes` in `knowledge/rfp/`. Record the exact guide section/page and separate source evidence from owner-selected procurement policy. Keep additions draft until owner review. Do not place services-specific obligations unconditionally into the shared skeleton.

Reuse existing physical systems and interfaces. New records should address a demonstrated gap, such as an unrecorded controls-access dependency, rather than create a parallel consultant-based knowledge model. The current module file is a relevance catalogue, not itself a ready-made store of shared clause text; inspect the assembler before implementing reusable wording.

Preserve the electrical simplification direction: replace vague wording in existing rows rather than add compound mandatory questions. Ask the consultant for detailed schedules in the return, with material scope allocations settled before issue. Existing project evidence can identify a need; it cannot buy an appointment.

An implementation would be Lane A because RFP scope affects answer trust and fee comparability. Keep existing stage boundaries, unknown/conflicting states, saved wording and immutable issued reports. Include scenarios for simple replacement, staged occupied works, unknown capacity, no BMS, existing BMS, design-only and commissioning-only appointments. Keep the three-page output gate and existing answer keys.

Relevant current p50/p90 budgets are Profile read/edit 50/150 ms, report assembly 300/1000 ms and filing 1000/2000 ms. Template selection should remain deterministic from saved information, with no new hot-path AI call. This review introduces no dependency or runtime change; it does not claim new benchmark results.

## Material to exclude

Exclude clinical ventilation/pressure tables, isolation and operating theatres, renal/CSSD/pathology treatment, medical gases, patient electrical protection, nurse call, patient tracking, clinical radiation shielding, pneumatic tubes, steam sterilisation, hospital disaster classifications and health-specific lift arrangements.

Also exclude mandatory dRofus/BIM workflows, NSW Health procurement/maintenance rules, hospital backup durations and spare-capacity percentages, prescribed plant-selection tables, and numeric lighting/acoustic/water criteria. Even broadly relevant topics do not make those numbers universal. Any later regulatory claim must be verified against its own applicable instrument; the guide is insufficient to mark it verified.
