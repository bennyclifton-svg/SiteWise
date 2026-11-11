# Accommodation schedule as lines

Owner request, 11 Nov 2026, after the first Accommodation table
(`2026-11-11-accommodation-and-full-capture.md`): move it below the
Consultant team; columns Space | Level | Area | Comment, no quantity; one
line per room ("Office 1", "Office 2"); typical spaces for the building type
shown as placeholders; spaces the documents name flagged and filled; the
whole table editable with the selection and right-click menu used elsewhere.
Return brief agreed the same day ("that is a good please proceed"), with the
four recommendations below.

## Outcome

- The last section of the Profile, below Consultant team.
- Columns Space | Level | Area (m²) | Comment. Each cell shows how it was
  decided: green from the documents, blue typical, rust the user's, amber to
  check, red where documents disagree.
- One line per room. Settings and fixtures (workstations, seats, WCs,
  showers, car spaces) are a count in the comment, not lines.
- Above 30 rooms of one kind, one line with the number in the comment;
  "Split into lines" makes the numbered lines.
- An area is never divided. A per-room area fills each line; an area for a
  group goes in the comment ("140 m² for all 6"). A level stated for the
  group shows on each line as written.
- Placeholders: typical spaces for the building type on the card, names
  only, blue. A fit-out or extension gets the tenancy spaces; a new building
  also gets base-building spaces. A deleted placeholder stays deleted for the
  project. Untouched placeholders follow a change of building type.
- Houses: Bedroom and Bathroom lines follow the card's counts; the card
  stays the count RFPs use, and a difference shows in words under the table.
- Comments: capacity, setting counts and requirement phrases ("AV-ready",
  "adjacent to dining") that Jev confirms in the passage's own request.
- The user's word is final: a typed cell is never overwritten by a reading;
  a deleted line stays deleted; when the documents later list fewer rooms,
  the extra lines are flagged in words, not removed.
- Editing as the document repository and the RFP scope table: click, Ctrl/Cmd
  toggle, Shift range, Ctrl/Cmd+A; double-click or F2 edits; right-click or
  ⋮ gives Insert above / below, Duplicate, Split into lines, Set level, Reset
  to documents, Delete; the Delete key; drag to reorder; a grey row adds a
  line.

Serves time-to-decision and answer trust: briefs for the architect,
services, fire and cost consultants start from the accommodation, and the
PM no longer retypes it.

## Lane

A (profile values a user trusts). Existing profile, knowledge, store and API
tests pass; the Meridian answer key stays at or above its floor of 70.

## Design

