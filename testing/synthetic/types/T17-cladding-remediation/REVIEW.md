# T17 Façade or combustible cladding remediation: independent review

Reviewer: independent agent, 30 October 2026. I changed nothing except this file. No Jev, no network.

## Verdict: **FIX**

The map is right at the root. The base scenario, the systems walk, the consultant team, the program logic,
the RFP stage and position expectations and the app-gap list all hold up against the knowledge and the Go.
The finding that matters most is in the RFP code: the structural `construction_phase` stage is still only an
option for Class 2, because `resolveStages` applies the boundary before `include_when`
(`internal/rfp/resolve.go` l.716–723). The map gets this right.

Before the map is used as an answer key, fix these:

- two triggers whose consequences contradict the code or the map's own prose (CDC, Project Remediate);
- a defects-claim variant that cannot happen on the 2011 base;
- a heritage evidence line that does not set the site's heritage status;
- the hazardous-materials Implication, which DOMAIN describes wrongly;
- owner ruling R5 (BASIX), which the map does not address;
- several base items that are missing.

**Validator:** `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T17-cladding-remediation/shape.yaml`,
run from the repo root, prints `ok …shape.yaml` and exits 0. Counts match DOMAIN.md: 15 triggers, 41 activities,
10 packages, 30 cost lines and 25 gaps.

## Findings

1. **`triggers[cdc-pathway].built_in.rfps: partial` is wrong; the CDC route blocks two RFPs.**
   `bca-certification` lists `cdc` in `unsupported_pathways` (`knowledge/rfp/bca-certification.yaml`;
   `checkPathway`, KNOWLEDGE_INDEX §5.1). The BCA consultant and the certifier share that template, so on a
   CDC **both** RFPs report `pathway_unsupported` and cannot be drafted. The certifier's "CDC assessment and
   determination" item is therefore not built in at all.
   **Change:** set `built_in.rfps: "no"`. In the `why`, add: "Both consultant.bca and consultant.certifier
   RFPs block (bca-certification unsupported_pathways cdc); the town-planning RFP is removed." Add a `gaps`
   entry (area `rfp`, evidence `knowledge/rfp/bca-certification.yaml unsupported_pathways`): "On a CDC the BCA
   and certifier RFPs block; no template drafts a CDC certifier appointment." Say the same in DOMAIN §8 and §9.

2. **`triggers[project-remediate]` contradicts DOMAIN §4 about who supplies the PM.** DOMAIN §4 and §9 say
   the program supplies the PM. The trigger instead *adds* a must_include to the OC's own PM RFP. Under
   Project Remediate the program engaged and paid for the project manager from its panel (verified: false).
   The OC does not run a PM RFP, and it has no PM fee line.
   **Change:**
   - Add `removes: {procurement: [consultant.project_manager], cost_plan: ["Project manager"], rfps: [consultant.project_manager]}`.
   - Drop the `adds.rfps` item.
   - Add the program activity "Program-appointed project manager mobilised (verified: false)".
   - In the `why`, add: "Registration for the program closed to new buildings years ago (verified: false), so a 2026 scenario needs an
     already-enrolled OC." See also my ruling under "Author's open questions".

3. **`triggers[defects-claim]` cannot happen on the 2011 base.** A building work rectification order under
   the RAB Act is limited to work finished within about 10 years, and the HBA statutory warranties run 6 years
   for a major defect (both verified: false). A building finished in 2011 is past both windows by October 2026.
   Your own evidence line ("within the period for a building work rectification order") needs a later year.
   **Change:** add `det.existing_building_year` to the trigger's scenario note with "≥ 2017 for this variant
   (verified: false)". Add the negation "The building was completed in 2011, outside the statutory warranty
   and rectification order periods." A claim that survives on the base is a contract or product-liability
   claim against the panel supplier. Say so in the `why`.

4. **`triggers[heritage-listed].evidence[1]` does not set the site's status.** "Schedule 5 of the LEP lists
   the adjoining building as a heritage item" is a site *in the vicinity of* an item. `det.heritage_status`
   options describe the site itself (`knowledge/determinants.yaml heritage_status`), so a correct reading stays
   `none` or unstated. The Heritage package, `fr.heritage-referral` and the complex RFP scale would then not
   fire. Keeping the line as evidence trains the scorer to reward a wrong reading.
   **Change:** replace it with "The building is listed as a local heritage item in Schedule 5 of the LEP."
   Move the adjoining-item sentence to the §10 trap table: "Vicinity of a heritage item is not a heritage
   status; a heritage impact statement may still be needed (LEP cl 5.10, verified: false)."

