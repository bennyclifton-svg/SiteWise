---
tier: topic
seed_type: sustainability
loaded_by: "task subject (BASIX, NatHERS, energy efficiency, sustainability, thermal comfort, glazing U-value, SHGC, insulation R-value, hot water, PV, BASIX certificate, OC, energy compliance)"
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary, small-commercial]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
topics: [basix, nathers, sustainability, energy]
summary: "BASIX and NatHERS posture for NSW residential work: application workflow, thermal simulation, common compliance failure points, procurement discipline for committed items, and certificate amendment. The agent never predicts a certificate outcome."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Sustainability and energy guide — BASIX and NatHERS

This seed gives SiteWise its energy efficiency and sustainability posture for residential projects. Load it when the task signal is BASIX, NatHERS, energy efficiency, thermal performance, glazing commitments, insulation, hot-water systems, PV, sustainability compliance, or BASIX certificate workflow.

Authority stack reminder (per `../AGENTS.md §1`): the project's BASIX certificate, DA or CDC consent conditions, CC documentation, and BASIX final certificate are the source of truth. Where evidence is missing, label the point as Assumption per `../00-doctrine/doctrine.md §evidence-discipline`. BASIX assessment is a licensed activity — the online BASIX tool and any NatHERS assessor produce the certificate. The agent must not attempt to assess BASIX compliance or predict a certificate outcome.

## 1. BASIX overview (NSW)

**BASIX** (Building Sustainability Index) is the NSW Government's energy and water sustainability assessment scheme for residential buildings. It applies to:
- new dwellings (Class 1a and Class 2);
- alterations and additions where the current BASIX application threshold or
  project criteria are met (confirm in the official BASIX tool and approval
  pathway; this seed does not freeze a monetary threshold);
- swimming pools (volume threshold applies).

BASIX is administered through the NSW planning system and official BASIX online
tool. Confirm the current administering agency, portal and lodgement process
before use. The tool generates a **BASIX certificate** that is lodged with the
DA or CDC application. The certificate commits the project to specific
performance targets for water, energy, and thermal comfort.

**What BASIX certifies:**
- **Water score** — rainwater tank capacity/use, water-efficient fixtures (taps, showers, dual-flush toilets), pool/spa cover (where applicable).
- **Energy score** — hot water system type and capacity, lighting (LED vs. other), pool/spa pump efficiency, PV system (where committed).
- **Thermal comfort** — NatHERS star rating of the thermal envelope (glazing type and area by orientation, wall/ceiling/floor R-values, shading, ventilation).

**The BASIX certificate is a legal commitment.** Items committed in the certificate (window specifications, insulation R-values, HW system type, PV size) must be installed as specified. Substitution without updating the certificate is a compliance breach that can prevent OC.

**State callout — VIC:** Victoria does not use BASIX. VIC uses NatHERS + NCC Section J (and the Nationwide House Energy Rating Scheme star rating) as the residential energy compliance pathway. For a VIC project, all BASIX workflow in this seed is not applicable — confirm the energy compliance pathway with the certifier and energy assessor.

**State callout — other states (QLD, SA, WA, TAS, NT, ACT):** BASIX is a NSW-specific scheme. Other states use NatHERS + NCC Section J (Part 3.12 or 2.6) as the residential energy efficiency compliance pathway. The BASIX workflow described in this seed does not apply to non-NSW projects. Flag: "Confirm energy compliance pathway with the certifier — BASIX is not applicable in this state."

## 2. BASIX application workflow (NSW)

### Step 1 — Certificate lodgement

The applicant (owner, architect, or certifier acting for the owner) completes the BASIX online tool and downloads the BASIX certificate before DA or CDC lodgement. The certificate number must appear on the DA/CDC application form.

The BASIX tool requires input on:
- site address and climate zone;
- proposed floor area and orientation of the building;
- glazing: area by orientation, frame type, glass type (U-value and SHGC);
- insulation: ceiling, wall, and floor R-values;
- hot water system: type (solar, heat pump, gas continuous, gas storage, electric), capacity;
- lighting: LED fraction of fittings;
- water efficiency fixtures;
- rainwater tank (if proposed);
- PV system (if proposed).

### Step 2 — Conditions of consent

The DA or CDC consent issued by council (DA) or the private certifier (CDC) will include a BASIX condition requiring the works to be built in accordance with the BASIX certificate. This condition binds the project — it is not optional.

