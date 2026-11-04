# T03 Commercial office fit-out: independent review

Reviewer: independent agent acting as a senior Sydney PM for tenant fit-outs in A-grade towers. Brief:
`testing/synthetic/briefs/02-review.md`. Read on 30 October 2026 against `main` plus the working tree. This
is the only file written. What I ran:

- the validator;
- read-only Python over the YAML;
- greps of the knowledge and the Go;
- two read-only Go runs. Each test file sat in the scratchpad and reached the package through
  `go test -overlay`; nothing was written in the repo, and `git status` confirms no new files. The runs:
  - `internal/program` `Generate` over the real catalogue, with the 34 base work items, refurb, cdc,
    traditional, Class 5 and existing building;
  - `internal/rfp` `Resolve` on the base facts for the seven templated disciplines.

## Verdict: **FIX**

The map is broad and well sourced. It covers all 13 top-level systems, the landlord as a third party, the
CDC chain and the occupied-building interfaces. Almost every built-in claim checks out. The Go runs
reproduced all 96 `app_differs` rows of the seven templated RFPs exactly, plus the certifier's three stage
rows. It is not ready to be an answer key, for these reasons:

- the program run shows three app ordering defects. The key catches only one and calls it unrun: services
  rough-in, fire stopping and sprinklers all land on the wrong side of the ceilings;
- the CDC is lodged before the landlord gives owner's consent;
- occupied-building protection is put under the wrong system;
- some smaller items about certification, scoring and testability.

Nothing is wrong at the root.

## Rulings on the author's open questions

