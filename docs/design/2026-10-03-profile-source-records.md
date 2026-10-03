# Complete source records for the project profile

Approved by the owner in chat on 3 October 2026. Extends the compact profile
design: every extracted source unit remains inspectable, including unresolved
readings and material for which there is no profile field. This supersedes the
500-passage limit, one-leaf-per-family routing and 120-character-only detail.

The building remains the model. Source units retain exact text, page/location,
section and surrounding context. Jev classifies their purpose, responsibility,
scope and any number of systems; code harvests values and copies source text.
No generated summaries, embeddings or other AI providers are introduced.

Extraction has complete/needs-attention coverage, not a silent success at a
limit. Full source text is preserved independently of its bounded reading
units. A textless PDF page is flagged for inspection, not assumed blank.
Clause boundaries, list introductions and table rows inform units; numeric
limits fail explicitly rather than quietly dropping text.

One label fan-out and one evidence fan-out per unit remain. System families are routed in the first call. Every child of a relevant family
is independently asked in the second call; no single-choice bottleneck. Unknown classification,
missing answers and unassigned substantive text remain in Needs mapping.
Completed requests are checkpointed by the full call and question version;
retries reuse them. Background concurrency is bounded and preserves intake's
reserved capacity. User values remain final. Superseded documents are excluded
from the current requirement view just as they are from profile reconciliation.

The profile stays compact. A paginated source-record view offers system,
unresolved and all-record filters; source links open the original page. Coverage
counts measure processing, never semantic accuracy. Machine mappings remain
provisional. New builds invalidate old extraction on the next explicit update.

Validation: Bankstown regression fixtures (83 pages, final-page survival,
eight storeys, 33 units, basement-level/car-space distinction, paired services,
consent date); org isolation; resume and repeat-update tests; UI integration.
Record live timing and Jev token usage without claiming perfect recall.
Background question construction and source splitting have deterministic
benchmarks; paginated source reads have p50 <= 50ms / p90 <= 150ms local budgets.

Jev references: [fan-out](https://docs.typesafe.ai/patterns/fan-out),
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[confidence](https://docs.typesafe.ai/confidence),
[limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
[API](https://docs.typesafe.ai/api).
