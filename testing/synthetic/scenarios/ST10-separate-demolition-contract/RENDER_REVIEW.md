# ST10-separate-demolition-contract render review

Reviewer: independent read, 2026-11-06. Corpus and manifest not edited. Read as a variant of ST10-base (final
corpus, see `../ST10-base/RENDER_REVIEW.md`); only the documents that differ from the base, and base wording that
could contradict the variant, were re-read in depth.

## Verdict: FIX

Lint: `ok  ST10-separate-demolition-contract`.

Corpus diff against ST10-base: d01, d05, d13 (declared in `changed_from_base`) and d09, d11 (not declared).
Expected diff: `fees.json` d05 Stage 4 label only, plus the trigger lists and `notes.md`. No profile, fee amount
or invoice value differs from the base.

## What a PM concludes from the documents alone

As the base, except: the owners engage a licensed demolisher (who also removes the bonded asbestos under licence)
under their own contract, under the same CDC; the architect obtains three demolition quotations in Stage 4 with
the hazmat report attached; the site is handed to the builder clear and level; one builder, one HIA lump-sum
contract for the new house. The $1,520,000 ex GST construction budget is the owners' total for both contracts.
Not staged construction. This matches `expected/profile.json` (`flag.staged` no from d01, `proj.budget_stated`
152000000), `fees.json` and `invoices.json`.

Stray triggers: the shape negation "Demolition and asbestos removal are included in the builder's contract" is gone
from every document; "demolition-only complying development certificate" does not appear; "Stage 1 … Stage 2" in
d01 is the planned trap. d05 "Stage 1 … Stage 4" and d02 "Demolition Code" are base incidentals already recorded.

## Findings

1. **Manifest `changed_from_base` omits d09 and d11.** d09 line 31 changed "the tender documents and the builder's
   demolition work plan" to "the demolition quotation documents and the demolisher's demolition work plan"; d11
   lines 66-67 and 71 changed the tender/builders/builder's wording to demolition quotation documents/demolishers/
   demolisher's. Both changes are needed (the base wording put asbestos removal in the builder's tender, which
   contradicts the trigger), so keep them and add:
   - `{"doc": "d09", "change": "changed", "why": "Removal scope goes into the demolition quotation documents and the demolisher's demolition work plan, not the builder's tender"}`
   - `{"doc": "d11", "change": "changed", "why": "Recommendations 1 and 3 name the demolition quotation documents and the demolisher's work plan in place of the builders' tender"}`
   Also update the d09 and d11 `content` strings ("for the tender and the demolition work plan"; "register in the
   tender documents") to the new wording. No expected value changes: d09 carries only "Total $2,650 excluding
   GST" and its `fees.json` labels and amounts are untouched; d11's carries ("Survey of the existing building and
   outbuildings before demolition", "built in 1962") and its traps are unchanged.
2. **`expected/notes.md` is stale.** It says "d01, d05 and d13 changed" and that d09 and d11 "were written
   neutrally in the base and stay true". Change to "d01, d05, d09, d11 and d13 changed" and replace the d09/d11
   clause with: "d09 and d11 put the removal scope in the builder's tender and the builder's work plan; both now
   name the demolition quotation and the demolisher. d07 and d12 stay true."
3. **d04 `RE Timing for the build.md`, line 13-14: base wording now contradicts the variant.** "the electricity
   and gas disconnections need to be ordered well before the builder starts" — under a separate demolition
   contract the disconnections precede demolition, which precedes the builder. Change to "well before the
   demolisher starts, as the retailers and the network can take several weeks". Optionally add after the tender
   sentence: "The demolition quotations can run alongside, so the site is clear before the builder starts." No
   key effect (correspondence is not read; `proj.target_construction_start` stays 2027-02-01).
4. **d01 budget sentence (line 52): correct for the key, cosmetic fix only.** "Construction budget: $1,520,000
   excluding GST, including demolition. This figure covers both the demolition contract and the builder's
   contract." is consistent with `proj.budget_stated` = 152000000: one owner budget, demolition inside it. It must
   not be split, and no second figure should be added. The line is 127 characters, unwrapped where every other line
   wraps at about 100; rewrap after "We have set aside".
5. **d12 `Geotechnical report SLG-26-0461`, line 64-66: whose scope is ambiguous.** "The old brick strip footings
   along the southern side boundary are to be excavated and removed after demolition" plus fill removal sits
   between "handed to the builder clear and level" (d01, d13) and the builder's earthworks. A PM would ask which
   contract carries footing removal. Leave the trap sentence intact; optionally add after it "The demolisher's
   scope should include removing the old footings and slabs; fill removal and backfill are the builder's
   earthworks." No key effect.
6. **d13 writer departure (line 45-46), harmless.** "a temporary supply is set up for the builder once the site
   is handed over" is not in the manifest `content`; it is consistent with the variant. Record it in the d13
   `changed_from_base` why or leave as is.
7. **Realism, d01 lines 66-72.** "Demolition before the builder starts is not staged construction; the new house is
   built by one builder under one contract" reads as written for a test; an owner would not volunteer the
   negation. It is the planned adversarial negation (review ruling 2), so keep it; noted only.

Title blocks: d01 (now adversarial), d05, d09, d11, d13 title lines unchanged from the base and match
`document_register.json`.
