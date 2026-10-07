# Independent Spec Home edit gate

The previous benchmark pooled profile edits from different workloads. It recorded
`spec_home_profile_edit` separately but did not give it a budget or verdict. A
passing aggregate therefore did not prove that the saved Spec Home workload met
the 50/150 ms edit budget. This is a gate defect, not a reason to relax the target.

The committed budgets now gate that workload independently at the same 50/150 ms.
The results writer retains its computed pass/fail verdict instead of overwriting
it with an unjudged summary. A regression using a fast aggregate and slow Spec
Home samples failed before the fix and passes afterward. Running the actual gate
against `.tools/wp45-work-edit-samples.json` now correctly exits 1 for Spec Home
p50 88.491 ms. The larger-workload speed requirement remains unmet.

Temporary stage timings found about 32 ms in coverage reads, compared with about
3 ms in computation. A project-scoped aggregation rewrite was tested. The first
attempt repeated grouping under stale statistics; materialising the aggregate
fixed that local regression (23.1/23.5 ms in the existing stale-statistics test).
However, the full Spec Home edit remained 90.6/93.4 ms. The rewrite was discarded.
No runtime query change or timing probe is retained.

Diagnostic records are `.tools/wp45-profile-trace.log`,
`.tools/wp45-coverage-plan.txt` and `.tools/wp45-coverage-candidate.json`. The
first narrow trace command completed the profile measurements but stopped later
because one uploaded file could not supply 40 deletion samples; it is timing
instrumentation only, not a complete benchmark result. The candidate used the
normal 20-file/two-round workload and correctly failed the new independent gate.

No new dependency, service, runtime AI call or schema change. Optimising the real
edit path and meeting the gate remain required before claiming this speed target.
The store, HTTP API, latency and benchmark test suites pass after reverting the
candidate (`.tools/wp45-profile-gate-tests.log`). Temporary timing probes are gone.

## Batched coverage and direct fingerprint encoding

A subsequent query uses each document's passage-ID array to count outcomes and
evidence calls in batches. The existing stale-statistics regression passes at
8.5/9.3 ms p50/p90 with exactly the expected counts, including missing source
rows, superseded documents, empty documents and foreign-org evidence calls.
The full frozen Spec Home edit improved to 65.7/68.4 ms in the coverage-only
diagnostic (`.tools/wp45-coverage-batched.json`), still above the p50 target.

Fingerprint construction now encodes rows directly rather than encoding a
whole slice and decoding it back to raw rows before sorting. A compatibility
test checks the previous canonical bytes, including null slices, raw JSON
whitespace and decimal formatting, escaping and ordering. Invalid raw evidence
still fails. The synthetic 1,153-fact comparison measured 10.1 ms / 13.3 MB for
the old round trip versus 5.9 ms / 5.0 MB for direct encoding. This is not a
full-route timing claim. Original input classes, version exclusion and order
independence remain covered by the fingerprint tests.

Store, API, latency and benchmark suites pass with both changes
(`.tools/wp45-profile-batching-tests.log`). These changes add no schema, service
or external dependency and retain the same source coverage and fingerprint
contracts. The budget is unchanged.

The combined full diagnostic is
`bench/results/2026-10-05-profile-batching-diagnostic.json`: Spec Home edit
p50/p90 65.185/67.167 ms (40 samples), rebuild 42.314/44.317 ms, filing
478.002/740.784 ms. The independent edit gate correctly exits 1 because p50
still exceeds 50 ms. The full-route gain is predominantly coverage batching;
the fingerprint microbenchmark gain must not be represented as an equivalent
end-to-end reduction. Component release gates also remain unproven on this
local replay run. Further edit-path optimisation is required.

## Single-statement source matching

Batching source metadata within another SQL join failed the stale-statistics
regression (90.3 ms snapshot median); that attempt was removed. The retained
change returns tagged fact, document and passage rows from one SQL statement,
then matches the metadata in Go maps. Each distinct metadata record is returned
once. It preserves the statement snapshot and org/project filters without
depending on join cardinality estimates. Facts retain their original ordering,
including missing passage IDs last within a document.

The existing regression passes at 7.9/8.5 ms for the snapshot. It checks stale
statistics, missing passages, document metadata, foreign-org source IDs and a
replacement immediately after the fact query, proving that source location does
not come from a later database state. The full Spec Home edit is now 59.4 ms p50
in `bench/results/2026-10-05-profile-snapshot-batching-diagnostic.json`; the
independent 50 ms gate still fails. No weaker gate or smaller fixture was used.

