# Procurement from fee proposal to award: implementation report

Date: 16 October 2026. Lane A. Task: `docs/plans/2026-10-16-procurement-process-to-award`
(the owner's plan); detailed plan, decisions taken while the owner was away
and the task list: `docs/plans/2026-10-16-procurement-to-award-implementation.md`.
Built on branch `procurement/award`; nothing here is pushed, and nothing is
marked reviewed.

## What a user sees

**Procurement.** Each package row shows its firms as chips (name, status,
total) in place of the Firm 1–3 columns. Dropping proposals on a row files
them in the register and opens one submission each; the chip says Reading…
and fills in about a second. An unassigned banner holds proposals filed from
the register that no rule could place. The row's button reads **Review** with
one firm, **Compare** with several, **Prices** once awarded.

**Price table.** One column per firm, one row per item of the RFP's own
returnable schedule (stages, then options, reimbursables and rates), totals
computed in Go, lowest base total shaded. A cell shows the figure, or
Included / Not priced / Excluded / Rates only; only cells that need an eye
carry a chip (Check · Jev, Not read); the tooltip says how the value was
decided and the printed line it came from. Click a cell to correct it; a
user's value is final. Rows no firm priced collapse into one line. Firm names
open a side panel with the PDF beside the firm's details. Flags (stated
total differs from the items, GST not stated or normalised from inclusive,
rates only, options A and B, no readable text) sit under each column; an
earlier proposal the same firm replaced is listed under the table. Export
writes the same layout to Excel.

**Award.** Two steps: choose and combine rows (one cost line per combined
group, named by the PM), then preview the cost-plan lines and the total the
server computed. Commit turns the package's fee line into a group that keeps
its budget, writes one child line per award line carrying the firm's contract
value, records which schedule items each child holds, sets the package
appointed with the firm's ABN, and marks the other firms unsuccessful. Undo
restores the line exactly; it is refused once an invoice is allocated to a
child. Invoices from that ABN then settle the package by rule and offer only
the awarded child lines.

**Comparison report.** One A4 landscape page: price summary by stage,
non-price criteria (the passages Jev picked from each proposal, rated by the
PM: strong / acceptable / concern), departures and exclusions, a
recommendation drafted by code from the computed facts and editable by the
PM, and sign-off. It prints on white paper whatever the screen theme.

**RFP.** The fee table is printed from the same schedule rows the price table
uses; returnables print as a table; Reports offers the **Response form**
workbook (firm details, then one row per schedule item with its key), which
the reader settles row by row by rule when a firm returns it.

## What calls what

```
POST /procurement/packages/{pkg}/submissions?name=
  → intake Upload (unchanged filing call, unchanged recordings)
  → store.OpenSubmission (package = the row dropped on; "reading"; event)
  → Runner.AfterFile → proposalReader.Start (own slots)
      identity.Extract (≤ 60 pages, 20,000 runs, 12 sheets)
      → proposals.Harvest (money per cell with row label and column heading,
        GST pairs, unit rates, sections, firm candidates, form cells)
      → rules: kind (strong cue + discipline), package, firm (directory ABN /
        name, or a printed name corroborated by filename or email domain),
        GST basis, stated total, response-form rows
      → ONE Jev fan-out (≤ 90 KB) for what rules leave open: kind, firm
        fields, every price and rate mapping, each criterion's passages
      → code: ex GST to the cent, other items, sum gap, departures
      → store.CommitSubmissionRead (never over a user's decision; supersede
        the same firm's earlier submission; event) → SSE wakes the page
GET  .../prices         store.ReadPriceTable → proposals.Totals in Go
PUT  .../cells/{key}    a user's cell → recomputed table
POST .../award/preview  store.PreviewAward (no write)
POST .../award          store.Award: one transaction (group, children,
                        cost_line_sources, package, statuses, awards row)
DELETE .../award        store.UndoAward (ErrAwardLocked once invoiced)
GET|PUT .../report      store.ReadComparisonReport, proposals.DraftRecommendation
serve: RunProposalRechecks every 15 s re-asks grey reads (at most 3 tries).
```

Data: migration `032_procurement_award.sql` adds `schedule_items`, `firms`
(org-wide directory), `submissions`, `submission_decisions`,
`submission_passages`, `submission_prices`, `submission_responses`,
`comparison_reports`, `awards`, `cost_line_sources`, and lets a non-posting
group line hold a budget (and only a budget). Every key is composite with
`org_id`; the foreign-key sweep covers the new tables. A deleted register
document leaves its submission (document set null).

What can fail, and what then happens:

- Jev times out or errors: the open fields go grey, rules' values stand, the
  recheck sweep retries at background priority; the column says Not read.
- A proposal with no text layer (scanned): flagged "No readable text: type
  the prices from the file"; nothing is invented.
- A firm cannot be settled: a draft firm (amber) from the picked name.
- One printed lump sum against a multi-stage schedule: stage answers are
  never green (see Accuracy).
- A wrong total on award: refused (422); the client must send the total the
  preview computed.
- Another org: every new route answers 404, as for a missing record
  (`internal/httpapi/procurement_award_test.go`, 13 routes).

## Accuracy (private answer key, aggregate only)

`cmd/proposal-eval` replays a live run over 42 real fee proposals (Hale and
Yarramundi; town planning, traffic, civil, structural, waste, services and
others), 461 labels, built by the agent from the documents and **not
reviewed**. Thresholds were fitted on a calibration split and scored on the
held-out split; green uses a strict fit (no calibration error at the cut).

