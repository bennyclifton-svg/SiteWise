# ST06-tank-needs-consent notes

The base, except the enlarged pump room and tank take the basement store and two car spaces, below the number the
original consent requires. At the pre-lodgement meeting council advised that the tank and pump house are not complying
development and a DA is required; the level 3 fit-out joins the same DA, followed by a construction certificate.
22 documents in 4 waves. Only `hdr.cond.planning` differs (cdc to da). The budget is the base's $2,700,000.

## What changed

- d09 is re-dated 24 April 2026 (wave 3), after the 16 April pre-lodgement meeting and the tank position.
- d09 (certifier) and d12 (options report) carry the trigger's evidence in place of both negations; d09's fee table
  becomes a construction certificate, and the DA fire upgrade negation is reworded ("council's consent is not
  expected to require an upgrade of the existing fire safety measures").
- d13, d16 and d17 move the tank over the two car spaces; d17 is issued for the DA.
- Added: d21 town planner fee proposal (SEE and car parking assessment) and d22 SEE. BCA has no proposal; the app
  suggests it anyway and the shape expects it.
- d10 (interior designer) said it would document the fit-out "for the complying development certificate"; its
  scope line now says "for certification and tender", since it predates council's advice. Fees and keyed text are unchanged.
- Nothing else in the base mentioned the CDC: the brief, fee proposals d05 to d08, d11, reports d14, d15 and
  the invoices say only "certification", so they are unchanged.

## Traps

- d09 and d12 `car-spaces-not-basement`: "basement car spaces will be lost" hits the basement trigger again;
  flag.basement stays no.
- d22 `fire-storage-not-special-hazard`: "fire pump room and fire water storage tank" hits the fire special hazards
  storage pattern; flag.fire_special_hazards stays no (d12).
- d02 (unchanged) still says "Complying development may be carried out on the land". It is now a trap
  (`complying-on-land-not-pathway`): land eligibility is not the works' pathway, and planning stays da.
- Base traps carry over.

## Missed by design

As the base.

## Key changes after run `20261107-t03-t04-t06` (7 November 2026)

- `hdr.cond.quality_level`: `read_by_design: false` with `not_read`. The field applies to residential only, so the
  app never asks it on a commercial building.
- `hdr.cond.access_constraints` accepts `urban_constrained` or `restricted_hours` (owner ruling R19).
- The T06 shape's certifier RFP was re-keyed for F9, and `position.existing` now expects `app: absent` (see T06
  `DOMAIN.md`).
