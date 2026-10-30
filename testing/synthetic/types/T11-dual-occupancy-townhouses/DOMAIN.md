# T11 Dual occupancy or townhouses: domain map

Status: draft for owner review (`reviewed: false`). State: NSW. Written 30 October 2026 from the
knowledge on `main` (e1a5389 plus the working tree) and professional practice. The form follows the
reviewed T01 map. Content that is the same as a single house is restated briefly, so this map stands on its own.

Tags: `[K: path anchor]` means SiteWise's knowledge already says it and I opened the file to check.
`[D]` means professional domain knowledge that is not in the repo. Instrument clause numbers I could not
check against the instrument are written `verified: false`. No prices, rates or week durations appear.
Owner rulings R1–R8 (`testing/synthetic/reference/OWNER_RULINGS.md`) are applied, in particular R4 (HBCF),
R6 (livable housing optional), R7 (critical-stage inspections) and R8 (defects liability period).

---

## 1. The job

**What it is.** An attached dual occupancy: two two-storey Class 1a dwellings side by side under one
roof form, divided by a fire-resisting and sound-insulating separating wall, each with an attached
Class 10a garage. It replaces an existing house on one suburban lot in Greater Sydney. The same DA approves
demolition of that house and Torrens title subdivision into two lots. In NSW this is the commonest small multi-dwelling job. "Townhouses" (three or
more attached Class 1a dwellings, multi dwelling housing) is the larger variant, trigger
`three-plus-townhouses`. The app's subclass for both is `townhouses`, labelled "Townhouses (Class 1a)".
[K: knowledge/profile/taxonomy.yaml townhouses]

| Item | Base value |
|---|---|
| Lot | One serviced lot of about 650–900 m², frontage about 15–20 m, R2 Low Density Residential zone, reticulated water and sewer, kerb and gutter, NBN available. It holds an existing single-storey house and garage (built 1958, so asbestos is likely), to be demolished |
| Buildings | Two attached dwellings of about 200–250 m² each: 4 bedrooms, 2–3 bathrooms, a garage each; one building on the lot |
| Construction | Slabs on ground (AS 2870), timber frames and trusses, brick veneer and cladding, a separating wall system with an FRL carried to the underside of the roof covering (masonry, or a proprietary lightweight fire-and-acoustic wall), metal or tiled roof |
| Services | Separate water service and meter, sewer junction, electricity connection and NBN lead-in for each lot; a BASIX rainwater tank each; on-site detention (OSD) required by council; all-electric |
| Pathway | One DA for demolition, the dual occupancy and the Torrens subdivision; CC from a registered certifier; Section 73 compliance certificate; subdivision certificate; plan registered at NSW Land Registry Services (LRS) |
| Procurement | Traditional: the owner-developer engages the design team, select tender to three or four residential builders, one HIA lump-sum contract for both dwellings |
| Value | Inside the PROJECT_TYPES range of $1–5M; the shape names no amount |

**Why demolition is in the base.** In middle-ring Sydney R2 streets a dual occupancy nearly always
replaces a 1950s–70s house (review F1). Demolition is handled as in T10: `det.existing_building: true`,
`site.demolition` with action `remove`, a hazmat consultant, and asbestos removal and clearance before
demolition. Three things differ from T10:

- the Housing and Productivity Contribution credits the existing lot;
- an existing sewer junction can serve only a retained home, so both new lots get new connections
  (Sydney Water dual occupancy guide §4.7);
- two crossings replace the old one.

A vacant lot is the `vacant-lot` trigger.

**Who procures it and how.** [D] Usually an owner-developer: a family that keeps one dwelling and sells
the other, or a small developer who sells both, often off the plan. The architect or building designer
is engaged first. The surveyor, hazmat consultant, geotechnical engineer, structural and civil engineer
(often one firm), the BASIX assessor, a landscape designer and usually a town planner follow. The owner appoints the
certifier. After the DA a Water Servicing Coordinator (WSC) runs the Section 73 application, and near
the end a solicitor prepares the 88B instrument. The builder holds one lump-sum residential contract
under the Home Building Act and takes out HBCF cover before any deposit or work (R4), one certificate
per dwelling (verified: false).

**What makes it go wrong.** [D]

1. **The lot does not meet the controls.** The minimum lot size or frontage for a dual occupancy, or the
   minimum lot size for the subdivision, is missed or misread. That leads to a clause 4.6 request or a
   refusal (trigger `cl46-lot-size`).
2. **Stormwater.** OSD sizing and the discharge point. A rear-falling lot needs an interallotment
   easement over a downhill neighbour, often as a deferred commencement condition (trigger
   `stormwater-easement`).
3. **Sydney Water.** The Notice of Requirements arrives late or asks for a main extension (trigger
   `sewer-extension-required`). Without the Section 73 certificate there is no subdivision certificate,
   and the developer cannot settle the sale.
4. **The separating wall.** These are leading defects, and costly to fix once the roof is on:
   - the wall stops short of the underside of a non-combustible roof covering, or of 450 mm above a
     combustible one;
   - combustible framing, fascia or eave members cross it (only roof battens of 75 × 50 mm or smaller and
     sarking may);
   - a shared eave open to the roof space is not lined;
   - services penetrate it;
   - there is no discontinuous construction where a bathroom, laundry or kitchen backs onto a habitable
     room (other than a kitchen) next door.

   (Housing Provisions 9.3.1 and 10.7.1; part numbers confirmed by review Q3, clause values verified: false.) [K:
   knowledge/clusters/fire/interfaces.yaml if.party-wall-roof-continuity;
   knowledge/clusters/fire/failure_modes.yaml fm.party-wall-stops-in-roof;
   knowledge/clusters/services-wet-air/failure_modes.yaml fm.bath-against-party-wall-without-isolation]
5. **The boundary.** The party wall must sit exactly on the future Torrens boundary. A set-out error
   shows up only on the final plan of subdivision.
6. **Titles late.** Subdivision registration slips after the OC, which delays the sale and the
   refinance. [K: knowledge/clusters/delivery/unforeseen.yaml uc.strata-or-subdivision-registration-delayed]
7. **The rest are as for a house:** budget overrun at tender, late site information, builder insolvency
   and HBCF, wet-area and flashing defects, and long-lead windows, joinery and DNSP connections (two
   connections here).

---

## 2. Systems and work items

