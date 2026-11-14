# Tender comparison slice 1: the reconciler and the gate

Date: 14 November 2026. Figures are in
`docs/evidence/2026-11-14-tender-comparison-gate.md`.

## What was built

- `internal/tender/amount.go`, `reconcile.go`, `cells.go`: rules R1 to R9
  as the plan gives them, with their 28 tests, plus five tests and four
  small rules the corpus survey called for (gate note, "What the rounds
  changed").
- `TestReconcileBudget` and the `tender_reconcile` budget row.
- `cmd/tender-eval`: the key (`key.go`), scoring (`score.go`), recording
  and replay at the `llm.Asker` seam (`recording.go`), the report, and the
  modes `-live`, `-fresh`, `-split`, `-show`, `-dump`, `-audit`, `-one`.
- `OCR.ReadMissing`: missing pages read four at a time up to a limit.
- Prompt `t2`.

## What calls what

`tender-eval` loads the key and checks each file's hash, extracts text,
OCRs pages with none, runs `tender.Pass` through the replayer (which asks
the live client only with `-live`), then `tender.Reconcile` and
`scoreCase`. The aggregate goes to `data/eval/tender/results/latest.json`.

## What could fail

A changed corpus file or a replay miss stops the run. A case that cannot
be read within four minutes is counted as not read and the run goes on. A
tender sum shown wrong with no flag exits 1.

## Timings

`tender_reconcile`: p50 1.6 ms, p90 2.1 ms for 330 lines (budget 100 and
250 ms). Read times are in the gate note.

## Where the plan was wrong, or was not followed

- The gate says stop. The run went on at the owner's instruction (gate
  note, "Result").
- The key has 20 cases, not 22, and leaves the large schedules unkeyed at
  line level.
- `bench/budgets.json`: a row cannot be marked release-only unless it is
  one of five component budgets (`TestCommittedBudgets`), so
  `tender_reconcile` is an ordinary row and the bench samples it.
  `TestCommittedBudgets` already failed on `main` before this work: ten
  budget rows are missing from its list.
- Task 8 step 4 (the shorter answer) was not run; two prompt rounds were.
- One pass hung for more than ten minutes in the first survey run and was
  not seen again. The eval now gives each case four minutes. The product's
  reader has its own 150 second limit.
