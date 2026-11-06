# ST12 scenario review (brief 03r)

Scenarios: ST12-base, ST12-sewer-main-in-backyard, ST12-da-pathway. Reviewer: independent (2026-11-06).

## Verdict: **FIX** (5 findings, all small; no key value is wrong)

Checker: `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST12-*` gives `ok` for all
three.

Fees, invoices and GST arithmetic check out. Each fee proposal's stages add up to its stated total. d07's figures are
GST-inclusive and divide by 1.1 exactly (1,320 / 550 / 1,870 inc GST become 1,200 / 500 / 1,700 ex GST). d16's
transposed total ($1,780 for $1,870) is flagged `arithmetic`. Register keys follow DISCOVERY §4.2 and §4.3. The d02
and d07 table title blocks and the d13 -> d14 supersession are declared as missed by design. Each variant changes only
its trigger's key. Fees, budget ($210k ex GST for 60 m2) and dates look like a real Sutherland Shire granny-flat job.

## Rulings on the author's questions

1. **`hdr.scale.garage_spaces: 0`: keep it, with `read_by_design: true`.** d12 is a read kind (report). The trigger
   `\bgarage\b` (taxonomy.yaml:90) fires on "Garage spaces: 0". An integer field takes every `\b(\d{1,4})\b` as a
   candidate (harvest.go:65, :233), so "0" is offered to Jev. The value is true. If Jev answers none, the app missed
   the value; the key is still right, and notes.md already says so. Do not drop the key from the shape: that would
   hide a true value, and ST01 and ST10 key the same field. d01's "no garage or car space" gives no candidate, which
   is correct.
2. **`det.site_class`: yes, add `det.site_class: M` to the T12 base profile.** Comment it as R10 applied by analogy,
   pending the owner's confirmation, because R10 is a T01 ruling. M fits d11 (stiff silty clay over weathered
   sandstone at Gymea). No T12 trigger changes the site class. The determinant has `profile_group: site`, so the
   value can be set. All three scenarios already key M from d11.
3. **da-pathway evidence: sound.** The trigger's `profile` is `[hdr.cond.planning]` only. Phrase 1 (a 420 m2 lot)
   would change `hdr.scale.site_sqm`, and phrase 2 (heritage conservation area) would change `det.heritage_status`
   and the heritage scope. Check 6 forbids both. Phrase 3 is the shape's own wording. See finding 4 for how it reads.

**"Existing building" in d03 cannot change the work type through code.** No code path derives `hdr.work_type` from
`det.existing_building`. `proposals.go:110` only goes the other way (work type new sets existing_building false), and
`rfp/drivers.go:285` derives the RFP flag from the work type. One residual risk is left: d03 is a report, and its
`## Existing house` passage falls within the first 12 passages, so it is header-routed (`questions.go:67`). Jev is
therefore asked the work type on that passage. If Jev picks refurb or extend, that is a Jev reading error and shows
as a header conflict with "new" from d01 and d12. It is not a key error. Finding 1 declares it.

## Findings

1. **All three, d03, trap `existing-building-retained`: the work-type risk is not declared.** Evidence: the passage
   is header-routed, as above. Change: append to the trap's `expect`: "; hdr.work_type stays new (no code path from
   existing_building to work type; a refurb or extend pick here is a Jev reading error)".
2. **All three, d03 carry `hdr.cond.access_constraints` (key `urban_constrained`): the stated evidence leans the
   other way.** The sentence is the restricted-side-access negation ("3 m driveway ... gives truck access to the rear
   yard"). Jev can fairly pick `unrestricted` from it. "Only way in, no rear lane, family's parking" is a weak
   counterweight. Change: add one sentence to the same d03 passage, or to d01's project description: "Construction
   access is constrained: the only way to the backyard is the side driveway past the occupied house, through a double
   gate." Keep the negation sentence as it is.
3. **ST12-da-pathway d18 (SEE): an undeclared garage trap.** "the rear neighbour's garage backs onto the fence" sits in
   the rear-setback passage with "1.5 m" and "0.9 m". That fires `\bgarage\b`, and integerRe offers 1, 5, 0 and 9 as
   garage_spaces candidates. Change: declare a d18 trap `neighbour-garage` with the expect "hdr.scale.garage_spaces
   stays 0 (from d12); the neighbour's garage is not the project's". Or reword it to "the rear neighbour's shed".
4. **ST12-da-pathway d01, d12 and d18: the evidence reads backwards.** "The flat's 1.5 m rear setback exceeds the CDC
   rear setback standard" suggests the setback is larger than the standard. A PM would query that. Change: keep the
   shape phrase word for word, but put a plain sentence before it with no SEPP figure: "The flat sits closer to the
   rear boundary than the complying development standards allow; the flat exceeds the CDC rear setback standard, so
   we will lodge a DA with council." This does not change the key. Optionally, the shape owner could reword phrase 3
   to "does not meet".
5. **ST12-da-pathway d12, d13, d14 and notes.md: the geometry does not add up.** In the base, the flat is 3.0 m from
   the rear and about 7 m from the house, which makes the flat about 7.9 m deep (17.87 m of yard behind the house).
   At a 1.5 m rear setback the gap to the house is about 8.5 m, not 9 m. Change "about 9 m" to "about 8.5 m"
   everywhere. `close-to-main-house` stays denied either way.

## Not findings (checked)

- Sewer variant: d03, d06 and d12 carry the shape evidence. d05 and d11 also state the asset and support `yes`.
  Leaving them out of `sources` is harmless. `det.sewer_connection` stays reticulated. The piers support the
  `substructure.piles` work item.
- `flag.community: no` with "Neighbour notification under council's community participation plan; no public
  exhibition" in the DA variant matches the reviewed ST01, ST05 and ST10 keys.
- Invoices are keyed in `invoices.json`, not in the register. Labels such as "Invoice number" are not register
  labels, which is expected, because the invoice reader is Jev.
