# Phase A content changes (draft, for owner review)

Date: 10 October 2026. Status: draft. Nothing here is reviewed.

The [peer review](peer-review-2026-10-09.md) asked for a content pass before
the golden drafts were encoded. That pass was done directly in the data
files under `knowledge/rfp/`, which are now the working copy of each
discipline's RFP. The `rfp.md` drafts in this folder are left as the
owner-approved record they were; this note lists every change from them, so
the owner can accept, reject or adjust each one. Every changed row carries a
`provenance` entry in the YAML; a deliberate client choice is marked
`client_decision` with its reason.

To see a draft as a consultant would, read the Markdown snapshots in
`internal/rfp/testdata/` (Arthur St, five disciplines; Yarramundi,
architecture and civil). Selection is proved separately by the twelve answer
keys in `data/eval/rfp/answer-keys/`.

## C1 Provenance repairs

| Review item | Outcome |
|---|---|
| 1. PACE fee transcription | **Not changed: the review was wrong.** `Architecture/Arch RFP3.pdf` page 1 (printed page 6) was re-read as an image at 300 dpi: Stage 1 $25,500, Stage 2 $20,000, total $45,500 excl. GST, GST $4,550, $50,050 incl. GST. The ledger is correct. |
| 2. Planning modification references | Fixed in `town-planning.yaml`: the modification row now cites the planning seed and practice, not ledger #62-63 (design stage N/A, design change) or #54 (expert witness). |
| 3. Civil design criteria `S p1` | Fixed: cited as `civil#65`. All rows use discipline-qualified IDs (`architecture#38`). |
| 4. Inherited source arithmetic | No calculation copies a source total. The ledger's labels are unchanged. |
| 5. Deliberate changes | Marked `client_decision` with reasons: architecture certifier liaison, document-system uploads, sustainability documentation, tenant fit-out; structural services fixings and metalwork; civil D&C tender deliverable. |
| 6. Scope beyond the evidence | Architecture design support during construction, contract administration and post-novation, structural construction phase and post-novation, traffic car park certification, CTMP, travel plan and developed-design review, and town planning post-approval all cite practice or seed sources. |

## C2 Shared skeleton (`skeleton.yaml`, `modules.yaml`)

- **Appointment boundary.** Each discipline suggests the stages an appointment
  buys (traditional, D&C and advice-only suggestions). The user confirms them
  before issue. Stages after the last one that apply are priced as options,
  never added to the base fee.
- **Four fact states.** Every flag is yes, no, not known or conflicting. A
  row whose condition is not known either raises a decision (the default) or
  prints its stated pricing assumption (`if_unknown: option` or `include`).
  Nothing becomes an exclusion because a fact is missing.
- **Fee basis per row:** fixed, time to a ceiling, rate per unit, or option.
  "Hourly capped" and "rate × cap" are gone.
- **Complete shared terms** (written out, not "Shared"): engagement under the
  named agreement; written authority for extra scope, work beyond an
  allowance or a fee change; sign-off sets a baseline and design development
  is not a variation; correcting your own errors is not a variation; payment;
  rates; validity; questions and numbered addenda.
- **Returnables.** R1 separates base fee and options and asks for addenda to
  be acknowledged. R2 covers format, review rounds, recipient, editable files
  and reliance. R4 and R7 cover resources by stage and input dates. R6 covers
  departures from the agreement as well as from sections 3 to 6, with the
  fee and time effect.
- **Client procurement defaults** (evaluation weights, PI/PL, validity,
  payment days, ISO "or an equivalent system") are thresholds of kind
  `client_default` with sources. They are not presented as statutory.
- **Shared modules:** existing building, live site, remediation, advisory,
  mixed use, staged or multi-building. When a module's activation is unknown,
  its rows are left out and listed under Before issue rather than blocking.

## C3 Discipline repairs

### Architecture

- Tender documents are now three deliverables:
  - traditional tender documents;
  - D&C design-intent and performance tender documents, with design
    responsibility stated at handover;
  - post-novation completion.

  For-construction documents apply only where the route is not D&C. A D&C
  appointment without novation ends at the D&C tender, with a role line
  saying so.
- Regulated design is a decision. The "NSW and Class 2, 3 or 9c" rule
  (unverified) only raises the question; the stage follows the recorded
  answer.
- Fit-out conflict resolved: an Interiors stage covers base-building and
  common-area interiors. Tenant fit-out design is included for a fit-out
  (refurb) job and optional otherwise. Loose furniture and FF&E is optional.
