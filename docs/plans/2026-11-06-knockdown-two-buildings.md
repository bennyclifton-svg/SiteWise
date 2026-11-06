# Knock-down jobs describe two buildings

Status: decided 6 November 2026 (owner approved the direction: "action items
1 to 4"). Lane A. Draft wording stays `status: draft` for owner review.

Source: synthetic run `testing/synthetic/runs/20261106-t10-t12`, findings
F17, F18, F19 and F30 in `testing/synthetic/FINDINGS.md`.

## The problem

On a knock-down rebuild, a dual occupancy on a cleared lot, or a granny
flat beside an old house, the documents describe two buildings. One is the
building being demolished or surveyed, and the other is the one being
built. The hazardous materials survey, the old-house survey, the geotechnical
report and the demolition passages all talk about the old one. The profile
has one set of answers per part, so the old building's passages vote on
the new building's answers:

- **F17.** "This is a new house: we will knock down the existing house" is
  read as a denial of `det.existing_building`.
- **F18.** "Lead paint on the window frames" and "the old strip footings are
  to be excavated" set the window and footing actions to `investigate`.
  A consultant's inspection service does the same ("inspections of the
  footings and the slab reinforcement before the pour"). Most house
  construction fragments then do not fire.
- **F19.** Hazmat and survey passages about "the existing house" give
  `hdr.subclass` ten "house" votes on a townhouse job. The subclass becomes
  a conflict. The blank subclass then falls back to the residential
  category list, which is the Class 2 apartment list, so a two-dwelling
  job gets lift, hydrant and access consultants.
- **F30.** Asbestos and demolition passages are read as a "Dangerous goods"
  hazardous process. The taxonomy limits that condition to industrial
  buildings, but only the profile view hides it; the reading and the RFP
  brief keep it.

## What was considered

1. **A separate part for the old building.** It is the faithful model, but
   it is a new workflow. Parts are created by the user, the reading has no
   way to route a passage to a part it has not been told about, and the
   scope, program and RFP code would all need to learn it. Too wide for a
   defect fix.
2. **A knock-down work type.** The shapes name this gap (T10 §11). It
   changes the taxonomy, the defaults and every work-type rule. It is also
   wider than these defects and is an owner decision.
3. **Narrow fixes where each leak enters.** Chosen. Each rule is small,
   testable on its own, and leaves the user's choices final.

## Decisions

**D1. Demolishing an existing building is work to an existing building
(F17).** The T10 and T11 keys already say so and two reviewers agreed. The
`existing_building` criteria gain one sentence: demolition of an existing
building on the site affirms the condition, and calling the project "a new
house" does not deny it. Known side effect, already named in the T10 shape
(§11): the structural "existing assessment" stage turns on, though nothing
is kept. That stays a known gap until there is a knock-down work type.

**D2. On new work, `investigate` and `remove` readings do not replace the
default action (F18).** In a part whose work type is `new`, the new
building's systems are new whatever the documents say about the old
building's systems or about inspections of the new work. Removing the old
building is the `site.demolition` system's job, and inspecting new work
belongs to that work (`actions.yaml`, `investigate.excludes`). Code rule in
`internal/profile/works.go`. Every other action (retain, alter, repair,
upgrade, replace) still applies, because "the existing sewer connection is
retained" is a fact about the new job. Extension, refurbishment and
remediation work types are unchanged. The user's action is final, as now.

**D3. The building type is the building being delivered (F19, reading).**
The header questions (building class, subclass, work type) gain one
sentence: an existing building that the works demolish, or that a survey,
hazardous materials report or inspection describes, is not the project's
building. This needs a question-version bump (`profile-12`).

**D4. An unknown residential subclass suggests only what every residential
class shares (F19, fallback).** When the category is known but the
subclass is not, the scope defaults are the systems common to every
new-build list in that category, not the category list. For residential,
that is the house, townhouse and apartment lists' common part, so no lift,
hydrant or accessible-access suggestion until the subclass is known.
Defaults are suggestions, never evidence, and the category lists are draft.
A category with no class lists keeps its own list.

**D5. A header condition limited to other categories is not kept (F30).**
When the building category is known and a condition's `applies_to` does
not include it, the profile drops the read row, so the RFP brief and every
other consumer agree with the profile view. A user-set value is kept.
Unknown category keeps the row.

## Not in this change

- F27 (the majority rule counts passages, not documents). It also feeds
  F19, but it changes the reconciliation for every field. It needs its own
  change and a re-run of all scored types.
- A knock-down work type and an old-building part (options 1 and 2).
- F24, F26 and F29. F24 needs a fee-reading change, F26 a flag wording
  change with its own evidence, and F29 is program library content.

## Speed

No new Jev call and no new round trip. D3 and D1 change question wording
inside the existing fan-out, so finished documents are read again on the
next "Update project profile" (normal for a version bump). D2, D4 and D5
are Go in the profile build, which is precomputed.

## Done when

- Unit tests for D2, D4 and D5, and the existing suites, pass.
- `python tools/check_knowledge.py` passes.
- T10 to T12 re-run one scenario at a time. The expected result is
  `det.existing_building` true on T10, footing, slab, frame and window
  actions `new`, the T11 subclass `townhouses` with no lift or hydrant
  consultants, and no "Dangerous goods" in any brief.
