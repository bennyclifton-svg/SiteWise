# Tender Comparison Slice 2: Switch, Storage, the Background Job and the Table Read

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A tender dropped on a package is read in the background, its lines are stored, its price cells are written from them, and the table read returns the first table's data. No screen changes yet.

**Architecture:** A `tenderReader` beside the existing `proposalReader` runs the pass after the first read commits. `CommitTenderRead` stores the lines and projects them into `submission_prices`, so award, the report and the cost plan read what they read today. One org setting gates everything that leaves the server.

**Tech Stack:** Go, PostgreSQL 17, pgx v5.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md` (contracts C1 to C7). **Design:** sections 5, 7, 11, 13, 17 and 19.

**Status:** written 14 November 2026 from reading `main`. Not compiled or run. Names marked *verify* were not opened.

---

### Task 0: Check the gate

`docs/evidence/2026-11-14-tender-comparison-gate.md` must exist and say go. If it does not, stop.

### Task 1: The migration

**Files:**
- Create: `internal/db/migrations/<next>_tender_comparison.sql`
- Test: `internal/db/migrate_tender_test.go` (follow `internal/db/migrate_proposals_test.go`)

**Step 1: Write the failing test.** `TestTenderMigration`: after `Migrate`, assert that:

- `orgs` has `model_enabled_at`; `packages` has `tender_version`; `submissions` has `tender_state` defaulting to `''`;
- a `tender_lines` row with `state = 'included'` and a non-null `amount_cents` is refused;
- a `tender_lines` row with `placed = 'elsewhere'` is refused;
- deleting the submission deletes its lines.

**Step 2:** Run it. Expected: FAIL, relation `tender_lines` does not exist.

**Step 3: Write the migration.**

```sql
-- Tender comparison (docs/design/2026-11-14-tender-comparison.md).

-- One switch per org covers every model use: the org agrees that its
-- project text may be sent to the model provider. Null is off.
ALTER TABLE orgs
 ADD COLUMN model_enabled_at timestamptz,
 ADD COLUMN model_enabled_by uuid;

-- Bumped by every write to a package's tender tables; an edit that names a
-- stale version is refused.
ALTER TABLE packages ADD COLUMN tender_version bigint NOT NULL DEFAULT 1 CHECK (tender_version > 0);

-- The tender pass on one submission. '' is never asked, or the pass is off.
ALTER TABLE submissions
 ADD COLUMN tender_state text NOT NULL DEFAULT '' CHECK (tender_state IN ('','pending','reading','read','failed')),
 ADD COLUMN tender_attempts integer NOT NULL DEFAULT 0 CHECK (tender_attempts >= 0),
 ADD COLUMN tender_retry_after timestamptz,
 ADD COLUMN tender_error text NOT NULL DEFAULT '' CHECK (length(tender_error) <= 200),
 ADD COLUMN tender_prompt text NOT NULL DEFAULT '' CHECK (length(tender_prompt) <= 40),
 ADD COLUMN tender_basis text NOT NULL DEFAULT '' CHECK (tender_basis IN ('','fixed','estimate','cost_plus','rates_only','unknown')),
 -- The firm's printed price for its base offer, ex GST, and how it was chosen.
 ADD COLUMN tender_sum_cents bigint,
 ADD COLUMN tender_sum_how text NOT NULL DEFAULT '' CHECK (length(tender_sum_how) <= 200),
 ADD COLUMN tender_sum_by text NOT NULL DEFAULT '' CHECK (tender_sum_by IN ('','rule','user')),
 ADD COLUMN tender_flags jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_flags) = 'array'),
 ADD COLUMN tender_notes jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_notes) = 'array'),
 ADD COLUMN tender_exclusions jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_exclusions) = 'array'),
 ADD COLUMN tender_read_at timestamptz,
 -- The manager's acceptance of the reading (AGENTS.md rule 1).
 ADD COLUMN tender_checked_at timestamptz,
 ADD COLUMN tender_checked_by uuid,
 ADD CONSTRAINT submissions_tender_checked CHECK ((tender_checked_at IS NULL) = (tender_checked_by IS NULL));
CREATE INDEX submissions_tender_due_idx ON submissions (tender_retry_after) WHERE tender_state IN ('pending','failed');

