# ST09-base notes

Kurrajong Apartments, 41 Kurrajong Parade, Hurstville (Georges River): a fully occupied 6-storey, 42-unit Class 2
strata building constructed in 2008, out of warranty. The owners corporation renews all 42 balcony membranes like
for like (exempt, DBP Act applies), through a remedial building consultant and one AS 4000 lump-sum contract;
order of cost $1,680,000 ex GST. 16 documents in 4 waves; every T09 trigger denied or absent.

## Where this departs from brief 03, and why

- **Fee proposals.** The brief's architect, civil and geotechnical engineers are never needed for this job
  (DOMAIN.md §4, owner ruling R2). The base has the shape's two consultant packages instead: two competing
  remedial building consultants (d05, and d06 dropped on the `consultant.building` row) and the remedial
  structural engineer (d07). Three proposals, two packages.
- **Wave 1** is what an owners corporation holds before design: the strata manager's consultant brief (d01), a
  s10.7 certificate (d02), the strata manager's building information and leak history (d03) and a committee email
  (d04). There is no survey on a remediation job.
- **Wave 3** has the defect report, structural assessment, remedial specification, conditions of tender and one
  drawing at two revisions (RW-101 A and B).

## Traps

- d01 `rfp-cued-as-proposal`: the brief says "submit a fee proposal"; `proposals.Cue` matches that text.
- d03 `bathroom-below` and `bedroom-sill`: implied scope fires in code on "bathroom" and "bedroom" (DOMAIN.md §10).
- d05 `fee-stages-not-staged-works`: "Stage 1" to "Stage 4" fee stages match the staged trigger.
- d07 `fire-stairs`: "the fire stairs" implies stairs and ramps.
- d08 `remediation-not-contamination` and `flood-test`; d09 `flood-test-hold-point` and `strip-out`.
- d13 `schedule-of-rates`: lump sum with a schedule of rates for provisional quantities only.
- d15 invoice `abn-typo`: ABN printed 77 650 369 250, fails the checksum (`abn_invalid`).
- d06 (GST-inclusive, $99,000 inc = $90,000 ex): `gst_inc_normalised`; F3 suggests the app keeps the inc figure.
- d02 `cdc-eligible-land`: "Complying development may be carried out on the land"; planning stays exempt.
- d11/d12: RW-101 Rev A and Rev B; d12 supersedes d11 (the app cannot link it today, DISCOVERY.md §4.5).

## Missed by design

- `proj.target_construction_start` (2027-02-01) is only in the committee email d04 (correspondence is not read).
- d02, d06 and d15 have Markdown-table title blocks: the d02 and d06 number and date are expected to be missed
  (keys keep the true values). d02 `Certificate no` is not a number label either.
- `proj.regulated_design` (stated in d05, d08, d09): never read. The detail has no extraction
  (`internal/knowledge/details.go` compileReading). The app shows it unknown (not derived "no", because the project
  has a Class 2 part) until the user records it. Keyed `read_by_design: false` with `sources: []` and the stating
  documents under `stated_in`, because the checker infers `read_by_design` from source kinds.
- `proj.lot` (d02 "Lots 1 to 42 in Strata Plan 912345"): `lotRe` has no lot range form, so there is no candidate.
  The wording is the realistic strata form and is kept. Keyed the same way as `proj.regulated_design`.
- `proj.client`: the key holds the legal name "The Owners - Strata Plan No. 912345", the form an SSMA owners
  corporation is named in. `clientCandidates` (`internal/profile/details.go`) stops at " - ", so the app is
  expected to offer only "The Owners". This is a probable app defect, not a scenario fix.

## Wording chosen to meet the app's triggers

- `hdr.scale.units`: "42 apartments" (d01, d03, d08). "42 residential units" matches neither the units trigger nor
  `unitCountRe`.
- `det.existing_building_year`: "constructed in 2008" (the trigger needs built in / constructed in / year built).
- `proj.budget_stated`: d08 recommends that the owners corporation "adopts a construction budget of $1,680,000
  excluding GST", rather than giving an order-of-cost estimate, which Jev may not take as a budget.
- `fact.contract_basis`: "lump sum contract" without the hyphen (the trigger is the literal `lump sum`).
- `flag.staged`: "There will be no staged delivery".
- `det.rise_in_storeys` is only asked on "rise in storeys"; d08 states it that way.

## Shape

Updated with the orchestrator's authority (4 Nov) after review: units phrasing "<n> apartments"; "constructed in
<year>" for the year and the hazmat evidence; the hazmat negation is now "The building was constructed in 2008."
(the old "Construction completed in 2012" missed the trigger and contradicted the base); the staged negation now
starts "There will be no staged delivery"; `proj.client` has no "(fictional)"; the podium trigger's `profile` now
lists `det.ncc_class` and `proj.budget_stated`.
