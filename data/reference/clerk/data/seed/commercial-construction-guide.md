---
tier: overlay
loaded_by: "building_class: commercial"
applies_to_classes: [commercial]
applies_to_work_types: [new, refurb, extend]
applies_to_subclasses: [office, retail_shopping_centre, retail_standalone, serviced_office_coworking]
topics: [commercial, construction, ncc, services, fitout]
summary: "Commercial office, retail and coworking construction guide: classification, fire, egress, accessibility, base-build versus fitout, services, ESM, sustainability, programming, QA, handover and refurbishment risks."
required_by: {create-pmp: 1}
---

# Commercial Construction Guide

This guide is routed to office, retail and serviced-office/coworking projects.
References to Class 7, 8 or 9 buildings below explain classification and
interface boundaries only; they do not provide industrial or institutional
project coverage. Use `industrial-construction-guide.md` for industrial work,
`institution-construction-guide.md` for institutional work, and
`mixed-use-construction-guide.md` for mixed-use work.

## Commercial Building Classifications (Class 5–9)

Commercial and institutional buildings in Australia are classified under the National Construction Code into five primary classes, each with distinct regulatory requirements for fire safety, access, services, and construction type.

Class 5 buildings are office buildings used for professional or commercial purposes, including government offices, corporate headquarters, and co-working spaces. Class 6 buildings are shops or retail premises, including shopping centres, restaurants, cafes, showrooms, and hairdressers. Class 7 is split into Class 7a (carparks) and Class 7b (warehouses and storage buildings used for wholesale or storage of goods). Class 8 buildings are laboratories, factories, workshops, and production facilities not classified as Class 7. Class 9 is divided into Class 9a (healthcare buildings including hospitals, day surgeries, and aged care), Class 9b (assembly buildings including theatres, schools, community halls, and places of worship), and Class 9c (residential care buildings for persons requiring personal care).

A building may contain multiple classifications where different parts serve different purposes. In mixed-use developments, each part is classified separately and must meet the requirements for its respective class. The most stringent fire safety requirements typically govern shared elements such as egress paths and structural systems.

Reference: NCC 2022 Volume 1, Part A6 — Building Classifications
Applies to: All Class 5–9 commercial and institutional buildings
Related standards: NCC Volume 1 — applicable state appendices for jurisdictional variations

Common pitfalls:
- Incorrectly classifying a mixed-use building as a single class rather than classifying each part separately
- Misclassifying aged care facilities — Class 9a vs 9c has materially different fire safety and access requirements
- Treating a commercial kitchen or food production area as Class 6 when it may be Class 8
- Not recognising that a Class 7a carpark attached to a Class 5 building may require fire separation

## Type of Construction and Fire Resistance Levels

The NCC prescribes fire resistance levels (FRL) based on building classification, effective height, and type of construction. For commercial buildings, Type A construction is required for buildings of 4 or more storeys or with an effective height greater than 25 metres. Type B construction is permitted for lower-rise commercial buildings meeting specific criteria. Type C construction applies to single-storey buildings of limited size.

Fire resistance levels are expressed as adequacy/integrity/insulation in minutes (e.g., 120/120/120). Structural adequacy relates to load-bearing capacity during fire, integrity relates to prevention of flame passage, and insulation relates to limiting temperature rise on the unexposed face.

For Type A construction in commercial buildings, typical minimum FRLs are 120/120/120 for load-bearing walls and columns, and 120/120/120 for floors. Non-load-bearing external walls require varying FRLs depending on distance from the fire-source feature (boundary). These requirements increase for buildings of greater effective height.

Sprinkler installation can provide concessions on fire resistance requirements in some cases, but cannot be used as a substitute for structural fire resistance in Type A construction above certain heights.

Reference: NCC 2022 Volume 1, Section C — Fire Resistance
Applies to: All Class 5–9 buildings
Related standards: AS 1530.4 (fire resistance tests), AS 1530.1 (combustibility tests)

Common pitfalls:
- Confusing effective height with total building height — effective height is measured from the lowest storey providing direct egress to a road
- Assuming sprinklers can fully offset FRL requirements when concessions only apply in specific circumstances
- Not checking state-specific variations that may modify deemed-to-satisfy FRL tables
- Underestimating FRL requirements for basement carparks under commercial buildings

## Fire Detection, Alarm, and Suppression Systems

Commercial buildings require fire detection and alarm systems designed to AS 1670.1, with the complexity of the system increasing with building classification and risk profile. Class 5 and 6 buildings of limited size may use smoke alarms complying with AS 3786, while larger buildings and all Class 9a/9c buildings require comprehensive addressable fire detection and alarm systems with brigade connection.

Automatic sprinkler systems complying with AS 2118.1 are mandatory for most commercial buildings above certain size and height thresholds. Class 9a healthcare buildings and Class 9c residential care buildings have specific sprinkler requirements due to the vulnerability of occupants. Shopping centres (Class 6) typically require sprinklers throughout due to high fire loads and occupant density.

Fire hydrant systems complying with AS 2419.1 are required in commercial buildings exceeding certain floor areas and effective heights. Combined systems using AS 2118.6 can serve both sprinkler and hydrant functions in smaller buildings, reducing infrastructure duplication.

Fire brigade booster connections must comply with AS 2419.3 and be located to provide ready access for fire brigade vehicles. The location, signage, and connection type must meet local fire brigade requirements which vary by state.

Reference: NCC 2022 Volume 1, Section E — Services and Equipment
Applies to: All Class 5–9 buildings
Related standards: AS 1670.1 (fire detection systems), AS 2118.1 (sprinkler systems), AS 2419.1 (fire hydrants), AS 3786 (smoke alarms)

Common pitfalls:
- Designing fire detection without early engagement with the fire brigade — connection requirements vary by jurisdiction
- Not accounting for sprinkler booster pump power requirements in the electrical design
- Overlooking the need for emergency warning and intercom systems (EWIS) in large Class 9b assembly buildings
- Failing to coordinate ceiling design with sprinkler head locations, leading to obstructed spray patterns

## Egress Design for Commercial Buildings

Egress design in commercial buildings must provide safe paths of travel from any point in the building to an exit and then to a road or open space. The NCC prescribes maximum travel distances, minimum exit widths, and the number of required exits based on building classification, floor area, and occupant numbers.

For Class 5 office buildings, maximum travel distances to an exit are typically 20 metres in a single-direction path and 40 metres where alternative exits are available. Class 6 retail buildings have similar travel distance limits but require wider exits due to higher occupant densities. Class 9a healthcare buildings have the most stringent egress requirements due to non-ambulant occupants, requiring horizontal evacuation zones and smoke compartments.

