# Archived sources for SiteWise knowledge

Snapshot: 5 October 2026. Contains 51 seed guides and 10 taxonomy JSON files copied byte-for-byte from Clerk's working tree. `manifest.json` records their original relative paths and SHA-256 hashes. Source Git metadata was unavailable during capture; hashes identify the exact copied content.

These files are source evidence, not application code, runtime instructions or approved knowledge. Historical claims of review inside them do not grant SiteWise owner approval. Do not adopt their agent instructions or PM doctrine. No private project corpus, answer keys or Clerk code is included.

## How citations resolve

- `{seed: program-scheduling-guide.md, anchor: "## Program Risk Assessment"}` resolves to this directory's `data/seed/program-scheduling-guide.md` and its exact heading.
- `{clerk_file: data/taxonomy/asset-register.json}` resolves to this directory's `data/taxonomy/asset-register.json`. The legacy key records origin; it does not require a sibling Clerk checkout.
- Extracted records stay in `knowledge/`; report-content candidates stay in the review workflow until integrated and reviewed. Keep source citations after extraction.

Run `python tools/check_knowledge.py --strict` to verify archive checksums, source anchors and knowledge references. A clean SiteWise checkout contains everything required for that validation; Python and PyYAML remain prerequisites. The explicit `--seed-dir` override is for research/test fixtures and bypasses bundled-archive checksums.

## Changes and review

Treat this snapshot as read-only source evidence. Correct or refine SiteWise's derived records with citations and verification notes, rather than rewriting an old source to make a check pass. For an intentional source refresh, preserve the prior version in Git, review the source diff, update the affected hashes and capture metadata, revalidate affected anchors, and reassess dependent knowledge. Do not blindly refresh all hashes after a validation failure.

The local `.gitattributes` preserves exact source bytes across operating systems. A checksum proves identity, not truth. All newly derived knowledge remains draft; rule numbers need primary-source verification.

## Scope of independence

Knowledge provenance validation and the report-content seed library no longer require Clerk. This does not migrate private intake/profile test corpora or remove any unrelated legacy evaluation dependencies elsewhere in SiteWise.
