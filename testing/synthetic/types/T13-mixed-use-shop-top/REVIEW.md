# T13 Mixed-use shop-top housing: independent review

Reviewer: independent agent acting as a senior NSW PM for developer mixed-use jobs. Brief:
`testing/synthetic/briefs/02-review.md`. Read 30 October 2026 against `main` plus the working tree. This is
the only file written. The only things run were the validator, read-only Python over the YAML, and greps of
the knowledge and Go.

## Verdict: **FIX**

The map is wide and mostly accurate. It covers every top-level system, the Class 2 DBP/RAB chain, the
mixed-class fire separation and the retail shell boundary. Its built-in claims hold up well against the
knowledge. It is not ready to be an answer key, for these reasons:

- one 25 m consequence is wrong: the fire control centre threshold is 50 m;
- the strata bond is cited to the wrong Act;
- novation is not keyed per package, and is overstated for the services engineers;
- two app hold points will put activities in the wrong order, and the key hides this;
- the base scope contradicts the `public-domain-works` negation;
- four triggers named in the prose are missing, and so is the commonest variant, a rise of three storeys.

All of these can be fixed in place. Nothing is wrong at the root.

## Rulings on the author's open questions

| Question | Ruling |
|---|---|
| D&C with novation as the base route | **Right.** A developer shop-top of $15–80M in Greater Sydney is usually taken to DA and design development, then let as a D&C lump sum with the design team novated. Keep `hdr.cond.procurement_route: design_construct`, `fact.contract_form: as4902` and `traditional-lump-sum` as the variant. Two corrections: novation must be keyed per package (finding 4), and the services engineers are often *not* novated, because D&C services subcontractors take over the design. |
| The 25 m effective-height thresholds | **Right for stair pressurisation, emergency lifts and EWIS; wrong for the fire control centre.** NCC 2022 Vol One requires a fire control centre (or room) at an effective height over **50 m**, not 25 m (verified: false; my recollection, not checked against the instrument). Keep every threshold `verified: false` (AGENTS.md rule 5). See finding 1. |
| HBCF exemption above 3 storeys | **Right in substance.** As I recall it, the exemption is for a building with a **rise in storeys of more than 3 that contains 2 or more dwellings**, not "over three storeys" loosely. A rise of 6 is exempt. It stays `verified: false` under R4. The rise-of-3 shop-top, which is not exempt, should be a variant (finding 9). |
| Strata building bond | **Right that it applies, wrong Act.** The bond and its defect inspection scheme are in Part 11 of the **Strata Schemes Management Act 2015**, not the Strata Schemes Development Act. The developer lodges it before applying for the OC. The amount (2% of the contract price, as I recall) and the inspection windows stay `verified: false`. See finding 2. |
| 25 packages and 23 RFPs | **About the right size, not over-procured.** 20 packages are `always`, 3 are `usually` (waste, environmental, fire engineer) and 2 are `optional` (vertical transport, waterproofing). That matches the 18–22 firms a developer appoints for a 40–50 unit shop-top with retail and two basements. Each `usually` or `optional` package is justified, and the scorer copies `need` onto each miss (SCORING.md, Procurement). What is missing is a Water Servicing Coordinator (finding 15). The five RFPs with `template: null` will always score `missing`; they are right to keep as knowledge gaps. |

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T13-mixed-use-shop-top/shape.yaml`,
run from the repo root, gives `ok` (exit 0, no warnings). Run from `testing/synthetic/` it crashes with
`FileNotFoundError: knowledge\profile\taxonomy.yaml` (`load_ids(Path("."))`), so it must be run from the root.

Counts: 13 triggers, 25 packages, 55 cost lines, 97 activities, 23 RFPs and 86 work items. The `det.*` values
are not checked by the validator. I checked them against `knowledge/determinants.yaml` and all are valid
options.

## 2. Findings

1. **`shape.yaml triggers[effective-height-over-25m]` and `DOMAIN.md §2 fire-active`: the fire control
   centre is not a 25 m consequence.** Its threshold is an effective height over 50 m (NCC 2022 Vol One Part
   E1; verified: false). Stair pressurisation, an emergency lift and EWIS are the 25 m items.
   **Change:**
   - remove `{system: fire-active.fire-control, action: new}` from `adds.work_items`;
   - rename the cost line "EWIS and fire control room" to "Emergency warning and intercom system (EWIS)";
   - in DOMAIN §2, change `fire-active.smoke-control`, `.fire-control` to "smoke control and stair
     pressurisation over 25 m; fire control centre over 50 m (both verified: false)";
   - add a keyword trap: a rooftop communal room, as opposed to an open terrace, can become the topmost
     storey and raise the effective height.

2. **`DOMAIN.md §5 "Home warranty and bond"` and `shape.yaml program` "Strata bond interim and final
   inspections" `why`: wrong Act.** **Change** "Strata Schemes Development Act 2015 Part 11" to "Strata
   Schemes Management Act 2015, Part 11 (verified: false)" in both places and in §12. Also change the HBCF
   sentence to "HBCF is not required for a building with a rise in storeys of more than 3 containing 2 or
   more dwellings (R4; verified: false)".

3. **`shape.yaml procurement[*]`: novation is not keyed.** The comment above `rfps` says the architect,
   structural, civil, hydraulic, fire services, mechanical, electrical and landscape packages are elected
   `yes` and the rest `no`, but no package carries a `novation` field. T02 does carry one, for example
   `testing/synthetic/types/T02-apartments/shape.yaml` line 173. As written, the app's "option on every
   consultant" (`DefaultNovation`) cannot be scored as wrong for the certifier, planner, traffic engineer or
   QS. **Change:** add `novation:` to every package, in T02's form:
   - `yes`: architect, structural, civil, landscape;
   - `option`: hydraulic, fire services, mechanical, electrical, because the D&C services subcontractors
     often take over the design; this is for the owner to confirm;
   - `no`: all the rest, including the certifier ("never novated").

   Then soften DOMAIN §4 to "novated, or replaced by the D&C services subcontractor's designer" for the four
   services engineers.

4. **`shape.yaml rfps` for hydraulic, fire services, mechanical and electrical, following finding 3.** If
   the owner rules that the services engineers are `option`, then `continuation`, `construction` and
   `handover` should be `stages_option`, not `stages_include`, and their `app_differs` rows go. If the owner
   rules `yes`, leave them as they are. Record whichever is chosen in OWNER_RULINGS.

5. **`shape.yaml program`, reinforcement hold point: an app logic defect is hidden.**
   `if.reinforcement-inspected-before-pour` has `hold_point: {after: structure.concrete, before:
   substructure.slab-on-ground}` (`knowledge/clusters/structure/interfaces.yaml:276`), and
   `holdPointLinks` links **every** `after` row to the hold, then the hold to **every** `before` row
   (`internal/program/generate.go:962`). With `substructure.slab-on-ground` scoped, as this shape does,
   "Floor cycle" and "Roof slab and plant plinths" come before "Reinforce and pour the ground slab"
   (`fr.slab-2`), so the ground slab is scheduled after top-out. `fr.slab-2` also duplicates
   `fr.basement-2`'s "Pour the basement slab". **Change:**
   - add `{label: "Reinforce and pour the ground slab", stage: construction, fragment: fr.slab-2, after:
     ["Excavate and pour footings"], built_in: "yes", source: "knowledge/program/fragments.yaml fr.slab-2"}`,
     with aliases or a note that it duplicates "Pour the basement slab";
   - change the hold row's `after` to `["Excavate and pour footings"]` and set "Pour the basement slab"
     `after` that hold, so the scorer checks the order;
   - add a `gaps` row: "Reinforcement hold point links every structure.concrete row before the ground slab,
     so a concrete-frame building pours its basement slab after the roof slab".

6. **`shape.yaml program`, sprinkler hold point: the same defect.** `if.sprinkler-pressure-test-before-ceilings`
   (`after: fire-active.sprinklers`) takes in `fr.sprinklers-2`'s "Fit off and commission sprinklers", so
   ceilings are forced after sprinkler fit-off. In practice heads are fitted off after the ceilings.
   **Change:**
   - add `{label: "Fit off and commission sprinklers", stage: construction, fragment: fr.sprinklers-2,
     after: ["Install ceilings"], built_in: "partial", source: "knowledge/program/fragments.yaml
     fr.sprinklers-2"}` so the scorer reports the inversion;
   - add a matching `gaps` row.

7. **`shape.yaml program`: "Construction certificate" and "Pay long service levy" are given stage
   `approvals`, but their `after` links point into later stages** (construction documentation and
   procurement). Under `sk.new-dc-da`, approvals comes before procurement. The key should state the
   *correct* stage and let `wrong_stage` record the app's difference. **Change:** set both to `stage:
   construction_documentation`. In the CC `why`, say "staged CCs after award and the regulated designs; the
   app puts the CC in approvals before procurement".

8. **`shape.yaml program` "Lodge the supply application and substation design": the logic is late.** The
   developer's electrical engineer lodges the DNSP connection application and starts the substation design
   after DA consent, before the D&C award, because it is the longest authority lead. **Change** `after` to
   `["Local planning panel determination"]`, and "Lodge services applications and lead-in connections"
   likewise, since the Section 73 application follows consent. Keep the app's position in `why`.

9. **`shape.yaml triggers`: common variants are missing.** DOMAIN §2 itself calls `site.trees`,
   `site.acid-sulfate-soils`, `substructure.piles` and `substructure.underpinning` "triggers", but none of
   them is one. T02 carries trees, acid sulfate soils and underpinning. **Change:** add:
   - `trees-affected`: `flag.arborist_tree_impacts`, `consultant.arborist`, `fr.tree-protection`, arborist RFP;
   - `acid-sulfate-soils`: `det.acid_sulfate_soils_mapped`, `fr.investigate-acid-sulfate`, an ASS management
     plan, civil earthworks suitability;
   - `underpinning-neighbour`: `substructure.underpinning`, structural `dilapidation` and
     `temporary_works_interfaces`;
   - **`rise-three-storeys`**: the commonest suburban and regional shop-top, two apartment levels over a shop.
     It adds HBCF (R4) and removes the strata building bond. Below a rise of 4 it also removes the Class 2
     sprinkler requirement (verified: false) and changes the type of construction, so the derived
     `det.type_of_construction` changes too.

   Optional, but common since 2023: an in-fill affordable housing bonus (Housing SEPP; it can push the job
   into SSD; verified: false), and regionally significant development (CIV over $30M goes to a Sydney
   district planning panel; verified: false). The second matters here because the type's $15–80M range
   straddles the threshold.

10. **`shape.yaml work_items site.public-domain` and `profile flag.public_road: "no"` against the
    `public-domain-works` trigger.** The base scope is "footpath reconstruction, vehicle crossing, kerb and
    gutter, street trees, awning in the road reserve under s138". The trigger's negation is "Only a new
    vehicle crossing is proposed in the road reserve". So the base itself would be read as `public_road`
    yes: the `public_road` triggers include `road reserve` and `works (?:in|within) (?:the )?(?:public
    )?road` (`knowledge/profile/flag_readings.yaml public_road`). **Change** the base `why` to "vehicle
    crossing, footpath making good, street trees; awning over the footpath". Add a scenario_input phrasing
    that denies public road works, for example "No works in the public road other than the vehicle crossing
    and making good the footpath".

11. **`DOMAIN.md §10`: three keyword traps the base documents will trip are missing.**
    - "dewatering of seepage" (DOMAIN §2 `substructure.basement-retention`) and any geotechnical "groundwater
      was not encountered" both hit the `groundwater` triggers (`dewater`, `ground ?water`). Base documents
      must say "no groundwater above the basement; seepage only".
    - Waste and loading vehicles written "MRV" or "HRV" hit `heavy_vehicles` (`\b[hms]rv\b`). Council waste
      trucks are often HRVs, which would make the traffic RFP complex.
    - "road reserve" in the awning or s138 text hits `public_road` (finding 10).

    **Change:** add all three to §10, and add negation phrasings to `scenario_inputs`.

12. **`shape.yaml profile det.existing_building: false` against the map's own expectations.** The base
    demolishes 1950s–70s shops, and the map wants:
    - the hazardous-materials implication, which needs `det.existing_building_year` (DOMAIN §3);
    - architect `existing_interfaces: I` (shape `rfps[architect] app_differs`).

    T10, also a demolish-and-build job, sets `det.existing_building: true` with `det.existing_building_year`.
    **Change (owner decision):** either
    - follow T10: set `det.existing_building: true` and `det.existing_building_year: 1965`, add a
      scenario_input with "The site contains existing buildings … built in 1965", then re-key the RFPs (the
      architect's `existing_interfaces` becomes I with no differ; the structural `existing_assessment` must
      then be expected `option` or `absent`, with an `app_differs` row); or
    - keep `false`, drop the architect `existing_interfaces` differ and say the hazmat implication cannot
      fire.

    The two maps must agree.

13. **`shape.yaml cost_plan` "Accessible sanitary facilities and hearing augmentation" `built_in: "no"` is
    wrong.** `knowledge/costs/elements.yaml:34` has `access-egress.accessible-facilities` with `costed:
    true`, and the shape scopes that item. It is the same case as "Demolition", which is marked `partial`.
    **Change** to `built_in: "partial"` with source `knowledge/costs/elements.yaml code 13`, why "fires once
    the item is scoped; not in the mixed default list".
    Likewise, `triggers[heritage-conservation-area].built_in.cost_plan: "yes"` should be `"partial"`: the
    Heritage fee line comes from the package, but "Facade retention and temporary support" has no line.

14. **`shape.yaml program`: activities the shape's own scope fires are missing, and one label is a
    composite.** With the shape's work items, the app also adds:
    - `fr.gas-commercial` "Install consumer gas" (`hydraulic.gas`, classes 2 and 6);
    - `fr.insulation-2`, `fr.lighting-2`, `fr.comms-2`, `fr.roof-drainage-2`, `fr.smoke-alarms-2`;
    - `fr.hydrants-2`, `fr.detection-2`, `fr.electrical-2`, `fr.hydraulic-cold-2`, `fr.hydraulic-hot-2`
      (`reference/index/type-predictions.md` T13; `reference/index/program-fragments.md`).

    These are correct activities for the job, so they belong in the key, not as `extra`. "Rough-in hydraulic,
    electrical and fire services" can match only `fr.hydraulic-sanitary-2`. **Change:**
    - split it into one activity per fragment, with the fragment ids;
    - add the activities above;
    - add the duplicate "Procure and deliver / Install" pair from `fr.switchboard-major`, so the duplicate
      shows as matched-twice, not hidden.

15. **`shape.yaml procurement`: no Water Servicing Coordinator.** Sydney Water's Section 73 application is
    made through an accredited WSC that the developer engages, often a civil or hydraulic firm.
    `data/intake/disciplines.json` has no id for it. **Change:**
    - add a `gaps` row (`area: procurement`, "no discipline id for a Water Servicing Coordinator");
    - add "Water Servicing Coordinator role or appointment for the Section 73 application" to the hydraulic
      RFP `must_include`.

16. **`shape.yaml rfps` for `consultant.bca` and `consultant.certifier`: the harness can pair them the wrong
    way round.** Both use template `bca-certification`. `score_rfps` scores a template match and a
    discipline match equally at 1.0 (`testing/synthetic/tools/score_shapes.py:676`). The BCA expectation can
    therefore claim the certifier's report. **Change:**
    - add a `gaps` row (`area: harness`, "RFP pairing should prefer the discipline over the template when two
      expectations share a template");
    - tell the orchestrator, because the fix belongs in the scorer, not this map.

17. **`DOMAIN.md §5 "Inspections and hold points"`: the Class 2 critical-stage list is shaky.** As I recall
    it, for Class 2–4 the 2021 Regulation lists: commencement; before covering waterproofing in wet areas,
    for at least 10% of the rooms with wet areas; before covering stormwater connections; and the final.
    "At fire-safety stages" is not on that list. **Change:**
    - drop "at fire-safety stages", or mark it unconfirmed;
    - add the 10% wet-area sample;
    - keep `verified: false`.

18. **`shape.yaml expected_derived det.effective_height` holds the value "under 25 m".** The determinant has
    `profile_group: classification`, so it can be set, and the 25 m trigger reads it. **Change:**
    - move it to `profile` as a number (for example `det.effective_height: 18.9`) and add a scenario_input,
      "Effective height: 18.9 m (BCA report)";
    - the variant then becomes a one-key change.

19. **Minor, consistency.** T13 uses `consultant.esd` for BASIX and NatHERS; T02 uses `consultant.basix`. ESD
    is defensible here because of Section J for the retail, but the orchestrator should settle one id for
    the Class 2 maps. In the `why`, say that `consultant.basix` is an accepted alias.

## 3. Built-in claims checked

Each claim was checked against the cited file and `reference/KNOWLEDGE_INDEX.md`, and against
`reference/index/type-predictions.md` T13 (which assumed a traditional route, so the skeleton and novation
were re-checked).

**Procurement**

- ✔ Architect, Surveyor, Geotechnical, Structural, Civil, Hydraulic, Electrical, Certifier: `baselines[0]`
  (new, not infrastructure).
- ✔ Town Planner: `complexity_additions[0]` (planning `da`), also `baselines[0]`.
- ✔ Fire Services, Mechanical, BCA, Access, Traffic: `baselines[1]` (class `mixed`).
- ✔ Vertical Transport: `scope_additions[0]`.
- ✔ `trade.main_works` `partial`: `ProposeHeadContract` works only on accepted items.
- Spot-checked `no` (confirmed, no package rule): ✔ Acoustic, ✔ ESD, ✔ Landscape, ✔ Waste, ✔ QS, ✔ Hazmat,
  ✔ Fire Engineer, ✔ Waterproofing, ✔ Environmental (`complexity_additions[2]` reads the non-existent
  `contamination_level`).

**Scope "not default" claims** (`scope_defaults.yaml categories[1]`, 62 systems):

- ✔ sprinklers, slab-on-ground, demolition, ground-conditions, contamination, occupant-warning,
  accessible-facilities, trade-waste, gas, loading-docks, flashings-seals, openings, external-walls,
  temporary-power, temporary-works and cctv are absent from the list.
- ✔ adjoining-structures, public-domain, waste-storage, facade-acoustics, carpark-ventilation,
  livable-housing, acoustic-separation, hydrants and detection are present.

**Cost plan**

- ✔ P04 `da-fees` (planning `da`), P05 `cc-fees`, P06 `service-authority` `partial`, P01/P02/P03/P07.
- ✔ Fee lines for the 15 packages.
- ✔ Construction 01–12.
- ✔ 13.1, 13.3, 13.4 and 13.9.
- ✔ Demolition `partial`.
- ✔ Split `commercial_hotel` (`residential_retail` listed; no fire-passive share).
- ✔ HPC, long service levy, s138 and strata lines `no` (not in `starter_lines.yaml`).
- ✘ 13.2 accessible facilities `no` (it should be `partial`; finding 13).

**Program**

- ✔ Skeleton `sk.new-dc-da` (new, D&C, da).
- ✔ The process fragments `-2`, `fr.da-*`, `fr.contract-award`, `fr.novation` (route design_construct),
  `fr.practical-completion`, `fr.occupation-certificate`, `fr.handover`, `fr.defects-*`.
- ✔ Construction fragments: `fr.basement-2` (all four activity labels match), `fr.footings-2`,
  `fr.lift-shafts-2`, `fr.structure-concrete-2`, `fr.facade-2`, `fr.windows-2`, `fr.balcony-waterproof-2`,
  `fr.ac-2`, `fr.carpark-vent-2`, `fr.linings-2`, `fr.ceilings-2`, `fr.wet-area-2`, `fr.tiling-2`,
  `fr.joinery-2`, `fr.internal-doors-2`, `fr.painting-2`, `fr.floors-2`, `fr.landscape-2`, `fr.stormwater-2`,
  `fr.earthworks-2`, `fr.lifts`, `fr.utility-major` (class 2), `fr.services-connections-major`,
  `fr.switchboard-2`, `fr.switchboard-major` duplicate (class 6), `fr.erosion-control`, `fr.fire-stopping`.
- ✔ Hold point `if.fire-stopping-before-concealment` `yes`.
- ✔ `partial`: `fr.investigate-survey` and `fr.investigate-services` (need `investigate`; shape gives `new`),
  `fr.investigate-geotech`, `fr.investigate-contamination`, `fr.temporary-power`, `fr.demolition-2` (needs
  `remove`), `fr.sprinklers-2`, `fr.construction-certificate` (stage), the sprinkler hold point.
- ✘ Reinforcement hold point: the `partial` label is right, but its consequence and order are wrong (finding 5).
- ✔ `no` spot checks: no fragment for `structure.steel` in class 2 or 6 (only `fr.steel-7b` and
  `fr.steel-9b`), `site.loading-docks`, `site.waste-storage`, `site.public-domain`, `hydraulic.trade-waste`,
  `fire-active.occupant-warning`; `fr.existing-conditions` needs refurb or extend.

**RFPs**

- ✔ Town planning `yes`: every stage and position checked against `reference/index/rfp-templates.md`; with
  `flag.panel` yes, `panel_attendance` is I and scale complex; with `cl46` no, `variation_request` is excluded;
  `submissions_response` is `if_unknown` option, so O.
- ✔ Architecture `partial`:
  - boundary `brief_masterplan … dnc_tender` under D&C;
  - `interiors` included by class 6;
  - `regulated_design` and `post_novation` are after the boundary, so options;
  - `for_construction` and `construction_support` excluded;
  - `apartment_design` O, `tenant_fitout` O, `tender_evaluation` complex O;
  - `fact:budget` blocking.
- ✔ Structural `partial`: `construction_phase` after the boundary is an option although scale is complex
  (`resolveStages`, the default branch).
- ✔ BCA and Certifier `partial`: boundary `a1…a3` unless the client flags are set; unknown stages after the
  end become options (`resolveStages`, the unknown branch).
- ✔ The other 13 RFPs were checked for template ids and labelled `partial`, consistent with the client-flag
  rules.

**Triggers**

- ✔ heritage procurement `yes` (`complexity_additions[1]`); ✘ heritage cost_plan `yes` (should be
  `partial`; finding 13).
- ✔ cl46 rfps `yes`.
- ✔ flood rfps `yes`.
- ✔ contaminated-land rfps `yes` (civil `earthworks_suitability`) and procurement `no`.
- ✔ traditional-lump-sum program and rfps `yes` (`sk.new-traditional-da`; `fr.novation` drops).
- ✔ public-domain rfps `yes` (civil `public_road`).
- ✔ effective-height procurement, work_items and program `no` (no generator reads `det.effective_height`).

**Rule and interface anchors:** all 18 cited ids exist in the files named, for example
`rule.ncc.type-of-construction` (C2D2, clause_verified true; it notes C2D4 for mixed classes),
`rule.nsw.section-73-certificate` and `cq.pre-2004-fabric-hazardous-materials-survey`.

---

# Round 1 response (author, 30 October 2026)

All 19 findings applied. Nothing disputed. Two needed a choice, and one item is outside this folder.

| # | Response |
|---|---|
| 1 | Applied. The `fire-control` item is gone from the 25 m trigger. The cost line is now "Emergency warning and intercom system (EWIS)". DOMAIN §2 gives 25 m for smoke control and 50 m for the fire control centre, both `verified: false`. The rooftop-room trap is in §10. |
| 2 | Applied. The Act is now Strata Schemes Management Act 2015 Part 11, in DOMAIN §5 and §12 and in the shape's `why`. The HBCF sentence now reads "rise in storeys of more than 3 containing 2 or more dwellings". |
| 3 | Applied. Every package has `novation`: architect, structural, civil and landscape `yes`; the four services engineers `option`; all the rest `no`. DOMAIN §4 is softened to match. Note that T02 gives the services engineers `yes`, so the orchestrator should settle this for both. |
| 4 | Applied on the `option` reading. For hydraulic, fire services, mechanical and electrical, `continuation`, `construction` and `handover` are now `stages_option`, their positions are `O`, and those `app_differs` rows are gone. **OWNER_RULINGS is outside my folder**, so the orchestrator must record the ruling there. If the owner rules `yes`, revert this. |
| 5 | Applied. The hold point now comes after "Excavate and pour footings" and "Form lift pits". "Pour the basement slab" and the new `fr.slab-2` row (marked as a duplicate) both come after it. The hold is `built_in: yes`, and its `why` names the app's wrong links. A gap row is added. |
| 6 | Applied. A new row, "Fit off and commission sprinklers", comes after "Install ceilings". The pressure-test `why` names the inversion. A gap row is added. |
| 7 | Applied. Both rows are now `construction_documentation`. CCs are also staged (see the coordinator item below). |
| 8 | Applied. The DNSP supply application and the services and Section 73 applications now come after "Local planning panel determination". A gap row records the app's stage-start anchor. |
| 9 | Applied. New triggers: `rise-three-storeys` (adds HBCF; removes the bond and the Class 2 sprinkler rows), `trees-affected`, `acid-sulfate-soils`, `underpinning-neighbour`, and `regionally-significant` (zero-delta, because `flag.panel` is already yes). `rise-three-storeys` is added to `variants`. The affordable-housing bonus is not added; it is optional. |
| 10 | Applied. The base `site.public-domain` text no longer uses "road reserve". A `flag.public_road` scenario_input with a denial is added. |
| 11 | Applied. All three traps are in §10. Negation scenario_inputs are added for `flag.groundwater`, `flag.heavy_vehicles` and `flag.public_road`. The §2 text "dewatering of seepage" now reads "seepage drainage". |
| 12 | T10's approach chosen. `det.existing_building: true` and `det.existing_building_year: 1965`, with scenario_inputs; the phrasing uses the singular, because `\bexisting building\b` does not match "existing buildings". The architect `existing_interfaces` differ is removed (the row is now I). The structural `existing_assessment` (app include) and `strengthening` (app O) are expected absent, recorded as differs. The access `existing` row (app I) is expected absent, recorded as a differ. Civil `retained_services` is now expected I. |
| 13 | Applied. Accessible facilities is `partial` (elements.yaml code 13). Heritage `cost_plan` is `partial`. |
| 14 | Applied. The composite rough-in row is split into sanitary, cold water, hot-water plant, electrical, hydrants and detection rows. New rows: `fr.gas-commercial`, `fr.insulation-2`, `fr.roof-drainage-2`, `fr.lighting-2`, `fr.comms-2` and `fr.smoke-alarms-2`, plus a services fit-off row. The `fr.switchboard-major` duplicate is now its own expected row. |
| 15 | Applied. The WSC is in the hydraulic `must_include`, has a gap row, and has a row in the DOMAIN §4 table (no id). |
| 16 | Applied as a gap row (`area: harness`). **For the orchestrator:** the fix belongs in `score_shapes.py score_rfps`. |
| 17 | Applied. The Class 2–4 list is now commencement, the 10% wet-area sample, stormwater connections and the final, all `verified: false`. |
| 18 | Applied. `det.effective_height: 18.9` is in `profile`, with a scenario_input. It is removed from `expected_derived`. |
| 19 | Applied. The ESD `why` and DOMAIN §4 name `consultant.basix` as an accepted alias. The orchestrator should settle one id. |

**Coordinator item (T02 keyword phrases).** I tested each phrase against the flag, determinant and implied_by regexes. These fire: "four buildings" (`multi_building`), "Stage 1" (`staged`), "water main" (`authority_asset`, `water_supply_source`), "traffic noise" (`noise_affected_site`), "street trees" (`arborist_offsite_trees`), "MRV/HRV" (`heavy_vehicles`), "road reserve" (`public_road`), "basement storage cages" (`basement`), "kitchen", "party wall" and "sewer junction". These fire nothing: "existing buildings", "existing shops", "retail tenancy", "loading bay/dock", "grease arrestor" and "shop-top housing". All of them are now a table in DOMAIN §10.

**Staged CCs.** The program now has "Construction certificate: early works (retention, excavation, basement)", then the main CC (`fr.construction-certificate`), which comes after the regulated designs. Demolition runs under the DA consent with no CC (`verified: false`). Site establishment follows award. The principal certifier and notice of commencement come before retention, after the early-works CC. A gap row is added.

---

## Round 2 (30 October 2026)

Read against the revised `DOMAIN.md` and `shape.yaml`, and against `reference/CONVENTIONS.md` C1–C3. Nothing
was edited except this section. The validator, run from the repo root, gives `ok` (exit 0). Counts: 18
triggers, 25 packages, 55 cost lines, 113 activities, 23 RFPs.

### Verdict: **FIX** (small)

All 19 round 1 findings were applied, and every one checks out except the leftovers below. The new staged-CC
rows and the "no CC for demolition" logic are right for NSW: a CC covers building work, and demolition
authorised by the DA consent needs none (verified: false is kept). Five small changes remain. The first
would make the scorer report a false failure, so the key is not yet usable as is.

### Round 1 findings, rechecked

| # | Status | Note |
|---|---|---|
| 1 | ✔ mostly | `fire-active.fire-control` removed, EWIS line renamed, 50 m stated, rooftop-room trap added. **Left over:** `DOMAIN.md` §9 table, `effective-height-over-25m` row, still lists "FCC" under Adds (R2-3). |
| 2 | ✔ | Strata Schemes Management Act 2015 Part 11 in §5 and in the program `why`; HBCF wording is rise > 3 with 2+ dwellings. |
| 3 | ✔ | `novation` on every package; matches C1 (architect, structural `yes`; four services `option`; rest `no`). Civil and landscape are `yes`, as in T02. C1 names neither (R2-5). |
| 4 | ✔ | Services RFPs: continuation, construction and handover are now options, with the matching positions `O`; `must_include` reworded as options. |
| 5 | ✔ | `fr.slab-2` added; the hold row sits after footings and lift pits, both slabs after it; gap rows added. |
| 6 | ✔ | "Fit off and commission sprinklers" after "Install ceilings"; commissioning row now follows it. |
| 7 | ✔ | CC, early-works CC and LSL in `construction_documentation`; PC appointment follows the early-works CC. |
| 8 | ✔ | DNSP application and services applications follow the panel determination. |
| 9 | ✔ | `rise-three-storeys`, `trees-affected`, `acid-sulfate-soils`, `underpinning-neighbour`, `regionally-significant` added. Fragments cited exist (`fr.underpinning-2` needs class 2 and new; `fr.investigate-acid-sulfate`). `trees-affected` follows the owner-approved T01 form. One gap in `rise-three-storeys` (R2-4). |
| 10 | ✔ | Base public-domain scope narrowed; no public road works in the base. |
| 11 | ✔ | Traps added in §10 for `road reserve`, MRV/HRV and `dewater`/`groundwater`. |
| 12 | ✔ mostly | Followed T10: `det.existing_building: true`, year 1965, scenario phrasing that respects the singular-only trigger. Architect `existing_interfaces` differ dropped; structural `existing_assessment` and access `existing` differs added. **Missed:** civil (R2-1). Gap text stale (R2-3). |
| 13 | ✔ | 13.2 `partial` (elements.yaml code 13); heritage `cost_plan: partial`. |
| 14 | ✔ | Composite rows split; `fr.gas-commercial`, `-insulation-2`, `-lighting-2`, `-comms-2`, `-roof-drainage-2`, `-smoke-alarms-2`, `-hydrants-2`, `-detection-2`, `-electrical-2`, `-hydraulic-cold-2`, `-hydraulic-hot-2` and the `fr.switchboard-major` duplicate added. |
| 15 | ✔ | WSC in DOMAIN §4, in the hydraulic `must_include`, and as a `gaps` row. |
| 16 | ✔ | Harness gap recorded; C3 settles it (discipline first). |
| 17 | ✔ | Class 2–4 list with the 10% wet-area sample; "fire-safety stages" dropped. |
| 18 | ✔ | `det.effective_height: 18.9` in `profile`, with a scenario_input. |
| 19 | ✔ partly | `consultant.esd` kept, which is right under C2 (one firm does BASIX and Section J). See R2-2. |

### Round 2 findings

1. **`shape.yaml rfps[consultant.civil]`: missing `app_differs` for `retained_services`.** With
   `det.existing_building: true`, the `existing_building` module is on, and civil `retained_services` is
   "I, always [module existing_building]" (`reference/index/rfp-templates.md`, civil positions). The shape
   expects `retained_services: absent` with no differ, so the scorer will report a new failure.
   **Change:** add `{key: position.retained_services, app: I, expect: absent, why: "existing_building module
   on, but every existing service is abolished before demolition"}`. If the owner prefers the CCTV and
   potholing brief for old lead-ins, use `expect: O` instead.

2. **`shape.yaml procurement[consultant.esd]`: C2 says to add `consultant.basix` to `aliases`.** The alias
   is only in `why`. **Change:**
   - add `aliases: ["consultant.basix"]`;
   - add a `gaps` row (`area: harness`): `score_procurement` matches on the discipline id only and ignores
     aliases (`tools/score_shapes.py` `same` rule), so an app package created as BASIX scores as a miss.

3. **Stale text.**
   - `DOMAIN.md` §9 table, `effective-height-over-25m` row: drop "FCC" from Adds.
   - `shape.yaml gaps` "ADG design verification is an option; demolition drawings absent because the
     existing_building module is off on a new build": now false. Reword to "ADG design verification is an
     option".

4. **`shape.yaml triggers[rise-three-storeys]`: the variant cannot remove the sprinkler scope it removes
   from the program.** It removes the sprinkler activities, but `fire-active.sprinklers` stays in the base
   `work_items` and `det.sprinklered: true` stays in `profile`. **Change:**
   - add `det.sprinklered` to `profile`;
   - say in `why` that the scenario sets `det.sprinklered: false` and leaves the `fire-active.sprinklers`
     scope row out;
   - add `rfps: [{discipline: consultant.fire_services, must_include: ["Hydrants and detection only; no
     sprinkler design"]}]` so the fire services RFP is also tested.

5. **`shape.yaml program` "Bulk earthworks" (`fr.earthworks-2`): the logic is wrong.** It follows only
   "Demolish existing structures and clear". On a boundary-to-boundary basement, bulk excavation follows the
   piled retention, and it is building work under the early-works CC. **Change** `after` to
   `["Install retention shoring", "Appoint principal certifier and give notice of commencement"]`. In `why`,
   say "bulk excavation within the retention".

   **Conventions (no change to the map):** C1 should name the civil engineer and the landscape architect,
   both `yes` in T02 and T13, so later D&C maps agree.

### Claims rechecked in round 2

- ✔ The new fragment ids exist with the predicates claimed: `fr.slab-2`, `fr.gas-commercial` (2, 5, 6, 9b),
  `fr.underpinning-2` (class 2; new, alter, repair), `fr.investigate-acid-sulfate`.
- ✔ `flag.arborist_offsite_trees` is a flag key.
- ✔ The `det.existing_building` phrasing matches the singular trigger note.
- ✔ The novation values agree with C1.
- ✔ The services RFP `stages_option` lists agree with C1.
- ✔ The C3 pairing gap is recorded.
- ✘ The civil `retained_services` expectation is wrong for the new profile (R2-1).

# Round 2 response (author, 30 October 2026)

All five items are applied.

1. Civil `retained_services`: a domain decision. Every existing service is abolished before demolition and the new connections are designed from scratch, so I keep the row **absent**. An `app_differs` entry records that the app gives I. This replaces the round 1 change, which set I.
2. `consultant.esd` now has `aliases: ["consultant.basix"]`. A harness gap records that `score_procurement` ignores aliases.
3. "FCC" is removed from the DOMAIN §9 row. The ADG gap now reads "ADG design verification is an option".
4. `rise-three-storeys` now lists `det.sprinklered` in its profile keys. Its `why` says the scenario sets `det.sprinklered: false` and leaves out the sprinkler scope row. The trigger also adds a fire services RFP check: "Hydrants and detection only; no sprinkler design".
5. "Bulk earthworks" now comes after "Install retention shoring" and the notice of commencement under the early-works CC. Bulk excavation happens within the retention.