| Question | Ruling |
|---|---|
| Sprinkler pressure-test hold point against its fragment | **Confirmed by a run.** `Generate` emits note `link_skipped` on `fragment:fr.ceilings-fitout`: "Link from Hold point: a recorded pressure test of the sprinkler pipework to interiors.ceilings left out: it would make a loop." `fr.sprinklers-fitout` attaches after `fr.ceilings-fitout` (`knowledge/program/fragments.yaml` line 2304), so the fragment link ceilings → sprinklers is placed first. `holdPointLinks` then adds sprinklers → hold, and the guard drops hold → ceilings. The hold row ends with no successor, and the ceilings are not held. The root cause is the same as in the T13 review (finding 6). The hold point keys on the whole sprinkler system, while the library has no row for the concealed-pipework phase. The mechanism differs: T13's fragment has a fit-off row, so the hold drags the ceilings after fit-off; T03's single row sits after the ceilings, so the hold link is dropped. See finding 2. |
| `site.impact-protection` for occupied-building protection | **Wrong; drop the work item.** The system is "Vehicle and forklift impact protection": bollards, crash barriers and column guards (`knowledge/clusters/structure/systems.yaml` line 611). Its terms are "Bollards" and "Rail" (`knowledge/works/system_terms.yaml`). No fit-out document says that, so a scenario written to carry the item would read falsely, and a PM would strike "Vehicle and forklift impact protection" from a tower fit-out scope. Temporary protection of lifts, lobbies and the dock is the builder's preliminaries. T05, T07 and T14 treat it that way and record the fragment's dependency only as a gap. See finding 4. |
| Construction and witness stages for the services engineers | **Right for mechanical, electrical and fire services; keep `stages_include`.** On a traditional, fully documented A-grade fit-out, the tenant buys the engineers through construction. They do shop drawing review, site inspections, witness testing and the integrated fire test, and they give the design and installation certificates that the landlord's fit-out guide and the certifier ask for. Convention C1 (services engineers optional under D&C novation) does not apply to a traditional job. Hydraulic as `option` is right for a kitchenette. The agent-draft keys `mechanical-office-design` and `fire-services-office-design` put these stages as options only because their scenario sets the boundary to end at tender. That is a user choice, not the default practice. Two follow-ups are in finding 9. |
| Clause numbers, LSL threshold, contributions, all `verified: false` | **Right to keep them unverified** (AGENTS.md rule 5; no owner ruling covers them). LSL: the base job is above any plausible threshold. PROJECT_TYPES.md puts the type at $0.3–5M, which is above the $250,000 that T12 carries for owner confirmation, so the line is unconditional here. Contributions: as I recall, EP&A Act s7.21 makes the certifier impose a contributions condition on a CDC where the council's plan levies one (verified: false). The line is therefore right in principle, but it depends on the LGA. Two label fixes are in finding 12. |

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T03-office-fitout/shape.yaml`,
run from the repo root, gives `ok` (exit 0). Counts: 34 work items, 13 packages, 35 cost lines,
69 activities (as DOMAIN §7 says), 11 RFPs and 13 triggers. Every `after` names an activity in the list.

## 2. Findings

1. **`shape.yaml program` "Install ceilings" `after`, and `gaps`: services rough-in lands after the
   ceilings, and the key does not say so.**
   - **Evidence.** Six fragments attach `after: fr.strip-out-fitout`: sanitary, cold water, hot water,
     electrical, mechanical and ventilation. With no strip-out in the base, none of them has a predecessor.
     `chainLinks` (`internal/program/generate.go:976`) then chains each one behind the row before it, in
     library order. The run gives:
     - sanitary "Rough-in" after "Hang doors";
     - electrical "Rough-in" after the hot-water unit;
     - "Alter ductwork and pipework" after "Install luminaires", which itself follows "Install ceilings".
   - **The effect.** Ceilings start on day 8 of construction and ductwork on day 14. The key's
     ceilings-after-ductwork link will show as violated. Electrical and sanitary rough-in are only reached
     through the fire-stopping chain, so the scorer will put their inversion down to fire stopping.
   - **Change:**
     - add "Electrical rough-in" and "Kitchenette sanitary and water rough-in" to "Install ceilings"
       `after`;
     - add a `gaps` row: `{area: program, what: "With no strip-out, every fit-out services fragment
       attached after fr.strip-out-fitout loses its anchor; the chain step puts sanitary rough-in after
       doors, electrical rough-in after the hot-water unit and ductwork after luminaires, so services
       rough-in follows the ceilings (run 30 Oct 2026)", evidence: "knowledge/program/fragments.yaml
       fr.electrical-fitout, fr.mechanical-fitout attach; internal/program/generate.go chainLinks"}`;
     - add a matching item in DOMAIN §11.

2. **`shape.yaml program`, the sprinkler rows and the `gaps` row: confirm the hold-point defect, and model
   the two sprinkler phases.**
   - **The domain sequence.** Sprinkler pipework is altered above the ceiling, the altered section is
     pressure tested, the ceilings close, then heads are fitted off and the system is restored and tested.
     The key has one row, "Alter and test sprinklers", before the test. That row is matched to the app's
     row, which correctly sits after the ceilings as a fit-off.
   - **Change:**
     - add `{label: "Alter sprinkler pipework above the ceiling", stage: construction, fragment: null,
       after: ["Book services shutdowns, fire-system isolations and hot-works permits"], built_in: "no",
       why: "The library has no concealed-pipework row; fr.sprinklers-fitout is one row after the
       ceilings"}`;
     - set "Sprinkler pressure test before ceilings close" `after: ["Alter sprinkler pipework above the
       ceiling"]`, and in its `why` replace "(read from generate.go, not run)" with: run 30 Oct 2026, note
       link_skipped "Link from Hold point: a recorded pressure test of the sprinkler pipework to
       interiors.ceilings left out: it would make a loop";
     - set "Alter and test sprinklers" `after: ["Install ceilings"]`, with `why`: "Fit off heads in the new
       ceiling and restore; the app's single row is in the right place for fit-off";
     - add "Alter and test sprinklers" to the `after` of "Fire services commissioning and integrated test
       with the base-building panel and EWIS". Flow switches and valve monitoring are part of the cause and
       effect;
     - in `gaps`, replace "(read, not run)" with "(run 30 Oct 2026; same root as T13: the hold point keys
       on the whole system, and the library has no concealed-pipework row)";
     - in DOMAIN §7 "Hold points" and §12, change "read, not run" to "confirmed by a run" and quote the
       note.
   - The same two-phase gap exists for detection: cabling before the ceilings, heads after. It has no
     hold point. One line in DOMAIN §11 is enough.

3. **`shape.yaml program` "Fire-stopping inspection before concealment" and "Seal service penetrations":
   the order was not run. The run shows it is wrong in two ways.**
   - **Evidence.** `if.fire-stopping-before-concealment` has `hold_point: {after:
     fire-passive.penetrations, before: interiors}` (`knowledge/clusters/fire/interfaces.yaml`), and
     `tagged("interiors")` returns every interiors row.
     - The hold therefore precedes "Frame and line partitions", "Install ceilings", "Hang doors", "Procure
       and deliver" (off-site joinery), joinery "Install", painting and floors.
     - `fr.fire-stopping` has no `attach`. The chain step's floors → seal link would close a loop, and a
       chain link is skipped silently, so "Seal service penetrations" has no predecessor. It starts on
       day 0 of construction, before any partition, cable or duct exists.
   - **Change:**
     - in the inspection row's `why`, replace "its order against the interiors rows was not run" with:
       "Run 30 Oct 2026. The app seals at construction start and holds every interiors row behind the
       inspection, including partitions and off-site joinery";
     - add "Alter ductwork and pipework", "Kitchenette sanitary and water rough-in" and "Alter sprinkler
       pipework above the ceiling" to "Seal service penetrations" `after`;
     - add a `gaps` row: `{area: program, what: "if.fire-stopping-before-concealment holds the whole
       interiors tree (partitions, joinery procurement, doors, paint, floors), and fr.fire-stopping has no
       attach, so sealing starts at construction start before the services it seals", evidence:
       "knowledge/clusters/fire/interfaces.yaml if.fire-stopping-before-concealment; knowledge/program/
       fragments.yaml fr.fire-stopping"}`;
     - remove the matching bullet from DOMAIN §12's unsure list.

