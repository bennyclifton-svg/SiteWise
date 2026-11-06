# Run 20261106-t10-t12-after-fixes: T10 to T12 after the F17 to F28 fixes

Re-run of the nine T10 to T12 scenarios after the fixes of 6 November 2026.
The app is local main `94dc593` plus:

- the quick fixes F20, F21, F22, F23, F25 and F28;
- the knock-down design D1 to D5
  (`docs/plans/2026-11-06-knockdown-two-buildings.md`);
- another session's uncommitted display-only changes, as in the first run.

Live Jev. The scenarios ran one at a time in one harness process. There were
no 409s this time. The keys and shapes are unchanged since the first run, so
the two runs compare directly.

## Results, before → after

| Scenario | Profile correct | Wrong | Program matched | Links violated | RFP new failures |
|---|---|---|---|---|---|
| T10 base | 52 → 54 | 3 → 1 | 56 → 64 /104 | 8 → 0 | 16 → 10 |
| T10 DA | 51 → 53 | 3 → 1 | 56 → 63 /104 | 5 → 0 | 19 → 13 |
| T10 separate demolition | 50 → 53 | 3 → 1 | 54 → 64 /107 | 6 → 0 | 20 → 12 |
| T11 base | 50 → 51 | 1 → 0 | 49 → 53 /107 | 5 → 0 | 4 → 2 |
| T11 CDC | 52 → 52 | 1 → 0 | 47 → 51 /106 | 4 → 0 | 6 → 6 |
| T11 three or more | 50 → 52 | 1 → 0 | 50 → 58 /109 | 5 → 0 | 4 → 2 |
| T12 base | 54 → 54 | 0 → 0 | 46 → 53 /73 | 4 → 0 | 10 → 10 |
| T12 DA | 54 → 54 | 0 → 0 | 47 → 51 /74 | 5 → 0 | 12 → 12 |
| T12 sewer main | 55 → 55 | 0 → 0 | 46 → 53 /75 | 4 → 0 | 10 → 10 |

These were unchanged or no worse:

- Skeleton right in all nine.
- No fabricated amounts.
- Invoices still right.
- Fee totals 54 of 57.
- Supplier 25 of 27, up from 24 (T12 base).

## Per finding

| Finding | Result |
|---|---|
| F17 existing building | **Fixed.** True in T10 base, T10 separate demolition, T11 base and T11 three-plus. T10 DA and T11 CDC are blank: the "new house" passage no longer denies it, but the survey's yes is below the floor (0.53). |
| F18 investigate actions | **Fixed.** Program matched up 4 to 10 activities per run; the built-in footing, slab, frame, window and lining rows now appear. |
| F19 old-house subclass votes | **Old-house votes gone** (T11 base: "house" 11 → 4). The 4 left are about the new dwellings ("two Class 1a dwellings"), so T11 base and CDC are still a conflict. See "still open". |
| F19 apartment fallback | **Fixed.** No Access, Fire Services or Vertical Transport consultant, RFP or cost line in any T11 run (extra RFPs 6 → 3). Cost: the shared residential list also drops balustrades and stairs, which the key expects (T11 cost plan 27 → 25 and 26 → 24). |
| F20 DA fees on CDC | **Fixed.** Gone from all six CDC runs; kept on the three DA runs. |
| F21 Caringbah South | **Fixed.** Full address in all T10 runs. |
| F22 "for client" | **Fixed.** Client correct in all T10 runs. Five T10 RFPs no longer block on Client. |
| F23 letterhead descriptor | **Fixed in code, not visible in this run.** The app reused the firm saved by ABN in the first run ("Civil and stormwater engineers"). Reading the real file now gives "DRUMMOND CIVIL PTY LTD". See "harness". |
| F25 building designer | **Small effect.** Filing is unchanged, because Jev sees only the label "Architect". The supplier match improved in T12 base. |
| F28 hold point | **Fixed.** Under-slab drainage → hold point → pour → cure in every run. Links violated 4 to 8 → 0 in all nine. |
| F30 dangerous goods | **Fixed.** In no brief. |

## Still open

- **The T11 subclass.** With the old house removed, the conflict is
  "townhouses" (11) against "house" (4) from passages about the two new
  Class 1a dwellings. Whether a dual occupancy is "house" or "townhouses"
  needs an owner ruling or a taxonomy note. F27 (counting passages, not
  documents) would also settle it.
- **F6.** The T10 BASIX proposal still files with no discipline (Jev 0.36),
  in all three runs.
- **Not attempted in this change:** F24, F26, F27 and F29, and the T01 open
  items.

## Harness

The firm directory is organisation-wide, and every harness run uses the
same organisation. A firm saved in an earlier run, wrong or right, is
reused by ABN. Firm-name scores therefore need a fresh `sitewise_synth`
database, or each run's results need checking against the first one.

## Tests

The Go suite passes, except two tests that fail with or without these
changes:

- `TestMeridianAnswerKey`: wrong `sig.acoustic-separation-stated`, 39/46.
- `TestProgramRoutes`: "a duration came from nowhere" on
  fr.existing-conditions.

`tools/check_knowledge.py` passes, with 0 errors.
