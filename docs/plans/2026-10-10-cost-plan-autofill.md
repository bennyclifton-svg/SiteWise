# Cost plan autofill: lines from the profile and procurement

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A user opens Costs on a profiled project and the plan already has
its shape: every consultant fee, every building system and the usual
project-wide allowances appear as suggested lines. Typing a budget into a
suggested line keeps it. **Add all suggested** keeps the lot. A line the
user wants that nobody suggested is typed straight into the grid. Nobody
adds lines one at a time through a form.

**Founding objective:** as automated as possible. Everything the system
already knows (the profile, the work items, the packages) turns into cost
lines without the user asking. The user only types money and the odd
missing line.

**Architecture:** Suggestions are worked out when the plan is read, never
stored, in the same pattern as Implications and procurement suggestions.
Each suggestion has a stable `source_key`. Keeping one writes an ordinary
cost line carrying that key, so it is never suggested twice. Dismissing one
records the key. The work is pure lookup: knowledge YAML plus rows already
in the database. **No Jev** (see §Jev).

**Tech stack:**
- Go: `internal/costs`, `internal/knowledge`, `internal/store`,
  `internal/httpapi`.
- Migrations: `021p`, `021q`.
- Knowledge: `knowledge/costs/elements.yaml` and
  `knowledge/costs/starter_lines.yaml`, both `status: draft`.
- React: `web/src/CostGrid.tsx`, `web/src/Costs.tsx`, `web/src/costApi.ts`.
- Playwright.

---

## 0. Task statement (AGENTS.md)

- **User outcome:** Petersham's Costs grid shows about 20 suggested lines
  on first open, with no typing. Turning them into the working plan takes
  one click, or happens line by line as budgets are typed.
- **Why:** time-to-decision. A cost plan that starts from a blank grid
  takes an hour of form-filling before it says anything.
- **Lanes:**
  - Parts 1–3 are **lane A**. They create lines a user trusts, and change
    a schema constraint and a read path.
  - Part 4 (inline add) is **lane B**.
  - Part 5 (knowledge files) is knowledge work. `check_knowledge.py` must
    pass.
- **Budgets:**

| Path | Route | p50 | p90 |
|---|---|---|---|
| `costs_read` | `GET /projects/{id}/cost-plan` (now includes suggestions) | 100 ms | 250 ms |
| `costs_write` | keep one suggestion; inline add; inline cell edit | 100 ms | 250 ms |
| `costs_accept_all` (new) | `POST /projects/{id}/cost-plan/suggestions/accept` | 250 ms | 500 ms |

  - No Jev on any path. Tests assert `jevHits == 0`.
  - Opening Costs makes no more requests than today. Suggestions ride on
    the plan read.
- **Out of scope:**
  - Rates and amounts. Every suggested line's budget is "—".
  - Contingency as a percentage.
  - Reading cost documents.
  - Tender review and comparison (see §Later).
  - Changing how packages are created (that's the Procurement table's
    job).
- **Done:**
  - Go, store and migration tests pass.
  - `check_knowledge.py` passes.
  - `npm run build` passes, and `costs.spec.ts` (extended) passes.
  - The bench gate passes for the three paths above.
  - Petersham shows the suggested lines in the browser, and Add all
    suggested turns them into lines.

## Precondition: merge the Procurement table first

Owner decision C5. `procurement/table` is mid-build in
`../sitewise-procurement`:
- Tasks 1–6 of `2026-10-10-procurement-table.md` are committed.
- Edits to `Reports.tsx` and `reportApi.ts` are uncommitted.
- `main` has moved past its base.

That session finishes Tasks 7–11, rebases and fast-forwards into local
main. This plan starts from that main.

This plan doesn't edit the Procurement files. It only **reads**
`packages.discipline_id`, `kind`, `works_scope` and `retired_at`, which
exist on main today. If the merge slips, Parts 1–3 can still be built and
tested against Petersham's existing consultant packages.

---

## Owner decisions (2026-10-10)

