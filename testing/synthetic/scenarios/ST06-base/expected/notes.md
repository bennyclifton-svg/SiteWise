# ST06-base notes

41 Callowhill Street, North Sydney (North Sydney Council): a 1988 6-storey Class 5 office building over a Class 7a
basement car park, sprinkler protected throughout, single owner (Corvina Property Holdings). The owner replaces the
single electric pump with electric duty, diesel standby and jockey pumps and a 150 kL tank in the enlarged basement
pump room, and fits out the vacant level 3: two CDCs, two AS 4000 lump sum contracts. 20 documents in 4 waves; every
T06 trigger denied or absent.

## Where this departs from brief 03, and why

- **Fee proposals.** Civil and geotechnical engineers have no role (no excavation, ground or stormwater work), and
  there is no architect (DOMAIN.md §4). The shape's team proposes instead: fire services (d05, dropped on
  `row:consultant.fire_services`), structural (d06), electrical (d07, GST-inclusive), mechanical (d08), certifier
  (d09), interior designer (d10) and hazmat (d11).
- **Wave 1** is what the owner holds: the brief (d01), the s10.7 certificate (d02), the 2025 AFSS and fire safety
  schedule (d03) and the Sydney Water statement of available pressure and flow (d04). No site survey.
- **Budget.** `proj.budget_stated` is the whole job: $2,700,000 ex GST (fire services upgrade $1,200,000; level 3
  fit-out $1,500,000), d01.

## Traps

- d01 `rfp-cued-as-proposal`: "submit a fee proposal" must not open a submission.
- d01 `pump-room-in-basement`: "fire pump room in the basement"; flag.basement stays no.
- d02 `acid-sulfate-class-not-ncc`: "class 5" acid sulfate soils; det.ncc_class stays ["5","7a"].
- d03 `fire-stairs-implied-stair`: "fire stairs" must not add stairs-and-ramps scope or a 13.x cost line.
- d03 `afss-kind`: the AFSS must file as a read kind (certificate or report), not correspondence.
- d04 `sydney-water-asset` and `backflow-contamination`: authority_asset stays no, contaminated land false.
- d05 `hydraulic-and-fire-name`: a "Hydraulic and Fire Services" firm; the row drop holds fire_services.
- d05 `fee-stages-not-staged-works`: "Stage 1" to "Stage 8" fee stages; flag.staged stays no.
- d06 `structural-fire-tank`: "structural engineer will confirm the slab for the fire water storage tank";
  flag.fire_specialist_systems stays no.
- d12 `pump-room-flooding`, `stage-1-cut-over`, `fire-storage-not-special-hazard`, `sprinkler-alias-trade`.
- d19 `gst-half-rate`: GST $375 on $7,500 (`gst_rate`).
- d07 states GST-inclusive figures ($33,000 inc = $30,000 ex): `gst_inc_normalised`.
- d01 `building-over-basement`: "a 6-storey office building over a single basement car park level" hits the
  authority_asset trigger `build(?:ing)? over`; flag.authority_asset stays no (d12).
- d12 `modular-fire-tank`: "150 kL modular fire water tank" hits `modular[^.]{0,100}fire`; flag.fire_specialist_systems
  stays no (d12).
- Incidental `basement` hits in d03, d09, d12, d16 and d17; flag.basement stays no (d06, d13).

Every other trigger's negation is in a read document, in the shape's wording: fire safety order, tenant-procured
fit-out, interim occupation, single ownership, pump-tank-only (d01); existing fire-engineered building (d03);
tank-needs-consent (d09 and d12), DA fire upgrade (d09); performance solution, town main, hydrant, roof or crane
tank, electrical supply (d12); asbestos (d14); fit-out amenities (d10, d15); affected-part access (d15); heritage
(d02). `diesel-noise-exhaust` is denied in d12 with the shape's revised negation (exhaust through the existing basement
plant exhaust shaft; test run not audible in the tenancies).

## Missed by design

- `det.type_of_construction` (A): derived only, never read; blank while rule.ncc.type-of-construction and its table
  are draft (reason unreviewed). Table C2D2 gives A for Class 5 and 7a at rise 6. The key keeps the true value.
- `proj.regulated_design` ("no"): not read; the RFP derives it from det.ncc_class 5 and 7a. The key keeps the value
  (`stated_in` d09).
- d02 and d08: Markdown-table title blocks, so number and date are missed and titles come from the heading. d19's
  table is an invoice; its fields come from the invoice reading. Keys keep the true values.
- d04 may file as correspondence and go unread; `det.water_supply_source` is also stated in d12.
- d17 supersedes d16 (FS-101 Rev B over Rev A): the app cannot link it (DISCOVERY.md §4.5).
- Two CDCs: the profile holds one planning value (`cdc`); the second CDC is a shape gap.

## Review (REVIEW.md, FIX, 12 findings) applied 7 Nov 2026

All 12 findings applied; none disputed. Shape edits made with the review's agreement (noted in DOMAIN.md):

- `diesel-noise-exhaust` negation is now "The diesel exhaust discharges through the existing basement plant exhaust
  shaft, clear of tenancies, air intakes and neighbours; the weekly test run is not audible in the tenancies."
- `pump-tank-only` profile is `[hdr.cond.operational_constraints, proj.budget_stated]`; `tenant-procured-fitout`
  profile is `[proj.budget_stated]`. The operational-constraints key still accepts both values in all scenarios.
- d12 sentences for water supply ("town water main"), specialist fire systems and staging now match their triggers;
  d13's existing-building sentence says "the existing building's basement slab".

## Render review (RENDER_REVIEW.md, FIX, 11 findings) applied 7 Nov 2026

All 11 findings applied, none disputed. d13 retitled "Structural assessment: sprinkler tank location and pump room
slab" (manifest and document_register.json, also in ST06-pump-tank-only and ST06-tank-needs-consent); d13
groundwater wording, d01/d03/d09/d12/d14 content and the two new traps carried into all three manifests. d09's unkeyed
$450 inspection rate was deleted rather than keyed.

## Key changes after run `20261107-t03-t04-t06` (7 November 2026)

- `hdr.cond.quality_level`: `read_by_design: false` with `not_read`. The field applies to residential only, so the
  app never asks it on a commercial building.
- `hdr.cond.access_constraints` accepts `urban_constrained` or `restricted_hours` (owner ruling R19).
- The T06 shape's certifier RFP was re-keyed for F9, and `position.existing` now expects `app: absent` (see T06
  `DOMAIN.md`).
