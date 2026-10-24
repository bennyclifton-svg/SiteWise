# Automatic proposal package matching

24 October 2026. Lane A: remove redundant package selection while preserving
the user's decisions. No new dependency, service, AI call or schema change.

Previously, an exact discipline match was applied only during proposal reading.
Packages created later, or a corrected filing discipline, left the proposal
unassigned. Now package writes, discipline corrections, filing/OCR completion
and proposal-read completion reconcile unassigned received/review proposals.

Matching uses exactly one active package with the same saved discipline, in the
same organisation and project. Retired/cancelled packages do not qualify.
Multiple matches remain unassigned. Existing assignments and explicit manual
blanks stand. Matching records rule provenance and emits a procurement event
in the same transaction. Project locking serialises package/discipline changes
with manual assignment and read completion. Price cells, awards and appointment
status are not inferred by this operation.

## Verification

- Regression coverage: discipline correction, package creation, package
  discipline change, duplicate-match ambiguity resolved by retirement,
  package changes during a read, manual assignment, manual blank and wrong-org
  correction rejection.
- Relevant store and HTTP API suites pass. An earlier broader test-name filter
  also selected an unrelated Program preparation fixture that fails its
  expected activity assertion; this was not changed.
- Final twenty matching writes per path: discipline correction p50 6.10 ms / p90
  9.00 ms against 50/150 ms; package edit 2.99/3.77 ms against 100/250 ms.
  These regression tests enforce the budgets.
- Local HTTP filing benchmark: 24 files, four concurrent uploads, recorded
  Jev latency, two background callers and 10% injected stalls: p50 595.1 ms /
  p90 1343.6 ms against 1000/2000 ms. HTTP correction: 5.3/7.0 ms.
  This is replay evidence, not a live provider or VPS release measurement.
  The focused harness intentionally omitted other surfaces; its overall gate
  reports their missing samples and the existing large-profile median miss
  (52.0 ms against 50 ms). No thresholds were relaxed.

## Chen Residence 1

The running local server was updated and the actual procurement board re-read.
It contains zero unassigned proposals: two Harrison Clarke documents under
Architect and the stormwater quote under Hydraulic. All three assignments were
already manual before this fix and retain user provenance afterwards. No live
assignment was cleared to stage a demonstration. Automatic transitions are
verified with isolated test records.

Private board snapshot and benchmark outputs remain ignored under
`tmp/chen1-procurement-after-matching.json` and `tmp/proposal-matching-*`.
