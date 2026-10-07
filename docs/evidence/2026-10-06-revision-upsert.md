# Revision counter round-trip reduction

Date: 6 October 2026. Lane A, existing speed-gate work.

Profile edits still exceed the 50 ms median budget in some local runs.
Inspection found two sequential revision statements: insert a default row if
missing, then increment one counter. This change combines them into one scoped
upsert while retaining the project advisory lock and caller's transaction.
It does not claim to explain the separate intermittent 700 ms delays.

The domain identifier still comes only from the six-name switch. The SELECT
still requires the org/project pair to exist. A new row starts its selected
counter at one and version at two, exactly as before; an existing row increments
that counter and version once. Other counters, creation time, rollback and
not-found behavior are preserved. No dependency, migration, model call or
product behavior is added. Edit/rebuild budgets remain 50/150 and 100/300 ms;
filing remains 1000/2000 ms.

The new domain-counter test passes on both the previous implementation and the
upsert. It covers row initialization, all six domains, repeated increments,
version preservation, wrong-org access and an invalid domain. Existing tests
cover concurrent writers, rollback after failed rebuild and freshness.
Store and HTTP API suites passed in 28.831 s and 13.371 s respectively.

The complete 20-file/two-round/40-sample replay with integrated proposals is
`bench/results/2026-10-06-revision-upsert.json`:

| Path | p50/p90 ms | Budget ms | Result |
| --- | --- | --- | --- |
| Filing | 454.411 / 689.786 | 1000 / 2000 | Pass |
| Spec Home edit | 50.502 / 53.670 | 50 / 150 | Fail |
| Integrated rebuild diagnostic | 89.086 / 95.202 | 100 / 300 | Pass |
| Package write | 2.089 / 2.680 | 100 / 250 | Pass |
| Work write | 7.386 / 8.208 | 100 / 250 | Pass |

The immediately preceding final-code run measured edits at 51.298/54.663 ms.
That difference is within observed run variation and does not establish a
reliable end-to-end speedup. The retained change eliminates one statement and
preserves behavior; the speed gate remains red. Extraction and Jev-admission
component overruns are also reported for target-VPS assessment. No live call or
release claim is made, and automatic proposal generation remains gated.
