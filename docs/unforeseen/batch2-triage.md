# Batch 2 triage: unforeseen circumstances sorted and mapped

Status: draft for owner review. Nothing here is verified, and no knowledge record has been written or changed. This is Pass A for the owner's second list of 1,000 rows (dataset batch 2, rows B2-0001 to B2-1000): each row has been sorted by kind and category, mapped to the SiteWise building systems, checked for duplicates within the batch and checked against the failure modes, unforeseen conditions and interfaces already in the knowledge base. Pass B will turn the usable rows into records.

The sorted rows are in `data/unforeseen/batch2_triage.csv`. `python tools/check_triage.py --dataset batch2` checks the file. The owner's top-level system names are in `knowledge/works/system_terms.yaml`.

## How to read this

- Kind is the owner's own label, translated: latent condition becomes unforeseen condition; the other three keep their meaning. I changed the kind on 43 rows where the text plainly describes something else (each has a note in the file, and the later section lists the ones I am least sure of).
- Category (one of the nine groups) is given only to unforeseen conditions and process, authority, supply or weather rows. Design and workmanship rows become failure modes and carry none.
- Side A and side B are the two things the row is about. Each is the most specific SiteWise system I could justify from the element and trigger named in the row, or a "who" (occupant, authority, utility, neighbour, weather, programme, contract). Where the row text does not narrow a side beyond the owner's top-level name (for example "Hydraulic services"), the top-level system is used rather than a guess; those rows say so in the note. This happens on roughly 200 second sides.
- Cluster is the knowledge area that owns the record. It follows side A when that is a physical system, otherwise side B.

## Counts

