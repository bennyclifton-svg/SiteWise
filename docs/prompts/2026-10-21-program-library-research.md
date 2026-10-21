# Prompt: research and draft the SiteWise program library

Paste everything below the line into the research agent. It needs read and
write access to the repository at `D:/AI Projects/sitewise` and the ability to
run `python tools/check_knowledge.py --strict`.

---

## Your job

You are drafting the knowledge library that SiteWise uses to build a
construction program (a staged schedule with activities, durations and links)
for Australian building projects. When a project manager clicks "Build
program", code reads the project's facts and this library, and produces the
program. No AI runs at that moment: the program can only contain what you
write into these files.

Use your own industry knowledge of how Australian projects are designed,
approved, procured, built and handed over. The goal is a **reasonably accurate
framework**: sensible stages, the activities a PM expects to see, realistic
duration ranges, and correct finish-to-start style links, varying sensibly by
work type, building class and scale. It does not need to be perfect. Every
row you write is a draft the owner will review.

Write only these files:

- `knowledge/program/skeletons.yaml`
- `knowledge/program/fragments.yaml`
- `knowledge/program/durations.yaml`
- `knowledge/program/calendars.yaml` (optional, Part D)
- `docs/research/2026-10-21-program-library.md` (your research note, required)

Do not edit any other file. Do not change code.

## How the program is built (read this first)

A program has two tiers: **stages** and **activities** under them.

1. **Skeleton** picks the stages and their order. Code evaluates every
   skeleton's `applies_when` against the project's facts and picks the match
   with the most conditions. If none match, it uses `sk.generic` (already in
   the file; do not change it).
2. **Fragments** add activities under a stage. A fragment applies once for
   each included work item on one of its systems whose facts make its
   `applies_when` true. (A project has work items such as "New footings" on
   system `substructure.footings` with action `new`.)
3. **Durations** give each activity a low / typical / high length in working
   days, either fixed or scaled by a project fact (floor area, storeys).
4. **Links** inside a fragment sequence its activities; `attach` places the
   fragment relative to the stage start/finish or another fragment. Code
   chains anything left unlinked in order.

Facts a predicate can test (use only these):

| Operand | Values | Use for |
| --- | --- | --- |
| `{det: work_type, eq/any_of}` | `new`, `refurb`, `extend`, `remediation`, `advisory` | skeleton choice, fragment variants |
| `{cond: planning, eq/any_of}` | `exempt`, `cdc`, `da`, `ssd`, `planning_permit` | approval stages and activities |
| `{cond: procurement_route, eq/any_of}` | `traditional`, `design_construct`, `eci`, `managing_contractor`, `alliance`, `ppp` | where procurement sits; novation |
| `{det: ncc_class, any_of}` | `"1a"`, `"1b"`, `"2"`, `"3"`, `"4"`, `"5"`, `"6"`, `"7a"`, `"7b"`, `"8"`, `"9a"`, `"9b"`, `"9c"`, `"10a"`, `"10b"`, `"10c"` (quote them) | building-class variants |
| `{det: rise_in_storeys, gte/lte/gt/lt: n}` | integer | scale bands (low-rise vs high-rise) |
| `{det: floor_area, gte/lte: n}` | m² | scale bands |
| `{det: bushfire_prone_land, is: true}`, `{det: flood_hazard, is: true}`, `{det: existing_building, is: true}`, `{det: sprinklered, is: true}` | boolean | extra activities |
| `{det: heritage_status, any_of: [...]}` | `none`, `conservation_area`, `local_item`, `state_register` | heritage approvals and works (`heritage` is deprecated; do not use it) |
| `{det: state, eq}` | `NSW`, `VIC`, `QLD`, `SA`, `WA`, `TAS`, `ACT`, `NT` | state-specific approvals |
| `{system_present: <system id>}` | a system id | e.g. lifts, basement retention |
| `{work_action: {system: <id>, any_of: [...]}}` | `new`, `replace`, `upgrade`, `alter`, `repair`, `remove` | **fragments only**: what is being done to that system |

Combine with `all: [...]`, `any: [...]`, `not: {...}`. An unknown fact never
makes a predicate true. More conditions in a skeleton = more specific.

Building-class differences are expressed with `det: ncc_class` (1a houses,
2 apartments, 5 offices, 6 retail, 7b warehouses, 9b schools, and so on).
There is no separate subclass operand.

## What should drive the program (owner-approved)

