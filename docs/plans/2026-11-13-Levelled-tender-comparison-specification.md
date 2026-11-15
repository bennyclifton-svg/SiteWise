# Levelled tender comparison: specification

Nov 13, 2026 · @Ben clifton

## Outcome, lane and budgets

A project manager compares consultants, trades and head contractors on one levelled table and has no reason to rebuild it in Excel. Each firm shows a quoted total and a levelled total, and every dollar between the two is a listed, sourced adjustment. The generated table is a first draft, so every value in it can be corrected in place in one gesture.

- **Lane:** A for the comparison (price arithmetic, award amounts). C for the LLM provider, because quote text leaves the host.
- **Serves:** time-to-decision and answer trust.
- **Read and write budgets are unchanged:** price table 150/300 ms, cell and adjustment writes 100/250 ms, award 200/500 ms (p50/p90). No model call on any read or write.
- **Background reading:** the LLM pass runs once per quote after the existing read. Proposed budget p90 30 s, to be measured on real quotes before it is fixed.
- **Basis:** written from the repository at `main` on 13 November 2026, which builds and passes its proposal and comparison tests unchanged.

## What the code does today

The price table has four separate ways of computing a firm's total, and none of them levels. `packageColumns` in `internal/store/comparison.go` picks one per package.

| Path | Used for | Rows compared | Total | What is missing |
| --- | --- | --- | --- | --- |
| Schedule (`proposals.Totals`) | Trades; consultants who answer on the response form | The RFP or RFT rows | Sum of base cells; unknown if any base row is not priced or excluded | No adjustment for a missing row, so the firm has no total and is skipped by `MarkLowest` |
| Quoted fee (`appointmentRows`, `ReadOffer`) | Consultants who quote in their own format | The firm's own fee lines; stage mapping is a suggestion | Included lines, or the confirmed appointment fee | Firms sit on different rows; a bundled fee cannot be compared row by row |
| Builder (`ensureBuilderSchedule`, `ReviewBuilderQuote`) | Head contract | One row, "Quoted construction price" | The printed total | The breakdown is display-only and `checkBuilderAlignment` keeps it out of arithmetic |
| Group (`GroupTotals`) | Combined engagements | The members' fee schedule | Group totals | Same gap as the schedule path |

Exclusions and departures are collected by heading (`departureHeading`) into the R6 criterion. They are not tied to a row or an amount.

## The price schedule each quote maps to

Yes: the existing RFP and RFT scopes are the price schedules, and every quote is mapped to its package's own rows. `rfp.Schedule` already turns the resolved RFP or RFT into `schedule_items`, and `syncRFPSchedule` keeps them in step with the current draft. Head contracts are the one exception today.

| Package | Rows today | Rows under this spec |
| --- | --- | --- |
| Consultant | The RFP's stages (or elements), each option under its stage, reimbursables, hourly-rate roles, non-price criteria | Unchanged |
| Trade | The RFT's rows: design and shop drawings, preliminaries, then the systems that trade builds; provisional sums and prime cost items; rates; criteria | Unchanged |
| Head contract | One lump-sum row. The Main Works RFT prints system rows (preliminaries, demolition, site works, substructure, structure and so on) but the comparison does not use them | The Main Works RFT rows, with the firm's printed total kept as the control figure |

Rules that follow:

- **The RFP or RFT owns the rows.** Scope edits in the RFP editor change the rows at the next sync, as now. Comparison labels and custom rows stay as user overrides and never edit an issued document.
- **Each row's scope lines are its definition.** The stage text and its scope lines, including lines the PM added, are what a mapping is judged against.
- **Carve-outs hold.** A Main Works row whose trade is let as a separate package stays excluded and is not levelled.
- **No RFP, fewer rows.** A package with no RFP keeps the discipline default or the generic lump sum. A one-row schedule cannot be levelled by row, and the screen says so.
- **Unasked items get their own rows.** What a firm offers outside the schedule is listed separately and stays out of totals until the PM adds a custom row for it.

## One comparison model

All three kinds of package use one model: a firm's offer is stored as lines exactly as printed, and each line is mapped to a schedule row. The four total calculations become one.

1. **Offer lines.** Every priced line in the quote: label, amount, page, its parent line if it has one, and its role (base, option, allowance, rate, subtotal, total, not a price). Stored once and never changed by mapping. The quoted-fee rows and the builder review sections become views over these lines.
2. **Mapping.** Each base line goes to one row, to a span of rows (a bundle), or outside the schedule. A row's state for that firm is then priced, included in another row, excluded by the firm, not priced, or unread.
3. **Qualifications.** Each exclusion, assumption, departure or allowance is a passage quoted from the quote and pinned to a row, or to the whole offer.

