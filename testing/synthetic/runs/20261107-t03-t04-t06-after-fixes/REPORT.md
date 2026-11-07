# Run 20261107-t03-t04-t06-after-fixes: T03, T04 and T06 after the F43–F50 fixes

Re-run of the nine T03, T04 and T06 scenarios, unchanged, after the fixes
in `docs/plans/2026-11-07-synthetic-findings-f43-f50.md`.

- **App:** local main `bd24ab1` plus another session's uncommitted display
  edits (as in the first run).
- **How it ran:** `serve.sh fresh`, live Jev, one scenario at a time
  (`run.log`). No HTTP errors; every RFP draft returned 201.
- **Scores:** both runs scored with the same current shapes and keys
  (after the owner rulings R15 to R19 and the T03, T04 and T06 shape
  edits). Detail: `FINDINGS.md`, section "Status after the fixes
  (7 November 2026, T03, T04 and T06)", and F51 to F53.

## Outcome

- **Six findings fixed and confirmed:**
  - F45: denial sentences no longer add scope.
  - F46: the T06 client is read, so five RFPs no longer block.
  - F47: every program is dated, so all 175 previously unknown links are
    now scored.
  - F48: the T03 council area is read.
  - F49: fire services drawings file as fire services.
  - F50: no interior designer on a services-only job.
- **F43, mostly fixed:** works filed as existing fall from 18 to 2 (T03),
  7 to 2 (T06) and 2 to 1 (pump-tank-only). The fit-out's partitions,
  ceilings, joinery, floors, lighting and cabling are back in scope and in
  the program. T06's fire water and sprinklers are back. pump-tank-only
  splits $800,000 fire and $400,000 hydraulic (was all on Hydraulic).
  Residual: F51.
- **F44, fixed where the run reaches it:** services rough-in and the
  sprinkler pressure test now come before the ceilings, with no skipped
  link. Fire stopping is not read in any run, so its order rests on the
  unit test.
- **RFP new failures 0 in all nine runs** (T06 was 5 each).
- **No wrong profile values, and no regressions.**
- **Program matched up 6 to 10 rows per T03 and T06 run.**
- **Links now checkable:** 0 to 4 violated per run, each explained below.

## Before and after

| Scenario | Profile correct | Cost plan matched | Program matched | Links violated / unknown | RFP new failures |
|---|---|---|---|---|---|
| ST03-base | 44 → 45 | 21 → 22 / 38 | 29 → 39 / 71 | 0/0 → 1/0 | 0 → 0 |
| ST03-da-pathway | 45 → 45 | 24 → 24 / 39 | 33 → 40 / 73 | 0/12 → 1/0 | 0 → 0 |
| ST03-strip-out | 44 → 45 | 22 → 23 / 39 | 25 → 34 / 74 | 0/0 → 0/0 | 0 → 0 |
| ST04-base | 65 → 65 | 33 / 49 | 51 / 95 | 0/52 → 3/0 | 0 → 0 |
| ST04-cdc | 65 → 65 | 32 / 49 | 49 / 91 | 0/47 → 4/0 | 0 → 0 |
| ST04-flood | 64 → 64 | 33 / 51 | 51 / 97 | 0/52 → 3/0 | 0 → 0 |
| ST06-base | 44 → 45 | 16 → 19 / 36 | 24 → 30 / 86 | 1/0 → 1/0 | 5 → 0 |
| ST06-pump-tank-only | 45 → 46 | 13 → 14 / 28 | 22 → 22 / 62 | 0/0 → 0/0 | 5 → 0 |
| ST06-tank-consent | 44 → 44 | 18 → 22 / 38 | 26 → 32 / 87 | 0/12 → 1/0 | 5 → 0 |

The "before" column is the first run re-scored with today's shapes and
keys, so it differs from the first report's numbers.

Every fee total, firm name, invoice amount and invoice flag stays correct in
all nine runs.

## Violated links

- **F52 (new app defect):** in ST03 base and DA, "Fit off electrical and
  hydraulic services" is not linked after "Install joinery".
- **F53 (new app defect, F29 family):** on T04, site establishment, erosion
  controls and bulk earthworks all start on construction day 0.
- **Known gap, T04 base and flood:** the single construction certificate
  sits before the design-and-construct contractor's documentation.
- **Known gap, ST04-cdc:** the D&C plus CDC job falls back to the generic
  template, which puts documentation before novation. This needs an owner
  answer (question 1 below).
- **Scorer limit, T06:** a start-to-start link that the scorer reads as
  finish-to-start.
- **Shape error, fixed:** T06 had one sprinkler row where the app now
  rightly has two (pipework, then fit-off). This is corrected and re-scored.

## Still open

- **F51:** T06 structure alterations and T03 painting are still filed as
  existing.
- F52 and F53 above.
- The ST06 work type now scores `conflict_false`. This is a scorer limit:
  refurb is correct, but the scorer counts any cell with two losing
  alternatives as a conflict.
- Missed "no" flags and `proj.regulated_design` (as before).

## Checks

- **Go suite:** passes apart from TestProgramRoutes, which fails without
  these changes too.
- **Speed:** the intake bench gate passes on `bd24ab1`.
  - profile edit p50 32 ms (budget 50);
  - Spec Home profile edit p50 45.4 ms against 48.4 ms before the fixes
    (budget 50). The first run after the Go suite read 55.9 ms; a clean
    re-run passed, so the margin is thin, as it was before;
  - program generate p50 38 ms (budget 300);
  - whole intake p50 336–429 ms (budget 1,000).

## Owner questions

1. Answered (R20): on D&C with a CDC the contractor is usually engaged
   first, and the CDC follows its design. That made the ST04-cdc order an
   app defect (F54), now fixed with a new `sk.new-dc-cdc` skeleton. Re-run
   in `runs/20261107-st04-cdc-r20/`: skeleton matched, 0 wrong-stage rows,
   links violated 4 → 2 (the two left are F53).
2. The five questions in `docs/design/2026-11-07-combined-services-package.md`
   (R17), before anything is built.
