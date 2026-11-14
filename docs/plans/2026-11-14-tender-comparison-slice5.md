# Tender Comparison Slice 5: Adds and Subs

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Under the first table sits a second, "Adds and subs": the system suggests additions and subtractions firm by firm, the manager applies, edits and adds their own, and the tender sum plus the applied ones is the like-for-like total.

**Architecture:** `tender.Suggest` is a pure function from the firms' lines, cells and marks to suggestions with stable keys. The store keeps suggestions in step after every read, edit and undo without touching anything the manager applied or wrote. Nothing is applied by the system.

**Tech Stack:** Go, PostgreSQL, React, TypeScript, Playwright.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md` (contracts C3, C5, C6). **Design:** section 6.

**Status:** written 14 November 2026. Not compiled or run. Names marked *verify* were not opened.

---

### Task 1: The migration

**Files:**
- Create: `internal/db/migrations/<next>_tender_adjustments.sql`
- Test: add to `internal/db/migrate_tender_test.go`

**Step 1: Write the failing test.** An adjustment with origin `system` and no suggestion key is refused; so is origin `user` with one. An amount in state `applied` with a null amount is refused. Two system adjustments with the same key in one package are refused. Deleting the package deletes both tables' rows. A marked write to either table is captured by the change log.

**Step 2: Write the migration.**

```sql
-- One addition or subtraction for a package: a row of the second table.
CREATE TABLE tender_adjustments (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 300),
 note text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
 origin text NOT NULL CHECK (origin IN ('system','user')),
 reason text NOT NULL CHECK (reason IN ('gap','allowance','option','saving','taken_out','other')),
 row_key text CHECK (row_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 -- Stable for a system row: line:<line id>, gap:<row key> or option:<item key>.
 suggestion_key text CHECK (length(suggestion_key) BETWEEN 1 AND 300),
 -- A system row the manager removed. It is kept so it does not come back.
 dismissed_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 CHECK ((origin = 'system') = (suggestion_key IS NOT NULL))
);
CREATE UNIQUE INDEX tender_adjustments_key_uq ON tender_adjustments (org_id, package_id, suggestion_key) WHERE suggestion_key IS NOT NULL;
CREATE INDEX tender_adjustments_package_idx ON tender_adjustments (org_id, package_id, ordinal);

-- One firm's signed amount on one adjustment. Only applied amounts count.
-- A suggested gap has no amount until the manager chooses one.
CREATE TABLE tender_adjustment_amounts (
 org_id uuid NOT NULL,
 adjustment_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 project_id uuid NOT NULL,
 amount_cents bigint,
 state text NOT NULL CHECK (state IN ('suggested','applied')),
 -- firm: the firm's own printed figure. none: a gap with no figure yet.
 source text NOT NULL CHECK (source IN ('firm','others_highest','others_average','typed','none')),
 line_id uuid,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, adjustment_id, submission_id),
 FOREIGN KEY (org_id, adjustment_id) REFERENCES tender_adjustments(org_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (state = 'suggested' OR amount_cents IS NOT NULL),
 CHECK ((source = 'none') = (amount_cents IS NULL))
);
```

Attach the change log's capture trigger to both, as slice 4 did for `tender_lines`. `line_id` has no foreign key on purpose: a re-read replaces the model's lines, and an amount the manager applied must outlive the line it came from.

**Step 3:** Run. Expected: PASS. **Step 4:** Commit: `Tender: tables for adds and subs`.

### Task 2: Suggestions, as a pure function

**Files:**
- Create: `internal/tender/suggest.go`
- Test: `internal/tender/suggest_test.go`

**Step 1: Write the failing tests** with invented lines.

- `TestOptionLineIsSuggestedAtTheFirmsFigure`: an option line of 18,000.00 for firm A gives one suggestion keyed `line:<id>`, label "Option: <label>", reason `option`, with A's amount 1,800,000 cents, source `firm`, and the line's id and page. Firm B has no amount on it.
- `TestOptionOnAScheduleOptionRowIsShared`: two firms each with an option line mapped to the schedule's option row `option:x` give one suggestion keyed `option:option:x` labelled with the row's label, holding both firms' amounts.
- `TestSavingIsNegative`: a saving line of 6,500.00 gives an amount of −650,000.
- `TestAllowanceOutsideThePrice`: a provisional sum placed outside and not counted gives "Extra to the price: <label>", reason `allowance`. The same line once counted (it carries `counted_to_match`) gives nothing.
- `TestGapSuggestsARowWithNoFigure`: firm B is marked `gap` on a row firms A and C priced. One suggestion keyed `gap:<row>`, label "Allowance for <row label>", reason `gap`, with an entry for B only: amount nil, source `none`.
- `TestGapSharesARow`: two firms with a gap on the same row share the suggestion.
- `TestRatesSubtotalsAndCountedLinesSuggestNothing`.
- `TestOrderIsGapsThenAllowancesThenOptionsThenSavings`, and within each by page then ordinal.
- `TestOthersRange`: `Others(cols, "B", row)` over priced cells of 100, 300 and 500 from three other firms gives low 100, high 500, average 300, count 3; with an odd cent the average rounds half up.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// SuggestColumn is one tender-read firm as Suggest needs it.
type SuggestColumn struct {
	ID    string // submission id
	Lines []Line
	Cells map[string]Cell
	Marks map[string]string // by row key
}

// Amount is one firm's figure on a suggestion.
type Amount struct {
	Cents  *int64 // signed; nil for a gap with no figure yet
	Source string // "firm" or "none"
	LineID string
	Page   int
}

// Suggestion is one row the system proposes for the second table. Nothing
// here is applied: the manager applies.
type Suggestion struct {
	Key     string
	Label   string
	Reason  string // gap | allowance | option | saving
	Row     string
	Amounts map[string]Amount // by submission id
}

// Suggest lists what a manager would want to consider when levelling.
// optionRows maps a schedule option row's key to its label.
func Suggest(rows []Row, optionRows map[string]string, cols []SuggestColumn) []Suggestion

// Range is what the other firms priced a row at.
type Range struct {
	Low, High, Average int64
	Count              int
}

func Others(cols []SuggestColumn, id, row string) Range
```

