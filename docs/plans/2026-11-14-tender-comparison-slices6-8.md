# Tender Comparison Slices 6 to 8: Head Contract, Report and Export, Retiring the Old Reading

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Head contracts and combined engagements use the new tables; both tables export, print and feed the comparison report; and the parts of the old reading that the new one replaces are switched off for tender-read submissions.

**Architecture:** No new mechanism. These slices bring the remaining package kinds and outputs onto the lines and cells the earlier slices built, and bring the written rules of the product in line with what was built.

**Tech Stack:** Go, PostgreSQL, React, TypeScript, the existing `internal/xlsx` writer.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md`. **Design:** sections 8, 10, 19 and 20.

**Status:** written 14 November 2026. These three slices touch code this plan's author read least. Every name here is marked *verify* unless the master plan's "Read before starting" lists it. Start each slice by reading the files it names and correcting this file where it is wrong, in the same commit as the work.

Do the three slices in order, one evidence note each.

---

## Slice 6: Head contract and combined engagements

### Task 6.1: Read what depends on the lump-sum row

Read `internal/store/builder_comparison.go` (`builderPackage`, `ensureBuilderSchedule`, `builderReview`, `checkBuilderAlignment`), where `ensureBuilderSchedule` is called, the builder branch of `awardPlan` in `internal/store/awards.go` (it needs the user's `comparison:appointment` cell), `web/src/BuilderComparison.tsx`, and the uses of `stage:lump_sum` in `web/src/Prices.tsx`. Write down, in the slice note, every place that reads `stage:lump_sum`.

### Task 6.2: Head contract rows

**Files:**
- Modify: `internal/store/builder_comparison.go`, and the schedule sync it sits in
- Test: `internal/store/tender_builder_test.go`

**Step 1: Write the failing tests.**

- `TestHeadContractHasMainWorksRowsWhenTheSwitchIsOn`: with the org's switch on, a new head contract package's schedule holds the Main Works RFT rows ("Preliminaries and site establishment", "Substructure", and so on) and no `stage:lump_sum`.
- `TestHeadContractKeepsTheLumpSumRowWhileItIsUsed`: a package with an old-style submission whose `stage:lump_sum` cell the user confirmed, or with a live award on it, keeps that row when the switch is turned on.
- `TestHeadContractWithTheSwitchOffIsUnchanged`.
- `TestTenderInputForAHeadContractOffersMainWorksRows`.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.** A head contract package resolves its schedule from the Main Works RFT (`knowledge/rfp/main-works.yaml`, discipline `trade.main_works`) the way a trade package does. `ensureBuilderSchedule` inserts `stage:lump_sum` only when the org's switch is off, or the package already has a submission that is not tender-read. It retires `stage:lump_sum` when the switch is on and no cell decided by a user and no live award refers to it.

**Step 4:** Run the tests, then every existing builder test in `internal/store` and `internal/proposals`. Expected: PASS, with no existing test changed.

**Step 5:** Commit: `Tender: a head contract is compared on the Main Works rows`.

### Task 6.3: Awarding a tender-read head contract

**Files:**
- Modify: `internal/store/awards.go` (`awardPlan`)
- Test: `internal/store/tender_award_test.go`

Tests: a tender-read, checked builder submission awards on its priced rows and "Not broken down" with a total equal to the tender sum, and does not need the `comparison:appointment` confirmation; unchecked, it is refused with slice 3's words; an old-style builder submission still needs its confirmation.

In `awardPlan`, a tender-read submission takes the plain path (lines by item key) whatever the package kind. The builder confirmation applies only to a submission that is not tender-read.

Commit: `Tender: award a head contract from its checked column`.

### Task 6.4: The screen for a head contract

**Files:**
- Modify: `web/src/Prices.tsx`
- Test: `web/tests/tender-builder.spec.ts`

In tender mode a builder package renders the same two tables. `BuilderWorkSections` and `BuilderQuotedTotals` are not rendered for a tender-read column. The Award button follows the checked state, not `cells["stage:lump_sum"]`.

Browser test: switch on; a Main Works row; two builders with synthetic quotes of six lines whose labels start with the first words of six Main Works rows, and a total; both tables show; check; award; the cost plan shows the lines adding to the tender sum. Run `web/tests/builder-comparison.spec.ts` unchanged: it runs with the switch off and must pass.

Commit: `Tender: the head contract screen`.

### Task 6.5: Combined engagements

**Files:**
- Modify: `internal/store/tender_read.go` (`TenderInput` uses the group schedule), `internal/store/group_award.go`
- Test: `internal/store/tender_group_test.go`, `web/tests/tender-group.spec.ts`

Tests: for a two-discipline combined engagement the rows offered to the pass are the group's rows, each key starting with its member's key; a reading with lines on both members' rows projects cells on both; the table groups rows under each discipline as the old table does; award through the group path writes each member's lines under its discipline and puts "Not broken down" under the engagement itself.

Read how `group_award.go` treats a key with no member prefix before writing the last test, and make the system row a line under the engagement.

Commit: `Tender: combined engagements on the new tables`.

### Finish slice 6

`docs/evidence/2026-11-14-tender-comparison-slice6.md`, with `price_table_group` and `award_write` re-run.

---

## Slice 7: Export and the comparison report

### Task 7.1: Excel and CSV

**Files:**
- Modify: the handler `exportPriceTable` and what it calls (*verify*: `internal/httpapi/submissions.go`, `internal/xlsx`)
- Test: beside the existing export test

For a package in tender mode the workbook has three sheets, every figure written as a value computed in Go:

1. **As tendered.** Rows down, firms across, ex GST. Each cell's amount or its word (Included, Excluded, Not priced). "Not broken down". "Tender sum".
2. **Adds and subs.** One row per adjustment, signed amounts, with a second column per firm saying "applied" or "suggested". "Like for like", counting applied only.
3. **Lines.** One row per line: firm, row, the firm's wording, printed figure, amount ex GST, GST basis, page, counted or not, and its flags in words.

CSV gives the first two tables, one after the other.

Tests: open the written workbook with the reader the existing export test uses; each firm's column on sheet 1 adds (in the test) to its tender sum; a suggested amount is not in "Like for like"; no formula cells. Export of an old-style package is byte-for-byte what it was (keep the old path).

Budget: the existing export budget for this route (*verify* its name in `bench/budgets.json`), with five firms of 330 lines.

Commit: `Tender: export both tables and the lines`.

### Task 7.2: The comparison report

**Files:**
- Modify: `internal/store/comparison_report.go`, `internal/proposals/report.go`, `web/src/ProcurementReport.tsx`
- Test: beside the existing report tests, and `web/tests/tender-report.spec.ts`

For a tender-read column the report's price section shows three figures: the tender sum, the applied adds and subs as one signed figure with the count of rows behind it, and the like-for-like total. It lists the applied rows by label and amount. It lists each column's "To look at" flags in words.

Sign-off is refused while any live tender-read column of the package is not checked: "Check every firm's column before signing off." Add the store test and the HTTP test.

The report still prints on one A4 page for three firms. With more than four firms, or more than eight applied rows, the adds and subs list moves to a second page: test the page count the way the existing report test does.

Commit: `Tender: the comparison report shows like-for-like totals`.

### Task 7.3: The Procurement board

**Files:**
- Modify: `internal/store/comparison.go` (`cardOf`, `ProcurementBoard`), the board's cards in `web/src/Procurement.tsx`

A tender-read firm's card shows its like-for-like total, "Lowest" by that total, and "To check" until its column is checked. An old-style card is unchanged. The `procurement_board` bench must stay in budget with eight rows of five tender-read firms.

Commit: `Tender: board cards show like-for-like and to-check`.

### Task 7.4: Print

The two tables print with the page's existing print rules (*verify* the `@media print` rules in `web/src/prices.css`): both tables, firms in columns, the lines panels closed, the flags row kept. Check by printing to PDF in the browser test and counting pages.

Commit: `Tender: print the two tables`.

### Finish slice 7

`docs/evidence/2026-11-14-tender-comparison-slice7.md`.

---

## Slice 8: Retiring the old reading, and the written rules

Do this slice only after the owner has used the new tables on a live package and said so.

### Task 8.1: Stop asking Jev what the pass now reads

**Files:**
- Modify: `internal/proposals/read.go` (`Context`, `ask`), `internal/store/submissions.go` (`SubmissionForReading`)
- Test: `internal/proposals/` beside the existing question tests

Add `SkipPrices bool` to `proposals.Context`. `SubmissionForReading` sets it when the org's switch is on, the server has a model, and the submission has a package. With it set, `ask` leaves out the price questions and the total question and keeps the firm, package, kind and rate questions.

Tests: with `SkipPrices` the call holds no question whose field starts `price:` for a fee row and none for the total; without it the call is exactly as before (compare against the existing golden or recorded call). `cmd/proposal-eval` still passes: it never sets the flag.

When the pass then fails for good, the submission has no priced cells. That is correct: the column says "Prices not read" and offers "Try again".

Commit: `Tender: Jev is not asked for prices the pass reads`.

### Task 8.2: Hide what no longer applies

**Files:**
- Modify: `web/src/Prices.tsx`, `web/src/ConsultantComparison.tsx`, `web/src/ComparisonAlignment.tsx`

For a package in tender mode: no "Move or combine fee" bar, no "Fees awaiting alignment" row, no quote-review panel. Leave their code in place for old-style packages. Delete nothing a test still covers.

Commit: `Tender: the old alignment controls are off in tender mode`.

### Task 8.3: The written rules

**Files:**
- Modify: `PRODUCT.md` (the Procurement entry and the Positioning section)
- Modify: `AGENTS.md` (the Procurement sentences in "These paths do not call Jev"; the tender entry in the surface notes: drop "nothing built")
- Modify: `docs/developing.md` (only the lines that say Jev is the only AI, as `docs/design/2026-11-14-chat-control-and-email-intake.md` section 12 lists)
- Modify: `docs/deployment.md`, `docs/operations.md`
- Modify: `docs/design/2026-11-14-tender-comparison.md` (status: built; anything that changed)

What each must come to say:

- **PRODUCT.md, Procurement:** a tender dropped on a row is filed and read in the background; within about a minute the price screen shows two tables, "As tendered" on the RFP or RFT rows with each column equal to the firm's tender sum, and "Adds and subs" giving a like-for-like total; the manager moves, splits and corrects lines, applies or types adds and subs, marks each column checked, and awards; no edit calls a model.
- **PRODUCT.md, Positioning:** Jev is the only AI that files documents or reads them into fields; one model, which each org switches on, lists the lines of a tender in the background, supplies no amount of its own, and applies nothing without the manager's acceptance.
- **AGENTS.md:** "A proposal dropped on a procurement row is read after filing with one Jev fan-out and, when the org has switched the model on, one tender pass; the price table, line moves and splits, adds and subs, checking, award, undo, ratings and the comparison report call neither."
- **Deployment:** the two settings (`SITEWISE_ANTHROPIC_HAIKU_KEY`, `SITEWISE_TENDER_MODEL`); that page images need the same Python, pypdfium2 and Pillow the OCR helper needs; that without them tenders are read the old way; how to confirm the model is reachable (the log line `llm call`).

Do not write the privacy page's wording. Add a bracketed placeholder where the processors are listed, in the form the legal pages already use, naming Anthropic as a processor of tender pages for orgs that switch the model on, and tell the owner it needs their lawyer.

Commit: `Tender: bring the written rules in line with what is built`.

### Task 8.4: Going live

A checklist for the owner, to run with them, not for them:

1. The key and the model id are set on the server, and the server log shows the model client is on.
2. The owner switches the model on for their own org in Account.
3. Three real tenders the owner knows well are dropped on a package. Compare each column with the document.
4. `cmd/tender-eval` replays on the server's build with the reviewed key and passes.
5. The retention setting of the provider account is confirmed by the owner.
6. Nothing is switched on for any other org by anyone but its owner.

### Finish slice 8

`docs/evidence/2026-11-14-tender-comparison-slice8.md`: what was retired, what still reads the old way and why, and the final figures against every budget in design section 12.
