# T06 review: sprinkler pump and tank upgrade, with fit-out

Reviewer: independent (brief 02), 30 October 2026. I did not edit `DOMAIN.md` or `shape.yaml`.

## Verdict: **FIX**

The map is deep and mostly right. Its reading of the app is careful, and nearly every built-in claim holds.
It is not usable as an answer key yet, for four reasons:

- the NSW approval and certification model is built on the wrong scope for the Fire Safety Code;
- the pump room ventilation is filed under the wrong system;
- several trigger-added activities use labels the app never prints;
- a handful of common triggers are missing.

Every change below can be made in place. Nothing is wrong at the root.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T06-sprinkler-pump-tank-upgrade/shape.yaml`
(run from the repo root) → `ok`, exit 0.

## 2. Findings

Ordered by severity. "Instrument" means the statute or standard itself. A secondary source (a government
information sheet, an agency web page, a standards store listing) supports a ruling, but it does not set
`verified: true`.

### 1. The Fire Safety Code covers the pump works only, and may not need a fire safety schedule or final certificate (serious)

- **Where:** `DOMAIN.md` §5 (pathway, certificate steps 2, 6 and 7, and "The pathway moves to a DA… when"),
  §8 (certifier RFP); `shape.yaml` header comment, `program` rows "Fire safety schedule issued with the CDC"
  and "Final fire safety certificate…", `triggers[tank-needs-consent]` and `triggers[pump-tank-only]`.
- **What is wrong:**
  - The map treats the whole job as one CDC pathway, and every fire safety step hangs off it.
  - The Fire Safety Code only covers alterations to a hydraulic fire safety system made to manage water
    pressure: fire mains, **fire water storage tanks, on-site fire pumps**, booster connections, and the
    pump house or enclosure.
  - **General modifications to sprinkler systems that are not about water pressure are outside it.** So the
    level 3 sprinkler, detection, EWIS and exit-sign changes belong to the fit-out CDC, not the fire safety
    one.
  - The Regulation was amended to **remove the need for a fire safety schedule and a fire safety
    certificate for an alteration to a hydraulic fire safety system "in certain circumstances"**.
  - The Code also has development standards for water storage tanks and external pump houses, so a tank
    outside the building is not, by that fact alone, a DA trigger.
- **Evidence:**
  - NSW Planning & Environment, Information Sheet 8.1 "Fire Safety Code" (January 2015), sections
    "Managing hydraulic fire safety systems as complying development" and "Streamlining the paperwork".
    It cites the 2000 Regulation; the 2021 equivalent is not confirmed.
  - The Code's part number comes from the as-made amending instrument, EPI 2013 No 706, Sch 1: "Fire Safety
    Code means the code for complying development set out in Part 8". The conditions are in Schedule 10.
- **Exact change:**
  - `DOMAIN.md` §5 pathway: "Complying development under the Codes SEPP. The pump, tank, pump room and fire
    service works go under the Fire Safety Code (Part 8, from EPI 2013 No 706; in-force text not read,
    verified: false). That Code covers alterations to hydraulic fire safety systems for water-pressure
    management. The level 3 fit-out, including its sprinkler, detection and EWIS changes, goes under the
    commercial internal alterations code (see finding 4). Two CDCs is the default; whether one CDC can
    rely on both codes: verified: false."
  - §5 step 2 becomes: "Fire safety schedule with the **fit-out** CDC. The Fire Safety Code CDC may be
    exempt from the schedule and the fire safety certificate (Info Sheet 8.1, citing the 2000 Regulation;
    2021 provision verified: false)."
  - Step 6 gets the same caveat. The FFSC is still needed for the level 3 measures.
  - `shape.yaml` program "Fire safety schedule issued with the CDC": rename it to "Fire safety schedule
    issued with the fit-out CDC". Its `why` gets the exemption caveat.
  - `triggers[pump-tank-only]`:
    - add `authority: ["Fire safety schedule and final fire safety certificate may not be required for a
      Fire Safety Code alteration to a hydraulic fire safety system (verified: false)"]`;
    - replace the non-activity `program: ["One CDC under the Fire Safety Code only"]` with `program: []`.
    - Do not remove the FFSC rows until the owner confirms.
  - `triggers[tank-needs-consent]`:
    - relabel it to "Tank or pump house fails the Fire Safety Code standards, or the land is excluded from
      complying development";
    - drop the evidence line "located in the rear car park" as the sole cause;
    - keep "Council advised … a DA is required" and the car-space evidence.
  - `DOMAIN.md` §5 DA list: replace "the tank sits outside the building" with "the tank or pump house fails
    a Fire Safety Code development standard (for example, it sits over a registered easement)".
  - §8 certifier *must carry*: "CDC assessment under each code".

### 2. Pump room ventilation and the diesel exhaust are `mechanical.local-exhaust`, not `mechanical.ventilation` (serious)

- **Where:** `shape.yaml` `work_items[mechanical.ventilation]`; the program row "Alter exhaust ventilation"
  (`fragment: fr.ventilation-fitout`, `built_in: partial`) and the links that name it; `rfps[consultant.mechanical]`
  `positions.local: absent` and `app_differs[position.local]`; `DOMAIN.md` §2 mechanical and §10 ("tea kitchen …
  `mechanical.local-exhaust` (not in scope)").
- **What is wrong:**
  - `mechanical.ventilation` is outdoor air for occupied rooms. Its `excludes` says "Exhaust from … **plant
    rooms (local-exhaust)**".
  - `mechanical.local-exhaust` lists "plant rooms … make-up air for these systems and their discharge points
    to outside".
  - Because of the mis-mapping, the map credits `fr.ventilation-fitout` (a fit-out row on outdoor air) as
    partial support for the pump room, and it expects the mechanical RFP's `local` row to be absent.
- **Evidence:** `knowledge/clusters/services-wet-air/systems.yaml` `mechanical.ventilation` excludes;
  `mechanical.local-exhaust` describes. No fragment attaches to `mechanical.local-exhaust`
  (`knowledge/program/fragments.yaml`).
- **Exact change:**
  - The work item becomes `{system: mechanical.local-exhaust, action: new, why: "Pump room ventilation:
    diesel combustion and cooling make-up air, plant room exhaust and the engine exhaust flue discharge."}`.
  - Program: relabel the row "Pump room ventilation and diesel exhaust flue", with `fragment: null`,
    `built_in: "no"`, `why: "mechanical.local-exhaust has no fragment"`. Update the cut-over's `after` to
    match.
  - Mechanical RFP: `positions.local: I`, with `app_differs` `{key: position.local, app: ask, expect: I,
    why: "client flag mech_local; plant room exhaust is local exhaust"}`. Reword the `must_include`
    interfaces line to name local exhaust.
  - `DOMAIN.md` §10: the "tea kitchen" trap still adds local exhaust, but local exhaust is now in scope, so
    the trap is harmless for the system list.
  - Cost line "Pump room ventilation" stays 08. Add `mechanical.local-exhaust` to the gaps list of systems
    with no fragment.

### 3. AS 2941 edition: rule it 2013; the knowledge's 2014 is wrong

- **Where:** `DOMAIN.md` §1 and §12; `shape.yaml` `gaps[area: knowledge]`.
- **What is wrong:** The author was unsure whether it is 2013 or 2014. The knowledge cites `AS 2941:2014`.
- **Evidence:**
  - The Standards Australia store lists **AS 2941-2013**, *Fixed fire protection installations — Pumpset
    systems*, as current; it superseded the 2008 edition.
  - `knowledge/clusters/fire/rules.yaml` `rule.as2941.fire-pumps` has `instrument: AS 2941:2014`.
  - AS 2118.1:2017 in `rule.as2118.sprinkler-design` matches the current edition.
- **Exact change:**
  - `gaps[knowledge]`: replace "AS 2941 is cited as 2014 (edition verified: false)" with "rule.as2941.fire-pumps
    cites AS 2941:2014; the current edition is AS 2941-2013 (Standards Australia store). Correct the
    instrument; clauses stay clause_verified: false."
  - `DOMAIN.md` §12: same wording. No clause numbers.

### 4. Codes SEPP part numbers and names

- **Where:** `DOMAIN.md` §5 and §12; `shape.yaml` header, `scenario_inputs[hdr.cond.planning]`, program row
  `fr.cdc-application` `why`, certifier and interior design RFP `must_include`.
- **Ruling:**
  - **Fire Safety Code = Part 8.** Source: as-made instrument EPI 2013 No 706, Sch 1 definition; Schedule 10
    conditions. Keep `verified: false` until the in-force consolidation is read; legislation.nsw.gov.au
    refused my fetch.
  - **The fit-out code is uncertain.** The 2015 information sheets call it the "Commercial and Industrial
    Alterations Code" (Part 5). The Department's 2022 housekeeping amendments table lists "applicable
    development standards for internal alterations and change of use" under "**Part 5A Industrial and
    Business Buildings Code**". That suggests the alterations code has since been renamed or folded into
    Part 5A.
- **Exact change:** Wherever the map says "Commercial and Industrial Alterations Code", write "the Codes SEPP
  code for internal alterations to commercial buildings (historically Part 5 Commercial and Industrial
  Alterations Code; now possibly Part 5A Industrial and Business Buildings Code; verified: false)".
- **One CDC or two:** model two (finding 1). Add a program row
  `{label: "Prepare and lodge the fit-out CDC", stage: approvals, fragment: null, after: ["Client sign-off for tender"], built_in: "no", why: "Second code; the app has one CDC fragment."}`.
  Add it to `pump-tank-only` `removes.program`.

### 5. Missing NSW fire safety step: CFSP endorsement for a relevant fire safety system

- **Where:** `DOMAIN.md` §5 certificate list; `shape.yaml` program (approvals).
- **What is missing:**
  - Since the 2017 reforms, plans and specifications for installing, extending or modifying a "relevant
    fire safety system" (sprinklers among them) must be prepared or endorsed by a competent fire safety
    practitioner. The endorsement goes with the CC or CDC application.
  - Info Sheet 8.1 also says the Regulation "outlines documentation that must accompany an application
    for an alteration to a hydraulic fire safety system".
  - The map has the CFSP only at the end of the job.
- **Exact change:**
  - Add a §5 step between 1 and 2: "Plans and specifications of the altered sprinkler system and its water
    supply endorsed by a competent fire safety practitioner, lodged with the CDC application (EP&A (DCFS)
    Regulation 2021, verified: false)".
  - Add a program row `{label: "Competent fire safety practitioner endorses the fire services plans and specifications", stage: construction_documentation, fragment: null, after: ["Fire services design, hydraulic calculations and specification"], built_in: "no"}`.
  - Add that row to the `after` list of "Prepare and lodge the complying development certificate".

### 6. Who receives the final fire safety certificate

- **Where:** `DOMAIN.md` §5 step 6; `shape.yaml` program label "Final fire safety certificate to the certifier,
  council and FRNSW, displayed in the building".
- **What is wrong:** The owner does not send it to the council. The owner gives FRNSW a copy, with the current
  fire safety schedule, and displays both in the building. The council receives it from the principal
  certifier with the OC.
- **Evidence:** FRNSW, "Lodge a fire safety certificate" (fire.nsw.gov.au/page.php?id=9158), citing s 85 of the
  EP&A (DCFS) Regulation 2021. This is a secondary source, so the section stays `verified: false`.
- **Exact change:** Relabel the row "Owner issues the final fire safety certificate; copy to FRNSW with the fire
  safety schedule; displayed in the building", and update the three `after` lists that name it. In §5:
  "(s 85 per FRNSW; verified: false). The council receives it with the OC."

  The rest of the sequence is right: fire safety schedule at CDC, CFSP assessment of each new and altered
  measure, FFSC before the OC, AFSS continuing and covering the new measures, and only a final inspection
  for Class 5 (verified: false).

### 7. Trigger activities the scorer cannot match

- **Where:** `triggers[tank-needs-consent].adds.program`, `triggers[electrical-supply-upgrade].adds.program`,
  `triggers[fit-out-amenities].adds.program`.
- **What is wrong:**
  - The scorer adds trigger activities with `fragment: None`, so only an exact or 0.75-Jaccard label can
    match (`testing/synthetic/tools/score_shapes.py` `_program_rules`, `SIMILAR_LABEL`).
  - The app prints different labels:
    - `fr.da-lodgement` "Lodge the application", `fr.da-exhibition` "Public exhibition" (missing from the
      trigger), `fr.da-assessment` "Assessment", `fr.construction-certificate` "Construction certificate";
    - `fr.switchboard-major` "Procure and deliver" and "Install".
  - One-activity work-item fragments print the work item title (the system label) instead of the fragment
    label (`internal/program/generate.go` `instantiate`). So `fr.wet-area-fitout` prints "Wet-area
    waterproofing", and `fr.tiling-fitout` prints "Wall and floor tiling".
  - As written, `tank-needs-consent` and `fit-out-amenities` claim `program: yes` but would score as missed.
- **Exact change:**
  - `tank-needs-consent` program: `["Lodge the application", "Public exhibition", "Assessment", "Construction certificate"]`.
  - `fit-out-amenities` program: `["Wet-area waterproofing", "Wall and floor tiling"]`.
  - `electrical-supply-upgrade`: keep the descriptive labels but set `built_in.program: "no"`, with a note
    that `fr.switchboard-major`'s generic labels clash with joinery's. Or use the generic labels and accept
    the ambiguity.

### 8. Missing common triggers

- **Where:** `shape.yaml` `triggers`; `DOMAIN.md` §9.
- **What is missing:** four triggers a Sydney PM expects on this job.

**a. `affected-part-access`**

- The owner, not a lessee, applies for the level 3 fit-out. So the Premises Standards "affected part" path
  of travel, from the principal entrance to the new work, may need upgrading. Disability (Access to
  Premises — Buildings) Standards 2010 (verified: false).
- Evidence: "The certifier advises the path of travel from the principal pedestrian entrance to level 3 is
  an affected part and must be upgraded."
- Negation: "The fit-out is lodged by the tenant (lessee concession)." or "No affected-part upgrade is
  required."
- Adds: `consultant.access`; `access-egress.accessible-access` upgrade; cost line "13.1 Accessible paths,
  TGSIs and braille/tactile signage".
- built_in: procurement yes (already suggested), cost_plan yes.

**b. `existing-fire-engineered-building`**

- The building's existing performance solutions (fire engineering report) rely on the sprinkler system, so
  any change needs fire engineering review.
- Evidence: "The building operates under a fire engineering report dated 2006 that relies on the sprinkler
  system for extended travel distances."
- Adds: `consultant.fire_engineer`.
- built_in: procurement no.

**c. `da-fire-upgrade`** (or fold it into `tank-needs-consent`)

- On a DA for alterations, the consent authority may require the existing building's fire safety to be
  upgraded. The old cl 94 is now in the 2021 Regulation, number verified: false.
- Adds: `consultant.bca` (fire safety upgrade report); authority "Consent authority may require fire safety
  upgrade of the existing building (verified: false)".

**d. `interim-occupation`**

- The tenant wants level 3 before the pumps are commissioned, which needs an interim fire safety
  certificate and an interim OC (verified: false), or a link change if the level 3 sprinklers cannot run
  on the old supply.
- built_in: program no.

### 9. BCA consultant: ruled not in the base, but the DA path adds one

- **Where:** `not_expected_procurement[consultant.bca]`; `DOMAIN.md` §4 and §12.
- **Ruling:** The map is right; leave it out of the base.
  - On a deemed-to-satisfy CDC, the certifier's assessment plus the engineers' design certificates cover
    compliance.
  - A registered certifier who is principal certifier may not also author the design-side compliance
    advice (certifier conflict-of-interest rules, verified: false). So a BCA consultant appears exactly when
    design-side compliance work exists: a performance solution (already a trigger) or a DA fire upgrade
    (finding 8c).
- **Exact change:**
  - Close the §12 judgement call with "Reviewer ruling: not in the base. Owner may overrule."
  - Add `consultant.bca` to `triggers[tank-needs-consent].adds.procurement` (with `built_in.procurement`
    still `yes`, since the app suggests it anyway).

### 10. Sydney Water product names

- **Where:** `DOMAIN.md` §5 steps 1 and 5, §12; `shape.yaml` `cost_plan` Sydney Water line; program rows; the
  fire services RFP line `position.authority`.
- **Ruling:**
  - "**Statement of available pressure and flow**", requested through Sydney Water **Tap in**, is the right
    product name. FRNSW requires "a statement of available pressure and flow" for any design or alteration
    of a hydraulic fire safety system (FRNSW position statement, hydraulic fire safety system design), and
    City of Sydney referral advice names Tap in. Keep it `verified: false` because I did not read a
    Sydney Water page.
  - Do **not** name a "fire service application" or a "capacity" product. Write "Sydney Water approval of
    the fire service and backflow alterations (product name verified: false)".
  - A Section 73 compliance certificate is not expected for this work (verified: false).
- **Exact change:**
  - Cost line: "Sydney Water: statement of available pressure and flow, fire service alteration approval
    and backflow registration".
  - RFP `position.authority` text: "Sydney Water statement of available pressure and flow (Tap in), and
    approval of the fire service and backflow alterations".

### 11. Long service levy: the repo holds no rate

- **Where:** `shape.yaml` `cost_plan["Long service levy"].why`; `DOMAIN.md` §5 step 3.
- **What is wrong:** "$250,000 incl. GST, 0.25%" is a rate the repo does not hold. Its cited source, T12, is
  another shape, not knowledge.
- **Evidence:** The Long Service Corporation page says 0.25% of works of $250,000 or more including GST, from
  1 January 2023, paid before work starts (longservice.nsw.gov.au/bci/levy/about-the-levy). This is
  secondary, so it stays verified: false.
- **Exact change:** Keep the figures, but cite "Long Service Corporation, About the levy (threshold and rate
  verified: false; not held in the repo)", the same way owner ruling R4 cites SIRA.

### 12. Flow-test Implication: the gap is overstated

- **Where:** `DOMAIN.md` §11 item 14; `shape.yaml` `gaps[area: implications]`.
- **What is wrong:** `uc.existing-fire-water-fails-flow-test` fires on `new`, `upgrade` or `alter` works to
  sprinklers or hydrants in an existing building. In the base and in `pump-tank-only`, `fire-active.sprinklers`
  is `alter`, so the Implication **does** fire.
- **Evidence:** `knowledge/clusters/fire/unforeseen.yaml` `uc.existing-fire-water-fails-flow-test` `when`.
- **Exact change:** "The Implication attaches to fire-water but is triggered by sprinkler or hydrant works.
  It fires here only because the sprinkler connection is `alter`. A pump job that leaves sprinklers and
  hydrants untouched gets none."

### 13. Budget split leaves structure and site at zero

- **Where:** `shape.yaml` `gaps[area: cost_plan]` (the commercial_fitout entry); `DOMAIN.md` §6.
- **What is wrong:** The map notes fire-active at 4% and interiors at 50%. It misses that `commercial_fitout`
  has **no `structure` or `site` share**. So "03 Structure" (plinths, cores) and "01 Site works" (pump
  removal, soft strip) get nothing from a stated budget.
- **Evidence:** `knowledge/costs/budget_splits.yaml` `commercial_fitout.shares`.
- **Exact change:** Append to that gap: "and has no structure or site share, so the 03 and 01 lines receive
  nothing from a stated budget."

### 14. Diesel noise is filed under the wrong system

- **Where:** `triggers[diesel-noise-exhaust].adds.work_items`.
- **What is wrong:** `interiors.room-acoustics` is reverberation and noise levels inside a room. Pump noise
  reaching tenancies is sound insulation between uses (`interiors.acoustic-separation`); plant noise design
  sits with mechanical.
- **Evidence:** `knowledge/clusters/envelope/systems.yaml` `interiors.room-acoustics`,
  `interiors.acoustic-separation`.
- **Exact change:** `{system: interiors.acoustic-separation, action: new}`. The cost line stays code 05.

### 15. `pump-tank-only` leaves hold points that cannot fire

- **Where:** `triggers[pump-tank-only].removes.program`.
- **What is wrong:**
  - With no interiors items, neither hold point has a `before` row, so `holdPointRows` adds neither.
  - "Practical completion inspection" keeps an `after` link to the removed "Lay floor finishes".
- **Exact change:** Add "Hold point: a recorded pressure test of the sprinkler pipework" and "Hold point: a
  location-based inspection record for concealed fire stopping" to `removes.program`. Note in `why` that the
  PC link to floor finishes falls away.

### 16. Backflow cost line: wrong package claim

- **Where:** `cost_plan["Fire service backflow prevention device"].why`.
- **What is wrong:** "costed under trade.hydraulic_plumbing" holds only when a live works package for that
  trade exists. Otherwise the line goes under the head contract or "Head contractor" (`mappedCostPackage`,
  KNOWLEDGE_INDEX §3.2), and the shape never creates that package.
- **Exact change:** "'09 Hydraulic services'; mapped to trade.hydraulic_plumbing only if that works package
  exists, else the head contract. In practice the fire contractor carries it."

### 17. Missing activity: pumpset submittal review

- **Where:** `shape.yaml` program (procurement).
- **What is missing:** The engineer reviews the pumpset, controller and tank submittals before the order is
  placed. The pump curves must meet the duty points. This is the commonest cause of a failed acceptance
  test (`fm.fire-pump-fails-flow-test`).
- **Exact change:** Add `{label: "Fire services engineer reviews pumpset, controller and tank submittals", stage: procurement, fragment: null, after: ["Award the contract"], built_in: "no"}`.
  Make "Order long-lead pumpsets, controllers and tank" follow it.

### 18. Minor notes on the record

- **Two NCC classes:** `det.ncc_class: ["5","7a"]` also adds the industrial category to the package facts
  (KNOWLEDGE_INDEX §2.1). It changes nothing, because `baselines[3]` already covers commercial, and the
  fragment `any_of` matches either part (`internal/knowledge/relevance.go`). Add one line to `DOMAIN.md` §4.
- **Live-site RFP rows:** `partial_occupation` sets the RFP `live_operations` flag to yes
  (`internal/rfp/drivers.go` `readFlags`), so the live-site module rows appear. That is correct; say so in
  §8.
- **One more keyword trap for §10:** "Sydney Water asset" in a Tap in or building-plan letter matches
  `flag.authority_asset` (trigger `sydney water asset`).

## 3. Built-in claims checked

✔ = the claim holds against the cited file and the KNOWLEDGE_INDEX selection logic. ✘ = it does not, or does
not as worded.

**Procurement**

- ✔ consultant.fire_services yes, baselines[3] (also scope_additions[1] via fire-water)
- ✔ consultant.electrical yes, baselines[2]
- ✔ consultant.mechanical yes, baselines[3]
- ✔ consultant.certifier yes, baselines[2]
- ✔ consultant.interior_design yes, baselines[4]
- ✔ trade.main_works partial, ProposeHeadContract (only from accepted items)
- ✔ not_expected Architect, Hydraulic (baselines[2]) and BCA, Access (baselines[3]): app_suggests yes
- Spot-checked `no`, all ✔: consultant.structural (no refurb baseline names it), consultant.hazmat (no rule),
  trade.fire_services (trades never suggested), consultant.acoustic (no rule, no template), fire engineer (no
  rule)

**Cost plan**

- ✔ CDC fees partial: `cc-fees` has `applies_when: true`, so it is kept on cdc
- ✔ Sydney Water partial: `service-authority`
- ✔ Fee lines yes for Fire Services, Electrical, Mechanical, Certifier and Interior Design; partial for
  Structural and Hazmat (via Suggest)
- ✔ Construction lines by element code: 01, 03, 05, 06, 07, 08, 09, 10, 11 (`elements.yaml`)
- ✘ Backflow "costed under trade.hydraulic_plumbing" (finding 16)
- ✔ Design contingency, construction contingency, escalation and exclusions yes
- ✔ Budget split `commercial_fitout`, with fire-active 4% and interiors 50%. Incomplete: structure and site
  get 0 (finding 13)
- Spot-checked `no`, all ✔: long service levy, temporary fire protection (no system or line), testing and
  commissioning, preliminaries, provisional sum

**Program**

- ✔ Skeleton `sk.refurb-traditional-cdc`
- ✔ `fr.existing-conditions` (3 rows), `fr.concept-design-fitout` (3), `fr.design-development-fitout`,
  `fr.fitout-documentation` (3), `fr.cdc-application` (2); labels exact
- ✔ `fr.tender-docs`, `-period`, `-evaluation-fitout`, `fr.contract-award`, `fr.site-establishment-fitout`
- ✔ `fr.strip-out-fitout` fires on site.demolition `remove`; `fr.structure-alterations` attaches after strip-out
- ✘ `fr.ventilation-fitout` partial: under the correct system (local-exhaust) it does not fire (finding 2)
- ✔ `fr.fire-stopping` has no attach, so it floats
- ✔ Hold-point labels "Hold point: a location-based inspection record for concealed fire stopping" and "Hold
  point: a recorded pressure test of the sprinkler pipework" are exact (`holdPointRows`)
- ✔ `fr.sprinklers-fitout` and `fr.detection-fitout` attach after `fr.ceilings-fitout`
- ✔ Fit-out fragments, by predicate and action: partitions, mechanical (on air-conditioning), electrical,
  sanitary, cold, hot, comms, ceilings, lighting, doors, joinery, painting, floors. Note that one-activity
  rows print the system label (`instantiate`), which the fragment rule still matches.
- ✔ `fr.practical-completion`, `fr.occupation-certificate` (no FFSC link), `fr.handover` partial, defects rows
- ✔ `fr.switchboard-major` ignores `alter`; `fr.services-connections-major` fires for class 5 upgrade;
  `fr.utility-major` only for 2, 7b and 9b; hydrant fragments only for 2 and 7b, new or replace
- ✔ No fragment for `fire-active.fire-water`, `occupant-warning`, `emergency-lighting`, `backflow-prevention`
  or `rated-elements`
- ✔ `rule.project.fire-system-impairment` gated on fire-control or exits; `rule.project.integrated-fire-testing`
  on fire-control
- ✘ `uc.existing-fire-water-fails-flow-test` "misses this job": it fires in the base (finding 12)

**RFPs**

- ✔ fire-services: boundary inception..design; investigation `ask`; staging always O; `fs_*` flags; hazards and
  strategy X
- ✔ structural:
  - existing_assessment on refurb;
  - concept_schematic always;
  - Class 5 makes it complex;
  - construction_phase option, because the boundary is applied before include_when (`resolve.go`);
  - geotech_brief I, services_fixings O, services_footings I, specialist_element O;
  - not opened: the CC wording in testdata
- ✔ electrical-services: planning, generation and ict `if_unknown: option`; boundary at design
- ✘ mechanical-services `local: absent` (finding 2); the other rows ✔
- ✔ bca-certification: `unsupported_pathways` includes cdc
- ✔ Interior Design and Hazmat have no template, so they get the generic `tpl.rfp-capex`

**Triggers**

- ✔ tank-needs-consent procurement, cost and RFP yes; ✘ its program labels (finding 7)
- ✔ performance-solution: procurement no, RFP partial
- ✔ town-main-inadequate partial
- ✔ hydrant-system-upgrade partial or no
- ✔ asbestos-found partial
- ✔ electrical-supply-upgrade partial, but its labels cannot be matched (finding 7)
- ✔ fit-out-amenities procurement and cost; ✘ its program labels (finding 7)
- ✔ heritage-item (complexity_additions[1], fr.heritage-referral)
- ✔ roof-or-crane-tank RFP partial
- ✘ diesel-noise-exhaust system choice (finding 14)

**Keyword traps (§10):** ✔ every regex quoted matches `flag_readings.yaml` or `determinants.yaml`: basement,
authority_asset, fire_special_hazards, fire_specialist_systems, staged, architect_lodges.

## 4. Owner-ruling conformance

- **R3:** proj.regulated_design is "no" for Class 5. ✔
- **R4:** HBCF not applicable. ✔
- **R5:** BASIX not applicable. ✔
- **R7:** applied only to Class 1/10; Class 5 inspections are kept verified: false. ✔
- **R8:** defects liability period length is verified: false. ✔
- **R1, R2, R6:** not engaged.
- **CONVENTIONS C2 and C3:** respected.

No ruling is broken.

## Round 1 response (author, 4 November 2026)

All 18 findings applied; none disputed. The `shape.yaml` side of most findings was already in the
committed tree from an earlier pass. I checked each one against the finding, and added the two items
still missing (below). `DOMAIN.md` had not been updated; it now carries every finding.

Go check (read-only): I ran a throwaway program, since deleted, that calls `program.Generate` on the
base, `pump-tank-only`, `fit-out-amenities`, `electrical-supply-upgrade` and `tank-needs-consent`
profiles. It confirmed findings 2, 7 and 15: no ventilation row fires; the DA rows print "Lodge the
application", "Public exhibition", "Assessment" and "Construction certificate"; wet-area and tiling print
"Wet-area waterproofing" and "Wall and floor tiling"; `fr.switchboard-major` prints "Procure and deliver"
and "Install"; and `pump-tank-only` gets neither hold point.

1. **Fire Safety Code scope and exemption.** Applied. Shape: header, two-CDC `scenario_inputs`, the row
   "Fire safety schedule issued with the fit-out CDC" with the exemption caveat, the `pump-tank-only`
   authority line and `program: []`, the `tank-needs-consent` relabel and evidence. FFSC rows kept until
   the owner confirms. DOMAIN §5: pathway, DA list and steps rewritten; §8 certifier "each code".
2. **Local exhaust.** Applied. Work item `mechanical.local-exhaust` new; row "Pump room ventilation and
   diesel exhaust flue" (`fragment: null`, built_in no), with the cut-over `after` updated; mechanical RFP
   `local: I` with `app_differs`; gap list names local-exhaust. DOMAIN §2, §6, §7, §8, §10 and §11.
3. **AS 2941-2013.** Applied in the knowledge gap and DOMAIN §11 and §12.
4. **Codes SEPP parts.** Applied. Fit-out code wording everywhere; row "Prepare and lodge the fit-out
   CDC", also in `pump-tank-only` `removes.program`.
5. **CFSP endorsement.** Applied. §5 step 2; the endorsement row, and the CDC lodgement `after` names it.
6. **FFSC recipients.** Applied. Row relabelled. DOMAIN §5 step 7 and the FRNSW list updated. New this
   round: the program gap no longer says "copy to … council".
7. **Trigger labels.** Applied as specified. `electrical-supply-upgrade` keeps descriptive labels with
   `built_in.program: "no"`.
8. **Four triggers.** Applied: `affected-part-access`, `existing-fire-engineered-building`,
   `da-fire-upgrade` and `interim-occupation`, in the shape and in the DOMAIN §9 table.
9. **BCA.** Applied. Kept out of the base; added to `tank-needs-consent`; §12 closed with the reviewer's
   ruling.
10. **Sydney Water.** Applied: cost line, RFP `position.authority`, program row, and §5, §6, §8 and §12.
11. **Long service levy.** Applied: cited to the Long Service Corporation, verified: false, not held in
    the repo.
12. **Flow-test Implication.** Applied in the gap and DOMAIN §11 item 14.
13. **Budget split.** Applied: no structure or site share, in the gap and DOMAIN §6.
14. **Acoustic separation.** Applied.
15. **Hold points in `pump-tank-only`.** Applied. New this round: the `why` also notes that the
    electrical, sprinkler and detection fit-out fragments still fire on the base-building items.
16. **Backflow package.** Applied in the cost line and DOMAIN §6.
17. **Submittal review.** Applied. The order row follows it. DOMAIN §7.
18. **Minor notes.** Applied: two classes (§4), live-site rows (§8), the "Sydney Water asset" trap (§10).

**New counts:** 18 triggers, 9 packages, 36 cost lines, 85 activities, 7 RFPs, 26 gaps.

Validator: `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T06-sprinkler-pump-tank-upgrade/shape.yaml`
→ `ok`.

## Round 2 review (4 November 2026)

I re-read `shape.yaml`, `DOMAIN.md` §1–12 and the Round 1 response, and checked them against `OWNER_RULINGS.md` and
`CONVENTIONS.md`.

- **Validator**, run from the repo root: `ok`, exit 0.
- **Script check:** 85 activities, 85 unique labels, and no dangling `after` reference in the base. It also
  listed the kept rows whose `after` names a row the trigger removes. R2-2 and R2-5 use that list.
- **Counts** match the response: 18 triggers, 9 packages, 36 cost lines and 7 RFPs.

### Verdict: **FIX**

Every round 1 finding is applied, and none was disputed. Three problems remain:

- two of the fixes left the program logic half-joined;
- one trigger I wrote in round 1 cannot be tested as worded;
- the levy and WHS wording do not yet follow CONVENTIONS C4.

There are five findings in all. Each is a small edit in place.

### Round 1 fixes checked

- **F1 ✔.**
  - The pump works go under the Fire Safety Code and the fit-out under the alterations code.
  - The exemption caveat is on the schedule row, the FFSC row and `pump-tank-only`.
  - `tank-needs-consent` has its new label and evidence, and the DA list is rewritten.
  - The certifier RFP says "each code", and the FFSC rows are kept for the owner.
- **F2 ✔.**
  - The work item is `mechanical.local-exhaust` new. Its row is `fragment: null`, built_in no, and the
    cut-over follows it.
  - The mechanical RFP has `local: I` with an `app_differs` row.
  - The gap list and DOMAIN §2, §10 and §11 agree.
- **F3 ✔.** AS 2941-2013 is in the gap, §11 item 18 and §12. The §12 bullet now says it twice (remaining
  item 2).
- **F4 ✔.** The fit-out code wording is used everywhere. The row "Prepare and lodge the fit-out CDC" is
  added and is in `pump-tank-only` `removes`. Its links are incomplete (R2-1).
- **F5 ✔.** The CFSP endorsement row is in construction documentation, and both CDC lodgements follow it.
- **F6 ✔.** The FFSC row is relabelled, the program gap is corrected, and §5 step 7 is right.
- **F7 ✔.** The DA, wet-area and tiling labels match the app; the author's Go run confirms them.
  `electrical-supply-upgrade` program is `no`.
- **F8 ✔, with one defect.** All four triggers are in the shape and in §9. The `affected-part-access`
  evidence and negation clash with the base (R2-3).
- **F9 ✔.** BCA stays out of the base and is added to `tank-needs-consent`. §12 is closed.
- **F10 ✔.** The Sydney Water wording is in the cost line, the RFP, the program row and §5. No product name
  is invented.
- **F11 ✔, overtaken by C4** (R2-4).
- **F12, F13, F14, F16, F17 and F18 ✔.**
- **F15 ✔.** Both hold points are removed, and the floor-finishes link falls away. The `why` needs one more
  note (R2-5).
- **Owner rulings:** R3, R4, R5, R7 and R8 are still followed, and no ruling is broken. C2 and C3 are
  followed too.

### Findings

**R2-1. The fit-out CDC holds nothing back (program logic).**

- **Where:** `shape.yaml` program rows "Prepare and lodge the fit-out CDC" and "Fire safety schedule issued
  with the fit-out CDC"; `DOMAIN.md` §7 "Logic links that matter".
- **What is wrong:**
  - No row follows "Fire safety schedule issued with the fit-out CDC". That row is the only stand-in for
    the fit-out CDC determination.
  - So the fit-out CDC never holds back the level 3 award or start.
  - "Award the contract" follows only "Certifier determination", which is the Fire Safety Code CDC.
  - As keyed, the level 3 strip-out can start with no fit-out approval.
- **Exact change:**
  - Add "Fire safety schedule issued with the fit-out CDC" to the `after` list of "Appoint the principal
    certifier and give notice of commencement". The `after` list of "Award the contract" also works.
  - In that row's `why`, add: "Also stands for the fit-out CDC determination; the app has one CDC
    fragment."
  - DOMAIN §7: change "The CDC and the contract award come before the start" to "Both CDC determinations
    and the contract award come before the start."

**R2-2. `tank-needs-consent` keeps CDC rows that are wrong under a DA.**

- **Where:** `triggers[tank-needs-consent]` `adds.program`, `removes.program` and `why`.
- **What is wrong:** The trigger removes both CDC lodgements and "Certifier determination". Three kept rows
  now link to rows that are gone:
  - "Fire safety schedule issued with the fit-out CDC". Under a DA the schedule is issued with the
    construction certificate, and there is no fit-out CDC.
  - "Pay the long service levy". It follows the removed CDC lodgement. Under a DA it is paid before the
    CC issues.
  - "Award the contract". It follows the removed determination.

  The trigger also removes the fit-out CDC without saying why. That assumes the owner folds the level 3
  fit-out into the one DA, but a separate fit-out CDC is also possible.
- **Exact change:**
  - Add "Fire safety schedule issued with the fit-out CDC" to `removes.program`.
  - Add "Fire safety schedule issued with the construction certificate" to `adds.program`. Set
    `built_in.program: "partial"`, because the app has no schedule row.
  - Append to `why`: "Assumes the fit-out is folded into the DA (a separate fit-out CDC is possible). The
    levy and the award re-link to 'Construction certificate'."

**R2-3. The `affected-part-access` evidence is true in the base, so the trigger cannot be tested.**

- **Where:**
  - `triggers[affected-part-access]` `evidence[1]` and `negation[0]`;
  - `not_expected_procurement[consultant.access].why`;
  - `DOMAIN.md` §2 access-egress and the §9 row.

  This is my own wording from round 1 (8a).
- **What is wrong:**
  - In the base, the owner fits out level 3 for a new tenant. So "The owner, not a lessee, is the applicant
    for the level 3 fit-out CDC" is true in the base. Base documents would fire the trigger.
  - The negation "lodged by the tenant (lessee concession)" contradicts the base. It belongs to
    `tenant-procured-fitout`.
  - Because the owner is the applicant, the base needs a stated reason why no affected-part upgrade
    applies. Neither the shape nor DOMAIN §2 gives one.
- **Exact change:**
  - Delete `evidence[1]`.
  - Replace `negation[0]` with: "The certifier confirmed the existing path of travel from the principal
    pedestrian entrance to level 3 already complies; no affected-part upgrade is required."
  - Add a `scenario_inputs` entry with that phrasing. Its note: "The owner is the applicant, so the base
    must state that the affected part already complies."
  - DOMAIN §2 access-egress: add "the affected part from the principal entrance already complies (stated
    in the base documents; Premises Standards 2010, verified: false)".
  - `not_expected_procurement[consultant.access].why`: change "Access joins under fit-out-amenities" to
    "Access joins under fit-out-amenities or affected-part-access".
  - Update the §9 row to match.

