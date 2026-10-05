# Local knowledge source archive

Date: 5 October 2026. Outcome: SiteWise knowledge validation and report-content research no longer require a sibling Clerk checkout. Lane A: provenance and reproducibility. This is the owner-requested prerequisite to the RFP, RFT and PMP content library.

## Decision

Preserve original seed and taxonomy data under `data/reference/clerk/`, with original relative paths and a SHA-256 manifest. Existing `{seed, anchor}` and `{clerk_file}` citations stay valid. The latter is a historical provenance key, not an external dependency. The checker defaults to the archive and validates identity before reporting its normal knowledge results.

Keep the original bytes, including source metadata and instructions, as archival evidence only. They do not govern SiteWise behaviour or confer review status. Do not remove citations after extraction. Derived building knowledge remains canonical in `knowledge/`; selected report wording passes through WP-40 and owner review. No new runtime catalogue or Jev question is introduced here.

The archive holds 51 Markdown seeds and 10 taxonomy JSON files. Per-file hashes were verified against the source working tree. Source Git metadata could not be read in this session, so the manifest records that limitation rather than inventing a commit identity. Source files are protected from Git newline conversion to keep hashes reproducible across platforms.

## Validation and failure handling

- `python tools/check_knowledge.py --strict`: zero errors; three existing warnings for unmapped `contamination_level`, `bushfire_exposure` and `flood_exposure` package-default fields.
- `python -m unittest discover -s tools -p 'test_*.py'`: 27 tests passed, including a temporary isolated SiteWise tree with no sibling Clerk directory.
- New regression cases reject missing, modified and escaping source paths and malformed manifests. The normal source checker continues to reject absent heading anchors and taxonomy files.
- `go test ./internal/knowledge` using the project Go executable and a workspace-local GOCACHE: passed. The initial attempt using the host's default cache failed on filesystem permissions; the workspace-local retry passed.
- Parsed delivery signal YAML matches HEAD exactly; only explanatory comments changed. No knowledge records were promoted to reviewed or verified.

No new dependency. No user-facing path, runtime source loading or model call changed, so no latency benchmark was run. Filing budgets remain p50 ≤1,000 ms / p90 ≤2,000 ms; no new timing result is claimed. The archive is used by tooling and content research, not sent to Jev.

Missing archive files, altered bytes or broken source headings fail knowledge validation. An explicit `--seed-dir` override remains for research fixtures and does not validate the bundled archive's hashes. An intentional refresh requires source-diff review, updated identity metadata and rechecking dependent knowledge; do not regenerate hashes to conceal unexplained edits.

## Integration boundary

The VS Code implementation agent should retain the checker default and archive checks when modifying knowledge shapes. The new source tests are automatically included by public CI's existing `test_*.py` discovery. No CI workflow change is necessary.

This task does not migrate private intake/profile corpora, replace their recordings, change the broader implementation sequence, or remove unrelated evaluation dependencies. Report-content review continues in `docs/content-review/2026-10-05-batch-01/`, whose source links now point to the archive. Delivery provenance is explained in `knowledge/clusters/delivery/README.md`.