Every top-level system in `knowledge/systems.yaml`. The app's townhouses/new default list has 45 systems,
all with action `new`. It is the house list plus `interiors.acoustic-separation`,
`fire-passive.rated-elements`, `access-egress.stairs-and-ramps` and `access-egress.barriers`. [K:
knowledge/profile/scope_defaults.yaml classes[1]] Items marked **+** are ones a PM expects that the
default list lacks.

### `site`

| System | Action | Why |
|---|---|---|
| `site.survey-boundaries` | new | Set-out of the separating wall on the proposed boundary, identification surveys, draft and final plans of subdivision. The system's own description includes "plans of subdivision or strata". [K: knowledge/clusters/structure/systems.yaml site.survey-boundaries] Default. |
| `site.ground-conditions` **+** | investigate | Geotechnical investigation; not in the default list, so `fr.investigate-geotech` never fires. |
| `site.earthworks` | new | Slab platforms. Default. |
| `site.erosion-sediment-control` | new | Standard DA condition. Default. |
| `site.stormwater-detention` | new | Drainage for both lots and the OSD tank or tanks (OSD per lot, or shared with easements and a positive covenant). Default. |
| `site.services-connections` | new | Separate water, sewer, electricity and NBN for each lot, as the subdivision requires. Default. |
| `site.vehicle-access` | new | Two driveways and paths. Default (no fragment). |
| `site.public-domain` **+** | new | Two vehicular crossings and footpath works under a Roads Act s138 consent (verified: false). |
| `site.landscape` | new | Landscape plan for each lot, front fences, and the new dividing fence on the lot boundary. Default. |
| `site.demolition` **+** | remove | Demolish the house, garage, slabs, paths and old driveway, after the hazmat survey, licensed asbestos removal and the clearance certificate. Not in the default list. `implied_by strip-out` adds it with action `new`, so `fr.demolition-1a` fires only on a `remove` reading. [K: knowledge/program/fragments.yaml fr.demolition-1a; knowledge/profile/scope_defaults.yaml implied_by strip-out] |
| `site.trees`, `site.retaining-walls`, `site.pools`, `site.flooding`, `site.contamination`, `site.acid-sulfate-soils` | out | Triggers or conditions. |
| `site.adjoining-structures` | out | The party wall is new work between two new dwellings, not a neighbour's structure. The app adds this item whenever a document says "party wall" (keyword trap 1). |
| `site.waste-storage` | out in the base | Bin areas only; it becomes real for three or more townhouses. |
| `site.impact-protection`, `.loading-docks`, `.signage` | out | Commercial systems. |

### `substructure`

`footings` (new, including the thickened footing under the separating wall), `slab-on-ground` (new, usually
one slab per dwelling with a break at the separating wall) and `termite-management` (new, AS 3660.1) are all
in the default list. `piles` only under `sewer-main-in-lot` or a Class P site. `basement-retention`,
`retaining-walls`, `damp-waterproofing` and `underpinning` are out. [K: knowledge/clusters/structure/rules.yaml
rule.as2870.footing-design-by-site-class]

### `structure`

`structure.timber` new (default). `structure.masonry` **+** new: brick veneer, and a masonry separating wall
where one is chosen; it arrives only from `implied_by masonry`. [K: knowledge/profile/scope_defaults.yaml
implied_by masonry] `structure.steel` **+** new: beams over garage doors and openings. `concrete`,
`precast`, `alterations` and `temporary-works` are out. `design-basis` is a consultant input.

### `envelope`

As for a house, and all in the default list: `external-walls`, `roof-coverings`, `windows-doors`,
`vehicle-doors` (two garage doors), `flashings-seals`, `pliable-membranes`, `thermal-insulation` and
`thermal-performance` (BASIX). At the party wall the separating wall runs to the underside of a
non-combustible roof covering, or 450 mm above a combustible one. Only roof battens of 75 × 50 mm or
smaller and sarking may cross it. Eaves and verandahs shared by both dwellings and open to the roof space
need a non-combustible vertical lining (Housing Provisions 9.3.1, verified: false).
`facade-acoustics` only under `noise-affected-site`. `bushfire-construction` only under `bushfire-prone`.
`external-waterproofing` only with a balcony over a habitable room. `curtain-wall`, `external-shading` and
`building-sealing` are out.

### `interiors`

All the house items (linings, ceilings, wet-area waterproofing to AS 3740, tiling, floors, joinery for two
kitchens, doors, internal glazing, painting) are in the default list. Plus:

| System | Action | Why |
|---|---|---|
| `interiors.acoustic-separation` | new | Separating wall sound insulation between the two dwellings: Rw + Ctr of 50 or more, to the underside of the roof or a rated ceiling. Discontinuous construction where a bathroom, sanitary compartment, laundry or kitchen in one dwelling adjoins a habitable room other than a kitchen in the other (ABCB Housing Provisions 10.7.1, verified: false). Default. [K: knowledge/clusters/envelope/systems.yaml interiors.acoustic-separation] |
| `interiors.ffe` **+** | new | Appliances as PC items. |
| `interiors.room-acoustics` | out | Not used on a dwelling. |

### `fire-passive`

| System | Action | Why |
|---|---|---|
| `fire-passive.rated-elements` | new | The separating wall as a fire-resisting element with an FRL (ABCB Housing Provisions Part 9.3, verified: false). It is in the default list. [K: knowledge/clusters/fire/systems.yaml fire-passive.rated-elements] |
| `fire-passive.compartments` **+** | new | Separation between the attached dwellings from the footings or slab to the underside of a non-combustible roof covering (or 450 mm above a combustible one). Only small battens and sarking may cross it, and shared eaves are lined (Housing Provisions 9.3.1, verified: false). The knowledge puts "fire-resisting party walls between attached Class 1 dwellings including their continuity to the roof" here. [K: knowledge/clusters/fire/systems.yaml fire-passive.compartments] It is not in the default list and arrives only from `implied_by party-wall`. |
| `fire-passive.penetrations` | out as a base item | Good design keeps services out of the separating wall. If a pipe or cable must cross it, fire stopping is needed. |
| `fire-passive.external-walls` | out | Side setbacks are usually 900 mm or more. An external wall within 900 mm of a boundary (Housing Provisions Part 9.2, verified: false) is a design variant. The new party-wall boundary is handled by the separating wall. |
| Others | out | No Type of Construction, no fire-hazard properties regime for Class 1a. |

### `fire-active`

