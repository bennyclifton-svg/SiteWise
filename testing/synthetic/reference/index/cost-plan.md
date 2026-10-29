# Cost plan knowledge
Generated on 28 October 2026 from the knowledge files on `main` (8a6a6c8 plus the working tree) by a read-only script. Regenerate rather than hand-edit. Back to [the index](../KNOWLEDGE_INDEX.md).

## Cost elements (`knowledge/costs/elements.yaml`)

One construction line per top-level system that has at least one included, live, non-group work item on it or below it (`costs.Suggest`, `internal/costs/suggest.go`; work items read by `costSuggestInput`, `internal/store/cost_suggest.go`, any review status). Line label is `cost_label`, else the system label. `trade` is the package the line is costed under: a live works package with that discipline id, else the head contract, else the planning label "Head contractor" (`mappedCostPackage`).

| code | system | line label | trade (discipline id) | overrides / itemised children |
|---|---|---|---|---|
| 01 | `site` | Site works and external works | `trade.civil_works` | — |
| 02 | `substructure` | Substructure | `trade.main_works` | — |
| 03 | `structure` | Structure | `trade.main_works` | `structure.steel` → `trade.structural_steel` (split out only when that trade package exists) |
| 04 | `envelope` | Building envelope | `trade.main_works` | — |
| 05 | `interiors` | Interiors and finishes | `trade.main_works` | `interiors.joinery` → `trade.joinery_kitchens` (split out only when that trade package exists) |
| 06 | `fire-passive` | Passive fire protection | `trade.main_works` | — |
| 07 | `fire-active` | Active fire protection | `trade.fire_services` | — |
| 08 | `mechanical` | Mechanical services | `trade.mechanical` | — |
| 09 | `hydraulic` | Hydraulic services | `trade.hydraulic_plumbing` | — |
| 10 | `electrical` | Electrical services | `trade.electrical` | — |
| 11 | `comms-security` | Communications and security | `trade.electrical` | — |
| 12 | `vertical-transport` | Vertical transport | `trade.main_works` | — |
| 13 | `access-egress` | Access and egress | `trade.main_works` | `access-egress.stairs-and-ramps`: "Stairs, ramps, handrails and nosings"; `access-egress.barriers`: "Compliant balustrades and barriers"; `access-egress.accessible-access`: "Accessible paths, TGSIs and braille/tactile signage"; `access-egress.accessible-facilities`: "Accessible sanitary facilities and hearing augmentation"; `access-egress.egress-doors`: "Egress doors and hardware"; `access-egress.exits`: **not costed** (Travel distances and exit layout are a design outcome priced in other elements.); `access-egress.livable-housing`: **not costed** (Livable-housing features are priced in interiors and hydraulic.) |

`access-egress` is `itemised: true`: it becomes a group line with one child per child system that has scope, codes `13.<n>` where n is the child's 1-based position in the sorted child list; children marked `costed: false` get no line.

## Starter lines (`knowledge/costs/starter_lines.yaml`)

`Catalog.CostStarterHolds` (`internal/knowledge/cost_elements.go`): `true` always; `{field, in}` reads the profile value of an `hdr.cond.*` key (only taxonomy conditions are allowed, `validateCostStarter`). The value is used only when every profile row for that key agrees and is known (`costSuggestInput` SQL, `internal/store/cost_suggest.go`). Unknown → the line is still suggested with "Applicability is unknown"; false → not suggested.

| code | id | label | category | section shown in | applies_when |
|---|---|---|---|---|---|
| P01 | `design-contingency` | Design contingency | contingency | contingency | `true` |
| P02 | `construction-contingency` | Construction contingency | contingency | contingency | `true` |
| P03 | `escalation` | Escalation | escalation | contingency | `true` |
| P04 | `da-fees` | DA fees and contributions | authority_fees | authority (fees and charges) | `{field:hdr.cond.planning,in:[da]}` |
| P05 | `cc-fees` | Construction certificate and certifier fees | authority_fees | authority (fees and charges) | `true` |
| P06 | `service-authority` | Service authority connection fees | authority_fees | authority (fees and charges) | `true` |
| P07 | `exclusions` | Exclusions | exclusions | contingency | `true` |

