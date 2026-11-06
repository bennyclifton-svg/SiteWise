# ST04-flood-affected notes

The base Ironstone Circuit warehouse on a lot within the flood planning area, with an overland flow path along the
eastern boundary and the warehouse floor set 500 mm above the 1% AEP flood level. Only `det.flood_hazard` (false ->
true) and `det.flood_planning_level` (new, 31.2 m AHD) differ from the base. 19 documents in 4 waves.

## Changes from the base

- d02 certificate: "The land is within the flood planning area and flood related development controls apply" and
  "Flood planning level: 31.2 m AHD (1% AEP flood level 30.7 m AHD plus 0.5 m freeboard)".
- d07 civil fee: adds "Flood impact assessment and floor levels for the DA" at $15,400 inc GST ($14,000 ex); total
  $103,400 inc GST ($94,000 ex). It also states the lot is within the flood planning area.
- d13 SEE: the flooding section carries the trigger's evidence (flood planning area, floor 500 mm above the 1% AEP
  flood level, overland flow path along the eastern boundary). The floodlights trap stays; its expectation changes
  (the lighting sentence is not the evidence).
- d14 civil report: a new flooding section with the flood planning area, the overland flow path and the floor level
  at RL 31.20 m AHD (the flood planning level).
- Checked: the d03 survey and d11 geotechnical report never mentioned flooding in the base and stay unchanged; no
  other document carried the base's "no flood" wording.

## What the scorer should see

Shape trigger `flood-affected`: the site.flooding investigate work item (not built in), the flood impact assessment
and raised pad cost lines (not built in), the council flood advice and flood impact assessment program rows
(partial), and the civil RFP's flood report and flood evacuation (option) positions (built in; civil scale goes
complex). The flood package rule can never fire (shape gap), so no new consultant is expected.

## Review (2026-11-07) applied

As ST04-base, plus the corrected `det.flood_planning_level` reason and the `site.flooding` note below.

## Missed by design

- `det.flood_planning_level` (31.2): a number determinant with no candidate rule of its own; `candidatesFor`
  (internal/profile/harvest.go) offers only m2 areas, so "31.2 m AHD" is never a candidate and the question is not
  asked (`not_read`), although the `flood planning level` trigger matches the d02 and d14 sentences.
- `site.flooding` can still be added through the evidence presence and action readings (the d07 flood phase, d13 and
  d14 floor-level sentences). If it appears, the scorer counts it as a match to the shape's investigate work item,
  not as noise; whether it appears is up to Jev.
- Otherwise as the base.

## Traps

Keep m2 figures out of the paragraphs that state the flood planning level (d02 flood item, d14 Flooding
section): a passage with an m2 figure would ask the FPL question with area candidates, and Jev must then answer
none.

As the base; d13 `floodlights` now expects true from the real flood sentences, not from the lighting.

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
