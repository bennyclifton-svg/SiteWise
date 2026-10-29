# T01 New detached house: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 28 October 2026 from the
knowledge on `main` (8a6a6c8 plus the working tree) and professional practice.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. Instrument clause numbers I could not
check against the instrument itself are written `verified: false`. No prices, rates or week durations
appear; the only durations named are the app's own `du.*` ids.

---

## 1. The job

**What it is.** A new, architect-designed, two-storey detached dwelling (NCC Class 1a) with an attached
double garage (Class 10a) on a vacant, serviced, suburban lot in Greater Sydney. The base scenario is:

| Item | Base value |
|---|---|
| Lot | Vacant Torrens-title lot of about 550–750 m² in an R2 Low Density Residential zone, reticulated water and sewer in the street, kerb and gutter, footpath, overhead or pit-and-pillar power, NBN available |
| Building | 4 bedrooms, 3 bathrooms, powder room, laundry, open kitchen/living, about 280–320 m² GFA over two storeys, double garage under the main roof |
| Construction | Waffle or stiffened raft slab on ground (AS 2870), timber wall frames and prefabricated roof trusses (AS 1684.2), brick veneer ground floor with lightweight cladding to the upper floor, metal sheet roof with sarking, aluminium windows, internal timber stair with balustrade |
| Services | Ducted reverse-cycle air-conditioning, heat-pump hot water, rainwater tank plumbed to toilets, laundry and garden taps (BASIX water), interconnected smoke alarms, NBN lead-in, all-electric (gas is a trigger) |
| Pathway | Development application (DA) to council, then a construction certificate (CC) from a registered certifier |
| Procurement | Traditional: the client appoints the design team directly, full documentation, select tender to three or four residential builders, lump-sum domestic contract (HIA or MBA NSW form) |
| Value | Inside the PROJECT_TYPES range of $0.5–3M; the shape names no amount |

**Scale.** One building, one owner, one contractor. Consultant team of five to eight firms, most of them
small practices. The builder subcontracts every trade.

**Who procures it and how.** The owner (a private individual or couple, sometimes through a family
trust) engages the architect first, usually on a percentage or staged lump-sum fee. The architect or a PM
then brings in the surveyor, geotechnical engineer, structural/civil engineer, BASIX/NatHERS assessor and,
where council requires one, a landscape designer. The owner appoints the registered certifier. A builder
is chosen by select tender on the construction documents and signs a Home Building Act compliant
lump-sum contract. A project-home builder doing design and construct is the main alternative (trigger
`builder-designs`). [D]

**What makes it go wrong.** [D]

1. **Design over budget at tender.** No cost check between concept and tender is the commonest failure.
   Owners add scope (pool, joinery, glazing) during design development.
2. **Site information too late.** Designing before the detail survey and the geotechnical site
   classification leads to redesigned footings, levels and stormwater. A sewer main in the lot or a fall to
   the rear found after DA lodgement forces redesign.
3. **Stormwater.** A lot that falls away from the street cannot drain to the kerb by gravity. It needs an
   interallotment easement over a downhill neighbour or a charged/pump-out system, and council will not
   determine the DA without the discharge resolved.
4. **DA delays.** Neighbour objections (privacy, overshadowing, height), council requests for information,
   and missing DCP items such as the landscape plan, shadow diagrams, waste management plan or BASIX
   certificate.
5. **CC conditions not met.** Long service levy unpaid, Sydney Water building plan approval missing, DA
   conditions requiring further documents (structural certification, stormwater design, dilapidation).
6. **Builder risk.** The domestic market has thin-margin builders. Insolvency, unlicensed subcontracting,
   missing Home Building Compensation Fund (HBCF) cover, deposits above the statutory cap, and variations
   from loose provisional sums and PC items.
7. **Defects.** Wet-area waterproofing, window and door flashings, balcony and deck waterproofing, and
   cracking on reactive clay.
8. **Long-lead items.** Windows and external doors, joinery, frames and trusses, and the electricity
   connection by the distributor (DNSP) are the usual items that hold up lock-up and handover. [K:
   knowledge/program/fragments.yaml fr.windows-1a, fr.joinery-1a, fr.utility-1a]

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. Actions use `knowledge/works/actions.yaml`. The app's
default house/new list (41 systems, action `new`) is `[K: knowledge/profile/scope_defaults.yaml
classes[0]]`. Items marked **+** are ones a competent PM expects that the default list lacks.

### `site`: Site, ground and external works

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | new | Set-out of the building, and identification surveys at slab and frame stage that council or the certifier usually require. The pre-design detail and boundary survey is a consultant task (§4). In default list. |
| `site.ground-conditions` **+** | investigate | Geotechnical investigation for AS 2870 site classification. Not in the default list, so `fr.investigate-geotech` never fires. [K: knowledge/program/fragments.yaml fr.investigate-geotech] |
| `site.earthworks` | new | Cut and fill to the slab platform, spoil disposal, site grading away from the building. In default list. |
| `site.erosion-sediment-control` | new | Sediment fence, stabilised entry, stockpile controls before any disturbance; a standard DA condition. In default list. |
| `site.stormwater-detention` | new | Site stormwater from downpipes, rainwater tank overflow, pits and the kerb outlet. OSD only where council requires it (trigger). In default list. |
| `site.services-connections` | new | Water service, sewer junction connection, electricity service, NBN lead-in conduits from the boundary. In default list. |
| `site.vehicle-access` | new | Driveway, paths, garage apron. In default list (no program fragment). |
| `site.public-domain` **+** | new | Council vehicular crossing (layback), footpath reinstatement and kerb works in the road reserve under a Roads Act s138 consent (verified: false). Not in default list. [D] |
| `site.landscape` | new | Turf, planting, paving, fencing, letterbox, clothes line; often required by the DA landscape plan. In default list. |
| `site.retaining-walls` | new (trigger) | Only on a sloping lot. Out of base. |
| `site.trees` | retain (trigger) | Only when trees on or next to the lot are affected. Out of base. |
| `site.pools` | new (trigger) | Pool. Out of base. |
| `site.demolition` | out | Vacant lot. Becomes `remove` only under trigger `old-structure-on-lot`. |
| `site.waste-storage` | out | Bin storage is a small area on the drawings, not a work item for a house. |
| `site.contamination`, `site.acid-sulfate-soils`, `site.flooding`, `site.adjoining-structures` | out | Conditions, not base scope (see triggers and §9). |
| `site.impact-protection`, `site.loading-docks`, `site.signage` | out | Commercial and industrial systems. |

### `substructure`

| System | Action | Why |
|---|---|---|
| `substructure.footings` | new | Edge and internal beams, pad footings, piers where the geotechnical report needs them. In default list. |
| `substructure.slab-on-ground` | new | Waffle or stiffened raft slab designed to AS 2870 for the site class. In default list. [K: knowledge/clusters/structure/rules.yaml rule.as2870.footing-design-by-site-class] |
| `substructure.termite-management` | new | AS 3660.1 physical or chemical barrier, durable notice in the meter box. In default list. [K: knowledge/clusters/structure/rules.yaml rule.ncc.termite-management] |
| `substructure.damp-waterproofing` | out | The vapour barrier under the slab is part of the slab; a separate item only with a basement or below-ground habitable rooms (trigger `basement-or-boundary-excavation`). |
| `substructure.piles` | new (trigger) | Bored piers to rock or below a sewer zone of influence. Out of base. |
| `substructure.basement-retention`, `substructure.retaining-walls` | new (trigger) | Basement garage or cut along a boundary. Out of base. |
| `substructure.underpinning` | out | No existing building. |

### `structure`

