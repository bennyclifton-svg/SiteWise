# K4 fire and services overlap review

6 October 2026. Knowledge source review only; no runtime, question, predicate,
severity, contract, source ledger or approval status changed.

The flagged fire-penetration/services-duct overlap was checked against five
complete current records and all ten cited dataset rows. The exact source
rows, current records and file hashes are retained in
`docs/unforeseen/k4-fire-services-overlap-review.json`, with `reviewed: false`.
The cited mechanical coordination/construction and hydraulic handover seed
sections were also read. They support coordination and inspection, not a
universal protection prescription or verified instrument number.

Keep these mechanisms distinct: plastic-pipe protection; unsealed hydraulic
penetrations; missing duct crossing protection; an unrated kitchen-exhaust
duct; and a rated duct outside its tested configuration. Shared fire-safety
consequences do not make those records interchangeable. A pipe described as
plastic and unsealed can legitimately overlap two current detectors; this
review does not claim runtime deduplication or measured Jev recall.

Specific review gaps:

- B1-0301 does not establish plastic pipe material. B1-0443, B1-0498 and
  B1-0364 describe crossings without expressly stating absent protection.
  Their citations must not be interpreted as exact defect-detector coverage.
- The hydraulic record attaches to `hydraulic.gas` although its detector and
  sources include sanitary risers. Correcting its attachment requires a
  bounded routing/placement change with coverage evidence, not a file move.
- B2-0201's proposed remedy is an unverified lead. It does not justify a
  universal damper instruction for kitchen exhaust. The existing detector
  describes a reported condition; its source is not instrument verification.

No records were merged or removed, including pre-K4 failure modes. The
assessment closes the initial comparison of this named overlap, not its
remaining attachment/coverage decisions, owner review, or K6 quality gate.
No additional runtime call or dependency was added; user-path timing is
unchanged because only review artifacts were written.

## Attachment correction following the review

The hydraulic record now attaches to
`if.services-penetrate-fire-rated-construction`, whose existing `from` includes
hydraulic services and whose `to` is `fire-passive.penetrations`. This expresses
the shared physical interface instead of classifying all its gas, vent and
sanitary-riser cases as gas. The record stays in the services cluster.

`k4-hydraulic-attachment-correction.json` records the exact before/after
attachment. A parsed comparison confirmed every other field unchanged:
detector instructions/criteria/routing, ID, severity, sources, flag and draft
status. The earlier assessment preserves its historical record snapshot.
The current inventory has been regenerated.

`loadFailures` uses detector `runs_on` to route evidence questions; it does not
load the attachment. Therefore this metadata correction adds no question or
runtime work. Strict knowledge validation passes with zero errors and the
same two contamination/flood mapping warnings; the knowledge Go suite passes
(3.584 s). No dependency, schema, budget or verified rule number changed.
The knowledge content version changes normally, so derived freshness follows
the existing version mechanism. The last measured profile timing failure
remains unresolved; this correction is not a speed-gate closure.
