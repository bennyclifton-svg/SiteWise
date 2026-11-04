# T14 Retail or café fit-out: independent review

Reviewer: independent agent (senior NSW PM view), 30 October 2026. Brief: `testing/synthetic/briefs/02-review.md`.
Files reviewed: `DOMAIN.md`, `shape.yaml`. Neither was edited.

## Verdict: **FIX**

The map is right at the root. The base is the right size, the over-procurement story is correct, and the
gap list is mostly accurate. One app prediction is wrong: the certifier RFP's predicted states. One owner
ruling is missed: R3 on the riser variant. One standard NSW authority step is missing: the Sydney Water
Tap in approval. Several program `after`/`stage` entries contradict the map's own logic. The changes below
are specific and none needs a redo.

## Rulings on the author's open questions

1. **Kitchen exhaust and plumbing subcontractors design and certify their own work, with no engineers in
   the base. Right.** On a single-tenancy café whose hood discharges through its own roof, the client does
   not appoint a mechanical or hydraulic consultant. The exhaust contractor provides the AS 1668.1/1668.2
   design and the certificate (verified: false). Sometimes that contractor's own engineer signs it, but it is
   still the contractor's design, not a client package. Plumbing and drainage sit outside the construction
   certificate in NSW: the licensed plumber, Fair Trading and Sydney Water deal with them (Plumbing and
   Drainage Act 2011; verified: false). Sydney Water sizes the arrestor through the trade wastewater permit.
   The electrician designs and installs. Three things follow, and the map has to show them (findings 4, 9
   and 10):
   - the DA still needs the exhaust discharge details, so someone has to produce them before lodgement;
   - the CC waits for the subcontractors' design certificates, which the map already shows;
   - the app cannot hold a design-and-construct subcontract under a traditional route.
2. **DA, not CDC, as the base pathway. Right.** A shop becoming food and drink premises is a change of use.
   With a new roof exhaust, a new in-ground arrestor and new signage, a Sydney practitioner usually lodges a
   DA. The council's environmental health officer also wants to review the kitchen and the plan of
   management at DA. The code route exists, which is why `cdc-pathway` is a variant, and the map says so.
   Keep DA as the base. Keep the Codes SEPP references `verified: false`. Finding 11 fixes how the CDC
   variant is modelled.
3. **Every Class 6 clause and standard number is `verified: false`. Right, and required** by AGENTS.md
   rule 5 and by `OWNER_RULINGS.md`. The map also correctly declines to stretch R7 (Class 1 and 10
   inspections) to Class 6. Two strings are missing the marker (finding 14).
4. **Is the base too lean? No, it is right.** The base has five packages: interior designer, certifier,
   optional town planner, usual hazmat surveyor and the shopfitter. That is what a 120 m² own-roof café
   needs. The only missing party is the kitchen designer or equipment supplier. `DOMAIN.md` §4 calls it
   "always", but it is absent from `shape.yaml` (finding 7). Structural, acoustic and an extra-toilets case
   belong as triggers, not in the base (finding 13).

## Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T14-retail-cafe-fitout/shape.yaml`
→ `ok` (exit 0).

## Findings

1. **`shape.yaml` `rfps[1]` (certifier) `app_differs`; `DOMAIN.md` §8 and §11.6; `gaps` rfp entry 2. The
   predicted app output is wrong.** The map says the boundary "falls back to a1..a3 without the client flag".
   It does not:
   - `resolveBoundary` (`internal/rfp/resolve.go`) sets a boundary only when a `when` is **yes**.
   - With `bca_advisory` and `bca_certification` both unknown, no boundary is set (start = end = −1).
   - So `resolveStages` gives every stage, a1–a4 and b1–b4, `Ask`, not `Option`.
   - The answer key `data/eval/rfp/answer-keys/bca-unknown-decisions.yaml` confirms this: all eight stages
     are in `stages_ask`.
   - The app also raises blocking checks `fact:flag.bca_advisory`, `fact:flag.bca_certification`,
     `fact:flag.bca_existing_assessment` and the other `bca_*` flags. `expect.blocking: []` with no
     `app_differs` entry hides them.

   Changes:
   - Set `app: ask` on the four b-stage rows.
   - Add `{key: stage.a4_construction, app: ask, expect: absent}`.
   - Reword each `why` to "no boundary resolves while bca_advisory and bca_certification are unknown, so
     every stage asks".
   - Add a note that the app's blocking list holds the `fact:flag.bca_*` checks.
   - Fix the same sentence in DOMAIN §8, §11.6 and the gaps entry: "defaults to advisory" should read
     "every stage asks until the client picks Package B".

