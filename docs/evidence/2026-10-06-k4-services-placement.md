# WP-K4 services placement review

Date: 6 October 2026. Status: partial merge-pass implementation.

Lane A knowledge work. Outcome: distinguish the physical ownership of two
flagged records and make the boiler-room investigation describe a check instead
of directing construction. No new runtime questions, systems or dependencies.

## Boiler-room oil tank

Read frozen dataset row B2-0870 and its triage entry, the complete unforeseen
record and signal, and `mechanical.central-plant` in the system catalogue.
The row concerns an old tank occupying the space required for replacement
boilers. Central plant explicitly includes boilers and their plant rooms.
Retain the record and signal in services-wet-air with that attachment.

The previous `de_risk.kind: investigation` label instructed removal. Replace
it with: "Check for an old oil tank in the proposed boiler room and confirm any
removal needed". The record's broad existing-building/central-plant-work
predicate does not itself establish that a tank exists, so an unconditional
removal instruction would overstate the evidence. A stated removal still
suppresses the proposal through the unchanged signal.

Only the label changes. Stable ID, trigger, target attachment, signal question
and routing, contract wording, severity, source and draft status are preserved.
The rule content hash and proposal fingerprint can change as designed; an old
decision is not silently treated as approval of a changed proposal. The source
row remains an unverified lead, not a legal requirement or owner approval.

## Operating crane on a runway

Read B2-0854, its triage entry, unforeseen record and signal. Its broad
`mechanical` mapping was explicitly chosen because no leaf system matched.
The present structure catalogue mentions temporary crane bases/outriggers and
tie-ins, which does not by itself resolve a permanent operating crane and its
runway. File relocation alone would preserve the questionable mechanical
predicate and signal routing. Leave this record unchanged pending an explicit
taxonomy decision; do not guess a new physical target or expand Jev questions.

## Validation and remainder

Strict knowledge validation: zero errors, three existing package-default
mapping warnings, 2,000 covered dataset rows and zero pending.
`go test -p 1 ./internal/knowledge ./internal/works` passes in 3.437 and 1.831
seconds. Current K4 audit hashes are refreshed.
This label edit adds no operation to a user-facing path;
no new latency measurement is claimed. Budgets remain filing 1,000/2,000 ms,
profile edit 50/150 ms and rebuild 100/300 ms (p50/p90). The existing full-fixture
latency failure is unresolved.

Generated contract wording, severity, broader trigger usefulness and owner
content review remain open. This review resolves the oil-tank file placement
question but does not complete WP-K4 or promote its content to reviewed.
