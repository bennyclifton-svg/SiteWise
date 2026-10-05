---
tier: role-overlay
seed_type: role-overlay
loaded_by: "user_role: d-and-c"
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
summary: "Role overlay for the D&C contractor: inherits the builder's obligations and adds design responsibility, the design responsibility matrix, consultant procurement, PI insurance posture and certifier submission control."
doctrine_anchors: [§seed-consultation-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §evidence-discipline, §voice-and-style, §owner-communication]
agents_anchors: [§1, §2, §3, §5, §6, §8, §9, §11]
required_by: {head-contractor-procurement: 3}
---

# Role overlay - D&C contractor

> **Retired — not loaded by the knowledge catalog as of 2026-07-24.** Role is no longer a user-facing overlay; every project uses the single `role-architect-pm.md` project-lead overlay. This file is retained for reference only.

The **D&C contractor** is the head contractor and also carries responsibility for completing, coordinating, and submitting the design required to deliver the project. The role inherits the builder's construction obligations and adds design responsibility, consultant procurement, professional indemnity posture, design programme control, design responsibility matrix discipline, and certifier submission control.

This overlay is loaded when the project `README.md` declares `user_role: d-and-c`. It is additive to the SiteWise project-lead doctrine and to `role-builder.md`. It does not replace project evidence, the doctrine, the declared archetype seed, or the executed contract.

NSW is the deep default. HBCF / HOW, LSL, PI insurance, certifier, NCC, BASIX, and contract references are current-as-authored guidance only. Before relying on them in a real project, verify current project evidence and current official NSW Building Commission / Fair Trading, HBCF, Long Service Corporation, certifier, council, Planning Portal, and NCC requirements.

## Builder obligations inherited

A D&C contractor is still a builder. Unless this overlay adds to or overrides the posture, load and apply `role-builder.md` for:

- builder licence and Qualified Supervisor evidence;
- HBCF / HOW eligibility and per-project certificate where required;
- LSL receipt or payment pathway where the project value and approval path require it;
- contract works insurance, public liability, and workers compensation where applicable;
- head contract execution, contract summary, deposit, stage payment, variation, EOT, and claim posture;
- subcontractor procurement and subcontractor claim assessment;
- construction management, waste management, traffic management, inspection, and defect records.

Do not copy the builder checklist into a second D&C-only builder checklist. The D&C overlay references `role-builder.md` so there is one builder obligation base. If `role-builder.md` changes, the D&C role inherits the current builder posture unless this overlay says otherwise.

## What the D&C contractor is - and is not

The D&C contractor is:

- **Head contractor** to the owner / principal for construction delivery.
- **Design-responsible contractor** for the design obligations assigned by the executed D&C contract, Principal's Project Requirements, design brief, scope, and special conditions.
- **Consultant procurer and coordinator** for consultants appointed by, novated to, or otherwise controlled by the D&C contractor.
- **Keeper of the design responsibility matrix** and the design deliverables register.
- **Certifier submission coordinator** for design packages that need certifier review before construction, inspection, or OC.
- **Holder of the D&C insurance stack**, which includes builder insurance plus PI posture for design liability.

The D&C contractor is not:

- The certifier. Design responsibility does not create certification authority.
- The owner or principal. Owner decisions, PPR changes, cost approvals, and time approvals still need owner / principal sign-off where the contract requires it.
- A free design adviser outside the contract. Design advice, value engineering, and redesign are contractual services with scope, fee, programme, and liability consequences.
- A reason to ignore the original brief. The D&C contractor may complete design, but the accepted baseline is still the executed contract, PPR, scope, drawings, and special conditions.

## Contract posture

D&C contract families vary by project scale. The executed contract and special conditions always win.

Common forms:

- **AS 4902** - primary high-complexity D&C form. Common where a Superintendent / administrator is appointed and the design completion obligation is material.
- **Bespoke D&C residential contracts** - common in developer, townhouse, or small multi-dwelling work.
- **HIA / MBA residential forms with D&C-like special conditions** - possible on smaller residential projects. Treat the special conditions carefully; a standard residential form may not carry a complete design responsibility mechanism.
- **AS 4000** - construct-only. If the contract is AS 4000 but the project is being run as D&C, escalate the mismatch before relying on a design-risk transfer.

At setup, record:

