# Seven project stages for every discipline, Program and the Cost Plan

Date: 7 November 2026. Status: built on branch `stages/common` (see section
10). Lane A (cost figures, procurement packages, Program dates).

## 1. Owner decisions (7 November 2026)

1. Every consultant RFP prices the same seven project stages, in this order:

   | Key | Stage |
   |---|---|
   | `brief_concept` | Brief and concept |
   | `schematic_design` | Schematic design |
   | `planning` | Planning |
   | `design_development` | Design development |
   | `tender_documentation` | Tender documentation |
   | `design_finalisation` | Design finalisation |
   | `delivery` | Delivery (contract administration, construction, completion, defects) |

2. A discipline keeps its own scope lines inside each stage and leaves out a
   stage it has no work in. "Across all stages" items stay as they are.
3. Architecture: Brief + Concept → Brief and concept; Planning approval +
   Planning response → Planning; Interiors folds into Design development;
   D&C tender documents → Tender documentation (generic); Regulated design +
   Post-novation → Design finalisation; Contract administration stays.
4. Traditional route: Tender documents and Bidding → Tender documentation;
   For-construction documents → Design finalisation; Design support during
   construction → Contract administration.
5. Program and the Cost Plan move from the nine catalogue stages
   (`knowledge/works/stages.yaml`) to the same seven.

Why: every fee then sits at a known project stage, so cost management and
cashflow forecasting read one timeline across all consultants.

## 2. What exists today

- `knowledge/works/stages.yaml`: Investigation, Design (Concept design,
  Approval documentation, Construction documentation), Authorities and
  approvals, Procurement, Construction (Construction-stage inspections),
  Completion, Defects liability, and `other`. Program skeletons, Program
  fragments, package stages and the stage sync all use these ids.
- 18 RFP files in `knowledge/rfp/`, each with its own stage list. Five tag
  stages with `delivery_stage`; no code reads that tag.
- Saved data holding a catalogue id: `package_stages.stage_id`,
  `program_activities.catalogue_stage_id`.
- Saved data holding an RFP stage key: `report_settings` keys
  `stage:<key>`, `line:<key>.<line>`, `custom:<key>`, and position homes;
  schedule rows `stage:<key>` that proposal prices, cost line sources and
  invoices point at.
- Award lines carry the consultant's own stage name, not a project stage.

## 3. The catalogue

`knowledge/works/stages.yaml` becomes the seven stages, flat, plus `other`.
Novation side: the first five `pre`, Design finalisation and Contract
administration `post`.

## 4. Program

### 4.1 Fragments (activities) move stage

