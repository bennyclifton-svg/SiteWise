# Review: ST06 scenarios (base, tank-needs-consent, pump-tank-only)

Reviewer: independent (brief 03r), 2026-11-07.

## Verdict: **FIX**

There are twelve findings. Two of them are serious:

- **Finding 1.** `proj.regulated_design` is keyed `read_by_design: true` from d09, but nothing in the app reads it.
  The RFP derives "no" from classes 5 and 7a. As keyed, every run scores a profile miss that is not an app fault.
- **Finding 4.** The stated budget covers the fire upgrade only ($1.2M), but the job includes the fit-out. The
  cost plan spreads `proj.budget_stated` over every construction line, and the RFPs print it as the construction
  budget. So the base tests a part-scope budget against a whole-scope plan, and the interior designer's RFP gets a
  budget that excludes its own works.

The register, fee and invoice keys are right. Each variant changes only its trigger. Three d12 sentences match no
profile trigger (findings 2, 3 and 12).

## Checks

1. **Checker.** `validate_expected.py testing/synthetic/scenarios/ST06-*` gives `ok` for ST06-base,
   ST06-pump-tank-only and ST06-tank-needs-consent. It does not read `read_by_design` semantics or notes.md.
2. **Faithful to the shape.** The base matches the shape's base job:
   - a 1988 Class 5 office, six storeys over a Class 7a basement, sprinklered throughout;
   - electric duty, diesel standby and jockey pumps and a 150 kL tank in the enlarged basement store;
   - level 3 vacant and fitted out, the other floors tenanted;
   - two CDCs and two AS 4000 lump sum contracts.

   Every trigger is denied in the shape's negation wording or is absent: fire safety order, tenant fit-out,
   interim occupation, single owner, pump-tank-only (d01); existing fire engineering (d03); tank-needs-consent
   (d09, d12); DA fire upgrade (d09); performance solution, town main, hydrant, crane, electrical supply (d12);
   asbestos (d14); amenities and affected part (d10, d15); heritage (d02). `diesel-noise-exhaust` is absent, which
   the brief allows (finding 10).

   The declared traps are right and real: basement (×2), RFP cue, acid sulfate class 5, fire stairs, AFSS kind,
   Sydney Water asset, backflow contamination, fee stages, structural fire tank, flooding, Stage 1 cut-over, fire
   storage and the sprinkler alias. Nothing undeclared fires a trigger: the documents say "tea point", not
   "kitchen", and no base document uses any trigger's evidence wording.
3. **Register.** Numbers, revisions, titles and dates follow DISCOVERY.md §4.3 and `markdown_identity.go:81`.
   - Labels used: `Ref`, `Proposal no`, `Proposal ref`, `Quote no`, `Report no`, `Drawing number`, `Revision`,
     `Report date`, `Date` and `Title`.
   - Every `#` heading is identical to its `Title:`. d16 and d17 have `Title:` and no heading.
   - d02 and d08 are table title blocks, so their number and date cannot be read; each title comes from its
     heading. The keys rightly hold the true values.
   - d17 → d16 cannot be linked, because `priorToLink` (`internal/intake/file.go:241`) needs a green Jev
     `supersedes`.
   - The invoices (d18–d20) are not register rows; their fields come from the invoice reading.
4. **Profile.** I tested every `carries` sentence for a `flag.*` or `det.*` key against its trigger in
   `flag_readings.yaml` and `determinants.yaml` (scratch script). All of them match except four:
   - d12 `det.water_supply_source` (finding 2);
   - d12 `flag.fire_specialist_systems` (finding 3);
   - d12 `flag.staged` and d13 `det.existing_building` (finding 12).

   Other points:
   - "constructed in 1988", "rise in storeys of 6", "sprinkler protected throughout", "Class 7a", "AS 4000",
     "lump sum" and "budget" all match.
   - `det.type_of_construction` (finding 5) and `proj.regulated_design` (finding 1) are not read.
   - No drawing, email or photo is a profile source.
