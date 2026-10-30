# T02 Multi-residential apartments, 4–8 storeys: independent review

Reviewer: an independent agent acting as a senior NSW PM who has delivered developer D&C residential flat
buildings. Brief: `testing/synthetic/briefs/02-review.md`. Read 30 October 2026 against `main` plus the
working tree. I wrote no file other than this one and ran no state-changing git command. I ran the validator,
read-only Python over the YAML, and web lookups of the instruments the author was unsure of.

## Verdict: **FIX**

The map is deep and mostly right. It covers every top-level system and the Class 2 statutory chain (DBP Act,
RAB Act, strata bond, s73, DNSP). Its RFP `expect` blocks use real template keys throughout. It is not yet
safe as an answer key, for three reasons:

- **The program's critical path is wrong at the CC.** One construction certificate is gated on every
  post-novation regulated design, and demolition and excavation wait for it. Real D&C apartment jobs use
  staged CCs, and demolition under a DA needs no CC.
- **Base-scenario documents will trip several regex triggers.** These are phrases every T02 document set
  uses ("existing buildings", "four buildings", "Stage 1", "water main", "traffic noise", "street trees").
  The map lists none of them, so scenario authors will produce profiles that disagree with `profile`.
- **A handful of `built_in` claims and trigger deltas are wrong.** For example, `traditional-procurement`
  RFPs `yes`, `rail-corridor` adding a consultant already in base, and `underpinning-neighbour` keyed to a
  flag that base already sets.

