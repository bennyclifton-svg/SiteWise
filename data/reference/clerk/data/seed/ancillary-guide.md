---
tier: archetype
seed_type: archetype
loaded_by: "archetype: ancillary"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_classes: [residential]
applies_to_work_types: [extend]
state_default: NSW
status: legacy-retained
superseded_by: seed/residential-construction-guide.md
summary: "Archetype coverage for secondary structures on an existing residential lot — granny flats, studios, garages, pool houses, sheds. Covers the NCC class fork, the NSW CDC Affordable Housing pathway, BASIX treatment, statutory instruments, and the sequencing risks specific to ancillary scope."
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §state-handling]
agents_anchors: [§1, §2, §3, §7, §8, §11]
---

# Archetype seed — Ancillary

An **ancillary** project delivers a secondary structure on an existing residential lot: a granny flat / secondary dwelling, a detached studio, a garage, a pool house, a shed, or a combination of these. The primary dwelling is not the work — either it already exists (ancillary addition) or a new primary dwelling is the primary scope and the ancillary structure is task-loaded alongside `new-dwelling-guide.md`.

This seed is loaded when the project `README.md` declares `archetype: ancillary`. It is role-neutral; the role overlay (`role-owner-builder.md`, `role-architect-pm.md`, `role-builder.md`, or `role-d-and-c.md`) sits on top.

NSW is the deep default. Non-NSW callouts appear inline where the instrument or pathway differs materially. If `state:` is not NSW and the task turns on a state-specific instrument for which no callout exists, **flag the gap** rather than extending NSW guidance.

## Scope of this archetype

This seed covers:

- standalone granny flat / secondary dwelling as the primary project scope;
- secondary dwelling added to an existing primary dwelling (task-loaded alongside `renovation-guide.md`);
- granny flat scope within a new-dwelling project (task-loaded alongside `new-dwelling-guide.md` per AGENTS.md §7);
- detached studios, pool houses, carports, and garages where these are Class 10a;
- sheds and similar structures where the NCC class is Class 10a or 10b.

This seed does not cover:

- a single detached Class 1a dwelling as the primary scope (load `new-dwelling-guide.md`);
- a renovation or addition where the existing primary dwelling is materially affected (load `renovation-guide.md`, task-load this seed if the ancillary structure is also in scope);
- two or more dwellings on one title as a development product (load `multi-dwelling-guide.md`);
- Class 5–8 commercial structures on a residential lot (flag as out-of-SiteWise scope and load `small-commercial-guide.md` for graceful degradation).

## NCC class fork — do this first

The NCC class of the ancillary structure determines the NCC volume, BASIX applicability, certifier submission set, and inspection regime.

| Structure | NCC class | NCC volume | BASIX | OC required |
|---|---|---|---|---|
| Secondary dwelling (granny flat) — habitable | Class 1a | Volume Two | Yes (cost threshold applies — see below) | Yes |
| Detached studio (sleepout, art studio, home office — not a separate dwelling) | Class 10a | Volume Two Part 3.7 / BCA Spec | No | No — unless required by certifier |
| Garage, carport, pool house, shed | Class 10a or 10b | Volume Two Part 3.7 / BCA Spec | No | No — unless required |
| Swimming pool | Class 10b | Volume Two, pool fencing under AS 1926 | No | No — separate pool fence certificate |

**Classification test for secondary dwelling vs Class 10a studio:** a secondary dwelling is a self-contained dwelling with sleeping, kitchen, and bathroom facilities. A studio or room that lacks kitchen facilities is typically Class 10a. The distinction matters: Class 1a requires full Class 1 compliance, BASIX (above threshold), and an OC. Class 10a is simpler. Do not assume — confirm with the certifier at design stage.

**Non-NSW:** The Class 1a / 10a fork is national (NCC is national), but BASIX is a NSW instrument only. Other states use NatHERS-based energy compliance or state equivalents. See non-NSW callout below.

## Planning pathway — CDC Affordable Housing (NSW default)

The **NSW Housing SEPP 2021** (State Environmental Planning Policy (Housing) 2021 — successor to the Affordable Rental Housing SEPP 2009) permits secondary dwellings on residential lots as complying development without a DA, subject to lot and design controls being met.

### CDC eligibility gate

Before assuming the CDC pathway, the project lead must test all of the following. If any criterion is not met, the DA pathway applies.

