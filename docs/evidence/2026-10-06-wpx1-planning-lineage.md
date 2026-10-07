# WP-X1: planning successors retain value ownership

Lane C. Outcome: planning history cannot name an unrelated value as its
replacement. Scope is the existing successor relationship; no new history
workflow, service or dependency. Planning writes retain their 100/250 ms
p50/p90 budget and profile edits retain 50/150 ms.

The original `(org_id,superseded_by)` foreign key rejected another tenant but
accepted a successor from another key, part, ownership scope or project in
the same organisation. `SetPlanningValue` already chooses the correct row;
the finding was insufficient database enforcement, not an exposed API field.
The failing test demonstrated all four same-organisation cases.

Migration `020b_planning_lineage.sql` replaces that key with a foreign key over
organisation, site, part, planning key, scope and successor ID, backed by an
explicit unique identity. Existing project/site constraints and V1's one
project per site rule enforce project ownership. A future shared-site change
must preserve that guarantee. Null successors remain valid for withdrawals
and registry-driven scope changes, which deliberately retire the old value
without linking it to a value with a different owner.

`TestPlanningSuccessorKeepsValueOwnership` checks the legitimate two-version
chain, cross-key/part/site-project/scope/tenant rejection, retained successor
after rejection, and withdrawal. The pinned composite-key inventory replaces
the old definition and remains at 56 constraints. The actual database sweep
also checks the strengthened constraint itself, rather than counting rejection
by an unrelated constraint.

Validation: database, store and HTTP suites pass on 6 October 2026
(`.tools/wpx1-planning-lineage-tests.log`). No runtime Go change or additional
query is needed. The unchanged replay workload, including 40 planning writes,
is saved in `bench/results/2026-10-06-planning-lineage-diagnostic.json`.
Aggregate profile edits measured p50/p90 21.9/43.4 ms; rebuilds 27.9/30.1 ms;
filing 447.7/789.4 ms. The independent Spec Home edit gate still fails at
50.391/52.8 ms against 50/150 ms. Local component overruns remain reported for
target-VPS judgment. No budget was relaxed. `git diff --check` passes.

The current UUID-column inventory leaves two fields outside direct primary or
foreign keys: historical `profile_facts.passage_id` and the polymorphic
`proposal_decisions.created_record_id`. The latter is constrained through its
generated typed-reference columns. Missing historical passages are explicitly
handled in snapshot tests. Reviewing covered columns still matters: separate
tenant-scoped document and project links on facts do not by themselves prove
that the document belongs to the same project. That relationship remains to
be checked in the semantic sweep.

Follow-up: `2026-10-06-wpx1-fact-source-scope.md` records the fact/document
ownership fix and document-scoped passage lookup. The original concern above
is now covered by that migration and regression tests.

This closes the successor ownership gap; it does not claim all remaining work
packages or release gates are complete.
