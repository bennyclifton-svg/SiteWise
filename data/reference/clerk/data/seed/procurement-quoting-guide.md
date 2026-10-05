---
tier: topic
seed_type: procurement-reference
loaded_by: "task subject (procurement, tender, quote, EOI, RFT, evaluation, contractor selection)"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
topics: [procurement, tender, quoting]
summary: "Residential procurement reality: informal subbie quote comparison alongside the formal EOI-RFT-evaluation-recommendation pathway, trade scope traps, weighting discipline and probity where it matters. The mandatory procurement seed for PMP drafting."
required_by: {create-pmp: 7, consultant-procurement: 1, trade-procurement: 1}
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §decision-discipline]
agents_anchors: [§1, §2, §8]
---

# Procurement & Quoting Guide — Residential

This is the residential procurement seed for SiteWise. It is the cross-cutting topic seed loaded by `seed-targeted-read` when the task signal is procurement, tender, quote, EOI, RFT, evaluation, recommendation, or contractor selection (see `../02-skills/atomic/seed-targeted-read.md §"Step 2 — Match task subject to cross-cutting topic seeds"`).

Residential procurement is **not** commercial procurement in miniature. The conventions diverge in shape, not just scale. This seed names the residential reality:

- most residential trades are procured **informally** — verbal-then-email, often without complete documentation, with comparison done by the builder on a small sample of known subbies;
- **formal residential procurement** (EOI → RFT → evaluation matrix → recommendation) is used by owners or architect-PMs selecting a head builder, and by some builders on high-value or specialist trade packages;
- the **probity** machinery of commercial procurement applies in a subset of residential contexts (architect-PM running a tender for an owner) and is overkill in others (a builder choosing a framer they have used twice before);
- **contract form** at award is most often an HIA Lump Sum / Cost Plus / Renovation, MBA equivalent, or NSW Fair Trading prescribed contract — AS 4000 / AS 4902 / ABIC sit at the high-end edge of residential and are loaded contextually;
- **owner-side procurement** (architect-PM selecting a builder for an owner) is a fiduciary duty, not a commercial tendering exercise — the owner is the procuree, not a sophisticated commercial Principal;
- **subcontractor procurement by a builder** is a price-quality-availability trade-off, with the audience for the comparison being the builder themselves, not a Principal demanding an audit trail.

The principles below are the abstract project-lead view. Role-specific obligations (who issues the tender, who evaluates, who awards) live in the role overlay seeds (`role-builder.md`, `role-architect-pm.md`, `role-owner-builder.md`, `role-d-and-c.md`) and are loaded alongside this seed.

Authority stack reminder (per `../AGENTS.md §1`): project evidence in the active project folder beats this seed. If the project has procurement evidence — a tender pack the architect issued, a quote schedule the builder uses internally, an evaluation matrix template the owner has used before — the project evidence wins. State the assumption explicitly where evidence is silent, per `../00-doctrine/doctrine.md §evidence-discipline`.

## 1. Residential procurement reality — informal quoting vs formal tendering

Residential procurement decisions split into two distinct workflows, and the same seed serves both. The role declaration drives the workflow:

| Role | Typical procurement workflow |
|---|---|
| `user_role: builder` | **Informal subbie quoting** for most trades (frame, brick, plumbing, electrical, painting, roofing). Formal procurement only where the trade is high-value or specialist (structural steel, specialist joinery, mechanical services on multi-dwelling). The audience is the builder themselves; the deliverable is a comparison matrix the builder uses to award. |
| `user_role: owner-builder` | **Informal subbie quoting** at the trade level (the owner-builder is acting as the builder in §1 above) **plus** an informal owner-builder-vs-self decision when packages span multiple trades. No formal tender unless the owner-builder voluntarily runs one (rare). |
| `user_role: architect-pm` | **Formal head-builder selection** on behalf of the owner: EOI shortlist → RFT pack → tender period with RFI / addenda → submission receipt → evaluation matrix → recommendation to owner. This is a fiduciary process, not a commercial probity exercise. |
| `user_role: d-and-c` | **Subcontractor procurement** as builder (per row 1 above) **plus** consultant procurement on the design side. Consultant comparisons must test scope, deliverables, design responsibility matrix fit, PI evidence, design programme, and certifier submission obligations. |

The distinction is not formality versus quality. Both workflows demand evidence discipline (Fact / Assumption / Judgement / Recommendation per `../00-doctrine/doctrine.md §evidence-discipline`), both demand seed-consultation discipline at award, and both produce a register entry that survives award (subcontractor register for the builder, contract execution decision for the owner). The distinction is in audience, weight, and the procurement machinery brought to bear.

**Reference:** HIA contracting practice; NSW Fair Trading residential building work guidance; AIQS Tendering Practice (commercial reference adapted with residential graceful degradation).

**Applies to:** archetypes `new-dwelling`, `renovation`, `multi-dwelling`, `ancillary`, `small-commercial`.

