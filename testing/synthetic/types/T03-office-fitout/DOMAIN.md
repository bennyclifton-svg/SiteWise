# T03 Commercial office fit-out: domain map

Status: draft for owner review (`reviewed: false`). Default state NSW. The machine-readable key is
`shape.yaml` in this folder. It passes `testing/synthetic/tools/validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Rule numbers I could not confirm against the
instrument are marked `verified: false`.

---

## 1. The job

**What it is.** A tenant fit-out of an existing NCC Class 5 office floor in an occupied, landlord-controlled
building. The **base scenario** is the most common Sydney version [D]:

- one whole floor of about 1,200 m² NLA in a 24-storey A-grade tower (`hdr.scale.nla_sqm: 1200`);
- the floor is handed over in **base-building condition**: suspended ceiling grid with base-building
  lighting, VAV air-conditioning, sprinklers, smoke detection and EWIS, the tenant distribution board, and
  amenities on the core, all owned and maintained by the landlord;
- the tenant builds open-plan work points, offices and meeting rooms, a boardroom, reception, a kitchenette
  and breakout space, a comms room, and storage;
- the other floors stay occupied by other tenants throughout (`hdr.cond.operational_constraints:
  partial_occupation`); noisy work and deliveries are after hours under the landlord's
  building rules (`hdr.cond.access_constraints: restricted_hours`);
- building completed in 2008, so no asbestos register applies (the `asbestos-in-tenancy` trigger covers
  older buildings);
- complying development certificate (CDC) by a private certifier; not heritage, not a change of use;
- traditional procurement: an interior designer leads, services engineers document, and fit-out
  contractors tender a lump sum.

**Scale.** $0.3–5M (PROJECT_TYPES.md). Cost is driven by area, density of enclosed rooms, quality of
finishes and joinery, and how much the base-building services must change. [D]

**Who procures it and how.** The client is the tenant, usually through a **tenant's project manager**, who
runs the agreement for lease obligations, the consultant team, the landlord's approvals and the
contractor. The usual routes [D]:

- **Traditional (base):** interior designer (or an architectural practice's interiors team) designs and
  documents; a mechanical, electrical, hydraulic and fire services engineer (often one multi-services
  firm) documents the services; a QS estimates; select tender to 3–5 fit-out contractors from the
  landlord's approved list; AS 4000 (or AS 2124) with fit-out amendments (`fact.contract_form: as4000`).
- **Design and construct:** the tenant tenders a D&C fit-out on a concept and performance brief, and the
  contractor's engineers design the services (trigger `design-and-construct`).

The **landlord** is a third party with real power: it approves the concept and detailed design, approves
the contractor, controls access, lifts, the loading dock and services shutdowns, and often nominates the
contractors who may touch the base-building fire and BMS systems. Its rules are in the **fit-out guide**
attached to the agreement for lease. [D]

**What makes it go wrong** [D]:

- **The lease date drives everything.** The access date and the rent-free fit-out period are fixed in the
  agreement for lease; late landlord approval, late CDC or late long-lead items eat the fit-out period.
- **Landlord approvals.** Concept and detailed design approval by the base-building engineers, contractor
  approval, permits to work and hot-works permits. Each is a gate the tenant does not control.
- **Base-building capacity.** Spare AC capacity and outside air, after-hours AC, the tenant's power
  allocation, the sprinkler and EWIS zones and the fire engineering report all limit the design. Knowledge
  has this only as an Implication [K: knowledge/clusters/services-wet-air/interfaces.yaml
  if.tenant-fitout-base-building-hvac].
- **Nominated contractors.** The landlord's fire and BMS contractors price without competition and keep
  their own program.
- **Occupied building.** After-hours premiums, lift and dock bookings, noise and dust complaints, fire-system
  isolations that must be restored every night.
- **Above-ceiling surprises.** Services in the way, base-building defects found on opening up, asbestos in
  older buildings.
- **Long-lead items.** Joinery, glazed partition suites, imported furniture and AV equipment.
- **Completion paperwork.** The final fire safety certificate and the OC need every altered fire safety
  measure tested and certified, and the landlord wants as-builts and manuals before it releases the floor.

---

## 2. Systems and work items

Refurb starts with **no scope defaults** [K: knowledge/profile/scope_defaults.yaml empty_work_types]. The
preset `typical_fit_out` lists 20 interiors and services systems but is only offered in the scope picker
[K: knowledge/profile/scope_defaults.yaml presets]. So every work item below reaches the app only from a read
document or the user. The default action on refurb is `alter` [K: knowledge/works/actions.yaml
work_type_defaults]; where a different action is right the shape says so.

**site**
- Temporary protection of the goods lift, lobby and corridors is the builder's preliminaries, not a work
  item. The app's occupied-protection fragment hangs off `site.impact-protection`
  [K: knowledge/program/fragments.yaml fr.occupied-protection], which is the wrong system: bollards and
  column guards [K: knowledge/clusters/structure/systems.yaml site.impact-protection] (gap).
- `site.signage` **new**: tenancy entry and wayfinding signage to the landlord's rules.
- `site.demolition`: **not in the base** (base-building condition). Added by `existing-fitout-strip-out`
  and `asbestos-in-tenancy` with action `remove`.
- Every other site child (earthworks, stormwater, services connections, landscape, trees and so on) is out
  of scope: the works are inside the building and take services from the base building. [D]

**substructure**: out of scope; no ground works.

**structure**: out of scope in the base. `structure.alterations` (slab penetration, trimming steel) arrives
with `internal-stair`; `structure.design-basis` (an investigate of floor loads) with `heavy-floor-loads`.

**envelope**: out of scope. Facade, windows and blinds are base building. [D]

**interiors** (the bulk of the cost)
- `interiors.walls-linings` **new**: plasterboard partitions, acoustic partitions to the boardroom.
- `interiors.internal-glazing` **new**: glazed meeting-room fronts and suites with manifestation; long lead.
- `interiors.ceilings` **alter**: the base-building grid is altered; tiles and lights at new rooms are
  removed and stored for the landlord; plasterboard ceilings and bulkheads in feature areas.
- `interiors.internal-doors` **new**; `interiors.joinery` **new** (reception, kitchenette, storage; long
  lead); `interiors.floor-finishes` **new** (carpet tiles, vinyl); `interiors.painting` **new**.
- `interiors.room-acoustics` **new**: meeting-room and boardroom speech privacy.
- `interiors.ffe` **new**: loose furniture and workstations, tenant supply outside the contract.
- Not in scope: `wet-area-waterproofing` and `tiling` (amenities are base building; added by
  `tenant-amenities`), `acoustic-separation` (no new separation between occupancies).

**fire-passive**
- `fire-passive.penetrations` **new**: fire stopping where new cabling and pipe pass through the rated riser
  and comms-room walls.
- `fire-passive.compartments` only with `internal-stair` (a void through a floor).

**fire-active** (base-building systems the tenant alters, often through the landlord's nominated contractor)
- `fire-active.sprinklers` **alter**: relocate and add heads for new rooms (AS 2118.1, clause verified:
  false; [K: knowledge/clusters/fire/rules.yaml rule.as2118.sprinkler-design] also marks its clauses
  unverified).
- `fire-active.detection` **alter**: detectors in enclosed rooms (AS 1670.1, verified: false).
- `fire-active.occupant-warning` **alter**: EWIS speakers and strobes (AS 1670.4, verified: false;
  [K: knowledge/clusters/fire/rules.yaml rule.as1670.warning-monitoring]).
- `fire-active.emergency-lighting` **alter**: emergency lights and exit signs (AS 2293.1, verified: false).
- Not in scope: hydrants, fire-water supply, pumps and tanks, smoke control, fire control centre, special
  suppression (a comms-room gas system is possible but uncommon). Portable extinguishers are the landlord's.

**mechanical**
- `mechanical.air-conditioning` **alter**: re-zoning the VAV system, added terminals and thermostats.
- `mechanical.air-distribution` **alter**: flexible duct, diffusers, return air paths.
- `mechanical.ventilation` **alter**: outside air to high-occupancy rooms within the base-building allowance.
- `mechanical.local-exhaust` **new**: kitchenette and print-room exhaust.
- `mechanical.controls-bms` **alter**: BMS points and after-hours AC controls, by the landlord's BMS
  contractor.
- Not in scope: central plant, cooling towers, car-park ventilation, fire-mode systems. Supplementary
  cooling and refrigerant pipework arrive with `comms-room-cooling`.

**hydraulic**
- `hydraulic.cold-water` **alter**, `hydraulic.hot-water` **new** (local unit), `hydraulic.sanitary`
  **alter**: the kitchenette only, off the base-building tenant points.
- Not in scope: gas (none on the floor, `det.gas_supply: none`), roof drainage, rainwater, trade waste,
  backflow (base building).

**electrical**
- `electrical.switchboards` **alter**: new circuits and a tenant sub-board off the base-building tenant DB.
- `electrical.cabling-wiring` **alter**: power to workstations, joinery and appliances.
- `electrical.lighting` **alter**: re-arranged luminaires, feature lighting, controls. Section J applies to
  the new lighting and power (NCC 2022 Part number verified: false; the knowledge itself flags the
  numbering as unsettled [K: knowledge/clusters/electrical-comms/rules.yaml rule.ncc.lighting-power-density]).
- Not in scope: utility supply (no new connection), standby power (added by `comms-room-cooling`),
  solar, EV, metering beyond the base-building tenant meter, temporary power (the builder uses base-building
  power).

**comms-security**
- `comms-security.structured-cabling` **new**; `comms-security.ict-rooms-pathways` **new** (comms room,
  racks, riser pathway); `comms-security.access-control` **new** (integrated with base-building
  security); `comms-security.audiovisual` **new** (often tenant direct; the builder provides power, data and
  conduits).
- Not in scope: carrier lead-in (base building), CCTV and intruder alarm (base building unless the tenant
  asks), active networks (tenant IT, owner supply).

**vertical-transport**: out of scope. The goods lift is used and protected, not altered.

**access-egress**
- `access-egress.exits` **alter**: new rooms change travel distances to the base-building fire stairs; a
  design outcome the BCA consultant checks. Not costed by design [K: knowledge/costs/elements.yaml code 13].
- `access-egress.egress-doors` **new**: tenancy entry doors with egress hardware and access control.
- `access-egress.accessible-access` **alter**: accessible path of travel, door circulation, hearing
  augmentation in the boardroom.
- Not in scope: stairs and ramps, barriers (both with `internal-stair`), accessible facilities (with
  `tenant-amenities`), livable housing, pool barriers.

---

## 3. Existing-building interfaces

The whole job is an existing-building interface. [D] unless tagged.

- **Occupation.** The building stays occupied by other tenants. The fit-out floor is vacant (base), but
  lifts, lobbies, the loading dock and the fire systems are shared. After-hours works, lift and dock
  bookings, protection of common areas, and security inductions are normal. The RFP driver turns
  `partial_occupation` into `live_operations: yes` [K: internal/rfp/drivers.go readFlags], which switches on
  the `live_site` module and occupied-route rows (access `occupied`). Staff staying on the floor is the
  `staged-occupied-tenancy` trigger.
- **Services cut-overs.** Every alteration to sprinklers, detection, EWIS, AC and BMS is a base-building
  system. The fire system is isolated zone by zone under a permit, the monitoring provider is told, and
  the system is restored at the end of each shift. The base-building fire panel's cause-and-effect is
  updated and integrated-tested at the end. Hot works need a landlord permit.
- **Base-building capacity.** AC capacity, outside air, after-hours AC, the tenant's power allocation,
  hydraulic tenant points and floor loads are set by the base building. The landlord's engineers approve
  the tenant's design against them. Knowledge holds this as an Implication only
  [K: knowledge/clusters/services-wet-air/interfaces.yaml if.tenant-fitout-base-building-hvac] and an
  unforeseen record whose de-risk step is landlord re-approval of tenant changes
  [K: knowledge/clusters/services-wet-air/unforeseen.yaml, signal
  sig.landlord-approval-of-tenant-services-stated in signals.yaml].
- **Hazardous materials.** A 2008 building needs no asbestos register (WHS Regulation applies to workplaces
  built before 31 December 2003, verified: false). The app still adds a hazardous-materials survey on every
  refurb [K: knowledge/program/fragments.yaml fr.existing-conditions]; the domain step is a check of the
  landlord's register. Pre-2004 fabric is an Implication [K: knowledge/clusters/structure/consequences.yaml
  cq.pre-2004-fabric-hazardous-materials-survey].
- **Structure.** None in the base. A slab penetration for a stair, heavy storage or new plant needs a
  structural engineer and the landlord's approval (triggers).
- **Matching existing work.** Base-building ceiling tiles, lights and diffusers are reused or matched;
  removed items are stored and returned to the landlord because they form part of the make-good at lease
  end.
- **Dilapidation.** A condition survey of the tenancy, lift lobby, goods lift and dock before work, so damage
  claims from the landlord can be settled.
- **Make-good at lease expiry** is a future obligation, not this job's scope. A document that says "the
  tenant must make good at the end of the lease" must not add strip-out to this job (§10).

---

## 4. Consultants and why

| discipline | stage engaged | need | built in |
|---|---|---|---|
| Project manager (`consultant.project_manager`) | inception | usually | no |
| Interior designer (`consultant.interior_design`), design lead | inception | always | yes [K: knowledge/works/package_defaults.yaml baselines[4]] |
| Mechanical engineer | concept | usually | yes [K: baselines[3]] |
| Electrical engineer | concept | usually | yes [K: baselines[2]] |
| Fire services engineer | concept | usually | yes [K: baselines[3]] |
| Hydraulic engineer | design | optional (kitchenette only) | yes [K: baselines[2]] |
| BCA consultant | concept | always | yes [K: baselines[3]] |
| Certifier | design (CDC) | always | yes [K: baselines[2]] |
| Access consultant | concept | usually | yes [K: baselines[3]] |
| Quantity surveyor | concept | usually | no |
| Acoustic consultant | design | optional | no |
| ICT, AV and security consultant | design | optional | no |
| Fit-out contractor (`trade.main_works`) | tender | always | partial [K: internal/procurement/proposed.go ProposeHeadContract] |

Notes [D]:

- **One services firm** commonly does mechanical, electrical, hydraulic and fire. The shape aliases each
  services discipline to `consultant.services` so a scenario with one multi-services firm still scores.
- **BCA consultant and certifier are different parties.** NSW certifier conflict rules keep the certifier
  from designing or advising on the design it certifies (Building and Development Certifiers Act 2018
  provisions verified: false).
- **Not expected: the architect.** The app suggests Architect for every refurb [K: baselines[2]] as well as
  Interior Design for commercial refurb [K: baselines[4]]. That is two design leads. Where the design firm
  is an architectural practice it is still one appointment.
- **Not needed in the base:** structural engineer, fire engineer, hazmat, ESD, town planner, heritage,
  surveyor, civil, geotechnical. Each arrives with a trigger.

---

## 5. Approvals and authorities

**Pathway: complying development** (`hdr.cond.planning: cdc`). Internal fit-out of an existing commercial
building is commonly complying development under the State Environmental Planning Policy (Exempt and
Complying Development Codes) 2008, Commercial and Industrial Alterations Code (part number verified: false).
It is not available for a heritage item, a change of use outside the code, or works that fail its
standards; those need a DA (triggers `da-pathway`, `heritage-item`). A minor refresh with no change to
partitions, services or fire safety measures can be exempt (trigger `exempt-minor-refresh`; exempt
standards verified: false). [D]

Sequence [D]:

1. Landlord approvals under the agreement for lease: concept design; later the contractor (insurances,
   SWMS, inductions), permits to work and hot-works permits.
2. BCA and access reports → landlord's detailed design approval by its base-building engineers and the
   landlord's **owner's consent** → CDC lodgement. A CDC application needs the owner's consent, and in a
   leased tower the owner is the landlord (Regulation clause verified: false). The certifier issues the CDC
   with the fire safety schedule listing the existing and altered measures. App: `fr.cdc-application`
   [K: knowledge/program/fragments.yaml fr.cdc-application].
3. Long service levy: 0.25% of the cost of building work of $250,000 or more including GST, paid before
   work starts (CONVENTIONS C4, checked against nsw.gov.au). The type's $0.3M lower bound is above it, so it
   applies in the base. The program keeps it before the determination: the levy must be paid before the
   certificate issues (clause verified: false).
4. Principal certifier appointed and notice of commencement given (period verified: false).
5. Mandatory inspections for a Class 5 building: **commencement and final** (the stormwater item does not
   apply inside a tower); the Class 1 and 10 list in owner ruling R7 does not apply. The Class 5–9 list is
   verified: false, for the owner to confirm as R7 did for Class 1 and 10.
6. Licensed trades' compliance: the electrician's Certificate of Compliance for Electrical Work
   [K: knowledge/clusters/electrical-comms/rules.yaml rule.state.electrical-certificate-of-compliance], the
   plumber's compliance certificate [K: knowledge/clusters/services-wet-air/rules.yaml
   rule.state.licensed-plumbing-and-compliance-certificate].
7. Fire services commissioning and the integrated test with the base-building panel; the **final fire
   safety certificate** for the measures the fit-out altered, before the OC (EP&A (Development
   Certification and Fire Safety) Regulation 2021 clause verified: false).
8. **Occupation certificate** for the altered part before the tenant occupies it. App:
   `fr.occupation-certificate` [K].
9. Landlord's completion inspection; as-builts, manuals, certificates and BMS records to the landlord.

Authorities [D]: the certifier (approval), council (CDC registration; s7.11 or s7.12 contributions the certifier
imposes where the LGA's plan levies them, verified: false), the Long Service Corporation, Fire and Rescue NSW (only where a performance
solution requires the certifier to consult it, verified: false), SafeWork NSW (asbestos trigger only).
No Sydney Water, DNSP or carrier application: there is no new connection.

**Not applicable:** BASIX (residential only; `det.basix_required: false` [K:
knowledge/clusters/envelope/rules.yaml rule.basix.certificate-required]); the Design and Building
Practitioners Act (Class 2 and 9c only, owner ruling R3); HBCF (residential only, owner ruling R4).

---

## 6. Cost plan shape

Names lines only. The app writes no amounts; a stated budget is spread by the `commercial_fitout` split
[K: knowledge/costs/budget_splits.yaml commercial_fitout] (verified: false, confidence low).

**Fees and charges**
- CDC and occupation certificate lodgement fees: the app's line is "Construction certificate and certifier
  fees" [K: knowledge/costs/starter_lines.yaml cc-fees], labelled for a CC this job does not have.
- Development contributions payable on the CDC (s7.11 or s7.12, verified: false): none. The `da-fees`
  starter is dropped for cdc [K: knowledge/costs/starter_lines.yaml da-fees] and nothing replaces it. The
  LGA's contributions plan decides it, so the scenario must name the LGA.
- Long service levy: none. 0.25% of building work of $250,000 or more incl. GST (C4); unconditional in the
  base, whose $0.3M lower bound is above the threshold.
- Landlord's design review and approval fees: none.
- Landlord's building charges during the works (after-hours lifts and dock, security, shutdowns, isolations):
  none.
- **Not expected:** "Service authority connection fees" [K: starter_lines.yaml service-authority] appears on
  every project; a fit-out has no new connection.

**Consultants**: PM (none), interior designer, mechanical, electrical, fire services, hydraulic (optional),
BCA, certifier, access (all from adopted packages [K: knowledge/works/package_defaults.yaml]), QS, acoustic,
ICT/AV (none). **Not expected:** the architect's fee line. The certifier also gets P05 cc-fees, a double
count risk.

**Construction** (all `partial`: each element line appears only once a work item on that system exists, and
refurb has none by default [K: knowledge/costs/elements.yaml])
- Preliminaries, supervision and after-hours premiums, including temporary protection of the goods lift,
  lobbies, corridors and dock: no line; the split spreads preliminaries pro rata.
- Tenancy signage: lands in "01 Site works and external works" [K: elements.yaml code 01], a misleading
  label inside a tower.
- 05 Interiors and finishes (about half the cost per the split), 06 Passive fire, 07 Active fire,
  08 Mechanical, 09 Hydraulic, 10 Electrical, 11 Communications and security.
- 13 Access and egress group: "Accessible paths, TGSIs and braille/tactile signage" and "Egress doors and
  hardware"; exits are not costed [K: elements.yaml code 13].
- Landlord's nominated contractor works (fire services and BMS): none. If a `trade.fire_services` package
  exists, 07 is costed under it [K: knowledge/costs/elements.yaml code 07 trade].
- Provisional sums for above-ceiling conditions and base-building interfaces: none.

**Contingency**: design contingency, construction contingency, escalation, exclusions (always
[K: starter_lines.yaml]); owner supply only from `proj.owner_supply_stated`
[K: internal/store/cost_budget.go applyBriefFigures]. The parser splits the phrasing on commas and "and", so
the key expects four lines: loose furniture, workstations, AV equipment, IT active equipment.

---

## 7. Program shape

**Skeleton:** `sk.refurb-traditional-cdc`: investigation → concept design → construction documentation →
approvals → procurement → construction → completion → defects [K: knowledge/program/skeletons.yaml]. That
order suits a CDC: documents are complete before lodgement.

**Process fragments that fire** (class 5, refurb, cdc) [K: knowledge/program/fragments.yaml]:
`fr.existing-conditions`, `fr.concept-design-fitout`, `fr.design-development-fitout`,
`fr.fitout-documentation`, `fr.cdc-application`, `fr.tender-docs-fitout`, `fr.tender-period-fitout`,
`fr.tender-evaluation-fitout`, `fr.contract-award`, `fr.site-establishment-fitout`, `fr.practical-completion`,
`fr.occupation-certificate`, `fr.handover`, `fr.defects-inspection`, `fr.defects-liability-end`.
`fr.heritage-referral` is false because `det.heritage_status: none` is set.

**Work-item fragments** fire once their item is in scope: partitions, ceilings, doors, joinery, painting,
floors, electrical, lighting, mechanical, ventilation, sanitary, cold, hot, comms, sprinklers, detection
(`*-fitout`) and `fr.fire-stopping`. `fr.occupied-protection` needs `site.impact-protection`, which a
correct fit-out scope does not have.
`fr.strip-out-fitout` needs `site.demolition` with action `remove`.

**Driving sequence** [D]:

1. Agreement for lease and fit-out guide → base-building information → measure, trace services → capacity
   check with the landlord's engineers.
2. Brief → test fit and concept → client review → **landlord concept approval** → design development →
   cost plan.
3. Fit-out and services documentation → BCA and access reports → **landlord detailed design approval and
   owner's consent** → sign-off.
4. CDC lodgement (after the BCA report and the landlord's consent) → LSL → determination.
5. Tender → evaluate → award → landlord contractor approval → principal certifier and notice of
   commencement → **access date**.
6. Commencement inspection and condition survey → establish → shutdowns and isolations booked → partitions
   framed; ductwork, electrical and kitchenette rough-in; **sprinkler pipework above the ceiling** →
   **sprinkler pressure test** → cabling → fire stopping after the services it seals → **fire-stopping
   inspection** (holds only the ceilings and bulkheads that conceal the penetrations) → **ceilings** → sprinkler
   head fit-off, detection and EWIS, lights, emergency lighting → doors, glazing, joinery → paint → floors →
   fit-off → BMS and mechanical commissioning → **fire integrated test** → signage → furniture, AV and IT by
   tenant suppliers.
7. PC → final fire safety certificate → final inspection → OC → landlord completion and as-builts →
   handover → move-in.
8. Defects inspection → defects liability period → final defects inspection.

**App order defects, confirmed by a Go run (30 October 2026, review).** The key states the correct order;
the app's order is recorded in `gaps`:

- **Sprinkler hold point** [K: knowledge/clusters/fire/interfaces.yaml
  if.sprinkler-pressure-test-before-ceilings]. `fr.sprinklers-fitout` attaches after `fr.ceilings-fitout`,
  so the hold-to-ceilings link is dropped. Note `link_skipped`: "Link from Hold point: a recorded pressure
  test of the sprinkler pipework to interiors.ceilings left out: it would make a loop"
  [K: internal/program/generate.go holdPointRows, holdPointLinks]. The library has no concealed-pipework
  row; the app's single sprinkler row is right for fit-off. The key therefore has two rows: pipework above
  the ceiling before the test, and fit-off after the ceilings.
- **Services rough-in after the ceilings.** Six fit-out services fragments attach after
  `fr.strip-out-fitout`; with no strip-out they lose their anchor and the chain step puts sanitary rough-in
  after doors, electrical rough-in after the hot-water unit and ductwork after luminaires
  [K: internal/program/generate.go chainLinks].
- **Fire stopping.** `if.fire-stopping-before-concealment` holds every interiors row (partitions, off-site
  joinery, doors, paint, floors), and `fr.fire-stopping` has no attach, so sealing starts at construction
  start [K: knowledge/clusters/fire/interfaces.yaml if.fire-stopping-before-concealment].

**Long-lead items** [D]: joinery, glazed partition suites, imported furniture and workstations, AV
equipment, and supplementary AC units where a comms room needs them. Durations: the per-unit fit-out rows
read `plan.gross_floor_area`, which comes from `hdr.scale.gfa_sqm` [K: internal/store/program_prepare.go
profileFloorArea]. The office subclass records `nla_sqm`, so every fit-out is sized at the 250 m² default
[K: knowledge/program/durations.yaml assumptions].

The full activity list with links is in `shape.yaml` (71 activities).

---

## 8. RFP shape

The RFP expectations in `shape.yaml` use the answer-key keys exactly. Recurring app differences: the
services templates' default boundary ends at `design`, so `tender`, `construction` and `handover` are
priced as options [K: knowledge/rfp/mechanical-services.yaml boundary; internal/rfp/resolve.go
resolveStages]; most rows hang on client flags that cannot be keyed and so ask; an `existing` row whose
investigation stage is `ask` is left out, because the `stage` key needs include or option
[K: internal/rfp/resolve.go stage].

- **Interior designer** (no template; generic capex [K: testing/synthetic/reference/index/rfp-templates.md]):
  brief and test fit, concept, landlord approval submissions, design development with finishes and FF&E,
  documentation and specification, CDC drawings, tender documents and evaluation input,
  construction-phase support. Not: base-building, facade, services design, a DA, structure.
- **Project manager** (no template): brief, budget, program, consultant procurement, landlord liaison,
  tender and contract administration, cost reporting, tenant-direct suppliers, OC and handover.
- **Mechanical** (`mechanical-services`): investigation of base-building capacity, HVAC design for new zones,
  local exhaust, BMS points, interfaces, tender, construction inspections, witness commissioning. Not
  commercial kitchen exhaust (the row's own label says "not kitchenette"), smoke control, planning,
  seasonal commissioning, declarations (no NSW declarations for Class 5, owner ruling R3).
- **Electrical** (`electrical-services`): investigation of the tenant DB, distribution, lighting and
  controls, emergency lighting, passive ICT pathways, tenancy metering, tender, construction, witness. Not
  utility application (the app includes it), plant power (app includes it), generation, detection (allocated
  to fire services), studies.
- **Construction and witness stages** stay included for mechanical, electrical and fire services: on a
  traditional, fully documented fit-out the landlord and certifier need the tenant's engineers to inspect
  and certify the installation (review ruling). The scenario's consultant brief must say so.
- **Fire services** (`fire-services`): existing layout and FER review, sprinkler and detection/EWIS
  alterations, cause-and-effect, isolation and restoration while occupied, tender, construction, witness.
  Not water supply, pumps, authority connection, upgrade, strategy (fire engineer).
- **Hydraulic** (`hydraulic-services`, optional): kitchenette water and waste, investigation of tenant
  points. Not pumps, rainwater, roof (the app includes all three through `if_unknown: include`), trade
  waste, gas, site stormwater [K: data/eval/rfp/answer-keys/hydraulic-office-fitout.yaml].
- **BCA consultant** (`bca-certification`, Package A): inception and detailed-design advice (a3 stands for
  the BCA report behind the CDC), fit-out assessment of the affected area, option of construction advice.
  Not statutory functions, design or supervision. **The template blocks on cdc for the whole template**,
  advisory included [K: knowledge/rfp/bca-certification.yaml unsupported_pathways]; the shape expects no
  block and records the difference.
- **Certifier**: CDC eligibility and determination with the fire safety schedule, principal certifier,
  final inspection, final fire safety certificate check, OC. The template blocks on cdc and has no CDC
  stage. An agent-draft key, `bca-unsupported-cdc` (a new Class 5 CDC, `reviewed: false`), treats the block
  as the safe answer until a CDC scope exists; for the owner to confirm
  [K: data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml]. Same treatment as T12.
- **Access** (`access`): existing work on affected routes, shared lifts and amenities, hearing augmentation
  and signage, construction approval report for the CDC, occupied-route continuity
  [K: data/eval/rfp/answer-keys/access-live-fitout.yaml as a pattern]. Not housing, planning report.
- **QS** (`cost-planning`): basis, concept, design cost plan, pre-tender estimate and tender evaluation;
  total project budget including FF&E, IT, AV, relocation and landlord charges; strip-out, make-good,
  after-hours and isolation costs (`existing_costs` is on for refurb [K: knowledge/rfp/cost-planning.yaml
  existing_costs]). Not authority estimate, lender reports, remediation.
- **Acoustic** (no template, optional): criteria, ratings in documentation, services noise, option of
  testing. Not facade acoustics or a DA noise assessment.

---

## 9. Triggers

Each is in `shape.yaml` with evidence, negation and what it adds or removes.

| id | how it shows up | how a document denies it | adds |
|---|---|---|---|
| `da-pathway` | "a development application will be lodged with council for the fit-out"; "change of use, so complying development is not available" | "the certifier will issue a CDC" | Town planner, DA fees, DA lodgement, exhibition, assessment, CC; skeleton `sk.refurb-traditional-da`; certifier RFP stops blocking; the consent may require a fire safety upgrade of the existing building (verified: false), so the BCA consultant assesses it. Removes the CDC rows |
| `heritage-item` | "listed as a local heritage item"; "the original lift lobby and pressed-metal ceilings are significant fabric" | "not a heritage item"; "the heritage item next door is not affected" | Heritage consultant (no template), heritage impact statement and referral, DA. Removes the CDC rows and the CDC contributions line (contributions move into the DA fees line) |
| `design-and-construct` | "D&C fit-out contract on the interior designer's concept"; "the contractor's engineers will design the services" | "traditional lump-sum tender on fully documented design" | Performance brief, contractor's design and landlord approval, design review. Removes the four services consultants; C1 does not apply, as no tenant engineer is appointed to be novated (whether a tenant keeps a services reviewer on a D&C fit-out is an owner question). App wrongly adds novation and falls back to `sk.generic` |
| `staged-occupied-tenancy` | "staff remain in occupation; the works are staged in three zones"; "decant into swing space on Level 11" | "the floor is vacant for the whole fit-out period" | Staging, decant, after-hours premium, interim OCs; `flag.staged` |
| `asbestos-in-tenancy` | "the asbestos register records bonded vinyl floor tiles under the carpet"; "asbestos-containing material in the ceiling space around the core" (a 1979 completion date alone gives only the register check and survey) | "completed in 2008, no register required"; "no asbestos in the tenancy" | Hazmat consultant, survey, SafeWork NSW notice, licensed removal, clearance certificate |
| `existing-fitout-strip-out` | "the previous tenant's fit-out will be stripped out"; "remove the existing partitions, ceilings, kitchen and carpet" | "handed over in base-building condition"; "make-good completed by the outgoing tenant" | `site.demolition` remove, strip-out rows, strip-out cost line |
| `internal-stair` | "a new internal stair will connect Levels 11 and 12"; "slab penetration for the communicating stair" | "the tenancy is on one floor" | Structural and fire engineers, slab penetration, stair and balustrades, compartmentation |
| `slab-core-drilling` | "new floor boxes cored through the post-tensioned slab"; "core holes for the kitchenette waste"; "scan the slab before coring" | "power is from power poles"; "the existing floor boxes are reused" | Structural engineer, slab scanning and landlord's structural approval before rough-in, core drilling |
| `heavy-floor-loads` | "mobile compactus units in the records room"; "storage loads of 5 kPa" | "records are digital"; "loads within the base-building office allowance" | Structural floor-load assessment and landlord approval, strengthening |
| `fire-engineered-solution` | "travel distance exceeds the deemed-to-satisfy limit; a performance solution is proposed"; "the base-building FER must be updated" | "the layout complies with the DTS provisions" | Fire engineer, FEB, FER, FRNSW consultation |
| `comms-room-cooling` | "24/7 split-system cooling in the comms room"; "connect to the tenant condenser water loop"; "UPS" | "IT is cloud-hosted; a small comms cupboard only" | Refrigerant pipework, UPS, landlord approval of condensers |
| `tenant-amenities` | "new unisex accessible toilet and shower within the tenancy"; "end-of-trip showers in the tenancy" | "amenities are base-building on the core" | Wet areas, accessible facilities, hydraulic design; note "shower" and "accessible toilet" do not match the wet-room implied pattern |
| `sustainability-rating` | "targets a 5 Star Green Star Interiors rating"; "NABERS Energy tenancy rating under the green lease" | "no rating targeted"; "the base building's NABERS rating" (the landlord's, not the tenant's) | ESD consultant, rating registration and submission, sub-metering |
| `exempt-minor-refresh` | "exempt development: new carpet, paint and loose furniture only" | "a CDC is required"; "new partitions affect sprinkler coverage" | Exempt confirmation. Removes the certifier, CDC rows, principal certifier appointment and notice, commencement and final inspections, final fire safety certificate, OC, and the CDC contributions line. LSL still applies at $250,000 or more (C4); a carpet-and-paint refresh may fall below it |

Variants worth a one-change test: `da-pathway`, `design-and-construct`, `staged-occupied-tenancy`,
`asbestos-in-tenancy`, `existing-fitout-strip-out`, `internal-stair`, `fire-engineered-solution`,
`exempt-minor-refresh`, `slab-core-drilling`.

---

## 10. Keyword traps and negations specific to this type

- **"DA" for the base building.** "DA 2005/1234 approved the tower" or "the building's original
  occupation certificate" is the base building's history, not this job's pathway. It must not set
  `hdr.cond.planning: da`.
- **Storeys of the building.** "Level 12 of a 24-storey tower" gives the building's storeys
  (`hdr.scale.storeys`, `det.rise_in_storeys`), not the extent of the works. Nothing in the fit-out split
  reads storeys today, but a reader that treats it as the works' extent is wrong.
- **NLA, not GFA.** Fit-out documents state NLA. Durations read GFA, so the area never reaches the program
  (§7).
- **"Fire stair".** "Egress via the base-building fire stairs" matches the `stair` implied pattern and adds
  `access-egress.stairs-and-ramps` and a stairs cost line [K: knowledge/profile/scope_defaults.yaml
  implied_by stair]. Nothing in a single-floor fit-out builds a stair.
- **"Bathrooms" on the core.** "Base-building bathrooms on the core are not in scope" matches the wet-room
  pattern and adds waterproofing, tiling, sanitary, hot and cold water and exhaust [K: implied_by wet-room];
  implied readings have no negation [K: internal/profile/implied.go ImpliedReadings].
- **"Kitchenette" versus "kitchen".** `\bkitchens?\b` does not match "kitchenette", so the kitchenette's
  joinery and hydraulic systems are not implied; "staff kitchen" is. A kitchenette is not a commercial
  kitchen: it must not set `det.commercial_kitchen` or a kitchen-exhaust scope (the mechanical row
  `kitchen` says "not kitchenette").
- **"No strip-out" / "no demolition".** Matches the strip-out pattern and adds `site.demolition`.
- **Make-good at lease end.** "The tenant must make good at expiry" is a future lease obligation, not this
  job's strip-out.
- **Landlord's works versus tenant's works.** Agreements for lease list "landlord's works" (base-building
  items the landlord delivers) next to the tenant's. Those systems are `existing` or landlord-provided, not
  the tenant's scope. A document describing base-building VAV boxes, sprinklers or lighting must not become
  scope; refurb's `existingPresence` handles a plain description [K: internal/profile/existing.go].
- **The base building's ratings.** "The building has a 5.5 star NABERS Energy base-building rating" is not a
  tenant sustainability target (`flag.sustainability_target`).
- **Heritage next door.** "Opposite a heritage item" or "adjoining the heritage-listed bank" must not set
  `det.heritage_status`.
- **"Common wall".** "The common wall with the neighbouring tenancy" matches the party-wall pattern and adds
  `site.adjoining-structures`, compartments and acoustic separation; "intertenancy wall" does not match.
- **"BASIX not applicable".** Matches the energy-rating pattern and adds `envelope.thermal-performance`.
- **"Sprinklered".** "The building is sprinklered throughout" is a fact about the base building
  (`det.sprinklered: true`), not sprinkler works. Sprinkler works are "relocate heads", "additional heads".
- **"Asbestos-free".** "The asbestos register records no asbestos in the tenancy" is a negation for
  `asbestos-in-tenancy`.
- **Annual fire safety statement.** The building's annual statement is the landlord's; the fit-out needs a
  final fire safety certificate for the measures it altered.

---

## 11. Gaps in the built-in knowledge

What the app will probably produce wrongly for this job, in order of effect:

1. **Empty scope.** Refurb has no defaults, and the fit-out preset is never applied, so the program has no
   construction activities, the cost plan no construction lines and the procurement no scope-driven
   consultants until documents are read [K: knowledge/profile/scope_defaults.yaml empty_work_types,
   presets].
2. **Landlord missing.** No profile keys, program rows, cost lines or packages for the agreement for lease,
   landlord approvals, access date, building charges or nominated contractors.
3. **Certification.** Both BCA and certifier RFPs block on CDC; the template has no CDC scope
   [K: knowledge/rfp/bca-certification.yaml]. No principal certifier appointment, final inspection, final
   fire safety certificate or LSL rows.
4. **Design lead.** Architect and Interior Design are both suggested; Interior Design has no template, so the
   lead consultant gets the generic capex RFP; no PM or QS suggested.
5. **Occupied-building work.** Occupied protection needs `site.impact-protection` (vehicle and forklift
   impact protection, the wrong system); strip-out needs `remove`; no rows for shutdowns, isolations,
   after-hours or the fire integrated test.
6. **Services rough-in after the ceilings** (confirmed by a run). With no strip-out, the fit-out services
   fragments lose their anchor and the chain step puts sanitary, electrical and mechanical rough-in after
   doors, the hot-water unit and luminaires.
7. **Fire stopping at construction start, holding every interiors row** (confirmed by a run), including
   partitions and off-site joinery.
8. **Sprinkler hold point contradicts the fragment order** (confirmed by a run: the hold-to-ceilings link
   is skipped as a loop). The library has no concealed-pipework row; detection has the same two-phase gap
   without a hold point.
9. **Services RFPs** price tender, construction and witness as options and include utility and plant rows.
10. **Durations** ignore the stated NLA.
11. **Cost plan** carries service authority fees, no CDC contributions, LSL, landlord charges or
    preliminaries; signage lands in "Site works and external works".
12. **D&C refurb** falls back to `sk.generic` with novation; **exempt** still gets an OC row and a certifier.
13. **No structural engineer** follows from structure alterations or core holes on refurb.
14. **Performance solutions** cannot be keyed, so the fire engineer never follows.
15. **implied_by traps** (§10).

The `gaps` list in `shape.yaml` gives the evidence for each.

---

## 12. Sources and unverified claims

**Knowledge opened for this map:** `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and
`index/type-predictions.md` (T03 row), `index/program-fragments.md`, `index/program-skeletons.md`,
`index/cost-plan.md`, `index/scope-defaults.md`, `index/systems.md`, `index/rfp-templates.md`;
`knowledge/works/package_defaults.yaml`; `knowledge/costs/starter_lines.yaml`, `elements.yaml`;
`knowledge/clusters/fire/interfaces.yaml`; `knowledge/clusters/services-wet-air/interfaces.yaml`,
`unforeseen.yaml`; the rules named above; `knowledge/determinants.yaml`; `knowledge/profile/taxonomy.yaml`;
`internal/rfp/drivers.go`, `internal/rfp/resolve.go` (`resolveStages`, the `stage` key);
`internal/program/generate.go` (`holdPointRows`, `holdPointLinks`); `internal/store/program_prepare.go`
(`profileFloorArea`); `data/eval/rfp/answer-keys/` (`bca-unsupported-cdc`, `hydraulic-office-fitout`,
`access-live-fitout`, `cost-live-fitout`, `bca-refurb`, `mechanical-office-design`,
`electrical-office-design`, `fire-services-office-design`); `data/intake/disciplines.json`.

