# Render review: ST12-sewer-main-in-backyard

Reviewer: independent read (brief 04r), 2026-11-06. Variant of ST12-base (see its RENDER_REVIEW.md);
this read concentrates on the seven changed documents (d03, d05, d06, d11, d12, d13, d14) and on
base wording that could still contradict a sewer main in the backyard. Corpus and manifest not edited.

## Verdict: FIX (minor)

Two small text edits and two notes. No unplanned trigger fires; no key value is contradicted.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST12-sewer-main-in-backyard`
gives `ok`. 17 documents; exactly the seven listed in `changed_from_base` differ from ST12-base.

## 2. What a PM concludes from the documents alone

As ST12-base (Class 1a detached granny flat, 60 m2, CDC, Class M, no groundwater to 2.5 m, $210,000
ex GST, same consultants and invoices), with one change: a 150 mm Sydney Water sewer main runs about
1.0 m inside the rear (southern) boundary, about 2.0 m deep, with a maintenance hole near the
south-western corner. With a 3.0 m rear setback the flat is about 2 m from the main and within its
zone of influence, so the rear edge goes on bored piers into weathered sandstone, the plans go to
Sydney Water (Tap in building plan approval and asset assessment) before the CDC, and the main is
CCTV-inspected before and after. The house drain now discharges to that main, not to the street.
Structural fee rises to $2,250 ex GST (items $1,250 + $650 + $350; $280/visit). Designer's fee
unchanged at $9,000 ex.

## 3. Comparison with expected/

- `profile.json`: only `flag.authority_asset` moves no -> yes, citing d03, d06, d12. Agrees. d05, d11,
  d13 and d14 also say a main is at the rear; all agree with yes.
- `fees.json`: d06 `stated_total_cents` 225000 and the new 65000 stage "Authority asset assessment and
  pier design to Sydney Water's guidelines" match the document; 1,250 + 650 + 350 = 2,250. d05
  unchanged at 900000. Agrees.
- `invoices.json`: unchanged and unaffected.
- Base wording: no remaining "in the street", "sewer junction", "no Sydney Water assets" or
  front-boundary sewer anywhere. d10's Tap in sentence holds for both. The base review's fixes (d10
  "stages 2 and 3", d11 "0.5 m beyond its edge", d12 "garden planting") are present here.

## 4. Sensitive words checked (the variant's new wording)

| Word | Where | Result |
|---|---|---|
| zone of influence | d06 l.30, d11 l.66-68, d12 l.71-72, d13/d14 l.20-21 | Matches only the planned `authority_asset` evidence. No basement, excavation or groundwater regex contains it |
| piers / bored piers | d06, d11 l.67-70, d12 l.72, d13/d14 | Not the sloping-backyard evidence ("suspended floor on piers with a deck"); d03 carries that trigger's negation "generally level". No flag regex reads "pier" |
| 2.0 m deep (d03 l.60), 1.8-2.2 m depth (d11 l.68) | | `basement` needs `deep excavation` or `excavat\w* to a depth`; neither phrase is formed. `excavation_adjoining` needs "excavat" in the same sentence as boundary/neighbour; d11 l.66-71 has no "excavat" |
| excavat* | d11 l.75-76, l.81 | Unchanged declared traps (excavation-near-boundary, excavate-to-depth) |
| groundwater | d11 l.39 | Unchanged declared carry (no) |
| CCTV | d12 l.74 (new), d07 (base) | No flag regex or system term reads CCTV; `comms-security.cctv` is reached only through cluster questions about cameras, which this passage cannot satisfy |
| maintenance hole (d03 l.60-61, d12 l.75) | | Invented but consistent: rear = south (d12 bedrooms at the southern end, roof falls south), house drain on the western side, so a SW maintenance hole fits. No trigger |
| 600 kPa end bearing (d11 l.69) | | Invented; plausible for extremely to highly weathered sandstone. No key field or rule reads it |
| sewer main / Sydney Water asset | d03, d05, d06, d11, d12, d13, d14 | All planned or consistent with authority_asset = yes |

No unplanned shape.yaml trigger evidence appears. No DOMAIN.md §10 trap is newly hit; trap 5
("connect to the sewer main") now correctly reads yes.

## 5. Title blocks

All seven changed documents keep the base title blocks: label lines before the first `##`, one `#`
heading equal to `Title:`, number, revision and date matching `document_register.json` (unchanged
from base). d03, d11, d12 stay adversarial with only their declared traps.

## 6. Realism

Reads as a real file. d06 l.22-23 ("...building next to a Sydney Water asset: the sewer main along the
rear boundary") repeats the main twice to hold a verbatim carry; acceptable, no change.

## Findings

1. **d12 `Design statement - 18 Tuckeroo Crescent.md`, line 74**: "The main is CCTV-inspected before
   and after construction" is present tense in a pre-CDC statement, as if done. Change to "The main
   will be CCTV-inspected before and after construction". Not a carry; no key effect.
2. **d11 `Geotechnical report TG-26-0307 - 18 Tuckeroo Crescent.md`, line 68**: piers "at about 1.8 m
   to 2.2 m depth" start shallower than the main itself ("about 2.0 m deep", d03 l.60). Geometrically
   defensible (the flat is ~2 m off the main, so the zone-of-influence line is shallow there), but a
   PM or Sydney Water reviewer would query piers founded above the invert. Change to "at about 2.2 m
   to 2.5 m depth, below the invert of the main" and update d11 `content` in the manifest to match.
   No trigger effect either way.
3. **Scorer note, no corpus change** (`expected/notes.md` "The trigger adds `substructure.piles`"):
   `knowledge/works/system_terms.yaml` maps the terms "Pier" and "Pier footings" to
   `substructure.footings`, not `substructure.piles`. The documents say "piers" / "bored piers", so a
   footings item rather than a piles item is the likely app output. Record this in notes.md as a
   knowledge gap, so a missing piles item is not blamed on the corpus.
4. **Manifest note, no corpus change**: the d03 `content` already names the maintenance hole, but the
   d12 `content` and the d11 600 kPa end bearing are writer additions. Both are harmless and
   consistent; add them to the manifest `content` when it is next revised so a re-render keeps them.