**Common pitfalls:**
- Treating an informal subbie quote comparison as if it were a public-sector tender — drowns the builder in paperwork and slows the award.
- Treating a formal head-builder tender as if it were a subbie quote round — the owner gets a price-only recommendation with no risk normalisation and no contractual posture for the recommended tenderer.
- Architect-PM running a tender without distinguishing their advisory role from a Superintendent role — the architect-PM is **not** the Superintendent unless expressly appointed (per `../00-doctrine/doctrine.md §owner-communication` and `role-architect-pm.md`).

## 2. Informal subbie quote comparison — how the builder selects a trade

The informal subbie quote workflow is the residential default for most trades. The builder approaches two or three subbies — usually known, sometimes recommended — and asks for a quote. The quote arrives by email, sometimes in PDF, sometimes in the body of the email, sometimes verbal at a site visit.

The comparison is the builder's responsibility. The deliverable is a **comparison matrix** the builder uses to award. The matrix lives in `05-procurement/05-evaluation/<trade>-comparison-v<NN>.md` (per `../02-skills/atomic/markdown-draft-for-review.md`'s voice / folder table, this path defaults to **stakeholder** voice — the audience is the builder, not the owner).

**Required matrix columns (minimum):**

| Column | Purpose |
|---|---|
| Quoter (subbie name + contact) | Identification and follow-up |
| Quote reference (email date, PDF filename) | Audit trail |
| Lump sum or schedule of rates | Pricing basis — must be stated |
| Scope inclusions | What the quote covers |
| Scope exclusions | What the quote does not cover (the scope-gap risk) |
| Programme | Start availability and duration |
| Lead time | Material order time, especially for trusses / windows / joinery / tiles |
| References | Past projects, recency, reachability |
| Licence + insurance check | Builder's licence (where applicable), public liability, workers comp |
| Payment terms | Deposit, progress, retention — informally captured but matters |
| Total compared (after scope normalisation) | The apples-to-apples figure |
| Risk flags | Scope gaps, capability concerns, references not yet reached |
| Fact / Assumption / Judgement label | Per `../00-doctrine/doctrine.md §evidence-discipline` — every figure carries a label |

**Scope-gap surfacing is non-negotiable.** If quote 2 is silent on the render allowance and quote 1 includes it at $4,200, the matrix records this as a scope gap, not as a $4,200 lower price for quote 2. Silent scope imputation is a §evidence-discipline failure — it confuses the builder's award and creates a variation dispute when the work begins and the gap surfaces.

**The matrix does not auto-recommend.** A recommendation paragraph is optional, on request from the builder. The award decision is the builder's; the agent's role is to make the comparison legible.

**Reference:** HIA Schedule of Variations (for subcontractor scope discipline); NSW Fair Trading Contractor Licence database (for licence check); state workers compensation regulators.

**Common pitfalls:**
- Verbal qualifications captured nowhere — the subbie says "I assumed scaffold is shared" at the site visit, the builder forgets, the variation lands on day three.
- Apples-to-pears comparison: lump sum for quote 1 vs SOR-with-no-quantum for quote 2. The builder picks quote 2 on headline rate; the package lands $8,000 over.
- Licence currency not checked — the subbie has a lapsed Qualified Supervisor certificate, the builder finds out at first inspection.
- Reference check skipped because the builder has worked with the subbie before — but it has been 18 months and the subbie's circumstances have changed.

## 3. Schedule of rates approach in residential

A schedule of rates (SOR) is the right pricing basis when scope is fluid. In residential it commonly suits:

- **Renovations** where the dilapidation report or first wall-opening surfaces unforeseen scope. The trade quotes a rate per square metre or per linear metre; the builder applies it to actual measurements.
- **Tight-block urban infill** where access and sequencing are unknown until the site is mobilised. The trade quotes a day rate or an hourly rate with a minimum call-out.
- **Specialist or unusual work** where lump sum pricing forces the trade to load the price against unknown unknowns. SOR keeps risk with the party best placed to price it as it crystallises.
- **Repeat trades on multi-dwelling** where the same scope is replicated across units. Per-unit rates with adjustment for left / right hand and party-wall variants.

A schedule of rates is the **wrong** pricing basis when scope is well documented and stable — a project home on a benign site does not need SOR for the frame trade. Lump sum gives the builder certainty; SOR introduces measurement-and-payment overhead with no offsetting benefit.

**A SOR quote must include:**
- the rate (per m, per m², per item, per hour, per day);
- the **basis** of the rate (what is included in the rate, what is extra);
- the **applicability** (which scope items the rate covers and which are out of scope);
- the **measurement method** (how the actual quantity will be agreed — typically the builder measures, the trade verifies);
- **markup and adjustment** rules (variation in rate for left/right hand, after-hours, weekend, deep excavation, etc.);
- a **provisional sum** total based on indicative quantities, so the builder can compare to lump-sum quotes from competitors.

