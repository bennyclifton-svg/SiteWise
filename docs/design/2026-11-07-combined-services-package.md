# Combined services package (R17)

Status: direction only. No code. Lane A (procurement packages and cost
figures). Owner ruling R17, `testing/synthetic/reference/OWNER_RULINGS.md`.

## User outcome

On each project the PM chooses how services engineering is bought: one
combined services engineer under one agreement, or separate mechanical,
electrical, hydraulic and fire engineers. Whichever they choose, the
project shows that many appointments: one row, one RFP, one award, one fee
line, and one supplier for invoices.

## Why it matters

- **Answer trust.** T03 shows five services fee lines for four services
  (`FINDINGS.md`, run `20261107-t03-t04-t06`). A cost plan that counts a
  service twice cannot be trusted as a total.
- **Time to decision.** Today the PM must find and remove the four spare
  rows and their fee lines by hand. The Meridale invoice sat unmapped.

## What happens today

1. Default suggestions split services into four disciplines
   (`knowledge/works/package_defaults.yaml`, owner decision D1 in
   `docs/plans/2026-10-11-procurement-default-disciplines.md`). "Services
   Engineer" is still offered under Add discipline.
2. A filed proposal gets a Procurement row for its filed discipline if none
   exists (`internal/store/proposal_package.go`, `EnsureProposalPackage`).
   Meridale's proposal filed as `consultant.services`, so a fifth row,
   "Services Engineer", appeared beside the four accepted defaults.
3. Each package gets one draft fee line, `fee:<package>`
   (`internal/costs/suggest.go:202`). Five packages, five lines (F05, F06,
   F07, F09, F11 in the T03 snapshot).
4. A proposal is linked to a package by its filed discipline; Jev chooses
   only when no rule settles it (`internal/store/submissions.go:199-240`).
5. Award writes into one package's fee line (`internal/store/awards.go`,
   `packageFeeLine`, `Award`).
6. An invoice settles its supplier by ABN or firm name, and only when
   **exactly one** package matches (`internal/invoices/read.go`,
   `settleSupplier`, `settleSupplierByFirm`).
7. One discipline is allowed once per project (`disciplineTaken`,
   `internal/store/package_proposed.go:202`).
8. There is no RFP template for `consultant.services`. RFPs are built per
   discipline template (`internal/rfp`).

## Options

### A. A combined services row that lists the disciplines it covers

One package, discipline "Services Engineer", with a "covers" list (for
example mechanical, electrical, hydraulic, fire). Covered disciplines are
not separate rows. The PM chooses it with one action on Procurement:
**Combine services** (pick the disciplines) or **Split services**.

- Better: one row, one fee line, one award, one ABN so invoices match. The
  program gets one line per stage. It fits every rule that already assumes
  one appointment is one package.
- Worse: a combined RFP must be built from the covered disciplines'
  templates. A fee is per stage, not per discipline, so the cost plan cannot
  say what the electrical design cost. Comparing a combined firm against
  separate firms needs a sum of the separate prices.

### B. Keep four discipline rows, grouped under one appointment

The four rows stay. A new "appointment" record ties them to one firm.

- Better: each discipline keeps its own RFP and scope.
- Worse: award must split one lump-sum fee into four lines. The proposal
  rarely gives that split (Meridale prices by stage only), so the app would
  invent numbers. That breaks "unknown stays unknown". Invoices would match
  four packages by ABN, so the supplier rule never settles. It needs a new
  table and changes award, undo, the ledger and the price table.

### C. A display toggle only

A project setting folds the four rows into one line on screen. The data
stays four packages.

- Better: smallest build.
- Worse: it hides the problem. Four fee lines still sit underneath, and
  award and invoices still see four suppliers.

### Recommendation

**Option A.** The row itself is the per-project choice; there is no
separate setting to drift out of step. It is the only option where one
agreement really is one package everywhere, with no invented split.

## How each surface behaves under A

- **Procurement row.** The row reads "Services engineering (combined)" and
  names what it covers: "Mechanical, Electrical, Hydraulic, Fire". **Combine
  services** picks two or more disciplines. It removes their unawarded rows
  and adds the combined row, in one confirmation. **Split services** does
  the reverse. Default suggestions stay split (D1). A combined row stops the
  covered disciplines being suggested again.
