# Scope defaults by class, subclass and work type
Generated on 28 October 2026 from the knowledge files on `main` (8a6a6c8 plus the working tree) by a read-only script. Regenerate rather than hand-edit. Back to [the index](../KNOWLEDGE_INDEX.md).

How to read this. `profile.Build` (`internal/profile/build.go`) calls
`Catalog.ScopeDefaults(category, subclass, work_type)` (`internal/knowledge/profile.go`) with the
reconciled header values. The rule, in order:

1. `hdr.work_type` empty → nothing. `refurb`, `remediation`, `advisory` (`scope_defaults.yaml
   empty_work_types`) → nothing.
2. A `classes[]` entry whose `class` equals the **subclass** and whose `work_type` equals the work type.
3. Else a `classes[]` entry for that subclass with `work_type: new` (so `apartments/extend` gets the
   apartments new list, `townhouses/extend` the townhouses new list).
4. Else the `categories[]` list for the subclass's building class. The category is re-derived from the
   subclass by scanning every class; the **last** class holding that subclass id wins. Subclass `other`
   exists in all six classes, so any `other` subclass resolves to the **infrastructure** list
   (`categories[5]`), whatever `hdr.building_class` says. This is a defect, recorded in the index gaps.

Each default becomes a `scope.<leaf>` row with band `suggested`, unless the user set that row `out`.
`profile.ProposedWorks` (`internal/profile/works.go`) then makes one proposed work item per in-scope
system with action from `actions.yaml work_type_defaults` (new→new, extend→new, refurb→alter,
remediation→repair, advisory→investigate) unless an applied `sys.<leaf>.action` reading or user value says
otherwise. Work items are written by `syncProposedWorks` (`internal/store/works.go`) with
`review_status: proposed`.

## Which list each class, subclass and work type gets

| class | subclass | NCC (taxonomy) | new | extend | refurb / remediation / advisory |
|---|---|---|---|---|---|
| residential | house | 1a | classes[0] (house/new): 41 systems | classes[2] (house/extend): 16 systems | empty |
| residential | apartments | 2 | classes[5] (apartments/new): 62 systems | classes[5] (apartments/new): 62 systems | empty |
| residential | townhouses | 1a | classes[1] (townhouses/new): 45 systems | classes[1] (townhouses/new): 45 systems | empty |
| residential | btr | 2 | categories[0] (residential): 62 systems | categories[0] (residential): 62 systems | empty |
| residential | student_housing | 3 | categories[0] (residential): 62 systems | categories[0] (residential): 62 systems | empty |
| residential | retirement_living | 2/9c | categories[0] (residential): 62 systems | categories[0] (residential): 62 systems | empty |
| residential | residential_aged_care | 9c | categories[0] (residential): 62 systems | categories[0] (residential): 62 systems | empty |
| residential | social_affordable_housing | 1a/2/3 | categories[0] (residential): 62 systems | categories[0] (residential): 62 systems | empty |
| residential | other | varies | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| commercial | office | 5 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | retail_shopping_centre | 6 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | retail_standalone | 6 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | hotel | 3 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | food_beverage | 6 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | serviced_office_coworking | 5 | categories[2] (commercial): 64 systems | categories[2] (commercial): 64 systems | empty |
| commercial | other | varies | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| industrial | warehouse | 7b | classes[3] (warehouse/new): 46 systems | classes[4] (warehouse/extend): 39 systems | empty |
| industrial | logistics_ecommerce | 7b | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | manufacturing | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | cold_storage | 7b | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | data_centre | 7b | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | dangerous_goods | 7a | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | heavy_manufacturing | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | food_processing | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | pharmaceutical_gmp | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | cleanroom | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | battery_manufacturing | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | waste_to_energy | 8 | categories[4] (industrial): 46 systems | categories[4] (industrial): 46 systems | empty |
| industrial | other | varies | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| institution | education_early_childhood | 9b | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | education_primary_secondary | 9b | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | education_tertiary | 9b | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | healthcare_hospital | 9a | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | healthcare_medical_centre | 5/9a | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | healthcare_clinic_allied | 5/9a | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | government_civic | 5/9b | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | religious | 9b | categories[3] (institution): 64 systems | categories[3] (institution): 64 systems | empty |
| institution | other | varies | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| mixed | residential_retail | 2/6 | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | residential_commercial | 2/5 | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | hotel_residential | 2/3 | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | retail_office | 5/6 | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | btr_retail | 2/6 | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | vertical_village | 2/9c | categories[1] (mixed): 62 systems | categories[1] (mixed): 62 systems | empty |
| mixed | other | varies | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | roads_highways | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | rail_metro | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | water_utilities | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | energy_renewables | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | marine_ports | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | airports | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | telecommunications | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |
| infrastructure | other | not_applicable | categories[5] (infrastructure): 10 systems | categories[5] (infrastructure): 10 systems | empty |

