# T12 Secondary dwelling (granny flat): domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (e1a5389 plus the working tree), the shared index under
`testing/synthetic/reference/`, the reviewed T01 pilot, the owner rulings R1–R8, and professional
practice.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. Instrument clause numbers and
statutory figures I could not check against the instrument are written `verified: false`. No prices,
rates or week durations appear. The only durations named are the app's own `du.*` ids.

The closest reviewed map is T01 (new house). This map reuses its form. It departs where a granny flat
differs: CDC rather than DA, a much smaller consultant team, an occupied house on the same lot, and
services taken from that house. **Over-procurement is the main risk this type tests.**

---

## 1. The job

**What it is.** A detached, single-storey secondary dwelling in the backyard of an existing house on a
suburban Sydney lot. The owners keep living in the main house. The base scenario:

| Item | Base value |
|---|---|
| Lot | About 600 m² Torrens-title lot, R2 Low Density Residential zone, reticulated water and sewer, existing two-storey or single-storey house at the front with a side passage and gate to the rear |
| Building | 2 bedrooms, 1 bathroom with a European laundry, open kitchen/living, about 60 m² GFA, single storey, no garage or car space |
| Construction | Waffle or stiffened raft slab (AS 2870), prefabricated timber or light steel frames and trusses, brick veneer or lightweight cladding, metal skillion or low-pitch roof, aluminium windows |
| Services | Water branched from the house's service after the meter; drainage connected to the house drain or the sewer junction; electrical sub-main from the house's switchboard to a sub-board in the flat; split-system air-conditioning; heat-pump or electric hot water; hard-wired smoke alarms; TV and data points. No gas. No new DNSP connection in the base |
| Pathway | Complying development certificate (CDC) from a registered certifier |
| Procurement | Traditional: the owners engage a building designer, surveyor, geotechnical engineer, structural/civil engineer and BASIX assessor directly, then take lump-sum quotes from two or three builders on a domestic contract (HIA or MBA NSW form) |
| Value | Inside the PROJECT_TYPES range of $0.15–0.4M; the shape names no amount |
| Occupation | The owners live in the main house throughout (`hdr.cond.operational_constraints: partial_occupation`) |

**Why this base and not the turnkey builder.** A fixed-price turnkey package from a specialist
granny-flat builder (the builder designs, gets the CDC and builds) is at least as common as the base.
It is the first trigger (`builder-designs`). The owner-engaged designer is the base because it is the
version a PM or architect user of SiteWise manages, and because it gives each consultant a document to
test. The reviewer ruled this choice right (REVIEW.md §2). [D]

**Scale.** One small building, one owner, one builder. The team is five or six small firms. The
structural and civil engineer are usually one firm. The BASIX certificate is often produced by the
designer. [D]

**Who procures it and how.** The owner (often adding rental income or housing a family member) engages a
building designer, rarely a registered architect. The designer recommends the surveyor, geotechnical
engineer and structural/stormwater engineer, and produces or commissions the BASIX certificate. The
owner appoints the certifier, who determines the CDC and acts as principal certifier. Builders quote on
the CDC drawings. The builder must hold a contractor licence and take out Home Building Compensation
Fund (HBCF) cover, since the contract price is over $20,000 (owner ruling R4). [D]

**What makes it go wrong.** [D]

1. **The CDC is not available, found late.** The lot is too small or too narrow, it is in a heritage
   conservation area, flood control lot or high BAL, or the design breaks a setback or floor-area
   standard. The job falls back to a DA, adding months. The planning certificate settles this at the
   start.
2. **The sewer.** A Sydney Water main often runs along the rear boundary of a suburban lot, exactly where
   a granny flat goes. Building over or next to it needs Sydney Water approval and piers. A flat lower
   than the house drain needs a pump.
3. **Service capacity at the main house.** The existing single-phase supply or an old switchboard (often
   asbestos-backed) cannot take the added load. The house's stormwater line is cracked or blocked.
   These surface during construction and become variations.
4. **Access.** A narrow side passage means no excavator, hand-dug footings, concrete pumped over the
   house, frames crane-lifted, and fence panels removed. Quotes that assume machine access are wrong.
5. **Living next to a building site.** The family keeps using the yard, the side passage, the clothes
   line and the bins. Dust, noise, service shutdowns and safety separation have to be planned.
6. **Over-scoping the team.** A small job carrying a town planner, services engineers or a QS it does not
   need. This is the risk the type tests in SiteWise.
7. **Fire separation.** A flat close to the main house or a side boundary needs fire-resisting walls. It
   is missed on a tight lot.
8. **Builder risk.** Turnkey packages with loose provisional sums for site costs ("site costs extra"),
   excluded service connections, and deposits taken before HBCF cover.

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. Actions use `knowledge/works/actions.yaml`. The app's
house/new default list (41 systems, action `new`) is `[K: knowledge/profile/scope_defaults.yaml
classes[0]]`. **+** marks an item the default list lacks. **−** marks a default item this job does not
have.

### `site`

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | investigate | Detail, level and boundary survey before design: setbacks, sewer and house drain locations, the main house's floor level. Keyed `investigate` so `fr.investigate-survey` fires; the builder's set-out is a later row. [K: knowledge/program/fragments.yaml fr.investigate-survey] |
| `site.ground-conditions` **+** | investigate | AS 2870 site classification. Not in the default list. [K: knowledge/program/fragments.yaml fr.investigate-geotech] |
| `site.earthworks` | new | Minor cut and fill; spoil removed down the side passage. |
| `site.erosion-sediment-control` | new | Sediment fence and stabilised access. |
| `site.stormwater-detention` | new | Roof water to the existing site drainage, the kerb or an absorption trench as the stormwater plan directs. OSD only if council requires it (trigger). |
| `site.services-connections` | alter | Tie-ins to the main house's services: water branch after the meter, drainage line to the house drain or junction, electrical sub-main. `alter` because the existing site services are extended, not newly supplied. `fr.services-connections-1a` still fires. [K: fragments.yaml fr.services-connections-1a] |
| `site.adjoining-structures` **+** | retain | The occupied main house, its paths and the boundary fences, protected during the works; side fence panels removed for access are reinstated. |
| `site.landscape` | new | Path from the street, turf reinstatement, the flat's private open space. |
| `site.vehicle-access` **−** | out | No driveway or car space for the flat in the base. In the default list (no fragment). |
| `site.trees` | retain (trigger) | Only when a tree is affected. |
| `site.demolition` | out | Becomes `remove` under `old-shed-demolition`. |
| `site.retaining-walls` | new (trigger) | Sloping backyard only. |
| `site.public-domain` | out | No new vehicle crossing. Road occupancy for a crane is an authority permit, not a work item. |
| `site.flooding`, `site.contamination`, `site.acid-sulfate-soils` | out | Conditions, not base scope. |
| `site.pools` | out | An existing pool nearby is an interface (barrier kept compliant), not works. |
| `site.waste-storage`, `site.impact-protection`, `site.loading-docks`, `site.signage` | out | Not used. |