- Construction-phase work is split into two stages:
  - design support during construction (inspections, RFIs, shop drawings),
    an option;
  - contract administration (Superintendent duties), offered only as an
    option until the client decides.
- New rows from the seed modules:
  - brief gaps and input schedule;
  - specialist-input coordination;
  - existing-building interfaces;
  - live-site staging inputs;
  - mixed-use class and interface map;
  - apartment design-quality submission (an option for a Class 2 part in
    NSW).
- Technical items follow facts, not class: community consultation, Country,
  sustainability documentation and extra declarations each have their own
  trigger.
- Labels were shortened to fit three pages. Their meaning is unchanged.

### Structural

- Regulated design: a decision, as for architecture.
- Shoring and piled retention, temporary-works interfaces, dilapidation and
  super-flat floors follow the excavation and use facts.
- Services fixings and miscellaneous metalwork are always priced options
  (v1 excluded them on simple jobs).
- Temporary works and propping design is excluded and named as the
  contractor's temporary works designer's.
- The existing structure assessment is separate from strengthening design,
  which is an option pending the assessment.
- The authority asset (SEA) and rail stage follows its own flag.
- New rows:
  - design-basis schedule;
  - residential footing and framing (Class 1 and 10);
  - a delegated-design schedule merged with the performance specifications;
  - an inspection schedule with hold points.

  The certificate is "based on the inspections made".
- D&C tender stage and post-novation stage added; construction documents
  only off D&C.

### Civil

- Stages are cut by the boundary.
  - **Traditional route:** detailed design for tender.
  - **D&C route:** design for the D&C tender, with the deliverable defined
    by what the tender needs rather than by "70%".
  - **Novation:** final design after novation.
- Element subtotals (A to D) apply to the design stages only; later stages
  are priced as one total.
- Drainage modelling, water quality (MUSIC) and NorBE are separate rows, each
  with its own trigger.
- Flood has its own trigger. The v1 simple column made it unreachable.
- OSD:
  - feasibility of keeping the existing system is included, with the
    review's example wording as the assumption when not known;
  - replacement design is an option pending feasibility;
  - new OSD follows council's requirement.
- Named responsibility:
  - the pavement designer is named in R4;
  - retaining-wall structural design is the structural engineer's;
  - the hydraulic-to-civil connection says who designs each side;
  - authority asset approvals are the structural engineer's.
- New rows:
  - discharge point and downstream capacity;
  - an investigation brief for retained drainage;
  - earthworks suitability coordination (no promise of balance);
  - an authority submission schedule;
  - completion with detention certification and handover support.

### Town planning

- Supported pathways: DA, SSD, and advice only (also used for exempt work).
  CDC, REF, modification, planning proposal and planning permit are marked
  unsupported. They block issue until their own stage set is written and
  reviewed, and never fall back to SEE wording.
- Advice-only work has no pre-lodgement, report, lodgement, assessment or
  determination stage. Its role line is "planning adviser".
- Submissions and panel are separate rows with their own triggers. The
  separate "Submissions and panel" stage, which duplicated two section 5 rows,
  is gone.
- Post-approval support (an option) now holds the conditions register.
- The compliance matrix and specialist report schedule is a new included row
  where an application is made.

### Traffic

- Separate stages:
  - parking and access advice;
  - surveys (per location);
  - intersection modelling (SIDRA);
  - the report.

  Surveys and modelling follow a modelling trigger and are options when
  that is unknown. Survey periods are "representative of the use", not
  school-term peaks.
- The green travel plan follows a confirmed requirement.
- The road safety audit is by an independent auditor and is an option where
  public-road works apply.
- New rows: the vehicle schedule (waste, emergency, service and operator
  vehicles, against current layouts), and a developed-design review option.
- Quantities are printed once, in the shared project table.

## Layout changes made to meet the three-page limit

Measured across all 27 scenario RFPs at body size, nothing dropped: longest
2.78 pages.

- Scope positions print as three lists (included, optional, excluded with
  the party) instead of a table row per item.
- R1 lists each option stage by name and then one line "Each optional item in
  section 5, one line each", as the golden R1 did.
- Option stages sit at the end of the stage table, marked "(option)".
- Civil elements print as a short list.
- Consultant RFP PDFs do not print the citation list; it stays in the issue
  record. Lines are set at 1.2 rather than 1.3; the type size is unchanged.

## For the owner

1. Accept or change each item above, discipline by discipline.
2. Approve the answer keys (`reviewed: true` with your name), or say which
   expectation is wrong.
3. When a discipline's rules are right, set `status: reviewed`,
   `reviewed_by` and `reviewed_on` in its YAML. Issue stays blocked until
   then.
