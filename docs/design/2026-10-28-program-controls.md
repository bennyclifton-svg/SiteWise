# Program controls and shared file actions

28 October 2026. Owner direction: simplify Program and align the common file action across Project Profile, Cost Plan, Program, Procurement and Specifications.

## Outcome and scope

Make the chart the primary information surface and reduce time to find its controls. Layout is lane B. Saving the working week is lane A because it changes computed dates. No new services, dependencies, AI calls, holiday sources or scheduling engine changes.

The download icon sits above the shell, without a circular border. Its pop-out contains the current page's available actions. Profile and Procurement offer browser print; Cost Plan preserves its Excel/CSV exports; Specifications offers draft CSV and print; Program offers Excel, CSV, print and file import. Issuing and frozen specification exports keep their existing workflow.

Program has one expand/collapse chevron next to Name. Zoom is explicitly labelled. The summary, completion range, targets, question strip and checks text are removed from the screen. Critical cycle errors and actionable proposal review remain. Hold-point releases are behind an explicit button. Accept all is labelled Accept proposed values: it confirms generated rows and applies eligible imported proposals using the unchanged acceptance rules, with Undo.

Calendar offers Monday-Friday, Monday-Saturday or custom weekdays (at least one). Save uses the existing versioned, organisation-scoped operations endpoint with set_working_week, recomputes in Go and records an inverse for Undo/Redo. Existing day overrides remain. Date constraints stay dates; durations remain working-day counts. Failed writes retain the existing error/reload handling. No migration is needed because working_week is already stored.

## Fit defect

A repeated fit could set the same zoom state, so React skipped rendering and the queued scroll reset waited indefinitely. The browser regression reproduced a 912 px scroll remaining instead of returning to 48 px. Each explicit fit now requests a render even when the scale is unchanged. The regression passes.

## Validation

- TypeScript and Vite build pass (existing large-bundle warning remains).
- 234 frontend unit tests pass.
- All TestProgram store tests pass, including calendar persistence, recomputation, Undo/Redo and invalid masks, plus existing organisation isolation and stale-write checks.
- Store read p50/p90: 7.7/8.5 ms; write: 12.4/13.1 ms, within the existing 100/250 ms path budgets. These are local store measurements, not production HTTP percentiles.
- Browser first paint with 1,000 activities: 82.6 ms (budget 150 ms).
- Browser scroll with 2,000 activities: no tasks above 50 ms.
- Browser fit with 2,000 activities: screen 14.5 ms, selection 7.1 ms, height 10.7 ms (budget 16 ms).
- Focused browser coverage includes repeated fit, working-week save and Undo, rejecting an empty week, master disclosure, and switching the shared action menu between pages.

The full filing benchmark and entire browser suite were not rerun for this change. Existing Program tests describing the old question strip and summary need updating to the new owner direction before that suite is a useful full-screen acceptance gate.

## Owner correction: panel-aligned actions

The follow-up on 28 October moves file actions from the screen corner into a shared row immediately above the middle panel. Profile's update action is on the left of that row, with the download icon on the right. The documents toggle sits above the left edge of the right panel, separately from the theme control. Both text actions use the existing neutral `.btn` style; the custom rectangular Profile override is removed.

Documents now start visible on every surface, including Cost Plan. Explicit hide/show remains a shared view preference during navigation. When hidden, Show documents is reachable in the workspace action row. On narrow screens, the document controls stay with the stacked document panel.

This correction is lane B: no backend, filing, AI, export or calculation changes and no dependencies. Existing read/write budgets remain 100/250 ms (Profile retains its tighter budget); latency benchmarks were not rerun for this layout-only change. Build and two navigation browser tests pass, covering alignment, shared styling, documents remaining visible between tabs, explicit hide/show, refresh and mobile horizontal overflow. Desktop Profile/Cost Plan and mobile screenshots were inspected.
