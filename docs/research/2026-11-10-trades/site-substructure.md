# Trades research: site and substructure

Date: 10 November 2026. Slice T3 of `docs/plans/2026-11-10-trades-catalogue.md`.
Status: draft for owner review. Sources are the seed guides in
`data/reference/clerk/data/seed/` only, plus general Australian practice where
marked. Every standard, NCC clause and licence named below is
`verified: false`: it is written as the seed names it and has not been checked
against the instrument, SafeWork NSW, the NSW EPA or NSW Fair Trading. The
seeds use NCC 2019 numbering in places (for example "NCC Volume Two Part
3.1.3"); NCC 2022 renumbered these, so treat them as leads to check. RFT line
text carries no clause numbers, rates, quantities, durations or product
selections.

RFT files written (all `status: draft`, shape of `trade-structural-steel.yaml`):

| Trade id | File |
|---|---|
| trade.demolition | `knowledge/rfp/trade-demolition.yaml` |
| trade.asbestos_removal | `knowledge/rfp/trade-asbestos-removal.yaml` |
| trade.earthworks | `knowledge/rfp/trade-earthworks.yaml` |
| trade.stormwater_drainage | `knowledge/rfp/trade-stormwater-drainage.yaml` |
| trade.landscaping | `knowledge/rfp/trade-landscaping.yaml` |
| trade.fencing | `knowledge/rfp/trade-fencing.yaml` |
| trade.paving | `knowledge/rfp/trade-paving.yaml` |
| trade.piling | `knowledge/rfp/trade-piling.yaml` |
| trade.concreting | `knowledge/rfp/trade-concreting.yaml` |
| trade.formwork | `knowledge/rfp/trade-formwork.yaml` |
| trade.steel_fixing | `knowledge/rfp/trade-steel-fixing.yaml` |
| trade.termite_management | `knowledge/rfp/trade-termite-management.yaml` |
| trade.below_ground_waterproofing | `knowledge/rfp/trade-below-ground-waterproofing.yaml` |

Every file has `design` (design, shop drawings or the trade's plan),
`establishment`, one to four stages of the trade's own work, and `handover`.
Positions use `kind: technical, value: X` with a `responsible` party, as in the
structural steel file.

## The slab and pour sequence (concreting, formwork, steel fixing)

The three trades share footings, the ground slab and the suspended structure
(`substructure.footings`, `substructure.slab-on-ground`, `structure.concrete`),
so the boundary is the whole question. What the seeds give, and what the RFTs
assume:

1. **Set-out.** The head contractor (or, with no head contractor, the builder
   or owner-builder) provides the survey set-out, grid lines and levels from
   the survey control. Seeds: `residential-construction-guide.md` (Construction
   Sequence: "builder's set-out of footings from survey"),
   `surveying-guide.md` (set-out responsibility must be stated in the tender),
   `role-owner-builder.md` (trap: "assuming one trade includes set-out").
   All three RFTs carry the position `set_out`, by the head contractor. The
   formwork trade sets its edge forms to that set-out; it does not set out the
   building.
2. **Excavation.** Footing and beam excavation is in the earthworks RFT
   (`detailed.footings`); the concreting RFT names it as a position by the
   earthworks trade. General practice: on houses the slab crew often digs its
   own beams; the PM excludes the earthworks line then (open question S2).
3. **Formwork.** Edge forms, rebates, set-downs and void formers, by the
   formwork trade (`formwork ground.edge_forms`, `ground.void_formers`). The
   formworker checks dimensions, levels and alignment before reinforcement is
   fixed (`as-standards-reference.md`, AS 3610 pitfalls: "Failing to inspect
   formwork dimensions and alignment before placing reinforcement").
4. **Vapour barrier under a slab on ground.** No seed says which trade lays
   it. `substructure.damp-waterproofing` holds it as a system and
   `trade.below_ground_waterproofing` builds that system, but on houses the
   slab crew lays it. The RFTs put it in the formwork line
   (`ground.void_formers`) and again as an excludable line in below-ground
   waterproofing (`membranes.dpm`, "where the documents give it to you"). The
   PM keeps one. Open question S1.
5. **Termite barriers and services.** Plumber and electrician set
   penetrations and cast-in conduits in their final position; the termite
   trade then fits penetration collars and under-slab and edge barriers before
   the pour (`residential-construction-guide.md`, Termite Management:
   "Failing to install physical barriers at all slab penetrations").
6. **Reinforcement, bar chairs and tie wire.** By the steel fixer: supply,
   cut and bend to schedules, fix, with bar chairs, spacers and tie wire that
   hold the specified cover (`steel_fixing supply.chairs`). The concreting and
   formwork RFTs both carry the position `reinforcement`, "Reinforcement, bar
   chairs, spacers and tie wire, by the steel fixer". The seeds list cover,
   laps and tie wire as pre-pour inspection items
   (`multi-residential-apartments-guide.md`, Quality Hold Points) but do not
   say who supplies chairs; giving them to the steel fixer is general
   practice so the trade that is inspected on cover owns what makes cover.
   Open question S3.
7. **Pre-pour (reinforcement) inspection.** A mandatory hold point: footing
   excavation and slab reinforcement are inspected before concrete
   (`residential-construction-guide.md`, Key hold points;
   `structural-residential.md`, Inspection and evidence gates: "certifier or
   accredited inspector sign-off"; `new-dwelling-guide.md`, Structural
   inspection gates: "Certifier or engineer"; `program-scheduling-guide.md`:
   "Inspection booking must be a separate hold point"). **The head contractor
   books it** (position `pre_pour_booking` in all three RFTs). The steel fixer
   attends and fixes defects (`steel_fixing inspection.attend`,
   `inspection.rectify`); the concreter pours only once it has passed and the
   head contractor releases the pour (concreting duties). With no head
   contractor, the PM or owner-builder books it; the position text then needs
   the PM to edit "head contractor" (open question S4).
8. **Pour.** The concreter supplies concrete to the engineer's mix, pumps,
   places, finishes and cures, and samples and tests each pour with dockets
   (`as-standards-reference.md`, AS 1012 Series; `new-dwelling-guide.md`,
   "Slab post-pour conformance ... Concrete Conformance Test"). The steel
   fixer keeps a fixer on call during the pour (general practice; excludable
   line).
9. **Stripping and propping.** Formworker strips only when the engineer or
   specification allows and back-props to the design (AS 3610 pitfalls on
   early stripping and reshoring).

When the three are let to one firm (bundle `concrete`), the positions between
them become internal; the combined RFT (slice T4) should drop the three
cross-positions between these trades and keep `set_out` and
`pre_pour_booking`.

---

## trade.demolition

- **Summary:** takes down and removes structures to a demolition work plan,
  protecting what is kept and the neighbours, and leaves the site clear.
- **House vs commercial:** houses are mostly mechanical demolition of a
  single dwelling with a short work plan. Commercial work is often staged
  strip-out next to occupied tenancies, with an engineer's sequence and
  propping, vibration monitoring and fuller waste classification
  (`commercial-construction-guide.md`, Demolition and Strip-Out).
- **Interfaces:** takes from `trade.asbestos_removal` (areas cleared before
  general demolition) and the authorities/licensed trades (services
  disconnected). Leaves for `trade.earthworks` (site clear, footings and
  buried structures removed as the documents show) and
  `trade.temporary_fencing` (hoarding where let separately).
- **Hold points:** hazardous materials survey before any demolition (client);
  asbestos clearance before demolition in each area (independent assessor);
  any engineer's sequence hold points. Notice to the regulator before
  demolition of certain structures (the seed names one; the condition is a
  lead to check).
- **Certificates:** none it signs as such; it hands over waste disposal
  records and the recycling report. Standard named: AS 2601 (Demolition of
  Structures), `verified: false`.
- **Licence (NSW):** a SafeWork NSW demolition licence for licensed
  demolition work; a NSW Fair Trading contractor licence may also be needed
  for residential building work. `verified: false`. The seeds name no licence
  for demolition itself.
- **Main Works line:** row `demolition`, lines `demolish` and `disposal`
  (exist today). `disconnect` stays with Main Works or the client.
- **Sources:** `as-standards-reference.md` (AS 2601 — The Demolition of
  Structures); `commercial-construction-guide.md` (Demolition and Strip-Out
  for Commercial Refurbishment); `renovation-guide.md` (Dilapidation and
  neighbour management); `waste-management-guide.md`.
- **Open questions:** D1. Is the demolition work plan a tender deliverable
  (design stage) or a pre-start condition only? D2. Who disconnects services
  on a builder-lets-every-trade job: the PM's electrician and plumber, or the
  demolisher arranging them?

## trade.asbestos_removal

- **Summary:** removes and disposes of the asbestos in the register under the
  licence class each material needs, until an independent assessor clears
  each area.
- **House vs commercial:** houses usually have bonded sheeting (eaves,
  wet-area linings, fences) removed in a day or two. Commercial refurbishments
  more often have friable material, enclosures, staged removal around
  occupied areas and a longer program (`commercial-construction-guide.md`:
  "Not allowing adequate program time for asbestos removal").
- **Interfaces:** takes from the client's survey and register. Leaves for
  `trade.demolition` (cleared areas), and for the remediation scope when
  asbestos is in soil (`site.contamination`, not this trade unless the
  documents say so).
- **Hold points:** independent licensed assessor's air monitoring during
  removal and clearance inspection before re-occupation or other trades
  return (`remediation-due-diligence-guide.md`, Asbestos Assessment). The
  assessor must be independent of the removalist (seed). The client engages
  the assessor (position `assessor`).
- **Certificates:** the clearance certificate is the assessor's, not the
  removalist's. The removalist hands over waste disposal and tracking records.
  Instruments the seeds name: WHS Regulations Chapter 8 (Asbestos); SafeWork
  Australia Model Codes of Practice "How to Safely Remove Asbestos" and "How
  to Manage and Control Asbestos in the Workplace". All `verified: false`.
- **Licence (NSW):** a SafeWork NSW asbestos removal licence. The higher
  class covers friable asbestos; the lower class covers bonded (non-friable)
  asbestos above a small quantity that the regulation sets; below that
  quantity no licence is needed but the work is still regulated. The assessor
  holds a separate asbestos assessor licence. `verified: false`; the seed's
  quantity is not carried here or into the RFT.
- **Main Works line:** row `demolition`, line `hazardous` (exists).
- **Sources:** `remediation-due-diligence-guide.md` (Asbestos Assessment and
  Management in Construction); `as-standards-reference.md` (AS 2601 — The
  Demolition of Structures); `commercial-construction-guide.md` (Demolition
  and Strip-Out for Commercial Refurbishment).
- **Open questions:** A1. Should asbestos found outside the register be
  priced as a provisional sum in R2, or left as a variation? A2. Is the
  assessor ever in the removalist's tender (some small jobs)? The seed says
  independent, so the RFT says client.

## trade.earthworks

- **Summary:** site preparation, erosion and sediment controls, bulk cut and
  fill and compaction, detailed excavation, trenching and spoil removal.
- **House vs commercial:** houses: a site cut, footing and service trenches,
  a few loads of spoil, often by a small excavator crew. Commercial: bulk
  excavation as an early works package, haul routes, fill lots and testing to
  the earthworks standard, rig and crane platforms; basement shoring is a
  separate retention package (`multi-residential-apartments-guide.md`, Trade
  Package Structure).
- **Interfaces:** takes from `trade.demolition` (cleared site). Leaves for
  `trade.piling` (working platform), `trade.concreting` and
  `trade.formwork` (excavated footings), `trade.stormwater_drainage` and the
  services trades (trenches), `trade.below_ground_waterproofing` (backfill
  after inspection, position on that RFT). Overlaps `trade.civil_works`
  (existing RFT has an earthworks stage).
- **Hold points:** geotechnical report in place before earthworks
  (`civil-residential.md`, Geotechnical sign-off trigger); compaction testing
  of fill before the slab (Compaction and fill quality); footing excavation
  inspected before concrete (certifier); founding inspections where the
  geotechnical engineer sets them.
- **Certificates:** none it signs; compaction test results come from the
  testing authority. Standard named: AS 3798 (Guidelines on Earthworks),
  `verified: false`.
- **Licence (NSW):** a NSW Fair Trading contractor licence for excavation
  work where it is residential building work; none named for commercial.
  `verified: false`.
- **Main Works line:** row `site`, line `clearing`; row `substructure`, line
  `excavation` (both exist).
- **Sources:** `civil-residential.md` (Site earthworks — cut and fill;
  Geotechnical sign-off trigger; Compaction and fill quality);
  `as-standards-reference.md` (AS 3798); `remediation-due-diligence-guide.md`
  (Environmental Management Plans); `residential-construction-guide.md`
  (Construction Sequence).
- **Open questions:** S2 (above). E1. Who engages the compaction testing
  authority: the client (RFT position today) or the earthworks contractor?
  E2. With Civil Works also live, should earthworks lines in the civil RFT be
  excluded automatically?

## trade.stormwater_drainage

- **Summary:** site stormwater from downpipes to the legal point of
  discharge, detention and water quality devices, and the council connection.
- **House vs commercial:** houses: a drainer laying pits and pipes, often a
  detention tank in Sydney councils, a kerb connection. Commercial: larger
  detention and water quality devices, usually inside the civil works
  package; this trade is used less often on its own there.
- **Interfaces:** takes from `trade.roof_plumbing` (downpipes to the first
  site pit, position), `trade.earthworks` (trenches). Leaves for
  `trade.paving` (grated drains in paving, position on paving) and
  `trade.landscaping` (levels to pits). Overlaps `trade.civil_works`
  (stormwater stage) and `trade.hydraulic_plumbing`.
- **Hold points:** council drainage inspection before the connection is
  buried (`civil-residential.md`, Connection to council drainage); detention
  installed and certified before the occupation certificate
  (`new-dwelling-guide.md`, Structural inspection gates: "OSD Structural
  Inspection Certificate", by the engineer;
  `trade-interfaces-coordination-guide.md`, Interface 8).
- **Certificates:** the drainer's plumbing and drainage compliance
  certificate (`mep-residential.md` names the "plumbing compliance
  certificate in NSW"); the engineer certifies the detention system.
  Standard named: AS/NZS 3500.3 (stormwater drainage), `verified: false`.
- **Licence (NSW):** a NSW Fair Trading licence for drainage work (plumbing
  and drainage licensing). `verified: false`.
- **Main Works line:** row `site`, line `drainage` (exists).
- **Sources:** `civil-residential.md` (Stormwater drainage; Connection to
  council drainage; On-site detention (OSD); Rainwater tank);
  `new-dwelling-guide.md` (Structural inspection gates);
  `trade-interfaces-coordination-guide.md` (Interface 8).
- **Open questions:** W1. Who lodges the council connection application: the
  drainer (excludable design line today) or the PM? W2. Rainwater tank
  supply and its pump: this trade, hydraulic plumbing, or a supply?

## trade.landscaping

- **Summary:** finished levels graded away from the building, soil, planting,
  turf, irrigation, low garden structures and the establishment period.
- **House vs commercial:** houses: a garden build, often after handover or as
  owner's work. Commercial and apartments: deep soil and podium planting to
  the consent, landscape architect inspections and an establishment period
  (`landscape-architecture-guide.md`).
- **Interfaces:** takes from `trade.stormwater_drainage` (pits),
  `trade.civil_works` or Main Works (retaining walls), the plumbing and
  electrical trades (irrigation water and power). Leaves for the certifier's
  final inspection (levels falling away from the building). Must not bridge
  `trade.termite_management` barriers or the cladding (soil and mulch against
  walls).
- **Hold points:** certifier inspects external levels at the final inspection
  (`trade-interfaces-coordination-guide.md`, Interface 8); landscape
  architect's set-out, soil and plant stock inspections, practical completion
  and end of establishment (`landscape-architecture-guide.md`).
- **Certificates:** none statutory; the landscape architect's completion
  sign-offs.
- **Licence (NSW):** a NSW Fair Trading contractor licence for structural
  landscaping where it is residential building work; planting alone needs
  none. `verified: false`.
- **Main Works line:** row `site`, line `external` today holds both fencing
  and landscaping. Propose splitting it: `{key: landscaping, text:
  "Landscaping, planting and garden beds the documents show"}` and
  `{key: fencing, text: "Fences and gates, including pool barriers the
  documents show"}`.
- **Sources:** `trade-interfaces-coordination-guide.md` (Interface 8:
  External landscaping, paths, and drainage before occupation);
  `landscape-architecture-guide.md`; `arborist-guide.md`;
  `residential-construction-guide.md` (Termite Management, bridging pitfall).
- **Open questions:** L1. Is the establishment and maintenance stage on by
  default, or an option the PM turns on? L2. Who protects retained trees
  during landscaping when the head contractor has left site?

## trade.fencing

- **Summary:** boundary and internal fences, gates and pool safety barriers
  on the surveyed line.
- **House vs commercial:** houses: dividing fences and pool barriers, the
  barrier being the compliance item. Commercial and apartments: perimeter and
  security fencing and gates, often with automation (seed lists "Fencing and
  gates" as an external works package).
- **Interfaces:** takes from the surveyor (boundary line), the electrical
  trade (gate motors, intercoms). Leaves for the certifier or council (pool
  barrier compliance). Different from `trade.temporary_fencing` (site
  fencing during the works).
- **Hold points:** pool barrier inspection before the pool is filled or
  used, by the certifier or council (general practice; the seed covers
  registration and periodic inspection).
- **Certificates:** the fencer signs none; the pool barrier compliance
  certificate is the certifier's or council's. Standard named: AS 1926.1-2012
  (Safety barriers for swimming pools), `verified: false`.
- **Licence (NSW):** a NSW Fair Trading contractor licence for fencing where
  it is residential building work; pool barriers may fall under the swimming
  pool building category. `verified: false`.
- **Main Works line:** row `site`, line `external` (see landscaping: propose
  the `fencing` line).
- **Sources:** `residential-construction-guide.md` (Swimming Pools and Pool
  Safety); `as-standards-reference.md` (AS 1926 Series — Swimming Pool
  Safety); `multi-residential-apartments-guide.md` (Trade Package Structure);
  `surveying-guide.md`.
- **Open questions:** F1. Should the pool barrier stage show only when Profile
  scope holds `access-egress.pool-barriers`? (Not done: no `when` on these
  stages yet.) F2. Do dividing fences under the neighbours' fencing law sit
  with this trade or the client?

## trade.paving

- **Summary:** subgrade and base, driveway, paths and paved areas to levels
  and falls, and the council crossover.
- **House vs commercial:** houses: a concrete or unit-paved driveway and the
  crossover, done late, after services. Commercial: car park pavements, kerbs
  and line marking, usually inside the civil works package.
- **Interfaces:** takes from the services and drainage trades (everything
  under the paving in first), `trade.earthworks` (subgrade levels). Leaves for
  `trade.landscaping` (edges) and the council (crossover sign-off). Overlaps
  `trade.civil_works` (pavements stage).
- **Hold points:** council road inspector before the crossover pour or
  backfill (`civil-residential.md`, Driveway crossover: Approvals); the
  certifier checks external levels at the final.
- **Certificates:** none it signs; council signs off the crossover.
- **Licence (NSW):** a NSW Fair Trading contractor licence for concreting or
  structural landscaping where it is residential building work.
  `verified: false`.
- **Main Works line:** row `site`, line `paving` (exists).
- **Sources:** `civil-residential.md` (Driveway crossover; Approvals; Kerb and
  gutter reinstatement); `trade-interfaces-coordination-guide.md` (Interface
  8).
- **Open questions:** P1. Driveway let to the concreter instead (common on
  houses): should the concreting RFT carry an excludable driveway line?

## trade.piling

- **Summary:** installs piles, piers or screw piles to founding level,
  tests them and gives the engineer the records to accept them.
- **House vs commercial:** houses: screw piles or bored piers on reactive,
  sloping or problem sites (`residential-construction-guide.md`, Foundations).
  Commercial: bored, CFA or driven piles, integrity and load testing, and
  piled basement walls (those are `substructure.basement-retention`, not this
  trade's system; the trades list builds only `substructure.piles`).
- **Interfaces:** takes from `trade.earthworks` (working platform, position),
  the surveyor (pile positions). Leaves for `trade.concreting`,
  `trade.formwork`, `trade.steel_fixing` (pile caps and ground beams,
  position). Supplies its own cages in bored piles (position on the steel
  fixing RFT).
- **Hold points:** founding inspections set by the geotechnical or
  structural engineer (`geotechnical-guide.md`, Construction and completion;
  `structural-commercial-industrial-guide.md`, Construction and completion);
  integrity testing as specified.
- **Certificates:** the piler's installation certificate and pile records;
  the engineer accepts. Standard named: AS 2159 (Piling — Design and
  Installation), `verified: false`.
- **Licence (NSW):** the seeds name none. Residential piling is building work
  under a NSW Fair Trading contractor licence; commercial piling has no trade
  licence the seeds name. Rig operators may need high-risk work licences.
  `verified: false`.
- **Main Works line:** row `substructure`, line `footings` ("Footings, piers
  or piles and the ground slab") is shared. Propose a new line `{key: piles,
  text: "Piles, piers or screw piles the documents show"}` under
  `substructure`, with `footings` reworded to "Footings and the ground slab".
- **Sources:** `as-standards-reference.md` (AS 2159 — Piling — Design and
  Installation); `commercial-construction-guide.md` (Structural Systems for
  Commercial Buildings); `geotechnical-guide.md`;
  `structural-commercial-industrial-guide.md`.
- **Open questions:** PL1. Who trims piles to cut-off level and removes
  piling spoil: the piler (lines today) or the earthworks and concreting
  trades? PL2. Should piled retaining walls get their own trade
  (`substructure.basement-retention` has no trade in the list)?

## trade.concreting

- **Summary:** supplies, places, finishes and cures concrete to footings,
  ground slab and suspended structure, with test records for each pour.
- **House vs commercial:** houses: often one "slab" firm doing form, reo and
  pour for a raft or waffle slab; pours booked around the pre-pour inspection
  and weather (`new-dwelling-guide.md`, Sequencing risks: "Concrete pour
  weather"). Commercial: one concrete structure package (formwork,
  reinforcement, supply and placement) or separate packages, post-tensioning
  as its own package (`multi-residential-apartments-guide.md`, Trade Package
  Structure); post-strip inspection of surfaces.
- **Interfaces:** see "The slab and pour sequence". Takes from
  `trade.formwork`, `trade.steel_fixing`, `trade.termite_management`,
  `trade.earthworks`, plumbing and electrical (cast-in items). Leaves for the
  frame trades (`trade.carpentry_framing`, `trade.structural_steel`,
  `trade.masonry`) after curing and the certifier's slab inspection.
- **Hold points:** pre-pour reinforcement inspection (certifier or engineer;
  booked by the head contractor); concrete conformance tests at the pour;
  slab inspection by the certifier before frame (`structural-residential.md`,
  Inspection and evidence gates); post-strip surface inspection on apartments
  (`multi-residential-apartments-guide.md`).
- **Certificates:** concrete conformance records (delivery dockets, sample
  and strength results). Standards named: AS 3600 (Concrete Structures),
  AS 1012 Series (Methods of Testing Concrete), AS 2870 (Residential slabs and
  footings), all `verified: false`.
- **Licence (NSW):** a NSW Fair Trading contractor licence for concreting
  where it is residential building work. `verified: false`.
- **Main Works line:** row `substructure`, line `footings`; row `structure`,
  line `suspended`. Propose rewording `footings` to "Concrete to footings and
  the ground slab" and adding the formwork and reinforcement lines below.
- **Sources:** `structural-residential.md` (Slab and footing types; Inspection
  and evidence gates); `new-dwelling-guide.md` (Structural inspection gates);
  `residential-construction-guide.md` (Foundations and Footing Systems;
  Construction Sequence); `as-standards-reference.md` (AS 3600; AS 1012
  Series); `multi-residential-apartments-guide.md` (Trade Package Structure;
  Quality Hold Points); `program-scheduling-guide.md`.
- **Open questions:** S1–S4 above. C1. Who supplies concrete when the
  builder buys it direct (supply, not trade)?

## trade.formwork

- **Summary:** designs where required, builds, props and strips formwork true
  to set-out, levels and the specified surface finish.
- **House vs commercial:** houses: edge forms, set-downs and void formers on
  a slab on ground; little or no design. Commercial: suspended deck and
  column/wall formwork, formwork design and certification, propping and
  back-propping across floors, surface finish classes.
- **Interfaces:** takes set-out from the head contractor. Leaves for
  `trade.steel_fixing` (checked forms), `trade.concreting` (pour), and back
  from the concreter for stripping. Edge protection and scaffold at slab
  edges are the head contractor's (`trade.scaffolding` where let), position.
- **Hold points:** the formworker's own dimension and level check before
  reinforcement; formwork certification before the pour where the formwork
  is the contractor's design (general practice; the seed covers design to
  the formwork standard); stripping only when the engineer allows.
- **Certificates:** formwork certificate for formwork it designs. Standards
  named: AS 3610.1:2010 and AS 3610.2:2010 (Formwork for Concrete),
  `verified: false`.
- **Licence (NSW):** the seeds name none. Likely a NSW Fair Trading contractor
  licence for concreting or carpentry where it is residential building work.
  `verified: false`; open question S5.
- **Main Works line:** propose `{key: formwork, text: "Formwork, edge forms
  and void formers to footings and the ground slab"}` under `substructure`
  and `{key: suspended_formwork, text: "Formwork and propping to suspended
  slabs and the concrete frame"}` under `structure`.
- **Sources:** `as-standards-reference.md` (AS 3610 Series — Formwork for
  Concrete); `residential-construction-guide.md` (Foundations and Footing
  Systems); `multi-residential-apartments-guide.md` (Trade Package
  Structure).
- **Open questions:** S1 (vapour barrier), S5 (licence).

## trade.steel_fixing

- **Summary:** supplies and fixes reinforcement to the engineer's drawings,
  with chairs and spacers for cover, ready for the pre-pour inspection.
- **House vs commercial:** houses: mesh and trench mesh in a raft or waffle
  slab, usually inside the slab firm's price. Commercial: scheduled bar,
  starter bars, cages in columns and walls, shop drawings, inspection before
  every pour.
- **Interfaces:** takes from `trade.formwork` (checked forms) and the head
  contractor (set-out). Leaves for `trade.concreting` after the inspection
  passes. `trade.piling` supplies its own pile cages.
- **Hold points:** pre-pour reinforcement inspection covering position, size,
  spacing, cover, laps and tie wire (`multi-residential-apartments-guide.md`,
  Quality Hold Points), by the certifier or engineer, booked by the head
  contractor; the fixer attends and rectifies.
- **Certificates:** the supplier's compliance certificates for the
  reinforcement. Standard named: AS/NZS 4671:2001 (Steel Reinforcing
  Materials); the seed also names third-party certification of the supplier,
  not carried into the RFT as a product rule. `verified: false`.
- **Licence (NSW):** the seeds name none. Possibly within the concreting
  category of a NSW Fair Trading contractor licence for residential work.
  `verified: false`; open question S5.
- **Main Works line:** propose `{key: reinforcement, text: "Reinforcement to
  footings and the ground slab"}` under `substructure` and
  `{key: suspended_reinforcement, text: "Reinforcement to suspended slabs and
  the concrete frame"}` under `structure`.
- **Sources:** `as-standards-reference.md` (AS/NZS 4671 — Steel Reinforcing
  Materials); `structural-residential.md` (Inspection and evidence gates);
  `multi-residential-apartments-guide.md` (Quality Hold Points for Apartment
  Construction; Trade Package Structure).
- **Open questions:** S3 (bar chairs), S4 (who books with no head
  contractor).

## trade.termite_management

- **Summary:** installs the termite management system at penetrations,
  joints, under-slab and perimeter, with certificate, durable notice and
  maintenance instructions.
- **House vs commercial:** required for new Class 1 and 10 buildings in
  termite-prone areas (the seed: most of Australia except Tasmania). Usual on
  houses and low-rise residential; less common on commercial, where the
  documents decide.
- **Interfaces:** takes from the plumbing and electrical trades (penetrations
  set in final position), `trade.formwork` (slab ready). Leaves for
  `trade.concreting` (pour after barriers), and for `trade.landscaping` and
  `trade.paving`, which must not bridge the barrier. Two visits: before the
  slab and after external levels are final.
- **Hold points:** installation before concealment (before the pour, and
  before perimeter works cover the edge). The seeds do not name a certifier
  inspection for it; the certifier relies on the installer's certificate
  (general practice).
- **Certificates:** certificate of installation and a durable notice; owner's
  maintenance details in the building manual. Standards and code named:
  AS 3660.1-2014 (New building work), AS 3660.2-2017 (existing buildings),
  NCC Volume Two Part 3.1.3. All `verified: false`.
- **Licence (NSW):** a NSW EPA pest management licence covering timber pests.
  `verified: false`; the seed says "licensed pest controller".
- **Main Works line:** row `substructure`, line `termite` (exists).
- **Sources:** `residential-construction-guide.md` (Termite Management).
- **Open questions:** T1. Is termite management under the slab the
  termite trade's, or the slab crew's using supplied collars?

## trade.below_ground_waterproofing

- **Summary:** basement tanking, membranes, waterstops and drained cavities
  that keep ground moisture and groundwater out, inspected before cover.
- **House vs commercial:** houses: a damp-proof membrane under the slab and
  waterproofing to a retaining or basement wall. Commercial: tanking or a
  drained cavity system below groundwater, waterstops at every construction
  joint; failures are very costly to fix (`commercial-construction-guide.md`,
  Waterproofing).
- **Interfaces:** takes from `trade.concreting` and `trade.formwork`
  (substrate, waterstops cast in), and from dewatering by the excavation or
  retention contractor. Leaves for `trade.earthworks` (backfill after
  inspection) and the hydraulic and electrical trades (sump pumps). Not
  `trade.waterproofing` (wet areas, balconies, roofs above ground).
- **Hold points:** substrate inspection before the membrane; membrane
  inspection before backfill or cover; tests as specified
  (`multi-residential-apartments-guide.md`, Quality Hold Points;
  `commercial-construction-guide.md`).
- **Certificates:** the installer's waterproofing certificate and warranty.
  The seeds name no standard for below-ground waterproofing (AS 4654.1/.2 are
  for external above-ground use; AS 4678 covers waterstops in retaining
  walls). `verified: false`.
- **Licence (NSW):** a NSW Fair Trading contractor licence for waterproofing
  where it is residential building work; the commercial seed notes
  state-specific waterproofing licensing. `verified: false`.
- **Main Works line:** row `substructure`, line `below_ground` (exists).
- **Sources:** `commercial-construction-guide.md` (Waterproofing in
  Commercial Buildings); `multi-residential-apartments-guide.md` (Quality
  Hold Points); `as-standards-reference.md` (AS 4678 — Earth-Retaining
  Structures); `ncc-reference-guide.md`.
- **Open questions:** S1 (vapour barrier). B1. Who supplies waterstops: this
  trade or the steel fixer or formworker who places them?

---

## Proposed Main Works lines (for slice T2; main-works.yaml not edited)

| Row | Line | Text | Taken by |
|---|---|---|---|
| demolition | `demolish`, `disposal` (exist) | — | trade.demolition |
| demolition | `hazardous` (exists) | — | trade.asbestos_removal |
| site | `clearing` (exists) | — | trade.earthworks |
| site | `drainage` (exists) | — | trade.stormwater_drainage |
| site | `paving` (exists) | — | trade.paving |
| site | `landscaping` (new, replaces `external`) | Landscaping, planting and garden beds the documents show | trade.landscaping |
| site | `fencing` (new, replaces `external`) | Fences and gates, including pool barriers the documents show | trade.fencing |
| substructure | `excavation` (exists) | — | trade.earthworks |
| substructure | `piles` (new) | Piles, piers or screw piles the documents show | trade.piling |
| substructure | `footings` (reword) | Concrete to footings and the ground slab | trade.concreting |
| substructure | `formwork` (new) | Formwork, edge forms and void formers to footings and the ground slab | trade.formwork |
| substructure | `reinforcement` (new) | Reinforcement to footings and the ground slab | trade.steel_fixing |
| substructure | `below_ground` (exists) | — | trade.below_ground_waterproofing |
| substructure | `termite` (exists) | — | trade.termite_management |
| structure | `suspended` (exists) | — | trade.concreting |
| structure | `suspended_formwork` (new) | Formwork and propping to suspended slabs and the concrete frame | trade.formwork |
| structure | `suspended_reinforcement` (new) | Reinforcement to suspended slabs and the concrete frame | trade.steel_fixing |

The `site` row's `separate: trade.civil_works` today carves the whole row
out; with line-level carve-out, Civil Works and these site trades will both
claim `drainage`, `paving` and `clearing`. T2 needs a rule for two live
packages naming the same line (open question O1).

## Open questions for the owner (collected)

- **S1** Vapour barrier under a slab on ground: formwork trade (default in
  the RFT) or below-ground waterproofing?
- **S2** Footing excavation: earthworks (default) or the slab crew?
- **S3** Bar chairs, spacers and tie wire: steel fixer (default) or the
  concreter or formworker?
- **S4** With no head contractor, should `set_out` and `pre_pour_booking`
  read "by the client" or "by the project manager"? Today they say head
  contractor.
- **S5** NSW licence category for formwork and steel fixing.
- **O1** Civil Works and a separate site trade both live: which takes the
  shared Main Works lines?
- D1, D2, A1, A2, E1, E2, W1, W2, L1, L2, F1, F2, P1, PL1, PL2, C1, T1, B1 as
  listed under each trade.

## Not sourced from the seeds (general practice, written without numbers)

- Who supplies bar chairs (S3), who lays the vapour barrier (S1), the
  fixer on call during the pour, formwork certification before the pour,
  pile trimming and spoil.
- Every NSW licence category above except asbestos (seed names the classes)
  and waterproofing (seed says licensed waterproofer is required in NSW for
  wet areas).
- Pool barrier inspection before filling.
