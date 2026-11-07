# Combined engagements, slice 1: combine, uncombine and rename on Procurement

Date: 7 November 2026. Owner: "Yes, please start building." Design:
`docs/design/2026-11-07-combined-services-package.md` (direction and owner
decisions agreed). This is build step 1 of 5.

## Feature

- **User outcome:** on Procurement the PM ticks two or more consultant rows
  and presses Combine. One group row appears with the chosen rows indented
  beneath it. The PM can rename the group by double-clicking its name,
  Uncombine it, or Remove one member. In the cost plan, the group has one
  parent fee line with each member's fee line beneath it, collapsible to one
  contract sum. An invoice from the group's firm finds the group as its
  supplier.
- **Why:** answer trust. A combined proposal today adds a duplicate row and
  fee line, so the cost plan counts a service twice (T03: five services fee
  lines for four services).
- **Lane:** A (procurement packages, cost figures).

## In this slice

1. **Data.** Migration `045_package_groups.sql`:
   - `packages.group_id`, nullable, with a foreign key on (org_id,
     project_id, group_id) to packages, like every package reference now;
   - a way to mark a row as a group (a role column, `single` or `group`).
     A group is a services package with no discipline.
   - Constraints: a group has no group_id; members are services; a
     discipline row belongs to at most one group.
2. **Store and API**, each in one transaction, scoped by org and project:
   - **Combine**(ids, title):
     - refuse fewer than 2 ids, non-services, retired, already grouped,
       another project's package, or a live award on any member;
     - create the group row (title defaults to the members' discipline
       labels joined: "Structural and civil engineering");
     - set each member's group_id;
     - create the group's fee line and re-parent each member's fee line
       under it, keeping budgets;
     - move members' unawarded proposals (submissions) to the group,
       recording the member each came from.
   - **Uncombine**(group): refuse if the group has a live award. Otherwise
     reverse every step: members' fee lines back to the top level with their
     budgets, submissions back to their member, the empty group line and
     group row removed.
   - **Remove from group**(member): the same, for one member. If one member
     would be left, uncombine the whole group.
   - **Rename:** the existing package title patch, allowed on a group.
3. **Rules that must learn groups:**
   - `disciplineTaken`: members count as their discipline; a group has none.
   - Package suggestions do not re-suggest a member's discipline.
   - Cost plan prepare (`internal/costs/suggest.go`) must not create a
     duplicate fee line for a member or flatten the nesting.
   - Invoice supplier matching (`internal/invoices/read.go`) leaves members
     out and matches the group.
   - A member's own RFP cannot be issued while it is in a group (server
     check). The group row offers no RFP yet (slice 2).
4. **Procurement screen** (`web/src/Procurement.tsx`):
   - a tick box on consultant rows that can be combined;
   - Combine when two or more are ticked, with one confirmation listing the
     members, the fee lines that move and the group name;
   - the group row, with members indented beneath it and the firm columns
     on the group;
   - double-click the group name to rename (the cost plan's inline edit
     pattern);
   - Uncombine on the group, Remove from group on a member.
   - Plain labels, no new styling system.
5. **Benchmark:** a `packages_combine` path (combine then uncombine) in
   `cmd/intake-bench`, with budget p50 ≤ 100 ms and p90 ≤ 250 ms (the
   `packages_write` budget) added to `bench/budgets.json`.

## Out of scope (later slices)

- the combined RFP document (slice 2);
- proposal routing by member discipline, the price table by discipline and
  stage, and award writing under member lines (slice 3);
- program stage lines for a group (slice 4);
- prompts for existing "Services Engineer" and "engaged by" rows, and
  retiring "engaged by" (slice 5).

No Jev call in any path of this slice. No new dependency.

## Speed

- Combine, uncombine, remove and rename: p50 ≤ 100 ms, p90 ≤ 250 ms.
- Procurement read (`packages_read`), cost plan read (`costs_read`) and
  invoice intake keep their budgets.
- Filing stays p50 ≤ 1 s and p90 ≤ 2 s.

## Done

- Store tests for:
  - combine, uncombine and remove;
  - every refusal: fewer than two, works package, already grouped, awarded,
    wrong org, wrong project;
  - fee line nesting and budgets kept;
  - submissions moved and returned;
  - invoice supplier settles on the group;
  - prepare keeps the nesting;
  - a member's RFP issue refused.
- API tests for the new routes, including a wrong-org case.
- A browser test: tick two rows, combine, rename, uncombine.
- Go suite and web build pass (failures that pre-date this work aside), and
  the bench gate passes with the new path.

## Edge cases

- Duplicate combine requests (the same ids twice).
- A member retired after combining.
- A group whose members are all removed.
- A combine racing an award on a member (the transaction must re-check).
- Cost lines already frozen in a baseline: the baseline is untouched; only
  live lines move.
