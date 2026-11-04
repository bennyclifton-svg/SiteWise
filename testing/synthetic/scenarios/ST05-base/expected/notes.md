# ST05-base notes

The Ferreira house: a single-storey 55 m2 rear addition and internal alterations to a 140 m2 brick house built in 1958 at 41 Tallowwood Avenue, Concord (City of Canada Bay, R2). DA, owners move out, architect-led tender, HIA lump sum; budget $640,000 ex GST. 19 documents in 4 waves; every T05 trigger denied or absent.

## Traps

- d02 `cdc-land-eligible` (messy): the s10.7 certificate says complying development may be carried out on the
  land; `hdr.cond.planning` stays da.
- d03 `sewer-main-in-street`: "sewer main in the street" matches the authority_asset trigger; stays no.
  `trees-listed-not-affected`: a frangipani and a street tree are listed, none affected.
- d07 `osd-negated` (messy, GST-inclusive): "OSD is not required" after "existing gravity system" fires both OSD
  triggers; both stay no.
- d11 `fibre-cement-not-asbestos` and `loose-fill-register-negative`: the hazmat report uses the asbestos and
  loose-fill negations; no hazardous-materials consequence.
- d12 `excavate-to-depth`: topsoil strip "to a depth of 300 mm" matches the basement trigger.
- d13 `dulux-heritage` (heritage stays none), `flood-test-shower` (flood stays false), `build-over-footprint`
  (authority_asset stays no).
- d18 invoice: GST $345 on $2,300 (15%) with a total that adds up: `gst_rate`, not `arithmetic`.
- Incidental, not declared: d05 "Stage 1 … Stage 4" fee stages match the `staged` trigger; the key keeps
  `flag.staged` no from d01's "No staged construction". d05/d16 "lodgement" fires `architect_lodges`, which is yes.
- The header facts sit under `## Project description` and `## Contract and procurement` in d01 so that
  `HeaderRouted` asks them (F4); the brief avoids "renovation" so work type is not read as refurb.

## Missed by design

- `proj.target_construction_start` (2027-03-01) is only in the owner's email d04 (correspondence is not read).
- d02 register number (PC-2026-00942) and date (2026-02-16) sit in a Markdown table, and `Certificate no` is not
  a number label, so both are expected to be missed; the key keeps the true values. The same holds for the d07
  number and date (LCS-P-0311, 2026-03-13). d19 is an invoice in a table; the invoice reader is Jev, so no
  register loss.
- The drawings d14/d15 repeat the GFA figures, which are also in the read documents d13 and d16, so nothing is lost.
- d15 supersedes d14 (A-201 Rev A → Rev B); the app cannot write the link today (DISCOVERY.md §4.5).
- `proj.regulated_design` has no reading in the app; the d05 sentence is kept for realism. Review finding 2
  asked for `read_by_design: false`; disputed in form only: `validate_expected.py` derives `read_by_design`
  from the source kinds (d05 is commercial, a read kind) and rejects false, and the checker may not change.
  The key stays true with this line declaring the miss; the scorer should treat it as missed by design.

## Shape comments

- Applied after review (4 Nov): `hazardous-materials-found` now has `profile: []` (finding asbestos does not
  change the year built), and its adds no longer repeat base items (hazmat package, survey and demolition
  lines); it adds "Licensed asbestos removal and clearance".
- Applied after review: `first-floor-addition` now lists `hdr.scale.added_gfa_sqm` and `hdr.scale.gfa_sqm`.
  The variant still keeps 55 m2 (30 m2 rear, 25 m2 upstairs, split only on the unread drawings); no redo needed.
- Extra keys beyond the shape's `profile`: `det.site_class` (M), `hdr.scale.site_sqm`, `hdr.scale.gfa_sqm`,
  `flag.arborist_offsite_trees` and `flag.arborist_pruning` (both no), as T01 did. Suggest adding
  `flag.arborist_offsite_trees` and `det.site_class` to the shape profile.
