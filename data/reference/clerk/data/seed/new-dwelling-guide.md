---
tier: archetype
seed_type: archetype
loaded_by: "archetype: new-dwelling"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_classes: [residential]
applies_to_work_types: [new]
state_default: NSW
status: legacy-retained
superseded_by: seed/residential-construction-guide.md
summary: "Archetype coverage for a standalone Class 1a dwelling on a vacant or cleared site, including knockdown-rebuild: the typical NSW lifecycle sequence, site due diligence, planning pathway, BASIX baseline, structural posture and inspection gates, and new-dwelling sequencing risks."
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline]
agents_anchors: [§1, §2, §3, §8]
---

# Archetype seed — New dwelling

A **new dwelling** is a standalone Class 1a residential building constructed on a vacant or cleared site (or as a knockdown-rebuild). One dwelling per title in this archetype; subdivisions of two or more attached dwellings load `multi-dwelling-guide.md` instead. A new dwelling with a granny flat addition loads `ancillary-guide.md` task-loaded for the granny flat scope per AGENTS.md §7.

This seed is loaded when the project `README.md` declares `archetype: new-dwelling`. It is the **domain coverage** for what a new Class 1a residential build involves — sequencing, inspections, BASIX baseline, structural posture, sequencing risk. It is not role-specific; the role overlay (`role-owner-builder.md` / `role-architect-pm.md` / `role-builder.md` / `role-d-and-c.md`) layers role-divergent content on top.

NSW is the deep default. Non-NSW callouts appear inline where the instrument or pathway differs materially.

## Scope of this archetype

This seed covers:
- detached Class 1a dwellings (the dominant residential build type);
- knockdown-rebuild of an existing Class 1a where the new dwelling is the work;
- secondary dwellings (granny flats) **only when** the granny flat is the primary scope — for granny flats as additions to an existing primary dwelling, load `ancillary-guide.md`.

This seed does not cover:
- renovations or additions to an existing dwelling (load `renovation-guide.md`);
- two or more attached dwellings on one title (load `multi-dwelling-guide.md`);
- pure ancillary structures (load `ancillary-guide.md`);
- Class 2+ residential or mixed-use (out of scope for SiteWise — use the commercial harness).

## NCC class context

A new dwelling sits under **NCC Volume Two** (Housing Provisions) for Class 1a buildings. NCC 2022 is the current edition in NSW (adopted via the EP&A Regulation). Key Class 1a posture:

- Performance-based code with Deemed-to-Satisfy (DTS) provisions in Volume Two.
- Structural design typically by AS 2870 (slabs and footings on reactive sites), AS 1684 (timber framing) or AS 4100 / AS 3600 (steel / concrete where used), AS 4055 (wind), AS 1170 (general actions), AS 3959 (bushfire BAL-rated construction).
- Plumbing under AS 3500 series and the NSW Plumbing and Drainage Act.
- Energy efficiency: Section J does **not** apply to Class 1; the NSW pathway is BASIX (Building Sustainability Index) — see BASIX section below.

NCC Class 1a focus means: single-family residential, attached or detached, not exceeding three storeys in most cases, with the kitchen serving the household.

For NCC reference detail, load `ncc-reference-guide.md` task-loaded. For specific standards (AS 2870, AS 1684, etc.), load `as-standards-reference.md` task-loaded.

## The new-dwelling lifecycle — typical NSW sequence

A NSW new Class 1a dwelling typically follows this sequence. Durations are residential-typical; project-specific durations live in the master programme.

