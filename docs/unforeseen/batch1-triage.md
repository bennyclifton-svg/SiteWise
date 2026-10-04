# Batch 1 triage: unforeseen circumstances sorted and mapped

Status: draft for owner review. Nothing here is verified, and no knowledge record has been written or changed. This is Pass A of the unforeseen-conditions library (K4): every one of the 1,000 owner-supplied rows has been sorted by kind and category, mapped to the SiteWise building systems, checked for duplicates and checked against the failure modes and interfaces already in the knowledge base. Pass B will turn the usable rows into records.

The sorted rows are in `data/unforeseen/batch1_triage.csv`. The mapping of the 946 distinct system names is in `knowledge/works/system_terms.yaml` and is reused for batch 2. `python tools/check_triage.py` checks the file.

## How to read this

- Kind says what sort of surprise it is. Only an unforeseen condition is a state of the site or an existing building found during works. Design errors, workmanship defects and process, authority, supply or weather problems are kept separate so they are not raised as conditions to investigate before tender.
- Category is one of the nine groups in the plan.
- Cluster is the knowledge area that would own the record. It follows the main physical system in the row. Rows with no physical system on either side (programme, contract, authority and so on) go to "delivery".

## Counts

**By kind**

| Kind | Rows |
| - | - |
| Design or coordination error | 371 |
| Workmanship defect | 278 |
| Process, authority, supply or weather | 250 |
| Unforeseen condition (a state of the site or existing building found during works) | 101 |

**By category**

| Category | Rows |
| - | - |
| Supply and site operations | 347 |
| Design and scope | 320 |
| Authorities and utilities | 115 |
| Third parties and occupation | 50 |
| Ground and site | 49 |
| Existing systems on test | 46 |
| Existing structure | 38 |
| Concealed services and earlier work | 27 |
| Hazardous materials | 8 |

**By cluster**

| Cluster | Rows |
| - | - |
| structure (site, substructure, structure) | 309 |
| envelope (envelope, interiors) | 202 |
| services-wet-air (hydraulic, mechanical) | 180 |
| delivery (no physical system on either side) | 125 |
| electrical-comms (electrical, comms, lifts) | 124 |
| fire (fire, egress and access) | 60 |

**By work type (a row can have several)**

| Work type | Rows |
| - | - |
| New build | 847 |
| Refurbishment or upgrade | 300 |
| Extension | 208 |
| Remediation of defects | 139 |

**Existing building**

| Existing building | Rows |
| - | - |
| No | 846 |
| Yes (concerns an existing building, fabric or services) | 154 |

Of the 154 rows about an existing building, 53 are unforeseen conditions, 58 are design or coordination errors, 29 are process, authority, supply or weather, and 14 are workmanship defects.

Only 101 of the 1,000 rows are true unforeseen conditions. Most of the list is design errors, workmanship and process. That matters for Pass B: the library of conditions to raise before tender will be about a tenth of the list, and the rest feeds design-review checks and contract and programme risks.

## Duplicates and rows already covered

- Duplicate groups within the batch: 6 groups covering 14 rows. The batch is much less repetitive than its size suggests; most rows that look alike differ in the element or the cause.
  - B1-0043, B1-0810: Retaining wall drains blocked / Retaining wall drainage fails
  - B1-0150, B1-0209: Masonry movement joints omitted / Movement joints omitted
  - B1-0161, B1-0965: Termite damage in timber frame / Termite damage found after stripping
  - B1-0391, B1-0519, B1-0600: Noise from mechanical plant exceeds the limit / Condenser unit location violates noise rules
  - B1-0046, B1-0827, B1-0864: Rock hammering vibration damages a neighbour / Vibration from demolition cracks an adjoining wall
  - B1-0004, B1-0682: Unmapped utility in the footing line / Underground cable found during excavation
- Rows with a similar existing failure mode or interface: 166. "Similar" means an existing record already covers the same pitfall or a close part of it. Many are partial matches (the existing record states the principle, the row gives one example), so Pass B should treat these as candidates to extend or cite rather than as rows to drop. The existing id is in the `similar_existing` column.

## The 20 most uncertain mappings