Exit stairways must be fire-isolated with the appropriate FRL and must discharge directly to a road or open space. Pressurisation of stairways is required in buildings with an effective height of more than 25 metres. Stairway widths must accommodate the calculated occupant load plus provide capacity for counterflow where firefighters may need to enter.

Emergency lighting to AS 2293 must be provided in all required exit paths, with illumination of not less than 1 lux at floor level. Exit signs must be illuminated and visible from within the required distance. Emergency lighting must have a minimum 90-minute battery backup duration.

Reference: NCC 2022 Volume 1, Section D — Access and Egress
Applies to: All Class 5–9 buildings
Related standards: AS 2293 (emergency lighting), AS 1428.1 (access for people with disability), AS 1657 (fixed platforms and stairways)

Common pitfalls:
- Calculating occupant load incorrectly, particularly in open-plan offices where future tenancy fit-out may change densities
- Not providing adequate egress from mezzanine levels within retail or warehouse spaces
- Failing to pressurize stairways in buildings just over the 25m effective height threshold
- Overlooking that exit doors must swing in the direction of egress and not obstruct the exit path when open

## Accessibility and DDA Compliance

The Disability Discrimination Act 1992 (DDA) and the Disability (Access to Premises — Buildings) Standards 2010 establish federal requirements for accessible building design. The NCC references AS 1428.1 for detailed technical requirements. These requirements apply to all new Class 5–9 buildings and to existing buildings undergoing substantial refurbishment.

Key accessibility requirements for commercial buildings include: accessible entrances from the principal pedestrian entrance and any accessible carparking; accessible paths of travel to and within all areas normally used by the public or building occupants; accessible sanitary facilities on every level containing sanitary compartments; accessible lifts where the building has more than one storey; hearing augmentation in rooms designed for public address; and tactile ground surface indicators at stairways and hazardous locations.

For Class 6 retail, accessible counters, point-of-sale areas, and fitting rooms are required where provided. Class 9a healthcare buildings require accessible patient bedrooms, treatment areas, and sanitary facilities meeting enhanced accessibility provisions. Class 9b assembly buildings require accessible seating positions, wheelchair spaces, and companion seating.

An Access Consultant accredited under the Association of Consultants in Access Australia (ACAA) should be engaged early in design to conduct an access audit and provide a Disability Access Report. Many local councils require this report as part of the development application process.

Reference: Disability (Access to Premises — Buildings) Standards 2010; NCC 2022 Volume 1, Part D3
Applies to: All new and substantially refurbished Class 5–9 buildings
Related standards: AS 1428.1 (general access), AS 1428.2 (enhanced access), AS 1428.4.1 (tactile indicators), AS 1428.5 (communication for people who are deaf or hearing impaired)

Common pitfalls:
- Designing accessible sanitary facilities that meet dimensional minimums but fail practical usability testing
- Not engaging an access consultant until after design development, leading to costly rework
- Overlooking hearing augmentation requirements in meeting rooms and assembly spaces
- Assuming existing building exemptions under the DDA without obtaining a formal unjustifiable hardship determination

## Structural Systems for Commercial Buildings

Commercial building structural systems are selected based on building height, span requirements, load paths, ground conditions, and construction methodology. Common structural systems in Australian commercial construction include reinforced concrete frames, post-tensioned concrete, structural steel frames, and composite steel-concrete systems.

Reinforced concrete frames designed to AS 3600 are the most common system for commercial buildings in Australia, particularly for buildings up to 20 storeys. Post-tensioned concrete slabs are widely used for office buildings, providing longer spans with thinner slab profiles, which reduces floor-to-floor height and overall building height. This system is cost-effective for regular grid layouts typical of Class 5 office buildings.

Structural steel frames designed to AS 4100 are preferred for industrial buildings (Class 7b, Class 8), long-span retail spaces, and buildings where speed of erection is critical. Composite steel-concrete systems using metal decking combine the erection speed of steel with the fire resistance of concrete. Steel structures typically require applied fire protection (intumescent coatings, board systems, or spray-applied cementitious coatings) to achieve required FRLs.

Foundation systems for commercial buildings range from pad footings and strip footings for low-rise buildings on competent ground, to bored piles and driven piles for multi-storey buildings or poor ground conditions. Geotechnical investigation to AS 1726 is essential before foundation design. Basement construction for carparking commonly requires contiguous piled walls, diaphragm walls, or secant pile walls with associated temporary and permanent retention systems.

Reference: NCC 2022 Volume 1, Part B1 — Structural Provisions
Applies to: All Class 5–9 buildings
Related standards: AS 3600 (concrete structures), AS 4100 (steel structures), AS/NZS 1170 series (structural design actions), AS 2159 (piling), AS 1726 (geotechnical site investigations)

Common pitfalls:
- Not engaging a geotechnical engineer early enough — inadequate site investigation leads to foundation redesign during construction
- Underestimating the cost impact of applied fire protection on structural steel in Type A construction
- Failing to coordinate structural grid spacing with carpark geometry in basements, leading to inefficient parking layouts
- Not accounting for construction loads and temporary propping in post-tensioned concrete design sequences

## Facade and Curtain Wall Systems

Facade systems for commercial buildings range from masonry and precast concrete to unitised curtain walls and ventilated rainscreen systems. System selection depends on building height, exposure conditions, performance requirements (thermal, acoustic, weather), and project budget.

Curtain wall systems must comply with AS 2047 for weather performance and AS 1288 for glazing selection and installation. Facade testing to AS 4284 should be conducted for curtain wall systems to verify weather resistance, air infiltration, structural adequacy under wind loads, and water penetration resistance. Mock-up testing of a representative facade panel is standard practice for buildings above 10 storeys.

Following the Grenfell Tower fire and the combustible cladding crisis in Australia, cladding material selection is subject to heightened regulatory scrutiny. Aluminium composite panels with polyethylene cores are prohibited on Type A and Type B construction buildings. Compliance with NCC Section C, including fire hazard properties of materials, must be verified through testing to AS 1530.1 (combustibility) and AS 1530.3 (fire hazard properties). State cladding audit programs have additional requirements — Victoria's Cladding Safety Victoria (CSV) and NSW's Project Remediate are the most established.

Waterproofing of facade penetrations, window reveals, and curtain wall perimeters is critical. Facade waterproofing interfaces with the structural frame must accommodate building movement (slab deflection, thermal movement, seismic drift) through properly designed movement joints with appropriate sealant systems.

Reference: NCC 2022 Volume 1, Specification C1.1 and Part C2 — Fire Hazard Properties
Applies to: All Class 5–9 buildings, particularly buildings of Type A construction
Related standards: AS 2047 (windows), AS 1288 (glass selection), AS 4284 (facade testing), AS 1530.1 (combustibility), AS 1530.3 (fire hazard properties)

