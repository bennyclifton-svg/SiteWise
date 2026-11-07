# Combined engagements, slice 5: prompts for existing rows, retire "engaged by"

Date: 7 November 2026. Owner: "Yes, please start building" (the five-step
build order). Design: `docs/design/2026-11-07-combined-services-package.md`,
sections "What happens today" (items 2 and 6), "Procurement"
(Subconsultants), "Edge cases" ("A combined proposal on a split project"),
"Data and migration" (Existing projects) and owner decision 3
(subconsultants are just a group). Slices 1 to 4 are merged on local main
(`adb23c7`). This is the last slice.

## Feature

- **User outcome:**
  - a project that already carries a "Services Engineer" row beside the
    separate services rows, or a row marked "engaged by the lead
    consultant", sees one plain prompt on Procurement offering to combine
    them. The members are ticked; the PM can untick, then confirms with the
    existing Combine confirmation. Dismiss hides the prompt on this project
    for good. The app never combines on its own;
  - a combined proposal ("Services Engineer") filed on a project whose
    services are split no longer makes a fifth row. It waits as unplaced,
    with **Combine for this proposal** (the PM ticks the members; nothing
    is pre-ticked) or **Place on one discipline**;
  - "engaged by" is retired as a setting. Existing values keep their RFP
    wording until the PM combines, when the value is cleared.
- **Why:** answer trust. T03 carried five services fee lines for four
  services because the combined proposal made its own row. Existing
  subconsultant rows count their fee twice. Both now have a one-click way
  into a group, and no new duplicate row is made.
- **Lane:** A (procurement packages, RFP wording).

## In this slice

1. **Combine prompts** on the Procurement read (`packages_read`), computed in
   Go from rows already read, no extra query beyond the decisions the read
   already loads (`internal/procurement/combine_prompts.go`, new):
   - **services:** a live, ungrouped "Services Engineer" row
     (`consultant.services`) and at least one other live, ungrouped row for
     mechanical, electrical, hydraulic or fire services. Members ticked: the
     Services Engineer row and those rows;
   - **engaged by:** one or more live, ungrouped rows with
     `engaged_by = lead_consultant`, and a live, ungrouped lead row (the
     discipline whose RFP role names its subconsultants: architecture
     today, from the knowledge file, not a hard-coded id). Members ticked:
     the lead and its subconsultants;
   - a prompt needs two or more members; grouped, retired or awarded rows
     are never offered (Combine would refuse an award anyway);
   - each prompt has a stable key (`combine:services`,
     `combine:engaged_by`). **Dismiss** records it in the existing
     `proposal_decisions` table as a dismissed decision, the way a
     dismissed package suggestion is kept. No migration;
   - **Combine** from the prompt is the existing combine route with the
     ticked ids; the existing confirmation shows the fee lines that move.
2. **A combined proposal on a split project:**
   - `EnsureProposalPackage` makes no "Services Engineer" row when any live
     row (grouped or not) already holds mechanical, electrical, hydraulic or
     fire services. A project with none of them still gets the row, as
     today;
   - the proposal reader, for a combined services filing that no candidate
     serves (no group covers it), asks Jev no package question: it would
     have to guess one discipline. The proposal stays unplaced. Still one
     fan-out for the rest of the reading;
   - the Procurement board marks such an unplaced proposal `combined`;
   - **Combine for this proposal:** the combine route takes an optional
     `submission_id`. In the one combine transaction the proposal is placed
     on the new group as the PM's choice (the same placement and user
     decision as moving a proposal by hand). Refused when the proposal is
     another project's or org's, already placed, or awarded;
   - **Place on one discipline** is the existing package choice for an
     unplaced proposal, relabelled for a combined one.
3. **Retire "engaged by":**
   - creating a package with `engaged_by`, or a patch that sets it to any
     value, is refused (422) with "Engaged by is retired: combine the
     consultant with its lead on Procurement instead." A patch clearing it
     to blank is allowed;
   - the column, its constraints and every stored value stay. No data is
     dropped. The RFP keeps printing the lead's subconsultant sentence and
     the design team table from stored values, unchanged;
   - combine clears `engaged_by` on every member in the same transaction,
     so the group's RFP (slice 2) is the one statement of who is engaged;
   - the web client's `engagedBy` write helper is removed. There is no
     control on screen today (Procurement stopped asking, owner decision 4
     of 21 October 2026); the browser test pins that none appears.
4. **Tests changed only where a write is now refused:**
   - `internal/store/package_engaged_test.go`: setting a value is refused;
     clearing works; a stored value (seeded directly, as old data) is kept
     by unrelated edits, prints in the lead's RFP and is cleared by combine;
   - `internal/store/report_settings_test.go`: its fixture no longer
     creates a package with `engaged_by = client` (the value printed the
     same as blank, so the RFP it builds is unchanged);
   - `internal/rfp/house_test.go` builds drivers in memory and still passes
     unchanged. Standalone RFP goldens stay byte-identical.
5. **Benchmark:** `packages_read` with prompts in place keeps its budget
   (p50 ≤ 100 ms, p90 ≤ 250 ms). Combine for this proposal is timed as
   `packages_combine` (p50 ≤ 100 ms, p90 ≤ 250 ms) beside the existing
   combine samples.

## Out of scope

- dropping the `engaged_by` column or rewriting old values;
- restoring `engaged_by` on uncombine (the group was the PM's statement);
- re-reading a proposal's prices after the PM places it (as for any
  proposal placed by hand today);
- combining automatically, or guessing members for a combined proposal;
- prompts for other combined filings than services (no other combined
  discipline exists in the filing taxonomy);
- relabelling a group's cost-plan line on rename (known gap, slice 3).

No Jev call in prompts, dismiss, combine or placement. The reader asks one
question fewer for a combined filing with no group. No new dependency. Every
query scoped by `org_id` and project.

## Done

- Go tests:
  - prompts (pure): services row with split services; services row alone
    (no prompt); engaged-by sub with a lead; sub without a lead (no
    prompt); grouped, retired rows left out; dismissed keys hidden;
  - store: prompts on the overview, dismiss remembered, wrong org and
    project refused;
  - a Services Engineer filing on a split project makes no row and stays
    unplaced; on a project with no services rows the row is made as today;
  - the board marks the combined unplaced proposal;
  - combine for a proposal places it on the group; refused for a placed,
    awarded, wrong-project or wrong-org proposal, and nothing is combined
    when refused;
  - engaged by: set refused, clear allowed, stored value kept and printed,
    cleared by combine;
  - API: combine with `submission_id`, dismiss route, wrong org.
- A browser test: the prompt shows, Combine makes the group; a combined
  unplaced proposal offers Combine for this proposal, which makes the group
  and places it; no "engaged by" control on Procurement.
- Go suite and web build pass (failures that pre-date this work aside). The
  bench shows `packages_read` and `packages_combine` within budget.

## Edge cases

- A prompt whose members change (one removed or combined elsewhere): the
  prompt is recomputed from live rows on every read; with fewer than two
  members it is gone.
- Dismiss then a new Services Engineer row later: the prompt stays
  dismissed on this project (the owner asked for "remembered").
- Combine from a prompt racing a combine elsewhere: the existing combine
  refuses an already grouped row (409).
- A combined proposal dropped by the PM on a row: the drop is the PM's
  choice and places it, as today.
- An `engaged_by = client` value: printed the same as blank; it is cleared
  on combine like any other value.
