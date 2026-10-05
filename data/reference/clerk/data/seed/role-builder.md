---
tier: role-overlay
seed_type: role-overlay
loaded_by: "user_role: builder"
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
summary: "Role overlay for the head-contract builder: statutory instruments (licence, HOW/HBCF, LSL), head-contract administration, progress-claim and variation issuance, EOT, subcontractor management and the builder mobilisation checklist."
doctrine_anchors: [§seed-consultation-discipline, §register-discipline, §decision-discipline, §escalation-triggers, §evidence-discipline, §voice-and-style]
agents_anchors: [§1, §2, §3, §6, §9]
required_by: {head-contractor-procurement: 3}
---

# Role overlay — Builder (head contractor)

> **Retired — not loaded by the knowledge catalog as of 2026-07-24.** Role is no longer a user-facing overlay; every project uses the single `role-architect-pm.md` project-lead overlay. This file is retained for reference only.

The **builder** holds the head contract with the owner. The builder is the residential head contractor — sole contractual interface to the owner for construction, sole party responsible for statutory builder instruments (licence, HOW/HBCF, LSL, insurance), and the party that issues progress claims and assesses subcontractor claims.

This overlay is loaded when the project `README.md` declares `user_role: builder`. It adds builder-specific obligations on top of the abstract `project lead` doctrine in `../00-doctrine/doctrine.md`. It does **not** replace the doctrine — the doctrine still holds. Where doctrine and overlay diverge, the overlay carries role-specific weight; where the overlay is silent, the doctrine spine applies.

NSW is the deep default throughout. Non-NSW states get inline callouts where the instrument or pathway differs materially. Where a non-NSW state has no callout for the task at hand, **flag the gap** (per `../00-doctrine/doctrine.md §state-handling`) rather than silently extending NSW guidance.

## What the builder is — and is not

The builder is:
- **Principal-facing head contractor** under an HIA / MBA / NSW Fair Trading prescribed / AS 4000 / AS 4902 contract. Sole party signing the head contract with the owner.
- **Holder of the builder's licence** (NSW Fair Trading Contractor Licence for residential work over $5,000; Qualified Supervisor certificate held by the licensee or by a nominated supervisor).
- **Holder of HOW / HBCF cover** for the owner's protection (NSW: HBCF eligibility and certificate per project for residential work over $20,000).
- **LSL payer** before CC for projects at or above the current NSW Long Service Levy threshold (currently $250,000 including GST, verify before use; NSW Long Service Corporation).
- **Contract works and public liability insurer.**
- **Issuer of progress claims** to the owner under the head contract's payment mechanism.
- **Assessor of subcontractor claims** under the subcontract chain.
- **Issuer of variations** to the owner under the head contract's variation mechanism, and **assessor** of subcontractor variations.
- **Party giving EOT notices** to the owner under the head contract; **party assessing** EOT notices from subcontractors.

The builder is **not**:
- The Certifier. The certifier (Principal Certifier under NSW EP&A Act, formerly PCA) is appointed separately by the owner. The builder must not act as Certifier.
- The Superintendent. AS 4000 / AS 4902 use the term; HIA does not. Where a Superintendent is named, that role sits with the owner's appointed advisor (often an architect-PM). The builder must not direct the Superintendent.
- A consultant. The builder does not give design advice unless the contract is D&C (in which case load `role-d-and-c.md`, not this overlay).
- The owner-builder's substitute. A builder contracted by an owner-builder is still a builder under this overlay; the owner-builder remains the principal.

## Statutory instruments — NSW deep default

This section names what the builder must hold, when, and in what evidence form. The `setup-and-commission-guide.md` cross-cutting seed gives the end-to-end commissioning workflow; this section is the **builder-specific requirements list** that workflow needs.

### Builder's licence

