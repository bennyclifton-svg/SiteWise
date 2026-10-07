# WP-26: proposal engine (in progress)

6 October follow-up: the draft tenant-fitout edge now routes base-building
capacity investigations for alterations and upgrades. Full Go suites pass;
the local integrated timing gate still fails on the Spec Home edit median.
Interface determinant applicability was subsequently fixed and tested. See
`docs/evidence/2026-10-06-tenant-fitout-capacity.md` and
`docs/evidence/2026-10-06-interface-applicability.md` for scope and evidence.

Lane A. Outcome: work on one building system raises reviewable proposals for
what it affects, with stable decisions and reasons. No Jev calls, dependency,
service or UI is added. The full engine must fit profile edit p50/p90 50/150 ms
and rebuild 100/300 ms on Spec Home and 0991 before runtime integration.

## Pure implementation completed so far

- `internal/works/proposals.go`: projection and reason types; semantic keys;
  fingerprints over record content, semantic triggers, relevant determinants
  and origins, other-side existence and signal states. Work-item IDs and
  re-extracted evidence row IDs are excluded from the fingerprint. Duplicate
  semantics introduced by splitting are collapsed.
- Dismissals reopen when their fingerprint changes. Acceptance remains
  accepted with an input-change flag; this helper makes no database writes.
- D-28 severity/specificity sorting with a deterministic key tie-breaker.
  Life-safety and explicitly critical proposals remain visible beyond the
  requested top-list cap. Complete output is retained for evaluation.
- `internal/works/interfaces.go`: all nine loaded interface-consequence rules,
  directional system matching, per-part existing-system evaluation, grouped
  triggers and source reasons. Unknown existence remains visible; known absent
  or replaced targets suppress the proposal. Groups, exclusions and retired
  items neither trigger proposals nor establish existence. Another part's
  replacement does not erase this part's existing system.
- Record fingerprints include interface content and the separately decoded
  action list. YAML encoding handles the loaded boolean-keyed noul criteria;
  the existing YAML dependency is reused.
- `internal/knowledge/works_trace.go` records determinant keys, matching work
  inputs, system-presence/existence states and specificity while reusing the
  existing predicate semantics. Missing inputs remain unknown.
- `internal/works/consequences.go` evaluates the complete loaded ic/cq/uc set
  per affected part. It records relevant determinant values and origins,
  signal states and system-state reads. Known false predicates suppress the
  proposal; unknown predicates retain it. System-targeted unforeseen records
  target that system, including existing fire-water supply for hydrant work.
  Draft status is preserved and no signal automatically dismisses a proposal.
- `internal/profile/proposals.go` adapts reconciled rows and canonical work
  items. Applied stated values inherit from whole project only when no local
  row exists; explicit unknowns shadow them. Assumptions, requirements,
  allowances and forecasts cannot supply a value that suppresses a risk.
  Work-type predicates receive all applied project/part types (D-07).
- Completed-building presence now supports three-valued inputs independently
  of existing-site evidence. New work can establish completed presence without
  proving that a system already existed. Missing presence remains unknown.
- Proposals explicitly flag unaccepted triggers (D-10), independently of
  whether their knowledge record is draft. Accepting the trigger does not
  change its semantic fingerprint.
- `intake-bench -measure-proposals` is an opt-in diagnostic: after each saved
  Spec Home rebuild it reads actual canonical work items and evaluates the
  full catalogue. It records compute and rebuild-plus-diagnostic durations.
  It includes extra reads/reconciliation and excludes future projection
  writes; the command refuses combination with `-release`.
- Migration `016_proposals.sql` adds the proposal projection, canonical
  trigger relation and durable decisions. A trigger relation replaces an
  unchecked UUID array so composite foreign keys enforce tenant/project
  ownership; the API will aggregate its IDs. Decisions intentionally have no
  FK to the replaceable projection. The first accept target is work items;
  later packages will extend target types with their own constraints.
- `internal/store/proposals.go` now persists projections and their trigger
  relationships with bulk COPY in one transaction under the project lock.
  An explicit rebuild entry point supports measurement without enabling
  ordinary edits. Reads use a repeatable-read transaction so the projection
  and decision metadata are from the same snapshot.
- Dismissal checks the current fingerprint and tenant-scoped actor, stores
  the decision-time projection (`016a_proposal_history.sql`), updates the
  visible state, and uses the works revision/event path. Identical retries
  return the original ID/version. Decisions are reapplied on rebuild;
  changed inputs reopen a dismissal.

