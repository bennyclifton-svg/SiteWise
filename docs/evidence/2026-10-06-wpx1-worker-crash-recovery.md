# WP-X1: extraction worker process-crash recovery

Lane C. Outcome: a killed extraction worker must preserve the last complete
source and allow its leased job to recover without intervention. This supports
filing reliability and time-to-decision. No production code, dependency,
service or AI request changed. Filing keeps its 1000/2000 ms p50/p90 budget;
the deliberate lock and lease wait are not a performance benchmark.

`internal/jobs/crash_test.go` runs the real `jobs.Worker.Once` in an owned child
test process against the guarded `sitewise_test` database. Text comes from a
fixed local test callback. The parent holds a read-compatible lock on
`document_sources`; PostgreSQL must confirm the exact child application is
waiting at the source INSERT, after passage replacement inside the extraction
transaction. Only that child process handle is killed.

The test verifies:

- Before and after the kill, saved source, passage IDs/content, source units,
  profile facts, revisions and events equal the pre-crash snapshot.
- The first attempt remains durably leased until natural expiry. No explicit
  lease-expiration helper or clock manipulation is used.
- Normal job discovery finds the abandoned lease. A new process completes the
  same job ID on attempt two; another worker iteration is idle.
- Exactly two expected passages, two source units, one document source and one
  completion event exist. Extraction does not enqueue label/evidence work.
- The old lease token cannot complete the job or publish an additional event.

Validation on 6 October 2026: focused process test passes in 1.435 seconds;
the full jobs suite passes in 5.928 seconds
(`.tools/wpx1-worker-crash-tests.log`). `git diff --check` passes.

This is extraction recovery evidence. AT-21 mid-reading recovery is now
covered separately in `2026-10-06-wpx1-reading-crash-recovery.md`, including
cached results and profile projection. Report assembly recovery is also
recorded separately. M2 issue recovery and the semantic relationship review
remain outstanding; this does not claim WP-X1 or release readiness.
