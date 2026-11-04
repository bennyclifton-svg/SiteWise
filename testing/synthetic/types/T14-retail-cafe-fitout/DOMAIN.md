# T14 Retail or café fit-out (kitchen, grease trap): domain map

Status: draft for owner review (`reviewed: false`). Default state NSW. The machine-readable key is
`shape.yaml` in this folder. It passes `testing/synthetic/tools/validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Rule numbers I could not confirm against the
instrument are marked `verified: false`.

---

## 1. The job

**What it is.** A tenant fits out an existing shop as a café with a commercial kitchen. The building is
NCC Class 6 [K: knowledge/profile/taxonomy.yaml food_beverage `ncc_class: '6'`]. The **base scenario**:

- a vacant ground-floor shop of about 120 m² in a single-storey 1960s strip of shops on a Sydney local
  main street;
- the previous use was a non-food shop, so the café is a new use and a DA is lodged for the food and
  drink premises use, the kitchen exhaust and the signage;
- about 36 seats inside and no outdoor seating;
- a commercial kitchen of about 25 m² with all-electric cooking (combi oven, flat grill, fryer), so an
  exhaust hood is needed. It discharges vertically through the tenancy's own roof;
- a new in-ground grease arrestor at the rear, draining by gravity to the existing sewer junction;
- an accessible unisex WC replaces the old staff toilet;
- new shopfront glazing and entry door, fascia and under-awning signs;
- level entry from the footpath; the building is not sprinklered, not heritage, not flood or bushfire
  affected. [D]

**Scale.** $0.1–1M (PROJECT_TYPES.md). This base sits in the lower half. The tenant usually buys the
kitchen equipment directly, outside the builder's contract, and that equipment can cost as much as the
kitchen's share of the fit-out. [D]

**Who procures it and how.** The client is the café operator, who is the tenant. The landlord owns the
building and must approve the design and sign owner's consent for the DA. The usual small team [D]:

- an **interior designer** (sometimes an architect, never both) leads design, prepares the DA drawings,
  documents the fit-out and runs a select tender;
- a **commercial kitchen designer**, often the equipment supplier, lays out the kitchen and schedules
  the equipment and its services loads;
- a **certifier** issues the construction certificate (CC) and the occupation certificate (OC) for the
  new use, and assesses NCC and access compliance itself on a job this size;
- a **town planner** is optional. The designer often writes the statement of environmental effects
  (SEE) for a simple café DA;
- a **hazardous materials surveyor**, unless the landlord's asbestos register already covers the areas
  to be disturbed;
- a **shopfitter** builds under a lump-sum contract (ABIC or a similar commercial minor-works form). The
  kitchen-exhaust contractor, the licensed plumber (who also sizes the grease arrestor and applies to
  Sydney Water) and the electrician **design and certify their own work** as subcontractors.

There is normally no mechanical, hydraulic, electrical, fire services, BCA or access consultant.
**Over-procurement is the main way the app gets this job wrong** (§4, §11).

**What makes it go wrong** [D]:

- **The grease arrestor has nowhere to go**, or the sewer junction is too shallow to drain it by
  gravity. This is found late if no one checks the sewer diagram before design.
- **The exhaust route.** A riser through floors above, residents objecting to odour, or a roof that
  cannot carry the fan. Each one turns a contractor-certified item into an engineered one.
- **Power.** An old single-phase supply that cannot carry the kitchen, with a long DNSP lead time.
- **The landlord.** Slow design approvals, lease conditions on contractors and working hours, and
  make-good at the end of the lease.
- **Change of use.** Council or the certifier requires a fire safety or access upgrade of the existing
  building, or extra toilets for the seat numbers.
- **Opening date.** Rent-free periods end on a fixed date. The council food premises inspection, the
  Sydney Water permit and the OC must all land before trading.
- **Kitchen equipment** arriving late, or arriving with loads and connections the services were not
  designed for.

## 2. Systems and work items

Refurbishment starts with **no default work items** [K: knowledge/profile/scope_defaults.yaml
empty_work_types], and the refurb default action is `alter` [K: knowledge/works/actions.yaml
work_type_defaults]. The items below are the base scope the documents must establish.

| system | in scope (action) | out of scope, and why |
|---|---|---|
| `site` | `site.demolition` remove (strip-out of the old retail fit-out); `site.signage` new (fascia and under-awning signs; [K: knowledge/clusters/structure/systems.yaml site.signage]); `site.waste-storage` new (bins incl. food organics, to the DA waste plan) | No earthworks, landscape or stormwater: no external works. `site.public-domain` only with footway dining (trigger). No hoarding on the base job. |
| `substructure` | none | Existing slab retained; saw-cutting for the arrestor and floor wastes is part of the hydraulic work. |
| `structure` | none in base | `structure.alterations` only when an exhaust riser crosses floors or the roof needs strengthening for the fan (trigger). |
| `envelope` | `envelope.windows-doors` alter (shopfront glazing and accessible entry door); `envelope.roof-coverings` alter (exhaust penetration, flashing, fan plinth) | No external wall, insulation or facade work. Section J envelope does not apply to a fit-out that leaves the envelope unchanged [D; verified: false]. |
| `interiors` | walls-linings, ceilings, floor-finishes, tiling, wet-area-waterproofing, joinery, internal-doors, painting, all new; `interiors.ffe` new (tenant-supplied kitchen equipment, coffee machine, refrigerated display, furniture) | Acoustic separation only with dwellings above (trigger). Kitchen surfaces must be smooth, impervious and cleanable, and floors coved (FSANZ Standard 3.2.3; AS 4674-2004; verified: false). |
| `fire-passive` | `fire-passive.penetrations` new (services through walls between tenancies) | Rated shaft only for a riser (trigger). |
| `fire-active` | `fire-active.portable-extinguishers` new (and a fire blanket at the cooking line; AS 2444, verified: false [K: knowledge/clusters/fire/rules.yaml rule.as2444.extinguishers]); `fire-active.emergency-lighting` new (exit signs and emergency lighting; AS 2293.1, verified: false) | No sprinklers, detection or hydrants in an unsprinklered single-storey shop; a centre tenancy changes this (trigger). Kitchen hood wet-chemical suppression is not usually required for a café [D; verified: false]. |
| `mechanical` | `mechanical.local-exhaust` new (kitchen hood and roof fan, WC exhaust; the system covers commercial kitchen hoods [K: knowledge/clusters/services-wet-air/systems.yaml mechanical.local-exhaust]); `mechanical.ventilation` new (kitchen make-up air [K: knowledge/clusters/services-wet-air/interfaces.yaml if.kitchen-exhaust-makeup-air]); `mechanical.air-conditioning` new (dining area) | No central plant or BMS. Hood design to AS 1668.2 (verified: false). |
| `hydraulic` | `hydraulic.trade-waste` new (grease arrestor, trade waste drainage, sampling point [K: knowledge/clusters/services-wet-air/systems.yaml hydraulic.trade-waste]); `hydraulic.sanitary` alter; `hydraulic.cold-water` alter; `hydraulic.hot-water` new; `hydraulic.backflow-prevention` new (containment and individual protection for coffee and ice machines) | Gas only with gas cooking (trigger). No roof drainage change. |
| `electrical` | `electrical.switchboards` upgrade (tenancy distribution board); `electrical.cabling-wiring` alter; `electrical.lighting` new (Section J lighting power, verified: false) | `electrical.utility-supply` only when the supply must grow (trigger). |
| `comms-security` | `comms-security.structured-cabling` new (POS and data) | CCTV and alarms are usually the tenant's own supply. |
| `vertical-transport` | none | Ground floor. |
| `access-egress` | `access-egress.accessible-facilities` new (accessible unisex WC; AS 1428.1, verified: false [K: knowledge/clusters/fire/rules.yaml rule.ncc.accessible-sanitary]); `access-egress.accessible-access` alter (entry and path of travel to the new part); `access-egress.egress-doors` alter | Ramps only with a step at the entry (trigger). |

## 3. Existing-building interfaces

- **Occupation.** The tenancy is vacant. Neighbouring shops keep trading, so noisy work, deliveries and
  skip bins need managing on a main street. Base `hdr.cond.operational_constraints: vacant`. [D]
- **Landlord's works and tenant's works.** The lease and fit-out guide set what the landlord provides
  (shell, supply, sewer junction, any existing arrestor) and what the tenant builds, the landlord's
  approvals, any nominated contractors, and make-good at the end of the lease. [D]
- **Services cut-overs.** The tenancy's water, sewer and power are isolated within the tenancy. The
  sewer junction depth decides whether the arrestor drains by gravity. Supply capacity decides whether
  the kitchen needs an upgrade. [D]
- **Hazardous materials.** A 1965 building is likely to contain asbestos (vinyl tiles, fibre-cement
  sheet, switchboard backing). A workplace built before 2004 must have an asbestos register, and the
  areas to be disturbed must be surveyed before refurbishment (WHS Regulation 2017 (NSW); clauses
  verified: false). The knowledge has the pre-2004 consequence as an Implication only [K:
  testing/synthetic/reference/KNOWLEDGE_INDEX.md §6, `cq.pre-2004-fabric-hazardous-materials-survey`].
- **Structure.** Roof framing must carry the exhaust fan and canopy hangers [K:
  knowledge/clusters/services-wet-air/unforeseen.yaml uc.kitchen-canopy-heavier-than-existing-roof-framing].
  Saw-cutting the slab for the arrestor and floor wastes may hit services. [D]
- **Fire separation.** Walls between tenancies may be fire-rated, so every penetration is fire-stopped
  and recorded before concealment [K: knowledge/clusters/fire/interfaces.yaml
  if.fire-stopping-before-concealment].
- **Matching existing work.** Shopfront, awning and fascia, especially in a conservation area
  (trigger). [D]
- **Dilapidation.** A photographic record of the neighbouring tenancies and the footpath before work is
  good practice; a formal survey is not usual at this scale. [D]

## 4. Consultants and why

| discipline | stage engaged | need | why |
|---|---|---|---|
| Interior designer (`consultant.interior_design`) | brief | always | Design lead: concept, DA drawings, documentation, landlord submissions, tender. The app suggests it [K: knowledge/works/package_defaults.yaml baselines[4]]. |
| Certifier (`consultant.certifier`) | concept | always | CC, principal certifier, inspections, fire safety schedule, OC for the new use [K: knowledge/works/package_defaults.yaml baselines[2]]. Engaged at concept: on a change of use the B1 preliminary assessment of fire safety and access upgrade risk must come before DA lodgement. |
| Town planner (`consultant.town_planner`) | concept | optional | SEE and plan of management. Suggested because the pathway is DA [K: knowledge/works/package_defaults.yaml complexity_additions[0]]. |
| Hazmat (`consultant.hazmat`) | investigation | usually | Refurbishment asbestos survey before strip-out. No package rule names it. [D] |
| Commercial kitchen designer | concept | usually (often the supplier) | Kitchen layout, equipment schedule, hood size and services loads; also gives the exhaust concept for the DA. **There is no discipline id** [K: data/intake/disciplines.json], and `supplier.*` rows are refused, so `shape.yaml` keys it as `discipline: none` and records a gap. [D] |
| Shopfitter (`trade.main_works`) | procurement | always | Lump-sum fit-out contract with D&C services subcontracts. Proposed only from gaps on accepted work items [K: internal/procurement/proposed.go ProposeHeadContract]. Under the traditional route the proposal carries install lines only, so the subcontractors' design of the exhaust, trade waste and electrical stays an open gap (`BuilderDesigns` is false). |

**Conditional** (triggers): mechanical engineer (exhaust riser through other floors), structural
engineer and acoustic consultant (same trigger), hydraulic engineer (arrestor with no room or no
gravity fall), access consultant (step at the entry), BCA consultant and fire engineer (fire safety
upgrade on change of use), heritage consultant (conservation area).

**Not expected, though the app suggests them** [K: testing/synthetic/reference/index/type-predictions.md
T14]: Architect (a second design lead), BCA (duplicates the certifier), Mechanical, Hydraulic,
Electrical and Fire Services (D&C trades certify their own work), and Access (level entry; the certifier
assesses). That is seven of the app's ten suggestions. Mechanical, Hydraulic and Electrical come from two
rule sets: the refurb baselines, and again from `scope_additions[2]`–`[4]` once any services item is in
scope [K: knowledge/works/package_defaults.yaml scope_additions], so a fix is needed in both.

## 5. Approvals and authorities

**Pathway.** A DA to council for the change of use from a shop to food and drink premises, with the
kitchen exhaust, shopfront and signage (EP&A Act 1979; the use term from the council's Standard
Instrument LEP; verified: false). Some café fit-outs qualify as complying development under the Codes
SEPP Commercial and Industrial Alterations Code, or as exempt where the use does not change (State
Environmental Planning Policy (Exempt and Complying Development Codes) 2008; verified: false). That is
the `cdc-pathway` trigger. Signage is in the DA unless it meets the exempt standards (SEPP (Industry and
Employment) 2021 Chapter 3; verified: false). [D]

**DA documents.** Drawings, SEE, plan of management (hours, seats, odour, noise, deliveries), waste
management plan, kitchen exhaust discharge details (a concept from the kitchen designer or an exhaust
contractor engaged early: hood size, duct route, fan position, discharge height and distance to
neighbours), signage details, owner's consent signed by the
landlord. Council's environmental health officer reviews the kitchen against FSANZ Standard 3.2.3 and
AS 4674 (verified: false). [D]

**Certificates.**

- Construction certificate, after the subcontractors' exhaust and plumbing design certificates and the
  Sydney Water Tap in approval. [D]
- Notice of commencement and appointment of the principal certifier before work. [D]
- Fire safety schedule with the CC, and a final fire safety certificate for the new essential fire
  safety measures (exit signs, emergency lighting, extinguishers) before the OC (EP&A (Development
  Certification and Fire Safety) Regulation 2021; verified: false). [D]
- Occupation certificate for the new use before trading. [D]
- Owner rulings R4 (HBCF) and R3 (DBP Act) do not apply: this is not residential building work and not
  a Class 2 or 9c building [K: testing/synthetic/reference/OWNER_RULINGS.md R3, R4].
- Long service levy before the CC, only when the cost of works reaches the threshold; the scenario
  budget decides (Building and Construction Industry Long Service
  Payments Act 1986; threshold verified: false). [D]

**Authority applications.**

- **Sydney Water Tap in building plan approval** of the approved plans before the CC, which DA consents
  routinely require; a new in-ground arrestor and drainage make it likely (verified: false). It is
  separate from the trade wastewater permit. [D]
- **Sydney Water trade wastewater permit** for a food business, with the grease arrestor sized to Sydney
  Water's requirements; no discharge before the permit (verified: false). The knowledge has the rule,
  but it never applies (§11) [K: knowledge/clusters/services-wet-air/rules.yaml
  rule.water-authority.trade-waste-agreement].
- **Food business notification** to the NSW Food Authority and council food premises inspection before
  trading (Food Act 2003 (NSW); verified: false). [D]
- **Plumbing**: licensed plumber, notice of work and certificate of compliance to NSW Fair Trading;
  testable backflow device report (Plumbing and Drainage Act 2011; verified: false) [K:
  knowledge/clusters/services-wet-air/rules.yaml rule.state.licensed-plumbing-and-compliance-certificate].
- **Landlord**: approval of concept and documentation; owner's consent for the DA (Retail Leases Act
  1994 (NSW) for disclosure of lessor's works; verified: false). [D]
- **Council**: skip bin and footpath occupancy permits during works. [D]

**Inspections and hold points.** Owner ruling R7 lists critical-stage inspections for Class 1 and 10
only. For a Class 6 fit-out the CC lists the inspections; expect at least a final inspection before the
OC (EP&A (Development Certification and Fire Safety) Regulation 2021; verified: false). Plumbing
inspections of in-ground drainage before backfill. Fire stopping inspected and recorded before
concealment (an app hold point). Exhaust air-flow commissioning before the fire safety and mechanical
certificates. [D]

## 6. Cost plan shape

**Fees and charges:** DA fees and contributions (s7.12 levy where applicable; verified: false);
construction certificate and certifier fees; service authority connection fees; Sydney Water Tap in
approval fee; Sydney Water trade wastewater permit and charges; long service levy (only above the
threshold); council food premises inspection and registration
fees; landlord's design review and base-building contractor fees; council permits during works.

**Consultants:** interior designer; certifier (one line only, not also P05); town planner (optional);
commercial kitchen designer; hazardous materials survey.

**Construction (by element):** preliminaries and builder's margin; strip-out and make good; site works
(signage, bin store); building envelope (shopfront, roof penetration); interiors and finishes; joinery
(counter, coffee bar, back bar); passive fire protection; active fire protection; mechanical (kitchen
exhaust, make-up air, AC); hydraulic (grease arrestor, trade waste, sanitary, hot water, backflow);
electrical; communications; access and egress children (accessible WC, accessible path and entry,
egress door hardware); provisional sums and PC items.

**Contingency and other:** design contingency; construction contingency; escalation; exclusions; owner
supply (kitchen equipment, coffee machine, furniture, smallwares). The documents must say the tenant's
supplier supplies and installs the equipment outside the contract (provider owner), or the same
equipment is costed twice: once in interiors from the `interiors.ffe` item and again as owner supply.

What the app produces [K: testing/synthetic/reference/index/cost-plan.md; knowledge/costs/starter_lines.yaml;
knowledge/costs/elements.yaml]: P01–P07 including DA fees; ten F-lines, seven of them not expected;
one construction line per top-level system with a work item, and `access-egress` itemised. A stated
budget spreads by `commercial_fitout`, an office split with no site or envelope share and only 2% for
hydraulic [K: knowledge/costs/budget_splits.yaml commercial_fitout]. No amount is invented.

## 7. Program shape

**Skeleton:** `sk.refurb-traditional-da`: investigation, concept design, approval documentation,
approvals, construction documentation, procurement, construction, completion, defects [K:
knowledge/program/skeletons.yaml sk.refurb-traditional-da].

**Driving activities and logic** [D, with app fragments where named]:

1. Lease and fit-out guide review; measure-up, hazmat survey and services trace (`fr.existing-conditions`);
   planning certificate; **sewer diagram check for the arrestor**.
2. Brief, kitchen layout and equipment schedule, **kitchen exhaust concept and discharge location**,
   concept, client review, **landlord's concept approval**, design development (`fr.concept-design-fitout`, `fr.design-development-fitout`).
3. DA documents and owner's consent. **No fit-out approval-documentation fragment exists**, so the app's
   approval documentation stage is empty.
4. Lodge, notify, assess (`fr.da-lodgement`, `fr.da-exhibition`, `fr.da-assessment`).
5. Fit-out documentation (`fr.fitout-documentation`), landlord approval, tender sign-off; apply for the
   Sydney Water trade wastewater permit.
6. Tender and award (`fr.tender-*-fitout`, `fr.contract-award`); subcontractors' exhaust and plumbing
   design certificates; Sydney Water Tap in approval; long service levy where it applies; **then** the CC;
   notice of commencement. The key places these in procurement; the app puts the CC in approvals, before
   documentation.
7. Construction: establish; bin store; strip-out (`fr.strip-out-fitout`, needs action remove); arrestor and trade
   waste drainage; rough-ins; exhaust duct and roof fan; make-up air; fire stopping and its hold point;
   partitions, wet areas, ceilings; shopfront; **coved kitchen floor before joinery and equipment** (the app
   lays floors after painting); joinery; painting; kitchen equipment; fit-offs; accessible WC fittings; exit and emergency lighting, extinguishers;
   signage; commissioning and air-flow tests; backflow test and plumbing certificates.
8. Completion: PC; final fire safety certificate; final inspection; OC; Sydney Water arrestor inspection
   and permit; food business notification and council inspection; handover; **open for trade**.
9. Defects inspection, defects liability period (length verified: false), final defects inspection.

**Long-lead items:** fabricated stainless exhaust canopy and fan, imported kitchen equipment, joinery,
shopfront glazing, the tenancy switchboard; a DNSP supply upgrade (trigger) can be the longest. [D]
Durations are not quoted; the app's own come from `knowledge/program/durations.yaml`.

## 8. RFP shape

**Interior designer** (generic `tpl.rfp-capex`; no template [K:
testing/synthetic/reference/index/rfp-templates.md, disciplines with no RFP template]). Must carry:
measure-up; concept and design development with the kitchen designer's layout; DA drawings for the
change of use, exhaust and signage; fit-out documentation, finishes schedule and specification; food
premises finishes; landlord submissions; performance briefs for the D&C exhaust, plumbing and electrical
trades; tender support; contract administration as an option. Must not carry: base-building design,
BIM coordination, Green Star or NABERS, engineering design, novation.

**Certifier** (`bca-certification`). Must carry B1 preliminary assessment of the change of use, B2 CC,
B3 principal certifier services, B4 OC and fire safety certificate, an existing-building assessment
limited to the affected part, and performance-solution assessment as an option. Must not carry the
advisory A-stages as the appointment, residential BASIX evidence, staged occupation or fire referrals.
In the app, every stage (A1–A4 and B1–B4) asks until the client picks Package B, because no boundary
resolves while `bca_advisory` and `bca_certification` are unknown, and the `fact:flag.bca_*` checks block
issue [K: knowledge/rfp/bca-certification.yaml boundary; data/eval/rfp/answer-keys/bca-unknown-decisions.yaml].

**Town planner** (`town-planning`, optional). Must carry preliminary advice on the use definition and DCP
controls, an SEE with plan of management, lodgement, assessment responses, and certificate and prior
consent searches. Response to submissions priced as an option. Must not carry a clause 4.6 request,
panel attendance, SSD or specialist reports.

**Conditional RFPs** (triggers): mechanical (`position.kitchen`: hood, fire-rated duct, make-up air,
discharge; needs the client flag `mech_kitchen` [K: knowledge/rfp/mechanical-services.yaml kitchen]);
hydraulic (`position.trade`: arrestor sizing and trade waste; needs `hyd_trade` [K:
knowledge/rfp/hydraulic-services.yaml trade]); access (existing entry, performance solution, approval
report).

## 9. Triggers

| id | how it shows up (direct and indirect) | how a document denies it | what it adds |
|---|---|---|---|
| `cdc-pathway` | "complying development certificate"; "the certifier can approve it under the code" | "a DA will be lodged"; "the exhaust does not meet the code, so a DA is required" | CDC rows; landlord's consent for the CDC; removes town planner, DA fees, DA rows and the CC; cc-fees stays and carries the CDC fee; exhaust and arrestor design move before tender; the certifier RFP blocks on cdc |
| `existing-food-premises` | "previously a café"; "existing 1,000 L arrestor in the rear yard"; "retain and service the canopy" | "previous use was a clothing shop; no arrestor" | inspection of the existing arrestor and exhaust; new permit for the new operator; no new arrestor |
| `exhaust-riser-through-building` | "ground floor of a three-storey shop-top"; "riser in a fire-rated shaft to above the roof"; "residents above" | "single storey; discharges through its own roof" | mechanical, structural and acoustic consultants; rated shaft; owners corporation approval; the building is partly Class 2, so DBP Act regulated designs and declarations for the work on the Class 2 part (owner ruling R3 applied **by inference**; whether a ground-floor Class 6 fit-out alone is caught is `verified: false` and flagged for the owner) |
| `grease-arrestor-location-constrained` | "no rear yard"; "arrestor in the basement car park"; "junction too shallow; pumped" | "in-ground in the rear yard, gravity to the junction" | hydraulic engineer; pumped or remote arrestor; landlord or strata approval |
| `outdoor-footway-dining` | "12 outdoor seats"; "tables on the footway" | "all seating inside" | footway dining approval and licence fee; outdoor furniture; layout plan |
| `step-at-entry` | "180 mm step at the shopfront"; "floor above the footpath" | "level with the footpath; step-free" | access consultant; ramp; performance solution or hardship assessment |
| `gas-cooking` | "wok burners and chargrill are natural gas"; "Jemena meter" | "all electric; no gas" | gas meter and connection; consumer gas (`fr.gas-commercial`); compliance certificate |
| `power-supply-upgrade` | "single-phase cannot carry the kitchen"; "three-phase upgrade"; "Ausgrid application" | "existing three-phase supply has capacity" | DNSP and Level 2 fees; application and upgrade rows (no class 6 fragment) |
| `shopping-centre-tenancy` | "Shop 14 in the centre"; "tenancy design criteria"; "landlord's nominated fire contractor" | "freestanding strip shop"; "not sprinklered" | sprinkler and detection alterations with a hold point; landlord review; nominated-trade costs; out-of-hours premium |
| `heritage-conservation-area` | "within the Main Street HCA"; "contributory shopfront and awning" | "not a heritage item, not in a conservation area" | heritage consultant; heritage referral; HIS; complex town-planning scale |
| `asbestos-found` | "bonded asbestos in the vinyl tiles"; "fibre cement to the rear wall" | "no asbestos in the areas to be disturbed" | removal provisional sum; SafeWork notice; clearance certificate |
| `bca-upgrade-on-change-of-use` | "consent requires a BCA fire safety upgrade"; "non-compliant separation to the dwelling at the rear" | "certifier confirms no upgrade is required" | BCA consultant and fire engineer; upgrade works; FRNSW referral |
| `residential-neighbours-acoustic` | "dwellings adjoin the rear lane"; "council requires an acoustic report for the roof fan and hours" | "no residential receivers; backs onto a car park" | acoustic consultant (never suggested); acoustic and odour report for the DA; fan attenuation |
| `roof-cannot-carry-exhaust-fan` | "timber roof framing not adequate for the fan and canopy hangers"; "engineer to certify the fan support" | "proprietary plinth on purlins the supplier confirms adequate" | structural engineer; `structure.alterations` alter; certification of the fan and canopy support |
| `patron-numbers-need-more-toilets` | "60 seats"; "separate male and female patron toilets" | "the accessible unisex WC is accepted"; "centre amenities" | patron toilets; NCC Part F4 (verified: false); the count rule cannot run (`det.occupant_load` unsettable) |
| `landlord-asbestos-register-covers-tenancy` | "the landlord's register and refurbishment survey cover the tenancy" | "no register is available"; "register excludes areas not accessed" | removes the hazmat surveyor, its fee line and the survey row |
| `liquor-licence` | "on-premises liquor licence"; "alcohol with meals until 10 pm" | "no liquor licence" | licence application and fees; plan of management for licensed hours (no app effect today) |

Variants for one-change tests: `cdc-pathway`, `outdoor-footway-dining`, `exhaust-riser-through-building`,
`grease-arrestor-location-constrained`, `step-at-entry`, `gas-cooking`, `shopping-centre-tenancy`,
`existing-food-premises`.

## 10. Keyword traps and negations specific to this type

- **"Kitchen"** fires `implied_by kitchen`, which adds joinery, sanitary, water, hot water and local
  exhaust [K: knowledge/profile/scope_defaults.yaml implied_by kitchen], but **not** trade waste. A
  "kitchenette" or "staff kitchen" in an office is not a commercial kitchen [K: knowledge/determinants.yaml
  commercial_kitchen notes].
- **"Grease trap", "grease arrestor", "GA", "trade waste", "canopy", "hood"** match no implied_by pattern,
  so the two systems that define this job appear only if a document states them as included.
- **"Exhaust"** in "WC exhaust" and "kitchen exhaust" both map to `mechanical.local-exhaust`; the app's
  `fr.ventilation-fitout` row ("Alter exhaust ventilation") is really on `mechanical.ventilation`.
- **"Strip-out"** fires `site.demolition` with the default action `alter`, so the strip-out fragment does
  not fire [K: knowledge/program/fragments.yaml fr.strip-out-fitout].
- **"Existing grease arrestor to be removed"** reads as trade waste in scope; the action is remove, not new.
- **"Sprinkler"** in "the centre is sprinklered" versus "no sprinklers in the tenancy" (det.sprinklered
  triggers on the bare word [K: testing/synthetic/reference/index/triggers.md §1]).
- **"Shop"** is both the old use and a building type; "change of use from shop to café" is the planning
  fact, but the app has no change-of-use field.
- **"Outdoor seating within the tenancy courtyard"** is not footway dining.
- **"Accessible toilet in the centre amenities"** means the tenancy may not need its own; the shared
  facility has to be confirmed with the certifier.
- **"Landlord's works"** lists items the tenant does not build; do not count them as tenant scope. In
  the profile they map to `sys.<leaf>.provider: landlord`, as tenant-supplied equipment maps to
  `provider: owner`; neither can be written as a scenario input, so the documents must say it.
- **"Separating wall" or "common wall"** between tenancies fires `implied_by party-wall` [K:
  knowledge/profile/scope_defaults.yaml implied_by party-wall], which adds `fire-passive.compartments`,
  `interiors.acoustic-separation` and `site.adjoining-structures`. Fit-out documents use these words for
  the walls between shops.
- **Negations:** "no gas", "no outdoor dining", "no liquor licence", "not heritage", "level entry",
  "existing supply is adequate", "no asbestos identified".

## 11. Gaps in the built-in knowledge

What the app will probably produce wrongly for the base scenario:

1. **Over-procurement.** Ten consultants are suggested and adopted on cost-plan prepare; seven are not
   needed (Architect, BCA, Mechanical, Hydraulic, Electrical, Fire Services, Access) [K:
   knowledge/works/package_defaults.yaml baselines[2]–[4], scope_additions[2]–[5]]. The refurb rules ignore
   size and design-and-construct services, and the scope additions re-suggest the services consultants
   from any services item. Under the traditional route the head-contract proposal drops design lines, so
   once those consultants are removed the services design belongs to no one [K:
   internal/procurement/proposed.go ProposeHeadContract].
2. **Missing people.** No kitchen designer or equipment supplier id; hazmat surveyor never suggested
   [K: data/intake/disciplines.json].
3. **The two defining systems are invisible.** Refurb scope starts empty; nothing implies
   `hydraulic.trade-waste`; `det.commercial_kitchen` and `det.trade_waste_use` cannot be set or read, so
   the kitchen exhaust and trade waste rules never apply [K: knowledge/determinants.yaml
   commercial_kitchen, trade_waste_use; knowledge/clusters/services-wet-air/rules.yaml
   rule.as1668-1.kitchen-exhaust-fire-rated-duct, rule.water-authority.trade-waste-agreement].
4. **Program.** No fragments for the grease arrestor, kitchen exhaust, kitchen equipment, signage,
   accessible WC, exit lighting or shopfront in a fit-out; no approval-documentation fragment; no rows for
   landlord approvals, Sydney Water, the food premises inspection, the fire safety certificate or opening;
   the CC sits before documentation; floors follow painting; the strip-out fragment does not fire.
5. **Cost.** The `commercial_fitout` split suits an office, not a café; no Sydney Water, food premises,
   landlord or levy lines; the certifier is double counted; tenant-supplied equipment may be costed twice.
6. **RFPs.** The design lead gets the generic RFP; on the certifier RFP every stage asks until the client
   picks Package B, and the `fact:flag.bca_*` checks block issue; the exhaust and trade-waste RFP rows sit
   on client flags.
7. **Knowledge.** Nothing on FSANZ 3.2.3, AS 4674, food premises approval, footway dining, signage
   approval, retail leases or change of use.
8. **Implications only.** The trade-waste and kitchen-exhaust unforeseen records exist but never reach
   the generators [K: knowledge/clusters/services-wet-air/unforeseen.yaml
   uc.shop-to-cafe-change-has-no-path-for-grease-arrestor, uc.trade-waste-approval-delayed,
   uc.existing-grease-arrestor-undersized-for-new-kitchen].

## 12. Sources and unverified claims

Opened in the repo: `knowledge/profile/taxonomy.yaml` (food_beverage), `knowledge/determinants.yaml`
(commercial_kitchen, trade_waste_use, gas_supply, existing_building_year), `knowledge/works/package_defaults.yaml`
(baselines, scope_additions, complexity_additions), `knowledge/profile/scope_defaults.yaml` (implied_by),
`knowledge/clusters/services-wet-air/systems.yaml`, `rules.yaml`, `interfaces.yaml`, `unforeseen.yaml`,
`knowledge/clusters/structure/systems.yaml` (site.signage), `knowledge/clusters/fire/rules.yaml`,
`knowledge/clusters/fire/interfaces.yaml` (fire-stopping and sprinkler hold points),
`knowledge/program/skeletons.yaml`, `knowledge/program/fragments.yaml`, `knowledge/costs/elements.yaml`,
`knowledge/costs/starter_lines.yaml`, `knowledge/reports/templates.yaml` (tpl.rfp-capex),
`internal/program/generate.go` (holdPointRows), and the index files under
`testing/synthetic/reference/`.

**Unverified (all `verified: false`):** every clause of NCC 2022 Volume One (Class 6 definition, sanitary
facilities Part F4, accessible sanitary facilities, exit signs and emergency lighting, extinguishers,
Section J for a fit-out); AS 1668.1 and AS 1668.2 (when a hood is required and where it discharges);
AS 2444; AS 2293.1; AS 1428.1; AS/NZS 5601.1; FSANZ Standard 3.2.3 and AS 4674-2004; the Food Act 2003
(NSW) notification; Sydney Water trade wastewater permit rules and arrestor sizing; EP&A Act 1979 and the
EP&A (Development Certification and Fire Safety) Regulation 2021 (fire safety schedule and certificate,
inspections for Class 6, upgrade on change of use); the Codes SEPP Commercial and Industrial Alterations
Code; SEPP (Industry and Employment) 2021 Chapter 3 signage; s7.12 levies; the s10.7 certificate; the
long service levy threshold; Roads Act 1993 s125 footway dining; WHS Regulation 2017 (NSW) asbestos
register, survey and notice; the Disability (Access to Premises - Buildings) Standards 2010 affected part;
the Plumbing and Drainage Act 2011; the Sydney Water Tap in requirement in DA consents; the DBP Act's
reach to a Class 6 fit-out in a partly Class 2 building (owner ruling R3 by inference); the Retail Leases Act 1994 (NSW); the Strata Schemes Management Act
2015; Liquor & Gaming NSW licence types; the defects liability period length.

**Judgement calls the owner should check:** that the kitchen exhaust through the tenancy's own roof is
normally designed and certified by the exhaust subcontractor, with no mechanical engineer; that the
plumber, not a hydraulic engineer, normally sizes the arrestor and applies to Sydney Water; that a level
entry leaves access to the certifier; and that a DA, not a CDC, is the more common pathway for a café in a
former shop.
