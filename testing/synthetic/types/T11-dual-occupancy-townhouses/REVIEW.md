# T11 Dual occupancy or townhouses: independent review

Reviewer brief: `testing/synthetic/briefs/02-review.md`. Read 30 October 2026 against `main` plus the working
tree. I wrote only this file. I ran the validator and read-only Python over the YAML. I made a few public web
reads (NSW Planning, Sydney Water, ABCB) to rule on the author's open questions. They are cited where used. I
made no Jev call and ran no git command that changes state.

## Verdict: **FIX**

The map is right at the root. It names the attached-dwelling content (the separating wall, acoustic
separation, stairs and barriers) and the subdivision chain (Section 73, the final plan, the 88B instrument,
the subdivision certificate and LRS registration), and it shows where the app has neither. The package,
fragment, cost and RFP claims mostly check out against the code.

It is not yet a reliable answer key, for these reasons:

- the base leaves out the demolition that most Sydney jobs start with;
- it gets the separating-wall roof rule wrong;
- it treats two charges that do apply as "may apply";
- several pieces of trigger evidence would not even raise the profile question they are meant to set;
- a few `app_differs` rows predict the app wrongly.

Every one of these can be fixed in place.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T11-dual-occupancy-townhouses/shape.yaml`
→ `ok`, exit 0, no warnings.

Counts: 15 triggers, 11 packages, 40 cost lines, 92 activities, 10 RFPs, 53 work items and 23 gaps.

## 2. Rulings on the author's open questions

| # | Question | Ruling | Finding |
|---|---|---|---|
| Q1 | A cleared lot as the base | **Wrong. Demolition goes in the base.** In middle-ring Sydney R2 streets a dual occupancy nearly always replaces a house. A vacant lot is the variant. | F1 |
| Q2 | OSD in the base | **Keep.** Most Greater Sydney councils' stormwater policies require OSD for a dual occupancy with subdivision. Some eastern and inner councils do not. The base documents must say it (the shape's `scenario_inputs` already ask for council policy wording). [D] | — |
| Q3 | Housing Provisions part numbers | **Part 9.3 "Fire protection of separating walls and floors" and Part 10.7 "Sound insulation" are right.** The map's detail under them is partly wrong (see F2). Source: ABCB NCC 2022 Housing Provisions online, [Part 9.3](https://www.abcb.gov.au/editions/ncc-2022/adopted/housing-provisions/9-fire-safety/part-93-fire-protection-separating-walls-and-floors), [Part 10.7](https://www.abcb.gov.au/editions/ncc-2022/adopted/housing-provisions/10-health-and-amenity/part-107-sound-insulation). I read the clauses through a fetch that summarised them, so the owner should open the page before marking anything `verified: true`. | F2 |
| Q4a | One BASIX certificate for both dwellings | **Keep, sourced.** For two new dwellings, the Planning Portal tells the user to start one project with "more than 1 dwelling", entering shared features as common areas and central systems. NatHERS can be done per dwelling. Source: [Planning Portal BASIX, dual occupancy](https://www.planningportal.nsw.gov.au/basix-project-details/project-type/dual-occupancy). The page does not say outright that one certificate issues, so keep "one certificate" as [D]. | F6 |
| Q4b | One HBCF certificate per dwelling | **Unconfirmed. Keep `verified: false`.** R4 governs the threshold and timing. It says nothing about per-dwelling certificates, so DOMAIN §1 must not cite R4 for that. | F6 |
| Q5 | OC before the subdivision certificate | **It is a consent condition, not a statute.** Most councils impose it for a dual occupancy with concurrent Torrens subdivision. Keep the link, tagged [D], and label it "where the consent requires it". Who issues it: some councils keep the Torrens subdivision certificate for themselves (Campbelltown: "Council only"). Base: council. A registered certifier stays `verified: false`. | F5 |
| Q6a | Housing and Productivity Contribution | **It applies.** A dual occupancy is medium-density residential development under the Ministerial Planning Order. The combined dual-occupancy and Torrens-subdivision case has its own worked inputs. The credit is for the existing lot, not the demolished house. Source: DPHI, [HPC Applicant Guide to Dual Occupancies, Dec 2025](https://planning.nsw.gov.au/sites/default/files/2025-12/applicant-guide-to-dual-occupancies.pdf) §4.1, §4.3. The shape must carry the line without an amount. | F3 |
| Q6b | Sydney Water developer charges | **They apply.** Sydney Water's "infrastructure contributions" were reintroduced from 1 July 2024: 25% in 2024–25, 50% in 2025–26, and the full price from 1 July 2026. They are charged per equivalent tenement through the Notice of Requirements. Source: [Sydney Water, land development](https://www.sydneywater.com.au/plumbing-building-developing/developing/land-development.html); [infrastructure contributions engagement page](https://sydneywatertalk.engagementhq.com/infrastructure-contributions). | F3 |
| Q6c | 2024–25 low-rise reforms | **The map's wording is wrong.** From 1 July 2024, dual occupancies became permissible with consent in **every R2 zone across NSW**, not "much of Greater Sydney". The Housing SEPP non-discretionary standards (450 m² lot, 12 m width, 9.5 m height, 0.65:1 FSR) stay `verified: false`. Commentators disagree on how they interact with LEP minimum lot sizes. Source: [Hornsby Council duty planner page](https://hornsby.nsw.gov.au/property/build/duty-planner/duty-planner-form/dual-occupancyduplex); [Mills Oakley](https://millsoakley.com.au/?p=5207). | F4 |

## 3. Findings

### F1. The base should include demolition of the existing house. Severity: high

- **Where:**
  - `shape.yaml`: header comment, `profile`, `work_items`, `procurement`, `cost_plan`, `program.activities`,
    `rfps[*].expect`, `triggers[existing-dwelling-demolition]` and `variants`;
  - `DOMAIN.md`: §1 table and "Why a cleared lot", §3, §4, §6, §7, §9 and §12.
- **What is wrong:** The brief says the base is the most common version of the job. In Sydney a dual
  occupancy is overwhelmingly a knock-down: a 1950s–70s house on a 600–900 m² R2 lot is replaced by two
  dwellings [D]. The author agrees (§1, §12). Avoiding overlap with T10 is a reason to test differently, not a
  reason to change the base. The demolition content also differs here from T10:
  - the HPC credits the existing lot;
  - Sydney Water's guide (§4.7) lets an existing junction serve only a *retained* home, so both lots get new
    connections;
  - two crossings replace one.
- **Exact change:**
  1. `profile`:
     - Follow T10's convention for `det.existing_building` (T10 sets it `true`, which turns on the
       `existing_building` RFP module). Add `det.existing_building_year: <year>`.
     - Copy T10's two `scenario_inputs` rows. The documents must say "existing building", and "built in
       <year>"; "1950s" is not read (`knowledge/determinants.yaml existing_building, existing_building_year`).
  2. `work_items`: add `{system: site.demolition, action: remove, why: "Demolish the house, garage, slabs,
     paths and driveway; implied_by strip-out gives action new, so fr.demolition-1a fires only on a remove
     reading"}`.
  3. `procurement`: add `consultant.hazmat` (`need: always`, `built_in: "no"`).
  4. `cost_plan`: add these lines.
     - "Demolition and hazardous materials removal": `partial`, source `knowledge/costs/elements.yaml code 01`,
       as in T10.
     - "Hazmat survey (consultant fee)": `no`.
  5. `program.activities`: add these rows.
     - "Hazardous materials survey" (investigation, `no`).
     - "Disconnect services (Sydney Water, DNSP, NBN)" (construction, `no`).
     - "Isolate, demolish and clear" (`fr.demolition-1a`, `partial`), placed after "Install erosion and
       sediment controls" and before "Cut, fill and trim levels" (`fr.earthworks-1a` attaches
       `after: fr.demolition-1a`).
     - "Asbestos clearance certificate" (`no`).
  6. `rfps[*].expect`: re-derive the rows the existing-building module changes, using T10's blocks. These
     include civil `retained_services` and structural `existing_assessment`.
  7. Triggers:
     - Replace `existing-dwelling-demolition` with `vacant-lot`. Evidence: "The site is vacant"; "the previous
       house was demolished under a separate consent". Negation: "the existing dwelling built in 1958 will be
       demolished".
     - `removes` lists the hazmat package, the demolition cost lines and the four program rows.
     - Put `vacant-lot` in `variants`.
  8. `DOMAIN.md`: rewrite §3 (asbestos, disconnections, the existing driveway crossing, dilapidation only if
     there is excavation near the boundary) and the §1 rationale.

