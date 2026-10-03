# Project profile and a one-line register

Date: 3 October 2026. Owner: Benny Clifton. Status: agreed in design session;
not built. Plan: `docs/plans/2026-10-03-project-profile.md`.

Read `docs/design/2026-09-29-foundation-design.md` and `knowledge/SCHEMA.md`
first. Everything here keeps their non-negotiables: Jev is the only AI, code
parses, computes and controls flow, and every user-facing path has a budget.

## 1. What we are building

Two changes to the project page:

1. **A project profile** that SiteWise drafts from the documents a project
   manager has already filed, and the user then polishes. It replaces the
   project profile in Clerk, keeping its header and adding the building model:
   a **systems checklist** and a **compliance** section.
2. **A one-line document register.** Today each filing is a title-block tile.
   It becomes one compact schedule row per document, about a third of the page
   wide, like the Clerk register.

Page layout (owner decision, 3 October): a left nav with the wordmark and a
project switcher (after the Clerk project nav), the profile in the middle, and
the register in the right third. Below 1100 px the register stacks under the
profile; below 760 px the nav becomes a top row. Dropping files anywhere on the
page still files them.

## 2. Two workflows, one screen

Both workflows start the same way: upload everything and let instant intake
classify it. The profile is then built in the background from filed documents.

| Workflow | What the user has | What fills the profile |
|---|---|---|
| Thin brief | A short brief, an email, little else | The user picks class, subclass and work type. Code fills a typical systems list for that type, shown as *suggested*. |
| Rich documents | Briefs, specs, reports, quotes | Jev picks from the evidence; code reconciles across documents. Evidence replaces suggestions and links to its document and passage. |

**Accuracy target.** About 80% of keyed values right with no user action, and
no wrong green. The user corrects the remaining 20% by ticking, unticking,
choosing from a list or editing a line. A user value is final and is never
overwritten by a later document; a later conflicting document raises a
visible conflict instead.

## 3. Profile shape

### A. Project (Clerk header, copied as data)

From `../clerk/data/taxonomy/` (data, not code): building class, subclass,
work type, the subclass's scale fields (site area, GFA/GLA/NLA, storeys,
units, bedrooms, car spaces and so on), and these project conditions:
planning pathway, procurement route, access constraints, operational
constraints (live environment), stakeholder complexity and environmental
sensitivity. Option labels lose Clerk's cost-uplift text (`+10-20%`).

New project facts (not building systems): planning consent number, consent
date and lapse date, contract form (AS 4902, AS 4000, AS 2124, ABIC, HIA/MBA
domestic, other), contract basis (lump sum, cost plus, other), defects
liability period, design life, certifier.

Where Clerk and the building model both describe a site condition, **the
determinant wins and Clerk's field is not asked**: bushfire (`bal`,
`bushfire_prone_land`), flood (`flood_hazard`, `flood_planning_level`),
heritage (`heritage_status`) and contamination (`potentially_contaminated_land`).

Every area records its measurement basis (GFA per the Standard Instrument,
GLA per PCA, NLA, site area). Hale states GLA; Petersham states GFA.

### B. Parts

Facts belong to a building, part, storey, compartment or tenancy (SCHEMA
"Determinant evidence contract"). The profile keeps a small parts list per
project. v1 starts with one part, "Whole project"; the user adds parts
(Petersham: Building A, Building B, basement car park; Mornington: dwelling,
pavilion, pool). A fact goes to a part only when its passage names that part's
label exactly (code string match); otherwise it stays on "Whole project". Two
different values on one part show as a conflict, which is how the user learns
to split parts.

### C. Systems checklist

One row per leaf system from `knowledge/`, grouped under the 13 top-level
systems. Each row has:

- **Status:** Included, Not included, or Unknown (tri-state control).
- **Provider:** Contractor, Owner, Others, or not stated.
- **Note:** one line. Either the user's text or a verbatim excerpt (at most
  120 characters) copied by code from the source passage. Never generated.
- **Provenance:** band mark plus a link to the document and passage.

Default view shows rows that are included, stated as not included, conflicted
or suggested. "Show all systems" reveals the rest. "Not applicable", "not
required", "no allowance" and "not included" are evidence of absence, not
silence.

### D. Compliance

