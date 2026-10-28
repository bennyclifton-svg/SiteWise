# Synthetic Test Corpus Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> to run this plan task by task. Live state is in
> `testing/synthetic/STATUS.md`. Read it before doing anything.

**Goal:** a ground-truth test regime for 20 common Australian project types.
For each type it measures whether SiteWise:

- files the documents correctly;
- builds the right profile;
- proposes the right consultants;
- reads fees and invoices correctly;
- produces a cost plan, program and RFP set of the right *shape* for that
  job.

**Architecture:** truth first, documents second. Each type gets an
owner-reviewed **domain map** (`DOMAIN.md` plus a machine-checked
`shape.yaml`). From that come scenarios: a base case, plus variants that
each flip one site condition. Each scenario has `expected/` answer keys and
a Markdown `corpus/` written from those keys. A harness drops each corpus
through the local API, wave by wave. It compares the app's register,
profile, procurement, invoices, cost plan, program and RFPs with the keys
and classifies every difference.

**Tech stack:** Python 3 with PyYAML and unittest (the repo's tool
convention), the local SiteWise dev server, and live Jev with every
exchange recorded. No new dependencies.

**Inputs:** `testing/synthetic/DISCOVERY.md` (what the app does) and
`testing/synthetic/PROJECT_TYPES.md` (the 20 locked types).

---

## How the work is run (context discipline)

- **One fresh subagent per type per stage.** Each works from a written
  brief in `testing/synthetic/briefs/`, writes files, and returns a summary
  of 150 words or fewer. The orchestrator never reads whole maps or corpora.
  It reads the summaries, the validator output and `STATUS.md`.
- **Every stage has a machine check and an independent reviewer.** The
  author never reviews their own work. A reviewer verdict of FIX sends the
  author brief back with `REVIEW.md` attached.
- **Owner gates** sit after the pilot domain maps, after the pilot
  scenarios and after the pilot run. Nothing scales to all 20 before its
  pilot gate passes.
- **Resumable.** `STATUS.md` holds the state of every task and type. A new
  agent starts from `testing/synthetic/README.md`.
- **Batches.** Run 3 types at a time in the pilot, and up to 6 in parallel
  after it.

## Provisional decisions (until the owner answers DISCOVERY.md §13)

These are defaults so the work can proceed. Each one is reversible, and
each is recorded in the keys.

| Q | Default |
|---|---|
| 2 | Fields the app does not model are keyed with `supported: false` and reported as coverage gaps, kept apart from accuracy. |
| 3 | An amber value counts as the app's answer. The band is reported next to it. |
| 4 | The key holds the true value. Outcomes are correct, conflict shown including the true value, or wrong. |
| 5 | Clean tier: title blocks as label lines. Messy tier: tables and fixed-width blocks. Adversarial tier: label lines with traps. |
| 7 | The expected profile is the truth from the whole corpus. Each value is tagged with its source document, so "not read by design" is reported separately. |
| 9 | Procurement suggestions are scored against the true profile *and* the app's own profile. The difference is a knock-on error. |
| 10 | NSW default. Other states only as variants. |
| 12, 13 | The kind and discipline mappings in DISCOVERY.md §4.2 and Q13. |
| 14 | Invoices go to the Invoices tab only. Fee proposals go mostly to the repository. |
Owner answers, 28 October 2026:

| Q | Decision |
|---|---|
| 1 | **Live Jev calls are approved.** The owner says they cost almost nothing. Keep recording every exchange anyway, so a run can be replayed. |
| 8 | **The harness may press profile Update** (`POST /projects/{id}/profile/read`) after each wave. |
| 16 | The owner left this to the agent. **Near-duplicates (different bytes, same number and revision) are left out of the pilot**, because they are rare in practice. Exact re-drops of the same bytes stay in. One near-duplicate case goes into a later stress scenario, to confirm the suspected stuck filing (DISCOVERY.md §2.4). |
| 17 | **Commit to `main` and push periodically.** Stage only `testing/synthetic/` and this plan, never other sessions' working-tree changes. Before each push, scan `git diff origin/main..main` for real firm, fee or personal data, because the repository is public. |

---

### Task 0: Scaffolding and shape validator (done 28 Oct)

**Files:** `testing/synthetic/{README,PROJECT_TYPES,STATUS}.md`, `briefs/00–02`,
`tools/validate_shape.py`, `tools/test_validate_shape.py`.

**Check:** `python -m unittest discover -s testing/synthetic/tools`
gives `OK` with 5 tests. The validator rejects any discipline, system,
action, fragment, skeleton, RFP template or profile key that is not in the
app's own files. It also rejects a built-in claim with no existing source,
and a trigger without evidence, negation or a consequence.

### Task 1: Knowledge index

