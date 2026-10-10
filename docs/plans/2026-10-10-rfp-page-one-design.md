# RFP page one: the project, then the appointment

Agreed with the owner on 10 October 2026, reviewing the Petersham
architecture RFP. Design only; nothing is built.

**Outcome.** A consultant RFP reads as a document, not a form with a document
under it. Page one is the project, the same on every RFP; page two onward is
the appointment. Each fact appears once, and the user corrects it where it
prints. The RFP stays within the two-to-three-page limit.

**Why.** Answer trust and time to decision. Today the Project section prints
a Project/Detail table and, behind an Edit toggle, a second list of the same
facts in different words. A "Decide before issue" banner and a "Draft uses
saved project information" block sit above the document and push it down.
Automation fills about 80% of an RFP; the user polishes the rest, and that is
quickest when the correction happens on the printed value.

## Mental model

- **Profile** holds every attribute ever stored for the project.
- **RFP page one, "1. Project"**, shows the project in the same groups and
  order as the Profile card. It is shared by every consultant RFP and later
  by the head-contract RFT and trade tenders.
- **RFP page two onward** is the discipline: role, scope by stage,
  allowances, positions, terms, evaluation, returnables. Each discipline's
  subset of attributes is refined later, discipline by discipline.
- A project fact edited in the RFP writes to its home in Profile (owner
  decision D1, unchanged; `internal/rfp/homes.go`).

## Page one, "1. Project"

Always open. No Edit toggle, no separate "Project facts / This appointment"
panel. **The printed page is the form**: clicking a value edits it in place.

In order:

1. **Issue strip.** One line: Ref · Issued · Closes · Contact. Printed only;
   edited in the closing Issue section.
2. **Opening sentence.** "{client} intends to {work} {parts} at {address}."
3. **Fact grid.** Three columns, matching the Profile card's columns
   (`web/src/Profile.tsx`, `COLUMNS`):

   | The building | Site and planning | Delivery |
   |---|---|---|
   | Parts and NCC classes | Site address | Client |
   | Work type | Lot and DP | Procurement route |
   | Storeys, GFA | LGA, zone | Program milestones |
   | Units, car parks | Site area | Sustainability target |
   | Existing building | Heritage | Regulated design applies |
   | | Planning pathway | Construction budget (internal) |

   The construction budget shows on screen and prints only when the user
   discloses it at issue, as today.
4. **Site conditions.** Every flag this discipline uses, three per line,
   **each printed with its answer** (Yes, No or Not known). On screen each
   answer is a Yes / No / Not known control. These stay on page one by owner
   choice, so page one's conditions differ by discipline while the rest of the
   page is identical.
5. **Information provided.** One line.

Removed from page one: the Project/Detail table (the grid replaces it), the
Edit toggle, and "Parts this appointment covers" (moved to section 2).

## Page two onward

| Section | Change |
|---|---|
| 2. Your role | Gains **Parts this appointment covers** and **This appointment buys: stage X to Y** (moved from the stages editor). |
| 3. Scope by stage | Loses the stage-range picker. Service level stays. |
| 4. Allowances, 5. Scope position | No change. |
| 6. Terms | Gains the commercial terms as in-place values: client agreement, payment days, proposal validity, rates hold until, PI and PL. |
| 7. Evaluation, Returnables | No change. |
| **Issue** (new, last, screen only) | Issue details, budget disclosure, the stale-draft safeguard and the Issue button. |

Sections 3 to 7 keep their Edit toggles in this change; in-place editing for
them is a follow-up once page one has been used.

### The closing Issue section

Compact grid, screen only, never printed:

```
ISSUE                                     2 items block issue  [Show first]
Ref [ ___ ]   Issued [ date ]   Closes [ date ]   Questions by [ date ]
Contact [ __________ ]          Start by [ date ]
☐ Include the internal budget                       [ Issue report ]
Architecture rules are a draft: issue waits for owner review.
```

It replaces the toolbar's Issue report button and form. Issuing a stale draft
still needs the existing tick and a reason.

## Removed blocks

- **"Decide before issue."** Each blocking check shows as a small red marker
  beside the value it concerns (stage range in section 2, a missing project
  detail on its page-one value, an include/exclude conflict on its row in
  section 5, and so on). The count and the draft-rules note move to the Issue
  section. Markers are screen only. Blocking rules are unchanged.
