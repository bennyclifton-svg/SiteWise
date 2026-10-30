# T10 Knock-down rebuild house: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (e1a5389 plus the working tree), the reviewed T01 pilot map, the owner rulings
R1 to R8, and professional practice.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. Clause numbers I could not check
against the instrument itself are written `verified: false`. No prices, rates or week durations appear.
The only durations named are the app's own `du.*` ids.

The closest reviewed map is T01 (new house). This map repeats T01 only where the new house is the same,
and says so. What is new here is the existing house: demolition, hazardous materials, service
disconnections, dilapidation of the neighbours, and the trees and stormwater of an established lot.

---

## 1. The job

**What it is.** The owner demolishes the existing house on their own suburban lot and builds a new,
two-storey detached house (NCC Class 1a) with an attached double garage (Class 10a) in its place.
The base scenario is:

| Item | Base value |
|---|---|
| Lot | Established Torrens-title lot of about 550–700 m² in an R2 Low Density Residential zone in Greater Sydney. Reticulated water and sewer, kerb and gutter, footpath, an overhead electricity service from the street pole, NBN, and a natural gas service to the existing house. A council street tree in the verge and two mature trees in the rear yard that the design keeps clear of. |
| Existing house | A 1962 single-storey brick-veneer and fibro (asbestos cement) house of about 120–150 m², with a detached fibro garage and a garden shed. It has asbestos-cement eaves, wet-area linings and garage cladding, likely lead paint, and a vinyl kitchen floor. The house drains to the kerb through an old earthenware line and kerb outlet. There is an existing driveway and vehicle crossing. |
| New house | 4 bedrooms, 3 bathrooms, about 280–320 m² GFA over two storeys, double garage under the main roof. Construction as T01: waffle or stiffened raft slab, timber frames and trusses, brick veneer and lightweight cladding, metal roof, aluminium windows, internal stair. All-electric (heat-pump hot water, induction cooktop), ducted air-conditioning, BASIX rainwater tank. |
| Pathway | One complying development certificate (CDC) from a registered certifier covering demolition (Demolition Code) and the new dwelling (Housing Code), both under SEPP (Exempt and Complying Development Codes) 2008 (Parts 3 and 7, verified: false). |
| Procurement | Traditional. The owner appoints an architect or building designer and the consultants, then runs a select tender with three or four residential builders. The builder's lump-sum HIA or MBA contract includes demolition and asbestos removal by licensed subcontractors. |
| Occupation | The owners move out before demolition and rent elsewhere until handover. |
| Value | Inside the PROJECT_TYPES range of $0.6–2.5M. The shape names no amount. |

**Why CDC and traditional for the base.** PROJECT_TYPES gives "CDC or DA". On established lots in
Greater Sydney a code-compliant knock-down rebuild usually goes CDC because it is faster and needs no
council assessment. [D] The base keeps T01's traditional, consultant-led procurement so that the two maps
differ only in the existing house and the pathway. The project-home "knock-down rebuild package", where
the builder designs and demolishes, is very common by volume. It is the trigger `builder-designs`, as it
is in T01, because the app models it badly (§11). Reviewer accepted this base (T10 REVIEW §2); the owner may override.

**Who procures it and how.** The owner is usually a private individual or couple who has lived in the
house. They engage the architect or designer first. The architect brings in:

- the surveyor;
- the hazardous materials consultant (pre-demolition asbestos survey);
- the geotechnical engineer;
- the structural and civil engineer;
- the BASIX/NatHERS assessor.

The owner appoints the certifier, who determines the CDC and becomes the principal certifier. The builder
is chosen by select tender and signs a Home Building Act compliant contract. HBCF cover is needed before
any deposit or work, because the contract is over $20,000 including GST (R4). [D]

**What makes it go wrong.** [D] T01's list applies: the design is over budget at tender, site information
comes too late, stormwater, defects and long-lead items. The knock-down adds the following:

1. **Asbestos found late.** A survey done without destructive sampling misses asbestos under the floor
   coverings, in the wall cavities, in the meter board or in the soil under the old garage. Removal
   stops the demolition, needs a SafeWork NSW notification and a clearance, and becomes a variation.
2. **Services not disconnected in time.** The DNSP has to abolish the overhead service and the gas
   network has to remove the meter, and both can take weeks. The demolisher cannot start until both are
   done, and each is ordered through a retailer or an accredited service provider that the owner must
   brief. The house has no power once the electricity goes, so the builder needs temporary power from
   a separate connection.
3. **CDC eligibility lost.** CDC is lost when the lot or design misses a Housing Code standard, when a
   tree that needs a permit must come out, when the lot is in a heritage conservation area, or when a
   flood or bushfire control applies. The job then reverts to a DA, which costs time (triggers
   `da-pathway`, `heritage-conservation-area`).
4. **Neighbour claims.** Demolition vibration, dust and excavation lead to crack claims from adjoining
   owners. Without a pre-demolition dilapidation report there is no baseline.
5. **What is under the old house.** Removing it can reveal old footings, rubble, buried drainage, an old
   septic tank, fill, or asbestos in soil. The geotechnical boreholes had to stay clear of the house, so
   the site class may change once it is gone. Cost these as provisional sums and latent conditions.
6. **The existing drainage.** The old kerb outlet line, an illegal connection of stormwater to the
   sewer, or a rear yard rubble pit with no legal point of discharge. The new, larger roof and hard
   surfaces need a proper connection, and may need on-site detention (OSD) (triggers
   `stormwater-no-gravity`, `osd-required`).
7. **Owner costs outside the contract.** Rent, removal and storage for the build period, insurance of
   the vacant house before demolition, and the bridging or construction loan timing. None of these is
   in the builder's price.

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. Actions are from `knowledge/works/actions.yaml`. For
work type `new` the app starts from the house/new default list (41 systems, action `new`)
[K: knowledge/profile/scope_defaults.yaml classes[0]; knowledge/works/actions.yaml work_type_defaults].
Items marked **+** are ones a competent PM expects that the default list lacks. Where the new house is
the same as T01, the row says "as T01".

### `site`: Site, ground and external works

| System | Action | Why |
|---|---|---|
| `site.demolition` **+** | remove | The defining work item. Demolish the house, garage and shed and remove their slabs, footings, paths and old driveway. Before that: the hazardous materials survey and asbestos register, licensed asbestos removal and clearance, lead-paint controls, a demolition work plan to AS 2601, dust, noise and vibration control, and waste separation and tracking. All of these sit in the system's own definition [K: knowledge/clusters/structure/systems.yaml site.demolition]. It is not in the default list. A read document that says "demolish" or "demolition" adds it through `implied_by strip-out`, but with the work-type default action `new`, not `remove` [K: knowledge/profile/scope_defaults.yaml implied_by strip-out]. Only `remove` makes `fr.demolition-1a` fire [K: knowledge/program/fragments.yaml fr.demolition-1a]. |
| `site.adjoining-structures` **+** | investigate | Dilapidation surveys of the adjoining houses, fences and boundary walls before demolition, and again at completion. Protection of neighbouring property during demolition [K: knowledge/clusters/structure/systems.yaml site.adjoining-structures]. Not in the default list. As an `investigate` item it gets only the generic Investigation row, because no `fr.investigate-*` names this system [K: internal/program/generate.go investigationRows]. |
| `site.trees` **+** | retain | Protect the council street tree and the retained rear-yard trees during demolition and construction. Fence their tree protection zones and keep machinery off the roots. Not in the default list. A `retain` item brings no program row, because `fr.tree-protection` needs `new`, `alter` or `repair` [K: knowledge/program/fragments.yaml fr.tree-protection]. |
| `site.survey-boundaries` | new | Building set-out and identification surveys, as T01. The pre-design detail survey, which picks up the existing house, is a consultant task (§4). In the default list. |
| `site.ground-conditions` **+** | investigate | Geotechnical investigation for the AS 2870 site classification, with boreholes in the yard and the driveway clear of the existing house. It may need confirming after demolition. Not in the default list. |
| `site.earthworks` | new | Strip the old building platform, remove old fill, cut and fill to the new slab platform, dispose of spoil. In the default list. |
| `site.erosion-sediment-control` | new | Sediment fence, stabilised entry and stockpile controls before demolition starts. Demolition is the first disturbance. In the default list. |
| `site.stormwater-detention` | new | New site stormwater from downpipes and the tank overflow to the kerb. The old earthenware line and kerb outlet are removed or replaced, and any downpipe found connected to the sewer is disconnected. In the default list. |
| `site.services-connections` | new | The existing water service and meter, sewer junction and street electricity connection point are reused or upgraded, and the private lines inside the lot are all new. The disconnections and abolishments before demolition are part of `site.demolition` and the service providers' work (§3). The work type is new and the action `new`; `alter` is defensible but would not change the fragment, because `fr.services-connections-1a` accepts both. In the default list. |
| `site.vehicle-access` | new | New driveway and paths. The old driveway is demolished. In the default list. |
| `site.public-domain` **+** | new | Council vehicular crossing rebuilt to the new driveway position. The redundant layback is reinstated to kerb and gutter and the footpath made good, under a Roads Act s138 consent (verified: false). Not in the default list. [D] |
| `site.landscape` | new | Turf, planting, paving, new dividing and front fencing where agreed with the neighbours, letterbox. In the default list. |
| `site.contamination` | out (trigger) | Asbestos in soil, or a house on the loose-fill asbestos register (triggers `asbestos-in-soil`, `loose-fill-asbestos`). Building hazardous materials are `site.demolition`; soil is this system [K: knowledge/clusters/structure/systems.yaml site.demolition excludes]. |
| `site.pools` | out (trigger) | An existing pool to be demolished and backfilled (trigger `existing-pool`). |
| `site.retaining-walls`, `site.flooding`, `site.acid-sulfate-soils` | out | Conditions and variants, not base scope. |
| `site.waste-storage`, `site.impact-protection`, `site.loading-docks`, `site.signage` | out | Not a house work item, or commercial. |