Common pitfalls:
- Using non-compliant cladding materials without adequate testing evidence — resulting in costly remediation or demolition
- Not conducting facade mock-up testing for high-rise curtain wall systems, leading to water ingress discovered only after installation
- Failing to design adequate movement joints at facade-to-structure interfaces, causing cracking and water penetration
- Not engaging a facade consultant for buildings above 5 storeys, leading to performance issues with complex facade systems

## Base Building vs Fitout — Scope Delineation

In commercial leasing, the distinction between base building works and tenant fitout is fundamental to cost allocation, design responsibility, and regulatory compliance. Base building comprises the core structure, building envelope, common areas, central services, and landlord-provided infrastructure up to agreed demarcation points.

Typical base building scope includes: structural frame and slabs; facade and roofing; common area finishes to lobbies, corridors, amenities, and lift cars; fire services to common areas and base building risers; air-conditioning to shell and core with capacity allocated per tenancy; electrical distribution to floor distribution boards with metered tenant supply; hydraulic services to building risers and common area amenities; lifts and escalators; carparking and loading dock; landscaping and external works.

Tenant fitout scope typically includes: partitioning and internal walls; floor, wall, and ceiling finishes; supplementary air-conditioning (e.g., additional splits for server rooms); tenant lighting and power distribution from the floor distribution board; data and communications cabling; security and access control within the tenancy; signage; and any tenant-specific services.

The Agreement to Lease (ATL) and fitout guide define demarcation points for each service. Common demarcation approaches include: electrical at the floor distribution board; mechanical at the VAV box or fan coil unit; hydraulic at the riser tap-off; fire services at the floor isolation valve. Clear scope documentation at lease negotiation prevents disputes during fitout.

Reference: Property Council of Australia — Fitout Guide and Base Building Standards
Applies to: Class 5 (office), Class 6 (retail)
Related standards: AS/NZS 3000 (Wiring Rules for electrical demarcation), AS 1668.2 (mechanical ventilation requirements)

Common pitfalls:
- Ambiguous ATL scope definitions leading to disputes over who pays for services upgrades needed to support tenant fitout loads
- Not verifying that base building air-conditioning capacity can support the tenant's planned occupancy density and heat loads
- Assuming base building fire services will accommodate tenant fitout without modification — high-density or high-risk tenancies often require supplementary protection
- Tenant fitout designs proceeding before base building drawings are issued, leading to clashes and abortive work

## Commercial HVAC and Mechanical Services

Mechanical services in commercial buildings must comply with AS 1668.2 for ventilation rates and NCC Section F for energy efficiency. The mechanical system is typically the single largest services cost in a commercial building and the largest consumer of operational energy.

Common HVAC system types for Australian commercial buildings include: variable air volume (VAV) systems for large open-plan offices, providing zoned comfort control and energy efficiency; fan coil units (FCU) for perimeter zones, hotel rooms, and spaces requiring individual control; chilled beam systems (active and passive) for high-performance office buildings seeking reduced energy consumption; VRF/VRV systems for smaller commercial buildings and fitout applications where central plant is not available.

Central plant typically comprises water-cooled chillers with cooling towers for buildings above 5,000 square metres, or air-cooled chillers for smaller buildings. Heat rejection through cooling towers must comply with public health regulations for Legionella management, including registration, maintenance, and water treatment programs as required by state health regulations.

Mechanical ventilation rates are prescribed by AS 1668.2 based on occupancy type and density. Outdoor air quantities must provide adequate dilution of indoor pollutants while managing energy consumption. CO2 monitoring is increasingly used for demand-controlled ventilation in variable-occupancy spaces.

Exhaust systems for carparks, commercial kitchens, and plant rooms must comply with AS 1668.1 for fire and smoke control requirements. Kitchen exhaust systems require fire-rated ductwork and compliance with state-specific planning requirements for odour and noise discharge.

Reference: NCC 2022 Volume 1, Section J — Energy Efficiency; AS 1668.2 (ventilation)
Applies to: All Class 5–9 buildings
Related standards: AS 1668.1 (fire and smoke control), AS 1668.4 (natural ventilation), AS/NZS 3823 (air conditioner performance), AS 4254 (ductwork)

Common pitfalls:
- Undersizing cooling plant by not accounting for diversity correctly — actual peak loads often exceed design calculations due to IT equipment and occupant density growth
- Not planning for Legionella compliance on cooling towers from project inception — registration and design requirements vary by state
- Failing to coordinate ceiling void depth between structural, mechanical, and fire services, resulting in inadequate space for ductwork routing
- Designing to minimum ventilation rates without considering indoor air quality expectations of premium-grade tenants

## Commercial Electrical Services

Electrical services design for commercial buildings must comply with AS/NZS 3000 (Wiring Rules) and the relevant state electrical safety legislation. The electrical system encompasses high-voltage supply, switchboard distribution, power reticulation, lighting, emergency and exit lighting, and telecommunications infrastructure.

Power supply to commercial buildings is typically via a dedicated high-voltage connection from the distribution network service provider (e.g., Ausgrid in NSW, CitiPower in Melbourne). The building provides a main switchroom housing a transformer and main switchboard, with sub-distribution boards on each floor. Transformer sizing must account for total building demand including tenant fitout loads, mechanical plant, lifts, and common area services, with diversity factors applied per AS/NZS 3000.

Lighting design must comply with AS/NZS 1680 for required illumination levels by space type (e.g., 320 lux for general office areas, 160 lux for corridors, 40 lux for carparks). NCC Section J energy efficiency requirements set maximum illumination power densities (W/m²) to limit energy consumption. LED technology is now standard for all commercial lighting applications.

Emergency and exit lighting must comply with AS 2293 series, with emergency luminaires providing minimum 1 lux at floor level along required exit paths. Testing and maintenance of emergency lighting systems is required at prescribed intervals per AS 2293.2.

Lightning protection systems to AS 1768 are required for buildings where the risk assessment indicates the need. Taller commercial buildings in exposed locations typically require a full lightning protection system with air terminals, down conductors, and earthing system.

Reference: NCC 2022 Volume 1, Part G — Ancillary Provisions (electrical installations)
Applies to: All Class 5–9 buildings
Related standards: AS/NZS 3000 (Wiring Rules), AS/NZS 3008 (cable selection), AS/NZS 1680 (interior lighting), AS 2293 (emergency lighting), AS 1768 (lightning protection)

Common pitfalls:
- Not reserving adequate switchroom and riser space early in design — electrical infrastructure requires significant floor area that competes with lettable space
- Underestimating tenant power density requirements, particularly for technology-heavy tenancies and commercial kitchens
- Not coordinating metering strategy with the network provider early — sub-metering requirements for tenancies affect switchboard design
- Failing to provide adequate electrical capacity for future electric vehicle charging in carparks

## Commercial Hydraulic Services

