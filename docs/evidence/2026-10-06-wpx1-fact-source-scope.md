# WP-X1: facts retain the correct source document

Lane C, with a product-core provenance invariant: a fact's source belongs to
its project, and page/location metadata belongs to its source document.
This protects answer trust. Scope is those relationships, not a new source
model or reading workflow. No dependency, service or Jev call is added.
Budgets remain profile edit 50/150 ms, rebuild 100/300 ms and filing
1000/2000 ms, p50/p90 respectively.

Two regression tests exposed distinct gaps:

- Separate tenant-scoped links on `profile_facts.project_id` and `document_id`
  allowed a fact to point to another project's document in the same tenant.
- Snapshot passage metadata was keyed only by passage ID. A mismatched
  historical locator could attach another document's page/location to the
  fact, even though the fact's document metadata was correct.

Migration `020c_fact_document_scope.sql` replaces the document reference with
`(org_id,project_id,document_id)` and adds the corresponding document identity.
Document deletion still cascades to facts. The pinned next-wave inventory now
contains 57 composite keys, with real document fixtures for the new sweep case.

The single-statement snapshot now joins passage ownership and indexes returned
metadata by both document and passage ID. Missing or mismatched locators leave
location blank while retaining the historical fact. There is no additional
database round trip. The stale-statistics fixture now includes real passage
ownership instead of source-location rows without passage parents.

Focused tests cover cross-project and cross-tenant documents, valid writes,
document-deletion cascade, matching and mismatched passage locations, missing
historical passages, and consistent snapshot behavior during a source
replacement. They pass. The stale-statistics snapshot measured p50/p90
9.510/9.993 ms; this is a component measurement, not the full edit gate.

The affected database, store, jobs and HTTP suites pass serially
(`.tools/wpx1-fact-scope-tests.log`). A subsequent run also passes
(`.tools/fact-scope-pretrace-tests.log`). Historical source IDs, immutable
provenance snapshots and generated typed proposal references must not be
mistaken for unconstrained live relationships in the remaining audit.

## Timing and unresolved tail anomaly

The unchanged full workload initially exposed a serious tail anomaly:
`bench/results/2026-10-06-fact-source-scope-diagnostic.json` measured rebuild
p50/p90 28.9/664.4 ms and Spec Home edit 52.0/687.9 ms. The first seven
edit/rebuild pairs were slow, after which timings returned to normal within
the same run. This is retained as failed evidence, not discarded as noise.

Two repeats with temporary slow-snapshot query-plan capture did not reproduce
the anomaly, including one after the original four-suite test sequence.
No snapshot exceeded the 100 ms capture threshold. Isolated probes with only
location statistics refreshed, only passage statistics refreshed, and passage
statistics from a previous tenant also did not reproduce it. A statistics-driven
plan change remains a hypothesis, not an established cause. The temporary
tracing and additional probe variants were removed; the source-location
statistics case remains in the existing stale-statistics regression test.

The final uninstrumented run is
`bench/results/2026-10-06-fact-scope-repeat.json`: filing 440.5/677.6 ms,
rebuild 30.3/33.4 ms, and Spec Home edit 52.9/56.7 ms. The rebuild and filing
budgets pass on this run, but the 50 ms edit median still fails. The gate exits
1. The earlier tail failure remains unexplained and must not be reported as
fixed. Local extraction/admission component overruns remain separately judged
on the target VPS. None of these runs is release evidence.

A later tail recurrence prompted a bounded, exact-ID lateral ownership lookup
in the same snapshot statement. The document-and-passage guard remains.
See [ownership lookup follow-up](2026-10-06-proposal-existence-and-source-streams.md)
for the rejected separate-stream experiment, final 48.0/50.4 ms edit timing,
and the still-unproven cause of the earlier tail failures.