The full serial Go suite passes (`.tools/wp45-snapshot-all-tests.log`). A later
focused fingerprint regression also passes: work edit timestamps are excluded
from semantic fingerprints, matching the existing version/timestamp exclusion
contract, while changed actions and authors remain detectable. The input item
and its saved audit time are not mutated. Final measured edit p50/p90 is
59.399/61.122 ms and rebuild is 36.913/39.230 ms; the edit target remains open.

## Typed work-item reads

Work-item reads now scan the stored columns directly, decoding only the target
and provenance JSON. Previously PostgreSQL encoded the complete row as JSON and
Go decoded it again. Quantities still cross the boundary as decimal text, so
there is no floating-point conversion. The correction/rebuild integration test
compares the result with the previous complete-row representation, including an
18-digit quantity with nine decimal places, source evidence and correction audit.
The store and API suites pass (`.tools/wp45-work-read-tests.log`).

The same 40-sample Spec Home workload measured edit p50/p90 57.332/60.931 ms
and rebuild 35.352/38.059 ms in
`bench/results/2026-10-05-profile-work-read-diagnostic.json`. Work reads measured
0.618/1.069 ms, writes 7.209/8.212 ms and filing 451.152/600.511 ms. The
independent edit gate still correctly fails its 50 ms median budget. Local
component extraction/rule timings also exceeded their separately reported VPS
budgets; this run is not release evidence.

A subsequent short diagnostic measured the remaining rebuild stages: roughly
10 ms for snapshot/fingerprint inputs, 2 ms computation, 4–6 ms work
synchronization/read, 7–8 ms fingerprinting and 9–10 ms projection writes.
It used ten samples to localize cost, so its insufficient-sample gate result is
not a validation result. The trace is `.tools/wp45-stage-trace.log`; all temporary
`[DEBUG-wp45-stage]` probes were removed. No schema, dependency or service was
added. Further optimization of the complete edit path is still required.

The complete serial Go suite also passes after typed reads
(`.tools/wp45-work-read-all-tests.log`). Later benchmark-harness and latency-gate
changes were validated separately after adding proposal endpoint coverage.

## Overlapping immutable hashing with projection writes

The measured hash stage and projection write stage have no data dependency.
The rebuild now hashes the captured immutable snapshot on one goroutine while
the calling goroutine writes the projection. Only the caller uses the database
transaction. A wait on every exit joins hashing before the rebuild returns;
the build record and event are written only after hashing succeeds. Hash
encoding, source inputs and the transaction boundary are unchanged. No cache,
dependency, schema or runtime AI call was added.

Store and HTTP suites pass (`.tools/wp45-hash-overlap-tests.log`). A forced COPY
constraint failure after projection deletion proves that the previous rows,
fingerprint, timestamp, revision and event count survive unchanged; the next
rebuild succeeds with the same semantic fingerprint and exactly one additional
revision (`.tools/wp45-hash-rollback-test.log`). This checks the early-return
path as well as ordinary builds. No race-detector run is claimed.

The unchanged 40-sample Spec Home workload measured edit p50/p90
50.680/52.186 ms and rebuild 28.052/29.597 ms in
`bench/results/2026-10-05-profile-hash-overlap-diagnostic.json`. This improves
the edit median from roughly 57 ms but still fails its 50 ms limit. The gate
correctly exits 1. Local component overruns remain separately reported for
target-VPS judgement; the run is not release evidence.

A follow-up removes the second site-ID lookup from projection writing: the
same project-locked transaction already populated `snap.Site.ID` in its
fingerprint inputs. Profile/rollback tests pass
(`.tools/wp45-profile-site-reuse-tests.log`). The full-path measurement is
`bench/results/2026-10-05-profile-site-reuse-diagnostic.json`: the edit median
is 50.929 ms and still fails 50 ms. Removing the redundant query did not show
an end-to-end improvement beyond run variation; it retains the already resolved
site identity without another database round trip. The material measured gain
in this iteration is hash/write overlap. Both diagnostics retain the same
fixture and gate. Further improvement is required; neither a marginal miss
nor a fast rebuild substitutes for the complete edit-path budget.

## Rejected earlier fingerprint preparation

A follow-up experiment encoded the immutable evidence components while work
items synchronized, then finished the canonical hash alongside projection
writes. Store and API tests passed, including hash-format compatibility and
rollback. The unchanged 40-sample full workload measured edit p50/p90
52.299 ms / 56.003 ms (`bench/results/2026-10-05-profile-prepare-diagnostic.json`).
It did not improve the full path, so the two-stage preparation was reverted.
The prior hash/write overlap and site-ID reuse remain. The canonical-hash
compatibility test is retained. The edit budget remains unmet; this diagnostic
does not authorize enabling full proposal evaluation on the edit path.

After reverting the experiment, the complete serial Go suite passes
(`.tools/profile-final-all-tests.log`). No new dependency was added.
