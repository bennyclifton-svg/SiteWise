# Trade research: structure and envelope

Date: 10 November 2026. Slice T3 of `docs/plans/2026-11-10-trades-catalogue.md`.
Status: draft for owner review. Seed guidance only.

Every standard and NCC part named below is written as the seed names it
and is **`verified: false`**: none was read from the instrument. Every
licence statement is **`verified: false`**: none was checked against NSW
Fair Trading. No rates, prices, quantities, durations or products.

RFT files written (all `status: draft`, shape of
`knowledge/rfp/trade-structural-steel.yaml`):

| Trade id | File |
|---|---|
| trade.carpentry_framing | `knowledge/rfp/trade-carpentry-framing.yaml` |
| supplier.frames_trusses | `knowledge/rfp/supplier-frames-trusses.yaml` |
| trade.masonry | `knowledge/rfp/trade-masonry.yaml` |
| trade.precast | `knowledge/rfp/trade-precast.yaml` |
| trade.scaffolding | `knowledge/rfp/trade-scaffolding.yaml` |
| trade.roofing | `knowledge/rfp/trade-roofing.yaml` |
| trade.roof_plumbing | `knowledge/rfp/trade-roof-plumbing.yaml` |
| trade.cladding | `knowledge/rfp/trade-cladding.yaml` |
| trade.rendering | `knowledge/rfp/trade-rendering.yaml` |
| trade.insulation | `knowledge/rfp/trade-insulation.yaml` |
| trade.garage_doors | `knowledge/rfp/trade-garage-doors.yaml` |
| trade.curtain_wall | `knowledge/rfp/trade-curtain-wall.yaml` |
| supplier.windows_glazing | `knowledge/rfp/supplier-windows-glazing.yaml` |

Seed paths below are under `data/reference/clerk/data/seed/`.

## The three boundaries the owner asked about

**Framing vs frames and trusses (install vs supply).** The supplier
designs, lays out, makes and delivers the prefabricated wall frames and
nailplated trusses, and gives the design certificate and the bracing,
tie-down and girder details. The framing carpenter erects, braces and ties
down, and presents the frame for the certifier's frame inspection. The
supply RFT has an "Installation (excluded from a supply-only tender)"
stage; when it is not priced, a position names the framing carpenter. The
carpentry RFT names the supplier in a position for frames let as a
separate supply. Seed: `procurement-quoting-guide.md` §4 lists "Truss
design as PC or trade" as a recurring framing scope trap.

**Roofer vs roof plumber.** Seed sequence (`trade-interfaces-coordination-guide.md`
Interface 1): frame certificate, trusses, sarking before battens, roof
plumbing rough-in (downpipe brackets, vent and flue penetrations) before
roofing, then covering, ridges, hips, valleys, then flashings and
penetrations sealed. Split used:

- Roofer: sarking, battens, roof blanket under metal sheet, covering,
  ridges, hips, valleys and cappings, flashings at walls, abutments and
  chimneys (the `envelope.roof-coverings` system includes valleys and
  roof flashings).
- Roof plumber: downpipe brackets and vent and flue penetrations with
  their flashings (rough-in, before covering), eaves and box gutters,
  sumps, rainwater heads, downpipes to the stormwater points, overflows,
  gutter guard or ember mesh (`hydraulic.roof-drainage`).
- Fascia and eaves linings: framing carpenter (seed frame stage lists
  "fascia and eaves"; §4 names the fascia boundary as a scope trap).

**Cladding, render and insulation vs wall wrap and fire-rated external
walls.** The cladding contractor installs the wall wrap, cavity battens and
flashings behind the cladding, then the cladding. The roofer lays roof
sarking. The insulation contractor installs bulk insulation in the frame
and ceilings and does not install sarking or wrap. On the fire side, the
seeds say that external walls of Type A and B buildings generally must be
non-combustible, that some cladding may qualify only through full-scale
fire testing, that the combustibility of cavity insulation and wrap
matters as much as the cladding, and that a product marketed as
non-combustible must be checked against its test evidence. The RFTs say
this in words only: the cladding, render and facade contractors give test
evidence that their materials meet the documents' fire requirements
before ordering. No clause or test number appears in the RFT text.
Seeds: `ncc-reference-guide.md` "Facade and External Wall Fire Safety",
"Cladding Combustibility Requirements", "Spandrel and Slab Edge Fire
Protection"; `multi-residential-apartments-guide.md` "Facade Selection
for Multi-Residential Buildings". The seeds name NCC 2022 Vol 1 Part C1.9,
Part C2.6, C2.7, Specification C2.6, Verification Method CV3, AS 5113,
AS 1530.1, AS 1530.3, AS 4284 and AS/NZS 4200.1. All `verified: false`.
The seeds' clause numbering is inconsistent (C1.9 vs C2.7 for the same
rule), which is another reason none of it goes in the RFT.

