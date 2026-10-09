# Consultant RFP from the golden set: analysis plan

Status: plan only. Nothing built. Owner iterates on each discipline's RFP in
chat before any code.

## Goal

One RFP generator for every consultant discipline. A shared skeleton
(project, role, allowances, terms, evaluation, returnable schedules) plus,
per discipline, a stage list and a scope-position list that switch on
building class, work type, pathway, procurement, state and site flags.
Generated to about 80% right; the user tweaks the rest the way Profile
works today, and a user edit survives regeneration.

## Golden set

`D:\AI Projects\Test Data\Golden Consultant RFP`, 43 PDFs, about 640 pages.

| Tranche | Discipline | Proposals | Pages | Text layer |
|---|---|---|---|---|
| 1 | Architecture | AJC, nettletontribe, PACE | 11 | none (scanned; read as images) |
| 1 | Town planning | Mecone, Urbis, Willow Tree | 63 | yes |
| 1 | Structural | BGE, Northrop, Costin Roe (priced with civil) | 83 | yes |
| 1 | Civil | same three proposals, civil lines only | | |
| 1 | Traffic | BGE, McLaren, Varga | 26 | yes |
| 2 | Services (multi) | ADP, BSE, COVA | 91 | yes |
| 2 | BCA / certifier (PCA) | BM+G, Certitude, Group DLA, MBC | 95 | yes |
| 2 | Fire engineer | ADP (+ infrastructure concept), E-Lab, Minerva | 74 | yes |
| 2 | Access | Morris Goding, Vallabh | 55 | yes |
| 3 | Geotech | Douglas, EI, Geotechnique, JK | 61 | yes |
| 3 | Land surveyor | Crux, HFA, Land Surveyors | 15 | yes |
| 3 | Cost planner (QS) | MBM, RLB, Slattery | 42 | yes |
| 3 | Landscape | OHD (two copies, check if identical) | 32 | yes |
| 3 | Waste (WMP) | AusWide, BRP, Elephants Foot, LID | 16 | yes |
| 3 | Arborist | AchanArb, Naturally Trees | 5 | yes |

Tranche 2 is the design team (AIA stages apply). Tranche 3 is
report/investigation disciplines, closer to town planning and traffic.

## Method: one discipline at a time, everything written to disk

Context is not the store. Each step writes a file, so a later session
resumes from files, not memory.

For each discipline:

1. **Extract.** Text to `scratch/golden/<discipline>/<firm>.txt` with page
   markers; render pages that have no text layer, or that carry tick-box
   tables, as images and read them. Check the image whenever an
   included/excluded table appears, because ticks are lost in text.
2. **Ledger.** Write `docs/rfp-golden/<discipline>/ledger.md`: every scope
   line from every firm, one row each, with firm, page, stage the firm put
   it in, included / excluded / optional / hourly, and any count (meetings,
   options, sheets, hours, visits). Also: fee by stage, durations, rate
   roles and rates, reimbursables, assumptions, exclusions, conditions,
   insurance. Nothing summarised away at this step.
3. **Coverage check.** Re-read each proposal against its ledger rows, page
   by page, and tick the page off. A page with no ledger row needs a note
   saying why (marketing, CVs, project photos).
4. **Normalise.** Merge rows meaning the same thing into one canonical scope
   item, keeping links to every source row. Mark items that only one firm
   offered, and items firms disagreed on (in vs out): those become the
   section-5 scope positions.
5. **Drivers.** For each canonical item and stage, write what switches it:
   class, work type, pathway, procurement, state, scale, site flags, other
   consultants engaged. Say which project the evidence came from, so a
   rule taken from one warehouse is not applied to everything.
6. **Draft RFP.** Discipline stages, allowances, section-5 table and
   discipline-specific returnables on the shared skeleton. Owner reviews in
   chat; iterate to sign-off.
7. **Skeleton update.** If the discipline breaks the shared skeleton (for
   example a report discipline with no AIA stages), change the skeleton
   and record why, before the next discipline.

## Order

Tranche 1: architecture (redo steps 2-5 properly: v2 draft came from a
read-through, not a ledger), town planning, traffic, structural, civil.

Structural and civil are separate RFPs even though the golden proposals
price them together. A user who wants one firm for both sends both RFPs.
The app never combines disciplines into one RFP (owner decision 2026-10-09:
too much code for the gain).
Town planning and traffic go before structural because they test the
skeleton hardest (report deliverables, authority stages, not AIA).

Then tranche 2, then tranche 3. Each tranche ends with a skeleton review.

## Output after all tranches

- `docs/rfp-golden/skeleton.md`: shared sections and returnables.
- `docs/rfp-golden/<discipline>/rfp.md`: signed-off draft per discipline.
- A drivers table across all disciplines: the profile fields the generator
  needs, and which of them SiteWise does not capture yet.

That drivers table is the input to the build plan (data model, editable
generated rows with source = profile / default / program / user, and
regeneration that keeps user edits). The build plan is written after
sign-off, not before.

## Stage naming

Stage names follow AIA: Concept, Schematic Design, Design Development,
Construction Documents (Tender / For Construction), Bidding, Construction
Administration, plus non-AIA stages where the work needs them (planning
approval, regulated design, post-novation). The RFP shows names, not codes.
Stable internal keys exist only in the data.

## Durations

From the SiteWise program when it has the stage; otherwise defaults
(Concept 1 month, Schematic 2, DD 3; others by pathway and scale), shown
as "estimate".