Hydraulic services in commercial buildings encompass potable water supply, sanitary drainage, stormwater drainage, trade waste systems, fire services water supply, and gas installations. Design must comply with AS/NZS 3500 series and the relevant state plumbing regulations.

Potable water supply systems for multi-storey commercial buildings typically require booster pump sets to provide adequate pressure to upper floors. System design must account for peak simultaneous demand, residual pressure requirements at the highest fixtures, and water authority connection size limitations. Backflow prevention devices are required at the water authority connection and at each tenancy or zone where cross-contamination risk exists.

Sanitary drainage design must accommodate the fixture unit loads for the building's intended occupancy. Commercial kitchens, laboratories, and medical facilities require trade waste treatment (grease traps, sediment traps, neutralising pits) before connection to the sewer. Trade waste agreements with the water authority must be in place before operation.

Stormwater drainage must be designed to handle the design storm event for the building's location and catchment area. Overflow provisions must ensure that if the primary stormwater system is blocked or overloaded, water is directed away from the building without causing structural damage or flooding of occupied areas. On-site detention (OSD) is required in most urban local government areas to limit post-development peak stormwater flows.

Hot water systems for commercial buildings should consider central vs distributed generation, solar pre-heat obligations under state energy efficiency requirements, and Legionella management in stored water systems. Tempering valves are required to limit delivery temperatures at accessible outlets.

Reference: AS/NZS 3500.0–.4 (Plumbing and Drainage); NCC 2022 Volume 3 (Plumbing Code of Australia)
Applies to: All Class 5–9 buildings
Related standards: AS/NZS 5601 (gas installations), local water authority requirements, state plumbing regulations

Common pitfalls:
- Not engaging the water authority early to confirm available supply pressure and connection size — this can dictate booster pump requirements and project cost
- Overlooking trade waste requirements for commercial kitchens, resulting in non-compliant connections and authority stop-work orders
- Undersizing stormwater overflow provisions — relying solely on the primary system without adequate redundancy for extreme events
- Not coordinating hydraulic riser locations with architectural core design, leading to riser clashes and loss of lettable area

## Vertical Transport — Lifts and Escalators

Lift systems in commercial buildings are a critical design element affecting building efficiency, occupant experience, and construction cost. Lift traffic analysis must be conducted during concept design to determine the number of lifts, speed, and configuration required to meet acceptable wait times and handling capacity.

For Class 5 office buildings, acceptable peak wait times are typically 25–30 seconds for A-grade buildings and 30–40 seconds for B-grade. Handling capacity should serve at least 12–15% of the building population within a 5-minute peak period. Destination control systems are now standard for office buildings above 10 storeys, improving handling capacity by 20–30% compared to conventional collective control.

Lift installations must comply with AS 1735.1.2 for passenger and goods-passenger lifts. At least one lift must comply with AS 1735.12 for accessibility requirements, providing minimum car dimensions, tactile and Braille controls, audible floor indicators, and contrasting finishes.

Goods lifts and car lifts for basement carparks must be sized for the intended loads and vehicles. Stretcher-capable lifts are required in buildings where Class 9a healthcare uses are present and may be required in other buildings based on local authority requirements.

Escalators to AS 1735.2 are common in retail (Class 6) buildings, particularly shopping centres. Escalator design must account for fire safety requirements including fire-rated enclosures or shutters where escalators penetrate fire-rated floors.

Reference: NCC 2022 Volume 1, Part E2 — Lifts
Applies to: All multi-storey Class 5–9 buildings
Related standards: AS 1735.1.2 (passenger lifts), AS 1735.2 (escalators), AS 1735.12 (lift accessibility)

Common pitfalls:
- Not conducting lift traffic analysis early enough — adding lift shafts after structural design is extremely costly
- Specifying lifts without considering future building use changes — conversion from single-tenant to multi-tenant can dramatically increase peak demand
- Failing to provide a firefighter lift complying with AS 1735.1.2 requirements in buildings where required
- Not coordinating lift shaft dimensions with the specific manufacturer's requirements, leading to shaft modifications during construction

## Acoustics in Commercial Buildings

Acoustic design in commercial buildings must achieve recommended sound levels prescribed by AS/NZS 2107 and manage noise intrusion, inter-tenancy transmission, and mechanical plant noise. Poor acoustic performance is a leading cause of tenant complaints and can render premium office space unlettable.

For Class 5 office buildings, recommended internal ambient noise levels range from 35–40 dB(A) for general open-plan offices to 30–35 dB(A) for private offices and meeting rooms. These levels must be achieved considering HVAC system noise, external noise intrusion, and inter-tenancy sound transmission.

Inter-tenancy acoustic separation must meet NCC requirements for airborne sound insulation (Rw rating) and impact sound insulation (Ln,w rating) where applicable. For adjacent tenancies in office buildings, a minimum Rw of 45 is typical, increasing to Rw 50–55 where confidential spaces adjoin common areas. Floor-to-floor transmission is often overlooked — lightweight raised access floors with inadequate acoustic treatment can provide flanking paths that bypass partition walls.

Mechanical plant noise control requires attention at the source (equipment selection and vibration isolation), in the path (duct lining, attenuators, acoustic louvres), and at the receiver (room absorption). Cooling tower noise in particular can cause planning consent issues if not adequately addressed in the acoustic design.

Reference: AS/NZS 2107 (recommended design sound levels)
Applies to: All Class 5–9 buildings, particularly Class 5 (office) and Class 9b (assembly)
Related standards: NCC 2022 Volume 1, Part F5 — Sound Transmission and Insulation; AS 3671 (road traffic noise)

Common pitfalls:
- Not commissioning acoustic testing during construction to verify Rw ratings before finishes are installed — rectifying acoustic deficiencies after completion is very costly
- Overlooking flanking sound paths through ceiling voids, raised floors, and services penetrations
- Not specifying vibration isolation for mechanical plant early enough — inadequate isolation allows structure-borne noise to propagate to occupied spaces
- Assuming that NCC minimum acoustic requirements will satisfy premium-grade tenant expectations — they often will not

## Essential Safety Measures (ESM)

Essential safety measures are the fire safety systems and features within a building that must be maintained in operational condition throughout the building's life. Each state and territory has legislation requiring building owners to maintain ESMs and submit annual essential safety measures reports (known as Annual Fire Safety Statements in NSW and Annual Essential Safety Measures Reports in Victoria).

The essential safety measures for a commercial building are determined at the time of the occupancy permit (or occupation certificate) and typically include: fire detection and alarm systems, sprinkler systems, fire hydrant systems, emergency lighting, exit signs, fire doors, fire-rated construction, mechanical smoke exhaust, stairway pressurisation, fire brigade booster connections, portable fire extinguishers, fire hose reels, fire control centres, and building occupant warning systems.

