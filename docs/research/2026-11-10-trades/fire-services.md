# Trades research: fire and services group

Date: 10 November 2026. Slice T3 of `docs/plans/2026-11-10-trades-catalogue.md`.
Status: draft for owner review. Nothing here is verified against an
instrument: every Australian Standard, NCC part and licence below is
written as the seed names it and is `verified: false`.

Trades: `trade.passive_fire`, `trade.gas_fitting`, `trade.solar_pv`,
`trade.data_security`, `trade.lifts`.

RFT files written:

- `knowledge/rfp/trade-passive-fire.yaml`
- `knowledge/rfp/trade-gas-fitting.yaml`
- `knowledge/rfp/trade-solar-pv.yaml`
- `knowledge/rfp/trade-data-security.yaml`
- `knowledge/rfp/trade-lifts.yaml`

Boundaries were drawn against the existing `trade-fire-services.yaml`,
`trade-electrical.yaml`, `trade-hydraulic-plumbing.yaml` and
`trade-mechanical.yaml`. Two overlaps remain in those files and need an
owner call (see "Overlaps with existing RFTs" at the end).

---

## trade.passive_fire — Fire stopping and passive fire

**Summary.** Seals every service penetration and joint in fire-rated
construction with a tested system, protects the elements the documents
give it, and hands over a located penetration register.

**House vs commercial.** Commercial first (Class 2 to 9, where
sole-occupancy units, shafts and compartments are fire-rated). On a
Class 1 house the only fire-rated construction is usually a party wall
or a wall near a boundary, sealed by the head contractor or plasterer;
a separate passive fire package is rare.

**Interfaces.**

- Takes from: `trade.plasterboard` and `trade.main_works` (fire-rated
  walls and ceilings built and openings formed), `trade.hydraulic_plumbing`,
  `trade.gas_fitting`, `trade.mechanical`, `trade.electrical`,
  `trade.fire_services`, `trade.data_security`, `trade.solar_pv` (services
  run and fixed through the opening before it is sealed),
  `trade.structural_steel` (shop-applied fire coatings).
- Leaves for: `trade.plasterboard` / `trade.painting` (close-up after
  inspection), the certifier (inspection), the client (register for the
  fire safety schedule).
- Not this trade: fire doors, shutters and fire dampers (openings:
  head contractor, door hardware, mechanical).
- Every other services RFT today prices "builder's work: penetrations
  ... and fire sealing, by the head contractor". When passive fire is let
  separately, the sealing moves here and the head contractor keeps the
  opening only. The four new services RFTs in this group already word it
  "fire sealing, by the head contractor or the passive fire contractor".

**Hold points and inspections.**

- Certifier and fire engineer: inspection of fire-rated construction and
  concealed penetrations before close-up (fire-life-safety-guide,
  Construction verification).
- Fire engineer / certifier: any untested combination referred to them,
  never approved by analogy.
- No authority hold point named in the seeds.

**Certificates it signs.** An installer's certificate or statement for
the fire-stopping and passive fire work, backed by the register and the
test evidence. Standards named in the seed: AS 1530.4 (fire resistance
tests), NCC 2022 Volume 1 Part C2.7 and "Part C3.15/C4.5" (protection of
penetrations) — all `verified: false`. The seed is itself unsure of the
NCC 2022 part number for penetrations; it must be read from the instrument.
The fire safety schedule and Annual Fire Safety Statement (NSW EP&A
Regulation 2021, per ncc-reference-guide) list fire-rated construction as
an essential measure — `verified: false`.

**Licence (NSW).** Seeds do not name one. General knowledge: NSW has a
fire safety practitioner accreditation scheme for some fire safety
measures, and a contractor licence may apply for work on a residential
building. `verified: false`; owner to confirm.

**Main Works line it carves out.** Row `fire_passive`, line
`penetrations` ("Sealing of service penetrations") exists today. Proposed
new lines under `fire_passive`:

- `{key: joints, text: "Fire-rated joints, perimeter gaps and junctions"}`
- `{key: structural_protection, text: "Fire protection applied on site to structural members"}`

`rated` and `fire_doors` stay with Main Works.

**Sources.**

- fire-life-safety-guide.md: "## Passive fire protection",
  "## Construction verification and integrated testing",
  "## Handover and operations", "## Common failures and low-confidence flags"
- ncc-reference-guide.md: "### SOU Separation and Service Penetrations"
- as-standards-reference.md: "## AS 1530 Series — Fire Testing"
- structural-commercial-industrial-guide.md (shop coatings, via the
  existing steel RFT)

**Open questions for the owner.**

