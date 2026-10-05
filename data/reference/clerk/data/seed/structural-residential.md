---
tier: topic
seed_type: trade
loaded_by: task-loaded
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
status: active
reconciliation: "Retained because Class 1 footing, framing, wind and BAL guidance is residential-specific rather than duplicated by a cross-class structural guide."
topics: [structural, footings, framing, wind, bal]
summary: "Structural literacy under the residential standards suite: AS 2870 footings and slabs, AS 1684 framing, AS 4055 wind classification, AS 1170 loads and AS 3959 BAL. Supports RFIs, scope descriptions and inspection-gate coordination; never replaces the structural engineer."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Trade seed — Structural (residential)

This seed provides residential-grade structural context for Class 1a and Class 10 buildings. It covers footing and slab design, timber framing, wind classification, and load principles under the Australian residential standards suite. It does not replace the structural engineer; it gives the agent and project lead enough literacy to ask the right questions, identify inspection gates, and flag when specialist input is required.

Load this seed task-loaded when a role × archetype task involves structural RFIs, scope descriptions, specification review, or inspection stage coordination.

---

## AS 2870 — Residential slabs and footings

### Site classification

Soil reactivity governs footing design. Classification is assigned by a geotechnical engineer after a site investigation (test pit, core, or borehole). The classes in ascending reactivity order:

| Class | Description | Characteristic surface movement (ys) |
|---|---|---|
| A | Stable, non-reactive — sand or rock | < 10 mm |
| S | Slightly reactive clay | 10–20 mm |
| M | Moderately reactive clay or silt | 20–40 mm |
| H1 | Highly reactive — moderate movement | 40–60 mm |
| H2 | Highly reactive — high movement | 60–75 mm |
| E | Extremely reactive | > 75 mm |
| P | Problem site | Fill, soft soil, collapsing or soluble material, unstable slope, swampy ground |

**P-class sites require a site-specific design by a structural engineer.** AS 2870 prescriptive solutions do not apply. The agent must not prescribe footing solutions for P-class sites; surface the risk and route to the structural engineer.

### Slab and footing types

AS 2870 prescribes stiffened raft slabs, waffle pods, and strip footings. The dominant residential form is the **stiffened raft slab** — a concrete slab with deepened edge and internal beams, reinforced per the classification table.

Key design parameters that vary by class:
- Beam depth and width
- Slab thickness
- Reinforcement bar size and spacing
- Edge beam projection below natural surface level

### Inspection and evidence gates

- **Soil report** before footing design commences (owner/builder obligation to provide to engineer and certifier)
- **Reinforcement inspection** before concrete pour — certifier or accredited inspector sign-off
- **Concrete conformance test** (CFT / slump test) at pour — documented by contractor
- **Slab inspection** by certifier before frame commences

The agent must not mark the slab stage complete without evidence of reinforcement inspection and certifier sign-off. Reference §evidence-discipline.

---

## AS 1684 — Residential timber framing

### Series structure

| Standard | Scope |
|---|---|
| AS 1684.1 | Design procedures — principles and methodology |
| AS 1684.2 | Non-cyclonic areas — prescriptive span tables |
| AS 1684.3 | Cyclonic areas (wind regions C and D) — modified span tables and connections |
| AS 1684.4 | Simplified — very small buildings and outbuildings |

The project's wind classification (from AS 4055) determines whether .2 or .3 applies.

### Key elements and span tables

AS 1684.2 / .3 contain span tables for:
- Floor joists and bearers
- Ceiling joists
- Rafters and roof battens
- Hanging and strutting beams
- Lintels and trimmers over openings
- Verandah beams and posts

Span tables are governed by: species group, member size (depth × breadth), spacing, and span. The agent should not select member sizes from general knowledge — span tables require knowledge of the actual species group and load case. Flag any member sizing query to the structural engineer or refer to the published tables.

### Notching, drilling, and modification rules

AS 1684 contains strict rules for notching floor joists and bearers and drilling through members. Common violation: plumbers and electricians notch or drill outside the permitted zones, weakening members.

- Notches must be in the permitted zone (typically the outer third of the span)
- Notch depth limits apply per member depth
- Drilling must not reduce the residual member depth below the minimum

Inspection point: check for unauthorised notching and drilling before lining. For renovations, concealed modifications may have occurred during prior works — flag as latent condition risk.

### Connection and tie-down requirements

Cyclonic and high-wind areas require additional metal strap connections at:
- Roof truss to top plate
- Top plate to stud
- Stud to bottom plate
- Frame to slab / subfloor

