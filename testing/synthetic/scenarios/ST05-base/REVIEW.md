# Review: ST05 scenarios (base, heritage-conservation-area, first-floor-addition)

Reviewer: independent (brief 03r), 2026-11-04.

## Verdict: **FIX**

Seven findings. Findings 1 and 2 are the most serious. They key two base profile values as readable,
but the app cannot read them as worded or at all. The ST01 run `runs/20261102-fixes-1-4-6` missed both
with the same wording.

## Checks

1. **Checker.** `validate_expected.py` reports `ok` for all three scenarios.
2. **Faithful to the shape.** No base document fires a trigger. Each negation is used in the shape's own
   words: d01 vacant; d03 sewer in the street and no solar; d07 OSD and "55 m² of new roof"; d11 asbestos,
   fibre cement and loose-fill; d13 single storey, the 1.5 m setback, the detached house, the supply
   capacity and Dulux Heritage; d16 the Housing Code DA and the 1987 approval. The declared traps (d02,
   d03, d07, d11, d12, d13, d18) are real trigger hits, and their expected outcomes are right. d02's
   `cdc-land-eligible` cannot fire at all. A certificate is not in `headerKinds`
   (`internal/profile/questions.go`), so d02 gets no header questions. The trap is harmless. One
   consistency problem is in finding 3.
3. **Register.** Every clean title block uses labels from DISCOVERY.md §4.3 (`Proposal no`, `Quote no`,
   `Quote number`, `Proposal ref`, `Report no`, `Document number`, `Drawing number`, `Ref`). Each has a `#`
   heading identical to `Title:`, or no heading (d14 and d15). The d02 and d07 table blocks are declared
   missed in notes.md. Supersession d15 → d14 is declared unsupported. Heritage d16, d20, d21 and d22 are
   right. See finding 6 for the d04 date.
4. **Profile.** I checked each `says` against `knowledge/determinants.yaml`, `flag_readings.yaml`,
   `project_details.yaml`, `project_facts.yaml` and the scale triggers in `taxonomy.yaml`. All of them
   trigger except `fact.contract_form` (finding 1) and `proj.regulated_design` (finding 2). The author
   fixed the T01 misses (staged, multi-building, sustainability, groundwater, existing building, gas,
   address) with trigger words: "staged construction", "multiple buildings", "Green Star, NABERS",
   "groundwater … dewatering", "existing building", "natural gas" and "Avenue". The route sits under
   `## Contract and procurement` (F4). `read_by_design` is right for d04 and the drawings. d16's
   "neighbour notification … no public exhibition" read correctly as `community` no in the ST01 run.
5. **Fees and invoices.** Every stage sum equals its total:
   - d05: 9,500 + 12,500 + 19,000 + 3,000 = 44,000.
   - d06: 6,800 in the base and 9,600 in the first-floor variant.
   - d07: 7,260 inc GST / 1.1 = 6,600 ex GST (stages 2,200, 3,200 and 1,200), with `gst_inc_normalised`.
   - d08: 3,200 in the base and 3,900 in the first-floor variant.
   - d09: 2,300. d10: 1,200. d20: 4,700. d21: 6,500.

   d18 has GST of $345 on $2,300 with a total that adds up. `ledger.go` raises `gst_rate` there and not
   `arithmetic`, so the key is right. d17 and d19 add up, and d19 equals d07's stage 1. All nine ABNs pass
   the checksum.
6. **Variants.** Each variant changes only its trigger's profile key: storeys 1 → 2, and heritage
   none → conservation_area. The evidence sentences are the shape's: "go up rather than out", "New
   first-floor addition over the existing house…", Schedule 5 and the heritage officer. The heritage
   variant replaces "contributory" with "neutral, not contributory". The value is still
   conservation_area, so that is acceptable. The passing mentions were searched and the changes are listed.
   One leftover is in finding 5.
7. **Realism.** A 55 m² rear addition plus bathroom, laundry and two walls removed, on a 1958 brick house
   in Concord (Canada Bay, R2, 557 m²), is a typical Sydney alterations job. A $640k budget fits. The fees
   fit the scale: architect $44k to tender, structural $6.8k, civil $6.6k, geotech $3.2k, hazmat $2.3k,
   BASIX $1.2k, plus HIS $4.7k and planner $6.5k under heritage. The calendar works: brief in February,
   survey and fee proposals in March, hazmat and geotech reports in April, DA set at the end of May, and a
   start in March 2027. In the first-floor variant, an unchanged $640k budget and an unchanged architect fee
   are light. The shape does not let either change, so this is noted only.

## Findings

1. **All three scenarios, d01: `fact.contract_form` = domestic_hia cannot be read as worded.** The fact is
   asked only when a passage matches `\bas\s?4902`, `\bas\s?4000`, `\bas\s?2124`, `\babic\b`,
   `building contract` or `domestic building contract` (`knowledge/profile/project_facts.yaml`
   contract_form). "HIA NSW Lump Sum Contract with the builder" matches none of them. ST01 used the same
   sentence and the key came back `missed` (`runs/20261102-fixes-1-4-6/ST01-base/field_score.json`).
   **Change:** in all three manifests, set d01 `carries` `fact.contract_form` `says` to "The builder will
   be engaged under the HIA NSW Lump Sum building contract". The document text must use that sentence.
   Keep the key.

