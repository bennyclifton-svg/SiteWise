# Source documents to cost plan and procurement: design

Date: 21 October 2026. Status: design agreed with the owner, nothing built.
Origin: peer review of the Meridian Fitout test project (six Chen Residence
files from `../clerk/data/synthetic-mobilisation-evidence/chen-residence`).
The profile read the $1,850,000 construction budget, but the cost plan
showed no figure, and the signed architect engagement sat in "2 proposals
need a package".

## Why

Reduce friction from upload to a usable cost plan and procurement table.
Every figure a source document states should land where a project manager
expects it, as a proposal they can correct in one move. Serves
time-to-decision: a cost plan with an identified line for every element and
a budget figure on each is something a PM can scan and adjust. An empty grid
gives them nothing to scan.

## Owner decisions (21 October 2026)

1. **Spread the stated construction budget** across the construction lines
   by a fixed new-house split, as soon as the budget is set. The PM adjusts
   the few that look wrong. **Edited lines do not rebalance the others.** The
   difference against the budget is shown instead.
2. **The budget lives in the cost plan's totals area** (bottom of the plan,
   where all totals live) **and in the Profile**. It is not a line.
3. **Procurement: no "suggested" state.** Default consultant rows become
   real packages, so the Consultants section of the cost plan lists each one
   (unpriced) from day one. The bin removes any not needed.
4. **Remove the Engaged by column.** Every consultant counts as engaged by
   the client. The architect RFP stops asking for it.
5. **Firm 1, Firm 2, Firm 3 placeholders** on every row, so the row reads as
   "this is where firms are compared".
6. **Bin icon at the far right** replaces the ⋯ menu and its Remove button.
   The existing removal confirmation (linked draft cost lines) stays.
7. **A fee proposal maps itself:** discipline row, firm slot, fee. **An
   engagement letter (accepted) awards automatically**. A wrong award is
   undone with the existing Undo.
8. **Risks a document names** (rock in TP-2, Sydney Water build-over,
   off-site tree) become unpriced lines in the cost plan.

## Part 1: Procurement table (lane B)

- `Procurement.tsx`: drop the `suggested` tag and Engaged by column. Firms
  cell becomes three slot columns (Firm 1-3). Chips fill them in order, a
  fourth or later firm shows as "+n" in Firm 3. Empty slots read "Drop a
  proposal". Bin button in the last column. Scope link moves beside the
  discipline name.
- Default services rows are created as packages when the profile's
  procurement defaults resolve (the existing create path, run for each
  default instead of on click). A dismissed default stays dismissed (the
  existing dismissal table), so a binned row does not come back.
- `engaged_by` stays in the schema (no migration). The RFP treats an
  unanswered row as client-engaged.
- Budget: `packages_read`, `packages_write` p50 ≤ 100 ms, p90 ≤ 250 ms.

## Part 2: Budget spread and brief figures (lane A)

