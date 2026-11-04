# Compact RFP and transmittal

Owner request, 28 October 2026. Lane A for the saved transmittal; layout is lane B.
The outcome is a readable report and a deliberate record of documents supplied
for pricing, reducing time to review an appointment. Filing is unchanged.

The RFP heading shows its discipline and version. Release and close dates remain;
missing-date checks sit in the closing Issue section. The redundant stage summary
is removed, while stage-sync review stays available. Project columns read Site
and planning, The building, Delivery. All fields are visible, including unknowns;
source pills are omitted here. Construction budget appears under Delivery.
Project conditions remain shared Profile edits, with an explicit explanation.
Scope positions use a compact dropdown at the right of each service row.

Transmittal replaces Information provided with Doc number, Title, Rev, Date and
Discipline. The initial draft lists current repository documents deterministically.
Select all documents opens the repository and selects all saved documents; its
existing Ctrl/Cmd/Shift selection curates the list. Update Transmittal saves that
selection, including an intentionally empty list. Subsequent refresh preserves
its document identities until another explicit update. Existing authored
information remains as a note. Older drafts refresh once to acquire the table.

The existing report setting endpoint accepts document IDs, checks each against
the report's organisation and project, reads the filed identities, and saves a
snapshot in report_settings. The same table is included in the issued PDF.
A missing/deleted or foreign document, duplicate IDs, or stale draft version
rejects the whole change; a failed save leaves the selection available to retry.
Issued content remains immutable. No Jev call, service or dependency was added.
Migration 042 permits the transmittal key and a bounded array in report_settings;
all other setting limits and the issued-content trigger remain unchanged.

Report read and setting budgets remain p50 100 ms / p90 250 ms. Tests cover
selection, reload, refresh, explicit clearing, identity corrections, failed saves,
missing and foreign documents, duplicate IDs and version conflict. Browser timing
uses 25 read/write samples against the real test server. Filing retains 1000/2000 ms.

Out of scope: changing release/close workflow, conditions ownership, scope rules,
Word export, costs, award or issued history. Existing PDF page limits still apply;
a large transmittal can exceed them and issue must refuse rather than truncate.

## Validation

- Frontend typecheck and production build pass.
- 23 browser regressions pass, covering all 18 consultant disciplines and
  repository selection/editing. The new transmittal regression also passes.
- Isolated 25-sample browser gate: report_setting p50 87.1 ms / p90 92.6 ms;
  report_read p50 57.0 ms / p90 60.8 ms, against 100/250 ms each.
  A concurrent full-suite run measured setting p50 103 ms; the isolated run
  removes that test contention without weakening the budget.
- RFP, reports, store and database packages pass. The broader HTTP suite reports
  three failures outside this change: TestProfilePlacesInputsInBuildOrder,
  TestProgramImportUploadReadAcceptUndo and TestProgramRoutes.
- Golden changes are limited to replacing Information provided with a five-column
  transmittal (and adding an empty transmittal where the prior row was absent).
- Desktop and narrow-screen layouts were checked. A narrow transmittal scrolls
  horizontally inside its container; its actions remain within the viewport.
