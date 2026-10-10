# RFP single report: project facts in Profile, edits in place

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A consultant RFP reads as one report you edit in place. Project
facts are answered once in Profile, and an edit made in the report saves
back to Profile.

**Architecture:** The RFP stops holding per-RFP overrides of project facts.
Facts are read from Profile into the draft's frozen drivers, as they already
are. A fact edited in the report writes to its Profile home, then patches
that one frozen driver and re-resolves the draft in the same request. The
"Finish this RFP" column goes away: each report section gets its own edit
controls, and the blocking decisions become a banner at the top.

**Tech stack:** Go (`internal/rfp`, `internal/store`, `internal/httpapi`),
YAML knowledge (`knowledge/profile`, `knowledge/rfp`), React
(`web/src/Reports.tsx`, `web/src/RfpEditor.tsx`, `web/src/Profile.tsx`),
Playwright.

---

## Owner decisions (2026-10-10)

- **D1.** An edit to a project fact made in the RFP saves to Profile. Profile is the one source of truth.
- **D2.** Existing per-RFP overrides are deleted, not migrated. All data is test data.
- **D3.** Client agreement, PI cover, PL cover, payment terms and proposal validity stay in the RFP, editable on each RFP. Their defaults belong to the discipline: a small engagement can carry lower cover.
- **D4.** The Reports tab is renamed Procurement: consultants first, then trades and the main works. PMP moves to its own tab later.

## Lane, scope, budget

- **Lane:** A (what the RFP prints, so answer trust). The new endpoint follows lane C rules: a missing record returns 404, a wrong org returns 404, and bad input returns 422.
- **Out of scope:**
  - Editing individual rows inside tables. Editing is per section in this version.
  - RFT and PMP reports.
  - Editing the budget from the report. The budget stays owned by the cost plan; the report links to Costs.
  - Any change to discipline rules or to `verified` flags.
  - Marking anything `reviewed`.
- **Budget:**
  - New `report_fact` bench: p50 ≤ 150 ms, p90 ≤ 400 ms. That is `profile_edit` plus `report_setting`.
  - `report_setting`, `report_read` and `report_assemble` stay inside their current budgets.
  - No Jev call is made on any of these paths.
- **Done:**
  - The Petersham structural RFP shows one full-width report.
  - Every project fact in it shows the source "Profile".
  - Changing the zone in the report changes it in Profile, and the zone shows in Profile after a reload.
  - All Go tests pass, `python tools/check_knowledge.py` passes, the Playwright RFP specs pass, and every bench is within budget.

## Where each control goes

| Control (current panel) | Home after this change |
|---|---|
| Work type, planning pathway, procurement route | Profile `hdr.work_type`, `hdr.cond.planning`, `hdr.cond.procurement_route` (existing) |
| Site area, GFA | Profile `hdr.scale.site_sqm`, `hdr.scale.total_gfa_sqm` / `gfa_sqm` (existing) |
| Construction budget | Cost plan (read-only in the report, with a link) |
| Flags that already have a determinant (bushfire, flood, contamination, acid sulfate, groundwater, existing building, heritage) | Profile `det.*` (existing; heritage writes `det.heritage`) |
| Live operations | Profile `hdr.cond.operational_constraints` (read-only in the report, with a link; it is a choice, not yes/no) |
| Mixed use | Derived from the parts' classes (read-only) |
| Other flags (coastal, Country, staged, multi-building, basement, excavation adjoining, transfer structures, authority asset, rail, OSD existing, OSD required) | **New** Profile keys `proj.flag.*` (Task 1) |
| Regulated design applies | **New** Profile key `proj.regulated_design`, with a note recording the advice relied on |
| Client, LGA, zone, program milestones, sustainability target | **New** Profile keys `proj.client`, `proj.lga`, `proj.zone`, `proj.program`, `proj.sustainability` |
| Site address, lot and DP | Site record `address` and `lot` (existing columns, endpoint `PATCH /sites/{site}`; shown in Profile for the first time) |
| Parts this appointment covers | Stays in the RFP (it belongs to this appointment) |
| Design loads, design vehicle, information provided | Stay in the RFP (specific to the discipline or to this RFP) |
| Appointment, stages, service level, elements, allowances, scope position | Stay in the RFP |
| Issue details and weights | Stay in the RFP; defaults per discipline (D3) |

