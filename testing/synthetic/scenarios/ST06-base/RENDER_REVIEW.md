# Render review: ST06-base

Reviewer: independent (brief 04r), 2026-11-07. Type: T06-sprinkler-pump-tank-upgrade.

## Verdict: **FIX**

There are 11 findings. The figures, the title blocks and the profile reading are right. Two findings matter most:

- **Finding 1.** d09 prices additional inspections at $450 each. Neither the manifest nor `fees.json` has that
  rate. If the app reads it, the scorer counts it against the app.
- **Finding 2 (writer departure 3).** The d13 title "Structural assessment: fire water tank…" matches the
  `fire_specialist_systems` trigger `structural[^.]{0,100}fire`. d13 is a clean document with no planned trap, and
  it denies nothing. One wrong "yes" from it against d12's "no" leaves the flag in conflict.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST06-base` gives
   `ok  testing/synthetic/scenarios/ST06-base` (exit 0).
2. **What I took from the documents before opening `expected/`.**
   - **Building:** a 1988 existing office building, Class 5 over a Class 7a basement car park, rise in storeys
     of 6, sprinklered throughout, approved to the deemed-to-satisfy provisions, single owner.
   - **Work:** a refurbishment. The sprinkler water supply gets electric duty, diesel standby and jockey pumps and
     a 150 kL tank in the enlarged basement pump room, and the vacant level 3 is fitted out as landlord's works.
     The other floors stay tenanted (partial occupation, live environment). The CBD site is boundary to boundary,
     with deliveries through the car park entry or the dock.
   - **Pathway:** two CDCs under the Codes SEPP, one under the Fire Safety Code and one for the fit-out. Zone E2,
     North Sydney LEP 2013. No heritage, bush fire, flood, coastal or contamination constraint; acid sulfate
     soils class 5.
   - **Team:** fire and hydraulic, structural, electrical, mechanical, interior designer, hazmat, certifier.
   - **Budget and contracts:** $2,700,000 ex GST for the whole job. Two AS 4000 lump sums, one stage each.
   - **Fees, ex GST:** Calder $101,000; Ferrule $24,000; Kilnworth $30,000 ($33,000 inc); Plenum $24,000; Lintel
     $22,000; Marlowe $68,000; Sentinel $4,500. Every table adds up.
   - **Invoices:**
     - CFH-INV-0412: $18,500 + $1,850 = $20,350.
     - SHS-INV-5530: $3,800 + $380 = $4,180.
     - FCE-2611: $7,500 + $375 = $7,875. The GST is 5%, so it is wrong.

   Against `profile.json`, `fees.json` and `invoices.json` this matches value for value. The one difference is
   d09's $450 rate (finding 1).
3. **Title blocks.**
   - The clean documents (d07, d09, d10, d11, d13, d14, d15, d16, d17, d18, d20) have label lines before the
     first `##`. Each has one `#` heading equal to `Title:`, and number, revision and date match
     `document_register.json`. The issue notes on d16 and d17 match "preliminary" and "for tender".
   - The messy documents (d02, d08, d19) have table title blocks, as their tier says.
   - The adversarial documents carry their planned traps.
4. **Stray triggers.** I ran every `flag_readings.yaml` and `determinants.yaml` trigger over the corpus. Planned
   hits: basement in d01, d06 and d13; staged in d05 and d12; flood, contamination and fire storage in d12;
   Sydney Water asset in d04; structural…fire in d06. Unplanned hits are findings 2, 5, 6, 7 and 8. The shape's
   trigger evidence phrases appear nowhere. The DOMAIN §10 kitchen trap does not fire: the documents say "tea
   point".

## The writer's three departures

- **(1) d14 "man-made mineral fibre lagging".** Accepted. The lint bans "synthetic" (`LEAK_WORDS`). MMMF is
  long-standing Australian hazmat usage, and no key reads this line. The manifest's d14 `content` should say
  "man-made mineral fibre" too (finding 11), so a re-render does not hit the lint again.
- **(2) d01 "fire water tank".** Accepted. d01 plans no `fire_special_hazards` trap and carries no denial of it,
  so "fire water storage tank" would have been an unplanned hit. The planned trap still fires in d12, line 59.
  Update the manifest's d01 `content` to match (finding 11).
- **(3) d13 title.** Not accepted as it stands; see finding 2. Rename the title. No key value changes:
  `flag.fire_specialist_systems` stays `"no"`, source d12. Only d13's title text changes, in the corpus, the
  manifest and `document_register.json`.

## Findings

1. **d09, `Fee proposal - Lintel Building Certifiers…`, "Additional inspections are charged at $450 each,
   excluding GST."**
   - **Problem:** the manifest does not plan this rate, and `fees.json` d09 has no rate line. The same kind of
     line in d06 ($480 each) is keyed as `kind: rate`. An app that reads the d09 rate is scored as wrong.
   - **Change:** delete that sentence, leaving "Item 3 allows for the mandatory critical-stage inspections and
     two further site inspections across both certificates." If the owner would rather keep the rate, add
     `{"label": "Additional inspection, each", "kind": "rate", "state": "priced", "amount_cents": 45000}` to
     `fees.json` d09 and add the rate to the manifest's d09 `content`.
