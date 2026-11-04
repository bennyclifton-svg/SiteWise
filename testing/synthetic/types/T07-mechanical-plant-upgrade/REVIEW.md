# T07 Mechanical plant upgrade (chillers, AHUs): independent review

Reviewer: independent agent, 30 October 2026. I changed nothing except this file. No Jev, no network, no
git writes. To check the RFP claims I ran the real `rfp.Resolve` for the mechanical, structural, electrical
and BCA templates on the base drivers. It ran as a throw-away test file supplied through `go test -overlay`
from my scratchpad, so no file was written in the repo. The drivers were: refurb, exempt, traditional, NSW,
regulated design no, novation no, one Class 5 part, and the shape's flags, with `live_operations` yes and
`heritage` no.

## Verdict: **FIX**

The map is right at the root:

- The base scenario, the team, the systems walk and the program spine all hold up against the knowledge and
  the Go.
- The handling of the exempt pathway in the base is correct:
  - P05 CC fees, P06 service-authority fees and the Certifier fee line are in `not_expected_cost_plan`.
  - No DA fees line appears (`da-fees` needs planning `da`).
  - `fr.exempt-confirmation` is present and `sk.refurb-traditional-cdc` is chosen.
  - The structural `planning_approval` stage and the BCA B1–B4 stages are excluded, and `exempt` is not in
    the BCA template's `unsupported_pathways` (confirmed by running the Go).
  - The Occupation certificate row is recorded as a gap.

Before the map is used as an answer key, fix these:

- three wrong `app_differs` entries in the RFPs;
- three variants whose expectations would score correct app output as false positives;
- three trigger evidence lines that either also hold in the base or cannot hold for this building;
- a missing hold point and two missing links in the program;
- the attribution of the 25 m threshold.

**Validator:** `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T07-mechanical-plant-upgrade/shape.yaml`,
run from the repo root, prints `ok …shape.yaml` and exits 0. The counts match DOMAIN.md: 15 work items,
6 packages, 28 cost lines, 44 activities, 4 RFPs with a template plus hazmat, 15 triggers and 27 gaps.

## Findings

1. **`rfps[consultant.mechanical].app_differs[position.existing]` and
   `rfps[consultant.electrical].app_differs[position.existing]` are wrong.**
   - **Evidence.** Both `existing` rows sit under `when: stage [investigation]`. The investigation stage is
     `ask`, because the `mech_investigation` and `elec_investigation` client flags are unknown. The `stage`
     test passes only for Include or Option (`internal/rfp/resolve.go` l.497–503), so the row is `-`. The
     resolver prints it as an inactive X ("not required by the current brief"), not `ask`. The Go run
     confirms both: `pos existing X inactive=true`.
   - **Change.** In both entries set `app: absent`, with the why "inactive until the investigation stage
     (client flag) is answered".
   - **DOMAIN §8.** In the mechanical "App:" bullet, take `existing` out of the list of rows that ask, and
     add: "the existing-condition row is hidden until someone answers the investigation stage". Make the
     same change in the electrical bullet.

2. **`rfps[consultant.structural].app_differs[position.inspection_schedule]` is wrong.** The app gives `I`,
   not `O`.
   - **Evidence.** The row's value is `I when stage [construction_phase]`. The `stage` test is true for an
     option stage, so the value is `I` even though the stage itself is only an option. The Go run confirms
     it: `pos inspection_schedule I`.
   - **Change.** Delete the entry. `expect.positions.inspection_schedule: I` then agrees with the app.

3. **`triggers[complying-development]`: the CDC route blocks the BCA consultant's RFP too, and the Certifier
   fee line stays a false positive.**
   - **Evidence, RFPs.** `bca-certification` lists `cdc` in `unsupported_pathways`, and `consultant.bca` and
     `consultant.certifier` share that template (KNOWLEDGE_INDEX §5.2, CONVENTIONS C3). On a CDC, the base
     BCA RFP and the added certifier RFP both report `pathway_unsupported`.
   - **Evidence, cost plan.** `expectations()` in `tools/score_shapes.py` (l.159–165) lifts a
     `not_expected_cost_plan` line only when `adds.cost_plan` names the same normalised label. Adding
     `consultant.certifier` to procurement does not lift the "Certifier (consultant fee)" line, so a correct
     certifier fee line would score as a false positive.
   - **Change:**
     - set `built_in.rfps: "no"`;
     - add `"Certifier (consultant fee)"` to `adds.cost_plan`;
     - in the `why`, write: "Both consultant.bca and consultant.certifier RFPs block (bca-certification
       unsupported_pathways cdc); the base BCA RFP's expect.blocking becomes [pathway_unsupported] in this
       variant";
     - add a `gaps` entry (area `rfp`, evidence `knowledge/rfp/bca-certification.yaml unsupported_pathways`)
       for the blocked BCA RFP. The existing gap names only the certifier and town planner.

   The variant cannot change `rfps[].expect` (the scorer has no hook for it), so say in the `why` that the
   BCA `expect.blocking` check will fail in this variant until the orchestrator adds that hook.

