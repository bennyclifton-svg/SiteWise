# Review: ST02 scenarios (base, groundwater-dewatering, traditional-procurement)

Reviewer: independent (brief 03r), 2026-11-06.

## Verdict: **FIX**

Five findings. Finding 1 settles the author's shape question. The shape should add `fact.contract_form` to the
`traditional-procurement` trigger. Then the base can key `as4902` exactly and the variant `as4000`. Finding 2 is a
value that d12 states and no key covers. The remaining findings are small: one contradiction in a read document,
one tier label, and one date that is too tight. Every other key checked out against the code.

## Checks

1. **Checker.** `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST02-*` reports
   `ok` for all three scenarios.
2. **Faithful to the shape.** The base matches the shape's base job: 6 storeys, 64 units, a 2,400 m² site, a GFA of
   6,200 m², Class 2 over 7a, sprinklered, an AS 4902 D&C with novation, a DA, and the Ryde LPP. Every trigger is
   denied in the shape's own negation wording or is absent, including transfer, groundwater, cl 4.6, heritage,
   flood, contamination, acid sulfate, trees, Sydney Water, rail, noise, gas, multi-building, facade and SSD.
   Each `scenario_inputs` keyword trap is in place with the base value stated beside it: existing building,
   multi_building, staged, authority_asset, noise and street trees. Declared and incidental traps are listed in
   notes.md. I found no undeclared phrase that fires a trigger and would change a base value. The only exception
   is d10's "anchors under neighbouring lots need the owners' consent", which fires `arborist_offsite_access`;
   that key is not in this profile, so nothing is lost.
3. **Register.** Each `Title:` matches its single `#` heading. These labels settle by rule (DISCOVERY.md §4.3):
   `Ref`, `Proposal no`, `Quote no`, `Proposal ref`, `Proposal number`, `Report no`, `Document number`,
   `Drawing number`, `Revision`, `Report date` and `Date`. d04 is a real email, so its title is the first
   Subject ("RE: …") and its date is the latest header (3 March). The d02, d07 and d12 table title blocks hold
   true values, and notes.md declares those values missed. Notes also declares that the d15→d14 supersession is
   unsupported. Invoices go to `invoices` and stay out of the register, which is correct.
4. **Profile.** I checked every flag `says` against `knowledge/profile/flag_readings.yaml`, and each one fires its
   trigger. Examples: "No staged construction", "not multiple buildings", "special fire hazards", "performance
   solution is proposed for the facade", "existing on-site detention", "MUSIC modelling" and "classified road".
   The determinants and details fire too:
   - ncc_class: "NCC building classification" plus `classRe`
   - rise: `riseRe` "Rise in storeys: 6"
   - built in 1962
   - facade: "rainscreen"
   - gas: "natural gas"
   - sewer and water: "sewer" and "water main"
   - lot: `lotRe` takes "Lots 11, 12 and 13 DP 236418"
   - LGA: `lgaCityOf`
   - zone: `zoneRe`
   - storeys: `storeyRe` reads "six"
   - units: `unitCountRe` "64 apartments"

   `read_by_design` is false only for the d04 email, which is right. notes.md declares `avg_unit_size_sqm` and
   `regulated_design` missed (the precedent from T05/T09).
5. **Fees and invoices.** The stage sums match each stated total:
   - Base: d05 $780k, d06 $156k, d08 $48k, d09 $72k.
   - Traditional: d05 $1.1M, d06 $309k.
   - d07 civil, GST-inclusive: every figure ÷ 1.1 is exact. The base total is $90,200 → $82,000 and the
     traditional total is $116,600 → $106,000. `gst_inc_normalised` is real (`proposals/read.go` FlagGSTInc).
   - Invoices: the d16 and d17 totals add up. In d18, $1,800 on $12,000 raises `gst_rate` and not `arithmetic`
     (`invoices/ledger.go` Check).
   - ABNs: all eight pass the checksum.
6. **Variants.**
   - **groundwater-dewatering** changes only `flag.groundwater`. The three evidence phrases sit in d10 verbatim,
     and the d11 passing mention ("drained cavity") is updated. See finding 4.
   - **traditional-procurement** changes `hdr.cond.procurement_route` and every novation to `no`. The evidence
     sits in d01 verbatim, and the D&C and novation mentions are removed from d01, d04, d05, d06 and d07. The
     variant also changes the contract form, which is outside the trigger's keys (finding 1).
