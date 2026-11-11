# Run-time feedback: watching the profile get drawn up

Date: 10 November 2026. Status: proposal for owner review; nothing built.
Lane B throughout. Slices 3 to 5 add small reads; none adds a Jev call or
changes what is read, stored, decided or accepted.

Read first: `docs/design/2026-10-19-activity-feedback.md` (the activity
contract this extends), `docs/design/2026-10-10-profile-layout.md` and
`web/DESIGN.md` (Trace). A clickable prototype with sample data shows the
motion and the compressed layout; it is shared separately.

## 1. The problem

Today an Update is a progress ring and a document count.

- The bottom ribbon counts documents: "Reading documents · 2 processing ·
  1 queued · 0 of 3 documents read".
- When a pass finishes, the whole profile is fetched again and re-rendered.
  Values change without a sign. Nothing says what changed, where, or why.
- On a 1536 × 864 laptop the Project card is 470 px tall, because each row
  is two lines (the provenance marks sit under the label), and the 100 px
  ribbon covers what is left. No Scope and works row is visible on arrival.
  At 1920 × 1080, two and a half rows are. (Petersham, dev server,
  10 November 2026.)
- When the run ends, the user has to work out what to do next and where the
  update matters elsewhere: Procurement, Cost Plan, Program, RFP drafts.

## 2. What a run is, and what can be shown truthfully

Update queues each unread document. The worker reads each one in two
passes, label then evidence, sixteen passages at a time (eight before 11 Nov
2026, `docs/evidence/2026-11-11-profile-update-speed.md`), one Jev fan-out per
passage. Each passage's reading is saved as it lands. At the end of each
pass the document's facts are committed and the profile is rebuilt in code,
which writes a `profile` event with a new revision.

| Real signal | Where it comes from today | What it can drive |
|---|---|---|
| Queued, reading, done, failed | `job` events and the profile's counts | One sheet token per document in the strip |
| Passages read (labelled and evidence, of units) | `coverage[]` in the profile response | Real progress per document |
| A passage's finds: question keys, systems, excerpt | Saved per passage; not yet readable as a feed | "Spotted" glints on fields and scope rows |
| Label pass committed | `profile` event, next revision | Card values and the summary ink in |
| Evidence pass committed | `profile` event, next revision | Scope cells fill or rub out |
| Run finished | no pending documents | The receipt and the next step |

Highlighting each attribute in turn, so the user sees Jev working through
them, is right in spirit and wrong as a timed sweep. Jev does not work field
by field: each passage is asked its questions in one fan-out. A sweep on a
timer would run the same whatever the documents said, and the activity
contract rules it out ("do not infer progress from time elapsed"). The
truthful version is better: a field lights up at the moment a passage speaks
to it, and inks in when the reading is committed. The rhythm is the real
reading speed, and the light lands only on the fields the document covers.

## 3. Three verbs, one meaning each

Like Trace's hand marks, each motion means one thing.

1. **Spotted.** A passage just said something about this field or system.
   A short cyan glint (cyan is work in progress) on its source mark, and the
   source count ticks up. It is not a decision. At most one glint per field
   every 600 ms.
2. **Inked.** A value changed at a commit. The value writes in left to right
   (about 260 ms) and its provenance mark draws in under it. A replaced value
   takes one pencil strike, then fades. A scope cell shades in with section
   hatch, then fills; a cell leaving scope rubs out. Counts roll ("5 / 8" to
   "6 / 8").
3. **Held.** The user's word does not move. While everything around it inks,
   a value the user set stays still. If a document now disagrees, the squared
   highlighter appears beside it (a decision to make), with the document's
   value and a choice to keep or use it. The reading works around the user.

After the run each changed value keeps a **changed mark** until the user has
seen it: a small graphite revision triangle in the margin, as a drawing marks
a revised detail. (Not a tick: a tick already means "from the documents".)
Hovering it says what it was and what changed it: "was 4 · now 5 · CC-A-101
rev B, p. 2". In a one-line row, a held or disagreeing value shows the
highlighter; the other source's value is in its tooltip and sources pop-up.

Rules:

- Every motion has a cause: a job state, a finished passage or a new
  revision. No timers, and no sweeps over fields nothing touched.
- Completion and failure never wait for an animation.
- Stagger only orders the real changes in one revision: the card by column
  then row, scope in building order then left to right. Under 900 ms in all.
  More than 40 changes in one revision ink together at once.
