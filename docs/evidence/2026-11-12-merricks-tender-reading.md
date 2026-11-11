# Merricks: reading a builder's tender (12 November 2026)

One document: `Coastal-Builders.pdf`, a 12-page Buildxact quote for a new
house at Merricks North (36 priced categories, $3,547,495.00 inc GST). The
first read (11 November) is critiqued in the session that wrote this note.
This records what was changed, what was verified, and what is still open.

Lane A. No new dependency. No new Jev question. No migration.

## What changed

Profile reading (`internal/jobs/source.go`, `fulltext.go`, `worker.go`,
`internal/store/sources.go`):

- A bullet glyph printed after its wrapped item is put back on its own line
  by page geometry (same baseline, to its left).
- A short plain line set left of the bulleted list below it, or set in type
  1.2 times the page's body size, is that list's or table's heading.
- A bulleted item is one passage; it ends at the next bullet, not at a full
  stop.
- Each row of a priced table is one passage (a line that is only an amount
  closes the row). "4.12 Rock for gravel driveway" above its price is a row,
  not a heading; "6.000 Ea" is a quantity, not a clause.
- A page header or footer is a passage of its own. From its second
  appearance it is stored as `repeated_header_or_footer` (outcome
  background) and is not sent to Jev.
- `SourceVersion` is `source-4`, so the next Update re-extracts a project's
  documents. That costs one label and one evidence fan-out per passage.

Profile reconcile (`internal/profile/reconcile.go`):

- A system's "not included" under another trade's heading no longer
  contradicts the same document including it elsewhere. The value is
  included, amber, with the exclusion listed as an alternative.
- A row's note and first source are its strongest reading.
- "Tenders differ" needs two documents.

Client (`internal/profile/addressee.go`, `knowledge/profile/project_details.yaml`):

- A line such as "Priya Nair & Tom Whitlock C/O Harbour Lane Design" offers the party before
  "C/O" as a client candidate. Jev still picks.

Quote reading (`internal/proposals/harvest.go`, `read.go`, `quoted.go`,
`builder.go`):

- A table row whose description wraps around its number and amount is one
  row with its own words.
- A priced row above a figure no longer lends it its words (three rows under
  "Preliminaries including ... insurance" were dropped as insurance cover).
  A row in a column that states its GST basis is a price whatever its words.
- A column heading is found past any number of priced rows, and on two
  lines ("Allowance incl." over "GST").
- Subtotal + printed GST = total is recognised when the GST was summed per
  row (here 8 cents off 10%).
- GST basis settles by rule when three or more priced columns agree. The
  stated total settles by rule on the printed ex-GST total.
- "Builders" and "Constructions" are company words, so the firm settles by
  the existing name-and-email-domain rule.
- Allowance schedules are marked "Allowance inside the quoted price". Rows
  under one title that add up to a printed total are its breakdown; five or
  more that add up to none raise an issue.
- A section longer than 19,000 characters continues in a new one, so no page
  is dropped from the stored text.

## Verified

On the real PDF, by running the code directly (no Jev):

| Check | Before | After |
|---|---|---|
| Passages that are a bare bullet | 25 | 0 |
| Trade headings carried as the section | 0 | all, pages 2–8 |
| Category rows with their own label | 26 of 36 | 36 of 36 |
| Category rows read | 33 of 36 | 36 of 36 |
| Rows reconcile to the printed total | not checked | yes, $3,547,495.00 |
| Firm, GST basis, stated total | blank | green by rule |
| Stored text, pages 5–8 | dropped | kept |

Timing on this document: splitting 1.58 ms before, 1.67 ms after; quote
harvest and rules 56 ms before, 65 ms after. Neither is on a path a user
waits on. `cmd/intake-bench` passes.

Tests: `internal/jobs`, `internal/proposals`, `internal/store`,
`internal/knowledge`, `cmd/...` pass. `internal/profile` fails only
`TestMeridianAnswerKey` at 73/78 with the same four misses as before the
change. `TestProgramRoutes` and the latency gate's path count fail as they
did before; neither package uses the changed code.

## Not verified

The live re-read of Merricks did not run. Every Jev call returned 402:
"Your organization has no available TypeSafe API credits". The document was
re-extracted (283 passages, 17 repeated footers) and its reading failed
after five attempts. The profile still shows the 11 November build with the
payment notice. After credits are added, press Update on Merricks.

The stored submission still holds the 11 November quote reading; a
submission is read when it is filed.

## Open

- Missed scope (solar and battery, blockwork, retaining walls, smoke
  alarms, the lift read from a lift circuit): expected to change with
  one-row passages. Check after the re-read.
- Coverage counted 233 of 247 passages as "needs mapping". Check after the
  re-read before changing the count.
- Bedrooms and bathrooms from room names ("Bed 4", "Master Ensuite") is an
  inference, not a reading. Owner decision.
- Where a builder's tender lands (head contract package, cost plan) and
  whether a tenderer's offer sets profile facts as stated. Owner decision.
- Victorian terms and planning rules. Owner decision.
- The summary block on page 12 is extracted with labels and amounts out of
  order for the profile reader. The quote reader pairs them by baseline.
