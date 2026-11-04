# ST01-trees-affected notes

The base Nguyen house where the driveway cut encroaches into the TPZ of the neighbour's jacaranda and the rear slab
edge is within the SRZ of the on-site blue gum. Only `flag.arborist_tree_impacts` and `flag.arborist_offsite_trees`
differ from the base (no -> yes).

## Changes from the base

- d11 design statement carries the two evidence phrases in place of the negation.
- d03 survey places the neighbour's jacaranda about 2 m from the common northern boundary; it still only lists
  the trees, and presence alone stays not an impact.
- d13 site plan Rev B adds the jacaranda, the TPZs, protection fencing and the pier-and-beam rear terrace edge.
- d14 SEE adds a Trees paragraph (two trees affected and retained, arborist's assessment and protection plan)
  and lists the assessment among the documents read with it.
- d18 arborist fee proposal (wave 2, repository; $3,400 ex GST) and d19 arboricultural impact assessment (wave 3)
  added; 19 documents.

## What the scorer should see

Shape trigger `trees-affected`: arborist package (from d18), site.trees retain, Arborist and Tree protection cost
lines, arborist impact assessment and tree protection program rows, arborist RFP with AS 4970 assessment, tree
protection plan and supervision hold points. The app does not suggest an arborist on its own (shape built_in
procurement no), so the package depends on the fee proposal being cued.

## Missed by design

As the base: `proj.target_construction_start` only in the d04 email; d02 register number (PC-2026-01187) and
date (2026-02-10) and d07's number and date are in Markdown tables, so the app is expected to miss them.

## Open points

- Root cutting is worded without 'pruning' so arborist_pruning stays unset; whether the trees-affected trigger should also set it is an open owner question.
- d11, d14 and d19 carry the rear terrace edge on bored piers. The scorer ignores a piles work item or piers cost line for the terrace edge: neither is keyed, and the structural fee (d06) predates the change.
