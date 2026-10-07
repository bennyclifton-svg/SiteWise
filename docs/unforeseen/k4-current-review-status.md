# K4 review status, 6 October 2026

This supplements [the original owner packet](k4-owner-review-packet.md). It
records current evidence, including the later vehicle-clearance file move;
it approves no content and does not close the cross-cluster merge pass or the
owner-review gate.

## Verified inventory

[The machine-readable audit](k4-current-audit.json) includes the exact pre-K4
commit, SHA-256 hashes of the current knowledge files, every changed existing
failure-mode ID and field, and all 67 work-type-predicate IDs. Regenerate it
from the repository root with:

```powershell
python tools/audit_k4_review.py > docs/unforeseen/k4-current-audit.json
```

The inventory matches the strict checker: 859 failure modes, 560 unforeseen
conditions, 331 signals and three consequences (including subsequent K2 fire
and mechanical commissioning review drafts). Every inspected record remains
draft. Of the 172 pre-K4 failure modes, none was removed; 72 changed, comprising
71 source-only changes and one substantive change. No existing failure-mode ID
moved files. The substantive change remains `fm.integrated-fire-test-failed`:
`detector.runs_on` adds smoke-control passages, with notes and sources added.
Its question wording is unchanged. The owner's accept/revert decision remains
open; the report has not changed its detector.

## Corrections to the packet's implementation status

The 67 unforeseen records still use `{det: work_type, ...}` syntax, but they
no longer necessarily evaluate to unknown. `Catalog.Holds` reads work types
from the separate code-supplied `WorksEnv.WorkTypes` input. The proposal adapter
supplies applied project/part types. Missing or invalid types remain unknown;
arbitrary determinant-map values cannot supply this input. See
`internal/knowledge/works_eval.go`, `internal/profile/proposals.go`,
`TestWorkTypeUsesCodeFedProjectAndPartTypes`, and the
[WP-25 evidence](../evidence/2026-10-05-wp25-works-predicates.md).
This is evaluator support, not evidence that every rule is useful or that
automatic proposal generation is enabled.

Across all 560 unforeseen records, attachment counts are 449 system, one
interface, 96 stage and 14 package-kind. Thus 110 records use stage/package-kind
attachments. The packet's reference to 49 delivery records is not the full
catalogue total; format review should consider the full current inventory.

## Remaining K4 review work

The [investigation-label pass](../evidence/2026-10-06-k4-investigation-labels.md)
corrected 82 draft labels that directed physical work or selected design
solutions prematurely. Exact
before/after text and sources are in `k4-investigation-label-review.json`.
Predicates, signals, contract wording and severities were preserved; broader
content and usefulness review remain open.

- The flagged cross-cluster overlaps, loose merges and generated contract
  wording/severities still need record-by-record review against their sources.
  The named fire/services penetration overlap now has a
  [five-record, ten-row assessment](k4-fire-services-overlap-review.json).
  Distinct pipe/duct failure mechanisms are retained; source-to-detector
  coverage limits remain explicit follow-ups. The hydraulic record's gas-only
  attachment has been corrected to the existing services/fire penetration
  interface with every other record field preserved. See
  [the review evidence](../evidence/2026-10-06-k4-fire-services-overlap.md).
  The three flagged batch-1 envelope merges now have a
  [source assessment](k4-envelope-merge-review.json): glass-capacity and
  batch-appearance mechanisms are retained with explicit limits; B1-0239's
  broader thermal-bridge condition remains a detector coverage gap. This is
  agent source review, not owner approval. See
  [the evidence](../evidence/2026-10-06-k4-envelope-merge-review.md).
  A further [electrical/fire assessment](k4-electrical-fire-merge-review.json)
  covers eight records and all 21 cited rows. Two missing trigger mappings
  (EV chargers and crossovers) are corrected and regression-tested; essential
  versus standby power, signal sufficiency, target specificity and broad
  applicability remain explicit review gaps. See
  [the implementation evidence](../evidence/2026-10-06-k4-electrical-fire-merges.md).
- `fm.vehicle-clearance-lost-under-structural-beam` has moved from fire to
  structure with its complete record preserved, including detector routing.
  See [the relocation evidence](../evidence/2026-10-06-k4-vehicle-clearance.md).
  `uc.old-oil-tank-occupies-plant-room-for-new-boilers` (B2-0870) stays in
  services: its source concerns space for replacement boilers and the existing
  central-plant definition includes their plant rooms. Its investigation label
  now asks for a check instead of directing removal. The crane-runway record
  (B2-0854) still needs a taxonomy decision; moving its file alone would not
  correct its broad mechanical predicate and signal routing. See
  [the services placement review](../evidence/2026-10-06-k4-services-placement.md).
- Dataset rows remain AI-generated, unverified leads, as stated in
  `data/unforeseen/manifest.json`. Their presence in a source list does not
  verify an instrument, year threshold or construction obligation.
- A second detector question and expanded runtime signals remain outside this
  review update. There are no additional Jev calls.
- Owner content review, K6 usefulness scoring and M1 quality approval remain
  prerequisites for later milestones. WP-70 polish remains in M3.

## Validation and scope

`python tools/check_knowledge.py --strict` reports zero errors, 2,000 covered
dataset rows and zero pending ledger entries. Its two current warnings cover
unmapped contamination and flood fields. The
[value-validation correction](../evidence/2026-10-06-package-default-value-validation.md)
exposed two legacy heritage values; the later
[exact identifier mapping](../evidence/2026-10-06-package-default-identifiers.md)
resolved those and the six BAL tokens. The later
[WP-30 implementation](../evidence/2026-10-06-package-complexity-suggestions.md)
uses only mapped, eligible whole-site values for draft complexity suggestions;
unresolved contamination/flood fields remain inactive.
All 32 tools tests pass. The read-only inventory uses the repository's
existing PyYAML dependency and Git; no dependency was added. No user-facing
runtime path changed, so this work makes no new latency claim. The existing
50/150 ms edit and 100/300 ms rebuild gates remain open.
