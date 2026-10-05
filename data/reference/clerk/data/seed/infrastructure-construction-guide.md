---
tier: overlay
seed_type: class-guide
loaded_by: "building_class: infrastructure"
applies_to_classes: [infrastructure]
applies_to_work_types: [new, refurb, extend, remediation]
state_default: NSW
topics: [infrastructure, rail, roads, utilities, energy, ports, airports, possessions, accreditation, linear-works, live-network, commissioning]
summary: "Infrastructure overlay for roads, rail/metro, water/utilities, energy, ports, airports and telecommunications. It governs authority and network-operator interfaces, possessions and outages, safety accreditation, linear works, live-network construction, commissioning into an operating asset, and handover without treating one subtype as representative of all others."
required_by: {create-pmp: 1}
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §register-discipline, §escalation-triggers]
status: seed
author: agent
reviewed_on: 2026-08-14
---

# Infrastructure Construction Guide

## Purpose and professional boundary

Use this guide as the infrastructure-class overlay for project planning and
consultant procurement. Infrastructure is a family of network and linear
assets, not a building type. A road upgrade, rail station, treatment plant,
solar farm, wharf, terminal expansion and tower rollout have different
authorities, access regimes, safety accreditation, commissioning tests and
operational consequences.

This guide is platform guidance, not project evidence. It does not classify
an asset, grant a possession, issue rail safety accreditation, confirm a
grid connection, certify compliance or warrant operability. It does not
invent authority acceptance. Those conclusions belong to the responsible
designers, network operators, regulators, certifiers, contractors and asset
owner. Verify the current NCC edition, NSW legislation, planning controls,
operator standards and approval conditions for the actual project. Do not
treat NCC deemed-to-satisfy provisions as the governing instrument for a
linear or network asset.

Common pitfalls:
- treating a rail station, road corridor or treatment plant as a building
  project governed by NCC pathways;
- citing this overlay as proof that an authority has accepted a solution;
- transferring building commissioning language onto a live network.

## Activation and subtype control

Activate this guide when the project class is `infrastructure` and the work
type is `new`, `refurb`, `extend` or `remediation`. Before using
subtype-specific guidance, identify the subtype from evidence:
roads/highways, rail/metro, water/utilities, energy/renewables,
marine/ports, airports, telecommunications, or other.

If the subtype is missing, mixed or novel, record it as an early decision.
Do not transfer possession, accreditation or commissioning assumptions
across subtypes. A track possession is not a road occupancy, a berth
window or an airside night-only window. Specialist regulated networks
require a project-specific operator reference; label guidance low
confidence until obtained. Default jurisdiction is NSW unless project
evidence names another state.

Common pitfalls:
- activating rail possession and rail safety accreditation language on a
  road, energy or water project;
- treating `other` (for example a trunk main) as generic civil work
  without naming the network operator.

## Evidence sweep

Before writing a PMP or consultant RFP, seek the operational brief and
continuity requirements; surveys, geotechnical data, services, easements,
tenure and authority information; operator standards and interface
drawings; possession, occupancy, outage and traffic-management calendars;
land acquisition, native title and landholder status; utility conflicts,
relocations and trenchless-crossing constraints; planning, environmental
and offset obligations; safety-accreditation and worker-competency rules;
and the process, grid-connection, accessibility or marine basis of design,
including commissioning and handover requirements of the receiving
operator.

Record each source's author, revision, date and issue purpose. Treat
operator standards and vendor packages as coordinated-design inputs, not
as proof that the network will accept the works.

Common pitfalls:
- writing the programme before the possession or occupancy calendar exists;
- relying on a walkover in place of tenure, Dial Before You Dig and
  operator interface data.

## Authority interfaces and network operators

Infrastructure programmes turn on interfaces outside the construction
footprint. Identify the consent authorities and the network operators who
must accept design, access, isolation and handover. In NSW, typical
interfaces include TfNSW for State roads and transport; the rail authority
and accredited rail infrastructure manager; the electricity or gas network
operator; the water utility; the port authority; and the aviation authority
and airport operator. Do not state that any of them has capacity or has
accepted a solution unless current written project evidence says so.

Programme applications, design reviews, network-operator approvals,
agreements, augmentation and inspections as explicit dependencies. For
rail stations, DSAPT accessibility is a governing public-transport
obligation, not a generic Premises Standards afterthought. For energy
generation, treat grid connection and generator performance standards as
critical-path network-operator approvals. For treatment plants, identify
the drinking-water quality regulator and the receiving water utility's
asset standards.

Common pitfalls:
- naming only the planning consent and omitting the network operator;
- assuming DSAPT is satisfied because an NCC access report exists;
- placing grid connection after construction award as residual
  administration.

## Possessions, outages and live-network access

Live-network access governs the programme. Track possessions govern rail
programmes: book them, resource them, and design the work to fit the
granted window, including electrical isolation near live overhead. Road
upgrades need staged occupancy; traffic management is a major
cost/programme item, not a site-establishment extra. Night works are
common where daytime network capacity cannot be given up.

Match the access product to the subtype. Marine works need berth and tidal
windows and must live with vessel movements. Airside works often have
night-only windows and escort controls. Water, energy and
telecommunications outages need isolation certificates, customer
notification and rollback. State operational continuity as a testable
requirement: what remains in service, what is isolated, who authorises
energisation, and what happens if the window is lost.