- **C1. Suggest and accept, never write behind the user's back.**
  Suggestions are faint rows.
  - Typing a budget into one keeps the line and the amount in one request.
  - **Add all suggested** keeps every suggestion in one request.
  - **×** dismisses a suggestion. A dismissed suggestion never comes back
    unless the user restores it from "Show dismissed".
- **C2. One construction line per building system** (the 13 top-level
  systems in `systems.yaml`, in building order), not one per work item.
  The line's work items are its scope. They're listed in the drawer.
- **C3. Access and egress is costed item by item.** NCC compliance there
  is real money:
  - stair nosings and handrails
  - compliant balustrades
  - tactile ground surface indicators (TGSIs)
  - braille and tactile signage
  - accessible sanitary facilities and hearing augmentation
  - egress hardware
  
  So it's a group line whose children are its included sub-items (see
  Part 5). Every other system stays one line.
- **C4. Trades map to systems the way a user would guess.**
  Electrical → Electrical Services, Mechanical → Mechanical Services, and
  so on. Anything with no trade goes to Main Works. A few sub-items are
  overridden (joinery to Joinery and Kitchens, structural steel to
  Structural Steel). The map is data (`elements.yaml`), so the tail can be
  corrected without code.
- **C5. The Procurement table merges first.**
- **C6. Tender breakdowns come later and land as children** of the
  package's line (§Later). This plan leaves the seam: `source_key` and the
  existing Subdivide.

---

## What a suggestion is

| Group | Source | One suggestion per | `source_key` | Shape |
|---|---|---|---|---|
| Fees | `packages` where `kind='services'`, not retired | consultant package | `fee:<package_id>` | `fee`, package set, stage empty (Part 1) |
| Construction | top-level systems with at least one **included** work item, in building order | building system | `works:<system_id>` | `works`, `system_id` set, no work item (Part 1) |
| Construction, access and egress | `access-egress.*` sub-items that are included and `costed: true` in `elements.yaml` | sub-item | `works:<system_id>` on the group, `works:<sub_id>` on each child | group: non-posting; child: posting `works` line with its `work_item_id` (today's shape) |
| Project-wide | `starter_lines.yaml`, filtered by `applies_when` against profile values | starter line | `pw:<starter_id>` | `project_wide`, category from the file |

**Not suggested:**
- Supply packages.
- Works packages: the head contract and trades are shown as each
  construction line's **Category**, not as lines of their own.
- Excluded or retired work items, and systems with nothing included.

**Category for a construction line** is worked out when the plan is read,
never stored. A line has `package_id` only when the user sets it. Otherwise
the grid shows the package for its system, resolved in this order:

1. A live package for the sub-item's override discipline, if a
   sub-item override applies (e.g. `interiors.joinery` →
   `trade.joinery_kitchens`).
2. A live package for the system's trade discipline (e.g. `electrical` →
   `trade.electrical`).
3. The live Main Works package (`trade.main_works`, or a `works_scope =
   'head_contract'` package).
4. "Unassigned".

This never writes. It means creating an Electrical Services row in
Procurement moves the Electrical line's category on the next read, with no
migration or sync job. It also avoids today's rule that a works line with
`package_id` needs a scope responsibility row (`costTargets`).

A sub-item override becomes its own suggested line under the same system
only when its package exists. Example: with a Joinery and Kitchens package,
"Joinery" becomes its own construction line under Interiors, so Interiors
doesn't hide the joinery price.

**Suggested label**:
- Fee line: the package title.
- Construction line: `elements.yaml` `cost_label`, or else the system label.
- Project-wide line: the starter label.

**Code**:
- Construction lines take the element code from `elements.yaml` (e.g.
  `02`).
- Children take `02.1`, `02.2` and so on.
- Fee lines take `F01`, `F02` and so on in title order.
- Project-wide lines take `P01`, `P02` and so on.

Codes are suggestions only and are editable once kept.

---

## Part 1 — Line shapes (lane A)

### Task 1: A construction line can stand for a building system

**Files:**
- Create `internal/db/migrations/021p_cost_line_sources.sql`.
- Modify `internal/costs/plan.go`: add `SystemID string json:"system_id"`
  to `Content`. `Item.SystemID` stays the read value.