## Evidence and limits

Focused works tests pass: each table row/direction, the rooftop plant example,
existing/unknown/absent/replaced states, excluded/group/retired rows, splits,
fingerprint ordering and relevant changes, retained acceptance, ranking and
critical visibility beyond the cap.

`BenchmarkInterfaceProposals`, 100 iterations on the loaded catalogue and a
three-item synthetic capex fixture: 995,784 ns/op (about 1.0 ms mean). This is
only the interface evaluator, not a full rebuild, not a p50/p90 measurement,
and not the WP-26 step-0 gate. `go test -p 1 ./...` passed with the test database
enabled; output is retained in ignored `.tools/wp26-interface-go.log`.

The subsequent complete-engine synthetic capex benchmark evaluates all 570
loaded records against three work items in three parts. At 100 iterations it
averages 12,306,108 ns/op (12.3 ms). The fixture uses the catalogue's `refurb`
work type and is synthetic; it is not the missing real 0991 rebuild fixture.
Focused tests pass for the capex water-supply investigation, replacement vs
upgrade actions, relevant determinant origins, unchanged unrelated values,
false predicates, full-engine split stability and traced input states.
`go test -p 1 ./...` also passes after this extension with the test database
enabled (`.tools/wp26-all-proposals-go.log`).

After the profile adapter and three-valued presence extension, focused
`go test ./internal/profile ./internal/knowledge ./internal/works` passes.
The full 184-file, two-round diagnostic intake benchmark also passes existing
user-path gates (`bench/results/2026-10-05-wp26-diagnostic.json`): filing p50/p90
393.1/1278.8 ms, ordinary profile edit 37.1/62.7 ms, and current rebuild
44.5/47.2 ms. Release-only component gates remain for the VPS.

**WP-26 step-0 performance issue:** on the real saved Spec Home workload,
570 records produce 388 proposals. Adapter plus proposal computation takes
p50/p90 68.155/77.326 ms. Rebuild plus the diagnostic's extra reads and
reconciliation takes 132.968/142.869 ms, before proposal projection writes.
This is not sufficient for the 100 ms rebuild p50 budget. Runtime wiring stays
off while the evaluator is optimised; the 0991 fixture remains outstanding.

A CPU profile of the synthetic capex benchmark locates repeated fingerprint
serialization as a concrete optimisation target: `ProposalFingerprint`
accounts for 34.1% cumulative sampled CPU and YAML marshal for 29.0%; predicate
tracing accounts for 20.1%. The profile is retained in ignored
`.tools/wp26-cpu.pprof`. This profile supports precomputing record hashes; it
does not yet prove the required rebuild speed improvement.

The next optimisation adds `works.NewEvaluator`: it precomputes record hashes
once per immutable loaded catalogue and caches no project inputs or decisions.
The uncached path remains available for equivalence tests. Tests compare the
entire ordered output before and after changing values/actions; keys and
fingerprints match exactly. Synthetic capex evaluation averages 5.02 ms over
10 iterations; catalogue load plus evaluator construction averages 169.23 ms
(startup allowance 200 ms).

The repeated real Spec Home diagnostic
(`bench/results/2026-10-05-wp26-precomputed.json`) reduces proposal compute to
p50/p90 33.328/37.036 ms and rebuild-plus-diagnostic to 99.676/103.639 ms.
Current rebuild alone is 44.070/46.657 ms and ordinary profile edit is
38.873/64.444 ms. Existing user-path gates pass. The diagnostic p50 is narrowly
within 100 ms; projection writes and the 0991 workload are still unmeasured,
so this does not yet authorise claiming WP-26 step 0 complete or Verified.
`go test -p 1 ./...` passes after precomputation and the profile adapter, with
the test database enabled (`.tools/wp26-precomputed-go.log`).

The follow-up run with the bounded manual 0991 fixture records 0991 proposal
compute p50/p90 5.002/5.331 ms and rebuild-plus-diagnostic 11.789/12.520 ms.
Spec Home in that run is 100.607/110.135 ms, confirming that its diagnostic
remains marginal at the 100 ms p50 budget. No runtime projection writes have
been enabled. See WP-28 evidence for the manual fixture's limitations.

The next optimisation removes per-predicate determinant-map copies, computes
an interface proposal's fingerprint only after collecting all its triggers,
and skips unused explanation-building for predicates known false. Unknown
predicates still get complete reasons. Focused knowledge/profile/works tests
pass. The compiled synthetic capex benchmark averages 2.94 ms (100 iterations).