---

### Task 0: Worktree

Create the worktree `../sitewise-rfp-single` on branch `rfp/single-report`.
Use @superpowers:using-git-worktrees. Copy across any private data the gate
needs (see memory note "next-wave gate running").

### Task 1: Profile keys for project details (knowledge)

**Files:**
- Create: `knowledge/profile/project_details.yaml`
- Modify: `knowledge/SCHEMA.md` (add a short section for the new file), `knowledge/profile/key_scope.yaml`, the Go knowledge loader for profile files (`internal/knowledge/profile*.go`), `tools/check_knowledge.py`

**Steps:**
1. Write a failing loader test: `proj.zone` loads with scope `site`, and `proj.client` loads with scope `project`.
2. Add the file. Each key has `key`, `label`, `value` (text or tristate), `scope`, `status: draft`, and `entered: user`. `entered: user` means no extraction and no Jev question.
   - Keys: client, lga, zone, program, sustainability, regulated_design, and the 11 flags in the table.
   - Use the labels from `knowledge/rfp/skeleton.yaml` so the wording matches.
3. Teach the loader and `check_knowledge.py` to read the file. A `proj.*` key must not also exist as a `det.*` or `hdr.*` key.
4. Run `go test ./internal/knowledge/...` and `python tools/check_knowledge.py`. Both must pass.
5. Commit: "Add user-entered project detail keys to the profile (lane A)".

### Task 2: Profile accepts and shows project details

**Files:**
- Modify: `internal/httpapi/profile.go` (`validProfileValue`), the profile builder that emits `data.facts`, `web/src/Profile.tsx`, `web/src/profileApi.ts`
- Test: `internal/httpapi/profile_test.go`, `web/tests/profile*.spec.ts`

**Steps:**
1. Write failing tests:
   - `PUT /projects/{id}/profile/proj.zone` with `"R2"` returns 200, and the value appears in `GET /profile`.
   - A tristate value of `"maybe"` returns 422.
   - A request from another org returns 404.
2. Implement. Writes go through the existing `SetUserValue` path, which keeps provenance, versioning and Reset.
3. Profile UI: add a "Site and project details" group to the Project section.
   - It shows site address, lot and DP, LGA, zone, client, program and sustainability using the `FieldRow` idiom. Address and lot save through `PATCH /sites/{site}`.
   - The flags and regulated design go under "More detail".
4. Run the Go tests and the Playwright profile specs.
5. Commit.

### Task 3: The RFP reads facts and never overrides them

**Files:**
- Modify: `internal/rfp/drivers.go`, `internal/rfp/settings.go` (`validKey`, `textFacts`), `internal/rfp/resolve.go` (check `Control` targets), `internal/rfp/controls.go`, `knowledge/rfp/skeleton.yaml` (drop the moved `group: project` fields)
- Create: migration `internal/db/migrations/NNN_drop_rfp_fact_settings.sql`
- Test: `internal/rfp/resolve_test.go`, `internal/rfp/drivers_test.go`, `internal/store/report_settings_test.go`, the RFP answer keys and golden tests

**Steps:**
1. Write failing tests:
   - `Drivers` reads `proj.zone`, `proj.client`, `proj.flag.basement` and `proj.regulated_design` from a snapshot, with source `profile`.
   - `ValidateSetting` rejects `driver:flag.flood` and `field:client`.
   - Checks that used to point at `driver:x` now point at `fact:x`.
2. Implement.
   - Templates that used `{field:client}`, `{field:lga}` and similar now read the driver text.
   - Tests that set `driver:*` settings (`resolve_test.go:112-134`, `report_settings_test.go:128-212`) now set the fixture drivers instead. The changed-source check for facts goes away along with the overrides.
3. Migration (D2): `DELETE FROM report_settings WHERE key LIKE 'driver:%' OR key IN ('field:client','field:address','field:lga','field:zone','field:lot_dp','field:program','field:sustainability');`
4. Run `go test ./internal/rfp/... ./internal/store/...`. The answer keys must still pass; any change to their expected output is explained in the commit.
5. Commit.

### Task 4: Edit a fact in the report and save it to Profile

