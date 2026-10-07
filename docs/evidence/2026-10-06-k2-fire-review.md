# WP-K2: draft existing-fire approval-basis review

Lane A, knowledge. Help the project reviewer identify approval implications
when existing fire measures change. This bounded slice adds one source-anchored
draft consequence; it does not complete WP-K2, the fire cluster's research,
owner review or M1. No shared contract, service, dependency or Jev question changes.

`cq.existing-fire-measures-approval-basis-review` applies to included alteration,
replacement, upgrade or removal of active/passive fire measures in an existing
building. It proposes review of the existing strategy and approval implications.
The frozen fire guide's compliance-strategy and inception sections are cited by
exact headings. The record contains no numeric requirement and sets
`clause_verified: false`, `governed_by: []` and `status: draft`.

The existing `approval` proposal kind can create a delivery review through the
user's acceptance. It does not require inventing a physical target. Consequence
investigations still lack that target in their knowledge shape and were not
added. A certificate-existence signal cannot establish that subsequent changes
are addressed, so this record adds no suppression signal. Its overlap with the
broader authority-design-change unforeseen record remains a K6 usefulness-review
item; that existing record is unchanged.

Scope tests exercise active/passive matches, new-building exclusion, retain,
repair, excluded work and unrelated services. The first fixture used an unknown
passive-fire ID; it was corrected to the catalogue's rated-elements ID. The
complete PostgreSQL-backed Go suite then passed
(`.tools/k2-fire-review-all-tests.log`). A further real-catalogue acceptance
test passes: status stays `not_submitted`, review status is
`accepted_for_planning`, no package/work/stage is inferred, and the saved
provenance retains draft status and the triggering work
(`.tools/k2-fire-acceptance-test.log`).

The full uninstrumented 20-file/two-round/40-sample workload with integrated
proposal diagnostics is `bench/results/2026-10-06-k2-fire-review.json`.
It **fails**: ordinary edit p50/p90 679.626/685.590 ms versus 50/150 ms;
combined rebuild 713.996/724.880 ms versus 100/300 ms. This repeats the earlier
large slowdown and establishes no passing speed gate or isolated incremental
cost for the new record. Automatic proposal generation remains disabled.

The strict checker reports zero errors and three existing package-default
warnings, with 2,000 ledger rows and zero pending. The current review inventory
was regenerated to include the second consequence; no record was promoted to
reviewed. No instrument verification, owner usefulness score, live-call or
target-VPS result is claimed. The cluster REPORT records scope and unresolved
overlap explicitly.
