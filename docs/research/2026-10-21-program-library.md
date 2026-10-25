# Program library research

### Overview

This note is the basis for every row in `knowledge/program/`. Code builds a program from the project's facts and these rows. The owner accepted the duration rows on 24 October 2026, so generation uses them. Fragments and skeletons remain draft and are still included. The figures are planning estimates for NSW metropolitan work, from ordinary Australian practice, except the four statutory clocks, which stay `verified: false`.

Nine skeletons sit after the unchanged `sk.generic`. A project takes the matching skeleton with the most conditions. An unknown fact never matches. The generic skeleton remains the fallback, and its stage order still puts construction documentation before approvals. The new skeletons put the consent or certifier gate in the place the procurement route actually uses.

| Skeleton | Selects when |
| --- | --- |
| `sk.new-traditional-da` | new, traditional, DA or planning permit |
| `sk.new-traditional-cdc` | new, traditional, CDC or exempt |
| `sk.new-dc-da` | new, design and construct, DA or planning permit |
| `sk.new-eci-da` | new, early contractor involvement or managing contractor, DA or planning permit |
| `sk.refurb-traditional-da` | refurbishment, traditional, DA or planning permit |
| `sk.refurb-traditional-cdc` | refurbishment, traditional, CDC or exempt |
| `sk.extend-traditional` | extension, traditional, DA, CDC, exempt or planning permit |
| `sk.ssd-traditional` | new or extension, traditional, State significant development |
| `sk.ssd-dc` | new or extension, design and construct or early contractor or managing contractor, State significant development |

A Victorian planning permit is treated like a development application: consent before construction documentation and tender. Complying development and exempt work skip that consent stage. Design and construct procures the builder after consent and lets construction overlap the builder's documentation. Early contractor involvement and a managing contractor bring the builder on during design. State significant development keeps the long assessment in front of tender. Alliance, PPP, remediation and advisory work still fall through to `sk.generic`.

Fragments follow work items. Class variants are separate rows so a house, an apartment building, a warehouse, a school and a commercial fit-out do not share one sequence. New work and extensions use `work_type` `new` or `extend`. Refurbishment and fit-out use `work_type` `refurb`. Demolition and strip-out use `work_action` `remove` only. Within the construction stage the file is grouped by class, because anything left unlinked is chained in that order. Long-lead orders that must start with the stage (lifts, major switchboards, authority connections) are anchored at `stage_start` and listed last, so they do not push the early trades out.

The three programs this set is written for are a new Class 1a house, a new Class 2 apartment building of about 3 to 12 storeys with a basement, and a Class 5 or 6 fit-out in an existing building. House extensions use the Class 1a new-work rows plus underpinning. Warehouses (Class 7b) and low-rise schools (Class 9b) have their own new-work rows. A school or commercial refurbishment uses the fit-out rows.

Known gaps, left out on purpose: classes 1b, 3, 4, 7a, 8, 9a, 9c and 10; a new Class 5 or 6 base building (the commercial rows are fit-out); alliance and PPP stage patterns; a second procurement stage for managing-contractor trade packages; facade starting before the structure finishes (the link is finish-to-start, which is conservative); services fit-off separated from rough-in by a lag allowance rather than a link to the lining fragment; calendars. NSW public holidays for 2026 and 2027 are published, but the construction RDO roster and the Christmas shutdown are set each year and were not copied from a named roster. A wrong shutdown is worse than weekends only, so `calendars.yaml` is unchanged and empty.


## Skeletons


### sk.new-traditional-da

New work procured traditionally, where planning is a development application or a Victorian planning permit.
Consent sits before construction documentation and tender, which is the ordinary sequence when a builder prices a documented design.
Neighbouring stages stay finish-to-start. Construction is not overlapped onto the approval, so the program does not show site work before consent.
Alliance, PPP and a design-and-construct path do not select this skeleton.


### sk.new-traditional-cdc

New work procured traditionally on a complying-development or exempt path.
There is no development-application stage. The certifier's approval replaces consent, and the construction set is not a separate stage because on this path it is usually the package the certifier sees.
Tender still follows that approval. A development application does not select this skeleton.


### sk.new-dc-da

New design-and-construct work with a development consent or planning permit.
The client team takes the design through consent, the builder is then procured, and the builder completes construction documentation.
Construction is start-to-start with that documentation, lagged by du.dc-site-overlap, so early site work can start while the builder's documents are finished.
The overlap replaces the automatic finish-to-start link between those two neighbouring stages.


### sk.new-eci-da

New work using early contractor involvement or a managing contractor, with a development consent or planning permit.
Procurement sits before approval documentation so the contractor is on board during design.
Approval documentation is start-to-start with that appointment, lagged by du.eci-overlap, instead of waiting until procurement finishes.
A managing contractor's later trade-by-trade buying is not a second procurement stage. The library has only one procurement stage.


### sk.refurb-traditional-da

Refurbishment procured traditionally where a development consent or planning permit is required.
That includes change of use and work that needs consent. Investigation stays first, and consent still gates construction documentation and tender.
The stage list matches the new traditional path. The activities differ because fragments follow the refurbishment work items, including strip-out.
An exempt fit-out does not select this skeleton.


### sk.refurb-traditional-cdc

Refurbishment or fit-out procured traditionally where the path is complying development or exempt.
This is the commercial fit-out case. Investigation and a construction-documentation set come first, then a short certifier approval, then tender and construction.
There is no separate approval-documentation stage, because the fit-out drawings are the set that is priced and certified.
A refurbishment that needs a development application uses sk.refurb-traditional-da instead.


### sk.extend-traditional

An extension or addition procured traditionally, on a consent, complying, exempt or planning-permit path.
Investigation of the existing building stays first, then approval, then construction documentation and tender.
One skeleton covers those planning paths so a house extension does not fall through to sk.generic, which puts construction documentation before the approval.
The approval stage is short when the path is exempt and long when it is a development application. The fragments carry that difference.


### sk.ssd-traditional

New work or an extension that is State significant development and procured traditionally.
The stage order matches a consented traditional project: the long assessment finishes before construction documentation and tender.
It is a separate skeleton because this path must not use the generic order, and it does not match the development-application skeletons.
The long assessment activities are the project-level SSD fragments, not a different stage id.


### sk.ssd-dc

State significant new work or an extension where a contractor is appointed under design and construct, early contractor involvement or a managing contract.
Consent still finishes before procurement. The department's assessment is not overlapped with tender.
The contractor's documentation then overlaps the start of construction by du.dc-site-overlap, as on a design-and-construct development application.
Refurbishment that is State significant is not in this skeleton and falls through to sk.generic.


## Fragments


### fr.temporary-power

Temporary power is a construction fragment on `electrical.temporary-power`.
A builder's temporary supply, anchored at the start of construction so the rest of the stage does not wait out a long activity. Sized as a few days to connect a site board, not a permanent substation.
Activities, in order: connect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.erosion-control

Erosion and sediment control is a construction fragment on `site.erosion-sediment-control`.
Install controls before the ground is opened. One short activity, anchored at stage start. Ongoing maintenance is not given a long duration, because that would hold the following trade until the end of the job.
Activities, in order: install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tree-protection

Tree protection is a construction fragment on `site.trees`.
Protect trees that are to remain before demolition or excavation. Anchored at stage start. Appears only when the project has a work item on site.trees.
Activities, in order: protect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.occupied-protection

Occupied-building protection is a construction fragment on `site.impact-protection`.
Hoardings and protection where an existing building stays occupied. Used for a refurbishment that has this work item. Anchored at stage start so the strip-out chain is not forced to the end of a long activity.
Activities, in order: hoard.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.demolition-1a

Demolition (house) is a construction fragment on `site.demolition`.
Knockdown of an existing house or outbuilding before a new Class 1a building or extension. Remove only, so a retained structure does not select it. A renovation strip-out is fr.strip-out-1a.
Activities, in order: demolish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.strip-out-1a

Strip-out (house renovation) is a construction fragment on `site.demolition`.
Soft strip of a Class 1a renovation, not a full knockdown. Remove only, and only when the work type is refurbishment.
Activities, in order: strip.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.earthworks-1a

Earthworks (house) is a construction fragment on `site.earthworks`.
Site cut, fill and levels for a house. Follows demolition when that work item exists, and otherwise follows the short early site activities.
Activities, in order: cut.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.underpinning-1a

Underpinning (house) is a construction fragment on `substructure.underpinning`.
Underpin the existing house before new footings for an extension or renovation. Not used on a clear new build unless that work item is present.
Activities, in order: underpin.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.footings-1a

Footings (house) is a construction fragment on `substructure.footings`.
Excavate, inspect, pour and cure strip or pad footings for a Class 1a house. The cure is calendar time and is the last activity, so the slab waits for it when this fragment is present.
Activities, in order: excavate, pour, cure.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.termite-1a

Termite management (house) is a construction fragment on `substructure.termite-management`.
Physical or chemical termite barrier to a Class 1a house, tied to the footings so it is in place before the slab pour. The slab fragment does not repeat it.
Activities, in order: barrier.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.damp-1a

Damp-proofing (house) is a construction fragment on `substructure.damp-waterproofing`.
Damp-proof membrane and vapour barrier to a house slab, tied to the footings. It should be finished before the pour. It is a separate fragment so the work item on this system is what brings it in.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stormwater-1a

Stormwater (house) is a construction fragment on `site.stormwater-detention`.
House stormwater connection and any on-site detention the work item calls for. Tied to earthworks. A simple street connection is the low end.
Activities, in order: drain.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.flood-1a

Flood works (house) is a construction fragment on `site.flooding`.
House works that respond to a mapped flood hazard: set levels, and install the barriers or the raised works the work item describes. Only when flood hazard is true.
Activities, in order: protect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.slab-on-ground-1a

Slab on ground (house) is a construction fragment on `substructure.slab-on-ground`.
Ground slab for a Class 1a house after the footings. Termite management and the membrane are their own fragments, tied to the same footings, and are assumed to finish during set-out. Under-slab sanitary drainage is inside this fragment because it has to be in before the pour.
Activities, in order: set-out, under-slab, pour, cure.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.frame-timber-1a

Timber frame (house) is a construction fragment on `structure.timber`.
Wall frames, roof trusses and bracing for a one or two storey Class 1a house. Scaled by floor area. Follows the slab cure.
Activities, in order: walls, roof.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.masonry-1a

Brick veneer (house) is a construction fragment on `structure.masonry`.
Brick veneer to a timber-framed house, after the frame. A full masonry house is uncommon in this row; the high end covers a heavier skin.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-1a

Roof cover (house) is a construction fragment on `envelope.roof-coverings`.
Sarking and roof cover after the trusses. Gutters and downpipes are the roof-drainage fragment. A two-storey house does not get a second roof, so this row is fixed rather than scaled by floor area.
Activities, in order: sark, cover.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.external-walls-1a

External walls (house) is a construction fragment on `envelope.external-walls`.
Wrap and lightweight cladding, or the non-brick skin, after the frame. Brick veneer stays on the masonry fragment.
Activities, in order: clad.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.insulation-1a

Insulation (house) is a construction fragment on `envelope.thermal-insulation`.
Wall and ceiling insulation after the frame and before linings. Scaled by floor area as a proxy for the insulated envelope.
Activities, in order: insulate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.windows-1a

Windows and external doors (house) is a construction fragment on `envelope.windows-doors`.
Order and install house windows and external doors after the frame. The lead is the remaining delivery after the frame, assuming the order was placed from the construction documents. It is not the full factory time measured from development approval.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.bushfire-1a

Bushfire construction (house) is a construction fragment on `envelope.bushfire-construction`.
Extra ember and flame-zone construction to a Class 1a house on bushfire-prone land, after the external wall skin. Only when bushfire-prone land is true and this work item exists.
Activities, in order: screen.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-drainage-1a

Roof drainage (house) is a construction fragment on `hydraulic.roof-drainage`.
Gutters, downpipes and the connection to the stormwater system, after the roof cover.
Activities, in order: gutters.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.linings-1a

