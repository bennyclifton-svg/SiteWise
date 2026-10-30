# Status

The orchestrator updates this file after every task. Anyone resuming the
work starts here, then reads `README.md`.

Cell values: `—` not started, `run` in progress, `ok` done, `fix` needs
rework (see that type's REVIEW.md), `gate` waiting for the owner.

## Shared tasks

| Task | State | Output | Note |
|---|---|---|---|
| 0 Scaffolding and shape validator | ok | `tools/validate_shape.py`, briefs, this file | 35 tests pass (validator + scorer) |
| 1 Knowledge index | ok | `reference/KNOWLEDGE_INDEX.md` | brief `briefs/00-knowledge-index.md` |
| 4 Shape scoring rules | ok | `SCORING.md`, `tools/score_shapes.py` | TDD from the T01 shape |
| 5 Scenario brief and expected JSON schema | — | `briefs/03-scenarios.md`, `tools/validate_expected.py` | after the pilot maps are approved |
| 6 Rendering brief and leak lint | — | `briefs/04-render.md`, `tools/lint_corpus.py` | |
| 7 Harness | — | `harness/` | Python, local API |
| 8 Pilot run | — | `REPORT.md`, `FINDINGS.md` | live Jev approved 28 Oct |

## Project types

| ID | Type | Map | Validate | Review | Owner | Scenarios | Docs | Run |
|---|---|---|---|---|---|---|---|---|
| T01 | New detached house (pilot) | ok | ok | ok | ok | — | — | — |
| T02 | Apartments | ok | ok | ok | gate | — | — | — |
| T03 | Office fit-out | — | — | — | — | — | — | — |
| T04 | Warehouse | — | — | — | — | — | — | — |
| T05 | House alterations and additions (pilot) | ok | ok | ok | ok | — | — | — |
| T06 | Sprinkler pump and tank upgrade | — | — | — | — | — | — | — |
| T07 | Mechanical plant upgrade | — | — | — | — | — | — | — |
| T08 | Roof remediation | — | — | — | — | — | — | — |
| T09 | Waterproofing remediation (pilot) | ok | ok | ok | ok | — | — | — |
| T10 | Knock-down rebuild | ok | ok | ok | gate | — | — | — |
| T11 | Dual occupancy or townhouses | ok | ok | ok | gate | — | — | — |
| T12 | Granny flat | ok | ok | ok | gate | — | — | — |
| T13 | Mixed-use shop-top | ok | ok | ok | gate | — | — | — |
| T14 | Retail or café fit-out | — | — | — | — | — | — | — |
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