### `substructure`, `structure`, `envelope`, `interiors`

As T01, all action `new`. The new house is the same building on a cleared lot:

- `substructure`: footings, slab-on-ground and termite management are in the default list.
- `structure`: timber is in the default list. Brick veneer (`structure.masonry`) and steel or LVL beams
  (`structure.steel`) are **+** items.
- `envelope`: the eight default systems.
- `interiors`: the nine default systems, plus appliances (`interiors.ffe`) as **+**.

Two points differ from T01:

- **No `substructure.underpinning` and no `structure.alterations`.** Nothing of the old house is kept. A
  knock-down that keeps a wall or slab is a different job (T05, `extend` or `refurb`).
- **Old footings under the new slab.** These are removed under `site.demolition`, not re-used. Leaving
  part of the old footings in place under a new slab invites differential movement. [D]

### `fire-passive`

Out of scope, as T01. A detached Class 1a house has no rated elements unless an external wall is within
900 mm of a boundary (ABCB Housing Provisions Part 9.2, verified: false). Garages on narrow lots often
are. That is a design variant, not a trigger.

### `fire-active`

`fire-active.smoke-alarms`, action `new`, as T01. Everything else is out.

### `mechanical`

`mechanical.air-conditioning` and `mechanical.local-exhaust`, action `new`, as T01. The existing
house's split systems or ducted unit are removed under demolition. A refrigerant gas recovery by a
licensed technician is needed before an air-conditioner is scrapped (verified: false). [D]

### `hydraulic`

| System | Action | Why |
|---|---|---|
| `hydraulic.cold-water`, `.hot-water`, `.sanitary`, `.roof-drainage` | new | As T01. In the default list. |
| `hydraulic.non-potable-water` **+** | new | BASIX rainwater tank plumbed to toilets, laundry and garden. Not in the default list, and it has no fragment. |
| `hydraulic.gas` **+** | remove | The existing gas meter and service are abolished by the gas network (Jemena) before demolition. The new house is all-electric. Not in the default list. With `remove` no fragment fires (`fr.gas-1a` needs `new` or `replace`) [K: knowledge/program/fragments.yaml fr.gas-1a]. The `hydraulic` scope addition already suggests a Hydraulic consultant. |
| `hydraulic.backflow-prevention`, `.trade-waste`, `.specialist-process` | out | As T01. |

### `electrical`

| System | Action | Why |
|---|---|---|
| `electrical.utility-supply` | new | New service and metering from the DNSP network, after the old overhead service has been abolished. Long lead. In the default list. |
| `electrical.temporary-power` **+** | new | Builder's temporary supply. More important than on T01, because the house loses its power when the service is abolished. Not in the default list, so `fr.temporary-power` does not fire [K: knowledge/program/fragments.yaml fr.temporary-power]. |
| `electrical.switchboards`, `.cabling-wiring`, `.lighting` | new | As T01. |
| `electrical.solar-pv` | out of base | An existing roof PV system is a less common variant. A CEC-accredited installer removes it before demolition; it is reinstated or replaced. [D] |
| Others | out | As T01. |

### `comms-security`

`comms-security.carrier-lead-in` (the old copper or NBN lead-in removed, then a new NBN lead-in) and
`comms-security.structured-cabling`, action `new`, as T01. Others are out.

### `vertical-transport`

Out, as T01.

### `access-egress`

`access-egress.stairs-and-ramps` and `access-egress.barriers`, action `new`, as T01 (two-storey house,
both **+**). Livable housing is optional in NSW and not expected (R6). The other children are out.

---

## 3. Existing-building interfaces

This is the part that makes T10 different from T01. The existing building is demolished, not kept, so
there is no matching of existing work and no structural tie-in. Each interface below ends when the
lot is clear.

**Occupation.** [D]
- Base: the owners move out before demolition, and the house stays vacant until the demolisher takes
  possession.
- A vacant house needs to be secured. The owner's home insurance usually lapses or is conditioned once
  the house is vacant or handed over for demolition, so the owner's cover and the builder's contract
  works cover must meet with no gap.
- Rent, removal and storage for the whole build period are a real owner cost that no contract carries.
- The postal address, utility accounts with retailers, and council rates continue. Retailer accounts
  must be closed as part of the abolishment requests.
- In the app, `hdr.cond.operational_constraints: vacant` turns off the RFP `live_operations` flag
  [K: internal/rfp/drivers.go readFlags].

**Services disconnections.** [D] Every service is disconnected and made safe before demolition. Each one
is a separate application with its own lead time:

- **Electricity.** One neutral sequence for Ausgrid, Endeavour Energy and Essential Energy (the whole
  sequence `verified: false`):
  - the owner, as account holder, asks the retailer to close the account; the retailer's metering
    provider removes the meter;
  - a Level 2 accredited service provider, engaged by the owner or the builder, disconnects and removes
    the service line, and the DNSP is notified;
  - temporary builders power is arranged through the ASP, often from the same point of attachment.

  The new permanent service follows the T01 process.
- **Gas.** The gas network (Jemena in most of Sydney) removes the meter and caps the service at the
  main or at the property boundary. The request goes through the retailer. The house cannot be
  demolished while the meter is live.
- **Water.** The water service and meter are usually kept for dust suppression and builder's water,
  protected during demolition, then renewed or relocated for the new house through Sydney Water's Tap in
  service (verified: false for when Sydney Water requires a new meter).
- **Sewer.** A licensed plumber caps the existing sewer at the property connection point before
  demolition and records it. Any downpipe found draining to the sewer is disconnected.
- **Telecommunications.** The old copper and NBN lead-in are removed. The NBN new-development or
  re-connection application for the new house follows the T01 process.
- **Solar PV, if any.** A CEC-accredited electrician removes and isolates it (variant).
- **Before You Dig.** A search for the street assets near the crossing and the boundary.

The app's knowledge holds none of these as a work item, cost line or program row. The program has only
the single demolition activity "Isolate, demolish and clear" [K: knowledge/program/fragments.yaml
fr.demolition-1a].

**Hazardous materials.** [D]
- The house was built before 2004, so the WHS duty to identify asbestos before demolition applies. The
  app's threshold is the same year, held as an Implication only
  [K: knowledge/clusters/structure/consequences.yaml cq.pre-2004-fabric-hazardous-materials-survey].
- The base survey is a **demolition** hazardous materials survey of the whole house, garage and shed.
  Because nothing is kept, it uses destructive sampling: under floor coverings, inside wall cavities, the
  roof space and the meter board. The result is an asbestos register and a removal scope.