### F2. Separating-wall rules are partly wrong. Severity: high

- **Where:**
  - `DOMAIN.md` §1 item 4, §2 `envelope` ("the roof covering and the battens must not bridge the separating
    wall") and the §2 `fire-passive.compartments` row ("eaves, gutters and fascias kept separate");
  - `shape.yaml work_items[fire-passive.compartments].why` and `rfps[architect].must_include`.
- **What is wrong:** Housing Provisions 9.3.1 (ABCB online edition, read 30 Oct 2026):
  - FRL 60/60/60, or masonry at least 90 mm thick;
  - from the footings or ground slab;
  - to the underside of a **non-combustible** roof covering, or 450 mm above a combustible one;
  - not crossed by combustible elements **except roof battens of 75 × 50 mm or smaller and roof sarking**;
  - eaves and verandahs open to the roof space and shared by more than one Class 1 dwelling need a
    non-combustible vertical lining.

  Gutters and fascias are not named. Housing Provisions 10.7.1: Rw + Ctr ≥ 50. Discontinuous construction is
  needed where a bathroom, sanitary compartment, laundry or kitchen adjoins a habitable room **other than a
  kitchen** next door. The wall runs to the underside of the roof or to a ceiling with the required rating.
- **Exact change:**
  - Rewrite the §2 envelope sentence and the §1 item 4 defect list to the rules above. The defect is
    "combustible framing, fascia or eaves members cross the wall, or a shared eave is not lined".
  - In `work_items[fire-passive.compartments].why`, say "to the underside of a non-combustible roof covering;
    only small battens and sarking may cross; shared eaves lined".
  - In `work_items[interiors.acoustic-separation].why`, add "(adjoining habitable room other than a kitchen)".
  - Keep every clause and value `verified: false` until the owner checks the cited ABCB pages. The two part
    numbers are confirmed (Q3).

### F3. The HPC and Sydney Water contributions apply; they are not "may apply". Severity: high

- **Where:**
  - `shape.yaml cost_plan` "Housing and Productivity Contribution" and "Sydney Water Section 73 application
    fees and developer charges";
  - `DOMAIN.md` §5 ("possibly the Housing and Productivity Contribution", "developer charges may apply") and
    §12.
- **Evidence:** Q6a and Q6b above.
- **Exact change:**
  1. In the HPC line's `why`, write: "State contribution for medium-density residential development and
     subdivision in Greater Sydney; credit for the existing lot (DPHI HPC Applicant Guide to Dual
     Occupancies, Dec 2025 §4.3); amount unknown".
  2. Rename the Sydney Water line to "Sydney Water Section 73 fees and infrastructure contributions (per
     equivalent tenement, set in the Notice of Requirements)". Its `why` gives the phase-in and the source.
     Keep both lines `built_in: "no"` and give no amounts.
  3. Delete "possibly" and "may apply" in §5, and move both items out of §12 "least sure".
  4. Gap 6 and `gaps[7]` already list them. No change there.

