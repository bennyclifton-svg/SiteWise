# T06 Sprinkler pump and tank upgrade, with fit-out: domain map

Draft for owner review, 30 October 2026; round 1 review findings applied 4 November 2026 (see
`REVIEW.md`). The machine key is `shape.yaml` in this folder; it passes `validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Any statute or standard clause not confirmed from
the instrument is marked **verified: false**.

Most of the app's knowledge is aimed at new buildings. This map tests what it misses on an existing,
occupied Class 5 building where the job is mostly one system, `fire-active.fire-water`, which has no
program content at all [K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §8 gap 3].

---

## 1. The job

**What it is.** The owner of an occupied office building replaces a sprinkler water supply that no longer
performs. The usual cause is one of three [D]:

- the annual fire safety statement (AFSS) round, or the routine AS 1851 flow test, shows the existing pump
  falls short of the system's demand [K: knowledge/clusters/fire/failure_modes.yaml
  fm.fire-pump-fails-flow-test];
- a new Sydney Water pressure and flow statement shows the town main can no longer meet the AS 2118.1
  demand without storage [K: knowledge/clusters/fire/failure_modes.yaml
  fm.fire-water-supply-reported-insufficient];
- an old pumpset reaches end of life and spares are gone.

The fix is a new pumpset arrangement (electric duty, diesel standby, jockey), a fire water storage tank,
and pump room works. The design must satisfy AS 2118.1 for the supply and AS 2941 for the pumpsets
[K: knowledge/clusters/fire/rules.yaml rule.as2118.sprinkler-design, rule.as2941.fire-pumps]; both rules
are `clause_verified: false`, and no clause number is used in this map.

**Base scenario** (used in `shape.yaml`):

- a 1988 Class 5 office building in Sydney, six storeys over one basement car park (Class 7a),
  sprinklered throughout, single owner;
- levels 1–2 and 4–6 stay tenanted and in use; level 3 is vacant;
- the old single electric pump, boosted from the town main, fails its flow test, and Sydney Water's
  statement confirms the main alone cannot meet demand;
- the owner installs a new electric duty pumpset, a diesel standby pumpset, a jockey pump, controllers and
  a modular fire water storage tank in the basement pump room, enlarged into an adjoining store. The tank
  panels come in through the car park entry;
- the existing fire service connection to the street main is kept; its backflow device is upgraded;
- the hydrant system and booster are not changed and stay in service;
- the owner also fits out level 3 for a new tenant. That fit-out's sprinkler layout is calculated against
  the new supply, so its sprinkler commissioning and its occupation certificate wait on the new pumps.
  This is the "link";
- complying development with one principal certifier and two CDCs (the Fire Safety Code for the pump
  works, the commercial internal alterations code for level 3; §5), traditional procurement, and two
  AS 4000 lump-sum contracts: a fire protection contractor for the upgrade, and a fit-out builder for
  level 3 [D].

**Scale.** $0.2–2 M [K: testing/synthetic/PROJECT_TYPES.md T06 row]. The cost is driven by the pumpset
and tank, the pump room builder's works, and the out-of-hours cut-over, not by floor area [D].

**Who procures it and how** [D]:

- **Client.** The building owner, through its asset or facilities manager, with a project manager.
- **Lead consultant.** A hydraulic and fire services engineer investigates the supply, designs the
  upgrade, documents it, reviews submittals and witnesses commissioning. It also designs the level 3 wet
  and dry fire alterations.
- **Support.** A structural engineer (tank loads, plinths, coring), an electrical engineer (fire pump
  supply, level 3), a mechanical engineer (diesel pump room ventilation, level 3 air), a hazardous
  materials consultant (1988 building), an interior designer for level 3, and a registered certifier.
- **Contractors.** A licensed fire protection contractor under a direct contract, which subcontracts the
  builder's works and the electrical supply. A fit-out builder for level 3, whose fire contractor alters
  the floor's sprinklers.
- **Ongoing.** The building's AS 1851 maintenance contractor, usually also the competent fire safety
  practitioner for the AFSS, manages isolations and does the final assessments [D; role names
  verified: false].

**What makes it go wrong** [D unless tagged]:

- The supply is assumed rather than tested, and the design is fixed before the Sydney Water statement or
  flow test is in [K: knowledge/clusters/fire/failure_modes.yaml fm.fire-water-supply-assessed-late].
- The tank operating load is not given to the structural engineer
  [K: knowledge/clusters/fire/interfaces.yaml if.fire-tanks-load-structure].
- The fire pump supply is taken from the wrong side of the main switch, or its cable route is not
  fire-protected, or the pump's starting current drops the board voltage
  [K: knowledge/clusters/fire/interfaces.yaml if.essential-power-supplies-fire-water;
  knowledge/clusters/electrical-comms/failure_modes.yaml fm.fire-pump-start-drops-board-voltage].
- The pump room cannot be reached by the brigade, is not fire separated, or floods
  [K: knowledge/clusters/fire/unforeseen.yaml uc.fire-pump-room-not-reachable-from-outside;
  knowledge/clusters/fire/failure_modes.yaml fm.fire-pump-room-flood-exposure].
- The sprinkler system is isolated in an occupied building without temporary protection, or restored
  without a test [K: knowledge/clusters/fire/failure_modes.yaml
  fm.fire-impairment-without-temporary-protection; knowledge/clusters/fire/rules.yaml
  rule.project.fire-system-impairment].
- The fit-out's sprinkler calculations are done on the old supply, or the fit-out ceilings close before
  its pipework is pressure tested [K: knowledge/clusters/fire/interfaces.yaml
  if.sprinkler-pressure-test-before-ceilings].
- Long-lead diesel pumpsets and controllers are ordered late.
- The final fire safety certificate is forgotten until the tenant wants to move in, so the OC slips.

---

## 2. Systems and work items

Action ids are the app's eight [K: knowledge/works/actions.yaml actions]. The refurb default is `alter`
[K: knowledge/works/actions.yaml work_type_defaults], and refurb has no default scope at all
[K: knowledge/profile/scope_defaults.yaml empty_work_types], so every item below comes from documents.
The app makes one work item per system and part [K: internal/profile/works.go ProposedWorks], so where
both contracts touch a system (sprinklers, ventilation, cabling) they share one item.

- **site.** `site.demolition` **remove**: decommission and remove the old pump, controller and pipework;
  soft-strip level 3. Out of scope: earthworks, stormwater, landscape, vehicle access, trees, public
  domain. `site.services-connections` comes in only under `town-main-inadequate`.
- **substructure.** Out of scope; no footings, piles or excavation. The tank sits on the existing basement
  slab.
- **structure.** `structure.alterations` **alter**: plinths, any slab strengthening found by the
  assessment, and core holes. No new frame.
- **envelope.** Out of scope. The diesel exhaust flue may penetrate the roof (trigger
  `diesel-noise-exhaust`).
- **interiors** (level 3 only). `walls-linings` **new**, `ceilings` **alter** (grid kept, new tiles),
  `internal-doors` **new**, `internal-glazing` **new**, `joinery` **new** (tea point, reception),
  `painting` **replace**, `floor-finishes` **replace**. Out: wet areas and tiling (trigger
  `fit-out-amenities`), acoustic separation, FF&E (tenant's).
- **fire-passive.** `rated-elements` **alter** (the enlarged pump room's fire-resisting walls and door;
  AS 2941 separation, verified: false). `penetrations` **new** (fire stopping of new pipe and cable
  penetrations). Out: compartments, external walls.
- **fire-active.** The heart of the job.
  - `fire-water` **upgrade**: new pumpsets, jockey, controllers, tank, suction and test lines, diesel fuel
    tank. The system's description includes pumpsets, pump rooms, tanks and diesel fuel storage
    [K: knowledge/clusters/fire/systems.yaml fire-active.fire-water]. Upgrade, because capacity rises.
  - `sprinklers` **alter**: connect the new pump discharge to the existing valve sets, and relocate and
    add level 3 heads. There is no change of hazard or design density.
  - `hydrants` **retain**: the hydrant system stays in service through the works.
  - `detection` **alter**: level 3 detectors, and the pump run, fault and diesel signals to the fire
    indicator panel (FIP).
  - `occupant-warning` **alter**: level 3 EWIS speakers.
  - `emergency-lighting` **alter**: level 3 exit signs and luminaires.
  - Out: smoke control, special suppression, smoke alarms (Class 5 has none), and `fire-control` (the
    pump signals are FIP inputs, not a new cause-and-effect matrix).
- **mechanical.** `local-exhaust` **new** (pump room ventilation: diesel combustion and cooling make-up
  air, plant room exhaust and the engine exhaust flue discharge). Plant room exhaust is local exhaust:
  `mechanical.ventilation` is outdoor air for occupied rooms and its `excludes` names plant rooms, while
  `mechanical.local-exhaust` lists plant rooms, their make-up air and discharge points [K:
  knowledge/clusters/services-wet-air/systems.yaml mechanical.ventilation excludes,
  mechanical.local-exhaust]. `air-distribution` **alter** (level 3 VAV boxes and diffusers);
  `air-conditioning` **new** (comms room split system). Out: outdoor-air `ventilation`, central plant,
  BMS beyond new points, smoke modes.
- **hydraulic.** `backflow-prevention` **upgrade** (the fire service containment device brought to
  current requirements; Sydney Water registration, verified: false). `cold-water` **alter**, `hot-water`
  **new**, `sanitary` **alter** (tea point). Out: gas, trade waste, rainwater, roof drainage.
- **electrical.** `switchboards` **alter** (supply-side fire pump connection at the MSB; AS/NZS 3000,
  verified: false); `cabling-wiring` **alter** (fire-rated submain; level 3 power); `lighting`
  **replace** (level 3). Out: utility supply (trigger `electrical-supply-upgrade`), standby power, PV.
- **comms-security.** `structured-cabling` **new** (level 3). Out: CCTV, access control (tenant's).
- **vertical-transport.** Out of scope. The lifts are not touched, but goods-lift and dock bookings are a
  preliminaries cost [D].
- **access-egress.** Out of scope in the base: an open-plan refit with no amenities change, and the
  affected part from the principal entrance already complies (stated in the base documents; Premises
  Standards 2010, verified: false). The owner is the applicant, so this must be stated. Exits and
  egress paths must stay usable during the works; that is an impairment and site-management matter, not a
  work item.

---

## 3. Existing-building interfaces

- **Occupation.** Five of six floors stay tenanted, so the sprinkler system must stay protected during
  business hours. The cut-over happens out of hours under a planned impairment [D]:
  - written notice to tenants, fire wardens and the building manager;
  - isolation through the AS 1851 contractor;
  - notice to the alarm monitoring provider and, through it, Fire and Rescue NSW (process verified:
    false);
  - notice to the building insurer;
  - temporary protection (fire watch, or a hired pumpset);
  - a restoration test before the impairment ends.

  The knowledge has the rule but it reaches no generator [K: knowledge/clusters/fire/rules.yaml
  rule.project.fire-system-impairment]. It also has the NSW continuity hold point, as an Implication only
  [K: knowledge/clusters/fire/consequences.yaml cq.nsw-essential-measures-continuity-review].
- **Services cut-overs.** Sprinkler cut-over to the new pumps; electrical shutdown at the MSB for the new
  supply-side connection, usually a short, early-morning whole-building outage [K:
  knowledge/clusters/electrical-comms/consequences.yaml cq.existing-electrical-shutdown-plan-review]. Also
  the fire service isolation while the backflow device is changed, and the level 3 floor sprinkler valve
  isolated for the fit-out.
- **Hazardous materials.** The building is from 1988, so an asbestos register should exist (WHS Regulation
  2025 (NSW; clause numbers verified: false)). A pre-refurbishment survey covers the pump room, the store, the risers and level 3
  before any strip-out [K: knowledge/program/fragments.yaml fr.existing-conditions hazmat activity;
  knowledge/clusters/delivery/consequences.yaml cq.nsw-disturbance-information-review is an Implication].
- **Structure.** The tank's operating weight goes on the basement slab. The engineer checks the slab and
  scans for reinforcement and post-tensioning before coring [D;
  K: knowledge/clusters/fire/interfaces.yaml if.fire-tanks-load-structure].
- **Matching existing work.** New pipework connects to existing valve sets and mains of unknown condition.
  Level 3 ceilings reuse the base-building grid. Detection and EWIS must suit the existing panel's spare
  capacity [K: knowledge/clusters/fire/unforeseen.yaml uc.fire-panel-has-no-spare-loop-capacity].
- **Dilapidation.** Not needed for adjoining properties: there is no excavation. A pre-start photographic
  record of the car park, dock and ramp route used for the tank panels is prudent [D].

---

## 4. Consultants and why

| Discipline (app id) | Stage engaged | Need | App today |
|---|---|---|---|
| Fire services engineer (`consultant.fire_services`) | Investigation | Always: lead designer | Suggested [K: knowledge/works/package_defaults.yaml baselines[3], scope_additions[1]] |
| Structural engineer (`consultant.structural`) | Investigation | Always: tank load, plinths, coring | **Never suggested**: the refurb baselines have no Structural [K: baselines[2], baselines[3]] |
| Electrical engineer (`consultant.electrical`) | Investigation | Always: pump supply, MSB, level 3 | Suggested [K: baselines[2]] |
| Mechanical engineer (`consultant.mechanical`) | Concept | Always: diesel pump room ventilation, level 3 air | Suggested [K: baselines[3]] |
| Certifier (`consultant.certifier`) | Before documentation ends | Always: CDC, fire safety schedule, principal certifier, OC | Suggested [K: baselines[2]]; its RFP blocks (§8) |
| Interior designer (`consultant.interior_design`) | Concept (fit-out) | Always, with the fit-out | Suggested [K: baselines[4]], **and Architect too** [K: baselines[2]] |
| Hazmat consultant (`consultant.hazmat`) | Investigation | Always for a pre-2004 building | Never suggested |
| Fire engineer | Concept | Only with a performance solution, or where existing performance solutions rely on the sprinklers | Never suggested |
| BCA consultant | Concept | Only with a performance solution or a DA fire safety upgrade | Always suggested [K: baselines[3]] |
| Access consultant | Concept | Only if amenities change or an affected-part upgrade applies | Always suggested [K: baselines[3]] |
| Hydraulic engineer | Concept | Only if amenities change | Always suggested [K: baselines[2]] |
| Acoustic consultant | Concept | Only if diesel noise affects tenants or neighbours | Never suggested |
| Town planner | Concept | Only if the tank needs a DA | Suggested only on DA [K: complexity_additions[0]] |
| Heritage consultant | Concept | Only on a heritage item | Suggested on heritage [K: complexity_additions[1]] |

Not separately procured [D]:

- **The competent fire safety practitioner** who assesses each new and altered measure for the final fire
  safety certificate. Usually it is the AS 1851 maintenance contractor under its existing contract. There
  is no discipline id for it.
- **A quantity surveyor.** Optional at this scale; the PM and the engineer's estimate usually suffice.
- **A BCA consultant.** On a deemed-to-satisfy CDC, the certifier's assessment and the engineers' design
  certificates cover compliance. A registered certifier acting as principal certifier may not also author
  the design-side compliance advice (conflict-of-interest rules, verified: false), so a BCA consultant
  appears exactly when design-side compliance work exists: a performance solution (`performance-solution`)
  or a DA fire safety upgrade (`da-fire-upgrade`, and with it `tank-needs-consent`). Reviewer ruling: not
  in the base (§12).

**Two NCC classes.** `det.ncc_class: ["5","7a"]` also adds the industrial category to the package facts
[K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §2.1]. It changes nothing here: `baselines[3]` already
covers commercial, and a fragment's `any_of` matches either class [K: internal/knowledge/relevance.go].

---

## 5. Approvals and authorities

**Pathway.** Complying development under the Codes SEPP. The pump, tank, pump room and fire service works
go under the Fire Safety Code (Part 8, from EPI 2013 No 706; in-force text not read, verified: false).
That Code covers alterations to hydraulic fire safety systems for water-pressure management: fire mains,
fire water storage tanks, on-site fire pumps, booster connections and the pump house or enclosure. General
sprinkler modifications that are not about water pressure are outside it. So the level 3 fit-out,
including its sprinkler, detection and EWIS changes, goes under the Codes SEPP code for internal
alterations to commercial buildings (historically Part 5 Commercial and Industrial Alterations Code; now
possibly Part 5A Industrial and Business Buildings Code; verified: false). Two CDCs is the default;
whether one CDC can rely on both codes: verified: false [D; NSW Planning & Environment Information Sheet
8.1 "Fire Safety Code", January 2015, citing the 2000 Regulation].

The app models one pathway value, `hdr.cond.planning: cdc`, which gives `sk.refurb-traditional-cdc` and
one `fr.cdc-application` [K: knowledge/program/skeletons.yaml sk.refurb-traditional-cdc;
knowledge/program/fragments.yaml fr.cdc-application]. The second (fit-out) CDC has no row. Each CDC
comes after full documentation, which that skeleton's stage order respects (construction documentation
before approvals).

The pathway moves to a DA plus a construction certificate when [D, verified: false]:

- the tank or pump house fails a Fire Safety Code development standard (for example, it sits over a
  registered easement). The Code has standards for water storage tanks and external pump houses, so a
  tank outside the building is not, by that fact alone, a DA trigger;
- the tank takes car spaces that a consent requires;
- the land is excluded from complying development, or the building is a heritage item;
- a performance solution is not allowed under the code.

On a DA for alterations the consent authority may require the existing building's fire safety to be
upgraded (the old cl 94, now in the 2021 Regulation; number verified: false; trigger `da-fire-upgrade`).

**Certificates and statutory steps** (NSW, EP&A Act and the Development Certification and Fire Safety
Regulation 2021; all clause numbers verified: false):

1. Sydney Water **statement of available pressure and flow**, requested through Sydney Water Tap in,
   before design is fixed. FRNSW requires one for any design or alteration of a hydraulic fire safety
   system (FRNSW position statement on hydraulic fire safety system design; product name verified: false).
2. Plans and specifications of the altered sprinkler system and its water supply endorsed by a competent
   fire safety practitioner, lodged with the CDC application (EP&A (DCFS) Regulation 2021, verified:
   false). The Regulation also lists documentation that must accompany an application for an alteration
   to a hydraulic fire safety system (Info Sheet 8.1).
3. CDC determinations. **Fire safety schedule with the fit-out CDC**, listing the existing and proposed
   essential fire safety measures and their standards of performance. The Fire Safety Code CDC may be
   exempt from the schedule and the fire safety certificate (Info Sheet 8.1, citing the 2000 Regulation;
   2021 provision verified: false).
4. Long service levy: 0.25% of the cost of building work of $250,000 or more, including GST (CONVENTIONS
   C4, checked). C4 records it as paid before work starts; paying it before the CDC issues is verified:
   false.
5. Principal certifier appointed, and a notice of commencement given before work starts (period verified:
   false).
6. Sydney Water approval of the fire service and backflow alterations (product name verified: false); the
   testable device registered and tested. A Section 73 compliance certificate is not expected for this
   work (verified: false).
7. **Final fire safety certificate** from the owner, based on a competent fire safety practitioner's
   assessment of each new and altered measure. The owner gives FRNSW a copy with the current fire safety
   schedule and displays both in the building (s 85 per FRNSW "Lodge a fire safety certificate"; verified:
   false). The council receives it with the OC, from the principal certifier. It is needed for the level 3
   measures; the Fire Safety Code works may be exempt (step 3, verified: false). The FFSC rows stay in the
   key until the owner confirms the exemption.
8. **Occupation certificate** for the altered parts, needing the final fire safety certificate (verified:
   false).
9. The **annual fire safety statement** continues each year and from now on covers the new measures. If
   the works span the AFSS due date, a statement still has to be made; the impaired system cannot be
   certified, which is often how these jobs start [D, verified: false].
10. SafeWork NSW notification before any licensed asbestos removal (trigger only; period verified: false).

**FRNSW** [D, verified: false]:

- receives a copy of the final fire safety certificate and the fire safety schedule from the owner;
- requires a statement of available pressure and flow for the design;
- is consulted on any booster or hydrant change (trigger `hydrant-system-upgrade`);
- gets a referral (initial and final fire safety reports) only when a performance solution is proposed
  against a Category 2 fire safety provision (trigger `performance-solution`).

FRNSW does not approve a deemed-to-satisfy sprinkler design.

**Inspections and hold points** [D]:

- the certifier's final inspection before the OC (critical-stage inspections for Class 5 verified:
  false);
- the engineer's hold points: plinth reinforcement before pour; hydrostatic test of new pipework;
  fire stopping before concealment [K: knowledge/clusters/fire/interfaces.yaml
  if.fire-stopping-before-concealment]; level 3 sprinkler pressure test before ceilings close [K:
  if.sprinkler-pressure-test-before-ceilings]; pump acceptance tests (AS 2941, verified: false);
- an integrated test of pumps, flow switches, FIP, EWIS and monitoring [K:
  knowledge/clusters/fire/rules.yaml rule.project.integrated-fire-testing].

Not applicable: BASIX (non-residential); the DBP Act (Class 5 is outside it, owner ruling R3); HBCF
(not residential work, owner ruling R4); a Section J assessment for the pump works [D]. A Section J check
of level 3 lighting may apply (verified: false); it is minor and left out.

---

## 6. Cost plan shape

Lines only, never amounts. The app's elements give one construction line per top-level system
[K: knowledge/costs/elements.yaml elements], and the starters give P01–P07 [K:
knowledge/costs/starter_lines.yaml lines].

**Fees and charges:**

- CDC and certifier fees (two CDCs). The app has P05 "Construction certificate and certifier fees",
  which is the wrong label on a CDC.
- Long service levy: 0.25% of the cost of building work of $250,000 or more, including GST (CONVENTIONS
  C4, checked). No line.
- Sydney Water: statement of available pressure and flow, fire service alteration approval and backflow
  registration (product names verified: false). Only the generic P06 exists.
- No DA fees in the base. The app correctly drops `da-fees` on a CDC.

**Consultants:** fire services engineer, structural engineer, electrical engineer, mechanical engineer,
certifier, interior designer, hazardous materials consultant, and fire safety practitioner assessments.
The app makes a fee line for every adopted package [K: internal/costs/suggest.go Suggest]. Because
cost-plan prepare adopts every suggestion first, it adds Architect, Hydraulic, BCA and Access fee lines
the job does not need [K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §3.2].

**Construction, base-building upgrade** (fire protection contract):

- removal of the old pump, controller and pipework;
- pumpsets and controllers;
- the tank;
- pump room pipework and the connection to the valve sets;
- diesel fuel tank, bunding and exhaust;
- pump room ventilation (local exhaust, still '08 Mechanical services');
- plinths, strengthening and core holes;
- the fire-rated pump room enclosure and fire stopping;
- the fire pump electrical supply;
- the backflow device ('09 Hydraulic services'; mapped to `trade.hydraulic_plumbing` only if that works
  package exists, else the head contract [K: KNOWLEDGE_INDEX §3.2 mappedCostPackage]; in practice the
  fire contractor carries it);
- temporary fire protection during the impairment;
- testing and commissioning;
- preliminaries and occupied-building controls.

**Construction, level 3 fit-out** (fit-out contract): soft strip; interiors and finishes; sprinkler,
detection, EWIS and emergency lighting alterations; air distribution and comms room cooling; tea point
plumbing; lighting and power; structured cabling.

**Contingency:** design contingency, construction contingency, escalation, exclusions, and a provisional
sum for asbestos removal.

Where the app falls short:

- **The two contracts merge.** Base-building and fit-out work on the same element share one line (07, 08,
  09 and 10), though they are different contracts and often different budgets (landlord capex against a
  fit-out budget or tenant incentive) [D].
- **Plant removal is filed as site works.** It prices under "01 Site works and external works".
- **The budget split is wrong for this job.** The `commercial_fitout` split gives active fire protection
  4% and interiors 50% [K: knowledge/costs/budget_splits.yaml commercial_fitout], so a stated budget for
  a pump-and-tank job is spread mostly into interiors. It also has no `structure` or `site` share, so the
  "03 Structure" (plinths, cores) and "01 Site works" (pump removal, soft strip) lines receive nothing
  from a stated budget.
- **Missing lines:** temporary fire protection, testing and commissioning, preliminaries and provisional
  sums.

---

## 7. Program shape

**Skeleton.** `sk.refurb-traditional-cdc`, with stages investigation → concept design → construction
documentation → approvals → procurement → construction → completion → defects [K:
knowledge/program/skeletons.yaml]. That is right for this job.

**Activities that drive the job:**

1. Investigation:
   - collect the fire safety schedule, the last AFSS, the AS 1851 records and the as-built drawings;
   - apply for the Sydney Water pressure and flow statement;
   - flow and pressure test of the existing supply;
   - measured survey, hazmat survey and services trace [K: fr.existing-conditions];
   - structural assessment of the tank location;
   - MSB capacity and maximum demand check.
2. Concept design:
   - the **fire services options report** (supply strategy, pump configuration, tank size and location);
   - the impairment and staging strategy;
   - the owner's approval of the option;
   - in parallel, the fit-out brief, concept and design development [K: fr.concept-design-fitout,
     fr.design-development-fitout].
3. Documentation: fire services design and calculations; the competent fire safety practitioner's
   endorsement of the fire services plans and specifications; structural and electrical design; fit-out
   documentation and sign-off [K: fr.fitout-documentation].
4. Approvals:
   - Fire Safety Code CDC lodgement [K: fr.cdc-application] and fit-out CDC lodgement (no fragment);
   - the long service levy;
   - CDC determination [K: fr.cdc-application], and the fire safety schedule with the fit-out CDC;
   - Sydney Water approval of the fire service alterations.
5. Procurement:
   - two tenders and two awards [K: fr.tender-docs-fitout, fr.tender-period-fitout,
     fr.tender-evaluation-fitout and fr.contract-award each give one];
   - the fire services engineer's review of the pumpset, controller and tank submittals: pump curves
     against the duty points, the commonest cause of a failed acceptance test [K:
     knowledge/clusters/fire/failure_modes.yaml fm.fire-pump-fails-flow-test];
   - then **ordering the long-lead pumpsets, controllers and tank**.
6. Construction, upgrade:
   - principal certifier appointment and notice of commencement;
   - site establishment and the impairment notices;
   - plinths and cores [K: fr.structure-alterations];
   - pump room enclosure;
   - tank assembly and pump installation;
   - pump room ventilation and the diesel exhaust flue (`mechanical.local-exhaust`; no fragment);
   - the fire pump electrical supply;
   - fire stopping and its hold point [K: fr.fire-stopping; if.fire-stopping-before-concealment];
   - the backflow device;
   - the hydrostatic test;
   - **the planned impairment and cut-over**;
   - pump commissioning;
   - removal of the old pump.
7. Construction, fit-out: the fit-out fragments for partitions, mechanical, electrical, hydraulic,
   sprinklers, ceilings, detection, lighting, comms, doors, joinery, painting and floors [K:
   knowledge/program/fragments.yaml suffix fitout], plus the sprinkler pressure-test hold point [K:
   if.sprinkler-pressure-test-before-ceilings]. One-activity fragments print the work item title (the
   system label), not the fragment's activity label [K: internal/program/generate.go instantiate]: for
   example `fr.ceilings-fitout` prints "Ceilings and room heights" (checked by running `program.Generate`
   on the base profile, 4 November 2026). The scorer still matches them by fragment.
8. Joint: the integrated fire system test, then the fire safety practitioner's assessment, the **final fire
   safety certificate**, PC, the certifier's final inspection, the OC and handover. Handover includes
   as-builts, the block plan, the pump logbook and the handover to the AS 1851 contractor.
9. Defects: the defects inspection and liability period [K: fr.defects-inspection,
   fr.defects-liability-end], and the next AFSS.

**Logic links that matter** [D]:

- The flow test and the Sydney Water statement come before the options report, and the options report
  before any design.
- The tank selection comes before the structural design.
- The practitioner's endorsement of the fire services plans comes before both CDC lodgements.
- Both CDC determinations and the contract award come before the start; the submittal review comes before the pumpset
  order.
- The impairment notices, the hydrostatic test, the electrical supply, the backflow device and the
  ventilation all come before the cut-over.
- The cut-over comes before pump commissioning, and commissioning before both the old pump's removal and
  the integrated test.
- The fit-out's sprinkler pressure test comes before its ceilings close.
- The integrated test comes before the practical completion inspection and the final fire safety
  certificate, and the certificate before the OC.

**Long-lead items** [D]: the diesel pumpset and controllers, the electric pumpset, the modular tank, and
any MSB section. There is no lead-time row for fire-water [K: knowledge/program/durations.yaml has no
fire-water row]. No durations in weeks are given here.

**What the app gets wrong:**

- `fire-active.fire-water` has no fragment, so the core sequence (tank, pumps, test, cut-over,
  commissioning) is absent.
- `fr.sprinklers-fitout` is attached after `fr.ceilings-fitout` [K: knowledge/program/fragments.yaml
  fr.sprinklers-fitout attach], while the hold point needs sprinklers before ceilings. The generator drops
  the hold point's link to the ceilings as a loop [K: internal/program/generate.go holdPointLinks,
  addGuarded], so the ceilings are not held. T13 found the same ordering bug.
- `fr.fire-stopping` has no predecessor, so it floats to the start of construction. Its hold point
  precedes every interiors row, including the level 3 partitions.
- The level 3 strip-out rows also stand in for the old pump's removal, at the start of construction,
  when the old pump must in fact stay running until the new system is proven.

---

## 8. RFP shape

**Fire services** (template `fire-services` [K: knowledge/rfp/fire-services.yaml]).

- *Must carry:*
  - evidence review (fire safety schedule, AFSS, AS 1851 records);
  - investigation of the existing supply with a flow test;
  - the Sydney Water statement of available pressure and flow (Tap in), and approval of the fire service
    and backflow alterations;
  - demand and storage calculations;
  - pumpset and tank design;
  - upgrade design;
  - pump room interfaces;
  - the impairment and restoration plan;
  - level 3 wet and dry fire alterations;
  - pump signals to the FIP;
  - tender documentation;
  - construction-phase hold points;
  - commissioning witness;
  - design certificates for the CDC.
- *Must not carry:* fire engineering strategy, special hazard suppression, DA documentation.
- *App differences:*
  - The boundary is always inception to design [K: fire-services.yaml boundary], so documentation,
    construction and commissioning are priced as options.
  - `staging` is always O.
  - Every wet, supply, authority, plant, upgrade, existing and declarations row is `ask` on client flags
    (`fs_*`) that a shape cannot key.

**Structural** (template `structural` [K: knowledge/rfp/structural.yaml]). The package is not suggested,
so this RFP appears only once the user adds it.

- *Must carry:* existing slab assessment; plinth and strengthening design; fixings and seismic restraint
  of the tank, pumps and pipe; coring review with scanning; structural certification for the CDC;
  inspections.
- *Must not carry:* geotechnical brief, retention, Revit model, regulated design, authority asset, CC
  wording.
- *App differences:*
  - A Class 5 part makes the scale complex, so the app includes a shared Revit model.
  - `geotech_brief`, `services_footings` and `concept_schematic` are always included.
  - `services_fixings`, the core of this commission, is always O.
  - The construction phase falls after the boundary end, so it is priced as an option.
  - The construction-documents wording says "construction certificate".

**Electrical** (template `electrical-services`).

- *Must carry:* investigation of MSB capacity and maximum demand; fire pump supply (`plant`);
  distribution; level 3 lighting and data; tender; witness of the supply tests; declarations.
- *Must not carry:* a DNSP application (base), dry fire design (it is the fire engineer's), generation.
- *App differences:* `planning` and `generation` are priced as options when unknown; tender,
  construction and handover fall after the boundary; several rows are `ask` (`elec_*` flags).

**Mechanical** (template `mechanical-services`).

- *Must carry:* investigation of AHU capacity and the existing pump room ventilation; pump room local
  exhaust: diesel combustion and cooling make-up air, plant room exhaust and the engine flue discharge
  (`local`, expected I); coordination of plinths, openings and power for pump room plant
  (`interfaces`); level 3 HVAC; BMS points; tender; commissioning.
- *Must not carry:* smoke control, kitchen exhaust, a planning stage.
- *App differences:* the boundary and the `mech_*` flags, as for electrical; `local` is `ask` on the
  client flag `mech_local`.

**Certifier** (template `bca-certification`). *Must carry:*

- CDC eligibility and assessment under each code: the Fire Safety Code (Part 8) for the pump works and
  the commercial internal alterations code for level 3 (verified: false);
- the fire safety schedule with the fit-out CDC, and whether the Fire Safety Code CDC needs one
  (verified: false);
- principal certifier services and inspections;
- review of the final fire safety certificate;
- the OC.

*Must not carry:* a CC, DA advice, design authorship or building supervision.

*App:* the template lists `cdc` as an unsupported pathway and its B stages need DA or SSD, so the RFP
blocks with `pathway_unsupported` [K: knowledge/rfp/bca-certification.yaml unsupported_pathways;
data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml, which is itself a Class 5 CDC case]. The key
expects the block, as T12 does.

**Live-site rows (all templates).** `partial_occupation` sets the RFP `live_operations` flag to yes [K:
internal/rfp/drivers.go readFlags], so the live-site module rows appear in every RFP. That is correct
for an occupied building.

**Interior designer** (no template; the app issues the generic `tpl.rfp-capex`).

- *Must carry:* brief and test fit; concept and design development to the landlord's fit-out guide; CDC
  drawings; coordination with the fire engineer on head layout and ceilings; tender; site queries.
- *Must not carry:* DA, base-building fire design, BASIX, structure.

**Hazmat** (no template; generic).

- *Must carry:* the survey of the pump room, store, risers and level 3; NATA laboratory analysis; a
  register and management plan update; a removal scope and clearance if anything is found.
- *Must not carry:* contaminated land work.

---

## 9. Triggers

Each trigger is in `shape.yaml` with evidence and negation phrasing. Summary:

| id | Shows up as (incl. indirectly) | Denied by | Adds |
|---|---|---|---|
| `tank-needs-consent` | "two car spaces lost"; "council advised a DA"; "external pump house over a registered easement fails the Fire Safety Code standards" | "fits within the basement store"; "certifier confirmed the Code standards are met" | DA, CC, town planner, BCA; both CDC rows out. App: yes through `hdr.cond.planning`, printing "Lodge the application", "Public exhibition", "Assessment", "Construction certificate" |
| `performance-solution` | "pump room not directly accessible from outside; a performance solution is proposed"; "fire engineering brief" | "complies with the deemed-to-satisfy provisions" | Fire engineer, BCA consultant, FRNSW referral (Category 2, verified: false) |
| `town-main-inadequate` | "the 100 mm main cannot meet the refill rate"; "new connection in George Street with a road opening permit" | "existing connection retained, main adequate" | New connection, road opening, augmentation provisional sum |
| `hydrant-system-upgrade` | "new pumps serve sprinklers and hydrants"; "booster relocated to the frontage" | "hydrants unaffected" | Hydrant upgrade, FRNSW booster consultation, block plan |
| `asbestos-found` | "asbestos gaskets in the pump room"; "vinyl tiles under the level 3 carpet" | "survey found no asbestos" | Licensed removal, SafeWork notice, clearance |
| `diesel-noise-exhaust` | "weekly diesel test run audible in the ground-floor tenancy"; "residential building adjoins the lane" | "two electric pumpsets, no diesel" | Acoustic consultant, `interiors.acoustic-separation` (sound insulation between uses, not room acoustics), treatment, flue |
| `electrical-supply-upgrade` | "MSB has no space for a supply-side connection"; "an Ausgrid application is required" | "MSB has capacity" | MSB section, DNSP application; `fr.switchboard-major` fires on upgrade but prints the generic "Procure and deliver" and "Install", which clash with joinery, so the key scores its program as no |
| `pump-tank-only` | "scope limited to the pump and tank"; "level 3 remains leased" | "fit-out delivered with the upgrade" | Removes the fit-out, the fit-out CDC, the fit-out fire safety schedule and both hold points (no interiors row to hold); tests that the fit-out project fragments wrongly still fire. The schedule and FFSC may not be needed (verified: false) |
| `tenant-procured-fitout` | "tenant will carry out its fit-out under the agreement for lease" | "landlord delivers the fit-out" | Landlord design review; removes the interior designer and fit-out builder |
| `fire-safety-order` | "notice of intention to give a fire safety order"; "AFSS could not be issued for the sprinkler system" | "voluntary capital works" | Order response and compliance period (verified: false) |
| `fit-out-amenities` | "new accessible toilet and two showers" | "toilets not altered" | Access and hydraulic consultants, wet-area and tiling fragments, printed as "Wet-area waterproofing" and "Wall and floor tiling" |
| `heritage-item` | "local heritage item in Schedule 5 of the LEP" | "not a heritage item" | Heritage consultant, referral; CDC unavailable (verified: false) |
| `strata-office` | "The Owners - Strata Plan … (commercial)"; "pump room is common property" | "single ownership" | General meeting and levy |
| `roof-or-crane-tank` | "tank on the roof plant deck"; "panels lifted by crane from George Street" | "carried in through the car park entry" | Roof assessment, crane, road occupancy |
| `affected-part-access` | "the path of travel from the principal pedestrian entrance to level 3 is an affected part and must be upgraded" | "the certifier confirmed the existing path of travel already complies; no affected-part upgrade" | Access consultant, accessible-access upgrade, "13.1 Accessible paths, TGSIs and braille/tactile signage" (Premises Standards 2010, verified: false) |
| `existing-fire-engineered-building` | "operates under a 2006 fire engineering report that relies on the sprinklers for extended travel distances" | "approved to the deemed-to-satisfy provisions; no performance solutions" | Fire engineer review of the existing report (never suggested) |
| `da-fire-upgrade` | "council's consent requires the fire safety measures to be upgraded"; "a fire safety upgrade report is required with the DA" | "proceeds as complying development" | BCA fire safety upgrade report and upgrade works (former cl 94, number verified: false) |
| `interim-occupation` | "tenant requires level 3 before the pump upgrade is complete"; "an interim OC is sought" | "level 3 occupied only after commissioning" | Interim fire safety certificate and interim OC (verified: false), only if level 3 can run on the old supply |

---

## 10. Keyword traps and negations specific to this type

The app reads with regex triggers and then asks Jev. These ordinary phrases in pump-upgrade documents hit
the wrong question:

- **"pump room in the basement"** → `flag.basement` [K: knowledge/profile/flag_readings.yaml basement,
  trigger `\bbasement`]. The question is whether "the project includes a basement". A yes switches on
  structural retention, temporary works interfaces, dilapidation and civil shoring [K:
  knowledge/rfp/structural.yaml positions retention_design, dilapidation], and doubles the substructure
  budget share where a split has it. Expected: **no**; no basement is built or excavated.
- **"connect to the water main in the street"** → `flag.authority_asset` (trigger `(?:sewer|water)
  (?:main|asset)`). A yes adds the structural authority-asset stage and complex scale. Expected: **no**.
- **"fire water storage tank", "diesel fuel storage for the fire pump"** → `flag.fire_special_hazards`
  (triggers `fire[^.]{0,90}storage` and the reverse). The question itself warns against inventing the
  hazard. Expected: **no**.
- **"structural engineer to confirm the slab for the fire tank"** → `flag.fire_specialist_systems`
  (trigger `structural[^.]{0,100}fire`). Expected: **no** in the base.
- **"protect the pump room from flooding", "flood the tank for the hydrostatic test"** →
  `det.flood_hazard` (trigger `\bflood`) [K: knowledge/determinants.yaml flood_hazard]. Expected:
  **false**.
- **"the backflow device prevents contamination of the drinking water supply"** →
  `det.potentially_contaminated_land` (trigger `contaminat`). Expected: **false**.
- **"Stage 1 cut-over", "stage 2 commissioning"** → `flag.staged` (trigger `\bstages? [1-9a-c]\b`). The
  question excludes fee and approval stages but not impairment stages. Expected: **no**.
- **"egress via the fire stairs during the impairment"** → `implied_by stair` →
  `access-egress.stairs-and-ramps` scope → a "Stairs, ramps, handrails and nosings" cost line [K:
  knowledge/profile/scope_defaults.yaml implied_by stair]. Listed in `not_expected_cost_plan`.
- **"tea kitchen"** → `implied_by kitchen`: joinery, hydraulic and `mechanical.local-exhaust`. Local
  exhaust is now in scope (the pump room), so the trap is harmless for the system list; it may still add
  a kitchen-exhaust reading to the level 3 item. "Kitchenette" does not match `\bkitchens?\b`.
- **"Sydney Water asset"** in a Tap in or building-plan letter → `flag.authority_asset` (trigger
  `sydney water asset`) [K: knowledge/profile/flag_readings.yaml authority_asset]. Expected: **no**.
- **"sprinkler"** → `det.sprinklered` (trigger `sprinkler`). It is correctly true here, but every page
  will trigger the question; only an explicit statement should answer it.
- **"lodge"** → `flag.architect_lodges`. CDC lodgement wording will trigger it; no template here reads
  it.
- **Filing:**
  - "Hydraulic and Fire Services Engineer" proposals may file under `consultant.hydraulic`.
  - `trade.fire_services` carries the aliases "Sprinkler" and "Sprinklers" [K:
    data/intake/disciplines.json], so a consultant's "Sprinkler upgrade report" may file as the trade.
  - An AFSS or fire safety schedule has no obvious kind; if it files as correspondence it is never read
    [K: testing/synthetic/DISCOVERY.md §5.2].
- **Negations that must hold:**
  - "the hydrant system is not affected";
  - "no performance solution is proposed";
  - "the existing connection is retained";
  - "not a heritage item";
  - "level 3 toilets are not altered".

---

## 11. Gaps in the built-in knowledge

What is missing, and what the app will probably produce wrongly as a result:

1. **No default scope for refurb** [K: knowledge/profile/scope_defaults.yaml empty_work_types]. Nothing
   appears until documents are read, so a profile miss is a total miss.
2. **No program content for `fire-active.fire-water`** [K: testing/synthetic/reference/index/program-fragments.md].
   The app will produce a fit-out program with no pump, tank, test, cut-over or commissioning rows.
3. **No NSW fire safety process.** There is no fire safety schedule, final fire safety certificate, AFSS,
   competent fire safety practitioner or FRNSW step. The OC does not depend on the fire safety
   certificate. The only trace is an Implication [K: knowledge/clusters/fire/consequences.yaml
   cq.nsw-essential-measures-continuity-review] and a rule note [K: knowledge/clusters/fire/rules.yaml
   rule.as1851.fire-system-maintenance].
4. **No impairment management** in program, cost or procurement. The rule exists but reaches no generator.
5. **The hold point ordering is wrong** (sprinkler test against ceilings; fire stopping floating), as in
   §7.
6. **Structural and Hazmat are never suggested on refurb.** The app suggests Architect, Interior Design,
   Hydraulic, BCA and Access on a job that needs only one fit-out designer and no BCA or access
   consultant.
7. **One head contract only.** Two works contracts are normal here; trade packages are never suggested
   [K: internal/procurement/proposed.go ProposeHeadContract].
8. **The cost plan merges contracts by element**, and the `commercial_fitout` split spreads a pump-heavy
   budget mostly into interiors. P05 has a CC label on a CDC. There are no long service levy, Sydney
   Water, temporary protection, testing or preliminaries lines.
9. **The certifier RFP blocks on a CDC**, so the approval authority cannot be briefed.
10. **The fire services RFP prices documentation, construction and commissioning as options**, because
    the boundary always ends at design.
11. **The structural RFP is set for a new building.** It includes a Revit model and a geotechnical brief,
    and prices services fixings as an option.
12. **Fit-out project fragments fire on any Class 5 refurb**, so a pump-only job gets fit-out concept and
    design rows.
13. **Missing fragments.** `mechanical.air-distribution`, `mechanical.local-exhaust` (pump room
    ventilation), `fire-active.occupant-warning`, `fire-active.emergency-lighting`,
    `hydraulic.backflow-prevention` and `fire-passive.rated-elements` have none, and
    `fr.switchboard-major` ignores `alter`. Running `program.Generate` on the base profile (4 November
    2026) confirmed that no ventilation row fires.
14. **The flow-test Implication is keyed to the wrong system.** `uc.existing-fire-water-fails-flow-test`
    attaches to fire-water but is triggered by sprinkler or hydrant works [K:
    knowledge/clusters/fire/unforeseen.yaml uc.existing-fire-water-fails-flow-test when]. It fires here
    only because the sprinkler connection is `alter`. A pump job that leaves sprinklers and hydrants
    untouched gets none.
15. **Two CDCs cannot be modelled.** One project has one planning value, and the Fire Safety Code's scope,
    its possible schedule and certificate exemption, and the practitioner's endorsement of plans are not
    encoded.
16. **Missing discipline ids:** the fire safety practitioner, the AS 1851 contractor, the alarm monitoring
    provider and a Sydney Water listed provider.
17. **Keyword traps** in §10, chiefly basement, authority asset, special hazards and stairs.
18. **Wrong AS 2941 edition.** `rule.as2941.fire-pumps` cites AS 2941:2014; the current edition is AS
    2941-2013 (Standards Australia store). Correct the instrument; clauses stay `clause_verified: false`.

---

## 12. Sources and unverified claims

**Knowledge files opened:**

- `knowledge/clusters/fire/systems.yaml`, `rules.yaml`, `interfaces.yaml`, `unforeseen.yaml`,
  `consequences.yaml`, `failure_modes.yaml`;
- `knowledge/works/package_defaults.yaml`, `actions.yaml`;
- `knowledge/profile/scope_defaults.yaml` (via the index), `flag_readings.yaml`, `taxonomy.yaml`;
- `knowledge/determinants.yaml`;
- `knowledge/costs/starter_lines.yaml`, `elements.yaml` and `budget_splits.yaml` (via the index);
- `knowledge/program/fragments.yaml`, `skeletons.yaml`;
- `knowledge/rfp/fire-services.yaml`, `bca-certification.yaml`, `structural.yaml`,
  `electrical-services.yaml`, `mechanical-services.yaml`;
- `knowledge/clusters/services-wet-air/systems.yaml`, `rules.yaml`;
- `data/intake/disciplines.json`;
- `data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml`;
- Go: `internal/program/generate.go` (instantiate, holdPointRows, holdPointLinks, addGuarded),
  `internal/rfp/drivers.go` (deriveFlags, readFlags), `internal/profile/reconcile.go` (derive),
  `internal/profile/works.go` (work item title is the system label);
- Go run (4 November 2026, read-only; a throwaway program deleted after use): `program.Generate` on the
  base, `pump-tank-only`, `fit-out-amenities`, `electrical-supply-upgrade` and `tank-needs-consent`
  profiles. It confirmed the DA, wet-area, tiling and switchboard labels, that no ventilation row fires,
  and that `pump-tank-only` gets no hold point;
- the reference index files under `testing/synthetic/reference/`.

**Secondary sources (support a ruling; do not set `verified: true`):**

- NSW Planning & Environment, Information Sheet 8.1 "Fire Safety Code" (January 2015): the Code's scope
  and the schedule and certificate exemption (cites the 2000 Regulation; the 2021 equivalent is not
  confirmed). The Part 8 number comes from the as-made amending instrument EPI 2013 No 706, Sch 1, with
  conditions in Schedule 10 (reviewer's reading).
- FRNSW, "Lodge a fire safety certificate" (fire.nsw.gov.au/page.php?id=9158), citing s 85 of the EP&A
  (DCFS) Regulation 2021; FRNSW position statement on hydraulic fire safety system design.
- Long Service Corporation, About the levy (longservice.nsw.gov.au/bci/levy/about-the-levy); the levy
  itself is now cited as checked under CONVENTIONS C4.
- Standards Australia store: AS 2941-2013 current, superseding 2008.

**Unverified (owner or primary source to confirm):**

- AS 2118.1 and AS 2941 clause content, and the pump room separation and access requirements. The current
  edition is AS 2941-2013 (Standards Australia store), not the 2014 that `rule.as2941.fire-pumps` cites;
  clauses stay `clause_verified: false`;
- the Codes SEPP part numbers: the Fire Safety Code as Part 8, and the fit-out code (the Codes SEPP code
  for internal alterations to commercial buildings: historically Part 5 Commercial and Industrial
  Alterations Code; now possibly Part 5A Industrial and Business Buildings Code); whether one CDC can
  rely on both codes; and when lost car spaces or a failed Code standard force a DA;
- the Development Certification and Fire Safety Regulation 2021: fire safety schedule and its exemption
  for a Fire Safety Code alteration, final fire safety certificate (s 85), AFSS, competent fire safety
  practitioner and the endorsement of plans, Category 2 fire safety provisions, FRNSW referral, notice of
  commencement, critical-stage inspections for Class 5, interim certificates, and the former cl 94
  upgrade power;
- the Disability (Access to Premises — Buildings) Standards 2010 affected-part provisions and the lessee
  concession;
- certifier conflict-of-interest rules on design-side compliance advice;
- the Sydney Water product names (statement of available pressure and flow through Tap in is the
  reviewer's ruling; the fire service and backflow approval product name is not named), backflow
  registration and testing, listed providers, and that no Section 73 certificate is needed;
- whether the long service levy must be paid before the CDC issues (C4 says before work starts);
- AS/NZS 3000 supply-side connection and fire-rated cabling for fire pumps; AS/NZS 3013;
- the WHS Regulation 2025 (NSW; clause numbers verified: false) asbestos register duty for pre-2004 workplaces, and the SafeWork NSW notification
  period;
- the fire safety order provisions, and whether work under an order needs a CDC or CC;
- the AS 4000 defects liability period length.

**Judgement calls for the owner:**

- **BCA consultant not in the base.** Reviewer ruling: not in the base. Owner may overrule. Many Sydney
  fit-out CDCs do carry one; if the owner says it is usual, move `consultant.bca` from
  `not_expected_procurement` into `procurement` with `built_in: yes`.
- **Fire Safety Code exemption.** The FFSC and schedule rows stay for the pump works until the owner
  confirms the exemption.
- **Two works contracts.** Some owners let one fit-out builder carry the fire upgrade as a subcontract.
- **Diesel standby in the base.** A second electric pumpset on a separate supply is a common alternative;
  `diesel-noise-exhaust` covers its effects.
- **Fire services engineer designs the level 3 dry fire.** Some projects give detection and EWIS to the
  electrical engineer.

---

## Shape edits after the ST06 scenario review (7 November 2026)

Applied from `testing/synthetic/scenarios/ST06-base/REVIEW.md` (findings 4, 10 and 11), with the reviewer's agreement:

- `diesel-noise-exhaust` negation: the old wording ("two electric pumpsets ... no diesel engine") contradicted the
  base's diesel standby pump. It now reads "The diesel exhaust discharges through the existing basement plant exhaust
  shaft, clear of tenancies, air intakes and neighbours; the weekly test run is not audible in the tenancies."
- `pump-tank-only` profile: `[hdr.cond.operational_constraints, proj.budget_stated]`. With no fit-out every floor is
  tenanted (`live_environment`; keys accept `partial_occupation` too), and the stated budget loses the fit-out's share.
- `tenant-procured-fitout` profile: `[proj.budget_stated]`, for the same budget reason.

## Shape corrections after run `20261107-t03-t04-t06` (7 November 2026)

From FINDINGS "Shape and key gaps" for that run, and owner ruling R19.

- Certifier RFP re-keyed for F9, as the T12 certifier was on 6 November. It expects `b2_cdc`, `b3_construction`
  and `b4_occupation` included, `b1_preliminary` option (as T11), `b2_construction_approval` absent and no
  blocking check (was `pathway_unsupported`, every stage absent). The CDC, principal certifier and OC phrases are
  keyed to those stages. The `text.cdc_scope` gap is removed.
- Fire services, electrical and mechanical `position.existing`: `app_differs` now says `app: absent` (was `ask`),
  as the T03 keys do from a Go run. The row is left out while the investigation stage is ask.
- `tank-needs-consent`:
  - sets `skeleton: sk.refurb-traditional-da`;
  - adds "Town planner" to `adds.cost_plan`;
  - restates the certifier `expect` (`a2_planning` and `b2_construction_approval` included, `b2_cdc` absent);
  - adds a BCA RFP expectation with `a2_planning` included. A1 and A3 are included and A4 is option. The B stages
    are absent with `app_differs` option, as the T03 BCA key has them: the CC is the certifier's.
  The CDC phrases leave the certifier lists.
- `pump-tank-only` keeps the generic design rows ("Confirm the brief", "Prepare the concept design", "Client review
  of the concept", "Develop the design"): they are also the fire services design sequence. It removes only
  "Prepare fit-out documentation".
- `hdr.cond.quality_level` applies to residential only (`taxonomy.yaml` `applies_to`), so the ST06 keys mark it
  `not_read`.
- R19: the ST06 keys accept `urban_constrained` or `restricted_hours` for `hdr.cond.access_constraints`
  (`operational_constraints` already accepted both occupation values).

## Shape correction after run `20261107-t03-t04-t06-after-fixes` (7 November 2026)

- The sprinkler fit-out is two rows, as in T03. Since the F44 fix `fr.sprinklers-fitout` makes "Alter sprinkler
  pipework above the ceiling" (before the pressure-test hold point and the ceilings) and "Alter and test sprinklers"
  (heads fitted off and tested after the ceilings). The shape had one row, "Alter and test sprinklers", meaning the
  pipework, with the hold point after it. The scorer paired that label with the app's fit-off row, so the correct
  app order (pipework, hold point, ceilings, fit-off) scored "violated" in ST06-base and tank-needs-consent. The
  pipework row is now "Alter sprinkler pipework above the ceiling", the hold point follows it, and "Alter and test
  sprinklers" follows "Install ceilings". The gap "the hold point's link to the ceilings is dropped as a loop" is
  removed: the re-run shows the link held, with no `link_skipped` note. Both runs were re-scored.
