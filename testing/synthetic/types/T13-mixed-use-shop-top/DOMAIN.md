# T13 Mixed-use shop-top housing: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (e1a5389 plus the working tree) and professional practice. Nothing in the
application or the knowledge was changed.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. A clause or section number I could not
check against the instrument is written `verified: false`. No prices, rates or week durations appear.
Owner rulings (`testing/synthetic/reference/OWNER_RULINGS.md`) apply; R3 (DBP Act for Class 2) and R4
(HBCF) are the ones this type leans on.

---

## 1. The job

**What it is.** A new mid-rise mixed-use building on a town-centre main street in Greater Sydney:
ground-floor retail (NCC Class 6) under apartments (Class 2), over a basement car park (Class 7a). The
app profile is `mixed / residential_retail / new` with `det.ncc_class: [2, 6, 7a]`
[K: knowledge/profile/taxonomy.yaml building_classes mixed residential_retail]. Base scenario:

| Item | Base value |
|---|---|
| Site | Two or three consolidated lots of about 1,500–2,500 m² in an E1 Local Centre or MU1 Mixed Use zone (zone names after the 2023 employment-zones reform; verified: false), fronting a local main street with a rear lane or side street. Existing one- and two-storey shops and a dwelling from the 1950s–1970s to be demolished. Nil-setback neighbours on one or both sides. |
| Building | Six storeys (rise in storeys 6, effective height under 25 m): ground-floor retail in three or four shell tenancies with a residential lobby, waste rooms, loading bay and substation; five residential levels of about 35–50 apartments; communal open space on the podium or roof; two basement levels of parking, storage and plant. |
| Construction | Reinforced concrete: bored or soldier-pile retention with shotcrete and temporary anchors, pad footings to rock, post-tensioned suspended slabs, a transfer slab at level 1 where the apartment grid meets the car-park and retail grid, concrete or masonry cores. Masonry, precast or FC-clad facade, aluminium windows and balcony doors, glazed retail shopfronts with an awning over the footpath, concrete roof with membrane. |
| Services | Sprinklered throughout, hydrants and booster, detection and occupant warning, two passenger lifts, mechanical car-park ventilation, split AC to apartments, kitchen-exhaust riser and grease-arrestor provision for food tenancies, separate retail metering, a padmount or chamber substation, gas to the retail tenancies, BASIX rainwater reuse. |
| Retail | Shell and core only: slab, shopfront, services stubs (power, water, sewer, gas, exhaust riser, grease-arrestor provision, sprinkler grid at shell level), no fit-out. Each tenant later fits out under its own approval. |
| Pathway | DA to council, determined by the local planning panel (Housing SEPP Chapter 4 development; verified: false), then construction certificates, often staged (early works, structure, full). |
| Procurement | Developer client. Design and construct lump sum (AS 4902) after DA and design development; architect and core engineers novated to the builder. Strata subdivision for sale. |
| Value | Inside the PROJECT_TYPES range of $15–80M; the shape names no amount. |

**Who procures it and how.** A private developer, often a special-purpose company with a financier,
appoints a development manager or PM, then the architect, planner and DA consultants (around 15–20
firms). The DA is the commercial gate: the design is taken to DA and design development by the developer's
team, then tendered as a D&C lump sum to three or four mid-tier builders, and the architect, structural,
civil, hydraulic, fire services, mechanical, electrical and landscape consultants are novated. The
developer keeps the planner, traffic, BCA adviser, principal certifier, QS, acoustic and ESD advisers on
its own side. A traditional lump-sum tender on full documentation is the main alternative (trigger
`traditional-lump-sum`). [D]

**What makes it go wrong.** [D]

1. **The DA.** Height, FSR or ADG non-compliances (building separation, solar access, deep soil,
   communal open space), a cl 4.6 request, design review panel comments, objectors, and a panel
   determination timetable the developer cannot control.
2. **Mixed-class compliance found late.** Fire separation between Class 6, Class 7a and Class 2;
   separate egress for the retail from the residential stairs; sound insulation of the floor between
   retail and apartments; and the retail kitchen-exhaust riser that nobody allowed space for. These change
   the plans after DA and trigger s4.55 modifications.
3. **Basement and neighbours.** Excavation to the boundary next to old nil-setback buildings: anchors
   need neighbours' or council's consent, dilapidation disputes, groundwater, rock removal rates,
   vibration complaints.
