# Proposal undo preserves historical use

6 October 2026. Lane A, answer trust. The owner confirmed that undo retires
an untouched, unreferenced work item and refuses once it has been edited or
used. No new workflow, schema, dependency or Jev call is introduced.

An accepted work item could be referenced in a different proposal decision's
input snapshot. If that decision was undone and replaced, its current snapshot
no longer necessarily contained the reference, although its undo history did.
The work-item undo query checked current decisions only and could therefore
retire previously used scope.

The reference check now also examines other decisions' preserved undo-history
entries for trigger IDs, created-record IDs and input-snapshot trigger IDs.
The existing project lock, organisation/project scope, version check and
transaction remain in force. A match refuses undo without changing either
the item or the decision. The item's own acceptance history is excluded, so
ordinary undo/reacceptance remains possible.

The new historical-reference regression first failed on the prior behaviour:
undo returned success. With the fix, the complete store suite passes (28.975 s)
and HTTP API suite passes (13.721 s), including existing edited/referenced
refusals and ordinary acceptance/undo behaviour. The test checks that refused
undo leaves the work live and the decision version/history unchanged.

The endpoint budget remains p50 <= 100 ms / p90 <= 250 ms. In
`bench/results/2026-10-06-proposal-undo-history.json`, undo passes at
8.750/10.628 ms (80 samples); acceptance passes at 7.533/8.024 ms (40 samples).
No test or compilation ran concurrently with measurement.

The overall benchmark exits 1: the separate intermittent profile slowdown
reproduced (spec-home edit 682.045/688.426 ms; integrated proposal rebuild
719.407/726.238 ms). Filing remains within 1,000/2,000 ms in both rounds.
This is not a full gate pass and does not close the proposal-generation edit
budget or M1 quality gate. The successful trace run used session slow-query
logging, whereas this failing run used the ordinary connection settings;
whether observability changes reproduce the timing difference is a remaining
diagnostic question, not an established cause or proposed production fix.