**R2-4. CONVENTIONS C4 is not applied to the levy and the WHS Regulation.**

- **Where:**
  - Levy: `cost_plan["Long service levy"].why`; DOMAIN §5 step 4, §6, and the §12 list of unverified
    claims.
  - WHS: `procurement[consultant.hazmat].why`; the hazmat RFP `must_include[2]`; DOMAIN §3 and §12.
- **What is wrong:**
  - C4 lets maps cite the levy as checked: 0.25% of building work of $250,000 or more, including GST. The
    map still says "threshold and rate verified: false; not held in the repo". §5 step 4 cites C4 and
    contradicts it in the same sentence.
  - The WHS Regulation is cited with no year. C4 records that the NSW WHS Regulation 2025 has replaced the
    2017 Regulation.
- **Exact change:**
  - Levy: write "0.25% of the cost of building work of $250,000 or more, including GST (CONVENTIONS C4,
    checked)". Take the levy off the §12 list of unverified claims.
  - Keep "before the CDC issues" as `verified: false`, because C4 records only "paid before work starts".
  - WHS: write "WHS Regulation 2025 (NSW; clause numbers verified: false)" in all four places.

**R2-5. `pump-tank-only` removes strip-out rows that the app still prints, and does not say so.**

- **Where:** `triggers[pump-tank-only]` `removes.program` and `why`.
- **What is wrong:**
  - The variant keeps `site.demolition` remove, because the old pump is still removed.
  - So `fr.strip-out-fitout` still fires: it needs `site.demolition` remove, Class 5 and refurb
    (`knowledge/program/fragments.yaml`).
  - "Alter and protect retained structure" still attaches after it.
  - Removing the three strip-out rows is right for the real job, which has its own row "Remove the old
    pump and redundant pipework".
  - But the `why` lists the electrical, sprinkler and detection rows that still fire, and not these
    three. Someone reading a score would take them for an error in the key.