Invariants, all checked in Go:

- **Each line counts once.** A parent and its children never both enter a total.
- **Lines reconcile to the printed total.** Mapped and unmapped base lines add to the firm's printed total to the cent, or the column shows the gap.
- **A bundle is never split by guess.** Its price sits on one row and the others read "included in" that row. The table adds a subtotal over the bundled span so every firm is compared on it.
- **Equal amounts never establish equal scope.** This existing rule stays.
- **The user's mapping is final.** A later read keeps it, and every move has undo, as `AlignSubmission` does now.

## Levelling adjustments and the levelled total

A levelling adjustment is an amount added to, or taken off, one firm on one row, for comparison only. It never changes a quoted price or an award amount.

**When one is offered**

- A row this firm did not price or excluded, which other firms priced.
- A qualification that moves cost, such as "excludes rock excavation".
- An allowance that differs from the amount the RFT told tenderers to carry.
- An option the PM wants in the base for every firm.

**How the amount is set**

| Method | Amount |
| --- | --- |
| PM's figure | Typed by the PM |
| Highest other tender | The highest price any other firm gave for that row |
| Median of other tenders | The median of the other firms' prices for that row |
| Cost plan | The budget on the cost plan line for that row, when one exists |

Code proposes amounts; nothing is applied until the PM accepts it. Each adjustment records its amount, method, reason, who set it and when, and the qualification it answers. A firm's clarified price is not an adjustment: it is a cell correction with a note, through the existing `SetSubmissionCell`.

**Arithmetic, in Go**

- **Quoted total:** as computed today.
- **Levelled total:** the known base plus accepted adjustments. It shows only when every base row is priced, included, or adjusted. Until then the column reads "Not yet levelled" with the count of open rows.
- **Lowest levelled:** marked among fully levelled columns only, beside the existing "lowest complete price".
- **GST:** every figure ex GST, as now.
- **Rates and options:** compared on their own rows and outside the levelled total.

**Award**

Award writes quoted amounts, exactly as today. The award screen lists any adjustment still open on the chosen firm, so each is settled as a clarified price or carried as a provisional sum before contract.

## Reading offers: rules, Jev, then the LLM

Rules and Jev keep every job they do now; the LLM takes only what neither can settle, and it never supplies a number.

| Job | Who | Result |
| --- | --- | --- |
| Firm, package, GST basis, printed total | Rules, then Jev | Unchanged |
| Fee lines that add to a printed total | Code (`ReadOffer`) | Unchanged |
| Line structure of a breakdown: which amounts are parents, children, subtotals, or allowances inside the price | LLM | A role and parent for each harvested amount |
| Line to row | Word match, then Jev's coverage reading, then the LLM for lines still unmapped | A suggestion marked "check mapping" |
| Qualification to row | LLM | A quoted passage pinned to a row, as a suggestion |
| Clarification letter to a firm | LLM | A draft the PM edits |

Guardrails:

- **It picks, it does not type amounts.** Code harvests every amount with an id, as it does for Jev. The LLM answers with ids. An id that was not sent is rejected.
- **Structure must reconcile.** A proposed line structure is kept only if its base lines add to the printed total to the cent. Otherwise it is discarded and today's view stands.
- **Quotes must be literal.** A qualification's text has to be a substring of a stored passage.
- **Background only.** One job per quote after the existing read. Results are stored; no screen read or write calls the model.
- **Optional.** With no `SITEWISE_LLM_API_KEY` the server starts and levelling works by hand on rule and Jev readings.
- **Small client.** `internal/llm` mirrors the Jev client: pinned model version, one deadline, circuit breaker, latency and tokens logged, recorded responses replayed in tests.
- **Org opt-in.** Quote text goes to a third party, so each org turns the pass on.

Proposed replacement for `AGENTS.md` rule 1: "Jev makes every judgement on a path a user waits on. One LLM, in background jobs only, may propose line structure, row mappings and qualifications for quotes, and draft text. It never supplies a number, and nothing it proposes is applied without the code checks above and the user's acceptance."

## Screen, report, export and award

One price table serves consultants, trades and head contractors: schedule rows down the side, firms across, and three total lines under each firm.