Walls and linings (house) is a construction fragment on `interiors.walls-linings`.
Internal partitions and plasterboard after the roof is on, which is the usual lock-up point before linings. Insulation is its own fragment.
Activities, in order: partitions, sheet.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ceilings-1a

Ceilings (house) is a construction fragment on `interiors.ceilings`.
Ceiling linings after the wall linings have started. A single activity because the trade is one pass.
Activities, in order: ceilings.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.wet-area-1a

Wet-area waterproofing (house) is a construction fragment on `interiors.wet-area-waterproofing`.
Waterproof bathrooms, the laundry and other wet areas after linings and before tiling. Fixed, because it follows the wet rooms rather than the whole floor area.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.internal-doors-1a

Internal doors (house) is a construction fragment on `interiors.internal-doors`.
Hang internal doors after linings. A short fixed task on a house.
Activities, in order: hang.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-sanitary-1a

Sanitary plumbing (house) is a construction fragment on `hydraulic.sanitary`.
Above-slab sanitary rough-in and fixture fit-off for a house. Under-slab drainage is in the slab fragment. The lag before fit-off stands in for linings and waterproofing.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-cold-1a

Cold water (house) is a construction fragment on `hydraulic.cold-water`.
Cold-water rough-in and fit-off for a house, after the slab. The authority meter is the services-connection fragment.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-hot-1a

Hot water (house) is a construction fragment on `hydraulic.hot-water`.
Hot-water unit and pipework for a house. A single unit is typical. A recirculating or heat-pump system is the high end.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.gas-1a

Gas (house) is a construction fragment on `hydraulic.gas`.
Consumer gas piping and the connection to appliances in a house, after the slab. The authority main is the services-connection fragment.
Activities, in order: pipe.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-1a

Electrical wiring (house) is a construction fragment on `electrical.cabling-wiring`.
Cable rough-in and fit-off for a house, after the frame. The switchboard and the street supply are their own fragments. The lag stands in for linings.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lighting-1a

Lighting (house) is a construction fragment on `electrical.lighting`.
Install luminaires after linings, when the ceilings can take them.
Activities, in order: fit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.switchboard-1a

Switchboard (house) is a construction fragment on `electrical.switchboards`.
A house switchboard, after the frame. Residential boards are short-lead items, so this is installation rather than a long procurement wait.
Activities, in order: board.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.smoke-alarms-1a

Smoke alarms (house) is a construction fragment on `fire-active.smoke-alarms`.
Interconnected smoke alarms to a Class 1a house, after linings. Apartment alarms are a separate fragment.
Activities, in order: alarms.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ac-1a

Air-conditioning (house) is a construction fragment on `mechanical.air-conditioning`.
A split-system or ducted unit to a house, after linings. It is not a commercial plant room.
Activities, in order: install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tiling-1a

Tiling (house) is a construction fragment on `interiors.tiling`.
Wall and floor tiling to house wet areas and the kitchen splash, after waterproofing. Fixed, because it does not follow the whole floor area.
Activities, in order: tile.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.joinery-1a

Joinery (house) is a construction fragment on `interiors.joinery`.
Kitchen, vanities and robes. Procurement starts after linings, which is conservative: a real order is often placed from the construction documents. The lead shown here is the shop time still to run at that point.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-1a

Painting (house) is a construction fragment on `interiors.painting`.
Internal paint after joinery is installed, which is the usual house sequence. External paint of lightweight cladding is included in the high end.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.floors-1a

Floor finishes (house) is a construction fragment on `interiors.floor-finishes`.
Timber, hybrid or carpet finishes after painting, so the floors are not trapped under wet trades.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.landscape-1a

Landscape (house) is a construction fragment on `site.landscape`.
External paving, turf and planting after the wall skin, so it can run beside internal finishes. Driveways on difficult access use the high end.
Activities, in order: landscape.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.linings-1a-reno

Linings (house renovation) is a construction fragment on `interiors.walls-linings`.
Patch and reline after a house strip-out. Fixed, because a renovation is not the whole floor plate. A new house uses fr.linings-1a.
Activities, in order: reline.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.wet-area-1a-reno

Wet areas (house renovation) is a construction fragment on `interiors.wet-area-waterproofing`.
Waterproof replaced bathrooms in a house renovation, after the new linings.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-1a-reno

Sanitary alterations (house renovation) is a construction fragment on `hydraulic.sanitary`.
Alter sanitary pipework in a house renovation. Capped services and new fixture positions. The lag allows the rooms to be relined.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.water-1a-reno

Water alterations (house renovation) is a construction fragment on `hydraulic.cold-water`.
Alter cold water in a house renovation. Hot water stays on its own system if that work item exists; this row does not invent a new unit.
Activities, in order: alter.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-1a-reno

Electrical alterations (house renovation) is a construction fragment on `electrical.cabling-wiring`.
Alter wiring in a house renovation after the strip-out. A switchboard upgrade is the switchboard fragment if that work item exists.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.joinery-1a-reno

Joinery (house renovation) is a construction fragment on `interiors.joinery`.
A replacement kitchen or vanity after the renovation linings. Manufacture is included in the one duration because the fragment is a single activity and takes the work item's title.
Activities, in order: install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-1a-reno

Painting (house renovation) is a construction fragment on `interiors.painting`.
Repaint the renovated rooms after joinery. Not the whole-house new-work rate.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.floors-1a-reno

Floors (house renovation) is a construction fragment on `interiors.floor-finishes`.
New floor finishes to the renovated rooms after paint.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.demolition-2

Demolition (apartments) is a construction fragment on `site.demolition`.
Demolish existing structures before a new Class 2 building. Remove only. A large existing building or asbestos is the high end.
Activities, in order: demolish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.earthworks-2

Earthworks (apartments) is a construction fragment on `site.earthworks`.
Bulk earthworks for an apartment site, excluding the basement excavation, which is the basement fragment. Follows demolition when that item exists.
Activities, in order: cut.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.underpinning-2

Underpinning (apartments) is a construction fragment on `substructure.underpinning`.
Underpin an adjoining building before apartment excavation. Urban sites with a party wall or a close neighbour.
Activities, in order: underpin.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.piles-2

Piles (apartments) is a construction fragment on `substructure.piles`.
Bored or driven piles for a Class 2 building, after bulk earthworks. Screw piles to a house are not this row.
Activities, in order: establish, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.basement-2

Basement retention (apartments) is a construction fragment on `substructure.basement-retention`.
Shoring, bulk basement excavation, anchors and the basement slab for a Class 2 building. Follows piles when that work item exists. Sized for one to two basement levels; a third level should use the high end and may still be short. Not scaled by storeys, because rise in storeys includes the tower.
Activities, in order: shoring, excavate, anchors, slab, cure.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.footings-2

Footings (apartments) is a construction fragment on `substructure.footings`.
Pad or raft footings where the Class 2 building is not sitting on a basement slab. Tied to earthworks. A project with only a basement may still carry this work item for the core or the lift pits; the pits themselves are the lift-shaft fragment.
Activities, in order: pour.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.slab-2

Slab on ground (apartments) is a construction fragment on `substructure.slab-on-ground`.
A ground or podium slab on ground for a Class 2 building that is not the basement slab. The basement slab is in the basement fragment.
Activities, in order: pour.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lift-shafts-2

Lift shafts and pits (apartments) is a construction fragment on `vertical-transport.lift-shafts-pits`.
Lift pits and shafts built with the basement and the structure. Follows the basement. The lift car and rails are the passenger-lift fragment, which is anchored at the start so manufacture can run early.
Activities, in order: pits, shafts.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.structure-concrete-2

Concrete structure (apartments) is a construction fragment on `structure.concrete`.
The apartment floor cycle after the basement, plus a fixed allowance for the roof slab and plant plinths. One cycle activity is multiplied by rise in storeys, so it must not be split into several cycled activities or the storeys would be counted twice.
Activities, in order: cycle, roof.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.precast-2

Precast (apartments) is a construction fragment on `structure.precast`.
Precast walls or facade panels for a Class 2 building, after the in-situ structure starts being able to receive them. The lead is shop time after that point and is conservative if the order was placed earlier.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stairs-2

Stairs (apartments) is a construction fragment on `access-egress.stairs-and-ramps`.
Fire stairs and common ramps built with the concrete structure. The duration is per storey.
Activities, in order: stairs.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-2

Roof (apartments) is a construction fragment on `envelope.roof-coverings`.
A trafficable or membrane roof to the top of a Class 2 building, after the roof slab. Fixed, because there is one roof, not one per storey.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.facade-2

Facade (apartments) is a construction fragment on `envelope.external-walls`.
The apartment facade after the structure, finish-to-start. In practice the facade often starts before the last floor is poured. This link waits for the structure and is conservative. Scaled by floor area as a proxy for wall area.
Activities, in order: facade.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.windows-2

Windows (apartments) is a construction fragment on `envelope.windows-doors`.
Apartment windows or a window wall, procured after the structure. The lead is the remaining delivery, not the full time from the shop drawing start.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.balcony-waterproof-2

Balcony waterproofing (apartments) is a construction fragment on `envelope.external-waterproofing`.
Waterproof balconies, the podium and external wet areas after the facade zone is available.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.insulation-2

Insulation (apartments) is a construction fragment on `envelope.thermal-insulation`.
Facade and roof insulation to a Class 2 building, with the facade.
Activities, in order: insulate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-drainage-2

Roof drainage (apartments) is a construction fragment on `hydraulic.roof-drainage`.
Roof outlets, overflow and downpipes after the roof membrane.
Activities, in order: drains.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.linings-2

Walls and linings (apartments) is a construction fragment on `interiors.walls-linings`.
Party walls, corridors and apartment partitions after the facade has started to close the building. Scaled by floor area.
Activities, in order: line.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ceilings-2

Ceilings (apartments) is a construction fragment on `interiors.ceilings`.
Apartment and corridor ceilings after wall linings.
Activities, in order: ceilings.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.wet-area-2

Wet-area waterproofing (apartments) is a construction fragment on `interiors.wet-area-waterproofing`.
Waterproof bathrooms and laundries in each apartment, after linings and before tiling.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.internal-doors-2

Internal doors (apartments) is a construction fragment on `interiors.internal-doors`.
Apartment entry doors and internal doors after linings. Fire-rated entries are included.
Activities, in order: hang.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-sanitary-2

Sanitary plumbing (apartments) is a construction fragment on `hydraulic.sanitary`.
Sanitary stacks and apartment rough-in, then fixture fit-off. Starts once the structure is up so the risers can rise with the floors. The lag stands in for linings and waterproofing.
Activities, in order: rough-in, fit-off, commission.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-cold-2

Cold water (apartments) is a construction fragment on `hydraulic.cold-water`.
Cold-water risers, meters and in-apartment pipework. The authority connection is the services-connection fragment.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-hot-2

Hot water (apartments) is a construction fragment on `hydraulic.hot-water`.
Central or local hot water. The first activity includes plant delivery. Fit-off is the apartments.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-2

Electrical wiring (apartments) is a construction fragment on `electrical.cabling-wiring`.
Apartment and common cable rough-in and fit-off after the structure. The main switchboard is a separate long-lead fragment.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lighting-2

Lighting (apartments) is a construction fragment on `electrical.lighting`.
Common and apartment lighting after linings.
Activities, in order: fit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.sprinklers-2

Sprinklers (apartments) is a construction fragment on `fire-active.sprinklers`.
Sprinkler mains, range pipes and heads for a Class 2 building, after the structure. Commissioning is the last activity. A house does not use this row.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydrants-2

Hydrants (apartments) is a construction fragment on `fire-active.hydrants`.
Hydrant riser and boosters for a Class 2 building, with the structure.
Activities, in order: hydrants.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.detection-2

Detection (apartments) is a construction fragment on `fire-active.detection`.
Detection and occupant warning to common areas and, where required, the apartments. Smoke alarms inside sole-occupancy units are the smoke-alarm fragment.
Activities, in order: detect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.smoke-alarms-2

