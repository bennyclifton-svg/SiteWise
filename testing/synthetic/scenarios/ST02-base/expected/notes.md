# ST02-base notes

Calloway Street apartments: a new six-storey residential flat building of 64 apartments (Class 2) over two basement
levels (Class 7a) on three amalgamated R4 lots at 18-22 Calloway Street, West Ryde (City of Ryde), replacing three 1962
dwellings. DA determined by the Ryde Local Planning Panel, then a D&C contract under AS 4902 with the architect,
structural and civil engineers novated; budget $27,600,000 ex GST. 18 documents in 4 waves (d01-d04, d05-d09,
d10-d15, d16-d18); every T02 trigger denied or absent.

## Traps

- d03 (adversarial): `three-buildings` (multi_building pattern), `water-main-in-street` (authority_asset),
  `street-trees-listed` (arborist_offsite_trees), `existing-single-storey` (storeys).
- d05: `planner-lodges` ("lodges" fires architect_lodges; value no).
- d09: `public-exhibition` (community stays no), `we-lodge` (architect_lodges stays no).
- d10: `seepage-not-groundwater` (groundwater stays no).
- d11: `stage-1-cc` (staged stays no), `new-street-trees`, `basement-drained`.
- d13: `road-traffic-noise` (noise_affected_site stays false).
- d18 invoice: `gst-15-percent`: GST $1,800 on $12,000; the ledger flags `gst_rate`, not `arithmetic`.
- Incidental, not declared: d01 "Strata subdivision on completion" is not `stakeholder_complexity: strata`;
  d01 "A development application will be lodged" also fires architect_lodges (value no); d05 "Stage 1 … Stage 4" fee
  stages fire the staged pattern (key stays no from d01 and d11); d02 "complying development" item is land
  eligibility, not the pathway.
- d14/d15: A-DA-1101 Rev A and Rev B; d15 supersedes d14 (the app cannot link it today, DISCOVERY.md §4.5).
- Phrasing chosen for the readers: "64 apartments" (units trigger), "Rise in storeys: 6", "NCC building
  classification: Class 2 … Class 7a", "Facade system: … no curtain wall or rainscreen" (facade_type trigger),
  "AS 4902" (contract_form trigger), "no natural gas supply", "built in 1962", and each "no" flag worded so its
  trigger fires ("No staged construction", "not multiple buildings", "not traffic-generating development",
  "No intersection modelling or SIDRA", "No green travel plan").

## Review (2026-11-06) applied

- Shape: `traditional-procurement` now lists `fact.contract_form`; the base keys `as4902` exactly (the
  traditional variant keys `as4000`).
- `det.effective_height` 18.6 keyed from d12 ("Effective height: about 18.6 m").
- d01 says "a new building on an amalgamated site" (no longer "cleared").
- d08/d10 standpipe readings run four weeks, to fit the 11 March to 24 April dates.

## Missed by design

- `proj.target_construction_start` (2027-11-01) is only in the d04 email (correspondence is not read).
- `hdr.scale.avg_unit_size_sqm` (80) is stated in d01, but the taxonomy gives that field no trigger, so the app is
  expected to miss it even though the source is a read kind.
- `proj.regulated_design` ("yes") is stated in d01 and d12 (read kinds), but the app has no reading for it
  (FINDINGS: no extraction in `project_details.yaml`); it will score as missed.
- d02 register number and date, d07 number and date, and d12 number, revision and date sit in Markdown tables, so
  they are expected to be missed; the keys hold the true values.

## Shape comments

- `hdr.scale.avg_unit_size_sqm` has no trigger in `taxonomy.yaml`; suggest the shape note that it is never read.
- `det.existing_building_year: 1962` sits beside `det.existing_building: false`; the year describes the demolished
  dwellings. Kept as the shape says.
- `flag.panel` note: the base keeps the CIV under the regionally significant threshold (budget $27.6M ex GST, no CIV
  stated), so the Ryde Local Planning Panel determines it.
