# Proposed trade package master list

11 October 2026. Status: draft backlog for owner review. See the [audit and pilot plan](../plans/2026-10-11-consultant-audit-trade-rft-plan.md).

This is a starting library for Australian building work, not a claim that every project needs every package. Select packages from the physical systems and project delivery strategy. Split or combine procurement only with explicit responsibility boundaries. The list does not change the application taxonomy.

## Core trade candidates

Each semicolon-separated entry is a potential standalone framework. It can instead be a separately allocated component of a broader package where that is how the project buys it.

| Building area | Candidate frameworks |
|---|---|
| Site and ground | Demolition and strip-out; asbestos/hazardous-material removal; earthworks and excavation; civil drainage and external civil works; piling; retention and shoring; dewatering; ground improvement |
| Concrete structure | **Formwork and falsework**; **reinforcement fixing**; concrete placement and finishing; post-tensioning; precast concrete; tilt-up concrete; shotcrete |
| Other structure | Structural steel; structural timber and carpentry; light-gauge steel framing; masonry and blockwork |
| Envelope | Roofing and roof plumbing; windows and external glazing; curtain wall and façade systems; external cladding; waterproofing; insulation; sealants and movement joints |
| Interiors | Partitions and linings; suspended ceilings; doors, frames and hardware; joinery and kitchens; tiling and stone finishes; resilient flooring and carpet; timber flooring; painting and protective coatings; metalwork, balustrades and handrails; operable walls and specialist acoustic treatments |
| Services | Hydraulic and plumbing; mechanical ventilation and air conditioning; electrical; wet-fire systems; fire detection and alarm systems; passive-fire stopping; communications and data; security and access control; audiovisual; BMS and controls; lifts and other vertical transport; solar PV and batteries; specialist/process services |
| External works | Landscaping and planting; irrigation; external paving and kerbs; fencing and gates; line marking; signage and wayfinding; external furniture and play equipment |
| Existing building specialists | Concrete repair and strengthening; specialist remedial works defined from diagnosis, with overlaps to façade/waterproofing/structural packages expressly removed |

Concrete supply and pumping require explicit allocation, whether purchased separately or included with placement. Reinforcing supply requires a separate procurement variant, not automatic inclusion in steel fixing. Façade systems may include windows/cladding/sealants; identify the boundary rather than buying each twice. Services bundles must assign controls, dry fire, fire stopping and integrated testing explicitly.

## Common site services

Maintain these as optional procurement candidates with their own terms and boundaries when separately purchased: scaffolding and access; cranes and hoists; temporary site facilities and temporary services; traffic management; construction waste removal; final cleaning; construction survey/set-out; testing and inspection services; integrated commissioning coordination.

Some are hired plant or professional services rather than installing trades. Do not force all of them into `trade.*` or duplicate consultant appointments. Main Works remains a head-contract package and is not one of these individual trades.

## Existing taxonomy reconciliation

| Existing ID | Proposed treatment before any later encoding |
|---|---|
| `trade.main_works` | Retain as head contract |
| `trade.civil_works` | Preserve ID; define earthworks, drainage and external works allocations |
| `trade.electrical` | Preserve; allocate dry fire, controls, data, security, PV and specialist services explicitly |
| `trade.mechanical` | Preserve; allocate controls, electrical supplies and commissioning |
| `trade.hydraulic_plumbing` | Preserve; allocate roof plumbing, civil connections, irrigation and wet fire |
| `trade.structural_steel` | Preserve; never use as the reinforcement-fixing label |
| `trade.fire_services` | Preserve; settle wet/dry system allocation before deciding if extra IDs are useful |
| `trade.joinery_kitchens` | Preserve; define fit-off and services interfaces |
| `supplier.windows_glazing` | Retain supply-only meaning; installing window/façade scope needs an explicit trade mapping |
| Formwork and reinforcement fixing | Propose new stable IDs only at implementation; no existing exact trade labels found |

## Research order

1. Formwork pilot, then steel fixing and their joint boundary check.
2. Concrete placement, post-tensioning and structural steel to test adjacent structure packages.
3. Waterproofing, roofing, windows/façade and cladding to test enclosure interfaces.
4. Services packages, including wet/dry/passive fire and controls allocation.
5. Interiors, site/external works and project-specific specialists as actual projects require.

This order is provisional. An upcoming project can move a trade forward without starting the whole list. The immediate authorised output is the plan and backlog; no individual trade ledger/RFT is claimed complete by this list.
