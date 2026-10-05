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
reconciliation: "Retained for residential OSD, sewer, crossover and subdivision interfaces; a future cross-class civil guide should complement rather than replace it."
topics: [civil, stormwater, earthworks, drainage]
summary: "Residential civil and site-works literacy: earthworks cut and fill, stormwater and OSD, sewer connection, driveway crossovers, and subdivision civil interfaces. Equips the project lead to scope work, raise the right RFIs, and flag when the civil or hydraulic engineer must be engaged."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Trade seed — Civil (residential)

This seed provides residential-grade civil and site works context for Class 1a and Class 10 buildings. It covers earthworks, stormwater drainage, OSD, sewer connection, and driveway crossover. It does not replace the civil or hydraulic engineer; it gives the agent and project lead sufficient literacy to identify scope, ask the right questions, and flag when specialist input is required.

Load this seed task-loaded when a role × archetype task involves earthworks scope, drainage RFIs, stormwater design, infrastructure connections, or civil inspection coordination.

---

## Site earthworks — cut and fill

### Basic principles

**Cut:** soil is excavated below existing ground level. The excavated material is either removed from site (spoil) or reused as fill elsewhere on site.

**Fill:** imported or reused material is placed above existing ground level to build up a platform for the slab, driveway, or landscaping.

Most residential sites involve a combination of cut and fill to achieve a level building platform or subfloor.

### Batter slopes and retention

When cut or fill exceeds certain heights, the soil face (batter) must either be graded to a stable angle or retained by a wall:

| Soil type | Maximum unreinforced batter (approx.) |
|---|---|
| Dense clay, gravel | 1:1 (45°) |
| Sandy clay | 1:1.5 |
| Loose sand | 1:2 |

Where site geometry prevents battered slopes (boundary constraints, neighbouring buildings), retaining walls are required. **Retaining walls > 600 mm high** in NSW require a building approval (CDC or DA) and, where > 1.0 m high, structural engineer design. The agent must flag proposed cut or fill > 1.0 m and any retaining wall > 600 mm as requiring approval and engineer input.

### Geotechnical sign-off trigger

AS 2870 relies on a geotechnical report for site classification. Beyond classification, geotechnical advice may be required for:
- Cut > 1.5 m into unknown material
- Fill platforms > 500 mm deep (fill quality and compaction specification)
- Sites adjacent to drainage lines, watercourses, or unstable slopes
- Sites with evidence of prior fill or contamination

The agent must not proceed with earthworks scope without confirming a geotechnical report is in place. Reference §evidence-discipline.

### Compaction and fill quality

Imported fill and on-site reused fill must be compacted to a specified density ratio (typically 95% standard Proctor compaction for structural fill under slabs). Uncontrolled fill is a P-class site trigger under AS 2870 — see `structural-residential.md`. Evidence of fill compaction testing (nuclear density testing or sand replacement test) must be kept as project documentation before the slab is poured.

---

## Stormwater drainage

### Gravity drainage principles

Residential stormwater systems collect rainwater from:
- Roof (gutters, downpipes)
- Paved areas (driveways, patios, paths)
- Pool overflow and backwash

Stormwater is conveyed by gravity (preferred) or pump to the point of discharge. Minimum grades for stormwater pipes: 1:100 for 100 mm diameter, 1:200 for 150 mm diameter (AS/NZS 3500.3 indicative).

### Connection to council drainage

NSW: Most urban council areas have a kerb-side stormwater pit network. New dwellings must connect to council's system unless the site is in an area without reticulated stormwater. Connection requires:
- Application to council (or relevant authority) for a connection approval
- Inspection by council's drainage inspector before the connection is buried

In rural or fringe areas without reticulated stormwater, overflow to a legal point of discharge (e.g. natural drainage line, absorption trench if permitted) requires council/certifier approval.

### On-site detention (OSD)

Many NSW councils require OSD for new developments and significant alterations. OSD temporarily holds a portion of stormwater runoff on site (in a tank or basin) and releases it at a controlled rate to the council system, preventing downstream flooding.

