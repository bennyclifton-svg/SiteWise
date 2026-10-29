# T05 House alterations and additions: domain map

Status: draft for owner review (`reviewed: false`). Default state NSW. The machine-readable key is
`shape.yaml` in this folder. It passes `testing/synthetic/tools/validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Rule numbers I could not confirm against the
instrument are marked `verified: false`.

---

## 1. The job

**What it is.** Work to an existing detached house (NCC Class 1a) that adds floor area and alters the
existing house at the same time. The most common Sydney version, and the **base scenario** here:

- a single-storey rear addition (new kitchen, living and dining opening to the garden);
- partial demolition of a rear skillion and laundry lean-to;
- removal of two internal loadbearing walls, and a wide new opening in the existing rear wall on steel beams;
- the existing bathroom renovated and the laundry relocated;
- a 1950s brick house of about 140 m², with about 55 m² added;
- a DA to council; not heritage, bushfire or flood affected;
- the owners move out for the main works. [D]

**Scale.** $0.3–1.5M construction (PROJECT_TYPES.md). Program and cost are driven less by the added
area than by how much of the existing house is touched. [D]

**Who procures it and how.** The homeowner is the client, sometimes with a project manager. The usual
pattern is traditional procurement:

- the architect (or a building designer) designs, documents and lodges the DA;
- the client appoints the structural engineer, surveyor, geotechnical engineer, BASIX assessor and
  hazardous materials surveyor directly, or the architect coordinates them;
- stormwater drainage is shown on the architect's site plan and designed and certified by the plumber;
  an engineer's stormwater plan is needed only under `stormwater-engineering-required` (OSD, a charged
  line, an easement or a fall to the rear);
- documentation goes to tender with 3–4 licensed builders;
- the builder signs a lump-sum domestic contract: HIA or MBA NSW, or occasionally ABIC (`fact.contract_form:
  domestic_hia`). [D]

The certifier is appointed by the owner. The plumber, electrician and air-conditioning installer design
their own systems under the builder, so there is normally no hydraulic, electrical, mechanical or civil
consultant. [D]

**What makes it go wrong** [D]:

- **Latent conditions in the existing house.** Undersized or shallow footings, rotten or termite-damaged
  subfloor and wall plates, out-of-level floors, sagging roof framing, and asbestos found in places the
  survey could not reach. This is the single biggest cost and time risk.
- **The junction between old and new.** Floor levels that do not meet, roof tie-ins, damp-proof course
  and flashing details, and cracking where new footings meet old ones.
- **Matching existing work.** Face bricks, roof tiles, timber floor boards, mouldings and window
  profiles, which may be discontinued or need salvage.
- **Services that are older than they look.** Old fuse boards, galvanised water pipe, earthenware
  sewer, and drainage that turns out to be under the addition.
- **Neighbours.** Objections during notification, overshadowing and privacy, cracking claims, and
  boundary disputes.
- **Brief creep once walls are opened.** Owners add rooms to the scope.
- **Owners' time and money pressure.** Rent while they are out of the house means delay costs them
  directly.

## 2. Systems and work items

Base-scenario actions. `shape.yaml work_items` holds the same list. Where one system is both new
(addition) and altered (existing house), I give the dominant action and say so.

**`site`**

- `survey-boundaries` investigate.
- `ground-conditions` investigate: an AS 2870 site classification, plus a test pit at an existing footing.
- `demolition` remove: the rear skillion and lean-to, internal walls, and strip-out of the bathroom and
  kitchen.
- `earthworks` new: minor levels.
- `erosion-sediment-control` new.
- `stormwater-detention` new: the added roof drains by gravity to the existing system or a kerb outlet,
  shown on the architect's site plan; on-site detention (OSD) and an engineer only under
  `stormwater-engineering-required`.
- `services-connections` alter: extend the sewer junction and the water service; cap services to the
  demolished wing.
- `landscape` alter: reinstatement.
- `adjoining-structures` retain: fences and neighbours' structures.
- Out of scope: `trees`, `flooding`, `contamination`, `acid-sulfate-soils` (all triggers);
  `retaining-walls` (trigger `sloping-site`, common on Sydney's sloping lots);
  `vehicle-access`, `pools`, `public-domain`, `loading-docks`, `signage`,
  `waste-storage`, `impact-protection` (not part of a rear addition).

**`substructure`**

- `footings` new, founded to match the existing house.
- `slab-on-ground` new.
- `damp-waterproofing` new: vapour barrier and DPC at the junction.
- `termite-management` new.
- `underpinning` only for a first-floor addition or excavation next to existing footings (trigger).
- `piles` only near a sewer (trigger).
- `basement-retention` is out of scope; building `retaining-walls` only under `sloping-site`.

**`structure`**

- `alterations` alter: openings, removed walls, beams.
- `timber` new: framing and deck.
- `steel` new: beams.
- `temporary-works` new: propping (designed by the builder's temporary works designer, not the structural
  engineer), and temporary weatherproofing where the existing envelope is opened.
- `masonry` only for matching face brick (fold into envelope otherwise).
- `concrete` (beyond the slab), `precast` and `design-basis` are not work items.

**`envelope`**

- `roof-coverings` new, tied in to the existing roof.
- `external-walls` new.
- `windows-doors` new.
- `flashings-seals` new: every old-to-new junction.
- `pliable-membranes` new.
- `thermal-insulation` new.
- `thermal-performance` new: BASIX commitments.
- `bushfire-construction` (trigger).
- `curtain-wall`, `facade-acoustics` and `external-waterproofing` are out of scope.
- `vehicle-doors` is out of scope (no garage in the base scenario).
- `external-shading` and `building-sealing` are optional.

**`interiors`**

- `walls-linings` alter: new to the addition, relined in the house.
- `ceilings` new.
- `wet-area-waterproofing` new.
- `tiling` new.
- `joinery` new: kitchen, laundry, robes.
- `internal-doors` new.
- `internal-glazing` new: shower screen.
- `floor-finishes` alter: sand the existing boards, new boards to the addition.
- `painting` alter.
- `acoustic-separation` only with a party wall.
- `room-acoustics` and `ffe` are out of scope (appliances may be owner supply).

**`fire-passive`**

- Out of scope for a detached house, except two triggers:
  - an external wall within 900 mm of a boundary (`rated-elements`);
  - a party wall.

**`fire-active`**

- `smoke-alarms` upgrade: new hard-wired alarms in the addition, interconnected with upgraded alarms in the
  existing house. [D]
- Everything else is Class 2–9 only.

**`mechanical`**

- `local-exhaust` new.
- `air-conditioning` new: installer-designed split or ducted system.
- `heating-appliances` only where an existing fireplace or flue is altered.
- Central plant, BMS, car-park and smoke-control systems are out of scope.

**`hydraulic`**

- `sanitary` alter.
- `cold-water` alter.
- `hot-water` replace: a BASIX-compliant unit.
- `roof-drainage` new.
- `gas` only if the gas service is altered.
- `non-potable-water` only where BASIX or the DCP calls for a rainwater tank (`verified: false` whether
  BASIX for alterations and additions requires one).
- `backflow-prevention`, `trade-waste` and `specialist-process` are out of scope.

**`electrical`**

- `switchboards` upgrade.
- `cabling-wiring` alter.
- `lighting` new.
- `temporary-power` new.
- `utility-supply` only for a mains upgrade or an overhead service relocation (trigger).
- `solar-pv`, `battery-storage` and `ev-charging` are optional client items.
- Standby power, earthing studies and hazardous-area work are out of scope.

**`comms-security`**

- Out of scope in the base scenario. At most, relocate the NBN lead-in if the addition is built over it,
  and add data points (builder or electrician).

**`vertical-transport`**

- Out of scope. A home lift is rare at this scale.

**`access-egress`**

- `stairs-and-ramps` new: steps to the garden.
- `barriers` only where a deck or floor is more than 1 m above the ground (first floor or sloping site).
- `livable-housing`: NSW did not adopt the NCC 2022 livable housing provisions (`verified: false`). [D]
- `pool-barriers` applies only if a pool exists on the property and the work affects the barrier.
- Accessible paths and accessible facilities are Class 2–9 only.

## 3. Existing-building interfaces

This is the part that makes T05 different from T01.

**Occupation.**
- Base: the owners vacate. Their rent and storage are a real project cost the app never models.
- Variant `occupied-during-works`: the addition is built and locked up as Stage 1, then the house is broken
  into and renovated as Stage 2. This needs:
  - a working kitchen and bathroom at all times, or temporary ones;
  - dust screens and protection to the occupied rooms;
  - separation between the site and the family;
  - longer preliminaries.
- Once the bathroom is out, many families leave anyway. [D]
- A vacant house under construction usually voids or conditions the owner's home and contents cover. The
  owner's policy and the builder's contract works insurance must be lined up before possession, and the
  vacant house needs security. This is an owner cost line. [D]
- In the app, `hdr.cond.operational_constraints` drives the RFP `live_operations` flag and the `live_site`
  module [K: testing/synthetic/reference/index/triggers.md §2]. Its wording is written for an institutional
  site [K: knowledge/rfp/modules.yaml live_site].

**Services cut-overs.** [D]
- The demolished wing is isolated and capped before demolition.
- The sewer junction may need extending or relocating, possibly with the old earthenware line replaced
  where it runs under the addition.
- Water: galvanised pipe is commonly replaced back to the meter.
- Electrical:
  - the switchboard is replaced with RCD protection once new circuits are added (AS/NZS 3000);
  - the existing wiring in altered rooms is checked;
  - the house is on builder's temporary power while the main board is out;
  - a consumer mains upgrade or overhead service relocation goes through the DNSP (Ausgrid, Endeavour or
    Essential Energy) and a Level 2 accredited service provider.
- Gas: hot water or cooktop changeover; the gas meter may need relocating by the gas network if it is
  on the rear wall.
- Stormwater: the added roof connects into an existing system of unknown condition, so CCTV or locating is
  needed. In older houses rear downpipes are often found discharging to the sewer; Sydney Water requires
  them disconnected once found.
- Existing rooftop solar PV is removed at the roof tie-in and reinstated by a CEC-accredited installer;
  the NBN lead-in is relocated if the addition is built over it (trigger `existing-solar-pv`).
- The app proposes "Keep the served system working during the works" through
  `ic.supplies-keep-served-working` [K: knowledge/works/interface_consequences.yaml], and an electrical
  isolation review through `cq.existing-electrical-shutdown-plan-review`
  [K: knowledge/clusters/electrical-comms/consequences.yaml]. Both reach Implications only.

**Hazardous materials.**
- The WHS duty to identify asbestos before demolition or refurbishment applies to a house built before
  2004, following the 31 December 2003 ban (`verified: false`); the app uses the same year
  (`cq.pre-2004-fabric-hazardous-materials-survey`). A house built before about 1990 is likely to
  contain:
  - bonded asbestos cement: eaves, wet-area linings, the "fibro" lean-to, Super Six roofing, vinyl floor
    tiles and backing, and electrical meter board backing;
  - lead paint in pre-1970 coats, and lead dust in the ceiling void, disturbed whenever ceilings come
    down at removed walls (older inner-ring houses especially);
  - possibly loose-fill insulation (in NSW the Mr Fluffy register is checked; `verified: false` for how far
    the register's coverage extends) [D].
- Base scope: a refurbishment and demolition hazardous materials survey before design is fixed. The app
  has this as an activity in `fr.existing-conditions` [K: knowledge/program/fragments.yaml
  fr.existing-conditions].
- Trigger scope: licensed removal with a SafeWork NSW notification, then a clearance certificate before
  demolition continues (WHS Regulation 2017 (NSW): licensing classes, notification period and
  10 m² threshold all `verified: false`). [D]
- Lead paint is managed per AS 4361.2 (`verified: false`). [D]
- The app's `cq.pre-2004-fabric-hazardous-materials-survey` needs an existing-building year before 2004
  **and** an alter/replace/upgrade/repair/remove item [K: knowledge/clusters/structure/consequences.yaml].

**Structure.** [D]
- The existing house's walls, footings and roof framing carry new loads at:
  - new openings;
  - removed walls (beams bearing on existing walls need new footings or pads);
  - roof tie-ins;
  - for a first floor, all of it.
- The engineer needs an existing-structure assessment, often after opening up.
- New footings are founded to match the existing house so the two parts do not move differently. A control
  joint or flexible junction is detailed where they cannot be matched (AS 2870 site classification,
  `rule.as2870.footing-design-by-site-class`, clause `verified: false`
  [K: knowledge/clusters/structure/rules.yaml]).
- Modifying existing framing members falls under `rule.as1684.member-modifications` (clause
  `verified: false`) [K: knowledge/clusters/structure/rules.yaml].
- Demolition stability: `rule.as2601.demolition-stability` [K: knowledge/clusters/structure/rules.yaml].
- The structural RFP adds the `existing_assessment` stage for extend [K: knowledge/rfp/structural.yaml].
- **Subfloor.** A 1950s brick house usually sits on a ventilated timber subfloor. A slab-on-ground
  addition along the rear wall can block the subfloor vents, break the termite inspection zone and leave a
  step in the floor levels. The junction needs relocated vents or cross-flow, a termite barrier with
  inspection access, and a detailed level change. [D]
- **Termites.** Active termites or damage found on opening up (bearers, plates, wall frames) add
  treatment and repair under a provisional sum (trigger `termite-damage-found`). [D]

**Matching existing work.** [D]
- Face brick (salvaged or a blended match), roof tiles (terracotta profiles are often discontinued), floor
  boards (species and width), skirting and architrave profiles, window style and sill heights, and ceiling
  heights.
- This drives long lead times, provisional sums and a "make good to existing finishes" line in every
  domestic alterations tender. No knowledge item covers it.

**Dilapidation of neighbours.** [D]
- Needed when there is excavation or footings near a neighbour's house, a party wall, or vibration
  (rock breaking).
- Pre- and post-works dilapidation surveys are often a DA condition.
- Access to a neighbour's land for construction is by agreement, or under the Access to Neighbouring Land
  Act 2000 (NSW) (`verified: false` for its current procedure).
- The structural RFP prices dilapidation as an option when `excavation_adjoining` or `basement` is set,
  and excludes it otherwise [K: knowledge/rfp/structural.yaml dilapidation].

**Approval history.**
- Earlier additions without approval (enclosed verandahs, garage conversions) must be regularised with a
  building information certificate, or demolished, before the new work relies on them [D].
- The app holds this as `uc.existing-building-or-addition-has-no-approval`, Implications only
  [K: knowledge/clusters/delivery/unforeseen.yaml].

## 4. Consultants and why

| Discipline | Stage engaged | Need | Notes |
|---|---|---|---|
| Architect or building designer (`consultant.architect`) | Concept | Always | Measures the existing house or relies on a measure-up; DA, CC and tender documents; often lodges. |
| Surveyor (`consultant.surveyor`) | Investigation | Always | Detail, level and boundary survey for the DA; neighbours' ridge and window levels; peg-out near the boundary. |
| Structural engineer (`consultant.structural`) | Concept to CC | Always | Openings, beams, footings, framing; existing-structure assessment; inspections. |
| Civil / stormwater engineer (`consultant.civil`) | DA | Trigger | Only under `stormwater-engineering-required` (OSD, charged line, easement, fall to the rear, council pipe, or a council checklist asking for an engineer's plan) or `flood-affected`. Otherwise the architect shows the drainage on the site plan and the plumber designs and certifies it to AS/NZS 3500.3 (`verified: false`). |
| Geotechnical engineer (`consultant.geotechnical`) | Investigation | Usually | Site classification for the new footings; some engineers accept an existing report. |
| Certifier (`consultant.certifier`) | After DA | Always | CC, principal certifier, critical-stage inspections, OC. |
| BASIX assessor (`consultant.basix`) | Approval documentation | Usually; the architect may generate it | The BASIX certificate is always needed above the cost threshold (`verified: false`, believed to be $50,000); for alterations the architect often generates it with the DIY thermal method. |
| Hazardous materials surveyor (`consultant.hazmat`) | Investigation | Always for a house built before 2004 (WHS, `verified: false`); asbestos is likely before about 1990 | Survey of the demolition and disturbance zone; clearance after removal. |
| Heritage consultant (`consultant.heritage`) | Concept | Trigger | Heritage impact statement in a conservation area or for an item. |
| Town planner (`consultant.town_planner`) | Concept | Trigger | Heritage, a clause 4.6 variation, objections, or a complex DA. Otherwise the architect writes the statement of environmental effects. |
| Arborist (`consultant.arborist`) | Concept | Trigger | Arboricultural impact assessment and tree protection plan. |
| Bushfire consultant (`consultant.bushfire`) | Concept | Trigger | BAL assessment and bushfire report. |
| Dilapidation surveyor (`consultant.building`) | Pre-construction | Trigger | No dilapidation discipline id exists; often a structural or building consultant. |
| Quantity surveyor (`consultant.quantity_surveyor`) | Design development | Optional | Pre-tender estimate; council needs a registered QS report only above a cost threshold this job rarely reaches (`verified: false`). |
| Interior designer, landscape architect | Design | Optional | A landscape plan only when the council requires one for the DA. |
| Hydraulic, electrical and mechanical engineers | — | Not needed | Licensed trades design these under the builder. |

For the default extend scope the app suggests Architect, Certifier, Civil, Electrical, Geotechnical,
Hydraulic, Structural, Surveyor and Town Planner [K: knowledge/works/package_defaults.yaml baselines[0],
complexity_additions[0]; testing/synthetic/reference/index/type-predictions.md T05]. This shape's scope
has `mechanical.local-exhaust` and `mechanical.air-conditioning`, so `scope_additions[2]` adds Mechanical
as a tenth [K: package_defaults.yaml scope_additions[2]]. Against the list above:

- **Civil, Electrical, Hydraulic, Mechanical and Town Planner are false positives for the base job.** The
  plumber, electrician and air-conditioning installer design their systems; the architect shows the
  drainage and writes the SEE.
- **BASIX and Hazmat are missed.** Arborist and Dilapidation are missed only under their triggers.

The head contract (`trade.main_works`) is proposed only once work items are accepted for planning
[K: internal/procurement/proposed.go ProposeHeadContract].

## 5. Approvals and authorities

**Pathway.** [D]

- A DA to council under the Environmental Planning and Assessment Act 1979 (NSW).
- The DA package:
  - statement of environmental effects;
  - survey;
  - existing, demolition and proposed plans;
  - shadow diagrams;
  - BASIX certificate;
  - stormwater drainage on the architect's site plan (an engineer's plan only under
    `stormwater-engineering-required`);
  - demolition waste management plan;
  - erosion and sediment control plan;
  - a cost estimate report;
  - notification plans.
- The council notifies adjoining owners under its community participation plan.
- CDC alternative (trigger `cdc-pathway`): complying development under the Housing Code in the Exempt and
  Complying Development Codes SEPP 2008, issued by a registered certifier.
  - It is not available for a heritage item or in a heritage conservation area. Under the Housing Code
    land exclusions, only detached outbuildings, some detached development and pools remain complying
    development in an HCA (clause `verified: false`; the repo's seed says the same,
    `data/reference/clerk/data/seed/ancillary-guide.md`).
  - It needs the setback and other standards met (code part `verified: false`).

**Certificates and steps.** [D]

- Before design and the DA:
  - s10.7 planning certificate (heritage, bushfire, flood, zone);
  - title and DP search;
  - Sydney Water asset search, plus Building Plan Approval through Sydney Water Tap in where the work is
    over or near a sewer;
  - BASIX certificate.
- After consent:
  - construction certificate;
  - long service levy paid before the CC (`verified: false` for the threshold);
  - home building compensation fund cover before the deposit or start, for residential work above the
    statutory value (Home Building Act 1989 (NSW), threshold `verified: false`);
  - principal certifier appointed and notice of commencement given (period `verified: false`);
  - council bonds and permits (footpath, skip bins, a kerb outlet or road opening under the Roads Act if
    the stormwater goes to the kerb).
- At completion:
  - occupation certificate for the new work;
  - plumbing compliance (Fair Trading notification and certificate of compliance);
  - electrical certificate of compliance;
  - waterproofing certificate;
  - BASIX completion evidence.

**Inspections and hold points** (critical-stage inspections by the principal certifier, list
`verified: false`) [D]:

- footing excavation before pour;
- slab reinforcement before pour;
- frame before linings or roof cladding;
- wet-area waterproofing before tiling (AS 3740, NCC Housing Provisions Part 10.2, `verified: false`);
- stormwater drainage connections before backfill;
- final.

Builder's own hold points: no break-through into the existing house before the addition is weathertight;
the asbestos clearance certificate before demolition continues.

The app produces `fr.da-lodgement`, `fr.da-exhibition`, `fr.da-assessment` and
`fr.construction-certificate`, and the frame hold point `if.frame-inspected-before-roof`
[K: knowledge/program/fragments.yaml; knowledge/clusters/structure/interfaces.yaml]. It has no
waterproofing or stormwater hold point, and no reinforcement hold point for a timber house (that hold
point needs `structure.concrete`).

**NCC.** NCC 2022 Volume Two and the ABCB Housing Provisions apply to the new work [D]:

- smoke alarms (H3D6 and Housing Provisions Part 9.5, `verified: false`), to AS 3786
  [K: knowledge/clusters/fire/rules.yaml rule.as3786.smoke-alarms, clause_verified false];
- fire separation within 900 mm of a boundary (Housing Provisions Part 9.2, `verified: false`);
- wet areas;
- stairs and barriers;
- energy efficiency through BASIX in NSW.

Existing parts are not upgraded unless altered, apart from smoke alarms. The app holds a
"do not assume a whole-building upgrade" review [K: knowledge/clusters/delivery/consequences.yaml
cq.nsw-alteration-approval-basis-review].

## 6. Cost plan shape

No amounts. Lines only; `shape.yaml cost_plan` has the full list with built-in status.

**Fees and charges**

- DA fees and contributions; the s7.12 levy applies where the council's plan imposes one [K:
  knowledge/costs/starter_lines.yaml da-fees].
- Construction certificate and certifier fees [K: cc-fees].
- Service authority connection fees [K: service-authority].
- Long service levy, home building compensation fund insurance, and council deposits and permits [D].
  None of these exist in the app.

**Consultants**

- Architect, Surveyor, Structural, Geotechnical, Certifier, BASIX, Hazardous materials survey. Civil
  only under `stormwater-engineering-required` or `flood-affected`.
- The app produces one fee line per adopted package [K: internal/costs/suggest.go], so it also produces
  Civil, Electrical, Hydraulic, Mechanical and Town Planner lines the base job does not need.
- The Certifier fee line and the "CC and certifier fees" starter line double count.

**Construction (elemental, house alterations convention)**

- Preliminaries and builder's margin.
- Demolition, strip-out and hazardous materials removal.
- Site works and external works.
- Substructure.
- Structure.
- Building envelope.
- Interiors and finishes.
- Active fire protection (smoke alarms).
- Mechanical services.
- Hydraulic services.
- Electrical services.
- Access and egress (steps).
- Make good to existing finishes.

The app's `residential_extension` budget split has no `structure` share, and its preliminaries share is
spread pro rata because there is no preliminaries line [K: knowledge/costs/budget_splits.yaml
residential_extension]. A stated budget therefore leaves Structure
at zero, although every alterations job has beams and framing.

**Contingency and other**

- Design contingency, construction contingency, escalation, exclusions [K: starter_lines.yaml].
- A provisional sum for latent conditions in the existing structure [D].
- Owner supply (appliances, tapware, light fittings) [K: internal/store/cost_budget.go applyBriefFigures].
- Temporary accommodation, removal and storage [D].
- Owner's insurance for a vacant house under construction [D].

Contingency on alterations is held higher than on a new house because of latent conditions [D]. No
percentage is given here.

## 7. Program shape

Skeleton `sk.extend-traditional`: investigation → concept design → approval documentation → approvals →
construction documentation → procurement → construction → completion → defects
[K: knowledge/program/skeletons.yaml sk.extend-traditional]. The same skeleton serves CDC and exempt.

**Activities that drive the job** [D]:

- **Investigation runs first and is longer than on a new house:**
  - measure the existing house;
  - hazardous materials survey;
  - trace services;
  - survey and geotechnical investigation;
  - council records search.
- **The DA is on the critical path**, including notification and any objections
  (`uc.extension-approval-slow-or-objected` [K: knowledge/clusters/delivery/unforeseen.yaml]).
- **Construction documentation** can overlap assessment at the client's risk.
- **The CC needs** structural design, the stormwater drainage on the architect's site plan (an engineer's
  plan only under `stormwater-engineering-required`), the levy paid and pre-CC conditions met (plus the
  Sydney Water approval where it applies).

**Construction logic:**

1. Owners vacate.
2. Site establishment, temporary power, sediment control.
3. Isolate services.
4. Asbestos removal and clearance (if found).
5. Demolition and strip-out.
6. Footings, termite management, slab.
7. Frame.
8. Frame inspection.
9. Roof tie-in and cover.
10. Windows and cladding: the addition is now weathertight.
11. **Break through into the existing house.** Propping goes in at this point, and the existing openings
    and beams are formed.
12. Services rough-in, including the switchboard upgrade.
13. Insulation.
14. Linings.
15. Wet-area waterproofing.
16. Waterproofing inspection.
17. Tiling.
18. Joinery.
19. Painting.
20. Floor finishes and make good, with sanding of existing boards last.
21. Landscape reinstatement.
22. PC.
23. OC.
24. Handover.
25. Owners return.

When the owners stay (variant), this becomes Stage 1 (addition to lock-up) then Stage 2 (existing
rooms).

**Long-lead items** [D]:

- windows and stacking doors (`du.windows-lead-1a` exists [K: knowledge/program/durations.yaml]);
- joinery (`du.joinery-lead-1a`);
- steel beams;
- matching bricks or tiles;
- DNSP mains upgrade;
- Sydney Water building plan approval;
- heat-pump water heater;
- skylights.

**What the app produces.**

- The project process fragments (concept, DA, CC, tender, award, PC, OC, defects) and
  `fr.existing-conditions` all fire.
- The new-build fragments fire for this scope's `new` and `replace` items: survey and geotechnical
  investigation, temporary power, erosion control, demolition (`remove` on extend), earthworks, footings,
  damp-proofing, termite, slab, stormwater, frame, roof, roof drainage, windows, cladding, insulation,
  ceilings, internal doors, wet areas, tiling, joinery, lighting, air-conditioning and hot water
  [K: knowledge/program/fragments.yaml].
- **Alter and upgrade items on interiors, hydraulic, electrical, landscape and `structure.alterations` get
  no construction row on `extend`.** The 1a fragments need `new` or `replace`; the `1a-reno` family and
  `fr.structure-alterations` need `refurb` [K: knowledge/program/fragments.yaml]. That covers:
  - `structure.alterations` alter (`fr.structure-alterations`);
  - linings, painting and floor finishes alter (`fr.linings-1a`, `fr.painting-1a`, `fr.floors-1a`);
  - sanitary and cold water alter (`fr.hydraulic-sanitary-1a`, `fr.hydraulic-cold-1a`);
  - cabling alter and the switchboard upgrade (`fr.electrical-1a`, `fr.switchboard-1a`);
  - landscape alter (`fr.landscape-1a`).
- **These still fire for alter or upgrade:** services connections (`fr.services-connections-1a`, no
  work-type gate), smoke alarms (`fr.smoke-alarms-1a` accepts upgrade), underpinning, utility supply,
  erosion control and tree protection.
- With the **default** extend scope (all `new`), linings, painting, floors and cabling do fire; the gap
  appears once a document or the user sets those items to `alter`.
- There is no break-through, staging, owners-vacate or waterproofing-hold-point activity.

## 8. RFP shape

| Consultant | Template | Must carry | Must not carry |
|---|---|---|---|
| Architect | `architecture` (house and existing_building modules) | Measured drawings or reliance on them; existing, demolition and proposed drawings and old-to-new interfaces (`existing_interfaces`); concept, schematic, DA with SEE, shadow diagrams and notification plan; planning response; design development; one tender and construction set (`house_documents`); kitchen, bathroom, laundry and joinery design and selections; BASIX commitments on the plans; bidding; construction support as an option | BIM model or coordination; interiors stage for beyond-house; separate tender and for-construction stages; regulated design; apartment design, facade certification, Green Star |
| Structural | `structural` | Existing-structure assessment; beams, posts and bearing pads at openings and removed walls; residential footing, framing, bracing and tie-down; footings matched to the existing; strengthening option; construction documents; inspections option | Temporary propping design (the builder's temporary works designer); shoring or piled retention; included shared Revit model; regulated design; D&C or post-novation stages |
| Surveyor | `land-surveying` | Detail and level survey including the existing house's floor, ridge and eave levels; boundary identification; neighbours' windows and ridges; sewer and drainage locations; peg-out option | Subdivision or strata; monitoring without excavation; occupied-site access when vacant |
| Certifier | `bca-certification` | CC (B2), principal certifier inspections (B3), OC (B4), BASIX check; limited existing-house assessment (smoke alarms, affected part) | BCA advisory A1–A3 as the default; fire referrals; included performance solutions; staged occupation when vacant |
| Geotechnical | `geotechnical` | Investigation plan; boreholes or augers plus a test pit at an existing footing; AS 2870 classification and footing advice; report | Contamination; groundwater monitoring; acid sulfate unless mapped; pavement parameters |
| Civil / stormwater (trigger `stormwater-engineering-required` only) | `civil` | Stormwater concept for the DA, discharge point, connection to the existing system or kerb; CC stormwater design; investigation of retained drainage; OSD only if the DCP requires it | Roads and pavements element; public road works; MUSIC, NorBE, trunk drainage; 3D model and design criteria document |
| BASIX | none (generic report) | BASIX certificate for alterations and additions; thermal assessment of the addition; commitments schedule; revision at CC | Section J; Green Star; rating of unaltered parts |
| Hazmat | none (generic report) | Pre-demolition and refurbishment survey of every affected area including roof space, subfloor and eaves; NATA-analysed sampling including ceiling dust where ceilings come down; register for the tender; clearance option | Doing the removal; soil contamination |

**Expected state.** Each templated RFP in `shape.yaml` carries an `expect` block in the template's own
stage and position keys, and an `app_differs` list. The main differences:
- client-decision flags leave survey, geotechnical and certifier rows at "ask";
- the certifier boundary falls back to the A stages;
- `bidding` is priced as an option after the house boundary;
- the measure-up, the SEE and the test pit at an existing footing are not named in any template.

The base `profile` sets every flag those blocks read (all `no` except `architect_lodges`) so the rows are
deterministic.

**Built-in status.**

- The architecture and structural expectations match the repo's own answer key
  [K: data/eval/rfp/answer-keys/small-class1-extension.yaml]. That key is a CDC case, so the DA stages
  (`planning_response`) are added here.
- The architecture template blocks issue without a budget [K: knowledge/rfp/architecture.yaml
  required_facts].
- The certifier boundary defaults to the advisory A stages unless the client sets `bca_certification`
  [K: knowledge/rfp/bca-certification.yaml boundary].
- The civil template has no house module, so roads and pavements stay in [K: knowledge/rfp/civil.yaml
  elements].
- The surveyor's detail, boundary and services stages each wait on a client flag [K:
  knowledge/rfp/land-surveying.yaml].

## 9. Triggers

Full evidence, negation and consequences are in `shape.yaml triggers`. Summary:

| Trigger | How documents show it (incl. indirectly) | How a document denies it | What it adds |
|---|---|---|---|
| `occupied-during-works` (variant) | "owners remain in occupation; works staged"; "kids' bedrooms must stay usable" | "owners will vacate for the works" | Stage 1 / Stage 2 program, temporary kitchen and bathroom, dust screens, longer prelims; architect staging; certifier occupation scope |
| `hazardous-materials-found` (variant) | "bonded AC sheet to eaves"; "the old fibro lean-to out the back"; "Super Six roof"; "lead paint on window frames"; "lead dust in the ceiling void" | "no ACM detected"; "samples 1-6: no asbestos; the fibre-cement eaves date from the 1996 re-roof" | Hazmat clearance, licensed removal, SafeWork NSW notice, removal line, lead paint management |
| `heritage-conservation-area` (variant) | "within the X HCA, LEP Schedule 5"; "contributory building"; "Federation streetscape" | "not a heritage item and not in a HCA" (s10.7); "Dulux Heritage colour range" is not heritage | Heritage consultant, HIS, town planner, `fr.heritage-referral`, complex architecture/structural scale; CDC unavailable for an item or in an HCA (`verified: false`) |
| `first-floor-addition` (variant) | "new first floor over the existing house"; "go up rather than out" | "single-storey addition only" | Underpinning, stair, barriers, roof removal and temporary roof, structural capacity check, geotech test pits; owners usually vacate |
| `building-over-sewer` (variant) | "Sydney Water main across the rear yard"; "zone of influence of the sewer"; "build over" | "no sewer within the property" | Sydney Water building plan approval, piering, CCTV, structural authority-asset stage, architect water-asset position |
| `neighbour-dilapidation` | "excavation within 1 m of No. 14"; DA condition requiring dilapidation | "neighbours more than 6 m away, no excavation" | Dilapidation surveyor, pre/post surveys, structural dilapidation scope |
| `semi-detached-party-wall` | "pair of semi-detached cottages sharing a party wall"; "terrace" | "free-standing detached house" | Fire and acoustic separation, party wall consent and dilapidation, passive fire line |
| `wall-near-boundary` | "laundry wall 450 mm off the boundary"; "nil setback" | "minimum side setback 1.5 m" | Fire-resisting wall and eaves details, passive fire line, boundary peg-out |
| `bushfire-prone` | "mapped bushfire prone land"; "BAL-29"; "back fence adjoins bushland" | "not bushfire prone land" | Bushfire consultant, AS 3959 construction, `fr.bushfire-1a`, PBP assessment |
| `significant-tree` (variant) | "jacaranda within 3 m of the addition"; "neighbour's gum roots in the footing zone" | "no trees within 5 m" | Arborist AIA and TPP, tree protection, council tree permit |
| `cdc-pathway` (variant) | "complying development under the Housing Code"; "CDC to a registered certifier" | "a DA is required as it doesn't meet the Housing Code"; "the site is in a heritage conservation area, so complying development is not available" | CDC instead of DA, CC and notification; removes DA fees; certifier and planner RFPs block in the app |
| `electrical-supply-upgrade` | "overhead service crosses the addition roof"; "three-phase supply" | "existing supply has capacity" | DNSP application, Level 2 ASP works, `fr.utility-1a`, authority fees |
| `flood-affected` | "flood control lot; floor 500 mm above 1% AEP"; "overland flow across the rear yard" | "not flood affected"; "flood test the shower recess" is not flood | Flood report, floor levels, `fr.flood-1a`, civil flood scope |
| `stormwater-engineering-required` (variant) | "OSD required above the DCP threshold"; "site falls to the rear; inter-allotment easement over No. 9"; "charged line to the kerb"; "council's pipe and easement through the rear yard"; "stormwater plan by a qualified engineer" | "roof water connects to the existing gravity system to the kerb; OSD not required" | Civil/stormwater engineer, Civil fee line, civil RFP, council stormwater approval and easement |
| `sloping-site` (variant) | "rear yard falls 1.8 m to the back boundary; suspended floor with retaining to the cut"; "deck 2.1 m above ground" | "level site; slab at natural ground" | Retaining walls, piers, barriers over 1 m; geotech slope and retaining parameters; structural retaining design |
| `termite-damage-found` | "timber pest inspection found active termites in the subfloor bearers"; "termite damage to the wall plates found when linings came off" | "no evidence of termite activity or damage" | Treatment, repair of bearers, joists and plates, latent-conditions provisional sum |
| `loose-fill-asbestos` | "listed on the NSW loose-fill asbestos insulation register"; "loose-fill asbestos insulation found in the ceiling space" | "register search returned no listing" | Register process before any ordinary alterations (`verified: false`); the job cannot proceed as planned |
| `existing-solar-pv` | "existing solar panels on the rear roof plane removed for the tie-in and reinstated"; "NBN lead-in and gas meter on the rear wall relocated" | "no solar panels; meters on the front elevation" | Remove and reinstate PV by a CEC-accredited installer, relocate lead-in and meter |
| `unapproved-existing-work` | "sunroom enclosed without approval"; "no record of approval for the garage conversion" | "council holds the approval and OC for the 1994 addition" | Building information certificate, rectify or demolish |

Three triggers also take items out of the base (`removes` in `shape.yaml`):
- `cdc-pathway` removes the DA fees line, the CC and certifier fees line, and the DA, notification,
  assessment and CC activities;
- `occupied-during-works` removes the owners-vacate and owners-return activities, the temporary
  accommodation line and the vacant-house insurance line;
- `sloping-site` removes the slab and its activity, because the addition is on a suspended floor.

The app's five false-positive consultants and their fee lines are listed under `not_expected_procurement`
and `not_expected_cost_plan`. The facts a scenario's documents must state, with the exact phrasing the
triggers need, are under `scenario_inputs`.

## 10. Keyword traps and negations specific to this type

**"Existing house" is not "existing building".**
- `det.existing_building` triggers only on `\bexisting building\b` [K: knowledge/determinants.yaml
  existing_building]. "Existing dwelling", "existing residence", "the cottage" and "the house" never
  trigger it.
- The RFP derives the flag from work type, so RFPs are fine. The existing-building consequences and
  unforeseen records stay silent.

**Construction year.**
- "Built in 1958" and "year built" trigger; "c. 1950s", "Federation", "inter-war" and "post-war fibro"
  do not [K: determinants.yaml existing_building_year].

**Implied scope ignores negation.**
- `implied_by` is a regex over read passages [K: knowledge/profile/scope_defaults.yaml implied_by;
  testing/synthetic/reference/index/scope-defaults.md]. Each of these adds scope despite saying the
  opposite:
  - "No works to the existing bathroom" adds wet-area scope.
  - "The kitchen is retained" adds joinery and sanitary scope.
  - "No demolition of the front façade" adds `site.demolition`.
  - "There is no party wall" adds fire compartments and acoustic separation.

**"Flood".**
- `\bflood` matches "flood test the shower recess" (AS 3740 practice), so the flood determinant question
  is asked. Jev must answer not stated.

**"Build over".**
- The `authority_asset` regex `build(?:ing)? over` matches "build over the existing slab" or "build over
  the old garage footprint" [K: knowledge/profile/flag_readings.yaml authority_asset].

**"Railway".**
- An address on "Railway Parade" or "Railway Street" (common in Sydney) matches the `rail` flag and makes
  structural scale complex [K: flag_readings.yaml rail; knowledge/rfp/structural.yaml scale].

**"Lodge".**
- `architect_lodges` matches "Lodge Street" or "the existing lodge".

**"Stage".**
- "Stage 1 demolition" in a builder's program sets `staged`.

**"Heritage".**
- "Dulux Heritage range", "Heritage Red", and "heritage-style mouldings to match" trigger the heritage
  question. Only a planning-certificate or LEP statement is evidence. "Not a heritage item and not within
  a HCA" is the enumerated `none`.

**Extend versus refurb.**
- Documents call this job a "renovation", "reno", "alterations" or "extension". If Jev reads `refurb`
  from one document and `extend` from another, the work type conflicts. The skeleton then falls to
  `sk.generic`, and the house/extend scope list is lost (refurb has none).

**Extension of time.**
- "Extension of time" and "contract extension" are contract terms, not building work.
- "Additional", "in addition" and "additions to the contract sum" are not additions.

**Asbestos.**
- "Fibro" and "Super Six" mean asbestos cement in older houses.
- "Hardiflex", or a fibre cement sheet laid after the late 1980s, is not necessarily asbestos.
- No regex reads any of these.

**Class 10a.**
- An existing garage or shed mentioned as "Class 10a" adds `10a` to `det.ncc_class` even when it is out of
  scope.

**Granny flat.**
- An attached "granny flat" or "secondary dwelling" is T12, not T05.
- `uc.secondary-dwelling-fire-separation` exists for the extend case [K: knowledge/clusters/delivery/unforeseen.yaml].

**Cross-contamination.**
- "Cross-contamination of potable water" (backflow) matches `contaminat`.

## 11. Gaps in the built-in knowledge

Also in `shape.yaml gaps` with evidence. In order of effect on this type:

1. **Much of the renovated half of the job has no program.** Alter and upgrade items on interiors
   (linings, painting, floors), hydraulic (sanitary, cold water), electrical (cabling, switchboard),
   landscape and `structure.alterations` match neither the 1a fragments (new or replace) nor the 1a-reno
   and structure-alterations fragments (refurb only). Expect no rows for structural alterations, the
   switchboard upgrade, altered plumbing and electrical work, relining, painting or floor refinishing.
   Services connections, smoke alarms, underpinning, utility supply, erosion control and tree protection
   still fire for alter or upgrade. With the default scope (all `new`) the gap appears only once a
   document or the user sets those items to `alter`.
2. **Default scope is thin and all `new`.** The 16 house/extend systems have no site, demolition,
   wet-area, sanitary, water, joinery, exhaust, slab, termite or temporary-works items, and
   `structure.alterations` defaults to `new`. Expect no Site works line, no demolition activity, and no
   existing-building Implications, until documents or the user add scope.
3. **Consultant list.** Expect five wrong fee lines (Civil, Electrical, Hydraulic, Mechanical, Town
   Planner) and two missing ones (BASIX, Hazmat). Arborist and Dilapidation are missed only under their
   triggers.
4. **RFPs.** No template for BASIX, Hazmat, Heritage, Bushfire or a building consultant. The certifier
   defaults to BCA advisory. Certifier and town planner RFPs block on CDC. The civil RFP keeps roads and
   pavements. Retained-services investigation in the hydraulic and electrical templates waits on client
   flags.
5. **Cost plan.** The extension split gives Structure 0 %. There is no demolition, preliminaries,
   make-good, latent-conditions, long service levy, HBCF, s7.12, Sydney Water or temporary accommodation
   line. The certifier is counted twice.
6. **Occupied homes.** No staging, temporary facilities or break-through logic. Protection hangs on a
   vehicle-impact system. The live-site module is worded for institutions.
7. **Hold points.** No waterproofing or stormwater hold point fires. There are no fragments for flashings,
   exhaust, adjoining structures, retaining walls or access items, and a retained tree gets no
   tree-protection row (`fr.tree-protection` needs new, alter or repair).
8. **Profile triggers.** The existing-building and year triggers are narrow (§10). BASIX applicability is
   pending: `det.basix_required` and `det.energy_compliance_pathway` have no `profile_group`, so the API
   cannot set them; `shape.yaml` lists them under `expected_derived`.
9. **Ids missing.** No trade id for a licensed asbestos removalist or a demolition contractor, and no
   dilapidation-surveyor discipline.
10. **Inconsistency.** The lead paint threshold is 1970 in the determinant notes and 1990 in
    `uc.lead-paint-on-old-windows`. No `rules.yaml` holds an asbestos or lead rule.

## 12. Sources and unverified claims

**Opened and cited.**

- `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and `index/*.md` (type-predictions T05, procurement,
  cost-plan, program-fragments, program-skeletons, rfp-templates, triggers, unforeseen, scope-defaults,
  systems).