2. **`shape.yaml` trigger `exhaust-riser-through-building`. It misses owner ruling R3.** Apartments above
   make the building Class 2 in part (the trigger already lists `det.ncc_class`). R3 says the DBP Act's
   regulated designs and compliance declarations apply to Class 2 buildings. A fire-rated riser through the
   residential floors is work on the Class 2 part. Whether a ground-floor Class 6 fit-out alone triggers the
   Act is uncertain.

   Changes:
   - Add `proj.regulated_design` and `hdr.cond.stakeholder_complexity` (`strata`) to the trigger's
     `profile`.
   - Add to `adds.authority`: "DBP Act regulated designs and compliance declarations for work affecting the
     Class 2 part (R3; whether a ground-floor fit-out alone is caught: verified: false)".
   - Add to `adds.program`: "Registered design practitioner declarations before the CC".
   - Note in `why` that the scorer should see the regulated-design RFP rows change when
     `proj.regulated_design` becomes yes.

3. **`DOMAIN.md` §5 authorities; `shape.yaml` `program`, `cost_plan`. The Sydney Water Tap in building
   plan approval is missing.** Sydney councils' DA consents routinely require the approved plans to go
   through Sydney Water Tap in before the CC is issued. A new in-ground grease arrestor and saw-cut drainage
   in an old strip make this likely (verified: false). This step is separate from the trade wastewater
   permit.

   Changes:
   - Add an authority bullet.
   - Add the activity `{label: "Sydney Water Tap in building plan approval", stage: procurement, after:
     ["DA assessment and determination", "Prepare fit-out documentation"], built_in: "no"}`.
   - Add it to the `after` list of "Construction certificate".
   - Add the cost line `{line: "Sydney Water Tap in approval fee", section: fees_and_charges, built_in:
     "no"}`.

4. **`shape.yaml` program, approval documentation; `DOMAIN.md` §5 "DA documents". The DA needs exhaust
   discharge details, but in the map the exhaust design first exists after award.** The first exhaust
   activity is "Contractor's kitchen exhaust and plumbing design certificates", after "Award the contract".
   Yet the DA documents list "kitchen exhaust discharge details". In practice the kitchen designer or an
   exhaust contractor engaged early gives a concept: hood size, duct route, fan position, discharge height
   and distance to neighbours.

   Change: add `{label: "Kitchen exhaust concept and discharge location for the DA", stage: concept_design,
   after: ["Commercial kitchen layout and equipment schedule"], built_in: "no", why: "Kitchen designer or
   exhaust contractor; council assesses odour and discharge at DA"}`. Then add it to the `after` list of
   "Prepare DA drawings, …".

5. **`shape.yaml` program. Two activities sit in a stage that comes before their predecessors.** The
   skeleton order is approvals → construction_documentation → procurement.
   - "Construction certificate" (stage `approvals`) is after "Contractor's … design certificates" (stage
     `procurement`).
   - "Apply for the Sydney Water trade wastewater permit" (stage `approvals`) is after "Prepare fit-out
     documentation" (stage `construction_documentation`).
   - A scorer reading stage membership gets a contradiction.

   Changes:
   - Move "Construction certificate" to `stage: procurement`. Keep `built_in: "partial"` and keep the `why`,
     which says the app puts it in approvals.
   - Move the permit application to `stage: construction_documentation`.
   - "Pay long service levy" can move with the CC to `procurement`.