Smoke alarms (apartments) is a construction fragment on `fire-active.smoke-alarms`.
Smoke alarms inside apartments, after linings. Common-area detection is the detection fragment.
Activities, in order: alarms.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ac-2

Air-conditioning (apartments) is a construction fragment on `mechanical.air-conditioning`.
Apartment air-conditioning. Plant is ordered from the start of construction so the lead can run, and the install and commission follow that lead. On a long tower the high lead should be used or the install will show early.
Activities, in order: procure, install, commission.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.carpark-vent-2

Carpark ventilation (apartments) is a construction fragment on `mechanical.carpark-ventilation`.
Carpark ventilation and its controls, after the basement is poured.
Activities, in order: vent.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.comms-2

Communications (apartments) is a construction fragment on `comms-security.structured-cabling`.
Apartment communications riser and in-unit cabling, after linings.
Activities, in order: cable.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tiling-2

Tiling (apartments) is a construction fragment on `interiors.tiling`.
Apartment bathroom and laundry tiling after waterproofing.
Activities, in order: tile.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.joinery-2

Joinery (apartments) is a construction fragment on `interiors.joinery`.
Kitchen and vanity packages. Shop time starts after linings, which is conservative against an order placed from the construction documents.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-2

Painting (apartments) is a construction fragment on `interiors.painting`.
Apartment and common-area paint after joinery.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.floors-2

Floor finishes (apartments) is a construction fragment on `interiors.floor-finishes`.
Apartment floor finishes after paint.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.landscape-2

Landscape (apartments) is a construction fragment on `site.landscape`.
Podium and ground landscaping after the facade, so it can overlap the internal fit-out.
Activities, in order: landscape.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stormwater-2

Stormwater (apartments) is a construction fragment on `site.stormwater-detention`.
On-site detention and the connection for an apartment site, after bulk earthworks.
Activities, in order: detention.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.flood-2

Flood works (apartments) is a construction fragment on `site.flooding`.
Flood protection for a Class 2 site that is in a mapped flood hazard. Only when that fact is true.
Activities, in order: protect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.demolition-7b

Demolition (warehouse) is a construction fragment on `site.demolition`.
Clear existing structures or hardstand before a new Class 7b warehouse. Remove only.
Activities, in order: demolish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.earthworks-7b

Earthworks (warehouse) is a construction fragment on `site.earthworks`.
Pad preparation for a warehouse. Fixed, and sized for roughly 2,000 to 8,000 square metres. A much larger estate should use the high end and may still be short.
Activities, in order: pad.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.footings-7b

Footings (warehouse) is a construction fragment on `substructure.footings`.
Pad footings for a portal frame, before the steel arrives.
Activities, in order: pour.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.steel-7b

Steel portal frame (warehouse) is a construction fragment on `structure.steel`.
Fabricate and erect a portal frame after the pads. The slab is poured after the roof, which is a common industrial sequence so the slab is cast under cover.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-7b

Roof (warehouse) is a construction fragment on `envelope.roof-coverings`.
Metal roof to a portal frame, after erection.
Activities, in order: roof.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.cladding-7b

Wall cladding (warehouse) is a construction fragment on `envelope.external-walls`.
Metal wall cladding after the frame, in parallel with the roof.
Activities, in order: clad.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.slab-7b

Slab on ground (warehouse) is a construction fragment on `substructure.slab-on-ground`.
The warehouse slab, poured after the roof is on. Joints, vapour barrier and a trim are included. Fixed for a building of a few thousand square metres because the library cannot express a rate below one hundredth of a day per square metre.
Activities, in order: prepare, pour, cure.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.vehicle-doors-7b

Vehicle doors (warehouse) is a construction fragment on `envelope.vehicle-doors`.
Industrial doors after the cladding opening exists.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-sanitary-7b

Amenities plumbing (warehouse) is a construction fragment on `hydraulic.sanitary`.
A small amenities block in a warehouse, after the slab. Not an apartment riser.
Activities, in order: amenities.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-7b

Electrical (warehouse) is a construction fragment on `electrical.cabling-wiring`.
Power and the high-bay circuit wiring after the frame. Luminaires are the lighting fragment.
Activities, in order: wire.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lighting-7b

High-bay lighting (warehouse) is a construction fragment on `electrical.lighting`.
High-bay and office lighting after the cladding has closed the building.
Activities, in order: lights.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.sprinklers-7b

Sprinklers (warehouse) is a construction fragment on `fire-active.sprinklers`.
Warehouse sprinklers, which may be an ESFR or in-rack system. The range is wide because storage height changes the system. Commissioning is included.
Activities, in order: sprinklers.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydrants-7b

Hydrants (warehouse) is a construction fragment on `fire-active.hydrants`.
Site hydrants and hose reels for a warehouse, after the slab so the yard pipework has a finished level.
Activities, in order: hydrants.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stormwater-7b

Stormwater (warehouse) is a construction fragment on `site.stormwater-detention`.
Industrial stormwater and detention, after the pad is cut.
Activities, in order: detention.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.landscape-7b

Landscape (warehouse) is a construction fragment on `site.landscape`.
A modest industrial landscape and the street interface, after cladding.
Activities, in order: landscape.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-7b

Painting (warehouse) is a construction fragment on `interiors.painting`.
Line marking and the small office paint. Not a commercial fit-out paint package.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.demolition-9b

Demolition (school) is a construction fragment on `site.demolition`.
Demolish existing structures before a new school building. Remove only. A school refurbishment uses the fit-out strip-out instead.
Activities, in order: demolish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.earthworks-9b

Earthworks (school) is a construction fragment on `site.earthworks`.
Bulk earthworks for a low-rise school building. Playing fields are not included unless they are this work item.
Activities, in order: cut.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.piles-9b

Piles (school) is a construction fragment on `substructure.piles`.
Piles for a school building where the ground needs them. Follows earthworks.
Activities, in order: install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.footings-9b

Footings (school) is a construction fragment on `substructure.footings`.
Pad and strip footings for a low-rise school.
Activities, in order: pour.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.slab-9b

Slab on ground (school) is a construction fragment on `substructure.slab-on-ground`.
Ground slab for a low-rise school, after footings, including cure.
Activities, in order: pour, cure.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.structure-concrete-9b

Concrete structure (school) is a construction fragment on `structure.concrete`.
A low-rise school frame, one to four storeys. The cycle includes slabs and walls for that building. It is not an apartment tower cycle.
Activities, in order: frame.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.steel-9b

Steel structure (school) is a construction fragment on `structure.steel`.
A steel school frame where that is the structure instead of, or as well as, concrete. Follows the footings. A hall roof often sits on this frame.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.roof-9b

Roof (school) is a construction fragment on `envelope.roof-coverings`.
The school roof after whichever structure is present. Listed after both the concrete and steel fragments so the chain falls to the one that exists.
Activities, in order: roof.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.external-walls-9b

External walls (school) is a construction fragment on `envelope.external-walls`.
Masonry or lightweight external walls after the roof is on, for a low-rise school.
Activities, in order: walls.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.windows-9b

Windows (school) is a construction fragment on `envelope.windows-doors`.
School windows after the external walls. The lead is remaining delivery.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.bushfire-9b

Bushfire construction (school) is a construction fragment on `envelope.bushfire-construction`.
Extra bushfire construction to a school on bushfire-prone land, after the external walls.
Activities, in order: screen.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stairs-9b

Stairs (school) is a construction fragment on `access-egress.stairs-and-ramps`.
Stairs and ramps after the frame. Per storey, so a single-storey school is a short duration.
Activities, in order: stairs.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.linings-9b

Walls and linings (school) is a construction fragment on `interiors.walls-linings`.
Classroom and corridor partitions after the building is closed in.
Activities, in order: line.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ceilings-9b

Ceilings (school) is a construction fragment on `interiors.ceilings`.
Classroom and corridor ceilings after partitions.
Activities, in order: ceilings.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.wet-area-9b

Wet areas (school) is a construction fragment on `interiors.wet-area-waterproofing`.
Student amenities and staff wet areas, after partitions.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.joinery-9b

Joinery (school) is a construction fragment on `interiors.joinery`.
Classroom joinery and reception, after partitions. A fixed range including a modest shop lead.
Activities, in order: install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-9b

Painting (school) is a construction fragment on `interiors.painting`.
Paint after joinery.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.floors-9b

Floor finishes (school) is a construction fragment on `interiors.floor-finishes`.
Resilient floors and carpet after paint.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-sanitary-9b

Sanitary plumbing (school) is a construction fragment on `hydraulic.sanitary`.
Amenities and classroom sink rough-in and fit-off.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-9b

Electrical (school) is a construction fragment on `electrical.cabling-wiring`.
School power and data pathway rough-in. Communications cabling is its own fragment. The switchboard lead is the shared late switchboard row for this class.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lighting-9b

Lighting (school) is a construction fragment on `electrical.lighting`.
Classroom and common lighting after ceilings.
Activities, in order: fit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ac-9b

Air-conditioning (school) is a construction fragment on `mechanical.air-conditioning`.
School air-conditioning. Plant lead then install and commission, starting once the slab exists so the lead is not waiting on finishes.
Activities, in order: procure, install, commission.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.sprinklers-9b

Sprinklers (school) is a construction fragment on `fire-active.sprinklers`.
Sprinklers to a school building where the work item includes them, after the roof.
Activities, in order: sprinklers.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.detection-9b

Detection (school) is a construction fragment on `fire-active.detection`.
Detection, occupant warning and the fire indicator panel for a school.
Activities, in order: detect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.comms-9b

Communications (school) is a construction fragment on `comms-security.structured-cabling`.
Structured cabling to teaching spaces, after partitions.
Activities, in order: cable.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.landscape-9b

Landscape (school) is a construction fragment on `site.landscape`.
Play areas, paths and planting. This is often on the critical path of a school and is longer than a house landscape.
Activities, in order: landscape.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.stormwater-9b

Stormwater (school) is a construction fragment on `site.stormwater-detention`.
School stormwater and detention, after earthworks.
Activities, in order: detention.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.strip-out-fitout

Strip-out (fit-out) is a construction fragment on `site.demolition`.
Isolate services, soft-strip the tenancy or school rooms, and make good the base building. Remove only, and only for a refurbishment. A new commercial shell does not use this row. The three rates sum to a few weeks for about 500 to 2,000 square metres.
Activities, in order: isolate, strip, makegood.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.structure-alterations

Structural alterations is a construction fragment on `structure.alterations`.
Openings, strengthening and protection of retained structure during a refurbishment. Heritage fabric is included as protection of what is being kept. The permit itself is the project-level heritage fragment, not this row.
Activities, in order: alter.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.partitions-fitout

Partitions (fit-out) is a construction fragment on `interiors.walls-linings`.
New and altered partitions for a Class 5 or 6 fit-out, or a school refurbishment, after the strip-out.
Activities, in order: frame.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ceilings-fitout

Ceilings (fit-out) is a construction fragment on `interiors.ceilings`.
New or altered ceilings after partitions. Grid and tile, or set plaster, is the same range.
Activities, in order: ceilings.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.wet-area-fitout

Wet areas (fit-out) is a construction fragment on `interiors.wet-area-waterproofing`.
Amenities waterproofing in a fit-out, after partitions.
Activities, in order: membrane.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.internal-doors-fitout

Doors (fit-out) is a construction fragment on `interiors.internal-doors`.
Suite entry and internal doors after partitions.
Activities, in order: hang.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-sanitary-fitout

Sanitary alterations (fit-out) is a construction fragment on `hydraulic.sanitary`.
Alter sanitary stacks and fit new amenities fixtures. The lag allows partitions and waterproofing.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-cold-fitout

Cold-water alterations (fit-out) is a construction fragment on `hydraulic.cold-water`.
Alter cold water for a new amenities or tea point. Not a new incoming main.
Activities, in order: alter.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.hydraulic-hot-fitout

Hot-water alterations (fit-out) is a construction fragment on `hydraulic.hot-water`.
A local hot-water unit for a fit-out amenities.
Activities, in order: unit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.electrical-fitout

