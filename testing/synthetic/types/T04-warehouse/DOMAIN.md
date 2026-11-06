# T04 Industrial warehouse: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (7b59d9b plus the working tree) and professional practice. Nothing in the
application or the knowledge was changed.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. A clause or section number I could not
check against the instrument is written `verified: false`. No prices, rates or week durations appear.
Owner rulings (`testing/synthetic/reference/OWNER_RULINGS.md`) apply. R3 matters here in the negative: the
Design and Building Practitioners Act obligations apply to Class 2 and Class 9c, not to a Class 7b
warehouse, so `proj.regulated_design` is `no`. Conventions C1 (novation of services engineers is
`option`) and C2 (`consultant.esd` for Section J) are followed.

---

## 1. The job

**What it is.** A new single-storey, single-tenant warehouse with an ancillary office, on a vacant lot in
an established or newly subdivided industrial estate in Western or South-Western Sydney. The app profile
is `industrial / warehouse / new` with `det.ncc_class: [7b, 5]`
[K: knowledge/profile/taxonomy.yaml building_classes industrial warehouse, ncc_class 7b]. Base scenario:

| Item | Base value |
|---|---|
| Site | One lot of about 1.6 ha in an IN1 General Industrial (or E4 General Industrial after the 2023 employment-zones reform; verified: false) zone, fronting a local industrial road. Vacant and benched by the estate developer, with a validation report for industrial use. Gentle fall; no trees, flood, bushfire, heritage or contamination constraints. |
| Building | About 7,500 m² of warehouse and 500 m² of single-storey office and amenities (office about 6% of GFA). Internal clear height about 12 m at the haunch. Single storey, so rise in storeys 1 and Type C construction [K: knowledge/tables/type_of_construction.yaml rows classes 5,6,7a,7b,8 rise 1 → C]. One fire compartment. |
| Construction | Pad footings, a ground-bearing industrial slab (sawn or jointless, designed for racking leg and forklift loads, with a floor flatness specification), structural steel portal frame with purlins and girts, **off-site precast concrete panels** to the dock face, the front and the boundary walls (dado or full height) with metal cladding above, metal deck roof with insulation blanket and skylights, recessed loading docks with dock levellers and shelters, plus on-grade roller doors, awnings over the docks. |
| External works | Bulk earthworks to the pad, heavy-duty concrete hardstand for 26 m B-doubles, a separate car park for about 40 cars, two heavy-duty vehicle crossings, on-site detention (OSD) with water-quality treatment (gross pollutant trap, bioretention or proprietary filters), landscape setback to the street, security fencing and gates. |
| Fire | Sprinklered throughout with ceiling-level ESFR sprinklers (storage to about 11 m in selective racking), sprinkler and hydrant tanks with a pump room, hydrant booster at the street, hose reels, extinguishers, emergency lighting, building occupant warning. Sprinklers are required by C3D4(1)(b) and by E1D13 (storage over 4 m). A large isolated building under NCC 2022 C3D4, because the compartment exceeds Table C3D3 [K: knowledge/tables/compartment_limits.yaml exclusions "C3D4 large isolated buildings"]: C3D4(1)(b) lets a Class 5–9 building of at most 18,000 m² and 108,000 m³ exceed Table C3D3 if it is sprinklered (Spec 17) and has perimeter vehicular access to C3D5(2) (at least 6 m wide, unobstructed, furthest boundary no more than 18 m from the building). The base, about 8,000 m² and about 100,000 m³, is inside both limits. NCC 2022 Vol One Parts C3 and E1 (ABCB, https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c3-compartmentation-and-separation; verified: true, T04 review). Performance solutions are usual (perimeter vehicular access, travel distances, smoke hazard management). |
| Services | Kiosk substation and consumer mains from the DNSP (Endeavour Energy in most of Western Sydney), main switchboard, high-bay LED lighting with controls, power to docks, doors and chargers, office split or VRF air-conditioning, amenities exhaust, water, sewer, rainwater reuse for toilets and irrigation, NBN lead-in, CCTV and access control. |
| Pathway | Local DA to council (Class 7b warehouse or distribution centre is permitted with consent in the industrial zone; verified: false), then construction certificates, usually staged (early works for bulk earthworks, then the building). On D&C both CCs follow award and the contractor's design. Where the developer lets bulk earthworks under a separate early-works contract before the D&C award, the early-works CC moves ahead of the award (a variant, not the base). |
| Procurement | Owner or developer client. DA by the owner's design team, then a design and construct lump sum under AS 4902 [K: knowledge/profile/project_facts.yaml contract_form as4902]. Architect, structural and civil engineers and the landscape architect are novated; the fire, electrical and hydraulic consultants are novated or replaced by the builder's D&C subcontractors' designers (C1). |
| Value | Inside the PROJECT_TYPES range of $5–30M; the shape names no amount. |

Classification: the office is Class 5 and the warehouse Class 7b. A BCA consultant may treat a small
office as part of the 7b major use under the NCC's 10 per cent concession (verified: false), but most
BCA reports for a warehouse of this kind name both classes, and PROJECT_TYPES does. [D]

**Who procures it and how.** A developer (a listed industrial REIT, a fund or a private developer) or an
owner-occupier logistics company. The developer appoints a development manager or PM, then an architect,
planner and the DA consultants (about 15 firms). The DA design is taken to D&C tender with a performance
brief and a design responsibility matrix; three or four industrial builders price it. The builder
completes the design with the novated team and its D&C services subcontractors. A tenant pre-lease often
sets the brief (clear height, dock count, floor loads, sprinkler type, office spec), and the tenant's own
fit-out (racking, in-rack sprinklers, forklift charging, IT) is excluded. A traditional fully documented
lump-sum tender is the main alternative (trigger `traditional-lump-sum`); builder-led D&C from concept is
common for owner-occupiers. [D]

**What makes it go wrong.** [D]

1. **Fire strategy settled late.** The large isolated building conditions (sprinklers, perimeter vehicular
   access width and continuity, hydrant coverage, smoke hazard management) and the performance solutions
   that relax them must be agreed before the DA site plan is fixed. FRNSW's comments on the fire
   engineering brief or report can move the perimeter road, the booster, the tank and the pump room. The
   storage height and commodity the tenant actually brings can make the ESFR design wrong.
2. **Ground and earthworks.** Uncontrolled fill, reactive clay, rock in cuts, saline soils and wet-weather
   earthworks drive the slab and pavement design. The cut-fill balance and the retaining walls at the
   boundaries are often the largest unknown in the site works.
3. **Authority long-lead items.** The kiosk substation (DNSP design, contestable works by an accredited
   service provider, easement, energisation), Sydney Water Section 73 through a Water Servicing
   Coordinator, the fire-water pressure and flow from the main, and council's OSD and road-reserve
   approvals sit on the critical path to occupation.
