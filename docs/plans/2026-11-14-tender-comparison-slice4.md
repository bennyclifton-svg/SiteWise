# Tender Comparison Slice 4: Editing the First Table

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** The manager moves a line to another row, splits it, corrects it, adds one, takes one out of the count and corrects the tender sum, by drag, right-click or keyboard, and can undo each change. No edit calls a model.

**Architecture:** Each edit is a named operation. `tender.Apply` is a pure function that applies one operation to a submission's lines and says what is wrong when it cannot. `ApplyTenderOps` loads, applies, stores, recounts and re-projects in one transaction marked for the project change log, which gives undo.

**Tech Stack:** Go, PostgreSQL, React, TypeScript, Playwright.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md` (contract C6). **Design:** sections 5 and 10.

**Status:** written 14 November 2026. Not compiled or run. Names marked *verify* were not opened.

---

### Task 0: The change log must be in

Open `internal/store/change.go`. It must define `Change`, `WithChange` and `UndoChange`, and the route `POST /projects/{id}/changes/{change}/undo` must exist (`docs/plans/2026-11-14-change-log-undo-operations.md`, tasks 1 to 4). If any is missing, stop and tell the owner. Do not build a second undo.

Read that plan's tasks 1, 3 and 6 now: how a table is registered for capture, what a finisher is, and how a write route is classed for its coverage test.

### Task 1: Operations as pure functions

**Files:**
- Create: `internal/tender/ops.go`
- Test: `internal/tender/ops_test.go`

**Step 1: Write the failing tests.** Each builds a few `Line` values by hand (invented figures) and calls `Apply`.

Move:
- `TestMoveLineChangesItsRow`; `TestMoveToNoRowIsTheSystemRow` (row `""`); `TestMoveToAnUnknownRowIsRefused`; `TestMoveMarksTheLineAsTheUsers` (`DecidedBy` becomes `user`, and the `row_unsure` and `row_unknown` flags are dropped).

Split:
- `TestSplitMakesPartsThatAddUp`: a counted line of 100,000.00 split into 60,000.00 on row A with the rest on row B gives two counted lines of 60,000.00 and 40,000.00, each with `SplitOf` the original's id, the original no longer counted, and the same page, heading, GST basis and label (the label gains " (part)").
- `TestSplitRefusesMoreThanTheLine`: parts of 120,000.00 on a 100,000.00 line.
- `TestSplitOfAnIncGSTLineKeepsItsBasis`: a line printed 110.00 inc GST split with a part of 55.00: parts are typed in the line's own basis, so the parts are 5,000 and 5,000 cents ex GST and they add to the line's cents exactly (put any odd cent on the rest).
- `TestSplitOfAnUncountedLineIsRefused`; `TestSplitOfASplitPartIsRefused`.
- `TestUnsplitRestoresTheLine`: `unsplit_line` on the original deletes the parts and counts it again.

Set:
- `TestSetAmountRecomputesCents` (with `gst: "inc"` the cents are one eleventh less); `TestSetAmountClearsAmountNotFound`; `TestSetLabel`; `TestSetKindToOptionUncountsAndPlacesOutside`; `TestSetStateIncludedClearsTheAmount`; `TestSetWithNothingToChangeIsRefused`.

Add and delete:
- `TestAddLineIsCountedAndTheUsers`; `TestAddLineNeedsALabelAndAnAmount`; `TestDeleteOnlyAUsersOwnLine` (a model line is refused: "Take it out of the count instead.").

Count:
- `TestUncountAndCountAgain`; `TestCountNeedsAnAmount`; `TestCountRefusesASubtotalOrARate` ("Only a priced line of scope can be counted.").

Tender sum:
- `TestSetTenderSumAddsTheUsersTotalLine`: a line of kind `total`, `DecidedBy` `user`, not counted, with the amount. A second `set_tender_sum` replaces it. `UserTenderSum(lines)` returns it.

Errors:
- `TestErrorsNameTheField`: an `*OpError` carries `Field` and a plain message.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// Op is one named edit to a submission's lines (contract C6).
type Op struct {
	Op         string `json:"op"`
	Line       string `json:"line,omitempty"`
	Submission string `json:"submission,omitempty"`
	Row        *string `json:"row,omitempty"`
	Label      *string `json:"label,omitempty"`
	Amount     *string `json:"amount,omitempty"` // decimal, in the GST basis given or the line's own
	GST        *string `json:"gst,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	Placed     *string `json:"placed,omitempty"`
	State      *string `json:"state,omitempty"`
	Counted    *bool   `json:"counted,omitempty"`
	Page       int     `json:"page,omitempty"`
	Parts      []Part  `json:"parts,omitempty"`
	RestRow    *string `json:"rest_row,omitempty"`
}

// Part is one named share of a split. The rest goes to RestRow.
type Part struct {
	Row    string `json:"row"`
	Amount string `json:"amount"`
}

// OpError says which field of an operation is wrong, in plain words.
type OpError struct{ Field, Msg string }

func (e *OpError) Error() string { return e.Msg }

