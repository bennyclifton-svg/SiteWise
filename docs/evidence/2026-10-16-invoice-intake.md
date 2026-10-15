# Invoice intake and ledger: implementation report

Date: 16 October 2026. Lane A. Task: `docs/plans/2026-10-15-invoice-intake.md`;
detailed plan and the answers assumed for the open questions:
`docs/plans/2026-10-16-invoice-intake-implementation.md`. Built while the
owner was away; nothing here is pushed, and nothing is marked reviewed.

## What a user sees

Cost Plan now has three tabs. **Cost plan** is the existing grid with three
more columns from the ledger (Invoiced to Date, Budget Remaining, % Spent),
section subtotals, an Unallocated row, removed lines that still hold spend,
and the totals strip gains Invoiced to date and Budget remaining. Clicking an
invoiced figure opens **Invoices** filtered to that line. **Invoices** has a
drop zone, filters (supplier, month, line, status), and one row per
allocation, newest first; each decided field carries a chip with words and an
icon (From document, Confirmed · Jev, Check · Jev/Rule, Not checked, Set by
you, Not set). A row opens a side panel with the PDF beside editable fields,
the checks, split, void/restore, and "Replaces …" / "Keep both" for a flagged
duplicate. **Variations** is an empty state. Export to Excel, cost plan CSV,
invoices CSV and Print sit in the header. Files dropped anywhere on the panel
are invoices; they never fall through to the document register.

## What calls what

```
POST /api/projects/{id}/invoices?name=    → blob store (content hash) → store.CreateInvoice
    (files row + invoices row "reading" + "invoices" event, one transaction)
    → invoiceReader.Start (own 16 slots, 15 s timeout)
        → identity.Extract (≤ 6 pages) → invoices.Harvest (code)
        → invoices.NewDraft: rules (kind, number, date, amounts, supplier by
          learned ABN, line by supplier history / only line)
        → at most one Jev call, interactive priority, for what is still open
        → code: ex GST = total − GST, credit notes negative, period = upload
          month in Sydney
        → store.CommitInvoiceRead (decisions, never over a user's; invoice
          recomputed; one allocation; event) → SSE wakes the panel
GET    /invoices            → store.InvoiceLedger: rows + cost plan budgets →
                              invoices.Summarize + invoices.Check in Go
PATCH  /invoices/{inv}      → user decisions (old machine value kept as
                              proposed_value for calibration), actions, recompute
PUT    /invoices/{inv}/allocations → split
GET    /invoices/{inv}/file?view=1 → PDF inline (frame-ancestors 'self')
GET    /invoices/export?format=xlsx|csv&sheet=… → stdlib .xlsx / CSV
serve: RunInvoiceRechecks every 10 s re-asks grey questions at background
       priority (at most 3 attempts); Resume restarts unread invoices.
```

Invariant by construction and tested: ledger total (counted invoices' ex GST)
= Σ line invoiced + Unallocated. Counted means read, an invoice or credit
note, in AUD, with a known ex-GST amount.

Data: migration `031_invoices.sql` adds `invoices`, `invoice_allocations`,
`invoice_decisions` and `packages.supplier_abn`. Every key is composite with
`org_id`; allocations follow the stable `cost_items` id, so renames and new
drafts keep them; a removed line keeps its spend (label snapshot). Deleting a
register document no longer removes a file row an invoice uses; dropping an
invoice's bytes into the register later files them normally (a previously
unreachable `CommitIntake` error path, fixed).

## Accuracy (90 real files, private answer key)

`cmd/invoice-eval` over `data/eval/invoice/private/` (git-ignored; Hale,
0991 and Caves invoices; labels drafted by the agent from the PDF text,
`reviewed: false`). Live run recorded; thresholds fitted on a 60 % calibration
split by supplier family with Wilson lower bounds at z = 1.645 (green ≥ 0.95,
amber ≥ 0.70; agent's proposal). Production order: rule, then Jev, then code.

| Field | Scored | Correct | Wrong (amber) | Blank | By rule (wrong) | Jev raw correct |
|---|---|---|---|---|---|---|
| kind | 88 | 83 | 0 | 5 | 60 (0) | 83/88 |
| number | 79 | 79 | 0 | 0 | 78 (0) | 78/78 |
| date | 79 | 74 | 0 | 5 | 61 (0) | 79/79 |
| issuer ABN | 79 | 70 | 4 | 5 | — | 67/78 |
| total | 79 | 75 | 0 | 4 | 63 (0) | 79/79 |
| GST | 79 | 74 | 0 | 5 | 63 (0) | 79/79 |
| supplier | 79 | 58 | 3 | 18 | — | 73/79 |
| line | 77 | 55 | 4 | 18 | 54 (0)* | 71/77 |

No wrong value is green (the eval fails the build otherwise). *Lines derived
from an amber supplier are amber. Both unreadable PDFs are "Not read".

Defects found on the real corpus and fixed before calibration: a claim
history's "Invoice No." column read as this invoice; multi-part numbers
("NSW 231837", "06862.14 – 1B"); a far cell beating the value printed under
its label; ABNs glued to "ABN"; a certification page's contract totals
settled green by the amount rule (now refused when a second consistent set
exists); and a lone ABN that was the customer's settled green (there is no
ABN rule now; Jev picks the issuer in the same fan-out).

