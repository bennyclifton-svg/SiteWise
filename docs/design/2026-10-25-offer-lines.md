# Fee lines above the line by default

25 October 2026. Lane A (consultant fee comparison, award amounts).
Owner request while testing Hale (town planner, traffic, access): each
firm's own fee lines should sit above the line and add to the firm's total,
mapped to the RFP stages where possible; optional fees stay below; the
row arrows remain the override.

## Why rows fell below the line

The appointment view only put source fees above the line when they added
to the *settled stated total*, and the stated total was rarely settled: an
insurance cover ($20,000,000) or a contract-clause amount ($50,000) on a
later page is larger than the fee, and the reader refuses a total smaller
than another figure. Without it, the view showed one empty "Professional
fee" row and every task below the line, together with insurance cover,
rate ranges, invoicing splits, repeated agreement pages and GST-inclusive
restatements.

## What changed

`proposals.ReadOffer` reads the offer from the stored quoted fees, when the
comparison is read. No Jev call, no re-reading, no stored value changes.

1. Figures that are not fees are set aside: insurance cover (and the same
   figure printed again), clause amounts, rate ranges, invoicing splits
   ("$2,079 (30%)"), repeated listings, the total restated, and the total
   including GST.
2. The fee lines are the run of fees just above a printed total that add
   to it to the cent. The total may be labelled ("Total (excl. GST)") or
   not (a bare "$15,500 + CFT" that is the sum of Tasks 1–4). A repeated
   breakdown with the same amounts counts once; the better-labelled copy
   is shown.
3. With no printed total, the lines are the current fees as listed and the
   total is their sum; the total says so.
4. One printed total with no breakdown stays the single professional-fee
   row, now priced from that total when the letter's tax basis is known.
5. Optional, conditional and alternative fees stay below the line.
6. A line with no basis of its own takes ex GST only from an ex-GST total
   it adds to exactly. Otherwise its amount stays unknown and the editable
   professional-fee row remains for the PM to confirm, with the printed
   figures shown below it.

Labels: a bulleted scope sentence keeps its row's phase number ("Phase 1A –
Undertake …"); a fragment gives way to the row's first cell.

Stage mapping, in order: a stage whose every word is in the fee's label
("Lodgement" in "Task 4 – Collation and Lodgement"); Jev's coverage
reading; Jev's partial coverage. Each is a suggestion marked "check
mapping". Equal amounts never establish equal scope.

Arrows and mappings: moving a row is a placement and is kept. Saving a
stage mapping keeps the row in its default place. A choice saved before
this change that kept an offer line below the line only repeated the old
default, so it is not applied; one that moved a row above is.

Award resolves the same projection, so it awards exactly the included
rows. A PM-confirmed fee still wins unless the lines add to it.

## Hale, before and after (stored readings, no re-read)

| Firm | Before | After |
|---|---|---|
| Town planner A | Not confirmed | 4 task lines, matches quoted total |
| Town planner B | Not confirmed | 2 phase lines, optional site visit below |
| Town planner C | Not confirmed | 2 stage lines, matches quoted total |
| Traffic A | Not confirmed | one professional fee from the printed total |
| Traffic B | Not confirmed | printed lines; GST basis unclear, confirm the fee |
| Traffic C | Not confirmed | one line, matches quoted total |
| Access A | Not confirmed | one line, no total printed |
| Access B | Not confirmed | 2 stage lines (appendix repeat hidden) |

## Not changed / follow-ups

- Traffic B prints "(including GST)" beside both fee lists; the reader
  marks its basis unknown because hourly rates elsewhere say excluding GST.
  Taking GST from a fee table's lead-in line is a reader change that needs
  the proposal answer-key replay and a re-read; not done here.
- No schema, service, dependency or Jev call. Budgets unchanged:
  price_table 150/300 ms, submission_write 100/250 ms, award_write
  200/500 ms (p50/p90).
