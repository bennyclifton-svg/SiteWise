# Agent instructions

SiteWise is a construction-project app built from first principles on the
physical building, with **Jev (TypeSafe System One) as the only AI** and
**speed as the first objective**. Read `docs/design/2026-09-29-foundation-design.md`
before changing anything.

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
