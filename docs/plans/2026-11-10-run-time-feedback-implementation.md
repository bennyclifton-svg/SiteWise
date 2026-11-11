# Run-time feedback implementation plan

> **For Claude:** executed inline in session sitewise-6b (owner away, no
> subagents). Design: `docs/design/2026-11-10-run-time-feedback.md`.

**Goal:** a Profile update shows what the reading really does, when it does
it; the scope is on the first screen; the page says what changed and what to
do next.

**Architecture:** the browser diffs the previous and next profile (keyed by
revision) and marks the changes; two small reads feed the run strip (document
progress, passage finds); one small read counts what the update changes
elsewhere. Motion is CSS on stable data attributes. No Jev call is added and
nothing changes what is read, stored, decided or accepted.

**Tech stack:** Go handlers and pgx store reads; React 19 + TypeScript; CSS;
Vitest for pure functions; Playwright against the e2e server.

**Owner decisions (10 Nov 2026):** Update and its menu move into the header
band; the Counts row appears on Profile and on RFP page one; summary
underlines follow the 8 November origin rule; no time estimate; changed
marks are remembered per browser.

**Shared tree:** other sessions have uncommitted work in Profile.tsx,
ScopeWorks.tsx, trace-profile.css, Project.tsx, internal/rfp and
internal/knowledge/rfp.go. Use Edit only; new code goes in new files. Do not
commit (owner asks for commits; shared files hold others' work).

---

## Slice 1: layout

### Task 1.1 Header slot for page actions
- Modify `web/src/CockpitFrame.tsx`: a `headerTarget` div first in
  `.cockpit-header-actions`; add it to `PageActionsContext`.
- Modify `web/src/PageActions.tsx`: `PageHeaderAction({ surface, children })`,
  a portal into `headerTarget` when the surface is active.
- Modify `web/src/Profile.tsx`: Update and the three-dot menu render through
  `PageHeaderAction surface="profile"`; the old `pf-head` row renders only
  while Add part is open.

### Task 1.2 One-line title block rows
- Create `web/src/profile-run.css`, imported in `main.tsx` after
  `trace-pages.css`.
- `.pf-titleblock.pf-paired .pf-row`: three columns (label, value, how it was
  decided), `.pf-pair-head { display: contents }`, 30 px rows; the red mark
  shows its icon, not its words; the value cell's edit box stops at the
  provenance column.

### Task 1.3 Counts row on Profile
- Create `web/src/ProfileCounts.tsx`: `CountsRow` for bedrooms, bathrooms,
  car parks (or garage spaces when it says something different). Each value
  is its own input with its own dot and source count; the row is labelled
  "Counts"; each input keeps its field label as its accessible name.
- Modify `ProjectCard` in `Profile.tsx`: the count keys leave their own rows;
  one Counts row takes the first count's place.

### Task 1.4 Counts row on RFP page one
- Modify `internal/knowledge/rfp.go` `ProjectGrid`: `bedrooms`, `bathrooms`
  and `parking` become one `{"counts", "Counts", "The building", false}`.
- Modify `internal/rfp/resolve.go` `gridText`: `counts` joins the known
  numbers as "4 bedrooms, 3 bathrooms, 1 car park" (singular at 1; a
  conflict reads "to be confirmed").
- Tests: `internal/rfp/house_test.go` expects the counts cell; regenerate the
  three golden RFPs that print counts and check the diff is the grid only.

### Task 1.5 Strip, chevron, scope rows
- Create `web/src/ProfileStrip.tsx`; the rest state is one line: updated
  time, documents read (and not read, with Show), Source requirements, "to
  check" with Go, and the next step at the right.
- CSS: View more as a chevron tab on the card's bottom rule; scope entry rows
  34 px.

### Task 1.6 Check
- `npx tsc --noEmit`, `npm test`, the Profile Playwright specs.
- Measure on Petersham (dev data) at 1536 × 864: five or more scope rows
  visible on arrival.

## Slice 2: inked, held, changed marks, summary, receipt

### Task 2.1 Profile diff (pure)
- Create `web/src/profileDiff.ts`: `snapshotOf(profile)` (key and part to
  value, band and source count; scope leaf to in, check; revision) and
  `diffSnapshots(before, after)` returning changes:
  `filled | changed | cleared | conflict | held | sources` for fields,
  `in | out | check` for scope. A field whose band is user and gained a
  disagreeing alternative is `held`.
- Test `web/src/profileDiff.test.ts` (Vitest): filled, changed, held, conflict,
  scope in, out and check, sources-only, and no change across equal
  snapshots.

### Task 2.2 Summary underlines follow the origin rule
- `web/scripts/make-marks.mjs`: add `pencil-green`, `pencil-green-dot` and
  dark versions; run it; existing marks must not change (git diff).
- `trace.css`: `.pencil-doc` solid green, `.pencil-jev` dotted green (a Jev
  reading of the documents, as the card's ring), `.pencil-typical` dotted
  blue, `.pencil-user` rust.
- `ProjectSummary`: suggested band uses `.pencil-typical`; each token carries
  `data-key`. Update DESIGN.md and PRODUCT.md wording.

### Task 2.3 Ink presenter
- Create `web/src/profileInk.ts`: `inkChanges(root, changes, opts)` finds
  elements by data attribute and plays one class each, staggered in reading
  order (card by column then row, scope in building order), 900 ms cap, no
  stagger over 40 changes; skips reduced motion, a hidden tab and elements
  outside the scroll view, and returns the off-screen count.
- CSS keyframes in `profile-run.css`: write, draw, pop, cell fill (hatch then
  fill), rub out, check, roll, glint.

### Task 2.4 Changed marks
- `FieldRow` takes `changed?: string` (the tooltip): `data-changed` on the
  row; `data-key` on every row.
- `ScopeWorks` takes `changes?`: entry rows get `data-entry` and
  `data-changed`; cells get `data-leaf` and, when changed, `data-new` and a
  title. Nothing else in ScopeWorks moves.
- Memory per browser: localStorage `sitewise.profile.seen.<project>` holds the
  seen snapshot; marks are the diff from it; Mark all seen saves the current
  snapshot. Storage failures leave marks for the visit only.

### Task 2.5 Receipt
- `ProfileStrip` done state: documents read, passages, elapsed time (all
  measured), then chips: filled, changed, systems added, removed, to check.
  A chip scrolls to its first item and flashes them.

## Slice 3: the run strip

### Task 3.1 Progress read
- Store `ProfileProgress(ctx, org, project)` in `internal/store/profile_run.go`:
  documents with a queued, leased or failed full-text, label or evidence job:
  state (`queued | reading | failed`), pass (`label | evidence`), units,
  labelled, evidence; plus the profile revision.
- Handler `GET /projects/{id}/profile/progress` (`profile_run.go` in httpapi),
  latency path `project_profile_read`.
- Test: shape, foreign org 404, budget p50 50 ms / p90 150 ms.

### Task 3.2 Run state in the browser
- Create `web/src/profileRun.ts`: API and `useProfileRun`: polls progress each
  second while anything is pending (and right after Update), remembers every
  document seen in this run, marks one done when it leaves the list, and
  asks for the full profile only when the revision moves (the 2 s full poll
  goes).
- Strip run state: one sheet per document (hatched to labelled, inked to
  evidence, a tick per committed pass), the live line, failure first.
- Away: Profile reports its run state up; the Project Profile nav item
  carries the activity ring while reading and a count of changes after; the
  tab title reads "Reading 2 of 3 · Project". One live announcement per
  stage.

## Slice 4: spotted

### Task 4.1 Finds read
- Store `ProfileFinds(ctx, org, project, docs, after)`: label and evidence
  calls on those documents' passages after the time, oldest first, at most
  200: passage, document, stage, page, mapped keys, systems, a 120-character
  excerpt, time.
- Handler `GET /projects/{id}/profile/finds?docs=…&after=…` (at most 20
  documents; bad input 400; documents outside the project ignored).
- Test: isolation, bad input, budget.

### Task 4.2 Glints
- Poll finds with progress (after = last time − 2 s, de-duplicated by passage
  and stage). Keys glint their rows and add a pending "+n"; systems light
  their scope row's ring and cell; the register row glints; the strip shows
  the latest find. Never more than one glint per field each 600 ms.

## Slice 5: next

### Task 5.1 Next counts
- Store/handler `GET /projects/{id}/profile/next`: procurement suggestions not
  yet packages, cost suggestions not yet lines, pending program review
  changes, RFP and RFT drafts built from an older profile revision. Latency
  path `packages_read`.

### Task 5.2 Next in the strip and nav
- Rest state next step, first that applies: retry a failure; to check; new
  documents; unknown RFP key facts (zone, lot and DP) with the field; work
  elsewhere out of date; else nothing.
- While reading: unknown budget, procurement route and program milestones as
  inputs (the user's word).
- After a run: "What this changes elsewhere" with links (Project passes
  `onOpenView`); the same counts on the left nav until that view is opened.

## Verification
- `go test ./internal/rfp/ ./internal/knowledge/ ./internal/httpapi/ -run 'Profile|Grid|House'`
  and `./internal/store/ -run Profile`, `gofmt -l`.
- `npm test`, `npx tsc --noEmit`, `npm run build`, Playwright: profile,
  activity, and an RFP spec that reads the grid.
- Report timings against the budgets, what calls what, what could fail.