The comparison matrix in §2 must surface the pricing basis explicitly: lump sum, SOR, or hybrid. A row showing only a per-m² rate against a row showing a lump sum is not a comparison — it is a presentation that hides the risk allocation difference.

**Common pitfalls:**
- SOR adopted because the scope is unclear, but no measurement discipline is set up at award. The builder and the trade argue over actual quantities at claim time.
- SOR rates not adjusted for the project's reality — a generic per-m² rate that does not account for the actual height of the wall, the access conditions, or the specification grade.
- SOR provisional sum total treated as a fixed price by the owner — the builder's "estimate at signing" becomes a contractual benchmark the trade hits at 110%, and the variation discussion is fraught.

## 4. Residential-typical trade scopes and scope traps

A residential build is the sum of ~20 trade packages. Each package has a typical scope and a set of recurring scope traps. The comparison matrix in §2 is only as good as the scope baseline against which quotes are compared.

**Typical trade packages and recurring scope traps (NSW Class 1a new build):**

| Trade | Typical scope | Recurring scope trap |
|---|---|---|
| Excavation / earthworks | Site cut, footing trench, spoil removal, rock allowance (provisional) | Rock allowance basis (m³ vs allowance dollar); spoil destination (often $5–10/t variable) |
| Concrete (slab) | Formwork, reo, pump, supply, place, finish, cure | Slab class adjustment (M to H1 doubles reo); pump access on tight blocks |
| Frame (timber) | Wall frames, roof trusses, internal partitions, bracing per AS 1684 / AS 4055 wind | Truss design as PC or trade; balcony / cantilever specialty framing; window opening setout responsibility |
| Roof (tile or Colorbond) | Battens, sarking, cladding, flashings, gutter, downpipes, fascia | Sarking and insulation overlap (BASIX commitment); fascia and eave lining boundary with cladding trade |
| Brick / external cladding | Supply and lay brick, render, weatherboard, EPS, FC; flashings; control joints | Owner-supplied bricks (PC or owner-supplied?); render system specification often unclear |
| Windows / external doors | Supply, install, flash, glaze; security screens optional | BASIX U-value and SHGC compliance evidence; lead time (often 8–12 weeks); colour match with cladding |
| Internal linings (plaster) | Supply, fix, set, sand; cornices; ceiling | Wet-area moisture-resistant grade; back-blocking for ceiling sheets; bulkhead detail |
| Floor coverings | Supply and lay tile / timber / carpet / vinyl per PC | PC vs owner-supplied (see `cost-management-principles.md §3` and §4); waste rate allowance |
| Wet areas (tiling) | Waterproofing, screed, tile, grout, silicone, accessories | Waterproofing inspection sign-off (AS 3740); shower screen interface; tapware install ownership |
| Kitchen joinery | Carcass, doors, drawers, benchtop, splashback, sink, tapware (PC) | Benchtop material (stone / engineered stone / laminate); appliance integration; soft-close hardware grade |
| Joinery (non-kitchen) | Robes, vanities, laundry, study | Owner-supplied handles (PC or supplied); shadow-line vs scribe; mirror integration |
| Electrical | Switchboard, sub-circuits, lighting (PC for fittings), data, security, PV | BASIX PV size; lighting fittings PC vs owner-supplied; data and AV consultant interface |
| Plumbing / gas | Sanitary, HW system (BASIX), gas service, rainwater tank if required | HW system BASIX commitment substitution; tapware install ownership; rainwater tank concrete pad |
| HVAC / mechanical | Split / ducted; bathroom and laundry exhaust per NCC | Bulkhead allowance; NCC ventilation rate vs comfort; condensate drain run |
| External works | Driveway, crossover, paths, fencing, retaining, landscape | Crossover council approval and cost; retaining-wall structural sign-off; landscape often owner direct |
| Painting | Internal and external paint, stains | Substrate prep (especially weatherboard); colour match samples; touch-up at PC |

**The scope baseline.** A comparison matrix needs an agreed scope before it can compare. The builder should write the scope baseline once per package (drawings reference + specification clauses + inclusions / exclusions / owner-supplied list) and issue it to all quoters. Without a baseline, every quote is responding to a slightly different scope and the comparison is theatre.

**Common pitfalls:**
- The scope baseline lives in the builder's head, not on paper. Each subbie quotes a slightly different scope. The comparison matrix shows a $4,000 price spread that is in fact a scope spread.
- Specialist trade scope assumed to be self-evident (e.g. "structural steel" without a structural drawing reference) — the trade later argues the connection design was assumed to be by the structural engineer, not in their scope.
- Inter-trade boundary not nominated (e.g. who supplies and installs the bath waste — plumber or joiner). Variation lands at install.

## 5. Formal procurement pathway — EOI → RFT → tender period → evaluation → recommendation

