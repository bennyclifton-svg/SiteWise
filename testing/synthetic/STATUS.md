# Status

The orchestrator updates this file after every task. Anyone resuming the
work starts here, then reads `README.md`.

Cell values: `—` not started, `run` in progress, `ok` done, `fix` needs
rework (see that type's REVIEW.md), `gate` waiting for the owner.

## Shared tasks

| Task | State | Output | Note |
|---|---|---|---|
| 0 Scaffolding and shape validator | ok | `tools/validate_shape.py`, briefs, this file | 7 unit tests pass |
| 1 Knowledge index | ok | `reference/KNOWLEDGE_INDEX.md` | brief `briefs/00-knowledge-index.md` |
| 5 Scenario brief and expected JSON schema | — | `briefs/03-scenarios.md`, `tools/validate_expected.py` | after the pilot maps are approved |
| 6 Rendering brief and leak lint | — | `briefs/04-render.md`, `tools/lint_corpus.py` | |
| 7 Harness | — | `harness/` | Python, local API |
| 8 Pilot run | — | `REPORT.md`, `FINDINGS.md` | live Jev approved 28 Oct |

## Project types

| ID | Type | Map | Validate | Review | Owner | Scenarios | Docs | Run |
|---|---|---|---|---|---|---|---|---|
| T01 | New detached house (pilot) | ok | ok | ok | gate | — | — | — |
| T02 | Apartments | — | — | — | — | — | — | — |
| T03 | Office fit-out | — | — | — | — | — | — | — |
| T04 | Warehouse | — | — | — | — | — | — | — |
| T05 | House alterations and additions (pilot) | ok | ok | ok | gate | — | — | — |
| T06 | Sprinkler pump and tank upgrade | — | — | — | — | — | — | — |
| T07 | Mechanical plant upgrade | — | — | — | — | — | — | — |
| T08 | Roof remediation | — | — | — | — | — | — | — |
| T09 | Waterproofing remediation (pilot) | ok | ok | ok | gate | — | — | — |
| T10 | Knock-down rebuild | — | — | — | — | — | — | — |
| T11 | Dual occupancy or townhouses | — | — | — | — | — | — | — |
| T12 | Granny flat | — | — | — | — | — | — | — |
| T13 | Mixed-use shop-top | — | — | — | — | — | — | — |
| T14 | Retail or café fit-out | — | — | — | — | — | — | — |
| T15 | Childcare centre | — | — | — | — | — | — | — |
| T16 | Office base-building refurbishment | — | — | — | — | — | — | — |
| T17 | Cladding remediation | — | — | — | — | — | — | — |
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