Determinants grouped as Classification, Site, Services and Fire. Each shows
its value, part, **assertion** (see 5), band and source. When nothing states
it, the row says which kind of document usually does ("usually in: geotech
report"), from the determinant's `stated_in`. Derived values come only from
verified tables: today, Type of Construction and compartment limits. The
evaluator (`internal/knowledge/evaluate.go`, `Derive`) also requires the
owner to have marked the rule and the table `reviewed`; until then they show
"Awaiting owner review". Every other derivation shows "Pending: table not
verified", never a seed number. Derivations use only facts asserted as
`stated` or set by the user. A `required` or `allowance` value is shown but
never used as an input.

## 4. Where Jev is used, and where it is not

All profile Jev work is **background**, after filing. Opening or editing the
profile never waits on Jev. Calls stay at two fan-outs per passage, as today:
labelling, then evidence.

| State (one fan-out) | Added questions | Type | Routing (code) | TypeSafe doc |
|---|---|---|---|---|
| Passage + `candidates` (label call) | One per determinant whose trigger matched: pick a candidate or `none` | choice | Code trigger must match in the passage; no trigger, no question | [Pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook), [intent routing](https://docs.typesafe.ai/patterns/intent-routing) |
| Same call | One assertion question per asked determinant: `stated`, `required`, `allowance`, `not_stated` | choice | Asked only with its determinant | [How to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one) (atomic) |
| Same call | Project header choices (class, subclass, work type, Clerk conditions), each with `not_stated`; scale fields (storeys, areas, units) as candidate picks when their trigger matched | choice | Only passages in a description, outline, scope or introduction section, or the first 12 passages of a brief, specification, report, contract or quote. Scale fields: any passage where their trigger matched | [Fan-out](https://docs.typesafe.ai/patterns/fan-out) |
| Passage with accepted labels (evidence call) | Per labelled leaf: presence `included`, `not_included`, `not_stated` | choice | Only leaves the passage was labelled with | [Fan-out](https://docs.typesafe.ai/patterns/fan-out) |
| Same call | Per labelled leaf: provider `contractor`, `owner`, `others`, `not_stated` | choice | Same | [How to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one) (atomic) |

Code does everything else: harvesting candidates (numbers, areas, BAL and
class tokens, consent and contract references), unit and basis handling,
reconciliation across documents, conflicts, supersession, independence of
sources, derivations from verified tables, type suggestions, and every write.
Jev never counts, compares, converts units or writes text
([jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).

**Confidence and bands** ([confidence](https://docs.typesafe.ai/confidence),
[confidence routing](https://docs.typesafe.ai/patterns/confidence-routing)).
Choice confidence depends on option count, so each question shape gets its
own threshold in `data/profile/thresholds.json`. Presence and provider have a
fixed option count, so one threshold each is legitimate. Before calibration,
green is withheld: Jev picks at or above a provisional amber floor apply as
amber; below it they stay blank. A wrong profile value costs the user a
click, so a provisional amber floor is acceptable here. The intake bands are
not reused.

**Reconciliation rules (code):**

1. A user value wins and is never overwritten.
2. Facts from superseded documents are ignored.
3. Agreement between documents of kind `commercial` (quotes, tenders) shows
   as "consistent across tenders", not as independent proof. They read the
   same tender package. Disagreement between them shows as "tenderers differ".
4. Included and not-included evidence for one system is a conflict (red).
   Show both sources, apply neither.
5. Two values for one determinant on one part is a conflict. The value with
   more independent supporting documents is displayed, marked red, not applied.
6. A type suggestion never outranks evidence.

## 5. Schema audit (3 October 2026)

Method: every scope line, inclusion, exclusion and stated fact in the six
research documents was mapped to `knowledge/`. The documents were: Newham
builder's specification (22 Class 1a dwellings), Mornington's three tenders
(custom house, VIC, BAL-40, septic, tank water), Petersham PPR (5-storey
mixed use, heritage area), Hale design brief (warehouse extension), and
Rutherford building performance brief (2 warehouse buildings, 6 units).

**Sufficient.** 13 top-level systems and 130 leaves cover almost every scope
line, with literal `describes` and boundary-case `excludes`. `ncc_class`
already has 10a, 10b and 10c. The 66 determinants already include BAL,
bushfire-prone land, wind class and region, site class, climate zone,
corrosivity, acid sulfate, termite, noise, NCC edition and compliance pathway.
The verified `compartment_limits` table agrees with Rutherford's brief (Class
7b, Type C: 2,000 m², 12,000 m³), which makes it a good test fixture. Clusters
are an authoring split only; the profile groups by top-level system, so no
cluster reshaping is needed.

**Gaps to fix before building.** Section and page references are to the
research documents.

| # | Gap | Evidence | Change |
|---|---|---|---|
| S1 | No system for fireplaces, solid-fuel heaters, chimneys and flues | Mornington: all three tenders price a wood heater and chimney blockwork | New leaf `mechanical.heating-appliances` (NCC 2022 Vol One Part G2 and Housing Provisions Part 12.4; clause `clause_verified: false` until checked) |
| S2 | Roller shutters, sectional and tilt garage doors, dock doors are not named | Hale §4 Roller Shutter Doors, Rutherford §4, Newham §16, Mornington garage doors | New leaf `envelope.vehicle-doors` |
| S3 | No loading docks or vehicle and MHE impact protection | Hale §4 Recessed Docks and Pedestrian and Asset Protection, Rutherford §4.1 Bollards | New leaves `site.loading-docks`, `site.impact-protection` |
| S4 | No waste storage and collection | Petersham §7.13 | New leaf `site.waste-storage` |
| S5 | No public domain works (council footpaths, kerbs, street trees in the road reserve) | Petersham §7.7, §7.9; Newham §33, §34 | New leaf `site.public-domain` |
| S6 | No non-statutory signage | Petersham §7.21; Rutherford §4.4, §5.6, §8.4, §8.6 | New leaf `site.signage` |
| S7 | No furniture, fittings, equipment, appliances or window furnishings | Newham turnkey inclusions; Mornington owner-supply lists | New leaf `interiors.ffe` |
| S8 | Drinking water from rainwater tanks or a bore falls between `hydraulic.cold-water` (authority main) and `hydraulic.non-potable-water` | Mornington: tank water and filtration in all three tenders | Widen `hydraulic.cold-water` describes to private potable sources and treatment; adjust `non-potable-water` excludes |
| S9 | Battery-charging ventilation, safety showers and eyewash are not named | Hale §4 MHE Charging Area | Widen `mechanical.local-exhaust` and `hydraulic.specialist-process` describes |
| S10 | `envelope.bushfire-construction` and `fire-passive.bushfire-construction` are duplicates with the same label | Both describe AS 3959 construction | Deprecate the fire-passive one with `replaced_by`, move its references |
| D1 | No way to say how a value is asserted | Newham "allowed for BAL-LOW" is a pricing allowance; Rutherford "compartments must remain under 2,000 m²" is a requirement; Rutherford "sprinklers … will not be required" is an expectation | Assertion question on every extracted determinant: `stated`, `required`, `allowance`, `not_stated` |
| D2 | No water supply source | Mornington tanks; Newham mains plus recycled; Hale town water | New determinant `water_supply_source` |
| D3 | No sewer connection type | Mornington septic; Newham "no pressure sewer" | New determinant `sewer_connection` |
| D4 | No gas supply | Newham mains gas assumed; Mornington LPG; Hale not applicable | New determinant `gas_supply` |
| D5 | No saline soil | Newham salinity and site exposure report | New determinant `saline_soil` |
| D6 | No storage height | Hale "min. top of storage height"; ESFR sprinklers | New determinant `storage_height` |
| D7 | `heritage` is boolean but documents state the kind | Petersham "heritage conservation area C5" | New choice determinant `heritage_status`; deprecate `heritage` |
| D8 | Determinant questions route by system labels, but determinants sit in BCA, geotech and bushfire reports and planning certificates, often in passages with no system label | None of the six documents states NCC class, rise, effective height, climate zone, wind class or site class; Newham only refers to its geotech report | Add `triggers` (code patterns that gate the question) and `stated_in` (document kinds) to determinants |
| D9 | No home for project facts | Consent numbers and lapse dates, contract forms, defects periods in Petersham, Rutherford and the Mornington tenders | New `knowledge/profile/` data: taxonomy copied from Clerk plus `project_facts.yaml` |
| D10 | Scope exists in the evidence contract but nothing declares a project's parts | Petersham A and B; Rutherford 2 buildings × 3 units; Mornington pavilion and pool | Runtime parts table (section 3B) |
| D11 | No source type for records drafted from project documents rather than seeds | Bollards, salinity and eyewash have no seed heading | Allow `sources: [{document: <eval id>, anchor: <exact line>}]`, checked against `data/eval/profile/manifest.json` |

**Not systems.** Outbuildings, pavilions, carports, pools and mezzanines are
parts (section 3B), not systems. Preliminaries such as insurance, site
toilets, skips and scaffold hire are cost-plan items and stay out of the
checklist.

**Not audited for depth.** Interfaces and failure modes are not used by the
profile v1. The checklist's presence values become `system_present` inputs
for them later. Only 3 of 226 rules have verified clauses and 9 tables are
missing, so most of the compliance section will read "pending" until they
are verified against the instruments (AGENTS.md rule 5).

## 6. The one-line register

Replaces the tile layout in `web/src/DocumentRow.tsx` and the page chrome in
`web/src/Project.tsx`. Reference: the Clerk register
(`../clerk/frontend/src/components/project/DocumentRepositoryPanel.tsx`,
the schedule table). Copy the look, not the code.

**Remove:** the tally line (filed, to check, stored not filed), the large
drop panel and the legend. Show the live state only when it is not live
("Reconnecting…", "Offline, retrying"); say nothing while live.

**Keep:** drop anywhere on the page, an "Add files" button in the register
header, live updates without moving focus, every band state, correction in
place, retry, download, and the duplicate notice (announced, not a row).

**Row:** one line, 0.75 rem type, about 32 px high, sticky header, sortable
columns:

| Column | Width | Content |
|---|---|---|
| No. | 5.5 rem | Document number, truncated, full value on hover |
| Title | flexible | Title, or filename while unfiled |
| Rev | 2.5 rem | Revision |
| Date | 4.75 rem | `dd/mm/yy` |
| Disc. | 3 rem | Discipline short code; full name on hover |
| Mark | 1.25 rem | One icon for the row's worst state |

A row whose fields all came from the document or the user shows no mark. A
mark always has an icon plus a text label for screen readers and on hover
("2 fields to check", "Not checked: Jev didn't answer in time", "Stored, not
filed: no text layer", "Filing stopped. Retry"). Colour never carries meaning
alone. A value set by the user keeps the ember mark in its cell.

**Expand:** Enter or a click on a row opens one detail line under it with all
eight fields as compact editable cells (reuse the existing cell editor and
its provenance text), plus Download and, for a sheet, "Sheet n of N" with a
link to the source. One row open at a time; Escape closes it and returns
focus to the row.

**Uploading:** the row shows the filename with a thin progress line along
its bottom edge. A failed upload shows the error in the Title cell and a
dismiss button in the Mark cell.

**Drawing sets:** the retained source shows as a slim group row
("filename · 12 sheets") that toggles its child sheet rows, open by default.

**Default order:** uploads in progress first, then document number ascending.
Clicking a header sorts by that column.

**Width:** the register is one third of the page. Below about 360 px of
register width, the Date column hides; the value stays in the expanded row.

## 7. Speed

New budgeted paths in `bench/budgets.json`. The build fails when broken.

| Path | p50 | p90 |
|---|---|---|
| `project_profile_read`: GET profile, one SQL read of precomputed rows | 50 ms | 150 ms |
| `profile_edit`: PUT one profile value, including a type suggestion | 50 ms | 150 ms |

The register keeps `project_document_list`. Profile building is background
work and is not a user-facing path. Record per document: pages, passages, Jev
calls, elapsed time and errors. Set a freshness target after the first live
run (owner decision). Foreground filing keeps priority and the interactive
Jev reserve.

## 8. Failure handling

- The profile never blocks filing, and a profile failure never hides.
- Jev unavailable: profile rows keep their last values; the affected
  documents show "profile not read yet" in the profile's source list; jobs
  retry with backoff.
- Thresholds missing or zero: Jev answers are recorded but not applied. The
  profile says so in one line. It does not pretend to be empty.
- A deprecated system or determinant ID in stored facts is shown under its
  replacement.

## 9. Evaluation

Answer keys for Newham, Hale, Petersham, Rutherford and Mornington in
`data/eval/profile/answer-keys/`, drafted from the documents with
`reviewed: false`. Only the owner marks them reviewed. Seed the header fields
from Clerk's `data/eval/jev/answer-keys/profile-setup.yaml`, which is also
unreviewed. Score per field: correct without user action, wrong and applied,
blank. Score presence per system. Gates: no wrong green; report the share
correct without user action against the 80% target. Recorded Jev responses
replay in tests, as in intake.

## 10. Out of scope for v1

Harvesting "referenced but not uploaded" documents; tender comparison;
interfaces and failure-mode views; verifying the 9 missing tables; per-part
extraction beyond exact label matches; OCR.