Default action for every default item: new and extend → `new`; refurb → `alter`; remediation → `repair`; advisory → `investigate` (`knowledge/works/actions.yaml work_type_defaults`). Because refurb, remediation and advisory start empty, they get **no** default work items at all.

## The lists

### `classes[0]` — subclass `house`, work type `new` (41 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `substructure.footings`, `substructure.slab-on-ground`, `substructure.termite-management`, `structure.timber`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.vehicle-doors`, `envelope.flashings-seals`, `envelope.pliable-membranes`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `mechanical.air-conditioning`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `fire-active.smoke-alarms`

### `classes[1]` — subclass `townhouses`, work type `new` (45 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `substructure.footings`, `substructure.slab-on-ground`, `substructure.termite-management`, `structure.timber`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.vehicle-doors`, `envelope.flashings-seals`, `envelope.pliable-membranes`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `mechanical.air-conditioning`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `fire-active.smoke-alarms`, `interiors.acoustic-separation`, `fire-passive.rated-elements`, `access-egress.stairs-and-ramps`, `access-egress.barriers`

### `classes[2]` — subclass `house`, work type `extend` (16 systems)

`structure.alterations`, `substructure.footings`, `structure.timber`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.flashings-seals`, `envelope.thermal-insulation`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.floor-finishes`, `interiors.painting`, `electrical.cabling-wiring`, `electrical.lighting`, `hydraulic.roof-drainage`, `fire-active.smoke-alarms`

### `classes[3]` — subclass `warehouse`, work type `new` (46 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.impact-protection`, `site.signage`, `substructure.footings`, `substructure.slab-on-ground`, `structure.steel`, `structure.precast`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.vehicle-doors`, `envelope.thermal-insulation`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.internal-doors`, `interiors.painting`, `fire-active.hydrants`, `fire-active.portable-extinguishers`, `fire-active.emergency-lighting`, `access-egress.exits`, `access-egress.accessible-access`, `access-egress.accessible-facilities`, `access-egress.maintenance-access`, `mechanical.air-conditioning`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `fire-passive.compartments`, `fire-active.fire-water`, `electrical.utility-supply`, `comms-security.carrier-lead-in`

### `classes[4]` — subclass `warehouse`, work type `extend` (39 systems)

`structure.alterations`, `site.stormwater-detention`, `site.services-connections`, `site.vehicle-access`, `site.impact-protection`, `site.signage`, `substructure.footings`, `substructure.slab-on-ground`, `structure.steel`, `structure.precast`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.vehicle-doors`, `envelope.thermal-insulation`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.internal-doors`, `interiors.painting`, `fire-active.hydrants`, `fire-active.portable-extinguishers`, `fire-active.emergency-lighting`, `access-egress.exits`, `access-egress.accessible-access`, `access-egress.accessible-facilities`, `access-egress.maintenance-access`, `mechanical.air-conditioning`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`

### `classes[5]` — subclass `apartments`, work type `new` (62 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.waste-storage`, `site.public-domain`, `site.adjoining-structures`, `substructure.basement-retention`, `substructure.footings`, `substructure.damp-waterproofing`, `structure.concrete`, `envelope.external-walls`, `envelope.windows-doors`, `envelope.external-waterproofing`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `envelope.facade-acoustics`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `interiors.acoustic-separation`, `fire-passive.rated-elements`, `fire-passive.compartments`, `fire-passive.penetrations`, `fire-active.smoke-alarms`, `fire-active.detection`, `fire-active.hydrants`, `fire-active.emergency-lighting`, `fire-active.portable-extinguishers`, `access-egress.exits`, `access-egress.stairs-and-ramps`, `access-egress.barriers`, `access-egress.egress-doors`, `access-egress.accessible-access`, `access-egress.livable-housing`, `mechanical.air-conditioning`, `mechanical.ventilation`, `mechanical.local-exhaust`, `mechanical.carpark-ventilation`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `comms-security.access-control`, `vertical-transport.passenger-lifts`, `vertical-transport.lift-shafts-pits`