6. **`shape.yaml` program, "Paint the fit-out" and "Lay floor finishes"; gap "Floor finishes follow
   painting". The key contradicts its own gap.** The gap says the coved kitchen floor goes down "before
   equipment and joinery plinths". The expected logic is install joinery → paint → floors, so the floor
   comes after the joinery.

   Change: set "Lay floor finishes" `after: ["Waterproof amenities", "Install ceilings"]`. Add "Lay floor
   finishes" to "Install joinery" `after`. Leave "Paint the fit-out" after joinery and ceilings. Note in
   `why` that the app orders floors after painting (`fr.floors-fitout` attach).

7. **`shape.yaml` `procurement`. The kitchen designer is missing, though `DOMAIN.md` §4 says "always".**
   The validator accepts `discipline: none`.

   Change: add `{discipline: none, kind: supply, need: usually, built_in: "no", why: "Commercial kitchen
   designer, usually the equipment supplier: layout, equipment schedule, hood size and services loads; no
   discipline id (data/intake/disciplines.json); supplier.* rows are refused by DisciplinePackage"}`. Change
   DOMAIN §4's "always" to "usually (often the supplier)".

8. **`shape.yaml` `not_expected_procurement` Mechanical, Hydraulic, Electrical; gap procurement 1;
   `DOMAIN.md` §11.1. Part of the cause is not cited.** Once documents establish the base work items,
   `scope_additions[2]` (any `mechanical.*`), `[3]` (any `hydraulic.*`) and `[4]` (any `electrical.*` or
   `fire-active.emergency-lighting`) suggest the same three again
   (`knowledge/works/package_defaults.yaml`). Fixing the refurb baselines alone would not remove them.

   Change: add the scope-addition anchor to each `source`/`why`, as was done for Access. Add to the gap:
   "scope additions re-suggest them from any services item, so a size or D&C test is needed in both rule
   sets".

9. **`shape.yaml` `procurement` `trade.main_works`; `gaps`. The D&C subcontract design cannot appear in
   the app under a traditional route.** `CheckGaps` (`internal/procurement/gaps.go`) raises a `design` gap
   for new or altered items. `ProposeHeadContract` (`internal/procurement/proposed.go`) drops design lines
   when `!BuilderDesigns(route)`, and `traditional` returns false. So once the user removes the Mechanical
   and Hydraulic consultants, as this key expects, the exhaust and trade-waste design belong to no one.

   Change: add to `trade.main_works.why`: "the proposal carries install lines only; design of
   mechanical.local-exhaust, hydraulic.trade-waste and electrical stays a gap". Add a gap entry `{area:
   procurement, what: "Traditional route with design-and-construct services subcontracts cannot be
   expressed: design gaps are never proposed into the head contract", evidence:
   "internal/procurement/proposed.go ProposeHeadContract; internal/procurement/gaps.go CheckGaps"}`.

10. **`shape.yaml` work item `interiors.ffe` and cost line "Owner supply items". There is a double-count
    risk.** The item is in the works, so it rolls into "Interiors and finishes", and the same equipment also
    appears as an owner-supply line. Profile rows allow `sys.interiors.ffe.provider` = `owner`
    (`internal/httpapi/profile.go`, options contractor/owner/landlord/others), but the validator rejects
    `sys.*` in `scenario_inputs`.

    Change: in the work item's `why`, say the scenario documents must state "supplied and installed by the
    tenant's kitchen supplier, outside the contract" (provider owner). Add a gap: "provider owner does not
    stop an FF&E item from being costed in Interiors and finishes" (or confirm it in the harness). Map
    §10's "Landlord's works" trap to `.provider: landlord` in the same way.

11. **`shape.yaml` trigger `cdc-pathway`. Two removals are wrong.**
    - A CDC application also needs the land owner's consent, so the landlord's signature stays. Do not
      remove "Landlord signs owner's consent for the DA". Relabel it in `adds.program` as "Landlord signs
      owner's consent for the CDC".
    - `cc-fees` is always produced (`knowledge/costs/starter_lines.yaml applies_when: true`). T03 aliases
      it to the CDC fee rather than removing it. Remove "Construction certificate and certifier fees" from
      `removes.cost_plan`. Change `adds.cost_plan` "CDC fee" to `{line: "CDC and occupation certificate
      fees", aliases: ["Construction certificate and certifier fees"]}` (aliases only if the trigger schema
      allows; otherwise say so in `why`).
    - Also add to `why`: under a CDC the exhaust and arrestor design must be complete before determination,
      so the subcontractors' design moves before tender (early exhaust contractor or a mechanical engineer).