**OSD trigger:** Typically applies when impervious area increases above a threshold (site-specific, governed by council's DCP or drainage policy). The certifier and hydraulic engineer will advise.

**OSD components:**
- Detention tank (typically HDPE or concrete, buried)
- Orifice plate or pit fitting to restrict outflow rate
- Overflow provision at the roof level to prevent surcharge flooding the building
- Access for maintenance

OSD sizing is a hydraulic engineer scope item. The tank must be installed before occupation certificate (OC) is issued — it is a critical sign-off item. The agent must confirm OSD status early in the programme.

### Rainwater tank

NSW BASIX may require rainwater tank provision for garden use, toilet flushing, or cold water supply. Key considerations:
- Tank size (kL) specified by BASIX certificate
- First flush diverter and mosquito screening required
- Pump and internal plumbing if tank is used for toilet/laundry supply (requires licensed plumber)
- Tank location: setback from boundaries, from septic or sewer infrastructure

---

## Sewer connection

### Gravity sewer

Most urban NSW sites connect to Sydney Water's (or relevant water utility's) reticulated sewer by a gravity connection from the building to the street main. The connection requires:
- Application to Sydney Water (or relevant authority) for a Section 73 Certificate (certificates of compliance)
- Hydraulic design showing the sewer system layout
- Inspection at key stages (trenches open before backfill)
- Final plumbing compliance certificate

The **Section 73 Certificate** (NSW) confirms that water and sewer infrastructure is available and that the development complies with the network requirements. It is required before an OC can be issued for most new developments. Flag absence of a Section 73 early — long lead times from the water utility can delay OC.