### F4. Planning permissibility and lot-size standards are misstated. Severity: medium

- **Where:**
  - `DOMAIN.md` §5 "Permissibility and controls";
  - `shape.yaml triggers[cl46-lot-size].evidence[1]` and `.why`.
- **What is wrong:**
  - The permissibility statement is too narrow (see Q6c).
  - The second cl46 evidence phrase ("frontage of 14.2 m is less than the 15 m required") does not match the
    `cl46` flag triggers (`clause 4\.6`, `cl 4.6`, "variation to a development standard";
    `knowledge/profile/flag_readings.yaml cl46`). The trigger's own `why` also says a DCP frontage needs no
    clause 4.6 request. So the phrase is positive evidence for something the key says does not follow.
- **Exact change:**
  1. Rewrite §5: "From 1 July 2024 dual occupancies are permissible with consent in all R2 zones in NSW (Housing
     SEPP). The SEPP sets non-discretionary standards (verified: false). The LEP may set its own dual
     occupancy and subdivision minimum lot sizes. Some LEPs restrict Torrens subdivision of a new dual
     occupancy, for example Randwick's 5-year rule after the OC (verified: false)."
  2. Replace `cl46` evidence[1] with "The LEP's 15 m minimum lot width for a dual occupancy is not met (14.2 m);
     a clause 4.6 request to vary the development standard accompanies the DA".
  3. Add "The frontage is 14.2 m against the DCP's 15 m control; a merit justification is given" to `negation`.
     It tests the map's own distinction.

### F5. The subdivision chain is missing steps, and Section 73 timing is overstated. Severity: medium

- **Where:**
  - `shape.yaml program.activities` (completion);
  - `rfps[certifier].must_include`;
  - `DOMAIN.md` §5 certificates table ("Here the app's rule is right") and §7.