### `substructure`

| System | Action | Why |
|---|---|---|
| `substructure.footings` | new | Edge beams and pads, founded clear of the house drain and any sewer zone of influence. |
| `substructure.slab-on-ground` | new | Waffle or stiffened raft to AS 2870. [K: knowledge/clusters/structure/rules.yaml rule.as2870.footing-design-by-site-class] |
| `substructure.termite-management` | new | AS 3660.1. [K: knowledge/clusters/structure/rules.yaml rule.ncc.termite-management] |
| `substructure.piles` | new (trigger) | Sewer main or sloping backyard. |
| Others (damp-waterproofing, basement-retention, retaining-walls, underpinning) | out | Slab on ground, no basement, no work to the main house. |

### `structure`

| System | Action | Why |
|---|---|---|
| `structure.timber` | new | Prefabricated frames and trusses. Many granny-flat builders use light steel frames (`structure.steel`) instead; either is fine in a scenario, but only one per scenario. |
| `structure.alterations` | out | Only under `attached-secondary-dwelling` or `garage-conversion`. |
| `structure.concrete`, `.masonry`, `.precast`, `.design-basis`, `.temporary-works` | out | Not used as items; brick veneer is cladding here. |

### `envelope`

| System | Action | Why |
|---|---|---|
| `envelope.external-walls` | new | Brick veneer or lightweight cladding. |
| `envelope.roof-coverings` | new | Metal sheet roof. |
| `envelope.windows-doors` | new | Aluminium windows and sliding doors; long lead. |
| `envelope.flashings-seals`, `.pliable-membranes`, `.thermal-insulation` | new | Standard. |
| `envelope.thermal-performance` | new | BASIX commitments for a new dwelling. [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required] |
| `envelope.vehicle-doors` **−** | out | No garage. In the default list. |
| `envelope.bushfire-construction` | new (trigger) | Bushfire-prone land. |
| Others | out | Not used on a small dwelling. |

### `interiors`

Walls and linings, ceilings, wet-area waterproofing (AS 3740) [K: knowledge/clusters/envelope/rules.yaml
rule.state.waterproofing-inspection-before-tiling], tiling, floor finishes, joinery (kitchen, vanity,
robes), internal doors, internal glazing (shower screen), painting: all `new`, all in the default list.
`interiors.ffe` **+** `new`: cooktop, oven and rangehood supplied by the builder. Acoustic separation and
room acoustics: out for a detached flat (in under `attached-secondary-dwelling`).

### `fire-passive`

Out in the base. A detached flat far enough from the house and the boundaries needs no fire-resisting
construction. Under `close-to-main-house`, the facing external walls (and eaves and openings) need
fire-resisting construction: `fire-passive.external-walls` `new`. Under `attached-secondary-dwelling`, a
separating wall: `fire-passive.compartments` `new`. The distances (900 mm to a boundary, 1.8 m between
buildings on one allotment) are in the ABCB Housing Provisions Part 9.2 under NCC 2022 Volume Two
(verified: false). [D]

### `fire-active`

`fire-active.smoke-alarms` `new`: hard-wired smoke alarms in the flat (AS 3786; Housing Provisions
Part 9.5, verified: false). [K: knowledge/clusters/fire/rules.yaml rule.as3786.smoke-alarms] Nothing else.

### `mechanical`

`mechanical.air-conditioning` `new` (wall-mounted split systems), `mechanical.local-exhaust` `new`.
Everything else out.

### `hydraulic`

| System | Action | Why |
|---|---|---|
| `hydraulic.cold-water` | new | Branched from the main house's service after the meter. A separate meter is an owner option. |
| `hydraulic.hot-water` | new | Heat pump or electric unit to the BASIX commitment. |
| `hydraulic.sanitary` | new | Under-slab drainage to the house drain or junction, inspection opening, overflow relief gully. Pump-out under `drainage-not-by-gravity`. |
| `hydraulic.roof-drainage` | new | Gutters and downpipes. |
| `hydraulic.non-potable-water` | out of base | A rainwater tank is often chosen for BASIX water, but a small dwelling can usually meet the target with fixtures (verified: false). [D] |
| `hydraulic.gas` | out | All-electric. |
| Others | out | Not used. |

### `electrical`

| System | Action | Why |
|---|---|---|
| `electrical.switchboards` | new | Sub-board in the flat, plus a sub-main breaker added to the main house's board. |
| `electrical.cabling-wiring` | new | Trenched sub-main from the house; final subcircuits. |
| `electrical.lighting` | new | Internal and external lighting. |
| `electrical.utility-supply` **−** | out of base | Supply is from the main house. The default list includes it, so the app makes "Lodge the supply application" and "Authority connection" rows a granny flat does not need. It belongs to `electrical-supply-upgrade` (action `upgrade`). [K: knowledge/profile/scope_defaults.yaml classes[0]; knowledge/program/fragments.yaml fr.utility-1a] |
| `electrical.metering` | new (trigger) | Separate meter, under the supply upgrade. |
| `electrical.solar-pv`, `.battery-storage`, `.ev-charging` | out | Owner options. |
| `electrical.temporary-power` | out | The builder usually draws temporary power from the main house's board by agreement. [D] |

### `comms-security`

`comms-security.structured-cabling` `new` (TV and data points). `comms-security.carrier-lead-in` **−**:
out of base; a separate NBN service is an owner option. It is in the default list.

### `vertical-transport`

Out.

### `access-egress`

Out in the base: a single-storey flat on a slab needs no stair or barrier. `access-egress.barriers` under
`sloping-backyard` (a deck more than about 1 m above ground; Housing Provisions Part 11.3, verified:
false). Livable housing is optional in NSW (owner ruling R6).

---

## 3. Existing-building interfaces

The work is a new detached building. The main house is not altered, apart from a breaker in its
switchboard. But the house is occupied and its services feed the flat, so these interfaces drive the job.
[D]

**Is this an "existing building" job?** The app's `det.existing_building` means "Work to an existing
building" [K: knowledge/determinants.yaml existing_building]. The answer for the base is **false**.
Setting it true would switch on the RFP `existing_building` module (existing-structure assessment,
measured drawings of the house) and 363 unforeseen records about altering existing fabric, none of which
fit. The cost of the honest answer is that the one `existing_building` row this job does need, the civil
`retained_services` investigation, is dropped (see §8 and §11). A document saying "the existing building
on the lot" will match the trigger regex `\bexisting building\b` and may set it true: keyword trap 1.
Under `attached-secondary-dwelling` and `garage-conversion` it is true.

**Occupation.**
- The owners (or tenants) live in the main house. The works zone is the backyard. The family keeps the
  front of the lot, the house, and usually some of the yard.
