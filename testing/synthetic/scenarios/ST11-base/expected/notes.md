# ST11-base notes

The Kovac dual occupancy: demolition of a 1958 brick house and an attached pair of two-storey, four-bedroom
townhouses (460 m2 GFA, 2 dwellings) with Torrens subdivision into proposed Lots 1 and 2, on a 700 m2 R2 lot at
18 Wattlebird Street, Carlingford (City of Parramatta). One DA, council OSD, architect-led tender, HIA lump sum,
budget $1,650,000 ex GST. 19 documents in 4 waves; every T11 trigger denied or absent.

## Traps

- d02 `cdc-land-eligible` (messy): the s10.7 says complying development under the Low Rise Housing Diversity Code
  may be carried out on the land; a certificate gets no header questions, and the pathway stays da.
- d03 `sewer-main-in-street` (authority_asset stays no) and `trees-present-not-affected` (gum near the rear
  boundary, present but unaffected; the street tree is on the verge in front of the northern neighbour, well
  clear of both new crossings: the trees-affected negation).
- d05 and d10 `fee-stages-not-staged`: "Stage 1", "Stage 2" ... fee stages fire the staged trigger; stays no.
- d07 `osd-existing-negated` (messy, GST-inclusive): "no existing on-site detention" fires osd_existing; stays no
  while osd_required is yes (council's policy, also in d16).
- d11 `loose-fill-register-negative`; bonded asbestos is found, which is the base (licensed removal before demolition).
- d12 `excavate-to-depth`: "excavate to a depth of 300 mm" fires the basement trigger; stays no.
- d06 and d12 `party-wall-not-neighbour` (the thickened footing under the separating wall): no
  site.adjoining-structures.
- d01 `rainwater-tank` ("a BASIX rainwater tank for each dwelling"): water supply stays reticulated.
- d13 DA reason: the design misses the code's landscaped area standard (not an LEP standard, so no cl 4.6).
- d13 `building-a-dual-occupancy` (multi_building stays no), `party-wall-not-neighbour` (no site.adjoining-structures),
  `fire-rated-wall` (no fire engineer or acoustic consultant), `rainwater-tank` (water supply stays reticulated),
  `dual-occupancy-as-house` (subclass townhouses).
- d18 invoice: total $4,610 against $4,100 + $410: `arithmetic`.
- Incidental: "lodge" in d05, d10 and d16 fires architect_lodges, which is yes; d10 says the architect lodges.

## Missed by design

- `proj.target_construction_start` (2027-02-01) is only in the email d04 (correspondence is not read).
- `hdr.scale.avg_dwelling_size_sqm` (230) has no triggers in taxonomy.yaml, so it is never read; it is stated only
  on the drawings d14/d15 (not read), so `read_by_design` is false. The read documents give GFA without a
  per-dwelling area so the GFA passage has one candidate.
- `proj.regulated_design` has no reading in the app (no extraction in project_details.yaml); its sentence sits
  only in the drawing general notes, so `read_by_design` is false.
- d02 register number (PC-2026-03318) and date (2026-02-09), and d07 number (MCE-P-0412) and date (2026-03-12),
  sit in Markdown tables, and `Certificate no` / `Our ref` are not number labels; the key keeps the true values.
- d15 supersedes d14 (A-101 Rev A -> Rev B); the app cannot write the link today (DISCOVERY.md §4.5).
- `hdr.scale.dwellings`: the integer candidates in "2 dwellings on proposed Lots 1 and 2" are 2, 1, 2; Jev must pick 2.

## Shape comments

- Extra keys beyond the shape's `profile`: `det.site_class` (M) and `flag.arborist_offsite_trees` (no), as the
  pilots did; suggest adding both to the shape.
- `hdr.scale.avg_dwelling_size_sqm` is unreadable (no triggers); the shape should say so or drop it.
- Applied after review (finding 1 and 2): `three-plus-townhouses` now lists `hdr.scale.avg_dwelling_size_sqm` beside
  `hdr.scale.dwellings`, and its "R3 zone" evidence sentence is deleted.
