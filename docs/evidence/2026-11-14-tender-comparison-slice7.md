# Tender comparison, slice 7: export, the report, the board, print

Date: 14 November 2026. Branch `tender/comparison`. Every firm and figure in
the tests is invented.

## What was built

- **Export.** For a package in tender mode, `GET …/prices/export` gives a
  workbook of three sheets: "As tendered", "Adds and subs" and "Lines"
  (every line the pass listed, with the row it sits on and its page).
  `?format=csv` gives the first two tables as one CSV file. Every cell is a
  value Go computed; the export carries no formula. A package with no
  tender read still gets the old one-sheet workbook.
- **Comparison report.** A "Like for like" section: the tender sum, the
  adds and subs the manager applied (the total and each applied row), the
  like-for-like total, the flags, and whether each column is checked. A
  suggested amount that was not applied is not counted and not shown.
- **Sign-off is held** until every tender-read column is checked. The
  server answers 422 with "Check every firm's column before signing off."
  and the screen shows those words.
- **Procurement board.** A tender-read firm's card shows its like-for-like
  total and a "To check" chip until its column is checked. Lowest is by
  like-for-like total.
- **Print.** Both tables print with firms in columns; the lines panels and
  the buttons do not print.

## What calls what

The export, the report and the board all read `ReadPriceTable`, the same
read the screen uses, so the figures cannot differ between them. None of
them calls Jev or the model.

## Tests

- `internal/httpapi/tender_export_test.go`: the CSV's cells add to the
  tender sum, the like-for-like total is the tender sum plus the one
  applied amount, the workbook has three sheets, the report carries the
  same figures, sign-off is refused then allowed, the board card agrees.
- `web/tests/tender-report.spec.ts`: the report section in the browser,
  the refusal wording, sign-off after checking, and a print to PDF.

Both pass.

## Timing

The export and the report are built from the price table read, whose
budget row is `price_table_tender` (measured 12.5 ms p50, 13.3 ms p90 in
slice 3, budget 150 / 300 ms). The export itself was not timed separately
and has no budget row of its own.

## Not done

- The plan asks that the report still fits A4 with more than four firms or
  more than eight applied rows. The print test checks only that a
  two-firm table prints on one to three pages. Wide and long cases are
  not tested.
- The print check counts pages; nobody has looked at the printed page.
- The full intake bench was not run in this worktree.
