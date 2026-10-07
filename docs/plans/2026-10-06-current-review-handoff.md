# Current review handoff

> **Current progress:** [NEXT-WAVE-STATUS.md](NEXT-WAVE-STATUS.md). On 7 October the owner authorized completing M2/M3 unattended; earlier sequencing/permission holds are historical. Acceptance evidence remains explicit.

> 7 October owner update: publish the current implementation checkpoint to main/GitHub; move broader acceptance testing beyond the thin 0991 project. See implementation plan "Current owner direction". Outstanding gates remain open; this is not acceptance or deployment approval.

**Historical snapshot — 6 October 2026.** The body below records the implementation,
decisions and evidence at that date, when the work was local and unmerged; it is
not the current publication or completion status. Use [NEXT-WAVE-STATUS.md](NEXT-WAVE-STATUS.md)
for current progress and the [7 October publication checkpoint](../evidence/2026-10-07-publication-checkpoint.md)
for publication evidence.

The two direction questions below were resolved on 7 October: partition work
uses [explicit per-work layout context](../evidence/2026-10-07-k1-layout-input-proposal.md),
and actionless physical proposals require [an explicit action choice on acceptance](../evidence/2026-10-06-proposal-action-gap.md#follow-up--7-october-2026).
Their historical pending/hold wording below no longer applies. Owner review and
acceptance gates remain as recorded on the current progress page.

## Decisions already resolved

The owner's undo choice is implemented: retire an untouched, unreferenced
created work item; refuse after editing or use. Historical references also
block undo. Store/API and endpoint evidence is in
`docs/evidence/2026-10-06-proposal-undo-history.md`.

## Pending direction questions

1. Partition work: `interiors.walls-linings` combines room partitions and
   lining-only repairs. The pending question offers splitting them, accepting
   broader coordination for all new/alter work, or deferring the K1 edge.
   `docs/evidence/2026-10-06-k1-partition-air-distribution-design.md` records
   why the proposed edge has not been added.
2. Physical make-good proposals: require an explicit action choice on
   acceptance, or create a repair item on the penetrated construction.
   `ic.penetrates-make-good` currently lacks an action and acceptance refuses
   it. `docs/evidence/2026-10-06-proposal-action-gap.md` records the scope.

These questions are already pending in the chat; silence is not a selection.

## Missing quality evidence

`data/eval/profile/answer-keys/0991.yaml` is a draft key for two pinned
revision-A drawings, not a complete project-scope or obligation key. It
contains three work expectations and explicit exclusions. Owner review must
confirm or correct those expectations and the scope of the scored corpus;
accepting this limited key alone cannot establish whole-project completeness.

The 0991 critical-obligation checklist and top-list usefulness assessment
remain owner work under plan §8.6. The relevant draft knowledge/clauses need
review, plus the manual comparison and active-time measurement. A synthetic
fixture and automated tests cannot supply those judgments. 0777 project/corpus
identification and its owner-reviewed key remain outstanding for M3. K5 also
requires the owner's actual project lessons.

## Technical gates still open

- Automatic proposal generation on edits remains OFF. Ordinary edits commonly
  hover around or above the 50 ms median limit and intermittently reach about
  680 ms. Integrated rebuilds commonly take roughly 85–95 ms but also show the
  large intermittent delay. A passing repetition is not resolution.
- The last SQL rank-only experiment was reverted. Projection/integrated checks
  passed after restoration. Evidence:
  `docs/evidence/2026-10-06-proposal-sql-rank-experiment.md`.
- Changed action/location questions still need fresh approved live evaluation.
  Automatic approval review rejected the earlier transmission of private
  source/Hale passages to `api.typesafe.ai`; the existing consent question
  remains unresolved. Local replay does not replace those recordings.
- K1–K4 contain draft work and source assessments, not owner-reviewed content.
  The mechanical commissioning proposal now has a real acceptance/undo test,
  but it does not complete K2 or establish usefulness.
- M2 cost/issue/export and remaining delivery functionality must follow M1
  acceptance. M3 additionally needs 0777 evidence. Target-host performance,
  restore and production approval remain separate release gates.

Keep the full objective open. Do not merge, deploy, mark knowledge reviewed,
promote a timing fixture into a scored key, or relax a budget to bypass these
gaps. Independent work can continue when it has a concrete source, requirement
and validation path; repeated inconclusive benchmarks are not progress.
