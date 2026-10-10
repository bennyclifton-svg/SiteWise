# Golden RFP encoding register

10 October 2026. All encoded material remains draft; only the owner reviews it.

## Outcome and boundary

Encode the 13 prepared discipline pairs, one discipline at a time, into the
existing deterministic RFP engine. This is Lane A: priceable, traceable scope
reduces procurement decisions and preserves answer trust. The revised Section
5 tables are the source for positions. Fire engineering and fire services stay
separate. No per-item UI toggle, new AI call, service or dependency is in scope.

Existing answer keys must remain valid. Resolve/sections p90 stays below 20 ms;
PDF rendering p90 below 100 ms. Existing user-facing budgets remain read/settings
p50/p90 100/250 ms, refresh 300/1,000 ms and filing 1,000/2,000 ms.

Done means all 13 draft YAML files load, provenance resolves, representative
known/unknown/conflicting and appointment-boundary scenarios pass, and populated
RFPs fit the existing three-page gate. Quantities, interfaces and evidence
conflicts survive condensation. Unknown scope never becomes exclusion.

## Decisions

- A genuine `?` uses `if_unknown: ask`. `option` renders O, so it is reserved
  for a justified pricing assumption rather than blanket conversion of `?`.
- Existing shared defaults remain unchanged for the first five disciplines.
  Explicit per-discipline overrides accommodate certification duties, evaluation
  criteria and other evidenced differences without contradictory shared wording.
- New client choices are recorded as project flags. A flag is not automatically
  document-discovered merely because it exists; that requires a separate reading
  definition. Manual Profile answers remain authoritative.
- Focused authoring and review proceed sequentially; this register and per-
  discipline handovers retain the decisions across context changes.

## Progress

| Discipline | State |
|---|---|
| BCA / certification | Draft encoded; 11 scenarios; 2.12–2.98 pages; reviewed against ledger |
| Geotechnical | Draft encoded; 9 scenarios; 2.30-2.73 pages; reviewed against ledger |
| Land surveying | Draft encoded; 9 scenarios; 2.57-2.92 pages; set-out/final-check positions separated |
| Cost planning | Draft encoded; 8 scenarios; 2.17-2.66 pages; provenance and claims/lender separation checked |
| Access | Draft encoded; 7 scenarios; 2.36-2.83 pages; conditional positions and appointed audit extent checked |
| Waste management | Draft encoded; 9 scenarios; 2.27-2.99 pages; operational/construction and concept boundaries checked |
| Arborist | Draft encoded; 8 scenarios; 2.49-2.97 pages; inventory/report and investigation/monitoring boundaries checked |
| Fire engineering | Draft encoded; 8 scenarios; 2.07-2.95 pages; strategy/brief/FER/witnessing and services separation checked |
| Landscape architecture | Draft encoded; 9 scenarios; 1.95-2.78 pages; design/inspection and future-option boundaries checked |
| Fire services | Draft encoded; 8 scenarios; 2.27-2.95 pages; concept/calculation, dry-fire and test-owner boundaries checked |
| Mechanical services | Draft encoded; 8 scenarios; 2.23-2.96 pages; smoke/controls, condensate and seasonal/test duties checked |
| Electrical services | Draft encoded; 8 scenarios; 2.23-2.96 pages; seven allocation rows and independent declarations checked |
| Hydraulic services | Draft encoded; 8 scenarios; 2.25-3.00 pages; seven connection rows, source quality and test-owner boundaries checked |

## Encoding support

Explicit `replace: true` on a discipline's `project_rows`, `terms` or
`returnable_notes` substitutes the corresponding shared row. Project rows and
terms retain shared ordering; unmarked additions retain existing semantics.
`{procurement_route}` names the project delivery route without asserting that
this consultant is novated. Discipline `evaluation`, `evaluation_note` and
`allowances_note` override shared defaults only when supplied. Evaluation
weights must total 100 and their settings use the discipline's actual keys.
An empty complex-scale condition means no complex service-level trigger.
These changes add no AI calls and do not change the first five RFP snapshots.


`stage_included: [stage_key]` is true only for a base stage. Use it in a
position value cell to promote an optional service to I when the client actually
appoints its stage. Existing `stage` remains true for base or option, preserving
existing answer keys and option allowance schedules.

BCA follow-up: requested certification remains O until at least one Package B
stage is in the base appointment; unknown role remains a blocking question.


New appointment-choice flags carry `scope_decision: true`. The project retains
one answer, but only RFPs that use that choice print it in their conditions.
Shared physical site conditions remain visible as before. Regression coverage
checks unrelated scope choices cannot enlarge other consultants' RFPs.