Common pitfalls:
- assuming night works are available without an operator booking;
- planning rail station works as if the line can be taken out of service;
- omitting traffic management, possessions or tidal windows from the
  critical path.

## Safety accreditation and worker competency

Do not send a building-site workforce onto a live network. Rail work
requires rail safety accreditation under the Rail Safety National Law and
demonstrable worker competency for the tasks, locations and electrical
conditions involved. Road work requires traffic-control accreditation.
Airside work requires airside security and access control, including
escort and restricted movement where the operator requires it. Marine
work may require diving competency, vessel-interface rules and port
induction.

Record who holds accreditation, who is a visitor under escort, and which
tasks are prohibited without a named competency. Do not infer that a
White Card and a construction WHS plan satisfy the operator.

Common pitfalls:
- treating rail safety accreditation as a licence that can be obtained
  after award;
- allowing unescorted airside or trackside movement on a visitor
  induction;
- omitting electrical isolation competency for work near live overhead.

## Linear works, land access and utilities

Linear assets are won or lost on land and utilities, not on the typical
section. Resolve land acquisition, easements, native title and landholder
agreements before treating the alignment as constructible.
Telecommunications and regional energy often run as multi-site repeated
delivery: one governance system, many access deals, many power
connections, many environmental approvals. Do not schedule them as a
single site.

Map utility relocations as a package with design status, lead times and
outage windows. Where a creek, rail, road or service cannot be open-cut,
nominate trenchless crossings and the settlement controls they need.
Programme environmental approvals and biodiversity offsets with the same
discipline as the civil works; an offset lag stops the corridor as surely
as a missing easement.

Common pitfalls:
- issuing construction documents on land still subject to acquisition or
  native title;
- assuming utilities will relocate inside the civil programme;
- treating a multi-site tower rollout as one project with one access date.

## Design, packaging and procurement

Select packaging from the actual risk allocation and the operator's
interface rules. Potential packages include land and enabling works,
utility relocations, civil and structures, track or pavement, process or
treatment, grid-connection and substation, marine or airside specialist
works, systems (lifts, baggage, signalling, controls), and integrated
commissioning. For each early package, define design status, scope
boundaries, possession dates, accreditation obligations, test
requirements and who owns end-to-end performance.

Avoid procuring long-lead equipment against an unapproved
network-operator standard. Station, terminal and plant buildings may
still need NCC compliance for those building parts; keep that scope
inside a defined envelope and do not let it rewrite the linear-network
design basis.

Common pitfalls:
- procuring a process, baggage or grid package against a civil drawing
  set;
- leaving operator asset standards out of the tender;
- using a building-works contract form without possession, isolation and
  accreditation clauses.

## Construction in a live network

For work on an operating road, railway, plant, port, airport or utility,
the PMP should control verified isolation points and exclusion zones;
possession areas and public or passenger segregation; shutdown requests,
rehearsals and rollback plans; temporary traffic, pumping and operational
support; work near live overhead, live traffic, live plant, vessels or
aircraft; and daily operational handbacks with evidence of safe
restoration.

The operator should nominate an authorised interface for permits and
restoration acceptance. Never infer that a routine construction
methodology is compatible with a live network. Passenger flow, vessel
berth availability and through-traffic are operational products, not
site inconveniences.

Common pitfalls:
- occupying a platform, lane, berth or airside area beyond the booked
  window;
- restoring a network without a recorded isolation and test status.

## Commissioning into a live network

Plan commissioning from the operator's asset-creation rules, not at the
end of construction. A linear or network asset is accepted when it can
run inside an operating asset, not when a building certificate is issued.
Do not treat NCC DtS as the governing instrument for linear or network
assets. Building-envelope certification, where it applies, is a subset.

Develop inspection and test plans, isolation and energisation procedures,
controls proving and integrated tests under normal, degraded and
emergency modes. Water and process plants need process proving against
treatment performance and the drinking-water quality regulator. Energy
projects need hold points against the connection agreement and generator
performance standards. Rail, road, port and airport systems need
interface tests with the operating network before the next service.
Practical completion should not silently substitute for operational
acceptance by the network operator.

Common pitfalls:
- using occupation-certificate language as the completion test for a
  corridor, plant or grid connection;
- omitting process proving or generator-performance hold points.

## Handover, operability and residual risk

Handover is transfer into an operating network. Deliver asset data,
as-builts, settings, spares, warranties, licences, test records and
operator training in the form the receiving operator specifies. Confirm
maintenance access, isolation points and residual hazards while the
surrounding network stays live. Record defects, deferred tests and any
temporary operational restriction that remains after first service.

Keep an assumptions and residual-risk register for unclosed land,
utility, accreditation, offset and proving items. Flag low confidence
when the subtype, operator, possession regime, accreditation status,
land tenure, connection agreement or acceptance tests are absent. Do
not fill those gaps with a building precedent or general model
knowledge.

Common pitfalls:
- handing over without operator asset-data and isolation records;
- leaving residual possessions or operating restrictions unowned;
- treating first service as proof that proving, DSAPT or generator
  performance obligations are closed.