4. **`triggers[heritage-item]` has the same Certifier fee false positive, and it changes the skeleton.**
   - **Change.** Add `"Certifier (consultant fee)"` to `adds.cost_plan`.
   - **Skeleton.** With `hdr.cond.planning: da` the app picks `sk.refurb-traditional-da`, but the scorer
     takes the skeleton from the base `program.skeleton` only (`expectations()`), so the variant will report
     a skeleton mismatch. Say this in the `why`, and raise it with the orchestrator (see finding 10).

5. **`triggers[heating-electrification]`: P06 stays not expected although the map says it becomes
   appropriate.**
   - **Evidence.** The trigger's `why` says "P06 Service authority connection fees becomes appropriate", but
     `adds.cost_plan` names "Electricity supply augmentation and DNSP connection" instead. The
     `not_expected_cost_plan` entry for P06 is therefore never lifted.
   - **Change.** Add `"Service authority connection fees"` to `adds.cost_plan`, and keep the augmentation
     line as a separate construction line.

6. **`triggers[nabers-target].evidence[1]` is not a target, by the map's own rule.** "The building's current
   Commercial Building Disclosure certificate shows a 3.5-star NABERS Energy rating" describes a rating that
   exists. DOMAIN §10 and `gaps` (profile, `sustainability_target`) both say "a rating that exists is not a
   target".
   - **Change.** Replace it with "The upgrade is to lift the base building from its current 3.5-star NABERS
     Energy rating to at least 5 stars."
   - Move the CBD-certificate sentence to §10 as a trap: it sets `flag.sustainability_target` in the base,
     and the base expects `no`.

7. **`triggers[temporary-cooling].evidence[1]` is also true in the base.**
   - **Evidence.** DOMAIN §3 says that in the base "cooling must be kept at the levels the leases require
     during business hours". A lease obligation does not force temporary cooling when N-1 carries the load
     in the cooler months.
   - **Change.** Replace it with "The replacement must proceed over summer, when the remaining chiller
     cannot carry the building load."

8. **`triggers[fire-engineered-smoke-control].evidence[2]` cannot hold for the base building.** A 6-storey
   office cannot have an effective height of 32 m. Effective height is measured to the floor of the topmost
   storey, so here it is about 18–20 m.
   - **Change.** Either replace it with "Zone smoke control was approved by a performance solution in the
     original fire engineering report", or keep it and add to the trigger's `why`: "This variant needs a
     taller building: set hdr.scale.storeys and det.rise_in_storeys to 10 or more."

9. **The 25 m threshold is misattributed, and the base fire mode is justified by the wrong thing.** See
   ruling 3 below.
   - **Change, `shape.yaml` header comment (l.15–16) and DOMAIN §1 base bullet.** Replace "because the
     effective height is under 25 m" with "as listed on the building's fire safety schedule (effective height
     under 25 m, so no zone smoke control or stair pressurisation)".
   - **Change, DOMAIN §5 and §12.** Write: "The 25 m effective-height trigger for zone smoke control and stair
     pressurisation is an NCC smoke hazard management provision (Part E2 and its tables), not an AS 1668.1
     number (verified: false). AS 1668.1 sets how the fire mode works."

10. **Exempt pathway: the Occupation certificate row cannot be scored as wrong.**
    - **Evidence.** `fr.occupation-certificate` is gated on work type only (`applies_when` new, extend,
      refurb), and `fr.handover` attaches after it (`attach: {after: fr.occupation-certificate}`). Both are
      confirmed. The schema has no `not_expected_program` (`validate_shape.py` `OPTIONAL`), and
      `score_shapes.py` builds `not_activities` only from trigger `removes`. So on an exempt job the row
      counts as `extra`, not as a false positive.
    - **Change (orchestrator, not this map).** Add an optional `not_expected_program` list to the validator
      and the scorer. Then key `{label: "Occupation certificate", source: "knowledge/program/fragments.yaml
      fr.occupation-certificate", why: "No OC on the exempt path"}`. Have `complying-development` and
      `heritage-item` lift it, since both already add "Occupation certificate". Until then the `gaps` entry
      is the only record. Leave it as it is.
    - T06, T08 and any other exempt refurb have the same problem. The same hook would let a trigger change
      the expected skeleton (finding 4).

