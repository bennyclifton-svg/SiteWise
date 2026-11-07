# Combined engagements: several disciplines under one appointment (R17)

Status: direction agreed by the owner on 7 November 2026 (decisions at the
end); not built. Lane A (procurement packages, RFPs, cost figures). Owner
ruling R17 and the owner's follow-up of 7 November 2026. This replaces the
first draft of this note, which covered services only.

## What the owner asked for

Any group of consultant disciplines can be bought under one agreement, one
fee proposal and one invoice stream. Examples the owner gave:

- structural and civil, in one fee proposal from one RFP;
- architecture and interiors;
- architecture with access as the architect's subconsultant;
- services engineering: mechanical, electrical, hydraulic and fire, sometimes
  with acoustic;
- an ESD firm that also offers fire engineering, facade and acoustic.

On Procurement the PM ticks several packages and presses **Combine**. The
RFP that comes out is the separate discipline RFPs gathered into one
document. It keeps the same detail for each discipline's scope and
deliverables as if each were issued alone.

## User outcome

One combined engagement is one appointment everywhere: one Procurement row,
one RFP document, one set of bids, one award, one fee line, one supplier for
invoices. Inside it, each discipline keeps its own scope, stages,
deliverables and settings.

## Why it matters

- **Answer trust.** Today a combined proposal adds a new row beside the
  separate ones, so the cost plan counts the same service twice. In T03 a
  four-discipline services job carried five fee lines.
- **Time to decision.** The PM can issue one RFP and compare like with like,
  instead of removing spare rows and fee lines by hand.

## What happens today

1. Each package is one discipline. A discipline may appear once per project
   (`disciplineTaken`, `internal/store/package_proposed.go`).
2. A filed proposal gets a row for its filed discipline if none exists
   (`EnsureProposalPackage`, `internal/store/proposal_package.go`). That is
   how "Services Engineer" appeared as a fifth row.
3. Each package gets one draft fee line (`internal/costs/suggest.go`).
   Award writes stage lines under that one line (`internal/store/awards.go`).
4. An invoice finds its supplier only when exactly one package matches the
   firm (`settleSupplier`, `internal/invoices/read.go`).
5. An RFP is one report per package (`reports.package_id`,
   `internal/db/migrations/020_reports.sql`). It is built from one discipline
   template plus the shared skeleton (`rfp.Resolve`,
   `knowledge/rfp/*.yaml`). Its choices are stored per report
   (`report_settings`: stages, scope lines, positions, custom lines).
6. A subconsultant is a package marked "engaged by the lead consultant"
   (`packages.engaged_by`, migration `021r`). The lead's RFP gets one
   sentence naming it (`engagement`, `internal/rfp/resolve.go`). The
   subconsultant's scope is not in the lead's RFP. Its own row and fee line
   stay, so its fee can be counted twice.

## Options

### A. A group row whose members stay as discipline rows (recommended)

Combine creates one new **group** row. The chosen packages become its
members and show indented beneath it. Each member keeps its discipline, its
RFP choices and its scope. The group holds what belongs to the appointment:
the bidders, the award, the fee line, the firm and the invoices.

- Better:
  - nothing is lost when combining or uncombining, because every member
    keeps its own settings;
  - the RFP can show each discipline in full;
  - it covers every example above, including subconsultants;
  - one appointment is one package for award, cost and invoices, so the
    rules that already assume that keep working.
- Worse:
  - a new link from package to group, and the screens must show nesting;
  - the RFP assembles several discipline templates (more work in one
    build, still in memory).

### B. One row that lists the disciplines it covers

The first draft's idea: the separate rows are replaced by one row with a
"covers" list.

- Better: simpler data.
- Worse:
  - the members' RFP choices have nowhere to live, so a combined RFP cannot
    keep per-discipline detail;
  - uncombining starts each discipline from scratch.

This fails the owner's granularity requirement.

### C. Keep separate rows and split one fee across them

- Worse: award would have to invent each discipline's share when the bid
  gives a lump sum. That breaks "unknown stays unknown". The firm's ABN would
  match several packages, so invoices never find their supplier.

## How option A works on each surface

### Procurement