| System | Action | Why |
|---|---|---|
| `structure.timber` | new | Wall frames, roof trusses, upper floor joists or cassettes, bracing and tie-down (AS 1684.2), usually supplier-designed and engineer-reviewed. In default list. [K: knowledge/clusters/structure/rules.yaml rule.as1684.timber-framing] |
| `structure.masonry` **+** | new | Brick veneer to the ground floor. Not in the default list; it arrives only when a read document says "brickwork" or "masonry" (`implied_by` masonry). The fragment `fr.masonry-1a` exists. [K: knowledge/profile/scope_defaults.yaml implied_by masonry; knowledge/program/fragments.yaml fr.masonry-1a] |
| `structure.steel` **+** | new | Steel or LVL beams over the garage door and the open-plan living opening, lintels, possibly a steel portal. Not in default list. [D] |
| `structure.concrete` | out | Only the slab, which is `substructure.slab-on-ground`. A suspended concrete floor or terrace is a variant. |
| `structure.design-basis` | out as a work item | Wind class (AS 4055) and site class feed the structural design; they are consultant inputs, not works. |
| `structure.precast`, `structure.alterations`, `structure.temporary-works` | out | Not used on a lightweight house. Temporary works are the builder's. |

### `envelope`

| System | Action | Why |
|---|---|---|
| `envelope.external-walls` | new | Brick veneer and lightweight cladding with wall wrap. In default list. |
| `envelope.roof-coverings` | new | Metal sheet roof with sarking and anticon blanket. In default list. |
| `envelope.windows-doors` | new | Aluminium windows and doors to AS 2047, glazing to AS 1288, BASIX/NatHERS glazing specification. Long lead. In default list. |
| `envelope.vehicle-doors` | new | Garage sectional door, motorised. In default list. |
| `envelope.flashings-seals` | new | Sill and head flashings, damp-proof course, cavity flashings. In default list. |
| `envelope.pliable-membranes` | new | Sarking and wall wrap, condensation management. In default list. |
| `envelope.thermal-insulation` | new | Ceiling, wall and (where required) floor insulation to the NatHERS specification. In default list. |
| `envelope.thermal-performance` | new | BASIX commitments: thermal, energy and water. In default list. [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required] |
| `envelope.external-waterproofing` | out | Only with a balcony or deck over a habitable room (variant, not a trigger here). |
| `envelope.bushfire-construction` | new (trigger) | Bushfire-prone land. Out of base. |
| `envelope.external-shading` | out | Eaves are part of the roof; louvres or pergolas are a variant. |
| `envelope.facade-acoustics` | out | Only on a noise-affected site. |
| `envelope.building-sealing` | out | Covered by the BASIX and NatHERS commitments in NSW. |
| `envelope.curtain-wall` | out | Not used on a house. |

### `interiors`

| System | Action | Why |
|---|---|---|
| `interiors.walls-linings` | new | Plasterboard, villaboard in wet areas. In default list. |
| `interiors.ceilings` | new | Plasterboard ceilings, cornices, bulkheads for ducted air. In default list. |
| `interiors.wet-area-waterproofing` | new | Bathrooms, ensuite, laundry, powder room to AS 3740; NSW inspection before tiling. In default list. [K: knowledge/clusters/envelope/rules.yaml rule.state.waterproofing-inspection-before-tiling] |
| `interiors.tiling` | new | Wet areas, kitchen splashback, possibly living areas. In default list. |
| `interiors.floor-finishes` | new | Timber or hybrid flooring, carpet to bedrooms. In default list. |
| `interiors.joinery` | new | Kitchen, butler's pantry, vanities, laundry, robes, linen. Long lead. In default list. |
| `interiors.internal-doors` | new | Doors, frames, hardware. In default list. |
| `interiors.internal-glazing` | new | Shower screens, mirrors. In default list. |
| `interiors.painting` | new | Internal painting. In default list. |
| `interiors.ffe` **+** | new | Kitchen appliances and fixed fittings, usually PC items in the contract; loose furniture excluded. Not in default list. [D] |
| `interiors.acoustic-separation`, `interiors.room-acoustics` | out | Detached dwelling; no separating walls. |

### `fire-passive`

Out of scope. A detached Class 1a house has no fire compartments or rated elements, and the attached
Class 10a garage needs no separation from it. [D] The one common exception is an external wall within
900 mm of a boundary or of another building on the lot (for example a garage built to the side
boundary), which needs fire-resisting construction under the ABCB Housing Provisions Part 9.2, called up by NCC 2022 Volume Two (verified: false).
That is a design variant, not a trigger here.

### `fire-active`

| System | Action | Why |
|---|---|---|
| `fire-active.smoke-alarms` | new | Hard-wired, interconnected smoke alarms to AS 3786 on each storey and between bedrooms and the rest of the dwelling (ABCB Housing Provisions Part 9.5, verified: false). In default list. |

All other fire-active systems (detection, sprinklers, hydrants, emergency lighting, extinguishers): out.
A Class 1a house has no fire safety schedule and no Fire and Rescue NSW involvement. [D]

### `mechanical`

| System | Action | Why |
|---|---|---|
| `mechanical.air-conditioning` | new | Ducted reverse-cycle system, design-and-install by the air-conditioning subcontractor. In default list. |
| `mechanical.local-exhaust` | new | Bathroom, ensuite, laundry and rangehood exhaust ducted outside. In default list. |
| `mechanical.refrigerant-condensate` | out as an item | Part of the AC installation. |
| `mechanical.heating-appliances` | out | Fireplace or wood heater is a variant. |
| Others (ventilation, central plant, BMS, car-park ventilation, cooling towers, smoke modes, process) | out | Commercial systems. |

### `hydraulic`

| System | Action | Why |
|---|---|---|
| `hydraulic.cold-water` | new | Water service from the Sydney Water meter, internal reticulation. In default list. |
| `hydraulic.hot-water` | new | Heat-pump hot water unit (BASIX energy). In default list. |
| `hydraulic.sanitary` | new | Under-slab drainage, above-ground plumbing, connection to the sewer junction. In default list. |
| `hydraulic.roof-drainage` | new | Gutters, downpipes, box gutters if any. In default list. |
| `hydraulic.non-potable-water` **+** | new | Rainwater tank, pump and mains top-up plumbed to toilets, laundry and garden taps to meet the BASIX water target. Not in default list and has no program fragment. [D] |
| `hydraulic.gas` | new (trigger) | Only if natural gas is connected. |
| `hydraulic.backflow-prevention` | out as an item | Covered by the tank top-up device and the plumber's work. |
| `hydraulic.trade-waste`, `.specialist-process` | out | Not used on a house. |

### `electrical`

| System | Action | Why |
|---|---|---|
| `electrical.utility-supply` | new | New connection from the DNSP network (Ausgrid, Endeavour Energy or Essential Energy), consumer mains, meter. Long lead. In default list. |
| `electrical.temporary-power` **+** | new | Builder's temporary supply from the pole or pillar. Not in default list, so `fr.temporary-power` does not fire. [K: knowledge/program/fragments.yaml fr.temporary-power] |
| `electrical.switchboards` | new | Main switchboard and RCD protection. In default list. |
| `electrical.cabling-wiring` | new | Final subcircuits, earthing. In default list. |
| `electrical.lighting` | new | Downlights, external lighting. In default list. |
| `electrical.solar-pv` | out of base | Often chosen to meet the BASIX energy target, but it is a design choice, not a requirement, so it is not in the base key. [D] |
| `electrical.ev-charging`, `.battery-storage` | out | Owner options. |
| `electrical.metering`, `.earthing-lightning` | out as items | Metering is the retailer's; earthing is part of wiring. |
| `electrical.standby-power`, `.process-hazardous` | out | Not used on a house. |

### `comms-security`