12. **`shape.yaml` triggers `existing-food-premises`, `step-at-entry`, `power-supply-upgrade`,
    `bca-upgrade-on-change-of-use` (`det.existing_building`), `asbestos-found`
    (`det.existing_building_year`) and `grease-arrestor-location-constrained` (`det.sewer_connection`).
    The `profile` keys do not change.** Each key already holds its base value, which stays true, 1965 or
    reticulated. `sewer_connection` options are reticulated, onsite_treatment and pressure_sewer, none of
    which is a pumped arrestor. A scorer would see no profile delta.

    Change: set `profile: []` (the T03 convention) and say "no profile key" in `why`.

13. **`DOMAIN.md` §9 and `shape.yaml` `triggers`. Common café triggers are missing.** Add:
    - `residential-neighbours-acoustic`. Evidence: "dwellings adjoin the rear lane"; "council requires an
      acoustic report for the roof exhaust fan and trading hours". Negation: "no residential receivers
      within …". Adds: `consultant.acoustic` (never suggested), "Acoustic and odour report for the DA",
      cost line. Today this case is only reachable through the riser trigger, but a single-storey strip
      with houses behind is the common case.
    - `roof-cannot-carry-exhaust-fan`. Evidence: "the existing timber roof framing is not adequate for the
      fan and canopy hangers"; "structural engineer to certify the fan support". Adds:
      `consultant.structural`, `structure.alterations` alter, "Structural certification of fan and canopy
      support". Cite `uc.kitchen-canopy-heavier-than-existing-roof-framing` (Implication only). DOMAIN §1
      names this risk, but there is no trigger for it.
    - `patron-numbers-need-more-toilets`. Evidence: "60 seats"; "the certifier requires separate male and
      female patron toilets". Adds sanitary work items and cost lines; NCC Part F4 `verified: false`;
      `rule.ncc.sanitary-facility-quantities` gates on `det.occupant_load`, which cannot be set.
    - The negation `landlord-asbestos-register-covers-tenancy` drops `consultant.hazmat`. That turns the
      base's "usually" into a testable variant.

    Also add a keyword trap to §10: "separating wall" or "common wall" fires `implied_by party-wall`
    (`knowledge/profile/scope_defaults.yaml`), which adds `fire-passive.compartments`,
    `interiors.acoustic-separation` and `site.adjoining-structures`. Fit-out documents use these words for
    the walls between tenancies.

14. **`shape.yaml` `rfps[0].must_include` "Food premises finishes to FSANZ Standard 3.2.3 and AS 4674", and
    `scenario_inputs` det.heritage_status note "s10.7". These numbers lack `verified: false`.** Change the
    RFP text to "Food premises finishes to FSANZ Standard 3.2.3 and AS 4674 (verified: false)", or drop the
    numbers from the phrase. Add "(verified: false)" after s10.7.

15. **`shape.yaml` program "Procure and deliver the tenancy switchboard". `built_in: "yes"` is overstated.**
    `fr.switchboard-major` has `stage: construction`. The key places the activity in `procurement`, and its
    own `why` says the app does not. Change it to `built_in: "partial"`.