11. **`program.activities`: a missing hold point and two missing links.**
    - **The structural inspection.** DOMAIN §5 hold point 2, structural inspection of the frames before
      plant is set down, has no activity. Add `{label: "Hold point: structural engineer inspects support
      frames before plant is set down", stage: construction, fragment: null, after: ["Fabricate and install
      chiller support frames"], built_in: "no", why: "Contract hold point; no interface hold point resolves
      for structure.steel (KNOWLEDGE_INDEX §4.4)."}`. Add it to the `after` list of "Stage 1: crane out
      chiller 1 and lift in new chiller 1".
    - **Power before start-up.** "Stage 1: connect, start up and commission chiller 1" and "Commission AHUs
      and balance air" do not wait for their new supply. Add "Install submains and mechanical services
      switchboards" to the `after` list of both.
    - **Switchboard lead time.** The mechanical services switchboards are listed as long-lead in DOMAIN §7,
      but no activity carries their lead time. Rename "Manufacture and deliver chillers and AHUs" to
      "Manufacture and deliver chillers, AHUs and mechanical services switchboards" (update every `after`
      that names it). Add it to the `after` list of "Install submains and mechanical services switchboards".

12. **The base negations have nowhere to live.** A base scenario has to carry three negations, or a reader
    cannot tell the base from a variant:
    - "air-cooled; no cooling towers";
    - "staged in the cooler months; no temporary cooling";
    - "surrounded by commercial buildings".

    `scenario_inputs` holds only heritage.
    - **Change:** extend `scenario_inputs[flag.staged].phrasing` to "The works will be carried out in two
      stages in the cooler months: Stage 1 chiller 1, Stage 2 chiller 2. The remaining chiller carries the
      load, so no temporary cooling is required."
    - The other two have no app key (the validator accepts only hdr/det/fact/proj/site/flag keys). Add a
      short list "Base documents must also state" to DOMAIN §1, with the two negation sentences from the
      `cooling-towers` and `noise-sensitive-receivers` triggers.

13. **DOMAIN §10 "Kitchen" trap is wrong as written.** The pattern is `\bkitchens?\b`
    (`knowledge/profile/scope_defaults.yaml implied_by kitchen`). It does not match "kitchenettes". "Toilet
    exhaust fans not affected" contains no word that the `wet-room` pattern matches.
    - **Change.** Use "tenant kitchens on each floor are not affected" and "bathroom exhaust fans are not
      affected" as the trap examples.

14. **`triggers[asbestos-in-plant-room].negation[1]` is unrealistic.** "The plant rooms were cleared of
    asbestos in 2009" cannot be true while the 1988 flanged plant, the source of the gaskets, is still in
    service.
    - **Change.** Replace it with "The refurbishment survey sampled the chilled-water flange gaskets, pipe
      lagging and AHU flexible connections; no asbestos was detected."

15. **Missing common triggers.** Each is a one-change variant a real job throws up:
    - **`capacity-increase`.**
      - Evidence: "the replacement chillers are 20 % larger to suit tenant loads", "an N+1 standby chiller is
        added".
      - Negation: "like-for-like capacity".
      - Adds: action `upgrade` on `mechanical.central-plant`, a maximum-demand check that may lead to supply
        augmentation and P06, and a heavier deck load check.
      - The map already notes the replace/upgrade wording in §10, but nothing tests it.
    - **`crane-oversails-neighbour`.**
      - Evidence: "the crane jib will oversail 14 Smith Street during the lift".
      - Negation: "the lift radius stays within the site and the road reserve".
      - Adds: a crane oversail licence with the adjoining owner (verified: false), a dilapidation survey of
        the adjoining building, and a program activity before the lift.
      - DOMAIN §3 names this, but nothing tests it.
    - **`flammable-refrigerant`.**
      - Evidence: "the replacement chillers use R1234ze (A2L)".
      - Adds: refrigerant safety-group checks and machinery-room requirements under AS/NZS 5149 (verified:
        false).
      - This is a current issue as HFCs are phased down. It is optional.

16. **Small corrections.**
    - **Access why.** `not_expected_procurement[consultant.access].why` names "AS 1657" without
      `(verified: false)`. Add it.
    - **Crane day.** `program.activities["Stage 1: crane out chiller 1 …"].why` says "One crane day", a
      duration the repo does not hold. Change it to "A single planned crane lift, usually at a weekend."
    - **DOMAIN §4.** It says Structural and Hazmat "are keyed in `shape.yaml not_expected_procurement`".
      They are in `procurement` with `built_in: "no"`. The six unneeded consultants are the ones in
      `not_expected_procurement`. Fix the sentence.
    - **DOMAIN §5, Section J on the exempt path.** If the exempt route relies on the Codes SEPP, its general
      requirements (as I recall them) include meeting the relevant BCA deemed-to-satisfy provisions.
      Section J compliance is then a condition of the exemption, not only "a design requirement". Rewrite
      the sentence that way, verified: false.