Electrical alterations (fit-out) is a construction fragment on `electrical.cabling-wiring`.
Alter power and the tenancy distribution after the strip. The tenancy switchboard, if replaced, is the fit-out switchboard fragment.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lighting-fitout

Lighting (fit-out) is a construction fragment on `electrical.lighting`.
New luminaires after the ceiling grid.
Activities, in order: fit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.mechanical-fitout

Air-conditioning alterations (fit-out) is a construction fragment on `mechanical.air-conditioning`.
Adjust or extend the base-building air-conditioning for the new layout. A full plant replacement should use the high end.
Activities, in order: rough-in, fit-off.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ventilation-fitout

Exhaust ventilation (fit-out) is a construction fragment on `mechanical.ventilation`.
Toilet exhaust and local exhaust for a fit-out, after the strip.
Activities, in order: exhaust.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.sprinklers-fitout

Sprinkler alterations (fit-out) is a construction fragment on `fire-active.sprinklers`.
Relocate heads and range pipes to the new ceiling layout, then test. A new system for a shell is not this row.
Activities, in order: heads.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.detection-fitout

Detection alterations (fit-out) is a construction fragment on `fire-active.detection`.
Relocate detectors and sounders to the new layout and update the cause-and-effect if needed.
Activities, in order: detectors.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.comms-fitout

Communications (fit-out) is a construction fragment on `comms-security.structured-cabling`.
Structured cabling to the new workstation layout, after partitions.
Activities, in order: cable.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tiling-fitout

Tiling (fit-out) is a construction fragment on `interiors.tiling`.
Amenities tiling after waterproofing.
Activities, in order: tile.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.joinery-fitout

Joinery (fit-out) is a construction fragment on `interiors.joinery`.
Reception, breakout and workstations joinery. Shop time is included and starts after partitions, which is conservative.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.painting-fitout

Painting (fit-out) is a construction fragment on `interiors.painting`.
Paint after joinery.
Activities, in order: paint.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.floors-fitout

Floor finishes (fit-out) is a construction fragment on `interiors.floor-finishes`.
Carpet tile or resilient flooring after paint.
Activities, in order: lay.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.fire-stopping

Fire stopping is a construction fragment on `fire-passive.penetrations`.
Seal penetrations after services are through the walls and slabs. Placed late so it sits near completion. Classes that commonly need it are apartments, offices, shops and schools.
Activities, in order: seal.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.gas-commercial

Gas (commercial) is a construction fragment on `hydraulic.gas`.
Consumer gas to a commercial kitchen, a school lab or an apartment plant room. Appears only when that work item exists.
Activities, in order: pipe.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.lifts

Passenger lifts is a construction fragment on `vertical-transport.passenger-lifts`.
Manufacture, install and commission passenger lifts. Anchored at the start of construction so the factory time runs from the beginning. The typical lead lands installation around the end of a mid-rise structure. A tall tower should use the high end. Shafts and pits are a separate fragment.
Activities, in order: procure, install, commission.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.switchboard-2

Main switchboard (apartments) is a construction fragment on `electrical.switchboards`.
A Class 2 main switchboard, ordered from the start of construction. A house board is fr.switchboard-1a.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.switchboard-major

Main switchboard (commercial or school) is a construction fragment on `electrical.switchboards`.
A main switchboard for a warehouse, school or commercial building, ordered from the start of construction. Fit-out distribution boards that are not a main board should use the high end only if they are genuinely long-lead.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.utility-1a

Utility supply (house) is a construction fragment on `electrical.utility-supply`.
The permanent electrical connection to a house, applied for at the start of construction. The temporary board is a different fragment.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.utility-major

Utility supply (larger building) is a construction fragment on `electrical.utility-supply`.
Permanent supply for an apartment building, warehouse or school. Applications start with construction. A chamber substation is the high end.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.services-connections-1a

Services connections (house) is a construction fragment on `site.services-connections`.
Water, sewer and communications lead-in to a house, applied for at the start. Electrical supply is the utility fragment.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.services-connections-major

Services connections (larger building) is a construction fragment on `site.services-connections`.
Water, sewer and communications lead-in for an apartment building, warehouse, school or fit-out upgrade. Starts with construction.
Activities, in order: procure, install.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


## Durations

The owner accepted these duration rows on 24 October 2026. Each row in `durations.yaml` is `status: reviewed`. The four statutory clocks stay `verified: false`.


### du.concrete-cure

Cure a concrete slab or footing before the next trade loads it.
Low 7, typical 7, high 14. A fixed range in calendar days, so weekends count.
Seven calendar days is the ordinary minimum before a following trade loads a house or podium slab. The high end allows cool weather or a slower mix. It is not a cylinder-test result.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-1a

Lag between rough-in and fit-off on a Class 1a house.
Low 8, typical 12, high 20. A fixed range in working days. It does not grow with floor area.
Stands in for linings and wet-area waterproofing, which are their own fragments. It is an allowance, not a link to those activities.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-2

Lag between rough-in and fit-off on a Class 2 apartment building.
Low 15, typical 30, high 50. A fixed range in working days. It does not grow with floor area.
Allows the floors to be lined and wet areas waterproofed before fit-off. A tall building should use the high end. It is not a link to the lining fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-fitout

Lag between rough-in and fit-off on a commercial fit-out.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
Sized for one floor or a modest tenancy. A multi-floor fit-out in occupation should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-9b

Lag between rough-in and fit-off on a low-rise school.
Low 10, typical 20, high 35. A fixed range in working days. It does not grow with floor area.
Allows partitions and ceilings to close in before fixtures are fitted. Sized for one building, not a campus of separate blocks.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-7b

Lag between rough-in and fit-off in a warehouse.
Low 5, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
Warehouse services are a small amenities and lighting package, so the close-in gap is short.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.closein-reno

Lag between rough-in and fit-off on a house renovation.
Low 5, typical 8, high 12. A fixed range in working days. It does not grow with floor area.
Sized for a substantial house renovation, not a single-room repair.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.dc-site-overlap

Lag from the start of design-and-construct documentation to the start of construction.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
Used as a start-to-start lag so early site work can begin while the builder finishes documents. Ten to twenty-five working days is an ordinary early-works overlap, not a measured study.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.eci-overlap

Lag from early-contractor appointment to the continuation of approval documentation.
Low 10, typical 15, high 20. A fixed range in working days. It does not grow with floor area.
Used as a start-to-start lag. The documentation stage starts a short time after the contractor is engaged, rather than after procurement finishes.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.da-assessment-period

Deemed-refusal assessment period for a NSW development application.
Low 40, typical 60, high 90 calendar days. Instrument: Environmental Planning and Assessment Regulation 2021, section 91.
Forty days is the ordinary local-development period in section 91(4). Sixty days covers designated development, most integrated development, concurrence and the biodiversity-credit case in section 91(2). Ninety days is the State significant period in section 91(3). As at the public consolidation read for this draft.
Marked verified false. This is the legislated clock as read from a public consolidation, not a check of the in-force text, and not the allowance a program should carry once an information request stops the clock.


### du.da-exhibition-period

Minimum public exhibition for a NSW development application that is not State significant.
Low 14, typical 14, high 28 calendar days. Instrument: Environmental Planning and Assessment Act 1979, Schedule 1, clauses 7, 8 and 8A.
Clause 7 is 14 days unless the community participation plan says otherwise. Designated development, nominated integrated development and threatened-species development are 28 days under clauses 8 and 8A. As at the public consolidation read for this draft.
Marked verified false. This is the legislated clock as read from a public consolidation, not a check of the in-force text, and not the allowance a program should carry once an information request stops the clock.


### du.cdc-determination-period

Time for a certifier to determine a complying development certificate.
Low 10, typical 20, high 20 calendar days. Instrument: Environmental Planning and Assessment Regulation 2021, section 133.
Section 133 gives 20 days where section 134 applies and 10 days otherwise. Part 3BA of the Codes SEPP is also 10 days. As at the public consolidation read for this draft.
Marked verified false. This is the legislated clock as read from a public consolidation, not a check of the in-force text, and not the allowance a program should carry once an information request stops the clock.


### du.ssd-exhibition-period

Minimum public exhibition for State significant development.
Low 14, typical 28, high 28 calendar days. Instrument: Environmental Planning and Assessment Act 1979, Schedule 1, clause 9.
Clause 9 sets 28 days, or the period in the Planning Secretary's community participation plan for a relevant SSD application, and that plan period must be at least 14 days. As at the public consolidation read for this draft.
Marked verified false. This is the legislated clock as read from a public consolidation, not a check of the in-force text, and not the allowance a program should carry once an information request stops the clock.


### du.da-program-allowance

Program allowance for development-application assessment in metropolitan NSW.
Low 40, typical 90, high 180. A fixed range in calendar days, so weekends count.
The low end is the ordinary 40-day deemed-refusal clock. Typical and high are planning allowances for requests for information and a metropolitan council or planning panel. They are not legislated.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ssd-program-allowance

Program allowance for State significant assessment after exhibition.
Low 90, typical 180, high 365. A fixed range in calendar days, so weekends count.
The low end matches the 90-day deemed-refusal period for SSD. Typical and high allow agency comments, a response to submissions and a long determination. They are estimates.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.defects-liability

Defects liability period after practical completion.
Low 90, typical 365, high 730. A fixed range in calendar days, so weekends count.
Ninety calendar days is about a 13-week residential defects period. One year is the ordinary commercial defects liability period. Two years is used on some larger contracts. This is not the Home Building Act statutory warranty, which is a separate obligation.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.temporary-power

Connect a temporary builders supply.
Low 1, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
A site board on an urban lot. A new substation is the utility-supply fragment, not this one.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.erosion-control

Install erosion and sediment controls.
Low 1, typical 2, high 5. A fixed range in working days. It does not grow with floor area.
A house lot is the low end. A large disturbed site is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tree-protection

Install tree protection.
Low 1, typical 2, high 5. A fixed range in working days. It does not grow with floor area.
A few trees on a house lot is the low end. A row of street trees or a large site is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.occupied-protection

Hoardings and protection in an occupied building.
Low 2, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A single-tenancy fit-out is the typical end. A building that stays partly occupied across several floors is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.demolition-1a

Demolish a house or outbuilding.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single dwelling and garage. A heavily built or asbestos-affected house is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.strip-out-1a

Strip out a house renovation.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
A bathroom and kitchen renewal is the low end. A whole-house strip is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.earthworks-1a

House site cut and fill.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A near-level suburban lot. A steep site or substantial retaining is the high end and may still be short.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.underpinning-1a

Underpin an existing house.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A short length of wall is the low end. A whole party wall or poor ground is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.footings-excavate-1a

Excavate and inspect house footings.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
Includes a short allowance for the certifier's inspection.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.footings-pour-1a

Pour house footings.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
A single house. Piers on reactive clay can use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.termite-1a

Install a house termite barrier.
Low 1, typical 1, high 2. A fixed range in working days. It does not grow with floor area.
One system to the perimeter and penetrations. A complex plan is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.damp-1a

Install a house damp-proof membrane.
Low 1, typical 1, high 2. A fixed range in working days. It does not grow with floor area.
A single slab. A stepped slab is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stormwater-1a

House stormwater and on-site detention.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
A charged or gravity connection without a large tank is typical. A detention tank is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.flood-1a

Flood protection to a house.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
Raised levels already in the earthworks are not counted again. A barrier or a flood gate is the typical case.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-setout-1a

Set out and prepare a house slab.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
Includes boxing and a thin bedding layer. A stepped slab is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.under-slab-services-1a

Under-slab sanitary drainage for a house.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
A single dwelling. A secondary dwelling on the same slab is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-pour-1a

Reinforce and pour a Class 1a slab on ground.
Low 0.02, typical 0.03, high 0.05. Working days per square metre of gross floor area. At 250 m² the typical rate is about 8 days before rounding up.
At about 200 to 300 m² this is several working days for the pour itself. It excludes curing and wet weather.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.frame-walls-1a