16. **`shape.yaml` program "Pay long service levy", cost line "Long service levy". These are
    unconditional, but the levy applies only above a threshold** (verified: false). The base sits in the
    lower half of $0.1–1M and may fall below it. Change both `why`s to "only when the cost of works reaches
    the levy threshold; the scenario budget decides". Alternatively, tell the scenario author to state a
    budget that clearly clears the threshold.

17. **Minor.**
    - **Certifier timing.** `DOMAIN.md` §4 and `procurement[1]`: the certifier is engaged at "design
      development". On a change of use, the B1 preliminary assessment (fire safety and access upgrade risk)
      should come before DA lodgement. Change the stage to concept.
    - **Shopping-centre hold point.** The trigger `shopping-centre-tenancy` adds "Sprinkler pressure test
      before ceilings". The app's row is labelled "Hold point: a recorded pressure test of the sprinkler
      pipework" (`if.sprinkler-pressure-test-before-ceilings`). Add that string as an alias so the scorer
      matches it.
    - **Bin store.** `site.waste-storage` has no fragment and no program activity. Add "Build the bin
      store" (`built_in: "no"`) after "Establish the site".

## Domain accuracy: what is right

These points were checked and need no change:
- The NSW authorities are right: DA, CC, PC appointment and notice of commencement, the fire safety
  schedule, the final fire safety certificate, the OC for the new use, the Sydney Water trade wastewater
  permit, food business notification, Fair Trading plumbing compliance and the landlord's consent.
- R3 and R4 are correctly ruled out for the base, and R7 is not misapplied.
- No price, rate or duration is invented. Durations defer to `durations.yaml`.
- The consultant exclusions (Architect, BCA, Mechanical, Hydraulic, Electrical, Fire Services, Access) are
  right for the base, and each has a trigger that brings it back.
- Kitchen-hood wet-chemical suppression is correctly left out of the base.

## Testability

- The evidence phrasings read like real briefs, SEEs and certifier letters, and include indirect forms
  ("residents above", "junction too shallow; pumped", "Shop 14 in the centre").
- The negations are realistic.
- The consequences are visible in the app outputs, except `liquor-licence` (correctly flagged as
  zero-delta and kept out of the variants) and the program-only items noted above.
- The scenario author must state the presence and action for `hydraulic.trade-waste`,
  `mechanical.local-exhaust` and `site.demolition` (remove). Refurb starts empty, and nothing implies trade
  waste. The map says so in §10 and §11.3.

## Built-in claims checked

Procurement and not-expected procurement (against `package_defaults.yaml`, `type-predictions.md` T14 and
`KNOWLEDGE_INDEX.md` §2):

| claim | result |
|---|---|
| Interior Design yes, baselines[4] (commercial refurb) | ✔ |
| Certifier yes, baselines[2] | ✔ |
| Town Planner yes, complexity_additions[0] (planning da) | ✔ |
| trade.main_works partial, ProposeHeadContract only on accepted items | ✔ (design lines also dropped: finding 9) |
| App suggests Architect (b[2]), BCA (b[3]), Fire Services (b[3]) | ✔ |
| App suggests Mechanical b[3], Hydraulic b[2], Electrical b[2] | ✔ (scope_additions[2]–[4] also fire: finding 8) |
| App suggests Access b[3] and scope_additions[5] | ✔ |
| Hazmat `no` (no package rule) | ✔ (spot-check of a `no`) |
| Kitchen designer `no` (no discipline id) | ✔ (spot-check of a `no`; only trade.joinery_kitchens) |

Cost plan (against `starter_lines.yaml`, `elements.yaml`, `budget_splits.yaml`, `cost_budget.go`):

