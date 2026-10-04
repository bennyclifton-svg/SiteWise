# K4 owner review packet

Date: 5 October 2026. Package WP-K4 (`docs/plans/2026-10-04-next-wave-agent-work-packages.md`). This packet supports your review of the unforeseen-conditions work (K4 Pass A and B, batches 1 and 2). It changes no knowledge. The agents' own reports are in `docs/unforeseen/pass-b-reports.md`.

## 1. Did K4 change existing failure modes? (PRG L260: "Existing `fm.*` records stay as they are")

I compared every failure mode that existed at `02094db`, before K4, with `main`.

| | Count |
| - | - |
| Failure modes before K4 | 172 |
| Removed | 0 |
| Changed | 72 |
| … sources appended only (wording, detector and fields unchanged) | 71 |
| … changed in substance | 1 |

**The one substantive change: `fm.integrated-fire-test-failed` (fire).**

- The detector wording is unchanged.
- Its `runs_on` gained `fire-active.smoke-control`, so the existing question is now also asked on smoke-control passages. Before, it was asked on `fire-active.fire-control` only.
- A note was added: "B1-0305: a smoke control system failing its commissioning test is the same defect; smoke-control passages are included."

**Your call:** accept, or revert to `fire-control` only and record B1-0305 as its own record.

Records with sources appended only (each now also cites the owner rows that support it):

| Cluster | Count | Records |
| - | - | - |
| electrical-comms | 15 | `fm.accessible-lift-landing-path`, `fm.battery-ventilation-environment`, `fm.carrier-entry-civil-route`, `fm.downlights-insulation-interface`, `fm.electrical-rough-in-before-lining`, `fm.emergency-power-supplies-lifts`, `fm.fire-alarm-controls-lift-recall`, `fm.fire-alarm-controls-security-release`, `fm.ict-room-power-cooling`, `fm.lift-shaft-fire-separation`, `fm.lift-shaft-structural-dimensions`, `fm.power-data-cable-segregation`, `fm.pv-array-loads-roof`, `fm.pv-supports-penetrate-roof`, `fm.switchrooms-depend-on-ventilation` |
| envelope | 27 | `fm.envelope-adhesive-membrane-compatibility`, `fm.envelope-basix-glazing-substitution`, `fm.envelope-ceiling-floor-acoustic-flanking`, `fm.envelope-cladding-drainage-cavity`, `fm.envelope-clinical-finish-cleaning`, `fm.envelope-facade-frame-movement`, `fm.envelope-floor-cupping`, `fm.envelope-floor-finish-impact-sound`, `fm.envelope-insulated-glass-seal`, `fm.envelope-insulation-gaps`, `fm.envelope-joinery-service-positions`, `fm.envelope-landscaped-roof-protection`, `fm.envelope-lining-sag`, `fm.envelope-old-new-weather-barrier`, `fm.envelope-paint-adhesion`, `fm.envelope-paint-before-flooring`, `fm.envelope-room-acoustic-finishes`, `fm.envelope-rough-in-before-linings`, `fm.envelope-safety-glass-substitution`, `fm.envelope-sarking-before-roof`, `fm.envelope-sealed-envelope-ventilation`, `fm.envelope-services-acoustic-penetrations`, `fm.envelope-services-ceiling-space`, `fm.envelope-vapour-barrier-position`, `fm.envelope-waterproofing-before-tiles`, `fm.envelope-wet-membrane-puncture`, `fm.envelope-wet-substrate-membrane` |
| fire | 5 | `fm.brigade-access-obstructed`, `fm.damper-access-maintained`, `fm.plastic-pipe-without-fire-collar`, `fm.sprinklers-share-ceiling-services`, `fm.suppression-hazard-changed` |
| services-wet-air | 13 | `fm.cooling-tower-stagnant-dead-leg`, `fm.exhaust-intake-separation`, `fm.exhaust-penetrates-envelope`, `fm.hydraulic-rough-in-before-linings`, `fm.kitchen-exhaust-makeup-air`, `fm.occupied-stage-services-isolated`, `fm.plant-loads-structure`, `fm.plant-noise-vibration`, `fm.potable-nonpotable-separation`, `fm.process-vendor-utility-mismatch`, `fm.roof-drainage-waterproofing`, `fm.services-plant-maintenance-access`, `fm.services-share-ceiling-space` |
| structure | 11 | `fm.crane-outrigger-ground-bearing`, `fm.drainage-moisture-affects-footings`, `fm.excavation-affects-adjoining-foundations`, `fm.groundwater-loads-basement`, `fm.podium-landscape-waterproofing`, `fm.podium-soil-loads-structure`, `fm.retaining-drainage-protects-wall`, `fm.services-modify-structural-framing`, `fm.survey-datum-controls-structural-setout`, `fm.temporary-works-load-new-concrete`, `fm.termite-barrier-service-penetrations` |

## 2. Format approval (K0 gate)

The 560 `uc.*` records, 331 signals and the new failure modes all use the K0 shapes in `knowledge/SCHEMA.md` ("Works layer"). The PRG's K0 gate is "owner approves shapes", and no approval is recorded yet.

**Your call:** approve the shapes as the format. The content stays `status: draft` either way.

## 3. Open items from the Pass B reports

1. **Records that test `work_type` check nothing yet.** 67 unforeseen-condition records test it in their `when` clauses with `{det: work_type, ...}`. That determinant has no options, and the runtime treats such a test as unknown until D-07 is decided (decision packet §4).
2. **Placeholder year thresholds.** The asbestos threshold is pre-2004 (built in 2003 or earlier). Lead paint is 1970 in the determinant note but 1990 in the envelope records. Both stay placeholders until K3 reads the instruments.
3. **Cross-cluster merge pass.**
   - Overlaps: fire penetrations against services ducts.
   - Misplaced records: vehicle clearance in fire; crane runway in services.
   - Loose merges flagged in the reports.
   - Generated contract wording and severities to review.
4. **A second detector question.** Should detectors also ask "does the specification require the prevention?", which would make the records useful at tender stage? This is a new Jev question per record and changes evidence-call size (see `docs/evidence/2026-10-05-evidence-workload.md`).
5. **Sources.** Most batch 2 records cite only your rows. Your rows are AI-generated and unverified (`data/unforeseen/manifest.json`), so they are leads, not sources of fact.
6. **Attachment targets (D-24).** 49 delivery records attach to a stage or package kind, which the PRG did not list. The recommendation is to accept.
