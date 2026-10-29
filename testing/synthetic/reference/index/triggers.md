# Triggers and consequences
Generated on 28 October 2026 from the knowledge files on `main` (8a6a6c8 plus the working tree) by a read-only script. Regenerate rather than hand-edit. Back to [the index](../KNOWLEDGE_INDEX.md).

Every link is written **condition → profile key → consequence (source)**. The profile key is what
the harness or a document must set. "Generator" means one of: procurement suggestion, work item, cost line,
program fragment or row, RFP stage or position. "Implication" means a proposal on the Implications table
(`GET /projects/{id}/proposals`, `works.EvaluateProposals`, `internal/works/consequences.go`), which never
creates work, a package, a cost line or a program row by itself (`ApplyProposalDecision`,
`internal/works/proposals.go`). Unforeseen-condition implications are also printed into RFT and PMP reports,
not RFPs (`internal/reports/kinds.go`).

## 1. Determinants: document trigger → `det.*` → consumers

A passage is asked a determinant question only when one of its `triggers` (Go RE2, case-insensitive, `compile` in `internal/knowledge/profile.go`) matches. A determinant with no `profile_group` is not shown or writable in Profile (`validProfileValue`, `internal/httpapi/profile.go`), so the harness cannot set it by `PUT /profile/{key}`. Consumers are every knowledge file or Go path that reads the value.

