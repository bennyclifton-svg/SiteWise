---
tier: topic
seed_type: trade
loaded_by: task-loaded
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
status: legacy-retained
reconciliation: "Retained as the domestic integrated-services overview; deep discipline scope lives in the mechanical, hydraulic, electrical, and ICT/AV/security guides."
topics: [mep, electrical, plumbing, ventilation]
summary: "Mechanical, electrical and plumbing literacy for residential work: hot water, gas, electrical, NBN and mechanical ventilation, with the inspection gates per service. Never a substitute for the licensed trade or hydraulic engineer."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Trade seed — MEP (residential)

This seed provides residential-grade mechanical, electrical, and plumbing context for Class 1a and Class 10 buildings. It covers domestic hot water, gas, electrical, communications, and mechanical ventilation. It does not replace the licensed plumber, electrician, or hydraulic engineer — it gives the agent and project lead sufficient literacy to scope services work, identify inspection gates, and flag when specialist input is required.

Load this seed task-loaded when a role × archetype task involves services RFIs, scope of works, specification items, or services inspection coordination.

---

## Scope boundary and deep-guide handoffs

This file is retained as a compact Class 1 domestic-services overview. It must
not become a second source of truth for full discipline appointments:

- load `mechanical-services-guide.md` for mechanical design, coordination and
  commissioning scope;
- load `hydraulic-services-guide.md` for water, sanitary drainage, hot water,
  gas, trade waste and hydraulic handover scope;
- load `electrical-services-guide.md` for supply, demand, distribution,
  lighting, metering, controls and electrical commissioning;
- load `ict-av-security-guide.md` for communications, structured cabling,
  security and integrated technology;
- load `sustainability-energy-guide.md` for BASIX/NatHERS commitments.

Where the integrated overview and a discipline guide differ, use project
evidence first and the deeper discipline guide second. Keep the issue TBC and
route it to the responsible designer or licensed trade rather than combining
incompatible assumptions.

---

## Hot water systems

### System types

| Type | Fuel / energy | Key characteristics |
|---|---|---|
| Electric storage | Grid electricity | Large tank (125–315 L), off-peak tariff eligible, cheap upfront, higher running cost |
| Gas storage | Natural gas or LPG | Faster recovery than electric storage, moderate cost |
| Gas continuous flow (instantaneous) | Natural gas or LPG | No tank — heats on demand; compact; requires minimum flow rate to activate |
| Heat pump | Electricity (refrigerant cycle) | Efficient (COP 3–4), requires adequate air volume around unit, louder than storage |
| Solar thermal with electric / gas boost | Solar + boost | Roof-mounted collectors; most cost-effective long-term in sunny climates; BASIX credit |
| Solar PV + heat pump | Grid / solar PV | Emerging pairing; grid-interactive tariffs |

**BASIX:** NSW BASIX requires thermal comfort and energy compliance. Hot water system selection contributes to the BASIX energy score. Changing the specified HW system type after BASIX approval requires a BASIX amendment — flag this risk before substitutions.

### Sizing

Rule of thumb for sizing electric and gas storage systems:
- 1–2 people: 80–125 L
- 3–4 people: 160–250 L
- 5+ people: 315 L or continuous flow

Continuous-flow systems are sized by flow rate (litres per minute) and simultaneous outlet demand, not storage.

### Compliance points

- **AS/NZS 3500.4** — Heated water services
- Tempering valves required for all new HW systems delivering to bathrooms and ensuites (max 50°C delivered temperature, AS/NZS 3500.4 §6)
- HW systems must be installed by a licensed plumber
- NSW Fair Trading registration required for installation of Type A gas appliances
- Expansion control valve (ECV) and pressure relief valve (PRV) mandatory on all storage systems

### Inspection gate

Hot water rough-in is inspected before wall and ceiling linings close. Evidence: licensed plumber's work completion certificate (plumbing compliance certificate in NSW).

---

## Gas

### System overview

Gas services in residential buildings cover:
- **Meter and supply main** — supplied by the gas network operator (Jemena in most of NSW); owner applies for connection
- **Internal distribution** — copper or CSST (corrugated stainless steel) pipework from meter to appliances
- **Appliances** — cooktop, oven, HW system, space heating, outdoor BBQ point

### Type A vs Type B appliances

