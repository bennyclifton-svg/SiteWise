# T08 Roof remediation, including asbestos roof replacement: domain map

Draft for owner review, 30 October 2026. The machine key is `shape.yaml` in this folder; it passes
`validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Any statute or standard clause not confirmed from
the instrument is marked **verified: false**. App behaviour marked "verified by running" was checked by
running the real Go (`procurement.SuggestedPackages`, `costs.Suggest`, `program.Generate`, `rfp.Resolve`)
through `go -overlay`, with no file written in the repo.

---

## 1. The job

**What it is.** The owner of an existing NSW warehouse (Class 7b) replaces a failing roof of corrugated
asbestos cement (AC) sheeting with a new metal roof. The building is usually still trading. A licensed
removalist takes off the AC sheeting, with an independent hygienist doing air monitoring and clearance.
The job then puts on:

- new metal sheeting, cappings and flashings, an insulation blanket and safety mesh;
- new skylights;
- replacement box gutters, sumps and overflows checked for capacity;
- roof ventilators;
- a permanent roof safety system (anchors, static lines, walkways, access ladder).

A structural engineer checks the purlins and their connections for the new sheeting span and for wind
uplift. Rooftop solar PV is often added once the asbestos is gone (trigger). [D]

**Base scenario** (used in `shape.yaml`):

- single-storey steel portal-frame warehouse of about 4,000 m², constructed in 1978;
- investor landlord as client, one logistics tenant who keeps operating;
- not sprinklered, not heritage, no roof plant, no PV;
- AC sheeting weathered but intact and bonded (non-friable), so a Class B licence applies
  (Work Health and Safety Regulation 2025 (NSW), which replaced the 2017 Regulation; clause numbers below are 2017 numbering, verified: false);
- exempt from consent (like-for-like), traditional procurement, one AS 4000 lump-sum contract with a
  roofing contractor who holds, or subcontracts, the asbestos removal licence [D].

**Scale.** $0.1–2 M [K: testing/synthetic/PROJECT_TYPES.md T08 row]. Cost is driven by roof area, access
and lifting, the removal method, the tenant's constraints and concealed conditions (purlins, debris
inside), not by GFA as such [D].

**Who procures it and how** [D]:

- **Client.** A landlord, a fund, or an owner-occupier. A facilities or project manager runs it; the PM
  is the SiteWise user, not a package. The lease decides who pays, the access notice the tenant must
  get, and any rent abatement.
- **Hygienist.** A licensed asbestos assessor or occupational hygienist, engaged directly by the owner so
  that it is independent of the removalist.
- **Roof specification and contract.** A roofing or building consultant writes the roof specification
  and drawings, runs the tender and acts as superintendent. On many jobs the structural engineer's
  commission carries the sheeting and fixing specification instead; either is accepted (as T09 does
  under owner ruling R2). A PM rarely writes a technical roofing specification.
- **Engineers.** A structural engineer for the purlins and anchor loads; a hydraulic engineer for the box
  gutters.
- **Contractor.** A roofing contractor as head contractor. The asbestos removal licence is held by the
  roofer or a subcontractor. The height-safety system is a specialist design-and-install subcontract
  element.

**What makes it go wrong** [D]:

- Someone falls through the brittle AC sheeting or a skylight. Falls through fragile roofs are the
  classic fatality on these jobs.
- AC debris or dust falls onto stock and racking below. The tenant loses the zone until clearance, and
  stock may have to be destroyed.
- Rain gets into an open bay overnight and floods live stock.
- The new lighter roof is fixed to purlins and cleats that cannot take the higher net uplift. AC sheeting
  is several times heavier than metal sheeting, and its dead load was resisting uplift.
- Purlin spacing set for AC does not suit the new profile
  [K: knowledge/clusters/envelope/unforeseen.yaml uc.existing-purlin-spacing-unsuited-to-replacement-sheet].
- Box gutters are replaced like for like though they were the cause of the leaks, or the sumps are not at
  the low point [K: uc.existing-box-gutter-sump-not-at-low-point,
  uc.box-gutter-overflow-built-over-discharges-into-ceiling].
- The tender says "no asbestos beyond the roof sheets", the removalist finds debris in the gutters and
  roof space, and the latent-condition clause does not cover it
  [K: knowledge/clusters/delivery/unforeseen.yaml uc.latent-condition-clause-does-not-cover-asbestos-tender-said-absent].
- Sprinkler ranges or lights hung from the purlins are disturbed (trigger).

---

## 2. Systems and work items

Action ids are the app's eight [K: knowledge/works/actions.yaml actions]. The remediation default is
`repair` [K: knowledge/works/actions.yaml work_type_defaults], which is wrong for every base item.

| system | in scope (base) | action | note |
|---|---|---|---|
| **site** | `site.demolition` | remove | Licensed removal of the AC sheeting, cappings, flashings and old skylight sheets, and the debris in the gutters. The system covers "licensed asbestos removal and clearance" [K: knowledge/clusters/structure/systems.yaml site.demolition]. `site.contamination` is never in scope; see §10. No ground works, so earthworks, stormwater detention and landscape are out (the stormwater connection is a trigger). |
| **substructure** | none | — | Not touched. |
| **structure** | `structure.alterations` | investigate | Purlins, cleats, bolts, bridging, rafters and holding-down checked for the new span, uplift and anchor loads. This system covers "assessment of existing structural condition and capacity … new loads on existing members" [K: clusters/structure/systems.yaml structure.alterations]. Strengthening is provisional. Replacing purlins is `structure.steel` replace (trigger `purlin-replacement`); the system covers purlins [K: structure.steel describes]. |
| **envelope** | `envelope.roof-coverings` | replace | New metal sheeting, cappings, flashings and fasteners. The system's label is "Pitched roof coverings", but its describes includes metal sheet roofing and roof flashings [K: clusters/envelope/systems.yaml envelope.roof-coverings]. |
| | `envelope.windows-doors` | replace | Skylights; roof-coverings excludes them [K: same file, roof-coverings excludes]. |
| | `envelope.thermal-insulation` | new | Foil-faced blanket under the sheeting for condensation and rain noise. The AC roof had none. |
| | others | — | Walls, cladding and vehicle doors are untouched. `envelope.external-waterproofing` is a **trap**: it covers membrane roofs, not a low-pitch metal deck (§10). |
| **interiors** | none | — | Office ceilings below are protected, not altered. |
| **fire-passive** | none | — | A single-storey 7b warehouse is usually Type C, with no fire-rated roof (verified: false). |
| **fire-active** | none in base | — | Sprinklers hung from purlins are a trigger (`sprinklered-building`). |
| **mechanical** | `mechanical.ventilation` | replace | Roof ventilators (whirlybirds, ridge vents) replaced with the sheeting by the roofer. Rooftop AC units are a trigger. |
| **hydraulic** | `hydraulic.roof-drainage` | upgrade | Box gutters, sumps, rainwater heads and overflows sized to current rainfall intensities; downpipes reconnected to the existing site system. The system covers gutters, box gutters, sumps, overflows and design rainfall intensity [K: clusters/services-wet-air/systems.yaml hydraulic.roof-drainage]. Adding overflows and capacity is `upgrade`. |
| **electrical** | none in base | — | High-bay lights and cable trays fixed to purlins are isolated and protected by the builder's electrician [D]. Solar PV is a trigger. |
| **comms-security** | none | — | Not affected. |
| **vertical-transport** | none | — | None in a single-storey warehouse. |
| **access-egress** | `access-egress.maintenance-access` | new | Roof access ladder or hatch, walkways, static lines and anchors. The system covers AS 1657 access to roofs [K: clusters/fire/systems.yaml access-egress.maintenance-access]. It does **not** name fall-arrest anchors or static lines (AS/NZS 1891.4, AS/NZS 5532, verified: false), so this is the closest id, recorded in gaps. |

---

## 3. Existing-building interfaces

The whole job is an existing-building interface [D].

- **Occupation.** The tenant keeps trading.
  - Each bay is stripped and re-sheeted under an **exclusion zone** below. Nobody works or stores goods
    under an open bay or under AC removal.
  - Stock is covered or moved bay by bay.
  - Forklifts are kept clear of the roofing work.
  - Working hours are agreed under the lease.
  - The `existing_building` and `live_site` RFP modules switch on from the work type and
    `hdr.cond.operational_constraints` (verified by running `rfp.Resolve`)
    [K: knowledge/rfp/modules.yaml; internal/rfp/drivers.go readFlags].
- **Weather.** A stripped bay is re-sheeted the same day or made weathertight. Rain on open bays over
  live stock is the main operational risk [D].
- **Services.** Light fittings, cable trays, sprinkler ranges (trigger) and roof plant (trigger) are fixed
  to or through the roof. They are isolated, protected or re-supported. There are no authority
  cut-overs on the base.
- **Hazardous materials.** The AC sheeting is the point of the job. In NSW, the Work Health and Safety Regulation 2025 (NSW) (which
  replaced the 2017 Regulation) and the SafeWork NSW code of practice apply. Clause numbers are 2017
  numbering and all are verified: false:
  - a licence for removal: Class B for more than 10 m² of non-friable asbestos, Class A for friable;
  - an asbestos removal control plan;
  - written notice to SafeWork NSW before licensed removal starts;
  - air monitoring (mandatory for Class A only, 2017 numbering r 475; for the Class B base it is a
    specification choice, and the map specifies it);
  - a clearance inspection and certificate by a person independent of the removalist;
  - an asbestos register and management plan for the workplace;
  - waste transported and disposed of at a licensed facility with tracking.

  Lead paint on old steelwork and PCB capacitors in old light fittings are a trigger. The app raises a
  hazardous-materials survey only as an Implication
  [K: knowledge/clusters/structure/consequences.yaml cq.pre-2004-fabric-hazardous-materials-survey].
- **Structure.**
  - Removing the sheeting can remove lateral restraint the purlins relied on. The contractor provides
    temporary restraint.
  - The lighter roof raises net uplift on purlins and cleats.
  - New anchors put point loads on purlins.
  - [K: clusters/structure/consequences.yaml cq.existing-structural-intervention-sequence-review;
    clusters/structure/unforeseen.yaml uc.existing-roof-framing-sags-under-new-cover].
- **Matching existing work.** Sheet profile and coating are chosen for the exposure. The new sheets are
  flashed to the existing wall cladding and parapets. Existing gutter outlets may not match new downpipe
  bosses [K: clusters/envelope/unforeseen.yaml uc.existing-gutter-outlet-does-not-match-downpipe-boss].
- **Dilapidation.** Record the condition of the tenant's areas, the office ceilings and the stock below
  before work starts, to resolve damage claims. Neighbours are affected only by crane lifts over a
  boundary [D].

---

## 4. Consultants and why

| discipline (app id) | stage engaged | need | why |
|---|---|---|---|
| Building or roofing consultant (`consultant.building`; `consultant.structural` accepted where the engineer carries the roof specification) | Investigation through defects | **Always** | Roof scope report, specification and drawings, tender, superintendent under AS 4000, sheeting and roof safety hold points. |
| Hazmat: licensed asbestos assessor or occupational hygienist (`consultant.hazmat`) | Investigation through completion | **Always** | Survey and sampling, friability, removal specification, review of the removal control plan, air monitoring, clearance per bay, register update. Independent of the removalist. |
| Structural (`consultant.structural`) | Investigation, documentation, construction | **Always** | Purlin and connection capacity for span, uplift and anchors; strengthening design; inspection before cover. |
| Hydraulic (`consultant.hydraulic`) | Investigation and documentation | **Always where there are box gutters** (most warehouses) | Box gutter, sump and overflow capacity to AS/NZS 3500.3 (verified: false). Gutter overflow is often why the job exists. |
| Roofing contractor (`trade.main_works`) | Procurement | **Always** | Head contract; asbestos removal licence (own or subcontracted); height-safety system as a design-and-install subcontract element. |

Conditional:

| discipline (app id) | need |
|---|---|
| Roof access (`consultant.roof_access`) | Roof plant needs permanent access routes and platforms (trigger `roof-plant-relocation`), or the owner wants an independent height-safety layout (trigger `independent-height-safety-design`). |
| Civil (`consultant.civil`) | Downpipes added or site stormwater capacity in question (trigger `drainage-capacity-upgrade`). |
| Certifier (`consultant.certifier`) | The work is not exempt and goes by CDC (trigger `cdc-pathway`). |
| Heritage and town planner (`consultant.heritage`, `consultant.town_planner`) | Heritage item (trigger). |
| Solar installer (`trade.electrical` stands in) | PV (trigger). The installer's accredited designer designs the array. |
| Fire services trade (`trade.fire_services`) | Sprinkler impairment and re-support (trigger). |
| Structural steel trade (`trade.structural_steel`) | Purlin replacement (trigger). |

**Never needed for the base:** architect, BCA, access, mechanical, electrical and fire services consultants,
geotechnical, surveyor, traffic, ESD, contamination (environmental) consultant.

**What the app does** (verified by running):

- It suggests **Hydraulic** (from the roof-drainage item [K: knowledge/works/package_defaults.yaml
  scope_additions[3]]) and **Mechanical** (from the roof-ventilator item [K: scope_additions[2]]).
- It suggests nothing else. There is no remediation baseline [K: package_defaults.yaml baselines], so
  the Building consultant, Hazmat and Structural never appear.
- Mechanical is keyed in `not_expected_procurement`.

---

## 5. Approvals and authorities

- **Planning.** Like-for-like replacement of roof sheeting, with no change to form, height or
  appearance, is normally treated as exempt or as maintenance needing no consent. Confirm it in
  writing, and check the council's asbestos policy (Codes SEPP 2008 provisions verified: false).
  - The app adds only `fr.exempt-confirmation` [K: knowledge/program/fragments.yaml
    fr.exempt-confirmation] (verified by running).
  - Exempt development must still meet the relevant deemed-to-satisfy provisions of the BCA (Codes SEPP
    general requirements, verified: false). That is why an exempt re-roof still needs AS 1562.1
    sheeting and AS/NZS 3500.3 drainage.
  - A raised roof, a changed form or appearance, or added roof plant can move it to CDC under Codes
    SEPP Part 5 or Part 5A, depending on whether the work is an alteration or an addition (verified:
    false), or to a DA on a heritage item.
- **No construction certificate and no occupation certificate** on the exempt path. The app still
  suggests P05 [K: knowledge/costs/starter_lines.yaml lines[4]].
- **Design and Building Practitioners Act.** It does not apply. Owner ruling R3 confines it to Class 2 and
  9c. Class 7b is outside it.
- **Home Building Act and HBCF.** They do not apply: this is not residential building work (verified:
  false).
- **SafeWork NSW** (Work Health and Safety Regulation 2025 (NSW); 2017 numbering, all verified: false):
  - licensed asbestos removal (r 458 exception for 10 m² or less of non-friable) and notice before it
    starts (r 466);
  - high-risk construction work SWMS for work at height above 2 m and for asbestos (r 291, r 299);
  - the duty to manage falls through fragile roofing (r 78, r 79).
- **NSW EPA.** Asbestos waste is transported and disposed of at a licensed facility, with tracking for
  loads over the threshold (POEO (Waste) Regulation 2014, verified: false).
- **Long Service Corporation.** The levy is 0.25% of the cost of the work where it is $250,000 or more
  including GST, paid before building work begins; with no CC it is paid directly. Source: NSW
  Government, long service levy page
  (https://www.nsw.gov.au/employment/rights-responsibilities/portable-long-service/bci-long-service-payments-scheme/long-service-levy).
  Kept verified: false until the owner rules. At the bottom of the $0.1–2 M range no levy is expected.
- **Council.** A crane or deliveries standing in the road need approval under the Roads Act 1993 s 138
  (trigger, verified: false).
- **DNSP**, for PV only: a connection application for embedded generation, with export limits and
  network studies above a size threshold (verified: false).
- **NCC.** Requirements that may apply to the new work (all verified: false):
  - metal roofing to AS 1562.1;
  - wind actions to AS/NZS 1170.2;
  - roof drainage to AS/NZS 3500.3, called up by NCC 2022 Volume One Part F1;
  - Section J fabric requirements, which apply to a conditioned envelope, so mostly not to an
    unconditioned warehouse;
  - fire hazard properties of roof lights.
- **Hold points** (the contract's) [D]:
  1. asbestos clearance of each bay before the area below is reoccupied;
  2. structural inspection of exposed purlins and strengthening before re-sheeting;
  3. sheeting fixing inspection against the manufacturer's pattern before handover (warranty; keyed in
     `program`);
  4. static line load test and certification before use.

  The app has none of these [K: knowledge/clusters/structure/interfaces.yaml; KNOWLEDGE_INDEX §4.4
  hold points].

---

## 6. Cost plan shape

Lines only; no amounts (see `shape.yaml cost_plan`).

- **Fees and charges.** Long service levy (0.25% at $250,000 or more including GST). No DA, CC or
  service-authority fees on the base.
- **Consultants.**
  - building consultant (specification, tender, superintendent);
  - hazardous materials consultant, plus air monitoring and clearance inspections priced as rates per
    visit;
  - structural engineer;
  - hydraulic engineer.
- **Construction** [D]:
  - preliminaries and live-warehouse controls;
  - access and lifting (crane, EWPs, edge protection, temporary safety mesh);
  - stock, racking and plant protection below;
  - dilapidation survey before and after;
  - licensed AC removal, decontamination and disposal;
  - provisional sum for purlin strengthening;
  - metal sheeting, flashings and cappings;
  - insulation blanket and permanent safety mesh;
  - skylights;
  - box gutters, sumps, overflows and downpipe connections;
  - roof ventilators;
  - roof safety system with certification;
  - temporary weather protection;
  - schedule of rates for concealed conditions.
- **Contingency and other.** Design contingency, construction contingency, escalation, exclusions (tenant
  stock moves and fit-out on purlins), and the owner's lease costs (tenant disruption, rent abatement).

**What the app produces** (verified by running `costs.Suggest`):

- fee lines F01 Hydraulic and F02 Mechanical;
- 01 Site works and external works (the asbestos removal, costed under `trade.civil_works`);
- 03 Structure, 04 Building envelope, 08 Mechanical services and 09 Hydraulic services;
- 13 Access and egress with child 13.7 Plant and maintenance access;
- starters P01, P02, P03, P05, P06 and P07. P04 is dropped for exempt
  [K: knowledge/costs/starter_lines.yaml lines[3]].

There is no budget split for remediation [K: knowledge/costs/budget_splits.yaml]. The lines it should
not carry are P05, P06 and the Mechanical fee (`not_expected_cost_plan`).

---

## 7. Program shape

The base has 35 activities; see `shape.yaml program`.

1. **Investigation.**
   - Records: the asbestos register and management plan, drawings, leak log, the lease.
   - Roof inspection with fall protection.
   - Then three assessments in parallel: asbestos survey and sampling, structural assessment of the
     purlins, and box gutter capacity.
   - Scope report with options and order of cost.
2. **Decision.** The owner approves scope and budget and consults the tenant (filed under
   `concept_design`).
3. **Approvals.** Confirm the exempt path.
4. **Documentation** (after the owner's scope decision, which may choose overlay). The building
   consultant's roof specification and drawings, purlin strengthening design, gutter design,
   and the hygienist's removal specification.
5. **Procurement.** Tender documents, then the tender period with a site inspection, then evaluation.
   Evaluation checks the removal licence class, insurances without an asbestos exclusion, the
   live-warehouse method and warranty terms. Then contract award.
6. **Construction.**
   - Before removal starts: review of the removal control plan, SWMS and work-at-height plan; the tenant
     operations plan; the dilapidation record; SafeWork notification; the long service levy; site establishment with crane, EWPs
     and edge protection.
   - Then **per bay**: protect below → remove AC with air monitoring → clearance hold point → purlin
     repairs and inspection → box gutter → insulation, mesh and sheeting → fixing hold point.
   - Gutters, skylights and ventilators, and the roof safety system with certification.
7. **Completion.** Final clearance, waste records and the updated register → PC → handover of warranties
   and certificates.
8. **Defects.** Defects liability period and final inspection (length verified: false).

**Logic.**

- The survey comes before the removal specification.
- The purlin assessment comes before the strengthening design, and both come before tender.
- SafeWork notification comes before removal.
- In each bay, clearance comes before reoccupation below, and the purlin inspection comes before
  re-sheeting.
- The roof safety system is certified before anyone relies on it.

**Long-lead items** [D]:

- roll-formed sheeting ordered to measured lengths;
- fabricated box gutters and sumps;
- skylights;
- replacement purlins (trigger);
- PV modules and inverters, and the DNSP approval (trigger);
- crane bookings.

**What the app produces.**

- **Skeleton.** There is no remediation skeleton, so the app uses `sk.generic`. Running `program.Generate`
  gives only two activities: "Confirm the exempt path" and one generic investigation row titled with the
  structure item.
- **Gated fragments.** These exist but are gated to new and extend (or to new, extend and refurb), so
  none fire:
  - `fr.roof-7b` ("Install the metal roof"), `fr.demolition-7b` and `fr.steel-7b`;
  - the 7b documentation, tender and site-establishment fragments;
  - contract award, practical completion, handover and defects.
- **Missing fragments.** No fragment names asbestos, roof drainage on 7b, skylights or maintenance
  access.

[K: knowledge/program/fragments.yaml; testing/synthetic/reference/index/program-fragments.md]

---

## 8. RFP shape

**Building or roofing consultant** (no template; the app uses the generic `tpl.rfp-capex`).

- **Must carry:** the scope report with removal or overlay options; the roof specification and drawings;
  coordination of the hazmat, structural and hydraulic inputs; tender management and recommendation; the
  AS 4000 superintendent role (SOP Act, verified: false); the sheeting fixing and roof safety hold points;
  PC, defects and warranties.
- **Must not carry:** planning services, architectural concept stages, or asbestos monitoring and
  clearance (which stay independent with the hygienist).

**Hazardous materials consultant** (no template; the app uses the generic `tpl.rfp-capex`).

- **Must carry:**
  - register review;
  - survey and sampling of the sheeting, cappings, skylights, gutters and high-level surfaces;
  - friability assessment;
  - removal specification with sequence, exclusion zones and clearance hold points;
  - review of the removalist's removal control plan and notification;
  - air monitoring;
  - clearance inspection and certificate per area, independent of the removalist;
  - register and management plan update;
  - returnables: licence or hygienist qualification, NATA laboratory, PI and PL insurance without an
    asbestos exclusion.
- **Must not carry:** contaminated land or soil assessment, planning reports, or the removal works
  themselves.

**Structural** (template `structural`). The expected state is in `shape.yaml`, in the template's own keys.

- **Must carry:**
  - the existing structure assessment (included because the work type is remediation
    [K: knowledge/rfp/structural.yaml stages existing_assessment]);
  - a design basis for uplift on the lighter roof;
  - strengthening as an option (`strengthening` O);
  - anchor loads under delegated design;
  - safety in design for the fragile roof;
  - construction-phase inspection before cover;
  - an inspection schedule.
- **Must not carry:** a geotechnical brief, a shared model, concept and schematic design, planning
  approval, in-ground services footings, or regulated design (DBP does not reach 7b).
- **What the app gets wrong** (verified by running `rfp.Resolve`; 11 differences in `app_differs`):
  - Scale is simple, because Class 7b is not in `scale.complex` [K: structural.yaml scale].
  - The construction phase is therefore only an option (`default: option`, and it falls after the
    boundary end).
  - Concept and schematic design, the geotechnical brief and in-ground services footings are always
    included.
  - Five rows that do not belong are priced as options or excluded with a named party (shared model,
    specialist elements, services fixings, metalwork, super-flat floor; compaction and demolition
    drawings X).

**Hydraulic** (template `hydraulic-services`).

- **Must carry:**
  - investigation of the existing roof drainage;
  - roof drainage design (`roof`);
  - retained-system investigation of the downpipes and site connection;
  - tender documentation;
  - coordination with the structural engineer and the roofer;
  - construction review as an option.
- **Must not carry:** domestic water, sanitary, pumps, rainwater reuse, an authority stage, or site
  stormwater design.
- **What the app gets wrong** (24 differences; verified by running):
  - The template includes domestic water, sanitary, pumps and rainwater design whenever the design stage
    is included [K: knowledge/rfp/hydraulic-services.yaml positions].
  - It blocks issue on client flags (`hyd_investigation`, `hyd_planning`, `hyd_treatment`, `hyd_process`,
    `hyd_external`, `hyd_declarations`) that no document can set.
  - It includes a concept stage.
  - It prices tender only as an option, because tender falls after the boundary end (`design`).

  A roof-drainage commission comes out as a whole-building hydraulic brief.

**Conditional:**

- **Roof access**: no template.
- **Civil** (`civil`): the template would include a utility plan, discharge, drainage model,
  earthworks suitability and retained services (all I for this profile, verified by running), far more
  than a downpipe connection check.
- **Certifier** (`bca-certification`): blocks on `cdc` [K: knowledge/rfp/bca-certification.yaml
  unsupported_pathways].
- **Heritage**: no template.

---

## 9. Triggers

There are 16 triggers; full keys are in `shape.yaml`. Each changes procurement, cost, program or RFP.

| id | how documents show it (including indirectly) | how a document denies it | adds |
|---|---|---|---|
| `friable-asbestos` | "can be crumbled by hand pressure; the hygienist classifies it as friable", "must be removed by a Class A licensed removalist" (weathering or moss alone does not make AC friable) | "weathered but intact and bonded; Class B", "heavily weathered and moss-covered but bonded" | Class A method, enclosure, mandatory monitoring, licensed assessor clearance |
| `asbestos-debris-inside` | "AC debris on the racking tops", "fragments on stored goods", "asbestos contamination of the box gutters" | "dust samples from the racking were negative" | Internal decontamination, stock cleaning or disposal, tenant decant, internal clearance |
| `purlin-replacement` | "purlins fail in uplift", "1.2 m spacing suits AC", "heavily corroded below the box gutters" | "engineer confirms purlins adequate" | Structural steel trade, purlin supply (long lead), temporary restraint |
| `drainage-capacity-upgrade` | "box gutters overflowed into the dispatch area", "undersized for the 1% AEP intensity", "new connection to the site pit" | "existing capacity adequate; overflows only" | Civil consultant, larger gutters and downpipes, site connections |
| `solar-pv` | "300 kW rooftop array", "power purchase agreement", "roof designed for the future array" | "no solar proposed" | Solar installer, DNSP application, PV loads on purlins, PV after roof |
| `sprinklered-building` | "ranges hung from the purlins", "notify the insurer of any impairment" | "not sprinkler protected" | Fire services trade, impairment plan, re-support and test |
| `roof-plant-relocation` | "rooftop package AC units craned off", "exhaust fans need walkways", "plant reached across fragile sheeting" | "no plant on the roof" | Roof access consultant, plant lift and recommission, platforms |
| `lead-paint-and-other-hazmat` | "lead-based primer on purlins", "PCB capacitors in old fittings", "SMF above the office" | "survey found none" | Wider survey, lead controls, PCB disposal |
| `sensitive-stock-occupancy` | "food distributor under HACCP", "24/7 dispatch", "pharmaceutical stock to be relocated" | "warehouse vacant", "packaged building materials covered in place" | Sealed protection or decant, off-site storage, night work |
| `cdc-pathway` | "roof raised 600 mm", "CDC under the Codes SEPP commercial and industrial codes", "colour changes the approved appearance" | "certifier confirmed exempt" | Certifier, CDC, occupation certificate; removes the exempt confirmation |
| `heritage-listed` | "Schedule 5 local item", "sawtooth roof is significant fabric" | "not a heritage item" | Heritage consultant and town planner, DA or cl 5.10(3) minor works (verified: false) |
| `crane-road-occupancy` | "crane will stand in the street", "road occupancy licence and traffic control plan" | "crane in the rear yard" | s 138 approval, traffic control |
| `storm-damage-insurance` | "hailstorm of 14 November", "loss adjuster approved", "make-safe tarping" | "planned capital works, no claim" | Make-safe, loss adjuster scope, insurer price approval |
| `independent-height-safety-design` | "owner has engaged a height safety consultant", "portfolio height safety standard" | "roofing contractor designs, installs and certifies" | Roof access consultant (no template) |
| `overlay-over-ac` | "installed over the existing AC on a spacer system", "Option B: overlay" | "removed in full before re-sheeting" | Spacer system, hygienist work method and inspections for fixing through AC, spacer fixing inspection, register update; removes the removal item, removal specification, notification, removal, clearance hold point, purlin repair row, final clearance, removal and monitoring costs, and the removal-only hazmat RFP items |
| `strata-industrial-unit` | "roof is common property of Strata Plan 54321", "roof over units 1 to 6 is continuous" | "single ownership on its own lot" | General meeting and levy, notices to each unit |

**Variants** (the eight most common): overlay over AC, purlin replacement, drainage capacity upgrade, solar PV, sprinklered
building, asbestos debris inside, roof plant relocation, CDC pathway.

---

## 10. Keyword traps and negations specific to this type

| trap | right reading |
|---|---|
| **"Asbestos contamination"** of gutters, racking or the warehouse | A hazardous **building material**, not contaminated land. It matches the `potentially_contaminated_land` trigger `contaminat` [K: knowledge/determinants.yaml]. Expected: not true. Asbestos in soil or fill is `site.contamination` [K: clusters/structure/systems.yaml site.demolition excludes]; asbestos in the building is `site.demolition`. No Environmental consultant and no RAP. |
| **"Remediation"** | Roof rectification, not land remediation. |
| **"Flooding" / "flooded the dispatch area"** | Box gutter overflow, **not** a flood-prone site. It matches the `flood_hazard` trigger `\bflood` [K: determinants.yaml]. Expected: `det.flood_hazard` not true. |
| **"Low-pitch roof" / "flat roof"** | A 1–3° warehouse metal deck is still metal roofing (`envelope.roof-coverings`), not a membrane roof (`envelope.external-waterproofing`). The app's metal-roof rule carries an unverified claim that roofs below 5° need a membrane [K: clusters/envelope/rules.yaml rule.ncc.metal-roof-installation numbers], which invites the wrong reading. |
| **"Demolition" / "strip-out"** | Matches the implied `site.demolition` row [K: knowledge/profile/scope_defaults.yaml implied_by strip-out]. Correct here for the AC removal, but "demolition of the building" is a different job (T10-like). |
| **"Office kitchen", "staff amenities", "bathroom"** | Warehouses have an office and amenities under the same roof. The implied-scope patterns fire **in code** [K: scope_defaults.yaml implied_by wet-room, kitchen; internal/profile/implied.go]. The result is wet-area, sanitary, water and exhaust scope, then Hydraulic and Mechanical consultants and 08/09 lines. They are not in scope. |
| **"Common wall"** between strata units | Fires the party-wall pattern: adjoining structures, compartments and acoustic separation. |
| **"Stairs"** to the mezzanine or office | Fires stairs-and-ramps. |
| **"Brickwork parapet", "masonry"** | Fires structure.masonry. |
| **"Sprinkler"** in an insurer letter ("not sprinkler protected") | Routes to `det.sprinklered`; the enumerated negative is evidence. |
| **"Battery" / "fire"** near PV | "Battery storage … fire risk" matches `fire_special_hazards` [K: knowledge/profile/flag_readings.yaml fire_special_hazards]. A rooftop PV array without batteries is not a special hazard. |
| **"Dangerous goods"** stored by the tenant | Matches `fire_special_hazards` and suggests `hdr.cond.hazardous_process: dangerous_goods`. Relevant only if the works change the store; in the base it is a tenant fact. |
| **"Lodge"** (SafeWork notification, waste dockets) | Matches `architect_lodges` [K: flag_readings.yaml]. There is no architect. |
| **"Stage 1", "Stage 2"** for bays | Bay sequencing is not staged delivery. It matches the `staged` flag pattern `\bstages? [1-9a-c]\b`. Expected: `staged` not yes unless separable portions are stated. |
| **"Constructed in 1978"** vs **"completed in 1978"** | Only "built in" or "constructed in" a year matches the year trigger [K: determinants.yaml existing_building_year]. |
| **"Skylight"** | Belongs to `envelope.windows-doors`, not roof-coverings. |
| **"Clearance"** | An asbestos clearance certificate, not a building height clearance, a site "clearance", or a crane clearance. |
| **"PC"** | Practical completion versus a prime cost sum. |
| **"Exempt"** | Planning exempt development versus GST-exempt or "exempt from HBCF". |
| **Common negations** | "The sheeting is non-friable", "no asbestos debris found inside", "purlins adequate", "no plant on the roof", "not sprinklered", "crane within the site", "no PV proposed". |

---

## 11. Gaps in the built-in knowledge, and what the app will get wrong

1. **The work itself is invisible.** Remediation has no scope defaults, so no roof, asbestos or gutter
   item exists until a document or the user adds one [K: scope_defaults.yaml empty_work_types]. The
   default action `repair` is wrong for all eight base items.
2. **Consultants.** The Building consultant, Hazmat and Structural are never suggested. Hydraulic comes in only through the
   roof-drainage item, and Mechanical wrongly through the roof ventilators.
   - Hazmat has no RFP template, and its filing aliases lack "occupational hygienist" and "licensed
     asbestos assessor" [K: data/intake/disciplines.json consultant.hazmat].
   - No id exists for a roofing contractor, an asbestos removalist, a height-safety installer or a solar
     installer.
3. **Program.** `sk.generic` with two activities.
   - Every 7b construction and process fragment is gated to new or extend.
   - Nothing models asbestos (notification, removal control plan, monitoring, clearance) or bay-by-bay
     work under a live tenant.
   - There is no hold point for clearance or purlin inspection.
4. **Cost plan.** Asbestos removal lands in "01 Site works and external works" under civil works.
   - No preliminaries, access, stock protection, provisional sums or long service levy.
   - P05 and P06 are suggested though they do not apply.
   - There is no budget split.
5. **RFPs.**
   - The hydraulic RFP becomes a whole-building brief that blocks on unsettable client flags.
   - The structural RFP carries concept, geotechnical and footing rows and makes construction-phase
     inspection an option.
   - The certifier RFP blocks on CDC.
6. **Profile.**
   - "Asbestos contamination" and "flooding" route to the contaminated-land and flood questions.
   - Amenities and kitchen wording add wet-area and services scope in code.
   - "Completed in" years are missed.
   - The low-pitch membrane claim may misfile the roof.
7. **Knowledge.**
   - WHS asbestos law is not encoded.
   - There is no friable versus bonded distinction.
   - There is no system id for static lines and anchors.
   - `uc.asbestos-found-in-existing-building` does not fire on `remove` or `investigate`
     [K: clusters/structure/unforeseen.yaml].

Two more from review: scope additions fire on any action, including `retain` (so protected sprinklers
bring a Fire Services consultant), and under traditional procurement the head contract proposal drops
the roof safety design line [K: internal/profile/works.go ProjectWorkScope;
internal/procurement/proposed.go ProposeHeadContract].

The full machine list is in `shape.yaml gaps` (25 entries).

---

## 12. Sources and unverified claims

**Opened and cited:**

- `knowledge/clusters/structure/{systems,consequences,unforeseen}.yaml`
- `knowledge/clusters/envelope/{systems,rules,unforeseen}.yaml`
- `knowledge/clusters/services-wet-air/systems.yaml`, `knowledge/clusters/fire/{systems,unforeseen}.yaml`
- `knowledge/clusters/electrical-comms/{systems,failure_modes,interfaces,unforeseen}.yaml`
- `knowledge/clusters/delivery/unforeseen.yaml`
- `knowledge/works/{actions,package_defaults}.yaml`
- `knowledge/costs/{elements,starter_lines,budget_splits}.yaml`
- `knowledge/program/{skeletons,fragments}.yaml`
- `knowledge/rfp/{structural,hydraulic-services,civil,bca-certification,modules}.yaml`, through the index
  and `rfp.Resolve`
- `knowledge/profile/{taxonomy,flag_readings,scope_defaults,project_details,project_facts}.yaml`,
  `knowledge/determinants.yaml`
- `data/intake/disciplines.json`
- `internal/costs/suggest.go`, `internal/rfp/resolve.go`, `internal/rfp/drivers.go`
- `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and `index/*.md`