- Expected finds in a 1962 house:
  - bonded asbestos cement eaves, wet-area linings, fibro cladding to the garage and lean-to;
  - vinyl floor tiles and their backing;
  - meter board backing;
  - lead paint on joinery and in the ceiling dust;
  - synthetic mineral fibre insulation;
  - PCB capacitors in old fluorescent fittings.
- **Removal.**
  - A SafeWork NSW licensed removalist does it: Class B for more than 10 m² of bonded material, Class A
    for friable.
  - SafeWork NSW is notified before work starts.
  - A clearance inspection and certificate (a licensed asbestos assessor for friable work) come before
    the rest of the demolition proceeds.
  - Asbestos waste goes to a licensed facility with EPA waste tracking.
  - Thresholds, the notice period, the licence classes and the tracking threshold are all
    `verified: false` (WHS Regulation 2017 (NSW); Protection of the Environment Operations (Waste)
    Regulation 2014).
- **Lead.** Lead paint and ceiling dust are managed to AS 4361.2 (verified: false), mainly by keeping
  dust off the neighbours.
- **Loose-fill asbestos.** The NSW Loose-Fill Asbestos Insulation register is checked. A listed house is
  a separate, much larger job (trigger `loose-fill-asbestos`).
- The app holds the demolition stability and hazardous-material sequence rule
  `rule.as2601.demolition-stability` (clause_verified false) [K: knowledge/clusters/structure/rules.yaml].
  It holds no asbestos rule.

**Structure.** No existing structure is kept, so no existing-structure assessment is needed. [D]
- The structural engineer's existing-assessment stage is not needed. The app includes it once
  `det.existing_building` is true [K: knowledge/rfp/structural.yaml existing_assessment].
- The engineer's input to demolition is limited to two things:
  - advice on any part of the old structure that braces a neighbour's boundary wall or retaining wall;
  - checking that removing the old footings leaves the new founding level sound.
- AS 2601 demolition sequencing is the demolisher's job.

**Matching existing work.** Not applicable. Nothing is retained.

**Dilapidation of neighbours.** [D]
- On an established lot the neighbours' houses are often within 1–2 m of the boundary. Demolition
  vibration, dust and excavation all reach them.
- Base: a building consultant prepares a pre-demolition dilapidation report for each adjoining property
  (both sides and the rear): photos, crack mapping, fences and boundary walls. A copy goes to each
  neighbour, with access by agreement. A post-completion survey follows.
- It is not a statutory requirement for every CDC, but builders and owners do it on almost every
  knock-down. Some CDC or DA conditions require it where excavation is near a neighbour (verified: false
  for the Codes SEPP condition).
- The structural RFP prices dilapidation only with `excavation_adjoining` or `basement`. Otherwise it
  marks it "client (separate survey)", which is the base answer here [K: knowledge/rfp/structural.yaml
  dilapidation].
- Dividing fences are kept and protected, or replaced by agreement under the Dividing Fences Act 1991
  (verified: false).

**Trees on the established lot.** [D]
- The street tree is council's. Council's DCP or the CDC conditions require it to be protected, and
  council may take a tree bond.
- The retained rear-yard trees need tree protection fencing during demolition, which is when machinery
  does the most damage.
- A CDC cannot approve removing a tree that council's tree controls protect. That needs a separate
  council permit, or it makes the job a DA (verified: false for the exact rule).
- A tree that has to come out, or works inside a tree protection zone, is the trigger `trees-affected`.

**Stormwater of the existing site.** [D]
- The old house drains through an earthenware line to a kerb outlet in poor condition. The survey and a
  CCTV inspection or locating show whether the outlet can be reused, and where the sewer junction is.
- The new roof is larger and the hard surface grows, so the civil engineer designs a new system. This
  matches R1's reasoning for extensions.
- Council may require OSD for a new dwelling even where the old house had none (trigger `osd-required`).
- A rear-falling lot whose old house drained to a rubble pit has no legal point of discharge (trigger
  `stormwater-no-gravity`).
- In the app the civil RFP's `retained_services` position (CCTV, survey and potholing of the drainage)
  needs the `existing_building` module, which follows `det.existing_building`
  [K: knowledge/rfp/civil.yaml retained_services; knowledge/rfp/modules.yaml existing_building].

**Approval history.** An unapproved earlier addition or garage does not matter here, because it is
demolished. A structure built over the boundary or over an easement (an old garage on a sewer line) does
matter: the survey must show it. [D]

**How `det.existing_building` is keyed.** The determinant is "Work to an existing building"
[K: knowledge/determinants.yaml existing_building]. Demolition is work to an existing building, so the
base key sets it **true**. This turns on the RFP `existing_building` module, which is right for two
rows:

- the architect's `existing_interfaces` row (demolition plan, investigations);
- the civil `retained_services` row (existing drainage investigation).

It is wrong for one stage and one position: the structural `existing_assessment` stage and its
`strengthening` position. It also turns on many existing-building Implications meant for retained
fabric, for example `uc.asbestos-cement-eaves-at-roof-tie-in` on a new roof [K:
testing/synthetic/reference/index/unforeseen.md]. Its only trigger is the phrase "existing building", so
documents that say "existing dwelling" or "existing house" never ask it (§10).

---

## 4. Consultants and why

App suggestion for this profile: Architect, Certifier, Civil, Electrical, Geotechnical, Hydraulic,
Mechanical, Structural, Surveyor and Town Planner [K: knowledge/works/package_defaults.yaml baselines[0],
scope_additions[2]–[4]]. With CDC, Town Planner comes from `baselines[0]` alone (the DA rule
`complexity_additions[0]` does not fire). No rule reads `site.demolition`, so the demolition adds nothing.

| Discipline id | Engaged at | Need | Why | App |
|---|---|---|---|---|
| `consultant.architect` | Brief / concept | Always (or a building designer) | Design lead, demolition plan, CDC documents, construction documents, tender | yes, `baselines[0]` |
| `consultant.surveyor` | Before concept | Always | Detail and boundary survey with the existing house, floor and ridge levels, structures to be demolished, trees, pits, kerb outlet, sewer junction, crossing and neighbours' windows. Later set-out and identification surveys. | yes, `baselines[0]` |
| `consultant.hazmat` | Investigation, before the CDC and the demolition tender | Always (pre-2004 house) | Demolition hazardous materials survey and asbestos register; removal scope; clearance inspection and certificate after removal | **no** rule names it |
| `consultant.geotechnical` | Before structural design | Always | AS 2870 site class from boreholes clear of the house; advice on removing old footings and fill; option to confirm after demolition | yes, `baselines[0]` |
| `consultant.structural` | Design development, before the CDC | Always | Slab and footings, beams, framing review, design certificate for the CDC, inspections | yes, `baselines[0]` |
| `consultant.civil` | Approval documentation | Always | New stormwater to the kerb, assessment of the existing outlet line, driveway levels, sediment control. Often the same firm as structural. | yes, `baselines[0]` |
| `consultant.basix` | Approval documentation | Always (NSW) | NatHERS assessment and BASIX certificate for a new dwelling, lodged with the CDC | **no** rule names it |
| `consultant.certifier` | Before CDC lodgement | Always | Determines the CDC (demolition and new dwelling), principal certifier, pre-start inspection before the CDC is determined and critical-stage inspections (R7), occupation certificate | yes, `baselines[0]` |
| `consultant.building` | Before demolition | Usually | Pre-demolition and post-completion dilapidation reports on the adjoining properties | **no** rule names it |
| `consultant.quantity_surveyor` | Concept and pre-tender | Optional | Cost plan including demolition, asbestos and latent-condition allowances; lender drawdown reports | **no** rule names it |
| `consultant.arborist` | Investigation | Conditional (trigger) | Tree impact assessment and protection plan | **no** rule names it |
| `consultant.heritage` | Concept | Conditional (trigger) | Heritage impact statement for demolition in a conservation area | yes when `det.heritage_status` is set, `complexity_additions[1]` |
| `consultant.environmental` | Investigation / after demolition | Conditional (trigger) | Asbestos in soil, loose-fill asbestos | **no**: the contamination rule never fires [K: knowledge/works/package_defaults.yaml complexity_additions[2]] |
| `trade.main_works` | Procurement | Always | Head contract with a licensed builder, including licensed demolition and asbestos removal subcontractors | partial: proposed only from gaps on accepted items [K: internal/procurement/proposed.go ProposeHeadContract] |