4. **`shape.yaml work_items` `site.impact-protection`, and the rows that rest on it: wrong system.** The
   ruling above has the reasoning. **Change:**
   - delete the `site.impact-protection` work item;
   - "Install hoardings and protection to occupied areas": keep `built_in: "partial"` and
     `fragment: fr.occupied-protection`. Change `why` to: "The fragment exists but needs a
     site.impact-protection item (vehicle and forklift impact protection), which a correct fit-out scope
     does not have";
   - cost line "Protection of common areas and tenant signage": rename it "Tenancy signage", keeping the
     alias "Site works and external works" and `built_in: "partial"`. Fold "protection of lifts, lobbies
     and the loading dock" into the `why` of "Preliminaries, supervision and after-hours premiums";
   - in DOMAIN §2 site, replace the `site.impact-protection` bullet with "Temporary protection of the goods
     lift, lobby and corridors is the builder's preliminaries. The app's occupied-protection fragment hangs
     off `site.impact-protection`, which is the wrong system (gap)";
   - keep both `gaps` rows that name it.

5. **`shape.yaml program` "Prepare and lodge the complying development certificate" `after`: lodged
   before the landlord consents.**
   - **Evidence.** A CDC application needs the owner's consent (EP&A (Development Certification and Fire
     Safety) Regulation 2021, clause verified: false). In a leased tower the owner is the landlord. The
     landlord signs only after its base-building engineers approve the detailed design. The row now
     follows only "BCA and access compliance reports".
   - **Change:**
     - set `after: ["BCA and access compliance reports", "Landlord detailed design approval"]`;
     - in `why`, add "landlord signs owner's consent (verified: false)";
     - in DOMAIN §5, merge step 1's detailed-design approval into step 2 as "landlord's detailed design
       approval and owner's consent → CDC lodgement".

6. **`shape.yaml program` and DOMAIN §5 item 5: no commencement inspection.**
   - **Evidence.** As I recall, the Class 5–9 list in the 2000 Regulation, which the 2021 Regulation
     carried over, is: the commencement of work, before covering stormwater connections (not applicable
     inside a tower), and the final inspection. All of it is verified: false. Owner ruling R7 covers only
     Class 1 and 10, and the T13 review used commencement for Class 2–4.
   - **Change:**
     - add `{label: "Commencement inspection by the principal certifier", stage: construction, fragment:
       null, after: ["Appoint principal certifier and give notice of commencement"], built_in: "no", why:
       "Class 5–9 list: commencement and final (verified: false; owner to confirm, as R7 did for Class 1
       and 10)"}`;
     - add it to "Establish the site" `after`;
     - in DOMAIN §5 item 5, change the text to "commencement and final".

7. **`shape.yaml procurement` and `triggers`: core holes through the slab have no structural cover.**
   - **Evidence.** The base phrasing has "core drilling" (`scenario_inputs`, access constraints) and "floor
     boxes" (`electrical.cabling-wiring` `why`). In a 2008 tower the floor is usually post-tensioned. A
     new floor box or pipe penetration needs the slab scanned and the landlord's structural sign-off, and
     often the tenant's structural engineer. `gaps` already says no structural engineer follows on refurb.
   - **Change:**
     - in the base, change the cabling `why` to "power poles or the existing floor boxes";
     - add a trigger `slab-core-drilling`:
       - evidence: "new floor boxes cored through the post-tensioned slab", "core holes for the
         kitchenette waste through the slab", "scan the slab before coring";
       - negation: "power is from power poles; no new slab penetrations", "existing floor boxes are
         reused";
       - adds: procurement `consultant.structural`; program "Scan the slab and obtain the landlord's
         structural approval for core holes", before the rough-in rows; cost_plan "Structural engineer",
         "Slab scanning and core drilling"; rfps structural "Core-hole locations against post-tensioning
         tendons and the landlord's structural approval";
       - built_in: `{procurement: "no", program: "no", cost_plan: "partial", rfps: "partial"}`, source
         `knowledge/rfp/structural.yaml`;
     - add it to `variants`.

