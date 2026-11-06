# Render review: ST02-base

Reviewer: independent (brief 04r), 2026-11-06.

## Verdict: **FIX**

Four findings, all small. None of them changes a scored value. Fees, invoices, the register and every
profile key that a read document carries agree with `expected/`. The fixes are one internal contradiction
about the site layout, one sentence that contradicts itself, one date-order wart and one test-flavoured
paragraph.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST02-base` →
   `ok testing/synthetic/scenarios/ST02-base`.

2. **Cold read (before opening `expected/`).** As a PM I would conclude the following:
   - **The job:** a new six-storey residential flat building, 64 apartments (14/38/12), NCC Class 2 over a
     Class 7a basement. Rise in storeys 6, effective height about 18.6 m, Type A, sprinklered.
   - **Site:** three amalgamated R4 lots (Lots 11-13 DP 236418, 2,403.6 m² by survey), with three 1962
     dwellings to be demolished. Vacant possession, single owner.
   - **Pathway:** a DA to the City of Ryde under Housing SEPP Ch 4 and the ADG, determined by the Ryde LPP.
     The planner lodges, there is no cl 4.6, no TfNSW referral and only statutory exhibition.
   - **Procurement:** AS 4902 D&C, with the architect, structural and civil engineers novated after DA.
     The DBP Act applies.
   - **Site conditions:** no heritage, bushfire, flood, contamination, ASS, coastal, rail or classified
     road. No groundwater; minor seepage only. Fill and clay over Ashfield shale over sandstone at
     4.7-5.4 m. The 7 m basement comes within 1 m of the northern boundary, so excavation adjoins a
     neighbour. Access is urban-constrained, OSD and MUSIC are needed, there are s138 road works and no
     NorBE. The site is not noise affected, has no transfer structure, a standard masonry facade and is
     all-electric. It is climate zone 5 and on BASIX only. One stage of construction and one building.
     No trees are affected.
   - **Budget:** $27.6M ex GST. The construction start target of 1 Nov 2027 appears only in the email.
   - **Fees (ex GST):** architect $780k plus a $520k option, structural $156k plus a $210k option, civil
     $82k ($90.2k inc) plus an $18k option, geotech $48k, planner $72k. Rates are $260, $420, $480 and
     $240.
   - **Invoices:** AVA-INV-0214 $185,000 + $18,500 = $203,500. BRG 4471 $48,000 + $4,800 = $52,800.
     HP-0093 $12,000 + **$1,800** (15%) = $13,800, so its GST is wrong.

   **Compared with `profile.json`, `fees.json` and `invoices.json`:** every value matches, including the
   accept lists for site area, GFA and lot, the `gst_inc_normalised` civil figures and the HP-0093
   `gst_rate` flag. No difference needs a corpus change.

3. **Stray triggers.** I searched the corpus for every `shape.yaml` trigger's evidence and negation, and
   for the DOMAIN.md §10 traps. Patterns covered: existing building, N buildings, stage N, water/sewer
   main, traffic noise, street trees, lodge, public exhibition, strata, party/common wall, flood,
   heritage, gas, rainscreen, transfer, underpin, dewater, 4.6, SSD, Class 2/7a. Every hit is a planned
   carry or trap, or is listed in notes.md as incidental. "Substation" in d04 is correspondence and is
   not read. I found no undeclared hit.

4. **Title blocks.** All clean documents (d01, d04, d06, d08, d14, d15, d16) have `Label: value` lines
   before the first `##` and one `#` heading that equals `Title:`. Their number, revision and date agree
   with `document_register.json`. d04 is a real two-message email thread. The messy documents (d02, d07,
   d12, d17) carry table title blocks as their tier says. The adversarial documents carry their declared
   traps.

5. **ABNs (extra check).**
   - Corrigan Lane (d03) has 18 568 916 604, Firecode (d12) has 76 827 666 579 and Resonance (d13) has
     23 181 239 043. All three pass the ABN checksum. Each appears once, on that firm's own document.
   - None of these three firms has a fee proposal or invoice, so none appears in `fees.json` or
     `invoices.json`. There is nothing there to match and nothing that conflicts.
   - The five firms that are keyed match the expected ABNs on every document: d05/d16 Arden Vale, d06
     Kestrel, d07 Northfall, d08/d10/d17 Basalt Ridge, d09/d18 Hollis.
   - All eight ABNs are distinct, so no two firms share one.

