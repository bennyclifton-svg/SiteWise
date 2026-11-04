# ST09-podium-over-basement notes

The base with a ground-floor communal courtyard podium over a single-level basement car park, leaking into the car
park. The podium is added to the scope; landscape (d17) and hydraulic (d18) fee proposals are added. 18 documents
in 4 waves.

## What changed

d01, d03, d08 carry the trigger's evidence (water into the car park under the courtyard, calcite and
efflorescence on the B1 soffit, pavers on pedestals over a 2008 membrane). Every base mention of "no basement",
"no podium" or the at-grade car park (d01, d07, d08, d10) is changed. d05, d06, d09, d13 add the podium to the
scope. Fees and invoices are unchanged in amount, to keep the comparison with the base clean.

## Traps

- `podium-over-basement` on d01, d03, d05, d06, d07, d08, d09, d10, d13 and d17, and `podium-over-basement-quote`
  on d18: the high-risk basement trap (DOMAIN.md §10). Each of these names the existing basement car park.
  `flag.basement` must stay no: the basement is existing and nothing is excavated. Its "no" sources are d07 and
  d08, which say the works include no excavation and no new basement, and d10, which says the existing basement
  car park structure is not altered.
- Base traps carry over; the podium flood test in d08 and d09 adds to the flood-test trap.

## Missed by design

As the base: the target start in d04, the table title blocks of d02, d06 and d15, `proj.regulated_design`,
`proj.lot`, and `proj.client` cut at " - ".

## Trigger profile keys

With the shape's podium trigger now listing them, d08 states "NCC building classification: Class 2, with a Class 7a
basement car park" (`det.ncc_class` ["2", "7a"]) and recommends a construction budget of $1,960,000 ex GST
(about $280,000 for the podium), keyed as `proj.budget_stated` and `rfps.json` `budget_stated_cents`. Fees are
deliberately unchanged in amount.

`det.ncc_class` is not in `derived` for this variant: the 7a part is read from d08 and cannot be derived from
"apartments", so a derived ["2"] would contradict the keyed ["2", "7a"].
