# Run 20261104-after-fixes: the nine pilot scenarios after the F1 to F16 fixes

The same nine scenarios (T01, T05, T09: base plus two variants each), the
same corpora and answer keys, and the same harness. App: local main at
`b1bd66d` plus the harness fix `b3c1ae2`, with all six fix branches merged
(plan: `docs/plans/2026-11-04-synthetic-findings-fixes.md`). Live Jev.

Compared with `../20261102-fixes-1-4-6` (T01) and `../20261104-t05-t09`
(T05, T09). Both sides are scored by the current scorers (RFP scores use
the scope-by-stage view; see "Scorer changes").

A first attempt at this run was discarded: `serve.sh stop` had not killed
the old server, so the run hit the pre-fix build. Fixed in `serve.sh`
(`b3c1ae2`).

## Results

| Scenario | Profile correct | Conflicts | Supplier | Fee totals | Skeleton | Program matched | Procurement | Cost plan | RFP new failures |
|---|---|---|---|---|---|---|---|---|---|
| T01 base | 43 → 46 | 5 → 4 | 0/3 → 3/3 | 4/5 → 5/5 | generic → **new-traditional-da** | 51 → 51 /83 | 8 → 7 /11 | 25 → 24 /34 | 34 → 14 |
| T01 CDC | 43 → 45 | 4 → 4 | 0/3 → 3/3 | 4/5 → 5/5 | generic → **new-traditional-cdc** | | | | 32 → 15 |
| T01 trees | 43 → 46 | 5 → 4 | 0/3 → 3/3 | 5/6 → 6/6 | generic → **new-traditional-da** | | | | 27 → 14 |
| T05 base | 52 → 55 | 6 → 3 | 0/3 → 3/3 | 5/6 → 6/6 | generic → **extend-traditional** | 31 → 46 /79 | 6 → 8 /9 | 19 → 23 /35 | 16 → 4 |
| T05 heritage | 51 → 55 | 7 → 3 | 0/3 → 3/3 | 7/8 → 8/8 | generic → **extend-traditional** | 31 → 46 /81 | 8 → 10 /11 | 22 → 25 /37 | 16 → 6 |
| T05 first floor | 51 → 55 | 6 → 3 | 0/3 → 3/3 | 5/6 → 6/6 | generic → **extend-traditional** | 31 → 48 /83 | 6 → 8 /9 | 21 → 24 /37 | 16 → 4 |
| T09 base | 27 → 31 (all) | 2 → 0 | 0/3 → 3/3 | 0/3 → 3/3 | none exists | | | | 4 → 4 |
| T09 podium | 26 → 30 | 3 → 1 | 0/3 → 3/3 | 0/5 → 5/5 | none exists | 5 → 1 /36 | | | 5 → 4 |
| T09 hazmat | 26 → 31 (all) | 3 → 0 | 0/3 → 3/3 | 0/4 → 4/4 | none exists | 5 → 1 /34 | | | 2 → 1 |

T09 base and hazmat now score every readable profile value correct.

## Finding by finding

| Finding | Before | After |
|---|---|---|
| F1 Update cut off | 15 cuts in six runs; 22 of 30 in a direct probe | **0** in nine runs; 30 of 30 complete in the probe |
| F2 council area and address | "City of … Section/Duty"; address with "Date", ". It" | **Fixed**: LGA right in all nine; T05 address right |
| F3 inc-GST total kept inc | T01 civil $9,900, T05 civil $7,260 | **Fixed**: ex-GST totals in all |
| F4 procurement route | blank everywhere | **Fixed**: `traditional` in T01 and T05; skeletons now match |
| F5 arithmetic flagged as GST rate | T01 d15 `gst_rate` | **Fixed**: `arithmetic` in all three T01 runs |
| F7 s10.7 certificate as town planner | all nine | **Fixed** |
| F8 invoice supplier | 0 of 27 | **Fixed**: 27 of 27 |
| F9 to F12 RFP | certifier blocked on CDC, no inspections, garage = mixed use, no root-zone row | **Fixed** in knowledge and rules; inspection and BASIX clause numbers `verified: false` |
| F13 invalid ABN | never flagged | **Fixed**: flagged in all three T09 runs |
| F14 fee totals missed | 0 of 12 T09 proposals | **Fixed**: 12 of 12 |
| F15 lopsided choice blank | work type, heritage, contract basis, existing building | **Fixed**: all settle (amber, dissent shown) |
| F16 rise in storeys | missed in T09 | **Fixed** |
| Client with care-of party | T09 wrong | **Fixed** |
| Invoice date in a table | missed | **Fixed** |

## Worse or unchanged, and why

- **T01 base procurement 8 → 7 and cost plan 25 → 24.** The BASIX fee
  proposal (d09) filed with no discipline: Jev 0.59, just under the
  threshold. The same document scored 0.62 to 0.68 in every other run,
  before and after. Run-to-run variation at a borderline, not a fix side
  effect; F6 is fragile.
- **T09 program 5 → 1 matched.** The work type now settles as
  `remediation`, so the generic skeleton no longer fires; remediation has
  no skeleton or fragments at all (a known gap the shape names). The 5
  earlier matches were accidental.
- **Still open, unchanged:** T01's missed "no" flags (groundwater, staged,
  multi_building, sustainability target), gas supply, contract form and
  existing building; `flag.authority_asset`, `flag.community` and access
  constraints as conflicts; GFA and site area conflicts (the T01 documents
  disagree on purpose, 650 vs 650.2 m²); `proj.regulated_design` (no
  reading exists). These were listed in FINDINGS as "to check wording
  before calling them app misses".
- **T05 d04** (an email): the key says no discipline; the fix agent judges
  the top message is the architect's, so the key may be wrong. Not
  changed.

## Scorer changes

`tools/score_shapes.py` now reads a by-stage RFP the way it prints:
`option`/`ask` as included, `O`/`ask` positions as I, `done` as excluded,
and an out-of-scope position (printed X) as absent, for both `expect` and
`app_differs`. Three tests. Before this, the T01 comparison counted about
20 "new failures" per scenario that were only the new layout.
