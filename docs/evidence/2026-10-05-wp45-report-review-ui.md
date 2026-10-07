# WP-45a: saved RFP review interface

Date: 2026-10-05. Status: local implementation and browser evidence; M1 owner
acceptance and 0991 quality/time run remain outstanding.

Outcome: a project user can create and reopen an RFP draft for an existing
services package, assemble it explicitly, inspect citations/assumptions, protect
wording and refresh without losing it. Lane A; supports answer trust and time
to decision. No issue/export, RFT/PMP, cost ledger or broad navigation redesign.
Budgets remain report read/write 100/250 ms, assembly 300/1000 ms. No dependency
or Jev call added.

The existing project workbench gains Filing/Reports view controls. Report data
is fetched after the first Reports visit. The report library calls the new
organisation/project-scoped GET report-list route and the existing package
overview. Creation saves identity; assembly is a separate action. A services
package must already exist. Reading-incomplete responses offer explicit wait or
last-completed choices. The report header always says “Draft — not for issue”.

The reader renders saved sections and expandable E/U/C/A text-labelled citations,
with material assumptions listed separately. Source changes show stale/conflict
notices. Saving uses the displayed draft version. After a concurrent edit,
“Review latest draft” fetches the current wording while preserving unsaved text;
the user must explicitly reconcile it before saving against that version.
Switching back to Filing preserves the report component and any unsaved editor.
Relevant project events refresh saved data, but do not overwrite an open editor.

Validation:

- Frontend TypeScript/Vite build passes.
- Report/store/API/benchmark suites pass (`.tools/wp45-api-tests.log`). The list
  route is covered for saved identity and cross-organisation 404 behavior.
- Real-server Playwright report workflow passes (2.1 s test body): creation,
  explicit assembly, client reading-incomplete choice, protected editing,
  source inspection, package-change refresh, real concurrent edit/reconciliation,
  saved reopen and 390 px horizontal-overflow check. The incomplete-reading
  response alone is simulated at the browser boundary; its backend behavior has
  separate store/API evidence.
- Existing intake Playwright workflow passes (5.4 s test body), covering file
  formats, correction, reconnection and organisation isolation after navigation
  changed.
- Windows Playwright web-server cleanup hung after the tests completed. Each
  owned test server was identified by PID and workspace executable path and
  stopped; Playwright then returned exit 0 for the passing report/intake runs.
  This is not a claim of clean automatic teardown.
- One mechanical design-detector pass returned no findings. Desktop/mobile
  captures are `.tools/wp45-reports-desktop.png` and `...-mobile.png`; concurrent
  reconciliation is `...-conflict.png`.

Independent Impeccable review initially returned **fix**. Its three material
findings were fixed and rescored:

| Finding | Verdict |
|---|---|
| Preserve unsaved text while reviewing a concurrent edit | Resolved |
| Keep “not for issue” outside editable content | Resolved |
| Replace reading-status booleans with ordinary report language | Resolved |

Final reviewer disposition: **ship**, no remaining visible regressions for this
bounded UI slice. This is not M1 acceptance or a whole-package completion claim.

The reduced local benchmark passed all user-path gates
(`bench/results/2026-10-05-wp45-report-list-diagnostic.json`): report read
p50/p90 2.7/4.8 ms (80 mixed single-report/list samples, list grows to 40), write
5.0/6.2 ms (80) and assembly 8.0/10.5 ms (40). Filing was 401.2/637.5 ms.
These basic report and 20-file/two-pass results do not establish full 0991,
production/VPS or release performance. The established UI system is documented
from source in `web/DESIGN.md` and its sidecar; no visual identity was replaced.

Protected-edit reset and orphan removal are now implemented. A version-checked
DELETE edit route restores the generated wording saved in this draft, or removes
an orphan with no remaining generated source. It updates the edit record,
sections, citations and report revision atomically, without reading newer project
inputs or reassembling. The UI asks for confirmation before discarding wording.
Existing drafts that predate the saved generated text must be refreshed first;
the UI explains this rather than offering a reset that cannot succeed.

Report/store/API/benchmark tests pass (`.tools/wp45-reset-tests.log`), including
source-hash preservation, orphan removal, stale reset and cross-org rejection.
The real browser workflow also passes (2.4 s test body), now covering cancelled
and confirmed restore and the removal of the protected-wording label. The same
identified test-server cleanup workaround was needed after the passing test.

The reset-inclusive reduced diagnostic passes all user-path gates
(`bench/results/2026-10-05-wp45-reset-diagnostic.json`): report read p50/p90
1.2/3.9 ms (80), mixed create/edit/reset writes 5.3/5.7 ms (120), assembly
7.6/7.9 ms (40), and filing 447.9/670.1 ms (40). These remain local basic-workload
measurements, not the full 0991 or release gate.

Remaining: minimum work/proposal/delivery review controls and further package editing,
the predefined manual baseline and stopwatch active-minute comparison (no M1
analytics table required by §8.6), full 0991 draft acceptance with
owner-reviewed evidence, larger-corpus performance, full phase security gates
and the later M2 issue/export implementation. The local draft demonstration does
not establish an issuable RFP or the owner’s active minutes.

## Minimum package allocation and live updates

The Packages view now creates planned services, works and supply packages using
the existing versioned APIs. Services can have separate stages before and after
novation. Users can add explicit scope obligations or assign a work responsibility
and stage, and edit scope wording, inclusion and deliverable. Concurrent edits
retain unsaved wording and require explicit comparison against the latest version.
Saved assignments display the stage's novation phase as well as its name.
This lane-A slice reduces the steps from accepted work to an RFP draft; it does
not add appointments, verified clauses, new dependencies or Jev calls.

The browser regression exposed a missing live-stream wake after committed domain
writes. Work, package, scope, stage, proposal, delivery and report mutation handlers
now wake the existing org-scoped broker only after successful store writes. The
broker reads the durable log; this neither duplicates events nor holds the write
transaction open for a client. Existing broker tests cover out-of-band committed
events. The browser test proves that a newly created work item becomes selectable
in an already-open Packages view without reloading.

Frontend type checking/build, the targeted API/event suites, and the full serial
Go suite pass (`.tools/wp45-packages-all-tests.log`). The real
browser flow passes (3.3 s test body), covering novated package creation, obligation
and responsibility assignment, saved stage phase, RFP assembly, citations, protected
edits, concurrent-edit reconciliation, reset and mobile overflow. The identified
Windows test-server cleanup workaround remains necessary. Desktop/mobile captures
are `.tools/wp45-packages-desktop.png` and `.tools/wp45-packages-mobile.png`.
The independent Impeccable reviewer initially required the saved novation-phase
label; after correction and recapture, disposition is **ship**, with no remaining
visible regression in this bounded slice.

The reduced local diagnostic
`bench/results/2026-10-05-wp45-packages-diagnostic.json` passes all user-path gates:
package reads p50/p90 3.3/5.6 ms and writes 2.2/2.9 ms; scope reads 1.0/1.1 ms
and writes 2.3/2.8 ms, each against 100/250 ms. Report assembly is 10.8/12.1 ms
against 300/1000 ms; filing is 448.2/653.2 ms against 1000/2000 ms. Extraction
and deterministic-rule component gates remain target-VPS checks and exceeded
their local budgets. These 20-file/two-round measurements are not release or
full 0991 evidence. No dependency was added.