| Phase | Typical NSW duration | Key gate |
|---|---|---|
| Concept and site due diligence | 2–6 weeks | Site survey, dilapidation, soil report, BAL/flood/bushfire/heritage check, planning pathway test |
| Schematic / DA design | 4–12 weeks | Scheme endorsed against brief and budget |
| DA lodgement (if DA pathway) | 8–24 weeks council assessment | DA issued with conditions of consent |
| CDC lodgement (if CDC pathway) | 2–6 weeks | CDC issued (faster but tighter compliance envelope) |
| CC documentation and lodgement | 4–12 weeks | CC issued (LSL Receipt is a prerequisite) |
| Builder procurement | 4–12 weeks (formal) or 2–4 weeks (informal) | Builder selected; head contract executed; HBCF issued |
| Pre-construction mobilisation | 2–4 weeks | Site possession, management plans, baselined programme |
| Site preparation and earthworks | 1–4 weeks | Site cleared, set-out verified |
| Substructure (slab or footings) | 1–3 weeks | Slab pour (inspection gate) |
| Frame | 3–8 weeks | Frame inspection sign-off |
| Roof and lockup | 2–6 weeks | External envelope sealed; windows in; external doors lockable |
| Fixing (plasterboard, joinery, tiling, services rough-in completion) | 4–10 weeks | Internal lining and trades complete |
| Finishing and PC (paint, floor finishes, final trades, commissioning) | 3–6 weeks | PC walk; defects identified; OC applied |
| OC, BASIX final, handover | 2–6 weeks | OC issued; HBCF certificate to owner; HOW handover |
| DLP | 13 weeks (HIA Lump Sum default) or 26 weeks (often varied) | Defects close-out |

Typical total: **10–14 months** site duration for a single-storey 200 m² Class 1a on a benign site; **12–18 months** for a two-storey complex site. Pre-construction (design, planning, procurement) can add another 9–18 months. The owner often experiences project duration as the full 18–30 months end to end; the builder experiences it as the site duration.

Residential cycle-time deeper benchmarks (e.g. days per square of frame, days per metre of brick) sit in `program-scheduling-guide.md`. Slice-of-time risks (weather, supplier lead time, BPA scheduling) sit there too. This section is the **typical sequence**, not the detailed planning.

## Site due diligence — what a new dwelling triggers

Before scheme design locks, the project lead must establish:

- **Survey** — registered land surveyor: title boundary, levels, existing services, easements (sewer, drainage, electricity, telecommunications), street trees, neighbour wall positions.
- **Dilapidation** — adjoining property condition at survey date, photographic record. Sets the baseline against which any construction-period damage claim is judged. Filed under `03-design/01-due-diligence/`.
- **Soil report (geotechnical)** — slab classification per AS 2870 (M / H1 / H2 / E / P / S — increasing reactivity); bearing capacity for footings; presence of fill or rock. Drives footing design and slab type. A reactive site (H1 / H2) often requires a stiffened slab or piered footings.
- **Stormwater and drainage** — site grading, neighbour falls, council drainage point. On sites with no street kerb-and-gutter discharge available, on-site detention (OSD) is required. Sydney Water sewer alignment from the diagram check.
- **Sewer alignment (NSW)** — Sydney Water sewer diagram identifies sewer mains crossing or adjacent to the lot. Building over or near a sewer requires Sydney Water Build Over Sewer (BOS) approval; non-trivial geometry can require a deviation or Section 73 amendment.
- **Bushfire (BAL) assessment** — required where the site is within a bushfire-prone area mapped by council. Outcome is a BAL rating (BAL-LOW, 12.5, 19, 29, 40, FZ); BAL-12.5 and above triggers AS 3959 construction requirements (sealed envelope, ember-resistant detailing, restricted materials).
- **Flood assessment** — if mapped in a flood study, floor levels may be controlled by the FPL (flood planning level), typically the 1% AEP + 500 mm freeboard. Affects slab type and finished floor level.
- **Heritage / character** — heritage item, heritage conservation area, or character area triggers heritage consent requirements. Even where the dwelling itself is not heritage-listed, a neighbouring listing or HCA mapping can impose constraints.
- **Aboriginal heritage** — AHIMS (Aboriginal Heritage Information Management System) search. PACHCI process (NSW) for sites with known sensitivity.
- **Contamination** — for knockdown-rebuild on land with historical industrial or fill use, Phase 1 contamination assessment may be triggered by the planning pathway.
- **Native vegetation / trees** — biodiversity offsets, tree removal permits, neighbour tree TPO checks.
- **Services availability** — water, sewer, electricity, gas (where available), NBN. Confirm capacity for new dwelling load; some sites require service upgrades (e.g. mains gas not present, three-phase electricity needed for HW or appliance load).

