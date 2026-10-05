# Reusable RFP wording for owner review

Eight draft adaptations of Clerk domain content. All are version 1, status **draft**, intended for RFPs. None is runtime content or owner-reviewed wording. Proposed stable IDs must be checked against the final catalogue before integration.

For each card mark **Keep / Edit / Conditional / Drop**, then record the missing service or boundary. Wording approval does not verify regulatory claims or project inputs. Applicability descriptions await mapping to supported SiteWise predicates; unknown applicability requires review.

## Appointment boundary

**ID:** `cl.rfp-appointment-boundary`

> This appointment covers the services and stages selected in the scope schedule. Identify any additional specialist appointment, client input or contractor-designed element needed to complete those services. Price optional services separately.

- **Applies when:** a consultant appointment has an explicit scope schedule.
- **Excludes:** automatically including every specialist service under a discipline label.
- **Inputs:** selected services, stages and responsibilities.
- **Source:** Clerk fire guide, Services by stage, lines 156–165; hydraulic guide, Consultant request for fee proposal, lines 273–297.
- **Examples:** include for wet-fire design with separate fire engineering; do not add a performance-solution report merely because a fire-services consultant is invited.
- **Owner review:** Is the specialist-role boundary clear enough? Pending.

## Existing information and investigations

**ID:** `cl.rfp-evidence-review`

> Review the issued information for the affected systems and locations. Record the basis relied on, conflicts, missing inputs and proposed investigations. Identify limitations for design and tendering, and submit additional investigation scope for approval before proceeding.

- **Applies when:** existing evidence is part of the appointment's design or assessment basis.
- **Excludes:** authorisation for intrusive work, extra fees or a declaration that supplied records are verified.
- **Inputs:** issued documents, affected systems/parts and review stage.
- **Source:** fire guide, Evidence sweep and Inception; hydraulic guide, Consultant request for fee proposal.
- **Examples:** include for unclear supply information; do not commission a repeat test automatically when a suitable current result exists.
- **Owner review:** Should the memo include an accept/query/reject schedule for supplied records? Pending.

## Water supply and design basis

**ID:** `cl.rfp-water-supply-basis`

> Review available water-supply and system-performance information against the proposed wet-fire scope. Identify any further test needed and define its purpose, conditions and deliverable. State adopted design assumptions and unresolved limitations.

- **Applies when:** wet-fire assessment or design depends on supply/capacity information.
- **Excludes:** an automatic new test, assumed flow/pressure, prescribed standard edition or compliance conclusion.
- **Inputs:** affected systems, available tests and investigation responsibilities.
- **Source:** fire guide, Active systems and water supply and Inception; hydraulic guide, Fire-water boundary.
- **Examples:** include where work changes demand on an existing supply; exclude an unrelated electrical appointment.
- **Owner review:** Who specifies, arranges, performs and pays for testing? Pending.

## Design and tender deliverables

**ID:** `cl.rfp-design-tender-deliverables`

> For the approved scope and procurement route, provide the agreed drawings, calculations, specification and schedules needed for consistent contractor pricing. Identify interfaces, remaining contractor design, assumptions and exclusions. Define testing and completion evidence for the designed work.

- **Applies when:** design/tender documentation is selected.
- **Excludes:** requiring full detailed design where only a performance specification is commissioned; treating preparation as approval.
- **Inputs:** procurement route, scope, design roles and deliverable formats.
- **Source:** fire guide, Documentation and tender; hydraulic guide, Consultant deliverables.
- **Examples:** detailed design for construct-only; bounded performance requirements where design is expressly allocated to a D&C works package.
- **Owner review:** Which deliverables and file formats should be mandatory? Pending.

## Operational continuity and interfaces

**ID:** `cl.rfp-operational-interface`

> Identify how the proposed work affects retained systems and site operations. Define design constraints for staging, temporary arrangements and interruptions, and identify the parties whose input is needed. State which shutdown or impairment submissions require technical review under this appointment.

- **Applies when:** works affect an existing service or operational site.
- **Excludes:** appointing the designer as operator, principal contractor or shutdown approver; promising uninterrupted service.
- **Inputs:** operating constraints, retained systems, access and responsibilities.
- **Source:** fire guide, Existing buildings, refurbishment and remediation; hydraulic guide, Refurbishment and commercial fit-out overlay and Hydraulic trade tender.
- **Examples:** include replacement affecting an operating service; exclude a new isolated installation without such interfaces.
- **Owner review:** What operating information must the client supply for pricing? Pending.

## Construction support allowance

**ID:** `cl.rfp-construction-support`

> Price selected construction support separately. State allowances for submission reviews, RFIs, inspections, test witnessing and close-out review, with deliverables and rates for additional attendance. Identify review limits and the completion statements the consultant is engaged and able to provide.

- **Applies when:** construction support is included or requested as an option.
- **Excludes:** unlimited attendance, blanket certification or transfer of contractor responsibility.
- **Inputs:** selected tasks, programme, attendance and reporting basis.
- **Source:** fire guide, Construction and completion and Consultant RFP minimum returnables; hydraulic guide, Consultant request for fee proposal.
- **Examples:** nominated inspections and witnessed tests; not unpriced continuous supervision.
- **Owner review:** Which allowances should be explicit project inputs? Pending.

## Comparable proposal and fee return

**ID:** `cl.rfp-fee-return`

> Return a fee for each selected stage, with optional services and disbursements identified separately. State the tax basis, team, availability, programme, deliverables, attendance allowances, client inputs, assumptions, exclusions, departures and rates for additional services.

- **Applies when:** comparable consultant proposals are requested.
- **Excludes:** default fees, tax rates, evaluation weightings or contractual terms.
- **Inputs:** stages, return instructions and project commercial requirements.
- **Source:** fire guide, Consultant RFP minimum returnables; hydraulic guide, Consultant request for fee proposal.
- **Examples:** staged fee with optional construction support; never treating the returned proposal as an approved commitment.
- **Owner review:** Is this sufficient without unnecessary tender burden? Pending.

## Close out evidence

**ID:** `cl.rfp-closeout-evidence`

> For selected completion services, review the agreed test results and handover records, identify outstanding items, and provide the stated completion deliverable with its scope and limitations. Record unresolved defects, deferred tests and the party responsible for closing each item.

- **Applies when:** completion review is selected.
- **Excludes:** universal certification, approval to occupy or implied inspection of every system.
- **Inputs:** completion scope, required records, results and outstanding-item owners.
- **Source:** fire guide, Construction verification and integrated testing and Handover and operations; hydraulic guide, Consultant deliverables.
- **Examples:** completion evidence for the upgraded system; not whole-building compliance from a limited appointment.
- **Owner review:** What precise completion deliverable would you accept? Pending.

## Integration and source identity

WP-40a owns selected wording integration, WP-31 responsibility data, WP-41 assembly and WP-42 citations. A sentence is not itself an executable consequence; K2 may reuse its underlying concept after deduplication.

All references above mean `data/seed/fire-life-safety-guide.md` or `data/seed/hydraulic-services-guide.md` in Clerk. Exact hashes and section headings are recorded in `source-manifest.json`. These are paraphrased draft adaptations, not quotations or independently verified technical rules. No regulatory numbers or standard editions have been imported.

Use the approved `knowledge/SCHEMA.md` shape only after the implementation owner confirms placeholders and conditional selection. Preserve IDs and wording versions through conversion. Nothing here is marked reviewed.
