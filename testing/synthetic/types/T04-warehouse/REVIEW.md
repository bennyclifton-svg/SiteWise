# T04 Industrial warehouse: review, round 1

Reviewer: independent (senior NSW PM who reads YAML and Go). Date: 30 October 2026.
Files reviewed: `DOMAIN.md` and `shape.yaml` (validator `ok`). Neither was edited.

**Verdict: FIX.** The map is right at the root: the base, its size, the 20 packages, the 19 RFPs
and the 15 triggers suit a D&C warehouse. Both of the author's unrun claims are confirmed by a Go
run. What needs fixing:

- the staged-CC rows contradict the T02 and T13 rulings;
- the Class 5–9 inspection list is the 2000 Regulation's wording, not the 2021 one;
- the C3D4 trigger mixes up which conditions come from C3D4 and which from E1D15 and E2D10;
- about 45 RFP keys where the app answers `ask` or `option` have no `app_differs` entry.

## How I checked

1. **Validator.** `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T04-warehouse/shape.yaml`
   → `ok  testing/synthetic/types/T04-warehouse/shape.yaml`, exit 0.
2. **Read-only Go run.** I wrote a `package main` in my scratchpad and ran it with
   `go run -overlay overlay.json ./internal/zzt04review` from the repo root. The overlay maps a
   virtual `internal/zzt04review/main.go` to the scratch file, so nothing was written to the repo.
   There was no database and no Jev call. The code path:
   - `knowledge.Load` → `profile.Build` (the shape's `profile` block as user values, with each
     `flag.X` as `proj.flag.X`, plus a stated budget) → `ProposedWorks` → `ProjectWorkScope`;
   - `PackageBaselineFacts` → `procurement.SuggestedPackages` → `costs.Suggest`;
   - `program.Generate`, run twice: once on the default scope, once on the shape's own `work_items`;
   - `rfp.Drivers` on a `reports.Snapshot` built as `store/report_snapshot.go` builds it, then
     `rfp.Resolve` for each of the 17 templated RFPs. The package novation was `option`, the app's
     D&C default.

   For each RFP I compared the app's output with the shape's prediction: `expect` overridden by
   `app_differs`. I then ran it again with `det.ncc_class: 7b` alone, to isolate the second class.
3. **NCC text.** I read NCC 2022 Vol One online on the ABCB site (Parts C3, E1 and E2).
   legislation.nsw.gov.au returned 403, so for the Regulation I used council pages.

### The four claims the coordinator asked me to rule on

| Claim | Ruling | Evidence |
|---|---|---|
| No Class 7b hold point fires | **Confirmed (Go run).** 0 hold-point rows on the default 46-item scope, and 0 on the shape's 67-item scope, which adds sprinklers, ceilings, ground investigation and temporary power. | Only four interfaces have system-id hold points (`holdPointRows`, `internal/program/generate.go`). `if.reinforcement-inspected-before-pour` needs a row tagged `structure.concrete`; `fr.slab-7b`'s pour row carries only `substructure.slab-on-ground`. `if.sprinkler-pressure-test-before-ceilings` needs an `interiors.ceilings` row; the only ceilings fragment is `fr.ceilings-fitout`, which needs refurb. The frame interface needs `structure.timber`. `if.services-modify-structural-framing` (fire stopping) needs `fire-passive.penetrations`, which is not in scope. |
| Mixed-use rows switch on because one part carries 7b and 5 | **Confirmed (Go run).** With one whole part holding `7b,5`, `flag.mixed_use` = `{yes, derived, "parts with different classes"}`. Architecture `mixed_use_map` is I, fire-engineering `occupants` I, access `shared_routes` I and waste `shared` O, all as the shape predicts. With `7b` alone the flag is `no` and all four go. | `deriveFlags` counts distinct classes across the parts, and the whole part counts when there are no others (`parts`, `internal/rfp/drivers.go`). **New:** `FlagHome("mixed_use")` is `HomeDerived` (`internal/rfp/homes.go`), so a user cannot set `proj.flag.mixed_use: no`. The only cure is to drop class 5. That also changes the architecture RFP: with 7b alone, scale is simple, `interiors` is excluded, `base_interiors` is absent, and `specification` and `typical_interiors` drop to O. See finding 10. |
| Class 5–9 critical-stage inspections | **The author's list is the old wording.** "At the commencement of the building work; before covering stormwater drainage connections; final before the OC" is cl 162A(4) of the 2000 Regulation (still on a Cabonne Council form). Bega Valley Council, citing cl 61 of the EP&A (Development Certification and Fire Safety) Regulation 2021, gives Class 5–9 as **first footing (after excavation, before the first footing), stormwater, final**. A June 2026 Tamworth contract, quoted in search results, says the same. I could not open the Regulation itself. | Bega Valley, "About inspections": https://www.begavalley.nsw.gov.au/Plan-and-build/Building-and-certification/About-inspections. The T02 review used the same source for Class 2. All stay `verified: false`. Finding 3. |
| C3D4 thresholds (18,000 m², 108,000 m³) | **Confirmed against the instrument. Set `verified: true`.** C3D4(1)(b): a Class 5–9 building of at most 18,000 m² and 108,000 m³ may exceed Table C3D3 if it is sprinklered (Spec 17) and has perimeter vehicular access to C3D5(2). C3D4(2): above either figure, the **same two conditions**. C3D5(2): access at least 6 m wide and unobstructed, with its furthest boundary no more than 18 m from the building. **The conditions in the trigger come from other clauses.** E1D15(b): a fire control centre for a Class 6, 7, 8 or 9 building with a total floor area over 18,000 m² (floor area only, not volume). E2D10(2): above 18,000 m² **or** 108,000 m³, smoke exhaust (Spec 21) or smoke-and-heat vents (Spec 22) where the ceiling is ≤ 12 m, and smoke exhaust only where it is > 12 m. E2D10(1): at or below both limits, sprinklers plus perimeter vehicular access satisfy smoke hazard management. E1D13: combustible goods stored above 4 m, over 1,000 m³ in aggregate, in a compartment over 2,000 m² or 12,000 m³ need sprinklers. That is the base's own sprinkler trigger. | ABCB NCC 2022 Vol One, Part C3 (https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c3-compartmentation-and-separation), Part E1 (…/e-services-and-equipment/part-e1-fire-fighting-equipment) and Part E2 (…/part-e2-smoke-hazard-management). Clause text read through the fetch tool. Finding 4. |

