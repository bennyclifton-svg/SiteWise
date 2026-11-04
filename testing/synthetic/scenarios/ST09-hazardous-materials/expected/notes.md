# ST09-hazardous-materials notes

The base with the building constructed in 1986: the asbestos register lists bonded fibre cement balcony soffits
and the original bituminous membrane and mastics are of unknown composition. A hazmat fee proposal (d17) and survey
report (d18) are added. 18 documents in 4 waves. Only `det.existing_building_year` differs (2008 to 1986).

## What changed

d01, d03, d08, d10 change the year (every "constructed in 2008" and "strata plan registered 2008"). d03 and d08
replace the base negation "the 2019 hazardous materials survey found no asbestos in balcony build-ups" with the
trigger's evidence. d09 adds licensed asbestos removal and clearance before membrane works; d13 adds the licensed
asbestos removalist to tenderer eligibility. Fee proposals and invoices are unchanged.

## Traps

- d18 `not-contaminated-land`: "not a contaminated land assessment" and "asbestos remediation" match the
  contamination trigger; `det.potentially_contaminated_land` stays false (from d02).
- Base traps carry over.

## Missed by design

As the base.

## Shape comment

- Asbestos removal is known from d18 (8 April) before the defect report d08 (14 April), yet the budget stays at the
  base $1,680,000, because the hazmat trigger lists only `det.existing_building_year`. Suggest adding
  `proj.budget_stated` to the hazmat trigger's `profile`.
- The d09 `strip-out` trap's expect is reworded: the only demolition item is the asbestos removal the trigger adds.