4. **Heavy-vehicle access.** Swept paths for 26 m B-doubles, sight distance at the crossings, the queuing
   area at the gate and the separation of cars from trucks. A late change to the design vehicle moves the
   hardstand and the crossings.
5. **The slab.** Flatness and joint layout must suit the racking and forklift type; a VNA or high-bay
   racking brief changes the slab specification and the program.
6. **D&C scope gaps.** A thin performance brief leaves the builder free to choose the cheapest wall,
   roof, sprinkler and pavement solutions; the tenant's requirements arrive as variations.
7. **Wet weather and steel and panel lead times** during bulk earthworks and erection.

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. Actions from `knowledge/works/actions.yaml`. The app
gives this profile the 46-system warehouse list, all with action `new`
[K: knowledge/profile/scope_defaults.yaml classes[3] (warehouse/new)]. Items marked **+** are ones a
competent PM expects that the default list lacks; items marked **−** are in the default list but carry no
work in the base.

### `site`

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | new | Set-out, level control, as-built survey [D] |
| `site.ground-conditions` **+** | investigate | Boreholes, test pits, CBR and reactivity tests, salinity [K: knowledge/clusters/structure/rules.yaml rule.as1726.ground-investigation] |
| `site.earthworks` | new | Bulk earthworks to the pad, Level 1 compaction testing [K: knowledge/clusters/structure/rules.yaml rule.as3798.controlled-fill] |
| `site.erosion-sediment-control` | new | Sediment basin and controls for a large disturbed area [D] |
| `site.stormwater-detention` | new | OSD and water-quality treatment to council's policy [D] |
| `site.services-connections` | new | Water, sewer, fire-water main, electricity, NBN [D] |
| `site.vehicle-access` | new | Heavy-duty hardstand, car park, internal roads, line marking [K: knowledge/works/system_terms.yaml Hardstand → site.vehicle-access] |
| `site.loading-docks` **+** | new | Recessed docks, levellers, shelters, dock bumpers [K: knowledge/works/system_terms.yaml Dock leveller → site.loading-docks] |
| `site.impact-protection` | new | Bollards, column and door-jamb guards, dock edge protection [D] |
| `site.public-domain` **+** | new | Two heavy-duty crossings, kerb and footpath reinstatement, street trees, under a Roads Act s138 consent (verified: false). No other road works in the base [D] |
| `site.waste-storage` **+** | new | Bin enclosure and collection point [D] |
| `site.signage` | new | Building identification and traffic signage [D] |
| `site.landscape` | new | Street setback landscaping, car park trees, fencing and gates [D] |
| `site.trees`, `site.flooding`, `site.contamination`, `site.acid-sulfate-soils`, `site.retaining-walls`, `site.demolition` | — | Triggers only (`trees`: none on a benched estate lot) |
| `site.adjoining-structures`, `site.pools` | — | Not applicable: setbacks to boundaries, no pool |

### `substructure`

| System | Action | Why |
|---|---|---|
| `substructure.footings` | new | Pad footings to the portal columns, strip footings to the precast panels [D] |
| `substructure.slab-on-ground` | new | Industrial slab: racking and forklift loads, joints, flatness, vapour barrier [K: knowledge/works/system_terms.yaml Industrial slab → substructure.slab-on-ground] |
| `substructure.piles` | — | Trigger (deep fill or soft ground) |
| `substructure.basement-retention`, `.damp-waterproofing`, `.retaining-walls`, `.underpinning` | — | No basement; site retaining walls are a trigger |
| `substructure.termite-management` | — | Steel and concrete structure; not required (verified: false) |

### `structure`

| System | Action | Why |
|---|---|---|
| `structure.steel` | new | Portal frame, purlins, girts, dock awnings, office roof framing [K: knowledge/clusters/structure/rules.yaml rule.as4100.steel-design] |
| `structure.precast` | new | Precast wall panels to the dock face, front and boundaries [K: knowledge/works/system_terms.yaml Tilt-up panels → structure.precast] |
| `structure.temporary-works` **+** | new | Panel bracing until the frame and roof are tied in; crane pads [D] |
| `structure.design-basis` | — | A design matter (wind region, importance level 2) inside the structural engineer's scope [K: knowledge/clusters/structure/rules.yaml rule.as1170.wind-actions] |
| `structure.concrete`, `.masonry`, `.timber`, `.alterations` | — | No suspended concrete, masonry or timber structure; new build |

### `envelope`

| System | Action | Why |
|---|---|---|
| `envelope.external-walls` | new | Metal cladding above the panels; office facade [D] |
| `envelope.roof-coverings` | new | Low-pitch metal deck roof, skylights, roof safety mesh [K: knowledge/clusters/envelope/rules.yaml rule.ncc.metal-roof-installation] |
| `envelope.windows-doors` | new | Office glazing, personnel doors [D] |
| `envelope.vehicle-doors` | new | Panel-lift dock doors and roller shutters [D] |
| `envelope.thermal-insulation` | new | Roof insulation blanket, office walls [D] |
| `envelope.flashings-seals` **+** | new | Roof and wall flashings, precast panel joint sealants [D] |
| `envelope.thermal-performance` **+** | new | NCC Section J for the office and the warehouse lighting (clauses verified: false) [D] |
| `envelope.curtain-wall`, `.external-waterproofing`, `.facade-acoustics`, `.pliable-membranes`, `.building-sealing`, `.external-shading`, `.bushfire-construction` | — | Not in the base (`bushfire-construction` is a trigger) |

### `interiors`

| System | Action | Why |
|---|---|---|
| `interiors.walls-linings` | new | Office partitions and linings |
| `interiors.ceilings` | new | Office suspended ceilings |
| `interiors.wet-area-waterproofing` | new | Amenities and showers |
| `interiors.tiling` | new | Amenities |
| `interiors.floor-finishes` | new | Office carpet and vinyl |
| `interiors.internal-doors` | new | Office doors |
| `interiors.joinery` **+** | new | Office kitchen and amenities vanities (the implied_by `kitchen` pattern adds it when a read document says "kitchen") [K: knowledge/profile/scope_defaults.yaml implied_by kitchen] |
| `interiors.painting` | new | Office painting; warehouse floor line marking sits with painting in the app's fragment [K: knowledge/program/fragments.yaml fr.painting-7b] |
| `interiors.ffe`, `.internal-glazing`, `.acoustic-separation`, `.room-acoustics` | — | Tenant fit-out or not needed |

### `fire-passive`

| System | Action | Why |
|---|---|---|
| `fire-passive.compartments` **−** | — | One compartment under the large isolated building provisions; a design question for the BCA consultant and fire engineer, not a construction item. In the default list, so the app makes a Passive fire protection cost line [K: knowledge/costs/elements.yaml code 06] |
| `fire-passive.rated-elements`, `.penetrations`, `.external-walls`, `.openings`, `.structural-fire-protection`, `.linings-fire-hazard` | — | Type C with setbacks of more than 3 m: no rated walls in the base (verified: false). A wall built close to a boundary would need a fire rating |

