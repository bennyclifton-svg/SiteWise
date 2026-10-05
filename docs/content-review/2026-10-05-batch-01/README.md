# Report content review batch one

Status: draft for Benny's review. Prepared 5 October 2026. User outcome: a reusable content inventory and the first reference RFP, ready for editorial review while the main application is implemented in VS Code. No runtime or knowledge-schema implementation is included.

## Review order

1. Review the private 0991 reference RFP in this chat's artifact folder. It is a proposed appointment document with unresolved inputs, not an issue-ready instrument or an answer key. Project-specific material stays outside the public repository.
2. Review `review-cards.md`: eight initial reusable wording candidates, with explicit applicability and exclusion examples.
3. Use `source-inventory.md` to choose the next drafting batch. `source-manifest.json` records all 51 seed and 10 taxonomy sources and their hashes.

Please comment by card ID or RFP section. For each, mark Keep, Edit, Conditional or Drop, with the boundary or missing deliverable. All items remain draft until you explicitly approve them. Editorial approval does not verify regulatory clauses, prices or project facts.

## Handoff to the VS Code implementation agent

This chat owns only `docs/content-review/2026-10-05-batch-01/` and its private review artifacts for this batch. This is a proposed coordination boundary recorded for you to read; no direct agent-to-agent acknowledgement has occurred. Existing work packages and runtime files remain yours.

Content preparation for WP-40a is underway separately. Reuse these drafts rather than independently drafting the same fire-services content. Please confirm the catalogue contract before integration: stable clause IDs/version/status/source; template sections and placeholders; conditional inclusion; unknown input behaviour; and how to represent role boundaries. The current `knowledge/SCHEMA.md` clause example does not specify all of these.

Do not load these Markdown review cards as runtime content. Once wording and the schema are agreed, the WP-40 owner maps selected IDs to the final YAML, records the mapping, runs the knowledge checker and marks only owner-approved content reviewed. Do not create a second catalogue or assembler. No edit here changes application implementation sequencing or permissions.

For the first demo, keep the consultant scope distinct from contractor construction duties and separately appointed fire-engineering/certifier roles. Unknown quantities, drawings, approvals, deadlines, attendance and fees remain explicit. Refer to plan D-31–36 and §8.6 for trust/quality gates.

## Independent evaluation

The reference RFP and draft cards are training/editorial material. They cannot become the held-out answer key merely by being copied. The owner or independent reviewer must create/review the expected critical-obligation list from underlying project evidence before a scored run; protect a contrasting evaluation case from drafting-time tuning.

## Boundary and validation

Only new editorial documentation is written. No changes to `knowledge/`, loaders, validators, Go, SQL, UI, thresholds or tests; no Jev calls, dependencies, commits or deployments. Runtime latency is unchanged and no runtime benchmark was run. Clerk is read-only. Source locators and hashes are checked; source truth and legal applicability are not thereby verified. Private corpus text and personal details are excluded from these repository files.

## Standalone source update

Source archival is now implemented separately from this editorial batch: `data/reference/clerk/` holds all inventoried seeds/taxonomies, and the knowledge checker uses them locally. `knowledge/clusters/delivery/README.md` explains delivery provenance. The implementation agent should preserve the new checker source resolution and manifest check; there is no new runtime dependency. The content-only boundary above describes the original drafting batch, not this separately authorised provenance migration.