8. **`shape.yaml rfps` consultant.certifier `why` and `expect.sources`, and DOMAIN §8: "the owner's key"
   is not an owner's key.**
   - **Evidence.** `data/eval/rfp/answer-keys/bca-unsupported-cdc.yaml` has `reviewed: false` and
     `label_source: agent-draft`. Its job is `work_type: new`, not refurb, with `bca_advisory: 'no'`. It
     lists `pathway_unsupported` under `checks`, not `blocking`. The run does show `pathway_unsupported`
     as blocking.
   - **Change:**
     - replace "The owner's key" with "The agent-draft key";
     - in DOMAIN §8, replace "the owner's key ... says the block is the intended answer" with "an
       agent-draft key ... treats the block as the safe answer until a CDC scope exists; for the owner to
       confirm";
     - keep the treatment, for consistency with T12.

9. **`shape.yaml rfps` electrical and fire services `expect.sources`: the inclusion of construction-phase
   services is not justified, and the documents give a reader no way to see it.** Follows the ruling.
   **Change:**
   - add to both the `sources` line that mechanical carries: "Construction and witness included: the
     landlord and certifier need the tenant's engineer to inspect and certify the installation";
   - add to `scenario_inputs` (no profile key; the note says so) the phrasing "The services engineers'
     scope runs through construction-phase inspections, witness testing and completion certification", so
     the scenario's consultant brief states it.

10. **`shape.yaml rfps` `expect.positions`: rows the app emits are left unscored.** The run shows these
    `O` or `ask` rows. The shape neither lists them nor expects them absent. **Change** (add each one, with
    an `app_differs` row):
    - mechanical: `continuation: absent` (app O), `seasonal: absent` (app O), `remediation: absent`
      (app O);
    - electrical: `continuation: absent` (app O);
    - hydraulic, a kitchenette that buys none of these:
      - `process: absent` (app ask; `hydraulic-office-fitout.yaml` already lists it absent);
      - `utility: absent` (app O);
      - `alternative_sources: absent` (app O);
      - `staging: absent` (app O).

11. **`shape.yaml cost_plan` "Owner supply items (furniture, workstations, AV and IT equipment)": one
    expected line against four app lines.** `ParseOwnerSupply` splits on commas and " and "
    (`internal/costs/budget.go:162`). The phrasing yields four lines, "Owner supply: loose furniture",
    "Owner supply: workstations" and so on, and `pair` is one-to-one, so three show as extras. **Change:**
    make it four expected lines, each with `built_in: "partial"`, the same source and the label the parser
    gives.

12. **`shape.yaml cost_plan` contributions line, and two rule numbers.** **Change:**
    - rename "Section 7.12 development contributions" to "Development contributions payable on the CDC
      (s7.11 or s7.12, verified: false)", the form T12 uses;
    - in its `why`, add "decided by the LGA's contributions plan; the scenario must name the LGA";
    - in the "Long service levy" `why`, add "the type's $0.3M lower bound (PROJECT_TYPES.md) exceeds any
      plausible threshold, so the line is unconditional in the base";
    - add "(verified: false)" after "AS 1428.1" in the access consultant `why` and in the `tenant-amenities`
      access `must_include`. DOMAIN §12 lists it as unverified; the shape does not.

13. **`shape.yaml triggers[asbestos-in-tenancy]` `evidence`: a date is not a finding.** "The building was
    completed in 1979" establishes that a register applies and that a survey is due. It does not establish
    asbestos in the tenancy, licensed removal or clearance. **Change:**
    - drop that phrase from `evidence`;
    - add a separate `scenario_inputs` row: `det.existing_building_year` 1979 → "Check the landlord's
      hazardous materials register" plus "Hazardous-materials survey" only;
    - keep the register and ceiling-space findings as the trigger evidence.

14. **`shape.yaml triggers[da-pathway]`: no upgrade of the existing building.** Under a DA, the consent
    authority may require the existing building's fire safety to be upgraded (the successor to old cl 94,
    verified: false). A CDC carries no such power. **Change:**
    - add to `adds.rfps` `{discipline: consultant.bca, must_include: ["Assessment of the existing
      building's fire safety against any upgrade the consent requires"]}`;
    - add to `adds.authority` "Council may require a fire safety upgrade of the existing building
      (verified: false)".

15. **DOMAIN §11 and §12: housekeeping after findings 1–4.** **Change:**
    - add finding 1 (services after ceilings) and finding 3 (fire stopping at start) to §11, ranked just
      below item 5;
    - change item 6 to "confirmed by a run";
    - in §12's unsure list, keep only the services-stages and LSL items, now ruled.

## 3. Built-in claims checked

**Procurement** (`knowledge/works/package_defaults.yaml`)