5. **DOMAIN §3 (Hazardous materials) and §1 misdescribe the hazmat Implication.**
   `uc.asbestos-cladding-disturbed` (`knowledge/clusters/envelope/unforeseen.yaml`) fires only for
   `envelope.external-walls` with action alter, upgrade, repair or remove. The base item is **replace**, so
   on a pre-2004 building the app raises **nothing** from this record. The shape's `hazardous-materials.why`
   says this correctly; DOMAIN §3 suggests the Implication appears.
   **Change:** in DOMAIN §3, write: "The record needs an alter, upgrade, repair or remove item on the external
   walls and a year before 2004; with the correct `replace` action it does not fire, so even an old building
   gets no hazmat prompt from the cladding item." Add a `gaps` entry (area `knowledge`) with the same text.

6. **Owner ruling R5 (BASIX) is not addressed.** R5 says alterations and additions valued at $50,000 or more
   need a BASIX certificate. DOMAIN §5 says only "No … BASIX involvement on the base", with no reason. A
   multi-million-dollar façade replacement on a building that contains dwellings has to state why R5 does not
   apply: the work is to common property and does not alter a dwelling, so it falls outside the BASIX
   alterations stream (verified: false). Otherwise it must apply.
   **Change:** in DOMAIN §5, add: "R5 (BASIX for alterations ≥ $50,000) is taken not to apply: the work is to
   common property, not to a dwelling (verified: false; confirm on the Planning Portal)." Add it to §12's
   judgement calls. Do not add a BASIX line or package.

7. **R3 is applied but still cited as `verified: false`.** Under R3 the DBP regime applies to Class 2 work,
   including remedial work. Several items still mark that applicability as unverified:
   - `procurement[trade.main_works].why` ("registered building practitioner (DBP Act, verified: false)");
   - the architect `must_include[4]`;
   - the structural `must_include[4]`;
   - the façade `must_include[4]`;
   - the certifier `must_include[0]`;
   - the "Building compliance declaration (DBP Act)" activity `why`.

   **Change:** cite "(DBP Act, owner ruling R3)" for applicability in each of these. Keep `verified: false`
   only on section numbers, practitioner classes and lodgement timing. This is the same change T09 made after
   the rulings.

8. **Missing base items** (all `built_in: "no"`):
   - **Owners' consent to the DA.** The DA needs the OC's consent as owner of the common property. Add "and
     owners' consent to lodge the DA (verified: false)" to the label of the activity "General meeting resolves
     option, design fees and first levy". Add "Owners corporation consent" to that activity's `why`.
   - **Pre-CC conditions of consent.** Add the activity "Satisfy pre-CC conditions of consent (colours and
     materials, dilapidation, LSL)", stage `approvals`, after "DA assessment and determination". Add it to
     the `after` list of "Construction certificate".
   - **Samples and mock-up hold point.** Add the activity "Hold point: panel samples, colour and visual
     mock-up approved", stage `construction`, after "Contract award". Add it to the `after` list of "Install
     non-combustible cladding". Add it to DOMAIN §5's hold points.
   - **Condition surveys cost line.** Add the cost line "Pre-commencement condition surveys of units,
     footpath and neighbours", section `construction`. The activity already exists; the line does not.
   - **Earthing of the metal façade.** Add the cost line "Equipotential bonding and earthing of the
     aluminium cladding and subframe (verified: false)", section `construction`. In the `why`, add: "by the
     builder's electrician; an `electrical.*` item would wrongly add the Electrical consultant."

9. **Interim fire safety measures sit too late in the program.** "Fire engineering risk assessment and
   interim fire safety measures" follows sampling, but councils usually require interim measures as soon as
   the order is received.
   **Change:** split the activity in two:
   - "Put interim fire safety measures in place", after "Receive fire safety order…";
   - "Fire engineering risk assessment and removal-extent advice", after the sampling activity.

   Keep the options report after the second one.

10. **`triggers[swing-stage-access].removes.cost_plan` removes the hoarding too.** Swing stages remove the
    scaffold, but the footpath still needs overhead protection or a hoarding.
    **Change:** split the base line into "Perimeter scaffold, shade cloth and edge protection, incl.
    engineering and certification" and "Footpath hoarding or overhead protection". Remove only the first.

