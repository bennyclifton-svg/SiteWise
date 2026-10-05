---
tier: topic
loaded_by: task subject (programme, schedule, critical path, lead time, residential cycle time, risk register)
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
topics: [programme, critical-path, lead-times, risk, commercial]
summary: "Residential and non-residential programme discipline: baseline programmes, CPM logic, milestones, short-term planning, procurement sequencing, delay analysis, EOT evidence, commissioning, handover and recovery planning."
required_by: {create-pmp: 6}
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §voice-and-style, §escalation-triggers]
seed_type: programme-reference
state_default: NSW
agents_anchors: [§1, §2, §3, §5, §6, §8, §9, §11]
---

# Programme scheduling guide - residential

This seed gives SiteWise its residential programme and cycle-time posture. It is loaded by `seed-targeted-read` when the task signal is programme, scheduling, lead time, critical path, lookahead, EOT support, or risk register.

The filename uses `program` for compatibility with the existing seed catalogue. The content uses Australian construction language: **programme**.

Authority stack reminder (per `../AGENTS.md §1`): project evidence in the active project folder beats this seed. The executed contract, current programme, approval conditions, certifier inspection requirements, supplier commitments, and site records are the source of truth. Where evidence is missing, the agent labels the point as Assumption per `../00-doctrine/doctrine.md §evidence-discipline`.

## 1. Purpose and scope

Residential programming is not just a bar chart. It is the time-control system that connects approvals, procurement, trade sequencing, inspections, stage-payment entitlement, EOT evidence, owner decisions, and handover.

This seed covers:
- master programmes for residential projects;
- milestone trackers;
- four to six week lookahead programmes;
- delay registers and EOT-supporting programme updates;
- programme risk commentary for reports and owner updates;
- residential cycle-time benchmarks for slab, frame, lockup, fixing, completion, and handover.

This seed does not create a standalone programme-drafting system skill. System skills that need programme coverage load this seed and then use `markdown-draft-for-review`, `register-row-draft`, and, where approved, the Excel atomics.

## 2. Residential programme outputs

The usual residential programme outputs are:

| Output | Folder | Voice | Purpose |
|---|---|---|---|
| Master programme | `06-programme/` | contractual | Baseline sequence from mobilisation to PC, OC, and DLP |
| Milestone tracker | `06-programme/` | contractual | Key dates and movement against baseline |
| Lookahead programme | `06-programme/` | contractual | Rolling four to six week coordination tool |
| Delay register | `06-programme/` or `07-construction/07-programme-eot/` | contractual | Contemporaneous delay events, cause, impact, owner, next action |
| EOT-supporting update | `07-construction/07-programme-eot/` | contractual | Contractual evidence for time claims or assessments |
| Owner programme summary | `08-meetings-reporting/owner-programme*` | stakeholder | Plain-English date movement and decisions needed from the owner |
| Programme risk commentary | `08-meetings-reporting/` or `00-brief-pmp/` | contractual or stakeholder by audience | Integrated time-risk view for governance |

Every programme output should include:
- baseline date and revision date;
- source evidence consulted;
- cycle-time assumptions;
- working calendar assumptions, including public holidays, industry shutdown, and weather allowance;
- critical path or near-critical path summary;
- float assumptions;
- lead-time assumptions for approvals, utilities, inspections, and long-lead items;
- owner, builder, consultant, certifier, supplier, or authority decisions required;
- next review date.

## 3. Master programme shape

A residential master programme is typically 50 to 200 lines. A small owner-builder or ancillary project may sit at the lower end. A multi-dwelling or high-end D&C project may need more detail, but the same discipline applies.

PMP stage anchor:

Where a PMP exists, use its programme / staging section as the high-level stage source before drafting the detailed milestone spine. If the PMP does not define staging clearly, use this baseline as an explicit Assumption and recommend updating the PMP:

| Stage | Baseline meaning |
|---|---|
| Stage 1 | Concept and schematic design to DA submission |
| Stage 2 | Design development |
| Stage 3 | Construction documentation and delivery |

Detailed programme activities may be more granular than these stages, but they must map back to them. If authority, consultant, funding, contract or construction evidence requires a different stage regime, flag the conflict and recommend aligning the PMP, programme and affected procurement / consultant artefacts.

For multi-dwelling, apartment, staged OC or D&C-signalled projects, establish a more detailed PMP stage regime before the programme is drafted. A typical detailed regime is:

| Stage | Detailed meaning |
|---|---|
| Stage 1 | Design status, DA / consent status, certifier pathway and responsibility confirmation |
| Stage 2 | Procurement and D&C / head contract pathway confirmation |
| Stage 3 | Enabling works, demolition, excavation, shoring, services authority setup and site establishment |
| Stage 4 | Basement, structure and superstructure |
| Stage 5 | Envelope, services rough-in, fire/access/acoustic compliance, waterproofing, finishes and repeated dwelling / apartment cycles |
| Stage 6 | Commissioning, OC, staged handover, defects close-out and DLP commencement |

Minimum residential milestone spine:

| Milestone | Why it matters |
|---|---|
| Design lock | Starts the chain for approvals, BASIX/NatHERS modelling, structural design, and procurement |
| Planning approval / CDC / DA | Confirms the planning pathway and conditions |
| CC or building permit issued | Enables lawful construction start |
| LSL / state equivalent cleared | Blocks CC in NSW if missing |
| Site possession | Starts site risk and usually programme accountability |
| Slab / footing inspection and pour | First major site stage and common claim milestone |
| Frame complete | Structural inspection gate and claim milestone |
| Roof / lockup | Envelope control, window/door lead-time proof point, claim milestone |
| Rough-in complete | Services ready for lining and wet-area sequencing |
| Fixing complete | Joinery, linings, wet areas, and services progressing toward completion |
| PC inspection | Defects list and contract completion posture |
| OC / occupancy approval | Authority close-out and handover gate |
| Handover | Keys, manuals, warranties, HOW/HBCF handover where applicable |
| DLP start and end | Defect time window pinned to PC/handover mechanics |

The programme must not treat stage-payment names as proof of completion. HIA stages are useful programme milestones, but each stage still needs physical evidence and contract-specific meaning.

## 4. Residential cycle-time examples

The ranges below are planning benchmarks only. They are not a substitute for project evidence. The agent should label them as Assumption when project-specific durations are not evidenced.

| Cycle | Typical range | Main variables | Evidence to confirm |
|---|---|---|---|
| Site possession to slab pour | 2-6 weeks | demolition, tree removal, cut/fill, rock, services disconnections, piering, weather, certifier availability | survey, geotech, structural drawings, demolition plan, certifier inspection booking |
| Slab pour to frame start | 3-10 working days | curing requirement, supplier mobilisation, frame delivery, set-out check, wet weather | concrete docket, engineer/certifier sign-off, frame supplier delivery date |
| Frame start to frame complete | 2-6 weeks single storey; 4-10 weeks two storey | complexity, steel beams, truss timing, weather, crew size, stair/install sequencing | framer quote, truss order, structural drawings, inspection booking |
| Frame complete to roof-on | 1-3 weeks | truss/roof sheet lead time, scaffolding, weather, roof complexity, gutters | truss certificate, roof supply commitment, scaffold plan |
| Roof-on to lockup | 2-6 weeks | windows/doors lead time, brick/cladding sequence, weather, BAL detailing | window order, BASIX/NatHERS glazing commitments, cladding procurement |
| Lockup to rough-in complete | 2-5 weeks | electrical, plumbing, HVAC, gas, NBN, inspection availability | trade scopes, services drawings, inspection hold points |
| Rough-in to lining complete | 1-3 weeks | insulation inspection, plasterboard crew, moisture in framing, design changes | insulation spec, frame moisture check, lining schedule |
| Lining to fixing complete | 4-10 weeks | joinery, tiling, waterproofing, floor finishes, PC selections, owner decisions | joinery shop drawings, PC item schedule, tile/appliance orders |
| Fixing to PC | 3-6 weeks | painting, flooring, final services, commissioning, defect pickup, BASIX final evidence | commissioning records, inspection schedule, defect list |
| PC to OC / handover | 2-6 weeks | OC evidence, BASIX final, defects, final claim, owner manuals, keys | certifier requirements, BASIX assessor confirmation, handover checklist |