- **Exact change:** Append to `why`: "fr.strip-out-fitout still fires on the pump-removal demolition item,
  so the app prints the three strip-out rows; they are expected noise, like the fit-out concept rows."

### Remaining items (optional, not blocking)

1. Six fit-out finishing rows have no successor:
   - "Electrical fit-off";
   - "Sanitary fit-off";
   - "Commission the altered system";
   - "Install luminaires";
   - "Hang doors";
   - "Install structured cabling".

   Practical completion follows only "Lay floor finishes". This predates round 1. If the scorer checks
   float, add these rows to the `after` list of the practical completion row.
2. DOMAIN §12, first bullet of the unverified list: the AS 2941 sentence repeats itself ("ruled 2013; …
   cites 2014; the current edition is AS 2941-2013 …"). Keep one sentence.

## Round 2 response (author, 4 November 2026)

All five findings and both optional items are applied; none is disputed.

- **R2-1 applied.** "Appoint the principal certifier and give notice of commencement" now follows "Fire
  safety schedule issued with the fit-out CDC". That row's `why` says it also stands for the fit-out CDC
  determination. DOMAIN §7 now says "Both CDC determinations and the contract award come before the
  start".
- **R2-2 applied.**
  - "Fire safety schedule issued with the fit-out CDC" is added to `removes.program`.
  - "Fire safety schedule issued with the construction certificate" is added to `adds.program`, with
    `built_in.program: "partial"`.
  - The `why` names the folded-fit-out assumption and the levy and award re-link.
- **R2-3 applied.**
  - `evidence[1]` is deleted. A second evidence line that is false in the base replaces it, because the
    validator needs two: an access report finds the entrance doors and lift lobby non-compliant.
  - The negation is replaced with the reviewer's wording.
  - A `scenario_inputs` entry carries that phrasing. No app row key covers access, so the entry is keyed
    to `hdr.cond.planning`, because the statement comes with the certifier's CDC advice. The validator
    rejected `affected-part-access` as a key.
  - The access `why`, DOMAIN §2 and the §9 row are updated.
- **R2-4 applied.**
  - Levy: "0.25% of the cost of building work of $250,000 or more, including GST (CONVENTIONS C4,
    checked)" is in the cost line and in §5 and §6. "Before the CDC issues" stays verified: false. The
    levy rate is off the §12 unverified list.
  - WHS: "WHS Regulation 2025 (NSW; clause numbers verified: false)" is in the hazmat `why`, the hazmat
    RFP, and DOMAIN §3 and §12.
- **R2-5 applied.** The `pump-tank-only` `why` now says the three strip-out rows are expected noise.
- **Optional 1 applied.** The six fit-out finishing rows are added to the practical completion `after`
  list.
- **Optional 2 applied.** The §12 AS 2941 sentence is now one sentence.

**Counts:** 18 triggers, 9 packages, 36 cost lines, 85 activities, 7 RFPs, 26 gaps, 10 scenario inputs.

**Validator:** `ok`.
