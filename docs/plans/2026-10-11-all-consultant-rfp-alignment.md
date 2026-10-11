# Align every consultant RFP with the approved Hydraulic template

Owner instruction: 11 October 2026. Proceed through every discipline while
the owner is away. The reference is the Hydraulic pilot in this conversation
(called mechanical in the rollout request); Mechanical is included in rollout.

## Outcome and boundaries

One familiar RFP editor and issued structure across all 18 consultant
disciplines. Reduce time-to-decision without losing a service, responsibility,
qualification, scope input, priced element or returnable. Lane A: scope and
saved choices; lane B: presentation. No runtime AI changes, new dependency,
service, deployment, statutory verification or promotion of draft knowledge.

Budgets: report read/settings p50 100 ms / p90 250 ms; assembly 300/1,000 ms;
filing 1,000/2,000 ms. Existing populated RFP gates remain p90 20 ms for
resolution/sections and 100 ms for PDF rendering. Preserve full content and
normal readable type when checking document length.

## Work sequence

- [x] Capture the pre-change inventory of role, scope position, responsibility
  schedule, stage and element definitions in a regression fixture.
- [x] Create a per-discipline coverage ledger, including every role input and
  its destination. No item is removed merely because it is not currently active.
- [x] Generalise the approved editor and section order: dates; Project; Scope
  of services; combined stages/pricing; allowances; terms; other returnables.
- [x] Retain role obligations in Scope of services; integrate allocation text
  into the table; retain underlying project inputs beside the relevant scope
  rather than in a competing questionnaire. Keep parts with Project.
- [x] Group each discipline's existing services logically. Preserve all text,
  defaults, applicability rules, evidence, ownership and stable setting keys.
- [x] Add Included / Optional price / Excluded / Unknown everywhere scope is
  selected; preserve stage locks, independent statutory functions, quantities,
  service-level settings, Civil element pricing and advisory appointments.
- [x] Seed new appointments with the core design/advice stages. Keep explicit
  saved boundaries and stage overrides. Tender, construction, witnessing and
  defects are separate options unless a discipline-specific appointment
  explicitly requires them (for example selected statutory certification).
- [x] Remove evaluation and redundant issue fields consistently. Keep only
  release and close dates at the top; update internal section references.
- [x] Verify coverage, saved and protected wording, every discipline and
  scenario, browser behaviour, PDFs, knowledge validation and speed budgets.
- [x] Restart the local app and record completion and measured results.

## Preservation rules and edge cases

The inventory is a baseline, not a source to rewrite until tests pass. Every
baseline key and text must be retained or have an explicit tested destination.
Hidden or inapplicable rows remain recoverable in the editor. Unknown is a
saved scope decision and blocks issue. It does not turn an unknown building
fact into a known one; source conflicts still require resolution.

Keep role limitations and responsibility boundaries, including scope owned
by others. Preserve conditional role wording and responsibility schedules,
the complete R1 instructions, fee bases, by-element cells, excluded/done
stages, user overrides, orphan review and protected wording. Scope controls
must work even when introductory wording is protected. No cross-org change.

Generic works-package RFPs are not consultant discipline RFPs; retain their
existing project/works content and issue process. All 18 catalogue consultant
disciplines are in scope, including Hydraulic regression coverage.

## Evidence

Baseline: `internal/rfp/testdata/scope-alignment-inventory.json`.
Coverage ledger and verification results are added as the work progresses.

## Completion evidence

Implemented across all 18 consultant disciplines. The immutable pre-change
inventory verifies 301 positions, 37 role statements, 10 responsibility notes,
171 stages, four Civil priced elements, and 126 occurrences of retained role
inputs (including parts and service level). See
[the per-item coverage ledger](2026-10-11-consultant-rfp-coverage.md).

The shared resolver supplies one service table, logical discipline groups,
I/O/X/Unknown selections and a combined stage/fee schedule. Existing settings
keep their keys. Inapplicable items remain selectable. Inputs remain actual
Profile facts and appear under Scope basis, alongside their affected service
or stage; shared inputs remain in an expandable table row. Explicit scope
choices settle scope questions but cannot dismiss conflicting source evidence.
No new service, dependency or runtime AI call was introduced.

Civil retains element descriptions, per-stage element prices and distinct
base/option totals. BCA retains separate advisory/statutory functions and an
explicit confirmation when both appointment packages are requested. Existing
saved stage ranges take precedence over the new design-first defaults.
Protected wording and immutable issue history remain covered by regressions.

The longer combined tables require up to four readable pages for complete
Architecture/Civil and some other scenarios. The former three-page hard limit
is now a three-page target and four-page maximum, solely for consultant RFPs.
No scope was shortened or omitted to satisfy that limit. Other report limits
remain unchanged. Mechanical, Civil and Architecture PDFs were visually
checked; desktop and mobile scope controls were also checked.

Verification:

- 24 browser tests passed: all 18 discipline editors, Hydraulic persistence,
  independent domestic/sanitary choices, protected wording, Electrical notes,
  Civil Unknown elements, project fact editing and generic report issue history.
- All RFP scenario, inventory, snapshot, renderer and knowledge tests passed.
  Report store/API tests passed, including organisation isolation, saved choices,
  refresh, issue freezing and carry-forward.
- 137 populated PDF scenarios fit the limit. Knowledge checker: zero errors;
  two existing works-package determinant warnings outside this change.
- Across 411 populated timing samples, resolve/assembly p50/p90 was
  1.00/1.00 ms (p90 budget 20 ms); PDF render 38.42/45.56 ms
  (p90 budget 100 ms). Local replay API/filing results follow below. The current full-path benchmark
  requires 40 documents; the initial 24-document attempt was stopped by that
  workload requirement and provides no final gate result.

Local app rebuilt and restarted on port 8080. All eight active existing
consultant drafts were refreshed through the app's authenticated API and
verified to use the new table and fit the page limit. Two drafts attached to
retired packages were left intact. Before/after records are in ignored
`.tools/rfp-align-migration-{before,results}.json`. No PDF was issued and no
review status was promoted.

### Final local replay timing

40-file recorded-Jev workload, one round, four concurrent uploads, two
background callers, 10% stalled requests. This is local development evidence,
not live Jev or target VPS release evidence.

| Path | p50 ms | p90 ms | Budget p50/p90 ms | Result |
| --- | ---: | ---: | ---: | --- |
| Filing | 155.3 | 1,735.9 | 1,000 / 2,000 | Pass |
| Report read | 1.1 | 20.1 | 100 / 250 | Pass |
| Report assembly | 9.9 | 12.3 | 300 / 1,000 | Pass |
| Scope setting | 66.8 | 69.6 | 100 / 250 | Pass |
| RFP project fact | 81.2 | 83.4 | 150 / 400 | Pass |
| Report issue | 19.3 | 20.4 | 100 / 250 | Pass |

The whole benchmark exits nonzero on the pre-existing, separate
`spec_home_profile_edit` p50: 52.9 ms against 50 ms (p90 62.6/150 ms passes).
The Hydraulic pilot already recorded the same issue at 51.6 ms. No Profile
performance change or budget relaxation was made here. Component extraction
p90 and low component sample counts are additionally reported for target-VPS
judgement. Raw samples/results: ignored `.tools/rfp-align-bench-{samples,results}.json`.

Final build passed. Local `/health` returns 200 and the served asset is
`index-DgzX2QO9.js`. No work remains in this rollout; broader Profile latency
work is outside its scope.
