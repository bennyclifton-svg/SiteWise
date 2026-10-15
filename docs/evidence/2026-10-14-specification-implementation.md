# Specification implementation and verification

This is the first-release record. The table-first workspace, Profiler boundary,
consultant coverage and latest measurements are documented in the
[coordinated implementation evidence](2026-10-14-coordinated-specification-implementation.md).

Date: 14 October 2026. Lane A. First working release of the shared specification
records and workspace; the full five-slice plan remains in progress.

Local preview refreshed at approximately 19:30 Sydney time. The development
server on `http://127.0.0.1:8080` started successfully with the additive migrations.
A read-only smoke check of Custom Home 2 returned the new specification endpoint,
54 attribute definitions and zero specification items. No reference products or
synthetic test selections were inserted into that project. Refresh an existing
browser tab to load the new Specifications navigation item.

## User outcome

Open a project and select **Specifications**. Add an item, assign its locations,
enter a known detail or an explicit unknown, review the value and view the
assembled schedule. Issue a preliminary, coordination or pricing snapshot and
download its PDF or CSV. A consultant package or uploaded document is not needed.

For existing source material, file the document in the project register, choose
it under **Import project information**, and start the background reading. Import
new candidates or attach the document's proposals to an existing item. Review
source excerpts and markings before accepting values. A rerun does not erase
accepted decisions. Fragmented text and finished schedules use these same records.

## Implemented contract

- Project-owned material, product, finish, assembly and equipment definitions;
  display codes need not be unique. Definitions may refer to a building system.
- Separate applications record location, surface, work action, inclusion and
  selection. Optional part/work-item references and quantity exist in the API;
  the current form uses a location description rather than a physical hierarchy.
- Fifty-four registered attribute definitions cover identity, finishes, dimensions,
  performance, installation, quality, commercial evidence and retained notes.
  Numeric values validate their unit family and preserve unknown separately
  from zero. Minimum/maximum, basis, test method and datum are explicit context.
- Assertions distinguish requirements, selections, capabilities, allowances,
  exclusions and observed conditions. Mixed imported prose is unclassified.
  Local accepted values override a matching type default; another meaning is
  preserved, so a product capability does not replace a requirement.
- Proposed, accepted, rejected and superseded values retain source file hash,
  document identifier, page/location and excerpt. Source deletion is surfaced as
  unavailable; issued snapshots keep their evidence as it was at issue time.
- Replacement of an accepted value checks the expected prior decision and
  requires a reason. Optimistic versions reject stale edits. History and issued
  snapshots are immutable at the database boundary.
- Basic component, finish, dependency, matching and replacement relationships
  support layer order and blend percentage/basis. Dependency cycles are rejected.
- Scheme and alternative-group labels, application inclusion/choice, and local
  overrides are supported. These are labels and decisions, not a conditional
  selection or compatibility engine.
- Draft assembly and exports use accepted assertions. Pending values and
  unresolved locations/choices remain visible. PDFs are basic text schedules;
  images and the golden document's designed table presentation are not yet built.
- PDF bytes are frozen at issue time, stored by content hash and registered in
  the existing file store. CSV is reconstructed from the immutable snapshot.
  A new issue does not change previous PDF bytes or selections.

## Runtime and ownership

Migrations 022–026 add the specification tables, history, immutable issues,
import requests/readings and PDF references. Composite foreign-key coverage
includes every new reference and checks other projects and organisations.
All writes validate the project, user and referenced records, then append a
revision/event in the same transaction.

The React workspace calls the new project API. Direct entry, review, schedule
assembly and export do not call Jev. A queued `specification` job runs in the
existing worker, reuses full source extraction, harvests candidate spans, and
makes one bounded Jev fan-out per candidate for item/kind/meaning. Readings are
cached by source and parser version. No new runtime dependency or service was
added. Authoring remains available when Jev is unavailable.

PDF run geometry is retained alongside source text so a code column can be
distinguished from product codes and supplier names elsewhere in a row.
Older native PDF caches without geometry are refreshed for specification import
without replacing profile passages or facts. Cached OCR-only pages are preserved.
Native text and labelled literal fields are retained. Numeric interpretation,
automatic application assignment and cross-page continuation reconciliation are
not claimed. Continued text is kept as a separate reviewable fragment.

The importer does not detect strike-through/highlights or extract image crops.
All imported document values require visual source review before acceptance;
explicitly withdrawn values cannot be accepted. Jev classifications are proposals
and have not been calibrated for automatic acceptance. An import failure is
reported and retryable; partially imported candidates remain reviewable.

## Verification

The standalone browser workflow and failed-load recovery pass. Desktop and
390-pixel mobile captures were reviewed. Long project names now fit the mobile
header, and opening an item/form/schedule focuses and scrolls to the reader.
The independent UI finish review returned **ship**, with both findings resolved.

The knowledge check passes with zero errors and two existing package-default
mapping warnings. Frontend build passes. The complete Go run passed every
package except `internal/httpapi`: `TestProfilePlacesInputsInBuildOrder` reports
95 `proj.flag.*` entries with no display home. Specification tests within that
package pass. The unrelated profiler implementation was not changed to make
this test pass.

