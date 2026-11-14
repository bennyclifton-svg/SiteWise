# Tender comparison slice 3: the first table on screen

Date: 14 November 2026. Branch `tender/comparison`.

## What was built

- `web/src/tender/`: `tenderApi.ts` (contract C5 types, the operations
  client), `words.ts` (every word the tables show, with its tests),
  `AsTendered.tsx`, `LinesPanel.tsx`, `useTenderEdits.ts`, `tender.css`.
- `Prices.tsx`: the firm head lifted into `firmHead` and rendered by both
  tables; in tender mode (a firm of the package went to the tender pass)
  the two tender tables stand in for the old grid and the consultant
  comparison. Rates, earlier proposals, the firm panel and the award dialog
  are unchanged.
- `store.ApplyTenderOps` with `check_column`, and
  `POST /projects/{id}/procurement/packages/{pkg}/tender/ops`.
- Award: `awardPlan` refuses a tender-read submission until its column is
  checked ("Check this firm's column before awarding."), and awards a
  checked one on its priced cells, "Not broken down" included.

## What the manager sees

One table, "As tendered, ex GST": the schedule rows, then "Not broken
down", then "Tender sum". A cell is a button that opens the firm's own
lines behind it (wording, amount, page, flags, "not counted"). High, Low
and "Others priced this" are small tags; an amber dot and the cell's left
edge mark a figure to check. Under each firm's name: "To check" and "Mark
checked", "Read again", or "Prices not read" with "Try again". A column
read the old way says so and offers "Read prices" when the model is on.
The page number opens the filed document in the middle panel; the viewer
cannot be sent to a page, so the number stays as text beside it.

## What calls what

The screen reads `GET .../prices` (member `tender`). "Mark checked" posts
one `check_column` operation with the table's version; the answer replaces
the table. Award calls the existing award routes; `awardPlan` reads
`tender_state` and `tender_checked_at`.

## What could fail

A stale version answers 409 and the screen reloads and says "This table
changed. It has been reloaded." A column that has not been read, or whose
lines add to more than its tender sum, cannot be checked (422, in words).
Another org gets 404.

## Timings

`price_table_tender`: p50 12.5 ms, p90 13.3 ms (budget 150 and 300 ms),
five firms of 330 lines each, now that the screen reads it.

## Checked in the browser

`web/tests/tender-table.spec.ts` (2 tests) and
`web/tests/account-model.spec.ts` pass. Screenshot: `.tools/tender-table.png`
(1440 by 1000).

`builder-comparison`, `comparison-alignment` and `procurement` still pass.
`procurement-award.spec.ts` (2 tests) and `proposal-reading.spec.ts` (1)
fail, **and fail the same way on unmodified `main`** (checked on a clean
checkout of the starting commit with its own database).

## Where the plan was wrong, or was changed

- A column cannot be checked when its *lines add to more than the tender
  sum* (the gap is negative). The plan said "when the system-row cell is
  negative"; that cell also holds counted lines with no row, so it can be
  zero while the lines are over.
- The waiting state of an unread column is shown in the first body cell,
  not one tall cell: a row-spanning cell breaks when a lines panel opens
  between rows.
- A reload returns to the project, not the price table (the open package
  is not in the URL). The browser test opens the table again.
- The "Checked" button shows "Checked ✓" and the separate state word is
  dropped once checked, so the head does not say "Checked" twice.