| det id | profile group | type / options | document triggers | consumers |
|---|---|---|---|---|
| `ncc_class` | classification | multi_choice: 1a, 1b, 2, 3, 4, 5, 6, 7a, 7b, 8, 9a, 9b, 9c, 10a, 10b, 10c | `\bclass(ification)?\s*(1a\|1b\|7a\|7b\|9a\|9b\|9c\|10a\|10b\|10c)\b`; `\b(bca\|ncc\|building)\s+class(ification)?\b`; `\bclass\s*[2-8]\s+(building\|part\|occupancy\|portion\|use)\b` | RFP drivers → RFP part classes (class / only_class); consequence cq.nsw-class2-remedial-design-applicability-review; consequence cq.nsw-residential-contracting-applicability-review; derivation rule rule.basix.certificate-required (gives basix_required); derivation rule rule.ncc.accessible-lift-required (gives accessible_lift_required); derivation rule rule.ncc.energy-monitoring (gives energy_monitoring_required); derivation rule rule.ncc.ev-charging-readiness (gives ev_readiness_required); derivation rule rule.ncc.fire-compartment-limits (gives max_floor_area_m2); derivation rule rule.ncc.fire-compartment-volume-limit (gives max_volume_m3); derivation rule rule.ncc.residential-nathers-rating (gives nathers_min_stars); derivation rule rule.ncc.roof-ceiling-insulation-minimum (gives roof_min_total_r_value); derivation rule rule.ncc.sanitary-facility-quantities (gives required_sanitary_fixture_count); derivation rule rule.ncc.sou-floor-airborne-sound (gives sou_floor_min_rw_ctr); derivation rule rule.ncc.sou-floor-impact-sound (gives sou_floor_max_lnw_ci); derivation rule rule.ncc.sou-wall-sound-insulation (gives sou_wall_min_rw_ctr); derivation rule rule.ncc.type-of-construction (gives type_of_construction); derivation rule rule.ncc.wall-insulation-minimum (gives wall_min_total_r_value); 195 fragments |
| `rise_in_storeys` | classification | integer | `rise in storeys` | derivation rule rule.ncc.accessible-lift-required (gives accessible_lift_required); derivation rule rule.ncc.type-of-construction (gives type_of_construction); duration driver du.floor-cycle-2; duration driver du.lift-shafts-2; duration driver du.precast-install-2; duration driver du.stairs-2; duration driver du.stairs-9b; duration driver du.structure-9b |
| `effective_height` | classification | number | `effective height` | none |
| `sprinklered` | fire | boolean | `sprinkler` | interface if.fire-construction-depends-on-sprinklers |
| `type_of_construction` | fire | choice: A, B, C | — | derivation rule rule.ncc.fire-compartment-limits (gives max_floor_area_m2); derivation rule rule.ncc.fire-compartment-volume-limit (gives max_volume_m3) |
| `floor_area` | classification | number | `\bfloor area\b`; `\bfire compartments?\b` | derivation rule rule.ncc.accessible-lift-required (gives accessible_lift_required); derivation rule rule.ncc.energy-monitoring (gives energy_monitoring_required) |
| `state` | site | choice: ACT, NSW, NT, QLD, SA, TAS, VIC, WA | `\b(nsw\|vic\|qld\|sa\|wa\|tas\|act\|nt)\s+\d{4}\b` | RFP drivers → RFP fact state; consequence cq.nsw-alteration-approval-basis-review; consequence cq.nsw-class2-remedial-design-applicability-review; consequence cq.nsw-disturbance-information-review; consequence cq.nsw-essential-measures-continuity-review; consequence cq.nsw-residential-contracting-applicability-review; consequence cq.nsw-state-heritage-pathway-review; derivation rule rule.basix.certificate-required (gives basix_required); derivation rule rule.ncc.residential-nathers-rating (gives nathers_min_stars) |
| `bal` | site | choice: BAL-LOW, BAL-12.5, BAL-19, BAL-29, BAL-40, BAL-FZ | `\bbal[- ]?(low\|12\.5\|19\|29\|40\|fz)\b`; `bush ?fire attack level` | package_defaults complexity → Bushfire |
| `wind_class` | site | choice: N1, N2, N3, N4, N5, N6, C1, C2, C3, C4 | `wind class(ification)?`; `\b(n[1-6]\|c[1-4])\s+wind\b` | none |
| `site_class` | site | choice: A, S, M, H1, H2, E, P | `site class(ification)?` | none |
| `climate_zone` | site | choice: 1, 2, 3, 4, 5, 6, 7, 8 | `climate zone` | derivation rule rule.ncc.roof-ceiling-insulation-minimum (gives roof_min_total_r_value); derivation rule rule.ncc.wall-insulation-minimum (gives wall_min_total_r_value) |
| `flood_hazard` | site | boolean | `\bflood` | RFP drivers → flag.flood; fragment fr.flood-1a; fragment fr.flood-2 |
| `existing_building` | classification | boolean | `\bexisting building\b` | RFP drivers → flag.existing_building; consequence cq.existing-electrical-shutdown-plan-review; consequence cq.existing-fire-measures-approval-basis-review; consequence cq.existing-structural-intervention-sequence-review; consequence cq.nsw-alteration-approval-basis-review; consequence cq.nsw-disturbance-information-review; consequence cq.nsw-essential-measures-continuity-review; derivation rule rule.basix.certificate-required (gives basix_required); fragment fr.occupied-protection; interface if.tenant-fitout-base-building-hvac; 363 unforeseen records |
| `ncc_edition` | classification | choice | `\b(ncc\|bca)\s*(19\|20)\d\d\b` | none |
| `work_type` | — (fed from hdr.work_type) | choice | — | consequence cq.building-remediation-investigation-basis-review; consequence cq.nsw-class2-remedial-design-applicability-review; skeleton sk.extend-traditional; skeleton sk.new-dc-da; skeleton sk.new-eci-da; skeleton sk.new-traditional-cdc; skeleton sk.new-traditional-da; skeleton sk.refurb-traditional-cdc; skeleton sk.refurb-traditional-da; skeleton sk.ssd-dc; skeleton sk.ssd-traditional; 67 unforeseen records; 195 fragments |
| `patient_care_area` | — (not settable) | boolean | — | 1 unforeseen records |
| `car_parking_provided` | — (not settable) | boolean | — | derivation rule rule.ncc.ev-charging-readiness (gives ev_readiness_required) |
| `bushfire_prone_land` | site | boolean | `bush ?fire prone` | RFP drivers → flag.bushfire; fragment fr.bushfire-1a; fragment fr.bushfire-9b; 1 unforeseen records |
| `noise_affected_site` | site | boolean | `\b(road\|rail\|aircraft\|traffic) noise\b`; `\banef\b` | 1 unforeseen records |
| `facade_type` | site | choice: standard, complex | `\bcurtain ?wall`; `\buniti[sz]ed\b`; `\brain ?screen\b`; `\bfa[cç]ade (system\|engineer\|consultant\|design\|testing\|mock-?up)`; `\bcladding system` | package_defaults complexity → Facade |
| `estimated_works_cost` | — (not settable) | number | — | derivation rule rule.basix.certificate-required (gives basix_required) |
| `occupant_load` | — (not settable) | integer | — | derivation rule rule.ncc.sanitary-facility-quantities (gives required_sanitary_fixture_count) |
| `wind_region` | site | choice | `wind region`; `\bregion\s+[a-d]\b` | none |
| `acid_sulfate_soils_mapped` | site | boolean | `acid sulf(ate\|ide)` | RFP drivers → flag.acid_sulfate |
| `potentially_contaminated_land` | site | boolean | `contaminat` | RFP drivers → flag.contamination |
| `termite_prone_area` | site | boolean | `termite` | none |
| `groundwater_above_excavation` | — (not settable) | boolean | — | RFP drivers → flag.groundwater |
| `existing_building_year` | site | integer | `\byear (?:built\|of construction\|constructed)\b`; `\b(?:built\|constructed) in \d{4}\b` | consequence cq.pre-2004-fabric-hazardous-materials-survey; 2 unforeseen records |
| `flood_planning_level` | site | number | `flood planning level`; `\bfpl\b` | none |
| `water_supply_source` | services | choice: reticulated, rainwater_tank, bore, mixed | `rain ?water tanks?`; `\btown water\b`; `\bbore\b`; `\bpotable water\b`; `\bmains water\b`; `\bwater main\b` | none |
| `sewer_connection` | services | choice: reticulated, onsite_treatment, pressure_sewer | `\bseptic\b`; `wastewater treatment`; `\bsewer`; `pressure sewer` | none |
| `gas_supply` | services | choice: natural_gas_main, lpg, none | `\blpg\b`; `natural gas`; `gas (main\|supply\|bottles?\|cylinders?)` | none |
| `saline_soil` | site | boolean | `salin(e\|ity)` | none |
| `storage_height` | fire | number | `storage height`; `top of storage`; `racking height` | none |
| `heritage_status` | site | choice: none, conservation_area, local_item, state_register | `heritage` | RFP drivers → flag.heritage (none → no, else yes); consequence cq.nsw-state-heritage-pathway-review; fragment fr.heritage-referral; package_defaults complexity → Heritage; 2 unforeseen records |

Determinants with neither triggers nor consumers are omitted. `det.work_type` in predicates is fed from `hdr.work_type`, never from a determinant value (`Catalog.Holds`, `internal/knowledge/works_eval.go`).

## 2. Conditions: `hdr.cond.*` → consumers