### Step 3 — CC documentation

When applying for the Construction Certificate, the certifier checks that the CC plans and specification are consistent with the BASIX certificate. Typical checks:
- Window schedule: glass type and frame thermal specifications match BASIX certificate values.
- Insulation schedule: specified R-values in ceiling, wall, and floor match certificate.
- Hot water system: type and model consistent with BASIX commitment.

If the CC documentation does not match the BASIX certificate, the certifier will require an amended BASIX certificate before issuing the CC.

### Step 4 — During construction

The builder and project lead must ensure that:
- **Insulation** is installed per the certificate specification before lining commences. The frame/insulation inspection is a hold point.
- **Windows** supplied match the glass type and frame specification in the certificate. Window substitution at procurement is the most common BASIX failure mode — see Section 4.
- **Hot water system** model and capacity match the certificate.
- **PV system** (if committed) is sized and connected as specified.

### Step 5 — BASIX final certificate (hold point before OC)

**The BASIX final certificate is required before the Occupation Certificate can be issued. This is a mandatory hold point.**

At practical completion / PC walk, the project lead must confirm:
- All BASIX-committed items are installed as specified.
- The builder (or relevant tradesperson) provides evidence: photos, product specifications, compliance paperwork.
- The certifier or BPA has signed off the relevant inspection stages (frame/insulation, final).