For multi-dwelling, these cycles repeat by dwelling, block, or stage. The agent should identify whether the programme is sequential, overlapping, or staged for settlement/handover.

For `user_role: d-and-c`, the master programme needs linked design and construction tracks. Design deliverables, design review, certifier submissions, authority responses, and construction-release milestones must be visible before the affected site activity. Do not let a site activity rely on unreleased design; surface it as a programme gap, RFI, or design deliverable action.

For `archetype: multi-dwelling`, show the classification / approval gate, first-of-type party-wall inspection, metering and utility application lead times, infrastructure / OSD / stormwater authority gates, and staged OC / subdivision / strata / handover assumptions where they affect sequence or cashflow.

## 5. Trade duration guidance

Residential trade durations vary sharply by project size, access, design maturity, and labour availability. Use the ranges below as a first-pass planning check.

| Trade / activity | Typical duration signal | Programme note |
|---|---|---|
| Demolition / clearing | 1-10 working days for simple demolition; longer if asbestos or neighbour protection applies | Separate disconnection, hazmat, and waste certificates from physical demolition |
| Excavation and cut/fill | 2 days to 3 weeks | Rock, tight access, spoil disposal, and wet weather drive variance |
| Formwork / reo / vapour barrier | 2-8 working days | Inspection booking must be a separate hold point |
| Concrete pour | 1 day plus cure/strip time | Weather and concrete supply can move dates by days at a time |
| Termite treatment | 1 day, but gate-sensitive | Must precede pour or relevant enclosure point depending on system |
| Framing | 2-10 weeks | Two-storey, steel beams, complex roof forms, and stair openings add time |
| Roof trusses and roof cover | 1-3 weeks after frame ready | Truss lead time often starts well before site readiness |
| Windows and external doors | 1-3 weeks install after manufacture | Lead time can dominate lockup; BASIX/NatHERS commitments constrain substitution |
| Brickwork / cladding | 2-8 weeks | Weather and scaffold access matter; brick choice may have 6-12 week lead time |
| Plumbing rough-in | 3-10 working days | Wet areas, gas, HW system, rainwater tank, and sewer constraints change duration |
| Electrical rough-in | 3-10 working days | Lighting changes and owner selections can destabilise fixing dates |
| HVAC / ventilation rough-in | 2-8 working days | Ducted systems and mechanical ventilation require ceiling coordination |
| Insulation | 1-4 working days | BASIX/NatHERS commitments and inspection timing matter |
| Plasterboard / linings | 1-4 weeks | Moisture in frame and incomplete services are common blockers |
| Waterproofing | 1-2 weeks including cure and inspection | Do not compress cure/test periods to recover programme |
| Tiling | 1-4 weeks | Large-format tiles and substrate prep slow production |
| Joinery | 1-3 weeks install, but 6-16 weeks lead time | Shop drawings and site measure drive reliability |
| Painting | 1-4 weeks | Exclusive possession improves productivity; rework follows late trades |
| Floor finishes | 1-3 weeks | Moisture tests and protection need explicit activities |
| Services final fix | 1-3 weeks | Appliances, HW, PV, NBN, commissioning records feed completion |
| Defect close-out | 1-3 weeks pre-PC plus DLP later | Separate pre-PC defects from DLP defects |

## 6. Lead-time guidance

The master programme must show long-lead decisions early enough that the project lead can act before they become critical path.

| Item | Typical residential lead-time signal | Trigger point |
|---|---|---|
| DA determination | 8-24 weeks in NSW where DA pathway applies | Before design promises site-start date |
| CDC | 2-6 weeks where design fits complying pathway | After compliance test, before assuming faster start |
| CC / building permit | 4-12 weeks depending on documentation and prerequisites | Before site possession |
| LSL / state equivalent | Days to weeks depending on payment and processing | Before CC in NSW |
| HBCF / HOW or state equivalent | Days to weeks, longer if eligibility/sum changes | Before contract requirements and deposit/commencement posture |
| Certifier / BPA inspections | 3 days to 3 weeks depending on market | Before each hold point |
| Utility connections | Water/sewer 4-12 weeks; electricity 6-16 weeks; NBN variable | Lodge at or before CC where possible |
| Trusses | 4-8 weeks | Order after final frame/roof design |
| Windows / glazing | 8-14 weeks | Order after BASIX/NatHERS and final openings confirmed |
| Bricks / cladding | 6-12 weeks for non-standard selections | Order before frame/roof sequence makes facade critical |
| Joinery | 6-16 weeks; imported longer | Site measure and shop drawing approval are separate activities |
| Tiles / floor finishes | 4-12 weeks for selected/imported items | Owner selection must precede fixing start |
| Appliances | 4-12 weeks, volatile by brand | Confirm before joinery and services final fix |
| HW system | 4-10 weeks for heat pump/specialist systems | BASIX/NatHERS commitment must be checked before substitution |
| PV system | 4-10 weeks plus grid/installer availability | Program commissioning before OC/handover |

If a lead time is not evidenced by supplier quote, purchase order, approval notice, or written commitment, treat it as Assumption and record the next action.

## 7. Cycle-time annotation discipline

Every master programme, lookahead, or programme summary produced through SiteWise should annotate residential cycle times. The annotation can be a table, note field, or appendix. It must state:

| Field | Requirement |
|---|---|
| Activity / milestone | The programme line being annotated |
| Cycle-time basis | Fact, Assumption, Judgement, or Recommendation per §evidence-discipline |
| Duration | Working days or calendar days; state which calendar is used |
| Source / evidence | Quote, contract, prior programme, supplier commitment, approval, inspection requirement, or seed assumption |
| Float / criticality | Critical, near-critical, or has float, with basis |
| Lead-time trigger | Date by which procurement, approval, or decision must occur |
| Owner | Who must act next |
| Review date | ISO date for the next check |

Example:

| Activity | Duration | Basis | Source | Float | Owner | Review |
|---|---|---|---|---|---|---|
| Window manufacture | 10 weeks | Assumption | BASIX-linked glazing package; supplier quote not yet received | Near-critical to lockup | Builder | 2026-06-15 |

## 8. Programme risk triggers

The following risk triggers should feed `risk-register-system` when relevant:

| Trigger | Programme consequence | Register signal |
|---|---|---|
| BAL/bushfire rating not confirmed | Window, cladding, decking, and ember-sealing lead times can change | Risk row for BAL/bushfire |
| Latent moisture / wet substrate | Lining, flooring, joinery, and painting can be blocked | Risk row for latent moisture |
| Neighbour dilapidation not captured | Access, excavation, vibration, and damage claims become harder to manage | Risk row for neighbour dilapidation |
| Tight-block access | Deliveries, craneage, spoil removal, scaffold, and trade overlap slow down | Risk row for tight-block site access |
| Owner change requests | Design, procurement, PC sums, variations, and trade resequencing move | Risk row for owner change requests |
| Weather | Excavation, slab, roof, cladding, painting, and external works slip | Risk row for weather |
| Materials lead time | Lockup/fixing/completion dates move if long-lead items are late | Risk row for materials lead time |
| Subcontractor no-show | Lookahead fails; downstream trades lose workface | Risk row for subcontractor no-show |
| Certifier / BPA availability | Hold-point inspections strand the programme | Risk row for inspection availability |
| Utility connection delay | OC/handover can be blocked late | Risk row for utility connection |
| BASIX / NatHERS substitution | Re-certification and procurement changes delay completion | Risk row for energy-commitment substitution |
| OSD / stormwater hold point | OC or external works completion can be blocked | Risk row for stormwater/OSD |

Risks are useful only where they create a next action. A risk with no owner, no source, no review date, or no next action fails §register-discipline.

## 9. Programme reporting and voice

Voice is folder-driven per `../AGENTS.md §6` and `../02-skills/atomic/markdown-draft-for-review.md`.

