# Electrical, communications and vertical transport extraction report

Status: draft, 2026-09-29. Preserved 26 systems and 47 rules. After lead canonical reconciliation and this full-file sweep: 30 interfaces and 34 affirmative failure detectors. Lead merged 6 determinants into the canonical catalogue; the proposal file is empty. All numeric and clause claims remain unverified. This is a draft graph checkpoint, not a verified compliance database.

## Coverage

| Area | Coverage |
|---|---|
| Electrical supply | Utility capacity, demand, switchboard access, ventilation and flood exposure; distribution protection in inherited rules |
| Life safety | Fire pump and smoke-control supply, emergency lift supply, lift recall, security release, shutdown continuity |
| Renewables | PV roof loading and weatherproofing, inverter/network protection, battery escape routes and ventilation, EV load management |
| ICT and security | Carrier/civil boundary, cable segregation, cooling and UPS dependencies, CCTV lighting, AV environment, intrusion/HVAC interference |
| Lifts | Shaft geometry, fire separation, ventilation and accessible landing route |
| Concealment | Electrical rough-in before lining; luminaire/insulation compatibility |


## Full-file coverage matrix (continuation and follow-up)

Each file below was read in full during this continuation, including the follow-up sweep. Long files were read in bounded chunks. Reading coverage does not imply that every seed assertion was copied or verified.

| Source files in `../clerk/data/seed/` | Extraction treatment |
|---|---|
| `ncc-reference-guide.md`; `as-standards-reference.md` | Full reference sweep; applicable physical requirements retained as draft/unverified; no executable table created from seed prose |
| `structural-residential.md`; `structural-commercial-industrial-guide.md`; `geotechnical-guide.md` | Structural and ground design, construction and failure interfaces |
| `civil-residential.md`; `civil-commercial-industrial-guide.md`; `surveying-guide.md` | Levels, fill, drainage, ground and infrastructure interfaces |
| `arborist-guide.md`; `landscape-architecture-guide.md`; `traffic-transport-guide.md`; `waste-management-guide.md` | Root zones, podium load/drainage, site access and collection interfaces; specialist design remains external |
| `electrical-services-guide.md`; `ict-av-security-guide.md`; `mep-residential.md` | Electrical, communications, security and integrated domestic services |
| `mechanical-services-guide.md`; `hydraulic-services-guide.md`; `architectural-services-guide.md`; `access-consultant-guide.md` | Cross-system coordination; detailed wet-air/access content remains with the owning clusters |
| `trade-interfaces-coordination-guide.md` | Concealment, sequence and installation boundaries; lead canonical aliases preserved |
| `remediation-due-diligence-guide.md`; `building-remediation-rectification-guide.md`; `defects-and-dlp-guide.md` | Contamination, cap integrity, vapour pathways, retained structure and investigation before repair |
| `residential-construction-guide.md`; `new-dwelling-guide.md`; `renovation-guide.md`; `ancillary-guide.md`; `multi-dwelling-guide.md`; `multi-residential-apartments-guide.md` | Full residential/archetype sweep; extracted physical dependencies, not legacy role or approval doctrine |
| `commercial-construction-guide.md`; `small-commercial-guide.md`; `industrial-construction-guide.md`; `institution-construction-guide.md`; `mixed-use-construction-guide.md`; `infrastructure-construction-guide.md` | Full nonresidential/archetype sweep; use boundaries, live services and structural interfaces; no assumed universal classification |
| `sustainability-energy-guide.md`; `non-residential-sustainability-energy-guide.md` | Certificate commitments, supply impacts, metering and operational verification; no rating prediction |

Intentionally excluded from extraction: fees and RFP wording, consultant appointment stages, commercial contracts, warranties/liability, cost rates, durations, procurement methods, role-specific agent behaviour and all legacy PM doctrine. Technical facts embedded in those sections were considered. Role overlays, dedicated contract/cost/procurement/program guides, `setup-and-commission-guide.md`, and the frozen clerk PM doctrine were not adopted as technical authorities. Detailed fire/access, envelope/finishes and wet-air extraction is owned by the other clusters; this report does not claim a full reread of their dedicated specialist seed inventories.

