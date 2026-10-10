# RFP page one, slice 1 (lane A): what the RFP prints

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** The consultant RFP's section 1 prints one shared project grid and a
site-conditions grid instead of the Project/Detail table; discipline-specific
project rows move to section 2; commercial terms and issue details become
separate field groups; the editor's control groups follow.

**Design:** `docs/plans/2026-10-10-rfp-page-one-design.md` (owner-agreed
10 October 2026).

**Architecture:** The grid's rows are knowledge (keys, labels, columns) in
`internal/knowledge/rfp.go`; the resolver fills them from facts it already
holds (`DriverSet`), so no query or Jev call is added. `sections.go` renders
them as two document tables, which the PDF renderer already lays out
(`internal/reports/render/pdf.go`, up to 8 columns, widths by content).
Each cell carries its control key in the block basis for slice 2.

**Tech stack:** Go, YAML knowledge, the existing Markdown snapshots.

**Lane A rules.** Answer keys (`TestSelectionAnswerKeys`) and the page-fit
test must still pass; snapshots are regenerated with `-update` and every diff
is read. No new dependency, no Jev call, nothing reads the database.

**Run tests with** (bash, worktree `D:\AI Projects\sitewise-pageone`):

```bash
source <scratchpad>/goenv.sh   # PATH, GOCACHE, GOPATH, GOTOOLCHAIN=local, TEMP
go test ./internal/rfp/ ./internal/knowledge/
```

## Decisions made while planning (within the agreed design)

1. **Discipline project rows.** Every discipline row whose key is a grid key
   and `replace: true` (the procurement wording of 13 disciplines) replaces
   that grid cell's text. All other discipline project rows (design vehicle,
   assessment basis, controls and so on) print in section 2 under the role,
   as labelled lines. Town planning's `site_controls` (lot, LGA, zone) and
   architecture's `sustainability` row repeat grid cells, so they are deleted.
2. **Discipline-only fields** `design_vehicle` and `design_loads` move to a
   new field group `appointment` and are offered only to a discipline whose
   rows use them. They were shown on every RFP's project panel.
