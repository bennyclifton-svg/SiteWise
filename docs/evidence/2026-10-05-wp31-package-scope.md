# WP-31 responsibilities and obligations — implementation in progress

Lane A. Outcome: record who holds each responsibility or general obligation,
with explicit wording and project-scoped references. This supports answer
trust and time to decision. Pricing and the gap-check UI are outside this
package. Endpoint budget: p50 100 ms / p90 250 ms.

Implemented locally:

- Migration 018 stores scope wording, roles, stages, interfaces, source
  snapshots and review provenance. Composite foreign keys prevent references
  to work in another project or a stage in another package. A live included
  role is unique within a package; overlaps across packages remain available
  to the future gap checker. Referenced records cannot be hard-deleted alone;
  organisation deletion still cascades atomically.
- Pure validation allows responsibilities or general obligations, requires
  exactly one exact-version catalogue clause or user text, rejects unknown
  interfaces and restricts supply packages to the supply role. Draft clause
  wording stays provisional; planning acceptance cannot approve it.
- Proposal undo now treats package scope references as use of the created
  work item or package. This follows the owner's explicit decision to retire
  untouched items and refuse undo when used or edited. Historical references
  also block undo.

Validation: procurement and knowledge unit suites passed. The migration test
passed against PostgreSQL, including wrong tenant, wrong package stage,
duplicate role, wording checks and organisation cascade. Store regression
tests passed for edited/referenced work, package scope use, package undo and
reacceptance. No new dependency or Jev call.

Scope store/API read/create/edit now runs under the project write lock with
optimistic versions, membership checks, strict request decoding and package
revision events. GET and POST use `packages/{pkg}/scope`; PATCH addresses
`scope/{scope}`. Reads/writes use the existing 100/250 ms package budgets.
New source selections resolve document and optional passage IDs in the same
project, copying authoritative hash, revision, location and text. Unrelated
edits retain saved snapshots after passage reprocessing. Store tests reject
foreign sources and forged metadata and preserve the snapshot after passage
deletion. No source client metadata is trusted.

Live scope blocks package/stage retirement and incompatible supply-package
conversion. Profile reset refuses referenced work; rebuild retirement skips
referenced work. The complete store, HTTP, procurement and database suites
passed (`.tools/wp31-scope-tests.log`); subsequent source snapshot and work
scope tests passed after the rebuild/reset guards were added.

Obligation proposals can now be accepted with an explicit `package_id` and
optional `work_item_id`, `role` and `stage_id` on the proposal accept endpoint.
The selected package is never inferred. A selected retained work item requires
a works package and `maintain_operation` or `protect`. Migration 018a anchors
`package_scope_item` decision targets through a tenant/project composite FK.
Scope snapshots keep draft proposal provenance visible after acceptance.
Concurrent retries create one record, changed assignments conflict, and retries
still work after projection removal. Undo retires untouched scope, refuses
edited scope and reacceptance revives the same ID. Future cost/delivery/report
references must extend these blockers when their tables are added.

The store, HTTP, database, latency and benchmark test suites passed in
`.tools/wp31-accept-tests.log`. The expanded proposal API test also passed.
Dedicated `package_scope_read` and `package_scope_write` paths use p50 100 ms /
p90 250 ms gates; the benchmark now exercises create, edit, read and retire.

Local diagnostic benchmark passed all user-path gates:
`bench/results/2026-10-05-wp31-scope-diagnostic.json`. Scope reads measured
p50/p90 0.6/1.1 ms (40 samples); scope writes 2.1/2.3 ms (120 samples), against
100/250 ms. Filing measured 465.9/703.6 ms against 1000/2000 ms. This is a
20-file-per-round, two-round development diagnostic using recorded latency,
not full-corpus or release/VPS evidence. Proposal acceptance/undo timing is
still separate outstanding work; these scope CRUD measurements do not cover it.

Remaining: expanded retain/reference regression tests and proposal endpoint
timing;
WP-31 is not complete. Group inheritance follows WP-24's M2 integration.