## Rulings on the author's open questions

| question | ruling |
|---|---|
| Codes SEPP exempt provision for like-for-like plant | **Keep exempt as the base; the provision stays verified: false.** In practice NSW owners do treat like-for-like replacement in the same position, with no rise in height, footprint or noise, as needing no consent. But I cannot name the provision with confidence. The "air-conditioning units" subdivision of Part 2 reads as written for small units. My recollection of the general requirements for exempt development (cl 1.16: meet the BCA deemed-to-satisfy provisions, not on a heritage item for most subdivisions) is also unconfirmed. Do not let the scenario author name a clause. Phrase it as the BCA consultant's conclusion, as `scenario_inputs` already does. Where the plant grows, the usual next route is a CDC under the Commercial and Industrial Alterations Code (Part 5, verified: false). Put that name in `complying-development.adds.authority` in place of "code relied on". |
| NCC 2022 Part for HVAC energy efficiency | **J6 matches my reading of NCC 2022.** It renumbered Section J so that J6 is air-conditioning and ventilation (it was J5 in NCC 2019), J7 is lighting and power, J8 is heated water, and J9 is energy monitoring and on-site distributed energy. No NCC 2022 text is in the repo, and the app's rule `rule.ncc.hvac-energy-efficiency` carries three Part numbers with `clause_verified: false`. **Keep verified: false** wherever J6 appears; every place in the map already marks it. The sub-metering in `nabers-target` would sit under J9 (verified: false). |
| AS 1668.1 25 m threshold | **Wrong source; keep verified: false.** The 25 m effective-height trigger for zone smoke control and stair pressurisation is an NCC smoke hazard management provision (Part E2 and its tables; NCC 2019 E2.2 numbering in the app's seed). It is not an AS 1668.1 number. AS 1668.1 sets how the fire mode, the pressurisation and the testing work. The base fire mode should rest on the building's fire safety schedule, not on today's threshold, because an existing building keeps its approved measures (finding 9). The 50 Pa and 110 N figures in the app's rule also stay unverified. |
| BCA consultant in the base | **Keep, at the limited scope written.** An institutional owner of an occupied Class 5 office normally wants written advice before relying on exempt status and before isolating listed essential fire safety measures. The app suggests BCA anyway (`baselines[3]`), so this also keeps the key honest. A variant in which the owner relies on the mechanical engineer alone is not worth a trigger. |
| Electrical engineer in the base | **Keep.** New VSD pumps and chillers, new mechanical services switchboards, new submains and the MSB protection changes need a maximum demand check and protection and discrimination design. Often the same firm does it, which the `consultant.services` aliases already cover. Note one limit: the scorer pairs one-to-one, so a single `consultant.services` package can satisfy only one of the two rows. A multi-services-firm scenario will show one miss. That is acceptable. |
| Acoustic as a trigger, not base | **Keep as a trigger.** With commercial neighbours and like-for-like plant, the mechanical engineer's noise data for the selections is enough. That is already a `must_include`, and the exempt test of "no increase in noise" relies on it. The base documents must state the negation (finding 12). |
| AHU fire-mode re-test by the building's fire contractor, no fire services consultant | **Agree.** The cause and effect does not change. The mechanical engineer specifies the interface, and the incumbent fire contractor isolates, reinstates and tests under the AFSS regime (verified: false). |

Prices, rates and durations: none quoted except "one crane day" (finding 16). Durations otherwise defer to
`knowledge/program/durations.yaml`. Owner rulings: R1 (no change of roof or hard surfaces, so no civil), R3
(DBP Act not for Class 5), R4 and R5 (not residential) and R7 (Class 1 and 10 only) are respected. R8 does
not apply (the contract is AS 4000, length kept verified: false). ✔

## Built-in claims checked

| # | claim | result |
|---|---|---|
| 1 | procurement `consultant.mechanical` yes, `baselines[3]` (also `scope_additions[2]`) | ✔ |
| 2 | procurement `consultant.electrical` yes, `baselines[2]` (also `scope_additions[4]`) | ✔ |
| 3 | procurement `consultant.bca` yes, `baselines[3]` | ✔ |
| 4 | procurement `trade.mechanical` partial, `ProposeHeadContract` (main_works only, accepted items only) | ✔ |
| 5 | not expected, app suggests: Architect `[2]`, Interior Design `[4]`, Access `[3]`, Hydraulic `[2]`, Fire Services `[3]` (+`scope_additions[1]` via detection), Certifier `[2]` | ✔ |
| 6 | cost: Mechanical, Electrical, BCA fee lines yes (prepare adopts suggestions) | ✔ |
| 7 | cost: Structural and Hazmat fee lines partial (`Suggest` per package; never suggested) | ✔ |
| 8 | cost: nine 08/03/04/06/07/10 lines partial (one generic line per element; one-to-one pairing leaves the rest as known gaps) | ✔ |
| 9 | cost: "Plant access platforms…" partial, alias "Plant and maintenance access" (unlisted itemised child takes the system label, `suggest.go` l.276–292) | ✔ |
| 10 | cost: design contingency, construction contingency, escalation, exclusions yes | ✔ |
| 11 | not expected cost: P05 `cc-fees` and P06 `service-authority` `applies_when: true` | ✔ |
| 12 | budget split `commercial_fitout` (no structure or envelope share; mechanical 12, electrical 11) | ✔ |
| 13 | skeleton `sk.refurb-traditional-cdc` (refurb, traditional, cdc or exempt) | ✔ |
| 14 | `fr.existing-conditions` ×3, refurb gate | ✔ |
| 15 | structural assessment partial, `investigationRows` (no `fr.investigate-*` for structure) | ✔ |
| 16 | `fr.concept-design-fitout` ×3, `fr.design-development-fitout`, `fr.fitout-documentation` ×3 (class 5 refurb) | ✔ |
| 17 | `fr.exempt-confirmation` (planning exempt) | ✔ |
| 18 | `fr.tender-docs-fitout`, `fr.tender-period-fitout`, `fr.tender-evaluation-fitout`, `fr.contract-award`, `fr.site-establishment-fitout` (5, 6) | ✔ |
| 19 | `fr.mechanical-fitout` partial on `air-conditioning` replace | ✔ |
| 20 | `fr.electrical-fitout` partial on `cabling-wiring` alter; `fr.switchboard-major` avoided by `alter` | ✔ |
| 21 | `fr.detection-fitout` partial on alter; `fr.fire-stopping` yes (class 5, alter) | ✔ |
| 22 | `fr.practical-completion`, `fr.handover` (attached after OC), `fr.defects-inspection`, `fr.defects-liability-end` ×2 | ✔ |
| 23 | every fragment-backed activity's `stage` equals its fragment's stage (26 checked by script) | ✔ |
| 24 | RFP mechanical partial: stages and 19 of 20 `app_differs` | ✔ |
| 25 | RFP mechanical `app_differs[position.existing]` app ask | ✘ (finding 1) |
| 26 | RFP electrical partial: stages and 19 of 20 `app_differs` | ✔ |
| 27 | RFP electrical `app_differs[position.existing]` app ask | ✘ (finding 1) |
| 28 | RFP structural partial: complex scale (class 5), `existing_assessment` in, `construction_phase` option, 10 of 11 `app_differs` | ✔ |
| 29 | RFP structural `app_differs[position.inspection_schedule]` app O | ✘ (finding 2) |
| 30 | RFP BCA partial: A1/A3 ask, A2 and B1–B4 excluded on exempt, no `pathway_unsupported`, 8 `app_differs` | ✔ |
| 31 | RFP hazmat: no template, generic `tpl.rfp-capex` | ✔ |
| 32 | trigger `cooling-towers` cost partial (08), rfps partial | ✔ |
| 33 | trigger `temporary-cooling` program partial (`fr.temporary-power` "Connect temporary builders power") | ✔ |
| 34 | trigger `asbestos-in-plant-room` program partial (`fr.strip-out-fitout`), cost partial (01) | ✔ |
| 35 | triggers `noise-sensitive-receivers`, `independent-commissioning` cost partial (`Suggest` per package) | ✔ |
| 36 | triggers `basement-plant-room`, `bms-replacement`, `nabers-target` (code 10), `facade-opening` (code 04) cost partial | ✔ |
| 37 | trigger `complying-development` procurement, cost and program yes (`fr.cdc-application`; OC already fires) | ✔ |
| 38 | trigger `complying-development` rfps partial | ✘ (finding 3) |
| 39 | trigger `heritage-item` procurement yes (`complexity_additions[1]`, `[0]`, `baselines[2]`), program yes (`fr.heritage-referral`, `fr.da-*` labels match, `fr.construction-certificate`), cost yes (`da-fees`) | ✔ |
| 40 | trigger `fire-engineered-smoke-control` rfps partial (fire-engineering template) | ✔ |
| 41 | trigger `heating-electrification` program no (`fr.utility-major` excludes 5; `fr.gas-commercial` has no remove) | ✔ |
| 42 | trigger `plant-deck-strengthening` program partial (`fr.structure-alterations` on upgrade, refurb) | ✔ |
| 43 | all 31 cited `rule.`, `uc.`, `fm.`, `if.` and `cq.` ids exist in `knowledge/` | ✔ |

Spot checks of `no` claims (knowledge really has nothing):

| # | claim | result |
|---|---|---|
| N1 | Structural never suggested for refurb (`baselines[2]`, `[3]`) | ✔ |
| N2 | Hazmat: no package rule and no RFP template | ✔ |
| N3 | Long service levy, crane and road permits: no starter line | ✔ |
| N4 | Acoustic and fire engineer: no package rule | ✔ |
| N5 | `central-plant`, `controls-bms`, `refrigerant-condensate`, `fire-mode-air-systems`, `cooling-towers`, `structure.temporary-works`, `access-egress.maintenance-access`: no work-item fragment | ✔ |
| N6 | `structure.steel` has no Class 5 fragment (`fr.steel-7b`, `fr.steel-9b` only, new or extend) | ✔ |
| N7 | `classified-road-crane-lift` cost and program no | ✔ |
| N8 | no interface hold point resolves for any base system (`if.fire-stopping-before-concealment` needs interiors) | ✔ |

## Round 1 response (author, 4 November 2026)

Most edits were already in the committed `shape.yaml` and `DOMAIN.md` (applied before the tree was
committed, without a response here). I checked each one against the finding and finished the rest. I ran
the real `rfp.Resolve` again through a scratchpad test passed with `go test -overlay` (nothing written in
the repo). Drivers: class 5, refurb, traditional, NSW, regulated design no, novation no, the shape's flags,
`heritage` no, `live_operations` yes, pathway exempt and then cdc. Results: mechanical and electrical
`existing` = `X inactive=true` on both pathways; structural `inspection_schedule` = `I`; bca-certification
blocks with `pathway_unsupported` on cdc only.

| # | finding | response |
|---|---|---|
| 1 | mechanical and electrical `app_differs[position.existing]` | **Applied.** Both are `app: absent` with the inactive-until-answered why. DOMAIN §8 mechanical and electrical "App:" bullets say the row is hidden until the investigation stage is answered. Go confirmed. |
| 2 | structural `app_differs[position.inspection_schedule]` | **Applied.** Entry deleted; `expect.positions.inspection_schedule: I` agrees with the app. Go confirmed. |
| 3 | `complying-development` RFPs and Certifier fee | **Applied.** `built_in.rfps: "no"`; "Certifier (consultant fee)" in `adds.cost_plan`; the `why` names both blocked RFPs and the missing scorer hook; new `gaps` entry (area `rfp`) for the blocked BCA RFP; `adds.authority` names the Commercial and Industrial Alterations Code (Part 5, verified: false) per ruling 1. Go confirmed the cdc block. |
| 4 | `heritage-item` Certifier fee and skeleton | **Applied.** Certifier fee line added; the `why` says the scorer will report a skeleton mismatch (`sk.refurb-traditional-da` is correct). The scorer hook is for the orchestrator. |
| 5 | `heating-electrification` P06 | **Applied.** "Service authority connection fees" added; the augmentation line kept separate. |
| 6 | `nabers-target.evidence[1]` | **Applied.** Replaced with the 3.5-to-5-star lift. The CBD-certificate sentence is a §10 trap. Ruling 2: the `why` now notes sub-metering would sit under Part J9 (verified: false). |
| 7 | `temporary-cooling.evidence[1]` | **Applied.** "The replacement must proceed over summer, when the remaining chiller cannot carry the building load." |
| 8 | `fire-engineered-smoke-control.evidence[2]` | **Applied** (first option). Replaced with the performance-solution sentence; no taller building needed. |
| 9 | 25 m threshold | **Applied.** Header comment and DOMAIN §1 rest the fire mode on the fire safety schedule; DOMAIN §5 and §12 attribute the 25 m trigger to NCC Part E2 (verified: false), with AS 1668.1 setting how the fire mode works. |
| 10 | Occupation certificate not scorable | **Accepted, no change here** (orchestrator: `not_expected_program` and a skeleton hook). The `gaps` entry stays the record. |
| 11 | hold point and links | **Applied.** Structural hold point activity added before the Stage 1 lift; submains and switchboards added to the `after` of chiller 1 start-up and AHU commissioning; delivery renamed "Manufacture and deliver chillers, AHUs and mechanical services switchboards" in every `after`, and it precedes the submains row. DOMAIN §7 logic updated. |
| 12 | base negations | **Applied.** `scenario_inputs[flag.staged].phrasing` carries the no-temporary-cooling sentence; DOMAIN §1 has "Base documents must also state" with the cooling-tower and commercial-neighbours sentences. |
| 13 | Kitchen trap | **Applied.** DOMAIN §10 uses "tenant kitchens on each floor" and "bathroom exhaust fans", and notes that "kitchenettes" and "toilet" do not match. |
| 14 | asbestos negation | **Applied.** Replaced with the sampled-and-not-detected sentence. |
| 15 | missing triggers | **Applied.** `capacity-increase`, `crane-oversails-neighbour` and `flammable-refrigerant` added to `shape.yaml` and DOMAIN §9. Not added to `variants`. |
| 16 | small corrections | **Applied.** Access why carries AS 1657 (verified: false); Stage 1 lift why is "A single planned crane lift, usually at a weekend", and Stage 2 now says "a second planned crane lift" rather than "crane day"; DOMAIN §4 sentence fixed; DOMAIN §5 makes Section J a condition of the exemption (verified: false). |

Disputed: none.

New counts: 15 work items, 6 packages (6 not expected), 28 cost lines (8 not expected), 45 activities,
5 RFPs (4 with a template plus hazmat), 18 triggers, 28 gaps, 8 variants.

Validator: `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T07-mechanical-plant-upgrade/shape.yaml`
prints `ok` and exits 0.

## Round 2 review (4 November 2026)

Re-read `shape.yaml`, `DOMAIN.md` and the Round 1 response, against `OWNER_RULINGS.md` and
`CONVENTIONS.md`. Validator, run from the repo root: `ok`, exit 0. A script check found 45 activities,
28 cost lines, 18 triggers and 28 gaps, matching the response. It found no dangling `after` reference,
no duplicate activity label, and no trigger `removes` entry that misses a base label.

### Verdict: **FIX**

Every round 1 finding is applied correctly. Four new items: one comes from the round 1 hold-point fix,
two come from conventions recorded since round 1 (C4), and one comes from a new trigger.

### Fixes checked

- **F1 ✔.** Both `position.existing` entries are `app: absent`, with the inactive-until-answered why.
  DOMAIN §8 matches.
- **F2 ✔.** The structural `inspection_schedule` entry is gone, and `expect` is `I`.
- **F3 ✔.** `rfps: "no"`, the Certifier fee is in `adds.cost_plan`, and the `why` names both blocked RFPs
  and the missing scorer hook. The new `gaps` entry is in place, and so is the Part 5 code name
  (verified: false).
- **F4 ✔.** The Certifier fee is added, and the skeleton mismatch is stated in the `why`.
- **F5 ✔.** "Service authority connection fees" is added, and the augmentation line is kept separate.
- **F6 ✔.** The evidence now describes the 3.5-to-5-star lift. The CBD sentence is a §10 trap, and J9
  is noted (verified: false).
- **F7 ✔, F8 ✔.** Both evidence lines are replaced. Neither holds in the base.
- **F9 ✔.** The header comment, §1, §5, §11 and §12 rest the fire mode on the fire safety schedule.
  They attribute the 25 m trigger to NCC Part E2 (verified: false).
- **F10 ✔ (accepted, orchestrator).** The `gaps` entry stays as the record.
- **F11 ✔ as written**, but see new finding 1.
- **F12 ✔.** The `flag.staged` phrasing and the §1 "Base documents must also state" list are in place.
- **F13 ✔, F14 ✔, F16 ✔.** No "crane day" is left. DOMAIN §4 is fixed. §5 makes Section J a condition
  of the exemption (verified: false).
- **F15 ✔.** All three triggers are added to `shape.yaml` and §9 (see new finding 4).

### Findings

1. **The frame hold point sits before the old chiller is out, which cannot happen.**
   - **Where.** `program.activities`, "Fabricate and install chiller support frames", the hold point and
     "Stage 1: crane out chiller 1 and lift in new chiller 1".
   - **What is wrong.** `work_items[structure.steel]` and DOMAIN §2 put the new frames "on the existing
     plinths". The old chiller 1 sits on those plinths until the Stage 1 lift removes it. As keyed, the
     frames are installed and inspected before the lift that takes the old chiller off, so the plinths
     are still occupied when the frames go on. The same holds for chiller 2 inside the Stage 2 row. My
     round 1 F11 added the hold point to this chain and made the conflict a hard link.
   - **Change.**
     - Rename "Fabricate and install chiller support frames" to "Fabricate chiller support frames".
     - Split the Stage 1 lift into "Stage 1: crane out chiller 1", after the isolate row, the council
       approval and the lift plan, and "Stage 1: lift in new chiller 1".
     - Add "Stage 1: install support frame on the chiller 1 plinths", after "Stage 1: crane out
       chiller 1" and "Fabricate chiller support frames".
     - Point the hold point's `after` at the install row, and put the hold point and the delivery row
       in the `after` of "Stage 1: lift in new chiller 1". Point the flushing hold point at that row.
     - Add to the Stage 2 `why`: "including frame install and its hold point between the two lifts".
     - In DOMAIN §7, rewrite the Stage 1 chain as: isolate and recover → crane out → install frame →
       hold point → lift in → flushing → start-up. Note: the crane often stays for the weekend, so
       both lifts may be one booking.
     - The activity count becomes 47.

2. **The WHS Regulation is cited as 2017, against CONVENTIONS C4.** C4 records that the NSW WHS
   Regulation 2025 has replaced the 2017 Regulation.
   - **Where.** `procurement[consultant.hazmat].why`, `rfps[consultant.hazmat].must_include[0]`, DOMAIN
     §4 (l.161), §5 pressure equipment (l.272) and §12 (l.670).
   - **Change.** Write "WHS Regulation 2025 (NSW), verified: false" in each place.

3. **The long service levy figures are now a checked fact (CONVENTIONS C4), but the map keeps them
   unverified.**
   - **Where.** `cost_plan["Long service levy"].why`, DOMAIN §5 (l.283–285) and §12 (l.679).
   - **Change.** Write "0.25% of the cost of building work of $250,000 or more, including GST, paid
     before work starts (CONVENTIONS C4)". Keep only "whether like-for-like plant replacement is
     building work for the levy" as verified: false.

4. **`triggers[capacity-increase]` expects a conditional cost and duplicates a base activity.**
   - **Cost lines.** `adds.cost_plan` expects "Electricity supply augmentation and DNSP connection" and
     "Service authority connection fees" outright. The trigger's own `why` says they apply "only if the
     demand check shows the supply must grow". None of the evidence says it does, so an app that
     correctly omits them scores two misses.
     - **Change.** Either add evidence that settles it ("The maximum demand assessment shows the
       building supply must be augmented for the added chiller"), or drop both lines from
       `adds.cost_plan` and leave the condition in the `why`.
     - If you drop them, `heating-electrification` stays the only trigger that lifts P06.
   - **Activity.** `adds.program` "Structural check of the deck for the heavier or added plant" repeats
     the base row "Structural assessment of the roof plant deck and plant room slabs". It will pair
     with nothing or double-count.
     - **Change.** Drop it, and keep the structural RFP `must_include`, which carries the change.

### Optional, not blocking

- `flammable-refrigerant` and `basement-plant-room` add `mechanical.refrigerant-condensate` `upgrade`
  without removing the base `replace`. `score_shapes.py` does not score `work_items` today, so nothing
  fails. For consistency with `bms-replacement` and `capacity-increase`, add `removes.work_items:
  [{system: mechanical.refrigerant-condensate, action: replace}]` to both.

## Round 2 response (author, 4 November 2026)

| # | finding | response |
|---|---|---|
| 1 | frame hold point before the old chiller is out | **Applied.** "Fabricate chiller support frames" (renamed). The Stage 1 lift is split into "Stage 1: crane out chiller 1" (after isolate, council approval, lift plan) and "Stage 1: lift in new chiller 1" (after the hold point and the delivery row). The new row "Stage 1: install support frame on the chiller 1 plinths" sits between them, and the hold point now follows it. The flushing hold point follows the lift in. The Stage 2 why names the frame and hold point between its two lifts. `basement-plant-room.removes.program` names both new lift rows. DOMAIN §7 chain and logic are rewritten, with the one-weekend booking note. |
| 2 | WHS Regulation 2017 | **Applied.** "WHS Regulation 2025 (NSW), verified: false" is in the hazmat procurement why, the hazmat RFP must_include, DOMAIN §3 and §5, and §12 (which also notes that it replaced 2017, C4). |
| 3 | long service levy | **Applied.** The cost line why, DOMAIN §5 and §12 give 0.25% of building work of $250,000 or more, including GST, paid before work starts (CONVENTIONS C4). Only whether like-for-like plant replacement is levied stays verified: false. |
| 4 | `capacity-increase` | **Applied.** I dropped both supply cost lines, so `adds.cost_plan` is gone and `built_in.cost_plan` with it. The condition stays in the `why`; `heating-electrification` is the only trigger that lifts P06. I also dropped the duplicate structural activity; the structural RFP must_include carries the change. Source is now `electrical-services.yaml positions utility`. DOMAIN §9 row updated. |
| opt | refrigerant `removes` | **Applied.** `basement-plant-room` and `flammable-refrigerant` now remove `mechanical.refrigerant-condensate` `replace`. |

Disputed: none.

New counts: 15 work items, 6 packages (6 not expected), 28 cost lines (8 not expected), 47 activities,
5 RFPs (4 with a template plus hazmat), 18 triggers, 28 gaps, 8 variants. A script check finds no
dangling `after` and no trigger `removes.program` label missing from the base.

Validator: `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T07-mechanical-plant-upgrade/shape.yaml`
prints `ok`.