-- One line a firm printed. amount_cents is ex GST; printed is the firm's
-- own figure as text. A line with no amount is priced-but-unknown,
-- included, excluded or not priced.
CREATE TABLE tender_lines (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 page integer NOT NULL DEFAULT 0 CHECK (page >= 0),
 label text NOT NULL CHECK (length(label) <= 500),
 printed text NOT NULL DEFAULT '' CHECK (length(printed) <= 80),
 amount_cents bigint,
 gst text NOT NULL DEFAULT 'unknown' CHECK (gst IN ('ex','inc','unknown')),
 state text NOT NULL CHECK (state IN ('priced','included','excluded','not_priced')),
 kind text NOT NULL CHECK (kind IN ('base','ps','pc','option','saving','rate','subtotal','total','gst')),
 -- sum: its own part of the price. inside: already in another line.
 -- outside: extra to the price. ("where" is a reserved word.)
 placed text NOT NULL CHECK (placed IN ('sum','inside','outside')),
 heading text NOT NULL DEFAULT '' CHECK (length(heading) <= 300),
 row_key text CHECK (row_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 note text NOT NULL DEFAULT '' CHECK (length(note) <= 300),
 counted boolean NOT NULL DEFAULT false,
 flags jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(flags) = 'array'),
 decided_by text NOT NULL CHECK (decided_by IN ('model','rule','user')),
 split_of uuid,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (state = 'priced' OR amount_cents IS NULL),
 CHECK (NOT counted OR amount_cents IS NOT NULL)
);
CREATE INDEX tender_lines_submission_idx ON tender_lines (org_id, submission_id, ordinal);
```

**Step 4:** Run the test, then `go test ./internal/db -count=1`. Expected: PASS. If a migration-rehearsal or foreign-key sweep test lists expected tables (*verify* `migration_rehearsal_test.go`, `migrate_fk_sweep_test.go`), add `tender_lines` where it asks.

**Step 5:** Commit: `Tender: tables for lines, the pass state and the org switch`.

### Task 2: The org switch

**Files:**
- Create: `internal/store/model_switch.go`
- Test: `internal/store/model_switch_test.go`
- Modify: `internal/httpapi/billing.go` (both route maps, `getAccount`)
- Test: `internal/httpapi/` beside the existing account tests (*verify* the file name)
- Modify: `web/src/Account.tsx`, and the account client it uses (*verify*)
- Test: `web/tests/account-model.spec.ts`

**Step 1: Write the failing store tests.**

- `TestModelSwitchIsOffByDefault`: `ModelEnabled(ctx, orgA)` is false.
- `TestModelSwitchOnAndOff`: `SetModelEnabled(ctx, orgA, userA, true, now)` then `ModelEnabled` is true and the row holds `now` and `userA`; off again clears both.
- `TestModelSwitchIsPerOrg`: switching org A on leaves org B off.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// ModelEnabled reports whether the org has agreed that its project text
// may be sent to the model provider.
func (s *Store) ModelEnabled(ctx context.Context, org string) (bool, error)

// SetModelEnabled records the owner's choice. The caller has checked that
// actor is the org's owner.
func (s *Store) SetModelEnabled(ctx context.Context, org, actor string, on bool, now time.Time) error
```

**Step 4: The route.** `PUT /account/model` with `{"enabled": true}`. Use `ownerSession` as `postCheckout` does. A member who is not the owner gets 403. Add it to both maps in `billing.go` with the budget path the other account writes use. `GET /account` gains:

```json
"model": {"available": true, "enabled": false, "enabled_at": null}
```

`available` is whether the server has a key (`deps` needs to know: add `ModelAvailable bool` to `Deps`, set from `opts.TenderLLM != nil` in task 7).

Tests: owner switches on and `GET /account` shows it; a non-owner gets 403; no session gets 401; with `ModelAvailable` false the PUT returns 409 with "This server is not set up to use a model."

**Step 5: The screen.** In `Account.tsx`, a section headed "Reading tenders with a model" shown to the owner when `model.available`:

> SiteWise can send the pages of a tender you file to Claude, a model made by Anthropic, so it can list the prices for you to check. Nothing is sent while this is off, and tenders are read as they are now.

with one switch, "Send tender pages to the model", and under it, when on, "On since <date>". Use the switch control the Procurement menus use (*verify* in `web/src/Procurement.tsx`).

Browser test: sign in as the owner, open Account, turn the switch on, reload, it is still on.

