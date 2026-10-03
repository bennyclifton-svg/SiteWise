# Brief extraction accuracy

Lane A: make an uploaded brief populate a trustworthy project profile. Scope is
source structure, routing, count candidates and evidence interpretation. No new
service or dependency. Filing remains p50 <= 1,000ms / p90 <= 2,000ms; profile
reads and edits remain 50ms / 150ms. Background evidence work stays off these
interactive paths.

The Hale review found missed classification because split PDF headings hid the
description from routing. Door dimensions and tenancy identifiers became counts;
the natural-gas heading was separated from its exclusion. The enum value `none`
was also discarded as though it were always a candidate-extraction sentinel.

## Decisions

- Source version 3 recognises split numbered headings without altering original
  bytes or source offsets, rejects quantity/code cells as headings, and keeps a
  heading attached to its short answer. Area-table labels and values stay together.
- Project descriptions can route header questions beyond the initial passages.
  An individual use-list bullet cannot classify the whole building. General power
  specifications are not project descriptions. Class remains the product category;
  an NCC class still requires explicit evidence.
- Counts need count-shaped candidates. Door dimensions, decimal section numbers
  and tenancy identifiers are not offered as counts. Jev selects among candidates;
  it does not count or convert measurements.
- Profile version 3 reads the bounded clause with its section/list context.
  Presence questions receive system definitions and boundaries. Literal service
  terms can route speculative evidence questions even if a family label is weak;
  only Jev decides inclusion. No additional serial judgement stage is introduced.
- A valid enumerated `none` survives reading and reconciliation. Candidate `none`
  and `not_stated` remain silence. Unsupported choices are rejected.
- Explicit sprinkler installation requirements are displayed as required, not
  existing protection. Required/allowance values remain excluded from derivation.
- Forklift charging is not road-vehicle EV charging. A recessed-dock exclusion
  does not exclude on-grade loading bays. Scoped exclusions remain inspectable in
  source records. Conflicting rainwater requirements remain a visible conflict.
- Known scale facts remain visible while building type is unresolved or its default
  fields use another area basis (GLA is not silently relabelled as GFA). User edits
  remain authoritative; extraction upgrades occur on explicit profile update.

These follow TypeSafe's [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[fan-out](https://docs.typesafe.ai/patterns/fan-out), and
[literal questions and numeric limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
The existing provisional floors are unchanged; green remains withheld. The small
regression suite is not [confidence calibration](https://docs.typesafe.ai/confidence).

## Verification

Deterministic tests cover split headings, source-offset round trips, area-table
context, detached exclusions, late descriptions, ancillary office bullets,
dimension/identifier/section-number rejection, genuine counts, enumerated `none`,
required sprinklers and visibility of scale facts before classification.

The private Hale cases exercise the actual uploaded PDF through the production
extractor before replaying recorded Jev answers. The existing Bankstown suite is
also retained. Source PDFs, excerpts and recordings stay private. The draft Hale
answer key remains unreviewed; only the owner can approve it.

Final checks on the development host:

- Hale: 12/12 expected readings automatically applied; no forbidden values applied.
  This includes actual-PDF cases for principal classification, GLA, gas exclusion,
  required sprinklers, MHE ventilation, rainwater irrigation and loading bays, plus
  negative checks for door counts, tenancy identifiers, EV charging, ancillary
  office classification and recessed-dock overgeneralisation.
- Existing Bankstown/control suite: 15/15 expected readings automatically applied;
  no forbidden values applied. Both suites also pass deterministic replay.
- All Go packages passed (`go test -p 1 ./...`); web build and all three profile
  browser scenarios passed. The Windows browser-test server needed cleanup after
  assertions completed. Final profile API changes passed the profile tests again.
- Profile reads: p50 2.78ms / p90 4.09ms. Edits: 6.39ms / 6.74ms, against 50/150ms.
  These are local HTTP tests with the real store and code-only reconciliation.
- Knowledge checker: zero errors/warnings. No dependency added.
- The full intake replay benchmark could not complete: 36 requests did not match
  existing recordings. It correctly failed closed. This is not a passing filing
  benchmark or production timing evidence; the intake recordings need refreshing
  before that broader gate can be claimed. Intake code was not changed by this fix.

Hale was reprocessed with the owner's approval. Its 21 pages and 238 reading units
remain preserved. The saved profile has Industrial/Warehouse/Extension, live
operations, GLA 2,135 m2, no gas, mains sewer and required sprinklers. Invalid door
and tenancy counts and EV charging readings are absent. On-grade loading facilities
are included; the recessed-dock exclusion remains a source record for review. The
rainwater conflict remains unresolved rather than silently choosing a side. The
NCC classification remains unstated. Machine readings remain provisional amber.
