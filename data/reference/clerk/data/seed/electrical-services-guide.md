---
tier: topic
seed_type: discipline
loaded_by: "discipline: electrical-services"
applies_to_classes: [residential, commercial, industrial]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
state_default: NSW
topics: [electrical-services, utility-supply, distribution, lighting, emergency-power, generation, ev-charging, metering, controls, commissioning, design-review, consultant-procurement, trade-procurement]
summary: "Cross-lifecycle electrical-services guidance for supply, distribution, lighting, emergency power, generation, metering, controls, construction, testing and handover. It manages interfaces and evidence without replacing the responsible electrical designer, utility, certifier or licensed electrician."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
status: seed
author: agent
reviewed_on: 2026-07-26
---

# Electrical Services Guide

## Purpose and professional boundary

Use this guide for electrical project planning, consultant procurement, trade
tendering, design review, coordination, construction, commissioning and
handover.

This guide is platform guidance, not project evidence. It does not calculate
maximum demand, select protection, establish fault levels, design earthing or
lightning protection, assess a hazardous area, approve utility connections or
certify electrical work. Those duties belong to the responsible electrical
engineer, network service provider, accredited service provider, certifier and
licensed electrical contractor as applicable.

Check the current NCC, NSW service and safety legislation, utility rules,
approval conditions, project specification and applicable standards. If a
standard or utility determination is not available, record the verification gap
instead of inventing a requirement.

## Activation

Activate the relevant sections when the project includes:

- a new, altered or augmented utility supply;
- main switchboards, distribution boards, submains or major reticulation;
- lighting, emergency lighting or exit-sign interfaces;
- standby generation, UPS, batteries or critical-power systems;
- solar generation, electric-vehicle charging or demand management;
- metering, controls, BMS, power monitoring or tenant billing interfaces;
- earthing, lightning, surge or power-quality requirements;
- refurbishment, live switchboards, shutdowns or temporary power;
- electrical consultant or trade procurement and commissioning.

Do not activate the whole guide for a minor isolated item. Match scope to
evidence and risk.

## Electrical evidence sweep

Seek:

- project brief, room data, operating hours and critical-load criteria;
- utility correspondence, connection offers, point-of-attachment information
  and metering requirements;
- existing single-line diagrams, switchboard schedules, test records, bills,
  interval data and condition reports;
- architectural, equipment, fire, mechanical, hydraulic, security, ICT and
  controls information;
- equipment loads, starting characteristics, diversity and future capacity
  assumptions;
- hazardous-area, dangerous-goods, fire-engineering and emergency-response
  information where applicable;
- sustainability commitments for lighting, metering, electrification,
  generation and EV infrastructure;
- tenancy, landlord, strata, operator and vendor responsibility boundaries;
- commissioning, maintenance, asset-data and training requirements.

Record document revisions and distinguish measured data, utility-approved data,
designer calculations, vendor estimates and assumptions.

## Supply, demand and resilience

The responsible designer should establish:

- existing and forecast demand, diversity and load profile;
- utility capacity, application pathway and augmentation dependencies;
- supply voltage, point of connection, substations and easements where relevant;
- normal, essential, life-safety and critical load categories;
- acceptable interruption, recovery and maintenance modes;
- standby generation, UPS, battery or alternative supply where justified;
- expansion capacity and the physical pathway for future work.

Resilience language must be testable. Terms such as "N+1", "critical" or
"generator backed" require defined system boundaries, operating modes,
autonomy, failure scenarios and acceptance tests.

## Distribution, protection and access

Coordinate switchroom and board locations, clearances, fire separation,
ventilation, flood exposure, access, maintainability and replacement paths.
Resolve cable routes, risers, penetrations, containment, segregation, supports
and interfaces before construction issue.

Protection, discrimination, arc-energy risk, earthing and fault-level studies
must be completed by competent designers where applicable. The PMP should track
required studies and approval gates, not attempt to state their result.

