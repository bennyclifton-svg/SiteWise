# ST13-rise-three-storeys notes

The base project cut to three storeys: three shell shops under 18 apartments on two levels, over one basement level
(30 spaces), rise in storeys of 3, effective height 7.6 m, not sprinklered; budget $10,800,000 ex GST; the builder
provides HBCF insurance instead of the developer lodging a strata building bond. 18 documents, 4 waves; 10 changed,
none added or removed. Fee proposals and invoices are unchanged (amounts kept so the variant stays one change).

## Changed keys

The shape's `triggers[rise-three-storeys].profile` now lists all nine (review finding 1):

- `det.rise_in_storeys` 6 -> 3 (d01, d12)
- `hdr.scale.total_storeys` 6 -> 3 (d01)
- `det.effective_height` 18.9 -> 7.6 (d12)
- `det.sprinklered` true -> false (d12: "The building is not sprinklered ..."; the `sprinkler` trigger fires on the
  negation, so Jev must answer false)
- `hdr.scale.tower_levels` 5 -> 2 (d01)
- `hdr.scale.residential_units` 42 -> 18 (d01, d13)
- `hdr.scale.total_gfa_sqm` 5500 -> 2700 (d01)
- `hdr.scale.car_parks` 55 -> 30 (d01; also on drawings d14, d15)
- `proj.budget_stated` $24,500,000 -> $10,800,000 ex GST (d01; also `rfps.json` `budget_stated_cents`)

`hdr.scale.podium_levels` (1), `retail_gla_sqm` (600) and `site_sqm` stay at the base values. Fees and invoices stay
at the base amounts.

Kept as the base on purpose: the transfer slab (apartment grid over retail and car park), excavation to the No. 38
boundary, OSD, the Local Planning Panel (the planner's proposal still names it), D&C with novation, AS 4902.

## Traps and misses

As the base. The rise-of-3 HBCF and sprinkler thresholds are read by no generator (shape gap), so the app's program
and cost plan should differ from the base only through floor-cycle durations. `det.type_of_construction` is copied
from the shape (A); Table C2D2 gives Type B for the Class 6 part at a rise of 3, but C2D4 (top-storey classification) makes the whole
building Type A (d12 says so; `verified: false`). The Local Planning Panel stays: at three storeys the Chapter 4
referral line no longer applies, so d13 gives council's expected-objections referral as the reason (render review
finding 3; the 4-storey line not checked against the Direction).