| Row | Circumstance | Why it is uncertain |
| - | - | - |
| B1-0004 | Unmapped utility in the footing line | Unknown utility could be private or authority; mapped to site.services-connections, which is authority infrastructure. Kind chosen as unforeseen_condition. |
| B1-0011 | Dewatering settles neighbouring buildings | Neighbour settlement from dewatering could be read as an unforeseen condition; chose design_or_coordination_error because the method causes it. |
| B1-0052 | Heavy racking loads exceed the slab design | Racking is a tenant item with no system id; mapped to slab-on-ground. |
| B1-0126 | Column reduced late to widen a corridor | Late column change: workmanship_defect is a weak fit; could be design. |
| B1-0159 | Existing structure not built to drawings | Existing structure not as drawn: side B mapped to design (the drawings). |
| B1-0305 | Smoke control fails commissioning | Commissioning failure treated as workmanship defect; could be design. |
| B1-0310 | Cladding fails the fire test | Cladding fire test failure treated as supply/product (process); could be workmanship. |
| B1-0401 | Low water pressure on upper floors | Building height mapped to design (stack height), not the planning authority. |
| B1-0454 | Easement drain discovered under the footprint | Neighbour term here concerns an easement drain; kind unforeseen_condition could be process. |
| B1-0463 | Water supply isolation for tie-ins disrupts occupants | Existing supply is water here, not electrical; side A overridden to cold water. |
| B1-0505 | Services stack shortens ceiling height | Services term is generic; mapped to mechanical. |
| B1-0506 | No access panels for valves | Services term is generic; mapped to mechanical. |
| B1-0684 | Tenant fit-out exceeds the electrical rising main capacity | Rising main is the electrical rising main, not water; mapped to switchboards. |
| B1-0689 | Intrusion into ceilings reveals hidden hazards | Hidden hazards in ceilings: hazardous_materials is possible; category concealed services chosen. |
| B1-0712 | Slab moisture reading too high to lay flooring | Slab moisture delaying flooring treated as programme/process. |
| B1-0784 | Cold storage vapour barrier damaged | Damaged vapour barrier on a cold store treated as unforeseen; could be workmanship. |
| B1-0839 | Bushfire smoke stops works | Environment here means bushfire smoke; mapped to weather. |
| B1-0841 | Flooded basement from a storm drains into a lift pit | Site water here is stormwater, not temporary supply; sides overridden. |
| B1-0941 | Strata plan registration delayed | Settlement here is property settlement, not ground settlement; mapped to contract. |
| B1-0962 | Staged handover needs separate services isolation | Staged handover isolation treated as design; could be process. |

Other rows with a note in the file: 18. A recurring cause of doubt is that the owner list uses generic names ("Services", "Structure", "Electrical", "HVAC", "Controls") and the same word means different things in different rows. These map to the top-level system by default, and a row-level override is used only where the row clearly needs a more exact system.

## Rows I suggest rejecting

These are not recorded in the coverage ledger (every row is still pending there). They are suggestions for you to accept or overrule.

Too vague to turn into a record:

| Row | Circumstance | Reason |
| - | - | - |
| B1-0107 | Early shrinkage cracking | Names a symptom with no building element or cause; could attach to any concrete element. |
| B1-0108 | Steel fabrication error | No indication of what went wrong; every steel element could fit. |
| B1-0905 | Latent site condition triggers a claim | Restates the contract concept of a latent condition rather than a condition. |
| B1-0909 | As-built differs from the drawings | Generic; the useful versions are the specific ones (for example B1-0159). |
| B1-0911 | Design not coordinated before tender | Generic coordination failure with no system. |
| B1-0912 | Drawings issued with unresolved clashes | Generic; the specific clash rows are more useful. |
| B1-0914 | Design team does not coordinate on a 3D model | Describes team practice, not a condition of the building. |
| B1-0928 | Procurement strategy mismatched to risk | A management judgement with no observable trigger. |
| B1-0985 | Spare parts and keys not handed over | Trivial handover item with no system consequence. |
| B1-0989 | Building performance below the NABERS or Green Star target | Outcome with no cause; depends on the target and the whole design. |
| B1-0990 | Post-occupancy complaints about comfort | Outcome with no cause or system. |
| B1-0999 | Insurance does not cover a weather event | Depends entirely on the policy wording. |

Project finance and cost events with no building consequence (out of scope for building knowledge, though they may belong in contract or budget guidance):

- B1-0938: Developer finance withdrawn
- B1-0939: Presales not met and finance falls over
- B1-0940: Interest rate rises affect feasibility
- B1-0894: Material price spike
- B1-0994: Developer contributions are re-assessed
- B1-0996: Utility headworks charges increase

## Decisions I made where the brief was silent

- Workmanship defects and design errors on new buildings still need one of the nine categories. They go to "Supply and site operations" and "Design and scope" respectively, because none of the nine is a workmanship category.
- Failures of new building services at commissioning are "Existing systems on test"; the category is the nearest fit for a failed test.
- Remediation is added to the work types only where a row is about repairing defects or rectification, and for the defects-period rows.
- "Settlement" in the finance rows means property settlement, not ground settlement, and is mapped to the contract anchor.
- Duplicates were judged by circumstance, not by wording: two rows are duplicates only when the same thing goes wrong in the same element.

