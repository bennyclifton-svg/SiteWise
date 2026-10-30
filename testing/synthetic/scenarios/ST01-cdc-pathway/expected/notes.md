# ST01-cdc-pathway notes

The base Nguyen house approved as complying development under the Housing Code by a registered certifier. Only
`hdr.cond.planning` differs from the base (da -> cdc).

## Changes from the base

- d01 brief and d11 design statement carry the trigger's evidence phrases in place of the two DA negations.
- d05 architect's Stage 2 becomes CDC documentation and lodgement (same $22,000); `flag.architect_lodges` stays yes.
- d14 statement of environmental effects removed (16 documents).
- d02's "complying development may be carried out on the land" trap stays; here it agrees with the pathway.

## What the scorer should see

Shape trigger `cdc-pathway`: town planner removed from procurement and RFPs; DA lodgement, notification,
determination and CC rows removed; CDC rows added; DA and CC fee lines replaced by the CDC fee and contributions.
The app is expected to keep suggesting the town planner and to block the certifier and town-planning RFPs on cdc
(shape gaps).

## Missed by design

As the base: `proj.target_construction_start` only in the d04 email; d02 register number (PC-2026-01187) and
date (2026-02-10) and d07's number and date are in Markdown tables, so the app is expected to miss them.