- Modify `internal/store/costs.go`: `writeCostContent` writes `system_id`.
  The read `CASE` becomes `COALESCE(w.system_id, r.system_id)` on drafts.
- Modify `internal/costs` validation: a works line has exactly one of
  `work_item_id` or `system_id`.
- Test in `internal/db/migrate_cost_sources_test.go` and
  `internal/store/costs_test.go`.

The migration:
- Replaces the line-kind `CHECK` on `cost_item_revisions`:
  - **works**: `(work_item_id IS NOT NULL AND system_id IS NULL) OR
    (work_item_id IS NULL AND system_id IS NOT NULL)`, with no stage and no
    category.
  - **fee**: `package_id IS NOT NULL`, with `package_stage_id` now
    optional (Task 2).
  - **project_wide**: unchanged.
- Adds `source_key text` with
  `CHECK (source_key IS NULL OR length(source_key) BETWEEN 1 AND 200)` and a
  unique index on `(org_id, plan_version_id, source_key) WHERE source_key
  IS NOT NULL`.
- Adds `cost_suggestion_dismissals(org_id, project_id, source_key, actor,
  dismissed_at)` with PK `(org_id, project_id, source_key)` and the FK to
  projects. Add it to the tenant cascade and the FK sweep fixtures (pattern:
  `021b_cost_tenant_cascade.sql`, `next_wave_foreign_keys.json`).

Existing rows already satisfy the new checks. Frozen versions aren't
rewritten; this only relaxes the constraint.

`system_id` on a works line is checked against the catalogue in
`costTargets`. It must be a top-level system or a sub-item of one. If it
isn't, return 422.

**Tests:**
- Before the migration:
  - A works line with `system_id='electrical'` and no work item is
    rejected.
- After the migration:
  - That line is accepted.
  - A line with both `system_id` and `work_item_id` is rejected.
  - An unknown system returns 422.
  - Org B can't see or write org A's dismissals.

`cloneCostRows` (baseline freeze) copies `system_id` and `source_key`, so
the new draft keeps its keys. Test: freeze, then no suggestion reappears.

Commit: `A cost line can stand for a building system; lines carry their source (lane A)`

### Task 2: A fee line without a stage

**Files:**
- `internal/costs` validation.
- `costTargets`: check the stage only when one is given.
- `internal/store/costs_test.go`.

A fee line needs a package. A stage is optional, and Subdivide splits a
fee line into stages later. The existing "stage must belong to the
package" check is unchanged.

Commit: `Fee lines need a package, not a stage (lane A)`

---

## Part 2 — Suggestions on the plan read (lane A)

### Task 3: Work out suggestions in Go

**Files:**
- Create `internal/costs/suggest.go` and `internal/costs/suggest_test.go`.

Write a pure function with no database or catalogue loading inside:

```go
// Suggest lists the lines a plan is missing. Inputs are what the store has
// already read; nothing here is judged, only looked up.
func Suggest(in SuggestInput) []Suggestion
```

`SuggestInput` holds:
- `Elements` (from `elements.yaml`) and `Starters` (from
  `starter_lines.yaml`).
- Systems in building order.
- Included, live work items: `system_id`, `id`, `title`.
- Live packages: `id`, `kind`, `discipline_id`, `works_scope`, `title`.
- Profile values, for `applies_when`.
- Existing `source_key`s in the draft.
- Dismissed keys.

`Suggestion` holds:
- `Key`, `Group` (fee, works or project_wide) and `Label`, `Code`.
- `Content`: the `costs.Content` that keeping it would write, with
  `origin: calculation`, `meaning: allowance` and a rationale naming the
  source, e.g. "Suggested from the Architect package" or "Suggested from 6
  included work items in Electrical".
- `ParentKey`, for access and egress children.
- `ScopeWorkItems` (ids), shown in the drawer.
- `Category`: the resolved package title (see the resolution order above).

The same resolver also gives `Category` for kept works lines, so the grid
reads one rule.

**Failing tests** (table-driven, Petersham-shaped fixture):
- Three consultant packages give three fee suggestions, sorted by title,
  coded F01–F03.