- **Cells.** Each shows the amount or state, with small markers for a qualification, an adjustment, or a mapping to check.
- **Totals.** Quoted total, adjustments, levelled total. A printed-total gap stays visible, as now.
- **Cell panel.** Opening a cell shows its source lines with page links, the qualifications pinned to it, and the adjustment editor with the proposed amounts.
- **Bundles.** A subtotal line appears over any span a firm has bundled.
- **Outside the schedule.** Items a firm offered unasked are listed under its column.
- **Clarifications.** A per-firm list of open rows and qualifications, with the draft letter when the LLM pass is on.
- **Report.** The A4 comparison report adds the levelled totals and a schedule of adjustments with their reasons.
- **Export.** The Excel export writes the levelled total as a formula over the quoted rows and the adjustment rows, so the workbook can be checked and extended. CSV carries the same rows.
- **Award.** Amounts written to the cost plan are unchanged. Open adjustments on the chosen firm are listed before confirmation.

Edits lock after award and are scoped to org and project, as the existing comparison edits are.

## Editing the comparison table

The generated comparison is a first draft: the PM corrects any part of it in place, in one gesture, and can undo it. The table takes the gestures the Register, Program and Cost Plan already share in `web/src/table`, which the price table does not use today.

| Today | Replaced by |
| --- | --- |
| An arrow button on each fee, moving one fee above or below the line per click (`ConsultantComparison.tsx`) | Drag, one key, or one menu action, on a single fee or a selection |
| A three-step move: click the icon, choose "Move here", confirm in a bar (`Prices.tsx`) | Drag the fee onto a row; dropping on a priced cell combines |
| Separate consultant, builder and schedule tables | One table component for every package kind |
| One level of undo per firm | Undo and redo for every edit |
| Fee detail inside a disclosure in each cell | A side panel for the selected cell |

In the quoted-fee view each firm's fee lines get their own rows unless a line maps to exactly one stage. Firms rarely line up, and a moved fee jumps to another part of the table. Mapping every line to schedule rows removes that.

**Gestures**

- **Select.** Click a cell; Ctrl/Cmd toggles, Shift extends, arrow keys move.
- **Edit an amount.** Type, double-click or press F2. Enter saves and moves down, Tab moves right, Escape cancels.
- **Set a state.** On the selection: I for included, X for excluded, N for not priced, Delete to return to the reading.
- **Move a fee.** Drag it within the firm's column. Dropping on an empty row moves it; dropping on a priced cell combines, with the sum previewed before the drop. A fee never moves between firms.
- **Above or below the line.** The table has three bands: Base, Options, Not counted. Drag a fee or a selection between bands, or press Alt+Up or Alt+Down. Dragging a row's handle moves the row for every firm.
- **Rows.** Add a row, rename it by double-clicking its label, reorder by its handle, remove a custom row.
- **Right-click.** Every action above, plus add an adjustment by method, open the source page, and confirm a reading.
- **Paste.** A column of figures copied from Excel pastes down a firm's column from the selected cell. Copy works the other way.
- **Bulk.** "Level open rows" applies one method to every open row of a firm in one step. "Confirm readings" works on a selection.

Every drag has a keyboard and menu equivalent.

**Behaviour**

- **One gesture, one save.** A selection or a paste is one batch write in one transaction, inside the 100/250 ms write budget.
- **Instant cell, Go totals.** The typed value shows at once. Totals come back from Go in the same response; the browser does no arithmetic.
- **Nothing is lost.** A failed save keeps the draft in the cell to retry.
- **The user's value is final.** A re-read never overwrites it, and the machine reading is kept so it can be restored.
- **Amber means check.** Editing or confirming a cell clears it. Nothing has to be confirmed before it can be edited.
- **Undo and redo.** Ctrl+Z and Ctrl+Shift+Z step through every comparison edit on the package, across reloads, until award.
- **Narrow panes.** Firm columns scroll under a fixed row-label column.

Open question: which failures the owner has seen with the current arrows and move bar, so each becomes a regression case.

## Data changes

Four new tables are added and no existing table changes meaning. `submission_prices` stays the row cell of record, so award, cost lines and invoices keep pointing at what they point at now.

| Table | One row is | Key columns |
| --- | --- | --- |
| `submission_adjustments` | One accepted adjustment for one firm on one row | submission, item key, amount in cents ex GST, method, reason, qualification, actor, version |
| `submission_lines` | One priced line as printed in a quote | submission, ordinal, parent line, label, amount, role, page, passage, band, decided by |
| `submission_qualifications` | One exclusion, assumption, departure or allowance | submission, item key or whole offer, kind, quoted text, page, band, decided by, state (open, adjusted, clarified, dismissed) |

