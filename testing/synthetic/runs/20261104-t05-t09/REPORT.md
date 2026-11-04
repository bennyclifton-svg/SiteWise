# Run 20261104-t05-t09: pilots T05 and T09

The second and third pilot types, run end to end on 4 November 2026. App
at `c56c9de` (main, with auth/accounts and procurement/appointment-fee
merged). Same harness as the T01 runs: a separate SiteWise on :8091 with
its own database, live Jev, Update pressed after every wave, then cost-plan
prepare, program prepare and one RFP per services package. Nothing was
corrected, accepted or dismissed.

Scores: `<scenario>/field_score.json` (register, profile, invoices, fees)
and `<scenario>/shape_score.json` (procurement, cost plan, program, RFPs).
Findings are in `../../FINDINGS.md`, section "From run
`runs/20261104-t05-t09`" (F13 to F16, plus repeats).

## Results

| Scenario | Docs | Profile correct | Conflicts (true value in) | Missed | Fees: stated total | Procurement matched | Cost plan matched | Program matched |
|---|---|---|---|---|---|---|---|---|
| ST05-base | 19 | 52 | 5 (+1 without) | 2 (+1 by design) | 5 of 6 | 6/9, 4 false + | 19/35 | 31/79, `sk.generic` |
| ST05-heritage-conservation-area | 22 | 51 | 6 (+1 without) | 2 (+1) | 7 of 8 | 8/11, 3 false + | 22/37 | 31/81, `sk.generic` |
| ST05-first-floor-addition | 19 | 51 | 5 (+1 without) | 3 (+1) | 5 of 6 | 6/9, 4 false + | 21/37 | 31/83, `sk.generic` |
| ST09-base | 16 | 27 | 2 | 1 (+3) | 0 of 3 | 2/3, 2 false + | 9/20 | 1/32 (no skeleton exists) |
| ST09-podium-over-basement | 18 | 26 | 3 | 1 (+3) | 0 of 5 | 4/5, 1 false + | 9/24 | 5/36 |
| ST09-hazardous-materials | 18 | 26 | 3 | 1 (+3) | 0 of 4 | 3/4, 2 false + | 9/22 | 5/34 |

Register: status, kind and title correct on every filed document, except
T09's conditions of tender (kind blank). Lifecycle is mostly blank, as in
T01 (known gap).

## Did each variant's change land?

| Variant | Key the trigger changes | App |
|---|---|---|
| T05 first-floor-addition | storeys 2, added GFA 55 m², GFA 195 m² | all three correct |
| T05 heritage-conservation-area | `det.heritage_status` conservation_area | **blank**: 14 passages for, 1 against (F15) |
| T09 podium-over-basement | class 2 + 7a, budget $1,960,000 | both correct |
| T09 hazardous-materials | building year 1986 | correct |

The planned packages from added fee proposals appeared: heritage and town
planner (T05 heritage), landscape and hydraulic (T09 podium), hazmat (T09
hazmat).

## Traps

- Caught: the 15% GST invoice (`gst_rate`, T05); the "flood test", "Dulux
  Heritage", "sewer main in the street", CDC-permitted, "bathroom below",
  "fire stairs" and remediation-without-contamination traps set nothing
  wrong.
- Partly fell: "schedule of rates" (T09) put `schedule_of_rates` beside
  `lump_sum`, blanking the contract basis (F15); "podium over the basement"
  left `flag.basement` a conflict, with the true "no" among the values.
- Fell: the invalid ABN (T09) is never flagged; the app cannot see it (F13).

## What the app did badly, by impact

1. **The program opens on `sk.generic` for every T05 run.** Two causes
   meet: the procurement route is never read (F4, open since T01) and the
   work type is a blank conflict (F15). About half the expected
   activities are missing.
2. **No T09 fee proposal has a stated total** (0 of 12 across the three
   scenarios), so no remediation consultant can be compared or awarded at
   a stated total (F14).
3. **Lopsided evidence still blanks a value** (F15), including the heritage
   status the heritage variant exists to test.
4. **Remediation still gets almost nothing built in**, as the knowledge
   index predicted: no skeleton, and certifier and authority-connection
   cost lines that a remediation job does not have.
5. Repeats: F1 (Update cut off: 15 times across the six runs), F2 (council
   area and now the site address), F3, F7, F8.

## Notes on the keys

- `proj.regulated_design` is keyed readable in T05 (the checker derives
  it from the fee proposal's kind) and unreadable in T09; the app has no
  reading for it in either. Declared in each notes.md.
- `proj.client` in T09 was predicted to be cut at " - "; instead the app
  kept the care-of party ("…, c/- Rhiannon Teale, Northgate Strata
  Management"). The key stays the owners corporation's name.

## RFP scores (re-scored after the scorer fix, 4 November)

The first scoring overstated RFP failures. Since the scope-by-stage change
(3 November), the app shows an `option` or `ask` answer as included, and
the scorer translated the shapes' `expect` into that view but not their
`app_differs`. Fixed in `tools/score_shapes.py` (`_stage_view`, two new
tests); `shape_score.json` files are re-scored.

| Scenario | RFPs found | expect ok | new failures | known differences | awaiting user input |
|---|---|---|---|---|---|
| ST05-base | 6/8 | 93 | 36 | 13 | 18 |
| ST05-heritage-conservation-area | 7/9 | 93 | 36 | 13 | 18 |
| ST05-first-floor-addition | 6/8 | 93 | 36 | 13 | 18 |
| ST09-base | 2/2 | 19 | 4 | 9 | 2 |
| ST09-podium-over-basement | 4/4 | 18 | 6 | 9 | 2 |
| ST09-hazardous-materials | 2/2 | 19 | 5 | 9 | 2 |

The T05 surveyor and certifier RFPs are missing because those packages are
not proposed (procurement "missed built in").