**Catalogue** (`knowledge/profile/spaces.yaml`, version 2, draft). Each
space gains `line` (a line's name, singular), `kind` (room or setting),
`unit` (the plural used in a comment), and optionally `count_from` (a card
count: bedrooms, bathrooms, garage spaces, beds, rooms). New spaces cover
the typical lists (pantry, print and copy, sick bay, canteen, playrooms,
sleep room, nappy change, nurse station, mail room, communal open space,
loading dock and others). A requirement list (`requirements`) names the
phrases a comment may carry (AV-ready, acoustic separation, secure,
natural light, sink, operable wall, dedicated cooling, compactus).
`schedules` lists the typical spaces per building type (subclass, with a
class fallback) with their sources:

- schools: NSW Educational Facilities Standards and Guidelines,
  accommodation schedules
  (https://education.nsw.gov.au/about-us/efsg/design-components/new-primary-school-builds/large/accommodation-schedule.html,
  https://efsg.det.nsw.edu.au/primary/acc.html)
- early learning: Education and Care Services National Regulations
  regs 106 to 115 by heading, and the Education Standards Board (SA) guide
  "Creating safe facilities"
  (https://www.esb.sa.gov.au/creating-safe-facilities-guide-building-and-renovating-education-and-care-services)
- health: Australasian Health Facility Guidelines, health planning units and
  their schedules of accommodation (https://aushfg.com.au)
- aged care: National Aged Care Design Principles and Guidelines
  (https://www.health.gov.au/sites/default/files/2025-03/national_aged_care_design_principles_and_guidelines_summary.pdf)
- apartments: NSW Apartment Design Guide (common spaces only; dwellings are
  the card's unit mix)
- offices, retail, food and beverage, hotels, industrial, data centres,
  places of worship: practice, marked `basis: practice`.

No area or rule number comes from these sources here; a minimum set by an
instrument would come later as a rule with `verified: false`.

**Reading** (no new call, no new round trip). The space questions stay as
they are, in the passage's label fan-out, on brief-like passages only
(https://docs.typesafe.ai/patterns/fan-out). Additions, each asked only
where code finds its words on the space's clause:

- `space.<id>.need.<requirement>`: yes / no / not_stated
  (https://docs.typesafe.ai/primitives/choice).
- `space.<id>.next_to.<other>`: "kitchen adjacent to the dining room",
  yes / no / not_stated.
- `space.<id>.area_each_sqm`: an area written per room ("15 m² each").
- `space.<id>.mix`: a list of room sizes ("4 × 8-person, 2 × 16-person"),
  offered pre-parsed as one candidate
  (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).

`QuestionVersion` profile-16, `ReadRevision` read-16.

**Lines** (code, `internal/profile/accommodation.go`). Built on every
profile read from the readings, the placeholders, the card's counts and the
stored edits; nothing is judged while the user waits. A generated line's id
is `<space>#<n>`, so a placeholder the documents confirm is the same line,
and an edit stays with its line across readings.

**Storage** (`accommodation_lines`, migration 054, every row scoped to its
org and project). A row holds the user's edits to a generated line (or a
line the user added), which fields the user set, its position, whether it
is split, and whether it was removed.

**API.** The profile read carries `accommodation` (lines and notices).
`POST /api/projects/{id}/profile/accommodation` takes one operation (set,
reset, delete, insert, duplicate, split, order) on a list of line ids and
returns the table. A wrong org or project is 404; a bad operation, field
or value is 422.

## Budget

`project_profile_read` stays p50 ≤ 50 ms, p90 ≤ 150 ms. New path
`accommodation_edit` takes the profile edit budget, p50 ≤ 50 ms,
p90 ≤ 150 ms, with a benchmark. No Jev call on read or edit. The Update's
added questions are only where a requirement, adjacency, per-room area or
size list is written (Meridian before this change: 44 space questions of
1,999, in 6 of 68 requests).

## Out of scope

A total-area row; use of the schedule in RFPs, the Cost Plan or the
Program; drawing room tags; per-level subtotals; NCC sanitary facility
counts from the occupant load (a good next step); undo.

## Edge cases

- No building type yet: no placeholders, the grey add row only.
- Documents say a space is not required: its placeholder is hidden.
- A count the documents disagree on: one line, the comment says the
  numbers ("6 or 8 in the documents").
- Count above 30: one line until split; a split is capped at 500.
- A renamed, edited line whose space later leaves the documents and the
  placeholders: kept, flagged in words.
- Infrastructure: no placeholders.

## Done

Knowledge loads and checks; harvest and line-builder tests; store and API
tests (wrong org, bad input, missing line); Meridian re-read with its lines
checked against the documents; benchmark within budget; web build; browser
look on desktop and phone.

## Result (11 Nov 2026, built, uncommitted on main)

- Live Update of Fitout Meridian (profile-16, read-16): 5 of 5 documents,
  none failed. The schedule reads Reception (Level 3, secure), Open-plan
  workspace (42 workstations), Office 1 to 8 (Level 3, Level 4), Meeting
  room 1 to 4 (capacity 8, AV-ready) and 5 to 6 (capacity 16, AV-ready),
  Library (180 m²), Records room (secure), Breakout (capacity 60), Kitchen,
  Kitchenette, Toilets (4 WCs), Accessible WC, Showers (Level 4) and Comms
  room from the documents; Boardroom, Quiet room, Print and copy and Store
  room as blue placeholders of an office fit-out.
- Meridian answer key (draft) extended with kitchenette, the room-size list
  and three requirement phrases; fixture re-exported from this read (the
  profile-15 export kept beside it): 73/78, floor 73. All the schedule's
  entries are right. New miss: the live-building condition went red on
  this read (Jev read "partially occupied building" as partial occupation
  once; every question was asked again under profile-16). Still wrong, as
  before: `sig.acoustic-separation-stated`.
- Timing (replayed bench, dev machine): `accommodation_edit` p50 8.9 ms,
  p90 10.0 ms (budget 50/150). `project_profile_read` p50 1.0 ms, p90
  11.1 ms. The schedule adds about 0.5 ms to each profile response (one
  query and a 0.13 ms build) and 3 to 5 KB. `spec_home_profile_edit` read
  59.2, 52.6 and 50.9 ms p50 over three runs against its 50 ms budget; it
  was 49.8 ms on 6 Nov with this addition absent, so the path has no
  headroom and needs its own look. `project_delete` reads about 950 ms
  (59 ms on 6 Nov); its time goes in deletes this change does not touch.
- Not done: totals, use in RFPs, the Cost Plan and the Program, drawing room
  tags, NCC sanitary counts from the occupant load.
