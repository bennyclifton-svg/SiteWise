# Trades catalogue: let one head contract or every trade

Date: 10 November 2026. Lane A (procurement packages, issued documents).
Status (10 November 2026): T1 built (trades list loaded, Trades drop-down
offers 55, row create accepts them) and T3 drafted (47 new RFT files,
research notes in `docs/research/2026-11-10-trades/`). T2, T4-T6 not built.
Uncommitted.

Skeleton: `knowledge/works/trades.yaml` (55 trades, 9 existing, 46 new,
5 suggested bundles).

## The user outcome

Two people use Procurement Trades:

- **A PM letting the works to one main contractor.** They add Main Works
  and, sometimes, carve out one or two trades (say, structural steel or
  joinery) let directly. This already works for the existing nine trades.
- **A builder letting every trade.** They leave out Main Works and add a
  row per trade: demolition, concreting, formwork, steel fixing, framing,
  roofing, plasterboard, tiling, painting, and so on. They often let a few
  trades to one firm (form, reo and pour to one concreter).

Both need a full list of trades, a tender (RFT) for each, and a way to
put several trades into one package. Today the list has nine trades
because it is borrowed from the filing vocabulary.

## Why it serves time-to-decision and answer trust

A builder who can't find concreting in the list can't tender it, compare
prices or award it, so the Cost Plan and Program never see that contract.
A missing trade sends the PM back to a spreadsheet.

## Owner decisions (10 November 2026)

1. The trades list in `knowledge/works/trades.yaml` is agreed (house and
   commercial, including piling, precast, curtain wall and lifts).
2. Commercial trades go in now.
3. Trades combine like consultants: the PM ticks any trades and combines
   them into one package for one firm. Suggested bundles are prompts
   only, never a limit.
4. Research of each trade's detail goes to research subagents after the
   owner reviews this plan and the skeleton.

## Design

### 1. A trades list separate from filing

`data/intake/disciplines.json` is the filing vocabulary: Jev picks one of
its 58 entries for every uploaded document. It holds nine trades. Adding
46 more there would make Jev choose from about 104 options on every
upload, slowing filing and risking filing answer keys, and the loader
pins the count at 58.

So the trades list lives in `knowledge/works/trades.yaml`. Filing is
unchanged. Procurement reads:

- consultants: the `consultant.*` entries of the filing vocabulary (as
  today);
- trades and supplies: `knowledge/works/trades.yaml`.

The nine existing trade ids keep their ids and labels, so no stored
package, cost line or RFT changes.

Code that reads the trade list today, and moves to the merged list:

| Place | Today | Change |
|---|---|---|
| `GET /catalog` (`internal/httpapi/documents.go` `getCatalog`) | filing list | filing consultants + trades file; the web drop-down is unchanged |
| `postProcurementRow` (`internal/httpapi/procurement.go`) | accepts filing ids | accepts merged ids; 422 for anything else |
| proposal reader labels (`internal/httpapi/server.go`) | filing labels | merged labels |
| RFP register labels (`internal/rfp/register.go`) | filing labels | filing labels (filing shows filed documents; no change) |
| cost elements check (`internal/knowledge/cost_elements.go`) | trades from filing list | trades from trades file |
| `procurement.DisciplinePackage` | prefix `trade.`/`supplier.` | unchanged; new ids keep the prefixes |

A document filed by Jev still files under one of the existing 58. A
tender dropped on a concreting row is read as a tender because the row's
discipline starts with `trade.` (`internal/proposals/read.go`). No new
Jev question.

The knowledge checker gains a `trades.yaml` check: unique ids, `trade.`
or `supplier.` prefix, `builds` names real systems, bundle members exist,
and every id the filing list holds under `trade.`/`supplier.` is present
(so the two never disagree on the nine).

### 2. An RFT for every trade

Each researched trade gets `knowledge/rfp/trade-<name>.yaml` in the
existing trade RFT shape (role, duties, stages and lines, provenance).
Until its file exists, a trade uses `trade-general.yaml`, which already
prints the package title, so a new trade gets a usable RFT from day one.

### 3. Main Works carve-out per trade

Main Works price rows follow the building (site, substructure,
structure, envelope, interiors, fire, services). Today a whole row is
carved out with `separate:` when that trade is let separately. That
works for the nine trades, where one trade holds a whole row. It does not
work for concreting, formwork and steel fixing, which share the
substructure row.

Proposed: `separate:` moves down to the line. A Main Works line names
the trade that may take it (`{key: footings, text: ..., separate:
trade.concreting}`). When a live package holds that trade, the line prints
"By separate contract: Concreting" and the row's price excludes it.
A row whose lines are all let separately prints as excluded, as today.
Main Works gains the lines it lacks so every new trade has a line to take
(about 15 lines).

When a trade sits in a combined package, the line names the package
("By separate contract: Formwork, steel fixing and concreting").

