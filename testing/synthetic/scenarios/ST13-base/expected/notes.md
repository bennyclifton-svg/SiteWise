# ST13-base notes

Halloran Street mixed-use: Halcyon Eastwood Pty Ltd replaces two 1965 single-storey shops and a dwelling on a 1,800 m2
three-lot site at 40-44 Halloran Street, Eastwood (City of Ryde, E1 Local Centre) with a six-storey shop-top building:
three shell shops (600 m2 GLA) under 42 apartments on five levels, over two basement levels (55 spaces). DA to the Local
Planning Panel, then an AS 4902 D&C lump sum with novation; budget $24,500,000 ex GST. 18 documents in 4 waves; every
T13 trigger denied or absent.

## Traps

- d02 `potentially-contaminated-negated` and d05 `contamination-denied`: both fire `contaminat` (d05 names
  service station, dry cleaner and underground tanks only to deny them); `det.potentially_contaminated_land` stays false.
- d03 `sewer-main-in-lane` (authority_asset), `street-trees-listed` (presence is not impact), `no-existing-osd`.
- d10 `planner-lodges`: "We will lodge the development application" fires architect_lodges; the planner lodges, so no.
  d06 also says the planner lodges; d01 "lodge the strata building bond" fires it too (incidental, answer no).
- d11 `groundwater-negated`: "No groundwater was encountered ... seepage only" (the regex matches the negation).
- d13 `design-review-panel`, `shop-top-housing`, `strata-subdivision` (stakeholder stays single_owner),
  `public-exhibition` (community stays no).
- d18 invoice `total-mistyped`: $20,200 + $2,020 printed as $22,420; the ledger flags `arithmetic`.
- d02 and d08 are messy (table title blocks); d17 is a table invoice.
- d14/d15: A-DA-1100 Rev A and Rev B; d15 supersedes d14 (the app cannot link it today, DISCOVERY.md §4.5).
- Phrasing chosen for the readers and the DOMAIN §10 phrase table (the T02 keyword traps): "existing building" in the
  singular; "built in 1965"; "one building ... not planning multiple buildings"; fee phases named, never "Stage 1";
  "early works" never "CC Stage 1"; no "water main", "road reserve", "MRV/HRV/SRV"; the street trees always paired with
  "not affected"; "AS 4902" for `fact.contract_form` (matches `\bas\s?4902`); "42 apartments" for the unit count.
  Documents carry an instruction not to write "building a/b" (multi_building `\bbuildings? [a-d]\b`) or "stage" plus
  a digit or single letter (staged).
- The client name, Halcyon Eastwood Pty Ltd, has no " - ", so it should not be cut.

## Missed by design

- `proj.target_da_lodgement` (2026-07-31) and `proj.target_construction_start` (2027-06-01) are only in the d04 email.
- d02's certificate number (PC-2026-01187) and date sit in a table, and `Certificate no` is not a number label, so both
  are expected to be missed. d08's number (LCE-P-0388) and date likewise.
- `proj.regulated_design` is stated in d01 and d12 (read kinds), but the app has no reading for it
  (`project_details.yaml` has no extraction), so expect a miss although the checker marks it read_by_design true.
- `hdr.scale.podium_levels` and `tower_levels` are stated only as prose ("one-level retail podium; the five
  residential levels sit above it"); there may be no reader for them.
- Novation in `procurement.json` follows the shape and C1 (architect, structural, civil, landscape yes; the four
  services engineers option; the rest no). The app sets option on every consultant under D&C (shape gap).

## Shape comments

- `flag.arborist_offsite_trees` is not in the shape's base profile; d13 states it ("no"), so the key carries it, as T01
  and T10 do (R10).
- `det.facade_type: standard` can only be set when a facade trigger fires; d06 says "no curtain wall or unitised facade
  system" for that.