### `categories[0]` — category `residential` (62 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.waste-storage`, `site.public-domain`, `site.adjoining-structures`, `substructure.basement-retention`, `substructure.footings`, `substructure.damp-waterproofing`, `structure.concrete`, `envelope.external-walls`, `envelope.windows-doors`, `envelope.external-waterproofing`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `envelope.facade-acoustics`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `interiors.acoustic-separation`, `fire-passive.rated-elements`, `fire-passive.compartments`, `fire-passive.penetrations`, `fire-active.smoke-alarms`, `fire-active.detection`, `fire-active.hydrants`, `fire-active.emergency-lighting`, `fire-active.portable-extinguishers`, `access-egress.exits`, `access-egress.stairs-and-ramps`, `access-egress.barriers`, `access-egress.egress-doors`, `access-egress.accessible-access`, `access-egress.livable-housing`, `mechanical.air-conditioning`, `mechanical.ventilation`, `mechanical.local-exhaust`, `mechanical.carpark-ventilation`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `comms-security.access-control`, `vertical-transport.passenger-lifts`, `vertical-transport.lift-shafts-pits`

### `categories[1]` — category `mixed` (62 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.waste-storage`, `site.public-domain`, `site.adjoining-structures`, `substructure.basement-retention`, `substructure.footings`, `substructure.damp-waterproofing`, `structure.concrete`, `envelope.external-walls`, `envelope.windows-doors`, `envelope.external-waterproofing`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `envelope.facade-acoustics`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `interiors.acoustic-separation`, `fire-passive.rated-elements`, `fire-passive.compartments`, `fire-passive.penetrations`, `fire-active.smoke-alarms`, `fire-active.detection`, `fire-active.hydrants`, `fire-active.emergency-lighting`, `fire-active.portable-extinguishers`, `access-egress.exits`, `access-egress.stairs-and-ramps`, `access-egress.barriers`, `access-egress.egress-doors`, `access-egress.accessible-access`, `access-egress.livable-housing`, `mechanical.air-conditioning`, `mechanical.ventilation`, `mechanical.local-exhaust`, `mechanical.carpark-ventilation`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `comms-security.access-control`, `vertical-transport.passenger-lifts`, `vertical-transport.lift-shafts-pits`

### `categories[2]` — category `commercial` (64 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.waste-storage`, `site.public-domain`, `site.adjoining-structures`, `substructure.basement-retention`, `substructure.footings`, `substructure.damp-waterproofing`, `structure.concrete`, `envelope.external-walls`, `envelope.windows-doors`, `envelope.external-waterproofing`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `fire-passive.rated-elements`, `fire-passive.compartments`, `fire-passive.penetrations`, `fire-active.detection`, `fire-active.hydrants`, `fire-active.emergency-lighting`, `fire-active.portable-extinguishers`, `access-egress.exits`, `access-egress.stairs-and-ramps`, `access-egress.barriers`, `access-egress.egress-doors`, `access-egress.accessible-access`, `mechanical.air-conditioning`, `mechanical.ventilation`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `comms-security.access-control`, `vertical-transport.passenger-lifts`, `vertical-transport.lift-shafts-pits`, `fire-active.sprinklers`, `fire-active.occupant-warning`, `mechanical.air-distribution`, `mechanical.central-plant`, `mechanical.controls-bms`, `access-egress.accessible-facilities`, `comms-security.cctv`

### `categories[3]` — category `institution` (64 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.waste-storage`, `site.public-domain`, `site.adjoining-structures`, `substructure.basement-retention`, `substructure.footings`, `substructure.damp-waterproofing`, `structure.concrete`, `envelope.external-walls`, `envelope.windows-doors`, `envelope.external-waterproofing`, `envelope.thermal-insulation`, `envelope.thermal-performance`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `fire-passive.rated-elements`, `fire-passive.compartments`, `fire-passive.penetrations`, `fire-active.detection`, `fire-active.hydrants`, `fire-active.emergency-lighting`, `fire-active.portable-extinguishers`, `access-egress.exits`, `access-egress.stairs-and-ramps`, `access-egress.barriers`, `access-egress.egress-doors`, `access-egress.accessible-access`, `mechanical.air-conditioning`, `mechanical.ventilation`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.utility-supply`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `comms-security.carrier-lead-in`, `comms-security.structured-cabling`, `comms-security.access-control`, `vertical-transport.passenger-lifts`, `vertical-transport.lift-shafts-pits`, `fire-active.sprinklers`, `fire-active.occupant-warning`, `mechanical.air-distribution`, `mechanical.central-plant`, `mechanical.controls-bms`, `access-egress.accessible-facilities`, `comms-security.cctv`

### `categories[4]` — category `industrial` (46 systems)