## Findings

1. **`shape.yaml program.activities`, the two CC rows: the fragment sits on the wrong CC, and both
   rows are in the wrong stage.**
   - *Wrong:* `fr.construction-certificate` and the alias "Construction certificate" sit on "Construction
     certificate: early works", and "Construction certificate: building" has `fragment: null`. Both rows
     are `stage: approvals`, yet both come after the award and the novated documentation. Under
     `sk.new-dc-da`, approvals is a stage that finishes before procurement, so the rows' stage and their
     links contradict each other.
   - *Evidence:*
     - T02 F1 (applied) and T13 F7 and its staged-CC item (applied, confirmed in round 2) both put the
       app's single CC fragment on the **main** CC and leave the early-works CC as `null`/`no`.
     - T13 F7 ruled that the expected stage is the *correct* stage, and that `wrong_stage` records the
       app's difference.
     - `CONVENTIONS.md` says nothing on this. C1–C3 cover novation, BASIX/ESD and RFP pairing only.
     - The scorer pairs by label, then alias (`_program_rules`). So as written, the early-works row claims
       the app's "Construction certificate" and the building CC is scored a miss.
   - *Change:*
     - "Construction certificate: early works": `fragment: null`, drop `aliases`, `built_in: "no"`,
       `stage: construction_documentation`, why "Bulk earthworks and civil; the app has one CC row only".
     - "Construction certificate: building": `fragment: fr.construction-certificate`,
       `aliases: ["Construction certificate"]`, `built_in: "partial"`,
       `source: "knowledge/program/fragments.yaml fr.construction-certificate"`,
       `stage: construction_documentation`, why "Main CC with the fire safety schedule after the
       contractor's design and FRNSW consultation; the app puts one CC in approvals before procurement".
     - Rewrite the matching `gaps` row: "One CC fragment, in approvals before D&C procurement; no staged
       CCs. Expected: an early-works CC, then a building CC, both after award".

2. **`shape.yaml program.activities`: building work is not gated on the building CC.**
   - *Wrong:* "Excavate and pour pad footings" follows only compaction testing and set-out. Nothing
     after the early-works CC waits for "Construction certificate: building". Footings, the frame, the
     panels and the slab are building work under the second CC, not the early-works one.
   - *Evidence:* NSW: building work needs a CC covering it before it starts (verified: false). T02 R2
     made the same correction ("bulk excavation is building work under the early-works CC").
   - *Change:* add "Construction certificate: building" to the `after` of "Excavate and pour pad footings"
     (and so of everything downstream). Keep "Fabricate the portal frame" and "Manufacture precast wall
     panels" off it: off-site fabrication is not building work on the site.

3. **`DOMAIN.md` §5, §11 gap 7, §12, and `shape.yaml` program and certifier RFP: the Class 5–9
   critical-stage list.**
   - *Wrong:* the list says "commencement inspection", which is the 2000 Regulation's wording.
   - *Evidence:* see the table above (Bega Valley, citing cl 61 of the 2021 Regulation).
   - *Change:*
     - Split "Notice of commencement and commencement inspection" into two rows:
       - "Notice of commencement" (construction, after "Construction certificate: early works" and
         "Contractor provides security and insurances", `built_in: "no"`);
       - "Inspection after excavation, before the first footing" (construction, after "Building set-out
         survey" and "Construction certificate: building", `built_in: "no"`, why "Class 5–9 critical
         stage, cl 61 EP&A (DC&FS) Reg 2021 (verified: false)").
     - Make "Excavate and pour pad footings" follow the new inspection row.
     - Keep the stormwater and final rows.
     - DOMAIN §5: replace "at the commencement of building work" with "after excavation and before the
       first footing", and cite Bega Valley. Note that the CC lists the inspections that apply (as R7
       says for Class 1).
     - Certifier `must_include`: "…mandatory inspections for a Class 5–9 building (first footing,
       stormwater, final; verified: false)".

