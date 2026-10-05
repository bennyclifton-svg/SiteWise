# Delivery knowledge and its sources

`signals.yaml` contains drafted evidence questions. `unforeseen.yaml` contains drafted delivery-related unforeseen-condition records. Source citations explain where those records came from; they are not instructions to perform extraction later.

Seed filenames resolve to `data/reference/clerk/data/seed/` within SiteWise. The source archive's manifest preserves their identity, and `python tools/check_knowledge.py --strict` checks hashes and exact heading anchors without a Clerk checkout. Dataset citations resolve to SiteWise's existing `data/unforeseen/` manifests and rows.

Keep citations when records are reviewed, corrected or used in reports. A draft question does not prove that a project meets an obligation, and a cited seed does not verify a regulatory claim. This migration changes source resolution only; question text, IDs, criteria, applicability and review statuses remain unchanged.

For RFP, RFT and PMP content, reuse these canonical record IDs where relevant. Put reusable wording through WP-40 and the owner-review workflow; do not turn this cluster into a second report-template catalogue. Initial editorial work is in `docs/content-review/2026-10-05-batch-01/`.