4. **The retail shell boundary.** Leasing agents sign tenants with requirements (grease arrestor size,
   exhaust riser, power, a supermarket's loading) after the D&C contract, so they become variations.
   Shell work left vague in the contract is the commonest dispute.
5. **DBP Act and RAB Act compliance.** Regulated designs and design compliance declarations not lodged
   before the work they cover, variations built without updated declarations, and the Building
   Commissioner's powers (stop-work, prohibition on OC) for a Class 2 building (R3).
6. **Waterproofing and facade defects.** Balconies, podium planters and the roof are the classic Class 2
   defects, now policed through the strata building bond and Building Commission audits.
7. **Authority long-lead items.** Ausgrid or Endeavour substation design and energisation, Sydney Water
   Section 73 and building-plan approval, NBN, and the occupation certificate depending on the fire
   safety certificate and the Section 73 certificate.
8. **Builder insolvency** in the mid-tier residential market, and the developer's financier's cost-to-
   complete tests.

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. Actions from `knowledge/works/actions.yaml`. The app
gives this profile the 62-system `mixed` category list, the same YAML anchor as the apartments list, all
with action `new` [K: knowledge/profile/scope_defaults.yaml categories[1] mixed]. Items marked **+** are
ones a competent PM expects that the default list lacks. The default list has nothing specific to retail
or to a car park beyond car-park ventilation.

### `site`

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | new | Set-out, level control, identification surveys; the strata plan is a consultant task (§4). Default. |
| `site.ground-conditions` **+** | investigate | Geotechnical investigation for the basement, rock levels and groundwater. Not default, so `fr.investigate-geotech` never fires. [K: knowledge/program/fragments.yaml fr.investigate-geotech] |
| `site.contamination` **+** | investigate | Preliminary site investigation: commercial land going to residential use (SEPP Resilience and Hazards Ch 4; verified: false). |
| `site.demolition` **+** | remove | Demolish the existing shops and dwelling, hazardous-materials removal first. Not default; `fr.demolition-2` fires only with action `remove`. [K: knowledge/program/fragments.yaml fr.demolition-2] |
| `site.adjoining-structures` | retain | Protect nil-setback neighbours: dilapidation, monitoring, party-wall and anchor consents. Default gives `new`. |
| `site.earthworks` | new | Bulk excavation and spoil classification and disposal. Default. |
| `site.erosion-sediment-control` | new | Default. |
| `site.stormwater-detention` | new | OSD tank in the basement or under the driveway, WSUD treatment, pump-out for basement drainage. Default. |
| `site.services-connections` | new | Water, sewer, gas, electricity and NBN lead-ins; Sydney Water Section 73. Default. |
| `site.vehicle-access` | new | Driveway ramp off the lane, basement parking layout to AS 2890.1 (verified: false). Default. |
| `site.loading-docks` **+** | new | Retail loading bay and waste-collection bay for a council or private medium rigid vehicle. Not default. |
| `site.waste-storage` | new | Separate residential and retail bin rooms, bulky-waste room, chute or bin-carting route, collection point. Default. |
| `site.public-domain` | new | Vehicle crossing, making good the footpath, street trees, awning over the footpath; Roads Act s138 consent (verified: false). No public road works in the base (trigger `public-domain-works`). Default. |
| `site.landscape` | new | Deep soil, podium and roof communal open space, street planting. Default. |
| `site.signage` **+** | new | Building identification and retail signage zones. Tenant signs are the tenant's. Not default. |
| `site.trees` | retain (trigger) | Only where street or neighbour trees are affected. |
| `site.flooding`, `site.acid-sulfate-soils` | out | Triggers. |
| `site.retaining-walls`, `site.pools`, `site.impact-protection` | out | Not in a typical scheme. |

### `substructure`

| System | Action | Why |
|---|---|---|
| `substructure.basement-retention` | new | Piled retention, shotcrete, anchors or props, seepage drainage (no groundwater in the base). Default. |
| `substructure.footings` | new | Pad and strip footings to rock. Default. |
| `substructure.slab-on-ground` **+** | new | Basement slab on ground. Not default; without it `if.reinforcement-inspected-before-pour` never fires. [K: knowledge/clusters/structure/interfaces.yaml if.reinforcement-inspected-before-pour] |
| `substructure.damp-waterproofing` | new | Drained or tanked basement walls and slab. Default. |
| `substructure.piles`, `.underpinning`, `.retaining-walls`, `.termite-management` | out | Piles and underpinning are triggers (poor ground, weak neighbours); concrete structure needs no termite system. |

### `structure`

| System | Action | Why |
|---|---|---|
| `structure.concrete` | new | PT slabs, columns, cores, transfer slab, roof slab. Default. |
| `structure.steel` **+** | new | Footpath awning frame, roof plant screens and canopy. Not default. |
| `structure.temporary-works` **+** | new | Tower crane base, back-propping, hoarding and gantry over the footpath. Not default. |
| `structure.design-basis` | out | A design deliverable, not a work item. |
| `structure.masonry`, `.precast`, `.timber`, `.alterations` | out | Not typical (precast is an alternative facade/wall system). |

### `envelope`

| System | Action | Why |
|---|---|---|
| `envelope.external-walls` | new | Masonry, precast or FC cladding. Default. |
| `envelope.windows-doors` | new | Apartment windows and balcony doors, retail shopfronts, lobby entry. Long lead. Default. |
| `envelope.external-waterproofing` | new | Balconies, podium, planters and the roof membrane to AS 4654.2 [K: knowledge/clusters/envelope/rules.yaml rule.as4654.external-membrane-falls-upstands]. Default. |
| `envelope.thermal-insulation`, `envelope.thermal-performance` | new | BASIX and NatHERS for the apartments; NCC Section J for the retail [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required, rule.ncc.section-j-envelope-fabric]. Default. |
| `envelope.facade-acoustics` | new | Glazing and facade to the acoustic report for street, retail and plant noise. Default. |
| `envelope.flashings-seals` **+** | new | Window, door and balcony threshold flashings; a top defect area. Not default. |
| `envelope.vehicle-doors` **+** | new | Car-park entry door or security gate, loading-bay roller door. Not default. |
| `envelope.roof-coverings` | out | Concrete roof; the membrane is `external-waterproofing`. |
| `envelope.curtain-wall`, `.external-shading`, `.building-sealing`, `.pliable-membranes`, `.bushfire-construction` | out | Not typical; curtain wall is the complex-facade case. |

### `interiors`

| System | Action | Why |
|---|---|---|
| `interiors.walls-linings`, `.ceilings`, `.wet-area-waterproofing`, `.tiling`, `.floor-finishes`, `.joinery`, `.internal-doors`, `.internal-glazing`, `.painting` | new | Apartments and common areas. Retail shells get none. Default. |
| `interiors.acoustic-separation` | new | Between apartments, and between apartments and the retail below (NCC Part F7 sound transmission applies to a floor separating a unit from a part of a different classification; verified: false) [K: knowledge/clusters/envelope/rules.yaml rule.ncc.sou-floor-airborne-sound]. Default. |
| `interiors.ffe` **+** | new | Apartment appliances (cooktop, oven, rangehood, dishwasher). Not default. |
| `interiors.room-acoustics` | out | |

### `fire-passive`

| System | Action | Why |
|---|---|---|
| `fire-passive.rated-elements` | new | Type A construction: Class 2, 6 and 7a at a rise of 4 or more all map to Type A [K: knowledge/clusters/fire/rules.yaml rule.ncc.type-of-construction]. Default. |
| `fire-passive.compartments` | new | Separation of classifications: retail from residential, car park from both (the rule says mixed classifications need C2D4 assessment first; verified: false). Default. |
| `fire-passive.penetrations` | new | Fire stopping. Default. |
| `fire-passive.openings` **+** | new | Fire doors and shutters between car park, retail, lobbies and fire-isolated stairs. Not default. |
| `fire-passive.external-walls` **+** | new | Non-combustible external walls for Type A, cladding product evidence [K: knowledge/clusters/fire/rules.yaml rule.ncc.external-wall-fire]. Not default. |
| `fire-passive.linings-fire-hazard`, `.structural-fire-protection` | out | Covered by concrete construction and normal specification. |

### `fire-active`

| System | Action | Why |
|---|---|---|
| `fire-active.sprinklers` **+** | new | Required for a Class 2 building with a rise of four storeys or more (NCC 2022 Part E1; verified: false), and then the retail and car park are sprinklered too. **Not in the default list**, so `fr.sprinklers-2` and the sprinkler hold point never fire. [K: knowledge/program/fragments.yaml fr.sprinklers-2] |
| `fire-active.hydrants` | new | Hydrants and booster to AS 2419.1 [K: knowledge/clusters/fire/rules.yaml rule.as2419.hydrant-design]. Default. |
| `fire-active.detection`, `.smoke-alarms`, `.emergency-lighting`, `.portable-extinguishers` | new | Default. Hose reels for the Class 6 part where required [K: knowledge/clusters/fire/rules.yaml rule.ncc.hose-reel-provision]. |
| `fire-active.occupant-warning` **+** | new | Building occupant warning in the sprinklered building (verified: false). Not default. |
| `fire-active.fire-water` | out (trigger) | Pumps and tanks only when Sydney Water's pressure and flow statement is inadequate. |
| `fire-active.smoke-control`, `.fire-control` | out (trigger) | Smoke control and stair pressurisation over 25 m effective height; a fire control centre only over 50 m (both verified: false). |
| `fire-active.special-suppression` | out | |

### `mechanical`

| System | Action | Why |
|---|---|---|
| `mechanical.air-conditioning` | new | Split systems to apartments, condensers on balconies or roof; retail AC is the tenant's. Default. |
| `mechanical.ventilation`, `.local-exhaust` | new | Apartment kitchen, bathroom and laundry exhaust; garbage-room exhaust; **retail kitchen-exhaust riser provision to roof** [K: knowledge/clusters/services-wet-air/rules.yaml rule.as1668-1.kitchen-exhaust-fire-rated-duct]. Default. |
| `mechanical.carpark-ventilation` | new | Mechanical car-park ventilation with CO control [K: knowledge/clusters/services-wet-air/rules.yaml rule.as1668.carpark-ventilation]. Default. |
| `mechanical.fire-mode-air-systems` | out (trigger) | Stair pressurisation over 25 m. |
| other mechanical children | out | Central plant, cooling towers, BMS beyond car-park controls: not typical. |

### `hydraulic`

| System | Action | Why |
|---|---|---|
| `hydraulic.cold-water`, `.hot-water`, `.sanitary`, `.roof-drainage` | new | Default. Domestic booster pumps are usual at six storeys. |
| `hydraulic.trade-waste` **+** | new | Grease-arrestor provision (space, drainage, vent, sampling point) for food tenancies; Sydney Water trade waste approval is the tenant's [K: knowledge/clusters/services-wet-air/rules.yaml rule.water-authority.trade-waste-agreement]. Not default. |
| `hydraulic.gas` **+** | new | Gas service and meters for the retail tenancies. Not default. |
| `hydraulic.backflow-prevention` **+** | new | Testable boundary and zone backflow for the fire and commercial services. Not default. |
| `hydraulic.non-potable-water` **+** | new | BASIX rainwater reuse to landscape and toilets where committed. Not default. |
| `hydraulic.specialist-process` | out | |

### `electrical`

| System | Action | Why |
|---|---|---|
| `electrical.utility-supply` | new | Substation (padmount or chamber) and consumer mains; DNSP long lead. Default. |
| `electrical.switchboards`, `.cabling-wiring`, `.lighting`, `.metering` | new | Main switchboard, apartment and retail meters (embedded network or DNSP metering), common-area lighting. Default. |
| `electrical.ev-charging` **+** | new | EV-ready capacity and distribution in the car park [K: knowledge/clusters/electrical-comms/rules.yaml rule.ncc.ev-charging-readiness, applies to 7a]. Not default. |
| `electrical.temporary-power` **+** | new | Builder's supply. Not default. |
| `electrical.standby-power`, `.solar-pv`, `.battery-storage`, `.earthing-lightning` | out | Solar PV is a BASIX option, not base. |

### `comms-security`

| System | Action | Why |
|---|---|---|
| `comms-security.carrier-lead-in`, `.structured-cabling`, `.access-control` | new | NBN, MATV and data, intercom and access control. Default. |
| `comms-security.cctv` **+** | new | Car-park and entry CCTV. Not default. |
| other children | out | |

### `vertical-transport`

| System | Action | Why |
|---|---|---|
| `vertical-transport.passenger-lifts`, `.lift-shafts-pits` | new | Two passenger lifts serving basement to roof, one stretcher-sized (verified: false). Default. |
| `vertical-transport.lift-fire-service` | out (trigger) | Emergency lift over 25 m (verified: false). |
| `vertical-transport.goods-lifts`, `.platform-lifts`, `.escalators-walks` | out | |

### `access-egress`

| System | Action | Why |
|---|---|---|
| `access-egress.exits`, `.stairs-and-ramps`, `.barriers`, `.egress-doors`, `.accessible-access`, `.livable-housing` | new | Two fire-isolated stairs, separate retail egress, balustrades, accessible path from street and car park to every apartment entry floor, ADG livable-housing share (verified: false). Default. |
| `access-egress.accessible-facilities` **+** | new | Accessible unisex toilet for the retail and common areas, accessible parking. Not default. |
| `access-egress.maintenance-access` **+** | new | Roof plant access and fall protection [K: knowledge/clusters/fire/rules.yaml rule.as1657.maintenance-access]. Not default. |
| `access-egress.pool-barriers` | out | |

---

## 3. Existing-building interfaces

The building is new, but the base site carries existing buildings and neighbours. [D]

- **Occupation.** Vacant possession of the existing shops at settlement or before demolition; any
  remaining lease needs notice before the program fixes demolition.
- **Services cut-overs.** Disconnection and abolishment of the existing electricity, gas, water and
  sewer services before demolition (DNSP, Jemena, Sydney Water); live services in the footpath and lane
  are located and protected.
- **Hazardous materials.** A hazardous-materials survey of the pre-1990 buildings (asbestos, lead paint,
  PCBs in old light fittings), licensed asbestos removal with SafeWork NSW notification and a clearance
  certificate before demolition (verified: false). The app has the pre-2004 implication only when
  `det.existing_building_year` is applied [K: knowledge/clusters/structure/consequences.yaml
  cq.pre-2004-fabric-hazardous-materials-survey].
- **Structure.** Demolition to AS 2601 [K: knowledge/clusters/structure/rules.yaml
  rule.as2601.demolition-stability]; party or boundary walls shared with nil-setback neighbours must be
  kept stable and weatherproofed; neighbours' footings may be undermined by the basement excavation.
- **Profile.** The base sets `det.existing_building: true` and `det.existing_building_year: 1965` (as T10 does), so the hazmat implication and the architect's demolition-plan row fire. The cost: the structural existing-assessment stage and the access `existing` row also switch on and are recorded as app differences.
- **Matching existing work.** Not applicable, apart from flashing and making good to neighbours' exposed
  walls.
- **Dilapidation.** Before demolition and excavation: neighbouring buildings and council assets
  (footpath, kerb, road, street trees), usually a DA condition; repeat at completion. Vibration and
  movement monitoring during excavation.

---

## 4. Consultants and why

Engaged by the developer unless noted. "Novated" means transferred to the D&C builder after award.

| Discipline | Id | Stage engaged | Need | Notes |
|---|---|---|---|---|
| Architect | `consultant.architect` | Feasibility | always, novated | Registered architect; ADG design verification statement with the DA; regulated designs for architectural building elements (R3). |
| Town planner | `consultant.town_planner` | Feasibility | always | SEE, ADG and LEP/DCP compliance, cl 4.6 where needed, panel. |
| Surveyor | `consultant.surveyor` | Feasibility | always | Detail and boundary survey, neighbours' levels and windows, strata plan, set-out by builder's surveyor. |
| Geotechnical engineer | `consultant.geotechnical` | Concept | always | Basement, rock, groundwater, retention parameters, inspections. |
| Structural engineer | `consultant.structural` | Concept | always, novated | Basement retention, transfer slab, PT slabs, regulated design. |
| Civil engineer | `consultant.civil` | Concept | always, novated | Stormwater, OSD, WSUD, driveway, s138 crossing. |
| Hydraulic engineer | `consultant.hydraulic` | Concept | always, novated or replaced by the D&C services subcontractor\'s designer (novation option; owner to confirm) | Water, sewer, gas, trade-waste provision, Section 73. |
| Fire services engineer | `consultant.fire_services` | Concept | always, novated or replaced by the D&C services subcontractor\'s designer (novation option; owner to confirm) | Sprinklers, hydrants, detection, pressure and flow enquiry. |
| Mechanical engineer | `consultant.mechanical` | Concept | always, novated or replaced by the D&C services subcontractor\'s designer (novation option; owner to confirm) | Car-park ventilation, exhaust risers, AC. |
| Electrical engineer | `consultant.electrical` | Concept | always, novated or replaced by the D&C services subcontractor\'s designer (novation option; owner to confirm) | Substation, metering, EV readiness, lighting. |
| BCA consultant | `consultant.bca` | Concept | always | NCC report for DA and CC, mixed-class fire strategy, performance solutions coordination. |
| Principal certifier | `consultant.certifier` | DD or CC | always | CC(s), principal certifier, critical-stage inspections, OC, fire safety certificate; appointed by the developer (owner), never novated (verified: false). |
| Access consultant | `consultant.access` | Concept | always | DA access report, Premises Standards and NCC Part D4, livable-housing share. |
| Acoustic consultant | `consultant.acoustic` | Concept | always | DA acoustic report (road noise, retail-to-residential, plant), NCC Part F7, pre-OC testing. |
| Traffic engineer | `consultant.traffic` | Concept | always | Parking, access, loading and waste vehicle swept paths, report for DA. |
| Waste consultant | `consultant.waste` | Concept | usually | Operational and C&D waste management plans to the council DCP; the architect sometimes does it. |
| Landscape architect | `consultant.landscape` | Concept | always, novated | ADG deep soil and communal open space, podium planters, street trees. |
| ESD / BASIX consultant | `consultant.esd` | Concept | always | BASIX and NatHERS for the apartments, Section J for the retail. `consultant.basix` (T02's id) is an accepted alias. |
| Quantity surveyor | `consultant.quantity_surveyor` | Feasibility | always | Feasibility estimates, CIV report for the DA fee, tender evaluation, financier reports. |
| Environmental consultant | `consultant.environmental` | Feasibility | usually | PSI for the change to residential use; escalates under `contaminated-land`. |
| Hazmat consultant | `consultant.hazmat` | Before demolition | always (base demolishes) | Hazardous-materials survey and clearance. |
| Fire engineer | `consultant.fire_engineer` | DD | usually | At least one performance solution is common (travel distance, car-park smoke, separation). |
| Vertical transport | `consultant.vertical_transport` | DD | optional | Often design-and-supply by the lift contractor at this height. |
| Waterproofing consultant | `consultant.waterproofing` | DD | optional | Independent review of podium and balcony details; common since the DBP Act. |
| Water Servicing Coordinator | none | After DA | always | Sydney Water accredited WSC makes the Section 73 application; often the civil or hydraulic firm. No discipline id. |
| Head contractor (D&C) | `trade.main_works` | After DA/DD | always | AS 4902 lump sum. |

The app suggests 15 of these by rule: Access, Architect, BCA, Certifier, Civil, Electrical, Fire
Services, Geotechnical, Hydraulic, Mechanical, Structural, Surveyor, Town Planner, Traffic and Vertical
Transport [K: testing/synthetic/reference/index/type-predictions.md T13]. **It never suggests Acoustic,
ESD/BASIX, Landscape, Waste, QS, Environmental, Hazmat or Fire Engineer** [K:
knowledge/works/package_defaults.yaml baselines, scope_additions, complexity_additions]. Under D&C the app
gives every consultant `novation: option`, including the certifier, planner, BCA, QS and traffic, which
are never novated [K: internal/procurement/proposed.go BuilderDesigns].

---

## 5. Approvals and authorities

**Pathway.** DA under the EP&A Act. Shop-top housing of three or more storeys with four or more
dwellings is assessed against SEPP (Housing) 2021 Chapter 4 and the Apartment Design Guide, with a
design verification statement from the registered architect and usually a council design review panel
(all verified: false). Determination by the local planning panel; over the regional threshold, by the
Sydney district planning panel (verified: false). The LEP's active street frontage clause requires
non-residential ground-floor uses on the mapped frontage (clause number varies by LEP; verified: false).
SEPP (Transport and Infrastructure) 2021 road-noise provisions for residential next to busy roads, and
SEPP (Resilience and Hazards) 2021 Chapter 4 for contamination (verified: false). Integrated
development where dewatering needs a WaterNSW approval (verified: false). Section 4.55 modifications
during design development are normal.

**Certificates.** Demolition is authorised by the DA consent itself and needs no CC (verified: false). Construction certificates are staged: early works (retention, excavation, basement), then the main works; principal certifier appointed by the
owner and notice of commencement; occupation certificate (interim residential plus retail shells, then
tenant fit-out OCs under each tenant's own approval); final fire safety certificate and the fire safety
schedule; Sydney Water Section 73 compliance certificate [K: knowledge/clusters/services-wet-air/rules.yaml
rule.nsw.section-73-certificate]; strata plan and strata certificate, then registration at NSW LRS
(verified: false). BASIX certificate with the DA and completion check at OC [K:
knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required].

**DBP Act and RAB Act (R3).** Class 2 parts: regulated designs prepared by registered design
practitioners and lodged with design compliance declarations before the related building work; the
builder is a registered building practitioner and makes a building compliance declaration before the
OC; variations need updated declarations. Under the Residential Apartment Buildings (Compliance and
Enforcement Powers) Act 2020 the developer notifies the expected completion date before applying for
the OC, and the Building Commissioner can audit and stop work (timing verified: false). The retail and
car-park parts are not Class 2, but the regulated building elements they share (fire safety systems,
structure, waterproofing over the car park) are part of the Class 2 building (verified: false).

**Home warranty and bond.** HBCF is not required for a building with a rise in storeys of more than 3 containing 2 or more dwellings (R4; verified: false); the developer lodges the strata building bond with NSW Fair Trading before the OC,
with interim and final defect inspections (Strata Schemes Management Act 2015 Part 11; amount and timing
verified: false).

**Authority applications.** Council s138 consents for the vehicle crossing, footpath works, hoarding,
work zone and crane swing; anchors under the road need council or TfNSW consent (verified: false);
Ausgrid or Endeavour connection and substation design; Sydney Water Feasibility, Section 73 and
building-plan approval; NBN developer registration; Jemena gas; FRNSW referral of fire safety performance
solutions (verified: false); long service levy before the CC.

**Inspections and hold points.** Critical-stage inspections for a Class 2-9 building differ from R7's
Class 1 list: for Class 2-4 the certifier inspects at commencement, before covering waterproofing in at least 10% of the rooms with wet areas, before covering stormwater connections, and the final (EP&A (Development Certification and Fire Safety)
Regulation 2021; verified: false). Plus: geotechnical sign-off of founding rock, reinforcement before each
pour, fire stopping before concealment, sprinkler pressure test before ceilings, waterproofing before
tiling, and pre-OC acoustic testing.

---

## 6. Cost plan shape

Lines only. The app's lines [K: testing/synthetic/reference/index/type-predictions.md T13]: F01–F15 fee
lines, construction 01–13 with access-egress children, P01–P07 starters including DA fees for `da`.

- **Fees and charges:** DA fees and contributions (s7.11 or s7.12; verified: false) `P04`;
  Housing and Productivity Contribution (verified: false) — no line; construction certificate and
  certifier fees `P05`; service authority fees `P06` (Sydney Water developer charges, Ausgrid
  substation, NBN, gas) — one generic line; long service levy — none; council s138, hoarding, work zone
  and damage deposit — none; strata building bond — none; strata plan registration and subdivision
  certificate fees — none.
- **Consultants:** one line per package. App lines exist for the 15 suggested. Missing: acoustic, ESD,
  landscape, waste, QS, environmental, hazmat, fire engineer.
- **Construction:** preliminaries and margin (none; spread pro rata if a budget is stated); demolition
  and hazardous-materials removal (in 01 only if `site.demolition` is scoped); 01 site works; 02
  substructure (excavation, retention, basement slab, waterproofing); 03 structure; 04 envelope; 05
  interiors; 06 passive fire; 07 active fire; 08 mechanical; 09 hydraulic; 10 electrical (incl.
  substation); 11 communications and security; 12 vertical transport; 13 access and egress (13.1
  accessible paths, 13.2 accessible facilities (missing), 13.3 barriers, 13.4 egress doors, 13.9 stairs);
  retail shell provisions and landlord's works (no line); public domain works (inside 01).
- **Contingency:** design contingency, construction contingency, escalation, exclusions.

The budget split the app uses is `commercial_hotel` ("Hotel or residential over retail"), low confidence,
25% preliminaries and **no fire-passive share** [K: knowledge/costs/budget_splits.yaml commercial_hotel].

---

## 7. Program shape

Skeleton `sk.new-dc-da`: investigation → concept design → approval documentation → approvals →
procurement → construction documentation → construction → completion → defects [K:
knowledge/program/skeletons.yaml sk.new-dc-da]. Class 2 process fragments fire because `2` is in
`det.ncc_class`; none exists for Class 6 or 7a [K: knowledge/program/fragments.yaml; index
program-fragments.md "Which NCC classes"].

Drivers and logic [D]:

1. Due diligence (planning certificate, survey, geotech, PSI) → concept → pre-DA and design review panel
   → design development to DA → DA reports → lodge → exhibition → assessment → panel determination.
2. After consent: D&C tender package → tender → award → novation → contractor's design development and
   regulated designs → staged CCs (early works for retention, excavation and basement, then main works). Demolition proceeds under the DA consent without a CC (verified: false).
3. Construction critical path: dilapidation → hazmat removal → demolition → piling and retention →
   excavation and anchors → footings and basement slab → basement structure and lift pits → transfer
   slab → floor cycle × levels → roof slab → facade, windows, balcony waterproofing behind the frame →
   services, linings, wet areas, joinery → lifts → commissioning → fire safety certificate → OC →
   strata registration and settlement.
4. Long-lead items: substation design and energisation (DNSP), lifts, windows and shopfronts, main
   switchboard, Section 73 works, NBN, sprinkler and booster valve sets.
5. RAB Act expected-completion notice before the OC application; strata bond lodged before OC.

---

## 8. RFP shape

Per consultant, what its RFP must carry. The `shape.yaml` `expect` blocks give stage-by-stage answers
in the RFP answer-key format and list where the app differs. The common thread: **every template's
default boundary ends at design or approval, so the novated post-contract stages, regulated design and
construction-phase services come out as options**, and most discipline flags (`hyd_*`, `fs_*`, `mech_*`,
`elec_*`, `bca_*`, `waste_*`, `access_*`, `cost_*`, `geotech_*`, `survey_*`, `landscape_*`) are client
decisions that cannot be keyed, so many rows are asked.

- **Architect** (`architecture`): concept to DA, planning response, DD, interiors of apartments and
  common areas, D&C tender documents, regulated design and declarations, post-novation documentation,
  ADG design verification statement, class and use map with shell versus fit-out deliverables
  (`mixed_use_map`), demolition plans. Not: tenant fit-out design, house-module rows, traditional tender
  or bidding, Green Star.
- **Structural** (`structural`): retention design, temporary-works interfaces, transfer slab, PT
  delegated design, regulated design, post-novation, construction-phase inspections. Not: residential
  framing, existing structure assessment, authority asset, super-flat floor.
- **Civil** (`civil`): stormwater concept for DA with OSD and WSUD (MUSIC), D&C tender design,
  post-novation, civil design declarations, as-built for OSD certification. Not: public road works,
  flood, NorBE, trunk drainage, shoring (structural's).
- **Town planner** (`town-planning`): pre-lodgement, SEE, ADG compliance, lodgement, assessment, panel
  attendance. Not: cl 4.6 (trigger), court.
- **BCA consultant** and **principal certifier** (`bca-certification`): two separate appointments.
  The BCA RFP carries A1–A3 advisory and the mixed-use classification map; the certifier RFP carries
  B1–B4, the statutory functions, staged occupation, fire referrals and the BASIX check.
- **Hydraulic** (`hydraulic-services`): authority stage (Sydney Water), design and D&C tender, trade-waste
  drainage and grease-arrestor provision, gas to retail, rainwater, pumps, declarations, post-novation.
  Not: onsite treatment, process waste, site stormwater (civil's).
- **Fire services**, **mechanical**, **electrical**: concept to tender, post-novation, construction and
  witnessing, regulated design declarations; mechanical adds car-park ventilation and the retail kitchen
  exhaust; electrical adds the DNSP application, substation, metering with retail shell energisation, EV.
- **Traffic**: parking and access advice, DA report, assessment response, car-park certification. Not:
  intersection modelling, travel plan, road works (triggers).
- **Waste**: operational and C&D plans, separate retail and residential streams (`shared`), consent
  review.
- **Access**: concept, DA and CC reports, shared routes for the mixed uses, housing and facilities.
- **Geotechnical**, **surveyor**, **landscape**, **QS**, **fire engineer**: as in the shape; the
  surveyor must carry the strata plan, the QS the mixed-use cost allocation.
- **No template**: acoustic, ESD/BASIX, hazmat, vertical transport, environmental. The app falls back to
  the generic `tpl.rfp-capex` [K: testing/synthetic/reference/index/rfp-templates.md "Disciplines … with
  no RFP template"].

---

## 9. Triggers

Each trigger is in `shape.yaml` with evidence, negation and what it adds. Summary:

| Trigger | How a document shows it (incl. indirectly) | Denial | Adds |
|---|---|---|---|
| `effective-height-over-25m` | "eight storeys, effective height 26.4 m"; "stair pressurisation" | "effective height under 25 m" | Fire engineer, stair pressurisation, emergency lift, EWIS |
| `heritage-conservation-area` | "within the Main Street Heritage Conservation Area"; "the facade of No. 42 is a contributory item" | "not a heritage item and not in a conservation area" | Heritage consultant, HIS, referral, facade retention |
| `classified-road-frontage` | "fronts Great Western Highway, a classified road"; "TfNSW concurrence" | "local road; no classified road frontage" | TfNSW referral, complex traffic, road-noise design |
| `cl46-variation` | "clause 4.6 request to vary the height standard"; "exceeds the 21 m height by 1.8 m" | "complies with height and FSR" | Variation request stage |
| `groundwater-dewatering` | "groundwater at 4.5 m"; "dewatering during excavation" | "no groundwater encountered" | Dewatering approval, tanked basement, monitoring |
| `contaminated-land` | "former service station"; "dry cleaner on the ground floor" | "PSI: low risk, no further investigation" | DSI, RAP, site auditor, validation |
| `flood-affected` | "flood planning level"; "overland flow along the lane" | "not flood affected" | Flood report, floor levels, flood barrier to basement |
| `rail-corridor` | "adjoins the rail corridor"; "Sydney Trains concurrence" | "no rail infrastructure nearby" | Rail referral, vibration, structural rail stage |
| `sydney-water-asset` | "sewer main crosses the rear of the site" | "no Sydney Water assets within the site" | Asset assessment, diversion or protection |
| `traditional-lump-sum` | "full documentation, select tender, lump sum AS 4000" | "D&C with novation" | Traditional skeleton, no novation |
| `supermarket-anchor` | "a 1,500 m² supermarket"; "B-double deliveries" | "small specialty shops only" | Loading dock, heavy vehicles, refrigeration plant acoustic |
| `fire-pumps-tanks` | "pressure and flow inadequate; fire pump room and tanks" | "town mains meet the hydrant demand" | Fire-water pumps and tanks |
| `rise-three-storeys` | "three-storey shop-top"; "two levels of apartments over a shop" | "six storeys" | HBCF; removes the strata bond and Class 2 sprinklers |
| `trees-affected` | "excavation within the TPZ of the street tree" | "no trees affected" | Arborist, tree protection |
| `acid-sulfate-soils` | "Class 3 on the Acid Sulfate Soils Map" | "not mapped" | ASS assessment and management plan |
| `underpinning-neighbour` | "No. 38 on shallow strip footings must be underpinned" | "neighbours founded on rock" | Underpinning, consent |
| `regionally-significant` | "CIV over $30 million"; "Sydney District Planning Panel" | "Local Planning Panel" | Regional panel (zero app delta) |
| `public-domain-works` | "council's public domain plan requires new footpath, kerb realignment and street trees" | "only a vehicle crossing" | Public road works, landscape public domain |

---

## 10. Keyword traps and negations specific to this type

- **"Shop-top housing"** is a planning land-use term, not an NCC class; the building is still 2/6/7a.
  "Residential flat building" is a different land use but the same Class 2.
- **"Retail"** in a shop-top is a shell. "Kitchen", "grease trap", "exhaust" in a retail provisions
  schedule do not mean a commercial kitchen is in the base works; `det.commercial_kitchen` should stay
  unset or false for the shell. The `kitchen` implied-by pattern adds joinery and exhaust from apartment
  kitchens anyway [K: knowledge/profile/scope_defaults.yaml implied_by kitchen].
- **"Basement"** fires `flag.basement` even in "no basement storage cages" [K:
  knowledge/profile/flag_readings.yaml basement]; read negations carefully.
- **"Lodgement"**: `flag.architect_lodges` matches any "lodge"; in this type the planner lodges.
- **"Panel"**: "fibre cement panels" do not match the panel regex, but "design review panel" is not a
  planning panel determination either.
- **"Stage 1"**: CC staging ("CC Stage 1 – early works") matches `flag.staged`, which means staged
  delivery of the building [K: knowledge/profile/flag_readings.yaml staged]. Staged CCs are not staged
  delivery.
- **"Strata"**: strata subdivision is base; `hdr.cond.stakeholder_complexity: strata` means an
  existing owners corporation and is wrong for a developer's new building.
- **"Demolish"**: matches `site.demolition` through implied_by but gives action `new`, so
  `fr.demolition-2` still does not fire [K: knowledge/profile/scope_defaults.yaml implied_by strip-out].
- **"Class 7a"**: also maps the project to the industrial category in package suggestions [K:
  testing/synthetic/reference/index/procurement.md "NCC class adds building categories"]. For this type it
  adds nothing new because `mixed` already brings `baselines[1]`.
- **"Shell"** is also "cold shell", "warm shell": both are shells for this map.

---

- **Phrases checked against the app's code-side triggers** (flag_readings, determinant triggers,
  implied_by), as base shop-top documents use them:

  | Phrase | Fires | Effect on T13 |
  |---|---|---|
  | "existing buildings", "existing shops" | nothing | `det.existing_building` needs the singular `\bexisting building\b`; the base documents must say "existing building" |
  | "four buildings", "two buildings" | `flag.multi_building` | Turns on the staged/multi-building module (masterplan stage, workshops). A shop-top is one building; say "one building" |
  | "Stage 1", "CC Stage 1", "staged construction certificates" | `flag.staged` | Staged CCs are not staged delivery; write "early works construction certificate" |
  | "water main", "sewer main" | `flag.authority_asset`, `det.water_supply_source` | A main in the street is not an asset in the site; say "the street water main" only where needed, or deny "no Sydney Water assets within the site" |
  | "traffic noise", "road noise" | `det.noise_affected_site` | Already true in the base; harmless |
  | "street trees" | `flag.arborist_offsite_trees` | Arborist RFP rows only; base keeps street trees unaffected, so pair with "not affected by the works" |
  | "road reserve", "works in the road" | `flag.public_road` | The base awning and crossing text must not use these; see `scenario_inputs` |
  | "MRV", "HRV" | `flag.heavy_vehicles` | Makes the traffic RFP complex; describe "council's standard collection vehicle" |
  | "dewater", "groundwater was not encountered" | `flag.groundwater` | The regex matches the negation too; Jev must answer "no"; base says "seepage only" |
  | "basement storage cages" | `flag.basement` | Base is yes anyway |
  | "Local Planning Panel" | `flag.panel` | Intended |
  | "design review panel" | nothing | Correct: not a determination |
  | "kitchen" (apartment or retail exhaust) | `implied_by kitchen` | Adds joinery and exhaust; retail "kitchen exhaust riser" is provision only |
  | "party wall" | `implied_by party-wall` | Adds adjoining-structures, compartments, acoustic separation; all already in scope |
  | "sewer junction" | `det.sewer_connection` | Harmless |
  | "retail tenancy", "loading bay", "loading dock", "grease arrestor", "shop-top housing" | nothing | So trade waste, loading docks and the retail part are never implied from words |
- **Rooftop communal room.** An enclosed rooftop room (not an open terrace) can become the topmost storey
  and raise the effective height, possibly over 25 m.

## 11. Gaps in the built-in knowledge

**How the taxonomy handles the mixed class.** [K: knowledge/profile/taxonomy.yaml mixed]

- `residential_retail` carries `ncc_class: 2/6`: no 7a, and the compound value matches no category
  mapping. `det.ncc_class` is never derived from the subclass, so a scenario must set `[2, 6, 7a]`.
- Scale fields are `total_storeys`, `total_gfa_sqm`, `podium_levels`, `tower_levels`,
  `residential_units`, `retail_gla_sqm`, `car_parks`. Budget splits read `hdr.scale.storeys`, and program
  per-unit durations read `plan.gross_floor_area` (assumed 250 m²) [K: knowledge/costs/budget_splits.yaml;
  knowledge/program/durations.yaml assumptions]. No field for basement levels, tenancy count or effective
  height in the scale.
- `quality_level` is `applies_to: residential` only.
- The project can hold parts, but package suggestions read `det.*` and `scope.*` from the whole part
  only [K: internal/store/packages.go packageOverview query]. There is no per-part class model for
  scope, cost or program.
- The mixed scope list is the apartments list: no sprinklers, trade waste, gas, loading dock, signage,
  accessible facilities, slab on ground, vehicle doors, occupant warning or openings.

**What the app will produce wrongly.**

0. Two hold points order activities wrongly: the reinforcement hold links every concrete-frame row before the ground slab (basement slab after the roof), and the sprinkler hold forces ceilings after sprinkler fit-off [K: internal/program/generate.go holdPointLinks]. The shape expects the correct order. One CC row only, no staged CCs.

1. No acoustic, ESD/BASIX, landscape, waste, QS, environmental, hazmat or fire-engineer package, so no
   fee lines or RFPs for them.
2. Sprinklers missing from scope: no sprinkler activities, no sprinkler hold point.
3. No Class 6 or 7a activities: no retail shell, grease-arrestor provision, tenancy handover, loading
   dock or car-park fit-out rows. Class 2 fragments labelled "(apartments)" stand for the whole
   building.
4. `fr.switchboard-2` and `fr.switchboard-major` both fire on the one switchboard item (6 matches
   `major`) [K: knowledge/program/fragments.yaml fr.switchboard-major].
5. Under D&C: `fr.construction-docs-2` keeps "Client sign-off for tender" after award; the CC sits in
   approvals before procurement; novation option is offered for the certifier, planner and others.
6. No rows or lines for the DBP Act declarations, RAB Act notice, strata bond, strata plan, s138 works,
   Section 73, long service levy, HPC or the fire safety certificate.
7. RFP boundaries end before post-novation, regulated design and construction services; the ADG design
   verification is an option; demolition drawings are absent because the existing-building module is off
   on a new build; hydraulic trade waste and gas are options.
8. Budget split is hotel-centred with no fire-passive share.

---

## 12. Sources and unverified claims

Knowledge files opened: `knowledge/profile/taxonomy.yaml`, `knowledge/profile/scope_defaults.yaml`,
`knowledge/profile/flag_readings.yaml`, `knowledge/profile/project_details.yaml`,
`knowledge/profile/project_facts.yaml`, `knowledge/determinants.yaml`,
`knowledge/works/package_defaults.yaml` (through the index), `knowledge/costs/budget_splits.yaml`
(through the index), `knowledge/program/skeletons.yaml`, `knowledge/program/fragments.yaml` (through the
index), `knowledge/rfp/modules.yaml`, `knowledge/rfp/skeleton.yaml`, the RFP templates through
`reference/index/rfp-templates.md`, `knowledge/clusters/fire/rules.yaml`,
`knowledge/clusters/envelope/rules.yaml`, `knowledge/clusters/electrical-comms/rules.yaml`,
`knowledge/clusters/services-wet-air/rules.yaml`, `knowledge/clusters/*/interfaces.yaml` (hold points),
`internal/rfp/drivers.go` (`deriveFlags`, `partFacts`), `internal/rfp/resolve.go` (`resolveStages`),
`internal/store/packages.go`.

Unverified (all `verified: false`): NCC 2022 Part E1 sprinkler trigger for Class 2 at four storeys;
C2D4 mixed classification; Part F7 sound between a unit and another classification; effective-height
25 m triggers (pressurisation, emergency lift, EWIS); SEPP (Housing) 2021 Chapter 4 and ADG
applicability, design verification and panel referral; SEPP (Transport and Infrastructure) and SEPP
(Resilience and Hazards) provisions; active street frontage clause numbers; s7.11/s7.12, HPC, long
service levy thresholds; Roads Act s138; DBP Act declaration timing and application to non-Class 2
parts; RAB Act expected-completion notice timing; strata building bond amount and inspection timing;
HBCF over-three-storey exemption (R4); the EP&A (Development Certification and Fire Safety) Regulation
2021 critical-stage list for Class 2–9; WaterNSW dewatering approval; certifier appointment by the owner;
AS 2890.1 for parking.
