---
tier: topic
loaded_by: task subject (contract setup, variation, EOT, claim, notice, clause interpretation)
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
topics: [contract-admin, variations, eot, claims, notices, commercial]
summary: "Residential and commercial contract administration across HIA, MBA, Fair Trading, ABIC and Australian Standard forms: notices, payment, variations, EOT, delays, completion, defects, security, WHS and dispute mechanisms."
required_by: {create-pmp: 4}
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §voice-and-style, §register-discipline]
seed_type: contract-reference
state_default: NSW
agents_anchors: [§1, §6, §9]
---

# Contract administration guide — NSW residential

This seed gives the **NSW residential head-contract landscape** as it applies across the four SiteWise user roles. It covers contract families (HIA, MBA, NSW Fair Trading prescribed, AS, ABIC), the residential-relevant mechanisms (payment, variation, EOT, latent conditions, notices, suspension, completion, DLP), and the discipline of clause citation.

This seed is **clause-aware but not clause-exhaustive**. Where a specific clause interpretation is needed, the project's actual executed contract (with any special conditions) is the source of truth under the §1 authority stack. This seed gives the **shape** of each contract family and the **administration discipline** that applies across them; it does not substitute for reading the executed contract.

NSW is the deep default. Non-NSW callouts appear inline where contract family use or threshold differs materially.

## Contract families in NSW residential — at a glance

| Family | Form references | Typical use | Issued by |
|---|---|---|---|
| **HIA Lump Sum** (NSW New Homes) | "HIA Building Contract — NSW New Homes" | New Class 1a, defined scope | Housing Industry Association |
| **HIA Cost Plus** | "HIA Cost Plus Contract" | Uncertain scope, high-end residential, owner-builder hybrid | HIA |
| **HIA Renovation** | "HIA Renovations Contract" | Renovations, additions, alterations to existing dwelling | HIA |
| **HIA Trade Contract** | "HIA Sub-contract" | Subcontract between builder and subcontractor | HIA |
| **MBA NSW equivalents** | MBA NSW residential building contracts | Alternative to HIA for new homes, renovations, cost-plus | Master Builders Association NSW |
| **NSW Fair Trading prescribed contract** | "Small Jobs Contract" (the official Fair Trading template) | Residential work between $5,000 and $20,000 (mandatory written contract is triggered above $5,000) | NSW Fair Trading |
| **AS 4000-1997** | General conditions of contract | High-end residential where Superintendent is appointed; rare for standard residential | Standards Australia |
| **AS 4902-2000** | General conditions of contract for design and construct | D&C residential (sits on `role-d-and-c.md`) | Standards Australia |
| **ABIC** | ABIC SW (Simple Works), ABIC MW (Major Works) suites | Rare in NSW residential; sometimes architect-administered residential | Australian Building Industry Contracts (joint Master Builders Australia and the Australian Institute of Architects) |

The **HIA Lump Sum (NSW New Homes)** is the dominant new-dwelling contract in NSW. Most residential builders use it as default; most owners encounter it as the contract their builder presents. The HIA Renovation form is the dominant renovation contract. Cost Plus is for narrower situations.

For each contract family below, citation discipline applies: when referring to a clause in a project deliverable, **cite the clause number verbatim** and quote the relevant text. Do not paraphrase a clause into a project document — paraphrase loses precision and risks creating a contradicting record.

## HIA Lump Sum (NSW New Homes) — mechanism shape

The HIA Lump Sum contract is structured as standard conditions + schedules (special conditions, particulars, drawings, specifications) + appendices (prescribed forms).

Key mechanisms (clause numbers and exact text vary by edition — current edition is referenced; the executed contract is authoritative):

### Deposit

- Statutory cap on deposit: **10% in NSW** for residential building work under the Home Building Act 1989 — but commonly limited to **5% by the contract form**. Special conditions sometimes increase to 10%; never exceed the statutory cap.
- Deposit is taken **after HBCF certificate issued**, not before.

### Stage payment schedule

- Typical default stages and percentages (the contract editable):
  - Deposit: 5%
  - Base / Slab: 10%
  - Frame: 15%
  - Lockup: 35%
  - Fixing: 20%
  - Completion: 15%
- Percentages add to 100% of contract sum.
- Each stage is defined in the contract's Definitions or Particulars — read these carefully because "lockup" in particular varies.
- Builder issues a progress claim **only after** the stage is physically achieved.

### Owner-payment timing

- Owner must pay within the contract's stated period (often 5 to 7 working days from claim).
- Late payment triggers interest and, after sustained delay, suspension rights for the builder.

### Variation mechanism

- **HIA Schedule of Variations** — prescribed form within the contract.
- Variation **must** be written, priced, and signed by both parties **before work begins**.
- The variation clause is explicit: unwritten variations are recoverable only by argument and are unlikely to succeed in dispute.
- The variation form records: variation number, description, cost (or method of pricing if cost not yet known), time impact (or none), date signed by each party.

### EOT mechanism

- Notice given within the contract's notification window (commonly within 10 working days of the cause becoming apparent, but read the executed contract — special conditions can shorten this).
- Qualifying delay events typically: inclement weather above a stated threshold (e.g. more than a stated number of wet days per month), owner-caused delay, authority delay outside builder's control, latent conditions, force majeure.
- Time bar engaged for late notice.

### Latent conditions

- HIA Lump Sum has limited latent conditions provisions (more developed in HIA Renovation). For a new Class 1a on a previously cleared site, latent conditions are rare; for knockdown-rebuild they are common (existing footings, services not on diagrams, unrecorded fill).
- Where applicable, builder gives notice and stops work in the area of the latent condition; owner directs how to proceed; the direction is treated as a variation.

### Practical completion (PC)

- PC is reached when the works are complete except for minor defects and minor omissions that do not prevent the dwelling from being used for its intended purpose.
- Builder issues a notice of PC; owner inspects with builder; defects schedule is agreed.
- PC starts the DLP.

### Defects liability period (DLP)

- HIA Lump Sum default DLP: **13 weeks** (sometimes called the "warranty period" in the contract).
- Often varied by special conditions to 26 weeks.
- During DLP, builder rectifies defects identified at PC and notified during DLP.
- **Statutory warranty** runs independently and longer: in NSW, **6 years** for major defects and **2 years** for minor, under the Home Building Act 1989. The statutory warranty is not the DLP and is not coterminous.

### Insurance, HBCF, security

- The contract requires HBCF (referenced as "home building compensation cover" or by the older "HOW" name in older forms).
- Contract works and public liability insurance are contract obligations.
- Security (bank guarantee or retention) is uncommon in HIA Lump Sum but can be added by special conditions.

### Special conditions discipline

Special conditions modify the standard form and almost always shift risk. The project lead must:
- read every special condition;
- record the effect on payment, variation, EOT, latent conditions, defects, DLP, dispute resolution;
- not assume the standard form mechanism applies where a special condition has modified it.

A frequent special-condition trap: a clause reducing the EOT notification window from the standard form's default to (say) 5 working days. The shorter window catches builders out.

## HIA Cost Plus — mechanism shape

Cost Plus pays the builder for actual cost incurred plus a margin (typically 10–20%).

Key mechanism differences from Lump Sum:

- **No stage payments.** Builder invoices periodically (typically monthly or fortnightly) for actual cost incurred in that period, plus the margin.
- **Cost substantiation.** Every cost claimed must be substantiated by invoice or wage record. Owner is entitled to inspect records.
- **No variation form** in the same sense — Cost Plus is "everything is a variation" relative to the work directed. Owner direction in writing is still required; pricing is open by definition.
- **No fixed contract sum.** Contract may state a target cost with a defined sharing arrangement above target, or simply leave the total open.
- **EOT mechanism still applies** but with reduced commercial significance because the builder is paid time costs.
- **PC, DLP** as for Lump Sum.

Cost Plus is appropriate where scope is genuinely uncertain (high-end residential with bespoke detail, complex renovation, owner-builder hybrid where some packages are taken by builder and others by owner-builder). It is **inappropriate** for a defined-scope new Class 1a — Lump Sum is the right form there.

## HIA Renovation — mechanism shape

HIA Renovation is structurally a Lump Sum variant with **explicit latent conditions provisions** and stage definitions adapted for renovations (often: deposit / preparation / structural / lockup / fixing / completion, instead of slab as the structural milestone).

Renovation-specific mechanism notes:

- **Latent conditions clause is more developed** — builder gives notice, stops work in the affected area, owner directs (treated as variation).
- **Dilapidation report becomes a contract document** — the dilapidation establishes the baseline against which neighbour damage claims are judged.
- **Existing-services investigation** — discoveries of unrecorded services often become latent conditions.
- **Owner-occupied during renovation** — special conditions often cover access, working hours, dust and noise mitigation, security of the existing dwelling.

Slice 07 (Owner-builder + renovation cell) deepens renovation-specific contract content beyond this stub.

## HIA Trade Contract — subcontract mechanism

Used between builder and subcontractor. Mirrors the head Lump Sum form in shape:

- defined scope and price;
- progress claims against milestones or measured work;
- variations in writing, before work;
- subcontractor's insurances (PL, workers comp) named in the subcontract.

Builder must hold a Trade Contract (or equivalent written subcontract) with every subcontractor before site possession. Verbal subcontracts are common in residential and a recurring dispute source.

## MBA NSW equivalents

The Master Builders Association NSW publishes contracts that mirror the HIA forms — Lump Sum, Cost Plus, Renovation, Trade. Mechanism is functionally similar; clause numbering and specific drafting differ.

**Citation discipline:** if the project uses MBA forms, cite MBA clause numbers in project deliverables. Do not substitute HIA clause numbers — they are not equivalent and the contract is the MBA form.

## NSW Fair Trading prescribed contract — small jobs

The NSW Office of Fair Trading publishes a small-jobs contract template for residential building work between $5,000 and $20,000. Above $5,000, a written contract is mandatory under the Home Building Act; the prescribed form satisfies the requirement.

Mechanically thin compared to HIA / MBA:

- shorter form;
- simpler payment mechanism (often a small deposit and a single completion payment, or two payments);
- limited variation, EOT, and dispute provisions;
- DLP is the statutory minimum.

Appropriate for short-duration small-value work (a fence, a deck rebuild, a small bathroom renovation). **Not appropriate** for a new build or a renovation of any complexity — the thin mechanism cannot administer a long-running project.

## AS 4000-1997 — general conditions of contract

AS 4000 is the standard for general construction contracting in Australia. Use in residential is uncommon and limited to:

- high-end residential where the owner appoints a Superintendent (often the architect-PM);
- residential projects of unusual scale or complexity (multi-dwelling subdivisions, custom homes at the top of the market);
- owners with prior commercial experience who prefer AS contract structure.

Mechanism differences from HIA Lump Sum:

- **Superintendent** is named; the Superintendent issues directions, assesses claims, certifies practical completion. The Superintendent is the owner's appointed representative — the builder takes instructions through the Superintendent.
- **Time bars are explicit and strictly enforced** — late notices for EOT or variation under AS 4000 are generally unrecoverable.
- **Security** — bank guarantee or retention is standard.
- **Contemporaneous claims discipline** — claims are assessed as at the time of the event, not retrospectively.

