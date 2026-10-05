---
tier: topic
seed_type: trade
loaded_by: task-loaded
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
topics: [trade-interfaces, coordination, defects]
summary: "The critical sequencing interfaces between residential trades — who is responsible, what evidence gates the next trade, and the common failure mode at each boundary. Used for programme coordination, scope drafting, hold points and defect root-cause work."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Trade seed — Trade interfaces and coordination (residential)

This seed documents the critical sequencing interfaces between residential building trades. Each interface section states who is responsible, what evidence is required before the next trade proceeds, and what the most common failure mode is. Use this seed when coordinating programmes, drafting scope of works, reviewing hold points, or investigating defects at trade boundaries.

Load this seed task-loaded when a role × archetype task involves trade sequencing, hold-point management, scope boundary disputes, or defect root-cause analysis.

---

## How to use this seed

Each interface is structured as:
- **Sequence:** the order trades must proceed
- **Hold point:** what must be inspected or certified before the next trade begins
- **Responsibility:** who owns the hold point
- **Common failure mode:** the defect that results when this interface is mismanaged

The agent must not allow a programme to advance past a hold point without confirmed evidence. Reference §evidence-discipline.

---

## Interface 1: Frame to roof

### Sequence

1. Frame erected and inspected (certifier frame inspection certificate issued)
2. Roof trusses or rafters installed and braced
3. Sarking laid over rafters before battens
4. Roof plumbing rough-in (downpipe brackets, vent penetrations, flue penetrations) before roofing commences
5. Tiling battens fixed; tiles or metal sheeting laid
6. Ridge, hip, and valley work completed
7. All roof flashings and penetrations sealed

### Hold point

Frame inspection certificate from certifier before any roof cladding is fixed. No exceptions — roof cladding conceals the frame from inspection.

**Responsibility:** Builder / owner-builder to arrange certifier inspection. Roofing subcontractor must not commence until certificate is sighted.

### Common failure mode

- **Sarking omitted or installed after battens:** Sarking is difficult or impossible to install correctly after battens are fixed. The agent must confirm sarking specification and installation sequence is included in the roofing scope.
- **Roof plumbing penetrations sealed after tiling:** Roof plumbing rough-in brackets and vent locations must be coordinated before tiling. Flashings for plumbing penetrations cannot be correctly installed if tiles are already laid.
- **Mortar-bedded ridge caps:** Ridges pointed with mortar rather than dry-fixed with foam closures crack and detach within a few years. Specify dry-fix or mechanical fixing.

---

## Interface 2: Wet area waterproofing to tiling

### Sequence

1. Frame inspection complete
2. Wet area substrate installed (MR plasterboard, FC sheet, or masonry)
3. All penetrations (pipes, waste outlets, floor wastes) trimmed and set before membrane
4. Waterproofing membrane applied — corners and junctions reinforced
5. Membrane cured (minimum period per manufacturer — typically 24–48 hours; do not shorten)
6. Certifier waterproofing inspection certificate issued
7. Tiles installed over membrane

### Hold point

Certifier waterproofing inspection before tiling commences. This is a mandatory NCC hold point and cannot be waived.

**Responsibility:** Builder / owner-builder to arrange certifier inspection. Tiler must not commence until certificate is sighted and membrane is fully cured.

### Common failure modes

- **Tiling before certification:** The single most common wet area defect exposure. Tiles concealing an uninspected or defective membrane can result in full bathroom strip-out at the builder's cost years later when leaks appear.
- **Penetrations not set before waterproofing:** Pipes trimmed through the membrane after application create unsealed holes. All plumbing waste and supply penetrations must be in final position and set in the substrate before membrane application.
- **Tile adhesive applied before membrane cures:** Solvent or moisture in the adhesive can attack a partially cured liquid membrane. Always confirm manufacturer's curing times.
- **No membrane at substrate transitions:** Membrane must lap continuously over the junction between the floor and wall substrate (e.g. concrete floor to plasterboard wall). A gap at this junction is an assured leak path.

---

## Interface 3: Kitchen and bathroom services rough-in before wall lining

### Sequence

