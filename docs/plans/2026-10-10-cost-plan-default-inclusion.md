# Cost plan: four sections and automatic inclusion

Date: 10 October 2026. Status: approved and implemented locally; final verification recorded below.

The owner confirmed the revised workflow, approved both legacy-row placement
and group-deletion confirmation, and then authorised implementation. This
plan supersedes the Keep/Dismiss workflow in
[the original autofill plan](2026-10-10-cost-plan-autofill.md).

## Outcome and scope

The project user opens Costs and sees an ordinary, editable cost plan derived
from the available profile, work items and procurement packages. No acceptance
step is required. Four sections follow the owner's preferred ledger order;
unwanted lines are removed using a bin at the far right of the row.

This reduces time-to-decision and preserves trust in totals: every displayed
cost line is saved and participates in the existing accounting rules. An
unpriced line remains unknown, not zero or an invented estimate.

**Lane A:** inclusion, totals, deletion and baseline correctness are product
core, even though section names and order are presentation changes.

Retain the existing paper-workbench table, building-system model, line kinds,
money arithmetic, inline editing, quantity calculation and construction scope.
No new dependency, service, Jev call, rates, automatic prices, procurement
deduplication, tender workflow, or live-project data edit is included.

## 1. Four sections

Use a deterministic presentation section, separate from the existing storage
line kind. Do not rename `fee` to mean an authority fee in the database.

| Order | Heading | Existing content |
|---|---|---|
| 1 | Fees and charges | `project_wide` / `authority_fees`: DA contributions, certificate and authority charges, service connections |
| 2 | Consultants | `fee`: existing consultant/service-package lines |
| 3 | Construction | `works`: existing systems, groups and children |
| 4 | Contingency | `project_wide` / `contingency` and `escalation`: design contingency, construction contingency, escalation |

The screenshot guides the layout; the owner's written allocation controls
where consultants belong. Existing codes, names, assignments and prices are
preserved. Section order is fixed; sorting remains within sections and tree
levels. Children stay with their parent.

**Legacy categories:** existing `other` and `exclusions` lines must remain
visible. Proposed compatibility rule: place these at the end of Contingency,
retaining their actual category label and excluded state. This keeps exactly
four sections without pretending those lines are contingencies. Do not add
new default categories or silently discard legacy rows.

Use one server-side section classifier for both returned row membership and
section totals. Return additive section totals without changing the meaning
of existing `fees`, `works` and `project_wide` API totals used elsewhere.
Reuse the current exact-money, range, unknown and excluded-line rules;
never sum money with browser floating point. Group headings are not counted
as posting rows. Each posting row is counted once in its section and overall.

Update the inline add-row defaults: authority fees in Fees and charges,
service package in Consultants, building system in Construction, contingency
in Contingency (with escalation available). Category changes move the row
and update the correct subtotals together.

## 2. Save default lines automatically

Current evidence: `ReadCostPlan` is read-only; `CostGrid` overlays unsaved
suggestions; `AcceptCostSuggestions` already supports batched insertion with
source identity, project locking and duplicate prevention. Merely hiding
Keep/Dismiss would leave rows absent from persisted totals and baselines.

Recommended approach: an explicit, automatic prepare operation when opening
the editable Costs view. Keep GETs read-only and avoid a browser chain of
read → accept → read → totals.

1. Add an authenticated `POST /projects/{id}/cost-plan/prepare` operation.
   It takes no client-supplied line contents or prices. Under the existing
   project lock, load the current draft or initialise one, derive applicable
   source lines, and insert only missing, non-deleted lines in a batch.
2. Extract the existing insertion logic for reuse rather than calling one
   HTTP handler from another. Preserve source keys, provenance, parent-first
   insertion, manual-overlap suppression and protection against splitting an
   already priced posting line.
3. Return the saved plan and its totals from the same transaction snapshot.
   Publish one revision/event only when saved state changes. Repeated prepare
   requests with unchanged inputs are no-ops, including multiple tabs and
   retries after a lost response.
4. Replace the editable screen's initial plan/totals pair with this single
   operation. Existing package/work selectors may continue their independent
   reads. Render editable rows only after preparation succeeds; a failure
   shows a retryable load error, never apparently saved suggestion rows.