`smoke-alarms` new, in each dwelling (default). Nothing else: a Class 1a dwelling has no fire safety
schedule, hydrants or sprinklers, even in a row of townhouses. [D]

### `mechanical`, `hydraulic`, `electrical`, `comms-security`

As for a house, doubled:

- `mechanical.air-conditioning` and `.local-exhaust` new (default).
- `hydraulic.cold-water`, `.hot-water`, `.sanitary` and `.roof-drainage` new (default). Each lot is
  separately metered and drained, and roof drainage stays separate either side of the party wall.
- `hydraulic.non-potable-water` **+** new: a BASIX rainwater tank for each dwelling.
- `hydraulic.gas`: out (all-electric).
- `electrical.utility-supply` new: two DNSP connections. `switchboards` (one per dwelling), `cabling-wiring`
  and `lighting` new (default).
- `electrical.temporary-power` **+** new. `solar-pv` is an owner choice, out of the base.
- `comms-security.carrier-lead-in` (two lead-ins) and `.structured-cabling` new (default). Security items out.

### `vertical-transport`

Out.

### `access-egress`

`stairs-and-ramps` (an internal stair in each dwelling) and `barriers` (stair, void, balcony and window
fall protection) are new, and both are in the townhouses default list. `livable-housing` is out: optional
in NSW (R6). Accessible-access, accessible-facilities, exits, egress doors and maintenance access are out.
`pool-barriers` only with a pool.

---

## 3. Existing-building interfaces

The existing house is demolished, not kept, so there is no matching of existing work and no structural
tie-in. Each interface below ends when the lot is clear. This follows T10, adjusted for two lots. [D]

- **Occupation.** The owner or tenants move out before demolition. The vacant house is secured, and the
  owner's insurance and the builder's contract works cover must meet with no gap.
  `hdr.cond.operational_constraints: vacant` keeps the RFP `live_operations` flag off. [K:
  internal/rfp/drivers.go readFlags]
- **Hazardous materials.** A 1958 house almost certainly has bonded asbestos (eaves, wet-area sheets,
  fibro walls, vinyl, fences) and may have lead paint. The sequence:
  1. check the loose-fill asbestos (LFAI) register;
  2. a demolition hazmat survey with an asbestos register;
  3. the SafeWork NSW notification;
  4. licensed removal;
  5. a clearance certificate;
  6. then demolition.

  The hazmat survey row in the app exists only in `fr.existing-conditions`, which is gated to refurb or
  extend. [K: knowledge/program/fragments.yaml fr.existing-conditions]
- **Services disconnections.** The retailer removes the electricity meter and a Level 2 ASP abolishes
  the service. The plumber caps the sewer junction. The NBN or telecom lead-in is removed. The water
  meter stays only if it will serve a new lot. An existing junction may serve only a retained home, so
  each new lot gets its own Section 73-approved connection (Sydney Water dual occupancy guide §4.7).
  Temporary power follows the abolishment.
- **Structure and ground.** Old footings, slabs, buried drainage, fill and rubble are removed. The site
  class under the old house is uncertain until the lot is clear. A provisional sum for latent ground
  conditions is usual.
- **Existing drainage and crossing.** CCTV or locate the old outlet line and kerb outlet, which may be
  reused for one lot. The old driveway crossing is removed and two new crossings are built.
- **Dilapidation.** Only where excavation is near a boundary or the neighbours' structures are close.
  Not in the base.

Interfaces that are not existing-building work: [D]

- **The party wall between the two new dwellings.** After subdivision it is shared by two owners, so the
  88B instrument creates easements for support, maintenance and access. A party wall that ends up off the
  boundary is a title problem.
- **Council assets.** Kerb, footpath and street trees: a damage deposit and a condition record.
- **Neighbours.** Dividing fences, overlooking and overshadowing during notification, and a downhill
  owner where a drainage easement is needed.

---

## 4. Consultants and why

App suggestion for this profile: Architect, Certifier, Civil, Electrical, Geotechnical, Hydraulic,
Mechanical, Structural, Surveyor, Town Planner. [K: testing/synthetic/reference/index/type-predictions.md T11;
knowledge/works/package_defaults.yaml baselines[0], scope_additions[2]–[4], complexity_additions[0]]