| Fragments | Today | New |
|---|---|---|
| Investigations (geotech, survey, contamination, acid sulfate, trees, flood, services, existing conditions) | investigation | Brief and concept |
| Concept design (1a, 2, fitout, 7b, 9b) | concept_design | Brief and concept |
| Design development (1a, 2, fitout, 7b, 9b), which sit before approval documentation today | concept_design | **Schematic design**, relabelled "Schematic design (…)" |
| Approval documentation, SSD EIS | approval_documentation | Planning |
| DA lodgement, exhibition, assessment; CDC; exempt confirmation; SSD assessment; heritage referral | approvals | Planning |
| Construction documentation, fit-out documentation: traditional, managing contractor | construction_documentation | **Design development** |
| The same: D&C and ECI (after the builder's tender) | construction_documentation | **Design finalisation** (duplicate fragments with `applies_when` on procurement route) |
| Tender documentation, tender period, evaluation, contract award, novation | procurement | Tender documentation |
| Construction certificate | approvals | Design finalisation |
| Construction activities (165), practical completion, occupation certificate, handover, defects inspection, end of defects | construction, completion, defects | Delivery |

No duration changes. Every activity keeps its library duration and label.

### 4.2 Skeleton stage orders

The order is per skeleton, as today:

- Traditional, DA/CDC/SSD, new/refurb/extend: brief_concept, schematic_design,
  planning, design_development, tender_documentation, design_finalisation,
  delivery.
- Refurb traditional CDC (documentation before the certificate): brief_concept,
  schematic_design, design_development, planning, tender_documentation,
  design_finalisation, delivery.
- D&C DA/SSD: as traditional.
- D&C CDC (builder obtains the CDC): brief_concept, schematic_design,
  design_development, tender_documentation, design_finalisation, planning,
  delivery.
- ECI DA: brief_concept, schematic_design, tender_documentation, planning,
  design_development, design_finalisation, delivery.

Stage-to-stage links stay finish-to-start in that order.

### 4.3 Effect on a generated program

The total duration is unchanged. Construction, practical completion and the
defects period now sit inside one stage, Delivery; their
activities and milestones still show. In a D&C program, construction
documentation already followed the tender, so dates do not move. In a
traditional program, nothing moves either: documentation still precedes the
tender.

## 5. Package stages, cost lines and saved programs

- New packages: `DefaultStages` gives the seven stages (Contract
  administration included).
- Migration `047_common_stages.sql` (046 is taken by the combined-engagements branch) re-points saved catalogue ids. Labels the
  user saw stay as saved:

  | Old id | New id |
  |---|---|
  | investigation, design, concept_design | brief_concept |
  | approval_documentation, approvals | planning |
  | construction_documentation | design_development |
  | procurement | tender_documentation |
  | construction, construction_stage_inspections, completion, defects | delivery |
  | other | other |

- Existing programs keep their rows. Program offers the seven stages on the
  next Generate, as today for a changed skeleton. Cost lines, invoices and
  frozen baselines are untouched: they point at package stage rows, not at
  catalogue ids.

## 6. RFPs

### 6.1 Mechanics

- Each discipline's `stages:` becomes at most seven stages keyed by the
  catalogue id, plus option-only stages that sit outside the timeline
  (standalone advisory, peer review, specialist investigation). Those keep
  their own keys and print under "Options outside the stages".
- A merged stage holds the union of the old stages' lines. Each line keeps
  its old stage's `when`, so a line shows only where its old stage did. The
  merged stage shows when any of its lines would.
- Where a merged stage mixes a fixed fee with a time-to-ceiling part
  (architecture Planning response, Bidding), the time part becomes a priced
  item inside the stage, so the stage still has one fixed fee and the
  ceiling is priced beside it.
- Boundaries (base start and end) are re-expressed in the new keys.
- Draft RFP settings are migrated in Go at start-up, once per report:
  `stage:old → stage:new`, `line:old.x → line:new.x`, `custom:old →
  custom:new` (arrays concatenated), position homes re-pointed. A line key
  that collides inside a merged stage gets its old stage key as a prefix.
- Issued RFP versions, their schedules, received proposals, priced cells,
  awards, cost lines and invoices keep their old keys. Issued history stays
  intact. A package whose RFP was issued keeps its issued stage list until a
  new draft is made.

### 6.2 Mapping per discipline

BC Brief and concept · SD Schematic design · PL Planning · DD Design
development · TD Tender documentation · DF Design finalisation · DL Delivery · OPT option outside the stages.

| Discipline | Old stage → new |
|---|---|
| Architecture | Brief and masterplan, Concept → BC; Schematic design → SD; Planning approval, Planning response → PL; Design development, Interiors → DD; Tender and construction documents (house), Tender documents, Bidding, D&C tender documents → TD; Regulated design, For-construction documents, Post-novation → DF; Design support during construction, Contract administration → DL |
| Access | S1 Inception, S2 Concept advice → BC; S3 Planning report → PL; S4 Construction approval → DD; S5 Tender support → TD; S6 Post-contract review → DF; S7 Construction/completion → DL; S8 Advisory → OPT |
| Arborist | S1 Inventory, S2 Concept/impact → BC; S3 Consent/protection → PL; S4 Tender/contractor design → TD; S5 Construction/completion → DL; S6a, S6b, S7a, S7b, S7c → OPT |
| BCA / certification | A1 Inception/schematic advice → SD; A2 Planning advice, B1 Preliminary assessment, B2 CDC assessment → PL; A3 Detailed design/tender advice → DD; B2 Construction approval assessment → DF; A4 Construction advice, B3 PC construction services, B4 Occupation → DL |
| Civil | Schematic design for planning → SD; Detailed design for tender, Design for the D&C tender → TD; Final design for construction, Final design after novation → DF; Construction phase, As-built → DL |
| Cost planning | S1 Cost basis, S2 Feasibility/options → BC; S3 Authority estimate → PL; S4 Design cost planning → DD; S5 Procurement/tender → TD; S6 Post-contract, S7 Final account → DL; S8 Standalone advisory → OPT |
| Electrical, hydraulic, mechanical | S1 Evidence, S2 Investigation → BC; S3 Concept → SD; S4 Authority/Planning → PL; S5 Design → DD; S6 Tender/Documentation → TD; S7 Post-contract/Later design → DF; S8 Support/Construction, S9 Witness/Commissioning, S10 Defects/Seasonal → DL; S11 Advisory → OPT |
| Fire services | S1 Evidence, S2 Investigation → BC; S3 Concept → SD; S4 Developed design → DD; S5 Documentation → TD; S6 Post-contract → DF; S7 Construction, S8 Commissioning → DL; S9 Advisory → OPT |
| Fire engineering | S1 Evidence/strategy → BC; S2 Early approval support → PL; S3 Performance design brief, S4 Analysis/FER → DD; S5 Tender/integration → TD; S6 Construction support, S7 Verification/occupation → DL; S8a, S8b, S9a, S9b → OPT |
| Geotechnical | G1 Brief/plan, G2 Field/laboratory, G3 Interpretation/report → BC; G4 Concept/planning → PL; G5 Design/tender support → DD; G7 Contractor advice/transfer → DF; G6 Construction/completion → DL |
| Land surveying | S1 Brief, S2 Detail, S3 Boundary, S4 Services → BC; S5 Approval/tender support → PL; S6 Set-out/monitoring, S7 Completion → DL |
| Landscape | S1 Inception, S2 Concept → BC; S3 Approval/responses → PL; S4 Consent/developed design → DD; S5 Traditional documentation → TD; S6 D&C/post-contract → DF; S7 Construction, S8 Establishment → DL; S9 Advice → OPT |
| Structural | Existing structure assessment → BC; Concept and schematic design → SD; Planning approval, Authority asset assessment → PL; Design development → DD; Tender documents, D&C tender documents → TD; Regulated design, Construction documents, Post-novation → DF; Construction phase → DL |
| Town planning | Preliminary advice → BC; Pre-lodgement, Planning report, Variation request, Lodgement, Assessment, Determination review → PL; Post-approval support → DD |
| Traffic | Parking and access advice → BC; Surveys, Modelling, Report, Assessment response, Conditions review → PL; Developed design review, Green travel plan → DD; CTMP → DF; Car park certification → DL |
| Waste | S1 Inception, S2 Operational concept → BC; S3 Operational plan, S3 C&D plan → PL; S4 Consent/design review → DD; S5 Tender support → TD; S6 Contractor review → DF; S7 Construction/completion, S8 Post-occupation → DL; S8 Advisory → OPT |

## 7. Questions for the owner

1. Consultant concept stages: services (electrical, hydraulic, mechanical,
   fire) concept goes to Schematic design above, because services join the
   DA scheme; landscape, access and cost concept go to Brief and concept.
   Keep that split?
2. Program: construction, practical completion and defects become activities
   inside Contract administration (section 4.3). Accept, or keep a Program-only
   "Construction" label for that stage?
3. Program today has no post-approval design development separate from
   construction documentation. Traditional construction documentation lands
   in Design development and D&C in Design finalisation (4.1). That leaves
   Design development with no library activity on a D&C or ECI program. The
   stage stays (packages take their stages from the program, so dropping it
   would drop the architect's DD fee) and shows undated until someone adds an
   activity or a duration is sourced. Accept?

## 8. Slices

1. Catalogue, Program fragments and skeletons, package defaults, migration
   046. Tests: Program generate per skeleton (unchanged total duration),
   package defaults, migration rehearsal on a copy.
2. Architecture RFP to the seven stages, draft-settings migration, golden and
   answer keys refreshed deliberately, alignment inventory updated.
3. The other 17 disciplines, one commit per group (services, specialists,
   approvals).
4. Browser check on STO: architecture and access RFP tables, Program, Cost
   Plan.

Out of scope: linking award lines to a project stage for cashflow (the next
step this enables), changing any duration, issued RFP versions.

## 9. Speed

No new path. Budgets in `bench/budgets.json` stand: report_read and
report_setting p50 ≤ 100 ms / p90 ≤ 250 ms, report_assemble 300 / 1,000 ms,
program_generate 300 / 800 ms, packages_read and stage_sync_write 100 /
250 ms. The settings migration runs once at start-up, not on a request.

## 10. Later owner decisions and what was built (7 November 2026)

- Stage 7 is **Delivery**, not Contract administration: broader, better
  known, and it holds the builder's construction as well as contract
  administration.
- Design and construct keeps **Design development** before the tender (to
  about 70-80%), then design finalisation under the contractor. Program
  gives D&C, ECI and managing-contractor jobs a "Design development"
  activity with no sourced duration yet; it takes the default length and
  says so until the owner gives one. Traditional jobs keep the library's
  construction documentation in Design development, so their dates do not
  move.
- Construction certificate sits in Design finalisation (D&C) and in Design
  development on a traditional route, as before.
- Interiors fold into Design development.
- **Existing projects are test data** and need no migration. Migration 047
  re-points saved Program and package stage ids only so test projects do not
  error; draft RFP choices saved under old stage keys are dropped.
- RFP editor: the "Whole stage" line is gone (the stage row keeps its own
  In/Excluded ticks), the boxed chevron is a thin caret and the stage name
  itself opens the stage, and rows are tighter.

How merged RFP stages behave:

- Each line keeps the condition of the stage it came from. While that
  condition is unknown the line stays and, where the old stage asked, the
  question is raised (`if_unknown: ask`); an old stage priced as an option
  keeps its line quietly (`include`); a regulated-design line keeps its
  suggestion (`suggest`, `threshold`). Questions are raised only for stages
  in the appointment.
- A rule that tested an old stage now tests the project stage and the old
  stage's condition. Two flag tests in one clause use `flag_all`.
- Boundaries cover every project stage their old stages fall in.
- One fee per stage: architecture Planning response and Bidding, and the
  traffic surveys, are now priced inside their stage's fixed fee.

Selection changes that follow from merging, recorded in the answer keys:
services whose old stage sat outside the appointment but now share a project
stage with scope inside it move from excluded to included (land surveying
plan-only and set-out-only, geotechnical plan-only, landscape inception,
waste inception and advisory, arborist inventory, hydraulic witness-only).
The civil as-built, which followed construction phase, is now in Delivery and
outside a design-only appointment.