1. Is on-site fire protection to steel (intumescent paint, boards,
   sprays) passive fire's by default, or the steel contractor's? Today the
   steel RFT prices "protective coatings and fire protection to the
   specification".
2. Should fire-rated walls and shaft linings ever sit in this package, or
   always with plasterboard / Main Works? The RFT carries shaft linings as
   "where the documents give them to this package".
3. Which certificate wording does the owner want on a NSW job?

---

## trade.gas_fitting — Gas fitting

**Summary.** Installs gas from the meter to each appliance, pressure
tests it before concealment, connects and commissions each appliance and
certifies the installation.

**House vs commercial.** House: natural gas or LPG to cooktop, oven,
hot water and heating (Type A appliances). Commercial: adds engineered
Type B appliances (commercial kitchens, boilers) that need design and
acceptance before use, larger meters and regulators, and plant-room
combustion air. Gas is activated only where gas use is evidenced
(hydraulic-services-guide).

**Interfaces.**

- Takes from: `trade.main_works` (penetrations), `trade.earthworks` /
  `trade.civil_works` (trench for an underground service), the client
  (network connection, appliance supply).
- Leaves for / shares with: `trade.hydraulic_plumbing` (hot water units and
  water connections), `trade.mechanical` (gas-fired heating plant; gas to
  its isolation valve), `trade.electrical` (power to appliances),
  `trade.passive_fire` (fire sealing), `trade.joinery_kitchens` (cooktop
  cut-out and gas point location).

**Hold points and inspections.**

- Pressure (soundness) test of pipework before linings close; evidence
  required before concealment (mep-residential, Key compliance).
- Type B appliances: engineering design and acceptance before use
  (as-standards-reference, AS/NZS 5601.1). Who accepts it is set by the
  state gas technical regulator — not named in the seed.
- Hydraulic engineer or certifier where the documents require it.

**Certificates it signs.** A gas compliance certificate for the
installation. Standards named: AS/NZS 5601.1 (gas installations), AS/NZS
1596 (LPG storage) — `verified: false`.

**Licence (NSW).** A NSW licence for gasfitting work, issued by NSW Fair
Trading; Type B work needs an additional authorisation. Seeds say only
"licensed gasfitter" and "Type B gasfitter supervision". `verified: false`.

