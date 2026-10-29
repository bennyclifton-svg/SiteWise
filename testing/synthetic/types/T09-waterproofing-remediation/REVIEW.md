# T09 Waterproofing remediation: independent review

Reviewer: independent agent, 29 October 2026. Read-only on everything except this file. No Jev, no network.

## Verdict: **FIX**

The map is sound at the root. The base scenario, the systems walk, the consultants, the NSW pathway, the
program logic and most built-in claims hold up. But it understates one trap that runs in code, misdescribes how
two others work, mislabels several `built_in` values, and leaves out two base-scenario items (the DBP building
compliance declaration and the condition survey of the lots). Fix the items below before it is used as an
answer key.

**Validator:** `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T09-waterproofing-remediation/shape.yaml`
→ `ok`, exit 0.

## Findings

1. **shape.yaml `gaps[19]` (wet-room) and DOMAIN.md §10, §11.6: the bathroom trap is real but understated.**
   `ImpliedReadings` runs in code on every non-boilerplate passage, before and without Jev
   (`internal/jobs/worker.go` ~l.219; `internal/profile/implied.go`). `withImpliedScope`
   (`internal/profile/build.go`) has no work-type gate, so it fires on remediation. The `wet-room` term adds
   `interiors.wet-area-waterproofing`, `interiors.tiling`, **`hydraulic.sanitary`, `hydraulic.cold-water`,
   `hydraulic.hot-water` and `mechanical.local-exhaust`** (`knowledge/profile/scope_defaults.yaml implied_by
   wet-room`). Package rules count scope rows of any band (KNOWLEDGE_INDEX §2.1), so "leaks into the
   bathroom below" also brings the **Hydraulic and Mechanical consultants** (`scope_additions[2]`, `[3]`),
   the 08 and 09 cost lines, and work items with action `repair`.
   **Change:** rewrite `gaps[19].what` to: "The wet-room implied_by pattern fires in code on 'the bathroom
   below', suggesting wet-area waterproofing, tiling, sanitary, cold and hot water and local exhaust scope,
   which then suggests Hydraulic and Mechanical consultants and 08/09 cost lines." Say the same in DOMAIN §10.

2. **shape.yaml `gaps` and DOMAIN §10: this type's own wording fires more implied-scope traps that are
   missing.** All of these come from the same `implied_by` list and run in code:
   - `bedroom` → `interiors.internal-doors`. Your own trigger evidence ("the Unit 304 bedroom", `internal-water-damage`) fires it.
   - `kitchen` → joinery, three hydraulic systems and exhaust, so Hydraulic and Mechanical again.
   - `common wall(s)` (for example "the common wall between balconies") → `site.adjoining-structures`, `fire-passive.compartments`, `interiors.acoustic-separation`.
   - `stair(s)` (for example "material via the fire stairs") → `access-egress.stairs-and-ramps`.
   - `brickwork`, `masonry`, `repointing` (balcony upstands, planter walls) → `structure.masonry`.

   **Change:** add one `gaps` entry (area `profile`, evidence `knowledge/profile/scope_defaults.yaml
   implied_by bedroom, kitchen, party-wall, stair, masonry`). Add these words to the §10 trap table. Use them
   in negative-control documents.

3. **shape.yaml `gaps[17]` (basement) and `gaps[18]` (flood): the mechanism is described wrongly.** Both
   regexes are *triggers*. A match only routes the passage to a Jev question:
   - basement: "does the passage state that the project includes a basement or deep excavation?" (`knowledge/profile/flag_readings.yaml basement asks`);
   - flood: the `flood_hazard` instruction (`knowledge/determinants.yaml flood_hazard`).

   Matching sets nothing on its own. The flood trap is therefore low-consequence: one wasted question and a
   small risk of a wrong reading. The basement trap is high-risk, because the building really does include a
   basement, so Jev may well answer yes. Structural scale is already complex from class 2, so "adds complex
   scale" changes nothing.
   **Change:** for `gaps[17]`, write "the basement trigger routes 'podium over the basement car park' to the
   basement question; a yes reading (likely, as the basement exists) switches on structural retention design,
   temporary works and dilapidation, and civil shoring." For `gaps[18]`, write "routes 'flood test' passages to
   the flood_hazard question; the expected reading is not stated." Scorer expectation: `proj.flag.basement` is
   not `yes`, and `det.flood_hazard` is not `true`.

