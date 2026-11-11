# Accommodation table and full capture of a fit-out brief

Owner request, 11 Nov 2026, after reviewing Fitout Meridian (five Markdown
documents): the profile missed the brief's rooms and areas, and more. Owner:
"the accommodation table should also be used in other projects ... make sure
all details are captured in the profile, proceed with changes." The owner
also pointed out that a year-built field already exists
(`det.existing_building_year`); it missed Meridian's "building constructed
1984" only because its trigger needs "constructed in".

## Outcome

1. **Accommodation**, a new Profile section between the Project card and the
   Scope of works: one row per space the building provides after the works,
   with Space | Qty | Area m² | Capacity | Level. It works for any building
   type (offices, schools, houses, clinics) from one catalogue of space types.
   Each cell is an ordinary profile value: sources, conflicts, origin colour,
   the user's word final. The user adds a space from the catalogue and removes
   one with the bin.
2. Every fact in the Meridian documents that the profile has a field for is
   read, and the facts it had no field for get one (list below).

Serves answer trust and time-to-decision: the architect, services, fire and
cost briefs all start from the accommodation and the occupant load, and today
the PM retypes them.

## Lane

A (profile values a user trusts). Existing profile, knowledge and API tests
must still pass. The Meridian answer key was already failing before this
work (40/46, `sig.acoustic-separation-stated` read false); its floor stays.

## How accommodation is read

- Knowledge: `knowledge/profile/spaces.yaml`, a draft catalogue of space types
  (id, label, name pattern). Bedrooms, bathrooms, car spaces, units and
  tenancies stay on the card's scale fields and are not spaces, so no fact
  is held twice.
- Keys: `space.<id>.provided` (yes/no), `.count`, `.area_sqm`, `.capacity`,
  `.level`, project scope.
- Code harvests per line of a passage's own text: every space name, then
  every count, area (m²), capacity ("8-person", "60-seat") and level
  ("Level 3", "Ground floor") on that line, each given to the nearest space
  name. A table row's first cell owns its row's list of "N × …" items: code
  offers their sum as one more count candidate ("4 × 8-person, 2 × 16-person"
  is 6) when no other space name sits inside the list. Several levels for one
  space are also offered joined ("Level 3, Level 4").