Maintenance must be carried out by appropriately qualified and licensed practitioners in accordance with the relevant Australian Standards. AS 1851 provides the framework for routine maintenance frequencies and testing requirements for all fire protection systems. Maintenance records must be retained and available for inspection by the relevant building authority.

Failure to maintain ESMs is a significant compliance risk. Building owners and managers face penalties for non-compliance, and insurance coverage may be voided if ESMs are found to be non-operational at the time of a fire event.

Reference: State building legislation (e.g., Environmental Planning and Assessment Regulation 2021 (NSW), Building Regulations 2018 (VIC))
Applies to: All Class 5–9 buildings with an occupancy permit
Related standards: AS 1851 (routine service of fire protection systems), state-specific fire safety legislation

Common pitfalls:
- Not establishing an ESM register at practical completion that accurately reflects the installed systems — discrepancies between the occupancy permit schedule and actual installations create ongoing compliance gaps
- Allowing maintenance to lapse or using unqualified contractors, voiding insurance coverage
- Not updating the ESM schedule when building modifications change fire safety systems (e.g., tenant fitout adds or removes sprinkler heads)
- Treating ESM compliance as a facilities management task rather than an owner obligation — ultimately the building owner is legally responsible

## Workplace Health and Safety in Commercial Construction

Workplace health and safety (WHS) in commercial construction is governed by the harmonised Work Health and Safety Act and Regulations adopted in most Australian states and territories (except Victoria, which retains the Occupational Health and Safety Act 2004). The principal contractor (or principal contractor equivalent in Victoria) has primary duty of care for the construction workplace.

The principal contractor must prepare a WHS management plan before construction commences. For projects exceeding $250,000 in value (the threshold for notifiable construction work), the plan must be notified to the relevant state WHS regulator. The plan must address: site-specific hazards and risk controls, high-risk construction work activities, traffic management, emergency procedures, and site induction requirements.

Safe Work Method Statements (SWMS) are required for all high-risk construction work as defined in the WHS Regulations. High-risk activities common in commercial construction include: work at heights above 2 metres, demolition, excavation deeper than 1.5 metres, crane operations, working near live electrical services, and working in confined spaces. SWMS must identify hazards, assess risks, and specify control measures before work commences.

Working at height is the leading cause of fatalities in Australian construction. Scaffolding must comply with AS/NZS 1576 series. Edge protection, safety mesh, and catch platforms are required where there is a risk of falling more than 2 metres. Scaffolding must be designed, erected, and dismantled by competent persons with relevant licensing.

Reference: Work Health and Safety Act 2011 (harmonised); WHS Regulations 2011
Applies to: All commercial construction projects
Related standards: AS/NZS 1576 (scaffolding), AS 2601 (demolition), AS 1418/2550 (cranes), AS/NZS 4576 (scaffolding design)

Common pitfalls:
- Not appointing a principal contractor before construction commences — this is a legal requirement with significant penalties for non-compliance
- Treating SWMS as a documentation exercise rather than a genuine hazard identification and control process
- Not conducting regular site safety audits and toolbox talks to maintain safety culture throughout the project
- Failing to manage asbestos risks in refurbishment projects — asbestos audits must be completed before any demolition or refurbishment work commences

## Environmental Sustainability and Green Star

Sustainability in commercial construction encompasses energy efficiency, water conservation, materials selection, indoor environment quality, and ecological impact. The NCC Section J prescribes minimum energy efficiency requirements, while voluntary rating tools such as Green Star, NABERS, and WELL provide frameworks for achieving performance beyond minimum compliance.

NCC Section J energy efficiency requirements for commercial buildings include maximum glazing areas relative to wall area, minimum insulation R-values for walls, roofs, and floors, maximum solar admittance through glazing, and maximum building sealing limits. These requirements are intended to reduce operational energy consumption and are a baseline — many commercial developments target performance significantly above NCC minimums.

Green Star (administered by the Green Building Council of Australia) provides a voluntary design and construction rating framework. Ratings range from 1 Star to 6 Stars (World Leadership). Most institutional-grade commercial developments target a minimum 5 Star Green Star rating. The rating covers categories including energy, water, materials, indoor environment quality, transport, land use and ecology, emissions, management, and innovation.

NABERS (National Australian Built Environment Rating Scheme) rates the operational performance of existing buildings on a 1–6 star scale for energy, water, waste, and indoor environment. Many lease agreements now require buildings to achieve and maintain minimum NABERS ratings (typically 4.5–5.5 stars for energy). NABERS Commitment Agreements allow new buildings to commit to a target rating during design and verify performance post-occupancy.

Reference: NCC 2022 Volume 1, Section J — Energy Efficiency
Applies to: All new and substantially refurbished Class 5–9 buildings
Related standards: AS/NZS 4859 (thermal insulation), Green Star Design & As Built tool, NABERS rating methodology

Common pitfalls:
- Targeting Green Star design credits without understanding NABERS operational performance — a building can achieve 5 Star Green Star at design yet fail to achieve the target NABERS rating in operation
- Not modelling energy performance early enough in design to inform facade and HVAC system decisions
- Specifying sustainable materials without verifying supply chain availability — leading to substitutions that compromise the rating
- Failing to commission building systems properly, resulting in operational energy performance 30–50% worse than design intent

## Construction Programming for Commercial Projects

Commercial construction programs are typically prepared using critical path method (CPM) scheduling software. The program must identify the critical path, key milestones, procurement lead times, and construction sequences for all trades and disciplines.

Typical commercial construction durations in Australia range from 12–18 months for a 5–10 storey office building, 18–30 months for a major retail development, and 24–36 months for a large mixed-use project. These durations assume normal procurement and exclude approvals and pre-construction periods. Fit-out programs for office tenancies typically range from 8–16 weeks depending on scope.

Key programming considerations for commercial projects include: structural cycle times (typically 7–10 days per floor for reinforced concrete, 5–7 days for post-tensioned slabs); facade installation lead times (unitised curtain wall can require 6–9 months from order to first delivery); lift installation (typically 4–8 weeks per lift after shaft completion); and commissioning periods (typically 4–8 weeks for building services).

Programme acceleration techniques include: weekend and shift work (with associated cost premiums of 30–50%), parallel sequencing of trades on different floors, prefabrication and off-site manufacture, and early procurement of long-lead items. Each acceleration measure carries cost implications that must be evaluated against delay costs.

Reference: AS 4817 (construction programming best practice)
Applies to: All Class 5–9 commercial construction projects
Related standards: AS 4000 Clause 33 (time and programming under contract), AS 2124 Clause 33 (programming requirements)