4. **`DOMAIN.md` §1 Fire, §9 trigger 3, §12, and `shape.yaml` trigger
   `large-isolated-building-over-limit`: the C3D4 thresholds and conditions.**
   - *Wrong:*
     - The thresholds are marked `verified: false`, but the instrument confirms them.
     - Evidence phrase 2 ("the volume exceeds 108,000 m3, so the building must be sprinklered with
       perimeter vehicular access") describes the **base** too, because C3D4(1)(b) already requires
       both. It cannot tell the variant from the base.
     - The fire control room is added on any exceedance, but E1D15 counts floor area only.
   - *Evidence:* see the table above.
   - *Change:*
     - Mark the C3D4, C3D5(2), E1D13, E1D15(b) and E2D10 figures `verified: true`, with the ABCB URLs.
     - Evidence phrase 2: "the fire compartment exceeds 18,000 m2, so smoke exhaust and a fire control
       centre are required".
     - In `why`: "fire control centre only where the total floor area exceeds 18,000 m² (E1D15(b)); smoke
       exhaust or smoke-and-heat vents where the area or volume exceeds the limits (E2D10(2)); smoke
       exhaust only where the ceiling is over 12 m". Since the base clear height is 12 m at the haunch,
       the variant expects smoke exhaust.
     - Negation: give a volume that adds up. 7,500 m² at a 12 m haunch, plus the roof rise, is already
       close to 100,000 m³. Add a `scenario_inputs` row for `det.floor_area` with phrasing "Floor area
       8,000 m2; volume of the single fire compartment about 100,000 m3 (below the C3D4(1) limits of
       18,000 m2 and 108,000 m3)". This stops a scenario author from writing an over-limit base by
       accident.
     - Add to the base fire `why` (DOMAIN §1, `fire-active.sprinklers`): "sprinklers required by C3D4(1)(b)
       and E1D13 (storage over 4 m)".

5. **`shape.yaml rfps` (12 templates): `app_differs` misses the keys where the app answers `ask` or
   `option`.**
   - *Wrong:* the scorer counts any key where the app differs from `expect` and no `app_differs` entry
     explains it as a `new_failure` (`_compare_expect`). A client-flag `ask` on a key `expect` calls
     `absent` is "present" to the scorer.
   - *Evidence:* the Go run (`rfp.Resolve`, base profile, budget stated). App values below.
   - *Change:* add these entries (`expect` as the shape already has it):
     - bca: `position.existing_assessment`, `fire_referrals`, `residential_evidence` and
       `staged_occupation` are `ask` (client flags `bca_*`); `stage.b2_construction_approval`,
       `b3_construction` and `b4_occupation` are `option` (after the a1..a3 boundary).
     - certifier: `position.classification_map`, `existing_assessment`, `residential_evidence` and
       `staged_occupation` are `ask`; `stage.a4_construction` is `option`.
     - fire_services: `position.upgrade` is `ask` (`fs_upgrade`).
     - fire_engineer: `position.brief` and `position.fer` are `ask` (`fe_solutions`);
       `position.strategy_review` is `O`; `position.existing` is `ask`; `stage.s8_specialist`,
       `s8_staging`, `s9_advisory` and `s9_peer` are `option`.
     - electrical: `stage.investigation` is `ask` (`elec_investigation`).
     - hydraulic: `position.process` and `position.treatment` are `ask`.
     - geotechnical: `position.ass`, `asset_exposure` and `handback` are `ask`.
     - surveyor: `position.buried` and `position.sewer` are `ask`.
     - landscape: `position.irrigation_detail`, `podium`, `public`, `special` and `staging` are `ask`.
       Also decide `position.establishment` and `position.construction`: the app gives both O and the
       shape does not list them. I would expect O, following the s7/s8 options.
     - quantity_surveyor: `position.allocation`, `operator` and `remediation` are `ask`.
     - waste: `position.operational`, `cd`, `collection` and `rooms` are `ask` (`waste_operational`,
       `waste_cd`).

   Architecture, structural, civil, town planning, traffic and access matched the shape's prediction
   exactly.

6. **`shape.yaml rfps` geotechnical and landscape: two `app_differs` entries are wrong.**
   - *Wrong:* geotech `stage.g4_planning` says `app: ask`; the app gives `option`. Landscape
     `position.design` says `app: ask`; the app gives `O`.
   - *Evidence:* Go run.
   - *Change:* set them to `app: option` and `app: O`.

7. **`shape.yaml rfps` architect `expect.blocking`.**
   - *Wrong:* it is `["fact:budget"]`, but `scenario_inputs` states the budget.
   - *Evidence:* with `proj.budget_stated` set, the Go run shows no `fact:budget` check. The T01 precedent
     (STATUS, 30 October) set the architect's `blocking: []` for this reason.
   - *Change:* `blocking: []`. Change the `sources` line to "blocking: none; the scenario states the
     budget, so the fact:budget block must clear".

8. **`shape.yaml expected_derived`: three values the app does not produce today.**
   - *Wrong:* `det.type_of_construction: C`, `det.max_floor_area_m2: 2000` and `det.max_volume_m3: 12000`
     are given with no word that the app returns nothing.
   - *Evidence:* the Go run's `profile.Build` gives blank, with `Derived.State unknown`, reason
     `unreviewed`, for both `7b,5` and `7b` (`knowledge/evaluate.go`: an unreviewed rule or table
     returns unknown). Brief 01's example says to note "returns unknown today".
   - *Change:* add "returns unknown today: rule/table unreviewed (Go run)" to each note. The values
     stay, because they are the correct answers.

9. **`DOMAIN.md` §11 gap 6, `shape.yaml` gaps row and the "Reinforcement inspection" activity: the
   hold-point wording.**
   - *Wrong:* the text says "read, not run" and names a "services-before-paving interface". No such
     system-id hold point exists. Only four do (see the table).
   - *Evidence:* Go run, 0 hold points with either scope.
   - *Change:* replace "(read, not run)" and "Not confirmed by a Go run" with "(confirmed by a Go run,
     default and expected scope)". Drop "services-before-paving". Add: "fr.fire-stopping applies to
     class 5, so the fire-stopping hold point would fire (before fr.painting-7b) if
     fire-passive.penetrations were added".