- Jev, in the existing label fan-out (no new call, no new round trip,
  https://docs.typesafe.ai/patterns/fan-out): for each space named, "does the
  passage state this project provides it?" (yes / no / not_stated), and for
  each attribute with candidates, which candidate or none
  (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
  Asked only on passages that get the project questions (brief-like
  passages, `HeaderRouted`), so a specification's room references do not
  fill the table.
- A space shows when the user added it or a document reading says it is
  provided or gives it a value; the user's "no" removes it.

## Other facts captured

| Fact in the documents | Where it goes |
|---|---|
| "building constructed 1984" | `det.existing_building_year`: trigger also takes "built/constructed 1984" |
| Occupancy 98 → 142 persons | `det.occupant_load` joins the profile (classification group) with triggers and person-count candidates; Jev picks the design (proposed) load |
| "Levels 3 & partial Level 4" | new detail `proj.works_levels`, Floors in the works |
| Level 4 mezzanine ~185 m² | office gains `added_gfa_sqm`; a mezzanine counts as added floor area unless "existing" is nearer |
| Possession 1 Nov 2026 | `proj.target_construction_start` also triggers on "possession" |
| SSD lodgement September 2026 | `proj.target_da_lodgement` also triggers on SSD/CDC lodgement |
| Occupation 1 July 2027 | new detail `proj.target_completion` |
| 22–26 weeks on site | `proj.preliminary_construction_months` also offers week counts |
| Feasibility allowance $1.35–1.55M (not confirmed) | new detail `proj.budget_aspiration`; never the construction budget, so the cost plan does not spread it |
| Landlord HVAC contribution $180k | new detail `proj.contributions_stated` |
| Westgate Properties (landlord) | new detail `proj.landlord` |
| After-hours window | new detail `proj.work_hours` |
| No works before SSD consent and the landlord's deed | new detail `proj.start_conditions` |
| Two invited builders | new detail `proj.tenderers` |
| Partially occupied building | operational-constraint wording: other floors still occupied is a live environment; vacant possession of the works floors alone is not "vacant" |
| Tenant with a landlord | new stakeholder option `landlord_tenant` |
| Partitions, ceilings, finishes, lighting, switchboard, signage, smoke control, slab penetrations, acoustic partitions, mezzanine, kitchenette, shower, WC | implied-scope words (`scope_defaults.yaml implied_by`): suggestions only, never evidence, the user's out is final |

Not added (stated in the report to the owner): HVAC design temperatures and
air rates, the SSD and CC assessment periods, the rent-free period, ROM
assumptions (asbestos), the architect's fee (Procurement reads appointment
letters), and who provides sprinklers and exits as between landlord and
tenant (the landlord stays a user choice, owner ruling 2026-10-24).

## Budget

No new path and no new Jev call. Profile read stays at
`project_profile_read` in `bench/budgets.json`; the added questions ride the
label fan-out of passages already read. Filing is untouched (profile reading
waits for Update). Profile writes reuse the existing value route.

## Versions

`QuestionVersion` profile-15 (operational and stakeholder wording, spaces);
`ReadRevision` read-15, so finished documents are read again on the next
Update. Passages whose questions did not change answer from the call cache.

## Out of scope

Free-text space types not in the catalogue, per-level splits of one space
type, a total floor area from the table, RFP, cost plan and program use of
the accommodation (next step), and drawing room tags.

## Edge cases

- A space named with no number ("Kitchenette, breakout, records and comms
  rooms"): one provided question, a row with Qty unknown.
- An existing space demolished or a consultant's own office: not_stated.
- "Ground-floor lobby presence not required": provided no.
- Two documents with different counts: a conflict, as on the card.
- Unknown stays unknown: no count is inferred from an area or a capacity.

## Done

Harvest tests on the Meridian lines, knowledge load and key-scope tests, API
listing and validation tests, the profile, knowledge and httpapi suites,
`check_knowledge.py`, a web build, one live re-read of Fitout Meridian with
the captured values checked against the documents, and a browser look.

## Result (11 Nov 2026, built, uncommitted on main)

- Live re-read of Fitout Meridian (profile-15): 5 of 5 documents, none
  failed. The table reads 42 workstations, 8 enclosed offices on Level 3 and
  Level 4, 6 meeting rooms, a 180 m² library, a 60-seat breakout, a
  kitchenette, records and comms rooms (1 each), 4 toilets, 1 accessible
  toilet, reception on Level 3 and a shower on Level 4.
- Also read: year built 1984, occupant load 142, added GFA 185 m², floors
  "Level 3 and part of Level 4", live environment, tenant and landlord,
  Westgate Properties, all three targets, 22–26 weeks, the $1.35M–$1.55M
  feasibility allowance, the $180,000 contribution, permitted hours, the
  start condition and two tenderers. Scope gains lighting, switchboards,
  power circuits, floor finishes, painting, signage, smoke control, slab
  penetrations, acoustic separation and the mezzanine's structure, fire
  separation and stair, as suggestions.
- Answer key (draft) extended with these values; private fixture
  re-exported (the 2026-10-24 one kept beside it): 70/74, floor raised from
  38 to 70. Still wrong, as before this work: `sig.acoustic-separation-stated`.
  Missed: procurement route (Jev now answers not_stated; the evidence is
  thin), switchboard and smoke-control presence.
- Timing: `project_profile_read` p50 4.7 ms, p90 6.4 ms; `profile_edit`
  p50 10.8 ms, p90 11.8 ms (budgets 50/150 ms). No new Jev call.
- Two merges added to reconciliation, scoped to their fields: floor runs
  inside a wider run are one answer (`proj.works_levels`), and the same sums
  written two ways are one contribution (`proj.contributions_stated`).
- Failing before and after this work, unrelated: TestProgramRoutes,
  TestCommittedBudgets (81 budget paths), the Program engine golden tests in
  web/src/program.