Common pitfalls:
- Not identifying the critical path accurately — in commercial construction, facade delivery and services commissioning are frequently on the critical path, not structure
- Underestimating commissioning duration, particularly for complex building management systems with integrated controls
- Not allowing adequate float for weather delays in programs for buildings with significant exposed structural work
- Failing to coordinate procurement lead times for lifts, switchboards, and facade systems, which can each require 6–12 months from order

## Quality Assurance and Commissioning

Quality assurance in commercial construction requires a systematic approach to inspection, testing, and documentation throughout the construction process. The quality system should be established at project inception and implemented through inspection and test plans (ITPs) for each trade package.

Hold points and witness points are the primary quality control mechanisms. Hold points require the superintendent or relevant consultant to inspect and release the work before it is covered or the next stage proceeds. Critical hold points in commercial construction include: foundation inspections before concrete pour, structural frame connections before fireproofing, waterproofing before tiling or cladding, fire-rated construction before ceiling close-up, and services installations before concealment.

Commissioning is the systematic process of verifying that all building systems perform in accordance with the design intent and contract requirements. For commercial buildings, commissioning typically covers: HVAC and controls, electrical switchgear and distribution, fire protection systems, lifts, building management systems (BMS), security and access control, hydraulic systems, and facade performance.

Independent commissioning agents (ICA) are increasingly engaged on commercial projects to provide third-party verification of system performance. The ICA develops commissioning specifications, witnesses testing, and certifies system readiness for handover. For Green Star and NABERS rated buildings, independent commissioning is effectively mandatory.

Reference: AIRAH Best Practice Guidelines for Commissioning; CIBSE Commissioning Code M
Applies to: All Class 5–9 commercial construction projects
Related standards: NCC Section J (requires commissioning for energy efficiency compliance), Green Star Management credits (commissioning requirements)

Common pitfalls:
- Not engaging the commissioning agent early enough — the ICA should be appointed during design to review specifications and commissioning requirements
- Rushing commissioning to meet handover deadlines, resulting in defects and operational issues that persist through the defects liability period
- Not conducting integrated systems testing — testing each system in isolation does not verify that systems interact correctly (e.g., fire mode operation of HVAC, lifts, and BMS)
- Failing to provide adequate training and documentation for building operators at handover, resulting in systems being operated incorrectly

## Handover and Practical Completion

Practical completion in commercial construction is the contractual milestone when the works are complete except for minor omissions and defects that do not prevent the building from being used for its intended purpose. Under AS 4000, practical completion requires the works to be complete in accordance with the contract and reasonably fit for occupation. Under AS 2124, the superintendent issues a certificate of practical completion.

The handover process for commercial buildings requires coordination of multiple milestones: occupation certificate (or occupancy permit) issued by the certifying authority, essential safety measures schedule established, utility connections completed and tested, BMS and all services commissioning complete, as-built documentation submitted and verified, operations and maintenance manuals provided, training of facilities management staff completed, and defect lists issued and agreed.

The defects liability period (DLP) commences from the date of practical completion and typically runs for 12 months under standard contract conditions, though some projects specify 24 months. During the DLP, the contractor is obligated to rectify all defects notified by the principal. Security (usually a bank guarantee) is partially released at practical completion and the balance at the end of the DLP.

A comprehensive as-built documentation package is critical for ongoing building management. This should include: as-built architectural, structural, and services drawings; equipment manuals and warranties; test and commissioning certificates; fire engineering reports; BCA assessment reports; access certificates; and product compliance documentation for all specified materials and systems.

Reference: AS 4000-1997 Clause 42 (Practical Completion); AS 2124-1992 Clause 42 (Practical Completion)
Applies to: All Class 5–9 commercial construction projects
Related standards: State building legislation for occupation certificates, AS 1851 (ESM maintenance schedule establishment)

Common pitfalls:
- Issuing practical completion before commissioning is genuinely complete — resulting in the principal inheriting incomplete or non-functional systems
- Not establishing the ESM schedule at practical completion, creating a compliance gap from day one
- Providing inadequate as-built documentation — incomplete documentation becomes a persistent issue for facilities management and future fitout projects
- Not planning for staged practical completion in large mixed-use developments where different parts of the building are completed at different times

## Cost Planning for Commercial Projects

Cost planning for commercial buildings follows the Australian Institute of Quantity Surveyors (AIQS) guidelines, with estimates progressing from feasibility through to post-contract cost management. Elemental cost plans provide the framework for tracking costs against functional elements of the building.

Commercial benchmarking is commonly normalised by gross floor area (GFA), net
lettable area (NLA), functional area or another stated denominator. This guide
does not supply market rates: source any benchmark from current, attributable
project evidence or a qualified cost adviser, record its date and inclusions,
and leave the amount TBC when that evidence is unavailable.

Services costs in commercial buildings typically represent 35–45% of total construction cost, compared to 15–25% for residential projects. This higher proportion reflects the complexity of HVAC, electrical, fire protection, lift, and BMS systems. Services cost must be tracked separately and managed as a distinct cost centre.

Design and construction contingency should be applied in accordance with the project risk profile. Typical contingency ranges for commercial projects are: feasibility stage 15–20%, concept design 10–15%, design development 8–10%, tender 3–5%. These ranges assume a traditional lump-sum procurement method — design and construct projects may warrant lower contingency on the principal's side as design risk transfers to the contractor.

Reference: AIQS Cost Management Practice Guide; Rawlinsons Australian Construction Handbook
Applies to: All Class 5–9 commercial construction projects
Related standards: AIQS Standard Method of Measurement, NRM1 (New Rules of Measurement for cost planning)

Common pitfalls:
- Benchmarking on GFA without adjusting for building efficiency (NLA/GFA ratio) — a building with 80% efficiency looks cheaper per m² GFA but may cost more per m² NLA
- Not escalating cost benchmarks for construction cost inflation, which has run at 5–8% per annum in major Australian markets since 2021
- Underestimating services costs by using residential cost data as a reference for commercial projects
- Not accounting for authority contributions, development contributions, and infrastructure charges that can add 5–10% to project cost

## Commercial Lease and Make-Good Obligations

Make-good obligations are a critical consideration in commercial fitout projects. At lease expiry, tenants are typically required to return the premises to the condition described in the lease — usually the base building condition existing at lease commencement. The extent of make-good work depends on the lease terms and the condition survey (or schedule of condition) prepared at lease commencement.

Common make-good requirements include: removal of all tenant partitions, joinery, and finishes; removal of supplementary services installations (additional air-conditioning, power, and data cabling); reinstatement of ceiling grids and tiles to base building standard; patching and making good of penetrations through fire-rated construction; reinstatement of sprinkler heads and fire detection devices to base building configuration; removal of tenant signage and reinstatement of affected surfaces.