Formal residential procurement is the workflow for an architect-PM selecting a head builder on behalf of an owner, an owner-builder running a formal tender on a large package (uncommon), or a builder running a formal tender on a high-value or specialist subcontract.

These five stages are procurement-process stages, not the project's design / delivery stage regime. Where consultant procurement is being prepared, file it under `02-consultant/`, require one selected consultant discipline before drafting an RFP, save each RFP as `consultant_procurement_<discipline>_vNN.draft.md` (for example `consultant_procurement_structural_engineer_v01.draft.md`), and map that discipline's RFP scope and fee stages to the PMP programme / staging regime. If PMP staging is missing or vague, use the baseline project stages from the setup and programme guidance as an explicit Assumption and recommend a PMP update.

The five stages, each producing a discrete artefact:

### 5.1 EOI (Expression of Interest)

Used to shortlist 3 to 5 builders from a longer field before tender. Not always used in residential — sometimes the architect-PM and owner agree a shortlist directly from known relationships. Where used:

- the EOI is a short pack (3–10 pages) describing the project, the indicative scope, the indicative programme, and the EOI response format;
- the EOI response is a qualifications-only return — capability, capacity, references, financial standing, builder's licence, HOW capacity. **No pricing in the EOI.**
- the evaluation is qualitative — shortlist to 3 (typical residential), 4 (cautious), or 5 (where the field is genuinely competitive and the architect-PM is happy to run a larger RFT).

EOI output: `05-procurement/01-eoi/eoi-pack-v<NN>.md` (contractual). EOI evaluation: `05-procurement/05-evaluation/eoi-evaluation-v<NN>.md` (contractual). Shortlist decision: recorded in the decision register per `../00-doctrine/doctrine.md §decision-discipline`.

### 5.2 RFT (Request for Tender)

The RFT pack is the full tender invitation issued to the shortlisted builders. A residential RFT pack typically contains:

- **Conditions of Tender** — closing time, lodgement format, evaluation criteria summary, validity period (typically 60–90 days), RFI protocol, addenda mechanism, evaluation rights;
- **Form of Tender** — the binding declaration the builder signs to lodge the tender;
- **Draft Conditions of Contract** — the proposed head contract form (HIA Lump Sum / Renovation / Cost Plus / MBA equivalent / NSW Fair Trading prescribed / AS 4000 or AS 4902 for high-end) with the Particulars filled in. Special conditions are flagged for builder response;
- **Scope of Works** — written scope, cross-referenced to drawings and specification;
- **Drawings** — the IFC or near-IFC set (architectural, structural, hydraulic, electrical schematic where designed);
- **Specification** — written technical specification, finishes schedule, BASIX schedule;
- **Programme** — indicative programme with handover date and key milestones;
- **Schedule of Rates for Variations** — the rates the builder will use for future variations (margins on labour, materials, subcontract);
- **Returnable Schedules** — the forms the builder must complete (Form of Tender, Trade Breakdown, Proposed Key Personnel, Proposed Programme, Schedule of Departures, Insurance evidence, HOW capacity evidence);
- **Information documents** — geotechnical, dilapidation, BAL, BASIX certificate, planning approvals; provided for information.

RFT output: `05-procurement/02-tender-pack/rft-pack-v<NN>.md` (contractual). The pack is the entry point to the formal tender period.

### 5.3 Tender period — RFI and addenda

The tender period is typically **3 to 6 weeks** for residential, depending on documentation completeness and shortlist size. During the period:

- **RFIs (Requests for Information)** from tenderers go to the architect-PM (or owner-builder running the tender). Responses must be **anonymised** and **circulated to all tenderers**. Probity discipline — a single tenderer receiving preferential information taints the entire tender.
- **Addenda** are formal changes to the tender documents. Numbered (`Addendum 01`, `Addendum 02`, etc.), dated, circulated to all tenderers, formally acknowledged by tenderers in their returnable schedules.
- **Site briefings** — sometimes mandatory for renovations and tight-block sites. Notes circulated to all attendees.

RFI register: `05-procurement/03-rfi-addendum/rfi-register-v<NN>.md` (contractual). Addenda: `05-procurement/03-rfi-addendum/addendum-<NN>.md` (contractual).

### 5.4 Evaluation matrix

The evaluation matrix is the structured comparison that defends the recommendation. For a residential head-builder tender, typical weighting:

| Criterion | Weight (typical) | Notes |
|---|---|---|
| Price (lump sum, after normalisation) | 50–60% | Normalised for departures, exclusions, qualifications |
| Programme and methodology | 15–20% | Realism of dates, staging logic, BASIX delivery method |
| Capability and relevant experience | 15–20% | Past similar residential projects, financial stability, HOW capacity, references |
| Contract departures | 5–10% | Financial and risk impact of proposed special conditions |
| Schedule of rates for variations | 5% | Reasonableness of variation rates and margins |

**Weights must be specified per project, not assumed.** A renovation with high latent-condition risk should weight programme and methodology higher than a new dwelling on a benign site. The matrix must declare the weights up front and apply them consistently.