- ✔ Interior Design `yes`, `baselines[4]` (commercial, institution or mixed, refurb).
- ✔ Mechanical, Fire Services, BCA and Access `yes`, `baselines[3]`; Electrical, Hydraulic and Certifier
  `yes`, `baselines[2]`.
- ✔ Architect not expected, `app_suggests: yes`, `baselines[2]` (every refurb).
- ✔ `trade.main_works` `partial`: `ProposeHeadContract` builds lines from gaps only.
- ✔ `no` spot checks: project manager, QS, acoustic and ICT. No baseline, scope addition or complexity
  addition names them.
- ✔ Triggers:
  - `da-pathway` procurement `yes` (`complexity_additions[0]`, da → Town Planner);
  - `heritage-item` `yes` (`[1]` local_item → Heritage, plus `[0]`);
  - `tenant-amenities` `yes` (Hydraulic is already `baselines[2]`);
  - `design-and-construct` `no` (the app keeps the services consultants).

**Cost plan** (`knowledge/costs/starter_lines.yaml`, `elements.yaml`)

- ✔ `cc-fees` `partial`: always on, and labelled for a construction certificate.
- ✔ `da-fees` only for da.
- ✔ `service-authority` always on, so it is right as not expected.
- ✔ Design contingency, construction contingency, escalation and exclusions: always.
- ✔ The eight consultant fee lines follow the packages. ✔ F02 Architect is not expected.
- ✔ Code 01 "Site works and external works", and codes 05–11 `partial`.
- ✔ Code 13 items: accessible access and egress doors are costed; exits are not costed.
- ✔ Owner supply `partial` (`applyBriefFigures`, `ParseOwnerSupply`); the scoring problem is finding 11.
- ✔ `no` spot checks: no starter or element line for LSL, contributions on a CDC, landlord fees or
  charges, preliminaries, nominated contractors or provisional sums.

**Program** (`knowledge/program/`, plus the run)

- ✔ Skeleton `sk.refurb-traditional-cdc` (run).
- ✔ All 15 process fragments fire (run). `fr.heritage-referral` stays false with `heritage_status: none`.
- ✔ `fr.existing-conditions` adds "Hazardous-materials survey" with no year test (run).
- ✔ These work-item fragments fire on their items (run): partitions, ceilings, doors, sanitary, cold
  water, hot water, electrical, lighting, mechanical, ventilation, sprinklers, detection, comms, joinery,
  painting, floors, fire stopping, occupied protection.
- ✘ Their implied order: the services rough-in rows land after the ceilings (finding 1).
- ✔ The sprinkler hold point is `partial`, and the defect is confirmed (run; finding 2).
- ✔ The fire-stopping hold point is `partial`. ✘ Its order is wrong, both against the interiors rows and
  at construction start (finding 3).
- ✔ `fr.occupied-protection` conditions: an impact-protection item, class 5, refurb and an existing
  building.
- ✔ `fr.strip-out-fitout` needs `remove`.
- ✔ `no` spot checks: no fragment tags emergency lighting, occupant warning, internal glazing, BMS, access
  control, signage, local exhaust, air distribution, AV or ICT rooms (grep count 0 each).
- ✔ No fragment names a landlord, LSL, final fire safety certificate, principal certifier, notice of
  commencement, final inspection, condition survey or move-in.
- ✔ Triggers:
  - D&C refurb falls to `sk.generic` (no refurb D&C skeleton), and `fr.novation` fires on
    design_construct;
  - exempt keeps `fr.occupation-certificate` (work type only) and gets `fr.exempt-confirmation`;
  - da adds `fr.da-lodgement`, `fr.da-exhibition` and `fr.construction-certificate`;
  - `fr.wet-area-fitout` and `fr.structure-alterations` fire for refurb Class 5.
- ✔ `profileFloorArea` reads only `hdr.scale.gfa_sqm`. The office subclass records `nla_sqm`.

**RFPs** (the run reproduced every `app_differs` row)

- ✔ Mechanical: boundary inception–design; 17 rows.
- ✔ Electrical: 18 rows, including utility I and plant I.
- ✔ Fire services: 19 rows.
- ✔ Hydraulic: 14 rows, including pumps, rainwater and roof I.
- ✔ BCA: 9 rows; `pathway_unsupported` is blocking, and a4 is an option. The certifier's a1, a3 and a4
  rows match the same run.
- ✔ Access: 7 rows; occupied is I.
- ✔ Cost planning: 12 rows; `existing_costs` is I.
- ✔ The `existing` rows are absent while investigation is `ask`.
- ✘ The certifier's "owner's key" is an agent draft (finding 8).
- ✔ No template exists for interior design, project manager or acoustic (`knowledge/rfp/` listing).

**Keyword traps** (`knowledge/profile/scope_defaults.yaml implied_by`, `flag_readings.yaml`)