| Control | NSW Housing SEPP general position | Source |
|---|---|---|
| Zone | Zone must permit residential development and the lot must contain a principal dwelling (or one is being built concurrently) | Housing SEPP Part 2, Div 1 |
| Minimum lot size | Lot must meet the minimum lot size required by the applicable LEP or the Housing SEPP default. Confirm the current SEPP and LEP provisions — **Assumption: confirm against current Housing SEPP and local LEP before relying on a specific area figure** | Housing SEPP / LEP |
| Secondary dwelling floor area cap | Detached secondary dwelling: maximum 60 sqm gross floor area. Attached secondary dwelling: lesser of 60 sqm or the principal dwelling floor area. **Assumption: confirm against current SEPP wording** | Housing SEPP |
| Height | Maximum height typically 8.5 m, but confirm against LEP height limit and Housing SEPP override | Housing SEPP |
| Setbacks | Rear, side, and street setbacks as prescribed by the Housing SEPP or LEP (whichever is more permissive). Confirm numerically — **Assumption: confirm against current provisions** | Housing SEPP / LEP |
| Car parking | Generally one off-street space per secondary dwelling, where car parking is required by the LEP | Housing SEPP |
| Bushfire / flood / heritage | CDC pathway under Housing SEPP is unavailable if the lot is mapped as high-risk bushfire prone, in a heritage conservation area, or flood-affected above the LEP threshold. DA required | EP&A Act / SEPP |

**All Housing SEPP numerical thresholds above are labelled Assumption** — they reflect the SEPP as understood at authoring. Confirm against the current Housing SEPP 2021 instrument and the applicable council LEP before presenting to the project lead as facts. SEPP controls are subject to policy amendment.

### CDC pathway (fast track)

Where all CDC eligibility criteria are met:

1. Designer produces DA / CDC documentation set: site plan, floor plans, elevations, sections, shadow diagrams (if required), Statement of Environmental Effects (SEE) or CDC checklist, BASIX certificate (if applicable).
2. CDC application lodged with an accredited certifier (private certifier or council).
3. Certifier issues CDC within **7–20 business days** (approximate — varies by certifier workload and completeness of submission). No council assessment period.
4. CDC subsumes the Construction Certificate for the complying development scope — the certifier issues the combined CDC/CC instrument or issues the CC as part of the CDC set.
5. Construction commences; mandatory inspections as specified in the CDC.
6. OC issued by certifier on completion (Class 1a secondary dwelling).

### DA pathway (fallback)

Where the site or design does not satisfy Housing SEPP CDC controls (or where heritage / bushfire / flood / high flood risk applies):

1. DA lodged with council.
2. Council assessment: **8–24 weeks** typical (varies by council and complexity).
3. DA issued with conditions of consent; CC then lodged and issued by certifier.
4. Construction and OC as for CDC pathway, plus compliance with conditions of consent.

The project lead must record the pathway choice as a §decision-discipline entry with the basis (CDC eligibility test outcome). If switching from CDC to DA mid-project, the programme and cost impact must be registered immediately.

### Non-NSW — planning pathway callout

- **VIC:** No direct equivalent to the NSW Housing SEPP CDC pathway for secondary dwellings. Secondary dwelling approvals in VIC are typically via planning permit (DA equivalent) unless exempt under the VPP (Victorian Planning Provisions) residential zones. Granny flat / dependent person's unit provisions exist under the VPP but are not a CDC equivalent. Flag gap — confirm with a local town planner.
- **QLD:** Secondary dwellings may be permissible without approval under the Planning Regulation 2017 exempt development provisions in some zones. Confirm applicable zone and planning scheme. Flag gap.
- **SA:** Ancillary accommodation units under the SA Planning and Design Code; check code assessment pathway vs deemed-to-satisfy. Flag gap.
- **WA / TAS / NT / ACT:** Confirm the state/territory planning framework for secondary dwelling approval. Flag gap.

## Lifecycle — typical NSW CDC granny flat sequence

| Phase | Typical NSW duration | Key gate |
|---|---|---|
| Site due diligence (CDC eligibility test) | 1–2 weeks | Lot size, setback, floor area cap confirmed; heritage / flood / bushfire test complete |
| Design and CDC/DA documentation | 3–8 weeks | Scheme endorsed against brief and budget; BASIX certificate (if applicable) |
| CDC application lodgement and issue | 1–4 weeks (CDC) or 8–24 weeks (DA) | CDC or DA issued; CC instrument available |
| Builder procurement | 2–6 weeks (informal) or 4–10 weeks (formal) | Builder selected; head contract executed; HBCF checked against threshold |
| Pre-construction mobilisation | 1–2 weeks | Site possession, management plans |
| Site preparation, services, formwork | 1–2 weeks | Site cleared, set-out, services layout |
| Slab / footing | 1–2 weeks | Pre-pour inspection gate |
| Frame | 2–4 weeks | Frame inspection sign-off |
| Roof and lockup | 1–3 weeks | External envelope sealed |
| Fit-out (services rough-in, plasterboard, joinery, tiling) | 3–6 weeks | Internal lining and trades |
| Finishing, commissioning, and OC application | 2–4 weeks | BASIX final (if applicable); OC issued |
| DLP | Per contract | Defects close-out |

