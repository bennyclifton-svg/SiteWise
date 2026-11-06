# ST13-contaminated-land notes

The base project on contaminated land: the rear part of No. 40 (now the grassed yard behind the dwelling) was a service station until 1992 (underground tanks
may remain) and a dry cleaner operated in the shop at No. 42. The PSI recommends a detailed site investigation; a
remediation action plan, validation and a site audit statement follow. 19 documents, 4 waves; 3 changed, 1 added
(d19, the environmental consultant's DSI, RAP and validation fee proposal, $74,500 ex GST).

## Changed key

`det.potentially_contaminated_land` false -> true (d02, d05, d13, d19). The `contaminat` trigger fires in all four.

## What a correct app should do, and what this app will do

- Procurement: the key (shape plus trigger) expects `consultant.environmental`. The knowledge index says its package
  rule (`package_defaults.yaml complexity_additions[2]`) can never fire, so the app will **not suggest** it from the
  profile. The base shape already lists `consultant.environmental` (need: usually), and the app misses it in the base. In
  this variant it is matched only as a package opened by the d19 proposal (`state: package`), never as a suggestion.
  The procurement change between base and variant comes from d19, not from `det.potentially_contaminated_land`.
- RFPs: `flag.contamination` follows the determinant, so the civil RFP's earthworks suitability position should turn on.
- Program: the DSI, remediation and validation, and site audit statement rows are only partly built in; the cost lines
  for the DSI, RAP, auditor, remediation and validation are not built in (trigger `built_in`).

## Other mentions checked

d01, d03, d04, d06 to d12 and the invoices say nothing about contamination, so they are unchanged. The geotechnical
report (d11) only says spoil is classified before disposal. No variant document uses "groundwater", "dewater" or
"water table", so `flag.groundwater` stays no (d11). Traps and missed-by-design values are as the base.
