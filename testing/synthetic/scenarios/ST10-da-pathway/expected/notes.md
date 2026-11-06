# ST10-da-pathway notes

The ST10 base with one change: the second storey exceeds the Housing Code side setback, so complying development is
not available and one DA to Sutherland Shire Council covers demolition of the existing dwelling and the new
two-storey dwelling, followed by a construction certificate. Only `hdr.cond.planning` differs from the base
(cdc -> da). 21 documents in 4 waves: base d01-d18 with d01, d04, d05, d06, d07, d10, d13 and d15 changed; d19 town
planning fee proposal, d20 landscape fee proposal and d21 statement of environmental effects added.

## Passing mentions checked

Searched every base `content`, `carries` and title block for CDC, complying development, Housing Code, certifier
and lodgement. d01, d04, d05, d06, d07, d10, d13 and d15 named the CDC and were changed; d02 keeps its land
eligibility item (the `cdc-land-eligible` trap, which now points away from the pathway). d03, d08, d09, d11, d12,
d14 and the invoices never named the pathway. Fee amounts are unchanged; only the d05, d06, d07 and d10 stage labels
change. After render review: d05 Stage 3 is "DA and CC documentation, demolition plan and lodgement"
($42,000) and Stage 4 stays "Tender" ($7,000); d21's panel sentence reads "the application does not need referral
to the local planning panel" (value no). `flag.architect_lodges` stays yes (the architect lodges the DA; the town planner writes the SEE). The cl 4.6,
panel and community sentences are rewritten for a DA and keep their "no" values.

## Traps

As the base. d21 is clean. Review finding 2: the d13 and d21 community sentence now reads "No community
consultation is required beyond council's standard neighbour notification; no public exhibition" (value no).

## What the scorer should see

Shape trigger `da-pathway`: town planner and landscape packages (also created by d19 and d20), DA fees and
contributions, Landscape designer and Town planner cost lines, DA lodgement, notification, determination and CC
rows, the town planner SEE and certifier CC rows in the RFPs; the CDC fee line and CDC program rows removed; the
skeleton becomes `sk.new-traditional-da`.

## Missed by design

As the base: `proj.target_construction_start` and `proj.regulated_design` only in the d04 email; d02 and d07
register number and date in Markdown tables.

## Key correction after run `20261106-t10-t12` (6 November 2026)

- `proj.lga` also accepts "Sutherland". The ST12 keys for the same council already accept it, as the ST11
  keys accept "Parramatta" for the City of Parramatta: the keys accept the area name without its "Shire" or
  "City of". The run read "Sutherland" from "Sutherland Shire Council". Whether the app should keep "Shire"
  is noted in FINDINGS (F22).