| System | Action | Why |
|---|---|---|
| `comms-security.carrier-lead-in` | new | NBN pit and pipe and lead-in conduit; NBN new-development application. In default list. |
| `comms-security.structured-cabling` | new | Data points and TV. In default list. |
| `comms-security.access-control`, `.intruder-alarm`, `.cctv` | out | Owner options (video intercom, alarm). |
| Others | out | Commercial systems. |

### `vertical-transport`

Out of scope. A home lift (`vertical-transport.platform-lifts`) is an uncommon owner option.

### `access-egress`

| System | Action | Why |
|---|---|---|
| `access-egress.stairs-and-ramps` **+** | new | Internal stair to ABCB Housing Provisions Part 11.2 (verified: false). Not in default list; arrives only when a read document says "stair" (`implied_by` stair). [K: knowledge/profile/scope_defaults.yaml implied_by stair] |
| `access-egress.barriers` **+** | new | Stair and void balustrades, balcony balustrade, and fall protection to upper-floor bedroom windows (ABCB Housing Provisions Part 11.3, verified: false). Not in default list. [D] |
| `access-egress.livable-housing` | out | NSW did not adopt the NCC 2022 livable housing provisions for Class 1a when other states did (verified: false). [D] |
| `access-egress.pool-barriers` | new (trigger) | Only with a pool. |
| `access-egress.accessible-access`, `.accessible-facilities`, `.exits`, `.egress-doors`, `.maintenance-access` | out | No NCC accessibility or egress provisions for a Class 1a house beyond the stair and barriers. |

---

## 3. Existing-building interfaces

Not applicable: new build on a clear (vacant) lot. No occupation, no service cut-overs inside a building,
no hazardous materials survey, no existing structure, no matching existing work.

Three neighbouring interfaces still apply and must not be confused with existing-building work: [D]

- **Council assets.** The kerb, gutter, footpath and street trees in front of the lot. Council takes a
  damage security deposit and expects a photographic or written record of their condition before work
  starts.
- **Adjoining properties.** Dividing fences (Dividing Fences Act 1991), overlooking and overshadowing
  during DA notification, and dilapidation surveys only when excavation is near a boundary (trigger).
- **Street services.** Connection to the existing water main, sewer junction, DNSP network and NBN in the
  street. These are new connections, not cut-overs.

A leftover shed or slab on the "vacant" lot is the trigger `old-structure-on-lot`, not a base condition.

---

## 4. Consultants and why

App suggestion for this profile: Architect, Certifier, Civil, Electrical, Geotechnical, Hydraulic,
Mechanical, Structural, Surveyor, Town Planner. [K: knowledge/works/package_defaults.yaml baselines[0],
scope_additions[2], scope_additions[3], scope_additions[4], complexity_additions[0]]

| Discipline id | Engaged at | Need | Why | App |
|---|---|---|---|---|
| `consultant.architect` | Brief / concept | Always | Design lead, DA drawings, construction documents, tender | yes, `baselines[0]` |
| `consultant.surveyor` | Before concept | Always | Detail and boundary survey with AHD levels, neighbours' windows and ridge heights; later set-out and identification surveys at slab and frame stage | yes, `baselines[0]` |
| `consultant.geotechnical` | Before structural design (concept) | Always | AS 2870 site classification and footing advice | yes, `baselines[0]` |
| `consultant.structural` | Design development, before CC | Always | Slab and footings, beams, bracing review, retaining walls; certificates for the CC and inspections | yes, `baselines[0]` |
| `consultant.civil` | Approval documentation (DA) | Always | Stormwater concept plan for the DA, detailed stormwater design, driveway levels, sediment control. Often the same firm as the structural engineer. | yes, `baselines[0]` |
| `consultant.basix` | Approval documentation (DA) | Always (NSW) | NatHERS thermal assessment and BASIX certificate lodged with the DA (or CDC), revised for the CC | **no** rule names it |
| `consultant.certifier` | Before CC (often appointed at DA) | Always | CC assessment, principal certifier, critical stage inspections, occupation certificate | yes, `baselines[0]` |
| `consultant.landscape` | Approval documentation (DA) | Usually (council DCP) | Landscape plan many Sydney councils require for a new dwelling DA; sometimes drawn by the architect | **no** rule names it |
| `consultant.town_planner` | Concept / pre-DA | Optional | Statement of environmental effects and DA management; essential only with a variation request, objections, heritage or a complex DCP. The architect often writes the SEE for a simple house. | yes, `baselines[0]`, `complexity_additions[0]` |
| `consultant.quantity_surveyor` | Concept and pre-tender | Optional | Cost plan at concept and pre-tender; lenders may also want a QS for construction-loan drawdowns | **no** rule names it |
| `consultant.arborist` | Investigation | Conditional (trigger) | Tree impact assessment and protection | **no** rule names it |
| `consultant.bushfire` | Investigation | Conditional (trigger) | BAL assessment | yes when `det.bal` is set, `complexity_additions[5]` |
| `consultant.heritage` | Concept | Conditional (trigger) | Heritage impact statement in a conservation area | yes when `det.heritage_status` is set, `complexity_additions[1]` |
| `consultant.hazmat` | Investigation | Conditional (trigger) | Only when an old structure is on the lot | **no** |
| `trade.main_works` | Procurement | Always | Head contract with a licensed builder | partial: proposed only from gaps on accepted items |

**Not expected for this job, but the app suggests them** [D]:

- `consultant.electrical`, `consultant.hydraulic` and `consultant.mechanical`. A house is designed
  and installed by the electrician, plumber and air-conditioning subcontractor under the builder. The
  architect's drawings show light, power and plumbing points. A services engineer is unusual below the
  top of the value range, and even there a lighting designer is more common than an electrical
  engineer. The app adds them through `baselines[0]` (Electrical, Hydraulic) and
  `scope_additions[2]–[4]` (any `mechanical.*`, `hydraulic.*` or `electrical.*` in scope). [K:
  knowledge/works/package_defaults.yaml]

Also optional and not keyed: an independent building inspector for the owner at slab, frame, lock-up and
pre-handover. [D]

---

## 5. Approvals and authorities

Clause numbers below are from memory of the instruments, so they are all `verified: false`. [D]

### Planning pathway (base: DA)

- **Development application** to council under the EP&A Act 1979, Part 4 (evaluation s4.15, verified:
  false), lodged through the NSW Planning Portal. It includes the architectural set, survey, stormwater
  concept plan, landscape plan, BASIX certificate, statement of environmental effects, shadow diagrams,
  waste management plan, cost estimate, and the owner's consent.
- **Notification** to neighbours under council's community participation plan (typically a short
  neighbour-notification period for a dwelling, not a public exhibition; verified: false). Objections
  can send the DA to the local planning panel.
- **Determination** with conditions: contributions (s7.11 or s7.12 levy under council's plan, verified:
  false), damage deposit, Sydney Water building plan approval before the CC, tree protection and
  survey certificates.
- **Alternative: complying development** under the Housing Code (State Environmental Planning Policy
  (Exempt and Complying Development Codes) 2008, Part 3) where the lot and design comply. The
  certifier determines it within a statutory period (trigger `cdc-pathway`). The app's
  `du.cdc-determination-period` already holds this as a statutory row. [K:
  knowledge/program/fragments.yaml fr.cdc-application]

### Certificates

| Certificate | By | When |
|---|---|---|
| BASIX certificate | NatHERS/BASIX assessor via the BASIX portal | With the DA; revised if the CC design changes (SEPP (Sustainable Buildings) 2022, verified: false) [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required, clause_verified false] |
| Construction certificate | Registered certifier (or council) | After DA conditions for the CC are met and construction documents are complete; before work starts |
| Principal certifier appointment and notice of commencement | Owner appoints; notice to council at least two days before work (EP&A Act s6.6, verified: false) | Before site establishment |
| Occupation certificate | Principal certifier | After the final inspection, with the BASIX completion requirements met, before occupation (EP&A Act s6.9, verified: false) |
| Home Building Act documents | Builder | HBCF insurance certificate before any deposit or work for a contract over the threshold; licensed builder; compliant written contract with the deposit cap (Home Building Act 1989 Part 6, s7, s8, verified: false) |
| Electrical certificate of compliance (CCEW) | Licensed electrician | At connection and completion |
| Plumbing certificate of compliance | Licensed plumber, to NSW Fair Trading as plumbing regulator (verified: false) | At completion |
| Termite management notice, waterproofing certificate, glazing certificate, truss and frame certificates, structural and stormwater engineer's certificates, works-as-executed drainage plan | Subcontractors and engineers | Collected by the builder for the occupation certificate |