Typical total site duration for a detached 60 sqm CDC secondary dwelling on a benign urban site: **12–18 weeks**. Combined with pre-construction: **6–10 months** end to end. Significantly shorter than a full new dwelling because scope is constrained and certifier turnaround is faster.

## Site due diligence for ancillary scope

The ancillary due diligence is narrower than a full new-dwelling check but must still establish:

- **CDC eligibility test** — lot size, floor area cap, setbacks, zone, bushfire/flood/heritage. Test all criteria before design begins.
- **Survey** — title boundary, relevant existing structures, setback confirmation (the secondary dwelling setback is often different from the primary dwelling setback). Confirm easements that may affect footprint.
- **Soil / geotechnical** — on fill sites or sites with documented reactive soil, a soil test before slab design is prudent. For a small concrete slab, the cost of a soil test is low relative to the risk of a slab redesign post-pour.
- **Sewer and stormwater** — the secondary dwelling requires its own or shared connection. Confirm sewer riser availability and Sydney Water position on shared connections. Stormwater discharge point.
- **Services** — water, sewer, electricity, NBN for the secondary dwelling. Shared metering or separate metering (has tenancy / rental implications). Gas supply if in scope.
- **Access** — if the secondary dwelling will be tenanted, pedestrian access may need to be separate from the primary dwelling. Flag if access design affects the footprint or siting.
- **Heritage / character conservation area** — CDC pathway is blocked. DA required. Heritage impact statement may be needed.

## BASIX — ancillary structures

**Class 1a secondary dwelling:**

- BASIX applies to a Class 1a secondary dwelling when the estimated cost of the works is above the NSW BASIX threshold for alterations and additions. **Assumption: confirm current NSW BASIX threshold — the commonly cited figure is $50,000 but confirm against current BASIX regulations before relying on it.**
- If the secondary dwelling is a standalone new structure (not an addition to the existing house), BASIX is triggered in the same way as a new dwelling — not the alteration/addition threshold. Confirm with the BASIX assessor which threshold applies to a new detached secondary dwelling.
- BASIX commitments for a granny flat are proportionately simpler than a full dwelling (smaller floor area, fewer systems) but the same commitment-discipline applies: commitments are locked at CDC/DA, locked again at CC, and evidenced at OC.

**Class 10a structures:**

- BASIX does not apply to Class 10a garages, carports, sheds, pool houses, or studios.

**Non-NSW:** BASIX is a NSW instrument only. Other states use NatHERS-based energy compliance or state equivalents under NCC Volume Two. Flag gap for non-NSW ancillary Class 1a work.

## Statutory instruments — ancillary scope

### HBCF / HOW

The HBCF (Home Building Compensation Fund, formerly HOW) threshold applies to residential building work performed by a licensed builder. The HBCF threshold in NSW is typically triggered at $20,000. A granny flat contract frequently sits above this threshold.

- **Below threshold (e.g. a small shed or minor works under $20,000):** HBCF is not required. The work is still licensed work.
- **Above threshold:** HBCF certificate required before any deposit is taken and before the contract is signed. Same discipline as a full new dwelling — see `role-builder.md` for the builder-side workflow.
- **Confirm the current NSW threshold** — the $20,000 figure is the commonly cited legislative threshold. Confirm against current NSW Fair Trading guidance. **Assumption.**

### Owner-builder permit

An owner-builder permit is required where the owner proposes to supervise or carry out owner-builder work and the reasonable market cost (including labour and materials) exceeds the NSW owner-builder permit threshold (commonly $10,000 for owner-builder permit). A granny flat at $100,000–$200,000 construction cost is well above this threshold. See `role-owner-builder.md` for the permit process.

### Occupation Certificate

A Class 1a secondary dwelling requires an OC issued by the certifier before occupation. Class 10a structures do not require an OC unless the certifier specifies one.

## Structural posture — ancillary structures

A small secondary dwelling or ancillary structure typically involves:

- **Slab** — waffle slab or conventional slab per AS 2870 site classification. The same soil class assessment applies as for a full new dwelling — reactive soil (H1/H2/E) can require piered or stiffened construction even for a small slab.
- **Frame** — lightweight timber or steel frame per AS 1684 / AS 4100. Some granny flats are brick veneer; confirm structural design with the designer.
- **Roof** — light timber or steel truss; metal sheet or tile.
- **Wind classification** — AS 4055 applies (same as new dwelling). Coastal or exposed sites may be N3+.