2. **d13, `Structural assessment of tank location FCE-R-2207-01.md`, title and `#` heading "Structural assessment:
   fire water tank location and pump room slab".**
   - **Problem:** this matches `structural[^.]{0,100}fire`, and it reads on its face like a structural fire
     assessment. d13 is clean and has no trap. Clean documents in this manifest plan none.
   - **Change:** use `Title: Structural assessment: sprinkler tank location and pump room slab` and the same `#`
     heading. Change the manifest's d13 `title` and `title_block` and `document_register.json` d13 `title` to
     match. "Sprinkler" fires only `det.sprinklered`, which is true. Nothing else in d13 contains "fire", so the
     document then fires no `fire_specialist_systems` trigger.
3. **d09, "We have reviewed the concept prepared by Calder Fire and Hydraulic Engineers. On that review the
   certifier confirmed…"**
   - **Problem:** d09 is dated 4 March 2026. Calder's flow test was in March, and its options report (the
     concept) is dated 20 April. A PM would see that the certifier is reviewing a concept that does not exist
     yet. A certifier also would not call itself "the certifier" in its own letter.
   - **Change:** replace the first sentence with "We have discussed the proposed arrangement with Calder Fire
     and Hydraulic Engineers: new pumpsets and a tank inside the existing basement pump room and the adjoining
     store." Keep the shape negation sentence "On that arrangement the certifier confirmed the tank and pump
     house meet the Fire Safety Code development standards." as is, changing only its opening words.
4. **d04, "Applicant: Calder Fire and Hydraulic Engineers Pty Ltd, on behalf of Corvina…", dated 2 February
   2026.**
   - **Problem:** Calder had not been invited to propose (brief, 9 February) or appointed (end of March). Its
     own fee stage 2, invoiced as complete, then charges for obtaining this statement.
   - **Change:** "Applicant: Corvina Property Holdings Pty Ltd". d12 still compares its test with this
     statement.
5. **d01, "a 6-storey office building over a single basement car park level".**
   - **Problem:** this matches the `authority_asset` trigger `build(?:ing)? over`. It is unplanned, and d01
     carries no denial; the key's source is d12. The risk is low, because the question asks about a water
     authority asset.
   - **Change:** none to the text, because three `carries` quote it. Add a trap to the manifest's d01:
     `building-over-basement`, expect "flag.authority_asset stays no (d12)". List it in `notes.md` Traps.
6. **d03, "Occupants evacuate by egress via the fire stairs to the street."**
   - **Problem:** this matches the `fire_vulnerable_occupants` trigger `occupants?[^.]{0,80}evacuat`. d03 plans
     no such trap.
   - **Change:** "The building is evacuated by egress via the fire stairs to the street." This keeps the
     planned `fire-stairs-implied-stair` trap.
7. **d03 and d09, "Class 7a basement car park"; d12, three "basement" lines; d16 and d17, title "Basement fire
   pump room".**
   - **Problem:** these are `\bbasement` hits that the manifest does not list. Only d01's is a planned trap.
     The basement question now excludes an existing basement (F36), so the risk is low. But five documents
     besides d01 put the question, and only d06 and d13 answer it.
   - **Change:** no text change; the wording is what the real documents would say. Add a line to `notes.md`
     Traps: "incidental `basement` hits in d03, d09, d12, d16 and d17; flag.basement stays no (d06, d13)".
8. **d12, "a 150 kL modular fire / water tank".**
   - **Problem:** this matches `modular[^.]{0,100}fire` for `fire_specialist_systems`. The manifest's d12
     wording plans the phrase but lists no trap for it. d12 denies the flag itself, at line 61.
   - **Change:** add to the manifest's d12 traps `modular-fire-tank`, expect "flag.fire_specialist_systems
     stays no". d12 also dropped "storage" from "diesel fuel storage within a bunded day tank". That is fine,
     because line 59 still fires the planned fire-storage trap.
9. **d12, "The existing fire service connection is retained; Sydney Water confirmed the main is adequate.
   Sydney Water confirmed the main is adequate to refill the tank within the required time."**
   - **Problem:** the repeated sentence reads as negations stacked for a test.
   - **Change:** keep the first sentence (the shape negation). Make the second "The main can also refill the
     tank within the required time."
10. **Realism, test-shaped passages.**
    - **d13 "Scope limits":** three sentences in a row deny basement, adjoining excavation and groundwater, and
      two of them open "No excavation is involved". Change the last to "Groundwater is therefore not
      encountered and no dewatering is needed." Update the manifest's d13 `flag.groundwater` `says` to that
      wording, or the lint fails.
    - **d12:** the OSD, sewer main and road works lines sit under "## Compliance" in a fire options report. Put
      lines 65 and 66 under a new heading `## Site and authority matters`. No key changes.
    - **d01 "This is one building; we are not planning multiple buildings." and d12's negation lines:** these
      read as written for a test, but keys rely on them and the shape fixes the wording. I accept them.
    - **d03:** "has been given to the council" should read "has been given to the council and to the
      Commissioner of Fire and Rescue NSW", as an AFSS is. No trigger matches it. Also change "The owner has
      commissioned an upgrade" (20 January, before the brief went out) to "The owner is planning an upgrade".
11. **Manifest, d01, d12 and d14 `content`.** Bring it in line with the accepted departures: d01 "fire water
    tank"; d12 "150 kL modular fire water tank" (as rendered); d14 "man-made mineral fibre lagging". This is
    wording only, with no key change.

No change is needed for the "Fee stage 1/2" lines in d18. They sit on an invoice, filed through the invoices
drop, and match d05's planned `fee-stages-not-staged-works` trap.
