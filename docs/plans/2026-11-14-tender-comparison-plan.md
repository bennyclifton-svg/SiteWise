# Tender Comparison Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A project manager drops tenders on a package and sees two tables, "As tendered" (every column equal to the firm's printed tender sum) and "Adds and subs", giving a like-for-like total, without retyping anything.

**Architecture:** One background tender pass per submission: code picks the pages, Claude Haiku 5.5 lists every printed price with a schedule row for each, and Go checks every amount against the page text and reconciles to the printed total. Lines are stored; the existing price cells are a projection of them, so award, the report and the cost plan keep working. Every edit is a named operation that never calls a model.

**Tech Stack:** Go 1.27, PostgreSQL 17, pgx v5, React + TypeScript (Vite, vitest, Playwright), `github.com/anthropics/anthropic-sdk-go/v2` (new), the existing Python render helper pattern (pypdfium2, Pillow).

**Design:** `docs/design/2026-11-14-tender-comparison.md`. Read it first. Section numbers below refer to it.

**Status:** written 14 November 2026 from reading `main`. Nothing in it has been compiled or run, except two things: the Go code quoted in slice 0 (the model client, page selection, request, parsing and the pass) was compiled against SDK v2.9.0 and exercised, and the reconciler in slice 1 tasks 1 to 3 was written and tested before the plan (24 tests, and a replay over the 22 real trial answers). Names marked *verify* were not opened; open them before relying on them.

---

## Feature statement (AGENTS.md run sheet)

- **Outcome:** design section 1.
- **Why:** time-to-decision and answer trust (design section 2).
- **Lane:** A for the reconciler, the cells, award and the cost figures. C for the model client, the org switch, the new dependency and what leaves the server.
- **Out of scope:** design section 18.
- **Budgets:** design section 12. `price_table` and `submission_write` keep their recorded budgets.
- **Done:** design section 22.
- **Edge cases:** design section 13.

## The slices

| Slice | File | Delivers | Needs |
|---|---|---|---|
| 0 | `2026-11-14-tender-comparison-slice0.md` | Model client, page selection, page images, the pass, a command that runs one tender | The key in `.env`, the private corpus |
| 1 | `...-slice1.md` | The reconciler (rules R1 to R9), the answer key, scoring, prompt tuning. **Ends with the go or no-go.** | Slice 0 |
| 2 | `...-slice2.md` | Org switch, tables, the background job, cells projected, the table read | A go from slice 1 |
| 3 | `...-slice3.md` | The first table on screen, read only, with "to check" and "checked" | Slice 2 |
| 4 | `...-slice4.md` | Editing the first table as operations, with undo | Slice 3, and the project change log (below) |
| 5 | `...-slice5.md` | Adds and subs, like-for-like total | Slice 4 |
| 6 to 8 | `...-slices6-8.md` | Head contract rows and award gate; export and report; retiring the old reading | Slice 5 |

Do the slices in order. Stop at the end of each and report (see "Reporting").

**Slice 1 ends with a gate.** If its pass marks are not met, stop and take the result to the owner. Do not start slice 2.

**The change log.** Slice 4 uses `store.WithChange` and `UndoChange` from `docs/plans/2026-11-14-change-log-undo-operations.md` (its tasks 1 to 4). Before starting slice 4, check that `internal/store/change.go` exists on `main`. If it does not, stop and tell the owner; do not build a second undo.

## Read before starting

| What | Where |
|---|---|
| The design | `docs/design/2026-11-14-tender-comparison.md` |
| Rule 1 and its conditions | `AGENTS.md`; `docs/design/2026-11-14-chat-control-and-email-intake.md` section 12 |
| How a proposal is read today | `internal/httpapi/submissions.go` (`proposalReader`), `internal/proposals/read.go` (`Read`), `internal/store/submissions.go` (`SubmissionForReading`, `CommitSubmissionRead`, `replaceMachinePrices`) |
| The price table | `internal/store/comparison.go` (`PriceTable`, `priceTable`, `packageColumns`, `needsReview`), `internal/proposals/compare.go` (`Totals`, `MarkLowest`) |
| The schedule | `internal/rfp/schedule.go`, table `schedule_items` in `internal/db/migrations/032_procurement_award.sql` |
| Award | `internal/store/awards.go` (`awardPlan`) |
| The Jev client, as the pattern for the model client | `internal/jev/client.go`, `internal/jev/admission.go` |
| Page images | `internal/identity/visual.go`, `internal/identity/visual_render.py` |
| Text and OCR | `internal/identity/identity.go` (`Text`, `Run`, `Extract`), `internal/identity/proposal_ocr.go` (`MissingTextPages`), `cmd/sitewise/main.go` lines 188 to 215 |
| An eval command, as the pattern | `cmd/proposal-eval/main.go`, `internal/eval/` |
| A bench, as the pattern | `cmd/intake-bench/award.go`, `bench/budgets.json` |
| Routes | `internal/httpapi/projects.go` (two maps: handler by pattern, budget path by pattern), `internal/httpapi/billing.go` (account routes, `ownerSession`) |
| The screen | `web/src/Prices.tsx`, `web/src/submissionApi.ts`, `web/src/states.css` |
| Browser tests | `web/tests/builder-comparison.spec.ts`, `web/tests/e2eserver/main.go` |
| The trial | `data/eval/tender/private/trial-2026-11-14/` (git-ignored; `trial4.py` holds the prompt, schema, page selection and a first reconciler) |

