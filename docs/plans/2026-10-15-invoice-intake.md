Invoice intake and ledger: feature task
15 Oct 2026 · @Ben clifton 
Summary
A PM drops a batch of invoices into a project's cost plan panel, and each one lands in a single invoice ledger within about a second, mapped to a cost plan activity line. The Cost plan tab's expenditure to date and remaining budget update live from that same ledger, so the two views can never disagree. Both views print and export to Excel.
This slice is intake, ledger, mapping and export. Approval, the certification certificate and sending to accounts are the next slice; the Variations tab ships as an empty shell until its scope is agreed.
Task header
This is a Lane A change: it is money a PM certifies, so trust and org isolation rules apply in full. Budgets marked proposed are mine and need the owner's sign-off.
Item
Value
User outcome
Drop any number of invoices; see each in the ledger, mapped to an activity line, with spend to date and remaining correct on the cost plan.
Why
Time-to-decision: budget position at a glance, without a spreadsheet rebuild each month. Answer trust: every field shows whether rule, Jev or the user decided it.
Lane
A, product core
Budget: one invoice, upload complete to ledger row
p50 ≤ 1,000 ms, p90 ≤ 2,000 ms (same gate as filing)
Budget: batch of 20 dropped together, last row visible
p50 ≤ 2,500 ms, p90 ≤ 4,000 ms (proposed)
Budget: Invoices tab load, 1,000 invoices
p50 ≤ 150 ms, p90 ≤ 300 ms (proposed)
Budget: summary refresh after a commit or re-map
p90 ≤ 100 ms server side, then SSE push (proposed)
Budget: Excel export, 1,000 invoices
p90 ≤ 2,000 ms (proposed)
Jev
One fan-out per invoice; foreground priority over background jobs
Scope rule
Every row carries org_id; no LLM; no new service
Out of scope for this slice: approval workflow and certification certificate; sending to accounts or Xero/MYOB; Security of Payment deadline tracking; Variations tab content; email-forward intake; scanned invoices beyond "stored, not read"; splitting a multi-invoice PDF; foreign currency; retention on progress claims.
Cost plan panel
The panel has three tabs: Cost plan, Invoices and Variations. Both the Cost plan and Invoices tabs read from the same ledger, so a change in one shows in the other within one SSE push.
Cost plan tab. This is the existing summary, with three columns computed from the ledger: invoiced to date, remaining and % spent. An Unallocated row holds invoices with no activity line, so the totals always add up. Clicking a line opens the Invoices tab filtered to that line.
Invoices tab. A drop zone sits above one table of every invoice, from every consultant and contractor. An invoice split across several activity lines shows as one row per line, each with its share of the amount. Rows arrive live as each invoice is read, newest first, and the user can filter by supplier, month, activity line and status. Clicking a row opens a side panel with the PDF beside the fields; changing the activity line there is one move, and the summary updates straight away.
The columns below are provisional until the owner confirms the field list.
Column
Decided by
Notes
Invoice no.
Jev picks from candidates
Copied verbatim from the document
Supplier
Rule (ABN match), else Jev
Picks among the project's packages; no match shows amber
Invoice date
Jev picks from candidates

Month
Code: month uploaded; user can override
The period it is certified in, in the project's local time
Activity line
Rule (supplier history), else Jev
Blank band = Unallocated
Amount ex GST (AUD)
Code arithmetic, Jev if ambiguous

GST (AUD)
Code

Total inc GST (AUD)
Code

