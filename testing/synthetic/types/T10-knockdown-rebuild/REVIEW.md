# T10 Knock-down rebuild house: independent review

Reviewer brief: `testing/synthetic/briefs/02-review.md`. Read 30 October 2026 against the working tree.
Only this file was written. No Jev, no network, no state-changing git command.

## Verdict: **FIX**

The map is sound at the root. The base job, the demolition story, the consultant list, the CDC sequence
and the gap list are right for a Sydney knock-down rebuild, and the owner rulings are followed except R7
item 1. It is not yet a reliable answer key: one program row breaks R7, four `built_in` claims are wrong
against the code, the program leaves out rows the app really produces, and two triggers are built
badly. Each finding below gives the exact change.

## 1. Validator

Run from the repo root (it resolves `knowledge/` relative to the cwd; from `testing/synthetic` it
crashes with `FileNotFoundError: knowledge\profile\taxonomy.yaml`):

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T10-knockdown-rebuild/shape.yaml`
→ `ok`, exit 0. No warnings.

## 2. Rulings on the author's open questions

These are the reviewer's rulings. The owner can override any of them.

1. **CDC with traditional procurement as the base: accepted.** A code-compliant knock-down rebuild in
   Greater Sydney usually goes CDC. Keeping T01's consultant-led traditional route means T01 (DA) and
   T10 (CDC) differ only in the pathway and the existing house, and the harness gets cover for
   `sk.new-traditional-cdc`. The project-home package is the most common job by volume, but the app
   models it as novation and falls back to `sk.generic`. It stays the `builder-designs` trigger.
   **Change:** in DOMAIN §1, replace "**The owner should confirm this choice of base.**" with "Reviewer
   accepted this base (T10 REVIEW §2); the owner may override." In §12, drop the first "least sure"
   bullet or mark it "accepted at review".
2. **One CDC for demolition and the new dwelling: accepted.** Certifiers routinely issue one CDC
   described as "demolition of the existing structures and construction of a new two-storey dwelling".
   The Demolition Code and Housing Code part numbers, and any demolition conditions, stay
   `verified: false`. **Change:** §12 bullet 2 → "accepted at review; part numbers `verified: false`".
3. **Who removes the old electricity service: do not split by DNSP.** Nothing in the app, or in this
   key, changes with the DNSP, so the key should not depend on it. Use one neutral sequence for Ausgrid,
   Endeavour and Essential:
   - the owner, as account holder, asks the retailer to close the account; the retailer's metering
     provider removes the meter;
   - a Level 2 accredited service provider, engaged by the owner or the builder, disconnects and removes
     the service line, and the DNSP is notified;
   - temporary builders power is arranged through the ASP, often from the same point of attachment.

   Keep the whole sequence `verified: false`. **Change:** DOMAIN §3 "Electricity" bullet: replace "A
   Level 2 accredited service provider removes the overhead service and the meter" with the three
   points above. In `shape.yaml`, relabel "Request electricity abolishment and gas meter removal" to
   "Request meter removal and service abolishment (retailer, Level 2 ASP, gas network)". Keep the why
   text and add "(verified: false)".
4. **`det.existing_building: true` for a knock-down: accepted.** "Work to an existing building" is
   literally true: demolition, the WHS asbestos duty, the existing services and drainage, and the
   neighbours' dilapidation all attach to the existing house. True also gives the two RFP rows the job
   needs: architect `existing_interfaces` and civil `retained_services`. The cost is the structural
   `existing_assessment` stage, with `strengthening` priced as an option, and some noise in the
   Implications. Both are already recorded in `app_differs` and `gaps`, which is the right place for
   them. With false, or unset (work type `new` then derives `flag.existing_building = no` in
   `internal/rfp/drivers.go deriveFlags`), the job would lose two rows it really needs in order to drop
   one it doesn't. **Change:** §12 last bullet → "accepted at review (T10 REVIEW §2.4)".

## 3. Findings

### F1. The pre-start inspection is placed after the CDC, which breaks R7 item 1. Severity: high

- **Where:** `shape.yaml program.activities` "Pre-commencement inspection" (stage `construction`, after
  "Appoint principal certifier…"), "Certifier determination" (its `after`), `rfps[consultant.certifier]
  must_include` "Pre-commencement inspection before demolition"; `procurement[consultant.certifier].why`;
  DOMAIN §5 "Inspections and hold points" and §7 step 6.
- **What is wrong:** R7 item 1 says the pre-start inspection "comes before the construction certificate
  or the start of works". On a CDC job the CDC does the CC's work, so the certifier inspects the site
  (the existing house still standing) before determining the CDC. The map puts the inspection after
  award, HBCF and notice of commencement, which is after the CDC has issued.
- **Exact change:**
  1. Move the row to `stage: approvals`. Label it "Pre-start site inspection by the certifier (R7 item 1)"
     with `after: ["Prepare and lodge the complying development certificate"]`, `built_in: "no"`, and
     why "Before the CDC is determined; the clause stays verified: false".
  2. Add it to "Certifier determination" `after`.
  3. "Establish the site" `after`: replace "Pre-commencement inspection" with "Appoint principal certifier
     and give notice of commencement". Do the same in "SafeWork NSW asbestos removal notification" `after`.
  4. Certifier RFP `must_include`: replace "Pre-commencement inspection before demolition" with "Pre-start
     site inspection before determining the CDC (R7 item 1)".
  5. DOMAIN §5: replace "On a knock-down rebuild the **pre-commencement inspection comes before
     demolition**, because demolition is the start of work" with "the pre-start inspection (R7 item 1)
     comes before the CDC is determined, so the certifier sees the existing house; demolition then waits
     for the notice of commencement". Move it from §7 step 6 to step 4.

### F2. The reinforcement hold point is produced today; `partial` is wrong. Severity: medium

- **Where:** `shape.yaml program.activities` "Reinforcement inspection before slab pour",
  `built_in: "partial"`, why "Hold point needs a structure.concrete item".
- **Evidence:** `gen.holdPointRows` (`internal/program/generate.go`) fires when rows are *tagged* with
  `hold_point.after` and `.before`. It does not need a work item. `fr.slab-on-ground-1a` activity `pour`
  carries `systems: [substructure.slab-on-ground, structure.concrete]`, so `tagged("structure.concrete")`
  and `tagged("substructure.slab-on-ground")` are both non-empty and `if.reinforcement-inspected-before-pour`
  fires on every new house with a slab. The T01 review accepted the same claim (T01 REVIEW line 275),
  but the code reads otherwise.
- **Exact change:** `built_in: "yes"`, `source: "knowledge/clusters/structure/interfaces.yaml
  if.reinforcement-inspected-before-pour"`, why "Fires from the structure.concrete tag on fr.slab-on-ground-1a
  pour; app label 'Hold point: Reinforcement inspection release identifying the pour.'". The pour row is
  both the `after` and a `before` row, so the app may place the hold point oddly. Confirm with one §7.1
  Go run, and carry the same fix to T01.