**Unverified (verified: false):**

- The Codes SEPP part for commercial alterations, its exempt standards, and the exclusion of heritage items
  from complying development.
- NCC 2022 Section J part numbers for lighting and air-conditioning; AS 2118.1, AS 1670.1, AS 1670.4, AS 2293.1
  and AS 1428.1 clause references.
- The Class 5–9 mandatory inspection list and the final fire safety certificate clause in the EP&A
  (Development Certification and Fire Safety) Regulation 2021; FRNSW consultation on performance solutions.
- s7.11 or s7.12 contributions imposed on CDCs; the LSL payment-before-certificate clause; the notice-of-commencement
  period; the owner's-consent requirement for a CDC application; a DA's power to require a fire safety
  upgrade of the existing building.
- The WHS Regulation asbestos register date (31 December 2003) and the SafeWork NSW notice period.
- The Building and Development Certifiers Act 2018 conflict provisions.
- Defects liability period length.

**Claims I am less sure of:**

- Base services consultant stages (construction and witness included): ruled right by the review; some
  tenants still buy only design to tender.
- The long service levy threshold: settled by C4 (0.25% at $250,000 or more incl. GST).

The program order defects (sprinkler hold point, services rough-in, fire stopping) were confirmed by the
review's read-only Go run on 30 October 2026.

