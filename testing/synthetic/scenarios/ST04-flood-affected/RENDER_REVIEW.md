# ST04-flood-affected render review

Reviewer: independent read (brief 04r), 7 November 2026. Base: ST04-base (corpus final and reviewed).

## Verdict: PASS

None of the findings below changes a key. Findings 1 to 4 are realism fixes in the rewritten documents.
Making them would make the variant read better, but scoring does not need them.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST04-flood-affected`
   gives `ok  ST04-flood-affected`.
2. **Copies.** 15 of the 19 documents are byte-identical to ST04-base (`cmp`): d01, d03, d04, d05, d06,
   d08, d09, d10, d11, d12, d15, d16, d17, d18 and d19. Only d02, d07, d13 and d14 differ, which matches
   `changed_from_base`. The diffs are confined to the flood content.
3. **PM read before the keys.** The project is a new Class 7b warehouse with a Class 5 office, 8,000 m2,
   on a vacant benched E4 lot in Penrith, by local DA with AS 4902 D&C and novation, at $14.8 m. These
   conclusions are unchanged from the base. New in this variant: the lot is flood affected. The d02
   certificate puts it within the flood planning area, with FPL 31.2 m AHD (1% AEP flood level 30.7 m
   AHD plus 0.5 m). An overland flow path runs along the eastern boundary (d13, d14). The floor is set
   at the FPL (d07, d13, d14). The civil fee is $103,400 inc GST: four phases of $35,200, $15,400,
   $30,800 and $22,000, which add up. That is $94,000 ex GST. The invoices are as in the base.
4. **Against the keys.** `profile.json` has `det.flood_hazard` true (d02, d07, d13, d14) and
   `det.flood_planning_level` 31.2 (d02, d14, `not_read` by design). `fees.json` adds the $14,000 ex
   flood phase, and the stated total is $94,000 ex. Both agree with the documents. `invoices.json` and
   `document_register.json` are unchanged from the base. The title blocks of d02, d07, d13 and d14 match
   the register: number, revision, title and date.
5. **Stray triggers.** The `flood-affected` evidence appears only where planned. "The land is within the
   flood planning area" is in d02, d13 and d14. "an overland flow path runs along the eastern boundary"
   is in d13 and d14. No other shape trigger's evidence appears in the four rewritten documents.
   DOMAIN.md §10 traps in those documents are all planned ones: d13 floodlights, public exhibition,
   planning panel and TfNSW denial; d14 hydrocarbon contamination; d02 negated contamination and
   heritage. There is no storage or racking within 90 characters of fire. No m2 figure appears in the
   d02 flood item or the d14 Flooding section, so the FPL passages offer no area candidates.
6. **Tiers.** d02 and d07 are still messy, with table title blocks. d13 and d14 are still clean label
   lines that carry their planned traps (adversarial).

## The RL 46.20 to RL 31.20 change in d14

- **Realistic.** In NSW the flood planning level already includes the freeboard. The FPL is the 1% AEP
  flood level plus 0.5 m, as d02 states. Penrith LEP 2010 cl 5.21 and DCP 2014 Part C3 set industrial
  floor levels at or above the FPL. A floor at the FPL is the normal outcome. Adding freeboard on top of
  the FPL would count it twice. "500 mm above the 1% AEP flood level, at RL 31.20 m AHD, the flood
  planning level" is arithmetically and procedurally right, and d07, d13 and d14 agree.
- **No contradiction in the copies.** No copied document gives an absolute ground level. The d03 survey
  gives only "a gentle fall of about 1.4 m from west to east". The d15 and d16 site plans say "Levels are
  to AHD" with no RL. The d11 geotechnical report has no RLs. The fall towards the east agrees with an
  overland flow path and drainage easement along the eastern boundary. The pad is raised 0.4 m to reach
  31.20, which puts the benched pad at about 30.8, just above the 30.7 flood level. That is consistent
  with a lot that is only marginally flood affected.
- **External plausibility (no change).** Erskine Park estate land is generally higher, roughly 40 to
  55 m AHD. The base 46.20 was the more typical figure for the suburb, and a 1% AEP of 30.7 m AHD reads
  like the lower South Creek or Ropes Creek floodplain. No document or key can test this, and the
  scenario is invented.
- **Key effect.** The change is required. `det.flood_planning_level` = 31.2 cites d14 as a source.
  Keeping 46.20 would have put a second, contradictory floor level beside the 31.2 FPL. The base 46.20
  touched no key. Both "31.20" sentences (Bulk earthworks and Flooding) are free of m2 figures.

## Findings

1. **d14 `Civil and stormwater report CCE-R-0412 - 14 Ironstone Circuit.md`, Flooding:** "the OSD tank
   sits above the 1% AEP flood level so that it is not drowned out in that event". The car park can be
   no higher than about 31.5 m AHD, since the floor is 31.20 and the lot falls 1.4 m. A 540 m3 tank under
   it must have its base below 30.7. A civil engineer would check the outlet against tailwater, not claim
   the tank is above the flood. This wording comes from the manifest content, so update both. Change to:
   "and the OSD outlet is designed for a tailwater level equal to the 1% AEP flood level of 30.7 m AHD."
   No key effect.
2. **d14, Bulk earthworks:** "with cut and fill balanced on site" contradicts the Flooding section's "the
   pad is raised about 0.4 m above its benched level". Raising the building pad 0.4 m needs more than
   3,000 m3 of fill. Change the sentence to: "Earthworks trim the pad to the design levels for the
   warehouse, hardstand and car park; the building pad is raised about 0.4 m (see Flooding below) using
   cut from the OSD tank and car park, so earthworks stay close to balanced." Avoid "excavated to a depth"
   (basement trap). No key effect.
3. **d14, Bulk earthworks (wording copied from the base, now material):** "about 0.3 m above the finished
   hardstand at the docks". Recessed docks with levellers put the truck apron about 1.2 m below the floor,
   which here would be about 30.0 m AHD, below the 1% AEP level of 30.7. A flood engineer would say how
   the dock aprons are handled. This is the owner's choice because the base is reviewed. The smallest
   change: "about 1.2 m above the recessed dock aprons, which fall away from the dock face to grated drains
   and are the only external areas below the 1% AEP flood level." No key effect. Do not add an m2 figure.
4. **d13 `Statement of environmental effects - 14 Ironstone Circuit.md`, Flooding and stormwater:** "meets
   the flood planning provisions of the Penrith Development Control Plan". A real SEE cites the LEP clause.
   Change to: "meets clause 5.21 (Flood planning) of Penrith Local Environmental Plan 2010 and the flood
   planning provisions of Penrith Development Control Plan 2014." This adds no trigger: it is not clause
   4.6 and has no m2 figure.
5. **Note, no change. d02, item 7:** a s10.7 certificate usually states only that flood related controls
   apply. The FPL figure normally comes in council's flood advice or a flood information letter. Some
   councils add it as a s10.7(5) notation, and the sentence that follows ("Applicants should seek council's
   flood advice") makes it read as council wording. It is acceptable as written.

## Realism

d02, d07, d13 and d14 read like the real documents. The flood text uses ordinary NSW practice terms
(flood planning area, FPL, 1% AEP, flood impact assessment, flood compatible materials, overland flow
path) and does not read as written for a test. The only physical slips are findings 1 to 3.