### F3. The hazmat survey row cites a fragment that cannot fire. Severity: medium

- **Where:** `program.activities` "Demolition hazardous materials survey", `fragment: fr.existing-conditions`,
  `built_in: "partial"`.
- **Evidence:** `fr.existing-conditions applies_when {det: work_type, any_of: [refurb, extend]}`. This
  profile is `new`, so the fragment is false whatever the scenario says. It is the brief's own example
  of a default that "only fires for its work types". The map's own gap (`gaps[4]`) says so.
- **Exact change:** `fragment: null`, `built_in: "no"`, drop `source`, why "A hazmat activity exists only
  in fr.existing-conditions, gated to refurb or extend (gap)".

### F4. The program leaves out rows the app produces, and set-out is in the wrong order. Severity: medium

- **Where:** `program.activities` (footings, slab, roof, linings, defects); "Building set-out survey".
- **Evidence:** `knowledge/program/fragments.yaml` activities that fire for this profile but have no
  matching row: `fr.footings-1a` "Pour footings", "Cure footings"; `fr.slab-on-ground-1a` "Set out and
  excavate slab", "Cure slab"; `fr.roof-1a` "Install sarking"; `fr.linings-1a` "Frame internal
  partitions"; `fr.defects-liability-end` "Final defects inspection". Without them the app's correct rows
  score as extras. "Building set-out survey" is marked `no`, but the app's "Set out and excavate slab"
  covers part of it. A surveyor's set-out on a cleared lot also follows the bulk cut, which destroys
  earlier pegs.
