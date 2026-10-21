---
target: Builder quote comparison for Alterations and Additions
total_score: 20
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
timestamp: 2026-10-19T10-02-57Z
slug: web-src-prices-tsx
---
Method: dual-agent (A: /root/design_review; B: /root/evidence_review), plus source-PDF review by parent.

The current comparison collects the offers successfully but does not yet provide a dependable basis for selecting a builder. The running app reports that its UI build is behind the code; live findings describe the served build.

Design specificity: the compact workbench and side-by-side firms suit SiteWise, but consultant fee terminology and rates do not suit residential building tenders. This is primarily a comparison-model and workflow problem.

Strengths: explicit unknowns, ex-GST basis, inspectable source excerpts, direct corrections, Excel export, and source-code support for award preview and undo. No winner was falsely highlighted in the observed incomplete table.

Priority issues:

1. P1: Reconcile totals before comparison. Nexus has Not read for its lump sum while an extracted total sits under Offered, not asked for. Its PDF page 11 states subtotal $191,136.71, GST $19,113.68 and inclusive total $210,250.39. Enmore states $228,000 including GST; the table's $207,272.73 ex GST conversion is reasonable. Kaposi states $182,417 excluding GST then $200,658.70 after 10% builder margin; margin must remain distinct from GST. Preserve printed amounts and separately label calculations and rounding discrepancies.
2. P1: Preserve relationships between amounts. Kaposi's 26 loose entries mix components, stage subtotals and an overall subtotal. Enmore's allowances are labelled Offered, not asked for, although absence from a generic schedule does not establish they are extras. Distinguish quoted total, included breakdown, allowance, optional extra and unresolved relationship; never add a total to its components.
3. P1: Use a builder comparison basis. Lump sum fee, Reimbursables and five consultant hourly roles are inappropriate defaults. Compare building scope, owner supply, allowances and exclusions. Included without a separate price must differ from Not stated, Excluded and Not read. Confirm scope against project documents; residential/house/extension alone does not establish quantities or inclusions.
4. P2: Make the next action review. Show unresolved total/scope counts per firm and Review issues before prominent Award. Existing award preview is useful; this critique does not claim the final award is unguarded. Keep quoted totals separate from user-approved comparison adjustments and final award values.
5. P2: Reduce reconciliation effort. Clean firm names, retain filenames secondarily, show source excerpts in a read-only side panel, move legend near the table, collapse raw candidates and remove the disconnected composer from this work area. Use aligned scope rows instead of three unrelated lists.

Evidence examples: Nexus excludes switchboard upgrades while Enmore includes a new meter box; Kaposi carries an air-conditioning provisional sum; Enmore lists ducted and split systems with inclusion needing clarification. Nexus owner-supplied PC items and Enmore allowances need explicit scope comparison. Enmore gives 26 weeks then refers to 22 weeks in delay wording. These are clarification prompts, not automatic corrections. Original PDFs: D:/AI Projects/Test Data/house-reno-caves/Enmore.pdf, Kaposi.pdf, NexusBuilt.pdf.

Suggested workflow: select building-work package, upload all quotes, review identity/price basis, review aligned scope differences, record clarifications, compare, then preview award. Keep setup short; show existing quoted total immediately with review status. No scope or rates should be invented to fill blanks.

Heuristics (0-4, observational):

| Heuristic | Score | Main finding |
|---|---:|---|
| Status | 2 | Received does not explain readiness |
| Real-world fit | 1 | Consultant schedule for builders |
| Control | 3 | Corrections and award preview supported |
| Consistency | 2 | Mixed identities and labels |
| Error prevention | 2 | Totals and components mixed |
| Recognition | 2 | Source relationships must be remembered |
| Efficiency | 2 | Many manual candidate moves |
| Minimalism | 2 | Long unaligned lists and unused composer |
| Recovery | 2 | Correction tools lack prioritised guidance |
| Help | 2 | Legend follows long table |
| Total | 20/40 | Material workflow weaknesses |

Cognitive load: high; Kaposi's 26 independent candidates require arithmetic and source relationships to be held in memory. Emotional journey goes from reassurance at three columns to uncertainty over incomplete totals, then premature invitation to award. A busy PM risks mistaking extraction completion for comparison readiness; a builder risks treating a subtotal as an extra.

Minor observations: procurement defaults to Design & Construct when unanswered; nine suggested consultants precede Main Works. Profile shows two quotes included and one excluded from reading. Offer content should not silently become accepted project scope. Upload dates should be distinguished from quote dates; these PDFs include 2021 dates.

Detector: Prices.tsx scan returned zero findings. This does not validate monetary semantics or usability. Both independent assessments agree on the live workflow issues. No overlay was injected because browser evaluation is read-only.

Questions for the next design decision: should the next change repair the existing comparison first or design a builder-specific comparison workflow? Should scope rows start from confirmed project scope with quote-only additions clearly marked? No implementation is authorised by this critique.