**By kind** (the owner's original labels are in brackets)

| Kind | Rows |
| - | - |
| Unforeseen condition (532) | 489 |
| Design or coordination error (279) | 321 |
| Workmanship defect (114) | 115 |
| Process, authority, supply or weather (75) | 75 |

**By category** (unforeseen conditions and process rows only; 436 design and workmanship rows carry none)

| Category | Rows |
| - | - |
| Concealed services and earlier work | 161 |
| Existing structure | 79 |
| Authorities and utilities | 76 |
| Third parties and occupation | 67 |
| Existing systems on test | 58 |
| Hazardous materials | 45 |
| Ground and site | 37 |
| Supply and site operations | 29 |
| Design and scope | 12 |

**By cluster**

| Cluster | Rows |
| - | - |
| Services, wet and air (hydraulic, mechanical) | 259 |
| Structure (site, substructure, structure) | 239 |
| Envelope (envelope, interiors) | 203 |
| Electrical and comms (electrical, comms, lifts) | 172 |
| Fire (fire, egress and access) | 113 |
| Delivery (no physical system on either side) | 14 |

**By work type** (a row can have several; "replacement or upgrade of an existing system" is counted as refurbishment)

| Work type | Rows |
| - | - |
| Refurbishment, fit-out or upgrade | 596 |
| New build | 266 |
| Extension | 135 |
| Remediation of defects | 126 |

**By building category** (a row can have several): commercial 700, residential 286, institution 216, industrial 134. Institution is new in this batch; the checker now accepts it.

**Existing building**

| Existing building | Rows |
| - | - |
| Yes | 868 |
| No | 132 |

Of the 868 existing-building rows, 462 are unforeseen conditions, 261 design or coordination errors, 87 workmanship defects and 58 process rows. A row is "no" only when it is purely new build and nothing in the text concerns existing fabric, an occupant or earlier work.

## Duplicates and rows already covered

- Duplicate groups within the batch: 62 groups covering 128 rows (58 pairs and 4 groups of three). Most are a long, specific row in the first 940 rows and a short restatement of it in the last 60 (B2-0942 to B2-1000); 48 of those 59 tail rows duplicate an earlier row.
  - B2-0196, B2-0996: The vision panel cut into an existing fire door is larger than the size the door was te...
  - B2-0305, B2-0953: Voltage drop fails on the longest final subcircuit once the real route through the exis...
  - B2-0117, B2-0947: The floor-waste flange in an existing bathroom will not accept the lap of the replaceme...
  - B2-0192, B2-0936: Make-up air for a new smoke exhaust has no path into the floor once the facade is sealed.
  - B2-0180, B2-0991: Hose-reel coverage misses a room created by the fit-out partitions.
  - B2-0247, B2-0959: Plant noise at the boundary exceeds the limit once the replacement units are selected.
  - B2-0408, B2-0964: Rainwater reuse is cross-connected to the potable supply.
  - B2-0231, B2-0957: The outside-air intake on an existing plant deck is closer to the kitchen exhaust than...
  - B2-0384, B2-0774, B2-0963: A trade-waste arrestor will not fit the level or the access the authority requires.
  - B2-0104, B2-0977: Photovoltaic feet are a penetration pattern the existing membrane warranty will not acc...
  - B2-0207, B2-0998: A warden phone cannot be cabled back to the existing fire panel because the riser is full.
  - B2-0155, B2-0965: The kicker joint at the base of a basement wall leaks after the first wet week.
  - B2-0154, B2-0966: A pile head penetrates the blinding membrane of a new basement with no collar.
  - B2-0383, B2-0962: Tempered water cannot reach the new outlet within the pipe length the valve allows.
  - B2-0178, B2-0995: The kitchen hood suppression does not cover the appliances the tenant actually installs.
  - B2-0377, B2-0961: A new soil stack clashes with an existing beam on the only available riser line.
  - B2-0217, B2-0997: A gaseous-suppression room in an existing building is not tight enough to hold the desi...
  - B2-0219, B2-0451: The firefighter lift is not on the power circuit that the fire strategy keeps alive.
  - B2-0323, B2-0955: Starting the fire pump drops the voltage on the rest of the existing board.
  - B2-0351, B2-0982: Equipment in a hazardous area is not rated for the zone the process actually creates.
  - B2-0234, B2-0514, B2-0956: Condensate from a new fan coil in a fit-out cannot fall to a lawful discharge.
  - B2-0317, B2-0512: A luminaire, a sprinkler and a diffuser cannot share the existing tile module.
  - B2-0187, B2-0988: A new cable tray passes through a fire-isolated stair the strategy treats as a shaft wi...
  - B2-0742, B2-0978: A hearing loop does not work because of the existing slab reinforcement.
  - B2-0204, B2-0949: Emergency lights spaced for an open plan fail once the fit-out rooms are built.
  - B2-0411, B2-0773: Trade-waste quality from the new process is outside the agreement with the authority.
  - B2-0309, B2-0951: The existing switchroom is too small for the replacement board and the clearance in fro...
  - B2-0215, B2-1000: A fire shutter drops onto a sloping existing floor and leaves a gap at one end.
  - B2-0179, B2-0519, B2-0993: A late services bulkhead hides the exit sign the strategy relies on.
  - B2-0167, B2-0913: A new hydraulic stack through an existing sole-occupancy wall has no fire collar.
  - B2-0428, B2-0981: The chemical-drain material is wrong for the chemical the process actually uses.
  - B2-0212, B2-0999: Smoke seals are painted over during a repaint and no longer seat.
  - B2-0118, B2-0148, B2-0946: A 1970s bathroom slab was cast without a set-down, so the new membrane has nowhere to f...
  - B2-0235, B2-0958: The tenant's kitchen exhaust has no riser in the existing shaft.
  - B2-0073, B2-0944: A new balcony door threshold ends up below the external tiling once the falls are rebui...
  - B2-0200, B2-0987: Racking installed after occupation is taller than the sprinkler design the warehouse wa...
  - B2-0172, B2-0990: A smoke door will not close because the existing floor is out of level and the leaf binds.
  - B2-0072, B2-0943: Weep holes in the existing veneer were rendered over during a previous paint cycle, and...
  - B2-0202, B2-0986: A cable tray through a rated wall is filled beyond the percentage the seal was tested for.
  - B2-0597, B2-0766: The flood planning level changes after the extension floor has been documented.
  - B2-0177, B2-0992: A detection zone crosses two compartments the fire strategy keeps separate.
  - B2-0271, B2-0445: A new plant room covers the lift-shaft vent.
  - B2-0439, B2-0950: The existing lift shaft is out of plumb beyond the guide-rail adjustment of the replace...
  - B2-0373, B2-0881: The existing main switchroom is below a wet area the fit-out adds, and the slab between...
  - B2-0734, B2-0979: The gas valve and the mechanical exhaust are not interlocked.
  - B2-0096, B2-0975: Safety glass was not supplied to a low-level pane in a school corridor that requires it.
  - B2-0001, B2-0362: Coring a 1980s office slab for a new kitchen stack cuts a post-tension tendon the archi...
  - B2-0171, B2-0984: A compartment wall in a tenancy fit-out stops at the ceiling grid instead of the slab.
  - B2-0085, B2-0974: Aluminium flashing in contact with the existing copper valley is corroding at the junct...
  - B2-0194, B2-0948: A fire door installed in a two-hour wall is rated for one hour.
  - B2-0353, B2-0544: Crane collector gear clashes with the new cable tray on the crane beam.
  - B2-0173, B2-0994: Sprinkler heads in a new ceiling clash with the grid and end up too far from the wall t...
  - B2-0174, B2-0954: The existing booster cannot be seen from the brigade attendance point once the new entr...
  - B2-0302, B2-0952: The consumer mains cannot carry the maximum demand of the new equipment.
  - B2-0416, B2-0691: A soil stack fixed hard to a bedroom wall is audible in the room.
  - B2-0801, B2-0967: Scaffold for a facade repair cannot found on a slab that is still back-propped.
  - B2-0413, B2-0980: An eyewash has no drainage and no tempered supply.
  - B2-0415, B2-0503: A pipe is cast into an existing slab penetration with no sleeve, so the membrane cannot...
  - B2-0628, B2-0760: A retained tree's protection zone covers the only footing line for the extension.
  - B2-0170, B2-0985: The product proposed to seal a gap at an existing slab edge is not tested for the gap w...
  - B2-0283, B2-0740: A dust collector's relief is aimed at a path workers use.
  - B2-0307, B2-0983: The breakers supplied do not discriminate with the upstream device.
- Rows with an existing record that covers the same circumstance: 121 (12 per cent). They point to 103 distinct records: 81 failure modes, 17 unforeseen conditions and 5 interfaces. 109 of the 121 rows point to at least one record that came from batch 1 (105 only to batch 1 records, 88 distinct batch 1 records). I listed a record only when it names the same pitfall, not just the same system; many near misses (same system, different problem) are deliberately left out. Rows by kind: 58 unforeseen conditions, 39 design errors, 18 workmanship defects, 6 process rows. Pass B should extend these records with the batch 2 detail rather than write new ones.
- Matching was done by shared system and word overlap, then each candidate was read. Lower-scoring matches (below about 0.45) were not read, so the covered count is a lower bound; Pass B should recheck rows in the same area before writing.

## The 20 most uncertain mappings

| Row | Circumstance | Why it is uncertain |
| - | - | - |
| B2-0066 | A combustible panel found where the brief said solid aluminium | Found on site, so an unforeseen condition, but the wrong brief is also a design miss. |
| B2-0303 | The supplier's point of supply is not where the design assumed | Process (authority) problem or a condition; kept as the owner's latent condition, side B mapped to the utility anchor. |
| B2-0356 | A high-voltage easement crosses the extension site | Could be a neighbour or authority matter; mapped to the utility anchor. |
| B2-0170 | A seal product tested for a different gap | Design choice or workmanship; kept as the owner's kind. |
| B2-0750 | Access-control reader on a hollow wall and the lock will not suit the latch | Two causes in one row; mapped to access control only. |
| B2-0667 | Sealant stains the stone beside the joint | Tiling and flashings/seals fit loosely; the stone is a finish and the joint is the envelope. |
| B2-0097 | Skylight well drips condensation once the roof space is cooled | Side A windows-doors versus roof coverings; side B only known as mechanical. |
| B2-0220 | Refuge communication point has no cable path to the fire panel | Comms cabling and fire detection both fit; mapped to structured cabling and detection. |
| B2-0405 | Overflow relief gully buried by new paving | Roof drainage versus site stormwater; chose roof drainage for side A. |
| B2-0841 | Subfloor clearance too small to run new services | Side A hydraulic sanitary assumed; row names no specific service. |
| B2-0549 | Expansion loop fills the cupboard beside it | Pipework is central plant here; joinery is the cupboard. |
| B2-0932 | Soakage system designed in clay that fails the permeability test | Ground condition versus hydraulic design; owner says latent condition, side A ground. |
| B2-0324 | Essential and general loads combined on a board the fire strategy treats as separated | Changed from latent condition to design error; the board is existing, so it could be either. |
| B2-0365 | Common-area lighting fed from a tenancy board | Changed from latent condition to design error; could be an earlier-work condition. |
| B2-0907 | Concrete strength results below the grade the PT procedure assumed | Changed from latent condition to workmanship defect; could be supply. |
| B2-0933 | Town-main pressure will not serve the top floor without a pump the design omitted | Changed to design error; the authority supply pressure is the surprising part. |
| B2-0042 | Crane runway out of level beyond the replacement crane's tolerance | Existing record on runway settlement is related but not the same, so not listed as covered. |
| B2-0108 | Sealed box-gutter overflow discharges into the ceiling | Roof coverings versus roof drainage; existing blocked-overflow record is related, not listed. |
| B2-0236 | Tenant condensers exceed the base-building allowance | Tenant fit-out against base-building plant; side B left as the top-level electrical or mechanical system from the owner list. |
| B2-0332 | External lighting spills across the boundary | Neighbour anchor chosen for side B; could be authority (permit). |

Other rows with a note: about 240, mostly "side B is not narrowed by the row text". The owner list uses broad names for the second system ("Interiors and finishes", "Structure") and the row often does not say which part. These map to the top-level system on purpose, so Pass B can attach the record to the right leaf system after reading the row.

## Rows I suggest rejecting

These are suggestions only. They are not recorded in the coverage ledger, where every row is still pending.

**Short restatements of an earlier, more specific row (48 rows):** B2-0943, B2-0944, B2-0946, B2-0947, B2-0948, B2-0949, B2-0950, B2-0951, B2-0952, B2-0953, B2-0954, B2-0955, B2-0956, B2-0957, B2-0958, B2-0959, B2-0961, B2-0962, B2-0963, B2-0964, B2-0965, B2-0966, B2-0967, B2-0974, B2-0975, B2-0977, B2-0978, B2-0979, B2-0980, B2-0981, B2-0982, B2-0983, B2-0984, B2-0985, B2-0986, B2-0987, B2-0988, B2-0990, B2-0991, B2-0992, B2-0993, B2-0994, B2-0995, B2-0996, B2-0997, B2-0998, B2-0999, B2-1000. Each has the same circumstance as an earlier row (see the duplicate groups) in fewer words, usually with the element and trigger lost. Keep the earlier row and record the later one as a duplicate.

**Not recommended for rejection:** the other 11 short tail rows (for example B2-0961 "A stack clashes with a beam", B2-0999 "Paint covers a smoke seal") are terse but have no longer twin; they are usable once Pass B adds context from the nearest specific row.

## What batch 2 adds compared with batch 1

- Much more about existing buildings: 868 of 1,000 rows against 154 in batch 1. Batch 1 was mostly new-build pitfalls; batch 2 is mostly what is found when you open, extend, upgrade or fit out something already built.
- A far bigger share is unforeseen conditions: 489 against 101 in batch 1 (the owner's label, with 43 changed to design or workmanship). Process, authority, supply and weather rows are fewer (75 against 250), so delivery-only rows are rare (14 against 125).
- Each row says what triggered it, how it would be seen early and how to reduce it. That is the content for the early-warning and de-risk fields in Pass B, which batch 1 lacked.
- Every row names the element involved, so system mapping is more exact than in batch 1 for the first side. The second side is often only the owner's top-level name.
- New building category: institution (schools, hospitals, aged care, courts), 216 rows.
- Far less overlap with the existing knowledge: 12 per cent of rows already have a matching record (batch 1: 17 per cent), nearly all of them records that batch 1 created. Heavy new ground: post-tension and transfer slabs, hazardous materials in existing buildings, change of use, institution and occupied-building constraints, tenant fit-out against base-building plant.