- **"Draft uses saved project information."** The toolbar button reads
  "Refresh: Profile changed" (reason in its tooltip) while the draft is stale.

## Common UI pattern

The start of a consistent, professional look across the app:

- A fact is a small muted label above its value, the row pattern the Profile
  card already uses; the RFP reuses it rather than copying it.
- On screen only, a value carries a source pill: Profile · You · Costs ·
  Not known, and red for Sources disagree.
- The print stylesheet and the PDF carry no pills, controls or hover states.

## Build

Two slices, one lane each.

**Slice 1, lane A (what the RFP prints).**

- `internal/rfp/sections.go` `project()`: replace the Project/Detail table
  with two document tables, the fact grid (six columns: label and value three
  times) and the conditions grid. This reuses the pairing `allowances()`
  already uses, so `internal/reports/render/pdf.go` needs no new layout. Each
  cell's fact key goes in the block basis so the screen can attach its
  control.
- `role()`: add parts covered and the stage range.
- `knowledge/rfp/skeleton.yaml`: `project_rows` drops Building, Quantities,
  Planning pathway, Procurement and Site conditions; agreement, payment days,
  validity, rates hold until, PI and PL move from group `issue` to `terms`.
- `internal/rfp/controls.go`: groups follow the new placement; checks carry
  the section their marker sits in.
- No Jev call, no new query. Budgets `report_setting` (p50 100 ms, p90
  250 ms) and `report_fact` (p50 150 ms, p90 400 ms) must still pass.

**Slice 2, lane B (the screen).**

- `web/src/Reports.tsx`: page one always open with in-place controls on grid
  cells; check markers in their sections; the closing Issue section; the
  stale label on Refresh; `RfpDecisions` removed.
- `web/src/RfpEditor.tsx`: the Profile card's field-row pattern.

## Done when

- Go tests cover the page-one tables, the role additions and the regrouped
  terms; every discipline's golden tests still pass.
- Page fit stays within the limit on the Petersham architecture RFP and on a
  heavy discipline (civil).
- Playwright: editing a page-one value updates Profile; a marker jumps to its
  control; issue stays blocked until the markers clear; the printed PDF has no
  pills or controls.
- Speed budgets above pass.

## Edge cases

- A fact with no home that can be written (budget, derived mixed use,
  heritage link) shows its value with a link to where it changes, not a
  control.
- Sources disagree: the value shows red with both readings; the marker counts
  as blocking only where it does today.
- A discipline that uses no flags prints no Site conditions grid.
- An empty essential value prints "___" as today and shows a marker on screen.

## Out of scope

In-place editing for sections 3 to 7; moving the RFT (`tpl.rft`) to the
shared page one; changing which flags each discipline uses; any change to
document reading.

## As built (10 October 2026)

Built on branch `rfp/page-one` in two slices (plans `...-slice1.md`,
`...-slice2.md`). Where the build departs from the text above, and why:

- **Print layout, to hold the three-page limit.** The scenario RFPs already
  reached 3.00 pages, so page one could not grow. The grid prints as three
  columns of "Label: value" cells (a six-column label/value table wrapped
  too much), and the conditions print as one line per answer ("Conditions
  that apply: …", "do not apply", "not yet known", "to be confirmed") rather
  than a table. Every answer still prints. Across 137 scenario RFPs the
  longest is 2.98 pages and the mean length is unchanged.
- **Heading.** "Project conditions" on screen: several are not about the
  site (staging, consultation, contract administration).
- **Heritage and existing building** print as conditions, not grid rows:
  they are flags, and printing them twice would repeat them.
- **Scope-decision flags** ("Investigation selected" and the like) are the
  client's choices for one appointment: their controls sit in section 2 and
  they never print as conditions.
- **Discipline rows.** A discipline's procurement wording rewords the grid
  cell in place; its other project rows (design vehicle, assessment basis)
  print at the end of section 2. Town planning's site row and
  architecture's sustainability row were deleted as repeats of the grid.
- **Unset rows** do not print but are offered on screen behind "+ N more"
  in their column, as on the Profile card.
- **Stale draft.** Refresh keeps its name; every stale reason shows beside
  it as one compact line (a renamed button broke nothing but would have
  hidden which source changed).
- **Section 2** opens with an Edit toggle like sections 3 to 7; its markers
  show under the heading while it is closed.