- A site fence separates the works from the family's yard and the side passage, which is often the only
  access to both. Bins, clothes line and pets need a plan.
- Working hours, dust and noise follow the CDC conditions; neighbours are close on three sides.
- `hdr.cond.operational_constraints: partial_occupation` gives the RFP `live_operations` flag and the
  `live_site` module [K: internal/rfp/drivers.go readFlags; knowledge/rfp/modules.yaml live_site]. That
  brings the surveyor's `access` row and the architect's `live_staging` row, both right for this job,
  though the wording is institutional ("agreed with the operator").
- `fr.occupied-protection` needs work type `refurb` and `det.existing_building` true [K:
  knowledge/program/fragments.yaml fr.occupied-protection], so it never fires here.

**Services tie-ins at the main house.**
- **Sewer:** the flat's drainage joins the house drain or the property's sewer junction. The house drain
  in a 1950s–1970s house is often earthenware, its route unknown and its levels shallow. A camera
  inspection and locating before design avoid a pump or a redesign. Where the flat sits lower than the
  connection, a pump-out unit is needed (trigger).
- **Water:** a branch off the house's service after the meter. Old galvanised pipe may limit pressure. A
  separate meter is optional.
- **Electrical:** the electrician's maximum-demand calculation decides whether the existing supply and
  main switchboard take the flat. An upgrade (trigger) goes through the DNSP (Ausgrid, Endeavour Energy
  or Essential Energy) and a Level 2 accredited service provider. A pre-1990 meter board may be asbestos
  backed. A shutdown of the occupied house's power is planned.
- **Stormwater:** the new roof water connects into the house's existing system, whose condition and
  capacity are unknown, or goes to the kerb or an absorption trench. The civil engineer checks it (owner
  ruling R1: the catchment changes).
- **Comms:** wifi from the main house, or a separate NBN service (owner option).
- The app reaches some of this only as Implications: `ic.supplies-keep-served-working` [K:
  knowledge/works/interface_consequences.yaml] needs work items touching existing systems.

**Hazardous materials.** None in the base: no demolition and no work to the house's fabric. An old fibro
shed or garage removed to make room is the common exception (trigger `old-shed-demolition`). Before 2004
the WHS duty to identify asbestos applies (verified: false); the app uses the same year in
`cq.pre-2004-fabric-hazardous-materials-survey`, which also needs `det.existing_building` [K:
knowledge/clusters/structure/consequences.yaml].

**Structure.** Not applicable in the base. Footings are set clear of the house drain and its zone of
influence, and of the main house's footings when the flat is close. [D]

**Matching existing work.** Optional: owners often want the flat's cladding and roof colour to suit the
house. No knowledge item covers it, and it is rarely a constraint. [D]

**Dilapidation.** Not usually needed: no deep excavation. A photographic record of the main house's
paths, the fences and the neighbours' outbuildings near the 900 mm setback is good practice. A
neighbour's access agreement is needed when access is across their yard (trigger
`restricted-side-access`). [D]

**Approval history.** An earlier unapproved structure in the backyard (a converted garage, an enclosed
patio) can block a CDC until it is regularised or removed. The app holds the general case as
`uc.existing-building-or-addition-has-no-approval`, Implications only. [K:
knowledge/clusters/delivery/unforeseen.yaml]

**Fire separation from the main house.** The app's own record `uc.secondary-dwelling-fire-separation`
("Certifier review of separation between a secondary dwelling and the main dwelling") needs work type
`extend` and `det.existing_building` true, so it never fires for the detached new flat it describes. [K:
knowledge/clusters/delivery/unforeseen.yaml uc.secondary-dwelling-fire-separation]

---

## 4. Consultants and why

App suggestion for this profile: Architect, Certifier, Civil, Electrical, Geotechnical, Hydraulic,
Mechanical, Structural, Surveyor, Town Planner. [K: knowledge/works/package_defaults.yaml baselines[0],
scope_additions[2]; testing/synthetic/reference/index/type-predictions.md T12]

| Discipline id | Engaged at | Need | Why | App |
|---|---|---|---|---|
| `consultant.architect` | Brief | Always | Usually a **building designer**. Concept, CDC drawings and site plan, construction drawings, quote package. | yes, `baselines[0]` |
| `consultant.surveyor` | Before concept | Always | Detail and boundary survey; locates the sewer main, house drain and pits; levels at the main house. | yes, `baselines[0]` |
| `consultant.geotechnical` | Before structural design | Always | AS 2870 site classification. | yes, `baselines[0]` |
| `consultant.structural` | Design development | Always | Slab and footings, framing review, design certificate. | yes, `baselines[0]` |
| `consultant.civil` | Approval documentation | Always | Stormwater plan for the CDC; the added roof changes the catchment (owner ruling R1). Usually the structural firm. | yes, `baselines[0]` |
| `consultant.basix` | Approval documentation | Usually | BASIX certificate for a new dwelling, by a NatHERS assessor or by the designer with the DIY method. | **no** rule names it |
| `consultant.certifier` | Approval documentation | Always | Determines the CDC, principal certifier, critical-stage inspections, occupation certificate. | yes, `baselines[0]` |
| `trade.main_works` | Procurement | Always | Licensed builder, often a granny-flat specialist, on a domestic lump-sum contract. | partial: proposed only from gaps on accepted items |
| `consultant.arborist` | Investigation | Trigger | Tree affected. | **no** |
| `consultant.hazmat` | Investigation | Trigger | Old shed or garage removed; attached or conversion. | **no** |
| `consultant.bushfire` | Investigation | Trigger | BAL. | yes when `det.bal` set, `complexity_additions[5]` |
| `consultant.town_planner` | Concept | Trigger (`da-pathway`), optional even then | SEE and DA management; the designer often writes the SEE. | yes, always, `baselines[0]` |
| `consultant.building` | Construction | Optional | Independent inspector for the owner, mostly under a turnkey contract. | **no** |

**Not expected for this job, but the app suggests them** (the over-procurement this type tests): [D]

- `consultant.town_planner`. A CDC has no DA, no SEE and no council assessment. The app adds it through
  `baselines[0]` for every new build, and its RFP then blocks on `cdc` [K:
  knowledge/rfp/town-planning.yaml unsupported_pathways].
- `consultant.electrical`, `consultant.hydraulic`, `consultant.mechanical`. The electrician, plumber
  and AC installer design and certify a 60 m² dwelling's services. The app adds them through
  `baselines[0]` and `scope_additions[2]`–`[4]`.

So the app suggests ten consultants where the job needs six (plus BASIX, which the app misses): four
false positives and one miss. A quantity surveyor, landscape architect, hydraulic engineer and
town planner are all wrong for the base.

---

## 5. Approvals and authorities

Every instrument clause and figure here is from memory, so `verified: false`. [D]

### Planning pathway (base: CDC)