4. **Program and DOMAIN §5, §7: the DBP building compliance declaration is missing.** The DBP Act requires a
   registered building practitioner to give a building compliance declaration at the end of the work, not
   only design declarations before work starts (DBP Act 2020 s 17 and Regulation lodgement timing for work
   with no occupation certificate: **verified: false**). The program closes with no DBP step.
   **Change:** add an activity `{label: "Lodge building compliance declaration (DBP Act, verified: false)",
   stage: completion, fragment: null, after: ["Balustrade fixing certification"], built_in: "no"}`. Make
   "Practical completion inspection" follow it. Add "Review of the builder's compliance declaration and
   varied regulated designs" to `rfps[consultant.building].must_include`.

5. **New gap: the structural RFP's DBP wording assumes a construction certificate.** In
   `knowledge/rfp/structural.yaml`:
   - `stages regulated_design` reads "Regulated designs and design compliance declarations for one construction certificate";
   - `construction_documents` reads "design certification for the construction certificate".

   On the exempt path there is no CC, yet `proj.regulated_design: "yes"` switches the stage on.
   **Change:** add a `gaps` entry (area `rfp`, evidence `knowledge/rfp/structural.yaml stages regulated_design,
   construction_documents deliverable`). Add "Construction certificate certification" to
   `rfps[consultant.structural].must_not_include`.

6. **DOMAIN §8 contradicts shape.yaml on the structural RFP.** DOMAIN says the app "mis-sizes" the RFP
   because complex scale "includes the Revit model **and construction phase**". But
   `rfps[consultant.structural].must_include` asks for construction-phase inspection. The construction phase
   is right; it comes in for the wrong reason (`construction_phase include_when scale complex`).
   **Change:** in DOMAIN §8, say only the shared Revit model, the concept stage and the geotechnical brief are
   wrong. The construction phase is correct but arrives through complex scale.

7. **`rfps[consultant.structural].must_include[4]` (structural regulated design) is not a base certainty.**
   The DBP s 6 building elements (verified: false) cover load-bearing components essential to stability,
   waterproofing, the enclosure, fire safety and services. A balustrade fixing is probably not among them; a
   slab repair is.
   **Change:** move this line to `triggers[concrete-spalling].adds.rfps[consultant.structural].must_include`.
   In the base, replace it with "Confirm whether any structural work is a regulated building element (DBP
   Act, verified: false)". Leave the app's stage firing as a known effect of `proj.regulated_design`.

8. **`cost_plan` "Investigation opening-up, water testing and make-good", section `consultants`: no package
   can carry it.** The app's consultants section holds only fee lines, one per live services package
   (`internal/costs/suggest.go Suggest`). The `why` says a separate contractor often does this work, and that
   would be a works line.
   **Change:** pick one:
   - (a) rename it "Remedial building consultant: investigation stage incl. opening-up and make-good disbursements" and keep it as part of that package's fee;
   - (b) make it section `construction`, with a `gaps` note that no early-works package exists before the head contract.

   I recommend (a): it matches common NSW practice, where the consultant engages the opening-up contractor.

9. **Missing base items: a condition survey of the lots and the long service levy step.** DOMAIN §3 says to
   record every lot below and beside the works before starting, but shape.yaml has no activity or line for
   it. The long service levy is a cost line with no program step, and with no CC it is paid directly before
   work starts (verified: false).
   **Change:** add these activities, both `built_in: "no"`, and add "Strip tiles…" `after` both:
   - "Pre-commencement condition survey of affected lots", stage construction, after "Contract award";
   - "Pay long service levy before work starts", stage construction, after "Contract award".

   Also add the cost line "Pre-construction condition surveys of lots", section construction,
   `built_in: "no"`.

10. **`triggers[door-threshold-replacement].adds.procurement: [supplier.windows_glazing]` is wrong.**
    `supplier.*` rows are refused (`internal/procurement/discipline.go DisciplinePackage`; your own
    `gaps[6]`). In practice the doors are supplied and installed under the remedial builder's head contract.
    **Change:** remove `procurement` from `adds` and from `built_in`. Note in the `why` that it sits within
    `trade.main_works`.

