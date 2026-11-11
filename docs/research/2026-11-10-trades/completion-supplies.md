# Trades research: completion and supplies

Date: 10 November 2026. Slice T3 of `docs/plans/2026-11-10-trades-catalogue.md`.
Status: draft for owner review. Agent research from the seed guides only;
nothing here has been checked against an instrument. Every standard,
NCC part and licence named below is `verified: false`.

Group: `trade.builders_clean`, `trade.waste_bins`, `trade.temporary_fencing`,
`supplier.stone_benchtops`, `supplier.door_hardware`, `supplier.sanitaryware`,
`supplier.appliances`.

RFT files written (all `status: draft`, same shape as
`knowledge/rfp/trade-structural-steel.yaml`):

| Trade id | RFT file |
|---|---|
| trade.builders_clean | `knowledge/rfp/trade-builders-clean.yaml` |
| trade.waste_bins | `knowledge/rfp/trade-waste-bins.yaml` |
| trade.temporary_fencing | `knowledge/rfp/trade-temporary-fencing.yaml` |
| supplier.stone_benchtops | `knowledge/rfp/supplier-stone-benchtops.yaml` |
| supplier.door_hardware | `knowledge/rfp/supplier-door-hardware.yaml` |
| supplier.sanitaryware | `knowledge/rfp/supplier-sanitaryware.yaml` |
| supplier.appliances | `knowledge/rfp/supplier-appliances.yaml` |

The three completion trades build no building system; their Main Works
lines all sit in the `preliminaries` row of `knowledge/rfp/main-works.yaml`.
Today that row holds `establishment` (which includes fencing), `waste`
(cleaning and waste together) and `handover` (which includes the final
clean). Each of those lines bundles two things, so a line-level carve-out
(plan section 3, slice T2) needs them split. The proposals below do that.

---

## trade.builders_clean — Builder's clean

**Summary.** Progressive rough cleans and the final clean that makes the
building ready for the certifier's final inspection and handover.

**House vs commercial.** On a house it is often one final clean, sometimes
done by the builder's own labour; the RFT lets the PM exclude the
progressive stage. On commercial and apartments it is usually a separate
cleaning contractor working floor by floor or unit by unit, with a
touch-up clean after defects and before each sectional handover; external
glazing above safe reach needs scaffold, swing stage or EWP access, which
stays with the head contractor in the RFT.

**Interfaces.**
- Takes from: every finishing trade (`trade.painting`, `trade.tiling`,
  `trade.floor_coverings`, `trade.timber_floors`, `trade.joinery_kitchens`,
  `trade.shower_screens`), who must finish and remove their own protection
  first. Uses the site bins (`trade.waste_bins`) and temporary services.
- Leaves for: the handover inspection, defects rectification, and the
  touch-up clean.
- Positions in the RFT: bins by head contractor or waste contractor;
  water and power from site temporary services; height access by the head
  contractor; sealing of stone and timber by the trade that laid them.

**Hold points and inspections.** No statutory hold point. The final clean
precedes the certifier's final inspection and the PC inspection (the
residential guide lists "the site is cleaned, temporary works removed" as a
PC criterion). The joint area walk with the head contractor is a
contractual sign-off, not an authority one.

**Certificates.** None.

**Licence (NSW).** No NSW Fair Trading contractor licence category for
cleaning is named in the seeds; we believe none is needed for cleaning
alone. `verified: false`.

**Main Works line it carves out.** Row `preliminaries`. Existing lines
`waste` ("Cleaning and waste removal throughout") and `handover` ("Final
clean, commissioning records, manuals, warranties and as-built records")
both hold part of it. Proposed:
- new `{key: final_clean, text: "Progressive cleans and the final builder's clean", separate: trade.builders_clean}`
- `waste` reworded to "Waste removal throughout" (see waste bins below)
- `handover` reworded to "Commissioning records, manuals, warranties and as-built records"

**Sources.**
- `residential-construction-guide.md` — "Construction Sequence and
  Programming for Residential Projects" (step 12, final clean) and the
  practical completion criteria and handover documentation lists.
- `trade-interfaces-coordination-guide.md` — Interface 6 (painting before
  flooring: protection and clean-up between trades).
- `cost-management-principles.md` — "1. The residential cost plan in HIA
  Schedule of Allowances format" (preliminaries: site set-up, temporary
  services).
