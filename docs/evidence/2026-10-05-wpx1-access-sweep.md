# WP-X1 access-control sweep — first M1 route matrix

Lane C. Outcome: a regression test proves that requests cannot read or change
another organisation's or project's planning records. This protects answer
trust without adding a runtime service or dependency. No production path was
changed, so existing API latency budgets and measurements remain applicable.
This is partial WP-X1 evidence, not completion of the full recovery/security gate.

`internal/httpapi/access_sweep_test.go` creates real work, a services package and
stages, responsibility scope, an approval, evaluated proposals and a saved
dismissal, an assembled RFP with a protected edit, and a planning value. It
exercises 37 method/path combinations covering works, gaps, packages/stages/
scope, delivery, proposal decisions, reports/draft/edits, site, profile/source
reads, profile edits/read request, parts and planning values.

The 116 rejection cases check signed-out access (401), wrong-origin writes
(403), foreign-organisation access (404), and cross-project/site targets (404)
where a request actually contains a target owned by the original project.
Same-organisation project collection reads and unconstrained creates are not
misclassified as forbidden merely because they address another owned project.
Responses are checked for fixture IDs and private wording. Before/after
snapshots of 20 authoritative tables across both organisations include versions,
histories, projections, protected edits and durable events; all remain unchanged.
The complete sweep makes zero Jev calls.

The V1 schema enforces `projects_one_per_site_v1`. The test verifies that a
second project cannot be linked to the same site, rather than dropping the
constraint to fabricate an impossible fixture. Both cross-project and
cross-site checks therefore use a real second project on a different site.
If the product later supports shared-site interventions, add that distinct
matrix when its schema changes.

Validation: the focused matrix passes (`.tools/wpx1-route-sweep.log`), and the
full HTTP suite passes (`.tools/wpx1-httpapi-tests.log`). `git diff --check`
passes. Initial fixture errors (V1 site uniqueness and the scope value spelling)
were corrected to match the actual contract; no production authorization defect
was exposed by this matrix.

## Document batches and nested targets

The matrix now covers 39 method/path combinations and 148 rejection cases.
It adds document reading-policy and deletion batches, with real saved source
records. Each mixed batch combines an allowed document with a document from
another project, another organisation, or a nonexistent document, in both
orders. Nested substitutions test stages and scope under the wrong package,
work from another project, stage/work references on delivery and proposal
acceptance, and package blocks outside the addressed report. All reject.

The before/after snapshots now cover 30 tables across both organisations,
including files, documents, source passages, reading calls/evidence and jobs.
No captured row, version, protected edit, projection, queue or event changes.
The updated focused matrix passes (`.tools/wpx1-nested-sweep.log`); report,
store and full HTTP suites pass (`.tools/wpx1-nested-tests.log`).

This extension exposed one response-contract defect: resetting a protected
edit for a block outside the report returned 422, whereas editing it returned
404. The pure reset helper now distinguishes a missing edit, and the store
maps that error to not-found. Existing-target invalid states still return 422.
No data leak or partial mutation was observed. The failing nested request is
retained as the regression test. No extra query, dependency or Jev call.

The unchanged benchmark workload was rerun after the fix:
`bench/results/2026-10-05-wpx1-reset-diagnostic.json`. Report writes, including
40 resets, measured p50/p90 5.256/5.680 ms against 100/250 ms; report assembly
8.955/9.897 ms against 300/1000 ms; filing 458.553/693.663 ms against 1000/2000 ms.
The overall gate still correctly fails: Spec Home edits measured 51.543/53.691
ms against 50/150 ms, and local component overruns remain separately reported
for target-VPS judgement. This is replayed local diagnostic evidence only.

The 51 new/changed composite foreign keys are now covered separately in
`2026-10-06-wpx1-foreign-keys.md`. Still required for WP-X1: semantic review of
relationships without foreign keys and issue recovery once that M2 endpoint
exists. Reading job recovery passes (`2026-10-06-wpx1-reading-crash-recovery.md`). Extraction
worker recovery now passes (`2026-10-06-wpx1-worker-crash-recovery.md`). Assembly process-crash
recovery now passes; see `2026-10-06-wpx1-report-crash-recovery.md`. Existing
individual package isolation tests supplement this matrix but do not stand in
for those remaining requirements. No live/VPS, restore or release claim.