10. **`DOMAIN.md` §8 and §10 trap 12, `shape.yaml` gaps row "Two classes on one part…": the mixed-use
    gap is understated.**
    - *Wrong:* the text says the rows switch on, but not that the user cannot switch them off.
    - *Evidence:* `FlagHome("mixed_use")` is `HomeDerived`. Dropping class 5 is the only cure, and that
      flips the architect's scale and interiors stage (Go run).
    - *Change:* append "flag.mixed_use is derived only (internal/rfp/homes.go FlagHome); a user cannot
      override it, and removing class 5 from det.ncc_class also removes the office interiors stage and
      makes the architecture scale simple" to the gap and to §8.

11. **`shape.yaml triggers[traditional-lump-sum].built_in.program`: `yes` → `"partial"`.**
    - *Evidence:* `sk.new-traditional-da` exists and puts construction documentation before
      procurement. But there is no bidding or tender-support fragment, and the app's row is "Prepare
      construction documentation", not the label "…before tender".
    - *Change:* set it to `"partial"`, or alias the first added row to "Prepare construction
      documentation".

12. **`shape.yaml triggers[aboriginal-heritage].built_in.cost_plan`: `yes` → `"partial"`.**
    - *Evidence:* the Archaeologist fee line follows the package (`complexity_additions[4]`), but no
      starter or element line exists for "Test excavation and salvage".
    - *Change:* set it to `"partial"`.

13. **`shape.yaml program` "Bulk earthworks to the warehouse pad": the alias is not an app label.**
    - *Wrong:* `aliases: ["Prepare the warehouse pad"]` is not what the app shows.
    - *Evidence:* a single-activity 7b fragment row takes the work item's title. The Go run shows
      "Earthworks, cut and fill" on the default scope. Matching still works through `fragment:
      fr.earthworks-7b`, so the alias does nothing.
    - *Change:* set it to `aliases: ["Earthworks, cut and fill"]`. The same applies, if wanted, to:
      - "Pad and strip footings";
      - "Pitched roof coverings";
      - "External walls and cladding";
      - "Fire hydrants and hose reels";
      - "Site stormwater, detention and water quality";
      - "Landscape, planting and external grading";
      - "Internal painting and coatings";
      - "Sanitary plumbing and drainage";
      - "Submains, final subcircuits and wiring";
      - "General lighting and lighting controls";
      - "Erosion, sediment and dust control".

14. **`DOMAIN.md` §1 Pathway and §5: the staged-CC order needs one sentence.**
    - *Wrong:* both sections say "construction certificates, usually staged". They never say that in
      this D&C program both CCs follow award and the contractor's design, or that the PC is appointed
      and the notice of commencement is given after the early-works CC.
    - *Change:* add "On D&C both CCs follow award and the contractor's design. Where the developer lets
      bulk earthworks under a separate early-works contract before the D&C award, the early-works CC
      moves ahead of the award (a variant, not the base)". This matches the T02 and T13 treatment.

**Program order: my ruling.** CONVENTIONS.md has no rule on CC order. Under the reviewed T02 and
T13 handling, the right D&C order is:

1. DA determination;
2. D&C tender and award;
3. novation;
4. the contractor's documentation;
5. the early-works CC;
6. the notice of commencement and the PC appointment;
7. earthworks;
8. the building CC, after the principal's review and the FRNSW consultation;
9. the first-footing inspection, then the footings.