- `finishes-residential.md` — Joinery / Kitchen cabinetry (undamaged
  benchtop before PC).

**Open questions.**
1. Should the final clean be a separate line in Main Works preliminaries,
   or stay inside `handover`?
2. Is window cleaning at height in this trade (with its own access) or
   with the head contractor's access, as drafted?

---

## trade.waste_bins — Waste and skip bins

**Summary.** Supply, exchange and lawful disposal of site skip bins, with
the dockets and recycling reports the construction waste management plan
asks for.

**House vs commercial.** House: a few general skips and perhaps a
separate bin for bricks, concrete or green waste; road placement often
needs a council permit. Commercial and apartments: more streams separated
(the waste consultant's construction waste plan sets recycling targets and
reporting), sometimes hook-lift or compactor bins, and monthly waste
tracking reports to the consent conditions.

**Interfaces.**
- Takes from: every trade (they load the bins), the head contractor (who
  orders exchanges and gets the road permit), the waste consultant (plan
  and targets; `consultant.waste`).
- Excludes: asbestos and hazardous materials (licensed removalist under
  `trade.demolition` or the head contractor), and demolition waste where
  `trade.demolition` is let separately.
- Leaves for: the waste consultant's construction-stage review and the
  occupation certificate (waste tracking and recycling reports).

**Hold points and inspections.** None statutory for bins. Records
(dockets, weights, recycling reports) are evidence the consent conditions
and the waste management plan may require before the occupation
certificate; set by the consent authority through the plan.

**Certificates.** None. It provides disposal dockets and reports.

**Licence (NSW).** No building licence. Transporting and receiving some
waste types needs an environment protection licence from the NSW EPA, and
the receiving facility must be licensed for the waste; the seeds do not
cover this. `verified: false`.

**Main Works line it carves out.** Row `preliminaries`, existing line
`waste` ("Cleaning and waste removal throughout"). Proposed:
- new `{key: waste_bins, text: "Skip bins and lawful disposal of site waste", separate: trade.waste_bins}`
- `waste` then removed (cleaning goes to `final_clean`, disposal to
  `waste_bins`), or kept as "Gathering waste from the work areas" for the
  head contractor's own labour.

**Sources.**
- `waste-management-guide.md` — "Purpose and professional boundary" (the
  builder implements construction waste), "Documentation and tender"
  (recycling targets and reporting), "Construction and completion" (waste
  tracking and recycling reports).
- `setup-and-commission-guide.md` — "Pre-site mobilisation — Builder"
  (waste management plan drafted before site possession).
- `role-builder.md` — "Setup checklist — builder mobilisation".
- `commercial-construction-guide.md` — asbestos audit before demolition.

**Open questions.**
1. Who holds the council permit for bins on the road: the head contractor
   (as drafted) or the bin supplier?
2. Keep a residual `waste` line in Main Works for the head contractor's
   own labour gathering waste?

---

## trade.temporary_fencing — Temporary fencing and hoarding

**Summary.** Supply, install, maintain, move and remove site fencing,
gates and screening; on street-front and commercial sites, hoarding and
overhead protection over the footpath.

**House vs commercial.** House: hire panels on bases with a gate and shade
cloth, installed at site establishment and removed at completion.
Commercial and apartments: solid hoarding on frontages, overhead
protection (gantry) where work is above the footpath, council approval for
hoarding on public land, engineered design for wind, lighting and
signage; the RFT carries a design stage and a hoarding stage the PM can
exclude for a house.

**Interfaces.**
- Takes from: the head contractor (fence line, construction management
  plan, council approvals, traffic management plan, statutory signage).
- Coordinates with: `trade.civil_works` and `trade.earthworks` (erosion
  and sediment controls on the fence line; fence moves for access),
  `trade.demolition` (first trade inside the fence).