**Required matrix columns (residential head-builder tender):**

| Column | Source |
|---|---|
| Tenderer | Returnable schedules |
| Lump sum (ex GST and inc GST shown separately) | Form of Tender |
| Departures and qualifications | Schedule of Departures |
| Scope normalisation adjustments | Architect-PM / QS judgement (labelled Judgement) |
| Normalised lump sum (apples-to-apples) | Calculated |
| Proposed programme | Returnable schedule |
| Proposed methodology | Returnable schedule |
| References (called and rated) | Reference check log |
| Financial standing | Builder ASIC search + financial returns |
| Capacity (current pipeline, key personnel committed) | Returnable schedule + reference check |
| Builder licence + class + currency | NSW Fair Trading record |
| HOW / HBCF eligibility + capacity remaining | Builder declaration + iCare confirmation if material |
| Contract works + PL + workers comp Certificate of Currency | Insurance evidence pack |
| BASIX / energy compliance method declared | Returnable schedule |
| Weighted score per criterion | Calculated |
| Total weighted score | Calculated |
| Commentary | Architect-PM / evaluator narrative |
| Risk flags | Explicit |
| Fact / Assumption / Judgement label per row | Per `../00-doctrine/doctrine.md §evidence-discipline` |

Evaluation matrix output: `05-procurement/05-evaluation/evaluation-matrix-v<NN>.md` (contractual). Sometimes split into `price_evaluation_v<NN>.md` and `non_price_evaluation_v<NN>.md` per the commercial parent — both options are valid; the split is preferred where the price evaluation is being kept blind to the non-price evaluation until both are settled (a probity practice that residential can borrow).

### 5.5 Recommendation to owner

The recommendation is the owner-facing artefact. It lives in `05-procurement/06-recommendation/recommendation-to-owner-v<NN>.md` and defaults to **stakeholder** voice per `../02-skills/atomic/markdown-draft-for-review.md`'s table — the owner is a non-technical residential principal, not a sophisticated commercial Principal.

The recommendation follows `../00-doctrine/doctrine.md §owner-communication`:

1. **What this means for you** — the recommended tenderer and the headline reasoning, in two or three sentences.
2. **What we need from you** — the owner's decision and sign-off, with a clear due date.
3. **What's happened** — the procurement process followed, the tenderers invited and received, the evaluation outcome.
4. **What's next** — the steps from owner sign-off to contract execution.
5. **Background detail** — the evaluation matrix, the departures normalised, the residual risks, the conditions of award.

A single recommended tenderer is named. Three options without a recommendation is a doctrine failure (per `../00-doctrine/doctrine.md §owner-communication`) — the owner is not paid to choose between three; the project lead is paid to recommend.

A parallel **contractual** recommendation report is sometimes prepared for the architect-PM's file: `05-procurement/06-recommendation/tender_recommendation_v<NN>.md` (contractual voice). This is the formal advice record. The owner-facing stakeholder summary is the artefact the owner reads.

**Reference:** AS 4120-1994 Code of Tendering (adapted); HIA contracting practice; AIQS Tendering Practice (commercial reference with residential graceful degradation).

**Common pitfalls:**
- EOI criteria written so narrowly only one or two builders qualify — competitive tension is lost before tender.
- Shortlist of seven for residential — top-tier builders decline due to low statistical chance of winning vs. the cost of tendering.
- RFI answered verbally — probity breach, taints the tender.
- Addendum issued two days before close without an extension — tenderers price worst-case contingency.
- Evaluation matrix weights set after submissions received — reverse-engineering to favour a preferred tenderer is the classic probity breach.
- Recommendation presented as "three roughly equal options" — the owner cannot decide; the architect-PM has not done their job.

## 6. Tender pack content — the RFT in detail

A residential RFT pack is smaller than a commercial RFT. The components are:

1. **Conditions of Tender** (1–2 pages). Closing date and time, lodgement method (email / portal), validity period (60–90 days typical), evaluation criteria summary, RFI protocol, addenda mechanism, evaluator's right to reject any tender, no-collusion declaration. Probity language scaled to the procurement context.
2. **Form of Tender** (1 page). The legally binding offer the builder signs. Includes the lump sum (or SOR / GMP basis), the contract term offered, and the validity period.
3. **Draft Conditions of Contract**. The proposed contract form with Particulars pre-filled by the architect-PM / owner side. Special Conditions clearly marked. Where the form is HIA, the stationery and current edition are stated; where MBA or NSW Fair Trading prescribed, those forms are stated; where AS 4000 / AS 4902 for high-end residential, the standard form is cited.
4. **Scope of Works**. Written prose plus drawing references plus specification cross-references. The scope of works must align with the architectural drawings and the specification — divergence is the most common source of tender disputes.
5. **Drawings**. Architectural set (plans, elevations, sections, schedules), structural set, hydraulic schematic, electrical schematic (often light), landscape. For BASIX-driven specifications, the BASIX Certificate is annexed.
6. **Specification**. The technical specification (NATSPEC / Masterspec / project-specific). Finishes schedule with PC items identified and allowances stated.
7. **Programme**. Indicative programme — the builder's tendered programme will refine it.
8. **Returnable schedules**:
   - Trade Breakdown — the lump sum split by trade element. Used later to assess progress claims.
   - Schedule of Rates for Variations — labour, materials, subcontract mark-up, percentages on prime cost.
   - Proposed Key Personnel — site supervisor, contract administrator, foreperson.
   - Proposed Programme — Gantt chart with critical path, milestones, BASIX inspection holds.
   - Schedule of Departures — the **only** place the tenderer is allowed to nominate deviations.
   - Insurance evidence — Certificate of Currency for CWI, PL, workers comp.
   - HOW / HBCF capacity evidence — builder's per-project capacity declaration.
   - Reference list with contactable referees.
