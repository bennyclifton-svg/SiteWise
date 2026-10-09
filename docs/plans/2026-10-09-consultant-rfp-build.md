# Consultant RFP from the golden set: implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

Revision 2, 9 October 2026. This revision takes in the
[peer review](../rfp-golden/peer-review-2026-10-09.md). §7 lists every
recommendation in the review and where this plan handles it. Nothing in this
plan has been built.

**Goal:** Reports → "RFP — services appointment" → Architect / Town Planner /
Traffic / Civil / Structural produces an RFP built from approved, conditional
scope modules. The modules come from the golden set in
`docs/rfp-golden/<discipline>/rfp.md`, repaired as described in Phase A.
SiteWise selects the position for **this** project from saved facts: the
appointment boundary, the affected parts and systems, the work type, the
pathway, the procurement route and the site conditions. It never prints
simple and complex columns for the user to resolve. If a material fact is
unknown, SiteWise raises a focused decision or prints an explicit pricing
assumption. It never quietly turns an unknown into "no". The user finishes
with toggles, counts and I/O/X pickers, the same way they edit the Profile
page.

**Architecture:** Each discipline's reviewed RFP becomes a data file
(`knowledge/rfp/<discipline>.yaml`), and shared work-type and use modules
live in `knowledge/rfp/modules.yaml`. These files hold canonical scope items,
stages, allowances, positions, returnables, fee bases, and the conditions
under which each row applies. A pure resolver uses three-valued logic: a
condition can be true, false or unknown. It turns *drivers* (project facts
held as known yes, known no, unknown or conflicting, frozen at refresh) plus
the user's *settings* into a resolved RFP. The resolved RFP is rendered into
the existing `reports.Section`/`Block` model, so citations, protected wording
edits, "Before issue", changes since issue, page fit and PDF issue all keep
working. A setting change re-resolves from the frozen drivers. It does not
reload the project, so it stays inside the `report_write` budget.