The system never invents a figure: every suggested amount is a line's own cents, or absent.

**Step 4:** Run. Expected: PASS. **Step 5:** Commit: `Tender: suggest adds and subs from the lines`.

### Task 3: Keeping suggestions in step

**Files:**
- Create: `internal/store/tender_adjust.go`
- Modify: `internal/store/tender_refresh.go` (call it), `internal/store/tender_commit.go` (call it)
- Test: `internal/store/tender_adjust_test.go`

**Step 1: Write the failing tests.**

- `TestReadCreatesSuggestions`: after a commit of a reading with an option line, the package has one system adjustment with a suggested amount.
- `TestSuggestionFollowsTheLine`: correct the option line's amount; the suggested amount follows.
- `TestAppliedAmountDoesNotMove`: apply it, then correct the line: the applied amount is unchanged.
- `TestSuggestionGoesWhenItsLineGoes`: take the option line into the count; the suggestion's amount is gone, and the row too once it has no amounts.
- `TestAppliedAmountSurvivesAReRead`: apply, then commit a new reading: the applied amount is still there though its line id is not.
- `TestDismissedSuggestionStaysAway`: dismiss a system row, commit again: it is not suggested again.
- `TestUsersRowsAreLeftAlone`: a user's row and its amounts are the same after a sync.
- `TestGapAppearsWhenAnotherFirmArrives`: one firm alone has no gap suggestion; when a second firm that priced the row is read, the first gets one.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// syncSuggestions brings the package's system adjustments in step with
// tender.Suggest. It never changes an applied amount, a user's row or a
// dismissed row.
func syncSuggestions(ctx context.Context, tx pgx.Tx, org, project, pkg string) error
```

1. Build a `SuggestColumn` for every tender-read live submission of the package (lines, cells, marks, as the table read does: share that code).
2. For each suggestion: find the system row by key. If it is dismissed, skip it. Otherwise insert it (ordinal after the last) or update its label and row.
3. For each of its amounts: no stored amount: insert it `suggested`. Stored and `suggested`: update the cents, source, line id. Stored and `applied`: leave it.
4. Delete `suggested` amounts on system rows that `Suggest` no longer gives.
5. Delete system rows that are not dismissed and have no amounts left.

Call it at the end of `CommitTenderRead` and of `refreshTender` (once per package, not once per submission).

**Step 4:** Run. Expected: PASS. **Step 5:** Commit: `Tender: suggestions stay in step with the lines`.

### Task 4: The operations

**Files:**
- Modify: `internal/store/tender_ops.go`
- Test: `internal/store/tender_ops_test.go`

**Step 1: Write the failing tests.**

- `TestApplyFirmsOwnFigure`: `apply_adjustment` on a suggested amount with source `firm` makes it `applied`; the like-for-like total moves by it. `applied: false` puts it back to `suggested`.
- `TestApplyGapWithTheHighest`, `TestApplyGapWithTheAverage`: the amount stored is the highest, or the average, of the other firms' priced cells on that row at that moment, with the source recorded. `TestApplyGapTyped`: the typed amount, source `typed`. `TestApplyGapWithNoSourceIsRefused`: "Choose the highest, the average, or type a figure."
- `TestApplyGapWhenNoOtherFirmPricedIt`: refused: "No other firm priced this row."
- `TestAddUsersRowAndTypeAmounts`: `add_adjustment` then `set_adjustment_amount` for two firms, one positive and one negative: both `applied`, source `typed`. A null amount removes the firm's amount.
- `TestRenameAndNote`: `set_adjustment` on a user's row. On a system row only the note can change.
- `TestRemoveUsersRowDeletesIt`; `TestRemoveSystemRowDismissesIt`.
- `TestTakeOutLine`: `take_out_line` on a counted line of 34,000.00 adds a user's row "Less: <label>", reason `taken_out`, with −3,400,000 applied for that firm and the line's id. The first table is unchanged. Taking out an uncounted line is refused.
- `TestApplySuggested`: `apply_suggested` for one firm applies every suggested amount of that firm whose source is `firm`, and leaves gaps alone.
- `TestAdjustmentOpsAreUndoable`: apply then `UndoChange`: suggested again.
- `TestAmountForASubmissionOfAnotherPackageIsRefused`.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement** the seven operations of contract C6 in `ApplyTenderOps`. They change only the two adjustment tables. Add to `TenderOp`: `Adjustment string`, `Applied *bool`, `Source string`, `Note *string`. After them, `syncSuggestions` runs with the rest of the refresh.

**Step 4:** Run. Expected: PASS. **Step 5:** Commit: `Tender: apply, type, add and remove adds and subs`.

### Task 5: The table read

**Files:**
- Modify: `internal/store/tender_table.go`, `internal/httpapi/submissions.go` (`tableBody`)
- Test: `internal/store/tender_table_test.go`

**Step 1: Write the failing tests.**

- `TestTableListsAdjustments`: rows in order, not the dismissed ones, each with its amounts by submission, and for a gap row `others` (low, high, average, count) for each firm that lacks an amount.
- `TestLikeForLikeIsSumPlusApplied`: tender sum 737,000.00, one applied add of 15,000.00 and one suggested of 18,000.00 gives 752,000.00.
- `TestLowestIsByLikeForLike`: the firm with the lower tender sum but the higher like-for-like total is not the lowest.
- `TestSuggestedCount`.

**Step 2:** Implement: fill `Adjustments`, `Suggested` and each column's `LikeForLike`; mark the lowest by like-for-like among tender-read columns with no `gap` mark that is still unanswered (a gap with an applied amount no longer bars the firm). **Step 3:** PASS. **Step 4:** Commit: `Tender: like-for-like totals in the table read`.

### Task 6: On screen

**Files:**
- Create: `web/src/tender/AddsSubs.tsx`
- Modify: `web/src/tender/tenderApi.ts`, `web/src/tender/AsTendered.tsx`, `web/src/Prices.tsx`
- Modify: `internal/tender/sample/model.go` (so browser tests can have options and exclusions)
- Test: `web/tests/tender-adds-subs.spec.ts`

**The stand-in learns four things** (with tests): a label starting "Option:" is an option placed outside; "Saving:" a saving; "Extra:" a provisional sum placed outside; a label ending "(excluded)" is an excluded line with no amount.

**What the second table shows.** A `<table>` with the caption "Adds and subs", directly under the first, with the same columns so the firms line up.

- One row per adjustment: the label, then each firm's signed amount ("+$15,000.00", "−$34,000.00") or nothing.
- A **suggested** amount is greyed with the word "suggested" and an "Apply" button. Its source shows under it: "the firm's figure, p. 34". For a gap the cell says "Not priced. Others $24,000.00 to $126,000.00" and the button opens three choices: "Use the highest", "Use the average", "Type a figure".
- An **applied** amount is solid, with its source under it ("highest of the others", "typed by you") and an "Undo apply" in its menu.
- A user's row: the label and each amount edit in place (F2 or double-click). A system row's label does not edit.
- Each row's menu: "Note…", "Remove".
- A ghost row at the foot: "New add or sub".
- Then the row "Like for like" in bold, with "Lowest" on the lowest.
- Above the table: "3 suggestions not applied" and, per firm in its head menu, "Apply this firm's own figures".
- Dragging a line from the first table onto the second table makes "Less: <label>" for that firm (`take_out_line`).

The browser never adds: every figure, range and total comes from the server.

**The browser test.** Three firms read by the stand-in: A prices every row and offers "Option: epoxy grout"; B marks one row "(excluded)"; C has "Extra: asbestos removal".

1. The second table lists a gap row for B, the extra for C and the option for A, all suggested. "3 suggestions not applied". Like for like equals each tender sum.
2. Apply C's extra: C's like-for-like rises by it. Reload: still applied.
3. On B's gap choose "Use the highest": the amount is the higher of A's and C's cells on that row, and says "highest of the others".
4. Add a row "Site shed", type −5,000 for A: A's like-for-like falls by 5,000.
5. Drag one of A's lines down: "Less: …" appears, applied, and A's first table is unchanged.
6. Remove A's option row: it is gone, and still gone after "Read again" on A.
7. Undo: the row is back.
8. The lowest mark sits on the lowest like-for-like total.

Run `npx playwright test tests/tender-adds-subs.spec.ts` and `npm run build`. Expected: PASS.

Commit: `Tender: the adds and subs table`.

### Finish

Re-run the `tender_edit` and `price_table_tender` benches with twenty adjustments on the package; both must stay in budget. Write `docs/evidence/2026-11-14-tender-comparison-slice5.md`.
