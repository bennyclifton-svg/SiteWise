# Profile scope, reading selection and deletion: evidence

Date: 3 October 2026. Branch `profile/scope`. Design:
`docs/design/2026-10-03-profile-scope.md`. Plan:
`docs/plans/2026-10-03-profile-scope.md`.

## Why (dev database, before the change)

"Update project profile" read every document with text. On `sitewise_dev`:

| | Drawing | Specification | Design brief |
|---|---|---|---|
| Spec Home passages | 6,513 | 183 | – |
| Label calls, all projects | 6,510 of ~10,500 | 1,626 | 238 |
| Passages mapped to anything | 2.4% | 17% | 45% |

Every wrong applied header value on Spec Home came from a drawing:
`garage_spaces = 230` (wall dimensions "GARAGE 230 15,720"), `bedrooms = 1`
("MIN. 1 LIVING AREA & 1 BEDROOM"), `beds = 3` (room labels), and
`state = NSW` was read 137 times from title blocks.

## Timing (local HTTP tests, real store, code-only rebuilds)

| Path | p50 | p90 | Budget |
|---|---|---|---|
| `project_profile_read` (now computes relevance at read) | 5.6 ms | 6.9 ms | 50 / 150 ms |
| `profile_edit` (value, reading setting, scope) | 11.3 ms | 12.0 ms | 50 / 150 ms |
| `document_delete` (one document) | 2.7 ms | 4.0 ms | 500 / 1,000 ms |

Before this change the same profile tests measured read 2.6–2.8 ms and edit
6.4 ms. `cmd/intake-bench` now also times reading-setting and scope writes
under `profile_edit` and 20 deletions under `document_delete`; the full
replay bench was not run (its intake recordings are stale, see
`docs/design/2026-10-03-profile-brief-accuracy.md`).

## Relevance, examples from the real schema

| Scope | Rules | Profile determinants shown |
|---|---|---|
| Sprinkler pump replacement (`fire-active.sprinklers`, NCC 7b) | 4 | 6 of 28: effective_height, ncc_class, ncc_edition, sprinklered, state, storage_height |
| Typical fit-out preset | 66 | 9 of 28 (includes `bal` through a rule on a fit-out system: a schema item to review) |
| New house defaults, class unknown | 130 | 22 of 28 |
| New warehouse defaults | 136 | 24 of 28 |
| New apartments defaults | 158 | 25 of 28 |

Rows limited to other classes drop out once NCC class is known; an unknown
value never hides a row. With nothing in scope, the API shows every row.

## Checks

- `go test -p 1 ./...` (private database on port 5434), `go vet ./...`: pass.
- `python tools/check_knowledge.py --strict`: 0 errors, 0 warnings;
  `python -m unittest discover -s tools`: 19 pass.
- Playwright: 10 pass (intake, OCR, profile, plus new reading selection,
  deletion and scope specs).
- Private profile regression replays: source suite 15/15 and Hale 12/12
  readings applied, 0 forbidden. Jev question wording is unchanged: the
  refurbishment rename uses a new `display_label`, because renaming `label`
  changed the recorded header question and made both recordings stale.

## Deviations from the plan

- Relevance is computed when the profile is read, from the stored scope rows,
  not stored with the rows. It costs about 3 ms per read.
- Scope rows reuse the existing band values (`suggested`, evidence bands,
  `user`) instead of new ones, so no migration was needed for them.
- No separate `profile_rebuild` budget: every profile edit includes the
  rebuild and is timed as `profile_edit`.
- A document has exactly one file row (`documents_file_uq`), so deletion
  always removes its file row; bytes are kept while any org references them.
- `restore-check` now exempts a deleted document's event history, using its
  `deleted` event.

## Not done / for the owner

- Category default lists, the fit-out preset and determinant `systems` links
  are drafts for owner review (`status: draft`).
- The deploy package ships neither `knowledge/` nor `data/profile/`, and the
  service passes no `-knowledge` flag. This predates this branch, but `serve`
  now also needs `data/profile/reading.json`.
- A blob younger than two minutes, or one whose reference check fails, is left
  as an orphan. Nothing sweeps orphans yet (`intake.RecoverOrphans` exists but
  is not called).