**Step 6:** Run the Go tests, `npm run build`, and the browser test. Expected: PASS.

**Step 7:** Commit: `Tender: one switch per org for sending text to the model`.

### Task 3: The system row

**Files:**
- Create: `internal/store/tender_rows.go`
- Modify: the function that syncs `schedule_items` with the RFP (`ensureSchedule`; *verify* its file, and the group and builder variants `ensureGroupSchedule`, `ensureBuilderSchedule`)
- Modify: `internal/store/comparison.go` (`packageColumns`)
- Test: `internal/store/tender_rows_test.go`

**Step 1: Write the failing tests.**

- `TestSystemRowIsAddedOnce`: `ensureTenderRow` twice leaves one `tender:not_broken_down` item, base, kind `stage`, label "Not broken down", ordered after every other base item.
- `TestScheduleSyncLeavesTheSystemRowAlone`: add the system row, change the package's RFP so the schedule re-syncs (follow an existing test that does this), and the system row is still live with the same label.
- `TestOldColumnsIgnoreTheSystemRow`: a package with one old-style submission whose base cells are all priced has a `BaseTotal`. Add the system row. `BaseTotal` is unchanged and `Unread` is still 0.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// ensureTenderRow adds the "Not broken down" row to a package's schedule.
// It is a schedule item so that award, the report and export treat it as
// any other row.
func ensureTenderRow(ctx context.Context, tx pgx.Tx, org, project, pkg string) error
```

Insert with `ON CONFLICT (org_id, package_id, key) DO NOTHING`, `source = 'generic'`, `ordinal` one more than the package's highest, `detail` "What the firm did not break down, and anything its lines do not add up to".

In every schedule sync, skip keys with the prefix `tender:` when retiring or relabelling.

In `packageColumns`, the `items` passed to `proposals.Totals` for a submission that is not tender-read must leave out keys with the prefix `tender:`.

**Step 4:** Run the tests and `go test -p 1 ./internal/store -count=1`. Expected: PASS, and no existing procurement test changes.

**Step 5:** Commit: `Tender: the "Not broken down" row`.

### Task 4: What the pass needs, and claiming a read

**Files:**
- Create: `internal/store/tender_read.go`
- Test: `internal/store/tender_read_test.go`

**Step 1: Write the failing tests.**

- `TestTenderInputGivesRowsAndTheStatedTotal`: a submission on a town-planning package returns the package's priced rows (base rows and option rows, not the system row, not rates or criteria), the file's hash and size, and `StatedEx` converted from an inc GST stated total.
- `TestQueueNeedsTheSwitch`: with the org's switch off `QueueTenderRead` returns false and the state stays `''`. With it on, true and `pending`.
- `TestQueueNeedsAPackage`: a submission with no package is not queued.
- `TestQueueSkipsWhatIsNotAProposal`: status `not_proposal`, `superseded`, `withdrawn` and `awarded` are not queued.
- `TestClaimIsExclusive`: two `ClaimTenderRead` calls, one true and one false; the state is `reading` and attempts is 1.
- `TestWrongOrgSeesNothing`: every function called with org B for org A's submission returns `ErrNotFound` or false and changes nothing.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// TenderInput is what one tender pass needs from the database.
type TenderInput struct {
	Submission
	SHA256   []byte
	Size     int64
	Rows     []tender.Row // the package's priced rows, in schedule order
	StatedEx *int64       // the total the existing rules found, ex GST
}

func (s *Store) TenderInput(ctx context.Context, org, id string) (TenderInput, error)

// QueueTenderRead marks a submission for the pass when the org's switch is
// on, it has a package, and it is a live proposal that has not been read.
func (s *Store) QueueTenderRead(ctx context.Context, org, id string, again bool) (bool, error)

// ClaimTenderRead moves pending or due-failed to reading and counts the attempt.
func (s *Store) ClaimTenderRead(ctx context.Context, org, id string, now time.Time) (bool, error)

// FailTenderRead records why a pass failed and when to try again (nil: never).
func (s *Store) FailTenderRead(ctx context.Context, org, id, reason string, retry *time.Time) error

func (s *Store) PendingTenderReads(ctx context.Context) ([]SubmissionRef, error)
func (s *Store) DueTenderRetries(ctx context.Context, now time.Time) ([]SubmissionRef, error)
```