**Not expected for this job, but the app suggests them** [D]:

- `consultant.town_planner`. A code-compliant CDC needs no statement of environmental effects or DA
  management. The app suggests it through `baselines[0]` on every new house, and its RFP then blocks on
  `cdc` [K: knowledge/rfp/town-planning.yaml unsupported_pathways, via
  testing/synthetic/reference/index/rfp-templates.md]. It returns under `da-pathway`.
- `consultant.electrical`, `consultant.hydraulic`, `consultant.mechanical`: as T01. House services are
  designed and installed by the trades.

**Not engaged on a house knock-down, though the ids exist:**

- `consultant.demolition`. A licensed demolisher under the builder plans a house demolition; a
  demolition engineer is for large structures.
- `consultant.waste`. A demolition waste management plan, where council asks for one, is a form the
  architect or builder fills in.
- `consultant.landscape`. A Housing Code CDC needs landscaped-area compliance on the site plan, not a
  landscape plan. A DA usually does need one (trigger `da-pathway`). [D]

---

## 5. Approvals and authorities

Clause numbers are from memory of the instruments, so they are all `verified: false`. [D]

### Planning pathway (base: CDC)

- **One CDC for demolition and the new dwelling.**
  - Demolition of a dwelling is complying development under the Demolition Code (Part 7), and the new
    two-storey dwelling under the Housing Code (Part 3), of SEPP (Exempt and Complying Development
    Codes) 2008. One certificate can cover both.
  - It is lodged through the NSW Planning Portal with the registered certifier, who determines it.
- **What the CDC needs.**
  - The certificate needs construction-level documents at lodgement: architectural plans including a
    demolition plan, the survey, structural and stormwater design, the BASIX certificate and the
    geotechnical report.
  - It also needs evidence that every Housing Code standard is met, and the hazardous materials survey,
    which the certifier and the Demolition Code conditions call for.
  - Many certifiers also ask for a demolition work plan to AS 2601 and a waste plan.
- **Eligibility checks before design is fixed.** A section 10.7 planning certificate, the lot size and
  frontage, heritage, bushfire (BAL-40 and FZ excluded), flood controls, easements and the tree permits.
- **Notification.** The certifier notifies neighbours of the CDC application before determining it,
  and the owner or builder notifies neighbours before work starts (EP&A Regulation 2021, verified: false
  for both periods and the distance).
- **Contributions.** Council's s7.11 or s7.12 contributions can still be payable under a CDC. A credit
  for the existing dwelling usually leaves little or nothing for a one-for-one replacement (verified:
  false).
- **Alternative: DA** (trigger `da-pathway`). Demolition and the new dwelling in one DA, then a
  construction certificate.

### Certificates

| Certificate | By | When |
|---|---|---|
| Hazardous materials survey and asbestos register | Hazardous materials consultant | Before CDC lodgement and before the demolition contract is priced |
| BASIX certificate | NatHERS/BASIX assessor | With the CDC (SEPP (Sustainable Buildings) 2022, verified: false) [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required, clause_verified false]. A knock-down rebuild is a new dwelling for BASIX, not an alteration (R5 applies to alterations only). |
| Complying development certificate | Registered certifier | Before any demolition or building work |
| Principal certifier appointment and notice of commencement | Owner appoints; notice to council and neighbours before work (verified: false for the periods) | Before demolition |
| HBCF insurance certificate | Builder | Before any deposit or work; contract over $20,000 including GST (R4) |
| Asbestos clearance certificate | Licensed asbestos assessor or competent person | After asbestos removal, before the rest of the demolition goes on |
| Demolition waste and asbestos disposal records | Demolisher | For the certifier and council, where the CDC conditions ask |
| Electrical (CCEW), plumbing and gas compliance certificates | Licensed trades | At disconnection where required, and at completion |
| Termite, waterproofing, glazing, truss and frame, engineers' and stormwater works-as-executed certificates | As T01 | For the occupation certificate |
| Occupation certificate | Principal certifier | After the final inspection and the BASIX completion check |

### Authority applications [D]

- **Certifier:** the CDC, then principal certifier services.
- **Council:**
  - registration of the CDC (the certifier does this);
  - s138 consent for the vehicular crossing and the redundant layback (verified: false);
  - a road or footpath occupancy permit for skip bins and the demolition plant;
  - the damage security deposit and any street-tree bond;
  - a tree permit if any regulated tree is removed;
  - contributions if levied.
- **Long Service Corporation:** the long service levy is paid before the CDC is issued, for work above
  the threshold (verified: false).
- **SafeWork NSW:** notification of licensed asbestos removal before it starts. No notification is
  needed for the house demolition itself (verified: false). The WHS duties of the principal contractor
  apply.
- **EPA:** asbestos waste tracking (verified: false for the threshold).
- **DNSP, through the retailer:** permanent disconnection and abolishment of the old service; a
  temporary builders supply; a new connection application.
- **Gas network (Jemena), through the retailer:** meter removal and service abolishment.
- **Sydney Water:**
  - Tap in building plan approval for the new house;
  - the meter and service changes;
  - sewer capping records.
- **Section 73 certificate:** not expected for one dwelling replacing one on a serviced lot, unless the
  consent or CDC conditions require it. The app's rule says it is needed "for most new developments"
  [K: knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate, clause_verified
  false].
- **NBN Co:** removal of the old lead-in, and the new lead-in application.
- **Fire and Rescue NSW:** none.

### Inspections and hold points

Critical-stage inspections by the principal certifier follow R7: a pre-start inspection, after
excavation and before the footings, before the slab pour, before covering the frame, before covering
wet-area waterproofing, before covering the stormwater connections, and the final inspection before the
occupation certificate. The clause stays `verified: false`. On a CDC job the pre-start inspection (R7
item 1) comes before the CDC is determined, so the certifier sees the existing house; demolition then
waits for the notice of commencement. [D]

Hold points that are not statutory but drive the job [D]:
1. All services disconnected and made safe, with written confirmation, before demolition.
2. The asbestos clearance certificate before the rest of the demolition goes on.
3. Dilapidation reports issued to the neighbours before demolition.
4. Tree protection installed before demolition.
5. Erosion and sediment control installed before demolition.

The app's own hold points: `if.frame-inspected-before-roof` fires, and so does
`if.reinforcement-inspected-before-pour`, from the `structure.concrete` tag on the `fr.slab-on-ground-1a`
pour activity (confirmed by a read-only Go run, 30 October 2026). None of the demolition hold
points exists. The only interface on `site.demolition` is `if.demolition-maintains-retained-structure`,
which is about a retained structure [K: knowledge/clusters/structure/interfaces.yaml].

---

## 6. Cost plan shape

Lines only, never amounts. The sections follow the app [K: knowledge/costs/starter_lines.yaml;
knowledge/costs/elements.yaml].

**Fees and charges**

| Line | App |
|---|---|
| CDC fee and certifier's CDC lodgement charges (Planning Portal and council registration fees) | **partial**: no CDC line. The app keeps P05 "Construction certificate and certifier fees" on CDC though no CC issues. |
| Development contributions (s7.11 or s7.12), if levied | **no**: the DA-fees starter, which holds contributions, is dropped for `cdc` [K: knowledge/costs/starter_lines.yaml da-fees] |
| Long service levy | no |
| Service authority connection **and disconnection** fees (DNSP abolishment and new connection, gas abolishment, Sydney Water, NBN) | partial: P06 is one generic line, with nothing for abolishment |
| Council road-reserve fees: s138 crossing, footpath and road occupancy for skips and plant, damage deposit, street-tree bond | no |
| Council tree permit | no (only with `trees-affected`) |
| HBCF insurance | no line. It is the builder's premium inside the contract sum, listed so a builder's exclusion is caught. |

**Consultants** (one fee line per package)

Architect; Surveyor; Hazardous materials consultant; Geotechnical engineer; Structural engineer; Civil
(stormwater) engineer; BASIX/NatHERS assessor; Certifier; Dilapidation surveys (building consultant);
Quantity surveyor (optional).