### `fire-active`

| System | Action | Why |
|---|---|---|
| `fire-active.sprinklers` **+** | new | ESFR roof sprinklers to AS 2118.1 or FM Global data sheets [K: knowledge/clusters/fire/rules.yaml rule.as2118.sprinkler-design]. Required by C3D4(1)(b) and E1D13 (combustible goods stored above 4 m, over 1,000 m³ in aggregate, in a compartment over 2,000 m² or 12,000 m³), NCC 2022 Vol One (verified: true) |
| `fire-active.fire-water` | new | Sprinkler and hydrant tanks, diesel and electric pumps, booster [K: knowledge/clusters/fire/rules.yaml rule.as2941.fire-pumps] |
| `fire-active.hydrants` | new | Ring main, external hydrants, hose reels [K: knowledge/clusters/fire/rules.yaml rule.as2419.hydrant-design, rule.ncc.hose-reel-provision] |
| `fire-active.occupant-warning` **+** | new | Building occupant warning activated by the sprinkler system (verified: false) [D] |
| `fire-active.emergency-lighting` | new | Emergency and exit lighting |
| `fire-active.portable-extinguishers` | new | Extinguishers [K: knowledge/clusters/fire/rules.yaml rule.as2444.extinguishers] |
| `fire-active.detection`, `.smoke-control`, `.fire-control`, `.special-suppression`, `.smoke-alarms` | — | Not in the base. Smoke control and a fire control centre (E1D15(b), C4) come with trigger `large-isolated-building-over-limit`; special suppression with `dangerous-goods-storage` |

### `mechanical`

| System | Action | Why |
|---|---|---|
| `mechanical.air-conditioning` | new | Office split or VRF systems |
| `mechanical.local-exhaust` | new | Amenities exhaust |
| `mechanical.ventilation` **+** | new | Office outside air (AS 1668.2) and warehouse roof ventilators [K: knowledge/clusters/services-wet-air/rules.yaml rule.as1668-2.outdoor-air-rates] |
| other `mechanical.*` | — | No central plant, BMS, car park ventilation or process systems |

### `hydraulic`

| System | Action | Why |
|---|---|---|
| `hydraulic.cold-water`, `.hot-water`, `.sanitary` | new | Amenities services |
| `hydraulic.roof-drainage` | new | Large roof: box gutters, siphonic or gravity downpipes, overflows [K: knowledge/clusters/services-wet-air/rules.yaml rule.pca.roof-drainage-design] |
| `hydraulic.non-potable-water` **+** | new | Rainwater tank reuse for toilets and irrigation, often a DCP control [D] |
| `hydraulic.backflow-prevention` **+** | new | Containment devices at the water and fire services [K: knowledge/clusters/services-wet-air/rules.yaml rule.pca.backflow-prevention] |
| `hydraulic.gas`, `.trade-waste`, `.specialist-process` | — | No gas or trade waste in a general warehouse |

### `electrical`

| System | Action | Why |
|---|---|---|
| `electrical.utility-supply` | new | Kiosk substation, consumer mains; long lead [D] |
| `electrical.temporary-power` **+** | new | Builder's supply |
| `electrical.switchboards` | new | Main switchboard; long lead |
| `electrical.cabling-wiring` | new | Submains, power to docks, doors, pumps |
| `electrical.lighting` | new | High-bay LED with controls, external lighting [K: knowledge/clusters/electrical-comms/rules.yaml rule.ncc.lighting-controls] |
| `electrical.metering` | new | Energy monitoring where Section J requires it [K: knowledge/clusters/electrical-comms/rules.yaml rule.ncc.energy-monitoring] |
| `electrical.earthing-lightning` **+** | new | Lightning risk assessment and protection to AS/NZS 1768 [K: knowledge/clusters/electrical-comms/rules.yaml rule.as1768.lightning-protection] |
| `electrical.solar-pv`, `.ev-charging` | — | Trigger `green-star-solar`; EV readiness may be required for the car park (derivation pending) [K: knowledge/clusters/electrical-comms/rules.yaml rule.ncc.ev-charging-readiness] |
| `electrical.standby-power`, `.battery-storage`, `.process-hazardous` | — | Not in the base |

### `comms-security`

| System | Action | Why |
|---|---|---|
| `comms-security.carrier-lead-in` | new | NBN pit and pipe |
| `comms-security.structured-cabling` **+** | new | Office data |
| `comms-security.cctv` **+** | new | Site and dock cameras [D] |
| `comms-security.access-control` **+** | new | Office entry and gates [D] |
| other `comms-security.*` | — | Tenant |

### `vertical-transport`

Out of scope: a single-storey office. A two-storey office with a lift is trigger `two-storey-office-lift`.

### `access-egress`

| System | Action | Why |
|---|---|---|
| `access-egress.exits` | new | Exit layout and travel distances, often by performance solution [K: knowledge/costs/elements.yaml code 13 exits costed false] |
| `access-egress.egress-doors` **+** | new | Warehouse exit doors and hardware |
| `access-egress.accessible-access` | new | Accessible parking, path from the street, office entry [K: knowledge/clusters/fire/rules.yaml rule.as1428.accessible-paths] |
| `access-egress.accessible-facilities` | new | Accessible toilet and shower [K: knowledge/clusters/fire/rules.yaml rule.ncc.accessible-sanitary] |
| `access-egress.stairs-and-ramps` **+** | new | Dock stairs and the office entry ramp |
| `access-egress.maintenance-access` | new | Roof access, walkways, anchors, ladders [K: knowledge/clusters/fire/rules.yaml rule.as1657.maintenance-access] |
| `access-egress.barriers`, `.livable-housing`, `.pool-barriers` | — | Not applicable |

Outside every system: **storage racking** (AS 4084; verified: false), in-rack sprinklers, forklift charging
and the tenant's IT are tenant fit-out and sit under Exclusions. The knowledge maps "racking" to the slab
because rack loads govern its design [K: knowledge/works/system_terms.yaml Racking → substructure.slab-on-ground].

---

## 3. Existing-building interfaces

Not applicable: a new build on a vacant, benched lot. The brownfield case (demolishing an old factory with
an asbestos cement roof) is trigger `brownfield-demolition`.

---

## 4. Consultants and why

Novation follows C1 for services. "Need" is always, usually or optional for the base.

