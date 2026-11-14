# Tender comparison: the gate after slice 1

Date: 14 November 2026. Branch `tender/comparison`. No case ids, firm
names or amounts appear here; the working files are in
`data/eval/tender/private/` (git-ignored).

## Result

**Not a clean go.** As the plan words it, slice 2 should wait for the owner.
The owner's instruction for this run was to build every slice without
stopping, so the later slices were built anyway, behind the org switch,
which is off until an owner turns it on. Read this note before switching
it on for a live package.

What holds on every run:

- no tender sum is shown wrong without a flag;
- no option, saving or extra is counted in a price without a flag, on the
  tune split (one disputed provisional sum on the hold-out, below);
- every amount shown is in the page text SiteWise sent (R3 keeps 99% or
  more, scans included);
- the read budgets for tenders of up to 100 lines.

What does not hold, or is not known:

- **The row mark swings from run to run.** The same prompt on the same
  files gave 87%, 77% and 69% of dollars in the right row on the tune
  split. The pass mark is 80%.
- **Tender sum right outright** was 86% and 93% on the tune split (mark
  90%) and 6 of 6 on the hold-out.
- **Large schedules are not measured for rows.** See "The key".
- **The key is not reviewed by the owner.**

## The marks

Model `claude-haiku-5-5`, thinking off, effort low. Twenty keyed cases:
fourteen tune, six hold-out.

| Mark | Pass | Tune, prompt t1 | Tune, t2 run A | Tune, t2 run B | Hold-out, t2 (once) | All 20, t2 |
|---|---|---|---|---|---|---|
| Sum shown wrong with no flag | 0 | 0 | 0 | 0 | 0 | 0 |
| Sum right | 90% | 86% | 93% | 86% | 100% | 90% |
| Dollars in the right row | 80% | 86.9% | 77.0% | 68.9% | 92.4% | 75.2% |
| The same, large tenders only | reported | not measured | not measured | not measured | not measured | not measured |
| Outside money counted, unflagged | 0 | 0 | 0 | 0 | 1 | 1 |
| Amounts kept by R3 | reported | 99.2% | 100% | 99.8% | 99.8% | 99.8% |
| Amounts kept, scans (3 cases) | reported | 100% | 100% | 100% | none | 100% |
| Seconds, up to 100 lines, p50 / p90 | 15 / 30 | 9.1 / 14.7 | 10.0 / 16.2 | about the same | 11.3 / 19.4 | 9.9 / 19.4 |
| Seconds, over 100 lines, p50 / p90 | 60 / 90 | 44.9 (1 case) | 56.6 (1 case) | 62.5 (1 case) | 51.1 / 63.5 (2 cases) | 62.5 / 63.5 (3 cases) |
| Cost | reported | | | | | about half a US cent a tender |

"Dollars in the right row" is the mean over cases that have keyed lines.
The "t1" column is after the reconciler and OCR fixes below. With three
long cases the p50 over 100 lines sits at the 60 second budget.

The answer is about 110 tokens a line. The shorter-answer round (plan
slice 1, task 8, step 4) was not run.

## The survey

Before the key was written, every file in the corpus (64) went through the
pass once under prompt t1. All 64 were read. Two of the first six answers
came back with no lines at all for files that plainly print prices; asked
again, both were read. The pass now treats an empty answer for price pages
as a failed read and asks again (`tender.ErrEmpty`).

## The key

- Twenty cases: twelve builder submissions (four for the amenities block,
  two of the five camp schedules, six for the two houses) and eight
  consultant proposals. Three are scans with no text layer.
- **How it was made.** Each case was seeded from a reading, then checked
  against the document: the tender sum was found printed on the page
  (image or text) and, for every case with keyed lines, the counted lines
  add up to that printed sum to the dollar, including a 10% and a 15%
  margin printed at the foot. One 75-section estimate was keyed from the
  document's own summary table, not from a reading. Rows were judged from
  each line's wording against the schedule row labels; a row was changed
  only where the reading's choice was clearly wrong.
- **What is missing.** Three of the five camp schedules are not in the key:
  their tender sum could not be established with confidence in the time.
  The two that are in it are keyed for the tender sum only. So the key
  says nothing about rows on a 200-line schedule. Three files whose price
  is ambiguous (a recommendation report, a two-variation proposal, an
  approved-and-proposed fee) were left out.
- **The hold-out is not blind.** Its cases were read in the survey. It was
  held out of the prompt rounds only.
- `"reviewed": false`. Only the owner sets it.

A proxy for the large schedules, from the two that are keyed for their
sum: both tender sums are right; in one the lines differ from the sum by
3%, in the other by 14% the wrong way (lines counted twice), and both say
so pack by pack.

## What the rounds changed

1. **Reconciler** (each with a failing test first):
   - a subtotal or total of money outside the price is never counted;
   - a part total that adds up lines listed under other headings is not
     counted beside them;
   - a figure on the page that is the tender sum plus a round percentage,
     and that the model did not list, raises "check the tender sum";
   - a total the model read as including GST, whose own label does not say
     so, raises the same flag (a builder's margin read as GST).

   The third and fourth removed the one unflagged wrong sum.
2. **OCR.** A scan was read for its first four pages only, so a price on
   page seven was thrown away by R3. The tender pass now reads up to
   thirty pages, four at a time (`OCR.ReadMissing`).
3. **Scoring.** A column flag "lines add to more than the tender sum"
   counts as a flag on outside money; wording where an amount should be
   ("at cost") is not counted as an amount R3 threw away.
4. **Prompt t2.** One sentence: a margin added at the foot is not GST.
   Kept: the margin case is right in both t2 runs. The row mark fell in the
   same runs for reasons the sentence does not explain (a one-line
   proposal's row, a scan read as one line on one run), which is the
   run-to-run swing above.

## The one hold-out miss

A provisional sum that a builder lists inside a trade's lines, and that
its printed contract sum leaves out, was counted. The column is flagged
"lines are short of the tender sum", not "over", so it scores as
unflagged. Whether that sum is in the price is arguable from the document.

## What the owner is asked

1. Review the key (`data/eval/tender/private/answer-key.json`, with
   `buildkey.py` beside it showing every decision).
2. Decide on the large schedules: key two of them line by line and measure
   Haiku there; or measure Sonnet 5.5 on those files only
   (`SITEWISE_TENDER_MODEL`); or ship with the pack-by-pack flags.
3. Decide whether a swing of this size in the row mark is acceptable for a
   table the manager checks before use, or whether each tender should be
   read twice and the more self-consistent reading kept (not built).