11. **`rfps[consultant.quantity_surveyor]`: `must_not_include` conflicts with an expected position.**
    "Contamination costing" is in `must_not_include`, yet `positions.remediation: I` is expected. That row's
    label is "Defect/remediation/contamination costing" (`knowledge/rfp/cost-planning.yaml positions
    remediation`), so a wording scorer will fail either way.
    **Change:** change `must_not_include[0]` to "Contaminated-land costing", and add an `app_differs` note:
    the remediation row's label also names contamination.

12. **Missing common triggers.** I would expect four more:
    - **`mixed-use-podium`**: "ground-floor shops", "Class 5/6 podium". Changes `det.ncc_class` to `["2","6"]`.
      It adds BCA scope for the podium, and probably a commercial tenancy interface and a lease or access
      notice. It adds no package, because remediation has no baseline.
    - **`staged-funding`**: "the levy will be raised in two instalments", "a strata loan". Adds per-elevation
      stages or separable portions and a GM before each stage. T09 has the same trigger.
    - **`balcony-waterproofing-interface`**: "failed membrane upturns found behind the spandrel ACP". Adds
      `envelope.external-waterproofing` repair, a hold point and the waterproofing licence.
    - **`order-compliance-no-consent`**: "council advised that no development consent is needed for works
      required by the fire safety order". Sets `hdr.cond.planning: exempt` and `fr.exempt-confirmation`,
      and removes the DA fragments, the Town Planner and DA fees. DOMAIN §5 names this route but nothing
      tests it (verified: false).

13. **`triggers[window-replacement].evidence[1]`: the wording is not realistic.** "The glazed spandrel panels
    contain a PE core" mixes two products.
    **Change:** "The ACP spandrel infill panels within the window framing are replaced with the windows."

14. **DOMAIN §5 needs a few authority corrections** (all verified: false):
    - Fire and Rescue NSW can also issue fire safety orders, not only council.
    - Council does not "sign off" an order. It confirms compliance or revokes the order.
    - The 2018 ACP ban covers Class 2, 3 and 9 buildings of two or more storeys, and Class 5–8 buildings of
      three or more storeys, not only Class 2.
    - Cladding registration covered Class 2–9 buildings of two or more storeys.
    - "SSMA s 106, s 108" in §5 needs its own `verified: false`.
    - Add a line: "RAB Act expected completion notice before the OC application: whether it applies to
      remedial work on an existing building (verified: false)."

15. **DOMAIN §12: add the reviewer's rulings** (see the next section) to the list of judgement calls, and
    record that they came from review, not from the owner.

Note for the orchestrator, outside T17: the code confirms this map's reading that structural
`construction_phase` is an option for Class 2 (`resolve.go` l.718 runs before l.720). The premise of T09's
round 1 finding 6, that it "arrives through complex scale", is therefore wrong for that template. T09 may
need a one-line correction.

## Rulings on the author's open questions

| question | ruling |
|---|---|
| DA as base, not CDC | **Keep DA.** A whole-façade redesign that changes the appearance of an 8-storey RFB is most often a DA. A CDC under the Codes SEPP (the Fire Safety Code, verified: false) and the "no consent needed for work an order requires" route are real alternatives. Both stay triggers (finding 12 adds the second). |
| Owner-funded base | **Keep owner-funded.** Project Remediate stopped taking new buildings years ago (verified: false), so a 2026 job is usually levy- or loan-funded. Fix the trigger's PM consequence (finding 2). |
| Full removal to DTS | **Keep.** It is the standard NSW outcome for high-risk PE-core ACP and EPS. Partial retention under a performance solution stays a trigger. |
| Fire safety order regime | Division 9.3 and Schedule 5 of the EP&A Act (development control orders, including fire safety orders, with a notice of intention first) matches my understanding. **Keep verified: false.** Add that FRNSW can also issue orders (finding 14). |
| Project Remediate details | 10-year interest-free loans; program-appointed PM, design and builder panels; administered by the Office of Project Remediate, now in Building Commission NSW. These match my recollection but are unconfirmed. **All verified: false.** |
| DBP lodgement timing | The Act requires the regulated designs and declarations to be lodged before the building work starts. In practice the certifier assesses the same designs for the CC, so "regulated designs before the CC" is a safe program dependency. The legal deadline is "before work", not "before the CC". **Timing verified: false.** The building compliance declaration comes before the OC application; your order is right. |
| FRNSW referral | The initial and final fire safety reports go to FRNSW only for performance solutions against the fire safety provisions the DCFS Regulation 2021 lists. I cannot confirm that a cladding-only performance solution (spread of fire) is among them. **Keep verified: false.** Write it in the trigger as "where the regulation requires". |
| NCC 2022 clause numbers | C2D10 (non-combustible building elements) and CV3 (AS 5113) match my reading of NCC 2022. Other numbers I recall but cannot confirm: C2D14 (ancillary elements), C2D15 (bonded laminated panels) and the sarking concession inside C2D10. Table C2D2 gives Type A for a Class 2 building with a rise of 4 or more storeys. No NCC 2022 text is in the repo, and the app's rule uses 2019 numbering with `clause_verified: false`. **All stay verified: false.** No unmarked rule number was found. |

