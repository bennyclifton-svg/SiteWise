# Review: ST17 scenarios (base, project-remediate, cdc-pathway)

Reviewer: independent (brief 03r), 2026-11-06.

## Verdict: **FIX**

Nine findings. Finding 1 is the most serious. `ST17-base/expected/notes.md` holds another type's notes. It is
byte-identical to `ST02-base/expected/notes.md` (Calloway Street, 64 apartments, AS 4902). So the base declares no
trap, no missed-by-design value and no shape comment. Both variants' notes say "As the base" and point at nothing.
Finding 2 is next. The shape's note on `det.type_of_construction` is wrong. In the app, the key's derived "A" comes
out blank.

The manifests and keys themselves are sound. The register, fee and invoice keys are right. The profile values
are supported, and both variants change only their trigger.

## Checks

1. **Checker.** `validate_expected.py` reports `ok` for ST17-base, ST17-cdc-pathway and ST17-project-remediate.
   It does not read notes.md, so finding 1 passes it.
2. **Faithful to the shape.** The base matches the shape's base job:
   - 8 storeys, 56 apartments, Class 2, constructed 2011, not sprinklered, Type A.
   - PE-core ACP and EPS render, a fire safety order and special levies.
   - A DA, AS 4000 lump sum, traditional procurement and the DBP Act.

   Every trigger is denied in the shape's own negation wording, or is absent:
   - Project Remediate ("special levies without government assistance", d02, d13).
   - Performance solution (d14). Hazmat ("2020 hazardous materials survey found no asbestos", d04).
   - Concealed damage and balcony waterproofing (d15). Balustrade ACP (glass balustrades retained, d04, d13).
   - Facade services ("no services are fixed to or through the cladding", d04). Windows retained (d13).
   - Fire upgrades ("the order relates to the external wall cladding only", d01, d14). Boundary access and
     scaffold (d13). Heritage (d03, d13, d16).
   - D&C (d02, d13). Defects claim, HBCF, staged funding (d13). Mixed-use podium (d04). Order-without-consent (d02).

   The declared traps are right for the base: remediation, the RFP cue, cdc-eligible land, bedroom, basement (x2),
   BASIX, Type A, not-sprinklered, fee stages and exempt repainting. Finding 8 lists two keyword triggers that fire
   but are not declared.
3. **Register.** Numbers, revisions, titles and dates agree with DISCOVERY.md §4.3.
   - Labels used: `Ref`, `Report no`, `Proposal no`, `Proposal ref`, `Quote no`, `Drawing number`, `Revision`,
     `Report date` and `Date`.
   - Every `#` heading is identical to its `Title:`. d03 has a heading and no `Title:`, so its heading settles the
     title. d17/d18 have `Title:` and no heading.
   - d03 and d11 are table title blocks, so their number and date cannot be read. d18 → d17 supersession cannot be
     linked (§4.5). The keys rightly hold the true values. These are declared only in the variants' notes (finding 1).
   - d05 is a proper email, so its title is the Subject.