- **Exact change:**
  1. Add the seven rows with their fragments, `built_in: "yes"`, and `after` as in the fragment links:
     pour after "Excavate and inspect footings"; cure after pour; "Set out and excavate slab" after
     "Cure footings"; "Under-slab sanitary drainage" after "Set out and excavate slab"; "Cure slab" after
     "Reinforce and pour slab"; sarking before "Install roof cover"; partitions before "Sheet and set
     plasterboard"; "Final defects inspection" closes the defects stage.
  2. "Building set-out survey": `built_in: "partial"`, `source: "knowledge/program/fragments.yaml
     fr.slab-on-ground-1a"`, `after: ["Cut, fill and trim levels"]`. Change "Cut, fill and trim levels"
     `after` to `["Geotechnical check of the cleared platform", "Isolate, demolish and clear"]`, and make
     "Excavate and inspect footings" follow "Building set-out survey".

### F5. The `separate-demolition-contract` trigger claims app content it does not have, and over-adds. Severity: medium

- **Where:** `triggers[separate-demolition-contract]` `built_in.program: "partial"`, `source:
  fr.demolition-1a`; `adds.program` "Separate demolition approval"; `adds.authority`.
- **What is wrong:**
  - None of the three added rows comes from `fr.demolition-1a`, which already fires in the base, so the
    trigger adds nothing the app produces.
  - A separately engaged demolisher can work under the same CDC (ruling §2.2), so a separate approval is
    needed only for a demolition-only CDC.
- **Exact change:**
  - `built_in: {program: "no", cost_plan: "no"}` and drop `source`.
  - Make "Separate demolition approval" conditional: label it "Demolition-only CDC, only where demolition
    is approved apart from the dwelling (verified: false)", or move it into `why`.
  - Add to `adds.cost_plan` "Provisional sum for latent conditions and asbestos variations (owner-held)",
    because with a separate demolisher that risk sits with the owner.

### F6. `loose-fill-asbestos` mixes two different conditions. Severity: medium

- **Where:** `triggers[loose-fill-asbestos].evidence[2]` "friable asbestos pipe lagging under the floor",
  `negation[1]`, and `adds`.
- **What is wrong:** Friable lagging found by the hazmat survey means Class A removal by a licensed
  removalist with air monitoring and clearance by a licensed asbestos assessor. It does not put the house
  on the LFAI register. It does not need remediation of the footprint soil or the government
  loose-fill process. A scenario written from evidence 3 would be scored against consequences it should
  not carry.
- **Exact change:** Remove the lagging evidence and its negation from `loose-fill-asbestos`. Add:
  ```yaml
  - id: friable-asbestos
    label: Friable asbestos found by the demolition survey
    evidence: ["Friable asbestos pipe lagging was identified in the subfloor", "the hazardous materials survey found friable asbestos in the old heater flue"]
    negation: ["No friable asbestos was identified; all asbestos-containing materials are bonded"]
    profile: []
    adds:
      cost_plan: ["Class A friable asbestos removal with enclosure and air monitoring"]
      program: ["Friable asbestos removal under enclosure", "Licensed asbestos assessor clearance"]
      rfps: [{discipline: consultant.hazmat, must_include: ["Air monitoring during friable removal", "Licensed asbestos assessor clearance"]}]
      authority: ["SafeWork NSW Class A licensed removal notification (verified: false)"]
    built_in: {cost_plan: "no", program: "no", rfps: "no"}
  ```
  Keep `consultant.environmental` and the soil remediation on `loose-fill-asbestos` only. DOMAIN §9:
  split the row the same way.

