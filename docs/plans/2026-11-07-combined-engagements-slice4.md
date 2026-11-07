# Combined engagements, slice 4: program stage lines for a group

Date: 7 November 2026. Owner: "Yes, please start building" (the five-step
build order). Design: `docs/design/2026-11-07-combined-services-package.md`,
section "Program": "The group gets one set of stage lines through the
existing stage sync: the members' stages in program order, one line per
stage. Dates are computed in Go, as now." Slices 1 to 3 are merged on local
main (`da3f4d2`).

## What "stage lines" are today

Since the owner direction of 25 October 2026 (program plan 13.5.1),
packages add no program rows: every consultant follows the stage's own
activities, and the program's dates and links live only on those rows. A
package's stage lines are its `package_stages`, each pointing at a program
stage (`program_stage_id`, plan 7.6). The stage sync
(`internal/store/stage_sync.go`, `stage_sync_write`) carries program
changes one way to them, to the RFP draft built from them and, on an
awarded package, to the award's cost lines (added stage: an unpriced line
under the award; removed stage: kept, marked "removed from program";
rename: copied). A group today has no stage lines, so its RFP shows no
stage rows and its award gets nothing from a program change.

## Feature

- **User outcome:** a combined engagement has one set of stage lines: one
  per program stage that any member includes, in program order. The group's
  RFP shows them and offers Update or Keep when the program's stages change,
  and the group's award follows them as a single package's award does. The
  members' own stage lines sit unchanged and inert while grouped, and are
  live again after uncombine or remove.
- **Why:** answer trust and time to decision. One appointment has one stage
  plan; a program change is offered once on the engagement, not once per
  discipline, and the awarded group's cost lines follow it.
- **Lane:** A (procurement packages, awarded cost lines).

## In this slice

1. **The group's stage lines** (`internal/store/group_stages.go`, new),
   written in the combine and remove transactions, under the project lock:
   - wanted set: each member's live stage, keyed by its live program stage
     (or, with no program, its catalogue stage); the union, one line per
     key, labelled as the program labels it, in program order (catalogue
     order with no program);
   - combine creates them through the existing `insertPackageStage`, which
     links each to the program and records the plan on the group;
   - remove from group (two or more members stay) brings the group's lines
     to the remaining members' union: an existing line keeps its id and
     version; a line no remaining member includes is retired unless
     something holds it (`stageHeld`), in which case it stays;
   - uncombine retires the group row as now; its lines go with it.
2. **Members contribute no separate lines while grouped:**
   - a member's package stage sync reads with `group_id` set, no picker and
     no offer; Update or Keep on it is refused (`ErrPackageGrouped`, 409);
   - creating or changing a member's stage is refused the same way;
   - a member's own RFP draft offers no stage change while grouped, and its
     Update or Keep is refused;
   - members' rows are never rewritten: ids, labels, versions and recorded
     plans stay as they were, so uncombine or remove restores them, and the
     next stage sync on a released member offers whatever the program
     changed meanwhile.
3. **The existing stage sync on the group, unchanged code:** the group's
   package stage sync and its RFP's (slice 2's group RFP is on the group
   package) read the group's lines. Award (slice 3) is on the group, so an
   awarded group is `awarded_contract`: an added stage becomes one unpriced
   line under the group's award line, a removed one is kept marked "removed
   from program", a rename is copied. Members' and combined award lines are
   never revalued.
4. **Group RFP screen** (`web/src/Reports.tsx`): show the existing stage
   notice (`RfpStages`) on the group RFP too; slice 2 hid it because the
   group had no lines.
5. **Program is untouched:** no rows per package, no Jev, every date in Go.
   Combine, remove and rename change no program row, duration, link or
   constraint.
6. **Benchmark:** `stage_sync_write_group`, Update on a five-member
   group's stages after a stage rename in Program, budget p50 ≤ 100 ms,
   p90 ≤ 250 ms (the `stage_sync_write` budget). `packages_combine` keeps
   its budget with the group's lines written in the same transaction.

## Out of scope

- rows per package in Program (owner direction 13.5.1 stands);
- a stage picker on the Procurement group row (stage controls stay inside
  the RFP, as for single packages);
- relabelling the group's cost-plan line on rename (known gap from slice 3);
- prompts for existing "Services Engineer" or "engaged by" rows (slice 5);
- backfilling lines for groups combined before this slice: the feature is
  unreleased; such a group gets its lines when combined again.

No Jev call in combine, uncombine, remove, rename, stage sync or Program. No
new dependency. Every query scoped by `org_id` and project.

## Done

- Go tests:
  - combine gives the group one line per stage any member includes, in
    program order, linked to the program and with the plan recorded;
    members' rows unchanged;
  - with no program: the union in catalogue order; a program prepared later
    links the group's lines;
  - a member's stage sync is inert while grouped; its Update, Keep, stage
    create and stage change are refused;
  - remove from group: the group's lines become the remaining union, kept
    lines keep their ids; the removed member's stage sync is live again;
  - uncombine: the group's lines are gone with the row; members' stage sync
    offers the program changes made while grouped;
  - rename: the lines and the sync are unchanged;
  - a program rename, removal and addition offered once on the group;
    Update copies, retires and adds as for a single package;
  - an awarded group: Update adds an unpriced line under the group's award
    line; contract values unchanged; a removed stage is kept;
  - user edits to a program stage's duration and links survive combine and
    remove;
  - wrong org and wrong project refused.
- A browser test: combine two rows, Create RFP on the group, rename a stage
  in Program, the group RFP offers the rename once and Update clears it;
  uncombine, and a member's stage sync is live again.
- Go suite and web build pass (failures that pre-date this work aside). The
  bench shows `stage_sync_write_group` within budget.

## Edge cases

- Members with different stages: the union; a stage none includes is not a
  line.
- A member's stage kept after its program stage was deleted ("removed from
  program"): it stays the member's, held by the member's own records; the
  group does not take it.
- Two program stages with the same label: one line, as for a single
  package's defaults.
- A program created after combine: prepare links the group's lines as it
  links any package's.
- Remove while the group is awarded: still refused (slice 1).
