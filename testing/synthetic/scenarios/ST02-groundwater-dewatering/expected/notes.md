# ST02-groundwater-dewatering notes

The ST02 base with one change: groundwater sits at RL 37.2 AHD, about 3 m above bulk excavation level, so the basement is
dewatered during excavation (WaterNSW approval) and tanked. Only `flag.groundwater` differs from the base (no -> yes).
18 documents in 4 waves: base d01-d18 with d10 and d11 changed.

## Passing mentions checked

Searched every base `content`, `carries` and title block for groundwater, water table, seepage, standpipe, drained and
dewater. d10 (the finding, the seepage trap and "a drained basement with sub-floor drainage") and d11 ("drained cavity
basement walls with sub-floor drainage") were changed. d08 mentions standpipes with water level readings, which
describes the investigation, not a finding, and stays. d06 and d08 were written without the words groundwater or
water table. No invoice, email, drawing or other report names the groundwater condition.

## Traps

As the base, less d10 `seepage-not-groundwater` and d11 `basement-drained`, which no longer apply.

## What the scorer should see

Shape trigger `groundwater-dewatering`: civil "Earthworks suitability coordinated with groundwater advice" and
geotechnical "Groundwater monitoring and hydrogeological assessment" RFP rows; "Dewatering, treatment and discharge"
and "Tanked basement" cost lines and the monitoring, approval and dewatering program rows (not built in).

## Missed by design

As the base: `proj.target_construction_start` only in the d04 email; `hdr.scale.avg_unit_size_sqm` has no trigger;
`proj.regulated_design` has no reading; table title blocks in d02, d07 and d12.

## Shape comments

As the base. Review (2026-11-06): d10 is now `clean` (its trap was removed); the base review fixes apply here too.
