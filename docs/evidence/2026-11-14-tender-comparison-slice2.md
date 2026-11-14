# Tender comparison slice 2: switch, storage, the background job, the table read

Date: 14 November 2026. Branch `tender/comparison`. Built although the
gate was not a clean go (`2026-11-14-tender-comparison-gate.md`); the org
switch is off by default.

## What was built

- Migration `055_tender_comparison.sql`: `orgs.model_enabled_at/by`,
  `packages.tender_version`, the tender columns on `submissions`, and
  `tender_lines`.
- `store`: `ModelEnabled`, `SetModelEnabled`; the "Not broken down" row
  (`ensureTenderRow`, schedule key `tender:not_broken_down`);
  `TenderInput`, `QueueTenderRead`, `ClaimTenderRead`, `FailTenderRead`,
  `PendingTenderReads`, `DueTenderRetries`; `CommitTenderRead` with
  `refreshTender` and `projectTenderCells`; the tender member of the price
  table (`packageTender`, `applyTender`).
- `tender.Marks` (high, low, gap) and `tender.OnRows`.
- `httpapi`: `tenderReader`, `PUT /account/model`,
  `POST /projects/{id}/submissions/{sub}/tender-read`, the `tender` member
  of the prices JSON (contract C5), `model` in `GET /account`.
- `internal/tender/sample`: a stand-in model for tests, browser tests and
  the bench.
- Account screen: "Reading tenders with a model", shown to the owner when
  the server has a key.

## What calls what

A tender is dropped on a package. Filing and the first read are unchanged
(rules and one Jev fan-out). When that read commits, its `after` hook calls
`tenderReader.Queue`. `QueueTenderRead` marks the submission `pending`
only when the org's switch is on, it has a package and a file, and it is a
live proposal. `Start` claims it (`reading`), `TenderInput` gives the
package's priced rows and the file, text is extracted, pages with no text
go through OCR (up to thirty), `tender.Pass` makes one model call,
`tender.Reconcile` checks and reconciles, `CommitTenderRead` stores the
lines, sets the tender sum and flags, writes the price cells from the
counted lines, bumps `packages.tender_version` and appends a procurement
event, which the open screen hears.

Placing a submission in a package (`PATCH`) queues it the same way. "Read
again" queues it from `read` or `failed`.

Award, the comparison report and the cost plan still read
`submission_prices`. For a tender-read submission those cells are written
only from its lines; the old reader then touches hourly rates alone.

## What could fail, and what the screen is told

| Cause | State | Words | Tried again |
|---|---|---|---|
| No key on the server | never queued | (tenders read the old way) | |
| Org switch off | never queued | | |
| Model unreachable, rate limited, timed out, bad JSON | failed | "The model could not be reached. It will be tried again." | after 30 s, 2 min, 10 min; then "The model could not be reached." |
| The model lists nothing for price pages | failed | "The model listed no prices. It will be tried again." | same schedule |
| The model declines | failed | "The model declined to read this file." | no |
| No page renderer (Python helper missing) | failed | "Page images cannot be made on this server." | no |
| File kind not readable | failed | "This kind of file cannot be read for prices." | no |
| File missing or unreadable | failed | "This file could not be opened." | no |
| Answer cut off | read | column flag "The reading was cut off. Some lines may be missing." | |
| Server restarted mid-pass | pending again at start-up | | yes |

A failed pass leaves the old reading's cells in place.

## Timings

| Path | Budget p50 / p90 | Measured |
|---|---|---|
| `price_table_tender` (five firms, 330 lines each) | 150 / 300 ms | 14.7 / 15.7 ms (`TestTenderTableFiveFirmsBudget`) |
| `tender_reconcile` | 100 / 250 ms | 1.6 / 2.1 ms |
| `tender_read`, `tender_read_long` | 15 / 30 s, 60 / 90 s | live figures in the gate note; these two are a live gate run with `cmd/tender-eval`, not rows of `bench/budgets.json` (design section 12) |

All lines of a package are read in one query. Neither fallback the plan
names was needed.

`cmd/intake-bench/tender.go` samples `tender_reconcile`,
`price_table_tender` and `tender_edit`. **The full bench was not run in
this worktree** (it needs the private intake corpus and recordings); the
same budgets are held by tests that were run.

## New dependency

None in this slice. Slice 0 added `anthropic-sdk-go`.

## Where the plan was wrong, or was changed

- Tests could not use the shared `a`/`b` e2e orgs with the switch on
  without changing what other browser tests see; tender browser tests sign
  up a fresh account each.
- `tender_sum_read_cents` and `tender_sum_read_how` are in the first
  migration, not slice 4's: the commit needs them to restore the pass's
  sum when a user's sum is undone.
- The migration test lives in `internal/store/tender_test.go`
  (`TestTenderMigrationConstraints`) where the fixtures are; the
  foreign-key inventory in `internal/db` gained the new table.
- Re-reading: a line the manager moved, split or corrected stands in for
  the same line read again (matched on page and wording or figure).
  Without that, "Read again" counted every corrected line twice. A figure
  typed into a cell before the first reading becomes the manager's line
  for that row, and the model's lines on that row are kept beneath it,
  uncounted.
- A scan is OCRed up to thirty pages for the tender pass (four for the old
  reader), in batches of four.
- `tender_read` is timed from the upload (the first read's start), so it
  includes filing and the Jev fan-out.
- Switching the model on does not send anything by itself: submissions
  already on a package are read when the manager asks ("Read prices" in
  the column head, slice 3).