5. **Fees and invoices.**
   - Every stage table sums to its stated total, in all three scenarios:
     - base: d05 $101,000, d06 $24,000, d08 $24,000, d09 $22,000, d10 $68,000, d11 $4,500;
     - pump-tank-only: $90,000, $12,000, $13,000 and $3,300;
     - tank-needs-consent: d09 $18,500, d21 $15,500.
   - d07 is GST-inclusive. Each figure divided by 1.1 is right: 4,400 → 4,000, 19,800 → 18,000, 2,200 → 2,000,
     6,600 → 6,000, total 30,000, rate 209 → 190. In pump-tank-only the total is 18,700 → 17,000.
     `gst_inc_normalised` is a real flag.
   - All nine ABNs pass the checksum, and the invoice ABNs match their firms.
   - Invoices:
     - d18: 18,500 + 1,850 = 20,350, clean.
     - d19: GST is $375 on $7,500, so `GSTOf` is off by more than 1 cent and the ledger raises `gst_rate`
       (`ledger.go:312`). The total adds up, so there is no arithmetic flag.
     - d20: clean. In pump-tank-only it is 2,600 + 260 = 2,860, following the smaller survey.
6. **Variants.**
   - **tank-needs-consent** changes only `hdr.cond.planning` (cdc → da).
     - Both of the shape's first two evidence sentences are in d09 and d12, verbatim.
     - Both negations are gone.
     - The added town planner proposal (d21) and SEE (d22) are what the trigger adds.
     - d02's "complying development may be carried out" is correctly turned into a declared trap.
     - The chronology is broken (finding 7).
   - **pump-tank-only** changes no profile value as keyed.
     - Both evidence sentences are in d01 and d12.
     - The fit-out negation is replaced, and d10 and d15 go.
     - The fee cuts follow from the removed scope, and `changed_from_base` records them.
     - Stages 1 and 2 of d05 are unchanged, so d18 stands.
     - The operational constraint really moves (finding 11), and so would the budget (finding 4).
7. **Realism.** A Sydney fire services PM would recognise this. The sequence is real:
   - AFSS with a thin flow-test margin → Sydney Water pressure and flow statement → owner's brief → seven
     proposals → hazmat survey, slab assessment and options report → FS-101 A then B for tender.
   - A Fire Safety Code CDC for the pump works beside a commercial fit-out CDC is the usual NSW route.
   - The 150 kL tank on a slab on ground in a basement store is plausible for an OH2 car park demand.
   - North Sydney E2 under LEP 2013 is right.

   The fees are plausible. The fire services fee ($101k) is at the top of the range for a $1.2M upgrade, but it
   includes the level 3 design and the investigation. The interior designer's $68k suits one floor of about
   1,000 m². The certifier's $22k for two CDCs, principal certifier and OC is plausible.

## On the author's shape problems

1. **Diesel negation contradicts the base: agree. Change the shape, and do not use the author's replacement.**
   "Two electric pumpsets on separate supplies … no diesel engine" cannot be said of a job with a diesel standby
   pump. Leaving the trigger absent in the base is allowed and keeps the keys right.

   The author's replacement says the exhaust discharges "at the car park entry". Discharging diesel exhaust at a
   pedestrian and vehicle entry is not what a mechanical engineer would design, and it reads as contrived. The
   discharge must also not be "above the roof, through the existing riser", because that is the trigger's own
   evidence. See finding 10 for the wording.
2. **Budget with no profile key: disagree with the workaround. Change the shape.** Keeping the base budget to the
   upgrade only does keep the variant's key unchanged. But it makes the base's own budget answer wrong for the
   job (finding 4). The trigger really does change the budget, because removing a fit-out removes its money. So
   the shape should list `proj.budget_stated` on `pump-tank-only` and on `tenant-procured-fitout`, as the author's
   first option said.
3. **Operational constraints in pump-tank-only: agree. Change the shape. No key change.** With every floor
   tenanted, the true value is `live_environment`, and the trigger moves an input it does not list. Both keys
   already accept `partial_occupation` and `live_environment`, so no score changes. Narrowing the variant key to
   `live_environment` would test Jev's choice between two labels that are close, which is not worth the
   fragility (finding 11).

## On the missed-by-design list

- **Type of construction: right but incomplete.** It is derived, never read. Also, the rule and its table are
  `status: draft`, so `Derive` returns unknown (reason `unreviewed`) and the app shows it blank. This is the same as
  ST17's finding 2. See finding 5.
- **Regulated design: wrong in the key, right in spirit.**
  - Nothing reads it. `internal/profile` has no extractor for it, and `flag_readings.yaml` says "Regulated design
    needs professional advice".
  - The RFP derives "no", because no part is Class 2, 3 or 9c (`internal/rfp/drivers.go:357-369`).
  - The key must say `read_by_design: false` (finding 1). Note also that the shape comment "DBP Act covers Class 2
    and 9c" omits Class 3, which the code includes. That makes no difference for Class 5.
