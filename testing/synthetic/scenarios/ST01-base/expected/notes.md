# ST01-base notes

The Nguyen house: a new architect-designed two-storey, four-bedroom house with a double garage on a vacant 650 m2
R2 lot at 27 Corella Rise, Denistone (City of Ryde). DA to council, select tender, HIA lump-sum contract; budget
$1,450,000 ex GST. 17 documents in 4 waves; every T01 trigger denied or absent.

## Traps

- d02 `cdc-land-eligible` (messy): the s10.7 certificate says complying development may be carried out on the land.
  Tests that land eligibility does not set `hdr.cond.planning` to cdc.
- d03 `sewer-main-in-street`: "connect to the sewer main in the street" matches the authority_asset trigger (trap 8).
- d03 `trees-present-not-affected`: trees are listed, none affected; arborist flags stay no.
- d07 `osd-negated` (messy, GST-inclusive): "no OSD is required" matches the OSD trigger (trap 10).
- d10 `excavate-to-depth`: topsoil strip "to a depth of 300 mm" matches the basement trigger (trap 9);
  `h2-timber`: H2 treated pine next to the site class (trap 4).
- d11 `garden-beds` (trap 13) and `no-demolition` (trap 1: "No demolition is required" adds site.demolition today).
- d15 invoice: arithmetic error ($3,800 + $380 stated as $4,280).
- d12/d13: site plan A-101 at Rev A and Rev B; d13 supersedes d12 (the app cannot link it today, DISCOVERY.md §4.5).
- d07 and d17 use Markdown-table title blocks, so their number and date are expected to be missed.

## Missed by design

- `proj.target_construction_start` (2027-02-01) is only in the owner's email d04 (correspondence is not read).
- d02 register number (PC-2026-01187) and date (2026-02-10) sit in a Markdown table and `Certificate no` is not a
  number label, so the app is expected to miss both; the key keeps the true values. The same holds for d07 and d17.
- Drawing values (site area, garage on d12/d13) are also stated in read documents, so nothing else is lost.

## Shape comments

- `flag.arborist_offsite_trees` is not in the shape's base `profile`, though the trees-affected trigger lists it; this
  key adds it as "no" so the variant has a base value to differ from. Suggest adding it to the shape profile.
- `det.site_class` (M) is stated by the geotechnical report and keyed here; the shape's base profile omits it.