---

## trade.carpentry_framing: Carpentry (framing)

**Summary.** Erects, braces and ties down floor, wall and roof framing,
and installs fascia, eaves and (where given) windows with their flashings.
House: AS 1684 span-table timber framing, trusses, fascia, often windows.
Commercial: less common; engineered timber (LVL, glulam, CLT) to the
engineer's design, and window installation usually goes with the window
or facade package.

**Interfaces.** Takes from trade.concreting (slab level and set out, cast-in
hold-downs), supplier.frames_trusses (frames, trusses, layouts),
supplier.windows_glazing (windows to install). Leaves for trade.roofing
(sarking and battens), trade.roof_plumbing (fascia ready for gutters),
trade.masonry and trade.cladding (frame and window flashings before
brickwork or cladding), trade.insulation, trade.plasterboard.
trade.termite_management works at the slab before the frame.

**Hold points.** Frame inspection by the certifier (or accredited
inspector) before any lining, ceiling or external cladding; the engineer
signs off non-standard elements (steel beams, LVL, portal frames);
tie-down and connections confirmed in the sign-off. Notching and drilling
checked before lining.

**Certificates it signs.** None statutory found in the seeds. It gives
the bracing and tie-down records for the frame sign-off. Standards named:
AS 1684 series, AS 4055, AS 4440 (truss installation), AS 1720.
`verified: false`.

**Licence (NSW).** A NSW contractor licence for carpentry (a specialist
trade class) or general building work, as the role-builder seed describes
subspecialty classes for specialist trades. `verified: false`.

