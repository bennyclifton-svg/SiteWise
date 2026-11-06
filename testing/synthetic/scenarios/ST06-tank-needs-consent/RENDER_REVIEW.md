# Render review: ST06-tank-needs-consent

Reviewer: independent (brief 04r), 2026-11-07. Type: T06-sprinkler-pump-tank-upgrade. Variant of ST06-base
(trigger `tank-needs-consent`).

## Verdict: **FIX**

There are 9 findings. The figures, the title blocks, the copies and the planning pathway are right. No key value
is wrong. Two findings matter most:

- **Finding 1.** d22 line 29 says "An enlarged basement fire pump room". The basement question counts "an
  extension of an existing" basement as basement works. Read literally, "enlarged basement" is that. This is a new,
  unplanned `\bbasement` hit against a key of **no**, and d22 does not deny it.
- **Finding 5.** d21 line 17 and d22 lines 14 and 29 say "fire water storage tank" / "fire pump room and fire water
  storage". These match the `fire_special_hazards` trigger `fire[^.]{0,90}storage`. They are unplanned, and neither
  new document denies the flag. Only d12 does.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST06-tank-needs-consent`
   gives `ok  ST06-tank-needs-consent` (exit 0).
2. **Copies.** I ran `cmp` against `ST06-base/corpus`. The 14 unchanged documents (d01 to d08, d11, d14, d15, d18
   to d20) are byte-identical to the base. Exactly d09, d10, d12, d13, d16 and d17 differ, and d21 and d22 are new.
   That is the manifest's `changed_from_base` set. Each diff is limited to what the notes describe, plus the d09
   fee table and exclusions. Nothing in the base is missing from the variant.
3. **What I took from the documents before opening `expected/`.**
   - **Building:** an existing 1988 office building, Class 5 over a Class 7a basement car park, rise in storeys 6,
     sprinklered throughout, DTS only, single owner (Corvina). It is in Zone E2, North Sydney LEP 2013, with no
     heritage, flood, bush fire, coastal, contamination or rail constraint. Acid sulfate soils are class 5.
   - **Work:** a refurbishment. Electric duty, diesel standby and jockey pumps and a 150 kL tank go into an enlarged
     pump room, which takes the basement store and two car spaces. The vacant level 3 is fitted out as landlord's
     works. The other floors stay tenanted. The CBD site has no on-street loading.
   - **Pathway:** a DA. At the 16 April pre-lodgement meeting, council said the tank and pump house are not
     complying development, because losing two spaces leaves the building below the original consent's number.
     The fit-out is in the same DA. A construction certificate follows, issued by Lintel as PC. The DA is
     determined under delegation, with no panel and no cl 4.6 variation. I read d02's "Complying development may be
     carried out on the land" as land eligibility only. It does not conflict.
   - **Team:** fire/hydraulic, structural, electrical, mechanical, interior designer, hazmat, certifier and town
     planner. I would also expect a BCA consultant, though no proposal was received.
   - **Budget and contracts:** $2,700,000 ex GST for construction ($1.2M fire, $1.5M fit-out). Two AS 4000 lump
     sums, one stage each.
   - **Fees, ex GST:** Calder $101,000; Ferrule $24,000; Kilnworth $30,000 ($33,000 inc, rate $190 ex); Plenum
     $24,000; Lintel $18,500 (CC model); Marlowe $68,000; Sentinel $4,500; Quillon $15,500. Every table adds up.
   - **Invoices:** CFH-INV-0412 $18,500 + $1,850 = $20,350; SHS-INV-5530 $3,800 + $380 = $4,180; FCE-2611
     $7,500 + $375 = $7,875. The FCE-2611 GST is wrong (5%), which is the planned trap.

   This matches `profile.json`, `fees.json` and `invoices.json` value for value. `hdr.cond.planning` = da is
   sourced from d09, d12, d21 and d22, and each states it. The planner fee labels and the "car parking assessment"
   wording match `fees.json` and the shape's RFP `must_include`.
4. **CDC wording.** Outside d02, no document says "CDC", "Codes SEPP" or "Fire Safety Code". The only "complying"
   hits are council's advice sentence in d09, d12 and d21, and that sentence says the works are *not* complying
   development. The base negations ("tank fits within the existing basement store", "certifier confirmed … Fire
   Safety Code development standards", "proceed as complying development") are gone from the corpus. d09's
   reworded fire-upgrade line is a soft negation. It does not use the `da-fire-upgrade` evidence wording.
5. **Stray triggers.** I ran every `flag_readings.yaml` and `determinants.yaml` trigger over the variant and the
   base, then diffed the hits. These are the new hits:
   - **Basement:** d09 line 18; d12 lines 39 and 41 (d12 is planned, `car-spaces-not-basement`); d13 line 15
     (base wording); d22 lines 23 and 29 (findings 1 and 6).
   - **Fire special hazards:** d21 line 17; d22 lines 14 and 29 (finding 5).
   - **Authority asset:** d22 line 23, "building over" (finding 4).
   - **Rail:** d22 line 48, "railway" (finding 3).
   - **Panel and cl 4.6:** d09 and d22. Both are correct denials that agree with the key.
   - **Architect lodges:** "pre-lodgement" and "Lodgement" in d09, d12 and d21. No template reads this, so it is
     harmless.

   No shape evidence phrase for any other trigger appears. "Car spaces", "car park" and "bays" match no trigger.
   The `\bbasement` hits come only where the word "basement" is adjacent.
6. **Title blocks.**
   - The clean documents d09, d10, d13, d16, d17, d21 and d22 have label lines before the first `##`, with one `#`
     heading equal to `Title:`. Their number, revision and date match `document_register.json`: d09 is 24 April
     2026; d21 is QP-26-112, 27 April; d22 is QP-26-112-SEE, 15 June; d17's "Issued for development application"
     matches "for DA".
   - d12 (adversarial) keeps its planned traps.