For existing buildings, verify rather than assume spare capacity, labelling,
board condition, circuit origin, containment capacity or compliance.

## Lighting and controls

Define:

- visual-task, amenity, external and specialist lighting intent;
- controls zones, occupancy, daylight, schedules and override needs;
- emergency-lighting and exit-sign design responsibility and test requirements;
- facade, landscape, security, planning and neighbour interfaces;
- maintainability, access, replacement and standardisation;
- metering and energy-performance evidence.

Coordinate controls across electrical, mechanical, security, AV and BMS
systems. A functional description should identify points, sequences, alarms,
permissions, fail states and the party responsible for integrated testing.

## Generation, storage and EV infrastructure

Where applicable, establish:

- operational objective and connection arrangement;
- structural, roof, fire, access, ventilation and weatherproofing interfaces;
- network approval, export control and metering requirements;
- charging demand, diversity, load management and user allocation;
- isolation, emergency response, signage and maintenance arrangements;
- communications, monitoring, cybersecurity and data ownership;
- commissioning and handover evidence.

Do not use a nominal charger, inverter or battery rating as proof of utility or
building capacity.

## Life-safety and specialist interfaces

Create an interface matrix for fire alarm, occupant warning, emergency
lighting, smoke control, fire pumps, lifts, security release, emergency power
and the fire-control centre. The fire strategy or cause-and-effect matrix should
define required behaviour; electrical documentation implements coordinated
functions but must not silently become the fire strategy.

For industrial or specialist facilities, obtain competent advice for hazardous
areas, process shutdown, high-voltage systems, power quality, static control,
special earthing or mission-critical distribution. Flag these as specialist
scope, not generic electrical design.

## Refurbishment and live work

Before alteration:

- survey and trace existing supplies, circuits, loads and isolation points;
- assess switchboard, protective-device and cable condition;
- reconcile drawings with labels and field observations;
- identify retained, relocated, removed and new work;
- develop shutdown, temporary-power, communication and rollback plans;
- protect operations, residents, tenants, alarms, access and life-safety
  systems;
- coordinate hazardous-material and intrusive-work controls.

Shutdowns should identify authority, notice, test, acceptance and restoration
evidence. Never rely on a label alone to prove isolation.

## Services by stage

> **AGENT REVIEWED FOR LOCAL USE (Codex, 27 September 2026); owner review deferred.** Stage ids
> match `architectural-services-guide.md` (SEED_MAP owner decision Q3).

Each stage opens with an **RFP line** that code can render as the lead of a
grouped RFP item. Use only the stages inside the appointment boundary; price
the rest separately or leave them out.

### Inception, site and existing information [inception]

**RFP line:** Review the brief, the site and the existing electrical supply
and installation, and identify the supply constraints and investigations the
design needs.

- Attend a site visit; record the existing supply, main switchboard,
  substation or kiosk, and the condition of the installation.
- Obtain supply information from the distribution network service provider
  (Ausgrid, Endeavour Energy or Essential Energy in NSW).
- Estimate the likely maximum demand and flag an early risk of supply
  upgrade or a new substation.
- List investigations needed (thermal imaging, load logging, asbestos in
  switchboards).

### Concept and schematic design [concept]

**RFP line:** Prepare the electrical supply strategy and space allowances for
the schematic design, with the cost and programme risk of any supply upgrade.

- Supply strategy: connection point, substation or kiosk location and size,
  main switchroom and riser allowances.
- Identify electrification, solar PV, battery storage and EV charging
  requirements from the brief, the planning controls and the NCC.
- Provide electrical information for the cost planner's estimate, including
  authority and contestable works allowances.

### Pre-DA consultation and development application [development_application]

**RFP line:** Provide the electrical inputs the development application needs
and respond to council requests on them until determination.

- Substation or kiosk location, easements and external lighting for the DA
  drawings.
- Input to the ESD report (PV, EV, electrification commitments).
- Respond to council requests for information on electrical matters.