9. **Information documents**. Geotechnical, dilapidation, BAL, BASIX Certificate, planning consent and conditions of consent (for DA pathway), CC plans, BPA (if relevant). Marked "for information only — not part of the contract".

**Reference:** AS 4120-1994 (Code of Tendering); HIA contracting practice; NATSPEC structures.

**Common pitfalls:**
- Special Conditions left as "to be negotiated" — tenderers price worst-case.
- Drawings stamped "Not For Construction" included in a Construct-Only RFT — the lump sum may be legally unenforceable.
- Critical scope items only in the specification, not on the drawings — contractor omissions at install.
- Trade breakdown not requested — no baseline for progress claim assessment downstream.

## 7. Evaluation criteria with weighting discipline

Evaluation criteria must be:

1. **Stated up front** in the Conditions of Tender. Tenderers should know what they are being judged on before they submit.
2. **Weighted explicitly** per project. A renovation weights programme realism higher than a project-home new build does. The weights must be agreed by the architect-PM and the owner before tenders close.
3. **Applied consistently** across all tenders. No retrofitting weights to suit a preferred outcome.
4. **Documented** in the evaluation matrix and the recommendation report. The audit trail must survive — if the matter goes to dispute (or to the owner's lender for finance approval), the weighting and reasoning must be reproducible.

**Residential-typical weighting (architect-PM head-builder tender):**

- Price (normalised lump sum, ex GST): 50–60%
- Programme and methodology: 15–20%
- Capability, references, capacity: 15–20%
- Contract departures (financial and risk impact): 5–10%
- Schedule of rates for variations: 5%

**Residential adjustments to commercial conventions:**

- **References weighted higher** than typical commercial — residential owners care about the lived experience of working with the builder; one bad reference for owner communication can outweigh a 5% price advantage.
- **Departures weighted lower** than typical commercial — residential standard forms (HIA / MBA) are well-known and departures are usually minor.
- **Owner-side intangibles** — owner comfort, perceived rapport, communication style — are real factors in residential and should be acknowledged in the architect-PM's commentary even if not formally weighted. They belong in the recommendation narrative, not in the matrix score.

**Reference:** AS 4120-1994 (adapted); AIQS Tendering Practice; common-sense residential adjustments.

**Common pitfalls:**
- Single-weighting (price only) — the owner gets the lowest price and the longest list of variations.
- Weights set after submissions are received — probity breach, no defensible audit trail.
- Owner intangibles smuggled into matrix scores without disclosure — looks like a numbers-driven decision but is not.

## 8. Recommendation discipline

A recommendation is more than a winner. It carries the architect-PM's (or owner-builder's or builder's) reasoning, the conditions on award, and the residual risks the principal is asked to accept.

**A residential head-builder recommendation must state:**

1. **The recommended tenderer** — one name. Not three options.
2. **The reasoning** — why this tenderer over the others. Specific. Tied to the evaluation criteria and weights.
3. **The contractual posture** — which contract form, which Special Conditions accepted, which departures negotiated out, which Particulars to be set.
4. **Conditions of award** — what the tenderer must do or evidence before contract execution. Typical: builder licence currency confirmed, HOW per-project certificate issued, CWI bound naming owner as joint insured, LSL paid before CC, BASIX compliance method confirmed.
5. **Residual risks** — the risks the owner is taking on at award. Specific. Examples: thin contingency declared by the tenderer, tight programme that depends on no rain in May, owner-supplied items dependent on overseas shipping.
6. **Cost of acceptance vs decline** — implicit in the recommendation. If the owner declines the recommendation, the next-ranked tenderer is named (typically not the lowest price; the discipline of normalisation has already exposed why).
7. **Next steps** — owner sign-off, contract execution, mobilisation, site possession date.

**Voice register split.** The owner-facing recommendation at `05-procurement/06-recommendation/recommendation-to-owner-v<NN>.md` is **stakeholder**. The internal contractual record at `05-procurement/06-recommendation/tender_recommendation_v<NN>.md` (if produced) is **contractual**. Both can exist; the owner reads one and the file holds the other.

**Reference:** AIQS Tendering Practice (recommendation report structure); HIA contracting practice (post-recommendation contract execution); `../00-doctrine/doctrine.md §owner-communication`.

**Common pitfalls:**
- Recommendation presented as three options with no recommendation — owner cannot decide; the project lead has not done their job (per `../00-doctrine/doctrine.md §owner-communication`).
- Conditions of award buried in attachments — the owner signs without seeing them and the builder mobilises before the conditions are met.
- Residual risks omitted to "keep the recommendation positive" — surfacing risk is the project lead's duty, not a marketing question.
- Cost of acceptance vs decline implicit — the owner does not understand what changes if they say no.

## 9. Probity and fairness — when it matters

Probity is the discipline of conducting procurement fairly. In residential, it matters in some contexts and is overkill in others.

**Probity matters when:**

- An architect-PM is running a tender for an owner. The architect-PM has a fiduciary duty. Fair treatment of tenderers is part of that duty — but the higher duty is to the owner, and the architect-PM is not subject to the same probity rules as a public-sector procurement officer.
- An owner-builder is running a formal tender for a head builder or large package. Fair treatment of tenderers is good practice; it also builds market reputation.
- A builder is running a formal tender for a high-value or specialist subcontract. Fair treatment helps secure a competitive market for future projects.

**Probity is overkill when:**

- A builder is choosing a framer they have used twice before. The quote round is informal, the comparison is internal, and the audience is the builder themselves. Probity machinery (anonymised RFIs, sealed price submissions, scoring blinded from price) is theatre.
- An owner-builder is choosing a subbie on recommendation from a friend. The quote round is informal, the relationship is personal, and probity discipline would alienate the trade.

**Probity discipline (where applied):**

- **Fair treatment** — all tenderers receive the same information, the same access, the same timeframes.
- **Anonymised RFIs** — questions and responses circulated without naming the asker.
- **Addenda for all** — formal changes go to all tenderers, with acknowledgement required in returnable schedules.
- **No price disclosure** — no tenderer's price is shared with another, ever.
- **Evaluation criteria stated up front and applied consistently** — no retrofitting.
- **Sealed price submission** — the price envelope is opened only after non-price evaluation is settled, to avoid price-driven bias in qualitative scoring (a useful discipline; not always practicable in residential).
- **Conflict of interest declared** — the evaluator declares any prior relationship with tenderers.
- **Records kept** — the procurement file survives the award and supports any future dispute or audit.

**Where the architect-PM is also recommending one of their preferred / repeat builders**, the conflict of interest must be disclosed to the owner before tender invitation, not after recommendation. This is a §evidence-discipline and §escalation-triggers obligation.

**Reference:** AS 4120-1994; Government Procurement Guidelines (adapted to residential reality); `../00-doctrine/doctrine.md §evidence-discipline`, `§escalation-triggers`.

**Common pitfalls:**
- Probity machinery applied to a subbie quote round — theatre, wastes everyone's time.
- Probity discipline skipped on a formal head-builder tender — owner cannot defend the decision if challenged or audited.
- Architect-PM's prior relationship with a tenderer undisclosed — surfaces at award and damages owner trust.

## 10. Role-divergent escalation routing

Procurement escalations route per role (per `../00-doctrine/doctrine.md §escalation-triggers`):

| Role | Escalation route |
|---|---|
| `owner-builder` | Self-flag log in `08-meetings-reporting/`. Procurement issues (subbie no-show, quote rejection beyond budget, scope gap discovered post-award) park as decision items for the owner-builder to action. |
| `architect-pm` | Owner-facing summary in `08-meetings-reporting/owner-update*` using `§owner-communication` format. Procurement issues affecting tender award, contract execution, or owner spend land on the owner's desk via stakeholder-voice summary. |
| `builder` | Owner via contractual notice for any procurement issue affecting head-contract scope or programme (route via `07-construction/08-rfi-notices/` for the written record). Subbie procurement issues are internal — recorded in the subcontractor register, not escalated outside the builder's own management. |
| `d-and-c` | As builder for head-contract scope; plus design-side procurement (consultant engagement, novation, PI evidence, DRM fit, design deliverables) routes to the certifier where compliance is affected and to the owner where the PPR, cost, or programme baseline changes. |

**Procurement-specific escalation triggers:**

- **Missing builder licence or insurance evidence** during head-builder tender — high priority; cannot award without it.
- **Low-bid quality risk** — the lowest tender is materially below the next, with no plausible scope explanation. Risk of contractor distress mid-project; escalate before recommendation.
- **HOW / HBCF capacity exhausted** — the tenderer's per-project capacity for HOW is already committed to other projects. Builder cannot accept additional jobs without iCare approval; this is a hard block.
- **BASIX method of compliance not declared** in tender — the tenderer has not stated how they will meet the BASIX commitments. Surfaces as a delivery problem at first BASIX inspection.
- **Contract form mismatch** — the tenderer proposes an AS 4000 contract on a residential project where HIA Lump Sum is appropriate. Either the tenderer misread the brief or the project lead has misframed the procurement strategy.
- **Owner change request mid-tender** — the owner wants to change scope while tender is open. Either issue an addendum and extend the tender period, or withdraw and re-tender. Cannot informally accept the change without notifying tenderers.
- **Tender clarification overdue** — the architect-PM has not responded to RFIs within the stated window. Probity risk; tenderers cannot price; risk of tender extension.
- **Subbie scope gap material to award** — the builder is about to award a trade on a quote that excludes a meaningful scope item. Surface before the award; do not let the gap land as a variation post-award.

**The agent must refuse to suppress an escalation trigger.** Per `../00-doctrine/doctrine.md §escalation-triggers`, if the agent sees a trigger, the project lead sees it.

## 11. State callouts (graceful degradation per AGENTS.md §8)

NSW is the deep default in this seed. Non-NSW callouts:

**VIC** — *Slice 14 will deepen this.* Residential procurement uses HIA Victoria-edition contracts widely; the prescribed state contract under the Domestic Building Contracts Act 1995 (VIC) is the Domestic Building Contract. Domestic Building Insurance (DBI) replaces NSW HBCF — administered by the VMIA. The Building Industry Fairness regime in VIC is BCISP Act 2002 (very similar to NSW SOP Act). Builder registration is via the Victorian Building Authority (VBA) — categories DB-U and DB-L. **Treat as Assumption** in any tender pack for a VIC project — confirm against current VIC instruments before relying on residential conventions that differ from NSW.

**QLD** — *Callout only; deep coverage deferred.* QBCC replaces NSW Fair Trading and HBCF (Home Warranty Insurance is mandatory under QBCC). The Security of Payment regime is the Building Industry Fairness (Security of Payment) Act 2017 (QLD), which has materially different payment claim mechanics — including monthly statutory claim windows that do not align with HIA stage-payment schedules. BASIX does not apply — QDC Part MP 4.1 / NCC Section J. Treat as Assumption.

**SA, WA, TAS, NT, ACT** — *Callout only; deep coverage deferred.* Each state has its own statutory warranty regime, builder registration, and energy compliance instrument. WA Building Services (Registration) Act 2011. TAS Building Act 2016. NT Building Practitioners Board. Treat as Assumption and flag for project lead supplementation.

Where a non-NSW state has no callout for the task at hand, the skill (calling this seed) **flags the gap** rather than silently extending NSW guidance — per `../AGENTS.md §8` and `../00-doctrine/doctrine.md §state-handling`.

## See also

- `../AGENTS.md §1` (authority stack), `§2` (declaration gate), `§3` (seed loading rules), `§5` (output discipline), `§6` (voice register — `05-procurement/` is contractual by default with stakeholder exceptions for owner-facing recommendations), `§8` (state callouts), `§9` (skill invocation), `§11` (active-project boundary)
- `../00-doctrine/doctrine.md §seed-consultation-discipline`, `§evidence-discipline`, `§register-discipline`, `§decision-discipline`, `§escalation-triggers`, `§voice-and-style`, `§owner-communication`
- `setup-and-commission-guide.md` — commissioning workflow that procurement feeds into at contract execution
- `contract-administration-guide.md` — contract clause coverage for the contract executed after recommendation
- `cost-management-principles.md` — PC sums, owner-supplied items, contingency band, variation pricing under HIA Schedule of Variations
- `new-dwelling-guide.md` (or other Tier 2 archetype seed) — archetype-shaped procurement context
- `role-builder.md` — role overlay for builder; subcontractor procurement responsibilities
- (Slice 07) `role-owner-builder.md` — role overlay for owner-builder; informal trade quoting from the owner-builder seat
- `role-architect-pm.md` — role overlay for architect-PM; formal head-builder selection on behalf of owner
- `role-d-and-c.md` — role overlay for D&C contractor; subcontractor and consultant procurement
- `../02-skills/systems/procurement-evaluation-system.md` — the system skill that orchestrates this seed (role-branched: informal subbie comparison vs formal head-builder selection)
- `../02-skills/atomic/seed-targeted-read.md` — loads this seed when procurement is the task subject
- `../02-skills/atomic/evidence-sweep.md` — surfaces procurement evidence (quotes, tender packs, prior procurement decisions)
- `../02-skills/atomic/markdown-draft-for-review.md` — voice / folder table for `05-procurement/` paths
- `../02-skills/atomic/register-row-draft.md` — subcontractor register, EOI shortlist register, RFI register
- `../../Harness/01-seed/procurement-tendering-guide.md` — commercial parent of this seed; the EOI / RFT / probity machinery originates there