// Apply applies one operation to one submission's lines and returns the
// lines as they then stand. rows holds the keys a line may sit on. It
// changes nothing when it returns an error.
func Apply(lines []Line, rows map[string]bool, op Op, newID func() string) ([]Line, error)

// UserTenderSum is the tender sum the manager set, if any.
func UserTenderSum(lines []Line) *int64
```

Rules the code must hold:

- Amounts are parsed with `invoices.ParseCents`. An amount typed with `gst: "inc"` is stored ex GST through `exGST`, and `Printed` keeps what was typed.
- Every line an operation touches gets `DecidedBy = "user"`.
- A split's named parts are parsed in the line's own basis, converted to cents one by one, and the rest is the line's cents less their sum, so the parts always add to the line exactly. The rest must be greater than zero.
- Only lines of kind `base`, `ps` or `pc` with an amount can be counted.
- `set_tender_sum` never changes a line of scope. The user's total line has an empty row and heading and the label "Tender sum (set by you)".
- `Apply` copies the slice it changes; the caller's lines are untouched on error.

Export `ExGST` if slice 2 has not already.

**Step 4:** Run the tests. Expected: PASS. **Step 5:** Commit: `Tender: line edits as pure operations`.

### Task 2: The change log learns about tenders

**Files:**
- Create: `internal/db/migrations/<next>_tender_changes.sql`
- Test: add to `internal/db/migrate_tender_test.go`

**Step 1: Write the failing test.** A write to `tender_lines` inside a transaction marked with a change of area `procurement` leaves one `project_change_rows` row for it; the same write unmarked leaves none.

**Step 2: Write the migration.** Following the change-log migration exactly (*verify* its file: the `capture_change()` trigger and how it is attached to a table):

- widen the `area` check on `project_changes` to include `'procurement'`;
- attach the capture trigger to `tender_lines`, naming `updated_at` as a column that may change without counting.

Do not capture `submission_prices` or `submissions`: both are rewritten from the lines by the finisher in task 3.

**Step 3:** Run. Expected: PASS. **Step 4:** Commit: `Tender: line edits are captured for undo`.

### Task 3: Applying a batch

**Files:**
- Modify: `internal/store/tender_ops.go`
- Create: `internal/store/tender_refresh.go`
- Modify: the undo finisher switch in `internal/store/change_undo.go` (*verify*)
- Test: `internal/store/tender_ops_test.go`

**Step 1: Write the failing tests.**

- `TestMoveKeepsTheColumnTotal`: move a line between rows; the cells still add to the tender sum; the two cells changed by the line's amount.
- `TestMoveToSystemRowGrowsNotBrokenDown`.
- `TestCorrectAmountChangesNotBrokenDownNotTheSum`: `set_line` with a new amount: the tender sum is as it was and the system row took the difference.
- `TestSetTenderSumChangesTheSum`: the sum, `tender_sum_by = 'user'` and the how "set by you"; the system row took the difference.
- `TestBatchIsAllOrNothing`: a good move followed by a bad one changes nothing, and the error names "operation 2".
- `TestOpsAcrossTwoFirmsInOneBatch`.
- `TestEditDoesNotUncheck`: an edit to a checked column leaves it checked.
- `TestEditReturnsAChangeID` and `TestUndoPutsItBack`: apply a move, call `UndoChange` with the returned id: the line is on its old row, the cells are as before, the version has moved on.
- `TestUndoAfterALaterEditOfTheSameLineIsRefused`: `ErrChangedSince`, as the change log does for the cost plan.
- `TestCheckColumnIsNotInTheChangeLog`: a batch holding only `check_column` returns no change id.
- `TestALineOfAnotherPackageIsRefused`; `TestAnotherOrgIsNotFound`.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.** Make `TenderOp` an alias of, or embed, `tender.Op`, keeping `Checked`.

```go
// ApplyTenderOps applies a batch, or none of it, and returns the id of the
// change that can be undone ("" when the batch held nothing undoable).
func (s *Store) ApplyTenderOps(ctx context.Context, org, project, pkg, actor string, version int64, ops []TenderOp) (changeID string, err error)
```

1. When any operation other than `check_column` is in the batch, make a change id and wrap the context: `WithChange(ctx, Change{ID, OrgID, ProjectID, Actor, Via: "screen", Area: "procurement"})`.
2. Begin, `lockProject`, check the version.
3. Find each operation's submission (from its line, or its `submission` field); refuse one that is not a tender-read submission of this package. Load each affected submission's lines once, `FOR UPDATE`.
4. Apply the operations in order with `tender.Apply`. Wrap an `*OpError` as `ErrInvalidInput` with "operation N: " before its message.
5. Write the difference for each affected submission: insert new lines, update changed ones, delete removed ones.
6. `refreshTender` for each affected submission.
7. Bump the version, raise the event, commit.

```go
// refreshTender brings everything derived from a submission's lines back
// in step with them: the tender sum, the flags and the price cells. It
// runs after every edit and after every undo.
func refreshTender(ctx context.Context, tx pgx.Tx, org, project, pkg, sub string) error
```

It loads the lines and the stored reading fields, takes the tender sum from `tender.UserTenderSum(lines)` when there is one (setting `tender_sum_by = 'user'` and how "set by you") and otherwise restores the one the pass chose (keep the pass's own sum and how in two new columns, `tender_sum_read_cents` and `tender_sum_read_how`, written by `CommitTenderRead`: add them in this slice's migration), calls `tender.Recount`, updates the submission, and calls `projectTenderCells`.

Register `refreshTender` as the finisher for area `procurement` in `UndoChange`: after an undo, run it for every submission that owns a restored line, and bump the package's version.

**Step 4:** Run the tests and the store package. Expected: PASS.

**Step 5:** The route. `POST .../tender/ops` now answers with the table plus `"change": "<id>"` when there is one. If the change-log plan's coverage test (`TestWriteRoutesHaveOperations`) exists, class the two tender write routes the way it asks. HTTP tests: a move returns 200 with a change id; undo through the change log's route returns 200 and the table read shows the line back.

**Step 6:** Commit: `Tender: apply and undo line edits`.

### Task 4: Editing on screen

**Files:**
- Modify: `web/src/tender/tenderApi.ts` (the operations of this slice in `TenderOp`; `undo`)
- Modify: `web/src/tender/AsTendered.tsx`, `web/src/tender/LinesPanel.tsx`
- Create: `web/src/tender/LineMenu.tsx`, `web/src/tender/SplitDialog.tsx`, `web/src/tender/LineEditor.tsx`
- Test: `web/tests/tender-edit.spec.ts`

Follow the document register and the cost plan for selection, the right-click menu and inline editing (`web/src/Register.tsx`, `web/src/CostGrid.tsx`, `web/src/states.css`): click selects, Ctrl or Cmd toggles, Shift extends, F2 or double-click edits, Escape cancels.

**What the manager can do.**

- **Move by drag.** A line in the lines panel, and a whole priced cell, can be dragged onto any cell in the same firm's column, including "Not broken down". Dragging a cell moves every counted line in it, as one batch. A column that is not the line's own does not accept the drop.
- **Move by menu or key.** Right-click a line or a cell: "Move to" lists the rows. With a line or cell selected, pressing M opens the same list; arrow keys and Enter choose.
- **Split.** "Split…" on a line opens a small dialog: the line's label and printed amount; one or more "amount, row" entries; and "the rest goes to" a row. The browser does not add: it sends `split_line` with the named parts and `rest_row`, and shows the server's message when the parts are more than the line. "Put back together" on the original calls `unsplit_line`.
- **Correct.** F2 or double-click on a line edits its label and amount in place, with a GST choice (ex or inc) beside the amount. A line marked "Amount to type" opens straight into the amount.
- **Count.** "Take out of the count" and "Count" on a line. An uncounted line shows greyed with "not counted".
- **Add.** A ghost row at the foot of the lines panel, as in the cost plan: label, amount, GST; Enter adds it to that row.
- **Delete.** Only on a line the user added.
- **Tender sum.** Double-click the tender sum cell: amount and GST. It then shows "set by you".
- **Undo.** An "Undo" button in the page head, enabled when this session has made a change. It keeps the change ids in order and undoes the latest. When the server refuses ("Can't undo: changed since"), show its words and drop that id.
- **A stale table.** On 409, reload the table and show "This table changed. It has been reloaded."

Every action goes through one function that calls `tenderApi.ops` with the table's `version` and replaces the table with the answer.

**The browser test.** With two firms read by the stand-in (set up as in `tender-table.spec.ts`):

1. Right-click a line, Move to another row. Both cells change; the tender sum does not.
2. Drag a line to "Not broken down". That cell grows by the line's amount.
3. Split a line in two rows. Both parts show; the column total is unchanged.
4. Correct a line's amount. "Not broken down" takes the difference; the tender sum is unchanged.
5. Set the tender sum. It shows "set by you".
6. Add a line, then delete it.
7. Undo three times. After each, the table is as it was one step before.
8. Select a line and press M, choose a row with the keyboard.
9. In a second browser context make an edit; in the first make another: it reloads and says the table changed.

Run `npx playwright test tests/tender-edit.spec.ts`. Expected: PASS. Run `npm run build`.

Commit: `Tender: move, split, correct and undo on screen`.

### Task 5: The edit budget

**Files:**
- Modify: `cmd/intake-bench/tender.go`, `bench/budgets.json`, `internal/httpapi/projects.go` (the tender routes' budget path)

Add the row and give the two tender write routes this path:

```json
	{"name":"tender_edit","p50_us":100000,"p90_us":250000,"note":"one move, split or correction on a package of five firms with 330 lines each, recomputed table returned; no model"},
```

In the bench, on the five-firm, 330-line package of slice 2, time fifty `move_line` batches and ten undos. Expected: within budget. If it is not, the likely cost is re-reading every line of the package to build the table: recompute only the edited submission and reuse the others, and report it.

Commit: `Tender: the edit budget`.

### Finish

Write `docs/evidence/2026-11-14-tender-comparison-slice4.md`: the bench figures, what the change log captures and what the finisher rebuilds, and what an undo cannot put back (a checked state; a re-read).