The project lead then assists the owner (or the owner's architect/certifier) to obtain the BASIX final certificate via the BASIX online tool.

**Link to Slice 13:** BASIX final certificate is a handover-critical item. At the PC walk, confirm the BASIX compliance pack is complete before the certifier issues the OC. See `handover-pc-dlp-defects.md` (Slice 13) for the full PC and handover sequence.

**Consequence of missing BASIX final certificate:** The certifier cannot issue an OC. The project cannot be legally occupied. This is not a minor defect — it blocks settlement if the project involves a sale.

## 3. NatHERS — thermal simulation method

**NatHERS** (Nationwide House Energy Rating Scheme) is the thermal energy simulation framework used across Australia to predict the heating and cooling energy demand of a dwelling.

In the context of BASIX (NSW):
- NatHERS simulation is one of the methods accepted by the BASIX online tool to demonstrate compliance with the thermal comfort target (instead of using the simplified BASIX assessment tool's own calculation).
- A NatHERS assessor uses accredited software (AccuRate Sustainability, BERSPro, FirstRate5, etc.) to model the building's thermal performance and generate a star rating.
- **6 stars** is the NCC 2022 minimum for a new Class 1 dwelling nationally. BASIX's thermal comfort target may require a higher effective performance in some NSW climate zones.

In non-NSW states (VIC, QLD, etc.):
- NatHERS + the NCC Section J energy provisions directly set the compliance pathway. The certifier requires either a NatHERS report from an accredited assessor, or a Section J DTS compliance statement.

**VIC-specific:** VIC requires a minimum 7-star NatHERS rating for new dwellings (effective from 1 May 2023 under the NCC 2022 adoption). This is a higher bar than the national 6-star minimum. For any VIC new-dwelling project, confirm the current NatHERS minimum with the energy assessor.

**The agent's role with NatHERS:** surface the need for a NatHERS assessor when the project evidence shows a glazing-heavy design, non-standard orientation, or a climate zone where thermal comfort is challenging. Do not attempt to estimate NatHERS star ratings from first principles.

## 4. Common BASIX compliance failure points

These are the failure modes the agent should surface proactively during design review and procurement — before they become OC blockers.

### 4.1 Window U-value and SHGC substitution

**What happens:** The BASIX certificate commits to specific window thermal performance values (U-value for heat loss, SHGC — Solar Heat Gain Coefficient — for solar gain control). During procurement, the window supplier substitutes a product that is cheaper but has worse thermal performance (higher U-value or SHGC). The substituted product no longer matches the BASIX certificate.

**Consequence:** The certifier cannot sign off the CC or the BPA frame inspection without confirmation that the windows meet the certificate specification. Either an amended BASIX certificate is required (which may require reducing glazing area or upgrading other elements) or the windows must be replaced.

**Agent trigger:** At procurement or when reviewing the window schedule, check: does the proposed glass type and frame construction match the U-value and SHGC values on the BASIX certificate? Flag any discrepancy.

### 4.2 Insulation R-value not installed as specified

**What happens:** The BASIX certificate commits to, for example, R3.5 ceiling batts. The builder's insulation subcontractor installs R2.5 (a cheaper product). At the frame/insulation inspection, the certifier or BPA notes the discrepancy.

**Consequence:** The batts must be replaced before lining can proceed. This causes programme delay and additional cost.

**Agent trigger:** At the frame inspection stage, confirm the insulation specification in the project evidence matches the BASIX certificate. Flag if the builder's procurement record does not specify the correct R-value product.

### 4.3 Hot water system substitution

**What happens:** The BASIX certificate commits to a heat pump or solar HW system. At procurement or during construction, the builder substitutes a cheaper electric storage system on the basis of upfront cost or lead time.

**Consequence:** Electric storage does not meet the BASIX energy score. An amended BASIX certificate may be required — which may not be achievable without adding other energy-saving measures. Alternatively, the correct system must be installed before OC.

**Agent trigger:** When reviewing PC procurement documentation or at the practical completion walk, confirm the hot water system type, brand, and model match the BASIX certificate. Flag substitutions immediately.

### 4.4 PV system not installed (when committed)

**What happens:** The BASIX certificate commits to a PV system of a specified size. The owner defers PV installation "until after handover" to reduce upfront cost. The BASIX certificate still requires it for OC.

**Consequence:** OC cannot be issued. The certifier will require the PV system to be installed and connected before issuing the BASIX final certificate.

**Agent trigger:** At the handover stage, confirm PV is installed and commissioned per the BASIX certificate. If the owner is considering deferring, flag the OC consequence immediately.

### 4.5 Ventilation provisions not met

**What happens:** The BASIX certificate commits to natural ventilation provisions (e.g. openable windows on both sides of a room for cross-ventilation) or mechanical ventilation (exhaust fans with specific capacity). Changes during design development reduce openable window area or relocate exhaust fans to non-compliant positions.

**Consequence:** The building may not meet the BASIX thermal comfort target. An amended certificate may be required or the design must be corrected.

**Agent trigger:** During design review, check that any changes to window operability, room layout, or exhaust fan specification are assessed against the BASIX certificate. Raise with the designer if openable window area is reduced.

### 4.6 Alterations and additions — existing vs new elements

**What happens:** For a renovation BASIX certificate, the tool assesses the whole-of-home performance including existing elements. A common failure is committing the existing hot water system or existing insulation to the certificate, then discovering the existing system is non-compliant or that the existing insulation is insufficient.

**Consequence:** The BASIX certificate is based on incorrect input. The certifier or council may require an amended certificate and the installation of compliant systems.

**Agent trigger:** On renovation projects with a BASIX certificate, confirm the certificate inputs accurately reflect existing conditions at the site — not assumed conditions.

## 5. BASIX and procurement discipline

BASIX compliance should be treated as a **procurement constraint**, not a post-construction sign-off. The project lead must:

1. At tender/quote stage: include BASIX certificate as a contract document and require the builder to acknowledge the certificate and commit to supply products matching the specification.
2. At procurement of windows, insulation, HW systems, and PV: require the builder to confirm product specifications match the certificate before ordering.
3. At frame inspection: physically verify insulation R-values against the certificate.
4. At practical completion walk: confirm all committed items (windows, insulation, HW, PV) are installed as specified. Do not sign off PC without BASIX compliance confirmed.

## 6. BASIX certificate amendment

If a BASIX-committed item must be changed (window specification, HW system type, insulation R-value), the process is:
1. Return to the BASIX online tool and amend the certificate inputs.
2. If the amended design still meets the BASIX targets, a new certificate is generated.
3. The amended certificate must be lodged with the certifier before the affected work proceeds.
4. If the amendment is significant, a CC amendment may also be required.

The agent should surface the BASIX amendment pathway when any change to a committed item is proposed — do not assume the change can proceed without amendment.

## Services by stage

> **AGENT REVIEWED FOR LOCAL USE (Codex, 27 September 2026); owner review deferred.** Stage ids
> match `architectural-services-guide.md` (SEED_MAP owner decision Q3). This
> covers the residential BASIX and NatHERS assessor; stages outside that role
> are usually excluded.

Each stage opens with an **RFP line** that code can render as the lead of a
grouped RFP item. Use only the stages inside the appointment boundary; price
the rest separately or leave them out.

### Inception, site and existing information [inception]

**RFP line:** Review the brief and the site, and confirm the BASIX pathway and
the energy, water and thermal comfort targets.

- Confirm whether BASIX applies and whether thermal comfort uses the NatHERS
  simulation or the DIY method.
- Identify site factors: orientation, overshadowing and the NatHERS climate
  zone.

### Concept and schematic design [concept]

**RFP line:** Advise on design moves that affect BASIX compliance before the
design is fixed.

- Early advice on orientation, glazing, shading, insulation and hot water and
  heating system choices.

### Pre-DA consultation and development application [development_application]

**RFP line:** Prepare the NatHERS assessment and BASIX certificate for the
development application or complying development certificate.

- NatHERS thermal simulation by an accredited assessor, and the BASIX
  certificate with the commitments stamped on the drawings.
- Respond to council or certifier queries on BASIX.

### Conditions of consent and design development [design_development]

**RFP line:** Check the developed design against the BASIX commitments and
amend the certificate when the design changes.

- Review changes to glazing, insulation, fixtures and systems against the
  certificate; issue an amended certificate where needed.

### Documentation and tender [documentation_tender]

**RFP line:** Confirm the construction certificate drawings and specification
match the BASIX certificate.

- Check the BASIX commitments appear on the construction certificate
  drawings and in the specification.

### Novation and post-contract design [post_contract_design]

**RFP line:** Usually excluded; price any post-contract BASIX amendments at
hourly rates.

- Reassess substitutions (windows, hot water, insulation) that affect the
  certificate.

### Construction and completion [construction_completion]

**RFP line:** Support the BASIX completion evidence the certifier needs before
the occupation certificate.

- Review evidence of installed BASIX commitments and support the BASIX
  completion receipt (verify the current requirement with the certifier).

### Throughout the appointment [coordination]

**RFP line:** Respond to design team queries on BASIX and NatHERS throughout
the appointment.

- Keep the current BASIX certificate number and revision in the approvals
  register.

## Coverage depth

| Topic | Depth |
|---|---|
| BASIX overview and purpose | Deep |
| BASIX NSW application workflow | Deep |
| BASIX final certificate as OC hold point | Deep |
| Common BASIX failure points | Deep |
| NatHERS overview and context | Moderate |
| BASIX certificate amendment process | Moderate |
| Non-NSW energy compliance pathways | Shallow (signpost + state callouts) |
| NatHERS simulation methodology | Minimal (not this seed's role) |

## Low-confidence flags

- **BASIX alteration threshold:** The dollar threshold for triggering BASIX on alterations/additions ($50,000 noted here) is subject to change. Confirm the current threshold with the BASIX tool or certifier for any renovation project.
- **BASIX tool updates:** The BASIX online tool's target levels and accepted methods are updated periodically. The certificate outcome for a given design may change between tool versions. Confirm any BASIX certificate was generated using the current tool version.
- **NatHERS star rating minima:** The NCC 2022 6-star minimum is the national floor. Individual states may set higher minimums (VIC 7-star). Confirm with the energy assessor for any non-NSW project.
- **Multi-dwelling BASIX:** Class 2 apartment buildings are subject to BASIX under a different pathway (BASIX for multi-dwelling). The workflow differs materially from Class 1a — this seed's Class 1a focus does not fully address Class 2 multi-dwelling compliance.
- **Swimming pool BASIX:** Pools above the volume threshold require a BASIX certificate. The pool cover, pump efficiency, and heating system are assessed. This seed does not cover pool BASIX in detail.

## See also

- `ncc-reference-guide.md` — NCC energy efficiency provisions, Class 1 vs Section J (task-loaded)
- `as-standards-reference.md` — AS 3500 hot water compliance, glazing standards (task-loaded)
- `mep-residential.md` — hot water systems, HW types and sizing, electrical, PV context (task-loaded)
- `new-dwelling-guide.md` — BASIX section in the new-dwelling lifecycle, BAL/bushfire glazing interaction
- `renovation-guide.md` — BASIX for alterations and additions
- `handover-pc-dlp-defects.md` — PC walk, OC requirements, BASIX final certificate at handover (Slice 13)
- `../00-doctrine/doctrine.md §evidence-discipline` — labelling BASIX commitments as confirmed vs assumed
- `../00-doctrine/doctrine.md §escalation-triggers` — when to route to certifier, energy assessor, or BASIX consultant
- `../AGENTS.md §1` — authority stack; BASIX certificate beats seed guidance
