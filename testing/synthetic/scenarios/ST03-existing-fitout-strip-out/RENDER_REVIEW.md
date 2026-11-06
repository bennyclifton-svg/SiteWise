# ST03-existing-fitout-strip-out: render review

Reviewer: independent read (brief 04r), 7 November 2026. Variant of ST03-base (corpus final and reviewed).

## Verdict: PASS

No finding changes what a reader should conclude or what the keys score. Four small wording fixes are
recommended below. None of them is required before a run.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST03-existing-fitout-strip-out`
gives `ok`, exit 0.

## 2. Copied documents

The 12 documents the manifest does not list in `changed_from_base` are byte-identical to `ST03-base/corpus`
(checked with `cmp`): d02, d04, d07, d08, d09, d11, d12, d14, d15, d16, d17 and d18. The only differences are
the six rewritten documents (d01, d03, d05, d06, d10, d13) and the new d19. `expected/` differs from the base
only where it should: `profile.json`, `invoices.json` and `rfis.json` are unchanged. `fees.json` has the two
rewritten phase labels. `document_register.json` adds d19. cost_plan, program, procurement and rfps add only
`triggers: ["existing-fitout-strip-out"]`; the scorer merges the trigger's `adds` from shape.yaml
(`score_shapes.expectations`).

## 3. A PM's read before opening expected/

The project is a commercial office (Class 5) refurbishment/fit-out of the whole of Level 14 (1,200 m2 NLA)
in a 24-storey A-grade tower built in 2008 and sprinklered throughout. The other floors stay occupied and
noisy works are after hours. The pathway is a CDC, the route is a traditional lump sum with AS 4000, and
quality is high. The fit-out budget is $2.4M ex GST, with the tenant supplying furniture, AV and IT.

**New in this variant.** The previous tenant's fit-out is still on the floor. The incoming tenant strips
partitions, plasterboard ceilings, joinery, carpet and old services back to base building. The base-building
grid, lights, VAV, sprinklers, detection/EWIS and the distribution board are retained and protected.
d01, d03, d13 and d19 say this consistently. d05 prices a demolition plan inside the documentation phase,
d06 makes the old services safe inside the investigation item, and d10 prices strip-out and make-good in
every estimate.

The consultants are the interior designer (lead), the multi-services engineer, BCA, access, the certifier
and the QS. The fees are $130,000 ex, $68,200 inc ($62,000 ex), $16,500 ex and the unchanged base proposals.
The invoices are unchanged, including the d17 mis-add.

This matches `profile.json` (the trigger has no profile keys), `fees.json` and `invoices.json`. The variant
leaves no base wording that contradicts the strip-out:

- "handed over in base-building condition" is gone from d01, d03 and d05.
- "No strip-out or demolition" is gone from d13.
- A grep of every copied document for handover, condition, demolition, strip, removal, vacant and make-good
  finds nothing.
- The remaining "make good the premises to base-building condition" (d01 line 102) is the planned
  `make-good-at-expiry` lease trap.
- "the whole floor is handed to the contractor at once" (d01 line 76) is the staging negation and fits.

## 4. Stray triggers

The changed documents carry only the planned trigger evidence:

- d01: "the landlord has agreed that the incoming tenant demolishes the existing offices".
- d13: "The previous tenant's fit-out will be stripped out back to base building: remove the existing
  partitions, ceilings, kitchen joinery and carpet".

A search for every other T03 trigger's evidence and the DOMAIN.md §10 traps finds only the base's planned
hits: DA 2004/0617, NABERS, the core bathrooms, the staging and Green Star negations. There is one unplanned
pattern hit (finding 2).

As notes.md says, the implied strip-out pattern fires on "demolition plan" (d05) and "strip-out" (d06, d10).
It does not fire on "stripped out", "strips it out" or "demolishes" (d01, d03, d13). Only d13's "remove..."
gives the action remove.

## 5. Title blocks

- d19 (clean) has four label lines (Drawing number ID-090, Revision A, Title, Date 29 April 2026) and one
  `#` heading that matches its Title. All four match `document_register.json`. Its issue status is recorded
  as "for landlord approval", the same as the base's d14 and d15.
- The rewritten d05 and d10 (clean) keep their base title blocks.
- d06 (messy, Markdown-table block) is unchanged.
- d01, d03 and d13 (adversarial) are unchanged in tier.

## 6. Realism

Every rewritten document reads like the real thing. The landlord agreeing that the incoming tenant strips
the old fit-out is a common lease-incentive arrangement. d19 reads as a real demolition sheet: dashed
removals, retained-and-protected base-building items, isolations by nominated contractors restored nightly,
and waste out through the goods lift after hours. No document reads as written for a test. The d04 email's
dates (access 7 September, staff in by early December) are now tight with an after-hours strip-out. That
is plausible for a PM, and the key does not depend on it.

## Findings

1. **d13 `Concept design report - Avenor Level 14 fit-out.md` line 14, "remove the existing partitions,
   ceilings, ...".**
   - **What is wrong:** A PM reading "remove the ... ceilings" and then line 36 ("the base-building ceiling
     grid altered ... tiles and lights stored for the landlord") briefly asks whether the grid goes. d03
     (lines 32 and 38) and d19 (lines 17 and 19) make clear that only the previous tenant's plasterboard
     ceilings go.
   - **Change:** "remove the existing partitions, plasterboard ceilings and bulkheads, kitchen joinery and
     carpet". The action remove and the trigger evidence are unaffected.
2. **d13 line 14, "kitchen joinery".**
   - **What is wrong:** This matches the `kitchen` implied pattern (`\bkitchens?\b`, scope_defaults.yaml).
     notes.md records it, but the manifest's d13 `traps` list does not. Every implied system is already in
     scope (joinery, sanitary, cold and hot water, local exhaust), so nothing scores differently. The one
     risk is a reader setting `det.commercial_kitchen` from the old tenant's kitchen.
   - **Change:** Add a d13 trap `old-kitchen-joinery` with expect "No det.commercial_kitchen; implied systems
     already in scope". Alternatively, write "kitchenette joinery" (no pattern hit).
3. **d05 `Fee proposal - Interior design services - Level 14 Wenlock Place.md` line 21.**
   - **What is wrong:** The rewritten sentence was not re-wrapped. The line runs to about 140 characters
     against about 100 everywhere else, which shows the edit.
   - **Change:** Re-wrap lines 19 to 23 at the document's width. No wording change.
4. **`expected/fees.json` d06 first price label.**
   - **What is wrong:** The label reads "Investigation of base-building services and capacity, and making
     safe the previous tenant's services". The document's item 1 (line 39) ends "... before the strip-out".
     Base labels are not always verbatim, so this is only a consistency point.
   - **Change:** Append " before the strip-out" to the label, so it matches the document as d05's label
     does.

Note on `expected/notes.md` (not a corpus finding): the last line, "As the base, less `no-strip-out`
(including `hdr.scale.tenancies` and `det.type_of_construction` keyed `not_read`)", reads as if the
parenthesis belongs to the dropped trap. Suggested text: "Traps and misses are the base's, less d13
`no-strip-out`. The five base keys stay `not_read` (`hdr.scale.floor_plate_sqm`,
`hdr.scale.grade`, `hdr.scale.tenancies`, `det.type_of_construction`, `proj.regulated_design`), and
`proj.target_construction_start` is still only in the d04 email. d19 is a drawing and is filed, not read for the
profile."
