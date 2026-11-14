# Tender comparison: as tendered, adds and subs, like for like

Date: 14 November 2026. Status: built on branch `tender/comparison` (slices 0
to 8), not merged, not deployed. The measured gate was not a clean pass and
the answer key is not owner-reviewed: see
`docs/evidence/2026-11-14-tender-comparison-gate.md`. What differs from this
note is listed in `docs/evidence/2026-11-14-tender-comparison-slice8.md`.
Lane A (product core).

This note replaces every earlier comparison plan, including
`docs/plans/2026-11-13-Levelled-tender-comparison-specification.md` and
`docs/plans/2026-10-19-comparison-alignment.md`. It was written from 64 real
tender submissions and a measured trial, not from those plans.

It works inside `AGENTS.md` rule 1 as the owner replaced it on 14 November
2026 (`docs/design/2026-11-14-chat-control-and-email-intake.md`, section 12).
This is that rule's "background job that proposes line structure, row
mappings and qualifications for quotes", and it brings what that design
waits on: the model client, the org switch and the first measured timings.
Section 19 says how each condition of the rule is met.

Firm names and prices from the real tenders are left out of this note on
purpose: the repository is public. The corpus lives in `testing/rft/`, which
git ignores. Figures in the examples below are invented.

## 1. Outcome

A project manager drops the tenders for a package onto Procurement and, within
about a minute, sees two tables:

1. **As tendered.** The RFP or RFT price schedule down the left, the firms
   across the top, a price in each cell, and each firm's tender sum at the
   foot. Each column always adds up to the figure the firm printed.
2. **Adds and subs.** A short schedule of additions and subtractions, firm by
   firm, that the system proposes and the manager edits. The tender sum plus
   the adds and subs is the **like-for-like total**.

The manager glances down the first table to see whether the firms agree on
what the works are worth, sees where the system thinks there is a gap or an
oddity, fixes what is wrong by moving lines, and finishes the levelling in the
second table. They do not open Excel.

The aim is 80% of the work, 80% right. The manager finishes the rest. The
system never claims to have levelled the tender on its own.

This is price only. Non-price ratings, the recommendation and sign-off stay
where they are in the comparison report.

## 2. Why

It serves time-to-decision and answer trust. Today a manager retypes each
tender into a spreadsheet to compare them. That takes hours, and the one
hand-built comparison table in the corpus has a subtotal that does not add up.

## 3. What the real tenders show

The corpus: 15 builder submissions over four projects (a camp redevelopment
of about $50M with five builders, an amenities block, a new house, a house
renovation), 48 consultant fee proposals over 15 disciplines for one
warehouse job, and one recommendation report a project manager wrote.

| What the tenders do | Seen in | What this design does |
|---|---|---|
| Return the issued schedule, filled in differently: priced at heading level with the lines marked "Included"; "incl", "refer Pack 2", "na", "NIL", "PSUM" | Camp, amenities block, consultants | A cell can be an amount, Included, Excluded or Not priced. Every line keeps the firm's own wording. |
| Put preliminaries in one pack, or spread across packs, or in a block of their own; margin as its own line or inside the rates | Camp | Rows are the RFT's rows. Each firm's lines fold into them wherever the firm put them. The manager moves or splits a line. |
| Ignore the schedule: one firm gives a single total, another 36 categories, another 77 sections | New house | Whatever is not broken down sits in one row, "Not broken down", so the column still equals the tender sum. |
| Print money that must not be added: options, savings, alternatives ("add $X"), hourly rates, provisional sums outside the price | All | These never enter the first table. They are offered in the second. |
| Print the same price twice: the firm's own table and the returned form; a summary and its detail; a fee and its payment schedule | Consultants, new house | Code counts each dollar once and says which level it counted. |
| Leave margin out of the lines and add it at the foot | New house, renovation | Code sees that the gap is a round percentage of the lines and labels it. |
| Mix GST bases: lines inc GST, total ex GST | New house, consultants | Every amount is converted to ex GST in Go. |
| Price in parts with no overall total | Camp, amenities block | The tender sum is the sum of the parts, and the screen says so. |
| Quote on a different basis: a cost-plus estimate beside fixed lump sums | New house | Flagged on the firm's column head. |
| Scatter other dollar figures through the document: insurance limits, the value of past projects | Most | The reader is told to ignore them. The arithmetic check catches a stray. |
| Print clean tables that text extraction scrambles | Camp, new house, renovation | The reader is given the page image as well as the text. |
| Carry no text at all | Three architecture proposals | Read from the image, marked "read from a scan". |

