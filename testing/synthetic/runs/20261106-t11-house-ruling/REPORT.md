# Run 20261106-t11-house-ruling: T11 after the owner's house ruling

Owner ruling, 6 November 2026: a dual occupancy is a house, because there is
no strata scheme. Townhouses are a registered strata or community scheme with
shared common property. The T11 base and CDC keys now expect `house`; the
four-townhouse variant (community title, association property) keeps
`townhouses`. The subclass criteria carry the rule (`profile-13`).

Live Jev, ST11-base then ST11-cdc-low-rise-code, one harness process. Compared
with `20261106-t10-t12-after-fixes` (scored against the old key there).

| Scenario | Subclass | Profile correct | Program matched | Cost plan | Procurement | RFP new failures |
|---|---|---|---|---|---|---|
| T11 base | conflict → house | 51 → 52 | 53 → 57 /107 | 25/44 (same) | 8/12 (same) | 2 (same) |
| T11 CDC | conflict → house | 52 → 53 | 51 → 55 /106 | 24/45 (same) | 7/11 (same) | 6 (same) |

Unchanged: links violated 0, no fabricated amounts, invoices and fees right.

Still missing (expected by the key, which describes the job, not the
subclass): "Stairs, ramps, handrails and nosings" and "Compliant balustrades
and barriers". The house default list has neither, so a two-storey house or
dual occupancy only gets them when the documents or the user add them. The
separating wall's acoustic and fire lines still come through the party-wall
signal. T11 CDC `det.existing_building` stays blank (below the floor), as
before.