Package-title suggestions now support whole phrases (for example, access
consultant and fire services). The most specific alias wins; equal-strength
matches across disciplines return no suggestion. Stable discipline IDs remain
authoritative, and suggestions still require user confirmation.

The knowledge checker now resolves numeric ledger citations, including ranges
and comma lists, so deleted or mistyped source rows fail the standard gate.

Arborist evidence questions for affected offsite trees, impacts, pruning and
documented inspection access use the existing background Profile fan-out.
They do not buy reports or establish permission to execute tree works.
Silence stays unknown; appointment choices stay client-confirmed. This follows
[TypeSafe fan-out](https://docs.typesafe.ai/patterns/fan-out) and
[confidence routing](https://docs.typesafe.ai/confidence); no serial call,
provider or confidence threshold was added. Live reading accuracy has not
been recalibrated by these deterministic tests.

Fire discipline correction: the runtime taxonomy now has separate
`consultant.fire_engineer` and `consultant.fire_services` entries; installation
remains `trade.fire_services`. The frozen 57-entry source remains untouched,
and the runtime list records the owner-requested split to 58. Existing saved
classifications are not rewritten: an old package called Fire Services with
the fire-engineer ID needs its discipline checked before choosing its RFP.
The full intake test suite passes against the test database. Recorded tests
do not establish live model accuracy for the additional choice.
The changed question set is recorded as `intake-77` in code, manifest and
threshold metadata. Existing provisional amber review cutoffs are retained;
automatic green remains disabled, and no new calibration is claimed. Intake
and evaluation suites pass using workspace-local temporary files and the
dedicated test database.

Responsibility lookup also distinguishes fire strategy, fire-services design
and installation. A legacy Fire Services title with a fire-engineer ID cannot
satisfy the separate designer exclusion. An unresolved "fire services or
electrical" alternative still asks for allocation. Regression tests cover
typed and untyped titles; original five output snapshots remain unchanged.

Supplemental Section 5 allocation schedules will remain separate document
blocks after the grouped positions. A `scope_schedule` uses the existing
provenanced row shape: unlabelled introductory instructions plus labelled
system/interface rows. This preserves electrical dry-fire and hydraulic
responsibility schedules without treating allocation instructions as I/O/X
positions. It uses the existing document table renderer and saved-report
editing model, adds no per-item toggle and leaves older templates unchanged.

## Timing evidence during encoding

Local replay benchmark (20 files x two rounds, 20 API samples, four upload
workers, two background callers, 10% injected stalls) measured filing p50/p90
565/1535 ms, RFP read 19/34 ms, settings 71/82 ms and assembly 12/19 ms.
These paths pass their budgets. Replay is not live Jev accuracy or VPS release
evidence. The overall gate does not pass: Spec Home Profile edit p50 was
63.7 ms against 50 ms (p90 89.3 ms against 150 ms). The pre-existing
7 October `bench/results/latest.json` already records this path failing at
68.0/103.7 ms. No unrelated Profile performance change or budget relaxation
was made. Raw encoding-run results are in ignored `tmp/rfp-encoding-bench.json`.

## Completion validation

All 13 new disciplines are encoded, bringing the library to 18 draft RFPs.
The 110 new scenarios bring the total to 137 populated cases. New-discipline
examples span approximately 1.95-3.00 rendered pages; every case passes the
three-page gate. Longer project-specific edits still require a fresh render.
The original five discipline YAMLs and output snapshots remain unchanged.

The knowledge checker passes with zero errors and two pre-existing package-
default warnings. Every new encoded row has provenance and its numeric ledger
references resolve. The final isolated RFP benchmark (411 samples) measured
resolve/sections p50/p90 1.00/1.00 ms and PDF rendering 47.05/62.52 ms, within
the 20/100 ms p90 budgets. All three browser RFP tests pass, including saved
edits/reset/refresh, confirmed suggestions, seven-row electrical allocation,
mobile layout and separate fire disciplines. Refresh tests now wait for the
new draft version before editing, avoiding stale-version conflicts.

Owner review is still required before issue. None of the 13 was marked reviewed.

Final repository validation: `go test -p 1 ./...` passed against the dedicated
test database; `npm run build` passed TypeScript and production bundling.
The local development app was rebuilt/restarted and its authenticated
`/api/rfp/disciplines` endpoint returned all 18 draft templates, including
separate fire-services and fire-engineering identifiers. No new dependency,
commit, deployment or issued RFP was created.