| Discipline | Stage engaged | Need | Novation | Why |
|---|---|---|---|---|
| `consultant.architect` | Concept | always | yes | Design lead, DA drawings, D&C tender documents and performance brief, office design; novated [K: knowledge/works/package_defaults.yaml baselines[0]] |
| `consultant.town_planner` | Concept | always | no | Planning advice, SEE, lodgement, assessment [K: knowledge/works/package_defaults.yaml complexity_additions[0]] |
| `consultant.surveyor` | Investigation | always | no | Detail and boundary survey, easements, services records [K: baselines[0]] |
| `consultant.geotechnical` | Investigation | always | no | Footing, slab and pavement parameters, fill, salinity [K: baselines[0]] |
| `consultant.civil` | Concept | always | yes | Earthworks, hardstand pavements, stormwater, OSD and water quality, crossings [K: baselines[0]] |
| `consultant.structural` | Concept | always | yes | Portal frame, panels, footings, slab [K: baselines[0]] |
| `consultant.traffic` | Concept | always | no | Traffic and parking report, swept paths for B-doubles [K: baselines[1]] |
| `consultant.bca` | Concept | always | no | BCA report for the DA and tender, large isolated building strategy [K: baselines[1]] |
| `consultant.certifier` | Before the CC | always | no | CC (staged), principal certifier, fire safety schedule, OC; owner-appointed [K: baselines[0]] |
| `consultant.fire_services` | Concept | always | option | Fire-water strategy, pressure and flow enquiry, tank and pump room, booster, sprinkler concept for the D&C brief [K: baselines[1], scope_additions[1]] |
| `consultant.fire_engineer` | Schematic | usually | option | Performance solutions and FRNSW consultation; no package rule names it [D] |
| `consultant.electrical` | Schematic | usually | option | DNSP application, substation siting, external lighting spill for the DA [K: baselines[0]] |
| `consultant.hydraulic` | Schematic | usually | option | Sydney Water servicing, fire-water supply, rainwater reuse [K: baselines[0]] |
| `consultant.access` | Schematic | always | no | DA and CC access reports [K: baselines[1]] |
| `consultant.landscape` | Schematic | always | yes | DA landscape plan for the street setback; no package rule [D] |
| `consultant.acoustic` | Schematic | usually | no | Noise impact assessment for 24-hour truck movements and plant (NSW Noise Policy for Industry; verified: false); no package rule [D] |
| `consultant.esd` | Schematic | usually | no | Section J report for the office and lighting; council ESD controls; no package rule (C2) [D] |
| `consultant.quantity_surveyor` | Concept | always | no | Feasibility, cost estimate for the DA fee on a development over $3M (verified: false), tender evaluation; no package rule [D] |
| `consultant.waste` | Approval documentation | usually | no | Operational and C&D waste management plans for the DA; no package rule [D] |
| `trade.main_works` | Procurement | always | — | D&C head contract (AS 4902); proposed only from gaps on accepted work items [K: internal/procurement/proposed.go ProposeHeadContract] |

**The app also suggests `consultant.mechanical`** (baselines[1] and scope_additions[2]). The base does not
need one: the office air-conditioning and amenities exhaust are design-and-install by the builder's
mechanical subcontractor, and Section J sits with the ESD consultant. [D]

**Novation default.** The app gives every suggestion novation `option` under D&C
[K: internal/procurement/packages.go DefaultNovation]. The expected answer is `yes` for the architect,
structural, civil and landscape; `option` for fire services, fire engineer, electrical and hydraulic; `no`
for everyone else. Only the user can choose `yes`.

---

## 5. Approvals and authorities

**Pathway.** A local DA to council, assessed by council staff (a regional planning panel only when the
capital investment value exceeds $30M; verified: false). A warehouse or distribution centre over a CIV of
$50M can be State significant development (verified: false); that is outside this type's range. Complying
development under the Industrial and Business Buildings Code is trigger `cdc-industrial-code`. Under SEPP
(Transport and Infrastructure) 2021 a warehouse above the Schedule 3 thresholds is traffic-generating
development referred to TfNSW (verified: false): trigger `traffic-generating-development`.

**DA documents.** Architectural plans, survey, SEE, traffic and parking report, civil stormwater concept
with OSD and MUSIC modelling, landscape plan, BCA report, preliminary fire safety strategy, geotechnical
and salinity report, noise impact assessment, access report, waste management plan, Section J or ESD
statement, cost estimate report, and the estate developer's validation report or a statement that the land
is not contaminated (SEPP (Resilience and Hazards) 2021 Chapter 4; verified: false). [D]

**Certificates.**

1. Construction certificates, usually staged: early works (bulk earthworks, civil), then the building.
   The fire safety schedule is attached [D]. On D&C both CCs follow award and the contractor's design;
   the PC is appointed and the notice of commencement given after the early-works CC, and the building
   CC follows the principal's review and the FRNSW consultation. Where the developer lets bulk earthworks
   under a separate early-works contract before the D&C award, the early-works CC moves ahead of the
   award (a variant, not the base). Both CCs sit in construction documentation (C5).
2. Fire safety performance solutions: the certifier consults FRNSW before issuing the CC (EP&A (Development
   Certification and Fire Safety) Regulation 2021; clause verified: false). Fire safety system designs are
   prepared or endorsed by a competent fire safety practitioner (verified: false). [D]
3. Occupation certificate, with the final fire safety certificate and the Section 73 compliance
   certificate [K: knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate].
   The owner then lodges annual fire safety statements; fire systems are maintained to AS 1851
   [K: knowledge/clusters/fire/rules.yaml rule.as1851.fire-system-maintenance].
4. Positive covenant and restriction on title for the OSD and water-quality devices (verified: false). [D]

**Authority applications.** Sydney Water: Section 73 through a Water Servicing Coordinator, building plan
approval, the pressure and flow statement for fire water, backflow device registration. DNSP: connection
application, kiosk substation design by an accredited service provider, easement, energisation. NBN.
Council: s138 Roads Act consent for the crossings and footpath (verified: false), s68 Local Government Act
approval for the stormwater connection where council requires it (verified: false), road occupancy for
deliveries. Long service levy: 0.25% of the cost of building work of $250,000 or more, including GST,
paid before work starts (C4, checked against nsw.gov.au). SafeWork NSW: precast and tilt-up erection
and crane work are high-risk construction work needing a SWMS (NSW WHS Regulation 2025, which replaced
the 2017 Regulation (C4); clause verified: false). [D]

**Inspections and hold points.** Owner ruling R7 covers Class 1 and 10 only. For a Class 5 to 9 building
the principal certifier's mandatory inspections are after excavation and before the first footing,
before covering stormwater drainage connections, and a final inspection before the OC (cl 61 EP&A
(Development Certification and Fire Safety) Regulation 2021; clause verified: false). Source: Bega Valley
Council, "About inspections", citing cl 61
(https://www.begavalley.nsw.gov.au/Plan-and-build/Building-and-certification/About-inspections). "At the
commencement of the building work" is the 2000 Regulation's wording (cl 162A(4)), now replaced. As R7
says for Class 1, the CC lists the inspections that apply. The structural
engineer inspects footings, slab reinforcement, panel connections and the steel frame and certifies them
for the OC; the geotechnical testing authority certifies fill (Level 1, AS 3798)
[K: knowledge/clusters/structure/rules.yaml rule.project.structural-inspections, rule.as3798.controlled-fill].
Fire systems are pressure tested and commissioned with FRNSW witnessing the booster and hydrant tests
(verified: false). [D]

---

## 6. Cost plan shape

Sections follow the app's four (`fees_and_charges` is the app's `authority`)
[K: internal/costs/totals.go CostSections]. No amounts.