The same full intake/Spec Home/0991 diagnostic run
(`bench/results/2026-10-05-wp26-traced.json`) records Spec Home proposal compute
p50/p90 24.182/29.584 ms and rebuild-plus-diagnostic 92.415/96.602 ms. The
0991 rebuild-plus-diagnostic is 10.078/11.058 ms. Existing user-path gates pass;
current ordinary edits are 38.0/64.1 ms and current rebuilds 44.5/47.0 ms.
Projection write timing is still required before completing runtime wiring.

The rollback-only migration test applies the actual DDL and checks FK/check
error codes: foreign projects, wrong sites/parts, foreign or other-project
triggers, foreign actors and created work items are rejected. Missing accept
types and created records on dismissals are rejected. Deleting a projection
retains its decision, and a decision prevents hard deletion of its trigger.
The full `go test -p 1 ./...` suite passes with the test database after these
changes (`.tools/wp26-schema-go.log`).

Persistence integration tests pass for stored trigger IDs, foreign reads and
actors, stale fingerprints, concurrent idempotent dismissals, disappearance
and reappearance of a knowledge interface, input-driven reopening, and
transaction rollback after a projection COPY constraint failure. The old
projection and durable decision survive that failed replacement.

## Transactional acceptance and cascade correction

The first persistence-wide suite exposed an organisation-deletion regression:
an immediate audit actor FK was checked before the project cascade removed its
decisions. Migration `016b` defers the restrictive actor/work references until
transaction completion. The migration test forces deferred checks for invalid
references, proves standalone referenced-user/work deletion fails, and proves
organisation deletion removes projections, triggers and decisions together.
The full Go suite passes after the fix and acceptance implementation
(`.tools/wp26-accept-go.log`).

`AcceptProposal` now serialises with rebuilds, checks tenant/actor/fingerprint,
and atomically creates a calculation-origin work item accepted for planning
plus its decision. It preserves the full projection as provenance and the
decision-time snapshot. Matching retries return the same created item even if
the projection subsequently changes or disappears. Unsupported kinds and
missing explicit targets/actions fail closed. Integration coverage includes
concurrent retries, foreign actors/projects and stale fingerprints. This is
not yet exposed through an API or measured as a user-facing route.

The diagnostic now separately measures profile rebuild plus explicit proposal
persistence, including repeated input reads/reconciliation and bulk projection
writes. It remains a two-transaction diagnostic, not integrated-edit or
release evidence.

The persisted diagnostic completed (`bench/results/2026-10-05-wp26-persisted.json`):
Spec Home p50/p90 146.659/165.950 ms, above the rebuild p50 budget of 100 ms;
manual 0991 24.994/30.841 ms. No runtime edit wiring is enabled. The measurement
includes a second snapshot/reconciliation and transaction, so it identifies
remaining cost without establishing the final integrated-path timing. Current
ordinary profile edits remain 39.8/65.4 ms, rebuilds 46.9/52.1 ms and intake
326.9/668.1 ms; current user-path gates pass. Development-host component budgets
for text extraction and deterministic field rules are over their target-VPS
limits; this is not release evidence. The added retry-after-record-removal
acceptance test passes separately after the full suite.

## Remaining

GET proposals is implemented with `show=all` and a configured default count
from `data/profile/proposals.json` (10). Critical and life-safety proposals
remain visible beyond the cap; full counts and decision snapshots are retained.
The serve command validates the policy at startup. Works, HTTP and command
tests pass, including configured listing, ordering and foreign-tenant reads.

Owner clarification received in this chat: undoing acceptance retires the
created item only while untouched and unreferenced; otherwise it is refused.
Undo implementation must preserve its history and support stale-version
conflicts. This resolves the previously unspecified consequence of undo.

Undo is now implemented in the store and DELETE API. Migration 016c retains
undo snapshots, records the created work version and marks the decision
inactive without resetting its version. Accepted work is retired only at the
recorded version and while no child or other authoritative proposal decision
references it. Derived projections are replaceable and do not constitute a
person using the item. Reacceptance revives the same item only when its current
version exactly matches an undo record; changed/independently retired items
cannot be revived through this path. Dismissals can also be undone. The focused
HTTP test covers stale undo, retirement, inactive decisions, reacceptance and
an intervening dismissal. WP-31/33/35/41 must extend the reference blocker as
their responsibility/cost/delivery/report tables are introduced.

