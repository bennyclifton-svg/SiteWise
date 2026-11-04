# Review: ST01 scenarios (base, cdc-pathway, trees-affected)

Reviewer: independent (brief 03r), 2026-10-30.

## Verdict: **FIX**

Four findings. Finding 1 is the most serious: it makes the base and the CDC variant
expect a value the app cannot read.

## Checks

1. **Checker.** `validate_expected.py` reports `ok` for all three scenarios.
2. **Faithful to the shape.** No base document fires a trigger. The traps (d02 CDC-eligible land,
   d03 sewer main in the street, d07 OSD negation, d10 "to a depth of 300 mm", d11 garden beds and
   "No demolition") match DOMAIN.md §10 and the expected outcomes are right.
3. **Register.** See findings 2 and 3. The rest agree with DISCOVERY.md §4.3. The d07 table block is
   declared as missed in the notes. That d13 supersedes d12 is also declared as unsupported.
4. **Profile.** The values agree with `carries`, and `read_by_design` is right (only d04 email holds
   the start date). See finding 1.
5. **Fees and invoices.** d07 GST-inc figures /1.1 are right ($9,900 → $9,000; stages 3,000 /
   4,500 / 1,500). Stage sums equal the stated totals. `gst_inc_normalised` and the ledger's
   `arithmetic` flag (d15: $3,800 + $380 stated as $4,280) are flags the app really raises.
6. **Variants.** cdc-pathway changes only `hdr.cond.planning` (d01, d05, d11 changed, d14 SEE removed;
   the panel and community wording was adjusted to stay coherent, and their values are unchanged).
   trees-affected changes only the two arborist flags and adds d18 and d19. Both use the shape's
   evidence wording.
7. **Realism.** A $1.45M architect-designed 300 m² house in Denistone (Ryde, R2) is plausible.
   The fees are plausible: architect $80k to tender, structural $8k, civil $9k, geotech $3.8k,
   BASIX $2k and arborist $3.4k. The dates run Feb to May 2026 to a Feb 2027 start. That start is
   tight but credible.

## On the author's shape comments (orchestrator question)

Keying both values was **right**, and the shape should add them to its base `profile`:

- `flag.arborist_offsite_trees`: the `trees-affected` trigger lists it in `profile`. Without a base
  value, the variant has nothing to differ from. It is a read flag
  (`knowledge/profile/flag_readings.yaml`), so "no" is a legitimate key **if a document states it in
  words the triggers catch** (see finding 1).
- `det.site_class` = M: `knowledge/determinants.yaml` `site_class` has `profile_group: site`, the
  trigger `site class(ification)?`, option `M`, and `stated_in` the geotechnical report. d10 (a
  report, which is read) says "Site classification: Class M in accordance with AS 2870". The value
  is readable and correct. It also gives the `reactive-site-class` variant its base.

## Findings

1. **ST01-base and ST01-cdc-pathway, d11: `flag.arborist_offsite_trees` = "no" cannot be read.**
   The app asks Jev only when a passage matches the flag's triggers:
   `(?:neighbou?r|adjoin|street|council|off[- ]?site)[^.]{0,70}\btrees?\b` or the reverse. d11
   says "No trees on or next to the site are affected by the works", and "next to the site" matches
   neither pattern. The flag is therefore never asked and stays unstated, not "no". d03 ("trees over
   5 m with spreads, street features") may trip the trigger, but it states no impact either way, so
   it cannot settle "no".
   **Change:** in both manifests, d11 `carries` `flag.arborist_offsite_trees` `says` →
   "No neighbouring or street trees are affected by the works". The d11 document text must use that
   sentence. Keep the key "no", sourced from d11.

2. **All three scenarios, d02: the number and date are in a table.** The title block is
   `| Certificate no | PC-2026-01187 | | Issue date | 10/02/2026 |`. DISCOVERY.md §4.3 says nothing
   inside a table is read, and `Certificate no` is not a number label anyway. The key expects
   `number: PC-2026-01187` and `date: 2026-02-10`, and notes.md does not declare them missed. The
   title "Planning Certificate" comes from the `#` heading and is fine.
   **Change:** add d02 number and date to "Missed by design" in all three notes.md files. Keep the
   true values, as for d07.

3. **All three scenarios, d05: two different title candidates.** The title block has
   `Title: Fee proposal: architectural services` and `# Fee proposal`. A `# Heading` that differs
   from `Title:` gives no title (brief 03r §3), yet the key expects "Fee proposal: architectural
   services".
   **Change:** make the d05 heading `# Fee proposal: architectural services`. Otherwise, declare the
   d05 title missed in notes.md and say why.

4. **ST01-trees-affected, d03 trap `trees-present-not-affected`.** Its `expect` still says "both
   arborist flags stay no". In this variant both flags are "yes" from d11, d18 and d19.
   **Change:** reword `expect` to "d03 alone does not set either arborist flag: presence is not an
   impact".
