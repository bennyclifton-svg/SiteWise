# T02 Multi-residential apartments, 4–8 storeys: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (e1a5389 plus the working tree) and professional practice.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. Clause numbers I could not check
against the instrument are written `verified: false`. No prices, rates or week durations appear.

**How the app predictions were checked.** I ran the real Go with no file written in the repo
(`go test -overlay` with a scratch test file): `rfp.Resolve` for 18 templates against this profile,
and `program.Generate` for the 62-item default scope and for the expected scope. Statements below such
as "the app prices it as an option" come from those runs.

---

## 1. The job

**What it is.** A new residential flat building (NCC Class 2) of six storeys over two basement levels
of car parking (Class 7a). It sits on an amalgamated infill site in Greater Sydney. The base scenario:

| Item | Base value |
|---|---|
| Site | About 2,400 m², three or four amalgamated residential lots in an R4 High Density zone. Each lot has a 1960s dwelling to demolish. Sewer and water in the street, overhead power, footpath and kerb |
| Building | 6 storeys, about 64 apartments (1, 2 and 3 bedrooms), about 6,200 m² GFA, two lifts, one or two fire-isolated stairs, rooftop or ground communal open space, deep soil, bin room |
| Basement | Two levels, about 70 car spaces, piled or shotcrete retention with temporary anchors, drained or tanked walls. Excavation close to neighbours |
| Construction | Reinforced or post-tensioned concrete frame and cores (Type A construction), masonry, precast or FC facade, aluminium windows, balcony and podium membranes |
| Services | Sprinklers throughout, hydrants and booster, detection and occupant warning, car park ventilation, split-system AC, heat-pump hot water, a DNSP substation, NBN, intercom and access control, EV-ready car park. All-electric (gas is a trigger) |
| Pathway | DA to council, determined by the local planning panel. Then a construction certificate and an occupation certificate, then strata subdivision |
| Procurement | Design and construct (AS 4902). The developer's design team takes the scheme to DA and a D&C tender set. The architect and engineers are then novated to the builder |
| Value | Inside the PROJECT_TYPES range of $10–60M. The shape names no amount |

**Who procures it and how.** A developer (often a single-purpose company with a financier) buys and
amalgamates the lots. It engages a development or project manager, the architect and a town planner
first, then the specialist consultants for the DA. After the DA it tenders a D&C contract to a few
mid-tier builders, novates the design team, and keeps the certifier, QS, planner and surveyor on its
own side. Units are often sold off the plan, so the strata plan and the occupation certificate decide
settlement. [D]

**What makes it go wrong.** [D]

1. **Yield lost at DA.** The ADG (building separation, solar access, cross ventilation, deep soil),
   the height and FSR standards and design review panel comments cut apartments. A clause 4.6
   variation, panel deferral or Land and Environment Court appeal adds months.
2. **Basement risk.** Groundwater, rock quality, anchors under neighbours who refuse consent,
   neighbour damage claims, a sewer main in the excavation, and contaminated fill.
3. **Novation gaps.** The design is incomplete at tender, so the builder prices risk and value-manages
   the facade, waterproofing and services. Who owns which design after novation is unclear.
4. **Defects.** Waterproofing (balconies, podium, basement), fire safety (fire stopping, fire doors,
   combustible cladding), facade leaks and acoustic failures. NSW now polices these through the
   Building Commissioner (RAB Act orders), the DBP Act declarations and the strata building bond.
   (verified: false)
5. **Authority long lead.** The DNSP substation, the Sydney Water Section 73 certificate and lift
   delivery hold up the OC. [K: knowledge/program/fragments.yaml fr.utility-major, fr.lifts,
   fr.services-connections-major]
6. **OC and strata timing.** A missing fire safety certificate, BASIX completion receipt, s73
   certificate or the RAB Act expected-completion notice delays the OC, and with it the strata plan
   and settlements. (verified: false)
7. **Contractor insolvency.** Mid-tier builder failure during the structure stage.

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. The app's apartments/new default list (62
systems, action `new`) is `[K: knowledge/profile/scope_defaults.yaml classes[5]]`. Items marked **+**
are ones a competent PM expects that the default list lacks.

