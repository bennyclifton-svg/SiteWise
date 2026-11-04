# Review: ST09 scenarios (base, podium-over-basement, hazardous-materials)

Reviewer: independent (brief 03r), 2026-11-04.

## Verdict: **FIX**

Nine findings. Finding 1 is the most serious. No document in any of the three scenarios states the
unit count in a form the app can read, so `hdr.scale.units` = 42 is a key the app cannot meet.
Findings 2 to 4 are profile values the app cannot read. The notes do not declare them as missed,
or declare one only as a "may".

## Checks

1. **Checker.** `validate_expected.py` reports `ok` for ST09-base, ST09-hazardous-materials and
   ST09-podium-over-basement.
2. **Faithful to the shape.** The base matches the shape's base job: 6 storeys, 42 lots, Class 2,
   built 2008, out of warranty, exempt, AS 4000 lump sum, DBP Act applies. Every trigger is denied
   in the shape's own negation wording or is absent. That covers hazmat (the 2019 survey), spalling,
   door thresholds (adequate set-down), warranty, heritage, access equipment, internal damage,
   balcony services, HBCF, roof terrace and podium ("no podium ... separate at-grade car park").
   The §10 traps are all placed, and their expected outcomes are right for the base: remediation,
   flood test, bathroom, bedroom, fire stairs, strip-out, fee stages, schedule of rates and the RFP
   cue. See finding 9 for one trap that is missing.
3. **Register.** The numbers, revisions, titles and dates agree with DISCOVERY.md §4.3. The labels
   used are `Ref`, `Proposal no`, `Quote no`, `Report no`, `Document number`, `Drawing number`,
   `Revision`, `Report date` and `Date`, and each document has a single `#` heading identical to
   its `Title:`. d03 and d11 have no heading, so `Title:` alone settles their titles. The d02 and
   d06 tables are declared missed for number and date, and they keep the true values. That d12
   supersedes d11 is declared as unsupported. d04 is a proper email, so its title is the Subject.
