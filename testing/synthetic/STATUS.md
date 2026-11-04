# Status

The orchestrator updates this file after every task. Anyone resuming the
work starts here, then reads `README.md`.

Cell values: `—` not started, `run` in progress, `ok` done, `fix` needs
rework (see that type's REVIEW.md), `gate` waiting for the owner.

## Shared tasks

| Task | State | Output | Note |
|---|---|---|---|
| 0 Scaffolding and shape validator | ok | `tools/validate_shape.py`, briefs, this file | 75 tests pass (validator + scorer) |
| 1 Knowledge index | ok | `reference/KNOWLEDGE_INDEX.md` | brief `briefs/00-knowledge-index.md` |
| 4 Shape scoring rules | ok | `SCORING.md`, `tools/score_shapes.py` | TDD from the T01 shape |
| 5 Scenario brief and expected JSON schema | ok | `briefs/03-scenarios.md`, `03r`, `tools/validate_expected.py` | 23 tests; T01, T05 and T09 done |
| 6 Rendering brief and leak lint | ok | `briefs/04-render.md`, `04r`, `tools/lint_corpus.py` | 10 tests; T01 rendered |
| 7 Harness | ok | `harness/` | serve.sh (own DB on :8091), run.py, score.py; 10 tests |
| 8 Pilot run | gate | `runs/20261031-pilot/REPORT.md`, `runs/20261104-t05-t09/REPORT.md`, `FINDINGS.md` | T01 run 31 Oct; T05 and T09 run 4 Nov; owner gate 3 pending |

## Project types

| ID | Type | Map | Validate | Review | Owner | Scenarios | Docs | Run |
|---|---|---|---|---|---|---|---|---|
| T01 | New detached house (pilot) | ok | ok | ok | ok | ok | ok | ok |
| T02 | Apartments | ok | ok | ok | gate | — | — | — |
| T03 | Office fit-out | ok | ok | ok | gate | — | — | — |
| T04 | Warehouse | ok | ok | fix | — | — | — | — |
| T05 | House alterations and additions (pilot) | ok | ok | ok | ok | ok | ok | ok |
| T06 | Sprinkler pump and tank upgrade | ok | ok | fix | — | — | — | — |
| T07 | Mechanical plant upgrade | ok | ok | ok | gate | — | — | — |
| T08 | Roof remediation | ok | ok | ok | gate | — | — | — |
| T09 | Waterproofing remediation (pilot) | ok | ok | ok | ok | ok | ok | ok |
| T10 | Knock-down rebuild | ok | ok | ok | gate | — | — | — |
| T11 | Dual occupancy or townhouses | ok | ok | ok | gate | — | — | — |
| T12 | Granny flat | ok | ok | ok | gate | — | — | — |
| T13 | Mixed-use shop-top | ok | ok | ok | gate | — | — | — |
| T14 | Retail or café fit-out | ok | ok | ok | gate | — | — | — |
| T15 | Childcare centre | — | — | — | — | — | — | — |
| T16 | Office base-building refurbishment | — | — | — | — | — | — | — |
| T17 | Cladding remediation | ok | ok | ok | gate | — | — | — |
| T18 | Warehouse extension or mezzanine | — | — | — | — | — | — | — |
| T19 | Lift and access upgrade | — | — | — | — | — | — | — |
| T20 | School classroom block | — | — | — | — | — | — | — |

## Log

- 2026-10-28: Project types locked (PROJECT_TYPES.md). Plan written
  (`docs/plans/2026-10-28-synthetic-test-corpus.md`). Validator built and
  tested.
- 2026-10-28: The owner answered four questions. Live Jev calls are
  approved. The harness may press profile Update. Near-duplicates are out
  of the pilot. Commits go to `main` and are pushed periodically, staging
  only `testing/synthetic/` and the plan.
- 2026-10-28: Knowledge index done: KNOWLEDGE_INDEX.md plus 10 tables,
  with 15 claims checked (12 by running the real Go). Gaps it found:
  - Remediation gets nothing built in: no scope, consultants, skeleton,
    fragments or budget split. Refurb starts with no work items.
  - 74 of 137 leaf systems have no program fragment.
  - The contamination and flood package rules can never fire.
  - 8 suggested consultants have no RFP template.
- 2026-10-28: Pilot map authors launched for T01, T05 and T09.
- 2026-10-29: All three pilot authors stopped at the session usage limit and
  were resumed. T01 map done: 15 triggers, 11 packages, 32 cost lines, 76
  activities, 10 RFPs, validator ok. The validator now accepts a bare YAML
  yes/no. T01 review launched.
- 2026-10-29: T09 map done: 14 triggers, 3 packages, 19 cost lines, 29
  activities, 2 RFPs, 21 gaps, validator ok. The built-in templates produce
  almost nothing for this job, and the author found 3 keyword traps
  ("flood test", "podium over the basement", "bathroom below"). T09 review
  launched.
- 2026-10-29: T05 map done: 14 triggers, 9 packages, 34 cost lines, 60
  activities, 8 RFPs, 13 gaps, validator ok. Altered or repaired items on an
  extend job get no program activity. The reviewer is asked to settle
  whether Civil belongs in the base scenario. T05 review launched.
- 2026-10-29: T09 review came back FIX, 20 findings, sound at the root.
  - The bathroom keyword trap fires in code with no Jev call. It adds
    hydraulic and exhaust systems, which bring in unwanted consultants.
  - "Bedroom", "kitchen", "common wall", "stairs" and "masonry" are similar
    traps.
  - The DBP Act compliance declaration is missing from the program.
  - The structural RFP wording assumes a construction certificate the exempt
    path doesn't have.
  The author is applying the round 1 fixes.
- 2026-10-29: T01 review came back FIX, 26 findings.
  - Many RFPs marked `yes` would include options the map rules out, because
    base flags (public_road, norbe, sustainability_target) are unset and
    some templates always price options (MUSIC, Revit, strata titling).
  - The architect RFP blocks without a budget.
  - Novation on a project home is wrongly taken as correct.
  - The certifier and CDC claims are confirmed. Livable housing and the
    HIA/MBA defects period are still open.
  The author is applying the round 1 fixes.
- 2026-10-29: T09 round 1 fixes are in: 20 applied, 0 disputed, validator
  ok. Counts are now 14 triggers, 3 packages, 20 cost lines, 32
  activities, 2 RFPs, 24 gaps. Round 2 review launched.
- 2026-10-29: T09 round 2 review: PASS. Two minor wording fixes were sent
  to the author.
- 2026-10-29: T05 review came back FIX, 15 findings.
  - Civil moves from the base scenario to a stormwater trigger keyed on
    flag.osd_required and flag.authority_asset.
  - The claim that altered items on an extend job get no program activity
    was overstated: six fragments still fire.
  - CDC is also unavailable in a heritage conservation area.
  The author is applying the round 1 fixes.
- 2026-10-29: T09 is ready for owner review (gate 1). The minor fixes are
  done and the validator is ok.
- 2026-10-29: T01 round 1 fixes are in: 26 applied, 0 disputed, validator
  ok. Counts are now 17 triggers, 11 packages, 34 cost lines, 81
  activities, 10 RFPs, 27 gaps. The author added keys that are not in the
  schema: not_expected_cost_plan, expected_derived, scenario_inputs, and
  rfps[].expect and app_differs. Round 2 review launched; it will also judge
  whether to add these keys to the schema before the other 17 types are
  mapped.
- 2026-10-29: T05 round 1 fixes are in: 15 applied, validator ok. Counts are
  now 19 triggers, 8 packages, 34 cost lines, 73 activities, 7 RFPs, 15
  gaps, 9 variants. significant-tree was restored as a variant because the
  owner named it and the variant count is only guidance. Round 2 review
  launched.
- 2026-10-29: T05 round 2 review was FIX (minor), 3 items: the
  flood-affected built_in marks, a duplicate timber work item in
  termite-damage-found, and the cost line for existing solar PV. All three
  applied, validator ok. T05 is ready for owner review (gate 1).
- 2026-10-29: T01 round 2 review was FIX (minor).
  - Six base flags and determinants were unset.
  - The architect `bidding` option is missing from app_differs.
  - Finding 6's dropped packages are in prose only.
  The reviewer recommends adding all the new keys to the schema, plus a
  structured app_differs and triggers[].removes. The author is applying
  these, then the schema is updated before the other 17 types are mapped.
- 2026-10-29: T01 round 2 fixes are in, validator ok, ready for owner
  review (gate 1).
- 2026-10-29: Schema extended with not_expected_procurement,
  not_expected_cost_plan, expected_derived, scenario_inputs, rfps[].expect,
  rfps[].app_differs and triggers[].removes. The validator checks them,
  rejects unknown top-level keys, and has 7 tests. Brief 01 is updated, with
  T01 as the reference example. T05 and T09 authors are backfilling the new
  keys.
- 2026-10-29: T05 and T09 have the new keys, validator ok. All three pilot
  maps (T01, T05, T09) are at owner gate 1.
- 2026-10-30: Owner gate 1 passed. The depth and format are approved for
  all 20 types. The owner's domain answers are in
  `reference/OWNER_RULINGS.md` (R1 to R8), and briefs 01 and 02 now point
  to it. The HBCF threshold ($20,000) and the BASIX threshold for
  alterations ($50,000) were checked against NSW sources. R1 returns civil
  to the T05 base. An agent is applying the rulings to T01, T05 and T09.
  Task 3 batch 1 (T10, T11, T12, T02, T13, T17) and Task 4 launched.
- 2026-10-30: Owner rulings applied to the pilots, validator ok. T01 now
  has 83 activities. T05 has civil in the base: 20 triggers, 9 packages,
  35 cost lines, 79 activities, 8 RFPs. The new `internal-only-alteration`
  variant takes civil out. T09 counts are unchanged. The T01 civil RFP's
  for_construction stage is being settled.
- 2026-10-30: T01 civil for_construction is settled: it stays included,
  with an app_differs entry because the app prices it as an option. All six
  batch 1 authors and the Task 4 agent hit the session limit and were
  resumed.
- 2026-10-30: Task 4 done: score_shapes.py and SCORING.md, 31 tests pass.
  Provisional: the harness may press cost-plan prepare, program prepare and
  RFP create in its own test org. Optional `aliases` and `{text, key}`
  phrases are being added to the schema. T01 architecture expect blocking
  is now [], because the scenario states the budget.
- 2026-10-30: T10 (15 triggers, 11 packages, 37 cost lines, 97 activities,
  10 RFPs), T11 (15, 11, 40, 92, 10), T12 (15, 8, 28, 71, 7) and T17 (15,
  10, 30, 41, 9) maps done, validator ok, reviews launched. Common gaps:
  - demolition is invisible;
  - CDC breaks the certifier RFP;
  - subdivision has nothing;
  - remediation gets no team and no fire-safety-order model.
- 2026-10-30: T13 map done (13 triggers, 25 packages, 55 cost lines, 97 activities, 23 RFPs), validator ok, review launched.
- 2026-10-30: Optional `aliases` and `{text, key}` phrases are in the validator, scorer and brief 01. 34 tests pass.
- 2026-10-30: T02 map done (15 triggers, 23 packages, 58 cost lines, 99 activities, 22 RFPs), validator ok, review launched. Batch 1 reviews resumed after the session limit.
- 2026-10-30: T10 review: FIX, 13 findings.
  - The pre-start inspection must come before the CDC determination (R7).
  - The reinforcement hold point may already fire through
    fr.slab-on-ground-1a. If so, T01 has the same wrong `partial`.
  The author is applying the fixes and checking with one Go run.
  The CDC plus traditional base is accepted.
- 2026-10-30: T17 review: FIX, 15 findings; 56 of 57 built-in claims
  check out.
  - On a CDC, the BCA and certifier RFPs both block; there is a new gap.
  - The Project Remediate trigger should remove the PM package.
  The base is DA, owner-funded and full removal. The author is applying the
  fixes.
- 2026-10-30: T12 review: FIX, 11 findings.
  - Secondary-dwelling CDC standards are in Housing SEPP 2021 Schedule 1.
  - A Go run confirms the reinforcement hold point fires through
    fr.slab-on-ground-1a. T01, T05 and T10 are being corrected.
  - The garage-conversion trigger is to be split.
  The author is applying the fixes. The validator now finds the repo root
  itself, so it runs from any directory.
- 2026-10-30: T10 round 1 fixes are in: 13 applied, validator ok. Counts
  are now 17 triggers, 11 packages, 37 cost lines, 104 activities, 10
  RFPs, 27 gaps. A Go run confirmed the reinforcement hold point fires for
  CDC, DA and extend, so T01, T05 and T10 now mark it yes. T10 round 2
  review launched.
- 2026-10-30: T17 round 1 fixes are in: 15 applied, 0 disputed, 4 new
  triggers, validator ok. Counts are now 19 triggers, 10 packages, 33 cost
  lines, 44 activities, 9 RFPs, 27 gaps. Round 2 review launched.
- 2026-10-30: T11 review: FIX, 11 findings.
  - The base now demolishes an existing house; a vacant lot becomes a
    variant.
  - The separating-wall rule is corrected (Housing Provisions 9.3.1).
  - The Housing and Productivity Contribution and the Sydney Water
    contributions apply.
  The author is applying the fixes.
- 2026-10-30: T13 review: FIX, 19 findings. The base and size are
  confirmed: D&C with novation; 25 packages and 23 RFPs are right-sized.
  - The fire control centre threshold is 50 m, not 25 m.
  - App bug: two hold points order work wrongly (the ground slab after the
    roof slab, and ceilings after sprinkler fit-off). Log it in FINDINGS
    at the pilot run.
  The author is applying the fixes.
- 2026-10-30: T12 round 1 fixes are in: 11 applied, 0 disputed, validator
  ok. Counts are now 19 triggers (9 variants), 8 packages, 28 cost lines,
  73 activities, 7 RFPs. Round 2 review launched.
- 2026-10-30: T10 round 2 review: PASS. Two optional items (the set-out
  survey row and the SafeWork notice link) are being applied.
- 2026-10-30: T17 round 2 review: PASS. Three minor variant items (R2-1
  to R2-3) are being applied.
- 2026-10-30: T10 minor items applied, validator ok. T10 is ready for owner review.
- 2026-10-30: T17 minor items applied, validator ok. Compliance with an
  order and no consent means no CC, principal certifier or OC, but a final
  fire safety certificate is still needed (verified: false; the owner
  should confirm). T17 is ready for owner review.
- 2026-10-30: T12 round 2 review: FIX (minor), 3 items: a removes label,
  garage-conversion RFPs and geotech, and wording. The author is applying
  them.
- 2026-10-30: T12 round 2 items applied, validator ok. T12 is ready for owner review.
- 2026-10-30: T02 review: FIX, 16 findings. 23 packages and 22 RFPs are
  right-sized. E1D6 is confirmed.
  - The program needs staged CCs, and demolition under a DA needs no CC.
  - Six keyword traps in ordinary apartment wording set wrong profile
    values. They were passed to T13 too.
  The author is applying the fixes.
- 2026-10-30: T11 round 1 fixes are in: 11 applied, 0 disputed. Counts are
  now 16 triggers, 12 packages, 44 cost lines, 107 activities, 11 RFPs.
  Round 2 review launched.
- 2026-10-30: T13 round 1 fixes are in: 19 applied. Counts are now 18
  triggers, 25 packages, 55 cost lines, 113 activities, 23 RFPs. Round 2
  review launched.
- 2026-10-30: New `reference/CONVENTIONS.md`, and briefs 01 and 02 point to
  it:
  - C1 (provisional): on D&C, services engineers' novation is option; the
    architect and structural engineer are yes.
  - C2: consultant.basix for BASIX, consultant.esd for Section J.
  - C3: the scorer pairs RFPs by discipline first; that fix is in progress.
- 2026-10-30: The scorer now pairs RFPs by discipline first (C3). 35 tests pass.
- 2026-10-30: T11 round 2 review: FIX (minor), 3 items: the vacant-lot
  evidence and removes, the no-subdivision removes, and a stale gap. The
  author is applying them.
- 2026-10-30: T02 round 1 fixes are in: 15 applied, 1 disputed (F16,
  water_asset is in the house module only). Staged CCs, keyword traps and
  C1 are applied. Counts are now 16 triggers, 23 packages, 59 cost lines,
  103 activities, 22 RFPs. Round 2 review launched.
- 2026-10-30: T13 round 2 review: FIX (minor), 5 items: the
  retained_services row, the earthworks link, the C2 alias, leftover
  lines, and the rise-three variant's sprinklers. The author is applying
  them. The scorer and validator are learning discipline `aliases`.
- 2026-10-30: T11 round 2 items applied, validator ok. T11 is ready for owner review.
- 2026-10-30: T13 round 2 items applied, validator ok. retained_services is expected absent (every existing service is abolished), with an app_differs entry. T13 is ready for owner review.
- 2026-10-30: T02 round 2 review: FIX (minor). The F16 dispute is upheld.
  Items: {text, key} RFP items on the traditional trigger, the earthworks
  link, panel wording, and an ssd-affordable-housing trigger. The author is
  applying them.
- 2026-10-30: T02 round 2 items applied, validator ok, 17 triggers. T02 is ready for owner review. All six batch 1 maps are at owner review.
- 2026-10-30: Batch 1, the conventions and the scorer committed and pushed
  (7b59d9b). Batch 2 (T03, T04, T06, T07, T08, T14) launched.
- 2026-10-30: Task 5 started with T01 only (owner asked to run one type end to end before mapping more). Brief 03 and `tools/validate_expected.py` written; 23 tests pass. Cost plan, program and RFP keys point at the shape, because `score_shapes.py` already derives them. The checker enforces the metamorphic rule: a variant's profile differs from the base only on its trigger's keys, and every changed document is declared. T01 author launched for base, cdc-pathway and trees-affected.
- 2026-10-30: Owner is away and asked to go straight on to rendering once the T01 scenarios pass review (owner gate 2 waived for T01; the owner reviews the pilot run instead). Written: brief 03r (scenario review), brief 04 (render), brief 04r (render review), `tools/lint_corpus.py` with 10 tests.
- 2026-10-30: T01 scenarios written, checker ok: ST01-base (17 documents, 4 waves), ST01-cdc-pathway (16), ST01-trees-affected (19). Missed by design: the target construction start is only in an owner email. The author keyed `flag.arborist_offsite_trees` and `det.site_class`, which the shape's base profile lacks. Review launched (brief 03r).
- 2026-10-30: T01 scenario review: FIX, 4 findings. The base must deny off-site trees in words the profile reading catches ("No neighbouring or street trees are affected by the works"). d02's number and date sit in a table, and the notes must say the app will miss them. The reviewer agrees the shape's base profile should gain `det.site_class: M` and `flag.arborist_offsite_trees: "no"`. That shape change is left for the owner. The author is applying the fixes.
- 2026-10-30: T01 scenario fixes applied (4 of 4, none disputed), checker ok on all three, and the orchestrator confirmed them. Scenarios done. ST01-base render launched (brief 04); the variants are rendered from it afterwards.
- 2026-10-30: ST01-base rendered: 17 documents, lint ok. The orchestrator removed an unrealistic "(stated)" from the d15 invoice total, in the corpus and all three manifests. Render review launched (brief 04r).
- 2026-10-30: T08 map done (14 triggers, 4 packages, 23 cost lines, 33 activities, 3 RFPs, 23 gaps), validator ok, review launched. The app has no asbestos knowledge and only 2 program activities for this job.
- 2026-10-30: T14 map done (13 triggers, 5 packages, 34 cost lines, 79 activities, 3 RFPs), validator ok, review launched. The app suggests 10 consultants, 7 of them wrong, and the grease arrestor and exhaust never appear.
- 2026-10-30: T03 map done (13 triggers, 13 packages, 35 cost lines, 69 activities, 11 RFPs), validator ok, review launched. A refurb starts with no scope, the landlord is missing everywhere, and CDC blocks the certifier RFP.
- 2026-10-30: ST01-base render review: FIX, 9 findings. The writer applied the corpus-only fixes, including moving the neighbour's jacaranda about 12 m clear of the boundary. The orchestrator applied the four that needed new manifest wording, in the base corpus and all three manifests: no "(single owner)", the Connecting with Country line without "heritage", "No public road works are required", and climate zone 5 without "zone 56". Checker and lint ok. ST01-base corpus is final. Variant renders launched.
- 2026-10-30: T07 map done (15 triggers, 6 packages, 28 cost lines, 44 activities, 5 RFPs), validator ok, review launched. There are no plant fragments, the consultants are wrong, and an exempt job still gets CC fees and an OC row.
- 2026-10-30: T06 map done (14 triggers, 9 packages, 36 cost lines, 82 activities, 7 RFPs), validator ok, review launched. There is no fire-water fragment, NSW fire safety law is not encoded, and the sprinkler test is ordered after ceilings (third sighting).
- 2026-10-30: T04 map done (15 triggers, 20 packages, 49 cost lines, 94 activities, 19 RFPs), validator ok, review launched. All six batch 2 maps are written and in review.
- 2026-10-30: Variants rendered, lint ok. ST01-cdc-pathway (16 documents): the first pass left "DA" in 8 documents, some of them read (fee proposals, survey), which would have made the expected `cdc` a conflict. Rewritten, with each one now in changed_from_base. Brief 03 now requires every passing mention of the old condition to change. ST01-trees-affected (19 documents): the survey puts the jacaranda 2 m from the boundary, the arborist's figures are realistic, and the SEE gains a trees paragraph. Both variants are in render review.
- 2026-10-30: T08 review: FIX, 16 findings.
  - The map now cites the WHS Regulation 2025.
  - Add a building consultant as the author of the roof specification.
  - Long service levy: 0.25% on work of $250,000 or more (added to
    CONVENTIONS C4).
  The author is applying the fixes.
- 2026-10-30: ST01-cdc-pathway render review: FIX, 4 findings (DA hints: cl 4.6 wording, "complying development application"). All applied, checks ok. The CDC corpus is final.
- 2026-10-30: T14 review: FIX, 17 findings.
  - The certifier RFP stages are "ask", not "option".
  - R3 (the DBP Act) applies on the riser variant with apartments above.
    This is an inference; flag it for the owner.
  - Add the Tap in approval and a kitchen designer.
  The base (DA, trades design the exhaust and plumbing) is confirmed. The
  author is applying the fixes.
- 2026-10-30: ST01-trees-affected render review: FIX, 6 findings, all applied. The site plan Rev B now shows the jacaranda, TPZ fencing and piers. Root cutting is worded without "pruning" so `flag.arborist_pruning` stays unset. Open owner questions: (1) should trees-affected also set arborist_pruning; (2) should the shape's base profile add `det.site_class: M` and `flag.arborist_offsite_trees: "no"`. All three T01 corpora are final: checker and lint ok, 52 documents. Next is Task 7, the harness.
- 2026-10-30: T08 round 1 fixes are in: 16 applied, 0 disputed, 2 new
  triggers. Counts are now 16 triggers, 5 packages, 25 cost lines, 35
  activities, 4 RFPs. Round 2 review launched.
- 2026-10-30: T08 round 2 review: FIX (minor), 3 items. The overlay-over-ac
  variant is incoherent, and two labels need fixing. The author is applying
  them.
- 2026-10-30: T08 round 2 items applied, validator ok. T08 is ready for
  owner review. Follow-up: removes.rfps can only drop a whole RFP. The
  scorer is learning partial removes, after which T08's overlay-over-ac
  should use one.
- 2026-10-30: T14 round 1 fixes are in: 17 applied, 0 disputed, 4 new
  triggers. Counts are now 17 triggers, 6 packages, 35 cost lines, 82
  activities, 3 RFPs. Round 2 review launched.
- 2026-10-30: T03 review: FIX, 15 findings. A Go run confirms app program
  order bugs on a refurb fit-out:
  - services rough-in lands after ceilings, because fragments anchored to
    the missing strip-out get chained;
  - fire stopping starts on day 0 and holds every interiors row;
  - the sprinkler-to-ceilings link is dropped as a loop (as in T13).
  Also, the CDC must follow landlord approval. The author is applying the
  fixes. To do before the next commit: compile the Go-confirmed app
  defects from the REVIEW files into FINDINGS.md.
- 2026-10-30: T14 round 2 review: PASS. A duplicate gap is being merged.
  Scorer limit: blocking differences were only text. A `blocking.` key for
  app_differs is being added. The owner still has to confirm R3 on the
  riser variant.
- 2026-10-30: Partial RFP removes are supported (73 tests pass). T08 is
  switching to them.
- 2026-10-30: T14 duplicate gap merged, validator ok. T14 is ready for
  owner review. Follow-up: its blocking differences stay as text.blocking
  until the `blocking.` key lands.
- 2026-10-30: T06 review: FIX, 18 findings.
  - The Fire Safety Code (Part 8) covers only the pump, tank and fire-main
    works; the sprinkler changes go in the fit-out CDC.
  - Pump room ventilation belongs under mechanical.local-exhaust.
  - AS 2941 is the 2013 edition.
  The author is applying the fixes.
- 2026-10-30: T07 review: FIX, 16 findings, the exempt base confirmed.
  - Three app_differs are corrected by an rfp.Resolve run.
  - Variant false positives need `removes`.
  - The 25 m threshold comes from NCC E2.
  The author is applying the fixes.
- 2026-10-30: T08 overlay-over-ac now uses partial RFP removes, validator ok.
- 2026-10-30: `blocking.<id>` app_differs key added (75 tests pass). T14 and T03 are converting to it.
- 2026-10-30: T14 blocking differences are now 15 blocking.<id> entries, validator ok.
- 2026-10-30: T04 review: FIX, 14 findings. A Go run confirms no 7b hold
  point fires, and that 7b plus 5 on one part sets flag.mixed_use. About 45
  RFP app_differs are missing. C3D4 is verified. The author is applying the
  fixes.
- 2026-10-30: Fire control centre threshold checked in NCC 2022 E1D15:
  above 25 m (S19C7's 50 m is the separate room). The T13 round 1 change to
  50 m was wrong and is being reverted. CONVENTIONS C4 and C5 (staged CCs)
  were added, and T02 and T13 are aligning with them.
- 2026-10-30: T03 round 1 fixes are in: 15 applied, 0 disputed. Counts are
  now 14 triggers, 13 packages, 38 cost lines, 71 activities, 11 RFPs.
  Round 2 review launched.
- 2026-10-31: Owner rulings R9 (root zones and foundations; root cutting is not pruning) and R10 (T01 base: site class M, no neighbouring trees affected) recorded. The T01 shape gains both base values and a structural RFP item under trees-affected. Harness built: `serve.sh` runs a separate SiteWise on its own database, so test projects stay out of the owner's dev organisation. First live pilot run of all three T01 scenarios: see `runs/20261031-pilot/REPORT.md` and `FINDINGS.md` (F1 to F7). Raw Jev exchanges are not recorded, because the server has no switch for it; stored decisions are saved instead. Next: owner gate 3, triage of the RFP differences, and the invoice-flag key fix.
- 2026-11-02: Re-ran T01 after the owner's fixes (`runs/20261102-fixes-1-4-6`): F6 is fixed, F1 is improved (3 of 12 waves), F4 is still open. F4 traced to `HeaderRouted`: the route sentence is never asked. F8 found: the invoice supplier is never read. The scorer now expects the state-driven invoice flags. RFP triage: the scorer treats inactive positions as absent and reports user-input blocks apart; T01 shape phrases keyed. Base RFP new failures 105 to 34, all knock-ons. F9 to F12 are RFP defects (certifier CDC, critical-stage inspections, mixed_use for 1a+10a, SRZ footing row). Next: T05 and T09 scenarios.
- 2026-11-04: Owner away; asked to commit everything, merge to main and carry on. Main's shared working tree committed (4e4416f); auth/accounts and procurement/appointment-fee merged (c56c9de). Not pushed. T05 scenarios launched (base, heritage-conservation-area, first-floor-addition) and T09 (base, podium-over-basement, hazardous-materials). As for T01, owner gate 2 is waived: review, render and run follow without stopping.
- 2026-11-04: T05 scenarios written, checker ok: ST05-base (19 documents, 4 waves; traps include a CDC-permitted line on the planning certificate, "sewer main in the street", "Dulux Heritage", "flood test the shower", and an invoice charging 15% GST), ST05-heritage-conservation-area (22; the SEE moves to the town planner; heritage and planner fee proposals and a heritage impact statement added), ST05-first-floor-addition (19, 10 changed). Missed by design: the target construction start (email only), and the d02 and d07 table title blocks. Possible shape issues raised: hazardous-materials-found lists existing_building_year as its profile key, and first-floor-addition may need added_gfa_sqm. Review launched (brief 03r).
- 2026-11-04: T09 scenarios written, checker ok: ST09-base (16 documents, 4 waves, RW-101 at Rev A and B; fee proposals from two competing remedial building consultants and the structural engineer, because the brief's new-build consultant defaults do not fit remediation), ST09-podium-over-basement (18; landscape and hydraulic proposals added), ST09-hazardous-materials (18; hazmat proposal and survey added, built 1986). Traps include "bathroom below", "fire stairs", "flood test", remediation without contamination, and an invoice ABN typo. Missed by design: the target start (email only), the d02, d06 and d15 table title blocks, and probably proj.regulated_design. Author's questions (shape phrasings that match no app trigger, the podium trigger's profile keys, "(fictional)" in proj.client) went to the reviewer. Review launched.
- 2026-11-04: T05 scenario review: FIX, 7 findings. The contract form "HIA NSW Lump Sum Contract" matches no fact.contract_form trigger (ST01 missed it the same way), and proj.regulated_design has no reading, so it must be read_by_design false. Also: a tree contradiction between d03 and d13, three added-area candidates in the first-floor d13, a stale summary, an ambiguous email date, and two position-dependent headings. Shape: hazardous-materials-found gets profile: []; first-floor-addition gains added_gfa_sqm and hdr.scale.gfa_sqm. The author is applying the fixes, including those two shape edits.
- 2026-11-04: T09 scenario review: FIX, 9 findings. "42 residential units" matches no units trigger (use "42 apartments"; the shape's phrasing is wrong too). "Client: The Owners - Strata Plan No. 912345" is cut at " - "; the documents keep the legal name and the key keeps the full name, so a cut is logged as an app defect. proj.regulated_design and proj.lot are unreadable. The podium trigger may change det.ncc_class and proj.budget_stated; "(fictional)" comes out of the shape's client. The remediation consultant set is confirmed. The author is applying the fixes and the shape edits.
- 2026-11-04: T05 fixes in: 6 applied, 1 partly disputed (proj.regulated_design stays read_by_design true, because the checker derives it from the d05 fee proposal's read kind; each notes.md declares it missed by design). Shape: hazardous-materials-found has profile: [] and no duplicate adds; first-floor-addition lists hdr.scale.added_gfa_sqm and hdr.scale.gfa_sqm. Validator and checker ok. Scenarios done; ST05-base render launched.
- 2026-11-04: T09 fixes in: 9 applied, none disputed. "42 apartments" is in d01, d03 and d08. proj.regulated_design and proj.lot are read_by_design false with sources [] (the stating documents go under stated_in). The podium variant's budget is $1,960,000 and its class is 2 + 7a. The hazmat negation is now "constructed in 2008". Shape edits are applied. Checker, validator and tool tests ok (78 and 11). Scenarios done; ST09-base render launched.
- 2026-11-04: ST05-base rendered: 19 documents, lint ok. Departures from the manifest: d09 says "pre-demolition survey" (dropping "refurbishment" so the work type stays extend); d17 and d18 gain Title lines; the d16 SEE has no rear-setback row. Render review launched (brief 04r).
- 2026-11-04: ST09-base rendered: 16 documents. The lint failed twice because two procurement-route `says` began with a placement note ("Under '## …': ") that no document prints. The orchestrator removed the prefix in all three ST09 manifests; lint and checker ok. Render review launched.
- 2026-11-04: ST05-base render review: FIX, 7 findings; every keyed value matches expected/. The d16 SEE's reason for a DA (a Housing Code rear-setback failure) is not credible on a 45.7 m lot, so it now says the owners chose a DA. The d01 vacant-during-works line is ambiguous. Departures 1 and 2 accepted. The writer is applying the fixes, keeping the three ST05 manifests consistent.
- 2026-11-04: ST05-base render fixes in: 7 applied, part of finding 4 disputed ("no other buildings on the lot" matches no multiple-buildings pattern; it now says "We are not planning multiple buildings: just the one house"). The matching manifest wording is updated in all three ST05 scenarios, and the T05 shape's move-out phrasing follows d01 ("for the whole of the works"). Lint and checker ok. The ST05-base corpus is final. Both T05 variant renders launched.
- 2026-11-04: ST09-base render review: FIX, 7 findings. Every planned trap is placed and no unplanned trap word appears. The d13 tender conditions say "a rise of more than three storeys", a second storey number beside the keyed 6. fees.json leaves out Halyard's $220/h ex GST rate. Smaller: a duplicate balcony in the leak log, a brief deadline before the proposals, an overflow clause, the street name, and key-shaped wording. The writer is applying the fixes.
- 2026-11-04: ST09-base render fixes in: 7 applied. d13 now says the rise is 6 storeys, and Halyard's $220/h ex GST rate and new address are in all three fees.json. Halyard moved to 7 Gray Street to avoid sharing a street with Northgate. Lint and checker ok. The ST09-base corpus is final. ST05-first-floor-addition rendered (19 documents, 10 changed, 140 + 55 = 195 m2 throughout) and ST05-heritage-conservation-area rendered (22; d20 to d22 added); both lint ok, both in render review. The ST09 variant renders are running. Live run of ST05-base and ST09-base started on :8091 into `runs/20261104-t05-t09/`.
- 2026-11-04: All four variant corpora rendered, render-reviewed (FIX 3, 2, 5 and 6 findings) and fixed; lint and checker ok on all six. Live run of all six T05 and T09 scenarios: `runs/20261104-t05-t09/REPORT.md`. The changed key landed in three of the four variants; the heritage status was blank on 14 passages to 1 (F15). New app findings: F13 (an invalid invoice ABN can never be flagged: BadABNs is never read), F14 (stated fee total missed for a second-column total row and for inc-only totals; 0 of 12 T09 proposals), F15 (one dissenting passage blanks a choice, so with F4 every T05 program opens on sk.generic), F16 (rise in storeys read only from a BCA report). F1, F2 (now the site address too), F3, F4, F7 and F8 repeat. Next: owner gate 3 on all three pilots.
- 2026-11-04: Batch 2 picked up where it stopped on 30 October. T03's round 1 fixes were in, but no round 2 review had run; it is now launched. T04, T06 and T07 had FIX reviews with no author response; round 1 fixes are launched for all three.
- 2026-11-04: T07 round 1 fixes in: 16 applied, 0 disputed (most were already in the committed files; the last two are added). An rfp.Resolve run confirms the app_differs. Finding 10 needs a scorer hook, so it stays a gap. Counts are now 18 triggers, 6 packages, 28 cost lines, 45 activities, 5 RFPs, 28 gaps. Validator ok. Round 2 review launched.
- 2026-11-04: T07 round 2 review: FIX (minor), 4 findings: the support frames are installed before the old chiller leaves its plinths (split the crane lifts), the WHS Regulation 2025, the levy is verified per C4, and capacity-increase expects supply augmentation outright. The author is applying them.
- 2026-11-04: T03 round 2 review: FIX (minor), 9 findings. The base still says "core drilling" (evidence for the slab-core-drilling trigger). The exempt variant keeps the commencement inspection, certifier notice, final fire safety certificate and CDC contributions. heritage-item counts contributions twice. The levy should cite C4. A fresh author is applying them.
- 2026-11-04: T07 round 2 items applied (4 and one optional, 0 disputed): the crane-out and lift-in are split with the frames and hold point between them, the WHS Regulation 2025, the levy per C4 (only whether plant replacement is levied stays unverified), and capacity-increase without the supply lines or the duplicate activity. Counts: 18 triggers, 6 packages, 28 cost lines, 47 activities, 5 RFPs, 28 gaps. Validator ok. T07 is ready for owner review.
- 2026-11-04: T06 round 1 fixes in: 18 applied, 0 disputed. Most were already in the committed shape; the last two are added, and DOMAIN.md §1–12 is rewritten. A throwaway program.Generate run confirmed findings 2, 7 and 15. Counts are now 18 triggers, 9 packages, 36 cost lines, 85 activities, 7 RFPs, 26 gaps. Validator ok. Round 2 review launched.
- 2026-11-04: T03 round 2 items applied (9, 0 disputed): "core drilling" is out of the base, the levy cites C4, and the exempt variant removes the certifier appointment, commencement inspection, final fire safety certificate and CDC contributions. heritage-item counts contributions once. Counts unchanged. Validator ok. T03 is ready for owner review.
- 2026-11-04: T04 round 1 fixes in: 14 applied, F1 partly disputed (both CCs now sit in construction documentation after award, but per C5 the app's CC row stays matched to the early-works CC, and the building CC is built_in no). A read-only Go run confirms F5 to F8, and all 17 templated RFPs now match the app. Counts are now 15 triggers, 20 packages, 49 cost lines, 95 activities, 19 RFPs, 26 gaps. Validator ok. The author reports that since 3 November rfp.Resolve's scope-by-stage layout reads every option and ask stage as include, which would affect every shape's RFP expectations. Round 2 review launched; it is also asked to confirm or refute this from the code.
