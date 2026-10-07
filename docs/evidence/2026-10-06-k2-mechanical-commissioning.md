# K2 mechanical commissioning-plan review

6 October 2026. Lane A, time-to-decision. Draft knowledge only; no new
service, dependency, statutory rule or runtime AI question.

`cq.mechanical-commissioning-plan-review` follows the mechanical-services
seed's `## Commissioning and handover` section. New, altered, replaced or
upgraded mechanical work produces a draft hold-point proposal to review
commissioning requirements before tender. The record concerns planning the
affected systems, prerequisites, witnesses, acceptance criteria and handover
evidence. It does not assert that equipment has been commissioned.

The existing evaluator carries the affected part and triggering work into
the reason. The existing hold-point acceptance path creates an undated
delivery milestone, initially unfinished; it does not infer a package, stage,
responsible party or physical target. Its programme severity is provisional,
as is all content in this draft. Repair/removal/retention and unrelated work
are outside this bounded trigger. Wider commissioning coverage needs separate
source and usefulness review.

Strict knowledge validation passes with zero errors and the same two existing
mapping warnings. The new evaluator regression covers four positive actions
and five negative cases, checks draft/part/trigger provenance, and rejects an
invented physical target/action. Works (2.745 s), knowledge (3.535 s), store
(29.265 s) and HTTP API (13.939 s) suites pass. Evidence-question routing is
unchanged: this is a code-evaluated consequence, with no signals added.

Decision CRUD retains the 100/250 ms budget; profile edit/rebuild retain
50/150 and 100/300 ms. Timing is recorded in
`bench/results/2026-10-06-mechanical-commissioning.json`. Automatic proposal
generation remains disabled pending the integrated edit and owner-quality
gates. No M1 or K2 completion is claimed.

The timing run exits 1: spec-home edit is 58.454/60.503 ms, missing the median
budget. Rebuild passes at 88.927/95.560 ms. The existing delivery acceptance
benchmark passes at 3.732/4.425 ms and undo at 6.998/10.173 ms; these exercise
the shared endpoint, not a newly added commissioning-specific acceptance case.
No tests or compilation ran concurrently with measurement. Retain the timing
failure as open evidence rather than claiming a full green gate.

## Real-record acceptance check

`TestMechanicalCommissioningProposalCreatesUndatedReview` now evaluates the
actual catalogue record from altered air-conditioning work, persists its
proposal and accepts it through the store. It verifies a planned milestone,
no baseline/target/forecast/actual dates, no inferred package/stage/work/owner
assignment, and the original draft proposal's part and triggering work in
provenance. Undo retires the untouched milestone. This focused test and the
existing delivery acceptance/undo regression pass (1.443 s). No production
source changed in this follow-up, so no fresh timing claim is made.
