# Invoice intake and ledger: implementation plan

Date: 16 October 2026. Lane A (money a PM certifies). Source task:
`docs/plans/2026-10-15-invoice-intake.md`. The owner is away; the open
questions there are answered below with the most conservative reading and
are marked **assumed**. Each can be reversed without a migration unless
noted.

## 1. Mapping the placeholders to the repo

| Spec name | Repo name | Notes |
|---|---|---|
| packages | `packages` | Disciplines, not firms. No ABN column today. |
| supplier ABN | `packages.supplier_abn` (new, nullable) | Learned only from a user act (§4.4). |
| cost_lines | `cost_items` (stable id) + `cost_item_revisions` (per plan version) | Allocations point at `cost_items.id`, so a rename or a new draft keeps them. |
| budget per line | `cost_values` metric `budget` on the current draft | Ex GST when the plan's `tax_basis` is `ex_tax`. |
| invoices | `invoices` (new) | One row per dropped file; links to `files`, not `documents`. |
| invoice_allocations | `invoice_allocations` (new) | `cost_item_id` null = Unallocated. |
| decisions | `invoice_decisions` (new, same shape as `decisions`) | `decisions` has a foreign key to `documents`; invoices are not register documents (assumed, §2). |

## 2. Open questions, as assumed

1. **Field list.** The provisional columns in the spec, unchanged.
2. **Packages.** No ABN today. Add `packages.supplier_abn` (11 digits,
   checksum-valid, nullable). It is set only when a user confirms an
   invoice's supplier and the invoice's issuer ABN was read; never by Jev.
3. **GST basis.** Budgets are compared ex GST. When a plan's tax basis is
   `inc_tax`, invoiced to date still shows (ex GST) but remaining and
   % spent are unknown, with the reason shown. Unknown stays unknown.
4. **Contractor claims.** A contractor's tax invoice is an invoice. A payment
   claim or schedule that is not a tax invoice is "not an invoice" and stays
   out of totals until the user confirms it. No retention, no SOP dates.
5. **Document register.** Invoices do not appear in the register in v1. The
   bytes share the project's content-addressed `files` row.
6. **Variations.** Out of scope; the tab is an empty state.

## 3. Data model (migration `031_invoices.sql`)

All tables carry `org_id`; every foreign key is composite with `org_id`, so a
row can never point into another org.

```
packages.supplier_abn text CHECK (supplier_abn ~ '^[0-9]{11}$')

invoices(org_id, id, project_id, file_id, filename,
  status   reading | filed | needs_review | not_read | void | superseded,
  kind     invoice | credit_note | not_invoice,
  number, invoice_date date, period date (first of month),
  package_id, issuer_abn, currency (default AUD),
  ex_gst_cents, gst_cents, total_cents   bigint, null = unknown,
  not_read_reason, invoice_count int (distinct invoice numbers seen),
  superseded_by, duplicate_ok bool, retry_after, created_at, version)
  UNIQUE (org_id, project_id, file_id)   -- same file = "Already filed"

invoice_allocations(org_id, id, project_id, invoice_id, cost_item_id null,
  amount_cents bigint null, line_label text, decided_by, band, created_at)

invoice_decisions(org_id, invoice_id, field, value, band, decided_by,
  question_version, confidence, proposed_value, version)
  PRIMARY KEY (org_id, invoice_id, field)
```

`invoices.file_id → files` is `NO ACTION`, so a document delete cannot take
an invoice's bytes: `DeleteDocuments` also keeps a file row an invoice uses.
Project delete removes allocations before cost items, then invoices cascade.

## 4. Pipeline

`POST /api/projects/{id}/invoices?name=` streams bytes to the blob store,
upserts the `files` row and inserts an `invoices` row with status `reading`
in one transaction, appends an `invoice` event, and returns 201 (or 200 with
`already_filed` for a known hash). A foreground reader then:

1. **Extracts** identity text (up to 6 pages). No text layer: status
   `not_read`, user may type fields. Unsupported format: `not_read`.
2. **Harvests** (code): money amounts, dates, invoice-number candidates,
   ABNs (checksum), currency marks, labelled cells (`Invoice No`, `Invoice
   Date`, `Total`, `GST`, `Subtotal`), heading words (tax invoice, credit
   note, quote, statement, remittance, receipt).
3. **Rules** settle what code can:
   - amounts: exactly one distinct (ex, GST, total) triple with ex + GST =
     total ±1 c and GST = 10 % of ex within 1 c → green;
   - number / date: one distinct labelled candidate → green;
   - issuer ABN: one valid ABN that is not another package's customer ABN →
     green; supplier: issuer ABN equals exactly one package's
     `supplier_abn` → green;
   - activity line: earlier user-confirmed invoices from the same package all
     went to one live line → green; else the package has exactly one live
     posting line → green;
   - kind: "tax invoice" heading and no other kind word → invoice.