## Conventions

- **Worktree.** Work in a git worktree made from local `main`, not `origin/main` (local `main` is ahead). Ignored files do not come with a worktree: copy `.env`, `testing/rft/` and `data/eval/tender/private/` into it.
- **Private data.** `testing/rft/` and `data/eval/tender/private/` hold real firms and prices. Never commit them, never paste a firm name or a real price into a tracked file, a test, a commit message or a log line. Tests use invented names and figures. The repository is public.
- **The key.** `SITEWISE_ANTHROPIC_HAIKU_KEY` in `.env`. Never print it, log it or put it in an error.
- **Go tests.** Pure packages: `go test ./internal/tender/... ./internal/llm/... -count=1`. Store and HTTP tests need `SITEWISE_TEST_DATABASE_URL` and run one package at a time: `go test -p 1 ./internal/store -count=1`. Read `docs/developing.md` for the gate that must pass before a merge.
- **Web.** `cd web`, then `npm run build` (type check and build), `npm test` (vitest), `npx playwright test tests/<file>.spec.ts`.
- **Migrations.** Take the next free number in `internal/db/migrations/` at the moment you write the file. Another plan is adding migrations too. Never renumber an applied migration.
- **Money.** Cents as `int64` in Go, `bigint` in SQL. JSON carries exact decimal strings through `invoices.FormatCents`, or null for unknown (`moneyField` in `internal/httpapi/invoices.go`). The browser never adds.
- **Every row carries `org_id`.** Every store query filters on it. Every new route gets a wrong-org test that expects 404.
- **Commits.** One per task, message in the repo's style (`Tender: ...`). Do not push.
- **Words on screen.** Plain words, no jargon. "Tender sum", "Not broken down", "Adds and subs", "Like for like", "To check", "Checked".

## Shared contracts

Every slice uses these. Change one only by changing this file in the same commit.

### C1. Vocabulary

| Term | Meaning |
|---|---|
| Line | One priced line a firm printed, with its page |
| Counted | The line's amount enters the first table |
| Row | A schedule item of the package with `Price()` true and `Base` true, or the system row |
| System row | Schedule item `tender:not_broken_down`, label "Not broken down" |
| Tender sum | The firm's printed price for its base offer, cents ex GST |
| Remainder | Tender sum less the sum of counted lines that sit on a row other than the system row |
| Like-for-like total | Tender sum plus the firm's applied adds and subs |

The system row's cell for a firm is: counted lines with no row, plus whatever is left so that the column equals the tender sum. So the system row's cell = tender sum − (sum of counted lines on real rows).

### C2. Go types (`internal/tender/types.go`)

This is the whole file after its `package tender` line. It is the file the reconciler in slice 1 was tested against.

