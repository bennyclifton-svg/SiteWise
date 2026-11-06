# ST04-cdc-industrial-code notes

The base Ironstone Circuit warehouse approved by a complying development certificate under the Industrial and
Business Buildings Code (Codes SEPP Part 5A, verified: false) that the registered certifier determines, instead of a
DA. Only `hdr.cond.planning` differs from the base (da -> cdc). 16 documents in 4 waves.

## Changes from the base

- d01 carries the trigger's evidence ("Complying development certificate under Part 5A Industrial and Business
  Buildings Code of the Codes SEPP"; "meets the code's setbacks, height and landscaping standards, so the certifier
  can determine it") in place of the DA and code-height sentences. The panel, clause 4.6 and community denials are
  reworded for complying development, so those flags keep a read source after d13 goes.
- d05 architect: the DA documentation phase becomes CDC documentation; the development manager lodges the CDC
  (architect_lodges stays no); d05 is now a second source for planning. d06, d07, d10, d12, d14: "for the DA" becomes
  "for the CDC"; d10's RFI responses go to the certifier. Amounts unchanged. d16 Rev B note "Amended for CDC".
- d04 email is about CDC timing. `proj.target_da_lodgement` keeps the base date (2026-06-30) because the checker lets
  only `hdr.cond.planning` move; under a CDC it is the CDC lodgement target. Not read by design anyway (email).
- Removed: d09 town planner fee proposal (the trigger removes consultant.town_planner), d13 SEE (no SEE under a CDC),
  d18 town planner invoice (no planner engaged).
- Checked: no remaining document says "DA", "development application" or "development consent". d02, d03, d08, d11,
  d15, d17 and d19 were already neutral. d02's "complying development may be carried out on the land" now agrees
  with the pathway; its trap expectation was written neutrally, so d02 is unchanged.

## What the scorer should see

Shape trigger `cdc-industrial-code`: town planner removed from procurement and RFPs; DA, notification,
determination, pre-DA and CC rows removed; CDC preparation and certifier determination added; DA fee and CC cost
lines replaced by the CDC fee and contributions. The app still suggests the town planner through baselines[0]
(shape: procurement built_in no), so a suggestion row may remain although no proposal creates the package.

Whether a 12 m clear-height warehouse meets the code's height standard is `verified: false`; the documents state
that it does, and the variant tests the app's reading, not the lawfulness of the route.

## Review (2026-11-07) applied

As ST04-base (dock doors, tfnsw wording in d10, fire_specialist_systems wording, office share line, regulated
design not_read); d13 is absent here, so the dock count rests on d01.

## Missed by design

As the base: `proj.target_da_lodgement` and `proj.target_construction_start` (d04 email), `hdr.scale.hardstand_sqm`
and `hdr.scale.office_percent` (no triggers), `proj.regulated_design` (no extraction, keyed `not_read`), d02 and d07 table identity.

## Traps

As the base, less d09 `planner-lodges` and the d13 traps (removed). `det.flood_hazard` now rests on d02 alone, and
`flag.panel`, `flag.cl46` and `flag.community` on d01 alone.

## Render review (2026-11-07) applied

- `det.site_class` (M) and `det.saline_soil` (true) keyed from d11 ("classified as site class M (moderately reactive)
  under AS 2870", "The soils are moderately saline"), and added to d11's carries in the manifest. **Owner:** neither
  key is in `types/T04-warehouse/shape.yaml`'s profile; the checker accepts them, the shape was not edited.
- `fees.json` firm phone and address now follow the letterheads written in the base corpus (d05-d10, and the d17-d19
  invoices); the variants copy those documents, so their keys changed the same way.
- Base corpus wording (no key effect): d10 exclusion "Parking demand surveys at comparable sites"; d13 heading
  "## Stormwater" and "reports submitted"; d12 perimeter access with the office-frontage performance solution; d15/d16
  "Two fire water tanks and pump room".