**Owner rulings applied:** R3 (the DBP Act reaches Class 2 and 9c only, so not this Class 7b job). R4
and R7 are residential and do not apply.

**Unverified (verified: false), to confirm from the instruments:**

- Work Health and Safety Regulation 2025 (NSW), which replaced the 2017 Regulation. Numbers below are 2017 numbering; each
  2025 clause is verified: false:
  - licence classes and the 10 m² Class B threshold;
  - r 464 asbestos removal control plan;
  - r 466 notice (5 days);
  - r 473 and r 474 clearance and its independence;
  - r 475 air monitoring;
  - r 425 and r 429 register and management plan;
  - r 458, the 10 m² non-friable exception to licensed removal;
  - r 78 and r 79, managing the risk of falls;
  - r 291 and r 299, high-risk construction work and SWMS;
  - the definition of friable asbestos.
- The SafeWork NSW code of practice on removing asbestos.
- POEO (Waste) Regulation 2014: asbestos transport, disposal and tracking.
- Codes SEPP 2008: exempt status of like-for-like re-roofing; the Part 5 Commercial and Industrial
  Alterations Code and Part 5A additions code; the general requirement that exempt work meets the
  BCA deemed-to-satisfy provisions; heritage limits; LEP cl 5.10(3).
- SEPP (Transport and Infrastructure) 2021: exempt rooftop solar; DNSP embedded-generation thresholds.
- NCC 2022 Volume One:
  - Part F1 roof drainage (AS/NZS 3500.3);
  - Section J applicability to an unconditioned warehouse;
  - roof-light fire hazard properties;
  - Type C construction for a single-storey 7b.
