# Tender Comparison Slice 3: The First Table on Screen

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** The manager sees "As tendered": the schedule down the left, the firms across, each column equal to its tender sum, with the firm's own lines behind every cell and a "to check" state they clear with one click.

**Architecture:** A new `web/src/tender/` folder holds the table. `Prices.tsx` keeps its header, menus, firm panel and award dialog and swaps its grid for the new table when the package has a tender-read submission. The only write in this slice is `check_column`.

**Tech Stack:** React, TypeScript, vitest, Playwright; Go for one operation and the award gate.

**Master plan:** `docs/plans/2026-11-14-tender-comparison-plan.md` (contracts C5 and C6). **Design:** sections 4, 5, 9 and 19.

**Status:** written 14 November 2026 from reading `main`. Not compiled or run. Names marked *verify* were not opened.

---

### Task 1: Types and the client

**Files:**
- Create: `web/src/tender/tenderApi.ts`
- Modify: `web/src/submissionApi.ts` (`PriceTable` gains `tender?: TenderTable`)

Write the TypeScript types for contract C5 exactly: `TenderFlag`, `TenderLine`, `TenderCell`, `TenderColumn`, `TenderRow`, `TenderAdjustment`, `TenderTable`. Money is `Money` from `submissionApi.ts` (a decimal string or null).

```ts
export type TenderOp =
  | { op: "check_column"; submission: string; checked: boolean };
// Slices 4 and 5 add the other operations.

export const tenderApi = {
  ops: (project: string, pkg: string, version: number, ops: TenderOp[]) =>
    request<PriceTable>("POST", `/projects/${project}/procurement/packages/${pkg}/tender/ops`, { version, ops }),
  readAgain: (project: string, sub: string) =>
    request<PriceTable>("POST", `/projects/${project}/submissions/${sub}/tender-read`),
};
```

Run `npm run build`. Expected: PASS. Commit: `Tender: client types for the table`.

### Task 2: The words, as pure functions

**Files:**
- Create: `web/src/tender/words.ts`
- Test: `web/src/tender/words.test.ts`

**Step 1: Write the failing tests** (vitest).

- `cellWords`: a priced cell gives its amount as `$96,000.00` (use `fmt` from `CostGrid`); priced with a null amount gives "Amount to type"; `included` gives "Included"; `excluded` "Excluded"; `not_priced` "Not priced"; no cell gives "—".
- `markWords`: `high` "High", `low` "Low", `gap` "Others priced this", `""` nothing.
- `remainderWords(why)`: `""` gives nothing; `rounding` "Rounding"; `margin:15` "Margin not shown at line level (15%)"; `unbroken` "The firm did not break this down"; `differs` "Lines differ from the tender sum".
- `columnState(col)`: `pending` "Waiting to read prices"; `reading` "Reading prices…"; `failed` the column's `error`; `read` and not checked "To check"; `read` and checked "Checked".
- `basisWords`: `fixed` nothing; `estimate` "Estimate, not a fixed price"; `cost_plus` "Cost plus, not a fixed price"; `rates_only` "Rates only".

**Step 2:** Run `npm test -- words`. Expected: FAIL. **Step 3:** Implement. **Step 4:** PASS. **Step 5:** Commit: `Tender: the table's words`.

### Task 3: The table

**Files:**
- Create: `web/src/tender/AsTendered.tsx`
- Create: `web/src/tender/LinesPanel.tsx`
- Create: `web/src/tender/tender.css`
- Modify: `web/src/Prices.tsx`

Read `web/src/cost-tables.css` and `web/src/states.css` first and use their classes and tokens for the table, the header band, selected, hover and to-check states. Do not invent colours.

**What it shows.** One `<table>` with the caption "As tendered, ex GST".

