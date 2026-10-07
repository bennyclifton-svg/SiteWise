# K4 electrical/fire merge review and missing work triggers

Date: 6 October 2026. Draft source review plus two bounded predicate fixes.

Lane A for the predicate changes. Outcome: proposed investigations follow the
physical work named by their source leads, even when no separate supply or
lighting work is in scope. Budgets remain edit 50/150 ms, rebuild 100/300 ms
and filing 1000/2000 ms. No dependency, runtime Go behavior, Jev question,
additional call, record, verified rule or automatic work creation is added.

## Source assessment

Read all 21 dataset rows cited by eight flagged electrical/fire records.
`docs/unforeseen/k4-electrical-fire-merge-review.json` preserves their exact
wording, record-specific findings, open decisions and the before/after probe.
These are AI-generated unverified leads. Their references to NCC or Australian
Standards do not establish a verified instrument requirement.

- Lift power: a fire-strategy essential circuit is not necessarily equivalent
  to the detector's specified standby supply. This terminology/coverage gap
  remains; the existing detector was preserved.
- Device coordination: actual non-coordination and a device/study mismatch
  remain distinct conditions, both explicitly named by the current detector.
  A changed device list alone does not prove coordination failure.
- Added electrical load: the common capacity investigation is retained. A
  stated capacity does not itself establish adequacy, and a larger load does
  not automatically mandate three-phase supply.
- Overhead lines: crane reach and delivery access share a mechanism. The
  existing-supply predicate does not establish overhead lines or lifting work;
  this applicability limitation remains for review.
- Fault rating: inadequate and unknown ratings share an investigation but are
  different evidence states. The current phases-or-fault-level signal cannot
  establish that the board's rating is adequate. Signals do not auto-dismiss
  proposals; no signal or detector was changed here.
- In-ground assets: pole bases, pits and crossovers share a location/clash
  concern. A structure record already covers footing investigations; adding
  another footing trigger would need an overlap decision.
- PCB leads: known capacitor contamination and untested transformer oil must
  remain distinct. Generic survey evidence is not a material-specific finding;
  the switchboard target for lighting-triggered work remains a review issue.
- Performance solutions: layout and occupant-management assumptions share an
  applicability review. The broad current predicate does not establish that
  a performance solution exists or its assumptions are breached. Generated
  contract allocation, severity and usefulness remain unapproved.

## Changes and reproduction

The loaded engine failed to raise the named capacity UC for new EV-charging
work (B2-0360), or the named in-ground-asset UC for new vehicle-access work
(B2-0370). The existing taxonomy explicitly includes chargers and driveways/
crossovers in those respective systems. Both source rows were already cited.

Added `electrical.ev-charging` to
`uc.existing-supply-or-main-capacity-below-added-load.when.works.system` and
`site.vehicle-access` to
`uc.in-ground-supply-asset-clashes-with-new-works.when.works.system`.
No actions, existence conditions, targets, signals, wording, severities or
draft statuses changed. The coarse vehicle-access system also includes work
that may not involve excavation; the suggestion remains provisional and K6
must assess its usefulness. This is not a new claim of an actual cable clash.

The synthetic probe inspects only three named UC records, not total proposal
recall. Before the fixes it showed the two misses and the existing footing
coverage. Afterwards the two missing investigations appear, while footing
coverage remains through the structure record. A real-catalogue regression
failed before the fix and passes after it. Its unrelated-system fixture first
used a nonexistent ID; that fixture was corrected independently of the two
demonstrated predicate misses.

## Validation

Tests cover new/alter, retain, exclusion, unrelated work, known-false and
unknown existing-building evidence, targets, trigger/part provenance and
draft status. Suites passed: works 2.554 s, knowledge 3.517 s, profile 7.498 s,
store 28.107 s, API 13.364 s. The expanded works cases passed in 2.551 s.
Strict knowledge check: zero errors, two existing package-field warnings,
2000 associated ledger rows, zero pending. The K4 hash inventory was refreshed.
Owner knowledge review, K6, M1 and release speed gates remain open.

## Timing

`bench/results/2026-10-06-k4-electrical-triggers.json` records the final
20-file/two-round/40-sample integrated-proposal diagnostic. No tests or other
compilation ran during measurement.

| Path | p50/p90 ms | Budget ms | Result |
| --- | --- | --- | --- |
| Filing | 468.288 / 620.423 | 1000 / 2000 | Pass |
| Spec Home edit | 48.599 / 51.021 | 50 / 150 | Pass |
| Integrated rebuild diagnostic | 87.337 / 92.143 | 100 / 300 | Pass |
| Proposal read | 4.168 / 5.405 | 100 / 250 | Pass |

The local gate passed. This does not demonstrate that the earlier intermittent
large delays are fixed or replace live-provider/target-VPS evidence. Automatic
proposal generation remains gated by the other outstanding criteria.
`git diff --check` passed with the existing line-ending warnings.