- Get the hash and size the way `SubmissionForReading` does (*verify*).
- `Rows`: run `ensureSchedule` (or the group variant) for the package, read the schedule, keep items that are not retired, whose `Price()` is true and whose key does not start `tender:`. Map to `tender.Row{Key, Label, Detail}`.
- `StatedEx`: `stated_total_cents` with `gst_basis`: `inc` goes through the same one-eleventh arithmetic as `tender.exGST` (export a `tender.ExGST` for this); `ex` and `unknown` as they are.
- `again` true is "Read again": it queues from `read` and `failed` as well as `''`.
- `reason` is plain words for the screen, never an error string from a library.

**Step 4:** Run the tests. Expected: PASS.

**Step 5:** Commit: `Tender: queue and claim a pass`.

### Task 5: Storing a reading and writing the cells

**Files:**
- Create: `internal/store/tender_commit.go`
- Modify: `internal/store/submissions.go` (`replaceMachinePrices`)
- Test: `internal/store/tender_commit_test.go`

**Step 1: Write the failing tests.** Build readings by calling `tender.Reconcile` on invented raw lines, as the reconciler's own tests do.

- `TestCommitStoresLinesAndState`: after `CommitTenderRead` the submission is `read`, has the tender sum, how, basis, flags, notes and prompt version, and `tender_lines` holds every line in order with its flags.
- `TestCommitWritesCellsThatAddUp`: the `submission_prices` rows on the package's base rows add up to the tender sum, to the cent, including the system row.
- `TestCommitWritesWhatTheFirmSaid`: a row the firm marked included has a cell in state `included` with the firm's note as its basis.
- `TestCommitWritesOptionCells`: a line placed outside and mapped to an option row gives that option row a priced cell, and it is not in the base total.
- `TestCommitLeavesRatesAlone`: a `rate:` cell written by the old reader is still there.
- `TestSecondCommitKeepsTheUsersLines`: insert a line with `decided_by = 'user'` between two commits. After the second commit it is still there and still counted, and the cells include it.
- `TestSecondCommitClearsChecked`: a checked column is unchecked after a second commit.
- `TestFirstCommitKeepsAUsersOldCell`: before the first commit the submission has a `submission_prices` row with `decided_by = 'user'` on a base row. After the commit there is a user line for it, counted, and model lines on that row are not counted and carry `treated_as_breakdown`.
- `TestOldReaderDoesNotOverwriteTenderCells`: after a commit, call `CommitSubmissionRead` again as a recheck does. The base cells are unchanged.
- `TestCommitBumpsTheVersion`: `packages.tender_version` goes up by one.
- `TestCommitOnAGoneSubmissionDoesNothing`: an unknown id returns nil and writes nothing.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// CommitTenderRead stores one reading: its lines, its tender sum and flags,
// and the price cells projected from the counted lines. Lines the user
// decided are kept. incomplete is true when the answer was cut off.
func (s *Store) CommitTenderRead(ctx context.Context, org, id string, r tender.Reading, prompt string, incomplete bool, now time.Time) error
```

In one transaction, in this order:

1. Read the submission's project and package; return nil if it is gone. `lockProject`. Select the submission `FOR UPDATE`.
2. `ensureTenderRow` for the package.
3. If the submission has no `tender_lines` yet, turn each `submission_prices` row that has `decided_by = 'user'`, a non-null amount and a key on a base row into a line: label the cell's label, the amount, `gst` `ex`, state `priced`, kind `base`, placed `sum`, the row, counted, `decided_by` `user`.
4. Delete the submission's lines whose `decided_by` is not `user`.
5. Insert the reading's lines with new ids, `decided_by` as the reading gives it. For each base row that now has a user line, set the model's lines on that row to not counted and add the flag `treated_as_breakdown` with the text "Your figure for this row stands."
6. If incomplete, add the column flag `reading_incomplete`: "The reading was cut off. Some lines may be missing."
7. Load every line of the submission back into a `tender.Reading` (tender sum from the reading unless `tender_sum_by = 'user'`, in which case keep the stored one) and call `tender.Recount`.
8. Update the submission: state `read`, the sum, how, `tender_sum_by` (`rule` unless it was `user`), basis, flags, notes, exclusions, prompt, `tender_read_at`, attempts unchanged, error cleared, checked cleared.
9. `projectTenderCells` (below).
10. `UPDATE packages SET tender_version = tender_version + 1`.
11. `procurementEvent` so the screen refreshes, as `CommitSubmissionRead` does.

```go
// projectTenderCells rewrites a tender-read submission's price cells from
// its lines (rule R9). It owns the cells on base rows, option rows and the
// system row, and no others.
func projectTenderCells(ctx context.Context, tx pgx.Tx, org, project, sub string, r tender.Reading, items []ScheduleItem) error
```

- Rows for `tender.Cells`: the package's live base rows other than the system row.
- Delete the submission's `submission_prices` whose key is a live price row (base, option or system row), then insert one per cell: `label` the schedule item's label; `amount_cents` and `state` from the cell; `basis` the cell's note, or "<n> lines" when priced; `page` the lowest page among its counted lines; `band` `amber` when `ToCheck`, else `green`; `decided_by` `user` when every counted line behind it is the user's, else `rule`.
- For each option row: the sum of lines placed outside whose row is that option's key, as a priced cell. No cell when there are none.
- Delete the submission's row in `comparison_alignment`: the old alignment does not apply to a tender-read submission.

In `replaceMachinePrices`, when the submission's `tender_state` is `read`, delete and insert only cells whose key starts `rate:`.

**Step 4:** Run the tests and the whole store package. Expected: PASS.

**Step 5:** Commit: `Tender: store a reading and write its cells`.

### Task 6: The table read

**Files:**
- Create: `internal/tender/compare.go`
- Test: `internal/tender/compare_test.go`
- Create: `internal/store/tender_table.go`
- Modify: `internal/store/comparison.go` (`PriceTable`, `priceTable`, `packageColumns`, `needsReview`)
- Modify: `internal/httpapi/submissions.go` (`tableBody`)
- Test: `internal/store/tender_table_test.go`, and an HTTP test beside the existing price-table tests

**Step 1: Write the failing tests for the marks**, pure and in `internal/tender`:

```go
// Column is one firm's cells for marking.
type Column struct {
	ID       string
	Sum      *int64          // tender sum
	OnRows   int64           // counted dollars on real rows
	Cells    map[string]Cell // by row key
}