## Budget splits (`knowledge/costs/budget_splits.yaml`, `verified: false`)

`Catalog.ChooseBudgetSplit` (`internal/knowledge/budget_splits.go`): first split in file order whose `class` contains `hdr.building_class`, whose `subclass` (if listed) contains `hdr.subclass`, and whose `work_type` contains any of the applied work types. `storeys_min` needs `hdr.scale.storeys`: when storeys are unknown a split with `storeys_min` is **skipped**. `basement` multiplies the listed share when `proj.flag.basement` is `yes`. A split only spreads a stated `proj.budget_stated` over top-level posting construction lines (`spreadConstructionBudget`, `internal/store/cost_budget.go`); `preliminaries` has no line and is spread pro rata. No split exists for remediation or advisory, nor for industrial or institution refurb.

| # | id | label | conf. | class | subclass | work type | storeys | basement | shares (%) |
|---|---|---|---|---|---|---|---|---|---|
| 0 | `residential_house_new` | New house | medium | residential | house/townhouses/social_affordable_housing/retirement_living | new | — | {substructure:2} | preliminaries 18, site 5, substructure 11, structure 6, envelope 24, interiors 24, fire-active 0.2, mechanical 2.5, hydraulic 4.5, electrical 3.8, comms-security 0.5, access-egress 0.5 |
| 1 | `residential_apartments_high_rise` | Apartments, high-rise | medium | residential | apartments/btr/social_affordable_housing | new | >= 5 | — | preliminaries 19, site 2, substructure 9, structure 18, envelope 11, interiors 22.3, fire-active 2.8, mechanical 2.7, hydraulic 5.5, electrical 4.5, comms-security 0.2, vertical-transport 2.2, access-egress 0.8 |
| 2 | `residential_apartments_low_rise` | Apartments, low-rise | low | residential | apartments/btr | new | — | {substructure:2} | preliminaries 18.5, site 3.5, substructure 10, structure 12, envelope 17.5, interiors 23.2, fire-active 1.5, mechanical 2.6, hydraulic 5, electrical 4.2, comms-security 0.3, vertical-transport 1.1, access-egress 0.6 |
| 3 | `residential_student` | Student housing and co-living | medium | residential | student_housing | new | — | — | preliminaries 18.4, site 1.5, substructure 3.4, structure 10.4, envelope 18, interiors 26.2, fire-active 2.3, mechanical 3.2, hydraulic 7.8, electrical 5.2, comms-security 0.2, vertical-transport 3.2, access-egress 0.4 |
| 4 | `residential_aged_care` | Residential aged care | medium | residential | residential_aged_care | new | — | — | preliminaries 16.8, site 3.9, substructure 5.3, structure 7.2, envelope 17.6, interiors 28.1, fire-active 3.2, mechanical 5.9, hydraulic 5.5, electrical 4.5, comms-security 0.8, vertical-transport 0.9, access-egress 0.2 |
| 5 | `residential_extension` | House extension or alteration | medium | residential | any | extend/refurb | — | — | preliminaries 10, site 7.2, substructure 20.8, envelope 29.4, interiors 23.7, fire-active 0.4, mechanical 1.6, hydraulic 3.9, electrical 2.4, access-egress 0.7 |
| 6 | `residential_other` | Residential, other | low | residential | any | new | — | {substructure:2} | preliminaries 18, site 5, substructure 11, structure 6, envelope 24, interiors 24, fire-active 0.2, mechanical 2.5, hydraulic 4.5, electrical 3.8, comms-security 0.5, access-egress 0.5 |
| 7 | `commercial_fitout` | Commercial fit-out or refurbishment | low | commercial/mixed | any | refurb | — | — | preliminaries 14, interiors 50, fire-passive 1, fire-active 4, mechanical 12, hydraulic 2, electrical 11, comms-security 5, access-egress 1 |
| 8 | `commercial_office_high_rise` | Office, high-rise | low | commercial | office | new/extend | >= 8 | — | preliminaries 18, site 1, substructure 5, structure 19, envelope 14, interiors 14, fire-passive 1, fire-active 2, mechanical 8, hydraulic 3, electrical 7, comms-security 2, vertical-transport 5, access-egress 1 |
| 9 | `commercial_office` | Office | low | commercial | office/serviced_office_coworking | new/extend | — | {substructure:2} | preliminaries 16, site 8, substructure 5, structure 14, envelope 15, interiors 14, fire-passive 1, fire-active 3, mechanical 11, hydraulic 2, electrical 7, comms-security 2, vertical-transport 1, access-egress 1 |
| 10 | `commercial_shopping_centre` | Shopping centre | low | commercial | retail_shopping_centre | new/extend | — | — | preliminaries 20, site 3, substructure 11, structure 16, envelope 27, interiors 6, fire-passive 1, fire-active 2, mechanical 6, hydraulic 1, electrical 4, comms-security 1, vertical-transport 1, access-egress 1 |
| 11 | `commercial_hotel` | Hotel or residential over retail | low | commercial/mixed | hotel/hotel_residential/residential_retail/btr_retail/residential_commercial/vertical_village | new/extend | — | — | preliminaries 25, site 1, substructure 5, structure 6, envelope 13, interiors 26, fire-active 1, mechanical 8, hydraulic 2, electrical 7, comms-security 3, vertical-transport 2, access-egress 1 |
| 12 | `commercial_retail` | Standalone retail, food and beverage | low | commercial/mixed | any | new/extend | — | — | preliminaries 16, site 10, substructure 6, structure 6, envelope 20, interiors 14, fire-active 3, mechanical 12, hydraulic 1, electrical 8, comms-security 3, access-egress 1 |
| 13 | `industrial_high_bay` | Logistics, high-bay | low | industrial | logistics_ecommerce | new/extend | — | — | preliminaries 12, site 17, substructure 14, structure 16, envelope 15, interiors 4, fire-passive 1, fire-active 8, mechanical 3, hydraulic 2, electrical 6, comms-security 1, access-egress 1 |
| 14 | `industrial_cold_storage` | Cold storage | low | industrial | cold_storage | new/extend | — | — | preliminaries 11, site 10, substructure 12, structure 10, envelope 20, interiors 2, fire-passive 1, fire-active 5, mechanical 22, hydraulic 1, electrical 5, comms-security 1 |
| 15 | `industrial_data_centre` | Data centre | low | industrial | data_centre | new/extend | — | — | preliminaries 10, site 3, substructure 2, structure 3, envelope 3, interiors 1, fire-passive 1, fire-active 2, mechanical 22, hydraulic 1, electrical 50, comms-security 2 |
| 16 | `industrial_heavy` | Heavy manufacturing | low | industrial | heavy_manufacturing/waste_to_energy/battery_manufacturing | new/extend | — | — | preliminaries 13, site 13, substructure 16, structure 17, envelope 11, interiors 5, fire-passive 1, fire-active 4, mechanical 6, hydraulic 3, electrical 9, comms-security 1, access-egress 1 |
| 17 | `industrial_light_manufacturing` | Light manufacturing | low | industrial | manufacturing | new/extend | — | — | preliminaries 12, site 15, substructure 13, structure 13, envelope 13, interiors 8, fire-passive 1, fire-active 5, mechanical 6, hydraulic 4, electrical 8, comms-security 1, access-egress 1 |
| 18 | `industrial_dangerous_goods` | Dangerous goods store | low | industrial | dangerous_goods | new/extend | — | — | preliminaries 12, site 20, substructure 15, structure 12, envelope 12, interiors 3, fire-passive 3, fire-active 9, mechanical 4, hydraulic 3, electrical 5, comms-security 1, access-egress 1 |
| 19 | `industrial_food_processing` | Food processing | low | industrial | food_processing | new/extend | — | — | preliminaries 12, site 10, substructure 11, structure 10, envelope 13, interiors 12, fire-passive 1, fire-active 4, mechanical 14, hydraulic 6, electrical 6, comms-security 1 |
| 20 | `industrial_gmp` | Pharmaceutical GMP or cleanroom | low | industrial | pharmaceutical_gmp/cleanroom | new/extend | — | — | preliminaries 12, site 6, substructure 6, structure 7, envelope 7, interiors 16, fire-passive 2, fire-active 3, mechanical 23, hydraulic 7, electrical 9, comms-security 2 |
| 21 | `industrial_warehouse` | Warehouse or distribution | low | industrial | any | new/extend | — | — | preliminaries 12, site 20, substructure 14, structure 14, envelope 14, interiors 6, fire-passive 1, fire-active 6, mechanical 3, hydraulic 2, electrical 6, comms-security 1, access-egress 1 |
| 22 | `institution_early_childhood` | Early childhood centre | low | institution | education_early_childhood | new/extend | — | — | preliminaries 13, site 14, substructure 7, structure 5, envelope 17, interiors 16, fire-passive 1, fire-active 1.5, mechanical 9, hydraulic 7, electrical 6, comms-security 2.5, access-egress 1 |
| 23 | `institution_school` | School | low | institution | education_primary_secondary | new/extend | — | — | preliminaries 12, site 10, substructure 6, structure 12, envelope 18, interiors 15, fire-passive 1, fire-active 1.5, mechanical 8, hydraulic 4, electrical 7, comms-security 3.5, vertical-transport 1, access-egress 1 |
| 24 | `institution_tertiary` | Tertiary teaching or laboratory | low | institution | education_tertiary | new/extend | — | — | preliminaries 13, site 4, substructure 4, structure 9, envelope 11, interiors 12, fire-passive 1.5, fire-active 2.5, mechanical 18, hydraulic 7, electrical 10, comms-security 4, vertical-transport 2, access-egress 2 |
| 25 | `institution_hospital` | Hospital | low | institution | healthcare_hospital | new/extend | — | — | preliminaries 15, site 4, substructure 4, structure 8, envelope 12, interiors 15, fire-passive 1.5, fire-active 2.5, mechanical 12, hydraulic 10, electrical 9, comms-security 4, vertical-transport 2, access-egress 1 |
| 26 | `institution_medical_centre` | Medical centre | low | institution | healthcare_medical_centre | new/extend | — | — | preliminaries 15, site 5, substructure 5, structure 8, envelope 15, interiors 16, fire-passive 1, fire-active 2, mechanical 11, hydraulic 7, electrical 9, comms-security 3, vertical-transport 2, access-egress 1 |
| 27 | `institution_allied_health` | Allied health clinic | low | institution | healthcare_clinic_allied | new/extend | — | — | preliminaries 15, site 6, substructure 6, structure 6, envelope 15, interiors 20, fire-passive 1, fire-active 1.5, mechanical 10, hydraulic 6, electrical 8, comms-security 3, vertical-transport 1, access-egress 1.5 |
| 28 | `institution_religious` | Religious building | low | institution | religious | new/extend | — | — | preliminaries 13, site 8, substructure 6, structure 12, envelope 24, interiors 17, fire-passive 1, fire-active 1.5, mechanical 7, hydraulic 2, electrical 6, comms-security 1.5, access-egress 1 |
| 29 | `institution_civic` | Civic or community building | low | institution | any | new/extend | — | — | preliminaries 13, site 7, substructure 5, structure 10, envelope 18, interiors 18, fire-passive 1, fire-active 2, mechanical 10, hydraulic 4, electrical 7, comms-security 3, vertical-transport 1, access-egress 1 |
| 30 | `infrastructure_roads` | Infrastructure (road categories) | low | infrastructure | any | new/extend/refurb | — | — | preliminaries 19, site 49, structure 32 |

