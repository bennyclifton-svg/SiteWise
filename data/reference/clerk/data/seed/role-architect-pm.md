---
tier: role-overlay
seed_type: role-overlay
loaded_by: "user_role: architect-pm"
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
state_default: NSW
summary: "Cross-class role overlay for the client-side architect-PM: two-brief discipline, engagement pack, consultant coordination, procurement posture, claims and variation support, escalation routing and client-facing voice register."
doctrine_anchors: [§seed-consultation-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §evidence-discipline, §voice-and-style, §owner-communication]
agents_anchors: [§1, §2, §3, §5, §6, §8, §9, §11]
---

# Role overlay - Architect-PM

The **architect-PM** is the client-side project lead advising the project owner or principal. The role sits between the client, consultants, authorities, certifier, and builder. It carries the discipline of briefing, procurement, consultant coordination, reporting, recommendation, and contract-administration support, but it does not become the builder, certifier, or Superintendent by implication.

This overlay is loaded when the project `README.md` declares `user_role: architect-pm`. It adds role-specific obligations to the SiteWise project-lead doctrine. It does not replace project evidence, the doctrine, or the declared archetype seed.

NSW is the deep default. Registration, insurance, HBCF / HOW, LSL, certifier, and appointment references are current-as-authored guidance only. Before relying on them in a real project, verify current project evidence and current official NSW Architects Registration Board, Building Commission NSW / Fair Trading, HBCF, Long Service Corporation, certifier, council, and Planning Portal requirements.

Residential-only controls in this overlay, including HBCF / HOW and BASIX,
activate only when the confirmed taxonomy and project evidence make them
applicable. They must not be carried into commercial, industrial, advisory or
remediation outputs by analogy. For those projects, use the governing contract,
approval pathway, insurance requirements and class/work-type guides.

## What the architect-PM is - and is not

The architect-PM is:

- **Client-side advisor** translating technical, commercial and delivery issues for the project owner.
- **Brief holder** for the owner's project brief and the architect-PM's own engagement brief.
- **Coordinator** of design, consultants, certifier interface, procurement, owner decisions, and reporting.
- **Evaluator / recommender** for head-builder procurement where the engagement includes that service.
- **Reviewer / assessor** of builder claims, variations, RFIs, programme risk, and authority risk where the engagement and contract appointment allow.
- **Keeper of the owner-facing decision trail** so the owner can decide quickly without reconstructing the project history.

The architect-PM is not:

- The builder. The architect-PM does not take out builder-side HBCF / HOW, pay builder-side LSL, bind the builder's contract works insurance, or hold the builder's contractor licence for the construction works.
- The Certifier. The certifier / principal certifier is appointed separately and must remain independent. Do not imply certification authority from an architect-PM engagement.
- The Superintendent unless expressly appointed in writing under a contract form that uses that role, and unless the architect-PM is competent and insured for that scope.
- The owner's silent decision-maker. The architect-PM recommends; the owner decides unless an express authority matrix says otherwise.
- A substitute for project evidence. A recommendation based on missing builder evidence is an Assumption and must be labelled as such.

## Role declaration

Every architect-PM project needs a role-declaration record at setup. The declaration states:

- whether the architect-PM is acting as architect, project manager, contract administrator, Superintendent, Certifier, or none of those formal roles;
- the instrument that creates each role: engagement letter, scope of services, head contract, consultant appointment, or statutory appointment;
- the authority boundary for each role: advise, recommend, issue instruction, assess, certify, approve, or observe only;
- the insurance basis for the role, especially professional indemnity cover and any exclusion for certification or Superintendent functions;
- who signs owner decisions and who can issue formal contract directions.

If the role is unclear, setup does not treat the gap as paperwork. It is an escalation because authority drift creates claim, certification, and owner-trust risk.

## Engagement pack

The architect-PM's engagement pack is the role's setup instrument. It is filed under `00-brief-pmp/` and referenced by procurement, cost, programme, risk, and contract-administration drafts.

Minimum engagement evidence:

- accepted fee proposal;
- executed engagement letter or services agreement;
- scope of services, including inclusions, exclusions, deliverables, meetings, reporting cadence, procurement services, and contract-administration services;
- fee basis, reimbursables, variation-to-service mechanism, and payment schedule;
- Project Management Plan (PMP) or equivalent;
- governance map: who decides, who recommends, who is consulted, who is informed;
- communications protocol: owner update cadence, builder RFI route, authority / certifier route, emergency contact route;
- role declaration, including Superintendent / Certifier status;
- conflict disclosure where the architect-PM recommends preferred builders, consultants, suppliers, or related parties;
- PI insurance evidence for the advisory scope and project period.

