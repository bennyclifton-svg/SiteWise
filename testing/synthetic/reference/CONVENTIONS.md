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
