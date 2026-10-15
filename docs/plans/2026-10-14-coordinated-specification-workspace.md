# Coordinated specification workspace

Status: implemented, 14 October 2026. See the
[verification and limits](../evidence/2026-10-14-coordinated-specification-implementation.md).
Supersedes the earlier
item-library interface and its assumption that importing a document transfers
maintenance of its specification into SiteWise.

User outcome: open Specifications to a useful editable schedule, share the
Profiler's building hierarchy and work scope, reference consultant documents
without creating a competing specification, and fill gaps from direct input or
reviewable Jev readings. This reduces time to decision and preserves answer trust.
Lane A. Reads, writes and update requests: p50 <= 100 ms, p90 <= 250 ms;
issue creation uses the existing 100 / 250 ms gate. Filing stays 1000 / 2000 ms.
No new runtime dependency, service, AI provider, global styling or profile rewrite.

## Ownership and interaction

- Use the existing building-order catalogue for section IDs, labels and order.
  Starter rows are suggestions, never project scope or product selections.
  Profiler work IDs supply scope/action/location/provider by read-through.
- The agreed table is the landing view, with inline edits and expandable item
  details. Keep the new left navigation and position Specifications after
  Procurement. Inherit current colours and typography.
- Section/item coverage records distinguish needs attention, managed here,
  covered externally, partially covered and not applicable. Pin document identity,
  revision and hash with stated extent, exclusions, maintainer and reviewer.
  Coverage is a user decision, never inferred from upload alone.
- Document-derived records are source-linked readings. Confirming a reading does
  not select a product or transfer maintenance. Manual changes remain proposals
  while an external specification owns that item. Explicit adoption requires a
  reason and version check. A newer upload never automatically becomes authority.
- Update specification queues current project documents in the existing worker,
  reuses unchanged readings, proposes bounded item/category matches and keeps
  source spans. Neither navigation nor typing calls Jev. Match uncertainty creates
  a reviewable candidate, not a guessed merge or an invented product.
- Issues freeze local rows, scope and reference metadata; later document revisions
  flag live references for review without rewriting issued packages.

## Verification

Exercise wrong-org and wrong-project references, stale edits, duplicate imports,
source supersession/removal, partial coverage, external ownership, adoption,
linked work changes, empty projects, inline keyboard editing, failed reads and
Jev failure/retry. Reuse existing specification and filing tests and latency gates.
Verify the agreed table in the current shell at desktop and mobile widths.

Jev integration follows [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[one fan-out](https://docs.typesafe.ai/patterns/fan-out),
[confidence](https://docs.typesafe.ai/confidence) and the
[pinned API](https://docs.typesafe.ai/api). Proposed matches are not calibrated for
automatic acceptance. Specialist technical adequacy is not inferred by the model.
