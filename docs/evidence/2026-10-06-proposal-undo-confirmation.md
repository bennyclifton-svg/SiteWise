# Proposal undo decision confirmed

The owner confirmed: retire the untouched, unreferenced item created by an
accepted proposal; refuse undo after edits or use. This confirms the existing
implementation rather than changing the runtime contract.

Lane A: preserve accepted planning work and its audit history. No new endpoint,
dependency, Jev call or service. The existing decision CRUD budget remains
p50 100 ms / p90 250 ms; this test-only change introduces no runtime work and
does not constitute a new timing measurement.

Extended `TestProposalUndoRefusesEditedOrReferencedWork` to independently cover
persisted child work, a delivery milestone and a saved RFP draft, in addition
to edits, another proposal decision and package scope. Every refused undo must
leave the work active and the decision version and undo history unchanged.
The child fixture uses SQL because the split API belongs to a later package.

Validation on 6 October 2026: the focused PostgreSQL-backed proposal tests pass
in 5.885 seconds. This includes untouched retirement/reacceptance and refusal
tests for work, packages, scope and delivery. Reacceptance retains identity.
The first child-fixture run omitted required inclusion; the fixture was
corrected to copy its parent's inclusion before the successful run.
