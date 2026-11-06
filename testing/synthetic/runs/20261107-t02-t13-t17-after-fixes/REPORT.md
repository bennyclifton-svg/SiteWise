# Run 20261107-t02-t13-t17-after-fixes: T02, T13 and T17 after the F31–F42 fixes

Re-run of the eight live T02, T13 and T17 scenarios after the owner's
answers (rulings R11 to R14) and the fixes in
`docs/plans/2026-11-07-synthetic-findings-f31-f42.md`. ST17-cdc-pathway is
retired (R11) and was not re-run.

- **App:** local main `d314652` plus another session's uncommitted Profile
  display edits (as in the first run).
- **How it ran:** `serve.sh fresh` (empty database, so no firm carried over
  from earlier runs), live Jev, one scenario at a time. No HTTP errors;
  every RFP draft returned 201.
- **Scores:** final shapes and keys (commit `3a03adc` and the shape updates
  that followed the app fixes). The first run is re-scored with the same
  shapes in the "before" columns where the shape changed the count.

## Outcome

- **No wrong profile values in any run** (was 1 in each T02 run).
- **Every fee total, firm name, invoice amount and invoice flag is right**
  in all eight runs (was: T02 structural total lost, T02 civil firm wrong,
  T17 QS total lost, T17 invoice misread).
- **Filing discipline:** 15 of 15 on T02 and T13 (was 13), 17 of 18 on T17
  (was 15).
- **T13 reads its key metrics (R14):** 42 apartments (18 in the
  three-storey variant), 600 m² of shop area and the 1,800 m² site.
- **RFP new failures** fell to 1–2 on T02, 2–6 on T13 and 1 on T17. What is
  left is the known open items: missed "no" flags and
  `proj.regulated_design`.

## Before and after

| Scenario | Profile wrong | Profile correct | RFP new failures | Fee totals | Firms |
|---|---|---|---|---|---|
| ST02-base | 1 → 0 | 66 → 67 | 4 → 1 | 4/5 → 5/5 | 4/5 → 5/5 |
| ST02-groundwater | 1 → 0 | 66 → 67 | 5 → 1 | 4/5 → 5/5 | 4/5 → 5/5 |
| ST02-traditional | 1 → 0 | 66 → 67 | 21 → 2 | 5/5 | 4/5 → 5/5 |
| ST13-base | 0 | 66 → 67 | 28 → 6 | 5/5 | 5/5 |
| ST13-rise | 0 | 66 → 69 | 28 → 2 | 5/5 | 5/5 |
| ST13-contaminated | 0 | 66 → 68 | 29 → 3 | 6/6 | 6/6 |
| ST17-base | 0 | 40 → 42 | 9 → 1 | 6/7 → 7/7 | 7/7 |
| ST17-project-remediate | 0 | 40 → 42 | 9 → 1 | 5/6 → 6/6 | 6/6 |

Program, cost plan and procurement: T02 program 65 → 66 matched; T13 77 →
78; T17 unchanged (no remediation skeleton, a known gap). No fabricated
amounts and no program link violated in any run. The T13 contaminated-land
environmental package is now also a built-in suggestion (F41).

## Each fix

| Fix | Result in the live run |
|---|---|
| F31 invoice "Fee" line | Confirmed: T17 K-26-0412 flags `arithmetic`, no false `gst_rate` |
| F32 / R14 mixed-use scale | Confirmed: units, shop area and site area read on T13 |
| F33 line-wrapped trigger | Fixed in code and tests, but `flag.cl46` is still blank live; it now behaves like the other missed "no" flags (below) |
| F34 low traffic noise | Confirmed: `det.noise_affected_site` no longer wrong on T02 |
| F35 / R12 client majority | Confirmed: the T17 client settles amber |
| F36 basement question | Confirmed: T17 `flag.basement` is no longer a conflict |
| F37 firm by name | Confirmed on a fresh database: every firm and ABN right |
| F38 optional fee | Confirmed: T02 structural total $156,000 |
| F39 QS cost estimate | Confirmed: T17 QS total $45,000 |
| F40 filing discipline | Confirmed: planning and BCA documents file to the right discipline |
| F41 contamination rule | Confirmed in tests; the contaminated variant's package now has a built-in source as well as the proposal |
| F42 reinforcement hold | Confirmed: the hold is before the basement slab pour on T13 |
| R12 majority ratio | Changed from 3x to 2x after the first re-run: at 3x T17's work type (11 documents to 4) stayed a conflict and blocked five RFPs |
| R13 architect brief | Confirmed: brief priced, masterplan lines absent on single buildings |

## Still open

- **Missed "no" flags** (open since the pilot): T13 `flag.cl46`,
  `flag.public_road`, `flag.community` and `flag.architect_lodges` stay
  blank although the documents deny them. This is 5 of T13 base's 6 RFP
  new failures.
- **`proj.regulated_design`** has no reading (1 RFP failure per run).
- **T02:** `quality_level`, `environmental_sensitivity` and
  `avg_unit_size_sqm` are not read; the GFA and `multi_building` conflicts
  are planned.
- **T17:** access constraints are a conflict (urban against restricted
  hours, both reasonable); no remediation skeleton; the "Service authority
  connection fees" cost line is a known false positive.
- **F24, F26, F29, F6** stay open from T10–T12.
- **Survey site area:** the "Site area by survey:" line is still not read
  (the colon cut was not the cause; untraced).
- **Fee-proposal answer key:** the private 42-proposal replay could not run
  (the key is no longer on this machine). The fee fixes rest on unit tests
  and this run.

## Checks

- Go suite: passes apart from TestMeridianAnswerKey (39/46) and
  TestProgramRoutes, which failed before these changes.
- Invoice answer key (private replay): 90 cases, 0 wrong values marked
  green, unchanged.
- Intake bench gate: passed. Filing p50 336 ms, p90 792 ms (budget 1 s /
  2 s); profile edit p50 34 ms (budget 50 ms).
- Synthetic tools: 88 tool and 12 harness tests pass; every shape and all
  27 scenarios validate.