7. **Realism.** This reads like a real West Ryde R4 amalgamation: three 1962 dwellings, a Housing SEPP Ch 4 DA to
   the Ryde LPP, a drained shale and sandstone basement, and an AS 4902 D&C with novation. $27.6M ex GST for
   6,200 m² GFA plus two basements (about $4,450/m² GFA) is plausible mid-range. The architect's fees are 2.8% to
   tender plus 1.9% post-novation, and the traditional variant runs 4.0% to construction. The structural, civil,
   geotech and planner fees are all plausible. See findings 3 and 5.

## On the author's shape question

**Yes: add `fact.contract_form` to the trigger.** The evidence itself says "lump-sum tender under AS 4000", and
`\bas\s?4000` is a `contract_form` trigger (`knowledge/profile/project_facts.yaml`). So the true value in the
variant is `as4000`, not the base's `as4902`, and that is a real profile change caused by the trigger. Under the
checker rule (a variant may differ only on the trigger's keys), the shape is what is wrong. Today's workaround
(`accept: [as4902, as4000]` in all three keys) means the base would score an app that reads AS 4000 from an AS 4902
brief as correct. T13's equivalent trigger already lists both keys (`types/T13-mixed-use-shop-top/shape.yaml`
line 872).

## Findings

1. **Shape plus all three `expected/profile.json`: `fact.contract_form` is loosened to two values.**
   - **Evidence.** See the section above. d01 (base) says "AS 4902 General conditions of contract for design and
     construct". The traditional d01 says "AS 4000-1997 General conditions of contract, lump sum", and the
     traditional d05 says "a lump-sum tender under AS 4000".
   - **Change.**
     - In `types/T02-apartments/shape.yaml`, `traditional-procurement`, line 782:
       `profile: [hdr.cond.procurement_route]` → `profile: [hdr.cond.procurement_route, fact.contract_form]`.
     - Then set `fact.contract_form.value`:
       - ST02-base: `"as4902"`, sources `["d01"]`.
       - ST02-groundwater-dewatering: `"as4902"`, sources `["d01"]`.
       - ST02-traditional-procurement: `"as4000"`, sources `["d01", "d05"]`.
     - Drop the "Shape comments" bullet in all three notes.md files.
2. **All three scenarios, d12: `det.effective_height` is stated but not keyed.**
   - **Evidence.** d12's `content` states "effective height about 18.6 m" in a BCA report, which is a read kind.
     `det.effective_height` has `profile_group: classification` and the trigger 'effective height'
     (`knowledge/determinants.yaml`). `effectiveHeightRe` (`internal/profile/harvest.go`) accepts "about 18.6 m".
     Brief 03 says to key "any other value your documents state". ST13 keys the same value. Without a key, a
     wrong reading goes unscored.
   - **Change.**
     - In each d12, add a carries entry `{"key": "det.effective_height", "says": "Effective height: about 18.6 m"}`.
     - In each profile.json, add `"det.effective_height": {"value": 18.6, "sources": ["d12"], "read_by_design": true}`.
     - (d12's "NCC 2022" also fires `ncc_edition`, but that determinant has no options or candidates, so leave it
       unkeyed.)
3. **All three scenarios, d01 `content`: "a cleared, amalgamated site" contradicts the same brief.**
   - **Evidence.** The same brief says the three 1962 dwellings stand and "will be demolished". d03 (24 February)
     surveys them standing. A read document that calls the site both cleared and built on invites a wrong
     `existing_building` or work-type reading, and a PM would query it.
   - **Change.** In all three d01 `content` and the `hdr.work_type` `says`, replace "This is a new building on a
     cleared, amalgamated site" with "This is a new building on an amalgamated site". The value `new` is
     unchanged.
4. **ST02-groundwater-dewatering, d10: the tier is `adversarial` but the trap list is empty.**
   - **Evidence.** The seepage trap was removed (rightly), but the tier was not updated. Brief 03 requires each
     adversarial document to carry a declared trap. The variant still has five other adversarial documents.
   - **Change.** Set d10 `"tier": "clean"` in that manifest.
5. **All three scenarios, d08 and d10: the six-week standpipe record does not fit the dates.**
   - **Evidence.** The proposal is dated 11 March and the report 24 April, six weeks and two days later. That
     leaves no time for acceptance, drilling and standpipe installation before six weeks of readings.
   - **Change.**
     - d08 `content`: "over six weeks" → "over four weeks".
     - Base and traditional d10 `flag.groundwater` `says`: "both standpipes stayed dry over six weeks" → "both
       standpipes stayed dry over four weeks".
     - Groundwater d10 `content`: "in both standpipes over six weeks" → "in both standpipes over four weeks".
     - No key changes.