| condition | options | consumers |
|---|---|---|
| `hdr.cond.planning` | exempt, cdc, da, ssd, planning_permit, ref (user only), modification (user only), planning_proposal (user only), advice_only (user only) | RFP fact pathway; cost starter da-fees; fragment fr.cdc-application; fragment fr.construction-certificate; fragment fr.da-assessment; fragment fr.da-exhibition; fragment fr.da-lodgement; fragment fr.exempt-confirmation; fragment fr.ssd-assessment; fragment fr.ssd-eis; package_defaults complexity → Town Planner; skeleton sk.extend-traditional; skeleton sk.new-dc-da; skeleton sk.new-eci-da; skeleton sk.new-traditional-cdc; skeleton sk.new-traditional-da; skeleton sk.refurb-traditional-cdc; skeleton sk.refurb-traditional-da; skeleton sk.ssd-dc; skeleton sk.ssd-traditional |
| `hdr.cond.procurement_route` | traditional, design_construct, eci, managing_contractor, alliance, ppp | RFP fact procurement; package novation default (DefaultNovation); head-contract design lines (BuilderDesigns); fragment fr.novation; skeleton sk.extend-traditional; skeleton sk.new-dc-da; skeleton sk.new-eci-da; skeleton sk.new-traditional-cdc; skeleton sk.new-traditional-da; skeleton sk.refurb-traditional-cdc; skeleton sk.refurb-traditional-da; skeleton sk.ssd-dc; skeleton sk.ssd-traditional |
| `hdr.cond.access_constraints` | unrestricted, urban_constrained, restricted_hours, remote | none |
| `hdr.cond.operational_constraints` | vacant, partial_occupation, live_environment, 24_7_occupied | RFP flag.live_operations (vacant → no; partial_occupation/live_environment/24_7_occupied → yes) |
| `hdr.cond.quality_level` | project_home, mid, high, prestige | RFP fact quality (house quality labels) |
| `hdr.cond.stakeholder_complexity` | single_owner, strata, government, multiple_agencies | none |
| `hdr.cond.environmental_sensitivity` | standard, sensitive, protected_habitat, aboriginal_heritage | package_defaults complexity → Archaeology; package_defaults complexity → Ecology |
| `hdr.cond.power_source` | grid, off_grid_solar, hybrid, diesel_backup | none |
| `hdr.cond.hazardous_process` | none, dangerous_goods, gmp, biosafety | none |

## 3. Flag readings: document trigger → `proj.flag.<key>` → RFP rows

`knowledge/profile/flag_readings.yaml`: a passage is asked yes / no / not stated only when a trigger matches. The value is stored as `proj.flag.<key>` (`rfp.FlagHome`, `internal/rfp/homes.go`) and read by `rfp.Drivers` (`internal/rfp/drivers.go`). In `shape.yaml` the validator's key is `flag.<key>` (`testing/synthetic/tools/validate_shape.py _profile_key_ok`). `proj.flag.basement` also scales the cost split's substructure share (`statedConstructionBudget`, `internal/store/cost_budget.go`).

