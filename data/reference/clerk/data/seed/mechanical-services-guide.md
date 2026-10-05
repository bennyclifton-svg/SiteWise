---
tier: topic
seed_type: discipline
loaded_by: "discipline: mechanical-services"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, mixed, commercial, industrial, institution]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
state_default: NSW
topics: [mechanical-services, hvac, ventilation, exhaust, smoke-control, bms-controls, commissioning, design-review, consultant-procurement, trade-procurement]
summary: "Cross-lifecycle mechanical-services guidance for project planning, consultant procurement, trade tendering, design review, coordination, construction, commissioning and handover. It routes project evidence and applicable requirements without replacing the responsible mechanical designer, certifier or licensed contractor."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
reviewed_on: 2026-07-25
---

# Mechanical Services Technical Guide

## Purpose and professional boundary

This guide is the shared mechanical-services knowledge source for Clerk. It may
inform a PMP, a consultant request for fee proposal, a mechanical trade tender,
a design review, construction coordination, commissioning, defects review and
handover. A workflow must load only the sections relevant to its purpose.

This guide is platform guidance, not project evidence. It does not design,
approve, certify or warrant a mechanical system. Project-specific conclusions
must come from current project evidence and must be referred to the responsible
mechanical engineer, registered design practitioner, certifier, contractor or
other competent person as applicable.

Do not transfer a requirement between building classes, jurisdictions or
projects without checking the current adopted NCC edition, state variations,
referenced-standard editions, approval conditions and project specification.
Where a standard is not available in full, record the verification gap rather
than paraphrasing a requirement from memory.

## When to activate this guide

Activate the guide when any of the following applies:

- the user requests mechanical, HVAC, ventilation, air-conditioning, exhaust,
  smoke-control, stair-pressurisation, refrigeration or BMS/controls work;
- the project profile selects `mechanical_hvac`, `bms_controls`,
  `car_parking`, `fire_services` or a services-upgrade scope that has a
  mechanical interface;
- an uploaded document is identified as a mechanical drawing, specification,
  schedule, calculation, report, commissioning plan, test result, shop drawing
  or technical submittal;
- the brief, PPR, authority material or consultant advice identifies mechanical
  performance, indoor-air-quality, acoustic, energy, smoke-control or
  commissioning requirements;
- the PMP needs a mechanical strategy, responsibility allocation, programme
  dependency, risk, procurement action or handover requirement.

Do not activate the entire guide merely because a building contains a domestic
split system or exhaust fan. Select the smallest relevant section set.

## Source hierarchy and evidence discipline

For project-specific work, use this order:

1. Current approved project profile and recorded owner decisions.
2. Current project evidence: brief/PPR, drawings, specifications, approvals,
   conditions, consultant reports, schedules, calculations and correspondence.
3. Responsible-consultant advice and current regulated-design records.
4. Applicable platform guidance from this guide and related seeds.
5. General model knowledge only when the preceding sources do not answer the
   question, labelled as low-confidence guidance requiring verification.

Never convert platform guidance into a project fact. If the project evidence
does not identify a system, state that it is unresolved and ask the responsible
party to confirm it.

## Mechanical evidence sweep

Before preparing or reviewing a mechanical artefact, look for:

- project title, address, client, jurisdiction, building classifications,
  storeys, uses, occupancy assumptions and project stage;
- brief/PPR mechanical requirements, performance criteria, design
  responsibility matrix and departures;
- architectural plans, reflected ceiling plans, sections, elevations, roof and
  basement plans, room data, finishes and facade information;
- fire strategy, fire-engineering report, smoke-control matrix and cause-and-
  effect requirements;
- acoustic criteria and plant-noise, duct-noise, vibration and neighbour-noise
  constraints;
- BASIX, NatHERS, Section J or other energy and sustainability commitments;
- electrical supply, switchboard, emergency-power and controls interfaces;
- hydraulic drainage, condensate, tundish, trade-waste and water-service
  interfaces;
- structural loads, plant platforms, seismic restraint where applicable,
  penetrations, openings and supports;
- civil, planning and authority constraints affecting external plant,
  discharges, louvres, roof equipment and public-domain interfaces;