The app writes F-lines for its own suggestions only. It misses the hazardous materials, BASIX and
dilapidation lines, and it adds Electrical, Hydraulic, Mechanical and Town Planner
[K: testing/synthetic/reference/index/type-predictions.md T10]. The certifier is again counted twice, as
the F-line and as P05, as in T01.

**Construction** (by element; one head contract)

| Element | Holds | App |
|---|---|---|
| Preliminaries and builder's margin | Site establishment, supervision, insurances, HBCF premium, temporary services, fencing, scaffold | **no** line [K: knowledge/costs/budget_splits.yaml residential_house_new] |
| Demolition and hazardous materials removal | Licensed asbestos removal and clearance, lead controls, demolition of the house, garage and shed, removal of old slabs and footings, waste separation, tracking and disposal | **partial**: no line of its own. `site.demolition` rolls into 01 Site works. The `residential_house_new` budget split has no demolition share, so a stated budget under-allocates it. |
| Services disconnections and make-safe | Electrical, gas, sewer capping, telecom and solar removal before demolition | **no** |
| 01 Site works and external works | Earthworks, sediment control, stormwater, services, driveway and crossing with layback reinstatement, landscaping, fencing, tree protection | yes |
| 02 Substructure; 03 Structure; 04 Envelope; 05 Interiors; 08 Mechanical; 09 Hydraulic; 10 Electrical; 11 Communications and security | As T01 | yes |
| 13.x Stairs; 13.x Balustrades | As T01 | partial / partial (only when a document or the user adds the item; no `implied_by` for barriers) |
| Provisional sums and PC items | Selections | no |
| Provisional sum for latent ground conditions after demolition | Old footings, rubble, buried drainage, fill, unexpected asbestos | **no** [D] |

The app again makes **07 Active fire protection** from the smoke-alarm item, as T01. It is a known extra.

**Contingency** (app section): Design contingency; Construction contingency; Escalation; Exclusions
(all yes). Owner-supply lines appear only from `proj.owner_supply_stated`.

**Owner's own costs** that no contract holds: temporary accommodation, removal and storage for the build
period; insurance of the vacant house before possession. No app line. A PM keeps them visible on the
owner's budget. [D]

---

## 7. Program shape

**Skeleton:** `sk.new-traditional-cdc`: investigation → concept design → approval documentation →
approvals → procurement → construction → completion → defects [K: knowledge/program/skeletons.yaml].
The skeleton has no construction-documentation stage. As I read `resolveStage`, the construction
documentation fragment then falls back to the nearest earlier catalogue stage, approval documentation
[K: internal/program/generate.go resolveStage; knowledge/works/stages.yaml]. For a CDC that placement
is right by accident: the CDC needs CC-level documents before lodgement. (Read from the code, not a Go
run.)

**The order a PM actually runs** [D]:

1. **Investigation.**
   - Planning certificate and title search.
   - A CDC eligibility check against the Housing Code (lot, heritage, bushfire, flood, easements, trees).
   - The LFAI register search.
   - Sydney Water sewer diagram.
   - Before You Dig.
   - Detail survey with the existing house.
   - Demolition hazardous materials survey.
   - Geotechnical investigation clear of the house.
   - CCTV or locating of the existing drainage.
2. **Concept and design development**, as T01.
3. **Approval documentation.** The full CDC set: architectural plans with the demolition plan, NatHERS
   and BASIX, stormwater design, structural design, the demolition work and waste plans, the
   construction drawings and specification.
4. **Approvals.**
   - Sydney Water Tap in approval.
   - Pay the long service levy.
   - Lodge the CDC.
   - The certifier notifies neighbours.
   - Pre-start site inspection by the certifier (R7 item 1), with the existing house still standing.
   - CDC determination.
5. **Procurement.** Tender (it can run in parallel with the CDC), award, HBCF certificate.
6. **Pre-start.**
   - The owner vacates.
   - Disconnection and abolishment requests go in early; this is the long lead.
   - Dilapidation reports of the neighbours.
   - Principal certifier appointed and notice of commencement given to council and neighbours.
7. **Demolition.**
   - Site fencing and signage, sediment controls, tree protection.
   - Services confirmed disconnected.
   - Temporary power.
   - SafeWork notification, then licensed asbestos removal and clearance.
   - Demolish, remove old slabs, footings and buried drainage, separate and dispose of the waste.
   - A geotechnical check of the exposed platform if the boreholes could not reach the house footprint.
8. **Construction**, as T01, from set-out and earthworks to landscaping and utility connections.
   Earthworks follow demolition [K: knowledge/program/fragments.yaml fr.earthworks-1a attach after
   fr.demolition-1a].
9. **Completion.**
   - Practical completion.
   - Post-construction dilapidation survey.
   - Works-as-executed and certificates.
   - Final inspection and occupation certificate.
   - Handover.
10. **Defects.** The contract defects liability period (R8, length `verified: false`), then the statutory
    warranties.

**Logic that drives the job:**

- Survey, hazardous materials survey and geotechnical investigation come before design development. The
  hazardous materials survey also comes before the demolition is priced.
- The CDC needs BASIX, stormwater, structural and the full documents.
- Before demolition: pre-start inspection and CDC issued, contract, HBCF, notice of commencement,
  and all services abolished. Then asbestos removal → clearance → demolition → earthworks → T01 sequence.
- The DNSP abolishment and the gas meter removal are long-lead items that sit on the critical path to
  demolition.
- The new DNSP connection must be live before final testing and the occupation certificate.

**Long-lead items:** the old electricity service abolishment and the gas meter removal (both before
demolition); windows; joinery; frames and trusses; the new DNSP connection; Sydney Water. The app holds
lead-time rows only for windows, joinery, the utility supply and services connections. It has none for
abolishment [K: knowledge/program/fragments.yaml fr.windows-1a, fr.joinery-1a, fr.utility-1a,
fr.services-connections-1a].

**What the app does with demolition.**

- **With the correct action.** With `site.demolition` = `remove`, `fr.demolition-1a` adds one activity,
  "Isolate, demolish and clear" (`du.demolition-1a`, fixed working days), and earthworks follow it. A
  one-activity fragment takes the work item's title as its label, so the row may read "Demolition and
  hazardous building materials" [K: knowledge/program/fragments.yaml fr.demolition-1a; internal/program/generate.go].
- **With the action a document gives today.** `implied_by strip-out` gives `new`, so `fr.demolition-1a`
  is false and earthworks have nothing to follow.
- **The hazardous materials survey row.** It exists only inside `fr.existing-conditions`, gated to
  refurb or extend, so it never appears for a new build
  [K: knowledge/program/fragments.yaml fr.existing-conditions].
- **Missing rows.** Asbestos removal, clearance, SafeWork notification, the disconnections, the
  dilapidation surveys and the R7 pre-start inspection have no fragment.
- **Placement of the CDC rows.** `fr.cdc-application` is in approvals and links to nothing, as with
  T01's CC [K: knowledge/program/fragments.yaml fr.cdc-application].

---

## 8. RFP shape

With `det.ncc_class` 1a and 10a the `house` RFP module is on and `beyond_house` is off. With
`det.existing_building: true` the `existing_building` module is on [K: knowledge/rfp/modules.yaml]. The
shared returnables R1–R9 apply to every RFP. Each RFP in `shape.yaml` carries an `expect` block in the
answer-key format, and an `app_differs` list. The base sets the same "no" flags as T01, so unknowns do
not turn excluded rows into priced options.

### Architect (`architecture`)

- **Must carry:**
  - concept and schematic design on the survey of the existing site;
  - the existing-building interface: a demolition plan of every structure to go, the investigations
    needed (hazardous materials survey, drainage) and the trees to keep (`existing_interfaces`, on through
    the `existing_building` module);
  - CDC documentation to the Housing Code, with coordination of the survey, hazardous materials, BASIX,
    stormwater and structural inputs;
  - lodging the CDC with the certifier on the NSW Planning Portal (`lodgement`, with
    `flag.architect_lodges: yes`);
  - design development;
  - house tender and construction documents with kitchen, bathroom and joinery design and a selections
    schedule;
  - specification;
  - tender support;
  - certifier liaison.