The fee proposal is not the project brief. It is evidence of the architect-PM's commission. The project brief is the owner's building brief and must be recorded separately.

## Two-brief discipline

Architect-PM setup uses two briefs:

1. **Architect-PM engagement brief** - the contract between owner and architect-PM: fee proposal, scope of services, PMP, role declaration, reporting cadence, and authority matrix.
2. **Owner's project brief** - what the owner wants built, why, to what budget and programme, and with what planning / approval assumptions.

Do not merge them. If the owner asks for a bigger kitchen, that belongs in the project brief and decision register. If the owner asks the architect-PM to run an extra tender round, that belongs in the engagement / services record as a service-scope change.

## Statutory and professional posture - NSW deep default

### Architect registration

Where the practice or individual holds themselves out as an architect, record the architect registration evidence and responsible registered architect. The setup record should capture the registration name / number, expiry or renewal status where evidenced, and whether the architect-PM is providing architectural services, project management services, or both.

If the user is acting as a project manager but not using the protected architect title or not providing architectural services, record that distinction rather than forcing an architect-registration row.

### Professional indemnity insurance

Professional indemnity insurance is the core architect-PM insurance evidence. At setup, record:

- policy holder;
- policy period;
- limit of indemnity;
- retroactive date if relevant;
- project-specific exclusions;
- whether contract administration, Superintendent functions, design services, procurement advice, and authority submissions are within scope;
- whether any proposed special condition creates uninsured advisory or certification exposure.

If PI currency is missing, excluded, or not aligned with the project period, surface an escalation to the owner-facing summary before recommending procurement, contract signature, or site start.

### Builder-side instruments verified, not held

The architect-PM does **not** take out builder-side HBCF / HOW, pay builder-side LSL, bind the builder's contract works insurance, or hold the builder's contractor licence for the construction work.

The architect-PM must verify the builder's evidence before recommending contract signature, payment, mobilisation, or site start:

- builder contractor licence and Qualified Supervisor evidence;
- HBCF / HOW eligibility and per-project certificate where required;
- LSL receipt or confirmed payment pathway before CC where applicable;
- executed head contract and special conditions;
- contract works insurance naming the correct insured parties where required by the contract;
- public liability insurance and workers compensation where applicable;
- BASIX certificate, CC / CDC / DA path, certifier appointment, and authority prerequisites;
- construction management, waste management, traffic management, and other consent-condition management plans where site-start relevant.

Verification is evidence work. If the evidence is missing, the architect-PM may explain the risk and recommend next steps, but must not present the item as satisfied.

### Certifier and authority path

The architect-PM often coordinates the owner, certifier, consultants, and council. Coordination is not certification. Authority and certifier correspondence is filed under `04-planning-and-authorities/` and uses contractual register unless it is a separate owner-facing explanation.

Authority assumptions must be labelled:

- planning pathway unknown;
- CC / CDC status not evidenced;
- BASIX certificate not issued or not aligned with the drawings;
- consent conditions not extracted;
- certifier appointment not evidenced;
- Sydney Water / utility approvals not evidenced.

Each assumption creates an action row, risk row, or owner decision summary depending on what it blocks.

## Architect-PM setup workflow

When `contract-setup-system` runs for `user_role: architect-pm`, setup is not "builder mobilisation" alone. It is the point where the architect-PM can show: the commission is agreed, the role boundary is declared, the owner brief is recorded, the consultant and authority pathway is visible, and the builder evidence needed for award or site start is verified or gap-reported.

Minimum setup pack:

1. Complete README frontmatter: `archetype`, `user_role`, and `state` declared and not `TBC`.
2. Architect-PM engagement pack filed under `00-brief-pmp/`.
3. Owner project brief filed separately under `00-brief-pmp/`.
4. Role declaration recorded, including Superintendent / Certifier / contract administrator status.
5. Architect registration evidence recorded where relevant.
6. PI insurance currency and scope recorded.
7. Consultant scope and responsibility matrix opened where consultants are involved.
8. Authority / certifier pathway recorded with assumptions labelled.
9. Builder evidence verification tracker opened for licence, HBCF / HOW, LSL, insurance, contract, BASIX / CC, and site-start evidence.
10. Owner reporting cadence and escalation route confirmed.
11. Ready-to-start checklist issued as a draft for human review.

## Consultant coordination

Architect-PM projects often depend on a consultant web: architect, structural engineer, hydraulic consultant, BASIX assessor, surveyor, geotechnical engineer, certifier, landscape, traffic, heritage, or other specialists depending on archetype and approval path.

The architect-PM must keep consultant scope visible:

- who is appointed by the owner;
- who is appointed by the builder or D&C contractor;
- scope boundaries and deliverables;
- fee basis and excluded services;
- design responsibility matrix where multiple consultants touch the same element;
- advice register for material consultant advice;
- design RFI register for open questions;
- deliverables schedule and current revision status;
- authority or certifier submissions each consultant must support.

Consultant advice that affects cost, time, scope, quality, or compliance becomes a decision, risk, action, variation, or owner update. Do not leave it buried in minutes.

## Procurement posture

Where the architect-PM is engaged to select a head builder, load `procurement-quoting-guide.md` and use `procurement-evaluation-system` Branch B.

Architect-PM procurement posture:

- run a fair, auditable process proportionate to residential scale;
- disclose conflicts or repeat-builder relationships before tender invitation;
- keep RFI and addenda discipline where a formal tender is open;
- agree evaluation criteria and weights with the owner before tender close;
- normalise scope and departures before comparing prices;
- check builder licence, HBCF / HOW capacity, insurance, programme, methodology, references, and BASIX / compliance method;
- provide a clear recommendation to the owner, not a bundle of options with no view;
- state conditions of award and residual risks plainly.

Owner-facing recommendation summaries live in stakeholder register and follow `owner-communication`. Formal tender packs, addenda, evaluation matrices, and internal recommendation reports live in contractual register.

## Progress claims, variations, and contract administration

Architect-PM involvement in progress claims and variations depends on the engagement and the head contract.

The architect-PM may:

- assess or review a builder's progress claim and recommend payment, part-payment, withholding, or evidence request;
- prepare an owner-facing explanation of the recommendation;
- draft a formal assessment record where appointed;
- assess variations for scope, cost, time, BASIX / approval impact, and owner sign-off requirement;
- recommend an EOT response or owner decision pathway;
- coordinate consultant input to technical claims or variations.

The architect-PM must not:

- certify a claim unless the contract and appointment give that authority;
- approve a variation on the owner's behalf without express authority;
- issue a Superintendent direction unless appointed as Superintendent under the relevant contract;
- replace executed contract clause text with generic seed knowledge in a clause-cited assessment;
- let an owner decision happen verbally without recording the decision, basis, cost/time consequences, and follow-up.

Formal assessments stay contractual. Owner explanations sit separately in stakeholder voice under `08-meetings-reporting/owner-update*` or another owner-facing destination.

## Escalation routing - architect-PM

The doctrine says when to escalate. This overlay says where the escalation goes.

For architect-PM projects, escalation defaults to a formatted owner-facing summary using `owner-communication`: practical consequence first, explicit owner ask second, facts and background below.

| Trigger | Destination | Draft / register | Rule |
|---|---|---|---|
| Owner decision required | Owner | `08-meetings-reporting/owner-update*` or decision summary | Include recommendation, due date, cost/time/risk consequence |
| Builder evidence missing | Owner-facing summary plus action row | Authority / insurance tracker and action register | Do not recommend contract signature or site start as ready |
| Role ambiguity | Owner-facing summary plus role-declaration action | `00-brief-pmp/role-declaration.md` and action row | Clarify before issuing formal directions or assessments |
| Consultant scope gap | Owner-facing summary plus consultant advice/action row | `02-consultant/` register | Do not assume a consultant covers excluded scope |
| Authority / certifier uncertainty | Authority correspondence plus owner summary if decision needed | `04-planning-and-authorities/` and `08-meetings-reporting/` | Separate formal authority record from owner explanation |
| Contract notice or RFI needed | Builder / certifier / consultant per contract or appointment | `07-construction/08-rfi-notices/` or `04-planning-and-authorities/` | Use contractual register; cite contract or authority basis where evidenced |
| Budget / contingency movement | Owner-facing summary plus cost/risk row | `08-meetings-reporting/owner-update*` and cost/risk registers | Give a recommendation, not just a warning |

The agent must refuse to suppress an escalation trigger. If the architect-PM sees a trigger, the owner sees it in a form they can act on.

## Voice register - architect-PM defaults

Per `voice-and-style` and AGENTS.md Sec. 6, voice is folder-driven, with role-specific emphasis:

- **Stakeholder register**: owner updates, monthly owner summaries, procurement recommendation summaries to owner, cost / programme / risk summaries for owner decisions, and any note whose purpose is to help a non-technical owner decide.
- **Contractual register**: builder-facing RFIs, formal notices, authority / council correspondence, certifier submissions, tender packs, addenda, formal evaluation matrices, formal advice records, claim assessments, variation assessments, and anything relying on contract or authority mechanisms.

The same underlying issue may need two documents: a formal contractual record for the project file and a stakeholder-register owner summary. Do not collapse them where the audiences differ.

Owner-facing summaries follow `owner-communication`:

