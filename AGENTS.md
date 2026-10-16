# Agent instructions

SiteWise is a construction-project app built from first principles on the
physical building, with **Jev (TypeSafe System One) as the only AI** and
**speed as the first objective**. Read `PRODUCT.md` for the current surfaces.
Read `docs/design/2026-09-29-foundation-design.md` for the binding rules:
speed, Jev only, the building as the model, and the intake budget.

## Non-negotiables

1. **Jev only.** No LLMs, embeddings, chat agents or other AI providers at
   runtime. Jev picks; code parses, computes, looks up and controls flow.
2. **Follow TypeSafe's docs** for every Jev use and cite the page:
   [patterns](https://docs.typesafe.ai/patterns),
   [how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one),
   [confidence](https://docs.typesafe.ai/confidence),
   [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
   [API](https://docs.typesafe.ai/api). Full index: https://docs.typesafe.ai/llms.txt
3. **Speed has budgets.** Every user-facing path has a p50/p90 budget and a
   benchmark that fails the build when it is broken. One Jev fan-out per
   state; never serial round trips on a hot path; nothing judged while a user
   waits if it can be precomputed.
4. **The building is the model.** Knowledge is organised as systems,
   determinants, rules, interfaces and failure modes (`knowledge/SCHEMA.md`).
   Consultant disciplines are a view over it, not its structure.
5. **Rule numbers come from the instrument itself** (NCC 2022, the standard),
   never from seed prose alone. Unverified numbers stay `verified: false`.

## Current product

`PRODUCT.md` names the workbench. The foundation design is the origin of the
rules above and describes the intake slice. The workbench also includes
Profile, Cost Plan, Procurement and Specifications.
Project Plan and Program are navigation placeholders. Delivery records and
implication proposals stay on the project and are off the left nav.

When a task does not name a budget, use the budget already recorded for that
path in `bench/budgets.json`. Filing stays p50 ≤ 1 s and p90 ≤ 2 s. Profile
edit stays at its existing tighter budget. Cost, procurement and specification
reads and writes stay at p50 ≤ 100 ms and p90 ≤ 250 ms unless that named path
is tighter.

These paths do not call Jev: cost reads and writes, procurement row create,
report create, and specification direct edit, review, schedule and issue.
Specification import is a background job in the existing worker: one fan-out
per candidate, cached by source. A reading is a proposal. Navigation and
typing do not call Jev. An uploaded invoice is read with one interactive
fan-out for the fields rules leave open; invoice edits, splits, the ledger
read and export do not call Jev. A proposal dropped on a procurement row is
read with one fan-out after filing; the price table, cell corrections,
award, undo, ratings and the comparison report do not call Jev. Award writes
contract values only from the firm's priced cells; it never writes a budget.

Money is computed in Go. Unknown stays unknown. Do not invent a rate, a price,
a rule number or a product selection. An upload leaves an accepted
specification value and a frozen cost baseline as they were. Removing a
procurement package may delete linked draft cost lines only through the
existing confirmation, and leaves frozen baselines and issued history intact.

Before changing one of these surfaces, read its current note:

- Profile: `docs/design/2026-10-03-project-profile.md`
- Cost plan: `docs/plans/2026-10-10-cost-plan-grid.md`,
  `docs/plans/2026-10-10-cost-plan-default-inclusion.md`,
  `docs/plans/2026-10-11-cost-plan-direct-editing.md`
- Procurement: `docs/plans/2026-10-10-procurement-table.md`,
  `docs/plans/2026-10-11-procurement-cost-removal.md`,
  `docs/plans/2026-10-11-procurement-default-disciplines.md`.
  That plan's header still says nothing is built; the knowledge file, the
  loader and the browser test are in the tree. Trust the code.
- Specifications: `docs/design/2026-10-14-specification-workspace.md`,
  `docs/plans/2026-10-14-coordinated-specification-workspace.md`,
  `docs/evidence/2026-10-14-specification-implementation.md`
- Procurement to award: `docs/plans/2026-10-16-procurement-process-to-award`,
  `docs/plans/2026-10-16-procurement-to-award-implementation.md`,
  `docs/evidence/2026-10-16-procurement-to-award.md`
- Invoices (Cost Plan panel): `docs/plans/2026-10-15-invoice-intake.md`,
  `docs/plans/2026-10-16-invoice-intake-implementation.md`,
  `docs/evidence/2026-10-16-invoice-intake.md`

## Stack

Go single binary (API, workers, embedded Vite + React SPA, SSE), PostgreSQL 17
on the same host (full-text search, no pgvector), files on local disk by
content hash, Caddy + systemd. Stack changes need a stated reason.

## Reference repo

`../clerk` is frozen reference. Copy data from it (`data/seed/`,
`data/taxonomy/`, answer keys, corpora), never code. Ignore its PM doctrine
(`docs/clerk-brief.md`) and its Jev plan's process rules; this repo's design
supersedes them.

## Knowledge work

- Run `python tools/check_knowledge.py` after editing `knowledge/`; it must pass.
- Everything written from seeds is `status: draft`. Only the owner marks
  `reviewed`.
- Follow the question authoring rules at the end of `knowledge/SCHEMA.md`.

## Style

Small obvious functions, validation at boundaries, no speculative
abstraction, comments explain why. Dependencies only when the alternative is
non-trivial; justify each in the commit message.

## Starting a feature

The why, the three lanes, and the prompt run sheets live in
`docs/developing.md`. Follow them.

A change is one lane:

- **A, product core** (filing, org isolation, rule numbers, cost figures,
  procurement packages, issued specifications, accepted specification values,
  anything a user trusts). An existing test or answer key must still pass, and
  the speed budget must still pass. Unverified rule numbers stay
  `verified: false`.
- **B, edge** (a label, an empty state, a layout). Move fast. Do not break
  filing.
- **C, security and dependencies** (login, file access, anything new in the
  build). Handle a missing file, a wrong org, and unexpected input. Prefer
  a small dependency many people already watch over one this repo would be
  the only watcher of.

Before coding a new feature, the task states the user outcome, why it
serves filing speed, answer trust, or time-to-decision, the lane, what is
out of scope, the p50/p90 budget in milliseconds, what "done" looks like,
and the edge cases. When those are missing, apply these defaults and do
not invent a wider task: use the budget already recorded for the path in
`bench/budgets.json`; filing stays p50 ≤ 1 s and p90 ≤ 2 s; one Jev
fan-out per state; every row scoped to `org_id`; no new service or dependency
without a stated reason.

A clear defect is fixed only as far as that defect. A direction choice
(what the next screen or workflow should be) is decided in prose before
code. After implementing, report timing against the budget, what changed
in plain language (what calls what, what could fail), and any new
dependency. Do not widen the feature.