- Changes off screen do not animate. They keep the changed mark, and the
  strip says "4 changes below".
- A hidden tab animates nothing. The marks and the receipt wait for the user.
- Reduced motion: values and marks appear at once; changed marks and the
  receipt stay.
- The user's own edits keep the existing saved feedback. They are never
  spotted or inked.
- Focus never moves. A reading that lands on a field being edited waits
  beside it ("Read: 5 · Use") until the user leaves the field.
- One aggregate live announcement per stage (document started, pass
  committed with its counts, run finished), never one per glint.
- Motion animates the existing static marks with CSS (clip-path, background
  size, opacity). Nothing is drawn at runtime. Marks keep their print and
  high-contrast rules.

## 4. The focal moment: the sentence writes itself

The summary sentence is the one place the motion is authored. As each label
pass commits, its words write themselves in: "A new build: a residential and
retail building, of five storeys and 781 m² …". Each new word writes in with
its pencil underline; a changed word is struck and rewritten. By the end of a
run the user has watched the project describe itself, and every underline
still says where its word came from. Everything else on the page stays quiet
and quick.

## 5. Layout: scope on the first screen

Target: on a 1536 × 864 laptop with Petersham's profile, at least five Scope
and works rows are visible on arrival, above the resting strip.

1. **One-line title block rows.** Label, value, then the provenance cluster
   (dot, count, reset) at the end of the same line. Rows drop from 48 px to
   about 30 px. Columns, field order and RFP page-one positions stay.
2. **Counts on one row.** Bedrooms, bathrooms and car parks (and the unit
   mix) share one Counts row in The building: "2 bed · 31 bath · 9 car", each
   value still its own editable cell. The building then has seven rows like
   the other columns, and the card drops from 470 px to about 250.
3. **Update into the header band.** DESIGN.md already puts page actions on
   the butter paper. Update and its three dots move there and free a 72 px
   row.
4. **View more** becomes a small chevron on the card's bottom rule instead of
   its own line.
5. **The strip rests as one line**: updated time, documents read, Source
   requirements and "to check". 32 px instead of 100. It opens only for a
   run, its receipt or a failure.
6. Scope rows from 41 px to 34 px.

Estimate: the Scope and works header lands near 520 px on 1536 × 864, with
about seven rows visible. At 1920 × 1080, nearly all fourteen.

## 6. A run, step by step

| Step | Real signal | What the user sees |
|---|---|---|
| Before | unread count and kinds | Update carries "3 new". Hover: "Reads 3 documents: design statement, specification, geotechnical report. 45 others are filed but not read for the profile, mostly drawings." |
| Request | the POST returns | The strip opens (240 ms) with one sheet token per document, queued (the muted clock). |
| Label pass | job leased; labelled of units | The token turns cyan and fills as passages are read: "Design statement · 41 of 96 passages". Fields glint as passages speak to them; the latest find reads beside it: "Storeys · 'five storey mixed use development' · p. 3". |
| Label commit | `profile` event | Card values ink in, the sentence rewrites itself, the token takes its first tick. |
| Evidence pass | job leased; evidence of units | Scope rows whose systems the current passages are about carry the small cyan ring; cells glint where evidence lands. |
| Evidence commit | `profile` event | Cells shade in or rub out in building order, counts roll, a disagreement turns amber. The token takes its second tick. |
| Done | no pending documents | The receipt, then the next step. |
| Away | the user left Profile | The run carries on. The Project Profile nav item carries the cyan ring, then a small count of changes. The browser tab reads "Reading 2 of 3 · Petersham". |
| Failure | failed count, 402 | The strip leads with the problem and Retry. What was read stays inked, because it is real. |

**The receipt.** "Read 3 documents, 294 passages, in 1 min 12 s. 11 values
filled · 1 changed · 9 systems added · 2 removed · 2 to check." Every phrase
is a button that brings those items into view and marks them for a moment.
Counts and times are measured, never estimated.

## 7. Anticipate what comes next

All of this is code over saved data. No Jev call, and nothing is shown that
cannot be counted.

**On arrival**, the strip offers one next step, the first that applies:

1. Reading failed: Retry.
2. Values to check: "3 to check" (exists today).
3. New documents: "3 new documents · Update".
4. Facts every RFP asks for are unknown: "Zone and Lot and DP are unknown.
   Every RFP's page one asks for them", with the fields right there.
5. Work elsewhere is out of date (below).
6. Otherwise nothing. A calm page is the reward for a finished one.

