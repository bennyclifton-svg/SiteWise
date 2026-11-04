# T07 Mechanical plant upgrade (chillers, AHUs): domain map

Draft for owner review, 30 October 2026. The machine key is `shape.yaml` in this folder; it passes
`validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Any statute, standard or NCC clause not confirmed
from the instrument is marked **verified: false**.

---

## 1. The job

**What it is.** The owner of an occupied NSW Class 5 office building replaces end-of-life central
cooling plant while tenants stay in. The usual package is:

- the chillers and their chilled-water pumps;
- the floor or central air-handling units (AHUs);
- the mechanical services switchboards;
- integration of the new plant into the building management system (BMS).

The usual driver is age (20 to 30 years), unreliable plant, and a refrigerant (R22) that can no longer be
bought to top up leaks [K: knowledge/clusters/services-wet-air/unforeseen.yaml
uc.refrigerant-phase-out-forces-plant-replacement] [D]. The second driver is energy: a better NABERS
rating, or tenant lease obligations [D].

**Base scenario** (used in `shape.yaml`):

- a 6-storey office of about 6,000 m² NLA, completed in 1988, one owner, fully tenanted;
- two air-cooled R22 screw chillers on the roof plant deck, replaced like for like in the same
  positions, with new variable-speed chilled-water pumps;
- six floor AHUs, one per floor, replaced in their plant rooms;
- new mechanical services switchboards, new submains to suit, and integration into the retained BMS;
- AHU fire mode is shutdown on smoke detection only, as listed on the building's fire safety schedule
  (effective height under 25 m, so no zone smoke control or stair pressurisation; see §5);
- chillers replaced **one at a time in the cooler months**, so the other chiller carries the load (N-1)
  and no temporary cooling is needed;
- each chiller lifted by mobile crane standing on the local street;
- exempt pathway, traditional procurement, one AS 4000 lump-sum contract with the **mechanical
  contractor as head contractor**, and a **mechanical engineer as lead consultant**.

Water-cooled plant with cooling towers, temporary cooling, asbestos removal, acoustic treatment, BMS
replacement and NABERS targets are the common variants and are triggers (§9).

**Base documents must also state** these negations, which have no app key:

- "Heat rejection is by air-cooled chillers; the building has no cooling towers or evaporative
  condensers."
- "The building is surrounded by commercial buildings; there are no noise-sensitive receivers near the
  roof plant."
- The staging and no-temporary-cooling sentence is in `scenario_inputs` (`flag.staged`).

**Scale.** $0.5–5 M [K: testing/synthetic/PROJECT_TYPES.md T07 row]. Cost is driven by plant capacity
and count, crane and access, the staging regime (out-of-hours work, temporary cooling) and the condition
of what is kept (pipework, valves, switchboards, BMS), not by floor area [D].

**Who procures it and how** [D]:

- **Client.** The owner: a private owner, a fund or REIT manager, or a government agency. The facilities
  or building manager is the owner's agent on site and controls shutdowns, tenant notices, after-hours
  access and the incumbent maintenance contractors (mechanical, BMS, fire). The building manager is not
  procured as a project consultant.
- **Lead consultant.** A mechanical services engineer (often a multi-services firm that also does the
  electrical supply). It does the following:
  - assesses the existing plant;
  - writes an options and replacement-strategy report with an order-of-cost estimate;
  - designs and specifies;
  - runs the tender;
  - reviews submittals and inspects;
  - witnesses commissioning and returns for the seasonal review.
- **Structural engineer** for the plant deck and support frames; **electrical engineer** for supply
  capacity; **hazmat consultant** for the plant-room survey; **BCA consultant** for a written pathway and
  fire-safety-measure advice.
- **Head contractor.** A mechanical services contractor under a lump-sum contract, often AS 4000 with
  special conditions. Its subcontractors are crane, rigging, electrical, structural steel, roofing, the
  BMS contractor (often the incumbent, sometimes nominated) and the building's fire services contractor.
  Some owners buy long-lead chillers directly and novate the supply contract to the contractor; that is
  not the base.

**What makes it go wrong** [D]:

- The replacement is sized from the original design, not measured loads, so it ends up oversized and
  inefficient, or undersized after decades of tenant changes.
- The new chiller does not fit the old footprint, plinths, clearances or lift route. This is found after
  ordering [K: knowledge/clusters/services-wet-air/unforeseen.yaml
  uc.new-plant-does-not-fit-old-plant-room-or-route; failure_modes.yaml fm.plant-lift-exceeds-crane-or-route].
- Isolation valves do not hold, so a "one-chiller" shutdown drains the whole chilled-water system
  [K: unforeseen.yaml uc.existing-plant-has-no-isolation-for-live-cutover].
- The shutdown window is shorter than the change-over [K: knowledge/clusters/delivery/unforeseen.yaml
  uc.agreed-shutdown-or-cutover-window-shorter-than-changeover]. The season slips, and plant is changed
  in summer without temporary cooling.
- The new chillers cannot talk to the old BMS [K: failure_modes.yaml fm.bms-protocol-incompatible-with-plant;
  unforeseen.yaml uc.pneumatic-controls-have-no-points-for-new-bms].
- Asbestos gaskets or lagging are found at disconnection and stop the job
  [K: unforeseen.yaml uc.asbestos-gaskets-and-rope-in-plant-being-disconnected].
- The new plant is louder or has a different noise spectrum, and the neighbours complain
  [K: unforeseen.yaml uc.plant-noise-to-neighbours-over-limit].
- The AHU fire-mode shutdown is not reconnected or tested, so the essential fire safety measure is out of
  service while tenants are in.
- Commissioning is done in winter and nobody proves the plant at summer peak.

---

## 2. Systems and work items

Action ids are the app's eight [K: knowledge/works/actions.yaml actions]. The refurb default is `alter`
[K: actions.yaml work_type_defaults]. The chillers and AHUs are **replaced**: like-for-like capacity is
`replace` by the app's own definition, and a stated capacity increase would be `upgrade`.

| system | in scope (base) | action | note |
|---|---|---|---|
| **site** | none | — | The crane stands on the road, but that is occupation, not works in the road reserve (`site.public-domain`). `site.demolition` is a **trap**: removing old plant is part of `replace`. It becomes real only for licensed asbestos removal (trigger `asbestos-in-plant-room`). |
| **substructure** | none | — | Not touched. A basement car park exists but is not in the works (see the basement trap, §10). |
| **structure** | `structure.alterations` | investigate | Capacity check of the roof plant deck, plinths and floor plant-room slabs for the new operating weights and footprints [K: knowledge/clusters/structure/systems.yaml structure.alterations "new loads on existing members"]. Strengthening is a trigger. |
| | `structure.steel` | new | New galvanised support frames on the plinths, with vibration isolation and seismic restraint [K: clusters/services-wet-air/interfaces.yaml if.plant-loads-structure]. |
| | `structure.temporary-works` | new | Crane set-up, outrigger pads and load spreading [K: clusters/structure/systems.yaml structure.temporary-works "crane bases, outrigger pads"]. Designed by the contractor's temporary works designer. |
| **envelope** | `envelope.external-waterproofing` | repair | The roof membrane is made good around altered plinths, frame fixings and pipe penetrations [K: interfaces.yaml if.roof-drainage-waterproofing]. Facade removal for AHU entry is a trigger. |
| **interiors** | none | — | Ceilings in plant-room corridors are lifted and reinstated locally by the contractor. No fit-out. |
| **fire-passive** | `fire-passive.penetrations` | alter | New and disturbed cable and pipe penetrations through fire-rated plant room walls, risers and slabs are fire-stopped and certified [D]. Fire dampers at the AHU connections are retained, and checked and tested [D]. |
| **fire-active** | `fire-active.detection` | alter | Duct and return-air smoke detectors and the FIP fire-trip relays to each AHU are disconnected and reinstated, and the cause and effect re-tested by the building's fire contractor [D]. Smoke control is a trigger. |
| **mechanical** | `mechanical.central-plant` | replace | Chillers, chilled-water pumps, valves and header connections [K: clusters/services-wet-air/systems.yaml mechanical.central-plant]. |
| | `mechanical.air-conditioning` | replace | The app files AHUs here [K: systems.yaml mechanical.air-conditioning describes "air-handling units"]. |
| | `mechanical.refrigerant-condensate` | replace | R22 is recovered from the old chillers before they are cut out, and the new factory-charged units use a current refrigerant. AHU condensate trays and drains are renewed [K: unforeseen.yaml uc.refrigerant-in-old-plant-needs-recovery-before-removal]. |
| | `mechanical.controls-bms` | alter | New plant controllers and points are integrated into the retained BMS: sequences, graphics, alarms, trends. Full replacement is a trigger. |
| | `mechanical.fire-mode-air-systems` | alter | The fire-mode shutdown of the new AHUs is reinstated and tested [K: clusters/services-wet-air/rules.yaml rule.as1668-1.air-handling-fire-mode, clause_verified false; interfaces.yaml if.bms-fire-mode-priority]. |
| | `mechanical.air-distribution` | alter | Duct transitions, flexible connections and dampers to suit the new AHUs. |
| | others | — | Outdoor air (`ventilation`) comes through the new AHUs and needs no separate item. Local exhaust, car-park ventilation, process systems and heating appliances are untouched. Cooling towers are a trigger. |
| **hydraulic** | none | — | Air-cooled plant needs no make-up water. AHU condensate drains reconnect to existing tundishes. Gas boilers replaced by heat pumps are a trigger. |
| **electrical** | `electrical.cabling-wiring` | alter | Submains and final subcircuits to the new plant. |
| | `electrical.switchboards` | alter | New outgoing devices on the main switchboard; mechanical services switchboards replaced. The MSB itself is kept, so `alter`, not `replace`. `replace` would fire `fr.switchboard-major` (procure and install a main switchboard) [K: knowledge/program/fragments.yaml fr.switchboard-major], which is wrong here. |
| | others | — | Supply augmentation is a trigger (`heating-electrification`). Temporary power is a trigger (`temporary-cooling`). Energy sub-metering is a trigger (`nabers-target`). |
| **comms-security** | none | — | The BMS network is reused. A full BMS replacement touches `comms-security.networks-integration` (trigger). |
| **vertical-transport** | none | — | The goods lift carries AHU sections and is protected, booked and possibly operated by an attendant [D]. |
| **access-egress** | `access-egress.maintenance-access` | alter | Walkways, platforms and guardrails rearranged for safe access and service clearances to the new chillers (AS 1657, verified: false) [K: interfaces.yaml if.services-plant-maintenance-access]. |

---

## 3. Existing-building interfaces

The whole job is an existing-building interface [D].

- **Occupation.** Tenants stay in. Cooling must be kept at the levels the leases require during business
  hours, so shutdowns and noisy work are out of hours and AHUs are changed floor by floor, usually over a
  weekend each. The RFP `existing_building` and `live_site` flags come from the work type and from
  `hdr.cond.operational_constraints` [K: knowledge/rfp/modules.yaml; internal/rfp/drivers.go deriveFlags,
  readFlags]. The mechanical template has no row that reads them; its `staging` row is always O
  [K: knowledge/rfp/mechanical-services.yaml positions staging].
- **Services cut-overs.**
  - **Chillers.** Stage 1 takes chiller 1 out while chiller 2 carries the load, then the reverse. Each
    cut-over needs isolation valves that hold, a drain-down and refill plan, flushing and water treatment
    before start-up [K: unforeseen.yaml uc.chilled-water-flushing-delays-commissioning], and a fallback
    if the new unit fails to start.
  - **AHUs.** Each floor loses its AHU for the change-over.
  - **Electrical.** The existing chiller must stay energised while the other's supply is altered.
  - **BMS.** Points move across without losing control of the running plant.
  - **Fire.** The AHU fire trips are isolated only under the fire contractor's control, with the
    isolation logged and restored the same shift (NSW essential fire safety measures, verified: false).
  - The knowledge's rule for occupied cut-overs [K: knowledge/clusters/services-wet-air/rules.yaml
    rule.project.occupied-services-cutover] reaches Implications only.
- **Hazardous materials.** A 1988 workplace must hold an asbestos register, and a refurbishment survey of
  the areas the works disturb is needed before work starts (WHS Regulation 2017, verified: false). Typical
  finds in plant rooms [D]:
  - asbestos gaskets and rope seals at flanges;
  - lagging on old heating pipework;
  - woven flexible duct connections on AHUs;
  - vinyl floor tiles;
  - fibre-cement linings;
  - PCB capacitors in old starters or power-factor correction;
  - synthetic mineral fibre insulation.

  The refrigerant itself is a controlled substance (§5). The app raises a survey as an Implication
  [K: knowledge/clusters/structure/consequences.yaml cq.pre-2004-fabric-hazardous-materials-survey], and
  `fr.existing-conditions` schedules a "Hazardous-materials survey" row [K: knowledge/program/fragments.yaml
  fr.existing-conditions].
- **Structure.** New operating weights, point loads and footprints on an old deck; vibration isolation;
  seismic restraint of plant (AS 1170.4 applicability, verified: false); crane outrigger loads if the crane
  stands on a suspended slab or over a basement under the footpath [K: interfaces.yaml
  if.plant-loads-structure; clusters/services-wet-air/unforeseen.yaml
  uc.replacement-cooling-tower-heavier-than-roof-allowance].
- **Matching existing work.** New pipework meets old headers and valves. New refrigerants and oils must not
  meet old pipework [K: unforeseen.yaml uc.new-refrigerant-unsuitable-for-existing-pipework]. New AHU coil
  handing must suit the existing risers [K: uc.replacement-air-handler-coil-handing-opposite-to-risers].
  New controllers must speak the BMS protocol. Fire dampers and the existing duct must accept new
  connections.
- **Dilapidation.** An internal condition record is needed of the roof, plant rooms, lifts and common
  areas on the delivery route, and of the street and footpath at the crane position (council usually asks
  for this with the crane permit) [D]. There is no dilapidation of adjoining buildings unless the crane
  oversails them [K: knowledge/clusters/delivery/unforeseen.yaml
  uc.neighbour-boundary-risks-from-crane-and-falling-objects].

---

## 4. Consultants and why

| discipline (app id) | stage engaged | need | why |
|---|---|---|---|
| Mechanical engineer (`consultant.mechanical`; a multi-services firm may file as `consultant.services`) | Investigation through seasonal review | **Always** | Lead: condition and load assessment, options and strategy, design and Section J compliance, documentation, tender, construction reviews, commissioning witness, seasonal. |
| Structural engineer (`consultant.structural`) | Investigation, documentation, construction | **Always in base** | Deck and plinth capacity, support frames, seismic restraint, inspection before plant is set down. |
| Electrical engineer (`consultant.electrical`) | Investigation, design, construction | **Always in base** (often the same firm as mechanical) | Maximum demand and MSB capacity, submains, protection, staged energisation, VSD harmonics. |
| Hazmat (`consultant.hazmat`) | Investigation | **Always in base** (pre-2004 building) | Refurbishment survey of the areas disturbed; register update; removal scope for tender. |
| BCA consultant (`consultant.bca`) | Concept and tender | **Base, limited scope** | Written pathway advice (exempt, CDC or DA), the essential fire safety measures affected, and the Section J basis for the new plant. Some owners get the same letter from a certifier [D]. |
| Mechanical contractor as head contractor (`trade.mechanical`; the app models it as `trade.main_works`) | Procurement | **Always** | Principal contractor under the WHS Act for the works. |

Everything below is conditional.

| discipline (app id) | stage engaged | need | why |
|---|---|---|---|
| Acoustic (`consultant.acoustic`) | Concept | Conditional: noise-sensitive receivers, a consent noise limit or complaints | Criteria, selection review, screens and attenuators, compliance measurement. |
| ESD (`consultant.esd`) | Concept | Conditional: NABERS target or green lease | Energy model of options, metering strategy, rating support (CONVENTIONS C2). |
| Commissioning agent (`consultant.commissioning`) | Design through seasonal | Conditional: owner policy or rating target | Independent verification of commissioning. |
| BMS consultant (`consultant.bms`) | Design | Optional, with BMS replacement | Usually the mechanical engineer specifies the BMS [D]. |
| Fire engineer (`consultant.fire_engineer`) | Design | Conditional: AHUs in a performance-solution smoke control system | Review against the fire engineering report. |
| Certifier (`consultant.certifier`) | Approvals | Conditional: CDC or DA path | Certificate, inspections, occupation certificate and fire safety certificate. |
| Heritage, town planner (`consultant.heritage`, `consultant.town_planner`) | Approvals | Conditional: heritage item | DA and heritage impact. |
| Quantity surveyor (`consultant.quantity_surveyor`) | Concept or tender | Conditional: large jobs or a fund's policy | Usually the mechanical engineer gives the order of cost [D]. |

**Not needed for the base:** architect, interior design, access, hydraulic, fire services and certifier.
The app suggests all six for a commercial refurb [K: knowledge/works/package_defaults.yaml baselines[2],
baselines[3], baselines[4]; testing/synthetic/reference/index/type-predictions.md T07], and cost-plan prepare
adopts them as fee lines [K: internal/httpapi/costs.go prepareCostPlan, per KNOWLEDGE_INDEX §3.2]. The app
also misses two the job does need: **Structural** (refurb baselines do not name it) and **Hazmat** (no rule
names it); both are in `shape.yaml procurement` with `built_in: "no"`. The six unneeded consultants are
keyed in `shape.yaml not_expected_procurement`.

---

## 5. Approvals and authorities

- **Planning pathway.** In practice, like-for-like replacement of plant in its existing location, with
  no increase in height, footprint or noise and no change to the building's appearance, is treated in
  NSW as maintenance or exempt development. The exact Codes SEPP 2008 provision relied on, and whether its
  general requirements (for example, meeting the BCA deemed-to-satisfy provisions) are met, must be
  confirmed in writing. All verified: false.
  - With exempt planning the app adds `fr.exempt-confirmation` and picks `sk.refurb-traditional-cdc`
    [K: knowledge/program/fragments.yaml fr.exempt-confirmation; knowledge/program/skeletons.yaml
    sk.refurb-traditional-cdc].
  - Bigger plant, a new or enlarged screen or a new plant deck can push it to CDC (trigger).
  - A heritage listing usually pushes it to DA (trigger).
- **No construction certificate and no occupation certificate** on the exempt path [D]. The app still
  suggests the CC fees line [K: knowledge/costs/starter_lines.yaml cc-fees] and an Occupation certificate
  program row [K: fragments.yaml fr.occupation-certificate].
- **NCC Section J.** The new plant should meet NCC 2022 Volume One Section J requirements for
  air-conditioning and ventilation:
  - minimum chiller efficiencies;
  - fan and pump power;
  - pipe and duct insulation;
  - economy cycle and control requirements.

  On the CDC or DA path they are checked through a certificate. On the exempt path, if the exemption
  relies on the Codes SEPP, its general requirements include meeting the relevant BCA deemed-to-satisfy
  provisions, so Section J compliance is a condition of the exemption, which the BCA consultant confirms
  (verified: false). The Part number is **verified: false**: the app's own
  rule carries three conflicting Part numbers from its seeds
  [K: knowledge/clusters/services-wet-air/rules.yaml rule.ncc.hvac-energy-efficiency, clause_verified false].
  I believe NCC 2022 numbers it Part J6, but I have not confirmed this from the instrument.
- **Essential fire safety measures.** The AHU shutdown, fire dampers, smoke detection and automatic
  fail-safe devices are listed measures on the building's fire safety schedule. They must stay working
  or be managed during the works, and be tested and recorded so the owner's next annual fire safety
  statement can rely on them. The NSW Environmental Planning and Assessment (Development Certification and
  Fire Safety) Regulation 2021 provisions are verified: false. The app's NSW essential-measures continuity
  check fires as an Implication [K: knowledge/clusters/fire/consequences.yaml
  cq.nsw-essential-measures-continuity-review, per index/triggers.md §5]. The 25 m effective-height
  trigger for zone smoke control and stair pressurisation is an NCC smoke hazard management provision
  (Part E2 and its tables), not an AS 1668.1 number (verified: false). AS 1668.1 sets how the fire mode
  works (verified: false) [K: knowledge/clusters/services-wet-air/rules.yaml
  rule.as1668-1.air-handling-fire-mode numbers]. An existing building keeps its approved measures, so the
  base fire mode rests on the fire safety schedule, not on today's threshold.
- **Refrigerant.** The old R22 charge is recovered by a technician holding an Australian refrigerant
  handling licence, and returned for destruction. Venting is an offence. The Ozone Protection and
  Synthetic Greenhouse Gas Management Act 1989 (Cth) and its Regulations are verified: false.
- **Pressure equipment.** Chillers and their receivers may be plant whose design must be registered
  under the WHS Regulation 2017, by AS 4343 hazard level [K: rules.yaml
  rule.whs.pressure-equipment-registration, clause_verified false].
- **Crane and road.** The crane on the street and the footpath closure need council approval under the
  Roads Act 1993 s 138, with a traffic control plan. A classified road or a full closure adds a Road
  Occupancy Licence from the TfNSW Transport Management Centre (trigger). The crane is registered plant
  with a licensed operator and dogger or rigger (WHS Regulation). All verified: false.
- **SafeWork NSW.** Licensed asbestos removal is notified in advance, and a clearance certificate is
  issued before the area is reoccupied (trigger; verified: false).
- **Electrical.** The licensed electrician issues a Certificate of Compliance for Electrical Work for
  the new submains (NSW, verified: false). There is no DNSP application unless the maximum demand rises
  (trigger).
- **Long service levy.** The NSW building and construction long service levy is paid before work starts.
  With no CC it is paid directly. Its threshold and rate, and whether plant replacement counts as
  building work for the levy, are verified: false.
- **Not involved in the base:** Sydney Water (unless cooling-tower bleed-off needs a trade waste consent;
  trigger), BASIX, DBP Act (Class 5 is outside it: owner ruling R3), Home Building Act.
- **Hold points** (the contract's, set by the mechanical engineer) [D]:
  1. hazmat clearance of each plant area before disconnection;
  2. support frames inspected by the structural engineer before plant is set down;
  3. pipework pressure test, then flushing and water treatment, before chiller start-up;
  4. manufacturer start-up before the old second chiller is removed;
  5. fire-mode and FIP interface test before each floor is handed back;
  6. integrated systems test before practical completion.

  The app has no interface hold point for any of these. Only five `sequences` hold points resolve,
  none of them mechanical [K: KNOWLEDGE_INDEX §4.4].

---

## 6. Cost plan shape

Lines only; no amounts (see `shape.yaml cost_plan`).

- **Fees and charges.** Long service levy; council and road occupancy permits for the crane. There are
  **no** DA fees, no CC fees and no service-authority fees in the base.
- **Consultants.** Mechanical engineer (staged: investigation and options, design and tender,
  construction and commissioning, seasonal), structural engineer, electrical engineer, hazardous materials
  consultant, BCA consultant.
- **Construction**, by trade or element [D]:
  - preliminaries and occupied-building controls;
  - crane, lift plan and traffic control;
  - decommissioning and refrigerant recovery;
  - chillers;
  - pumps, pipework, flushing and water treatment;
  - AHUs;
  - mechanical services switchboards;
  - BMS integration;
  - testing, commissioning, balancing and seasonal commissioning;
  - support frames;
  - roof membrane make-good;
  - fire interface and re-test;
  - fire stopping;
  - submains and MSB changes;
  - plant access platforms;
  - provisional sums for latent conditions.
- **Contingency and other.** Design contingency, construction contingency, escalation (chillers are
  often imported and priced in foreign currency), exclusions (tenant supplementary cooling and tenant BMS
  points), and the owner's own building-management costs during the works (after-hours building manager,
  security, contractor call-outs).

App behaviour:

- With no documents the app gives nine consultant fee lines, the four contingency lines, P05 and P06
  [K: testing/synthetic/reference/index/type-predictions.md T07].
- Once work items exist, construction appears as generic element lines: 03 Structure, 04 Building
  envelope, 06 Passive fire, 07 Active fire, 08 Mechanical services, 10 Electrical services and a 13.x
  "Plant and maintenance access" child [K: knowledge/costs/elements.yaml; internal/costs/suggest.go
  itemised children]. The whole mechanical scope is **one** 08 line.
- A stated budget is spread by the `commercial_fitout` split [K: knowledge/costs/budget_splits.yaml
  commercial_fitout, verified: false]. Only lines with a share get money, normalised over those present
  [K: internal/store/cost_budget.go spreadConstructionBudget]. So with mechanical, electrical and fire
  lines only, electrical gets almost as much as mechanical, and structure and envelope get nothing. That is
  a misleading picture of a plant job.
- **Lines the app produces that this job should not carry** (`shape.yaml not_expected_cost_plan`): P05
  CC fees, P06 service authority fees, and the Architect, Interior Design, Access, Hydraulic, Fire
  Services and Certifier fee lines.

---

## 7. Program shape

The base has 45 activities; see `shape.yaml program`. The app's skeleton, `sk.refurb-traditional-cdc`, is
the right one [K: knowledge/program/skeletons.yaml]. Stages and the activities that drive each:

1. **Investigation.** Plant condition, capacity and refrigerant review; measure the plant rooms and
   routes; hazardous-materials survey; trace existing services; structural assessment of the deck;
   maximum demand check.
2. **Concept design.** Confirm the brief, then the options and replacement-strategy report (plant type,
   refrigerant, staging, temporary cooling, crane), then the owner's selection and capital approval, then
   design development.
3. **Construction documentation.** Specification, drawings, staging and cut-over plan; coordinated
   consultant documents; owner sign-off.
4. **Approvals.** Confirm the exempt path, in parallel with documentation.
5. **Procurement.** Tender package (AS 4000, staging and out-of-hours preliminaries, schedule of rates),
   tender with plant-room inspection, evaluation, award.
6. **Construction.**
   - Submittals, then plant manufacture and delivery. This is **the long-lead item that sets the season**
     [D].
   - Tenant notices and the shutdown schedule.
   - Council crane approval and traffic control plan.
   - Lift plan and support frames, then a hold point: the structural engineer inspects the frames before
     plant is set down.
   - Then **Stage 1**: isolate and recover refrigerant → crane out and lift in → flushing hold point →
     start-up and commissioning.
   - Then **Stage 2**, the same again.
   - In parallel: AHUs floor by floor out of hours; submains and switchboards; BMS integration; fire-mode
     re-test; fire stopping.
   - Integrated systems test.
7. **Completion.** Practical completion; update the essential fire safety measure records for the next
   AFSS; handover (O&M manuals, as-builts, BMS graphics, training).
8. **Defects.** Seasonal commissioning and tuning in the opposite season; defects inspection; defects
   liability period (the contract's; length verified: false); final inspection after the seasonal
   review.

**Logic.**

- The hazmat survey and the structural and electrical capacity checks come before the concept.
- Plant delivery, the crane approval, the lift plan and the inspected frames all come before each lift.
- Switchboard delivery comes before the submains and switchboards are installed, and the new supply
  comes before chiller start-up and AHU commissioning.
- Stage 1 is commissioned before Stage 2 starts: N-1 means the second chiller must not go out until the
  first new one runs.
- The AHU commissioning comes before its fire-mode re-test.
- The seasonal review comes before final defects.

**Long-lead items** [D]:

- chillers (often imported, made to order);
- AHUs (custom-sized to fit the plant rooms);
- mechanical services switchboards;
- variable-speed drives;
- council crane approval, and a Road Occupancy Licence if needed.

The app has no lead-time row for any of them on a Class 5 refurb. Its only AC lead-time duration is
`du.ac-plant-lead-2`, attached to the Class 2 fragment `fr.ac-2` [K: knowledge/program/durations.yaml;
fragments.yaml fr.ac-2].

**What the app builds** [K: testing/synthetic/reference/index/type-predictions.md T07; fragments.yaml]:

- the fit-out process rows (concept, design development, fit-out documentation, tender, site
  establishment, award);
- `fr.existing-conditions`;
- `fr.exempt-confirmation`;
- `fr.mechanical-fitout` on the AHU item ("Alter ductwork and pipework" and "Commission the altered
  system", duration scaled by gross floor area [K: durations.yaml du.mechanical-rough-fitout]);
- `fr.electrical-fitout`, `fr.detection-fitout` and `fr.fire-stopping` on their items;
- a generic Investigation row for the structural check;
- PC, an Occupation certificate row it should not have, handover and defects.

**Nothing for the chillers, the BMS, the refrigerant, the crane or the staging**: those systems have no
fragment [K: testing/synthetic/reference/index/program-fragments.md, leaf systems with no work-item
fragment: `mechanical.central-plant`, `mechanical.controls-bms`, `mechanical.refrigerant-condensate`,
`mechanical.fire-mode-air-systems`, `mechanical.cooling-towers`, `structure.temporary-works`].

---

## 8. RFP shape

**Mechanical engineer** (`mechanical-services` template [K: knowledge/rfp/mechanical-services.yaml]).

- **Must carry:**
  - investigation of the existing plant (condition, capacity, refrigerant, BMS trend logging of actual
    loads);
  - options and replacement strategy;
  - selection and design to measured loads and Section J (Part verified: false);
  - fire mode of the new AHUs to AS 1668.1 and the fire safety schedule (verified: false);
  - BMS integration;
  - coordination of supports, crane, power, refrigerant and condensate;
  - a staged cut-over plan;
  - refrigerant recovery requirements;
  - plant noise criteria in the selections;
  - traditional tender documents and evaluation;
  - construction-phase reviews and hold points;
  - commissioning witness and balancing;
  - a seasonal review;
  - design certification.
- **Must not carry:** a planning stage, novation or continued design, kitchen or car-park exhaust,
  whole-building energy rating (base), DBP Act regulated design.
- **App:** the boundary is inception to design, so tender, construction, commissioning and seasonal
  stages become priced options. The investigation stage and the smoke, controls, interfaces and
  declarations rows hang on client flags (`mech_*`) that no document can set, so they ask. The
  existing-condition row is hidden until someone answers the investigation stage. `staging` is always O.
  All listed in `app_differs`.

**Structural engineer** (`structural` template [K: knowledge/rfp/structural.yaml]).

- **Must carry:** existing-structure assessment of the deck and plinths, support frame and seismic
  restraint design (`services_fixings`), construction documents, construction-phase inspection before
  plant is set down, an inspection schedule, and review of crane loads where they bear on the building.
- **Must not carry:** a geotechnical brief, a shared Revit model, a planning or regulated-design stage,
  retention design, dilapidation of neighbours.
- **App:** concept and schematic design always included; Class 5 makes the scale complex (shared model
  I); `services_fixings` always O; construction phase an option because it falls after the boundary end.

**Electrical engineer** (`electrical-services` template [K: knowledge/rfp/electrical-services.yaml]).

- **Must carry:** existing supply, MSB and submain investigation with maximum demand logging; plant
  supply design; distribution and protection; staged energisation; metering and BMS interface;
  construction support; VSD harmonics.
- **Must not carry:** lighting, a DNSP application (base), ICT or security cabling, generation, PV or EV,
  fire detection design.
- **App:** concept included and the utility row I by default; lighting I whenever design is in;
  investigation, staging, controls and declarations ask on `elec_*` client flags. The existing-condition
  row is hidden until someone answers the investigation stage.

**BCA consultant** (`bca-certification` template [K: knowledge/rfp/bca-certification.yaml]).

- **Must carry:** pathway confirmation naming the provision relied on; essential fire safety measures
  affected; Section J advice; a tender-stage compliance review; an evidence register.
- **Must not carry:** certification functions, construction approval assessment, performance solutions,
  residential energy evidence, an access upgrade report.
- **App:** the A stages need the client flag `bca_advisory`, so they ask. The B stages are absent on the
  exempt pathway.

**Hazmat consultant**: no template; the app issues the generic `tpl.rfp-capex`.

- **Must carry:** a refurbishment survey of the plant rooms, risers and voids the works disturb; NATA
  laboratory analysis; PCBs, synthetic mineral fibre and lead paint; a register update and a removal scope
  for the tender; and, as an option, clearance and air monitoring.
- **Must not carry:** a contaminated land assessment, a survey of areas not affected, or removal works.

---

## 9. Triggers

Each trigger's evidence, negation and additions are in `shape.yaml triggers`. Summary, with how documents
show them:

| id | shows up as (including indirectly) | denied by | adds |
|---|---|---|---|
| `cooling-towers` | "water-cooled centrifugal chillers", "condenser water", "Legionella sampling", "the tower's risk management plan", a council cooling-tower registration notice | "air-cooled chillers; no cooling towers" | Tower replacement, water treatment, council notification and registration (Public Health Act 2010, verified: false), trade waste for bleed-off (verified: false), mechanical RFP scope (AS/NZS 3666). |
| `temporary-cooling` | "only one chiller", "data room needs 24/7 cooling", "must proceed over summer, when the remaining chiller cannot carry the load" | "staged in the cooler months; no temporary cooling" | Hire chiller, temporary power or generator, connection points, extra cut-overs. |
| `asbestos-in-plant-room` | register entries for gaskets and lagging, "presumed asbestos" flexible connections, plant-room vinyl tiles | a refurbishment survey that sampled the gaskets, lagging and flexible connections and found no asbestos | Licensed removal, SafeWork notice, air monitoring and clearance (verified: false). |
| `noise-sensitive-receivers` | "apartments across the lane", a consent plant-noise condition, "complaints about chiller noise at night" | "surrounded by commercial buildings" | Acoustic consultant, screens and attenuators, compliance measurement. |
| `classified-road-crane-lift` | a named classified road, "Road Occupancy Licence", "Transport Management Centre", "full road closure" | "crane in the forecourt", "local street under council permit" | ROL, closure costs, out-of-hours lift premium. |
| `basement-plant-room` | "B2 plant room", "removable floor hatch", "delivered in sections", "machinery room leak detection" | "chillers on the roof plant deck" | Dismantle and reassemble, machinery room to AS/NZS 5149.3 (verified: false). Removes the crane lines. |
| `bms-replacement` | "discontinued proprietary BMS", "open-protocol BACnet", "pneumatic actuators" | "BMS upgraded in 2021 and retained" | BMS replacement, parallel running, cybersecurity. |
| `nabers-target` | "5.5-star NABERS Energy target", "green lease", "lift the base building from its current 3.5 stars to at least 5 stars" | "no rating target; like-for-like" | ESD consultant, sub-metering, a tuning period before the rating. |
| `complying-development` | "taller chillers need an enlarged screen above the parapet", "certifier advises a CDC" | "certifier confirmed exempt in writing" | Certifier (and its fee line), CDC under the Commercial and Industrial Alterations Code (verified: false), OC, fire safety certificate. Removes the exempt confirmation. Both the BCA and the certifier RFPs block (`pathway_unsupported`, bca-certification lists cdc). |
| `heritage-item` | LEP Schedule 5 listing, conservation area, "plant visible from the street" | "not a heritage item or in a conservation area" | DA, heritage and planning consultants, certifier, CC and OC. |
| `fire-engineered-smoke-control` | "zone smoke control under the fire engineering report", "approved by a performance solution in the original fire engineering report", the fire safety schedule listing zone smoke control | "AHUs only shut down on alarm" | Fire engineer, smoke-control tests, certifier advice. |
| `heating-electrification` | "gas boilers replaced with heat pumps", "gas meter removed", "net zero" | "boilers retained, outside scope" | Gas abolishment, maximum demand and DNSP application, heat pump scope. |
| `plant-deck-strengthening` | "slab cannot carry the new operating weight", "new steel platform", "carbon-fibre strengthening" | "existing plinths carry the new loads" | Strengthening design, a hold point, roof membrane renewal. |
| `independent-commissioning` | "independent commissioning agent", "commissioning authority engaged by the Principal" | "commissioning witnessed by the mechanical engineer" | Commissioning agent, witness tests. |
| `facade-opening-for-ahu-delivery` | "craned in through the plant room louvres", "glazing panels removed to deliver AHU sections" | "delivered in sections through the goods lift" | Facade removal and reinstatement, temporary weatherproofing, a water test. |
| `capacity-increase` | "replacement chillers are 20 % larger", "an N+1 standby chiller is added" | "like-for-like capacity" | Action `upgrade`; maximum demand check, possible supply augmentation and P06; deck load check. |
| `crane-oversails-neighbour` | "the crane jib will oversail 14 Smith Street" | "the lift radius stays within the site and the road reserve" | Oversail licence with the adjoining owner (verified: false), dilapidation survey of the neighbour, an activity before the lift. |
| `flammable-refrigerant` | "R1234ze, an A2L refrigerant", "mildly flammable" | "a non-flammable A1 refrigerant" | Safety-group and charge-limit checks and leak detection to AS/NZS 5149 (verified: false). Optional. |

Variants worth one-change tests: `temporary-cooling`, `asbestos-in-plant-room`,
`noise-sensitive-receivers`, `cooling-towers`, `bms-replacement`, `nabers-target`,
`complying-development`, `plant-deck-strengthening`.

---

## 10. Keyword traps and negations specific to this type

- **"Basement".** "Basement car park", "B2 plant room" or "the main switchroom in the basement" route to
  the basement question [K: knowledge/profile/flag_readings.yaml basement]. A yes switches on structural
  retention design, temporary-works interfaces, dilapidation and complex scale. Expectation:
  `flag.basement` is not yes in the base.
- **"Demolish", "demolition", "strip-out" of old plant** imply `site.demolition` [K:
  knowledge/profile/scope_defaults.yaml implied_by strip-out]. That fires `fr.strip-out-fitout` ("Soft-strip
  partitions, ceilings and services") and an 01 Site works line. Removing plant is part of `replace`.
- **"Stairs"** in "AHU sections carried up the fire stairs" implies `access-egress.stairs-and-ramps`
  [K: scope_defaults.yaml implied_by stair], which brings a "Stairs, ramps, handrails and nosings" cost
  line.
- **"Kitchen"** in "tenant kitchens on each floor are not affected" or **"bathroom"** in "bathroom exhaust
  fans are not affected" implies hydraulic and local-exhaust scope [K: scope_defaults.yaml implied_by
  kitchen, wet-room]. A stated "not affected" should deny it, but the pattern match does not read
  negation. ("Kitchenettes" and "toilet" do not match the patterns.)
- **"NABERS"** anywhere, even "the building's current NABERS rating is 4 stars" or "the Commercial
  Building Disclosure certificate shows a 3.5-star NABERS Energy rating", sets
  `flag.sustainability_target` [K: flag_readings.yaml sustainability_target]. A rating that exists is not
  a target; the base expects `no`.
- **"Net zero"** in an owner's corporate statement sets the same flag.
- **"Road occupancy"** sets `flag.public_road`, and **"TfNSW" or "classified road"** sets
  `flag.state_road` [K: flag_readings.yaml public_road, state_road]. The crane occupies the road; there
  are no road works. Low consequence: only civil and traffic rows read them.
- **"Stage 1 / Stage 2"** sets `flag.staged` [K: flag_readings.yaml staged]. That is correct here (the
  chiller staging). But "fee stages" or "Stage 2 of the NABERS commitment" are not construction stages.
- **"Fire"** words: "fire mode", "fire trip" and "fire dampers" are mechanical and passive-fire interfaces,
  not fire services design. They should not make a Fire Services consultant look needed.
- **"Cooling tower"** in a negation ("there are no cooling towers") must not add
  `mechanical.cooling-towers`.
- **Capacity words.** "Increased capacity", "additional chiller" or "N+1 redundancy added" mean
  `upgrade`, not `replace`. "Like for like" means `replace`.
- **"Refrigerant"** in "refrigerant leak detection in the machinery room" belongs to the basement
  machinery-room variant, not the base.
- **"Exempt"** can appear in "GST exempt" or "exempt from the long service levy". Only the planning
  context counts for `hdr.cond.planning`.
- **The landlord's NLA versus GFA.** Durations scale by `plan.gross_floor_area`, assumed 250 m² until
  stated [K: KNOWLEDGE_INDEX §4.4]. An office NLA figure is neither the GFA nor the plant-room area, so the
  AHU durations scale on the wrong driver either way.

---

## 11. Gaps in the built-in knowledge

What is missing, and what the app will probably produce wrongly as a result (all keyed in `shape.yaml
gaps`):

1. **Refurb starts empty, and nothing implies mechanical plant** [K: scope_defaults.yaml
   empty_work_types, implied_by]. Unless a filed document is read as including the chillers, AHUs and
   BMS, the app has no construction scope at all: no 08 line, no fit-out fragments, no scope-addition
   consultants beyond the baselines.
2. **The core systems have no program rows** (`central-plant`, `controls-bms`, `refrigerant-condensate`,
   `fire-mode-air-systems`, `cooling-towers`, `structure.temporary-works`, Class 5 `structure.steel`)
   [K: index/program-fragments.md]. The program misses:
   - the critical path: plant lead time, crane approval, staged cut-overs;
   - every hold point;
   - seasonal commissioning.

   It will show a fit-out program with a GFA-scaled "Alter ductwork and pipework" row.
3. **Wrong consultants.** Six unneeded consultants are suggested and adopted into the cost plan.
   Structural and Hazmat, both needed, are never suggested. Acoustic, ESD, commissioning, BMS consultant
   and fire engineer are never suggested [K: package_defaults.yaml].
4. **Head contractor modelling.** The app proposes only a `trade.main_works` head contract; a mechanical
   contractor as head contractor is not representable except as a trade package
   [K: internal/procurement/discipline.go DisciplinePackage].
5. **Package stages.** A consultant's default stages under `sk.refurb-traditional-cdc` are concept
   design and construction documentation only. The mechanical engineer's investigation, tender,
   construction, commissioning and seasonal stages never appear as fee stages [K: KNOWLEDGE_INDEX §2.3].
6. **Exempt path not respected everywhere.** CC fees and service-authority fees are always suggested
   [K: starter_lines.yaml], and an Occupation certificate row is always added for refurb
   [K: fragments.yaml fr.occupation-certificate].
7. **Cost shape.** The whole job is one 08 line. There is no preliminaries, crane, permits, levy or
   provisional-sum line. The `commercial_fitout` split misallocates a stated budget [K:
   budget_splits.yaml commercial_fitout; internal/store/cost_budget.go].
8. **RFP selection.** The mechanical template prices tender, construction, commissioning and seasonal as
   options and asks on six client flags. The structural template includes concept design, a geotechnical
   brief and a shared model, and prices the core support-frame design as an option [K:
   mechanical-services.yaml, structural.yaml]. No template exists for hazmat, acoustic, ESD or
   commissioning.
9. **NSW and Commonwealth law for this type is not encoded**, beyond draft rules with
   `clause_verified: false` that reach only Implications [K: knowledge/clusters/services-wet-air/rules.yaml]:
   - ozone protection licensing;
   - pressure equipment registration;
   - Roads Act crane approvals;
   - POEO noise;
   - cooling-tower regulation;
   - essential fire safety measures and the AFSS;
   - NABERS and Commercial Building Disclosure;
   - the long service levy.
10. **No outbound-noise determinant.** `det.noise_affected_site` measures noise coming in from roads,
    rail and aircraft [K: knowledge/determinants.yaml noise_affected_site]. Neighbour exposure to plant
    noise has no profile key.

---

## 12. Sources and unverified claims

**Repository sources opened:**

- `testing/synthetic/PROJECT_TYPES.md` (T07 row);
- `testing/synthetic/DISCOVERY.md` §5–§10;
- `testing/synthetic/reference/KNOWLEDGE_INDEX.md`, `OWNER_RULINGS.md`, `CONVENTIONS.md`;
- `reference/index/type-predictions.md` T07, `systems.md`, `scope-defaults.md`, `cost-plan.md`,
  `program-fragments.md`, `rfp-templates.md`, `triggers.md`;
- `knowledge/systems.yaml`;
- `knowledge/clusters/services-wet-air/systems.yaml`, `rules.yaml`, `interfaces.yaml`, `unforeseen.yaml`;
- `knowledge/clusters/structure/systems.yaml`, `consequences.yaml`;
- `knowledge/clusters/delivery/unforeseen.yaml`;
- `knowledge/clusters/fire/systems.yaml`;
- `knowledge/works/actions.yaml`, `package_defaults.yaml`;
- `knowledge/costs/elements.yaml`, `starter_lines.yaml`, `budget_splits.yaml`;
- `knowledge/program/fragments.yaml`, `skeletons.yaml`, `durations.yaml`;
- `knowledge/rfp/mechanical-services.yaml`, `structural.yaml`, `electrical-services.yaml`,
  `bca-certification.yaml`;
- `knowledge/profile/flag_readings.yaml`, `project_details.yaml`, `taxonomy.yaml`;
- `knowledge/determinants.yaml`;
- `data/intake/disciplines.json`;
- `internal/costs/suggest.go`, `internal/costs/budget.go`, `internal/store/cost_budget.go`.

The reviewed pilot maps T05 and T09 were used for form.

**Unverified claims (verified: false):**

- The Codes SEPP 2008 provision that makes like-for-like plant replacement exempt, and its
  general-requirements clause.
- That heritage items generally cannot use Codes SEPP exempt development, and the LEP cl 5.10(3)
  minor-works route.
- The NCC 2022 Section J Part number and the specific requirements for air-conditioning plant (I believe
  Part J6).
- AS 1668.1 fire-mode requirements, and the 25 m effective-height trigger for zone smoke control and
  stair pressurisation, which is an NCC smoke hazard management provision (Part E2 and its tables), not
  an AS 1668.1 number.
- The Codes SEPP Commercial and Industrial Alterations Code (Part 5) as the CDC route.
- A crane oversail licence with an adjoining owner.
- AS 1170.4 seismic restraint of plant in an existing building.
- AS 1657 for plant access.
- AS/NZS 5149.3 machinery rooms.
- AS/NZS 3666 for cooling towers.
- The Public Health Act 2010 and Public Health Regulation 2022 cooling water system duties and council
  registration.
- Sydney Water trade waste consent for bleed-off.
- The Ozone Protection and Synthetic Greenhouse Gas Management Act 1989 (Cth) licensing and recovery
  duties.
- WHS Regulation 2017: plant design registration, crane registration and operator licensing, the asbestos
  register for pre-2004 workplaces, the refurbishment survey, licensed removal, SafeWork notification and
  clearance.
- Roads Act 1993 s 138 for the crane, and the Road Occupancy Licence from the Transport Management
  Centre.
- POEO Act 1997 offensive noise and the NSW Noise Policy for Industry.
- The EP&A (Development Certification and Fire Safety) Regulation 2021: essential fire safety measures,
  the AFSS, fire safety certificates and competent fire safety practitioners.
- The NSW Certificate of Compliance for Electrical Work.
- The NSW long service levy threshold, rate and application to plant replacement.
- The NABERS rating basis (12 months of data) and the Commercial Building Disclosure scheme.
- The DNSP connection process.
- The contract's defects liability period.

**Domain judgements the owner may want to check:**

- that a BCA consultant is engaged in the base (some owners rely on the mechanical engineer and skip it);
- that the electrical engineer is a base package rather than part of the mechanical engineer's scope;
- that acoustic is a trigger rather than base for rooftop air-cooled plant;
- that the AHU fire-mode re-test is done by the building's fire contractor with no fire services
  consultant.
