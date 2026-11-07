# Combined engagements, slice 2: one RFP document for a group

Date: 7 November 2026. Owner: option 1, "proceed" once the RFP role and
purpose work was committed (`4cd9271`). Design:
`docs/design/2026-11-07-combined-services-package.md`, section "RFP: one
document, every discipline in full". Slice 1 (groups on Procurement) is
merged at `738b085`.

## Feature

- **User outcome:** a group row on Procurement offers Create RFP. The RFP is
  the separate discipline RFPs gathered into one document, with the same
  detail for each discipline's scope and deliverables. The PM edits each
  discipline's part exactly as they would edit that discipline's RFP alone,
  and issues the whole document once.
- **Why:** time to decision and answer trust. One RFP for one appointment,
  with nothing dropped or counted twice.
- **Lane:** A (an issued RFP is something a user trusts).

## In this slice

1. **Group RFP report.** Create RFP on a group makes one `rfp` report whose
   `package_id` is the group. Each member keeps its own report and settings
   as the store of its choices. Any member without a report gets a draft one
   created in the same transaction. Uncombining keeps every member report
   unchanged.
2. **Document assembly** (`internal/rfp`, the report build in
   `internal/httpapi/report_build.go`, `internal/store/reports.go`):
   1. **Shared parts once,** from the skeleton: page one, the project
      description, program, the client's terms, insurance, submission
      requirements and evaluation. Page one names the engagement (the group's
      title) and lists its disciplines.
   2. **One part per member,** in Procurement order. Each is resolved from the
      member's template and the member's own stored settings, with the
      current role section (purpose, duties, fit for purpose, boundary) and
      scope by stage. The bidder is told they may deliver any discipline
      in-house or by a subconsultant they name.
   3. **Within-engagement items.** A position or line one member's template
      allocates to another member's discipline is shown once, under the
      discipline that does it, marked "within this engagement". It is never
      dropped. When the other discipline is not in the group, wording is
      unchanged.
   4. **One fee schedule:** a section per discipline with its included
      stages, a sum per discipline, and the engagement total. The form asks
      for a sum per discipline, and allows a stage to be priced as one
      combined amount (owner decision 2).
   5. **Checks:** every member's warnings and blocks, each labelled with its
      discipline.
3. **Editing.** The RFP editor shows the shared parts, then each discipline's
   part with that discipline's existing controls. A setting edit inside a
   part writes to that member's report settings (existing setting route,
   addressed to the member's report). No new kind of setting key.
4. **Issue.** One action and one transmittal for the whole document. A
   member RFP still cannot be issued alone while grouped (slice 1). An RFP
   issued before combining stays in history.
5. **Export.** The existing export and print produce the combined document.
6. **Benchmark:** `report_assemble` for a five-member group in
   `cmd/intake-bench`, within the existing budget (p50 ≤ 300 ms,
   p90 ≤ 1,000 ms). `report_read` and `report_setting` keep 100 / 250 ms.

## Out of scope

- reading a combined proposal by discipline, the price table by discipline,
  and award under member lines (slice 3);
- program stage lines for a group (slice 4);
- prompts for existing rows and retiring "engaged by" (slice 5);
- relabelling a group's cost-plan line on rename (a known slice 1 gap, its
  own small fix).

No Jev call: RFP create, edit, assemble, issue and export never call Jev.
No new dependency.

## Done

- Go tests:
  - assembly order (shared parts once, then each member);
  - every member's included stages and lines present, matching that member's
    standalone RFP part;
  - within-engagement items shown once;
  - fee schedule sections, sums and total;
  - checks merged and labelled;
  - member settings edited through the group;
  - issue as one;
  - uncombine keeps member reports;
  - wrong org and wrong project refused.
- A golden test: a structural and civil group, and a four-discipline
  services group.
- A browser test: combine two rows, Create RFP on the group, change a
  setting in one discipline's part, issue.
- Go suite, web build and existing RFP browser tests pass (failures that
  pre-date this work aside). The bench shows the new path within budget.

## Edge cases

- A member with an issued standalone RFP: its settings are used; its issued
  history stays.
- A member retired or removed from the group while a group RFP draft is
  open: the next build drops its part, and the editor shows that it left.
- A group of a discipline with no RFP template: its part says scope to be
  described, and a blocking check asks for it.
- Two members whose templates both carry the same shared clause: printed
  once.
