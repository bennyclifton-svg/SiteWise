# Tender comparison, slice 8: the old reading, the written rules, going live

Date: 14 November 2026. Branch `tender/comparison`. No firm name and no real
price appears in this note.

## Read this first

The plan says to do this slice "only after the owner has used the new
tables on a live package and said so". That has not happened: the owner was
away and asked for every slice to be built without stopping. So this slice
was built so that it changes nothing until the owner turns it on.

## What was retired

Nothing is retired by default.

- `proposals.Context` has a new field, `SkipPrices`. With it set, the old
  reader does not ask Jev the fee-row questions or the stated-total
  question. It still asks the firm, the package, the kind, the GST basis,
  the rates and the criteria.
- The proposal reader sets it only when all four hold: the server setting
  `SITEWISE_TENDER_SKIP_JEV_PRICES=1`, a model client on the server, the
  org's switch on, and a package known for the submission.
- The setting is **off by default**. Off, a tender is read both ways: Jev's
  reading sits underneath, and the tender pass's cells replace it when the
  pass succeeds. If the pass fails for good, the old figures are still there.
- With the setting on and a pass that fails for good, the submission has no
  priced cells. The column says the prices were not read and offers
  "Try again". That is what the plan asks for.

One difference from the plan: it names `SubmissionForReading` in the store
as the place that sets the flag. It is set in the reader in
`internal/httpapi/submissions.go` instead, beside the server setting, so the
store does not need to know whether the server has a model.

## What still reads the old way, and why

- Every org that has not switched the model on.
- Every server without the key, or without the Python that makes page images.
- A proposal no row claims (no package, so no rows to place lines on).
- Rates, criteria, firm details, the kind of document and the GST basis:
  Jev reads these for every tender. The pass does not.
- An appointment letter: unchanged.

## Old controls in tender mode

For a package in tender mode the screen shows the two new tables in place of
the old comparison. The "Move or combine fee" bar, the "Fees awaiting
alignment" row and the old fee schedule are not drawn (`web/src/Prices.tsx`,
the `tenderMode` conditions). Their code is unchanged for old-style packages
and their tests still run.

## The written rules

- `AGENTS.md`: the owner's uncommitted rule 1 ("Jev reads; one LLM acts only
  on what it is given") was copied from the main checkout's working tree
  into this branch, with the owner's other uncommitted edits to that file
  and to `PRODUCT.md`. The Procurement sentence now names the tender pass.
  The tender entry no longer says "nothing built" and points at the gate note.
- `PRODUCT.md`: the Procurement entry describes the two tables; Positioning
  says Jev is the only AI that files or reads into fields and describes the
  one model.
- `docs/developing.md`: the one line that said Jev is the only model.
- `docs/deployment.md`: a "Tender pass" section (the settings, the Python
  for page images, the log line `llm call`).
- `docs/operations.md` and `deploy/sitewise.env.example`: the three settings.
- `docs/design/2026-11-14-tender-comparison.md`: status.
- `web/public/legal/privacy.html`: one bracketed row in the processors
  table naming Anthropic. **The wording is not written. It needs the
  owner's lawyer**, and the page must not go live with the brackets in it.

## Where the build differs from the design note

- The gate after slice 1 was not a clean pass and the build went on, at the
  owner's instruction. See the gate note.
- The change log and undo did not exist. Tasks 1 to 4 of
  `docs/plans/2026-11-14-change-log-undo-operations.md` were built on this
  branch (migration 056, `internal/store/change.go`, `change_undo.go`).
  Tasks 5 to 10 of that plan are not built. Branch `chat/change-log` exists
  in another worktree with no commits of its own (only the plan file,
  uncommitted); if that work starts there it will meet this branch's
  version at merge.
- Budget rows are ordinary rows, not `where: release`, because the existing
  budget test refuses new release rows.
- The prompt is version t2: one sentence added after the survey, telling
  the model that a builder's margin at the foot of a schedule is not GST.
- A scanned tender is read by OCR up to 30 pages in batches of four, not
  four pages only.
- The reconciler has two flags the design does not list: a figure the pass
  could not find among the lines it listed, and a GST basis the document
  never states.
- Line ids are kept across a re-read so that a dismissed suggestion stays
  dismissed.

## Figures against every budget in design section 12