- existing-services surveys, condition reports, hazardous-material information
  and operational constraints for refurbishment work;
- consultant appointments, tender addenda, RFIs, submittals, inspections,
  commissioning records, O&M information and defects.

Record document number, title, author, revision, date and issue purpose. Do not
rely on superseded or construction-inappropriate issues without an explicit
warning.

## System map

Use the system map to decide which technical modules apply. Do not assume every
project requires every system.

### Heating and cooling

Confirm:

- spaces requiring heating or cooling and the operative comfort criteria;
- occupancy, operating hours, internal loads, envelope and climate inputs;
- system options, zoning, diversity, part-load operation and future flexibility;
- indoor and outdoor plant locations, access, replacement paths and screening;
- heat rejection, refrigerant, condensate and drainage arrangements;
- electrical demand, controls, metering, energy commitments and standby
  requirements;
- acoustic, vibration, facade, structural and maintenance interfaces.

The responsible designer must perform and retain the calculations appropriate
to the selected system. Do not infer equipment capacity from floor area alone.

### Outdoor air and indoor air quality

Confirm:

- which spaces use natural ventilation, mechanical outdoor air or a combined
  strategy;
- occupancy and contaminant sources;
- outdoor-air intake location and separation from exhausts, traffic, cooling
  towers and other contamination sources;
- filtration, air distribution, transfer paths and pressure relationships;
- controls, monitoring, after-hours operation and maintainability;
- the interaction between openable windows, acoustic constraints, facade
  design, security and weather exposure.

Room air-conditioning is not necessarily ventilation. A recirculating split
system does not by itself demonstrate an outdoor-air solution.

### Local exhaust

Test whether the project needs separate exhaust strategies for:

- bathrooms, ensuites, toilets and laundries;
- domestic or commercial kitchens;
- cleaners' rooms, refuse rooms and bin stores;
- car parks, loading areas and vehicle-related spaces;
- plant rooms, battery or charging areas, workshops, laboratories or other
  specialist contaminant sources;
- clothes dryers and other equipment requiring ducted discharge.

Confirm make-up air, transfer air, discharge location, odour migration,
condensation, fire/smoke dampers, access for cleaning and facade/roof
penetrations. Do not discharge contaminated air into concealed building spaces.

### Car-park and enclosed-vehicle areas

Where applicable, establish:

- whether the area qualifies as naturally ventilated or needs mechanical
  ventilation;
- normal ventilation, pollutant monitoring and control strategy;
- fire and smoke operating modes and their design responsibility;
- intake, exhaust and discharge locations;
- fan, duct, shaft, louvre, acoustic, vibration and power requirements;
- interfaces with security doors, loading operations, EV infrastructure,
  sprinklers, detection, fire control and emergency procedures;
- testing, functional verification and ongoing access for maintenance.

### Smoke control and fire interfaces

Mechanical fire and smoke functions may include smoke exhaust, stair or lobby
pressurisation, zone shutdown, damper operation, fire-mode fan operation and
interfaces with the fire-indicator panel.

For every function, identify:

- the fire engineer or other party defining the performance requirement;
- the mechanical party designing the equipment and air paths;
- the electrical/fire-controls party implementing cause and effect;
- the party providing power, monitoring and status indication;
- the certifier, testing authority and witnessing requirements;
- the integrated-testing evidence required before occupation.

Do not let a mechanical drawing silently become the fire strategy. Resolve
conflicts through the design responsibility matrix.

### Controls, BMS and metering

Define:

- systems and points to be controlled or monitored;
- sensors, set points, schedules, safeties, alarms and overrides;
- standalone controls versus BMS integration;
- interfaces with fire mode, access control, metering and remote monitoring;
- trend logging and data needed to verify performance;
- commissioning, seasonal tuning, training and controls documentation;
- cybersecurity or network responsibility where controls connect to project or
  owner networks.

Controls descriptions must be testable. Avoid requirements such as "complete
BMS controls" without a points list, sequence of operation and responsibility
allocation.

### Refrigerant, condensate and water interfaces

Confirm:

- refrigerant type and system arrangement;
- pipe routes, shafts, penetrations, insulation and protection;
- occupied-space concentration or leak-management considerations where
  applicable;
