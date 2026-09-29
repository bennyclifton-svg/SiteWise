# Mechanical and hydraulic extraction

Full relevant-seed reading is complete for the source set below. The cluster now contains 20 systems, 54 rules, 16 interfaces and 23 failure modes. Existing detailed contributions and merged edges were preserved.

The follow-up added four project-specific rules for functional/seasonal commissioning, specialist operator acceptance, occupied services cutover and sewage pump-out operation. Three physical edges cover tower water/treatment/drainage, vendor process utility boundaries and sewage pump supply/alarm. Seven failure detectors cover these and clinical verification, occupied-stage service isolation, start-up substituted for testing and air-conditioning substituted for outdoor air.

## Full-file source coverage

The continuation read the following files end-to-end. Large files were read in bounded chunks, and truncated output gaps were reread. Full reading is a discovery sweep, not primary verification of seed assertions.

| Files read in full | Technical coverage and disposition |
|---|---|
| `fire-life-safety-guide.md`; `access-consultant-guide.md` | Fire strategy, passive/active systems, egress, access, impairments, integrated testing and access audits |
| `ncc-reference-guide.md`; `as-standards-reference.md` | Entire references, including material outside this cluster; applicable technical content extracted or linked to canonical records, numeric prose retained only as unverified claims |
| `mechanical-services-guide.md`; `hydraulic-services-guide.md`; `mep-residential.md` | HVAC, ventilation, fire mode, water, drainage, gas, scope boundaries, testing and existing-building services |
| `trade-interfaces-coordination-guide.md` | Concealment, penetration, membrane, structure and services interfaces; merged canonical edges retained |
| `non-residential-sustainability-energy-guide.md`; `sustainability-energy-guide.md` | Services energy and certificate commitments; envelope retains canonical whole-of-home energy rule |
| `commercial-construction-guide.md`; `small-commercial-guide.md` | Base-building/tenant service boundaries, altered use, egress, HVAC, hydraulic, BMS and fit-out interfaces |
| `residential-construction-guide.md`; `multi-residential-apartments-guide.md` | House/apartment fire and acoustic separation, roof termination, risers, metering, ventilation, livable housing, commissioning and repeated details |
| `new-dwelling-guide.md`; `multi-dwelling-guide.md`; `ancillary-guide.md`; `renovation-guide.md` | Legacy archetype technical facts, retained to catch details absent in successors; classification assumptions, party walls, shared services, staged occupation and live cutovers |
| `industrial-construction-guide.md`; `institution-construction-guide.md`; `mixed-use-construction-guide.md`; `infrastructure-construction-guide.md` | Actual process/clinical/operator design basis, shared systems, emergency/degraded operation and staged acceptance; network assets are not assigned generic building NCC obligations |
| `building-remediation-rectification-guide.md` | Investigation before repair, physical evidence, temporary protection, impairments, occupied work and verification after rectification |
| `civil-residential.md` | Building-to-site drainage, sewer, pump-out failure alarms and authority connection boundaries |
| `defects-and-dlp-guide.md` | Technical defect symptoms and concealed-work interfaces; legal liability and DLP machinery excluded |
| `electrical-services-guide.md`; `ict-av-security-guide.md` | Essential power, controls, fire/security release and system commissioning boundaries; canonical electrical/comms ownership retained |

### Intentionally excluded material

PM doctrine, role workflows, fee/RFP prose, generic contract administration, registers, durations, cost rates, procurement process and legal warranty advice were not converted into physical-building rules. Technical requirements embedded in those sections were still considered. No source instructions were adopted as agent instructions.

`setup-and-commission-guide.md` was inspected at its introduction and complete heading list: “commission” means project mobilisation between award and site start, not technical services commissioning. Its role/setup machinery is outside this extraction; technical commissioning is covered by the full discipline and archetype reads above.

`remediation-due-diligence-guide.md` was inspected at its introduction and complete heading list. It explicitly covers contaminated land and directs building/fire/facade rectification to `building-remediation-rectification-guide.md`, which was read fully. Detailed contaminated-land methods remain with site scope. These two files are intentionally scoped exclusions, not claimed full reads. Other disciplines' detailed architecture, finishes, structure, site, acoustic, facade and vertical-transport seeds remain with their owning clusters; this report does not claim an independent full reread of them.

## Contradictions and suspect seed claims