1. Frame inspection complete
2. Plumbing rough-in: cold and hot water supply pipes run to final outlet positions; waste pipes run and set in floor/wall
3. Electrical rough-in: conduit and cable to all power, lighting, and exhaust fan positions
4. Gas rough-in: gas supply pipes run to cooktop, oven, and HW positions
5. All rough-ins pressure-tested and inspected before wall lining commences
6. Certifier inspection (where required for hydraulic and gas rough-in)
7. Wall and ceiling lining (plasterboard or FC sheet) installed
8. Plasterboard set, paint coat applied, kitchen/bathroom fit-off commences

### Hold point

All services rough-in pressure-tested and inspected by licensed tradesperson before lining. Plumbing compliance check by certifier at rough-in stage (NSW: certifier inspects plumbing at stages defined in the CC).

**Responsibility:** Each licensed trade (plumber, electrician, gasfitter) provides their own compliance evidence at rough-in. Builder coordinates inspection sequence.

### Common failure modes

- **Wall lined before services inspected:** Services concealed behind linings without sign-off is the most common cause of having to open walls later. The cost of opening and reinstatement far exceeds the cost of waiting for an inspection.
- **Outlet positions incorrect at rough-in:** Outlet positions confirmed on the plan do not match actual kitchen or bathroom layout. Correcting positions after lining is expensive. Builder must confirm final joinery and appliance layout against rough-in positions before lining proceeds.
- **Services too close to framing members:** Minimum clearances from framing members must be maintained for notching/drilling compliance (see `structural-residential.md`). Pipe or conduit installed without clearance causes frame weakening and may fail inspection.

---

## Interface 4: External cladding — windows and doors before render or brickwork

### Sequence

1. Frame inspection complete and any structural elements (lintels, supports) confirmed
2. Windows and external doors installed in openings
3. Window flashings installed: head flashing over window (diverts water away from frame top), sill flashing beneath window (diverts water away from sill), jamb flashings at sides
4. All flashing laps and seals confirmed watertight
5. Brickwork or render commences — flashings must be already in place before brick skin or render coat reaches window level
6. Weepholes left open in brickwork (not mortared over) at base of brick skin and above each window

### Hold point

