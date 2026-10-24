# Profile update correction and Chen recheck

Date: 24 October 2026. Lane A: trusted profile values. Owner authorised the
12-document Jev recheck. No new service, dependency or AI provider.

## Defects and corrections

- Header questions classified consultant services and fee terms as building
  category/work type. Questions now explicitly identify the physical project.
- A consultant's exclusions and budget exclusions became project exclusions.
  Scope questions distinguish these, completed investigations, alternatives,
  unaccepted quotes and whole-system versus partial supply.
- The area parser placed an ASCII word boundary after the superscript in m²,
  dropping valid area candidates. Fixed, and added the literal lot-area cue.
- The council question rejected the council named for the project's own DA.
- Programme milestones had no extraction path. Dated narrative targets from
  programme documents are now offered verbatim in the existing label fan-out;
  qualifications are retained. Activity tables stay with the Program reader.
- An explicit update requeued old labels but left completed evidence jobs done.
  The transaction now requeues both stages, with evidence waiting for labels.
  The regression test reproduced the stale evidence before the fix.

Final question version: profile-9; read revision: read-12. Existing provisional
floors remain provisional; green is not enabled. User values remain final.
Only the requested project's update was initiated. No direct edits were made
to its profile values to make the result pass.

## Live outcome

All 12 documents completed, with no failures or skips. All 1,306 saved profile
facts use profile-9. Residential/new-build false conflicts are gone; site area,
GFA, council and qualified programme targets are populated. False project
exclusions, unsupported pile selection and whole-lighting owner allocation are
gone. The separate owner-supply and budget statements remain intact.

Some systems are still taxonomy defaults, explicitly shown as suggestions,
not document-selected construction solutions. In particular a suggested slab
must not be mistaken for an engineer's footing selection. Stormwater provider
remains unknown because the quote does not establish appointment.

Private before/after snapshots and test inputs stay outside Git:
`tmp/chen1-profile-before-fix.json`, `tmp/chen1-profile-after.json`,
`data/eval/profile/private/chen1-facts.json`. The private expectation key is
agent-drafted and unreviewed, not owner-approved calibration.

## Verification

- Area, council, programme routing/qualification and stale-evidence regressions
  pass. The actual saved Chen readings pass the code-only answer-key replay.
- Profile suite excluding the separately existing Meridian case passes.
  The broader run retains the Meridian acoustic-signal failure seen before
  this fix. Jobs suite has four unrelated hold-point fixture failures.
- Knowledge checker: zero errors, two existing package-default warnings.
- Intake suite passes against the test database.
- Local profile GET, 20 samples: p50 11.76 ms, p90 12.97 ms; budget 50/150 ms.
- Explicit update acknowledgement: 25.42 ms (one observation, not percentiles).
  Full background refresh: 36.84 seconds, including queue time; this is not
  the filing or saved-profile read budget.

Final isolated local benchmark (recorded Jev latency, not live Jev or VPS
release evidence):

| Path | Samples | p50 ms | p90 ms | Budget ms |
| --- | ---: | ---: | ---: | ---: |
| Filing | 24 | 515.35 | 1403.85 | 1000 / 2000 |
| Saved profile read | 40 | 1.10 | 9.20 | 50 / 150 |
| Profile edit | 80 | 31.20 | 103.23 | 50 / 150 |
| Code-only profile rebuild | 20 | 26.65 | 32.96 | 100 / 300 |
| Existing large specification-home profile edit fixture | 20 | 57.78 | 66.35 | 50 / 150 |

The final gate still reports the last row's median miss. The owner explicitly
asked not to block this accuracy correction on that existing miss. No budget
threshold or gate was weakened. The focused harness omitted unrelated surface
benchmarks; raw results are in ignored `tmp/chen1-isolated-results.json`.

The live background refresh considered 12 documents and 103 passages. The
owner supports a workload-dependent budget for this refresh. A future budget
should distinguish fixed overhead, file count, passage count and text volume,
while retaining the fixed saved-profile interaction budgets. This run supplies
one workload observation, not enough to fit credible p50/p90 scaling
coefficients. No numerical refresh formula is asserted from this one sample.

The existing background states and fan-outs are retained, with no new serial
Jev stage. TypeSafe guidance used:
[fan-out](https://docs.typesafe.ai/patterns/fan-out),
[literal conditions and bounded state](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
[confidence](https://docs.typesafe.ai/confidence),
[API](https://docs.typesafe.ai/api/system-one).