5. Re-run preparation on relevant profile, work or package refresh events
   while Costs is open and on reopening it. Deduplicate in-flight refreshes;
   a no-op must not create an SSE refresh loop. Retain unsaved field input
   when a refresh arrives. Ordinary edits retain explicit version checks.
6. Reconcile missing defaults inside the baseline transaction before freezing,
   so a direct freeze cannot omit applicable lines. Historical baseline GETs
   never reconcile, create, or delete anything.

This scope prepares the current draft when Costs is opened/refreshed or
baselined. It does not introduce background jobs or writes on report GETs.
Other readers continue to consume saved draft state; issued documents and
historical snapshots retain their existing semantics.

Preserve existing edited rows and values even if their source later retires
or becomes inapplicable. Newly applicable sources may add blank rows.
Unknown starter conditions retain the current conservative inclusion and
rationale; known-false conditions do not create a line. Automatically
including a line does not promote draft knowledge to reviewed evidence.

## 3. Replace acceptance controls with deletion

Remove Keep, Dismiss, Add all suggested, Show dismissed and the suggested-row
badge/style from the main workflow. All saved rows use the normal editors;
source rationale remains available in the drawer without asking for approval.

Add a final Actions column after Remaining with a bin button on editable rows.
Use the existing icon conventions, an accessible name such as `Delete DA
fees`, a tooltip, visible keyboard focus and a usable touch target. The button
must not open the drawer. Adjust all header, group and subtotal column spans,
including quantity mode. Keep the action reachable through the existing table
scrolling at narrow widths; avoid page-wide horizontal overflow.

Leaf deletion uses the existing DELETE item route. Remove its current-draft
revision, values and scope links atomically; preserve stable identities,
historical baseline rows, source packages/work items and issued documents.
Record the source key in the existing suppression table so preparation cannot
recreate it. Retain all previously dismissed keys on upgrade.

**Group deletion proposal:** the bin on a parent deletes its complete current
draft subtree. Show one explicit confirmation naming the group and number of
child lines, because that one click affects multiple rows. Ordinary leaf
deletion has no confirmation step. Suppress the parent and descendant source
keys, including future derived children under that deleted source parent.
Deleting only a child must not delete its siblings or redistribute money.
Keep an empty parent until the user deletes it; do not invent a replacement
allowance or reallocate the deleted amount.

On success, update rows and totals consistently and close a drawer showing a
deleted row. On failure, retain the row and show the error. Disable repeated
actions while saving; stale writes cannot delete newly changed content.
Baseline/history views have no delete controls and remain server-protected.

## 4. Implementation sequence

1. **Section rules and totals:** add the shared classifier and additive section
   totals in `internal/costs`; define legacy and parent/child behaviour in tests.
2. **Automatic preparation:** reuse `internal/store/cost_suggest.go` insertion
   logic, add the transaction and API route, update baseline reconciliation and
   endpoint access coverage. No schema migration is expected; verify the
   existing source uniqueness and suppression constraints cover subtree rules.
3. **Deletion:** extend `RemoveCostItem` for atomic subtree removal and source
   suppression; expose it through `web/src/costApi.ts`.
4. **Grid:** update `Costs.tsx`, `CostGrid.tsx`, API types and `costs.css` for
   preparation, four sections, section-aware add rows and the final bin column.
   Remove acceptance UI. Existing acceptance endpoints can remain compatible
   wrappers during this change; do not couple this feature to API cleanup.
5. **Verification and documentation:** update cost browser tests and speed
   scenarios; record actual results and revise the original plan's superseded
   owner decisions. Deploy/live-project verification is a separate execution
   step, not part of this planning-only request.

## 5. Budgets and acceptance checks

| Path | p50 / p90 ceiling |
|---|---|
| Read-only plan/totals and no-op prepare | 100 / 250 ms |
| Prepare inserting default lines, including returned totals | 250 / 500 ms |
| Edit or delete, including representative subtree deletion | 100 / 250 ms |
| Freeze including reconciliation | 250 / 500 ms |
| Filing regression | 1,000 / 2,000 ms |

Measure at the API boundary, including transaction work and response totals;
add the new paths to the committed latency benchmark/gate using repository
release-versus-local gate conventions. Reuse the representative 13-system,
eight-consultant fixture and include a repeat/no-op case and a multi-child
deletion. Report timings; previous autofill measurements do not prove these
new paths meet their budgets.

Required evidence before declaring implementation complete:

- Fresh and existing projects open with saved default rows, four headings in
  order, no acceptance controls, and blank unknown prices.
- Authority fees and consultants land separately; contingency and escalation
  remain together. Legacy other/excluded rows stay visible. Section subtotals
  reconcile with overall totals using known, unknown, ranged and excluded data.
- Repeated opens, concurrent tabs and a retried prepare never duplicate lines
  or overwrite manual edits. Preparation rollback leaves no partial tree.
- Leaf and group deletion survive reload, source refresh and a new draft after
  baseline; deleting one child leaves sibling values unchanged. Existing
  dismissals remain suppressed. New children cannot resurrect deleted groups.
- Source retirement, missing profile data, manual overlapping lines and later
  trade packages preserve the existing safeguards against double counting.
- Wrong-org/missing-project requests fail safely; stale edit/delete and frozen
  writes are rejected. A no-op prepare emits no update event.
- Direct baseline creation includes current applicable defaults, while old
  baseline contents and issued documents remain unchanged.
- Browser coverage exercises actual preparation, edits, deletes, errors and
  refreshes; inspect desktop and 390 px layouts, keyboard access, quantity mode
  and totals. Existing cost lifecycle and report tests still pass.
- Run the Go cost/store/API tests, access sweep, relevant full-suite checks,
  frontend build and cost Playwright tests. Run the knowledge checker if any
  knowledge file changes; report unrelated failures separately.

## Implementation record

Implemented locally after owner approval:

- Four display sections and exact server-side section totals, including
  parent/child membership and legacy other/exclusion rows.
- Automatic POST preparation with project locking, batched insertion, stable
  source identity, no-op retries and plan/totals from the same saved snapshot.
  GET routes remain read-only. Baseline creation reconciles missing defaults.
- All write responses include totals calculated from their returned plan so
  edits and deletion cannot briefly show a new ledger with old totals.
- No acceptance/dismissal controls in the grid. Far-right bin buttons delete
  leaves directly; group confirmation names the group and child count.
- Atomic subtree deletion preserves baselines and source objects. Previously
  dismissed keys remain suppressed. Removing a manual fee/works line also
  suppresses any equivalent default uncovered by that removal.
- Refresh requests are serialised/coalesced; no-op preparation emits no event.
  Unsaved inputs survive failed writes. Section headings remain visible while
  the wide ledger scrolls horizontally.
- No schema migration, new dependency, service, AI call or automatic price.

Blank automatic rows also leave initial currency/tax settings editable. Manual
lines, rates and monetary values retain the existing protection against
reinterpreting prices; this has a dedicated regression test.

Verification completed:

- Frontend build and both real-server cost Playwright tests passed. Desktop,
  390 px and horizontally scrolled action-column screenshots were inspected.
  Four headings remain visible; no page-wide overflow or baseline delete controls.
- Concurrent preparation, no-op event suppression, subtree and manual-line
  deletion, direct baseline preparation, settings protection, exact section
  totals, API validation and the complete route-access sweep passed.
- Endpoint benchmark passed, including the filing regression gate. The
  13-system/eight-consultant endpoint fixture measured the following local
  timings (three prepare/freeze samples, 84 deletion samples):

| Endpoint path | p50 / p90 | Budget p50 / p90 |
|---|---|---|
| Prepare defaults | 18.7 / 18.7 ms | 250 / 500 ms |
| No-op prepare | 3.4 / 3.4 ms | 100 / 250 ms |
| Delete | 5.5 / 6.7 ms | 100 / 250 ms |
| Freeze | 24.8 / 25.2 ms | 250 / 500 ms |

The separate 20-cycle store benchmark also passed: prepare 17.88 / 21.58 ms,
no-op 1.89 / 3.13 ms, delete 4.00 / 4.94 ms, freeze 17.52 / 19.04 ms, and read
1.06 / 1.22 ms. These are local measurements, not VPS results.

`go test -p 1 ./...` passed all packages except `internal/httpapi`, where the
pre-existing `TestProfilePlacesInputsInBuildOrder` failure still reports
profile flags without a layout home. Cost API tests and the access sweep pass;
the final targeted run also passes the settings adjustment and no-op event
assertion. The design detector reports only the existing 8 px sort-arrow type
ramp finding. `git diff --check` passes.

Live deployment and live-project verification have not been performed by this
task. No new dependency was added.