- Leaves for: permanent fencing and pool barriers by the landscape or
  external works trades.

**Hold points and inspections.** None from the engineer or certifier for
panel fencing. Hoarding and gantries on public land are set by the
council approval and may need an engineer's certificate; the seeds do not
name the requirement. `verified: false`.

**Certificates.** Engineer's certification of hoardings and gantries
where needed (structural). No standard named in the seeds.

**Licence (NSW).** The seeds do not name a licence for temporary fencing.
Hoarding and gantry erection may involve work at height and scaffolding
licences under work health and safety law. `verified: false`.

**Main Works line it carves out.** Row `preliminaries`, existing line
`establishment` ("Site establishment, amenities, fencing and signage").
Proposed:
- new `{key: fencing, text: "Temporary fencing, gates and hoarding", separate: trade.temporary_fencing}`
- `establishment` reworded to "Site establishment, amenities and signage"

**Sources.**
- `residential-construction-guide.md` — "Construction Sequence and
  Programming for Residential Projects" (step 1, temporary fencing; step
  12, temporary works removed).
- `cost-management-principles.md` — "1. The residential cost plan in HIA
  Schedule of Allowances format" (preliminaries include hoarding).
- `setup-and-commission-guide.md` — "Pre-site mobilisation — Builder"
  (construction management plan and traffic management plan).
- `traffic-transport-guide.md` — "Construction and completion" (road
  occupancy coordination).
- `procurement-quoting-guide.md` — trade scope table, External works
  (permanent fencing).

**Open questions.**
1. Should hoarding and gantries be a separate trade from panel fencing
   on commercial jobs? They are drafted here as one stage the PM can
   exclude.
2. Should erosion and sediment controls sit with this trade on a house?
   Drafted with the head contractor or civil contractor.

---

## supplier.stone_benchtops — Stone benchtops

**Summary.** Templating, fabrication, delivery and, usually, installation
of stone benchtops, vanity tops and splashbacks after the joinery
carcasses are in.

**Who installs.** The supplier usually templates on site after the joiner
installs and levels the carcasses, then fabricates and installs. The RFT
keeps installation as a stage the PM can exclude for a supply-only
tender.

**House vs commercial.** House: one kitchen, a few vanities, client
selection of slabs. Apartments: many repeated tops; the apartment guide
lists stone benchtops as a finishes package procured weeks before the
trade starts, and stone as a long-lead item.

**Interfaces.**
- Takes from: `trade.joinery_kitchens` (carcasses installed level and
  strong enough to carry the stone), the appliance and tapware schedules
  (`supplier.appliances`, `supplier.sanitaryware`) for cut-outs.
- Leaves for: `trade.hydraulic_plumbing` (sink and tapware connection),
  `trade.gas_fitting` and `trade.electrical` (cooktop connection),
  `trade.tiling` (tiled splashbacks where not stone).

**Hold points and inspections.** No statutory hold point. Kitchen
cabinetry and the undamaged benchtop are checked at the fixing inspection
and before PC (finishes guide). The structural guide notes large stone
benchtops are a load case that can trigger engineer involvement for the
framing (AS 1684, `verified: false`).

**Certificates.** None statutory. Material and installation warranties.

**Safety.** Cutting stone releases respirable crystalline silica. The RFT
says in plain words: cut and polish only under the dust controls
workplace safety law requires, no dry cutting on site. General knowledge,
not in the seeds: Australia prohibited the manufacture, supply and
installation of engineered stone benchtops from 1 July 2024; the seeds
(finishes guide, procurement quoting guide) still list engineered stone as
an option. `verified: false`. No material is selected in the RFT.

**Licence (NSW).** The seeds name none. Installing benchtops may sit
inside a NSW contractor licence for joinery or general building work when
the contract value passes the licensing threshold; not stated in the
seeds. `verified: false`.

**Main Works line it carves out.** Row `joinery`, existing line `kitchen`
("Kitchen joinery, benchtops and appliances installation"). Proposed:
- new `{key: stone_benchtops, text: "Stone benchtops, vanity tops and splashbacks", separate: supplier.stone_benchtops}`
- `kitchen` reworded to "Kitchen joinery and appliances installation"