- 89 included items across 13 systems give 12 construction suggestions plus
  access and egress as one group with 5 children: stairs and ramps,
  barriers, accessible access, accessible facilities, egress doors. Exits
  and livable housing are `costed: false`.
- A system whose items are all excluded gives no suggestion.
- A key already in the draft, or dismissed, isn't suggested. When a group
  is kept, its children are still suggested, with `ParentKey` resolving to
  the kept group's id.
- Category resolution:
  - No works packages: "Unassigned".
  - Main Works only: every line shows Main Works.
  - Main Works plus Electrical Services: `electrical` →
    Electrical Services, `comms-security` → Electrical Services, the rest →
    Main Works.
  - Joinery and Kitchens package: a separate "Joinery" suggestion under
    Interiors.
  - A retired package is ignored.
- Starter lines filter on `applies_when`. With `hdr.cond.planning = da`,
  DA fees and contributions are suggested. With no value, they're
  suggested with a rationale saying "planning pathway unknown". An unknown
  value never removes a line.
- A supply package gives no suggestion.

Commit: `Suggest cost lines from packages, building systems and starter lines (lane A)`

### Task 4: Store reads and returns suggestions

**Files:**
- `internal/store/costs.go`: add to `ReadCostPlan` and `readCostPlan`.
- `internal/costs/plan.go`: add `Suggestions []Suggestion json:"suggestions"` and
  `Dismissed []Suggestion json:"dismissed"` on `Plan`.
- `internal/store/cost_suggest_test.go`.

Inside the existing repeatable-read transaction, read:
- Packages: one query.
- Included, live work items: one query, columns `system_id, id, title,
  part_id`.
- Dismissals: one query.
- The profile values `applies_when` needs (`hdr.*`, `proj.*`) from
  `profile_rows`: one query, keys filtered to those the starter file names.

Run these with `pgx.Batch` so they make one round trip, not four.

Rules:
- Suggestions are computed for the draft only. A baseline view returns
  `[]`.
- If the profile is stale or has pending documents, suggestions still show.
  They're structure, not money, and the next read updates them.

**Tests:**
- Petersham-shaped seed: the read returns suggestions as in Task 3.
- Baseline read: no suggestions.
- Org B: 404.
- Read-only transaction: no rows written (count `cost_item_revisions`
  before and after).

**Speed:** run the `costs_read` bench and record p50/p90 in the commit. If
p90 goes over 250 ms, cache the catalogue-derived part (elements, starters,
building order) on `Store`. It changes only with the knowledge version.

Commit: `Cost plan read carries its suggestions; costs_read within budget (lane A)`

---

## Part 3 — Keep, keep all, dismiss (lane A)

### Task 5: Keep one suggestion, with an optional budget

**Route:** `POST /projects/{id}/cost-plan/suggestions/{key}/accept` with
`{plan_version_id, version, budget?: string|null}`. Path: `costs_write`.

**Files:**
- `internal/store/cost_suggest.go`.
- `internal/httpapi/costs.go`.
- `internal/httpapi/projects.go`: route plus speed path.
- Tests in `internal/store` and `internal/httpapi`.

**Behaviour:**
- Recompute suggestions inside the write transaction. If `key` isn't
  currently suggested, return 404 and the client reloads.
- Insert the line with `source_key = key`, then the budget value if one
  was given (`origin: user`, `meaning: allowance`), in one transaction with
  one version bump.
- Keeping a child whose group isn't kept yet keeps the group first, in the
  same transaction.