Use **contractual register** for:
- master programme submissions in `06-programme/`;
- lookahead programmes issued to builders, subcontractors, consultants, certifiers, or authorities;
- EOT-supporting programme updates in `07-construction/07-programme-eot/`;
- delay registers;
- programme-risk commentary used for contract administration.

Use **stakeholder register** for:
- owner-facing programme summaries;
- monthly owner updates that explain date movement in plain English;
- owner decision requests where the purpose is to explain consequences rather than issue a formal notice.

The same programme movement can require two outputs: a contractual update for the project record and a stakeholder summary for the owner. Do not blend the voices inside one document unless the audience truly needs both.

## 10. State callouts

NSW is the deep default in SiteWise v1. Non-NSW states use inline graceful-degradation callouts per `../AGENTS.md §8`.

Common programme equivalents:
- **VIC:** building permit replaces NSW CC posture; domestic building insurance and VMIA / insurer processes can affect setup timing; NatHERS rather than BASIX shapes energy evidence.
- **QLD:** QBCC Home Warranty and inspection practices can affect setup and claims; NCC energy pathway rather than BASIX.
- **SA / WA / TAS / NT / ACT:** use the state building approval, warranty/insurance, and energy evidence pathways applicable to the project. Where this seed lacks detail, flag the gap rather than silently extending NSW guidance.

For non-NSW projects, the agent should state: "State-specific timing is an Assumption until the project lead confirms the applicable approval, warranty/insurance, and energy compliance pathway."

## 11. Agent behaviour

When a task touches programme or cycle time:

1. Confirm the §2 declaration gate has passed before any phase-gate draft.
2. Load the Tier 2 archetype seed, Tier 3 role overlay, and this guide via `seed-targeted-read`.
3. Run `evidence-sweep` before drafting so project evidence beats seed benchmarks.
4. Label unsupported durations and lead times as Assumption.
5. Include the cycle-time annotation discipline above in master programmes and lookaheads.
6. Feed programme risk triggers into `risk-register-system` when a risk needs an owner and next action.
7. Use `markdown-draft-for-review` for narrative programme outputs.
8. Use `register-row-draft` for delay, action, decision, risk, EOT, or owner-supplied rows that flow from programme work.

## 12. See also

- `../AGENTS.md §1` - authority stack
- `../AGENTS.md §2` - three-overlay declaration gate
- `../AGENTS.md §3` - seed loading rules
- `../AGENTS.md §5` - output discipline
- `../AGENTS.md §6` - voice register
- `../AGENTS.md §8` - state handling
- `../AGENTS.md §9` - skill invocation
- `../AGENTS.md §11` - active-project boundary
- `../00-doctrine/doctrine.md §seed-consultation-discipline`
- `../00-doctrine/doctrine.md §evidence-discipline`
- `../00-doctrine/doctrine.md §register-discipline`
- `../00-doctrine/doctrine.md §voice-and-style`
- `../00-doctrine/doctrine.md §escalation-triggers`
- `new-dwelling-guide.md` - lifecycle, inspection gates, sequencing risks
- `setup-and-commission-guide.md` - baseline programme and commissioning registers
- `contract-administration-guide.md` - EOT, variation, claim, and notice posture
- `cost-management-principles.md` - cashflow and cost-plan relationship to programme
- `../02-skills/atomic/seed-targeted-read.md` - loads this seed by task subject
- `../02-skills/atomic/evidence-sweep.md` - identifies programme evidence
- `../02-skills/atomic/register-row-draft.md` - drafts risk, delay, action, and EOT rows
- `../02-skills/atomic/markdown-draft-for-review.md` - wraps programme drafts with frontmatter and voice checks
- `../02-skills/systems/risk-register-system.md` - consumes the risk triggers above

# Commercial and Non-Residential Expansion

## Construction Programming Fundamentals

A construction program (also called a construction schedule) is a time-based model of the project that defines the sequence, duration, and interdependencies of all activities required to deliver the works from commencement to practical completion. In Australian practice, the program is a contractual document — most standard form contracts (AS 4000, AS 2124, AS 4902) require the contractor to submit a program within a specified period after contract award.

The program serves multiple purposes:
- **Planning tool** — establishes the logical sequence of work and identifies resource requirements
- **Communication tool** — aligns the expectations of the principal, contractor, subcontractors, and consultants on timing
- **Monitoring tool** — provides the baseline against which actual progress is measured
- **Contractual tool** — establishes the basis for assessing extensions of time, delay damages, and acceleration claims

A well-prepared construction program is not simply a Gantt chart of activities — it is a logic-linked network model that reflects the real constraints and dependencies of the project. The quality of the program directly affects the project team's ability to manage time, cost, and risk.

Reference: Industry best practice (AIPM Project Scheduling guidelines); AS 4000-1997 Clause 33 (Program); AS 2124-1992 Clause 33
Applies to: All building classes and project types
Related standards: AS 6116:2018 (Project Scheduling — Application Guidelines)

Common pitfalls:
- Treating the program as a contractual formality rather than an active management tool, resulting in a document that is submitted and then ignored
- Preparing a program without logic links between activities (a "linked bar chart" rather than a true network), making critical path analysis impossible
- Not involving subcontractors in the programming process, leading to unrealistic durations and sequences that do not reflect trade realities
- Failing to include non-construction activities (design approvals, authority inspections, commissioning, client fit-out) that often drive the overall timeline

## Types of Construction Programs

Australian construction practice uses several types of programs at different levels of detail and for different purposes. Understanding when each type is appropriate is essential for effective program management.

**Master Program (Project Program):** The overarching program covering the entire project from inception to completion, including pre-construction (design, approvals, procurement), construction, and post-construction (commissioning, defects liability) phases. This is typically prepared at summary level with 100-300 activities for a medium-sized project. The master program is the contractual baseline and the document against which extensions of time are assessed.

**Detailed Construction Program:** A more granular version of the construction phase, typically with 300-1,000+ activities depending on project complexity. This includes individual trade activities, concrete pours, crane movements, and detailed sequencing. Usually prepared by the contractor's project planner and updated fortnightly or monthly.

**Look-Ahead Program (Short-Term Program):** A rolling 3-6 week detailed program that focuses on immediate upcoming work. Updated weekly and used in site coordination meetings. The look-ahead program is the primary tool for day-to-day site management and subcontractor coordination.

**Milestone Program:** A summary-level program showing only key dates (e.g., site possession, structure complete, lock-up, practical completion, sectional completions). Often used for client and stakeholder reporting where the full program detail is not required.

**Design Program:** Covers the design documentation phase, including consultant deliverables, client reviews, authority submissions, and documentation milestones. Often prepared separately from the construction program but must be integrated for design-and-construct or early contractor involvement projects.

**Procurement Program:** Details the timeline for trade package procurement, including scope preparation, tender periods, evaluation, and subcontract award. Must be coordinated with both the design program (packages cannot be tendered until design is complete for that scope) and the construction program (subcontractors must be engaged before their work commences on site).

Reference: Industry best practice; AS 6116:2018 (Project Scheduling)
Applies to: All building classes and project types

Common pitfalls:
- Using only a master program without preparing look-ahead programs, resulting in poor week-to-week coordination on site
- Not linking the design, procurement, and construction programs, so delays in design documentation are not visible in the construction timeline until they cause actual site delays
- Preparing a milestone program for stakeholder reporting that is disconnected from the detailed program, leading to inconsistent date reporting

## Critical Path Method (CPM)

The Critical Path Method is the standard scheduling technique used in Australian construction. CPM identifies the longest sequence of dependent activities through the project network — the critical path — which determines the minimum project duration. Any delay to a critical path activity directly delays practical completion unless the program can be re-sequenced or accelerated.

Key CPM concepts:
- **Activity** — a discrete work task with a defined duration (e.g., "pour Level 3 slab" — 3 days)
- **Predecessor/Successor** — the logical relationship between activities that determines sequence
- **Dependency types** — Finish-to-Start (FS, most common: B cannot start until A finishes), Start-to-Start (SS: B can start when A starts), Finish-to-Finish (FF: B cannot finish until A finishes), Start-to-Finish (SF: rarely used)
- **Lag** — a time delay applied to a dependency (e.g., FS+7 days means B starts 7 days after A finishes — commonly used for concrete curing)
- **Float (slack)** — the amount of time an activity can be delayed without affecting the project end date. Activities on the critical path have zero float. Near-critical activities have low float (typically less than 10 working days) and should be monitored closely
- **Total float** — the time an activity can be delayed without delaying the project end date
- **Free float** — the time an activity can be delayed without delaying any successor activity

