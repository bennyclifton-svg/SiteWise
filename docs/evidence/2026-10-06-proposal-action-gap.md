# Physical-work proposal action gap

6 October 2026. Read-only catalogue and implementation audit; no runtime change.

The catalogue scan of interface consequences, consequences and unforeseen
conditions found one `kind: work_item` proposal:
`ic.penetrates-make-good` in `knowledge/works/interface_consequences.yaml`.
It proposes making good penetrated construction. It has no action.

The loaded `knowledge.Proposal` shape currently contains only kind and label.
The generated proposal schema supports an action, but the interface adapter
cannot populate one from this record. `Store.AcceptProposal` requires a
known action for physical work and therefore refuses this proposal; only
investigations have the explicit `investigate` action mapping. This is a
real incomplete acceptance path, not evidence of a completed WP-26.

Plan D-03 resolves sequencing and ownership of non-investigation acceptance,
not the action semantics of making good. The owner has been asked whether
acceptance should require an explicit action choice or create a `repair`
item on the penetrated construction. No action is silently inferred from
the label or copied from the triggering work.

If catalogue-authored action is chosen, update the documented proposal shape,
strict checker and Go loader validation before carrying it through evaluator
fingerprints and acceptance. If user selection is chosen, validate the action
at the request boundary and preserve the user's explicit choice in decision
provenance. Either implementation needs stale-fingerprint, idempotent retry,
undo/reacceptance and org/project isolation coverage. Current fail-closed
behaviour remains until the direction is resolved.

The existing 100/250 ms decision endpoint budget remains; there is no new
dependency or Jev call. Separate missing physical targets in other proposal
kinds and the integrated edit/owner quality gates are not resolved here.
