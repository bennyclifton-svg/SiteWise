# Tender comparison slice 4: editing the first table, with undo

Date: 14 November 2026. Branch `tender/comparison`.

## The change log came first

`internal/store/change.go` was not on `main`. The plan says to stop and
tell the owner rather than build a second undo. The owner's instruction for
this run was not to stop, so tasks 1 to 4 of
`docs/plans/2026-11-14-change-log-undo-operations.md` were built here, as
that plan specifies, so that there is still one undo:

- `056_project_changes.sql`: `project_changes`, `project_change_rows` and
  the `capture_change()` trigger on the Cost Plan and Profile value tables
  that plan lists.
- `store.WithChange`, the marked pool, `store.UndoChange` with its
  registry of captured tables, and
  `POST /projects/{id}/changes/{change}/undo`.
- `TestCapturedTablesMatchDatabase` holds the registry to the database: it
  caught four primary keys the plan had guessed wrong.

**Not built from that plan:** tasks 5 to 10 (the operations registry, the
apply endpoint, Cost Plan and Profile operations, their benches). No Cost
Plan or Profile write is marked yet, so nothing but tender edits is
undoable. The cost and profile finishers exist and are untested by an
end-to-end undo. If another session builds that plan, it should start from
task 5 on top of this branch; its migration number 055 is taken.

## What was built for tenders

- `internal/tender/ops.go`: `Apply`, a pure function for `move_line`,
  `split_line`, `unsplit_line`, `set_line`, `add_line`, `count_line`,
  `delete_line` and `set_tender_sum`, with `OpError` naming the field.
- `057_tender_changes.sql`: area `procurement`, and the capture trigger on
  `tender_lines`.
- `store.ApplyTenderOps`: one transaction, all or nothing, marked as a
  change when the batch holds anything undoable; it returns the change id.
  `refreshTender` runs after every edit and every undo.
- On screen: right-click or "⋯" on a line (Move to each row, Split,
  Correct, Take out of the count, Count, Take out as a subtraction, Put
  back together, Delete); M on a selected line opens the same list and the
  arrow keys walk it; double-click or F2 corrects; a ghost row adds a
  line; drag a line or a whole cell to another row of the same firm;
  "Correct" on the tender sum; "Undo" in the page head.

## What the change log captures, and what the finisher rebuilds

Captured: `tender_lines` (and, from slice 5, the two adjustment tables).
Not captured: `submissions` and `submission_prices`. After an edit or an
undo, `refreshTender` rebuilds them from the lines: the tender sum (the
manager's own total line if there is one, else the sum the pass chose,
kept in `tender_sum_read_cents`), the column flags, and the price cells.

## What an undo cannot put back

- A column's checked state: checking is not a change.
- A re-read: "Read again" replaces the model's lines and is not undoable.
- A change after which the same line was edited again: refused with
  "Can't undo: changed since".
- Only the user who made a change can undo it.
- The screen remembers changes for the session only. After a reload the
  Undo button is empty though the changes are still in the log.

## Timings

`TestTenderEditBudget`, five firms of 330 lines each, 24 moves and 24
undos:

| Path | Budget p50 / p90 | Measured |
|---|---|---|
| `tender_edit` | 100 / 250 ms | 31.7 / 33.3 ms |
| `change_undo` | 100 / 250 ms | 13.0 / 15.2 ms |

Only the edited submission's lines are loaded and rewritten; the
suggestions sync reads the package's other columns once.

## Checked in the browser

`web/tests/tender-edit.spec.ts` (2 tests): move by menu, move by keyboard,
correct an amount, split, set the tender sum, add and delete a line, undo
after each kind, and a stale table reloaded with its message. Dragging is
built but is not exercised by a browser test.

## Where the plan was wrong, or was changed

- `tender_sum_read_cents` and `tender_sum_read_how` are in migration 055.
- A line the manager added is known by its note ("Added by you"); only
  such a line, a figure typed for a whole row, or their own tender sum can
  be deleted.
- An awarded firm's lines are locked: "Undo the award before changing its
  lines."
- `change_undo` has its own budget row, as the change-log plan asks.