4. **One Jev fan-out** (interactive priority, 1.2 s deadline) for whatever is
   still open: `kind`, `number`, `date`, `issuer_abn`, `total`, `gst`,
   `supplier`, `line`. Every question picks among pre-found candidates or
   project rows; Jev never computes. Bands come from calibrated thresholds
   (`data/invoice/thresholds.json`, version `invoice-1`).
5. **Code** derives ex GST = total − GST from Jev's picks, negates a credit
   note, sets the period to the upload month in Australia/Sydney, and commits
   decisions, the invoice row and one allocation in one transaction with an
   `invoice` event. A user decision is never overwritten (version check).
6. **Timeout**: grey chips, rule values stand, `retry_after` set; a
   background sweep re-asks the grey questions at background priority.

## 5. Ledger and checks (pure Go, `internal/invoices`)

- `Summarize(lines, ledger)` → per line invoiced to date, remaining, % spent,
  and an Unallocated row = ledger total − Σ line allocations. The invariant
  (ledger total = Σ lines + Unallocated) holds by construction and is
  property-tested.
- Counted invoices: status not void/superseded/not_read/reading, kind
  invoice or credit note, currency AUD, ex GST known.
- `Check(...)` returns flags with plain reasons: same file, duplicate
  number, look-alike, arithmetic, GST rate, ABN invalid / differs, split
  total, period mismatch, over budget, unmapped, odd date, foreign currency,
  not an invoice, may contain N invoices. Flags never block entry.

## 6. API

| Route | Budget path |
|---|---|
| `POST /projects/{id}/invoices?name=` | `invoice_intake` (upload → row event) |
| `GET /projects/{id}/invoices` | `invoices_read` (ledger + summary + flags) |
| `PATCH /projects/{id}/invoices/{inv}` | `invoice_write` (field edit, re-map, void, supersede, keep both) |
| `PUT /projects/{id}/invoices/{inv}/allocations` | `invoice_write` (split) |
| `GET /projects/{id}/invoices/{inv}/file` | — (PDF beside the fields) |
| `GET /projects/{id}/invoices/export?format=xlsx|csv&sheet=` | `invoice_export` |

Every write returns the recomputed summary from the same transaction's
snapshot and appends an `invoice` event so other tabs refresh.

## 7. Budgets

| Path | p50 / p90 ms | Gate |
|---|---|---|
| `invoice_intake` one invoice, upload to row | 1,000 / 2,000 | bench |
| `invoice_batch` 20 dropped together, last row | 2,500 / 4,000 (proposed) | bench |
| `invoices_read` 1,000 invoices | 150 / 300 (proposed) | bench |
| `invoice_write` incl. summary | 100 server p90 (proposed); p50 50 | bench |
| `invoice_export` 1,000 invoices | 2,000 p90 (proposed); p50 1,000 | bench |

## 8. Export and print

The `.xlsx` is written with the standard library (`archive/zip` and escaped
XML): two sheets, real numeric cells, a SUM row, header block. **No
excelize**: it would add six modules for a writer this repo needs ~200
lines of. CSV per sheet ships alongside. Print uses `@media print`: A4
landscape, repeated table headers, chips print their text.

## 9. Evaluation

`cmd/invoice-eval` reads an answer key of real invoices from
`data/eval/invoice/private/` (git-ignored; the repo is public), runs harvest
and rules, asks Jev live (`-live`, records) or replays a recording, scores
each field and fits thresholds with `eval.Fit` (policy recorded in the
thresholds file; owner to confirm). The answer key is drafted from the PDFs
by the agent and stays `reviewed: false`.

## 10. Tasks

1. Migration, store (create, read ledger, commit reading, edits,
   allocations, void/supersede), delete paths, isolation test.
2. `internal/invoices`: harvest, rules, questions, checks, summary, xlsx/csv.
3. HTTP routes, reader pool, retry sweep, SSE event.
4. Answer key, `cmd/invoice-eval`, live run, thresholds.
5. Web: Cost plan panel tabs (Cost plan with invoiced/remaining/% columns
   and Unallocated row, Invoices tab with drop zone, filters, side panel,
   Variations empty state), print stylesheet.
6. Bench paths + budgets; Playwright scenario; e2e fake Jev answers.
7. Report in `docs/evidence/2026-10-16-invoice-intake.md`.

Out of scope as in the spec. Not built: approval, certification, Xero/MYOB,
SOP dates, email intake, OCR of scans, auto-splitting a multi-invoice PDF,
foreign-currency conversion, retention.
