# WP-K1 / WP-26 tenant-fitout capacity case

Follow-up: the interface applicability limitation identified here is fixed and
tested in `docs/evidence/2026-10-06-interface-applicability.md`. Owner/project
acceptance remains open. The historical measurements below describe this change
before that follow-up.

Date: 6 October 2026. Draft knowledge correction; acceptance remains partial.

Lane A. Outcome: a tenancy alteration or upgrade raises reviewable capacity
investigations for the existing base-building mechanical supply. This implements
the D-29 planning recommendation and reduces misleading structural-load advice.
Budgets remain profile edit p50/p90 50/150 ms, rebuild 100/300 ms and filing
1000/2000 ms. No dependency, service, Jev question or automatic work is added.

The existing `if.tenant-fitout-base-building-hvac` edge is now `supplies`, from
central plant and ventilation to tenancy air distribution and air conditioning.
Its cited mechanical seed's “Existing buildings and live environments” section
requires investigating existing capacity, condition and controls before adding
load. Existing questions, source references and draft status are preserved.
The existing supplies consequence now handles `alter` and `upgrade`; the loads
consequence previously missed alteration and described structure or ground.

The synthetic tenant fixture checks both supply targets, trigger and part
provenance, unknown existence, and suppression for retained/repaired/excluded
work, absent systems and replaced plant. It failed before the edge correction.
It also exposed two comma-containing flow-mapping labels that YAML silently
truncated. Quoting those labels preserves the complete capacity and continuity
wording. The knowledge checker now rejects unexpected proposal mapping keys,
with a regression test for this exact malformed YAML case.

Focused validation passed: strict knowledge checker (zero errors, two existing
unmapped package-field warnings), 32 checker tests, knowledge and works suites.

This does not close AT15, K1, K6 or M1: a synthetic direction/action/existence
case is not owner-reviewed project evidence. Interface `applies_when` is not
currently loaded or evaluated by the interface proposal engine; this fixture
does not establish determinant applicability. That existing limitation remains
an explicit follow-up. Knowledge remains draft and automatic proposal generation
remains gated.

## Full validation and timing

`go test -p 1 ./...` passed with the local PostgreSQL test database, including
store (29.203 s), HTTP API (13.570 s), knowledge (3.549 s) and works (1.940 s).
The K4 hash inventory was regenerated; `git diff --check` passed.

The 20-file/two-round/40-sample integrated diagnostic is saved at
`bench/results/2026-10-06-tenant-fitout-capacity.json`:

| Path | p50/p90 ms | Budget ms | Result |
| --- | --- | --- | --- |
| Filing | 473.768 / 638.405 | 1000 / 2000 | Pass |
| Spec Home edit | 50.095 / 52.801 | 50 / 150 | Fail |
| Integrated rebuild diagnostic | 88.555 / 94.441 | 100 / 300 | Pass |
| Proposal read | 4.377 / 5.677 | 100 / 250 | Pass |

The combined gate fails; the edit median exceeds its unchanged budget.
Identity extraction and deterministic-field substage overruns are also reported
in the artifact for target-VPS assessment. This is local replay evidence, not
a live-provider or release result. The earlier intermittent large profile
delays remain unresolved; this run does not prove they are fixed.