### `site`

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | new | Set-out, level control, identification surveys. In default list. |
| `site.ground-conditions` **+** | investigate | Cored boreholes and groundwater standpipes for the basement design. With this item `fr.investigate-geotech` fires. [K: knowledge/program/fragments.yaml fr.investigate-geotech] |
| `site.demolition` **+** | remove | Demolish the existing dwellings after hazmat removal. `fr.demolition-2` fires only with action `remove`. [K: knowledge/program/fragments.yaml fr.demolition-2] |
| `site.earthworks` | new | Bulk and rock excavation, spoil classification and disposal. In default list. |
| `site.erosion-sediment-control` | new | Standard DA condition. In default list. |
| `site.stormwater-detention` | new | OSD tank, water-quality treatment. In default list. |
| `site.services-connections` | new | Water, sewer, fire service, NBN, power lead-ins. In default list. |
| `site.vehicle-access` | new | Driveway, ramp, AS 2890.1 car park layout. In default list (no fragment). |
| `site.waste-storage` | new | Bin room, bulky goods room, collection point. In default list (no fragment). |
| `site.public-domain` | new | Footpath, kerb, crossing, street trees under s138 (verified: false). In default list (no fragment). |
| `site.adjoining-structures` | new | Dilapidation, support and protection of neighbours. In default list. |
| `site.landscape` | new | Deep soil, communal open space, podium planting. In default list. |
| `site.trees` | retain (trigger) | Street or neighbour trees. Out of base. |
| `site.contamination`, `site.acid-sulfate-soils`, `site.flooding` | investigate / new (trigger) | Site conditions, see §9. |
| `site.retaining-walls`, `site.pools`, `site.signage`, `site.impact-protection`, `site.loading-docks` | out | Not in a typical scheme. Signage is the street number only. |

### `substructure`

| System | Action | Why |
|---|---|---|
| `substructure.basement-retention` | new | Retention piles or shotcrete, anchors or props. In default list; `fr.basement-2` fires. |
| `substructure.footings` | new | Pads or piers at basement level. In default list. |
| `substructure.damp-waterproofing` | new | Drained cavity or tanked walls, under-slab drainage. In default list, but no fragment. |
| `substructure.underpinning` | new (trigger) | Shallow neighbour footings. `fr.underpinning-2` exists. |
| `substructure.piles` | out | Bearing piles only on poor ground. Retention piles belong to basement retention. |
| `substructure.slab-on-ground`, `substructure.retaining-walls`, `substructure.termite-management` | out | The basement slab is inside `fr.basement-2`. Termite management does not apply to a concrete basement building. [D] |

### `structure`

| System | Action | Why |
|---|---|---|
| `structure.concrete` | new | Slabs, columns, blade walls, cores. In default list; `fr.structure-concrete-2` gives a floor cycle. |
| `structure.precast` | out (variant) | Precast walls or facade panels are common but not universal. |
| `structure.design-basis`, `structure.temporary-works` | out as work items | Consultant input; the contractor's temporary works. |
| `structure.steel`, `structure.timber`, `structure.masonry`, `structure.alterations` | out | Minor steel only. No existing structure. |

### `envelope`

| System | Action | Why |
|---|---|---|
| `envelope.external-walls` | new | Facade. In default list. |
| `envelope.windows-doors` | new | Long lead. In default list. |
| `envelope.external-waterproofing` | new | Balconies, podium, planters, roof. In default list. `fr.balcony-waterproof-2` fires. |
| `envelope.flashings-seals` **+** | new | Sill and threshold flashings; a top Class 2 defect. [D] |
| `envelope.thermal-insulation`, `envelope.thermal-performance` | new | NatHERS and BASIX. In default list. [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required] |
| `envelope.facade-acoustics` | new | Glazing ratings from the acoustic report. In default list. |
| `envelope.roof-coverings` | out | A concrete roof with a membrane is covered by external waterproofing. |
| `envelope.curtain-wall`, `envelope.external-shading`, `envelope.pliable-membranes`, `envelope.building-sealing`, `envelope.vehicle-doors`, `envelope.bushfire-construction` | out | Not base. The car park roller door is minor. |

### `interiors`

| System | Action | Why |
|---|---|---|
| `interiors.walls-linings`, `ceilings`, `wet-area-waterproofing`, `tiling`, `floor-finishes`, `joinery`, `internal-doors`, `internal-glazing`, `painting` | new | Apartment fit-out. In default list. |
| `interiors.acoustic-separation` | new | SOU walls and floors. [K: knowledge/clusters/envelope/rules.yaml rule.ncc.sou-wall-sound-insulation] (clause verified: false) |
| `interiors.ffe` **+** | new | Appliances as PC items. |
| `interiors.room-acoustics` | out | Not for residential. |

### `fire-passive`

| System | Action | Why |
|---|---|---|
| `fire-passive.rated-elements` | new | Type A, Table C2D2, Class 2 at rise 4+. [K: knowledge/tables/type_of_construction.yaml] |
| `fire-passive.compartments` | new | Separation of the 7a car park from the Class 2 part; fire-isolated stairs. |
| `fire-passive.penetrations` | new | `fr.fire-stopping` and its hold point fire. [K: knowledge/clusters/fire/interfaces.yaml if.fire-stopping-before-concealment] |
| `fire-passive.openings` **+** | new | Fire-rated SOU entry doors with self-closers. [D] |
| `fire-passive.external-walls` **+** | new | Combustibility evidence for the facade. [K: knowledge/clusters/fire/rules.yaml rule.ncc.external-wall-fire] |
| `fire-passive.structural-fire-protection`, `linings-fire-hazard` | out | Concrete frame; linings are part of the finishes. |