- **NSW:** Contractor Licence for residential building work valued over $5,000. Issued by NSW Fair Trading (Service NSW portal). Class must match scope (Class 1 = building work; subspecialty classes for specialist trades).
- **Qualified Supervisor:** the licence holder or a nominated supervisor must hold the Qualified Supervisor certificate. The Qualified Supervisor must be physically able to supervise the work — multiple concurrent projects must be within reasonable supervision span.
- **Evidence captured at mobilisation:** licence number, class, expiry date, Qualified Supervisor name and certificate number. Filed under `00-brief-pmp/`.
- **In VIC:** the equivalent is Domestic Builder registration through the Victorian Building Authority (VBA), with categories DB-U (unlimited) and DB-L (limited). In QLD: QBCC licence. In SA: Building Work Contractors Licence (CBS).

### HOW / HBCF (Home Building Compensation Fund) — owner protection insurance

- **NSW:** required for any residential building work over $20,000 done by a licensed builder. Administered by iCare HBCF (formerly Home Owners Warranty / HOW). Certificate is per project, lodged before any deposit is taken and before contract is signed.
- **Threshold note:** the $20,000 figure is the legislative trigger. Below threshold, the builder is not required to take out cover; the work is still licensed work. Many small renovations and ancillary jobs sit either side of the threshold — confirm scope and contract sum before assuming exempt.
- **Eligibility prerequisite:** the builder must hold an iCare HBCF eligibility approval (the builder is "rated") before a per-project certificate can be issued. Eligibility is a pre-mobilisation builder-side instrument; the project-level certificate is the mobilisation instrument.
- **Evidence captured at mobilisation:** HBCF certificate (PDF), per project, naming the owner as beneficiary and stating the contract sum. Filed under `00-brief-pmp/` and indexed in the authority approvals tracker in `04-planning-and-authorities/`. A copy is delivered to the owner before the contract is signed.
- **Owner-builder note:** if the owner is themselves an owner-builder under permit (no contracted builder for the works), HOW/HBCF does not apply — the owner-builder is exempt because they cannot insure themselves. Load `role-owner-builder.md` for that case, not this overlay.
- **In VIC:** the equivalent is Domestic Building Insurance (DBI) under the Building Act and the Domestic Building Contracts Act, administered by the VMIA. Threshold is $16,000.
- **In QLD:** the equivalent is QBCC Home Warranty Insurance; threshold is $3,300 — materially different from NSW.

### LSL — Long Service Levy

- **NSW:** Long Service Corporation levy at 0.25% of total cost of building and construction work for projects valued at $250,000 or more including GST (current-as-authored; verify before use). Levy payable before Construction Certificate (CC) issue where the project triggers it.
- **Mechanic:** levy receipt (the "LSL Receipt") is one of the documents the certifier requires before issuing the CC. A CC will not issue without it. This makes LSL a **CC-blocking** instrument, not a nice-to-have.
- **Evidence captured at mobilisation:** LSL Receipt (PDF) naming the project, contract sum, and date paid. Filed under `04-planning-and-authorities/` and referenced in the authority approvals tracker.
- **In VIC:** the equivalent is CoINVEST contributions (different mechanism — employer-side ongoing contributions rather than a per-project levy). In QLD: QLeave levy at 0.475% of project cost ≥ $150,000.

### Contract works insurance (CWI)

- **NSW:** Contract Works (also called Construction All Risks) insurance is a contractual requirement under HIA / MBA contracts. Covers loss or damage to the works during construction.
- **Named insureds:** policy names the builder and the owner as joint insureds (HIA Lump Sum form requires this). Where consultants have site presence, they may be named additionally.
- **Period:** from site possession to practical completion, with run-off cover through DLP commonly included.
- **Sum insured:** the contract sum plus any owner-supplied items, plus removal-of-debris and professional fees. Under-insurance is a classic builder failure mode.
- **Evidence captured at mobilisation:** Certificate of Currency (PDF) naming the project, the named insureds, the sum insured, the period of cover, and the policy number. Filed under `07-construction/03-insurance-bgs/`. A copy is delivered to the owner before site possession.

### Public liability insurance (PL)

