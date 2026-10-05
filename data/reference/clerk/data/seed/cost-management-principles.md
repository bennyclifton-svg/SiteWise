---
tier: topic
loaded_by: task subject (cost, budget, contingency, claim, variation pricing)
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential, commercial, industrial, institution, mixed, infrastructure]
applies_to_work_types: [new, refurb, extend, remediation, advisory]
topics: [cost, budget, contingency, claims, variations, commercial]
summary: "Residential and non-residential construction cost-management conventions: cost plans, contingencies, PC/PS allowances, GST, progress claims, variations, forecasting, value engineering, final accounts, benchmarking and risk-adjusted budgets."
required_by: {create-pmp: 5, create-cost-plan: 1}
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §voice-and-style]
seed_type: cost-reference
state_default: NSW
agents_anchors: [§1, §2, §8]
---

# Cost Management Principles — Residential

This is the residential cost-management seed for SiteWise. It is the cross-cutting topic seed loaded by `seed-targeted-read` when the task signal is cost, budget, contingency, claim, or variation pricing (see `../02-skills/atomic/seed-targeted-read.md §"Step 2 — Match task subject to cross-cutting topic seeds"`).

Residential cost management is **not** commercial cost management in miniature. The conventions differ in shape, not just scale:

- the **HIA Schedule of Allowances** is the format owners and builders both expect — not the AIQS elemental breakdown;
- **PC (prime cost) sums** for owner-selected items (kitchen, bath, tiles, appliances, taps, sanitary, lighting, joinery, flooring) are how the contract handles the not-yet-chosen specification cost;
- **owner-supplied items** are a discrete category, contractually distinct from PC items;
- **contingency** sits in the project lead's accounting in the 5–10% band (not the 3–5% commercial construction band), with the residential reality of owner change appetite priced in;
- **GST** is typically inclusive in residential cost plans (the owner is usually an owner-occupier who cannot recover GST), the reverse of commercial convention;
- the **HIA stage-payment schedule** (deposit / base / frame / lockup / fixing / completion) is the typical claim mechanism — not monthly measured progress claims;
- **variation discipline** is enforced by the HIA Schedule of Variations form rather than by a Superintendent's direction.

The principles below are the abstract project-lead view; role-specific obligations (who issues a claim, who assesses it, who pays) live in the role overlay seeds (`role-builder.md`, `role-architect-pm.md`, `role-owner-builder.md`, `role-d-and-c.md`) and are loaded alongside this seed.

Authority stack reminder (per `../AGENTS.md §1`): project evidence in the active project folder beats this seed. If the project has a cost plan or contract that uses a different schema, the project evidence wins. State the assumption explicitly where evidence is silent, per `../00-doctrine/doctrine.md §evidence-discipline`.

## 1. The residential cost plan in HIA Schedule of Allowances format

The HIA Schedule of Allowances is the residential industry's accepted format for the cost-plan equivalent in an HIA lump-sum or HIA cost-plus contract. It is the basis on which an owner and a builder agree the budget shape, and it carries through into the contract's PC and variation mechanisms. A cost plan that does not map cleanly to the Schedule of Allowances is hard for the owner to read against the contract and hard for the builder to claim against.

The Schedule of Allowances structure (typical):

1. **Preliminaries** — site set-up, hoarding, site amenities, scaffold, temporary services, site supervision, project insurances charged to the builder, builder's licence fee allocation, HBCF premium (NSW) or DBI premium (VIC) allocation, LSL or CoINVEST allocation.
2. **Excavation and earthworks** — site cut and fill, footing excavation, spoil removal, rock allowance (provisional sum where soil report flags risk).
3. **Foundations and slab** — concrete, reinforcement, formwork, vapour barrier, termite management; slab classification per AS 2870 (M / H1 / H2 / E / P) drives volume and reinforcement.
4. **Frame** — timber framing per AS 1684 or steel framing where specified; wind classification per AS 4055 drives bracing.
5. **External cladding** — brick veneer, render, weatherboard, FC sheet, EPS panel, etc.
6. **Roof** — trusses, sarking, battens, roof cladding (tile / Colorbond / membrane), gutters and downpipes.
7. **Windows and external doors** — frame material, glazing performance per BASIX (U-value, SHGC), security screens.
8. **Internal linings and partitions** — plasterboard, insulation per BASIX commitments (R-values), cornices.
9. **Floor coverings** — tile, timber, carpet, vinyl, polished concrete. Usually a PC line per coverage type.
10. **Wet areas** — waterproofing, tile, taps and sanitary (PC items), shower screens, mirrors, accessories.
11. **Kitchen** — joinery (PC item), benchtop (PC item), appliances (PC items, often owner-supplied), splashback, sink and tapware (PC items).
12. **Joinery (non-kitchen)** — robes, vanities, laundry joinery, study or built-in furniture. Typically PC items.
13. **Electrical** — switchboard, sub-circuits, lighting (PC item for fittings), data, security, PV per BASIX.
14. **Plumbing and gas** — sanitary fixtures (PC item), HW system (BASIX commitment), gas service, rainwater tank if BASIX requires.
15. **HVAC and mechanical ventilation** — split / ducted system, range-hood discharge, bathroom and laundry mechanical exhaust per NCC.
16. **External works and landscaping** — driveway and crossover, paths, fencing, retaining, landscape. Often a defined PC or excluded with owner-supplied scope.
17. **Painting and final finishes** — internal and external paint, stains and clear finishes.
18. **PC sums schedule** — itemised allowances for owner-selected components (see §3).
19. **Owner-supplied allowances** — items the owner is supplying outside the builder's scope (see §4).
20. **Contingency** — 5–10% line, declared with a basis (see §5).
21. **Builder's margin** — where the contract is HIA Cost Plus, the margin is a separate line; for HIA Lump Sum it is embedded in the trade lines and not a discrete row.
22. **Variation totals to date** — running variation total as the project progresses (cross-reference variation register, see §7).

The line list above is typical, not canonical — the HIA stationery and the executed contract's particulars together define the exact structure for a project. **The project's contract is the source of truth.** This seed lists the typical pattern so the agent can draft a recognisable cost plan when no project-specific schedule exists yet.

**Claim-first cost planning:** A project lead may be asked to certify or assess a claim before a formal cost plan workbook exists. In that case, do not treat the missing workbook as a reason to produce a thin cost plan. Use the best structured active-project evidence available: executed contract schedule, schedule of allowances, progress claim schedule, payment claim schedule of values, tender return, accepted quote, or variation roll-up.

Where a progress claim contains trade/work-package rows with contract values, those rows are construction cost evidence. Use the latest reliable claim schedule as the working construction breakdown, reconcile it to the revised contract sum, and carry variation rows separately or clearly identify them as embedded in the revised sum. If the claim schedule and variation forms conflict, keep the trade breakdown visible and record the conflict; do not collapse the construction section to one contract total merely to avoid the reconciliation problem.

For commercial projects with a quantity surveyor, an AIQS elemental cost plan (substructure / superstructure / external envelope / internal finishes / fittings / services / external works / preliminaries) may be a contractual requirement. SiteWise's `small-commercial-guide.md` archetype seed addresses this graceful degradation. For all four residential archetypes (new-dwelling, renovation, multi-dwelling, ancillary), the HIA Schedule of Allowances is the default.

**Reference:** HIA Lump Sum NSW New Homes contract (current edition); HIA Renovations; HIA Cost Plus.
**Applies to:** archetypes new-dwelling, renovation, multi-dwelling (where each dwelling is HIA-procured), ancillary.

**Common pitfalls:**
- Using an AIQS elemental breakdown on an HIA lump-sum contract — the owner cannot read the cost plan against the contract claim schedule, and PC reconciliation becomes opaque.
- Folding PC sums and owner-supplied items into trade lines rather than itemising them — the contract's PC adjustment mechanism then has nowhere to attach.
- Omitting contingency as a line — leaves the owner with no buffer and the builder no way to flag latent risk without a variation.

## 2. Total project cost vs construction cost

The cost plan must distinguish total project cost from construction cost. Owners frequently treat the construction-cost figure as the total commitment and are surprised at handover by the additional spend.

**Construction cost** — the contracted amount payable to the builder under the HIA / MBA / AS construction contract. Includes preliminaries, trades, PC sums (as allowances), owner-supplied allowances (where contracted that way), contingency held within the contract, and the builder's margin (embedded or separate).

**Total project cost** — construction cost plus:
- consultant fees (architect, structural engineer, hydraulic, BASIX assessor, certifier, surveyor, geotechnical, landscape — where engaged);
- authority charges (DA fees, CC fees, BASIX certificate fee, Sydney Water charges or state equivalent, OSD certification, BPA where applicable, long service levy);
- statutory insurance and warranty premiums (HBCF / DBI premium where the builder passes through; CWI premium where the principal places the cover; PI premium where applicable);
- utility connection charges (electricity, water, sewer, gas, NBN, telecommunications);
- owner-supplied items procured outside the contract;
- finance costs during construction (where the owner has a construction loan);
- contingency held outside the contract by the owner (typical when owner change appetite is high);
- legal fees, conveyancing where applicable, land or pre-purchase costs (project-level);
- FF&E (furniture, fittings, equipment), moving costs, and post-PC fitout.

