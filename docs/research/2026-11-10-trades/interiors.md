# Trades research: interiors

Date: 10 November 2026. Slice T3 of `docs/plans/2026-11-10-trades-catalogue.md`.
Status: draft for owner review. Sources are the seed guides in
`data/reference/clerk/data/seed/` only. Every standard, NCC clause and licence
named below is `verified: false`: it is written as the seed names it and has
not been checked against the instrument or the NSW Fair Trading register.
The seeds use NCC 2019 clause numbers (for example F1.7, Part 3.8.1, D2.16).
NCC 2022 renumbered these, so the clause numbers here are leads to check, not
facts. RFT line text carries no clause numbers, rates, quantities, durations or
product selections.

RFT files written (all `status: draft`):

| Trade id | File |
|---|---|
| trade.plasterboard | `knowledge/rfp/trade-plasterboard.yaml` |
| trade.carpentry_fixout | `knowledge/rfp/trade-carpentry-fixout.yaml` |
| trade.waterproofing | `knowledge/rfp/trade-waterproofing.yaml` |
| trade.tiling | `knowledge/rfp/trade-tiling.yaml` |
| trade.painting | `knowledge/rfp/trade-painting.yaml` |
| trade.floor_coverings | `knowledge/rfp/trade-floor-coverings.yaml` |
| trade.timber_floors | `knowledge/rfp/trade-timber-floors.yaml` |
| trade.shower_screens | `knowledge/rfp/trade-shower-screens.yaml` |
| trade.metalwork | `knowledge/rfp/trade-metalwork.yaml` |

## The wet-area sequence (shared by four trades)

This is the one interface in this group where the order is a compliance gate,
not just good practice. The seeds put it as:

1. Frame inspection complete; services rough-in tested and inspected before
   lining (plumber, electrician, gasfitter each give their own evidence).
2. **Plumber** sets floor wastes, puddle flanges and pipe penetrations in their
   final position and at the specified level. Once the membrane is over the
   flange, the level cannot be changed without breaking the membrane.
3. **Falls**: made in the slab at the pour (concreter) or by a screed laid
   before the membrane. The seeds do not say who lays the screed (see open
   question W2).
4. **Plasterboard contractor** fixes the wet-area wall substrate:
   moisture-resistant board or fibre-cement sheet. The board is a substrate,
   not the water barrier, and must not be the final surface in a shower.
5. **Waterproofer** inspects and accepts the substrate, primes, fits bond
   breakers and corner reinforcement, seals to each flange and penetration,
   and applies the membrane. The membrane must lap continuously over the
   floor-to-wall junction.
6. Membrane cures for the manufacturer's period. Do not shorten it.
7. **Flood test** of horizontal membranes before tiling. The commercial seed
   requires it (it names a minimum period, not carried into the RFT); the
   apartments seed lists flood testing in the membrane hold point. The
   residential seeds do not mention a flood test, so the RFT asks for it
   "where the documents call for it".
