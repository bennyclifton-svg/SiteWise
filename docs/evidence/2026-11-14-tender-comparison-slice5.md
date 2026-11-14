# Tender comparison slice 5: adds and subs

Date: 14 November 2026. Branch `tender/comparison`.

## What was built

- `058_tender_adjustments.sql`: `tender_adjustments` and
  `tender_adjustment_amounts`, both captured by the change log.
- `internal/tender/suggest.go`: `Suggest` and `Others`, pure functions.
- `store.syncSuggestions`, run after every reading, edit and undo, and the
  seven operations of contract C6 in `ApplyTenderOps`.
- The table read fills `adjustments`, `suggested` and each column's
  `like_for_like`; the lowest is marked by like-for-like total among
  columns with no unanswered gap.
- `web/src/tender/AddsSubs.tsx`.

## The rules as built

- The system suggests: each option or alternative a firm prices (the
  firm's figure); each saving (negative); each provisional sum or prime
  cost item the firm says is extra to its price; and each row a firm
  excluded, did not price or left out while at least half the other firms
  priced it (no figure).
- **Nothing is applied by the system.** A suggested amount does not count.
- **The system never invents a figure.** A suggested amount is a printed
  line's own cents. A gap gets a figure only on the manager's click: the
  highest or the average of the other firms' cells on that row, computed in
  Go at that moment and stored with its source, or a figure they type.
- A suggested amount follows its line while it is suggested. An applied
  amount never moves, and outlives the line it came from.
- A system row the manager removes is dismissed and stays away, also after
  "Read again". A line read again keeps its id (matched on page, wording
  and printed figure), which is what keeps a dismissal, an applied amount
  and a suggestion tied to it.
- Taking a tendered line out makes "Less: …" for that firm, applied, at the
  firm's own figure. The first table does not change.
- "Apply this firm's own figures" applies every suggestion of that firm
  that carries a printed figure, and leaves gaps alone.

## What could fail

Every refusal is 422 with words: "Choose the highest, the average, or type
a figure."; "No other firm priced this row."; "A suggested row keeps its
name. Add a note instead."; "Only a counted line can be taken out."; a
submission of another package; a row that is no longer there.

## Timings

Re-run after this slice, five firms of 330 lines each:
`price_table_tender` 12.5 / 13.3 ms; `tender_edit` 31.7 / 33.3 ms;
`change_undo` 13.0 / 15.2 ms. The twenty-adjustment variant the plan asks
for was not run separately: the adjustment read is two indexed queries per
table read.

## Checked in the browser

`web/tests/tender-adds-subs.spec.ts`: three firms read by the stand-in;
three suggestions; apply the firm's own figure; a gap filled with the
highest of the others; the manager's own row with a typed subtraction; a
line taken out; a suggestion removed that stays removed after "Read
again"; undo brings it back; the lowest mark sits on the lowest
like-for-like total. Screenshot: `.tools/tender-adds-subs.png`.

## Where the plan was wrong, or was changed

- A dismissed system row loses its amounts, so nothing hidden goes on
  counting.
- "Undo apply" is for system rows. On the manager's own row an amount is
  cleared, not un-applied.
- Taking a line out is on the line's menu ("Take out as a subtraction")
  as well as by dragging it onto the second table; the browser test uses
  the menu.
- An option line mapped to a schedule option row is shared by the firms
  that priced it, as the plan says; a firm with several lines on one
  option row has them added in Go.
