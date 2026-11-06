# ST03-base notes

Avenor Capital Partners fits out the whole of Level 14 (1,200 m2 NLA) of Wenlock Place, 211 Wenlock Terrace, North
Sydney: an occupied 24-storey A-grade Class 5 tower constructed in 2008, handed over in base-building condition. CDC
by a private certifier; interior designer leads; traditional lump-sum tender under AS 4000; budget $2,400,000 ex GST.
18 documents in 4 waves; every T03 trigger denied or absent.

## Where this departs from brief 03, and why

- **Fee proposals.** The brief's architect, structural, civil and geotechnical engineers are not needed on a base
  fit-out (DOMAIN.md section 4; the architect is in `not_expected_procurement`). The base has the shape's consultants
  instead: interior designer (d05), one multi-services engineer (d06, `consultant.services`, the alias the shape gives
  mechanical, electrical, hydraulic and fire), BCA (d07, dropped on the BCA row), access (d08), certifier (d09) and QS
  (d10). The project manager wrote the brief (d01) and has no fee proposal.
- **Wave 1** has no survey: the tenant's equivalent is the landlord's base building information and fit-out guide (d03,
  author role `owner` = the building owner's manager; the client is the tenant).

## Traps

- d01 `make-good-at-expiry` (future lease obligation, not strip-out) and `kitchenette-not-kitchen`.
- d02 `cdc-land-eligible` (messy) and `heritage-next-door` (205 Wenlock Terrace).
- d03 `base-building-da` (DA 2004/0617 is the tower's), `base-building-nabers`, `sprinklered-not-works`,
  `storeys-of-building`.
- d11 `fire-stairs` (stair implied pattern) and `basix-not-applicable` (energy-rating pattern).
- d13 `no-strip-out` (strip-out implied pattern) and `core-bathrooms` (wet-room implied pattern).
- d17 invoice `total-misadded`: $6,000 + $600 printed as $6,900; flag `arithmetic`.
- d06 states GST-inclusive figures ($68,200 inc = $62,000 ex): `gst_inc_normalised`.
- d14/d15: ID-101 Rev A and Rev B; d15 supersedes d14 (the app cannot link it today, DISCOVERY.md section 4.5).
- Fee phases are named, never "Stage 1..", so the `staged` trigger is not fired by fee tables.

## Missed by design

- `proj.target_construction_start` (access date 2026-09-07) is only in the d04 email (correspondence is not read).
- `hdr.scale.floor_plate_sqm` and `hdr.scale.grade` have no triggers; `proj.regulated_design` has no extraction.
- `det.type_of_construction` is not asked; the app derives it from `det.ncc_class` and `det.rise_in_storeys`
  (rule.ncc.type-of-construction) only once both are reviewed, so it stays unknown on a harness run.
- `hdr.scale.tenancies` (1): `tenancyCountRe` needs "<n> tenancies", so a single tenancy gives no candidate. The
  natural d01 sentence is kept to record the gap.
- All five are keyed `not_read`.
- d02, d06 and d17 have Markdown-table title blocks: the d02 and d06 number and date are expected to be missed (keys
  keep the true values).

## Wording chosen for the readers

- "constructed in 2008" (the year trigger needs built/constructed in); "existing building"; "Class 5 building";
  "rise in storeys of 24"; "climate zone 5"; "no gas supply"; "NLA"; "single tenancy".
- Tenant-supplied items in their own sentence after a colon, as the shape asks.

## Shape comments

- `hdr.scale.floor_plate_sqm`, `hdr.scale.grade` and `det.type_of_construction` are in the base profile but the app
  never asks them; suggest the shape notes that.
- The app cannot read a count of one tenancy (`hdr.scale.tenancies`, `tenancyCountRe`).
- Consider moving `det.type_of_construction: A` to `expected_derived`, as T02 does (left for the owner; no shape edit).
- Brief 03's architect/structural/civil/geotechnical fee rule does not fit T03 (see above).

## Key changes after run `20261107-t03-t04-t06` (7 November 2026)

- `hdr.cond.quality_level`: `read_by_design: false` with `not_read`. The field applies to residential only, so the
  app never asks it on an office fit-out.
- `hdr.cond.operational_constraints` accepts `partial_occupation` or `live_environment` (owner ruling R19: an empty
  floor in an occupied tower is both).
- The certifier and BCA RFP keys in the T03 shape were re-keyed for F9 (see T03 `DOMAIN.md`).
