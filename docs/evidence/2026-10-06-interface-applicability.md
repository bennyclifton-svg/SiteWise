# WP-26 interface applicability

Date: 6 October 2026. Lane A, scoped correctness fix.

Outcome: interface consequences honor the conditions already declared on their
interface. Known-false conditions suppress proposals; unknown inputs remain
visible for review. This prevents existing-building advice being presented as
applicable when the affected part explicitly says it is not existing.
Edit and rebuild budgets remain p50/p90 50/150 and 100/300 ms. No dependency,
service, Jev question, approval or automatic work creation is introduced.

## Reproduction and change

A real-catalogue tenant-fitout regression initially produced capacity proposals
for all three parts: existing, new and unknown. Its reasons omitted the
existing-building determinant entirely. The interface loader discarded
`applies_when`, and the interface evaluator did not evaluate it. Part input
inheritance was already provided by the profile adapter.

The loader now retains the predicate. The complete evaluator passes effective
part evidence into the interface evaluator, which uses `TraceRelevantWorks`.
Traces are reused only within one evaluation for the same edge and part;
interfaces without conditions skip tracing. Relevant determinant values and
origins, and system-presence/existence states, enter the reason and existing
semantic fingerprint. A compiled evaluator hashes the loaded condition as part
of interface content. A nil condition is omitted from that serialized hash,
preserving the previous fingerprint shape for unconditional edges. A regression
compares against the old shape and checks that adding a condition changes it.
No new predicate language or persistent cache is added.

The narrower `InterfaceProposals` API has no determinant inputs: those remain
unknown. The production-facing `EvaluateProposals` and compiled `Evaluator`
receive part inputs. Both preserve draft status and existing target-existence
suppression. Unforeseen-condition records still evaluate their own predicates;
sharing an interface ID does not turn them into interface consequences.

## Validation

Regression coverage proves known true/false/unknown behavior, independent part
inputs, determinant origins, relevant versus unrelated fingerprint changes,
reevaluation after an edit, compiled/uncached equivalence and system-presence
traces. One presence-test assertion initially matched an unrelated UC record;
the assertion was restricted to `record_kind: ic`, without changing runtime
behavior to satisfy it.

Knowledge (3.483 s), profile (7.413 s), store (29.420 s), and API (13.517 s)
suites passed. After correcting the assertion and avoiding traces for nil
conditions, works (2.434 s) and profile (7.618 s) passed. Strict knowledge check:
zero errors, two existing unresolved package-field warnings, 2000 ledger rows
and zero pending. The final hash-compatibility change passed works (2.420 s)
and knowledge (3.818 s). Timing results follow separately.

This fixes the applicability limitation recorded in the tenant-fitout handoff.
It does not establish owner approval, AT15 project acceptance, K6 or M1 passage.
Automatic proposal generation remains behind the existing quality/speed gates.

## Local timing

Both runs used 20 files, two rounds, 40 API samples and the integrated-proposal
diagnostic, with no tests running concurrently. Before the final hash-compatibility
adjustment, `bench/results/2026-10-06-interface-applicability.json` passed the
local gate: Spec Home edit 49.546/51.824 ms and rebuild 87.011/93.329 ms.

The final code's artifact is
`bench/results/2026-10-06-interface-applicability-stable-hash.json`:

| Path | p50/p90 ms | Budget ms | Result |
| --- | --- | --- | --- |
| Filing | 452.865 / 661.188 | 1000 / 2000 | Pass |
| Spec Home edit | 51.298 / 54.663 | 50 / 150 | Fail |
| Integrated rebuild diagnostic | 88.170 / 98.940 | 100 / 300 | Pass |
| Proposal read | 4.534 / 5.737 | 100 / 250 | Pass |

The combined final gate fails. Identity extraction and deterministic-field
substage overruns are reported for target-VPS assessment. The first pass is not
evidence of reliable speed, and these runs do not isolate the cause of edit
variability. Earlier large intermittent delays remain unresolved. This is replay
evidence, not a live-provider or release result. `git diff --check` passed.