### Authority applications [D]

- **Council:** DA (or the CDC goes to the certifier), Roads Act 1993 s138 consent for the vehicular
  crossing and footpath works (verified: false), road or footpath occupancy for cranes and concrete
  pumps, damage security deposit, contributions.
- **Long Service Corporation:** long service levy paid before the CC for work above the threshold
  (Building and Construction Industry Long Service Payments Act 1986, verified: false).
- **Sydney Water:** Tap in building plan approval (checks build-over or next to assets), water service
  connection application, sewer junction inspection. A Section 73 compliance certificate is **usually not
  needed** for one new house on an existing serviced lot; it is needed when the consent conditions it or
  for a dual occupancy or subdivision. The app's knowledge says it is needed "before the occupation
  certificate for most new developments". [K: knowledge/clusters/services-wet-air/rules.yaml
  rule.nsw.section-73-certificate, clause_verified false]
- **DNSP:** connection application for the new service; Level 2 Accredited Service Provider for the
  service mains and metering if underground; a Level 1 ASP only if a network extension is needed.
- **NBN Co:** new-development application for pit-and-pipe and the lead-in.
- **SafeWork NSW:** no notification for a standard house build. WHS duties of the principal contractor
  apply for a construction project above the threshold (Work Health and Safety Regulation 2017 (NSW),
  verified: false).
- **Fire and Rescue NSW:** none for Class 1a.

### Inspections and hold points

NSW mandatory critical-stage inspections for Class 1 and 10 by the principal certifier (Environmental
Planning and Assessment (Development Certification and Fire Safety) Regulation 2021, verified: false): [D]

1. After excavation for footings, before the footings are poured.
2. Before pouring any in-situ reinforced concrete element (the slab).
3. Before covering the framework of floors, walls and roof.
4. Before covering waterproofing in wet areas. [K: knowledge/clusters/envelope/rules.yaml
   rule.state.waterproofing-inspection-before-tiling]
5. Before covering stormwater drainage connections.
6. After the building work is completed, before the occupation certificate.
7. With a pool: after the barrier is erected and before the pool is filled (trigger `swimming-pool`).

Engineering hold points the structural engineer usually adds: pier and footing base inspection,
reinforcement inspection before the slab pour, and frame or tie-down inspection where specified. The
surveyor provides identification surveys at slab (setbacks) and at frame or roof stage (height). [D]

The app's own hold points that can fire here: `if.frame-inspected-before-roof` (frame before roof
cladding) fires. `if.reinforcement-inspected-before-pour` needs a `structure.concrete` item, which a house
does not have, so it does not fire. `if.underground-services-before-paving` needs a program row on
`site.vehicle-access`, which has no fragment, so it does not fire. [K:
knowledge/clusters/structure/interfaces.yaml]

---

## 6. Cost plan shape

Lines only, never amounts. Sections follow the app: `fees_and_charges` (app `authority`),
`consultants`, `construction`, `contingency`. [K: knowledge/costs/starter_lines.yaml;
knowledge/costs/elements.yaml]

**Fees and charges**

| Line | App |
|---|---|
| DA fees and contributions (DA fee, notification fee, s7.11/s7.12 contributions) | yes, P04 `da-fees` when planning is `da` |
| Construction certificate and certifier fees (CC, principal certifier, inspections, OC) | yes, P05 `cc-fees`. See the double-count note below. |
| Service authority connection fees (Sydney Water, DNSP, NBN) | yes, P06 `service-authority`, as one generic line |
| Long service levy | no |
| Council road-reserve fees: s138 vehicular crossing, footpath occupancy, damage security deposit | no |
| Home Building Compensation Fund insurance | no line. It is the builder's premium inside the contract sum; list it so it is not missed when a builder excludes it. [D] |

**Consultants** (one fee line per package)

Architect; Structural engineer; Civil (stormwater) engineer; Geotechnical engineer; Surveyor (detail,
boundary, set-out, identification surveys); BASIX/NatHERS assessor; Landscape designer; Certifier;
Town Planner (optional); Quantity Surveyor (optional).

The app writes F-lines for its own suggestions: Architect, Certifier, Civil, Electrical, Geotechnical,
Hydraulic, Mechanical, Structural, Surveyor, Town Planner. [K: testing/synthetic/reference/index/type-predictions.md
T01] It is missing BASIX and Landscape, and it adds three services engineers that a house does not need.

**Double count.** The app makes both a Certifier fee line (F02, from the package) and the P05
"Construction certificate and certifier fees" line. A PM keeps the certifier's professional fee in one
place and only statutory lodgement fees in the other. [K: knowledge/costs/starter_lines.yaml cc-fees;
knowledge/works/package_defaults.yaml baselines[0]]

**Construction** (by element; one head contract)

| Element | Holds | App |
|---|---|---|
| Preliminaries and builder's margin | Site establishment, supervision, insurances, HBCF premium, temporary services, scaffold, waste, margin | **no** line; a stated budget spreads preliminaries pro rata [K: knowledge/costs/budget_splits.yaml residential_house_new] |
| 01 Site works and external works | Earthworks, sediment control, stormwater, services connections, driveway and crossing, landscaping, fencing | yes |
| 02 Substructure | Slab, footings, termite management | yes |
| 03 Structure | Frames, trusses, beams, brick veneer | yes |
| 04 Building envelope | Cladding, roof, windows, garage door, insulation, BASIX items | yes |
| 05 Interiors and finishes | Linings, waterproofing, tiling, joinery, floors, doors, painting, appliances | yes |
| 08 Mechanical services | Ducted AC, exhaust | yes |
| 09 Hydraulic services | Plumbing, drainage, hot water, rainwater tank | yes |
| 10 Electrical services | Power, lighting, switchboard, solar PV if chosen | yes |
| 11 Communications and security | NBN lead-in, data | yes |
| 13.x Stairs, ramps, handrails and nosings | Internal stair | **partial**: appears only when `implied_by stair` adds `access-egress.stairs-and-ramps` from a read document [K: knowledge/costs/elements.yaml code 13] |
| 13.x Compliant balustrades and barriers | Stair, void, balcony and window barriers | **no**: `access-egress.barriers` never enters house scope |
| Provisional sums and PC items | Allowances in the contract for undecided selections | no line [D] |

The app also makes **07 Active fire protection** from the smoke-alarm item. A QS would carry smoke alarms
in electrical. It is harmless but should not count as a separate element in the answer key. `shape.yaml not_expected_cost_plan` lists it with the Electrical, Hydraulic and Mechanical fee lines the app adopts on prepare, so a scorer reports them as known extras. [K:
knowledge/costs/elements.yaml code 07]

**Contingency** (app section): Design contingency; Construction contingency; Escalation; Exclusions
(all yes). Owner-supplied items (loose furniture, window furnishings, some appliances) come in as
"Owner supply: …" lines only when `proj.owner_supply_stated` is read. [K: knowledge/costs/starter_lines.yaml]

---

## 7. Program shape

**Skeleton:** `sk.new-traditional-da`: investigation → concept design → approval documentation →
approvals → construction documentation → procurement → construction → completion → defects. [K:
knowledge/program/skeletons.yaml]