| flag | triggers | RFP rows that test it |
|---|---|---|
| `groundwater` | `ground ?water`; `water table`; `dewater` | civil.positions.earthworks_suitability |
| `coastal` | `coastal (?:hazard\|use area\|environment\|wetland\|management\|zone)`; `coastal protection` | none |
| `country` | `connecting with country`; `aboriginal cultural`; `first nations`; `country (?:statement\|input)` | architecture.positions.country |
| `staged` | `\bstages? [1-9a-c]\b`; `staged (?:construction\|delivery\|development\|handover\|works\|release)`; `in (?:two\|three\|four\|\d) stages` | access.positions.occupied, arborist.positions.existing, architecture.allowances.workshops, architecture.stages.brief_masterplan, land-surveying.positions.access, module staged |
| `multi_building` | `\bbuildings? [a-d]\b`; `\bcampus\b`; `multiple buildings`; `(?:two\|three\|four\|five\|\d+) buildings` | access.positions.shared_routes, architecture.allowances.workshops, architecture.stages.brief_masterplan, module staged |
| `basement` | `\bbasement`; `deep excavation`; `excavat\w* to a depth` | civil.positions.shoring, structural.positions.dilapidation, structural.positions.retention_design, structural.positions.temporary_works_interfaces, structural.scale(complex) Also cost split basement factor. |
| `excavation_adjoining` | `excavat\w*[^.]{0,120}(?:adjoin\|neighbour\|boundar)`; `(?:adjoin\|neighbour\|boundar)\w*[^.]{0,120}excavat` | civil.positions.shoring, structural.positions.dilapidation, structural.positions.retention_design, structural.positions.temporary_works_interfaces |
| `transfer_structures` | `transfer (?:slab\|beam\|structure\|girder\|level\|floor)` | structural.scale(complex) |
| `authority_asset` | `build(?:ing)? over`; `(?:sewer\|water) (?:main\|asset)`; `trunk (?:main\|sewer\|drain)`; `sydney water asset`; `authority asset` | architecture.positions.water_asset, structural.scale(complex), structural.stages.authority_asset |
| `rail` | `rail (?:corridor\|line\|network\|infrastructure)`; `railway`; `sydney trains` | structural.scale(complex), structural.stages.authority_asset |
| `osd_existing` | `(?:on-?site detention\|\bosd\b)[^.]{0,100}(?:existing\|retain\|modif)`; `existing[^.]{0,60}(?:on-?site detention\|\bosd\b)` | civil.positions.osd_existing, civil.positions.osd_replacement, structural.stages.existing_assessment |
| `osd_required` | `on-?site (?:stormwater )?detention`; `\bosd\b` | civil.positions.osd_new |
| `norbe` | `neutral or beneficial`; `\bnorbe\b`; `drinking water catchment` | civil.positions.norbe, civil.positions.water_quality, civil.scale(complex) |
| `stormwater_quality` | `\bmusic model`; `stormwater quality`; `water sensitive urban design`; `\bwsud\b`; `pollutant (?:load\|reduction)` | civil.positions.water_quality |
| `public_road` | `road occupancy`; `works (?:in\|within) (?:the )?(?:public )?road`; `road opening`; `intersection upgrade`; `road reserve`; `public road works` | civil.elements.public_road, civil.positions.public_road_approvals, civil.positions.road_safety_audit, civil.scale(complex), traffic.positions.road_safety_audit |
| `state_road` | `state road`; `classified road`; `\btfnsw\b`; `transport for nsw`; `\brms\b`; `roads and maritime` | traffic.scale(complex) |
| `heavy_vehicles` | `heavy vehicle`; `b-?double`; `semi-?trailer`; `articulated vehicle`; `truck movements`; `\b[hms]rv\b` | traffic.scale(complex) |
| `tfnsw_referral` | `traffic[- ]generating development`; `referred to (?:tfnsw\|transport for nsw)` | traffic.scale(complex) |
| `intersection_modelling` | `\bsidra\b`; `intersection (?:count\|model\|analysis\|performance\|survey)`; `traffic (?:count\|model)` | traffic.positions.microsimulation, traffic.stages.modelling, traffic.stages.surveys |
| `gtp_required` | `green travel plan`; `\bgtp\b`; `workplace travel plan` | traffic.stages.travel_plan |
| `cl46` | `clause 4\.6`; `\bcl\.? ?4\.6`; `(?:variation\|exception) to (?:a )?development standard` | town-planning.scale(complex), town-planning.stages.variation_request |
| `panel` | `planning panel`; `regionally significant development`; `independent hearing and assessment panel`; `\bihap\b` | town-planning.allowances.panel_meetings, town-planning.positions.panel_attendance, town-planning.scale(complex) |
| `community` | `community (?:consultation\|engagement)`; `public exhibition`; `consultation plan` | architecture.positions.community |
| `architect_lodges` | `\blodge(?:d\|s\|ment)?\b`; `\blodging\b` | architecture.positions.lodgement |
| `sustainability_target` | `green star`; `nabers`; `passive ?haus\|passive ?house`; `\bleed\b`; `net[- ]zero` | architecture.positions.sustainability_docs |
| `arborist_offsite_trees` | `(?:neighbou?r\|adjoin\|street\|council\|off[- ]?site)[^.]{0,70}\btrees?\b`; `\btrees?\b[^.]{0,70}(?:neighbou?r\|adjoin\|street\|off[- ]?site)` | arborist.positions.offsite |
| `arborist_tree_impacts` | `(?:tree\|root\|canopy)[^.]{0,90}(?:impact\|affect\|excavat\|demoli\|trench\|foundation\|clearance\|protection)`; `(?:excavat\|demoli\|trench\|foundation\|works)[^.]{0,90}(?:tree\|root\|canopy)` | arborist.positions.impacts, arborist.stages.s2_impact, arborist.stages.s3_protection, arborist.stages.s4_tender, arborist.stages.s5_construction |
| `arborist_pruning` | `\bprun`; `\btrees?\b[^.]{0,70}(?:remov\|clear)`; `(?:remov\|clear)[^.]{0,70}\btrees?\b`; `canopy clearance` | arborist.positions.pruning |
| `arborist_offsite_access` | `(?:neighbou?r\|off[- ]?site\|adjoining)[^.]{0,100}(?:access\|permission\|consent)`; `(?:access\|permission\|consent)[^.]{0,100}(?:neighbou?r\|off[- ]?site\|adjoining)` | arborist.positions.offsite |
| `fire_vulnerable_occupants` | `assisted evacuation`; `vulnerable occupants`; `non[- ]ambulant`; `occupants?[^.]{0,80}(?:assist\|evacuat)`; `(?:staff\|carer)[^.]{0,80}evacuat` | fire-engineering.positions.occupants |
| `fire_special_hazards` | `special (?:fire )?hazard`; `dangerous goods`; `(?:storage\|process\|battery\|batteries\|electric vehicle\|\bev\b)[^.]{0,90}(?:fire\|thermal runaway\|explosion)`; `(?:fire\|thermal runaway\|explosion)[^.]{0,90}(?:storage\|process\|battery\|batteries\|electric vehicle\|\bev\b)` | fire-engineering.positions.hazards |
| `fire_specialist_systems` | `(?:facade\|cladding\|timber\|modular\|tested.system\|structural)[^.]{0,100}(?:fire\|departure\|performance solution)`; `(?:fire\|departure\|performance solution)[^.]{0,100}(?:facade\|cladding\|timber\|modular\|tested.system\|structural)` | fire-engineering.positions.specialist |

## 4. RFP flags read from determinants or conditions

| flag | read from | RFP rows that test it |
|---|---|---|
| `bushfire` | `det.bushfire_prone_land` | none |
| `flood` | `det.flood_hazard` | civil.positions.flood_evacuation, civil.positions.flood_report, civil.positions.trunk_drainage, civil.scale(complex) |
| `contamination` | `det.potentially_contaminated_land` | civil.positions.earthworks_suitability |
| `acid_sulfate` | `det.acid_sulfate_soils_mapped` | civil.positions.earthworks_suitability |
| `existing_building` | `det.existing_building` | access.positions.existing, arborist.positions.existing, module existing_building, structural.stages.existing_assessment |
| `heritage` | `det.heritage_status` | arborist.positions.heritage, architecture.scale(complex), structural.scale(complex), town-planning.scale(complex) |
| `live_operations` | `hdr.cond.operational_constraints` | access.positions.occupied, arborist.positions.existing, cost-planning.positions.existing_costs, land-surveying.positions.access, module live_site |
| `mixed_use` | `derived from parts' classes` | access.positions.shared_routes, fire-engineering.positions.occupants, module mixed_use, waste-management.positions.shared |

`flag.existing_building` is also derived from the work type when not stated: refurb, extend, remediation → yes; new → no (`deriveFlags`, `internal/rfp/drivers.go`).

## 5. RFP flags no document can set (user decision only)