Failure to establish any of the above before scheme lock is a common source of late-stage scope and cost movement. The project lead must do the due diligence work before design proceeds far enough that change becomes expensive.

## Planning pathway

NSW has three main planning pathways for a new Class 1a dwelling:

| Pathway | When applies | Approval | Builder mobilisation |
|---|---|---|---|
| **Exempt development** | Genuinely minor work meeting the SEPP exemptions — rarely covers a new dwelling | No approval required | Builder mobilises against CDC-equivalent compliance documentation |
| **Complying Development Certificate (CDC)** under SEPP (Exempt and Complying Development Codes) | Sites and designs meeting CDC envelope, height, setback, FSR, and other complying criteria | CDC issued by certifier (private or council); faster than DA but tighter envelope | Builder mobilises on CDC + CC equivalent (CDC subsumes CC for CDC pathway) |
| **Development Application (DA)** | Sites or designs outside CDC envelope, or where heritage / flood / bushfire / specific overlay applies | DA issued by council (8–24 weeks typical), conditions of consent attached. Then CC issued by certifier before construction starts | Builder mobilises on CC; conditions of consent flow into delivery |

For a project lead, the planning pathway choice drives:
- **Programme** — CDC is faster but if the design must change to fit CDC envelope, the time saved can be lost to redesign.
- **Cost** — CDC often controls construction approach (boundary distances, materials) in ways that constrain design.
- **Risk** — DA pathway carries more uncertainty (council assessment outcome) but allows more design flexibility. CDC pathway is faster but binary — either the design complies or it doesn't.

The project lead must do:
- test the planning pathway against site and design before assuming a pathway;
- record the pathway choice as a §decision-discipline entry with basis;
- if DA, track conditions of consent through tender, delivery, and handover (a buried condition discovered at PC blocks OC).

The project lead must not:
- assume CDC where the site or design fails the test;
- treat conditions of consent as a one-time approval document — they are obligations through delivery and handover.

## BASIX — NSW energy and water compliance

BASIX is the NSW residential sustainability compliance tool. Every new dwelling and most renovations require a BASIX Certificate. Posture in the new-dwelling lifecycle:

### When BASIX applies and binds

- BASIX certificate is generated by a BASIX assessor or designer modelling the dwelling against energy, water, and thermal comfort targets.
- The certificate states **commitments** — specific products, materials, and configurations the dwelling must include.
- Commitments are **locked at DA / CDC submission** and **re-locked at CC** (any post-DA design change that affects BASIX requires a re-modelled certificate).
- **Final certification** — at OC, the BASIX assessor confirms commitments are met. Without the BASIX completion receipt / certificate, OC will not issue.

### What gets committed

Common BASIX commitments for a new Class 1a dwelling:
- **Insulation** — wall and ceiling R-values (often R2.5 walls, R5.0 ceiling minima, though higher with PV trade-off).
- **Glazing** — window U-values and SHGC, by elevation. Specific products often named (or product class).
- **Air-tightness / ventilation** — sometimes mechanical ventilation as a thermal comfort offset.
- **Hot water system** — type (electric heat pump, solar HW, gas instantaneous) and efficiency.
- **PV system** — minimum kW (commonly 3–6 kW for a new Class 1a, depending on the energy score).
- **Lighting** — minimum LED proportion.
- **Water fixtures** — WELS rating minima for taps, showers, toilets, dishwasher.
- **Rainwater tank** — minimum capacity, connection points (toilet / laundry / external).

### Builder posture (cross-reference role overlay)

