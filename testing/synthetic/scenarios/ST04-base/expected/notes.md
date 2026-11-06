# ST04-base notes

Ironstone Circuit warehouse: Redgum Logistics Estates Pty Ltd builds a new single-storey, single-tenant warehouse
(7,500 m2, 12 m clear, 12 recessed docks, ESFR sprinklers, off-site precast panels, steel portal frame) with a 500 m2
single-storey office on a vacant, benched 1.6 ha estate lot at 14 Ironstone Circuit, Erskine Park (Penrith City
Council, E4 General Industrial). Local DA determined by council staff, then an AS 4902 D&C lump sum with the
architect, structural and civil engineers and landscape architect novated; budget $14,800,000 ex GST. 19 documents in
4 waves (d01-d04, d05-d10, d11-d16, d17-d19); every T04 trigger denied or absent.

## Traps

- d02 `cdc-available-on-land`: the certificate says the Industrial and Business Buildings Code may be carried out on
  the land; the project is a DA because the building does not meet the code's rear setback standard (d01, d09).
  `potentially-contaminated-negated`: "no record that the land is contaminated" fires `contaminat`.
- d09 `planner-lodges`: "We will lodge the development application" fires architect_lodges; value no. d05 also says
  the planner lodges; d01 "A development application will be lodged" fires it too (incidental, answer no).
- d10 `tfnsw-denied`: "so it will not be referred to TfNSW" asks state_road and tfnsw_referral; both stay no, `sidra-denied` (intersection_modelling stays no).
- d11 `excavated-to-a-depth` (basement), `bore-logs` (water supply stays reticulated), `groundwater-negated`.
- d13 `floodlights` (flood stays false), `public-exhibition` (community stays no), `planning-panel-denied`.
- d14 `hydrocarbon-contaminants`: "treat hydrocarbon contamination washed from the hardstand" fires `contaminat`;
  stormwater pollutants are not land contamination, so the key stays false.
- d19 invoice `gst-15-percent`: GST $2,820 on $18,800; the total adds up, so `gst_rate`, not `arithmetic`.
- Messy tier: d02 and d07 (table title blocks) and d18 (table invoice).
- d15/d16: A-1100 Rev A and Rev B; d16 supersedes d15 (the app cannot link it today, DISCOVERY.md §4.5). The drawing
  number has no "DA" in it, so the CDC variant keeps the same number.
- Phrasing chosen for the readers and DOMAIN.md §10: the sprinkler sentence stands alone; "top of storage" and
  "racking" never share a sentence with "fire"; "existing building" only in the singular denial; "no curtain wall or
  unitised facade system" (facade_type trigger); "AS 4902"; "all-electric ... no natural gas"; mains water and a
  sewer junction, never "water main" or "sewer main"; "No works in the public road other than the two heavy-duty
  vehicle crossings", never "road reserve"; no "Stage 1", "building a" or "demolition".
- Possible incidental (not changed): "top of storage" is also a trigger for `hdr.scale.clear_height_m`, so the 11 m storage line
  could be read as the clear height. The key is 12, from "Clear height: 12 m to the underside of the haunch".

## Missed by design

- `proj.target_da_lodgement` (2026-06-30) and `proj.target_construction_start` (2027-02-01) are only in the d04 email.
- `hdr.scale.hardstand_sqm` and `hdr.scale.office_percent` have no triggers in the warehouse scale fields, so the
  app never asks them (`not_read`).
- d02's certificate number (PC-2026-00418) and date, and d07's number (CCE-P-0412) and date, sit in tables, so the
  register is expected to miss them.
- `proj.regulated_design` (no) is stated in d01 and d12 but has no extraction; keyed `not_read`.
- Derived keys (Type C, the C3D3 limits, EV readiness, Section J DTS) are copied from the shape and are blank today.
- Novation follows the shape and C1; the app sets option on every consultant under D&C (shape gap).


## Review (2026-11-07) applied

- `proj.regulated_design` re-keyed `not_read` (no extraction).
- Dock count reworded to "12 dock doors, recessed, ..." in d01 and "12 dock doors" in d13, so `dockCountRe` gives a candidate.
- d10/d13 tfnsw denial reworded to "so it will not be referred to TfNSW" (fires `referred to tfnsw`); the shape's
  `traffic-generating-development` negation changed the same way (DOMAIN.md §9 notes it).
- d12 fire_specialist_systems reworded so "fire performance solution" sits next to "facade, cladding".
- d01 office line is now "Office share: about 6% of the gross floor area" (no 500 m2 GFA/floor-area candidate).
- d14 trap made live with "hydrocarbon contamination" ("contaminants" never matched `contaminat`); DOMAIN.md §10
  item 8 and the shape's keyword-trap gap corrected.

## Shape comments

- `det.ncc_class` [7b, 5] derives `flag.mixed_use` yes (shape gap already noted); not a scenario defect.
- `det.climate_zone` (6, from d12) is not in the shape's profile; keyed because d12 states it.
- The `traffic-generating-development` negation was unreadable by the tfnsw_referral trigger; reworded (review finding 3).

## Render review (2026-11-07) applied

- `det.site_class` (M) and `det.saline_soil` (true) keyed from d11 ("classified as site class M (moderately reactive)
  under AS 2870", "The soils are moderately saline"), and added to d11's carries in the manifest. **Owner:** neither
  key is in `types/T04-warehouse/shape.yaml`'s profile; the checker accepts them, the shape was not edited.
- `fees.json` firm phone and address now follow the letterheads written in the base corpus (d05-d10, and the d17-d19
  invoices); the variants copy those documents, so their keys changed the same way.
- Base corpus wording (no key effect): d10 exclusion "Parking demand surveys at comparable sites"; d13 heading
  "## Stormwater" and "reports submitted"; d12 perimeter access with the office-frontage performance solution; d15/d16
  "Two fire water tanks and pump room".

- Owner ruling R15 (7 November 2026): the vehicle crossings are work in the public road, so `flag.public_road` is keyed yes (was no). Ruling R18: the DA reason is now the code's rear setback standard, since a height breach would also need a clause 4.6 variation, which the documents deny.
