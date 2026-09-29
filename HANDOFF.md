# Handoff — 29 September 2026

For the next agent (any provider). Read `AGENTS.md`, then
`docs/design/2026-09-29-foundation-design.md`, then `knowledge/SCHEMA.md`.

## Where things stand

The design session with the owner is complete and recorded in the design doc.
Decisions that must not be re-litigated:

- **Jev (TypeSafe System One) is the only AI.** No LLMs or embeddings at
  runtime. Ground every Jev choice in docs.typesafe.ai (patterns, confidence,
  jev-1.13 limitations, cookbooks, API) and cite the page.
- **Speed is the first objective**; budgets and failing benchmarks.
- **Stack:** Go single binary, PostgreSQL 17 on the same VPS (no Supabase, no
  pgvector), local NVMe files, React SPA, SSE, Caddy + systemd. Replaces
  sitewise.au; rebuild that VPS clean. Carry over only secrets/keys.
- **v1 slice:** instant intake (filed in about 1 s p50, p90 < 2 s). PDF with
  text layer + DOCX/XLSX; drop into a chosen project; invite-only multi-org.
- **Domain model (owner's direction, most important):** ignore clerk's PM
  doctrine (`clerk/docs/clerk-brief.md`). Build from first principles on the
  physical building: **systems → determinants → rules (NCC, AS, state) →
  interfaces between systems → failure modes**. Jev reads the evidence; code
  does the physics (table lookups, graph walks, conflicts). Knowledge is
  compiled once into `knowledge/` from `clerk/data/seed/`.
- **Do not** benchmark the old app or back up Supabase (owner decision).

## In flight when this session ended

Six extraction agents were launched in parallel. Each writes only its own
folder and **does not commit**:

| Cluster folder | Scope | Top-level systems |
|---|---|---|
| `knowledge/clusters/fire/` | passive + active fire, egress, access | fire-passive, fire-active, access-egress |
| `knowledge/clusters/structure/` | site, ground, substructure, structure | site, substructure, structure |
| `knowledge/clusters/envelope/` | envelope, interiors, waterproofing, acoustics, BASIX/NatHERS/Section J fabric | envelope, interiors |
| `knowledge/clusters/services-wet-air/` | mechanical + hydraulic (incl. J5) | mechanical, hydraulic |
| `knowledge/clusters/electrical-comms/` | electrical, comms/security, lifts (incl. J6, PV, EV) | electrical, comms-security, vertical-transport |
| `knowledge/determinants.yaml`, `knowledge/tables/`, `knowledge/clusters/determinants/` | complete determinants; verify seed numbers and NCC 2019→2022 clause numbers against NCC 2022 primary text; write verified lookup tables only | — |

Each cluster must end with `systems.yaml`, `rules.yaml`, `interfaces.yaml`,
`failure_modes.yaml`, optional `proposed_determinants.yaml`, and `REPORT.md`
(coverage table, seed contradictions, cross-cluster interfaces, unresolved
system ids, tables needed). The determinants agent writes `VERIFICATION.md`
and `REPORT.md`.

**A folder without `REPORT.md` means that agent did not finish.** Re-run it
with this brief: read the files listed at the top of this handoff plus the
fire worked example; read every seed relevant to the cluster in full
(`ncc-reference-guide.md` and `as-standards-reference.md` for all clusters);
extract physical-building knowledge only (skip fees, RFPs, stages, contracts,
cost, programme, role/doctrine, except technical facts inside them); copy
seed clause numbers with "(seed numbering; NCC 2019)", `clause_verified:
false`, and every numeric claim with `verified: false`; interfaces are the
most important output, including cross-cluster edges (`cross_cluster: true`);
write only inside the cluster folder; no git writes; finish with
`python tools/check_knowledge.py --only <folder>` at 0 errors.

Known seed contradiction to resolve: compartment limits (ncc-reference-guide
says 5,500 m² unsprinklered Class 5; as-standards-reference AS 2118 section
says 3,500 m²). The seeds use NCC 2019 numbering throughout.

## Next steps (in order)

1. Check every cluster folder has `REPORT.md`; re-run any that don't.
2. `python tools/check_knowledge.py` across everything; fix errors.
3. **Merge (lead job):** reconcile cross-cluster interfaces (each agent saw
   only its own side; de-duplicate edges that two clusters both wrote, keep
   one id), resolve unresolved child-system ids, fold `proposed_determinants`
   into `knowledge/determinants.yaml`, apply `VERIFICATION.md` results to rule
   `clause`/`numbers` fields. Then `--strict` should pass.
4. Commit per cluster (the owner's git identity is configured in this repo;
   end messages with the agent's co-author line).
5. Report to the owner: counts, gaps, contradictions, whether NCC 2022
   primary text was readable, and tables written vs pending.
6. Then: pick the pilot cluster (recommended: fire, with its hydraulic fire
   water, electrical supply and structural load edges), install Go, and plan
   the intake implementation (`superpowers:writing-plans`).

## Notes

- Go is **not installed** on this machine yet. PyYAML is.
- `tools/check_knowledge.py` is a dev-only interim tool; the Go loader
  replaces it.
- YAML trap: `on`/`yes`/`no` keys parse as booleans (already bitten once).
- Owner memory for Claude sessions: `new-sitewise-jev-only-rebuild.md` in the
  clerk project memory.