- Standards: AS 1562.1, AS/NZS 1170.2, AS/NZS 1891.4, AS/NZS 5532, AS 1657, AS/NZS 4389, AS/NZS 5033,
  AS/NZS 4777.
- Roads Act 1993 s 138.
- The long service levy: 0.25% at $250,000 or more including GST, before work begins (reviewer's
  nsw.gov.au source; awaiting an owner ruling), and direct payment without a CC.
- The EP&A (Development Certification and Fire Safety) Regulation 2021 annual fire safety statement.
- That HBA licensing and HBCF do not reach this non-residential work.
- The defects liability period length.

**Judgement calls the owner may want to change:**

- **The roof safety system** is a design-and-install subcontract element of the head contract, not a
  separate `consultant.roof_access` package, which appears only in a trigger. Larger portfolios often
  engage a height-safety consultant directly.
- **Hydraulic in the base.** Most warehouses have box gutters. A warehouse with eaves gutters only may
  not need one.
- **`site.demolition` remove** for the AC removal, rather than folding the removal into
  `envelope.roof-coverings` replace.
- **`structure.alterations` investigate** for the purlin check, with strengthening provisional, rather
  than `structure.steel` upgrade.
- **AS 4000** as the base contract. AS 4906 (minor works) or AS 4905 (minor works, design and construct) is also seen at the lower end of the
  range.
- **`consultant.building` as the specification author and superintendent** (review finding 2), with
  `consultant.structural` accepted where the engineer carries the roof specification.
- **No PM package.** The PM is the SiteWise user. T17 keys a PM package because there the owners
  corporation appoints one.
- **`hdr.cond.operational_constraints: live_environment`.** Documents usually say "the tenant will
  continue to operate", which fits `partial_occupation` just as well. All three occupied values set
  `live_operations: yes` [K: internal/rfp/drivers.go readFlags]. **The scorer should accept any occupied
  value, and must not score this key on exact value.**
