# T17 Façade or combustible cladding remediation: domain map

Draft for owner review, 30 October 2026. The machine key is `shape.yaml` in this folder; it passes
`validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Any statute, regulation or standard clause not
confirmed from the instrument is marked **verified: false**. Owner rulings applied: R3 (DBP Act applies to
Class 2, including remedial work), R4 (HBCF threshold; the multi-storey exemption stays verified: false).

---

## 1. The job

**What it is.** The owners corporation (OC) of an occupied NSW strata Class 2 residential flat building
removes combustible external cladding that an audit has identified, and replaces it with a
non-combustible façade. The two products found most often are:

- **aluminium composite panel (ACP) with a polyethylene (PE) core**, on balcony spandrels, feature blades
  and soffits;
- **expanded polystyrene (EPS) rendered insulation systems** (EIFS), often on upper setback levels.

Combustible sarking, foil-faced insulation and PE-core ACP behind "feature" elements turn up as well.
[K: knowledge/clusters/fire/systems.yaml fire-passive.external-walls; data/reference/clerk/data/seed/ncc-reference-guide.md "### Cladding Combustibility Requirements"; knowledge/clusters/envelope/unforeseen.yaml uc.combustible-cladding-on-existing-facade]

**How the job starts** [D]. The building was found through the NSW cladding audit and registration
process. Owners of affected Class 2–9 buildings of two or more storeys had to register them on the NSW
Planning Portal (EP&A Regulation, verified: false). Council (or Fire and Rescue NSW, which can also issue
orders), usually prompted by the NSW Cladding Taskforce, gives a **notice of intention** and then a
**fire safety order** (EP&A Act 1979 Division 9.3 and Schedule 5, verified: false). The order requires the
OC to remove or remediate the cladding by a date, and often to keep interim fire safety measures in place
until it is done. The Building Products (Safety) Act 2017 (NSW) and its 2018 use ban on PE-core ACP (more
than 30% PE) on Class 2, 3 and 9 buildings of two or more storeys and Class 5–8 buildings of three or
more storeys are the legal background (verified: false).

**Base scenario** (used in `shape.yaml`):

- an 8-storey, 56-unit residential flat building completed in 2011 over one basement car park level;
- reinforced concrete frame, Type A construction, not sprinklered (effective height under 25 m;
  verified: false);
- PE-core ACP on the balcony spandrels and vertical blades of all four elevations, and an EPS render
  system on the level 7 and 8 setback walls;
- a fire safety order from council requiring full removal and replacement;
- **owner-funded** through special levies (Project Remediate is a trigger);
- replacement with a ventilated rainscreen of solid aluminium panels on a new subframe, with
  non-combustible insulation and sarking, which changes the external appearance;
- **development application** (the change of materials and appearance), then a construction certificate
  (CDC is a trigger);
- traditional procurement: consultant-designed and documented, one AS 4000 lump-sum contract with a
  remedial builder, the project manager as superintendent;
- the building stays fully occupied, worked elevation by elevation from a perimeter scaffold with a
  hoarding over the council footpath.

**Scale.** $1–20 M [K: testing/synthetic/PROJECT_TYPES.md T17 row]. Cost is driven by the area of
cladding, the access method (scaffold or swing stage), substrate condition found on removal, and the
replacement system, not by GFA [D].

**Who procures it and how** [D]:

- **Client.** The OC, through the strata committee, with the strata manager as its agent (not a project
  consultant). Every spending step needs a committee or general meeting resolution. Levies and loans are
  the critical path.
- **Project manager.** Appointed by the OC to run the job: consultant procurement, program, reporting
  to the committee, superintendent under AS 4000.
- **Design team.** Architect (façade redesign, DA, documentation, regulated design), fire safety engineer
  (risk assessment, interim measures, DTS confirmation or performance solution), façade engineer (system,
  fixings, weatherproofing, product evidence), structural engineer (primary-structure capacity, bracket
  fixings, pull-out testing), BCA consultant, town planner (DA), QS (estimates for the levy and the DA
  cost report) and a principal certifier (CC, inspections, OC).
- **Remedial builder.** Licensed and registered under the DBP Act, with a specialist façade
  subcontractor and a licensed scaffolder.

**What makes it go wrong** [D]:

- The scope is fixed before the materials are sampled and tested, so retained "unknown" products are
  found during removal.
- Concealed water damage, corroded subframes or failed fixings are found behind the cladding
  [K: knowledge/clusters/envelope/unforeseen.yaml uc.cladding-removal-reveals-concealed-damage].
- Existing slab edges cannot take the new brackets
  [K: knowledge/clusters/structure/unforeseen.yaml uc.existing-wall-fixings-cannot-take-overcladding].
- A "non-combustible" or "fire-rated" substitute product has no valid test evidence, or is discontinued
  [K: knowledge/clusters/envelope/unforeseen.yaml uc.cladding-product-fails-compliance-test,
  uc.cladding-product-discontinued-and-substituted].
- Cavity barriers and slab-edge fire protection are closed up uninspected
  [K: knowledge/clusters/fire/interfaces.yaml if.fire-facade-slab-edge].
- The façade is left open to the weather between removal and reinstatement
  [K: knowledge/clusters/envelope/interfaces.yaml if.envelope-repair-temporary-enclosure].
- Hot works on a façade that still carries combustible cladding.
- The fire safety order deadline passes while the OC is still raising the levy, and council escalates.
- Lot-owner items on the façade (AC condensers, lights, signs, satellite dishes) are found late and
  nobody knows who pays.

---

## 2. Systems and work items

Action ids are the app's eight [K: knowledge/works/actions.yaml actions]. The work-type default for
remediation is `repair` [K: knowledge/works/actions.yaml work_type_defaults], but removing a cladding
and installing a different one is `replace`.

| system | in scope (base) | action | note |
|---|---|---|---|
| **site** | none | — | `site.demolition` is a **trap**: removing cladding is part of the replacement, not demolition. It becomes real only for licensed asbestos removal (trigger `hazardous-materials`). The footpath hoarding and road occupancy are temporary works with no system id (§11). `site.adjoining-structures` only for a boundary wall (trigger). `site.contamination` is never in scope (§10). |
| **substructure** | none | — | The basement is existing and untouched. The scaffold stands on the footpath and the podium. |
| **structure** | `structure.concrete` | investigate | Slab edges and columns that carry the new brackets: cover, condition, edge distance, pull-out tests. Repairs are provisional and become `repair` under `concealed-substrate-damage`. `structure.temporary-works` excludes scaffolding [K: knowledge/clusters/structure/systems.yaml structure.temporary-works excludes]. |
| **envelope** | `envelope.external-walls` | replace | The core: strip ACP and EPS render, install the new non-combustible cladding and subframe [K: knowledge/clusters/envelope/systems.yaml envelope.external-walls]. |
| | `envelope.thermal-insulation` | replace | The EPS is the insulation; the replacement is non-combustible (mineral wool). |
| | `envelope.pliable-membranes` | replace | Combustible sarking and wall wrap behind the cladding are replaced with a compliant product (the NCC sarking concession, verified: false). |
| | `envelope.flashings-seals` | replace | Window head and sill flashings, sealant joints and the cladding-to-window interface. |
| | `envelope.windows-doors` | retain | Windows stay and are protected; the new cladding must seal to them. Replacement is a trigger. |
| | others | — | Curtain wall (none on this building), roof coverings and external shading are untouched. `envelope.external-waterproofing` only where the cladding returns into balcony upturns (§3). |
| **interiors** | none | — | Internal linings disturbed at window reveals are made good by the builder; internal water damage found on removal is a trigger. |
| **fire-passive** | `fire-passive.external-walls` | upgrade | The fire performance of the external wall is the point of the job: non-combustible materials, cavity barriers, and spandrel or slab-edge protection against vertical spread [K: knowledge/clusters/fire/systems.yaml fire-passive.external-walls]. NCC 2022 Vol One Part C2 non-combustibility (C2D10) and the AS 5113 verification method: verified: false. The app's own rule cites NCC 2019 numbering, `clause_verified: false` [K: knowledge/clusters/fire/rules.yaml rule.ncc.external-wall-fire]. |
| **fire-active** | none in base | — | The building is not sprinklered. Interim measures are operational, not a system. Upgrades the order requires are trigger `fire-safety-upgrades-required`. |
| **mechanical** | none in base | — | AC condensers on façade brackets and exhaust outlets through the cladding are trigger `facade-services-reinstatement`. |
| **hydraulic** | none in base | — | External downpipes on the façade are the same trigger. |
| **electrical** | none in base | — | Façade lights and external GPOs are the same trigger. |
| **comms-security** | none | — | Antennas and CCTV on the façade, if present, go with the services trigger. |
| **vertical-transport** | none | — | The lift is used by residents only; materials go up the scaffold hoist [D]. |
| **access-egress** | none in base | — | ACP balustrade infill panels are trigger `balcony-balustrade-acp`. Roof anchors for swing stages are trigger `swing-stage-access`. |

---

## 3. Existing-building interfaces

The whole job is an existing-building interface [D].

- **Occupation.** The building stays occupied. Units lose light and privacy behind scaffold shade cloth for
  months; balconies are closed while their spandrels are worked on; windows may not be opened during
  removal. Plan elevation by elevation with resident notices, a liaison person and noise and dust controls.
  The OC's right to enter lots needs written notice (SSMA 2015, verified: false). The `existing_building`
  and `live_site` RFP modules switch on from the work type and `hdr.cond.operational_constraints`
  [K: knowledge/rfp/modules.yaml; internal/rfp/drivers.go deriveFlags, readFlags].
- **Fire safety while the cladding is still there.** The order or the fire engineer usually sets interim
  measures: no storage or BBQs on balconies, a hot-works ban next to combustible cladding, more frequent
  fire safety checks, and resident information. Removal sequence matters: never leave exposed EPS or open
  cavities overnight without control [D].
- **Common property versus lot property.** The external walls are normally common property. Lot-owner
  additions on the façade (AC units, lights, satellite dishes, privacy screens) need a by-law check and a
  decision on who pays before tender (verified: false; depends on the plan and its by-laws).
- **Services cut-overs.** Disconnect, relocate and reinstate anything fixed to or through the cladding:
  exhaust and dryer vents, AC pipework, façade lights, downpipes, building signage. No authority
  cut-overs in the base.
- **Hazardous materials.** Not in the base (built 2011). Older buildings may have fibre-cement soffits or
  mastics with asbestos. The app's only hazmat prompt for cladding is the record
  [K: knowledge/clusters/envelope/unforeseen.yaml uc.asbestos-cladding-disturbed]. The record needs an
  alter, upgrade, repair or remove item on the external walls and a year before 2004; with the correct
  `replace` action it does not fire, so even an old building gets no hazmat prompt from the cladding item.
  Removed PE-core ACP and EPS are general solid waste in most cases; confirm the waste classification
  (verified: false).
- **Structure.** Bracket fixings into slab edges near reinforcement, pull-out capacity, added or changed
  wind load on a new rainscreen, corrosion of the existing subframe
  [K: knowledge/clusters/envelope/failure_modes.yaml fm.envelope-facade-bracket-clashes-with-slab-edge-reinforcement, fm.envelope-cladding-fixings-not-corrosion-rated].
- **Matching existing work.** The new cladding must seal to retained windows and to balcony waterproofing
  upturns [K: knowledge/clusters/envelope/interfaces.yaml if.envelope-balcony-facade-continuity,
  if.envelope-old-new-weather-barrier]. Reclad downpipes must still reach the existing risers
  [K: knowledge/clusters/envelope/failure_modes.yaml fm.envelope-reclad-downpipes-miss-existing-risers].
- **Dilapidation.** Record the internal condition of every unit next to the works and the condition of
  the footpath, podium and any neighbouring building before the scaffold goes up [D].

---

## 4. Consultants and why

| discipline (app id) | stage engaged | need | why |
|---|---|---|---|
| Project manager (`consultant.project_manager`) | Investigation, through to defects | **Always** at this scale | Runs the job for the OC: consultant procurement, reporting to the committee and general meetings, council liaison on the order, superintendent. Under Project Remediate (trigger) the program engaged and paid the PM, so the OC has no PM package (verified: false). |
| Architect (`consultant.architect`) | Investigation (options), through construction | **Always** | Façade redesign and appearance, DA drawings and photomontages, documentation, regulated design (architectural), design support during construction. |
| Fire safety engineer (`consultant.fire_engineer`) | Investigation, through completion | **Always** | Reviews the audit and test results, sets interim measures, advises the extent of removal, confirms DTS compliance or prepares a performance solution, inspects cavity barriers and slab-edge protection. |
| Façade engineer (`consultant.facade`) | Concept, through construction | **Always** | System selection, wind loads, subframe and fixing design or review of the supplier's design, weatherproofing details, product test evidence, façade testing. |
| Structural engineer (`consultant.structural`) | Investigation, through construction | **Always** in base | Primary-structure capacity for the new brackets, pull-out test regime, slab-edge repair specification, certification of fixings. The façade engineer sometimes covers fixings; the structural engineer still certifies the primary structure [D]. |
| BCA consultant (`consultant.bca`) | Investigation, through documentation | **Always** | NCC compliance report for the remediation design, existing-building assessment, the fire safety schedule changes. |
| Principal certifier (`consultant.certifier`) | Approvals, through completion | **Always** on the DA path | Construction certificate, critical-stage and other inspections, fire safety certificate, occupation certificate for the altered part (EP&A Act s 6.9, verified: false). |
| Town planner (`consultant.town_planner`) | Approval documentation | **Always** on the DA path | Statement of environmental effects for the change of appearance and materials. |
| Quantity surveyor (`consultant.quantity_surveyor`) | Investigation, through construction | **Always** at this scale | Option estimates for the levy, the cost estimate report the DA needs, tender estimate, progress-claim assessment. |
| Remedial builder (`trade.main_works`) | Procurement | **Always** | Head contract. |
| Hazmat (`consultant.hazmat`) | Investigation | Conditional: built before about 2004, or an asbestos register entry | Survey of the façade build-up; clearance after removal. |
| Fire services (`consultant.fire_services`) | Documentation | Conditional: the order also requires sprinkler or detection upgrades | Design of the added active systems. |
| Surveyor (`consultant.surveyor`) | Investigation | Conditional: a boundary wall | Confirms the new cladding stays within the boundary. |
| Roof access (`consultant.roof_access`) | Construction | Conditional: swing stage or BMU | Anchor inspection and certification. |
| Heritage (`consultant.heritage`) | Approval documentation | Conditional: heritage listing | Heritage impact statement. |
| Expert witness (`consultant.expert_witness`) | Before removal | Conditional: a defects claim | Evidence preservation, joint inspection, expert report. |

**Never needed for the base:** civil, geotechnical, traffic, mechanical, electrical, hydraulic,
landscape, acoustic, access, BASIX/ESD, waste. The app suggests only the Town Planner (DA) and, if the
profile reads the façade as complex, the Façade engineer
[K: testing/synthetic/reference/index/type-predictions.md T17; knowledge/works/package_defaults.yaml complexity_additions[0], complexity_additions[7]].
It never suggests the project manager, architect, fire engineer, structural engineer, BCA consultant,
certifier or QS, because remediation has no baseline [K: knowledge/works/package_defaults.yaml baselines].
Once documents mention façade services or "kitchen" and "bathroom" exhausts, it suggests Hydraulic,
Mechanical and Electrical consultants the job does not need (`shape.yaml not_expected_procurement`).

---

## 5. Approvals and authorities

- **Fire safety order.** The notice of intention (from council, or from FRNSW, which can also issue
  orders), the OC's representations, the order, its compliance date and any extension, and, when the work
  is done, council confirming compliance or revoking the order (it does not "sign off")
  (EP&A Act 1979 Division 9.3 and Schedule 5, verified: false). Progress reports to council are usual.
  The app has no profile key, fragment or cost line for an order (§11).
- **Owners' consent.** The OC owns the common property, so a general meeting resolution gives its consent
  to lodge the DA (verified: false).
- **Planning.** A change of external materials and appearance on a residential flat building normally
  needs a DA [D]. Whether a cladding replacement can be complying development under the Codes SEPP 2008
  (the Fire Safety Code), or needs no consent because an order requires it, depends on the instrument
  and the building: verified: false. These are triggers `cdc-pathway` and `order-compliance-no-consent`.
  Pre-CC conditions of consent (colours and materials, dilapidation, long service levy) are met before
  the CC. A DA for a residential flat building may need
  an architect's design verification statement (Housing SEPP Chapter 4 / former SEPP 65, verified: false).
  For a DA, the app adds `fr.da-lodgement`, `fr.da-exhibition`, `fr.da-assessment` and
  `fr.construction-certificate` [K: knowledge/program/fragments.yaml].
- **Construction certificate, inspections and occupation certificate.** The principal certifier issues the
  CC, carries out inspections, and issues an OC for the altered part with a final fire safety certificate
  (EP&A (Development Certification and Fire Safety) Regulation 2021, verified: false). The fire safety
  schedule and the OC's annual fire safety statement change after the works [D].
- **Fire and Rescue NSW.** A performance solution against certain fire safety provisions is referred to
  FRNSW through initial and final fire safety reports (verified: false; which provisions trigger referral
  must be confirmed). Trigger `performance-solution-retained-cladding`.
- **Design and Building Practitioners Act 2020 (NSW)** (owner ruling R3). The work is building work on a
  Class 2 building, and the external wall and its fire performance are building elements. So:
  - registered design practitioners (architectural, façade, structural and fire safety classes as they
    apply) prepare **regulated designs** and give **design compliance declarations**;
  - the designs and declarations are lodged on the NSW Planning Portal before building work starts; the
    certifier assesses the same designs for the CC, so the program puts them before the CC (timing
    verified: false);
  - varied designs are declared again;
  - the registered building practitioner gives a **building compliance declaration** at the end.

  Section numbers, the practitioner classes and the lodgement timing: verified: false. The app knows this
  only as `proj.regulated_design` and an unverified RFP threshold
  [K: knowledge/profile/project_details.yaml regulated_design; knowledge/rfp/structural.yaml thresholds dbp_regime].
- **Home Building Act 1989.** Residential building work on a Class 2 building needs a licensed contractor
  and an HBA-compliant contract. HBCF cover is required above a contract price of $20,000 including GST
  (R4), but buildings with a rise of more than three storeys are exempt (verified: false), so the 8-storey
  base needs none. Trigger `low-rise-hbcf`.
- **Strata Schemes Management Act 2015.** The OC's duty to repair and maintain common property, general
  meeting resolutions for levies and borrowing, and a special resolution where common property is altered
  rather than repaired (s 106, s 108; verified: false). Whether replacing a cladding with a different
  product is an alteration needing a special resolution: verified: false. Most OCs pass one anyway [D].
- **RAB Act 2020.** Whether the expected completion notice before the OC application applies to
  remedial work on an existing building: verified: false.
- **Council (roads).** A scaffold and hoarding over the footpath needs council approval under the Roads Act
  1993 s 138, and a crane or delivery may need a road occupancy licence (verified: false).
- **SafeWork NSW.** Scaffold above the licence height needs a licensed scaffolder; work at height; licensed
  asbestos removal is notified in advance (all verified: false).
- **Long service levy.** Paid before the CC is issued, on the DA path (rate and threshold verified: false).
- **No Sydney Water or DNSP** involvement on the base. Whether NCC Part J (energy efficiency) applies to
  the replacement wall is a BCA consultant question (verified: false).
- **BASIX.** R5 (BASIX for alterations ≥ $50,000) is taken not to apply: the work is to common property,
  not to a dwelling (verified: false; confirm on the Planning Portal). No BASIX line or package.
- **Hold points** (contract and certifier) [D]:
  0. panel samples, colour and visual mock-up approved before fabrication;
  1. sample removal and test results before the removal extent is fixed;
  2. substrate, subframe and bracket fixings (with pull-out results) before insulation;
  3. cavity barriers, slab-edge and spandrel fire protection before close-up;
  4. flashings and window interfaces before cladding;
  5. façade water test on a sample area (AS 4284 or hose test, verified: false);
  6. certifier inspections as the CC lists them.

  The app's only hold points come from five interfaces with system ids; none fires for an external wall
  [K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §4.4; knowledge/clusters/fire/interfaces.yaml if.fire-facade-slab-edge has no hold_point].

---

## 6. Cost plan shape

Lines only; no amounts (see `shape.yaml cost_plan`).

- **Fees and charges.** DA fees and contributions; construction certificate and certifier fees; long
  service levy; council hoarding, footpath and road occupancy fees.
- **Consultants.** Project manager, architect, fire engineer, façade engineer, structural engineer, BCA
  consultant, principal certifier (if not carried in the CC fee line), town planner, QS, and the
  investigation disbursements: sampling, laboratory combustibility testing, opening-up and pull-out
  testing.
- **Construction**, by trade or element [D]:
  - preliminaries and occupied-building controls;
  - perimeter scaffold, shade cloth and edge protection, with its engineering;
  - footpath hoarding or overhead protection (needed even with swing stages);
  - pre-commencement condition surveys of units, footpath and neighbours;
  - equipotential bonding and earthing of the aluminium cladding and subframe, by the builder's
    electrician (verified: false; an `electrical.*` item would wrongly add the Electrical consultant);
  - remove and dispose of the combustible cladding and EPS;
  - subframe and bracket fixings;
  - the non-combustible cladding system;
  - insulation and sarking;
  - cavity barriers and slab-edge fire protection;
  - flashings, sealants and window interfaces;
  - interim fire safety measures during the works;
  - façade testing and hold-point inspections;
  - provisional sums for concealed conditions behind the cladding and slab-edge repairs.
- **Contingency and other.** Design contingency, construction contingency (high; concealed conditions are
  certain), escalation (levy timing), exclusions (lot-owner items), and OC costs (strata manager, legal
  review of the contract and the levy or loan, finance costs).

App behaviour:

- Starter lines P01 to P07, including P04 DA fees (planning da) and P05 CC fees
  [K: knowledge/costs/starter_lines.yaml lines].
- Consultant lines only for the Town Planner and the Façade engineer, the two suggested packages
  [K: internal/costs/suggest.go Suggest].
- Construction lines appear as generic element codes once work items exist: 03 Structure (from the
  investigate item), 04 Building envelope, 06 Passive fire protection [K: knowledge/costs/elements.yaml].
- No budget split for remediation, so a stated budget is not spread
  [K: knowledge/costs/budget_splits.yaml].
- **Not expected** (`shape.yaml not_expected_cost_plan`): P06 service authority connection fees, and
  conditional Hydraulic, Mechanical and Electrical consultant fees.

---

## 7. Program shape

The base has 44 activities; see `shape.yaml program`. Stages and the activities that drive each:

1. **Investigation.** Order received → interim fire safety measures put in place at once, and in
   parallel the committee appoints the PM and approves the investigation budget → records → façade
   survey and sampling with laboratory testing → fire engineering risk assessment and removal-extent
   advice → intrusive opening-up and pull-out testing → options report with order of cost.
2. **Decision** (`concept_design`). General meeting resolves the option, the design fee, the first levy
   and owners' consent to lodge the DA → concept and schematic façade design.
3. **Approval documentation.** DA drawings, photomontages, statement of environmental effects, BCA and
   fire engineering letters, QS cost report.
4. **Approvals.** Lodge DA → notification → assessment and determination → pre-CC conditions of consent.
   Council is kept informed on the order; an extension of the compliance date is often needed.
5. **Construction documentation.** Design development of system, subframe, cavity barriers and fixings →
   product compliance evidence → BCA compliance report → regulated designs and declarations → tender
   documents → long service levy → CC.
6. **Procurement.** Tender with site inspection → evaluation (licence, DBP registration, scaffold and
   removal methodology, product evidence) → general meeting resolves award and the construction levy →
   contract award.
7. **Construction.** Samples, colours and visual mock-up approved (hold point, before fabrication) →
   hoarding approval → resident notices and condition surveys → site establishment and scaffold → then
   repeat per elevation: remove cladding → hold point: substrate and fixings → insulation,
   sarking, cavity barriers → hold point: fire protection before close-up → flashings → cladding → façade
   test → strike the scaffold.
8. **Completion.** Building compliance declaration (DBP) → final inspection, fire safety certificate and
   OC → council confirms compliance with or revokes the order; the cladding registration and fire safety schedule
   are updated → PC and handover of the product register and warranties.
9. **Defects.** Defects liability period and final inspection.

**Logic.** Sampling and testing come before the removal extent is fixed. The general meeting comes
before design spend and again before award. The DA comes before the CC; regulated designs come before the
CC (verified: false). Each elevation's fire-protection hold point comes before its cladding.

**Long-lead items** [D]: the cladding panels (shop drawings, test evidence, fabrication, often imported),
subframe and brackets, scaffold and hoarding approval, and any replacement windows (trigger).

**App behaviour.** No remediation skeleton, so the app uses `sk.generic`. The DA and CC fragments fire.
`fr.facade-2` and `fr.insulation-2` exist for Class 2 but need work type new or extend. Contract award,
PC, OC, handover and defects fragments are gated off for remediation. `fire-passive.external-walls` has no
fragment at all [K: knowledge/program/fragments.yaml; knowledge/program/skeletons.yaml;
testing/synthetic/reference/index/program-fragments.md "Leaf systems with no work-item fragment"].
Durations stay with the app's `knowledge/program/durations.yaml`; none are quoted here.

---

## 8. RFP shape

**Project manager** (no template; generic `tpl.rfp-capex`). Must carry: the order and council liaison,
consultant procurement and management, committee and general meeting reporting, levy or loan cash-flow
input, program, superintendent under AS 4000 (including SOP Act payment schedules, verified: false),
resident communication, and close-out of the order. Must not carry: design, certification, or cost
estimating beyond reporting.

**Architect** (`architecture`). Must carry: existing façade survey drawings, options, DA drawings and
photomontages, documentation, regulated design and declarations, tender documents and bidding,
for-construction documents and design support during construction. Must not carry: interiors, BIM model,
typical apartment interiors, door schedules, FF&E, tenant fit-out.
What the app gets wrong:

- The boundary ends at `interiors`, so tender, bidding, regulated design, for-construction and
  construction support are priced only as options [K: knowledge/rfp/architecture.yaml boundary;
  internal/rfp/resolve.go resolveStages].
- Any Class 2 part makes service level complex, which brings in a BIM model and typical interiors.

**Fire safety engineer** (`fire-engineering`). Must carry: review of the audit and laboratory results
(S1), early approval support for the DA (S2), interim and staged strategy for the occupied works (S8b),
tender integration (S5), construction support and inspection of cavity barriers and slab-edge protection
(S6), verification for the OC and fire safety certificate (S7), and closure evidence for the order. A
performance solution brief and FER are options in the base (trigger). Must not carry: sprinkler or
detection design, BCA certification, expert witness.
What the app gets wrong:

- The boundary ends at S4, so S5 to S8 are only options.
- S3 and S4 depend on a client flag (`fe_solutions`) that no document can set, so they are left as
  decisions.
- The `remediation` row ("Orders/defects … closure evidence") is always only an option, though an order
  is the reason for the job [K: knowledge/rfp/fire-engineering.yaml stages, positions remediation].

**Façade engineer** (no template; generic). Must carry: review of existing façade records and samples,
system options, performance specification (wind, water, movement, fire), subframe and fixing design or
supplier design review, product test evidence (AS 1530.1 non-combustibility, CodeMark or equivalent,
verified: false), regulated design and declaration (façade), shop drawing review, inspections and
façade testing. Must not carry: architecture, fire engineering conclusions, curtain wall design.

**Structural engineer** (`structural`). Must carry: existing structure assessment (slab edges and fixing
zones), pull-out test regime, design and certification of fixings into the primary structure, slab-edge
repair specification (provisional), regulated design for the structural work, construction-phase
inspection. Must not carry: geotechnical brief, retention, planning approval stage, shared Revit model,
footings, super-flat floors. What the app gets wrong: the same faults as T09 (concept stage, geotechnical
brief, shared model and services footings included; construction phase only an option).

**BCA consultant and principal certifier** (both `bca-certification`). The BCA consultant carries
advisory Package A (inception, planning, tender advice) with an existing-building assessment and staged
occupation. The certifier carries Package B (preliminary assessment, CC, principal-certifier inspections,
OC). The app cannot tell them apart: every stage hangs on client flags (`bca_advisory`,
`bca_certification`) that no document can set, so A stages come out as decisions and B stages as options
[K: knowledge/rfp/bca-certification.yaml stages; boundary]. On a CDC (trigger) both RFPs block, because
the template lists `cdc` under `unsupported_pathways`; no template drafts a CDC certifier appointment.

**Town planner** (`town-planning`). DA pre-lodgement, SEE, lodgement, assessment, determination review;
post-approval support as an option. The app gets this right.

**QS** (`cost-planning`). Cost basis, option estimates for the levy, the DA cost estimate report
(registered QS above a value threshold, verified: false), design cost plans, tender estimate and
normalisation, progress-claim reports. Must not carry contaminated-land costing; note that the template's
remediation row is labelled "Defect/remediation/contamination costing". The app leaves S2 to S4 as
decisions (client flags) and prices tender and post-contract stages only as options.

**Conditional:** hazmat (no template), fire services (`fire-services`), surveyor (`land-surveying`), roof
access (no template), heritage (no template), expert witness (no template).

---

## 9. Triggers

There are 19 triggers; full keys are in `shape.yaml`. Each changes procurement, cost, program or RFP.

| id | how documents show it (including indirectly) | how a document denies it | adds |
|---|---|---|---|
| `project-remediate` | "eligible for Project Remediate", "interest-free loan", "the program's appointed project manager" | "the owners corporation is funding the works through special levies" | Eligibility, loan agreement and program panel gates; program-appointed PM mobilised. Removes the OC's PM package, fee line and RFP. Registration closed to new buildings years ago, so the scenario needs an already-enrolled OC (verified: false) |
| `performance-solution-retained-cladding` | "retain the EPS at ground level subject to a performance solution", "fire engineering report to justify partial retention" | "all combustible materials removed to DTS" | FER, FRNSW referral, peer review, BCA performance assessment |
| `hazardous-materials` | "completed in 1998", "asbestos register lists fibre cement soffit linings" | "no asbestos found in the 2020 survey" | Hazmat survey, licensed removal, SafeWork notice |
| `concealed-substrate-damage` | "corroded steel studs behind the ACP", "water staining at the slab edge", "the render is drummy" | "substrate sound in all sample openings" | Structural repair design, provisional repairs, internal make-good |
| `balcony-balustrade-acp` | "ACP balustrade infill panels", "the blade walls form part of the balustrade" | "glass balustrades are retained" | Barrier replacement, temporary edge protection, structural barrier design |
| `facade-services-reinstatement` | "AC condensers on façade brackets", "exhaust vents through the cladding", "external downpipes" | "no services fixed to the cladding" | Disconnect and reinstate; the app wrongly adds services consultants |
| `window-replacement` | "aluminium windows replaced to suit the new rainscreen", "ACP spandrel infill panels within the window framing replaced with the windows" | "windows retained and protected" | Window replacement, long lead |
| `fire-safety-upgrades-required` | "the order also requires sprinklers to the balconies", "upgrade the fire detection system" | "the order relates to the external cladding only" | Fire services consultant and active systems |
| `boundary-wall-neighbour-access` | "nil-setback wall on the eastern boundary", "access over the neighbouring property" | "all elevations face the street or common property" | Surveyor, neighbour access agreement or order, dilapidation |
| `swing-stage-access` | "swing stages from the roof anchors", "no room for a scaffold on the laneway" | "full perimeter scaffold" | Roof access consultant, anchor certification |
| `heritage-listed` | "within a heritage conservation area", "listed as a local heritage item in Schedule 5" | "not a heritage item" | Heritage consultant and referral |
| `cdc-pathway` | "complying development certificate", "the certifier will determine the CDC" | "council advises a DA is required" | CDC; removes the DA, the planner and DA fees. Both BCA and certifier RFPs block on cdc |
| `order-compliance-no-consent` | "no development consent is required for works the order requires" | "a development application is required" | Exempt confirmation; removes the DA, CC, planner, DA fees, the certifier and the OC. A final fire safety certificate is still given and council confirms compliance (author's decision, verified: false). P05 stays in the app |
| `mixed-use-podium` | "three retail shops on the ground floor", "Class 6 podium" | "wholly residential" | `det.ncc_class` [2, 6]; podium BCA scope, tenant notices; mixed-use rows in the fire and architecture RFPs |
| `staged-funding` | "levy raised in two instalments", "strata loan for Stage 2" | "one continuous stage" | GM and levy per stage, separable portions, escalation |
| `balcony-waterproofing-interface` | "failed membrane upturns behind the spandrel ACP" | "upturns intact in all openings" | Membrane repair, hold point, waterproofing licence |
| `design-and-construct` | "D&C tender", "novation of the architect and façade engineer" | "the consultant team documents for a lump-sum tender" | D&C route, novation, contractor-designed façade |
| `defects-claim` | "proceedings against the builder and the panel supplier", "within the ten-year period for a building work rectification order" | "no claim is being pursued", "completed in 2011, outside the warranty and order periods" | Expert witness, evidence preservation, legal costs. Needs a building year of 2017 or later (verified: false); on the 2011 base only a contract or product-liability claim against the supplier survives |
| `low-rise-hbcf` | "a three-storey walk-up", "HBCF certificate before the deposit" | "rise of more than three storeys, so HBCF does not apply" | HBCF, returnable, pre-start check |

**Variants** (the eight most common): Project Remediate, performance solution, hazardous materials,
concealed substrate damage, ACP balustrades, façade services, CDC and D&C.

---

## 10. Keyword traps and negations specific to this type

| trap | right reading |
|---|---|
| **"Remediation" / "Project Remediate"** | Building-defect rectification, never contaminated land. `site.contamination` is labelled "Contaminated land and remediation" [K: knowledge/clusters/structure/systems.yaml]; the seed separates them [K: data/reference/clerk/data/seed/building-remediation-rectification-guide.md "Purpose and boundary"]. No Environmental consultant. |
| **"Fire safety order" vs "fire safety statement" vs "fire safety schedule"** | The order is council's legal direction; the annual statement (AFSS) and schedule are routine. An AFSS mention alone is not an order. |
| **"Fire-rated" / "FR core" ACP** | FR-core ACP is still combustible unless tested non-combustible; "fire-rated" is not "non-combustible" [K: data/reference/clerk/data/seed/ncc-reference-guide.md "Spandrel and Slab Edge Fire Protection" pitfalls]. |
| **"Cladding"** | Can mean roof sheeting ("re-clad the roof", `envelope.roof-coverings`) or a balcony membrane return. Here it is the external wall. |
| **"Cladding system", "rainscreen", "façade engineer"** | Match the `facade_type` triggers [K: knowledge/determinants.yaml facade_type]. Expected reading: `complex` for the base rainscreen, which brings the Façade package. A plain FC-sheet replacement may read `standard`, losing it. |
| **"Basement"** | "Scaffold base plates on the podium slab over the basement car park" matches the basement trigger [K: knowledge/profile/flag_readings.yaml basement]. The basement exists and nothing is excavated; `proj.flag.basement` must not read yes, or structural retention, dilapidation and complex scale switch on. |
| **"Kitchen" / "bathroom" exhausts** | "Kitchen and bathroom exhaust outlets penetrate the cladding" fires the implied-scope code with no Jev and no negation handling [K: knowledge/profile/scope_defaults.yaml implied_by wet-room, kitchen; internal/profile/implied.go]. It adds wet areas, joinery, sanitary, water and exhaust scope, then Hydraulic and Mechanical consultants and cost lines. |
| **Other implied-scope words** | "brickwork behind the render" or "masonry substrate" → `structure.masonry`; "materials up the fire stairs" → stairs; "the common wall between units" → adjoining structures and compartments; "strip-out" or "demolish the cladding" → `site.demolition`; "bedroom windows" → internal doors; "BASIX does not apply" still fires `energy-rating` → `envelope.thermal-performance` [K: knowledge/profile/scope_defaults.yaml implied_by]. |
| **"Class A hoarding", "Type A construction"** | Neither is an NCC building class. Only "Class 2" is. |
| **"Audit"** | The cladding audit, not a financial or fire safety statement audit. |
| **"EPS"** | Expanded polystyrene, not an electrical power system. |
| **"PE"** | Polyethylene core, not a professional engineer. |
| **"Like for like"** | Not possible here: the replacement must be a different, compliant material. A "like-for-like" phrase usually refers to appearance. |
| **"Exempt"** | "HBCF exempt" or "GST exempt" is not the planning pathway. |
| **"Adjoining heritage item"** | Being in the vicinity of a heritage item is not a heritage status: `det.heritage_status` describes the site itself [K: knowledge/determinants.yaml heritage_status]. A heritage impact statement may still be needed (LEP cl 5.10, verified: false). |
| **"Flood"** | "Flood test" of a balcony return is a water test, not a flood-prone site [K: knowledge/determinants.yaml flood_hazard]. |
| **"Sprinklered"** | "The building is not sprinklered" is evidence (`det.sprinklered` false), not a sprinkler scope. |
| **Common negations** | "No asbestos detected", "windows retained", "the order relates to cladding only", "balustrades retained", "lot-owner AC units by owners". |

---

## 11. Gaps in the built-in knowledge, and what the app will get wrong

1. **The work itself is never proposed.** Remediation has no scope defaults, so the cladding replacement
   appears only when a document or the user adds it; the default action `repair` misreads a replacement
   [K: knowledge/profile/scope_defaults.yaml empty_work_types; knowledge/works/actions.yaml work_type_defaults].
2. **Most of the team is never suggested.** Only the Town Planner (DA) and the Façade engineer (complex
   façade) are. PM, architect, fire engineer, structural, BCA, certifier and QS are missing
   [K: knowledge/works/package_defaults.yaml baselines]. The Façade, PM, hazmat and roof access
   disciplines have no RFP template.
3. **No fire safety order.** No profile key, fragment, cost line or RFP condition models an order, its
   compliance date or interim measures.
4. **Program.** `sk.generic`; no general meeting or levy gates; `fr.facade-2` and `fr.insulation-2` gated
   to new and extend; `fire-passive.external-walls` has no fragment; contract award, PC, OC, handover and
   defects gated off; no external-wall hold points (cavity barriers, fixings, façade test).
5. **Cost plan.** No preliminaries, scaffold, hoarding, removal, provisional-sum, long service levy or OC
   cost lines; P06 service authority fees always suggested; no remediation budget split.
6. **RFPs.** Boundaries end before tender in the architecture and fire templates; BCA and certifier
   cannot be told apart, and both block on a CDC; the fire template's order row is only an option.
6a. **Hazmat.** `uc.asbestos-cladding-disturbed` excludes the correct `replace` action, so an old
   building gets no hazmat prompt from the cladding item.
7. **Profile traps.** Exhaust and "kitchen"/"bathroom" wording adds services scope and consultants in
   code; the basement trigger can switch on retention design.
8. **NSW law.** The order regime, cladding registration, the Building Products (Safety) Act, Project
   Remediate, the RAB Act, the DBP Act (beyond one unverified RFP threshold) and the SSMA are not encoded.

The full machine list is in `shape.yaml gaps`.

---

## 12. Sources and unverified claims

**Opened and cited:**

- `knowledge/clusters/fire/{systems,rules,interfaces}.yaml`, `knowledge/clusters/envelope/{systems,interfaces,unforeseen,failure_modes}.yaml`,
  `knowledge/clusters/structure/{systems,unforeseen}.yaml`
- `knowledge/determinants.yaml` (facade_type, flood_hazard), `knowledge/profile/{taxonomy,flag_readings,scope_defaults,project_details}.yaml`
- `knowledge/works/{actions,package_defaults}.yaml`, `knowledge/costs/{elements,starter_lines,budget_splits}.yaml`
- `knowledge/program/{skeletons,fragments}.yaml`
- `knowledge/rfp/{architecture,fire-engineering,structural,bca-certification,town-planning,cost-planning,modules,skeleton}.yaml`
- `internal/rfp/{resolve,drivers}.go`, `data/intake/disciplines.json`
- `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and `index/*.md`
- seeds (not loaded by the generators): `ncc-reference-guide.md`, `multi-residential-apartments-guide.md`,
  `building-remediation-rectification-guide.md`