- **Options:** construction-phase support; contract administration; FF&E; modifications.
- **Must not carry:** a DA planning response (`planning_response` is absent on CDC); BIM; tenant fit-out;
  signage; façade certification; regulated design (R3: the DBP Act does not reach a Class 1 house); D&C
  or post-novation stages; the hazardous materials survey itself (the hazmat consultant does it); the
  demolition engineering (the demolisher does it).
- **App note:** `required_facts: [budget]` blocks until a budget is stated [K:
  knowledge/rfp/architecture.yaml required_facts]. Bidding is priced as an option because it falls after
  the house boundary, as T01.

### Structural (`structural`)

- **Must carry:**
  - design basis with the site class and wind class;
  - AS 2870 footing and slab design, noting old footings and fill removed;
  - residential framing review;
  - construction documents and a design certificate for the CDC;
  - responses to the certifier's requests during CDC assessment (`planning_approval` included: the
    certifier assesses the engineering at lodgement);
  - footings affected by in-ground services (the existing sewer junction and drainage).
- **Option:** construction-phase inspections with an inspection schedule; advice on boundary walls or
  neighbours' structures affected by demolition.
- **Must not carry:** an existing-structure assessment or strengthening (nothing is kept); shoring,
  retention or dilapidation (unless `excavation-adjoining`; the dilapidation survey is a separate
  appointment); demolition drawings (the architect's); temporary works design (the demolisher's); a
  Revit model; regulated design; D&C stages.
- **App note:** `existing_assessment` is included and `strengthening` priced as an option once
  `det.existing_building` is true [K: knowledge/rfp/structural.yaml existing_assessment, strengthening].
  That is the main knock-on error of keying the demolished house as an existing building.

### Civil / stormwater (`civil`)

- **Must carry:**
  - investigation of the existing drainage and kerb outlet (CCTV or locating) and how the findings
    change the design (`retained_services`);
  - stormwater design for the CDC with the discharge point confirmed;
  - detailed design for the tender and for construction;
  - driveway long section and crossing levels;
  - erosion and sediment control plan, effective from the start of demolition;
  - construction-phase inspection of the drainage before covering (R7 item 6);
  - the works-as-executed review and certificate for the OC. `as_built` follows `construction_phase`,
    so both are included; the app prices both as options.
- **Must not carry:** public road works; MUSIC or NorBE; a flood report; OSD (unless `osd-required`);
  existing OSD feasibility; a 3D model; trunk drainage; regulated design; D&C stages.

### Geotechnical (`geotechnical`)

- **Must carry:**
  - an investigation plan with boreholes or test pits placed clear of the existing house;
  - field work and laboratory reactivity testing;
  - an AS 2870 site classification and footing advice, noting the uncertainty under the house footprint;
  - advice on removing old footings and fill.
- **Option:** a confirmation inspection or extra boreholes after demolition; pier inspections.
- **Must not carry:** contamination or waste classification (an environmental consultant, only under
  `asbestos-in-soil`); groundwater wells; acid sulfate soils; transfer.

### Surveyor (`land-surveying`)

- **Must carry:**
  - detail and contour survey on AHD showing the existing house, garage and shed with floor and ridge
    levels, trees with spread, pits, the kerb outlet, the sewer junction, the crossing and street
    features;
  - boundary identification with easements and any encroachment of the old structures or fences;
  - adjoining houses' windows and ridge heights for the Housing Code privacy and height standards;
  - buried drainage and pits;
  - Sydney Water sewer records.
- **Options:** building set-out; identification surveys at slab and frame; a final survey.
- **Must not carry:** subdivision or strata titling; an interior measure of the house to be demolished;
  movement monitoring (unless `excavation-adjoining`).

### Certifier (`bca-certification`)

- **Must carry:**
  - CDC assessment and determination for demolition and the new dwelling, with neighbour notification;
  - principal-certifier services with the pre-start site inspection before the CDC is determined (R7
    item 1) and every other R7 critical-stage inspection;
  - the occupation certificate with the BASIX completion check.
- **Must not carry:** the BCA advisory A package; design; building supervision; performance solutions;
  fire referrals.
- **App note:** the template lists `cdc` as an unsupported pathway, so the RFP blocks, and every
  certification stage B1–B4 needs a `da` or `ssd` pathway. A CDC house, whose approval *is* the
  certifier's, cannot get a certifier RFP [K: knowledge/rfp/bca-certification.yaml
  unsupported_pathways, boundary].

### Hazardous materials consultant (no template)

- **Must carry:**
  - a demolition hazardous materials survey of every structure on the lot, with destructive sampling;
  - an asbestos register and a removal scope by material, location and friable or bonded state;
  - lead paint, SMF and PCB identification;
  - a check of the LFAI register;
  - a clearance inspection and certificate after removal;
  - an optional visual soil clearance after demolition.
- **Must not carry:** the removal works; a soil contamination assessment (unless a trigger); air
  monitoring for bonded removal (friable only); a refurbishment survey limited to "areas affected".
- The app has no template, so a `consultant.hazmat` package gets the generic `tpl.rfp-capex`.

### BASIX / NatHERS assessor (no template)

As T01, but for a CDC: a NatHERS assessment with stamped plans and the BASIX certificate for the CDC
lodgement. It must not carry Section J, Green Star or NABERS.

### Building consultant: dilapidation (no template)

- **Must carry:** pre-demolition dilapidation reports for each adjoining property (photographs, crack
  mapping and measurements, fences and boundary walls), a copy to each owner, and a post-completion
  survey with comparison.
- **Must not carry:** structural design; vibration monitoring with trigger levels (unless a trigger);
  defect inspections of the new house.
- No template: generic `tpl.rfp-capex`.

### Quantity surveyor (`cost-planning`, optional)

As T01, plus separate allowances for demolition and asbestos removal, the disconnections and a latent
ground conditions provisional sum.

---

## 9. Triggers

Seventeen common conditions. Each changes procurement, cost plan, program or RFP. The machine-readable form
is in `shape.yaml`.

