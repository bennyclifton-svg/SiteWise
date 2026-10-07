# WP-45a: minimum work correction API

Lane A. Outcome: correct a saved work item's action, title, inclusion, condition
note, target and planning quantity before allocating responsibility. This serves
answer trust and time-to-decision. Write budget: p50 100 ms / p90 250 ms. This
brings the correction needed by M1 forward without implementing M2 subdivision
or retirement. Shared site condition keeps its existing versioned profile route;
part/system reassignment is not introduced by this slice.

`PATCH /api/projects/{id}/works/{wi}` takes an expected version and the corrected
fields. It locks the project, checks actor and item ownership, reads the live
item, validates the merged result, and commits the correction, works revision,
durable event and rebuilt profile together. The handler wakes the org-scoped
event broker after success. Stale versions return 409, unknown/foreign items 404,
invalid planning values 422 and unrecognised request fields 400.

Quantities retain decimal digits exactly. An explicit null quantity clears its
unit; omitted fields retain their values. Corrections keep item identity,
original sources and proposal linkage, add the latest editor/time, and become
user-owned accepted-for-planning input. They never establish verification.
Rebuilds preserve the correction. Report citations identify the latest editor
while retaining original source material. Proposal undo refuses the edited item
because its version no longer matches the accepted snapshot.

Targeted works, store, HTTP API, reports and benchmark suites pass
(`.tools/wp45-work-edit-tests.log`). Tests cover two edits racing on one version,
decimal precision, clearing quantity, rebuild preservation, retained evidence,
author attribution, rejected invalid targets/actions/quantities, stale writes,
cross-org and cross-project access. The proposal undo regression now uses the
real correction method to establish that an edited created item cannot be undone.

The reduced local diagnostic
`bench/results/2026-10-05-wp45-work-edit-diagnostic.json` passes all user-path
gates. Works writes include 40 creations and 40 corrections: p50/p90 7.1/9.1 ms
against 100/250 ms. Works reads are 1.6/1.8 ms. This uses 20 files over two rounds
with replayed recorded latency, not full 0991 or production release evidence.
The separately reported Spec Home profile-edit diagnostic is 88.5/93.3 ms,
above the 50 ms p50 profile-edit target despite the aggregated profile-edit gate
passing (27.0/52.4 ms). That larger-workload performance gap remains unresolved.
Extraction and deterministic-rule component budgets also remain target-VPS gates.

No dependency, service, runtime AI call or schema migration was added. The
remaining M1 proposal/delivery controls are still required. This is
not completion of WP-24's later edit/split/retire package or the M1 acceptance gate.

## Work review controls

The Works view now lists saved work with its location, action and inclusion and
shows its system, review status, origin, shared existing condition, planning
quantity, structured target values and source excerpts. The correction form edits
title, action, inclusion, condition note, target wording and quantity/unit. It
preserves structured target values and clause references; shared site condition
remains in the profile. Switching project views keeps an unsaved correction.

Live events refresh saved work without replacing the editor. If its version
changes, saving is disabled until the user compares the saved record with their
unsaved correction and explicitly adopts that version. Reconciliation takes the
latest structured targets while retaining the user's typed target wording.
An unavailable item cannot be saved. Empty, loading and recoverable error states
are present. Deprecated systems explain why correction is unavailable.

Frontend type-check/build passes. The real browser regression passes (1.3 s test
body), including retained edits across navigation, concurrent updates, explicit
reconciliation, preservation of another editor's structured target values,
decimal quantity and clearing its unit, and mobile overflow. The existing package
and RFP browser flow also passes (3.4 s). Test-server cleanup still needs the
identified Windows process workaround. The intake browser regression passes
(5.5 s), including file formats, correction, reconnect and other-org isolation.
Screenshots are
`.tools/wp45-works-desktop.png` and `.tools/wp45-works-mobile.png`.
The detector returned no findings. Independent Impeccable review disposition:
**ship**, with no material fixes for this bounded UI extension. This is not a
whole-M1 acceptance or accessibility certification.