- **Type A appliances** (domestic scale): HW systems, cooktops, ovens, space heaters. Installation and first-fix by a licensed plumber/gasfitter.
- **Type B appliances** (commercial scale, > 10 MJ/h input): Typically not relevant for Class 1a; flag if commercial cookers or heating appear in a residential brief.

### Sizing and pressure

Natural gas is supplied at low pressure (1.1 kPa at the meter). Internal pipework is sized to ensure adequate pressure at each appliance outlet. LPG is supplied from a cylinder/tank at higher pressure, regulated at the meter/regulator.

Pipe sizing is a licensed gasfitter/hydraulic engineer scope item. The agent should not size gas pipework; it can confirm that a gas sizing exercise is included in the services design.

### Key compliance

- **AS/NZS 5601.1** — Gas installations (Part 1: General)
- All gas work must be performed by a licensed gasfitter
- Clearances: appliances must meet minimum clearances from combustible materials (AS/NZS 5601.1 Section 4)
- Isolation: each appliance must have an accessible isolation cock
- Pressure testing: all internal pipework pressure-tested before concealment — evidence required before linings close

### LPG specific

For sites without natural gas supply, LPG is supplied via cylinder (single or dual) or above-ground tank. Tank size and location governed by AS/NZS 1596 and local council requirements. Safety exclusion zones apply around the tank.

---

## Electrical

### Switchboard and supply

Residential buildings are supplied at 230 V single-phase (most dwellings) or 400 V three-phase (larger homes, EV charging, large heat pumps). The supply agreement is with the DNSP (distribution network service provider — e.g. Ausgrid, Endeavour Energy in NSW).

Key switchboard elements:
- **Main switch** — isolates entire board
- **Safety switches (RCDs)** — Type 2 RCDs required on all power and lighting circuits in new domestic wiring under AS/NZS 3000 (Wiring Rules)
- **Circuit breakers (MCBs)** — over-current protection per circuit
- **Tariff metering** — smart meter now standard in NSW; separate off-peak circuit for HW if applicable

### Circuit layout for Class 1a

Typical circuit allocation:
- General power outlets (GPOs): 10 A circuits, typically 8–12 GPOs per circuit
- Lighting circuits: 10 A circuits per area
- Dedicated circuits: cooktop (20 A or 32 A), oven, HW system, air conditioning condensers, EV charger (if provided), pool/spa
- Wet area circuits: all GPOs in bathrooms, laundries, and outdoor areas must be RCD-protected

### Safety switches (RCDs)

AS/NZS 3000:2018 (Wiring Rules) requires RCD protection on all final sub-circuits in Class 1a dwellings. This is the key change from earlier editions. Retrofit or renovation work that extends existing circuits should verify RCD compliance — the agent must flag non-compliant boards as a risk rather than assuming compliance.

### Key compliance

- **AS/NZS 3000** — Wiring Rules (current edition)
- All electrical work must be performed by a licensed electrician
- NSW: work must be covered by a Certificate of Compliance — Electrical Work (CCEW) issued by the electrician
- Hot work (mains connection, metering) requires an authorised service provider

### EV charging provision

Growing expectation in new dwellings. A conduit stub from the switchboard to the garage, plus a spare 32 A circuit breaker position, is a low-cost provision. A full EV charger installation requires a dedicated circuit and potentially a supply upgrade. Flag if the brief does not address EV charging for new dwellings.

---

## NBN / communications

### Pit-and-pipe (conduit infrastructure)

For new Class 1a dwellings, NBN Co requires:
- A lead-in conduit from the street boundary to the first access point (typically in the garage or comms cupboard)
- Internal conduit routes from the access point to each room where outlets are planned
- A comms cabinet or wall plate location accessible for NBN equipment installation

This conduit work is typically done by the electrician or data cabler and must be in place before slabs are poured (underground conduit) and before internal linings close.

### Technology types

- **FTTP (Fibre to the Premises):** NBN Co fibre direct to the building — highest performance. Growing coverage in greenfield and some brownfield areas.
- **FTTN (Fibre to the Node):** Existing copper from street node to premises. Speed limited by copper length. Legacy technology in many suburban areas.
- **HFC (Hybrid Fibre-Coaxial):** Cable network; typically former Foxtel or Telstra cable areas.

For new dwellings, check NBN Co's address checker for technology type at the specific site. The conduit and cabling requirement is technology-agnostic.

