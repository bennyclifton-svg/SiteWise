# Render review: ST12-base

Reviewer: independent read (brief 04r), 2026-11-06. Corpus and manifest not edited.

## Verdict: FIX (minor)

Three small text edits. None of the hits below is expected to change a profile value: every
sensitive word the brief named is either a planned trap or carry, or (for the three undeclared
hits) its Jev question rules it out. They are still unplanned hits, so they are findings.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST12-base` gives
`ok`. 17 documents.

## 2. What a PM concludes from the documents alone

- Residential, new detached single-storey secondary dwelling (granny flat), Class 1a, 60 m2,
  2 bed, 1 bath, no garage, behind an occupied house on a 600 m2 R2 lot in Gymea (Sutherland
  Shire, Lot 32 DP 237114). Main house retained and not altered.
- CDC from a registered certifier under the Housing Code; designer lodges. No DA, no cl 4.6, no
  panel, no exhibition (neighbour notices by the certifier).
- Site: Class M, no groundwater to 2.5 m, generally level, no easements, sewer and water mains in
  the street, no gas, no OSD existing or required, not heritage, bushfire, flood, ASS,
  contaminated, coastal or rail. Constrained access by a 3 m side driveway past the occupied
  house. Trees present but none affected. Climate zone 5.
- Consultants: building designer, structural, stormwater, geotechnical, BASIX, certifier,
  surveyor (already engaged; invoice only). Builder by lump-sum quotes from three, HIA NSW new
  dwellings contract.
- Budget $210,000 ex GST (certifier's $231,000 inc GST agrees). Target start 1 September 2026.
- Fees: designer $9,000 ex (+$600 option, $140/h); structural $1,600 ex (+$280/visit);
  stormwater $1,870 inc ($1,700 ex) with CCTV at cost; geotech $1,700 ex (+$380/visit); BASIX
  $800 ex; certifier $4,700 ex (+$260/visit, registration fee at cost).
- Invoices: SBD-INV-0331 $2,800 + $280 = $3,080; BHS-1043 $2,450 + $245 = $2,695; TG 2207
  $1,700 + $170 stated as $1,780 (should be $1,870; arithmetic error).

## 3. Comparison with expected/

`profile.json`, `fees.json` and `invoices.json` agree with every conclusion above. No value in the
key lacks a supporting document; no document states a value the key contradicts.

## 4. Sensitive words checked

| Word / trigger | Where | Status |
|---|---|---|
| existing building | d03 l.37, d12 l.17 | Declared traps (existing-building-retained, existing-building-main-house) |
| sewer main / Sydney Water asset | d03 l.59-60 | Declared trap sewer-main-in-street |
| water main | d03 l.64 | Declared carry (det.water_supply_source); not named in the trap, see finding 4 |
| trees | d03 table, d12 Trees | Declared trap trees-listed-not-affected; d12 carries |
| excavat* | d11 l.68-69, l.74 | Declared traps excavation-near-boundary, excavate-to-depth; carry |
| demolition | d12 l.45 | Declared trap no-demolition |
| separating wall | d12 l.45 | Declared trap no-separating-wall |
| dual occupancy / subdivided | d12 l.46 | Declared trap not-dual-occupancy |
| garage / car space | d12 l.21; d01 l.40 | d12 declared trap garage-zero; d01 planned in content, value 0 agrees with key |
| Stage N (staged trigger) | d05, d15 | Planned fee stages (manifest content); d15 is an invoice drop |
| Stage N (staged trigger) | d10 l.47 "stages 2 and 3" | **Not planned**, finding 1 |
| beds (bedrooms trigger) | d12 l.37 "planting beds" | **Not planned**, finding 3 |
| lodge* | d05, d09 l.26, d14 l.43 | Key is architect_lodges yes; d09 and d14 can only agree |
| on-site detention | d03 l.70, d07 | Declared trap osd-negated and carries |
| occupied | d03 l.24 | Agrees with key partial_occupation |
| 900 mm | d11 l.68 (declared), d12 l.35 (close-to-house negation, planned) | OK |
| Unit 7 (tenancies trigger) | d08, d16 address | Commercial scale field, not a house field; no effect |

No shape.yaml trigger evidence phrase appears positively anywhere.

## 5. Title blocks

All clean documents have recognised `Label: value` lines before the first `##`, one `#` heading
equal to `Title:`, and number, revision and date matching `document_register.json`. d02, d07 and
d16 (messy) carry identity only in tables or fixed-width blocks. d03, d11 and d12 (adversarial)
have clean blocks plus the declared traps. d04 has full email headers on both messages.

## 6. Realism

Reads as a real granny-flat file. Two passages read as written for a check, both forced by
verbatim carries, so no corpus change without a manifest change:

- d03 l.24-27: the two access sentences repeat "through a double gate".
- d12 Trees and Planning sections: three tree negations and the cl 4.6 / panel sentences read as
  a checklist; a real design statement would say "No trees are affected" once.

## Findings

1. **d10 `Certifier fee proposal LBC-26-0318.md`, line 47**: "Our fees for stages 2 and 3 are
   payable on appointment as principal certifier." matches the `staged` trigger
   `\bstages? [1-9a-c]\b` in a repository (read) document, and no trap plans it. Low risk (the
   question excludes fee stages) but unplanned. Change to: "Our fees for the inspections and the
   occupation certificate are payable on appointment as principal certifier."
2. **d11 `Geotechnical report TG-26-0307 - 18 Tuckeroo Crescent.md`, lines 74-75**: "and for 1 m
   beyond its edge" contradicts the 0.9 m western setback (d12, d13, d14): the strip would cross
   the boundary. Change "for 1 m beyond its edge" to "for 0.5 m beyond its edge". The trap wording
   "Excavate to a depth of 200 mm to strip topsoil under the slab platform" stays.
3. **d12 `Design statement - 18 Tuckeroo Crescent.md`, line 37**: "screened from the main house by
   a fence and planting beds" matches the bedrooms trigger (`beds`). Unplanned. Change "planting
   beds" to "garden planting".
4. **Manifest note, no corpus change** (d03 line 64): "Water main in Tuckeroo Crescent" also
   matches `authority_asset` `(sewer|water) main`. It is a declared carry and the right answer is
   still no, but trap `sewer-main-in-street` names only the sewer sentence. When the manifest is
   next revised, add the water-main sentence to that trap's `what`.
