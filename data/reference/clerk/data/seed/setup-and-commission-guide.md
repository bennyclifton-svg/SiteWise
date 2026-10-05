---
tier: topic
seed_type: lifecycle
loaded_by: "task subject (mobilisation, commissioning, ready-to-start)"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
state_default: NSW
stub_depth: {owner-builder: deep, architect-pm: deep, builder: deep, d-and-c: deep}
topics: [setup, mobilisation, commissioning, risk]
summary: "The end-to-end setup and commissioning workflow between award and start on site: statutory instruments held, contract executed, evidence captured, programme baselined and the right registers opened, with deep role-specific paths for all four roles. The mandatory mobilisation seed for PMP drafting."
required_by: {create-pmp: 3}
doctrine_anchors: [§seed-consultation-discipline, §register-discipline, §decision-discipline, §evidence-discipline]
agents_anchors: [§1, §2, §3, §5, §9]
---

# Setup and commission guide

The setup and commission phase is the work done between project award and the start of construction-on-site. Its purpose is to put the project lead in a position to commence delivery with statutory instruments held, contract executed, evidence captured, programme baselined, and the right registers opened.

This seed gives the **end-to-end workflow** for setup and commissioning across all four user roles. It is called by `contract-setup-system` (the system skill that orchestrates mobilisation for the active role) and is also useful as a checklist reference at any phase-gate review.

NSW is the deep default. Non-NSW callouts appear inline. The **builder**, **owner-builder**, **architect-PM**, and **D&C** role sections are deep enough for their current cells.

## What "commissioned" means

A project is **commissioned** when the project lead can answer **yes** to all of:

- Frontmatter declared: `archetype`, `user_role`, `state`, plus role-specific items (per AGENTS.md §2).
- Brief / scope of works recorded in a form proportionate to the role.
- Statutory instruments held: licence (or owner-builder permit), HOW/HBCF (or exemption documented), LSL paid (where applicable), insurances bound.
- Head contract executed (where applicable), with special conditions and amendments documented.
- Planning pathway approval in hand (DA / CDC / exempt) and CC issued (where applicable).
- Baseline programme created and milestones recorded.
- Project folder structure populated, README frontmatter declared, evidence indexed.
- Owner (or self, for owner-builder) briefed on payment, variation, communication mechanisms and cadence.

The agent does not declare commissioning **complete** until each item is evidenced. Where an item is genuinely deferred (e.g. PV system selection late), label it explicitly as an Assumption with a follow-up action.

## Shared setup workflow — all roles

Regardless of role, the setup workflow follows the same skeleton. Role-specific additions sit on top of this skeleton.

### Step 1 — Declare overlays

The project `README.md` is the first deliverable. It declares `archetype`, `user_role`, `state`, and the other required frontmatter fields. The agent **stops and asks** if any of the three overlays is missing, blank, or `TBC` (per AGENTS.md §2). No guessing.

The project slug, client, site address, planning pathway (if known at mobilisation), and budget total are also captured here. Budget can be `TBC` at mobilisation but must be firm by concept endorsement.

### Step 2 — Open the project folder structure

Copy `03-project-template/` into the projects parent folder, rename to the project slug. Subfolders are advisory; small projects need not populate all. Empty subfolders signal where evidence will land — leave them rather than deleting.

### Step 3 — Capture the brief

In the form proportionate to the role (see role sections below). File under `00-brief-pmp/`. The brief is the basis against which all later scope is judged. A missing or vague brief makes scope changes invisible.

### Step 4 — Establish statutory instruments

Each role has a different statutory pack (see role sections below). The agent runs through the role-specific checklist, flagging missing items as Assumptions to be resolved before site possession.

### Step 5 — Execute or document the head contract

Where a head contract exists (architect-PM, builder, D&C), execution is the formal gate. Where there is no head contract (owner-builder building for themselves), the equivalent is the **self-defined scope and decision log**.

### Step 6 — Procure insurances

Builder / D&C: contract works, public liability, workers compensation. Architect-PM: PI insurance for the advisory scope (likely already in place practice-wide, but confirm currency for the project period). Owner-builder: home and contents during construction, public liability, and personal accident — owner-builder cannot hold contract works insurance in the conventional sense.

### Step 7 — Baseline the programme

Master programme drafted, baselined, filed under `06-programme/`. Key milestones recorded (design lock, planning approval, CC, construction start, slab pour, frame complete, lockup, fixing, PC, OC, DLP start, DLP end — adapt to project).

Where a PMP is used, its programme / staging section must first define the high-level project stage regime that the programme will adopt. If project evidence does not define a better regime, use this baseline:

| Stage | Baseline meaning |
|---|---|
| Stage 1 | Concept and schematic design to DA submission |
| Stage 2 | Design development |
| Stage 3 | Construction documentation and delivery |

If programme evidence suggests a more detailed or different stage regime, do not silently depart from the PMP. Record the conflict and recommend updating the PMP, programme and downstream consultant RFP / fee staging together.

For multi-dwelling, apartment, staged OC or D&C-signalled projects, do not stop at the three-stage fallback. Establish a detailed regime covering design/approvals, procurement, enabling works, structure, envelope/services/finishes, commissioning, OC, handover and DLP, then map those stages back to the baseline where useful.

### Step 8 — Open registers

At minimum: cost register, variation register, EOT register, decision register, action register, risk register, defects register (carries placeholder until first defect logged), inspection register. Each register's first row is the register being opened, per §register-discipline (ID + description + owner + status + due date + source + next action).

### Step 9 — Brief the owner / brief self

For owner-facing roles, a mobilisation briefing covers: payment mechanism (stage payments / cost plus / fee instalments), variation mechanism (written direction before work begins), communication cadence (weekly update, monthly report), escalation path. For owner-builder, the equivalent is a self-briefing recorded in the decision log so that future decisions reference an explicit framework.

### Step 10 — Confirm commissioned

Run the ready-to-start checklist (see role sections). Each item is Fact / Assumption / Judgement labelled per §evidence-discipline. Commissioning complete = all items Fact, or all Assumptions explicitly waived with documented basis.

## Role-specific setup — Builder (deep)

**This is the deep section for issue 02.** The builder section drives the `contract-setup-system` skill's primary path. Reference `role-builder.md` for the underlying builder posture.

### Brief / scope

The builder's brief is the **tender response** as accepted by the owner. The accepted tender becomes the contract basis. Filed under `00-brief-pmp/`. Key elements:

- scope of works (typically referenced through the contract's specifications and drawings — not duplicated in the brief);
- exclusions and qualifications agreed at tender;
- owner-supplied items list (with expected supply dates and risk position);
- provisional sums and PC items;
- assumptions made by the builder at tender (which become Assumptions in the project's evidence trail unless converted to Facts by owner confirmation).

The builder's brief is **not** the design — the design is the owner's responsibility (or the D&C's in that case). The brief is the commercial and scope envelope under which the builder agrees to deliver to that design.

### Statutory instrument pack — Builder (NSW)

| Instrument | When obtained | Source | Filed under | CC-blocking? |
|---|---|---|---|---|
| Contractor Licence (builder's licence) | Pre-existing | NSW Fair Trading | `00-brief-pmp/` | No (but contract is invalid without) |
| Qualified Supervisor certificate | Pre-existing | NSW Fair Trading | `00-brief-pmp/` | No (but linked to licence) |
| HBCF eligibility | Pre-existing (builder-side) | iCare HBCF | builder's practice records (referenced in project README) | No |
| HBCF per-project certificate | Before deposit and before contract signature | iCare HBCF | `00-brief-pmp/` and `04-planning-and-authorities/` | No, but contract-blocking |
| LSL Receipt | Before CC | NSW Long Service Corporation | `04-planning-and-authorities/` | **Yes** |
| Contract Works insurance | Before site possession | Insurer | `07-construction/03-insurance-bgs/` | No, but site-possession-blocking |
| Public Liability insurance (≥ $10M) | Before site possession | Insurer | `07-construction/03-insurance-bgs/` | No, but site-possession-blocking |
| Workers Compensation | Before any worker on site | icare workers insurance | `07-construction/03-insurance-bgs/` | No, but WHS-blocking |

For each row, the evidence (Certificate of Currency, receipt, certificate) is the source-of-truth document. Markdown summaries reference the evidence; do not substitute for it (per `03-project-template/README.md` §"Source-of-truth rule").

**State callout — VIC:** In VIC the statutory instrument pack differs materially from NSW:
- **HBCF / HOW → DBI:** The equivalent of NSW HBCF is Domestic Building Insurance (DBI) administered by the Victorian Managed Insurance Authority (VMIA) or approved insurers under the *Domestic Building Contracts Act 1995* (Vic). DBI is required before any deposit is taken for residential work valued at $16,000 or more. The builder must be registered as a Domestic Builder with the Victorian Building Authority (VBA) — not NSW Fair Trading.
- **LSL Receipt → CoINVEST:** VIC does not use a per-project Long Service Levy Receipt as a CC-blocking instrument. Instead, VIC building employers make ongoing CoINVEST contributions (a portable long service leave scheme). The payment mechanism and timing differ materially from the NSW LSL Receipt model — confirm obligations with the VBA and CoINVEST before advising.

If `state: VIC` and the task references HBCF, HOW, or LSL, **flag the gap** and confirm the VIC equivalent with the client's VIC construction lawyer or the VBA before proceeding.

### Head contract execution — Builder

Typical NSW residential builder uses **HIA Lump Sum (NSW New Homes)** for a new dwelling. Other forms:

- HIA Cost Plus — high-end / uncertain-scope work;
- HIA Renovation — additions and renovations;
- MBA equivalents — alternative to HIA;
- NSW Fair Trading prescribed contract — small jobs $5,000–$20,000;
- AS 4000 / AS 4902 — high-end residential with Superintendent appointed.

For the contract family the project uses, the builder must:

- review special conditions before signing (special conditions often shift risk to the builder in ways not visible from the standard form);
- record the deposit amount, the stage payment schedule, the DLP duration, the EOT clause reference, the variation form reference, the latent condition clause (if Renovation), and the contract sum;
- file the executed contract under `07-construction/02-fioa-contract/`;
- summarise the contract's key mechanisms in a project markdown summary at `00-brief-pmp/contract-summary.md` (a §evidence-discipline-labelled aid for retrieval — does not substitute for the contract itself).

Contract execution sequence:
1. HBCF certificate issued for the contract sum.
2. Contract pack assembled with special conditions, drawings, specifications.
3. Owner signs; builder signs.
4. HBCF certificate copy delivered to owner.
5. Deposit invoiced and received (within statutory cap).
6. Contract pack filed and indexed.

For contract clause coverage detail, load `contract-administration-guide.md`. This section is the **commissioning sequence**, not the clause reference.

### Pre-site mobilisation — Builder

After contract execution and before site possession, the builder commissions:

- **CC issued** (LSL paid, BASIX certificate, structural certification all upstream prerequisites — confirm each).
- **Insurances bound** (CWI, PL, workers comp).
- **Management plans drafted** — Construction Management Plan (CMP), Waste Management Plan (WMP), Traffic Management Plan (TMP) where required by council or site conditions. Filed under `07-construction/04-management-plans/`.
- **Site possession date confirmed** in writing to the owner.
- **Baselined programme** filed under `06-programme/`.
- **Subcontractor pre-engagement** — head subcontractors approached, scopes confirmed, insurances and licences requested.
- **Authority approvals lodged for utility connections** (water, sewer, electricity, gas where applicable, NBN). Lead times begin counting from lodgement.

### Ready-to-start checklist — Builder

The agent produces this checklist as the final commissioning deliverable. Each item is a row with status (Yes / No / N/A / Pending) and an evidence reference. See `role-builder.md` for the full list. The checklist is drafted to `00-brief-pmp/ready-to-start.md` with `status: draft`, owner-reviewed, then re-issued as `status: reviewed`.

### Common builder commissioning failures

- Deposit taken before HBCF issued (statutory breach).
- CC issued without LSL paid (rare, but recoverable only by paying LSL retrospectively).
- CWI sum insured to contract sum only (excludes owner-supplied items, debris removal, professional fees).
- Site possession taken before insurance binds.
- Subcontractor mobilises before licence and insurance verified.
- Management plans drafted late (after first delivery, after first concrete pour, after first wet-weather event).

## Role-specific setup - Owner-builder (deep - slice 07)

Slice 07 deepens the owner-builder path. For an owner-builder, commissioning is not head-contract execution. It is the point where the owner-builder can show: the scope is written, the permit / approval path is understood, renovation due diligence has been gathered, trades are controlled, insurance assumptions are explicit, and deferred decisions are parked in a visible queue.

### Brief / scope - Owner-builder

The owner-builder's brief is a self-defined scope of works. It is written for the owner-builder's own future use and for any trade, certifier, advisor, insurer, or buyer who later needs to reconstruct what was intended.

Minimum content:

- the dwelling or renovation scope, including inclusions and exclusions;
- the owner-builder's planned construction approach;
- which work the owner-builder will do personally;
- which work will be contracted to licensed trades;
- the budget envelope, contingency, and funding limit;
- the programme outline and live-occupancy assumptions;
- the approval path and certifier status;
- unresolved design selections and decisions;
- the first risk scan.

The self-defined brief is filed under `00-brief-pmp/` and referenced by cost, procurement, variation, decision, and handover records.

### Statutory instrument pack - Owner-builder (NSW)

| Instrument | When obtained / checked | Filed under | Setup treatment |
|---|---|---|---|
| Owner-builder permit | Before appointing the principal certifier and before starting work where the permit trigger applies | `00-brief-pmp/` and `04-planning-and-authorities/` | Required unless a documented no-permit basis applies |
| Course / competency / white card evidence | Before permit issue where required | `00-brief-pmp/` | Evidence item, not a substitute for the permit |
| DA / CDC / CC / approval path | Before construction starts | `04-planning-and-authorities/` | Gate item |
| Principal certifier appointment | Before critical-stage inspections | `04-planning-and-authorities/` | Gate item |
| BASIX certificate for alterations/additions | Where the BASIX trigger applies | `04-planning-and-authorities/` | Gate item for relevant works |
| Contractor licence checks | Before each trade starts | `05-procurement/` or `07-construction/` | One row per trade |
| Contractor HBCF certificates | Where a directly engaged contractor package triggers HBCF | `05-procurement/` or `07-construction/03-insurance-bgs/` | Required before that contractor starts or is paid |
| Warranty / sale disclosure reminder | At setup and carried to handover / DLP | `09-handover-dlp/` and decision / risk records | Required reminder |

**HOW / HBCF rule:** the owner-builder does not lodge builder-side HBCF for their own owner-builder work. The setup checklist must omit HBCF lodgement and per-project builder HBCF certificate rows for the owner-builder's own work. It must still check contractor HBCF where a directly engaged contractor's package triggers it.

**Future-buyer warranty rule:** HBCF unavailability does not remove warranty exposure. In NSW, if the property is sold within 7 years and 6 months after the owner-builder permit was issued, the sale contract needs the required consumer warning and the next immediate owner may have statutory warranty rights.

### Renovation due diligence - owner-builder primary cell

For `archetype: renovation`, the owner-builder setup pack must include or gap-report:

- survey / measure-up and existing levels;
- dilapidation record for neighbours and retained parts of the building;
- existing services locating: sewer, stormwater, water, gas, electrical, NBN, solar, HVAC;
- structural investigation and engineer input for wall removals, openings, second-storey work, underpinning, roof changes, or temporary works;
- hazardous-material check where age or evidence suggests asbestos, lead paint, mould, or contamination;
- moisture, termite, rot, roof, drainage, and subfloor condition checks;
- heritage / character / streetscape planning checks;
- BASIX alteration/addition trigger check;
- live-occupancy and staging assumptions.

Missing renovation due diligence is not neutral. It becomes an Assumption, risk row, action row, or park-for-decision item depending on what decision is blocked.

### Insurance posture - owner-builder

The owner-builder does not simply inherit the builder insurance stack. At setup, record:

- public liability insurance;
- home / construction-period cover and any exclusions for owner-builder or renovation work;
- personal accident / income protection where the owner-builder is doing site work;
- trade contractor public liability, workers compensation, contract works, and HBCF evidence where relevant;
- whether owner-supplied materials, stored materials, theft, storm, fire, and water damage are covered.

If the insurer excludes the renovation or owner-builder work, raise an escalation and park a decision: accept risk, obtain alternative cover, change delivery method, or stop.

### Park-for-decision queue

The owner-builder escalation queue is a standard register opened under `08-meetings-reporting/park-for-decision-queue.md`.

Each row uses the `Park-for-decision queue` register type from `register-row-draft.md` and must carry:

- ID (`PFD-<seq>`);
- decision required;
- owner / decision-maker;
- due date;
- source;
- options considered;
- consequence if deferred;
- next action.

Statuses are `parked`, `due`, `overdue`, `decided`, and `superseded`. Overdue rows are escalations and must surface during setup re-runs and risk reviews.

### Ready-to-start checklist - owner-builder

The owner-builder ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Self-defined brief and scope filed.
- [ ] Owner-builder permit filed, or documented no-permit basis recorded.
- [ ] Course / competency / white card evidence filed where required.
- [ ] DA / CDC / CC / principal certifier path recorded.
- [ ] BASIX alteration/addition trigger checked and certificate filed where required.
- [ ] Renovation due diligence pack filed or gap-reported.
- [ ] Trade register opened with licence, insurance, scope, HBCF, and payment checks.
- [ ] Public liability and construction-period insurance posture recorded.
- [ ] Park-for-decision queue opened.
- [ ] Warranty / future-buyer disclosure reminder opened for handover / DLP.
- [ ] First risk review date set.

The checklist deliberately omits owner-builder HBCF lodgement.

## Role-specific setup - Architect-PM (deep - slice 08)

Slice 08 deepens the architect-PM path. For an architect-PM, commissioning is not builder mobilisation alone. It is the point where the architect-PM can show: the commission is agreed, role authority is declared, the owner project brief is separate from the engagement brief, consultant coordination is visible, builder instruments are verified or gap-reported, and owner-facing escalation has a clear route.

Reference `role-architect-pm.md` for the full role posture.

### Brief / scope - Architect-PM

The architect-PM setup pack has two distinct briefs:

1. **The architect-PM engagement brief** - accepted fee proposal, executed engagement letter, scope of services, PMP (Project Management Plan), governance map, communications protocol, role declaration, and fee / service variation mechanism.
2. **The owner project brief** - the owner's brief for the building work, captured through briefing workshops or owner direction and filed separately under `00-brief-pmp/`.

Do not merge these. A change to the owner's building scope is a project decision. A change to the architect-PM's services is an engagement-scope decision.

### Role declaration - Architect-PM

The role declaration records whether the architect-PM is acting as architect, project manager, contract administrator, Superintendent, Certifier, or none of those formal roles. It must name the appointment instrument and authority boundary for each role.

Default position:

- architect-PM is client-side advisory;
- architect-PM is not the builder;
- architect-PM is not the Certifier;
- architect-PM is not the Superintendent unless expressly appointed in writing under the head contract and insured / competent for that scope.

Role ambiguity is an escalation. It is not a harmless setup gap.

### Statutory and professional pack - Architect-PM (NSW)

| Instrument / evidence | When checked | Filed under | Setup treatment |
|---|---|---|---|
| Architect registration, where relevant | Before relying on architect title / architectural services | `00-brief-pmp/` | Verify and record responsible registered architect |
| Professional indemnity insurance | Before advisory work is relied on for procurement, contract administration, or owner decisions | `00-brief-pmp/` or `07-construction/03-insurance-bgs/` | Required advisory evidence; check period, limit, exclusions |
| Engagement letter and scope of services | At appointment | `00-brief-pmp/` | Required role instrument |
| PMP / governance map / communications protocol | At mobilisation | `00-brief-pmp/` | Required setup control |
| Role declaration | At mobilisation and before formal directions / assessments | `00-brief-pmp/` | Required authority control |
| Builder licence / Qualified Supervisor | Before recommending contract signature or site start | `00-brief-pmp/` or `04-planning-and-authorities/` | Verify builder evidence; architect-PM does not hold it |
| Builder HBCF / HOW evidence | Before contract signature / deposit where required | `00-brief-pmp/` and `04-planning-and-authorities/` | Verify builder evidence; architect-PM does not lodge it |
| Builder LSL receipt / payment pathway | Before CC / site-start dependency | `04-planning-and-authorities/` | Verify builder / project evidence; architect-PM does not pay it unless separately engaged to administer payment |
| Builder insurance evidence | Before contract signature or site possession | `07-construction/03-insurance-bgs/` | Verify CWI, PL, workers comp where applicable |
| DA / CDC / CC / BASIX / certifier path | Before procurement and site-start recommendations | `04-planning-and-authorities/` | Verify or gap-report authority assumptions |

### Consultant coordination - Architect-PM

Where consultants are involved, setup opens or updates:

- consultant appointment tracker;
- consultant responsibility matrix;
- consultant advice register;
- design RFI register;
- deliverables schedule;
- certifier submission schedule;
- BASIX assessor coordination notes where sustainability commitments affect design, procurement, or handover.

Consultant advice that affects cost, time, scope, quality, compliance, or owner decisions must become a register row or owner-facing summary. It must not remain buried in correspondence.

### Builder evidence verification

The architect-PM's construction-readiness obligation is evidence verification. Before recommending contract signature, contract award, payment, or site start, the architect-PM checks:

- builder licence and Qualified Supervisor evidence;
- HBCF / HOW eligibility and per-project certificate where required;
- LSL receipt or payment pathway where CC / site start depends on it;
- executed head contract and special conditions;
- contract works insurance, public liability, and workers compensation where applicable;
- BASIX certificate, CC / CDC / DA pathway, certifier appointment, and authority prerequisites;
- management plans required by consent conditions or site constraints.

Missing evidence becomes an Assumption, action row, risk row, or owner-facing escalation depending on what it blocks.

### Procurement / award readiness

Where the architect-PM is engaged to select a head builder, setup links to `procurement-quoting-guide.md` and `procurement-evaluation-system.md` Branch B. The architect-PM:

- agrees evaluation criteria and weights with the owner before tenders close;
- runs RFI and addenda discipline for formal tenders;
- normalises scope, qualifications, and departures before comparing price;
- checks builder licence, HBCF / HOW capacity, insurance, programme, methodology, references, and BASIX / compliance method;
- gives the owner one clear recommendation with conditions of award and residual risks.

### Owner briefing - Architect-PM

At mobilisation, the architect-PM briefs the owner on:

- decision cadence and owner approval pathway;
- reporting cadence;
- budget and contingency reporting;
- procurement and contract award process;
- variation and owner-decision mechanism;
- role boundary: what the architect-PM can advise, assess, issue, certify, or approve;
- escalation route for urgent cost, time, authority, or compliance risk.

Owner briefings use stakeholder register and `owner-communication`: practical consequence first, explicit owner ask second, background below.

### Ready-to-start checklist - Architect-PM

The architect-PM ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Accepted fee proposal and executed engagement letter filed.
- [ ] Scope of services filed, including exclusions and contract-administration authority.
- [ ] PMP or equivalent governance plan filed.
- [ ] Owner project brief filed separately from the engagement pack.
- [ ] Role declaration recorded: architect / project manager / contract administrator / Superintendent / Certifier / neither.
- [ ] Architect registration evidence filed where relevant.
- [ ] PI insurance currency, period, limit, and exclusions recorded.
- [ ] Consultant scopes, responsibility matrix, and advice register opened where consultants are involved.
- [ ] Authority / certifier pathway recorded, including DA / CDC / CC, BASIX, principal certifier, and consent-condition assumptions.
- [ ] Builder licence and Qualified Supervisor evidence verified before recommendation / contract signature.
- [ ] Builder HBCF / HOW eligibility and per-project certificate verified where required.
- [ ] Builder LSL receipt or payment pathway verified where CC / site-start depends on it.
- [ ] Builder contract works, public liability, and workers compensation evidence verified where applicable.
- [ ] Executed head contract and special conditions reviewed or contract-execution gap recorded.
- [ ] Programme baseline and owner reporting cadence agreed.
- [ ] Owner escalation route confirmed: owner-facing summary with recommendation and due date.

The checklist deliberately omits architect-PM HBCF / HOW lodgement, builder-side LSL payment, builder licence, and contract works insurance as personal architect-PM obligations. These are builder evidence items the architect-PM verifies.

## Role-specific setup - D&C contractor (deep - slice 09)

Slice 09 landed the D&C path. For a D&C contractor, commissioning is the point where the project can prove two things at once: the builder pack is ready enough for site start, and the design pack is controlled enough that design responsibility, consultant scope, certifier submissions, and construction release are visible.

Reference `role-d-and-c.md` for the full role posture. Reference `role-builder.md` for the builder obligation base that the D&C contractor inherits.

### Brief / scope - D&C

The D&C setup pack has two linked briefs:

1. **Construction / commercial brief** - accepted tender response, executed head contract, special conditions, contract sum, qualifications, owner-supplied items, and stage / milestone payment basis.
2. **Design brief / PPR** - Principal's Project Requirements, design baseline, accepted design assumptions, design exclusions, design deliverables, consultant scope, and certifier submission requirements.

Do not merge these into one loose scope summary. A construction scope change and a design responsibility change may have different contractual consequences.

Minimum brief evidence:

- accepted D&C tender response;
- executed head contract and special conditions;
- Principal's Project Requirements / design brief;
- design fee basis or design-fee schedule;
- design programme;
- design responsibility matrix;
- consultant appointments / novations;
- design deliverables register;
- certifier design-submission protocol;
- current drawing and specification revision register.

### Statutory and insurance pack - D&C (NSW)

The D&C contractor carries the builder pack plus design-side instruments.

| Instrument / evidence | When checked | Filed under | Setup treatment |
|---|---|---|---|
| Builder licence and Qualified Supervisor | Before contract execution / site start | `00-brief-pmp/` | Inherited builder evidence |
| HBCF / HOW eligibility and per-project certificate | Before money is taken or work starts where required | `00-brief-pmp/` and `04-planning-and-authorities/` | Inherited builder evidence |
| LSL receipt / payment pathway | Before CC where required | `04-planning-and-authorities/` | Inherited builder / project evidence |
| Contract works insurance | Before site possession | `07-construction/03-insurance-bgs/` | Builder-side insurance |
| Public liability | Before site possession | `07-construction/03-insurance-bgs/` | Builder-side insurance |
| Workers compensation | Before workers attend site where applicable | `07-construction/03-insurance-bgs/` | Builder-side insurance |
| D&C PI insurance | Before design is relied on for contract, certifier, procurement, or construction release | `00-brief-pmp/` or `07-construction/03-insurance-bgs/` | Design liability evidence |
| Consultant PI evidence | Before relying on consultant design | `02-consultant/` or `07-construction/03-insurance-bgs/` | One row per consultant |
| Certifier submission protocol | Before design packages are issued for construction | `04-planning-and-authorities/` and `03-design/` | Required design-release control |

Contract works insurance and public liability do not evidence design liability. If CWI / PL are present but PI evidence is missing, setup records a design insurance gap.

### Design responsibility controls

Setup opens or updates:

- design responsibility matrix;
- design deliverables register;
- design RFI register;
- consultant advice register;
- certifier submission schedule;
- current drawing / specification revision register;
- design-change decision trail.

Each design responsibility row should identify the element, responsible designer, reviewer, D&C manager, deliverable, revision status, certifier submission requirement, construction hold point, due date, source, and next action.

Consultant advice that affects cost, time, scope, quality, compliance, or owner decisions must become a register row or owner-facing summary. It must not remain buried in correspondence.

### Multi-dwelling setup overlay

For `archetype: multi-dwelling`, setup must include or gap-report:

- classification basis: Class 1a attached dwelling vs Class 2;
- party-wall / fire-separation strategy and inspection hold points;
- separate metering / services strategy;
- utility and authority application status;
- infrastructure contribution and authority charge assumptions;
- OSD / stormwater and civil approvals;
- staging, subdivision / strata, OC, and handover pathway;
- first-of-type inspection strategy for repeated details;
- per-dwelling and common-area defect / handover assumptions.

Missing evidence on these topics becomes an Assumption, action row, risk row, or decision row. It is not neutral.

### Ready-to-start checklist - D&C

The D&C ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Accepted D&C tender response filed.
- [ ] Principal's Project Requirements / design brief filed.
- [ ] Executed head contract and special conditions filed.
- [ ] Builder licence and Qualified Supervisor evidence verified.
- [ ] HBCF / HOW eligibility and per-project certificate issued where required.
- [ ] LSL receipt or payment pathway verified where required before CC.
- [ ] Contract works insurance, public liability, and workers compensation evidence filed.
- [ ] D&C PI insurance evidence filed and exclusions checked.
- [ ] Consultant appointments / novations filed.
- [ ] Consultant PI evidence filed.
- [ ] Design fee basis recorded.
- [ ] Design programme baselined under `06-programme/`.
- [ ] Design responsibility matrix opened and current.
- [ ] Design deliverables register opened.
- [ ] Design RFI register opened.
- [ ] Certifier design-submission protocol agreed and filed.
- [ ] Authority / certifier path recorded.
- [ ] Management plans drafted where required.
- [ ] Subcontractor register opened.
- [ ] Risk register includes D&C design responsibility and interface risks.

For `archetype: multi-dwelling`, add:

- [ ] Class 1a attached dwelling vs Class 2 classification tested and evidenced.
- [ ] Party-wall / fire-separation strategy evidenced.
- [ ] Separate metering / services strategy evidenced.
- [ ] Infrastructure contribution / authority charge assumptions recorded.
- [ ] Staging, subdivision / strata, OC, and handover pathway recorded.

### D&C commissioning failures

- Builder pack is complete but PI insurance is missing.
- Design responsibility matrix is prepared at tender and never updated.
- Design programme is not linked to construction release dates.
- Consultant appointment excludes a scope the D&C contractor assumes is covered.
- Certifier submission protocol is not agreed before design packages are issued for construction.
- Party-wall, metering, staging, or contribution assumptions are unresolved on a multi-dwelling project.
- D&C treats normal design development as a variation without testing the PPR and contract baseline.

## Cross-cutting registers opened at commissioning

Regardless of role, the following registers are opened at commissioning. Their first row is the register being opened (per §register-discipline). Filed in the relevant folder.

| Register | Folder | First-row purpose |
|---|---|---|
| Decision register | `08-meetings-reporting/` | Records the commissioning decisions (overlays, contract form, deposit, programme baseline) |
| Action register | `08-meetings-reporting/` | Opens with the first set of follow-up actions from commissioning |
| Risk register | `00-brief-pmp/` or `08-meetings-reporting/` | Opens with the mobilisation risk scan |
| Cost register | `01-cost/` | Opens with the contract sum baseline |
| Variation register | `07-construction/06-variations/` | Empty at commissioning, opened with header row |
| EOT register | `07-construction/07-programme-eot/` | Empty at commissioning, opened with header row |
| Defects register | `07-construction/11-defects/` (later moves to `09-handover-dlp/`) | Empty at commissioning, opened with header row |
| Inspection register | `07-construction/12-reports/` | Opens with the inspection schedule from the certifier |
| Authority approvals tracker | `04-planning-and-authorities/` | Opens with CC, BASIX, LSL, BPA (where applicable), Sydney Water (where applicable), HOW/HBCF |
| Consultant appointment / status tracker | `02-consultant/` | Opens with required disciplines, evidence source, appointment status, scope stage, fee basis, programme dependency and next action |
| Consultant advice register | `02-consultant/` (architect-PM and D&C primarily) | Opens with the consultant team and their scope |
| Design responsibility matrix / register | `02-consultant/` or `03-design/` (D&C primarily) | Opens with design elements, responsible designer, reviewer, certifier submission need, and hold point |
| Design deliverables register | `03-design/` (D&C primarily) | Opens with design packages, revision status, submission status, and construction-release dependency |
| Design RFI register | `03-design/` or `07-construction/08-rfi-notices/` | Opens with unresolved design questions affecting cost, time, compliance, or release |
| Subcontractor register | `05-procurement/` or `07-construction/` | Opens with head subcontractors (builder and D&C primarily) |
| Park-for-decision queue | `08-meetings-reporting/` (owner-builder) | Opens with deferred self-decisions using `PFD-<seq>` rows |

Each register row is drafted with `register-row-draft` to enforce schema. Registers live in Excel for cost / programme / risk (numeric and time-series weight); markdown registers are acceptable for decision / action / defects (narrative weight).

## Commissioning escalations

Common commissioning-stage triggers (per §escalation-triggers):

- a required statutory instrument cannot be obtained in the planned window (HBCF declined, LSL receipt delayed, insurance broker delay);
- contract execution stalls (special conditions in dispute, owner unwilling to sign deposit form);
- planning approval delayed beyond the assumed window;
- budget breach detected at contract sum review;
- a role-declaration ambiguity (Superintendent role declined by all parties; Certifier appointment not made);
- a consultant gap discovered (no BASIX assessor engaged, no structural engineer for footing design);
- for D&C projects, a design responsibility matrix missing or stale;
- for D&C projects, PI evidence missing or excluded for design liability;
- for multi-dwelling projects, classification, party-wall, metering, contribution, or staging assumptions not evidenced before procurement or site start.

Each is routed per the role overlay's escalation table.

## Agent behaviour during commissioning

When invoked for a commissioning task (typically via `contract-setup-system`):

1. The agent confirms the §2 declaration gate has passed (otherwise stops and asks).
2. The agent loads the archetype seed + the role overlay + this guide via `seed-targeted-read`.
3. The agent runs `evidence-sweep` over the active project folder to identify statutory instruments and existing artefacts.
4. The agent drafts the commissioning checklist for the role, mapping each item to evidence found (Fact), evidence missing (Assumption + follow-up action), or evidence judged (Judgement with basis).
5. The agent uses `register-row-draft` to open the standard registers above.
6. The agent uses `markdown-draft-for-review` for narrative deliverables (brief summary, contract summary, ready-to-start checklist).
7. Every deliverable records `seed_consulted:` listing this guide, the role overlay, the archetype seed, and `contract-administration-guide.md` where contract content was referenced.

## Stub-depth ledger

All four v1 role sections are now deep enough for their primary cells. Remaining extension points are topic depth, not role-stub depth:

- **Slice 07 (owner-builder + renovation):** landed. Owner-builder setup is now deep enough for the owner-builder + renovation cell, including permit, HOW/HBCF exemption logic, warranty / sale disclosure reminder, renovation due diligence, trade management, and park-for-decision mechanics.
- **Slice 08 (architect-PM):** landed. Architect-PM setup is now deep enough for the architect-PM cell, including engagement pack, role declaration, PI / registration evidence, consultant coordination, builder instrument verification, and owner-facing escalation / voice-register mechanics.
- **Slice 09 (D&C):** landed. D&C setup is now deep enough for the D&C + multi-dwelling cell, including builder-pack inheritance, design fee, design programme, design responsibility matrix, consultant procurement, PI evidence, certifier submission protocol, and multi-dwelling setup gaps.

The shared setup workflow skeleton and the cross-cutting registers section should remain stable. Later slices should deepen topic seeds and specialist systems rather than reopen role-stub depth.

## See also

- `../00-doctrine/doctrine.md` and folder-aligned doctrine, particularly the `00-brief-pmp` section
- `../00-doctrine/doctrine.md §seed-consultation-discipline`, `§decision-discipline`, `§register-discipline`, `§evidence-discipline`, `§escalation-triggers`
- `../AGENTS.md §1`, `§2`, `§3`, `§5`, `§9`
- `role-builder.md` — the role overlay that deepens the builder section above
- `role-owner-builder.md` — loaded for owner-builder commissioning (slice 07)
- `role-architect-pm.md` — loaded for architect-PM commissioning
- `role-d-and-c.md` — loaded for D&C commissioning (slice 09)
- `new-dwelling-guide.md` / `renovation-guide.md` / `multi-dwelling-guide.md` / `ancillary-guide.md` / `small-commercial-guide.md` — archetype context
- `contract-administration-guide.md` — head contract clause coverage
- `../02-skills/atomic/seed-targeted-read.md`, `evidence-sweep.md`, `register-row-draft.md`, `markdown-draft-for-review.md`
- `../02-skills/systems/contract-setup-system.md` — the system skill that orchestrates this workflow
