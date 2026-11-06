# ST10-base notes

The Delaney house: the 1962 brick-veneer and fibro house, detached garage and shed at 22 Marrakai Close, Caringbah
South (Sutherland Shire, R2, 620 m2) are demolished, with bonded asbestos removed under licence, and replaced by an
architect-designed two-storey, four-bedroom house with a double garage (302 m2 GFA). One CDC covers demolition and
the new house; select tender, HIA lump-sum contract including demolition; budget $1,520,000 ex GST. 18 documents in
4 waves; every T10 trigger denied or absent.

## Traps

- d02 `cdc-land-eligible` (messy): land eligibility for complying development is not the chosen pathway.
- d03 `sewer-main-in-street` (trap 11), `trees-listed-not-affected`, `no-existing-osd` (trap 13).
- d07 `osd-negated` (messy, GST-inclusive): both OSD flags stay no.
- d11 `existing-storeys`: "the existing building is a single-storey dwelling" in a read report; storeys stays 2.
  `not-contaminated-land` (trap 5); `asbestos-negations` (friable, LFAI, soil and fence negations).
- d12 `old-footings-boundary` (trap 12): excavating old footings along the boundary is not excavation_adjoining.
- d13 `gas-abolished` (trap 6) and `demolish-near-trees` (trap 8).
- d17 invoice `abn-transposed`: ABN 27 365 699 632 fails the checksum (the firm's ABN is 27 365 969 632); the
  ledger flags `abn_invalid`.
- Incidental, not declared: d05 "Stage 1 … Stage 4" fee stages match the `staged` trigger; the key keeps
  `flag.staged` no from d01. d05 "lodging" fires `architect_lodges`, which is yes.
- d14/d15: site plan A-101 Rev A and Rev B; d15 supersedes d14 (the app cannot link it today, DISCOVERY.md §4.5).
- Phrasing chosen for the readers: "existing building" (not only "existing dwelling") for `det.existing_building`;
  "built in 1962" for the year; "Class 1a … Class 10a" for the class; "building contract" for
  `fact.contract_form`; "face brickwork veneer" so `structure.masonry` can be added (trap 19). Header facts sit
  under `## Project description` and `## Contract and procurement` in d01.

## Missed by design

- `proj.target_construction_start` (2027-02-01) is only in the d04 email (correspondence is not read).
- `proj.regulated_design` ("no") is stated only in the architect's reply in d04, so `read_by_design` is false. The
  app has no reading for it in any case (no extraction in `project_details.yaml`).
- d02 register number (PC-2026-00318) and date (2026-02-09) sit in a Markdown table, and `Certificate no` is not a
  number label, so both are expected to be missed; the key keeps the true values. The same holds for the d07
  number and date (DC-P-0219, 2026-03-11). d18 is an invoice in a table; the invoice reader is Jev, so no register
  loss.
- Drawing values (site area, garage on d14/d15) are also stated in read documents, so nothing else is lost.

## Review (2026-11-06) applied

- d07 public-road sentence now says the civil scope has no public road works and the builder rebuilds the crossing
  under council's approval (key stays no). d12 groundwater sentence now reads "to refusal on sandstone at about
  2.6 m". The da-pathway community wording and the separate-demolition staged negation are in those notes.

## Shape comments

- Ruling 1 applied: `det.site_class: M` (d12, "Site classification: Class M in accordance with AS 2870") and
  `flag.arborist_offsite_trees: "no"` (d13, "No neighbouring or street trees are affected by the works") are now
  in the T10 shape's base profile. The owner confirmed this on 6 November 2026 (R10 extended to T10 and T12).
- `proj.regulated_design` is in the shape's base profile but has no reading in the app; suggest the shape note
  that it is never read.

## Key correction after run `20261106-t10-t12` (6 November 2026)

- `proj.lga` also accepts "Sutherland". The ST12 keys for the same council already accept it, as the ST11
  keys accept "Parramatta" for the City of Parramatta: the keys accept the area name without its "Shire" or
  "City of". The run read "Sutherland" from "Sutherland Shire Council". Whether the app should keep "Shire"
  is noted in FINDINGS (key fixes for this run). The owner confirmed this convention on 6 November 2026.