CPM analysis produces an early start, early finish, late start, and late finish date for every activity. The difference between early and late dates gives the total float. Activities where early dates equal late dates (zero total float) are on the critical path.

Reference: AS 6116:2018 (Project Scheduling — Application Guidelines)
Applies to: All building classes and project types
Related standards: PMI Practice Standard for Scheduling (international reference)

Common pitfalls:
- Using excessive lag values instead of modelling intermediate activities (e.g., using FS+28 days for concrete curing instead of creating a separate "cure Level 3 slab" activity), which obscures the logic and makes delay analysis unreliable
- Creating artificial constraints (mandatory start/finish dates) that override the CPM logic, masking the true critical path
- Not reviewing the critical path regularly — it can shift as the project progresses and activities are completed ahead of or behind schedule
- Ignoring near-critical paths (activities with low float) that can become critical with minor delays

## Program Preparation and Baseline Setting

Preparing a construction program requires structured input from the project team, subcontractors, and consultants. The process should be methodical and collaborative rather than a solo exercise by the project planner.

The standard program preparation process in Australian construction:

1. **Define the work breakdown structure (WBS)** — decompose the project into manageable work packages, typically aligned with trade or zone divisions (e.g., structure, facade, mechanical, electrical by floor or zone)
2. **Determine activity durations** — based on quantity take-offs, production rates, crew sizes, and historical data from similar projects. Durations should reflect realistic working conditions, not best-case scenarios
3. **Establish logic links** — define the dependencies between activities based on physical constraints (you cannot install services before the slab is poured), resource constraints (limited crane availability), and contractual constraints (authority hold points)
4. **Assign calendars** — apply the appropriate working calendar to each activity (e.g., 5-day week vs 6-day week, accounting for public holidays, RDOs, and any site-specific restrictions on working hours)
5. **Calculate the schedule** — run the CPM forward and backward pass to determine dates, float, and the critical path
6. **Review and validate** — compare the calculated completion date against the contractual date for practical completion. If the program shows completion after the contractual date, the sequence, durations, or resourcing must be revisited

Once accepted, the program is "baselined" — a snapshot is saved as the original plan against which all future progress is measured. The baseline should not be changed unless a formal program revision is agreed between the parties.

**Rostered Days Off (RDOs):** In Australia, many construction workers are covered by the Building and Construction General On-site Award, which includes a system of Rostered Days Off. RDOs typically fall on every second Monday or Friday, effectively reducing the available working days per month. The program calendar must account for RDOs, as failure to do so results in an optimistic program that cannot be achieved.

**Public holidays by state:** Each Australian state and territory has different public holiday schedules. Projects in different states require state-specific calendars. Multi-state portfolios must manage calendar variations across jurisdictions.

