# K4 envelope loose-merge source review

Date: 6 October 2026. Draft source assessment, not owner approval.

This addresses the three batch-1 envelope merges explicitly flagged in
`docs/unforeseen/pass-b-reports.md`. All eight cited dataset rows were read.
They are AI-generated unverified leads under `data/unforeseen/manifest.json`,
not primary instruments or project evidence. No new legal or numeric claim is
verified by this assessment.

| Record | Source comparison | Disposition |
| --- | --- | --- |
| `fm.envelope-thermal-break-missing` | B1-0232 states a missing bracket break. B1-0239 states a thermally bridged window frame. The latter need not explicitly identify a missing break. | Partial coverage. Preserve the current literal detector; record the broader frame-bridge condition as an unresolved gap. |
| `fm.envelope-glass-weight-exceeds-frame-capacity` | B1-0238 changes acoustic glass specifications; B1-0270 explicitly reports insufficient frame capacity; B1-0300 raises heritage-frame suitability. | Retain the common capacity mechanism. A specification change or heritage designation alone must not flag a defect. This does not cover acoustic performance or heritage consent. |
| `uc.colour-or-dye-batch-variation` | B1-0230 cladding, B1-0717 tiles and B1-0727 carpet share batch-related appearance variation. | Retain the common supply mechanism. Single-batch ordering remains a proposed control, not a verified universal requirement or guarantee. Contract wording, feasibility and severity remain for owner review. |

Each record now carries this qualification in `notes`, which the schema defines
as reviewer-only. The exact row wording, assessments and open follow-ups are
saved in `docs/unforeseen/k4-envelope-merge-review.json`. Detector instructions,
criteria, routing, predicates, signals, contract text and severities are unchanged.
All statuses stay draft; no records or dataset rows were added or removed.

The thermal finding must not be hidden by the checker's zero-pending result:
the ledger establishes that a row has an associated record, not that the model
detects every interpretation of it. Closing the gap needs a bounded detector
decision and evaluation; this note does not pretend to provide either.

The conservative interpretation follows TypeSafe's instruction to ask literal
conditions and align criteria with instructions ([Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
No model call or detector change was made. The [fan-out guidance](https://docs.typesafe.ai/patterns/fan-out)
continues to apply to any later approved question change. The linked how-to-build
page was unavailable through the browser tool during this review; no new claim
depends on it.

This is knowledge-review work, not a runtime speed change. The existing edit
and rebuild budgets remain 50/150 and 100/300 ms, and the outstanding speed gate
is not closed. Owner review, K6 usefulness scoring and M1 acceptance remain open.

Validation: strict knowledge checker passed with zero errors and the two
existing unresolved package-field warnings; 2000 ledger rows, zero pending.
All 32 tooling tests passed in 6.880 s. The K4 hash inventory was regenerated.