Prices, rates and durations: none are quoted. Durations defer to `knowledge/program/durations.yaml`. ✔

## Built-in claims checked

| # | claim | result |
|---|---|---|
| 1 | procurement `consultant.facade` yes, `complexity_additions[7]` (facade_type complex; base phrasing names a ventilated rainscreen, which matches the `rain ?screen` trigger) | ✔ |
| 2 | procurement `consultant.town_planner` yes, `complexity_additions[0]` | ✔ |
| 3 | procurement `trade.main_works` partial, `ProposeHeadContract` (accepted items only) | ✔ |
| 4 | cost: DA fees `lines[3]` (da-fees, planning da) | ✔ |
| 5 | cost: CC fees `lines[4]` | ✔ |
| 6–12 | cost: PM, architect, fire, structural, BCA, certifier, QS fee lines partial (Suggest fee loop; no package rule) | ✔ |
| 13 | cost: Facade fee yes (adopted on prepare) | ✔ |
| 14 | cost: Town Planner fee yes | ✔ |
| 15–18 | cost: subframe, cladding, insulation, flashings → `04` partial | ✔ |
| 19 | cost: cavity barriers → `06` partial (`fire-passive` code 06) | ✔ |
| 20 | cost: provisional sums → `03` from the investigate item (`costSuggestInput` has no action filter) | ✔ |
| 21–24 | cost: design contingency, construction contingency, escalation, exclusions `lines[0,1,2,6]` | ✔ |
| 25 | not expected: P06 `lines[5]` always on; Hydraulic, Mechanical and Electrical via `scope_additions[3,2,4]` | ✔ |
| 26 | program: opening-up partial (`investigationRows`) | ✔ |
| 27–30 | program: `fr.da-lodgement`, `fr.da-exhibition`, `fr.da-assessment`, `fr.construction-certificate` yes | ✔ |
| 31 | program: `fr.contract-award` partial (new, extend, refurb) | ✔ |
| 32 | program: `fr.insulation-2` partial (class 2; new or extend) | ✔ |
| 33 | program: `fr.facade-2` partial (new or extend; action new or replace) | ✔ |
| 34–37 | program: `fr.occupation-certificate`, `fr.practical-completion`, `fr.handover`, `fr.defects-liability-end` partial | ✔ |
| 38 | rfp architect: boundary ends at `interiors`, so tender, bidding, regulated design, for-construction and support are options; complex scale (Class 2) gives BIM I and typical interiors I; `extra_declarations` O (the `stage` key counts option stages) | ✔ |
| 39 | rfp fire: boundary s1–s4; s3 and s4 ask on `fe_solutions`; s5–s8 options; `remediation` always O; `occupants` and `hazards` off on flags no | ✔ |
| 40 | rfp structural: `existing_assessment` on remediation; `construction_phase` option despite complex scale (boundary first, `resolve.go` l.716–723); dilapidation X; `inspection_schedule` I | ✔ |
| 41 | rfp BCA: a1–a3 ask on `bca_advisory`; a4 and b1–b4 options after the boundary | ✔ |
| 42 | rfp certifier: same template, cannot be told apart | ✔ |
| 43 | rfp town planner yes; no differences (scale assumed simple: objections unknown) | ✔ |
| 44 | rfp QS: s2–s4 ask on client flags; s5–s7 options after `s4_design`; s8 advisory only | ✔ |
| 45 | trigger performance solution: rfps partial (s3_brief) | ✔ |
| 46 | trigger concealed damage: rfps partial (`strengthening` O), cost partial (03, 05) | ✔ |
| 47 | trigger ACP balustrade: cost partial (13 child "Compliant balustrades and barriers"), rfps partial (`metalwork` O) | ✔ |
| 48 | trigger façade services: cost partial (08, 09, 10) | ✔ |
| 49 | trigger window replacement: cost partial, program partial (`fr.windows-2` gated) | ✔ |
| 50 | trigger fire upgrades: procurement partial (Fire Services via `scope_additions[1]`; no trade suggestion); program partial (`fr.sprinklers-2`, `fr.detection-2` gated); rfps yes (`fire-services` design inside its boundary) | ✔ |
| 51 | trigger boundary wall: rfps partial (`land-surveying`); procurement no (no remediation baseline) | ✔ |
| 52 | trigger heritage: procurement, program (`fr.heritage-referral` has no work-type gate) and cost yes | ✔ (but finding 4 on the evidence) |
| 53 | trigger CDC: program yes (`fr.cdc-application`; CC fragment off for cdc; DA fees dropped) | ✔ |
| 54 | trigger CDC: rfps partial | ✘ no; both bca-certification RFPs block (finding 1) |
| 55 | trigger D&C: program partial (`fr.novation`), rfps yes (architecture `dnc_tender`, `post_novation`) | ✔ |
| 56 | expected_derived: `det.existing_building` from the RFP drivers; class and Type A not derived | ✔ |
| 57 | trigger hazardous materials: `why` says `uc.asbestos-cladding-disturbed` excludes replace | ✔ (DOMAIN §3 contradicts it: finding 5) |