```go
// Row is one schedule row offered to the model.
type Row struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

// Page is one page of a submission as SiteWise extracted it.
type Page struct {
	Number int
	Text   string
	OCR    bool
}

// RawLine is one line exactly as the model returned it.
type RawLine struct {
	P     int    `json:"p"`
	Label string `json:"label"`
	Amt   string `json:"amt"`
	G     string `json:"g"`
	St    string `json:"st"`
	Kind  string `json:"kind"`
	Where string `json:"where"`
	Group string `json:"group"`
	Row   string `json:"row"`
	Note  string `json:"note"`
	Sure  bool   `json:"sure"`
}

type RawExclusion struct {
	P    int    `json:"p"`
	Text string `json:"text"`
}

// RawReading is the model's whole answer.
type RawReading struct {
	Firm       string         `json:"firm"`
	Basis      string         `json:"basis"`
	GST        string         `json:"gst"`
	Lines      []RawLine      `json:"lines"`
	Exclusions []RawExclusion `json:"exclusions"`
	Notes      []string       `json:"notes"`
}

// Flag is one thing to look at, in plain words written by Go.
type Flag struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// Line is one line after Go has checked it.
type Line struct {
	ID        string
	Ordinal   int
	Page      int
	Label     string
	Printed   string
	Cents     *int64
	GST       string
	State     string
	Kind      string
	Placed    string
	Heading   string
	Row       string
	Note      string
	Counted   bool
	Flags     []Flag
	DecidedBy string
	SplitOf   string
}

// Reading is one submission after reconciling.
type Reading struct {
	Firm       string
	Basis      string
	GST        string
	TenderSum  *int64
	SumHow     string
	Lines      []Line
	Exclusions []RawExclusion
	Notes      []string
	Flags      []Flag
	// Gap is the tender sum less every counted line: what the lines do not
	// explain. Remainder is the system row's cell: Gap plus the counted
	// lines that have no row.
	Gap       int64
	Remainder int64
	Why       string // "", "rounding", "margin:15", "unbroken", "differs"
}

// Line and column flag codes.
const (
	FlagAmountNotFound = "amount_not_found"
	FlagFromScan       = "from_scan"
	FlagRowUnsure      = "row_unsure"
	FlagRowUnknown     = "row_unknown"
	FlagCountedToMatch = "counted_to_match"
	FlagBreakdown      = "treated_as_breakdown"
	FlagRepeat         = "repeat"

	FlagNotFixedPrice  = "not_fixed_price"
	FlagGSTUnknown     = "gst_unknown"
	FlagNoTotalPrinted = "no_total_printed"
	FlagCheckTenderSum = "check_tender_sum"
	FlagLinesShort     = "lines_short"
	FlagLinesOver      = "lines_over"
	FlagGroupDiffers   = "group_differs"
	FlagSummaryCounted = "summary_counted"
	FlagOptionsOffered = "options_offered"
	FlagIncomplete     = "reading_incomplete"
)

// States of a line.
const (
	StatePriced    = "priced"
	StateIncluded  = "included"
	StateExcluded  = "excluded"
	StateNotPriced = "not_priced"
)

// SystemRow is the schedule key of the "Not broken down" row.
const SystemRow = "tender:not_broken_down"
```

The flag codes and the line states are the constants at the foot of that file. `Gap` is what the counted lines do not explain. `Remainder` is the system row's cell.

### C3. Reserved keys

- `tender:not_broken_down`: the system row. A `schedule_items` row with kind `stage`, base true, source `generic`. `ensureSchedule` and the RFP refresh must leave any key starting `tender:` alone.
- Suggestion keys for adds and subs: `line:<line id>`, `gap:<row key>`, `option:<schedule item key>`.

### C4. The pass on a submission

`submissions.tender_state`: `''` (never asked, or the pass is off), `pending`, `reading`, `read`, `failed`.

A submission is **tender-read** when `tender_state = 'read'`. Only tender-read submissions have lines, a tender sum, a checked state and adds and subs. Every other submission behaves exactly as it does today.

### C5. Table JSON

`GET /projects/{id}/procurement/packages/{pkg}/prices` keeps everything it returns now and gains one member, `tender`:

```json
{
  "on": true,
  "version": 7,
  "rows": [
    {"key": "stage:preliminaries", "label": "Preliminaries and site establishment", "system": false},
    {"key": "tender:not_broken_down", "label": "Not broken down", "system": true}
  ],
  "columns": [{
    "submission_id": "uuid",
    "state": "read",
    "error": "",
    "basis": "fixed",
    "tender_sum": "737000.00",
    "tender_sum_how": "one printed total",
    "tender_sum_by": "rule",
    "remainder_why": "",
    "like_for_like": "737000.00",
    "lowest": false,
    "checked": false,
    "checked_at": null,
    "flags": [{"code": "lines_short", "text": "Lines are $31,000.00 short of the tender sum"}],
    "notes": ["Quote valid 30 days"],
    "exclusions": [{"p": 4, "text": "Authority fees"}],
    "cells": {
      "stage:preliminaries": {"state": "priced", "amount": "96000.00", "lines": 3, "mark": "", "to_check": false}
    },
    "lines": [{
      "id": "uuid", "ordinal": 0, "page": 9, "label": "Site establishment", "printed": "$96,000.00",
      "amount": "96000.00", "gst": "ex", "state": "priced", "kind": "base", "placed": "sum",
      "heading": "Preliminaries", "row": "stage:preliminaries", "note": "", "counted": true,
      "flags": [], "decided_by": "model", "split_of": ""
    }]
  }],
  "adjustments": [{
    "id": "uuid", "label": "Option: epoxy grout", "note": "", "origin": "system", "reason": "option", "row": "",
    "amounts": {
      "<submission id>": {"amount": "18000.00", "state": "suggested", "source": "firm", "line_id": "uuid", "page": 34}
    },
    "others": {"<submission id>": {"low": "24000.00", "high": "126000.00", "average": "75000.00", "count": 2}}
  }],
  "suggested": 3
}
```