- **What is wrong:**
  1. Councils commonly require these before the subdivision certificate [D, verified: false]:
     - the DNSP's notification of arrangement (both lots supplied);
     - telecommunications (NBN) provisioning evidence;
     - a survey certificate that each lot wholly contains its own services.

     Sydney Water requires each Torrens lot's private water and sewer lines to stay inside that lot, or to run
     in an easement and conduit where the lot is land-locked. Source: [Sydney Water, Servicing Dual Occupancy
     guide](https://www.sydneywater.com.au/content/dam/sydneywater/documents/servicing-dual-occupancy-other-two-home-developments-guide.pdf)
     §4.4, §4.5.
  2. `rule.nsw.section-73-certificate` says the certificate comes "before the occupation certificate"
     (`knowledge/clusters/services-wet-air/rules.yaml`). The map makes the subdivision certificate the gate.
     Both happen in practice, by consent condition. So "the app's rule is right" is overstated.
  3. The certifier option "subdivision certificate where the certifier is accredited" ignores councils that
     issue Torrens subdivision certificates only themselves (Q5).
- **Exact change:**
  1. Add three activities after "Lead-in connections", each with `built_in: "no"`, all feeding "Subdivision
     certificate":
     - "DNSP notification of arrangement for both lots";
     - "Telecommunications provisioning evidence for both lots";
     - "Survey certificate: services wholly within each lot".
  2. On "Subdivision certificate", set `why` to "Council for the base; OC first where the consent requires it
     (verified: false)".
  3. Set the certifier option to "Option: subdivision certificate, only where council permits a registered
     certifier to issue it (verified: false)".
  4. In §5, write: "The rule's OC timing and the map's subdivision-certificate gate are both consent-dependent.
     The test checks only that a Section 73 step exists".

### F6. BASIX and HBCF wording. Severity: low

- **Where:**
  - `DOMAIN.md` §1 ("takes out HBCF cover for each dwelling (R4)") and §5 certificates table;
  - `shape.yaml expected_derived[0].note`.
- **Exact change:**
  - In §1, drop "(R4)" from the per-dwelling clause and write "one certificate per dwelling (verified:
    false)".
  - For BASIX, write "one BASIX project for both dwellings (the tool's 'more than 1 dwelling' path), NatHERS
    per dwelling" and cite the Portal page. Drop the "(verified: false)" after the tool path, because it is a
    Portal fact, not an instrument clause.

### F7. Stairs and barriers lines get no budget share either. Severity: medium

- **Where:**
  - `shape.yaml cost_plan` "Stairs, ramps, handrails and nosings" and "Compliant balustrades and barriers";
  - `gaps[6]`;
  - `DOMAIN.md` §6 and §11 gap 4.
- **Evidence:**
  - `access-egress` is `itemised` (`knowledge/costs/elements.yaml`), so its group line is non-posting and the
    children carry a parent (`internal/costs/suggest.go` lines 258–276).
  - `spreadConstructionBudget` writes only to lines whose `SourceKey == "works:"+el.System && ParentItemID == ""
    && Posting` (`internal/store/cost_budget.go` lines 131–139).
  - So the 0.5 `access-egress` share in `residential_house_new` lands nowhere, and both child lines stay
    unpriced next to the unpriced 06 line.
- **Exact change:**
  - Keep both lines `built_in: "yes"`, since the lines do appear. Add to each `why`: "a stated budget leaves it
    unpriced (itemised group; spread skips child lines)".
  - Extend `gaps[6]` to "06 Passive fire, 13.3 and 13.9 get no budget share".
  - In DOMAIN §6, write: "Passive fire, stairs and barriers get no money".

### F8. Trigger evidence, `removes` and `built_in` errors. Severity: medium

