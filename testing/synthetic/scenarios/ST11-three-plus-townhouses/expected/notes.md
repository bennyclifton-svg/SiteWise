# ST11-three-plus-townhouses notes

The base site redeveloped as multi dwelling housing: one attached row of 4 two-storey, two-bedroom townhouses
(460 m2 GFA in total, 115 m2 each) with community title subdivision into 4 lots and association property and a common
driveway, on the same 700 m2 R2 lot. 21 documents in
4 waves.

`hdr.scale.dwellings` (2 -> 4) and `hdr.scale.avg_dwelling_size_sqm` (230 -> 115) differ from the base; the shape
trigger lists both since review finding 1.

## Changes from the base

- d01, d13 and d16 carry the evidence: "Multi dwelling housing comprising 4 two-storey townhouses" and
  "community title subdivision into 4 lots with a common driveway", plus "4 dwellings on proposed Lots 1 to 4".
- Every passing "dual occupancy", "both dwellings", "Lots 1 and 2", "Torrens" and "two driveways" mention in
  d01, d05, d06, d07, d10, d13, d14, d15 and d16 is reworded. d04, d12 and d17 had passing dual-occupancy wording
  reworded. d02, d08, d09, d11, d18 and d19 never named the dwelling count or the dual occupancy, so they are
  unchanged.
- d03 moves the large gum into the neighbouring rear yard, because the row of 4 runs the depth of the lot; d13
  says the gum is well clear of Townhouse 4. Neither arborist flag changes.
- The shared driveway and bin parcel is "association property", the community title term, throughout.
  A-101 lot areas are unequal (Lot 1 includes the front setback) and total 512.0 m2 plus 188.4 m2.
- Added d20 traffic and parking and d21 waste management fee proposals (trigger adds both packages).
- The zone stays R2 (review finding 2; the shape no longer offers an R3 evidence sentence); permissibility is put
  under the Housing SEPP low-rise housing provisions (verified: false).
- "4 lots" in the community title sentence also fires the dwellings trigger; the true count is 4 either way.
- Kept equal to the base because the checker allows only the trigger's keys to move: GFA 460 m2, site 700 m2,
  budget $1,650,000 and every fee. Two-bedroom townhouses keep the same GFA plausible; the budget and the
  architect's fee are light for 4 kitchens and 4 sets of services.

## Missed by design

As the base: `hdr.scale.avg_dwelling_size_sqm` (115) is stated only on the drawings d14/d15 (not read) and the
field has no triggers, so `read_by_design` is false; target start only in the d04 email; regulated design has no
reading; d02 and d07 identity in tables; d15 -> d14 supersession not written.

## Shape comments

- The app is expected not to suggest traffic or waste (shape gap: baselines[1] is non-residential only).