| Layer | Driven by |
| --- | --- |
| Skeleton (stages, order) | planning pathway, procurement route, work type |
| Activities (fragments) | work items and their actions; systems present; NCC class; heritage, bushfire, flood, existing building |
| Durations | fixed ranges, or scaled by gross floor area, storeys |
| Links | the normal construction sequence (design → approval → documentation → tender → site establishment → substructure → structure → envelope → services rough-in → linings → fit-off → commissioning → completion) |

## Part A: skeletons (`skeletons.yaml`)

Write about 6 to 10 skeletons covering the common combinations. Suggested set
(adjust using your judgement, and explain in the research note):

- `sk.new-traditional-da`: new, traditional, DA
- `sk.new-traditional-cdc`: new, traditional, CDC or exempt (no DA stage; certifier approval)
- `sk.new-dc-da`: new, design and construct, DA (procurement after approval documentation; contractor completes documentation)
- `sk.new-eci-da` or a managing-contractor variant
- `sk.refurb-traditional-da` and `sk.refurb-traditional-cdc`: refurbishment or fit-out (adds investigation; often exempt/CDC)
- `sk.extend-traditional-da`: extension or addition
- `sk.ssd-*`: state significant development for large projects, if you think it differs materially

Stage ids you may use (from `knowledge/works/stages.yaml`; never `other`, never
change `sk.generic`):

`investigation`, `design`, `concept_design`, `approval_documentation`,
`construction_documentation`, `approvals`, `procurement`, `construction`,
`construction_stage_inspections`, `completion`, `defects`

Rules:

- `stages` is one flat ordered list. Use `concept_design`,
  `approval_documentation`, `construction_documentation` rather than `design`.
  Keep `construction` as one stage.
- Code chains neighbouring stages finish-to-start automatically. Add `links`
  only for overlaps or non-neighbour dependencies, for example
  `{from: approvals, to: construction, type: FS, lag: null}` or an SS overlap
  between documentation and procurement. `lag` is `null` or a `du.*` id,
  never a number.
- No two skeletons may have identical `applies_when`.

## Part B: fragments (`fragments.yaml`)

Fragments are where most of the value is. Cover, in priority order:

1. **Construction trades by system**, for new work and for refurbishment, with
   class variants where the work differs materially (1a house vs 2 apartments
   vs 5/6 commercial vs 7b warehouse). For example: site establishment;
   demolition (`work_action ... any_of: [remove]`); bulk earthworks; piles;
   footings; slab on ground; basement retention; concrete or steel or timber
   frame (floor cycle for multi-storey); masonry; roof; windows and doors;
   external walls and façade; waterproofing; hydraulic, electrical and
   mechanical rough-in and fit-off; fire services; lifts; internal walls and
   linings; ceilings; joinery; tiling; painting; floor finishes; landscaping;
   services connections; commissioning.
2. **Refurbishment and fit-out**: strip-out, make-good, staging in an occupied
   building, services alterations.
3. **Design, approval and procurement activities.** See the important
   limitation below.

Fragment shape:

```yaml
- id: fr.slab-on-ground-1a
  label: Slab on ground (house)
  stage: construction
  systems: [substructure.slab-on-ground]
  applies_when:
    all:
      - {work_action: {system: substructure.slab-on-ground, any_of: [new, replace]}}
      - {det: ncc_class, any_of: ["1a"]}
  activities:
    - {key: set-out, label: Set out and excavate slab, systems: [substructure.slab-on-ground], duration: du.slab-setout-1a}
    - {key: under-slab, label: Under-slab services, systems: [hydraulic.sanitary], duration: du.under-slab-services-1a}
    - {key: pour, label: Reinforce and pour slab, systems: [substructure.slab-on-ground, structure.concrete], duration: du.slab-pour-1a}
  links:
    - {from: set-out, to: under-slab, type: FS, lag: null}
    - {from: under-slab, to: pour, type: FS, lag: null}
  attach: {after: fr.footings-1a}
  status: draft
  sources:
    - {design: docs/research/2026-10-21-program-library.md, anchor: "### fr.slab-on-ground-1a"}
```

Rules:

- `id` is `fr.` plus lowercase letters, digits and hyphens. Activity `key`s
  are unique within the fragment.
- `stage` is one of the stage ids above (not `other`).
- `systems` on the fragment and on **every** activity must be ids from the
  system list at the end of this prompt.