Reference: AS 6116:2018 (Project Scheduling); Building and Construction General On-site Award 2020
Applies to: All building classes and project types
Related standards: AS 4000-1997 Clause 33 (Contractor's obligation to provide a program)

Common pitfalls:
- Using generic 5-day calendars without accounting for RDOs, public holidays, and site-specific working restrictions, leading to programs that show dates 10-15% earlier than achievable
- Not consulting subcontractors on activity durations, resulting in unrealistic timeframes that subcontractors will not commit to
- Failing to baseline the program, making it impossible to demonstrate the impact of changes or delays at a later date
- Setting a single baseline and never formally revising it even when significant approved changes (e.g., variations, EOTs) alter the project scope and timeline

## Milestone Planning and Tracking

Milestones are significant project events or achievements that have zero duration in the program. They serve as checkpoints for measuring progress and triggering contractual or commercial actions. In Australian construction contracts, milestones may be linked to payment schedules, liquidated damages, sectional completion dates, or authority inspection requirements.

Common milestone categories in Australian construction projects:

**Pre-construction milestones:**
- Development Application (DA) determination
- Construction Certificate (CC) issued (NSW) / Building Permit issued (VIC)
- Site possession date
- Builder's program submission and acceptance
- Subcontractor procurement complete for early trades

**Structural milestones:**
- Commencement of piling/foundations
- Ground floor slab complete
- Structure complete (topping out) — each level for multi-storey
- Roof structure complete

**Enclosure milestones:**
- Building envelope weathertight (lock-up)
- Facade complete
- Roof waterproofing complete

**Services milestones:**
- Mechanical rough-in complete per level
- Electrical rough-in complete per level
- Fire services installation complete
- Lift installation commencement and completion
- BMS/controls commissioning start

**Completion milestones:**
- Sectional completion dates (if applicable)
- Practical completion (the contractual completion date)
- Final completion / end of defects liability period
- Occupation Certificate (OC) issued

Milestones should be logic-linked to the activities that drive them, not artificially constrained to fixed dates. A milestone driven by the program logic will move if predecessor activities are delayed, providing an early warning of schedule risk. A milestone with a fixed constraint will show as "on time" regardless of the underlying program status, which defeats its purpose as a monitoring tool.

Reference: AS 4000-1997 Clause 33 (Program), Clause 34 (Time); AS 2124-1992 Clause 35
Applies to: All building classes and project types
Related standards: Environmental Planning and Assessment Act 1979 (NSW), Building Act 1993 (VIC) — for authority milestone requirements

Common pitfalls:
- Setting too many milestones (diluting their significance) or too few (insufficient monitoring granularity)
- Using constrained milestone dates rather than logic-driven dates, which hides schedule slippage from stakeholders
- Not linking milestones to contractual obligations (e.g., sectional completion dates with liquidated damages), leading to confusion about which dates carry commercial consequences
- Failing to update milestone forecasts in monthly reports, giving stakeholders a false sense of program certainty

## Progress Monitoring and Reporting

Effective program management requires systematic measurement of actual progress against the baseline program. In Australian practice, progress is typically assessed monthly (aligned with progress claim cycles) and reported in a monthly program status report.

Methods for measuring progress:

**Percentage complete per activity:** The most common method. Each activity is assigned a percentage complete based on the site team's assessment. This is simple but subjective — a common bias is to over-report progress early in an activity and under-report towards the end (the "90% complete" problem).

**Earned value analysis:** A more rigorous approach that assigns a weighted value to each activity based on its cost or resource loading. Progress is measured as the ratio of work completed (earned value) to work planned (planned value). Earned value provides schedule performance index (SPI) and cost performance index (CPI) metrics that can be used to forecast completion dates and final costs.

**Physical measurement:** For quantity-driven activities (e.g., concrete poured, facade panels installed, cable trayed), progress can be measured against the total quantity. This is objective and verifiable but requires reliable quantity data.

The monthly program status report should include:
- Summary of progress for the reporting period
- Updated program with actual dates and revised forecasts
- Critical path analysis showing any changes to the critical path
- Float analysis identifying near-critical activities at risk
- Milestone status showing forecast vs baseline dates for key milestones
- Delay register summarising any delay events and their claimed or assessed impact
- Look-ahead for the next reporting period highlighting key activities and risks

Reference: Industry best practice; AS 6116:2018 (Project Scheduling)
Applies to: All building classes and project types
Related standards: AS 4000-1997 Clause 33 (Contractor's obligation to update the program)

Common pitfalls:
- Updating the program by simply moving future activities forward to match the reporting date without properly recording actual start and finish dates, destroying the historical record
- Over-reporting percentage complete to create an appearance of being on schedule, then facing a sudden and dramatic schedule correction when the reality becomes unavoidable
- Not producing a monthly narrative explaining what happened in the period, what the impacts are, and what recovery actions are being taken
- Failing to compare progress against the baseline (showing only the current program without reference to where the project should be)

## Delay Analysis Methods

Delay analysis is the technical process of determining the cause, responsibility, and time impact of delays to a construction project. In Australian construction, delay analysis is critical for assessing extensions of time (EOT) claims and, where applicable, delay damages and prolongation cost claims.

The principal delay analysis methods recognised in Australian and international practice are:

**As-Planned vs As-Built:** Compares the original baseline program against the actual sequence and timing of work. This is the simplest method but has limited ability to demonstrate causation — it shows that a delay occurred but not necessarily what caused it.

**Impacted As-Planned:** Inserts the delay events into the baseline program and recalculates the critical path to show the theoretical impact of each delay. This method is relatively straightforward but can be criticised because it assumes the baseline program was achievable and that no other changes occurred.

**Time Impact Analysis (TIA):** Considered the most robust method. Each delay event is inserted into the updated program at the point in time it occurred, and the critical path is recalculated. The difference between the projected completion date before and after inserting the delay event gives the net delay impact. This method accounts for the evolving state of the program and concurrent delays.

**As-Built But-For (Collapsed As-Built):** Starts with the as-built program and removes delay events one by one to determine what the completion date would have been "but for" each delay. This retrospective method is useful when the as-built record is reliable but the contemporaneous program updates are not.

**Windows Analysis:** Divides the project into time "windows" (typically monthly) and analyses the delay events and critical path within each window. This method is thorough and well-suited to complex projects with multiple overlapping delays, but is time-consuming and requires detailed records.

The choice of method depends on the quality of available records, the complexity of the delay events, and the contractual requirements. Many Australian contracts and dispute resolution practitioners prefer TIA for prospective claims and Windows Analysis for retrospective analysis.

Reference: Society of Construction Law Delay and Disruption Protocol (2nd Edition, 2017); AS 6116:2018
Applies to: All building classes and project types
Related standards: AS 4000-1997 Clause 34 (Extension of time); AS 2124-1992 Clause 35.5

Common pitfalls:
- Performing delay analysis using a method that does not match the quality of available records (e.g., attempting TIA without contemporaneous program updates)
- Not accounting for concurrent delays — where both the principal and contractor contribute to delay simultaneously, the correct allocation requires careful analysis
- Using a global claim approach ("the project is X weeks late, therefore the contractor is entitled to X weeks EOT") without linking specific delay events to specific critical path impacts
- Failing to maintain contemporaneous records (daily diaries, program updates, correspondence) during construction, which severely limits the ability to perform credible delay analysis after the fact

## Extension of Time (EOT) — Program Perspective

Extensions of time are assessed by reference to the construction program. The program is the evidential basis for demonstrating whether a delay event actually affected the critical path and, if so, by how many days.

Under AS 4000-1997 (Clause 34), the contractor must:
1. Give written notice of a delay event within the time required by the contract (typically 28 days of the delay becoming apparent)
2. Provide particulars of the delay, including the claimed impact on the critical path
3. Take reasonable steps to minimise the delay (duty to mitigate)

The superintendent assesses the claim by:
1. Confirming the delay event is a qualifying cause of delay under the contract
2. Verifying that proper notice was given within the required timeframe
3. Analysing the impact on the critical path using the contemporaneous program
4. Determining the net delay after accounting for any concurrent contractor-caused delays
5. Granting or rejecting the extension (in whole or in part)

**Qualifying causes of delay** under AS 4000-1997 typically include:
- Variations directed by the superintendent
- Latent conditions
- Delays caused by the principal or the principal's other contractors
- Suspension of work directed by the superintendent
- Industrial action (subject to specific contract amendments)
- Inclement weather exceeding allowances (if specified in the contract)

**Prevention principle:** In Australian law, if the principal causes or contributes to a delay and the contract does not provide a mechanism for extending time for that cause, the principal may lose the right to claim liquidated damages for the entire delay. This makes a comprehensive EOT clause (covering all potential principal-caused delays) essential.

Reference: AS 4000-1997 Clause 34; AS 2124-1992 Clause 35.5; Peninsula Balmain Pty Ltd v Abigroup Contractors Pty Ltd [2002] NSWCA 211 (prevention principle)
Applies to: All building classes and project types
Related standards: Security of Payment legislation (for payment of delay-related claims)

Common pitfalls:
- Missing the contractual notice period, which under many contracts bars the claim entirely regardless of merit
- Failing to demonstrate impact on the critical path — a delay to a non-critical activity does not entitle an EOT even if the delay was caused by the principal
- Not separating concurrent delays — where both parties contribute to delay, the analysis must apportion responsibility
- Granting EOT without proper analysis, which can set a precedent and erode the contractual incentive for the contractor to maintain the program

## Float Ownership and Management

Float is the amount of time an activity can be delayed without affecting the project completion date. Float ownership — who has the right to "use" the available float — is a significant and often contentious issue in Australian construction.

There are three main positions on float ownership:

**Contractor-owned float:** The float in the program belongs to the contractor, who may use it to manage their work sequence, accommodate minor delays, and optimise resource allocation. Under this approach, a delay by the principal that consumes float does not entitle the contractor to an EOT until the delay actually pushes the project beyond the contractual completion date. This is the position taken by most standard form contracts in Australia unless specifically amended.

**Principal-owned float:** The principal claims ownership of the float, arguing that it represents a buffer that should be preserved for the principal's benefit (e.g., to accommodate variations and design changes without extending the completion date). This approach is less common and can be problematic because it effectively penalises the contractor for efficient programming.

**Project float (shared):** Float belongs to the project rather than either party, and is consumed on a "first come, first served" basis. Whoever causes a delay first consumes the available float, and subsequent delays by either party are assessed against the reduced float. This approach is increasingly adopted in bespoke contracts and alliance agreements.

The practical implication of float ownership becomes critical when assessing EOT claims. If the contractor owns the float and a principal-caused delay consumes 10 days of float on a non-critical path, the contractor does not receive an EOT because the completion date was not affected. However, if a subsequent contractor-caused delay then pushes the activity onto the critical path, the contractor may argue that they would have had float available but for the principal's earlier delay.

Reference: Industry best practice; AS 6116:2018; Multiplex Constructions Pty Ltd v Abgarus Pty Ltd (1992) — float ownership principles
Applies to: All building classes and project types
Related standards: AS 4000-1997 Clause 34; Society of Construction Law Protocol

Common pitfalls:
- Not addressing float ownership in the contract, leading to disputes when float is consumed by either party
- A contractor deliberately programming excessive float (a "fat program") to create a buffer, then claiming EOT when the principal's actions consume any of that buffer
- A principal demanding a "zero float" program that shows completion exactly on the contractual date, which is unrealistic and eliminates the contractor's ability to manage day-to-day sequencing variations
- Confusing total float (impact on project end date) with free float (impact on the next activity), leading to incorrect delay assessments

## Look-Ahead Programs and Short-Term Planning

The look-ahead program (also called a short-interval program or rolling program) is the primary tool for week-to-week site coordination. While the master program provides the strategic plan, the look-ahead translates it into actionable daily and weekly work sequences.

A typical look-ahead program in Australian construction:
- **Horizon:** 3-6 weeks (commonly 4 weeks)
- **Update frequency:** Weekly (prepared and discussed at the weekly site meeting)
- **Detail level:** Individual activities with specific areas/zones, crew assignments, and prerequisites identified
- **Format:** Bar chart format, often with a simple traffic light status for each activity

The weekly site coordination meeting (commonly held on Tuesday mornings in Australian construction) reviews:
1. **Last week's performance** — which look-ahead activities were completed, started, or delayed
2. **Upcoming week's priorities** — what must start and finish, and what constraints need to be resolved
3. **Constraint log** — a register of items blocking upcoming activities (e.g., missing design information, outstanding RFIs, materials not yet delivered, preceding work incomplete)
4. **Subcontractor commitments** — each subcontractor confirms their crew sizes, start dates, and any issues

The look-ahead program is also the basis for the "Last Planner System" (lean construction approach) used on some Australian projects, where the focus is on removing constraints before activities are due to start, improving the reliability of weekly work plans.

Reference: Industry best practice; Lean Construction Institute guidelines
Applies to: All building classes and project types

Common pitfalls:
- Preparing look-ahead programs that are disconnected from the master program, creating two parallel and potentially conflicting schedules
- Not tracking the completion rate of look-ahead commitments (Percent Plan Complete — PPC), which is the key metric for assessing the reliability of short-term planning
- Using the weekly meeting as a status update session rather than a proactive constraint removal exercise
- Not inviting key subcontractors to the weekly coordination meeting, leading to coordination failures and rework

## Construction Sequencing by Project Type

Construction sequencing varies significantly by project type and building class. Understanding the typical sequence for each project type is essential for preparing realistic programs and identifying sequencing risks.

**Multi-storey residential (Class 2, apartments):**
The typical cycle time per floor is 7-14 days depending on floor area and structural system. The sequence follows a "wedding cake" pattern where multiple floors are under construction simultaneously at different stages:
- Structure leads by 3-5 floors ahead of facade installation
- Facade leads by 2-3 floors ahead of internal rough-in
- Rough-in (mechanical, electrical, hydraulic, fire) leads by 1-2 floors ahead of internal lining
- Lining and finishing follow progressively floor by floor
- Lift installation occurs in parallel once the shaft structure is sufficiently advanced

**Commercial office (Class 5):**
Similar to multi-storey residential but typically with larger floor plates and different services intensity. Base building and tenant fit-out are often separate programs. The base building contractor delivers the shell (structure, facade, core services) and the tenant fit-out follows as a separate contract.

**Residential houses (Class 1):**
Sequential construction with limited overlap between stages. Typical overall duration for a standard dwelling is 6-9 months. Key stages: site preparation and foundations (2-4 weeks), frame and roof (3-5 weeks), lock-up (windows, external cladding — 3-5 weeks), rough-in services (2-3 weeks), internal lining and finishes (4-6 weeks), external works (2-3 weeks), final fit-off and handover (2-3 weeks).

**Refurbishment and fit-out:**
Sequencing is driven by demolition and strip-out, structural modifications, services rough-in, and finishes — often with the constraint that the existing building may be partially occupied. Staging plans, noise restrictions, and out-of-hours work requirements significantly affect the program.

Reference: Industry best practice
Applies to: Applicable building classes as noted
Related standards: AS 4000-1997 (traditional procurement); AS 4902-2000 (D&C procurement)

Common pitfalls:
- Not accounting for the "wedding cake" effect in multi-storey construction, where delays to structure on upper levels cascade through all subsequent trades on those levels
- Underestimating the time required for services rough-in, which is typically the most complex and congested phase of internal construction
- Not programming tenant fit-out as a separate activity stream in commercial projects, leading to the base building program not allowing adequate time for tenant works before the lease commencement date
- Failing to account for occupied building constraints in refurbishment projects, particularly noise and vibration restrictions during business hours

## Weather and Seasonal Planning

Australian construction is significantly affected by weather and seasonal factors that vary by state and region. A realistic construction program must account for these factors both in activity durations and in the calendar assumptions.

**Inclement weather allowances:** Most Australian construction contracts include an allowance for inclement weather days — days on which weather conditions make it impractical to work on exposed activities. The allowance is typically specified as a number of days per month, based on historical weather data for the project location.

Typical inclement weather allowances (days per month when outdoor work is impractical):
- Sydney (NSW): 3-5 days per month (higher in winter: June-August)
- Melbourne (VIC): 4-6 days per month (higher variability, rain events year-round)
- Brisbane (QLD): 3-5 days per month (concentrated in wet season: November-March)
- Perth (WA): 2-4 days per month (dry summers, wet winters)
- Darwin (NT): Wet season (November-April) can lose 8-12 days per month; dry season 0-2 days
- Hobart (TAS): 4-6 days per month

**Seasonal construction considerations:**
- **Concrete pours in extreme heat:** Temperatures above 35°C require special curing measures (cooling of aggregates, ice in mix water, night pours). Summer pours in Western Australia, Queensland, and Northern Territory require careful scheduling.
- **Concrete pours in cold weather:** Temperatures below 5°C slow curing and can damage fresh concrete. Winter pours in southern states (VIC, TAS, ACT) and highland areas may require heated enclosures or extended curing periods.
- **Wet season in tropical regions:** In Darwin and North Queensland, the wet season (November-April) can halt earthworks, foundations, and external works entirely for extended periods. Programs for tropical projects should plan enclosed work during the wet season and external work during the dry season.
- **Cyclone season:** Northern Australia (above the Tropic of Capricorn) is subject to tropical cyclones from November to April. Programs should include provisions for site securing and potential work stoppages.
- **Christmas/New Year shutdown:** The Australian construction industry traditionally shuts down for 2-3 weeks over the Christmas/New Year period. This shutdown is typically specified in the enterprise agreement and must be included in the program calendar.

Reference: Bureau of Meteorology historical weather data; Building and Construction General On-site Award 2020 (shutdown provisions)
Applies to: All building classes and project types
Related standards: AS 3600-2018 (Concrete structures — hot and cold weather concreting); AS 4000-1997 Clause 34 (inclement weather as a qualifying cause of delay)

Common pitfalls:
- Using a generic national weather allowance rather than location-specific data, leading to under-allowance in high-rainfall regions and over-allowance in arid regions
- Not distinguishing between weather-sensitive activities (earthworks, concrete, external works, crane operations) and weather-independent activities (internal fit-out, services rough-in in enclosed areas)
- Programming critical concrete pours during peak summer months in northern Australia without accounting for hot weather restrictions
- Forgetting to include the Christmas shutdown in the calendar, which can shift the program by 2-3 weeks

## Design Phase Programming

For projects where the principal manages the design process (traditional procurement), a design program is essential for ensuring documentation is completed in time to support the construction program. Delays in design are one of the most common causes of delays to construction commencement and early-stage site works.

A typical design program for a medium-scale commercial project includes:

**Concept design phase (4-8 weeks):**
- Client brief confirmation
- Site analysis and constraints mapping
- Massing studies and concept options
- Preliminary cost estimate
- Client approval of preferred concept

**Design development phase (8-16 weeks):**
- Architectural design development with consultant coordination
- Structural scheme design
- Services (mechanical, electrical, hydraulic) scheme design
- Design development cost plan
- Planning submission (DA/Building Approval) preparation

**Authority approvals (variable, 8-26+ weeks):**
- Development Application lodgement and assessment (NSW: typically 8-16 weeks for standard projects)
- Building Permit / Construction Certificate (typically 4-8 weeks after DA approval)
- Other authority approvals (roads, water, sewer, fire brigade — variable)

**Construction documentation phase (12-20 weeks):**
- Detailed architectural documentation (typically the longest duration)
- Structural documentation (coordinated with architectural)
- Services documentation (coordinated with architectural and structural)
- Specification preparation
- Pre-tender estimate
- Documentation review and coordination check

**Tender period (4-8 weeks):**
- RFT preparation and issue
- Tender period including site inspections, RFIs, and addenda
- Tender evaluation, interviews, and recommendation
- Contract negotiation and award

The critical constraint in the design program is typically consultant coordination — architectural, structural, and services documentation must be coordinated to avoid clashes and gaps. Building Information Modelling (BIM) coordination processes can mitigate this but add their own time requirements.

Reference: Industry best practice; AIQS guidelines for design stage cost management
Applies to: All building classes and project types
Related standards: Environmental Planning and Assessment Act 1979 (NSW) — DA assessment timeframes; Building Act 1993 (VIC) — Building Permit timeframes

Common pitfalls:
- Not including realistic authority approval timeframes in the design program, particularly for complex or contentious projects that may face objections or additional information requests
- Programming all consultant documentation to finish simultaneously rather than staggering completion to match the procurement sequence (structural before facade, facade before services)
- Not allowing adequate time for design coordination between disciplines, leading to clashes discovered during construction
- Underestimating the time for specification writing, which is often left until the last minute and then rushed, leading to inconsistencies with drawings

## Procurement Sequencing and the Program

The construction procurement program must be integrated with both the design program and the construction program to ensure subcontractors and materials are engaged and available when needed on site. The sequence of trade package procurement is driven by the construction sequence — early trades (demolition, earthworks, piling, structure) must be procured first, followed by enclosure trades, then services and finishes.

A typical procurement sequence for a multi-storey commercial project:

**Lead time 16-20 weeks before site start:**
- Demolition contractor
- Piling contractor (if applicable)
- Structural steel fabricator (if applicable)

**Lead time 12-16 weeks before trade commencement:**
- Concrete structure subcontractor
- Facade/curtain wall supplier and installer (long lead-time materials — up to 20+ weeks for custom facade systems)
- Lift supplier and installer (typically 16-24 weeks for manufacture and delivery)

**Lead time 8-12 weeks before trade commencement:**
- Mechanical (HVAC) subcontractor
- Electrical subcontractor
- Hydraulic/plumbing subcontractor
- Fire protection subcontractor

**Lead time 4-8 weeks before trade commencement:**
- Internal partitions and lining
- Floor and wall finishes (tiles, carpet, vinyl)
- Joinery and fit-out
- Painting
- Landscaping and external works

**Long-lead materials requiring early procurement decisions:**
- Switchboards and transformers: 16-24 weeks
- Generators: 16-20 weeks
- Chillers and major plant: 12-20 weeks
- Structural steel: 8-16 weeks (longer for imported steel)
- Custom facade panels: 16-30 weeks
- Lifts: 16-24 weeks
- Stone and imported finishes: 10-16 weeks

Reference: Industry best practice
Applies to: All building classes and project types
Related standards: AS 4120-1994 (Code of tendering)

Common pitfalls:
- Not procuring long-lead items early enough, causing the construction program to be delayed while waiting for materials or equipment
- Starting procurement of a trade package before the relevant design documentation is complete, leading to tenders based on incomplete information and subsequent variations
- Not allowing adequate tender periods (minimum 3-4 weeks for most trades, 4-6 weeks for complex packages), resulting in rushed tenders with excessive qualifications
- Failing to coordinate the facade procurement timeline with the structure program — facade typically has the longest material lead time and is on the critical path for building enclosure

## Program Risk Assessment

Program risk assessment identifies activities and sequences that are most vulnerable to delay and evaluates the potential impact on the overall project timeline. It should be performed at the start of the project and reviewed regularly as the program evolves.

Key risk areas by project phase:

**Pre-construction risks:**
- Authority approval delays (DA, CC, OC — highly variable and often outside the project team's control)
- Design documentation delays and coordination issues
- Procurement delays (inability to attract competitive tenders, long-lead material availability)
- Site access and possession constraints
- Contamination or latent conditions discovered during investigation

**Construction risks:**
- Inclement weather exceeding allowances
- Structural design changes during construction
- Services coordination clashes requiring redesign
- Subcontractor insolvency or under-performance
- Industrial action or labour shortages
- Material supply chain disruptions
- Unexpected latent conditions (particularly for refurbishment and remediation projects)
- Authority inspection delays and compliance issues

**Completion risks:**
- Commissioning complexity and duration
- Fire engineering certification and compliance
- Essential services certification
- Client fit-out and move-in coordination
- Occupation Certificate approval process

A useful risk quantification technique is to assign a probability and impact (in weeks of delay) to each risk, then calculate the expected delay. Summing expected delays across all identified risks gives a risk-adjusted completion date that can be compared against the deterministic program completion date.

Reference: AS/NZS ISO 31000:2018 (Risk management); AS 6116:2018
Applies to: All building classes and project types

Common pitfalls:
- Performing a program risk assessment at the start and never updating it, allowing emerging risks to go unmanaged
- Identifying risks but not assigning mitigation actions or owners, turning the risk register into a passive document
- Not linking program risks to cost risks — a delay almost always has a cost consequence (prolongation, acceleration, or both)
- Focusing only on construction-phase risks and ignoring pre-construction and completion-phase risks, which often cause the most significant delays

## Acceleration and Recovery Programs

When a project falls behind the contractual program, the options are to extend the completion date (if entitled to an EOT), accelerate the works to recover lost time, or accept the delay and the consequential damages.

**Directed acceleration:** The principal or superintendent directs the contractor to accelerate the works to achieve an earlier completion date or to recover from a principal-caused delay for which an EOT has not been granted. Under AS 4000-1997, directed acceleration is treated as a variation, and the contractor is entitled to be paid the reasonable costs of acceleration.

**Voluntary acceleration:** The contractor accelerates the works at their own cost to recover from a contractor-caused delay or to achieve an earlier completion date for their own commercial reasons (e.g., to redeploy resources to another project).

**Constructive acceleration:** Where the contractor is entitled to an EOT but the superintendent refuses to grant it, and the contractor accelerates to avoid liquidated damages. Constructive acceleration may give rise to a claim by the contractor for the costs of acceleration, although this can be difficult to establish.

Common acceleration measures:
- Increasing crew sizes (subject to workspace constraints and safety considerations)
- Extended working hours (overtime, weekend work, night shifts — subject to council approvals and enterprise agreement provisions)
- Resequencing activities to create parallel work fronts where the original program had sequential activities
- Prefabrication of elements off-site to reduce on-site installation time
- Additional cranes, hoists, or equipment to increase productivity
- Splitting work zones to allow multiple subcontractors to work in different areas simultaneously

When preparing a recovery program, the team must assess the feasibility and cost of each acceleration measure. Acceleration is subject to diminishing returns — doubling the workforce does not halve the duration, because of workspace congestion, coordination overhead, and the learning curve for new workers. A common rule of thumb is that the maximum practical acceleration for most construction activities is a 30-40% reduction in duration.

Reference: AS 4000-1997 Clause 36 (Variations — directed acceleration); Industry best practice
Applies to: All building classes and project types

Common pitfalls:
- Attempting to accelerate without preparing a revised program showing how the acceleration measures will recover the lost time, resulting in increased cost without schedule benefit
- Not obtaining approval from the principal before incurring acceleration costs (for directed acceleration), leading to disputes about entitlement
- Ignoring the safety implications of extended working hours and increased worker density on site
- Underestimating the diminishing returns of acceleration — adding more workers to a congested site often increases cost without proportionate time savings

## Sectional Completion and Staged Handover

Many Australian construction projects require completion and handover in defined sections or stages, rather than as a single event. Sectional completion is a contractual mechanism that allows the principal to take possession of completed parts of the works while the contractor continues working on the remaining sections.

Under AS 4000-1997, sectional completion must be expressly provided for in the contract (typically in the annexure). For each section, the contract specifies:
- The scope of works included in the section
- The date for practical completion of the section
- The rate of liquidated damages applicable to delay in completing the section
- Any specific requirements for achieving practical completion of the section (e.g., fire separation from ongoing works, temporary access arrangements)

Common scenarios requiring sectional completion in Australian construction:
- **Multi-building developments** where individual buildings are completed progressively
- **Mixed-use developments** where the retail/commercial podium is completed before the residential tower
- **Hospital and healthcare projects** where departments are handed over in a planned sequence to maintain operational continuity
- **Staged residential developments** where groups of lots or apartments are settled progressively
- **Refurbishment of occupied buildings** where floors or wings are returned to the occupant as they are completed

The program implications of sectional completion are significant. Each section requires its own critical path, and the contractor must manage resources across multiple concurrent completion activities. The commissioning, inspection, and certification requirements for each section add program time that is often underestimated.

Reference: AS 4000-1997 Clause 35.2 (Practical completion of separable portions); AS 2124-1992 Clause 42.2
Applies to: Multi-building, mixed-use, and staged projects

Common pitfalls:
- Not defining the scope of each section with sufficient precision, leading to disputes about what work must be complete for sectional completion to be achieved
- Underestimating the time and resources required for multiple completion events — each section requires its own snagging, commissioning, inspection, and certification process
- Not programming the fire separation and temporary protection required to allow the principal to occupy one section while construction continues in adjacent sections
- Failing to account for the impact of ongoing construction on occupied sections (noise, dust, vibration, access restrictions)

## Commissioning and Handover Programming

Commissioning is the systematic process of verifying that all building systems and components perform according to their design intent and specifications. It is one of the most commonly underestimated activities in Australian construction programs.

The commissioning process typically includes:

**Pre-commissioning (during installation):**
- Inspection and testing of individual components as they are installed (pressure testing of pipes, insulation resistance testing of cables, duct leakage testing)
- Point-to-point verification of control wiring and sensor connections
- These activities should be programmed as part of the services installation activities, not as a separate phase

**Commissioning (after installation complete):**
- Individual system testing (mechanical: air balancing, water balancing, refrigerant charge verification; electrical: switchboard testing, protection relay calibration, emergency lighting duration test; fire: detector sensitivity testing, alarm panel programming, sprinkler flow test; hydraulic: hot water tempering valve testing, backflow prevention testing; lifts: load testing, door timing, fire recall testing)
- Integrated systems testing (BMS point verification, fire mode operation with HVAC shutdown, lift recall, and smoke control; emergency generator load test with essential services changeover)
- Witnessing by authorities (fire brigade, electrical supply authority, water authority)

**Handover:**
- Occupation Certificate application and inspection
- As-built documentation preparation and handover
- Defects list preparation and rectification
- Maintenance information and training for building management
- Warranty documentation

Commissioning duration varies significantly by project complexity:
- Simple residential (Class 1): 1-2 weeks
- Multi-residential apartments (Class 2): 4-8 weeks per building (plus individual apartment inspections)
- Commercial office (Class 5): 6-12 weeks
- Hospital/healthcare (Class 9a): 12-20+ weeks (the most complex commissioning requirements)

Reference: CIBSE Commissioning Codes; ASHRAE Guideline 0 (Commissioning Process); AS/NZS 3000:2018 (Electrical installations — commissioning)
Applies to: All building classes and project types
Related standards: AS 1851-2012 (Fire protection — maintenance); AS 1668.1 (Ventilation — fire and smoke control); National Construction Code — Essential Services provisions

Common pitfalls:
- Not including commissioning as a separate program phase, instead assuming it will happen concurrently with the final weeks of construction (it cannot — systems must be complete before they can be commissioned)
- Underestimating the time required for integrated systems testing, which requires all individual systems to be commissioned before integration can be tested
- Not programming authority witness testing dates early enough, as inspectors have limited availability and may require weeks of lead time
- Assuming commissioning can be compressed to meet the practical completion date — commissioning duration is driven by technical requirements, not commercial pressure

## Practical Completion — Program Perspective

Practical completion is the contractual milestone at which the works are complete (or substantially complete) and fit for their intended purpose, subject only to minor defects that do not prevent occupation or use. Achieving practical completion triggers important contractual consequences: the transfer of risk and insurance responsibility, the release of the first moiety of retention, the commencement of the defects liability period, and the cessation of liquidated damages.

Under AS 4000-1997 (Clause 35), practical completion occurs when:
- The works are complete except for minor omissions and defects that do not prevent the works from being reasonably capable of being used for their intended purpose
- Documents and other things required to be provided by the contractor at or before practical completion have been provided
- All tests and inspections required before practical completion have been completed satisfactorily

The program should include the following activities in the lead-up to practical completion:
1. **Internal completion inspection** (by the contractor's site team): 2-3 weeks before target PC date. Identify and list all incomplete and defective work.
2. **Defect rectification period**: 2-4 weeks for the contractor to address the items identified in the internal inspection.
3. **Superintendent's pre-completion inspection**: 1-2 weeks before target PC date. The superintendent inspects and issues a list of incomplete or defective items.
4. **Final rectification**: The contractor addresses the superintendent's list.
5. **Practical completion inspection**: The superintendent inspects and either issues a certificate of practical completion or advises that practical completion has not been achieved (with reasons).
6. **Authority inspections and certifications**: Occupation Certificate inspection, Essential Services certification, fire safety statement — these must be completed at or before practical completion.

Reference: AS 4000-1997 Clause 35 (Practical completion); AS 2124-1992 Clause 42
Applies to: All building classes and project types
Related standards: Environmental Planning and Assessment Act 1979 (NSW) — Occupation Certificate requirements

Common pitfalls:
- Not programming sufficient time for the practical completion process (inspections, defect rectification, re-inspection), leading to a rushed and contentious handover
- Treating practical completion as a calendar date rather than a milestone that is achieved when specific criteria are met
- Not coordinating authority inspections and certifications with the practical completion timeline — an Occupation Certificate cannot be issued until all required inspections and certifications are complete
- Failing to separate genuine defects (items that must be rectified before PC) from minor items that can be listed for rectification during the defects liability period

## Resource Loading and Levelling

Resource loading assigns labour, equipment, and material quantities to each program activity. Resource levelling then adjusts the program to avoid resource over-allocation — ensuring that the planned resources do not exceed the available capacity at any point in time.

In Australian construction, resource constraints that commonly affect the program include:

**Labour availability:**
- Skilled labour shortages in specific trades (particularly mechanical and electrical trades, and specialised façade installers)
- Enterprise agreement provisions limiting overtime and requiring RDOs
- Seasonal labour competition from concurrent major projects (e.g., infrastructure programs)

**Equipment constraints:**
- Crane availability and capacity (tower crane erection and dismantling are critical program activities)
- Hoist capacity for multi-storey buildings (a single personnel/material hoist can become a bottleneck for a 20+ level building)
- Concrete pump availability for large pours

**Space constraints:**
- Laydown and staging area limitations, particularly on constrained urban sites
- Work face availability — the number of floors or zones where work can proceed simultaneously is limited by the structural sequence, facade progress, and access
- Material storage and handling — just-in-time delivery may be required on sites with limited storage

Resource-loaded programs are not commonly required on all Australian construction projects, but are increasingly requested on large and complex projects (typically >$50M). Resource loading adds significant effort to program preparation but provides valuable information for identifying resource conflicts and bottlenecks.

Reference: AS 6116:2018 (Project Scheduling); Industry best practice
Applies to: All building classes, most relevant for medium-to-large commercial and multi-residential projects

Common pitfalls:
- Preparing a resource-loaded program at the start and never updating it as the project evolves, making the resource data meaningless
- Over-allocating resources in the program to show an aggressive timeline, without verifying that the required labour and equipment are actually available
- Not accounting for the productivity impact of resource constraints — adding a second crane does not double the concrete pour rate if the work face is the constraint
- Ignoring hoist logistics in multi-storey construction, where a single hoist can limit the number of workers that can efficiently access upper floors during peak construction activity

## As-Built Program Records

The as-built program is a record of what actually happened during construction — when activities actually started and finished, as opposed to when they were planned to. Maintaining an accurate as-built record is essential for delay analysis, dispute resolution, and lessons learned for future projects.

Requirements for maintaining as-built records:

**Contemporaneous recording:** Actual start and finish dates should be recorded as activities occur, not reconstructed after the fact. The most reliable source is the daily site diary, supplemented by progress photographs, inspection records, and delivery dockets.

**Program updates:** The construction program should be updated at least monthly (and ideally fortnightly) with actual dates and remaining durations. Each update should be saved as a separate snapshot (not overwritten) to preserve the historical record of program evolution.

**Documentation to support the as-built record:**
- Daily site diaries recording weather conditions, workforce numbers, equipment on site, work activities, and any delay events
- Weekly and monthly progress photographs (time-stamped and annotated with location)
- RFI logs and response dates
- Variation direction dates and completion dates
- Inspection and hold-point records
- Material delivery records for critical items
- Subcontractor work commencement and completion records

**Retention of records:** Under most Australian limitation periods, construction-related claims can be brought up to 6 years after practical completion (or up to 10-15 years for latent defect claims under some state legislation). Program records should be retained for the full limitation period.

Reference: AS 6116:2018 (Project Scheduling — record keeping requirements); Limitation Acts (vary by state)
Applies to: All building classes and project types

Common pitfalls:
- Not saving program update snapshots, making it impossible to reconstruct the state of the program at any given point in time (critical for delay analysis)
- Relying on the contractor's program updates as the sole record without independently verifying progress through inspections and site observations
- Failing to record delay events in the site diary at the time they occur, leading to reliance on memory and reconstruction months or years later
- Discarding program records after practical completion, leaving the principal exposed if claims arise during the defects liability period or beyond