- `on` is true when the server has a key and the org has switched the model on.
- A cell's `mark` is `""`, `"high"`, `"low"` or `"gap"`. `to_check` is true when any counted line in it has a flag.
- `others` is given only for a `gap` row and only for firms that lack an amount.
- `adjustments` and `suggested` are empty until slice 5.

### C6. Operations

`POST /projects/{id}/procurement/packages/{pkg}/tender/ops` with `{"version": 7, "ops": [...]}`. Returns the table (C5) on 200, with `"change": "<id>"` from slice 4 on when the batch can be undone. 409 when `version` is stale. 422 when an operation is invalid, with a message that names the operation's position and the field.

| `op` | Fields | Slice |
|---|---|---|
| `move_line` | `line`, `row` (`""` is the system row) | 4 |
| `split_line` | `line`, `parts: [{row, amount}]`, `rest_row` (named parts are in the line's own GST basis; Go works out the rest, so the parts always add to the line) | 4 |
| `unsplit_line` | `line` (the original; its parts are removed and it is counted again) | 4 |
| `set_line` | `line`, any of `label`, `amount`, `gst`, `kind`, `placed`, `state` | 4 |
| `add_line` | `submission`, `label`, `amount`, `gst`, `row`, `page` | 4 |
| `count_line` | `line`, `counted` | 4 |
| `delete_line` | `line` (only a line the user added) | 4 |
| `set_tender_sum` | `submission`, `amount`, `gst` (stored as the user's own total line, so undo covers it) | 4 |
| `check_column` | `submission`, `checked` | 3 |
| `add_adjustment` | `label`, `note`, `row` | 5 |
| `set_adjustment` | `adjustment`, any of `label`, `note` | 5 |
| `remove_adjustment` | `adjustment` (a user's row is deleted; a system row is dismissed) | 5 |
| `set_adjustment_amount` | `adjustment`, `submission`, `amount` (signed, or null to clear) | 5 |
| `apply_adjustment` | `adjustment`, `submission`, `applied`, and for a gap `source` (`others_highest`, `others_average` or `typed`) with `amount` when typed | 5 |
| `take_out_line` | `line` | 5 |
| `apply_suggested` | `submission` (every suggestion of that firm that carries the firm's own figure) | 5 |

Amounts in operations are decimal strings. Go parses them with `invoices.ParseCents`.

### C7. Budget paths

| Name | p50 | p90 | Kind |
|---|---|---|---|
| `tender_read` | 15 s | 30 s | live gate, up to 100 lines |
| `tender_read_long` | 60 s | 90 s | live gate, over 100 lines |
| `tender_reconcile` | 100 ms | 250 ms | a Go test, 330 lines (measured 1.4 ms) |
| `tender_edit` | 100 ms | 250 ms | bench |
| `price_table` | 150 ms | 300 ms | bench, unchanged, now with five firms of 330 lines |

## Reporting

At the end of each slice write `docs/evidence/2026-11-14-tender-comparison-slice<N>.md` with: what was built; what calls what; what could fail; the timings against the budgets; any new dependency and why; anything in this plan that turned out wrong. No firm names, no real prices.

## Risks to watch

- **The large schedules.** The trial did not reach 80% on 300-row schedules. Slice 1 measures it properly. Do not paper over it.
- **Text from two extractors.** The trial checked amounts against PyMuPDF text. The product uses `identity.Extract`. Slice 1 measures R3 with the product's extractor.
- **The system row and old columns.** A package can hold tender-read and old-style submissions together. Old-style totals must ignore the system row (slice 2 task 6).
- **Award.** It reads `submission_prices`. The projection must keep those cells truthful, or award writes wrong contract values.
- **Other sessions.** `main` moves. Rebase the worktree before each slice and re-read any file this plan quotes by line number.