- **Permissibility.** Secondary dwellings are permitted with a dwelling house in the residential zones
  under SEPP (Housing) 2021, Chapter 3, Part 1, which also caps the flat's floor area (60 m², or a higher
  LEP figure) and bars subdividing it off (all verified: false).
- **Complying development.** The complying development standards for secondary dwellings are in SEPP
  (Housing) 2021, Schedule 1 (NSW Planning, "Secondary dwellings",
  https://www.planning.nsw.gov.au/the-planning-system/housing/housing-sepp/secondary-dwellings). The lot
  must be at least 450 m² (confirmed by that page; the exception is a flat wholly inside an existing
  dwelling house). Lot width, setbacks (900 mm side and 3 m rear recalled for a single storey), height,
  landscaped area and private open space are `verified: false`. The principal dwelling's standards are
  in the Codes SEPP (State Environmental Planning Policy (Exempt and Complying Development Codes) 2008).
- **Exclusions.** A heritage item is excluded. Treat a heritage conservation area and BAL-40 or BAL-FZ as
  a DA. A flood control lot is restricted, not barred (all verified: false; reviewer's working set).
  These send the job to a DA (`da-pathway`).
- **Planning certificate** (EP&A Act s10.7, verified: false): read first; it states whether complying
  development may be carried out on the land.
- **Neighbour notices: two notices, two actors.** In a metropolitan area the **certifier** notifies
  neighbours within 20 m, and cannot determine the CDC until 14 days after that notice. The **person with
  the benefit of the CDC** gives neighbours written notice at least 7 days before work starts. Source:
  NSW Planning Portal, "What to tell your neighbours",
  https://www.planningportal.nsw.gov.au/development-and-assessment/planning-approval-pathways/complying-development/what-tell-your-neighbours.
  The Regulation 2021 clause numbers stay `verified: false`.
- **Determination** by the registered certifier within the statutory period [K:
  knowledge/program/fragments.yaml fr.cdc-application; app duration `du.cdc-determination-period`],
  with conditions (contributions, Sydney Water, working hours, prior-to-commencement items).
- **DA alternative** (`da-pathway`): a DA to council, then a construction certificate.

### Certificates

| Certificate | By | When |
|---|---|---|
| BASIX certificate | Assessor or designer via the BASIX portal | With the CDC application [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required, clause_verified false] |
| Complying development certificate | Registered certifier | Before work; there is no separate construction certificate |
| Principal certifier appointment and notice of commencement | The owner appoints, not the builder, also under a turnkey contract (EP&A Act Part 6, verified: false) | Before site establishment |
| HBCF certificate | Builder | Before deposit or work; contract over $20,000 (owner ruling R4) |
| Occupation certificate | Principal certifier | After the final inspection and BASIX completion [K: knowledge/clusters/envelope/rules.yaml rule.basix.completion-before-occupation] |
| Electrical and plumbing compliance certificates, termite notice, waterproofing certificate, engineers' certificates, stormwater works-as-executed | Trades and engineers | Collected for the OC |

### Authority applications [D]

- **Sydney Water:** Tap in building plan approval, which checks building over or next to assets and
  is needed before the CDC is determined (verified: false). [K: knowledge/clusters/services-wet-air/rules.yaml
  rule.water-authority.build-over-sewer] The firm requirement is before work starts. A Section 73
  certificate is **not** needed for a secondary dwelling approved as complying development under the
  Housing SEPP (Sydney Water, "Section 73 requirements for secondary dwellings",
  https://www.sydneywater.com.au/content/dam/sydneywater/documents/section-73-requirements-for-secondary-dwellings.pdf);
  building plan approval is still needed. The exception is a lot in a Priority Sewerage or Pressure Sewer
  Scheme area (trigger `priority-sewerage-area`). The app's rule says one is needed for "most new
  developments" [K: knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate]:
  keyword trap 9.
- **Long Service Corporation:** levy of 0.25% before the CDC issues, where the cost of work is $250,000
  or more including GST (since 1 January 2023; Long Service Corporation,
  https://www.longservice.nsw.gov.au/bci/levy/about-the-levy; figure for owner confirmation). The T12
  range straddles it, so the scenario must state the contract value.
- **Council:** registration of the CDC, contributions (s7.11 or s7.12, verified: false), and a road
  occupancy permit if a crane or concrete pump stands in the street. No s138 vehicle crossing in the
  base.
- **DNSP:** only under `electrical-supply-upgrade`.
- **SafeWork NSW:** only for licensed asbestos removal (`old-shed-demolition`).
- **Fire and Rescue NSW:** none.

### Inspections and hold points

The principal certifier's critical-stage inspections, from owner ruling R7: pre-commencement; after
excavation, before footings; before the slab pour; before covering the frame; before covering wet-area
waterproofing; before covering stormwater connections; final, before the OC. The app makes two hold points: frame
(`if.frame-inspected-before-roof`) and reinforcement (`if.reinforcement-inspected-before-pour`, which
fires through the `structure.concrete` tag on the slab fragment's "Reinforce and pour slab" row,
confirmed by the reviewer's Go run). [K: knowledge/clusters/structure/interfaces.yaml;
knowledge/program/fragments.yaml fr.slab-on-ground-1a]

---

## 6. Cost plan shape

Lines only, never amounts. Sections follow the app. [K: knowledge/costs/starter_lines.yaml;
knowledge/costs/elements.yaml]

**Fees and charges**

| Line | App |
|---|---|
| CDC fee (certifier's determination and council registration) | partial: no CDC line; P05 "Construction certificate and certifier fees" is kept on a CDC and can hold it under the wrong label |
| Development contributions on the CDC | **no**: the `da-fees` starter applies only to `da` [K: starter_lines.yaml da-fees] |
| Long service levy | no |
| Sydney Water Tap in and connection fees | partial: one generic P06 line |
| HBCF insurance | no line; it is the builder's premium, listed so an exclusion is caught |

**Consultants** (one fee line per package): Building designer (the app's "Architect"), Structural,
Civil, Geotechnical, Surveyor, Certifier, BASIX assessor. The app writes Architect, Certifier, Civil,
Electrical, Geotechnical, Hydraulic, Mechanical, Structural, Surveyor and Town Planner [K:
testing/synthetic/reference/index/type-predictions.md T12]. **Four extra fee lines** (Electrical,
Hydraulic, Mechanical, Town Planner) and one missing (BASIX). The certifier is counted twice: the F-line
and P05.

**Construction** (one head contract)

| Element | Holds | App |
|---|---|---|
| Preliminaries and builder's margin | Supervision, insurances, site fence, protection of the occupied house, restricted-access handling, waste | **no** line; a budget spreads it pro rata [K: knowledge/costs/budget_splits.yaml residential_house_new] |
| 01 Site works and external works | Earthworks, sediment control, stormwater, **tie-ins to the main house's services**, path, landscape, fence reinstatement | yes |
| 02 Substructure | Slab, footings, termites | yes |
| 03 Structure | Frames and trusses | yes |
| 04 Building envelope | Cladding, roof, windows, insulation | yes |
| 05 Interiors and finishes | Linings, waterproofing, tiling, kitchen, floors, painting, appliances | yes |
| 08 Mechanical services | Split systems, exhaust | yes |
| 09 Hydraulic services | Plumbing, drainage, hot water | yes |
| 10 Electrical services | Sub-main, sub-board, power, lighting, smoke alarms | yes |
| 11 Communications and security | TV and data | yes |
| Provisional sums and PC items | Undecided selections and unknown service tie-in conditions ("site costs") | no line |

The app also makes **07 Active fire protection** from the smoke-alarm item; a QS carries smoke alarms in
electrical. Listed in `not_expected_cost_plan`.

**Contingency:** design contingency, construction contingency, escalation, exclusions (all yes).
Hidden service conditions at the main house are the usual draw on the construction contingency. Owner
supply lines only when `proj.owner_supply_stated` is read.

---

## 7. Program shape

**Skeleton:** `sk.new-traditional-cdc`: investigation → concept design → approval documentation →
approvals → procurement → construction → completion → defects. It has no construction-documentation
stage. `fr.construction-docs-1a` is placed by `resolveStage` in the nearest earlier stage, approval
documentation [K: internal/program/generate.go resolveStage; knowledge/works/stages.yaml]. **That is
right for a CDC**, which is determined on construction-level documents. The reviewer's Go run confirmed
it, with "Client sign-off for tender" in the same stage.

**The order a PM runs** [D]:

1. **Investigation.** Planning certificate (CDC eligibility); Sydney Water sewer diagram; detail and
   boundary survey; geotechnical investigation; electrician's check of the existing supply and board;
   locating the house drain.
2. **Concept and design development.** Brief within the floor-area limit; concept placed clear of the
   sewer and inside the setbacks; owner review; design development.
3. **CDC documentation.** Drawings and site plan, construction drawings, structural design, stormwater
   plan, BASIX certificate, Sydney Water Tap in approval, owner sign-off.
4. **CDC.** Long service levy if the cost of work is $250,000 or more incl. GST; lodge; the certifier
   notifies neighbours within 20 m; determination no sooner than 14 days after that notice.
5. **Quotes** (in parallel with the CDC): quote package, two or three builders, comparison of
   inclusions and site costs, contract, HBCF certificate.
6. **Start.** The owner appoints the principal certifier and gives notice of commencement; the CDC
   holder gives neighbours written notice at least 7 days before work; pre-commencement inspection; site establishment with separation from the occupied house.
7. **Construction.** Sediment control, set-out, earthworks, footings (inspection), under-slab drainage,
   termite barrier, slab (inspection), frames and trusses, roof, windows, cladding (lock-up), rough-in,
   tie-ins to the house's services, frame inspection, insulation, linings, waterproofing (inspection),
   tiling, kitchen, splits, fit-off, painting, floors, stormwater (inspection), landscape and fence
   reinstatement.
8. **Completion.** PC, stormwater WAE and certificates, final inspection and OC, handover.
9. **Defects.** Contract DLP (length verified: false, owner ruling R8), then statutory warranties.

**Logic that drives the job:** planning certificate and sewer check → concept; survey and geotech →
design development; Tap in + BASIX + stormwater plan → CDC lodgement; CDC + contract + HBCF + notice
→ start; slab → frame → roof → lock-up → rough-in → linings → waterproofing → tiling → joinery →
fit-off → OC. The service tie-ins follow the under-slab drainage and precede landscape reinstatement.

**Long-lead items:** windows, frames and trusses, kitchen; a DNSP upgrade (trigger); a factory slot
for a modular flat (trigger). The app holds lead-time rows for windows and joinery. [K:
knowledge/program/fragments.yaml fr.windows-1a, fr.joinery-1a]

**What the app gets wrong:**
- `fr.utility-1a` ("Lodge the supply application", "Authority connection") fires in the base from the
  default `electrical.utility-supply` item. A flat fed from the main house has no DNSP connection. These
  rows belong to `electrical-supply-upgrade`. [K: knowledge/program/fragments.yaml fr.utility-1a]
- No rows for the planning certificate, Tap in, neighbour notices, levy, HBCF, notice of commencement,
  the pre-commencement inspection, protection of the occupied house, service tie-ins at the main house,
  or the critical-stage inspections other than the frame and slab-pour hold points.
- "Client sign-off for tender" from `fr.construction-docs-1a` lands in approval documentation; harmless,
  since quotes run in parallel with the CDC.
- `fr.ac-1a` follows linings. For wall-mounted splits that is right (unlike T01's ducted system).

---

## 8. RFP shape

The `house` module is on (`det.ncc_class: [1a]`), `beyond_house` off. `live_site` is on from
`partial_occupation`. `existing_building` is off. [K: knowledge/rfp/modules.yaml] Shared returnables
R1–R9 apply. The full `expect` and `app_differs` blocks are in `shape.yaml`.

### Building designer (`architecture`)

- **Must carry:** concept and schematic within the floor-area limit; CDC documentation (site plan with
  setbacks to the boundaries and to the main house, private open space, service routes from the main
  house, BASIX commitments); lodging the CDC through the Planning Portal (`flag.architect_lodges: yes`);
  certifier liaison; design development; tender and construction documents (`house_documents`);
  kitchen, bathroom and laundry layouts; selections schedule; specification; site plan showing
  construction access and protection of the occupied house (`live_staging`).
- **Options:** quote (bidding) support; construction-phase support; contract administration.
- **Must not carry:** planning response to council (no DA: `planning_response` is absent for `cdc`);
  SEE; measured drawings of or alterations to the main house; BIM; tenant fit-out; signage; facade
  certification; apartment design; Green Star; regulated design (R3: not Class 1); D&C; post-novation;
  community consultation.
- **App notes:** blocks on `fact:budget` until a budget is read [K: knowledge/rfp/architecture.yaml
  required_facts]. R4 asks for architect registration numbers, which a building designer does not have.

### Structural (`structural`)

- **Must carry:** design basis with site and wind class; AS 2870 slab and footings; footings clear of the
  house drain and any sewer zone of influence (`services_footings`); framing and truss review
  (`residential_framing`); drawings and design certificate for the CDC (`planning_approval` is right
  here, unlike a house DA).
- **Option:** inspections, with the inspection schedule included whenever that option is taken
  (`inspection_schedule: I`).
- **Must not carry:** existing-structure assessment (detached); Revit; shoring; dilapidation;
  authority-asset stage (unless the sewer trigger); regulated design; D&C.
- [K: knowledge/rfp/structural.yaml]

### Civil / stormwater (`civil`)

- **Must carry:** stormwater plan for the CDC with the discharge confirmed; condition and capacity of the
  main house's stormwater line where the flat connects; detailed design; sediment control; WAE and
  certificate for the OC.
- **Must not carry:** a driveway long section or crossing; public road works; MUSIC; NorBE; flood;
  OSD unless council requires it; 3D model; trunk drainage.
- **App notes:** the `roads` element is always included, but there is no driveway. The
  `retained_services` row ("investigation brief for retained drainage and services (CCTV, survey,
  potholing)") is exactly what this job needs, but it sits in the `existing_building` module and so is
  absent. [K: knowledge/rfp/civil.yaml retained_services] `for_construction` is an option, not an
  include: its deliverable is a construction-certificate issue set, and under a CDC there is no CC. The
  `detailed_tender` set that goes in with the CDC is the construction issue for a 60 m² flat.

### Geotechnical (`geotechnical`)

As T01: plan, field work with access down the side, reactivity testing, AS 2870 report. Access through
the occupied property is the owner's (`access: X`). No contamination, wells or acid sulfate soils. The
app prices `g4_planning` as an option; a CDC has no planning-stage geotechnical submission, so the key
has it absent.

### Surveyor (`land-surveying`)

- **Must carry:** detail and contour survey of the rear yard with the main house's floor level, fences,
  trees and outbuildings; boundary identification for the setbacks; the sewer main, house drain, pits and
  inspection openings (`buried`, `sewer`); access through the occupied house (`access`, on from
  `live_operations`).
- **Options:** set-out; final survey.
- **Must not carry:** subdivision or strata titling (a secondary dwelling cannot be subdivided off,
  verified: false); interior scan; movement monitoring.

### Certifier (`bca-certification`)

- **Must carry:** CDC eligibility check; CDC assessment and determination; the certifier's
  pre-determination neighbour notice; principal certifier; every R7 inspection; OC with the BASIX check.
- **App output besides the block:** A1 and A3 (BCA advisory) come back as "ask" and A4 as an option,
  because the `bca_advisory` client flag cannot be keyed. The key has all A stages absent.
- **The app cannot draft it.** The template lists `cdc` in `unsupported_pathways` (blocking check
  `pathway_unsupported`), and its B stages need `da` or `ssd`. The owner's answer key
  `bca-unsupported-cdc` says a CDC needs its own scope and the CC stages are not relabelled [K:
  knowledge/rfp/bca-certification.yaml; data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml]. So the
  key expects the block, and records the missing CDC scope as the gap. On this job the certifier is the
  approval authority, so this is the most important RFP the app cannot write.

### BASIX assessor (no template)

BASIX certificate (DIY or NatHERS thermal), commitments on the CDC plans, one revision. Not Section J,
Green Star, NABERS or the main house. Gets the generic `tpl.rfp-capex`.

### Not to be issued in the base

Town planner, electrical, hydraulic and mechanical engineers, quantity surveyor, landscape architect.

---

## 9. Triggers

Nineteen conditions, above the brief's 8 to 15. Review round 1 split the attached and garage-conversion
cases (they differ under owner ruling R1 and in the app) and added three the reviewer named as common.
Each changes procurement, cost plan, program or RFP. The machine-readable form is in `shape.yaml`.

| Id | How a document shows it | How a document denies it | What it adds (or removes) |
|---|---|---|---|
| `builder-designs` | "Fixed-price turnkey granny flat package: design, CDC and construction"; "the builder's drafting team will prepare the plans, engineering, BASIX and CDC"; "choose a plan from the builder's range" | "Our designer has documented the flat and we are seeking quotes" | Removes designer, structural, civil, geotech, surveyor and BASIX from the owner's procurement; certifier stays; no tender. App: novation `option` on every consultant, `fr.novation`, and `sk.generic` (no skeleton for new + D&C + CDC). |
| `kit-or-modular` | "Modular granny flat craned into the backyard"; "steel-frame kit home supplied with plans and engineering"; "delivered in two modules" | "Built on site in timber frame" | Supply package (no supplier id fits), crane and transport, road occupancy, factory lead time; owner-builder permit if the owner erects a kit (verified: false). |
| `da-pathway` | "The lot is 420 m², below the CDC minimum, so a DA"; "in a heritage conservation area; needs consent"; "exceeds the CDC rear setback" | "Meets the complying development standards; a CDC will be sought" | DA, notification, assessment, CC, DA fees and contributions; town planner optional. App handles it well (skeleton, fragments, starter line). |
| `sewer-main-in-backyard` | "Sydney Water main along the rear boundary"; "within the zone of influence"; "building over a Sydney Water asset" | "No Sydney Water assets within the property"; "The sewer main is in the street" | Piers, Sydney Water building-over approval, CCTV, structural authority-asset stage, architect water-asset submission. The flag also makes the app's structural scale complex: over-scoping for a granny-flat slab. |
| `drainage-not-by-gravity` | "The backyard falls away; stormwater cannot reach the kerb"; "interallotment easement"; "flat below the house drain, pump-out required" | "Stormwater and sewer drain by gravity to the existing connections" | Easement or absorption trench, sewage pump station. No app change (zero delta). |
| `osd-required` | "OSD is required under council's policy"; "OSD tank under the courtyard" | "Exempt from on-site detention" | Civil OSD design and tank; positive covenant. |
| `trees-affected` | "slab edge within the TPZ of the jacaranda"; "sewer trench through the neighbour's gum's root zone"; "the camphor laurel will be removed" | "No trees affected"; "The large gum in the front yard is well clear of the works" | Arborist (never suggested by the app), tree protection, council tree permit (a CDC does not authorise removing a protected tree, verified: false). The app's arborist RFP prices impact and protection as options, not includes. |
| `electrical-supply-upgrade` | "Maximum demand exceeds the 63 A single-phase supply"; "new consumer mains and meter panel"; "separate meter" | "Maximum demand calculation confirms the supply is adequate" | DNSP application, Level 2 ASP, meter, main board replacement (asbestos risk). The app's rows already appear in the base. |
| `restricted-side-access` | "Side passage only 900 mm wide; no excavator access"; "frames crane-lifted over the house"; "access across the neighbour's yard" | "A 3 m side driveway gives truck access" | Hand excavation, concrete pump, crane, fence removal, neighbour agreement, road occupancy. No app change. |
| `old-shed-demolition` | "Demolish the existing fibro shed"; "the old backyard garage will be removed"; "remove the 1960s outbuilding" | "The rear yard is clear lawn"; "No demolition is required" (trap 3) | Hazmat survey, licensed asbestos removal, demolition, clearance certificate. |
| `close-to-main-house` | "1.4 m from the rear wall of the main house"; "facing walls fire-resisting, no openings"; "The secondary dwelling's rear wall is 1.2 m from the main house's laundry wall" | "5 m clear of the house, 900 mm from every boundary" | Fire-resisting external walls, protected openings and non-combustible eaves facing the house: within 1.8 m of another building on the allotment (Housing Provisions Part 9.2, verified: false). A wall under 900 mm from a boundary also breaks the CDC side setback, so it belongs in a DA variant. The app's own `uc.secondary-dwelling-fire-separation` never fires. |
| `attached-secondary-dwelling` | "Attached to the rear of the main house"; "shares a wall with the house"; "new roof ties into the existing rear roof" | "Detached and free-standing" | Work type extend, `det.existing_building` true; separating wall (fire and sound); hazmat; existing-structure assessment. New roof, so civil stays (R1). The only trigger where `uc.secondary-dwelling-fire-separation` fires (Implication only). |
| `garage-conversion` | "Convert the existing double garage into a granny flat"; "within the existing garage walls and roof; no new hard surfaces" | "The garage is retained as a garage" | Work type refurb; hazmat; existing-structure assessment; slab, damp-proofing and insulation upgrade; door infill; change of use from Class 10a to 1a. **Removes civil** (R1: roof and hard surfaces unchanged) and geotechnical (existing slab kept, unless new footings are needed). The app moves to `sk.refurb-traditional-cdc`, which has no approval documentation stage, so the base's BASIX, stormwater, Tap in and lodgement sign-off rows move or vanish. The app's refurb scope starts empty and only `baselines[2]` applies, so it drops surveyor, geotech, structural and civil and keeps two services engineers. |
| `accessible-for-ageing-parent` | "Step-free shower and wider doors for Mum"; "designed to the Livable Housing silver level" | "No accessibility features are required" | Livable features on the drawings by the designer. No access consultant (R6: optional for Class 1a). A document stating the features are included, read as a presence row, adds `access-egress.livable-housing`, which suggests an Access consultant through `scope_additions[5]`: over-procurement to report. |
| `priority-sewerage-area` | "In a Sydney Water Priority Sewerage Program area"; "pressure sewer scheme" | "Gravity sewer main available in the street" | Section 73 certificate through a Water Servicing Coordinator, and its fees. |
| `unapproved-existing-structure` | "The rear studio was built without approval"; "no record of approval for the enclosed patio" | "All structures on the lot are approved" | Building information certificate, or removal, before the CDC. The app reaches it only as an Implication. |
| `sloping-backyard` | "The rear yard falls 2.5 m"; "suspended floor on piers with a deck"; "retaining wall along the rear" | "The backyard is generally level" | Piers, retaining walls, deck balustrade; structural pier and wall design. |
| `bushfire-prone` | "Mapped bush fire prone land"; "BAL-19 assessment"; "adjoins the bushland reserve" | "Not identified as bush fire prone land" | Bushfire consultant, AS 3959 construction; CDC not available at BAL-40 or FZ (verified: false). |
| `flood-affected` | "Subject to flood related development controls"; "overland flow path across the rear yard" | "Not subject to flood related development controls" | Flood report, floor levels, flood-compatible materials; often a DA (verified: false). The flood flag also makes the app's civil scale complex. |

**Zero-delta triggers.** `drainage-not-by-gravity` and `restricted-side-access` change nothing the app
produces today (no settable key, or `hdr.cond.access_constraints` is read by nothing; the civil
`discharge` position is already I). `electrical-supply-upgrade`'s program rows already appear in the
base. `unapproved-existing-structure` reaches only Implications. The harness should expect no change
from them.

Less common variants left out of the key: an unsewered lot (on-site wastewater, s68 approval, verified:
false), a two-storey flat over a garage, a rural-zone secondary dwelling (different floor-area rules),
a flat used for short-term letting, an existing pool whose barrier the works disturb, and a gas
connection.

---

## 10. Keyword traps and negations specific to this type

1. **"Existing building" is everywhere.** Granny-flat documents describe the main house constantly
   ("the existing building on the lot is retained"). The `existing_building` trigger
   `\bexisting building\b` then reads "work to an existing building" as true, switching on the
   existing-building RFP module and hundreds of unforeseen records. The work is not to that building.
   [K: knowledge/determinants.yaml existing_building]
2. **Two dwellings, one set of scale fields.** "Existing four-bedroom, two-storey house; new two-bedroom
   single-storey secondary dwelling of 60 m²". The `bedrooms`, `bathrooms`, `storeys` and `gfa` triggers
   read both, giving conflicts. The key is the flat's values. [K: knowledge/profile/taxonomy.yaml house
   scale_fields] "Existing GFA 180 m²" belongs to the main house, not `hdr.scale.existing_gfa_sqm` for
   this job. "Two buildings on the lot" matches `multi_building` (`(two|…) buildings`), which turns on the
   staged module; the answer is no. [K: knowledge/profile/flag_readings.yaml multi_building]
3. **"No demolition required"** adds `site.demolition` through `implied_by strip-out`, which has no
   negation. [K: knowledge/profile/scope_defaults.yaml implied_by; internal/profile/implied.go]
4. **"No separating wall is needed"** adds `fire-passive.compartments`, `interiors.acoustic-separation`
   and `site.adjoining-structures` through `implied_by party-wall` (`separating wall`). Only
   `attached-secondary-dwelling` has one.
5. **"Connect to the sewer main"** matches `authority_asset` (`(sewer|water) main`). Only a main inside
   the lot, near the flat, matters. [K: knowledge/profile/flag_readings.yaml authority_asset]
6. **"900 mm from the boundary".** Excavation wording near the CDC side setback ("footings excavated
   900 mm from the side boundary") matches `excavation_adjoining`, which prices dilapidation and
   shoring. A shallow footing near a fence is not adjoining excavation.
7. **"Garage".** "The existing garage is retained" or "no garage" sets `garage_spaces`; and an existing
   Class 10a garage on the lot can be read into `det.ncc_class`. The flat is `1a` only.
8. **"Dual occupancy".** A secondary dwelling is not a dual occupancy (which can be subdivided and is
   T11). "This is not a dual occupancy" must not change the subclass. "Cannot be subdivided" must not
   bring strata titling.
9. **Section 73.** A checklist or the app's rule may say a Section 73 certificate is needed. For a
   secondary dwelling with no subdivision it usually is not (verified: false).
10. **"Lodge".** `architect_lodges` fires on any `lodge`, including "the certifier will lodge the CDC on
    the Planning Portal" and "lodgement fee". [K: flag_readings.yaml architect_lodges]
11. **"Occupied" or "tenant".** "The tenant of the main house" means the property is occupied
    (`partial_occupation`), not that the flat is a rental fit-out.
12. **"Built in 1965".** `existing_building_year` reads the main house's year. It matters only for a
    shed to be demolished or for the attached variant.
13. **"Heritage" and "flood"** match street and estate names and "not flood affected", as in T01.
14. **"Turnkey" with "site costs extra".** A turnkey quote is a D&C contract, but "site costs",
    "connections" and "access" are often excluded. An exclusion is not a sign the owner engages others.
15. **"Granny flat" in a new-house brief.** A note that a secondary dwelling is "future, not part of
    this application" must not add a second dwelling (T01 trap 15).

---

## 11. Gaps in the built-in knowledge

What is missing, and what the app will probably produce wrongly. The `gaps` list in `shape.yaml` cites the
evidence for each.

1. **Over-procurement.** Ten suggested consultants where six are needed: Town Planner on a CDC job,
   and Electrical, Hydraulic and Mechanical for trade-designed services. BASIX is never suggested.
   On cost-plan prepare these become four extra fee lines. [K: knowledge/works/package_defaults.yaml
   baselines[0], scope_additions[2]]
2. **No certifier RFP on a CDC.** The approval authority of the base job cannot be sent an RFP:
   `bca-certification` blocks on `cdc` and its B stages need `da` or `ssd`. The owner's key expects the
   block; a CDC scope is the missing knowledge. The suggested Town Planner's RFP also blocks.
3. **The occupied house and its services are invisible to a "new" job.** No secondary-dwelling subclass
   or flag. `det.existing_building` is false (rightly), so the civil `retained_services` row and
   `fr.occupied-protection` never apply, and `uc.secondary-dwelling-fire-separation` needs `extend`.
   Nothing models service tie-ins at the main house.
4. **House default scope does not fit.** It adds a DNSP utility supply (so `fr.utility-1a` rows appear),
   a garage door, a driveway and an NBN lead-in, and lacks the ground investigation, protection of the
   adjoining house and fences, and appliances.
5. **Scale fields conflict** when documents describe both dwellings.
6. **No CDC cost shape.** The contributions line disappears with `da-fees`; P05 keeps a CC label; no CDC
   fee, levy, HBCF or preliminaries; Certifier counted twice; smoke alarms make a 07 element.
7. **No program rows** for the CDC eligibility check, Tap in, neighbour notices, levy, HBCF, notice of
   commencement, pre-commencement inspection, occupied-house protection, service tie-ins, or the R7
   inspections other than the frame and slab-pour hold points.
8. **Turnkey D&C with CDC** matches no skeleton (`sk.generic`) and adds novation. A granny-flat builder
   designs in house; there is nothing to novate.
9. **No 1a fragments** for piers, retaining walls, crane lifts or modular installation; no supplier id
   for a modular or kit supplier.
10. **Site conditions with no effect:** `hdr.cond.access_constraints` (restricted access), slope, site
    class, sewer levels.
11. **House RFPs price commercial options** (MUSIC, 3D civil model, Revit, strata titling, interior
    scan, monitoring, groundwater wells, transfer), and the civil `roads` element is always on.
12. **Architecture template** asks for architect registration and "the operator"; the norm here is a
    building designer and occupying owners.
13. **No calendar** of public holidays or the Christmas shutdown. [K: knowledge/program/calendars.yaml]

---

## 12. Sources and unverified claims

**Opened in the repo:** `knowledge/systems.yaml` (via the systems index); `knowledge/profile/
scope_defaults.yaml` (`classes[0]`, `implied_by`); `knowledge/profile/taxonomy.yaml` (house subclass,
conditions); `knowledge/profile/flag_readings.yaml`; `knowledge/determinants.yaml` (`existing_building`,
`sewer_connection`, `bal`, `basix_required` and others); `knowledge/works/package_defaults.yaml`;
`knowledge/works/system_terms.yaml` ("Secondary dwelling"); `knowledge/costs/starter_lines.yaml`;
`knowledge/costs/elements.yaml`; `knowledge/program/fragments.yaml`; `knowledge/program/skeletons.yaml`;
`knowledge/works/stages.yaml`; `knowledge/rfp/modules.yaml`; the RFP templates through
`testing/synthetic/reference/index/rfp-templates.md` (architecture, structural, civil, geotechnical,
land-surveying, bca-certification, arborist); `knowledge/clusters/envelope/rules.yaml`
(`rule.basix.certificate-required`); `knowledge/clusters/services-wet-air/rules.yaml`
(`rule.nsw.section-73-certificate`, `rule.water-authority.build-over-sewer`);
`knowledge/clusters/fire/rules.yaml` (`rule.ncc.external-wall-fire`);
`knowledge/clusters/delivery/unforeseen.yaml` (`uc.secondary-dwelling-fire-separation`);
`internal/rfp/drivers.go` (flags from operational constraints and work type);
`internal/program/generate.go` (`resolveStage`); `data/intake/disciplines.json`;
`data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml`; the T01 and T05 maps and reviews; and
`testing/synthetic/reference/OWNER_RULINGS.md`.

**Checked by the reviewer against a public source:** the secondary-dwelling CDC standards sit in SEPP
(Housing) 2021 Schedule 1, with a 450 m² minimum lot (NSW Planning); the two neighbour notices
(Planning Portal); no Section 73 for a CDC secondary dwelling outside Priority Sewerage or pressure sewer
areas (Sydney Water); the long service levy at $250,000 or more incl. GST, 0.25% (Long Service
Corporation; figure awaiting the owner's ruling). URLs are in §5.

**Instruments named from professional knowledge, every clause and figure `verified: false`:** SEPP
(Housing) 2021, Chapter 3 Part 1 (secondary dwellings, 60 m² cap, no subdivision) and Schedule 1
(secondary dwelling CDC standards: lot width, setbacks, height, landscaped area, private open space,
exclusions); SEPP (Exempt and Complying Development Codes) 2008 (the principal dwelling's standards);
EP&A Act 1979 (s10.7, Part 6, s7.11, s7.12) and EP&A Regulation 2021 (neighbour
notice, critical-stage inspections per R7); Building and Development Certifiers Act 2018; Home Building
Act 1989 (HBCF per R4); Building and Construction Industry Long Service Payments Act 1986; WHS
Regulation 2017 (NSW) asbestos; Access to Neighbouring Land Act 2000; NCC 2022 Volume Two and the ABCB
Housing Provisions (Parts 9.2, 9.5, 11.3); AS 2870, AS 1684, AS 3660.1, AS 3740, AS 3786, AS 3959,
AS 4055, AS 4970, AS/NZS 3000, AS/NZS 3500.

**Claims I am least sure of:**

- The Schedule 1 lot width (12 m) and setbacks (900 mm side, 3 m rear).
- Which site conditions exclude a secondary-dwelling CDC (heritage conservation area, flood control lot,
  BAL-40/FZ).
- The fire-separation distances (900 mm to a boundary, 1.8 m between buildings), until the Housing
  Provisions Part 9.2 text is checked.

## Review applied (round 1, 30 October 2026)

All eleven findings in `REVIEW.md` were applied to this file and `shape.yaml`; none is disputed. The
reply is in REVIEW.md, "Round 1 response".

## Disputed review findings

None.