| Discipline id | Engaged at | Need | Why | App |
|---|---|---|---|---|
| `consultant.architect` | Brief | Always | Design lead (often a building designer), demolition plan, DA, construction documents, tender | yes, `baselines[0]` |
| `consultant.surveyor` | Before concept | Always | Detail and boundary survey including the existing house, pits and outlet; draft plan of subdivision for the DA; set-out of the party wall on the boundary; final plan of subdivision | yes, `baselines[0]` |
| `consultant.hazmat` | Investigation | Always (pre-2004 house) | Demolition hazmat survey, asbestos register and clearance | **no** rule reads `site.demolition` |
| `consultant.geotechnical` | Concept | Always | AS 2870 site classification from boreholes clear of the house | yes, `baselines[0]` |
| `consultant.structural` | Design development | Always | Slabs, the separating-wall footing, beams, framing review, CC certificates | yes, `baselines[0]` |
| `consultant.civil` | Approval documentation | Always | Existing drainage investigation; stormwater concept with OSD; detailed drainage for each lot; driveway levels; OSD certification and works-as-executed for the positive covenant | yes, `baselines[0]` |
| `consultant.basix` | Approval documentation | Always (NSW) | NatHERS per dwelling; one BASIX project for both dwellings (the Planning Portal's "more than 1 dwelling" path; one certificate is [D]) | **no** |
| `consultant.certifier` | Before CC | Always | Pre-start inspection, CC, principal certifier, critical-stage inspections (R7), OC | yes, `baselines[0]` |
| `consultant.town_planner` | Concept / pre-DA | Usually | SEE covering permissibility, lot size and frontage for the dual occupancy and the subdivision, and DCP controls | yes, `complexity_additions[0]` |
| `consultant.landscape` | Approval documentation | Usually | DCP landscape plan, with landscaped area and deep soil for each lot | **no** |
| `consultant.quantity_surveyor` | Concept, pre-tender | Optional | DA cost estimate report where council asks for a QS; lender drawdown reports | **no** |
| Water Servicing Coordinator | After DA | Always for the subdivision | Lodges Section 73, manages the Notice of Requirements | **no id** |
| Solicitor | Completion | Always for the subdivision | 88B instrument | **no id** |
| `consultant.arborist`, `.bushfire`, `.heritage`, `.acoustic`, `.traffic`, `.waste`, `.building` | Investigation / DA | Conditional | Triggers; a building consultant for dilapidation only where excavation is near neighbours | bushfire and heritage yes; the others no |
| `trade.main_works` | Procurement | Always | One residential head contract, with licensed demolition and asbestos subcontractors | partial: proposed only from gaps on accepted items |

**Not expected for this job, but the app suggests them** [D]: `consultant.electrical`,
`consultant.hydraulic` and `consultant.mechanical`. Services are design-and-install. Section 73 is run by
a WSC, not a hydraulic engineer. Class 1a attached dwellings need no fire engineer, BCA consultant or
access consultant. They need an acoustic consultant only on a noise-affected site.

---

## 5. Approvals and authorities

Clause numbers are from memory and `verified: false` unless an owner ruling or a cited page confirms them.

### Planning pathway (base: DA)

- **One DA** under EP&A Act Part 4 for demolition of the existing dwelling, construction of an attached
  dual occupancy and Torrens title subdivision into two lots, lodged on the NSW Planning Portal. It
  includes:
  - the architectural set, survey, demolition plan and waste management plan;
  - the hazmat report where council asks for it;
  - the draft plan of subdivision;
  - the stormwater concept with OSD, landscape plan, BASIX certificate, SEE, shadow diagrams, cost
    estimate and owner's consent.
- **Permissibility and controls.** From 1 July 2024 dual occupancies are permissible with consent in all
  R2 zones in NSW (Housing SEPP). The SEPP sets non-discretionary standards (verified: false). The LEP may
  set its own minimum lot sizes for the dual occupancy and for its subdivision. Some LEPs restrict Torrens
  subdivision of a new dual occupancy, for example Randwick's 5-year rule after the OC (verified: false).
  Sources: Hornsby Council duty planner page; Mills Oakley commentary (review Q6c).
- **Notification** to neighbours, not public exhibition.
- **Determination** with conditions:
  - s7.11 or s7.12 contributions for the additional dwelling, usually with a credit for the demolished
    house;
  - the **Housing and Productivity Contribution**, which applies to a dual occupancy as medium-density
    residential development, with a credit for the existing lot (DPHI HPC Applicant Guide to Dual
    Occupancies, Dec 2025, §4.1, §4.3);
  - OSD and its positive covenant;
  - Section 73;
  - damage deposit;
  - Tap in before the CC;
  - demolition and asbestos conditions;
  - survey certificates.
- **Alternative: complying development** under the Low Rise Housing Diversity Code (Codes SEPP Part 3B,
  verified: false), with the subdivision approved separately (trigger `cdc-low-rise-code`).

### Certificates

| Certificate | By | When |
|---|---|---|
| BASIX certificate | BASIX assessor | One project for both dwellings (Planning Portal, dual occupancy page); with the DA; revised for the CC if needed [K: knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required] |
| Asbestos clearance certificate | Licensed assessor or hygienist | After removal, before demolition proceeds |
| Construction certificate | Registered certifier | After the pre-start inspection (R7 item 1) and the CC conditions (long service levy, Tap in, s138) |
| Principal certifier appointment and notice of commencement | Owner / certifier | Before any work, including demolition |
| HBCF certificates | Builder's insurer | Before deposit or work; contract over $20,000 including GST (R4); one per dwelling (verified: false) |
| Occupation certificate | Principal certifier | After the final inspection, with BASIX completion; one for both dwellings, or one each |
| Section 73 compliance certificate | Sydney Water, through the WSC | After the Notice of Requirements works and connections. The app's rule puts it before the OC; the map gates the subdivision certificate. Both are consent-dependent, and the test checks only that a Section 73 step exists. [K: knowledge/clusters/services-wet-air/rules.yaml rule.nsw.section-73-certificate, clause_verified false] |
| Subdivision certificate | Council in the base. Some councils issue Torrens subdivision certificates only themselves; a registered certifier only where council permits (verified: false) | After the Section 73 certificate, the DNSP notification of arrangement, NBN provisioning evidence and a survey certificate that each lot contains its own services; the OC first where the consent requires it [D] |
| Plan registration and new titles | NSW Land Registry Services | After the subdivision certificate |

### Authority applications [D]

- **Council:** the DA; two s138 crossings and removal of the old one (verified: false); damage deposit;
  contributions; the subdivision certificate.
- **Long Service Corporation:** levy before the CC (verified: false).
- **Sydney Water:**
  - disconnection of the old house's services;
  - Tap in building plan approval;
  - Section 73 through a WSC. It gives a Notice of Requirements: new connections for each lot, or
    developer works for a main extension. It carries **infrastructure contributions** per equivalent
    tenement, reintroduced from 1 July 2024 and phased in to full price from 1 July 2026 (Sydney Water
    land development pages, review Q6b).
  - Each lot's private lines must stay inside that lot, or run in an easement (dual occupancy guide §4.4,
    §4.5).
- **DNSP:** abolishment of the old service; two new connections; a notification of arrangement for the
  subdivision.
- **NBN Co:** a new-development application for two premises.
- **LRS:** plan of subdivision and 88B instrument.
- **SafeWork NSW:** licensed asbestos removal notification before removal (verified: false).
- **Fire and Rescue NSW:** none for Class 1a.

### Inspections and hold points

The principal certifier's inspections follow owner ruling R7. Item 1, the pre-start site inspection,
comes **before the CC is determined**, with the existing house still standing (as T10 places it before
the CDC determination). The rest: excavation before footings; before pouring the slab; before covering
the frame; before covering wet-area waterproofing; before covering stormwater connections; pool items
(trigger only); the final inspection before the OC. The clause stays `verified: false`.

For this type the frame inspection must also see the separating wall before it is lined or roofed over,
including its termination at the roof. A PM adds a builder or architect hold point at the roof line. [D]
`if.party-wall-roof-continuity` is a boundary interface, so it makes no hold point. The app's hold points
`if.frame-inspected-before-roof` and `if.reinforcement-inspected-before-pour` both fire. The second fires
through `fr.slab-on-ground-1a`, confirmed by a Go run. [K: knowledge/clusters/fire/interfaces.yaml;
knowledge/clusters/structure/interfaces.yaml]

---

## 6. Cost plan shape