| Field | Scored | Correct | Wrong | Blank | Green wrong |
|---|---|---|---|---|---|
| Firm (rule) | 72 | 42 | 0 | 30 | 0 |
| GST basis | 33 | 27 | 0 | 6 | 0 |
| Kind | 42 | 22 | 0 | 20 | 0 |
| Prices | 348 | 288 | 20 (all amber) | 40 | 0 |
| Rates | 113 | 94 | 8 (all amber) | 11 | 0 |
| Stated total | 29 | 15 | 0 | 14 | 0 |

Jev's raw pick before banding: prices 321/348 (92 %), rates 103/113 (91 %).
**No wrong value is shown green.** One held-out price did go green at 0.96
on the refit (a single $1,800 "full plan" fee read as one stage not priced);
the fix is structural, not a threshold: when a proposal prints one lump sum
against several priced stages, no stage answer can be green
(`lumpSum` in `internal/proposals/read.go`, test
`TestOneLumpSumAgainstStagesStaysAmber`). Most families other than prices
have fewer than 20 calibration answers per option-count shape, so they stay
amber or blank until more proposals are labelled. `tools/check.ps1` now runs
the replay; a replay miss or any false green fails the build.

Recorded proposal call latency (42 live calls): p50 305 ms, p90 400 ms.

## Speed

`cmd/intake-bench`, dev machine (Windows), replayed provider latency, full
run, exit 0 (every budget passes). Proposed budgets need owner sign-off.

| Path | p50 / p90 ms | Budget p50 / p90 ms |
|---|---|---|
| `proposal_read` (upload to read, 40 proposals) | 617 / 899 | 3,000 / 6,000 (proposed) |
| `price_table` (five firms) | 5.3 / 5.9 | 150 / 300 (proposed) |
| `submission_write` (cell, table returned) | 7.0 / 7.6 | 100 / 250 (proposed) |
| `award_write` (award and undo) | 12.8 / 15.1 | 200 / 500 (proposed) |
| `procurement_report` | 7.2 / 7.9 | 150 / 300 (proposed) |
| `whole_intake` (filing, unchanged) | 413 / 1,252 | 1,000 / 2,000 |
| `report_setting` (RFP setting click) | 95.6 / 106.8 | 100 / 250 |

`report_setting` regressed when the schedule rows were written on every RFP
write (100.7 and 105.7 ms p50 in two runs, against 93.5 ms on the pre-feature
commit on the same machine). The rows are now synced from the current RFP
draft where they are used (price table, proposal read, added firm), and the
setting click is back at its baseline. Two existing paths miss by a few
milliseconds in some runs on this machine with or without this work:
`invoice_write` p50 53.4 ms on the pre-feature commit (budget 50) and
`profile_edit` p90 once at 155 ms (budget 150).

`proposal_read` is upload to the column's read event: the unchanged filing
call (served a recorded filing's latency, because bench-made files have no
recording of their own) plus the proposal fan-out at recorded latency. The
replayed provider latency is a timing model, not release evidence; the
release gate is the same bench with `-live` on the VPS.

## Tests

- Go: `internal/rfp` (schedule, response form, goldens, page limit),
  `internal/proposals` (harvest, rules, one fan-out, timeout grey, user
  decisions kept, GST inclusive and percentage fees, rates only, directory
  firm, response form by key, lump sum, report validator with planted
  failures), `internal/store` (schedule, submissions, awards incl. undo and
  the invoice lock), `internal/costs` (group budget roll-up), `internal/invoices`
  (awarded lines only), `internal/httpapi` (happy path and cross-org).
- Playwright `procurement-award.spec.ts`: drop three proposals on a row,
  compare, correct a cell, award with a combined line, see the children in
  the cost plan, open and sign the report, undo.
- Full Go suite: passes except `TestProfilePlacesInputsInBuildOrder`, which
  fails identically on the pre-feature commit. `go vet` reports lock copies in
  `internal/jobs/loop.go` (not touched here).
- Full Playwright suite (63): 54 pass. The 8 failures in `intake.spec.ts` and
  `proposals.spec.ts` (implications) fail identically on the pre-feature
  commit; another session has uncommitted edits to those specs in the main
  checkout. `hydraulic-rfp.spec.ts` failed once on a version race and passed
  3 of 3 on repeat.
- `python tools/check_knowledge.py`: 0 errors.

## For the owner

1. **AGENTS.md #1 and Phase 4.** No generative model is wired. Code drafts
   the recommendation from computed facts; `proposals.CheckDraft` (cited ids
   exist, every summary cites, every number matches a figure or passage) is
   ready for a model if you amend non-negotiable #1.
2. **Budgets.** `proposal_read`, `price_table`, `submission_write`,
   `award_write`, `procurement_report` are proposed values.
3. **Thresholds and labels.** `data/proposal/thresholds.json` is the agent's
   fit; the answer key (`data/eval/proposal/private`) is unreviewed.
4. **RFP length.** Returnables as a table bring the longest scenario RFP to
   3.86 pages; it passes the renderer's 4-page gate but runs past O1's three.
5. **Not built.** Trade tenders beyond the generic lump-sum schedule (no RFT
   schedule knowledge yet), firm merge, emailing RFPs, and price escalation.