**State callout — VIC:** In Victoria there is no Section 73 Certificate. The equivalent instrument is a **Certificate of Compliance — Water / Sewer** issued by the applicable retail water corporation. The relevant authority varies by location (e.g. Melbourne Water, Yarra Valley Water, South East Water, Western Water, Greater Western Water — confirm with the project's hydraulic engineer or the council). Connection application process, fees, and lead times differ materially from Sydney Water — do not extend Section 73 guidance to a VIC project. If `state: VIC` and the task references a Section 73 or Sydney Water, **flag the gap** and confirm the applicable VIC water corporation and their requirements before proceeding.

### Inspection openings (IOs)

Sewer drains must have inspection openings at specified intervals and at changes of direction. IOs allow inspection and clearance of blockages. Access must be maintained permanently (not buried or paved over without a flush IO). The agent must confirm IOs are shown on the hydraulic drawings.

### Pump-out systems

Sites that cannot drain to the street sewer by gravity (low-lying sites, rear lots) require a pump-out system:
- Holding tank (typically PE, below finished floor level)
- Submersible grinder pump
- Rising main to the street or gravity sewer
- Alarm for pump failure

Pump-out systems require maintenance and have ongoing operating costs (electricity, pump replacement). The agent must flag pump-out requirement as a cost and maintenance risk.

### Septic and on-site wastewater

Sites without reticulated sewer require on-site wastewater treatment and disposal:
- **Septic tank + absorption trench:** Older technology; relies on soil permeability. Requires adequate land area and suitable soils.
- **Aerated wastewater treatment system (AWTS):** Treats wastewater to a higher standard; enables disposal to surface irrigation or smaller absorption area.
- **Composting / dry toilet systems:** Niche; may be applicable in very remote areas.

On-site wastewater systems require a separate approval from council (Environment Protection Licence or equivalent). Site suitability assessment (percolation test) required. The agent must flag on-site wastewater as a specialist scope item requiring early resolution — it affects site layout, setbacks, and building position.

---

## Driveway crossover

### Definition

The driveway crossover is the portion of the driveway between the property boundary and the road carriageway — typically crossing the footpath, kerb, and nature strip. It is the physical and jurisdictional interface between the private property and the public road.

### Approvals

In NSW, driveway crossover construction or modification requires a **road opening permit** or driveway approval from council. Requirements vary by council but typically include:
- Application and fee
- Minimum and maximum driveway widths
- Minimum sight-line setbacks
- Reinstatement of kerb and gutter if existing crossover is altered
- Inspection by council's roads inspector before backfill

### Construction requirements

Typical residential driveway crossover requirements:
- **Level at property boundary:** Must match the kerb and gutter design
- **Grade:** Driveway must transition from kerb level to slab or road level without creating a step or catch point. Maximum driveway grade typically 1:4 (25%) for the first 5 m from the road — council-specific
- **Width:** Minimum 3.0 m for single garage, 5.0 m for double garage — council-specific
- **Materials:** Concrete is standard for crossover; council may require matching the existing footpath material

### Kerb and gutter reinstatement

If an existing concrete kerb and gutter is removed or modified to form the crossover, council requires reinstatement of the kerb and gutter to either side of the crossover. This is typically included in the builder's civil scope. Costs can be significant on kerb returns.

---

## Subdivision and multi-dwelling civil interfaces

For multi-dwelling developments (strata or Torrens title subdivision), civil scope expands to include:
- **Internal roads and access driveways** to the satisfaction of council's traffic engineer
- **Shared stormwater drainage** infrastructure — charged pits, main drain to council system, OSD sized for the whole development
- **Sewer lead-in** sized for multiple units
- **Water main** connection and individual metering (Sydney Water requirement for strata)
- **Boundary fencing** as a civil item where it is a development condition

These items are out of scope for single-dwelling new-dwelling and renovation archetypes but are relevant for the multi-dwelling archetype. Reference `multi-dwelling-guide.md` for the full multi-dwelling civil interface picture.

---

## Services by stage

> **AGENT DRAFT (S12a, 26 September 2026): awaiting owner review.** Stage ids
> match `architectural-services-guide.md` (SEED_MAP owner decision Q3).

Each stage opens with an **RFP line** that code can render as the lead of a
grouped RFP item. Use only the stages inside the appointment boundary; price
the rest separately or leave them out.

### Inception, site and existing information [inception]

**RFP line:** Review the survey, the site and existing stormwater and sewer
information, and identify the civil constraints.

- Review the detail survey, BYDA plans, council stormwater information and
  the water authority's sewer records.
- Identify fall to the street, the need for an easement or pump-out, and
  sewer mains within the lot.

### Concept and schematic design [concept]

**RFP line:** Prepare the civil and stormwater concept for the schematic
design.

- Stormwater discharge strategy and on-site detention need under the council
  DCP.
- Cut and fill and driveway grade advice.

### Pre-DA consultation and development application [development_application]

**RFP line:** Prepare the concept stormwater plan and civil inputs for the
development application, and respond to council requests on them.

- Concept stormwater management plan and on-site detention calculations.
- Driveway long sections where council requires them.

### Conditions of consent and design development [design_development]

**RFP line:** Review the civil conditions of consent and develop the civil
design for the construction certificate.

- Detailed stormwater and on-site detention design, erosion and sediment
  control plan, and driveway design.
- Roads Act 1993 section 138 application for crossover and road-reserve
  work; sewer building plan approval or Section 73 as needed.

### Documentation and tender [documentation_tender]

**RFP line:** Prepare civil documentation for the construction certificate and
tender, and answer tender queries.

- Civil drawings and specification notes; respond to tenderer RFIs.

### Novation and post-contract design [post_contract_design]

**RFP line:** Usually excluded on residential work; price any contractor design
support at hourly rates.

- Review contractor-proposed changes to levels or drainage.

### Construction and completion [construction_completion]

**RFP line:** Inspect the civil works at hold points and provide the
certification and works-as-executed information the certifier needs.

- Inspect drainage and on-site detention before backfill.
- Works-as-executed survey review and on-site detention certification;
  support the positive covenant and restriction on title where required.

### Throughout the appointment [coordination]

**RFP line:** Respond to design team queries and coordinate civil levels with
the architect and structural engineer throughout the appointment.

- Keep council and water authority approvals in the approvals tracker.

## Coverage depth

| Topic | Depth | Notes |
|---|---|---|
| Cut and fill, batter, and retention | Moderate | Engineering triggers and approval thresholds covered; retaining wall and geotechnical design is specialist scope |
| Stormwater drainage and OSD | Moderate | Principles and OSD framework covered; hydraulic design is engineer scope |
| Rainwater tank (BASIX) | Shallow | BASIX requirement noted; sizing and plumbing is engineer/plumber scope |
| Sewer connection and Section 73 | Moderate | Process and certificate covered; hydraulic design is licensed professional scope |
| Pump-out and on-site wastewater | Shallow | Overview only; site suitability and system design is specialist scope |
| Driveway crossover | Moderate | Approval process and key requirements covered; detailed levels design is civil engineer scope |
| Multi-dwelling civil infrastructure | Shallow | Flagged as a scope boundary; detail is in multi-dwelling archetype seed |

---

## Low-confidence flags

- **Flood-affected sites:** Flood constraints (overland flow, riverine flooding) can govern fill heights, finished floor levels, and stormwater systems. The agent must not advise on flood-constrained sites without a flood study or council's flood planning information — surface the constraint and route to the hydraulic engineer.
- **Acid sulfate soils (ASS):** Present in low-lying coastal NSW (typically areas below RL 5 m AHD near estuaries). Disturbance of ASS can release sulfuric acid, damaging infrastructure and harming waterways. ASS assessment required before earthworks on potentially affected sites. Flag for any coastal or low-lying site.
- **Contaminated land:** Prior industrial, agricultural, or fill sites may have contamination. A Phase 1 Environmental Site Assessment (ESA) should precede earthworks. The agent must flag known or suspected contamination as a specialist scope item rather than extending this seed's guidance.
- **Sydney Water / water utility timelines:** Section 73 Certificate and infrastructure availability assessment can take 6–12 weeks from application. Early lodgement is essential. Do not assume availability — always confirm.
- **Stormwater absorption / dispersion (rural sites):** Percolation rates and soil type determine whether absorption trenches are viable. Site-specific testing required; general guidance should not be used to confirm suitability.