Lines only, never amounts. [K: knowledge/costs/starter_lines.yaml; knowledge/costs/elements.yaml]

**Fees and charges.**

- DA fees and contributions (P04, yes).
- Housing and Productivity Contribution (no line).
- CC and certifier fees (P05, yes).
- Service authority connection fees (P06, one generic line).
- Sydney Water Section 73 fees and infrastructure contributions (no line).
- Subdivision certificate and LRS registration fees (no line).
- Long service levy (no line).
- Council road-reserve fees and deposit for two crossings (no line).
- HBCF for both dwellings (no line; inside the builder's price).

**Consultants.** Architect, Structural, Civil, Geotechnical, Surveyor, Certifier and Town planner are
F-lines from the adopted packages. Hazmat survey, BASIX assessor, Landscape, WSC, Solicitor and QS
(optional) have no line. The app adds Electrical, Hydraulic and Mechanical fee lines that this job does
not need. The certifier is counted twice (F-line and P05).

**Construction.**

- Preliminaries and margin (no line; spread pro rata).
- **Demolition and hazardous materials removal** (rolls into 01 Site works; no line of its own and no
  demolition share in the split).
- Services disconnections and make-safe (no line).
- Provisional sum for latent ground conditions after demolition (no line).
- 01 Site works: drainage, OSD, services to each lot, two driveways, landscape, dividing fence.
- 02 Substructure; 03 Structure; 04 Envelope.
- 05 Interiors, including acoustic separation and two kitchens.
- **06 Passive fire protection** (the separating wall).
- 08 Mechanical; 09 Hydraulic (two rainwater tanks); 10 Electrical; 11 Communications.
- 13 Stairs, ramps, handrails and nosings; 13 Compliant balustrades and barriers.
- Provisional sums and PC items (no line).

The app also makes 07 Active fire protection from the smoke alarms, which a QS carries in electrical.

**Passive fire, stairs and barriers get no money.**

- The townhouse split is `residential_house_new`, which has no `fire-passive` share.
- `access-egress` is itemised, so its group line does not post and its children carry a parent.
- `spreadConstructionBudget` writes only to posting top-level lines whose system has a weight.

So a stated budget leaves 06, 13.3 and 13.9 unpriced while every other line is filled. [K:
knowledge/costs/budget_splits.yaml residential_house_new; knowledge/costs/elements.yaml code 13;
internal/store/cost_budget.go spreadConstructionBudget]

**Contingency.** Design contingency, construction contingency, escalation and exclusions (all yes);
owner-supply items only when stated. Sale costs belong in the developer's feasibility. The app has no
split by lot, which a developer wants for the lender and the sale. [D]

---

## 7. Program shape

**Skeleton:** `sk.new-traditional-da`. [K: knowledge/program/skeletons.yaml]

**The order a PM actually runs** [D]:

1. **Investigation.**
   - Planning certificate and title; LEP lot size and frontage.
   - LFAI register; demolition hazmat survey.
   - Sydney Water sewer diagram; CCTV or locate the existing drainage.
   - Services search; detail and boundary survey; geotechnical investigation.
2. **Concept and design development.** The two-dwelling layout, the lot split and the separating wall
   system.
3. **Approval documentation.**
   - DA drawings, demolition plan, demolition work plan and waste plan.
   - BASIX; stormwater with OSD; landscape plan.
   - Draft plan of subdivision; SEE.
4. **DA.** Lodge, notification, assessment, determination.
5. **After consent.**
   - Section 73 through the WSC, giving the Notice of Requirements.
   - Construction documentation; Tap in; long service levy.
   - **The pre-start inspection (R7 item 1)**, then the CC.
6. **Procurement.**
   - Tender, award, HBCF.
   - Meter removal and abolishment requests; occupants vacate.
   - Principal certifier and notice of commencement.
7. **Demolition.**
   - Site establishment and sediment control; disconnect and abolish services.
   - SafeWork notification, licensed asbestos removal, clearance certificate.
   - Demolish and clear; remove old footings and buried drainage.
8. **Construction.**
   - Set-out on the proposed boundary; earthworks; footings; slab (reinforcement hold point);
     identification survey.
   - Separating wall; frames; roof, with the wall termination inspected first; windows; cladding.
   - Rough-in; frame inspection; insulation; linings; wet areas; stairs and balustrades; joinery;
     fit-off; painting; floors.
   - Stormwater and OSD (inspection); tanks; connections for both lots; driveways and crossings;
     landscape and dividing fence.
   - DNSP notification of arrangement, NBN evidence, and the survey certificate of services within each
     lot.
9. **Completion.**
   - PC; works-as-executed and OSD certification; OC; Section 73 certificate.
   - Final plan and 88B; subdivision certificate; LRS registration; handover.
10. **Defects.** Defects inspection and the contract's defects liability period (R8).

**Logic that drives the job:**

- Hazmat survey → asbestos removal → clearance → demolition → earthworks.
- DA consent → Section 73 → Notice of Requirements → connections → Section 73 certificate → subdivision
  certificate → registration → sale settles.
- Pre-start inspection → CC.
- Separating wall → roof termination inspection → roof cover.

**Long-lead items:**

- Service abolishment before demolition.
- Windows, joinery, frames and trusses.
- Two DNSP connections.
- The Notice of Requirements and any developer works.
- NBN for two premises; LRS registration.

**What the app gets wrong.**

- As in T01: the CC sits in approvals ahead of construction documentation, the notification is labelled
  "Public exhibition", and `fr.ac-1a` follows the linings.
- Specific to this type: no subdivision chain, no separating wall, stair, barrier or OSD rows, and one
  services row and one utility row for two lots.
- Demolition: `fr.demolition-1a` fires only on a `remove` action, and there is no hazmat, asbestos or
  disconnection row.

---

## 8. RFP shape

The `house` module (only 1a and 10a parts) and the `existing_building` module (`det.existing_building:
true`) are on; `beyond_house` is off. [K: knowledge/rfp/modules.yaml]

### Architect (`architecture`)

- **Must carry:**
  - concept and schematic design for two dwellings and the lot split, on the survey of the existing site;
  - a demolition plan and the investigations needed (`existing_interfaces`, I through the module);
  - DA documentation for demolition, the dual occupancy and the subdivision;
  - planning response; design development;
  - `house_documents` for both dwellings;
  - separating-wall details to Housing Provisions 9.3.1 and 10.7.1 (the roof termination, small battens
    and sarking only, shared eaves lined, discontinuous construction);
  - kitchen, bathroom and joinery design; selections schedule; specification;
  - bidding support; certifier liaison; lodging the DA.
- **Options:** construction support; contract administration.
- **Must not carry:** BIM; tenant fit-out; signage; façade certification; apartment design quality;
  Green Star; regulated design (R3); D&C or post-novation; the brief and masterplan stage.
- **App:** blocks on `fact:budget` until the budget is read. No row names the separating wall or the
  subdivision.

### Structural (`structural`)

- **Must carry:** `residential_framing` scope, plus:
  - the footing under the separating wall and the wall's support and restraint;
  - slab design noting old footings and fill removed.
- **Must not carry:** existing-structure assessment; strengthening; shoring; dilapidation; demolition
  drawings; Revit model.
- **App:** adds `existing_assessment` and a `strengthening` option through the existing-building flag
  (as T10).

### Civil (`civil`)

- **Must carry:**
  - investigation of the existing drainage and kerb outlet (`retained_services`, I through the module);
  - stormwater concept with OSD (`osd_new` I) and discharge for both lots;
  - detailed drainage per lot, with any interallotment easement identified;
  - driveway long sections for two crossings;
  - erosion control from the start of demolition;
  - works-as-executed and OSD certification.
- **Option:** act as WSC.
- **Must not carry:** MUSIC, NorBE, public road works, flood report, 3D model, D&C stages.
- **App:** no Section 73 row.

### Geotechnical (`geotechnical`)

- **Must carry:** boreholes clear of the existing house; AS 2870 classification; advice on removing old
  footings and fill; permeability advice if the OSD needs it. Option: inspections.
- **App:** the `g4_planning` stage comes out an option, against an expected absent.

### Surveyor (`land-surveying`)

- **Must carry:**
  - detail and contour survey showing the existing house, pits, outlet, junction and crossing;
  - boundary identification with encroachments;
  - adjoining features;
  - buried drainage (`buried` I);
  - sewer records;
  - the **draft plan of Torrens subdivision**;
  - the **final plan of subdivision** with the party wall on the boundary;
  - inputs to the 88B instrument.
- **Options:** set-out; identification surveys.
- **App:** `titling` is always O, and `s5_support` and `s7_completion` are priced as options. This is
  the type's largest RFP difference. [K: knowledge/rfp/land-surveying.yaml titling]

### Certifier (`bca-certification`)

- **Must carry:** Package B, comprising:
  - the pre-start site inspection before determining the CC (R7 item 1);
  - the CC for both dwellings;
  - principal certifier services;
  - every R7 inspection, including the separating wall;
  - the OC with BASIX completion.
- **Option:** the subdivision certificate, only where council permits a registered certifier to issue it.
- **App:** defaults to the A package without the client flag, prices `a4_construction` as an option,
  and has no subdivision row.

### Hazmat (no template)

- **Must carry:** demolition survey of every structure with destructive sampling; asbestos register and
  removal scope; lead paint, SMF and PCB identification; LFAI check; clearance inspection and certificate.
- **Must not carry:** removal works; soil contamination assessment.

### BASIX assessor (no template)

NatHERS for each dwelling, one BASIX project for both dwellings, one revision for the CC.

### Landscape, town planner, quantity surveyor

- **Landscape:** concept and DA landscape plan for both lots. The app asks on the client flag
  `landscape_design` for the concept and approval stages, and prices the later stages as options, which
  matches the key.
- **Town planner:** permissibility, lot size, frontage and DCP advice; an SEE covering demolition, the
  dual occupancy and the subdivision; lodgement; assessment. No clause 4.6 request unless
  `cl46-lot-size`.
- **QS:** cost basis, concept estimate, the DA cost estimate report, and a pre-tender estimate with
  demolition, asbestos, disconnection and latent-condition allowances.

### WSC and solicitor

No discipline id and no template. WSC: lodge Section 73, manage the Notice of Requirements and any
developer works, obtain the certificate. Solicitor: the 88B instrument (party wall, services and drainage
easements, OSD covenant).

---

## 9. Triggers

Sixteen conditions. The machine-readable form is in `shape.yaml`.

| Id | How a document shows it | How a document denies it | What it adds |
|---|---|---|---|
| `cdc-low-rise-code` | "CDC for an attached dual occupancy under the Low Rise Housing Diversity Code"; "the design meets the code's standards" | "A DA will be lodged"; "the design exceeds the code's height standard, so a DA will be lodged" | CDC rows (`fr.cdc-application`); contributions still payable; separate subdivision approval; no town planner. The CC and certifier fees line stays, because the certifier is paid for the CDC. The app keeps the town planner, blocks the town-planning and certifier RFPs on `cdc`, drops the whole DA-fees line, and moves to `sk.new-traditional-cdc`, which has no construction-documentation stage |
| `strata-subdivision` | "Strata subdivision into two lots and common property"; "the driveway will be common property"; "SP plan and by-laws" | "Torrens title subdivision, each lot with its own frontage" | Strata plan, certificate, by-laws, owners corporation; replaces the Torrens certificate and 88B rows |
| `front-rear-shared-driveway` | "Dwelling 2 at the rear, reached by a shared driveway"; "right of carriageway over Lot 1"; "battle-axe handle" | "Each dwelling has its own driveway and crossing" | Shared driveway; right of carriageway and services easements; the rear lot's water line in a protective conduit (guide §4.5); one crossing, still under s138 |
| `three-plus-townhouses` | "Multi dwelling housing comprising 4 townhouses"; "community title into 4 lots"; "R3 zone" | "Attached dual occupancy (two dwellings only)" | Traffic and waste consultants (never suggested); common driveway, bin store, mailboxes |
| `detached-dual-occupancy` | "Detached dual occupancy: a front dwelling and a rear dwelling"; "two buildings separated by a courtyard" | "Attached dual occupancy with a common separating wall" | No separating wall; more envelope. `flag.multi_building` turns on the architect's masterplan stage and the staged module, which the job does not need |
| `vacant-lot` | "The site is vacant"; "the previous house was demolished under a separate consent" | "The existing dwelling built in 1958 will be demolished" | Removes hazmat, demolition, disconnections and asbestos rows. `det.existing_building` false turns off `existing_interfaces`, `retained_services` and `existing_assessment` |
| `no-subdivision` | "Both dwellings will be retained on the one title and rented"; "No subdivision is proposed" | "Torrens title subdivision into two lots is proposed" | Removes the subdivision plans, certificate, registration, solicitor and fees. Sydney Water may still want two connections if a later subdivision is intended (guide §4.2) |
| `stormwater-easement` | "Stormwater cannot reach the street by gravity"; "interallotment drainage easement through No. 22"; "deferred commencement" | "Discharges by gravity to the kerb" | Easement negotiation and costs; deferred commencement. No app change |
| `sewer-main-in-lot` | "Sewer main crosses the rear of the lot"; "under the proposed rear dwelling"; "building over a Sydney Water asset" | "No Sydney Water assets within the property" | Building-over approval; piers; structural authority-asset stage; architect water-asset submission as an option |
| `sewer-extension-required` | "The Notice of Requirements requires a sewer main extension to serve Lot 2"; "developer works are needed" | "Each lot can connect to the existing main" | Developer works by an accredited designer and constructor; critical to Section 73. No app change |
| `trees-affected` | "The second driveway encroaches into the street tree's TPZ"; "excavation for the rear dwelling's footings is within the structural root zone of the neighbour's tree" | "No trees affected"; a tree present but unaffected | Arborist (never suggested), AS 4970 report, tree protection, permit |
| `bushfire-prone` | "Mapped bush fire prone land"; "BAL-29"; "backs onto bushland" | "Not identified as bush fire prone land" | Bushfire consultant; AS 3959; the subdivision is integrated development needing an RFS bush fire safety authority (s100B, verified: false) |
| `flood-affected` | "Subject to flood related development controls"; "floors 500 mm above the 1% AEP"; "overland flow path" | "Not subject to flood related development controls" | Flood advice and report; flood-compatible construction; no CDC |
| `heritage-conservation-area` | "Within the Ashwell Heritage Conservation Area"; "heritage officer will review the infill dual occupancy" | "Not a heritage item and not in a conservation area" | Heritage consultant and referral; DCP may restrict dual occupancy |
| `noise-affected-site` | "Fronts a classified road; road traffic noise requires an acoustic report"; "inside the ANEF 20 contour"; "next to the rail corridor; rail noise assessment required" | "Quiet local street, away from main roads, rail and flight paths" | Acoustic consultant (never suggested); facade acoustics |
| `cl46-lot-size` | "Proposed Lot 2 is 290 m², below the 300 m² minimum; a clause 4.6 variation request is submitted"; "The LEP's 15 m minimum lot width for a dual occupancy is not met (14.2 m); a clause 4.6 request accompanies the DA" | "Both lots exceed the standards"; "the frontage is 14.2 m against the DCP's 15 m control; a merit justification is given" | Town-planning `variation_request` stage. A DCP control needs a merit argument, not clause 4.6 |

**Left out of the key, with reasons:**

- `lep-limits-subdivision`: the LEP bars or defers Torrens subdivision of a new dual occupancy (for
  example Randwick). It changes the program end-state, but the rule varies by LEP and stays unverified.
- `sloping-lot-retaining`: retaining walls, structural retaining design and cut and fill. It is common
  on Sydney's north and south, and its content is already keyed in T01's `sloping-site`.
- Townhouses over a shared basement: adding Class 7a makes the project industrial through
  `det.ncc_class`, so `baselines[1]` brings Traffic, BCA, Access, Mechanical and Fire Services. That is a
  different building type (closer to T02); keyword trap 16 covers it.
- Acid sulfate soils, gas, pools, a secondary dwelling, and solar PV for BASIX: keyed in T01 or T12.

**Zero-delta triggers.** `stormwater-easement`, `sewer-extension-required` and `noise-affected-site`
change nothing the app produces today.

---

## 10. Keyword traps and negations specific to this type

1. **"Party wall" or "separating wall" adds a neighbour's structure.** `implied_by party-wall` adds
   `site.adjoining-structures` with `fire-passive.compartments` and `interiors.acoustic-separation`. The
   last two are right here; the first is wrong. "Common wall" matches too. [K:
   knowledge/profile/scope_defaults.yaml implied_by party-wall]
2. **"Dual occupancy" may be read as a house.** The subclass label is "Townhouses (Class 1a)". If a
   reading picks `house`, scope loses the separating wall, acoustic separation, stairs and barriers. [K:
   knowledge/profile/taxonomy.yaml townhouses]
3. **"Granny flat" is not a dual occupancy.** A secondary dwelling (T12) cannot be subdivided off. A
   land-use table listing "dual occupancy (detached)" says nothing about this project.
4. **"Two dwellings" in words is not read.** The `dwellings` trigger needs a digit
   (`\b\d+\s+(dwellings|lots|homes)\b`). "2 lots" matches; "Lot 2" does not. "4 lots" may include a
   common-property lot.
5. **"Stage 1 / Stage 2" or "two buildings".** `flag.staged` fires on `stages? [1-9]`, and
   `flag.multi_building` on "two buildings". Either turns on the architect's masterplan stage. [K:
   knowledge/profile/flag_readings.yaml]