For new Class 1a, the AS 4000 mechanism is heavy. Where used, the role overlay for whoever is administering the contract (typically architect-PM as Superintendent's representative, or builder) drives the daily administration discipline.

## AS 4902-2000 — D&C variant

AS 4902 is the D&C variant. Loaded for `user_role: d-and-c` per `role-d-and-c.md`. Mechanism differences from AS 4000:

- Contractor carries design responsibility under the contract;
- Principal's Project Requirements (PPR) form the contractual scope basis;
- Design submission protocol with the Principal.

`role-d-and-c.md` covers the D&C role posture in depth, including design responsibility matrix, design programme, PI evidence, consultant procurement, and certifier submission control.

## ABIC — Australian Building Industry Contracts

ABIC suite (SW for Simple Works, MW for Major Works, EW for Early Works, BW for Basic Works) is used occasionally in NSW residential, more often where the architect is the contract administrator and there is preference for ABIC's clarity around architect-administered residential.

Mechanism in ABIC is broadly comparable to HIA but with the architect as the explicit administrator (similar to AS 4000's Superintendent). Architect issues instructions, assesses claims, certifies completion.

For a project using ABIC, cite ABIC clause numbers; do not paraphrase or substitute.

## Cross-family administration discipline

Across all contract families, the following discipline applies to the project lead's contract administration:

### Clause citation

When writing a project deliverable that references the contract:

- cite the clause number;
- quote the clause text verbatim (or the relevant portion);
- name the contract edition and date if not obvious from the project header;
- file the deliverable in a folder whose voice register is contractual (per AGENTS.md §6 — typically `07-construction/05-progress-claims/`, `06-variations/`, `07-programme-eot/`, `08-rfi-notices/`).

Example phrasing (contractual register):
> Under Clause 17.1 of the executed HIA Lump Sum Contract dated 14 March 2026, "the Builder may submit a progress claim within seven days after the end of each stage". This claim is submitted in accordance with that clause.

Counter-example (loose paraphrase, do not do):
> The contract lets the builder claim after each stage.

### Notice mechanics

A contractual notice has form requirements: addressed to the right party at the contract address, in the form the contract requires (often written, sometimes delivered or sent in a specific way), within the contract's time window. The form requirements are clause-specific.

- record the date of issue;
- record the method of delivery (email to contract email, hand delivery with signed receipt, registered post);
- save a copy in `07-construction/08-rfi-notices/`;
- log a register row with reference, date, recipient, subject, response-due-date.

### Time bars

Many contractual rights are barred by late notice:

- EOT typically time-barred under HIA Lump Sum (the precise window is in the clause; commonly 10 working days from the cause becoming apparent — but read the contract);
- Variation claims by builder for unwritten changes are not formally time-barred but are evidentially weak;
- Dispute notices are time-barred under HIA and most forms;
- Final claim under AS 4000 is time-barred at a fixed point relative to PC.

A late notice is the most common way builders lose otherwise valid claims. The §escalation-triggers anchor catches the trigger event; the contract administration discipline turns it into a timely notice.

### Variation discipline (cross-family)

Every contract family requires written variations before work begins. The mechanism varies (HIA uses Schedule of Variations form; AS 4000 uses Superintendent's directions; MBA uses MBA variation form), but the discipline is universal:

1. Owner / Superintendent / Principal issues written direction (or builder identifies a needed change and requests direction).
2. Builder prices the variation (cost + time impact).
3. Owner / Superintendent / Principal accepts in writing.
4. Work begins.
5. Builder records the variation in the variation register (§register-discipline) and the cost register.

Skipping any step is a §variation-management failure. The most common skip is step 4 starting before step 3 finishes — the work proceeds while pricing is "still being finalised", and by the time pricing surfaces, the bargaining position is gone.

### Progress claim discipline (cross-family)

For Lump Sum / Renovation contracts (HIA / MBA), the claim is against the stage payment schedule and requires:
- the stage to be physically achieved;
- trade evidence (inspections, certificates) supporting achievement;
- the claim form per the contract;
- the cover letter citing the relevant clause;
- supporting documentation list as the contract requires.

For Cost Plus, the claim is for actual costs incurred and requires invoice / wage record substantiation.

For AS 4000, the claim is contemporaneous with the period being claimed, assessed by the Superintendent.

For NSW Fair Trading prescribed and small jobs, the mechanism is simpler — typically one or two staged payments per the prescribed form.

The **assessor side** (whoever receives the claim and decides what to pay) is the subject of the `progress-claim-assessment-system` skill in slice 06. This guide gives the **issuer side** discipline and the cross-family shape.

### EOT discipline (cross-family)

1. Cause event occurs (weather, owner delay, latent condition, authority delay).
2. Builder identifies it as a qualifying delay event under the contract.
3. Builder gives notice within the contract's window.
4. Builder maintains contemporaneous evidence (programme, photos, correspondence, weather records).
5. Builder pricing time impact through the contemporaneous programme.
6. Owner / Superintendent / Principal assesses and grants (or part-grants) the EOT.
7. Builder updates programme and EOT register.

Step 3 is the failure point. Late notice = time bar = lost claim.

### Practical completion discipline (cross-family)

PC is reached when works are complete except for minor defects and minor omissions. Disputes about PC commonly arise around what is "minor". A minor defect:
- does not prevent the dwelling from being used for its intended purpose;
- is reasonably capable of rectification;
- is identified on the PC defects list.

A non-minor incomplete item (e.g. balustrade missing, kitchen joinery not installed) is not eligible to be on the PC list — it stops PC from being achieved. Builder must complete it before claiming PC.

PC starts the DLP. PC starts the statutory warranty period. PC is a non-trivial gate; do not let pressure to "call PC" rush past genuine completion.

### Defects liability — discipline

During DLP:
- defects notified by the owner are recorded in the defects register;
- builder rectifies within reasonable time;
- builder's rectification is inspected and closed in the register;
- final claim (retention release, final payment) is released after DLP expires and all defects closed.

The §register-discipline applies — every defect row has ID, description, owner (typically the trade responsible), status, due date, source / evidence reference, next action.

## Contract-family-by-role suitability

| Family | Owner-builder | Architect-PM | Builder | D&C |
|---|---|---|---|---|
| HIA Lump Sum | N/A (owner-builder has no head contract; subcontracts may use HIA Trade) | Recommends and administers on owner's behalf | Issues to owner | Possible head form |
| HIA Cost Plus | Hybrid — owner-builder may use Cost Plus for specific packages (e.g. brickwork) | Sometimes recommended for high-end / uncertain scope | Issues to owner | Possible |
| HIA Renovation | N/A | Recommends for additions to existing | Issues to owner | Possible |
| MBA equivalents | As HIA | As HIA | As HIA | As HIA |
| NSW Fair Trading prescribed | Owner-builder may use for small package subcontracts | Recommends only for genuinely small work | Issues to owner only for small jobs ($5,000–$20,000) | Rare |
| AS 4000 | N/A | Acts as Superintendent's representative if appointed | Subject to head contract if owner has selected AS form | Rare |
| AS 4902 | N/A | Possible | Possible D&C variant | **Primary form for D&C** |
| ABIC | N/A | Sometimes architect-administered residential | Subject to head contract | Possible |

## Non-NSW callouts

- **VIC:** HIA Victorian Conditions are different from NSW; *Domestic Building Contracts Act 1995* (Vic) applies (set of statutory implied terms). Builder licence is "Domestic Builder" registration with the Victorian Building Authority (VBA) — not NSW Fair Trading. **DBI (Domestic Building Insurance)** administered by the VMIA or approved insurers is the VIC equivalent of NSW HBCF / HOW; threshold is $16,000 and must be obtained before any deposit is taken. Contract forms: HIA Vic Lump Sum or Master Builders Victoria equivalents — confirm which edition is executed before citing clauses.
- **QLD:** QBCC governs residential contracts. Standard QBCC Level 1 and Level 2 contracts published. QBCC Home Warranty Insurance threshold $3,300 — much lower trigger than NSW.
- **SA:** Consumer and Business Services (CBS) governs Building Work Contractors. HIA SA forms differ.
- **WA / TAS / NT / ACT:** state-specific frameworks. Where a non-NSW project is administered, the agent flags the gap and asks the project lead to supplement.

Where `state:` is not NSW and the project uses a HIA-branded form, **the form will have state-specific variants** — confirm which edition is executed before citing clauses.

## Common contract-administration failures

- **Special conditions not read.** Builder signs the standard form without noticing that special conditions have, for example, shortened the EOT window to 3 days.
- **Variations worked before pricing.** "We'll sort it out at the end" — by the end, the bargaining position is gone.
- **Late EOT notices.** Time bar engaged.
- **PC declared while non-minor items incomplete.** DLP starts in dispute.
- **Statutory warranty conflated with DLP.** Owner expects DLP to cover what is actually a statutory warranty claim, or vice versa.
- **HIA clause numbers cited where the contract is MBA** (or vice versa). The cited clause does not exist or means something else.
- **Paraphrased clause text in project deliverables.** Paraphrase loses precision; if a dispute arises, the project record contradicts the contract.

## Agent behaviour under this seed

When this seed is loaded:

1. Confirm the project's executed contract is known. If not, flag and ask before drafting any contract-referring content.
2. Cite the actual executed contract's clause numbers and form name (e.g. "Clause 17.1 of the HIA Lump Sum NSW New Homes Contract, edition dated [X]"), not a generic reference.
3. Quote clause text verbatim where the deliverable depends on the wording.
4. Apply §evidence-discipline labelling — a clause interpretation that is the project lead's reading is Judgement; the clause text itself is Fact (from the executed contract); a position taken on the basis of the clause is Recommendation.
5. Default to contractual register per §voice-and-style for any deliverable that references the contract.

## See also

- `../00-doctrine/doctrine.md §voice-and-style`, `§evidence-discipline`, `§register-discipline`, `§escalation-triggers`
- `../AGENTS.md §1` (authority stack — the executed contract beats this seed), `§6` (voice register), `§9` (skill invocation)
- `role-builder.md` — builder-side issuance discipline for claims, variations, EOT
- `role-owner-builder.md` — owner-builder hybrid contract use (slice 07 deepens)
- `role-architect-pm.md` — architect-PM as Superintendent's representative (slice 08 deepens)
- `role-d-and-c.md` — AS 4902 and D&C contract administration
- `setup-and-commission-guide.md` — contract execution as part of mobilisation
- `new-dwelling-guide.md` — Class 1a project context for contract selection
- `../02-skills/atomic/seed-targeted-read.md` — the gate that loads this seed
- `../02-skills/systems/contract-setup-system.md` — orchestrates contract setup using this seed
- (Slice 06 future) `progress-claim-assessment-system`, `variation-management-system` — assessment side of the issuance discipline this seed establishes

# Commercial and Non-Residential Expansion

## Contract Selection and Risk Allocation

Selecting the appropriate contract form is one of the most consequential decisions made in the pre-construction phase of any Australian building project. The choice of contract determines how risk is allocated between the Principal and Contractor, how variations and claims are administered, the mechanisms for dispute resolution, and the overall commercial framework within which the project will be delivered. An inappropriate contract selection can embed structural risk from day one, leading to adversarial relationships, cost overruns, and protracted disputes that could have been avoided with a more considered approach.

The principal standard form contracts used in Australian commercial construction are AS 4000-1997 (General Conditions of Contract), AS 2124-1992 (General Conditions of Contract), AS 4902-2000 (Design and Construct), and ABIC MW-1 (Major Works). AS 4000 is the most widely used general conditions contract for traditional (construct-only) procurement in Australia and superseded AS 2124, although AS 2124 remains in use on some government and institutional projects. AS 4902 is the standard form for design and construct (D&C) procurement, where the Contractor assumes responsibility for both design and construction. ABIC MW-1 is published by the Australian Building Industry Contracts committee and is commonly used on projects procured by architects acting as contract administrators rather than a Superintendent.

The fundamental risk allocation decision is whether the project will be delivered on a lump sum, cost-plus, guaranteed maximum price (GMP), or managing contractor basis. In a lump sum contract, the Contractor accepts the risk of delivering the defined scope for a fixed price — design risk remains with the Principal (traditional) or transfers to the Contractor (D&C). In a cost-plus arrangement, the Principal pays the actual cost of the works plus an agreed fee or margin, retaining most cost risk but gaining transparency over actual expenditure. A GMP arrangement sits between these, providing cost transparency up to a cap beyond which the Contractor bears the risk. Managing contractor arrangements are typically used on complex or fast-tracked projects, where the managing contractor is engaged early and trade packages are let progressively as design develops.

A formal risk matrix should be developed during the procurement planning phase, identifying each material risk (latent conditions, design coordination, program, escalation, regulatory change, weather) and explicitly allocating it to the party best able to manage it. The contract form should then be selected to align with this risk allocation. Attempting to force an inappropriate risk profile into a standard form contract — for example, making a lump sum contractor responsible for undefined scope — invariably leads to inflated tender prices or post-contract claims.

Reference: AS 4000-1997; AS 2124-1992; AS 4902-2000; ABIC MW-1 2003; Standards Australia HB 220-2001 (Risk Management Guidelines for Procurement)
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4120-1994 (Code of Tendering), AS 4122-2010 (General Conditions for Engagement of Consultants)

Common pitfalls:
- Defaulting to AS 4000 lump sum for every project without considering whether the design is sufficiently advanced to support lump sum pricing
- Selecting a D&C contract to "transfer risk" without recognising that design risk transfer comes at a premium — contractors price for the unknowns they are now carrying
- Failing to develop a risk matrix before selecting the contract form, resulting in a mismatch between the intended and actual risk allocation
- Heavily amending standard form contracts with special conditions that shift risk disproportionately, leading to inflated tender prices and increased likelihood of claims
- Not considering the project team's familiarity with the selected contract form — an unfamiliar contract administered poorly is worse than a less optimal contract administered well

## Roles and Responsibilities Under Standard Form Contracts

The effective administration of an Australian construction contract depends on a clear understanding of the distinct roles, duties, and obligations of each party. Confusion about roles — particularly the dual role of the Superintendent — is one of the most common sources of dysfunction in contract administration. Each participant in the contractual chain has specific obligations that, if not discharged properly, can give rise to claims, delay entitlements, or breach of contract.

Under AS 4000-1997 and AS 2124-1992, the **Superintendent** occupies a unique and sometimes conflicting position. The Superintendent is appointed by the Principal and is usually the Principal's project manager, architect, or an independent contract administrator. However, the Superintendent has a dual function: when exercising certification and assessment functions (such as certifying progress claims, assessing extensions of time, or determining variation valuations), the Superintendent must act honestly and fairly — not as the Principal's advocate. This duty of impartiality when certifying was confirmed in *Perini Corporation v Commonwealth of Australia* (1969) and refined in subsequent case law. When the Superintendent is giving directions, issuing instructions, or making decisions on behalf of the Principal (such as directing variations or approving the construction program), the Superintendent acts as the Principal's agent. The distinction is critical: a payment certificate issued by a Superintendent who has failed to exercise independent judgment can be challenged as invalid.

The **Principal** has several express obligations under the standard form contracts. These include providing access to the site by the date specified in the contract (clause 25.1 of AS 4000), ensuring the Superintendent performs their functions (clause 23 of AS 4000), making payments in accordance with the contract, and not interfering with the Superintendent's certification functions. The Principal also has an implied obligation not to prevent or hinder the Contractor from performing the works — a breach of this obligation can give rise to the prevention principle, potentially disentitling the Principal from claiming liquidated damages.

The **Contractor** is obligated to execute and complete the works in accordance with the contract documents, comply with the construction program, rectify defects, maintain required insurances, and give all required notices within the stipulated timeframes. The Contractor must comply with all legislative requirements and is responsible for the means, methods, and sequences of construction unless the contract specifies otherwise. Under AS 4000 clause 29, the Contractor must give written notice of any claim for extra cost or time within 28 days of becoming aware of the event giving rise to the claim — failure to comply with this notice requirement may bar the claim entirely.

In the **subcontract chain**, the head contractor engages subcontractors under back-to-back terms (typically AS 4901-1998 for use with AS 4000, or AS 4903 for use with AS 4902). Subcontractors have no direct contractual relationship with the Principal. The Superintendent has no contractual authority over subcontractors — all directions must flow through the head contractor. This layered structure means that communication, particularly regarding variations and defects, must follow the contractual chain to have legal effect.

Reference: AS 4000-1997 clauses 20-23 (Superintendent), 25 (access), 29 (claims), 37 (payment); AS 2124-1992 clauses 21-23; *Perini Corporation v Commonwealth of Australia* (1969) 12 BLR 82
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4901-1998 (Subcontract for use with AS 4000), AS 4903-2000 (Subcontract for use with AS 4902)

Common pitfalls:
- The Superintendent acting solely as the Principal's advocate when certifying claims, exposing the Principal to challenges of invalid certificates
- The Principal directly instructing the Contractor or subcontractors, bypassing the Superintendent and creating confusion about the authority and contractual effect of such instructions
- The Contractor failing to give notice of claims within the 28-day time bar under AS 4000 clause 29, forfeiting otherwise valid entitlements
- Not formally appointing the Superintendent (or the Superintendent's Representative) in writing as required by the contract, creating uncertainty about who has authority to issue certificates and directions
- Subcontractors assuming they can deal directly with the Superintendent or Principal, rather than through the head contractor

## Contract Documentation Hierarchy

Every Australian standard form construction contract establishes an order of precedence among the contract documents. This hierarchy determines which document prevails when two or more documents contain conflicting requirements — a situation that arises with surprising frequency on real projects, particularly where design documentation has been produced by multiple consultants at different stages. Understanding and correctly applying the document hierarchy is fundamental to contract administration and to the fair resolution of scope disputes.

Under AS 4000-1997 (clause 2.2), the order of precedence is specified in the Annexure or the contract-specific schedules. The typical hierarchy, from highest to lowest precedence, is: (1) the Formal Instrument of Agreement, (2) the conditions of contract (general conditions as amended by special conditions), (3) the Annexure (or schedule of contract particulars), (4) the specification, (5) the drawings, and (6) the bill of quantities or schedule of rates (if any). Importantly, special conditions take precedence over the general conditions, and the specification typically takes precedence over drawings. This means that if a drawing shows a detail that conflicts with the specification, the specification prevails — a principle that is counter-intuitive to many practitioners who assume that the more detailed document should govern.

Under AS 2124-1992 (clause 8), the hierarchy is similar but the contract allows for the Superintendent to direct which document prevails in the event of an ambiguity or discrepancy. The Superintendent must issue this direction in writing, and the Contractor is entitled to a variation if the direction requires additional work beyond what a reasonable contractor would have included in their tender based on a reasonable reading of the contract documents. This provision incentivises careful document preparation but provides a safety valve when conflicts inevitably arise.

Amendments to standard form contracts are made through special conditions, which are project-specific clauses that modify, supplement, or delete provisions of the general conditions. Special conditions are common and often necessary, but poorly drafted amendments can create unintended consequences, internal contradictions, or ambiguities that generate disputes. Best practice is to use a tracked changes approach to special conditions, making it clear which general condition clauses have been amended and how. The Australian Construction Industry Forum (ACIF) and the Australian Procurement and Construction Council (APCC) have published guidelines discouraging excessive amendment of standard form contracts, noting that heavily amended contracts increase legal costs and dispute risk.

Reference: AS 4000-1997 clause 2 (interpretation and documents); AS 2124-1992 clause 8 (discrepancies); ACIF/APCC Best Practice Principles for the Selection and Management of Construction Contracts
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4902-2000 clause 2 (D&C contract documentation)

Common pitfalls:
- Not establishing a clear order of precedence in the contract, or leaving the default hierarchy in place without considering whether it is appropriate for the project
- Allowing conflicts between drawings and specifications to go unresolved during the tender period, creating scope ambiguity that generates post-contract disputes
- Drafting special conditions that contradict each other or inadvertently override important protections in the general conditions
- Failing to issue all contract documents to the Contractor as a complete set at contract execution, leading to disputes about which version of a document forms part of the contract
- Not maintaining a register of all contract documents and their revision status, making it difficult to determine the authoritative version of any given document

## Contract Correspondence and Notices

Formal correspondence and notices are the lifeblood of contract administration. In construction law, "you cannot claim what you did not notify" is a near-universal truth. The standard form contracts contain numerous provisions that require written notice within specified timeframes as a condition precedent to contractual entitlements. Failure to comply with these notice requirements can extinguish an otherwise legitimate claim, regardless of its merits. A disciplined approach to contract correspondence is therefore not merely good practice — it is a legal necessity.

Under AS 4000-1997, all notices required by the contract must be in writing and served in accordance with clause 13 (service of notices). Service is effective when delivered by hand, sent by pre-paid post to the party's address stated in the contract, or transmitted by facsimile or other electronic means where the contract permits. The critical question is when service is deemed to occur: delivery by hand is effective on delivery; postal service is effective 2 business days after posting (for domestic mail); electronic transmission is effective on confirmation of transmission. These deemed service provisions determine when time starts running for response deadlines and time bars, so precision in service method and timing is essential.

The most significant contractual time bars in AS 4000 include: the Contractor's obligation to give notice of a claim within 28 days of becoming aware of the circumstances giving rise to the claim (clause 29.2); the Contractor's obligation to notify qualifying causes of delay within 28 days of the cause first arising (clause 34.2); and the Superintendent's obligation to assess and grant extensions of time within 28 days of receiving the Contractor's claim. Under AS 2124-1992, similar time bar provisions exist at clause 44 (claims) and clause 35 (extensions of time), although the specific timeframes may differ depending on the project-specific amendments.

Managing the paper trail requires a systematic approach. Every project should maintain a correspondence register that logs all incoming and outgoing contractual correspondence, identifies the relevant contract clause, notes the response deadline, and tracks whether the response was given. Modern contract administration software (Aconex, Procore, InEight) provides tools for this, but the discipline of logging and tracking correspondence must be established from day one. The practice of "confirming in writing" any oral discussions or directions is not pedantic — it is protective. An instruction that is not confirmed in writing has no contractual standing under most standard form contracts, and attempting to rely on oral agreements in a dispute is fraught with evidentiary difficulties.

Reference: AS 4000-1997 clauses 13 (notices), 29 (claims), 34 (time); AS 2124-1992 clauses 13, 35, 44; *Gaymark Investments v Walter Construction Group* (1999) 16 BCL 449 (on notice requirements)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Failing to give notice within the contractual time bar, forfeiting valid claims — this is the single most common and costly administrative error in Australian construction
- Not keeping copies of all correspondence and relying on the other party's records, which may be incomplete or selectively preserved
- Using informal channels (email, text messages, verbal discussions) for contractual communications without following up with formal written notices
- Not tracking response deadlines for incoming notices, allowing time-critical obligations (such as issuing a payment schedule within 10 business days) to lapse
- Assuming that a blanket "reservation of rights" letter is sufficient to preserve specific contractual entitlements — it is not; each claim requires its own specific notice

## Site Instructions and Directions

The Superintendent's power to issue directions and instructions is one of the core mechanisms of contract administration under Australian standard form contracts. However, the distinction between a "direction" and an "instruction" — and the circumstances under which each can be issued — has significant implications for the Contractor's obligations and entitlements. A direction that changes the scope, character, or quality of the works constitutes a variation and triggers the Contractor's entitlement to additional time and cost. Understanding this distinction is essential to avoiding inadvertent variations and managing the project's cost and time exposure.

Under AS 4000-1997, the Superintendent has a general power to give directions "as to any matter arising under or in connection with the Contract" (clause 23). The Contractor is obligated to comply with all directions given by the Superintendent that are authorised by the contract. However, if the Contractor considers that a direction constitutes a variation — that is, it requires the Contractor to do something that is not already required by the contract documents — the Contractor must notify the Superintendent within 7 days of receiving the direction (clause 36.1). If the Superintendent does not confirm in writing that the direction is not a variation, the Contractor may treat it as a variation and claim accordingly. This 7-day notification mechanism is a critical safeguard that many contractors fail to exercise, often because the site team is focused on getting the work done rather than managing the contractual implications.

Under AS 2124-1992, the distinction is handled slightly differently. Clause 36 provides that the Superintendent may issue oral directions in circumstances of urgency but must confirm any oral direction in writing within 7 days. If the Superintendent fails to confirm the direction in writing, the Contractor may confirm it by written notice to the Superintendent, and if the Superintendent does not object within 7 days, the Contractor's confirmation is deemed to be a direction of the Superintendent. This provision protects contractors from being bound by unconfirmed oral directions while ensuring that urgent site instructions can be given without delay.

The Contractor also has a right to object to a direction under certain circumstances. Under AS 4000 clause 23.2, if the Contractor considers that a direction by the Superintendent is not authorised by the contract, the Contractor must give written notice to the Superintendent within 7 days setting out the grounds of objection. The Contractor must still comply with the direction pending resolution of the objection, but the notice preserves the Contractor's right to claim. Common scenarios where the right to object arises include: directions to accelerate the works without a formal acceleration agreement, directions to perform work that is clearly outside the contract scope, and directions that conflict with statutory requirements.

Reference: AS 4000-1997 clauses 23 (directions), 36.1 (variation by direction); AS 2124-1992 clauses 23, 36 (directions and variations)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Issuing verbal site instructions without written confirmation, leaving no contractual record and creating disputes about what was directed and when
- The Contractor failing to notify within 7 days that a direction constitutes a variation, losing the right to claim the work as a variation
- The Superintendent issuing directions that exceed the scope of their authority under the contract, potentially binding the Principal to unintended obligations
- Not maintaining a site instruction register that links each instruction to either the existing contract scope or a variation, making it impossible to track the cost impact of ad hoc directions
- Confusing the Superintendent's power to direct the manner of execution (which is limited) with the Contractor's right to choose the means and methods of construction (which is generally preserved under the contract)

## Progress Claims and Payment Certification

The progress claim and payment certification process is the contractual mechanism by which the Contractor is paid for work completed during the construction phase. In Australia, this process is governed by both the contract conditions and the applicable Security of Payment (SOP) legislation, which operates as a statutory overlay providing minimum payment protections regardless of what the contract says. Proper administration of progress claims is critical: errors in assessment can lead to overpayment (exposing the Principal to financial risk if the Contractor becomes insolvent) or underpayment (triggering SOP adjudication applications and damaging the commercial relationship).

Under AS 4000-1997 clause 37, the Contractor is entitled to submit a progress claim at intervals specified in the Annexure (typically monthly, on a nominated date such as the 25th of each month). The progress claim must show the value of work executed to date, the value of unfixed materials on site (where the contract permits), approved variations, and any other amounts the Contractor claims to be due. The Superintendent must then issue a progress certificate within 14 days of receiving the claim (or such other period as specified in the Annexure), stating the amount that the Superintendent considers to be due. The Principal must pay the certified amount within the period specified in the Annexure (typically 14-28 days after the progress certificate is issued). The total cycle from claim submission to payment is therefore typically 28-42 days.

The Superintendent's assessment methodology involves verifying the claimed percentage of work complete for each trade or element against actual site progress (usually confirmed by site inspection), checking that variation claims are supported by approved variation directions and agreed valuations, verifying that retention has been correctly calculated and withheld, and ensuring that the cumulative claim does not exceed the adjusted contract sum. Retention is typically 5% of each certified amount, capped at 5% of the contract sum (AS 4000 Annexure Part A, Item 15). For a $10 million contract, maximum retention would be $500,000. Some contracts reduce the retention rate to 2.5% or provide for security alternatives (bank guarantees) in lieu of cash retention.

Unfixed materials (materials delivered to site but not yet incorporated into the works) present a particular risk in progress claim assessment. Under AS 4000, the Contractor is only entitled to payment for unfixed materials where the contract expressly permits it and the materials are adequately stored and protected. The Superintendent should require evidence that: the materials are on site, they are intended for the works, they are properly stored and protected, they have been paid for by the Contractor (or the Contractor has title), and insurance coverage extends to them. Overpaying for unfixed materials creates significant exposure if the Contractor becomes insolvent — the materials may be subject to retention of title claims by suppliers and may not be available to a replacement contractor.

Reference: AS 4000-1997 clause 37 (payment); AS 2124-1992 clause 42 (certificates and payment); Building and Construction Industry Security of Payment Act 1999 (NSW) sections 8-15
Applies to: refurb, extend, new, remediation, advisory
Related standards: Security of Payment legislation in each state and territory

Common pitfalls:
- The Superintendent failing to issue a progress certificate within the contractual timeframe, which under SOP legislation may result in the Contractor's claimed amount becoming payable in full
- Not separately tracking and assessing variation amounts in progress claims, leading to cumulative overstatement of work in progress
- Paying for unfixed materials without verifying title, storage, and insurance, creating insolvency exposure
- Failing to reconcile cumulative progress claim assessments against the adjusted contract sum, potentially resulting in overpayment in later claims
- Not withholding retention as required by the contract, or releasing retention prematurely without obtaining the Contractor's defects obligations in return

## Security of Payment Legislation

The Building and Construction Industry Security of Payment Act (referred to as "SOP" or "SOPA" legislation) is the most significant statutory intervention in Australian construction payment law. Enacted in all states and territories, it provides a rapid, low-cost adjudication process for payment disputes and imposes strict obligations on parties who receive payment claims. The legislation was introduced to address the chronic problem of delayed and non-payment in the construction industry, which disproportionately affected subcontractors and smaller contractors. Failure to understand and comply with SOP obligations is one of the most common and costly errors in Australian contract administration.

The SOP legislation varies by jurisdiction but shares common core elements. In NSW (Building and Construction Industry Security of Payment Act 1999, as amended 2024), a person who has performed construction work or supplied related goods and services is entitled to a "progress payment" on and from each "reference date" under the contract. The reference date is typically the date specified in the contract for progress claims, or where no date is specified, the last day of each month. The claimant serves a payment claim that must identify the construction work to which the payment relates, indicate the amount of the progress payment claimed, and state that it is made under the Act. The respondent must serve a payment schedule within 10 business days (NSW) or the time specified in the contract (whichever is earlier), indicating the amount they propose to pay and the reasons for any difference between the claimed and scheduled amounts.

The consequences of failing to serve a payment schedule are severe. In NSW, if the respondent fails to provide a payment schedule within the required timeframe, the respondent becomes liable to pay the claimed amount in full, and the claimant may recover this as a debt due in court proceedings or suspend work. In Victoria (Building and Construction Industry Security of Payment Act 2002), the consequences are similar but the timeframes differ — the respondent has 10 business days to serve a payment schedule. In Queensland (Building Industry Fairness (Security of Payment) Act 2017), the legislation introduced additional requirements including project bank accounts for government projects and mandatory retention trust accounts.

The adjudication process under SOP provides a rapid determination of payment disputes. If the claimant is dissatisfied with the payment schedule (or the respondent fails to pay the scheduled amount), the claimant may apply for adjudication. An adjudicator must be appointed within 4 business days and must determine the application within 10 business days (or 15 business days for complex matters). The adjudication determination is binding on an interim basis — it must be complied with, but either party may subsequently have the dispute finally determined by a court or arbitration. Adjudication is not a final resolution but a "pay now, argue later" mechanism that ensures cash flow is maintained during construction. Adjudicator fees are typically $5,000-$25,000 depending on the complexity and value of the dispute.

The relationship between SOP and the head contract is nuanced. SOP does not override the contract but provides an alternative statutory route for payment that operates alongside the contractual payment mechanism. A party may pursue a SOP claim even while a contractual claim process is underway. However, certain contractual provisions — such as "pay when paid" clauses — are rendered void by SOP legislation in all jurisdictions. "Pay if paid" clauses (which make the obligation to pay conditional on actual receipt of payment from a third party) are also void under most SOP Acts.

Reference: Building and Construction Industry Security of Payment Act 1999 (NSW); Building and Construction Industry Security of Payment Act 2002 (VIC); Building Industry Fairness (Security of Payment) Act 2017 (QLD); Construction Contracts Act 2004 (WA); Building and Construction Industry Security of Payment Act 2009 (SA, TAS, ACT, NT); *Brodyn Pty Ltd v Davenport* (2004) NSWCA 394
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4000-1997 clause 37; AS 2124-1992 clause 42

Common pitfalls:
- Failing to serve a payment schedule within the statutory timeframe (10 business days in NSW), which results in the claimed amount becoming a debt due — this is perhaps the single most expensive administrative error in Australian construction
- Not including reasons for withholding payment in the payment schedule, which limits the respondent's ability to raise those reasons in adjudication
- Assuming that the head contract payment process displaces SOP rights — it does not; SOP operates as a parallel statutory entitlement
- Not training site administrators and project managers on SOP requirements, leaving compliance to chance
- Treating adjudication determinations as "optional" — failure to comply with an adjudication determination can result in judgment being entered as a court order and potential suspension of the right to do business in the construction industry

## Variation Administration

A variation in the context of Australian construction contracts is any change to the works directed by the Superintendent (or the Principal in contracts without a Superintendent) that alters the scope, character, quality, or quantity of the contracted works. Variation administration is consistently identified as one of the top three sources of cost overrun and dispute in the Australian construction industry. Effective variation management requires a systematic process for identification, direction, valuation, and approval — and discipline from all parties in adhering to that process, even under the time pressures of an active construction site.

Under AS 4000-1997, clause 36 defines a variation as a change to the work under the contract directed by the Superintendent. The Superintendent has the power to direct variations but this power is not unlimited — the variation must be within the general scope of the contract and the Superintendent cannot direct a variation that fundamentally changes the character of the works (a principle established in *Commissioner for Main Roads v Reed & Stuart* (1974)). The Superintendent must direct the variation in writing before the Contractor commences the varied work, except where the urgency of the situation makes prior written direction impractical. If the Superintendent directs a variation orally due to urgency, the Superintendent must confirm the direction in writing within 7 days.

Under AS 2124-1992, clause 40 provides a broadly similar framework but with some differences. The Superintendent's direction need not be in writing if confirmed subsequently, and the Contractor has an express obligation to carry out variations at the Superintendent's direction. Both contracts require the Contractor to give notice if the Contractor believes the scope of a direction exceeds what was contemplated by the contract — this notice is critical to the Contractor's claim entitlement. The direction-valuation-approval cycle should follow a defined sequence: (1) the Superintendent issues a written direction; (2) the Contractor provides a quotation or cost estimate; (3) the Superintendent assesses and approves the valuation; (4) the variation is incorporated into the next progress claim. In practice, this orderly sequence is often disrupted by urgency, which is why contemporaneous records (daily site diaries, time sheets, delivery dockets, photographs) are essential for retrospective valuation.

A critical distinction is between variations that change the scope of work (additions, omissions, or substitutions) and variations that change the conditions under which the work is performed (such as changed site conditions or revised sequencing). Both may give rise to entitlements under the contract, but they engage different valuation mechanisms and have different implications for the construction program. Omission variations (reducing the scope) also carry risks — the Superintendent cannot direct the omission of work for the purpose of having it carried out by another contractor (AS 4000 clause 36.4), and an excessive omission may constitute a breach of the implied term that the Principal will not substantially diminish the Contractor's opportunity to earn the contract sum.

Maintaining a variation register is essential. The register should record each variation with a unique reference number, the date of the Superintendent's direction, the Contractor's quotation, the Superintendent's assessed value, the current status (directed, quoted, assessed, approved, disputed), and the time impact. The register should be updated at least monthly and form part of the project's cost reporting framework. A well-maintained variation register provides an audit trail for final account settlement and is invaluable in dispute proceedings.

Reference: AS 4000-1997 clause 36 (variations and adjustments); AS 2124-1992 clause 40 (variations); *Commissioner for Main Roads v Reed & Stuart* (1974) 12 BLR 26
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4902-2000 clause 38 (D&C variations)

Common pitfalls:
- Allowing work to proceed without a formal written variation direction, making retrospective valuation difficult and disputed
- Not maintaining a contemporaneous variation register with running cost totals, allowing cumulative variation cost to become invisible until it is too late
- The Superintendent directing variations beyond the general scope of the contract, potentially exceeding their authority
- Confusing provisional sum adjustments with variations — provisional sum adjustments are contractual entitlements, not discretionary scope changes
- Failing to assess the time impact of variations at the same time as the cost impact, leading to program drift and eventual claims for disruption

## Variation Valuation Methods

The valuation of variations is one of the most technically demanding and frequently disputed aspects of contract administration. Australian standard form contracts establish a hierarchy of valuation methods that must be followed in sequence. Disputes over variation valuation account for a significant proportion of all construction claims in Australia, and the outcome often depends on the quality of the Contractor's contemporaneous records and the rigour of the Superintendent's assessment methodology.

Under AS 4000-1997 clause 36.2, the valuation hierarchy is: (1) where the variation is of a character and extent that rates or prices in the contract are directly applicable, those rates apply; (2) where contract rates are not directly applicable but provide a reasonable basis for valuation, those rates are used with appropriate adjustment; (3) where neither of the above applies, the Superintendent must determine a reasonable rate or price; and (4) as a last resort, valuation may be on a cost-plus basis (actual cost plus a margin for overheads and profit). Under AS 2124-1992 clause 40.5, a similar hierarchy applies, with the addition that the Superintendent may direct the Contractor to perform variation work on a "daywork" basis where the cost cannot be readily determined in advance.

Contract rates provide the most objective and least contentious basis for valuation. If a bill of quantities includes a rate for a particular item of work (e.g., reinforced concrete at $X/m3), and a variation requires additional work of the same character, the contract rate should be applied. However, contract rates are only appropriate where the character and conditions of the varied work are substantially the same as the work to which the rate applies. If the conditions differ materially — for example, concrete work in a confined space versus open slab work — the contract rate should be adjusted to reflect the different conditions. The onus is on the party seeking to depart from the contract rate to demonstrate that the conditions differ materially.

Cost-plus (or "daywork") valuation is the method of last resort and the most prone to dispute. Under a cost-plus valuation, the Contractor is reimbursed for the actual cost of labour, materials, plant, and subcontractors incurred in executing the varied work, plus an agreed margin for overhead and profit. The typical margin on cost-plus work is 10-15% for overhead and 5-10% for profit, giving a total margin of 15-25%. However, cost-plus valuation requires meticulous record-keeping: daily timesheets signed by the Superintendent's representative, material delivery dockets, plant hire records, and subcontractor time records. Without these contemporaneous records, cost-plus valuation becomes a matter of estimation and dispute. Quantum meruit (a reasonable value for work performed) is an equitable remedy that may apply where no contractual mechanism for valuation exists — for example, where work has been performed outside the scope of the contract entirely — but it is not a contractual valuation method and should not be confused with cost-plus valuation.

Reference: AS 4000-1997 clause 36.2 (valuation of variations); AS 2124-1992 clause 40.5; *Renard Constructions v Minister for Public Works* (1992) 26 NSWLR 234
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Defaulting to cost-plus valuation for every variation rather than applying the contractual valuation hierarchy, which inflates costs and reduces cost certainty
- Not maintaining contemporaneous records (signed timesheets, delivery dockets, plant records) for daywork, making it impossible to verify claimed costs
- Applying the margin for overhead and profit on top of subcontract costs that already include the subcontractor's own margin (double-dipping on margin)
- Failing to adjust contract rates where the conditions of the varied work differ materially from the original work, resulting in either under- or over-valuation
- Allowing the Contractor to submit lump sum variation quotations without a detailed breakdown, making assessment by the Superintendent impossible

## Constructive Variations and Scope Disputes

A constructive variation arises when the Contractor is required to perform work that is not expressly included in the contract documents, or when actions or omissions by the Principal or Superintendent effectively change the scope or conditions of the work, even though no formal variation direction has been issued. Constructive variations are one of the most contentious areas in Australian construction law because they sit at the boundary between what was contracted and what was actually required — a boundary that is often blurred by ambiguous documentation, evolving design, and informal site practices.

The most common scenarios giving rise to constructive variations include: design development by the architect after contract execution that requires additional work; the Superintendent's interpretation of the contract documents requiring a higher standard or greater scope than the Contractor reasonably included in its tender; changed site conditions that require different methods or materials; and interference by the Principal or third parties that disrupts the Contractor's planned sequence of work. In each case, the Contractor must demonstrate that the work or condition was not reasonably contemplated by the contract documents and that it has incurred additional cost or time as a result.

The Contractor's obligation to notify is critical. Under AS 4000-1997 clause 29.2, the Contractor must give written notice to the Superintendent within 28 days of becoming aware of any circumstance that may give rise to a claim for additional cost or time. Under clause 36.1, if the Contractor considers that a direction by the Superintendent constitutes a variation, the Contractor must give written notice within 7 days. If the Contractor proceeds with the work without giving the required notices, the Contractor may lose the right to claim, even if the work clearly constitutes a variation. This strict approach to notice requirements has been upheld by Australian courts and is the principal reason why site administration teams must be trained to identify and notify potential variations promptly.

The Principal's risk in not responding to variation claims should not be underestimated. Where the Contractor submits a claim for a constructive variation and the Superintendent fails to assess and respond to the claim within a reasonable time, the Principal may face exposure to adjudication under SOP legislation, an accumulation of unresolved claims that complicate the final account, and a breakdown in the commercial relationship that makes the project harder to manage. Best practice requires the Superintendent to assess every variation claim on its merits and respond in writing, even if the response is a rejection with reasons.

Reference: AS 4000-1997 clauses 29 (claims), 36 (variations); AS 2124-1992 clauses 40, 44; *Turner Corporation v Austotel* (1994) 13 BCL 378; *Pavey & Matthews v Paul* (1987) 162 CLR 221 (quantum meruit)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- The Contractor performing additional work without notifying that it considers the work to be a variation, then claiming retrospectively at the end of the project
- The Superintendent dismissing variation claims without proper assessment, accumulating a backlog of unresolved claims that become entrenched and costly to resolve
- Poorly drafted contract documents that create ambiguity about the scope of work, providing fertile ground for constructive variation claims
- Failing to distinguish between work that was clearly within the contract scope (and therefore not a variation) and work that was genuinely additional — a rigorous scope review at tender stage can prevent many of these disputes
- Not maintaining a record of all scope queries raised during construction and the Superintendent's responses, which are essential evidence if a constructive variation claim proceeds to dispute

## Extensions of Time (EOT)

Extensions of time (EOT) are the contractual mechanism by which the date for practical completion is extended when the Contractor has been delayed by events for which the contract provides relief. EOT administration is one of the most complex and high-stakes aspects of contract administration because the date for practical completion determines when liquidated damages begin to accrue. A failure to properly administer EOT claims can expose the Principal to the prevention principle (which may void the right to liquidated damages entirely), or alternatively, can leave the Contractor exposed to liquidated damages for delays that were not its fault.

Under AS 4000-1997 clause 34.2, the Contractor must give written notice to the Superintendent within 28 days after the commencement of any occurrence that the Contractor considers is a qualifying cause of delay. A "qualifying cause of delay" is defined in clause 34.1 and includes (among other things): variations directed by the Superintendent, latent conditions, delay by the Principal in providing access to the site, industrial action (subject to certain conditions), inclement weather beyond the seasonal norm, and any other cause specified in the Annexure. The notice must set out the cause of the delay, the estimated extent of the delay, and the measures the Contractor is taking to minimise the delay. The 28-day time bar is strictly applied — the Full Court of the Federal Court in *Gaymark Investments v Walter Construction Group* (1999) held that a failure to give timely notice did not disentitle the Contractor from an EOT under AS 2124 because the Superintendent had an independent duty to assess EOT, but this decision is specific to AS 2124 and should not be assumed to apply under AS 4000 where the notice requirement is more explicitly a condition precedent.

Under AS 2124-1992 clause 35, the Superintendent has a separate and independent duty to assess whether the Contractor is entitled to an EOT, regardless of whether the Contractor has given notice. This dual obligation — the Contractor to notify, the Superintendent to assess — is a feature of AS 2124 that does not exist in the same form under AS 4000. The Superintendent's duty to grant EOTs arises because the prevention principle requires that the Principal cannot benefit from liquidated damages for delays that the Principal (or its agents) caused. If the Superintendent fails to grant an EOT for a delay caused by the Principal, the date for practical completion may be set at large, meaning that liquidated damages cannot be levied and the Contractor is only required to complete within a reasonable time.

Concurrent delay — where two or more causes of delay operate at the same time, some of which are the Contractor's responsibility and some of which are qualifying causes of delay — presents one of the most difficult analytical challenges in contract administration. Australian courts have generally adopted the approach that where a qualifying cause of delay and a contractor risk event are truly concurrent (operating independently and simultaneously on the critical path), the Contractor is entitled to an EOT for the full period of the concurrent delay but is not entitled to delay damages (time-related costs). This approach, broadly consistent with the Society of Construction Law (UK) Delay and Disruption Protocol, means that neither party benefits or is penalised by the concurrency. However, the precise treatment of concurrent delay depends on the contract terms — some special conditions expressly address concurrent delay, and the analysis always depends on identifying the true critical path at the time of the delay.

Reference: AS 4000-1997 clause 34 (time and extension of time); AS 2124-1992 clause 35; *Gaymark Investments v Walter Construction Group* (1999) 16 BCL 449; *Peninsula Balmain v Abigroup* (2002) NSWCA 211; Society of Construction Law (UK) Delay and Disruption Protocol (2nd edition, 2017)
Applies to: refurb, extend, new, remediation, advisory
Related standards: AS 4902-2000 clause 34 (D&C time provisions)

Common pitfalls:
- The Contractor failing to give notice of delay within the 28-day time bar under AS 4000, potentially forfeiting its entitlement to an EOT
- The Superintendent failing to assess and grant EOTs, creating a risk that the date for practical completion is set at large under the prevention principle
- Not distinguishing between excusable delay (entitling the Contractor to time but not money) and compensable delay (entitling the Contractor to both time and money)
- Treating all weather delays as qualifying causes of delay without comparing actual weather data against the seasonal norm for the project location
- Failing to maintain an updated construction program that identifies the actual critical path, making it impossible to demonstrate the impact of any delay event on the completion date

## Delay Analysis Methods

Delay analysis is the technical process of determining the extent to which specific delay events have impacted the project completion date. In Australian construction disputes, the choice and application of delay analysis method can determine the outcome of multi-million dollar claims. There is no single "correct" method — the appropriate method depends on the available records, the complexity of the delay events, the stage of the project at which the analysis is performed, and the requirements of the contract. However, all methods share a common requirement: a reliable as-planned program and accurate as-built records.

The **as-planned vs as-built** method is the simplest approach and is often used as a first pass. It compares the planned start and finish dates from the baseline program against the actual dates achieved. Where the actual finish date exceeds the planned finish date, the difference is attributed to delay. The limitation of this method is that it does not identify the cause of the delay or demonstrate a causal link between specific delay events and the overall project delay — it merely quantifies the total delay. It is useful for identifying the existence and magnitude of delay but insufficient on its own to substantiate a contractual claim.

The **impacted as-planned** method models the effect of delay events on the baseline program by inserting delay activities (representing the delay events) into the as-planned critical path and observing the resulting impact on the completion date. This is a prospective method — it asks "what would have happened if these delays had not occurred?" Its advantage is simplicity and clarity; its limitation is that it relies on the as-planned program being a reasonable and accurate representation of what the Contractor actually intended to do, which is often challenged.

The **time impact analysis (TIA)** method is the most rigorous prospective method and is generally preferred for complex projects with multiple delay events. It analyses each delay event in sequence, inserting the delay into the program as it stood at the time the delay occurred (the "contemporaneous" or "updated" program) and measuring the impact on the critical path at that point. The TIA method accounts for the evolving nature of the critical path throughout the project and can distinguish between delays that affected the critical path and those that consumed float on non-critical activities. The Society of Construction Law (UK) Delay and Disruption Protocol recommends the TIA method as best practice.

The **windows analysis** method divides the project timeline into defined periods (windows) and analyses the delay experienced in each window against the progress and events that occurred in that window. This retrospective method uses the as-built records to determine what actually happened and then attributes the delay to specific causes. It is particularly useful where the as-planned program is unreliable or where there are many overlapping delay events. The **collapsed as-built** (or "but-for") method starts with the as-built program and removes delay events one at a time to determine what the completion date would have been "but for" each delay. This is a retrospective method that works well for demonstrating the effect of specific delay events but requires comprehensive as-built records.

Reference: Society of Construction Law (UK) Delay and Disruption Protocol (2nd edition, 2017); AACE International Recommended Practice 29R-03; *Thiess v MMAL* (2002) QCA 64
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Selecting a delay analysis method that does not match the available records — the most sophisticated method is useless if the underlying data is unreliable
- Relying solely on an as-planned vs as-built comparison without establishing causation, which is insufficient to substantiate a contractual claim for an EOT
- Not updating the program during construction, which undermines the ability to perform a credible time impact analysis because there are no contemporaneous snapshots of the critical path
- Failing to consider concurrent delays, which can change the allocation of responsibility from one party to another
- Using a delay analysis expert who applies a method that favours their client's position rather than the method most appropriate to the circumstances — this reduces credibility in adjudication or proceedings

## Liquidated Damages

Liquidated damages (LDs) are a pre-agreed sum payable by the Contractor to the Principal for each day (or week) that the Contractor fails to achieve practical completion by the date for practical completion (as extended by any granted EOTs). LDs serve a dual purpose: they provide the Principal with a certain remedy for late completion without the need to prove actual loss, and they provide the Contractor with a defined cap on liability for delay. The rate of LDs is specified in the contract Annexure and is typically expressed as a daily rate.

The enforceability of liquidated damages clauses in Australia was substantially clarified by the High Court decisions in *Andrews v ANZ Banking Group* (2012) 247 CLR 205 and *Paciocco v ANZ Banking Group* (2016) 258 CLR 525. The current position is that a liquidated damages clause will be enforceable if the stipulated amount represents a genuine pre-estimate of the loss likely to be suffered by the Principal as a result of late completion, or if it serves a legitimate interest in performance. A clause will be struck down as a penalty if it is "out of all proportion" to the legitimate interest being protected. In practice, this means that the LD rate should be supportable by reference to the Principal's actual anticipated losses — such as lost rental income, additional financing costs, additional consultant fees, cost of temporary accommodation, and reputational damage — rather than being set at an arbitrarily high figure designed to punish late completion.

Typical LD rates in Australian commercial construction range from $1,000-$5,000 per calendar day for small to medium projects, $5,000-$20,000 per calendar day for large commercial and institutional projects, and $20,000-$100,000+ per calendar day for major infrastructure and resources projects. The rate should be documented in a pre-contract assessment that records the assumptions and calculations used to derive it, so that the Principal can demonstrate it was a genuine pre-estimate if challenged.

The relationship between LDs and EOTs is fundamental. LDs can only accrue after the date for practical completion, including any extensions of time that have been granted. If the Superintendent has failed to grant an EOT for a delay caused by the Principal, the prevention principle may operate to set time at large, voiding the LD clause entirely and requiring the Principal to prove actual damages (general or unliquidated damages) through litigation. This is one of the reasons why timely and proper administration of EOT claims is so critical — not just for the Contractor's benefit, but to preserve the Principal's ability to rely on the LD regime. Some modern contracts include anti-prevention clauses designed to preserve the LD regime even where the Superintendent has failed to grant a timely EOT, but the enforceability of such clauses remains uncertain.

Reference: AS 4000-1997 clause 34.7 (liquidated damages); AS 2124-1992 clause 35.6; *Andrews v ANZ Banking Group* (2012) 247 CLR 205; *Paciocco v ANZ Banking Group* (2016) 258 CLR 525; *Multiplex Constructions v Honeywell* (2007) NSWCA 341
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Setting an LD rate that cannot be justified as a genuine pre-estimate of loss, risking the clause being struck down as a penalty
- Failing to administer EOT claims properly, creating the risk that time is set at large and the LD clause becomes unenforceable
- Not issuing a written notice to the Contractor when LDs commence, as required by the contract — failure to notify may waive the right to claim LDs
- Applying LDs beyond the contractual cap (where one exists) or continuing to deduct LDs after practical completion has been achieved
- Confusing liquidated damages (a fixed sum agreed in advance) with unliquidated damages (actual loss proven after the event), which are distinct legal remedies with different evidentiary requirements

## Programming and Contractor's Program

The construction program (or construction schedule) is one of the most important management tools on any construction project, yet its contractual status and the obligations it creates are frequently misunderstood. Under Australian standard form contracts, the program serves both as a management tool for planning and monitoring progress and as a contractual baseline against which delay and disruption claims are assessed. Understanding the distinction between a "for information" program and a contractually binding program is essential for both contract administrators and contractors.

Under AS 4000-1997 clause 33, the Contractor must submit a construction program to the Superintendent within the time specified in the Annexure (typically 14-28 days after the date of acceptance of tender). The Superintendent may direct the Contractor to update the program at any time. However, AS 4000 is notably silent on the contractual status of the program — it does not expressly state that the program is a contract document, that the Superintendent's review constitutes approval, or that the program becomes binding on the parties. The Superintendent's review of the program is generally regarded as acknowledgment that the program is a reasonable plan of work, not as a warranty that the program is correct or achievable. This distinction is important because it means the Superintendent is not estopped from later challenging the reasonableness of the Contractor's planned durations or sequences.

Under AS 2124-1992 clause 33, the Contractor's program has somewhat greater contractual significance. The Superintendent may direct the Contractor to revise the program, and the Contractor must update the program at regular intervals (typically monthly). The program must show the sequence, timing, and duration of all activities, including critical path activities, and should be in a critical path method (CPM) format. The Contractor's obligation to complete the works by the date for practical completion is absolute — the program shows how the Contractor intends to achieve that obligation, but the obligation itself derives from the contract, not the program.

Float ownership is a perennial issue in programming disputes. Float is the amount of time an activity can be delayed without delaying the project completion date. The question of who "owns" the float — the Contractor, the Principal, or the project as a whole — has significant implications for delay claims. In the absence of an express contract provision, Australian courts have generally treated float as belonging to the project rather than to any individual party. This means that if a qualifying cause of delay consumes float on a non-critical activity, it does not entitle the Contractor to an EOT (because the completion date is not affected). Conversely, if the Principal's delay consumes float and a subsequent Contractor delay then becomes critical, the Principal cannot argue that the Contractor "used up" the float that should have been available to absorb its own delay. Some modern contracts include express provisions addressing float ownership, and these should be reviewed carefully.

The baseline program (the program as at the contract date, before any changes or delays) is the benchmark against which delay claims are assessed. If the Contractor does not produce a credible baseline program, or if the program is not accepted by the Superintendent, the absence of an agreed baseline makes delay analysis extremely difficult and can undermine the Contractor's ability to substantiate EOT claims. Best practice requires the Contractor to produce a fully resourced, logic-linked CPM program within the first month of the contract, and for the Superintendent to review and accept (or require revision of) the program promptly.

Reference: AS 4000-1997 clause 33 (program); AS 2124-1992 clause 33; Society of Construction Law (UK) Delay and Disruption Protocol (2nd edition, 2017) — Core Principle 2 (Maintenance of Records)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- The Contractor failing to submit a compliant construction program within the contractual timeframe, undermining its ability to substantiate delay claims later
- The Superintendent not reviewing and responding to the Contractor's program, creating ambiguity about whether the program is accepted as a reasonable plan of work
- Not requiring the program to be in CPM format with logic links, making it impossible to identify the critical path or perform meaningful delay analysis
- Treating the Contractor's program as binding on the Principal or as a guarantee of the completion date, which it is not
- Failing to maintain updated programs at regular intervals, leaving no contemporaneous record of the critical path against which to measure delay impacts

## Practical Completion

Practical completion is arguably the most significant milestone in a construction contract. It triggers a cascade of contractual consequences: the transfer of risk and care of the works from the Contractor to the Principal, the commencement of the defects liability period, the release of the first moiety of retention (typically 50%), changes to insurance obligations, the cessation of liquidated damages liability, and the commencement of various limitation periods. Getting the determination of practical completion right — including addressing the difficult question of when minor defects are acceptable and when they prevent certification — is a core skill of contract administration.

Under AS 4000-1997, practical completion is defined in clause 1 as when the works are complete except for minor defects: (a) that do not prevent the works from being reasonably capable of being used for their intended purpose; and (b) that the Superintendent determines the Contractor has reasonable grounds for not promptly rectifying; and (c) rectification of which will not prejudice the convenient use of the works. This is a three-part test, and all three elements must be satisfied. The Superintendent must exercise professional judgment in applying this test — the question is not whether the works are perfect (they rarely are at this stage), but whether they are substantially complete and fit for use. Under AS 4000, the Contractor gives written notice to the Superintendent when the Contractor considers practical completion has been reached (clause 34.6), and the Superintendent must then assess the claim and either certify practical completion or advise what work needs to be completed before practical completion can be certified.

Under AS 2124-1992 clause 42.5, the concept is similar but uses the term "practical completion" without the detailed three-part definition of AS 4000. The Superintendent issues a Certificate of Practical Completion when the Superintendent is satisfied that the works have reached practical completion. The distinction between minor defects (which do not prevent practical completion) and incomplete work (which does) is critical and often contested. As a general guide, minor defects include cosmetic blemishes, minor paint touch-ups, missing hardware, and small finishing items that can be rectified without disrupting the Principal's occupation. Incomplete work includes building systems that are not operational, areas that cannot be safely occupied, and work that requires significant further construction activity to complete.

The consequences of achieving practical completion are far-reaching and must be carefully managed. Upon certification: (1) risk and care of the works passes to the Principal; (2) the defects liability period commences (typically 12 months); (3) 50% of accumulated retention is released to the Contractor (usually within 14 days); (4) the Contractor's obligations under the contract works insurance may reduce or cease (depending on the insurance terms); (5) liquidated damages cease to accrue; (6) the Principal gains access and may commence fit-out, commissioning, or occupation; (7) limitation periods for certain claims begin to run. The Superintendent should prepare a comprehensive defects list (sometimes called a "snag list") at the time of the practical completion inspection, recording all items requiring rectification during the defects liability period. This list should be detailed, specific, and accompanied by photographs where appropriate.

Reference: AS 4000-1997 clauses 1 (definitions), 34.5-34.6 (practical completion), 35 (defects liability); AS 2124-1992 clauses 42.5-42.6; *Alstom v Yokogawa* (2012) SASC 49 (on the meaning of practical completion)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Certifying practical completion prematurely (under pressure from the Principal to take occupation) when significant work remains incomplete, which transfers risk to the Principal and may release retention before defects are rectified
- Refusing to certify practical completion due to minor defects that clearly satisfy the "minor defects" test, which unfairly exposes the Contractor to ongoing liquidated damages
- Not preparing a comprehensive defects list at the time of practical completion inspection, making it difficult to enforce the Contractor's rectification obligations during the defects liability period
- Failing to issue the certificate of practical completion in writing as required by the contract, creating uncertainty about the date on which the defects liability period commenced
- Not notifying the insurer that practical completion has been achieved, potentially creating a gap in coverage during the transition from contract works insurance to the Principal's ongoing policy

## Defects Liability Period

The defects liability period (DLP) is the contractual period following practical completion during which the Contractor remains obligated to rectify any defects in the works that become apparent. The DLP provides the Principal with a defined window of protection, during which the Contractor must return to site and fix defects at its own cost. The Principal's remedy during the DLP is contractual — it arises from the express terms of the contract rather than from general law — and is secured by the retention monies (or bank guarantees) held by the Principal.

Under AS 4000-1997 clause 35, the DLP commences on the date of practical completion and runs for the period specified in the Annexure (typically 12 months, though periods of 24 months are used on some government and institutional projects). During the DLP, the Superintendent may direct the Contractor to rectify any defect that appears. The Contractor must rectify the defect within a reasonable time of receiving the direction. If the Contractor fails to rectify the defect within the time specified in the Superintendent's direction, the Principal may have the defect rectified by others and recover the cost from the Contractor — typically by deducting the cost from the retention held by the Principal or making a call on the Contractor's bank guarantee (clause 35.4). At the expiry of the DLP, the Superintendent issues a Final Certificate and the remaining retention (second moiety, typically 50% of accumulated retention) is released.

The distinction between **patent defects** and **latent defects** is important. Patent defects are those that are discoverable by reasonable inspection at the time of practical completion — they should be recorded on the defects list prepared at the practical completion inspection. Latent defects are those that are not discoverable by reasonable inspection at practical completion and only become apparent after occupation or over time — for example, waterproofing failures, structural cracking, or concealed installation defects. The DLP primarily addresses patent defects and those latent defects that manifest within the DLP period. However, the Principal's rights in respect of latent defects are not limited to the DLP — under general law, the Principal may have rights to claim for latent defects under the tort of negligence or breach of statutory duty for periods well beyond the DLP, subject to the applicable limitation period (typically 6 years for breach of contract and 6 years from discoverability for tortious claims, though this varies by jurisdiction).

The Contractor's right to re-enter the site and rectify defects during the DLP is an important counterpart to the obligation. If the Principal refuses to allow the Contractor access to rectify defects, and instead engages a third party, the Principal may lose the right to recover the (potentially higher) cost of the third party's rectification. The Contractor has a right to mitigate its liability by doing the rectification work itself. However, this right is not absolute — if the Contractor has demonstrated an inability or unwillingness to carry out the rectification, or if the nature of the defect requires specialist expertise that the Contractor cannot provide, the Principal may be justified in engaging others.

Reference: AS 4000-1997 clause 35 (defects liability); AS 2124-1992 clause 37; Limitation Act 1969 (NSW); Limitation of Actions Act 1958 (VIC); *Bellgrove v Eldridge* (1954) 90 CLR 613 (on measure of damages for defective work)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Not issuing written directions to the Contractor to rectify specific defects during the DLP, relying instead on verbal requests that have no contractual standing
- Allowing the DLP to expire without completing a final defects inspection, losing the contractual mechanism for requiring rectification
- Engaging a third party to rectify defects without first giving the Contractor a reasonable opportunity to do so, potentially forfeiting the right to recover costs
- Confusing the expiry of the DLP with the expiry of all rights to claim for defects — latent defects may be actionable well beyond the DLP under general law limitation periods
- Not maintaining a defects register during the DLP that tracks each defect, the date of notification to the Contractor, the Contractor's response, and the date of rectification

## Final Completion and Final Account

Final completion and final account settlement represent the definitive conclusion of the contractual relationship between the Principal and the Contractor. The final account is the reconciliation of all financial entitlements under the contract — the original contract sum, all approved variations, provisional sum adjustments, rise and fall, claims, retention, and any other adjustments. Reaching agreement on the final account and issuing the final certificate brings finality to the parties' obligations and releases the remaining security. Delays in settling the final account are endemic in the Australian construction industry and represent a significant source of commercial friction and wasted administrative effort.

Under AS 4000-1997 clause 37.4, within 28 days of the expiry of the last defects liability period, the Contractor must submit a final payment claim. This claim must include all amounts the Contractor claims to be due under the contract, including any outstanding variations, claims, and adjustments. The Superintendent must then assess the final claim and issue a final certificate within 42 days (or such other period as specified in the Annexure), stating the total amount payable or recoverable under the contract. The Principal must pay the certified amount (or the Contractor must repay any over-certification) within 14 days of the final certificate. Under AS 2124-1992 clause 42.8, the timeframes are similar but the Superintendent has broader discretion in the assessment process.

The effect of the final certificate is significant but not absolute. Under AS 4000-1997, the final certificate is evidence (but not conclusive evidence) of the matters stated in it. This means that either party may still challenge the final certificate in dispute proceedings, although the certificate carries significant evidential weight. Under some contracts (particularly in the UK tradition), the final certificate is expressed to be conclusive evidence of the matters stated — meaning that it cannot be challenged except in cases of fraud. Australian standard forms generally stop short of this, but the practical effect is similar: once the final certificate is issued and accepted without objection, it is very difficult to reopen the financial settlement.

Final account reconciliation is a detailed and often painstaking process. The Superintendent (or QS acting on the Principal's behalf) must verify: the value of all measured work against the contract rates and quantities; the valuation of all variations; all provisional sum adjustments; any rise and fall entitlements; any claims for additional costs (latent conditions, disruption, etc.); the retention balance and release; and any amounts to be deducted (liquidated damages, rectification costs, set-offs). Each of these items may be disputed, and the final account process often involves negotiation and compromise. Best practice is to maintain a running final account forecast throughout the construction phase, updating it monthly as part of the cost reporting cycle. This avoids the "final account shock" that occurs when all outstanding issues are confronted for the first time months after practical completion.

Reference: AS 4000-1997 clause 37.4 (final payment claim and certificate); AS 2124-1992 clause 42.8; Limitation Act 1969 (NSW) — limitation period for claims under the contract
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Allowing the final account to remain unresolved for years after practical completion, tying up retention and leaving the parties in a state of contractual limbo
- The Contractor failing to submit a final payment claim within the contractual timeframe, potentially losing entitlements that were not claimed
- Not maintaining a running final account forecast during construction, leading to surprises and protracted negotiations at the end of the project
- Failing to resolve outstanding variation disputes before attempting final account settlement, which can hold up the entire process
- Not recognising that limitation periods begin to run from the date the cause of action accrues (typically the date of breach or, for payment claims, the date the payment falls due), meaning that unresolved claims can become statute-barred if not pursued within 6 years

## Delay and Disruption Claims

Delay and disruption are related but legally and technically distinct concepts in Australian construction law. Delay refers to the extension of the project duration beyond the contractual date for practical completion — it is measured in time and gives rise to claims for extensions of time and, where compensable, delay costs (prolongation costs). Disruption refers to the loss of productivity caused by events that interfere with the Contractor's planned sequence, methods, or efficiency of working — it is measured in cost (additional resources expended to achieve the same output) and does not necessarily involve any delay to the completion date. A project can be disrupted without being delayed (if the Contractor absorbs the inefficiency by deploying additional resources), and delayed without being disrupted (if progress simply stops).

Proving disruption is considerably more difficult than proving delay. The preferred method is the **measured mile** approach, which compares the Contractor's productivity on an unimpacted portion of the works (the "measured mile") against productivity on the impacted portion. The difference in productivity, multiplied by the quantity of impacted work, yields the disruption cost. This method is considered the most reliable because it uses the Contractor's own project data and eliminates many external variables. Where a measured mile comparison is not possible (because the entire project was impacted), the Contractor may resort to the **total cost** method — which compares the Contractor's tendered cost against its actual cost and attributes the entire difference to the disrupting events. The total cost method is viewed with considerable scepticism by Australian courts and tribunals because it assumes that all cost overruns are attributable to the disrupting events, which is rarely true. A **modified total cost** method attempts to address this by adjusting for known contractor-caused overruns, but it remains a blunt instrument.

The Australian courts have dealt with the related concept of **global claims** — claims where the Contractor attributes a total loss to a group of events without particularising the loss caused by each individual event. In *John Holland Construction & Engineering v Kvaerner RJ Brown* (1996) 82 BLR 83, the Victorian Supreme Court held that global claims are permissible in principle but that the claimant must establish: (1) a causal link between the events relied on and the loss suffered; (2) that the events for which the respondent is responsible are the dominant cause of the loss; and (3) that no material contribution to the loss was made by events for which the claimant is responsible. In practice, global claims are risky and are more likely to succeed where the impacting events are numerous, closely interrelated, and impossible to disentangle.

The typical heads of claim in delay and disruption cases include: site overheads (prolongation of preliminaries), head office overheads (often calculated using the Emden or Hudson formula), additional labour and plant costs, loss of productivity (disruption), acceleration costs (if acceleration was directed or constructively required), and financing costs. Each head of claim requires separate substantiation, and the Contractor bears the burden of proof. The Superintendent's role in assessing delay and disruption claims is to evaluate the claim objectively, requesting further substantiation where necessary, and to certify the amount that is properly due — even if this means certifying an amount that the Principal may prefer not to pay.

Reference: AS 4000-1997 clause 29 (claims), 34 (time); AS 2124-1992 clauses 35, 44; *John Holland Construction & Engineering v Kvaerner RJ Brown* (1996) 82 BLR 83; *Thiess Contractors v Placer (Granny Smith)* (2000) 16 BCL 130; SCL Delay and Disruption Protocol (2nd edition, 2017)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Conflating delay and disruption in a single claim without separately quantifying the time impact (delay) and productivity impact (disruption)
- Relying on a total cost claim methodology without first attempting to use more reliable methods such as the measured mile
- Not maintaining contemporaneous productivity records (daily labour hours, plant hours, output quantities) that are essential for proving disruption
- Submitting a global claim without establishing the causal link required by *John Holland v Kvaerner*, leading to the entire claim being rejected
- Failing to claim for head office overheads as a separate head of delay damages, which can represent a significant entitlement on prolonged projects

## Contractual Claims Process

The contractual claims process under Australian standard form contracts follows a structured sequence of notification, substantiation, assessment, and determination. Each step has specific requirements — particularly regarding timing — that must be complied with to preserve the claim. The notice provisions in AS 4000-1997 and AS 2124-1992 have been the subject of extensive litigation, and the case law confirms that these provisions are generally enforced as conditions precedent to entitlement. A valid claim that fails on procedural grounds is a preventable tragedy in contract administration.

Under AS 4000-1997 clause 29, the Contractor must give written notice to the Superintendent within 28 days of becoming aware of anything that may give rise to a claim. The notice must identify the event or circumstance, the contractual basis for the claim, and the Contractor's estimate of the cost and time impact. The Contractor must then provide detailed particulars of the claim as soon as practicable and keep such contemporary records as are reasonable to substantiate the claim. The Superintendent must respond to the claim within 28 days, either agreeing, rejecting (with reasons), or requesting further information. The Superintendent's failure to respond within 28 days does not constitute deemed acceptance — the claim remains open for determination.

Under AS 2124-1992 clause 44, the notice period is similar (28 days) but the clause also requires the Contractor to "mitigate any loss and minimise any extra cost." This duty to mitigate is implied in all contracts but is expressed in AS 2124 and means that the Contractor cannot simply stand by and allow losses to accumulate — it must take reasonable steps to reduce the impact of the event giving rise to the claim. The failure to mitigate can reduce or extinguish the claim to the extent that the loss could have been avoided by reasonable mitigation measures.

The substantiation requirements are often where claims succeed or fail. A properly substantiated claim should include: (a) a detailed narrative of the events giving rise to the claim, with a timeline; (b) reference to the specific contract clause that provides the entitlement; (c) contemporaneous records supporting the factual assertions (site diaries, correspondence, photographs, survey records, delivery dockets); (d) a detailed cost breakdown with supporting documentation (timesheets, invoices, plant records); (e) a delay analysis (for time-related claims) demonstrating the impact on the critical path; and (f) a summary of the total amount claimed. Claims that are submitted as a single lump sum without detailed breakdown, or that rely on estimates rather than actual records, are significantly less likely to be assessed favourably by the Superintendent or upheld in dispute proceedings.

Reference: AS 4000-1997 clause 29 (Contractor's claims); AS 2124-1992 clause 44; *Jennings Construction v QH&M Birt* (1986) 8 NSWLR 18 (on time bar provisions); *Turner Corporation v Austotel* (1994) 13 BCL 378
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Missing the 28-day notice period, which under AS 4000 may operate as a strict time bar to the claim
- Submitting claims without sufficient substantiation, providing only a lump sum figure without a detailed breakdown or supporting documentation
- Not identifying the specific contractual basis for the claim, making it difficult for the Superintendent to assess the entitlement
- Failing to mitigate losses, which can reduce the recoverable amount even where the underlying entitlement is established
- The Superintendent failing to assess and respond to claims in a timely manner, allowing a backlog of unresolved claims to accumulate and complicate the final account

## Dispute Resolution Mechanisms

Construction disputes in Australia are resolved through a hierarchy of mechanisms that typically escalate from informal negotiation to formal proceedings. Most Australian standard form contracts include a multi-tiered dispute resolution clause that requires the parties to attempt resolution at each tier before escalating to the next. The cost, time, and relationship implications of dispute resolution escalation are significant, and selecting the right mechanism at the right time is a strategic decision that can determine the outcome.

Under AS 4000-1997 clause 42, the dispute resolution process begins with the Superintendent's role as first-instance decision maker. When a dispute arises between the parties, either party may give a notice of dispute to the Superintendent, who must then endeavour to resolve the dispute within 28 days by giving a written decision. If either party is dissatisfied with the Superintendent's decision (or the Superintendent fails to give a decision within 28 days), the dissatisfied party may refer the dispute to the next tier, which under AS 4000 is typically either mediation, expert determination, arbitration, or litigation as specified in the Annexure. AS 2124-1992 clause 47 provides a similar multi-tiered framework, with arbitration as the default final tier.

**Mediation** is the most commonly used intermediate dispute resolution mechanism in Australian construction. It involves a neutral third party (the mediator) who facilitates negotiation between the parties but does not impose a decision. Mediation is voluntary, confidential, and non-binding unless the parties reach agreement (which is then documented in a binding settlement deed). The success rate of construction mediation in Australia is generally reported at 70-80%. Mediation costs are typically $10,000-$30,000 per day (mediator's fees), with each party also bearing its own legal and expert costs. A one to two-day mediation of a mid-value construction dispute (say, $500,000-$2 million) might cost each party $30,000-$80,000 in total preparation and attendance costs.

**Arbitration** is a formal adversarial process in which an arbitrator (or panel of arbitrators) hears evidence and submissions from both parties and makes a binding decision (an award). Arbitration under Australian construction contracts is governed by the Commercial Arbitration Acts (uniform legislation in each state and territory) and is conducted in accordance with the rules specified in the contract (commonly the Resolution Institute Rules or the ICC Rules). Arbitration is private and confidential, which is attractive to parties concerned about reputational damage, but it can be as expensive and time-consuming as litigation. The arbitrator's award is enforceable as a court judgment but is subject to only limited grounds of appeal (essentially, jurisdictional errors and denial of natural justice). **Litigation** in the state or federal courts is the ultimate backstop and is appropriate where the dispute involves questions of law, where injunctive relief is required, or where the contract does not include an arbitration clause.

The costs of construction dispute resolution in Australia are substantial. As a general guide: adjudication under SOP (10-15 business days) costs $10,000-$40,000 per party; mediation (1-2 days) costs $30,000-$100,000 per party; arbitration (multiple hearing days) costs $100,000-$1 million+ per party; and litigation costs are similar to arbitration but with the additional risk of costs orders against the unsuccessful party (the general rule being that costs follow the event, meaning the loser pays the winner's party-party costs, which are typically 50-70% of actual costs). These figures underscore the importance of managing disputes proactively through proper contract administration, timely EOT and variation assessment, and genuine commercial negotiation before resorting to formal proceedings.

Reference: AS 4000-1997 clause 42 (dispute resolution); AS 2124-1992 clause 47; Commercial Arbitration Act 2010 (NSW) (and equivalent legislation in each jurisdiction); *Rinehart v Hancock Prospecting* (2019) 366 ALR 635 (on enforceability of multi-tiered dispute resolution clauses)
Applies to: refurb, extend, new, remediation, advisory
Related standards: Resolution Institute Mediation and Arbitration Rules; Building and Construction Industry Security of Payment Act (adjudication provisions)

Common pitfalls:
- Escalating a dispute directly to arbitration or litigation without genuinely engaging in the contractually required intermediate steps (mediation, expert determination), which may result in the proceedings being stayed
- Not serving a notice of dispute as required by the contract, which is a prerequisite to activating the dispute resolution clause
- Underestimating the cost and time required for formal dispute resolution, leading to commercial decisions that escalate disputes beyond the value at stake
- Failing to preserve contemporaneous records during construction that are essential evidence in any subsequent dispute proceedings
- Not engaging experienced construction lawyers early in the dispute process to ensure that procedural requirements are met and the claim is properly framed

## Insurance Requirements in Construction Contracts

Insurance is a fundamental risk transfer mechanism in construction contracts, and the insurance provisions of Australian standard form contracts impose specific obligations on both the Principal and the Contractor. The failure to maintain required insurances — or to ensure that the insurances are on the correct terms — can leave parties exposed to catastrophic uninsured losses. Contract administrators must understand the insurance regime not merely as a compliance checklist but as an integral part of the project's risk management framework.

Under AS 4000-1997 clause 16, the Principal is responsible for effecting and maintaining **contract works insurance** (also called construction all-risks or CAR insurance). This is a material damage policy that covers physical loss or damage to the works during the construction period and, typically, during the defects liability period. The policy is taken out in the joint names of the Principal, the Contractor, and all subcontractors, and includes a subrogation waiver so that the insurer cannot recover from any of the named parties after paying a claim. The typical policy excess (deductible) ranges from $10,000-$50,000 for standard projects and can be significantly higher for complex or high-risk projects. The premium is typically 0.3-0.8% of the contract sum, paid by the Principal. Some contracts (particularly those using AS 2124 or amended AS 4000) reverse this obligation and require the Contractor to effect the contract works insurance.

The Contractor is typically responsible for effecting and maintaining: **public liability insurance** (minimum $10-20 million per occurrence, covering third-party injury and property damage arising from the construction works); **workers' compensation insurance** (as required by state legislation, covering all employees for work-related injury and illness); and **motor vehicle insurance** (for all plant and vehicles used on the project). Where the contract is a design and construct contract, the Contractor is also required to maintain **professional indemnity (PI) insurance** to cover claims arising from errors in design. PI insurance is typically on a "claims made" basis (meaning the policy in force at the time the claim is made, not the time the error occurred, is the relevant policy) and must be maintained for a specified period after completion (typically 6-10 years). The cost and availability of PI insurance in the Australian construction sector has been significantly impacted by the building defects crisis (cladding, waterproofing), with premiums increasing by 50-200% in some sectors since 2019 and some classes of work becoming difficult to insure.

The **subrogation waiver** in the joint names contract works policy is particularly important. Without a subrogation waiver, the insurer who pays a claim for damage to the works could seek to recover the payment from the party who caused the damage (typically a subcontractor). A subrogation waiver prevents this, ensuring that the insurance operates as a true pooled risk mechanism. Contract administrators should verify that the contract works policy includes a subrogation waiver in favour of all named parties, and that all subcontractors are noted as insured parties.

Reference: AS 4000-1997 clauses 16-18 (insurance and indemnity); AS 2124-1992 clauses 16-18; Insurance Contracts Act 1984 (Cth); various state workers' compensation legislation
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Not verifying that all required insurances are in place before the Contractor commences work, creating an uninsured exposure from day one
- Failing to check that the contract works insurance is in joint names and includes a subrogation waiver, leaving subcontractors exposed to recovery actions by the insurer
- Not updating insurance sums insured when the contract sum increases due to variations, leaving a gap between the insured value and the actual value of the works
- Assuming that "all risks" insurance covers everything — all policies have exclusions (e.g., design defects, wear and tear, faulty workmanship in some policies) that must be understood
- Not requiring the Contractor to provide certificates of currency for all required insurances at regular intervals (typically annually), creating the risk that policies have lapsed

## Bank Guarantees and Retention

Security for performance is a critical element of Australian construction contracts, providing the Principal with a financial remedy if the Contractor fails to perform its obligations. The two primary forms of security are cash retention (withheld from progress payments) and unconditional bank guarantees (or insurance bonds). The administration of security — including the circumstances under which it can be called upon and the timing of its release — is governed by the contract terms and, in some jurisdictions, by statute. Mishandling security is a common source of disputes and, in extreme cases, can result in allegations of unconscionable conduct.

An **unconditional bank guarantee** (UBG) is an undertaking by the Contractor's bank to pay the Principal a specified amount on demand, without the need for the Principal to prove default or give reasons. This "pay first, argue later" characteristic makes UBGs a powerful security instrument but also one that is vulnerable to abuse. Under AS 4000-1997, the Contractor must provide security in the form and amount specified in the Annexure — typically 5% of the contract sum, which may be provided as either a UBG or cash retention (or a combination). The Annexure specifies the recourse events that entitle the Principal to have recourse to (call upon) the security: typically, the Contractor's failure to remedy a breach after notice, the Contractor's insolvency, or the Contractor's abandonment of the works. The Principal must not call upon the security except in the circumstances specified in the contract — a wrongful call can give rise to an injunction and a claim for damages.

**Cash retention** is the portion of each progress payment that is withheld by the Principal as security. The standard retention regime under AS 4000 is 5% of each progress claim, capped at a maximum of 5% of the contract sum. The first moiety (50%) is released within 14 days of practical completion, and the second moiety is released at the expiry of the defects liability period (subject to any set-offs for outstanding defect rectification costs). Retention trust accounts are now required in Queensland under the Building Industry Fairness (Security of Payment) Act 2017 (BIFSA), which requires the head contractor to hold retention money in a separate trust account for the benefit of subcontractors. Similar requirements are under consideration in other jurisdictions. Failure to comply with trust account requirements in Queensland can result in penalties and personal liability for directors.

**Insurance bonds** (also called surety bonds) are an alternative to bank guarantees and are offered by insurance companies rather than banks. They function similarly to bank guarantees but may have different terms and conditions. Some Principals prefer bank guarantees because they are more readily convertible to cash and carry the backing of an authorised deposit-taking institution. The choice between bank guarantee and insurance bond should be specified in the contract.

The timing and conditions for release of security are frequently contested. Under AS 4000, the contract specifies the progressive release: half at practical completion, the remainder at final certificate. However, the Principal may be entitled to retain security beyond these dates if there are outstanding claims or defects. The Contractor is entitled to substitute one form of security for another (e.g., replacing a bank guarantee with cash retention, or vice versa) where the contract permits. The cost of maintaining bank guarantees — typically 1-3% per annum of the face value — is a real cost to the Contractor that is factored into the tender price.

Reference: AS 4000-1997 clause 5 (security); AS 2124-1992 clause 5; Building Industry Fairness (Security of Payment) Act 2017 (QLD) Part 4 (retention trust accounts); *Clough Engineering v Oil & Natural Gas Corp* (2008) 249 ALR 458 (on unconditional bank guarantees)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Calling on a bank guarantee without contractual entitlement, which can result in injunctive relief and damages for wrongful call
- Not releasing retention or returning bank guarantees at the times specified in the contract, which may constitute a breach and trigger SOP claims
- Failing to comply with retention trust account requirements in Queensland (BIFSA), exposing the head contractor and its directors to statutory penalties
- Not tracking the expiry dates of bank guarantees and renewing them before they lapse, creating a period of unsecured exposure
- Accepting insurance bonds without verifying the bond terms match the contract requirements, potentially creating a less effective form of security than intended

## Work Health and Safety Obligations

Work health and safety (WHS) is regulated by harmonised legislation in most Australian jurisdictions — the Work Health and Safety Act 2011 (adopted by NSW, QLD, SA, TAS, ACT, and NT) and non-harmonised but broadly comparable legislation in Victoria (Occupational Health and Safety Act 2004) and Western Australia (Work Health and Safety Act 2020, which adopted the harmonised framework from March 2022). WHS obligations interact with construction contracts in important ways: the contract allocates WHS responsibilities between the Principal and Contractor, but statutory WHS duties cannot be contracted out of — a contractual provision that purports to transfer a non-delegable statutory duty is void to the extent of the inconsistency.

The key WHS duty holder on a construction site is the **Principal Contractor** (also called the "principal contractor for construction work" under the WHS Regulations). The Principal Contractor is the person or entity with management or control of the construction site and is responsible for: preparing a WHS management plan for the site; displaying site safety signage; ensuring all persons at the site are inducted; managing the safe work method statement (SWMS) process for high-risk construction work; managing workplace incidents; and coordinating the WHS activities of all subcontractors on site. Under most construction contracts, the head contractor assumes the role of Principal Contractor, and the contract should expressly confirm this appointment.

The concept of **PCBU** (person conducting a business or undertaking) is broader than the Principal Contractor. Every entity in the construction supply chain — the client (as Principal), the head contractor, each subcontractor, each design consultant — is a PCBU and has primary duties under the WHS Act, including the duty to ensure, so far as is reasonably practicable, the health and safety of workers and others who may be affected by the work. This means that the Principal cannot simply "hand off" all WHS responsibility to the head contractor — the Principal retains its own PCBU duties, particularly in relation to matters over which the Principal exercises influence or control (e.g., the design of the works, the project program, the budget for safety-related works). Penalties for WHS breaches are severe: up to $3 million for a body corporate and $600,000 or 5 years' imprisonment for an individual officer for a Category 1 offence (reckless conduct causing serious risk of death or serious injury).

**Safe Work Method Statements (SWMS)** are required for all high-risk construction work as defined in Schedule 3 of the WHS Regulations. High-risk construction work includes work at heights above 2 metres, work in excavations deeper than 1.5 metres, work involving demolition, work near energised electrical services, and work involving hazardous chemicals (among others). The SWMS must identify the high-risk construction work, state the hazards and risks, describe the control measures, and be kept on site and available for inspection. The Principal Contractor must ensure that SWMS are in place before high-risk work commences and must review and, if necessary, revise SWMS during the course of the work. Contract administrators should verify compliance with SWMS requirements as part of their regular site inspection regime.

Reference: Work Health and Safety Act 2011 (Cth/harmonised); Work Health and Safety Regulation 2017 (NSW); Occupational Health and Safety Act 2004 (VIC); Work Health and Safety Act 2020 (WA); AS 4000-1997 clause 11 (legislative requirements); AS 2124-1992 clause 11
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Assuming that appointing a head contractor as Principal Contractor transfers all WHS duties from the Principal — it does not; the Principal retains its own PCBU duties
- Not expressly appointing the Principal Contractor in the contract, creating ambiguity about who has WHS management obligations on site
- Failing to verify that SWMS are in place before high-risk construction work commences, which is a regulatory offence that can result in prohibition notices and penalties
- Not including WHS performance requirements in subcontract terms, leaving the head contractor unable to enforce site safety standards through the contractual chain
- Treating WHS compliance as a "Contractor problem" rather than a shared responsibility — in a serious incident, regulators will investigate every PCBU in the chain, including the Principal and its consultants

## Termination and Suspension

Termination of a construction contract is the ultimate contractual remedy and one that should only be exercised with extreme caution, clear legal advice, and strict compliance with the contractual procedures. A wrongful termination — one that does not comply with the contract terms or is not justified by the circumstances — can constitute a repudiation of the contract by the terminating party, reversing the parties' positions and entitling the innocent party to damages that may substantially exceed the cost of the original dispute. Suspension of the works is a less drastic measure but still carries significant commercial consequences.

Under AS 4000-1997 clause 39, the Principal may terminate the Contractor's employment (not the contract itself — this distinction is important) for substantial breach. The process requires: (1) the Superintendent to give written notice to the Contractor specifying the breach; (2) the Contractor to be given a reasonable opportunity to rectify the breach (typically 14 days); and (3) if the Contractor fails to rectify, the Principal may give written notice terminating the Contractor's employment. Grounds for termination include: the Contractor's failure to proceed with the works with due diligence, the Contractor's material breach of the contract, the Contractor's insolvency, and the Contractor's abandonment of the works. Some contracts also include a right of termination for convenience (without cause), which allows the Principal to terminate at any time by giving notice — but the Principal must then pay the Contractor for all work done, materials ordered, subcontract commitments, demobilisation costs, and loss of profit on the uncompleted work.

The Contractor's right to terminate is more limited under Australian standard form contracts. Under AS 4000 clause 39.4, the Contractor may terminate if the Principal fails to pay a certified amount within the contractual payment period and fails to rectify the non-payment within a further 14 days after receiving written notice from the Contractor. The Contractor may also suspend work under clause 39.5 if the Principal fails to make payment as required. Suspension must be preceded by written notice and takes effect 14 days after the notice. During suspension, the Contractor is entitled to its costs of suspension (including site standing charges, security, insurance, and financing costs) as a debt due by the Principal.

The consequences of termination are far-reaching. Following termination by the Principal for default: the Principal may take over the site, the works, and all plant and materials; the Principal may engage another contractor to complete the works; the Principal is entitled to withhold all further payment until the works are completed by the replacement contractor; and if the cost of completion exceeds the amount that would have been payable to the original Contractor, the difference is a debt due from the original Contractor to the Principal. Conversely, if the cost of completion is less, the original Contractor is entitled to the balance. Following wrongful termination: the innocent party may accept the repudiation and claim damages, including loss of profit on the uncompleted works, wasted tendering and mobilisation costs, and any other losses flowing from the breach.

Reference: AS 4000-1997 clause 39 (default and termination); AS 2124-1992 clause 44 (default); *Shevill v Builders Licensing Board* (1982) 149 CLR 620; *GEC Marconi Systems v BHP Information Technology* (2003) NSWSC 268 (on wrongful termination)
Applies to: refurb, extend, new, remediation, advisory

Common pitfalls:
- Terminating the Contractor's employment without strictly following the contractual notice and cure procedures, which converts a justified termination into a wrongful one
- Not obtaining legal advice before issuing a termination notice — the consequences of wrongful termination are severe, and the procedural requirements are technical
- Failing to distinguish between termination of the Contractor's employment (which preserves the contract for the purpose of completion by others) and termination of the contract itself (which ends all obligations)
- Not making adequate arrangements for the security, insurance, and maintenance of the partially completed works after termination, leaving the Principal exposed to damage, theft, or deterioration
- The Contractor suspending work without complying with the contractual notice requirements, potentially losing the right to recover suspension costs and exposing itself to a claim for abandonment

## Building and Construction Industry Regulation

The Australian building and construction industry is regulated by a complex web of state and territory legislation, building codes, and industry licensing requirements. Contract administrators must understand this regulatory framework because it affects the parties' obligations under the contract, the approvals required during construction, the mandatory inspection and certification regimes, and the statutory warranties and insurance requirements that apply on completion.

Each state and territory has a building authority responsible for regulating the construction industry within its jurisdiction. In **New South Wales**, the key regulators are NSW Fair Trading (which administers the Home Building Act 1989 and licences contractors and tradespeople) and the NSW Building Commissioner (who oversees compliance with the Design and Building Practitioners Act 2020 and the Residential Apartment Buildings (Compliance and Enforcement Powers) Act 2020). In **Victoria**, the Victorian Building Authority (VBA) administers the Building Act 1993 and registers building practitioners. In **Queensland**, the Queensland Building and Construction Commission (QBCC) administers the Queensland Building and Construction Commission Act 1991, which includes a comprehensive licensing and financial capacity regime. In **Western Australia**, the Building Commission administers the Building Act 2011 and Building Services (Registration) Act 2011.

Licensing requirements vary by jurisdiction but generally require that any person who carries out building work above a prescribed value must hold the appropriate licence class. In NSW, the Home Building Act 1989 requires a licence for residential building work valued at $5,000 or more. In Queensland, the QBCC licensing regime applies to all building work (not just residential) and requires contractors to meet minimum financial requirements (MFR) that are periodically audited. The consequences of performing unlicensed building work are severe: the builder may be unable to enforce the contract (including payment claims), may face criminal penalties, and may be ineligible for SOP adjudication.

The **National Construction Code (NCC)** (formerly the Building Code of Australia or BCA) sets the minimum performance requirements for buildings in Australia, including structural adequacy, fire safety, accessibility, energy efficiency, and plumbing. Compliance with the NCC is a mandatory requirement under all state and territory building legislation, and the contract should require the Contractor to comply with the version of the NCC in force at the date of the building approval. Changes to the NCC between contract date and completion may constitute a variation if the contract addresses regulatory changes — under AS 4000 clause 12, changes in legislative requirements that occur after the contract date and that require additional cost or time entitle the Contractor to a variation.

Mandatory **warranty periods** for residential building work are prescribed by state legislation and override any shorter periods in the contract. In NSW, the Home Building Act 1989 provides statutory warranties for 6 years for major defects and 2 years for other defects from the date of completion. In Victoria, the Domestic Building Contracts Act 1995 provides warranties for 10 years for structural defects and 6 years for other defects. In Queensland, the QBCC Act provides warranties for 6 years and 6 months from the date of practical completion. These statutory warranty periods exist independently of the contractual defects liability period and can significantly extend the builder's exposure.

Reference: Home Building Act 1989 (NSW); Design and Building Practitioners Act 2020 (NSW); Building Act 1993 (VIC); Queensland Building and Construction Commission Act 1991 (QLD); Building Act 2011 (WA); National Construction Code (NCC); AS 4000-1997 clause 12 (legislative requirements)
Applies to: refurb, extend, new, remediation, advisory
Related standards: National Construction Code (NCC) 2022; AS/NZS 2890 (parking); AS 1428 (accessibility)

Common pitfalls:
- Not verifying the Contractor's licence class and currency before contract execution, creating the risk that the contract is unenforceable for unlicensed work
- Failing to specify the applicable version of the NCC in the contract, creating ambiguity about which standard the works must comply with
- Not recognising that statutory warranty periods for residential work extend well beyond the contractual DLP, requiring the Contractor to maintain insurance and financial reserves for the statutory warranty period
- Assuming that compliance with the NCC is solely the Contractor's responsibility — the design consultants also have obligations under the Design and Building Practitioners Act 2020 (NSW) and equivalent legislation in other jurisdictions
- Not including the cost of mandatory independent inspections and certifications (building certifier, fire safety, accessibility) in the project budget

## Unfair Contract Terms in Construction

The extension of the Australian Consumer Law (ACL) unfair contract terms provisions to small business standard form contracts — and the further amendments in November 2023 making unfair terms void and subject to civil penalties — has significant implications for the construction industry. While the unfair contract terms regime was originally conceived for consumer contracts, its application to "small business" construction contracts (where at least one party has fewer than 100 employees or annual turnover below $10 million, and the contract value is below $5 million) captures a substantial portion of the Australian construction market, including many subcontracts and trade packages.

A term is "unfair" under the ACL if it would cause a significant imbalance in the parties' rights and obligations, it is not reasonably necessary to protect the legitimate interests of the party who would benefit from the term, and it would cause detriment to the other party if relied upon. In the construction context, clauses that are at risk of being found unfair include: broad unilateral variation powers that allow one party to change the scope, price, or terms without the other party's consent; excessive indemnity clauses that require the contractor or subcontractor to indemnify for losses beyond those it caused or could control; "pay when paid" or "pay if paid" clauses (which are already void under SOP legislation but may also be unfair under ACL); termination for convenience clauses that provide disproportionate compensation (or no compensation) to the terminated party; and clauses that impose unreasonably short time bars for claims, such as requiring notice within 24 hours of an event.

The **Australian Competition and Consumer Commission (ACCC)** has identified the building and construction industry as a priority sector for unfair contract terms enforcement. In 2023, the ACCC released specific guidance for the construction industry, highlighting common clauses in subcontracts and trade agreements that it considers potentially unfair. The guidance noted that "take it or leave it" standard form subcontracts — where the head contractor's terms are non-negotiable — are precisely the type of contract that the unfair terms regime is designed to address.

The interaction between unfair contract terms and proportionate liability legislation adds a further layer of complexity. Under the proportionate liability legislation enacted in each state and territory (e.g., Civil Liability Act 2002 (NSW) Part 4), a defendant in a claim for damages arising from a failure to take reasonable care is only liable for the proportion of the damage that reflects their responsibility. Clauses that attempt to contract out of proportionate liability — for example, requiring a subcontractor to indemnify the head contractor for 100% of a loss even where the subcontractor was only 30% responsible — may be void under the proportionate liability legislation and/or unfair under the ACL.

Reference: Competition and Consumer Act 2010 (Cth), Schedule 2 (Australian Consumer Law) sections 23-28; Treasury Laws Amendment (More Competition, Better Prices) Act 2022; Civil Liability Act 2002 (NSW) Part 4; ACCC Construction Industry Guidance (2023); *ACCC v JJ Richards & Sons* (2017) FCA 1224
Applies to: refurb, extend, new, remediation, advisory
Related standards: Building and Construction Industry Security of Payment Acts (void provisions)

Common pitfalls:
- Using heavily one-sided standard form subcontracts without considering whether the terms may be void and subject to penalties under the unfair contract terms regime
- Assuming that the unfair contract terms regime does not apply to construction contracts — it does, where the small business thresholds are met
- Including broad indemnity and hold harmless clauses that extend beyond the indemnifying party's proportionate liability, which may be void under both the ACL and proportionate liability legislation
- Not reviewing special conditions for compliance with the ACL before issuing contracts, creating the risk that key risk allocation clauses are later found to be unfair and void
- Failing to recognise that since November 2023, unfair contract terms are not merely voidable but attract civil penalties of up to $50 million for a body corporate, fundamentally changing the risk calculus for non-compliance
