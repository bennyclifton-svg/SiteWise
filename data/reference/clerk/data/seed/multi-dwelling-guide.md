---
tier: archetype
seed_type: archetype
loaded_by: "archetype: multi-dwelling"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_classes: [residential]
applies_to_work_types: [new]
state_default: NSW
status: legacy-retained
superseded_by: seed/residential-construction-guide.md
summary: "Archetype coverage for two or more dwellings delivered as one project — townhouses, terraces, villas, repeated detached dwellings. Covers the classification gate, party-wall fire rating, separate metering, infrastructure contributions, staging, and the cost, programme and procurement postures specific to small multi-dwelling work."
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §state-handling]
agents_anchors: [§1, §2, §3, §7, §8, §11]
---

# Archetype seed - Multi-dwelling

A **multi-dwelling** project delivers two or more dwellings as one coordinated project. In SiteWise v1 the primary case is a small residential development such as attached townhouses, terraces, villas, or repeated detached dwellings on one site. Class 2 apartment material is used only where the building classification and project evidence require it.

This seed is loaded when the project `README.md` declares `archetype: multi-dwelling`. It is role-neutral; the role overlay (`role-owner-builder.md`, `role-architect-pm.md`, `role-builder.md`, or `role-d-and-c.md`) sits on top. Slice 09's primary role pair is `user_role: d-and-c`.

NSW is the deep default. Non-NSW states get inline graceful-degradation callouts only. If a multi-dwelling task turns on a state-specific instrument and no callout exists, flag the gap instead of extending NSW guidance.

## Scope of this archetype

This seed covers:

- attached townhouses, terraces, row houses, villas, and small multi-dwelling residential developments;
- repeated Class 1a dwellings delivered under one contract or programme;
- small strata or community-title developments where subdivision / handover sequence affects delivery;
- Class 2 apartment-style buildings only where project evidence shows a Class 2 classification;
- D&C townhouse or small multi-dwelling delivery where design responsibility, certifier submissions, and construction staging are linked.

This seed does not cover:

- a single detached house (load `new-dwelling-guide.md`);
- a renovation to one dwelling unless the project includes a material multi-dwelling component (load `renovation-guide.md` and task-load this seed if needed);
- a standalone granny flat as the primary scope (load `ancillary-guide.md`);
- large commercial apartment towers as the default SiteWise v1 case. Use the commercial harness or task-load commercial apartment material if the project exceeds SiteWise's residential depth.

## Classification gate - do this first

Classification drives the NCC volume, fire separation, accessibility, energy, certifier submissions, inspections, and handover evidence.

Do not assume that "six units" means Class 2. A NSW 6-unit townhouse project may be a group of Class 1a attached dwellings if each dwelling:

- has its own ground-level entrance;
- is separated from the next dwelling by fire-resisting wall construction;
- is not located above or below another dwelling;
- does not rely on common internal corridors, common vertical circulation, or a shared building core.

Class 2 is more likely where dwellings are within one building form with common circulation, stacked dwellings, common lobbies / corridors, common fire stairs, or apartment-style sole-occupancy units.

Classification setup questions:

| Question | Why it matters | Evidence |
|---|---|---|
| Are dwellings side-by-side, stacked, or mixed? | Class 1a vs Class 2 test | Drawings, sections, certifier advice |
| Does each dwelling have direct ground-level entry? | Class 1a townhouse indicator | Architectural plans |
| Are dwellings separated by fire-resisting walls? | Class 1a attached dwelling pathway | Wall details, NCC report |
| Is there common internal circulation? | Class 2 indicator | Plans, fire egress strategy |
| Is there shared basement / podium / common property? | May shift classification or mixed-class treatment | Sections, certifier advice |
| Has the certifier confirmed classification? | Authority path and inspection schedule | Certifier advice / CC documents |

If classification is not evidenced, label it as an Assumption and open an action. Do not draft a classification-dependent ready-to-start checklist as though the matter is settled.

## Planning and approval posture

Multi-dwelling projects often carry more authority and consent-condition weight than a single dwelling.

Early approval evidence:

- planning pathway: DA, CDC where available, staged DA / CC, or other pathway;
- NCC classification advice;
- BASIX / energy certificate strategy for multiple dwellings;
- subdivision, strata, or community-title pathway where relevant;
- consent conditions register;
- Sydney Water / water authority servicing pathway;
- OSD and stormwater strategy;
- driveway / crossover and waste collection strategy;
- public-domain works, street tree, kerb, footpath, or road-opening requirements;
- infrastructure contributions / development contributions / utility augmentation assumptions;
- principal certifier appointment and inspection schedule.

Consent conditions become live delivery obligations. Extract them into the consent conditions register before procurement or site start.

## Party-wall fire-rating

Party walls are the signature multi-dwelling risk for attached townhouse projects.

The project lead must confirm:

- required fire-resistance level or fire-resisting construction basis;
- wall continuity from footing / slab to roof or required termination detail;
- roof cavity, eaves, fascia, and parapet treatment;
- acoustic and structural requirements where they overlap with fire;
- services penetrations through or near the wall;
- fire collars, wraps, sealants, dampers, or tested systems required for penetrations;
- inspection hold points before linings, roofing, or services conceal the wall;
- trade responsibility for penetrations and reinstatement;
- photographic evidence before cover-up.

Common party-wall failure modes:

- wall stops short at roof space or cavity, breaking fire separation;
- services trades penetrate the wall after inspection without reinstatement;
- tested system not matched to actual wall build-up;
- left-hand / right-hand dwelling variants change penetration locations;
- acoustic detail and fire detail conflict;
- inspection evidence not captured before plasterboard or roof works conceal the assembly.

For D&C projects, the party-wall detail must appear in the design responsibility matrix and design deliverables register.

## Separate metering and services

Every dwelling needs a clear services strategy. "Services by others" is not a strategy.

Metering / services questions:

- Are water meters per dwelling, common, or master plus submeter?
- Are electrical meters per dwelling, embedded network, common board, or strata common area?
- Is gas provided, omitted, or common plant?
- Are NBN / communications lead-ins per dwelling or common?
- Are solar PV systems individual, common-area, or not included?
- Are EV provisions required by planning, NCC, market expectation, or owner brief?
- Are sewer, stormwater, OSD, and water authority approvals coordinated with staging?
- Are common services protected during staged handover?

Evidence to capture:

- authority applications;
- meter provider / utility correspondence;
- single-line diagrams;
- hydraulic and civil drawings;
- electrical board and meter-panel drawings;
- service trenching / easement drawings;
- inspection and commissioning records;
- per-dwelling handover evidence.

Common metering failure modes:

- utility applications lodged too late;
- meter board location conflicts with fire, access, or architectural frontage;
- per-dwelling water or electrical metering not allowed for in cost plan;
- common-area power / water omitted;
- staged handover assumes services can be energised per dwelling when the design only allows whole-site energisation;
- NBN / communications pathway missing from civil works.

## Infrastructure contributions and authorities

Multi-dwelling projects commonly trigger contributions or authority works that are minor on a single dwelling but material across multiple dwellings.

Track:

- council infrastructure / development contributions;
- water and sewer servicing charges or Section 73-style requirements;
- stormwater / OSD design, certification, and inspection;
- driveway / crossover approval and construction;
- footpath, kerb, gutter, road-opening, and public-domain works;
- waste collection and bin-storage requirements;
- street tree protection or replacement;
- utility augmentation or service upgrades;
- survey, subdivision certificate, strata / community-title documentation where relevant;
- OC / interim OC / staged OC strategy.

These items must appear in the cost plan and authority approvals tracker. If the contribution amount or authority condition is unknown, label the allowance as an Assumption and assign an action.

## Staging guidance

Multi-dwelling delivery is usually staged even when there is one contract.

Staging decisions:

- build all dwellings in parallel or sequence by block / pair / stage;
- one site establishment or staged possession areas;
- trade repetition by dwelling, facade, services, and finishes;
- inspection strategy by dwelling, stage, or whole-site;
- staged OC / interim OC path;
- staged handover to owner, purchaser, or tenant;
- subdivision / strata registration dependency;
- buyer / owner access and safety controls;
- defects capture by dwelling and by common property;
- common-area completion before individual dwelling handover.