The cost of make-good works can be substantial and should be budgeted by tenants throughout the lease term through a make-good provision. Typical make-good costs for a standard office fitout range from $200–$500/m² depending on fitout complexity and the extent of services modifications. Specialist fitouts (commercial kitchens, laboratories, medical suites) can have make-good costs of $800–$1,500/m² or more.

Landlords and tenants should negotiate make-good scope and timing well before lease expiry. In a favourable market, landlords may accept a cash settlement in lieu of physical make-good if the incoming tenant requires a different configuration. Early negotiation avoids the cost and disruption of physical reinstatement followed by immediate demolition by the next tenant.

Reference: Property Council of Australia — Guide to Office Building Quality; relevant state retail/commercial leases legislation
Applies to: Class 5 (office), Class 6 (retail) tenancies
Related standards: AS/NZS 3000 (electrical disconnection requirements), state fire safety regulations for make-good of fire-rated construction

Common pitfalls:
- Not preparing a detailed schedule of condition at lease commencement — without baseline documentation, make-good scope becomes contested
- Tenants not budgeting for make-good costs throughout the lease term, leading to financial pressure at lease expiry
- Assuming that the landlord will accept a cash settlement — the lease typically gives the landlord the right to require physical reinstatement
- Not engaging fire safety consultants for make-good of fire-rated construction — incorrect reinstatement of fire walls and ceilings creates non-compliance

## Heritage and Adaptive Reuse in Commercial Construction

Adaptive reuse of heritage buildings for commercial purposes is increasingly common in Australian cities as organisations seek characterful office and retail space. Heritage-listed buildings (at local, state, or national level) are subject to additional planning controls that significantly influence design, approval timelines, and construction methodology.

Heritage listings impose constraints on external alterations to significant fabric, requiring conservation architects and heritage consultants to guide the design process. Development applications for heritage-listed buildings typically require a Heritage Impact Statement (HIS) and may require referral to state heritage authorities (Heritage NSW, Heritage Victoria, etc.), adding 3–6 months to the approvals timeline.

Structural interventions in heritage buildings must be designed to minimise impact on significant fabric. Common approaches include: independent structural frames inserted within the existing building envelope; underpinning of existing foundations to support additional loads; carbon fibre and steel reinforcement bonded to existing structural elements; and removal and reinstatement of heritage fabric where new services or structure must penetrate historic walls.

Services installation in heritage buildings requires careful routing to avoid damaging significant fabric. Surface-mounted and concealed raceways, false floors, and dropped ceilings are common strategies, but must be assessed against heritage significance — a dropped ceiling in a building with significant decorative cornices or pressed metal ceilings may not be acceptable to the heritage authority.

Reference: Heritage Act 1977 (NSW), Heritage Act 2017 (VIC), and equivalent state heritage legislation; Burra Charter (Australia ICOMOS)
Applies to: Heritage-listed Class 5–9 buildings undergoing adaptive reuse or refurbishment
Related standards: NCC provisions for existing buildings (compliance alternatives under Part A0.7)

Common pitfalls:
- Underestimating approval timelines for heritage projects — referral to state heritage authorities adds months to the DA process
- Not engaging conservation architects early in feasibility, leading to designs that heritage authorities reject
- Assuming BCA compliance can be achieved through standard deemed-to-satisfy solutions — heritage buildings often require fire-engineered alternative solutions
- Underestimating the cost premium for heritage work, which typically adds 20–40% to construction costs compared to equivalent new-build scope

## Security Systems and Access Control

Security systems in commercial buildings encompass electronic access control, CCTV surveillance, intruder detection, intercom systems, and physical security measures. System design should be informed by a security risk assessment conducted by a qualified security consultant.

Electronic access control systems manage building entry at main entrances, carpark entries, lift lobbies, stair doors, and tenancy entries. For multi-tenant commercial buildings, the base building typically provides access control at building entry points and common areas, with tenants managing access within their tenancies. Integration between base building and tenant systems requires careful planning of communication protocols and credential management.

CCTV systems to AS 4806 provide surveillance of building entries, carparks, loading docks, common areas, and building perimeters. Camera selection (fixed, PTZ, analytics-capable) and storage requirements must be specified based on the security risk assessment. Privacy legislation (e.g., Surveillance Devices Act) requires consideration of camera placement, signage, and data retention policies.

Intruder detection systems to AS/NZS 2201 are typically provided for ground-floor tenancies, after-hours protection of common areas, and back-of-house areas. Alarm monitoring must be conducted by a licensed monitoring centre complying with AS 2201.2.

Reference: AS/NZS 2201 (intruder alarm systems), AS 4806 (CCTV management and operation)
Applies to: All Class 5–9 commercial buildings
Related standards: AS/CA S009 (communications cabling), state surveillance devices legislation, Privacy Act 1988

Common pitfalls:
- Not conducting a security risk assessment before specifying systems, resulting in either over-specified or inadequate security
- Failing to integrate base building and tenant access control systems, creating security gaps at transition points
- Not providing adequate network infrastructure for IP-based CCTV systems — bandwidth and storage requirements are frequently underestimated
- Overlooking privacy legislation requirements for CCTV placement and data retention, exposing the building owner to legal liability

## Demolition and Strip-Out for Commercial Refurbishment

Demolition and strip-out works in commercial refurbishment projects require careful planning, regulatory compliance, and hazardous materials management. Work must comply with AS 2601 (Demolition of Structures) and the relevant state WHS regulations for demolition work.

An asbestos audit must be completed by a licensed asbestos assessor before any demolition or refurbishment work commences. The audit identifies the location, type, condition, and extent of all asbestos-containing materials (ACMs) within the building. An asbestos removal control plan must be prepared and the removal carried out by licensed asbestos removalists — Class A for friable asbestos and Class B for bonded asbestos.

Structural demolition sequences must be engineered to maintain building stability throughout the demolition process. A demolition work plan prepared by a competent person must address: sequence of demolition, temporary support and propping requirements, exclusion zones, adjacent building protection, dust and noise management, and waste classification and disposal.

Hazardous materials beyond asbestos that may be encountered in commercial demolition include: lead-based paint, polychlorinated biphenyls (PCBs) in electrical equipment, synthetic mineral fibres, and contaminated soil beneath floor slabs. Waste must be classified under the state waste classification guidelines and disposed of at appropriately licensed facilities.

Reference: AS 2601 (Demolition of Structures); WHS Regulations — Chapter 8 (Asbestos)
Applies to: All Class 5–9 commercial refurbishment and demolition projects
Related standards: State-specific asbestos codes of practice, EPA waste classification guidelines

Common pitfalls:
- Commencing demolition before completing the asbestos audit — this is a regulatory offence with significant penalties
- Not allowing adequate program time for asbestos removal, which must be completed before general demolition can proceed in affected areas
- Underestimating the cost of hazardous material removal and disposal — this can represent 10–20% of strip-out cost in older commercial buildings
- Not considering the impact of demolition vibration and noise on adjacent occupied tenancies in a staged refurbishment