These skeleton flags have no flag reading and no determinant home, so they stay unknown until the user sets `proj.flag.<key>`. Most are `scope_decision: true` client choices (`knowledge/rfp/skeleton.yaml flags`). Unknown makes the row ask, or price as an option where the row says `if_unknown: option`. The validator rejects these as `flag.<key>` profile keys because they are not in `flag_readings.yaml`.

| flag | label | RFP rows |
|---|---|---|
| `hyd_process` | Process-treatment interface selected | hydraulic-services.positions.process |
| `hyd_investigation` | Investigation selected | hydraulic-services.positions.existing, hydraulic-services.stages.investigation |
| `hyd_planning` | Authority inputs selected | hydraulic-services.stages.planning |
| `hyd_sources` | Alternative sources | none |
| `hyd_rainwater` | Rainwater collection and reuse | hydraulic-services.positions.rainwater |
| `hyd_alternative_sources` | River/carted/other water sources | hydraulic-services.positions.alternative_sources |
| `hyd_pumps` | Domestic pumps/storage | hydraulic-services.positions.pumps |
| `hyd_treatment` | Treatment connection | hydraulic-services.positions.treatment |
| `hyd_roof` | Roof drainage | hydraulic-services.positions.roof |
| `hyd_external` | External drainage | hydraulic-services.positions.civil |
| `hyd_trade` | Trade waste | hydraulic-services.positions.trade |
| `hyd_gas` | Gas/fluids | hydraulic-services.positions.gas |
| `hyd_utility` | Utility and Water Services Coordinator | hydraulic-services.positions.utility |
| `hyd_interfaces` | Building interfaces selected | hydraulic-services.positions.interfaces |
| `hyd_staging` | Phase/tenancy design selected | hydraulic-services.positions.staging |
| `hyd_declarations` | Declarations selected | hydraulic-services.positions.declarations |
| `elec_declarations` | Declarations selected | electrical-services.positions.declarations |
| `elec_investigation` | Investigation selected | electrical-services.positions.existing, electrical-services.stages.investigation |
| `elec_planning` | Authority inputs selected | electrical-services.stages.planning |
| `elec_controls` | Controls interface selected | electrical-services.positions.controls |
| `elec_generation` | Generation/storage/EV selected | electrical-services.positions.generation |
| `elec_ict` | Passive ICT/security/AV selected | electrical-services.positions.ict |
| `elec_dry_fire` | Dry-fire design selected | electrical-services.positions.dry_fire |
| `elec_staging` | Phase/tenancy design selected | electrical-services.positions.staging |
| `mech_investigation` | Investigation selected | mechanical-services.positions.existing, mechanical-services.stages.investigation |
| `mech_planning` | Planning inputs selected | mechanical-services.stages.planning |
| `mech_local` | Local exhaust selected | mechanical-services.positions.local |
| `mech_kitchen` | Kitchen exhaust selected | mechanical-services.positions.kitchen |
| `mech_smoke` | Smoke design selected | mechanical-services.positions.smoke |
| `mech_controls` | BMS design selected | mechanical-services.positions.controls |
| `mech_interfaces` | Plant interfaces selected | mechanical-services.positions.interfaces |
| `mech_declarations` | Declarations selected | mechanical-services.positions.declarations |
| `fs_wet` | Wet design selected | fire-services.positions.wet |
| `fs_supply` | Demand/storage selected | fire-services.positions.supply |
| `fs_authority` | Authority/test tasks allocated | fire-services.positions.authority |
| `fs_dry` | Dry design allocated | fire-services.positions.dry |
| `fs_controls` | Controls coordination allocated | fire-services.positions.controls |
| `fs_plant_interfaces` | Plant interface review selected | fire-services.positions.plant |
| `fs_investigation` | Existing investigation selected | fire-services.positions.existing, fire-services.stages.investigation |
| `fs_upgrade` | Upgrade design selected | fire-services.positions.upgrade |
| `fs_declarations` | Declarations selected | fire-services.positions.declarations |
| `fe_solutions` | Performance solution scope selected | fire-engineering.positions.brief, fire-engineering.positions.fer, fire-engineering.stages.s3_brief, fire-engineering.stages.s4_report |
| `fe_existing_reconciliation` | Existing FER reconciliation extent selected | fire-engineering.positions.existing |
| `arborist_impact_report` | Arboricultural impact report selected | arborist.stages.s2_impact |
| `arborist_protection_plan` | Tree protection plan selected | arborist.stages.s3_protection |
| `objections` | Notification likely to draw objections | town-planning.positions.submissions_response, town-planning.scale(complex) |
| `super_flat_floor` | Super-flat floor needed for the use | structural.positions.super_flat_floor |
| `contract_admin` | Client wants the consultant to administer the construction contract | architecture.stages.contract_admin |
| `bca_advisory` | BCA advisory requested | bca-certification.boundary, bca-certification.role.advisory, bca-certification.stages.a1_inception, bca-certification.stages.a2_planning, bca-certification.stages.a3_tender, bca-certification.stages.a4_construction |
| `bca_certification` | Certification requested | bca-certification.boundary, bca-certification.positions.statutory_functions, bca-certification.role.statutory, bca-certification.stages.b1_preliminary, bca-certification.stages.b2_construction_approval, bca-certification.stages.b3_construction, bca-certification.stages.b4_occupation |
| `bca_existing_assessment` | Existing BCA extent agreed | bca-certification.positions.existing_assessment |
| `bca_classification_map` | Classification map required | bca-certification.positions.classification_map |
| `bca_staged_occupation` | Occupied/staged BCA assessment | bca-certification.positions.staged_occupation |
| `bca_fire_referrals` | Fire referrals/witnessing required | bca-certification.positions.fire_referrals |
| `bca_residential_evidence` | Residential evidence review required | bca-certification.positions.residential_evidence |
| `geotech_intrusive` | Intrusive investigation selected | geotechnical.positions.clearance, geotechnical.stages.g2_field |
| `geotech_foundation` | Foundation advice selected | geotechnical.positions.foundation |
| `geotech_civil_parameters` | Civil ground parameters selected | geotechnical.positions.civil_parameters |
| `geotech_asset_exposure` | Asset exposure selected | geotechnical.positions.asset_exposure |
| `geotech_ass` | ASS assessment package selected | geotechnical.positions.ass |
| `geotech_handback` | Investigation handback selected | geotechnical.positions.handback |
| `geotech_planning` | Ground planning support selected | geotechnical.stages.g4_planning |
| `survey_detail` | Detail survey selected | land-surveying.stages.s2_detail |
| `survey_boundary` | Boundary survey selected | land-surveying.positions.boundary, land-surveying.stages.s3_boundary |
| `survey_services` | Services survey selected | land-surveying.stages.s4_services |
| `survey_adjoining` | Adjoining features selected | land-surveying.positions.adjoining |
| `survey_buried` | Buried geometry selected | land-surveying.positions.buried |
| `survey_sewer` | Sewer evidence selected | land-surveying.positions.sewer |
| `cost_concept` | Concept estimate selected | cost-planning.stages.s2_concept |
| `cost_authority` | Authority estimate selected | cost-planning.stages.s3_authority |
| `cost_design` | Design cost reviews selected | cost-planning.stages.s4_design |
| `cost_total_budget` | Total project budget extent agreed | cost-planning.positions.total_budget |
| `cost_contract_allowances` | Contract allowance schedule selected | cost-planning.positions.contract_allowances |
| `cost_operator` | Operator equipment cost inputs selected | cost-planning.positions.operator |
| `cost_allocation` | Shared cost allocation selected | cost-planning.positions.allocation |
| `cost_remediation` | Specialist remediation cost extent agreed | cost-planning.positions.remediation |
| `access_planning` | Access planning report selected | access.stages.s3_planning |
| `access_approval` | Access building-approval review selected | access.stages.s4_approval |
| `access_shared_routes` | Shared tenancy access review selected | access.positions.shared_routes |
| `access_housing` | Housing or SDA assessment selected | access.positions.housing |
| `access_facilities` | Hearing signage lifts and facility criteria review selected | access.positions.facilities |
| `waste_operational` | Operational waste plan selected | waste-management.positions.collection, waste-management.positions.operational, waste-management.positions.rooms, waste-management.stages.s2_concept, waste-management.stages.s3_operational |
| `waste_cd` | Construction and demolition waste plan selected | waste-management.positions.cd, waste-management.stages.s3_cd |
| `waste_shared` | Shared waste facilities or operator coordination selected | waste-management.positions.shared |
| `landscape_design` | Landscape design extent selected | landscape-architecture.stages.s2_concept, landscape-architecture.stages.s3_approval, landscape-architecture.stages.s4_developed, landscape-architecture.stages.s5_tender, landscape-architecture.stages.s6_dnc, landscape-architecture.stages.s7_construction, landscape-architecture.stages.s8_establishment |
| `landscape_podium` | Podium roof planter interface extent selected | landscape-architecture.positions.podium |
| `landscape_irrigation_brief` | Irrigation notation or performance brief selected | landscape-architecture.positions.irrigation_brief |
| `landscape_irrigation_detail` | Detailed irrigation design selected | landscape-architecture.positions.irrigation_detail |
| `landscape_public` | Public-domain package selected | landscape-architecture.positions.public |
| `landscape_special` | Special external-use coordination selected | landscape-architecture.positions.special |
| `landscape_staging` | Existing occupied mixed-use or staged acceptance extent selected | landscape-architecture.positions.staging |