2. **All three scenarios: `proj.regulated_design` is keyed `read_by_design: true`, but the app never reads
   it.** `project_details.yaml` gives `regulated_design` no `extraction` and no triggers, so
   `Harvested.addDetail` returns at once (`internal/profile/harvest.go`). The ST01 run missed it. The RFP
   derives "no" on its own when no Class 2, 3 or 9c part exists (`internal/rfp/drivers.go`, around line
   332), so the RFP keys are unaffected.
   **Change:** in all three `profile.json` files, set `proj.regulated_design` `read_by_design` to false.
   Add a line to "Missed by design" in each notes.md: "`proj.regulated_design` has no reading in the app;
   the d05 sentence is kept for realism." Keep the value "no".

3. **All three scenarios, d03 against d13: the frangipani is inside the "5 m of the works" that d13
   denies.** d03 puts the frangipani "about 9 m from the rear wall". A 55 m² addition across an 8 m to
   9 m wide house runs about 6 m to 7 m into the yard, which leaves the tree about 2 m to 3 m from the
   works. In the first-floor variant (30 m²) it is about 5 m. d13 then says "No trees within 5 m of the
   works". The `arborist_tree_impacts` "no" rests on a statement the survey contradicts. The PM would see
   it, and so would a later significant-tree variant built on this base.
   **Change:** in all three manifests, rewrite the d03 content `## Trees` as "frangipani near the back
   fence, about 16 m from the rear wall of the house". The lot is 45.7 m deep, so it fits.

4. **ST05-first-floor-addition, d13: the GFA split puts three "added" candidates in one passage.**
   "Added gross floor area: 55 m2 (rear addition 30 m2, first floor 25 m2)" yields the candidates 55, 30
   and 25. `areasOfRole` gives each the role "added": the cue "addition" is within 40 characters of
   "30 m2" and of "25 m2" in the same sentence (`internal/profile/harvest.go`). Jev must then choose 55. If
   it picks 30 or 25, that is a wrong value, and against d16's 55 it is a conflict.
   **Change:** set d13 `carries` `hdr.scale.added_gfa_sqm` back to the base's "Added gross floor area:
   55 m2". Describe the first-floor bedroom and ensuite in d13 `## Design` without m² figures. Keep the
   30/25 split only on the d14 and d15 area schedules, which are not read.

5. **ST05-first-floor-addition, `project.summary` still says "A single-storey 55 m2 rear … addition".**
   The corpus writer works from the manifest. A stray "single-storey" in a read document would make
   `hdr.scale.storeys` a 1/2 conflict.
   **Change:** set the summary to "A 30 m2 rear kitchen, living and dining addition and a 25 m2 first-floor
   main bedroom and ensuite over the rear of a 140 m2 brick house built in 1958, making it two storeys,
   with the bathroom and laundry redone and two loadbearing walls removed, on a 557 m2 R2 lot in Concord.
   DA to the City of Canada Bay, owners move out, architect-led tender to builders under an HIA lump-sum
   contract."

6. **All three scenarios, d04: the key date depends on which message is later.** An email's date is the
   latest `Date:` in the thread (DISCOVERY.md §4.3). The key is 2026-03-04, but the content does not
   say which of the two messages carries that date. If Tomas writes on 4 March and the architect replies
   on 5 March, the key is wrong.
   **Change:** in the d04 content, add "Tomas writes on 3 March 2026; the architect replies on 4 March
   2026."

7. **All three scenarios, d01 `hdr.cond.quality_level` and d13 `hdr.cond.environmental_sensitivity` sit
   under headings that header routing does not accept.** `## Budget and quality` and `## Site and context`
   do not match `headerSection`. So the questions are asked only if the passage ordinal is 12 or less
   (`HeaderRouted`). They probably are, but a longer `## What we want` or `## Project description` would
   push them past 12. This is low risk, and the fix is cheap.
   **Change:** rename d01's heading to `## Scope, budget and quality` and d13's to
   `## Site description and context`. Both match `scope` or `description`.

## On the author's shape comments (orchestrator question)

1. **`hazardous-materials-found` listing `det.existing_building_year`: the author is right. Change it to
   `profile: []`.** Finding asbestos does not change the year built. None of the trigger's evidence
   sentences contains "built in <year>", so the trigger cannot move that key. Listing it only lets the
   checker accept a changed year, which is the opposite of what the list is for. No readable profile key
   records hazardous materials: `planning_keys.yaml` `hazardous_materials_present` is a planning
   assumption, not a reading.

   A related problem: the trigger's `adds` repeat base items. `consultant.hazmat` is already in the base
   procurement, and "Hazardous materials survey" and "Demolition, strip-out and hazardous materials
   removal" are in the base cost plan. `score_shapes.py` `expectations()` drops the repeats, so a variant
   would differ from the base only in program and authority. The base is right to carry the survey, since
   a 1958 house needs it before demolition. The trigger's adds should name only the removal consequences,
   for example a cost line "Licensed asbestos removal and clearance".

2. **`first-floor-addition` and `hdr.scale.added_gfa_sqm`: agree. Add `hdr.scale.gfa_sqm` too.** Going up
   adds floor area, and the proposed GFA is existing plus added, so both move together. Under the shape as
   written, the variant is still a valid key, because the checker passes and the 30/25 split holds 55 m².
   No redo is needed. With both keys listed, the variant would not need the contrived split. That also
   removes finding 4 and eases the tension between "go up rather than out" and still building out.

3. **The town planner under heritage (the author's third note).** The scorer already handles it, so no
   change is needed. `expectations()` drops a trigger-added discipline from `not_procurement`.
   `norm()` strips the bracketed aside, so the added "Town Planner" line also clears
   "Town Planner (consultant fee)" from `not_cost_plan`.
