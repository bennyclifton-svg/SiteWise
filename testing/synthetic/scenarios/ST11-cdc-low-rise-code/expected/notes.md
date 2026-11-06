# ST11-cdc-low-rise-code notes

The base Kovac dual occupancy approved as complying development under the Low Rise Housing Diversity Code by a
registered certifier, with the Torrens subdivision under its own complying development certificate (verified: false).
Only `hdr.cond.planning` differs from the base (da -> cdc). 17 documents in 4 waves.

## Changes from the base

- d01 and d13 carry the trigger's evidence phrases in place of the two DA negations.
- d05 Stage 2 becomes CDC documentation and lodgement (same $24,000); `flag.architect_lodges` stays yes ("lodging the
  CDC application"). d06 and d07 name the CDC and the certifier; amounts unchanged.
- d04 (email) and d15 (revision note) no longer say DA.
- d10 town planner proposal and d16 SEE removed. Keys sourced from d16 keep their values from the other sources.
- d13 `flag.panel`, `flag.community` and `flag.cl46` reworded for a certifier; all stay no.
- d02's "complying development may be carried out on the land" trap stays; here it agrees with the pathway.
- Searched every base `content` and `says` for "DA", "development application", "council's determination", "CC" and
  "SEE"; d03, d08, d09, d11, d12, d14 and the invoices never named them, so they are unchanged.

## Missed by design

As the base: `proj.target_construction_start` only in the d04 email; `hdr.scale.avg_dwelling_size_sqm` (no triggers)
and `proj.regulated_design` (no reading) only on the drawings; d02 and d07 identity in Markdown tables; d15 -> d14
supersession not written.

## Shape comments

The trigger's `adds` "Separate approval for the subdivision" is met here by a second CDC. The scorer should expect the
app to keep suggesting the town planner and to block the certifier RFP on cdc (shape gaps).