Accept/dismiss HTTP routes now validate membership, project ownership, origin,
one strict JSON object, fingerprint shape and rationale length. Stale inputs
return 409; unavailable targets return 422. The focused HTTP integration test
passes for foreign tenants/origins, malformed bodies, stale inputs, dismissal
followed by acceptance, repeated acceptance and zero Jev calls. GET/undo and
route timing remain outstanding.

An incremental-write experiment was rejected and removed: comparing stored
projections before writing increased Spec Home diagnostic p50 to 160.719 ms
(p90 173.309 ms). The 0991 manual fixture improved to 17.693/20.328 ms, which
does not justify the large-fixture regression. The diagnostic used 20 intake
files per round, with the full saved profile fixtures and 40 profile samples;
it is not full-corpus release evidence. Results are retained in
`bench/results/2026-10-05-wp26-incremental-diagnostic.json`.

Temporary phase instrumentation (removed after diagnosis) measured repeated
snapshot reads around 22 ms, evaluation 27–33 ms, proposal deletion 17–19 ms,
projection insertion 18–21 ms and trigger insertion 15–17 ms. A second
experiment used JSON recordset upserts and retained unchanged trigger links.
It also regressed the diagnostic: Spec Home 155.970/171.739 ms and manual 0991
31.735/34.722 ms (`2026-10-05-wp26-sql-diagnostic.json`, same reduced intake
workload). It was removed, retaining the simpler bulk replacement baseline.
These experiments do not pass the timing gate and do not enable runtime wiring.

Complete rebuild measurement on
Spec Home and the full-pipeline 0991 fixture, undo semantics,
APIs and remaining integration/tenant-isolation tests remain. The
interface evaluator is deliberately not called by the runtime rebuild yet.
Signal evidence application is WP-27. Other accept targets arrive with their
packages. Draft records remain draft; this work does not verify legal rules.

WP-26 remains in progress. Its migration has been tested locally; no merge
or production deployment has been performed.

## Decision endpoint timing and confirmed undo policy

The owner's confirmed rule is implemented: undo retires the created work item
only while untouched and unreferenced, and otherwise returns a conflict without
changing the item or decision. Existing store/API regressions cover edits,
references, optimistic versions, retries and stable identity after reacceptance.
Package, scope, delivery and saved-report references added by later packages
also participate in the relevant blockers. Future cost references still need
their corresponding blocker when that package is implemented.

`intake-bench` now measures proposal reads, acceptance, dismissal and undo as
four independent 100/250 ms paths. A synthetic rooftop-plant scenario is
explicitly projected before timing; ordinary runtime rebuilds still do not
invoke the proposal evaluator. Forty cycles exercise real authenticated HTTP
decisions, check the saved decision, undo it, check that it is open again and
verify that accepted work is no longer active. Reacceptance must use the same
item ID. A harness regression checks that each operation actually contributes
its required samples. The command and latency tests pass
(`.tools/wp26-proposal-bench-tests.log`).

`bench/results/2026-10-05-proposal-decisions-diagnostic.json` records:

| Path | Samples | p50 ms | p90 ms |
|---|---:|---:|---:|
| Proposal read | 161 | 4.779 | 5.979 |
| Accept | 40 | 6.524 | 7.021 |
| Dismiss | 40 | 6.999 | 8.003 |
| Undo | 80 | 10.532 | 11.937 |

These endpoint budgets pass. This closes the missing basic decision-route
timing coverage, not the larger projection or milestone gates. The overall
diagnostic correctly exits 1 because independent Spec Home edits measure
58.168 ms p50 against 50 ms. Earlier aggregate edit pass statements above do
not establish that workload's target; see the profile-workload-gate evidence.
The reduced local replay also reports component overruns reserved for VPS
judgement. Full-corpus/live release, proposal generation and 0991 accuracy
remain unproven. No service, dependency or additional runtime AI call was added.

## Projection update follow-up, 6 October

The owner's confirmed undo rule now also guards historical use retained in
other proposal decisions' undo histories. See
[the regression and validation evidence](2026-10-06-proposal-undo-history.md).

Projection persistence now updates changed proposal fields and synchronizes
trigger links independently instead of deleting every row. The unchanged
Spec Home diagnostic measures persisted rebuild p50/p90 113.452/124.401 ms;
the median still exceeds 100 ms, and runtime generation remains disabled.
See [projection update evidence](2026-10-06-proposal-projection-updates.md)
for full-row retention, trigger replacement, rollback checks and the rejected
selective-delete experiment. This does not close missing CQ/UC target/action
semantics or owner-reviewed quality gates.
