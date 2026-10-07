# K4 precise detector coverage corrections — 7 October 2026

Lane A. Existing source rows include actual pavement/subgrade capacity failures,
explicit thermal bridging, and a lift missing the specified essential circuit.
Their previous coverage targets asked different narrower questions (omitted
heavy-load design, missing thermal break, missing standby supply). Add distinct
literal draft detectors instead of silently redefining the earlier ones. Update
row coverage to the precise new record while preserving historic source citations.

For maintenance access, retain the existing literal inaccessible-for-maintenance
question and add the concrete source systems for valves, air handling/filter
access and water tank access. Do not route every hydraulic/mechanical passage via
a top-level wildcard. No new call: these join the existing labelled evidence
fan-out. Labels continue to be the only routing gate.

Question decisions follow [TypeSafe patterns](https://docs.typesafe.ai/patterns)
and [Jev1.13 literal-reading/irrelevant-context guidance](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
Questions ask whether the source explicitly states a condition, not whether
engineering values are adequate. No calculation, regulation number, causal
inference or text generation. All new records remain draft.

Done: source records/checker/routing regression and positive/negative synthetic
answer keys. Source and Hale request-size preflights are required after catalogue
freeze. Existing latency budgets apply; measurements are not live accuracy.
No dependency. Missing silence/hypothetical/requirement-only passages are negative;
unknown remains absence of evidence, never proof of compliance. Owner review and
actual Jev answer quality remain separate from deterministic routing tests.

Implemented three new literal draft detectors, five coverage remaps and five
additional concrete maintenance routing labels. Existing detector wording remains
unchanged; the maintenance routing extension still requires owner acceptance.
Strict checker passes:862FM,13CQ,2000rows0pending,0errors2existingwarnings.
Routing/key-completeness test passes0.521s;9synthetic positive/negative examples
are checked in. This validates routing and key structure only, not Jev answers.

Final16CQ/862FM catalogue offline preflights pass: source14requests876questions
(max224questions/232858bytes), Hale22requests1605questions
(max227questions/224918bytes). These measure current questions on recorded
labels only; no live Jev answers were generated.