Staging risks:

- one dwelling ready but common services / access not ready;
- defects repeated across all dwellings because first-of-type inspection was skipped;
- OC path requires common areas complete before any dwelling can be occupied;
- buyer settlement / owner access promised before authority approvals allow it;
- trade productivity assumptions ignore constrained access and overlapping workfaces;
- design changes found on unit 1 are not flowed to units 2-6.

The first repeated element should be treated as a prototype. Inspect it before repeating the defect across the site.

## Design and documentation posture

Multi-dwelling design needs more coordination than a single dwelling because repeated units magnify mistakes.

Required design controls:

- current drawing register;
- dwelling type schedule: Unit A / Unit B / left-hand / right-hand / accessible or adaptable variants;
- room data or finish schedule per dwelling where selections vary;
- wall type schedule, including party-wall and acoustic wall types;
- services distribution / metering strategy;
- civil and stormwater coordination;
- BASIX / energy commitments per dwelling or per certificate;
- design responsibility matrix where consultants or D&C role make responsibilities material;
- certifier submission schedule;
- inspection hold-point schedule;
- revision control before trade release.

Where design differs between dwellings, do not let the agent use "typical" silently. It must state which dwelling type the advice applies to.

## Cost posture

Multi-dwelling cost planning must capture repetition and shared infrastructure.

Cost items often missed:

- authority contributions and utility charges;
- water / sewer / stormwater augmentation;
- OSD and civil works;
- separate metering and meter boards;
- party-wall fire and acoustic upgrades;
- repeated wet areas and waterproofing;
- common-area lighting, external works, paths, driveways, landscaping, and waste areas;
- staged preliminaries and extended site establishment;
- consultant and certifier fees across multiple submissions;
- subdivision / strata / survey fees;
- contingency for repeated-defect rectification.

Pricing posture:

- per-dwelling rates can work for repeated trade scopes;
- left-hand / right-hand variants need explicit adjustment;
- party-wall and services interfaces should not be hidden in generic wall / services rates;
- shared infrastructure should be separated from per-dwelling costs;
- staged handover can shift preliminaries, cashflow, and defect costs.

## Programme posture

The programme must show repeated cycles and staging logic.

Programme should identify:

- design lock and certifier submission dates;
- authority and utility lead times;
- civil / services rough-in before vertical works;
- first-of-type inspection for party walls, waterproofing, services risers or trenches, and finishes;
- repeated dwelling cycle: slab / frame / roof / lockup / rough-in / lining / waterproofing / fixing / commissioning;
- overlapping trade zones and access constraints;
- staged OC / handover / subdivision dependencies;
- defect close-out per dwelling and common area.

For D&C, the design programme and construction programme must talk to each other. A construction activity that depends on an unreleased design package is a risk, not a normal float item.

## Procurement posture

Multi-dwelling procurement often blends lump sum packages and per-unit rates.

Trade scopes should expose:

- scope per dwelling type;
- common works vs dwelling-specific works;
- party-wall penetrations and reinstatement;
- metering / services interfaces;
- first-of-type inspection and sign-off requirement;
- left-hand / right-hand variants;
- staging assumptions and access constraints;
- defect-response obligations where the same detail repeats.

For D&C projects, consultant procurement is also procurement. Consultant scope gaps affect design responsibility and certifier submissions; they should be recorded through the D&C role overlay and consultant advice / design responsibility registers.

## Handover and defects posture

Slice 13 owns the end-to-end handover / DLP system, but multi-dwelling setup must plant the right controls early.

Track:

- per-dwelling OC / interim OC / final OC assumptions;
- common-area completion and access controls;
- utility commissioning per dwelling;
- dwelling-by-dwelling defect lists;
- common-property defects;
- warranties and O&M evidence per dwelling and common area;
- purchaser / owner access and settlement constraints where relevant;
- DLP start date per stage if staged completion is permitted.