For `user_role: builder`, BASIX commitments are a delivery obligation inherited from the CC. The builder must:
- read the BASIX certificate at mobilisation and flow commitments to procurement (window schedule, HW selection, PV scope);
- if a substitution is needed (product unavailable, owner preference, value-engineering), trigger re-certification **before installation** — not after;
- maintain a BASIX evidence pack: product datasheets, installation photos, commissioning records, PV connection certificate;
- coordinate the BASIX assessor's final inspection.

For `user_role: owner-builder` or `user_role: architect-pm` or `user_role: d-and-c`, role-specific BASIX posture lives in the respective role overlay.

### Common BASIX failure modes

- **Substitution without re-certification.** Window swap for cost or availability, BASIX commitment broken, surfaces at final inspection. Re-cert + change orders + delays.
- **HW system change late.** Electric HW installed instead of committed heat pump (or vice versa). Same re-cert path.
- **PV underspec or under-installed.** PV commitment is a common score-balancing lever in the certificate — installing 3 kW where 5 kW was committed fails.
- **Wall / ceiling insulation R-value substitution.** Procurement substitutes an R2.0 batt where R2.5 was committed. Detected at frame inspection only if specifically tested.
- **Glazing fixed but elevation rotated.** Window schedule moved during late design changes; orientation changes the SHGC requirement; the original glazing now non-compliant.

For BASIX detail, load `sustainability-energy-guide.md` task-loaded. This section is the **archetype-level posture**, not the full BASIX coverage.

### Non-NSW callout — BASIX equivalent

- **VIC:** NatHERS 6-star (rising to 7-star under the National Construction Code stair-step) — a star-rating system rather than commitment-based. No direct equivalent of the per-product BASIX commitments; instead a whole-of-dwelling rating that the design must achieve.
- **QLD:** NCC energy efficiency (Volume Two, Part 3.12) plus the Sustainable Buildings Code in some council areas.
- **SA:** NatHERS-based, also stair-stepping to 7-star.
- The other states broadly follow NatHERS via NCC. The NSW BASIX system is unique in being commitment-based rather than rating-based.

If `state:` is not NSW and the task touches BASIX, **flag the gap** and ask the project lead to confirm the state-equivalent commitments rather than silently extending BASIX guidance.

## Structural posture — Class 1a new dwelling

A new Class 1a dwelling typically combines:

- **Footings or slab** — on a benign site, a waffle slab or stiffened raft per AS 2870 sized to the site class (M / H1 / H2 / E). On a reactive (H2, E) site, piered footings or deep edge beams. On rock, conventional strip footings.
- **Frame** — timber framing per AS 1684 is the dominant Class 1a solution. Steel framing per AS 4100 / NASH Standard used in cyclonic regions, BAL-29+ areas, and some volume-builder defaults.
- **Roof** — timber or steel trusses; metal sheet or concrete tile cladding; truss design typically by truss supplier per AS 1720 (timber) or AS 4100 (steel).
- **Wind classification** — AS 4055 for housing (N1–N6 in non-cyclonic regions; C1–C4 cyclonic). NSW most areas N2 or N3. Coastal exposure can shift up.
- **Bushfire** — AS 3959 BAL-rated construction where the BAL assessment requires (BAL-12.5+).

Builder / owner-builder / D&C structural delivery obligations (slab inspection, frame inspection, structural completion) sit on the role overlay. The archetype's role here is to flag the structural posture **so that procurement and inspection planning anticipate the right standards**.

For specific standards detail, load `as-standards-reference.md` task-loaded. For trade-specific structural delivery, load `structural-residential.md` task-loaded.

## Structural inspection gates — Class 1a new dwelling

Inspections vary by certifier but typically include:

| Inspection | Trigger | Inspector | Evidence captured |
|---|---|---|---|
| Pre-pour (slab / footing) | Reo placed, formwork complete, before concrete pour | Certifier or engineer | Photos of reo, certificate of conformance from inspecting engineer where required |
| Slab post-pour conformance | Concrete pour completed | Concrete supplier / engineer | Concrete Conformance Test (CFT) — slump, air, 7-day and 28-day strength |
| Frame | Frame complete, before plasterboard | Certifier or engineer | Frame inspection certificate |
| Wet-area waterproofing | Waterproofing complete, before tiling | Certifier or installer | Waterproofer's statement of compliance |
| Stormwater / OSD (if applicable) | OSD tank installed, plumbing complete | Engineer | OSD Structural Inspection Certificate (specific to the OSD scope) |
| Structural completion | Frame, roof, and external envelope complete | Engineer | Structural completion certificate |
| BASIX final | All BASIX commitments installed | BASIX assessor | BASIX completion receipt |
| Final / OC | All works complete | Certifier | Occupation Certificate |

Each inspection is a **mandatory project record** under §evidence-discipline and is filed under `07-construction/09-cc-pc-oc/` (CC/PC/OC documents) and `07-construction/12-reports/` (inspection reports). The inspection register row (per §register-discipline) carries inspection name, scheduled date, actual date, inspector, outcome, evidence reference.

Missing an inspection or proceeding past a gate without sign-off is a common cause of OC delay and HBCF exposure.

## Sequencing risks specific to new-dwelling

The big residential sequencing risks for a Class 1a new build:

- **Utility connection lead time.** Water and sewer connection are typically 4–12 weeks from application; electricity new-connect can be 6–16 weeks; NBN connection variable. Lead times become critical-path late if not lodged at CC.
- **BPA scheduling.** Engaged early-stage; can have lead times for inspections that strand site progress (e.g. frame inspection slot in three weeks while framer waits).
- **OSD inspection (where applicable).** OSD is a specific Sydney Water (or council) inspection; missed in the inspection schedule, blocks OC.
- **Concrete pour weather.** Slab pours on hot days (cure) or wet days (slump) get rescheduled; suppliers reschedule based on availability and weather; programme slips by days at a time.
- **Truss supply.** Roof trusses are typically 4–8 week lead. Delays in CC or in detail design ripple into truss order ripple into frame programme.
- **Window supply.** Many residential window specifications have 8–14 week lead times. Window order tied to BASIX commitments; substitution late is the BASIX failure mode above.
- **Brick supply.** Brick selection and supply can constrain the bricklayer's start; brick lead time is sometimes 6–12 weeks for a non-standard product.
- **PV installation, HW system installation, electrical commissioning.** All BASIX-linked; sequence at completion.

## Common failure modes — new-dwelling archetype

- **Soil report ordered too late** — design developed assuming benign site; H2 / E classification triggers slab redesign and cost movement.
- **BAL not tested until DA** — site is bushfire-prone, BAL-29 triggers full AS 3959 envelope, cost ripples through window schedule, decking, gutter guards.
- **Sydney Water BOS not lodged early enough** — sewer crosses building footprint; BOS approval delays start.
- **OSD designed late** — added after DA; redesign of stormwater and falls on the lot.
- **BASIX commitments substituted in procurement without re-certification.**
- **Planning condition of consent buried** — e.g. landscape works completion required before OC; not picked up until PC walk.
- **HOW/HBCF issued after deposit taken** — statutory breach by the builder (role-specific failure, see role overlay).
- **CC plans superseded mid-construction** — built to wrong revision (typically window schedule or BASIX-linked detail).
- **Builder programme assumes 5-day weeks year-round** — no allowance for industry close-down (mid-December to mid-January), wet weather, public holidays.
- **DLP start date not pinned to PC** — DLP exposure drifts, defects claims drift.

## Agent behaviour under this archetype

When `archetype: new-dwelling` is declared:

