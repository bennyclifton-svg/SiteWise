# ST12-base notes

The Whitfield granny flat: a new detached, single-storey, 2-bedroom, 60 m2 secondary dwelling in the backyard of an
occupied house at 18 Tuckeroo Crescent, Gymea (Sutherland Shire, R2, 600 m2). CDC from a registered certifier, an
owner-engaged building designer and small consultant team, lump-sum quotes from three builders on the HIA contract;
budget $210,000 ex GST ($231,000 inc GST, below the $250,000 long service levy threshold, so no levy: stated in d10).
17 documents in 4 waves; every T12 trigger denied or absent.

## Traps

- d03 (adversarial) `sewer-main-in-street`: "sewer main in the street" and "Water main in Tuckeroo Crescent" match
  authority_asset; stays no.
  `existing-building-retained`: "The existing building on the lot ... is retained" matches existing_building; stays
  false. `trees-listed-not-affected`: trees listed, none affected.
- d07 (messy, GST-inclusive) `osd-negated`: "exempt from on-site detention" and "no existing on-site detention".
- d11 (adversarial) `excavate-to-depth` (basement stays no) and `excavation-near-boundary` (edge beams 900 mm
  from the fence; excavation_adjoining stays no).
- d12 (adversarial) `existing-building-main-house`, `no-demolition`, `no-separating-wall`, `not-dual-occupancy`,
  `garage-zero` (DOMAIN.md §10 traps 1, 3, 4, 7, 8).
- d16 invoice: $1,700 + $170 stated as $1,780 (should be $1,870): `arithmetic`.
- Incidental, not declared: d05 "Stage 1 ... Stage 3" fee stages match the `staged` trigger (key stays no from d01).
  "lodge" in d05 and the CDC wording in d10 fire `architect_lodges`, which is yes. d10's "neighbour notices" has no
  tree word nearby, so no arborist flag.
- Every trigger negation used is the shape's own wording or a light paraphrase: detached and free-standing,
  CDC sought, no Sydney Water assets / main in the street, exempt from OSD, no trees affected, no demolition, clear
  lawn, 3 m driveway with truck access, maximum demand adequate, built on site in timber frame, generally level,
  not bush fire prone, not flood controlled, "about 7 m clear of the main house and at least 900 mm from every
  boundary".

## Missed by design

- `proj.target_construction_start` (2026-09-01) is only in the owner's email d04 (correspondence is not read).
- d02 number (PC-2026-04417) and date (2026-02-09) and d07 number (GSC-P-0412) and date (2026-03-10) sit in
  Markdown tables, so the register is expected to miss them; the key keeps the true values. d16 is a table invoice;
  the invoice reader is Jev, so no register loss.
- d14 supersedes d13 (A-101 Rev A -> Rev B); the app cannot write the link today (DISCOVERY.md §4.5).
- `proj.regulated_design` has no reading in the app (project_details.yaml: no extraction); the d05 sentence is kept
  for realism. The checker derives `read_by_design` true from d05's kind, so the key says true; the scorer should
  treat it as missed by design.
- `hdr.scale.garage_spaces` = 0: d12 states "Garage spaces: 0" so the integer candidate exists; if Jev reads "no
  garage" as absence, the value is missed. The brief's "no garage or car space" has no number, so it gives no
  candidate.

## Risks to watch (not traps)

- `hdr.subclass`: the app has no secondary-dwelling subclass (shape gap); "house" is the expected pick.
- d05 discipline: the issuer is a building designer; the title says "architectural design", which matches the
  architect alias. Discipline `none` would be a filing miss, not a key error.
- d03 and d12 describe two dwellings; the main house's storeys, bedrooms, bathrooms and floor area are never stated,
  so the scale fields have only the flat's values (DOMAIN.md §10 trap 2).

## Shape comments

- Applied after review (6 Nov, ruling 2): `det.site_class: M` is now in the T12 shape's base profile (R10 by
  analogy, pending owner confirmation); d11 states it ("Site classification: Class M in accordance with AS 2870").
- `hdr.scale.garage_spaces: 0` kept with `read_by_design: true` (review ruling 1).
- Extra keys beyond the shape: `fact.contract_basis` (lump_sum) and `flag.arborist_pruning` (no), as in ST05.

## Review applied (6 Nov)

1. d03 `existing-building-retained` expect now says hdr.work_type stays new (header-routed passage).
2. d03 access passage opens with "Construction access is constrained: the only way to the backyard is the side
   driveway past the occupied house, through a double gate."; the driveway negation is kept.
3. da-pathway d18: the neighbour's garage is now a shed (no stray garage candidate).
4. da-pathway d01, d12, d18: "closer to the rear boundary than the complying development standards allow" precedes
   the shape phrase.
5. da-pathway geometry: about 8.5 m (not 9 m) from the main house.