For an architect-PM (`user_role: architect-pm`), making the total project cost visible to the owner is part of the duty of care. For a builder (`user_role: builder`), the construction-cost number is what the builder is responsible for, but the builder should be aware which owner-side commitments depend on the construction programme. For an owner-builder (`user_role: owner-builder`), there is no distinction in practice — the owner-builder is responsible for everything.

Per `../00-doctrine/doctrine.md §evidence-discipline`, every cost figure in a cost plan must be labelled **Fact** (supported by a quote, invoice, or signed contract), **Assumption** (no source yet — must be confirmed), **Judgement** (the project lead's estimate based on evidence and benchmark), or **Recommendation** (the suggested allowance for owner decision). Owners frequently misread Assumption / Judgement figures as Fact when not labelled.

## 3. PC sums (prime cost items)

A PC sum is a contractual allowance for an item the owner has not yet selected at the time of contract signing. The most common PC items in residential are:

- kitchen appliances (cooktop, oven, dishwasher, rangehood, fridge cavity — note appliances are often owner-supplied rather than PC; see §4);
- sanitary ware (toilets, basins, baths);
- tapware (kitchen tapware, basin and bath taps, shower mixers);
- tiles (floor, wall, splashback — typically priced by area + supply rate);
- floor coverings (timber, carpet, vinyl);
- lighting (fittings only, not the wiring — wiring is in the electrical trade);
- joinery (kitchen joinery, robes, vanities, custom built-ins);
- ironmongery / door furniture;
- letterbox, house number, mailbox where specified;
- door and window furniture upgrades.

**The mechanism under HIA contracts:**

1. The contract Particulars or annexure lists each PC item with an **allowance** (a $ figure the builder has priced into the contract sum, typically supply-only, sometimes supply-and-install).
2. The owner selects the actual item. The owner has a contractual obligation to select within a programme window — failure to select can be a variation reason for time.
3. When the actual cost is known (typically by quote from the builder's nominated supplier or by an owner-procured invoice), the PC is **reconciled**: actual cost vs allowance.
4. The difference is a contractual adjustment: actual > allowance flows up as additional cost to the owner (with the builder's margin applied per the contract — typically the contract specifies whether margin applies to the increment or to the actual cost in full); actual < allowance flows down as a credit (margin treatment varies — read the contract).
5. The reconciliation is recorded in a **PC reconciliation register** (typically the variation register's adjacent columns or a discrete PC schedule). The reconciliation closes at PC.

**Common pitfalls:**
- The owner treats the PC allowance as the budget for the item rather than a placeholder. When the owner then chooses an $8,000 cooktop against a $1,500 PC, the additional $6,500 + margin lands as a contractual obligation, not a surprise — but it is invariably presented as a surprise.
- The builder does not reconcile PC items as selections are made — the budget shifts under the cost plan without a corresponding variation register entry, and the cost plan version goes stale.
- "Supply only" vs "supply and install" is not clearly stated in the PC line. The installation cost ambiguity becomes a dispute at fixing stage.
- PC items selected from an owner-supplied source (the owner brings their own tapware): this should usually be reclassified as **owner-supplied** (see §4), not retained as a PC. The two mechanisms are different — running a PC against an owner-supplied actual confuses everyone.

**Labelling discipline:** every PC line in the cost plan carries a status: `allowance` (not yet selected), `selection-pending-quote`, `selected-priced` (the actual is known and reconciliation is calculable), `reconciled` (the variation register entry is open), or `closed` (the contract adjustment has been signed).

**Reference:** HIA Lump Sum / HIA Cost Plus / HIA Renovations contract PC clauses; the contract's executed Schedule of Allowances or PC schedule is the source of truth for a given project.

## 4. Owner-supplied items and allowances

Owner-supplied items are distinct from PC sums. An **owner-supplied** item is one the owner procures and supplies directly — the builder does not include it in the contract sum and may charge a separate handling or installation fee where the contract permits. PC items by contrast are included in the contract sum (as allowances) and are procured by the builder (or by the owner with the builder's involvement, depending on the contract).

The two mechanisms must not be conflated:

| Aspect | PC sum | Owner-supplied |
|---|---|---|
| In contract sum? | Yes (as an allowance) | No (excluded) |
| Procurement | Builder (often) or owner via builder | Owner directly |
| Reconciliation | Actual vs allowance, with margin treatment per contract | Not applicable — no contract sum impact, but installation cost may be a separate item |
| Risk of cost overrun | Owner bears the increment over allowance (plus margin) | Owner bears entire cost outside contract |
| Programme risk | Builder selects within programme; delay on selection is owner's risk | Owner must deliver on programme; late delivery is owner's risk |
| Warranty | Builder warrants installation; supplier warrants the item | Owner warrants the item; builder warrants installation only |
| Insurance | Covered under CWI (builder-procured) | Owner must confirm CWI extension or arrange own cover |
| Defects | Builder's responsibility for installation; supplier's for product | Owner's responsibility for product defects |

**Common owner-supplied scope in residential:**
- whitegoods (fridge, freezer, washing machine, dryer) — usually owner-supplied;
- light fittings (where the owner has bought from a specialist supplier);
- joinery hardware (handles, pulls);
- furniture being built in (e.g. a vintage cabinet inserted into a niche);
- decorative tiles, splashback materials sourced overseas;
- pizza ovens, fireplaces, BBQs — items at the edge of "building" vs "furniture".

**Owner-supplied allowances** appear as a separate schedule in the cost plan, **below** the contract sum total. The schedule lists each item with:
- item description;
- supplier (where committed) or status `to-source`;
- expected delivery date (programme-critical for fixing stage);
- expected cost (for the owner's total-project-cost view);
- installation cost (in contract if builder is installing; separate if owner is sourcing the trade).

The agent should propose an **owner-supplied items register** (per `register-row-draft.md` standard register `Owner-supplied items register`, ID convention `OS-<seq>`) at commissioning. The register tracks status: `committed`, `ordered`, `delivered`, `installed`, `late`. Late delivery is one of the most common programme risks in residential — flag it under `../00-doctrine/doctrine.md §escalation-triggers` when delivery dates threaten the fixing-stage programme.

**Common pitfalls:**
- The owner agrees to supply an item, then expects the builder to chase it through procurement. The role boundary is unclear and the item is late.
- Insurance gap: the owner-supplied item is stored on site before installation; CWI does not extend to owner-supplied goods unless explicitly endorsed. Theft or damage falls on the owner without notice.
- Owner-supplied items not in the register at all — they live in email threads and surface as scope arguments at PC.

## 5. Contingency — the 5–10% band

Residential contingency sits in a different band from commercial construction contingency. Where commercial new-build construction contingency is typically 3–5% of contract sum at construction phase, residential contingency at the same stage is typically 5–10%.

The reasons residential contingency is higher:

- **owner change appetite** is intrinsic to residential — the owner is making personal-taste decisions throughout construction, and selections shift in ways that are not material in commercial fit-out;
- **PC reconciliation** drift — the gap between PC allowances and actual selections almost always lands above allowance (owners select up, not down);
- **owner-supplied items** late delivery cascades into builder standing time charges;
- **scope creep** through informal site-meeting agreement is harder to discipline in residential than in commercial — the architect (if present) is usually not the Superintendent, and the owner is on site weekly;
- **latent conditions** in renovations — the wall opens up and reveals something not in the dilapidation report;
- **BASIX commitment chase** — late discovery that a glazing or HW system upgrade is required to meet certificate commitments.

**Band guidance (cross-reference issue-03 brief's acceptance criterion of 5–10%):**

| Project profile | Contingency | Basis |
|---|---|---|
| Well-documented new build, owner-low-change-appetite, no latent risk (e.g. a project home on a benign site) | 5% | Tight specification, repeatable build pattern, low owner-side variation pressure. |
| Custom new build, NSW Class 1a, well-documented, M-class site | 5–7% | Custom always carries more PC reconciliation risk than project. |
| Custom new build with H1 / H2 / E / P slab class, or BAL ≥ 19, or significant cut/fill | 7–10% | Site risk priced in. |
| Renovation with full dilapidation report and structural intervention scope | 8–10% | Latent conditions live here. |
| Renovation, urban infill, neighbour-property dilapidation risk, tight site access | 10% | Tight-site and neighbour management overhead is material. |
| Multi-dwelling residential (Class 2 or repeated Class 1a) | 6–9% | Repeat efficiency offsets some risk, party-wall and metering coordination adds back. |
| Ancillary (granny flat, CDC pathway) on existing residential lot | 5–7% | Constrained scope, well-trodden pathway. |
| Heritage or character residential intervention | 10%+ | Latent conditions and authority-condition rework dominate. Treat as a renovation+ band. |

The contingency line must be declared with its band and basis in the cost plan. Reciting "10%" without basis is a §evidence-discipline failure: contingency is a **Judgement**, and the basis must be stated.

For `archetype: multi-dwelling`, the cost plan must make the shared-vs-per-dwelling split visible. At minimum, test whether classification, party-wall / fire and acoustic treatment, separate metering and service upgrades, authority / infrastructure contributions, shared civil works, staged preliminaries, and OC / subdivision / strata pathway costs have a line, allowance, or explicit exclusion. Repeated dwellings create trade efficiency, but they also multiply design and services coordination errors.

**Drawdown discipline:**

- Contingency is **not available scope money**. Owner-initiated scope changes are variations, not contingency uses. Variations have their own register entry, owner sign-off, and pricing path (see §6).
- Contingency is for genuinely unforeseen costs: latent conditions, BASIX commitment recovery, neighbour-property remedy where the dilapidation report did not foresee it, PC reconciliation drift the cost plan held against contingency, and similar.
- Track **drawdown rate against percent complete**. If contingency is 40% consumed at 25% complete, that is a flag — escalate per `../00-doctrine/doctrine.md §escalation-triggers`.
- Release unused contingency in the cost report at PC, not before. Releasing contingency mid-project — even when the project looks clean — gives the owner false comfort and removes the reserve for the most common claim period (fixing stage and PC walk).

**Reference:** HIA cost-management practice; AIQS Cost Management Practice Guide (residential application).

**Common pitfalls:**
- Treating contingency as a slush fund for owner-initiated scope. The owner sees it consumed and assumes the build is going badly.
- Setting contingency at the lower band on a renovation with latent risk — the cost report goes red at first wall-opening.
- Not tracking drawdown rate — contingency falls off a cliff at fixing stage with no early warning.

## 6. Variations under HIA Schedule of Variations

In residential construction, variations are administered through the HIA Schedule of Variations form (or the equivalent stationery in MBA / NSW Fair Trading prescribed / AS 4000-4902 contracts). The discipline is the same across contract families:

1. **No work without written direction.** The variation form is issued and signed *before* the work begins. Verbal directions on site are not variations.
2. **Cost + time + scope** assessed in the variation form. Cost is the dollar adjustment (plus builder's margin per the contract). Time is the EOT impact in calendar days. Scope is what is included and what is not, and which trades are affected.
3. **Owner sign-off** is contractual. The owner signs the variation form. For owner-builder projects, the owner-builder signs against themselves (the discipline is for the project record, not external sign-off).
4. **Variation register entry** opens at proposal (`V-<seq>`, status `proposed`), moves through `priced` and `owner-signed`, lands at `completed`. Per `register-row-draft.md` standard register convention.
5. **Cost-plan update.** The cost plan's variation total runs alongside the contract sum baseline. A cost plan that does not reconcile to the variation register's running total has lost its audit trail.

The **mechanism difference** vs AS 4000/4902 commercial:
- No Superintendent in most residential contracts; the role is split (owner makes the call, builder issues the form, architect-PM where present advises). Get this right at commissioning.
- HIA Schedule of Variations is the contract's named form; the standard form has the price breakdown and time impact fields built in.
- Time impact is regularly underestimated in residential variations because the variation often touches trades that are already scheduled — the cascade through following trades is the time cost, not the variation work itself. The variation form must include this cascade in the time impact, not just the direct duration.

The **assessment** of variations against the contract sits in slice 06 (`variation-management-system`). This seed covers the **principles**: discipline at proposal, register entry, cost-plan reconciliation, time impact realism, owner sign-off pathway per role.

**Common pitfalls:**
- Site agreements not formalised: the builder does the work because the owner asked verbally on a Saturday site visit, then has no contractual basis to charge.
- Time impact captured only as direct duration: the variation register shows "2 days" when the trade cascade gives "10 days" of programme slip.
- Variation register and cost plan kept by different people in different files — they diverge, and at PC neither matches the actual.
- "It's just a small variation" — sub-$500 variations not formalised. Cumulatively these become a significant unpriced amount and a dispute trigger.

## 7. Progress claim assessment against HIA stage-payment schedule

The HIA Lump Sum stage-payment schedule for a new home is typically (subject to the executed contract's particulars):

- **Deposit** — up to the statutory cap (5% NSW; varies by state; check NSW Home Building Act for the current cap). Released at contract signing.
- **Base stage** — typically the slab pour for slab-on-ground construction (or the bearer/joist completion for footings-and-suspended-floor construction). Typical 10–15% of contract sum.
- **Frame stage** — frame complete and bracing fixed, internal door frames in (or framing trade complete on slab-on-ground). Typical 15–20%.
- **Lockup stage** — external cladding, windows, external doors fitted and lockable, roof on. Typical 20–25%.
- **Fixing stage** — internal linings, internal doors, skirtings and architraves, kitchen and bathroom carcasses, cabinetry, tiling. Typical 25–30%.
- **Completion stage** — final fixings, painting, commissioning, PC walk readiness. Typical 10–15%.

Specific percentages vary — the executed contract is the source of truth for a given project.

**Assessment discipline (cross-reference slice 06's `progress-claim-assessment-system`):**

1. **Physical stage achieved?** The claim is for stage completion. The stage definition in the contract is the test, not a percentage estimate. Walk the site and verify against the contract definition. Photos to file (per `07-construction/13-photos/`).
2. **Trade evidence in place?** For slab: concrete conformance test (CFT), reinforcement inspection record, slab inspection by certifier. For frame: structural inspection. For lockup: window installation, roofing, external cladding. For fixing: trade sign-offs. For completion: PC walk and BASIX final.
3. **Prior claims and variations.** Cumulative claimed amount + this claim ≤ contract sum + signed variations. Variations not yet signed do not count.
4. **Statutory timing.** The contract specifies the claim payment window (typically 5–10 business days for HIA residential). NSW Security of Payment Act applies where the contract triggers SOP.
5. **Retention.** Residential HIA contracts often do **not** carry retention — the stage-payment structure substitutes. Check the executed contract.
6. **Defects withholding.** Defects identified at PC stage are typically withheld against the completion-stage claim. Quantum is the rectification cost estimate, not punitive.

If the claim is not a simple HIA stage claim and instead includes a trade/work-package table or schedule of values, that table is also cost-plan evidence. A claim assessment should be able to read across from the cost plan to the claim schedule. If no cost plan exists yet, draft the cost plan from the claim schedule, contract sum, signed variations, and project budget evidence before assessing whether the claim is reasonable.

The **principles** sit in this seed. The **system** is slice 06. The role overlay says **who** does the assessment:

- `user_role: builder` issues the claim;
- `user_role: architect-pm` typically assesses on behalf of the owner;
- `user_role: owner-builder` issues claims to subcontractors and assesses them as the head — does not issue head-contractor claims to themselves;
- `user_role: d-and-c` issues head claims and assesses subcontractor claims.

**Common pitfalls:**
- Claim issued before stage physically achieved — particularly the frame and lockup stages. Builder claims at "frame complete bar one window" — that is not lockup.
- Owner pays the claim because the percentage looks right against the schedule, without checking physical completion. The contract permits withholding; the discipline is to use it where evidence is incomplete.
- Variations rolled into stage claims without separate identification — the contract sum and the variation total are not separately visible.
- BASIX final certificate or OC missing at completion-stage claim — the completion stage cannot be claimed until these are in hand if the contract conditions claim on them.

## 8. GST in residential cost reporting

The GST convention in residential is different from commercial. Default residential cost plans are presented **GST-inclusive**.

The reason: most residential owners are owner-occupiers, not GST-registered entities. The owner cannot recover GST on construction costs through a BAS. GST is a real cost of $0.10 on every dollar. Presenting the cost plan GST-exclusive misleads the owner about their actual financial commitment by exactly the GST amount.

**Exceptions where GST-exclusive is appropriate:**
- The owner is a GST-registered entity (e.g. an investor running the property as a rental, or a developer building to sell), and GST is recoverable. In this case the cost plan should still **show both** the GST-exclusive figure and the GST amount to make the cashflow timing visible (the owner pays GST upfront and recovers it on BAS — there is a cash gap).
- An architect-PM working for an investor / developer owner — present GST-exclusive with GST shown separately.
- A multi-dwelling project where the dwellings are being sold (margin scheme may apply to the land component — get advice).

**The cost plan must state its GST basis explicitly in the frontmatter or in the cost-plan summary section.** Mixing GST-inclusive and GST-exclusive figures in the same cost plan is a §evidence-discipline failure that misleads readers by 10% — flag and rebase.

**Cashflow note:** even where the owner can recover GST, the cashflow timing matters. The owner pays GST on each progress claim and recovers it on the next quarterly BAS (or monthly for large taxpayers). For a residential investor, this is typically a 1–3 month float. Surface it.

**Reference:** A New Tax System (Goods and Services Tax) Act 1999; ATO residential construction guidance.

## 9. Role-divergent cost responsibilities

Cost-management responsibilities split across the four user roles. The seed below is the abstract project-lead view; the role overlay seeds are the source of truth for who does what.

| Role | Issues claims | Assesses claims | Pays claims | Owns cost plan | Owns variation register | Holds contingency |
|---|---|---|---|---|---|---|
| `owner-builder` | To subcontractors only | Own assessment of subcontractors | Pays subcontractors directly | Self | Self | Self |
| `architect-pm` | Does not issue | Assesses builder's claims on owner's behalf | Advises owner; owner pays | Architect-PM drafts; owner approves | Architect-PM maintains | Owner; architect-PM advises drawdown |
| `builder` | Issues head claims to owner; assesses subcontractor claims | Builder's internal | Owner pays to builder; builder pays subcontractors | Builder drafts; owner reviews | Builder maintains; owner signs each entry | Builder's contract contingency; owner may hold separate |
| `d-and-c` | As builder, plus design-fee claims if separate | As builder | Owner pays | D&C drafts | D&C maintains | D&C; owner may hold separate |

The role overlay seed loaded for the active project (per `../02-skills/atomic/seed-targeted-read.md`'s mandatory Tier 3 load) is the operational truth for these splits. This seed names the dimensions; the role overlay names the obligations.

For `user_role: d-and-c`, keep the builder construction sum, design fee, consultant fees, PI / consultant PI allowance, certifier submission allowance, authority / infrastructure contributions, party-wall / fire / acoustic allowances, separate metering / service upgrade costs, staging preliminaries, and per-dwelling vs shared infrastructure split visible. A design-fee claim is assessed against the design deliverables register, design programme, and certifier submission status where relevant; construction progress alone does not prove design-fee entitlement.

**Escalation routing** (per `../00-doctrine/doctrine.md §escalation-triggers`):
- **Owner-builder** — cost escalations go to the self-flag log in `08-meetings-reporting/`.
- **Architect-PM** — cost escalations route to the owner via a `§owner-communication`-shaped summary in `08-meetings-reporting/owner-update*`.
- **Builder** — cost escalations route to the owner per contract; high-value or contractual triggers route via `07-construction/08-rfi-notices/` for the written record.
- **D&C** — as builder, plus design-side cost issues route to the owner where PPR, cost, or programme choices are required; to the certifier where compliance is affected; and to the responsible consultant where scope or design advice is live.

## 10. State callouts (graceful degradation per AGENTS.md §8)

NSW is the deep default in this seed. Non-NSW callouts:

**VIC** — *Slice 14 will deepen this.* HIA contracts are used widely; the prescribed state contract is different. Domestic Building Insurance (DBI) replaces NSW HBCF. CoINVEST replaces NSW Long Service Corporation. The Security of Payment regime is BCISP Act 2002 (VIC). BASIX is not the energy instrument — NatHERS 6-star (or 7-star post-2022 amendment depending on adoption date) applies via the Victorian Building Authority. Building Act 1993 (VIC) carries different defect liability and statutory warranty provisions. **Treat as Assumption** in any cost plan for a VIC project — confirm against current VIC instruments before relying on residential conventions that differ from NSW.

**QLD** — *Callout only; deep coverage deferred.* QBCC replaces NSW Fair Trading and HBCF (Home Warranty Insurance is mandatory under QBCC). The Security of Payment regime is the Building Industry Fairness (Security of Payment) Act 2017 (QLD), which has materially different payment claim mechanics and statutory cooling-off. BASIX does not apply — QDC Part MP 4.1 / NCC Section J. Treat as Assumption.

**SA, WA, TAS, NT, ACT** — *Callout only; deep coverage deferred.* Each state has its own statutory warranty regime (e.g. WA BSA, TAS Building Act, NT Building Practitioners Board). Energy compliance varies by state adoption of NCC. Treat as Assumption and flag for project lead supplementation.

Where a non-NSW state has no callout for the task at hand, the skill (calling this seed) **flags the gap** rather than silently extending NSW guidance — per `../AGENTS.md §8` and `../00-doctrine/doctrine.md §state-handling`.

## 11. Cost reporting cadence and structure

Residential cost reports are typically monthly during construction, aligned to the HIA stage claim cycle. The report's job is to keep the owner's cost picture current and to surface drift early.

A residential cost report carries:

- **header**: project, period, author, status (draft until reviewed);
- **baseline**: contract sum, signed variations to date, current contract value (sum + variations);
- **claimed-to-date**: cumulative across stage claims (with each claim's date, amount, assessment status, payment status);
- **paid-to-date**: cumulative against claimed (gap is unpaid or disputed);
- **forecast to complete**: forecast cost remaining + any anticipated variations;
- **contingency drawdown**: opening contingency, used this period, used to date, remaining, drawdown rate vs % complete;
- **PC reconciliation**: PC line items with allowance vs actual where known, running net reconciliation total;
- **owner-supplied items**: status of each item (committed / ordered / delivered / installed / late);
- **RAG status**: green / amber / red against approved budget (see thresholds below);
- **narrative**: why anything moved; what is expected next period;
- **decisions required from owner**: per `§owner-communication` if the report is owner-facing.

**RAG thresholds (residential default):**
- **Green** — forecast total ≤ contract sum + signed variations + 30% of remaining contingency.
- **Amber** — forecast total within 5% above contract sum + signed variations, or contingency drawdown rate is materially ahead of % complete.
- **Red** — forecast total > 5% above contract sum + signed variations, or contingency is consumed and remaining work carries identified risk.

**Voice register** (per `../00-doctrine/doctrine.md §voice-and-style`): cost reports drafted into `01-cost/` are contractual register (formal Australian English, clause-cited where contractual mechanism is referenced, ISO short-form dates, AUD with $ symbol). Owner-facing monthly cost summaries drafted into `08-meetings-reporting/owner-update*` are stakeholder register (plain English, leads with "what this means for you" / "what we need from you" per `../00-doctrine/doctrine.md §owner-communication`).

The two are different documents. The contractual cost report sits in `01-cost/`. The owner-facing summary derived from it sits in `08-meetings-reporting/`. Do not collapse them — the owner needs the plain-language version and the project record needs the contractual version.

## 12. Cashflow and the residential S-curve

Residential cashflow follows the stage-payment schedule's curve, not the S-curve of commercial monthly measured progress. For a typical NSW Class 1a HIA Lump Sum project:

- **Month 0** (deposit + base): ~10–15% of contract sum;
- **Months 1–3** (frame + lockup): ~35–45% cumulative;
- **Months 3–6** (fixing): ~70–75% cumulative;
- **Months 6–9** (completion): ~100% cumulative.

Actual timing depends heavily on the executed programme. The cashflow forecast must mirror the executed contract's claim schedule, not a generalised S-curve.

**Cashflow risk surfacing:**
- Owner construction-loan drawdown windows often require certifier or QS sign-off at each draw. The cashflow forecast must align to draw timing.
- GST timing for investor owners (see §8) introduces a 1–3 month cash gap between payment and BAS recovery.
- Retention (if applicable — see §7) timing affects the back-end cashflow.
- Final claim release timing is contingent on PC walk close-out — model both an optimistic and pessimistic close-out date.

## 13. What this seed is and is not

This seed is **principles**. It is loaded by `seed-targeted-read` when the task signal is cost / budget / contingency / claim / variation pricing. It is not a set of skill steps and it does not draft.

For the **workflow** (seeding → sweeping → drafting → workbook update → verification), call `../02-skills/systems/cost-plan-system.md` — the residential cost-plan system skill. The system skill loads this seed plus the archetype seed plus the role overlay, and orchestrates the atomic skills.

For **register row format** for cost / variation / PC reconciliation / owner-supplied rows, call `../02-skills/atomic/register-row-draft.md`.

For **markdown draft wrapping** of any cost-plan narrative output, call `../02-skills/atomic/markdown-draft-for-review.md`. The voice register for `01-cost/cost-plan*`, `01-cost/cost-report*`, `01-cost/*claim-assessment*`, `01-cost/*variation-pricing*` is contractual.

For **workbook edits** to the project's cost workbook (e.g. a Master Project Finance.xlsx), call `../02-skills/atomic/excel-safe-edit.md` driven by an approved markdown source, followed by `../02-skills/atomic/excel-verify.md`. The atomics' hard rules (backup first, preserve named ranges and data validations, never `#REF!`, never overwrite the only copy, never commit assumptions directly) are non-negotiable.

This seed does not cover:
- **detailed elemental cost rates** (e.g. $/m² rates for slabs, framing, roofing) — those belong in a future trade-cost reference (not in scope for slice 03);
- **specific contract clause references** for HIA / MBA / NSW Fair Trading / AS — that is `contract-administration-guide.md`'s job;
- **progress claim assessment workflow** — that is slice 06's `progress-claim-assessment-system`;
- **variation pricing workflow** — that is slice 06's `variation-management-system`;
- **handover and final account** — that is slice 13's `handover-pc-system`.

## See also

- `../AGENTS.md §1` — authority stack (project evidence beats this seed)
- `../AGENTS.md §3` — seed loading rules (this seed is a Tier-cross-cutting topic seed)
- `../AGENTS.md §5` — output discipline (frontmatter on every cost-plan deliverable)
- `../AGENTS.md §6` — voice register (`01-cost/` is contractual per the table in `markdown-draft-for-review.md`)
- `../AGENTS.md §8` — state handling (non-NSW callouts above are graceful-degradation placeholders)
- `../00-doctrine/doctrine.md §evidence-discipline` — Fact / Assumption / Judgement / Recommendation labelling required on every cost figure
- `../00-doctrine/doctrine.md §seed-consultation-discipline` — why this seed must be loaded by `seed-targeted-read`, not assumed
- `../00-doctrine/doctrine.md §register-discipline` — cost / variation / PC reconciliation register schema
- `../00-doctrine/doctrine.md §voice-and-style` — two-register split (contractual for `01-cost/` content; stakeholder for owner-facing derivatives)
- `../00-doctrine/doctrine.md §owner-communication` — format for the owner-facing cost summary derived from the cost report
- `../00-doctrine/doctrine.md §escalation-triggers` — cost-related triggers (budget movement, contingency drawdown, BASIX cost recovery, PC reconciliation drift)
- `role-builder.md`, `role-architect-pm.md`, `role-owner-builder.md`, `role-d-and-c.md` — role-divergent obligations referenced in §9
- `contract-administration-guide.md` — clause-citation discipline and contract-family mechanics
- `setup-and-commission-guide.md` — opens the cost register and the owner-supplied items register at commissioning
- `../02-skills/systems/cost-plan-system.md` — primary caller; the workflow that loads this seed
- `../02-skills/atomic/seed-targeted-read.md` — loads this seed
- `../02-skills/atomic/evidence-sweep.md` — finds project cost evidence before drafting
- `../02-skills/atomic/markdown-draft-for-review.md` — wraps cost-plan narrative drafts
- `../02-skills/atomic/register-row-draft.md` — drafts cost / variation / PC / owner-supplied register rows
- `../02-skills/atomic/excel-safe-edit.md`, `../02-skills/atomic/excel-verify.md` — workbook update path

# Commercial and Non-Residential Expansion

## Cost Planning Fundamentals

Cost planning is the structured process of forecasting and controlling the capital cost of a construction project from inception through to completion. The objective is to provide the client with reliable cost advice at each stage of design development, enabling informed decisions about scope, quality, and budget.

In Australian practice, cost planning follows a progressive refinement model. At the earliest stage (feasibility), estimates may carry an accuracy tolerance of +/- 25-30%. By design development, this narrows to +/- 10-15%. At tender stage, the estimate should be within +/- 5% of the tendered price. Each stage of design development produces a more granular cost plan, moving from order-of-cost estimates based on $/m2 rates to fully measured elemental estimates with trade breakdowns.

The cost plan is a living document. It must be updated whenever design changes occur, market conditions shift, or new information emerges. A cost plan that is only produced at the start and never revisited will inevitably diverge from reality.

Reference: AIQS Cost Management Practice Guide; AS 4122-2010 (General conditions for engagement of consultants)
Applies to: All building classes and project types
Related standards: AIQS Standard Method of Measurement, NRM1 (UK, sometimes referenced in Australian practice)

Common pitfalls:
- Producing a cost plan at feasibility and not updating it through design development, leading to tender shock
- Failing to clearly state the basis, exclusions, and qualifications of the estimate
- Not distinguishing between construction cost and total project cost (which includes fees, FF&E, authority charges, land)

## Contingency Allowances by Project Type

Contingency is a budget provision for unforeseen costs that emerge during the project lifecycle. The appropriate contingency percentage depends on the project type, stage of design development, and risk profile. Contingency is not a slush fund for scope additions — it exists to absorb genuinely unforeseen costs such as latent conditions, design coordination gaps, and market price movements.

Typical contingency ranges in Australian construction:
- New build residential (Class 1): 5-10% of construction cost
- Multi-residential apartments (Class 2): 8-12% of construction cost
- Commercial office fit-out (Class 5): 5-8% of fit-out cost
- Commercial new build (Class 5-9): 8-10% of construction cost
- Refurbishment of existing buildings: 10-15% of construction cost
- Remediation projects: 20-30% of estimated remediation cost
- Heritage restoration: 15-25% of construction cost

These ranges assume a project at design development stage. Earlier stages (feasibility, concept) warrant higher contingency; later stages (tender, construction) warrant lower contingency as costs become more certain. A well-managed project should see contingency reduce from 15-20% at concept to 5% or less at construction stage.

Reference: AIQS Cost Management Practice Guide
Applies to: All building classes and project types
Related standards: AS 4122-2010 (consultant engagement)

Common pitfalls:
- Setting contingency too low on remediation or refurbishment projects where unknowns are inherently high
- Not adjusting contingency downward as design progresses and costs firm up (contingency hoarding)
- Double-counting risk allowances that are already included in trade package budgets
- Failing to track contingency drawdown against percentage of project completion
- Treating contingency as available for client-initiated scope additions rather than genuine unforeseen costs

## Design Contingency vs Construction Contingency

It is important to distinguish between design contingency and construction contingency, as they serve different purposes and are managed differently.

Design contingency covers the cost uncertainty inherent in incomplete design documentation. At concept stage, many design decisions have not been made, so the cost plan includes an allowance for the likely cost impact of those unresolved elements. As design progresses and decisions are made, design contingency is either absorbed into specific trade packages (if costs exceed initial allowances) or released back to the overall contingency pool (if costs are lower than expected). Design contingency should be fully absorbed by the time design is complete and documents are issued for tender.

Construction contingency covers risks that materialise during the construction phase, such as latent site conditions, weather delays, subcontractor insolvencies, material supply disruptions, and coordination issues not foreseeable from the design documents. Construction contingency typically sits at 3-5% of the contract sum for well-documented projects and may be higher for complex or high-risk projects.

Reference: AIQS Cost Management Practice Guide
Applies to: All building classes and project types

Common pitfalls:
- Carrying design contingency into the construction phase without explanation
- Not tracking the conversion of design contingency into firm trade costs as design progresses
- Setting construction contingency based on a percentage of the tender budget rather than assessing the actual risk profile of the project

## Budget Structure and Trade Packages

A construction budget is structured in a hierarchy that enables tracking at multiple levels of detail. The typical structure in Australian practice is:

- **Total project cost** — the complete cost to deliver the project, including land, fees, construction, FF&E, authority charges, and contingency
- **Construction cost** — the cost of physical construction works, including preliminaries, subcontract packages, and contractor margin
- **Trade packages** — individual work scopes typically aligned with subcontract procurement (e.g., concrete, structural steel, mechanical, electrical, hydraulic, facade)
- **Preliminaries** — the head contractor's time-related and fixed costs for site management, temporary works, scaffolding, cranes, insurances, and project administration
- **Margins** — the head contractor's overhead and profit, typically expressed as a percentage of the trade subtotal
- **Provisional sums** — allowances for work not yet fully designed or specified at the time of tender
- **Prime cost items** — allowances for specific materials or equipment to be selected by the client or architect

Preliminaries typically range from 12-18% of the trade subtotal for medium-to-large commercial projects, and can be higher (up to 25%) for small or complex projects. Margins (overhead and profit) typically range from 4-8% for competitive markets and 8-12% in constrained markets.

Reference: AIQS Standard Method of Measurement; AS 4000-1997 (General conditions of contract)
Applies to: All building classes and project types

Common pitfalls:
- Not breaking the budget down to trade package level early enough, leading to difficulty comparing tender returns against budget
- Failing to separately identify preliminaries and margins, making it hard to assess contractor pricing competitiveness
- Lumping provisional sums and prime cost items into trade totals rather than tracking them separately for adjustment on completion

## Cost Reporting and Forecast Management

Effective cost reporting requires clear definitions of the cost categories being tracked. The key financial metrics in Australian construction cost management are:

- **Original budget** — the approved budget at project commencement
- **Approved budget** — the original budget plus any approved variations or scope changes
- **Committed cost** — the total value of awarded contracts, purchase orders, and approved variations
- **Forecast cost** — the current estimate of the total cost to complete, including committed costs, anticipated variations, and remaining contingency
- **Actual spend** — amounts certified and paid to date
- **Cost to complete** — the forecast cost minus actual spend

A traffic light (RAG) status is commonly used to flag budget health:
- Green: forecast is within 5% of approved budget
- Amber: forecast is 5-10% above approved budget, or contingency drawdown is exceeding the rate of progress
- Red: forecast exceeds approved budget by more than 10%, or contingency is fully consumed with significant work remaining

Cost reports should be produced monthly (aligned with progress claim cycles) and include a narrative explanation of any material movements. The narrative is as important as the numbers — it explains the why behind the movements and the corrective actions being taken.

Reference: AIQS Cost Management Practice Guide
Applies to: All building classes and project types

Common pitfalls:
- Reporting only committed costs without forecasting anticipated variations and claims, giving a misleadingly positive picture
- Not reconciling cost reports back to the original budget structure, making it hard to identify where overruns are occurring
- Producing cost reports without narrative explanation, leaving stakeholders to interpret the numbers without context
- Confusing actual spend (amounts paid) with committed cost (amounts contracted), which can understate the true financial position

## Progress Claims and Payment Assessment

Progress claims (also called progress payments or payment claims) are the mechanism by which contractors and subcontractors invoice for work completed during a defined period. In Australia, progress claim assessment is governed by both the contract terms and the Security of Payment legislation in each state and territory.

The typical monthly progress claim cycle in Australian construction is:

1. **Claim submission** — the contractor submits a payment claim showing the value of work completed in the period, usually by a date specified in the contract (e.g., the 25th of each month)
2. **Assessment** — the superintendent (or contract administrator) assesses the claim by comparing the claimed amounts against actual progress, contract rates, and approved variations
3. **Payment schedule** — the principal issues a payment schedule (or payment certificate) showing the amount to be paid, with reasons for any differences from the claimed amount
4. **Payment** — the principal pays the scheduled amount within the contractual payment period (typically 15-30 business days from claim submission)

Retention is typically withheld at 5% of each progress claim up to a maximum of 5% of the contract sum. Half the retention is released at practical completion and the remaining half at the end of the defects liability period (typically 12 months after practical completion).

Reference: Security of Payment Act (varies by state: NSW Building and Construction Industry Security of Payment Act 1999, VIC Building and Construction Industry Security of Payment Act 2002, QLD Building Industry Fairness (Security of Payment) Act 2017)
Applies to: All building classes and project types
Related standards: AS 4000-1997 Clause 37, AS 2124-1992 Clause 42

Common pitfalls:
- Missing the statutory timeframe for issuing a payment schedule, which can result in the claimed amount becoming due in full
- Not assessing variations separately from measured work in progress claims
- Failing to withhold retention as required by the contract, creating exposure if defects emerge
- Not reconciling cumulative progress claims against the contract sum and approved variations

## Variations — Types and Assessment

A variation is a change to the scope, quality, or timing of the contracted works. Variations are one of the most significant sources of cost overrun in construction projects. Effective variation management requires prompt identification, assessment, and approval.

The main types of variations in Australian construction contracts are:

- **Client-directed scope changes** — additions or modifications requested by the client or their design team after contract award
- **Latent conditions** — unforeseen physical conditions at the site that differ materially from those that a competent contractor could have reasonably anticipated (e.g., contaminated soil, unexpected rock, uncharted services)
- **Provisional sum adjustments** — the difference between the provisional sum allowance in the contract and the actual cost of the work when it is designed and procured
- **Regulatory changes** — variations required to comply with changes in laws, codes, or regulations that occur after the contract date
- **Errors and omissions in documentation** — discrepancies, ambiguities, or missing information in the contract documents that require additional work to resolve

Variation assessment should follow the contract's valuation mechanism. Under AS 4000-1997 (Clause 36), variations are valued using contract rates where applicable, reasonable rates where no contract rate exists, or on a cost-plus basis as a last resort. The superintendent must direct the variation in writing before the contractor commences the work, unless the urgency of the situation makes prior direction impractical.

Reference: AS 4000-1997 Clause 36 (Variations); AS 2124-1992 Clause 40
Applies to: All building classes and project types
Related standards: Security of Payment legislation (for payment of variation claims)

Common pitfalls:
- Allowing work to proceed without a formal written variation direction, making it difficult to verify the scope and cost later
- Assessing all variations on a cost-plus basis rather than using contract rates, which inflates costs
- Not maintaining a variation register with running totals, leading to cumulative cost creep that is not visible until it is too late
- Confusing provisional sum adjustments (contractual entitlement) with discretionary scope changes (require approval)

## Provisional Sums — Defined and Undefined

Provisional sums are allowances included in a construction contract for work that is not fully designed or specified at the time of tender. They are a common and necessary feature of Australian construction contracts, but their management has significant cost implications.

Australian Standard AS 4000-1997 distinguishes between two types of provisional sums:

- **Defined provisional sums** — the work scope is sufficiently described for the contractor to have included for its programming, planning, and pricing of preliminaries. On completion of the work, the provisional sum is adjusted to the actual cost, but the contractor is not entitled to additional time or money for preliminaries because it was assumed they had priced for it.

- **Undefined provisional sums** — the work scope is not described in sufficient detail for the contractor to have reasonably allowed for it in programming and pricing. On completion of the work, the contractor is entitled to both the actual cost adjustment and any additional time and preliminaries costs resulting from the work.

The classification of a provisional sum as defined or undefined has significant cost consequences. An undefined provisional sum effectively gives the contractor an entitlement to a variation for time and preliminaries on top of the actual cost of the work. Projects with a high proportion of undefined provisional sums carry greater cost risk.

Reference: AS 4000-1997 Clause 1 (definitions) and Clause 36.5; Australian Building Industry Forum guidance
Applies to: All building classes and project types

Common pitfalls:
- Not clearly designating provisional sums as defined or undefined in the contract, leading to disputes about the contractor's entitlement to additional time and costs
- Including an excessive number of provisional sums (indicating the design is not sufficiently advanced for tender), which undermines the reliability of the contract sum
- Failing to adjust provisional sums on completion of the work, leaving phantom allowances in the cost plan
- Not tracking the aggregate value of provisional sum adjustments as a separate cost category in reports

## Rise and Fall (Escalation)

Rise and fall provisions in construction contracts address the impact of price changes for labour and materials that occur between the contract date and the time the work is actually performed. In a volatile market, escalation can represent a significant and often underestimated cost risk.

There are three main approaches to escalation in Australian construction contracts:

1. **Fixed price (no rise and fall)** — the contractor accepts all price risk. This is common for projects under 12 months duration or in stable market conditions. The contractor will include a risk premium in their tender price to cover anticipated price movements.

2. **Index-based adjustment** — the contract sum is adjusted periodically based on published cost indices (e.g., ABS Producer Price Indexes for construction). This shares the escalation risk between the principal and contractor. The adjustment formula typically references specific indices for labour, materials, and plant.

3. **Actual cost adjustment** — less common, where the contractor is reimbursed for actual price increases supported by evidence (supplier invoices, subcontract price increases). This places the full escalation risk on the principal.

For cost planning purposes, a QS should include an escalation allowance based on the anticipated construction duration and current market trends. A common approach is to apply half the annual escalation rate to the full contract sum (assuming work is evenly distributed over the construction period). Current market escalation in the Australian construction industry has ranged from 3-8% per annum in recent years, depending on location and trade.

Reference: AS 4000-1997 Clause 36.3 (rise and fall); ABS Cat. 6427.0 Producer Price Indexes
Applies to: All building classes and project types, particularly relevant for projects with construction durations exceeding 12 months

Common pitfalls:
- Using a fixed-price contract for a long-duration project in a rising market, exposing the contractor to untenable risk (which often results in claims and disputes)
- Not including an escalation allowance in the cost plan, leading to budget shortfall when the contract includes rise and fall provisions
- Applying escalation to the full contract sum from day one, rather than using a mid-point method that accounts for the S-curve distribution of expenditure
- Failing to monitor actual escalation against the allowance during construction, missing early warning signals of budget pressure

## Cashflow Forecasting

Cashflow forecasting predicts the timing and magnitude of project expenditure over the construction period. Accurate cashflow forecasting is essential for the client's financial planning, funding drawdown schedules, and early identification of cost or program issues.

The most common approach in Australian practice is the S-curve method, which plots cumulative expenditure against time. A standard S-curve follows a characteristic pattern: slow initial spend during mobilisation and early works, accelerating through the main construction period, and tapering off during commissioning and defects rectification.

Simplified rules of thumb for S-curve expenditure distribution:
- First quarter of construction duration: ~10-15% of contract value
- Second quarter: ~25-30% of contract value
- Third quarter: ~30-35% of contract value
- Final quarter: ~20-25% of contract value

The cashflow forecast should be updated monthly alongside the cost report and progress claim assessment. Significant deviations from the forecast S-curve are early warning signals that require investigation. An actual spend curve running ahead of the forecast may indicate accelerated work (positive) or overpayment on claims (negative). An actual curve running behind may indicate program delays or under-claiming.

Reference: AIQS Cost Management Practice Guide
Applies to: All building classes and project types

Common pitfalls:
- Producing a cashflow forecast at the start of the project and never updating it, making it useless as a management tool
- Not accounting for retention in the cashflow (5% withheld on each claim affects the timing of cash outflows)
- Assuming linear expenditure rather than an S-curve distribution, which materially misrepresents the timing of major payments
- Failing to align the cashflow forecast with the contractor's program, particularly when program changes occur

## Value Engineering

Value engineering (VE) is a structured process to identify opportunities for reducing cost without compromising the functional requirements or quality of the project. It is not simply cost cutting — effective VE maintains or improves value (the ratio of function to cost) while reducing overall expenditure.

The optimal timing for value engineering in the project lifecycle is:

- **Concept/schematic design stage** — the greatest potential for cost savings with the least disruption. Changes at this stage are inexpensive to implement because documentation has not progressed far. VE at this stage might reconsider structural systems, facade approaches, or services strategies.
- **Design development stage** — moderate potential, with increasingly detailed savings. Changes affect partially completed documentation. VE at this stage might substitute materials, simplify detailing, or rationalise services zones.
- **Construction documentation stage** — limited potential, and changes risk abortive design work. Only high-value items justify VE at this stage.
- **Construction stage** — VE is generally impractical because changes trigger variations, redesign, and delay. The cost of change typically exceeds the savings.

A structured VE workshop involves the design team, QS, and sometimes the contractor (in early contractor involvement or D&C contracts). Each element of the design is scrutinised against three questions: (1) what function does it serve? (2) what does it cost? (3) is there a more cost-effective way to achieve the same function?

Reference: AIQS Cost Management Practice Guide; SAVE International Value Methodology Standard
Applies to: All building classes and project types

Common pitfalls:
- Conducting VE too late in the design process, when the cost of change exceeds the potential savings
- Treating VE as a cost-cutting exercise rather than a value optimisation exercise, leading to reduced quality and increased lifecycle costs
- Not involving the QS in VE workshops, resulting in proposed alternatives that have not been costed
- Failing to document the VE decisions and their cost impact in the cost plan, making it impossible to track the cumulative benefit

## Quantity Surveying and Elemental Cost Plans

The quantity surveyor (QS) is the cost management professional responsible for preparing cost plans, assessing progress claims, valuing variations, and providing financial advice on construction projects. In Australia, QS practitioners are typically members of the Australian Institute of Quantity Surveyors (AIQS).

An elemental cost plan organises construction costs by building element rather than by trade or specification. The standard elemental breakdown used in Australian practice follows the AIQS Standard Method of Measurement and includes:

- **Substructure** — foundations, basement, ground floor slab
- **Superstructure** — structural frame, upper floors, roof, stairs
- **External envelope** — external walls, windows, doors, facade
- **Internal finishes** — partitions, wall finishes, floor finishes, ceiling finishes
- **Fittings** — joinery, built-in furniture, signage
- **Services** — mechanical (HVAC), electrical, hydraulic, fire, lift, communications, security
- **External works** — paving, landscaping, drainage, fencing, car parks
- **Preliminaries** — site management, temporary works, scaffolding, insurances

Each element is measured and priced using rates derived from the QS firm's cost database, benchmarked against recent comparable projects. The elemental breakdown enables comparison across projects of different types and sizes, and helps identify where a design is over- or under-costed relative to benchmarks.

Reference: AIQS Standard Method of Measurement; AIQS Building Cost Index
Applies to: All building classes and project types

Common pitfalls:
- Using outdated cost rates without adjusting for current market conditions and escalation
- Not reconciling the elemental cost plan back to the total budget including non-construction costs (fees, FF&E, authorities)
- Relying on a single benchmark project rather than a range, which can introduce bias from project-specific factors
- Failing to update the elemental cost plan as design develops, allowing it to become a stale reference document

## Tender Budget vs Contract Sum Reconciliation

When tenders are received, the QS must reconcile the tendered prices against the pre-tender estimate (budget). This reconciliation identifies where the market pricing differs from the budget assumptions and helps the client understand the financial implications of awarding the contract.

The reconciliation process involves:

1. **Normalise the tenders** — adjust all tenders to a common basis by adding back any qualifications, exclusions, or alternative offers so they can be compared fairly
2. **Compare against budget** — map the tender breakdown against the budget elements to identify over- and under-priced items
3. **Assess reasonableness** — evaluate whether significant price differences reflect genuine market pricing or errors/misunderstandings in the tender documents
4. **Identify risks** — flag any items where the tendered price appears unrealistically low (potential for claims and variations) or where key items have been excluded

A common outcome is that the lowest tender is higher than the pre-tender budget. In this case, the QS should advise on whether the difference is due to market conditions (requiring a budget adjustment), design issues (requiring VE), or tender-specific factors (requiring negotiation).

Reference: AIQS Cost Management Practice Guide; AS 4120-1994 (Code of tendering)
Applies to: All building classes and project types

Common pitfalls:
- Accepting the lowest tender without reconciliation, missing exclusions or qualifications that will generate variations post-award
- Not adjusting the budget to reflect the actual contract sum, leaving a false budget baseline for cost reporting
- Failing to investigate significant variances between tenderers for the same trade items, which may indicate different interpretations of the scope

## Head Contract vs Subcontract Cost Management

In traditional procurement, the head contractor (main contractor) engages subcontractors for most trade work. The client contracts only with the head contractor, who manages all subcontract relationships. This creates a layered cost structure that the QS must understand.

The typical cost hierarchy is:

- **Subcontract cost** — the price the subcontractor charges the head contractor for the trade work
- **Head contractor margin** — the overhead and profit the head contractor applies to subcontract costs (typically 5-15%)
- **Head contractor preliminaries** — the site management, temporary works, and administration costs that the head contractor charges for managing the project
- **Client-side costs** — professional fees, authority charges, and other costs outside the construction contract

Margin stacking occurs when costs pass through multiple layers (sub-subcontractors to subcontractors to head contractor), with each layer adding its margin. On complex projects, margin stacking can add 15-25% to the base cost of the work. The QS should be aware of this effect when comparing trade costs against benchmarks.

Reference: AS 4000-1997; AS 4901-1998 (Subcontract conditions for use with AS 4000)
Applies to: All building classes and project types using traditional or construct-only procurement

Common pitfalls:
- Not accounting for head contractor margins and preliminaries when comparing direct subcontract quotes against budget
- Failing to recognise that cost savings achieved through subcontract negotiation are partially offset by head contractor margin on the savings
- Not separately tracking head contractor claims for time-related preliminaries when program extensions occur

## Insurance and Bonds in Construction

Insurance and bonds are significant cost items in construction contracts that are often underestimated in budget allowances. The typical insurance and bond requirements in Australian construction contracts include:

**Contract works insurance (also called construction all-risks)** covers physical loss or damage to the works during construction. Under AS 4000-1997, the principal is responsible for effecting this insurance. The premium is typically 0.3-0.8% of the contract sum, depending on the risk profile and coverage requirements.

**Public liability insurance** covers claims by third parties for injury or property damage arising from the construction works. The contractor is typically required to maintain $10-20 million public liability cover. Cost is included in the contractor's preliminaries.

**Professional indemnity insurance** is required for design consultants (and the contractor in D&C contracts) to cover claims arising from errors in professional advice. Cover is typically maintained for 6-10 years after project completion. Premiums have increased significantly in recent years, particularly in the building industry following cladding and waterproofing defect issues.

**Bank guarantees** (or unconditional undertakings) serve as security for performance. The contractor typically provides bank guarantees equal to 5% of the contract sum, which are progressively released after practical completion and the end of the defects liability period. The cost of maintaining bank guarantees (typically 1-2% per annum of the guarantee face value) is a real cost to the contractor and is priced into their tender.

Reference: AS 4000-1997 Clauses 16-18 (Insurance and indemnity); AS 2124-1992 Clauses 16-18
Applies to: All building classes and project types

Common pitfalls:
- Not including insurance premiums and bank guarantee costs in the budget, which can add 1-3% to the total project cost
- Assuming the contractor includes all insurance costs in their tender without checking what the contract requires the principal to provide
- Not reviewing insurance policies for exclusions that may leave the project exposed (e.g., design defect exclusions in contract works policies)

## GST and Tax Treatment in Cost Reporting

All costs in Australian construction are subject to Goods and Services Tax (GST) at 10%. Cost reporting conventions vary between organisations, and it is critical to establish at the outset whether the cost plan and reports are prepared on a GST-inclusive or GST-exclusive basis.

The standard practice for most project managers and quantity surveyors is to report on a GST-exclusive basis, because GST is a pass-through cost for GST-registered entities. The client recovers the GST paid on construction costs through their Business Activity Statement (BAS). However, if the client is not GST-registered (e.g., a residential owner-occupier), GST is a real cost and must be included in the budget.

Key GST considerations in construction:
- The margin scheme may apply to residential property development, affecting the GST treatment of the land component
- Contractor payments include GST and must be grossed up in cashflow forecasts for clients who have cash constraints
- Not all project costs attract GST (e.g., some government charges, some financial services)

Reference: A New Tax System (Goods and Services Tax) Act 1999; ATO construction industry guidance
Applies to: All building classes and project types

Common pitfalls:
- Mixing GST-inclusive and GST-exclusive figures in the same cost report, creating confusion about the true cost position
- Not clarifying the GST basis at the start of the project, leading to budget misunderstandings of 10%
- Failing to account for GST in cashflow forecasts, particularly when the client has limited cash reserves and cannot fund the GST component pending BAS recovery

## Final Account Settlement

The final account is the definitive statement of the total amount payable under a construction contract. It represents the reconciliation of the original contract sum, all approved variations, provisional sum adjustments, rise and fall, and any other contractual entitlements. Settling the final account is one of the last acts in the contractual relationship and should be concluded as promptly as possible after practical completion.

The typical final account process is:

1. **Contractor submits a final payment claim** — including all remaining measured work, approved variations, provisional sum adjustments, and any outstanding claims
2. **QS or superintendent assesses the claim** — verifying quantities, rates, variation valuations, and provisional sum adjustments against the contract and supporting documentation
3. **Negotiation** — where the parties disagree on specific items, they negotiate to reach agreement. Common areas of dispute include the valuation of variations, scope of latent conditions claims, and provisional sum adjustments
4. **Final certificate** — the superintendent issues a final certificate confirming the total amount payable, and the remaining retention is released

The contract typically specifies a timeframe for final account settlement (e.g., within 6 months of practical completion under AS 4000). Delays in settlement tie up retention funds and can create cash flow issues for the contractor.

Reference: AS 4000-1997 Clause 37.4 (final payment); AS 2124-1992 Clause 42.8
Applies to: All building classes and project types

Common pitfalls:
- Allowing the final account to drag on for years after practical completion, creating administrative burden and risk of lost documentation
- Not maintaining a running final account forecast during construction, leading to surprises when the final reconciliation is performed
- Failing to resolve disputed variations before attempting final account settlement, which can stall the entire process
- Not releasing retention in accordance with the contractual timeframes, which may constitute a breach of contract and trigger Security of Payment claims

## Cost Benchmarking

Cost benchmarking compares a project's costs against historical data from similar completed projects. It is a primary tool for validating cost plan estimates and identifying elements that are over- or under-costed relative to market norms.

Typical benchmark cost ranges in Australian construction (as of 2024-2025, excluding GST, fees, and land):

- Detached residential (Class 1a): $2,000-$4,500/m2 GFA
- Multi-residential apartments (Class 2): $3,500-$6,000/m2 GFA
- Commercial office (Class 5): $3,000-$5,500/m2 GFA
- Retail (Class 6): $2,500-$4,500/m2 GFA
- Industrial warehouse (Class 7/8): $1,200-$2,500/m2 GFA
- Hospital/health (Class 9a): $5,000-$9,000/m2 GFA
- School/education (Class 9b): $3,500-$6,000/m2 GFA

These ranges are broad because costs vary significantly based on location (Sydney and Melbourne typically 10-20% higher than regional areas), quality specification, site conditions, and market conditions at the time of tender. Benchmarks should be adjusted using regional cost factors and escalation indices to bring historical data to current terms.

Reference: AIQS Building Cost Index; Rawlinsons Australian Construction Handbook; RLB Riders Digest
Applies to: All building classes and project types

Common pitfalls:
- Using benchmark rates without adjusting for time (escalation), location (regional factors), and quality (specification level)
- Relying on published benchmark ranges without understanding the basis of measurement (GFA, NLA, GIFA definitions differ)
- Comparing costs between projects of different procurement routes (e.g., D&C vs traditional) without adjusting for the contractor's design risk premium
- Using a single benchmark project rather than a range, which can introduce project-specific bias

## Feasibility Stage Estimates

Feasibility estimates are produced at the earliest project stage, often before any design has been prepared. Their purpose is to establish whether the project is financially viable and what budget range the client should plan for. Accuracy at this stage is typically +/- 25-30%.

The standard methodology for feasibility estimates in Australian practice is the order-of-cost approach:

1. **Determine the gross floor area (GFA)** from the architect's massing study or site analysis
2. **Apply a $/m2 rate** from benchmark data for the building type, adjusted for location, quality, and current market conditions
3. **Add allowances** for external works (typically 5-10% of building cost), abnormal site costs (if known), and escalation to the anticipated construction start date
4. **Add project costs** including professional fees (typically 10-15% of construction cost), authority charges, FF&E, and client contingency
5. **State qualifications and exclusions clearly** — the estimate must declare what is included and excluded

The feasibility estimate forms the basis for the initial business case and funding approval. It is critical that the estimate is accompanied by clear qualifications, particularly around the assumed scope, quality level, and program.

Reference: AIQS Cost Management Practice Guide; Rawlinsons Australian Construction Handbook
Applies to: All building classes and project types

Common pitfalls:
- Presenting a feasibility estimate as a firm budget, creating unrealistic expectations about cost certainty at this early stage
- Using $/m2 rates for the wrong building type or specification level (e.g., applying Class 5 office rates to a Class 9a hospital)
- Not including professional fees, authority charges, and FF&E in the total project cost, understating the true financial commitment
- Failing to include escalation from the estimate date to the anticipated tender/construction date, which can be significant for projects with a 2-3 year design program

## Risk-Adjusted Budgets

A risk-adjusted budget moves beyond simple percentage contingency by quantifying individual risk items and their probable cost impact. This approach provides a more defensible and transparent contingency allowance that can be actively managed during the project.

The process for developing a risk-adjusted budget is:

1. **Identify risks** — catalogue all known cost risks from the risk register
2. **Estimate the cost impact** — for each risk, estimate the low, most likely, and high cost impact if the risk materialises
3. **Assess the probability** — assign a probability of occurrence to each risk (e.g., 10%, 50%, 80%)
4. **Calculate the expected value** — multiply the most likely cost impact by the probability for each risk. The sum of expected values provides a statistically-grounded contingency figure
5. **Perform sensitivity analysis** — identify the top 5-10 risks by expected value to focus management attention

For larger or more complex projects, a Monte Carlo simulation can be used to model the range of possible total project costs based on the combined probability distributions of all identified risks. This produces a P50 (50th percentile — the cost that has a 50% chance of not being exceeded) and P80 (80th percentile) estimate, which provides the client with a more informed basis for setting the budget.

Typical results from risk-adjusted budgets: the P50 contingency figure is usually 60-80% of what a traditional percentage-based contingency would suggest, but the P80 figure may be higher. This means traditional percentage contingency often over-allocates for expected outcomes but under-allocates for worst-case scenarios.

Reference: AIQS Cost Management Practice Guide; Project Risk Management (AS/NZS ISO 31000:2018)
Applies to: All building classes and project types, particularly recommended for projects over $10M or with complex risk profiles

Common pitfalls:
- Performing a risk assessment at the start of the project and never updating it, allowing the risk-adjusted budget to become stale
- Not linking the risk register to the cost plan, so the risk assessment and cost planning operate as separate disconnected exercises
- Setting the budget at the P50 level without explaining to the client that there is a 50% chance of exceeding this amount — clients often interpret the budget as a ceiling rather than a median
- Failing to track risk expenditure separately from scope change expenditure, making it impossible to assess whether the risk contingency was adequate

## Dispute Resolution Cost Implications

Construction disputes can arise from variation valuations, extension of time claims, latent conditions, defective work, or contract interpretation disagreements. The cost implications of disputes extend well beyond the disputed amount itself, and should be factored into cost reporting and forecasting.

The common dispute resolution mechanisms in Australian construction, in order of escalation, are:

1. **Negotiation** — direct discussion between the parties. No external cost, but consumes project team time.
2. **Adjudication** (under Security of Payment legislation) — a rapid statutory process (10-15 business days) for payment disputes. Adjudicator fees are typically $5,000-$20,000 per determination. The outcome is binding on an interim basis.
3. **Mediation** — facilitated negotiation with a neutral mediator. Costs are typically $10,000-$30,000 per day, shared between the parties. Each party also bears their own legal and expert costs.
4. **Expert determination** — a binding decision by an agreed expert. Similar costs to mediation but with a binding outcome.
5. **Arbitration** — a formal private hearing similar to litigation. Costs can range from $50,000 to $500,000+ depending on the complexity of the dispute, the number of hearing days, and expert witnesses required.
6. **Litigation** — court proceedings. The most expensive option, with costs often exceeding $100,000 for even straightforward construction disputes.

A rough rule of thumb is that the cost of pursuing or defending a construction dispute is typically 10-25% of the amount in dispute for adjudication and mediation, and 25-50% or more for arbitration and litigation. These costs are often not recoverable in full, even for the successful party.

Reference: Security of Payment legislation (varies by state); AS 4000-1997 Clause 42 (disputes); Commercial Arbitration Act 2010 (Uniform legislation)
Applies to: All building classes and project types

Common pitfalls:
- Not including dispute costs in the forecast when disputes are in progress or likely, understating the total project cost
- Escalating disputes to arbitration or litigation without first attempting negotiation and mediation, which are faster and cheaper
- Failing to maintain contemporaneous records during construction, which weakens the party's position in any dispute resolution process
- Not seeking legal advice early when a dispute is emerging, leading to procedural missteps that prejudice the outcome

## Retention and Defects Liability

Retention is an amount withheld from each progress claim as security for the contractor's obligation to complete the works and rectify defects. It is one of the most commonly mismanaged aspects of construction cost management.

Under standard Australian construction contracts:

- **Retention rate**: typically 5% of each progress claim, up to a maximum of 5% of the contract sum (some contracts cap at 2.5%)
- **First moiety release**: 50% of accumulated retention is released at practical completion
- **Second moiety release**: the remaining 50% is released at the expiry of the defects liability period (DLP), typically 12 months after practical completion

The defects liability period is the contractual window during which the contractor is obligated to rectify any defects in the works. Defects that emerge during the DLP must be notified to the contractor in writing, and the contractor has a reasonable time to rectify. If the contractor fails to rectify, the principal may have the work done by others and deduct the cost from the retention.

For cost management purposes, the QS should:
- Track cumulative retention withheld and forecast the timing of releases
- Maintain a defects register during the DLP, with estimated rectification costs
- Include retention release dates in the cashflow forecast
- Advise the client on the adequacy of retention relative to identified defects

Reference: AS 4000-1997 Clauses 37 and 35 (retention and defects liability); AS 2124-1992 Clauses 42 and 37
Applies to: All building classes and project types

Common pitfalls:
- Releasing retention at practical completion without maintaining a comprehensive defects list, leaving no security for defect rectification
- Not tracking the DLP expiry date, missing the contractual deadline for notifying defects to the contractor
- Failing to account for the cash flow impact of retention — 5% withheld over the life of the project represents a significant amount that affects the client's cash requirements
- Not recognising that under some Security of Payment legislation, retention can be the subject of a payment claim by the contractor if not released in accordance with the contract

---

## Decision catalog (HITL)

Cost-plan-only assumptions. Shared brief/finishes decisions live in finishes and archetype
seeds; these ids are emitted in cost plan drafts and do not need to appear in the PMP.

```decision-catalog
- id: contingency-band
  section: Budget assumptions
  label: Contingency band
  cost_only: true
  applies_to:
    archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
    classes: [residential, commercial]
  options:
    - { value: low_3_5, label: 3–5% (well-defined brief) }
    - { value: mid_5_10, label: 5–10% (typical residential) }
    - { value: high_10_15, label: 10–15% (sparse brief / early stage) }
  default_hint: mid_5_10
- id: pc-ps-treatment
  section: Budget assumptions
  label: PC / PS sum treatment
  cost_only: true
  applies_to:
    archetypes: [new-dwelling, renovation, multi-dwelling]
    classes: [residential]
  options:
    - { value: included_allowances, label: Included as provisional allowances }
    - { value: owner_supply, label: Owner-supply / exclude from builder sum }
    - { value: separate_schedule, label: Separate PC/PS schedule }
  default_hint: included_allowances
- id: inclusions-baseline
  section: Budget assumptions
  label: Inclusions baseline
  cost_only: true
  applies_to:
    archetypes: [new-dwelling, renovation, multi-dwelling, ancillary]
    classes: [residential]
  options:
    - { value: builder_basic, label: Builder basic inclusions }
    - { value: mid_spec, label: Mid-spec turnkey }
    - { value: high_spec, label: High-spec / custom }
  default_hint: mid_spec
```