**The order a PM actually runs** [D]:

1. **Investigation.** Planning certificate (s10.7) and title/DP search with easements; Sydney Water sewer
   service diagram and asset check; Before You Dig services search; detail and boundary survey;
   geotechnical investigation (site class). Survey and geotech come **before** concept design.
2. **Concept and design development.** Brief, concept, client review, design development. QS estimate at
   concept if engaged.
3. **Approval documentation.** DA drawings, NatHERS/BASIX, stormwater concept plan, landscape plan, SEE,
   shadow diagrams, waste management plan, client sign-off.
4. **DA approvals.** Lodge, notification, assessment, council requests for information, determination.
5. **Construction documentation.** Construction drawings and specification, structural and stormwater
   design, BASIX revision if needed, Sydney Water Tap in approval.
6. **Construction certificate.** After the construction documents and the CC conditions (long service
   levy, s138, Sydney Water) are met.
7. **Procurement.** Tender documents, tender period, evaluation, contract signed, HBCF certificate,
   deposit. In practice the CC and the tender often run in parallel.
8. **Start.** Appoint the principal certifier, notice of commencement, site establishment.
9. **Construction.** Sediment control, temporary power, building set-out survey, earthworks, footings (inspection), slab
   (inspection), termite barrier, identification survey at slab, frames and trusses, roof, windows, brick veneer and cladding (lock-up),
   services rough-in, frame inspection (before wall batts conceal bracing and tie-downs), insulation, linings, ceilings, internal doors, waterproofing (inspection), tiling, joinery,
   fit-off, painting, floor finishes, stormwater (inspection), driveway and crossing, landscaping,
   utility connections.
10. **Completion.** Practical completion inspection, final inspection and occupation certificate,
    handover with certificates and warranties.
11. **Defects.** Contract defects or maintenance period (if any), then the statutory warranty periods
    under the Home Building Act (s18E: 2 years, 6 years for a major defect; verified: false).

**Logic that drives the job:** survey and geotech → concept; BASIX + stormwater + landscape → DA
lodgement; DA determination → construction documents → CC; CC + contract + HBCF + notice of commencement
→ site start; slab → frames → roof → windows → lock-up → rough-in → linings → waterproofing → tiling →
joinery → fit-off → painting → floors → PC; DNSP connection must be live before final electrical
testing and the occupation certificate.

**Long-lead items:** windows and external doors; joinery; frames and trusses; DNSP connection; NBN lead-in
and activation; Sydney Water connection. The app holds lead-time rows for windows, joinery, the utility
supply and services connections. [K: knowledge/program/fragments.yaml fr.windows-1a, fr.joinery-1a,
fr.utility-1a, fr.services-connections-1a]

**What the app gets wrong in order.** `fr.construction-certificate` is an `approvals`-stage fragment
with no `after` link, and the skeleton puts `approvals` before `construction_documentation`. So the CC
lands before the construction documents it depends on. [K: knowledge/program/fragments.yaml
fr.construction-certificate; knowledge/program/skeletons.yaml sk.new-traditional-da] The DA notification
row is labelled "Public exhibition" (`fr.da-exhibition`), which overstates a dwelling's neighbour
notification. `fr.ac-1a` attaches after linings, but ducted air needs its ducts in before the ceilings. [K:
knowledge/program/fragments.yaml fr.ac-1a]

---

## 8. RFP shape

The `house` RFP module is on for classes 1a + 10a and `beyond_house` is off, so commercial rows (BIM,
tenant fit-out, signage, façade certification) drop out. [K: knowledge/rfp/modules.yaml house] Shared
returnables R1–R9 apply to every RFP. [K: knowledge/rfp/skeleton.yaml returnables]

### Architect (`architecture`)

- **Must carry:** concept and schematic design; DA (planning approval) documentation, with coordination
  of the survey, stormwater, landscape and BASIX inputs (a PM also expects shadow diagrams and the BASIX
  commitments on the drawings; the template names neither, so that is a domain-only gap); lodging the DA
  through the NSW Planning Portal (`flag.architect_lodges: yes` in the base); planning response to council requests; design development; tender and construction
  documents for a house (`house_documents`), with kitchen, bathroom, laundry and joinery design and a
  finishes, fixtures and appliances selections schedule; specification; bidding/tender support; liaison
  with the certifier.
- **Options:** construction-phase site visits and design support; contract administration under the
  domestic contract (only if the owner asks, `flag contract_admin`); s4.55 modification applications;
  loose FF&E.
- **Must not carry:** BIM coordination or clash detection; tenant fit-out; signage; façade certification;
  apartment design-quality submission; Green Star or NABERS; regulated-design declarations (Design and
  Building Practitioners Act does not reach a Class 1a house; verified: false); D&C tender documents or
  post-novation stages; community consultation.
- **App note:** `required_facts: [budget]` raises the blocking check `fact:budget` until a construction
  budget is stated. The scenario documents state one (a scenario input, not an app price); without it the
  key expects the block. [K: knowledge/rfp/architecture.yaml]

### Structural (`structural`)

- **Must carry:** design basis (site class from the geotechnical report, wind class to AS 4055); footing
  and slab design to AS 2870; footing, framing, bracing and tie-down design with review of the supplier's
  truss and frame design (`residential_framing`, which also covers beams and lintels); retaining walls on
  the lot if any; construction documents and a design certificate for the CC.
- **Option:** construction-phase inspections with an inspection schedule (piers or footings,
  reinforcement before the pour) and certificates for the OC. `inspection_schedule` is included only
  when `construction_phase` is, and that stage is an option at simple scale.
- **Must not carry:** shared Revit model; shoring and retention design or dilapidation (unless the
  basement trigger); existing-structure assessment; authority-asset stage (unless the sewer-main trigger);
  regulated design; D&C or post-novation stages; super-flat floor.
- [K: knowledge/rfp/structural.yaml residential_framing]

### Civil / stormwater (`civil`)

- **Must carry:** stormwater concept plan for the DA with the discharge point confirmed (kerb by gravity);
  detailed stormwater design for the CC (pits, pipes, rainwater tank overflow, OSD only if council
  requires); driveway long section and boundary levels for council's vehicular crossing; erosion and
  sediment control plan; works-as-executed review and certification for the OC.
- **Must not carry:** public road works; MUSIC water-quality modelling or NorBE (unless council or the
  catchment requires them); flood report (unless the flood trigger); 3D civil model; design criteria
  document; road safety audit; trunk drainage; regulated-design declarations; D&C stages.
- **App note:** the `roads` element (Roads, pavements and levels) should hold only the driveway, and the
  always-included DRAINS model is more than a single-lot gravity system needs. [K: knowledge/rfp/civil.yaml
  drainage_model]

**Expected states.** Each RFP in `shape.yaml` carries an `expect` block in the
`data/eval/rfp/answer-keys` format (stages include / option / absent, positions I / O / X / absent) with
the correct state for this house, and an `app_differs` list. The base profile sets every settable RFP flag
that does not apply (`public_road`, `norbe`, `stormwater_quality`, `sustainability_target`, `panel`,
`community`, `groundwater`, `rail`, `transfer_structures`, `coastal`) to `no`, so that unknowns do not turn
excluded rows into priced options. Even so the app still prices some commercial rows as options
whatever the flags say: civil `water_quality` and `civil_model`, structural `shared_model` and
`super_flat_floor`, surveyor `titling`, `interior` and `monitoring`, geotechnical `monitoring` and
`transfer`, town-planning `panel_attendance`. Those RFPs are therefore `built_in: partial`.

### Geotechnical (`geotechnical`)

- **Must carry:** investigation plan; field work (two or three boreholes or test pits with dynamic cone
  penetrometer tests); laboratory reactivity testing; a report with the AS 2870 site classification,
  founding depth and bearing, and footing recommendations; site-specific notes on fill, rock and
  groundwater. Option: pier-base inspections during construction.