The final private schedule inventory compares `(page, code)` occurrences rather
than distinct codes: **131 expected, 131 harvested, no extra or missing coded
rows**. Another 26 uncoded/header/continued-text fragments are retained as
candidates. This measures candidate coverage, not semantic extraction accuracy.
The final check caught and fixed a duplicate `WC` label in the description
column; the regression fixture now checks repeated labels across columns.

The full benchmark ran on Windows/amd64, Go 1.27.1, PostgreSQL on localhost,
with 184 intake files in two rounds and recorded Jev latency. Its specification
workload grows to 131 synthetic items/applications with three accepted details
per item, issuing and retrieving a PDF after every addition. The new paths pass:

| Path | Samples | p50 ms | p90 ms | Budget p50 / p90 ms |
|---|---:|---:|---:|---:|
| Specification read | 131 | 4.1 | 6.1 | 100 / 250 |
| Specification write/review | 1048 | 1.9 | 2.5 | 100 / 250 |
| Issue, including PDF render/store | 131 | 84.4 | 144.9 | 100 / 250 |
| Frozen PDF retrieval | 131 | 3.9 | 6.1 | 100 / 250 |
| Whole intake | 368 | 350.0 | 820.2 | 1000 / 2000 |

The overall benchmark **fails** its separate `spec_home_profile_edit` median
gate: 84.1 ms against 50 ms (p90 93.2 ms against 150 ms). Identity extraction
p90 and deterministic field-rules p90 also exceed their component budgets;
the existing harness reports those on this development host and judges them
on the target VPS. These results are not a release pass, and no budgets were
relaxed. Larger/richer schedule tiers still require measurement and background
PDF preparation where the issue budget cannot be met synchronously.

The installed OCR runtime gate passes: 20 filings measured p50 3262 ms / p90
3368 ms against 10000/20000 ms, including the harness's polling allowance and
replayed provider latency. The existing report creation, citation, protected-edit,
refresh and mobile-reading browser test also passes after the project navigation
extension.

Accuracy replay remains blocked by stale/incomplete recordings: intake evaluation
finds 129 requests without a matching recording; profile evaluation reports
`bankstown-page9: recording is stale`, and the Hale set reports
`hale-description: recording is stale`. These are not counted as passes and the
recordings/baselines were not overwritten to hide them. See
`tmp/specification-intake-eval.log`, `tmp/specification-profile-eval.log` and
`tmp/specification-hale-eval.log`.

Reproducible commands and local logs:

- `go test -p 1 ./...`: `tmp/specification-all-tests.log`.
- New workflow/import/geometry tests: `tmp/specification-final-targeted.log`.
- `npm --prefix web run test:e2e -- specifications.spec.ts`:
  `tmp/specification-e2e.log`, two passing tests.
- Full benchmark: `go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json -samples-out tmp/specification-bench-samples.json -out tmp/specification-bench-results.json`.
- Native schedule extraction/coverage: ignored private scratch directory
  `tmp/pdfs/golden-ffe-review/` and `tmp/specification-coverage-check.json`.

Use the repository's Go runtime/environment setup from `tools/check.ps1` and
the dedicated `sitewise_test` database. Scratch logs and private extraction
artifacts are not committed.

A live call through the pinned `jev-1.13.0` client succeeded on three synthetic
excerpts: a finish requirement, an equipment capability, and unrelated meeting
text. Calls took 188–261 ms in this small connectivity check. This is not an
accuracy evaluation or production latency claim. Private PDF contents were not
used in that live check and have not been copied into tracked fixtures.

## Remaining target work

The [general plan](../plans/2026-10-14-project-specification-schema.md) retains the
full Newham 38-section and six-section architect-schedule analysis. Still needed:

1. A richer physical hierarchy and scope selector, multi-source reconciliation
   across revised documents, image/crop evidence and visual revision handling.
2. Typed fit constraints, conditional alternatives/fallbacks, coordinated scheme
   selection, assembly compatibility and the detailed golden semantic fixtures.
3. Structured quotes with tax/date/unit/validity, supply/install responsibilities,
   evidence-backed approvals, samples, tests, warranties and handover workflows.
4. Visual schedules, room and work-section views, revision comparisons and more
   complete deterministic specification clauses. The current PDF is a usable
   basic schedule, not a reproduction of the architect's golden presentation.
5. Profiler coverage summaries and architectural RFP/RFT projections; relevant
   facts must use the existing review path. No bulk finish fields were added to
   the profiler, and no source products were selected in Custom Home 2.
6. Calibrated import evaluation over both private golden PDFs and cross-sector
   fixtures, plus production-size/background completion budgets on the target
   host. The schema is shared across sectors; domain completeness for every
   commercial or industrial system has not been established by residential PDFs.

Concurrent profiler and styling files were preserved. The only shared-layout
change from this work is the project switcher's minimum-width correction in
`reports.css`; palette and typography were not changed.