- Keeping the same key twice (double click, retry) returns the existing
  line with 200. That's guaranteed by the unique index and `ON CONFLICT DO
  NOTHING` followed by a read.

**Tests:**
- Keep with and without a budget.
- Double keep: one line.
- Stale `version`: 409.
- Frozen plan: 409.
- Child before group: both written, child under group.
- Wrong org: 404.
- Unknown key: 404.
- `jevHits == 0`.

Commit: `Keep a suggested cost line, with its budget, in one request (lane A)`

### Task 6: Keep all

**Route:** `POST /projects/{id}/cost-plan/suggestions/accept` with
`{plan_version_id, version}`. New speed path: `costs_accept_all`, added to
`bench/budgets.json` at p50 **250 ms**, p90 **500 ms**. The reason
for a separate path: about 25 inserts in one transaction.

- One transaction. Inserts every current suggestion (groups before
  children) with `pgx.Batch`. No budgets are written, so they stay "—".
- Returns the plan.

**Tests:**
- Petersham-shaped seed gives the expected line count.
- A second call adds nothing and returns 200.
- Dismissed keys aren't written.
- Frozen: 409.
- Wrong org: 404.
- Bench seeds a project with 13 systems and 8 packages and records
  p50/p90.

Commit: `Keep all suggested cost lines in one request (lane A)`

### Task 7: Dismiss and restore

**Routes:**
- `POST /projects/{id}/cost-plan/suggestions/{key}/dismiss`
- `DELETE` on the same path to restore.

Both use `costs_write`.

- Dismissing a key that is no longer suggested still records the
  dismissal. It's harmless.
- Removing a kept line that has a `source_key` (existing
  `RemoveCostItem`) also records a dismissal in the same transaction.
  Otherwise the line would immediately come back as a suggestion.

**Tests:**
- Dismiss, then the suggestion is gone.
- Restore, then it's back.
- Remove a kept line, then it isn't re-suggested.
- Wrong org: 404.

Commit: `Dismiss or restore a suggested cost line (lane A)`

---

## Part 4 — The grid (lane B)

### Task 8: Suggested rows

**Files:** `web/src/CostGrid.tsx`, `web/src/costApi.ts`, `web/src/costs.css`.

- Suggested rows sit in their group, in code order, at reduced opacity, with
  a small "suggested" tag.
- Columns:
  - Code, Category and Item show as described above.
  - Budget is an input cell exactly like today's `BudgetCell`. Saving calls
    the keep route with `budget`, so one request keeps and prices the line.
  - Every other figure column shows "—".
- Row actions: **Keep** (keeps with no budget) and **×** (dismiss).
- Clicking the row opens the drawer read-only: the rationale and, for
  construction lines, the scope work items.
- Toolbar:
  - **Add all suggested (n)** shows only when n > 0.
  - "Show dismissed (m)" lists dismissed suggestions with Restore.
- Group headings show even when a group has only suggestions. An empty plan
  shows suggestions straight away, not an empty state.
- Kept construction lines show the resolved Category from the plan read.

### Task 9: Inline add and edit

**Files:** `web/src/CostGrid.tsx`, `web/src/Costs.tsx`.

The "Add cost line" drawer form is removed.

- Each group ends with a **+ Add line** row. Clicking it inserts a blank
  row in that group with focus in Item.
- **Item**: type the label. Enter or blur saves through `costApi.item`
  (POST) with the group's line kind. Esc removes the blank row.
  Project-wide defaults to `category: other`. A fee line requires a
  package, so its Category cell opens first.
- **Category** is a select in the row:
  - Fees: consultant packages.
  - Construction: building systems in building order. This sets
    `system_id`, and the package resolves itself.
  - Project-wide: the five categories.
- **Code** is editable in the cell. It saves on Enter or blur through
  PATCH.
- **Budget** works as today.
- A **Quantities** toggle in the toolbar shows Qty, Unit and Rate columns
  after Item. Editing Qty or Rate PATCHes the line. When both are set the
  server's existing quantity × rate writes the budget. The toggle is
  remembered per viewer in `localStorage`, wrapped in try/catch.
- Rationale, ranges, as-of date, exclusion, approved contract and claimed
  to date stay in the drawer. Clicking the row opens it as today.
- Each cell commit is one request, using today's 409/422 handling and kept
  input.

### Task 10: Playwright

**File:** `web/tests/costs.spec.ts`. Use a temporary config on a free port,
per the port-4173 gotcha.

1. Seed a project with three consultant packages and work items across
   three systems, including two access and egress items. Open Costs. The
   suggested rows are visible: three fees, three construction lines, the
   access and egress group with two children, and the starter lines.
2. Type 50000 into a suggested Architect budget. The row is no longer
   suggested, Budget shows 50,000.00, and Forecast shows 50,000.00.
3. Dismiss Escalation. It's gone. Show dismissed, then Restore.
4. Add all suggested. No suggested rows remain, and the line count matches
   the API.
5. **+ Add line** under Construction: type "Façade access allowance",
   choose Envelope, Enter. The row is saved and Category shows Main Works
   when that package exists.
6. Turn on Quantities: Qty 10, Unit m2, Rate 120. Budget shows 1,200.00.
7. Check 390 px: no horizontal page scroll. Freeze the baseline: no
   suggestions and no add rows.

---

## Part 5 — Knowledge (draft, owner review)

### Task 11: `knowledge/costs/elements.yaml`

One entry per top-level system, in `systems.yaml` order:

```yaml
# Cost elements: how building systems become cost-plan lines and which
# trade package each one falls under. Draft for owner review (C2–C4).
version: 1
status: draft
sources:
  - {design: docs/plans/2026-10-10-cost-plan-autofill.md, anchor: "## Owner decisions (2026-10-10)"}