1. What this means for you.
2. What we need from you.
3. What's happened.
4. What's next.
5. Background detail.

Do not lead an owner update with clause references, technical chronology, or a three-option decision with no recommendation.

## Setup checklist - architect-PM mobilisation

When the setup system runs for `user_role: architect-pm`, the ready-to-start checklist includes:

- [ ] README frontmatter declares `archetype`, `user_role`, and `state`.
- [ ] Accepted fee proposal and executed engagement letter filed.
- [ ] Scope of services filed, including exclusions and contract-administration authority.
- [ ] PMP or equivalent governance plan filed.
- [ ] Owner project brief filed separately from the engagement pack.
- [ ] Role declaration recorded: architect / project manager / contract administrator / Superintendent / Certifier / neither.
- [ ] Architect registration evidence filed where relevant.
- [ ] PI insurance currency, limit, period, and exclusions recorded.
- [ ] Consultant scopes, responsibility matrix, and advice register opened where consultants are involved.
- [ ] Authority / certifier pathway recorded, including DA / CDC / CC, BASIX, principal certifier, and consent-condition assumptions.
- [ ] Builder licence and Qualified Supervisor evidence verified before recommendation / contract signature.
- [ ] Builder HBCF / HOW eligibility and per-project certificate verified where required.
- [ ] Builder LSL receipt or payment pathway verified where CC / site start depends on it.
- [ ] Builder contract works, public liability, and workers compensation evidence verified where applicable.
- [ ] Executed head contract and special conditions reviewed or contract-execution gap recorded.
- [ ] Programme baseline and owner reporting cadence agreed.
- [ ] Owner escalation route confirmed: owner-facing summary with recommendation and due date.

Architect-PM HOW / HBCF lodgement, builder-side LSL payment, builder licence, and contract works insurance are **not** personal architect-PM checklist items. They are builder evidence items the architect-PM verifies.

## Failure modes specific to architect-PM

- Acting as de facto Superintendent without appointment or PI cover.
- Letting a certifier interface imply certification responsibility.
- Recommending contract signature before builder HBCF / HOW, licence, insurance, or contract evidence is verified.
- Treating LSL as the builder's problem only, then discovering the CC / site-start pathway is blocked.
- Burying the owner ask in a technical report.
- Presenting the owner with three options and no recommendation.
- Giving formal authority or builder correspondence in casual stakeholder language.
- Giving owner summaries in contractual language that the owner cannot act on.
- Not disclosing a preferred-builder relationship before tender.
- Consultant scope gaps discovered after tender because responsibility was not mapped at setup.

## Agent behaviour under this overlay

When `user_role: architect-pm` is declared:

1. The agent loads this overlay and the declared archetype seed on any phase-gate task.
2. The agent enforces the README declaration gate before drafting.
3. The agent keeps the engagement brief and owner project brief separate.
4. The agent records role declaration before treating the architect-PM as Superintendent, Certifier, or contract administrator.
5. The agent verifies builder-side HBCF / HOW, LSL, licence, insurance, and contract evidence but does not treat those as architect-PM instruments.
6. The agent routes owner decisions through an owner-facing summary with recommendation.
7. The agent uses stakeholder register for `08-meetings-reporting/owner-update*` and contractual register for `04-planning-and-authorities/**` and `07-construction/08-rfi-notices/**`.
8. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
9. The agent flags state gaps rather than extending NSW architect-PM assumptions to another state.

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
- `new-dwelling-guide.md` / `renovation-guide.md` / `multi-dwelling-guide.md` / `ancillary-guide.md` / `small-commercial-guide.md` - Tier 2 archetype seeds
- `setup-and-commission-guide.md` - setup workflow deepened by this slice
- `contract-administration-guide.md` - contract / variation / notice context
- `procurement-quoting-guide.md` - head-builder selection and recommendation discipline
- `cost-management-principles.md` - cost reporting and owner escalation
- `program-scheduling-guide.md` - programme reporting and date-risk escalation
- `../02-skills/atomic/seed-targeted-read.md` - loads this overlay
- `../02-skills/atomic/markdown-draft-for-review.md` - voice / folder gate used for owner updates and formal correspondence
- `../02-skills/atomic/register-row-draft.md` - action, decision, consultant advice, authority, RFI, and risk rows
- `../02-skills/systems/contract-setup-system.md` - architect-PM setup path
- `../02-skills/systems/procurement-evaluation-system.md` - Branch B formal head-builder selection
- `../02-skills/systems/progress-claim-assessment-system.md` - claim assessment / recommendation support
- `../02-skills/systems/variation-management-system.md` - variation assessment / recommendation support
- `../02-skills/systems/risk-register-system.md` - risk review and escalation routing