## Shape corrections after run `20261107-t03-t04-t06` (7 November 2026)

From FINDINGS "Shape and key gaps" for that run, and owner ruling R19.

- Certifier RFP re-keyed for F9, as the T12 certifier was on 6 November. The old key expected
  `blocking: [pathway_unsupported]` and every stage absent. The template now has `b2_cdc`, the CDC determination
  in place of the CC. The key expects `b2_cdc`, `b3_construction` and `b4_occupation` included, `b1_preliminary`
  option (as T11), `b2_construction_approval` absent and no blocking check. Its CDC, principal certifier,
  inspection and OC phrases are keyed to those stages. The `text.cdc_scope` gap is removed.
- BCA RFP: the `blocking.pathway_unsupported` difference is gone (cdc is no longer an unsupported pathway). The
  Package B stages (`b1_preliminary`, `b2_cdc`, `b3_construction`, `b4_occupation`) stay absent, with `app_differs`
  option: the app shows them after the boundary because the client flags cannot be set.
- `da-pathway` sets `skeleton: sk.refurb-traditional-da`. It restates the certifier `expect` (`a2_planning` and
  `b2_construction_approval` included, `b2_cdc` absent) and the BCA `expect` (`a2_planning` included). The BCA's
  `b2_construction_approval` stays absent with `app_differs` option, as the base keys the other B stages: the CC
  is the certifier's. The CDC phrases leave the certifier and BCA lists.
- `hdr.cond.quality_level` applies to residential only (`taxonomy.yaml` `applies_to`), so the ST03 keys mark it
  `not_read`. The shape keeps the value as a fact.
- R19: the ST03 keys accept `partial_occupation` or `live_environment` for `hdr.cond.operational_constraints`.
