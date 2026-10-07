# WP-41a report drafts — implementation in progress

Lane A. Outcome: a reproducible RFP draft assembled from saved project records,
with protected edits and explicit freshness. Budget p50 300 ms / p90 1000 ms.
M1 does not query costs, issue a report, render an export or generate AI prose.

Migration 020 defines reports, versions, edits and citation references using
tenant/project/report foreign keys. Issued versions and their child content
are protected by PostgreSQL triggers. Child mutations lock the parent version
before checking status so issue and editing cannot race past the guard. Issue
API remains unavailable until M2; the final schema is installed now per plan.

The migration runner now passes each embedded SQL body to PostgreSQL inside
the existing migration transaction. pgx's no-argument execution uses its simple
protocol; this preserves semicolons inside trigger functions and quoted text
instead of splitting them incorrectly. All database tests passed, including
scope backfill, isolation, report FK checks, draft org cascade, issued update/
delete refusal and issued edit/reference insertion or mutation refusal.

Pure report types, protected-edit application and static RFP dependencies are
implemented. Each generated block has a hash over content and source basis.
An edit survives changed input with a conflict marker; if the source disappears,
the wording remains in a visible review section. Duplicate targets are rejected
and input sections are not mutated. Freshness compares profile/works/packages/
delivery counters, profile fingerprint/revision, template, knowledge, question,
threshold and app versions. Report writes and M2 costs are not M1 dependencies.
Unit tests passed for edit preservation, removed-source conflicts and freshness.

The pure RFP assembler now fills the seven template sections from saved brief
values, physical scope/exclusions, the selected service package and its stages,
scope wording/deliverables, investigations, coordination, delivery dates/risks
and unresolved proposals. Draft clauses remain provisional and proposals remain
unaccepted. Fee rows request stage fees without inventing values or disclosing
budgets. Other packages' scope and dismissed proposals are omitted. Missing
brief values and sections are explicit. Pending/failed reading or a stale saved
profile requires explicit last-completed acceptance, with counts/status in the
draft. Retired packages cannot assemble.

Assembly tests pass for those trust boundaries and changed protected edits.
Content hashes now canonicalise JSON source key order without losing numeric
precision, so PostgreSQL JSONB persistence cannot cause false edit conflicts.
The profile reader has been extracted into a transaction-level helper without
changing its queries; a report snapshot reader gathers it with project/package,
scope, works, delivery, proposals, gaps and source versions in the caller's
transaction. Persistence/integration tests for that snapshot remain upcoming.
Report, store and HTTP suites passed after the profile-reader extraction
(`.tools/wp41-assembler-tests.log`); assembler tests also passed after explicit
scope-to-stage labels were added.

Report identity creation, saved draft refresh, read and protected-edit routes
are now wired. Refresh runs in a repeatable-read transaction under the project
lock, retrying serialization conflicts; source counters and generated sections
commit together. It reuses the draft ID and retains edits, while versioned edit
writes reject concurrent edits or intervening refreshes. The edit request's
`version` is the draft version returned by GET/refresh, not the edit-row version.
Pending/failed/stale reading returns a 409 with explicit wait/use-last-completed
choices; refused refresh leaves the saved draft unchanged. A retired package
prevents refresh but keeps the historical draft readable with a stale reason.

The report source state records VCS metadata when available and a once-per-
process executable hash; local modified builds cannot silently share a report
build identity. If executable metadata cannot be read, an explicitly unversioned
process identity prevents false freshness across restarts. This work is outside
the assembly hot path. Report/package references now block accepted-proposal
undo, as do report blocks that cite a created work item.

Relevant report/store/HTTP/latency/benchmark suites passed in
`.tools/wp41-store-api-tests.log`, including tenant isolation, edit conflicts,
pending-read rollback, saved identity, retired package behavior, strict request
fields and zero Jev calls. Dedicated report_read/write budgets are 100/250 ms;
report_assemble is 300/1000 ms. The route benchmark is wired for all three.

The reduced local diagnostic passed all user-path gates
(`bench/results/2026-10-05-wp41-report-diagnostic.json`): report read p50/p90
4.5/4.8 ms (40 samples), report write 2.9/3.6 ms (80), and assembly 7.6/8.5 ms
(40). Filing was 460.0/609.3 ms. This uses the basic report workload and 20
filing files over two passes, not the larger 0991 report acceptance workload or
release/VPS evidence. No Jev calls or dependencies were added.

The owner confirmed that undo retires only untouched, unused created records.
Saved report scope blocks now also prevent proposal undo, alongside work and
package references. The scope acceptance integration test verifies that a saved
draft blocks undo before any scope edit; after removing that test draft, an
independent edit also blocks undo. The targeted database test passed.

Remaining: citations (WP-42a), larger saved-corpus assembly evidence, further
concurrency/reference tests, protected-edit removal/conflict resolution UI and
M1 end-to-end security/review. No completion claim. No new dependency.
