# ST03-da-pathway notes

ST03-base with one change: the certifier advised that the fit-out does not meet the complying development standards,
so a development application is lodged with North Sydney Council, then a construction certificate. 20 documents in 4
waves (base 18, plus the town planner's fee proposal d19 and the SEE d20). Profile differs only on
`hdr.cond.planning` (`da`).

## What changed

- Every read document that named the CDC now names the DA or CC: d01, d05, d06, d07, d08, d09, d11, d12, and the d04
  email and the d15 revision note. Fee amounts are unchanged; labels follow the new pathway.
- d02 and d03 keep their text; only their trap expectations name the new pathway sources. The d02 land-eligibility
  sentence ("complying development may be carried out on the land") stays: it is still true of the land, and the trap
  now also tests that land eligibility does not override a stated DA. d03's tower DA number stays a base-building trap.
- The certifier stays (CC, principal certifier, OC); the BCA consultant adds the existing building's fire safety
  against any upgrade the consent requires (verified: false).
- `flag.community`, `flag.panel` and `flag.cl46` stay `no`. The d01 community sentence is the base's, unchanged
  (review finding 2); determination under delegation, no clause 4.6 variation. "public exhibition" is kept out of read documents because it fires the
  community trigger.

## Traps and misses

As the base (including `hdr.scale.tenancies` and `det.type_of_construction` keyed `not_read`).
`proj.target_construction_start` stays only in the d04 email. d20 has label lines, so its number and date
should be read.

## Known extra value (unscored)

`flag.architect_lodges` is not keyed. Its lodge pattern fires in d01 ("consent to lodge"), d15 ("DA lodgement"),
d19 and d20, and d19 says plainly that the town planner lodges the application, so the right value is `no` (source
d19). `validate_expected.py` rejects it here because the base does not key it and it is not one of the da-pathway
trigger's profile keys. A reader that sets it to `yes` is wrong, but this scenario does not score it.

## Key changes after run `20261107-t03-t04-t06` (7 November 2026)

- `hdr.cond.quality_level`: `read_by_design: false` with `not_read`. The field applies to residential only, so the
  app never asks it on an office fit-out.
- `hdr.cond.operational_constraints` accepts `partial_occupation` or `live_environment` (owner ruling R19: an empty
  floor in an occupied tower is both).
- The certifier and BCA RFP keys in the T03 shape were re-keyed for F9 (see T03 `DOMAIN.md`).