**Files:**
- Modify: `internal/store/report_rfp.go`, `internal/store/profile.go`, `internal/httpapi/projects.go` (route and path class `pathProfileEdit`), `web/src/reportApi.ts`
- Create: `bench` case `report_fact` in `bench/budgets.json` and its driver
- Test: `internal/store/report_fact_test.go`, `internal/httpapi/report_fact_test.go`

**Steps:**
1. Write failing tests for `PUT /reports/{id}/facts/{fact}` with body `{value, version}`:
   - The value lands in Profile: `GET /profile` shows the user value.
   - The returned draft shows the new value with source `profile`.
   - A stale draft version returns 409, a bad value returns 422, an unknown fact returns 422, and a wrong org returns 404.
   - `fact=budget` returns 422, because the budget is read-only.
2. Implement `Store.SetReportFact`. It runs in one transaction:
   1. Lock the project.
   2. Map the fact to its Profile home (the table above).
   3. Write it through a variant of `SetUserValue` that takes a transaction. Address and lot go to the site row instead.
   4. Patch that one fact in the draft's frozen `rfpState.Drivers`.
   5. Re-resolve, apply edits, cite, and save.
   - Move the tail that `SetReportSetting` and this function share into `rewriteRFPDraft`.
   - Other drafts on the project go stale through the existing source checks. Add a test that proves it.
3. Add the bench and run it. Report p50 and p90 against 150 ms and 400 ms.
4. Commit, giving the timings in the message.

### Task 5: Issue-detail defaults per discipline (D3)

**Files:**
- Modify: `internal/knowledge/rfp.go` (an optional `field_defaults` map on the discipline, checked against the skeleton's `issue` fields), `internal/rfp/resolve.go` (discipline default, then skeleton default), `tools/check_knowledge.py`
- Test: `internal/knowledge/rfp_test.go`, `internal/rfp/resolve_test.go`

**Steps:**
1. Write a failing test: a discipline with `field_defaults: {pi: "$5 million"}` resolves `field:pi` to that value with source `default`, and a user setting still wins.
2. Implement. No discipline YAML gets new figures in this change: structural keeps $10 million and $20 million until the owner sets otherwise.
3. Commit.

### Task 6: One report, edited section by section

**Files:**
- Modify: `internal/rfp/controls.go` (add `Section string` to `Group`; project→`project`, stages/elements→`stages`, allowances→`allowances`, positions→`positions`, issue→`terms`, weights→`evaluation`), `web/src/Reports.tsx`, `web/src/RfpEditor.tsx` (export the row and control parts; remove the aside), `web/src/reports.css`, `web/DESIGN.md`
- Test: `web/tests/rfp-editor.spec.ts`

Use @impeccable for layout and states before writing UI code.

**Steps:**
1. Rewrite the Playwright test as a failing test:
   - No `complementary` region named "Finish this RFP" exists.
   - The banner "Decide before issue" lists the blocking items.
   - Clicking "Edit" on the Project section shows the zone control. Typing `R2` and moving focus updates the report text and marks the zone "Profile".
   - After a reload, Profile shows `R2`.
   - Scope-position and allowance edits still update their sections.
2. Implement:
   - The report runs full width.
   - Each section heading has an Edit toggle that opens that section's controls directly under the heading.
   - In the Project section, facts write through Task 4 and show "Saves to the project profile". The budget and the live-operations choice link out instead.
   - Each decision in the banner opens its section and focuses its control. Advisory checks sit in a collapsed "Other checks" at the end.
3. Update `web/DESIGN.md` line 195 to describe the new layout.
4. Rebuild `web/dist` through bash (`dev.ps1 -Build` is broken in pwsh) and run the Playwright RFP specs.
5. Commit.

### Task 6b: Rename the Reports tab to Procurement (D4)

User-facing labels only: the tab button in `web/src/Project.tsx`, the
`<h1>` and `aria-label` in `web/src/Reports.tsx`, and the Playwright specs
that find them. Code identifiers, routes and tables keep the name
`report`. PMP stays in this tab for now; the owner will move it to its own
tab later.

### Task 7: Verify and report

1. Run `go test ./...`, `python tools/check_knowledge.py`, all Playwright specs, and the benches for `report_fact`, `report_setting`, `report_read` and `report_assemble`.
2. In the browser, open Petersham → Reports → Structural RFP and check the "Done" list above.
3. Report timings against budget, explain what calls what in plain language, and state that no new dependency was added. Merge to local main under the standing authorisation; do not push.