8. **Inspection hold point before tiling.** The waterproofer gives a
   certificate for the work (new-dwelling-guide calls it the "waterproofer's
   statement of compliance"). The builder books the certifier's inspection.
   The tiler must not start until the certificate is sighted and the membrane
   has cured.
9. **Tiler** checks substrate flatness, tiles to fall to the wastes, sets grates
   flush, puts movement joints at perimeters and changes of plane (sealant, not
   grout).
10. **Plumber** fits tapware and fixtures; **shower screen contractor**
    measures after tiling and fixes the screen without puncturing the
    membrane, sealing each fixing.

**Who signs and when.** The licensed waterproofer signs the waterproofing
certificate for each area once the membrane is complete and cured, before it is
covered. The seeds disagree on the certifier's role:
`finishes-residential.md` (Inspection gate) says that in NSW the certifier must
inspect and approve the membrane before tiling, with a certifier's waterproofing
inspection certificate as evidence; `residential-construction-guide.md` says
the same "in all states"; but `new-dwelling-guide.md` (Structural inspection
gates) lists the inspector as "Certifier or installer" with the waterproofer's
statement as the evidence. The RFT says "offered for the builder's and
certifier's inspection" and makes the builder book it. Whether a NSW certifier
must inspect every wet area (Class 1 or Class 2), or only some rooms, is open
question W1. `verified: false`.

---

## trade.plasterboard: Plasterboard and ceilings

**Summary.** Lines walls and ceilings, sets and sands ready for paint, and
leaves wet-area walls as the substrate for membrane and tiles.

**House vs commercial.** House: timber-framed walls are the framer's; the
plasterer fixes board, cornices and sets. Commercial and apartments: the same
contractor usually builds steel stud partitions, suspended ceilings and
bulkheads, and the fire-rated and acoustic wall systems between units, which
have their own hold points (inter-tenancy wall build-up inspected before final
lining). Shop drawings of ceilings and partitions are common on commercial
jobs, rare on houses.

**Interfaces.**
- Takes from: trade.carpentry_framing (frame, noggings and backing for
  fixtures, grab rails and screens), trade.insulation (wall and ceiling
  insulation before lining), trade.hydraulic_plumbing, trade.electrical,
  trade.gas_fitting, trade.mechanical (rough-in tested and inspected before
  lining).
- Leaves for: trade.waterproofing (wet-area wall sheets), trade.tiling,
  trade.painting (set and sanded to the specified finish level; painter does
  final fill, light sand and sealer), trade.passive_fire (seals penetrations
  through rated linings; matches trade-passive-fire.yaml `rated_walls`, which
  gives rated walls to the head contractor or the plasterboard contractor).
- **Painter vs plasterer on set and sand.** The seeds put "plasterboard
  installation, stopping and sanding, cornice installation" in the linings
  stage (residential-construction-guide) and the procurement seed lists
  "Supply, fix, set, sand; cornices; ceiling" as the plaster package. Painting
  starts when "all set and plasterboard work [is] complete and dry". So the RFT
  gives set and sand to the plasterer and final filling, light sanding,
  caulking and sealer to the painter. The level of finish (how smooth) is left
  to the specification; the seeds name no finish levels.

**Hold points and inspections.** Pre-lining: frame and services rough-in
inspected before lining (certifier for frame; each licensed trade and the
builder for services). Commercial: fire-rated and acoustic walls inspected
before close-up (superintendent or certifier; fire engineer where a
performance solution applies).

**Certificates it signs.** None on a house. On fire-rated or acoustic systems,
an installer's statement that the tested system was built (commercial and
apartments). `verified: false`.

**Licence (NSW).** A NSW contractor licence for plastering (the seed says only
that specialist trades have subspecialty classes; it does not name them).
`verified: false`.

