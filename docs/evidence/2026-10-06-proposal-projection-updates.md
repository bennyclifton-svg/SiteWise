# WP-26: update proposal projections without replacing unchanged links

Lane A/C. Keep proposal review responsive while preserving decisions, source
reasons and exact work-item references. Scope is persistence of the existing
projection; runtime generation, new proposal kinds and interpretation of
missing targets/actions remain out of scope. Budgets remain profile edit
50/150 ms, rebuild 100/300 ms and filing 1000/2000 ms (p50/p90). No service,
schema or dependency was added.

The prior trace localized substantial cost in deleting and recreating the
projection and committing its foreign-key work. An initial selective-delete
implementation compared complete rows and trigger IDs. It passed correctness
tests but measured 139.861/149.161 ms for the persisted diagnostic. A short
trace explained why: after changing subclass, only seven of approximately
400 proposals retained identical complete content. That version was rejected.
Its diagnostic remains in
`bench/results/2026-10-06-proposal-selective-projection.json`.

The retained implementation sends one batch within the existing project-locked
transaction. It inserts new proposals and updates changed columns in place,
using a full row comparison to avoid identical updates. It separately removes
obsolete trigger links, inserts missing links, and removes obsolete proposals.
Rank or label changes therefore preserve existing trigger rows. The comparison
includes every mutable proposal column; semantic fingerprints alone would
miss a replacement work-item ID. Durable decisions remain independent of the
projection and are applied before writing, as before. Any failed constraint
rolls back the complete transaction.

The new regression first failed against replacement-by-delete, showing that
identical rebuilds rewrote proposal and trigger rows. It now verifies physical
row retention, full output equality, stale label and rank repair, missing-link
repair, and replacing a parent trigger with a child while retaining the
proposal key and semantic fingerprint. Unrelated rows and rank-only trigger
links retain their row identities. Existing proposal tests cover decision
reopening, retry after knowledge removal, concurrent acceptance, foreign actors
and failed-projection rollback. Focused tests pass.

## Full-workload measurement

`bench/results/2026-10-06-proposal-upsert.json` uses the unchanged frozen
workload, 20 files in each of two rounds, 40 API samples and
`-measure-proposals`. It measures:

| Path | p50 ms | p90 ms |
|---|---:|---:|
| Spec Home edit, without automatic proposals | 49.5 | 52.4 |
| Profile rebuild, without automatic proposals | 29.0 | 31.2 |
| Proposal input preparation and evaluation | 26.182 | 29.974 |
| Rebuild plus explicit persisted proposals | 113.452 | 124.401 |
| Filing | 419.0 | 632.3 |

The persisted median improves from the preceding 127.172 ms trace but still
exceeds 100 ms. Its maximum sample was 166.429 ms. The command's ordinary-path
gate passed, but diagnostic proposal timings do not become passing gates as
a result. This operation still repeats input reads/reconciliation and uses a
second transaction; it does not establish integrated-edit timing. Runtime
proposal generation stays disabled. Owner-review, live and release gates are
unchanged. The earlier unexplained profile tail anomaly is not claimed fixed.

The complete serial `go test -p 1 ./...` run passes
(`.tools/proposal-upsert-all-tests.log`), including the access-control sweep,
process-crash recovery tests, report/decision behavior and the new projection
tests. Store took 27.488 seconds, HTTP 13.607 seconds and jobs 9.545 seconds.
`git diff --check` passes. Temporary comparison/stage tracing and the rejected
comparison helper are absent from production source.