### `fire-active`

| System | Action | Why |
|---|---|---|
| `fire-active.sprinklers` **+** | new | NCC 2022 E1D6: sprinklers throughout a Class 2 building with a rise in storeys of 4 or more and an effective height of not more than 25 m. E1D4(2): the system follows Specification 17 and 18 as applicable (verified: true, ABCB NCC 2022 Part E1, per the review). E1D9 also covers a Class 7a car park for more than 40 vehicles (verified: false). The Spec 18 concessions are not available with a Class 7 part (verified: false). Not in the default list, so `fr.sprinklers-2` and the pressure-test hold point never fire by default. [D] |
| `fire-active.fire-water` **+** | new | Fire service connection, booster, usually a pump set. No fragment. |
| `fire-active.hydrants`, `detection`, `smoke-alarms`, `emergency-lighting`, `portable-extinguishers` | new | In default list. |
| `fire-active.smoke-control` | out | Stair pressurisation applies above 25 m effective height (verified: false). Six storeys is below it. |
| `fire-active.occupant-warning`, `fire-control` | within detection | Occupant warning and the fire panel are part of the detection system here. |
| `fire-active.special-suppression` | out | None. |

### `mechanical`

`air-conditioning`, `ventilation`, `local-exhaust`, `carpark-ventilation`: new, all in the default
list. Split systems, corridor ventilation, bathroom and garbage room exhaust, car park ventilation to
AS 1668.2. Central plant, BMS, cooling towers, fire-mode systems, heating appliances and process
systems are out. Refrigerant and condensate are part of the AC.

### `hydraulic`

`cold-water`, `hot-water`, `sanitary`, `roof-drainage`: new, in the default list. **+**
`backflow-prevention` (boundary containment for a multi-storey with a fire service), **+**
`non-potable-water` (BASIX rainwater reuse). `gas` is a trigger. Trade waste and process systems are
out. [K: knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate]

### `electrical`

`utility-supply` (DNSP substation, long lead), `switchboards`, `cabling-wiring`, `lighting`,
`metering`: new, in the default list. **+** `ev-charging` (EV-ready circuits under the council DCP or
NCC J9; verified: false). **+** `temporary-power`. Solar PV is a BASIX option (variant). Standby,
battery and process systems are out.

### `comms-security`

`carrier-lead-in`, `structured-cabling`, `access-control`: new, in the default list. **+** `cctv`
(lobby and car park). Intruder alarm, AV and active networks are out.

### `vertical-transport`

`passenger-lifts`, `lift-shafts-pits`: new, in the default list (`fr.lifts`, `fr.lift-shafts-2`).
Car lifts are out (a narrow-site variant). Lift fire-service operation is part of the lift.

### `access-egress`

`exits`, `stairs-and-ramps`, `barriers`, `egress-doors`, `accessible-access`, `livable-housing`: new,
in the default list. **+** `maintenance-access` (roof plant ladders and anchors). Accessible
facilities and pool barriers are out.

---

## 3. Existing-building interfaces

No building is retained (`det.existing_building: false`), but the site is not clear:

- **Demolition** of three or four 1960s dwellings, sheds and pools. A hazardous materials survey is
  needed before demolition. Licensed asbestos removal is notified to SafeWork NSW, followed by a
  clearance certificate. Setting `det.existing_building_year` before 2004 with a `remove` item raises
  the hazmat implication. [K: knowledge/clusters/structure/consequences.yaml pre-2004] [D]
- **Services cut-overs.** Disconnect and seal each lot's water, sewer junctions, power and gas before
  demolition, and abolish the old Sydney Water connections. [D]
- **Neighbours.** Dilapidation surveys of adjoining dwellings and the road, excavation support,
  anchors under their land (consent), possible underpinning, vibration monitoring, crane oversail
  consent. [D]
- **Matching existing work.** Not applicable.
- **Occupation.** The site is vacant. The neighbours are occupied, so the works need noise and dust
  limits and controlled hours. [D]

---

## 4. Consultants and why

App suggestion for this profile (15): Access, Architect, BCA, Certifier, Civil, Electrical, Fire
Services, Geotechnical, Hydraulic, Mechanical, Structural, Surveyor, Town Planner, Traffic and
Vertical Transport. [K: knowledge/works/package_defaults.yaml baselines[0], baselines[1],
scope_additions[0]–[5], complexity_additions[0]] Traffic, BCA, Mechanical, Fire Services and Access
come through `baselines[1]` only because `7a` maps to the industrial category. Without basement
parking, a Class 2 building would lose BCA and Traffic.

