# RFP templates
Generated on 28 October 2026 from the knowledge files on `main` (8a6a6c8 plus the working tree) by a read-only script. Regenerate rather than hand-edit. Back to [the index](../KNOWLEDGE_INDEX.md).

Condition grammar (`internal/rfp/resolve.go` `eval`, `clause`, `driver`): a `when` is a list of
clauses, **any** clause true makes it true; inside a clause every key must hold. Three-valued: an unknown
fact makes the clause unknown, never false. Keys: `work_type`, `pathway` (= `hdr.cond.planning`),
`procurement` (= `hdr.cond.procurement_route`), `novation` (the package's novation: no/option/yes),
`state` (`det.state`), `regulated_design` (`proj.regulated_design`), `class` (any part's NCC class),
`only_class` (every part's classes within the list), `flag` (any listed flag is yes), `stage` (that stage
is include or option), `stage_included` (include only), `element`, `module`, `scale` (resolved simple or
complex), `budget_over` / `site_area_over` / `gfa_over` / `parking_over` (number strictly greater).

Unknown handling (`unknownState`): with `suggest`, false suggest → exclude, else ask; a conflicting fact →
ask; `if_unknown: option` → priced as an option with the printed assumption; `if_unknown: include` →
included with the assumption; otherwise **ask** (a decision before issue). Stages before the boundary
start are excluded; stages after the boundary end are priced as options. A `default: option` stage is an
option unless `include_when` holds. Position values: `I` included, `O` option, `X` excluded (with the
named `responsible` party), `-` not required by this brief (printed as inactive).

Scale (`resolveScale`): `complex` when any `scale.complex` clause is true, `simple` when all are false,
**assumed simple** (with a check) when any is unknown. Only `service_level` positions follow the scale.

Shared parts from `knowledge/rfp/skeleton.yaml` apply to every template: returnables R1–R9, terms,
fields, evaluation weights (price 40, experience 20, team 20, methodology 10, compliance 10), and the
modules in `knowledge/rfp/modules.yaml`.

## Shared returnables (`knowledge/rfp/skeleton.yaml returnables`)

- **R1 Fee by stage** — Excl. GST, in the table below. Base fee and options are totalled separately; price an item once. Acknowledge each addendum by number and confirm the signatory can bind your firm.
- **R2 Deliverables list** — Each deliverable with its format, stage, review rounds and recipient; whether editable files are included and what reliance others may place on them.
- **R3 Hourly rates** — Excl. GST, for work authorised in writing beyond the allowances.
- **R4 Team** — Named lead, % time by stage, subconsultants; resources must match R7.
- **R5 Experience** — Two or three comparable projects in the last five years, each with a referee.
- **R6 Departures** — Each departure from sections 2–5 or the agreement: clause or item, proposed change, effect on fee and time; or "nil". A general exclusion not listed here is non-compliant.
- **R7 Program** — Weeks by stage, separating your time, authority time and the dates you need inputs.
- **R8 Insurance and compliance** — PI {field:pi} and PL {field:pl} with insurer and expiry; workers compensation; quality, safety and environmental systems (ISO 9001, 45001, 14001 or an equivalent you describe); conflicts of interest.
- **R9 Reimbursables** — "Included in the fee" (preferred), or each item with its rate.

A discipline file's `returnables: {rN: text}` adds discipline-specific text under that shared heading; `returnable_notes` add a note when their `when` holds.

## Modules (`knowledge/rfp/modules.yaml`)

| key | label | active when | if unknown |
|---|---|---|---|
| `existing_building` | Existing building, extension or refurbishment | (work_type∈{refurb,extend,remediation}) OR (flag∈{existing_building}) | row left out, listed under Before issue |
| `live_site` | Live institutional or industrial site | flag∈{live_operations} | row left out, listed under Before issue |
| `remediation` | Remediation or rectification | work_type∈{remediation} | row left out, listed under Before issue |
| `advisory` | Advisory or due-diligence appointment | (work_type∈{advisory}) OR (pathway∈{advice_only}) | row left out, listed under Before issue |
| `mixed_use` | Mixed uses | flag∈{mixed_use} | row left out, listed under Before issue |
| `staged` | Staged or multi-building delivery | (flag∈{staged}) OR (flag∈{multi_building}) | row left out, listed under Before issue |
| `house` | House (Class 1, with any Class 10 outbuilding) | only_class∈{1a,1b,10a,10b} AND class∈{1a,1b} | row left out, listed under Before issue |
| `beyond_house` | Building other than a house | class∈{2,3,4,5,6,7a,7b,8,9a,9b,9c} | include |

## Contents

- [access](#access) — consultant.access
- [arborist](#arborist) — consultant.arborist
- [architecture](#architecture) — consultant.architect
- [bca-certification](#bca-certification) — consultant.bca, consultant.certifier
- [civil](#civil) — consultant.civil
- [cost-planning](#cost-planning) — consultant.quantity_surveyor
- [electrical-services](#electrical-services) — consultant.electrical
- [fire-engineering](#fire-engineering) — consultant.fire_engineer
- [fire-services](#fire-services) — consultant.fire_services
- [geotechnical](#geotechnical) — consultant.geotechnical
- [hydraulic-services](#hydraulic-services) — consultant.hydraulic
- [land-surveying](#land-surveying) — consultant.surveyor
- [landscape-architecture](#landscape-architecture) — consultant.landscape
- [mechanical-services](#mechanical-services) — consultant.mechanical
- [structural](#structural) — consultant.structural
- [town-planning](#town-planning) — consultant.town_planner
- [traffic](#traffic) — consultant.traffic
- [waste-management](#waste-management) — consultant.waste

## access

- File: `knowledge/rfp/access.yaml`; status `draft`, version 2
- Discipline ids: `consultant.access`; title aliases (suggest only): access consultant, accessibility consultant
- Title: Access consulting services
- Evidence limit: Two historic warehouse-extension offers do not establish industrial defaults, whole-project scope, current insurance or statutory requirements.

**Boundary** (first true rule suggests the appointment range):

- work_type∈{advisory} → s1_inception … s8_advisory
- always → s1_inception … s4_approval

**Scale.** Actual access chain, evidence and selected appointment govern scope for every use and work type.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_inception` | S1 Inception / evidence |  | always | ask | include |  |
| `s2_concept` | S2 Concept advice |  | work_type∈{new,refurb,extend,remediation} | ask | include |  |
| `s3_planning` | S3 Planning report |  | work_type∈{new,refurb,extend,remediation} AND flag∈{access_planning} | ask | include |  |
| `s4_approval` | S4 Construction approval |  | work_type∈{new,refurb,extend,remediation} AND flag∈{access_approval} | ask | include |  |
| `s5_tender` | S5 Tender support |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s6_postcontract` | S6 Post-contract review |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s7_completion` | S7 Construction / completion |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s8_advisory` | S8 Advisory / operational |  | always | ask | option; include when work_type∈{advisory} |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `review` | Access strategy and routes | Access review against confirmed basis at appointed stages only | technical | I | always |  |  |
| `existing` | Access strategy and routes | Retained/new work and affected routes; BCA/certifier assess obligations, no automatic whole-building upgrade or exemption | technical | I when (work_type∈{refurb,extend,remediation}) OR (flag∈{existing_building}), else - | always | ask |  |
| `shared_routes` | Access strategy and routes | Mixed uses/buildings/tenancies: continuous routes, shared entries/lifts/amenities; resolve adjoining-property/tenancy reliance | technical | I when flag∈{mixed_use,multi_building,access_shared_routes}, else - | always | ask |  |
| `housing` | Facilities and housing | Housing/SDA targets: named scheme, unit types and assessor; residential label alone does not appoint | technical | I when flag∈{access_housing}, else - | always | ask |  |
| `facilities` | Facilities and housing | Hearing/signage/tactile/lift/facility criteria review; services/AV designs hearing, architect signs, lift specialist lifts | technical | I when flag∈{access_facilities}, else - | always | ask |  |
| `solutions` | Specialist assessments | Performance departure: brief/method/report/revisions per departure; author/independent assessor, fire/egress interfaces and acceptance/completion conditions | technical | O | always |  |  |
| `testing` | Specialist assessments | Luminance/slip tests: competent named provider, installed-product certificates and field-test extent separately priced | technical | O | always |  |  |
| `occupied` | Access strategy and routes | Occupied/staged route-continuity review at appointed stages; repeat audits only within selected visit/stage allowances; contractor methods/safety, operator m… | technical | I when flag∈{live_operations,staged}, else - | always | ask |  |
| `inclusive` | Facilities and housing | Beyond-minimum inclusive/user/operational brief, sensory/cognitive needs and action plan; owner training/ongoing management | technical | I when stage_included∈{s8_advisory}, else O | always |  |  |
| `remediation` | Specialist assessments | Remediation/heritage access options from specialist diagnosis/advice; no structural/fire/conservation/hazardous-material design | technical | O | always |  |  |
| `peer` | Specialist assessments | Independent peer review/expert evidence: independence, reliance and legal instruction; no transfer of design duty | technical | O | always |  |  |

**Allowances**: `coverage` (Coverage, default —, owner client); `reviews` (Reviews / reports, default —, owner client); `meetings` (Meetings / RFIs, default —, owner client); `solutions` (Solution options, default —, owner client); `visits` (Tests / visits, default —, owner client); `duration` (Duration / extras, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price each selected stage/service: basis, quantity, ex GST/GST/total; base/options/alternatives and standalone/package dependency separate. State milestones/validity; acknowledge addenda and authorised signatory.
- Note to R2 when always: Reports/mark-ups/maps/audits/solutions/registers: purpose, extent, revisions, dates, formats/editable files, reliance and draft/final rounds; completion-opinion limits.
- Note to R3 when always: Actual role rates and fixed period; counted additional reviews/RFIs, stage meetings, visits/revisits/tests, solution departures/revisions, travel/delay/cancellation and approval ceiling.
- Note to R4 when always: Lead/reviewer, solution author/independent reviewer, housing assessor and survey/test providers; competence, conflicts, stage resources, substitutions and ISO/equivalent QA; named design/certifier/operator evidence owners.
- Note to R5 when always: Two or three comparable uses/work types, existing conditions, housing/operational targets and occupation stages; actual personal scope, lessons and referees.
- Note to R6 when always: Each sections 2-5/agreement departure or unresolved fact: reason, alternative, fee/time, owner/date/evidence, or nil. Cover meetings, authority scope, tests/solutions, housing, IP/reliance, liability/payment/insurance.
- Note to R7 when always: Input/access owners/dates, start, draft/final reviews, consultations, authority/tender responses, hold points before finishes, final response/end and later options; unavailable-evidence consequences.
- Note to R8 when always: Current PI {field:pi}/PL {field:pl}/workers evidence and agreed duration; claim/aggregate, expiry/retroactivity/exclusions/renewal. Task competence/accreditation and required registration; historic certificates insufficient.
- Note to R9 when always: Testing/survey/opening-up/reinstatement, specialist/authority/equipment/travel/platform costs: included/quantity/rate, payer, GST/markup/prior authority; client-direct evidence owners; no double counting.

**Flags this template tests:** `access_approval`, `access_facilities`, `access_housing`, `access_planning`, `access_shared_routes`, `existing_building`, `live_operations`, `mixed_use`, `multi_building`, `staged`

## arborist

- File: `knowledge/rfp/arborist.yaml`; status `draft`, version 2
- Discipline ids: `consultant.arborist`; title aliases (suggest only): arborist, arboricultural
- Title: Arboricultural consulting services
- Evidence limit: Two historical report offers differ in tree count, access and terms; neither appoints construction or physical tree works.

**Boundary** (first true rule suggests the appointment range):

- work_type∈{advisory} → s1_inventory … s7_advisory
- always → s1_inventory … s3_protection

**Scale.** Actual trees, impacts, evidence and selected services govern every use and work type.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_inventory` | S1 Inventory / condition |  | always | ask | include |  |
| `s2_impact` | S2 Concept / impact |  | flag∈{arborist_tree_impacts} | ask | option; include when flag∈{arborist_impact_report} |  |
| `s3_protection` | S3 Consent / protection |  | flag∈{arborist_tree_impacts} | ask | option; include when flag∈{arborist_protection_plan} |  |
| `s4_tender` | S4 Tender / contractor design |  | flag∈{arborist_tree_impacts} | ask | option |  |
| `s5_construction` | S5 Construction / completion |  | flag∈{arborist_tree_impacts} | ask | option |  |
| `s6_investigation` | S6a Specialist investigation |  | always | ask | option |  |
| `s6_risk` | S6b Formal risk assessment |  | always | ask | option |  |
| `s7_advisory` | S7a Advisory / remediation |  | always | ask | option; include when work_type∈{advisory} |  |
| `s7_monitoring` | S7b Ongoing monitoring |  | always | ask | option |  |
| `s7_peer` | S7c Independent peer review |  | always | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `inventory` | Tree evidence and impacts | Onsite tree survey/condition assessment; method, date and uncertainty stated | technical | I | always |  |  |
| `offsite` | Tree evidence and impacts | Neighbour/street trees; ownership, referrals and access limits; inaccessible trees retained with confidence limits | technical | I when flag∈{arborist_offsite_access}, else I | flag∈{arborist_offsite_trees} | ask |  |
| `heritage` | Tree evidence and impacts | Significant/heritage tree controls and referrals; confirm current authority and permission owner | technical | I when flag∈{heritage}, else - | always | ask |  |
| `impacts` | Tree evidence and impacts | Excavation/demolition/levels/foundations/services impacts and hold points at appointed stages; civil/structure design, surveyor locations | technical | I when stage_included∈{s2_impact,s3_protection}, else O | flag∈{arborist_tree_impacts} | ask |  |
| `pruning` | Recommendations and protection | Visual pruning/removal/clearance recommendations; specification only in appointed S3; tree-work contractor executes/disposes/reinstates | technical | I when flag∈{arborist_pruning}, else - | always | ask |  |
| `supervision` | Recommendations and protection | Tree works/protection supervision at scheduled visits/hold points | technical | I when stage_included∈{s5_construction}, else O | always |  |  |
| `landscape` | Specialist investigations and others | Replacement planting / landscape design | technical | X | always |  | landscape architect |
| `specialists` | Specialist investigations and others | Ecology, bushfire and heritage assessment | technical | X | always |  | appointed ecology, bushfire and heritage specialists |
| `existing` | Tree evidence and impacts | Existing/occupied/staged/remediation baseline, access constraints and temporary-protection advice; S3 plan separate; investigate before root attribution, str… | technical | I when (work_type∈{refurb,extend,remediation}) OR (flag∈{existing_building,live_operations,staged}), else - | always | ask |  |
| `investigation` | Specialist investigations and others | Internal/root/canopy/soil investigation; defined method and limits | technical | I when stage_included∈{s6_investigation}, else O | always |  |  |
| `risk` | Specialist investigations and others | Formal risk assessment; defined method/limits, no freedom-from-failure warranty | technical | I when stage_included∈{s6_risk}, else O | always |  |  |
| `monitoring` | Recommendations and protection | Ongoing monitoring with finite duration, visits and close-out | technical | I when stage_included∈{s7_monitoring}, else O | always |  |  |

**Allowances**: `coverage` (Tree coverage, default —, owner client); `reports` (Reports / rounds, default —, owner client); `investigations` (Investigation / risk, default —, owner client); `visits` (Works / monitoring, default —, owner client); `period` (Period / extras, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price selected stages/services by quantity/basis, ex GST/GST/total; separate base/options/alternatives/dependencies, price once. State milestones/validity, acknowledge addenda and authorised signatory.
- Note to R2 when always: Reports, IDs/registers, plans/CAD, calculations, methods and inspection/completion records: purpose/extent, revisions, formats, recipients/reliance, rounds and dates.
- Note to R3 when always: Role/additional tree/visit/revision/investigation rates; meetings/responses/visits/durations/hold points/rounds and monitoring limits; travel/callout/cancellation and approval before overruns.
- Note to R4 when always: Lead/reviewer, stage resources, competence, investigators/subcontractors/conflicts and tree-contractor affiliations; permission/survey/landscape/civil/structure/utilities interfaces; substitutions and ISO/equivalent QA.
- Note to R5 when always: Two or three comparable tree/works constraints, ownership/access, investigations and construction interfaces; named personnel roles, outputs, lessons and referees.
- Note to R6 when always: Sections 2–5 and agreement departures: reason, alternative, fee/time and decision or nil; unknown/conflicting facts or authorised assumption with owner/date/consequence/review point; name excluded-work parties.
- Note to R7 when always: Input/access providers/dates, stage resources and field/draft/review/final milestones; authority dependencies/hold-point notice. Survey reconciliation, calculation/risk/test methods/limits, urgent escalation and condition closure.
- Note to R8 when always: Current PI {field:pi}; PL {field:pl}; workers cover and required duration. Declare activity/height exclusions and renewals; match offered consulting/investigation; agreement/precedence acceptance or numbered departures.
- Note to R9 when always: Travel/access/permits, survey/locating, testing/lab/equipment, traffic controls, tree contractors/reinstatement and documents: quantity, included/allowance/unit basis, payer/GST/markup/prior approval; no R1 duplication.

**Flags this template tests:** `arborist_impact_report`, `arborist_offsite_access`, `arborist_offsite_trees`, `arborist_protection_plan`, `arborist_pruning`, `arborist_tree_impacts`, `existing_building`, `heritage`, `live_operations`, `staged`

## architecture

- File: `knowledge/rfp/architecture.yaml`; status `draft`, version 2
- Discipline ids: `consultant.architect`; title aliases (suggest only): architect, architectural, architecture
- Title: Architectural services
- Required facts (blocking when missing): budget
- Evidence limit: Three proposals: a masterplan and staged camp (AJC), a fitout (nettletontribe) and the Arthur St warehouse DA (PACE). Construction-phase and post-novation scope comes from practice.

**Boundary** (first true rule suggests the appointment range):

- procurement∈{design_construct} → brief_masterplan … dnc_tender
- (work_type∈{advisory}) OR (pathway∈{advice_only}) → brief_masterplan … concept
- module∈{house} AND procurement∈{traditional,managing_contractor} → brief_masterplan … house_documents
- always → brief_masterplan … interiors

**Scale.** Service levels (BIM detail, renders, specification, typical interiors) are suggested complex for a Class 2-6 or 9 part, a budget over $20 million, or a heritage site. Technical scope does not follow this scale.

Complex when: (class∈{2,3,4,5,6,9a,9b,9c}) OR (budget_over∈{20000000}) OR (flag∈{heritage})

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `brief_masterplan` | Brief and masterplan | concept_design | (flag∈{staged}) OR (flag∈{multi_building}) | option | include |  |
| `concept` | Concept | concept_design | always | ask | include |  |
| `schematic` | Schematic design | concept_design | always | ask | include |  |
| `planning_approval` | Planning approval | approval_documentation | pathway∈{cdc,da,ssd,planning_permit,ref,modification} | ask | include |  |
| `planning_response` | Planning response | approval_documentation | pathway∈{da,ssd,planning_permit,modification} | ask | include |  |
| `design_development` | Design development | construction_documentation | always | ask | include |  |
| `interiors` | Interiors | construction_documentation | (work_type∈{refurb}) OR (class∈{3,5,6,9a,9b,9c}) [module beyond_house] | option | include |  |
| `house_documents` | Tender and construction documents | construction_documentation | procurement∈{traditional,managing_contractor} [module house] | ask | include |  |
| `tender_traditional` | Tender documents | construction_documentation | procurement∈{traditional,managing_contractor} [module beyond_house] | ask | include |  |
| `bidding` | Bidding |  | procurement∈{traditional,managing_contractor} | ask | include |  |
| `dnc_tender` | D&C tender documents | approval_documentation | procurement∈{design_construct} | ask | include |  |
| `regulated_design` | Regulated design | construction_documentation | regulated_design∈{yes} | ask | include | state∈{NSW} AND class∈{2,3,9c} |
| `for_construction` | For-construction documents | construction_documentation | procurement∈{traditional,managing_contractor,eci,alliance,ppp} [module beyond_house] | ask | include |  |
| `construction_support` | Design support during construction | construction_stage_inspections | procurement∈{traditional,managing_contractor,eci,alliance,ppp} | ask | option |  |
| `contract_admin` | Contract administration | construction_stage_inspections | flag∈{contract_admin} | option | option |  |
| `post_novation` | Post-novation | construction_documentation | novation∈{option,yes} | ask | include |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `coordination` | Brief and design coordination | Coordinating and integrating consultants' work | service_level | simple I / complex I | always |  |  |
| `input_schedule` | Brief and design coordination | Brief gaps, required-input schedule, responsibility matrix and decisions register | service_level | simple I / complex I | always |  |  |
| `specialist_coordination` | Brief and design coordination | Specialist inputs (energy, access, fire, façade) shown in your documents; designer and certifier noted per interface | technical | I | always |  |  |
| `consultant_selection` | Brief and design coordination | Help selecting other consultants | service_level | simple O / complex O | always |  |  |
| `bim_model` | Design documentation and models | BIM model issued to the design team (level in section 4) | service_level | simple I / complex I | always [module beyond_house] |  |  |
| `bim_coordination` | Design documentation and models | BIM coordination meetings, clash detection | service_level | simple X / complex O | always [module beyond_house] |  | not_required |
| `specification` | Design documentation and models | Architectural specification | service_level | simple I when module∈{house}, else O / complex I | always | option |  |
| `house_interiors` | Design documentation and models | Kitchen, bathroom, laundry and joinery design: layouts, elevations and joinery details | technical | I | always [module house] |  |  |
| `house_selections` | Design documentation and models | Finishes, fixtures, fittings and appliances schedule, internal and external, with colour schemes | technical | I | always [module house] |  |  |
| `door_schedule` | Design documentation and models | Door schedule; hardware schedule by the supplier | technical | I | always |  |  |
| `finishes_schedules` | Design documentation and models | External and base-building finishes schedules | technical | I | always [module beyond_house] |  |  |
| `typical_interiors` | Design documentation and models | Interior layouts for typical rooms | service_level | simple O / complex I | always [module beyond_house] |  |  |
| `base_interiors` | Design documentation and models | Base-building and common-area interiors | technical | I | stage∈{interiors} |  |  |
| `tenant_fitout` | Design documentation and models | Tenant fit-out design | technical | I when work_type∈{refurb}, else O | always [module beyond_house] |  |  |
| `ffe` | Design documentation and models | Loose furniture and FF&E selection | technical | O | always |  |  |
| `marketing_renders` | Design documentation and models | Photoreal renders, marketing CGI, models | service_level | simple X / complex O | always |  | client |
| `value_management` | Procurement and approvals | Value management redesign (at R3 rates) | service_level | simple O / complex O | always |  |  |
| `tender_evaluation` | Procurement and approvals | Tender evaluation input | service_level | simple X / complex O | stage∈{tender_traditional,dnc_tender} |  | client |
| `dnc_performance_brief` | Procurement and approvals | Performance brief; retained versus contractor design schedule | technical | I | stage∈{dnc_tender} |  |  |
| `certifier_liaison` | Procurement and approvals | Liaison with the certifier | technical | I | always |  |  |
| `document_uploads` | Procurement and approvals | Uploading to the client's document system | technical | I | always |  |  |
| `modifications` | Procurement and approvals | Modification applications after approval | technical | O | always |  |  |
| `lodgement` | Procurement and approvals | Lodging the planning application: a DA with the consent authority, or a CDC with the certifier, who assesses it | technical | I when flag∈{architect_lodges}, else X | stage∈{planning_approval} [module house] |  | client |
| `water_asset` | Procurement and approvals | Water authority approval to build over or next to its asset: drawings and submission | technical | O when flag∈{authority_asset}, else - | always [module house] | option |  |
| `court` | Specialist and additional services | Court appeals | technical | X | always |  | client (separate engagement) |
| `community` | Specialist and additional services | Community consultation | technical | O when flag∈{community}, else X | always [module beyond_house] | option | client |
| `country` | Specialist and additional services | Connecting with Country input | technical | I when flag∈{country}, else - | always | option |  |
| `extra_declarations` | Procurement and approvals | Compliance declarations for extra certificates or variations | technical | O when stage∈{regulated_design}, else - | always |  |  |
| `apartment_design` | Procurement and approvals | Apartment design-quality submission and panel response, where planning advice confirms it | technical | O when class∈{2} AND state∈{NSW}, else - | always |  |  |
| `facade_certification` | Specialist and additional services | Façade performance certification (NCC Part F3) | technical | X | always [module beyond_house] |  | façade engineer |
| `performance_solutions` | Specialist and additional services | Performance solutions (fire, access, weatherproofing) | technical | X | always [module beyond_house] |  | fire and access engineers |
| `sustainability_docs` | Specialist and additional services | Green Star or NABERS documentation | technical | I when flag∈{sustainability_target}, else - | always | option |  |
| `signage` | Design documentation and models | Signage and wayfinding | service_level | simple X / complex O | always [module beyond_house] |  | signage designer |
| `outside_boundary` | Specialist and additional services | Work outside the site boundary; external hardstands | technical | X | always [module beyond_house] |  | civil engineer |
| `as_built` | Specialist and additional services | As-built or record drawings | service_level | simple X / complex O | always |  | contractor |
| `others_fees` | Specialist and additional services | Authority and other consultants' fees | technical | X | always |  | client |
| `existing_interfaces` | Brief and design coordination | Existing building: retain, demolish and alter drawings; investigations needed; old-to-new interfaces | technical | I | always [module existing_building] |  |  |
| `live_staging` | Brief and design coordination | Staged access, temporary services and occupation inputs agreed with the operator | technical | I | always [module live_site] |  |  |
| `mixed_use_map` | Brief and design coordination | Class and use map by part; shared-system interface schedule; shell versus fit-out deliverables | technical | I | always [module mixed_use] |  |  |

**Allowances**: `design_meetings` (Design meetings, default 4, owner client); `concept_options` (Concept options, default 2, owner client); `client_presentations` (Client presentations, default —, owner client); `renders` (Working 3D views, default 1, owner client); `issues_per_stage` (Issues per stage, default 1, owner client); `site_visits` (Site visits, design stages, default 2, owner client); `workshops` (Stakeholder workshops, default 1, owner client, when (flag∈{staged}) OR (flag∈{multi_building})); `qs_reviews` (QS cost plan reviews, default 3, owner client); `bim_lod` (BIM level of detail, default 200, owner client); `drawing_sheets` (Drawing sheets at planning, default 12, owner consultant, when stage∈{planning_approval}); `planning_cap` (Planning response ceiling, default —, owner client, when stage∈{planning_response}); `bidding_cap` (Bidding ceiling, default 40, owner client, when stage∈{bidding}); `ca_inspections` (Construction site inspections, default —, owner client, when stage∈{construction_support})

**Discipline returnables** (added under the shared R headings):

- R2: Drawing list: the sheets you expect at planning, tender and construction, with titles.
- R3: Roles: Principal or Director · Associate or Senior architect · Project architect · Architect · Graduate or Technician · Interior designer · Administration.
- R4: Named director, project lead and project architect, with registration numbers.
- R5: Choose projects of the same use and work type.
- Note to R4 when stage∈{regulated_design}: NSW design practitioner registration class for the regulated designs.
- Note to R8 when module∈{house}: PI {field:pi} and PL {field:pl} with insurer and expiry; workers compensation; how you check your documents before issue (no certified system is needed); conflicts of interest.

**Flags this template tests:** `architect_lodges`, `authority_asset`, `community`, `contract_admin`, `country`, `heritage`, `multi_building`, `staged`, `sustainability_target`

## bca-certification

- File: `knowledge/rfp/bca-certification.yaml`; status `draft`, version 2
- Discipline ids: `consultant.bca`, `consultant.certifier`; title aliases (suggest only): bca, certifier, certification
- Title: BCA advisory and building certification services
- Unsupported pathways (blocking check `pathway_unsupported`): cdc, planning_permit, ref, modification, planning_proposal
- Evidence limit: NSW advisory and CC/principal-certifier/occupation scope only. Current appointment, payment and independence requirements require confirmation; no guarantee of approval.

**Boundary** (first true rule suggests the appointment range):

- flag∈{bca_advisory} → a1_inception … a3_tender
- flag∈{bca_certification} → b1_preliminary … b4_occupation
- always → a1_inception … a3_tender

**Scale.** Scope follows the appointed role and agreed assessment extent, never a building-label service default.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `a1_inception` | A1 Inception / schematic advice |  | flag∈{bca_advisory} | ask | include |  |
| `a2_planning` | A2 Planning submission advice |  | flag∈{bca_advisory} AND pathway∈{da,ssd} | ask | include |  |
| `a3_tender` | A3 Detailed design / tender advice |  | flag∈{bca_advisory} | ask | include |  |
| `a4_construction` | A4 Construction advisory support |  | flag∈{bca_advisory} | ask | option |  |
| `b1_preliminary` | B1 Preliminary regulatory assessment |  | flag∈{bca_certification} AND pathway∈{da,ssd} | ask | include |  |
| `b2_construction_approval` | B2 Construction approval assessment |  | flag∈{bca_certification} AND pathway∈{da,ssd} | ask | include |  |
| `b3_construction` | B3 Principal-certifier construction services |  | flag∈{bca_certification} AND pathway∈{da,ssd} | ask | include |  |
| `b4_occupation` | B4 Occupation assessment / handover |  | flag∈{bca_certification} AND pathway∈{da,ssd} | ask | include |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `evidence_registers` | Compliance advice and evidence | Compliance evidence, action and condition registers (PDF/editable), every selected stage | technical | I | always |  |  |
| `statutory_functions` | Statutory assessment and occupation | Statutory/principal-certifier functions: Package B only, separate from A | technical | I when stage_included∈{b1_preliminary,b2_construction_approval,b3_construction,b4_occupation}, else O | flag∈{bca_certification} | ask |  |
| `existing_assessment` | Compliance advice and evidence | Existing/fit-out/extension/changed-use assessment: agreed extent; opening-up/surveys by named others; no whole-building upgrade assumed | technical | I when flag∈{bca_existing_assessment}, else - | always | ask |  |
| `classification_map` | Compliance advice and evidence | Mixed-use/building/tenancy classification and approval map: shared egress, fire/services and shell/tenant boundaries | technical | I when flag∈{bca_classification_map}, else - | always | ask |  |
| `staged_occupation` | Statutory assessment and occupation | Occupied/staged occupation boundaries and certificate packages; designers design temporary measures, contractor sets methods | technical | I when flag∈{bca_staged_occupation}, else - | always | ask |  |
| `performance_assessment` | Statutory assessment and occupation | Performance-solution assessment; specialist author and assessor are different parties | technical | O | always |  |  |
| `fire_referrals` | Statutory assessment and occupation | Fire-system/authority referrals and witnessing; contractor/specialists test and certify | technical | I when flag∈{bca_fire_referrals}, else - | always | ask |  |
| `residential_evidence` | Compliance advice and evidence | Residential energy/design-evidence review (e.g. BASIX), only if required | technical | I when flag∈{bca_residential_evidence}, else - | always | ask |  |
| `specialist_design` | Separate specialist responsibilities | Detailed access/energy/structural/acoustic/waterproofing/passive-fire/bushfire design | technical | X | always |  | separately appointed specialist (name in R4) |
| `other_approvals` | Separate specialist responsibilities | Utility/adjoining-owner/planning/road/demolition approvals | technical | X | always |  | client / project lead (name in R4) |
| `extra_visits` | Statutory assessment and occupation | Extra visits, missed inspections and changed approved design: contingency rates/notice; required statutory action is not optional | technical | O | always |  |  |
| `supervision` | Separate specialist responsibilities | Building supervision, contractual quality, methods and recurring annual inspections; no automatic retainer | technical | X | always |  | builder / contract administrator / owner (name in R4) |

**Allowances**: `plan_sets` (Plan assessments, default —, owner client); `reports` (Reports / checklists, default —, owner client); `meetings` (Meetings, default —, owner client); `rfi_hours` (RFI support, default —, owner client); `solution_versions` (Performance solutions, default —, owner client); `cc_packages` (CC applications, default —, owner client, when stage∈{b2_construction_approval}); `oc_packages` (OC applications, default —, owner client, when stage∈{b4_occupation}); `visits` (Visits, default —, owner client); `witness_events` (Tests / referrals, default —, owner client); `duration_months` (Duration / access, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R3 when always: Named-role/hourly and unit rates: review, meeting, visit, certificate, travel/out-of-hours, cancellation/reinspection, complaints/enforcement and solution versions. Distinguish staging from design change; state ceilings/escalation.
- Note to R8 when always: Current registrations/eligibility, PI/PL limits and exclusions, workers compensation, conflicts and QA (ISO or equivalent). Agree project cover before appointment; no universal statutory insurance limit inferred.
- Note to R1 when always: Separate A/B boundaries, stages/options, payment milestones, authority fees, unit rates, GST, stage dependencies, baseline and cancellation reconciliation.
- Note to R2 when always: Reports/markups, registers, fire schedule, inspection/test records, determinations and handover index: purpose, recipient, formats/editable files, rounds, dates, licence/reliance. Distinguish preparation from assessment.
- Note to R4 when always: Name adviser, certifier, inspectors, reviewer, substitutes and specialists; eligibility/prior involvement, stage availability and R7 resources. Allocate authorship, review, tests and assessment; name excluded parties.
- Note to R5 when always: Two or three comparable appointments with referees: actual use/work type, existing/live constraints, solutions, staging and your role/outcome.
- Note to R6 when always: Include unresolved/conflicting facts, decision owner/date and expressly authorised pricing assumptions with scope/allowance.
- Note to R7 when always: Completeness-review/assessment targets, inspection booking notice, authority dependencies and timing start/pause points; no guaranteed certificate date.

**Flags this template tests:** `bca_advisory`, `bca_certification`, `bca_classification_map`, `bca_existing_assessment`, `bca_fire_referrals`, `bca_residential_evidence`, `bca_staged_occupation`

## civil

- File: `knowledge/rfp/civil.yaml`; status `draft`, version 2
- Discipline ids: `consultant.civil`; title aliases (suggest only): civil
- Title: Civil engineering services
- Evidence limit: Part 1 is the Arthur St warehouse extension; part 2 is the 2024 Camp Yarramundi RFT, whose stage-by-element grid all four tenderers used.

**Boundary** (first true rule suggests the appointment range):

- procurement∈{design_construct} → schematic … dnc_tender
- always → schematic … detailed_tender

**Scale.** Service levels (design criteria document, 3D model, clash check) are suggested complex for a site over 2 ha, works in a public road, a flood-affected site, a drinking water catchment or a Class 3 or 9 part.

Complex when: (site_area_over∈{20000}) OR (flag∈{public_road}) OR (flag∈{flood}) OR (flag∈{norbe}) OR (class∈{3,9a,9b,9c})

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `schematic` | Schematic design for planning approval | concept_design | always | ask | include |  |
| `detailed_tender` | Detailed design for tender | construction_documentation | procurement∈{traditional,managing_contractor} | ask | include |  |
| `dnc_tender` | Design for the D&C tender | approval_documentation | procurement∈{design_construct} | ask | include |  |
| `for_construction` | Final design for construction | construction_documentation | procurement∈{traditional,managing_contractor,eci,alliance,ppp} | ask | include |  |
| `post_novation` | Final design after novation | construction_documentation | novation∈{option,yes} | ask | include |  |
| `construction_phase` | Construction phase | construction_stage_inspections | always | ask | option |  |
| `as_built` | As-built and completion | construction_stage_inspections | always | ask | include; follows construction_phase |  |

**Elements** (priced per element in R1)

| key | code | name | when | if unknown |
|---|---|---|---|---|
| `earthworks` | A | Earthworks and sediment control | always | ask |
| `roads` | B | Roads pavements and levels | always | ask |
| `stormwater` | C | Stormwater | always | ask |
| `public_road` | D | Public road works | flag∈{public_road} | option |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `utility_plan` | Design basis and coordination | Combined utility plan from others' layouts | technical | I | always |  |  |
| `discharge` | Stormwater and water management | Discharge point, downstream capacity, easements and approvals confirmed at schematic design | technical | I | always |  |  |
| `drainage_model` | Stormwater and water management | Stormwater drainage model (DRAINS or equal) | technical | I | always |  |  |
| `hydraulic_connection` | Design basis and coordination | Hydraulic-to-civil connections: location, levels and who designs each side | technical | I | always |  |  |
| `water_quality` | Stormwater and water management | Water quality modelling (MUSIC) and treatment design | technical | I when (flag∈{stormwater_quality}) OR (flag∈{norbe}), else O | always | option |  |
| `norbe` | Stormwater and water management | Neutral or beneficial effect assessment (drinking water catchment) | technical | I when flag∈{norbe}, else - | always | option |  |
| `osd_existing` | Stormwater and water management | Feasibility of keeping or modifying the existing on-site detention | technical | I when flag∈{osd_existing}, else - | always | include |  |
| `osd_replacement` | Stormwater and water management | Replacement on-site detention design, pending the feasibility assessment | technical | O when flag∈{osd_existing}, else - | always | option |  |
| `osd_new` | Stormwater and water management | On-site detention design | technical | I when flag∈{osd_required}, else - | always | option |  |
| `retained_services` | Design basis and coordination | Investigation brief for retained drainage and services (CCTV, survey, potholing) and how findings change the design | technical | I | always [module existing_building] |  |  |
| `earthworks_suitability` | Design basis and coordination | Geotechnical and environmental advice coordinated into earthworks reuse and disposal (balance depends on material suitability) | technical | I when (flag∈{contamination}) OR (flag∈{acid_sulfate}) OR (flag∈{groundwater}), else - | always | include |  |
| `flood_report` | Flood, roads and structures | Flood: floor levels and flood report for the application | technical | I when flag∈{flood}, else - | always | option |  |
| `flood_evacuation` | Flood, roads and structures | Flood evacuation plan | technical | O when flag∈{flood}, else - | always | option |  |
| `pavement_design` | Flood, roads and structures | Detailed pavement design where the geotechnical report does not give it (name the designer in R4) | technical | O | always |  |  |
| `retaining_structural` | Flood, roads and structures | Structural design of retaining walls | technical | X | always |  | structural engineer |
| `public_road_approvals` | Approvals and additional services | Authority submission schedule: designer, review rounds, asset handover | technical | I when flag∈{public_road}, else - | always | option |  |
| `clash_check` | Design basis and coordination | 3D clash check of combined services | service_level | simple X / complex O | always |  | not_required |
| `civil_model` | Design basis and coordination | Coordinated 3D civil model (surface and earthworks) | service_level | simple O / complex I | always |  |  |
| `design_criteria` | Design basis and coordination | Design criteria document | service_level | simple X / complex I | always |  | not_required |
| `water_strategy` | Stormwater and water management | Water management strategy (supply, reuse, effluent) | service_level | simple X / complex O | always |  | hydraulic engineer |
| `safety_in_design` | Design basis and coordination | Civil safety in design report | technical | I | always |  |  |
| `civil_declarations` | Approvals and additional services | Civil design compliance declarations for the regulated designs | technical | I when regulated_design∈{yes}, else - | always; suggest state∈{NSW} AND class∈{2,3,9c} |  |  |
| `quality_audit` | Approvals and additional services | Audit of the contractor's quality system | service_level | simple X / complex O | always |  | superintendent |
| `services_location` | Approvals and additional services | Locating services; designing service relocations | technical | X | always |  | services locator and utility designers |
| `shoring` | Flood, roads and structures | Shoring, rock anchors, slope stability | technical | O when (flag∈{basement}) OR (flag∈{excavation_adjoining}), else X | always | option | structural or geotechnical engineer |
| `trunk_drainage` | Stormwater and water management | Trunk drainage, external catchments, watercourse works | technical | O when flag∈{flood}, else X | always | option | council or a separate engagement |
| `sea` | Approvals and additional services | Authority asset approvals (specialist engineering assessment) | technical | X | always |  | structural engineer |
| `road_safety_audit` | Flood, roads and structures | Road safety audit by an auditor independent of the design | technical | O when flag∈{public_road}, else X | always | option | independent road safety auditor |
| `om_manual` | Stormwater and water management | Operation and maintenance manual for stormwater devices | service_level | simple X / complex O | always |  | contractor |
| `quantities` | Approvals and additional services | Pre-tender estimate or bill of quantities | technical | X | always |  | quantity surveyor |
| `modifications` | Approvals and additional services | Modification applications | technical | X | always |  | town planner |

**Allowances**: `design_meetings` (Design meetings, default 10, owner client); `authority_meetings` (Authority meetings, default 3, owner client); `site_visits` (Site visits during design, default 2, owner client); `layouts_reviewed` (Architect layouts reviewed, default 4, owner client); `concept_options` (Civil concept options, default 2, owner client); `pre_lodgement_revisions` (Drawing revisions before lodgement, default 2, owner client); `authority_requests` (Authority requests in assessment, default 1, owner client); `ca_inspections` (Construction inspections, default 6, owner client, when stage∈{construction_phase}); `rfi_hours` (Construction RFIs, default 10, owner client, when stage∈{construction_phase}); `cc_inputs` (Construction certificate inputs, default 2, owner client, when stage∈{construction_phase})

**Discipline returnables** (added under the shared R headings):

- R1: Meetings are inside the stage prices, not a separate line.
- R2: Drawing list per stage, reports, models (drainage, water quality, 12d or Civil 3D) and certificates.
- R3: Roles: Director · Associate · Senior engineer · Engineer · Designer or drafter · Graduate. Per-meeting and per-visit rates if used.
- R4: Named director and design lead; CPEng or NER; the pavement designer.
- R5: Civil packages for this use and site type, in the same LGA.
- Note to R4 when regulated_design∈{yes}: NSW design practitioner (civil or drainage) registration for the regulated designs.

**Flags this template tests:** `acid_sulfate`, `basement`, `contamination`, `excavation_adjoining`, `flood`, `groundwater`, `norbe`, `osd_existing`, `osd_required`, `public_road`, `stormwater_quality`

## cost-planning

- File: `knowledge/rfp/cost-planning.yaml`; status `draft`, version 2
- Discipline ids: `consultant.quantity_surveyor`; title aliases (suggest only): quantity surveyor, cost planning, cost consultant
- Title: Cost planning and quantity surveying services
- Evidence limit: Historical industrial planning estimates differ in scope, identity, iteration caps and terms; no current cost benchmark or universal CIV requirement established.

**Boundary** (first true rule suggests the appointment range):

- work_type∈{advisory} → s1_basis … s8_advice
- always → s1_basis … s4_design

**Scale.** Purpose, inputs and selected stages govern scope across all building/work types.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_basis` | S1 Inception / cost basis |  | always | ask | include |  |
| `s2_concept` | S2 Feasibility / options |  | work_type∈{new,refurb,extend,remediation} AND flag∈{cost_concept} | ask | include |  |
| `s3_authority` | S3 Authority estimate |  | work_type∈{new,refurb,extend,remediation} AND flag∈{cost_authority} | ask | include |  |
| `s4_design` | S4 Design cost planning |  | work_type∈{new,refurb,extend,remediation} AND flag∈{cost_design} | ask | include |  |
| `s5_tender` | S5 Procurement / tender |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s6_construction` | S6 Post-contract cost advice |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s7_completion` | S7 Final account |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s8_advice` | S8 Standalone advisory |  | work_type∈{advisory} | ask | include |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `construction_plan` | Cost planning and budget | Construction cost plan at selected stages; measured/assumed quantities and missing specialist inputs recorded | technical | I | always |  |  |
| `total_budget` | Cost planning and budget | Total project budget: fees/charges, off-site, FF&E, land/finance/tax; item boundaries and tax adviser agreed | technical | I when flag∈{cost_total_budget}, else - | always | ask |  |
| `contract_allowances` | Cost planning and budget | Contract PC/PS, owner supply and selection dates; no default contract form or contingency percentage | technical | I when flag∈{cost_contract_allowances}, else - | always | ask |  |
| `existing_costs` | Cost planning and budget | Fit-out/refurb/live-site strip-out, make-good, isolation, after-hours and decanting; designers assess retained systems | technical | I when (work_type∈{refurb,extend}) OR (flag∈{live_operations}), else - | always | ask |  |
| `operator` | Cost planning and budget | Operator equipment/specialist operations; evidenced requirements or explicit allowances only | technical | I when flag∈{cost_operator}, else - | always | ask |  |
| `allocation` | Cost planning and budget | Mixed-use/building/stage cost allocation; reconcile shared infrastructure and preliminaries to total | technical | I when flag∈{cost_allocation}, else - | always | ask |  |
| `remediation` | Specialist cost studies | Defect/remediation/contamination costing; specialists diagnose, QS prices defined extent/ranges | technical | I when flag∈{cost_remediation}, else - | always | ask |  |
| `risk` | Cost planning and budget | Escalation, risk register and sensitivity; dated endpoints, no duplicate contingency/margin | technical | I | always |  |  |
| `tender` | Procurement and construction advice | Tender/D&C/ECI review; normalise qualifications, PC/PS and alternatives; no novation or superintendent role | technical | I when stage_included∈{s5_tender}, else O | always |  |  |
| `claims` | Procurement and construction advice | Progress-claim reports; separate brief/inspection counts and administrator authority | technical | I when stage_included∈{s6_construction}, else O | always |  |  |
| `lender` | Procurement and construction advice | Lender reports; separate brief, inspections and reliance, never implied by progress-claim advice | technical | O | always |  |  |
| `valuation` | Specialist cost studies | Valuation, insurance replacement value and tax depreciation | technical | X | always |  | named valuer / insurer / tax adviser (R4) |
| `lifecycle` | Specialist cost studies | Life-cycle/carbon/LCA analysis; horizon/assumptions stated, designers approve performance effects; no compliance certification | technical | O | always |  |  |

**Allowances**: `plans` (Plans / options, default —, owner client); `authority` (Authority reports, default —, owner client, when stage∈{s3_authority}); `review` (Review / meetings, default —, owner client); `visits` (Visits / access, default —, owner client); `tender` (Tender services, default —, owner client, when stage∈{s5_tender}); `construction` (Post-contract / completion, default —, owner client, when stage∈{s6_construction,s7_completion}); `specialist` (Specialist options, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price stages/options once: base/options/alternatives totals, fixed/time-ceiling/unit basis, quantities/GST/payments and dependent/standalone authority fees. Separate already-authorised work/ad hoc allowances. Confirm validity, numbered addenda and signatory authority.
- Note to R2 when always: Each report/BoQ/workbook/register: purpose/scope, input cutoff, format/editability, recipient/reliance, dates/rounds and confidence. Include reconciliation, unpriced gaps and authority versus budget differences.
- Note to R3 when always: Personnel and extra plan/option/update/meeting/visit/tender/claim/variation rates, ceilings, duration extensions/travel/cancellation/escalation; quantity assumptions and authority before exceedance.
- Note to R4 when always: Lead, measurer/estimator, services specialist/reviewer/subconsultants, stage availability/conflicts and ISO or equivalent QA. Assign design/quantity/operator/owner/authority/tax inputs and claim assessor/certifier/payer.
- Note to R5 when always: Two or three matching building/work/purpose/procurement/maturity and existing/live/staging examples; personal role, deliverables, forecast lessons and referees.
- Note to R6 when always: Sections 2–5/agreement: comply/departure or nil; reason, alternative, fee/time effect, owner/date/evidence. Resolve identity/date, meeting/update/rate conflicts and unpriced gaps. Assumptions state extent/quantity, basis and consequence; unknown cost is not zero.
- Note to R7 when always: Start/complete-input conditions, owners/due dates, cutoff/draft/review/final, authority/design/tender milestones, optional reporting cycle and appointment end; pauses/update triggers/consequences.
- Note to R8 when always: Current PI {field:pi}, PL {field:pl}, workers evidence; project-selected limits before issue, claim/aggregate/expiry/exclusions and task-specific competence/registration. Old certificates establish no current cover.
- Note to R9 when always: Travel/site, investigation/data access, authority/lender and specialist costs: included/allowance/unit, quantity/payer/GST/markup/approval. Identify client-direct costs; no R1 or budget duplication.

**Flags this template tests:** `cost_allocation`, `cost_authority`, `cost_concept`, `cost_contract_allowances`, `cost_design`, `cost_operator`, `cost_remediation`, `cost_total_budget`, `live_operations`

## electrical-services

- File: `knowledge/rfp/electrical-services.yaml`; status `draft`, version 3
- Discipline ids: `consultant.electrical`; title aliases (suggest only): electrical services, electrical design
- Title: Electrical services
- Evidence limit: Historical offers do not establish retained capacity, current instruments or accepted appointments.

**Boundary** (first true rule suggests the appointment range):

- (work_type∈{advisory}) OR (pathway∈{advice_only}) → advisory … advisory
- always → inception … design

**Scale.** Actual loads, use, systems and purchased maturity govern scope.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `inception` | S1 Evidence |  | always | ask | include |  |
| `investigation` | S2 Investigation |  | flag∈{elec_investigation} | ask | include |  |
| `concept` | S3 Concept |  | always | ask | include |  |
| `planning` | S4 Authority |  | flag∈{elec_planning} | option | include |  |
| `design` | S5 Design |  | always | ask | include |  |
| `tender` | S6 Tender |  | always | ask | include |  |
| `continuation` | S7 Post-contract |  | always | ask | option |  |
| `construction` | S8 Support |  | always | ask | option |  |
| `handover` | S9 Witness/handover |  | always | ask | option |  |
| `defects` | S10 Defects |  | always | ask | option |  |
| `advisory` | S11 Advisory |  | (work_type∈{advisory}) OR (pathway∈{advice_only}) | ask | option; include when (work_type∈{advisory}) OR (pathway∈{advice_only}) |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `distribution` | Power supply and distribution | Detailed distribution/protection/earthing/containment; fault/voltage/discrimination | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `utility` | Power supply and distribution | Utility demand application; certified utility design separate | technical | I when stage_included∈{concept,planning,design,tender,continuation}, else O | stage∈{concept,planning,design,tender,continuation} |  |  |
| `asp3` | Power supply and distribution | Contestable utility design/works/charges/easements/energisation | technical | X | always |  | ASP3 / utility |
| `lighting` | Lighting and connected systems | Detailed general/external/emergency/exit lighting/controls; not detection | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `feature` | Lighting and connected systems | Feature/sports/process lighting, separate brief | technical | O | stage∈{design,tender,continuation} |  |  |
| `controls` | Lighting and connected systems | Metering/energy/BMS interface; programming/licences/cybersecurity owners | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_controls} | ask |  |
| `generation` | Power supply and distribution | Standby/UPS/storage/PV/EV: capacity/export/islanding/approvals | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_generation} | option |  |
| `ict` | Lighting and connected systems | Passive ICT/security/AV cabling; active/Wi-Fi/carrier/security/AV design separately selected | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_ict} | option |  |
| `dry_fire` | Fire detection and interfaces | Detection/alarm/warning/monitoring, allocation below; no duplicate fees | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_dry_fire} | ask |  |
| `plant` | Power supply and distribution | Plant/lift/pump/specialist power; process design separate | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `existing` | Existing services and project coordination | Existing/live condition/demand/circuit/label/route investigation; continuity and isolation owners | technical | I when stage_included∈{investigation}, else O | stage∈{investigation} AND flag∈{elec_investigation} | ask |  |
| `staging` | Existing services and project coordination | Mixed/staged/shell/fit-out metering, tenancy and energisation gates/tests | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_staging} | ask |  |
| `specialist` | Power supply and distribution | Hazardous/process/HV/critical distribution studies | technical | X | always |  | competent specialist |
| `continuation` | Later services | Continued design; novation only by separate agreed appointment | technical | I when stage_included∈{continuation}, else O | stage∈{continuation} |  |  |
| `construction` | Later services | Quantified construction support | technical | I when stage_included∈{construction}, else O | stage∈{construction} |  |  |
| `handover` | Later services | Selected witnessing/handover; execution by named contractors | technical | I when stage_included∈{handover}, else O | stage∈{handover} |  |  |
| `defects` | Later services | Defects/operational verification | technical | I when stage_included∈{defects}, else O | stage∈{defects} |  |  |
| `studies` | Power supply and distribution | Arc-energy/lightning/surge/power-quality assessments by agreed competent assessor | technical | O | always |  |  |
| `declarations` | Existing services and project coordination | Selected declarations/lodgement; verify jurisdiction/competence | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{elec_declarations} | ask |  |

**Allowances**: `design_meetings` (Design meetings, default 4, owner client, when stage∈{concept,design,tender,continuation}); `design_review_rounds` (Design review rounds, default 1, owner client, when stage∈{concept,design,tender,continuation}); `investigation_visits` (Investigation visits, default 1, owner client, when stage∈{investigation}); `construction_visits` (Construction inspections, default —, owner client, when stage∈{construction})

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Fees/basis/quantities/ceilings by stage/system; ex GST/GST/total, base/options/milestones/dependencies; authorised signatory, addenda/validity; no duplication.
- Note to R2 when always: Section 3 outputs/registers: maturity/formats/native files, recipients/reliance, input revisions/rounds/dates.
- Note to R3 when always: Lead/designer/specialist/ASP3/dry-fire/controls/checker/modeller/inspector/witness rates; extra visit/review/test/retest units/ceilings; travel/out-of-hours/cancellation/escalation.
- Note to R4 when always: Named lead/checker/competence/registrations, stage resources/conflicts/subconsultants, ISO/equivalent QA; return allocation matrix and utility/install/energisation/test/witness/statement owners.
- Note to R5 when always: Two or three comparable use/supply/existing/staged commissions; proposed staff roles/outputs, utility/test lessons and referees.
- Note to R6 when always: Sections 2–5/agreement departures: wording/reason/fee/time; exclusion owner. Resolve conflicts or authorised assumption: owner/date/consequence/verification/review point.
- Note to R7 when always: Inputs/providers/revisions/dates: loads/reserve/utility/interval data/single-lines/tests/fire strategy/MEP/ICT/operator; methods/checks/gates/shutdown/temporary power/test criteria/closure.
- Note to R8 when always: PI {field:pi}; PL {field:pl}; current workers cover/periods/certificates/activity exclusions; agreement/precedence/liability/reliance/IP/termination/novation.
- Note to R9 when always: Utility/ASP3/specialist/logging/scans/openings/access/isolation/temporary power/tests/travel/printing: quantity/payer/markup/GST/approval; no R1/contractor/other-package duplication.

**Flags this template tests:** `elec_controls`, `elec_declarations`, `elec_dry_fire`, `elec_generation`, `elec_ict`, `elec_investigation`, `elec_planning`, `elec_staging`

## fire-engineering

- File: `knowledge/rfp/fire-engineering.yaml`; status `draft`, version 2
- Discipline ids: `consultant.fire_engineer`; title aliases (suggest only): fire engineering, fire engineer, fire safety engineering
- Title: Fire safety engineering
- Evidence limit: Warehouse offers illustrate bounded strategies and differing solution schedules; they establish neither universal solution counts nor a fire-services design appointment.

**Boundary** (first true rule suggests the appointment range):

- (work_type∈{advisory}) OR (pathway∈{advice_only}) → s9_advisory … s9_advisory
- always → s1_strategy … s4_report

**Scale.** Actual issues, evidence and purchased stages govern scope across all building and work types.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_strategy` | S1 Evidence / strategy |  | always | ask | include |  |
| `s2_early` | S2 Early approval support |  | always | ask | option |  |
| `s3_brief` | S3 Performance design brief |  | flag∈{fe_solutions} | ask | include |  |
| `s4_report` | S4 Analysis / FER |  | flag∈{fe_solutions} | ask | include |  |
| `s5_integration` | S5 Tender / integration |  | always | ask | option |  |
| `s6_construction` | S6 Construction support |  | always | ask | option |  |
| `s7_verification` | S7 Verification / occupation |  | always | ask | option |  |
| `s8_specialist` | S8a Specialist investigation |  | always | ask | option |  |
| `s8_staging` | S8b Interim / staged strategy |  | always | ask | option |  |
| `s9_advisory` | S9a Defined advisory |  | always | ask | option; include when (work_type∈{advisory}) OR (pathway∈{advice_only}) |  |
| `s9_peer` | S9b Independent peer review |  | always | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `brief` | Fire strategy and evidence | Brief for scheduled solutions | technical | I when stage_included∈{s3_brief}, else O | flag∈{fe_solutions} | ask |  |
| `fer` | Fire strategy and evidence | FER for scheduled solutions | technical | I when stage_included∈{s4_report}, else O | flag∈{fe_solutions} | ask |  |
| `existing` | Fire strategy and evidence | Scheduled existing FER/evidence reconciliation; BCA/certifier upgrade interface; adjacent buildings by schedule | technical | I when stage_included∈{s1_strategy,s3_brief,s4_report,s5_integration,s9_advisory}, else O | flag∈{fe_existing_reconciliation} | ask |  |
| `interim` | Fire strategy and evidence | Interim strategies/shutdowns/staged occupation; each phase priced separately | technical | I when stage_included∈{s8_staging}, else O | always |  |  |
| `occupants` | Fire strategy and evidence | Mixed/vulnerable/assisted occupants: population/evacuation/staffing basis at purchased stages | technical | I when stage_included∈{s1_strategy,s3_brief,s4_report,s9_advisory}, else O | flag∈{mixed_use,fire_vulnerable_occupants} | ask |  |
| `wet_design` | Design ownership and verification | Sprinklers, hydrants, pumps, tanks and pipework design | technical | X | always |  | fire-services designer |
| `dry_design` | Design ownership and verification | Detection, alarm and warning design | technical | X | always |  | fire-services or electrical designer |
| `strategy_review` | Systems and specialist assessments | Smoke-control/cause-and-effect and selected services strategy review | technical | I when stage_included∈{s4_report,s5_integration}, else O | always |  |  |
| `witnessing` | Design ownership and verification | Selected script review/test witnessing; evidence/defects/retests, trades close defects | technical | I when stage_included∈{s7_verification}, else O | always |  |  |
| `matrix_design` | Design ownership and verification | Matrix design, controls programming and test execution | technical | X | always |  | matrix designer, controls programmer and test contractors |
| `specialist` | Systems and specialist assessments | Scheduled facade/timber/modular/tested-system/structural-fire assessment; insured competence, assembly acceptance checked | technical | I when stage_included∈{s8_specialist}, else O | flag∈{fire_specialist_systems} | ask |  |
| `hazards` | Systems and specialist assessments | Scheduled hazard/storage/EV/battery scenarios; inventory/insurer targets and process specialist | technical | I when stage_included∈{s8_specialist}, else O | flag∈{fire_special_hazards} | ask |  |
| `remediation` | Systems and specialist assessments | Orders/defects: preserve evidence, diagnose extent, temporary controls, insured rectification designer/verifier and closure evidence | technical | O | always |  |  |
| `peer` | Design ownership and verification | Independent peer review; designer retains duty | technical | I when stage_included∈{s9_peer}, else O | always |  |  |
| `other_roles` | Design ownership and verification | Bushfire, legal/expert-witness and statutory BCA certification | technical | X | always |  | bushfire adviser, legal/expert witness and statutory certifier |

**Allowances**: `solutions` (Solution schedule, default —, owner client, when stage∈{s3_brief,s4_report}); `analysis` (Analysis / modelling, default —, owner client, when stage∈{s4_report,s8_specialist}); `issues` (Issues / consultation, default —, owner client); `evidence` (Evidence / access, default —, owner client); `construction` (Tender / construction, default —, owner client, when stage∈{s5_integration,s6_construction}); `verification` (Verification, default —, owner client, when stage∈{s7_verification}); `options` (Other options, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Quantity/basis, ex GST/GST/total by stage/question; base/options/dependencies, milestones/validity; price once. Acknowledge numbered addenda; confirm authorised signatory.
- Note to R2 when always: Section 3 outputs/registers: extent, format/native files, recipients/reliance, input revisions, rounds and dates.
- Note to R3 when always: Lead/reviewer/engineer/modeller/specialist rates; extra solution/scenario/iteration, drawing update/meeting/visit/retest/RFI; ceilings/travel/cancellation/remobilisation.
- Note to R4 when always: Lead/checker/resources, registrations/competence, modelling/peer independence, conflicts/subconsultants and ISO/equivalent QA; name all designer/authority/trade/operator interfaces.
- Note to R5 when always: Two or three comparable use/solution/retrofit/staging commissions: personnel roles, outputs, process/lessons and referees.
- Note to R6 when always: Sections 2–5/agreement departures: reason, alternative, fee/time/decision; unknown/conflicting evidence, priced assumptions with owner/date/consequence/review point; excluded-work parties.
- Note to R7 when always: Methods/scenarios/criteria/checks, input owners/dates, resources, draft/final gates, authority rounds, hold-point/test/retest plan; urgent escalation/operator decisions.
- Note to R8 when always: PI {field:pi}; PL {field:pl}; current workers cover/duration/certificates; facade/material/hazard/modelling exclusions match offered work. Agreement/precedence departures, reliance/liability/inspection terms/novation.
- Note to R9 when always: Travel/out-of-hours, authority/reviewer, surveys/tests/lab/opening/access, models/specialists/printing/reinstatement: quantity/basis, payer/GST/markup/approval; no services/R1 duplication.

**Flags this template tests:** `fe_existing_reconciliation`, `fe_solutions`, `fire_special_hazards`, `fire_specialist_systems`, `fire_vulnerable_occupants`, `mixed_use`

## fire-services

- File: `knowledge/rfp/fire-services.yaml`; status `draft`, version 2
- Discipline ids: `consultant.fire_services`; title aliases (suggest only): fire services, fire protection design, wet fire design
- Title: Fire services
- Evidence limit: Historic concept and combined-services offers establish bounded scope, not system adequacy, current instruments or a fire-engineering appointment.

**Boundary** (first true rule suggests the appointment range):

- (work_type∈{advisory}) OR (pathway∈{advice_only}) → advisory … advisory
- always → inception … design

**Scale.** Actual systems, hazards, evidence and selected stages govern scope.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `inception` | S1 Evidence |  | always | ask | include |  |
| `investigation` | S2 Investigation |  | flag∈{fs_investigation} | ask | include |  |
| `concept` | S3 Concept |  | always | ask | include |  |
| `design` | S4 Developed design |  | always | ask | include |  |
| `tender` | S5 Documentation |  | always | ask | include |  |
| `continuation` | S6 Post-contract |  | always | ask | option |  |
| `construction` | S7 Construction |  | always | ask | option |  |
| `handover` | S8 Commissioning |  | always | ask | option |  |
| `advisory` | S9 Advisory |  | always | ask | option; include when (work_type∈{advisory}) OR (pathway∈{advice_only}) |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `wet` | Water supply and wet fire | Listed sprinklers/hydrants/hose reels/pumps/tanks/boosters by area; concept versus detailed extent | technical | I when stage_included∈{concept,design,tender,continuation}, else O | flag∈{fs_wet} AND stage∈{concept,design,tender,continuation} | ask |  |
| `supply` | Water supply and wet fire | Demand/storage calculations; pressure statement is not installed performance | technical | I when stage_included∈{design,tender,continuation}, else O | flag∈{fs_supply} AND stage∈{design,tender,continuation} | ask |  |
| `authority` | Water supply and wet fire | Listed enquiry/test/meter/backflow/connection/augmentation duties; operator/payer named | technical | I when stage_included∈{concept,design,tender,continuation}, else O | flag∈{fs_authority} AND stage∈{concept,design,tender,continuation} | ask |  |
| `dry` | Detection and systems coordination | Allocated detection/alarms/smoke alarms/monitoring/warning; one designer each | technical | I when stage_included∈{concept,design,tender,continuation}, else O | flag∈{fs_dry} AND stage∈{concept,design,tender,continuation} | ask |  |
| `controls` | Detection and systems coordination | Matrix coordination; name author/programmer/point-checker/test leader/witness | technical | I when stage_included∈{concept,design,tender,continuation}, else O | flag∈{fs_controls} AND stage∈{concept,design,tender,continuation} | ask |  |
| `plant` | Detection and systems coordination | Plant interface review; structural/MEP specialists design supports/enclosure/power/ventilation/drainage/penetrations | technical | I when stage_included∈{concept,design,tender,continuation}, else O | flag∈{fs_plant_interfaces} AND stage∈{concept,design,tender,continuation} | ask |  |
| `existing` | Existing systems and special hazards | Defined investigation; records are not condition proof; upgrade separate | technical | I when stage_included∈{investigation}, else O | flag∈{fs_investigation} AND stage∈{investigation} | ask |  |
| `upgrade` | Existing systems and special hazards | Existing upgrade design subject to investigation/accepted strategy | technical | I when stage_included∈{design,tender,continuation}, else O | flag∈{fs_upgrade} AND stage∈{design,tender,continuation} | ask |  |
| `staging` | Existing systems and special hazards | Live/staged/mixed-use isolation/protection/restoration; contractor/operator controls, fire engineer strategy | technical | O | always |  |  |
| `remediation` | Existing systems and special hazards | Orders/defects: evidence, diagnosis, separate remedy/verification; no whole-building conclusion | technical | O | always |  |  |
| `hazards` | Existing systems and special hazards | Special hazards/high storage/process/batteries/special suppression | technical | X | always |  | hazard / suppression specialist |
| `strategy` | Detection and systems coordination | Fire strategy and performance solutions | technical | X | always |  | fire engineer |
| `declarations` | Declarations and later services | Scoped declarations/lodgement/certification; verify jurisdiction/competence; no approval guarantee | technical | I when stage_included∈{design,tender,continuation,handover}, else O | flag∈{fs_declarations} AND stage∈{design,tender,continuation,handover} | ask |  |
| `continuation` | Declarations and later services | Express continued design/novation | technical | I when stage_included∈{continuation}, else O | always |  |  |
| `construction` | Declarations and later services | Quantified construction support | technical | I when stage_included∈{construction}, else O | always |  |  |
| `witness` | Declarations and later services | Selected test witnessing/handover evidence | technical | I when stage_included∈{handover}, else O | always |  |  |
| `execution` | Declarations and later services | Test execution separately priced to named trades | technical | O | always |  |  |

**Allowances**: `extent` (Extent, default —, owner client); `visits` (Visits/tests, default —, owner client); `design` (Design, default —, owner client); `coordination` (Coordination, default —, owner client); `tender` (Tender, default —, owner client); `construction` (Construction, default —, owner client); `completion` (Completion, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Quantity/basis/ceiling, ex GST/GST/total by stage/system; base/options/dependencies/milestones, validity/addenda; price once. Acknowledge numbered addenda; confirm authorised signatory.
- Note to R2 when always: Section 3 outputs and registers: purpose/maturity, native formats/calculations, recipients/reliance, input revisions/rounds/dates.
- Note to R3 when always: Lead/wet/dry designer, modeller/checker/inspector/specialist rates; extra visit/review/test/witness/retest/package units, ceilings/travel/out-of-hours/cancellation.
- Note to R4 when always: Named lead/checker/resources, competence/registration, conflicts/subconsultants, ISO/equivalent QA; complete allocation schedule with one owner per function.
- Note to R5 when always: Two or three comparable system/use/live-site commissions: staff roles, outputs, interface/test lessons and referees.
- Note to R6 when always: Sections 2–5/agreement departures: alternative/reason/fee/time; exclusions and parties, conflicts, assumptions with owner/date/consequence/review point.
- Note to R7 when always: Inputs/providers/revisions/dates (architecture, hazards, BCA/strategy/approvals, records/supply/MEP); methods/checks/gates, access/outages/operator permissions, hold-point/test plan, urgent findings and closure.
- Note to R8 when always: PI {field:pi}; PL {field:pl}; current workers cover/periods/certificates, activity/hazard exclusions; agreement/precedence, liability/reliance/novation departures.
- Note to R9 when always: Enquiry/tests/authority/connection fees, surveys/openings/specialists/access/traffic/isolation/reinstatement/disposal/travel/printing: quantity/basis/payer/markup/GST/approval; no R1 or other-discipline duplication.

**Flags this template tests:** `fs_authority`, `fs_controls`, `fs_declarations`, `fs_dry`, `fs_investigation`, `fs_plant_interfaces`, `fs_supply`, `fs_upgrade`, `fs_wet`

## geotechnical

- File: `knowledge/rfp/geotechnical.yaml`; status `draft`, version 2
- Discipline ids: `consultant.geotechnical`; title aliases (suggest only): geotechnical, geotech
- Title: Geotechnical services
- Evidence limit: Four historical warehouse-extension offers disagree on quantities and assumptions; other uses follow physical scope and client selections.

**Boundary** (first true rule suggests the appointment range):

- always → g1_plan … g3_report

**Scale.** Investigation follows the actual footprint, loads, disturbance and risk; building use never sets test quantities.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `g1_plan` | G1 Brief / plan |  | always | ask | include |  |
| `g2_field` | G2 Field / laboratory |  | flag∈{geotech_intrusive} | ask | include |  |
| `g3_report` | G3 Interpretation / report |  | always | ask | include |  |
| `g4_planning` | G4 Concept / planning |  | flag∈{geotech_planning} | ask | include |  |
| `g5_design` | G5 Design / tender support |  | always | ask | option |  |
| `g6_construction` | G6 Construction / completion |  | always | ask | option |  |
| `g7_transfer` | G7 Contractor advice / transfer |  | always | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `plan` | Investigation and access | Investigation plan and issues register reconciled to allowances | technical | I | always |  |  |
| `foundation` | Ground design parameters | Foundation/bearing/settlement advice; structural engineer designs | technical | I when flag∈{geotech_foundation}, else - | always | ask |  |
| `civil_parameters` | Ground design parameters | Earthworks/subgrade/pavement/retaining parameters; civil/structural designer designs, no cut/fill balance | technical | I when flag∈{geotech_civil_parameters}, else - | always | ask |  |
| `monitoring` | Ground design parameters | Beyond-spot groundwater monitoring / hydrogeology / dewatering; define objective, duration and specialist | technical | O | always |  |  |
| `asset_exposure` | Investigation and access | Buried tank/OSD/services/asset exposure; no integrity certification; civil capacity/structural adequacy | technical | I when flag∈{geotech_asset_exposure}, else - | always | ask |  |
| `clearance` | Investigation and access | Locating/clearance before intrusive work; name isolation/potholing party | technical | I when flag∈{geotech_intrusive}, else - | always | ask |  |
| `access` | Investigation and access | Occupied access, tenant liaison and permits | technical | X | always |  | client access lead (name in R4) |
| `defects` | Specialist assessment | Defect/distress diagnosis; expert evidence separate, structural engineer designs remedy | technical | O | always |  |  |
| `environment` | Specialist assessment | Contamination, waste classification, remediation/validation and independent audit | technical | X | always |  | separately appointed environmental consultant (name in R4) |
| `ass` | Specialist assessment | Acid sulfate soil assessment; define desktop/intrusive/management package from evidence | technical | I when flag∈{geotech_ass}, else - | always | ask |  |
| `handback` | Investigation and access | Reinstatement, spoil/fluid disposal; name repair, classification, transport and disposal owners | technical | I when flag∈{geotech_handback}, else - | always | ask |  |
| `construction` | Construction and transfer | G6 inspections/tests/scoped completion; contractor books, named others supervise/design temporary works | technical | I when stage_included∈{g6_construction}, else O | always |  |  |
| `transfer` | Construction and transfer | G7 novation/transfer; separate appointment and fee, never inferred from route | technical | I when stage_included∈{g7_transfer}, else O | always |  |  |

**Allowances**: `mobilisation` (Mobilisation, default —, owner client, when stage∈{g2_field}); `boreholes` (Boreholes / cores, default —, owner client, when stage∈{g2_field}); `pits` (Pits / exposures, default —, owner client, when stage∈{g2_field}); `tests` (In-situ / laboratory, default —, owner client, when stage∈{g2_field}); `groundwater` (Groundwater option, default —, owner client); `reports` (Reports / support, default —, owner client); `construction` (Construction option, default —, owner client, when stage∈{g6_construction}); `handback` (Handback / waste, default —, owner client, when stage∈{g2_field})

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price stages/options separately: fixed, time-ceiling or unit basis; quantities, rates, GST, dependencies and markups. Cored alternatives include auger: total and increment, no double count. Acknowledge addenda. Confirm the signatory can bind your firm.
- Note to R2 when always: Plan, logs/location/photos, labs, ground model/parameters, interim/final/monitoring reports and completion records: purpose, recipient, editable formats, dates, rounds, limits/reliance and pending tests.
- Note to R3 when always: Role/equipment and extra metre/hole/test/well/visit rates; minimum hire, travel, standby/cancellation, expedited lab/storage/restoration, ceilings and escalation. Expose markups.
- Note to R4 when always: Name lead, field supervisor, reviewer, environmental specialist, labs/subcontractors, competence and availability. Allocate clearance, sampling, design, disposal, reinstatement and handover; QA ISO or equivalent.
- Note to R5 when always: Two or three comparable investigations with referees: actual ground/excavation, existing/live constraints, scale, decision, role and outcome.
- Note to R6 when always: List scope/agreement departures, alternatives and fee/time effect, or nil. Include conflicting levels, counts/depths, reliance/environmental scope; authorised assumptions need quantity, decision owner/date.
- Note to R7 when always: Input/plan approval, mobilisation, field/lab, preliminary/final report and monitoring endpoint dates; optional construction coverage, booking notice, dependencies/start/pause and interim report limits.
- Note to R8 when always: Client-selected PI {field:pi} and PL {field:pl}. Current PI/PL/workers compensation as applicable, claim/aggregate limits, exclusions, continuity and competence/registration. Agree client-selected insurance limits before issue.
- Note to R9 when always: Labs/plant/locator, permits, water/skips/disposal/restoration, travel, instruments/storage: included/allowance/rate, payer, quantity, markup/GST and approval. No R1 duplication; identify client-direct work.

**Flags this template tests:** `geotech_ass`, `geotech_asset_exposure`, `geotech_civil_parameters`, `geotech_foundation`, `geotech_handback`, `geotech_intrusive`, `geotech_planning`

## hydraulic-services

- File: `knowledge/rfp/hydraulic-services.yaml`; status `draft`, version 3
- Discipline ids: `consultant.hydraulic`; title aliases (suggest only): hydraulic services, hydraulic design
- Title: Hydraulic services
- Evidence limit: Historical offers do not verify current capacity, instruments, insurance or appointment.

**Boundary** (first true rule suggests the appointment range):

- (work_type∈{advisory}) OR (pathway∈{advice_only}) → advisory … advisory
- always → inception … design

**Scale.** Actual systems, sources, affected works and purchased maturity govern scope.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `inception` | S1 Evidence |  | always | ask | include |  |
| `investigation` | S2 Investigation |  | flag∈{hyd_investigation} | ask | include |  |
| `concept` | S3 Concept |  | always | ask | include |  |
| `planning` | S4 Authority |  | flag∈{hyd_planning} | ask | include |  |
| `design` | S5 Design |  | always | ask | include |  |
| `tender` | S6 Tender |  | always | ask | include |  |
| `continuation` | S7 Later design |  | always | ask | option |  |
| `construction` | S8 Support |  | always | ask | option |  |
| `handover` | S9 Witness |  | always | ask | option |  |
| `defects` | S10 Defects |  | always | ask | option |  |
| `advisory` | S11 Advice |  | (work_type∈{advisory,remediation}) OR (pathway∈{advice_only}) | ask | option; include when (work_type∈{advisory}) OR (pathway∈{advice_only}) |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `concept` | Design basis | Concept options and spatial advice | technical | I when stage_included∈{concept}, else O | stage∈{concept} |  |  |
| `existing` | Design basis | Retained-system investigation | technical | I when stage_included∈{investigation}, else O | stage∈{investigation} AND flag∈{hyd_investigation} | ask |  |
| `advice` | Design basis | Diagnosis and advice | technical | I when stage_included∈{advisory}, else O | stage∈{advisory} |  |  |
| `domestic` | Water supply and storage | Design potable and hot water services | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `pumps` | Water supply and storage | Design and select domestic tanks and pumps | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_pumps} | include |  |
| `rainwater` | Water supply and storage | Design rainwater collection and reuse | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_rainwater} | include |  |
| `alternative_sources` | Water supply and storage | Design connections for alternative water supplies | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_alternative_sources} | option |  |
| `sanitary` | Sewerage and trade waste | Design sanitary plumbing and sewer connections | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `treatment` | Sewerage and trade waste | Design onsite sewage treatment | technical | X | stage∈{design,tender,continuation} AND flag∈{hyd_treatment} | ask | treatment designer |
| `trade` | Sewerage and trade waste | Design trade-waste drainage and discharge controls | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_trade} | option |  |
| `process` | Sewerage and trade waste | Design specialist process-waste treatment | technical | X | stage∈{design,tender,continuation} AND flag∈{hyd_process} | ask | process specialist |
| `roof` | Stormwater drainage | Design roof and terrace drainage | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_roof} | include |  |
| `civil` | Stormwater drainage | Design site stormwater and detention | technical | X | stage∈{design,tender,continuation} AND flag∈{hyd_external} | ask | civil engineer |
| `gas` | Gas and specialist fluids | Design gas and specialist-fluid services | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_gas} | option |  |
| `fixtures` | Coordination and approvals | Select plumbing fixtures | technical | X | stage∈{design,tender,continuation} |  | architect |
| `fire` | Coordination and approvals | Design fire-water systems and fire strategy | technical | X | stage∈{design,tender,continuation} |  | fire services consultant |
| `interfaces` | Coordination and approvals | Coordinate hydraulic design with other disciplines | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_interfaces} | include |  |
| `utility` | Coordination and approvals | Coordinate utility connections and approvals | technical | I when stage_included∈{planning,design,tender,continuation}, else O | stage∈{planning,design,tender,continuation} AND flag∈{hyd_utility} | option |  |
| `staging` | Coordination and approvals | Design staged and temporary hydraulic services | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_staging} | option |  |
| `declarations` | Coordination and approvals | Prepare hydraulic design declarations | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{hyd_declarations} | ask |  |
| `continuation` | Later services | Later design and novation | technical | I when stage_included∈{continuation}, else O | stage∈{continuation} |  |  |
| `construction` | Later services | Review construction and respond to site queries | technical | I when stage_included∈{construction}, else O | stage∈{construction} |  |  |
| `witness` | Later services | Testing review and handover | technical | I when stage_included∈{handover}, else O | stage∈{handover} |  |  |
| `defects` | Later services | Defects and operational verification | technical | I when stage_included∈{defects}, else O | stage∈{defects} |  |  |

**Allowances**: `extent` (Extent, default —, owner client); `investigation` (Investigation, default —, owner client); `design` (Design, default —, owner client); `coordination` (Approvals, default —, owner client); `tender` (Tender, default —, owner client); `construction` (Support, default —, owner client); `completion` (Completion, default —, owner client); `access` (Staging, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Fee/basis/quantities/ceilings by stage/system; ex GST/GST/total, base/options/milestones/dependencies; numbered addenda/validity and authorised signatory.
- Note to R2 when always: Outputs/calculations: maturity/native formats/revisions/rounds/reliance/dates; tests: pressure/leakage/flow/temperature/backflow/drain/pump/control/integrated.
- Note to R3 when always: Lead/designer/specialist/checker/modeller/inspector/witness rates; investigation/visit/review/test/retest units/ceilings; travel/out-of-hours/cancellation/escalation.
- Note to R4 when always: Lead/checker/competence/registrations/conflicts/stage team/specialists, ISO/equivalent QA; Section 2 responsibility allocations.
- Note to R5 when always: 2-3 comparable use/source/system/live/staged jobs; staff roles/outputs, authority/test lessons, referees.
- Note to R6 when always: Sections 2-5/agreement departures: wording/reason/fee/time, exclusion owner; resolve facts or priced assumption with owner/date/consequence/review point.
- Note to R7 when always: Inputs/providers/versions/dates: fixtures/occupancy/equipment, authority/levels/tests/source quality, civil/OSD/treatment/fire/operator; methods/checks/gates/isolation/temporary supply/test criteria/closure.
- Note to R8 when always: PI {field:pi}; PL {field:pl}; current workers cover/periods/certificates/activity exclusions; agreement/precedence/liability/reliance/IP/termination/novation.
- Note to R9 when always: Authority/Water Services Coordinator/surveys/CCTV/tests/lab/access/isolation/make-good/specialists/travel: quantities/payer/markup/GST/approval; no duplicate fees.

**Flags this template tests:** `hyd_alternative_sources`, `hyd_declarations`, `hyd_external`, `hyd_gas`, `hyd_interfaces`, `hyd_investigation`, `hyd_planning`, `hyd_process`, `hyd_pumps`, `hyd_rainwater`, `hyd_roof`, `hyd_staging`, `hyd_trade`, `hyd_treatment`, `hyd_utility`

## land-surveying

- File: `knowledge/rfp/land-surveying.yaml`; status `draft`, version 2
- Discipline ids: `consultant.surveyor`; title aliases (suggest only): surveying, surveyor
- Title: Land surveying services
- Evidence limit: Historical carpark/services offers do not establish other scopes, current insurance or legal requirements.

**Boundary** (first true rule suggests the appointment range):

- always → s1_plan … s4_services

**Scale.** Purpose, extent and accuracy govern survey scope; building type alone appoints no package.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_plan` | S1 Brief / survey plan |  | always | ask | include |  |
| `s2_detail` | S2 Detail / topographic |  | flag∈{survey_detail} | ask | include |  |
| `s3_boundary` | S3 Boundary / title |  | flag∈{survey_boundary} | ask | include |  |
| `s4_services` | S4 Services / drainage / sewer |  | flag∈{survey_services} | ask | include |  |
| `s5_support` | S5 Approval / tender support |  | always | ask | option |  |
| `s6_construction` | S6 Set-out / checks / monitoring |  | always | ask | option |  |
| `s7_completion` | S7 Completion / final plans |  | always | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `boundary` | Survey control and boundaries | Boundary identification for setback/easement design; surveyor reports geometry, solicitor advises rights | technical | I when stage_included∈{s3_boundary}, else O | flag∈{survey_boundary} | ask |  |
| `titling` | Survey control and boundaries | Subdivision, strata and easement creation; separately selected/priced | technical | O | always |  |  |
| `control` | Survey control and boundaries | Agreed horizontal/vertical control and threshold/slab/outlet/retained levels; record accuracy, resolve datum conflicts before merging | technical | I | always |  |  |
| `adjoining` | Survey control and boundaries | Neighbour heights/windows/trees/roof; permissions, inferred labels; arborist owns tree health, architect privacy/shadow | technical | I when stage_included∈{s2_detail}, else O | flag∈{survey_adjoining} | ask |  |
| `records` | Services and measured building evidence | Underground services records and visible features; locator states methods/limits, surveyed paint is not proof | technical | I | always |  |  |
| `exposure` | Services and measured building evidence | Exposure, potholing, isolation and surface restoration | technical | X | always |  | client-appointed exposure/isolation and reinstatement contractors (name each in R4) |
| `buried` | Services and measured building evidence | Buried tanks/drainage/sealed pits; measured/inferred separated, name opening party; civil owns capacity, structural adequacy | technical | I when stage_included∈{s4_services}, else O | flag∈{survey_buried} | ask |  |
| `sewer` | Services and measured building evidence | Sewer/main peg-out versus condition report; provider/method named, CCTV/cleaning/tracing/approvals separately selected | technical | I when stage_included∈{s4_services}, else O | flag∈{survey_sewer} | ask |  |
| `interior` | Services and measured building evidence | Interior measure-up/scan/point cloud; select rooms/levels, name scan-to-model party; boundaries only if needed | technical | O | always |  |  |
| `access` | Access and construction surveys | Occupied/staged site permissions/hours/daily handback; client obtains occupier/neighbour consent, return inaccessible register | technical | I when flag∈{live_operations,staged}, else - | always | ask |  |
| `monitoring` | Access and construction surveys | Movement monitoring/repeat observations; structural/geotechnical lead supplies triggers and interpretation | technical | O | always |  |  |
| `setout` | Access and construction surveys | Construction set-out; builder provides approved revisions, protects control and books | technical | I when stage_included∈{s6_construction}, else O | always |  |  |
| `asbuilt` | Access and construction surveys | Work-as-executed checks; missed hold points not certified later | technical | I when stage_included∈{s7_completion}, else O | always |  |  |

**Allowances**: `features` (Extent / features, default —, owner client); `control` (Control / cadastral, default —, owner client); `visits` (Visits / access, default —, owner client); `services` (Underground assets, default —, owner client, when stage∈{s4_services}); `issues` (Issues / review, default —, owner client); `setout` (Set-out option, default —, owner client, when stage∈{s6_construction}); `monitoring` (Monitoring option, default —, owner client, when stage∈{s6_construction}); `completion` (Completion / titling, default —, owner client, when stage∈{s7_completion})

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price stages/options separately, basis/quantity/GST/dependencies; base/options/alternative totals. No detail/stormwater/locator double count. Validity/milestones, addenda acknowledged; authorised signatory.
- Note to R2 when always: Selected plans/certificates, CAD/PDF/cloud/data, datum/accuracy, detection/sewer, missing features, set-out/as-built/monitoring and title records: purpose, editability, recipient, dates/rounds, reliance/limits.
- Note to R3 when always: Extra visits/hours/features/areas/pits/plans, travel/minimums, control replacement, standby/cancellation/after-hours/equipment/repeats; ceilings/escalation. Separate quantity, rate and each markup.
- Note to R4 when always: Name lead/reviewer, cadastral practitioner, crew, locator/plumber, subcontractors and availability; assign records/permissions/access/exposure/isolation/restoration, detection/measurement/checks/interpretation/lodgement. QA ISO or equivalent.
- Note to R5 when always: Two or three examples with referees matched to purpose, accuracy, hidden-service/boundary/existing/live constraints and scale; role and delivered evidence.
- Note to R6 when always: Sections 2–5/agreement: comply/departure, reason, alternative, owner and fee/time effect, or nil. Include extent/datum conflicts, boundaries/hidden assets/methods, duplicate scope/markups/reliance. Assumptions state quantity, consequence, decision owner/date.
- Note to R7 when always: Input/permission/control/revision dates, mobilisation/simultaneous specialists, field/draft/review/final/options; start/pause conditions, booking notice, currency/revisit triggers and authority dependencies.
- Note to R8 when always: Current PI {field:pi}, PL {field:pl}, workers compensation as applicable; project-selected limits agreed before issue, claim/aggregate/exclusions, registration/competence/continuity. Historical certificates are not current cover.
- Note to R9 when always: Searches, locator/plumber, authority/lodgement, travel/parking, traffic/access/safety, exposure/isolation/restoration and data: included/allowance/rate, payer/quantity/GST/markup/approval; client-direct work, no R1 duplication or stacked charges.

**Flags this template tests:** `live_operations`, `staged`, `survey_adjoining`, `survey_boundary`, `survey_buried`, `survey_detail`, `survey_services`, `survey_sewer`

## landscape-architecture

- File: `knowledge/rfp/landscape-architecture.yaml`; status `draft`, version 2
- Discipline ids: `consultant.landscape`; title aliases (suggest only): landscape architect, landscape architecture
- Title: Landscape architectural services
- Evidence limit: Two versions of one agreement with a narrow DA schedule; incorporated fee proposal absent, client acceptance unproved and novation heading/body conflict unresolved. No independent bid benchmark.

**Boundary** (first true rule suggests the appointment range):

- work_type∈{advisory} → s1_inception … s9_advisory
- always → s1_inception … s3_approval

**Scale.** Actual external extent and selected stages govern; no industrial default.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_inception` | S1 Inception / evidence |  | always | ask | include |  |
| `s2_concept` | S2 Concept |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} | ask | include |  |
| `s3_approval` | S3 Approval / responses |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} | ask | include |  |
| `s4_developed` | S4 Consent / developed design |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} | ask | include |  |
| `s5_tender` | S5 Traditional documentation |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} AND procurement∈{traditional} | ask | include |  |
| `s6_dnc` | S6 D&C / post-contract |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} AND procurement∈{design_construct} | ask | include |  |
| `s7_construction` | S7 Construction / completion |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s8_establishment` | S8 Establishment review |  | flag∈{landscape_design} AND work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s9_advisory` | S9 Advice / rectification |  | always | ask | option; include when work_type∈{advisory} |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `design` | Landscape and planting design | Planting/hardscape/external design only in enabled stages | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc} | ask |  |
| `trees` | Landscape and planting design | Retained/street/neighbouring trees: condition/impact/protection; survey locates; no removal authority | technical | X | always |  | arborist |
| `civil_structure` | Irrigation and design interfaces | Levels/runoff/WSUD by civil; retaining structure by structural; geotechnical soil advice | technical | X | always |  | civil engineer and structural engineer |
| `podium` | Landscape and planting design | Podium/roof/planter build-up and loads; structure verifies, architect/waterproofing and drainage owners named | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc} AND flag∈{landscape_podium} | ask |  |
| `irrigation_brief` | Irrigation and design interfaces | Irrigation notation/performance brief; hydraulic/electrical interfaces named | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc} AND flag∈{landscape_irrigation_brief} | ask |  |
| `irrigation_detail` | Irrigation and design interfaces | Detailed irrigation design only by expressly selected competent designer | technical | I when stage_included∈{s4_developed,s5_tender,s6_dnc}, else O | stage∈{s4_developed,s5_tender,s6_dnc} AND flag∈{landscape_irrigation_detail} | ask |  |
| `public` | Public realm and delivery | Public-domain manuals/approvals/bonds/handover; extent and authority lead agreed | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc} AND flag∈{landscape_public} | ask |  |
| `special` | Irrigation and design interfaces | Special external uses: name play/access, lighting/electrical, ecology/heritage/bushfire/horticulture specialists | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc} AND flag∈{landscape_special} | ask |  |
| `staging` | Public realm and delivery | Existing/occupied/mixed/staged work: retained-condition/tie-ins, access and separate acceptance/establishment dates | technical | I when stage_included∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc,s7_construction,s8_establishment,s9_advisory}, else O | stage∈{s2_concept,s3_approval,s4_developed,s5_tender,s6_dnc,s7_construction,s8_establishment,s9_advisory} AND flag∈{landscape_staging} | ask |  |
| `environment` | Irrigation and design interfaces | Contamination/failed planting/ingress: environmental specialist accepts soil/caps; relevant specialists diagnose cause; assign opening-up/testing/reinstatement | technical | X | always |  | environmental consultant and relevant cause specialist |
| `establishment` | Public realm and delivery | Plant establishment/maintenance inspections; contractor/operator maintains, quantified visits/end criteria | technical | I when stage_included∈{s8_establishment}, else O | stage∈{s8_establishment} | ask |  |
| `construction` | Public realm and delivery | Construction visits/completion report against selected evidenced scope; contract PC remains administrator | technical | I when stage_included∈{s7_construction}, else O | stage∈{s7_construction} | ask |  |

**Allowances**: `extent` (Extent / concepts, default —, owner client); `evidence` (Visits / tests, default —, owner client); `reviews` (Design / approvals, default —, owner client); `tender` (Tender / substitutions, default —, owner client); `works` (Works / completion, default —, owner client); `establishment` (Establishment, default —, owner client); `extras` (Duration / extras, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price selected stages/quantities once: fixed/ceiling/unit, base/options/dependencies, ex GST/GST/total and milestones; confirm validity, authorised signatory and numbered addenda.
- Note to R2 when always: Plans/details, plant/material/soil schedules, specifications/calculations, authority, completion/establishment reports and native registers: extent, formats, recipients/reliance, input revisions, rounds/dates.
- Note to R3 when always: Personnel/extra visit/review/substitution rates, meeting/response ceilings, durations, hold points/establishment, travel/callout/out-of-hours/cancellation and overrun authority.
- Note to R4 when always: Lead/reviewer, stage resources, competence, subconsultants/conflicts and ISO or equivalent QA; allocate civil/structure/arborist/irrigation/environmental/authority/contractor duties.
- Note to R5 when always: Two or three comparable actual-use/site/soil/podium/public/retained-landscape/establishment projects: personnel role, outputs, lessons and referees.
- Note to R6 when always: Sections 2–5/agreement departures: reason, alternative, fee/time; unresolved evidence or authorised assumption with owner/date/consequence/review; name excluded-work owners.
- Note to R7 when always: Input providers/dates, stage resources, access/test method, draft/review/final, authority/contractor dependencies, hold-point notice and establishment schedule; calculations/substitutions, evidence limits, urgent escalation/close-out.
- Note to R8 when always: Current selected PI {field:pi}, PL {field:pl}, workers cover, duration/exclusions/renewals; agreement/precedence acceptance or departures, novation and reliance/IP.
- Note to R9 when always: Travel/access, survey/locating, soil/lab/testing, authorities, specialists, samples/documents: included/allowance/unit, quantity/payer/GST/markup/approval; no R1 duplication or hidden installation/maintenance.

**Flags this template tests:** `landscape_design`, `landscape_irrigation_brief`, `landscape_irrigation_detail`, `landscape_podium`, `landscape_public`, `landscape_special`, `landscape_staging`

## mechanical-services

- File: `knowledge/rfp/mechanical-services.yaml`; status `draft`, version 2
- Discipline ids: `consultant.mechanical`; title aliases (suggest only): mechanical services, HVAC design
- Title: Mechanical services
- Evidence limit: Historical offers establish scope, not retained capacity, current instruments or accepted appointments.

**Boundary** (first true rule suggests the appointment range):

- (work_type∈{advisory}) OR (pathway∈{advice_only}) → advisory … advisory
- always → inception … design

**Scale.** Actual spaces, loads, hours, systems and purchased maturity govern scope.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `inception` | S1 Evidence |  | always | ask | include |  |
| `investigation` | S2 Investigation |  | flag∈{mech_investigation} | ask | include |  |
| `concept` | S3 Concept |  | always | ask | include |  |
| `planning` | S4 Planning |  | flag∈{mech_planning} | ask | include |  |
| `design` | S5 Design |  | always | ask | include |  |
| `tender` | S6 Documentation |  | always | ask | include |  |
| `continuation` | S7 Post-contract |  | always | ask | option |  |
| `construction` | S8 Construction |  | always | ask | option |  |
| `handover` | S9 Commissioning |  | always | ask | option |  |
| `seasonal` | S10 Seasonal |  | always | ask | option |  |
| `advisory` | S11 Advisory |  | always | ask | option; include when (work_type∈{advisory}) OR (pathway∈{advice_only}) |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `hvac` | Heating, cooling and ventilation | Selected detailed HVAC design: duties/air paths/outdoor air; loads/hours/reserve | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} |  |  |
| `local` | Heating, cooling and ventilation | Toilet/laundry/refuse/car-park/charging exhaust; make-up air/discharge separation | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_local} | ask |  |
| `kitchen` | Heating, cooling and ventilation | Kitchen/process exhaust; equipment brief/specialist, not kitchenette | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_kitchen} | ask |  |
| `smoke` | Smoke control and building systems | Smoke fans/shafts/dampers/pressurisation under fire-engineer strategy; wet fire separate | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_smoke} | ask |  |
| `controls` | Smoke control and building systems | BMS sequences/points/alarms/metering; programming/licences/commissioning owners | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_controls} | ask |  |
| `energy` | Smoke control and building systems | Whole-building energy/rating only; mechanical inputs remain in appointed design | technical | X | always |  | energy assessor |
| `acoustic` | Smoke control and building systems | Acoustic/envelope design only; mechanical noise inputs remain in appointed design | technical | X | always |  | acoustic consultant / architect |
| `interfaces` | Smoke control and building systems | Coordinate supports/plinths/openings/power/refrigerant leak/condensate; respective specialists design | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_interfaces} | ask |  |
| `existing` | Existing systems and specialist design | Condition/capacity/controls investigation; operation is not proof; upgrade separate | technical | I when stage_included∈{investigation}, else O | stage∈{investigation} AND flag∈{mech_investigation} | ask |  |
| `staging` | Existing systems and specialist design | Occupied/phased temporary ventilation/isolation/handback by use/phase | technical | O | always |  |  |
| `remediation` | Existing systems and specialist design | Diagnosis/trial/verification; no whole-building conclusion from samples | technical | O | always |  |  |
| `specialist` | Existing systems and specialist design | Laboratory/clean-room/medical gases/process refrigeration/hazardous areas | technical | X | always |  | competent specialist |
| `declarations` | Existing systems and specialist design | Declarations/lodgement; verify jurisdiction/competence, no approval guarantee | technical | I when stage_included∈{design,tender,continuation}, else O | stage∈{design,tender,continuation} AND flag∈{mech_declarations} | ask |  |
| `continuation` | Later design and verification | Separate continued design/novation | technical | I when stage_included∈{continuation}, else O | stage∈{continuation} |  |  |
| `construction` | Later design and verification | Quantified construction support | technical | I when stage_included∈{construction}, else O | stage∈{construction} |  |  |
| `witness` | Later design and verification | Selected witnessing/handover | technical | I when stage_included∈{handover}, else O | stage∈{handover} |  |  |
| `seasonal` | Later design and verification | Seasonal/defects reviews | technical | I when stage_included∈{seasonal}, else O | stage∈{seasonal} |  |  |
| `execution` | Later design and verification | Test execution separately priced to named contractors | technical | O | always |  |  |