Every CC is `stage: construction_documentation`, and the app's single CC sits on the building CC.
The rest of the T04 order (procurement before construction documentation, and services applications
right after award) is correct.

**Not findings (checked and right):**
- Consultant set: the fire engineer, landscape, acoustic, QS and waste are needed, and mechanical is
  not; the base matches the app's 14 suggestions except mechanical.
- The novation split (C1) and C2's `consultant.esd`.
- R3 (no DBP Act for 7b).
- The keyword traps; I spot-checked `\bflood`, `\bbore\b`, `road reserve` and `contaminat` in
  `determinants.yaml` and `flag_readings.yaml`.
- No prices, rates or durations.
- The off-site precast sequence after the roof.

## Built-in claims checked

Go = confirmed by the Go run; R = confirmed by reading the cited file and the selection logic.

**Procurement**
- ✔ Go: architect, town planner, surveyor, geotechnical, civil, structural, traffic, BCA, certifier,
  fire services, electrical, hydraulic and access, all suggested. The run's classes were
  `[commercial industrial]` (class 5 adds commercial); work type `new`.
- ✔ Go: `trade.main_works` `partial` (no head contract until items are accepted; R
  `ProposeHeadContract`).
- ✔ Go: not_expected mechanical, suggested.
- ✔ Go (`no` spot-checks): fire engineer, landscape, acoustic, ESD, QS and waste are never suggested.

**Cost plan**
- ✔ Go: `da-fees`, `cc-fees`, `service-authority` (one generic line), design and construction
  contingency, escalation, `exclusions`.
- ✔ Go: fee lines for each of the 14 suggestions.
- ✔ Go: elements 01–05, 07–11 and 13, with the three code 13 children (accessible paths, sanitary,
  maintenance access).
- ✔ Go: "06 Passive fire protection" made from `fire-passive.compartments` (not_expected).
- ✔ Go: stairs and ramps, and egress doors, are absent from the default scope (`partial` is right).
- ✔ R: hardstand and kiosk substation `partial` (rolled into 01 and 10).
- ✔ Go (`no` spot-checks): no line for HPC, long service levy, road reserve, preliminaries, D&C design
  or provisional sums.

**Program** (Go unless marked)
- ✔ skeleton `sk.new-dc-da`.
- ✔ every `yes` row is present by fragment:
  - concept-design-7b ×3, design-development-7b, approval-docs-7b ×3;
  - da-lodgement, da-exhibition, da-assessment;
  - tender-docs, tender-period and tender-evaluation 7b, contract-award, novation;
  - construction-docs-7b ×2, site-establishment-7b, erosion-control, earthworks-7b, stormwater-7b;
  - services-connections-major ×2, utility-major ×2;
  - footings-7b, steel-7b ×2, roof-7b, cladding-7b, slab-7b ×3;
  - vehicle-doors-7b ×2, hydrants-7b, switchboard-major, electrical-7b, lighting-7b;
  - hydraulic-sanitary-7b, landscape-7b, painting-7b;
  - practical-completion, occupation-certificate, handover, defects-inspection, defects-liability-end ×2.
- ✔ `partial`:
  - investigate survey, services and geotech: fragments exist and fire only on `investigate`; geotech
    fired in the shape-scope run;
  - temporary power: fires once added;
  - sprinklers: `fr.sprinklers-7b` fires once added;
  - "Principal's review" (app label "Client sign-off for tender").
- ✘ CC rows (finding 1).
- ✔ `no` spot-checks: no fragment for precast, fire pumps and tanks, hardstand, docks, office fit-out,
  pre-DA meeting or fire strategy.
- ✔ hold points: none fire (Go, both scopes).

**RFPs (`partial`: template present)**
- ✔ Go: architecture, structural, civil, town planning, traffic and access match `expect` plus
  `app_differs` exactly, including `mixed_use_map` I and `shared_routes` I.
- ✘ bca, certifier, fire services, fire engineering, electrical, hydraulic, geotechnical, surveying,
  landscape, cost planning and waste have incomplete `app_differs` (findings 5 and 6). Their
  `template` claims are right, and fire-engineering `occupants` I and waste `shared` O are confirmed.
- ✘ architect blocking (finding 7).
- ✔ R: acoustic and ESD have no template (`RFPForDiscipline` returns no template, per index §9 #9 and
  `rfp-templates.md`).

**Triggers** (R unless marked)
- ✘ `traditional-lump-sum`: program should be `partial` (finding 11). Procurement `yes` is right
  (mechanical is already suggested), and so are rfps `yes` (traditional stages through the boundary)
  and cost_plan `no`.
- ✔ `cdc-industrial-code`: program `partial` (`fr.cdc-application`). No `sk.new-dc-cdc` exists, so
  `sk.generic` is right. rfps `no`: both templates block on cdc. cost_plan `partial`.