### F7. Structural `planning_approval` on a CDC should be included, as the app does. Severity: low

- **Where:** `rfps[consultant.structural].expect.stages_option` includes `planning_approval`; `app_differs`
  row `stage.planning_approval`.
- **What is wrong:** T01's reason ("a house DA rarely needs structural input") does not carry over to
  a CDC. The certifier assesses the engineering at lodgement and sends structural requests during
  assessment. The stage deliverable ("responses to authority requests during assessment",
  `knowledge/rfp/structural.yaml planning_approval`) is real work on this job.
- **Exact change:** Move `planning_approval` to `stages_include` and delete that `app_differs` row.

### F8. Civil `as_built` is included while `construction_phase` is an option. Severity: low

- **Where:** `rfps[consultant.civil].expect` (`stages_include: as_built`, `stages_option:
  [construction_phase]`).
- **What is wrong:** `as_built` has `follows: construction_phase` (`knowledge/rfp/civil.yaml`). An
  engineer's certificate for the OC with no inspections is not something a certifier accepts. Either
  both stages are in or both are out.
- **Exact change:** Move `construction_phase` to `stages_include`, set `stages_option: []`, and add
  `app_differs {key: stage.construction_phase, app: option, expect: include, why: "stormwater inspection before
  covering (R7 item 6) supports the engineer's certificate for the OC"}`.

### F9. The balustrades cost claim is overstated. Severity: low

- **Where:** `cost_plan` "Compliant balustrades and barriers", `built_in: "no"`, why "never enters house
  scope".
- **Evidence:** `knowledge/costs/elements.yaml` code 13 is itemised and includes `access-egress.barriers`
  (T11 predicts "13.3 Compliant balustrades and barriers"). The shape's own `work_items` include
  `access-egress.barriers new`. The line appears whenever a document presence row or the user adds the
  item. It is only absent from the default list and from `implied_by`.
- **Exact change:** `built_in: "partial"`, `source: "knowledge/costs/elements.yaml code 13"`, why "Only
  when a document or the user adds access-egress.barriers; no implied_by pattern".

### F10. A common knock-down trigger is missing. Severity: low

- **Where:** `triggers`; DOMAIN §9.
- **What is missing:** Fibro (asbestos cement) dividing fences are common on 1960s Sydney lots. They need
  the neighbours' agreement on removal and replacement, licensed removal along the boundary, and
  cost-sharing. This changes the hazmat scope, the program and the cost plan.
- **Exact change:** Add:
  ```yaml
  - id: asbestos-dividing-fence
    label: Asbestos cement dividing fence to be removed
    evidence: ["The side and rear dividing fences are fibro (asbestos cement) sheeting", "replace the existing fibro boundary fences with Colorbond by agreement with the neighbours"]
    negation: ["Dividing fences are timber paling and are retained"]
    profile: []
    adds:
      cost_plan: ["Asbestos fence removal and replacement fencing (owner share)"]
      program: ["Fencing notices and agreement with neighbours", "Licensed removal of asbestos fencing"]
      rfps: [{discipline: consultant.hazmat, must_include: ["Dividing fences in the survey and removal scope"]}]
      authority: ["Dividing Fences Act 1991 notice (verified: false)"]
    built_in: {cost_plan: "no", program: "no", rfps: "no"}
  ```
  Existing boundary retaining walls on a sloping lot are also common. Name them under "less common
  variants" at least.

### F11. "Brick veneer" will not add `structure.masonry`. Severity: low

- **Where:** `scenario_inputs`; DOMAIN §10; `program` "Lay brick veneer".
- **Evidence:** `implied_by masonry` pattern is `\bmasonry\b|\bre-?point(?:ing)?\b|\bbrickwork\b`
  (`knowledge/profile/scope_defaults.yaml`). "Brick veneer", which is how house specifications say it, is
  missed.
- **Exact change:** Add a scenario input `{key: scope.structure.masonry, note: "'brick veneer' is not matched;
  implied_by needs 'brickwork' or 'masonry'", phrasing: "External walls: face brickwork veneer to the ground
  floor"}`. Add trap 19 to DOMAIN §10 with the same text.