**Reading.** In the existing profile reading fan-out (one per document, in
the background after filing), the harvest already finds money spans
(`budgetCandidates`). Add one question: for each money candidate, which is
it? Construction budget, contingency, owner-supplied allowance (with the
item), consultant fee, or none of these. Go parses the amount and the GST
basis ("excluding GST", "incl. GST", else *not stated*). Same fan-out, no
extra round trip ([how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one),
[patterns](https://docs.typesafe.ai/patterns)). Thresholds are fitted on an
answer key per [confidence](https://docs.typesafe.ai/confidence); until
fitted, readings stay amber.

**Where each figure lands:**

| Reading | Cost plan |
|---|---|
| Construction budget | Totals area: Budget, Allocated (sum of construction lines), Difference. Profile shows the same figure. Typing in either updates both |
| Contingency | Construction contingency line budget |
| Owner-supplied allowance | A line per item ("Owner supply: appliances"), outside the construction budget |
| Exclusions stated with the budget | Text on the Exclusions line |

**Spread.** When the construction budget is set (accepted reading or typed),
Go allocates it over the prepared construction lines by the split in
`knowledge/costs/budget_split.yaml`. The split is new-house only to start,
`status: draft` and `verified: false`. Each share needs a citation to a
published residential cost analysis. The seed docs in `../clerk` have none,
so the first draft is for owner review. Rules:

- Shares of elements the project does not have are shared across the
  remaining lines in proportion. Cents are rounded in Go and the residual
  goes to the largest share, so the allocation equals the budget exactly.
- Only lines with an unknown budget are filled. A user's figure, an awarded
  contract value and a frozen baseline are never written.
- Each value is `origin: calculation`, `review_status: proposed`, with the
  rationale "14 % of $1,850,000 construction budget (owner brief, draft
  new-house split)" and provenance to the source passage.
- Editing a line does not move others. A changed budget re-spreads only
  lines still holding an unedited proposal from the previous spread.
- GST basis not stated: the totals area says so. Nothing is converted.
- Budget: `costs_prepare` p50 ≤ 250 ms, p90 ≤ 500 ms. `costs_read`,
  `costs_write` p50 ≤ 100 ms, p90 ≤ 250 ms. The spread runs in the write
  transaction and never calls Jev.

## Part 3: Fee proposals and engagement letters (lane A)

- Any document the intake files as a fee proposal or engagement letter (from
  the register or dropped on a row) opens a submission. Rules settle
  discipline → package (the architect row), firm (printed name + ABN). The
  existing one proposal fan-out reads the fees. "Needs a package" remains
  only when rules and Jev cannot settle a discipline.
- **Acceptance** is decided by rules first: "accepting our fee proposal",
  a signed ACCEPTED block, "letter of engagement". Jev answers only when the
  rules leave it open. Accepted → `store.Award` runs with the firm's priced
  cells (stages as award lines), exactly as a PM award would. It never
  writes a budget. Undo is the existing `UndoAward` (refused once invoiced).
- Same firm in a fee proposal and its engagement letter: the letter
  supersedes the proposal (existing same-firm supersession).
- Stage sums must equal the stated total. Otherwise the award is not run and
  the row shows "Stages do not add to the stated fee". Example: the CA stage
  "Monthly × est. 12 months $8,000" adds up only as a total.
- Budget: `proposal_read` p50 ≤ 3 s, p90 ≤ 6 s (background). `award_write`
  p50 ≤ 200 ms, p90 ≤ 500 ms.

## Part 4: Risk lines (lane A)

A passage that names a cost risk ("allow for excavation difficulty in cost
plan", "build-over likely requires Sydney Water assessment") becomes a child
line under its element (by the system the passage maps to). Its budget is
unknown and it carries a Risk tag and the source sentence. The spread does
not give it a share, so it sits inside its parent's allowance until priced.
Read in the same profile fan-out as Part 2.

## Out of scope

Rates or quantities, any priced estimate other than the budget split,
commercial splits, invoices, variations, program.

## Done

Meridian Fitout re-read from its six files shows, without typing:

- the Budget $1,850,000 in the totals area and in the Profile, with
  "GST not stated"
- construction lines filled from the split
- contingency $120,000
- owner supplies $8,000 and $22,000
- Harrison Clarke Studio in the architect row, awarded at $148,500 ex GST
  across six stages
- the rock and build-over risk lines

Undo award restores the line. Go tests, store tests, the costs and proposal
benches, the profile answer key and `npm run build` all pass.

## Edge cases

- A second brief with a different budget arrives: a new proposal, never
  over a typed figure.
- A budget stated as a range or "approx." keeps its words. The spread uses
  it only when a single amount is parsed.
- An engagement letter for a consultant not in the default list adds the
  row.
- A fee proposal from another project (address or client mismatch) does not
  award.
- Wrong org: 404 on every route, as today.

## Separate small defects (found in the same review)

- Client cue misses `**Client:**` (markdown bold) and plural `Owners:`.
- `sig.*-stated` rows go red when one passage reads true and others false.
  For a stated signal, false means "not in this passage" (`reconcile.go`).
- Footings read "not included" from "strip footings not recommended".
  Ground conditions read "not included" from the architect's exclusions.
  Staged read "yes" from the staged *fee*.