Stand timber wall frames to a house.
Low 0.02, typical 0.03, high 0.05. Working days per square metre of gross floor area. At 250 m² the typical rate is about 8 days before rounding up.
Prefabricated frames. Site-cut framing should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.frame-roof-1a

Install roof trusses and bracing to a house.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A simple hip or gable. A complex roof is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.masonry-1a

Lay brick veneer to a Class 1a house.
Low 0.03, typical 0.05, high 0.08. Working days per square metre of gross floor area. At 250 m² the typical rate is about 12 days before rounding up.
Face brick to a single dwelling. Feature courses and a two-storey skin use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-sark-1a

Sark a house roof.
Low 1, typical 2, high 3. A fixed range in working days. It does not grow with floor area.
A single roof. A very cut-up roof is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-cover-1a

Install a house roof cover.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Concrete tile or metal sheet to a single dwelling. Slate or a complex roof is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.external-walls-1a

Wrap and clad a house.
Low 0.02, typical 0.03, high 0.05. Working days per square metre of gross floor area. At 250 m² the typical rate is about 8 days before rounding up.
Lightweight cladding. A brick skin is fr.masonry-1a and is not in this rate.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.insulation-1a

Insulate a Class 1a house.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Batts to walls and ceilings. A high-performance build is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-lead-1a

Remaining delivery of house windows after the frame.
Low 10, typical 15, high 25. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Standard residential aluminium. Imported or custom timber joinery is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-install-1a

Install house windows and external doors.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
A single dwelling. A highly glazed house is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.bushfire-1a

Bushfire construction measures to a house.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
Screens, a rated door and sarking upgrades for a lower BAL. Flame-zone construction is the high end and may still be short.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-drainage-1a

House gutters and downpipes.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
A simple hip roof. Box gutters are the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.partitions-1a

Frame internal partitions in a house.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Prefabricated or simple site framing.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sheet-1a

Sheet and set plasterboard in a house.
Low 0.03, typical 0.04, high 0.06. Working days per square metre of gross floor area. At 250 m² the typical rate is about 10 days before rounding up.
Walls and the set. Ceilings are the ceilings fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ceilings-1a

Install house ceilings.
Low 0.02, typical 0.03, high 0.04. Working days per square metre of gross floor area. At 250 m² the typical rate is about 8 days before rounding up.
Plasterboard ceilings. A raked ceiling is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.wet-area-1a

Waterproof house wet areas.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
Two bathrooms and a laundry. Extra bathrooms are the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.internal-doors-1a

Hang internal doors in a house.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
A typical three or four bedroom house.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-rough-1a

Above-slab sanitary rough-in for a house.
Low 3, typical 5, high 8. A fixed range in working days. It does not grow with floor area.
One kitchen, two bathrooms and a laundry.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-fit-1a

Sanitary fit-off for a house.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
Fixtures after the rooms are lined and waterproofed.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cold-rough-1a

Cold-water rough-in for a house.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
A single dwelling on mains pressure.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cold-fit-1a

Cold-water fit-off for a house.
Low 1, typical 2, high 3. A fixed range in working days. It does not grow with floor area.
Taps and the connection to fixtures.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hot-rough-1a

Hot-water rough-in for a house.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
Pipework to one plant position.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hot-fit-1a

Install the house hot-water unit.
Low 1, typical 1, high 2. A fixed range in working days. It does not grow with floor area.
A storage or instantaneous unit. A heat pump is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.gas-1a

House consumer gas piping.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
A cooktop and a heater. No gas on the project means this work item is absent and the fragment does not appear.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-rough-1a

Electrical rough-in for a house.
Low 0.02, typical 0.03, high 0.05. Working days per square metre of gross floor area. At 250 m² the typical rate is about 8 days before rounding up.
A single dwelling. A home with extensive extra circuits is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-fit-1a

Electrical fit-off for a house.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Power points and the switchboard dead-test are included. Luminaires are the lighting fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lighting-1a

Install house luminaires.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A builder's standard fitting package. A decorative package is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.switchboard-1a

Install a house switchboard.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
A single residential board. A large home with extra distribution is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.smoke-alarms-1a

Install house smoke alarms.
Low 1, typical 1, high 2. A fixed range in working days. It does not grow with floor area.
A complying interconnected set to a single dwelling.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-1a

Install house air-conditioning.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
A few split systems is typical. A ducted whole-house system is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tiling-1a

Tile house wet areas.
Low 3, typical 6, high 12. A fixed range in working days. It does not grow with floor area.
Two bathrooms, a laundry and a splashback. Large-format or extra bathrooms are the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-lead-1a

House joinery manufacture after linings.
Low 10, typical 15, high 25. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A standard kitchen and vanities. A highly custom kitchen is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-install-1a

Install house joinery.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One kitchen, vanities and robes.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-1a

Paint a Class 1a house.
Low 0.03, typical 0.05, high 0.08. Working days per square metre of gross floor area. At 250 m² the typical rate is about 12 days before rounding up.
Three coats to a new house. A repaint of sound surfaces is shorter and is the renovation row.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floors-1a

Lay floor finishes in a house.
Low 0.01, typical 0.02, high 0.04. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A mix of hard flooring and carpet. Parquetry or a full sand and finish is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.landscape-1a

House landscaping.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
A suburban front and rear yard. A pool surround is site.pools and is not in this row.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.linings-1a-reno

Reline a house renovation.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A few rooms is typical. A whole-house reline is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.wet-area-1a-reno

Waterproof renovated house bathrooms.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
One or two bathrooms.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hydraulic-rough-reno

Sanitary alterations rough-in for a house renovation.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One kitchen and one or two bathrooms.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hydraulic-fit-reno

Sanitary fit-off for a house renovation.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
New fixtures to the altered rooms.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.water-1a-reno

Alter cold water in a house renovation.
Low 2, typical 3, high 6. A fixed range in working days. It does not grow with floor area.
Re-pipe the altered rooms, not a whole-house replumb, unless the high end is used.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-rough-reno

Electrical rough-in for a house renovation.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
The altered rooms, plus a short allowance for making safe.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-fit-reno

Electrical fit-off for a house renovation.
Low 2, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
New outlets in the altered rooms.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-1a-reno

Replacement kitchen or vanity in a house renovation.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
Includes a short shop lead. A full custom kitchen is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-1a-reno

Paint a house renovation.
Low 4, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
The altered rooms. A whole-house repaint is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floors-1a-reno

Floor finishes in a house renovation.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
The altered rooms. A whole-house sand and finish is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.demolition-2

Demolition before a Class 2 building.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A low-rise existing building on an urban site. A multi-storey knockdown is the high end and may still be short.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.earthworks-2

Bulk earthworks for an apartment site.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A typical urban pad. Basement dig is not included.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.underpinning-2

Underpin adjoining structures for an apartment site.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
One adjoining wall is typical. Several boundaries are the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.piles-platform-2

Establish a piling platform.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
An urban site with access already in place is the low end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.piles-install-2

Install piles for a Class 2 building.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A mid-rise footprint. A deep basement or a large footprint is the high end. Not scaled, because pile count is not a profile fact.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.basement-shoring-2

Install basement retention.
Low 15, typical 25, high 45. A fixed range in working days. It does not grow with floor area.
Bored piers or sheet piles to one or two levels.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.basement-excavate-2

Excavate an apartment basement.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
One to two levels on an urban site.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.basement-anchors-2

Install basement anchors or props.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
Temporary anchors. A propped system that stays longer is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.basement-slab-2

Pour an apartment basement slab.
Low 8, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
The lowest slab, including a short allowance for the waterproofing under it.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.footings-2

Apartment footings outside the basement slab.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A low-rise walk-up without a basement is the typical case.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-2

Apartment ground or podium slab on ground.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A walk-up or a podium. The tower floors are the concrete floor cycle.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lift-pits-2

Form apartment lift pits.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
One or two pits.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lift-shafts-2

Build apartment lift shafts as the floors rise.
Low 1, typical 2, high 3. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 16 days before rounding up.
A day or two per storey beside the floor cycle, not a second full structure.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floor-cycle-2

Apartment floor cycle.
Low 6, typical 9, high 12. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 72 days before rounding up.
An ordinary post-tensioned or conventional floor. A transfer slab or a slow jump-form should use the high end. Basement levels are not in this cycle.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.structure-roof-2

Apartment roof slab and plant plinths.
Low 8, typical 12, high 20. A fixed range in working days. It does not grow with floor area.
The last level and the plant bases, once, not per storey.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.precast-lead-2

Precast manufacture for an apartment building.
Low 20, typical 40, high 70. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A mid-rise panel package. A fully panelised facade is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.precast-install-2

Erect precast as the floors are ready.
Low 1, typical 2, high 3. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 16 days before rounding up.
About one to three days per storey of erection, not the shop time.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stairs-2

Form apartment stairs per storey.
Low 1, typical 2, high 4. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 16 days before rounding up.
Concrete fire stairs beside the floor cycle. A feature stair is extra and should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-2

Apartment roof membrane.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One roof. A landscaped roof or a complex plant deck is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.facade-2

Install a Class 2 facade.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Precast, masonry or a window wall. A full curtain wall is the curtain-wall fragment and is not this rate.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-lead-2

Apartment window delivery after the structure.
Low 15, typical 25, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A mid-rise residential window package.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-install-2

Install apartment windows.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Installation only, after delivery.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.balcony-waterproof-2

Waterproof apartment balconies and the podium.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Uses floor area as a proxy. A building with deep planters is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.insulation-2

Insulate a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Typical apartment insulation. A high-performance facade is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-drainage-2

Apartment roof drainage.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One roof and the podium outlets.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.linings-2

Line a Class 2 building.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Party walls and partitions to a mid-rise building. The rate is days per square metre of floor, not of wall.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ceilings-2

Install apartment ceilings.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Plasterboard ceilings. A heavy feature-ceiling package is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.wet-area-2

Waterproof apartment wet areas.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Uses whole-building floor area as a proxy for the wet rooms. A building with large ensuites should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.internal-doors-2

Hang doors in a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A typical unit mix. Solid-core fire doors throughout are the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-rough-2

Sanitary rough-in for a Class 2 building.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Stacks and in-apartment pipework.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-fit-2

Sanitary fit-off for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Fixtures after wet areas are waterproofed.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-commission-2

Commission apartment sanitary systems.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Stack testing and a fixture sample. Not the occupation certificate.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cold-rough-2

Cold-water rough-in for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Risers and branch pipework.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cold-fit-2

Cold-water fit-off for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Meters and fixture connections.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hot-plant-lead-2

Hot-water plant delivery for an apartment building.
Low 15, typical 25, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A central plant or a set of heat pumps. Individual units are the low end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hot-fit-2

Hot-water reticulation and fit-off in apartments.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Risers and connections after the plant arrives.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-rough-2

Electrical rough-in for a Class 2 building.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Risers, apartments and common areas.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-fit-2

Electrical fit-off for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Accessories after linings. Luminaires are the lighting fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lighting-2

Install apartment and common luminaires.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard residential package.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sprinklers-rough-2

Sprinkler rough-in for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Mains and range pipes.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sprinklers-fit-2

Sprinkler heads and commissioning for a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Heads after ceilings, plus a short commission. The second rate includes that commission rather than a third activity.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hydrants-2

Apartment hydrant system.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One riser and a booster. A large site with several risers is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.detection-2

Detection and warning in a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard addressable system.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.smoke-alarms-2

Install smoke alarms in apartments.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
A walk-up is the low end. A large unit count is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-plant-lead-2

Apartment air-conditioning plant delivery from construction start.
Low 40, typical 80, high 140. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Measured from stage start. A few condensers are the low end. Central plant for a taller building is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-install-2

Install apartment air-conditioning.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Fan coil or split units and ductwork.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-commission-2

Commission apartment air-conditioning.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
A sample of apartments plus the plant. Not the occupation certificate.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.carpark-vent-2

Apartment carpark ventilation.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One basement. A multi-level carpark is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.comms-2

Communications cabling in a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard residential backbone and outlets.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tiling-2