- Industry-standard minimum for residential head contractor work is **$10M** PL. Some owners or certifiers require $20M.
- **Evidence captured at mobilisation:** Certificate of Currency (PDF) naming the policy holder, sum insured, period, and policy number. Filed under `07-construction/03-insurance-bgs/`.
- PL covers third-party injury and property damage — distinct from CWI, which covers the works themselves. Both are required.

### Workers compensation

- **NSW:** mandatory for any builder employing workers; managed via icare workers insurance. Sole-trader builders without employees are not required to hold it but must have personal accident cover.
- **Evidence captured at mobilisation:** workers comp policy number and currency. Filed under `07-construction/03-insurance-bgs/`.

### BASIX

- **NSW:** Building Sustainability Index. Commitments are locked at DA / CC stage. The builder inherits the BASIX commitments from the certifier-issued CC plans and the BASIX Certificate.
- **Builder obligation:** deliver to the locked commitments (window U-values and SHGC, wall and ceiling R-values, ventilation, hot water system, PV size). Substitution requires re-submission to the BASIX assessor and an updated certificate before installation — substitution without re-certification is a classic delivery failure that surfaces at the BASIX final inspection and blocks OC.
- **Evidence captured during construction:** product datasheets matching BASIX commitments, installation photos, BASIX assessor's final inspection and certificate. Filed under `04-planning-and-authorities/` and `07-construction/10-commissioning/`.

## Head contract — issuance and administration obligations

The builder's contractual posture under each common NSW residential head contract is summarised below. For full clause-by-clause coverage, load `contract-administration-guide.md`. This section gives the builder-side posture only.

### HIA Lump Sum (NSW New Homes)

- Primary contract family for new Class 1a builds in NSW.
- Owner pays a deposit (typically 5%, capped by statute) on signing.
- Progress claims are issued by the builder against the **stage-payment schedule** — typically: deposit / base or slab / frame / lockup / fixing / completion. The percentages vary by contract; the standard HIA Lump Sum has fixed defaults that can be edited by special conditions.
- **Claim issuance discipline (builder side):** a claim is issued only after the stage is physically achieved. "Physically achieved" means trade evidence — slab pour conformance test for the slab stage, frame inspection sign-off for the frame stage, lockup defined by the contract (typically external envelope sealed, windows in, external doors lockable).
- **Variation issuance:** variations under HIA use the HIA Schedule of Variations form. Builder issues a priced variation **before work begins**; owner signs; work proceeds. A variation worked before written direction and signed price is a §variation-management failure and is recoverable only by argument — the §register-discipline trail will show it was unwritten.
- **EOT:** notice given by the builder under the contract clause within the contract's notification window, supported by contemporaneous programme and contemporaneous evidence of the delay cause (weather records, supplier correspondence, site photos).

### HIA Cost Plus (NSW)

- Used where scope is genuinely uncertain at contract — typically high-end residential, complex renovations, owner-builder hybrid.
- Builder is paid actual cost plus a margin. Invoicing is per construction-period rather than per stage.
- **Claim issuance discipline (builder side):** every cost claimed must be substantiated by invoice or wage record. The owner is entitled to inspect records.
- Owner-builder hybrid use of Cost Plus is a known dispute hotspot — the owner-builder retains principal authority but pays the builder cost-plus for selected packages. Document scope and authority precisely.

### HIA Renovation (NSW)

- Variant of Lump Sum specifically scoped for renovations and additions. Same stage-payment mechanism but with stages adapted (often: deposit / preparation / structural / lockup / fixing / completion) and with explicit latent-condition allowances.
- **Latent conditions:** the contract typically defines latent conditions and the notice procedure. Builder must give prompt notice and not proceed beyond the latent condition until the owner directs.

### MBA equivalents

- Master Builders Association NSW publishes equivalents to HIA Lump Sum, Cost Plus, and Renovation. Mechanically similar; clause numbering differs. Where MBA forms are used, cite MBA clause numbers — do not substitute HIA clause numbers.

### NSW Fair Trading prescribed contract