### Internal cabling

- Co-ax (RG-6) cabling for pay TV / antenna distribution if required
- Cat 6 or Cat 6A for structured data cabling (future-proofing)
- All cabling to terminate at a central comms cabinet

---

## Mechanical ventilation

### Wet area exhaust

Building Code of Australia (NCC Volume Two) requires mechanical exhaust ventilation in:
- Bathrooms and ensuites (no operable window to outside, or as an alternative to operable window)
- WCs (water closets)
- Laundries (where not naturally ventilated)

Minimum exhaust rate: 25 L/s for bathrooms (NCC Volume Two, H6.3). Exhaust must discharge to outside — not into the roof cavity.

Common defect: exhaust fan ducted into the roof space rather than through the roof or ceiling plane to outside. This causes condensation and mould in the roof cavity. The agent should flag this as a non-compliant installation.

### Kitchen rangehood

Not mandated by NCC for Class 1a but standard practice. Two types:
- **Ducted:** Fan draws air through a filter and exhausts outside via a duct. Preferred for odour/moisture removal.
- **Recirculating:** Fan draws air through a carbon filter and returns air to the kitchen. No duct to outside required. Less effective for moisture.

Ducted rangehoods require an external penetration (wall or roof) and back-draught damper. Coordinate with external cladding/brick veneer trades.

### Whole-house ventilation

Not mandated for Class 1a in NSW under current NCC Volume Two, but increasing airtightness of new dwellings (driven by BASIX energy targets) is creating condensation and air quality risks. High-performance builds may specify:
- **HRV (Heat Recovery Ventilation):** Extracts stale air and recovers heat energy before exhausting; supplies fresh filtered air.
- **ERV (Energy Recovery Ventilation):** As HRV but also transfers moisture.

These are specialist mechanical systems. Flag if not specified on a tight-envelope build.

### Air conditioning

Residential AC is typically:
- **Split system:** Outdoor condenser + indoor head unit. Single room or zone. Most common.
- **Multi-split:** One outdoor condenser serving multiple indoor heads.
- **Ducted reverse cycle:** Central air handler with duct distribution throughout the house. Higher cost but whole-house coverage.

AC condenser location must be coordinated with the builder and structural trades (roof penetrations, platform, screening). Electrical load must be included in switchboard design. Refrigerant pipework chase must be allowed for in wall framing.

---

## Coverage depth

| Topic | Depth | Notes |
|---|---|---|
| Hot water systems and compliance | Moderate | BASIX interaction and tempering valve requirements covered; sizing is indicative |
| Gas installations | Moderate | Type A appliances and compliance framework covered; pipe sizing is licensed gasfitter scope |
| Electrical — switchboard and circuits | Moderate | Circuit types and RCD requirements covered; load calculations are licensed electrician scope |
| NBN / pit-and-pipe | Moderate | Infrastructure requirements covered; NBN Co design and technology-specific details vary by address |
| Wet area exhaust | Moderate | NCC requirement and common defect covered |
| Kitchen rangehood and whole-house ventilation | Shallow | Overview only; HRV/ERV design is specialist scope |
| Air conditioning | Shallow | System types noted; mechanical design and load calculation are specialist scope |

---

## Low-confidence flags

- **Three-phase supply requirements:** Trigger points (large heat pumps, EV fast chargers, large AC systems) are indicative. Actual supply capacity is a DNSP determination — do not advise on supply upgrades from general knowledge.
- **Gas network availability:** Natural gas is not available at all addresses. Verify network availability at the specific address before specifying gas appliances. SA and ACT have announced gas network phase-outs — flag for projects in those states.
- **BASIX / NatHERS interaction with MEP:** HW system type, insulation, and glazing choices interact to produce the BASIX score. Changes to MEP system types after BASIX approval require reassessment. Do not approve substitutions without flagging BASIX consequences.
- **Embedded networks (multi-dwelling):** Multi-dwelling developments may use an embedded electricity network (one meter point, internal metering per unit). This has different compliance, metering, and body corporate implications — flag for any multi-dwelling brief rather than treating it as standard residential electrical.
- **Solar PV and battery storage:** Rapidly evolving equipment and tariff landscape. Grid connection requirements and export limits vary by DNSP and change frequently. Do not prescribe system sizes or tariff arrangements from general knowledge.
