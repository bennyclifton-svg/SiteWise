# Coordinated specification workspace — implementation evidence

14 October 2026. Lane A. Implements the approved
[workspace plan](../plans/2026-10-14-coordinated-specification-workspace.md).
This supersedes the first release's library-and-reader landing view.

## Behaviour and ownership

Specifications opens to a six-column schedule after Procurement. Sections use
the same catalogue IDs, names and building order as Project Profile. Empty
starter prompts contain no selected products or inferred technical requirements.
Known scope limits applicable prompts; explicitly excluded works do not restore
the generic framework. Commercial and industrial systems use the same model.

Users can add a line within a section, edit local cells, expand item details,
link applications to Profiler works, and preview or issue the package. Linked
location, action and inclusion read through from work items; provider reads from
the Profiler. The specification does not offer a competing scope editor.

Section and item coverage records identify extent, gaps, maintainer, coordinator
and referenced documents. References pin document ID, number, revision and hash.
Source replacement/removal is visible. Each item has at most one coverage record,
even after classification changes. External maintenance blocks local direct
edits and acceptance of local proposals. Previously accepted local values remain
in history for coordination and are excluded from issued requirements while
external coverage governs. Explicit adoption requires a reason and version.
Confirming a source reading does not select a product or transfer maintenance.

Update specification queues new, failed or outdated current documents in the
existing worker. Navigation and typing do not call Jev. Each candidate makes one
fan-out for item, kind, meaning, system and a bounded proposed match. Code copies
source spans and labelled values. Users confirm identity/readings or explicitly
map a candidate to an existing item. Retries preserve decisions and mapping;
neither a newer document nor a matching code silently wins.

Issues freeze the local rows, scope, coverage and reference metadata. Both PDF
and CSV include the consultant reference register, including a reference-only
issue with no duplicated item rows. PDF uses the existing text document renderer;
the browser's table layout is not yet the exported PDF layout.

Migrations 027–029 add coverage, defer scope-link checks until deletion cascades
finish, and enforce one item coverage record. Existing immutable issue/history
and tenant boundaries remain in force. No runtime dependency, service or AI
provider was added. Global shell styling remains under the concurrent UI work;
this surface inherits its semantic colours and Instrument Sans.

## Verification

- Store/domain checks cover shared ordering, explicit scope, coverage precedence,
  revision pins, frozen issues, foreign references, stale edits, external edit
  protection, Profiler changes, update deduplication and candidate mapping retries.
- API checks cover the new routes, foreign-org access and latency gates. Direct
  authoring/review/issue produces zero Jev calls.
- The core Go run passed the specification, store, jobs, database and other
  packages except the existing Profiler display-home check in `internal/httpapi`
  (`proj.flag.*` inputs without a home). One corpus-path check initially failed
  under the Windows temporary directory; it passes when TEMP/TMP points inside
  the repository. The final targeted specification/FK/API run passes with all
  migrations, including the item-coverage uniqueness guard.
- Retired starter prompts can be started again without resurrecting old source
  candidates. External assembly/component edits are protected at the store
  boundary; their old local relationships are retained but not issued as
  consultant requirements. The domain/store regression checks pass.
- Browser checks pass for inline creation/editing, locations, value review,
  immutable PDF/CSV issue, load recovery and consultant reference-only coverage.
  Desktop and 390px mobile captures show no page-wide horizontal overflow.
- The independent finish reviewer returned **ship**. Mobile navigation and stale
  surface documentation findings are both resolved. See the
  [current surface brief](../design/2026-10-14-specification-workspace.md).
- Three synthetic live calls to `jev-1.13.0` returned the expected outcomes:
  paint matched an existing interior-paint item; cooling equipment mapped to the
  mechanical starter as a capability; unrelated meeting text was rejected.
  Background provider latency was 269, 225 and 239 ms. This is a smoke check,
  not a calibrated semantic-accuracy evaluation or an auto-accept threshold.

## Timing

Windows development host, Go 1.27.1, local PostgreSQL. The full benchmark used
184 intake files in two rounds with recorded Jev latency. The schedule workload
grew to 131 synthetic items with three details each, issuing and fetching a PDF
after every addition. No budgets were relaxed.

| Path | Samples | p50 ms | p90 ms | Budget p50 / p90 ms |
|---|---:|---:|---:|---:|
| Specification read | 131 | 5.8 | 7.7 | 100 / 250 |
| Specification write/review | 1048 | 1.8 | 2.3 | 100 / 250 |
| Issue with PDF rendering/storage | 131 | 94.1 | 155.9 | 100 / 250 |
| Frozen PDF retrieval | 131 | 11.0 | 12.6 | 100 / 250 |
| Whole intake | 368 | 393.3 | 1280.5 | 1000 / 2000 |

The final 20-sample API gate separately measured inline cells at 2.3 / 2.8 ms,
coverage at 1.7 / 2.3 ms and Update requests at 1.1 / 1.3 ms. These measure the
request, not completion of background document reading.

The full benchmark is **not an overall release pass**: `profile_edit` p90 was
454.0 ms against 150 ms and `spec_home_profile_edit` p50 was 59.0 ms against
50 ms. Identity extraction and field-rule component overruns were also reported
by the harness for assessment on the target VPS. Concurrent work and tests ran
on this development host. The specification and filing gates passed; unrelated
Profiler paths and their budgets were not changed to hide failures.

## Limits and reproducibility

Source reading still requires human review of revision markings and visual
strike-throughs. Image crops, automatic application assignment, semantic
cross-page reconciliation and designed PDF tables remain outside this slice.
Specification import is resilient to Jev failure; manual authoring remains
available. The older full accuracy replays remain limited by stale/incomplete
recordings described in the first implementation report.

Local evidence: `tmp/specification-coordinated-bench-results.json`,
`tmp/specification-coordinated-bench-samples.json`,
`tmp/specification-workspace-live-check.json`,
`tmp/specification-coordinated-core-tests.log`,
`tmp/specification-coordinated-final-tests.log`, and the `specification-*.png`
and `specifications-*.png` browser captures. These private/local artifacts are
not product data and do not seed customer projects.

Run the browser checks with `npm --prefix web run build`, then
`npm --prefix web run test:e2e -- specifications.spec.ts`.
Go database checks require `SITEWISE_TEST_DATABASE_URL` pointing to the dedicated
`sitewise_test` database; use repository-local TEMP/TMP on this Windows sandbox.
Specification latency is enforced in `TestSpecificationAPIAndSpeedBudgets`;
the larger workload runs in `cmd/intake-bench` against `sitewise_bench`.

The local development preview was rebuilt and restarted with the additive
migrations. A read-only check of Custom Home 2 returned 13 building sections,
54 attribute definitions and the existing project scope. Its existing item was
preserved; no golden-reference or synthetic products were seeded into the project.