**Tech Stack:** Go (`internal/knowledge`, new `internal/rfp`,
`internal/store`, `internal/httpapi`), PostgreSQL 17 (one new table), React
(`web/src/Reports.tsx`, new `web/src/RfpEditor.tsx`), Playwright. No new
dependency. No Jev call on any RFP path: generating an RFP is lookup and
computation only (AGENTS.md rule 1, "code ... computes, looks up and controls
flow"). §2.10 explains why.

---

## 0. The task statement (AGENTS.md "Starting a feature")

| | |
|---|---|
| **User outcome** | Pick a services package and confirm the appointment boundary. SiteWise returns an RFP that holds only the scope this project needs. The user answers the few open decisions in a minute or two and issues it. |
| **Why** | Time-to-decision and answer trust. The current RFP prints profile keys, work items and open reviews, none of which a consultant prices from. The new drafts follow the layout and scope positions in 12 historical proposals (three firms per discipline; structural and civil share the same three), most of them for two projects: the Arthur St warehouse extension and the Yarramundi camp redevelopment. Those firms priced *other* RFPs; nobody has yet priced these newly written drafts. The proposals are evidence for the drafts. They do not show that the drafts are market-tested. |
| **Lane** | **A** for the content pass, resolver, data files, store and issue gate (a user relies on the scope position, the appointment boundary and the rule thresholds). **B** for the editor layout. **C** for the new settings endpoint (wrong org, missing report, unexpected values). |
| **Out of scope** | Tranche 2/3 disciplines (they keep today's RFP until their golden files are signed off). New profile fields. A single project-level store for facts shared across RFPs (O9). Discipline chosen by Jev. Jev reading documents on the RFP path (§2.10). Combined structural + civil RFP (owner decision 2026-10-09). Firm lists, sending, tender returns. Promoting RFP-only settings (LGA, zone, design vehicle) into the profile. Moving RFPs from Reports to Procurement in the navigation: Reports also hosts the PMP, so moving RFPs needs its own decision. Changes to package stages or `DefaultStages`. |
| **Speed budgets** | Refresh (`report_assemble`) p50 ≤ 300 ms, p90 ≤ 1,000 ms (unchanged). Setting change (new `report_setting`) p50 ≤ 100 ms, p90 ≤ 250 ms, the same as `report_write`. Read unchanged (`report_read` 100/250). Issue/export unchanged (`report_issue` 100/250). Filing stays within p50 ≤ 1 s, p90 ≤ 2 s while RFP work runs alongside it. Zero Jev fan-outs on every RFP path. These are budgets, not measurements; task 14 measures them on populated, large and mixed-use RFPs. |
| **Done** | (1) Phase A content is signed off by the owner for each discipline. (2) Every selection answer key in task 8 (twelve scenarios) passes: expected inclusions, exclusions, options, unresolved decisions and allocation gaps, each with its source. (3) Source-fidelity snapshots for Arthur St and Yarramundi (task 7) match the owner-reviewed Markdown. (4) No material unknown becomes a silent exclusion. Each one shows as a decision or a printed assumption. (5) A user setting survives "Refresh draft" and the next draft after issue. If the source fact changed in the meantime, the setting stays and a conflict is shown. (6) The issued PDF meets the page policy decided under O1, with no shrunk text and no dropped scope. (7) The bench passes with the new path on populated cases. (8) `python tools/check_knowledge.py` passes. |
| **Edge cases** | The package has no confirmed discipline. A driver is unknown, or two sources conflict. The appointment boundary is not confirmed. The pathway is unsupported (REF, modification, planning proposal). The regulated-design regime is undetermined. An exclusion has no responsible party. A setting refers to a row the discipline file no longer has. The source fact changed after the user's setting. The setting value is invalid. The draft version is stale (409). The report belongs to another org (404). The draft is already issued. The discipline file is still `status: draft` (issue blocked, §2.8). Mixed-use parts. The civil R1 grid with inactive elements. An oversized report. |

---

## 1. Current RFP vs golden RFP

| | Today (`tpl.rfp-capex`, `internal/reports/assemble.go` + `rfp.go`) | Golden after Phase A (`docs/rfp-golden/*/rfp.md`) |
|---|---|---|
| Structure | 7 generic sections, the same for every consultant | Skeleton §1-7 + R1-R9 (`docs/rfp-golden/skeleton.md`). §2-5 and parts of R2-R5 differ by discipline |
| Content source | Every profile value, work item, scope record, latent-condition proposal, cost line | Scope modules selected by appointment boundary, affected parts and systems, work type, pathway, procurement and site conditions |
| Form | One labelled line per block | Mostly tables: project table, stage table, allowances, the project's own I/O/X column, R1 fee grid |
| Unknown facts | Printed as open reviews | A focused decision, or a printed pricing assumption or option |
| Editing | Free-text wording per block only | Choices: boundary, stage state, counts, I/O/X per row, responsible party, per-fact drivers. Wording edits still apply to prose rows |
| Length | ~2 pages after c03b05b; hard limit 2 (`render/pdf.go:209`, `fit.go:20`) | Draft target is 3 pages. The final policy is decided under O1, after measuring populated cases |

Keep from today's code: `Section`/`Block`/`Table`, `ApplyEdits` (protected
wording), `Cite`, the internal "Before issue" section (`ReviewSectionID`),
`ChangesSinceIssue`, `IssueContent` disclosure, `render.PDF`/`render.Fit`,
the issue gate, and the `report_versions` and `report_edits` tables. Do not
add a second document generator.

Replace, for the five disciplines only: `Assemble`, `shapeRFP` and the
`tpl.rfp-capex` section list.

---

## 2. Design decisions

Decisions marked **Owner** need a yes before the task that depends on them.
Each one has a recommendation.

### 2.1 Discipline files and modules are knowledge data, not code

`knowledge/rfp/skeleton.yaml` holds the shared parts:

- §1 shared project rows
- the complete §6 shared terms
- §7 evaluation
- R1-R9 with the full shared wording, not just the word "Shared"
- the flag vocabulary
- the client procurement defaults

`knowledge/rfp/modules.yaml` holds the shared work-type and use modules:
new, refurb/extend, remediation, advisory, mixed use, and live institutional
or industrial site. `knowledge/rfp/<discipline>.yaml` holds each
discipline's items, stages, allowances, positions and discipline modules.

All of these files load into `knowledge.Catalog`. Editing one therefore
changes `KnowledgeVersion`, and existing drafts go stale through
`reports.StaleReasons`, so no new staleness code is needed.

Every row carries `provenance:` with a `kind`:

- `evidence`: ledger rows, given as `discipline#n`, for example
  `architecture#38`
- `seed`: the seed file and anchor
- `practice`: a named practice source
- `client_decision`: a deliberate client choice that differs from the
  evidence

Review item P5 applies here: a firm excluding something in its proposal does
not make "include it" an approved default. Provenance is for review and is
never printed. Source lineage and project evidence are separate things. A
clause that came from a proposal is not, for that reason, applicable to this
project.

### 2.2 Thresholds have a kind

```yaml
thresholds:
  - {key: budget_complex, kind: client_default, text: "Budget over $20M suggests complex service levels", source: golden prose, verified: false}
  - {key: dbp_regime, kind: statutory, text: "NSW regulated-design regime applies", source: "check against the DBP Act, Regulation and current NSW guidance", verified: false}
```

There are three kinds:

- `statutory`: legal applicability, such as the regulated-design regime or a
  referral threshold.
- `client_default`: a client preference, such as $20M scale, insurance
  limits, ISO, weights, validity, meeting counts or durations.
- `evidence_estimate`: a figure taken from a historical proposal, such as
  SEA durations.

The loader enforces one rule here: **an unverified statutory threshold may
appear only in a row's `suggest:`, never in its `when:`**. It can raise a
decision but cannot select scope. Setting `verified: false` records
provenance. It does not make a class-only predicate fit to issue
(AGENTS.md rule 5).

### 2.3 Conditions: a closed vocabulary, OR-of-ANDs, three-valued

A `when:` is a list of clauses. A clause is a map of driver keys, and each
key lists the values it accepts. There is no expression language and no
evaluator to secure. Evaluation uses Kleene logic:

- A key is **true** if the driver is known and holds one of the listed
  values.
- A key is **false** if the driver is known and holds none of them.
- A key is **unknown** if the driver is unknown or conflicting.
- A clause (AND) is false if any key is false; otherwise it is unknown if
  any key is unknown; otherwise it is true.
- A `when` (OR) is true if any clause is true; otherwise it is unknown if
  any clause is unknown; otherwise it is false. An empty `when` is true.

Numeric keys end in `_over`. If the number is unknown, the key is unknown.
Keys must come from the closed driver list in §2.5, and the loader rejects
unknown keys.

When a row's `when` evaluates to unknown, the row's `if_unknown` decides
what happens:

| `if_unknown` | Effect |
|---|---|
| `ask` (default) | The row shows as "To confirm" in the draft. A blocking decision goes under "Before issue", saying which clauses change. |
| `option` | The row prints as a separately priced option, with the row's `assumption:` wording. An example is "Assess the feasibility of retaining the existing detention system ... Price replacement design separately as an option pending that assessment." |
| `include` | The row prints as included, with its `assumption:` wording stated in the RFP. |

An unknown never resolves to "excluded", "zero" or "verified".

```yaml
when:                       # D&C tender package, with or without novation
  - {procurement: [design_construct]}
when:                       # existing drainage retained: asset known present, OSD retained
  - {work_type: [refurb, extend], flag: [osd_existing]}
if_unknown: option
assumption: "Assess the feasibility of retaining the existing detention system using the listed records and agreed investigations. Report capacity, access and approval constraints. Price replacement design separately as an option pending that assessment."
suggest:                    # unverified statutory rule: raises a decision only
  - {threshold: dbp_regime, parts_class: ["2", "3", 9c]}
```

The example uses `refurb` because the taxonomy's work types are `new`,
`refurb`, `extend`, `remediation` and `advisory`. Fit-out falls under
`refurb` ("Refurbishment / fit-out / upgrade").

### 2.4 Canonical items, appointment boundary, fee basis

**Canonical items.** Each discipline file lists `items:` once, with a key,
a label, a `kind` and provenance. Stages reference item keys in their
deliverables. Positions, exclusions and returnables also point at item keys.
One item has one position for this project. The loader rejects an item that
is statically included by a stage and also has a static X position. The
resolver re-checks this after settings, and a clash becomes a blocking
"included and excluded at once" check. Design, documentation and
coordination of the same element are separate items when different parties
hold them. For example, pavement design, pavement documentation and
pavement coordination are three items.

There are two item kinds:

- `technical` items, such as shoring, services fixings, miscellaneous
  metalwork, flood modelling, MUSIC, SIDRA and inspections. Their `when`
  must test systems, risks or site facts. **The loader rejects a technical
  item whose position depends on `scale`.**
- `service_level` items, such as renders, BIM level of detail, meeting
  counts and review depth. These may use scale.

Complexity only suggests quantities and review depth.

**Exclusions need an owner.** An `X` position carries `responsible:`, for
example `geotech`, `contractor`, `client` or `not_required`. The resolver
looks the party up among the project's services packages. When the party
has no package, the RFP prints "by others: to be appointed", and an
"Unallocated: <item>" check is raised (§2.8 says whether it blocks).

**Appointment boundary.** These are settings with no silent default:

- `boundary:start` and `boundary:end` are stage keys.
- `stage:<key>` is one of `include`, `option` (a later stage, priced
  separately), `exclude` or `done` (completed by others; it goes into
  "Information provided").

The resolver suggests a boundary from procurement and novation: traditional
runs Concept to For-construction; D&C runs Concept to the D&C tender
package; novation adds post-novation as an option. The user confirms the
boundary once before issue. A due-diligence or planning-only boundary can
never pick up DD or construction documentation, because every stage is
intersected with the boundary before its `when` is evaluated. Stages after
`boundary:end` that would match are offered as `option` lines, so the client
can price them now and commission them later.

**Procurement stages are separate deliverables** (Phase A, C3):

- traditional tender documentation
- D&C design-intent and performance tender documentation
- post-novation completion

Each states its deliverables and who holds design responsibility at
handover. A percentage on its own is never the deliverable.

**Fee basis per row** replaces `pricing: lump|capped|own_line`:

| `fee_basis` | Meaning | R1 line |
|---|---|---|
| `fixed` | Fixed fee for the defined deliverable and allowance | Base fee |
| `time_to_ceiling` | Actual time at R3 rates up to a stated monetary ceiling | Ceiling amount, in its own column |
| `per_unit` | Rate × the client-stated quantity (traffic survey locations) | Own line |
| `option` | Priced separately, not in the base fee | Options block |

Exceeding an allowance is not an automatic paid variation. The skeleton
terms (Phase A, C2) say who approves extra work in writing, and when, under
the client's agreement. A stage sign-off sets a baseline; it does not entitle
the consultant to charge for all later design development. Correcting the
consultant's own error is not a variation.

**Allowances** carry `owner: client | consultant`, plus `unit`, `default`,
`fee_basis` and a stable key that becomes the R1 row identity:

- A blank `client` allowance is a blocking check. Examples are meeting
  counts, inspection counts and survey locations. A blank client quantity is
  not the same as a blank fee cell.
- A `consultant` allowance is a returnable that the consultant proposes in
  R1 or R2, such as a drawing sheet count.

**I / O / X.** Internally a position is `I`, `O`, `X`, `unknown` or `na`.
`O` is never counted as accepted responsibility and never rolls into the
base fee. R1 keeps three blocks: base fee, options, and approved allowances.
An item that appears in two sections is priced once, under its key.
`procurement.ValidateScope` (included/excluded only) is not touched. RFP
positions live in report settings, not in package scope records.

### 2.5 Drivers: where each fact comes from

`rfp.Drivers` is built once per refresh from the report snapshot and saved
in `source_revisions.rfp_drivers`. Each fact is
`{values, state: known|unknown|conflict, source, source_rev}`. The source is
one of `profile`, `package`, `default` or `user`. Gaps are filled by **RFP
settings**, not by new profile fields, which are out of scope.

**Parts keep their association.** The drivers carry
`parts: [{id, label, class, use, work_type, basis: existing|proposed, gfa, gfa_rev}]`
from the profile parts. The report snapshot already keeps part IDs and
labels. The setting `parts:affected` lists the parts this appointment
covers; it defaults to all parts and is shown so the user can confirm it.

In a `when`, the `class` key is evaluated over the affected parts, and
`Why` names the part that matched ("Class 3, part *Lodge*"). There is no
flattened project-wide class list and no `top_storey_class` shortcut. A
marketing label never decides the legal class: `class` comes only from
`ncc_class`. The project table prints one row per affected part, and
quantities from incompatible bases are never summed.

| Driver | Golden use | SiteWise source today | Missing → |
|---|---|---|---|
| `parts` / `class` / `use` | Scale suggestion; regulated-design suggestion; interiors; mixed-use module | profile parts, `ncc_class` per part | unknown → decision |
| `work_type` | Role wording; existing-building modules | profile `work_type` (`new`, `refurb`, `extend`, `remediation`, `advisory`) | setting |
| `pathway` | Planning stage bundles | taxonomy `planning`: `exempt`, `cdc`, `da`, `ssd`, `planning_permit`. RFP-only values: `ref`, `modification`, `planning_proposal`, `advice_only` | unknown → **blocking decision**. There is no DA default. A pathway with no reviewed bundle shows as "unsupported pathway", which blocks issue and never falls back to SEE wording (O6) |
| `procurement` | Tender deliverables, bidding, novation, boundary suggestion | taxonomy `procurement_route`: `traditional`, `design_construct`, `eci`, `managing_contractor`, `alliance`, `ppp`. The golden "CM" wording is mapped to `managing_contractor` only if the owner confirms it in Phase A; otherwise it is dropped | unknown → decision |
| `novation` | Post-novation option; R1 split | `Package.Novation` | from the package |
| `boundary` | Every stage | settings `boundary:start/end`, `stage:*` | suggested, **must be confirmed** |
| `state` | NSW-only rows | profile `state` | setting |
| `regulated_design` | Regulated-design stage, R4 accreditation | none. `suggest:` raises the decision when an affected part is Class 2, 3 or 9c in NSW, citing the unverified threshold and the NSW guidance date note (existing Class 3/9c alteration enters the regime from 1 July 2028; exemptions apply) | `unknown` → blocking decision: "applies", "does not apply" or "professional advice recorded" (O5) |
| `flag.<name>` (one fact per flag) | heritage, bushfire, flood, contamination, coastal, country, osd_existing, public_road, authority_asset, rail, basement, excavation_adjoining, heavy_vehicles, state_road, community, multi_building, staged, live_operations, … | `heritage`/`heritage_status`, `bushfire_prone_land`/`bal`, `flood_hazard`, `contamination_present`/`potentially_contaminated_land`, groundwater/excavation determinants, taxonomy `operational_constraints`; the rest are not captured | **unknown**, never off. The editor shows Yes / No / Unknown per flag |
| `budget` | Scale suggestion only | Cost plan total (`Snapshot.Commercial`) | unknown |
| `site_area`, `gfa`, `storeys`, `parking` | Project table; civil and traffic quantities | profile values, kept per part with their basis and revision | blank; blocking when the discipline marks the quantity as a pricing basis |
| `consultants` | Architecture §2; exclusion routing | services packages with their `lifecycle_status` | none |
| `lga`, `zone`, `lot_dp`, `design_vehicle`, `contact`, `ref`, `closes`, `agreement` | Header, project rows, terms | `Site.Address`, `Site.Lot` (address and lot only) | setting text field |

How some of these drivers behave:

- **Booleans** set `flag.x = yes` only when the source says `true`. `false`
  sets `no`. Missing sets `unknown`. Disagreeing sources set `conflict`, and
  both values are shown.
- **Flood.** `flood_hazard` sets `yes` for any option other than "none". A
  stated flood planning level is not taken as proof that the flood scope is
  complete.
- **Consultants** print with their status: `appointed` for appointed or
  closed packages, `to be appointed` for proposed, planned or procuring
  ones, and nothing for cancelled ones. An item with no matching package
  prints as `unallocated`. A proposed package is never described as an
  engaged consultant.
- **Budget** is an internal selection input. It is printed in the project
  table only when the report's existing disclosure choice is on, and
  `IssueContent` strips it from the block and from its basis otherwise
  (task 6). A complexity threshold never forces the budget to be disclosed.
- **Operational and access conditions** are enough to trigger a module or a
  decision. They are not enough to invent shutdown dates, access windows or
  stage counts.

A **driver report** is part of done. Task 14 prints, per scenario and
discipline, which drivers were filled from the profile, which from
settings, and which stayed unknown. That tells the owner which profile
fields are worth adding next.

### 2.6 Settings are per-fact overrides that survive regeneration

New table `report_settings(org_id, report_version_id, key, value jsonb,
basis jsonb, user_id, version, ...)`. It has the same shape and the same
carry-forward rule as `report_edits`: rows are copied into the next draft
after an issue. The rows of an issued version are immutable.

| Key | Value | Example |
|---|---|---|
| `driver:<name>` | `["..."]`, `"unknown"`, or `null` (follow the source) | `driver:pathway` → `["cdc"]` |
| `driver:flag.<name>` | `"yes"` / `"no"` / `"unknown"` / `null` | `driver:flag.flood` → `"yes"` |
| `parts:affected` | `["<part id>", ...]` | |
| `boundary:start`, `boundary:end` | stage key | `boundary:end` → `"dnc_tender"` |
| `scale` | `"simple"` / `"complex"` / `null` (rule) | service levels only |
| `stage:<key>` | `include` / `option` / `exclude` / `done` | `stage:construction_admin` → `"include"` |
| `element:<key>` | `true` / `false` | civil D |
| `allowance:<key>` | string count | `allowance:design_meetings` → `"4"` |
| `position:<key>` | `"I"` / `"O"` / `"X"` | |
| `responsible:<item>` | package key or `client` / `contractor` / `not_required` | |
| `field:<name>` | text | `field:closes` → `"2026-11-06"` |
| `weight:<name>` | integer % | `weight:price` → `40` |

**Each fact is overridden separately.** Changing the flood flag writes
`driver:flag.flood` only, so later evidence for other flags still flows
through. `basis` records the source value and revision at the moment the
user set the override. On refresh:

- If the source is unchanged, the setting applies silently.
- If the source has changed, the setting still applies, and a "Changed since
  your setting: Profile now says <new>" conflict appears under "Before
  issue". The conflict has Keep and Use-new actions. Keep re-stamps `basis`;
  Use-new deletes the setting.

Precedence: **user setting > profile/package > discipline default**.
Wording edits (`report_edits`) still apply on top for prose blocks (role,
project sentence, terms), and a structured setting is never lost because of
a wording edit. If a setting's key is not in the current discipline file, it
is not dropped. It shows under "Before issue" as "Setting no longer
applies", following the `protected_edits` precedent.

### 2.7 Fast editing: frozen drivers, pure re-resolve

`POST /reports/{id}/draft` (refresh) reloads the snapshot, computes the
drivers, and stores them in `source_revisions.rfp_drivers`. This is the
existing `report_assemble` path and budget.

`PUT /reports/{id}/settings/{key}` does the following in one transaction:

1. upsert the setting, stamping its basis from the stored drivers
2. read the stored drivers and settings
3. run `rfp.Resolve` → `rfp.Sections` → `ApplyEdits` → `Cite`
4. update `report_versions.sections`

There is no snapshot reload and no Jev, so this is budgeted as
`report_setting` at 100/250 ms. Nothing is judged while the user waits
(AGENTS.md rule 3).

### 2.8 Issue gate, blocking checks and the frozen basis

Blocks built from a `status: draft` discipline file or module are
`Provisional`, and their basis carries
`{"method":"rfp_discipline","status":"draft"}`. `unreviewedReportClause`
treats those blocks as unreviewed, so issue returns `unreviewed_clauses`.
This is today's rule for clauses. Only the owner flips a file to
`reviewed`, and the flip is the release gate for each discipline. The editor
shows "Rules for this discipline await owner review" rather than marking
every row "[PROVISIONAL]".

Checks are either **blocking** or **advisory**.

Blocking checks stop issue:

- the boundary is not confirmed
- an `ask` row is unresolved (including pathway, regulated design and
  conflicting facts)
- an unsupported pathway
- a blank `client` allowance
- a blank pricing-basis quantity
- an item that is included and excluded at once
- weights that do not total 100%
- a missing agreement reference
- a source-changed conflict on a setting that changes scope

Advisory checks allow issue:

- a blank `field:contact` once a ref exists
- an unallocated `X` item whose owner is a later tranche discipline
- a setting that no longer applies

Issue reuses the existing immutable issue model. `IssueSnapshot` records:

- the skeleton, module and discipline IDs and versions
- the resolved drivers with their sources
- every setting
- the allowance values
- the agreement and the revisions of the information provided

The renderer reads the page policy from `IssueSnapshot.SourceRevisions.TemplateID`,
so the issue gate and the screen agree.

### 2.9 Owner decisions

| # | Decision | Recommendation |
|---|---|---|
| O1 | RFP page policy. Today's limit is 2 (NW-REQ-017/018; `fit.go:20` and the hard check at `render/pdf.go:209`). The drafts assume 3 | **Measure first.** Task 11 renders the populated complex scenarios (Yarramundi civil, the mixed Class 2/6/7a case, Arthur St architecture) before any limit is set. Recommended policy: a short invitation and project summary, then readable scope and returnable schedules with natural pagination, and an RFP-specific limit chosen from the measured cases. Never shrink text below `MinimumSize` and never drop scope to fit. RFT and PMP keep 2 / 1.5. Blocks task 11 step 3 |
| O2 | Services packages with no golden discipline (Geotech, Surveyor, Certifier, Services, ...) | Keep today's RFP for them, labelled "Generic RFP: discipline template not yet built". Tranche 2/3 replace it |
| O3 | Blank placeholders | **Split by kind.** A consultant-returned cell (fee, hours, weeks, rates, drawing count) prints blank and is not a check. A client-defined allowance or quantity, the appointment boundary, the pathway, and a fee-basis assumption must each be set, turned into a separately priced investigation, or turned into a consultant-proposed returnable before issue. Until then it is a blocking check |
| O4 | Package → discipline mapping | `Package.DisciplineID` only (`consultant.architect`, ...; the IDs already exist in `knowledge/determinants.yaml`). A title match ("Architect", "Civil", ...) **suggests** a discipline. The user confirms it once, and the existing package `discipline_id` PATCH saves it. A later free-text rename never changes the selected scope |
| O5 | Regulated-design applicability | No class-only rule selects scope. The `suggest:` raises the decision; the user records "applies", "does not apply" or "professional advice recorded". Revisit when a verified rule exists that covers work type, new or existing status, affected part, dates and exemptions |
| O6 | Planning pathways without a reviewed bundle (REF, modification, planning proposal; CDC advice if Phase A does not author it) | Show them as unsupported and block issue until their bundle is authored and reviewed. REF gets its own determining-authority workflow, never DA lodgement wording |
| O7 | Success measure, replacing "80% right" | Measure avoidable edits and material omissions on the task 8 scenarios, plus reviewer time on real example RFPs. The owner sets a review-time target after seeing the Phase A examples |
| O8 | Client procurement defaults: insurance limits, ISO, evaluation weights, validity, meeting counts, durations | Keep them as editable `client_default` values, each with its source. Accept "or equivalent management system" alongside an ISO certificate unless the client requires certification. Never present them as market rates or statutory thresholds |
| O9 | Facts shared across the five RFPs of one project (flood, LGA, zone, lot) | Defer one project-level confirmed source to a later plan. For now each RFP holds its own settings, and the driver report lists the facts that are set in more than one RFP, so a profile field can be justified |

### 2.10 Jev: none on the RFP path, deliberately

The review suggested (its steps 2-4) that Jev could interpret document
evidence when judgement is needed, with one fan-out before the RFP opens.
This plan keeps **zero Jev calls on every RFP path**, for these reasons:

- **Every driver is already a saved fact or a setting.** Facts found in
  documents reach the profile through the existing intake and profile
  pipelines, which already run Jev off the user's path. That follows the
  [pre-parsed extraction pattern](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
  and AGENTS.md rule 3: "nothing judged while a user waits if it can be
  precomputed".
- **An unknown is handled without guessing.** Three-valued resolution turns
  an unknown into a decision or an assumption, which covers the
  [confidence routing](https://docs.typesafe.ai/patterns/confidence-routing)
  case without a model call.
- **The judgements that remain are not Jev's to make.** Regulated-design
  applicability, engineering adequacy and threshold arithmetic are outside
  what Jev should decide
  ([jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
  Code computes; the user or a professional decides.

If the driver report (task 14) shows a document-only fact that drives
material scope, for example existing OSD seen on drawings, it is added as a
profile question in a later plan, with one [fan-out](https://docs.typesafe.ai/patterns/fan-out)
per state and per-question thresholds evaluated on answer keys. It is never
added on this path.

---

## 3. File map

| Create | Purpose |
|---|---|
| `docs/rfp-golden/drivers.md` | Completed driver table (Phase A, C5) |
| `data/eval/rfp/answer-keys/*.yaml` | Selection answer keys, twelve scenarios (C6; draft until the owner approves) |
| `knowledge/rfp/skeleton.yaml` | Shared rows, complete terms, evaluation, R1-R9, flags, client defaults |
| `knowledge/rfp/modules.yaml` | Shared work-type and use modules |
| `knowledge/rfp/{architecture,town-planning,traffic,civil,structural}.yaml` | Per-discipline items, stages, allowances, positions and modules |
| `internal/knowledge/rfp.go`, `rfp_test.go` | Types, loader, validation |
| `internal/rfp/drivers.go`, `drivers_test.go` | Snapshot → facts (known, unknown, conflict), parts |
| `internal/rfp/resolve.go`, `resolve_test.go` | Discipline + modules + drivers + settings → Resolved (three-valued) |
| `internal/rfp/sections.go`, `sections_test.go` | Resolved → `[]reports.Section` |
| `internal/rfp/markdown.go`, `golden_test.go`, `testdata/*.md` | Source-fidelity snapshots |
| `internal/rfp/answerkey_test.go` | Selection answer keys |
| `internal/rfp/controls.go` | Resolved → editor controls (JSON for the UI) |
| `internal/db/migrations/021l_report_settings.sql` | Settings table |
| `internal/store/report_settings.go`, `_test.go` | Upsert, carry forward, source-changed conflicts, re-resolve |
| `internal/httpapi/report_settings.go`, `_test.go` | PUT/DELETE setting |
| `web/src/RfpEditor.tsx`, `web/src/rfp.css` | Editor panel |
| `web/tests/rfp-editor.spec.ts` | E2E |

| Modify | Change |
|---|---|
| `docs/rfp-golden/skeleton.md`, `*/rfp.md`, `*/ledger.md` | Phase A repairs (owner reviews in chat) |
| `knowledge/reports/templates.yaml` | Add `tpl.rfp-consultant` |
| `internal/knowledge/catalog*.go` | Load `knowledge/rfp/` |
| `internal/store/reports.go:105-115, 176-278` | Choose the template from the confirmed discipline; branch refresh to the RFP path |
| `internal/store/report_issue.go:170` | Draft discipline blocks are unreviewed; blocking checks; frozen RFP basis |
| `internal/reports/draft.go:30` | `Table.Display` (`"document"` vs saved comparison) |
| `internal/reports/disclosure.go` | Strip the RFP budget block and its basis unless disclosed |
| `internal/reports/render/pdf.go:138-150, 209`, `fit.go:20` | Document tables; the page limit comes from the template, not the constant 2 |
| `internal/httpapi/reports.go` | `getReport` returns `rfp` controls |
| `web/src/reportApi.ts`, `web/src/Reports.tsx` | Controls type and calls; package select shows the discipline, or "confirm discipline" |
| `tools/check_knowledge.py`, `tools/test_check_knowledge.py` | Validate `knowledge/rfp/` |
| `bench/budgets.json`, `cmd/intake-bench/reports.go` | `report_setting` path; populated scenarios |

---

## 4. Phase A: content pass (gates transcription)

The review found that the drafts' weakness is *when* each piece of wording
applies, not the wording itself. Transcribing them as they stand would
encode one warehouse and one camp as universal rules. Phase A repairs the
content first. It is lane A content work in `docs/rfp-golden/`, reviewed by
the owner in chat, one discipline at a time, following the method in the
[golden-set plan](2026-10-09-consultant-rfp-golden-set-plan.md). Everything
written from seeds stays `status: draft`. Seeds support scope discovery;
they do not verify code numbers.

Phase B tasks 1 and 4-6 (types, drivers, resolver, sections) can start once
C2 settles the skeleton shape. The decisions they rest on (three-valued
facts, boundary, fee basis, item kinds) do not depend on the final wording.
A transcription task for a discipline (tasks 2 and 7) waits for that
discipline's C3/C4 sign-off.

### C1: Provenance repairs

1. **Architecture PACE fee.** Re-read `Architecture/Arch RFP3.pdf`, PDF
   page 1 (printed page 6), as an image *before* editing. The review read it
   as $25,000 + $20,000 = $45,000 excl. GST and $49,500 incl. GST. The
   ledger says $45,500 / $50,050 / Stage 1 $25,500. Correct ledger rows 10
   and 28 to what the page shows, and note the check.
2. **Planning modification references.** #62-63 and #54 do not support
   modification applications or post-approval work. Re-cite those rows to
   the planning seed or a practice source.
3. **Civil design criteria.** Replace `S p1` with the canonical row #65.
   Give every selection a stable discipline-qualified ID (`civil#65`).
4. **Inherited source mistakes.** The Engine Room comparison, page 5, labels
   cumulative figures `Total Stage 2`. Keep the original label as an
   anomaly in the ledger and compute normalised totals independently. No
   source's arithmetic is reused as an application calculation.
5. **Deliberate changes.** Architecture's certifier liaison, document-system
   uploads and the sustainability inclusion are marked
   `provenance: client_decision` with the reason. They are not presented as
   evidence-backed defaults.
6. **Scope beyond the evidence.** Architecture's construction administration
   and post-novation rows get explicit seed or practice provenance, as the
   structural construction stage already has. The original precedents did
   not price these services in full.

### C2: Skeleton

1. **The controlling inputs come first:** appointment boundary (start and
   end stages, completed work, deliverables bought now, optional later
   stages), affected parts, and per-item applicability with the four fact
   states. Add a "How this RFP was selected" note for the reviewer only,
   not printed.
2. **Complete shared terms**, written out in full rather than as "Shared":
   - engagement under the attached agreement
   - written authority for scope changes, for exceeding an allowance and
     for fee changes, under that agreement
   - a stage sign-off is a baseline, not a blanket entitlement to charge
   - correcting the consultant's own error is not a variation
   - invoicing and payment
   - indexation
   - validity
   - questions and addenda
3. **R6 Departures** covers scope, allowances, program, fees and the
   attached agreement. Each departure gives the clause or item ID, the
   proposed change and its cost and time effect. R6 explains the difference
   between a departure (which is evaluated) and a general exclusion added
   without being listed (which is treated as non-compliant). Fix
   architecture and planning, which limit R6 to sections 3-5.
4. **R1** has stable stage and item IDs (internal; not printed as codes),
   separate base fee, options and approved-allowance blocks, the GST basis
   and disbursements. An option that appears in two sections is not counted
   twice.
5. **R2** gives the deliverable, format, stage, review rounds, preparer and
   recipient. BIM and model exchange follow the appointment's needs, not
   the building class. It also says whether editable source files are
   included, and what reliance is permitted.
6. **R4/R7** give the named lead and their availability by stage,
   subconsultant roles, input dependencies, and the earliest start. The
   resources must match the program. R7 separates consultant effort,
   elapsed authority time and required-input dates. Historical durations
   stay labelled as estimates.
7. **Addenda.** An acknowledgement of the issued document revisions and
   addenda, and of authorised offer acceptance, goes inside the existing
   returnables. There is no tenth schedule.
8. **Client procurement defaults** (O8): insurance, ISO or an equivalent
   management system, weights, validity and meeting counts are labelled
   with their sources as editable client defaults.
9. **The selection rule moves into the skeleton:** a scope position is
   selected from systems and risks; scale only suggests quantities and
   review depth; every X needs a named responsible party or shows as
   unallocated.
10. **Page policy text** waits for O1.

### C3: Discipline contradictions

| Discipline | Repair |
|---|---|
| All | One canonical item per scope position, so nothing is both included and excluded. Re-tag every position as `technical` (selected by trigger) or `service_level` (may use scale). Every X names a responsible party. Fee basis set per row. Stage bundles cut by boundary |
| Architecture | Split tender documentation into traditional, D&C design-intent/performance, and post-novation completion, each with deliverables and design responsibility at handover. Resolve the tenant fitout / FF&E stage in §3 against the "tenant-specific interiors, loose furniture" X in §5. Replace "hourly capped" and rate × cap with `time_to_ceiling` |
| Structural | Replace the class-only regulated-design rule with the O5 decision. Shoring, services fixings and miscellaneous metalwork become trigger-driven (excavation, services loads, metalwork present), each with a named alternative party when excluded. Remove the warehouse-specific defaults |
| Civil | Stages 1-3 cut by boundary. The 70% tender stage applies only to the route that uses it. Split drainage modelling, water-quality modelling (MUSIC) and catchment-specific assessment (NorBE), each triggered by the brief or an authority requirement. Make the flood branch reachable on its own trigger, not behind the complex flag. Name the pavement designer and the geotech input. Fix the element grid prose (which stages use element subtotals and which use one stage total) and omit inactive elements |
| Town planning | A complete stage bundle per supported pathway. Advice-only has no lodgement or determination stage. Modification, REF, planning proposal and CDC advice are separate bundles or are marked unsupported (O6). Panel attendance, submissions response and development-standard variations each get their own trigger and count; heritage alone does not trigger them |
| Traffic | Separate parking and access advice, the traffic report, surveys and intersection modelling (SIDRA); SIDRA appears only when modelling is triggered. A green travel plan follows a confirmed requirement, not a client opt-in that could drop a known condition. Road-safety audit commissioning names its scope and its independence from the design role |

### C4: Seed-backed modules (all `status: draft`)

Author the modules in review §"Seed-backed additions" and §"Shared modules
for work type and use" as short conditional rows. Each has an activation
condition, a concrete deliverable, a source and `provenance: seed`:

- **Architecture:** brief gaps, input schedule and responsibility matrix;
  D&C tender with or without novation; existing, occupied or staged work;
  mixed-use base-building/tenant split; specialist-input coordination; the
  apartment design-quality submission; contract administration actually
  commissioned (as distinct from design support).
- **Structural:** a design-basis schedule; the Class 1/10 footing, framing
  and tie-down module; alteration and change-of-load adequacy, with
  investigation kept separate from design; excavation, adjoining and
  temporary works; a delegated-design schedule; an inspection and
  hold-point schedule. Counts are aggregated by code and confirmed by the
  engineer. The RFP never guarantees certification regardless of the
  inspection evidence.
- **Civil:** the discharge point, downstream capacity and OSD retain,
  modify or replace decision; investigation of retained drainage and
  services; earthworks with contamination, acid-sulfate soil or groundwater
  (no promise of balanced cut and fill); pavement and retaining-wall
  responsibility; a public road and authority schedule (an SEA does not
  stand in for every utility approval); civil completion.
- **Town planning:** advice and due diligence; design being tested
  (compliance matrix); conditions register; modification, CDC, SSD, REF and
  planning proposal bundles, or their unsupported state.
- **Traffic:** a loading and access vehicle schedule; counts and modelling
  scope, with representative periods for the use; review of the approved
  design when the contractor develops or changes it; CTMP preparation
  versus review, and completed car park certification.
- **Shared** (`modules.yaml`): `new`, `refurb`/`extend`, `remediation`
  (building defects and contaminated land kept separate), `advisory`, mixed
  uses (per-part scope, shared-interface deduplication), and live
  institutional or industrial sites (operator inputs; hospital, school,
  warehouse and laboratory are not interchangeable).

### C5: Driver table

Complete `docs/rfp-golden/drivers.md` from the review's preliminary table
and §2.5. For each driver, record the existing basis, the gap, and the
treatment when it is unknown. Record the mapping of RFP stage keys to
`knowledge/works/stages.yaml` delivery stages, so R7 can read the program
by delivery stage. Package stages and `DefaultStages` are not changed.

### C6: Selection answer keys

Write one YAML file per scenario in `data/eval/rfp/answer-keys/`, following
the profile answer-key convention (`data/eval/profile/answer-keys/`). Each
key lists, per discipline:

- the drivers
- expected inclusions, exclusions with their responsible party, options,
  unresolved decisions, allocation gaps and assumptions
- the source of each expectation

| Scenario | Expected proof |
|---|---|
| Arthur St extension | Existing OSD and asset interface surfaced; disputed GFA surfaced; no unsupported strengthening assumption |
| Yarramundi staged redevelopment | Active civil elements; public road, flood and catchment requirements; quantified meeting and inspection scope |
| Small Class 1 extension | Footing, framing and existing-work scope; no warehouse BIM, SIDRA or campus allowances |
| Warehouse D&C without novation | Client-side tender and performance brief; contractor completion not sold as client consultant scope |
| Planning-only appointment | No automatic DD, construction or unrelated engineering deliverables |
| Mixed Class 2/6/7a building | Part-specific clauses plus shared interfaces; no duplicated project-wide fees |
| Existing Class 3/9c repair | Regime is an O5 decision, not decided by class alone |
| Live hospital / school work | The correct operational module and operator inputs; one use's requirements not transposed to the other |
| Advisory or remedial investigation | Limitations, testing and the next decision explicit; permanent design not assumed complete |
| Unknown pathway, missing survey, contradictory facts | Focused decision or assumption; no fabricated certainty; no blank client allowance at issue |
| User changes count, position or text, then regenerates | Every override preserved; affected output and fees consistent; conflicts visible |
| Wrong org, retired source, oversized report | Existing access, isolation, versioning and failure guarantees retained (store and HTTP tests) |

The answer keys stay unreviewed until the owner approves them. Success is
measured as avoidable edits and material omissions (O7), not only as
rendering correctness.

---

## 5. Phase B: build tasks

Run Go tests from the repo root with `go test ./internal/... -run <Name>`.
Commit after every green step. Each commit message states the lane.

### Task 1: Knowledge types and loader

**Files:** Create `internal/knowledge/rfp.go`, `internal/knowledge/rfp_test.go`,
and fixtures in `internal/knowledge/testdata/rfp/`. Modify the catalogue
loader.

**Step 1: Write the failing tests** (one behaviour each):

```go
func TestLoadRFPDisciplineRejectsUnknownDriver(t *testing.T)        // when: [{colour: [red]}] → `unknown driver "colour"`
func TestLoadRFPDisciplineRejectsBadPosition(t *testing.T)          // position value Y
func TestLoadRFPRejectsUnverifiedStatutoryInWhen(t *testing.T)      // when uses a threshold of kind statutory, verified false → error; suggest: allowed
func TestLoadRFPRejectsScaleOnTechnicalItem(t *testing.T)           // item kind technical, position cell keyed by scale → error
func TestLoadRFPRejectsExclusionWithoutResponsible(t *testing.T)    // X with no responsible: → error
func TestLoadRFPRejectsItemIncludedAndExcluded(t *testing.T)        // stage lists item k; static position for k is X → error
func TestLoadRFPRequiresProvenanceKind(t *testing.T)                // evidence | seed | practice | client_decision
func TestLoadRFPConditionalCellAndIfUnknown(t *testing.T)           // {value, when, else}; if_unknown option needs assumption
```

**Step 2: Run them.** `go test ./internal/knowledge -run RFP`. Expected: FAIL, `parseRFPDiscipline` undefined.

**Step 3: Implement.** These are the core types. Keep each validation a
small function.

```go
package knowledge

// RFP discipline files turn a reviewed golden RFP into data: canonical scope
// items, the stages and positions that hold them, and the facts that switch
// them. Conditions are a closed vocabulary so the loader can reject typos,
// and an unverified statutory rule may only raise a decision, never select scope.

var RFPDrivers = map[string]bool{
	"class": true, "use": true, "work_type": true, "pathway": true,
	"procurement": true, "novation": true, "state": true, "flag": true,
	"regulated_design": true, "scale": true,
	"budget_over": true, "site_area_over": true, "parking_over": true,
}

type RFPWhen []map[string][]string // OR of ANDs; empty = always; three-valued at resolve

type RFPProvenance struct {
	Kind string `yaml:"kind"` // evidence | seed | practice | client_decision
	Ref  string `yaml:"ref"`  // "architecture#38", seed path#anchor, ...
	Why  string `yaml:"why"`  // required for client_decision
}

type RFPItem struct {
	Key         string          `yaml:"key"`
	Label       string          `yaml:"label"`
	Kind        string          `yaml:"kind"` // technical | service_level
	Provenance  []RFPProvenance `yaml:"provenance"`
}

type RFPCell struct {
	Value string  `yaml:"value"`
	When  RFPWhen `yaml:"when"`
	Else  string  `yaml:"else"`
}

type RFPStage struct {
	Key           string          `yaml:"key"`
	Name          string          `yaml:"name"`
	Items         []string        `yaml:"items"`
	Deliverable   string          `yaml:"deliverable"`
	Responsibility string         `yaml:"design_responsibility"` // at handover, D&C/novation stages
	FeeBasis      string          `yaml:"fee_basis"`             // fixed | time_to_ceiling | per_unit | option
	DeliveryStage string          `yaml:"delivery_stage"`        // knowledge/works/stages.yaml id, for R7
	When          RFPWhen         `yaml:"when"`
	Suggest       []map[string][]string `yaml:"suggest"`
	IfUnknown     string          `yaml:"if_unknown"` // ask (default) | option | include
	Assumption    string          `yaml:"assumption"`
	Module        string          `yaml:"module"`
	ShownWhen     string          `yaml:"shown_when"`
	Provenance    []RFPProvenance `yaml:"provenance"`
}

type RFPAllowance struct {
	Key, Label, Text, Unit, Default, FeeBasis string
	Owner      string          `yaml:"owner"` // client | consultant
	When       RFPWhen         `yaml:"when"`
	Provenance []RFPProvenance `yaml:"provenance"`
}

type RFPPosition struct {
	Item        string  `yaml:"item"`
	Simple      RFPCell `yaml:"simple"`  // service_level items only
	Complex     RFPCell `yaml:"complex"` // service_level items only
	Value       RFPCell `yaml:"value"`   // technical items: one cell with when/else
	Responsible string  `yaml:"responsible"` // required when any cell can be X
	When        RFPWhen `yaml:"when"`
	IfUnknown   string  `yaml:"if_unknown"`
	Assumption  string  `yaml:"assumption"`
}

type RFPThreshold struct {
	Key, Text, Source string
	Kind     string `yaml:"kind"` // statutory | client_default | evidence_estimate
	Verified bool   `yaml:"verified"`
}
```

`RFPDiscipline` holds ID, Version, Status, Title, `DisciplineIDs`, Golden,
Thresholds, Scale (a suggestion rule plus its text), ProjectRows, Role,
Items, Stages, Elements, Allowances, Positions, Terms, Returnables and
Durations. Every field in every type above needs an explicit snake_case
`yaml` tag. `RFPAllowance` and `RFPThreshold` are shortened here, and
yaml.v3 would otherwise read `FeeBasis` as `feebasis`. The loader decodes
with `KnownFields(true)`. Remove the `Packages []string` title list: per O4,
title matching is a suggestion only, and it uses `Title` and the skeleton's
alias list.

`parseRFPDiscipline` checks:

- the id matches `^[a-z-]+$`
- the status is draft or reviewed
- keys are unique within each list
- every `when` and `suggest` key is in `RFPDrivers`
- every flag is in the skeleton's `flags:` list
- cell values are in `{I, O, X, -}`
- `fee_basis` is in `{fixed, time_to_ceiling, per_unit, option}`
- `if_unknown` is in `{ask, option, include}`, and `option` or `include`
  needs an `assumption`
- the five rejections tested above

It returns an error that names the file and the key.

Add `(*Catalog).RFP(id)`, `(*Catalog).RFPForDiscipline(disciplineID)` and
`(*Catalog).RFPSuggest(title)`. Pass strings, not `procurement.Package`, to
avoid an import cycle. Load `skeleton.yaml` and `modules.yaml` with their
own types, and every other `knowledge/rfp/*.yaml` as a discipline, from the
same place `loadReportTemplates` is called.

**Step 4: Run.** `go test ./internal/knowledge -run RFP`. Expected: PASS.

**Step 5: Commit.** `Load consultant RFP discipline files: canonical items, three-valued conditions, statutory-rule guard (lane A)`

---

### Task 2: Transcribe the skeleton, modules and architecture (after C2-C4 sign-off for architecture)

**Files:** Create `knowledge/rfp/skeleton.yaml`, `knowledge/rfp/modules.yaml`
and `knowledge/rfp/architecture.yaml`. Modify `knowledge/reports/templates.yaml`.

**Step 1: Failing test.** Load the real catalogue and assert that the
counts of items, stages, allowances and positions match the **signed-off**
`docs/rfp-golden/architecture/rfp.md`. Take the counts from that file at
sign-off; do not reuse v3's 13/10/26. Also assert that `Status == "draft"`.

**Step 2: Run.** Expected: FAIL (discipline missing).

**Step 3: Transcribe** each row, keeping the reviewed wording exactly. The
shape:

```yaml
id: architecture
version: 1
status: draft
title: Architectural services
discipline_ids: [consultant.architect]
golden: docs/rfp-golden/architecture/rfp.md
thresholds:
  - {key: budget_complex, kind: client_default, text: "Budget over $20M suggests complex service levels", source: golden prose, verified: false}
  - {key: dbp_regime, kind: statutory, text: "NSW regulated-design regime applies to the affected work", source: "DBP Act and Regulation; NSW guidance (existing Class 3/9c from 1 July 2028)", verified: false}
scale:          # suggestion for service_level items only; user can override
  text: Class 2-6 or 9 in an affected part, budget over $20M, or heritage suggests complex service levels.
  complex:
    - {class: ["2", "3", "4", "5", "6", 9a, 9b, 9c]}
    - {budget_over: [20000000]}
    - {flag: [heritage]}
items:
  - {key: coordination, label: "Coordinating and integrating consultants' work", kind: service_level, provenance: [{kind: evidence, ref: "architecture#38"}, {kind: evidence, ref: "architecture#63"}]}
  - {key: dnc_performance_brief, label: "Design-intent and performance tender documents", kind: technical, provenance: [{kind: seed, ref: "data/reference/clerk/data/seed/architectural-services-guide.md#appointment-boundary-and-procurement-route"}]}
stages:
  - key: dnc_tender
    name: D&C tender documents
    items: [dnc_performance_brief]
    deliverable: "Design-intent drawings and performance brief; retained versus contractor design duties; departures review; handover deliverables register"
    design_responsibility: "Consultant to tender issue; contractor after award"
    fee_basis: fixed
    delivery_stage: approval_documentation
    when: [{procurement: [design_construct]}]
    shown_when: Design & Construct
  - key: regulated_design
    name: Regulated design
    items: [regulated_designs]
    fee_basis: fixed
    when: [{regulated_design: ["yes"]}]
    suggest: [{state: [NSW], class: ["2", "3", 9c]}]   # raises the O5 decision only
    shown_when: Regulated-design regime confirmed for this work
positions:
  - item: bim_coordination
    simple: X
    complex: O
    responsible: not_required
  - item: community
    value: {value: O, when: [{flag: [community]}], else: X}
    responsible: client
```

Add `tpl.rfp-consultant` to `templates.yaml`. Its sections are project,
role, stages, allowances, positions, terms, evaluation and returnables, as
in revision 1; the section titles come from the signed-off skeleton.

**Step 4: Run** `go test ./internal/knowledge` and `python tools/check_knowledge.py`. Expected: PASS.

**Step 5: Commit.** `Transcribe the reviewed architecture RFP, skeleton and shared modules (lane A)`

---

### Task 3: check_knowledge.py validates `knowledge/rfp/`

**Files:** Modify `tools/check_knowledge.py` and `tools/test_check_knowledge.py`.

**Step 1: Failing tests** for:

- an rfp file at `status: reviewed` that the owner did not set (mirror the
  existing reviewed-status check)
- a threshold missing `verified` or `kind`
- a `client_default` threshold with no `source`
- a `golden:` path that does not exist
- a `client_decision` provenance with no `why`
- `data/eval/rfp/answer-keys/*.yaml` that reference unknown item or
  discipline keys

**Step 2: Run** `python -m pytest tools/test_check_knowledge.py -k rfp -p no:cacheprovider`. Expected: FAIL.

**Step 3:** Implement. Reuse the file's existing YAML and error helpers.
Write files with `newline=''` (memory: Windows CRLF).

**Step 4:** Run the tests and `python tools/check_knowledge.py`. Expected: PASS.

**Step 5: Commit.** `Check RFP discipline files and answer keys in the knowledge gate (lane A)`

---

### Task 4: Drivers from the report snapshot

**Files:** Create `internal/rfp/drivers.go` and `internal/rfp/drivers_test.go`.

**Step 1: Failing tests**

```go
func TestDriversKeepPartsAndClasses(t *testing.T)        // two parts (7b warehouse, 5 office) stay two parts with labels; no flattened class list
func TestUnknownFactStaysUnknown(t *testing.T)           // heritage Unknown → flag.heritage state unknown, not absent, not "no"
func TestFalseFactIsKnownNo(t *testing.T)                // bushfire_prone_land false → flag.bushfire known no
func TestConflictingSourcesAreConflict(t *testing.T)     // heritage true, heritage_status "none" → conflict with both values
func TestPathwayAndProcurementUseTaxonomyIDs(t *testing.T) // planning_permit, design_construct, managing_contractor pass through
func TestNoPathwayIsUnknownNotDA(t *testing.T)
func TestConsultantsCarryLifecycle(t *testing.T)         // proposed → "to be appointed"; appointed → "appointed"; cancelled omitted
func TestBudgetIsInternalOnly(t *testing.T)              // budget fact present; Text map has no budget unless disclosure is on
```

Before writing the tests, read `s.reportSnapshot` in `internal/store/` and
use the real `BriefValue.ID`/`Label` format and the real part IDs in both
the fixtures and the code.

**Step 2: Run** `go test ./internal/rfp -run Drivers`. Expected: FAIL.

**Step 3: Implement.**

```go
// Facts the RFP rules switch on, read once at refresh and frozen in the draft
// so a setting change never reloads the project. Unknown stays unknown: a
// missing fact must never quietly select an exclusion.
type Fact struct {
	Values    []string `json:"values"`
	State     string   `json:"state"`  // known | unknown | conflict
	Source    string   `json:"source"` // profile | package | default | user
	SourceRev string   `json:"source_rev,omitempty"`
}

type Part struct {
	ID, Label, Class, Use, WorkType, Basis, GFA, GFARev string
}

type DriverSet struct {
	Facts       map[string]Fact   `json:"facts"` // "pathway", "flag.flood", ...
	Parts       []Part            `json:"parts"`
	Consultants []Consultant      `json:"consultants"` // title, discipline, status
	Text        map[string]string `json:"text"`        // address, lot, quantities with basis
}
```

Map the profile keys as in revision 1, but each flag becomes its own fact
(`flag.heritage`, ...), and `top_storey_class` is dropped. Take the real
option IDs from `knowledge/profile/taxonomy.yaml` (`planning`,
`procurement_route`, `operational_constraints`). Do not rename them.

**Step 4: Run.** Expected: PASS.

**Step 5: Commit.** `Read RFP facts from the report snapshot: parts kept, unknown kept (lane A)`

---

### Task 5: Resolver

**Files:** Create `internal/rfp/resolve.go` and `internal/rfp/resolve_test.go`.

**Step 1: Failing tests.** One behaviour per test:

```go
func TestKleeneMatch(t *testing.T)                     // table: true/false/unknown across AND/OR
func TestUnknownNeverExcludes(t *testing.T)            // flood unknown → civil flood row is "ask" decision, not hidden
func TestIfUnknownOptionPrintsAssumption(t *testing.T) // OSD unknown → option line with assumption text
func TestBoundaryCutsStages(t *testing.T)              // boundary concept..schematic → DD/for-construction not included; later matching stages offered as option
func TestBoundaryUnconfirmedBlocks(t *testing.T)       // suggested boundary used for preview; blocking check present
func TestPlanningOnlyHasNoConstructionDocs(t *testing.T)
func TestDnCHasPerformanceTenderNotTraditional(t *testing.T)
func TestUnsupportedPathwayBlocks(t *testing.T)        // pathway ref with no bundle → blocking "unsupported pathway"; no SEE wording
func TestRegulatedDesignIsDecisionNotClassRule(t *testing.T) // Class 3 part, NSW → stage state "ask", why cites the threshold; driver set yes → included
func TestScaleOnlyMovesServiceLevels(t *testing.T)     // scale override changes BIM LOD / renders, not shoring
func TestUserSettingBeatsRule(t *testing.T)
func TestPerFlagOverrideLeavesOtherFlagsLive(t *testing.T)
func TestClientAllowanceBlankBlocks(t *testing.T)      // consultant allowance blank does not
func TestExclusionRoutesToResponsible(t *testing.T)    // responsible geotech with proposed package → "to be appointed"; none → "unallocated" check
func TestIncludedAndExcludedIsBlocking(t *testing.T)
func TestOptionsNotInBaseFee(t *testing.T)
func TestStaleSettingIsKeptAsCheck(t *testing.T)
func TestConflictingFactIsDecision(t *testing.T)
```

**Step 2: Run.** Expected: FAIL.

**Step 3: Implement.** `Settings` is `map[string]Setting{Value json.RawMessage, Basis json.RawMessage}`.
`Resolve` runs these steps in order:

1. Apply `driver:*` and `parts:affected` settings onto a copy of the facts,
   with source `user`. Compare each setting's `basis` with the current fact
   and record any source-changed conflicts.
2. Suggest the scale for service levels (setting, then rule), with
   `ScaleWhy` naming the matching clause and part.
3. Work out the boundary: the settings, or else the suggestion from
   procurement and novation, marked `suggested` with a blocking check.
4. Resolve each stage:
   - outside the boundary: excluded, or `option` if after the end and the
     `when` is true
   - `done`: listed in "Information provided"
   - otherwise evaluate `when` in three values; on unknown, apply
     `if_unknown`
   - evaluate `suggest` only to raise decisions
   - a `stage:` setting wins
5. Resolve elements the same way as stages.
6. Activate modules by their `when`, and merge their rows (deduplicated by
   item key).
7. Allowances: the setting, or else the default. A blank `client` allowance
   is a blocking check.
8. Positions: a `technical` item uses its `value` cell; a `service_level`
   item uses the cell for the suggested scale; then apply the setting and
   `if_unknown`. An X resolves its responsible party against the
   consultants.
9. Consistency: an item included by an `On` stage and positioned X is a
   blocking check. An option never enters the base fee.
10. Fill role, terms and project rows. An unfilled slot prints `___`; it is
    blocking if the slot is a client-defined pricing basis and advisory
    otherwise.
11. Fields and weights. Weights must total 100%, or there is a blocking
    check.
12. Any setting key that matches no row goes to `Orphans`.

Each check carries `Blocking bool`, a plain-English label, the control key
it jumps to, and "which clauses change". Keep each step a small function.
Each resolved row holds `Key/Label/Text/Value/State/Source/Why/Assumption`.

**Step 4: Run.** Expected: PASS.

**Step 5: Commit.** `Resolve RFP scope with three-valued conditions, appointment boundary and allocation checks (lane A)`

---

### Task 6: Resolved RFP → report sections

**Files:** Create `internal/rfp/sections.go` and `internal/rfp/sections_test.go`.
Modify `internal/reports/draft.go` (`Table.Display string`,
`json:"display,omitempty"`) and `internal/reports/disclosure.go`.

**Step 1: Failing tests**

- Section IDs follow the `tpl.rfp-consultant` order. The last section is
  the internal `reports.ReviewSectionID`.
- `rfp:stages` lists the included stages, then a separate "Options: price
  separately" table. The stage table has `Display: "document"`.
- The positions table is `Item | Position` for the project only, and every
  X shows "by <party>", "to be appointed" or "unallocated". The scale
  sentence above it says what scale changes ("service levels assessed as
  complex: …") and does not imply that technical scope follows it.
- Assumption rows print their `assumption` wording in full.
- The project table has one row per affected part. The budget block
  `rfp:project:budget` is absent from the issued content unless disclosure
  is on, and its basis is stripped as for `cost:line:` blocks (extend
  `IssueContent` and its test).
- Consultants print with their status.
- Block IDs are stable (`rfp:role:lead`, `rfp:terms:payment`, ...), so
  `ApplyEdits` protects wording across refreshes.
- Checks and orphans become "Before issue" blocks, with blocking checks
  listed first. Their labels are plain English ("Appointment boundary: not
  confirmed", "Flood: unknown. Changes: flood assessment stage, R1 flood
  line"), not keys.
- With the discipline at `draft`, every block has `Provisional: true` and
  basis `{"method":"rfp_discipline","discipline":"architecture","version":1,"status":"draft"}`.
- Civil R1 has `Stage | <active elements> | Total | Weeks`, inactive
  elements are omitted, and the stages that use one stage total follow the
  C3 rule.

**Step 2: Run.** Expected: FAIL.

**Step 3: Implement.** Keep one function per section: `projectSection`,
`roleSection`, `stagesSection`, `allowancesSection`, `positionsSection`,
`termsSection`, `evaluationSection`, `returnablesSection`, `reviewSection`.
Each block's `Origin` is `calculation` and its `Meaning` is `requirement`.
User settings get Origin `user`, so citations show "User input". Put the
source of each value in `Basis`.

**Step 4: Run.** Expected: PASS.

**Step 5: Commit.** `Render a resolved RFP into report sections (lane A)`

---

### Task 7: Source-fidelity snapshots, then the other four disciplines

**Files:** Create `internal/rfp/markdown.go`, `internal/rfp/golden_test.go`,
`internal/rfp/testdata/fixtures.go`, and
`internal/rfp/testdata/{arthur-st,yarramundi}/<discipline>.md`. Create
`knowledge/rfp/{town-planning,traffic,civil,structural}.yaml`, each after
that discipline's Phase A sign-off.

These snapshots prove the **transcription** is faithful. Task 8 proves
**selection** is right, independently. Matching today's prose alone would
only preserve today's errors.

These are the fixtures, from the ledgers with Phase A corrections. Drivers
that the ledgers do not establish stay unknown in the fixture.

- **Arthur St:** 145-151 Arthur St Homebush West NSW. Site 31,720 m². Two
  parts: warehouse 2,135 m² (Class 7b) and mezzanine office 215 m²
  (Class 5). Extension, DA, traditional, no novation, existing OSD. The GFA
  is disputed between sources: keep both values. Packages: Architect,
  Structural, Civil, Traffic, Town Planner.
- **Yarramundi:** youth camp, NSW, Class 3 and 9b parts, masterplan plus
  alterations, DA, flags `staged`, `multi_building` and `flood`, D&C with
  novation, live operations. Packages: Architect, Civil.

Each fixture also fixes the settings a user would have made (boundary
confirmed, client allowances set), so the snapshot shows an issuable RFP.

**Step 1: Failing test.** This is the revision 1 `TestGoldenRFPs`, run with
`fx.Settings` in place of `nil`.

**Step 2: Run.** Expected: FAIL (no files).

**Step 3:** Implement `Markdown` (headings, pipe tables and paragraphs in
the golden layout). Transcribe the four remaining discipline files. Then run
with `-update` once.

**Step 4: Owner review step (not automatable).** Open each generated
`testdata/*.md` next to the signed-off `docs/rfp-golden/<d>/rfp.md`. The
only expected differences are:

- placeholders filled from the fixture
- the project's single position column
- rows cut by boundary or `when`
- printed assumptions
- consultant statuses

Anything else is a transcription error: fix the YAML, not the golden file.
Record the comparison in `docs/evidence/2026-10-XX-rfp-golden.md`.

**Step 5: Commit**, one per discipline. `Transcribe the <discipline> RFP; source-fidelity snapshot (lane A)`

---

### Task 8: Selection answer keys

**Files:** Create `internal/rfp/answerkey_test.go`. It reads
`data/eval/rfp/answer-keys/*.yaml` (from C6).

**Step 1: Failing test.** For each scenario and discipline, run `Resolve`
and compare these **sets**, not prose: included items, excluded items with
their responsible party, options, blocking decisions, allocation gaps and
assumptions. A mismatch reports the item key and the expectation's source.
Scenarios whose proof is store or HTTP behaviour (overrides surviving
regeneration; wrong org, retired source, oversized report) are cross-linked
to their tests in tasks 9, 10 and 11 rather than duplicated here.

**Step 2: Run.** Expected: FAIL until the discipline files exist.

**Step 3:** Fix the YAML, or raise the disagreement with the owner. Never
edit an answer key to match the output without the owner's approval.

**Step 4: Run.** Expected: PASS for every discipline whose files exist. The
test skips, naming the discipline, when its file is still missing.

**Step 5: Commit.** `Selection answer keys for consultant RFPs (lane A)`

---

### Task 9: Settings table and store

**Files:** Create `internal/db/migrations/021l_report_settings.sql`,
`internal/store/report_settings.go`, `internal/store/report_settings_test.go`.
Modify `internal/store/reports.go`.

**Step 1: Migration.** This is revision 1's table plus a `basis` column and
the wider key pattern:

```sql
-- Structured RFP choices (boundary, stage state, counts, I/O/X, per-fact
-- drivers). Kept per draft version and copied forward like report_edits.
-- basis records the source value the user overrode, so a later source change
-- is shown as a conflict instead of being hidden by the override.
CREATE TABLE report_settings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 report_version_id uuid NOT NULL,
 key text NOT NULL CHECK (key ~ '^(driver|boundary|parts|stage|element|allowance|position|responsible|field|weight):[a-z0-9_.]{1,100}$' OR key = 'scale'),
 value jsonb NOT NULL CHECK (octet_length(value::text) <= 2000),
 basis jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (octet_length(basis::text) <= 4000),
 user_id uuid NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,report_version_id,key),
 FOREIGN KEY (org_id,report_version_id) REFERENCES report_versions(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,user_id) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED
);
```

Check how `021c_report_tenant_cascade.sql` and the FK sweep test
(`internal/db/migrate_fk_sweep_test.go`) expect new tenant tables to look,
and match them.

**Step 2: Failing store tests** (real Postgres, same harness as
`internal/store/reports_test.go`):

- `TestSetReportSettingReresolvesWithoutSnapshot`: assert the snapshot
  loader is not called.
- `TestReportSettingSurvivesRefreshAndIssue`
- `TestReportSettingSourceChangedShowsConflict`: set `driver:flag.flood` =
  no; change the profile flood fact; refresh. The setting still applies and
  a blocking "Changed since your setting" check appears. Keep re-stamps the
  basis.
- `TestPerFlagSettingDoesNotFreezeOtherFlags`: override flood; the profile
  then gains a heritage fact; refresh; heritage follows the profile.
- `TestIssuedSettingsImmutable`: a write to an issued version gives
  `ErrInvalidReport`, and its rows are unchanged.
- `TestReportSettingWrongOrg`: gives `ErrNotFound`, and no row is written.
- `TestReportSettingStaleVersion`: gives `ErrConflict`.
- `TestReportSettingRejectsUnknownKey`: validate the value type per prefix,
  including boundary stage keys, `parts:affected` IDs that must exist,
  `responsible:` values, and yes/no/unknown for flags.
- `TestDeleteReportSettingRestoresRule`
- `TestRetiredDisciplineVersionGoesStale`: retired or changed knowledge
  makes the draft stale through `StaleReasons`.

**Step 3: Implement**

- `reportTemplateFor(kind, pkg)` returns `tpl.rfp-consultant` only when
  `pkg.DisciplineID` maps to a discipline (O4). Otherwise it keeps the
  existing map, which per O2 is today's RFP.
- In `refreshReportOnce`, for `tpl.rfp-consultant`:
  1. run `rfp.Drivers(snap)`
  2. store `rfp_drivers`, `rfp_discipline` and its version, and the module
     and skeleton versions in `source_revisions`
  3. load the settings
  4. run `rfp.Resolve` → `rfp.Sections` → `reports.ApplyEdits` → `Cite`

  Copy `report_settings` forward with the same `INSERT ... SELECT` used for
  edits.
- `SetReportSetting(ctx, org, id, actor, key, value, version)`, in one
  transaction:
  1. lock the report row
  2. check it is a draft at the given version
  3. validate the key against the resolved controls
  4. stamp the basis from the stored fact
  5. upsert or delete
  6. re-resolve from the stored `rfp_drivers`
  7. update `sections`, `version+1` and the references, and return the
     `ReportDraft`

**Step 4: Run** `go test ./internal/store -run ReportSetting` and `go test ./internal/db`. Expected: PASS.

**Step 5: Commit.** `Store RFP settings per fact and re-resolve without reloading the project (lane A/C)`

---

### Task 10: HTTP endpoints and controls payload

**Files:** Create `internal/httpapi/report_settings.go`,
`internal/httpapi/report_settings_test.go` and `internal/rfp/controls.go`.
Modify `internal/httpapi/server.go` (routes), `internal/httpapi/reports.go`
(`getReport` adds `rfp`), and `internal/httpapi/access_sweep_test.go`.

**Step 1: Failing tests.** These are revision 1's PUT/DELETE, 400, 404, 409
and 403 cases and the access sweep, plus:

- `GET /reports/{id}` for a package with no confirmed discipline returns
  `rfp_suggestion: {discipline, from: "title"}` and the generic RFP. It
  does not switch template on a title alone.
- The controls payload puts **decisions first**:

```json
"rfp": {
  "discipline": "architecture", "status": "draft",
  "decisions": [
    {"key": "boundary:end", "label": "Appointment ends at", "blocking": true, "suggested": "for_construction", "changes": ["Tender documents", "Bidding", "R1 rows 6-8"]},
    {"key": "driver:regulated_design", "label": "Regulated design applies?", "blocking": true, "why": "Class 3 part 'Lodge', NSW; rule unverified", "options": ["yes","no","advice_recorded"]}],
  "groups": [
    {"id": "project", "title": "Project", "controls": [
      {"key": "driver:flag.flood", "label": "Flood", "kind": "tristate", "value": "unknown", "source": "profile"}]},
    {"id": "boundary", ...}, {"id": "stages", ...}, {"id": "allowances", ...},
    {"id": "positions", ...}, {"id": "issue", ...}],
  "checks": [{"label": "Client presentations: count not set", "blocking": true, "key": "allowance:client_presentations"}]
}
```

Stages outside the user's control (driven by a rule) are `locked` and show
their rule; changing the driver is clearer than a toggle that silently
fights the rule. Boundary and `stage:` state are always editable.

**Step 2-4:** Implement and run `go test ./internal/httpapi -run 'ReportSetting|AccessSweep'`.

**Step 5: Commit.** `Report settings endpoints and RFP decision-first controls (lane C)`

---

### Task 11: PDF document tables and page policy (step 3 needs O1)

**Files:** Modify `internal/reports/render/pdf.go:123-150, 209`,
`internal/reports/render/fit.go:20`, and the issue path that calls the
renderer.

**Step 1: Failing tests** in `render/pdf_test.go` and `fit_test.go`:

- A `Display: "document"` table prints without the "Saved source
  comparison" heading.
- A block with an empty label prints its text only.
- An 8-column table renders, and a 9-column table returns `ErrContent`.
- The page limit comes from the template. The hard `pages > 2` at
  `pdf.go:209` is replaced by the template's limit, and other templates
  stay at 2 / 1.5.

**Step 2: Measure.** Render the populated cases (Arthur St architecture,
Yarramundi civil, and the mixed Class 2/6/7a scenario) at `MinimumSize` or
above, with no text shrinking and no scope dropped. Report their page
counts to the owner.

**Step 3 (after O1):** Set the `tpl.rfp-consultant` limit and target from
the decision. Add a test that every C6 scenario renders within the limit.
If one does not, report it; do not raise the limit or trim scope to make it
pass.

**Step 4:** Run `go test ./internal/reports/...`.

**Step 5: Commit.** `Print RFP tables as document tables; RFP page policy from measured cases (lane A)`

---

### Task 12: Issue gate

**Files:** Modify `internal/store/report_issue.go:170`, and add tests in the
existing issue test file.

**Step 1: Failing tests**

- Issuing while `architecture.yaml` (or a module it uses) is
  `status: draft` returns `ErrReportUnreviewed`.
- With a reviewed test catalogue but an unconfirmed boundary, the issue is
  refused with that check named. The same applies to an unresolved
  regulated-design decision, an unsupported pathway, a blank client
  allowance, an item both included and excluded, and a missing agreement
  reference.
- With every blocking check cleared, the issue succeeds, no block is
  `Provisional`, and `IssueSnapshot` holds the discipline, module and
  skeleton versions, the drivers with their sources, every setting, and
  the agreement and information-provided revisions.
- The issued PDF contains no budget unless disclosure was chosen.

**Step 2-4:** Add the basis check (`method == "rfp_discipline" && status != "reviewed"`)
and a blocking-check gate that reads the resolved checks from the draft.
Reuse the existing refusal shape.

**Step 5: Commit.** `Block RFP issue on draft rules and open decisions; freeze the selection basis (lane A)`

---

### Task 13: Editor UI

**Files:** Create `web/src/RfpEditor.tsx`, `web/src/rfp.css` and
`web/tests/rfp-editor.spec.ts`. Modify `web/src/reportApi.ts` and
`web/src/Reports.tsx`.

Before writing UI code, use the `impeccable` skill for layout and states.
Follow the Profile `FieldRow` idiom (`web/src/Profile.tsx:391-482`): label,
control, source mark, Reset.

**Layout.** When `view.rfp` is present, the reader pane splits into two
columns.

The **left column, "Finish this RFP"**, is the review panel. It is sized to
the uncertainty: the selected scope is visible straight away, and a few
high-impact decisions come first. The groups, in this order:

1. *Decisions*: the blocking items, each saying which clauses it changes,
   with its control inline. Examples are the boundary, pathway, regulated
   design, conflicting facts and source-changed settings.
2. *Project*: affected parts (checkboxes per part), pathway, procurement,
   class (read-only per part, with a "change in Profile" link), flags as
   Yes / No / Unknown, budget (with the existing disclosure choice), and
   LGA / zone / Lot DP text.
3. *Appointment*: start and end stage, and the state of each stage
   (include, option, exclude, done).
4. *Service level*: a Simple / Complex segment with "why", labelled "Affects
   renders, BIM level, meeting counts", and Reset.
5. *Allowances*: number inputs, with client-defined ones marked.
6. *Scope position*: an I / O / X segment per row; X shows a responsible
   picker.
7. *Issue details*: ref, issued, closes, contact, agreement, and evaluation
   weights with a live total.
8. *Other checks* (advisory), with detail expandable.

The **right column** is the existing draft view of `draft.sections`,
unchanged, so wording edits, citations and issue still work.

Every control writes on change: on blur for text and numbers, immediately
for toggles and segments. The call is `reportApi.setting(id, key, value,
version)`, and the returned draft replaces the view. The source mark shows
"Profile / Rule / Suggested / Default / You". "You" has a Reset that calls
DELETE. A source-changed conflict shows Keep and Use new. A 409 shows the
existing concurrent-edit banner. While a write is in flight, only that
control is disabled.

In `Reports.tsx`, the package `<select>` shows `Architect: discipline RFP`,
`Civil: confirm discipline` (title suggestion; one click saves
`discipline_id` through the existing package PATCH) or
`Geotechnical: generic RFP`.

**Step 1: Failing Playwright test.** This is revision 1's flow, preceded by
confirming the discipline and the boundary:

```ts
test("finish an architect RFP: decisions first, then toggles and counts", async ({ page }) => {
  await openReports(page);
  await page.getByLabel("Report type").selectOption("rfp");
  await page.getByLabel("Services package").selectOption({ label: /Architect/ });
  await page.getByRole("button", { name: "Create RFP draft" }).click();
  await page.getByRole("button", { name: "Assemble draft" }).click();
  await expect(page.getByRole("heading", { name: "Decisions" })).toBeVisible();
  await page.getByRole("button", { name: "Confirm appointment: Concept to For-construction" }).click();
  await page.getByRole("radio", { name: "Flood: No" }).check();
  await page.getByRole("combobox", { name: "Construction administration" }).selectOption("include");
  await expect(page.getByRole("cell", { name: "Construction administration" })).toBeVisible();
  await page.getByLabel("Design meetings per stage").fill("4");
  await page.getByLabel("Design meetings per stage").blur();
  await expect(page.getByText("4 per stage")).toBeVisible();
  await page.getByRole("radio", { name: "BIM coordination meetings: excluded" }).check();
  await page.getByRole("button", { name: "Refresh draft" }).click();
  await expect(page.getByRole("radio", { name: "BIM coordination meetings: excluded" })).toBeChecked();
  await expect(page.getByRole("radio", { name: "Flood: No" })).toBeChecked();
});
```

**Step 2:** Run `cd web && npx playwright test rfp-editor`. Expected: FAIL.

**Step 3:** Implement. Keep `RfpEditor` as one file with small components:
`Group`, `Decision`, `Toggle`, `Count`, `Segment`, `TriState`,
`Responsible`, `SourceMark`.

**Step 4:** Run the new spec, plus `reports.spec.ts` and
`report-issue.spec.ts`. Expected: PASS. The old RFP flow still passes for a
non-golden package.

**Step 5: Commit.** `RFP editor: decisions first, then toggles, counts and I/O/X (lane B)`

---

### Task 14: Speed bench and driver report

**Files:** Modify `bench/budgets.json`, `cmd/intake-bench/reports.go`,
`internal/latency/gate_test.go`, and `internal/httpapi/speed.go` (add
`pathReportSetting`).

**Step 1:** Add the budget `{"name": "report_setting", "p50_us": 100000, "p90_us": 250000}`.

**Step 2:** The bench uses **populated** RFPs, not short empty templates:
the Arthur St and Yarramundi fixtures, the mixed Class 2/6/7a scenario, and
a large case (all five disciplines, every module active). For each one,
time:

- `report_assemble`
- 20 alternating `PUT /settings/position:...` calls and one DELETE
  (`report_setting`)
- `report_read`
- issue with PDF rendering (`report_issue`)

Run the same set while filing runs at the same time, and confirm that
filing stays within p50 ≤ 1 s and p90 ≤ 2 s.

**Step 3:** Run it the way `docs/developing.md` describes (the "speed check"
step), under the repository's existing gate and target-host policy.
Expected: every path within budget. Report the numbers. Do not infer them
from short templates.

**Step 4:** Add `go run ./cmd/intake-bench -rfp-drivers <project>`, or a
test helper, that prints for each scenario and discipline:

- the facts filled from the profile
- the facts left to settings
- the facts still unknown
- the facts set in more than one RFP (O9)

Paste the output into the evidence file from task 7.

**Step 5: Commit.** `Bench populated RFPs and setting changes against existing budgets (lane A)`

---

### Task 15: Retire the RFP clutter for golden disciplines

**Files:** Modify `internal/reports/rfp.go`, `internal/reports/rfp_test.go`
and `cmd/intake-bench/reports.go`.

`shapeRFP` stays for the generic path (O2) and is not called for
`tpl.rfp-consultant`. Add a test that an Architect RFP draft contains no
`work:`, `proposal:`, `gap:` or `brief:` blocks, and that its first visible
section is `1. Project`. Update `docs/plans/NEXT-WAVE-STATUS.md` with one
line.

**Commit.** `Golden-discipline RFPs no longer carry generic brief and review blocks (lane A)`

---

## 6. Order, parallelism, gates

```
Phase A:  C1 → C2 → C3 → C4 → C5 → C6        (per discipline, owner sign-off each)
Phase B:
          C2 → 1 → 3
               1 → 4 → 5 → 6 ──┬─→ 11 (O1 for step 3)
                     5 → 9 → 10 → 13
                         9 → 12
          C3/C4 (per discipline) + 1 → 2 → 7 → 8   (C6 for 8)
                         10 → 14 → 15
```

Tasks 1 and 4-6 can start once C2 is signed off. Transcription (tasks 2
and 7) for a discipline waits for that discipline's C3/C4 sign-off. Task 8
waits for C6. Concurrent sessions share this tree (memory: instant-intake
progress), so use a worktree (`superpowers:using-git-worktrees`).

**Release gates (owner):**

1. Phase A content signed off per discipline (C1-C4), and the answer keys
   approved (C6).
2. The source-fidelity comparison signed off per discipline (task 7,
   step 4).
3. The selection answer keys pass (task 8).
4. Each discipline file and module flipped to `status: reviewed` by the
   owner. Until then, issue is blocked (task 12), but drafting and editing
   work.
5. `verified: false` statutory thresholds are checked against the
   instrument before a file is reviewed. If they cannot be checked, they
   stay `suggest:`-only (O5).
6. O1 decided from the measured renders (task 11).

**Report back after each task:** timing against budget, what calls what,
what can fail, new dependencies (none expected).

---

## 7. Peer review disposition

Every recommendation in the [peer review](../rfp-golden/peer-review-2026-10-09.md)
is listed here. **Adopted** means the plan now does it. **Adapted** means it
does the substance in a different form, with the reason given. **Deferred**
means it is out of scope for this plan, with the reason given.

### Corrections before encoding

| Review item | Disposition | Where |
|---|---|---|
| Appointment boundary as a controlling input | Adopted | §2.4, C2.1, task 5, task 12 |
| D&C tender gap; percentages are not deliverables | Adopted | §2.4, C3 architecture/civil, task 5 test |
| Class-only regulated-design rule | Adopted: the statutory guard, plus O5 | §2.2, §2.5, O5, task 1 test |
| Simple/complex controlling technical items; unreachable civil flood branch | Adopted: item kinds, triggers, named alternative party | §2.4, C3, task 1 test |
| Planning pathway bundles; advice-only; REF | Adopted | §2.5 pathway row, O6, C3 planning |
| Missing information has no defined effect | Adopted: four fact states, `if_unknown` | §2.3, §2.5, task 4-5 tests |
| One canonical position per item (FF&E, pavement) | Adopted | §2.4, C3, task 1 and 5 tests |
| Fee basis per row; allowance exceedance; consultant error | Adopted | §2.4, C2.2 |
| MUSIC/NorBE and SIDRA made universal | Adopted | C3 civil/traffic |
| Complete shared terms; R6 scope | Adopted | C2.2-C2.3 |
| Insurance, ISO, weights etc. as client defaults | Adopted | §2.2 kinds, O8, C2.8 |
| Three pages vs a two-page cap | Adopted: measure first, natural pagination, no shrinking or dropping | O1, task 11 |

### Provenance repairs 1-6

All adopted in C1. Item 1 is re-checked against the PDF image before the
ledger is edited, because the review spot-checked only that page.

### Seed-backed and shared modules

All adopted as draft modules in C4, including:

- the civil element-grid clarification (C3)
- separate triggers for planning panel, submissions and variations (C3)
- the traffic green travel plan and RSA independence (C3)

The review's suggested fixed clauses are used as starting wording, for
owner review.

### Preliminary drivers table

| Row | Disposition |
|---|---|
| Category vs NCC class; work-type → module mapping | Adopted: `modules.yaml`; `class` only from `ncc_class` |
| Parts and shared systems | Adopted: parts kept, `parts:affected` |
| Pathway values not in the taxonomy | Adopted: RFP-only values, unsupported until reviewed |
| Procurement enum; CM wording | Adopted: taxonomy IDs; CM mapped only on owner confirmation |
| Novation and stage position | Adopted: boundary, `stage:` state, `design_responsibility` |
| Stage taxonomy mapping | Adopted: `delivery_stage`, C5. `DefaultStages` is not changed (out of scope) |
| Occupancy, access, stakeholders | Adopted: triggers modules or decisions only |
| Site facts not uniformly available | Adopted: unknown state; flood level ≠ flood scope |
| Missing typed drivers (asset proximity, catchment, design vehicle, OSD) | Adopted: flags or settings, only where they change selected scope |
| Quantities with basis and revision; no summing incompatible areas | Adopted: per-part quantities |
| Program mapping; effort vs authority time vs input dates | Adopted: C2.6, `delivery_stage` |
| Budget disclosure | Adopted: internal driver, existing disclosure, `IssueContent` extended |
| Other consultants' status | Adopted: lifecycle status printed; unallocated check |
| Allowance model | Adopted: unit, owner, fee basis, stable key |
| I/O/X; unknown and N/A internal | Adopted. `ValidateScope` is untouched: RFP positions are report settings, not package scope |
| Selected wording and structured edits protected | Adopted: `report_edits` plus `report_settings` with basis |

### "How to make it dynamic with Jev only"

| Step | Disposition |
|---|---|
| 1 Author once: modules with ID, version, conditions, provenance | Adopted (§2.1, task 1) |
| 2 Saved facts first; Jev only where judgement is needed | **Adapted**: saved facts and settings only. Jev stays off the RFP path; any document-only fact moves to the profile pipeline in a later plan (§2.10) |
| 3 One fan-out before opening | **Adapted**: zero fan-outs on the RFP path, which is stricter; the profile pipeline already runs before (§2.10) |
| 4 Uncertainty explicit; timeout/unknown → review item | Adopted without a model: an unknown becomes a decision. With no Jev call there is no Jev timeout on this path |
| 5 Code assembles and checks consistency | Adopted (task 5 steps 6-9) |
| 6 Review proportional to uncertainty; diffs; overrides survive | Adopted (task 13 layout, §2.6 conflicts, existing `ChangesSinceIssue`) |
| 7 Freeze the issued basis | Adopted (§2.8, task 12). No model version to pin, because no model is called |

### Shared returnable refinements

All adopted in C2.3-C2.7: R1 IDs and blocks, R2 detail, R4/R7 resources
and dependencies, R6 commercial and agreement departures, the addenda
acknowledgement inside the existing returnables, and written authority for
changes.

### Repository fit and validation

| Item | Disposition |
|---|---|
| Build on reports/procurement/snapshot/edits; no second generator | Adopted (§1) |
| Bench populated, large, mixed-use; PDF; concurrent filing; existing policy | Adopted (task 14) |
| Twelve-scenario selection answer keys | Adopted (C6, task 8) |
| Avoidable edits and omissions; "80%" is an aspiration | Adopted (O7) |

### Addendum on revision 1 of this plan

| Item | Disposition |
|---|---|
| Unknown flags off | Fixed: four states (§2.5) |
| O3 blank allowances | Fixed: split by kind (O3) |
| Flattened class list | Fixed: parts kept (§2.5) |
| Whole flag-list override | Fixed: per-fact keys with basis (§2.6) |
| DA default; consultants list | Fixed: no pathway default; lifecycle status printed (§2.5) |
| Thresholds: preference vs statute; `fitout` → `refurb` | Fixed (§2.2, §2.3) |
| O4 title matching | Fixed: stable ID; title only suggests (O4) |
| O1 three-page cap | Fixed: measure first (O1, task 11) |
| Two fixtures only | Fixed: source-fidelity snapshots plus independent answer keys (tasks 7-8) |
| Source history wording | Fixed (§0 Why) |
| One confirmed source for shared facts | **Deferred** to a later plan (O9). The driver report flags duplicates now |

### Recommended next content pass

Steps 1-4 are Phase A (C1-C6). For step 5 ("write the implementation plan
after that content review"), this revision **adapts** the order. Phase B is
written now so the owner can see the data shape the content must fit, but
transcription and answer-key tasks are gated on Phase A sign-off, and the
early Phase B tasks depend only on decisions already settled here. If the
owner prefers the review's strict order, run Phase A to completion and
revise §5 afterwards. The Reports-to-Procurement navigation question is out
of scope (§0).