11. **`triggers[balcony-services-relocation].built_in.procurement: partial` has no matching `adds.procurement`.**
    The app *will* suggest Mechanical and Electrical consultants (`scope_additions[2]`, `[4]`), and DOMAIN §2
    says none is needed. That is a false positive, not a partial match.
    **Change:** drop `procurement` from `built_in`. Add a `gaps` entry: "condenser and balcony-light items
    suggest Mechanical and Electrical consultants the job does not need" (`package_defaults.yaml
    scope_additions[2], [4]`).

12. **`triggers[staged-delivery-funding].built_in.rfps: partial` → `"no"`.** The must_include is for
    `consultant.building`, which has no template and gets the generic `tpl.rfp-capex`. The `staged` module
    (`knowledge/rfp/modules.yaml`) only switches template rows, and those exist only in architecture, access
    and arborist. **Change:** set `rfps: "no"` and drop the source, or cite a real template.

13. **`triggers[heritage-listed].built_in.cost_plan: partial` → `yes`.** The Heritage package is suggested
    (`complexity_additions[1]`), and Cost Plan prepare adopts every suggestion before it makes fee lines
    (`prepareCostPlan`; KNOWLEDGE_INDEX §3.2). **Change:** `cost_plan: "yes"`.

14. **`triggers[development-consent-required].built_in.program: yes` → `partial`.** The DA and CC fragments
    fire on `hdr.cond.planning` alone. But "Occupation certificate" (`fr.occupation-certificate`) is gated
    `work_type new, extend, refurb` (`knowledge/program/fragments.yaml` ~l.3360), so it is false for
    remediation. The skeleton also stays `sk.generic`. **Change:** `program: partial`.

15. **`triggers[hazardous-materials].source` points to the wrong record.** `cq.pre-2004-…` produces an
    Implication only (KNOWLEDGE_INDEX §6). The partial program link is `fr.existing-conditions` (a
    hazardous-materials survey), which is gated `refurb, extend`. **Change:** set `source:
    "knowledge/program/fragments.yaml fr.existing-conditions"` and keep the consequence in the `why`.

16. **`program` "Structural assessment of balcony slabs and balustrade fixings", `built_in: "no"` → `partial`.**
    The `structure.concrete investigate` item gets one generic Investigation row titled with the item
    (`internal/program/generate.go investigationRows`). The `why` already says so. **Change:** `built_in:
    partial`, with `source: "internal/program/generate.go investigationRows"`.

17. **DOMAIN §5 and `rfps[consultant.building]`: Home Building Act contract and licence obligations are
    missing.** All of these are verified: false:
    - Residential building work on a Class 2 building is HBA work. An AS 4000 contract needs HBA-compliant special conditions: a written contract, the deposit cap, the statutory warranty wording and the major-work contract requirements.
    - The remedial builder's own s 18B warranties then run to the OC.
    - The applicator needs a NSW waterproofing licence. Manufacturer accreditation is not enough.

    **Change:** add these to DOMAIN §5. Add "AS 4000 special conditions compliant with the Home Building Act
    1989 (verified: false)" to `must_include`. Add "NSW waterproofing licence for the applicator" to the
    evaluation activity's `why`.

18. **Triggers that change common property: the SSMA resolution is missing.** Replacing balustrades, raising
    sills, replacing doors and enclosing balconies are arguably *alterations* of common property, which need
    a special resolution, not repair under the s 106 duty (SSMA 2015 s 108, **verified: false**).
    **Change:** add "Special resolution if common property is altered rather than repaired (SSMA s 108,
    verified: false)" to `adds.authority` and to `program` in `balustrade-replacement`,
    `door-threshold-replacement` and `development-consent-required`.

19. **`profile hdr.cond.operational_constraints: 24_7_occupied`: test brittleness.** Jev's criterion for
    `24_7_occupied` is "explicitly occupied or operating 24 hours a day" (`internal/profile/questions.go`
    ~l.338). Strata documents say "the building remains occupied", which fits `live_environment` or
    `partial_occupation` (balcony-by-balcony possession) at least as well. All three set `live_operations:
    yes` (`internal/rfp/drivers.go` l.275). **Change:** record in DOMAIN §12 that the scorer accepts any of
    the three, and do not score this key on exact value.

20. **`triggers[heritage-listed]`: the minor route is missing.** A local item may still avoid consent
    through the LEP cl 5.10(3) minor-works notice to council (verified: false). Codes SEPP exempt
    development generally cannot be used on a heritage item (verified: false). **Change:** add both to
    `adds.authority`, and list `hdr.cond.planning` under `profile`.

## Answers to the author's open questions

- **DBP Act.** "Building work" includes repair, renovation and protective treatment of a Class 2 building,
  and waterproofing is a building element. The map is right that it applies to exempt remedial work, but it
  is missing the building compliance declaration (finding 4). Lodgement timing without a CC stays verified:
  false.
- **HBA.** s 18B with s 18E: 6 years for a major defect and 2 years otherwise. Waterproofing has been a
  major element since 2015. HBCF is not needed above a rise of three storeys. All of this matches my
  understanding; keep it verified: false. The HBCF contract-price threshold (about $20k) should be added to
  `low-rise-hbcf` as verified: false.
- **SSMA 2015.** s 106 (duty to maintain), Part 11 (building bond) and entry to lots are correctly
  attributed. Note that the building bond covers only contracts for new buildings from 2018, so the
  `statutory-warranty-defects-claim` bond evidence needs a later `det.existing_building_year` than the base.
  Add s 108 (finding 18).
- **Exempt.** No Codes SEPP exempt type fits like-for-like membrane renewal on a 6-storey building.
  Practitioners treat it as repair or maintenance needing no consent. `exempt` is the app's nearest option,
  and the wording in DOMAIN §5 is adequate.
- **AS 4654.** Parts 1 (materials) and 2 (design and installation), 2012, are correct. In NCC 2022 Vol One,
  external membranes are F1D5 and barriers are D3D17–D3D21, as I recall them. Keep both verified: false.
- **Structural engineer in the base.** Yes. The base re-fixes core-drilled posts, which needs design and
  certification for the barrier loads.
- **AS 4000.** A reasonable base for $0.5–3 M with a superintendent. Below about $0.5 M, AS 4905 or a
  minor-works form is common, and either way it needs HBA conditions (finding 17).
- **Three packages.** Right for this base (2008, no podium, work from inside the balconies). The strata
  manager is correctly not a package. Hydraulic, hazmat, access and landscape are correctly conditional. The
  scaffold and the applicator sit under the builder. The only real gap is the opening-up contractor
  (finding 8).
- **The three traps.** All three are real. Bathroom is the worst: it runs in code and is understated
  (findings 1 and 2). Flood and basement only route a question to Jev; basement is the risky one
  (finding 3).

## Built-in claims checked

| # | claim | result |
|---|---|---|
| 1 | `procurement trade.main_works` partial (ProposeHeadContract, accepted items only) | ✔ |
| 2 | cost: remedial consultant fee partial (Suggest fee loop; package never suggested) | ✔ |
| 3 | cost: structural fee partial | ✔ |
| 4 | cost: membrane → 04 partial | ✔ |
| 5 | cost: flashings → 04 partial | ✔ |
| 6 | cost: tiling → 05 partial | ✔ |
| 7 | cost: outlets → 09, `trade.hydraulic_plumbing` partial | ✔ |
| 8 | cost: balustrades → 13 child "Compliant balustrades and barriers" partial | ✔ |
| 9 | cost: design contingency `lines[0]` yes | ✔ |
| 10 | cost: construction contingency `lines[1]` yes | ✔ |
| 11 | cost: escalation `lines[2]` yes | ✔ |
| 12 | cost: exclusions `lines[6]` yes | ✔ |
| 13 | program: `fr.exempt-confirmation` yes (`cond planning eq exempt`) | ✔ |
| 14 | program: `fr.contract-award` partial (new, extend, refurb gate) | ✔ |
| 15 | program: `fr.balcony-waterproof-2` partial (new/extend and new/replace; `du` per_unit GFA) | ✔ |
| 16 | program: `fr.practical-completion` partial | ✔ |
| 17 | program: `fr.handover` partial (attach after `fr.occupation-certificate`) | ✔ |
| 18 | program: `fr.defects-liability-end` partial | ✔ |
| 19 | program: structural assessment `no` | ✘ partial (finding 16) |
| 20 | rfp: structural `existing_assessment` partial (`work_type remediation`) | ✔ |
| 21 | podium: procurement, cost and rfps partial (`scope_additions[3]`; Landscape has no rule) | ✔ |
| 22 | hazmat: program partial, source `cq.pre-2004…` | ✘ wrong source (finding 15) |
| 23 | spalling: rfps partial (`strengthening` option), cost partial (03) | ✔ |
| 24 | balustrade: cost partial, rfps partial (`metalwork` option) | ✔ |
| 25 | door: program partial (`fr.windows-2` gated), cost partial | ✔ |
| 26 | consent: procurement partial (Town Planner only), cost yes (`da-fees`, `cc-fees`), rfps yes | ✔ |
| 27 | consent: program yes | ✘ partial (finding 14) |
| 28 | heritage: procurement yes, program yes (`fr.heritage-referral`) | ✔ |
| 29 | heritage: cost partial | ✘ yes (finding 13) |
| 30 | internal damage: cost partial, program partial (`fr.ceilings-2` gated) | ✔ |
| 31 | services: procurement partial | ✘ a false positive (finding 11) |
| 32 | services: cost partial | ✔ |
| 33 | staged: rfps partial | ✘ no (finding 12) |
| 34 | roof terrace: cost partial (04) | ✔ |

Spot-checked `no` claims (all ✔: the knowledge has nothing):

- long service levy (no knowledge file mentions it);
- HBCF (none);
- preliminaries line (none);
- `consultant.building` package rule (none);
- strata manager id (none in `disciplines.json`);
- external-membrane hold point (`if.envelope-waterproofing-before-tiles` is wet-area only and free-text);
- expert witness template (none);
- remediation skeleton (none);
- `uc.strata-consent…` gated to refurb (confirmed).

All `[K:]` record ids cited in DOMAIN.md exist.

---

# Round 2 (re-review after the author's fixes, 29 October 2026)

**Validator:** `ok`, exit 0. Checked: DOMAIN.md is now 523 lines; shape.yaml has 32 activities and 24 gaps,
which matches DOMAIN §7 and §11.

| # | applied | note |
|---|---|---|
| 1 | ✔ | `gaps` wet-room entry rewritten; DOMAIN §10 names the code path, the extra systems and the Hydraulic and Mechanical consultants. |
| 2 | ✔ | New `gaps` entry and DOMAIN §10 rows for bedroom, kitchen, common wall, stairs and masonry; negative-control use stated. |
| 3 | ✔ | Basement and flood are now described as routing to a Jev question, with scorer expectations. |
| 4 | ✔ | Building compliance declaration added at completion (verified: false); PC now follows it; RFP review item added. |
| 5 | ✔ | New rfp gap for the CC wording; "Construction certificate certification" added to structural `must_not_include`. |
| 6 | ✔ in DOMAIN, ✘ in one gap | DOMAIN §8 is fixed. shape.yaml `gaps` rfp entry ("includes a shared Revit model and construction phase") still presents the construction phase as an error (new finding R2-1). |
| 7 | ✔ | Base line reworded; slab-repair regulated design moved to `concrete-spalling`. One stale `why` remains (R2-2). |
| 8 | ✔ | Option (a) taken; DOMAIN §6 matches. |
| 9 | ✔ | Condition survey activity and cost line, and the levy activity, added; "Strip tiles…" `after` both. |
| 10 | ✔ | `supplier.windows_glazing` removed; `why` says the doors sit under the head contract. |
| 11 | ✔ | `built_in.procurement` dropped; gap added; source now `elements.yaml code 08` (fine for the cost line). |
| 12 | ✔ | `rfps: "no"`, with a `why`. |
| 13 | ✔ | `cost_plan: "yes"`. |
| 14 | ✔ | `program: partial`, with a `why`. |
| 15 | ✔ | Source is now `fr.existing-conditions`; the consequence is kept in the `why`. |
| 16 | ✔ | `partial`, source `investigationRows`. |
| 17 | ✔ | HBA contract conditions, the builder's warranties and the applicator licence are added (verified: false), in DOMAIN §5, `must_include` and the evaluation `why`. |
| 18 | ✔ | s 108 special resolution added to the three triggers (verified: false) and to DOMAIN §5. |
| 19 | ✔ | DOMAIN §12 tells the scorer to accept any of the three values. |
| 20 | ✔ | cl 5.10(3) and the Codes SEPP limit added (verified: false); `hdr.cond.planning` added to the trigger's `profile`. |

**New legal numbers.** Every number added this round is marked verified: false: DBP s 17, SSMA s 108, the HBCF
threshold of about $20,000, the 2018 start of the building bond, LEP cl 5.10(3), and paying the long service
levy directly. Each appears in DOMAIN §5 or §9 inside an "all verified: false" block, and again in the §12
list of claims added after review. The trigger `why` for the defects claim keeps the 2018 date as verified:
false and correctly separates it from the 2008 base. I found no unmarked numbers.

## New findings (minor)

- **R2-1. shape.yaml `gaps`, rfp entry citing "structural.yaml scale.complex".** It says the RFP "includes a
  shared Revit model and construction phase", as if both were wrong. **Change:** "…includes a shared Revit model
  (the construction phase is right but arrives through complex scale)…".
- **R2-2. shape.yaml `procurement[1].why` (`consultant.structural`).** It still says the engineer "prepares the
  structural regulated design" in the base. **Change:** "…and confirms whether any structural work is a
  regulated building element (DBP Act, verified: false)."

## Round 2 verdict: **PASS**

Nineteen of the 20 findings are fully applied. Finding 6 is applied in DOMAIN but left one stale line in
shape.yaml. Neither of the two nits changes a scored expectation. Fix them when convenient; the map is usable
as an answer key.
