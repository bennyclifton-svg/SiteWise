# Status

The orchestrator updates this file after every task. Anyone resuming the
work starts here, then reads `README.md`.

Cell values: `—` not started, `run` in progress, `ok` done, `fix` needs
rework (see that type's REVIEW.md), `gate` waiting for the owner.

## Shared tasks

| Task | State | Output | Note |
|---|---|---|---|
| 0 Scaffolding and shape validator | ok | `tools/validate_shape.py`, briefs, this file | 5 unit tests pass |
| 1 Knowledge index | run | `reference/KNOWLEDGE_INDEX.md` | brief `briefs/00-knowledge-index.md` |
| 5 Scenario brief and expected JSON schema | — | `briefs/03-scenarios.md`, `tools/validate_expected.py` | after the pilot maps are approved |
| 6 Rendering brief and leak lint | — | `briefs/04-render.md`, `tools/lint_corpus.py` | |
| 7 Harness | — | `harness/` | Python, local API |
| 8 Pilot run | — | `REPORT.md`, `FINDINGS.md` | needs owner approval for live Jev |

## Project types

| ID | Type | Map | Validate | Review | Owner | Scenarios | Docs | Run |
|---|---|---|---|---|---|---|---|---|
| T01 | New detached house (pilot) | — | — | — | — | — | — | — |
| T02 | Apartments | — | — | — | — | — | — | — |
| T03 | Office fit-out | — | — | — | — | — | — | — |
| T04 | Warehouse | — | — | — | — | — | — | — |
| T05 | House alterations and additions (pilot) | — | — | — | — | — | — | — |
| T06 | Sprinkler pump and tank upgrade | — | — | — | — | — | — | — |
| T07 | Mechanical plant upgrade | — | — | — | — | — | — | — |
| T08 | Roof remediation | — | — | — | — | — | — | — |
| T09 | Waterproofing remediation (pilot) | — | — | — | — | — | — | — |
| T10 | Knock-down rebuild | — | — | — | — | — | — | — |
| T11 | Dual occupancy or townhouses | — | — | — | — | — | — | — |
| T12 | Granny flat | — | — | — | — | — | — | — |
| T13 | Mixed-use shop-top | — | — | — | — | — | — | — |
| T14 | Retail or café fit-out | — | — | — | — | — | — | — |
| T15 | Childcare centre | — | — | — | — | — | — | — |
| T16 | Office base-building refurbishment | — | — | — | — | — | — | — |
| T17 | Cladding remediation | — | — | — | — | — | — | — |
| T18 | Warehouse extension or mezzanine | — | — | — | — | — | — | — |
| T19 | Lift and access upgrade | — | — | — | — | — | — | — |
| T20 | School classroom block | — | — | — | — | — | — | — |

## Log

- 2026-10-28: Project types locked (PROJECT_TYPES.md). Plan written
  (`docs/plans/2026-10-28-synthetic-test-corpus.md`). Validator built and
  tested.
- 2026-10-28: The owner answered four questions. Live Jev calls are
  approved. The harness may press profile Update. Near-duplicates are out
  of the pilot. Commits go to `main` and are pushed periodically, staging
  only `testing/synthetic/` and the plan.