6. **Street on the east (extra check).** d01 line 34 puts the lots "on the western side of Calloway
   Street", so the frontage faces east. Everything else agrees:
   - d03: the land falls from the south-west corner to the frontage, and the neighbours sit on the
     northern, southern and western boundaries ("dwellings on three sides").
   - d03 floor levels: the southern neighbour is at 42.40, the western at 42.95 and the northern at
     41.05. These are consistent with a fall to the north-east.
   - d10: dilapidation surveys cover the north, south and west.
   - d11, d14 and d15: deep soil is at the rear (west).
   - d15: the lobby moves 1.2 m east, toward the street.

   Nothing contradicts the orientation.

7. **Realism.** The set reads as a real West Ryde amalgamation, and the commercial documents are
   convincing. Finding 4 lists a few sentences that read as written for the reader.

## Findings

1. **Deep soil sits over the basement.**
   - **Where:** d11 `Architectural design report - Calloway Street apartments.md` line 61; d14 and d15
     `Ground floor plan A-DA-1101 Rev A/B.md` line 21; d10
     `Geotechnical report BRG-R-1152-01 - 18-22 Calloway Street.md` lines 48-49.
   - **What is wrong:** d11, d14 and d15 put deep soil "along the rear boundary and the northern side".
     d10 says the basement comes "within 1 m of the northern boundary" and is set back only 2.5-4 m
     elsewhere. Deep soil cannot sit over a basement, and the ADG minimum dimension is 6 m, so neither
     strip is possible. A PM or council would pick this up straight away.
   - **Change d11 line 61** to "Deep soil zones of 8% of the site area are provided along the rear
     boundary."
   - **Change d14 and d15 line 21** to "- Deep soil zone along the rear boundary".
   - **Change d10 lines 48-49** to "Excavation extends to within 1 m of the northern boundary and the
     neighbouring dwelling. The basement is set back between 2.5 m and 4 m from the southern boundary
     and about 6 m from the rear boundary, where the deep soil zone is."

2. **The Sydney Water sentence contradicts itself.**
   - **Where:** d03 `Detail and boundary survey report - 18-22 Calloway Street.md` lines 71-72.
   - **What is wrong:** "no Sydney Water assets cross or adjoin the site" sits two lines after "drains
     to the Sydney Water sewer in the street". A main in the street does adjoin the frontage. The shape's
     own negation reads "no Sydney Water assets cross the site".
   - **Change:** "Connection to the existing water main in the street; no Sydney Water assets cross the
     site." Update the manifest d03 `flag.authority_asset` `says` to the same words. The trigger still
     fires and the value stays `no`.

3. **The consultants assess "DA drawings" before the DA set exists.**
   - **Where:** d13 `Acoustic report RA-26-0412-R01.md` lines 20-21 (dated 5 June); d12
     `BCA assessment report FBR-26-118.md` line 18 (dated 8 June).
   - **What is wrong:** both reports say they assess the architect's DA drawings. The only DA issue is
     A-DA-1101 Rev B of 12 June; Rev A of 25 May is "Preliminary issue for coordination".
   - **Change d13:** "It was prepared from the architect's drawings issued for coordination on 25 May
     2026."
   - **Change d12:** "This report assesses the architectural drawings by Arden Vale Architects issued
     for coordination on 25 May 2026".

4. **The trees paragraph reads as written for a test.**
   - **Where:** d11 lines 63-64.
   - **What is wrong:** three sentences say the same thing: "No trees on or next to the site are
     affected … no existing street or neighbouring trees are affected. No trees are pruned or removed."
     Separately, d03 lists three brush box street trees in front of the site, and d07 replaces the
     footpath, kerb and crossing beside them. A real design report would say why they are unaffected.
   - **Change:** "No trees on or next to the site are affected by the works. The new vehicle crossing
     is placed clear of the three existing brush box street trees, which are retained. New street trees
     will be planted." This keeps the shape negation and the `new-street-trees` trap. Avoid "protected",
     "pruned" or "removed" here: the `arborist_tree_impacts` and `arborist_pruning` patterns in
     `knowledge/profile/flag_readings.yaml` would then read the sentence as a yes.
   - **Also noted, no change:** d13 line 29, d12 lines 57-58 and d01 line 93 ("There are not multiple
     buildings") are also stilted. Each is a planned `says` that the trigger needs, so leave them.
