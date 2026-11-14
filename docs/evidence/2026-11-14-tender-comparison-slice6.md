# Tender comparison slice 6: head contract and combined engagements

Date: 14 November 2026. Branch `tender/comparison`.

## Every place that read `stage:lump_sum` (task 6.1)

| Where | What it does with the row |
|---|---|
| `internal/store/builder_comparison.go` `ensureBuilderSchedule` | inserts it as the head contract's one row |
| `internal/rfp/schedule.go` `GenericSchedule` | the generic "Lump sum fee" row of a discipline with no RFP knowledge (a different row, same key) |
| `web/src/BuilderComparison.tsx` | shows the quoted total and takes the confirmed comparison amount into its cell |
| `web/src/Prices.tsx` | relabels it, leaves it out of the builder's row list, gates Award on a user-decided cell, writes the confirmed amount |
| `internal/store/awards.go` `awardPlan` | a builder award needs every awarded cell decided by the user (in practice that cell) |

## What was built

- **Rows.** With the org's switch on, a head contract's schedule is the
  Main Works RFT rows (`ensureRows`, the path a trade package takes). The
  single "Quoted construction price" row is kept only while something
  rests on it: a live submission read the old way, a figure the manager
  confirmed there, or a live award on such a submission
  (`keepOrRetireLumpSum`). With the switch off nothing changes.
- **Old columns beside new ones.** While that row is live, a column read
  the old way is judged on it alone: the Main Works rows do not count
  against it as unread.
- **The pass.** Neither the system row nor `stage:lump_sum` is offered to
  the model.
- **Award.** A tender-read, checked submission is awarded on its priced
  cells whatever the package kind, and needs no confirmed quoted total. An
  old-style builder submission still needs its confirmation.
- **Combined engagements.** The pass is offered the group's rows. A
  reading projects cells onto each member's rows. `planGroupAward` treats
  "Not broken down" as a line under the engagement itself, requires it
  when it is priced, and refuses an unchecked tender-read column.
- **Screen.** In tender mode a head contract shows the same two tables;
  the old builder panels are not rendered and the firm's actions are not
  tucked behind "⋯". A combined engagement's rows are headed by
  discipline. A column read the old way offers "Read prices" when the
  model is on.

## What could fail

- **A table read meeting a writer.** The price table read keeps a
  package's rows in step as it reads, inside a repeatable-read snapshot.
  When a reading had just landed, the read failed with a serialization
  error (seen in the head contract browser test, HTTP 500). The read now
  starts again, up to four times.
- Turning the switch on changes the rows of every head contract package
  of the org the next time its table is opened. A package with old-style
  submissions keeps their quoted-price row and gains the Main Works rows.
- Turning the switch off afterwards does not put the single row back for a
  package whose submissions are all tender-read; its Main Works rows stay.

## Timings

Not re-run: `price_table_group` and `award_write` are bench rows and the
full bench was not run in this worktree (slice 2 note). The store tests of
both paths pass; the tender table and edit budgets are unchanged.

## Checked

- Store: `TestHeadContractWithTheSwitchOffIsUnchanged`,
  `TestHeadContractHasMainWorksRowsWhenTheSwitchIsOn`,
  `TestHeadContractKeepsTheLumpSumRowWhileItIsUsed`,
  `TestAwardATenderReadHeadContract`, `TestTenderOnACombinedEngagement`;
  the existing builder and group award tests pass unchanged.
- Browser: `web/tests/tender-builder.spec.ts`; `builder-comparison.spec.ts`
  passes unchanged with the switch off.
- **Not written:** `web/tests/tender-group.spec.ts`. The combined
  engagement is covered in the store, not in a browser.

## Where the plan was wrong, or was changed

- The award gate for a head contract was already in `awardPlan` from
  slice 3 (a tender-read submission takes the plain path).
- One submission failed to open once in the first browser run ("This file
  could not be opened.") and did not again; the reader now logs the cause.