Owner (10 November 2026): line-level carve-out, as recommended. Was O1:
line-level carve-out (proposed) keeps the
head contract tender to about 15 price rows. The alternative is one Main
Works price row per trade (about 45 rows), which reads like a builder's
trade breakdown but is a long tender for a head contractor. Recommend
line-level.

### 4. Combining trades

Built today for consultants only. Two checks block trades:

- the screen allows only consultant rows to be ticked
  (`web/src/Procurement.tsx` `combinable`);
- the store refuses non-consultant rows (`internal/store/package_groups.go`,
  "only consultant rows can be combined").

Change:

- allow trade and supply rows on the Trades tab; a group holds trades or
  consultants, never both;
- Main Works can't join a group (it already holds the whole building);
- the combined RFT prints each member trade as its own section with its
  own lines and one price per section, as the combined consultant RFP
  does per stage;
- award, undo and Program treat a trade group as they treat a consultant
  group today;
- suggested bundles (`bundles` in the skeleton) appear as a Combine
  prompt when two or more members are rows on the project, the same way
  the services consultants prompt works; dismissable.

### 5. Proposing trades

Out of the first slices. Later, Profile scope (which systems the project
has) could propose the trades a builder letting every trade needs. The
`builds` field makes that a lookup, not a Jev call.

## Research subagents (after owner review)

Five research agents in parallel, one per group: site and substructure;
structure and envelope; interiors; fire and services; completion and
supplies. Each fills, per trade:

- **Role and duties**: what the firm is engaged to achieve, in the
  existing RFT voice.
- **Scope lines**: the RFT stages and lines (design or shop drawings,
  preliminaries, the works, testing, handover).
- **Interfaces**: what it takes from and leaves for its neighbours
  (who supplies the reo chairs, who sets out the slab, roofer vs roof
  plumber on flashings).
- **Hold points and inspections**: which the engineer, certifier or
  authority sets.
- **Certificates it signs** (for example waterproofing, termite,
  electrical compliance).
- **Licence class** (NSW first; other states noted where they differ).
- **House and commercial differences**.
- **Main Works line** it carves out.
- **Sources**: a seed under `data/reference/clerk/data/seed/` with an
  anchor (residential-construction-guide, role-builder,
  role-owner-builder, trade-interfaces-coordination-guide,
  finishes-residential, mep-residential, structural-residential,
  commercial-construction-guide and others).

Rules for the agents: every file `status: draft`; any standard or rule
number named stays `verified: false` until checked against the
instrument itself; no rates, prices or product selections; no code.
`python tools/check_knowledge.py` must pass on what they write. The lead
session reviews each group before it lands.

## Slices

| # | Slice | Depends on |
|---|---|---|
| T1 | Trades file loader, checker rule, merged catalogue and row create; drop-down shows 55 | owner review of skeleton |
| T2 | Main Works line-level carve-out and the extra lines | O1 |
| T3 | Research: per-trade detail and RFT files (subagents) | owner review |
| T4 | Combine trades: lift the two checks, combined RFT by trade section, Main Works excluded | T1 |
| T5 | Suggested bundles as Combine prompts | T4 |
| T6 | Cost Plan lines follow the new trades (`knowledge/costs/elements.yaml` overrides) | T1, T2 |

## Budgets

No new Jev call anywhere: row create, catalogue read, RFT create and
refresh, combine and award stay off Jev.

| Path | Budget (bench/budgets.json) |
|---|---|
| Procurement row create | `packages_write` p50 ≤ 100 ms, p90 ≤ 250 ms |
| Catalogue read | `packages_read` (existing) |
| Combine trades | `packages_combine` p50 ≤ 100 ms, p90 ≤ 250 ms |
| RFT refresh | `stage_sync_write` p50 ≤ 100 ms, p90 ≤ 250 ms |

Filing stays p50 ≤ 1 s, p90 ≤ 2 s and is not touched.

## Edge cases

- A stored package with a trade id that is in neither list (old data):
  shown with its stored title, as today.
- A trade removed from the file later: existing rows keep working by
  title; the drop-down no longer offers it.
- Concreting let separately but formwork and steel fixing left with Main
  Works: each line follows its own trade.
- Every trade let separately and no Main Works row: no carve-out text is
  printed anywhere; nothing holds preliminaries unless a trade does
  (builder's own cost, shown as a Cost Plan line the PM adds).
- A supply (frames and trusses) let separately while carpentry stays with
  Main Works: the Main Works framing line reads "supply by separate
  contract; install included".
- Two bundles share a member: the prompt for the second hides once the
  member is grouped.

## Out of scope

Proposing trades from Profile scope (slice 5 of the design, later),
rates or prices, trade licence checks against a register, request for
quote, filing changes.

## Done

- The Trades drop-down offers 55 trades and supplies; filing answer keys
  and the filing budget unchanged.
- A builder can add concreting, formwork and steel fixing, combine them,
  issue one RFT and award it.
- A PM with Main Works and a separate concreter sees "By separate
  contract: Concreting" on the Main Works footing and slab lines.
- `python tools/check_knowledge.py` and the Go and web tests pass;
  benchmarks within budget.
