# Run-time feedback: evidence

Date: 10 November 2026. Design: `docs/design/2026-11-10-run-time-feedback.md`.
Plan: `docs/plans/2026-11-10-run-time-feedback-implementation.md`. Lane B.
Not committed: the shared tree holds other sessions' uncommitted work in the
same files (Profile.tsx, ScopeWorks.tsx, Project.tsx, internal/rfp).

## What changed, in plain words

- **Layout.** Profile's Update and its menu sit in the title band. Card rows
  are one line: label, value, then the dot and source count. Bedrooms,
  bathrooms and car parks share one Counts row on Profile and on RFP page one
  ("4 bedrooms, 3 bathrooms, 1 car park"); the RFP's Word and PDF use the same
  grid. View more is a chevron on the card's foot rule. The strip at the foot
  rests as one line on a wide screen (two at 1536 px with the register open).
- **During an update.** The strip shows one sheet per document (hatched as
  passages are labelled, inked as they are read for evidence, a tick per
  committed pass), the live pass and passage count, the passage just read and
  what it spoke to, and the facts documents seldom settle (budget,
  procurement route, milestones) to state while waiting. A passage that
  speaks to a field glints its row (cyan) and adds a "+n" until its pass
  commits; scope rows whose systems are being read carry the activity ring.
- **On commit.** Values that changed write in where they are, the summary
  rewrites itself (a replaced word is struck above the line), scope cells
  shade in or rub out, and each change keeps a revision triangle with its
  old and new value until Mark all seen. Remembered per browser.
- **After.** A receipt (documents, passages, measured time; filled, changed,
  systems added and removed, kept, to check) with chips that point at the
  items, and what the update changes elsewhere: proposed disciplines, cost
  lines, program changes to review, RFP drafts built before the update. The
  same counts sit on the left nav until that view is opened; the nav's
  Project Profile carries the reading ring while you are elsewhere, and the
  tab title reads "Reading 2 of 3 · Project".
- **Summary underlines** follow the origin rule: green from the documents
  (dotted where Jev leans on them), dotted blue typical, rust the user's word.

## What calls what

- `GET /api/projects/{id}/profile/progress` (new): each document's queued,
  reading or failed pass with its passage counts, and the profile revision.
  Polled each second only while an update runs (five seconds in a hidden
  tab). It replaces the 2 s poll of the ~200 KB profile; the profile is read
  when the revision moves, or when the profile still says pending after the
  work is done (a missed completion event).
- `GET /api/projects/{id}/profile/finds?docs=&after=` (new): passages of the
  documents being read, finished after a time, at most 200, oldest first,
  with their keys, systems and opening words. From `passage_calls` and
  `passage_sources`, which the worker already writes per passage.
- `GET /api/projects/{id}/profile/next` (new): Procurement's suggestions,
  the Cost Plan's suggestions, pending program changes and drafts whose saved
  profile revision differs. Read once when an update finishes here.
- No Jev call is added. Nothing changes what is read, stored, decided or
  accepted. No migration, no dependency. Every read is org-scoped and a
  foreign project is a 404.

## Timings

Endpoint tests (`internal/httpapi/profile_run_test.go`), 25 calls each,
within p50/p90: progress 50/150 ms, finds 50/150 ms, next 100/250 ms (the
packages read's budget, its route path). Store test: finds over a
6,000-passage document within 50/150 ms.

Dev server, real data, 25 calls each from the browser:

| Read | Petersham p50 / p90 | Demo project p50 / p90 | Budget |
|---|---|---|---|
| progress | 1.7 / 3.1 ms | 1.6 / 2.1 ms | 50 / 150 |
| finds | 2.8 / 3.2 ms | 3.4 / 4.0 ms | 50 / 150 |
| next | 10.1 / 20.7 ms | 3.2 / 3.4 ms | 100 / 250 |
| profile (for comparison) | 21.0 / 23.1 ms | 10.1 / 11.3 ms | 50 / 150 |

## First screen (Petersham, dev server)

| Viewport | Scope rows visible before | after | Card height before → after |
|---|---|---|---|
| 1536 × 864 | 0 | 6 | 470 → 248 px |
| 1920 × 1080 | 2½ | 13 | 470 → 248 px |

## A live update

Project "Run feedback demo (Calloway Street)" on the dev server: the ST02
development brief, then its geotechnical report (live Jev reading, 23
passages, 5 s). Seen during the run: six card rows with "+n" from passages,
seven row glints, three scope rows ringed, the strip naming "Excavation next
to adjoining structures or boundaries" and then "Basement excavation,
retention and dewatering"; at commit the values inked and the source counts
rose (work type 8 → 9, storeys 2 → 3); the receipt read "2 filled · 3
systems added" with Procurement 16 and Cost Plan 24 elsewhere. The first
update on the brief filled 33 values and added 65 systems.

## Tests

- Go: `internal/store` (all pass, including the new finds test),
  `internal/httpapi` (the three new tests and the route sweep pass),
  `internal/rfp` and `internal/knowledge` (the house test and three golden
  RFPs updated for the Counts row; the diff is the grid row only).
- Vitest: `profileDiff.test.ts` (9) and `profileRun.test.ts` (5) pass.
- Playwright: `profile-run.spec.ts` (an update end to end with marks,
  receipt and nav counts; reduced motion; the first-screen fold at
  1536 × 864) passes; `profile.spec.ts` card geometry updated for one-line
  rows; `activity.spec.ts` passes.
- Failing before this work and untouched by it: `profile.spec.ts` "thin
  brief" and "sprinkler pump" (expect the "Project" heading and "N systems"
  removed on 8 Nov), `navigation-refresh.spec.ts` (Cost plan tab name;
  documents toggle position), `reports.spec.ts` (mobile nav collapsed),
  two `rfp-editor.spec.ts` tests (RFP section numbering and register
  checkboxes, from other sessions' work), `src/program/engine.test.ts` (3),
  `TestProgramRoutes`, `TestCommittedBudgets` (budgets.json has 81 paths
  since 7 Nov) and the Meridian answer key.

## What could fail

- A dropped poll is retried each second; only a lost session stops the page.
- Refused storage (a private window) keeps the marks for the visit only.
- A passage the evidence pass skips never fills its sheet to the top; the
  sheet is honest about what was read.
- A user's value that documents later contradict is noted ("read again,
  kept"); raising it as a conflict needs the reconcile step to compare the
  two, which is lane A and not in this change.
