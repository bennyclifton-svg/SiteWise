# ST10-da-pathway render review

Reviewer: independent read, 2026-11-06. Corpus and manifest not edited. Variant of ST10-base (see its
RENDER_REVIEW.md); this read focuses on the eight rewritten documents (d01, d04, d05, d06, d07, d10, d13, d15),
the three added ones (d19, d20, d21) and any base wording that still implies a CDC.

## Verdict: FIX

Lint: `ok  ST10-da-pathway` (21 documents).

## What a PM concludes from the documents alone

As ST10-base (knock-down rebuild, 22 Marrakai Close, Caringbah South, Lot 31 DP 216847, Sutherland Shire, R2,
619.8 m2; Class 1a + 10a, 302 m2, 4 bed, 3 bath, two storeys; $1,520,000 ex GST; HIA lump sum, select tender to
three; all-electric; Class M, kerb discharge, trees retained), except the pathway: one DA to Sutherland Shire
Council for demolition and the new dwelling because the upper floor is 1.2 m from the northern boundary (Housing
Code and the 1.5 m DCP control not met), then a CC from a certifier. Architect lodges the DA; town planner writes
the SEE. Determined under delegation, no cl 4.6, no panel, standard notification only. Consultants add town
planner $5,400 and landscape architect $3,500 ex GST; the other fees, and the three invoices, are unchanged.

This matches `expected/profile.json` (only `hdr.cond.planning` = da differs from base), `fees.json` (d05, d06, d07,
d10 labels; d19 and d20 totals and the $210 rate) and `invoices.json`. Title blocks of d19, d20 and d21 match
`document_register.json`. The SEE setbacks, height, landscaped area and FSR (0.49:1 ≈ 302 / 619.8) agree with
A-101 Rev B and d13; the shadow reasoning (north wall, no shadow to the north) is right.

CDC sweep: no remaining "CDC", "complying development certificate" or certifier-determines wording. d02 item 4
("Complying development under the Housing Code may be carried out on the land") is the declared
`cdc-land-eligible` trap and is ordinary s10.7 text. The base review fixes (d03 gas, d06 "expected", d12 southern
boundary, d13 trees paragraph, d13 contamination trap) are all carried into this variant.

No stray `shape.yaml` trigger evidence or §10 keyword trap hits in d19, d20 or d21: "variation" in d21 is a DCP
variation and does not match the `cl46` triggers; "clearance certificate" in d21 is not near "tree".

## Findings

1. **d21 `Statement of environmental effects - 22 Marrakai Close.md`, line 102: banned word.** "no local planning
   panel referral is expected" uses "expected", which the render brief bans outright (the key folder is named
   `expected/`). Change the corpus and the manifest d21 `flag.panel` carries to:
   "Determination under delegation; the application does not need referral to the local planning panel."
   The `panel` trigger (`planning panel`) still fires and the answer stays no. This is also how a planner writes it.

2. **d05 `Fee proposal - Architectural services - 22 Marrakai Close.md`, lines 36 to 57: the fee split no longer
   fits the scope.** Stage 4 now carries the CC drawings coordinated with the engineers and the conditions, plus the
   whole tender, for $7,000, while DA documentation is $42,000. The amounts were sized for a CDC job where the full
   documentation set was Stage 3. An experienced PM would query this; an architect would not price it that way.
   Keep the amounts and move the CC drawings into Stage 3:
   - Stage 3 heading and table row: "DA and CC documentation, demolition plan and lodgement" ($42,000); append to its
     description "After consent, the construction certificate drawings, coordinated with the structural and civil
     engineers' documents and the conditions of consent, for your certifier."
   - Stage 4 heading and table row: "Tender" ($7,000); description as base ("Tender documents and a schedule of
     finishes, issue to three builders you have approved, responses to tender queries, a tender comparison and a
     recommendation.").
   Update the manifest d05 content and the two `fees.json` labels to match. (Alternative if the labels must stay:
   re-split the amounts, for example Stage 3 $26,000 and Stage 4 $23,000, total unchanged, with `fees.json` amounts
   updated.)

3. **d01 `Owner brief - 22 Marrakai Close.md`, lines 57 to 60: the owners know a design result before there is a
   design.** The brief is dated 2 February, asks the architect for a fee proposal (line 87), and predates the site
   walk (26 February, d05). "Our architect tells us the second storey exceeds the Housing Code side setback" cannot be
   known yet. The d01 carries line is only the DA sentence, and the trigger evidence sentence is still in d13 and d21,
   so change the corpus and the manifest d01 content to:
   "Ines told us at our first meeting that a second storey of the size we want is unlikely to meet the Housing Code
   side setback on a lot this width, so complying development is probably not available. We would like our
   architect to prepare and lodge the application with Sutherland Shire Council and see it through to approval."
   (Keep the preceding DA carries sentence unchanged.)

4. **d20 `Landscape plan fee proposal - Fernleaf.md`, lines 20 to 21: a drawing that does not exist yet.** The quote
   is dated 19 March and thanks the architect for "the site plan"; A-101 Rev A is first issued on 4 May. Change to
   "Thank you for the survey and the concept sketches." The scope bullet "coordinated with the architect's site plan"
   is future work and can stay.

5. **d21 lines 18 and 79, d10 line 41: unwrapped lines.** Each runs well past the ~100-column wrap of the rest of
   the corpus, which shows where a carries sentence was pasted in. Rewrap to match the surrounding paragraphs; wording
   unchanged.

## Realism

d19 and d21 read like real Sutherland Shire planning documents (compliance table, s4.15 headings, DCP objective
justification). d20 is a believable small-practice quote apart from finding 4. Across the rewrites only findings 2
and 3 read as wording swapped in rather than written for a DA job. None read as written for a test.
