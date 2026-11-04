# ST05-heritage-conservation-area notes

The ST05 base with one change: the lot is within the Tallowwood Estate Heritage Conservation Area (LEP Schedule 5). The 1958 house is a neutral infill building, not a heritage item and not contributory, so the addition is assessed against the HCA controls in the DA. Only `det.heritage_status` differs from the base (conservation_area). 22 documents in 4 waves: base d01-d19 with d02, d05, d13, d15 and d16 changed; d20 heritage fee proposal, d21 town planning fee proposal and d22 heritage impact statement added.

## Traps

- d02 `cdc-not-available` (messy): complying development is not available in the HCA; planning stays da.
- d16 states "complying development is not available" (the cdc-pathway negation); planning stays da.
- Base traps d03, d07, d11, d12, d13 (`flood-test-shower`, `build-over-footprint`) and the d18 `gst_rate`
  invoice are unchanged. `dulux-heritage` is removed from d13.
- d13 and d22 say the house is neutral, not contributory; the value is conservation_area, not local_item.

## Passing mentions checked

Searched every base `content` and `carries` for heritage, conservation, Schedule 5, CDC and complying
development. Only d02, d13 and d16 carried them; d05 (SEE scope) and d15 (revision note) were also changed
so the documents agree. d01, d03, d04, d06-d12, d14 and the invoices do not mention heritage.

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