## 6. Consequences (`knowledge/clusters/*/consequences.yaml`) → Implications

Evaluated in code over the included work items and the part's applied values (`EvaluateProposals`). `works:` reads work items (`action` and `system` lists; an item on the listed system or below matches). A true signal marks the proposal `addressed_by_evidence`.

| id | cluster | when | proposes (kind: label) | severity | signals |
|---|---|---|---|---|---|
| `cq.building-remediation-investigation-basis-review` | delivery | `{all:[{det:work_type,any_of:[remediation]},{works:{action:[alter,replace,upgrade,repair],system:[substructure,structure,envelope,interiors,fire-active,fire-passive,mechanical,hydraulic,electrical]}}]}` | hold_point: Review defect cause and extent before fixing the building-remediation solution | programme | — |
| `cq.nsw-alteration-approval-basis-review` | delivery | `{all:[{det:state,any_of:[NSW]},{det:existing_building,is:true},{works:{action:[alter,replace,upgrade],system:[structure,envelope,interiors,fire-active,fire-passive,mechanical,hydraulic,electrical]}}]}` | approval: Review NSW alteration approval basis, existing orders and certifier requirements; do not assume a whole-building upgrade | programme | — |
| `cq.nsw-class2-remedial-design-applicability-review` | delivery | `{all:[{det:state,any_of:[NSW]},{det:ncc_class,any_of:[2]},{det:work_type,any_of:[remediation]},{works:{action:[alter,replace,upgrade,repair],system:[structure,envelope,interiors,fire-active,fire-passive,mechanical,hydraulic,electrical]}}]}` | approval: Review NSW Class 2 remedial-design applicability, exclusions, emergency treatment and practitioner requirements | programme | — |
| `cq.nsw-state-heritage-pathway-review` | delivery | `{all:[{det:state,any_of:[NSW]},{det:heritage_status,any_of:[state_register]},{works:{action:[new,alter,replace,upgrade,repair,remove],system:[site,structure,envelope,interiors]}}]}` | approval: Review State Heritage Register work against exemption conditions and required approvals | programme | — |
| `cq.nsw-disturbance-information-review` | delivery | `{all:[{det:state,any_of:[NSW]},{det:existing_building,is:true},{works:{action:[alter,replace,remove],system:[structure,envelope,interiors]}}]}` | hold_point: Review potential material disturbance, asbestos information and inaccessible areas; identify any competent assessment needed before the work | life-safety | — |
| `cq.nsw-residential-contracting-applicability-review` | delivery | `{all:[{det:state,any_of:[NSW]},{det:ncc_class,any_of:[1a,1b,2,4]},{works:{action:[new,alter,replace,upgrade,repair,remove],system:[substructure,structure,envelope,interiors,fire-active,fire-passive,mechanical,hydraulic,electrical]}}]}` | hold_point: Review statutory work category, contract value and entity to establish NSW licensing, contract and insurance requirements and exemptions | programme | — |
| `cq.existing-electrical-shutdown-plan-review` | electrical-comms | `{all:[{works:{action:[alter,replace,upgrade,remove],system:[electrical]}},{det:existing_building,is:true}]}` | hold_point: Review electrical isolation and restoration planning before alteration | life-safety | — |
| `cq.existing-fire-measures-approval-basis-review` | fire | `{all:[{works:{action:[alter,replace,upgrade,remove],system:[fire-active,fire-passive]}},{det:existing_building,is:true}]}` | approval: Review the existing fire strategy and confirm approval implications of the changes | life-safety | — |
| `cq.nsw-essential-measures-continuity-review` | fire | `{all:[{det:state,any_of:[NSW]},{det:existing_building,is:true},{works:{action:[alter,replace,upgrade,repair,remove],system:[fire-active,fire-passive]}}]}` | hold_point: Review the NSW fire safety schedule, affected measures, maintenance and certification responsibilities, and continuity during the work | life-safety | — |
| `cq.mechanical-commissioning-plan-review` | services-wet-air | `{works:{action:[new,alter,replace,upgrade],system:[mechanical]}}` | hold_point: Review mechanical commissioning requirements before tender | programme | — |
| `cq.hydraulic-testing-plan-review` | services-wet-air | `{works:{action:[new,alter,replace,upgrade],system:[hydraulic]}}` | hold_point: Review hydraulic concealed-work and testing requirements before installation | programme | — |
| `cq.room-layout-air-distribution-review` | services-wet-air | `{all:[{works:{action:[new,alter,remove],system:[interiors.walls-linings],layout_change:yes}},{system_existing:mechanical.air-distribution}]}` | hold_point: Review air distribution and zoning for the changed room layout | programme | — |
| `cq.room-layout-smoke-strategy-review` | services-wet-air | `{all:[{works:{action:[new,alter,remove],system:[interiors.walls-linings],layout_change:yes}},{system_existing:mechanical.fire-mode-air-systems}]}` | hold_point: Review changed room boundaries against the existing smoke-control strategy, air paths and cause-and-effect design | life-safety | — |
| `cq.room-layout-thermal-zoning-review` | services-wet-air | `{all:[{works:{action:[new,alter,remove],system:[interiors.walls-linings],layout_change:yes}},{system_existing:mechanical.air-conditioning}]}` | hold_point: Review heat-load assumptions and thermal zoning for changed room boundaries with the responsible designer | cost | — |
| `cq.pre-2004-fabric-hazardous-materials-survey` | structure | `{all:[{works:{action:[alter,replace,upgrade,repair,remove]}},{det:existing_building_year,lt:2004}]}` | investigation: Hazardous materials survey of areas affected by the works | life-safety | sig.hazardous-materials-survey-stated |
| `cq.existing-structural-intervention-sequence-review` | structure | `{all:[{works:{action:[alter,replace,upgrade,remove],system:[structure,substructure]}},{det:existing_building,is:true}]}` | hold_point: Review load paths and temporary support before existing structural intervention | life-safety | — |