| claim | result |
|---|---|
| DA fees and contributions yes (da-fees, planning da) | ✔ |
| CC and certifier fees yes (cc-fees, always) | ✔ |
| Service authority partial | ✔ |
| Interior designer, Certifier, Town planner fee lines yes | ✔ |
| Strip-out partial (folded into code 01) | ✔ |
| Codes 01, 04, 05, 06, 07, 08, 09, 10, 11 lines yes | ✔ |
| Joinery split partial (needs trade.joinery_kitchens) | ✔ |
| Access children 13 ×3 labels | ✔ |
| commercial_fitout: no site or envelope share, hydraulic 2 % | ✔ |
| Design and construction contingency, escalation, exclusions yes | ✔ |
| Owner supply partial (applyBriefFigures "Owner supply: …") | ✔ |
| Not-expected F01–F09 labels | ✔ |
| Sydney Water and long service levy lines `no` | ✔ (spot-checks of `no`) |

Program (against `fragments.yaml`, `skeletons.yaml`, `generate.go`):

| claim | result |
|---|---|
| Skeleton sk.refurb-traditional-da | ✔ |
| fr.existing-conditions (refurb): measure, hazmat, trace | ✔ |
| fr.concept-design-fitout, fr.design-development-fitout (5, 6, 9b refurb) | ✔ |
| fr.da-lodgement, fr.da-exhibition, fr.da-assessment and aliases | ✔ |
| fr.construction-certificate partial (approvals stage) | ✔ |
| fr.fitout-documentation three activities | ✔ |
| fr.tender-docs, period and evaluation fitout; fr.contract-award | ✔ |
| fr.site-establishment-fitout (5, 6 refurb) | ✔ |
| fr.strip-out-fitout partial (needs remove) | ✔ |
| Sanitary, cold, hot, electrical, lighting, mechanical, ventilation, partitions, ceilings, wet-area, tiling, internal-doors, comms, joinery, painting and floors fit-out fragments (accept new and alter on 5, 6, 9b refurb) | ✔ |
| fr.fire-stopping (class 6, penetrations new) | ✔ |
| Fire-stopping hold point label from if.fire-stopping-before-concealment | ✔ |
| fr.switchboard-major yes (upgrade, class 6) | ✘ partial: stage is construction (finding 15) |
| PC, OC, handover, defects fragments (refurb) | ✔ |
| No fragment for trade-waste, local-exhaust, ffe, signage, accessible-facilities, emergency-lighting, extinguishers, public-domain | ✔ (spot-checks of `no`) |
| Shopfront: windows fragments only 1a, 2, 7b, 9b, new or replace | ✔ (spot-check of a `no`) |

RFPs (against `knowledge/rfp/*.yaml`, `resolve.go`):

| claim | result |
|---|---|
| Interior Design → generic tpl.rfp-capex | ✔ |
| Certifier on bca-certification; positions need bca_* client flags | ✔ |
| Certifier app_differs: b-stages `option` after an a1–a3 boundary | ✘ all stages ask (finding 1) |
| Town planning: pre_lodgement include on da; panel_attendance O with flag.panel no | ✔ |
| mechanical-services `kitchen` and hydraulic-services `trade` positions exist and sit on client flags | ✔ |
| access `existing`, `solutions`, `s4_approval` keys exist | ✔ |

Triggers (built_in maps):

| claim | result |
|---|---|
| cdc: fr.cdc-application yes; town planner drops yes; RFPs block on cdc (no) | ✔ |
| riser: Mechanical suggested, structural and acoustic not (partial) | ✔ |
| arrestor constrained: Hydraulic suggested (yes) | ✔ |
| step at entry: Access suggested (yes); program no (stairs fragments 2 and 9b new only) | ✔ |
| gas: fr.gas-commercial on class 6 (partial) | ✔ |
| power: fr.utility-major excludes class 6 (program no) | ✔ |
| shopping centre: sprinklers and detection fitout on alter; sprinkler hold point (partial) | ✔ |
| heritage: complexity_additions[1], fr.heritage-referral | ✔ |
| bca upgrade: BCA suggested, fire engineer never (partial) | ✔ |
| liquor: liquor_license_type has no reader | ✔ |
| Profile and determinant claims: commercial_kitchen and trade_waste_use have no profile_group or triggers; implied_by kitchen omits trade-waste; det.sprinklered bare-word trigger | ✔ |

## Round 1 response