1. The agent loads this seed and the matching `user_role:` overlay on any phase-gate task.
2. The agent reads project evidence under the §1 authority stack — survey, soil report, BAL assessment, dilapidation, DA / CDC / CC, BASIX certificate, Sydney Water / BOS, head contract, executed insurances and licences.
3. The agent applies §evidence-discipline labelling and contractual / stakeholder register per §6 folder rules.
4. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
5. The agent surfaces sequencing-risk triggers from the lifecycle table — particularly utility lead time and BASIX substitution — without waiting to be asked.

## See also

- `../00-doctrine/doctrine.md` — abstract project lead doctrine
- `../00-doctrine/doctrine.md §seed-consultation-discipline` — why this seed loads
- `../00-doctrine/doctrine.md §state-handling` — non-NSW callouts and gap-flagging
- `../AGENTS.md §1` (authority stack), `§2` (declaration gate), `§7` (cross-archetype tasks), `§8` (state handling)
- `role-owner-builder.md` / `role-architect-pm.md` / `role-builder.md` / `role-d-and-c.md` — role overlays loaded alongside this archetype
- `setup-and-commission-guide.md` — mobilisation workflow per role
- `contract-administration-guide.md` — head contract clause coverage
- `ncc-reference-guide.md` — NCC Class 1 / Class 10 reference (task-loaded)
- `as-standards-reference.md` — AS 2870, AS 1684, AS 4055, AS 3959, AS 3500 (task-loaded)
- `sustainability-energy-guide.md` — BASIX detail (task-loaded)
- `structural-residential.md` — slab classification, framing, wind, BAL (task-loaded)
- `mep-residential.md` — domestic HW, gas, electrical, NBN, mechanical ventilation (task-loaded)
- `civil-residential.md` — cut/fill, stormwater, OSD, sewer connection, driveway crossover (task-loaded)
- `finishes-residential.md` — external cladding, roofing, internal linings, wet area waterproofing, tiling, joinery, glazing, flooring (task-loaded)
- `trade-interfaces-coordination-guide.md` — residential trade sequencing, hold points, and common failure modes (task-loaded)
- `program-scheduling-guide.md` — residential cycle-time benchmarks (task-loaded)
- `../02-skills/atomic/seed-targeted-read.md` — the gate that loads this seed

---

## Decision catalog (HITL)

Brief/scope decisions typical of a sparse new-dwelling brief. Finishes detail lives in
`finishes-residential.md`; these cover dwelling configuration and services posture.

```decision-catalog
- id: dwelling-storeys
  section: Brief and scope
  label: Dwelling storeys
  applies_to:
    archetypes: [new-dwelling]
    classes: [residential]
  options:
    - { value: single_storey, label: Single storey }
    - { value: two_storey, label: Two storey }
    - { value: split_level, label: Split level }
  default_hint: single_storey
- id: garage-type
  section: Brief and scope
  label: Garage / car parking
  applies_to:
    archetypes: [new-dwelling]
    classes: [residential]
  options:
    - { value: none, label: No garage (on-street / carport later) }
    - { value: single_garage, label: Single garage }
    - { value: double_garage, label: Double garage }
    - { value: carport, label: Carport }
  default_hint: double_garage
- id: hot-water-system
  section: Brief and scope
  label: Hot water system
  applies_to:
    archetypes: [new-dwelling, renovation]
    classes: [residential]
  options:
    - { value: instantaneous_gas, label: Instantaneous gas }
    - { value: storage_electric, label: Electric storage }
    - { value: heat_pump, label: Heat pump }
    - { value: solar_boosted, label: Solar-boosted }
  default_hint: heat_pump
- id: heating-cooling
  section: Brief and scope
  label: Heating and cooling
  applies_to:
    archetypes: [new-dwelling, renovation]
    classes: [residential]
  options:
    - { value: split_system, label: Split-system air conditioning }
    - { value: ducted_reverse_cycle, label: Ducted reverse-cycle }
    - { value: multi_split, label: Multi-split }
    - { value: none_specified, label: Not yet specified }
  default_hint: split_system
```