3. **Site conditions** = every non-scope-decision flag the discipline uses,
   plus any other flag known to be yes. Each prints with Yes, No, Not known,
   or To be confirmed (sources disagree). Scope-decision flags ("Investigation
   selected") are appointment choices: their controls move to section 2 and
   they do not print on page one.
4. **The opening sentence names the whole project** (all parts). When the
   appointment covers only some parts, section 2 prints "This appointment
   covers: …".
5. **Grid rows print when they have a value**; the essentials (work type,
   building, site address, planning pathway, client, procurement route)
   always print, with ___ or "Not yet decided" as today.
6. **Budget** stays its own block (`BudgetBlockID`) after the grid, so issue
   still strips it unless disclosed.
7. **Checks.** Missing required fields raise one check per group: issue
   details (`fields`), terms (`terms`), project (`details`).

## Task 1: Field groups and grid keys in knowledge

**Files:** `internal/knowledge/rfp.go`, `internal/knowledge/rfp_test.go` (or
the existing knowledge test file that loads the skeleton), `knowledge/rfp/skeleton.yaml`.

1. Failing test: a skeleton field in group `terms` or `appointment` loads; a
   discipline `replace: true` row keyed `procurement` loads when the skeleton
   has no `procurement` project row; `knowledge.ProjectGrid` lists the three
   columns in order with the keys below.
2. Implement:
   ```go
   // GridRow is one labelled value on RFP page one. Column is the Profile
   // card's column it mirrors (web/src/Profile.tsx COLUMNS).
   type GridRow struct{ Key, Label, Column string; Essential bool }

   var ProjectGridColumns = []string{"The building", "Site and planning", "Delivery"}
   var ProjectGrid = []GridRow{
       {"work_type", "Work type", "The building", true},
       {"parts", "Building", "The building", true},
       {"storeys", "Storeys", "The building", false},
       {"gfa", "Gross floor area", "The building", false},
       {"parking", "Car parks", "The building", false},
       {"address", "Site address", "Site and planning", true},
       {"lot_dp", "Lot and DP", "Site and planning", false},
       {"lga", "Local government area", "Site and planning", false},
       {"zone", "Zone", "Site and planning", false},
       {"site_area", "Site area", "Site and planning", false},
       {"pathway", "Planning pathway", "Site and planning", true},
       {"client", "Client", "Delivery", true},
       {"procurement", "Procurement route", "Delivery", true},
       {"program", "Program milestones", "Delivery", false},
       {"sustainability", "Sustainability target", "Delivery", false},
       {"regulated_design", "Regulated design applies", "Delivery", false},
   }
   ```
   Field group validation accepts `issue`, `terms`, `project`, `appointment`;
   a profile home stays project-only. The project replacement check accepts a
   key in the skeleton's `project_rows` or in `ProjectGrid`.
3. Skeleton: `agreement`, `payment_days`, `validity_days`, `rates_hold_until`,
   `pi`, `pl` move to `group: terms`; `design_vehicle`, `design_loads` to
   `group: appointment`; `project_rows` keeps only `intro` and `information`.
4. Run knowledge tests and `python tools/check_knowledge.py --strict`: pass.
5. Commit.

## Task 2: Discipline rows

**Files:** `knowledge/rfp/town-planning.yaml`, `knowledge/rfp/architecture.yaml`.

Delete `site_controls` (town planning) and `sustainability` (architecture)
project rows; both repeat grid cells. Check knowledge passes. Commit together
with Task 3 if the resolver tests need it.

## Task 3: Resolver fills the grid, conditions and appointment rows

**Files:** `internal/rfp/resolve.go`, `internal/rfp/page_one_test.go` (new).

1. Failing tests (`page_one_test.go`), using `facts()` and `Resolve`:
   - Architecture with `work_type=new`, `pathway=da`, `procurement=traditional`,
     `site_area=31720`: `r.Grid` has three columns in order; "Site area" is in
     Site and planning as "31,720 m²"; "Work type" reads "New build"; an
     unknown storeys row is absent; the client row prints ___ (essential).
   - Access: the Procurement route cell reads the discipline's replacement
     ("Traditional lump sum; no automatic later appointment or novation.").
   - Architecture `r.Conditions` includes `country` as "Not known" and lists
     a known-yes flag the discipline does not use; no scope-decision flag.
   - Civil: `r.Appointment` holds the Design vehicle row; `r.Project` holds
     only intro and information.
   - A `parts:affected` subset keeps the intro naming every part, and
     `r.Appointment` starts with "This appointment covers: Part 7b."
2. Implement in `resolve.go`:
   - `type GridCell struct{ Key, Label, Text, Control string }`,
     `type GridColumn struct{ Title string; Cells []GridCell }`,
     `type Condition struct{ Key, Label, Answer, Control string }`; new
     `Resolved` fields `Grid`, `Conditions`, `Appointment`.
   - `resolveRows`: shared rows from the skeleton only; discipline rows that
     replace a grid key become grid overrides, the rest go to `Appointment`
     (filled like role rows, kept when complete or required).
   - `resolveGrid`: one cell per `ProjectGrid` row; text from a `gridText`
     switch over existing facts and fields (formatNumber, valueLabel, the
     `pathway`/`procurement` slots); conflict prints "To be confirmed";
     non-essential empty rows are dropped; `regulated_design` only when the
     discipline uses it. Control keys: `fact:<key>` where `FactHome` is
     writable or locked-with-link, else "".
   - `resolveConditions` per decision 3.
   - `partsPhrase` reads `AllParts`; the covers line is prepended to
     `Appointment` when `Parts` is a strict subset.
3. Run: pass. Commit.

## Task 4: Sections print page one and section 2

**Files:** `internal/rfp/sections.go`, `internal/rfp/page_one_test.go`.

1. Failing test: architecture `Sections` → section `project` blocks are, in
   order, `rfp:header`, `rfp:project:intro`, `rfp:project:grid` (table
   columns `The building, "", Site and planning, "", Delivery, ""`),
   `rfp:project:conditions` (columns `Site conditions, "", ""`, cells
   "Label: Answer"), `rfp:project:information`; no table has a "Detail"
   column. Section `role` ends with `rfp:appointment` when there are
   appointment rows.
2. Implement: grid rows zip the three columns (blank pairs pad short
   columns); conditions fill three per row; basis carries `grid` and
   `conditions` (with control keys). Budget block unchanged.
3. Run: pass. Commit.

## Task 5: Editor control groups

**Files:** `internal/rfp/controls.go`, `internal/rfp/controls_test.go` (new,
or in `page_one_test.go`).

1. Failing test: `BuildControls` for architecture has groups `project`
   (section project; no `parts:affected`, no `field:design_vehicle`, no
   scope-decision flag), `appointment` (section role; `parts:affected`),
   `terms` (section terms; `field:agreement`), `issue` (section `issue`;
   `field:ref`); civil's `appointment` group holds `field:design_vehicle`;
   project flag controls are exactly `r.Conditions`.
2. Implement. A discipline uses an appointment field when any of its rows'
   text holds `{field:<key>}`.
3. In `resolveRows`, split the missing-field check: `fields` (issue group),
   `terms` (terms group), `details` (profile-homed). Test the agreement gap
   raises `terms`.
4. Run: pass. Commit.

## Task 6: Snapshots, page fit, answer keys, whole suite

1. `go test ./internal/rfp/ -run TestGoldenRFPs -update`, then read
   `git diff internal/rfp/testdata`: page one shows the grid and conditions
   for every scenario; nothing outside sections 1 and 2 changed except
   terms wording that reads moved fields (it must not change).
2. `go test ./internal/rfp/ ./internal/knowledge/ ./internal/reports/...
   ./internal/store/ ./internal/httpapi/` with the test database; page fit
   and answer keys pass. Record the page counts.
3. `go run ./cmd/intake-bench ... -budgets bench/budgets.json` for
   `report_setting` and `report_fact` (or the full bench if it is quick);
   record timings.
4. Commit snapshots.

## Not in this slice

The screen (slice 2): in-place editing, markers, the Issue section, the
stale label. Between the slices the editor shows terms fields under Terms and
issue fields in a group whose section the report does not have yet; slice 2
renders it.