**Sources.**
- `finishes-residential.md` — "Joinery / Kitchen cabinetry" (benchtop
  types; stone is heavy, confirm cabinet support; fixing inspection).
- `procurement-quoting-guide.md` — trade scope table, Kitchen joinery
  (benchtop material, appliance integration).
- `multi-residential-apartments-guide.md` — "Trade Package Structure"
  (stone benchtops as a finishes package; stone as a long-lead item).
- `as-standards-reference.md` — AS 1684 note on large stone benchtops.
- `building-remediation-rectification-guide.md` — silica exposure
  controls.

**Open questions.**
1. Confirm the engineered stone prohibition and whether the RFT should
   say so in words (it now says only "dust controls workplace safety law
   requires").
2. Is the stone supplier normally let by the joiner (as a sub-supply) or
   directly by the builder? This affects whether the joinery RFT's
   "Cabinets, benchtops and splashbacks" line should drop benchtops when
   this supply is a live package.

---

## supplier.door_hardware — Door hardware

**Summary.** A door-by-door hardware schedule, keying plan and supply of
locks, latches, hinges, closers, stops and exit hardware, packed and
labelled per door for the fix-out carpenter.

**Who installs.** The fix-out carpenter (`trade.carpentry_fixout`) fits
the hardware; the RFT's installation stage is excludable.

**House vs commercial.** House: a short schedule of passage, privacy and
entry sets, often an owner PC selection ("ironmongery / door furniture"
in the PC list). Commercial and apartments: fire doors whose hardware
must belong to the tested doorset, exit hardware on egress doors,
accessible hardware, master key systems, electric strikes and access
control interfaces.

**Interfaces.**
- Takes from: the architect's door schedule, the fire engineer and fire
  safety reports, the access consultant.
- Leaves for: `trade.carpentry_fixout` (fitting), `trade.electrical` and
  `trade.data_security` (power and access control for electric locks),
  the fire door supplier (tested hardware set). Joinery handles stay with
  `trade.joinery_kitchens`.

**Hold points and inspections.** Fire doors and their hardware are part
of the fire safety measures inspected before the occupation certificate
(certifier); exit door hardware is checked against the fire and access
reports. The access consultant reviews hardware substitutions after
contract.

**Certificates.** Compliance evidence for hardware on fire doors,
belonging to the tested doorset under AS 1905.1 (fire-resistant doorsets),
`verified: false`. Accessible hardware to the AS 1428 series,
`verified: false`. Neither is named in the RFT text.

**Licence (NSW).** The seeds name none. A NSW licence for locksmithing
(security industry) may apply to master keying and electronic locks;
fitting hardware sits with the carpenter's licence. `verified: false`.

**Main Works line it carves out.** Row `interiors`, existing line `doors`
("Internal doors and hardware"). Fire doors' hardware also sits in row
`fire_passive` line `fire_doors` ("Fire doors and their hardware").
Proposed:
- new `{key: door_hardware, text: "Supply of door hardware and the keying system", separate: supplier.door_hardware}` under `interiors`
- `doors` reworded to "Internal doors and fitting of hardware"
- `fire_doors` unchanged (the tested doorset stays together; this
  supplier coordinates with it)

**Sources.**
- `as-standards-reference.md` — "AS 1905.1 — Components for the
  Protection of Openings in Fire-Resistant Walls — Part 1: Fire-Resistant
  Doorsets" (non-rated hardware invalidates the FRL); "AS 1428 Series —
  Design for Access and Mobility".
- `fire-life-safety-guide.md` — "Services by stage" (doors, hardware,
  smoke separation coordination).
- `access-consultant-guide.md` — "Novation and post-contract design"
  (review substitutions affecting hardware).
- `ict-av-security-guide.md` — doors, locks, hardware and intercom
  interfaces.
- `cost-management-principles.md` — "3. PC sums" (ironmongery / door
  furniture) and "4. Owner-supplied items" (joinery hardware).
- `residential-construction-guide.md` — construction sequence step 10
  (internal doors in second fix).

**Open questions.**
1. Does door hardware include external entry doors and garage doors, or
   only internal and fire doors? Drafted as all doors in the schedule.
2. Should keying and master keying be a separate locksmith line?

---

## supplier.sanitaryware — Sanitaryware and tapware

**Summary.** Supply of toilets, basins, baths, sinks, showers and tapware
to the fixture schedule, with plumbing product certification and water
ratings, delivered in two drops: concealed parts before rough-in,
fixtures before fit-off.

**Who installs.** The plumber (`trade.hydraulic_plumbing`) fits and
connects; the RFT's installation stage is excludable, and if kept says
"by a licensed plumber".

**House vs commercial.** House: mostly PC items the owner selects
(sanitary ware and tapware are named PC items; owners sometimes bring
their own, which should be treated as owner-supplied). Commercial:
schedules by the hydraulic consultant, accessible fixtures to the access
report, commercial-grade and sensor tapware, larger quantities.

**Interfaces.**
- Takes from: the hydraulic consultant's fixture schedule, the architect,
  the access consultant.
- Leaves for: `trade.hydraulic_plumbing` (rough-in set-outs, fitting),
  `trade.waterproofing` (baths, shower bases, floor wastes),
  `trade.shower_screens`, `trade.joinery_kitchens` (vanities),
  `supplier.stone_benchtops` (basin and sink cut-outs).

**Hold points and inspections.** Plumbing rough-in pressure test and
inspection before lining (Interface 3: licensed trade and certifier at
the stages the CC defines); fit-off checked by the plumber's compliance
certificate. These belong to the plumber, not the supplier.

**Certificates.** The supplier provides product evidence: WaterMark
certification (plumbing products must be certified under the Plumbing
Code of Australia, NCC 2022 Volume 3) and WELS water efficiency ratings.
Both `verified: false`. The plumbing compliance certificate is the
plumber's.

**Licence (NSW).** None to supply. Fitting needs a NSW contractor licence
for plumbing (Fair Trading). `verified: false`.

**Main Works line it carves out.** Row `hydraulic`, existing line
`fixtures` ("Fitting of sanitary fixtures and tapware") holds the fitting,
which stays with the plumber. Supply has no line today. Proposed:
- new `{key: fixtures_supply, text: "Supply of sanitary fixtures and tapware", separate: supplier.sanitaryware}` under `hydraulic`
- `fixtures` unchanged

**Sources.**
- `ncc-reference-guide.md` — Plumbing Code of Australia section
  (WaterMark-certified products; NCC 2022 Volume 3 Parts B1, B2).
- `as-standards-reference.md` — "AS/NZS 3500 Series — Plumbing and
  Drainage" (non-WaterMark products as a common failure).
- `new-dwelling-guide.md` — water fixtures, WELS rating minima.
- `cost-management-principles.md` — "3. PC sums" (sanitary ware,
  tapware) and "4. Owner-supplied items".
- `procurement-quoting-guide.md` — trade scope table (tapware install
  ownership as a recurring dispute).
- `trade-interfaces-coordination-guide.md` — "Interface 3: Kitchen and
  bathroom services rough-in before wall lining".
- `access-consultant-guide.md` — sanitary fixture substitutions.

**Open questions.**
1. Should the hydraulic row's `fixtures` line say "supplied by others"
   automatically when this supply is a live package?
2. Are basin and kitchen sinks in this supply or with the joinery and
   stone (drafted here)?

---

## supplier.appliances — Appliances

**Summary.** Supply of cooking, dishwashing, laundry and other fixed
appliances to the schedule, with installation specifications early
enough for joinery, services and benchtop cut-outs.

**Who installs and connects.** The electrician (power), the plumber
(water and drainage), a licensed gasfitter (gas appliances) and the
joiner (integrated appliances and panels). The RFT's installation stage
is excludable and, if kept, says "by licensed trades".

**House vs commercial.** House: often owner-supplied or a PC item;
whitegoods (fridge, washer, dryer) usually owner-supplied. Apartments:
bulk supply per unit type as a finishes package, a long-lead item that is
volatile by brand; delivery at fit-off unit by unit. Commercial kitchens
are not covered by this supply (Type B gas appliances are flagged as
outside residential scope in the seeds).

**Interfaces.**
- Leaves for: `trade.joinery_kitchens` (housings, panels, integration;
  the joinery RFT already holds "Installation of appliances the documents
  list"), `trade.electrical`, `trade.hydraulic_plumbing`,
  `trade.gas_fitting`, `trade.mechanical` (rangehood ducting),
  `supplier.stone_benchtops` (cooktop and sink cut-outs).
- Takes from: the client's selections and the appliance schedule.

**Hold points and inspections.** None for supply. The connecting trades
sign their own compliance (electrical safety certificate, gas compliance
for Type A appliances).

**Certificates.** None for the supplier; manuals and warranties for the
building manual. Gas appliances installed to AS/NZS 5601.1,
`verified: false`, by the gasfitter.

**Licence (NSW).** None to supply. Connection needs NSW licences for
electrical work, plumbing and gasfitting (Fair Trading). `verified: false`.

**Main Works line it carves out.** Row `joinery`, existing line `kitchen`
("Kitchen joinery, benchtops and appliances installation") holds the
installation. Supply has no line today. Proposed:
- new `{key: appliances_supply, text: "Supply of the appliances the documents list", separate: supplier.appliances}` under `joinery`
- `kitchen` installation wording unchanged apart from the stone split
  above

**Sources.**
- `program-scheduling-guide.md` — long-lead table (appliances: confirm
  before joinery and services final fix) and "Services final fix".
- `cost-management-principles.md` — "3. PC sums" (kitchen appliances,
  often owner-supplied) and "4. Owner-supplied items" (whitegoods).
- `procurement-quoting-guide.md` — trade scope table, Kitchen joinery
  (appliance integration).
- `mep-residential.md` — Type A vs Type B appliances (licensed
  plumber/gasfitter).
- `multi-residential-apartments-guide.md` — "Trade Package Structure"
  (appliances as a finishes package).
- `residential-construction-guide.md` — handover documentation (appliance
  manuals, warranty certificates).
- `trade-interfaces-coordination-guide.md` — Interface 3 (outlet
  positions at rough-in).

**Open questions.**
1. Should the RFT say freestanding whitegoods are excluded unless listed
   (drafted as a position held by the client)?
2. Should the Main Works `joinery` row keep appliance installation, or
   should installation move to the services rows?

---

## Proposed Main Works lines (summary; main-works.yaml not edited)

| Row | New line | Text | separate | Rewords |
|---|---|---|---|---|
| preliminaries | `final_clean` | Progressive cleans and the final builder's clean | trade.builders_clean | `handover` drops "Final clean" |
| preliminaries | `waste_bins` | Skip bins and lawful disposal of site waste | trade.waste_bins | `waste` removed or kept for own labour |
| preliminaries | `fencing` | Temporary fencing, gates and hoarding | trade.temporary_fencing | `establishment` drops "fencing" |
| joinery | `stone_benchtops` | Stone benchtops, vanity tops and splashbacks | supplier.stone_benchtops | `kitchen` drops "benchtops" |
| interiors | `door_hardware` | Supply of door hardware and the keying system | supplier.door_hardware | `doors` becomes "Internal doors and fitting of hardware" |
| hydraulic | `fixtures_supply` | Supply of sanitary fixtures and tapware | supplier.sanitaryware | none |
| joinery | `appliances_supply` | Supply of the appliances the documents list | supplier.appliances | none |

## Could not source from the seeds

- Licences for waste transport (NSW EPA), temporary fencing, hoarding,
  locksmithing and stone fabrication: not in the seeds.
- The engineered stone prohibition (from 1 July 2024): general knowledge;
  the seeds predate it and still list engineered stone.
- Council hoarding approval and engineer certification requirements:
  not in the seeds.