- The Office of Fair Trading publishes a small-jobs contract for residential work between $5,000 and $20,000. Below $20,000 a written contract is still required (above $5,000); the prescribed form satisfies the requirement.
- Mechanically thinner than HIA / MBA — fewer variation, EOT, and payment provisions. Suitable for short-duration small-value work; not suitable for a new build.

### AS 4000 / AS 4902 (high-end residential)

- Used for high-end residential, especially where a Superintendent is appointed and where the owner wants AS-family rigour rather than HIA simplicity.
- The builder takes on AS-style obligations: contemporaneous claims, Superintendent directions in writing, formal EOT mechanics, security in the form of bank guarantee or retention.
- If AS 4902 (D&C variant) is used, load `role-d-and-c.md` instead of this overlay.

### ABIC

- Australian Building Industry Contracts (RAIA / MBA). Rarely used in NSW residential. Treat as available but not default — if the owner's advisor has chosen ABIC, follow the chosen form; do not advocate switching mid-project.

## Progress claims — builder issuance side

The **assessment** side (assessor receiving the claim from a subcontractor, or the owner's advisor receiving the builder's claim) is the responsibility of `progress-claim-assessment-system` in slice 06. This overlay covers the **issuance** side: what the builder does when raising a claim.

The builder must do:
- Issue claims **only after** the stage is physically achieved (Lump Sum / Renovation) or for actual costs incurred and evidenced (Cost Plus).
- Match every claim to the contract's payment mechanism — claim form, due date relative to the claim period, supporting documentation list.
- Attach trade evidence: inspection sign-offs, conformance certificates (slab CFT, structural inspection certificate), installation photos.
- Issue under §voice-and-style **contractual register** — claim cover letters cite the contract clause and the stage definition. The covering correspondence is filed under `07-construction/05-progress-claims/`.
- Maintain the claim register row per §register-discipline (claim number, stage, value claimed, value certified, status, date issued, date due, date paid).

The builder must not:
- Claim a stage that is not physically achieved. This is a §evidence-discipline breach and exposes the builder to HBCF investigation and adverse PC outcomes.
- Submit a claim without the supporting evidence the contract requires.
- Allow stage definitions to drift between project and contract — the slab stage in the contract is the slab stage on site.

Standard deliverables:
- progress claim form (per contract);
- claim cover letter (contractual register, citing the relevant clause);
- supporting evidence pack (inspections, certificates, photos);
- claim register row entry (per `register-row-draft`).

Common failure modes:
- claim issued before stage physically achieved;
- claim issued without trade evidence;
- claim register lapses, leaving the builder unable to reconstruct the payment trail at PC;
- HBCF claim later by owner reveals the stage was claimed before achievement — the worst-case outcome.

## Variations — builder issuance side

The builder issues variations to the owner and assesses variations from subcontractors. Issuance side here.

The builder must do:
- Issue every variation in writing **before work begins** — HIA Schedule of Variations form or equivalent.
- Price the variation (cost + time impact) and obtain owner signature before commencing work.
- Maintain the variation register per §register-discipline (variation number, description, cost, time impact, status, date issued, date owner-signed, date completed).
- File the signed variation form under `07-construction/06-variations/` and the cost-tracking row in the project's cost register.

The builder must not:
- Begin work on a variation before written direction and signed price. The HIA Lump Sum contract is explicit on this — unwritten variations are recoverable only by argument.
- Allow informal site agreements between site supervisor and owner to become variations without going through the variation form. The §evidence-discipline trail must show owner sign-off on the priced variation.
- Bundle multiple unrelated changes into a single variation. Each scope item is a separate variation row so that pricing and status are individually trackable.

Standard deliverables:
- variation form (HIA Schedule of Variations or equivalent, signed by owner);
- variation register row;
- variation cost-tracking entry in the cost register;
- programme update if time impact applies (link to EOT register if EOT is also claimed).