| Discipline id | Engaged at | Need | Novated? | App |
|---|---|---|---|---|
| `consultant.architect` | Feasibility | Always | Yes | yes |
| `consultant.town_planner` | Feasibility, pre-DA | Always | No | yes |
| `consultant.surveyor` | Before concept; strata at completion | Always | No | yes |
| `consultant.geotechnical` | Before concept design | Always | No (reliance to the builder) | yes |
| `consultant.structural` | Concept | Always | Yes | yes |
| `consultant.civil` | Concept, DA | Always | Yes | yes |
| `consultant.hydraulic`, `electrical`, `mechanical`, `fire_services` | Concept, DA | Always | Option (CONVENTIONS C1) | yes |
| `consultant.bca` | Concept | Always | Option | yes (7a mapping) |
| `consultant.certifier` | Before CC, often at DA | Always | Never | yes |
| `consultant.access` | DA | Always | No | yes |
| `consultant.traffic` | Concept, DA | Always | No | yes (7a mapping) |
| `consultant.acoustic` | DA | Always | Option | **no** |
| `consultant.basix` (often an ESD firm) | DA | Always | Option | **no** |
| `consultant.waste` | DA | Always | No | **no** (template exists) |
| `consultant.landscape` | Concept, DA | Always | Yes | **no** (template exists) |
| `consultant.quantity_surveyor` | Feasibility | Always | No | **no** (template exists) |
| `consultant.fire_engineer` | Design development | Usually | Option | **no** (template exists) |
| `consultant.hazmat` | Investigation | Always (base demolition) | No | **no** |
| `consultant.vertical_transport` | Design development | Optional | No | yes (generic RFP) |
| `trade.main_works` | Procurement | Always (D&C) | — | partial: only from gaps on accepted items |

Conditional (triggers): heritage, arborist, environmental (contamination) and a facade engineer.
Not keyed: dilapidation surveyor, Water Servicing Coordinator, ASP Level 3 designer, strata inspector,
waterproofing consultant. None has a discipline id. [D]

**Novation.** The app gives every consultant novation `option` on a D&C route. That includes the
certifier, planner, QS, surveyor and traffic engineer, who are never novated. [K:
internal/procurement/packages.go DefaultNovation] Following `reference/CONVENTIONS.md` C1
(provisional), the expected values are: `yes` for the architect, structural, civil and landscape;
`option` for the services engineers (mechanical, electrical, hydraulic, fire services), because
builders often replace them with their own D&C services trades; `no` for the client-side advisers.
So the app's `option` is right for the services engineers and wrong for the rest. Following C2, the
BASIX/NatHERS package is `consultant.basix`, because the work is residential only.

---

## 5. Approvals and authorities

**Pathway.** DA under the EP&A Act. Housing SEPP Chapter 4 (formerly SEPP 65) and the Apartment Design
Guide apply to a building of 3+ storeys with 4+ dwellings (verified: false). A design verification
statement by a registered architect goes with the DA (EP&A Reg 2021 s29, verified: false). Where the
council has constituted a design review panel (Housing SEPP s145, verified: false), the panel reviews
the scheme before and after lodgement. That is common in Greater Sydney but not universal, and no
numeric threshold applies. The Local Planning Panels Direction sends Chapter 4 development of 4 or
more storeys to the local planning panel. If the CIV makes the application regionally significant
(threshold about $30 m, verified: false), the Sydney district planning panel determines it instead. A
scenario must keep the named panel consistent with its stated CIV; `flag.panel` is yes either way.
CDC is not available for this type. [D]

**Certificates.** Staged construction certificates: an early-works CC for excavation, retention and
the basement structure, then a main-works CC. Demolition proceeds under the DA conditions without a CC
(verified: false). The app has one CC fragment, so this stays a known difference;
principal certifier appointment and notice of commencement; occupation certificate (often interim,
then final); fire safety certificate; BASIX completion receipt [K: knowledge/clusters/envelope/rules.yaml
rule.basix.completion-before-occupation]; Section 73 compliance certificate [K:
knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate]; strata certificate,
then strata plan registration at NSW LRS. [D]

**Class 2 specific (owner ruling R3 and [D], verified: false).**
- DBP Act: regulated designs with design compliance declarations by registered design practitioners,
  and the builder's principal compliance declaration before the OC application. Lodged on the NSW
  Planning Portal.
- RAB Act: the developer gives Building Commission NSW an expected-completion notice, through the
  Planning Portal, at least 6 and not more than 12 months before applying for the OC (confirmed on
  nsw.gov.au; section number verified: false). Without it, a prohibition order can hold the OC and the
  strata plan.