1. **`trees-affected.evidence[1]`** ("the rear dwelling's slab edge is within the structural root zone of the
   neighbour's gum") matches neither `arborist_tree_impacts` nor `arborist_offsite_trees`. There is no impact
   verb after "root", and no "tree" after "neighbour" (`flag_readings.yaml`). **Change:** "excavation for the
   rear dwelling's footings is within the structural root zone of the neighbour's tree".
2. **`noise-affected-site.evidence[0]` and `[2]`** do not match the `noise_affected_site` triggers
   (`(road|rail|aircraft|traffic) noise`, `anef`; `knowledge/determinants.yaml`). **Change:** "The site fronts a
   classified road; road traffic noise requires an acoustic report" and "next to the rail corridor; rail noise
   assessment required".
3. **`existing-dwelling-demolition`** (if F1 is not adopted):
   - `built_in.work_items: "no"` is wrong. `implied_by strip-out` (`\bdemoli(?:tion|sh)\b`) adds `site.demolition`
     with action `new`, so set `"partial"`.
   - "knocked down" and "removed" match no pattern, and "1950s" sets no year.
   - `profile` lacks `det.existing_building`.
4. **`cdc-low-rise-code`:**
   - (a) `removes.cost_plan` "Construction certificate and certifier fees" is wrong. `cc-fees` is
     `applies_when: true` (`knowledge/costs/starter_lines.yaml`), and the certifier is still paid for the CDC.
     Keep the line, and drop "CDC fee" from `adds` as a duplicate.
   - (b) The skeleton changes to `sk.new-traditional-cdc`, which has no `construction_documentation` stage
     (KNOWLEDGE_INDEX §4.2), yet a CDC needs construction-level documents before lodgement. Note it in `why`,
     and add the gap that T01 review F8(d) added.
   - (c) The negation "heritage conservation area, so complying development is not available" is positive
     evidence for `heritage-conservation-area` (T01 review F8(c)). Replace it with "the design exceeds the
     code's height standard, so a DA will be lodged".
5. **`front-rear-shared-driveway`:**
   - `adds.work_items: site.vehicle-access new` is already a base item, so it is a zero delta. Remove it and
     drop `built_in.work_items`.
   - `removes.cost_plan` deletes the s138 line, but one crossing still needs s138 consent. Keep the line, and
     note "one crossing".
   - Add to `adds.authority`: "rear lot's private water and sewer lines in an easement for services, water line
     in a protective conduit (Sydney Water dual occupancy guide §4.5)".
6. **`sewer-main-in-lot.adds.rfps[architect]`** "Water authority asset submission": `architecture.yaml water_asset`
   is `O` when `authority_asset`, so write "Option: water authority asset submission".

### F9. Three `app_differs` blocks mispredict the app. Severity: low

- **Evidence:** `resolveStages` (`internal/rfp/resolve.go` lines 697–704) prices an *unknown* stage after the
  boundary end as an **option**, not "ask".
- **Change:**
  - Landscape (boundary `s1_inception … s3_approval`): `s4_developed`, `s5_tender` and `s7_construction` come out
    `option`, which equals `expect`. Delete those three `app_differs` rows.
  - Certifier (boundary `a1…a3`): `a4_construction` is `option` against `expect` absent. Add
    `{key: stage.a4_construction, app: option, expect: absent, why: "after the a1..a3 boundary"}`.
  - Geotechnical (boundary `g1…g3`): `g4_planning` is `option` against `expect` absent. Add
    `{key: stage.g4_planning, app: option, expect: absent, why: "client flag geotech_planning unknown, after the
    boundary"}`.

### F10. Keyword traps: one wrong, three missing. Severity: low

- **Where:** `DOMAIN.md` §10.
- **Exact change:**
  1. Trap 4 is wrong to say "Lot 2" counts. The pattern `\b\d+\s+(dwellings|lots|homes)\b` matches "2 lots", not
     "Lot 2".
  2. Add a trap: "Brick veneer" does not match `implied_by masonry` (`masonry|brickwork|re-pointing`). Without
     "brickwork" in a document, `structure.masonry` and `fr.masonry-1a` never arrive.
  3. Add a trap: `flag.multi_building` triggers on `\bbuildings? [a-d]\b`, so "building a dual occupancy" raises
     the question (if read case-insensitively). A "yes" turns on the architect's `brief_masterplan`.
  4. Add a trap: the street sewer main, the Section 73 text and "sewer main extension" all match
     `authority_asset`, so the base documents will raise that question. The answer must stay no.

### F11. Common triggers missing. Severity: low

Add at least the first one. Record the others under "Left out of the key" with a reason.

- `no-subdivision` (kept on one title for rent):
  - removes the final plan, the 88B instrument, the subdivision certificate, LRS registration, the solicitor
    and the subdivision fee line;
  - Sydney Water still wants two connections if a later subdivision is intended (guide §4.2).
- `lep-limits-subdivision`: the LEP bars or defers Torrens subdivision of a new dual occupancy (F4).
- `sloping-lot-retaining`: `site.retaining-walls`, structural retaining design and civil cut and fill. It is
  common on Sydney's north and south.
- Townhouses over a shared basement: adding Class 7a makes the project **industrial** through `det.ncc_class`,
  so `baselines[1]` brings Traffic, BCA, Access, Mechanical and Fire Services (KNOWLEDGE_INDEX §2.1). It is a
  strong keyword trap for the `three-plus-townhouses` family.

No owner ruling is broken. R1 (civil in the base), R3 (DBP Act not for Class 1), R4 ($20,000 including GST),
R6 (livable housing out), R7 (all eight inspection items) and R8 (DLP length `verified: false`) are all
followed. The one misattribution to R4 is covered in F6. There are no prices, rates or week durations.

## 4. Built-in claims checked

✔ = the claim holds against the cited file and the selection logic. ✘ = wrong; the finding gives the change.

**Scope and profile**

- ✔ 45 townhouses/new defaults, including `interiors.acoustic-separation`, `fire-passive.rated-elements`,
  `access-egress.stairs-and-ramps` and `.barriers` (`scope_defaults.yaml classes[1]`).
- ✔ `implied_by party-wall` adds `site.adjoining-structures`, `fire-passive.compartments` and
  `interiors.acoustic-separation`.
- ✔ `implied_by masonry` is the only route to `structure.masonry`.
- ✔ Subclass label "Townhouses (Class 1a)", `ncc_class: 1a`, no storeys field, dwellings regex needs a digit
  (`taxonomy.yaml`).
- ✔ `ncc_class` needs "Class 1a" (`determinants.yaml`).
- ✔ `staged` and `multi_building` patterns (`flag_readings.yaml`).

**Procurement**

- ✔ Architect, Surveyor, Geotechnical, Structural, Civil and Certifier: `baselines[0]`.
- ✔ Town Planner: `complexity_additions[0]` (also `baselines[0]`).
- ✔ `trade.main_works` `partial` (`ProposeHeadContract`, gaps only on accepted items).
- ✔ Not-expected Electrical and Hydraulic (`baselines[0]`) and Mechanical (`scope_additions[2]`).

**Cost plan**

- ✔ DA fees (`da-fees`, planning da), CC fees (`cc-fees`, always), service authority `partial`.
- ✔ Fee lines for the seven adopted packages (prepare adopts suggestions).
- ✔ Element lines 01–05, 08–11.
- ✔ 06 Passive fire `partial`: present, and no share in `residential_house_new`.
- ✔ 13.3 and 13.9 lines present. Their unpriced spread is a missing note; see F7.
- ✔ 07 Active fire as not expected (smoke alarms).
- ✔ Contingency, escalation and exclusions.
- ✔ Owner supply `partial` (`applyBriefFigures`).

**Program**

- ✔ Skeleton `sk.new-traditional-da`.
- ✔ Every process fragment marked `yes`: concept, design development, approval docs, DA lodgement, exhibition
  (labelled "Public exhibition"), assessment, construction docs, tender ×3, award, site establishment,
  erosion, PC, OC, handover, defects ×2.
- ✔ Every 1a work-item fragment marked `yes`: earthworks, footings, slab, termite, frame, roof, roof drainage,
  windows, external walls, sanitary, AC (attached after linings), insulation, linings, ceilings, doors, wet
  area, tiling, joinery, electrical, painting, floors, stormwater, landscape, services, utility.
- ✔ `fr.masonry-1a`, `fr.temporary-power`, `fr.investigate-services`, `-survey` and `-geotech` are `partial`
  (the item or action is missing).
- ✔ `fr.construction-certificate` `partial` (stage order).
- ✔ `if.frame-inspected-before-roof` fires.
- ✔ `if.reinforcement-inspected-before-pour` `partial` (needs `structure.concrete`).
- ✔ `if.underground-services-before-paving` correctly not counted (no `site.vehicle-access` rows).

**RFPs**

- ✔ Architecture `partial`: house module, `house_documents` boundary, bidding becomes an option, budget block.
- ✔ Structural `partial`.
- ✔ Civil `partial`: `osd_new` I, `as_built` follows `construction_phase`.
- ✔ Geotechnical `partial`, except one missing differ (F9 ✘).
- ✔ Surveyor `partial`: `titling` O always, s5 and s7 options.
- ✔ Certifier `partial`: falls back to a1–a3, except one missing differ (F9 ✘).
- ✘ Landscape `partial`: three wrong differs (F9).
- ✔ Town planning `partial`.
- ✔ QS `partial`.
- ✔ BASIX `no` template.

**Triggers**

- `cdc-low-rise-code`: ✔ program `yes` (`fr.cdc-application`), ✔ cost `partial`, ✔ procurement `no`,
  ✔ rfps `no`. ✘ `removes` CC-fees line (F8.4).
- `strata-subdivision`: ✔ rfps `partial`.
- `front-rear-shared-driveway`: ✔ rfps `partial` (`civil.yaml roads`). ✘ `work_items` `partial`: no delta (F8.5).
- `three-plus-townhouses`: ✔ procurement `no` (`baselines[1]` is non-residential), ✔ rfps `partial`.
- `detached-dual-occupancy`: ✔ rfps `partial` (`modules.yaml staged`), ✔ cost `no`.
- `existing-dwelling-demolition`: ✔ program `partial`, ✔ cost `partial`, ✔ procurement `no`.
  ✘ `work_items` `no` should be `partial` (F8.3).
- `sewer-main-in-lot`: ✔ rfps `yes` (`structural.yaml authority_asset`). ✘ the architect item is an option (F8.6).
- `trees-affected`: ✔ rfps `yes` (template; the package is never suggested, as in T01), ✔ program `partial`
  (`retain` brings no `fr.tree-protection`).
- `bushfire-prone`: ✔ procurement `yes` (`complexity_additions[5]` via `det.bal`), ✔ program `partial`,
  ✔ rfps `no` (no row tests bushfire), ✔ cost `partial`.
- `flood-affected`: ✔ rfps `yes` (`civil.yaml flood_report`), ✔ program `partial`.
- `heritage-conservation-area`: ✔ procurement and cost `yes` (`complexity_additions[1]`), ✔ program `partial`
  (`fr.heritage-referral`), ✔ rfps `partial`.
- `cl46-lot-size`: ✔ rfps `yes` (`variation_request`), ✔ program `no`.

**Knowledge anchors cited as [K]**

✔ `if.party-wall-roof-continuity` (boundary type), `fm.party-wall-stops-in-roof`,
`fm.bath-against-party-wall-without-isolation`, `uc.strata-or-subdivision-registration-delayed`,
`rule.nsw.section-73-certificate` (`clause_verified: false`), `rule.basix.certificate-required` (derivation
pending), `fire-passive.compartments` description (attached Class 1 party walls) and `site.survey-boundaries`
description (plans of subdivision).

**Spot checks of `no` claims (all ✔, the knowledge has nothing)**

1. No package rule names BASIX or Landscape (`package_defaults.yaml`).
2. No long-service-levy, Section 73, HPC or subdivision line (`starter_lines.yaml`).
3. No 1a stair fragment; only `fr.stairs-2` and `fr.stairs-9b` exist.
4. No fragment names subdivision, Section 73, HBCF or BASIX (`fragments.yaml`).
5. No fragment on `hydraulic.non-potable-water` (rainwater tanks).
6. No Traffic or Waste for a residential class (`baselines[1]`).
7. No acoustic consultant rule.
8. No WSC or solicitor discipline id (`data/intake/disciplines.json`).

## Round 1 response (author, 30 October 2026)

All eleven findings applied to `shape.yaml` and `DOMAIN.md`; none disputed.

- **F1** Demolition in the base, as T10: `det.existing_building: true`, `det.existing_building_year: 1958`, T10's scenario inputs plus the `det.ncc_class` phrasing, `site.demolition` remove, `consultant.hazmat` (always), the demolition, disconnection, latent-conditions and hazmat-fee cost lines, and the LFAI, hazmat survey, CCTV, demolition plan, abolishment, vacate, disconnect, SafeWork, removal, clearance, demolish and old-footings rows. The R7 pre-start inspection moved to approvals, before the CC. RFP `expect` blocks re-derived for the existing_building module: architecture `existing_interfaces` I, civil `retained_services` I, structural `existing_assessment` include and `strengthening` O as app_differs, surveyor `buried` I, hazmat RFP added. `existing-dwelling-demolition` replaced by `vacant-lot`, which is in `variants`. §1 and §3 rewritten.
- **F2** Separating-wall rules rewritten to 9.3.1 and 10.7.1 in §1, §2, the work items and the architect RFP; values stay verified: false.
- **F3** HPC and Sydney Water infrastructure contributions stated as applying, with sources; removed from "least sure".
- **F4** §5 permissibility rewritten; cl46 evidence[1] replaced; DCP-frontage negation added.
- **F5** DNSP, telecommunications and services-survey rows feed the subdivision certificate; the subdivision certificate `why` and certifier option reworded; §5 Section 73 wording corrected.
- **F6** HBCF per dwelling no longer cites R4; BASIX wording follows the Portal.
- **F7** Stairs and barriers marked unpriced by the spread; gap and §6 extended.
- **F8** trees and noise evidence fixed; cdc keeps the CC-fees line, drops "CDC fee", has a new negation, and gets a skeleton note and gap; front-rear loses the zero-delta work item and the s138 removal, and gains the §4.5 easement; the sewer-main architect item is an option. (F8.3 is moot after F1.)
- **F9** Three landscape differs deleted; certifier `a4_construction` and geotech `g4_planning` differs added.
- **F10** Trap 4 corrected; brick-veneer, "building a" and sewer-matches traps added, plus existing-building and basement traps.
- **F11** `no-subdivision` trigger added and in `variants`; the other three recorded under "Left out of the key" with reasons.
- Coordinator direction: the reinforcement hold point is now `built_in: yes` (fires through `fr.slab-on-ground-1a`, confirmed by a Go run).

## Round 2 (reviewer, 30 October 2026)

I re-read `shape.yaml` and `DOMAIN.md` after the author's round 1 response, and also
`testing/synthetic/reference/CONVENTIONS.md`. The validator still passes:
`python testing/synthetic/tools/validate_shape.py …/shape.yaml` → `ok`, exit 0.

Counts: 16 triggers, 12 packages, 44 cost lines, 107 activities and 11 RFPs. Every program `after` label
resolves to an activity in the file. The files are UTF-8 with no replacement characters.

### Verdict: **FIX** (three small changes, all in the variant keys)

The base is now right, and every round 1 finding is applied correctly. What remains would make two variant
tests fail for the wrong reason.

### Round 1 fixes checked

| Finding | Status | Note |
|---|---|---|
| F1 demolition in the base | ✔ | The base matches T10: `det.existing_building: true`, year 1958, the scenario-input phrasings, `site.demolition` remove, hazmat (always, `no`), demolition and asbestos rows, and `fr.demolition-1a` `partial` feeding `fr.earthworks-1a`. The existing-building RFP rows have been re-derived: architecture `existing_interfaces` I, civil `retained_services` I, and structural `existing_assessment` as an app difference. They match `rfp/architecture.yaml`, `civil.yaml` and `structural.yaml`. |
| F2 separating wall | ✔ | 9.3.1 (battens and sarking, non-combustible roof covering or 450 mm, shared eaves) and 10.7.1 (Rw + Ctr 50, kitchen exception) are in §1, §2, the work items and the architect RFP. All are still `verified: false`. |
| F3 HPC and Sydney Water | ✔ | Both are stated as applying, with sources, and no amounts. |
| F4 permissibility and cl46 | ✔ | The new cl46 evidence matches `clause 4\.6`. The DCP-frontage negation is in. |
| F5 subdivision chain | ✔ | The three new rows feed "Subdivision certificate". The certifier option and the §5 Section 73 wording are corrected. |
| F6 BASIX and HBCF | ✔ | |
| F7 stairs and barriers budget | ✔ | Cost-line `why` and the gap are both extended. |
| F8 triggers | ✔ | The trees, noise and cl46 phrasings now match their trigger regexes, and the round 1 regex misses are fixed. The cdc trigger keeps CC fees and has the skeleton note and a new negation. The front-rear trigger is fixed. The sewer-main architect item is now an option. |
| F9 RFP app_differs | ✔ | The landscape rows are deleted. Certifier `a4_construction` and geotechnical `g4_planning` are added. |
| F10 keyword traps | ✔ | |
| F11 missing triggers | ✔ | `no-subdivision` is added. The other three are under "Left out", with reasons. |
| Reinforcement hold point → `yes` | ✔ | My round 1 tick on `partial` was wrong. The "pour" activity of `fr.slab-on-ground-1a` is tagged `systems: [substructure.slab-on-ground, structure.concrete]` (`knowledge/program/fragments.yaml` line 208). That gives `if.reinforcement-inspected-before-pour` both its tagged rows. |
| CONVENTIONS C2 | ✔ | BASIX work uses `consultant.basix`, and no ESD consultant is listed. C1 and C3 do not apply: the route is traditional, and only the certifier uses `bca-certification`. |

### Round 2 findings

**R2-1. `triggers[vacant-lot]`: the evidence does not set `det.existing_building`, and `removes` is incomplete.
Severity: medium.**

- **Evidence wording.** The determinant's only trigger is `\bexisting building\b` (`knowledge/determinants.yaml`).
  "The site is vacant" and "the previous house was demolished" never raise it, so it stays unknown, not
  false. The trigger's `built_in.rfps: "yes"` depends on it being false: the module turns off, and the
  `existing_interfaces`, `retained_services` and `existing_assessment` rows leave. With the value unknown,
  the variant would test the wrong state.
  - **Change:** make `evidence[0]` "The site is vacant: there is no existing building on the lot".
- **Incomplete `removes.program`.** It removes four rows, but the base has eleven demolition-only rows.
  - **Change:** also remove "Loose-fill asbestos register search", "Demolition plan, demolition work plan and
    waste plan", "Request meter removal and service abolishment", "Occupants vacate the house", "Disconnect
    and abolish services; cap the old sewer junction", "SafeWork NSW asbestos removal notification" and
    "Remove old footings, slabs and buried drainage; record waste disposal".
  - In `removes.cost_plan`, add "Provisional sum for latent ground conditions after demolition", or rename it
    so it still applies to a vacant lot's fill.

**R2-2. `triggers[no-subdivision].removes.program` leaves the new subdivision-certificate inputs behind.
Severity: low.**

- **What is wrong:** the three rows added under F5 exist only to feed the subdivision certificate, and this
  trigger removes that certificate.
- **Change:** add these to `removes.program`:
  - "DNSP notification of arrangement for both lots";
  - "Telecommunications provisioning evidence for both lots";
  - "Survey certificate: services wholly within each lot".

  Also state in `why` that the certifier's subdivision-certificate option and the surveyor's `titling`,
  `s5_support` and `s7_completion` expectations go back to absent or option.

**R2-3. `gaps` line 769 is stale. Severity: low.**

- **What is wrong:** "critical-stage inspections other than the frame hold point" now contradicts the
  reinforcement hold point being `yes`.
- **Change:** "… other than the frame and reinforcement hold points".

No owner ruling is broken. There are no prices, rates or week durations. Once these three edits are made
the map is usable as an answer key, and I would expect a round 3 to give a **PASS** without further review.

## Round 2 response (author, 30 October 2026)

- **R2-1** applied: `vacant-lot` evidence[0] now reads "The site is vacant: there is no existing building on the lot". `removes.program` lists all eleven demolition-only rows. `removes.cost_plan` adds the latent-conditions provisional sum.
- **R2-2** applied: `no-subdivision` also removes the DNSP, telecommunications and services-survey rows. Its `why` names the certifier option, surveyor `titling` (back to absent), and `s5_support` and `s7_completion` (back to option).
- **R2-3** applied: the gap reads "other than the frame and reinforcement hold points".

The validator still passes.
