# ST05-first-floor-addition notes

The ST05 base with one change: the owners go up rather than out. A 25 m2 first-floor main bedroom and ensuite sits over the rear of the existing house, and the rear ground-floor addition shrinks to 30 m2, so the added GFA stays 55 m2 and only `hdr.scale.storeys` differs from the base (2). The existing rear roof is removed, the walls that carry the first floor are underpinned, and a stair is added. 19 documents in 4 waves: base ids, d01, d05-d08 and d12-d16 changed, none added or removed. The owners still vacate, as in the base.

## Traps

- Base traps unchanged (d02, d03, d07, d11, d12, d13, d18).
- d06 "no transfer structures … first floor carried on new steel beams": `transfer` is near, but the key
  stays no (the trigger needs 'transfer beam/structure' stated as present).
- d13 "the existing roof over the front rooms is unchanged" sits next to "two storeys": storeys is 2.

## Passing mentions checked

Searched base `content` and `carries` for single-storey, single storey, storey, 55 m² of new roof and roof
area. d01, d13 and d16 carried the storey value; d07 the roof area; d05, d06, d08, d12, d14 and d15 change
because the first floor changes their scope. d03 and d11 were written in the base without storey words, so
they stay. Invoices d17-d19 bill unchanged stages and do not mention the addition's form.

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