- ✔ The stair, wet-room, kitchen, strip-out, party/common wall and BASIX patterns match as DOMAIN §10
  says. "Kitchenette" and "toilet" do not match.
- ✔ `sustainability_target` triggers on "nabers", so the base-building rating trap is real.

## Round 1 response (author, 30 October 2026)

All 15 findings applied; none disputed. Validator `ok`.

1. Applied. "Install ceilings" now follows electrical and kitchenette rough-in; new `gaps` row (services
   fragments lose their strip-out anchor, run 30 Oct 2026); DOMAIN §7 and §11 item 6.
2. Applied. New row "Alter sprinkler pipework above the ceiling" before the pressure test; the test quotes
   the `link_skipped` note; "Alter and test sprinklers" is now fit-off after the ceilings and feeds the
   integrated fire test; `gaps` row says "run 30 Oct 2026; same root as T13"; detection two-phase gap noted
   in the same row and in DOMAIN §11.
3. Applied. "Seal service penetrations" now follows ductwork, kitchenette rough-in and sprinkler pipework;
   the inspection row states the run result; new `gaps` row; DOMAIN §12 unsure bullet removed.
4. Applied. `site.impact-protection` work item deleted; hoardings row `why` reworded; cost line renamed
   "Tenancy signage"; protection folded into preliminaries; DOMAIN §2 rewritten; both gaps kept.
5. Applied. CDC lodgement follows "Landlord detailed design approval"; owner's consent (verified: false)
   in the `why` and DOMAIN §5 step 2.
6. Applied. "Commencement inspection by the principal certifier" added before "Establish the site";
   DOMAIN §5 item 5 now "commencement and final".
7. Applied. Cabling `why` now "power poles or the existing floor boxes"; new trigger `slab-core-drilling`
   with the given evidence, negation, adds and built_in; added to `variants`.
8. Applied. "The agent-draft key" in the certifier `why`, `sources` and DOMAIN §8; treatment kept.
9. Applied. `sources` added to electrical and fire services. The `scenario_inputs` row needs an app row
   key (the validator rejects a keyless row), so it is keyed to `hdr.cond.procurement_route`, the fact the
   scope rests on, with a note saying no key records consultant stage scope.
10. Applied. Mechanical continuation, seasonal, remediation; electrical continuation; hydraulic process,
    utility, alternative_sources, staging: each expected absent with an `app_differs` row.
11. Applied. Four expected owner-supply lines with the parser's labels.
12. Applied. Contributions line renamed to the T12 form with the LGA note; LSL `why` notes the line is
    unconditional; AS 1428.1 marked verified: false in both places.
13. Applied. "Completed in 1979" dropped from the evidence (replaced by a survey finding, to keep two
    phrasings); new `scenario_inputs` row for a 1979 building giving the register check and survey only.
14. Applied. da-pathway adds the BCA existing-building fire safety upgrade item and the authority line.
15. Applied. DOMAIN §11 re-ranked (services rough-in, fire stopping and the confirmed hold point at 6–8);
    §12 unsure list keeps only the ruled services-stages and LSL items.

Coordinator item: the BCA consultant's `text.blocking` difference is now
`{key: blocking.pathway_unsupported, app: blocking, expect: not blocking}`. The certifier expects the block,
so it has no blocking difference; its `text.cdc_scope` row is about scope and stays text.

New counts: 33 work items, 13 packages, 38 cost lines, 71 activities, 11 RFPs, 14 triggers, 25 gaps,
9 variants.

## Round 2 review (4 November 2026)

I re-read `shape.yaml`, `DOMAIN.md` and the Round 1 response. I checked them against `OWNER_RULINGS.md`
(R1–R10) and `CONVENTIONS.md` (C1–C5; C4 and C5 are new since round 1). The validator, run from the repo
root, gives `ok`, exit 0. A read-only script check over the YAML counts 33 work items, 13 packages,
38 cost lines, 71 activities, 11 RFPs, 14 triggers, 25 gaps and 9 variants, as the response says. No
`after` names a missing activity, and no activity label repeats.

### Verdict: **FIX**

Every round 1 finding is applied correctly, and none was disputed. The program order now reads right.
Sprinkler pipework comes first, then the pressure test, then the ceilings, then fit-off. Rough-in and
sealing come before the inspection and the ceilings. Three things are left, all small:

- the base text still contains trigger evidence;
- C4 has made the long service levy threshold a checked fact;
- the exempt variant does not remove the rows the fixes added.

### Fixes checked

- **F1 ✔.** "Install ceilings" follows electrical and kitchenette rough-in. The `gaps` row and DOMAIN §7
  and §11 item 6 match.