Spot-checked `no` claims (all ✔: the knowledge has nothing):

- `consultant.project_manager` and `consultant.facade` have no RFP template (no `knowledge/rfp` file names them);
- Fire Engineer and QS have no package rule (`package_defaults.yaml`);
- scaffolding has no system (`structure.temporary-works` excludes it);
- `if.fire-facade-slab-edge` has no `hold_point` and is curtain wall only;
- there is no remediation skeleton or budget split;
- no knowledge file mentions a fire safety order or Project Remediate;
- there is no long service levy starter line.

All `[K:]` record ids cited in DOMAIN.md exist. The `implied_by` traps in §10 match
`knowledge/profile/scope_defaults.yaml` (wet-room, kitchen, bedroom, stair, party-wall, strip-out, masonry,
energy-rating).

## Round 1 response (author, 30 October 2026)

All 15 findings applied; none disputed.

1. CDC: `built_in.rfps: "no"`, `why` names both blocked RFPs; new `rfp` gap; DOMAIN §8, §9, §11.
2. Project Remediate: `removes` the PM package, fee line and RFP; `adds.rfps` dropped; "Program-appointed project manager mobilised (verified: false)" added; `why` notes closed registration; DOMAIN §4, §9.
3. Defects claim: `why` needs a building year of 2017 or later; negation for the 2011 base added; product-liability note.
4. Heritage evidence replaced with a site listing; the vicinity trap moved to DOMAIN §10.
5. DOMAIN §3 rewritten for the `replace` exclusion; new `knowledge` gap.
6. R5 addressed in DOMAIN §5 and §12; no BASIX line or package.
7. R3 cited for applicability in all six places; verified: false kept on classes, sections and timing only.
8. Owners' consent in the GM activity, pre-CC conditions activity, samples and mock-up hold point, condition-survey and earthing cost lines.
9. Interim measures split from the risk assessment and moved to straight after the order.
10. Scaffold and hoarding split; swing stage removes only the scaffold.
11. QS `must_not_include` now "Contaminated-land costing"; `text.remediation` app_differs added.
12. Four triggers added: `mixed-use-podium`, `staged-funding`, `balcony-waterproofing-interface`, `order-compliance-no-consent`.
13. Window evidence reworded.
14. DOMAIN §1 and §5 authority corrections (FRNSW orders, compliance or revocation, ban and registration classes, SSMA verified: false, RAB Act line).
15. Reviewer rulings recorded in DOMAIN §12 as review rulings, not owner rulings.

New counts: 19 triggers, 10 packages, 33 cost lines, 44 activities, 9 RFPs, 27 gaps. Validator: ok.

---

# Round 2 (re-review after the author's fixes, 30 October 2026)

**Validator:** `ok`, exit 0. Counts match the author's note and DOMAIN.md: 19 triggers, 10 packages,
33 cost lines, 44 activities, 9 RFPs and 27 gaps. Every `after` in the program names an existing
activity.