- **Must not carry:** contamination or waste classification; groundwater monitoring wells; acid sulfate
  soils assessment (unless mapped); asset exposure; novation or transfer.

### Surveyor (`land-surveying`)

- **Must carry:** detail and contour survey on AHD with trees, levels and street features; boundary
  identification with easements from title; adjoining-building windows, ridge heights and trees for the
  DA; Sydney Water sewer location from records. Options: building set-out; identification surveys at slab
  and frame or roof stage; final survey for the OC.
- **Must not carry:** subdivision or strata titling; interior scan; movement monitoring (unless
  excavation); occupied-site access.

### Certifier (`bca-certification`)

- **Must carry:** Package B only: CC assessment, principal-certifier services with every NSW
  critical-stage inspection listed in §5, and the occupation certificate with the BASIX completion check.
- **Must not carry:** the A package (BCA advisory at inception, planning and tender); authoring
  performance solutions; design of any kind; building supervision or contractual quality checks;
  classification maps, staged occupation and fire referrals.
- **App note:** without the client flag `bca_certification` the boundary defaults to the advisory A
  package, which is the wrong appointment for a house certifier. [K: knowledge/rfp/bca-certification.yaml
  boundary]

### BASIX / NatHERS assessor (no template)

- **Must carry:** NatHERS thermal assessment with stamped plans; BASIX certificate (water, energy, thermal
  comfort); one revision for the CC design.
- **Must not carry:** NCC Section J; Green Star; NABERS.
- The app has no template; a package for `consultant.basix` gets the generic `tpl.rfp-capex`.

### Landscape (`landscape-architecture`)

- **Must carry:** concept and DA landscape plan to the council DCP (planting schedule, landscaped-area and
  deep-soil compliance, tree replacement); option: documentation for tender and a completion inspection.
- **Must not carry:** public-domain package; podium or planter build-ups; detailed irrigation design
  (option at most); tree assessment (arborist).
- **App note:** every design stage needs the client flag `landscape_design`. [K:
  knowledge/rfp/landscape-architecture.yaml]

### Town planner (`town-planning`, optional)

- **Must carry:** preliminary advice on LEP and DCP controls (height, FSR, setbacks, landscaped area,
  privacy); statement of environmental effects; lodgement; assessment and responses to council requests.
- **Must not carry:** a clause 4.6 variation request (unless a standard is exceeded); panel attendance
  (unless objections send it there); SSD; specialist reports.

### Quantity surveyor (`cost-planning`, optional)

- **Must carry:** cost basis, concept estimate, pre-tender estimate, tender comparison.
- **Must not carry:** final account or post-contract advice unless selected; remediation costs.

---

## 9. Triggers

Seventeen common site and building conditions that add work. Each changes procurement, cost plan, program
or RFP. The machine-readable form is in `shape.yaml`.

| Id | How a document shows it (directly and indirectly) | How a document denies it | What it adds |
|---|---|---|---|
| `cdc-pathway` | "Complying development certificate under the Housing Code"; "the design complies with the Housing Code so the certifier can approve it in about 20 days" | "A development application will be lodged with council"; "The design exceeds the Housing Code side-setback standard, so a DA will be lodged" | CDC application to the certifier instead of the DA; no DA fee or notification but contributions still payable, a CDC fee and no CC fee (the app drops the whole DA-fees line and keeps the CC line); no town planner (the app still suggests one through `baselines[0]`); certifier RFP must carry the CDC. The app swaps to `sk.new-traditional-cdc` (which has no construction-documentation stage) and the certifier and town-planning RFPs block on `cdc`. |
| `bushfire-prone` | "Mapped bush fire prone land (Vegetation Category 1)"; "BAL-29 assessment"; "the lot backs onto the bushland reserve" | "Not identified as bush fire prone land on the planning certificate" | Bushfire consultant (BAL report); AS 3959 construction (ember screens, toughened glass, non-combustible cladding); s4.14 assessment, RFS referral at BAL-40 or FZ (verified: false); cost and program items |
| `flood-affected` | "Subject to flood related development controls"; "habitable floor 500 mm above the 1% AEP level"; "overland flow path across the rear yard" | "The land is not subject to flood related development controls" | Council flood advice; civil flood report and floor levels; flood-compatible materials below FPL; CDC usually excluded |
| `heritage-conservation-area` | "Within the Ashwell Heritage Conservation Area"; "council's heritage officer"; "infill in a conservation area" | "The site is not a heritage item and is not in a conservation area" | Heritage consultant (heritage impact statement); DA required; heritage referral; complex architecture and town-planning scale |
| `trees-affected` | "The driveway cut encroaches into the tree protection zone of the neighbour's jacaranda"; "the rear slab edge is within the structural root zone of the blue gum" | "No trees on or next to the site are affected by the works"; "a large blue gum near the rear boundary" (present but unaffected: tree presence alone is not an impact) | Arborist (AS 4970 impact assessment and protection plan); tree protection fencing; council tree permit or DA approval; supervision hold points |
| `sloping-site` | "The site falls about 4 m from the street to the rear"; "split-level with a 1.8 m cut"; "retaining wall along the eastern boundary" | "The site is generally level" | Retaining walls; piers or suspended slab; extra earthworks; structural retaining design; CDC limits on cut and fill |
| `stormwater-easement` | "The lot falls away from the street; stormwater cannot reach the kerb by gravity"; "an interallotment drainage easement over No. 14 is needed"; "charged system"; "pump-out" | "Stormwater discharges by gravity to the kerb in Tamar Street" | Civil design of the easement or pump-out; downhill owner's consent; easement survey and legal costs; DA cannot be determined until resolved |
| `osd-required` | "OSD is required under council's stormwater policy"; "on-site detention tank under the driveway" | "The site is exempt from on-site detention" | Civil OSD design; OSD tank; positive covenant and restriction on title |
| `sewer-main-in-lot` | "A Sydney Water sewer main crosses the rear of the lot"; "the sewer diagram shows the main 1.2 m from the rear boundary"; "building over or next to a Sydney Water asset" | "No Sydney Water assets are within the property" | Sydney Water building-over or adjacent-to-asset approval; piers to below the zone of influence; possible CCTV of the main; structural authority-asset stage |
| `reactive-site-class` | "Site classification Class H1 in accordance with AS 2870"; "highly reactive clay"; "Class P due to uncontrolled fill" | "Site classification: Class M"; "Class S" | H1/H2: stiffened raft or waffle slab with deeper edge beams and an engineer's footing inspection. Piers only for Class P, E or founding to rock. Drainage controls around the building. No app delta today (`site_class` is read by nothing). |
| `swimming-pool` | "In-ground concrete pool with frameless glass fencing"; "plunge pool next to the alfresco" | "No pool is proposed" | Pool and AS 1926.1 barrier; pool builder; pool registration and barrier compliance certificate; BASIX pool commitments; usually its own approval |
| `gas-connection` | "Natural gas to the cooktop and instantaneous hot water"; "Jemena meter on the side boundary" | "All-electric home: induction cooktop and heat-pump hot water" | Gas piping work item; Jemena connection; gas compliance certificate |
| `basement-or-boundary-excavation` | "Basement garage excavated to a depth of 3 m"; "excavation within 900 mm of the neighbour's dwelling" | "No basement; slab on ground at natural level" | Retention and shoring design; dilapidation surveys of neighbours; waterproofing; groundwater advice; possible ventilation |
| `builder-designs` | "Design and construct contract with a project-home builder"; "the builder's drafting team will prepare the DA and CC drawings" | "Architect-led traditional lump-sum tender" | No owner-engaged design team: the builder's in-house drafting and engineers produce the DA and CC set. The owner appoints only the principal certifier (and optionally an independent inspector). Program: builder's preliminary (design) agreement, builder prepares DA/CC set, new-homes contract; no tender. Nothing to novate. The app instead sets novation `option` on every consultant, adds `fr.novation` and post-novation RFP stages and moves to `sk.new-dc-da`: a gap, not the answer. |
| `old-structure-on-lot` | "Remove the old fibro garage out the back"; "existing concrete slab and shed to be demolished"; "the lot was cleared except for a 1960s outbuilding" | "The lot is vacant and clear"; "no structures on site"; "No demolition is required" (also keyword trap 1) | Demolition by a licensed contractor; asbestos survey and licensed removal; demolition notification to council and neighbours; clearance certificate; possibly a demolition approval in the DA. The app's `fr.demolition-1a` fires only if the action is set to `remove`; asbestos rows have no new-build fragment. |
| `acid-sulfate-soils` | "The site is Class 3 on the Acid Sulfate Soils Map"; "an acid sulfate soils management plan is required for excavation" | "The land is not mapped as acid sulfate soils" | ASS assessment (geotechnical) and management plan under the LEP's acid sulfate soils clause (number varies by LEP; verified: false). The only app reaction is the civil `earthworks_suitability` row; `fr.investigate-acid-sulfate` needs an investigate item. |
| `coastal-exposure` | "The lot is 300 m from the ocean and within the coastal use area"; "marine-grade fixings and sheeting required for the salt-spray zone" | "The site is not in a coastal management area and is well away from breaking surf" | Corrosion-resistant roofing, fixings, lintels and windows; corrosivity and wind class in the structural design basis; coastal management provisions of SEPP (Resilience and Hazards) 2021 (verified: false). `flag.coastal` exists but no RFP row reads it. |