- **Three table title blocks: two, not three.** d02 and d08 are right. d19 is an invoice and has no register row,
  and its number and date come from the invoice reading. So "titles come from the heading" does not apply to it
  (finding 6).
- **Rev B → A: right.** No Jev `supersedes` is applied, so `priorToLink` writes nothing.
- **Second CDC: right.** The profile holds one planning value. The fit-out CDC rows are the shape's program
  business, not a missed key.
- **Fit-out budget: should not be missed.** It is missed here only because of the workaround (finding 4).

## Findings

1. **All three `profile.json`: `proj.regulated_design` is keyed as read.**
   - **What is wrong.** The key is `{"value": "no", "sources": ["d09"], "read_by_design": true}`. No profile
     reader extracts it, and ST17 keyed the same field `read_by_design: false`. The "no" the user sees is the RFP
     derivation from classes 5 and 7a, not a reading of d09.
   - **Change**, in ST06-base, ST06-pump-tank-only and ST06-tank-needs-consent `expected/profile.json`:
     `"proj.regulated_design": {"value": "no", "sources": [], "read_by_design": false, "stated_in": ["d09"],
     "not_read": "No profile extraction; the RFP derives no because no part is Class 2, 3 or 9c (internal/rfp/drivers.go deriveRegulatedDesign)"}`.
     In base notes.md, the missed-by-design line reads: "`proj.regulated_design` ("no"): not read; the RFP
     derives it from det.ncc_class 5 and 7a. The key keeps the value."

2. **All three manifests, d12: the `det.water_supply_source` sentence matches no trigger.**
   - **What is wrong.** The sentence is "supplied from Sydney Water's reticulated town main". The triggers are
     `town water`, `water main`, `mains water`, `potable water`, `bore` and `rain ?water tanks?`, and "town main"
     matches none of them. The notes say d04 may file as correspondence and go unread, and that d12 is the
     fallback. It is not a fallback.
   - **Change.** In all three d12 `carries` and `content`, say "The sprinkler system is supplied from Sydney
     Water's reticulated town water main through the existing fire service connection". Keep `sources: ["d04",
     "d12"]`.

3. **All three manifests, d12: the `flag.fire_specialist_systems` sentence matches no trigger.**
   - **What is wrong.** "No special hazard suppression or other specialist fire systems are required" does not
     match `(facade|cladding|timber|modular|tested.system|structural)[^.]{0,100}(fire|departure|performance
     solution)` or its reverse. The only passage that triggers is d06's trap. The key's "no" with source d12 is
     therefore unsupported.
   - **Change.** In all three d12 `carries` and the Compliance section of `content`, say "No structural fire
     assessment and no facade, cladding or tested-system fire departure is required; no specialist fire systems
     are needed."

4. **Shape, ST06-base d01, all three `rfps.json`, and both variants: the stated budget is part-scope.**
   - **What is wrong.** `proj.budget_stated` is $1,200,000, "for the base-building fire services upgrade". The
     fit-out is "to be advised". But the cost plan frames its construction lines with this value
     (`internal/store/cost_budget.go:55`), and those lines include the fit-out. Each RFP's construction budget
     also comes from it (`internal/rfp/drivers.go:160`). The base therefore cannot be right, and the shape's own
     `scenario_inputs` asks for a split.
   - **Change.**
     - In `types/T06-sprinkler-pump-tank-upgrade/shape.yaml`, set `pump-tank-only` and `tenant-procured-fitout`
       to `profile: [proj.budget_stated]`, with the note "Removing the fit-out removes its budget; the stated
       budget becomes the fire services figure."
     - In ST06-base and ST06-tank-needs-consent d01 `carries` and `## Budget`, say "Approved budget: $2,700,000
       excluding GST (fire services upgrade $1,200,000; level 3 fit-out $1,500,000)". Drop the "funded
       separately … will be advised" sentence. $1.5M is about $1,400/m² over roughly 1,100 m², which is mid-range
       for a Sydney office fit-out.
     - In both of those scenarios, set `proj.budget_stated` to `270000000` and `rfps.json`
       `scenario_facts.rfps.budget_stated_cents` to `270000000`.
     - Leave pump-tank-only at $1,200,000. In its notes, say the value differs from the base because of the
       trigger. In its `changed_from_base` for d01, add "budget restated as the fire services figure only".
     - In base notes.md, remove the "Budget" departure and the "level 3 fit-out budget is not stated" line.