- ✔ `large-isolated-building-over-limit`: procurement, work items and program `no`; rfps `partial`;
  cost_plan `partial`. The domain content is ✘ (finding 4).
- ✔ `tilt-up-on-site`: work_items `partial` (precast is already default, temporary works is not);
  program `no`.
- ✔ `traffic-generating-development`: rfps `yes` (traffic modelling and surveys on
  `intersection_modelling` and `tfnsw_referral`).
- ✔ `classified-road-frontage`: rfps `yes` (civil `public_road` element and approvals).
- ✔ `flood-affected`: program `partial` (`fr.investigate-flooding`); rfps `yes`; procurement rule dead,
  as stated.
- ✔ `contaminated-land`: procurement `no` (`contamination_level` is dead); program `partial`
  (`fr.investigate-contamination`).
- ✔ `bushfire-prone`: procurement `yes` (`complexity_additions` `bal`, BAL-LOW and up; needs `det.bal`
  set, which the trigger's profile names); fragments for 1a and 9b only.
- ✘ `aboriginal-heritage`: cost_plan `partial`, not `yes` (finding 12). Procurement `yes` through
  `environmental_sensitivity: aboriginal_heritage` is right.
- ✔ `brownfield-demolition`: `fr.demolition-7b` needs `remove`, and `implied_by` gives `new`.
- ✔ `two-storey-office-lift`: `fr.lifts` covers class 5, which `det.ncc_class` carries; procurement
  `yes` (scope_additions[0]); cost code 12.
- ✔ `dangerous-goods-storage`: rfps `yes` (fire-engineering hazards on `fire_special_hazards`).
- ✔ `green-star-solar`: rfps `yes` (architecture `sustainability_docs` on `sustainability_target`,
  electrical `generation`).

## Notes for the orchestrator (outside T04)

- **T13 fire control centre.** NCC 2022 E1D15, read twice on the ABCB site: "(a) a building with an
  effective height of more than 25 m; and (b) a Class 6, 7, 8 or 9 building with a total floor area of
  more than 18 000 m2." T13 round 1 changed this to 50 m from memory, and 50 m is the NCC 2019 E1.8
  figure. T13 should go back to 25 m, with `verified: true` and this source.
- **`expected_derived` elsewhere.** T02 and T13 list `det.type_of_construction` the same way, and the
  derivation returns `unknown/unreviewed` today. Their notes need the same wording as finding 8.
- **Scorer blocking.** `_compare_expect` compares the whole set of blocking checks. In `rfp.Resolve`,
  every RFP also blocks on `slot:project:intro`, `fields`, `terms`, `details`, allowance checks and each
  unanswered `fact:flag.*` client flag. Every shape that says `blocking: []` will then report these as
  new failures, unless the harness's saved view filters them. Worth one decision before the pilot run.

## Round 1 response (author, 4 November 2026)

Applied 13, disputed 1 in part (finding 1, fragment placement, under C5) and corrected 1 detail of
finding 10 from my own Go run. I re-ran the Go read-only, with no database and no Jev:
`go run -overlay` mapping a virtual `internal/zzt04author/main.go` to a scratch file. It ran
`rfp.Drivers` and `rfp.Resolve` for all 17 templated RFPs on the shape's profile (budget stated, package
novation `option`), and `profile.Build` for `expected_derived`. Since 3 November the app lays these
templates out by stage, which turns every option and ask into include (`placeServices`). I set
`ScopeByStage` off on the loaded discipline so the comparison sees the rules' own answers, which is what
the shapes record and what `score_shapes.py` translates. After the edits, the predicted output
(`expect` overridden by `app_differs`) matches the Go output on every stage, element and position of all
17 RFPs.