6. **"Building a dual occupancy".** `multi_building` also triggers on `\bbuildings? [a-d]\b`. Read
   case-insensitively, "building a dual occupancy" raises the question. The answer is no.
7. **"Class 1" alone does not set the class.** It must say "Class 1a". [K: knowledge/determinants.yaml
   ncc_class triggers]
8. **"Existing dwelling" is not "existing building".** Only the phrase "existing building" asks
   `det.existing_building`. "1950s" sets no year; "built in 1958" does. "Knocked down" and "removed"
   match no demolition pattern; only "demolish" or "demolition" does.
9. **"Brick veneer" does not imply masonry.** `implied_by masonry` needs "masonry", "brickwork" or
   "re-pointing", so without one of those `structure.masonry` and `fr.masonry-1a` never arrive.
10. **"Lot 1 DP 12345"** is the existing lot. "Proposed Lot 1 and Lot 2" are the future lots.
11. **"Section 73"** is usually unnecessary for a single house (T01). Here it is required, so a missing
    Section 73 step is the defect.
12. **The sewer matches `authority_asset` everywhere.** "Sewer main" in the street, the Section 73 text
    and "sewer main extension" all match it, so the base documents will raise the question. The answer
    stays no unless a main is inside the lot.
13. **"No on-site detention required"** and "OSD exempt" match `osd_required`; the answer must be no.
14. **"Strata"** in "the neighbouring strata building" or "strata title is not proposed" is not a strata
    subdivision.
