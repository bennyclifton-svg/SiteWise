# WP-25 follow-up: code-fed work types and existing-system predicates

Date: 5 October 2026. Lane A. Local changes on base/head
`80a5a4244623109b5da70b14859c303a46e056b8`; no commit, merge or deployment.
Plan: `ceb67518a03701a7df948fe2f384cce816e0e404`, D-07/D-10.

The works loader and most predicate support were already implemented in
`b3a1048`. This follow-up completes the recorded mixed-work-type decision
and existing-system semantics needed by WP-26. It does not implement ranking
or connect a proposal engine to edits.

`Catalog.Holds` accepts applied project and part work types separately from
arbitrary determinant values. A predicate on `work_type` matches any supplied
type. Missing or invalid taxonomy values remain unknown; caller maps are not
mutated. The caller remains responsible for supplying only applied values.

`Catalog.SystemExisting` combines site evidence and live included work items.
An action on an existing system establishes existence; removal or replacement
overrides that result. New work alone does not establish prior existence.
Coarse parent work is only possible evidence about a child, and partial child
removal cannot establish that the entire parent system disappeared. Unresolved
actions remain unknown. `system_present` continues using its separate input.
The caller filters retired, excluded and group items; WP-26 owns that adapter.

Verification:

- Existing loader tests pass, including unknown action/system/determinant/signal
  references and malformed predicates, with the source file in the error.
- New tests cover mixed project/part types, invalid input, caller-map integrity,
  removal/replacement precedence in both item orders, other systems, coarse
  scope, partial removal, unknown actions and unchanged system presence.
- `go test ./internal/knowledge` passes.
- The full Go suite also passes with the dedicated database after the bulk
  write correction; `.tools/wp25-go-full.log` records that run.
- `BenchmarkWorksCatalogueLoad`, 15 iterations: the entire catalogue loads in
  132.385 ms per operation on this Windows host. This bounds the works-layer
  contribution below the 200 ms added-startup budget; it is not a VPS result.
  Log: ignored `.tools/wp25-startup.log`.

No new dependency, service, migration, model call, threshold or knowledge
status change. This code does not alter hot-path wiring. The WP-23 handoff
records the profile-write and cached-query-plan corrections, final full-suite
pass and original workload pass (82.3 ms p90 for the full Spec Home edit).
Private profile replays remain stale pending explicit live-call
approval; this package cannot make those unrelated gates green.

Traceability: NW-REQ-030/056/057/144/145/173/174/261/264 have loader/predicate
support; NW-REQ-376 gains D-07's code-fed mixed work-type evaluation. Their
product consumers remain in later packages. NW-REQ-315 integration/review is
pending. No requirement is promoted to Verified by this local handoff.