| # | Finding | Response |
|---|---|---|
| 1 | Staged CC rows | **Applied in part, disputed in part.** Both CCs are now `stage: construction_documentation`, after award and the contractor's documentation, and the gap row uses your wording. **Disputed:** I left the app's fragment and the alias "Construction certificate" on the **early-works** CC, and the building CC is `fragment: null`, `built_in: "no"`. CONVENTIONS C5, written after this review and citing it, says the app's CC fragment row corresponds to the early-works CC, and the building CC is recorded as built_in no. C5 binds the maps to one another, so I follow it. The order now follows your ruling: early-works CC (after documentation and the long service levy), then PC appointment, then notice of commencement. |
| 2 | Building work not gated on the building CC | **Applied.** "Excavate and pour pad footings" follows "Construction certificate: building". Frame fabrication and precast manufacture stay off it. |
| 3 | Class 5–9 inspection list | **Applied.** "Notice of commencement" and "Inspection after excavation, before the first footing" are separate rows, and the footings follow the inspection. The stormwater row cites cl 61 (verified: false). DOMAIN §5 now gives first footing, stormwater and final, cites Bega Valley, and says the CC lists the inspections. The certifier `must_include` names all three. §11 gap 7 and §12 are updated. |
| 4 | C3D4 and the trigger conditions | **Applied.** C3D4, C3D5(2), E1D13, E1D15(b) and E2D10 are `verified: true` with the ABCB URLs (DOMAIN §1, §9, §12; trigger `why`). Evidence phrase 2 is your wording. I also changed "fire control room" to "fire control centre" in phrase 3 and the cost line, per C4 (S19C7's separate room applies above 50 m). The cost line is now "Smoke exhaust", because the base haunch is 12 m. The negation and a new `scenario_inputs` row for `det.floor_area` give about 100,000 m³. The base sprinkler `why` cites C3D4(1)(b) and E1D13. |
| 5 | Missing `app_differs` | **Applied, confirmed by the Go run.** Every key you listed was added (41 entries across 11 RFPs). For landscape `position.establishment` and `position.construction` I took your suggestion: `expect` O, which matches the app. |
| 6 | Two wrong `app_differs` | **Applied, confirmed.** Geotechnical `stage.g4_planning` app `option`; landscape `position.design` app `O`. |
| 7 | Architect `blocking` | **Applied, confirmed** (no `fact:budget` with the budget stated). `blocking: []`, with the sources line and the RFP `why` reworded. |
| 8 | `expected_derived` | **Applied, confirmed** (`profile.Build`: state unknown, reason unreviewed, for all three). The values stay. |
| 9 | Hold-point wording | **Applied.** "Confirmed by a Go run, default and expected scope". "Services-before-paving" is dropped, and the fire-stopping note is added (DOMAIN §11 gap 6, the gap row, the reinforcement row). |
| 10 | Mixed-use gap understated | **Applied, with one correction.** The FlagHome sentence is in DOMAIN §8, §10 trap 12 and the gap row. My Go run with `det.ncc_class: 7b` alone confirms that the interiors stage is excluded and `base_interiors` is absent. It does **not** confirm the rest: the architect scale stayed `complex`, and `specification` and `typical_interiors` stayed I. So the text says only what the run showed. |
| 11 | `traditional-lump-sum` program | **Applied.** `"partial"`. |
| 12 | `aboriginal-heritage` cost plan | **Applied.** `"partial"`. |
| 13 | Program aliases | **Applied.** Earthworks takes "Earthworks, cut and fill", and the other 11 rows take the work-item titles you listed. I checked that each of these fragments has one activity, so `generate.go` uses the item title (line 648). |
| 14 | Staged-CC sentence | **Applied** in DOMAIN §1 Pathway and §5. |

Also applied from CONVENTIONS C4: the long service levy (0.25% of building work of $250,000 or more,
including GST, paid before work starts) is in the cost line, the program row and DOMAIN §5 and §12. The
WHS Regulation reference is now the NSW 2025 Regulation, with the clause still verified: false.

**New counts:** 15 triggers, 20 packages, 49 cost lines, 95 activities (was 94: the notice of
commencement and the first-footing inspection are now two rows), 19 RFPs, 26 gaps.

**Validator:** `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T04-warehouse/shape.yaml`
→ `ok`.

## Round 2 review (4 November 2026)

Reviewer: the round 1 reviewer. Re-read `shape.yaml`, `DOMAIN.md` and the Round 1 response; neither file
was edited. Validator: `ok`, exit 0. A script check found no dangling `after` references, no duplicate
activity labels and no duplicate aliases across the 95 activities.

### Verdict: **PASS**

### Fixes checked

- **F1 ✔ (dispute upheld).** Both CCs are `stage: construction_documentation`, after award and the
  contractor's documentation, and the gap row has the round 1 wording. Leaving
  `fr.construction-certificate` and the alias "Construction certificate" on the **early-works** CC, with
  the building CC `fragment: null`, `built_in: "no"`, is exactly what C5 says ("The app's CC fragment row
  corresponds to the early-works CC … record the building CC as built_in no"). C5 postdates round 1 and
  binds the maps to each other, so it overrides my round 1 suggestion. The scorer reports the app's
  approvals placement through `wrong_stage` (`score_shapes.py` line 521), so nothing else is needed. The
  order matches my ruling: levy → early-works CC → PC appointment → notice of commencement → site
  establishment → earthworks; building CC after the principal's review, the FRNSW consultation and the
  early-works CC. DOMAIN §1 Pathway, §5 item 1 and §7 agree and cite C5.
- **F2 ✔.** "Excavate and pour pad footings" follows "Construction certificate: building"; frame
  fabrication and precast manufacture stay off it.
- **F3 ✔.** Notice of commencement and the first-footing inspection are separate rows; the footings
  follow the inspection; cl 61 stays `verified: false`; DOMAIN §5 cites Bega Valley and notes the CC lists
  the inspections; the certifier `must_include` names all three.
- **F4 ✔.** `verified: true` with the ABCB URLs in DOMAIN §1, §9, §12 and the trigger `why`. Phrase 2
  now distinguishes the variant from the base. The fire control centre is tied to floor area only
  (E1D15(b), C4). The negation and the `det.floor_area` `scenario_inputs` row give a base inside both
  limits. The base sprinkler `why` cites C3D4(1)(b) and E1D13. "Fire control centre" per C4 is right.
- **F5, F6 ✔.** The 41 entries and the two corrections are in. They were verified against the rules'
  own answers (`ScopeByStage` off); see the cross-cutting section for what that means at scoring time.
- **F7 ✔.** Architect `blocking: []`, sources line reworded.
- **F8 ✔.** All three `expected_derived` notes say "Returns unknown today: rule/table unreviewed".
- **F9 ✔.** Hold-point wording and the fire-stopping note.
- **F10 ✔ (correction accepted).** The author's run is right and my round 1 claim about scale was wrong:
  `knowledge/rfp/architecture.yaml` `scale` goes complex on "a budget over $20 million" as well as on a
  Class 2–6 part, so dropping class 5 leaves this budget complex, and `specification` and
  `typical_interiors` stay I. The text now says only what the run showed.
- **F11, F12 ✔.** `traditional-lump-sum` program `"partial"`; `aboriginal-heritage` cost plan
  `"partial"`.
- **F13 ✔.** The 12 aliases are the work-item titles of single-activity fragments.
- **F14 ✔.** The staged-CC sentence is in DOMAIN §1 and §5.
- **C4 extras ✔.** Long service levy cost line, program row (before the early-works CC) and DOMAIN text;
  WHS Regulation 2025 with the clause `verified: false`.

### Remaining items (optional, not blocking)

1. **The first-footing inspection precedes the row that excavates.** "Inspection after excavation, before
   the first footing" sits before "Excavate and pour pad footings", so the excavation it inspects is
   inside the following row. Scoring is unaffected (the footings row matches by fragment), but a reader
   of the answer key sees the inspection before the dig. If wanted: relabel the footings row "Pour pad
   footings after the excavation inspection", keeping its alias "Pad and strip footings".
2. **Smoke exhaust at a 12 m haunch.** E2D10(2) allows smoke-and-heat vents where the ceiling is ≤ 12 m.
   With a 12 m haunch and a higher apex, "smoke exhaust" is defensible but not certain. The cost line
   could read "Smoke exhaust (or smoke-and-heat vents)" so a scenario stating vents is not a miss.
3. **DOMAIN §11 gap 4** now runs the new wording into the old sentence on one long line; cosmetic.

### Cross-cutting: `rfp.Resolve` and scope by stage

**Confirmed, with three exceptions.** `Resolve` calls `placeServices` (`internal/rfp/resolve.go` line
309), which returns at once unless the discipline has `scope_by_stage: true`. Since 3 November all 18
`knowledge/rfp/*.yaml` disciplines set it. Then (lines 992–1068):

- a stage at `option` or `ask` becomes `include`; **except** an `option` set by the boundary when the
  user set the boundary, which becomes `exclude`; and `done` becomes `exclude`;
- an element at `option` or `ask` becomes `include`;
- a position at `O` or `ask` becomes `I`; **except** a position whose home stage (the first listed stage
  the project has) is excluded, which becomes `X` (an `I` position moves to the next included stage
  instead, unless the user excluded the stage).

So the issued RFP has only I/X and include/exclude, and the rules' `option`/`ask` answers are no longer
visible in it.

**What it means for the shapes.** The `expect` blocks and the `app_differs.app` values in every shape
record the rules' answers (option, ask, O). That is the right thing to record and the shapes should
**not** change: the owner direction changed the layout, not the rules, and the T04 author's check
(`ScopeByStage` off) confirms the recorded values are what the rules give.

**The scorer is half translated, and should change.** `score_shapes.py` `_compare_expect` already maps
`expect` through `BY_STAGE` when the view has a `stage_scope` block (lines 595–622). It does **not**
map `app_differs.app`. So for every entry where `app` is `ask`, `option` or `O`:

- if the translated `expect` equals the translated app value (e.g. expect `include`, app `option`), the
  key is reported as **fixed**, though nothing in the app changed;
- if not (e.g. expect `absent`, app `ask`, now shown as `include`), the app value no longer equals the
  untranslated `app`, so the key is a **new_failure**, though `app_differs` explains it.

A static simulation over the 15 shapes (`app_differs` entries with app ask/option/O, translated as the
scorer does) gives about **401 false new failures and 482 false "fixed"** keys (T04: 65 and 71; T02: 51
and 87; T03: 55 and 42). Every live score of an RFP's `expect` since 3 November is noise until this is
fixed.

**Recommended scorer change (no app change, no shape change):** when `by_stage`, translate `d["app"]`
with the same map before comparing; add `done → exclude`; and for a `position.*` whose
home stage resolves to `exclude`, expect `X` (or read the app's own home stage from the view). A key whose
translated `app` equals its translated `expect` should count as `ok` (the difference is hidden by the
layout), not `fixed`; a key whose translated `app` differs from the translated `expect` stays a
`known_difference`, which keeps visible that the issued RFP now prices an `ask` service the answer key
calls absent. That last group is a real product question for the owner: under scope by stage, an open
client question is priced as included.