**Main Works line.** Row `structure`, existing line `frame` ("Structural
frame, walls and floors") is shared with concrete and steel. Propose
`{key: timber_frame, text: "Timber floor, wall and roof framing, bracing and tie-down", separate: trade.carpentry_framing}` under `structure`. The roof
structure part of envelope line `roof` moves here (see roofing).

**Sources.** `residential-construction-guide.md` "Structural Framing
Systems", "Construction Sequence and Programming for Residential
Projects"; `structural-residential.md` "Frame inspection gate",
"Notching, drilling, and modification rules", "Connection and tie-down
requirements"; `trade-interfaces-coordination-guide.md` Interfaces 1 and 4;
`procurement-quoting-guide.md` §4.

**Open questions.** O-SE1: does the framing carpenter install the windows
and their flashings by default on a house, or the window supplier? (RFT
gives it "where the documents give it to you".) O-SE2: is floor framing
and decking in this package or a separate deck trade?

## supplier.frames_trusses: Frames and trusses (supply)

**Summary.** Designs, lays out, makes and delivers prefabricated wall
frames and nailplated roof trusses. House: standard; commercial: used on
low-rise timber buildings; larger work uses engineered timber.

**Interfaces.** Takes from the architect and structural engineer (documents,
wind classification) and trade.concreting (slab set-out). Leaves for
trade.carpentry_framing (erection, bracing, tie-down) unless the install
stage is priced, and trade.roofing.

**Hold points.** Layouts for the design team's review before manufacture;
check of slab set-out before manufacture (practice, not seed-numbered);
frame inspection by the certifier after erection (the supplier's design
certificate is part of the evidence).

**Certificates it signs.** The truss and frame design certificate (seed:
"Truss design as PC or trade"; frame inspection evidence). Standards:
AS 1684, AS 4440, AS 4055. `verified: false`.

**Licence (NSW).** Supply only needs no contractor licence as far as the
seeds go; if it installs, the installer needs the carpentry or building
licence. `verified: false`.

**Main Works line.** Row `structure`. Propose
`{key: frames_trusses_supply, text: "Supply of prefabricated wall frames and roof trusses", separate: supplier.frames_trusses}`. With only the supply let
separately, `timber_frame` stays in Main Works ("supply by separate
contract; install included", as the plan's edge case says).

**Sources.** `residential-construction-guide.md` "Structural Framing
Systems"; `procurement-quoting-guide.md` §4; `structural-residential.md`
"Connection and tie-down requirements", "Frame inspection gate".

**Open questions.** O-SE3: should the supply RFT default to the install
stage priced as an option, rather than included-then-excluded by the PM?
O-SE4: steel framing (cold-formed) supply is not in this list; is it a
variant of this supply or of trade.structural_steel?

## trade.masonry: Bricklaying and blocklaying

**Summary.** Lays loadbearing and veneer brick and block walls with ties,
cavity flashings, damp-proof courses, weepholes and movement joints, plus
reinforced and core-filled blockwork. House: brick veneer, the most common
southern Australian wall. Commercial: blockwork (core-filled, reinforced),
lift and stair cores, fire-rated walls.

**Interfaces.** Takes from trade.carpentry_framing / head contractor
(windows installed and flashed before brickwork reaches window height),
trade.concreting (slab edge, rebate, starter bars for blockwork),
trade.scaffolding. Leaves for trade.rendering (render over masonry),
trade.roofing (abutment flashings at the brick line).

**Hold points.** Builder hold point: window installation and flashing
signed off before brickwork passes window height (no formal certifier
hold point). Engineer: reinforcement and core-fill inspection for
reinforced blockwork (from AS 3700 practice; not seed-detailed). Clear
weepholes and cavities checked at practical completion.

**Certificates it signs.** None found in the seeds. Standards: AS 3700,
AS 4773.1, AS 2699 (ties). `verified: false`.

**Licence (NSW).** A NSW contractor licence for bricklaying (specialist
trade) or general building work. `verified: false`.

**Main Works line.** Row `envelope`, existing line `walls` ("External
walls and cladding") is shared with cladding and render. Propose
`{key: masonry, text: "Brick and block walls, veneer, ties, cavity flashings and damp-proof courses", separate: trade.masonry}` under `envelope`
(loadbearing masonry also sits under `structure`; one line avoids
splitting a bricklayer's price).

**Sources.** `residential-construction-guide.md` "Building Envelope and
Weatherproofing"; `as-standards-reference.md` "AS 3700 — Masonry
Structures"; `trade-interfaces-coordination-guide.md` Interface 4;
`procurement-quoting-guide.md` §4 (owner-supplied bricks);
`new-dwelling-guide.md` "Sequencing risks specific to new-dwelling" (brick
supply lead time).

**Open questions.** O-SE5: brick supply: by the bricklayer, the builder,
or the client as a prime cost item? (RFT position covers client supply
only.) O-SE6: head flashings over openings in brick veneer: bricklayer or
window installer? The seed only says flashings come before brickwork.

## trade.precast: Precast concrete

**Summary.** Details, casts, delivers and erects structural precast and
tilt-up elements with connections, grouting and temporary bracing.
Commercial and mid-rise residential; rare on houses.

**Interfaces.** Takes from consultant.structural (design, shop drawing
review), trade.concreting (footings, slabs, cast-in plates), head
contractor (crane access, hardstand). Leaves for trade.passive_fire (joint
fire seals where rated; open), trade.curtain_wall / trade.cladding where
precast is the facade.

**Hold points.** Engineer reviews shop drawings, lifting and bracing
design before casting; engineer's inspection hold points at erection and
connections; temporary bracing removed only when the engineer says.
Seeds name robustness and progressive collapse in precast connections.

**Certificates it signs.** Concrete test records; erection and connection
sign-off (seed-general). Standards: AS 3600, AS 1012. `verified: false`.

**Licence (NSW).** Commercial work: no Home Building Act contractor
licence for non-residential work as far as the seeds go; high-risk work
licences for crane, dogging and rigging under WHS. `verified: false`.

**Main Works line.** Row `structure`. Propose
`{key: precast, text: "Precast and tilt-up concrete elements, their connections and temporary bracing", separate: trade.precast}`.

**Sources.** `structural-commercial-industrial-guide.md` "Documentation
and tender", "Construction and completion"; `ncc-reference-guide.md`
"Performance Requirements and Importance Levels";
`multi-residential-apartments-guide.md` "Facade Selection".

**Open questions.** O-SE7: architectural precast cladding carries no load
and is an envelope system; does it go to this trade or to cladding?
O-SE8: who seals panel joints that need a fire rating?

## trade.scaffolding: Scaffolding

**Summary.** Designs, erects, alters, inspects and dismantles scaffold,
edge protection and access. House: perimeter scaffold for upper storeys
and roof edge. Commercial: full-height, sheeted, engineered scaffold and
catch protection.

**Interfaces.** Takes from head contractor (ground ready, footpath and road
permits). Serves trade.masonry, trade.rendering, trade.cladding,
trade.roofing, trade.roof_plumbing, trade.painting. Leaves tie-point
patching to the head contractor (position).

**Hold points.** Inspection and tag before first use, after any
alteration or severe weather, and at regular intervals during use (WHS,
not the certifier). Engineered design above the height or configuration
the seed names.

**Certificates it signs.** Handover certificate and tag per section.
Standards named: AS/NZS 1576.1, AS/NZS 1576.3, AS 4576, AS 1657, WHS
Regulation 2017. `verified: false`.

**Licence (NSW).** A high-risk work licence (scaffolding basic,
intermediate or advanced) issued under WHS law by SafeWork NSW, needed
above a height the seed names; not a Fair Trading contractor licence.
`verified: false`.

**Main Works line.** Row `preliminaries`, existing line `temporary`
("Temporary services, access, scaffolding and edge protection"). Propose
splitting out `{key: scaffold, text: "Scaffolding and edge protection", separate: trade.scaffolding}`, leaving `temporary` as "Temporary services
and access".

**Sources.** `as-standards-reference.md` "AS/NZS 1576 Series —
Scaffolding"; `commercial-construction-guide.md` "Workplace Health and
Safety in Commercial Construction"; `cost-management-principles.md`
(scaffold in preliminaries).

**Open questions.** O-SE9: hire period basis: the RFT prices hire "for the
period the program holds each section"; should hire beyond that be a
weekly rate in the returnables? (No rate is set here.)

## trade.roofing: Roofing

**Summary.** Sarking, battens, tile or metal roof covering, ridges, hips,
valleys and roof flashings at walls, abutments and chimneys. House: tile
or metal pitched roofs. Commercial: metal deck roofs, and membrane roofs
belong to trade.waterproofing (`envelope.external-waterproofing`).

**Interfaces.** Takes from trade.carpentry_framing (certified frame,
fascia), trade.roof_plumbing (rough-in penetrations before covering),
trade.scaffolding. Leaves for trade.roof_plumbing (gutters, downpipes),
trade.insulation (ceiling batts after the roof is weathertight).

**Hold points.** Frame inspection certificate sighted before any roof
cladding (certifier; "no exceptions"). Builder confirms the roof
weathertight before ceiling insulation.

**Certificates it signs.** Manufacturer's warranty; no statutory
certificate in the seeds. BAL-rated roofs: non-combustible covering,
sarking and gutter mesh (AS 3959). Standards: AS 4055, AS/NZS 4200.1,
AS 3959. `verified: false`.

**Licence (NSW).** A NSW contractor licence for roof tiling or roofing
(specialist trade); metal roofing and its drainage may need a plumbing
licence (roof plumbing). `verified: false`.

**Main Works line.** Row `envelope`, existing line `roof` ("Roof structure,
roofing, gutters and downpipes") holds three trades. Propose replacing it
with `{key: roof_covering, text: "Sarking, battens, roof covering and roof flashings", separate: trade.roofing}`, and moving roof structure to
`structure.timber_frame`.

**Sources.** `trade-interfaces-coordination-guide.md` Interface 1;
`procurement-quoting-guide.md` §4 (sarking and insulation overlap, fascia
boundary); `residential-construction-guide.md` "Construction Sequence",
"Building Envelope and Weatherproofing", "Bushfire Construction
Requirements".

**Open questions.** O-SE10: valley gutters: roofer (as the system file
says) or roof plumber (common on tile roofs)? O-SE11: on a metal roof one
firm usually lays sheet and gutters; is the `roof` bundle the default, or
should the metal-roof RFT give the roofer gutters too?

## trade.roof_plumbing: Roof plumbing

**Summary.** Downpipe brackets and vent and flue penetrations (rough-in),
eaves and box gutters, sumps, rainwater heads, downpipes, overflows and
gutter guards, to the point drainage leaves the building. House: eaves
gutters and downpipes. Commercial: box gutters, sumps, overflows,
siphonic or charged systems to the hydraulic engineer's design.

**Interfaces.** Takes from trade.carpentry_framing (fascia),
trade.roofing (covering, wall flashings). Leaves for
trade.stormwater_drainage (from the downpipe connection points),
trade.hydraulic_plumbing (rainwater tanks and connections).

**Hold points.** Rough-in before the roof is covered (builder). Plumbing
inspection by the plumbing regulator for stormwater work, and a water test
of gutters and downpipes (seed-general for plumbing; not specific to roof
drainage).

**Certificates it signs.** Plumbing compliance certificate for the roof
drainage. Standards: AS/NZS 3500.3. `verified: false`.

**Licence (NSW).** A NSW plumbing licence covering stormwater or roof
plumbing work (seed: all plumbing work in NSW needs a licensed plumber).
`verified: false`.

**Main Works line.** Two lines carry roof water today: envelope `roof`
and hydraulic `roof_drainage` ("Roof water drainage to the stormwater
system"). Propose `{key: roof_drainage, text: "Gutters, box gutters, downpipes, overflows and penetration flashings", separate: trade.roof_plumbing}` under `envelope`, and either removing hydraulic
`roof_drainage` or making it "Below-ground roof water connection" so the
roof plumbing is not priced twice.

**Sources.** `hydraulic-services-guide.md` "Stormwater and roof
drainage"; `as-standards-reference.md` "5. AS 3500 — Plumbing and
drainage"; `trade-interfaces-coordination-guide.md` Interface 1;
`residential-construction-guide.md` "Bushfire Construction Requirements"
(gutter mesh).

**Open questions.** O-SE12: does the roof plumber own the vent and flue
penetration flashings (as written) or only provide the boots for the
roofer to fix? O-SE13: which Main Works row holds roof drainage after
T2, envelope or hydraulic?

## trade.cladding: Cladding

**Summary.** Wall wrap, cavity battens, flashings at openings and junctions,
lightweight cladding (fibre cement, weatherboard, metal, composite) and
its joints and trims. House: lightweight cladding on frame. Commercial:
rainscreens and panel systems on Type A and B buildings, with the external
wall fire rules.

**Interfaces.** Takes from trade.carpentry_framing / head contractor
(certified frame, windows installed and flashed), trade.insulation (wall
insulation in the frame before wrap is closed, where sequenced that way).
Leaves for trade.painting (site painting), trade.rendering (where render
goes over sheet).

**Hold points.** Builder inspection of wrap and flashings before cladding
covers them; frame certificate before covering. Facade water testing
where the documents ask (commercial). Product test evidence for
combustibility before ordering.

**Certificates it signs.** Product test evidence and system warranty; on
Type A and B buildings the fire test evidence the certifier needs.
Standards: AS/NZS 4200.1, AS 5113, AS 1530.1, AS 4284, NCC Part C1.9 and
C2.7 (as the seed numbers them). `verified: false`.

**Licence (NSW).** A NSW contractor licence for general building work or a
cladding-related specialist class. `verified: false`.

**Main Works line.** Row `envelope`, existing line `walls`. Propose
`{key: cladding, text: "Wall wrap, cavity battens and lightweight cladding", separate: trade.cladding}`; `walls` becomes the remainder.

**Sources.** `residential-construction-guide.md` "Building Envelope and
Weatherproofing"; `ncc-reference-guide.md` "Facade and External Wall Fire
Safety", "Cladding Combustibility Requirements";
`commercial-construction-guide.md` "Facade and Curtain Wall Systems";
`multi-residential-apartments-guide.md` "Weatherproofing and Facade
Defects".

**Open questions.** O-SE14: wall wrap on brick veneer (behind the frame)
is not cladding; is it the framing carpenter's or the builder's?
O-SE15: should the RFT carry a returnable naming each external-wall
material and its test report?

## trade.rendering: Rendering

**Summary.** Substrate preparation, render coats, rendered insulation panel
systems where documented, texture or finish coats, control joints and
sealant joints at frames. House: render to masonry or fibre cement.
Commercial: render and EIFS on multi-storey, where the seeds stress that
insulation cores can be combustible.

**Interfaces.** Takes from trade.masonry or trade.cladding (substrate),
head contractor (window flashings in place), trade.scaffolding. Leaves for
trade.painting (paint over render where documented).

**Hold points.** Builder hold point: window flashings complete before
render base coat reaches window level. Substrate check before rendering.

**Certificates it signs.** System warranty; fire test evidence for
rendered insulation systems on Type A and B buildings. Standards named in
the seeds: AS 5113, AS 1530.1. `verified: false`.

**Licence (NSW).** A NSW contractor licence for rendering or plastering
(specialist trade). `verified: false`.

**Main Works line.** Row `envelope`. Propose `{key: render, text: "Render and texture coat systems", separate: trade.rendering}`.

**Sources.** `residential-construction-guide.md` "Building Envelope and
Weatherproofing"; `trade-interfaces-coordination-guide.md` Interface 4;
`multi-residential-apartments-guide.md` "Facade Selection";
`procurement-quoting-guide.md` §4 (render system often unclear).

**Open questions.** O-SE16: EPS rendered panel systems: renderer or
cladding contractor?

## trade.insulation: Insulation

**Summary.** Bulk and reflective insulation in walls, ceilings, roofs and
floors to the energy commitments, clearances to downlights and flues, and
draught sealing where given. House: BASIX or NatHERS commitments.
Commercial: Section J total R-values, often installed by the plasterboard,
roofing or facade trades instead.

**Interfaces.** Takes from trade.roofing (roof weathertight; roof blanket
and sarking are the roofer's), trade.cladding (wall wrap),
trade.electrical (downlight positions, IC-rated fittings). Leaves for
trade.plasterboard (linings).

**Hold points.** Builder confirms roof weathertight before ceiling batts.
Coverage check before lining (photographs). BASIX or energy assessor's
final check of installed commitments.

**Certificates it signs.** Insulation installation certificate and the
product evidence the energy assessor needs. Standards: AS/NZS 4859.1,
NCC Section J and Part 13 (as the seed names them). `verified: false`.

**Licence (NSW).** A NSW contractor licence may apply to insulation
installation as residential building work; not confirmed in the seeds.
`verified: false`.

**Main Works line.** Row `envelope`, existing line `weatherproofing`
("Insulation, sarking, flashings and weatherproofing"). Propose
`{key: insulation, text: "Wall, ceiling and floor insulation and draught sealing", separate: trade.insulation}`, and rewording `weatherproofing`
to "Sarking, flashings and weatherproofing".

**Sources.** `residential-construction-guide.md` "Energy Efficiency and
Thermal Performance"; `as-standards-reference.md` "AS/NZS 4859 Series —
Thermal Insulation Materials"; `trade-interfaces-coordination-guide.md`
Interface 7.

**Open questions.** O-SE17: acoustic insulation in internal walls and
floors: this trade or plasterboard? (Not in this RFT.)

## trade.garage_doors: Garage doors

**Summary.** Supply and install garage, roller and sectional doors with
motors, controls, remotes and seals, wind-rated and ember-sealed where
required. House: one or two garage doors. Commercial: roller shutters and
dock doors (fire shutters are passive fire).

**Interfaces.** Takes from head contractor (opening, lintel, headroom,
side room), trade.electrical (power point at the door head). Leaves
nothing to others.

**Hold points.** None from the certifier or engineer in the seeds.
Commissioning at handover.

**Certificates it signs.** Manufacturer's warranty; remotes and keys at
handover (seed handover list names garage door remotes). Bushfire ember
seals: AS 3959. `verified: false`.

**Licence (NSW).** Probably a NSW contractor licence for general building
work or a specialist class; not in the seeds. `verified: false`.

**Main Works line.** None today; `envelope.openings` covers windows and
external doors only. Propose `{key: garage_doors, text: "Garage, roller and sectional doors with their motors", separate: trade.garage_doors}`
under `envelope`.

**Sources.** `residential-construction-guide.md` "Handover and Practical
Completion", "Bushfire Construction Requirements". **Weak sourcing:**
no seed covers garage doors as a trade; scope lines follow
`knowledge/clusters/envelope/systems.yaml` `envelope.vehicle-doors`.

**Open questions.** O-SE18: is it a supply-and-install trade (as written)
or should it be a supply with an optional install stage?

## trade.curtain_wall: Facade and curtain wall

**Summary.** Engineers, mocks up, tests, makes and installs stick or
unitised curtain wall and facade systems with brackets, anchors, spandrel
panels and perimeter seals. Commercial and multi-storey only.

**Interfaces.** Takes from consultant.structural and the facade engineer
(design review), head contractor (cast-in channels at slab edges, crane
and hoist). Leaves for trade.passive_fire (slab-edge fire-stopping, unless
the documents give it to the facade contractor), joins trade.roofing and
trade.cladding at perimeters.

**Hold points.** Mock-up testing before production (seed: standard above
a building height it names); facade engineer reviews shop drawings; fire
engineer and certifier coordinate spandrel and slab-edge details; site
water testing before handover.

**Certificates it signs.** Facade test reports and product fire test
evidence. Standards: AS 2047, AS 1288, AS 4284, AS 5113, AS 1530.1,
AS 1530.3, NCC Part C2 spandrel provisions. `verified: false`.

**Licence (NSW).** Commercial facade work: no Home Building Act contractor
licence as far as the seeds go; on Class 2 buildings NSW design and
building practitioner registration may apply (not in these seeds).
`verified: false`.

**Main Works line.** Row `envelope`. Propose `{key: curtain_wall, text: "Curtain wall and unitised facade", separate: trade.curtain_wall}`.

**Sources.** `commercial-construction-guide.md` "Facade and Curtain Wall
Systems"; `ncc-reference-guide.md` "Cladding Combustibility
Requirements", "Spandrel and Slab Edge Fire Protection";
`multi-residential-apartments-guide.md` "Trade Package Structure"
(facade and windows bundled or split), "Weatherproofing and Facade
Defects".

**Open questions.** O-SE19: slab-edge fire-stopping: facade contractor or
passive fire contractor? (RFT: passive fire unless the documents say.)
O-SE20: on apartments the seed warns against splitting facade and windows
when they need one firm; should trade.curtain_wall and
supplier.windows_glazing be a suggested bundle?

## supplier.windows_glazing: Windows and glazing (supply)

**Summary.** Makes and delivers windows, external doors and glazing with
the wind, water and air ratings, safety glass and energy performance the
documents require, with labels and test evidence. Optional install stage.
House: aluminium or timber windows to BASIX. Commercial: window walls and
shopfronts (curtain wall is its own trade).

**Interfaces.** Takes from trade.carpentry_framing (openings framed to
size). Leaves for the head contractor or framing carpenter (install and
flashings unless priced), trade.rendering / trade.cladding (sealant joints
at the frame), trade.masonry (brickwork after flashing).

**Hold points.** Check measure before manufacture; builder signs off
installation and flashings before brickwork or render reaches window
level.

**Certificates it signs.** Window performance labels, test evidence and
energy ratings (U-value and SHGC) for BASIX or Section J. Standards:
AS 2047, AS 4420, AS 1288, AS/NZS 2208. `verified: false`.

**Licence (NSW).** Supply needs no contractor licence as far as the seeds
go; installation is residential building work needing a NSW contractor
licence (glazing or general building). `verified: false`.

**Main Works line.** Row `envelope`, existing line `openings` ("Windows,
external doors and glazing"). A supply-only award should print "supply by
separate contract; install included" (plan edge case), so propose the line
carry `separate: supplier.windows_glazing` with that supply wording.

**Sources.** `as-standards-reference.md` "AS 2047 — Windows and External
Glazed Doors in Buildings", "AS 1288 — Glass in Buildings";
`procurement-quoting-guide.md` §4 (BASIX evidence, lead time, colour
match); `trade-interfaces-coordination-guide.md` Interface 4.

**Open questions.** O-SE21: adding this file replaces the general RFT
for windows packages; `internal/store/report_trade_rft_test.go` expects
`supplier.windows_glazing` to fall back to `trade-general` and will need a
different supply id in that assertion. O-SE22: security screens: in this
supply or a separate item?

---

## Gaps I could not source

- Garage doors: no seed treats them as a trade.
- NSW licence classes for every trade here: the seeds only say specialist
  classes exist and name licensed plumbing and waterproofing. All licence
  lines are unverified.
- Precast and scaffolding inspection intervals and height triggers: named
  in the seeds with numbers; kept out of the RFT text.
- Reinforced masonry engineer hold points: inferred from AS 3700 practice,
  not stated as a hold point in a seed.