15. **"Firewall", "fire-rated wall" and "acoustic wall"** in a duplex mean the separating wall. They need
    no fire engineer and no acoustic consultant.
16. **"Basement parking" for townhouses.** It adds Class 7a, which the app maps to industrial, and five
    commercial consultants follow.
17. **"Heritage", "flood", "lodge", "public exhibition"** as in T01. **"Sale", "off the plan"** are the
    developer's sale, not project scope.

---

## 11. Gaps in the built-in knowledge

1. **Subdivision has no content.**
   - No fragment for the draft or final plan, Section 73, the DNSP, NBN or survey evidence, the 88B
     instrument, the subdivision certificate or LRS registration.
   - No cost lines.
   - The surveyor's `titling` is always optional.

   The app's program ends at handover, with no titles.
2. **No Water Servicing Coordinator or solicitor discipline.**
   - The Section 73 rule reaches Implications only.
   - Its "before the OC" timing differs from the subdivision-certificate gate; both are
     consent-dependent.
3. **The separating wall is in scope but has no program row, no hold point and no budget share.**
   - The default item is `fire-passive.rated-elements`, while the knowledge describes party walls under
     `compartments`.
   - The house split has no fire-passive share.
4. **The stairs and barriers are in scope with no fragment and no budget share.** The `access-egress`
   group is itemised, and the spread skips child lines.