- contract family and edition;
- PPR / design brief reference;
- design baseline documents and revision status;
- design scope exclusions and assumptions;
- whether consultants are owner-appointed, novated, D&C-appointed, or retained separately by the owner;
- design review / approval rights;
- certifier submission obligations;
- PI insurance and limitation / exclusion position;
- design-fee amount or pricing basis where separately scheduled.

## Design responsibility pack

The D&C design pack is filed primarily under `00-brief-pmp/`, `02-consultant/`, `03-design/`, and `04-planning-and-authorities/`. It is live throughout the project, not a setup-only appendix.

Minimum design pack:

- Principal's Project Requirements / design brief;
- accepted tender response and D&C qualifications;
- design responsibility matrix;
- consultant appointment / novation records;
- consultant scopes and deliverables;
- consultant PI evidence;
- D&C PI evidence;
- design programme;
- design deliverables register;
- design RFI register;
- design review and approval protocol;
- certifier submission schedule;
- current drawing and specification revision register;
- design change and value-engineering decision trail;
- BASIX / energy / NCC compliance evidence where design choices affect commitments.

Missing design-pack evidence is a setup gap. It must become an action, risk, decision, or escalation row. Do not leave design responsibility gaps as narrative commentary.

## Design responsibility matrix

The design responsibility matrix (DRM) states who is responsible for each design element, interface, deliverable, review, and certification support item.

Minimum DRM columns:

- element / system;
- scope boundary;
- responsible designer;
- reviewer;
- D&C responsible manager;
- consultant appointment source;
- deliverable reference;
- revision / status;
- certifier submission requirement;
- construction hold point;
- related cost / programme exposure;
- next action and due date.

High-risk D&C matrix rows:

- structural frame, slabs, retaining, temporary works, and propping;
- fire separation, party walls, penetrations, and fire stopping;
- waterproofing, balconies, roof tie-ins, wet areas, and drainage;
- civil, OSD, driveway / crossover, stormwater, and public-domain works;
- hydraulic services, sewer, water, gas, and separate metering;
- electrical, main switchboard, meters, solar, EV, and communications;
- BASIX / energy / NatHERS / Section J pathway depending classification;
- facade, cladding, windows, acoustic separation, and external envelope;
- certifier submissions, inspection evidence, and OC evidence.

The DRM must be updated when consultant scopes change, an RFI changes design intent, a variation affects design, the certifier requests revision, or the owner changes the PPR. A stale DRM is itself a risk.

## Consultant procurement and coordination

D&C consultant procurement is different from architect-PM coordination because the consultant may be working for the D&C contractor, not the owner.

At setup, identify for each consultant:

- appointing party: owner, D&C contractor, novated, direct appointment, or retained reviewer;
- appointment document and fee basis;
- scope inclusions and exclusions;
- deliverables and dates;
- PI insurance evidence;
- reliance limitations;
- certifier submission obligations;
- who owns design coordination;
- who responds to RFIs;
- who signs or supports compliance evidence.

Typical D&C consultant set:

- architect or building designer;
- structural engineer;
- civil / stormwater engineer;
- hydraulic consultant;
- electrical consultant;
- BASIX / energy / NatHERS assessor;
- certifier interface lead;
- fire engineer or access consultant where classification or approval pathway triggers it;
- acoustic, facade, geotechnical, landscape, traffic, or heritage consultant where the project evidence demands it.

Consultant advice that affects cost, time, scope, quality, compliance, or owner decision becomes a register row. It is not enough to leave it in email.

## PI insurance and insurance stack

The D&C insurance stack has two halves.

Builder-side policies from `role-builder.md`:

- contract works insurance;
- public liability;
- workers compensation where applicable;
- HBCF / HOW where required by the residential work (**In VIC:** the equivalent is Domestic Building Insurance (DBI) administered by the VMIA or approved insurers under the *Domestic Building Contracts Act 1995* (Vic); threshold is $16,000; builder must be registered as a Domestic Builder with the Victorian Building Authority (VBA));
- other project-specific statutory or contract security.

Design-side policies:

- D&C professional indemnity insurance for design liability assumed under the contract;
- consultant PI evidence for each consultant whose design is relied on;
- run-off period and retroactive date where relevant;
- exclusions for D&C, cladding, waterproofing, fire engineering, structural design, certification, or novated design;
- sublimits or aggregate limits that may be inadequate for the project scale;
- contract clauses that create uninsured design liability.

Contract works insurance and public liability are not PI insurance. If the project file has CWI and PL but no PI evidence, the design liability is not evidenced. Surface that gap before contract execution, design submission, or site start.

