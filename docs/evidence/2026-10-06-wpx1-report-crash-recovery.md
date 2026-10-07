# WP-X1: report assembly process-crash recovery

Lane C, with a product-core invariant: an interrupted assembly must not replace
the last committed draft with partial sections, references or audit state.
This reduces recovery time without sacrificing confidence in saved reports.
Scope is M1 first assembly and draft refresh; issue/export recovery follows
their M2 implementation. No production code, dependency or service changed.
The assembly budget remains p50 300 ms / p90 1000 ms; deliberate test blocking
is not a latency measurement.

`internal/store/report_crash_test.go` starts an owned child of the Go test
executable, opens the existing `sitewise_test` database, loads the catalogue
and calls the production `Store.RefreshReport`. It does not recreate or erase
the parent's fixtures. The normal test database name guard applies to both
processes.

The parent holds a table lock permitting reads of report references but
blocking their replacement. PostgreSQL activity must confirm the child's
exact application name is waiting on `DELETE FROM report_references`. That
statement follows the actual draft INSERT/UPDATE, so the crash occurs after
assembly has made an uncommitted write. The test kills only its own child
process handle. This is neither graceful cancellation nor a simulated error.

For both an empty report and an existing protected draft, the test proves:

- Partial assembly is invisible while the child is alive and blocked.
- After process death, barrier release and backend disappearance, all stored
  report fields, versions, references, edits, revisions and events equal the
  pre-crash snapshot.
- A fresh child process successfully retries against the same saved state.
- Recovery publishes seven complete sections and references, with exactly
  one draft. Refresh retains draft identity, advances its version once, and
  preserves the protected user text.

Validation on 6 October 2026: the focused process tests pass in 1.864 seconds;
the full store suite passes in 19.183 seconds
(`.tools/wpx1-report-crash-store-tests.log`). No live Jev call occurs.

WP-X1 remains incomplete: semantic review of relationships without foreign
keys and M2 issue recovery are still required. Mid-reading recovery now passes;
see `2026-10-06-wpx1-reading-crash-recovery.md`.
Extraction worker process recovery now passes; see
`2026-10-06-wpx1-worker-crash-recovery.md`.
This is local crash evidence, not a deployment, restore or target-VPS claim.