- condensate collection, falls, traps, tundishes and discharge points;
- waterproofing and facade interfaces;
- access for leak testing, service and replacement;
- coordination with hydraulic, electrical and structural design.

### Existing buildings and live environments

For refurbishment or remediation:

- verify existing equipment, capacity, condition, controls and remaining life;
- undertake intrusive investigation where concealed routes or plant condition
  materially affect the design;
- confirm shutdowns, temporary services, isolation, staging and occupant
  impacts;
- identify hazardous materials and infection, dust, noise or operational
  controls;
- separate retained, relocated, modified, removed and new work;
- do not assume existing systems are compliant, balanced or suitable for added
  loads merely because they are operating.

## Building-class and project overlays

### Class 1 and Class 10 residential work

Select only the systems justified by the brief and building arrangement.
Coordinate wet-area and kitchen exhaust, air-conditioning, outdoor plant,
condensate, electrical capacity, roof/wall penetrations, acoustic impacts and
energy commitments. Load `mep-residential.md` for domestic system literacy, but
retain mechanical design and load calculations as specialist scope.

### Class 2 and mixed-use residential work

Also consider:

- apartment natural-ventilation commitments and internal-room exhaust;
- common-area conditioning and ventilation;
- basement or enclosed-car-park ventilation;
- retail or commercial tenancy provisions and base-building interfaces;
- central versus individual systems and strata/common-property consequences;
- facade appearance, condenser locations, risers, metering and access;
- fire/smoke interfaces and integrated systems testing;
- regulated-design, professional-engineering and declaration obligations in
  the applicable jurisdiction;
- commissioning, as-built and building-manual information needed by the owner
  or owners corporation.

Load `multi-residential-apartments-guide.md` for Class 2 and mixed-residential
context. Do not use Class 1 ventilation provisions as the design basis for a
Class 2 project.

### Commercial, industrial and institutional work

Confirm tenancy and process loads, hours of operation, after-hours use,
critical-space requirements, landlord/tenant boundaries, outside-air strategy,
central plant, heat rejection, controls, metering, maintainability and
commissioning. Load the relevant archetype seed and obtain specialist input for
process, laboratory, health, clean-room, hazardous-area or mission-critical
systems.

## PMP contribution

A PMP mechanical-services contribution should be concise and project-specific.
It should record:

- confirmed project requirements and their evidence;
- unresolved system or performance decisions;
- mechanical design responsibility and appointment status;
- required design inputs and multidisciplinary interfaces;
- design stages, review gates, approval submissions and construction-release
  dependencies;
- procurement strategy and long-lead items;
- key risks, assumptions, decisions and actions;
- inspection, commissioning, handover and operational-readiness requirements.

Do not insert this entire guide into a PMP. Include only the controls needed to
manage the project.

## Services by stage

> **AGENT REVIEWED FOR LOCAL USE (Codex, 27 September 2026); owner review deferred.** Stage ids
> match `architectural-services-guide.md` (SEED_MAP owner decision Q3).

Each stage opens with an **RFP line** that code can render as the lead of a
grouped RFP item. Use only the stages inside the appointment boundary; price
the rest separately or leave them out.

### Inception, site and existing information [inception]

**RFP line:** Review the brief, the site and existing mechanical plant and
records, and identify the constraints, investigations and inputs the
mechanical design needs.

- Attend a site visit; for existing buildings, inspect plant, risers, roof
  and plant rooms, and record condition and remaining life.
- Review existing drawings, maintenance records, the fire safety schedule and
  any existing smoke control or air handling fire modes.
- Confirm the design criteria to be agreed with the client: indoor
  conditions, occupancy, hours, redundancy and tenancy provisions.
- List the investigations needed (plant condition survey, acoustic
  measurement, asbestos in plant areas) and their programme dependencies.

### Concept and schematic design [concept]

**RFP line:** Prepare mechanical system options and space allowances for the
schematic design, with energy, cost and plant-space implications.

- Compare system options (for example VRF, chilled water, packaged units)
  against the brief, energy targets, maintenance and capital cost.
- Provide plant room, roof plant, riser and ceiling-void allowances to the
  architect and structural engineer, including plant weights.