- Toilet exhaust rates differ between NCC and apartment seeds, including per-fixture/per-compartment wording. All retained numbers remain unverified.
- NCC energy mapping is inconsistent: old J5 labels are variously called hot water and HVAC, while the mechanical guide signposts NCC 2022 J6. Seed locators do not prove adopted clauses.
- The AS 1668 seed blurs normal car-park ventilation and fire-mode smoke operation and gives generic air/filtration rates. Their scope and values need the adopted standard.
- Apartment prose lists split air-conditioning as a mechanical ventilation option; cooling/recirculating air alone does not establish an outdoor-air path. An explicit failure detector now distinguishes this error.
- Cooling-tower intake-to-intake wording, blanket registration/audit statements, refrigerant classification/charge examples and pressure-equipment thresholds require primary verification.
- Domestic gas Type B threshold conflicts with the detailed reference. No automatic category calculation is derived from either.
- Fixture ratios, trap seals, gradients, temperatures, roof overflows, backflow device selection and WaterMark product scope require authoritative tables and exceptions. Civil seed pool backwash-to-stormwater wording is suspect and was not promoted.
- Energy seeds conflict over national/state NatHERS minima, BASIX completion process and applying Section J to Class 1. Canonical envelope energy rules remain the owner of that reconciliation; this cluster preserves certificate-specific service commitments, not seed-wide compliance conclusions.

## Determinants, unresolved IDs and pending tables

`sanitary_fixture_counts` remains the extracted stated/provided count. The pending sanitary provision rule now gives derived `required_sanitary_fixture_count`, a separate minimum for a scoped fixture category and occupancy cohort. Both values need physical scope and provenance; one integer cannot represent a mixed fixture schedule. The derived determinant has been folded into knowledge/determinants.yaml; the proposal file is empty. Existing commercial-kitchen, trade-waste-use, occupant-load and provided-count proposals were already merged into global determinants and were not recreated.

`sanitary_facility_ratios` remains explicitly pending: full applicability inputs, occupancy categories and authoritative rows are unavailable, so code must return unknown. Remaining primary data includes outdoor-air/exhaust rates, energy efficiencies, duct criteria, demand/pressure loss, fixture-unit sizing, grades, scald/microbial limits, rainfall/overflow design, refrigerant limits and authority trade-waste conditions.

Owned-folder validation passes with no unresolved IDs. Merged aliases were read; hydraulic rough-in, waste/membrane and ceiling-space edges remain under their canonical envelope IDs. Whole-of-home energy remains `rule.ncc.whole-of-home-energy` in envelope. Fire-water systems remain fire-active-owned.

No incomplete reads remain within the listed full-file source set. Residual source depth is explicit: seeds flag specialist medical/laboratory/process requirements but do not supply complete technical acceptance standards, gas identity/purity protocols, infection-control criteria, refrigeration hazard calculations, treatment/discharge limits, vendor loads, operator test scripts or project cutover boundaries. Those are project/primary-source requirements; generic comfort HVAC or domestic water rules cannot fill them.

## Cross-cluster edges

- `if.exhaust-penetrates-envelope`: `mechanical.local-exhaust` → `envelope`.
- `if.plant-loads-structure`: `mechanical.central-plant`, `mechanical.air-conditioning`, `hydraulic.hot-water` → `structure`.
- `if.mechanical-power-supply`: `electrical` → `mechanical.air-conditioning`, `mechanical.central-plant`, `mechanical.controls-bms`.
- `if.plant-noise-vibration`: `mechanical.air-conditioning`, `mechanical.central-plant` → `interiors`, `envelope`.
- `if.roof-drainage-waterproofing`: `hydraulic.roof-drainage` → `envelope`.
- `if.building-drainage-to-site`: `hydraulic.roof-drainage` → `site`.
- `if.hot-water-energy-source`: `electrical`, `hydraulic.gas` → `hydraulic.hot-water`.
- `if.bms-fire-mode-priority`: `fire-active.detection` → `mechanical.controls-bms`.
- `if.services-plant-maintenance-access`: `mechanical.central-plant`, `mechanical.cooling-towers`, `hydraulic.trade-waste` → `access-egress.maintenance-access`.
- `if.process-equipment-services-demarcation`: `mechanical.process-specialist` → `hydraulic.specialist-process`, `electrical`.
- `if.sewage-pump-power-and-alarm`: `electrical` → `hydraulic.sanitary`.

## Question design and verification boundary

Passage questions name `text`, ask one literal judgement, and use narrow system routing. Positive defect detectors require an explicit failure statement; silence and a merely planned test are not physical defects. Resolution questions identify documentary evidence, not a compliance certificate issued by Jev. Code handles counts, dimensions, applicability, scope, conflicts and acceptance thresholds.

Primary TypeSafe pages read for question authoring:

- [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13): literal atomic questions, explicit state fields, no arithmetic, narrow context.
- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): same-state questions run together; code resolves relevance and combines results.
- [Confidence](https://docs.typesafe.ai/confidence): noul probability is distinct from Choice confidence; calibration is per question.

No live Jev evaluation or threshold calibration was performed. Seed-derived records remain `status: draft`. New project-specific acceptance rules identify their project design basis rather than inventing a statutory clause. Unverified editions, clauses and numeric seed claims remain unverified. Existing primary-backed lead corrections and tables were preserved.

## Validation and ownership

`python tools/check_knowledge.py --only knowledge/clusters/services-wet-air --strict` passes with zero errors and zero warnings. No git writes were made. Parent-owned global determinants, primary tables and verification reports were not edited. Only current owned files were loaded for append/update; merged lead corrections were preserved.