- **F2 ✔.**
  - "Alter sprinkler pipework above the ceiling" (`no`) comes before the pressure test.
  - The test quotes the `link_skipped` note.
  - "Alter and test sprinklers" follows the ceilings and feeds the integrated test.
  - The `gaps` row says "run 30 Oct 2026; same root as T13".
  - DOMAIN §7 and §11 item 8 say "confirmed by a run" and note the two-phase gap in detection.
  - The hold row has no alias. The scorer's `interface` matcher pairs it through `source`
    (`tools/score_shapes.py`), so it needs none.
- **F3 ✔.** "Seal service penetrations" follows ductwork, kitchenette rough-in and sprinkler pipework. The
  inspection `why` states the run result, the new `gaps` row is in, and the §12 bullet is gone. Its wording
  is finding 5.
- **F4 ✔.** The work item is gone. The hoardings `why`, "Tenancy signage" and the preliminaries `why` are
  reworded. DOMAIN §2 and §7 are rewritten. Both gaps are kept. One gap phrase is out of date (finding 4).
- **F5 ✔.** CDC lodgement follows "Landlord detailed design approval". Owner's consent (verified: false)
  is in the `why`, in DOMAIN §5 step 2 and in DOMAIN §7 step 3.
- **F6 ✔.** The commencement inspection is added, and "Establish the site" waits for it. DOMAIN §5 item 5
  says "commencement and final". The certifier RFP wording is finding 7. The exempt variant is finding 3.
- **F7 ✔.** The cabling `why` is reworded. The `slab-core-drilling` trigger is added as specified and is in
  `variants`. The base still says "core drilling" elsewhere (finding 1).
- **F8 ✔.** The certifier `why`, `sources` and DOMAIN §8 say "agent-draft key".
- **F9 ✔ (accepted as applied).** The `sources` lines are in. The validator rejects a `scenario_inputs` row
  with no key, so the row is keyed to `hdr.cond.procurement_route`, with a note. That is the right home;
  T10 F11 did the same.
- **F10 ✔.** All eight positions are expected `absent`, each with an `app_differs` row.
- **F11 ✔.** There are four owner-supply lines with the parser's labels. The phrasing risk is finding 9.
- **F12 ✔.** The contributions line is in the T12 form, with the LGA note. AS 1428.1 is marked verified:
  false in both places. Finding 2 replaces the LSL wording.
- **F13 ✔.** The date is out of the evidence, replaced by a survey finding. The separate 1979
  `scenario_inputs` row is in, and DOMAIN §9 matches.
- **F14 ✔.** `da-pathway` adds the BCA item on upgrading the existing building, and the authority line.
- **F15 ✔.** DOMAIN §11 is re-ranked (items 6–8). The §12 unsure list keeps the two ruled items.
- **Coordinator item ✔.** The BCA consultant's difference is `blocking.pathway_unsupported`: the app
  blocks, and the key expects no block. The certifier expects the block, so it has no blocking difference.

### Rulings and conventions

- ✔ **R3:** declarations are absent for Class 5.
- ✔ **R4, R5:** HBCF and BASIX are not applicable.
- ✔ **R7:** not applied to Class 5. The Class 5–9 list stays verified: false.
- ✔ **R8:** the defects liability period stays verified: false.
- ✔ **C2:** the rating trigger uses `consultant.esd`.
- ✔ **C3:** BCA and the certifier share one template.
- ✔ **C4, fire control centre:** a base-building measure, already out of scope in DOMAIN §2.
- **C5:** does not apply. A CDC has no CC, and the DA variant has one CC with no early works.
- ✘ **C4, LSL:** not yet applied (finding 2).
- ✘ **C1:** not addressed in the D&C variant (finding 8).

### Findings

1. **`shape.yaml scenario_inputs` (`hdr.cond.access_constraints` phrasing) and DOMAIN §1: the base still
   says "core drilling".**
   - **Problem.** "Noisy works, core drilling and deliveries are restricted to after hours" tells a reader
     that cores are drilled. That is indirect evidence for `slab-core-drilling`, whose negation is "no new
     slab penetrations". A base scenario written from this row would carry the trigger.
   - **Change.** Use "Noisy works and deliveries are restricted to after hours and weekends under the
     landlord's building rules". Make the same edit in DOMAIN §1.

2. **`shape.yaml` "Long service levy" cost `why` and the "Pay long service levy" `why`, and DOMAIN §5
   item 3, §6 and §12: C4 now gives the threshold.**
   - **Change.**
     - Replace "threshold verified: false" and "exceeds any plausible threshold" with: "0.25% of the cost
       of building work of $250,000 or more including GST, paid before work starts (C4, checked against
       nsw.gov.au); the type's $0.3M lower bound is above it, so the line is unconditional in the base".
     - In DOMAIN §12, take the threshold off the unverified list, and change the bullet to "settled by C4".
     - Keep the program row before "Certifier determination". The levy must be paid before the
       certificate issues (clause verified: false).