- Identify outdoor air, exhaust and discharge locations and the related
  AS 1668.2 and separation constraints.
- Provide mechanical information for the cost planner's estimate.

### Pre-DA consultation and development application [development_application]

**RFP line:** Provide the mechanical inputs the development application
needs and respond to council requests on them until determination.

- Plant locations, screening and discharge points for the DA drawings.
- Plant noise data to the acoustic consultant, and mechanical input to the
  ESD or sustainability report.
- Respond to council requests for information on mechanical matters. State
  whether this is in the lump sum or at hourly rates.

### Conditions of consent and design development [design_development]

**RFP line:** Review the conditions of consent and develop the mechanical
design, coordinated with the consultant team, to the level the procurement
route requires.

- Identify the conditions that affect the mechanical design (noise limits,
  plant screening, exhaust locations).
- Load calculations, system selection, plant schedules and coordinated
  duct and pipe routes.
- NCC Section J mechanical compliance, and fire-mode and smoke control design
  (AS 1668.1) coordinated with the fire engineer and BCA consultant.
- Update information for the cost plan.

### Documentation and tender [documentation_tender]

**RFP line:** Prepare mechanical documentation for the chosen procurement
route and support the tender through queries, addenda and tender review.

- Traditional route: construction certificate and tender drawings,
  specification and equipment schedules.
- Design and construct route: a performance specification stating the design
  responsibility the contractor takes over.
- Respond to tenderer RFIs, prepare addenda, and review mechanical tenders or
  the mechanical content of head contract tenders.
- Where the Design and Building Practitioners Act 2020 applies (confirm the current project-specific application with the responsible practitioner), state who provides the
  regulated design declarations.

### Novation and post-contract design [post_contract_design]

**RFP line:** Price, as a separate option, novation to the design and
construct contractor and the post-contract mechanical design.

- Novation terms, the fee basis after novation and the transfer of design
  responsibility.
- Construction documentation, shop drawing and equipment submittal review,
  and design queries.

### Construction and completion [construction_completion]

**RFP line:** Provide construction-phase mechanical services: inspections at
stated hold points, submittal and RFI responses, witnessing of commissioning,
and completion and defects support.

- Site inspections at stated hold points (for example before ceilings close
  and at plant installation) with written reports.
- Witness commissioning, air and water balancing, and fire-mode and smoke
  control testing; review the results.
- Provide the design certificates and essential fire safety measure
  statements the certifier needs for the occupation certificate.
- Review operation and maintenance manuals and as-built drawings; attend
  defects inspections.

### Throughout the appointment [coordination]

**RFP line:** Attend design meetings, coordinate the mechanical design with
the other disciplines and report on design status throughout the
appointment.

- Attend meetings at a stated frequency; take part in services coordination
  and clash review.
- Keep a mechanical deliverables schedule and record design changes with
  their cost and programme effect.

## Consultant request for fee proposal

An RFP for a mechanical services engineer should ask the consultant to:

1. Review the current evidence and identify missing inputs before accepting the
   design basis.
2. Confirm systems, performance criteria, design assumptions, calculations and
   applicable requirements.
3. Define scope and deliverables by project stage.
4. Coordinate architecture, structure, facade, fire, acoustic, electrical,
   hydraulic, civil, energy, controls and vertical-transport interfaces.
5. State regulated-design, professional-engineering, certification and
   lodgement responsibilities.
6. Include meetings, workshops, design reviews, authority responses, tender
   support, RFIs, submittals, inspections and defects allowances.
7. Define commissioning, witness testing, integrated testing, training,
   as-builts and O&M review.
8. State exclusions, optional services, reliance on other consultants,
   programme, personnel, fees, rates, expenses and required client inputs.

Request a responsibility schedule whenever design may be split between the
consultant, architect, contractor, supplier, fire engineer or controls
specialist.

## Mechanical contractor tender

A mechanical trade tender procures construction work, not merely professional
advice. Establish whether the package is:

- construct-only to completed consultant documents;
- design-and-construct to a performance brief;
- contractor-designed only for nominated portions;
- a refurbishment package relying on verified existing conditions.

The tender scope should address, where applicable:

- design completion, calculations, shop drawings and certifications;
- supply, delivery, storage, installation, supports, plinths, fixings,
  penetrations, sleeves, sealing, fire stopping and builders' work;
- plant, ductwork, pipework, insulation, grilles, louvres, dampers, controls,
  sensors, switchboards and interfaces;
- temporary works, access, cranage, shutdowns, protection and staging;
- samples, technical submittals, coordination drawings and BIM/model
  requirements;
- testing, flushing or cleaning where applicable, pressure/leak testing,
  balancing, controls verification and integrated systems testing;
- commissioning records, defects attendance, training, warranties, spares,
  O&M manuals and as-built documents;
- exclusions, provisional allowances, interfaces and scope demarcations.

Never use an engineer RFP as the trade scope without rewriting responsibility,
supply, installation, testing, warranty and commercial requirements.

## Mechanical design review

Confirm the review purpose before starting: brief alignment, phase gate,
authority readiness, tender readiness, construction release, coordination,
revision comparison or risk review.

### Document control

Check document numbers, titles, authors, revisions, dates, issue purpose,
status, drawing list and superseded information. Identify whether calculations,
specifications and schedules correspond to the drawing revision.

### Brief and design basis

Check that the design records:

- required spaces, uses, occupancy and operating assumptions;
- internal and external design conditions;
- system selection and zoning;
- performance, acoustic, energy and maintainability criteria;
- project-specific constraints, owner decisions and departures;
- information relied upon and unresolved assumptions.

### System completeness

Check applicable systems for:

- heating and cooling;
- outdoor air and natural-ventilation coordination;
- local exhaust and make-up air;
- car-park, plant-room and specialist ventilation;
- smoke control and fire-mode operation;
- controls, metering and BMS interfaces;
- refrigerant, condensate and drainage;
- plant access, replacement and maintenance.

### Coordination

Check:

- ceiling, riser, shaft, roof, basement and plant-space allowances;
- duct, pipe, louvre and equipment routes;
- structural loads, supports, penetrations and seismic restraint where
  applicable;
- facade, waterproofing and weathering details;
- fire-rating, smoke-control and fire-stopping interfaces;
- electrical supplies, controls and emergency power;
- hydraulic drainage and tundish requirements;
- acoustic treatment, vibration isolation and neighbour impacts;
- access panels, service clearances and replacement paths.

### Compliance and responsibility

Record the applicable NCC edition and state variations nominated by the project
team, the referenced standards relied upon, approval or performance-solution
inputs, responsible practitioners, declarations and certification deliverables.
Treat missing or conflicting responsibility as a review finding.

### Constructability and commissioning readiness

Check whether the documents define installation tolerances, access, isolation,
testing, balancing, controls sequences, commissioning acceptance criteria,
integrated testing, training, O&M and as-built requirements.

Classify findings as:

- confirmed coordination error;
- missing information;
- inconsistent information;
- unresolved design decision;
- compliance-verification item;
- constructability or maintainability risk;
- commissioning or handover gap;
- no issue identified within the stated review scope.

Every finding must cite the relevant project document, sheet, page or section.
Platform guidance may explain why an item matters but is not the evidence for
the finding.

## Construction and delivery controls

Before installation, verify:

- current construction-issued design and any regulated-design requirements;
- approved submittals, shop drawings, samples and equipment selections;
- coordinated openings, supports, plant bases, access and replacement routes;
- builder, consultant, contractor and supplier responsibilities;
- inspection and test plan, witness points and hold points;
- programme, procurement status, shutdowns and temporary-service arrangements.

During installation, record evidence for concealed work, supports, fire
stopping, insulation, duct cleanliness, pressure/leak tests, access,
identification, controls installation and departures from the design.

Do not permit an undocumented site variation to substitute for an updated
design where the change affects a regulated design, performance solution,
certification, fire/smoke function or other material design requirement.

## Commissioning and handover

Commissioning requirements should be defined before tender and progressively
verified, not assembled at project completion.

The project-specific commissioning plan should identify:

- systems and components to be commissioned;
- prerequisites and construction-completion checks;
- responsible parties and witnessing requirements;
- test instruments and calibration evidence;
- air and water balancing where applicable;
- controls point-to-point checks, sequences, alarms and safeties;
- normal, after-hours, fire, failure and emergency operating modes;
- integrated systems testing;
- performance acceptance criteria and treatment of failed results;
- seasonal or deferred testing;
- training, demonstrations, O&M manuals, as-builts, warranties and asset data;
- defects, retesting and final acceptance.

Do not equate equipment start-up with system commissioning.

## Common failure modes

- Mechanical scope is described only as "design HVAC" or "complete mechanical
  services" without systems, stages, interfaces or deliverables.
- Natural ventilation, outdoor air and room air-conditioning are treated as the
  same function.
- Exhaust is shown without make-up air, discharge coordination or cleaning
  access.
- Condensers or plant are selected without acoustic, structural, facade,
  replacement or maintenance coordination.
- Fire/smoke sequences are split across disciplines without one coordinated
  cause-and-effect basis.
- Penetrations, dampers, fire stopping and access panels are resolved after
  structure or ceilings are complete.
- Contractor design responsibility is implied by shop drawings but not stated
  contractually.
- Regulated-design variations are installed before updated design and
  declaration requirements are resolved.
- Controls sequences and commissioning acceptance criteria are deferred until
  the end of construction.
- O&M manuals and as-built drawings reproduce tender information rather than
  recording the installed system.

## Related platform knowledge

Select only what applies:

- `mep-residential.md` — Class 1 domestic MEP literacy.
- `multi-residential-apartments-guide.md` — Class 2 and mixed-residential
  services, fire, energy, quality and delivery context.
- `commercial-construction-guide.md` — commercial HVAC and landlord/tenant
  interfaces.
- `sustainability-energy-guide.md` — BASIX, NatHERS and energy commitments.
- `trade-interfaces-coordination-guide.md` — sequencing and trade interfaces.
- `as-standards-reference.md` — standards discovery; verify the current adopted
  edition and obtain the standard before relying on it.
- `ncc-reference-guide.md` — NCC discovery; verify current jurisdictional
  adoption and state variations.
- `procurement-quoting-guide.md` and `procurement-tendering-guide.md` —
  procurement and returnable controls.
- `defects-and-dlp-guide.md` — defect close-out and DLP controls.

## Current authoritative verification points

At the review date in the frontmatter:

- NCC Volume One Part F6 addresses light and ventilation, including mechanical
  ventilation and local exhaust provisions:
  `https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/f-health-and-amenity/part-f6-light-and-ventilation`.
- NCC Volume One Part J6 addresses energy-efficiency controls for
  air-conditioning and ventilation:
  `https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/j-energy-efficiency/part-j6-air-conditioning-and-ventilation`.
- NSW currently regulates design and professional-engineering work for Class 2,
  certain Class 3 and Class 9c buildings, including mixed-use buildings with a
  regulated part. Mechanical-design authority includes HVAC, air distribution,
  smoke control, exhaust and stair pressurisation:
  `https://www.nsw.gov.au/housing-and-construction/compliance-and-regulation/professionals-working-on-regulated-buildings/building-classes-and-roles`.
- NSW design-practitioner obligations require regulated designs and relevant
  declarations to be coordinated and lodged before associated building work
  starts:
  `https://www.nsw.gov.au/housing-and-construction/compliance-and-regulation/professionals-working-on-regulated-buildings/design-and-building-practitioners/design-obligations`.

Verify these sources again when the NCC, referenced standards, state adoption or
practitioner scheme changes.

## Low-confidence and escalation flags

Escalate rather than answer from general guidance when:

- equipment capacity, airflow, pressure, heat load, smoke-control performance
  or other engineering calculations are requested;
- a design appears inconsistent with the NCC, a standard, approval condition,
  fire strategy, energy certificate or regulated-design declaration;
- drawings do not identify the responsible designer or current issue status;
- a proposed substitution affects compliance, energy, acoustics, controls,
  fire/smoke operation, plant access or commissioning;
- the available seed is for a different building class or project type;
- the current adopted code or referenced-standard edition has not been
  verified;
- the user asks Clerk to approve, certify, sign or warrant a mechanical design.