**Unverified (verified: false), to confirm from the instruments:**

- NCC 2022 Vol One Part C2 non-combustibility (C2D10 and its concessions, including sarking), the AS 5113
  verification method, Part J for alterations; the seed and the app use NCC 2019 numbering.
- EP&A Act 1979 fire safety orders (Division 9.3, Schedule 5); s 6.9 occupation certificate for an altered
  part; the cladding registration requirement in the EP&A Regulation.
- EP&A (Development Certification and Fire Safety) Regulation 2021: fire safety certificates, FRNSW
  referral of fire safety reports, critical-stage inspections for Class 2.
- Codes SEPP 2008: whether cladding replacement can be complying or exempt; design verification statement
  for a residential flat building DA.
- Building Products (Safety) Act 2017 and the 2018 ACP use ban (core threshold, classes, storeys).
- DBP Act 2020 and Regulation: building element definition, practitioner classes, lodgement timing
  before CC.
- RAB Act 2020: serious defect includes a banned building product; the ten-year window.
- Home Building Act 1989: HBCF multi-storey exemption (R4).
- SSMA 2015: s 106, s 108 special resolution for a different cladding, levies and borrowing.
- Roads Act 1993 s 138; SafeWork scaffold licensing and asbestos notification; long service levy.
- AS 1530.1 and CodeMark as product evidence; AS 4284 façade water testing.
- Project Remediate eligibility, loan terms, panels and administering agency (Building Commission NSW).