- A fragment applies once per matching work item. Make each fragment's
  `systems` the system its work items sit on, so one fragment is not
  duplicated across many items. Prefer one fragment per system per variant.
- A fragment with exactly one activity takes the work item's own title as its
  label, so single-activity fragments are fine for simple trades.
- `duration` is a `du.*` id you define in `durations.yaml`, or `null` if you
  genuinely cannot estimate it.
- `links` join keys within the fragment, with `type` `FS`, `SS`, `FF` or `SF`.
  No cycles. Use an SS link with a `du.*` lag for overlaps (e.g. tiling starts
  some days after linings start).
- `attach: {after: ..., before: ...}` places the fragment after or before the
  stage start/finish (`stage_start`, `stage_finish`) or another fragment id.
  Use it to encode the trade sequence (footings after earthworks, frame after
  slab, roof after frame, and so on). Attach references must not form a cycle.
- Class variants: write separate fragments with different
  `det: ncc_class` conditions (e.g. `fr.frame-timber-1a`,
  `fr.frame-concrete-2`). Each points at its own durations.
- Demolition fragments must use `work_action` with `any_of: [remove]`.
  `retain` and `investigate` may not select a fragment.

**Important limitation, design, approval and procurement activities.**
Today a fragment only applies through a work item on a building system. Pure
process activities (DA lodgement, council assessment, tender period, contract
award, practical completion inspection, occupation certificate) belong to no
building system. The owner is adding a project-level fragment form. Until it
exists:

- write these fragments with an extra field `scope: project`, no
  `work_action`, an `applies_when` over `cond`/`det` facts only, and
  `systems: [site]` on the fragment and its activities;
- the checker and the app's loader **reject** `scope` today, and a rejected
  field stops the app starting. So do not put these fragments in
  `fragments.yaml`. Write them in a section at the end of your research note
  titled "Project-level fragments (pending scope: project)", in the same YAML
  shape, ready to move into `fragments.yaml` once the owner adds the field.
  Their durations may go in `durations.yaml` now.

Cover at least: concept design, design development, approval documentation,
DA lodgement and assessment (DA), CDC application (CDC), construction
certificate, tender documentation, tender period, tender evaluation, contract
award, novation (D&C), site establishment, practical completion inspection,
defects inspection, occupation certificate, handover, and end of defects
liability period.

## Part C: durations (`durations.yaml`)

```yaml
- id: du.slab-pour-1a
  version: 1
  basis: per_unit
  driver: {planning_key: gross_floor_area, unit: m2}
  unit_days: working
  low: "0.02"
  typical: "0.03"
  high: "0.05"
  geography: Sydney metropolitan, NSW
  limitations: [Single-storey slab; excludes wet-weather delays]
  status: draft
  sources:
    - {design: docs/research/2026-10-21-program-library.md, anchor: "### du.slab-pour-1a"}
```

| `basis` | Meaning | Required fields |
| --- | --- | --- |
| `fixed` | the activity takes low/typical/high days | low, typical, high |
| `per_unit` | days per unit × driver (e.g. per m² of floor area); code rounds up | `driver`, low, typical, high |
| `floor_cycle` | days per storey × storeys | `driver: {determinant: rise_in_storeys, unit: storeys}`, low, typical, high |
| `statutory` | a legislated period | low, typical, high, `instrument`, `clause`, `verified: false` |
| `lead_time` | procurement wait for long-lead items (lifts, switchboards, façade) | low, typical, high |

Do **not** write `time_cost` rows.

Drivers you may use: `{planning_key: gross_floor_area, unit: m2}`,
`{planning_key: site_area, unit: m2}`, `{determinant: floor_area, unit: m2}`,
`{determinant: rise_in_storeys, unit: storeys}`.

Rules:

- `id` is `du.` plus lowercase letters, digits and hyphens.
- `low`, `typical`, `high` are **quoted** decimal strings, non-negative, at
  most 2 decimal places, `low ≤ typical ≤ high`.
- `unit_days` is `working` for activities. Use `calendar` only for lags that
  run in calendar time (concrete curing, council periods quoted in calendar
  days).
- Lag rows (used as `lag:` in links) are ordinary `fixed` duration rows.
- Use an optional `applies_when` only to stop a rate being used where it makes
  no sense.
- `geography` is a short text; default "NSW metropolitan" unless you
  deliberately say otherwise. `limitations` is a list of short texts (may be
  empty, `[]`).
