---
tier: archetype
seed_type: archetype
loaded_by: "archetype: small-commercial"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_classes: [commercial]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
confidence: reduced
status: legacy-retained
superseded_by: seed/commercial-construction-guide.md
summary: "Thin graceful-degradation coverage for occasional small commercial work (Class 5, 6, 7b and 8 fitouts and shells) at the edge of SiteWise's residential domain. Every output under this archetype carries confidence: reduced and routes material commercial scope to specialists."
doctrine_anchors: [§seed-consultation-discipline, §evidence-discipline, §register-discipline, §state-handling]
agents_anchors: [§1, §2, §3, §8, §11]
---

# Archetype seed — Small commercial (graceful degradation)

> **Confidence limit — read before proceeding.**
> SiteWise is a residential project management harness. When `archetype: small-commercial` is declared, the agent operates at the **edge of SiteWise's domain coverage**. This seed provides a thin fallback for a builder or owner occasionally doing a small Class 5, 6, 7b, or 8 fitout, office tenancy, or light-industrial shell — it does not replicate a commercial harness. Every phase-gate output produced under this archetype must carry `confidence: reduced` in its frontmatter. The project lead should engage a commercial project manager, commercial QS, or specialist certifier for any material commercial scope.

This seed is loaded when the project `README.md` declares `archetype: small-commercial`. It is role-neutral; the role overlay (`role-owner-builder.md`, `role-architect-pm.md`, `role-builder.md`, or `role-d-and-c.md`) sits on top.

NSW is the deep default. Non-NSW callouts appear inline. If `state:` is not NSW and the task turns on a state-specific commercial instrument, **flag the gap** — do not extend NSW commercial guidance to another jurisdiction.

## Scope of this archetype

This seed covers (at reduced confidence):

- fitout of an existing tenancy in a Class 5 (office), Class 6 (shop/retail), Class 7b (storage/warehouse), or Class 8 (factory/industrial) building;
- small light-industrial or commercial shell where the builder's principal practice is residential;
- work where the builder is licensed for the relevant class of commercial building work (see licence check below — this must be confirmed before proceeding).

This seed does not cover:

- Class 9 buildings (public assembly, health, aged care, education, detention) — flag as exceeding SiteWise scope and require specialist engagement;
- large-floorplate commercial development, multi-tenancy commercial buildings, high-rise commercial, or any commercial work that by scale or complexity exceeds what a residential builder occasionally encounters — flag and recommend the commercial harness;
- Class 2–4 residential building work classified under the residential archetypes (`new-dwelling-guide.md`, `renovation-guide.md`, `multi-dwelling-guide.md`, `ancillary-guide.md`) — load the matching residential seed;
- mixed-use development where the commercial component is material in its own right — treat as exceeding SiteWise scope.

## Licence check — hard stop

A NSW contractor licence for residential building work does **not** automatically cover commercial building work. A builder whose licence endorsement covers only residential building work cannot lawfully perform commercial building work. This is a **hard stop before any work proceeds.**

The project lead must confirm:

- The builder holds a NSW contractor licence with an endorsement covering the proposed commercial work (e.g. general building work, or the applicable commercial endorsement under the NSW building licencing framework).
- **Assumption: confirm the current NSW Fair Trading contractor licence class structure and applicable endorsements for the proposed work before proceeding.** Commercial and residential licencing requirements differ and are subject to change.

If the licence class cannot be confirmed, the agent must flag this as a blocker and stop phase-gate work. Do not proceed on the assumption that a residential licence is sufficient.

**Non-NSW:** Each state and territory has its own contractor licencing framework for commercial building work. Flag gap and confirm with the relevant state authority before proceeding.

## NCC class — Volume One applies

Commercial buildings (Class 5–9) are governed by **NCC Volume One** (Building Code of Australia), not Volume Two (Housing Provisions). This is a fundamental shift from SiteWise's residential context:

| Feature | Residential (Volume Two) | Small commercial (Volume One) |
|---|---|---|
| NCC volume | Volume Two | Volume One |
| Energy compliance | BASIX (NSW) or NatHERS | Section J (Commercial Energy Efficiency) — BASIX does not apply |
| Fire separation | Volume Two fire separation for Class 1 | Volume One FRL, fire compartmentation, and egress requirements |
| Accessibility | Volume Two accessible housing provisions (if applicable) | AS 1428 series — access to premises standard; DDA compliance |
| Certifier accreditation | Residential certifier | Must be accredited / licensed to certify commercial work |
| Inspection regime | Residential inspection schedule | Commercial inspection schedule per the CC |

**Implications for the project lead:**

- **BASIX does not apply to commercial work.** A common error when a residential builder first does commercial work is treating energy compliance as a BASIX exercise. Section J requires modelling or DTS compliance under NCC Volume One, typically prepared by a mechanical or façade engineer. Flag this if the project lead or designer has not addressed it.
- **Section J** covers building fabric, HVAC, lighting, hot water, and on-site energy generation. DTS and modelling (JV3 simulation) pathways exist. This is a commercial specialist domain — SiteWise does not carry deep Section J guidance. **Assumption — confirm Section J applicability and pathway with the certifier and relevant engineer.**
- **AS 1428 accessibility** — fitout work in a Class 5 or Class 6 building may trigger access upgrade obligations (Part D3 of NCC Volume One). Confirm with the certifier whether access compliance is triggered by the fitout scope or the accumulated works threshold.
- **BCA compliance report** — a commercial certifier submission typically requires a formal BCA compliance report from a fire / BCA consultant. If this is not in the project's consultant scope, flag it as a gap.

## Planning pathway (NSW)

Fitout of an existing approved commercial tenancy may fall into one of three NSW planning categories:

| Pathway | When applies |
|---|---|
| Exempt development | Minor internal fitout meeting the SEPP (Exempt and Complying Development Codes) 2008 exempt provisions — no change of use, low fire load, limited scope. **Assumption: confirm criteria against current SEPP.** |
| Complying Development Certificate (CDC) | Fitout meeting CDC complying provisions under the SEPP if applicable (certain commercial and industrial fitout classes). **Assumption: confirm applicability.** |
| Development Application (DA) | Change of use, significant fitout, or where CDC/exempt provisions are not met — typical for most material fitouts |

A change of building use (e.g. converting a warehouse to a restaurant, or a storage facility to a retail shop) is almost always a DA. Even an internal fitout without change of use can require a DA if it exceeds the exempt/CDC thresholds.

Confirm the planning pathway with a town planner or the certifier before design proceeds. Do not assume exempt status.

## Construction Certificate

The CC for commercial work is issued by a commercially accredited certifier. The submission set typically includes:

- architectural drawings (plans, elevations, sections, finishes schedule);
- structural / mechanical / electrical / hydraulic engineering drawings and specifications;
- BCA compliance report (fire, accessibility, energy — Section J);
- specification.

This is materially more complex than a residential CC. Allow for the additional consultant scope in the programme and fee budget. **Assumption — the specific submission requirements vary by certifier and council.**

## HBCF / HOW — does not apply

The HBCF (Home Building Compensation Fund) is a residential instrument. It does not apply to commercial building work. The project lead must not treat the absence of HBCF as meaning no risk or statutory obligation — commercial contracts carry their own risk-transfer and insurance requirements (public liability, contract works, professional indemnity). Confirm the insurance suite with the builder and the contract.

## Programme — indicative commercial fitout shape

A small commercial fitout is typically condensed compared to a full residential build. Residential cycle-time benchmarks from `program-scheduling-guide.md` do not directly apply.

Indicative durations for a small commercial fitout (50–200 sqm) — **confidence: reduced:**

| Phase | Indicative duration |
|---|---|
| Design and documentation (DD to CC ready) | 4–10 weeks |
| CC application and issue (commercial certifier) | 2–6 weeks |
| Fitout construction (demolition, services, partition, ceiling, finishes) | 6–16 weeks |
| Defects and OC | 2–4 weeks |