Status
Code
Filed, Needs review, Not read, Void
Every decided field carries its band chip (green, amber, blank, grey) with text and an icon, as in filing. A user's correction is final and is never overwritten by a later judgement.
Variations tab. This slice ships an empty state that says variations are coming. It holds no data yet.
Intake pipeline
Each invoice follows the filing pipeline already built, with invoice questions in the Jev fan-out. Fields that rules settle leave the Jev call, so Jev only answers what code can't.
None of the three side exits stops other invoices in the batch; each file runs on its own, in parallel.
Data model
The ledger is the single source of truth: cost plan totals are computed from it in the same transaction as any invoice change, never typed in or cached separately. Money is stored as integer cents in AUD.
Table
Key columns
Notes
packages
Existing table, linked to a discipline or supplier
No new supplier table; invoices link to the package
invoices
org_id, project_id, file_id, package_id, invoice number, invoice date, period month (defaults to upload month), ex GST, GST, total, kind, status
Kind is invoice or credit note; void instead of delete
invoice_allocations
invoice_id, cost_line_id, amount ex GST, decided by
One row per activity line; intake creates one, the user splits in the side panel, and the ledger shows one row per allocation
decisions
Existing table
One row per field per invoice: value, confidence, band, decided by, question version
cost_lines
Existing cost plan lines
Already in the repo. Budget per activity line; a line with allocations is archived, never deleted
Table and column names above are placeholders; the coding agent maps them to the repo's real names before writing the plan.
Allocations are modelled from day one, so splitting an invoice across lines later needs no migration. Corrections are written as user decisions and kept as labelled examples for threshold calibration, the same as filing.
Reconciliation checks
All checks run in code, flag the row amber with a plain reason, and never block an invoice from entering the ledger. One invariant holds at all times: the sum of non-void invoices in the ledger equals the cost plan's invoiced to date plus Unallocated.
Check
Rule
Flag shown
Same file
Identical content hash
"Already filed" (no new row)
Duplicate number
Same supplier and invoice number
"Possible duplicate of INV-1042"
Look-alike
Same supplier, total and date, different number
"Looks like INV-1042"
Arithmetic
Ex GST + GST = total, within 1 cent
"Totals don't add up"
GST rate
GST is 10% of ex GST, within rounding
"GST isn't 10%"
ABN
Checksum valid and matches the supplier record
"ABN invalid" or "ABN differs from supplier"
Split total
Allocations add up to the invoice's ex GST amount
"Split doesn't add up"
Period mismatch
Invoice dated in an earlier month than its period
"Dated September, in October's period" (a hint, not an error)
Over budget
Line's invoiced to date exceeds its budget
Amber on the cost plan line
Unmapped
No activity line
Row sits in Unallocated
Odd date
After today or before the project start
"Check the date"
Credit note
Negative amounts
Reduces the line's spend
Export and print
Export writes one .xlsx workbook of what the current filters show, with real numbers, not text, so users can rework it in Excel.
• Cost plan sheet: each activity line with budget, invoiced to date, remaining and % spent, plus the Unallocated row and a SUM total row.
• Invoices sheet: one row per allocation, with invoice number, supplier, date, month, activity line, ex GST, GST, total, status and source file name.
• Header block on each sheet: project name, export date and the filters applied.
Writing .xlsx in Go needs a library; excelize is the widely used option, and its addition should be justified in the commit as Lane C. CSV export of each sheet needs no dependency and ships alongside.
Print uses a print stylesheet for both tabs: A4 or A3 landscape, the same header block, repeated table headers on each page, and no band colours without their text labels.
LLM use
This slice uses no LLM, so the "Jev only" non-negotiable in AGENTS.md holds unchanged. Jev picks and code computes every number.
Two later uses are worth a separate decision: a background vision fallback that reads scanned invoices into the same pipeline, and a drafted query to a supplier when a check fails. Neither ever sets a field or an amount.
Edge cases
Every case below keeps the invoice visible in the ledger; none silently drops or overwrites one.
Case
Behaviour
Scanned PDF, no text layer
Stored, status "Not read"; the user can type the fields in
Not an invoice (quote, statement, remittance)
Jev's kind question flags it; stays out of totals until the user confirms
Several invoices in one PDF
Flag "may contain 3 invoices"; no auto-split in v1
Invoice split across activity lines
User splits it in the side panel; the ledger shows one row per line
Last month's invoice uploaded early this month
Lands in this month's period; the user changes the month in one move
Revised invoice, same number, new amount
Flag duplicate; the user picks "replaces" (old one kept as superseded, out of totals) or "keep both"
No matching package
Flagged amber; the user picks or creates the package
Cost plan has no lines yet
Invoices go to Unallocated
Activity line renamed
Allocations follow the line id, so the new name shows
Line with allocations removed
Archived, not deleted; its spend still counts
Jev timed out
Grey chips; rule values stand; re-checked in the background
50 files dropped at once
Queued; each row appears as it finishes; foreground keeps reserved Jev capacity
Invoice in another currency
Flagged, excluded from totals until entered in AUD
Wrong org or project
Refused at the boundary; covered by the isolation test
Upload interrupted
Re-drop is safe through hash dedupe
Done criteria
The slice is done when every box below is ticked and tools/check.ps1 passes.
[ ] Speed benchmark fails the build when any budget in the task header is broken
[ ] An invoice answer key of at least 50 real invoices exists, and the accuracy eval sets each question's threshold
[ ] Invariant test: after a random run of adds, re-maps, splits, voids and credit notes, ledger total equals invoiced to date plus Unallocated
[ ] Org isolation test covers suppliers, invoices and allocations
[ ] Duplicate, look-alike, arithmetic, GST and ABN checks each have unit tests with a passing and a failing case
[ ] Playwright: create a project with a cost plan, drop 10 invoices at once, see all rows, re-map one, see the summary change, export, and check the workbook totals match the screen
[ ] Report written: timing against budget, what calls what, what could fail, and any new dependency
Open questions
Three questions were answered on 15 Oct 2026 and are now in the spec: splits show as one ledger row per activity line; the month is the period certified in, defaulting to the month uploaded; and cost plan activity lines already exist in the repo. Invoices link to the existing packages table, so no supplier table is needed. The field list still affects the data model.
[ ] Field list. Which exact columns go in the ledger? (Owner to supply.)
[ ] Packages. For the coding agent: confirm the packages table's shape, whether it stores an ABN (the supplier rule needs one), and the real cost plan table names, before writing the implementation plan.
[ ] GST basis. Is the cost plan budget ex GST? This spec assumes yes.
[ ] Contractor claims. Are contractor progress claims in scope now, or consultant invoices only? Claims bring retention and Security of Payment deadlines.
[ ] Document register. Should invoices also appear in the project's document register, or only in the cost plan panel?
[ ] Variations. When approved variations arrive, should they adjust each line's budget (a revised budget column)?