### F12. Rows that cite a fragment use labels the app does not print. Severity: low

- **Where:** `program.activities` "Detail and boundary survey including the existing house" (app:
  "Detail and boundary survey"), "Services search and locating" (matches). The hazmat row is covered by F3.
- **Exact change:** Where `fragment` is set, use the fragment's activity label exactly and put the
  knock-down detail in `why`. Otherwise the scorer cannot match the row.

### F13. The structural RFP's `must_include` contradicts its own option. Severity: low

- **Where:** DOMAIN §8 Structural lists an option "advice on a neighbour's boundary or retaining wall that the
  old structure supports". `shape.yaml` structural `must_not_include` "Existing structure assessment".
- **Exact change:** Keep the exclusion, and add to `rfps[consultant.structural].must_include` "Option:
  advice on boundary walls or neighbours' structures affected by demolition". The two then agree.

## 4. Completeness and domain checks with no change needed

- All 13 top-level systems are in or out with a reason (checked against `knowledge/systems.yaml`).
- Consultants are right. Hazmat, BASIX and dilapidation are in; town planner and services engineers are
  correctly not expected. Civil is in the base, as R1 requires: the roof and hard surfaces change.
- Owner rulings: R1 ✔, R3 ✔ (`proj.regulated_design: no`), R4 ✔, R5 ✔ (new dwelling, not
  alterations), R6 ✔, R8 ✔; R7 items 2–6 and 8 ✔, item 1 ✘ (F1).
- NSW approvals: Sydney Water Tap in and the long service levy before CDC determination, CDC
  neighbour notice, s138 for the crossing, SafeWork notice for licensed removal, and Section 73 not
  expected. All are plausible and all carry `verified: false`.
- No prices, rates or week durations appear. Rule and clause numbers are all marked `verified: false`.
- Trigger evidence reads like real documents and gives indirect forms. The negations are realistic.

## 5. Built-in claims checked

**Procurement:** architect `baselines[0]` ✔ · surveyor ✔ · geotechnical ✔ · structural ✔ · civil ✔ ·
certifier ✔ · `trade.main_works` partial (`ProposeHeadContract` gaps on accepted items) ✔ ·
not-expected town planner `baselines[0]` ✔ · electrical ✔ · hydraulic (also `scope_additions[3]`) ✔ ·
mechanical `scope_additions[2]` ✔.
Spot-checked `no`: hazmat ✔ · BASIX ✔ · building consultant ✔ · quantity surveyor ✔ · arborist ✔ (no
package rule in `knowledge/works/package_defaults.yaml` names any of them).

**Cost plan:** CDC fee partial (`cc-fees` applies always) ✔ · service-authority partial ✔ · fee lines for
the adopted packages ✔ · demolition partial (rolls into code 01; `residential_house_new` has no demolition
share) ✔ · 01–05, 08–11 ✔ · stairs code 13 partial ✔ · design and construction contingency, escalation,
exclusions ✔ · balustrades `no` ✘ (F9).
Spot-checked `no`: contributions (`da-fees` only for `da`) ✔ · long service levy ✔ · preliminaries ✔ ·
HBCF ✔ · road-reserve fees ✔.

