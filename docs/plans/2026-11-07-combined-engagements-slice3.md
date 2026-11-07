# Combined engagements, slice 3: proposals, price table and award for a group

Date: 7 November 2026. Owner: "Yes, please start building" (the five-step
build order). Design: `docs/design/2026-11-07-combined-services-package.md`,
sections "Fee proposals and the price table" and "Award and cost plan".
Slice 1 (groups) is merged at `738b085`, slice 2 (group RFP) at `a9b0f2b`.

## Feature

- **User outcome:** a fee proposal for a combined engagement lands on the
  group row. The price table compares every bidder on the group's fee
  schedule: discipline by stage, plus a combined cell where a bid gives one
  amount for a stage. Award on the group writes each discipline's priced
  stages under that discipline's fee line, and a combined stage under the
  group line. The cost plan shows one contract sum for the group, expandable
  to each discipline.
- **Why:** answer trust. The T03 combined services proposal today creates a
  duplicate fee line; after this slice it is one fee line with the four
  disciplines beneath it and nothing counted twice.
- **Lane:** A (cost figures, procurement packages).

## In this slice

1. **Routing by code rule** (submission matching, proposal placement):
   - a proposal dropped on the group row goes to the group;
   - a proposal filed as a member's discipline goes to the member's group;
   - a combined filing ("Services Engineer", or a discipline that maps to a
     combined services role) goes to the group whose members it covers;
     when no group exists, behaviour is unchanged (the existing row or
     unplaced) — this slice does not create groups;
   - still exactly one Jev fan-out per proposal, after filing. No new Jev
     question.
2. **Reading the proposal's prices onto the group's schedule** (Go, from the
   existing fan-out's answers and the existing price cell model):
   - a priced line is assigned to a member discipline only when the bid
     names that discipline (heading, line label or section) by code match
     against the member's discipline labels and aliases;
   - a stage amount the bid gives once for several disciplines is a
     **combined cell** for that stage;
   - an amount that cannot be placed stays unplaced and visible for a cell
     correction; never split by guesswork (owner decision 2).
3. **Price table** for a group: columns are member disciplines by stage, in
   Procurement order, then combined cells, then the total. Every bidder on
   the group uses the same columns. Cell corrections work as today and can
   move an amount between a discipline cell and the combined cell. Totals in
   Go.
4. **Award on the group** (`internal/store/awards.go`):
   - one award row on the group;
   - each discipline's priced stage cells become stage lines under that
     member's fee line;
   - combined cells become stage lines under the group's parent fee line;
   - contract values come only from the firm's priced cells;
   - the group line's roll-up (existing `internal/costs/rollups.go`) gives
     the one contract sum, computed in Go;
   - a member or group budget that is blank adopts its contract sum, and a
     budget the PM set is never changed, matching the award budget rule in
     `AGENTS.md` (see "Coordination" below);
   - undo removes every line it wrote and restores budgets, as today.
   - Members cannot be awarded alone while grouped; uncombine stays refused
     while the group is awarded (slice 1 checks remain).
5. **Comparison report** for a group uses the same columns.
6. **Benchmarks:** `price_table` for a five-member group with five bidders
   (p50 ≤ 150 ms, p90 ≤ 300 ms) and `award_write` award plus undo on that
   group (p50 ≤ 200 ms, p90 ≤ 500 ms), beside the existing single-package
   samples. Existing paths keep their budgets.

## Coordination

Main's working tree has another session's uncommitted edits in
`internal/store/awards.go` and `internal/store/awards_budget_test.go`
(a blank budget adopts the contract sum on award; undo restores it). Build
on local main as committed. Keep group award code in new functions where
practical so the merge with those edits stays small. The orchestrator
reconciles at merge.

## Out of scope

- program stage lines for a group (slice 4);
- prompts for existing "Services Engineer" or "engaged by" rows, and
  retiring "engaged by" (slice 5);
- "Combine for this proposal" on an unplaced combined proposal (slice 5,
  with the prompts);
- splitting a lump sum into disciplines;
- relabelling a group's cost-plan line on rename.

No Jev call in routing, the price table, cell corrections, award, undo or
the comparison report. No new dependency.

## Done

- Go tests:
  - routing: dropped on the group, filed as a member's discipline, combined
    services filing, no-group fallback unchanged;
  - placement: named discipline cells, combined stage cell, unplaced amount
    kept, no split;
  - price table columns and totals for two bidders on one group;
  - award writes under member lines and the group line, contract sum equals
    the sum of priced cells, budgets per the rule, undo restores everything;
  - member award refused while grouped;
  - wrong org and wrong project refused.
- A store or API test replaying the T03 combined services proposal shape:
  one group fee line, four member lines beneath it, no duplicate.
- A browser test: a group with a proposal, price table shows discipline and
  combined columns, award, cost plan shows one sum expanding to members.
- Go suite, web build and existing award browser tests pass (failures that
  pre-date this work aside). The bench shows both group paths within budget.

## Edge cases

- A bid that prices only some members: the others stay blank (unknown), not
  zero.
- A bid with only one lump sum: one combined cell per stage if staged, else
  one unplaced total that the PM places; no split.
- A proposal filed before combining: slice 1 already moved it to the group;
  it reads onto the group's columns.
- A member removed after a proposal was read: its cells show as left the
  group and are excluded from award until re-placed.
- Two groups on one project: routing picks the group holding the filed
  discipline.