All of these can be fixed in place. Nothing is wrong at the root.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T02-apartments/shape.yaml` →
`ok`, exit 0, no warnings. It must be run from the repo root: run from `testing/synthetic/` it crashes on
`knowledge\profile\taxonomy.yaml`, a path quirk of the tool, not of the map.

Counts: 15 triggers, 23 packages, 58 cost lines, 99 activities, 22 RFPs, 76 work items, 23 gaps.

I also checked every stage, position and element key in each `rfps[].expect` and `app_differs` against
`knowledge/rfp/<template>.yaml`. All of them exist.

## 2. Rulings on the author's open questions

| Question | Ruling | Evidence | Status |
|---|---|---|---|
| NCC clause for Class 2 sprinklers at rise 4+ | **NCC 2022 Vol One E1D6** "Where sprinklers are required: Class 2 and 3 buildings other than residential care buildings": sprinklers throughout the whole building if any part has a rise in storeys of 4 or more and an effective height of not more than 25 m. E1D4(2) says the system complies with **Specification 17 and Specification 18 as applicable**. The base building (rise 6, effective height well under 25 m) is caught. Separately, **E1D9** covers a Class 7a car park, other than open-deck, with "more than 40 vehicles"; I could read only the tail of that clause. | ABCB NCC 2022 online, Part E1 (https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/e-services-and-equipment/part-e1-fire-fighting-equipment), clause text read | E1D6 and E1D4(2): `verified: true` (source above). E1D9: `verified: false`. The FRNSW summary says the Spec 18 concessions do not apply to a Class 2 building that also contains a Class 7 part (https://www.fire.nsw.gov.au/page.php?id=9326). Keep that `verified: false`. |
| Class 2 critical-stage inspections | For Class 2, 3 and 4, the certifier's mandatory list is: (1) after excavation and before the **first footing**; (2) before covering waterproofing in wet areas, for **at least 10% of rooms with wet areas**; (3) before covering stormwater drainage connections; (4) final, before the OC. **Fire stopping at penetrations is not on the list I could confirm.** It stays as the app's quality hold point and should not be called a critical stage. There is no frame inspection and no slab-pour inspection by the certifier for Class 2: the structural engineer inspects reinforcement. | Bega Valley Council, citing cl 61 of the EP&A (Development Certification and Fire Safety) Regulation 2021 (https://www.begavalley.nsw.gov.au/Plan-and-build/Building-and-certification/About-inspections); Ryde Council information sheet for the 10% wording (old cl 162A). I could not open the Regulation itself (legislation.nsw.gov.au returned 403). | All `verified: false`, including cl 61. As owner ruling R7 says for Class 1, the project's CC lists the inspections that apply. |
| RAB Act expected-completion notice | Given **at least 6 months and not more than 12 months before applying for the OC**, to **Building Commission NSW** through the NSW Planning Portal. If the developer expects to apply within 6 months of starting work, the notice is due within 30 days of the start. If no notice is given, a prohibition order can hold the OC and the strata plan registration. | NSW Government, "Lodging an expected completion notice" (https://www.nsw.gov.au/housing-and-construction/compliance-and-regulation/professionals-working-on-regulated-buildings/developers/lodging-an-expected-completion-notice) | Window and recipient confirmed. The Act's section number stays `verified: false`. |
| Strata building bond rate and timing | **2% of the contract price.** The rise to 3% is deferred until 1 July 2028. It is **lodged with the Secretary (NSW Fair Trading) before the OC is issued** (the map says "before the OC": correct). Interim inspection and report **15–18 months** after the building work is finished. Final inspection and report **21–24 months** after. The scheme does not apply where HBCF applies. | NSW Government, "Strata Building Bond and Inspections Scheme" (https://www.nsw.gov.au/housing-and-construction/building/strata-building-bond-and-inspections-scheme) | Rate, lodgement and inspection windows confirmed. The "4 or more storeys" scope depends on the HBCF exemption above 3 storeys, which owner ruling R4 keeps `verified: false`, so the scope stays `verified: false`. SSMA 2015 Part 11 section numbers: `verified: false`. |
| Design review panel and panel thresholds | **Housing SEPP Chapter 4** applies to a building of 3 or more storeys (not counting underground car parking) with 4 or more dwellings. The **Local Planning Panels Direction** sends Chapter 4 development of **4 or more storeys** to the local planning panel. A **regionally significant** application (CIV above the Planning Systems SEPP threshold) goes to the Sydney district planning panel instead. Referral to a design review panel happens where the council has constituted one (Housing SEPP s145). That is common in Greater Sydney but not universal, and **there is no numeric DRP threshold**. | planning.nsw.gov.au, "Design of residential apartment development"; Local Planning Panels Direction PDF (https://www.planning.nsw.gov.au/sites/default/files/2023-03/local-planning-panels-direction-development-applications.pdf) | Chapter 4 scope and the LPP 4-storey trigger confirmed from NSW pages. The RSD threshold (generally $30 m CIV, from memory), the s145 wording and the design verification statement clause (EP&A Reg 2021 s29) stay `verified: false`. See finding 9. |
| Base: demolition of existing houses | **Keep in base.** An amalgamated R4 site almost always carries dwellings. Keep `det.existing_building: false`: the determinant means "work to an existing building" (`knowledge/determinants.yaml existing_building`). Keep the year (1962): with a `remove` item it raises `cq.pre-2004-fabric-hazardous-materials-survey`. Add the trap in finding 3. | — | — |
| Base: OSD | **Keep in base**, and add a negation for councils with no OSD policy (for example "OSD is not required under council's stormwater policy for this catchment"). | [D] | — |
| Base: MUSIC modelling | **Keep in base.** Metropolitan DCPs routinely require WSUD targets for a residential flat building. | [D] | — |
| Base: public-road works | **Keep in base.** New crossing, kerb, gutter and footpath under s138 are standard conditions. Note that `flag.public_road` also makes the civil scale complex in base, which is right for this job. | `knowledge/rfp/civil.yaml scale` | — |

**Is 23 packages / 22 RFPs the right size?** **Yes.** Every "always" consultant is one a Sydney council DA
checklist or a lender asks for on a 64-unit RFB: acoustic, BASIX/NatHERS, waste, landscape, registered-QS cost
report, access, traffic, BCA, geotech, survey and stormwater. Hazmat is justified by the base demolition. The
only soft entries are already marked soft: vertical transport `optional` (one or two lifts are often specified
by the services engineer or the lift supplier) and fire engineer `usually`. I would not add any base package.
Arborist is the most likely to belong in base, because a boundary-to-boundary basement almost always reaches a
street or neighbour tree TPZ. It is correctly a trigger and already in `variants`. Do not split RFPs further.
The certifier and BCA are separate appointments by design.

## 3. Findings

1. **`shape.yaml program.activities` (Construction certificate, Appoint principal certifier, Demolish, Install
   retention shoring): the CC logic is wrong for a D&C apartment job.**
   - *Wrong:* one CC follows "Prepare regulated designs and design compliance declarations", s73 and LSL. Site
     establishment, asbestos removal, demolition and shoring all wait for it, so the whole post-novation
     documentation period sits on the critical path ahead of the site.
   - *Evidence:* DOMAIN.md §5 itself says "staged CCs are common: excavation and shoring first". In NSW, a CC
     is needed for building work, not for demolition under a DA ([D], `verified: false`). The builder starts
     demolition on the DA conditions (hazmat, SafeWork notice, dilapidation) while documentation proceeds.
   - *Change:*
     - (a) Split the activity into "Construction certificate: early works (excavation, retention, basement
       structure)" and "Construction certificate: main works". Each needs the regulated designs for its stage.
     - (b) "Disconnect services and remove asbestos" `after: ["Award the contract", "Hazardous materials survey
       of the existing dwellings", "Dilapidation surveys of adjoining properties"]`, not after the CC.
     - (c) "Install retention shoring" after the early-works CC.
     - (d) "Floor cycle" after the main-works CC.
     - (e) Give `Construction certificate` the alias `Construction certificate: main works`.
   - Keep `built_in: partial`. The app has one CC fragment (`fr.construction-certificate`) and no staging.

2. **`shape.yaml program.activities`: the DBP builder's declaration and the OC application are missing.**
   - *Wrong:* DOMAIN §5 names the builder's principal compliance declaration, but the program has no row for it.
     The RAB Act notice is timed against the *application* for the OC, which is also not a row.
   - *Change:*
     - (a) Add `{label: "Principal compliance declaration (DBP Act)", stage: completion, fragment: null, after:
       ["Commission and test fire systems and essential services"], built_in: "no", why: "Builder's declaration
       before the OC application (owner ruling R3; verified: false)"}`.
     - (b) Make "Occupation certificate" `after` include it.
     - (c) Reword "Expected completion notice to the Building Commissioner" to "Expected completion notice to
       Building Commission NSW", with why "at least 6 and not more than 12 months before applying for the OC
       (nsw.gov.au; section verified: false)".

3. **DOMAIN.md §10 and `shape.yaml`: base-scenario wording trips regex triggers the map does not list.** These
   phrases appear in almost every T02 document set:
   - "**existing building(s)** on the site will be demolished" matches `\bexisting building\b`
     (`knowledge/determinants.yaml existing_building triggers`). A `stated_true` answer turns on the RFP
     `existing_building` module and the existing-assessment stages (`knowledge/rfp/modules.yaml`), against base
     `det.existing_building: false`.
   - "demolition of the **four existing buildings**" and "**building a** minimum 6 m from the boundary" match
     `(?:two|three|four|five|\d+) buildings` and `\bbuildings? [a-d]\b`
     (`knowledge/profile/flag_readings.yaml multi_building`). That gives the architect `brief_masterplan`.
   - "**Stage 1** CC (excavation and shoring)" matches `\bstages? [1-9a-c]\b` (`flag_readings.yaml staged`).
     That gives staged occupation and masterplan rows.
   - "connect to the **water main** / **sewer main** in the street" matches `(?:sewer|water) (?:main|asset)`
     (`flag_readings.yaml authority_asset`). That gives the structural `authority_asset` stage and the
     architecture `water_asset` row, against base `flag.authority_asset: "no"`.
   - "road **traffic noise** assessment" in the base DA acoustic report matches `\b(road|rail|aircraft|traffic)
     noise\b` (`determinants.yaml noise_affected_site`), against base `det.noise_affected_site: false`.
   - "new **street trees**" in the public-domain scope matches `flag_readings.yaml arborist_offsite_trees`.

   *Change:* add each to DOMAIN §10 with the base phrase and the expected reading. Add a matching negation
   phrase to `sydney-water-asset` ("connection to the existing water main in the street; no Sydney Water assets
   cross the site"), `two-buildings-shared-basement` ("demolition of the existing buildings; a single
   residential flat building is proposed") and `noise-affected-site` ("the acoustic report assesses road
   traffic noise from a local street and standard glazing achieves the internal criteria").

4. **`shape.yaml triggers[traditional-procurement].built_in.rfps: "yes"` is wrong. Make it `"partial"`.**
   - *Wrong:* for a non-house traditional job, the architecture boundary falls to the last rule, `brief_masterplan
     → interiors`. `tender_traditional`, `bidding` and `for_construction` come after it, so they are priced as
     **options**, not included (`knowledge/rfp/architecture.yaml boundary[3]`, stage order; resolution rule in
     `KNOWLEDGE_INDEX.md` §5.1).
   - The trigger also lists no procurement delta, although its `why` says novation changes. Under traditional,
     the services templates keep their fixed `inception → design` boundary, so their tender stage stays an
     option.
   - *Change:*
     - (a) `built_in: {program: "yes", rfps: "partial", procurement: "yes"}`.
     - (b) Add `adds.rfps` for the architect with `stage.tender_traditional` and `stage.for_construction`
       expected include and an app note "option after the boundary".
     - (c) Add `removes.rfps` entries for `stage.dnc_tender` and `stage.post_novation`.
     - (d) Add a sentence that every package's novation becomes `no` (`internal/procurement/packages.go
       DefaultNovation`).

5. **`shape.yaml triggers[rail-corridor].adds.procurement: [consultant.acoustic]` is already in base.**
   - *Wrong:* acoustic is `need: always` in `procurement`, so the trigger adds nothing and the scorer would
     count it twice.
   - *Change:* drop `procurement`. Add `rfps: [{discipline: consultant.acoustic, must_include: ["Rail noise and
     vibration assessment to the Transport and Infrastructure SEPP criteria (verified: false)"]}, {discipline:
     consultant.geotechnical, must_include: ["Asset exposure assessment for the rail corridor"]}]`. The geotech
     row is `asset_exposure` (client flag, ask).

6. **`shape.yaml triggers[underpinning-neighbour].profile: [flag.excavation_adjoining]` cannot distinguish the
   variant.**
   - *Wrong:* base already sets `flag.excavation_adjoining: "yes"`, so the variant changes no profile key. Its
     only real delta is the `substructure.underpinning` work item.
   - *Change:* `profile: []`, with the note "delta is the work item only; no flag or determinant names
     underpinning". Keep `fr.underpinning-2` (needs `new`/`alter`/`repair` plus class 2 plus new or extend,
     verified in `knowledge/program/fragments.yaml`) as `program: "partial"`. Add to `gaps`: "no profile key for
     neighbour underpinning".

7. **`shape.yaml gaps[cost_plan, budget split]`: the basement-multiplier gap is misstated.**
   - *Wrong:* `residential_apartments_high_rise` has no `basement` multiplier because its shares already come
     from "Belmore (R1) and St Leonards (R2) apartment estimates, both over basements"
     (`knowledge/costs/budget_splits.yaml`, `basis`). Substructure 9% is a basement share. Applying ×2 would
     double-count.
   - The real issue is the reverse: a 4-storey block (low-rise split, ×2) and a 5-storey block over the same
     basement (high-rise split, ×1) get different substructure shares.
   - *Change:* reword the gap to "the high-rise split assumes a basement and has no no-basement case; the
     low-rise split doubles substructure only when `flag.basement` is yes". Make the same change in DOMAIN §6.

8. **DOMAIN.md §5, `shape.yaml work_items[fire-active.sprinklers]`, `program` inspection rows: apply the
   rulings in §2.**
   - *Change, sprinklers:*
     - `fire-active.sprinklers` why: "NCC 2022 E1D6 (whole building, rise in storeys ≥ 4, effective height
       ≤ 25 m); E1D4(2): Specification 17 and 18 as applicable (verified: true, ABCB NCC 2022 Part E1)".
     - DOMAIN §2 `fire-active`: replace "E1D? / Specification 18; verified: false" with the same. Add "E1D9
       car park over 40 vehicles (verified: false)" and "Spec 18 concessions are not available with a Class 7
       part (verified: false)".
     - Add to the fire-services RFP `must_include`: "Car park sprinklers to AS 2118.1".
   - *Change, inspections:*
     - "Inspect fire stopping before concealment" why: drop "also a Class 2 critical stage". Say "quality hold
       point; not on the confirmed cl 61 list".
     - "Inspect wet-area waterproofing before tiling" why: "at least 10% of rooms with wet areas (cl 61,
       verified: false)".
     - DOMAIN §5, "Inspections and hold points": replace the list with the four items in §2.
   - *Change, strata bond:*
     - The DOMAIN §5 bullet becomes: 2% (3% deferred to 1 July 2028); lodged before the OC is issued; interim
       report 15–18 months and final 21–24 months after completion (nsw.gov.au).
     - Add the same timing to the "Strata bond interim and final inspection reports" why, as statutory windows,
       not durations.

9. **DOMAIN.md §1 and §5, `scenario_inputs[flag.panel]`: the determining panel depends on the CIV, and the
   base may well be regionally significant.**
   - *Wrong:* 64 apartments, about 6,200 m² GFA and two basements (PROJECT_TYPES range $10–60 m) may sit above
     the regionally significant threshold. If it does, the Sydney district planning panel determines it, not
     the local panel.
   - *Change:*
     - Phrase the scenario input as "The application will be determined by the <Local | Sydney District>
       Planning Panel". Tell the scenario author to keep it consistent with the stated CIV (threshold
       `verified: false`).
     - Give the activity "Local planning panel determination" the alias "Planning panel determination".
     - Mark the DRP rows in the program as "where council has constituted a design review panel" and give them
       the aliases "Urban design review panel" and "Design excellence panel".
   - `flag.panel` stays `"yes"` either way; the town-planning RFP is unaffected.

10. **`shape.yaml program.activities[Sydney Water…]`: s73 and building plan approval are merged and wrongly
    gated.**
    - *Wrong:* the s73 application needs the DA consent and goes in through the WSC soon after determination.
      The **s73 compliance certificate** is an OC (and strata) prerequisite. **Building plan approval** (Sydney
      Water Tap in) is the item needed before the CC and works. The map makes the CC wait on the whole s73
      process.
    - *Change:* split it into "Lodge the Section 73 application through a WSC" (`after: ["Local planning panel
      determination"]`) and "Sydney Water building plan approval" (`after: ["Coordinate consultant
      documentation"]`). The CC depends only on the second. "Fire safety certificate, BASIX completion and
      Section 73 certificate" already carries the certificate. Keep all `verified: false`.

11. **`shape.yaml program.activities[DNSP connection application…]`: it starts too late.**
    - *Wrong:* the substation location and easement must be on the DA plans, and Ausgrid and Endeavour design
      lead time is long. The connection enquiry starts in design development, not at award ([D]).
    - *Change:* `after: ["Develop the design"]`. Label it "DNSP connection enquiry and substation design". Keep
      `partial`: the app row sits in construction.

12. **`shape.yaml rfps[consultant.mechanical].expect` and `rfps[consultant.fire_engineer].expect`: stages left
    unscored.**
    - *Wrong:* mechanical `planning` (S4 Planning, flag `mech_planning`, `if_unknown: ask`) and fire engineering
      `s8_specialist`, `s8_staging`, `s9_peer` are in no list. Every other RFP lists every stage.
    - *Change:*
      - Mechanical: `stages_option: [planning, construction, handover, seasonal]`, plus `app_differs`
        `{key: stage.planning, app: ask, expect: option, why: "DA needs only plant and exhaust locations, normally
        carried by the architect and acoustic report"}`.
      - Fire engineering: `stages_absent: [s9_advisory, s8_staging]`, `stages_option` add `s8_specialist,
        s9_peer`.
    - Separately, the electrical `stage.planning` `why` should say "the S4 Authority stage carries the DNSP
      application", not "DNSP application is base work".

13. **`shape.yaml variants` and `triggers`: a common trigger that exercises a built-in rule is missing, and one
    variant tests nothing the app does.**
    - *Wrong:* `noise-affected-site` has every `built_in` `no` and zero app delta (`gaps[harness]`), yet it takes
      a variant slot. The map has no trigger for a complex facade, although `det.facade_type: complex` →
      Facade consultant is a live rule (`knowledge/works/package_defaults.yaml complexity_additions[7]`).
      Rainscreen and unitised facades are common on 6–8 storey Sydney RFBs.
    - *Change:*
      - (a) Add a trigger `complex-facade`. Evidence: "a ventilated rainscreen of terracotta baguettes on an
        aluminium subframe"; "the facade will be designed by a specialist facade engineer". Negation: "face
        brick and rendered masonry with standard aluminium windows". Profile `[det.facade_type]`. Adds
        procurement `[consultant.facade]`, cost plan `["Facade engineer"]`, RFP none (no template).
        `built_in: {procurement: "yes", cost_plan: "yes"}`, source `knowledge/works/package_defaults.yaml
        complexity_additions[7]`.
      - (b) Replace `noise-affected-site` in `variants` with `complex-facade`.
    - Consider also an `ssd-affordable-housing` trigger: the Housing SEPP in-fill affordable housing bonus with
      the CIV over the SSD threshold, `hdr.cond.planning: ssd` → `sk.ssd-dc`, `fr.ssd-*`, with thresholds
      `verified: false`. This is now a common route for this type, and it changes the skeleton, consent
      fragments and DA-fee line.

14. **`shape.yaml cost_plan["Strata building bond"]`: it is a refundable security, not a cost.**
    - *Change:* keep the line under `fees_and_charges`, with why "2% of contract price (nsw.gov.au), refundable
      security held at least until the final inspection report; carry as a cash-flow item, not a sunk fee".
    - Add the line "Strata inspector (SBBIS interim and final reports)" under consultants, `built_in: "no"`. The
      developer appoints and pays the inspector.

15. **`shape.yaml triggers[trees-affected]`: the program expectation cannot come from the stated work item.**
    - *Wrong:* `work_items: [{system: site.trees, action: retain}]`, but `fr.tree-protection` fires only for
      `new`, `alter` or `repair` (`knowledge/program/fragments.yaml`), and `retain` items bring no fragment.
      `program: "partial"` is right, but the why does not say so.
    - *Change:* add `why: "fr.tree-protection exists but retain brings no fragment; arborist never suggested, so
      the arborist RFP exists only if the user creates the package"`.

16. **`shape.yaml triggers[sydney-water-asset].adds.rfps`: one app delta is missing.**
    - *Change:* add `{discipline: consultant.architect, must_include: [{text: "Authority asset coordination",
      key: position.water_asset}]}`. `water_asset` reads `flag.authority_asset` and is `absent` in the base
      expect.

## 4. Owner rulings

- R3 (DBP Act for Class 2): applied correctly (regulated designs, declarations, `proj.regulated_design`).
  Finding 2 adds the missing builder's declaration row.
- R4 (HBCF): exemption above 3 storeys kept `verified: false` ✔.
- R6 (livable housing optional for houses): not breached. `access-egress.livable-housing` here rests on ADG
  livable-housing and council adaptable-unit targets for Class 2, which is a different instrument. Keep the
  NCC references `verified: false`.
- R7 (Class 1/10 inspections): correctly not applied to Class 2. See the §2 ruling.
- R8 (DLP length): kept `verified: false` ✔.
- R1, R2, R5: not applicable.

## 5. Built-in claims checked

Checked against the cited file **and** the selection logic in `reference/KNOWLEDGE_INDEX.md` (§2 package rules
and class mapping, §3 cost lines, §4 skeleton and fragment predicates, §5 boundaries and `if_unknown`).

**Procurement**

- ✔ Architect, Structural, Civil, Geotechnical, Surveyor, Certifier, Hydraulic, Electrical: `baselines[0]`
  (new, not infrastructure).
- ✔ Town Planner: `complexity_additions[0]` (da). Also `baselines[0]`.
- ✔ Traffic, BCA, Mechanical: `baselines[1]` via `7a` → industrial (`knowledge/profile/taxonomy.yaml` 7a
  subclass; index §2.1).
- ✔ Fire Services: `scope_additions[1]` (detection in defaults; sprinklers in expected scope).
- ✔ Access: `scope_additions[5]` (accessible-access, livable-housing in defaults).
- ✔ Vertical Transport: `scope_additions[0]`; generic `tpl.rfp-capex` (no template in `knowledge/rfp/`).
- ✔ `trade.main_works` partial (`ProposeHeadContract` only over accepted gaps).
- ✔ Novation `option` everywhere on D&C (`DefaultNovation`, `BuilderDesigns`).

**Cost plan**

- ✔ `da-fees`, `cc-fees`, `service-authority`, `design-contingency`, `construction-contingency`, `escalation`,
  `exclusions` exist with the stated `applies_when` (`knowledge/costs/starter_lines.yaml`).
- ✔ Fee lines F01–F15 from the 15 adopted packages.
- ✔ Elements 01–13 and the 13.x children; "Plant and maintenance access" partial (only with
  `maintenance-access` in scope).
- ✘ Budget-split gap (finding 7).

**Program**

- ✔ Skeleton `sk.new-dc-da` (new, design_construct, da).
- ✔ `fr.novation` attaches after `fr.contract-award`.
- ✔ `fr.construction-certificate` in approvals (partial).
- ✔ `fr.investigate-geotech`, `-survey`, `-services` need an `investigate` action (partial).
- ✔ `fr.demolition-2` needs `remove` plus class 2 plus new (partial).
- ✔ `fr.windows-2` attaches after `fr.structure-concrete-2`.
- ✔ `fr.utility-major` rows "Lodge the supply application", "Authority connection" in construction (partial).
- ✔ `fr.sprinklers-2` needs `fire-active.sprinklers` (partial).
- ✔ `fr.temporary-power` needs `electrical.temporary-power` (partial).
- ✔ `if.fire-stopping-before-concealment` (penetrations → interiors) fires.
- ✔ `if.sprinkler-pressure-test-before-ceilings` needs sprinklers (partial).
- ✔ `if.reinforcement-inspected-before-pour` names `substructure.slab-on-ground` (`knowledge/clusters/structure/
  interfaces.yaml`), so it never fires (partial).
- ✔ `fr.tender-docs-2` not D&C specific (partial).
- ✔ `fr.occupation-certificate`, `fr.handover`, `fr.defects-*`: yes.
- ✔ `fr.services-connections-major` for class 2.

**RFPs**

- ✔ Architecture: D&C boundary `brief_masterplan → dnc_tender`, so `regulated_design` and `post_novation`
  are options; `apartment_design` O for class 2 NSW; `mixed_use_map` I via `deriveFlags` counting classes 2
  and 7a (`internal/rfp/drivers.go:304`); `tenant_fitout` O beyond house.
- ✔ Structural: D&C boundary `concept_schematic → dnc_tender`; complex scale for class 2 or basement;
  `authority_asset` stage on `authority_asset` or `rail`.
- ✔ Civil: D&C boundary `schematic → dnc_tender`; `shoring` O with basement; `earthworks_suitability` '-'
  with no contamination, ASS or groundwater; `public_road_approvals` I.
- ✔ Geotechnical: boundary `g1_plan → g3_report`; `monitoring` O; `ass` ask.
- ✔ Land surveying: boundary `s1_plan → s4_services`; `titling` O always.
- ✔ BCA/certifier: shared template, fallback boundary `a1 → a3`.
- ✔ Town planning: `panel_attendance`, `variation_request`, complex scale on panel.
- ✔ Traffic: `report`.
- ✔ Services templates: fixed `inception → design` boundary, no D&C logic.
- ✔ Landscape `s6_dnc`, waste, cost-planning, fire-engineering templates exist; access template exists.

**Triggers**

- ✘ `traditional-procurement` RFPs `yes` (finding 4). ✔ Its program (`sk.new-traditional-da`) and procurement
  (novation `no`).
- ✔ `heritage-conservation-area`: `complexity_additions[1]` and `fr.heritage-referral`; cost line via the
  package.
- ✔ `cl46-variation`: `variation_request`.
- ✔ `flood-affected`: civil `flood_report`; `fr.flood-2` needs a `site.flooding` item (partial).
- ✔ `contaminated-land`: civil `earthworks_suitability`; `fr.investigate-contamination` (partial); no
  Environmental (dead rule `complexity_additions[2]`).
- ✔ `acid-sulfate-soils`: `fr.investigate-acid-sulfate` (partial); geotech `ass` ask.
- ✔ `groundwater-dewatering`: civil `earthworks_suitability` I on `flag.groundwater`.
- ✔ `trees-affected` RFP rows exist in `knowledge/rfp/arborist.yaml s2_impact` (see finding 15 for the
  program).
- ✔ `sydney-water-asset` and `rail-corridor`: structural `authority_asset`.
- ✔ `underpinning-neighbour` `fr.underpinning-2` (partial); see finding 6 for the profile key.
- ✔ `gas-reticulation`: `fr.gas-commercial` for class 2 (partial).
- ✔ `two-buildings-shared-basement`: architecture `brief_masterplan` on staged or multi_building.

**Expected derived**

- ✔ `det.type_of_construction: A` (`knowledge/tables/type_of_construction.yaml rows[0]`, `verified: true`,
  Table C2D2).

**Spot-checks of `no` claims (all ✔: the knowledge has nothing)**

- Acoustic, Landscape, Quantity Surveyor, Waste, Fire Engineer and Hazmat: no package rule names them
  (`knowledge/works/package_defaults.yaml`).
- No acoustic, BASIX or hazmat RFP template (`knowledge/rfp/` listing).
- No Housing and Productivity Contribution, LSL or strata line (`knowledge/costs/starter_lines.yaml`).
- No fragment for the design verification statement, DRP, RAB Act notice, strata bond or DBP declarations (grep
  of `knowledge/program/fragments.yaml`).
- No NCC E1D6 rule: `knowledge/clusters/fire/rules.yaml` holds only
  `rule.ncc.sprinkler-concession-type-of-construction` and `rule.as2118.sprinkler-design`.

## Round 1 response (author, 30 October 2026)

| # | Finding | Response |
|---|---|---|
| 1 | Staged CCs; demolition under the DA needs no CC | **Applied.** New rows: "Prepare early-works documentation and regulated designs", "Construction certificate: early works" and "Construction certificate: main works" (alias "Construction certificate", `fr.construction-certificate`, partial). Site establishment, asbestos removal and demolition now follow the award, dilapidation and hazmat, not a CC. Shoring follows the notice of commencement, which follows the early-works CC. The floor cycle follows the main-works CC. The app's single CC stays a known difference and a new gap. The expectation was not adjusted to the app. |
| 2 | DBP declaration, RAB notice wording | **Applied.** Added "Principal compliance declaration (DBP Act)", which the OC now follows. The notice is renamed "Expected completion notice to Building Commission NSW" with the confirmed window. |
| 3 | Regex traps in base wording | **Applied.** Added a DOMAIN §10 table, six `scenario_inputs` guidance rows, the requested negations, and a gap. One detail: `\bexisting building\b` does not match the plural "existing buildings". The singular does, so the trap stands. |
| 4 | `traditional-procurement` RFPs | **Applied.** RFPs are now `partial`. `adds.rfps` names the expected stage changes for the architect, structural and civil. The `why` covers novation `no` and the boundary. I did not use `removes.rfps`, because the schema reads it as removing the whole RFP; the D&C stage removals are written in `adds.rfps` instead. |
| 5 | Rail adds acoustic twice | **Applied.** Procurement dropped; acoustic and geotech asset-exposure RFP rows added. |
| 6 | Underpinning profile key | **Applied.** `profile: []` with the note; gap added. |
| 7 | Budget-split gap | **Applied** as worded, in the shape and in DOMAIN §6. |
| 8 | Sprinkler clause, inspections, strata bond | **Applied** to the work item, the DOMAIN §2 and §5 text, the inspection rows, the bond rows and the fire-services `must_include`. |
| 9 | Panel depends on the CIV; DRP where constituted | **Applied.** Placeholder phrasing in the scenario input, panel aliases, DRP aliases and "where constituted". |
| 10 | s73 split | **Applied.** "Lodge the Section 73 application through a WSC" follows the determination. "Sydney Water building plan approval" gates the early-works CC. |
| 11 | DNSP earlier | **Applied.** "DNSP connection enquiry and substation design" now follows "Develop the design". |
| 12 | Unscored stages | **Applied.** Mechanical `planning` is an option, with an `app_differs` row. Fire engineering has `s8_specialist` and `s9_peer` as options, and `s8_staging` absent. Electrical `why` reworded. |
| 13 | `complex-facade`; noise variant | **Applied** (a) and (b). The trigger count is now 16, one over the brief's aim of 8 to 15. I kept `noise-affected-site` as a trigger, because the regex trap needs its negation. **Not added:** the optional `ssd-affordable-housing` trigger. It would take the count to 17, and its thresholds are all unverified. It suits a later variant. |
| 14 | Strata bond is a security; strata inspector | **Applied.** |
| 15 | Trees `why` | **Applied.** |
| 16 | Architect `water_asset` delta for `sydney-water-asset` | **Disputed.** `knowledge/rfp/architecture.yaml` `water_asset` has `module: house`, so the row is absent for a Class 2 building whatever `flag.authority_asset` says. Adding it would key a delta the app cannot produce and that is not correct for this type. The structural `authority_asset` stage and the hydraulic diversion row remain. |

**Also applied, from `reference/CONVENTIONS.md`.**
- **C1:** mechanical, electrical, hydraulic and fire services are now novation `option`. Their RFP
  `continuation` stage is now an expected option; it was include. The architect, structural, civil and
  landscape stay `yes`.
- **C2:** `consultant.basix` is kept, because the work is residential BASIX/NatHERS only.

Validator: `ok`. Counts: 16 triggers, 23 packages, 59 cost lines, 103 activities, 22 RFPs, 26 gaps.

## Round 2 (reviewer, 30 October 2026)

I re-read `shape.yaml` and DOMAIN.md after the author's round 1 changes. The validator, run from the repo
root, returns `ok` with exit 0. Counts: 16 triggers, 23 packages, 59 cost lines, 103 activities, 22 RFPs and
26 gaps. Every program `after` names an existing activity, and the graph has no cycle (checked with
read-only Python).

### Round 1 findings: applied?

| # | Status | Note |
|---|---|---|
| 1 Staged CCs | ✔ mostly | Early-works and main-works CCs are in place. Demolition and asbestos removal follow award, dilapidation and hazmat. Shoring follows the notice of commencement, and the floor cycle follows the main-works CC. **One gap remains** (R2-2). |
| 2 DBP declaration, RAB wording | ✔ | The OC now follows "Principal compliance declaration (DBP Act)". The notice row names Building Commission NSW and the confirmed window. |
| 3 Regex traps | ✔ | DOMAIN §10 table, six `scenario_inputs` rows, negations and a gap. The author is right that `\bexisting building\b` does not match the plural. **Correction to my finding 3:** the architecture `water_asset` row is in the house module only, so a false `authority_asset` reading hits only the structural `authority_asset` stage and scale. The revised §10 already says this. |
| 4 `traditional-procurement` RFPs | ✘ form | `built_in.rfps` is `partial` and the `why` is right. But `adds.rfps.must_include` is written as pseudo-strings such as `'stage.dnc_tender: absent'`. The scorer matches a plain string as text in the report (`phrase_text`, `tools/score_shapes.py`), so none of them will ever match. The "absent" items also sit in `must_include`, which inverts them. See R2-1. |
| 5 Rail | ✔ | |
| 6 Underpinning | ✔ | |
| 7 Budget split | ✔ | |
| 8 Sprinklers, inspections, bond | ✔ | E1D6 and E1D4(2) are cited with `verified: true`. Inspection rows follow the cl 61 ruling, `verified: false`. The bond row carries 2%, the 2028 deferral and the 15–18 and 21–24 month windows. AS 2118.1 car park sprinklers are in the fire-services `must_include`. |
| 9 Panel and DRP | ✔ | The phrasing placeholder and aliases are in. Trivial: the `scenario_inputs[flag.panel].note` still says "determined by the local planning panel". Reword it to match the phrasing. |
| 10 s73 split | ✔ | |
| 11 DNSP earlier | ✔ | |
| 12 Unscored stages | ✔ | |
| 13 Complex facade, variants | ✔ | The trigger is well formed and the `complexity_additions[7]` source is correct. It replaces `noise-affected-site` in `variants`. Keeping `noise-affected-site` as a trigger is fine: its new negation tests the trap. |
| 14 Strata bond, inspector | ✔ | |
| 15 Trees `why` | ✔ | |
| 16 Architect `water_asset` | **Dispute upheld** | `knowledge/rfp/architecture.yaml` `water_asset` has `module: house` and is absent for Class 2 whatever the flag says. Finding 16 is withdrawn. The base `expect` value `water_asset: absent` is correct. |

**Conventions.** C1 is applied correctly. Mechanical, electrical, hydraulic and fire services are novation
`option`. Their `continuation` stage is an expected option, and `position.continuation` is O. Their
`must_include` reads "Option: post-novation design (CONVENTIONS C1)". The architect, structural, civil and
landscape stay `yes`. C2 is applied correctly: residential BASIX/NatHERS only, so `consultant.basix`. C1 is
still provisional and waits for the owner's answer.

### New findings

- **R2-1. `shape.yaml triggers[traditional-procurement].adds.rfps`: rewrite as keyed items.**
  - Architect:
    - `must_include`: `{text: "Tender documents", key: stage.tender_traditional}` and
      `{text: "For-construction documents", key: stage.for_construction}`.
    - `must_not_include`: `{text: "D&C tender documents", key: stage.dnc_tender}` and
      `{text: "Post-novation", key: stage.post_novation}`.
    - Keep "Option: design support during construction" as text.
  - Structural:
    - `must_include`: `stage.tender_traditional` and `stage.construction_documents`, keyed the same way.
    - `must_not_include`: `stage.dnc_tender` and `stage.post_novation`.
  - Civil:
    - `must_include`: `stage.detailed_tender` and `stage.for_construction`.
    - `must_not_include`: `stage.dnc_tender` and `stage.post_novation`.
    - Note in the `why` that civil's non-D&C boundary is `schematic → detailed_tender`, so the app prices
      `for_construction` as an option (`knowledge/rfp/civil.yaml boundary[1]`).
  - The app-option caveats move into the `why`, since trigger RFP items have no `app_differs`.

- **R2-2. `shape.yaml program.activities["Bulk earthworks"].after`: add "Appoint principal certifier and give
  notice of commencement".**
  - *Wrong:* bulk excavation is building work under the early-works CC. Only demolition proceeds on the DA
    alone. Today "Bulk earthworks" follows demolition only, so it can start before any CC.
  - "Excavate the basement" is already correct, through shoring.

### Should `ssd-affordable-housing` be added?

**Yes, as a trigger. Not in `variants`.**
- It is the only T02 route that exercises the app's SSD logic: `sk.ssd-dc`, `fr.ssd-eis` and
  `fr.ssd-assessment`, the DA-fees starter dropping for `ssd`, and town-planning complex scale on `ssd`. No
  other trigger reaches that code.
- The Housing SEPP in-fill affordable housing bonus is now a mainstream route for this building type.
- The 15 aim is guidance, and 17 triggers is not too many for a $10–60 m job with this many authorities.
- Write it with:
  - `profile: [hdr.cond.planning]` (value `ssd`);
  - `removes: {procurement: [], cost_plan: ["DA fees and contributions (s7.11 or s7.12)"], program: ["Lodge the
    DA", "Public exhibition", "DA assessment and council requests for information", "Local planning panel
    determination"]}`;
  - `adds.program: ["Prepare the EIS", "SSD assessment"]`, `built_in: {program: "yes", cost_plan: "yes"}`,
    source `knowledge/program/skeletons.yaml sk.ssd-dc`;
  - the affordable-housing share, bonus and SSD CIV threshold all `verified: false`.
- Note in the `why` that the base CIV range makes this the upper end of T02.

### Round 2 verdict: **FIX** (minor)

Two small edits (R2-1 and R2-2) and one wording fix (finding 9 note). Adding the `ssd-affordable-housing`
trigger is recommended. With those, the map is usable as an answer key, pending the owner's review and C1.

## Round 2 response (author, 30 October 2026)

- **R2-1 applied.** `traditional-procurement` RFP items are now `{text, key}`. The absent D&C stages are
  under `must_not_include`, and the app-option caveats are in the `why`: architecture
  `boundary[3]`, civil `boundary[1]`, and structural running to `construction_documents`.
- **R2-2 applied.** "Bulk earthworks" now also follows "Appoint principal certifier and give notice of
  commencement".
- **Finding 9 note reworded.** It now names the local or Sydney district panel, depending on the CIV.
- **`ssd-affordable-housing` added** as a trigger, not a variant, as specified, and DOMAIN §9 now
  lists it. The town-planning RFP is `partial`. Triggers: 17.