**Main Works line.** Row `interiors`, line `partitions` ("Internal walls,
linings and ceilings") exists today and is this trade's whole line.

**Sources.** finishes-residential.md (Plasterboard (gypsum board));
trade-interfaces-coordination-guide.md (Interface 3, Interface 6, Interface 7);
residential-construction-guide.md (Construction Sequence and Programming for
Residential Projects); procurement-quoting-guide.md (trade package table);
multi-residential-apartments-guide.md (SOU Separation — Fire and Acoustic;
Quality Hold Points for Apartment Construction).

**Open questions.**
- P1. Does the plasterer or the waterproofer supply the wet-area wall sheet
  (fibre-cement)? The RFT gives it to the plasterer.
- P2. On a house, are steel stud partitions and bulkhead framing ever the
  plasterer's, or always the framer's? The RFT includes them "the documents
  show".

---

## trade.carpentry_fixout: Carpentry (fix-out)

**Summary.** Hangs internal doors, fits hardware, and fixes skirtings,
architraves and other trims ready for paint.

**House vs commercial.** House: one fix-out carpenter does doors, trims,
shelving, manholes and often bathroom accessories. Commercial: doors often come
from a door and hardware package and fire doors are certified items; trims are
fewer (shadow-line, metal).

**Interfaces.**
- Takes from: trade.plasterboard (linings set), supplier.door_hardware
  (hardware supplied when let separately), trade.joinery_kitchens (joinery
  installed by its own contractor).
- Leaves for: trade.painting (fill, caulk, paint doors and trims),
  trade.floor_coverings and trade.timber_floors (doors trimmed to clear the
  finished floor; skirting order around floors).
- Fire and smoke doors are fire-passive (excluded from interiors.internal-doors);
  the RFT gives them to the head contractor unless the documents say otherwise.

**Hold points and inspections.** None statutory. The "fixing stage" is a
progress-payment stage in the HIA schedule (internal linings, internal doors,
skirtings and architraves, cabinets, tiling), so the builder inspects it.

**Certificates it signs.** None found in the seeds.

**Licence (NSW).** A NSW contractor licence for carpentry. `verified: false`.

**Main Works line.** Row `interiors`, line `doors` ("Internal doors and
hardware") exists. Propose a new line under `interiors`:
`{key: trims, text: "Skirtings, architraves and fix-out carpentry"}`, both
lines taken by this trade.

**Sources.** residential-construction-guide.md (Construction Sequence and
Programming for Residential Projects); trade-interfaces-coordination-guide.md
(Interface 6); cost-management-principles.md (7. Progress claim assessment
against HIA stage-payment schedule); fire-life-safety-guide.md.

**Open questions.**
- C1. Do skirtings go on before or after timber and carpet floors? The seed
  (Interface 6) puts skirtings before paint and floors after paint; floating
  timber floors usually need skirtings after. The RFT says "before the floor
  finishes the documents sequence after them".
- C2. Are bathroom accessories (towel rails, roll holders) the fix-out
  carpenter's or the plumber's on your jobs?

---

## trade.waterproofing: Waterproofing (wet areas and balconies)

**Summary.** Membranes to internal wet areas and to balconies, decks, terraces
and planters; proves them before they are covered and certifies each area.

**House vs commercial.** House: one licensed waterproofer does bathrooms,
laundry and any balcony; the inspection is the certifier's (see W1).
Apartments: waterproofing is the largest defect category; leaks travel to
units below, so hold points are tighter (substrate inspection, membrane
inspection with flood testing, balcony inspection) and balcony waterproofing is
often a separate enclosure package let early. Commercial: amenities, kitchens,
plant rooms, podium and roof decks; independent waterproofing inspection is
recommended for critical areas. Basement tanking is below-ground waterproofing
(trade.below_ground_waterproofing), not this trade.

**Interfaces.** See the wet-area sequence above.
- Takes from: trade.hydraulic_plumbing (wastes, puddle flanges, penetrations
  in final position and level), trade.concreting (falls in the slab),
  trade.plasterboard (wet-area wall sheets), trade.tiling or trade.concreting
  (falls screed under the membrane, open question W2).
- Leaves for: trade.tiling (cured, inspected, certified membrane),
  trade.shower_screens and trade.metalwork (fixings through the membrane must
  be sealed), trade.cladding and trade.curtain_wall (balcony membrane must be
  continuous with the wall waterproofing at the threshold).

**Hold points and inspections.**
- Substrate, falls and penetrations accepted before membrane (builder;
  apartments: quality hold point).
- Membrane inspected after cure, before tiling (NSW certifier per two seeds;
  "certifier or installer" per new-dwelling-guide). Mandatory hold point.
- Flood test of horizontal membranes before tiling (commercial and apartments
  seeds; the commercial seed names a minimum test period).
- Balcony inspection: falls, upstands, drainage, threshold (apartments).
- Photographs of every completed area before covering.

**Certificates it signs.** The waterproofer's certificate (statement of
compliance) for each wet area and balcony, against AS 3740 (internal) and
AS 4654.2 (external), with membrane material to AS 4858. Handed over with
the waterproofing warranty. Standards as named in the seeds:
AS 3740-2021, AS 4654.1-2012, AS 4654.2-2012, AS 4858-2004;
NCC Volume Two Part 3.8.1, Volume One Part F1.7. All `verified: false`.

**Licence (NSW).** A NSW contractor licence for waterproofing. The seeds say
waterproofing must be done by a licensed waterproofer in NSW, QLD, VIC and most
other jurisdictions. `verified: false`.

**Main Works line.** Row `interiors`, line `wet_areas` ("Wet area
waterproofing and tiling") exists but is shared with tiling. Propose splitting
it into `{key: wet_area_waterproofing, text: "Wet area waterproofing"}` and
`{key: tiling, text: "Wall and floor tiling"}` (the latter for trade.tiling).
Propose a new line under row `envelope`:
`{key: balcony_waterproofing, text: "Waterproofing of balconies, decks and roof terraces"}`.
The envelope `weatherproofing` line stays with Main Works or the roofing and
insulation trades.

**Sources.** finishes-residential.md (Wet area waterproofing; Inspection gate);
trade-interfaces-coordination-guide.md (Interface 2; Interface 5);
residential-construction-guide.md (Wet Area Construction and Waterproofing;
Handover and Practical Completion); new-dwelling-guide.md (Structural
inspection gates — Class 1a new dwelling); ncc-reference-guide.md (Wet Area
Waterproofing (AS 3740)); as-standards-reference.md (AS 3740 — Waterproofing
of Domestic Wet Areas; AS 4654 Series); commercial-construction-guide.md
(Waterproofing in Commercial Buildings); multi-residential-apartments-guide.md
(Wet Area Waterproofing; Balcony and External Waterproofing; Quality Hold
Points for Apartment Construction; Trade Package Structure).

**Open questions.**
- W1. In NSW, must the certifier inspect every wet area before tiling, or is
  the waterproofer's certificate enough for some classes or some rooms? Seeds
  disagree.
- W2. Who lays a falls screed under the membrane on your jobs: concreter,
  tiler or waterproofer?
- W3. Should flood testing be a default line on every job, or only where the
  documents call for it (current)?
- W4. Should balcony waterproofing be its own trade row on apartments (the
  apartments seed lists it as a separate enclosure package)?

---

## trade.tiling: Tiling

**Summary.** Wall and floor tiles over a certified membrane, falling to the
wastes, bonded, with movement joints.

**House vs commercial.** House: often let with waterproofing (the `wet_areas`
bundle); tiles often a prime cost item or client-supplied. Commercial and
apartments: larger formats, epoxy grout in commercial kitchens and amenities,
tile set-out drawings, and common-area tiling.

**Interfaces.**
- Takes from: trade.waterproofing (membrane and its certificate),
  trade.hydraulic_plumbing (wastes and grates at level), trade.plasterboard
  (flat substrate), trade.concreting (slab falls).
- Leaves for: trade.hydraulic_plumbing (tapware and fixture fit-off),
  trade.shower_screens (measure after tiling), trade.joinery_kitchens
  (splashbacks where the joiner does not).
- Scope trap from the procurement seed: "shower screen interface; tapware
  install ownership".

**Hold points and inspections.** Must not start until the waterproofing
inspection is passed and the certificate sighted (mandatory). Builder checks
falls. At handover, a tap test for hollow (drummy) tiles.

**Certificates it signs.** None statutory found. Work to AS 3958.1 (tile
installation guide), adhesives classed to AS ISO 13007. `verified: false`.

**Licence (NSW).** A NSW contractor licence for wall and floor tiling.
`verified: false`.

**Main Works line.** Proposed `{key: tiling, text: "Wall and floor tiling"}`
under row `interiors`, split from the existing `wet_areas` line (see
waterproofing).

**Sources.** finishes-residential.md (Tiling; Substrate preparation; Adhesive
and grout selection; Expansion and movement joints; Common defects);
trade-interfaces-coordination-guide.md (Interface 2; Interface 5);
defects-and-dlp-guide.md (2.4 Tile drumming (debonded tiles));
procurement-quoting-guide.md (trade package table).

**Open questions.**
- T1. Who sets the floor waste grate flush with the tiles: tiler (RFT) or
  plumber?
- T2. Who seals the wall-to-floor and screen junctions with silicone: tiler or
  screen installer?

---

## trade.painting: Painting

**Summary.** Prepares and paints internal and external surfaces to the colour
schedule; touches up at the end.

**House vs commercial.** House: one painter for internal and external, plus
stains. Commercial: larger areas, more factory-finished surfaces, line marking
and specialist coatings (intumescent coatings are fire-passive, not this trade).
In existing buildings old paint may contain lead (renovation seed lists lead
paint in the hazardous materials check).

**Interfaces.**
- Takes from: trade.plasterboard (set and sanded), trade.carpentry_fixout
  (doors and trims fixed), trade.rendering and trade.cladding (external
  surfaces ready).
- Leaves for: trade.floor_coverings and trade.timber_floors (painting before
  floors; the most common scheduling error on houses), trade.builders_clean.
- Set and sand: see plasterboard.

**Hold points and inspections.** No formal hold point. Builder enforces two
sequence rules: plaster dry and cured before painting; painting before carpet
and floors.

**Certificates it signs.** None found. Internal paint is usually a cosmetic
warranty item time-limited by the defects period.

**Licence (NSW).** A NSW contractor licence for painting and decorating.
`verified: false`.

**Main Works line.** Row `interiors`, line `finishes` ("Floor and wall
finishes and painting") is shared with two floor trades. Propose splitting it
into `{key: painting, text: "Internal and external painting"}`,
`{key: floor_coverings, text: "Carpet and resilient floor coverings"}` and
`{key: timber_floors, text: "Timber floors, sanding and coating"}`. External
painting is not an interiors system (envelope.external-walls excludes it from
interiors.painting), but the painter usually does both; the RFT has an
external stage the PM can exclude.

**Sources.** trade-interfaces-coordination-guide.md (Interface 6);
defects-and-dlp-guide.md (2.5 Paint defects); finishes-residential.md (Common
defects at key trade interfaces); procurement-quoting-guide.md (trade package
table); renovation-guide.md (hazardous materials check).

**Open questions.**
- PA1. Should the Main Works painting line sit under `interiors` for both
  internal and external painting, or should external painting go under
  `envelope`?

---

## trade.floor_coverings: Floor coverings

**Summary.** Carpet, underlay and resilient floors (sheet vinyl, vinyl plank
and tile), with transitions, laid after painting.

**House vs commercial.** House: carpet to bedrooms, vinyl plank to living and
wet zones; often a prime cost item. Commercial and apartments: carpet tiles,
welded sheet vinyl in wet and clinical areas, acoustic underlays (those are
acoustic-separation when part of a rated floor).

**Interfaces.**
- Takes from: trade.concreting (slab flatness), trade.painting (painting done),
  trade.carpentry_fixout (skirtings; doors trimmed to clear).
- Leaves for: trade.builders_clean.
- Not tiles (trade.tiling) or timber (trade.timber_floors).

**Hold points and inspections.** None statutory. Moisture testing of slabs
before laying; builder's sequence check (painting and wet trades finished).

**Certificates it signs.** None found.

**Licence (NSW).** Not named in the seeds. Residential building work in NSW
needs a contractor licence in a matching class (role-builder.md); the class
for floor coverings is to be checked. `verified: false`.

**Main Works line.** Proposed `{key: floor_coverings, text: "Carpet and
resilient floor coverings"}` under `interiors`, split from `finishes`.

**Sources.** finishes-residential.md (Flooring; Luxury vinyl plank / tile
(LVP / LVT); Carpet; Polished concrete); trade-interfaces-coordination-guide.md
(Interface 6); role-builder.md (Builder's licence); procurement-quoting-guide.md
(trade package table).

**Open questions.**
- F1. Should major slab levelling be priced by this trade (as a provisional
  line) or stay with the head contractor (current)?

---

## trade.timber_floors: Timber floors and sanding

**Summary.** Lays solid or engineered timber floors, sands and coats them, and
re-sands existing floors.

**House vs commercial.** Mostly houses and apartments. Over slabs it needs a
moisture barrier; boards are acclimatised on site. In apartments the acoustic
underlay may be part of an impact-rated floor (acoustic-separation).

**Interfaces.**
- Takes from: trade.carpentry_framing (joists and sheet subfloor),
  trade.concreting (slab, dry), trade.painting (seed sequence: paint before
  floors).
- Leaves for: trade.carpentry_fixout (skirtings or scotia after a floating
  floor), trade.painting (final touch-up), trade.builders_clean.

**Hold points and inspections.** None statutory. Moisture testing and
acclimatisation before laying.

**Certificates it signs.** None found.

**Licence (NSW).** Not named in the seeds; possibly the carpentry class or a
timber floor class. `verified: false`.

**Main Works line.** Proposed `{key: timber_floors, text: "Timber floors,
sanding and coating"}` under `interiors`, split from `finishes`.

**Sources.** finishes-residential.md (Timber flooring (solid and
engineered)); trade-interfaces-coordination-guide.md (Interface 6);
structural-residential.md; procurement-quoting-guide.md.

**Open questions.**
- TF1. On your jobs, is a timber floor sanded and coated before or after the
  final paint coat? The seed says floors after paint; sanding dust often
  argues for coating last.

---

## trade.shower_screens: Shower screens and mirrors

**Summary.** Measures after tiling, supplies and fixes shower and bath screens
in safety glass, mirrors and internal glazed panels.

**House vs commercial.** House: framed or frameless screens and vanity
mirrors. Apartments: a finishes package ("mirrors and shower screens"),
repeated per unit. Commercial: internal glazed partitions may join this
package or the glazing supplier's.

**Interfaces.**
- Takes from: trade.tiling (tiling finished; measure after), trade.waterproofing
  (membrane behind and under; fixings must not breach it unsealed),
  trade.carpentry_framing (noggings or backing for fixings).
- Leaves for: trade.builders_clean.
- Not glass balustrades (trade.metalwork) or windows (supplier.windows_glazing).

**Hold points and inspections.** None statutory. Safety glass in a wet area is
a certifier's final-inspection item (annealed glass in a safety location is
listed as a serious non-compliance).

**Certificates it signs.** A statement that the glass is safety glass where the
location needs it (toughened to AS/NZS 2208 or laminated, selected to AS 1288).
`verified: false`.

**Licence (NSW).** A NSW contractor licence for glazing. `verified: false`.

**Main Works line.** None exists. Propose `{key: screens, text: "Shower
screens, mirrors and internal glazing"}` under row `interiors`.

**Sources.** finishes-residential.md (Safety glazing requirements);
as-standards-reference.md (AS 1288 — Glass in Buildings);
ncc-reference-guide.md (safety glazing pitfall);
multi-residential-apartments-guide.md (Trade Package Structure; Wet Area
Waterproofing); procurement-quoting-guide.md.

**Open questions.**
- S1. Should vanity mirrors sit here or with the joinery contractor (mirror
  integration is listed as a joinery scope trap)?

---

## trade.metalwork: Metalwork and balustrades

**Summary.** Designs where left to it, makes and fixes balustrades, handrails,
steel stairs and miscellaneous metalwork, and shows the barriers meet their
loads.

**House vs commercial.** House: balcony and deck balustrades, stair
balustrades and handrails, often glass or stainless. Commercial: also
mezzanine and void edges, ramps, stair handrails in fire stairs, and more
public-loading. Plant guardrails and ladders are maintenance access (AS 1657),
not this trade by default. Pool barriers are trade.fencing.

**Balustrades as barriers (compliance in words).** A barrier is needed at the
edge of a floor, balcony, landing, stair or raised platform where a person can
fall more than a set height, and at openings in floors. It must be high enough
for its location (lower beside fixed seating), have no opening a small child
can pass through, not be climbable where the rules say so, and resist the
horizontal loads for its use (residential loads are lower than public
assembly). A handrail is not a barrier: a stair may need both. Glass in a
barrier must be safety glass of a type made for barriers; a frameless glass
balustrade needs laminated toughened glass that still holds the load if one
ply breaks. Fixings through a balcony membrane must be sealed into it. The
seeds name: NCC Volume One Part D2.16 to D2.19 (handrails and barriers),
Volume Two Part 3.9 (safe movement and access), AS/NZS 1170.1 (barrier loads),
AS 1288 Section 7 (balustrade glazing), AS 1657 (plant access). All
`verified: false`; NCC 2022 renumbered these parts. Barrier heights, opening
sizes and loads are in the seeds but are not carried into the RFT.

**Interfaces.**
- Takes from: head contractor or trade.carpentry_framing / trade.concreting
  (structure and blocking for fixings), trade.structural_steel (primary
  steel), trade.waterproofing (membrane sealed around fixings).
- Leaves for: trade.painting (site painting where not factory-finished),
  trade.builders_clean.

**Hold points and inspections.** Engineer's hold points for fixings and
connections where the engineer sets them; certifier checks barriers at final
inspection (a missing balustrade stops practical completion per the contract
administration seed).

**Certificates it signs.** An engineer's or designer's certificate that the
barriers and their fixings meet their loads, where the design is the
contractor's. `verified: false`.

**Licence (NSW).** Not named in the seeds. Installing balustrades on
residential work is residential building work and needs a contractor licence
in a matching class; glass balustrades may need the glazing class. To be
checked. `verified: false`.

**Main Works line.** None exists (row `structure` line `suspended` covers
"Suspended slabs, stairs and landings", which is the structure, not the
barriers). Propose `{key: balustrades, text: "Balustrades, handrails and
metalwork"}` under row `interiors`, or under `structure` if the owner prefers
it next to stairs.

**Sources.** ncc-reference-guide.md (Stairways, Ramps, and Barriers; Pool
Barriers and Construction in Flood Hazard Areas); as-standards-reference.md
(AS 1288 — Glass in Buildings; AS 1657 — Fixed Platforms, Walkways, Stairways
and Ladders; AS/NZS 1170 Series — Structural Design Actions);
finishes-residential.md (Safety glazing requirements);
contract-administration-guide.md (PC incomplete items);
multi-residential-apartments-guide.md (Balcony and External Waterproofing);
structural-commercial-industrial-guide.md.

**Open questions.**
- M1. Which Main Works row should hold balustrades: `interiors` or
  `structure`?
- M2. Should the RFT make the balustrade design the contractor's by default
  (common on houses) or only where the documents leave it (current)?

---

## Proposed Main Works lines (not applied; main-works.yaml unchanged)

| Row | Line today | Proposed | Trade |
|---|---|---|---|
| interiors | `partitions` | keep | trade.plasterboard |
| interiors | `doors` | keep | trade.carpentry_fixout (with supplier.door_hardware) |
| interiors | — | `{key: trims, text: "Skirtings, architraves and fix-out carpentry"}` | trade.carpentry_fixout |
| interiors | `wet_areas` | split: `{key: wet_area_waterproofing, text: "Wet area waterproofing"}` | trade.waterproofing |
| interiors | `wet_areas` | split: `{key: tiling, text: "Wall and floor tiling"}` | trade.tiling |
| interiors | `finishes` | split: `{key: painting, text: "Internal and external painting"}` | trade.painting |
| interiors | `finishes` | split: `{key: floor_coverings, text: "Carpet and resilient floor coverings"}` | trade.floor_coverings |
| interiors | `finishes` | split: `{key: timber_floors, text: "Timber floors, sanding and coating"}` | trade.timber_floors |
| interiors | — | `{key: screens, text: "Shower screens, mirrors and internal glazing"}` | trade.shower_screens |
| interiors | — | `{key: balustrades, text: "Balustrades, handrails and metalwork"}` | trade.metalwork |
| envelope | — | `{key: balcony_waterproofing, text: "Waterproofing of balconies, decks and roof terraces"}` | trade.waterproofing |

## Not sourced

- NSW licence class names for every trade: the seeds say only that specialist
  trades have subspecialty classes. All licence lines above are general
  knowledge, `verified: false`.
- Floor coverings, timber floors and metalwork licence classes: no seed.
- Plasterboard finish levels: no seed names them; left to the specification.
- Who lays a falls screed and who sets grates flush: no seed; owner questions
  W2 and T1.