Non-cyclonic areas still require tie-down at exposed or high-wind-speed locations (N3 and above in AS 4055 terminology — see below). Connection schedules are part of the engineer's specification and must be documented in the frame inspection sign-off.

### Frame inspection gate

Frame inspection is a mandatory hold point before any wall lining, ceiling lining, or external cladding proceeds. Evidence required:
- Certifier or accredited inspector frame inspection certificate
- Engineer's sign-off if non-standard elements are present (steel beams, LVL, portal frames)
- Confirmation of tie-down and connection installation

The agent must not advance the programme past lockup without confirmed frame sign-off. Reference §evidence-discipline.

---

## AS 4055 — Wind loads for housing

### Wind speed regions

Australia is divided into wind regions. NSW is predominantly Region A or B; cyclonic regions C and D apply to parts of QLD, WA, and NT.

| Region | Description | Design wind speed (kPa) |
|---|---|---|
| A (N1–N3) | Low to moderate — most of NSW, VIC, SA, TAS interior | N1: low; N3: moderate-high |
| B (N4–N6) | Medium — coastal NSW, south-east QLD | Higher than A |
| C (C1–C4) | Cyclonic — tropical QLD, WA, NT coast | Cyclonic loading |
| D (C4) | Severe cyclonic — Pilbara WA | Highest |

AS 4055 classifies structures using the nomenclature **N1–N6** (non-cyclonic) and **C1–C4** (cyclonic), derived from region, terrain, and shielding.

### Classification inputs

1. **Wind region** — from BOM wind region map, reproduced in AS 4055 Appendix A
2. **Terrain category (TC)** — TC1 (exposed, open), TC2 (suburban), TC3 (heavily shielded)
3. **Shielding** — Full Shielding (FS), Partial Shielding (PS), No Shielding (NS) from surrounding buildings

These three inputs determine the final wind classification (e.g. N2, N3, C2).

### Design wind class implications

- **N1–N2:** Standard framing from AS 1684.2 — most suburban NSW sites
- **N3–N6:** Increasing tie-down and connection requirements; may require upgraded glazing, roof cladding fixing schedules, and fascia/gutter attachment
- **C1–C4:** AS 1684.3 applies; cyclonic fixings, cyclone shutters, and specialist engineering review typically required

The wind classification must be established by the engineer or certifier at the design stage. It flows through to:
- Framing member selection
- Connection and tie-down schedules
- Window and glazing specification
- Roofing fixing schedules (number and type of screws/clips per sheet or tile)

---

## AS 1170 — Structural actions (loads)

### Relevant parts for residential

| Part | Load type | Residential application |
|---|---|---|
| AS 1170.1 | Permanent (dead) and imposed (live) | Floor live loads (1.5 kPa residential), roof live loads, balcony loads |
| AS 1170.2 | Wind | For Class 1a, AS 4055 is the simplified pathway — AS 1170.2 applies to Class 2+ or engineer-designed elements |
| AS 1170.3 | Snow and ice | Relevant for alpine areas (ACT, alpine VIC/NSW) |
| AS 1170.4 | Earthquake | Low-moderate seismic risk across most of Australia; Class 1a typically addressed through AS 3000 and framing standards, but SA1 and SA2 sites (higher seismicity) warrant engineer attention |

### Key residential load cases

- **Floor live load:** 1.5 kPa for domestic areas (AS 1170.1 Table 3.1)
- **Balcony / deck live load:** 3.0 kPa if accessible from the dwelling and used for gatherings
- **Roof live load:** Maintenance access only — 0.25 kPa non-trafficable; 1.5 kPa trafficable
- **Permanent (dead) loads:** Governed by material densities — tile roofs significantly heavier than metal; affects beam and footing design

The agent must not perform structural calculations. These load cases are provided so the agent can correctly describe scope, identify load-bearing elements, and flag when proposed changes (e.g. adding a tile roof over a metal roof framed structure) need engineer review.

---

## AS 3959 — Construction of buildings in bushfire-prone areas (BAL)

### BAL ratings and implications

A Bushfire Attack Level (BAL) is assigned as part of a bushfire assessment. Higher BAL = higher ember / flame exposure = more stringent construction.

