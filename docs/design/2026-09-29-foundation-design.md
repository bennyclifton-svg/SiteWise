# SiteWise foundation design

Date: 29 September 2026. Owner: Benny Clifton. Status: agreed in design session.

SiteWise is rebuilt from the ground up to replace the current sitewise.au app,
which nobody uses and which is too slow. Two things drive every decision:

1. **Speed is the first objective.** Every gateway and component has a latency
   budget, and the build fails when a budget is broken.
2. **Jev is the only AI.** TypeSafe's System One model (Jev) makes every
   judgement. There are no LLMs, embeddings or chat agents anywhere in the
   product. Code does parsing, maths, lookups and control flow; Jev picks.

The previous repo (`clerk`) is frozen reference material. Copy **data** from it
(seed knowledge, taxonomies, answer keys, test corpora), never code.

## 1. Foundation: TypeSafe's "AI-powered software"

TypeSafe describes three ways to build: traditional software, LLM agents and
AI-powered software. SiteWise is **AI-powered software**:

> "Code handles deterministic work and owns the control flow. The model appears
> only where the system needs programmable common sense or needs to interpret
> unstructured data." —
> [How to build with System One](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)

There is no agent loop and no chat runtime. Every Jev use follows the TypeSafe
docs:

| Doc | Binding use |
|---|---|
| [Fan-out](https://docs.typesafe.ai/patterns/fan-out) | All questions about one state go in one request, including speculative ones. Never serial round trips. |
| [Confidence routing](https://docs.typesafe.ai/patterns/confidence-routing) | Each question has its own threshold, set by the cost of a wrong answer. |
| [Confidence](https://docs.typesafe.ai/confidence) | Noul answers have probability only, no confidence. Choice confidence depends on option count, so thresholds never transfer between questions. |
| [Composite scoring](https://docs.typesafe.ai/patterns/composite-scoring) | Jev scores atomic criteria; weights live in code. |
| [Intent routing](https://docs.typesafe.ai/patterns/intent-routing) | Code fast paths first (for example a document number goes to a direct lookup). Jev only for real ambiguity. |
| [Pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook) | Code over-finds candidates; Jev picks one or "none"; the value is copied verbatim. |
| [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13) | No counting, arithmetic, date comparison or generation by Jev. Only relevant state. Literal, explicit questions. |
| [API](https://docs.typesafe.ai/api) | `POST https://api.typesafe.ai/v1/systemone`. Pin an exact model version, never `jev-latest`. |

## 2. Domain model: the building, from first principles

The seed knowledge (`clerk/data/seed/`) is organised by consultant. SiteWise
reorganises it around the physical building in five layers:

1. **Systems**: one hierarchy of what physically exists (site and ground,
   substructure, structure, envelope, interiors, passive and active fire,
   mechanical, hydraulic, electrical, comms and security, vertical transport,
   access and egress).
2. **Determinants**: the few facts that decide which rules apply (NCC class,
   rise in storeys, effective height, sprinklered, floor area, state, climate
   zone, BAL, wind class, site class, flood, heritage, new or existing, use).
3. **Rules**: NCC provisions, Australian Standards and state instruments. Many
   are table lookups over determinants (class + rise gives Type of
   Construction, which gives FRLs).
4. **Interfaces**: typed edges between systems (penetrates, sequences, loads,
   supplies, controls, depends-on, shares-space, boundary). Most building
   failures happen here.
5. **Failure modes**: the known pitfalls, attached to rules and interfaces.

Consultant disciplines become a *view* over systems and interfaces, not the
structure.

**Jev reads the evidence; code does the physics.**

| Job | Who |
|---|---|
| Which systems a passage is about | Jev: noul per top-level system + choice of leaf, one fan-out |
| What documents state for each determinant | Jev picks among code-parsed candidates, or "not stated" |
| Type of Construction, FRLs, compartment limits, applicable standards | Code table lookups |
| Which interfaces this building has | Code graph walk over present systems |
| Whether a document addresses a rule or interface; whether a failure mode is present | Jev noul, run only on passages labelled with that system |
| Conflicts between documents; system and interface status | Code |

Knowledge is compiled once into `knowledge/` (see `knowledge/SCHEMA.md`),
drafted from the seeds at build time and reviewed. Rule tables are verified
against the NCC 2022 text itself, not the seeds, because the seeds contain
contradictory numbers and NCC 2019 clause numbering.

## 3. v1 slice: instant intake

Users drop PDFs (with a text layer), DOCX and XLSX into a chosen project. Each
file is filed within about 1 s.

**Budget (from upload complete):**

| Step | p50 / p90 |
|---|---|
| Read identity pages only (pdfium / Go DOCX/XLSX) | 80 / 250 ms |
| Harvest candidates (filename + text) | 5 / 10 ms |
| Rules settle what they can; those fields leave the Jev call | 1 ms |
| One Jev fan-out | 350 / 800 ms |
| Commit + SSE push | 5 / 15 ms |
| **Total** | **~450 ms / ~1.1 s** (gate: p50 ≤ 1 s, p90 ≤ 2 s) |

**Filing fan-out (one request):** kind (16 kinds), discipline (57, always asked
speculatively), lifecycle area, number / revision / title (choices among
harvested candidates plus "none"), supersedes (choice among same-number
documents, plus "new"). Code orders revisions and dates. State is an object
(`filename`, `title_block`, `first_page`, `headings`, `candidates`), and each
question names the field it reads.

**Confidence to colour:** green (above that question's threshold), amber
(applied, flagged), blank (below band), grey (Jev timed out; rule values only,
re-checked in the background). Supersedes has the strictest threshold. User
values are never overwritten; overrides are kept as labelled examples.

**Background after filing:** full text, passages, Postgres full-text index,
Jev system labels, then the knowledge questions for those systems. Foreground
filing always has priority and reserved Jev capacity.

**Guardrails:** never delete or overwrite; identical hash = already filed;
never invent a revision. Scanned or unreadable files are stored and shown
"not filed".

## 4. Search (Jev only)

Postgres full-text shortlist (plus fuzzy matching for numbers and names and a
construction synonym list), then Jev judges the shortlist. Compare both
TypeSafe shapes on a gold set and keep the faster one that is no less accurate:
[per-passage noul](https://docs.typesafe.ai/cookbooks/rerank_typesafe) versus
[one choice over up to 255 passages](https://docs.typesafe.ai/cookbooks/semantic_find)
with an "is the answer here?" noul. Document numbers and exact titles go to a
direct lookup with no Jev call. If later workflows need prose, Jev picks the
content and approved template wording writes it.

## 5. Stack and hosting

- Go single binary: API + workers, embedded Vite + React SPA, SSE.
- PostgreSQL 17 with built-in full-text search (no pgvector: there are no
  embeddings), on the same VPS over a unix socket. sqlc + pgx. SQL migrations
  applied at startup.
- Files on local NVMe, named by content hash.
- Hand-written Jev HTTP/2 client: warm connections, ~24 in flight, one 1.2 s
  deadline per interactive call including queue wait, no hot-path retries, a
  circuit breaker, and every call's latency, tokens and confidence logged.
- Magic-link auth written in Go; invite-only multi-org; every row carries
  `org_id` and a test proves no cross-org reads.
- Tables: orgs, users, memberships, invites, sessions, projects, files,
  documents, decisions (one row per field per document: value, confidence,
  band, decided by rule/Jev/user, question version), passages, jobs.
- Hosting: rebuild the sitewise.au VPS clean. Caddy + systemd, no Docker,
  Dokploy or Supabase. Continuous Postgres backup and nightly file backup to
  off-site object storage; restore rehearsed before any user is invited. Only
  secrets and keys carry over from the old deployment.

## 6. Failure handling

A failure never blocks filing and never hides. Jev down: grey chips,
background retry with backoff, circuit breaker. Unreadable file: stored, not
filed. Interrupted upload: re-drop is safe (hash dedupe). Crash: jobs resume
from the `jobs` table. Missing secret at startup: refuse to start. `/health`
reports DB, Jev reachability, queue depth and oldest job; a built-in speed page
shows live p50/p90.

## 7. Testing and gates

- Go unit tests for harvesters (over-find), revision ordering, supersedes and
  org isolation.
- Recorded Jev responses replayed in tests (as TypeSafe's cookbooks do).
- Accuracy eval per field on Hale, Petersham and Newham answer keys; the same
  run sets each question's threshold.
- Speed benchmark fails the build when intake p50 > 1 s or p90 > 2 s; nightly
  live run against Jev.
- One Playwright end-to-end test: create a project, drop files, check chips.
- No benchmark of the old app (owner decision).