### Conditions of consent and design development [design_development]

**RFP line:** Review the conditions of consent and develop the electrical
design, coordinated with the consultant team, to the level the procurement
route requires.

- Maximum demand calculation (AS/NZS 3000), distribution and switchboard
  design, and protection and discrimination approach.
- Lighting design (AS/NZS 1680), emergency and exit lighting (AS/NZS 2293),
  and lighting controls to NCC Section J.
- Lodge the connection application with the network provider and engage
  (or coordinate) an Accredited Service Provider Level 3 designer for
  contestable works.
- Coordinate fire, security and ICT power and interfaces.

### Documentation and tender [documentation_tender]

**RFP line:** Prepare electrical documentation for the chosen procurement
route and support the tender through queries, addenda and tender review.

- Traditional route: drawings, specification, schedules and single-line
  diagrams.
- Design and construct route: a performance specification and the design
  responsibility the contractor takes over.
- Respond to tenderer RFIs, prepare addenda and review electrical tenders.

### Novation and post-contract design [post_contract_design]

**RFP line:** Price, as a separate option, novation to the design and
construct contractor and the post-contract electrical design.

- Novation terms and fee basis; shop drawing and switchboard submittal
  review; design queries.

### Construction and completion [construction_completion]

**RFP line:** Provide construction-phase electrical services: inspections at
stated hold points, submittal and RFI responses, witnessing of testing, and
completion and defects support.

- Site inspections at stated hold points with written reports.
- Witness testing of switchboards, emergency lighting and generator or
  battery systems; review test results and certificates of compliance for
  electrical work.
- Provide the design certificates the certifier needs for the occupation
  certificate; review O&M manuals and as-built drawings; attend defects
  inspections.

### Throughout the appointment [coordination]

**RFP line:** Attend design meetings, coordinate the electrical design with
the other disciplines and manage the network provider process throughout the
appointment.

- Attend meetings at a stated frequency; take part in services coordination.
- Track network provider applications, offers and connection dates in the
  approvals tracker.

## Consultant RFP minimum returnables

Request:

- scope understanding, exclusions and information relied on;
- survey, testing and load-data requirements;
- design basis, calculations and studies to be produced;
- utility and accredited-provider services;
- drawings, specifications, schedules and controls descriptions by stage;
- equipment, spatial and multidisciplinary interface schedules;
- sustainability, metering and resilience deliverables;
- construction submittal, RFI and inspection services;
- commissioning plan, witnessing, results review and defects support;
- as-built, O&M, training and asset-data requirements;
- programme, team, registrations, insurance, fee breakdown and variation
  triggers.

Trade RFPs should additionally define design portions, builder's work,
temporary power, utility works, testing, certification and integrated systems
responsibilities.

## Construction, commissioning and handover

Hold points should address incoming supply, concealed containment, earthing,
switchboard factory tests where specified, labelling, fire stopping, protection
settings, energisation and life-safety interfaces.

Commissioning should cover inspection and test records, point-to-point checks,
functional sequences, metering, generation, resilience and integrated systems
under defined modes. Capture settings, certificates, updated single-line
diagrams, circuit schedules, test results, asset data, warranties, spares,
training and shutdown instructions.

## PMP contribution

State confirmed supply and load evidence, utility pathway, design
responsibility, critical and life-safety systems, refurbishment controls,
multidisciplinary interfaces, long-lead items, approvals, construction hold
points, energisation governance, commissioning and unresolved assumptions.

## Common failures and low-confidence flags

Common failures are late utility applications, assumed spare capacity,
unverified existing circuits, switchrooms without access or replacement paths,
controls without sequences, fragmented life-safety interfaces, undocumented
shutdowns, premature energisation and incomplete settings or test records.

Flag low confidence when utility advice, existing single-line diagrams, demand
data, equipment loads, fire strategy, critical-load definition, hazardous-area
advice, shutdown windows or commissioning criteria are missing.