The project manager's own comparison table in the corpus is the target:
stages down the left, firms across, cells reading "Excluded", "Incl." and
"Not submitted", an allowance added to level one firm, a conforming total,
options below the line, and a note moving money between two rows.

## 4. The screen

One screen per package, replacing today's price table.

```
AS TENDERED (ex GST)                Firm 1      Firm 2      Firm 3
Preliminaries                       96,000      24,000     126,000
Demolition and strip-out            74,000      51,000      28,000
Substructure                        84,000      41,000   Included
Structure                           45,000      50,000     154,000  High
Envelope                            91,000      74,000      79,000
Interiors                          231,000     166,000     252,000
Hydraulic services                  77,000     133,000     135,000
Electrical services                 39,000      37,000      43,000
Provisional sums and prime cost     Not priced  Not priced  15,000
Not broken down                          0      31,000           0
TENDER SUM                         737,000     607,000     832,000

ADDS AND SUBS
Asbestos removal, provisional sum  +15,000     +15,000           .   suggested
Option: epoxy grout                +18,000           .           .   suggested
Less: pool fence (not in the RFT)        .           .     -34,000   applied
LIKE FOR LIKE                      737,000     607,000     798,000
```

- A cell opens to show the firm's own lines behind it: wording, amount, page.
  A page number opens the document at that page.
- "High" and "Low" mark a cell far from the other firms on that row.
- An amber dot marks a figure to check (section 9).
- The like-for-like total counts applied adds and subs only. The table says
  how many suggestions are not applied.
- The lowest like-for-like total is marked. A firm with rows not priced is
  never marked lowest.
- Each firm's column is **to check** until the manager marks it **checked**
  (one click a firm, or all at once). The reading is a proposal until then.
  Award and report sign-off refuse a firm whose column is not checked.
- Rates (consultants), earlier and superseded submissions, award and the
  comparison report stay as they are.

## 5. Rules of the first table

- **Rows** are the package's RFP or RFT price rows (section 8), then one
  system row, "Not broken down", then the tender sum.
- **The tender sum** is the firm's printed price for its base offer, ex GST.
- **A cell** is the sum of that firm's counted lines placed in the row.
- **Not broken down** is the tender sum less every counted line. It holds a
  lump sum the firm did not break down, a margin shown only at the foot, or a
  difference the reading could not explain. The column therefore always
  equals the tender sum. A negative figure there means lines are counted
  twice, and is flagged.
- Moving or splitting a line never changes the tender sum.
- Correcting a line's amount changes "Not broken down", not the tender sum.
  Correcting the tender sum is its own act.
- A user's value is never overwritten by a later reading.

## 6. Rules of the second table

- A row is one addition or subtraction. Each firm has a signed amount in it,
  or nothing.
- Each amount is **suggested** or **applied**. Only applied amounts count.
  The system never applies one.
- Each amount says where it came from: the firm's own figure with its page,
  the highest of the other firms, the average of the other firms, or typed.
