# Remove Procurement rows with linked draft costs

Status: implemented locally, not deployed.

Outcome: removing an obsolete Procurement discipline offers to remove its
corresponding cost-plan lines in the same confirmation, avoiding a trip to Costs.
Lane A: monetary data and historical records must remain trustworthy.

The Remove action reads the current cost plan and lists explicitly assigned
lines plus their descendants. The confirmation names these lines, including
subdivisions and amounts, with Cancel or Remove and delete cost lines. Automatic
construction category matches alone do not select physical allowances for deletion.

The package PATCH accepts opt-in removal and the reviewed cost-plan ID/version.
Under the existing project lock, it checks both versions and removes the draft
lines using the same suppression and subtree rules as the cost-plan bin. Package
retirement and cost removal commit together and emit both change events. Active
scope or open delivery records still block removal and roll back all changes.
Frozen baselines and issued report history remain intact. No schema or dependency
changes; invoices, variations and scope reassignment are outside this change.

Validation: cost/package/procurement store and API regressions passed apart from
an initially invalid new test fixture, corrected and rerun successfully. New tests
cover tenant isolation, stale confirmation, scope-blocked rollback, baseline
preservation and suppression after preparation. Frontend build, the combined
removal browser scenario and existing cost-default browser scenario pass; desktop
and mobile confirmation screenshots inspected. Local combined-removal benchmark
(10 samples) p50 4.7 ms / p90 5.8 ms against 100/250 ms, enforced by a test.