7. **Timeline.** d13 (14 April) is consistent with the 16 April meeting, and so are d12 (20 April), d09 (24 April),
   d21 (27 April), FS-101 A and B (11 May and 2 June) and d22 (15 June). The exceptions are findings 8 and 9.

## Findings

1. **d22 `Statement of environmental effects QP-26-112-SEE.md`, line 29, "- An enlarged basement fire pump room
   and fire water storage tank."**
   - **Problem:** this is a `\bbasement` hit that the manifest does not plan (d22 has no traps), and d22 never
     says that no basement is built. The basement question counts an extension of an existing basement.
     "Enlarged basement" reads as one. A wrong yes switches on retention, dilapidation and shoring.
   - **Change:** "- An enlarged fire pump room and a new fire water tank, both within the existing basement." Make
     the same change to the manifest's d22 `content` ("enlarged fire pump room and fire water tank in the existing
     basement"). This line is not quoted in `carries`. The change also removes one finding-5 hit.
2. **d22, lines 71 and 72, "The level 3 fit-out is internal only. It improves the building's sprinkler water supply
   for every floor and allows a vacant floor to be re-let."**
   - **Problem:** the fit-out does not improve the sprinkler supply. The sentence has lost its subject, and a
     planner would not write it.
   - **Change:** "The level 3 fit-out is internal only. Together, the works improve the building's sprinkler water
     supply for every floor and allow a vacant floor to be re-let."
3. **d22, lines 47 and 48, "The site is a short walk from North Sydney railway station".**
   - **Problem:** this matches the `rail` trigger `railway`. It is unplanned. The key is **no**, from d02.
   - **Change:** "The site is a short walk from North Sydney station and from frequent bus services."
4. **d22, lines 22 and 23, "a 6-storey office building over a single basement car park level".**
   - **Problem:** this matches the `authority_asset` trigger `build(?:ing)? over`. It is unplanned in d22. The base
     plans it in d01 only, as `building-over-basement`. It is also a second `\bbasement` hit.
   - **Change:** "a 6-storey office building with a single basement car park level". No `carries` quotes it.
5. **d21 `Fee proposal - Quillon Planning…`, line 17, "the new fire water storage tank"; d22 line 14, "for the
   enlarged fire pump room and fire water storage tank".**
   - **Problem:** both match `fire[^.]{0,90}storage` for `fire_special_hazards`. They are unplanned. Neither
     document denies the flag; d12 line 61 is the only denial.
   - **Change:**
     - d21: "We understand that the new fire water tank and the enlarged pump room will take two car spaces…".
     - d22 line 14 is quoted by the `hdr.cond.planning` carry. Either change it to "for the enlarged fire pump room
       and its new fire water tank, and the level 3 fit-out" and update that `says`, or keep the text and add a d22
       trap `fire-storage-not-special-hazard`, expect "flag.fire_special_hazards stays no (d12)".
     - Line 29 is covered by finding 1.
6. **d09 `Fee proposal - Lintel Building Certifiers…`, line 18, "two of the basement car spaces will be lost to the
   new tank".**
   - **Problem:** this is the trigger evidence. It is also a `\bbasement` hit, and the manifest plans it only for
     d12, so d09's traps are `[]`.
   - **Change:** no text change. Add to the manifest's d09 traps `car-spaces-not-basement`, expect "flag.basement
     stays no (d06, d13)". In `notes.md` Traps, extend the d12 line to "d09 and d12". If findings 1 and 4 are not
     applied, also list d22 lines 23 and 29.
7. **Manifest d02 traps.**
   - **Problem:** `notes.md` names d02's "Complying development may be carried out on the land" as a trap, with
     planning staying da. The manifest's d02 traps hold only `acid-sulfate-class-not-ncc`. Because profile
     `conflicts` is `[]`, a cdc reading of d02 would score as a conflict, and that trap needs recording.
   - **Change:** add the d02 trap `complying-on-land-not-pathway`, expect "hdr.cond.planning stays da (d09, d12,
     d21, d22); no conflict". The corpus does not change.
8. **d21, line 19, "At the pre-lodgement meeting on 16 April 2026 council set out its position."**
   - **Problem:** Quillon bills $2,500 for "pre-lodgement meeting notes" but does not say it attended. Its proposal
     is dated 11 days after the meeting. A PM would ask who took the notes.
   - **Change:** "At the pre-lodgement meeting on 16 April 2026, which we attended with Lintel Building Certifiers
     at your request, council set out its position." The fee label stays, because it is keyed.
9. **d10 `Fee proposal - Marlowe Interiors…`, line 27, "Document the fit-out for the development application, the
   construction certificate and tender."**
   - **Problem:** d10 is dated 5 March 2026. That is six weeks before council's 16 April advice and before the tank
     position was fixed, yet it already names the DA and the CC. The key does not depend on d10, and it agrees with
     da, but the line is anachronistic.
   - **Change:** "Document the fit-out for certification and tender." This matches its fee item 4. Update the
     manifest's d10 `changed_from_base` note to match.

## Realism

d21 and d22 read as a real North Sydney planner's proposal and a short SEE. The car-parking argument (maximum DCP
rates in the centre, public transport, the spaces were the vacant tenancy's) is the right argument. FS-101 Rev B's
relocated line marking and wheel stops is a good touch. d22 is thin for an SEE, with no s 4.15 headings and no
notification section. That is acceptable for a minor DA. Apart from findings 2, 8 and 9, nothing reads as written
for a test beyond what the base review already accepted.