`site.survey-boundaries`, `site.earthworks`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.landscape`, `site.vehicle-access`, `site.impact-protection`, `site.signage`, `substructure.footings`, `substructure.slab-on-ground`, `structure.steel`, `structure.precast`, `envelope.external-walls`, `envelope.roof-coverings`, `envelope.windows-doors`, `envelope.vehicle-doors`, `envelope.thermal-insulation`, `interiors.walls-linings`, `interiors.ceilings`, `interiors.wet-area-waterproofing`, `interiors.tiling`, `interiors.floor-finishes`, `interiors.internal-doors`, `interiors.painting`, `fire-active.hydrants`, `fire-active.portable-extinguishers`, `fire-active.emergency-lighting`, `access-egress.exits`, `access-egress.accessible-access`, `access-egress.accessible-facilities`, `access-egress.maintenance-access`, `mechanical.air-conditioning`, `mechanical.local-exhaust`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `hydraulic.roof-drainage`, `electrical.switchboards`, `electrical.cabling-wiring`, `electrical.lighting`, `electrical.metering`, `fire-passive.compartments`, `fire-active.fire-water`, `electrical.utility-supply`, `comms-security.carrier-lead-in`

### `categories[5]` — category `infrastructure` (10 systems)

`site.survey-boundaries`, `site.earthworks`, `site.ground-conditions`, `site.stormwater-detention`, `site.services-connections`, `site.erosion-sediment-control`, `site.vehicle-access`, `substructure.footings`, `structure.concrete`, `electrical.utility-supply`

Note: `categories[0]` residential and `categories[1]` mixed are the YAML anchor `&id001`, the same list as `classes[5]` apartments/new. `categories[3]` institution is `&id002`, the commercial list. `categories[4]` industrial is `&id003`, the warehouse/new list.

### Preset `typical_fit_out` — Typical fit-out (`scope_defaults.yaml presets`)

Offered by the scope picker as one click; never applied automatically. `interiors.walls-linings`, `interiors.ceilings`, `interiors.floor-finishes`, `interiors.joinery`, `interiors.internal-doors`, `interiors.internal-glazing`, `interiors.painting`, `mechanical.air-conditioning`, `mechanical.air-distribution`, `electrical.lighting`, `electrical.cabling-wiring`, `comms-security.structured-cabling`, `fire-active.sprinklers`, `fire-active.detection`, `fire-active.occupant-warning`, `fire-active.emergency-lighting`, `hydraulic.cold-water`, `hydraulic.hot-water`, `hydraulic.sanitary`, `comms-security.access-control`

## Words that imply systems (`scope_defaults.yaml implied_by`)

`profile.ImpliedReadings` (`internal/profile/implied.go`) matches each pattern (case-insensitive, Go RE2) against a read passage and adds each system as an `implied.<leaf>` suggestion. `withImpliedScope` (`internal/profile/build.go`) adds a `scope.<leaf>` row only when no row exists; it never overrides the user or document evidence. Only documents the profile reads produce these (kinds in `data/profile/reading.json`).

| id | pattern | systems |
|---|---|---|
| wet-room | `\b(?:bath(?:room)?s?\|ensuites?\|laundr(?:y\|ies)\|powder rooms?)\b` | `interiors.wet-area-waterproofing`, `interiors.tiling`, `hydraulic.sanitary`, `hydraulic.cold-water`, `hydraulic.hot-water`, `mechanical.local-exhaust` |
| kitchen | `\bkitchens?\b` | `interiors.joinery`, `hydraulic.sanitary`, `hydraulic.cold-water`, `hydraulic.hot-water`, `mechanical.local-exhaust` |
| robe | `\b(?:walk-in robes?\|wardrobes?\|built-in robes?)\b` | `interiors.joinery` |
| bedroom | `\b(?:bedrooms?\|study\|studies)\b` | `interiors.internal-doors` |
| stair | `\b(?:internal )?stair(?:s\|case\|cases)?\b` | `access-egress.stairs-and-ramps` |
| party-wall | `\b(?:party\|separating\|common) walls?\b` | `site.adjoining-structures`, `fire-passive.compartments`, `interiors.acoustic-separation` |
| strip-out | `\bstrip[- ]?out\b\|\bdemoli(?:tion\|sh)\b` | `site.demolition` |
| masonry | `\bmasonry\b\|\bre-?point(?:ing)?\b\|\bbrickwork\b` | `structure.masonry` |
| energy-rating | `\bBASIX\b\|\bNatHERS\b` | `envelope.thermal-performance` |