## Carpark Design for Commercial Buildings

Carparking is a significant component of most commercial developments, often representing 20–30% of construction cost. Design must comply with AS/NZS 2890.1 (off-street car parking) and AS/NZS 2890.6 (off-street parking for people with disabilities).

Minimum parking stall dimensions under AS/NZS 2890.1 are 2.4m wide by 5.4m long for user class 3 (short to medium term, typical of commercial buildings). Aisle widths depend on parking angle — 90-degree parking requires minimum 5.8m aisle for one-way traffic. Column placement must be coordinated with the structural grid to avoid encroaching on stall dimensions or impeding vehicle access. A structural grid of 8.4m x 8.4m or 9.0m x 9.0m typically provides efficient carpark layouts with 3 stalls between columns.

Accessible carparking spaces must be provided per AS/NZS 2890.6, with minimum 1 space for every 50 spaces (or part thereof) for Class 5 and 6 buildings. Accessible spaces require minimum 2.4m width with 2.4m shared area for vehicle access (or 3.2m wide without shared area), and must have access to an accessible path of travel to building entrances.

Mechanical ventilation is required in enclosed carparks per AS 1668.1 and AS 1668.2. CO monitoring systems are used to control ventilation demand. Fire sprinklers per AS 2118.1 are required in most enclosed carparks. Carpark fire hydrant systems must comply with AS 2419.1. Emergency lighting and exit signage per AS 2293 must be provided throughout.

Electric vehicle (EV) charging infrastructure is increasingly required by local planning policies. Even where not mandated, providing conduit and cable routes for future EV charging is recommended as a cost-effective measure during initial construction.

Reference: AS/NZS 2890.1 (off-street car parking), AS/NZS 2890.6 (accessible parking)
Applies to: All Class 5–9 buildings with carparking
Related standards: AS 1668 (carpark ventilation), AS 2118.1 (sprinklers), AS 2419.1 (hydrants), local planning controls for parking rates

Common pitfalls:
- Not coordinating structural grid with carpark geometry early in design — an inefficient grid can reduce carpark capacity by 10–15%
- Underestimating the floor-to-floor height required for carparks with sprinklers, ventilation ductwork, and drainage falls
- Not providing adequate queuing capacity at entry and exit barriers, causing traffic to back up onto public roads during peak periods
- Failing to plan EV charging infrastructure during construction — retrofitting conduits and switchboard capacity is significantly more expensive than including provision in initial design

## Waterproofing in Commercial Buildings

Waterproofing failures are among the most common and costly defects in commercial construction. Effective waterproofing requires correct system specification, careful detailing, and rigorous quality assurance during application. Waterproofing systems must comply with AS 4654.1 and AS 4654.2 for external above-ground applications, and AS 3740 for internal wet areas (though AS 3740 is primarily a residential standard, its principles apply to commercial wet areas).

Critical waterproofing areas in commercial buildings include: podium and rooftop decks over occupied spaces, planter boxes on structural slabs, basement walls and slabs below groundwater level, wet areas (amenities, commercial kitchens, cleaners' rooms), facade penetrations and window perimeters, and expansion joints in external-facing elements.

Below-ground waterproofing for basements typically uses either tanking systems (sheet membrane or cementitious coating applied externally) or drained cavity systems (internal drainage with cavity wall). System selection depends on groundwater conditions, structural movement expectations, and the intended use of the basement space. Critical failures in below-ground waterproofing are extremely costly to rectify as they require excavation or internal remediation.

Quality assurance for waterproofing must include: surface preparation inspection before membrane application, flood testing of horizontal membranes before tiling (minimum 24-hour flood test for wet areas), adhesion testing of applied membranes, and photographic documentation of all waterproofing before concealment. Independent waterproofing inspections are recommended for all critical areas.

Reference: AS 4654.1 (waterproofing materials), AS 4654.2 (waterproofing design and installation), AS 3740 (waterproofing of domestic wet areas)
Applies to: All Class 5–9 buildings
Related standards: AS 4200 (pliable building membranes), state-specific waterproofing licensing requirements

Common pitfalls:
- Not requiring flood testing of wet area waterproofing before tiling — defects discovered after tiling require complete removal and reinstatement
- Using inappropriate waterproofing systems for areas subject to hydrostatic pressure (e.g., tanking systems designed for above-ground use installed below ground)
- Failing to detail waterproofing terminations and upturns at the design stage, leading to ad-hoc detailing on site
- Not coordinating waterproofing with structural movement joints — movement joints must be carried through the waterproofing membrane with appropriate movement-capable details

## Building Management Systems (BMS)

Building management systems provide centralised monitoring and control of mechanical, electrical, fire, and other building services. A well-designed BMS is critical for energy efficiency, occupant comfort, and operational cost management in commercial buildings.

The BMS typically controls and monitors: HVAC systems (chillers, boilers, air handling units, VAV boxes, FCUs); lighting (common area control, time scheduling, daylight harvesting); lifts (monitoring and fire mode interface); fire systems (interface for alarm, sprinkler flow, and smoke control); electrical (main switchboard monitoring, power quality, generator interface); hydraulic (pump monitoring, tank levels); and security and access control (integration or interface).

Communication protocols for BMS in Australian commercial buildings commonly include BACnet (the open protocol standard for building automation), Modbus for legacy equipment and utilities metering, and proprietary protocols from major BMS manufacturers (Siemens, Honeywell, Schneider, Johnson Controls). Specifying open protocols (particularly BACnet) is recommended to avoid vendor lock-in and enable future system expansion.

BMS commissioning must verify all control sequences, setpoints, alarms, and system interlocks. Trend logging should be established during commissioning and maintained during the first 12 months of operation to verify system performance against design intent. This data is essential for NABERS rating verification and ongoing operational optimisation.

Reference: ASHRAE Guideline 13 (Specifying Direct Digital Control Systems); CIBSE Guide H (Building Control Systems)
Applies to: All Class 5–9 commercial buildings with centralised building services
Related standards: NCC Section J (energy efficiency — requires BMS for certain building types), NABERS Energy (requires BMS for operational rating), Green Star credits for metering and monitoring

Common pitfalls:
- Over-specifying BMS scope without considering the building operator's capacity to manage the system — a complex BMS with no trained operator delivers no benefit
- Not specifying open communication protocols, leading to vendor lock-in where only one contractor can maintain or modify the system
- Failing to commission BMS control sequences under all operating conditions (including fire mode, after-hours, and extreme weather) — partial commissioning leaves untested failure modes
- Not providing adequate BMS points for tenant sub-metering, which is required for NABERS rating and increasingly expected in tenant leases
