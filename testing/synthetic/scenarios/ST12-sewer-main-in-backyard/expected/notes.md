# ST12-sewer-main-in-backyard notes

The base Whitfield granny flat, but a Sydney Water sewer main runs along the rear boundary, about 1.0 m inside it,
and the flat's rear edge sits within its zone of influence. The flat stays where it is: the rear edge goes on piers
below the zone of influence, with Sydney Water building plan approval and asset assessment before the CDC, and CCTV
of the main before and after. Only `flag.authority_asset` changes (no -> yes).

## What changed

- d03: the base negation ("No Sydney Water assets are within the property; the sewer main is in the street") is
  replaced by the evidence "The sewer diagram shows a Sydney Water main running along the rear boundary"; the house
  drain now runs to that main. The `sewer-main-in-street` trap is gone. `det.sewer_connection` stays reticulated.
- d05: Stage 2 scope adds the water authority asset submission (Tap in approval and asset assessment); fee unchanged.
- d06: adds "Authority asset assessment and pier design to Sydney Water's guidelines" at $650; total $1,600 -> $2,250.
- d11: recommends bored piers below the zone of influence along the rear edge.
- d12: the evidence "the flat sits within the zone of influence of the sewer main" and "building next to a Sydney
  Water asset".
- d13, d14: the site plan shows the main, its zone of influence and the piers instead of a front-boundary junction.
- Searched for every base mention of the sewer location ("in the street", "front boundary", "no Sydney Water
  assets"): only d03, d13 and d14 had one. d10's Tap in sentence holds for both. d01, d02, d04, d07-d09 and the
  invoices never mention the sewer.

## Traps

As the base, minus d03 `sewer-main-in-street`. The d03 `existing-building-retained` and trees traps, d07, d11, d12
and d16 traps are unchanged.

## Review applied (6 Nov)

Findings 1 and 2 (d03) apply here as in the base; findings 3 to 5 are da-pathway only. See ST12-base
notes.

## Missed by design

As the base: d04 target start, d02/d07 table numbers and dates, d14 -> d13 supersession, `proj.regulated_design`,
and possibly `hdr.scale.garage_spaces`.

## Notes for the scorer

- The trigger adds `substructure.piles`; d06, d11 and d12 state piers. Knowledge gap: `knowledge/works/system_terms.yaml`
  maps "Pier" and "Pier footings" to `substructure.footings`, not `substructure.piles`, so the app will likely
  return a footings item instead of a piles item. A missing piles item is that mapping, not the corpus.
- The authority-asset flag also makes the app's structural scale complex (shape `why`): over-scoping to report, not
  a key error.