Less common NSW variants left out of the key: unsewered lot (on-site wastewater system, s68 Local
Government Act approval, verified: false), saline soils in western Sydney, road or rail noise (acoustic
report), battle-axe or narrow access, an upper-floor balcony or deck over a habitable room
(`envelope.external-waterproofing`, a leading defect source), an overhead power line or DNSP easement over
the frontage, and a solar PV array chosen to meet the BASIX energy target.

**Zero-delta triggers.** `stormwater-easement`, `reactive-site-class`, `gas-connection` and `coastal-exposure`
change nothing the app produces today: there is no settable key, or no generator reads the key (`site_class`,
`gas_supply`, `flag.coastal`, `wind_class`), and the civil `discharge` position is already included in the base. The harness should
expect no change from them; they test that the app does not invent one.

---

## 10. Keyword traps and negations specific to this type

1. **"No demolition required" implies demolition.** `implied_by strip-out` matches `demoli(tion|sh)` with
   no negation check, so a vacant-lot statement adds `site.demolition`. [K:
   knowledge/profile/scope_defaults.yaml implied_by; internal/profile/implied.go ImpliedReadings]
2. **"Detached, no party walls" implies separation.** `implied_by party-wall` adds
   `site.adjoining-structures`, `fire-passive.compartments` and `interiors.acoustic-separation`. A detached
   house has none of them. [K: same]
3. **"Class 1 dwelling" is not read as a class.** The `ncc_class` trigger needs `1a`/`1b`, "building
   classification" or `class 2–8 <noun>`. A document saying only "Class 1" leaves `det.ncc_class` unset,
   which multiplies conditional `If …` program rows. [K: knowledge/determinants.yaml ncc_class triggers]
4. **H2 timber is not site class H2.** "H2 treated pine" (termite hazard class) or "Class 2 durability"
   next to "site classification" can mislead a reading. Site class is from the geotechnical report only.
5. **BAL-LOW adds a bushfire consultant.** Any `det.bal` value, including BAL-LOW, fires
   `complexity_additions[5]`, though BAL-LOW needs no AS 3959 construction. [K:
   knowledge/works/package_defaults.yaml complexity_additions[5]]
6. **"Lodge" asks the architect-lodges question.** The `architect_lodges` flag fires on any `lodge`,
   `lodged`, `lodgement` or `lodging`, including "the owner will lodge" and "lodgement fee". [K:
   knowledge/profile/flag_readings.yaml architect_lodges]
7. **"Public exhibition"** sets the `community` flag; a dwelling DA gets neighbour notification. A
   council letter about "notification" is not an exhibition.
8. **"Sewer main" is not always an asset in the lot.** "Connect to the sewer main in the street" matches
   the `authority_asset` trigger. Only a main inside the lot, or within the zone of influence, matters.
   [K: knowledge/profile/flag_readings.yaml authority_asset]
9. **"Excavate to a depth of 300 mm"** for the slab matches the `basement` trigger
   (`excavat\w* to a depth`). [K: knowledge/profile/flag_readings.yaml basement]
10. **"No on-site detention required"** matches `osd_required` and `\bosd\b`; the answer must be no.
11. **"Flood"** matches `\bflood`, including "floodlights" over the driveway and "not flood affected".
12. **"Heritage"** matches estate and street names ("Heritage Park estate", "Heritage Drive") and "not a
    heritage item".
13. **Scale counts.** The `bedrooms` scale trigger `\bbed(room)?s?\b` matches "garden beds"; the
    `bathrooms` trigger counts "freestanding bath"; `storeys` matches "single-storey garage" on a
    two-storey house. [K: knowledge/profile/taxonomy.yaml house scale_fields]
14. **Section 73.** A consultant's checklist or the app's rule may say a Section 73 certificate is needed.
    For one house on an existing lot it usually is not, unless the consent says so.
15. **"Granny flat" or "future pool".** A note that a secondary dwelling or pool is "future, not part of
    this application" must not add scope.
16. **BASIX lists gas.** A BASIX certificate may list a gas cooktop even when the brief says all-electric.
    That is a conflict for the owner to resolve, not proof of a gas connection.
17. **"Certifier" in two places.** A certifier's fee proposal is a consultant fee; council's CC lodgement
    fee is a charge. Do not count them as the same line twice.

---

## 11. Gaps in the built-in knowledge

What is missing, and what the app will probably produce wrongly as a result. The `gaps` list in
`shape.yaml` cites the evidence for each.

1. **Consultants a NSW house always needs are never suggested:** BASIX/NatHERS assessor (every NSW house
   needs one), Landscape (most Sydney DAs), Arborist (trees trigger). No package rule names them. [K:
   knowledge/works/package_defaults.yaml]
2. **Services engineers are over-suggested.** Electrical, Hydraulic and Mechanical are suggested for every
   new house through `baselines[0]` and `scope_additions`; a house uses design-and-install trades.
3. **Default scope misses the stair, balustrades, brick veneer, steel beams, rainwater tank, vehicle
   crossing, temporary power, appliances and the geotechnical investigation.** So no access-and-egress cost
   line, no stair or balustrade program rows, no `fr.masonry-1a` unless a document says "brickwork", and no
   `fr.temporary-power`. [K: knowledge/profile/scope_defaults.yaml classes[0]]
4. **CC before construction documents.** `fr.construction-certificate` sits in the approvals stage, which
   `sk.new-traditional-da` runs before construction documentation.
5. **No program rows for NSW house steps:** BASIX certificate, Sydney Water Tap in, long service levy,
   HBCF, notice of commencement and principal certifier appointment, s138 crossing, NBN, critical-stage
   inspections other than the frame hold point, rainwater tank, driveway, garage door, stair, set-out and
   identification surveys.
6. **Investigation fragments do not fire on default actions.** `fr.investigate-survey` and
   `fr.investigate-geotech` need `investigate` actions; the defaults are `new`, and
   `site.ground-conditions` is not in the list at all.
7. **The certifier RFP defaults to the BCA advisory package** and blocks on `cdc`; the town-planning RFP
   also blocks on `cdc`. A CDC house, whose approval *is* the certifier's, cannot get a certifier RFP.