- Each consultant row gets a tick box. With two or more ticked, **Combine**
  appears. One confirmation lists:
  - the members;
  - the draft fee lines that will merge (see Cost plan);
  - the group's name. The default joins the disciplines ("Structural and
    civil engineering").
- **Renaming.** Double-click the group's name on Procurement to change it,
  with the same inline edit the cost plan uses for line labels. The package
  name can already be changed through the server (`procurement` patches
  carry `title`); today the table shows the standard discipline name
  instead. Member rows keep their standard discipline names, because those
  names drive matching and RFP templates.
- The group row shows the members underneath, one status, the bidders and
  the award.
- **Uncombine** reverses it. **Remove from group** takes one member out.
- Rules:
  - consultant (services) packages only;
  - a discipline belongs to one group at most;
  - a project may have several groups.
- Combine is refused while any member, or the group, has a live award. The
  PM undoes the award first. Undo is already refused once an invoice is
  allocated, and that stays.
- Default suggestions stay separate, as today (decision D1). Combining is
  one click.
- **Subconsultants.** "Access as the architect's subconsultant" is simply a
  group of architecture and access, the same as combining electrical and
  hydraulic. The client signs with the architect, who manages the access
  consultant. What the client needs is an agreed scope and a sum for each
  discipline, which the group gives. Combine therefore replaces the separate
  "engaged by" setting (see Migration).

### RFP: one document, every discipline in full

The group has one RFP report. Its document is built in this order:

1. **Shared parts, once.** Page one, the project description, program, the
   client's terms, insurance, submission requirements and evaluation. These
   come from the shared skeleton, as now.
2. **One part per discipline.** Each member's own part is resolved from its
   own template and its own stored choices: stages in or out, scope lines,
   added lines, positions, deliverables and its conditions. The PM edits a
   discipline's part exactly as they would edit that discipline's RFP alone.
3. **Within-engagement items.** Some items one template hands to another
   discipline, such as electrical's "fire detection design (allocated to
   fire services)". When both disciplines are in the group, the item is
   shown once, under the discipline that does it, marked as within this
   engagement. No item is dropped.
4. **One fee schedule.** A section per discipline with that discipline's
   stages, a sum for each discipline, and a total for the engagement. The
   RFP asks for a sum per discipline, because that is what the client needs.
   A bidder who still gives only one amount for a stage (covering several
   disciplines) is not rejected: the stage is recorded as combined, and the
   affected disciplines' sums show as not stated rather than guessed.
5. **Checks.** The RFP's warnings and blocks are every member's checks
   combined. Issuing is one action and one transmittal for the whole
   document.

While a discipline is in a group, its own RFP cannot be issued alone. An
RFP issued before combining stays in history and is not reissued.

### Fee proposals and the price table

- A proposal dropped on the group row, or filed as any member's discipline,
  goes to the group by a code rule. A combined filing such as "Services
  Engineer" also goes there. It gets its one existing Jev fan-out, no more.
- The price table's columns follow the fee schedule: discipline by stage,
  plus combined cells where the bid gives one amount. Every bidder on the
  group is compared on the same columns.

### Award and cost plan

- The group gets one fee line that is a **parent** in the cost plan. Each
  member's existing fee line moves under it as a child, keeping its own
  budget. This is the same nesting the cost plan already uses when an award
  puts stage lines under a fee line. Expand it to see one line per
  discipline, or collapse it to see one contract sum. The roll-up is the
  existing group roll-up (`internal/costs/rollups.go`), so the total is
  computed in Go and nothing is counted twice.
- Award is made on the group and writes under the matching child:
  - each discipline's priced stages under that discipline's line;
  - a stage priced as one combined amount goes under the group line itself;
  - never a made-up split.
- Frozen baselines and issued history are untouched. Construction lines
  (for example "Mechanical services") are not fee lines and are not
  touched.
- Uncombine moves each member's line back to the top level with its budget,
  and the empty group line is removed. Refused while the group is awarded.

### Invoices

While grouped, members are left out of supplier matching. The firm matches
only the group, so the existing rule settles the supplier. Allocation to
stage lines works as today.

### Program

The group gets one set of stage lines through the existing stage sync: the
members' stages in program order, one line per stage. Dates are computed in
Go, as now.

## What calls Jev

Nothing new. A dropped proposal still gets its one existing fan-out after
filing. The following make no Jev call:

- combine, uncombine and remove from group;
- the RFP and the price table;
- award, cost lines and invoice matching edits;
- the program.

## Speed budgets (`bench/budgets.json`)

| Path | p50 | p90 |
|---|---|---|
| Combine, uncombine, remove from group, rename (`packages_write`, with the fee lines re-parented in the same transaction) | 100 ms | 250 ms |
| Procurement read with groups (`packages_read`) | 100 ms | 250 ms |
| Group RFP build (`report_assemble`) | 300 ms | 1,000 ms |
| RFP read and setting edit (`report_read`, `report_setting`) | 100 ms | 250 ms |
| Price table (`price_table`) | 150 ms | 300 ms |
| Award or undo (`award_write`) | 200 ms | 500 ms |
| One invoice (`invoice_intake`) | 1,000 ms | 2,000 ms |
| Stage sync (`stage_sync_write`) | 100 ms | 250 ms |

New benchmarks:
- combine and uncombine;
- a five-member group RFP build, run beside the existing `report_assemble`
  benchmark.

## Data and migration

- `packages` gains a nullable `group_id`. It points to another package in
  the same org and project, and is enforced by a foreign key on (org,
  project, id) like every package reference now.
- A group row is a services package with no discipline and a role of
  "group". Its RFP report uses the existing `reports.package_id`.
- Each member keeps its own report and settings as the store of its choices.
  The group's RFP reads them. No settings are copied, so uncombining is
  lossless.
- `disciplineTaken` counts members as their own discipline. A group has no
  discipline of its own.
- **Existing projects.**
  - Nothing changes until the PM combines.
  - A live "Services Engineer" row is left as it is, with a prompt to
    combine it with the separate services rows.
  - A package marked "engaged by the lead consultant" gets the same prompt:
    combine it with the lead. Nothing is combined automatically.
  - Once combining exists, the "engaged by" setting is retired.

## Edge cases

- **Awarded member.** Combine is refused; undo the award first.
- **Draft RFPs on members.** They become the members' parts in the group's
  RFP, unchanged.
- **Removing the group.** Members are released as separate rows. The group's
  fee line goes through the existing confirmation.
- **Removing a member** takes its fee line out from under the group to the
  top level. Deleting the package uses the existing confirmation.
- **A combined proposal on a split project.** No new row. The proposal shows
  on Procurement as unplaced, with "Combine for this proposal" (the PM ticks
  the members; the app does not guess) or "Place on one discipline".
- **Two groups.** Allowed, as long as no discipline is in both.
- **Works and supply packages** (trades, head contract) cannot be combined.
- **Wrong org or project.** Refused, like any package write.

## Build order (each slice tested and benchmarked on its own)

1. Data, Combine, Uncombine, Remove from group and renaming on Procurement.
   Member fee lines nest under the group's line. Invoices match the group.
2. The group RFP document: shared parts, discipline parts, within-engagement
   items, the fee schedule and checks. Issue as one.
3. Proposal routing to the group, the price table by discipline and stage,
   award writing discipline or stage lines.
4. Program stage lines for a group.
5. Prompts for existing "Services Engineer" and "engaged by lead" rows, then
   retire the "engaged by" setting.

Measured by re-running the T03 scenarios, with the combined services
proposal placed on a services group: one fee line, the invoice supplier
settled, and the RFP holding all four disciplines.

## Out of scope

- Splitting a lump-sum fee into disciplines.
- Changing default suggestions.
- Combining trade or works packages.
- A Jev question about which disciplines a proposal covers.

## Owner decisions (7 November 2026)

1. **Roll-up.** Combined disciplines roll up into one figure, the way an
   architect's fee sub-items can be shown as several lines or collapsed into
   one contract sum. Each discipline keeps its own line beneath the group.
2. **Stage priced as one amount.** Accepted as recommended: the RFP asks for
   a sum per discipline, and a bid that gives only one amount for a stage is
   recorded as combined rather than rejected or split by guesswork.
3. **Subconsultants.** No separate treatment. Architecture with access is a
   group like electrical with hydraulic. The client needs an agreed scope and
   a sum for each discipline, not visibility of the subconsultant agreement.
4. **Names.** The default joins the discipline names, and the PM can
   double-click the group's name to change it. Member rows keep standard
   discipline names.
