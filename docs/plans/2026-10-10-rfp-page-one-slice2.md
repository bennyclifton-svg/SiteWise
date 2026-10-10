# RFP page one, slice 2 (lane B): the screen

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** The RFP report view shows page one always open and edited where it
prints, puts each blocking check beside what it concerns, closes with an
Issue section, and drops the "Decide before issue" and "Draft uses saved
project information" blocks.

**Design:** `docs/plans/2026-10-10-rfp-page-one-design.md`. Slice 1 (lane A,
`docs/plans/2026-10-10-rfp-page-one-slice1.md`) supplies the data: block
`rfp:project:grid` carries `basis.grid` (columns of cells with control
keys), page-one condition controls are exactly the printed conditions, and
the controls are grouped `project` (section project), `appointment` (role),
`terms` (terms) and `issue` (the screen's closing section).

**Lane B rules.** Do not break filing or issue. The issued PDF is server
rendered and already carries no controls; this slice only changes the
screen. No new dependency.

**Tech:** React + TypeScript (`web/src`), Playwright (`web/tests`). Build and
test from bash: `cd web && npm run build && npx playwright test <spec>`.
The e2e server port 4173 may be held by another session (memory
cost-plan-grid): use a copy of the config on another port if so.

## What the user sees

1. **Toolbar.** Title, page fit, Refresh. When the draft is stale the button
   reads "Refresh: Profile changed" (or the matching reason), with every
   reason in its title. The stale block below the toolbar is gone (all
   report kinds). The RFP's Issue button and form leave the toolbar; other
   report kinds keep theirs.
2. **1. Project** (always open, no Edit button):
   - Issue strip as printed; clicking it jumps to the Issue section.
   - Opening sentence, as today (wording can still be edited).
   - Project grid in the Profile card's layout (reusing `.pf-card`,
     `.pf-col`, `.pf-row`, `.pf-label`): each value is a button that opens
     its control in place; a locked value (derived, linked, cost plan) shows
     its text and why. Source pill on every value.
   - Conditions: one compact row per condition with Yes / No / Not known.
   - Information provided, editable in place.
   - Budget line with its Cost plan pill (shown only on screen when not
     disclosed, as today).
   - A value whose control a blocking check names is outlined red.
3. **2. Your role** gains an Edit toggle: the stage range ("This appointment
   buys"), the parts covered, scope choices and the discipline's own inputs.
4. **3 to 7** keep their Edit toggles. Stages lose the range picker (now in
   2). Terms show the agreement and commercial terms.
5. **Markers.** Under each section heading, the blocking checks whose control
   lives in that section, each with Show (opens and focuses it). Page one
   uses the red outline plus the same list.
6. **Issue** (after the returnable schedules, screen only): status line (N
   items block issue · Show first, or Ready to issue), the draft-rules note,
   issue details as a compact grid, any blocking check with no section,
   Other checks (advisory), reporting date, budget disclosure, the
   stale-state tick and reason, and Issue report.

## Tasks

### Task 1: Editor pieces (`web/src/RfpEditor.tsx`)

- `controlSection`: `boundary:range` → `role`; `scale` → `stages`.
- `editableSections`: add `role`; stop forcing `stages` unless it has
  controls or scale applies (it always has scale, so keep it).
- `RfpSectionEditor`: `role` renders `Boundary` then the appointment group;
  `stages` renders `Scale` only above its groups. Remove `ProjectGroup`.
- New `ProjectPage` (page one): props `blocks`, `controls`, `busyKey`,
  `onWrite`, `onJump`, `blocked` (set of control keys named by blocking
  checks). Renders the strip, intro (via a render prop for the existing
  block view), the grid from `basis.grid`, conditions, information, budget.
- New `GridValue`: closed shows the printed text as a button
  (`.rfp-cell`), open shows `Control` (autofocus); a choice or tristate
  closes on change, typed closes on commit or Escape.
- New `SectionMarkers` and `RfpIssue`. Remove `RfpDecisions`.

### Task 2: Report view (`web/src/Reports.tsx`)

- RFP sections: `project` renders `ProjectPage`; others render markers, the
  Edit toggle and the existing blocks. Hide "Edit wording" on grid and
  condition blocks (their values edit in place).
- After the sections, `RfpIssue` with the issue form state moved from the
  toolbar.
- Stale label on Refresh; remove the stale block; `rfp_open_checks` error
  text says "Resolve the items marked in red first".

### Task 3: Styles (`web/src/reports.css`)

`.rfp-page-one` spacing, `.rfp-cell` (text button: no border, dotted
underline on hover, focus ring), `[data-blocked]` red outline,
`.rfp-conditions` (auto-fit columns of rows), `.rfp-markers`, `.rfp-issue`
(compact grid, `repeat(auto-fit, minmax(14rem, 1fr))`), and `@media print`
hiding pills, buttons, markers and the Issue section.

### Task 4: Tests (`web/tests/rfp-editor.spec.ts`)

Update the spec to the new layout, and cover: page one has no Edit button
and shows the three grid columns; changing a grid value saves to Profile
(the fact request fires and the value shows); a condition answer saves; a
blocking marker's Show focuses its control; the Issue section lists issue
details and Issue report stays disabled while blocking checks remain; the
stale block is gone and Refresh carries the reason. Run the spec, the
costs and procurement specs (shared report view), `npm run build`, and
`tsc -b`.

## Done when

`npm run build` passes, the RFP specs pass, and a browser check of the
Petersham architecture RFP on the dev server (`/dev/login`, :8080) shows the
layout above.