- `knowledge/works/package_defaults.yaml`
- `knowledge/profile/scope_defaults.yaml` (`classes[2]`, `implied_by`)
- `knowledge/determinants.yaml` (`existing_building`, `existing_building_year`, `heritage_status`)
- `knowledge/profile/flag_readings.yaml`
- `knowledge/profile/taxonomy.yaml`
- `knowledge/costs/starter_lines.yaml`
- `knowledge/costs/budget_splits.yaml`
- `knowledge/program/skeletons.yaml`
- `knowledge/clusters/structure/interfaces.yaml`
- `knowledge/clusters/structure/rules.yaml`
- `knowledge/clusters/fire/rules.yaml`
- `internal/knowledge/relevance.go`, `internal/knowledge/works_eval.go` (work-type evaluation)
- `data/intake/disciplines.json`
- `data/eval/rfp/answer-keys/small-class1-extension.yaml`

**Not verified against the instrument** (all `verified: false`):

- BASIX cost threshold for alterations and additions, and whether it requires a rainwater tank.
- NCC 2022 H3D6 / Housing Provisions Part 9.5 (smoke alarms), Part 9.2 (fire separation near boundary),
  Part 10.2 (wet areas).
- AS 3740 edition referenced.
- Housing Code part numbers, and heritage and conservation-area exclusions.
- Long service levy threshold.
- Home Building Act contract and HBCF thresholds.
- Notice of commencement period.
- Critical-stage inspection list.
- WHS Regulation asbestos licensing classes, notification period and 10 m² threshold.
- AS 4361.2 for lead paint.
- Mr Fluffy register coverage.
- Access to Neighbouring Land Act procedure.
- QS cost-report threshold.
- NSW non-adoption of the livable housing provisions.

**Assumptions.** Domain practice tagged `[D]` reflects Sydney residential practice; regional NSW differs
on councils, the DNSP (Essential Energy) and water authorities (local council water and sewer instead of
Sydney Water).

**Multi-part profiles.** A profile with two parts, an `extend` addition and a `refurb` existing house,
might pick up the 1a-reno fragments, because work types are joined per part
(`workTypeValue`, `internal/knowledge/works_eval.go`). This is untested and outside this key.

## Disputed review findings

None. All 15 findings in `REVIEW.md` were applied. Brief 01's variant count was taken as guidance, so there are 9 variants.