- **Head.** The first column is "Item". One column per firm. Reuse the firm header cell that `Prices.tsx` renders now (name button that opens the firm panel, received date, status, upload control, Award button): lift that JSX into a component in `Prices.tsx` and render it from both tables, changing nothing in how it behaves. Under it, for a tender column, add: the state words from `columnState`; when `read`, a button "Mark checked" (or "Checked", which un-checks); when `failed`, a "Try again" button that calls `tenderApi.readAgain`; `basisWords` when there are any.
- **Body.** One row per `tender.rows` entry that is not the system row. The row header is the row's label. Each cell shows `cellWords`, the mark from `markWords` as a small tag, and an amber dot with the title "Something here needs a look" when `to_check`. A cell with lines is a button (`aria-expanded`) that opens the lines panel for that firm and row.
- **Foot.** The system row, labelled "Not broken down", showing the cell and, under it, `remainderWords`. Then the row "Tender sum" in bold: the sum, with the `tender_sum_how` words as its title, and "Lowest" on the lowest column. Then a row "To look at" listing each column's flags as short paragraphs (the `text` of each flag), followed by its notes and exclusions under a "The firm says" disclosure.
- A column whose submission is not tender-read (an old-style submission in the same package) shows "Read the old way" in the head and its `cells` from the existing column in the body. Its foot shows its `totals.base_total`.

**The lines panel.** `LinesPanel` is the list behind one cell: for each of the firm's lines on that row, counted first: the label, the amount ex GST, the printed figure when it differs ("printed $105,600.00 inc GST"), "p. 9", each flag's text, and "not counted" for an uncounted line. "p. 9" is a button that calls `onOpenDocument(documentId)` (*verify* whether it can take a page; if not, open the document and leave the page number as text). Render it as a row beneath the table row, spanning the columns, the way an expanded row works in the cost plan (*verify* in `CostGrid.tsx`). For the system row it lists the counted lines with no row.

**In `Prices.tsx`:**

```ts
const tenderMode = !!table?.tender && table.tender.columns.some(c => c.state !== "");
```

When `tenderMode`, render `<AsTendered …/>` in place of both the `ConsultantComparison` block and the `<details open><summary>Fee schedule</summary>…</details>` block. Keep everything else on the page. The rates section and the earlier-proposals list still render. The old "+ New, Row" menu item stays (it edits schedule rows, which the new table reads).

The Award button in a tender column's head is disabled until the column is checked, with the title "Check this column first".

**Empty and waiting states.** A column that is `pending` or `reading` shows the state words in every body cell's place as one tall cell ("Reading prices…", with the existing `Pending` component). No tender-read column at all and the switch on: the old screen, unchanged.

Run `npm run build`. Expected: PASS. Commit: `Tender: the first table, read only`.

### Task 4: Checking a column

**Files:**
- Create: `internal/store/tender_ops.go`
- Test: `internal/store/tender_ops_test.go`
- Create: `internal/httpapi/tender_ops.go`
- Modify: `internal/httpapi/projects.go` (both route maps; budget path `submission_write` for now)
- Test: `internal/httpapi/tender_ops_test.go`

**Step 1: Write the failing tests.**

Store:

- `TestCheckColumn`: `ApplyTenderOps` with `check_column` sets `tender_checked_at` and `tender_checked_by`, bumps the version, and the table read shows `checked: true`. With `checked: false` it clears them.
- `TestCheckNeedsARead`: a column that is `reading` or `failed` cannot be checked: `ErrInvalidInput`, "This firm's prices have not been read yet."
- `TestCheckRefusedWhileLinesAddToMore`: a column whose system-row cell is negative cannot be checked: "Lines add to more than the tender sum. Take the repeated lines out of the count first."
- `TestStaleVersionIsRefused`: `ErrVersionConflict` when the version is not the package's.
- `TestUnknownOpIsRefused`: `ErrInvalidInput` naming "operation 1".
- `TestOpsOnAnotherOrgsPackage`: `ErrNotFound`.

HTTP: 200 with the table; 409 on a stale version; 422 with the message on an invalid operation; 404 for another org; 401 with no session.

**Step 2:** Run. Expected: FAIL.

