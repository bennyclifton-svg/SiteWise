---
tier: overlay
seed_type: work-type-guide
loaded_by: "archetype: renovation"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_classes: [residential]
applies_to_work_types: [refurb, extend]
applies_to_subclasses: [house, townhouses]
applies_to_archetypes: [renovation]
state_default: NSW
topics: [renovation, heritage, dilapidation, latent-conditions, tie-in, basix, occupancy]
summary: "Residential refurbishment and extension overlay: due diligence package, latent conditions, dilapidation and neighbour management, structural intervention, BASIX for additions, heritage checks, live-occupancy staging and old-to-new tie-in risks."
required_by: {create-pmp: 2}
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §state-handling]
agents_anchors: [§1, §2, §3, §7, §8, §11]
status: seed
author: agent
reviewed_on: 2026-08-14
---

# Archetype seed - Renovation

A **renovation** changes an existing dwelling: alteration, addition, partial demolition, structural intervention, services upgrade, waterproofing renewal, facade / roof change, or staged work around an occupied home. The existing building is not background context. It is project evidence, project constraint, and project risk.

This overlay loads for residential `refurb` and `extend` on house and townhouse projects, and remains the archetype seed when a project still declares `archetype: renovation`. It is role-neutral; the role overlay sits on top.

NSW is the deep default. Non-NSW states get inline graceful-degradation callouts only. If a renovation task turns on a state-specific instrument and no callout exists, flag the gap instead of extending NSW guidance.

## Scope of this archetype

This seed covers:

- alterations and additions to an existing Class 1a dwelling;
- partial demolition and rebuild;
- second-storey additions;
- internal layout changes involving structure, waterproofing, services, or compliance;
- additions that tie into an existing roof, slab, drainage, facade, or services system;
- renovation work to a secondary dwelling where the existing building remains material.

This seed does not cover:

- a new detached dwelling on a cleared site (load `new-dwelling-guide.md`);
- a standalone granny flat or shed where the existing house is not affected (load `ancillary-guide.md`);
- two or more attached dwellings as a development product (load `multi-dwelling-guide.md`);
- strata apartment renovations as the primary v1 use case; if they appear, flag strata approval / design practitioner / by-law issues as state-specific gaps.

## Renovation posture

In a renovation, assumptions hide in walls, slabs, roofs, services, and neighbour interfaces. The project lead should assume existing conditions are unknown until proven otherwise.

The renovation rule set:

- **Locate before disturb.** Services, structure, asbestos, waterproofing, and drainage are investigated before demolition, cutting, drilling, or excavation.
- **Record before change.** Dilapidation, photos, survey, and existing condition notes come before intrusive work.
- **Decide before spend.** Latent conditions become decision records with cost, time, compliance, and scope consequences.
- **Do not normalise verbal changes.** Renovation projects invite small site decisions; each material decision still needs a record.
- **Treat tie-ins as high risk.** Old-to-new junctions cause water, movement, level, fire, acoustic, energy, and finish problems.

## Due diligence package

Before concept lock, procurement, or start-on-site, collect the evidence below or record why it is not available.

| Evidence | Purpose | Typical folder |
|---|---|---|
| Survey / measure-up | Existing levels, boundaries, wall positions, easements, floor levels, roof geometry | `03-design/01-due-diligence/` |
| Dilapidation report | Baseline condition of neighbouring properties and existing retained parts | `03-design/01-due-diligence/` |
| Existing structural information | Identify load-bearing walls, roof / floor structure, slab / footing assumptions | `03-design/01-due-diligence/` |
| Services locating | Water, sewer, stormwater, gas, electrical, solar, NBN, HVAC, hidden penetrations | `03-design/01-due-diligence/` |
| Hazardous materials check | Asbestos, lead paint, silica, mould, contaminated fill | `03-design/01-due-diligence/` |
| Moisture / termite / rot check | Hidden water ingress, subfloor ventilation, framing condition, pest damage | `03-design/01-due-diligence/` |
| Heritage / character review | Planning controls, demolition limits, streetscape, conservation area, character statements | `03-design/01-due-diligence/` |
| BASIX trigger check | Alteration/addition value, pool/spa trigger, modelling scope | `04-planning-and-authorities/` |
| Approval pathway check | Exempt, CDC, DA + CC, or other approvals | `04-planning-and-authorities/` |

Missing due diligence is not a minor gap. It directly affects price reliability, programme reliability, and authority risk.

## Latent conditions

Latent conditions are existing conditions that could not reasonably be known from the information available at pricing / setup. In renovations, they are common rather than exceptional.

Common categories:

- concealed structural damage or unauthorised prior alteration;
- rotten framing, termite damage, corroded lintels, failed slab or footing;
- asbestos, lead paint, mould, contaminated fill;
- unmarked services or non-compliant services;
- waterproofing failure behind tiles or balconies;
- drainage defects and insufficient falls;
- mismatched floor levels and old-to-new movement;
- heritage fabric discovered after opening up.

The agent should classify each latent condition signal:

| Classification | Meaning | Action |
|---|---|---|
| Fact | Source document or direct observation confirms the condition | Open cost / programme / risk / decision rows |
| Assumption | Reasonable risk not yet evidenced | Record as assumption and assign investigation |
| Judgement | Project lead interpretation of available evidence | State basis and recommended path |
| Recommendation | Proposed decision or action | Route to the correct decision-maker |

Do not price or programme renovation work as though unknowns do not exist. Where unknowns remain, use contingency, provisional sums, schedule-of-rates, or staged investigation rather than false certainty.

## Dilapidation and neighbour management

Renovations often affect shared boundaries, party walls, excavation zones, roof-water paths, and neighbour amenity.

Use dilapidation evidence before:

- demolition near a boundary;
- excavation, underpinning, piering, retaining walls, drainage works, or vibration;
- scaffold, craneage, tight access, or neighbour-property access;
- roof tie-ins and stormwater changes that may affect adjoining properties;
- work to heritage / character streetscapes.

The dilapidation record should be filed before disruptive work and referenced in the risk register. It is not a neighbour-relationship nicety; it is dispute reconstruction evidence.

## Existing services investigation

Existing services are a major renovation risk because drawings are often missing or inaccurate.

Investigate:

- sewer alignment, boundary trap, inspection openings, venting, and Sydney Water constraints;
- stormwater route, OSD, charged lines, pits, overland flow, and illegal connections;
- water service, meters, pressure, hot-water system, and isolation points;
- gas service and disconnection / relocation requirements;
- electrical mains, switchboard capacity, old wiring, solar / battery, earthing, temporary power;
- NBN / telecommunications and security cabling;
- HVAC, ventilation, exhaust, and condensation paths.

Default rule: locate before disturb. If a service cannot be located, the risk is recorded and work proceeds only with a conscious decision about potholing, scanning, isolation, or staged demolition.

## Structural intervention

Renovations frequently touch structure even when the scope is described as cosmetic.

Structural intervention includes:

- removing or widening openings in load-bearing walls;
- second-storey additions and roof conversions;
- underpinning, piering, or footing changes;
- cutting slabs, beams, joists, rafters, trusses, or bracing walls;
- temporary propping and sequencing works;
- roof tie-ins, valley / gutter reconfiguration, and old-to-new support conditions;
- balcony, deck, stair, balustrade, and retaining wall work.

Required posture:

- identify load paths before demolition;
- obtain engineer advice before structural removal;
- document temporary works where the structure is unsupported at any stage;
- record inspection hold points;
- file engineer certificates and site photos;
- treat unexpected structure as a latent condition and decision item.

The agent must not recommend structural changes from general knowledge. It can surface the risk, list evidence needed, and route to the engineer / certifier.

## BASIX baseline for additions

In NSW, BASIX applies to alterations and additions valued at $50,000 and over, and to pool / spa work over the stated volume trigger. BASIX for alterations and additions differs from the new-dwelling path; it aims to reduce water use and emissions through upgrades such as glazing, insulation, fixtures, lighting, hot water, and pool/spa measures.

For a renovation, the BASIX posture is:

- determine at setup whether the project triggers BASIX;
- file the BASIX certificate in `04-planning-and-authorities/` where required;
- flow commitments into drawings, specifications, trade scopes, and procurement;
- treat substitutions as compliance changes requiring BASIX / certifier review before installation;
- carry BASIX evidence to handover / OC.

Typical renovation BASIX failure modes:

- new windows selected without checking U-value / SHGC commitments;
- insulation added to only part of the altered area;
- hot-water system or lighting substituted without re-check;
- pool/spa heating or cover requirements missed;
- existing vs new building elements confused in the certificate;
- BASIX treated as a one-off approval document rather than a procurement and handover constraint.

**State callout — VIC:** Victoria does not use BASIX. VIC uses NatHERS + NCC Section J as the energy compliance pathway for alterations and additions. There is no per-commitment certificate equivalent to the NSW BASIX certificate — the compliance mechanism is a star-rating assessment by an accredited NatHERS assessor (or a Section J DTS compliance statement). If `state: VIC` and the task involves energy or water compliance for a renovation, **flag the gap**: confirm the applicable energy compliance pathway and trigger threshold with the certifier and energy assessor before proceeding. Do not extend BASIX guidance to a VIC project.

For deep sustainability guidance, load `sustainability-energy-guide.md` task-loaded.

## Heritage and character due diligence

Heritage / character is absorbed into this renovation seed. Do not create a standalone heritage seed in v1.

Check:

- heritage item status;
- heritage conservation area;
- character overlay, local streetscape controls, contributory building status;
- demolition controls;
- facade, roof form, window, fence, material, and visible-addition constraints;
- archaeological / Aboriginal heritage triggers where relevant;
- council pre-lodgement advice or heritage consultant advice.

If heritage / character applies, it shapes approval pathway, design options, demolition scope, programme, and cost. Treat it as an early design and approval risk, not as late paperwork.

## Live occupancy and staging

Many renovations occur while the owner lives on site. This changes delivery risk:

- temporary kitchen / bathroom / laundry arrangements;
- dust, noise, vibration, security, pets, children, access separation;
- temporary services and isolations;
- after-hours protection, weatherproofing, and temporary roof / wall closure;
- fire separation and safe egress;
- owner decisions occurring at the workface.

Where the owner remains in occupation, the agent should ask for a staging and safety plan. In an owner-builder project, the owner-builder still needs a written plan for their future self and trades.

## Waterproofing and old-to-new tie-ins

Water is the renovation archetype's quiet villain. High-risk zones:

- wet areas and waterproofing terminations;
- balconies, decks, roof terraces, and step-downs;
- roof tie-ins, valleys, box gutters, flashings, and penetrations;
- ground levels, subfloor ventilation, drainage falls, and weep paths;
- window / door replacement into old openings;
- render, cladding, sarking, and cavity continuity;
- old slab to new slab junctions and movement joints.

Record hold points and evidence:

- waterproofing product and installer details;
- substrate readiness;
- photos before covering;
- certificates / statements of compliance;
- flood test where required;
- flashing and drainage details before cladding / finishes close.

## Procurement and pricing posture

Renovation quotes should be read for assumptions and exclusions, not just price.

Preferred pricing posture:

- lump sum where design and existing conditions are sufficiently resolved;
- provisional sums where scope is known but quantity is uncertain;
- schedule-of-rates where hidden conditions or staged opening-up make quantities unreliable;
- staged investigation package before committing to the main works where risk is high.

Trade quote checks:

- demolition and protection included / excluded;
- waste and hazardous material handling;
- temporary works and propping;
- make-good and finishes matching;
- service isolation, relocation, and reconnection;
- access, scaffold, cranage, parking, and delivery constraints;
- waterproofing and certification;
- inspection and hold-point obligations.

## Renovation risk baseline

At risk-register setup, consider these categories:

- latent conditions;
- existing services;
- dilapidation / neighbour condition;
- structural intervention;
- hazardous materials;
- BASIX-for-additions;
- heritage / character constraint;
- live occupancy and staging;
- waterproofing / old-to-new tie-ins;
- trade scope gaps;
- authority / certifier uncertainty;
- contingency depletion;
- owner decisions / late selections.

For `user_role: owner-builder`, decision-heavy renovation risks should create park-for-decision rows as well as risk rows.

## Agent behaviour under this archetype

When `archetype: renovation` is declared:

1. The agent loads this seed and the matching role overlay on any phase-gate task.
2. The agent treats existing conditions as evidence to be verified, not background story.
3. The agent labels missing due diligence as an evidence gap or assumption.
4. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
5. The agent surfaces latent-condition, service, structural, waterproofing, BASIX, and heritage / character risks early.
6. The agent loads secondary archetype seeds task-loaded where a renovation includes a granny flat, detached studio, multi-dwelling component, or small-commercial fallback scope.

## See also

- `../00-doctrine/doctrine.md` - project lead doctrine
- `../00-doctrine/doctrine.md` seed-consultation-discipline
- `../00-doctrine/doctrine.md` evidence-discipline
- `../00-doctrine/doctrine.md` register-discipline
- `../00-doctrine/doctrine.md` decision-discipline
- `../00-doctrine/doctrine.md` escalation-triggers
- `../00-doctrine/doctrine.md` state-handling
- `../AGENTS.md` Sec. 1, Sec. 2, Sec. 3, Sec. 7, Sec. 8, Sec. 11
- `role-owner-builder.md` - slice-07 role overlay
- `setup-and-commission-guide.md` - setup and ready-to-start workflow
- `contract-administration-guide.md` - latent-condition, variation, EOT, and clause posture
- `procurement-quoting-guide.md` - quote comparison, SOR, and scope-gap discipline
- `cost-management-principles.md` - contingency, provisional sums, owner-supplied items, and cost movement
- `program-scheduling-guide.md` - staging, lead times, shutdowns, lookahead, and delay
- `ncc-reference-guide.md` — NCC Class 1/10 reference, DTS vs Performance Solution (task-loaded)
- `as-standards-reference.md` — AS 2870, AS 1684, AS 4055, AS 3959, AS 3500 (task-loaded)
- `sustainability-energy-guide.md` — BASIX/NatHERS guidance, common compliance failure points (task-loaded)
- `structural-residential.md` — AS 2870 footings, AS 1684 framing, wind classification, BAL (task-loaded)
- `mep-residential.md` — domestic HW, gas, electrical, NBN, mechanical ventilation (task-loaded)
- `civil-residential.md` — cut/fill, stormwater, OSD, sewer connection, driveway crossover (task-loaded)
- `finishes-residential.md` — external cladding, roofing, internal linings, wet area waterproofing, tiling, joinery, glazing, flooring (task-loaded)
- `trade-interfaces-coordination-guide.md` — residential trade sequencing, hold points, and common failure modes (task-loaded)
- `../02-skills/atomic/seed-targeted-read.md` - loads this archetype seed
- `../02-skills/systems/risk-register-system.md` - renovation risk baseline and park-for-decision surfacing