- Strata Building Bond and Inspections Scheme: 2% of the contract price (3% deferred to 1 July 2028),
  lodged before the OC is issued. The interim report falls 15–18 months and the final report 21–24
  months after completion (nsw.gov.au). The bond is a refundable security. Scope for 4+ storeys depends
  on the HBCF exemption (R4, verified: false). The developer appoints the strata inspector.
- Home warranty (HBCF) does not apply above three storeys. Owner ruling R4 keeps that exemption
  `verified: false`.

**Authority applications.** The Section 73 application goes through a Water Servicing Coordinator
soon after the DA determination; the s73 certificate is an OC and strata prerequisite. Sydney Water
building plan approval is the item the CC needs. A pressure enquiry is made for the fire service. The
DNSP (Ausgrid or Endeavour) connection enquiry starts in design development, because the substation
and its easement must be on the DA plans. The substation is designed by an ASP Level 3 and installed by an ASP
Level 1, substation easement or lease. NBN developer agreement. Council: s138 road works and
crossing, work zone, hoarding, crane, CTMP, street tree removal. Long service levy and contributions
(s7.11 or s7.12, Housing and Productivity Contribution) before the CC. Lift registration. [D]

**Inspections and hold points.** For Class 2, 3 and 4 the certifier's critical stages are: (1) after
excavation and before the first footing; (2) before covering waterproofing in wet areas, for at least
10% of rooms with wet areas; (3) before covering stormwater drainage connections; (4) a final
inspection before the OC (cl 61 of the 2021 Regulation, verified: false; the project's CC lists them).
There is no certifier frame or slab-pour inspection; the structural engineer inspects reinforcement.
Fire stopping is not on the confirmed list. The app's fire-stopping hold point is a quality hold
point, not a critical stage. The sprinkler pressure-test
hold point fires only when sprinklers are in scope. The reinforcement hold point names
`substructure.slab-on-ground`, so it never fires for apartments. [K:
knowledge/clusters/fire/interfaces.yaml; knowledge/clusters/structure/interfaces.yaml
if.reinforcement-inspected-before-pour]

---

## 6. Cost plan shape

App lines for this profile: F01–F15 (one per adopted consultant), construction 01–13 (all 13
elements; access-egress children 13.1, 13.3, 13.4, 13.9, and 13.7 if maintenance access is added),
and P01–P07 including DA fees. [K: knowledge/costs/elements.yaml; knowledge/costs/starter_lines.yaml]

- **Fees and charges:** DA fees and contributions (P04); Housing and Productivity Contribution; DRP
  fees; CC and certifier fees (P05); long service levy; service authority fees (P06, one generic
  line); Sydney Water s73 and developer charges; DNSP fees; council road-reserve fees and bonds
  (s138, work zone, hoarding, crane); strata building bond (a refundable security, carried as a
  cash-flow item, not a sunk fee); strata plan registration.
- **Consultants:** the 15 app fee lines, plus acoustic, BASIX, waste, landscape, QS, fire engineer,
  hazmat, strata inspector and dilapidation. The pre-novation fee stages stay here. **The post-novation stages
  move into the D&C contract sum.** A cost plan that keeps both double-counts them. [D]
- **Construction:** preliminaries, D&C margin and design management (no app line); the 13 elements;
  post-novation design fees inside the D&C sum; provisional sums (rock excavation, contaminated spoil)
  and PC items (appliances).
- **Contingency:** design, construction, escalation, exclusions (land, finance, marketing and sales
  are outside the construction plan).

Budget split: with `hdr.scale.storeys: 6` the app picks `residential_apartments_high_rise` (storeys
≥ 5). Its shares already assume a basement (its basis is two apartment estimates over basements), so
no multiplier is needed here. The gap runs the other way: the high-rise split has no no-basement case,
while the low-rise split doubles substructure only when `flag.basement` is yes. A 4-storey and a
5-storey block over the same basement therefore get different substructure shares. [K:
knowledge/costs/budget_splits.yaml]

---

## 7. Program shape

Skeleton `sk.new-dc-da`: investigation → concept → approval documentation → approvals → procurement →
construction documentation → construction → completion → defects. [K:
knowledge/program/skeletons.yaml]

**Driving path.** Survey and geotech → yield and ADG study → concept → pre-DA and DRP (where
constituted) → design development (with the DNSP enquiry and substation location) → DA documentation
(SEE, design verification statement, BASIX, acoustic, traffic, waste, landscape, stormwater, access,
BCA, QS cost report) → lodgement, exhibition, DRP, assessment → **panel determination** (with the s73
application lodged straight after) → D&C tender → award → **novation**.

From novation two streams run in parallel:

- **Site:** site establishment, then asbestos removal and demolition under the DA conditions, once
  the dilapidation surveys are done. No CC is needed.
- **Documents:** early-works documentation and regulated designs → Sydney Water building plan
  approval, LSL and contributions → **early-works CC** → notice of commencement → **retention and bulk
  excavation** → basement slab.

The main-works documentation and regulated designs lead to the **main-works CC**, which gates the
**floor cycle**. Then: facade and windows → waterproofing → services, sprinklers, fire stopping →
linings, wet areas, joinery, finishes → lifts, substation → commissioning and essential services →
PC → builder's principal compliance declaration, fire safety certificate, BASIX, s73 certificate,
strata bond, and the RAB Act notice given 6–12 months earlier → **OC** → strata plan registration →
handover → DLP and the strata bond reports.

**Logic the app gets wrong** (verified by run): there is one CC and no staging. The CC fragment sits in approvals, ahead of
procurement and post-novation documentation. The tender fragments are traditional. "Client sign-off
for tender" lands after award. Windows are linked after the roof slab instead of from award. An odd
chain link runs stormwater → fire stopping. Long lead: the substation, lifts, windows, switchboard,
joinery and AC plant.

---

## 8. RFP shape

Expected states are in `shape.yaml`. The main points:

- **Architect (`architecture`).** Concept to DA, DVS and DRP responses, design development, D&C
  tender set with a performance brief and design responsibility schedule, post-novation construction
  documentation, regulated designs and declarations. It must not carry house documents, traditional
  bidding, contract administration (which conflicts with novation) or tenant fit-out. The app prices
  regulated design and post-novation as **options**, because the D&C boundary ends at `dnc_tender`.
  It keeps `apartment_design` at O, and adds `mixed_use_map` because classes 2 and 7a sit in one part.
- **Structural.** Retention, anchors, temporary-works interfaces, delegated PT and pile design,
  regulated designs, post-novation. Not existing-structure assessment, super-flat floor or temporary
  works design. Same option problem as the architect.
- **Civil.** OSD, MUSIC, public road works for s138, D&C tender, post-novation, declarations,
  WAE/OSD certification. Not shoring (structural's), NorBE or flood (base).
- **Geotechnical.** Cored boreholes, groundwater, retention parameters, DA report, reliance for the
  builder. Most rows **ask** (client flags cannot be keyed).
- **Surveyor.** Detail, boundary, neighbours, services, **strata plan**. The app keeps titling at O.
- **Certifier / BCA (`bca-certification`).** Certifier: Package B (CC, PC, inspections, OC, fire
  referrals). BCA: Package A (schematic, DA, tender advice, classification map). Without the client
  flags the app asks about Package A and prices Package B as options for both.
- **Town planner, traffic.** The app matches the expectation (panel attendance included, scale
  complex for the planner; car park certification and CTMP as options).
- **Hydraulic, electrical, mechanical, fire services.** D&C tender documentation should be included.
  The templates have no D&C logic, so the app prices it as an option, and declarations ask.
  Post-novation continuation is expected as a priced option (novation `option`, CONVENTIONS C1),
  which matches the app. Fire services must carry car park sprinklers to AS 2118.1.
- **Landscape, waste, QS, fire engineer, access.** Templates exist, but no package rule creates the
  package. Most stages ask through client flags.
- **Acoustic, BASIX/ESD, hazmat.** No template. **Vertical transport** falls to the generic
  `tpl.rfp-capex`.

---

## 9. Triggers

| id | How it shows up (incl. indirectly) | Denied by | Adds |
|---|---|---|---|
| `traditional-procurement` | "full documentation, lump-sum tender", "consultants stay with the developer" | "D&C with novation" | Traditional skeleton; novation `no` on every package; tender_traditional and for_construction expected in, but the app prices them as options (non-house boundary ends at `interiors`); D&C stages drop out |
| `groundwater-dewatering` | "groundwater at RL…", "tanked basement", "dewatering" | "no groundwater encountered" | Dewatering approval, monitoring, tanking cost; civil earthworks suitability |
| `cl46-variation` | "clause 4.6", "lift overrun exceeds the height control" | "complies with height and FSR" | Variation request stage |
| `heritage-conservation-area` | "adjoins a Schedule 5 item", "conservation area" | "not a heritage item or in a conservation area" | Heritage consultant, HIS, referral |
| `flood-affected` | "flood related development controls", "entry crest above PMF" | planning certificate says not flood affected | site.flooding, flood report, flood barriers |
| `contaminated-land` | "former service station", "ash and slag fill", "PSI required" | "continuous residential use" | Environmental consultant, PSI/DSI/RAP, site auditor |
| `acid-sulfate-soils` | "Class 2 on the ASS map" | "not mapped" | ASS assessment and management plan |
| `trees-affected` | "basement wall in the TPZ of street trees", "neighbour's fig overhangs" | "no trees affected" | Arborist, tree protection, root mapping |
| `sydney-water-asset` | "sewer main crosses the rear of the lots", "zone of influence" | "no Sydney Water assets" | Authority asset assessment, diversion |
| `rail-corridor` | "adjoins the Sydney Trains corridor" | "not near a rail corridor" | TfNSW concurrence; rail noise and vibration scope in the (base) acoustic RFP; geotech asset exposure; monitoring |
| `transfer-structure` | "transfer slab at ground floor" | "columns run straight down" | Transfer slab cost and program (zero RFP delta) |
| `noise-affected-site` | "20,000 vehicles a day", "20 ANEF" | "quiet local street" | Acoustic glazing, ventilation (zero app delta) |
| `underpinning-neighbour` | "neighbour's footings are shallow", "underpin the party wall" | "basement set back" | Underpinning item, `fr.underpinning-2`. No profile key changes: base already sets `excavation_adjoining` |
| `gas-reticulation` | "gas cooktops", "Jemena meter room" | "all-electric" | hydraulic.gas, `fr.gas-commercial`, Jemena |
| `two-buildings-shared-basement` | "Building A and Building B" | "a single building" | Masterplan stage, staged OC, management statement |
| `complex-facade` | "ventilated terracotta rainscreen", "designed by a specialist facade engineer" | "face brick and rendered masonry" | Facade consultant (`complexity_additions[7]`), fee line; no RFP template |
| `ssd-affordable-housing` | "in-fill affordable housing bonus under the Housing SEPP", "SEARs requested" | "DA lodged with council, determined by the local planning panel" | `sk.ssd-dc`, `fr.ssd-eis`, `fr.ssd-assessment`; DA steps and the DA-fees line drop out; town-planning complex. Thresholds verified: false. Trigger, not variant |

Variants: groundwater, cl 4.6, trees, Sydney Water asset, contamination, traditional, complex facade,
heritage. `noise-affected-site` stays a trigger but not a variant, because it has zero app delta.
`trees-affected`: `fr.tree-protection` exists, but a `retain` item brings no fragment, and the arborist
RFP exists only if the user creates the package.

---

## 10. Keyword traps and negations

**Base-scenario phrases that trip regex triggers** (review finding 3). Scenario authors: state the base
value beside each phrase, as `shape.yaml scenario_inputs` shows.

| Base phrase | Pattern hit | Wrong reading | Expected reading |
|---|---|---|---|
| "the existing building on the site will be demolished" | `\bexisting building\b` (`knowledge/determinants.yaml existing_building`). Note that the plural "existing buildings" does not match | `det.existing_building` true → RFP `existing_building` module and existing-assessment stages | false: nothing is retained |
| "demolition of the four existing buildings"; "building a minimum 6 m from the boundary" | `(?:two\|three\|four\|five\|\d+) buildings`, `\bbuildings? [a-d]\b` (`flag_readings.yaml multi_building`) | `flag.multi_building` yes → `brief_masterplan` | no |
| "Stage 1 CC (excavation and shoring)" | `\bstages? [1-9a-c]\b` (`staged`) | `flag.staged` yes → staged occupation, masterplan | no: CC stages are not construction stages |
| "connect to the water main / sewer main in the street" | `(?:sewer\|water) (?:main\|asset)` (`authority_asset`) | `flag.authority_asset` yes → structural `authority_asset` stage | no |
| "road traffic noise assessment" in the base acoustic report | `\b(road\|rail\|aircraft\|traffic) noise\b` (`noise_affected_site`) | `det.noise_affected_site` true | false unless the report finds a noise-affected site |
| "new street trees" in the public-domain scope | `arborist_offsite_trees` pattern | `flag.arborist_offsite_trees` yes | no: new planting affects no existing tree |

**Other traps:**

- **"Basement" in an existing-dwelling description** ("the existing house has a basement garage to be
  demolished") sets `flag.basement` for the wrong reason. Here the outcome happens to be right. [K:
  knowledge/profile/flag_readings.yaml basement]
- **"Public exhibition"** in the program or SEE matches `flag.community` (consultation required). It is
  only statutory notification. [K: knowledge/profile/flag_readings.yaml community]
- **"Lodge"** matches `architect_lodges`. Here the town planner lodges.
- **"Planning panel"** in a council letter about another site, or "not referred to a planning panel".
- **"Party wall"** or **"common wall"** triggers `implied_by` party-wall (adjoining structures,
  compartments, acoustic). That is right here, but "no party walls" also adds them, because the
  pattern has no negation. [K: knowledge/profile/scope_defaults.yaml implied_by party-wall]
- **"Demolition"** adds `site.demolition` with the work-type default `new`, not `remove`, so
  `fr.demolition-2` does not fire. [K: knowledge/works/actions.yaml work_type_defaults]
- **"BASIX"** adds `envelope.thermal-performance` (already in scope).
- **"Strata"** in the developer's brief ("strata subdivision on completion") is not
  `stakeholder_complexity: strata`, which means an existing owners corporation.
- **"Car park ventilation"** versus "natural ventilation of the car park": both name the system; the
  second means no fans.
- **"Sprinkler"** in "the building is not required to be sprinklered" (a three-storey example).
- **"Novation"** in "the consultants will not be novated" (traditional), and "D&C" in a builder's
  marketing letter for a different job.
- **Class confusion:** "Class 7a car park" against "Class 7b"; "Class 2" against "Class 2 on the
  acid sulfate soils map".
- **"Section 73"** in a Roads Act or other context; **"s138"** of the Roads Act against the EP&A
  Regulation.

---

## 11. Gaps in the built-in knowledge

Full list in `shape.yaml gaps`. The ones that change the app's answer most:

1. **Procurement misses seven always-needed consultants** (acoustic, BASIX/ESD, waste, landscape, QS,
   fire engineer, hazmat). BCA and Traffic arrive only through the 7a mapping.
2. **Novation is wrong everywhere:** `option` for all, where it should be yes for the design team and
   no for the client-side advisers.
3. **The scope defaults omit sprinklers**, the fire pump and tank, demolition, ground investigation,
   fire doors and external-wall fire performance. The sprinkler fragment and the pressure-test hold
   point therefore never fire by default.
4. **The D&C program has one unstaged CC, misordered** (the expected program uses early-works and
   main-works CCs, with demolition under the DA before either). It has no Class 2 statutory steps (DBP Act, RAB Act notice,
   strata bond and plan, s73, DNSP, LSL and contributions, DRP, panel).
5. **D&C RFPs price the core novated work as options** (architecture and structural `post_novation`
   and `regulated_design`). The services templates have no D&C logic at all.
6. **The cost plan has no preliminaries or authority-specific lines**, and does not deal with
   post-novation fees moving into the D&C sum.
7. **`mixed_use` is derived from 2 plus 7a**, which adds mixed-use RFP rows.
8. **Base wording trips six regex triggers** (§10), so a profile built from ordinary T02 documents can
   disagree with the base unless the documents state the base value.
9. **No knowledge** of the NCC sprinkler requirement for Class 2, the DBP Act, the RAB Act, the strata
   bond or the ADG.

---

## 12. Sources and unverified claims

Opened in the repo: `knowledge/profile/scope_defaults.yaml` (classes[5]), `knowledge/works/package_defaults.yaml`,
`knowledge/costs/starter_lines.yaml`, `knowledge/costs/elements.yaml`, `knowledge/costs/budget_splits.yaml`,
`knowledge/program/skeletons.yaml`, `knowledge/program/fragments.yaml`, `knowledge/clusters/fire/interfaces.yaml`,
`knowledge/clusters/structure/interfaces.yaml`, `knowledge/clusters/fire/rules.yaml`,
`knowledge/clusters/envelope/rules.yaml`, `knowledge/clusters/services-wet-air/rules.yaml`,
`knowledge/tables/type_of_construction.yaml`, `knowledge/profile/flag_readings.yaml`,
`knowledge/profile/taxonomy.yaml`, `knowledge/determinants.yaml`, all `knowledge/rfp/*.yaml` through
`reference/index/rfp-templates.md`, `internal/rfp/drivers.go`, `internal/program/generate.go`. Owner
rulings R3 and R4 applied.

Confirmed in review round 1 (sources in REVIEW.md §2):
- NCC 2022 E1D6 and E1D4(2), Class 2 sprinklers (verified: true);
- the RAB Act notice window and recipient;
- the strata bond rate, lodgement and inspection windows;
- Housing SEPP Chapter 4 scope and the LPP 4-storey trigger.

All `verified: false`:
- E1D9 (car park over 40 vehicles); the Spec 18 concessions with a Class 7 part;
- the 25 m effective-height threshold for stair pressurisation;
- the design verification statement clause (EP&A Reg s29); Housing SEPP s145; the regionally
  significant CIV threshold;
- DBP Act lodgement timing; RAB Act and SSMA section numbers; the HBCF exemption above three storeys
  (R4) and so the 4+ storey bond scope;
- Class 2 critical-stage inspections under cl 61; demolition under a DA without a CC;
- s7.11/s7.12, the Housing and Productivity Contribution, the long service levy threshold, Sydney
  Water developer charges;
- Roads Act s138, WaterNSW/NRAR dewatering approval, the Transport and Infrastructure SEPP rail and
  road-noise provisions, the Access to Neighbouring Land Act;
- NCC Part F7 numbering for sound insulation (the knowledge itself records it as unverified);
- EV-ready requirements (council DCP or NCC J9).