| Id | How a document shows it (directly and indirectly) | How a document denies it | What it adds |
|---|---|---|---|
| `da-pathway` | "A development application will be lodged for demolition of the existing dwelling and construction of a new two-storey dwelling"; "the second storey exceeds the Housing Code's side setback, so complying development is not available" | "The demolition and new dwelling will be approved by a complying development certificate" | Town planner (SEE covering the demolition); landscape plan; DA fees; DA lodgement, notification and assessment; a construction certificate. CDC rows removed. The app swaps to `sk.new-traditional-da`. |
| `builder-designs` | "Knock-down rebuild package with a project-home builder, demolition included"; "the builder's drafting team prepares the CDC drawings"; "site costs quote after the builder's site inspection" | "Architect-designed house, select tender to three builders" | No owner design team. The builder arranges survey, soil test, BASIX, hazardous materials survey and demolition within its contract. Owner appoints the certifier. Program: preliminary agreement, site inspection, builder's CDC set, new-home contract. The app sets novation `option`, adds `fr.novation`, and with D&C on CDC falls back to `sk.generic`: a gap. |
| `separate-demolition-contract` | "Stage 1: demolition by a separately engaged licensed demolisher; Stage 2: the new dwelling"; "the site will be handed to the builder clear and level"; "demolition only CDC" | "Demolition and asbestos removal are included in the builder's contract" | A separate demolition works package and contract (no demolition trade id exists); a demolition-only CDC only where demolition is approved apart from the dwelling (a separate demolisher can work under the same CDC); handover of a clear site; an owner-held provisional sum, because the latent-condition and asbestos risk moves to the owner. The app produces nothing extra: `fr.demolition-1a` already fires in the base. "Stage 1" also wakes the `staged` flag (§10). |
| `loose-fill-asbestos` | "The property is listed on the NSW Loose-Fill Asbestos Insulation Register"; "Mr Fluffy insulation in the roof space" | "A search of the LFAI register returned no listing" | Class A friable removal under enclosure with air monitoring; licensed asbestos assessor clearance; environmental soil assessment and remediation of the footprint; a NSW government and council process for listed homes (verified: false). |
| `friable-asbestos` | "Friable asbestos pipe lagging was identified in the subfloor"; "the hazardous materials survey found friable asbestos in the old heater flue" | "No friable asbestos was identified; all asbestos-containing materials are bonded" | Class A removal under enclosure with air monitoring; licensed asbestos assessor clearance; SafeWork NSW Class A notification (verified: false). No register listing and no soil remediation. |
| `asbestos-dividing-fence` | "The side and rear dividing fences are fibro (asbestos cement) sheeting"; "replace the existing fibro boundary fences with Colorbond by agreement with the neighbours" | "Dividing fences are timber paling and are retained" | Fences in the hazmat survey and removal scope; fencing notices and agreement with the neighbours (Dividing Fences Act 1991, verified: false); licensed removal along the boundary; the owner's share of replacement fencing. |
| `asbestos-in-soil` | "Asbestos cement fragments in the soil where the old fibro garage stood"; "fill in the rear yard contains asbestos fragments" | "No asbestos-containing material was observed in soil" | Environmental consultant; asbestos-in-soil assessment, removal and clearance; disposal; `site.contamination` investigate; civil earthworks-suitability input. |
| `trees-affected` | "The new driveway is within the TPZ of the street tree"; "remove the camphor laurel near the rear boundary"; "demolition machinery must pass the neighbour's fig" | "No trees are affected; all works are outside tree protection zones"; "tree removal: nil" | Arborist (AS 4970 impact and protection plan); council tree permit (a CDC cannot approve it); tree bond; protection hold points. |
| `heritage-conservation-area` | "The site is within a heritage conservation area; demolition of the dwelling requires consent"; "a contributory building" | "Not a heritage item and not in a conservation area" | DA required (CDC unavailable); heritage consultant (heritage impact statement justifying demolition); town planner; heritage referral; complex architecture and structural scale. |
| `stormwater-no-gravity` | "The existing house drains to a rubble pit in the rear yard"; "the lot falls to the rear; an interallotment drainage easement over No. 12 is needed"; "no legal point of discharge" | "Stormwater discharges by gravity to the kerb through the existing outlet, to be replaced" | Easement negotiation and registration, or a pump-out or absorption system; civil design; legal and survey costs. |
| `osd-required` | "OSD is required because the impervious area increases"; "council's stormwater policy requires on-site detention for new dwellings" | "No OSD is required for a replacement dwelling of similar impervious area" | OSD design (civil), tank, positive covenant and restriction on title. |
| `sewer-main-in-lot` | "A Sydney Water sewer main crosses the rear of the lot under the old garage"; "build over or next to a Sydney Water asset" | "No Sydney Water assets are within the property" | Sydney Water building-over or adjacent-asset approval; piers below the zone of influence; possible CCTV of the main before and after; structural authority-asset stage. |
| `excavation-adjoining` | "Excavation for the garage within 900 mm of the neighbour's house"; "rock breaking near the boundary"; "neighbour's footings within the zone of influence" | "Slab on ground at natural level; no excavation near the boundaries" | Structural retention and dilapidation option; civil shoring; vibration monitoring; extra provisional sums. |
| `bushfire-prone` | "Mapped bush fire prone land, BAL-29"; "the lot backs onto bushland" | "Not bush fire prone land" | Bushfire consultant; AS 3959 construction; CDC unavailable at BAL-40 or FZ (verified: false). |
| `flood-affected` | "Subject to flood related development controls"; "habitable floor 500 mm above the 1% AEP" | "Not subject to flood related development controls" | Flood advice and report; floor levels; CDC conditions or DA; flood-compatible materials. |
| `existing-pool` | "Demolish the existing in-ground pool and backfill"; "the old fibreglass pool in the back yard will be removed" | "There is no pool on the site" | Pool demolition and compacted, tested backfill (geotechnical inspection if under the new footprint); removal from the pool register (verified: false); barrier maintained until removal. |
| `old-septic-tank` | "Disused septic tank in the rear yard from before the sewer connection"; "the house was connected to sewer in 1985; the old tank was never removed" | "No septic tank or on-site system on the lot" | Pump out, remove or fill the old tank; council approval under the Local Government Act s68 (verified: false); provisional sum. |

Less common variants left out of the key:

- existing rooftop solar PV;
- a battle-axe lot with constrained access for demolition plant;
- a party wall (semi-detached knock-down: a different job);
- termite infestation of the old house (it adds a pre-construction soil treatment);
- an existing secondary dwelling that is kept;
- a sloping lot (as T01, `sloping-site`), including existing boundary retaining walls.

**Zero-delta triggers.** `stormwater-no-gravity`, `existing-pool`, `old-septic-tank`, `friable-asbestos`
and `asbestos-dividing-fence` change nothing
the app produces today: no settable key, and no generator reads one. The harness should expect no
change from them.

---

## 10. Keyword traps and negations specific to this type

1. **"Demolition" gives the wrong action.** `implied_by strip-out` adds `site.demolition` with the
   work-type default `new`, so the one demolition fragment stays false. Only a Jev reading of
   `sys.site.demolition.action` = `remove` gives the right item. "No demolition of the street wall" or
   "the demolition of the old garage was completed in 2019" still adds it, because there is no negation
   check [K: knowledge/profile/scope_defaults.yaml implied_by strip-out; internal/profile/questions.go
   EvidenceQuestions].
2. **"Existing dwelling" is not "existing building".** The `existing_building` determinant's only
   trigger is `\bexisting building\b` [K: knowledge/determinants.yaml existing_building]. Knock-down
   documents say "existing dwelling", "existing house" or "existing residence", so the determinant may
   never be asked, and the RFP `existing_building` module stays off.
3. **"Built in the 1960s" is not a year.** `existing_building_year` triggers need "year built", "year of
   construction" or "built in 1962". "Circa 1960s", "post-war fibro cottage" or "c. 1962" are missed, so
   `cq.pre-2004-fabric-hazardous-materials-survey` cannot fire [K: knowledge/determinants.yaml
   existing_building_year].
4. **"Replace the house" reads as `replace`.** A sentence such as "replace the old house with a new
   two-storey home" can make Jev answer `replace` for systems such as roof coverings or windows.
   `replace` still fires the 1a fragments, but it tells the works model the system exists, which brings
   existing-building Implications. The expected action is `new`.
5. **"Asbestos contamination" is not contaminated land.** The `potentially_contaminated_land` trigger is
   `contaminat`. A hazmat report saying "asbestos contamination of the ceiling space" may set the land as
   contaminated. Building materials are not land contamination unless the soil is affected.
6. **Abolishing gas is not a gas connection.** "Abolish the gas service and remove the meter" mentions
   gas. The new house is all-electric: `det.gas_supply` stays `none` and there is no `hydraulic.gas` new
   item.
7. **"Stage 1 demolition" sets `staged`.** The `staged` flag fires on `\bstages? [1-9a-c]\b`, which turns
   on the RFP `staged` module [K: knowledge/profile/flag_readings.yaml staged; knowledge/rfp/modules.yaml
   staged]. A demolition-then-build sequence on one house is not staged delivery.
8. **"Demolish … tree" asks the arborist question.** `arborist_tree_impacts` matches "demoli" near
   "tree", for example "demolish the shed beside the street tree; the tree is retained and protected".
   The answer is no unless the tree is affected [K: knowledge/profile/flag_readings.yaml
   arborist_tree_impacts].
9. **"Remove the trees" versus "remove the shed near the trees."** `arborist_pruning` matches "remov"
   near "tree".
10. **"Road reserve"** for a skip bin or the layback reinstatement matches `public_road`. Those are not
    public road works.
11. **"Sewer main"** in "cap the sewer at the connection to the main" matches `authority_asset`. Only a
    main inside the lot matters.
12. **"Excavat… boundary"** in "remove old footings along the boundary" matches `excavation_adjoining`.
    Removing old shallow footings is not excavation next to a neighbour.
