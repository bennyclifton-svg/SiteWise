# Synthetic test corpus

This folder holds fictional project documents for 20 common Australian
project types, each with known correct answers. They test whether SiteWise:

- files the documents correctly;
- builds the right profile;
- proposes the right consultants;
- reads fees and invoices correctly;
- produces a cost plan, program and RFPs of the right shape for the job.

Application code is never changed here. Every person, firm, address,
lot/DP number and ABN is invented, and emails use `example.com`.

## Resuming this work (any agent)

1. Read `STATUS.md`. It shows what is done, in progress and waiting for the
   owner.
2. Read the plan: `docs/plans/2026-10-28-synthetic-test-corpus.md`. It sets
   the task order, the gates and the provisional decisions.
3. Read `DISCOVERY.md` when you need to know what the app does, and
   `PROJECT_TYPES.md` for the 20 locked types.
4. Work through subagents, using the briefs in `briefs/`. Give each
   subagent one type and one stage. Read only their short replies and the
   validator output. Update `STATUS.md` after every task.

## Layout

| Path | What it is |
|---|---|
| `DISCOVERY.md` | How the app files, reads and reconciles; the owner's open questions |
| `PROJECT_TYPES.md` | The 20 locked project types and the pilot |
| `STATUS.md` | Live tracker |
| `briefs/` | Self-contained subagent briefs: 00 knowledge index, 01 domain map, 02 review (03 and 04 come later) |
| `reference/` | The shared knowledge index (Task 1) |
| `types/<id>-<slug>/` | One project type: `DOMAIN.md`, `shape.yaml`, `REVIEW.md` |
| `scenarios/` | Base and variant scenarios: `manifest.json`, `expected/`, `corpus/` (Task 5 on) |
| `tools/` | Validators and lint, with unittest tests |
| `harness/` | Upload, snapshot and score through the local API (Task 7) |

## Checks

```
python -m unittest discover -s testing/synthetic/tools
python testing/synthetic/tools/validate_shape.py testing/synthetic/types/*/shape.yaml
```