| BAL | Exposure level | Key construction implications |
|---|---|---|
| BAL-LOW | Very low | Standard construction; no AS 3959 requirements |
| BAL-12.5 | Low | Subfloor and roof spaces enclosed; roof sarking; ember guards on vents |
| BAL-19 | Medium | Hardwood or non-combustible decking; wire mesh in gutters; windows may need protection |
| BAL-29 | High | Non-combustible or hardwood external cladding; upgraded window and door systems |
| BAL-40 | Very high | Metal framed windows/doors; non-combustible cladding; concrete slab preferred |
| BAL-FZ | Flame zone | Highest specification; often requires engineer certification of entire envelope |

BAL assessment is a specialist input (accredited BAL assessor or bushfire consultant). The agent must not determine BAL from description — surface the need for an assessment and document the result as project evidence.

For full BAL coverage in the new-dwelling context, see `new-dwelling-guide.md`. This section provides framing-level context for structural and specification tasks.

---

## Services by stage

> **AGENT DRAFT (S12a, 26 September 2026): awaiting owner review.** Stage ids
> match `architectural-services-guide.md` (SEED_MAP owner decision Q3).

Each stage opens with an **RFP line** that code can render as the lead of a
grouped RFP item. Use only the stages inside the appointment boundary; price
the rest separately or leave them out.

### Inception, site and existing information [inception]

**RFP line:** Review the geotechnical report, the site and any existing
structure, and confirm the structural design basis.

- Review the geotechnical report and site classification (AS 2870), wind
  classification (AS 4055) and bushfire attack level.
- For alterations, inspect the existing structure and record its condition.

### Concept and schematic design [concept]

**RFP line:** Advise on the structural system and spans for the schematic
design.

- Footing, slab and framing system options, transfer structures and
  retaining walls, with cost implications.

### Pre-DA consultation and development application [development_application]

**RFP line:** Provide any structural statement the development application
needs.

- For example structural adequacy of existing walls or retaining near
  boundaries.

### Conditions of consent and design development [design_development]

**RFP line:** Develop the structural design for the construction certificate.

- Footing and slab design (AS 2870), framing (AS 1684 or engineered),
  retaining walls and bracing.
- Coordinate with civil levels, drainage and the architect.

### Documentation and tender [documentation_tender]

**RFP line:** Prepare structural drawings and certification for the
construction certificate and tender, and answer tender queries.

- Structural drawings, the design certificate for the construction
  certificate, and responses to tenderer RFIs.

### Novation and post-contract design [post_contract_design]

**RFP line:** Usually excluded on residential work; price shop drawing and
truss or frame review at hourly rates.

- Review truss, frame and steel shop drawings when the builder submits them.

### Construction and completion [construction_completion]

**RFP line:** Inspect the structure at hold points and provide the inspection
certificates the certifier needs.

- Footing, slab reinforcement and framing inspections with certificates.
- Instructions where site conditions differ from the design basis.

### Throughout the appointment [coordination]

**RFP line:** Respond to design team and builder structural queries throughout
the appointment.

- Keep a register of inspections and certificates issued.

## Coverage depth

| Topic | Depth | Notes |
|---|---|---|
| AS 2870 site classification and slab types | Moderate | Prescriptive design requires engineer input; P-class always engineer |
| AS 1684 framing principles and inspection gates | Moderate | Span table selection requires species/load data — refer to engineer or tables |
| AS 4055 wind classification | Moderate | Region lookup and terrain category classification provided; final classification is engineer/certifier role |
| AS 1170 residential load cases | Shallow | Indicative values only; structural calculations are out of scope |
| AS 3959 BAL construction requirements | Shallow | Overview only; BAL assessment and specification is specialist input |

---

## Low-confidence flags

- **Non-standard structural systems** (CLT, SIP panels, light-gauge steel framing, post-and-beam): AS 1684 prescriptive solutions do not cover these. Engineer design and certification required. Flag immediately rather than extending this seed's guidance.
- **P-class sites and variable fill**: AS 2870 prescriptive solutions explicitly exclude P-class. The agent must escalate to the structural engineer without exception.
- **Earthquake-sensitive sites (SA1/SA2 seismic zones)**: AS 1170.4 may impose additional requirements beyond standard residential framing. Check with engineer for alpine NSW, ACT, and areas near known fault lines.
- **Cyclonic regions (C/D wind regions)**: AS 1684.3 and AS 4055 cyclonic requirements are noted but not detailed in this seed. Engage a structural engineer familiar with cyclonic construction before proceeding.
- **Existing buildings (renovations)**: Assumed soil classification, footing type, and framing species may not match reality. Treat all existing structural assumptions as unconfirmed until investigated.