elements:
  - {system: site, code: "01", cost_label: Site works and external works, trade: trade.civil_works}
  - {system: substructure, code: "02", trade: trade.main_works}
  - system: structure
    code: "03"
    trade: trade.main_works
    overrides: [{sub: structure.steel, trade: trade.structural_steel}]
  - {system: envelope, code: "04", trade: trade.main_works}
  - system: interiors
    code: "05"
    trade: trade.main_works
    overrides: [{sub: interiors.joinery, trade: trade.joinery_kitchens}]
  - {system: fire-passive, code: "06", trade: trade.main_works}
  - {system: fire-active, code: "07", trade: trade.fire_services}
  - {system: mechanical, code: "08", trade: trade.mechanical}
  - {system: hydraulic, code: "09", trade: trade.hydraulic_plumbing}
  - {system: electrical, code: "10", trade: trade.electrical}
  - {system: comms-security, code: "11", trade: trade.electrical}
  - {system: vertical-transport, code: "12", trade: trade.main_works}
  - system: access-egress
    code: "13"
    trade: trade.main_works
    itemised: true
    items:
      - {sub: access-egress.stairs-and-ramps, cost_label: "Stairs, ramps, handrails and nosings", costed: true}
      - {sub: access-egress.barriers, cost_label: Compliant balustrades and barriers, costed: true}
      - {sub: access-egress.accessible-access, cost_label: "Accessible paths, TGSIs and braille/tactile signage", costed: true}
      - {sub: access-egress.accessible-facilities, cost_label: Accessible sanitary facilities and hearing augmentation, costed: true}
      - {sub: access-egress.egress-doors, cost_label: Egress doors and hardware, costed: true}
      - {sub: access-egress.exits, costed: false, why: Travel distances and exit layout are a design outcome priced in other elements.}
      - {sub: access-egress.livable-housing, costed: false, why: Livable-housing features are priced in interiors and hydraulic.}
```

Why the `cost_label`s read the way they do:
- The access and egress labels name the items you raised: nosings, TGSIs,
  braille and tactile signage, compliant balustrades.
- Each sub-item's `describes` in `knowledge/clusters/fire/systems.yaml`
  already covers them, so no new catalogue system is needed.
- Sub-items not listed here (for example pool barriers) are suggested with
  their catalogue label and `costed: true`. A new catalogue sub-item is
  never dropped silently.

**Loader** (`internal/knowledge`). The loader rejects:
- an unknown `system` or `sub`, or a `sub` with the wrong parent;
- a missing top-level system;
- an unknown `trade` discipline id (checked against
  `data/intake/disciplines.json`);
- a duplicate code.

`check_knowledge.py` gets the same checks.

Petersham's `structure.design-basis` and `envelope.thermal-performance` are
scope items of their element line. They're never lines of their own,
because only access and egress is itemised.

### Task 12: `knowledge/costs/starter_lines.yaml`

```yaml
version: 1
status: draft
lines:
  - {id: design-contingency, label: Design contingency, category: contingency, applies_when: true}
  - {id: construction-contingency, label: Construction contingency, category: contingency, applies_when: true}
  - {id: escalation, label: Escalation, category: escalation, applies_when: true}
  - {id: da-fees, label: DA fees and contributions, category: authority_fees, applies_when: {field: hdr.cond.planning, in: [da]}}
  - {id: cc-fees, label: Construction certificate and certifier fees, category: authority_fees, applies_when: true}
  - {id: service-authority, label: Service authority connection fees, category: authority_fees, applies_when: true}
  - {id: exclusions, label: Exclusions, category: exclusions, applies_when: true}
