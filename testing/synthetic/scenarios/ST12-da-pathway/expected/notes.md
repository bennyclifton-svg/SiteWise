# ST12-da-pathway notes

The base Whitfield granny flat, but the owners place it 1.5 m from the rear fence to keep a large lawn between the
two dwellings. That exceeds the CDC rear setback standard, so the designer lodges a DA with Sutherland Shire Council
(with an SEE) and the certifier issues a construction certificate after consent. Only `hdr.cond.planning` changes
(cdc -> da). The evidence used is the shape's third phrase ("the flat exceeds the CDC rear setback standard, so we
will lodge a DA with council"); the other two would change `hdr.scale.site_sqm` or `det.heritage_status`, which the
trigger may not change. No setback figure from the Housing SEPP is stated anywhere (verified: false in DOMAIN.md §5).

## What changed

- Every CDC mention in a base document was searched ("CDC", "complying development", "certifier ... determine"):
  d01, d04, d05, d06, d07, d08, d09, d10, d12, d13 (none, but setbacks change), d14. All now name the DA or the
  construction certificate. d02 keeps the council wording "complying development may be carried out on the land"
  as the declared `cdc-land-eligible` trap. d03, d11 and the three invoices never named the pathway.
- d02 is copied byte for byte from the base; only its manifest entry changes (new trap `cdc-land-eligible`).
- d10: the certifier's proposal becomes CC, principal certifier and OC ($4,300); the certifier no longer notifies
  neighbours (council does, under its community participation plan).
- d05: Stage 2 rises to $5,600 for the DA set and SEE; total $10,400.
- d18 added: the SEE, by the designer (a town planner stays optional; no planner document).
- Geometry: the flat moves from a 3.0 m to a 1.5 m rear setback, so it is about 8.5 m (not 7 m) from the main house;
  still at least 900 mm from every boundary, so `close-to-main-house` stays denied.

## Traps

As the base, plus d02 `cdc-land-eligible` (planning stays da). d12's planning sentences no longer say "as
complying development".

## Review applied (6 Nov)

Findings 1 and 2 (d03) apply here as in the base; findings 3 to 5 are da-pathway only. See ST12-base
notes.

## Missed by design

As the base: d04 target start, d02/d07 table numbers and dates, d14 -> d13 supersession, `proj.regulated_design`,
and possibly `hdr.scale.garage_spaces`.