Residual source gaps: none in the full-file inventory above. Primary NCC/AS/state instruments, manufacturer instructions, network rules, project-specific engineering and test data are not comprehensively verified by this extraction. Seed completeness is limited by the source itself: specialist healthcare, process hazardous areas, linear infrastructure and complex remedial design are signposts, not detailed design knowledge. Full seed reading is complete for the listed cluster sources; engineering completeness and primary-instrument verification remain open.

## Contradictions and verification gaps

- NCC seeds use legacy Section J and lift numbering; inherited clauses are labelled seed numbering NCC 2019. Do not assume those addresses resolve in NCC 2022.
- Generic BESS energy thresholds, window separations, ventilation rates and PV isolator requirements are unverified seed claims, not approved design details.
- Seed renewable/network accreditation and edition statements may be stale; project-specific primary standard editions and network conditions are needed.
- Seed accessible-lift car examples conflate nominal capacity, clear dimensions, stretcher use and exemptions. Full route and scope conditions are needed before any table can operate.
- Generic CCTV retention, camera performance and cabling categories may be project recommendations rather than universal statutory minima. Rules remain draft; primary verification must distinguish them.
- Trade-interface source asserts a universal non-IC downlight clearance; new interface asks about manufacturer fitting compatibility, not this unverified dimension.
- Source mixes pressure-testing all services with electrical rough-in inspection. New electrical hold point uses inspection evidence only, not pressure testing.

## Tables and determinants

No tables written. Pending tables referenced by inherited rules: energy_monitoring_facilities, ev_charging_readiness, accessible_lift_requirement. Seed scope is insufficient to populate them safely. Lead has marked unavailable derivations pending; absence of determinant evidence remains unknown. Do not use overall floor_area for a per-storey lift exemption without explicit compatible scope. Proposed bess_energy_capacity values require code unit normalization; Jev only selects literal candidate values. Network operator should be extracted from stated evidence and must not be inferred from state alone.

## Cross-cluster edges

Fire: alarm-to-security-release, lift recall, essential power and shutdown continuity. Wet-air: switchroom/ICT/battery ventilation, smoke fans and HVAC detector interference. Structure: PV loading and lift shaft geometry. Envelope/interiors: PV waterproofing, insulation/downlights, rough-in closure, AV environment and switchroom access. Access-egress: security release, battery location and accessible lift landing. Site: flood exposure and carrier route. Cross edges use stable top-level IDs to avoid inventing child systems. Lead aliases are preserved for security release, lift recall, rough-in and downlights. Follow-up adds cable/thermal-insulation derating, lightning/electrical-earth bonding, meter data commissioning and independent staged energisation. The cable edge refers to envelope.thermal-insulation and does not duplicate luminaire clearance.

No unresolved system IDs found. The three unverified table dependencies are explicitly pending in the canonical determinant catalogue; no unresolved system IDs remain.

## Questions and validation

New questions ask one explicit fact using `text`, with positive criteria and narrowly scoped system routing. A failure detector requires stated defect evidence; document silence is not a failure finding. Runtime computation, calibration and thresholds are not performed by Jev.

Reviewed TypeSafe [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13), especially literal criteria, bounded state and arithmetic in code. The foundation/schema cite [AI-powered software](https://docs.typesafe.ai/concepts/how-to-build-with-system-one); direct reread failed in this continuation. No runtime tests or confidence calibration were performed.

Validation: python tools/check_knowledge.py --only knowledge/clusters/electrical-comms: 0 errors, 0 warnings after the full-file sweep (missing tables are explicitly pending). Checker success establishes structural validity only.

Additional source conflicts from the full sweep: residential/energy seeds disagree on six versus seven star requirements and conflate residential provisions with Section J; no new star-rating rule was created. Apartment/commercial guides overgeneralise individual metering, lift provision, classification and state obligations. Domestic MEP circuit counts and ratings are examples, not cable design defaults. The new staged energisation edge is a physical design dependency rather than a claim that every building legally requires staged handover.