## 7. Interface consequences (`knowledge/works/interface_consequences.yaml`) → Implications

Applies when a work item touches the named side of an interface of that `type` with one of `actions`, and the other side exists on site and is not being replaced (file header; `interfaceProposals`, `internal/works/interfaces.go`). Interfaces by type: boundary 48, depends_on 26, loads 15, sequences 15, supplies 13, penetrates 12, controls 9, shares_space 6 (144 total).

| id | interface type | touches | actions | proposes |
|---|---|---|---|---|
| `ic.loads-investigate-supported` | loads | from | new, replace, upgrade | investigation: Capacity of the existing structure or ground for the new load |
| `ic.supplies-investigate-supply` | supplies | to | new, upgrade, alter | investigation: Capacity of the existing supply (flow and pressure, electrical, cooling) |
| `ic.supplies-keep-served-working` | supplies | from | replace, upgrade, alter, remove | obligation: Keep the served system working during the works (changeover, temporary supply or shutdown window) |
| `ic.penetrates-make-good` | penetrates | from | new, alter, replace | work_item: Make good the penetrated construction at penetrations (for example fire-stopping) |
| `ic.controls-test-link` | controls | either | any | investigation: Test the control link at commissioning |
| `ic.depends-on-check-reliance` | depends_on | to | alter, replace, remove | investigation: Whether what relies on it still holds (for example FRL concessions relying on sprinklers) |
| `ic.sequences-hold-point` | sequences | either | any | hold_point: Hold point in the package doing the earlier work |
| `ic.shares-space-coordinate` | shares_space | from | new, alter | obligation: Coordinate set-out with the existing system in that space |
| `ic.boundary-assign` | boundary | either | any | obligation: Assign the boundary to a package (gap check) |

## 8. Hold points → program rows

