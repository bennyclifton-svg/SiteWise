# Source coverage document-range index — 7 October 2026

Lane A/C, no new dependency. Preserve source coverage semantics while reducing
profile response cost. Edit budgets remain p50/p90 50/150 ms. No asynchronous
work, counters, broad cache or weakened reading-state rules are introduced.

Before this change the stale-statistics fixture measured coverage at
8.348/9.018 ms p50/p90. Ordering passage IDs for evidence lookups alone measured
7.691/8.395 ms. Migration `021h_source_document_coverage.sql` adds the document
identity to each passage-source row, backfills it from the authoritative
passage, and enforces `(org_id, passage_id, document_id)` through a composite
foreign key. A compatibility trigger fills only omitted document IDs; supplied
wrong documents and foreign-organisation parents fail the constraint. The new
partial covering index scans non-pending outcomes by document, replacing
thousands of passage-ID probes. Evidence counts remain independent, bounded
by the document's actual passage IDs.

With the migration/query change, the same fixture measured 6.308/6.810 ms;
a second regression run measured 6.491/7.215 ms. The 6,000-passage fixture also
includes 3,000 evidence calls, missing outcomes, foreign rows sharing IDs,
superseded documents, and pending/mapped/background transitions. Counts remain
identical. These are component results, not a passing full-path latency claim.
The integrated benchmark is being run independently.

Migration backfill, legacy insert compatibility, wrong-document and wrong-org
rejection pass. The exact composite-FK inventory was extended by the single
changed source-parent constraint and the sweep passes. F31's stale-statistics
snapshot test and source pagination/isolation tests pass. Temp-table fixtures
explicitly seed the new authoritative document reference because PostgreSQL
`LIKE INCLUDING ALL` does not copy triggers.

Tradeoff: one UUID per source row, an additional parent uniqueness index and a
partial document-outcome index; source writes retain transactional integrity.
The older passage-ID covering index remains available to other source paths.


## Evidence-call follow-up (021k)

The same validated document reference now exists on `passage_calls`.
`021k_passage_call_document_coverage.sql` backfills from the authoritative
(org, passage), enforces the existing named passage FK over (org, passage,
document), fills omitted document IDs for legacy inserts, and adds a partial
(org, document) index for evidence calls. The coverage query no longer collects
or probes an array of passage IDs. No cached counts, new service, response
protocol or weaker budget was introduced.

Focused tests pass: backfill retains result/fingerprint; legacy inserts retain
independent label/evidence stages; incorrect document/org references and an
incorrect document update fail the exact composite FK; passage/document deletes
cascade; the 75-constraint reviewed FK sweep passes. Temp coverage fixtures pass
explicit document IDs because LIKE INCLUDING ALL does not copy triggers.

The stale-statistics coverage test now includes 6,000 passages, 3,000 evidence
calls, an additional 3,000 label calls on the same passages, foreign-tenant calls,
superseded documents and pending/outcome changes. Counts remain exact. p50
4.180 ms / p90 4.791 ms versus 021h's 6.308/6.810 ms. The independent upload
snapshot F31 test passes at 9.058/10.268 ms; source pagination/isolation passes.
These are component measurements, not the integrated speed gate.

SQLC was regenerated. The populated pre-M2 backup/restore/upgrade rehearsal now
covers 021k and passes (31 baseline + 12 new migrations, 42 original tables'
normalized counts/digests unchanged), including evidence-call backfill. Logs:
`tmp/passage-call-migration.txt`, `tmp/passage-call-coverage.txt`, and
`tmp/populated-upgrade-rehearsal.txt` (ignored).
