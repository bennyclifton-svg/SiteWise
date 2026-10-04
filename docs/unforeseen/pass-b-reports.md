# Pass B reports: unforeseen-circumstance lists turned into knowledge records

Date: 4 October 2026. Each section is a cluster agent's report, followed by the lead's checks. All records are `status: draft` and await owner review. Row ids: batch 1 is `B1-xxxx` (`data/unforeseen/batch1.csv`), batch 2 is `B2-xxxx` (`data/unforeseen/batch2.csv`).

Where each row ended up: `knowledge/works/coverage/<batch>/<cluster>.yaml`.
Records: `knowledge/clusters/<cluster>/{failure_modes,unforeseen,interfaces,signals}.yaml`.

## Totals

| | Batch 1 | Batch 2 |
| - | - | - |
| Rows covered by a record | 967 | 946 |
| Duplicate of another row | 7 | 53 |
| Rejected: too vague | 18 | 0 |
| Rejected: out of scope | 8 | 1 |
| Distinct records the rows feed | 694 | 780 |

Knowledge base after both batches: 859 failure modes (from 172), 560 unforeseen conditions (from 0), 331 Jev signal questions, 144 interfaces. Checker: 0 errors, 0 warnings, 0 rows pending.

Lead's checks on every cluster merge: no existing record lost a source or changed wording (only sources appended); each ledger overlay lists exactly its own cluster's rows; no record cites a row from another cluster.

## Open items across all clusters

1. Jev detector questions ask whether a passage reports a defect; a second question ("does the specification require the prevention?") would make the records useful at tender stage. Owner decision.
2. Merge pass: cross-cluster overlaps (fire penetrations against services ducts), misplaced records (vehicle clearance in fire, crane runway in services), loose merges flagged below, `work_type` conditions with no defined options, generated contract wording and severities.
3. Sources: most batch 2 records cite only the owner's rows; named standards are in `notes` as unverified. Seed anchors where present are section-level.
4. Owner review of rejections and of the asbestos (pre-2004) and lead-paint (1970 vs 1990) year thresholds, which are placeholders until K3.

---

## Batch 1

### Fire (60 rows)

- 39 new failure modes (fire doors and openings, sprinklers and fire water, smoke control and alarms, egress and emergency lighting, access and barriers). Merged pairs: B1-0312/0362, 0302/0332, 0339/0343, 0319/0400, 0369/0376, 0372/0752, 0185/0287, 0304/0495.
- 6 existing failure modes extended: `fm.plastic-pipe-without-fire-collar` (B1-0301), `fm.sprinklers-share-ceiling-services` (0303), `fm.damper-access-maintained` (0314), `fm.brigade-access-obstructed` (0337), `fm.integrated-fire-test-failed` (0305; `fire-active.smoke-control` added to runs_on), `fm.suppression-hazard-changed` (0334, 0335).
- 1 new interface `if.sprinkler-pressure-test-before-ceilings` (sequences, hold point, B1-0329); `fm.step-free-threshold-fails-accessibility` linked on `if.accessible-threshold-weatherproofing`.
- 5 unforeseen conditions: street main below fire flow (B1-0365), authority requires water main upgrade (0333), dust sets off detectors (0340), performance solution rejected (0354), fire alarm replacement needs occupant shutdown (0686).
- 4 signals.
- Rejected: B1-0886 (site-shed fire) out of scope.
- Doubts: `fm.fire-pump-room-flood-exposure` and `fm.fire-tank-footprint-not-in-site-layout` are thin; B1-0400 and 0329 triaged as process but recorded as failure modes; `fm.fire-panel-cannot-integrate-with-new-systems` could be an unforeseen condition; `uc.performance-solution-rejected-by-authority` uses a broad predicate because `compliance_pathway` has no values; the agent re-dumped YAML files (lead verified nothing changed except intended records).
- Lead found a checker bug from this report (ledger overlays read as knowledge files); fixed on main in 98ca0b2.

### Delivery (125 rows)

