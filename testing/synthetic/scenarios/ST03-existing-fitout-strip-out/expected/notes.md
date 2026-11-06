# ST03-existing-fitout-strip-out notes

ST03-base with one change: Level 14 still holds the previous tenant's fit-out, and the landlord has agreed that
Avenor strips it out back to base building. 19 documents in 4 waves (base 18 plus the demolition plan drawing d19).
The trigger has no profile keys, so the profile equals the base's.

## What changed

- The base's negations ("handed over in base-building condition" in d01 and d03; the `no-strip-out` trap in d13) are
  replaced by the trigger's evidence. d05 adds the demolition plan, d10 strip-out and make-good costs, d06 making safe
  the old tenant services. Fee amounts are unchanged.
- d01's `make-good-at-expiry` trap stays: the future lease obligation still must not be read as this job's strip-out;
  the strip-out now comes from the job's own sentences.
- The implied pattern matches 'demolition plan' (d05) and 'strip-out' (d06, d10), not the trigger sentences
  ('stripped out', 'demolishes' do not match). Those give `site.demolition` with the refurb default `alter`. Only a
  works reading of d13's 'remove the existing partitions...' gives `remove`, so `fr.strip-out-fitout` may stay false:
  a likely miss.
- d13 says "kitchenette joinery" (render review), so the kitchen implied pattern should not fire there; joinery and hydraulic are in scope anyway, and no
  commercial kitchen is implied.

## Traps and misses

As the base, less `no-strip-out` (including `hdr.scale.tenancies` and `det.type_of_construction` keyed
`not_read`). d19 is a drawing, so it is not read.