- **RFP.** One combined RFP. It has a section per covered discipline, from
  that discipline's existing template, and the shared conditions once. The
  fee table is one fee per stage (as in `2026-11-03-rfp-scope-by-stage`).
  RFP create does not call Jev.
- **Fee proposal and award.** A proposal filed as Services Engineer, or as
  any covered discipline, links to the combined row by a code rule. It gets
  its one existing fan-out, no more. Award writes the firm's stage items
  under the one fee line, as today. No per-discipline split is written.
- **Cost plan.** One fee line, "Services engineering (combined)". The
  covered disciplines' draft fee lines go through the existing removal
  confirmation (`docs/plans/2026-10-11-procurement-cost-removal.md`). It
  names each line and its budget. Frozen baselines and issued history stay.
  Services works lines (Mechanical services and so on) are construction
  costs and are not touched.
- **Invoices.** The firm's ABN is on one package, so the existing rule
  settles the supplier green. Nothing new in the ledger.
- **Program.** One package, so one set of stage lines through the existing
  stage sync (`internal/store/stage_sync.go`). Dates computed in Go as
  today.

## What calls Jev

Nothing new. A dropped proposal still gets its one existing fan-out after
filing. Combine, split, the RFP, award, cost lines, invoice matching edits
and the program do not call Jev.

## Speed budgets (`bench/budgets.json`)

| Path | p50 | p90 |
|---|---|---|
| Combine or split (`packages_write`, with linked line removal as `costs_delete`) | 100 ms | 250 ms |
| Procurement read (`packages_read`) | 100 ms | 250 ms |
| Cost plan read (`costs_read`) | 100 ms | 250 ms |
| Proposal read (`proposal_read`, includes filing) | 3,000 ms | 6,000 ms |
| Award or undo (`award_write`) | 200 ms | 500 ms |
| Price table (`price_table`) | 150 ms | 300 ms |
| One invoice (`invoice_intake`) | 1,000 ms | 2,000 ms |
| Stage sync (`stage_sync_write`) | 100 ms | 250 ms |

Combine and split need a benchmark, run beside the existing
`packages_write` one.

## Data and migration

- One new nullable list on `packages`: the disciplines a combined row
  covers. Every read and write stays scoped by `org_id` and project, as
  every packages query is now.
- "Services Engineer" may then appear twice in a project only if their
  covers do not overlap. The `disciplineTaken` check learns the covers list.
  A covered discipline counts as taken.
- Existing projects: a live "Services Engineer" row has no covers list. The
  row shows "Covers: not stated" and asks the PM to tick them. Nothing is
  removed automatically (keeps D9).

## Edge cases

- **Switching after award.** A discipline with a live award cannot be
  combined or split. The PM undoes the award first. Undo is already refused
  once an invoice is allocated (`ErrAwardLocked`). That stays.
- **Combined proposal on a split project.** Today a fifth row appears.
  Instead: no new row. The proposal sits on Procurement as unplaced, with
  two choices: "Combine these services" or "Place on one discipline".
  The PM ticks which disciplines it covers; the app does not guess.
- **Partial combinations.** Mechanical and electrical combined, hydraulic
  and fire separate, is allowed. Two combined rows are allowed if they do
  not overlap.
- **Removing a combined row with linked cost lines.** It uses the existing
  confirmation. It does not recreate the separate rows; the PM can split
  first if they want them.
- **Issued RFPs.** An RFP already issued for a discipline stays in history.
  Combining does not reissue it.
- **Budgets on removed discipline lines.** Not moved automatically (see
  owner question 3).
- **Wrong org or project.** Combine and split refuse it like any package
  write.

## Out of scope

- Splitting a combined fee into disciplines.
- Changing default suggestions (D1 stays).
- Trade packages (combined services contractors).
- A new Jev question for which disciplines a proposal covers.
- Fire engineering (performance solutions). It stays its own discipline.

## Owner questions

1. Default suggestions stay four separate disciplines, and combining is
   one click. Or should some project types (fit-outs) suggest combined?
2. Combined RFP fee table: one fee per stage, or per stage and per
   discipline? The second helps compare against separate firms, but asks
   more of the bidder.
3. When combining, a removed discipline fee line may carry a budget. Should
   the combined line take the sum, or start blank?
4. Should a combined proposal on a split project offer to combine, or simply
   place on one discipline and flag it?
5. Is two non-overlapping combined rows (say mechanical and electrical with
   one firm, hydraulic and fire with another) a real case worth allowing?
