# WP-30 mapped complexity suggestions

Date: 6 October 2026. Implemented and tested; overall latency gate remains red.

Lane A. Outcome: draft consultant suggestions reflect known mapped project
conditions as well as class/work type, reducing manual review of the copied
roster. Package read/write budget remains p50/p90 100/250 ms. No new service,
dependency, Jev question, appointment or automatic package creation.

## Behavior

GET packages reads relevant whole-site determinant/condition rows alongside
the existing whole-site class and project/part work types, in the same scoped
repeatable-read transaction. `PackageComplexityFacts` keeps applied values;
assumptions, allowances, unknowns, superseded and unapplied values are excluded.
Conflicting values for one field are discarded instead of choosing by row order.
Part-specific complexity rows do not supply whole-site defaults.

`SuggestedPackages` merges baseline suggestions with matching complexity
additions. Both the field and its value must exist in the active determinant
or condition catalogue; a legacy rule cannot activate merely because an input
map contains its old name. Mapped heritage, BAL, planning and environmental
sensitivity rules can contribute. Unresolved contamination/flood names remain
inactive. A known condition can suggest its consultant without inventing a
building class or work type.

Matching rules and existing active services-package titles deduplicate the
suggestions. Each complexity match appears in the additive API field
`matched_fields` (field name to applied value). A baseline match retains its
class/work-type reasons too. Output remains sorted, proposed and draft according
to catalogue status, with the catalogue version retained. Reading suggestions
does not persist a package or approve a consultant. No UI workflow was added.

## Validation

Pure tests cover exact mapped values, unmapped/legacy values, missing class,
deduplication, matched reasons, draft status, ineligible values and conflicts.
The store test proves whole-site selection, exclusion of another part's fact,
removal of an explicitly unknown match and foreign-tenant not-found behavior.

Serial suites passed: profile 7.336 s, procurement 1.203 s, store 28.113 s,
HTTP API 13.721 s. The benchmark suite first lacked its required test-database
environment variable; rerunning with the established local test DSN passed
in 7.097 s. No code change was needed for that configuration failure.

The package benchmark now applies four mapped conditions and checks 12
deduplicated suggestions, four matched-field reasons and draft/proposed status
on every read. The 20-file/two-round/40-sample run is
`bench/results/2026-10-06-package-complexity.json`:

| Path | p50/p90 ms | Budget ms | Result |
| --- | --- | --- | --- |
| Package read | 3.204 / 5.398 | 100 / 250 | Pass |
| Package write | 1.736 / 2.354 | 100 / 250 | Pass |
| Spec Home edit | 50.045 / 53.575 | 50 / 150 | Fail |
| Integrated rebuild diagnostic | 87.503 / 91.597 | 100 / 300 | Pass |

The combined gate still fails. This is local replay timing, not live-provider
or target-VPS release evidence. No merge, deployment or whole-package completion
is claimed. Earlier large intermittent profile delays remain unresolved;
unmapped rules, owner content/usefulness review and later milestone gates remain.