4. **Profile.** Checked against the real triggers:
   - `existing_building_year`: "constructed in 2011" matches `\b(?:built|constructed) in \d{4}\b`.
   - `units`: "56 apartments" matches `unitCountRe`. `rise_in_storeys`, `existing_building`, `sprinklered`,
     `heritage`, `flood` and `contaminat` all match.
   - `facade_type`: "rainscreen" matches. `contract_form`: `\bas\s?4000` matches. `contract_basis`: the literal
     "lump sum" matches.
   - `budget_stated` is worded as a budget, not an estimate (the T09 fix is applied).
   - Every keyed flag's sentence matches its `flag_readings.yaml` trigger: `staged`, `multi_building`, `basement`,
     `excavation_adjoining`, `transfer_structures`, `authority_asset`, `osd_existing`, `cl46`, `panel`, `community`,
     `country`, `sustainability_target`, `rail` and the three `fire_*`.
   - `read_by_design` is false for the d05 email, and for `proj.lot`, `proj.regulated_design` and
     `det.type_of_construction`. That is right (finding 2 covers the last one's wording).
   - One source does not trigger (finding 7).
5. **Fees and invoices.**
   - Every stage table sums to its stated total: d06 $182,000, d07 $234,000, d08 $48,000, d09 $76,000, d10 $33,000
     and d12 $16,500.
   - d11 is GST-inclusive, and each figure divided by 1.1 is right: 9,900 → 9,000, 3,300 → 3,000, 8,800 → 8,000,
     5,500 → 5,000, 22,000 → 20,000, total 45,000, rate 198 → 180. `gst_inc_normalised` is a real flag
     (`proposals/read.go`).
   - All eight ABNs pass the checksum.
   - d19 is clean. d20: 14,500 + 1,450 = 15,950 against $16,850 printed, so the ledger raises `arithmetic`
     (`invoices/ledger.go:306`). GST is exactly 10%, so it raises no `gst_rate`.
   - Variant fee labels follow the documents (the "CDC …" labels in cdc-pathway).
6. **Variants.**
   - **project-remediate** changes no profile value. Both evidence sentences are placed verbatim in d02 and
     paraphrased in d13. d06 is removed, and d05 (not read) is updated. No read document still says "special levies
     without government assistance".
   - **cdc-pathway** changes only `hdr.cond.planning` (da → cdc). Both evidence sentences are in d02 and d13. d12
     and d16 are removed.
   - **No read document in cdc-pathway says "DA", "development application" or "development consent"** (confirmed
     by search of every `content`, `carries` and `title_block`). It also does not say "lodge", "Planning Portal",
     "SEE" or "statement of environmental effects". The fee labels in fees.json are also clean. The only stale
     wording is in two places that are not documents: the manifest `project.summary` and the d03 trap's `expect`
     (finding 6).
   - Note: cdc-pathway loses d12, which was the only row-dropped proposal. That variant therefore does not exercise
     the row drop. This is acceptable, because the trigger removes the town planner.
7. **Realism.** A PM would recognise this as a real Sydney strata recladding:
   - City of Ryde, R4 under Ryde LEP 2014.
   - The sequence: audit (Sep 2025) → fire safety order with interim measures → strata manager's brief → seven
     consultant proposals → fire risk assessment, slab-edge pull-out tests and the options report → SEE and DA →
     start March 2027 against a December 2027 compliance date.
   - The building facts hang together. An 8-storey 2011 Class 2 building under 25 m effective height being
     unsprinklered fits the pre-2019 NCC.
   - Cost: $6.85M ex GST for 56 apartments is about $122k per apartment, including a full scaffold over the
     footpath. That is plausible for full ACP and EPS replacement.
   - Fees: consultants total about $634k, or 9.3% of construction. PM 2.7%, architect 3.4%, fire $48k, facade $76k
     and structural $33k are all plausible for remedial work.
   - Minor point for project-remediate: an OC enrolled in Project Remediate would normally have had its cladding
     identified before enrolment. A first audit in September 2025 sits awkwardly with an enrolment made "years ago"
     (the shape's own `why`). Leave it unless the owner minds; it changes no key.

## On the consultant adaptation

**Right.** The brief's civil and geotechnical defaults have no role in a recladding. There is no excavation, no
ground works and no stormwater change. d10 and d15 say so in words that set `flag.basement` "no", and d13 says the
existing stormwater is not altered.

The shape (`procurement`) names the PM, architect, fire engineer, facade engineer, structural engineer, QS and town
planner as the base team, with the BCA consultant and certifier later. The seven proposals match it exactly:
- The facade engineer is the only one the app suggests (`complexity_additions[7]`).
- The fire engineer and the PM test packages the app never suggests.
- The QS carries the GST-inclusive figures.

Keep it.

## On the author's shape questions

1. **"(fictional)" in `proj.client`: agree. Change the shape.** See finding 4.
2. **"completed in" and "residential units": agree. Both match no trigger.**
   - `existing_building_year` triggers only on `year (built|of construction|constructed)` or
     `(built|constructed) in \d{4}`.
   - `unitCountRe` and the taxonomy units trigger need the noun directly after the number (only "sole occupancy" may
     come between). "56 residential units" fails.
   - The scenarios already use "constructed in 2011" and "56 apartments", so no key is affected. Several shape
     strings need changing, though. See finding 3.
3. **Type of construction note: the author is right on both counts.**
   - A rule derives it: `rule.ncc.type-of-construction` (`knowledge/clusters/fire/rules.yaml`) looks up
     `knowledge/tables/type_of_construction.yaml` from `ncc_class` and `rise_in_storeys`.
   - Jev never reads it: the determinant has `derived: true` and no triggers. `HarvestOwn` skips derived
     determinants (`internal/profile/harvest.go`), and `derive()` in `reconcile.go` overwrites any non-user row
     anyway.
   - Further, both the rule and the table are `status: draft`. `Catalog.Derive` (`internal/knowledge/evaluate.go`)
     therefore returns unknown with reason `unreviewed`, and the app shows the row blank.
   - The shape's "verified: false" is also wrong: the rule's numbers and clause are `verified: true`.
   - See finding 2.
4. **`proj.client` in the `project-remediate` profile: agree. Remove it.** The client stays the owners corporation
   under the program; the variant keeps the same value, and nothing in the trigger changes it. The checker compares
   values only, so `profile: []` still passes. See finding 5.

## Findings

1. **ST17-base `expected/notes.md` is the ST02-base notes file.**
   - **What is wrong.** `diff` shows it identical to `ST02-base/expected/notes.md`. It describes Calloway Street,
     West Ryde, and declares ST02's traps, missed values and shape comments. It says nothing of Tideway
     Apartments. Both ST17 variants' notes say "Missed by design: As the base" and "Traps: As the base".
   - **Change.** Rewrite `ST17-base/expected/notes.md` for Tideway Apartments. Keep `ST02-base` as it is.
     - A short summary.
     - **Traps:** one line per declared trap, by document.
       - d02: `remediation-not-contamination`, `rfp-cued-as-proposal`.
       - d03: `cdc-eligible-land`.
       - d04: `bedroom-window`, `basement-exists-not-excavated`.
       - d06: `fee-stages-not-staged-works`.
       - d13: `basement-scaffold`, `basix-does-not-apply`.
       - d14: `type-a-not-a-class`, `not-sprinklered-is-evidence`.
       - d16: `exempt-repainting`.
       - d20: `total-arithmetic`.
       - The two traps from finding 8.
     - **Missed by design:**
       - `proj.target_construction_start`: only in the d05 email.
       - `proj.regulated_design`: no extraction (`details.go compileReading`); it stays unknown rather than derived
         "no", because a Class 2 part is present.
       - `det.type_of_construction`: wording from finding 2.
       - `proj.lot`: `lotRe` has no "Lots 1 to 56" range form.
       - `proj.client`: `clientCandidates` stops at " - ", so it offers only "The Owners".
       - d03 and d11: table title blocks, so number and date are missed and the title comes from the heading.
       - d18 supersedes d17: the app cannot link it (DISCOVERY.md §4.5).
     - **Shape comments:** the four questions above, with findings 2 to 5.

2. **Shape and all three notes: `det.type_of_construction` is derived, never read, and blank today.**
   - **What is wrong.** The shape's `expected_derived` note says "No derivation exists; it is read or typed only"
     and "(… Table C2D2, verified: false)". The `scenario_inputs` note "Stated in the BCA report" implies a read.
     All of this is wrong (see shape question 3). The variants' notes say "(not harvested)", which is half right.
   - **Change.**
     - In `types/T17-cladding-remediation/shape.yaml` `expected_derived`, replace the type_of_construction `note`
       with: "rule.ncc.type-of-construction derives it from det.ncc_class and det.rise_in_storeys
       (knowledge/tables/type_of_construction.yaml, NCC 2022 Vol One Table C2D2, verified: true). The determinant
       has no triggers, so Jev never reads it, and the derived row replaces any read value. The rule and the table
       are status: draft, so Derive returns unknown (reason unreviewed) and the app shows it blank until the owner
       reviews them."
     - Set `source:` to "knowledge/clusters/fire/rules.yaml rule.ncc.type-of-construction".
     - In `scenario_inputs`, change its `note` to: "Stated in the BCA or fire report for realism only; it is
       derived, never read."
     - In all three notes.md, the missed-by-design line reads: "det.type_of_construction (A): derived only, never
       read; blank while rule.ncc.type-of-construction and its table are draft (reason unreviewed). The key keeps
       the true value." Keep the profile entry (`sources: []`, `read_by_design: false`, `stated_in: [d14]`) and the
       `derived` entry as they are.

3. **Shape: phrasings that match no trigger.**
   - **What is wrong.** "completed in <year>" fails both `existing_building_year` triggers. "<n> residential units"
     fails `unitCountRe` and the units trigger. The scenarios avoided both, but the next author following the shape
     would not.
   - **Change in `types/T17-cladding-remediation/shape.yaml`:**
     - `scenario_inputs` `det.existing_building_year` `phrasing`: "The building was constructed in <year>."
     - `scenario_inputs` `hdr.scale.units` `phrasing`: "<n> apartments". Make its note: "Put 'apartments' or
       'units' directly after the number (unitCountRe); '56 residential units', '56-unit' and '56 lots' match
       nothing."
     - `hazardous-materials` evidence: "The building was constructed in 1998."
     - `hazardous-materials` negation: "The building was constructed in 2011." (replacing "Construction completed
       in 2011.").
     - `defects-claim` negation: "The building was constructed in 2011, outside the statutory warranty and
       rectification order periods."
     - Header comment (optional, prose only): "56-apartment … constructed in 2011".

4. **Shape and all three profiles: "(fictional)" in `proj.client`.**
   - **What is wrong.** The shape value is "The Owners - Strata Plan 87654 (fictional)". The documents say "The
     Owners - Strata Plan No. 87654". The accept entry with "(fictional)" exists only to satisfy the checker's
     base-equals-shape test, and T09 has already removed the same thing.
   - **Change.**
     - Shape: `proj.client: "The Owners - Strata Plan No. 87654"`.
     - In the `proj.client` accept list of all three `expected/profile.json`, remove
       "The Owners - Strata Plan 87654 (fictional)". Keep the other two entries.
     - The truncation to "The Owners" stays declared as missed (finding 1).

5. **Shape: `project-remediate` lists `proj.client` in `profile`.**
   - **What is wrong.** Nothing in the trigger changes the client, and the variant keeps the base value. The entry
     only loosens the checker for that key.
   - **Change.** `profile: []` in the `project-remediate` trigger. No scenario file changes. Re-run the checker; it
     compares values only, so it stays `ok`.

6. **ST17-cdc-pathway and ST17-project-remediate: stale base wording outside the documents.**
   - **What is wrong.**
     - cdc-pathway d03 trap `cdc-eligible-land` still expects "hdr.cond.planning stays da (d02, d13, d16)". In this
       variant the key is cdc and d16 is gone.
     - Both variants' `project.summary` still reads "Funded by special levies, it appoints a project manager …
       lodges a DA". That is false for project-remediate (funding, PM) and for cdc-pathway (DA).
   - **Change.**
     - cdc-pathway d03 trap `expect`: "hdr.cond.planning is cdc from d02 and d13; the certificate's
       land-eligibility wording agrees but is not the works' pathway and is not a source."
     - project-remediate summary: "Funded by a Project Remediate interest-free loan, with the program's appointed
       project manager; it appoints a consultant team from the program panels, lodges a DA …"
     - cdc-pathway summary: "… appoints a project manager and consultant team, obtains a complying development
       certificate from the certifier for the change of appearance …"

7. **All three scenarios, d14: the `det.ncc_class` source does not trigger.**
   - **What is wrong.** d14's carry "Class 2 residential flat building" matches none of the three `ncc_class`
     triggers. The third trigger needs `building|part|occupancy|portion|use` directly after "Class 2", and the
     passage has no "NCC/BCA/building classification". Harvesting is per passage, so d14 yields no question. d04
     ("NCC building classification: Class 2") still carries the value.
   - **Change.** Either make d14's `says` (and the document) "NCC building classification: Class 2", in all three
     manifests, or remove d14 from `det.ncc_class` `sources` in all three `profile.json`. The first is better: it
     keeps the `type-a-not-a-class` trap meaningful, because the trap only bites when the passage is asked.

8. **All three scenarios: two keyword triggers fire but are not declared as traps.**
   - **d07 (fee proposal, read).** The carry "Stage 5 includes the architectural regulated designs …" matches the
     `staged` trigger `\bstages? [1-9a-c]\b`. In project-remediate, d06's declared trap is gone, so this is the
     only fee-stage hit left, and it is undeclared.
     - Add to d07 in all three manifests: trap `{"id": "stage-5-fee-stage", "what": "'Stage 5 includes the
       architectural regulated designs' matches the staged trigger.", "expect": "flag.staged stays no (d13)"}`.
   - **d01 (fire safety order, kind accepts `certificate`, a read kind).** The interim-measures list ("no storage or
     barbecues on balconies, … monthly inspection of the essential fire safety measures") can match the
     `fire_special_hazards` trigger `(storage|…)[^.]{0,90}(fire|…)` within 90 characters.
     - Add to d01 in all three manifests: trap `{"id": "balcony-storage-not-special-hazard", "what": "The order's
       interim measure 'no storage or barbecues on balconies' sits near 'fire safety measures'; matches the
       fire_special_hazards storage pattern.", "expect": "flag.fire_special_hazards stays no (d14)"}`.
     - The d01 tier can stay `clean`.

9. **ST17-project-remediate notes.md: the scorer summary over-claims "procurement board is unchanged".**
   - **What is wrong.** The notes say the app "never suggested a PM for remediation, so the procurement board is
     unchanged in the app". That is right for suggestions. But the base board also had a PM row from d06's
     proposal (`from_proposals`). In this variant d06 is gone, so the app's board does lose the PM row, as the
     trigger's `removes` expects.
   - **Change.** Replace that sentence with: "The app never suggests a PM for remediation; with d06 removed no
     proposal creates the PM row, so the board matches the trigger's removal. The program and cost additions are
     expected to be missed."
