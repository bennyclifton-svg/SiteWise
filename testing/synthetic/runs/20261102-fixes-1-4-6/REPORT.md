# Re-run 20261102-fixes-1-4-6: after the owner's fixes for F1, F4 and F6

Same three T01 scenarios and harness as `../20261031-pilot`. The app was
built from the working tree on 2 November 2026 (the fixes are uncommitted).

| Finding | Result |
|---|---|
| F1 Update response cut off | Better: 3 of 12 waves (was 8 of 12). Now only wave 1 of each project, the first Update on a new project. Suspect the first-view path in `writeProfileQueued` (`EnsureWholePart`, then a second read). |
| F4 procurement route not read | Not fixed in this build: `hdr.cond.procurement_route` is still blank with no sources, in all three. Jev is asked (server log: confidence 0.54 to 1.0 across 373 asks) but nothing is applied, so it most likely answers "not stated" for the passages it sees. The program still opens on `sk.generic`. |
| F6 BASIX discipline | Fixed: d09 files as `consultant.basix` in all three. The BASIX package and its cost line now appear (procurement +1, cost plan +1 in each). |

Other changes from the first run, probably run-to-run variation in Jev:
`proj.client` became a conflict (base); `hdr.cond.environmental_sensitivity`
missed (CDC) and a conflict (trees); `flag.authority_asset` a conflict
(trees); the email d04's discipline is now right (CDC). `proj.lga` is now
the single wrong value "City of Ryde Section" (F2, still open).

## RFP triage (2 November)

See `RFP_TRIAGE.md`. After the scorer fixes and the shape corrections, RFP
new failures are base 34, CDC 32, trees 27, and awaiting user input is 50,
46 and 50. Every base failure that is left is a knock-on of a blank profile
value; F4 alone accounts for 14. New app findings: F9 to F12.
