---
tier: role-overlay
seed_type: role-overlay
loaded_by: "user_role: owner-builder"
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
summary: "Role overlay for the owner-builder acting as both principal and builder: permit and statutory posture, self-defined brief, direct trade engagement, claims and payment handling, self-direction discipline and escalation routing."
doctrine_anchors: [§seed-consultation-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §evidence-discipline, §voice-and-style, §owner-communication]
agents_anchors: [§1, §2, §3, §5, §6, §8, §9, §11]
---

# Role overlay - Owner-builder

> **Retired — not loaded by the knowledge catalog as of 2026-07-24.** Role is no longer a user-facing overlay; every project uses the single `role-architect-pm.md` project-lead overlay. This file is retained for reference only.

The **owner-builder** is the project lead acting as both principal and builder for their own residential work. There is no separate head builder carrying the builder-side compliance pack. The owner-builder coordinates the work, engages licensed trades directly, holds the permit where required, keeps the decision record, and carries the practical risk of their own scope, timing, quality, and compliance decisions.

This overlay is loaded when the project `README.md` declares `user_role: owner-builder`. It adds role-specific obligations to the SiteWise project-lead doctrine. It does not replace project evidence, the doctrine, or the declared archetype seed.

NSW is the deep default. Thresholds and statutory labels are current-as-authored guidance only; before relying on them for a real project, verify the current Building Commission NSW / Fair Trading, SIRA, council, certifier, and Planning Portal requirements.

## What the owner-builder is - and is not

The owner-builder is:

- **Owner and project lead** for their own home building work.
- **Coordinator of trades** rather than a licensed builder for the market.
- **Permit holder** where the NSW owner-builder permit trigger applies.
- **Decision-maker** for scope, budget, risk, programme, and self-directed variations.
- **Keeper of the project record** that a future buyer, certifier, insurer, tribunal, or trade may later inspect.

The owner-builder is not:

- A licensed builder by virtue of the permit. The permit is project-specific and does not authorise work for others.
- A holder of specialist trade licences. Electrical, plumbing, gasfitting, air-conditioning, refrigeration, waterproofing, and other licensed work still need the right licensed trades where the law requires it.
- Exempt from council, certifier, NCC, BASIX, WHS, tax, insurance, or statutory warranty obligations.
- Exempt from decision discipline because the decision-maker is themselves.

## Statutory instruments - NSW deep default

### Owner-builder permit

In NSW, an owner-builder permit is required where the owner wants to supervise or carry out owner-builder work on their own home, the reasonable market cost including labour and materials is over $10,000, and the owner is not contracting a licensed builder to supervise the work.

Owner-builder work includes supervision and coordination of construction, alterations, repairs, or additions to a single dwelling-house, secondary dwelling, or limited dual-occupancy cases, where the work requires DA consent or is complying development.

Setup evidence to capture:

- permit number, issue date, land / project reference, and permit holder;
- owner-builder course or approved competency evidence where the value trigger requires it;
- white card evidence;
- DA, CDC, CC, or other approval basis for the work;
- one-per-five-years eligibility check or special-circumstances basis where relevant;
- principal certifier appointment evidence and the date the permit was provided to the certifier.

The owner-builder permit is the role's licence-equivalent evidence for this project. It is not a licence to undertake other projects or specialist trade work.

### HOW / HBCF position

Home Owners Warranty / Home Building Compensation Fund cover is **not taken out by the owner-builder for the owner-builder's own work**. The owner-builder cannot insure themselves for their own work.

This does not mean there is no HBCF issue:

- A licensed contractor or trade business contracting directly with the owner-builder for residential building work over the HBCF threshold must provide its own HBCF certificate where the law requires it.
- The owner-builder should receive the contractor's HBCF certificate before that contractor starts work or receives payment.
- The setup checklist must **not** ask the owner-builder to lodge builder-side HBCF for their own work.
- The authority / insurance tracker may still include contractor HBCF rows where individual trade contracts trigger them.

The correct setup logic is:

| Scenario | Checklist treatment |
|---|---|
| Owner-builder doing / coordinating their own permitted work | Record owner-builder permit; mark owner-builder HBCF as not applicable |
| Licensed contractor engaged directly for a package over the HBCF threshold | Require contractor HBCF certificate before work or payment |
| Specialist trade below HBCF threshold but licensed work | Require licence and insurance checks; HBCF may be not applicable |

### Warranty and future-buyer disclosure

If the owner-builder sells the property within 7 years and 6 months after the owner-builder permit was issued, the contract for sale must include the required consumer warning. The next immediate owner receives statutory warranty rights against the owner-builder for the owner-builder work.

SiteWise tracks this as a live project issue:

- create a decision / risk note at setup recording the permit issue date and disclosure window;
- carry a warranty-to-future-buyers reminder into `09-handover-dlp/`;
- keep source evidence for concealed work, inspections, product warranties, trade invoices, and defects close-out;
- avoid describing owner-builder work as "no warranty" simply because HBCF is unavailable.

The short rule: **HBCF lodgement is omitted; warranty exposure is not omitted.**

### Approvals and certifier path

The owner-builder must hold the required planning and construction approvals before work starts. Typical NSW evidence:

- DA or CDC / complying development pathway;
- CC where DA path applies;
- principal certifier appointment;
- inspection schedule and critical-stage inspection requirements;
- BASIX certificate where the alterations/additions trigger applies;
- conditions of consent extracted to a tracker;
- Sydney Water / sewer / stormwater / driveway / tree / heritage approvals where triggered.

The owner-builder permit is needed before appointing the principal certifier and before commencing building work; development consent can be obtained before the owner-builder permit is issued.

### Insurance posture

Owner-builder insurance is practical risk management, not a single builder-pack equivalent. At setup, record:

- public liability insurance for third-party injury or damage arising from the work;
- home / construction-period cover confirming the existing home and works are insured during renovation;
- personal accident / income protection posture for the owner-builder where they are doing site work;
- contractor contract works insurance, public liability, and workers compensation evidence;
- workers compensation obligations where the facts mean a contractor is treated as a worker or the owner-builder employs labour;
- plant, tools, stored materials, theft, fire, storm, and water damage assumptions.

If the insurer excludes owner-builder work or renovation work to an occupied home, that is a high-risk escalation.

## Owner-builder setup workflow

When `contract-setup-system` runs for `user_role: owner-builder`, setup is not "execute the head contract". The equivalent is a **self-defined scope + permit + trade-control + decision-control** gate.

Minimum setup pack:

1. Complete README frontmatter: `archetype`, `user_role`, and `state` declared and not `TBC`.
2. Self-defined brief and scope filed under `00-brief-pmp/`.
3. Owner-builder permit evidence or a documented reason the permit trigger does not apply.
4. DA / CDC / CC and principal certifier pathway recorded.
5. BASIX position recorded, including whether the alteration/addition threshold applies.
6. Renovation due diligence filed: survey, dilapidation, existing services, structural, hazardous material, moisture / termite / rot, heritage / character.
7. Trade procurement plan opened: trade scopes, quote comparisons, licence checks, insurance checks, HBCF checks where triggered.
8. Insurance posture recorded.
9. Park-for-decision queue opened.
10. Ready-to-start checklist issued as a draft for human review.

## Self-defined brief

The owner-builder's brief is not a consultant PMP or builder tender. It is the written basis for the owner's own decisions:

- what is being renovated or added;
- what is excluded;
- which work the owner-builder intends to do personally;
- which work will be contracted to licensed trades;
- budget, contingency, and funding limits;
- approval pathway and current status;
- design status and unresolved selections;
- live-occupancy, staging, access, and safety assumptions;
- known risks and assumptions.

The self-defined brief is filed under `00-brief-pmp/` and referenced by the park-for-decision queue, cost plan, procurement comparisons, variation records, and handover notes.

## Subcontractor and trade management

The owner-builder engages trades directly. The discipline must be written down because there is no head builder absorbing the coordination risk.

For each trade, record:

- scope description with inclusions and exclusions;
- quote reference and accepted price;
- licence check evidence where licensing applies;
- public liability, workers compensation, contract works, and HBCF evidence where applicable;
- start and finish dates;
- dependencies on other trades;
- inspection / certificate deliverables;
- payment milestones and evidence required before payment;
- variations and verbal changes, if any.

Common owner-builder trade traps:

- assuming one trade includes set-out, penetrations, fire sealing, waterproofing, waste removal, protection, scaffold, or making good;
- paying before inspection evidence is filed;
- relying on verbal inclusions;
- starting work before owner-builder permit, certifier, or approvals are resolved;
- using unlicensed trades for specialist work.

## Claims, invoices, and payments

The owner-builder usually receives invoices from trades rather than issuing stage claims to an owner. Assessment still matters.

Before paying a trade invoice, check:

- the written trade scope and price basis;
- evidence that the invoiced work is complete;
- inspection or compliance certificate due at that point;
- licence / insurance / HBCF evidence required before payment;
- approved variations only;
- defects, incomplete work, or damage caused by that trade;
- GST and invoice identity details.

For packages with stage payments, make the trade stage definition explicit. Do not import the HIA builder-to-owner stage payment schedule unless the trade contract actually uses it.

## Variations and self-directions

An owner-builder variation is often a decision the owner makes to their own scope. It still needs a record.

Every change should state:

- source of direction: self, certifier, engineer, designer, trade, authority, or latent condition;
- scope change;
- cost effect;
- time effect;
- BASIX / NCC / approval effect;
- trade-interface effect;
- decision date and decision-maker;
- whether the change supersedes an earlier decision.

Use `variation-management-system` where the change affects contract value, time, approval, compliance, or trade sequencing. Use the park-for-decision queue where the owner-builder has not decided yet.

## Escalation routing - owner-builder

The doctrine says when to escalate. This overlay says where it goes.

For owner-builder projects, escalation defaults to the **park-for-decision queue** in `08-meetings-reporting/`. The agent must not suppress a flag because the decision-maker and user are the same person.

| Trigger | Destination | Register / draft | Rule |
|---|---|---|---|
| Owner decision required | Park-for-decision queue | `PFD-<seq>` row | Every row needs decision required, owner, due date, consequence if deferred, and next action |
| Technical decision required | Consultant / certifier RFI plus park-for-decision row | RFI row and PFD row | Technical answer does not itself decide cost / scope / time |
| Authority or compliance uncertainty | Authority / certifier correspondence plus risk row | Authority tracker, risk row, PFD row if owner choice exists | Do not proceed on an assumption where approval is at stake |
| Budget or contingency movement | Cost register plus park-for-decision row | Cost row and PFD row | The owner-builder decides whether to accept, redesign, defer, or stop |
| Critical-path delay | Programme update plus park-for-decision row | Risk / action / PFD row | Deferral is itself a decision with a programme consequence |
| Trade scope gap | Procurement / trade scope record plus park-for-decision row | Trade comparison and PFD row | Resolve before trade starts, not after invoice |

Status vocabulary for the queue is defined in `register-row-draft.md`: `parked`, `due`, `overdue`, `decided`, `superseded`.

Overdue park-for-decision rows are escalations. They appear in risk review summaries and owner-builder setup re-runs.

## Voice register - owner-builder defaults

Owner-builder projects use a lighter practical voice for internal decision notes, but formal records still stay formal where the audience or document type requires it.

- **Stakeholder register**: self-briefs, owner-builder decision notes, park-for-decision explanations, owner-facing summaries written for future self or family stakeholders.
- **Contractual register**: trade scopes, contractor correspondence, authority letters, certifier RFIs, variation records, payment disputes, formal notices.

The owner-builder can be both sender and recipient of a note. That does not make the note optional. It makes the note part of the future evidence trail.

## Renovation-specific overlay notes

When paired with `archetype: renovation`, this role overlay becomes especially risk-sensitive:

- Existing conditions are assumptions until opened up or evidenced.
- A trade quote that does not name latent-condition exclusions is not complete enough for confident award.
- Hidden services must be located before demolition or saw-cutting.
- Structural intervention needs engineer sign-off and hold points.
- BASIX-for-additions may apply even when the project feels like "just a renovation".
- Heritage / character controls may shape what can be demolished, altered, or seen from the street.
- Live occupancy creates safety, dust, noise, security, temporary services, and insurance exposure.

Load `renovation-guide.md` with this overlay for the owner-builder renovation cell.

## Setup checklist - owner-builder mobilisation

When the setup system is run for `user_role: owner-builder`, the ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Self-defined brief filed.
- [ ] Owner-builder permit filed, or documented no-permit basis if threshold / approvals do not trigger it.
- [ ] Course / white card / eligibility evidence filed where required.
- [ ] DA / CDC / CC pathway recorded; principal certifier appointment status recorded.
- [ ] BASIX alteration/addition trigger checked and certificate filed where required.
- [ ] Renovation due diligence filed: survey, dilapidation, services, structural, hazardous material, moisture / termite / rot, heritage / character.
- [ ] Public liability and construction-period home / works insurance position recorded.
- [ ] Trade register opened with licence, insurance, HBCF, scope, and payment checks.
- [ ] Park-for-decision queue opened under `08-meetings-reporting/`.
- [ ] Warranty / future-buyer disclosure reminder opened for handover / DLP.
- [ ] First risk review date set and overdue self-flags surfaced.

Builder-side HBCF lodgement is **not** on this checklist.

## Failure modes specific to owner-builder

- Treating the permit as a builder's licence.
- Starting before permit, certifier, DA / CDC / CC, or critical inspections are in place.
- Omitting contractor HBCF evidence where a direct contractor package triggers it.
- Assuming "no HBCF for owner-builder" means "no warranty exposure".
- Failing to disclose owner-builder work on sale within the disclosure period.
- Paying trades before inspection, certificate, or completion evidence.
- Letting self-decisions live only in memory or text messages.
- Treating latent conditions as trade problems rather than owner-builder risk decisions.
- Opening structure without engineer involvement or temporary works planning.
- Missing BASIX, waterproofing, or OC evidence until handover.

## Agent behaviour under this overlay

When `user_role: owner-builder` is declared:

1. The agent loads this overlay and the declared archetype seed on any phase-gate task.
2. The agent enforces the README declaration gate before drafting.
3. The agent omits builder-side HBCF lodgement from owner-builder setup and instead checks contractor HBCF evidence where triggered.
4. The agent opens or refreshes the park-for-decision queue whenever a decision is deferred.
5. The agent treats overdue queue rows as escalation items.
6. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
7. The agent flags state gaps rather than extending NSW owner-builder assumptions to another state.

## See also

- `../00-doctrine/doctrine.md` - project lead doctrine
- `../00-doctrine/doctrine.md` seed-consultation-discipline
- `../00-doctrine/doctrine.md` register-discipline
- `../00-doctrine/doctrine.md` decision-discipline
- `../00-doctrine/doctrine.md` evidence-discipline
- `../00-doctrine/doctrine.md` escalation-triggers
- `../00-doctrine/doctrine.md` voice-and-style
- `../00-doctrine/doctrine.md` owner-communication
- `../AGENTS.md` Sec. 1, Sec. 2, Sec. 3, Sec. 5, Sec. 6, Sec. 8, Sec. 9, Sec. 11
- `renovation-guide.md` - Tier 2 archetype seed for the slice-07 cell
- `setup-and-commission-guide.md` - setup workflow deepened by this slice
- `contract-administration-guide.md` - contract / variation / notice context
- `procurement-quoting-guide.md` - trade quoting and comparison
- `cost-management-principles.md` - budget, contingency, PC sum, variation, and owner-supplied item discipline
- `program-scheduling-guide.md` - cycle times, lookahead, delay, and lead-time risk
- `../02-skills/atomic/seed-targeted-read.md` - loads this overlay
- `../02-skills/atomic/register-row-draft.md` - includes the park-for-decision queue row type
- `../02-skills/systems/contract-setup-system.md` - owner-builder setup path
- `../02-skills/systems/risk-register-system.md` - overdue self-flag escalation