5. **All three notes and `profile.json`: the `det.type_of_construction` wording.**
   - **What is wrong.** "Derived by rule.ncc.type-of-construction … no document states it" omits that the rule
     and `knowledge/tables/type_of_construction.yaml` are `status: draft`. The derived row is therefore blank
     today, as ST17 found.
   - **Change.** Set the `not_read` text and the base notes line to: "Derived only, never read; blank while
     rule.ncc.type-of-construction and its table are draft (reason unreviewed). Table C2D2 gives A for Class 5
     and 7a at rise 6. The key keeps the true value." Keep `sources: []` and `read_by_design: false`.

6. **ST06-base `expected/notes.md`: three wrong claims.**
   - **What is wrong.**
     - (a) "d02, d08 and d19: Markdown-table title blocks … titles come from the heading" includes d19, which is
       an invoice with no register row.
     - (b) "`det.water_supply_source` is also stated in d12" is untrue until finding 2 is applied.
     - (c) The `proj.regulated_design` and type of construction lines (findings 1 and 5).
   - **Change.**
     - Make the title-block line read "d02 and d08: Markdown-table title blocks, so number and date are missed
       and titles come from the heading. d19's table is an invoice; its fields come from the invoice reading."
     - Keep the d04 line once finding 2 is applied.
     - Apply the wording from findings 1 and 5.

7. **ST06-tank-needs-consent d09: dated before the facts it states.**
   - **What is wrong.** d09 is dated 4 March 2026 (wave 2). It records council's advice that a DA is required,
     but d12 dates that advice to the pre-lodgement meeting on 16 April 2026. d09 also records that two car
     spaces are lost, but the options report that chose the tank position is dated 20 April. No certifier writes
     that in March.
   - **Change.** Set d09 `date` to `2026-04-24`, `wave` 3, and the title-block `Date: 24 April 2026`. Set the
     register key `d09.date` to `2026-04-24`. Add a `changed_from_base` note: "re-dated after the pre-lodgement
     meeting". The fees and invoices are unchanged.

8. **ST06-tank-needs-consent `fees.json` d09: an exclusion the document states is missing.**
   - **What is wrong.** The content's exclusions are "Planning Portal and long service levy charges; DA fees",
     but the key lists only the first.
   - **Change.** Add `{"label": "DA fees", "kind": "reimbursable", "state": "excluded", "amount_cents": null}`
     to d09 `prices`.

9. **ST06-pump-tank-only d01: the work-type sentence lost its anchor.**
   - **What is wrong.** The base's `hdr.work_type` sentence says "refurbishment and fit-out". The variant's says
     only "upgrading the sprinkler water supply … no tenancy works". A supply that "no longer meets AS 2118.1
     demand" invites `remediation` ("Remediation / rectification"). d12's sentence has the same weakness. The
     key `refurb` stays right, but its support is thin.
   - **Change.** In d01 `carries` and `content`, say "We are refurbishing the base-building fire services by
     upgrading the sprinkler water supply of the existing building. The scope is limited to the fire pump and
     tank upgrade; no tenancy works are included."

10. **Shape `diesel-noise-exhaust` negation, and later all three d12.**
    - **What is wrong.** This is shape problem 1. The negation contradicts the base, and the author's replacement
      is not credible.
    - **Change.** In `shape.yaml`, set `diesel-noise-exhaust.negation` to: "The diesel exhaust discharges
      through the existing basement plant exhaust shaft, clear of tenancies, air intakes and neighbours; the
      weekly test run is not audible in the tenancies." Once the shape changes, add that sentence to the
      Recommended option section of d12 in all three scenarios. No key changes. Until then, leave the trigger
      absent.

11. **Shape `pump-tank-only` profile.**
    - **What is wrong.** This is shape problem 3. The variant moves `hdr.cond.operational_constraints` (every
      floor tenanted), but the trigger does not list it.
    - **Change.** Set `profile: [hdr.cond.operational_constraints, proj.budget_stated]`, adding the first key to
      finding 4's change. Add the note "live_environment once every floor is tenanted; keys accept
      partial_occupation too." No scenario change.

12. **All three manifests: two source sentences match no trigger (minor).**
    - **What is wrong.**
      - d12 `flag.staged`, "Each contract is one construction stage", matches none of the staged triggers. d01
        carries the flag, so the key holds.
      - d13 `det.existing_building`, "the existing basement slab", does not match `\bexisting building\b`. The
        other sources carry it.
    - **Change.**
      - Make the d12 sentence "Each contract is one construction stage; there is no staged construction" ("The
        contract …" in pump-tank-only).
      - Remove `d13` from `det.existing_building.sources` in all three profiles, or change the d13 sentence to
        "the existing building's basement slab".