Structural inspection gates for an ancillary Class 1a structure follow the same pattern as a new dwelling (pre-pour, slab conformance, frame, wet-area waterproofing if applicable, final/OC). The inspection schedule will be specified in the CDC or CC; do not assume a lighter inspection set simply because the structure is small.

## Sequencing risks specific to ancillary

- **CDC eligibility not tested before design commission** — designer proceeds on assumed CDC eligibility; lot size or setback failure requires DA pathway; programme moves out 8–24 weeks.
- **Floor area cap crept** — design exceeds 60 sqm by minor additions (porch, covered carport adjacent); CDC eligibility lost.
- **Sewer connection not confirmed** — Sydney Water or local council will not connect a second dwelling without specific approval; a blocked sewer connection can hold up OC.
- **Separate metering decision deferred** — decision about separate vs shared electricity/water metering affects the services design and tenancy arrangements; resolving it after construction is costly.
- **BASIX threshold miscategorised** — works treated as below BASIX threshold when they are not; BASIX certificate required at CDC/DA application and not available.
- **OC not triggered** — owner treats the secondary dwelling as a "done" structure before OC is issued; occupation before OC creates statutory exposure.

## Common failure modes — ancillary archetype

- **CDC eligibility assumed, not tested** — site in heritage conservation area; bushfire mapped; lot size below minimum; CDC pathway blocked; DA required but no time budgeted.
- **Class 10a / Class 1a confusion** — project lead builds a habitable secondary dwelling under a Class 10a shed approval; OC cannot issue; dwelling cannot be legally tenanted.
- **HBCF not checked against contract sum** — contract sum above threshold; HBCF not issued before deposit; statutory breach by builder.
- **Setback variance needed** — secondary dwelling siting revised late to avoid tree or easement; revised siting fails setback; CDC amendment required; programme slips.
- **Slab design driven by assumed soil class** — no soil test; slab constructed to M-class standard on H2 site; remedial substructure work required.
- **Conditions of consent not tracked** — DA pathway; conditions include landscaping, architectural finish, or fencing works required before OC; conditions discovered at PC walk.

## Agent behaviour under this archetype

When `archetype: ancillary` is declared:

1. The agent loads this seed and the matching `user_role:` overlay on any phase-gate task.
2. The agent confirms the NCC class fork (Class 1a secondary dwelling vs Class 10a ancillary structure) before proceeding to compliance, BASIX, or inspection guidance.
3. The agent tests the CDC eligibility gate before assuming the CDC pathway. If any criterion is unconfirmed, the agent flags it as an Assumption and asks the project lead to confirm.
4. The agent applies HBCF threshold checking when `user_role: builder` and records this in the evidence register.
5. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.
6. The agent flags all Housing SEPP numerical thresholds as Assumptions requiring confirmation against the current instrument.
7. If `state:` is not NSW, the agent flags the planning-pathway gap and does not extend NSW Housing SEPP guidance to another jurisdiction.

## See also

- `../00-doctrine/doctrine.md` — abstract project lead doctrine
- `../00-doctrine/doctrine.md §seed-consultation-discipline` — why this seed loads
- `../00-doctrine/doctrine.md §state-handling` — non-NSW callouts and gap-flagging
- `../AGENTS.md §1` (authority stack), `§2` (declaration gate), `§7` (cross-archetype tasks), `§8` (state handling), `§11` (active-project boundary)
- `role-owner-builder.md` / `role-architect-pm.md` / `role-builder.md` / `role-d-and-c.md` — role overlays loaded alongside this archetype
- `new-dwelling-guide.md` — task-load when new primary dwelling + ancillary structure are concurrent scope
- `renovation-guide.md` — task-load when existing primary dwelling is materially affected
- `setup-and-commission-guide.md` — mobilisation workflow per role
- `contract-administration-guide.md` — head contract clause coverage
- `program-scheduling-guide.md` — residential cycle-time benchmarks
- `cost-management-principles.md` — ancillary contingency band (5–7%) and cost plan context
- `ncc-reference-guide.md` — NCC Class 1a / 10a reference (task-loaded)
- `as-standards-reference.md` — AS 2870, AS 1684, AS 4055, AS 3959, AS 3500 (task-loaded)
- `sustainability-energy-guide.md` — BASIX/NatHERS guidance, common compliance failure points (task-loaded)
- `../02-skills/atomic/seed-targeted-read.md` — the gate that loads this seed