| # | applied | note |
|---|---|---|
| 1 | ✔ | CDC: `rfps: "no"`, both blocked RFPs named, new `rfp` gap. |
| 2 | ✔ | Project Remediate removes the PM package, fee line and RFP; mobilisation activity added (verified: false); closed registration noted. |
| 3 | ✔ | Defects claim needs a building year of 2017 or later (verified: false); the 2011 negation and the product-liability note are added. |
| 4 | ✔ | The heritage evidence is now a site listing; the vicinity trap is in DOMAIN §10. |
| 5 | ✔ | DOMAIN §3 and the new `knowledge` gap state the `replace` exclusion. |
| 6 | ✔ | R5 is addressed in DOMAIN §5 and §12 (verified: false); no BASIX line. |
| 7 | ✔ | R3 is cited for applicability in the procurement entry and the four RFP items; classes, sections and timing stay verified: false. |
| 8 | ✔ | Owners' consent, pre-CC conditions (now feeding the CC), the samples and mock-up hold point (feeding cladding install), and the condition-survey and earthing lines are all added. |
| 9 | ✔ | Interim measures now follow the order directly; the risk assessment follows sampling and feeds the options report. |
| 10 | ✔ | Scaffold and hoarding are split; the swing stage removes only the scaffold. |
| 11 | ✔ | "Contaminated-land costing"; `text.remediation` app_differs added. |
| 12 | ✔ (with R2-1 to R2-3) | All four triggers added. |
| 13 | ✔ | Window evidence reworded. |
| 14 | ✔ | FRNSW orders, compliance or revocation, ban and registration classes, SSMA verified: false and the RAB Act line are all in DOMAIN. |
| 15 | ✔ | DOMAIN §12 records the rulings as the reviewer's, not the owner's. |

**New triggers checked against the code:**

- `mixed-use-podium`: `det.ncc_class` 2 and 6 derive `flag.mixed_use: yes` (`internal/rfp/drivers.go`
  l.304–306). That turns on the fire `occupants` row and the architecture `mixed_use` module, as claimed.
  It adds no package, because remediation has no baseline. ✔
- `staged-funding`: `flag.staged` yes does switch on architecture `brief_masterplan`. ✔
- `balcony-waterproofing-interface`: `envelope.external-waterproofing` repair gives the `04` line. ✔
- `order-compliance-no-consent`: `exempt` adds `fr.exempt-confirmation` and drops the DA and CC fragments,
  the DA fees line and the Town Planner. Certifier B stages need da or ssd. ✔

**New legal numbers:** none are unmarked. The 2017 year, the RAB Act and HBA periods, the Fire Safety Code
and the s 106 and s 108 references all carry verified: false.

## Items still open (minor; none changes the base scenario)

- **R2-1. `triggers[mixed-use-podium]`: one app effect is missing.** A class 6 part makes the architecture
  `interiors` stage true (`when: class 3, 5, 6, 9…`). It is the boundary end, so it is **included**, with
  `base_interiors` I: a false positive for a façade job.
  **Change:** add to the `why`: "architecture interiors stage and base_interiors become included (class 6);
  the expected state is absent."
- **R2-2. `triggers[staged-funding].built_in.rfps: partial` → `"no"`.** The expected must_include items are
  for `consultant.project_manager`, which has no template. `brief_masterplan` is a false positive, not a
  partial match. This is the same case as T09 finding 12.
  **Change:** set `rfps: "no"`, and keep the `brief_masterplan` effect in the `why` as a wrong addition.
- **R2-3. `triggers[order-compliance-no-consent]`: the certifier and OC are unresolved.** Without consent
  there may be no CC, no principal certifier and no OC (verified: false). The base activity "Final
  inspection, fire safety certificate and occupation certificate" and the certifier package would then go
  too. P05 CC fees stays suggested, because it is always on.
  **Change:** add `authority: ["Confirm whether a CC, principal certifier and OC are still required
  (verified: false)"]`. In the `why`, add that P05 stays in the app either way.

## Round 2 verdict: **PASS**

All 15 findings are applied, and the four new triggers match the code. The three open items are wording
and `built_in` corrections on variants. Fix them when convenient; the map is usable as an answer key.

## Round 2 response (author, 30 October 2026)

- R2-1 applied: the `mixed-use-podium` why records the architecture interiors stage and base_interiors as included (class 6); expected absent.
- R2-2 applied: `staged-funding` rfps set to "no"; brief_masterplan is kept in the why as a wrong addition.
- R2-3 decided: with no consent there is no CC, principal certifier or OC. The certifier package, fee line, RFP and the OC activity are removed. A final fire safety certificate and council's compliance confirmation are added. P05 stays in the app. All verified: false.

Validator: ok.