**Brief:** `briefs/00-knowledge-index.md`, one subagent.
**Output:** `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and
`reference/index/*.md`. It documents how the app picks work items,
packages, cost lines, the program skeleton and fragments, and RFP stages
and positions. It covers every encoded trigger-to-consequence link and how
to run the generators without Jev.
**Done when:** every section is cited; at least 10 claims have been checked
against the Go; the subagent's summary is logged in STATUS.

### Task 2: Pilot domain maps (T01, T05, T09)

**Brief:** `briefs/01-domain-map.md`, one subagent per type, in parallel.
Then `briefs/02-review.md`, a different subagent per type.

**Steps per type:**
1. Dispatch the author with the placeholders filled from PROJECT_TYPES.md.
2. Run `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/<id>-<slug>/shape.yaml`.
   Expect `ok`.
3. Dispatch the reviewer. Read only the verdict line of its reply.
4. On FIX or REDO, re-dispatch the author with "apply REVIEW.md". Repeat
   from step 2. After two FIX rounds, escalate to the owner.
5. Update STATUS.

**Done when:** all three are validated with a PASS review. → **Owner gate
1:** the owner reads the three DOMAIN.md files and either approves the
template and depth or corrects them.

### Task 3: Remaining 17 domain maps

The same steps as Task 2, in batches of up to 6. Start with the types that
share knowledge with the pilot: T10, T11, T12 (from T01), then T02, T13,
T17 (Class 2), then the rest. Owner review is per type, as each one
finishes.

### Task 4: Program, cost-plan and RFP shape scoring rules

**Files:**
- Create `testing/synthetic/SCORING.md`.
- Create `tools/score_shapes.py`, with `tools/test_score_shapes.py` written
  first.

These rules compare an app output with a `shape.yaml`.

- **Cost plan:** each expected line is matched (by label alias or source
  element) or missed. A line with no match in the shape is extra. Any amount
  without a source document is a fabrication.
- **Program:** the same matching for activities, plus checks that `after`
  links hold in the app's computed dates, that no activity is orphaned and
  that there are no cycles.
- **RFPs:** each `must_include` is present, and each `must_not_include` is
  absent.
- **Rubric:** a 1–5 owner rubric per type for "practical, logical,
  concise", recorded in `expected/rubric.md`.

TDD: write fixtures from the T01 shape first.

### Task 5: Scenario brief, expected schemas and pilot scenarios

**Files:** `briefs/03-scenarios.md`, `tools/validate_expected.py` and its
test. Then `scenarios/S<type>-<variant>/` for T01, T05 and T09.

Each scenario:
- `manifest.json`: each document's filename, author role, date, wave, drop
  target, the fields it carries and the traps it contains.
- `expected/`: `document_register.json`, `profile.json`,
  `procurement.json`, `fees.json`, `invoices.json`, `rfis.json`
  (`supported: false`), `cost_plan.json`, `program.json`, `rfps.json` and
  `notes.md`.

The base scenario is written from `shape.yaml`. Each variant changes
exactly one trigger, and its keys differ from the base only where that
trigger's `adds` says. That is a directional metamorphic pair, checked by
code. Invariance pairs (renamed files, reordered drops, changed people's
names) are added for at least 5 types. → **Owner gate 2** after the pilot
scenarios.

### Task 6: Rendering brief and corpus lint

**Files:** `briefs/04-render.md`, `tools/lint_corpus.py` and its test.

The lint fails a corpus on:
- any answer-key leakage (an expected value or field name appearing as a
  label, or words like "test" or "expected");
- a real-looking ABN that passes the checksum without being declared in the
  manifest;
- a non-`example.com` email address;
- a lot/DP number not in the manifest's invented list.

The render subagent works one scenario at a time: 8 to 30 documents, waves
from the manifest, the tier rules from Q5. Lint, then an independent read
by a reviewer subagent.

### Task 7: Harness

**Files:** `testing/synthetic/harness/` (Python, standard library HTTP, as
in `tools/corpus-upload.py`), with tests against a fake server.

1. Create a QA project.
2. For each wave, POST each file to its manifest drop target, then wait
   until filing and background jobs are idle (SSE or polling).
3. Optionally call profile Update (Q8).
4. Snapshot the documents, profile, procurement board, submissions,
   invoices, cost plan, program and RFP drafts.
5. Score, using `score_shapes.py` and the field comparisons: wrong, missed,
   hallucinated, wrong conflict.
6. Write `report.json` and `REPORT.md`, broken down per field, type, tier
   and variant, with confusion matrices for kind and discipline.
7. Record every Jev exchange so the run can be replayed.

Never correct or delete user data, and only run against `127.0.0.1`.

### Task 8: Pilot run

Needs the owner's yes on Q1. Run the T01, T05 and T09 scenarios. Write
`FINDINGS.md` entries with scenario, field, expected, actual, suspected
cause and code location. Never edit a key to match the app; a key changes
only when it was wrong, with the reason in that scenario's `notes.md`. →
**Owner gate 3.**

### Task 9: Scale to 20

Tasks 5 to 8 for the remaining types, in batches. Re-run all scenarios after
each app change, as a regression set.