## Setup workflow - D&C contractor

When `contract-setup-system` runs for `user_role: d-and-c`, setup proves that both the builder pack and the design pack are ready enough for the declared archetype and state.

Minimum setup sequence:

1. Confirm the README declares `archetype`, `user_role`, and `state`.
2. Load the declared archetype seed and this role overlay through `seed-targeted-read`.
3. Run `evidence-sweep` for builder evidence, design evidence, consultant evidence, authority evidence, and insurance evidence.
4. Assemble the builder pack from `role-builder.md`: tender response, executed contract, licence, Qualified Supervisor, HBCF / HOW, LSL, CWI, PL, workers compensation, BASIX / CC / approval evidence, management-plan prompts.
5. Assemble the design pack: PPR, design fee, design programme, DRM, consultant appointment evidence, consultant PI, D&C PI, deliverables register, design RFI register, certifier submission protocol.
6. For `archetype: multi-dwelling`, confirm classification, party-wall strategy, separate metering strategy, authority / infrastructure contribution allowances, staging / OC path, and per-dwelling handover evidence.
7. Open or refresh registers.
8. Draft the ready-to-start checklist through `markdown-draft-for-review`.
9. Surface gaps and escalations.

## D&C setup checklist

The D&C ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Tender response / accepted D&C offer filed.
- [ ] Principal's Project Requirements / design brief filed.
- [ ] Executed head contract and special conditions filed.
- [ ] Builder licence and Qualified Supervisor evidence verified.
- [ ] HBCF / HOW eligibility and per-project certificate issued where required.
- [ ] LSL receipt or payment pathway verified where required before CC.
- [ ] Contract works insurance certificate filed.
- [ ] Public liability certificate filed.
- [ ] Workers compensation evidence filed where applicable.
- [ ] D&C PI insurance evidence filed, with exclusions checked.
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

For a multi-dwelling project, add:

- [ ] Class 1a attached dwelling vs Class 2 classification tested and evidenced.
- [ ] Party-wall / fire-separation strategy evidenced.
- [ ] Separate metering / services strategy evidenced.
- [ ] Infrastructure contribution / authority charge assumptions recorded.
- [ ] Staging, subdivision / strata, OC, and handover pathway recorded.

## Progress claims and cost posture

D&C progress claims follow the executed contract.

The D&C contractor may claim:

- builder stage payments or milestone payments for construction work;
- actual-cost substantiation where the contract is cost-plus;
- design fee instalments where the contract schedules design fees separately;
- consultant disbursements where the contract permits pass-through or reimbursement;
- signed variations only where the contract mechanism has been satisfied.

Claims depending on design completion need design evidence:

- deliverable status;
- revision issue date;
- certifier submission / response status where relevant;
- owner / principal review status where contractually required;
- consultant certificate or advice where relied on.

Unsigned design changes, unresolved PPR changes, or incomplete design submissions do not become payable just because the D&C contractor has incurred effort.

## Variations and design development

D&C design development is not automatically a variation. The baseline must be tested before any variation is drafted.

Variation tests:

- What was in the PPR, scope, accepted tender, drawings, specification, and executed contract?
- Is the change owner-directed, authority-directed, consultant-driven, latent, compliance-driven, or D&C design completion?
- Does the contract allocate this design completion risk to the D&C contractor?
- Is there a cost impact, time impact, or both?
- Is certifier review or BASIX / energy reassessment needed?
- Has owner / principal sign-off been obtained before work proceeds?

If a design issue is within the D&C baseline, it may be a D&C obligation rather than a variation. If the owner changes the brief or an authority imposes a new condition outside the baseline, it may be a variation. Label the judgement and cite evidence.

## Escalation routing - D&C contractor

Builder-side escalations follow `role-builder.md`. Design-side escalations route as follows:

| Trigger | Destination | Draft / register | Rule |
|---|---|---|---|
| PPR / owner brief ambiguity | Owner / principal | Owner-facing summary plus decision row | Get instruction before design proceeds on assumption |
| DRM missing or stale | D&C project lead / design manager | Action row plus risk row | Update before certifier submission or construction package release |
| Consultant scope gap | Consultant / owner depending appointment | Consultant advice/action row | Do not assume excluded design scope is covered |
| PI insurance gap | D&C management / broker / owner where risk affects contract | Insurance gap note and risk row | Do not treat CWI or PL as PI |
| Certifier design submission blocked | Certifier / responsible consultant | Authority submission action and design deliverables row | Record revision and response path |
| Compliance interpretation | Certifier / consultant of record | Formal RFI or authority note | Use contractual register |
| Owner-driven design change | Owner / principal | Variation / decision row and owner-facing summary | Do not proceed without written sign-off where required |
| Design risk material to liability | D&C management / PI insurer as appropriate | Risk row and confidential management escalation | Do not bury in project minutes |

