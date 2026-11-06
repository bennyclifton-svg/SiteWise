# ST17-project-remediate notes

The base Tideway Apartments job, but the building is enrolled in NSW Project Remediate: the owners corporation has
accepted an interest-free loan, and the program's appointed project manager coordinates the design team and builder
from the program panels. No profile value differs from the base (the trigger's profile list is now empty).

## Changes from the base

- d02 brief: the trigger's two evidence phrases replace "funding the works through special levies without government
  assistance"; the brief invites only the design team (title changed to "... design team").
- d05 email (not read): the general meeting passes the resolution to sign the remediation and loan agreements,
  formalising the loan the committee accepted, instead of raising levies.
- d13 options report: funding by the Project Remediate loan; the program's project manager administers the contract.
- d06 project manager fee proposal removed: the program engages and pays the PM (trigger removes
  `consultant.project_manager`). 19 documents.
- `fact.contract_form` now rests on d13 alone (d06 carried it); `proj.client` loses d06 as a source.

## What the scorer should see

Shape trigger `project-remediate`: no PM package, PM fee line or PM RFP; program adds eligibility, loan agreement
resolution, program-appointed PM mobilisation and panel selection; cost plan adds loan establishment and repayment
levies (no amount). The app never suggests a PM for remediation; with d06 removed no proposal creates the PM row,
so the board matches the trigger's removal. The program and cost additions are expected to be missed.

## Missed by design

As the base: `proj.target_construction_start` (d05 email), `proj.regulated_design` (no extraction), `proj.lot` (no
range form), `proj.client` cut at " - ", the table title blocks of d03, d11 and d20, and d18 → d17 supersession.
det.type_of_construction (A): derived only, never read; blank while rule.ncc.type-of-construction and its table are
draft (reason unreviewed). The key keeps the true value.

## Traps

As the base, less d06 `fee-stages-not-staged-works` (removed with the PM proposal); d07 `stage-5-fee-stage` is now
the only fee-stage hit.