`gen.holdPointRows` (`internal/program/generate.go`) adds a zero-day `Hold point: <evidence>` row for a `sequences` interface whose `hold_point.after` and `.before` are both **system ids** and both have tagged program rows; `applies_when` false skips it. The Go comment says none resolve today; that is stale: the five below use system ids and can fire. The envelope cluster's ten hold points use free text and never do.

| interface | after (system) | before (system) | row label |
|---|---|---|---|
| `if.fire-stopping-before-concealment` | `fire-passive.penetrations` | `interiors` | Hold point: a location-based inspection record for concealed fire stopping |
| `if.sprinkler-pressure-test-before-ceilings` | `fire-active.sprinklers` | `interiors.ceilings` | Hold point: a recorded pressure test of the sprinkler pipework |
| `if.reinforcement-inspected-before-pour` | `structure.concrete` | `substructure.slab-on-ground` | Hold point: Reinforcement inspection release identifying the pour. |
| `if.frame-inspected-before-roof` | `structure.timber` | `envelope` | Hold point: Frame inspection release before roof cladding. |
| `if.underground-services-before-paving` | `site.services-connections` | `site.vehicle-access` | Hold point: Inspection evidence for concealed underground service routes. |

## 9. Unforeseen conditions (`knowledge/clusters/*/unforeseen.yaml`) → Implications

560 records. Each has a `when` (mostly a `works:` clause, often with `det: existing_building is true` or a work type), one `de_risk` proposal, a suggested `contract` treatment and `effect`. Determinants they read: `existing_building` 363, `work_type` 67, `heritage_status` 2, `existing_building_year` 2, `patient_care_area` 1, `bushfire_prone_land` 1, `noise_affected_site` 1. By cluster: delivery 65, electrical-comms 79, envelope 95, fire 43, services-wet-air 130, structure 148. The full list is in [unforeseen.md](unforeseen.md). Records whose condition is a site fact other than an existing building or the work type:

| id | when | de-risk |
|---|---|---|
| `uc.heritage-materials-and-trades-scarce` | `{all:[{det:work_type,any_of:[refurb]},{det:existing_building,is:true},{det:heritage_status,any_of:[conservation_area,local_item,state_register]}]}` | investigation: Confirm availability of authentic materials and heritage trades before pricing |
| `uc.hospital-refurbishment-needs-infection-control-barriers` | `{all:[{det:work_type,any_of:[refurb]},{det:existing_building,is:true},{det:patient_care_area,is:true}]}` | discipline: Infection control risk assessment and barrier design with the facility before works |
| `uc.overhead-power-lines-restrict-crane-and-delivery` | `{all:[{det:work_type,any_of:[new,extend,refurb,remediation]},{system_existing:electrical.utility-supply}]}` | investigation: Identify overhead power lines and plan crane and delivery access before site establishment |
| `uc.heritage-wall-opening-conflicts-with-fabric` | `{all:[{works:{action:[alter,upgrade],system:[envelope.external-walls]}},{det:existing_building,is:true},{det:heritage_status,any_of:[conservation_area,local_item,state_register]}]}` | investigation: Heritage fabric assessment before the opening is designed |
| `uc.asbestos-cladding-disturbed` | `{all:[{works:{action:[alter,upgrade,repair,remove],system:[envelope.external-walls]}},{det:existing_building,is:true},{det:existing_building_year,lt:2004}]}` | investigation: Hazardous materials survey of cladding before any cutting or removal |
| `uc.lead-paint-on-old-windows` | `{all:[{works:{action:[alter,upgrade,repair,replace,remove],system:[envelope.windows-doors,interiors.painting]}},{det:existing_building,is:true},{det:existing_building_year,lt:1990}]}` | investigation: Lead paint testing of old window and trim finishes before sanding or removal |
| `uc.bal-increased-after-design` | `{all:[{works:{action:[new,upgrade],system:[envelope.bushfire-construction,envelope.windows-doors,envelope.external-walls]}},{det:bushfire_prone_land,is:true}]}` | investigation: Bushfire attack level assessment before glazing, cladding and decks are designed |
| `uc.neighbouring-noise-source-requires-upgraded-facade` | `{all:[{works:{action:[new,upgrade],system:[envelope.facade-acoustics,envelope.windows-doors]}},{det:noise_affected_site,is:true}]}` | investigation: Acoustic assessment of the neighbouring noise source before facade design |

## 10. Failure modes (`knowledge/clusters/*/failure_modes.yaml`)

862 records (electrical-comms 144, envelope 227, fire 94, services-wet-air 208, structure 189). Each is a Jev noul detector loaded as an `fm:<id>` evidence question (`loadFailures`, `internal/knowledge/load.go`) and asked on passages labelled with its `runs_on` systems (`EvidenceCall`, `internal/jobs/worker.go`). No generator reads the answers: they add no consultant, work item, cost line, program row or RFP row. Where they surface in the UI is **unverified**.

## 11. Coverage ledgers (`knowledge/works/coverage/`)

`batch1.yaml` and `batch2.yaml` mark all 1,000 rows of each dataset `pending`; the per-cluster files under `batch1/` and `batch2/` map dataset rows to `fm.*` and `uc.*` record ids or reject them. They are provenance for the records above and are not read by any generator (they are not loaded by `knowledge.Load`; **unverified** beyond a search for `coverage` in `internal/knowledge`).

## 12. Words in documents that imply systems

See [scope-defaults.md](scope-defaults.md#words-that-imply-systems-scope_defaultsyaml-implied_by): `bathroom`, `kitchen`, `stair`, `party wall`, `strip-out`/`demolition`, `masonry`, `BASIX`/`NatHERS` and others suggest scope rows, which then drive work items, cost lines, scope-addition consultants and fragments.