The agent must not suppress a design-side escalation to keep setup moving.

## Voice register - D&C defaults

Per `voice-and-style` and AGENTS.md Sec. 6, voice is folder-driven:

- **Contractual register** for certifier submissions, authority correspondence, consultant instructions, design RFIs, design responsibility records, claims, variations, EOTs, formal notices, subcontractor correspondence, and design deliverables.
- **Stakeholder register** for owner-facing explanations where the owner needs a practical decision, cost / time consequence, or recommendation.

A formal design submission to the certifier is contractual. A plain-English owner summary explaining why a certifier submission blocks site start is stakeholder. Both may be needed.

## Multi-dwelling posture

For `archetype: multi-dwelling`, this overlay combines with `multi-dwelling-guide.md`.

The D&C contractor must treat these as early setup questions:

- Is the project repeated Class 1a attached dwellings, Class 2, or mixed?
- Are party-wall / fire-separation details resolved before procurement?
- Are service meters and authority applications per dwelling, per lot, or common-property?
- Are infrastructure contributions and utility charges included in the cost plan?
- Is staging aligned to inspections, OC, subdivision / strata, owner access, and defect close-out?
- Does the design programme show certifier submission dates for each design package and stage?

If the project is a 6-unit townhouse development, do not assume Class 2. Test Class 1a attached dwelling classification first and document the basis.

## Failure modes specific to D&C

- Builder pack complete but PI missing, leaving design liability uninsured or unverified.
- Design responsibility matrix prepared at tender and never updated.
- Consultants appointed without clear deliverables, causing design gaps at certifier submission.
- Design programme not linked to construction programme, so late design releases appear as construction delays.
- Certifier receives stale drawings because revision control failed.
- D&C claims design-fee progress without evidence of design deliverable completion.
- Owner change treated as design development, or D&C baseline design obligation treated as variation.
- Multi-dwelling classification assumed incorrectly, causing wrong NCC volume, fire, accessibility, energy, or inspection pathway.
- Party-wall penetrations and services interfaces left to trades without design responsibility assignment.
- Metering, OSD, infrastructure contributions, or staged OC path discovered after contract sum lock.

## Agent behaviour under this overlay

When `user_role: d-and-c` is declared:

1. The agent loads this overlay, the declared archetype seed, and the required topic seeds through `seed-targeted-read`.
2. The agent also uses `role-builder.md` as the builder obligation base.
3. The agent reads project evidence first under AGENTS.md Sec. 1.
4. The agent labels every design responsibility gap as Fact, Assumption, Judgement, or Recommendation.
5. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
6. The agent opens or refreshes design responsibility, design deliverables, consultant advice, design RFI, authority approvals, risk, action, decision, and subcontractor registers where relevant.
7. The agent surfaces builder-side and design-side escalation triggers.
8. The agent flags state gaps rather than extending NSW D&C assumptions to another state.

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
- `role-builder.md` - builder obligation base inherited by this role
- `multi-dwelling-guide.md` - primary archetype pair for slice 09
- `setup-and-commission-guide.md` - setup workflow deepened by this slice
- `contract-administration-guide.md` - AS 4902 / D&C contract posture
- `procurement-quoting-guide.md` - subcontractor and consultant procurement
- `cost-management-principles.md` - design-fee and multi-dwelling cost posture
- `program-scheduling-guide.md` - design programme and staging
- `../02-skills/atomic/seed-targeted-read.md` - loads this overlay
- `../02-skills/atomic/register-row-draft.md` - consultant advice, design responsibility, design deliverable, action, risk, RFI, and authority rows
- `../02-skills/atomic/markdown-draft-for-review.md` - draft frontmatter and voice / folder matching
- `../02-skills/systems/contract-setup-system.md` - D&C setup path
- `../02-skills/systems/progress-claim-assessment-system.md` - claim assessment where design deliverables affect entitlement
- `../02-skills/systems/variation-management-system.md` - variation control where design baseline is disputed
- `../02-skills/systems/risk-register-system.md` - D&C design and multi-dwelling risk seeding