- Statutory periods: give your best understanding of the instrument and
  clause, and always `verified: false`.
- Prefer scaled durations (`per_unit`, `floor_cycle`) for trades whose length
  grows with size, and `fixed` for things that don't (approvals, tender
  periods, inspections).
- Leave `status: draft`. Never write `reviewed`.

## Part D (optional): calendars (`calendars.yaml`)

If you are confident of the dates, add `cal.nsw-2026`, `cal.nsw-2027` and
`cal.nsw-2028`: NSW public holidays, the construction industry RDO days, and
the usual Christmas shutdown. Quote every date. `kind` is `public_holiday`,
`rdo` or `shutdown`. `verified: false`. One calendar per state per year. Skip
this part if you are unsure; a wrong holiday is worse than none.

```yaml
- id: cal.nsw-2027
  state: NSW
  year: 2027
  working_week: [mon, tue, wed, thu, fri]
  days:
    - {date: "2027-01-01", kind: public_holiday, label: New Year's Day}
    - {from: "2027-12-20", to: "2027-12-31", kind: shutdown, label: Christmas shutdown}
  verified: false
  status: draft
  sources:
    - {design: docs/research/2026-10-21-program-library.md, anchor: "### cal.nsw-2027"}
```

## The research note (required)

Write `docs/research/2026-10-21-program-library.md`. Every row in the YAML
files cites a heading in this note, and the checker verifies the heading line
exists exactly, so:

- give **each skeleton, fragment, duration and calendar its own heading**,
  exactly `### <id>` (e.g. `### du.slab-pour-1a`), and cite it as
  `{design: docs/research/2026-10-21-program-library.md, anchor: "### <id>"}`;
- under each heading, write two to five lines: what it covers, why these
  stages, activities or numbers, the basis (general industry practice,
  typical Australian residential build times, a standard or regulation you
  have in mind, a published rule of thumb), and any assumption. Say plainly
  when a number is an estimate;
- start the note with a short overview: the skeleton set and how the facts
  select them, the class and scale variants you chose, and the known gaps;
- end it with the "Project-level fragments (pending scope: project)" section
  if Part B needed it.

The note's own top-level `sources` line in each YAML file can point at
`### Overview`.

## File-level shape

Each YAML file is a mapping:

```yaml
version: 1
status: draft
sources:
  - {design: docs/research/2026-10-21-program-library.md, anchor: "### Overview"}
notes: <optional one line>
skeletons:   # or fragments: / durations: / calendars:
  - ...
```

`skeletons.yaml` already contains `sk.generic`. Keep it exactly as it is and
add yours after it.

## YAML traps

- Quote every date and every decimal string.
- Quote NCC classes: `"1a"`, `"2"`, `"5"`.
- Never use bare `on`, `off`, `yes` or `no` as a value or a label.
- Keys are checked strictly. A misspelt key is an error; do not invent keys
  (`scope: project` goes only in the research note, see Part B).

## Check your work

Run, from the repository root, and fix every error until it passes:

```
python tools/check_knowledge.py --strict
```

It checks ids, references to stages, systems, fragments and durations,
predicate operators, decimal rules, statutory fields, link cycles, calendar
dates, and that every cited heading exists in your note. The 2 existing
warnings about `package_defaults` are not yours to fix.

## Scope and size

Aim for complete coverage of three starting project types first, then widen
if time allows:

1. New Class 1a house (traditional, DA or CDC), single to two storeys.
2. New Class 2 apartments (traditional or D&C, DA), 3 to 12 storeys, with
   basement.
3. Class 5/6 commercial fit-out (refurb, exempt or CDC), in an existing
   building.

Then extensions or additions to houses, warehouses (7b), and schools (9b).

For each starting type, a built program should have every construction-stage
trade it normally needs, durations on the critical chain, and sensible links.

## Final report

When the checker passes, reply with:

1. The skeletons you wrote and the facts that select each.
2. A table of fragments by stage and class.
3. The number of durations by basis.
4. Anything you deliberately left out or could not express, and the list of
   project-level fragments written in the research note.
5. The checker's final output line.

## System ids you may use

Top level: `site`, `substructure`, `structure`, `envelope`, `interiors`,
`fire-passive`, `fire-active`, `mechanical`, `hydraulic`, `electrical`,
`comms-security`, `vertical-transport`, `access-egress`

Detailed:

`access-egress.accessible-access`, `access-egress.accessible-facilities`,
`access-egress.barriers`, `access-egress.egress-doors`, `access-egress.exits`,
`access-egress.livable-housing`, `access-egress.maintenance-access`,
`access-egress.pool-barriers`, `access-egress.stairs-and-ramps`,
`comms-security.access-control`, `comms-security.audiovisual`,
`comms-security.carrier-lead-in`, `comms-security.cctv`,
`comms-security.ict-rooms-pathways`, `comms-security.intruder-alarm`,
`comms-security.networks-integration`, `comms-security.structured-cabling`,
`electrical.battery-storage`, `electrical.cabling-wiring`,
`electrical.earthing-lightning`, `electrical.ev-charging`,
`electrical.lighting`, `electrical.metering`, `electrical.process-hazardous`,
`electrical.solar-pv`, `electrical.standby-power`, `electrical.switchboards`,
`electrical.temporary-power`, `electrical.utility-supply`,
`envelope.building-sealing`, `envelope.bushfire-construction`,
`envelope.curtain-wall`, `envelope.external-shading`,
`envelope.external-walls`, `envelope.external-waterproofing`,
`envelope.facade-acoustics`, `envelope.flashings-seals`,
`envelope.pliable-membranes`, `envelope.roof-coverings`,
`envelope.thermal-insulation`, `envelope.thermal-performance`,
`envelope.vehicle-doors`, `envelope.windows-doors`,
`fire-active.detection`, `fire-active.emergency-lighting`,
`fire-active.fire-control`, `fire-active.fire-water`, `fire-active.hydrants`,
`fire-active.occupant-warning`, `fire-active.portable-extinguishers`,
`fire-active.smoke-alarms`, `fire-active.smoke-control`,
`fire-active.special-suppression`, `fire-active.sprinklers`,
`fire-passive.bushfire-construction`, `fire-passive.compartments`,
`fire-passive.external-walls`, `fire-passive.linings-fire-hazard`,
`fire-passive.openings`, `fire-passive.penetrations`,
`fire-passive.rated-elements`, `fire-passive.structural-fire-protection`,
`hydraulic.backflow-prevention`, `hydraulic.cold-water`, `hydraulic.gas`,
`hydraulic.hot-water`, `hydraulic.non-potable-water`,
`hydraulic.roof-drainage`, `hydraulic.sanitary`,
`hydraulic.specialist-process`, `hydraulic.trade-waste`,
`interiors.acoustic-separation`, `interiors.ceilings`, `interiors.ffe`,
`interiors.floor-finishes`, `interiors.internal-doors`,
`interiors.internal-glazing`, `interiors.joinery`, `interiors.painting`,
`interiors.room-acoustics`, `interiors.tiling`, `interiors.walls-linings`,
`interiors.wet-area-waterproofing`, `mechanical.air-conditioning`,
`mechanical.air-distribution`, `mechanical.carpark-ventilation`,
`mechanical.central-plant`, `mechanical.controls-bms`,
`mechanical.cooling-towers`, `mechanical.fire-mode-air-systems`,
`mechanical.heating-appliances`, `mechanical.local-exhaust`,
`mechanical.process-specialist`, `mechanical.refrigerant-condensate`,
`mechanical.ventilation`, `site.acid-sulfate-soils`,
`site.adjoining-structures`, `site.contamination`, `site.demolition`,
`site.earthworks`, `site.erosion-sediment-control`, `site.flooding`,
`site.ground-conditions`, `site.impact-protection`, `site.landscape`,
`site.loading-docks`, `site.pools`, `site.public-domain`,
`site.retaining-walls`, `site.services-connections`, `site.signage`,
`site.stormwater-detention`, `site.survey-boundaries`, `site.trees`,
`site.vehicle-access`, `site.waste-storage`, `structure.alterations`,
`structure.concrete`, `structure.design-basis`, `structure.masonry`,
`structure.precast`, `structure.steel`, `structure.temporary-works`,
`structure.timber`, `substructure.basement-retention`,
`substructure.damp-waterproofing`, `substructure.footings`,
`substructure.piles`, `substructure.retaining-walls`,
`substructure.slab-on-ground`, `substructure.termite-management`,
`substructure.underpinning`, `vertical-transport.escalators-walks`,
`vertical-transport.goods-lifts`, `vertical-transport.lift-fire-service`,
`vertical-transport.lift-shafts-pits`, `vertical-transport.passenger-lifts`,
`vertical-transport.platform-lifts`
