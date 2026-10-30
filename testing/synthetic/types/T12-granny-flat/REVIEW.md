# T12 Secondary dwelling (granny flat): independent review

Reviewer brief: `testing/synthetic/briefs/02-review.md`. Read 30 October 2026 against the working tree.
This is the only file written. I ran the real Go generators (`program.Generate`, `rfp.Resolve`) through
`go run -overlay`, with the overlay source kept in my scratchpad, so nothing was added to the repo. No Jev
was called. I checked five NSW questions on the web, and each source is cited where it is used.

## Verdict: **FIX**

The map is right at the root. The base scenario, the consultant team, the over-procurement story, the
CDC skeleton, the existing-house interfaces and the trigger set all suit a backyard granny flat in
Sydney. It is not yet a reliable answer key, for four reasons:

- it cites the wrong instrument for the CDC standards;
- it marks one app hold point as missing when the app does make it;
- four RFP `expect` blocks disagree with what the app produces, and the disagreement is not in
  `app_differs`;
- it does not apply owner ruling R1 to the garage-conversion variant.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T12-granny-flat/shape.yaml`, run
from the repo root, gives `ok`, exit 0, with no warnings.

Run from `testing/synthetic/` instead, it fails with
`FileNotFoundError: knowledge\profile\taxonomy.yaml`, because `load_ids(Path("."))` assumes the repo root.
That is a validator problem, not a map problem. The brief's command should say "from the repo root".

## 2. Rulings on the author's open questions

| Question | Ruling |
|---|---|
| Base procurement: designer plus quoted builders, or turnkey | **Keep the designer-led base.** A turnkey package is at least as common in the market, but the designer-led version is the one a SiteWise PM runs, and it gives each consultant an RFP to test. Turnkey stays the first variant (`builder-designs`). §1 already says this; it is now ruled, so drop it from "claims I am least sure of". |
| CDC lot and setback standards and exclusions | **Wrong instrument (F1).** NSW Planning places the secondary-dwelling CDC standards in **SEPP (Housing) 2021, Schedule 1**, not the Codes SEPP Housing Code. The Codes SEPP covers the principal dwelling. The 450 m² minimum lot is confirmed by NSW Planning, "Secondary dwellings", https://www.planning.nsw.gov.au/the-planning-system/housing/housing-sepp/secondary-dwellings. I could not confirm the 12 m width, the 900 mm side and 3 m rear setbacks, or the exclusion list against the instrument, so they stay `verified: false`. The exclusions as mapped are a sound working set: a heritage item is excluded; treat a conservation area and BAL-40/FZ as a DA; a flood control lot is restricted, not barred. |
| Neighbour notice and Tap in timing | **Two notices, two actors (F6).** In metropolitan areas the certifier notifies neighbours within 20 m, and cannot determine the CDC until 14 days after that notice. The person with the benefit of the CDC gives neighbours written notice at least 7 days before work starts. Source: NSW Planning Portal, "What to tell your neighbours", https://www.planningportal.nsw.gov.au/development-and-assessment/planning-approval-pathways/complying-development/what-tell-your-neighbours. The Regulation 2021 clause numbers stay `verified: false`. **Tap in as mapped is right:** stamped building-plan approval goes with the CDC set. The firm requirement is before work starts, which matches the repo's own `rule.water-authority.build-over-sewer`. |
| Section 73 not needed | **Confirmed for the base (F7).** Sydney Water says a secondary dwelling approved as complying development under the Housing SEPP does not need a Section 73 certificate, but building plan approval is still required. The exception is a Priority Sewerage or Pressure Sewer Scheme area. Source: Sydney Water, "Section 73 requirements for secondary dwellings", https://www.sydneywater.com.au/content/dam/sydneywater/documents/section-73-requirements-for-secondary-dwellings.pdf. |
| Fire separation distances | **Right as mapped. Keep `verified: false`** until someone checks the ABCB Housing Provisions Part 9.2 text. An external wall needs fire-resisting construction when it is within 900 mm of an allotment boundary or within 1.8 m of another building on the same allotment, other than an appurtenant Class 10 building or a detached part of the same Class 1 building. The main house is a separate Class 1a building, so the 1.8 m applies between the flat and the house. The trigger evidence needs one fix (F5). |
| Long service levy threshold | **$250,000 or more, including GST, since 1 January 2023, at 0.25%.** Sources: Long Service Corporation, https://www.longservice.nsw.gov.au/bci/levy/about-the-levy; Building and Construction Industry Long Service Payments Regulation 2022. The T12 value range of $0.15–0.4M straddles the threshold, so the line is conditional and the scenario has to state the contract value (F8). The figure can move out of `verified: false` once the owner records it as a ruling. |
| Owner ruling R1 (civil when roof or impervious area changes) | **Civil is in the base. Correct.** A new detached flat adds a whole roof and paths, so the catchment always changes. R1's only exit is a variant whose documents say the roof footprint and hard surfaces are unchanged. For T12 that is the **conversion of an existing garage** inside its own walls and roof. The map does not apply that exit (F4). `builder-designs` also takes civil out of the *owner's* procurement, which is correct: the builder still engages a stormwater engineer, so R1 holds. |

## 3. Findings

### F1. The CDC standards are cited to the wrong instrument. Severity: high

- **Where:** `DOMAIN.md` §5 "Complying development"; §12, the instruments list and "Claims I am least
  sure of".
- **What is wrong:** the map says the secondary-dwelling CDC standards "moved from the Affordable Rental
  Housing SEPP 2009 into the Codes SEPP Housing Code when SEPP (Housing) 2021 commenced". NSW Planning
  says they are in **Schedule 1 of SEPP (Housing) 2021**, and that the principal dwelling's standards are
  in the Codes SEPP. Scenario documents written from the map would cite the wrong instrument, and a
  certifier's eligibility letter would not.
- **Evidence:** NSW Planning, "Secondary dwellings" (URL in §2). That page also confirms the 450 m²
  minimum lot, with an exception for a flat wholly inside an existing dwelling house.
- **Exact change:** in §5, replace the "Complying development" bullet with: "The complying development
  standards for secondary dwellings are in SEPP (Housing) 2021, Schedule 1 (NSW Planning, Secondary
  dwellings page). The lot must be at least 450 m² (confirmed by that page). Lot width, setbacks (900 mm
  side and 3 m rear recalled for a single storey), height, landscaped area and private open space are
  `verified: false`. The principal dwelling's standards are in the Codes SEPP." In §12, move "Housing
  Code (secondary dwelling CDC standards …)" under SEPP (Housing) 2021 Schedule 1. In "least sure of",
  keep only the width, setbacks and exclusions.

### F2. The reinforcement hold point does fire. Severity: high (and it also affects T01, T05 and T10)

- **Where:** `shape.yaml program.activities`, "Reinforcement inspection before slab pour"
  (`built_in: "partial"`, why "the hold point needs a structure.concrete item, so it does not fire");
  `DOMAIN.md` §5, the last paragraph; §7.
- **What is wrong:** `holdPointRows` (`internal/program/generate.go`) needs *rows tagged* with the
  `after` and `before` systems, not work items on them. `fr.slab-on-ground-1a` tags its "Reinforce and
  pour slab" activity with `[substructure.slab-on-ground, structure.concrete]`
  (`knowledge/program/fragments.yaml`), so `if.reinforcement-inspected-before-pour`
  (`after: structure.concrete, before: substructure.slab-on-ground`) resolves.
- **Evidence:** `program.Generate` on the T12 base (40 items, cdc, traditional, 1a) gave
  `hold_point | construction | Hold point: Reinforcement inspection release identifying the pour.` next to
  the frame hold point.
- **Exact change:** set `built_in: "yes"` and why to "App hold point 'Hold point: Reinforcement
  inspection release identifying the pour.' (tagged rows from fr.slab-on-ground-1a); R7 item 3". In §5,
  write "The app makes two hold points: frame (`if.frame-inspected-before-roof`) and reinforcement
  (`if.reinforcement-inspected-before-pour`, which fires through the slab fragment's `structure.concrete`
  tag)". In §7 "What the app gets wrong", remove the slab pour from the list of missing inspections. The
  T01, T05 and T10 shapes have the same `partial` row, and the T01 and T05 reviews passed it. The
  orchestrator should correct those maps too.

### F3. Four RFP `expect` blocks disagree with the app, and the difference is not in `app_differs`. Severity: medium

`rfp.Resolve` was run for each template with the base drivers. The drivers were the `profile` flags as
set, `live_operations` yes, `existing_building` no, a whole part of class 1a with 60 m², and every other
skeleton flag unknown. Architecture and land surveying match `expect` exactly. The others do not:

| Template | Key | App | `expect` | Exact change |
|---|---|---|---|---|
| structural | `position.inspection_schedule` | I | O | Set `expect` to `I`. The row is I whenever `construction_phase` is chosen (`knowledge/rfp/structural.yaml inspection_schedule`), and that stage is already an option, so I is correct. |
| civil | `stage.for_construction` | option | include | Move `for_construction` to `stages_option`. Its deliverable is "construction set; construction certificate issue". Under a CDC there is no CC, and the `detailed_tender` set that goes in with the CDC is the construction issue for a 60 m² flat. |
| geotechnical | `stage.g4_planning` | option | absent | Add `{key: stage.g4_planning, app: option, expect: absent, why: "after the boundary end; a CDC has no planning-stage geotechnical submission"}`. |
| bca-certification | `stage.a1_inception`, `stage.a3_tender` | ask, ask | absent | Add two `app_differs` rows: `app: ask`, why "client flag bca_advisory cannot be keyed". |
| bca-certification | `stage.a4_construction` | option | absent | Add `app_differs` with `app: option`, why "after the boundary end; no BCA advisory on a granny flat". |

The `pathway_unsupported` block, the absent B stages and the other `app_differs` rows were all confirmed.

### F4. R1 is not applied to the garage-conversion variant. Severity: medium

- **Where:** `shape.yaml triggers[attached-or-conversion]`; `DOMAIN.md` §9.
- **What is wrong:** one trigger holds two jobs that differ under R1:
  - **attached to the rear of the house:** new roof, so civil stays;
  - **conversion of the existing garage inside its own envelope:** no new roof or hard surface, so
    civil drops.

  The trigger has no `removes`. The two forms also differ in the app. A conversion is work type `refurb`,
  so the scope defaults are empty and only `baselines[2]` applies (Architect, Certifier, Electrical,
  Hydraulic; `knowledge/works/package_defaults.yaml`). An attached flat is `extend`, which uses the house
  extend list `classes[2]`. One trigger cannot predict both.
- **Exact change:** split it into two triggers.
  - `attached-secondary-dwelling`: `hdr.work_type: extend` and `det.existing_building: true`. Keep the
    current adds, keep civil, and keep `rfps` `yes` from `knowledge/rfp/structural.yaml
    existing_assessment`. The Go run with extend gives `existing_assessment` include and the
    `existing_building` module on.
  - `garage-conversion`: `hdr.work_type: refurb` and `det.existing_building: true`.
    - Evidence: "convert the existing double garage into a granny flat"; "the conversion is within the
      existing garage walls and roof; no new hard surfaces".
    - `removes`: procurement `[consultant.civil]`, cost_plan `["Civil (stormwater) engineer"]`, rfps
      `[consultant.civil]`.
    - why: "owner ruling R1: roof footprint and hard surfaces unchanged".
    - Adds: hazmat, change of use from Class 10a to 1a, an existing-structure assessment, and an
      upgrade of the slab, damp-proofing and insulation.
    - Note in the trigger that refurb scope starts empty (KNOWLEDGE_INDEX §8.2).
  - Put `garage-conversion` in `variants`. It is common in Sydney, and the R1 drop is testable.

### F5. The `close-to-main-house` evidence also breaks the CDC side setback. Severity: medium

- **Where:** `shape.yaml triggers[close-to-main-house].evidence[2]`, "the flat's wall is 800 mm from the
  side boundary"; the same phrase in `DOMAIN.md` §9.
- **What is wrong:** 800 mm is inside the recalled 900 mm CDC side setback. A document that says it also
  means `da-pathway`. The variant would carry two changes, and its "one-change" test fails.
- **Exact change:** replace it with "The secondary dwelling's rear wall is 1.2 m from the main house's
  laundry wall". Add the eaves to the `adds.cost_plan` line: "Fire-resisting external walls, protected
  openings and non-combustible eaves facing the main house (Housing Provisions Part 9.2, verified:
  false)". Fire on a boundary under 900 mm belongs in a DA variant, not here.

### F6. The two neighbour notices are merged and given to the wrong actor. Severity: medium

- **Where:** `shape.yaml program.activities`, "Notify neighbours of the CDC application" and "Appoint
  principal certifier and give notice of commencement" (whose why says "neighbours also get notice before
  work starts"); `DOMAIN.md` §5 "Neighbour notice" and §7 steps 4 and 6.
- **What is wrong:** the certifier gives the pre-determination notice, and determination cannot follow
  for 14 days in a metropolitan area. The CDC holder gives the pre-commencement notice at least 7 days
  before work (Planning Portal, §2). The second notice has no row of its own, so a scenario cannot test
  for it.
- **Exact change:**
  1. Relabel the first row "Certifier notifies neighbours within 20 m (14 days before determination)",
     with why "certifier's notice; metropolitan area; Regulation clause verified: false".
  2. Add `{label: "Give neighbours notice of commencement (at least 7 days before work)", stage:
     procurement, fragment: null, after: ["Certifier determination"], built_in: "no", why: "CDC holder's
     written notice; Planning Portal 'What to tell your neighbours'"}`.
  3. Add that row to the `after` list of "Establish the site".
  4. Drop "neighbours also get notice" from the principal-certifier row's why.

  In §5, write out both notices and who gives each.

### F7. The Section 73 ruling is confirmed, and its exception is missing. Severity: low

- **Where:** `DOMAIN.md` §5 "Authority applications"; §10 trap 9; §9 (no trigger).
- **Exact change:** in §5, replace "usually **not** needed … (verified: false)" with "not needed for a
  secondary dwelling approved as complying development under the Housing SEPP (Sydney Water, 'Section 73
  requirements for secondary dwellings'); building plan approval is still needed. The exception is a lot
  in a Priority Sewerage or Pressure Sewer Scheme area." Add a trigger `priority-sewerage-area`:
  - evidence "The property is in a Sydney Water Priority Sewerage Program area"; "pressure sewer scheme";
  - negation "Gravity sewer main available in the street";
  - adds program "Section 73 compliance certificate through a Water Servicing Coordinator";
  - adds cost_plan "Section 73 and Water Servicing Coordinator fees";
  - adds authority "Section 73 certificate";
  - `built_in: {program: "no", cost_plan: "partial"}`, source `knowledge/costs/starter_lines.yaml
    service-authority`.

  Trap 9 stands: the app's `rule.nsw.section-73-certificate` says "most new developments".

### F8. The long service levy line should carry the confirmed threshold. Severity: low

- **Where:** `shape.yaml cost_plan`, "Long service levy (above the threshold, verified: false)" and the
  activity "Pay long service levy"; `DOMAIN.md` §5 and §12.
- **Exact change:**
  - Relabel the line "Long service levy (cost of work $250,000 or more incl. GST, 0.25%; Long Service
    Corporation)". Mark the figure for owner confirmation rather than `verified: false`, if the owner
    agrees.
  - Add a `scenario_inputs` row `{key: proj.budget_stated, note: "the levy line and row apply only when
    the stated cost of work is $250,000 or more incl. GST; state which side", phrasing: "Construction
    budget: $<amount> excluding GST"}`. Alternatively, extend the existing budget row's note to say this.
  - Change the program row's why to "Before the CDC issues, where the cost of work is $250,000 or more
    incl. GST".

### F9. Common triggers are missing. Severity: low

Add these, or say in §9 why they are left out:

1. **`accessible-for-ageing-parent`.** Many granny flats are for an elderly parent. Evidence: "step-free
   shower and wider doors for Mum"; "designed to the Livable Housing silver level". Negation: "No
   accessibility features are required".
   - The app consequence is visible. A document naming livable housing adds
     `access-egress.livable-housing`, which makes `scope_additions[5]` suggest an **Access consultant**
     (`knowledge/works/package_defaults.yaml`). Under R6 that is over-procurement for a Class 1a
     dwelling.
   - The key's answer: no access consultant, the designer covers it.
   - Put the Access consultant in this trigger's `not_expected` notes.
2. **`priority-sewerage-area`**, from F7.
3. **`unapproved-existing-structure`.** Evidence: "the rear studio was built without approval".
   Negation: "all structures on the lot are approved". It adds a building information certificate, or
   removal, before the CDC, and the app reaches it only through
   `uc.existing-building-or-addition-has-no-approval` (Implications). §3 already describes this; it
   needs to be a trigger to be testable.

### F10. App rows and RFP behaviour that are missing or overstated. Severity: low

- `shape.yaml program.activities`: add `{label: "Coordinate consultant reports", stage:
  approval_documentation, fragment: fr.approval-docs-1a, after: ["Develop the design"], built_in: "yes",
  source: "knowledge/program/fragments.yaml fr.approval-docs-1a", why: "Structural, stormwater and BASIX
  reports for the CDC set"}`. The app produces it (Go run), and leaving it out makes the scorer report a
  surplus row.
- `triggers[trees-affected].built_in.rfps`: change `"yes"` to `"partial"`. With
  `flag.arborist_tree_impacts` yes, `s2_impact` and `s3_protection` resolve to **option**, not include
  (Go run: "Works affect trees roots or canopy"). The arborist RFP also exists only if the user creates
  the package, because no rule suggests one.
- `triggers[sewer-main-in-backyard]`: add to why "the authority-asset flag makes the structural scale
  complex (Go run), the app's over-scoping for a granny-flat slab beside a main".
  `triggers[flood-affected]` does the same to the civil scale. Note it there too.
- `triggers[trees-affected].negation[1]`: "a large gum in the front yard (present but unaffected)" is
  not document phrasing. Use "The large gum in the front yard is well clear of the works".

### F11. Small text fixes. Severity: low

- `DOMAIN.md` §12 "Claims I am least sure of": remove the base-procurement, Section 73, levy and
  `resolveStage` items, now ruled or confirmed by the Go run: `fr.construction-docs-1a` lands in
  approval documentation, together with "Client sign-off for tender". Keep the CDC width, setbacks and
  exclusions, and keep the fire distances.
- `DOMAIN.md` §5 Certificates: the principal certifier row says "Whether a builder may appoint on the
  owner's behalf: verified: false". `builder-designs` already states the answer (the owner appoints).
  Make the two consistent and keep it `verified: false`.

## 4. Built-in claims checked

**Procurement**

- ✔ Architect, Surveyor, Geotechnical, Structural, Civil, Certifier: `baselines[0]` (new, not
  infrastructure).
- ✔ `trade.main_works` partial: `ProposeHeadContract` only over accepted items (KNOWLEDGE_INDEX §2.4).
- ✔ `not_expected`: Town Planner, Electrical and Hydraulic from `baselines[0]`; Mechanical from
  `scope_additions[2]`.

**Cost plan**

- ✔ CDC fee partial (`cc-fees`, `applies_when: true`).
- ✔ Sydney Water partial (`service-authority`).
- ✔ The six F-lines.
- ✔ Elements 01, 02, 03, 04, 05, 08, 09, 10, 11 (`knowledge/costs/elements.yaml`).
- ✔ The four contingency starters.
- ✔ Owner supply partial.
- ✔ `not_expected` 07 from smoke alarms.

**Program** (all from the Go run)

- ✔ `sk.new-traditional-cdc`, with no construction documentation stage.
- ✔ `fr.investigate-survey` fires on `investigate`.
- ✔ Geotechnical partial.
- ✔ Concept and design development.
- ✔ `fr.construction-docs-1a` placed in approval documentation.
- ✔ `fr.approval-docs-1a`.
- ✔ `fr.cdc-application` (both rows).
- ✔ Tender documents, tender period, evaluation, award.
- ✔ Site establishment.
- ✔ Erosion control, earthworks.
- ✔ Footings ("Excavate and inspect footings").
- ✔ Slab and under-slab drainage.
- ✔ Termite barrier.
- **✘ Reinforcement hold point marked partial: it fires (F2).**
- ✔ Frame and trusses.
- ✔ Frame hold point.
- ✔ Roof, gutters.
- ✔ Windows (deliver, install).
- ✔ External walls.
- ✔ Rough-in.
- ✔ `fr.services-connections-1a` partial: fires on `alter`, framed as a lead-in.
- ✔ Insulation, linings, ceilings, doors.
- ✔ Wet areas, tiling.
- ✔ Joinery.
- ✔ Switchboard, AC, fit-off.
- ✔ Painting, floors.
- ✔ Stormwater, landscape.
- ✔ PC, OC, handover.
- ✔ Defects inspection, DLP.
- ✔ `fr.utility-1a` fires from the default `electrical.utility-supply` item (gap claim).

**RFPs** (from `rfp.Resolve`)

- ✔ architecture: the full `expect` matches, including `planning_approval` on cdc, `lodgement` I,
  `live_staging` I, and blocking on `fact:budget`.
- ✘ structural: `inspection_schedule` (F3).
- ✘ civil: `for_construction` (F3). `roads`, `retained_services`, `water_quality`, `civil_model`,
  `pavement_design` and `as_built` in `app_differs` are all ✔.
- ✘ geotechnical: `g4_planning` (F3).
- ✔ land-surveying: matches, with `access` I from `live_operations`.
- ✘ bca-certification: the A stages are not in `app_differs` (F3). `pathway_unsupported` blocking ✔.

**Triggers**

- ✔ `da-pathway`: `fr.da-lodgement` and the skeleton (index §4.2, §6).
- ✔ `sewer-main-in-backyard` rfps: Go run gives `authority_asset` include and `water_asset` O.
- ✔ `osd-required` rfps: `osd_new` I.
- **✘ `trees-affected` rfps yes: should be partial (F10).**
- ✔ `electrical-supply-upgrade` program: `fr.utility-1a` accepts `upgrade`.
- ✔ `old-shed-demolition` partial: `implied_by strip-out` regex `demoli(?:tion|sh)`; `fr.demolition-1a`
  needs `remove`.
- ✔ `close-to-main-house` cost partial (code 06).
- ✔ `attached-or-conversion` rfps: `existing_assessment` include on extend.
- ✔ `sloping-backyard` partial (code 01; 13.3 barriers).
- ✔ `bushfire-prone`: `complexity_additions[5]` and `fr.bushfire-1a`.
- ✔ `flood-affected`: `flood_report` I and `fr.flood-1a`.

**Other `[K]` claims**

- ✔ `uc.secondary-dwelling-fire-separation` needs extend and an existing building.
- ✔ `fr.occupied-protection` needs refurb and an existing building.
- ✔ The regexes behind traps 1–6 and 10: `existing_building`, `multi_building`, `strip-out`,
  `party-wall`, `authority_asset`, `excavation_adjoining`, `architect_lodges`.
- ✔ `rule.nsw.section-73-certificate`, `rule.water-authority.build-over-sewer` and
  `rule.basix.certificate-required` all carry `clause_verified: false`.

**`no` claims spot-checked**

- ✔ BASIX, Arborist, Hazmat and Building consultant are never suggested (no rule in
  `package_defaults.yaml`).
- ✔ No long service levy, HBCF or contributions-on-CDC starter (`starter_lines.yaml`).
- ✔ No fragment for the planning certificate, Tap in, neighbour notice, levy, HBCF or commencement
  (grep of `fragments.yaml`).
- ✔ No 1a piles fragment: only `fr.piles-2` and `fr.piles-9b`.
- ✔ No skeleton for new + D&C + CDC (`skeletons.yaml`).

## Round 1 response (author, 30 October 2026)

All eleven findings are applied; none is disputed. The validator, run from the repo root, gives `ok`.

| Finding | Applied |
|---|---|
| F1 | DOMAIN §5 now cites SEPP (Housing) 2021 Schedule 1 for the secondary-dwelling CDC standards, with the 450 m² lot confirmed and the width and setbacks `verified: false`; the Codes SEPP is named for the principal dwelling only. §12 instruments and "least sure of" list updated. |
| F2 | "Reinforcement inspection before slab pour" is `built_in: "yes"`, with the app's hold-point label and the `fr.slab-on-ground-1a` tag as the why. DOMAIN §5, §7 and §11 now name two app hold points. The T01, T05 and T10 rows are outside this folder and are left to the orchestrator. |
| F3 | Structural `inspection_schedule: I`. Civil `for_construction` moved to `stages_option`, with a source note. Geotechnical `app_differs` adds `stage.g4_planning`. bca-certification `app_differs` adds `a1_inception` and `a3_tender` (ask) and `a4_construction` (option). |
| F4 | Split into `attached-secondary-dwelling` (extend, civil kept, structural `existing_assessment` yes) and `garage-conversion` (refurb, `removes` civil from procurement, cost plan and RFPs under R1, with a note that refurb scope starts empty and only `baselines[2]` applies). `garage-conversion` is in `variants`. Civil stays in the base. |
| F5 | The 800 mm evidence is replaced by "The secondary dwelling's rear wall is 1.2 m from the main house's laundry wall"; the cost line names non-combustible eaves facing the house. The why says a wall under 900 mm from a boundary belongs in a DA variant. |
| F6 | Two rows: "Certifier notifies neighbours within 20 m (14 days before determination)" and "Give neighbours notice of commencement (at least 7 days before work)", which "Establish the site" now follows. The principal-certifier row no longer mentions neighbours. DOMAIN §5 and §7 give each notice and its actor. |
| F7 | DOMAIN §5 cites Sydney Water: no Section 73 for a CDC secondary dwelling, building plan approval still needed, with the Priority Sewerage and pressure sewer exception. New trigger `priority-sewerage-area` as specified. Trap 9 kept. |
| F8 | Levy line relabelled with the $250,000 incl. GST threshold and 0.25%, marked for owner confirmation. The existing budget `scenario_inputs` row now says which side to state. The program row's why uses the threshold. |
| F9 | Added `accessible-for-ageing-parent` (Access consultant suggested by `scope_additions[5]` is over-procurement, R6), `priority-sewerage-area` and `unapproved-existing-structure`. That makes 19 triggers, above the brief's 8–15. §9 says why. |
| F10 | Added "Coordinate consultant reports" (`fr.approval-docs-1a`, yes), which lodgement sign-off now follows. `trees-affected` rfps is `partial`, with the option-not-include note. The scale-complex note is on `sewer-main-in-backyard` and `flood-affected`. The trees negation now reads "The large gum in the front yard is well clear of the works". |
| F11 | "Least sure of" keeps only the Schedule 1 width and setbacks, the CDC exclusions and the fire distances. The certifier row in §5 now says the owner appoints, consistent with `builder-designs`, and keeps `verified: false`. |

New counts: 19 triggers (9 variants), 8 packages, 28 cost lines, 73 activities, 7 RFPs.

---

## Round 2 (30 October 2026)

I re-read `shape.yaml` and `DOMAIN.md` after the author applied all 11 findings. The validator, run from
the repo root, gives `ok`, exit 0.

### Verdict: **FIX** (three small edits; nothing is wrong at the root)

### Round 1 fixes checked

- ✔ **F1.** CDC standards are now cited to SEPP (Housing) 2021 Schedule 1, with the 450 m² lot sourced.
  Width and setbacks stay `verified: false`. The Codes SEPP is kept for the principal dwelling.
- ✔ **F2.** The reinforcement hold point is now `yes`, with the app's label. The gap row names the frame
  and slab-pour hold points.
- ✔ **F3.** Structural `inspection_schedule: I`. Civil `for_construction` is an option. Geotechnical
  `g4_planning` and the certifier's A1, A3 and A4 stages are in `app_differs`. All four now agree with my
  `rfp.Resolve` run.
- ✔ **F4.** The trigger is split. `attached-secondary-dwelling` keeps civil. `garage-conversion` removes
  civil under R1 and is in `variants`.
- ✔ **F5.** The `close-to-main-house` evidence now gives distances to the main house. The eaves are on the
  cost line. The 900 mm boundary case goes to a DA variant.
- ✔ **F6.** The certifier's 14-day notice and the holder's 7-day notice are separate rows, and "Establish
  the site" follows the 7-day notice.
- ✔ **F7.** Section 73 wording is updated and `priority-sewerage-area` is added.
  `det.sewer_connection: pressure_sewer` exists, and its trigger `\bsewer` matches "Sewerage".
- ✔ **F8.** The levy line, its program row and the `proj.budget_stated` note all carry the $250,000
  threshold.
- ✔ **F9.** The three new triggers are added.
- ✔ **F10.** "Coordinate consultant reports" is added. Trees `rfps` is now partial. The sewer and flood
  triggers note the complex scale. The trees negation is reworded.
- ✔ **F11.** "Least sure of" is trimmed, and the principal-certifier text is consistent.

### Open items

**R2-1. `da-pathway.removes.program` names a row that no longer exists.** Severity: medium

- The trigger still lists "Notify neighbours of the CDC application". F6 renamed that row, so a scorer
  will not match it.
- **Change:** replace it with "Certifier notifies neighbours within 20 m (14 days before determination)".
- Also add "Give neighbours notice of commencement (at least 7 days before work)" to the same list.
  That 7-day notice is a CDC requirement; under a DA, neighbour notice is the council's notification,
  which is already in `adds`.

**R2-2. `garage-conversion` overstates two items.** Severity: medium

- **`built_in.rfps: "yes"` should be `"partial"`.** The structural `existing_assessment` stage does fire on
  refurb. But the trigger's own why says that under refurb only `baselines[2]` applies, so the app never
  suggests a Structural package. The RFP exists only if the user creates that package.
- **Remove the geotechnical engineer.** The conversion keeps the existing slab and walls, so there are no
  new footings to classify.
  - Add `consultant.geotechnical` to `removes.procurement` and `removes.rfps`.
  - Add "Geotechnical engineer" to `removes.cost_plan`.
  - Add to why: "unless the structural assessment calls for new footings or underpinning".
- **Note the skeleton change in why.** Refurb + traditional + cdc selects `sk.refurb-traditional-cdc`. It
  has **no approval documentation stage** (KNOWLEDGE_INDEX §4.2, §2.3). The base's approval-documentation
  rows (BASIX, stormwater plan, Tap in, sign-off for lodgement) therefore move or vanish in this variant,
  and the scorer should expect that.

**R2-3. Small wording.** Severity: low

- **`close-to-main-house.label`** still says "or a side boundary". The why now sends the boundary case to
  a DA. **Change:** relabel it "Flat within 1.8 m of the main house".
- **`accessible-for-ageing-parent.why`** says "A document naming livable housing adds
  access-egress.livable-housing". No `implied_by` pattern names livable housing. The row comes only from
  a presence reading (`sys.livable-housing.presence = included`, applied at amber or better; KNOWLEDGE_INDEX
  §1.1). **Change:** say "A document stating livable-housing features are included, read as a presence
  row, adds …".
- **`attached-secondary-dwelling.authority`** cites the app's `uc.secondary-dwelling-fire-separation` only
  under `close-to-main-house`. That record's `when` is extend plus `det.existing_building` true, which
  is exactly this variant, so this is the one trigger where it fires (as an Implication). **Change:** add
  to why: "uc.secondary-dwelling-fire-separation fires here (Implication only)".

When R2-1 and R2-2 are applied, the map is usable as an answer key (PASS).

## Round 2 response (author, 30 October 2026)

All three applied; none disputed. Validator `ok` from the repo root.

- **R2-1.** `da-pathway.removes.program` now names "Certifier notifies neighbours within 20 m (14 days before determination)" and "Give neighbours notice of commencement (at least 7 days before work)".
- **R2-2.** `garage-conversion`: `built_in.rfps` is `partial`; geotechnical removed from procurement, cost plan and RFPs ("unless the structural assessment calls for new footings or underpinning"); the why notes `sk.refurb-traditional-cdc` has no approval documentation stage. DOMAIN §9 matches.
- **R2-3.** `close-to-main-house` relabelled "Flat within 1.8 m of the main house"; the livable-housing why says the row comes from a presence reading; `attached-secondary-dwelling` why notes `uc.secondary-dwelling-fire-separation` fires there (Implication only). DOMAIN §9 matches.

Counts unchanged: 19 triggers (9 variants), 8 packages, 28 cost lines, 73 activities, 7 RFPs.
