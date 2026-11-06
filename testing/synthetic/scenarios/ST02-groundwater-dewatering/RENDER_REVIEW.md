# Render review: ST02-groundwater-dewatering

Reviewer: independent (brief 04r), 2026-11-06. The base corpus (ST02-base) is already reviewed. This
review covers the two rewritten documents (d10, d11) and checks the 16 copied documents against the
groundwater condition.

## Verdict: **FIX**

There are three findings. None changes a scored value: `flag.groundwater` reads yes from d10 and d11, and
every other key, fee and invoice is the same as the base. Finding 1 is a real contradiction that a PM or
geotechnical reader would catch at once. It comes from the shape's evidence string, so the shape needs
an owner edit too.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST02-groundwater-dewatering`
   → `ok  testing/synthetic/scenarios/ST02-groundwater-dewatering`.

2. **Cold read (before opening `expected/`).** My conclusions match the base: a new six-storey Class 2
   over Class 7a, 64 apartments, a DA to the Ryde LPP, AS 4902 D&C with novation, mid quality, vacant,
   single owner, urban-constrained, $27.6M ex GST, and the same fees and invoices (HP-0093 GST at 15%).
   The one change is the site condition: **groundwater is above bulk excavation level.** The
   consequences are dewatering under a WaterNSW approval, treated discharge, monitoring until
   excavation, and a tanked basement that resists hydrostatic pressure and uplift (d10). d11 confirms
   the tanked basement.

   **Compared with `expected/`:** `profile.json` differs from the base only in `flag.groundwater` =
   yes (sources d10, d11), and both documents carry it. `fees.json`, `invoices.json` and
   `document_register.json` are identical to the base and still agree with the corpus. There are no
   differences.

3. **Groundwater contradictions in the copied documents.** I searched all 18 documents for seep,
   groundwater, water table, drain, dry, dewater, tank, waterproof, membrane, sump, inflow, pump, RL and
   AHD.
   - No copied document mentions seepage only, no groundwater, dry standpipes, a drained basement, a
     drained cavity or a sump. The base phrases ("No groundwater was encountered", "Minor seepage",
     "A drained basement is suitable", "drained cavity basement walls with sub-floor drainage to a sump")
     are gone from d10 and d11. They do not appear anywhere else.
   - d06 (structural fee, 12 March) allows for "a suspended or on-ground basement slab as the ground
     conditions allow" and assumes a contiguous piled wall "pending the geotechnical report". It was
     written before the investigation, so it does not contradict the finding.
   - d07 (civil fee): "a pump-out system for the basement ramp and car park floor drainage" is surface
     stormwater, which a tanked basement still needs. It is not a groundwater sub-floor drain, so there is
     no contradiction.
   - d08 (geotech fee): "water level readings over four weeks" describes the investigation. It is not a
     finding.
   - d01 (budget unchanged at $27.6M, "covers the building works, the basement") is consistent with a
     one-change variant. It is not a contradiction.

4. **Stray triggers.** The new d10 and d11 text adds "WaterNSW", "discharge criteria", "treated",
   "uplift", "hydrostatic" and "membrane". None of these matches another T02 trigger's evidence (for
   example contamination, the Sydney Water asset or acid sulfate) or a DOMAIN.md §10 trap. The
   `groundwater` flag pattern (`ground ?water|water table|dewater`) fires on d10 lines 38-49 and d11
   line 57, which is planned. The copied documents are unchanged from the reviewed base.

5. **Title blocks.** d10 is now `clean`. It has Title, Report no `BRG-R-1152-01`, Revision `0` and
   Report date `24 April 2026` before the first `##`, and one `#` heading equal to Title. All of these
   match the register. d11 has Title, Document number `AVA-DR-01`, Revision `A` and Date `15 June 2026`,
   with a matching heading, and also matches the register. d10 no longer carries a trap, so the `clean`
   tier is right.

6. **Realism.** d10 reads like a real Sydney geotechnical report, apart from findings 1 and 2. d11's
   sentence is plain but believable for a DA design report.

## Findings

1. **d10 line 40, "Groundwater was measured at RL 14.2"**: the level cannot be right. The survey (d03) is
   on AHD and puts ground at about RL 40.7 to 43.2. The existing floor levels are 40.74 to 41.62, and the
   land falls 2.5 m to the street. The 7 m basement (d10 line 53) therefore puts bulk excavation level at
   about RL 33.5 to 34. The boreholes reach only 11.6 to 12.2 m, about RL 29 to 31, so the standpipes
   could not have measured RL 14.2 at all. As written, the level sits about 20 m below excavation, which
   contradicts "about 3 m above bulk excavation level" in the same sentence. Any geotechnical engineer
   or PM reading d03 and d10 together would query it.
   **Change:** "RL 14.2" → "RL 37.2 AHD" in d10 line 40. That puts it about 3.5 to 4 m below ground, in
   the shale, which fits line 64's "inflow through the shale". Make the same change in the evidence
   string in `types/T02-apartments/shape.yaml` line 802 (owner edit; the shape is the truth), in the
   manifest d10 `says` and `content`, and in `expected/notes.md` line 3. Then re-run lint, which may
   check the evidence string word for word.

2. **d10 lines 62-64, "A contiguous piled wall … The pile toes should be cut off into the sandstone to
   limit inflow through the shale"**: with groundwater 3 m above excavation, groundwater gets in through
   the gaps between contiguous piles, not under the toes. The base's shotcrete infill panels went out
   with the old Drainage section, so nothing now closes those gaps. A real report would recommend a
   secant piled wall, or contiguous piles with sealed infill.
   **Change:** replace the first bullet with: "A secant piled wall, with piles socketed into the sandstone
   below bulk excavation level, is recommended for the full perimeter to cut off groundwater inflow
   through the soil and shale during excavation." Update the manifest d10 `content` to match. d06's
   pre-investigation "contiguous piled wall" can stay, because the report supersedes an early allowance.

3. **d10 line 40, wrapping and phrasing**: this is the only line in the corpus that is not wrapped at
   about 100 characters (about 170). It also joins the three shape evidence phrases into one sentence
   with semicolons, which reads as pasted. **Change:** keep the evidence phrases verbatim and wrap
   them as prose:
   "Groundwater was measured at RL 37.2 AHD, about 3 m above bulk excavation level. The level was
   steady in both standpipes over the four weeks of readings, so dewatering will be required during
   excavation, and a tanked basement is recommended." Then delete the current line 41.

## Observation (no change)

- A DA whose basement needs a WaterNSW dewatering approval would normally be lodged as integrated
  development. The shape lists this under `authority` as `verified: false`. Neither the d09 planner fee
  (10 March, written before the finding) nor d11 (June) mentions it. No key depends on it, so I propose
  no corpus change. It is noted for the owner in case the expected RFP or program later keys an
  integrated referral.