**While Jev reads**, the strip offers the key fields that are still unknown
and that documents seldom settle (budget, procurement route, program
milestones), so the wait becomes the time to state them. They are the user's
word, and nothing read later moves them.

**After the run: what this changes elsewhere.**

- Procurement: disciplines the new scope implies that are not packages yet.
  The table already proposes them.
- Cost Plan: systems now in scope with no cost line.
- Program: a review waiting to adopt or keep. Changes made elsewhere already
  arrive as a review.
- RFP drafts: drafts built from an older profile. Reports already record the
  profile revision they read and list what is stale.

Each is a count and a link that opens that surface at those items. The left
nav carries the same small counts until the surface is visited.

## 8. Data

Already there, with no backend change: job events, the profile's counts,
passage counts per document (`coverage`), the revision on every profile
event, and the previous and next profile in the browser. Those are enough to
compute every inked change, the receipt and the changed marks.

Needed:

1. **Passage feed** (slice 4). Passages finished since a cursor, for one
   project: document, page, question keys, systems and a short excerpt.
   These are saved per passage already; reading them as a feed may need a
   finished-at column. Polled each second, only while a document is being
   read. Budget p50 ≤ 50 ms, p90 ≤ 150 ms (the profile read budget).
2. **Light progress read** (slice 3, optional). Document states and passage
   counts without the ~200 KB profile. Today every job and profile event,
   plus a 2 s poll, fetches the whole profile. The full profile would then be
   fetched only when the revision changes.
3. **Next counts** (slice 5): procurement suggestions without a package,
   in-scope systems without a cost line, a pending program review, stale RFP
   drafts. Budget p50 ≤ 100 ms, p90 ≤ 250 ms.

Filing is untouched.

## 9. The same grammar elsewhere, later

- Filing: a dropped file's row appears queued at once; title, number and
  revision ink in when it files; the row slides into its sorted place; a
  drawing set unfolds into its sheets.
- Invoices: fields ink in; the code checks (duplicate, arithmetic, GST, ABN)
  tick down a short list; the ledger total rolls.
- Proposals: the firm's price column fills down the RFP's own rows; flagged
  cells take the highlighter; the receipt says "12 of 14 rows priced · 2 need
  a look".
- Specification import: coverage cells shade in as candidates are read.
- RFP Update: changed paragraphs ink in, with a receipt.
- Program: on first generation only, bars draw in left to right in logic
  order.

## 10. Slices

| # | Slice | Backend | Done when |
|---|---|---|---|
| 1 | Layout (section 5) | none | Five or more scope rows visible at 1536 × 864 on Petersham; no change to fields, order or provenance; the 390 px layout still stacks. |
| 2 | Inked, held, changed marks, rolling counts, the sentence, the receipt | none | A two-revision Playwright fixture: every changed field marked, receipt counts match the diff, reduced motion shows marks without motion, focus never moves. |
| 3 | The run strip: sheet tokens, passage counts, nav ring and count, tab title | optional progress read | Tokens follow job states, counts follow coverage, no ring after completion, one aggregate announcement per stage. |
| 4 | Spotted: field and row glints, latest find, register row glint | passage feed | Every glint maps to a saved passage reading; none when nothing is read. |
| 5 | Next: arrival step, while-reading fields, what this changes elsewhere, nav counts | next counts | Each count equals that surface's own count; each link lands on those items. |

Budgets: no Jev call added; `project_profile_read` and `profile_rebuild`
unchanged; the new reads as in section 8. In the browser, marks start within
one frame of the data, animation stays under 900 ms per revision, and no
main-thread task runs over 50 ms during a run.

Out of scope: what Jev reads and how values are decided; starting reading on
upload (filing and profiling stay apart); sound; time estimates.

## 11. Owner decisions (answered 10 November 2026)

1. Update and its menu move into the header band. Yes.
2. The Counts row (bedrooms, bathrooms, car parks) appears on Profile and on
   RFP page one, so a fact keeps one place on both. Yes. Card rows are one
   line: label, value, then the dot and source count.
3. Summary underlines follow the 8 November origin rule: green from the
   documents (dotted where Jev leans, as the card's ring), blue typical, rust
   set by you, amber fill to check. Yes.
4. No time estimate; the strip shows passages remaining.
5. "Changed since you looked" is remembered per browser (left to the
   implementer; no server change, can move to per user later).

Implementation plan: `docs/plans/2026-11-10-run-time-feedback-implementation.md`.
