# Conventions

Choices the orchestrator made so that maps agree with each other. These
are not owner rulings (see `OWNER_RULINGS.md`, which overrides this file).
Each one marked *provisional* waits for the owner to confirm it.

## C1. Novation of services engineers on D&C (provisional)

On a design and construct job with novation (T02, T13, and any later D&C
type), the architect and the structural engineer are novated: `yes`. The
services engineers (mechanical, electrical, hydraulic, fire services) are
`option`, because builders often replace them with their own D&C services
trades. Owner question: is that the usual practice?

## C2. BASIX assessor versus ESD consultant

- Residential BASIX and NatHERS work: `consultant.basix`.
- NCC Section J, Green Star or NABERS work on a non-residential part:
  `consultant.esd`.
- A mixed building that needs both may carry both. Where one firm does both,
  use `consultant.esd`, and add `consultant.basix` to `aliases`.

## C3. RFPs that share a template

`consultant.bca` and `consultant.certifier` both use the
`bca-certification` template. The scorer pairs an expected RFP with an app
report by discipline first, and by template only when no discipline
matches.

## C4. Facts a reviewer checked against a source

Maps may cite these as checked. They are still not owner rulings.

- **Long service levy (NSW):** 0.25% of the cost of building work of
  $250,000 or more, including GST, paid before work starts. Source:
  nsw.gov.au, cited in T08's REVIEW.md.
- **WHS Regulation:** the NSW WHS Regulation 2025 has replaced the 2017
  Regulation. Clause numbers stay `verified: false` until checked against
  the 2025 text (T08 review).
- **Fire control centre (NCC 2022 E1D15):** required in a building with an
  effective height of more than 25 m, or a Class 6, 7, 8 or 9 building
  with a total floor area of more than 18,000 m². The separate fire
  control room (S19C7) applies above 50 m. Source: NCC 2022 Volume One,
  Part E1, https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/e-services-and-equipment/part-e1-fire-fighting-equipment
  (checked 30 October 2026). Use `verified: true`.

## C5. Staged construction certificates

Where a job has an early-works CC and a main-works (building) CC, both sit
in the construction_documentation stage. The app's CC fragment row
corresponds to the early-works CC. Match it there, and record the
building CC as built_in no (T04 review).
