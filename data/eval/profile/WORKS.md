# Work extraction evaluation

`manifest.json` pins a bounded private-source baseline. Source documents
remain outside Git. `answer-keys/0991.yaml` is a draft, manually inspected
extraction key, not an owner-reviewed accuracy result. It covers only the two
pinned drawings and preserves their exclusion caveat. The 0777 design set does
not match its fit-out brief; no fit-out key is inferred from those drawings.

Validate the draft and local sources without scoring:

```powershell
go run ./cmd/works-eval -key data/eval/profile/answer-keys/0991.yaml -validate-only
```

Score an actual canonical export using `-actual <snapshot.json>` and
`-versions <expected-run.json>` instead of `-validate-only`. The expected run
file pins `model`, `question_version`, `knowledge_version`,
`thresholds_version` and `app_build`. The knowledge version must match the
loaded catalogue; all snapshot versions must match the expected run. Freeze
these pins before producing the actual export. Do not copy expectations into
an actual result or manufacture a recording.

The snapshot uses `eval.WorkSnapshot`: those same five version fields,
`project`, `key_sha256`, `corpus` (document ID to SHA-256), canonical `parts`,
and canonical `items`. Key SHA-256 is over the YAML bytes. Validation prints
the key and source hashes for use by the export pipeline. It does not print
private source text. Missing files, changed hashes, unknown fields, ambiguous
documents, path escapes and mismatched run pins fail closed.

Matches require the exact system, action and part label. Duplicate extra work
items count as false positives; missing items count as false negatives.
Excluded, retired and grouping rows are omitted. Undefined precision is null,
not a passing score. `-gate` exits unsuccessfully unless the key is reviewed
and both precision and recall are at least 90%. Only the owner may mark a key
reviewed. This gate checks extraction only: it does not certify critical
obligation coverage, proposal usefulness, or report quality.

No live Jev requests are made by this command. A production export adapter,
reviewed keys and the broader WP-28 quality gates remain separate work.

## 0991 timing fixture

`python tools/build_0991_bench.py` uses the existing Go key validator and
`pypdf` to write ignored `private/0991-bench.json`. It requires Go on PATH and
the document-tool Python runtime. The fixture contains the two pinned pages
and three **manually proposed** work items, with zero machine-read facts. Its
source annotations remain proposed. It is not a Jev output, owner-reviewed
scope, complete project corpus, or extraction accuracy result. Do not pass it
to `works-eval -actual`.

Measure it separately from Spec Home:

```powershell
go run ./cmd/intake-bench -measure-proposals -works-fixture data/eval/profile/private/0991-bench.json
```

The command rechecks source and key hashes and annotation semantics before
importing into its disposable benchmark organisation. Separate 0991 edit and
rebuild results use the existing 50/150 ms and 100/300 ms budgets respectively.
Proposal computation and rebuild-plus-diagnostic timing are reported
separately. This diagnostic excludes projection writes and cannot be combined
with `-release`. A full pipeline-derived 0991 fixture remains necessary before
claiming production workload or extraction-quality coverage.