**Fees and charges.** DA fees and contributions (s7.11 or s7.12; verified: false) [K:
knowledge/costs/starter_lines.yaml da-fees]; the Housing and Productivity Contribution for industrial
development in Greater Sydney (verified: false); construction certificate and certifier statutory fees
[K: cc-fees]; service authority connection fees (Sydney Water, DNSP, NBN) [K: service-authority, one generic
line]; long service levy; council road-reserve fees and bonds for the crossings.

**Consultants.** One fee line per owner-side consultant in §4, nineteen in all. After novation the builder
pays the novated consultants; the owner's lines cover the pre-novation stages only. The app gives a novated
package pre and post sides on its stages [K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §2.3], but
its fee line still carries the whole package.

**Construction (by element, as the app groups them)** [K: knowledge/costs/elements.yaml]: preliminaries and
margin (no app line); the D&C contractor's design after novation (no app line); 01 Site works and external
works, which a QS would split to show hardstand and pavements, docks, OSD and crossings; 02 Substructure;
03 Structure; 04 Building envelope; 05 Interiors and finishes; 07 Active fire protection; 08 Mechanical; 09
Hydraulic; 10 Electrical, including the kiosk substation and contestable works; 11 Communications and
security; 13 Access and egress children (accessible paths, accessible facilities, maintenance access,
stairs and ramps, egress doors); provisional sums. 06 Passive fire protection should be empty in the base.
A stated budget is spread by the `industrial_warehouse` split, which is low confidence and `verified:
false` [K: knowledge/costs/budget_splits.yaml industrial_warehouse].

**Contingency.** Design contingency, construction contingency, escalation, exclusions (tenant racking,
in-rack sprinklers, forklift charging, IT, tenant fit-out) [K: knowledge/costs/starter_lines.yaml].

---

## 7. Program shape

Skeleton `sk.new-dc-da`: investigation → concept design → approval documentation → approvals →
procurement → construction documentation → construction → completion → defects
[K: knowledge/program/skeletons.yaml sk.new-dc-da]. The app has a full set of Class 7b fragments for this
job [K: knowledge/program/fragments.yaml fr.*-7b].

**Activities that drive the job.** [D unless tagged]

1. *Investigation:* planning certificate and title, detail survey, services search, geotechnical
   investigation, the Sydney Water pressure and flow enquiry, and the DNSP load enquiry, all before concept.