**Program:** `sk.new-traditional-cdc` ✔ · `fr.investigate-services`, `fr.investigate-survey`,
`fr.investigate-geotech` partial (need `investigate`) ✔ · `fr.existing-conditions` partial ✘ (F3) ·
concept, design development, approval and construction documentation (placement by `resolveStage`
nearest-earlier, read) ✔ · `fr.cdc-application` ✔ · tender, evaluation, award ✔ · dilapidation partial
(`investigationRows`) ✔ · site establishment ✔ · `fr.erosion-control` ✔ · `fr.tree-protection` partial
(`retain` brings nothing) ✔ · `fr.temporary-power` partial ✔ · `fr.demolition-1a` partial (`remove`
only) ✔ · `fr.services-connections-1a` (new, alter, upgrade) ✔ · `fr.utility-1a` ✔ · `fr.earthworks-1a`
attach after `fr.demolition-1a` ✔ · footings, slab, termite ✔ · reinforcement hold point partial ✘ (F2) ·
frame timber and `if.frame-inspected-before-roof` ✔ · roof, roof drainage, windows ✔ · `fr.masonry-1a`
partial ✔ · external walls, hydraulics, AC, insulation, linings, ceilings, doors, wet area, tiling, joinery,
electrical, painting, floors, stormwater, landscape ✔ · PC, OC, handover, defects ✔.
Spot-checked `no`: LFAI search ✔ · rainwater tank (no `non-potable` fragment) ✔ · internal stair (only
`-2` and `-9b` stair fragments) ✔ · driveway and crossing (no `vehicle-access` or `public-domain`
fragment) ✔ · set-out survey ✘ (F4: `fr.slab-on-ground-1a` "Set out and excavate slab").

**RFPs:** architecture `existing_interfaces` on module `existing_building`; `required_facts: [budget]`;
bidding after the house boundary; `lodgement` on `architect_lodges` ✔ · structural `existing_assessment`
on `flag.existing_building` (from `det.existing_building`, `drivers.go readFlags`); `strengthening` O;
`dilapidation` X ✔ · civil `retained_services` module, `water_quality` O, `pavement_design` O, `as_built`
follows `construction_phase` ✔ · bca-certification blocks on `cdc`, B stages need `da`/`ssd` ✔ ·
geotechnical, land-surveying, cost-planning client-flag rows (via the index) ✔.

**Triggers:** `da-pathway` program yes, cost yes, procurement partial ✔ · heritage procurement
`complexity_additions[1]` ✔ · `trees-affected` rfps yes, program partial ✔ · `osd-required` rfps
`osd_new` ✔ · `sewer-main-in-lot` rfps ✔ · `excavation-adjoining` rfps ✔ · `bushfire-prone`
procurement via `det.bal` ✔ · `flood-affected` rfps ✔ · `asbestos-in-soil` rfps `earthworks_suitability` ✔ ·
`loose-fill-asbestos` program partial (`fr.investigate-contamination`) ✔ ·
`separate-demolition-contract` program partial ✘ (F5).

**Gaps spot-checked:** `implied_by strip-out` gives `new` ✔ · `existing_building` trigger is only
`\bexisting building\b` ✔ · year triggers ✔ · `uc.asbestos-found-in-existing-building` excludes `remove` ✔ ·
`cq.pre-2004-fabric-hazardous-materials-survey` fires on `remove` plus a year before 2004 ✔ · no package
rule reads `site.demolition` ✔.

## Round 1 response (author, 30 October 2026)

Every finding was applied except F11, which was applied in a different form (see below). Rulings §2.1–2.4 were applied to DOMAIN §1, §3 and §12. The validator returns `ok`.