**Allowances**: `extent` (Extent, default —, owner client); `investigation` (Investigation, default —, owner client); `design` (Design, default —, owner client); `coordination` (Coordination, default —, owner client); `tender` (Tender, default —, owner client); `construction` (Construction, default —, owner client); `completion` (Completion, default —, owner client); `access` (Access, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Fees/basis/quantities/ceilings by stage/system; ex GST/GST/total, base/options/dependencies/milestones. Acknowledge numbered addenda/validity and authorised signatory; price each service once.
- Note to R2 when always: Section 3 outputs/registers: purpose/maturity, native calculations/formats, recipient/reliance, revisions/rounds/dates.
- Note to R3 when always: Lead/engineer/controls/smoke/specialist/checker/modeller/inspector/witness rates; extra visit/review/test/retest units/ceilings, travel/out-of-hours/cancellation/escalation.
- Note to R4 when always: Lead/checker/registrations/competence, resources/conflicts/subconsultants, ISO/equivalent QA; responsibility matrix for design/strategy/controls/power/network/install/test/lead/witness/certify.
- Note to R5 when always: Two or three comparable systems/use/live/staged commissions: staff roles, outputs, coordination/test lessons and referees.
- Note to R6 when always: Sections 2–5/agreement departures: wording/reason/fee/time; exclusion owner; resolve conflicts or priced assumption with owner/date/consequence/review point.
- Note to R7 when always: Inputs/providers/revisions/dates: loads/hours, facade/energy/acoustics, plant/controls/tests, structure/MEP, strategy/approvals; checks/gates/outages/notice/response; balancing/integrated-test criteria/closure.
- Note to R8 when always: PI {field:pi}; PL {field:pl}; current workers cover/periods/certificates/activity exclusions; agreement/precedence, liability/reliance/IP/termination/novation.
- Note to R9 when always: Authority/surveys/logging/openings/lab/specialist/access/isolation/reinstatement/independent commissioning/travel/printing: quantity/payer/markup/GST/approval; no R1, other consultant or contractor-test duplication.

**Flags this template tests:** `mech_controls`, `mech_declarations`, `mech_interfaces`, `mech_investigation`, `mech_kitchen`, `mech_local`, `mech_planning`, `mech_smoke`

## structural

- File: `knowledge/rfp/structural.yaml`; status `draft`, version 2
- Discipline ids: `consultant.structural`; title aliases (suggest only): structural, structure, structures
- Title: Structural engineering services
- Evidence limit: One warehouse extension (BG&E, Northrop and the project manager's normalised comparison). No golden proposal priced construction-phase services or regulated design; those stages come from practice.

**Boundary** (first true rule suggests the appointment range):

- procurement∈{design_construct} → concept_schematic … dnc_tender
- work_type∈{advisory} → concept_schematic … existing_assessment
- always → concept_schematic … construction_documents

**Scale.** Service levels (shared model, construction-phase default) are suggested complex for a Class 2-6 or 9 part, a budget over $20 million, a heritage site, transfer structures, a basement or an authority asset.

Complex when: (class∈{2,3,4,5,6,9a,9b,9c}) OR (budget_over∈{20000000}) OR (flag∈{heritage}) OR (flag∈{transfer_structures}) OR (flag∈{basement}) OR (flag∈{authority_asset}) OR (flag∈{rail})

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `concept_schematic` | Concept and schematic design | concept_design | always | ask | include |  |
| `existing_assessment` | Existing structure assessment | concept_design | (work_type∈{refurb,extend,remediation}) OR (flag∈{existing_building}) OR (flag∈{osd_existing}) | option | include |  |
| `planning_approval` | Planning approval | approval_documentation | pathway∈{cdc,da,ssd,planning_permit,ref,modification} | ask | include |  |
| `authority_asset` | Authority asset assessment | approval_documentation | (flag∈{authority_asset}) OR (flag∈{rail}) | option | include |  |
| `design_development` | Design development | construction_documentation | always | ask | include |  |
| `tender_traditional` | Tender documents | construction_documentation | procurement∈{traditional,managing_contractor} | ask | include |  |
| `dnc_tender` | D&C tender documents | approval_documentation | procurement∈{design_construct} | ask | include |  |
| `regulated_design` | Regulated design | construction_documentation | regulated_design∈{yes} | ask | include | state∈{NSW} AND class∈{2,3,9c} |
| `construction_documents` | Construction documents | construction_documentation | procurement∈{traditional,managing_contractor,eci,alliance,ppp} | ask | include |  |
| `construction_phase` | Construction phase | construction_stage_inspections | always | ask | option; include when scale∈{complex} |  |
| `post_novation` | Post-novation | construction_documentation | novation∈{option,yes} | ask | include |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `design_basis` | Design basis and coordination | Design-basis schedule: geotechnical reliance, design life, importance level, loads, wind and earthquake inputs, durability and fire-resistance coordination; … | technical | I | always |  |  |
| `geotech_brief` | Design basis and coordination | Geotechnical brief for the client to commission | technical | I | always |  |  |
| `residential_framing` | Permanent structural design | Footing, framing, bracing and tie-down design; review of site and wind classification and of supplier truss and frame design | technical | I when class∈{1a,1b,10a,10b}, else - | always |  |  |
| `shared_model` | Design basis and coordination | Revit model shared with the design team | service_level | simple O / complex I | always |  |  |
| `safety_in_design` | Design basis and coordination | Safety in design report | technical | I | always |  |  |
| `value_management` | Design basis and coordination | Value management participation | service_level | simple I / complex I | always |  |  |
| `delegated_design` | Permanent structural design | Performance specifications and a delegated-design schedule for contractor-designed elements (piles, post-tensioned slabs, specialist floors): criteria, submi… | technical | I | always |  |  |
| `specialist_element_design` | Permanent structural design | Detailed design of post-tensioned slabs and piles | technical | X when procurement∈{design_construct}, else O | always | option | specialist contractor (D&C element) |
| `retention_design` | Excavation and temporary works | Shoring and piled retention design | technical | I when (flag∈{basement}) OR (flag∈{excavation_adjoining}), else - | always | option |  |
| `temporary_works_interfaces` | Excavation and temporary works | Retention, underpinning, temporary works and monitoring interfaces: designer and reviewer named, permanent and temporary loads stated | technical | I when (flag∈{basement}) OR (flag∈{excavation_adjoining}), else - | always | option |  |
| `temporary_works_design` | Excavation and temporary works | Temporary works and propping design | technical | X | always |  | contractor's temporary works designer |
| `strengthening` | Permanent structural design | Strengthening design found necessary by the existing structure assessment | technical | O when stage∈{existing_assessment}, else - | always |  |  |
| `services_fixings` | Permanent structural design | Fixing design for services and equipment, including seismic restraint | technical | O | always |  |  |
| `metalwork` | Permanent structural design | Miscellaneous metalwork: balustrades, screens, sunshades, fences | technical | O | always |  |  |
| `dilapidation` | Excavation and temporary works | Dilapidation surveys of adjoining structures | technical | O when (flag∈{excavation_adjoining}) OR (flag∈{basement}), else X | always | option | client (separate survey) |
| `super_flat_floor` | Permanent structural design | Super-flat floor design | technical | O when flag∈{super_flat_floor}, else - | always | option |  |
| `compaction` | Inspection and verification | Ground compaction inspections | technical | X | always |  | geotechnical engineer |
| `demolition_drawings` | Excavation and temporary works | Demolition drawings (engineering demolition and propping design is not replaced by them) | technical | X | always |  | architect |
| `services_footings` | Permanent structural design | Footings affected by in-ground services | technical | I | always |  |  |
| `inspection_schedule` | Inspection and verification | Inspection schedule by element and hold point, with notification lead time and the basis for reinspection | technical | I when stage∈{construction_phase}, else - | always |  |  |

**Allowances**: `design_meetings` (Design meetings, default 4, owner client); `site_visits` (Site visits, default 1, owner client); `authority_requests` (Authority requests in assessment, default 1, owner client, when stage∈{planning_approval}); `review_rounds` (Peer review or certifier rounds, default 1, owner client, when stage∈{construction_documents,post_novation}); `ca_inspections` (Construction inspections, default —, owner client, when stage∈{construction_phase})

**Discipline returnables** (added under the shared R headings):

- R1: Existing-structure and authority-asset stages are their own lines.
- R2: Drawing list by stage, specification, reports and certificates.
- R3: Roles: Director · Associate · Senior engineer · Engineer · Drafting or BIM lead · Drafter. Per-meeting rates if used.
- R4: Named director and project engineer; CPEng or NER.
- R5: Projects with the same class and structure type.
- Note to R4 when stage∈{regulated_design}: NSW design practitioner (structural) registration for the regulated designs.

**Flags this template tests:** `authority_asset`, `basement`, `excavation_adjoining`, `existing_building`, `heritage`, `osd_existing`, `rail`, `super_flat_floor`, `transfer_structures`

## town-planning

- File: `knowledge/rfp/town-planning.yaml`; status `draft`, version 2
- Discipline ids: `consultant.town_planner`; title aliases (suggest only): planner, planning
- Title: Town planning services
- Unsupported pathways (blocking check `pathway_unsupported`): cdc, ref, modification, planning_proposal, planning_permit
- Evidence limit: All three golden proposals are for one NSW industrial DA. SSD and post-approval scope comes from general NSW practice.

**Boundary** (first true rule suggests the appointment range):

- (pathway∈{advice_only,exempt}) OR (work_type∈{advisory}) → prelim_advice … prelim_advice
- always → prelim_advice … determination_review

**Scale.** Service levels are suggested complex for SSD, a planning panel, a varied development standard, a heritage site or likely objections.

Complex when: (pathway∈{ssd}) OR (flag∈{panel}) OR (flag∈{cl46}) OR (flag∈{heritage}) OR (flag∈{objections})

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `prelim_advice` | Preliminary planning advice | concept_design | always | ask | include |  |
| `pre_lodgement` | Pre-lodgement | approval_documentation | pathway∈{da,ssd} | ask | include |  |
| `planning_report` | Planning report | approval_documentation | pathway∈{da,ssd} | ask | include |  |
| `variation_request` | Variation request | approval_documentation | flag∈{cl46} AND pathway∈{da,ssd} | option | include |  |
| `lodgement` | Lodgement | approval_documentation | pathway∈{da,ssd} | ask | include |  |
| `assessment` | Assessment | approval_documentation | pathway∈{da,ssd} | ask | include |  |
| `determination_review` | Determination review | approval_documentation | pathway∈{da,ssd} | ask | include |  |
| `post_approval` | Post-approval support | construction_documentation | pathway∈{da,ssd} | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `specialist_review` | Planning evidence and submission | Review of specialist reports for planning compliance before lodgement | technical | I | pathway∈{da,ssd} |  |  |
| `certificates` | Planning evidence and submission | Planning certificate, title and DP searches (fees at cost) | technical | I | always |  |  |
| `historic_consents` | Planning evidence and submission | Historic consents through GIPA | technical | O | always |  |  |
| `compliance_matrix` | Planning evidence and submission | Controls compliance matrix with departures and the decisions needed; specialist report schedule with owner and required date | technical | I | pathway∈{da,ssd} |  |  |
| `submissions_response` | Assessment and coordination | Response to public submissions | technical | I when flag∈{objections}, else O | pathway∈{da,ssd} | option |  |
| `panel_attendance` | Assessment and coordination | Attendance at planning panel meetings, to the allowance in section 4 | technical | I when flag∈{panel}, else O | pathway∈{da,ssd} | option |  |
| `ad_hoc_advice` | Assessment and coordination | Ad hoc planning advice and consultant coordination, within the meeting allowance | service_level | simple I / complex I | always |  |  |
| `lodgement_format` | Planning evidence and submission | Pre-lodgement plans in council's required format | technical | I | stage∈{pre_lodgement} |  |  |
| `modification` | Separate services and costs | Modification application after approval | technical | O | pathway∈{da,ssd} |  |  |
| `court` | Separate services and costs | Land and Environment Court appeal or expert witness | technical | X | always |  | client (separate engagement) |
| `authority_fees` | Separate services and costs | Council, agency and panel fees; contributions | technical | X | always |  | client |
| `specialist_reports` | Separate services and costs | Specialist reports (traffic, acoustic and others) | technical | X | always |  | specialist consultants engaged by the client |

**Allowances**: `design_meetings` (Project design meetings, default 2, owner client); `site_inspections` (Site inspections, default 1, owner client); `plan_reviews` (Reviews of draft plans, default 1, owner client); `report_drafts` (Report drafts, default 1, owner client); `report_updates` (Report updates after plan revisions, default 1, owner client, when stage∈{planning_report}); `pre_lodgement_meetings` (Pre-lodgement meetings, default 1, owner client, when stage∈{pre_lodgement}); `rfi_responses` (Authority information requests, default 1, owner client, when stage∈{assessment}); `authority_meetings` (Authority meetings in assessment, default —, owner client, when stage∈{assessment}); `panel_meetings` (Planning panel meetings, default 1, owner client, when flag∈{panel})

**Discipline returnables** (added under the shared R headings):

- R2: Reports and letters you will issue, with drafts.
- R3: Roles: Director · Associate Director or Senior Associate · Senior Planner · Planner · Graduate.
- R4: Named director and day-to-day lead.
- R5: Applications on the same pathway for this use, in the same consent authority or LGA, with the outcome.
- R9: At cost with no handling fee is preferred.
- Note to R4 when pathway∈{ssd}: SSD: REAP or EIA accreditation of the EIS author.

**Flags this template tests:** `cl46`, `heritage`, `objections`, `panel`

## traffic

- File: `knowledge/rfp/traffic.yaml`; status `draft`, version 2
- Discipline ids: `consultant.traffic`; title aliases (suggest only): traffic, transport, parking
- Title: Traffic and parking services
- Unsupported pathways (blocking check `pathway_unsupported`): planning_proposal
- Evidence limit: All three golden proposals priced one DA traffic report for a warehouse. Later stages appear in the golden set only as exclusions; their scope comes from practice.

**Boundary** (first true rule suggests the appointment range):

- (pathway∈{advice_only,exempt}) OR (work_type∈{advisory}) → design_advice … design_advice
- always → design_advice … travel_plan

**Scale.** Service levels are suggested complex when the use generates heavy vehicles, the site fronts or affects a state road, or the development is referred to TfNSW.

Complex when: (flag∈{heavy_vehicles}) OR (flag∈{state_road}) OR (flag∈{tfnsw_referral})

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `design_advice` | Parking and access advice | concept_design | always | ask | include |  |
| `surveys` | Surveys | approval_documentation | flag∈{intersection_modelling} | option | include |  |
| `modelling` | Intersection modelling | approval_documentation | flag∈{intersection_modelling} | option | include |  |
| `report` | Traffic and parking report | approval_documentation | pathway∈{cdc,da,ssd,planning_permit,ref,modification} | ask | include |  |
| `assessment_response` | Assessment response | approval_documentation | pathway∈{da,ssd,planning_permit,modification} | ask | include |  |
| `conditions_review` | Conditions review | approval_documentation | pathway∈{da,ssd,planning_permit,modification} | ask | include |  |
| `design_review` | Review of the developed design | construction_documentation | always | ask | option |  |
| `car_park_certification` | Car park certification | construction_documentation | always | ask | option |  |
| `ctmp` | Construction traffic management plan | construction_stage_inspections | always | ask | option; include when scale∈{complex} |  |
| `travel_plan` | Green travel plan | construction_documentation | flag∈{gtp_required} | option | include |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `vehicle_schedule` | Transport access and parking | Vehicle schedule (waste, emergency, service and operator vehicles); ramps, gradients, headroom, sight lines and pedestrian conflicts checked against the curr… | technical | I | always |  |  |
| `consultation` | Assessment and consultation | Consultation with council traffic engineers and TfNSW while preparing the report | technical | I | stage∈{report} |  |  |
| `end_of_trip` | Transport access and parking | Bicycle parking and end-of-trip assessment | technical | I | stage∈{report} |  |  |
| `demand_reduction` | Transport access and parking | Parking demand reduction measures | service_level | simple O / complex I | always |  |  |
| `alternative_schemes` | Assessment and consultation | Alternative schemes tested | service_level | simple O / complex O | always |  |  |
| `swept_paths` | Transport access and parking | Swept paths for every loading dock and design vehicle | technical | I | always |  |  |
| `microsimulation` | Assessment and consultation | Microsimulation beyond SIDRA | technical | O when flag∈{intersection_modelling}, else X | always | option | not_required |
| `road_safety_audit` | Independent and separate services | Road safety audit by an auditor independent of the design team; state its stage and scope in R2 | technical | O when flag∈{public_road}, else X | always | option | independent road safety auditor |
| `public_road_design` | Independent and separate services | Design of works in the public road (signals, intersections) | technical | X | always |  | civil engineer (separate engagement) |
| `court` | Independent and separate services | Expert witness, Land and Environment Court | technical | X | always |  | client (separate engagement) |
| `survey_costs` | Independent and separate services | Survey costs | technical | I | stage∈{surveys} |  |  |

**Allowances**: `survey_locations` (Survey locations, default —, owner client, when stage∈{surveys}); `sidra_locations` (Modelled intersections, default —, owner client, when stage∈{modelling}); `plan_revisions` (Plan revisions assessed, default 1, owner client); `report_drafts` (Report drafts, default 1, owner client); `design_meetings` (Design team meetings, default 3, owner client); `site_inspections` (Site inspections, default 1, owner client); `authority_responses` (Authority responses, default 1, owner client, when stage∈{assessment_response}); `authority_meetings` (Authority meetings, default —, owner client, when stage∈{assessment_response})

**Discipline returnables** (added under the shared R headings):

- R2: Report, model files, swept path drawings and letters.
- R3: Roles: Director · Senior engineer · Traffic engineer · Graduate · CAD or graphics. Per-meeting rates if you use them.
- R4: Credentials: CPEng or RPEQ, road safety auditor level, traffic control planner.
- R5: Traffic reports for this use, in the same LGA or state road context.
- R8: Conflict of interest statement.
- R9: Included is preferred.
- Note to R7 when stage∈{surveys}: Surveys need a period representative of the use.

**Flags this template tests:** `gtp_required`, `heavy_vehicles`, `intersection_modelling`, `public_road`, `state_road`, `tfnsw_referral`

## waste-management

- File: `knowledge/rfp/waste-management.yaml`; status `draft`, version 2
- Discipline ids: `consultant.waste`; title aliases (suggest only): waste management, waste consultant
- Title: Waste management consulting services
- Evidence limit: Four historic warehouse offers differ on operational and C&D plans, reviews and equipment support; none establishes universal rates, appointment, current insurance or collection acceptance.

**Boundary** (first true rule suggests the appointment range):

- work_type∈{advisory} → s1_inception … s8_advisory
- always → s1_inception … s3_cd

**Scale.** Actual uses, waste streams and selected services govern scope; no industrial default.

Complex when: never (no rules: always simple)

**Stages**

| key | name | delivery stage | when | if unknown | default / include_when | suggest |
|---|---|---|---|---|---|---|
| `s1_inception` | S1 Inception / evidence |  | always | ask | include |  |
| `s2_concept` | S2 Operational concept |  | flag∈{waste_operational} | ask | include |  |
| `s3_operational` | S3 Operational plan |  | flag∈{waste_operational} | ask | include |  |
| `s3_cd` | S3 C&D plan |  | flag∈{waste_cd} | ask | include |  |
| `s4_review` | S4 Consent / design review |  | work_type∈{new,refurb,extend,remediation} | ask | include |  |
| `s5_tender` | S5 Tender support |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s6_contractor` | S6 Contractor review |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s7_completion` | S7 Construction / completion |  | work_type∈{new,refurb,extend,remediation} | ask | option |  |
| `s8_advisory` | S8 Advisory review |  | always | ask | option; include when work_type∈{advisory} |  |
| `s8_operational` | S8 Post-occupation review |  | always | ask | option |  |

**Positions** (scope of services)

| key | group | label | kind | value | row when | if unknown | responsible if X |
|---|---|---|---|---|---|---|---|
| `operational` | Waste planning and collection | Operational WMP: actual streams, bins/storage and collection; dated rates/records only | technical | I when stage_included∈{s3_operational}, else O | flag∈{waste_operational} | ask |  |
| `cd` | Waste planning and collection | C&D plan: separate quantities, reuse/disposal pathways and tracking | technical | I when stage_included∈{s3_cd}, else O | flag∈{waste_cd} | ask |  |
| `hazardous` | Waste planning and collection | Hazardous/contaminated/clinical/liquid/process waste classification and clearance; adviser coordinates findings | technical | X | always |  | competent environmental/hazardous-waste specialist named in R4 |
| `collection` | Waste planning and collection | Written confirmation of operational bins, frequency, vehicles and access; name confirmation owner/date, operator engagement separate | technical | I when stage_included∈{s2_concept,s3_operational}, else O | flag∈{waste_operational} | ask |  |
| `swept_paths` | Facilities and design interfaces | Vehicle swept paths/clearances; civil/structural confirm grades and loads | technical | X | always |  | traffic engineer named in R4 |
| `rooms` | Facilities and design interfaces | Operational room/chute/compactor capacity and bin routes; architect details space, services design ventilation/drainage/power | technical | I when stage_included∈{s2_concept,s3_operational}, else O | flag∈{waste_operational} | ask |  |
| `equipment` | Facilities and design interfaces | Equipment design, supply and commissioning | technical | X | always |  | equipment supplier or engineer named in R4 |
| `shared` | Facilities and design interfaces | Mixed uses/shared facilities/operators: separate streams, tenancy rights/duties; unknown tenants need stated assumptions | technical | I when stage_included∈{s2_concept,s3_operational,s3_cd,s4_review,s5_tender,s6_contractor,s7_completion,s8_operational}, else O | flag∈{mixed_use,waste_shared} | ask |  |
| `interim` | Delivery and verification | Occupied/staged/heritage/remediation interim plans/checks; contractor controls methods, operator maintains service | technical | O | always |  |  |
| `targets` | Delivery and verification | Recycling/circularity/rating/carbon support: agreed target and measurement, no credits/savings guarantee | technical | O | always |  |  |
| `completion` | Delivery and verification | Construction records and completion checks: defined samples/frequency/end, contractor supplies evidence | technical | I when stage_included∈{s7_completion}, else O | always |  |  |
| `postoccupation` | Delivery and verification | Post-occupation records/cost review: separately selected period and operator actions | technical | I when stage_included∈{s8_operational}, else O | always |  |  |
| `independent` | Delivery and verification | Independent audit/peer review/expert evidence: independence/reliance, no design or operating duty transfer | technical | O | always |  |  |

**Allowances**: `coverage` (Physical coverage, default —, owner client); `plans` (Plans / rounds, default —, owner client); `meetings` (Coordination, default —, owner client); `design` (Design / testing, default —, owner client); `visits` (Site / completion, default —, owner client); `duration` (Duration / extras, default —, owner client)

**Discipline returnables**: none of its own; the shared R1–R9 apply.
- Note to R1 when always: Price selected stages/services once: quantities, fixed/ceiling/unit basis, base/options, package dependencies, ex GST/GST/total, milestones and addenda. Confirm authorised signatory and addenda numbers.
- Note to R2 when always: Plans, calculations, layouts/forms and condition/decision/completion records: extent/input revisions, formats/editable files, recipients/reliance, rounds/dates and evidence limits.
- Note to R3 when always: Personnel and extra unit rates; counted rounds/meetings/visits/rooms/tests/reporting periods, travel/delay/cancellation and prior-overrun approval. State rate fixed period/indexation.
- Note to R4 when always: Lead/reviewer, resources, competence/conflicts, supplier affiliations and QA (ISO or equivalent); allocate architect/services/fire/access, traffic/specialists, contractor/operator and evidence owners.
- Note to R5 when always: Two or three comparable actual-use/work/stream/collection/staging projects: personal roles, outputs, lessons and referees.
- Note to R6 when always: Numbered scope/agreement departures: alternative, reason, fee/time; unknown/conflicting facts need resolution or priced assumption with owner/date/consequence/review point. Include IP/reliance/payment/liability and excluded parties.
- Note to R7 when always: Input/access owners/dates: plans/sections/DWGs/clearances, occupancy/activity, records, photos and quantity estimates; specialist reports and contractor processes/destinations. Method checks, council/operator/traffic dependencies, milestone/end, risks and expedited option.
- Note to R8 when always: Current PI {field:pi}, PL {field:pl}, workers cover/duration, exclusions/renewals; subcontractors and agreement/precedence acceptance or departures. Historical cover is insufficient.
- Note to R9 when always: Authority/collector/traffic/specialist/equipment/survey/travel/platform/expedite costs: included/allowance/unit, quantity, payer, GST/markup/prior approval; client-direct costs and no R1 duplication.

**Flags this template tests:** `mixed_use`, `waste_cd`, `waste_operational`, `waste_shared`

## Disciplines in `data/intake/disciplines.json` with no RFP template

`Catalog.RFPForDiscipline` (`internal/knowledge/rfp.go`) matches a package's `discipline_id` against `discipline_ids`. A services package with no match gets the generic `tpl.rfp-capex` report (`Store.reportTemplateFor` → `reportTemplate`, `internal/store/report_rfp.go`, `internal/store/reports.go`); a works package gets an RFT (`tpl.rft`); a supplier has no RFP or RFT (`procurement.DisciplinePackage` refuses it).

| id | label | suggested by package_defaults? |
|---|---|---|
| `consultant.acoustic` | Acoustic | no |
| `consultant.archaeology` | Archaeology | **yes** |
| `consultant.av` | AV Consultant | no |
| `consultant.basix` | BASIX | no |
| `consultant.bms` | BMS Consultant | no |
| `consultant.building` | Building Consultant | no |
| `consultant.bushfire` | Bushfire | **yes** |
| `consultant.commissioning` | Commissioning Agent | no |
| `consultant.construction_manager` | Construction Manager | no |
| `consultant.demolition` | Demolition Consultant | no |
| `consultant.ecology` | Ecology | **yes** |
| `consultant.environmental` | Environmental | **yes** |
| `consultant.esd` | ESD | no |
| `consultant.expert_witness` | Expert Witness | no |
| `consultant.facade` | Facade | **yes** |
| `consultant.hazmat` | Hazmat | no |
| `consultant.heritage` | Heritage | **yes** |
| `consultant.ict` | ICT Consultant | no |
| `consultant.interior_design` | Interior Design | **yes** |
| `consultant.lighting` | Lighting Designer | no |
| `consultant.programming` | Programmer | no |
| `consultant.project_manager` | Project Manager | no |
| `consultant.roof_access` | Roof Access | no |
| `consultant.security` | Security | no |
| `consultant.services` | Services Engineer | no |
| `consultant.signage` | Signage Consultant | no |
| `consultant.specialist` | Specialist Consultant | no |
| `consultant.vertical_transport` | Vertical Transport | **yes** |
| `consultant.waterproofing` | Waterproofing | no |
| `consultant.workplace` | Workplace Strategist | no |
| `trade.main_works` | Main Works | no |
| `trade.civil_works` | Civil Works | no |
| `trade.electrical` | Electrical Services | no |
| `trade.mechanical` | Mechanical Services | no |
| `trade.hydraulic_plumbing` | Hydraulic and Plumbing Services | no |
| `trade.structural_steel` | Structural Steel | no |
| `trade.fire_services` | Fire Services Trade | no |
| `trade.joinery_kitchens` | Joinery and Kitchens | no |
| `supplier.windows_glazing` | Windows and Glazing | no |