Author, 30 October 2026. All 17 findings are applied; none is disputed. The validator gives `ok`.

1. Applied. Every certifier stage (a1–a4, b1–b4) is now `app: ask`, with the reason "no boundary resolves while `bca_advisory` and `bca_certification` are unknown", and an a4 row is added. A `text.blocking` entry lists the seven `fact:flag.bca_*` checks (T03 convention). Fixed in DOMAIN §8, §11.6 and the rfp gap.
2. Applied. The riser trigger's `profile` adds `proj.regulated_design` and `hdr.cond.stakeholder_complexity`. `adds` gains the DBP Act authority line and the activity "Registered design practitioner declarations before the CC". The `why` names the regulated-design RFP rows. R3 is applied **by inference**: whether a ground-floor Class 6 fit-out alone is caught is `verified: false` and flagged for the owner (DOMAIN §9, §12).
3. Applied. The Tap in activity is added in procurement and made a predecessor of the CC. Added the cost line "Sydney Water Tap in approval fee" and an authority bullet in DOMAIN §5.
4. Applied. Added "Kitchen exhaust concept and discharge location for the DA" and made it a predecessor of the DA documents.
5. Applied. The CC and the long service levy move to `procurement`, and the trade wastewater application moves to `construction_documentation`.
6. Applied. Floors now come after waterproofing and ceilings, and joinery after floors. The `why` notes the app's order.
7. Applied. The kitchen designer is a `discipline: none` supply row (need `usually`), recorded as a gap. DOMAIN §4 now says "usually (often the supplier)".
8. Applied. `scope_additions[2]`–`[4]` are now cited on Mechanical, Hydraulic and Electrical, in the gap and in DOMAIN §4 and §11.1.
9. Applied. The `trade.main_works` why is updated and a gap entry is added for design-and-construct services under a traditional route.
10. Applied. The `interiors.ffe` why now states the provider-owner phrasing. Gaps are added for the double count and for `sys.*` not being a scenario input, and the §10 trap now maps landlord's works to `provider: landlord`.
11. Applied. Owner's consent stays and is re-added as "Landlord signs owner's consent for the CDC". `cc-fees` is no longer removed and carries the CDC fee. Trigger `adds` take only plain strings, so the alias is stated in `why` and no separate CDC fee line is added. The `why` also says the services design moves before tender.
12. Applied. Six triggers now have `profile: []` and say "No profile key" in their `why`.
13. Applied. Four triggers are added: `residential-neighbours-acoustic`, `roof-cannot-carry-exhaust-fan`, `patron-numbers-need-more-toilets` and `landlord-asbestos-register-covers-tenancy` (which removes the hazmat surveyor). The party-wall keyword trap is added to §10.
14. Applied. Both strings now carry `verified: false`.
15. Applied. The switchboard procure row is `built_in: "partial"`.
16. Applied. Both levy `why`s now read "only when the cost of works reaches the levy threshold; the scenario budget decides".
17. Applied:
    - the certifier is engaged at concept;
    - the shopping-centre trigger uses the app's hold-point label "Hold point: a recorded pressure test of the sprinkler pipework" (trigger `adds.program` takes strings only, so it replaces the label rather than aliasing it);
    - "Build the bin store" is added and made a predecessor of PC.

New counts: 17 triggers, 6 packages, 35 cost lines, 82 activities, 3 RFPs, 26 gaps.

---

## Round 2 (30 October 2026)

### Verdict: **PASS**

I checked all 17 round-1 findings against the revised `shape.yaml` and `DOMAIN.md`. Each is applied as
asked, and nothing was disputed. The validator passes again (`ok`, exit 0). The new content is sound. The
four notes below need no change before the key is used.

### The 17 findings