- The system suggests:
  1. each priced option or alternative a firm offers (the firm's figure);
  2. each saving a firm offers (the firm's figure, negative);
  3. each provisional sum or prime cost item the firm says is extra to its
     price (the firm's figure);
  4. each gap: a row the firm excluded or did not price while at least half
     the other firms priced it. The suggestion has no amount. Applying it
     asks the manager to choose the highest of the others, their average, or
     a typed figure.
- The manager adds a row, types amounts, applies, un-applies, deletes and
  notes freely.
- Dragging a tendered line down from the first table makes a subtraction for
  that firm ("Less: ..."), applied, linked to the line. The first table does
  not change.
- Suggestions are rebuilt when lines change. Applied amounts and the
  manager's own rows are left alone.

The system does not invent a price. A suggested amount is either printed in
the firm's tender or computed from the other firms' cells on the manager's
click, and it says which.

## 7. How a tender is read

```
drop -> filed (as today) -> Jev: firm, package, kind (as today)
     -> code picks the pages -> one tender pass (Haiku)
     -> Go checks and reconciles -> the two tables, to check
```

1. **Filing and identity are unchanged.** Rules and the one Jev fan-out
   settle the firm, the package and the kind of submission. When the tender
   pass is on, Jev is no longer asked the price, rate and total questions.
2. **Code picks the pages.** A page with priced lines goes as its image and
   its text. A page that talks about exclusions, qualifications, options or
   alternatives goes as text. The first two pages go as text. Insurance
   certificates are dropped. At most 30 image pages; the whole request stays
   under 100,000 tokens. A page with no text layer is put through the
   existing OCR first, so that it has text.
3. **One tender pass.** Claude Haiku 5.5, thinking off, effort low,
   structured JSON out. The model is given the package's schedule rows. It
   proposes the line structure: for every printed price, in document order,
   the firm's wording, which printed amount goes with it, its GST basis,
   what kind of money it is, whether it is part of the firm's price, the
   heading it sits under, the schedule row it belongs to, and whether it is
   sure. It lists subtotals and totals too. It also returns up to 15
   exclusions and up to 6 notes. It is told to do no arithmetic. The prompt
   and schema that were trialled are in section 16.
4. **Go checks and reconciles** (section 7.1), writes the lines, and
   refreshes the table over the existing event stream.

The pass runs in the background, as proposal reads do now, and only for an
org that has switched the model on (section 17). Nothing on the screen waits
for it except the cells of that one firm, which say "Reading prices".

### 7.1 What Go does with the reading

These rules are the core of the feature. They are deterministic and tested
without a model.

- **R1 Amounts.** Parse each printed amount to cents. An amount that will not
  parse keeps its line and is flagged.
- **R2 GST.** Convert to ex GST from the line's stated basis, else the
  document's. Unknown is treated as ex GST and flagged.
- **R3 The model supplies no amount.** An amount is kept only when the same
  figure is in the text SiteWise gave the model for the page the line cites
  (its text layer, or its OCR text). If the figure is on a page within two
  either side, the page is corrected. If it is in none of them, the amount
  is thrown away: the line stays, with no amount, marked "amount not found
  on the page", for the manager to type. A line whose page text came from
  OCR is marked "read from a scan".
- **R4 Subtotals.** A printed subtotal that equals the sum of the counted
  lines before it in its group is marked checked. A line the model called
  ordinary scope, whose amount equals the sum of two or more counted lines
  just before it in its group, is treated as a subtotal.
- **R5 Repeats.** Within a group, lines that repeat an earlier line (same
  amount and same row) are dropped when dropping them makes the group equal
  its printed subtotal.
- **R6 Level.** For a group with a printed subtotal: if its detail adds up,
  the detail is counted. If not, the printed subtotal is counted as one line
  and the detail is kept beneath it as a breakdown, flagged with the
  difference.
- **R7 Tender sum.** Candidates are each printed total and the sum of the
  printed part totals. Take, in order: the candidate equal to the counted
  lines; the candidate equal to the counted lines plus a round percentage;
  the only printed total; the sum of the part totals when no overall total
  is printed; otherwise the largest printed total, flagged. With no printed
  total, the tender sum is the counted lines, flagged "no total printed".
  The total the existing rules find is a cross-check: a disagreement is
  flagged.
- **R8 The remainder.** Tender sum less counted lines goes to "Not broken
  down", with the first explanation that fits: nothing left; a round
  percentage of the lines (5, 7.5, 10, 12, 12.5, 15, 17.5, 20 or 25), shown
  as "margin not shown at line level"; equal to one line the model put
  outside the price, which is then counted and flagged; equal to one counted
  line, which is then treated as a breakdown and flagged; otherwise "lines
  differ from the tender sum by $X".
- **R9 Cells.** The counted lines are summed by row into the existing price
  cells, so award, the comparison report and the cost plan read what they
  read today.

Go never changes a printed amount, never spreads a remainder across rows and
never scales a figure.

### 7.2 Where Jev sits

Jev keeps filing and the firm, package and kind of a submission. It does not
gate the tender pass, and low-confidence passages are not handed from Jev to
the model. The reasons, from the corpus and the trial:

- The failure is upstream of Jev. Jev picks among amounts that code harvests
  from extracted text. On the hardest tables that text has already lost which
  amount belongs to which label. Jev's confidence measures its pick from the
  text it was given; it cannot see that the text was wrong.
- The job is a different shape. A builder's tender is 30 to 300 lines to be
  listed, not a handful of fields to be picked.
- Cost is no reason to ration the model. A read costs about half a US cent.
- The check that matters is arithmetic. Lines that add up to the printed
  total are a stronger test than a confidence score, and Go does it for free.

A later slice may ask Jev to cross-check the row each line was put in, since
picking a row from a list is what Jev does well and its confidence is
calibrated. It is not in this version.

## 8. Rows

- A consultant package uses its RFP stage rows, options and reimbursables,
  as now.
- A trade package uses its RFT rows (the parts of the building), as now.
- **A head contract uses the Main Works RFT rows**: design completion,
  preliminaries, demolition, site works, substructure, structure, structural
  steel, envelope, interiors, joinery, the services, lifts, and provisional
  sums and prime cost items. The single "Quoted construction price" row is
  retired where nothing depends on it.
- A combined engagement uses its group rows, as now.
- The manager can add, rename and remove a row, as now.
- Packs, stages within a discipline and separable portions are folded in:
  a row sums across them, and each line shows the portion it came from. There
  is no second axis in this version.

## 9. What is flagged

On a line: amount not found on the page; read from a scan; row not sure;
counted to match the total; treated as a breakdown.

On a cell: High or Low (more than double or less than half the median of the
other firms on the row, with at least two others priced); Not priced or
Excluded where at least half the others priced it.

On a column: not a fixed price (estimate, cost plus, rates only); GST basis
not stated; no total printed; tender sum to check; lines differ from the
tender sum; a pack or section whose detail does not add up, with the
difference; options or alternatives offered.

Every flag is computed in Go and says why in plain words.

## 10. What the manager can do

All of these are saved reads and writes. None calls Jev or the model.

- Move a line, or a whole cell, to another row: drag, or right-click and
  choose the row. Keyboard: select, then M.
- Split a line across rows. The parts always add up to the line.
- Correct a line's amount, wording or kind. Add a line. Take a line out of
  the count.
- Correct the tender sum or the GST basis.
- Take a line out as a subtraction (drag down).
- Add, rename or remove a row.
- Everything in section 6.
- Mark a firm's column checked, or un-check it.
- Undo a change.
- "Read again" re-runs the tender pass for one firm. The manager's own
  lines and corrections are kept, and the column goes back to "to check".
- Export both tables to Excel and CSV; print with the comparison report.

Selection, inline editing and the right-click menu follow the document
register and the cost plan (`web/src/states.css`).

Every edit is a named operation (move line, split line, set line, add line,
count or uncount line, set tender sum, add adjustment, set adjustment amount,
apply adjustment, and so on) applied in one batch write, the way Program
edits are (`ApplyProgramOps`). Undo comes from the project change log in
`docs/plans/2026-11-14-change-log-undo-operations.md`, which must land before
the editing slice. That gives chat the same operations later with nothing
extra to build.

## 11. Data

Additive. Nothing existing is dropped.

- `tender_lines`: one row per line a firm printed. Submission, order, the
  firm's wording, the amount as printed, cents ex GST, GST basis, page, kind
  (base, provisional sum, prime cost, option, saving, rate, subtotal, total,
  GST), where (in the price, inside another line, outside the price),
  heading, schedule row, state (priced, included, excluded, not priced),
  note, checks, who decided (model, rule, user), what it was split from, and
  whether it is counted.
- `tender_adjustments`: one row per add or sub for a package: wording, note,
  order, origin (system or user), reason, linked schedule row, a stable key
  for system suggestions.
- `tender_adjustment_amounts`: adjustment, submission, signed cents, state
  (suggested or applied), source, linked line.
- `submissions` gains the tender-pass state, its attempts and its flags, the
  prompt version, the price basis, the tender sum with how it was chosen,
  and who checked the column and when.
- `submission_prices` stays, written from the counted lines (R9).
- The org gains one setting: the model is on or off, who switched it and
  when. It is off until an owner switches it on.
- The new tables join the project change log so their edits can be undone.

Every row carries `org_id`.

## 12. Speed and cost

Measured in the trial (section 14), one read per tender, thinking off:

| | Lines | Seconds |
|---|---|---|
| 18 tenders under 100 lines | 9 to 59 | 5 to 22, median 11 |
| 4 tenders of 124 to 206 lines | | 38 to 64 |

Time is the length of the answer: first output after 1 to 4 seconds, then
about 340 tokens a second, about 3.3 lines a second.

Proposed budgets, to go in `bench/budgets.json`:

| Path | p50 | p90 |
|---|---|---|
| `tender_read`: filed to prices shown, up to 100 lines | 15 s | 30 s |
| `tender_read_long`: over 100 lines | 60 s | 90 s |
| `tender_reconcile`: the Go work after the model answers, 330 lines | 100 ms | 250 ms |
| `price_table`: both tables, five firms of 330 lines | 150 ms | 300 ms (unchanged) |
| `tender_edit`: any move, split, correction, add, sub or undo | 100 ms | 250 ms |

The reconcile, table and edit budgets are benchmarks that fail the build. The
two read budgets depend on an outside service, so they are a live gate run on
demand against the private corpus, as the Jev live tests are.

A read is cut off at 120 seconds and retried with the existing back-off.

The first speed lever, to try before the feature ships, is a shorter answer:
about 108 tokens a line now, with a target of 60. The second is reading long
schedules in page groups in parallel; the trial tried it and it only halved
the wait while making the stitching unreliable, so it is not in this version.

Cost: US$0.002 to $0.015 a tender, median half a cent. The whole trial, 56
runs, cost US$0.39.

## 13. When things go wrong

- No key set, or the org has not switched the model on: the tender pass is
  off and proposals are read as they are now.
- The model is unreachable, rate limited or times out: retried with the
  existing back-off. The firm's column says "Prices not read" with a Retry.
  The tender sum the rules found still shows.
- The model declines or returns nothing usable: the same as unreachable,
  without the automatic retry.
- The answer is cut off: flagged "reading incomplete", what parsed is kept.
- A scan with no text layer: OCR supplies the text first. With no OCR text,
  the lines come back without amounts and the manager types them.
- A tender carries wording aimed at a model ("ignore your instructions"):
  it is data. The model can only return lines in the schema, every amount is
  checked against the page, and every row key against the schedule.
- A file that is not a PDF: the extracted text is sent without images.
- The package is not yet known: the price read waits until the manager
  places the submission, since the rows are needed.
- The schedule changes after a read: lines on a removed row fall to "Not
  broken down" and are flagged. Nothing is re-read.
- A revised submission is a new submission and supersedes the old, as now.
- Two firms, or one: outlier flags need two others; gaps need one other.
- A wrong org or a missing file: 404, as on every procurement path.

## 14. The trial

Run on 14 November 2026 against the real corpus with Claude Haiku 5.5: 56
runs over 22 tenders while the prompt was shaped, ending with one run of
each in the final set-up. Scripts and raw answers are kept outside git in
`data/eval/tender/private/trial-2026-11-14/`. The figures below are from
that final run. One of the 22 offers alternatives and prints no single
price, so 21 have a known tender sum.

- **Tender sum.** Right for 18 of 21 as the trial script scored it. Two of
  the three misses were the script's own rule for a price printed in parts.
  The reconciler since written and tested for the plan (slice 1), replayed
  over the same 22 answers with no new model calls, gets 20 of 21. The one
  miss is the model returning no total; in the product the total the
  existing rules find is the fallback.
- **Lines.** Replayed through that reconciler, of the 21: 13 add up to the
  tender sum to the dollar. One is explained as a 10% margin at the foot.
  Three are lump sums the firm did not break down. Three do not add up and
  are flagged: a 77-section estimate whose detail was read incompletely
  (14% short), and two of the three large camp schedules (0.5% and 3.7%
  short, flagged pack by pack, with 99% and 81% of their dollars on rows).
  One returned no total. In an earlier run that read the estimate's summary
  pages, all 77 sections matched the page and the gap was exactly a 15%
  margin.
- **Figures.** 959 of 961 amounts the model returned were found in the page
  text it had been given, so R3 would have thrown away two. The model often
  cites the page number printed on the page instead of the page's position,
  which R3 corrects.
- **Scrambled tables.** A two-page estimate whose text extracts as noise was
  matched up exactly from the image, margin included.
- **A scan.** A proposal with no text layer was read from the image alone,
  exactly. Rule 1 does not allow that in the product: the page goes through
  OCR first and each amount must be in the OCR text. That path was not
  trialled.
- **Thinking on** fixed some judgement calls but took two to three times as
  long to start answering, and it collapsed the large schedules into pack
  totals. It is off.
- **Wording matters.** The largest schedule came out nearly perfect under one
  prompt and poorly under the next. The prompt has to be tuned against an
  answer key, not by eye.

The honest reading: small and medium tenders are at the 80% mark now. The
300-row schedules are not, and the checks make that visible instead of
silent.

## 15. Proving it

Before any screen is built, slice 0 builds a private answer key and a Go
harness that replays the read against it.

- The key: at least 20 tenders from the corpus, each with its tender sum, GST
  basis, every counted line and amount, the lines that are outside the price,
  and the right row for each line. The owner checks it.
- Pass marks:
  - No tender sum is shown wrong without a flag: 100%.
  - Tender sum right outright: at least 90%.
  - Counted dollars landing in the right row: at least 80%, over all tenders.
  - Money outside the price that is counted in it: none without a flag.
  - The read budgets in section 12.
- If Haiku 5.5 cannot reach the marks on the large schedules after tuning,
  the model is a setting (`SITEWISE_TENDER_MODEL`) and the next step is to
  measure Sonnet 5.5 on those tenders only. That is an owner decision.

## 16. The reader's contract

Model `claude-haiku-5-5` (setting `SITEWISE_TENDER_MODEL`), thinking off,
effort low, streamed, structured output. Key from
`SITEWISE_ANTHROPIC_HAIKU_KEY`. New dependency: the official Go SDK,
`github.com/anthropics/anthropic-sdk-go/v2`.

The client lives in its own package, `internal/llm`, with the admission
limit, circuit breaker, deadline and call log that `internal/jev` has. Chat
will use the same client; it does not get a second one. Everything about
tenders lives in `internal/tender`.

Each line the model returns:

| Field | Meaning |
|---|---|
| `p` | Page, by the label code puts before each page |
| `label` | The firm's wording, 80 characters at most |
| `amt` | The amount exactly as printed; empty if none |
| `g` | GST basis of that amount: ex, inc, unknown |
| `st` | priced, included, excluded, na, tbc |
| `kind` | base, ps, pc, option, saving, rate, subtotal, total, gst |
| `where` | sum (its own part of the price), inside (already in another line), outside (extra to the price) |
| `group` | The heading it sits under: pack, stage, section, discipline |
| `row` | Schedule row key, or empty |
| `note` | 12 words at most |
| `sure` | False when the row or role was a guess |

With them: the firm's name, the price basis (fixed, estimate, cost plus,
rates only, unknown), the document's GST basis, exclusions and notes.

The instructions that were trialled, in short:

- List every printed price at every level, in order. Do not skip detail
  because a subtotal exists, or a subtotal because its detail is listed.
- Do no arithmetic. Never work out, correct or invent a figure.
- Use the image to see which amount belongs to which label; copy the amount
  from the text layer.
- A grid of prices (disciplines by stages) is one line per cell.
- The same prices printed twice are listed once. A summary and its detail
  are both listed, the summary as subtotals.
- Provisional sums and prime cost items are part of a builder's price unless
  the firm's words say they are extra.
- Do not list insurance limits, the value of past projects, payment
  milestones or late fees.

The full prompt is in the trial folder and becomes
`internal/tender/prompt.txt`. It is versioned, and a reading records the
version that produced it.

## 17. Privacy

A tender pass sends pages of the submission to Anthropic's API. Submissions
name people and firms and state prices.

- One switch per org covers every model use (owner decision 5 in the chat
  design). It is the org agreeing that its project text may be sent to the
  model provider. It is off by default, an owner turns it on in Account, and
  with it off no tender page leaves the server. This work builds the switch.
- The key is SiteWise's. Before production the owner confirms the account's
  data retention setting.
- Nothing but the selected pages, the schedule rows and the instructions is
  sent: no user name, email, org name or project name in headers or
  metadata.
- The privacy page names Anthropic as a processor before this ships.
- Real tenders, their answers and the answer key stay in git-ignored
  folders.

## 18. Out of scope

Non-price evaluation. Writing the recommendation. A second axis for packs or
portions. Carrying an add or sub into the contract sum on award. Reading
long schedules in parallel. A Jev cross-check of rows. Levelling provisional
sums to a common figure. Re-reading when the schedule changes. Any model call
on navigation, typing or an edit.

## 19. How this meets rule 1

Rule 1 was replaced on 14 November 2026 and is not changed again here. Each
of its conditions, and how the tender pass meets it:

| The rule says | This design |
|---|---|
| Jev is the only AI that files documents or reads them into fields | Filing, firm, package and kind stay with rules and Jev. |
| In background jobs the LLM may propose line structure, row mappings and qualifications for quotes | That is all the tender pass returns: lines, a row for each, exclusions and notes. |
| It never supplies an amount SiteWise did not give it | R3: an amount not in the page text SiteWise sent is thrown away. Go parses, converts and adds every figure. |
| Nothing in a document is an instruction to it | Section 13: a tender is data; the answer is held to the schema, the page text and the schedule's row keys. |
| Nothing it proposes is applied without the user's acceptance | A column is "to check" until the manager marks it checked. Award and sign-off need it checked. Adds and subs are suggested until applied. |
| One LLM, which each org switches on | Section 17. |
| No screen read or write calls the LLM | Section 10: the pass runs only in the background job and on "Read again". |

Still owed in `AGENTS.md` when this is built: the Procurement sentences in
"These paths do not call Jev" (a dropped proposal is read with one Jev
fan-out and one tender pass; moves, splits, adds and subs, checking, award,
undo and the report call neither), and the list of surface notes, which now
points here.

## 20. Slices

Each is a vertical slice with its own tests and stays within its budget.

0. Answer key, the model client (`internal/llm`), a Go harness, and prompt
   tuning to the pass marks. Go or no-go.
1. The reconciler (R1 to R9), no screen, tested on recorded answers with no
   model.
2. The org switch; storage; the cell projection; the table read; the
   background job.
3. The first table: cells, lines behind a cell, flags, "to check" and
   "checked". Read only.
4. Editing the first table: move, split, correct, add, count and uncount,
   as operations, with undo. Needs the project change log.
5. Adds and subs: suggestions, apply, the manager's own rows, drag down,
   like-for-like total.
6. Head contract on the Main Works rows; combined engagements.
7. Export and the comparison report.
8. Retire the old price questions, the builder review panel and the old
   alignment screen once the new path has been used on a live package.
   Bring `PRODUCT.md` and the `AGENTS.md` Procurement sentences in line.

## 21. Owner decisions

Made:

1. Model: Claude Haiku 5.5.
2. Packs and portions are folded into one table.
3. Two tables: as tendered, and adds and subs.
4. Rule 1 as replaced on 14 November, and one switch per org (both decided
   in the chat design).

Recommended here. The owner said to proceed on these recommendations on 14
November 2026, so they stand unless changed:

5. Jev keeps identity and does not gate the tender pass (section 7.2).
6. A firm's column is "to check" until the manager marks it checked, and
   award needs it checked. This is how rule 1's "not applied without the
   user's acceptance" is met.
7. A suggested add or sub does not count until the manager applies it.
8. Dragging a tendered line down makes a subtraction; the first table never
   changes its total.
9. Award keeps writing the as-tendered figures. Adds and subs are for
   comparison only in this version.
10. A head contract's rows become the Main Works RFT rows.
11. Scanned pages go through OCR first, and an amount not in the OCR text is
    typed by the manager. The trial read a scan exactly from the image
    alone; allowing that would need rule 1 to say so.
12. The budgets in section 12 and the pass marks in section 15.
13. The privacy steps in section 17, including the line on the privacy page.

## 22. Done

A manager drops three tenders on a package and, without typing a figure,
sees both tables with each column equal to its printed tender sum; moves a
line; applies one suggested add; marks the columns checked; reads a
like-for-like total; and exports it. With the org's switch off, no tender
page leaves the server and the old reading still works. The answer key
passes its marks. Every budget in section 12 is met.