8. **No template** for the BASIX assessor, bushfire, heritage or hazmat consultants; they get the generic
   `tpl.rfp-capex`.
9. **Many RFP rows wait on client flags** the validator cannot express (`bca_certification`,
   `landscape_design`, `geotech_intrusive`, `survey_detail`, `survey_boundary`, `hyd_rainwater` and
   others), so most stages arrive as "ask".
10. **Site conditions that change a house but no generator reads:** `site_class` (reactive clay),
    `wind_class`, `termite_prone_area`, `saline_soil`, `sewer_connection` (septic), `gas_supply`. [K:
    testing/synthetic/reference/index/triggers.md §1]
11. **No 1a fragments for piers, basement retention, retaining walls, pools or pool barriers**, so those
    triggers add no construction rows. [K: knowledge/program/fragments.yaml fr.piles-2, fr.basement-2]
12. **Tree protection never fires on a retained tree.** `fr.tree-protection` needs `site.trees` with
    action new, alter or repair; a retained tree (`retain`) brings no rows, and no package rule suggests
    an arborist.
13. **Cost plan:** no preliminaries line, no long service levy, no council road-reserve fees, no
    provisional sums; Certifier counted twice (F-line and P05); smoke alarms become an "Active fire
    protection" element.
14. **No trade or supplier for a pool builder**, and the app cannot express HBCF insurance or a domestic
    contract's deposit cap.
15. **The Section 73 rule overstates** a single dwelling's water-authority requirement.
16. **D&C route is modelled as novation.** A project-home D&C has no owner design team, but the app sets
    every consultant to novation `option` and adds `fr.novation` and post-novation stages. [K:
    internal/procurement/proposed.go BuilderDesigns; knowledge/program/fragments.yaml fr.novation]
17. **CDC cost and program shape.** CDC drops the whole DA-fees starter (losing contributions a CDC still
    pays), keeps the CC-fees line though no CC issues, has no CDC fee line, and `sk.new-traditional-cdc` has
    no construction-documentation stage although a CDC needs CC-level documents before lodgement. [K:
    knowledge/costs/starter_lines.yaml; knowledge/program/skeletons.yaml sk.new-traditional-cdc]
18. **Architecture RFP blocks on `fact:budget`** in the base scenario until a budget is stated.
19. **House RFPs price commercial options** whatever the flags say (MUSIC, 3D civil model, Revit model,
    super-flat floor, strata titling, interior scan, movement monitoring, groundwater wells, novation
    transfer, panel attendance).
20. **`fr.ac-1a` follows linings**; ducted rough-in must precede ceilings.
21. **No settable key for slope, cut depth or retaining height** (`max_excavation_depth`,
    `max_retaining_wall_height` have no `profile_group`), and `basix_required` is derived, not set.
22. **Demolition needs a `remove` action** that `implied_by strip-out` never gives.
23. **No calendar**: no public holidays or the building industry's Christmas shutdown. [K:
    knowledge/program/calendars.yaml]

---

## 12. Sources and unverified claims

**Opened in the repo:** `knowledge/systems.yaml`; `knowledge/profile/scope_defaults.yaml` (`classes[0]`,
`implied_by`); `knowledge/profile/taxonomy.yaml` (conditions, house scale fields);
`knowledge/profile/flag_readings.yaml`; `knowledge/profile/project_details.yaml`;
`knowledge/determinants.yaml`; `knowledge/works/package_defaults.yaml`; `knowledge/costs/starter_lines.yaml`;
`knowledge/costs/elements.yaml`; `knowledge/program/fragments.yaml`; `knowledge/rfp/architecture.yaml`,
`structural.yaml`, `civil.yaml`, `geotechnical.yaml`, `land-surveying.yaml`, `bca-certification.yaml`,
`town-planning.yaml`, `landscape-architecture.yaml`, `cost-planning.yaml`, `arborist.yaml`, `modules.yaml`;
`knowledge/clusters/envelope/rules.yaml`; `knowledge/clusters/structure/rules.yaml`;
`knowledge/clusters/structure/interfaces.yaml`; `knowledge/clusters/services-wet-air/rules.yaml`;
`internal/profile/implied.go`; `data/intake/disciplines.json`; and the shared index under
`testing/synthetic/reference/`.

**Instruments named from professional knowledge, every clause `verified: false`:** EP&A Act 1979 (Part 4,
s4.14, s4.15, s6.6, s6.9, s7.11, s7.12, s10.7) and EP&A (Development Certification and Fire Safety) Regulation 2021 (critical-stage inspections);
SEPP (Exempt and Complying Development Codes) 2008, Part 3 Housing Code; SEPP (Sustainable Buildings)
2022 (BASIX); SEPP (Biodiversity and Conservation) 2021, Chapter 2 (trees); Home Building Act 1989 (s4,
s7, s8, s18B, s18E, Part 6 HBCF); Building and Construction Industry Long Service Payments Act 1986;
Roads Act 1993 s138; Local Government Act 1993 s68; Dividing Fences Act 1991; Work Health and Safety
Regulation 2017 (NSW); Planning for Bush Fire Protection 2019; NCC 2022 Volume Two and the ABCB Housing Provisions (Parts 9.2, 9.5, 10.2,
11.2, 11.3); AS 2870, AS 1684.2, AS 4055, AS 3660.1, AS 3740, AS 2047, AS 1288, AS 3786, AS 3959,
AS 4970, AS 1926.1.

**Claims I am least sure of:**

- That NSW did not adopt the NCC 2022 livable housing provisions for Class 1a.
- The exact NSW critical-stage inspection list for Class 1 and 10 (I listed six from memory of the
  2000 and 2021 Regulations).
- Whether the HIA and MBA NSW domestic contracts carry a defects or maintenance period beyond the
  statutory warranties.
- That a Section 73 certificate is usually not needed for one new house on an existing serviced lot.
- That the app's `fr.da-exhibition` and `fr.construction-certificate` placement is as I read it (from the
  fragment and skeleton files, not a Go run).
- That the critical-stage inspections sit in the EP&A (Development Certification and Fire Safety)
  Regulation 2021 rather than the general EP&A Regulation 2021 (reviewer's belief, not confirmed offline).

## Review applied (29 October 2026)

All 26 findings in `REVIEW.md` were applied to this file and `shape.yaml`. Notes on the open choices:
finding 1, `flag.architect_lodges` set to `yes` (the architect lodges through the Planning Portal in the
base); finding 2(a), each RFP now has an `expect` block in the answer-key format plus `app_differs`;
finding 3, the budget is a `scenario_inputs` entry; finding 16, the two non-settable determinants moved to
`expected_derived`; finding 17, solar PV removed from the base and named among the less common variants
(not a keyed trigger, to keep the trigger list near the brief's 8–15); finding 22, acid sulfate soils and
coastal exposure added as triggers (17 in all).

**Round 2 (29 October 2026).** All seven round-2 findings applied to `shape.yaml`: six more base values set
(`flag.osd_existing`, `flag.country`, `flag.staged`, `flag.multi_building` no;
`det.potentially_contaminated_land`, `det.acid_sulfate_soils_mapped` false), so civil `osd_existing` and
`earthworks_suitability` and architecture `country` are absent in `expect`; `expect` now uses the answer-key
keys exactly (`elements_include`/`elements_absent`, `blocking` as check ids, conditions in `sources`);
`app_differs` is structured `{key, app, expect, why}` and complete, including architect `bidding` priced as
an option; `cdc-pathway` and `builder-designs` carry a machine-readable `removes` block; the QS final
account is an option; the acid sulfate clause note and the zero-delta list are corrected; `scenario_inputs`
carries the wording the documents must use, and `not_expected_cost_plan` carries the app's labels.

## Disputed review findings

None.
