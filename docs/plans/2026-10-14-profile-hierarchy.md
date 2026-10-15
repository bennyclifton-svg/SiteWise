# Profile hierarchy: the Seven Hills layout over every Petersham field

Lane B (layout). No field, endpoint, band or evidence affordance is removed.
Reference: Clerk `TaxonomyPicker.tsx` (structure only; no code copied).

## Problem

The current Project card is three equal columns of label/control rows
(The building · Site and planning · Delivery). Every field has the same
weight, so the four answers that define the project (what class, what work,
what kind, how big) sit at the same level as the consent lapse date. The
Seven Hills page reads top down instead: identity, then class, then work,
then the detail of that kind of building.

## Target hierarchy

| Tier | Seven Hills | Current fields mapped to it |
|---|---|---|
| 0 Title | Project name + address under it | Banner (unchanged) |
| 1 Identity | Site address · Client · State | `site.address`, `proj.client`, `det.state`; quiet second line: `site.lot`, `proj.lga`, `proj.zone` |
| 2 Class | All classes as chips, one selected | `hdr.building_class` (all options visible) |
| 3 Work type + Budget | All work types as chips; budget at right | `hdr.work_type` chips; `proj.budget_stated` |
| 4a Subclass card | Radio list for the chosen class | `hdr.subclass` radios; `det.ncc_class` beneath as "NCC class" (Petersham conflict 2 vs class choice stays visible) |
| 4b Scale card | Stacked inputs, typical as placeholder | `hdr.scale.*` (SAME_MEASURE dedupe kept), `det.rise_in_storeys`, `det.floor_area`, `det.effective_height`, `det.existing_building(_year)` |
| 4c Complexity card (2 cols) | Planning, procurement, constraints | all `hdr.cond.*` incl. planning, procurement route, quality level, stakeholder/operational/access/environmental; flood/bushfire/heritage (`det.heritage_status`) join here |
| 5 Approvals strip | (not in Seven Hills) | `fact.consent_number`, `fact.consent_date`, `fact.consent_lapse_date`, `det.ncc_edition` |
| 6 Delivery strip | (not in Seven Hills) | `fact.contract_form`, `fact.contract_basis`, `fact.defects_liability_period`, `fact.design_life`, `proj.program`, `proj.sustainability`, `proj.regulated_design`, `proj.flag.*`, other `home: project` inputs |

Tiers 5–6 sit below the cards, collapsed to "filled + essential" with
"+ n more", as now. Unmapped future keys fall into tier 6 so nothing hides.

## Design moves (the hierarchy lessons)

1. **Labels above values**, not beside them. Frees width; the value reads first.
2. **Choices you can see.** Class and work type become chip rows (pressed =
   selected); subclass is a radio list. You see what it is *and* what it isn't.
3. **Provenance moves off the value line.** Mark + source count sit at the
   label's right end as a small badge; the control stays clean. Red still
   tints the control border and shows "Use X (n)" picks under it. Reset ↺
   stays. Chips carry their band as a border tint + sr-only text.
4. **Section titles as small caps rules** (`— CLASS`, `— WORK TYPE`) like
   Seven Hills; cards only for the three tier-4 groups.
5. **One status line.** Reading/coverage/threshold notes collapse into one
   quiet line with the progress bar; "Update project profile" moves to the
   top-right of the card (where "Save profile" was). Parts bar sits beside it.
6. **Width ladder:** tier-4 cards are 1fr · 1fr · 2fr on wide screens,
   stacking on narrow.

## Out of scope

Scope/works panes, API changes, new fields, Clerk's starter work-scope.
Typical ranges (`typical:` in taxonomy) as placeholders need one API field
(`typical`) — listed as optional slice 3, lane A-adjacent, only if you agree.

## Slices

1. Identity + class chips + work-type chips + budget (Profile.tsx, new
   `ProfileChoice` chip/radio components reusing Mark/Sources/Conflict).
2. Subclass / Scale / Complexity cards; approvals + delivery strips; label-above
   field row variant (`FieldRow stacked`).
3. (Optional) `typical` placeholders from taxonomy.

Done = every field present today is reachable on Petersham, red checks still
jump/focus, `web/tests/profile.spec.ts` passes (selectors updated), light and
dark checked in browser.

## Open decisions

- O1: Changing class: chip click writes immediately (as selects do now)? Proposed yes.
- O2: NCC class under Subclass, or in Scale? Proposed Subclass.
- O3: Heritage/flood/bushfire in Complexity (Seven Hills) vs Approvals? Proposed Complexity.
- O4: Slice 3 typical placeholders: yes/no.