Known weak spot: on one supplier's template Jev picks the customer's ABN
(amber, 4 cases). Thresholds are thin: most shapes have 15–60 answers, so
many Jev answers land amber or blank. More labelled invoices, or the owner
loosening the policy, would raise green coverage.

## Speed (local replay, `cmd/intake-bench`)

Invoice questions in the replay bench use a latency model drawn from the
recorded filing calls (the recordings hold no invoice calls); `-live` uses
System One.

| Path | p50 / p90 ms | Budget p50 / p90 ms |
|---|---|---|
| invoice_intake (one invoice, upload to row event) | 255 / 314 | 1,000 / 2,000 |
| invoice_batch (20 at once, last row) | 547 / 612 | 2,500 / 4,000 (proposed) |
| invoices_read (1,000 invoices) | 27 / 29 | 150 / 300 (proposed) |
| invoice_write (re-map, summary returned) | 30 / 31 | 50 / 100 (proposed) |
| invoice_export (1,000 invoices, .xlsx) | 23 / 23 | 1,000 / 2,000 (proposed) |

Last of four full runs; the invoice paths varied by under 10 % across runs.
40 samples per path (the gate needs 20). Filing in the same run:
whole_intake 339 / 729 ms.

Playwright, 10 invoices dropped at once: all rows read in about 1–2 s in the
browser.

One bench run showed every ledger call at ~290 ms. The ledger read went
through `ReadCostPlan`, which computes cost suggestions twice; under stale
statistics that dominated. The ledger now reads the draft directly and
resolves each line's package with the same resolver, and decisions are read
per project through their own index. Four later runs: 23–35 ms.

The full bench gate still fails intermittently on `profile_edit` p90 (129 to
564 ms across runs) and once on `spec_home_profile_edit` p50 (51–58 ms against
50 ms). The unchanged tree before this work (bench run in a worktree of the
exact starting snapshot, same database) shows the same: `profile_edit` p90
90 and 498 ms, `spec_home_profile_edit` 47–49 ms. The final run of this tree
measured 47.8 ms. These are the known profile-edit flakes (F31), not this
change; they are not fixed here.

## Tests

- `internal/invoices`: every check with a passing and failing case; ABN
  checksum; money parsing; rules (plain invoice, the two-set trap, claim
  history numbers); one fan-out then code; timeout keeps rule values and goes
  grey; user values never replaced; credit notes; not-read paths; Sydney
  period; pro-rata GST shares; workbook SUM row, escaping, CSV injection;
  ledger invariant over 200 random runs of adds, re-maps, splits, voids,
  credit notes and unknown amounts.
- `internal/store`: lifecycle (dedupe, late read dropped, user correction
  survives a re-check, re-map, split flagged, duplicate → replaces, credit
  note, void), learned supplier ABN and line history, grey re-check
  scheduling, org isolation (store and schema level: allocation, decision
  and package links across orgs refused), document and project delete.
- `internal/db`: the composite foreign-key sweep covers the seven new keys.
- `internal/httpapi`: upload → background read → ledger, re-drop, edit,
  invalid amount (422), xlsx and filtered CSV, inline PDF, unsupported file
  stored as not read, and every route refused from another org (404).
- Playwright `web/tests/invoices.spec.ts`: drop 10, re-drop, cost plan total,
  re-map moves spend to Unallocated, workbook total equals the screen,
  line click filters, Variations, desktop / print / phone screenshots.
- `cmd/invoice-eval` replay added to `tools/check.ps1`.

Full Go suite: passes except `TestProfilePlacesInputsInBuildOrder`, which
fails identically on the tree before this work (14 October profile edits).

Full Playwright suite: 51 of 62 pass. The same 11 fail on the starting tree
(run in a worktree of the exact snapshot): costs and cost-autofill (at
1280 px the cockpit's centre column is ~400 px, so the grid's sticky columns
cover the Budget cell), intake, seven proposals tests and one RFP editor test.
This change first broke cost-autofill by giving its ledger footer the grid's
group-heading class; fixed. The invoice scenario runs at 1700 px for that
reason; the narrow centre column is worth an owner decision (the
Specifications view already offers "Hide documents").

## Dependencies

None added. The .xlsx writer is ~200 lines over `archive/zip`; excelize
would have brought six modules for two plain sheets. `time/tzdata` (stdlib)
is embedded so the Sydney period is right on any host.

## What could fail

- A new supplier template can fool a harvest rule; rules are tested only on
  this corpus. A wrong rule value would be green: the eval gate guards it.
- Thresholds were fitted in-sample on 90 files; held-out counts are small.
- The replay bench models Jev time; it is not a VPS or live measurement.
- The supplier ABN is learned from the user's confirmation; a wrong
  confirmation teaches a wrong ABN (shown on the package; flags "ABN differs").
- Odd date checks only "after today": no project start date exists.

## Open for the owner

1. Review the answer key labels (`reviewed: false`) and the fit policy.
2. Sign off the proposed budgets (batch, read, write, export).
3. The six open questions in the task; current assumptions are in the plan §2
   (invoices not in the register; budgets ex GST; contractor tax invoices
   count, payment schedules do not; variations later).
4. The work is saved on branch `invoices/intake` as two commits: the
   uncommitted tree found at the start (specification workspace, project
   delete, cockpit restyle, from the 14 October sessions), then this feature.
   `main` was not moved and the working tree was left as it is.
