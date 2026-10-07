# WP-28: work extraction evaluation (partial implementation)

Lane A. Outcome: compare extracted canonical work items against pinned,
owner-reviewable expectations by system, action and part. This protects answer
trust without adding work to filing or profile edits. No service, dependency,
runtime AI call, UI or report generation is introduced.

## Implemented locally

- Offline `cmd/works-eval` and `internal/eval` scorer. Exact semantic matching;
  duplicates count as extra, abstentions as missing. Excluded, retired and
  grouping rows do not count as included physical work.
- Strict key/snapshot parsing, source file hash verification, relative path
  containment including resolved links, frozen key hash and expected run
  versions. The expected knowledge version must match the loaded catalogue.
- Extraction gate requires owner-reviewed keys and precision and recall each
  at least 90%. An empty key or undefined precision cannot pass. This is not
  the critical-obligation or proposal-usefulness gate.
- `0991` manifest entry and three draft expectations from two visually
  inspected, revision-A drawings: hydrant upgrade in Warehouse C and sprinkler
  replacement in Warehouses A and B. The key explicitly preserves the marked
  exclusion within Warehouse A. No private source prose or images were copied
  into tracked artifacts. These drawings form a bounded baseline, not a claim
  about the complete or latest project scope.
- Usage and input contracts in `data/eval/profile/WORKS.md`.
- `tools/build_0991_bench.py` creates an ignored, bounded manual-source timing
  fixture from the pinned drawings. It contains two whole-page text passages,
  three manually proposed work items and no machine-read facts. No source
  annotation is marked accepted or verified; no Jev result is manufactured.
- `intake-bench -measure-proposals -works-fixture <path>` validates the fixture
  against current source/key hashes and draft annotation semantics, imports
  into the disposable benchmark org, and reports separate 0991 timings. Edit
  and rebuild paths use the existing 50/150 ms and 100/300 ms budgets.

## Validation

`go test -p 1 ./...` passed with the test database enabled; output retained in
ignored `.tools/wp28-scorer-go.log`. Tests cover wrong part/action/system,
duplicate extraction, abstention, unreviewed keys, changed or absent files,
invalid paths, ambiguous input and stale run pins. Windows sandbox tests used
the repository-local `.tools` temp directory because resolving the default
external temporary root was denied.

`go run ./cmd/works-eval -key data/eval/profile/answer-keys/0991.yaml
-validate-only` validates the real local source hashes and draft key. It reports
`scored: false`; no extraction accuracy result has been produced.

This offline addition does not change a user-facing path. Latest runtime
baseline remains WP-23: filing p50/p90 328.4/929.6 ms (1000/2000 budget), profile
edit 40.8/66.0 ms (50/150), rebuild 43.6/45.4 ms (100/300). No new timing claim
is made for the incomplete 0991 fixture.

The later bounded manual-source benchmark run is recorded in
`bench/results/2026-10-05-wp28-0991-diagnostic.json`. With 40 samples, 0991 edit
p50/p90 is 10.266/11.001 ms (50/150 budget), rebuild 4.997/5.494 ms (100/300),
proposal compute 5.002/5.331 ms and rebuild-plus-diagnostic 11.789/12.520 ms.
The complete 184-file, two-round intake run passes existing user-path gates.
This is timing evidence for the explicitly manual, two-drawing baseline;
it is not an extraction quality score, live Jev run or full-corpus replay.

Focused fixture tests reject changed key/source hashes, wrong file/document
links, wrong actions, invented machine facts and a falsely labelled fixture
kind. The existing Go YAML parser is reused; the offline builder uses the
already available document-tool `pypdf`, with no runtime Go dependency added.

## Remaining work and gates

- Actual canonical snapshot export and a full pipeline-derived 0991 fixture
  remain. The bounded manual timing fixture does not establish extraction
  accuracy or coverage of the complete project record.
- The 0777 PMP describes an office fit-out, but the inspected CC-04 design is
  a residential development. It is excluded from a fit-out key pending source
  clarification; no empty or fabricated key substitutes for it.
- Owner review of keys, critical-obligation labels and the wider WP-28 quality
  gates remain. WP-28 is partial, not Verified, and nothing was merged.
- Earlier automatic approval review rejected sending private source/Hale
  passages to TypeSafe for live re-recordings without explicit payload and
  destination consent. That question remains pending; this command is wholly
  local and does not bypass that gate.