**Main Works line it carves out.** Row `hydraulic`, line `gas` ("Gas,
where the documents show it") exists today.

**Sources.**

- mep-residential.md: "## Gas", "### System overview",
  "### Type A vs Type B appliances", "### Sizing and pressure",
  "### Key compliance", "### LPG specific"
- hydraulic-services-guide.md: "### Gas", "## Hydraulic trade tender",
  "## Construction, testing and handover"
- as-standards-reference.md: "## AS/NZS 5601.1 — Gas Installations — General Installations"

**Open questions for the owner.**

1. When gas fitting is let separately, the hydraulic RFT's `gas` stage
   should be excluded. Today `trade-hydraulic-plumbing.yaml` always shows
   it. Add a carve-out there?
2. Does the gas fitter or the plumber own the gas connection to the hot
   water unit? The RFT gives it to the gas fitter.

---

## trade.solar_pv — Solar PV and batteries

**Summary.** Installs and connects the PV system and any battery, gets
the network operator's approval, commissions to the approved settings
and hands over certified and registered.

**House vs commercial.** House: small roof array, often with a battery
in the garage; network approval is usually a basic application.
Commercial: larger arrays need a detailed network application and may
face export limits or protection studies; battery rooms need ventilation
and fire detection; monitoring joins the BMS. Seed thresholds for "basic"
vs "detailed" approval are not carried into the RFT.

**Interfaces.**

- Takes from: `trade.roofing` (roof covering; mounts sealed through it by
  this trade), structural engineer (roof capacity for the array and wind
  load), `trade.electrical` (switchboard space and a dedicated circuit for
  the inverter), `trade.data_security` (network for monitoring).
- Leaves for: `trade.electrical` (switchboard labelling coordination),
  `trade.passive_fire` (fire sealing), the client (meter change by the
  retailer or metering provider).
- Boundary with electrical: today `trade-electrical.yaml` has no solar
  line, so nothing overlaps; the electrical RFT stops at the switchboard.
  Solar hot water is hydraulic, not this trade (systems.yaml excludes).

**Hold points and inspections.**

- Network operator: connection approval before the inverter is energised;
  export and demand-response settings it sets.
- Structural engineer: roof capacity, where the documents require it.
- Electrical inspection by the network operator or its inspector in NSW —
  general knowledge, not in the seed; `verified: false`.

**Certificates it signs.** Electrical compliance certificate for the
installation (CCEW in NSW per mep-residential), system registration with
the electrical authority (as-standards-reference, AS/NZS 5033 pitfalls).
Standards named: AS/NZS 5033 (PV arrays), AS/NZS 4777.1 and 4777.2 (grid
connection via inverters), AS/NZS 5139 (battery storage), AS/NZS 3000 —
all `verified: false`. CEC accreditation named in the seed's related
standards — `verified: false`.

**Licence (NSW).** A NSW electrical contractor licence from NSW Fair
Trading for the work, with the installer's accreditation for PV and
batteries (the seed names CEC accreditation). `verified: false`.

**Main Works line it carves out.** None exists. Proposed under row
`electrical`:

- `{key: solar, text: "Solar PV and battery storage the documents show"}`

**Sources.**

- electrical-services-guide.md: "## Generation, storage and EV infrastructure",
  "## Construction, commissioning and handover"
- as-standards-reference.md: "## AS/NZS 5033 — Installation and Safety
  Requirements for Photovoltaic (PV) Arrays", "## AS/NZS 4777 Series — Grid
  Connection of Energy Systems via Inverters", "## AS/NZS 5139 — Electrical
  Installations — Safety of Battery Energy Storage Systems"
- mep-residential.md: "### Key compliance" (under "## Electrical", CCEW)

**Open questions for the owner.**

1. Battery is a stage that always shows with "where the documents show a
   battery". Should it hang on Profile scope (`electrical.battery-storage`)
   once child systems can drive RFT stages?
2. Who applies to the network operator on a house: this trade or the
   electrician? The RFT gives it to this trade.

---

## trade.data_security — Data, AV and security

**Summary.** Installs, configures and tests structured cabling, carrier
lead-in, AV and electronic security, proves the interfaces with fire and
lifts, and hands over records with credentials kept out of the manuals.

**House vs commercial.** House: NBN lead-in pit and pipe, a comms
cupboard, data outlets, maybe an alarm and intercom; commonly done by the
electrician. Commercial: communications rooms and risers, backbone and
horizontal cabling, wireless, AV rooms, access control, CCTV, intruder
alarms, integration with BMS and lifts; a separate specialist package.

**Interfaces.**

- Takes from: `trade.electrical` (power to racks, devices and locks;
  cable trays and conduits unless the documents give them here),
  `trade.carpentry_fixout` / `supplier.door_hardware` / Main Works (doors
  and frames prepared for electric locks), the carrier (network
  connection, carrier equipment, activation), `trade.fire_services`
  (fire alarm signal for door release).
- Leaves for: `trade.lifts` (lift access control and emergency
  communication line), `trade.solar_pv` (monitoring network),
  `trade.passive_fire` (fire sealing).
- Pit and pipe before slabs are poured (mep-residential): coordinate
  with `trade.concreting`.

**Hold points and inspections.**

- Inspection of concealed pathways, cable separation from power,
  outlets and fire stopping before linings close (ict-av-security-guide).
- Carrier: lead-in and pit and pipe to the carrier's requirements before
  activation.
- Integrated test of door release on fire alarm with the fire services
  contractor (fire strategy / cause and effect sets it).

**Certificates it signs.** Cabling compliance (the seed names AS/CA S009
as the customer cabling wiring rules and AS/NZS 3080 for commercial
generic cabling), cable test results. Security: AS/NZS 2201 series
(intruder alarms), AS 4806 series (CCTV). All `verified: false`.

**Licence (NSW).** Cabling: a registered cabler under the
Telecommunications Cabling Provider Rules (the seed names the rules, not
the registration). Security: a NSW security licence for installing alarms,
CCTV and access control (general knowledge; issued by the NSW Police
Security Licensing and Enforcement Directorate, not Fair Trading).
Electrical work to power devices stays with a licensed electrician. All
`verified: false`.

**Main Works line it carves out.** Row `electrical`, line `comms`
("Communications, data and security cabling the documents show") exists
today. AV has no Main Works line; propose under `electrical`:

- `{key: av, text: "Audiovisual systems the documents show"}`

**Sources.**

- ict-av-security-guide.md: "## ICT rooms, pathways and cabling",
  "## Audiovisual systems", "## Electronic security",
  "## Networks, integration and cybersecurity",
  "## Construction, commissioning and handover",
  "### Documentation and tender [documentation_tender]"
- mep-residential.md: "## NBN / communications",
  "### Pit-and-pipe (conduit infrastructure)"
- as-standards-reference.md: "## AS/CA S009 and AS/NZS 3080 —
  Telecommunications Cabling", "## AS/NZS 2201 Series and AS 4806 Series —
  Intruder Alarm and CCTV Systems"
- commercial-construction-guide.md: "## Security Systems and Access Control"

**Open questions for the owner.**

1. `trade-electrical.yaml` stage `comms` (data and security) shows
   whenever comms-security is in scope. When this trade is let separately
   it should be excluded. Add a carve-out there?
2. Should AV and security be separable from data (three trades) on large
   commercial jobs? The agreed list has one.

---

## trade.lifts — Lifts

**Summary.** Supplies, installs, tests and commissions the lifts (and any
escalators, platform or home lift), has them inspected and registered
before use, and hands over with records and any defects-period
maintenance.

**House vs commercial.** Commercial first: Class 2 to 9 buildings with
more than one storey usually need an accessible lift; firefighter lifts,
stretcher lifts, goods and car lifts and escalators appear on larger
jobs. House: an optional home or platform lift, priced as an option
(the Main Works row is already `if_unknown: option`).

**Interfaces.**

- Takes from: `trade.main_works` / `trade.concreting` / `trade.precast`
  (shaft, pit, pit waterproofing and drainage, lifting beams, fire rating
  of the shaft), `trade.electrical` (power to the lift isolator, shaft and
  machine-space lighting and power), `trade.fire_services` (fire alarm
  signal for recall), `trade.data_security` (emergency communication line,
  lift access control), `trade.mechanical` (shaft or machine room
  ventilation where needed).
- Leaves for: Main Works (landing surrounds, floor finishes up to the
  landing sill), `trade.passive_fire` (sealing around landing entrances
  where the tested system needs it).

**Hold points and inspections.**

- Early: the lift supplier confirms shaft, pit, headroom and machine
  space before the structure is built (commercial-construction-guide,
  pitfalls).
- Authority: design registration and registration of the lift as plant
  with the work health and safety regulator (SafeWork NSW in NSW) before
  use, and inspection by a registered lift inspector at the intervals
  state regulation sets (as-standards-reference). In words only;
  `verified: false`.
- Certifier: accessible lift features and fire-service operation as part
  of the occupation certificate.

**Certificates it signs.** Commissioning and test certificates,
the registration evidence. Standards named: AS 1735 series (lifts,
escalators and moving walks), AS 1735.1.2, AS 1735.2, AS 1735.12
(accessible lifts), NCC 2022 Volume 1 Part E2 / E3 (the seed names
"Part E2 — Lifts", "Part E3.6" and "Clause E3.3" inconsistently) — all
`verified: false`; the NCC part must be read from the instrument.

**Licence (NSW).** Seeds name no contractor licence. Lift work is
regulated as plant under WHS law (registration, competent installer);
electrical parts need a licensed electrician. `verified: false`.

**Main Works line it carves out.** Row `vertical_transport`, line
`lift` ("Lift supply, installation and commissioning") exists today;
`lift_shaft` stays with Main Works. The row has no `separate:` today.

**Sources.**

- commercial-construction-guide.md: "## Vertical Transport — Lifts and
  Escalators", "## Base Building vs Fitout — Scope Delineation"
- as-standards-reference.md: "## AS 1735 Series — Lifts, Escalators and
  Moving Walks"
- electrical-services-guide.md: "## Life-safety and specialist interfaces"
- ict-av-security-guide.md: "## Electronic security" (lift interfaces)
- multi-residential-apartments-guide.md (accessible lifts in Class 2)

**Open questions for the owner.**

1. Defects-period maintenance: in the lift price, or a separate
   maintenance contract the client signs? The RFT asks for it only
   "where the contract asks for it".
2. Should home lifts be a separate trade or supply (often bought direct
   from a home-lift supplier)?

---

## Overlaps with existing RFTs (not edited)

| Existing file | Stage | Overlaps | Proposed |
|---|---|---|---|
| `trade-hydraulic-plumbing.yaml` | `gas` | `trade.gas_fitting` | exclude while a live gas fitting package exists |
| `trade-electrical.yaml` | `comms` | `trade.data_security` | exclude while a live data, AV and security package exists |
| `trade-structural-steel.yaml` | `fabrication` line `finishes` (fire protection) | `trade.passive_fire` `elements.structural` | steel keeps shop-applied; passive fire takes site-applied (owner Q1) |
| every services RFT | position `builder` (fire sealing by head contractor) | `trade.passive_fire` | word as "by the head contractor or the passive fire contractor" |

These need the stage-level carve-out the plan proposes for Main Works
(slice T2), applied to trade RFTs too.
