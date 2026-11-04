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
