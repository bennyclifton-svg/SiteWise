# K1 explicit layout-change input proposal — 7 October 2026

Implementation choice made by the lead under the owner’s 7 October full-completion authorization; implementation and checks in progress. This does not mark knowledge owner-reviewed. Intended outcome: a wall
lining repair must not generate partition/HVAC coordination, while a user-stated
room-layout change can raise a traceable review without splitting the physical
system taxonomy or adding a Jev question. Lane A; edit/rebuild 50/150 and 100/300
ms and zero extra runtime AI calls. This is a narrow resolution of the original
K1 partition requirement, not a new generic attributes framework.

## Input and persistence

Add an optional tri-state `layout_change` boolean to an individual work item:
`true` = room/space boundaries change; `false` = they do not; null/missing = not
recorded. Use a nullable dedicated `work_items.layout_change` column and the
existing version, user-touch and actor/time provenance. A checked enum with
unknown/yes/no is also possible, but don't store both representations. Do not
put it in a site-wide determinant, generic target wording or project planning
value: one partition item may change boundaries while another only replaces
lining. No title-based inference, automatic false, or default based on `alter`.

Initially permit authoring only for included leaf work on
`interiors.walls-linings`, action new/alter/remove. Reject incompatible non-null
values at API boundaries so stale edits changing system/action cannot leave
misleading context. Read old rows as null. Preserve historical value in report
basis. When splitting, child context starts unknown unless explicitly supplied;
parent grouping cannot propagate a layout assertion to every child. Retiring
or excluding the item makes its context ineligible as with other work evidence.

## UI and provenance

In the existing Work editor show “Does this work change room or space
boundaries?” with Not recorded / Yes / No, only for the eligible physical scope.
Explain that changing boards/linings without moving boundaries is No, and that
Yes requests coordination rather than proving any mechanical design outcome.
Save through the current optimistic work correction endpoint. Show the saved
answer and actor in work detail; changing it invalidates proposal fingerprints
and protected generated report content through the existing works revision.
No additional document extraction field or live recording is required.

## Predicate and first bounded graph use

Extend the existing `works` predicate with optional `layout_change: true|false`,
not a new general expression language. Evaluate action, system and layout value
against the **same triggering item**. Missing context is Unknown, a different
record's true answer cannot satisfy it, and explicit false suppresses a true
predicate. Trace the exact work ID and saved layout value in proposal reasons
and record fingerprints. Interface applicability must be evaluated on the
triggering source item rather than an unbound part-wide list, otherwise two
lining items in one part can contaminate each other's result.

The first candidate edge is source `interiors.walls-linings` to receiving
`mechanical.air-distribution`, limited to a recorded boundary change. A draft
coordination relationship can request review of air paths/zoning/return air;
it must not claim capacity failure or prescribe equipment replacement. Verify
whether the existing shares-space consequence wording is sufficient; if not,
record a narrowly scoped consequence rather than misuse a controls/loads type.
An unknown layout answer should be shown as applicability needing information,
never accepted as proven room alteration or an issued standard obligation.
Smoke-control/thermal-performance impacts require their own source and endpoint
analysis and are not automatically closed by the first airflow edge.

## Acceptance before merging

- Yes on eligible source + existing receiving system generates the intended
  draft review with exact work/part provenance; explicit No does not.
- Unknown stays unresolved; excluded/retired/group/lining-only and unrelated
  systems do not create a confirmed obligation.
- Two same-system sibling items with different answers do not share truth;
  different parts do not leak context. Missing/absent/removed receiving system
  follows existing interface target rules.
- API invalid values, wrong organisation, stale version and system/action changes
  are handled; split does not silently copy assertion; report snapshot preserves
  the input and changes are detected on refresh.
- Existing classifier/live question fingerprints remain unchanged; pure evaluator,
  DB/API/browser checks and the original integrated speed gate pass. No new
  dependency or service.

The lead chose this dedicated input approach in the plan’s
7 October implementation decision record. The user's broad implementation authorization permits
that routine design decision, but it should be visible and reviewable in prose;
no additional owner permission is inherently required for the reversible change.

## Source for bounded room-layout coordination