Common failure modes:
- variation worked before signed direction;
- variation priced after the fact, when bargaining position is gone;
- variation register dies during the busy mid-construction period;
- BASIX-impacting variations (window swap, HW system change) made without re-certification — surfaces at the BASIX final inspection.

## EOT — builder issuance side

The builder issues EOT notices under the head contract. The EOT clause varies by contract:

- **HIA Lump Sum / Renovation:** notice given within the contract's notification window (typically 10 working days from the cause becoming apparent), supported by contemporaneous evidence. The contract specifies what counts as a qualifying delay event — weather above a defined threshold, owner-caused delay, latent condition, authority delay outside the builder's control.
- **AS 4000 / AS 4902:** more formal EOT mechanics. Notice within the time bar, supported by contemporaneous programme showing critical-path impact.
- **NSW Fair Trading prescribed:** thinner — typically requires written notice but with less prescription.

The builder must do:
- Give notice within the contract's time bar. **Late notice is fatal** under most contracts — the EOT entitlement is barred even if the underlying delay event is valid.
- Maintain a contemporaneous programme. EOT claims rely on demonstrating critical-path impact; a stale programme cannot do that.
- Record the delay cause contemporaneously — daily site diary, weather records, supplier correspondence, dated photos.
- Maintain the EOT register per §register-discipline (EOT number, cause, days claimed, days granted, status, date noticed, date assessed).

The builder must not:
- Bank a delay event and claim it later. Time bars kill late claims.
- Rely on the project lead's verbal acknowledgement — the EOT must go to the owner's appointed representative (or the owner directly) in writing.
- Conflate EOT (time relief) with delay damages (cost claim for delay). Time relief is one mechanism; cost recovery for delay is a separate claim with separate prerequisites.

Standard deliverables:
- EOT notice (contractual register, citing the relevant clause and the notification window);
- supporting evidence pack (programme excerpt, weather record, photos, correspondence);
- EOT register row;
- programme update reflecting the granted (or pending) days.

Common failure modes:
- late notice — time bar engaged;
- programme not contemporaneous, so critical-path impact cannot be shown;
- delay banked through multiple events, then claimed in a single composite — assessment becomes impossible.

## Escalation routing — builder

The doctrine §escalation-triggers anchor sets the **when**. This overlay sets the **where**: routing the trigger to the right destination under an HIA / MBA / AS contract.

The builder must route:

| Trigger | Destination | Form | Filed under |
|---|---|---|---|
| Owner decision required (scope, variation, finish selection, owner-supplied item) | Owner directly, or owner's appointed representative if named in the contract | Written request, plain English with technical summary if needed (§owner-communication may apply) | `07-construction/08-rfi-notices/` |
| Technical question (structural, hydraulic, BASIX, certification) | Consultant of record / engineer of record / BASIX assessor / certifier | Written RFI citing the drawing reference and the question | `07-construction/08-rfi-notices/` (RFI register) |
| Contract notice required (EOT, variation, latent condition, suspension, payment dispute) | Owner / owner's representative per the contract's notice clause | Formal contract notice (contractual register, clause-cited) | `07-construction/08-rfi-notices/` (contractual notices subfolder or labelled by document type) |
| Authority decision required (CC amendment, BPA, OC) | Certifier / authority | Formal submission per authority procedure | `04-planning-and-authorities/` |
| Safety risk | Stop work, then route per WHS escalation | Whatever the WHS framework requires | `07-construction/04-management-plans/` and incident records |
| Compliance risk (BASIX deviation, NCC interpretation) | Certifier / BASIX assessor | RFI or formal request for confirmation | `04-planning-and-authorities/` (BASIX-related) or `07-construction/08-rfi-notices/` (general compliance) |

The builder must not:
- Take an owner decision on the owner's behalf. If the owner does not respond within a reasonable period, escalate (chase letter, then formal notice) rather than presuming consent.
- Route technical questions to the owner where the consultant of record is the right destination. The owner is not the engineer.
- Allow informal site conversations to substitute for the formal route — the contract record is what survives at PC and any dispute.