| # | Fix | Result |
|---|---|---|
| 1 | Certifier `app_differs`: all eight stages are `ask`; a4 added; `text.blocking` lists the `bca_*` checks; DOMAIN §8/§11.6 and the gap are reworded | ✔ |
| 2 | Riser trigger: `proj.regulated_design` and `stakeholder_complexity` in `profile`; DBP authority item marked as an R3 inference for the owner; practitioner declarations row | ✔ |
| 3 | Tap in: authority bullet, procurement-stage activity, CC `after`, fee line | ✔ |
| 4 | "Kitchen exhaust concept and discharge location for the DA" added and fed into the DA documents | ✔ |
| 5 | CC, levy and Tap in in `procurement`; trade wastewater permit in `construction_documentation` | ✔ (the permit still sits under the `# approvals` comment; cosmetic) |
| 6 | Floors → joinery → paint; no cycle; `why` records the app's order | ✔ |
| 7 | Kitchen designer as `discipline: none`, kind supply, plus a gap; DOMAIN §4 says "usually (often the supplier)" | ✔ |
| 8 | scope_additions[2]–[4] cited on Mechanical, Hydraulic and Electrical, and in the gap | ✔ |
| 9 | `trade.main_works.why` and a new gap: design lines are dropped under traditional | ✔ |
| 10 | FF&E provider phrasing; double-count gap; `sys.*` harness gap | ✔ |
| 11 | CDC: owner's consent kept and relabelled; cc-fees kept, with the reason stated; design-before-determination note | ✔ |
| 12 | `profile: []` on the six triggers whose key does not change | ✔ |
| 13 | Four triggers added (below); party-wall keyword trap in DOMAIN §10 | ✔ |
| 14 | `verified: false` on the FSANZ/AS 4674 RFP text and on s10.7 | ✔ |
| 15 | Switchboard procure row is `partial` | ✔ |
| 16 | Long service levy conditional on the threshold | ✔ |
| 17 | Certifier engaged at concept; app hold-point label used for the sprinkler test; bin store row | ✔ |

### The new triggers

- **`residential-neighbours-acoustic`.** Realistic evidence and negations. `consultant.acoustic` is never
  suggested, so `procurement: "no"` is right. The POEO Act reference is `verified: false`. ✔
- **`roof-cannot-carry-exhaust-fan`.** `fr.structure-alterations` holds for an `alter` item on refurb with
  no class gate. Its label "Alter and protect retained structure" matches the fragment, so
  `program: "partial"` is right. ✔
- **`patron-numbers-need-more-toilets`.** The domain content is right and Part F4 is `verified: false`. ✔
  See note 2.
- **`landlord-asbestos-register-covers-tenancy`.** It correctly says the app keeps the survey row from
  `fr.existing-conditions`, so `program: "no"`. ✔

### Notes (no change needed for PASS)

1. **Blocking checks will score as new failures.** `score_shapes.py _compare_expect` compares
   `expect.blocking` directly and does not read `text.*` keys, so `text.blocking` is recorded as not
   checked. The seven `fact:flag.bca_*` checks will therefore show as `new_failure`. So will the
   client-owned allowance counts with no default. `resolveAllowances` blocks on these whenever a stage is
   bought: for example `allowance:plan_sets` on the certifier and `allowance:authority_meetings` on the
   town planner. This is a harness convention shared with T03, not a map defect. Either the scorer learns a
   known-blocking list, or the harness sets the allowance settings the way the RFP answer keys do. If the
   author wants it, append "and allowance:* counts" to the certifier `text.blocking` app string.
2. **`patron-numbers-need-more-toilets` changes nothing the app produces.** Its work items
   (`hydraulic.sanitary`, `interiors.wet-area-waterproofing`) are already in the base. Every other output
   is `no`. `rule.ncc.sanitary-facility-quantities` is also `pending: true`. Like `liquor-licence`, it is
   outside `variants`, so nothing breaks. It belongs in the harness zero-delta gap.
3. **Duplicate gap.** The kitchen-designer gap appears twice (gaps procurement 2 and the new
   `discipline none` entry). Merge them when convenient.
4. **The R3 inference still needs the owner.** It is correctly flagged for the owner and is the only open
   domain question in this map.