**Step 3: Implement.**

```go
// TenderOp is one named edit (contract C6). Slices 4 and 5 add fields.
type TenderOp struct {
	Op         string `json:"op"`
	Submission string `json:"submission,omitempty"`
	Checked    *bool  `json:"checked,omitempty"`
}

// ApplyTenderOps applies a batch to one package's tender tables, or none of
// it. It never calls a model.
func (s *Store) ApplyTenderOps(ctx context.Context, org, project, pkg, actor string, version int64, ops []TenderOp) error
```

One transaction: `lockProject`; read `packages.tender_version FOR UPDATE` (404 when the package is not in this org and project); refuse a stale version with `ErrVersionConflict`; apply each operation; bump the version; `procurementEvent`. At most 200 operations in a batch. An error names the operation by its position, counting from 1.

The handler decodes `{version, ops}`, calls it, and answers with `writePriceTable`.

**Step 4:** Run the tests. Expected: PASS. **Step 5:** Commit: `Tender: mark a column checked`.

### Task 5: Award needs a checked column

**Files:**
- Modify: `internal/store/awards.go` (`awardPlan`)
- Test: `internal/store/tender_award_test.go`

**Step 1: Write the failing tests.**

- `TestAwardRefusedUntilChecked`: a consultant package with a tender-read, unchecked submission: `PreviewAward` and `Award` return `ErrInvalidInput` with "Check this firm's column before awarding."
- `TestAwardWritesTheTenderSum`: check the column, award every priced row including "Not broken down": the award total equals the tender sum and the cost plan lines add to it.
- `TestAwardOfAnOldSubmissionIsUnchanged`: an old-style submission awards exactly as before.

**Step 2:** Run. Expected: FAIL. **Step 3:** In `awardPlan`, after the status check, read `tender_state` and `tender_checked_at` for the submission and refuse an unchecked tender-read one. Leave the builder branch alone: slice 6 changes it. **Step 4:** PASS, and `go test -p 1 ./internal/store -count=1` passes. **Step 5:** Commit: `Tender: award needs the column checked`.

### Task 6: In the browser

**Files:**
- Create: `web/tests/tender-table.spec.ts` (follow `web/tests/builder-comparison.spec.ts` for sign-in, project and package set-up, and the synthetic proposal PDFs)

**Step 1: Write the test.**

1. Sign in as org `a`. `PUT /api/account/model` with `{enabled: true}`.
2. New project. Add a Town Planner row (`POST .../procurement/rows` with its discipline id: *verify* the id in `web/tests/procurement.spec.ts`). Add two firms.
3. Upload a synthetic proposal for each (`/__e2e/proposal.pdf` with three priced lines whose labels start with the first word of three schedule rows, and a "Total" line). Give the second firm one line with a label that matches no row.
4. Open the package. Expect the caption "As tendered, ex GST".
5. Each column's "Tender sum" cell shows the printed total. The second firm's "Not broken down" shows the unmatched line's amount.
6. Click a priced cell: the lines panel shows the line's label and "p. 1".
7. Both columns say "To check" and Award is disabled. Click "Mark checked" on the first: it says "Checked" and Award is enabled. Reload: still checked.
8. `GET .../prices` and assert in JSON that each column's priced cells add up to `tender_sum` (add the decimal strings in the test).

Add a second test in the same file: with the switch off, the same uploads show the old screen (no "As tendered" caption).

**Step 2:** Run `npx playwright test tests/tender-table.spec.ts`. Expected: PASS. Run `npx playwright test tests/builder-comparison.spec.ts tests/procurement-award.spec.ts`: they must still pass, since the switch is off by default.

**Step 3:** Take one screenshot at 1440 by 1000 into `.tools/` for the owner, as the builder test does.

**Step 4:** Commit: `Tender: browser test for the first table`.

### Finish

Write `docs/evidence/2026-11-14-tender-comparison-slice3.md`. Include the `price_table_tender` figure again now that the screen reads it, and the screenshot's path.