**The agent must refuse to suppress an escalation trigger** (per §escalation-triggers). If the agent sees a trigger and the builder is mid-flow, the agent surfaces it.

## Voice register — builder defaults

Per §voice-and-style and AGENTS.md §6, voice is folder-driven. For the builder, the defaults that matter most:

- **Contractual register** is the default for everything under `07-construction/05-progress-claims/`, `07-construction/06-variations/`, `07-construction/07-programme-eot/`, `07-construction/08-rfi-notices/`, and for any correspondence to the certifier or authorities under `04-planning-and-authorities/`.
- **Stakeholder register** is the default for owner-facing summaries under `08-meetings-reporting/owner-update*` and for any plain-English handoff (e.g. an owner explanation of a complex variation, even though the formal variation form itself is contractual).

A builder writing a variation form uses contractual register. A builder writing an owner email explaining the variation uses stakeholder register and may attach the contractual form. Both registers can coexist on the same project day — what matters is the document type and folder location, not the topic.

## Communications with the owner

The builder communicates with the owner constantly: variations, claims, EOT, site changes, supplier delays, finish selections. The doctrine spine §owner-communication anchor gives the format (what this means for you / what we need from you / what's happened / what's next / background detail). The builder-specific application:

- **Plain-English first, contractual attachment second.** A variation should be explained in stakeholder register in the body of the email; the contractual variation form is attached.
- **Lead with the ask if one exists.** "We need your signed variation by Friday to keep the framing programme" beats "Please find attached variation #4 for your consideration".
- **Owner-supplied items** are a recurring source of friction. Track them in a dedicated register with name, supplier, expected on-site date, and impact-if-late noted. Communicate slippage early.
- **Never substitute owner conversation for the formal record.** A verbal owner agreement to a variation must be followed by a written variation form before work begins.

## Subcontractor management — builder posture

The builder assesses subcontractor claims, issues subcontracts, manages subcontractor variations and EOT. This overlay flags the role; the operational systems (`progress-claim-assessment-system`, `variation-management-system`) sit in slice 06 and inherit from this overlay.

The builder must:
- Issue written subcontracts before site possession. Verbal subcontracts are common in residential and a recurring dispute source.
- Maintain a subcontractor register (name, scope, contract value, licence number, insurance currency, key dates).
- Assess subcontractor progress claims against physical completion of the subcontract scope, not against the head contract stage. The two are linked but not identical.
- Pass on owner directions to the subcontractor through the contract chain — direct owner-to-subcontractor instruction is a classic head-contract breach.
- Maintain trade insurance and licence currency throughout the subcontract.

The builder must not:
- Let an owner direct a subcontractor without going through the builder.
- Pay a subcontractor for work that has not been done, even on relationship grounds.
- Allow a subcontractor to start before insurance and licence are evidenced.

## Setup checklist — builder mobilisation

When the agent runs `contract-setup-system` for a `user_role: builder` project, the deliverable is a **ready-to-start checklist**. The checklist's builder-specific items are:

- [ ] Builder's licence verified (number, class, expiry, Qualified Supervisor)
- [ ] HBCF eligibility confirmed and per-project HBCF certificate issued, copy delivered to owner
- [ ] LSL paid for the project value; LSL Receipt filed and indexed
- [ ] Head contract executed (HIA / MBA / NSW Fair Trading / AS family); special conditions reviewed
- [ ] Contract works insurance bound; certificate of currency filed; owner named as joint insured
- [ ] Public liability insurance bound (≥ $10M); certificate of currency filed
- [ ] Workers compensation bound (where employees apply); certificate filed
- [ ] BASIX certificate available and CC plans referenced; commitments understood and flowed to procurement
- [ ] CC issued by the certifier (LSL Receipt is a CC prerequisite — verify upstream)
- [ ] Construction management plan, waste management plan, traffic management plan drafted and filed under `07-construction/04-management-plans/`
- [ ] Site possession date confirmed and notified to owner; insurance period aligned
- [ ] Subcontractor register opened; head contract programme baselined under `06-programme/`
- [ ] Owner briefed on payment mechanism, variation mechanism, and communication cadence