[CIBSE Journal Module 227](https://www.cibsejournal.com/cpd/modules/2023-12-fcui/),
November 2023, was read on 7 October 2026. Its fit-out discussion describes
interior reconfiguration changing heating/cooling demands and whether existing
fan coils serve the new rooms appropriately; later duct/diffuser adaptation
serves partitioned spaces. This supports reviewing distribution and zoning,
not a universal plant replacement, compliance claim or choice of its featured
product. Mechanical-services-guide coordination provides the local seed anchor.
Smoke-control and statutory thermal-performance requirements are separate topics.

## Recorded implementation decision

The lead accepted this design under the owner full-implementation instruction. Persistence/API use the explicit enum `unknown`/`yes`/`no`, default unknown, rather than SQL NULL. Omitted/empty legacy input normalizes to unknown; null PATCH follows existing pointer omission semantics. Predicate literals are quoted `yes`/`no`. The initial bounded knowledge output is a draft planning-review CQ with same-item guard; it does not claim every K1 graph relationship complete. No new Jev question.

## Implementation and verification — 7 October 2026

Implemented under the lead's recorded enum decision. Migration 021f adds the
unknown/yes/no field and database checks; model and API validate compatible
wall/partition actions. Work writes retain optimistic versions and actor/time
provenance; reads, report basis and wording include the saved answer. Split
children default unknown rather than inheriting a parent's yes. The existing
editor displays a tri-state question and clears context when switching to an
incompatible action. Null PATCH follows omitted-field semantics; explicit
unknown clears the answer. No new Jev question, determinant or dependency.

The initial `cq.room-layout-air-distribution-review` is a draft hold-point
review of existing distribution/zoning, supported by the mechanical seed and
the source assessment above. It uses the same-item guard plus existing air
system evidence. Unknowns remain in the proposal reason; absence suppresses it.
This does not implement smoke-control, legal thermal requirements, or every
missing interface edge. Detailed K1 remainder stays in the interface audit.

Verification:

- Focused works/knowledge predicate, trace and fingerprint tests pass (0.672 /
  0.940 s): yes/no/unknown, excluded scope, absent target and mixed-action/system
  sibling isolation. Context changes alter the saved proposal fingerprint.
- Store tests including split persistence and the PMP known-range regression
  pass (0.821 s). The first focused run exposed the explicit work-list SELECT
  missing the new field; corrected and rerun green before browser validation.
- HTTP boundary test passes (0.549 s): wrong org, invalid enum/type, stale
  version, incompatible action and explicit clear; store additionally tests
  null omission. Work creation defaults to unknown.
- TypeScript/Vite build passes. `work-layout.spec.ts` passes, 1 test (8.3 s
  including server startup; interaction 1.0 s): unknown initial input, yes save
  visible and persisted, changing action to repair clears it. Initial sandbox
  cache access failed; approved outside-sandbox rerun succeeded.
- Strict knowledge checker passes: 0 errors, same 2 existing mapping warnings,
  8 CQ, 2,000 ledger rows and 0 pending. No classification question changed.

These elapsed test durations are not endpoint percentiles. The existing
works_write 100/250 ms and integrated edit/rebuild gates remain the lead's
final timing responsibility; no budget relaxation or release pass is claimed.

## POST boundary correction

The subsequent benchmark's authored-yes create exposed an omission in the
HTTP POST DTO: the store/model supported the field, but `postWork` neither
decoded nor forwarded it. The earlier API/browser checks created an unknown
item then authored Yes through PATCH, so they did not exercise this independent
create boundary. Fixed the explicit DTO and forwarding. Added
`TestWorkLayoutChangeCreateAPIBoundary` for authored-yes create/read persistence,
wrong organisation, invalid enum/type and incompatible action/system. Both
POST and PATCH API tests now pass (0.914 s). No Jev calls occur. This correction
is included in the final timing candidate; earlier focused results alone did
not establish POST completeness.

## Bounded smoke and thermal review decision

The authored same-item layout predicate now permits two further draft reviews:
changed boundaries plus existing fire-mode air systems reviews the smoke strategy
and air paths; changed boundaries plus existing air conditioning reviews heat-load
and thermal zoning design basis. Mechanical seed Heating and cooling / Smoke
control and fire interfaces, and fire seed Egress and occupant safety support
reviewing these interfaces. Neither review calculates loads, asserts a statutory
upgrade, changes a smoke zone, or selects equipment. Missing layout/existence
remains unknown. No new detector/question or physical target is introduced.

The two later K1 CQs pass10 scope tests (yes/no/unknown, absent existing target and excluded work). Combined K1/K3 works tests1.018s. Checker0errors2existingwarnings; catalogue13CQs.