3. **`triggers[exempt-minor-refresh]` `removes`: the certification rows added in round 1 survive the exempt
   path.**
   - **Problem.** Exempt development has no principal certifier and no inspections. Its own negation says
     it alters no fire safety measure. Contributions attach to a CDC or a DA, not to exempt work.
   - **Change.**
     - Add to `removes.program`: "Appoint principal certifier and give notice of commencement",
       "Commencement inspection by the principal certifier" and "Final fire safety certificate".
     - Add to `removes.cost_plan`: "Development contributions payable on the CDC (s7.11 or s7.12,
       verified: false)".
     - Add to the `why`: "LSL still applies at $250,000 or more (C4); a carpet-and-paint refresh may fall
       below it".
   - **The same double count in `triggers[heritage-item]`.** It adds "DA fees and contributions" but keeps
     the CDC contributions line. Add that line to its `removes.cost_plan`, as `da-pathway` does.
   - Update DOMAIN §9 to match.

4. **`shape.yaml gaps`, the cost_plan row "P06 service authority fees…": out of date after F4 and F12.**
   - **Problem.** It says "protection and signage land in 01 Site works", but protection is no longer a
     work item. It also says "no s7.12 contributions", where the line now names s7.11 or s7.12.
   - **Change.** Use "signage lands in 01 Site works and external works" and "no s7.11 or s7.12
     contributions for a CDC". DOMAIN §11 item 11 is already right.

5. **The `why` of "Fire-stopping inspection before concealment", and DOMAIN §7 step 6: "holds only the
   linings and ceilings".**
   - **Problem.** In the key, "Frame and line partitions" comes before sealing. That is right: a
     penetration in a rated lined wall is sealed after lining. So the inspection does not hold linings.
   - **Change.** Use "holds only the ceilings and bulkheads that conceal the penetrations" in both places.

6. **`shape.yaml program`: two construction rows have no successor, and one is not before the ceilings.**
   - **Problem.**
     - "Alter exhaust ventilation" follows only the hoardings. The kitchenette and print-room exhaust runs
       above the ceiling into the base-building riser, through a rated wall. It must come before the
       ceilings and before sealing.
     - Nothing follows "Install glazed partitions and manifestation" or "Install access control and
       integrate with base-building security", so practical completion does not wait for them.
   - **Change.**
     - Add "Alter exhaust ventilation" to the `after` of "Install ceilings" and of "Seal service
       penetrations".
     - Add both of the other rows to the `after` of "Practical completion inspection".
   - The defect predates round 1. This round's script check found it.

7. **`shape.yaml rfps` consultant.certifier `must_include`: the inspections line predates F6.**
   - **Change.** Replace "Mandatory inspections, incl. the final inspection (Class 5 list verified:
     false)" with "Mandatory commencement and final inspections (Class 5 list verified: false)".

8. **`triggers[design-and-construct]` `why`: C1 is not addressed.**
   - **Problem.** The variant removes all four services engineers, but C1 makes them `option` on a D&C job
     with novation. Here nothing is novated: the interior designer gives the tenant's design intent, and
     the contractor's engineers design.
   - **Change.** Keep the `removes`. Add to the `why`: "C1 does not apply: no tenant engineer is appointed
     to be novated; whether a tenant keeps a services reviewer on a D&C fit-out is an owner question".

9. **`shape.yaml scenario_inputs` (`proj.owner_supply_stated`): the four expected labels assume the read
   value is the list alone.**
   - **Problem.** `splitOutsideBrackets` splits at the first comma (`internal/costs/budget.go`). If the
     stored value keeps the lead-in "Tenant supply, outside the fit-out contract: …", the app writes
     "Owner supply: Tenant supply" and "Owner supply: outside the fit-out contract: loose furniture".
   - **Change.** Add to the note: "the expected labels assume the read value is the list after the colon;
     state the list in its own sentence, e.g. 'Tenant-supplied items: loose furniture, workstations, AV
     equipment, IT active equipment'". The other option is to accept the lead-in as a scored extra.

### Built-in claims re-checked this round

- ✔ The scorer's `interface` matcher pairs the hold-point rows through `source`.
- ✔ `ParseOwnerSupply` writes one "Owner supply: <item>" line per item, provided the value has no lead-in
  (finding 9).
- ✔ `slab-core-drilling` is `no` for program and procurement. No fragment or package rule names a slab
  scan, or a structural engineer on refurb. Its cost and RFP claims are `partial`, on the same pattern as
  `heavy-floor-loads`.