Complex fitouts (base building integration, major services work, heritage fabric, access upgrades) may be materially longer. These durations are indicative only — baseline the project programme against the specific scope.

## Common failure modes — small commercial archetype

- **Licence not confirmed** — residential builder commences commercial work without confirming the licence endorsement; statutory breach, possible site shut.
- **BASIX applied to commercial work** — BASIX certificate commissioned for a Class 5 fitout; certifier will not accept it; Section J work required; delay and additional consultant cost.
- **Certifier not commercially accredited** — residential certifier engaged for commercial CC; certifier discovers they cannot certify; replacement certifier required mid-process.
- **BCA report not scoped** — fire / BCA consultant not engaged; certifier withholds CC; delay and additional cost.
- **Accessibility upgrade obligation triggered unexpectedly** — accumulated works threshold met; access upgrade required; not in the budget or programme.
- **Change of use not recognised** — fitout proceeds under an assumed exempt-development position; council enforcement.
- **Section J not addressed** — energy analysis required but not in designer's scope; CC withheld.

## Agent behaviour under this archetype

When `archetype: small-commercial` is declared:

1. The agent loads this seed and the matching `user_role:` overlay. Both carry **reduced confidence** for commercial-specific content.
2. The agent confirms the NCC class of the proposed work (Class 5, 6, 7b, or 8) and states that NCC Volume One applies — not Volume Two.
3. The agent performs the licence check prompt immediately: "Has the builder confirmed that their contractor licence covers the NCC class of work proposed? This must be confirmed before any phase-gate work proceeds."
4. Every phase-gate output produced under this archetype carries `confidence: reduced` in its frontmatter.
5. The agent flags commercial-specific gaps as `Assumption — confirm with specialist` rather than presenting them as fact.
6. The agent does not refuse to proceed, but states the confidence limit explicitly in each output and recommends specialist engagement for material commercial scope.
7. If the scope indicates Class 9 work, large-scale commercial development, or scope clearly exceeding what a residential builder occasionally encounters, the agent flags this as exceeding SiteWise scope and stops phase-gate work until the project lead confirms or redirects.
8. The agent records this seed in `seed_consulted:` for every phase-gate deliverable.

## Non-NSW — reduced-confidence callout

State-specific commercial planning, licencing, and certification frameworks differ from NSW. This seed carries no deep state-specific commercial guidance. If `state:` is not NSW:

- Flag the licencing framework gap — confirm with the state licencing authority.
- Flag the planning pathway gap — confirm with a town planner or state planning portal.
- Do not extend NSW-specific instrument names (SEPP, EP&A Act, NSW Fair Trading) to other jurisdictions.
- The NCC class structure and Volume One applicability are national — this holds across jurisdictions.

## See also

- `../00-doctrine/doctrine.md §seed-consultation-discipline` — why this seed loads and the confidence-limit framework
- `../00-doctrine/doctrine.md §state-handling` — non-NSW callouts and gap-flagging
- `../AGENTS.md §1` (authority stack), `§2` (declaration gate), `§8` (state handling)
- `role-owner-builder.md` / `role-architect-pm.md` / `role-builder.md` / `role-d-and-c.md` — role overlays loaded alongside this archetype (all carry residential depth; commercial-specific obligations must be confirmed with specialists)
- `setup-and-commission-guide.md` — mobilisation workflow (residential depth — confirm commercial-specific obligations separately)
- `contract-administration-guide.md` — contract clause coverage (covers AS 4000 / AS 4902 which applies to some commercial work; confirm applicability)
- `cost-management-principles.md` — for commercial work with a QS, AIQS elemental cost plan format applies (not HIA Schedule of Allowances)
- `program-scheduling-guide.md` — residential benchmarks; commercial benchmarks require separate verification
- `../02-skills/atomic/seed-targeted-read.md` — the gate that loads this seed