| Path | Budget p50 / p90 | Measured | How |
|---|---|---|---|
| `tender_read`, up to 100 lines | 15 s / 30 s | 9.1 to 11.3 s / 14.7 to 19.4 s over the survey rounds | live, private corpus, 20 keyed cases |
| `tender_read_long`, over 100 lines | 60 s / 90 s | 62.5 s / 63.5 s (3 cases) | live; p50 is over budget by 2.5 s on three cases |
| `tender_reconcile`, 330 lines | 100 ms / 250 ms | 1.6 ms / 2.1 ms | Go benchmark |
| `price_table`, five firms of 330 lines | 150 ms / 300 ms | 12.5 ms / 13.3 ms | server test against PostgreSQL |
| `tender_edit` | 100 ms / 250 ms | 31.7 ms / 33.3 ms | server test against PostgreSQL |
| `change_undo` | (same as edit) | 13 ms / 15 ms | server test against PostgreSQL |

The timings in the last four rows are from the development machine, not the
server. The full intake bench (`cmd/intake-bench`) was not run on this
branch; the tender rows were measured by their own tests.

The shorter-answer prompt round (design section 12's first speed lever) was
not run.

## Final test run

Run on the development machine at the end of the build, against a private
PostgreSQL 17.

- `go vet ./...`: clean.
- `go test -p 1 ./...`: 38 packages pass. Two tests fail,
  `TestProgramRoutes` (httpapi) and `TestCommittedBudgets` (latency). Both
  fail the same way on commit c9eb78e, which this branch started from.
- `npm run build`: passes.
- `npm test` (vitest): 268 pass, 3 fail, all three in
  `src/program/engine.test.ts`. The same three fail on c9eb78e.
- `python tools/check_knowledge.py`: 0 errors.
- Browser tests (Playwright, the whole suite): 85 pass, 12 skipped, 61
  fail. Every tender spec passes (`account-model`, `tender-table`,
  `tender-edit`, `tender-adds-subs`, `tender-builder`, `tender-report`).
  The 61 failures are in Program, RFP, profile, invoices and other specs.
  The same spec files were then run on an untouched copy of c9eb78e: the
  same 61 tests fail there, name for name, and none fails only on this
  branch. Those tests are stale against screens that changed before this
  work; they are not fixed here.

Not written: a browser test for a combined engagement in tender mode (the
server test covers it), and a browser test of dragging a line between rows
(the move is tested through the same operation the drag sends).

## Going live: a checklist to run with the owner

Before any of this:

- The owner reviews the answer key (20 cases, written by the assistant,
  `reviewed: false`) and marks it. Three large schedules have no key.
- The owner reads the gate note and decides whether the row-placement
  result is good enough to use. Sums were not silently wrong in any round;
  row placement varied between rounds.
- The owner looks at the screens in a browser. Nobody has yet.
- This branch is merged. Migrations 055 to 058 must be renumbered if main
  has taken those numbers by then.

Then:

1. The key and the model id are set on the server, the OCR Python is
   present, and the server log shows the model client is on.
2. The owner switches the model on for their own org in Account.
3. Three real tenders the owner knows well are dropped on a package. The
   owner compares each column with the document.
4. `cmd/tender-eval` replays on the server's build with the reviewed key
   and passes.
5. The owner confirms the retention setting of the provider account.
6. The privacy page's bracketed row is replaced with the lawyer's wording.
7. Nothing is switched on for any other org by anyone but its owner.
8. Only after step 3 goes well: set `SITEWISE_TENDER_SKIP_JEV_PRICES=1`.

## What could fail

- The model provider is down or slow: the read is retried with back-off,
  then the column says the prices were not read. With the skip setting on
  there is then no figure for that firm until "Try again" works.
- A tender whose lines the pass places on the wrong rows: the column still
  adds to the firm's tender sum, but a row-by-row comparison misleads until
  the manager moves the lines. The column stays "to check" and cannot be
  awarded or signed off until the manager checks it.
- A tender of more than about 200 lines: the answer may be cut off at the
  output limit. The lines that arrived are kept, the read is marked
  incomplete, and the reconciler shows the shortfall on the "not broken
  down" row rather than a wrong total. This was measured on few cases.

## New dependency

`github.com/anthropics/anthropic-sdk-go/v2` (slice 0): the provider's own
client, used only by `internal/llm`.