13. **"Existing OSD"** is rare on a 1960s lot. "No existing on-site detention" matches `osd_existing`.
14. **"Party wall"** in a dilapidation report ("no party walls; neighbour's wall 900 mm from the
    boundary") adds compartments and acoustic separation through `implied_by party-wall`.
15. **"Pool" means the existing one.** "The existing pool will be removed" is not a new pool. "Pool
    barrier to remain during works" is not new pool scope.
16. **"Class 1 dwelling"** is not read as a class (as T01). Set `det.ncc_class`.
17. **"Clearance".** An "asbestos clearance certificate" is not tree or canopy clearance.
    `arborist_tree_impacts` matches "clearance" near "tree" or "root".
18. **"Lodge".** The certifier "lodges" the CDC registration with council. `architect_lodges` asks on any
    "lodge". Who submits the CDC application is the question.
19. **"Brick veneer" does not add masonry.** `implied_by masonry` needs "masonry", "brickwork" or
    "repointing" [K: knowledge/profile/scope_defaults.yaml implied_by masonry], so a specification saying
    "brick veneer" leaves `structure.masonry` out. Scenario documents should say "face brickwork veneer to
    the ground floor". The validator does not accept a `scope.*` scenario input, so this is a gap.

---

## 11. Gaps in the built-in knowledge

What is missing, and what the app will probably produce wrongly as a result. The `gaps` list in
`shape.yaml` cites the evidence for each.

1. **Demolition is invisible to procurement.** No rule reads `site.demolition`. No hazardous materials,
   dilapidation or demolition consultant is ever suggested, and there is no demolition trade id for a
   separate demolition contract [K: knowledge/works/package_defaults.yaml; data/intake/disciplines.json].
2. **The demolition item gets the wrong action.** `implied_by strip-out` adds `site.demolition` as `new`,
   so `fr.demolition-1a` never fires unless Jev reads `remove`. Then earthworks have nothing to follow.
3. **No demolition-phase program content.** One activity covers all of these:
   - the services disconnections and abolishment;
   - temporary power (not in scope);
   - the SafeWork notification;
   - licensed asbestos removal and clearance;
   - waste tracking;
   - dilapidation surveys;
   - the R7 pre-start inspection;
   - owner move-out.

   There is no demolition hold point (services off, clearance, dilapidation, tree protection).
4. **The hazmat survey activity is gated to refurb and extend** (`fr.existing-conditions`), so a new build
   on a demolished lot never gets it.
5. **No work type for a knock-down rebuild.** The job is `new` with a `remove` item. `det.existing_building`
   true is the closest key, but it turns on rows meant for kept buildings: the structural existing
   assessment and strengthening, and Implications such as asbestos eaves at a roof tie-in. False loses
   the architect's demolition interface and the civil drainage investigation.
6. **The existing-building trigger misses "existing dwelling"** and the year trigger misses "1960s". The
   asbestos consequence then cannot fire.
7. **`uc.asbestos-found-in-existing-building` excludes `remove`.** It fires on `site.demolition` with
   `new`, `alter`, `upgrade`, `repair` or `replace`, so the correct demolition action suppresses it
   [K: testing/synthetic/reference/index/unforeseen.md].
8. **CDC is badly supported:**
   - the certifier RFP blocks on `cdc` and has no CDC stages;
   - the town planner is still suggested and its RFP blocks;
   - the cost plan has no CDC fee or contributions line and keeps the CC-fees line;
   - the CDC lodgement links to nothing.
9. **Consultants a NSW knock-down always needs are never suggested:** BASIX, hazardous materials,
   dilapidation (building consultant). The arborist is never suggested, even under the trigger.
10. **Services engineers are over-suggested** (as T01), and Hydraulic is reinforced by the `hydraulic.gas`
    remove item through `scope_additions[3]`.
11. **Cost plan:**
    - no demolition and asbestos line, no disconnection line and no latent-condition provisional sum;
    - no preliminaries line, no long service levy and no road-reserve fees;
    - the house budget split has no demolition share;
    - the certifier is counted twice;
    - smoke alarms make an Active fire protection element.
12. **Retained trees get no program row.** `fr.tree-protection` needs `new`, `alter` or `repair`; a
    `retain` tree item brings nothing.
13. **No rule for asbestos** (identification, licensed removal, clearance) and no rule for demolition
    notification. Only `rule.as2601.demolition-stability` exists, with its clause unverified.
14. **Site conditions found under the old house** (an old septic tank, an existing pool, buried drainage,
    rubble pits) have no key, no work item that fits, and no fragment.
15. **D&C on CDC falls back to `sk.generic`.** The project-home knock-down, the most common version by
    volume, gets no skeleton.
16. **"Stage 1 demolition" turns on the `staged` module**, and many other regex flags read knock-down
    wording wrongly (§10).
17. **The Section 73 rule overstates** the requirement for one-for-one replacement.
18. **Carried from T01:**
    - the default scope lacks stairs, barriers, brick veneer, steel beams, the rainwater tank, the
      crossing, temporary power, appliances and ground investigation;
    - the investigation fragments need `investigate` actions;
    - `fr.ac-1a` follows the linings;
    - house RFPs price commercial options;
    - client-decision flags cannot be keyed;
    - there is no calendar.

---

## 12. Sources and unverified claims

**Opened in the repo:**

- `knowledge/systems.yaml`;
- `knowledge/clusters/structure/systems.yaml` (`site.demolition`, `site.adjoining-structures`);
- `knowledge/clusters/structure/consequences.yaml`;
- `knowledge/clusters/structure/rules.yaml` (`rule.as2601.demolition-stability`);
- `knowledge/clusters/structure/interfaces.yaml`;
- `knowledge/profile/scope_defaults.yaml` (`classes[0]`, `implied_by`);
- `knowledge/profile/flag_readings.yaml`;
- `knowledge/profile/taxonomy.yaml` (conditions);
- `knowledge/determinants.yaml` (`existing_building`, `existing_building_year`,
  `potentially_contaminated_land`, `gas_supply`);
- `knowledge/works/package_defaults.yaml`;
- `knowledge/works/actions.yaml`;
- `knowledge/works/stages.yaml`;
- `knowledge/costs/starter_lines.yaml`, `elements.yaml`, `budget_splits.yaml`;
- `knowledge/program/fragments.yaml`, `skeletons.yaml`, `durations.yaml` (`du.demolition-1a`,
  `du.hazmat-survey`);
- `knowledge/rfp/modules.yaml`, `architecture.yaml`, `structural.yaml`, `civil.yaml`, `arborist.yaml`,
  `bca-certification.yaml`, `land-surveying.yaml`, `geotechnical.yaml`;
- `internal/rfp/drivers.go` (`readFlags`, `deriveFlags`);
- `internal/program/generate.go` (`resolveStage`, `investigationRows`);
- `internal/profile/questions.go`, `proposals.go`;
- `data/intake/disciplines.json`;
- the shared index under `testing/synthetic/reference/`;
- the T01 and T05 maps;
- `reference/OWNER_RULINGS.md`.

**Instruments named from professional knowledge, every clause `verified: false`:**

- EP&A Act 1979 and EP&A Regulation 2021 (CDC neighbour notice, notice of commencement), with the
  critical-stage inspections per R7;
- SEPP (Exempt and Complying Development Codes) 2008, Part 3 Housing Code and Part 7 Demolition Code;
- SEPP (Sustainable Buildings) 2022;
- Work Health and Safety Regulation 2017 (NSW): asbestos identification before demolition, licensed
  removal classes, notification and clearance;
- Protection of the Environment Operations (Waste) Regulation 2014 (asbestos waste tracking);
- AS 2601 (demolition), AS 4361.2 (lead paint), AS 4970 (trees), AS 2870, AS 3959;
- Home Building Act 1989 (HBCF per R4);
- Building and Construction Industry Long Service Payments Act 1986;
- Roads Act 1993 s138;
- Local Government Act 1993 s68;
- Dividing Fences Act 1991.

**Claims I am least sure of:**

- CDC with traditional procurement as the base: accepted at review (T10 REVIEW §2.1).
- One CDC covering demolition and the new dwelling: accepted at review; part numbers `verified: false`.
- The electricity disconnection is one neutral sequence for every DNSP (T10 REVIEW §2.3),
  `verified: false`. Whether Sydney Water requires a new meter is still open.
- That SafeWork NSW needs no notification for the house demolition itself, only for licensed asbestos
  removal.
- That contributions are usually nil or small for a one-for-one replacement.
- Keying `det.existing_building: true` for a demolished house: accepted at review (T10 REVIEW §2.4).