- 58 unforeseen conditions covering 108 rows, attached to stages (investigation, design, approvals, procurement, construction, completion, defects) or the supply package kind. 27 signals.
- Rejected too vague (9): B1-0909, 0911, 0912, 0914, 0928, 0985, 0989, 0990, 0999. Out of scope (7): B1-0938, 0939, 0940, 0894, 0994, 0996, and 0925 (tender price exceeds budget; agent's addition). Duplicate: B1-0864 of B1-0046.
- Doubts: `when` uses `{det: work_type, any_of: [...]}`, but `work_type` has no defined options, so these conditions check nothing yet; heritage record conditions on `heritage_status`, hospital record on `patient_care_area`; weather, planning limits and change of use are deliberately broad single records; B1-0393, 0396, 0348, 0366 could move to fire.

### Envelope and interiors (202 rows)

- 99 new failure modes; 21 existing envelope failure modes extended with sources only.
- 30 unforeseen conditions: 9 physical (heritage wall opening, corroded facade ties, asbestos cladding, lead paint, combustible panels, roof-space fire separation, ceiling-cavity hazards, combustible cladding, damage behind cladding), 21 process/supply/authority.
- 18 signals; reuses `sig.hazardous-materials-survey-stated`.
- Covered by other clusters: B1-0316 (`fm.fire-facade-slab-edge`), B1-0361 (`fm.party-wall-stops-in-roof`).
- Rejected too vague (agent's own call, cosmetic): B1-0253, 0715, 0728, 0747, 0764.
- Doubts: no seed anchors, only row sources; asbestos (<2004) and lead paint (<1990) thresholds are placeholders; loose merges `thermal-break-missing`, `glass-weight-exceeds-frame-capacity`, `colour-or-dye-batch-variation`.

### Electrical, comms and lifts (124 rows)

- 61 new failure modes (lifts 11, lighting 8, comms/security 10, cabling/switchboards/testing/temporary power ~22, standby power/battery/solar/fire ~10). Consolidations: `fm.electrical-verification-test-failed` (B1-0624, 0629, 0690), `fm.interference-disturbs-sensitive-equipment` (0632, 0692, 0693), `fm.protective-device-nuisance-tripping` (0628, 0630).
- 12 existing failure modes extended (B1-0109, 0559, 0562, 0564, 0604, 0670, 0615, 0608, 0636, 0651, 0658, 0700, 0535, 0536).
- 2 new interfaces: `if.lift-pit-waterproofing`, `if.cable-routes-share-services-space`.
- 26 unforeseen conditions (13 existing-condition: existing wiring, aluminium wiring, switchboard asbestos, supply capacity, overhead lines, full risers, legacy dimmers, earth stake in rock; 13 process).
- 15 signals.
- Covered by other clusters: B1-0587 (`fm.fire-warning-acoustic-environment`), B1-0749 (`fm.envelope-joinery-service-positions`).
- Duplicate: B1-0682 of B1-0004 (holds: B1-0004 became a structure record).
- Doubts: merges B1-0679/0684 (supply too small) and B1-0808/0899 (overhead lines); lift and comms seed anchors are nearest headings only.

### Services: hydraulic and mechanical (180 rows)

- 93 new failure modes; 12 extended (`fm.exhaust-penetrates-envelope`, `fm.exhaust-intake-separation`, `fm.kitchen-exhaust-makeup-air`, `fm.cooling-tower-stagnant-dead-leg`, `fm.hydraulic-rough-in-before-linings`, `fm.roof-drainage-waterproofing`, `fm.plant-loads-structure`, `fm.services-plant-maintenance-access`, `fm.services-share-ceiling-space`, `fm.process-vendor-utility-mismatch`, `fm.plant-noise-vibration`, `fm.occupied-stage-services-isolated`).
- 1 new interface `if.tenant-fitout-base-building-hvac` (tenant ducting, supplementary cooling, extended hours and outdoor air drawing on base-building plant).
- Tenant fit-out records: `fm.tenant-fitout-hvac-exceeds-base-capacity` (B1-0533), `fm.tenant-extended-hours-cooling-not-provided` (0534), `uc.tenant-fitout-changes-after-approval-affect-base-services` (0772), `fm.outdoor-air-based-on-lower-occupancy-than-intended` (0515). Thermal performance via insulation, stratification and pool-hall humidity records.
- 28 unforeseen conditions (19 process/authority, 9 existing building: deteriorated pipework, lead, undersized drainage, roof drainage, HVAC capacity, plant room fit, asbestos in ductwork, undocumented services, drains under new footprint). 28 signals.
- Duplicates: B1-0519, 0600 of B1-0391. Rejected too vague: B1-0924 (agent's own call; confirm).
- Covered by other clusters: B1-0581, B1-0599 (envelope records).
- Doubts: noise rows added to `fm.plant-noise-vibration`; some uc read like hold points; duct-through-fire-wall rows are new failure modes here rather than joining fire's penetration records.

### Structure: site, substructure and structure (309 rows)

- 112 new failure modes; 10 extended (`fm.excavation-affects-adjoining-foundations`, `fm.groundwater-loads-basement`, `fm.retaining-drainage-protects-wall`, `fm.temporary-works-load-new-concrete`, `fm.termite-barrier-service-penetrations`, `fm.drainage-moisture-affects-footings`, `fm.crane-outrigger-ground-bearing`, `fm.services-modify-structural-framing`, `fm.podium-soil-loads-structure`, `fm.podium-landscape-waterproofing`).
- 75 unforeseen conditions; 30 signals.
- Rejected too vague: B1-0107, 0108, 0905. Duplicates: B1-0810 of 0043, 0965 of 0161, 0827 of 0046.
- Covered by other clusters: B1-0113, 0140 (`fm.plant-loads-structure`), 0181 (`fm.pv-array-loads-roof`), 0206 (`fm.envelope-envelope-thermal-bridges`), 0210 (`fm.envelope-recurring-efflorescence`).
- Doubts: thematic seed anchors only; loose groups (slab flatness, equipment loads); B1-0169, 0860, 0383 placed in uc records outside their kind; B1-0837 (snow load) attached to `structure`.

---

## Batch 2

### Delivery (14 rows)

- 7 new unforeseen conditions: crane permit or airspace limits (B2-0763, 0782); road-opening permit hours too short (0764); noisy work refused near neighbour (0765); existing building or addition has no approval (0771, 0861); strata consent withheld (0791); latent-conditions clause does not cover asbestos the tender said was absent (0792); shutdown window shorter than changeover (0827, 0855).
- 3 extended: `uc.occupation-certificate-withheld` (0772), `uc.occupied-building-works-restrict-hours-and-methods` (0825), `uc.occupancy-window-constrains-programme` (0882, 0884).
- 5 signals. No rejections.
- Doubts: B2-0791 and 0792 have no seed anchor; B2-0765 is a new record rather than widening the pours record.

### Fire (113 rows)

- 33 new failure modes; 37 new unforeseen conditions, each with its own signal (37 signals).
- 11 extended: `fm.exit-sign-sightline-obstructed`, `fm.hose-reel-coverage-gap`, `fm.sprinkler-spray-blocked-by-fixed-construction`, `fm.suppression-hazard-changed`, `fm.brigade-access-obstructed`, `fm.travel-distance-to-exit-reported-exceeded`, `fm.accessible-room-plan-cannot-achieve-clearances`, `fm.tactile-indicators-clash-with-stair-geometry`, `fm.fit-out-narrows-egress-path`, `uc.street-main-below-fire-flow-demand`, `uc.fire-alarm-replacement-needs-occupant-shutdown`.
- Duplicates (15 tail restatements): B2-0948, 0954, 0984, 0985, 0987, 0990–0997, 0999, 1000. Out of scope: B2-0698 (garage-door motor noise).
- Covered by other clusters: B2-0171, 0228 (`fm.envelope-fire-rated-wall-stops-at-ceiling`), 0556 (`fm.mezzanine-addition-egress-not-reviewed`), 0871 (`uc.new-plant-does-not-fit-old-plant-room-or-route`).
- Doubts: `fm.vehicle-clearance-lost-under-structural-beam` (B2-0916, 0917) belongs in structure or site; B2-0559/0770 merge; broad `existing_building` predicates where no change-of-use or performance-solution determinant exists.

### Structure (239 rows)

- 73 new unforeseen conditions (post-tension and prestress clashes, concealed beams, stormwater connection points, soil permeability, contamination and waste classification, neighbour and authority constraints); 45 new failure modes (pre-pour inspection, waterstop, stressing hold points); 18 signals.
- About 40 extended (13 fm, 27 uc), e.g. `fm.coring-or-slab-opening-cuts-reinforcement-or-tendon` (B2-0001, 0016), `uc.existing-wall-removed-is-loadbearing` (0003, 0834, 0845, 0057), `uc.unmapped-service-or-main-in-footing-line-or-excavation` (six rows).
- Duplicates: B2-0148, 0946 of 0118; 0760 of 0628; 0766 of 0597; 0965 of 0155; 0966 of 0154; 0967 of 0801.
- Covered by other clusters: B2-0631 (`fm.envelope-external-levels-moisture-bridge`), 0860 (`fm.bund-volume-below-stored-volume`).
- Doubts: dataset-row sources only; contract text is one default per category; borderline extensions B2-0033, 0057, 0615, 0599, 0578; several triage system ids corrected from row meaning (e.g. B2-0023, 0703).

### Electrical, comms and lifts (172 rows)

- 48 new failure modes; 53 new unforeseen conditions (lifts 10, existing electrical ~30, comms and security 11); 22 signals.
- Extended: 25 failure modes and 9 unforeseen conditions (e.g. `fm.switchboard-in-wet-area`, `fm.door-hardware-incompatible-with-access-control`, `uc.existing-risers-and-conduits-full`).
- Duplicates (12 tail rows): 0949→0204, 0950→0439, 0951→0309, 0952→0302, 0953→0305, 0955→0323, 0978→0742, 0982→0351, 0983→0307, 0986→0202, 0988→0187, 0998→0207.
- Covered by another cluster: B2-0362 (`fm.coring-or-slab-opening-cuts-reinforcement-or-tendon`).
- Doubts: merges 0333/0354/0370 (in-ground supply assets), 0347/0348 (fault rating), 0307/0349/0375 (discrimination), 0472/0480 (PCBs); 0219 and 0451 loosely extend `fm.emergency-power-supplies-lifts`; some signals loose fits.
- Reported that agents overwrote each other's scratch files; lead verified no record cites another cluster's row.

### Envelope and interiors (203 rows)

- 65 new failure modes; 65 new unforeseen conditions; 23 signals (`sig.envelope-*`).
- 40 extended (33 fm, 7 uc), e.g. shower-screen, niche and door-jamb membrane cuts (B2-0120, 0145, 0149) into the later-trade membrane puncture record.
- Duplicates: B2-0943 of 0072, 0944 of 0073, 0974 of 0085, 0975 of 0096.
- Covered by other clusters: B2-0086, 0636, 0637 (structure records), 0583 (`uc.existing-roof-drainage-short-for-extension`).
- Doubts: merges that could be split (0471/0473/0490 ceiling-void dust; 0475/0484/0485 biological contamination; 0849/0874 bonded tile or stone; 0560/0839 energy provisions; 0761/0875 heritage interior; 0088/0642/0848 coating on chalking surface; 0664/0671 piece too large for lift or stair); B2-0570 covered by a failure mode (kind changes); category overrides B2-0159, 0849, 0812, 0813; generated contract, severity and de-risk kind need review.

### Services: hydraulic and mechanical (259 rows)

- Landed on main: 92 new failure modes, 102 new unforeseen conditions, 102 signals (the agent's two reports gave slightly different counts; these are the merged figures).
- Extended: 22 failure modes and 3 unforeseen conditions (`uc.plant-noise-to-neighbours-over-limit`, `uc.new-plant-does-not-fit-old-plant-room-or-route`, `uc.existing-services-not-as-documented`).
- Duplicates (15): B2-0913→0167, 0936→0192, 0947→0117, 0956→0234, 0957→0231, 0958→0235, 0959→0247, 0961→0377, 0962→0383, 0963→0384, 0964→0408, 0979→0734, 0980→0413, 0981→0428, 0691→0416.
- Covered by other clusters: B2-0401 (`fm.envelope-grab-rail-backing-before-lining`), 0291 (`fm.envelope-louvre-free-area-restricts-airflow`), 0271 (`fm.lift-shaft-vent-blocked`; loosest fit).
- Doubts: dataset-row sources only; `works` actions mapped mechanically from work type; contract text one default per category; stretched extensions (B2-0416, 0300, 0516, 0506, 0527, 0528, 0539, 0408, 0409, 0438); B2-0854 (crane runway) and 0870 (oil tank) barely services; B2-0376 absorbs design row 0960; B2-0270 not treated as covered by fire's stair-pressurisation record.