- Every row carries `org_id` and composite foreign keys, and the isolation test covers each new table.
- Adjustments are never deleted by a re-read; a retired row keeps its adjustment visible, as retired schedule rows keep their prices.
- `submission_lines` is filled from the stored quoted fees and passages without re-reading any document, the way `ReadOffer` was introduced.
- `decided_by` gains the value `llm` on lines and qualifications, so every field still states how it was decided.

The fourth table, `comparison_edits`, is an append-only log of every comparison edit on a package: who, when, and the values before and after. Undo and redo read it, and it replaces the one-level `comparison_alignment` snapshot.

## Slices, tests and done

Five slices, each shippable on its own; the first two need no LLM and no new source documents.

| Slice | What ships | LLM | Needs real quotes |
| --- | --- | --- | --- |
| 1. Editable table and adjustments | The one editable table for every package kind (shared grid gestures, drag, paste, undo), with levelling adjustments, proposed amounts and the levelled total on existing rows, in the table, report and export | No | No |
| 2. Head contract rows | Head-contract packages compare on the Main Works RFT rows; the printed total is the control figure; the builder review lines sit beside the rows | No | No |
| 3. Offer lines | One stored line model behind the quoted-fee and builder views; bundle subtotals; one total calculation | No | Yes, as regression keys |
| 4. LLM reading | `internal/llm`, line structure and mapping proposals, behind the key and the org opt-in | Yes | Yes |
| 5. Qualifications | Qualifications pinned to rows, adjustment proposals from them, clarification drafts | Yes | Yes |

Tests and gates:

- **Arithmetic.** Unit tests for the levelled total: missing row, excluded row, bundle span, option moved to base, deduction, unknown amounts.
- **Award unchanged.** A regression proving an adjustment never alters an award amount or a cost line.
- **Existing keys.** The Hale, Kavanagh, Walsh and synthetic proposal tests pass untouched through slices 1 to 3.
- **New keys.** One answer key per real tender set, scored separately for line capture, mapping accuracy and qualification capture. Thresholds are not lowered to reach a figure.
- **Isolation.** Wrong-org reads and writes fail on each new table and endpoint.
- **Speed.** The price table and write benchmarks hold their current budgets with five firms and the full Main Works schedule.
- **Browser.** One Playwright flow per package kind: drop three quotes, correct them by keyboard, drag and paste, undo, level, export, award.

Done means a PM levels a three-firm tender of each kind inside SiteWise, correcting the generated table by hand without leaving it, the exported workbook matches the screen to the cent, and no adjustment has changed an award.

## Decisions needed from the owner

Four choices change the code, and each has a recommended answer.

| Decision | Recommended | Alternative |
| --- | --- | --- |
| Head-contract rows | Use the Main Works RFT rows, printed total as the control figure | Keep the single lump-sum row and level only on custom rows |
| Default proposed amount for a missing row | Highest other tender, the conservative choice | Median of other tenders, or the cost plan budget |
| LLM provider and model | Open: owner's choice. The client is provider-neutral and pins one model version | None |
| `AGENTS.md` rule 1 | Replace with the wording under Reading offers | Keep Jev-only and stop after slice 3 |

Slices 1 and 2 depend only on the first two rows.

## Source documents needed

Real quotes are needed from slice 3 onward; slices 1 and 2 need none. The repository's fixtures are consultant fee proposals, one builder cost-advice email, and no trade tenders.

For each tender set:

- The RFP or RFT as issued, or the SiteWise package it came from.
- Every firm's submission as received, PDF or Excel.
- The comparison the PM built by hand in Excel. This is the answer key.
- Any clarification letters sent to tenderers, and their replies.

Sets wanted, in priority order:

1. **Head contract:** two or three builders quoting the same job, with trade breakdowns.
2. **Trade:** two or three tenders for the same package, for two different trades.
3. **Consultant:** one set where firms bundled stages differently and the PM levelled them.

Private quotes stay out of the repository, as the four private PDFs did for the alignment work. Fixtures committed to the repo are synthetic rewrites that keep the structure and change the names and figures.

## Out of scope

- Choosing or recommending a firm automatically.
- Splitting a bundled price across rows by estimate.
- Non-price scoring; the PM's ratings stay as they are.
- Checking quantities or rate build-ups inside a bill of quantities.
- Any model call while a user waits, a chat surface, or embeddings.
- Changing RFP or RFT wording from the comparison.
- Quotes with no text layer, beyond the existing OCR path.
- Variations and post-award price changes.