Tile apartment wet areas.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
Uses floor area as a proxy for the wet rooms.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-lead-2

Apartment joinery manufacture.
Low 15, typical 25, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A repeated unit package. A high level of variation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-install-2

Install apartment joinery.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Kitchens and vanities after delivery.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-2

Paint a Class 2 building.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A standard scheme. A high-specification finish is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floors-2

Lay floor finishes in a Class 2 building.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Carpet and hard flooring to a typical unit mix.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.landscape-2

Apartment landscaping.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A ground plane and a modest podium. A large deep-soil garden is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stormwater-2

Apartment stormwater detention.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single tank or basin. A water-sensitive system across the site is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.flood-2

Flood protection for an apartment site.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
Local barriers and a pump-out. A site-wide flood strategy is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.demolition-7b

Clear a warehouse site.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
Hardstand and a small existing shed. A large existing warehouse is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.earthworks-7b

Prepare a warehouse pad.
Low 5, typical 10, high 25. A fixed range in working days. It does not grow with floor area.
A level industrial lot. Poor ground or a large cut is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.footings-7b

Warehouse pad footings.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single portal building. A multi-bay building is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.steel-lead-7b

Fabricate a warehouse portal frame.
Low 20, typical 40, high 70. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Shop time after the pads, which is conservative if the mill order was placed at tender.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.steel-erect-7b

Erect a warehouse portal frame.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
Sized for about 2,000 to 8,000 square metres. A larger building should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-7b

Install a warehouse metal roof.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single span building of a few thousand square metres.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cladding-7b

Install warehouse wall cladding.
Low 8, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A single building. A high-bay or a heavily perforated office annex is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-prepare-7b

Prepare a warehouse slab subgrade.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Under cover, after the frame and roof.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-pour-7b

Pour a warehouse slab.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A few thousand square metres. A high-tolerance floor is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.vehicle-doors-lead-7b

Manufacture warehouse vehicle doors.
Low 10, typical 20, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A few roller doors. A large dock package is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.vehicle-doors-install-7b

Install warehouse vehicle doors.
Low 2, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Installation after the openings are framed.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-7b

Warehouse amenities plumbing.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One amenities block. A large office annex is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-7b

Warehouse power cabling.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single warehouse and a small office. A process fit-out is the high end and is often a separate scope.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lighting-7b

Install warehouse lighting.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A single building. A high-lux picking warehouse is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sprinklers-7b

Warehouse sprinkler installation.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A standard hazard roof system is the low end. ESFR or in-rack is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hydrants-7b

Warehouse hydrants.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A single building. A large yard main is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stormwater-7b

Warehouse stormwater detention.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
A single basin or tank.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.landscape-7b

Warehouse landscaping.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A front setback. It is not a civic landscape.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-7b

Warehouse office paint and line marking.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A small annexed office and the aisle marks.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.demolition-9b

Demolition before a new school building.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
One existing building. A campus of several buildings is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.earthworks-9b

Earthworks for a low-rise school.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building pad. A campus regrade is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.piles-9b

Piles for a school building.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A low-rise footprint. A large hall on poor ground is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.footings-9b

School footings.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building of one to four storeys.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.slab-9b

School ground slab.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building. A large hall slab is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.structure-9b

Low-rise school concrete frame per storey.
Low 8, typical 12, high 18. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 96 days before rounding up.
One to four storeys. A long-span hall is not well described by a floor cycle and should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.steel-lead-9b

Fabricate a school steel frame.
Low 20, typical 40, high 60. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A low-rise teaching block or a hall.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.steel-erect-9b

Erect a school steel frame.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
One building. A long-span hall is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.roof-9b

Install a low-rise school roof.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building. A set of separate roofs is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.external-walls-9b

School external walls.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
One building of a few thousand square metres. Not scaled, so a large campus building should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-lead-9b

School window delivery.
Low 15, typical 30, high 50. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A teaching-block package. A fully glazed hall is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.windows-install-9b

Install school windows.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
One building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.bushfire-9b

Bushfire construction to a school.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A lower BAL. Flame zone is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stairs-9b

School stairs and ramps per storey.
Low 2, typical 3, high 5. Working days per storey, multiplied by rise in storeys. At 8 storeys the typical cycle is about 24 days before rounding up.
A low-rise teaching block. A large assembly stair is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.linings-9b

Line a school.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A teaching block of a few thousand square metres. At 3,000 m² the typical rate is about 60 days.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ceilings-9b

Install school ceilings.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard teaching block.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.wet-area-9b

Waterproof school amenities.
Low 3, typical 6, high 12. A fixed range in working days. It does not grow with floor area.
A typical amenities set. A large senior school is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-9b

School joinery.
Low 10, typical 20, high 35. A fixed range in working days. It does not grow with floor area.
A teaching block. A performing-arts fit-out is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-9b

Paint a school.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A standard scheme to one building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floors-9b

Lay school floor finishes.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Vinyl and carpet to a teaching block. A sports floor is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-rough-9b

Sanitary rough-in for a school.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A teaching block amenities set.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-fit-9b

Sanitary fit-off for a school.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Fixtures after amenities are waterproofed.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-rough-9b

Electrical rough-in for a school.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
A teaching block.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-fit-9b

Electrical fit-off for a school.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Accessories after linings.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lighting-9b

Install school lighting.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard teaching-block package.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-lead-9b

School air-conditioning plant delivery.
Low 20, typical 40, high 60. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Packaged plant for one building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-install-9b

Install school air-conditioning.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Ductwork and room units.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ac-commission-9b

Commission school air-conditioning.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One building. Not the occupation certificate.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sprinklers-9b

Sprinklers in a school.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard system to one building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.detection-9b

School detection and warning.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building. A campus network is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.comms-9b

School structured cabling.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A teaching block. A high-density computer room is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.landscape-9b

School landscaping.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
Paths, turf and modest play equipment. A full sports field is the high end and may be a separate package.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.stormwater-9b

School stormwater detention.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One building's catchment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.strip-isolate-fitout

Isolate services for a fit-out strip.
Low 1, typical 2, high 5. A fixed range in working days. It does not grow with floor area.
One tenancy. An occupied building is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.strip-fitout

Soft-strip a fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
At 1,000 m² the typical rate is about 10 days. A heavier base-building strip should use the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.makegood-fitout

Make good after a fit-out strip.
Low 2, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
Patching the slab and capping services. Structural alterations are the alterations fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.structure-alterations

Structural alterations in an existing building.
Low 5, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A few openings is typical. A significant strengthening package is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.partitions-fitout

Partitions for a commercial fit-out.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
At 1,000 m² the typical rate is about 20 days.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ceilings-fitout

Ceilings for a commercial fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
At 1,000 m² the typical rate is about 10 days.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.wet-area-fitout

Waterproof fit-out amenities.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
One amenities set. A large end-of-trip facility is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.internal-doors-fitout

Hang fit-out doors.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
A single tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-rough-fitout

Sanitary alterations rough-in for a fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Relocation within one tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sanitary-fit-fitout

Sanitary fit-off for a fit-out.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
New fixtures to the amenities.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cold-fitout

Cold-water alterations in a fit-out.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
One tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hot-fitout

Fit-out hot water.
Low 1, typical 2, high 4. A fixed range in working days. It does not grow with floor area.
One local unit.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-rough-fitout

Electrical rough-in for a fit-out.
Low 0.01, typical 0.02, high 0.03. Working days per square metre of gross floor area. At 250 m² the typical rate is about 5 days before rounding up.
At 1,000 m² the typical rate is about 20 days.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.electrical-fit-fitout

Electrical fit-off for a fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Accessories after ceilings.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lighting-fitout

Install fit-out lighting.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard office or retail package.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.mechanical-rough-fitout

Air-conditioning alterations rough-in for a fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Duct and pipe changes to one tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.mechanical-commission-fitout

Commission fit-out air-conditioning.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
Balance the altered zones.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ventilation-fitout

Fit-out exhaust ventilation.
Low 2, typical 4, high 8. A fixed range in working days. It does not grow with floor area.
Amenities exhaust. A commercial kitchen exhaust is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.sprinklers-fitout

Alter sprinklers in a fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Head relocations to one tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.detection-fitout

Alter detection in a fit-out.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One tenancy. A full panel replacement is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.comms-fitout

Fit-out structured cabling.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A standard office density.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tiling-fitout

Tile fit-out amenities.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One amenities set.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-lead-fitout

Fit-out joinery manufacture.
Low 10, typical 20, high 30. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A reception and a tea point. A large trading floor is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.joinery-install-fitout

Install fit-out joinery.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Installation after delivery.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.painting-fitout

Paint a commercial fit-out.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
At 1,000 m² the typical rate is about 10 days.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.floors-fitout

Lay fit-out floor finishes.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
Carpet tile to one tenancy.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.fire-stopping

Fire-stop service penetrations.
Low 0.01, typical 0.01, high 0.02. Working days per square metre of gross floor area. At 1000 m² the typical rate is about 10 days before rounding up.
A mid-rise or a large fit-out. A house is not in this fragment.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.gas-commercial

Commercial consumer gas.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One kitchen or one plant connection. An industrial process is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lift-lead

Passenger-lift manufacture from the start of construction.
Low 60, typical 100, high 160. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A few stops is the low end. A taller rise or a special car is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lift-install

Install passenger lifts.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
Rails, car and doors after the shaft is available. If the shaft finishes later than this lead, the dates need a manual link.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.lift-commission

Commission passenger lifts.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
Includes testing with the certifier's witness as a duration, not as a statutory clock.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.switchboard-lead-2

Manufacture an apartment main switchboard.
Low 20, typical 40, high 70. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A mid-rise residential board.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.switchboard-install-2

Install an apartment main switchboard.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
Set, connect and test.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.switchboard-lead-major

Manufacture a commercial or school main switchboard.
Low 15, typical 30, high 60. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A single building. A main board with a long factory queue is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.switchboard-install-major

Install a commercial or school main switchboard.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
Set and connect.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.utility-apply-1a

Apply for a house permanent supply.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
An urban connection with capacity available.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.utility-lead-1a

House permanent supply connection.
Low 10, typical 20, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A standard suburban connection. A new pole or pillar is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.utility-apply-major

Apply for a permanent supply to a larger building.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
Includes the initial authority application, not the civil works.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.utility-lead-major

Permanent supply connection for a larger building.
Low 20, typical 40, high 80. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
A standard urban connection is typical. A chamber substation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.services-apply-1a

Apply for house water, sewer and communications.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
Urban authorities. A rural or constrained connection is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.services-lead-1a

House services lead-in.
Low 10, typical 20, high 40. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Standard suburban mains.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.services-apply-major

Apply for services to a larger building.
Low 10, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
Section 73 or equivalent authority work can sit at the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.services-lead-major

Services lead-in for a larger building.
Low 20, typical 40, high 80. A procurement wait for manufacture and delivery, counted in working days. Installation is separate where the fragment has an install activity.
Urban mains with capacity. A main upgrade is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.concept-design-1a

Concept design for a house.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A new house or a substantial alteration. A small renovation is the low end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.concept-design-2

Concept design for an apartment building.
Low 20, typical 40, high 80. A fixed range in working days. It does not grow with floor area.
A mid-rise building. A tall or constrained urban site is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.concept-design-fitout

Concept design for a fit-out.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
One tenancy or a school refurbishment of one building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.concept-design-7b

Concept design for a warehouse.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single industrial building. A business park is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.concept-design-9b

Concept design for a school.
Low 15, typical 30, high 50. A fixed range in working days. It does not grow with floor area.
One new teaching building. A whole campus master plan is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.design-development-1a

Design development for 1a.
Low 10, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
Consultant coordination is included. Specialist reports are investigation work items, not this duration.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.design-development-2

Design development for 2.
Low 20, typical 45, high 80. A fixed range in working days. It does not grow with floor area.
Consultant coordination is included. Specialist reports are investigation work items, not this duration.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.design-development-fitout