**Judgement calls the owner may want to change:**

- DA as the base pathway rather than CDC;
- traditional procurement as the base rather than D&C with novation;
- owner-funded as the base rather than Project Remediate;
- the PM, structural engineer and QS as base packages;
- full removal to DTS as the base, with the performance solution as a trigger;
- `det.facade_type: complex` and `flag.fire_specialist_systems: yes` as expected readings;
- `hdr.cond.operational_constraints: 24_7_occupied`: as in T09, the scorer should accept
  `live_environment` or `partial_occupation` too;
- R5 (BASIX) taken not to apply, because the work is to common property, not to a dwelling
  (verified: false).

**Reviewer rulings from round 1** (from the independent review, 30 October 2026, **not** from the owner):

- keep DA as the base; CDC (Fire Safety Code) and "no consent for order work" stay triggers;
- keep owner-funded as the base; Project Remediate stopped taking new buildings (verified: false);
- keep full removal to DTS as the base;
- the order regime (EP&A Act Div 9.3, Sch 5; FRNSW can also issue orders) stays verified: false;
- Project Remediate details (10-year interest-free loans, program PM and panels, Building Commission
  NSW) stay verified: false;
- DBP lodgement is legally "before work starts"; "before the CC" is a safe program dependency; the
  building compliance declaration comes before the OC application;
- FRNSW referral applies only where the DCFS Regulation lists the provision; stays verified: false;
- NCC 2022 C2D10, CV3, C2D14, C2D15 and Table C2D2 all stay verified: false.