// Marks returns, for each column id and row key, "high", "low" or "gap".
func Marks(rows []Row, cols []Column) map[string]map[string]string
```

- `TestHighAndLow`: three firms on one row at 100, 110 and 300: the third is `high`. At 100, 110 and 40: the third is `low`. With only two firms priced there is no mark.
- `TestExplicitGap`: a firm whose cell is `excluded` on a row the other two priced gets `gap`. A firm whose cell is `included` does not.
- `TestSilentGapNeedsABreakdown`: a firm with no cell on a row two others priced gets `gap` when at least 80% of its tender sum is on rows, and no mark when it is not (a lump sum).
- `TestGapNeedsHalfTheOthers`: with four other firms of which one priced the row, no `gap`.
- `TestTheSystemRowIsNeverMarked`.

Rules: high is more than double the median of the other priced firms; low is less than half; both need at least two others priced. Gap needs at least one other firm, and at least half of the others, to have priced the row.

**Step 2:** Implement `Marks`. Run. Expected: PASS.

**Step 3: Write the failing store tests.**

- `TestTableHasTenderBlock`: a package with two tender-read submissions returns `Tender` with the rows (base rows then the system row), two columns, each with state, sum, how, flags, cells and lines.
- `TestTableTotalsUseTheTenderSum`: a tender-read column's `Totals.BaseTotal` is its tender sum, `Unread` is 0 and `Gap` is nil, even when it has no cell on most rows.
- `TestTableMarksLowest`: the lower tender sum is `Lowest`. A column with a `gap` mark is never lowest.
- `TestTableOnIsFalseWithoutTheSwitch`: `Tender.On` false, and still the stored columns of submissions read earlier.
- `TestTableMixesOldAndNew`: one tender-read and one old-style submission: both appear in `Columns`; only the first has a tender column.
- `TestNeedsReviewUntilChecked`: a tender-read column needs review until `tender_checked_at` is set.

**Step 4: Implement.**

```go
// TenderTable is the tender member of the price table (contract C5),
// with cents where the JSON has decimal strings.
type TenderTable struct {
	On      bool
	Version int64
	Rows    []TenderRow
	Columns []TenderColumn
	// Adjustments and Suggested are filled from slice 5.
}
```

Give `TenderRow`, `TenderColumn`, `TenderCell` and `TenderLine` the fields C5 lists. Add `Tender *TenderTable` to `PriceTable` with `json:"-"`; `tableBody` in `internal/httpapi/submissions.go` writes it as C5 with `cents(...)`.

In `priceTable`, after the columns are loaded: read every line of the package's tender-read submissions in one query ordered by submission and ordinal; build each column's `tender.Reading` from its stored fields and lines; get its cells with `tender.Cells`; get marks with `tender.Marks`; set `like_for_like` to the tender sum for now (slice 5 adds the adjustments).

For a tender-read column replace what `proposals.Totals` gave with: `BaseTotal` and `KnownBase` the tender sum, `Unread` 0, `Gap` nil, `OptionsTotal` and `OthersTotal` 0, `NotPriced` the number of its `gap` marks. Then mark the lowest: among tender-read columns the lowest tender sum with no `gap` mark; old-style columns keep `MarkLowest` among themselves when the package has no tender-read column.

`needsReview` for a tender-read column: true while the pass is `pending`, `reading` or `failed`, or the column is not checked.

**Step 5:** Run all the tests. Expected: PASS. Add the wrong-org HTTP test: 404.

**Step 6:** Commit: `Tender: the table read returns the first table`.

### Task 7: The background reader

**Files:**
- Create: `internal/httpapi/tender_reader.go`
- Create: `internal/tender/sample/model.go` (a stand-in model for tests, the browser tests and the bench)
- Modify: `internal/httpapi/server.go` (`Options`, `Deps`, construction, the restart resume near line 339)
- Modify: `internal/httpapi/projects.go` (both route maps), `internal/httpapi/submissions.go` (`patchSubmission`: queue when a package is set)
- Modify: `cmd/sitewise/main.go` (build the client and the renderer; start the retry loop)
- Modify: `web/tests/e2eserver/main.go` (wire the stand-in)
- Test: `internal/httpapi/tender_reader_test.go`

**Step 1: The stand-in.** `sample.Model{Delay time.Duration}` implements `llm.Asker`. It reads the request's text parts, takes each line of page text that ends in a `$` amount as a line (label is the words before the amount), calls a line whose label starts "Total" a `total`, and gives every other line kind `base`, placed `sum`, and the row whose label starts with the same word as the line's label (the rows are the JSON in the last part), or no row. It returns the answer JSON. It is to tests what `internal/proposals/sample.Jev` is to the old reader. Test it on the text of a `proposals/sample.PDF`.

**Step 2: Write the failing reader tests** with the stand-in, a renderer that returns a tiny JPEG for every page asked, and the store.

- `TestDroppedTenderIsReadIntoLines`: file a sample proposal on a package with the switch on. Within the test's deadline the submission is `read`, has lines, and the price table's cells add up to its tender sum.
- `TestSwitchOffNothingIsSent`: the same with the switch off. The stand-in is never called and the state stays `''`.
- `TestPlacingASubmissionQueuesIt`: a submission with no package is not read; `PATCH` it into a package and it is.
- `TestReadAgainKeepsUserLines`: `POST .../tender-read` re-runs the pass; a user line survives; the column is unchecked.
- `TestTransportErrorIsRetried`: the asker returns a plain error: state `failed`, `tender_error` "The model could not be reached. It will be tried again.", `tender_retry_after` set.
- `TestRefusalIsNotRetried`: the asker returns `llm.ErrRefused`: `failed`, "The model declined to read this file.", no retry time.
- `TestNoRendererFails`: a nil renderer: `failed`, "Page images cannot be made on this server.", no retry time.
- `TestFourthFailureStopsRetrying`.
- `TestReadIsTimedOnceItCommits`: the observer receives `tender_read` for a reading of up to 100 lines and `tender_read_long` above.

**Step 3: Implement** `tenderReader` on the model of `proposalReader`: `Start(orgID, id string, began time.Time)`, the same in-flight map and wait group, two slots, a 150-second timeout.

```go
func (r *tenderReader) read(ctx context.Context, orgID, id string) error
```

1. `ClaimTenderRead`; return if false.
2. `TenderInput`. Open the blob (`r.blobs.Path(in.SHA256)`).
3. `identity.Extract(ctx, format, f, in.Size, tender.ExtractLimits())`, with the format from `proposalFormat(in.Filename)`. For a PDF note `identity.MissingTextPages(text)` and, when there are any, call `r.ocr` (the function `Options.ProposalOCR` already carries).
4. `tender.PagesOf(text, missing)`.
5. `tender.Pass(ctx, r.ask, r.render, r.model, path, format == "pdf", pages, in.Rows)`.
6. On an error: `FailTenderRead` with plain words and a retry time. `llm.ErrOff`: set the state back to `''` with no error. `tender.ErrNoRenderer` and `llm.ErrRefused`: no retry. Otherwise, when `llm.Retryable(err)` and attempts are under four: retry after 30 seconds, 2 minutes, then 10 minutes.
7. `tender.Reconcile(raw, pages, baseRows, in.StatedEx)`, where `baseRows` is `in.Rows` (base and option rows).
8. `CommitTenderRead(..., tender.PromptVersion, info.Cut || info.Dropped > 0, now)`.
9. Wake the broker. Observe the time since `began` when it is not zero.

Wiring:

- `Options` gains `TenderLLM llm.Asker`, `TenderModel string`, `TenderRender tender.Renderer`. With `TenderLLM` nil there is no reader and `Deps.ModelAvailable` is false.
- The old reader's `after` hook already starts the proposal-program read (*verify* where `preader.after` is set). Make it also call `QueueTenderRead(org, id, false)` and, when that returns true, `tenderReader.Start`.
- `patchSubmission`: after a write that sets `package_id`, queue and start the same way.
- New route `POST /projects/{id}/submissions/{sub}/tender-read`: `QueueTenderRead(..., true)`, start, return the table. Budget path `submission_write`. 409 with "The model is switched off for this organisation." when the switch is off.
- On start-up, beside `PendingSubmissionReads`, start every `PendingTenderReads`. Add `RunTenderRetries(ctx, 15*time.Second)` on the model of `RunProposalRechecks`, using `DueTenderRetries`.
- `cmd/sitewise/main.go`: `llm.New(llm.Options{APIKey: cfg.TenderKey, Logger: ...})`; pass it as `TenderLLM` only when `On()`. The renderer wraps `ocr.Pages` into `map[int]string`; pass nil when OCR is not set up, and log one line saying tenders will not be read by the model until Python rendering is installed.
- `web/tests/e2eserver/main.go`: `TenderLLM: tendersample.Model{Delay: 150 * time.Millisecond}` and a renderer that returns a one-pixel JPEG.

**Step 4:** Run the reader tests, then `go test -p 1 ./internal/httpapi -count=1`. Expected: PASS.

**Step 5:** Commit: `Tender: read a dropped tender in the background`.

### Task 8: Budgets and bench

**Files:**
- Modify: `bench/budgets.json`
- Create: `cmd/intake-bench/tender.go` (follow `cmd/intake-bench/award.go` and `group_award.go`)

**Step 1:** Add rows:

```json
	{"name":"tender_read","p50_us":15000000,"p90_us":30000000,"note":"tender filed to its prices shown, up to 100 lines: one model pass; a live gate run with cmd/tender-eval, not a replayed bench"},
	{"name":"tender_read_long","p50_us":60000000,"p90_us":90000000,"note":"the same, over 100 lines"},
	{"name":"price_table_tender","p50_us":150000,"p90_us":300000,"note":"price table for one package with five tender-read firms of 330 lines each (the price_table budget)"},
```

*Verify* how `bench/budgets.go` treats a row with no bench sample (some rows carry `"where"`); mark the two live rows the way it expects.

**Step 2:** The bench: seed one package with five submissions, commit a 330-line reading to each through `CommitTenderRead` (build the readings with `tender.Reconcile` from generated lines, as `BenchmarkReconcile330Lines` does), then time `GET .../prices` as `price_table_tender`.

**Step 3:** Run the bench as `docs/developing.md` describes. Expected: within budget. If the read is over budget, the first thing to try is one query for all lines of the package (not one per submission); the second is to leave `lines` out of the read and fetch them per cell. Report which was needed.

**Step 4:** Commit: `Tender: budgets and the table bench`.

### Finish

Write `docs/evidence/2026-11-14-tender-comparison-slice2.md`: what calls what (drop, old read, queue, pass, reconcile, commit, event), what can fail and what the screen will be told, the bench figures, and the one new dependency.