Design development for fitout.
Low 5, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
Consultant coordination is included. Specialist reports are investigation work items, not this duration.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.design-development-7b

Design development for 7b.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
Consultant coordination is included. Specialist reports are investigation work items, not this duration.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.design-development-9b

Design development for 9b.
Low 15, typical 30, high 45. A fixed range in working days. It does not grow with floor area.
Consultant coordination is included. Specialist reports are investigation work items, not this duration.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.approval-docs-1a

Approval documentation for 1a.
Low 15, typical 25, high 40. A fixed range in working days. It does not grow with floor area.
Approval drawings for a house, including the set a certifier or a council will see.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.approval-docs-2

Approval documentation for 2.
Low 30, typical 50, high 80. A fixed range in working days. It does not grow with floor area.
Approval drawings for an apartment building. Planning reports are extra and sit in the lodgement activity.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.approval-docs-7b

Approval documentation for 7b.
Low 10, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
Approval drawings for a warehouse.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.approval-docs-9b

Approval documentation for 9b.
Low 20, typical 40, high 60. A fixed range in working days. It does not grow with floor area.
Approval drawings for a school building.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.fitout-documentation

Fit-out documentation.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
One tenancy or one school building. A staged occupation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.construction-docs-1a

Construction documentation for 1a.
Low 15, typical 30, high 50. A fixed range in working days. It does not grow with floor area.
Coordinated architectural and services documents. Shop drawings are inside the trade fragments.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.construction-docs-2

Construction documentation for 2.
Low 40, typical 70, high 110. A fixed range in working days. It does not grow with floor area.
Coordinated architectural and services documents. Shop drawings are inside the trade fragments.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.construction-docs-7b

Construction documentation for 7b.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
Coordinated architectural and services documents. Shop drawings are inside the trade fragments.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.construction-docs-9b

Construction documentation for 9b.
Low 25, typical 40, high 70. A fixed range in working days. It does not grow with floor area.
Coordinated architectural and services documents. Shop drawings are inside the trade fragments.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.measure-existing

Measure an existing building.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
One house or one floor. A whole commercial building is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.hazmat-survey

Hazardous-materials survey.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A desk and site survey. Laboratory testing that follows is inside the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.trace-services

Trace existing services.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A house or a single tenancy. A campus is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.da-lodgement

Lodge a development application or planning permit.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A complete package. Incomplete-application delay is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.cdc-prepare

Prepare a complying development certificate application.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A house or a small commercial complying development. Drawing time already spent in approval documentation is not fully excluded, so the low end avoids double counting a finished set.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.exempt-confirmation

Confirm exempt development.
Low 1, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A clear exemption is the low end. Written advice is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ssd-sears

SEARs and study scope for State significant development.
Low 15, typical 30, high 45. A fixed range in working days. It does not grow with floor area.
A straightforward declaration. A contested scope is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.ssd-eis

Prepare an environmental impact statement.
Low 40, typical 80, high 140. A fixed range in working days. It does not grow with floor area.
A building SSD. A complex industrial or extractive project is outside this high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.construction-certificate

Construction certificate review.
Low 10, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A complete house set is the low end. A large building with several information requests is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.heritage-referral

Heritage referral.
Low 10, typical 20, high 40. A fixed range in working days. It does not grow with floor area.
A local item or a conservation area is typical. A State Heritage Register approval is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-docs-1a

Tender documentation for 1a.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
A single trade or a single head contract. A staged set of packages is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-docs-2

Tender documentation for 2.
Low 15, typical 25, high 40. A fixed range in working days. It does not grow with floor area.
A single trade or a single head contract. A staged set of packages is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-docs-fitout

Tender documentation for fitout.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single trade or a single head contract. A staged set of packages is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-docs-7b

Tender documentation for 7b.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A single trade or a single head contract. A staged set of packages is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-docs-9b

Tender documentation for 9b.
Low 10, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
A single trade or a single head contract. A staged set of packages is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-period-1a

Tender period for 1a.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
A house tender, often a closed list of builders.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-period-2

Tender period for 2.
Low 15, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
An apartment tender. A two-stage tender is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-period-fitout

Tender period for fitout.
Low 10, typical 15, high 20. A fixed range in working days. It does not grow with floor area.
A fit-out tender. A short closed tender is the low end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-period-7b

Tender period for 7b.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
A warehouse tender.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-period-9b

Tender period for 9b.
Low 15, typical 20, high 30. A fixed range in working days. It does not grow with floor area.
A school tender, allowing a longer question period.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-evaluation-1a

Tender evaluation for 1a.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
A short list. A contested or multi-package evaluation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-evaluation-2

Tender evaluation for 2.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
A short list. A contested or multi-package evaluation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-evaluation-fitout

Tender evaluation for fitout.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
A short list. A contested or multi-package evaluation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-evaluation-7b

Tender evaluation for 7b.
Low 5, typical 10, high 15. A fixed range in working days. It does not grow with floor area.
A short list. A contested or multi-package evaluation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.tender-evaluation-9b

Tender evaluation for 9b.
Low 10, typical 15, high 20. A fixed range in working days. It does not grow with floor area.
A short list. A contested or multi-package evaluation is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.contract-award

Award a building contract.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A straightforward lump sum is the low end. A negotiated or financed close is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.novation

Novate consultants to the contractor.
Low 10, typical 15, high 30. A fixed range in working days. It does not grow with floor area.
A small consultant team is the low end. A large novation with deeds still in draft is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.site-establishment-1a

Site establishment for 1a.
Low 2, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A house site: fencing, shed and amenities.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.site-establishment-2

Site establishment for 2.
Low 10, typical 15, high 25. A fixed range in working days. It does not grow with floor area.
An apartment site: hoarding, crane base and amenities.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.site-establishment-fitout

Site establishment for fitout.
Low 3, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A fit-out: protection, a site office and deliveries.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.site-establishment-7b

Site establishment for 7b.
Low 3, typical 8, high 15. A fixed range in working days. It does not grow with floor area.
A warehouse site: fencing and a compound.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.site-establishment-9b

Site establishment for 9b.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A school site, including separation from students where the campus is live.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.practical-completion

Practical completion inspection.
Low 1, typical 3, high 5. A fixed range in working days. It does not grow with floor area.
One building. A campus or a staged completion is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.occupation-certificate

Occupation certificate.
Low 5, typical 10, high 20. A fixed range in working days. It does not grow with floor area.
A house is the low end. A large building with outstanding fire items is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.handover

Handover.
Low 2, typical 5, high 10. A fixed range in working days. It does not grow with floor area.
A house is the low end. A building with several user groups is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.defects-inspection

Defects inspection.
Low 1, typical 2, high 5. A fixed range in working days. It does not grow with floor area.
One inspection. A large building with several visits is the high end.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


### du.final-defects

Final defects inspection.
Low 1, typical 2, high 5. A fixed range in working days. It does not grow with floor area.
One visit at the end of the period.
Planning estimate for the stated geography from ordinary Australian practice, not a published benchmark. Accepted by the owner on 24 October 2026.


## Project-level fragments (pending scope: project)

These process activities are in `knowledge/program/fragments.yaml` with `scope: project`. Each one uses `systems: [site]` and an `applies_when` of conditions and determinants only. There is no `work_action`. Generation includes each one once for the project.


### fr.concept-design-1a

Concept design (1a) is a concept design fragment on `site`.
The first design pass: brief, options and a concept the client can approve. It sits in concept design. A small project is the low end.
Activities, in order: concept.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.concept-design-2

Concept design (2) is a concept design fragment on `site`.
The first design pass: brief, options and a concept the client can approve. It sits in concept design. A small project is the low end.
Activities, in order: concept.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.concept-design-fitout

Concept design (fitout) is a concept design fragment on `site`.
The first design pass: brief, options and a concept the client can approve. It sits in concept design. A small project is the low end.
Activities, in order: concept.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.concept-design-7b

Concept design (7b) is a concept design fragment on `site`.
The first design pass: brief, options and a concept the client can approve. It sits in concept design. A small project is the low end.
Activities, in order: concept.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.concept-design-9b

Concept design (9b) is a concept design fragment on `site`.
The first design pass: brief, options and a concept the client can approve. It sits in concept design. A small project is the low end.
Activities, in order: concept.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.design-development-1a

Design development (1a) is a concept design fragment on `site`.
Develop the approved concept into a coordinated design before approval drawings. It follows the matching concept-design fragment. Still inside the concept-design stage, because design development is not its own stage id.
Activities, in order: develop.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.design-development-2

Design development (2) is a concept design fragment on `site`.
Develop the approved concept into a coordinated design before approval drawings. It follows the matching concept-design fragment. Still inside the concept-design stage, because design development is not its own stage id.
Activities, in order: develop.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.design-development-fitout

Design development (fitout) is a concept design fragment on `site`.
Develop the approved concept into a coordinated design before approval drawings. It follows the matching concept-design fragment. Still inside the concept-design stage, because design development is not its own stage id.
Activities, in order: develop.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.design-development-7b

Design development (7b) is a concept design fragment on `site`.
Develop the approved concept into a coordinated design before approval drawings. It follows the matching concept-design fragment. Still inside the concept-design stage, because design development is not its own stage id.
Activities, in order: develop.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.design-development-9b

Design development (9b) is a concept design fragment on `site`.
Develop the approved concept into a coordinated design before approval drawings. It follows the matching concept-design fragment. Still inside the concept-design stage, because design development is not its own stage id.
Activities, in order: develop.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.approval-docs-1a

Approval documentation (1a) is a approval documentation fragment on `site`.
The drawing set for the consent or the certifier. It is the approval-documentation stage. Fit-out drawings are fr.fitout-documentation, because the fit-out skeleton has no separate approval-documentation stage.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.approval-docs-2

Approval documentation (2) is a approval documentation fragment on `site`.
The drawing set for the consent or the certifier. It is the approval-documentation stage. Fit-out drawings are fr.fitout-documentation, because the fit-out skeleton has no separate approval-documentation stage.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.approval-docs-7b

Approval documentation (7b) is a approval documentation fragment on `site`.
The drawing set for the consent or the certifier. It is the approval-documentation stage. Fit-out drawings are fr.fitout-documentation, because the fit-out skeleton has no separate approval-documentation stage.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.approval-docs-9b

Approval documentation (9b) is a approval documentation fragment on `site`.
The drawing set for the consent or the certifier. It is the approval-documentation stage. Fit-out drawings are fr.fitout-documentation, because the fit-out skeleton has no separate approval-documentation stage.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.fitout-documentation

Fit-out documentation is a construction documentation fragment on `site`.
The drawing set that is priced and given to the certifier for a fit-out. One stage, because the fit-out skeleton does not split approval drawings from construction drawings.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.construction-docs-1a

Construction documentation (1a) is a construction documentation fragment on `site`.
Construction documents after the approval path, for a traditional or design-and-construct job. On a design-and-construct skeleton this stage is the builder's documents. Fit-out uses fr.fitout-documentation instead.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.construction-docs-2

Construction documentation (2) is a construction documentation fragment on `site`.
Construction documents after the approval path, for a traditional or design-and-construct job. On a design-and-construct skeleton this stage is the builder's documents. Fit-out uses fr.fitout-documentation instead.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.construction-docs-7b

Construction documentation (7b) is a construction documentation fragment on `site`.
Construction documents after the approval path, for a traditional or design-and-construct job. On a design-and-construct skeleton this stage is the builder's documents. Fit-out uses fr.fitout-documentation instead.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.construction-docs-9b

Construction documentation (9b) is a construction documentation fragment on `site`.
Construction documents after the approval path, for a traditional or design-and-construct job. On a design-and-construct skeleton this stage is the builder's documents. Fit-out uses fr.fitout-documentation instead.
Activities, in order: draw.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.existing-conditions

Existing-conditions investigation is a investigation fragment on `site`.
Measure the existing building, survey hazardous materials, and trace services. Used for a refurbishment or an extension. Geotechnical investigation is an investigate work item and is added by code without a fragment.
Activities, in order: measure, hazmat, services.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.da-lodgement