5. **Demolition.**
   - `fr.demolition-1a` needs a `remove` action that `implied_by strip-out` never gives.
   - The hazmat row exists only for refurb or extend.
   - No package rule suggests hazmat.
   - `det.existing_building` true adds a structural existing-assessment stage that a knock-down does not
     need.
6. **Consultants:** BASIX, Landscape, Arborist, Acoustic and Hazmat are never suggested; Electrical,
   Hydraulic and Mechanical are over-suggested; three-plus townhouses get no Traffic or Waste.
7. **Cost plan:** none of these has a line:
   - subdivision fees;
   - Section 73 and infrastructure contributions;
   - the Housing and Productivity Contribution;
   - long service levy, s138 and HBCF;
   - demolition, disconnections and latent conditions;
   - preliminaries and provisional sums.

   The certifier is double-counted, and there is no split by lot.
8. **Program:**
   - the CC sits before construction documents;
   - no rows for BASIX, Tap in, the levy, HBCF, notice of commencement, the pre-start inspection, OSD,
     tanks, driveways, or most critical-stage inspections;
   - one services row for two lots;
   - investigation fragments need `investigate` actions;
   - under the CDC trigger the CDC skeleton has no construction-documentation stage.
9. **RFPs:**
   - nothing names the separating wall, the subdivision, Section 73 or the subdivision certificate;
   - the surveyor's subdivision and completion stages are options;
   - the certifier defaults to BCA advisory;
   - architecture blocks on budget;
   - town-planning and certifier RFPs block on CDC;
   - `multi_building` turns on a campus masterplan stage.
10. **Profile:**
    - no dual-occupancy wording in the subclass;
    - no storeys field for townhouses;
    - the dwellings trigger needs digits;
    - `implied_by party-wall` adds `site.adjoining-structures`.
11. **No calendar.**

---

## 12. Sources and unverified claims

**Opened in the repo:**

- `testing/synthetic/PROJECT_TYPES.md`; `DISCOVERY.md` §5–§10; `reference/KNOWLEDGE_INDEX.md`;
  `reference/OWNER_RULINGS.md`; `reference/index/type-predictions.md` (T11), `systems.md`,
  `program-fragments.md` and `rfp-templates.md`
- `knowledge/profile/scope_defaults.yaml`; `knowledge/profile/taxonomy.yaml`;
  `knowledge/profile/flag_readings.yaml`; `knowledge/determinants.yaml`
- `knowledge/costs/elements.yaml`, `starter_lines.yaml` and `budget_splits.yaml`
- `knowledge/clusters/fire/systems.yaml` and `interfaces.yaml`; `knowledge/clusters/envelope/systems.yaml`;
  `knowledge/clusters/structure/systems.yaml`; `knowledge/clusters/services-wet-air/rules.yaml`
- `knowledge/rfp/land-surveying.yaml`, `town-planning.yaml` and `traffic.yaml`
- `internal/store/cost_budget.go`; `data/intake/disciplines.json`
- The T01 and T10 shapes and domain maps (T10 for the demolition handling and the existing-building RFP
  rows)

**Public pages cited by the review (not opened by me; the owner should check them before any
`verified: true`):**

- ABCB NCC 2022 Housing Provisions Parts 9.3 and 10.7
- Planning Portal BASIX dual-occupancy page
- DPHI HPC Applicant Guide to Dual Occupancies (Dec 2025)
- Sydney Water land development and infrastructure contributions pages
- Sydney Water Servicing Dual Occupancy guide (§4.2, §4.4, §4.5, §4.7)
- Hornsby Council duty planner page; Mills Oakley commentary

**Instruments named from professional knowledge, every clause `verified: false`:**

- EP&A Act 1979 Parts 4 and 6, s7.11, s7.12, s10.7
- Housing SEPP 2021; Codes SEPP 2008 Part 3B; SEPP (Transport and Infrastructure) 2021; the Standard
  Instrument clause 4.6 and lot size clauses
- Conveyancing Act 1919 s88B and s88E; Strata Schemes Development Act 2015
- Sydney Water Act 1994 s73; Rural Fires Act 1997 s100B; Roads Act 1993 s138
- Home Building Act 1989
- AS 2870, AS 3660.1, AS 3740, AS 4970, AS 3959, AS 2021, AS 2601

**Claims I am least sure of:**

- One HBCF certificate per dwelling.
- Whether a registered certifier may issue the Torrens subdivision certificate at a given council.
- The Housing SEPP non-discretionary standards and how they interact with LEP minimum lot sizes.
- The DNSP, NBN and survey evidence each council asks for before the subdivision certificate.

## Review applied (30 October 2026)

All 11 findings in `REVIEW.md` were applied to this file and `shape.yaml`, with no disputes. The headline
change is F1: demolition of the existing house is now in the base, handled as in T10. The pre-start
inspection sits before the CC, and `vacant-lot` is the variant. The coordinator's direction also marks
the reinforcement hold point `yes`, because it fires through `fr.slab-on-ground-1a`.