```

- **Labels only. No percentages and no amounts.** Rule numbers (for
  example contingency bands or levy rates) wait for an instrument and stay
  `verified: false` when added later (non-negotiable 5).
- `applies_when` uses the existing works predicate vocabulary and is
  validated by `checkWorksRefs` / `validateCostPredicate`.
- Residential and commercial share this list for now. A class-specific
  list (for example the HIA Schedule of Allowances groupings for houses)
  is an owner call after Petersham is reviewed.

Commit both files with `python tools/check_knowledge.py` passing:
`Draft cost elements and starter lines (knowledge)`

**Order of work:** Task 11 and Task 12 land before Task 3's test fixture
reads them. Task 3's unit test builds its own input, so it doesn't depend
on the files.

---

## Jev

There's no Jev call in this plan. Every step is a lookup:
- package → fee line
- system → element line → trade package
- profile value → starter line

TypeSafe's [patterns](https://docs.typesafe.ai/patterns) page frames the
model's job as answering small classification questions, with routing and
combining left to code. Nothing here needs a judgement that code can't
make exactly. A model call would only add latency and a way to be wrong
(non-negotiables 1 and 3).

## Later (not this plan)

**Tender review and comparison** (C6).

An RFP or RFT names a price breakdown. Tenders come back either in that
breakdown or in their own. A tender-review module will either adopt the
nominated breakdown or settle an adjusted one from the winning tender. It
then writes the result as **children of the package's line**, through the
existing Subdivide:
- Each child carries `source_key = tender:<report_id>:<row>`.
- The awarded total becomes the commitment.

This plan's `source_key` and stage-optional fee lines are the seam. Nothing
else needs to change.

That module will look at reading tender submissions with a model. AGENTS.md
allows Jev only. Whether Jev's pick-from-options (for example "which
nominated breakdown row does this tender line belong to") is enough, or the
rule has to change, gets decided in that module's design. It isn't assumed
here.

**Other later work:**
- **Rates:** owner-supplied, reviewed $/m² per element, multiplied by
  `hdr.scale.gfa_sqm` through the existing benchmark path
  (`knowledge/costs/benchmarks.yaml`). Shown as a one-click suggested
  budget, never written automatically.
- **Contingency as a percentage** of the construction subtotal, once the
  owner confirms the rate.
- **Reading an uploaded QS cost plan** into kept lines. Jev picks the
  element, precomputed at filing.

## Edge cases

- **Wrong org:** 404 on every new route, and covered by the access sweep.
- **Frozen plan, or viewing the baseline:** no suggestions, no add rows,
  and keep, keep-all and dismiss return 409.
- **Double click on Keep or Keep all:** one line per key (unique index).
- **Suggestion vanishes between read and keep** (package retired, work item
  excluded): 404, and the grid reloads.
- **Package retired after its fee line was kept:** the line stays and shows
  its package title with "retired". No new suggestion appears.
- **Work items in a system all excluded after its line was kept:** the line
  stays. The drawer scope shows "no included work items".
- **A trade package created later:** kept construction lines re-categorise
  on the next read, with no write.
- **User sets `package_id` on a construction line:** the explicit package
  wins over the mapped one.
- **Starter line whose `applies_when` field is unknown:** still suggested,
  with the rationale saying so.
- **Profile not yet built:** construction suggestions come from work items.
  With none, fees and starters still show.
- **Removing a kept suggested line:** it's recorded as dismissed, so it
  doesn't come back.
- **Legacy duplicate package** (Petersham's "Structural Design" alongside
  "Structural"): both are suggested. The user dismisses one. This plan
  doesn't de-duplicate packages; Procurement owns that.