2. *Concept and DA:* brief (often the tenant's), concept, client review, a pre-DA meeting, design
   development, the fire safety strategy, then the DA reports in parallel and lodgement
   [K: fr.concept-design-7b, fr.design-development-7b, fr.approval-docs-7b, fr.da-lodgement,
   fr.da-exhibition, fr.da-assessment].
3. *D&C procurement:* tender package with performance brief, tender period, evaluation, award, novation
   [K: fr.tender-docs-7b, fr.tender-period-7b, fr.tender-evaluation-7b, fr.contract-award, fr.novation].
4. *Construction documentation (the builder):* design completion by the novated team and D&C
   subcontractors, the final fire engineering report and FRNSW consultation, Section 73 application,
   long service levy, then the early-works CC, the PC appointment and notice of commencement, and the
   building CC [K: fr.construction-docs-7b, fr.construction-certificate (the app puts it in approvals;
   it matches the early-works CC, C5; see gaps)].
5. *Construction:* site establishment, erosion control, bulk earthworks and compaction testing,
   stormwater and OSD, footings, portal frame (fabrication lead), precast panels (manufacture lead), roof,
   cladding, slab under the roof, dock doors and levellers, fire pumps and tanks, sprinklers and hydrants,
   switchboard and substation, power and lighting, office fit-out, amenities, hardstand and car park,
   crossings, landscape, line marking [K: fr.site-establishment-7b, fr.erosion-control, fr.earthworks-7b,
   fr.stormwater-7b, fr.footings-7b, fr.steel-7b, fr.roof-7b, fr.cladding-7b, fr.slab-7b,
   fr.vehicle-doors-7b, fr.hydrants-7b, fr.sprinklers-7b, fr.switchboard-major, fr.utility-major,
   fr.services-connections-major, fr.electrical-7b, fr.lighting-7b, fr.hydraulic-sanitary-7b,
   fr.landscape-7b, fr.painting-7b].
6. *Completion:* fire systems commissioning and integrated testing, works-as-executed and OSD
   certification, Section 73 certificate, final fire safety certificate, practical completion, OC,
   handover [K: fr.practical-completion, fr.occupation-certificate, fr.handover].
7. *Defects:* defects inspection, defects liability period, final inspection [K: fr.defects-inspection,
   fr.defects-liability-end].

**Logic that matters.** The fire strategy precedes the DA site plan. The substation and services
applications must start in design, not at construction start (the app attaches them to the construction
stage start; see gaps). Footings → portal frame → precast panels → roof → slab under the roof suits
off-site precast [K: fragments.yaml fr.slab-7b attach after fr.roof-7b]; tilt-up reverses it (slab first,
panels cast on it). Stormwater and services under the hardstand precede the pavement. Sprinkler
commissioning precedes the fire safety certificate, which precedes the OC.

**Long-lead items.** Kiosk substation and energisation; portal frame steel; precast panels; fire pumps and
tanks; main switchboard; dock levellers and doors; long-length roof sheeting. The app has lead-time
durations for steel, vehicle doors, the switchboard and the utility [K: knowledge/program/durations.yaml
du.steel-lead-7b, du.vehicle-doors-lead-7b, du.switchboard-lead-major, du.utility-lead-major]; none for
precast panels or fire pumps.

---

## 8. RFP shape

What each consultant's RFP must carry for this job, and what it must not. The template rows each item
maps to are in `shape.yaml`. On a D&C profile the architecture, structural and civil boundaries end at
`dnc_tender`, so every later stage, including post-novation, is priced as an option
[K: knowledge/rfp/architecture.yaml, structural.yaml, civil.yaml boundary]. The services templates end
at `design`, so their tender stage is priced as an option.

The ancillary office makes the app derive `flag.mixed_use` = yes, because the profile has two classes
[K: internal/rfp/drivers.go deriveFlags]. That switches on mixed-use rows (architecture class and use map,
access shared routes, fire engineering occupant basis, waste shared facilities) that a warehouse with an
ancillary office does not need. `flag.mixed_use` is derived only (`FlagHome("mixed_use")` is
`HomeDerived`, internal/rfp/homes.go), so a user cannot override it. Removing class 5 from
`det.ncc_class` is the only cure, and it also excludes the architect's office interiors stage and drops
`base_interiors` (Go run, 4 November 2026).

| Consultant | Must carry | Must not carry |
|---|---|---|
| Architect | Concept and schematic; DA documentation and planning response; design development; office interiors and finishes; D&C tender documents with performance brief and design responsibility matrix; post-novation documentation; BIM model; certifier liaison | House rows; traditional tender or bidding; regulated design; tenant fit-out as base; hardstand design (civil's) |
| Structural | Design basis (wind, importance level); portal frame; precast panels and connections; footings; industrial slab for racking and forklift loads with a joint and flatness specification; delegated design schedule; D&C tender documents; post-novation; construction-phase inspections | Basement retention; dilapidation; existing structure assessment; regulated design; residential framing |
| Civil | Earthworks and cut-fill; heavy-duty hardstand pavement design; stormwater, OSD and MUSIC water quality; crossings and levels; erosion control; D&C tender design; post-novation; as-built and OSD certification | Public road works; flood report; NorBE; trunk drainage; regulated-design declarations |
| Town planner | Preliminary advice; pre-lodgement; SEE; lodgement; assessment; determination review | Clause 4.6; panel attendance; court |
| Traffic | Parking and access advice; traffic report; B-double swept paths; assessment response; conditions review; car park and hardstand certification | Intersection modelling; green travel plan; public road design |
| BCA | A1–A3 advisory; large isolated building strategy; performance solution schedule | Package B statutory functions; supervision |
| Certifier | B1–B4: staged CCs, principal certifier, fire referrals, fire safety schedule and final fire safety certificate, OC | Package A advisory; design; supervision |
| Fire services | Fire-water strategy and pressure and flow enquiry; tanks, pumps, booster; ESFR storage sprinklers and hydrants by area; occupant warning; plant interfaces; D&C performance documentation; declarations | Fire strategy and performance solutions (fire engineer's); existing system investigation |
| Fire engineer | Strategy; performance design brief with FRNSW; FER for scheduled solutions (perimeter access, travel distance, smoke hazard management); verification before OC | Wet and dry fire design; mixed or vulnerable occupant basis; special hazards (base) |
| Electrical | Concept; DNSP application and substation; distribution; high-bay and external lighting with spill control; metering; ICT and security pathways; D&C documentation | Contestable ASP design; fire detection design; HV studies |
| Hydraulic | Concept; Sydney Water servicing and Section 73 coordination; water, sewer, amenities; roof drainage; rainwater reuse; D&C documentation | Booster pumps; trade waste; gas; onsite treatment; site stormwater (civil's) |
| Access | Concept advice; DA and CC access reports; facilities review | Shared routes for mixed uses; housing; occupied-site staging |
| Geotechnical | Investigation plan; boreholes and pits; foundation, slab and pavement parameters; salinity; preliminary report for the DA | Contamination; groundwater monitoring; acid sulfate soils |
| Surveyor | Detail and contour survey; boundary and easements; services records | Strata or subdivision; interior scan; movement monitoring |
| Landscape | DA landscape plan to the DCP; developed design; D&C documentation; irrigation brief | Podium build-ups; public-domain package; tree assessment |
| QS | Cost basis; feasibility; cost estimate report for the DA; design cost plan; D&C tender evaluation | Valuation or depreciation; remediation costing; mixed-use allocation |
| Waste | Operational and C&D waste management plans; bin enclosure and collection; consent review | Hazardous waste classification; equipment supply |
| Acoustic (no template) | Noise impact assessment for operations and plant; conditions review; pre-occupation verification if conditioned | Sound insulation between dwellings |
| ESD (no template) | Section J (DTS or JV3) for the office; lighting and energy monitoring; council ESD statement | Green Star or NABERS (base) |

---

## 9. Triggers

Fifteen triggers. For each: how documents show it, how they deny it, and what it adds. The full keys are
in `shape.yaml`.

1. **`traditional-lump-sum`.** "Fully documented for a lump-sum tender", "construct only". Denied by
   "design and construct under AS 4902 with novation". Adds traditional tender, for-construction and
   construction-support stages; removes novation. The app moves to `sk.new-traditional-da`.
2. **`cdc-industrial-code`.** "Complying development under the Industrial and Business Buildings Code".
   Denied by "a DA will be lodged". Replaces the DA and CC with a CDC. D&C plus CDC gives `sk.generic`;
   the town-planning and certifier RFPs block on `cdc`.
3. **`large-isolated-building-over-limit`.** "A single fire compartment of 24,000 m²", "exceeds
   18,000 m², so smoke exhaust and a fire control centre are required". Denied by "floor area 8,000 m²;
   volume about 100,000 m³, below the C3D4(1) limits". Sprinklers and perimeter vehicular access are
   required on both sides of the limit (C3D4(1)(b), C3D4(2)), so they cannot tell the variant from the
   base. Adds a fire control centre only where the total floor area exceeds 18,000 m² (E1D15(b), C4);
   smoke exhaust or smoke-and-heat vents where the area or volume exceeds the limits (E2D10(2)), smoke
   exhaust only where the ceiling is over 12 m, so smoke exhaust at the 12 m haunch; and the fire
   engineer as always needed. NCC 2022 Vol One Parts C3, E1, E2 (verified: true).
4. **`tilt-up-on-site`.** "Tilt-up panels cast on the slab". Denied by "precast panels manufactured off
   site" or "metal-clad walls". Reverses the slab and wall sequence and adds casting and bracing.
5. **`traffic-generating-development`.** "Traffic generating development under Schedule 3", "referred to
   TfNSW", "SIDRA". Denied by "below the Schedule 3 thresholds, so it will not be referred to TfNSW" (the shape negation was reworded on 7 November 2026 so the tfnsw_referral trigger `referred to tfnsw` fires; ST04 review). Adds surveys and intersection modelling
   and the TfNSW referral.
6. **`classified-road-frontage`.** "Fronts the Great Western Highway, a classified road", "left-in
   left-out". Denied by "access from a local industrial road". Adds public road works, a road safety audit
   and TfNSW approvals.
7. **`flood-affected`.** "Within the flood planning area", "overland flow path". Denied by "no flood
   notation on the s10.7 certificate". Adds a flood report and flood-compatible design.
8. **`contaminated-land`.** "Former market garden", "PSI found elevated TRH and asbestos in fill", "site
   audit statement". Denied by "the estate developer's validation report". Adds the environmental
   consultant, investigations and remediation.
9. **`bushfire-prone`.** "Mapped as bush fire prone land", "APZ to the riparian corridor". Denied by "not
   bush fire prone". Adds the bushfire consultant and construction measures.
10. **`sloping-site-retaining`.** "Cut of 6 m at the rear", "keystone retaining walls on the boundaries",
    "imported fill". Denied by "generally level". Adds retaining walls and fill supervision.
11. **`aboriginal-heritage`.** "Artefacts found in the due diligence assessment", "AHIP". Denied by "the
    AHIMS search found no sites". Adds the archaeologist and an AHIP before earthworks.
12. **`brownfield-demolition`.** "Demolish the existing 1970s factory with an asbestos cement roof".
    Denied by "vacant lot in a new estate". Adds hazmat, demolition and clearance.
13. **`two-storey-office-lift`.** "Two-storey office with a passenger lift". Denied by "single-storey
    office". Adds the lift, stair and balustrades (a Class 5 lift fragment exists).
14. **`dangerous-goods-storage`.** "Class 3 flammable liquids above manifest quantities", "preliminary
    hazard analysis". Denied by "no dangerous goods above minor quantities". Adds hazard studies, special
    suppression and firewater containment.
15. **`green-star-solar`.** "5 Star Green Star Buildings", "rooftop solar PV". Denied by "no rating
    targeted". Adds the ESD consultant as always needed, solar and EV charging.

---

## 10. Keyword traps and negations specific to this type

All regex triggers are case-insensitive [K: internal/knowledge/profile.go compile].

1. **"Floodlights".** The `flood_hazard` trigger is `\bflood`, so "LED floodlights to the hardstand"
   raises a flood question [K: knowledge/determinants.yaml flood_hazard triggers]. Jev must answer no.
2. **"Bore logs".** `water_supply_source` fires on `\bbore\b`; geotechnical reports say "bore logs" and
   "Bore 3" [K: knowledge/determinants.yaml water_supply_source triggers]. ("Borehole" does not match.)
3. **"Excavated to a depth".** The basement flag's pattern `excavat\w* to a depth` matches "test pits
   excavated to a depth of 3 m" in the geotechnical report; "pits excavated near the northern boundary"
   matches `excavation_adjoining` [K: knowledge/profile/flag_readings.yaml basement, excavation_adjoining].
   A yes would add retention, dilapidation and shoring rows.
4. **"Building a".** `multi_building` matches `\bbuildings? [a-d]\b`, so "building a new warehouse" and
   "Building A" in an estate both match [K: flag_readings.yaml multi_building]. "Stage 2 of the estate"
   matches `staged`.
5. **"Storage … fire".** `fire_special_hazards` matches storage within 90 characters of fire, so "high-bay
   storage protected by ESFR fire sprinklers" raises it [K: flag_readings.yaml fire_special_hazards].
   "Structural steel portal frame with no fire rating" raises `fire_specialist_systems` the same way.
6. **"Road reserve".** `public_road` matches "road reserve", which every crossing description uses
   [K: flag_readings.yaml public_road]. Documents must say "no works in the public road other than the
   vehicle crossings".
7. **"TfNSW" and "traffic generating development"** raise `state_road` and `tfnsw_referral` even in a
   denial ("referral to TfNSW is not required").
8. **"Contamination" in a stormwater sentence.** `potentially_contaminated_land` fires on the literal `contaminat`, so
   "hydrocarbon contamination" or "contaminated runoff" in a stormwater quality report raises it; "contaminants" does not
   match (contamina-n-ts) [K: determinants.yaml potentially_contaminated_land]. Corrected 7 November 2026 (ST04 review).
9. **"Aboriginal heritage"** raises the `heritage_status` question (`heritage`), whose options are
   built heritage only [K: determinants.yaml heritage_status].
10. **"Public exhibition"** in a notification letter sets `community` [K: flag_readings.yaml community];
    "lodge" sets `architect_lodges` (no effect off the house module).
11. **"No demolition is required"** adds `site.demolition` through `implied_by strip-out`, with action
    `new` [K: knowledge/profile/scope_defaults.yaml implied_by strip-out].
12. **Two classes make "mixed use".** Stating "Class 7b and Class 5" derives `flag.mixed_use` = yes
    [K: internal/rfp/drivers.go deriveFlags]. The warehouse is not a mixed-use building. The flag is
    derived only (internal/rfp/homes.go FlagHome), so a user cannot override it; removing class 5 also
    removes the office interiors stage.
13. **"Kitchenette"** does not match the kitchen pattern (`\bkitchens?\b`); "staff kitchen" does.
14. **"Fire wall"** in a warehouse is a compartment wall, not an external wall; "sprinkler" in the
    tenant's in-rack system is outside the base scope.

---

## 11. Gaps in the built-in knowledge

1. **Default scope misses core warehouse items.** No sprinklers, occupant warning, loading docks, public
   domain, waste storage, temporary works, temporary power, rainwater, backflow, egress doors, stairs and
   ramps, CCTV, access control or ground investigation [K: knowledge/profile/scope_defaults.yaml
   classes[3]]. The app will miss the ESFR sprinkler work item, so `fr.sprinklers-7b` never fires, though
   the cost line 07 appears from hydrants.
2. **No program content for the heaviest elements.** `structure.precast`, `site.vehicle-access`
   (hardstand), `fire-active.fire-water` (pumps and tanks), `site.loading-docks`, `hydraulic.roof-drainage`
   in 7b, and the office interiors and mechanical items have no 7b fragment [K:
   testing/synthetic/reference/index/program-fragments.md leaf systems with no work-item fragment].
3. **Tilt-up sequence.** `fr.slab-7b` follows the roof; there is no tilt-up variant that pours the slab
   first and casts panels on it [K: knowledge/program/fragments.yaml fr.slab-7b attach].
4. **D&C program order.** One CC fragment, `fr.construction-certificate`, in `approvals` before D&C
   procurement; no staged CCs. Expected: an early-works CC, then a building CC, both after award; `fr.construction-docs-7b` ends with "Client sign-off for
   tender", which follows award in D&C [K: knowledge/program/skeletons.yaml sk.new-dc-da; fragments.yaml
   fr.construction-certificate, fr.construction-docs-7b].
5. **Authority applications start too late.** `fr.utility-major` and `fr.services-connections-major`
   attach to the construction stage start [K: fragments.yaml attach after stage_start].
6. **No hold points fire** (confirmed by a Go run, default and expected scope). The reinforcement
   interface needs a row tagged `structure.concrete`, which no 7b fragment carries; the
   sprinkler-before-ceilings interface needs an `interiors.ceilings` row, and the only ceilings fragment
   (`fr.ceilings-fitout`) needs refurb; the frame interface needs `structure.timber`
   [K: knowledge/clusters/structure/interfaces.yaml if.reinforcement-inspected-before-pour;
   clusters/fire/interfaces.yaml if.sprinkler-pressure-test-before-ceilings]. `fr.fire-stopping` applies
   to class 5, so the fire-stopping hold point would fire (before `fr.painting-7b`) if
   `fire-passive.penetrations` were added [K: knowledge/program/fragments.yaml fr.fire-stopping].
7. **No process rows** for the fire strategy, FRNSW consultation, commissioning and integrated testing, the
   fire safety certificate, Section 73, WAE and OSD certification, the positive covenant, notice of
   commencement, long service levy or the Class 5–9 critical-stage inspections (first footing,
   stormwater, final).
8. **Consultants never suggested:** fire engineer, landscape, acoustic, ESD, QS and waste. Contamination
   and flood rules are dead (`contamination_level`, `flood_exposure`) [K:
   knowledge/works/package_defaults.yaml complexity_additions[2], [6]].
9. **Over-suggested:** mechanical consultant; the D&C novation default of `option` for every consultant.
10. **RFP rows that misread a warehouse.** Derived mixed use from two classes; fire-services `hazards`
    (high storage) excluded; civil `pavement_design` only an option; D&C stages after `dnc_tender` priced
    as options; super-flat floor a client flag that cannot be keyed [K: knowledge/rfp/fire-services.yaml
    hazards; civil.yaml pavement_design; structural.yaml super_flat_floor].
11. **No RFP template** for acoustic or ESD [K: testing/synthetic/reference/index/rfp-templates.md].
12. **Cost plan.** No HPC, long service levy, road-reserve fees, preliminaries, D&C design or provisional
    sum lines; hardstand, docks and the substation roll into generic elements; an empty Passive fire
    protection line; the budget split is low confidence [K: knowledge/costs/starter_lines.yaml;
    budget_splits.yaml industrial_warehouse].
13. **Profile.** No key for perimeter vehicular access, compartment volume or design vehicle; storage
    height and the warehouse scale fields (clear height, dock doors, hardstand, office percent) feed no
    generator [K: knowledge/determinants.yaml storage_height; knowledge/profile/taxonomy.yaml warehouse
    scale_fields].
14. **No public holidays** or Christmas shutdown [K: knowledge/program/calendars.yaml].

---

## 12. Sources and unverified claims

**Knowledge files opened:** `knowledge/profile/scope_defaults.yaml`, `taxonomy.yaml`, `flag_readings.yaml`,
`project_facts.yaml`, `project_details.yaml`; `knowledge/determinants.yaml`; `knowledge/works/package_defaults.yaml`,
`system_terms.yaml`; `knowledge/costs/elements.yaml`, `starter_lines.yaml`, `budget_splits.yaml`;
`knowledge/program/skeletons.yaml`, `fragments.yaml`, `durations.yaml`; every `knowledge/rfp/*.yaml` named
in the shape (through `reference/index/rfp-templates.md`, and the raw files for warehouse wording);
`knowledge/tables/compartment_limits.yaml`, `type_of_construction.yaml`; `knowledge/clusters/fire/rules.yaml`,
`interfaces.yaml`; `clusters/structure/rules.yaml`, `interfaces.yaml`; `clusters/services-wet-air/rules.yaml`;
`clusters/electrical-comms/rules.yaml`; `internal/rfp/drivers.go`; `internal/knowledge/profile.go`.

**Verified by the knowledge (instrument-checked rows):** NCC 2022 Table C2D2 (Class 7b, rise 1 → Type C)
and Table C3D3 (Class 7b, Type C: 2,000 m² and 12,000 m³) [K: knowledge/tables/*.yaml, `verified: true`].
The app returns unknown for type of construction and both compartment limits today, because the rules
and tables are unreviewed (Go run, `profile.Build`).

**Verified against the instrument by the T04 reviewer (`verified: true`):** NCC 2022 Vol One, ABCB site:
C3D4(1)(b) and C3D4(2) (18,000 m², 108,000 m³; sprinklers and perimeter vehicular access), C3D5(2)
(access at least 6 m wide, furthest boundary within 18 m), E1D13 (sprinklers for storage above 4 m),
E1D15(b) (fire control centre for Class 6–9 over 18,000 m² total floor area; also above 25 m effective
height, C4), E2D10(1) and (2) (smoke hazard management above the limits; smoke exhaust only above a 12 m
ceiling). Sources: https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c3-compartmentation-and-separation,
…/e-services-and-equipment/part-e1-fire-fighting-equipment, …/e-services-and-equipment/part-e2-smoke-hazard-management.
Long service levy rate, threshold and timing (C4).

**Unverified (`verified: false`), for the owner:**

- Occupant warning triggers for Class 7b.
- The NCC 10 per cent concession for an ancillary office; Section J applicability to an unconditioned 7b.
- Zone names (IN1 or E4), permissibility, the $30M regional panel and $50M SSD thresholds.
- SEPP (Transport and Infrastructure) 2021 Schedule 3 thresholds and the referral clause.
- EP&A (Development Certification and Fire Safety) Regulation 2021: FRNSW consultation, competent fire
  safety practitioners, the Class 5–9 critical-stage inspections in cl 61 (first footing, stormwater,
  final, from Bega Valley Council's page; the Regulation itself could not be opened).
- Roads Act s138, Local Government Act s68, the positive covenant provisions, the Housing and
  Productivity Contribution, the $3M cost estimate report.
- NSW WHS Regulation 2025 clause numbers for high-risk construction work (precast and tilt-up); AS 4084
  for racking.
- NSW Noise Policy for Industry; SEPP (Resilience and Hazards) 2021 Chapters 3 and 4; Planning for Bush
  Fire Protection 2019 and s4.14; National Parks and Wildlife Act AHIP.

**Claims checked by a Go run (round 1):** no Class 7b hold point fires (reviewer, both scopes);
`flag.mixed_use` derives yes from one part with two classes (reviewer; author re-ran `rfp.Resolve`);
the RFP `app_differs` for all 17 templated RFPs (author, `rfp.Resolve` with scope-by-stage off, so the
rules' own answers are compared). **Least sure of:** the Class 5–9 inspection list, taken from a council
page, not the Regulation.

## Shape corrections after run `20261107-t03-t04-t06` (7 November 2026)

From FINDINGS "Shape and key gaps" for that run.

- `cdc-industrial-code` sets `skeleton: sk.generic`, as its `why` already said. It restates `expect`:
  - architect `planning_response` absent;
  - traffic `assessment_response` and `conditions_review` absent;
  - BCA `a2_planning` absent (`b2_construction_approval` absent; `b2_cdc` absent with `app_differs` option);
  - certifier `b2_construction_approval` and `a2_planning` absent, `b2_cdc` included (F9).
  The matching unkeyed phrases ("Planning response", "Assessment response", "Conditions review", "A2 DA BCA
  report", "B2 construction certificate assessment ...") leave those RFPs for the variant. "CDC assessment and
  determination" is keyed to `stage.b2_cdc`.
- `cdc-industrial-code` cost plan: "Construction certificate and certifier fees" stays and stands for the CDC fee
  (`aliases: ["CDC fee"]`), as T03 and T06 alias it. The variant no longer removes it or adds "CDC fee".
- `flood-affected` restates the civil positions `flood_report`, `flood_evacuation` and `trunk_drainage` as I, and
  adds "Trunk drainage" to `must_include`. The base `must_not_include` "Flood report" and "Trunk drainage" are
  dropped for the variant. The complex service-level rows are already I in the base (public road works, R15).
- The base profile gains `det.site_class: M` and `det.saline_soil: true`. The ST04 scenarios already key both
  from the geotechnical report.