- **F1 applied.** "Pre-start site inspection by the certifier (R7 item 1)" is now in `approvals`, after CDC lodgement. It is also an input to "Certifier determination". "Establish the site" and the SafeWork notification now follow the notice of commencement. The certifier `must_include`, the procurement `why`, DOMAIN §5 and §7 (step 4) are updated.
- **F2 applied, and confirmed by a read-only Go run.** The test was overlaid into `internal/program` from the scratch area with `go test -overlay`; nothing was written to the repo.
  - `if.reinforcement-inspected-before-pour` fires as "Hold point: Reinforcement inspection release identifying the pour." It fires for CDC, for DA and for an extend scope with a new slab.
  - T10 is set to `yes`. The same one-claim correction was made in T01 and in T05 (their shape row, their DOMAIN line, and T05's matching gap text), as the coordinator asked.
  - The same run confirmed that the app labels `fr.demolition-1a`'s row with the work item's title.
- **F3 applied.** The hazmat survey row now has `fragment: null` and `built_in: no`.
- **F4 applied.** Seven fragment rows were added: Pour footings, Cure footings, Set out and excavate slab, Cure slab, Install sarking, Frame internal partitions, Final defects inspection. Set-out is now `partial` and follows the cut, and the footings follow the set-out.
- **F5 applied.** Program `no`, source dropped, the demolition-only CDC made conditional, and an owner-held provisional sum added.
- **F6 applied.** The friable evidence moved to a new `friable-asbestos` trigger. Loose-fill now has the register case only.
- **F7, F8, F9, F10, F12, F13 applied.**
  - F7: structural `planning_approval` included.
  - F8: civil `construction_phase` included, with an `app_differs` row.
  - F9: balustrades `partial`.
  - F10: `asbestos-dividing-fence` trigger added; boundary retaining walls named among the less common variants.
  - F12: the survey row uses the fragment label.
  - F13: the structural option has been added to `must_include`.
- **F11 applied differently.** The validator rejects `scope.structure.masonry` as a `scenario_inputs` key ("not an app row key"). The phrasing is recorded instead as a `gaps` entry and as DOMAIN §10 trap 19.

New counts: 17 triggers, 11 packages, 37 cost lines, 104 activities, 10 RFPs, 27 gaps.

## Round 2 (reviewer, 30 October 2026)

Re-read `shape.yaml`, `DOMAIN.md` and the Round 1 response. Validator, run from the repo root: `ok`,
exit 0. A script check found no dangling `after` references and no duplicate activity labels across
the 104 activities.

### Verdict: **PASS**

### Fixes checked

- **F1 ✔.** The pre-start inspection is in `approvals`, after CDC lodgement, and feeds "Certifier
  determination". "Establish the site" and the SafeWork notice now follow the notice of commencement.
  The certifier `must_include` and `why`, DOMAIN §4, §5 and §7 step 4 all agree. No "pre-commencement"
  row is left.
- **F2 ✔.** `yes`, with the interface as the source; the Go run confirms it. "Reinforce and pour slab"
  follows the hold point.
- **F3 ✔.** `fragment: null`, `built_in: no`.
- **F4 ✔.** The seven rows are added. Their order matches the fragment links and attaches:
  - `fr.roof-1a`: sark, then cover;
  - `fr.linings-1a`: partitions after the roof, then sheeting;
  - `fr.defects-liability-end`: wait, then final inspection.

  Set-out now follows the cut, and the footings follow set-out.
- **F5 ✔.** Program and cost are `no`. The demolition-only CDC is conditional. The owner-held
  provisional sum is added.
- **F6 ✔.** `loose-fill-asbestos` keeps the register case only. `friable-asbestos` is added, and DOMAIN
  §9 is split to match.
- **F7 ✔.** Structural `planning_approval` is included, with a certifier-requests line, and its
  `app_differs` row is removed.
- **F8 ✔.** Civil `construction_phase` and `as_built` are both included, with matching `app_differs` rows.
- **F9 ✔.** Balustrades are `partial` against code 13.
- **F10 ✔.** `asbestos-dividing-fence` is added, and boundary retaining walls are named under the variants.
- **F11 ✔ (accepted as applied).** A `scope.*` key is not a valid scenario input, so the `gaps` entry plus
  DOMAIN §10 trap 19 is the right home.
- **F12 ✔.** The survey row uses the fragment label.
- **F13 ✔.** The structural option is in `must_include`.
- **Rulings §2.1–2.4 ✔.** Applied in DOMAIN §1, §3 and §12, and to the abolishment activity label.

### Remaining items (optional, not blocking)

1. **"Building set-out survey" claims the same app row as "Set out and excavate slab".** Now that
   "Set out and excavate slab" is its own row (`fr.slab-on-ground-1a`), one app row is claimed by both.
   The surveyor's set-out has no app row of its own. Suggested: `built_in: "no"`, drop `source`, why
   "Surveyor set-out has no fragment; the app's slab set-out is the separate row 'Set out and excavate
   slab'". This was my own F4 wording, overtaken by the added row.
2. **The SafeWork notice depends on the notice of commencement.** The removalist can give SafeWork
   notice before the notice of commencement. Only the removal itself needs to wait. Leave it as is
   unless the scorer checks float.