4. **Profile.** See findings 1 to 6. Every other value is caught by a real trigger:
   - `existing_building_year`: "constructed in 2008" matches `\b(?:built|constructed) in \d{4}\b`.
   - `ncc_class`: "NCC building classification: Class 2" matches the second trigger, and
     `classRe` yields 2.
   - `rise_in_storeys`: matched by "rise in storeys".
   - `existing_building`: matched by "existing building".
   - `heritage_status`, `flood_hazard` and `potentially_contaminated_land`: matched in d02.
   - `contract_form`: matched by `\bas\s?4000`.
   - LGA: `lgaCouncil` yields "Georges River".
   - Zone: `zoneRe` yields "R4 High Density Residential".
   - The flags: `staged` ("no staged delivery" matches `staged (?:…delivery…)`),
     `excavation_adjoining`, `transfer_structures` ("transfer slabs"), `authority_asset` ("sewer
     main"), `rail` ("rail corridor"), `osd_existing` ("existing on-site detention") and
     `basement`.

   `read_by_design` is false only for d04, the email, and that is right.
5. **Fees and invoices.**
   - d06: each GST-inclusive figure divided by 1.1 is right: 29,700 → 27,000, 22,000 → 20,000,
     8,800 → 8,000 and 38,500 → 35,000, for a total of $90,000 ex GST. `gst_inc_normalised` is
     a real flag (`proposals/read.go` FlagGSTInc).
   - Stage sums: d05 = $81,000, d07 = $17,000, podium d17 = $9,400, podium d18 = $8,500 and
     hazmat d17 = $7,000.
   - Invoices: 10% GST and the totals agree.
   - ABNs: every ABN passes the checksum except 77 650 369 250 on d15. The ledger raises
     `abn_invalid` for it (`invoices/ledger.go:318`).
6. **Variants.**
   - hazardous-materials changes only `det.existing_building_year` (2008 → 1986). It adds the
     trigger's evidence (asbestos register, bituminous membrane and mastics) and d17/d18.
   - podium-over-basement changes no profile value. It adds the three evidence sentences
     verbatim, the podium scope, and d17 (landscape) and d18 (hydraulic). Fees are deliberately
     unchanged.
   - See findings 7 and 8.
7. **Realism.** The set is recognisable as a real Sydney strata remediation job:
   - Setting: Hurstville, in Georges River, zoned R4 under Georges River LEP 2021.
   - Documents: a strata manager's brief, a s10.7 certificate, a leak log, a defect report with
     options, a remedial specification and tender conditions.
   - Dates: February to July 2026, with a start in February 2027 once the levy is raised.
   - Cost: $1.68M ex GST for 42 full balcony renewals is about $40k a balcony including
     preliminaries, which is plausible.
   - Fees: the consultant at $81k or $90k (about 5%) and the structural engineer at $17k are
     plausible.

## On the consultant adaptation

**Right.** The brief's architect, civil and geotechnical defaults have no role on a like-for-like
membrane renewal. The shape (owner ruling R2) names exactly three base packages: the remedial
building consultant, the remedial structural engineer, and the remedial builder as
`trade.main_works`. Two competing `consultant.building` proposals still exercise fee comparison
within one package. They are a good pair: Parapet bills opening-up as a disbursement, Halyard
includes it, and Halyard prices GST-inclusive. Keep it.

## On the author's shape comments

1. **Phrasings.** Each was checked against the app's real triggers.
   - **"completed in <year>": the author is right.** The triggers are
     `\byear (?:built|of construction|constructed)\b` and `\b(?:built|constructed) in \d{4}\b`
     (`knowledge/determinants.yaml` existing_building_year), so "constructed in" is needed. The
     hazmat negation "Construction completed in 2012" both misses the trigger and contradicts the
     2008 base. Leaving it unused was correct.
   - **"Order of cost": the trigger fix is right but not enough.** "construction cost" and "cost of
     the works" do match `budget_stated`. But Jev is asked for "the construction budget or target
     construction cost", and a consultant's estimate is neither. See finding 5.
   - **"one continuous stage": the author is right.** It matches no `staged` trigger, while "no
     staged delivery" matches `staged (?:…|delivery|…)`.
   - **Shape phrasing to fix.** The shape's own units phrasing "<n> residential units" is also
     broken (finding 1), and the shape should be corrected.
2. **Podium and ncc_class/budget: agree.** A basement car park is a Class 7a part. A defect report
   describing it would say "Class 2 with a Class 7a car park", and a podium renewal with planters
   adds real cost. The shape should add `det.ncc_class` and `proj.budget_stated` to the podium
   trigger's `profile`. See finding 8. As written, the variant is consistent with the shape, so this
   is a shape change first.
3. **"(fictional)" in proj.client: agree.** The shape value should be
   "The Owners - Strata Plan No. 912345". The accept-list entry with "(fictional)" exists only to
   satisfy the checker's base-equals-shape test. Drop it after the shape is fixed. See also
   finding 2.
4. **regulated_design: confirmed, and it is certain, not "probably".** See finding 3.

## Findings

1. **All three scenarios, d01/d03/d08: `hdr.scale.units` = 42 cannot be read.**
   - **What is wrong.** Each document says "42 residential units". The trigger is
     `\b\d+\s+(units|apartments|dwellings)\b` (`knowledge/profile/taxonomy.yaml` units). The
     candidate pattern `unitCountRe` (`internal/profile/harvest.go`) allows only
     "sole occupancy" between the number and the noun. Neither matches "42 residential units"
     (checked). "42 lots", "42-unit" and "all 42 balconies" match nothing either.
   - **Evidence.** No passage in any of the three scenarios yields the count.
   - **Change.** In all three manifests (and the documents), make the `hdr.scale.units` `says`
     read "42 apartments":
     - d01: "Kurrajong Apartments is a 6-storey residential flat building of 42 apartments".
     - d03: "42 apartments in a 6-storey building".
     - d08: "a 6-storey residential flat building of 42 apartments".

     Apply the same wording to the `hdr.building_class` and `hdr.subclass` `says`, and the podium
     d01 variant. Keep the key at 42. Tell the orchestrator that the shape's `scenario_inputs`
     phrasing for units must change too.

2. **All three scenarios: `proj.client` is cut to "The Owners".**
   - **What is wrong.** `clientCandidates` (`internal/profile/details.go`) takes words after
     "Client:" or "Principal:" until a lower-case or non-capital token. The " - " in "The Owners -
     Strata Plan No. 912345" ends the name, so the only candidate is "The Owners". Without the
     dash, the full stop in "No." would end it at "…Plan No" instead.
   - **Evidence.** The key accepts nothing the app can produce.
   - **Change.** Keep the documents: the SSMA legal name is the realistic form, and the truncation
     is a real defect worth catching. Add to "Missed by design" in all three notes.md: "proj.client:
     clientCandidates stops at ' - ', so the app offers only 'The Owners'; the key keeps the true
     name." After the shape fix in comment 3, remove the "(fictional)" accept entry.

3. **All three scenarios: `proj.regulated_design` is never read, and the notes hedge.**
   - **What is wrong.** `project_details.yaml` gives `regulated_design` no `extraction`, triggers
     or asks. `compileReading` (`internal/knowledge/details.go`) leaves such a detail unread, and
     `flag_readings.yaml` says "Regulated design needs professional advice". It also stays unknown
     rather than "no": `deriveRegulatedDesign` (`internal/rfp/drivers.go`) returns early when a
     Class 2 part is present.
   - **Evidence.** The checker forces `read_by_design: true` from the kinds of the sources.
   - **Change.** In all three notes.md, replace "the app may never ask for it ... Keyed as read by
     design" with: "Never read: the detail has no extraction (details.go compileReading). The app
     shows it unknown (not derived 'no', because the project has a Class 2 part) until the user
     records it. read_by_design is true only because the checker infers it from source kinds."

4. **All three scenarios, d02: `proj.lot` cannot be read.**
   - **What is wrong.** "Lots 1 to 42 in Strata Plan 912345" passes the trigger, but `lotRe`
     (`details.go`) has no range form: it accepts only `,`, "and" or "&" between lot numbers. It
     yields no candidate (checked). "(SP 912345)" has no "Lot" prefix.
   - **Change.** Add to "Missed by design" in all three notes.md: "proj.lot: lotRe has no lot
     range ('Lots 1 to 42'), so no candidate; the key keeps the true value." This is the realistic
     strata wording, so do not reword it.

5. **All three scenarios, d08: `proj.budget_stated` is offered as an estimate, not a budget.**
   - **What is wrong.** The passage matches "construction cost". But the question asks for "the
     construction budget or target construction cost", and the text is a consultant's
     "Order-of-cost estimate for Option B". Jev may select none. This also feeds
     `rfps.json` `budget_stated_cents`.
   - **Change.** Make the d08 `says` (all three): "We recommend the owners corporation adopts a
     construction budget of $1,680,000 excluding GST for Option B (construction cost of the works
     including preliminaries and provisional sums)." This is realistic, because a defect report
     recommends the levy budget. Keep the key at 168000000.

6. **All three scenarios, d01: `fact.contract_basis` source does not trigger.**
   - **What is wrong.** d01 says "lump-sum contract". The trigger is the literal `lump sum`
     (`project_facts.yaml` contract_basis), which the hyphen breaks. d13 ("Lump sum contract")
     still carries the value.
   - **Change.** Either make the d01 `says` "tendered to remedial builders for a lump sum
     contract", or remove d01 from the key's `sources`.

7. **ST09-hazardous-materials, d09 trap `strip-out`: `expect` is wrong for this variant.**
   - **What is wrong.** The expect still says "No site.demolition work item". The hazmat trigger
     adds `site.demolition` `remove` (licensed asbestos removal), and the variant's d09 now has a
     Hazardous materials section.
   - **Change.** Reword the expect to: "The strip-out wording alone does not add site.demolition;
     the only demolition item is the asbestos removal the trigger adds."

8. **ST09-podium-over-basement, d08: realism of the class and the cost (shape change first).**
   - **What is wrong.** d08 describes a single-level basement car park and recommends renewing the
     podium too. Yet it states "Class 2" only and keeps the $1,680,000 balcony-only cost.
   - **Change.** Once the shape adds `det.ncc_class` and `proj.budget_stated` to the trigger's
     `profile`:
     - Make d08's class line "NCC building classification: Class 2, with a Class 7a basement car
       park", and key `det.ncc_class` ["2", "7a"].
     - Raise the d08 budget (author's figure; the podium membrane, drainage cell, soil, planting
       and pavers add a few hundred thousand dollars). Re-key `proj.budget_stated` and
       `rfps.json` `budget_stated_cents`.

     Until then, leave the variant as is. Unchanged consultant fees are acceptable for a clean
     comparison. The hazmat variant has the same smaller issue: asbestos removal is known from d18
     (8 April) before d08 (14 April), yet the cost is unchanged. Note it in the shape comment.

9. **All three scenarios, d02: a missing trap for complying-development wording.**
   - **What is wrong.** The content lists the s10.7 item "complying and exempt development". A
     certificate's standard text ("Complying development may be carried out on the land under…")
     is read (certificate kind), and it may pull `hdr.cond.planning` towards cdc. ST01 keyed the
     same trap.
   - **Change.** Add a d02 trap `cdc-eligible-land` with the expect: "hdr.cond.planning stays
     exempt (d08, d09); land eligibility is not the works' pathway."