### Which split each class and work type reaches (storeys unknown, no basement)

| class | subclass | new | extend | refurb | remediation | advisory |
|---|---|---|---|---|---|---|
| residential | house | `residential_house_new` | `residential_extension` | `residential_extension` | none | none |
| residential | apartments | `residential_apartments_low_rise` | `residential_extension` | `residential_extension` | none | none |
| residential | townhouses | `residential_house_new` | `residential_extension` | `residential_extension` | none | none |
| residential | btr | `residential_apartments_low_rise` | `residential_extension` | `residential_extension` | none | none |
| residential | student_housing | `residential_student` | `residential_extension` | `residential_extension` | none | none |
| residential | retirement_living | `residential_house_new` | `residential_extension` | `residential_extension` | none | none |
| residential | residential_aged_care | `residential_aged_care` | `residential_extension` | `residential_extension` | none | none |
| residential | social_affordable_housing | `residential_house_new` | `residential_extension` | `residential_extension` | none | none |
| residential | other | `residential_other` | `residential_extension` | `residential_extension` | none | none |
| commercial | office | `commercial_office` | `commercial_office` | `commercial_fitout` | none | none |
| commercial | retail_shopping_centre | `commercial_shopping_centre` | `commercial_shopping_centre` | `commercial_fitout` | none | none |
| commercial | retail_standalone | `commercial_retail` | `commercial_retail` | `commercial_fitout` | none | none |
| commercial | hotel | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| commercial | food_beverage | `commercial_retail` | `commercial_retail` | `commercial_fitout` | none | none |
| commercial | serviced_office_coworking | `commercial_office` | `commercial_office` | `commercial_fitout` | none | none |
| commercial | other | `commercial_retail` | `commercial_retail` | `commercial_fitout` | none | none |
| industrial | warehouse | `industrial_warehouse` | `industrial_warehouse` | none | none | none |
| industrial | logistics_ecommerce | `industrial_high_bay` | `industrial_high_bay` | none | none | none |
| industrial | manufacturing | `industrial_light_manufacturing` | `industrial_light_manufacturing` | none | none | none |
| industrial | cold_storage | `industrial_cold_storage` | `industrial_cold_storage` | none | none | none |
| industrial | data_centre | `industrial_data_centre` | `industrial_data_centre` | none | none | none |
| industrial | dangerous_goods | `industrial_dangerous_goods` | `industrial_dangerous_goods` | none | none | none |
| industrial | heavy_manufacturing | `industrial_heavy` | `industrial_heavy` | none | none | none |
| industrial | food_processing | `industrial_food_processing` | `industrial_food_processing` | none | none | none |
| industrial | pharmaceutical_gmp | `industrial_gmp` | `industrial_gmp` | none | none | none |
| industrial | cleanroom | `industrial_gmp` | `industrial_gmp` | none | none | none |
| industrial | battery_manufacturing | `industrial_heavy` | `industrial_heavy` | none | none | none |
| industrial | waste_to_energy | `industrial_heavy` | `industrial_heavy` | none | none | none |
| industrial | other | `industrial_warehouse` | `industrial_warehouse` | none | none | none |
| institution | education_early_childhood | `institution_early_childhood` | `institution_early_childhood` | none | none | none |
| institution | education_primary_secondary | `institution_school` | `institution_school` | none | none | none |
| institution | education_tertiary | `institution_tertiary` | `institution_tertiary` | none | none | none |
| institution | healthcare_hospital | `institution_hospital` | `institution_hospital` | none | none | none |
| institution | healthcare_medical_centre | `institution_medical_centre` | `institution_medical_centre` | none | none | none |
| institution | healthcare_clinic_allied | `institution_allied_health` | `institution_allied_health` | none | none | none |
| institution | government_civic | `institution_civic` | `institution_civic` | none | none | none |
| institution | religious | `institution_religious` | `institution_religious` | none | none | none |
| institution | other | `institution_civic` | `institution_civic` | none | none | none |
| mixed | residential_retail | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| mixed | residential_commercial | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| mixed | hotel_residential | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| mixed | retail_office | `commercial_retail` | `commercial_retail` | `commercial_fitout` | none | none |
| mixed | btr_retail | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| mixed | vertical_village | `commercial_hotel` | `commercial_hotel` | `commercial_fitout` | none | none |
| mixed | other | `commercial_retail` | `commercial_retail` | `commercial_fitout` | none | none |
| infrastructure | roads_highways | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | rail_metro | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | water_utilities | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | energy_renewables | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | marine_ports | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | airports | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | telecommunications | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
| infrastructure | other | `infrastructure_roads` | `infrastructure_roads` | `infrastructure_roads` | none | none |