Window installation and flashing complete (builder's quality inspection record) before brickwork passes window height or render base coat reaches window level. No formal certifier hold point but a critical builder hold point.

**Responsibility:** Builder to inspect and sign off window installation and flashing before masonry or render subcontractor proceeds.

### Common failure modes

- **Brickwork laid before window flashing installed:** The most common cause of water ingress at windows in brick veneer construction. Once the brick skin is at lintel height, flashings cannot be correctly installed.
- **Head flashing omitted or too short:** Head flashing must extend the full width of the opening plus a minimum overhang each side. A short flashing allows water to track back and enter the wall cavity.
- **Render applied over the window frame edge without a backing rod and sealant joint:** Render keyed directly to the window frame cracks as the frame moves. A backing rod and flexible sealant joint must be raked out and sealed at the render-to-frame junction.
- **Weepholes blocked by mortar:** Bricklayers filling weepholes with mortar during construction is common. Weepholes must be clear at completion; check and clear during PC inspection.

---

## Interface 5: Wet area floor waste and tile falls

### Sequence

1. Concrete slab poured with floor waste set at correct level (falls created at slab pour or by levelling screed)
2. If screed required: laid to correct level before waterproofing
3. Waterproofing applied over screed and slab
4. Certified inspection
5. Tiles laid with consistent tile falls to waste (minimum 1:100 fall to floor waste — AS 3958)

### Hold point

Floor waste outlet level set and confirmed before waterproofing membrane applied. Once waterproofing is over the floor waste flange, level cannot be adjusted without breaking waterproofing.

**Responsibility:** Hydraulic engineer confirms floor waste positions and levels at design stage. Plumber sets waste to specified level. Builder confirms level before waterproofing commences.

### Common failure modes

- **Flat tile fields pooling water:** Tile falls to waste < 1:100 or inadvertent reverse falls (tiles pitching away from waste). Water pools on shower floor creating hygiene and slip risk.
- **Floor waste flange at wrong height:** Flange set too high protrudes above finished tile; set too low is concealed below the tile surface and requires a grate recessed into the tile (difficult and expensive to fix later).

---

## Interface 6: Internal painting before flooring

### Sequence

1. All set and plasterboard work complete and dry
2. All cornices, architraves, and skirtings installed
3. Two coats of paint applied to walls and ceilings (or as specified)
4. Final coat of paint applied; tack-free dry
5. Timber, engineered, or LVP flooring laid
6. Final carpet installation (must follow painting — foot traffic damages carpet pile; paint splatter damages carpet)
7. Final paint touch-up and skirting board top coat (if required)

### Hold point

No formal hold point, but the sequencing failure (flooring before painting) is the most common scheduling error on residential projects.

**Responsibility:** Builder to enforce the sequence via the programme and subcontractor briefing.

### Common failure mode

- **Flooring installed before painting:** Paint spatters on finished flooring, and foot traffic from painting works damages finished floor surface. Rectification of paint splatter on timber or LVP is expensive. Carpet can rarely be cleaned of paint satisfactorily.
- **Plaster not cured before painting:** Fresh plaster (particularly base coat set on new plasterboard) requires a minimum curing period (typically 4 weeks for a fully cured coat) before a quality paint finish can be achieved. Painting too early causes patchy absorption and finish failure.

---

## Interface 7: Roof framing to insulation and ceiling

### Sequence

1. Frame inspection complete
2. Roof insulation (sarking, reflective foil, or blanket insulation between rafters) installed as part of the roof frame or roof cladding installation
3. Ceiling joists and top chord of trusses confirmed
4. Ceiling batts (bulk insulation) laid in ceiling space after roof is weathertight
5. Ceiling lining (plasterboard) installed
6. Recessed lighting and exhaust fan penetrations cut and sealed after ceiling lining

### Hold point

Roof weathertight (all roofing, flashings, and penetrations complete and sealed) before ceiling insulation is installed — bulk insulation batts wet during installation lose R-value and may not recover.

**Responsibility:** Builder to confirm weathertightness before authorising ceiling insulation installation.

### Common failure mode

- **Ceiling insulation installed before roof is weathertight:** Batts wet during or after installation — compressed, displaced, or mouldy insulation. Thermal performance not achieved; BASIX non-compliance risk.
- **Recessed downlights cutting into insulation barrier:** IC-rated (insulation contact) downlights are required wherever recessed downlights are installed in insulated ceilings. Non-IC-rated fittings require a 200 mm clearance from insulation — a thermal bridge and fire risk if not maintained.

---

## Interface 8: External landscaping, paths, and drainage before occupation

### Sequence

1. External ground levels set (no slab or footing undermining)
2. Driveway and path concrete poured after all underground services (conduit, drainage) are in place
3. Garden beds graded away from the building (minimum 50 mm fall in first 1 m from building per NCC)
4. Stormwater pit and grate locations confirmed
5. OSD tank (if required) installed and operational before OC application

### Hold point

External drainage and finished ground levels must direct stormwater away from the building before OC is issued. Certifier will inspect external levels as part of final inspection.

**Responsibility:** Builder to confirm external grading; hydraulic engineer (or hydraulic contractor) to confirm OSD installation.

### Common failure mode

- **Garden bed soil against external cladding:** Soil or mulch piled against timber weatherboard, FC sheet, or even brick veneer creates a moisture pathway into the wall. The agent must flag DPC height and ground clearance requirements at PC inspection.
- **OSD not operational at OC application:** A common cause of delayed OC. OSD tank installation requires council approval and a compliance certificate. Flag early in the programme as a long-lead item.

---

## Coverage depth

| Interface | Depth |
|---|---|
| Frame to roof | Moderate |
| Wet area waterproofing to tiling | Deep |
| Services rough-in before wall lining | Moderate |
| Windows and doors before render/brickwork | Moderate |
| Wet area floor waste and tile falls | Moderate |
| Internal painting before flooring | Moderate |
| Roof framing to insulation and ceiling | Moderate |
| External landscaping and drainage before OC | Shallow |

---

## Low-confidence flags

- **Non-standard wet area substrates** (e.g. compressed fibre cement over heated floors, or render over masonry for tiling): Compatibility between substrate, waterproofing system, adhesive, and tile must be confirmed with each manufacturer — do not assume compatibility.
- **Multi-dwelling fire and acoustic interfaces:** Party wall construction sequencing (acoustic isolation, fire stopping at penetrations) is not covered in this seed. Reference NCC Volume One Section C/F and engage an acoustic engineer.
- **Prefabricated bathroom pods:** Used in some multi-dwelling projects. Interface with the structural frame, services rough-in, and waterproofing differs from on-site wet area construction. Flag for specialist advice if pods are specified.