Development-application lodgement is a approvals fragment on `site`.
Assemble and lodge the development application or the planning permit. The drawing time is the approval-documentation fragment. This is the lodgement itself.
Activities, in order: lodge.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.da-exhibition

Development-application exhibition is a approvals fragment on `site`.
Public exhibition of a development application that is not State significant. The duration is the statutory minimum, verified false. A planning permit uses the same allowance as a placeholder until a Victorian period is added.
Activities, in order: exhibit.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.da-assessment

Development-application assessment is a approvals fragment on `site`.
Council or panel assessment after lodgement. The duration is a program allowance, not the deemed-refusal clock. The clock itself is du.da-assessment-period and is not the activity duration, because information requests stop it.
Activities, in order: assess.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.cdc-application

Complying development certificate is a approvals fragment on `site`.
Prepare, lodge and have a certifier determine a complying development certificate. Preparation is an estimate. Determination uses the statutory section 133 clock, verified false.
Activities, in order: prepare, determine.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.exempt-confirmation

Exempt-development confirmation is a approvals fragment on `site`.
A short confirmation that the work is exempt, or a certifier's written advice where the team wants one. Exempt work has no development application.
Activities, in order: confirm.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ssd-eis

State significant environmental impact statement is a approval documentation fragment on `site`.
SEARs, specialist studies and the environmental impact statement for State significant development. This is the approval-documentation stage. It is an estimate of consultant time, not a statutory period.
Activities, in order: sears, eis.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.ssd-assessment

State significant exhibition and assessment is a approvals fragment on `site`.
Exhibition uses the statutory Schedule 1 clause 9 minimum, verified false. Assessment after exhibition uses a program allowance. The 90-day deemed-refusal clock is the low end of that allowance and is also recorded as du.da-assessment-period.
Activities, in order: exhibit, assess.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.construction-certificate

Construction certificate is a approvals fragment on `site`.
Lodge and obtain the construction certificate after consent. A complying development certificate already authorises construction, so this fragment does not apply on a CDC path. It is listed after the consent paths so library order places it behind whichever assessment applies. The duration is a certifier's review of a complete set, not a legislated clock.
Activities, in order: certify.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.heritage-referral

Heritage referral is a approvals fragment on `site`.
Heritage referral or permit alongside the planning path, for a conservation area, a local item or the State Heritage Register. Anchored at the start of approvals so it runs with the application rather than after the construction certificate. The status value none does not select it.
Activities, in order: refer.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-docs-1a

Tender documentation (1a) is a procurement fragment on `site`.
Assemble the tender package from the construction documents. This is not a second drawing stage. It sits in procurement.
Activities, in order: pack.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-docs-2

Tender documentation (2) is a procurement fragment on `site`.
Assemble the tender package from the construction documents. This is not a second drawing stage. It sits in procurement.
Activities, in order: pack.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-docs-fitout

Tender documentation (fitout) is a procurement fragment on `site`.
Assemble the tender package from the construction documents. This is not a second drawing stage. It sits in procurement.
Activities, in order: pack.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-docs-7b

Tender documentation (7b) is a procurement fragment on `site`.
Assemble the tender package from the construction documents. This is not a second drawing stage. It sits in procurement.
Activities, in order: pack.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-docs-9b

Tender documentation (9b) is a procurement fragment on `site`.
Assemble the tender package from the construction documents. This is not a second drawing stage. It sits in procurement.
Activities, in order: pack.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-period-1a

Tender period (1a) is a procurement fragment on `site`.
The time tenderers have to price. It follows the matching tender-documentation fragment.
Activities, in order: price.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-period-2

Tender period (2) is a procurement fragment on `site`.
The time tenderers have to price. It follows the matching tender-documentation fragment.
Activities, in order: price.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-period-fitout

Tender period (fitout) is a procurement fragment on `site`.
The time tenderers have to price. It follows the matching tender-documentation fragment.
Activities, in order: price.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-period-7b

Tender period (7b) is a procurement fragment on `site`.
The time tenderers have to price. It follows the matching tender-documentation fragment.
Activities, in order: price.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-period-9b

Tender period (9b) is a procurement fragment on `site`.
The time tenderers have to price. It follows the matching tender-documentation fragment.
Activities, in order: price.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-evaluation-1a

Tender evaluation (1a) is a procurement fragment on `site`.
Compare tenders, clarify and recommend. It follows the matching tender period.
Activities, in order: evaluate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-evaluation-2

Tender evaluation (2) is a procurement fragment on `site`.
Compare tenders, clarify and recommend. It follows the matching tender period.
Activities, in order: evaluate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-evaluation-fitout

Tender evaluation (fitout) is a procurement fragment on `site`.
Compare tenders, clarify and recommend. It follows the matching tender period.
Activities, in order: evaluate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-evaluation-7b

Tender evaluation (7b) is a procurement fragment on `site`.
Compare tenders, clarify and recommend. It follows the matching tender period.
Activities, in order: evaluate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.tender-evaluation-9b

Tender evaluation (9b) is a procurement fragment on `site`.
Compare tenders, clarify and recommend. It follows the matching tender period.
Activities, in order: evaluate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.contract-award

Contract award is a procurement fragment on `site`.
Negotiate and execute the building contract after evaluation. One fragment for every class. It is listed after the evaluation fragments so library order places it behind whichever evaluation applies.
Activities, in order: award.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.novation

Novation is a procurement fragment on `site`.
Novate the design consultants to the contractor after award. Used for design and construct and for early contractor involvement. A managing contractor's trade packages are not this single novation.
Activities, in order: novate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.site-establishment-1a

Site establishment (1a) is a construction fragment on `site`.
Establish the site at the start of construction. Anchored at stage start. Hoardings for an occupied fit-out are also the occupied-protection fragment when that work item exists.
Activities, in order: establish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.site-establishment-2

Site establishment (2) is a construction fragment on `site`.
Establish the site at the start of construction. Anchored at stage start. Hoardings for an occupied fit-out are also the occupied-protection fragment when that work item exists.
Activities, in order: establish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.site-establishment-fitout

Site establishment (fitout) is a construction fragment on `site`.
Establish the site at the start of construction. Anchored at stage start. Hoardings for an occupied fit-out are also the occupied-protection fragment when that work item exists.
Activities, in order: establish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.site-establishment-7b

Site establishment (7b) is a construction fragment on `site`.
Establish the site at the start of construction. Anchored at stage start. Hoardings for an occupied fit-out are also the occupied-protection fragment when that work item exists.
Activities, in order: establish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.site-establishment-9b

Site establishment (9b) is a construction fragment on `site`.
Establish the site at the start of construction. Anchored at stage start. Hoardings for an occupied fit-out are also the occupied-protection fragment when that work item exists.
Activities, in order: establish.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.practical-completion

Practical completion inspection is a completion fragment on `site`.
The practical-completion inspection and the list of remaining items. It is the completion stage. Trade commissioning stays on the trade fragments.
Activities, in order: inspect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.occupation-certificate

Occupation certificate is a completion fragment on `site`.
The occupation certificate after practical completion. The duration is the certifier's review of a complete package. It is not a legislated clock.
Activities, in order: certificate.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.handover

Handover is a completion fragment on `site`.
Manuals, keys, training and the handover meeting, after the occupation certificate.
Activities, in order: hand.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.defects-inspection

Defects inspection is a defects fragment on `site`.
The inspection that opens the defects liability period. Making good the items is not given a duration, because the list is unknown until the inspection.
Activities, in order: inspect.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


### fr.defects-liability-end

End of defects liability is a defects fragment on `site`.
The wait until the end of the defects liability period, then the final inspection. The long duration is the period itself, in calendar days. It is a contract period, not the Home Building Act warranty.
Activities, in order: wait, final.
Each duration it names is a draft planning estimate. The fragment stays draft until the owner reviews it.


These rows are in `knowledge/program/fragments.yaml` with `scope: project`. Generation includes each one once when its conditions hold. The activity lists stay draft. Their durations are the reviewed rows in `durations.yaml`.

## Owner defaults (25 October 2026)

The owner directed on 25 October 2026 that opening Program shows a complete program at once: every activity has a duration, so the Gantt chart draws and the user drags a bar that looks wrong instead of typing durations and links from nothing. These rows are defaults for a quick first program. They come from ordinary Australian residential and small-commercial practice, not a published benchmark. Rows marked with an assumption show a marker in the grid.

The same direction folds consultant packages into the stage's own lines. A design stage shows three or four lines (confirm the brief, prepare the design, coordinate consultants, client review or sign-off), and every consultant, one or five, is taken to follow them. Packages still map to program stages for Procurement and the Cost Plan.

### Assumed gross floor area

250 m² while the Profile has no gross floor area. The owner's figure: about the average size of a new Australian house (published analyses of ABS building approvals put new houses at roughly 235 to 250 m² in recent years). Every duration that scales with floor area uses it until the Profile states a value; then the review offers the recomputed durations. Rows computed from it show "Assumes 250 m² gross floor area until the Profile states it."

### du.brief-confirm-1a

Confirm the brief for a house: meet the client, agree rooms, budget and priorities, and record them.
Low 3, typical 5, high 10 working days. A fixed range.

### du.brief-confirm

Confirm the brief for an apartment, commercial or public building, where more stakeholders sign it off.
Low 5, typical 10, high 20 working days. A fixed range.

### du.client-review-1a

Client review of a house concept: present, take comments, agree the direction before design development.
Low 5, typical 10, high 15 working days. A fixed range.

### du.client-review

Client review of a larger building's concept, often with a board, tenant or funder.
Low 10, typical 15, high 25 working days. A fixed range.

### du.client-signoff-1a

Client sign-off of a house's approval set before lodgement, or its construction set before tender.
Low 3, typical 5, high 10 working days. A fixed range.

### du.client-signoff

Client sign-off of a larger building's approval or construction set.
Low 5, typical 10, high 15 working days. A fixed range.

### du.investigate-geotech

Geotechnical investigation: book the rig, drill boreholes, test and report. The owner's figure is four weeks.
Low 10, typical 20, high 30 working days. Rock, a congested site or laboratory testing is the high end.

### du.investigate-survey

Detail and boundary survey by a registered surveyor, from booking to drawing.
Low 5, typical 10, high 15 working days.

### du.investigate-contamination

Preliminary (stage 1) contamination investigation: desktop history, site walkover and report. A detailed (stage 2) investigation with sampling takes longer and is entered by the user.
Low 10, typical 15, high 30 working days.

### du.investigate-acid-sulfate

Acid sulfate soils assessment, sampling and report, where the planning map shows a class.
Low 10, typical 15, high 25 working days.

### du.investigate-trees

Arborist's assessment (arboricultural impact assessment) of trees on or near the site.
Low 5, typical 10, high 15 working days.

### du.investigate-flooding

Flood advice from council or a flood engineer; a full flood study is the high end.
Low 10, typical 20, high 40 working days.

### du.investigate-services

Services search and locating: Before You Dig enquiry, authority plans and on-site locating.
Low 5, typical 10, high 15 working days.

### du.investigate-default

Any other investigation the Profile asks for, such as checking an existing roof or structure.
Low 5, typical 10, high 20 working days. Two weeks typical.

### du.activity-default

An activity no library row covers yet. Five working days, so the bar is visible and easy to drag.
Low 3, typical 5, high 10 working days.

### fr.investigate-geotech

An investigate work item on ground conditions becomes a geotechnical investigation under Investigation, with du.investigate-geotech. It starts at the stage start, alongside other investigations.

### fr.investigate-survey

An investigate work item on survey and boundaries, with du.investigate-survey.

### fr.investigate-contamination

An investigate work item on contamination, with du.investigate-contamination.

### fr.investigate-acid-sulfate

An investigate work item on acid sulfate soils, with du.investigate-acid-sulfate.

### fr.investigate-trees

An investigate work item on trees, with du.investigate-trees.

### fr.investigate-flooding

An investigate work item on flooding, with du.investigate-flooding.

### fr.investigate-services

An investigate work item on services connections, with du.investigate-services.