Do not assume all dwellings share one clean PC / OC date. The executed contract and approval pathway decide that.

## Risk baseline

At risk-register setup, consider these categories:

- classification unresolved: Class 1a vs Class 2;
- party-wall / fire separation;
- service penetrations through fire-rated construction;
- separate metering and utility lead times;
- OSD / stormwater / civil approvals;
- infrastructure contributions or authority charges omitted;
- staged OC / subdivision / strata path unclear;
- repeated trade defect across dwellings;
- common-area completion blocking handover;
- design responsibility matrix missing for D&C projects;
- consultant scope gap;
- first-of-type inspection skipped;
- buyer / owner access before authority path permits it.

## Non-NSW callouts

NSW is the deep default. For non-NSW projects:

- verify the local planning pathway, building permit / certificate language, and certifier / surveyor role;
- verify state-specific home building insurance, domestic building insurance, or warranty scheme;
- verify state-specific infrastructure contribution and water authority processes;
- verify whether BASIX is replaced by NatHERS / NCC energy pathway or another state-specific sustainability instrument;
- flag gaps rather than translating NSW terms directly.

## Agent behaviour under this archetype

When `archetype: multi-dwelling` is declared:

1. The agent loads this seed and the matching role overlay on any phase-gate task.
2. The agent tests classification before relying on Class 1a or Class 2 assumptions.
3. The agent treats party-wall fire-rating, separate metering, infrastructure contributions, and staging as setup topics, not late delivery topics.
4. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
5. The agent labels missing classification, metering, contribution, staging, or party-wall evidence as Assumptions with actions.
6. The agent loads secondary archetype seeds task-loaded only where the project has a genuine secondary scope.

## See also

- `../00-doctrine/doctrine.md` - project lead doctrine
- `../00-doctrine/doctrine.md` seed-consultation-discipline
- `../00-doctrine/doctrine.md` evidence-discipline
- `../00-doctrine/doctrine.md` register-discipline
- `../00-doctrine/doctrine.md` decision-discipline
- `../00-doctrine/doctrine.md` escalation-triggers
- `../00-doctrine/doctrine.md` state-handling
- `../AGENTS.md` Sec. 1, Sec. 2, Sec. 3, Sec. 7, Sec. 8, Sec. 11
- `role-d-and-c.md` - slice-09 role overlay and D&C design responsibility posture
- `role-builder.md` - builder obligation base
- `setup-and-commission-guide.md` - setup and ready-to-start workflow
- `contract-administration-guide.md` - AS 4902, HIA / MBA, variation, EOT, and clause posture
- `procurement-quoting-guide.md` - repeated trade scopes, per-unit rates, and consultant procurement
- `cost-management-principles.md` - allowances, authority contributions, and shared infrastructure
- `program-scheduling-guide.md` - repeated cycles, lookahead, staging, and delay
- `ncc-reference-guide.md` — NCC Class 1/10 reference, DTS vs Performance Solution (task-loaded)
- `as-standards-reference.md` — AS 2870, AS 1684, AS 4055, AS 3959, AS 3500 (task-loaded)
- `sustainability-energy-guide.md` — BASIX/NatHERS guidance, common compliance failure points (task-loaded)
- `structural-residential.md` — AS 2870 footings, AS 1684 framing, wind classification, BAL (task-loaded)
- `mep-residential.md` — domestic HW, gas, electrical, NBN, mechanical ventilation (task-loaded)
- `civil-residential.md` — cut/fill, stormwater, OSD, sewer connection, driveway crossover (task-loaded)
- `finishes-residential.md` — external cladding, roofing, internal linings, wet area waterproofing, tiling, joinery, glazing, flooring (task-loaded)
- `trade-interfaces-coordination-guide.md` — residential trade sequencing, hold points, and common failure modes (task-loaded)
- `../02-skills/atomic/seed-targeted-read.md` - loads this archetype seed
- `../02-skills/systems/contract-setup-system.md` - D&C + multi-dwelling ready-to-start path
- `../02-skills/systems/risk-register-system.md` - multi-dwelling risk baseline