The full checklist (across all four user roles) is in `contract-setup-system.md`. This list is the builder-specific subset.

## Failure modes specific to the builder role

- **Missing HBCF.** Builder takes deposit or signs contract without per-project HBCF certificate. Statutory breach. The deposit cap exists precisely to surface this — many builders learn it the hard way.
- **CC issued without LSL paid.** Sometimes the certifier issues despite LSL not paid (e.g. on a credit arrangement); construction starts; the builder discovers the levy is unpaid mid-project. Block this by making LSL Receipt a mandatory mobilisation evidence item.
- **CWI sum insured set to contract sum only.** Excludes owner-supplied items, debris removal, and professional fees. Under-insurance discovered at claim time.
- **Owner-supplied items not insured by the builder's CWI.** Some policies exclude owner-supplied items; some include them. Read the policy; do not assume.
- **Stage claim before stage achievement.** HBCF claim later reveals the builder claimed completion before the stage was achieved — the worst-case builder failure.
- **Verbal variations.** Site supervisor agrees a change with the owner; work proceeds; no variation form. Recoverable only by argument.
- **Late EOT notice.** Time bar engaged. Even valid delay events become unrecoverable.
- **BASIX commitments not flowed to procurement.** Window specification, HW system, PV size — substitute product purchased without re-certification. Surfaces at the BASIX final inspection and blocks OC.
- **CC plans superseded mid-construction.** Revision issued but not flowed to subcontractors. Built to the wrong revision. Detected at inspection.
- **DLP exposure underestimated.** Statutory warranty in NSW is 6 years (major defects) and 2 years (minor) under the Home Building Act, independent of the DLP in the contract. The contractual DLP and the statutory warranty are not the same period and not coterminous.

## Agent behaviour under this overlay

When `user_role: builder` is declared:

1. The agent loads this overlay first on any phase-gate task (per `seed-targeted-read` skill).
2. The agent reads the project's evidence under the §1 authority stack — head contract, executed variations, HBCF certificate, LSL receipt, CWI certificate, BASIX certificate, CC plans.
3. The agent applies §evidence-discipline labelling: Fact (from executed documents), Assumption (gap-filled), Judgement (builder interpretation), Recommendation (proposed action).
4. The agent uses contractual register for `07-construction/05-progress-claims/`, `06-variations/`, `07-programme-eot/`, `08-rfi-notices/`, and stakeholder register for owner-facing summaries — folder-driven per §6.
5. The agent records every deliverable's `seed_consulted:` list including this overlay and the archetype seed.
6. The agent surfaces every §escalation-triggers signal and routes per the table above. It does not suppress to keep the user moving.

## See also

- `../00-doctrine/doctrine.md` — abstract project lead doctrine (this overlay is additive)
- `../00-doctrine/doctrine.md §seed-consultation-discipline` — why this overlay loads
- `../00-doctrine/doctrine.md §register-discipline`, `§decision-discipline`, `§evidence-discipline`, `§escalation-triggers`, `§voice-and-style`, `§owner-communication`
- `../AGENTS.md §1` (authority stack), `§2` (declaration gate), `§6` (voice register), `§9` (skill invocation)
- `new-dwelling-guide.md` — Tier 2 archetype seed, loaded alongside this overlay for `archetype: new-dwelling`
- `setup-and-commission-guide.md` — cross-cutting commissioning workflow (builder section)
- `contract-administration-guide.md` — head contract clause coverage (HIA / MBA / NSW Fair Trading / AS / ABIC)
- `../02-skills/atomic/seed-targeted-read.md` — the gate that loads this overlay
- `../02-skills/atomic/evidence-sweep.md` — used during setup to find existing evidence
- `../02-skills/atomic/register-row-draft.md` — used for variation, claim, EOT, defects register rows
- `../02-skills/systems/contract-setup-system.md` — the end-to-end builder commissioning workflow
