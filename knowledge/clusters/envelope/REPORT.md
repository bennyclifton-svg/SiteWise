# Envelope and interiors extraction report

Status: draft. Completed continuation of the previously interrupted extraction on 29 September 2026. No item is owner-reviewed.

## Scope and coverage

The original systems, rules and determinant proposals were preserved. This continuation supplied the missing interface graph and failure detectors, added product-specific workmanship rules, annotated remaining NCC seed locators, and marked unverified table derivations pending. All 133 seed numeric claims remain unverified; none was promoted from prose into executable thresholds.

| Physical topic | Coverage | Remaining limitation |
|---|---|---|
| External walls, curtain wall, windows and doors | Weatherproofing, performance grades, glazing, movement, testing, service entries | Verify product/test-method claims and project-specific loading |
| Pitched roofs and sarking | Coverings, fixings, flashings, frame and enclosure sequence | Profile-specific pitch and fixing design remain product-dependent |
| Wet-area waterproofing and tiles | Extent, falls, materials, substrate compatibility, flanges, inspection before covering, defects | Seed dimensions, inspection-law generalisations and product classifications need primary review |
| Balconies, podiums and membrane roofs | External membrane rules, facade junctions, drainage/overflow, movement | Green-roof root protection and membrane access added; basement waterproofing belongs to substructure |
| Insulation, wrap and sealing | Thermal bridges, vapour control, insulation quality, roof and downlight interfaces | No executable universal R-value or condensation table |
| Fabric energy / BASIX / NatHERS / Section J | Pathways, specification commitments, substitutions and dependent services | State adoption, whole-of-home scope and apartment rating methodology need authoritative applicability |
| Bushfire envelope | BAL evidence, envelope elements, ember entries and site assumptions | BAL-specific numeric rules and static water attribution are unverified |
| Inter-occupancy, facade and room acoustics | Airborne/impact, flanking, service penetrations, ventilation with closed windows, absorption | No acoustic compliance inferred from a document; field/lab metrics and scope need verified tables |
| Interior linings, joinery, floors and paint | Product workmanship, substrate moisture, cabinet support, pre-lining/painting sequence, observed defects | Clinical finish/brief boundary added; specialist material standards and secure hardware details remain incomplete |
| Renovation junctions | Old-to-new weather barrier and existing-energy-input defects | Investigation-specific remedial design is not generated |

### Full-file coverage audit (follow-up)

The following files were read in full across the original continuation and follow-up, including technical facts embedded in programme, procurement and role sections. Large NCC/AS and archetype files were read in bounded chunks; truncated portions were reread. Full-file reading means source coverage, not authoritative verification or a claim that every sentence merits a graph record.

| Full file read (under clerk/data/seed/) | Envelope/interiors disposition |
|---|---|
| ncc-reference-guide.md | All sections read; envelope, energy, acoustics and cross-system boundaries retained; other requirements owned by their clusters |
| as-standards-reference.md | All sections read; material, membrane, glazing, insulation and acoustic standards mapped; numerical prose remains unverified |
| finishes-residential.md | Cladding, roofing, lining, wet areas, tiles, joinery, flooring and coatings mapped |
| trade-interfaces-coordination-guide.md | Concealment gates and physical trade junctions mapped |
| residential-construction-guide.md | Full Class 1/10 guide including folded multi-dwelling/ancillary sections; new and retained construction boundaries mapped |
| commercial-construction-guide.md | Facade, waterproofing, fitout/acoustics, energy, heritage and services interfaces mapped |
| multi-residential-apartments-guide.md | SOU separation, external membranes, facade, trafficable/green roofs, energy and concealed work mapped |
| new-dwelling-guide.md | Physical sequence and BASIX commitments; added window-orientation/model-change edge |
| renovation-guide.md | Existing conditions, old/new junctions and temporary enclosure |
| multi-dwelling-guide.md | Repeated and handed assemblies require scoped evidence; no project-wide inference from a typical detail |
| ancillary-guide.md | Read including conflicting BASIX/classification claims; applicability remains pending |
| small-commercial-guide.md | BASIX/Section J distinction; superseded harness scope and role instructions excluded |
| sustainability-energy-guide.md | Fabric, glazing, commitments, substitutions and rating-pathway evidence mapped |
| non-residential-sustainability-energy-guide.md | Fabric/services model boundary; rating and operational assumptions remain project-specific |
| defects-and-dlp-guide.md | Observed waterproofing, facade, floor, tile, paint and glazing defects mapped; contract periods excluded |
| building-remediation-rectification-guide.md | Diagnosis, retained junctions, compatible repairs, temporary protection and acceptance evidence |
| industrial-construction-guide.md | Envelope/process-environment coordination reviewed; no cold-store technical assembly invented from a brief |
| institution-construction-guide.md | Clinical finishes added as project-brief evidence; subtype-specific standards are not universal rules |
| mixed-use-construction-guide.md | Added shared-structure acoustic path between different uses |
| landscape-architecture-guide.md | Podium/roof build-up and envelope boundary reviewed; green-roof protection edge added |
| access-consultant-guide.md | Threshold and accessible-fitout boundaries covered by existing graph |
| architectural-services-guide.md | Physical consultant boundaries reviewed; appointment, fee and novation doctrine excluded |
| structural-residential.md | Frame/roof, movement, wind/BAL dependencies reviewed; calculations belong to structure/code |
| mep-residential.md | Roof/wall penetrations, rough-in, ventilation, condensation and BASIX interfaces reviewed |

The follow-up adds six interfaces and six observed-condition detectors: landscaped-roof root protection, trafficable-roof membrane access, mixed-use structure-borne sound, temporary enclosure during repairs, changed window orientation versus the energy assessment, and clinical finishes versus the facility's infection-control brief. Existing merged IDs and records were preserved; no new numeric thresholds or determinants were added.

Intentionally excluded content: fees, market rates, contract administration, statutory warranty periods, procurement forms, agent role doctrine, generic schedules and programme durations. Technical facts appearing inside those sections were considered. No seed instructions override SiteWise's foundation design.

Exact file-reading boundary: remediation-due-diligence-guide.md lines 1–110 were read to establish its explicit contaminated-land scope and handoff to building-remediation-rectification-guide.md. The rest of that contaminated-land file is site/substructure ownership and was not read by this cluster. Deep mechanical, hydraulic, electrical, ICT and civil appointment guides are owned by the corresponding clusters; this audit does not claim full reads of those files. There is no separate acoustic/interiors/facade seed in the seed inventory; their coverage is in the full-file references above. The core envelope/archetype file-reading gap previously disclosed here is closed.

Residual technical gaps: primary instrument verification (including state adoption and amendments), complete applicability for the five pending tables, specialist cold-store and cleanroom assemblies, secure-facility and early-childhood hardware details, below-ground waterproofing, facade maintenance access design, acoustic field-versus-laboratory acceptance, and project-specific remedial acceptance criteria. Seed prose alone cannot complete these. A clinical finish evidence match does not establish infection-control compliance; a repaired assembly needs project-specific testing. Model changes to schedules/controls remain a services energy boundary, and the geometry edge records explicit reassessment evidence rather than recalculating a rating.

## Seed contradictions and verification risks

| Issue | Conflicting or unsupported seed claims | Treatment |
|---|---|---|
| Wet versus external membranes | Finishes and defects seeds attribute balcony work to AS 3740; AS reference distinguishes AS 4654 external use | Separate internal and external systems; retain explicit wrong-product detector |
| Balcony upstands | Finishes/defects say 100 mm; AS reference says 150 mm with 75 mm protected threshold cases | All numeric claims unverified; no lookup emitted |
| Falls | Trade interface says 1:100; AS wet-area text says 1:80; external AS section claims different ratios by membrane type; defects attributes external falls to AS 3740 | Retain scope-specific rule evidence; do not infer a universal minimum |
| Adhesive coverage | Finishes says 85%; defects says 95% | Both remain seed claims, no computed pass/fail |
| Membrane application | Finishes says two coats for sheet membranes; AS reference discusses liquid dry-film thickness and multiple coats | Product instructions control; general seed application wording needs review |
| Safety glass locations | Finishes says 300 mm at doors / 500 mm low-level; AS reference gives different distances and conditions | All unverified; glass-location table pending |
| NatHERS | Sustainability seed says national 6 stars; NCC expansion/apartments say 7 stars and oversimplify individual apartment minima | Pending derivation; verify class, adoption, exclusions and rating method |
| Section J | Seed part names and numbering disagree; housing Section 13 and NCC 2019 locators are mixed | Preserve locators with seed-numbering annotation; clause_verified remains false |
| Condensation | NCC guide uses F6; AS guide uses F8; universal vapour-barrier advice is climate-dependent | No universal assembly deduction; assessment evidence only |
| Acoustic criteria | Commercial guide office levels differ from AS reference; Rw, Rw+Ctr, Ln,w and Ln,w+CI are not interchangeable | No numerical evaluation by Jev; verified metric-specific criteria needed |
| Bushfire | NCC guide uses BAL-FMP versus other seeds BAL-FZ; dedicated water requirements attributed broadly to AS 3959 | Keep draft notes; state planning/fire authority provisions require separate verification |
| Inspection and licensing | Seeds assert universal mandatory NCC waterproofing inspection and broad state licensing coverage | Interface is a physical concealment gate; legal authority and required inspector remain project-specific |
| Facade tests | AS reference mixes some AS 4420 part references and generalises test pressures | Retain unverified methods; require representative specimen and project-specific evidence |
| Brick ties | AS reference gives 600 mm horizontally and vertically; finishes seed gives 600 mm vertically and 900 mm horizontally | Seed spacing is not an executable fixing design |
| Ancillary classification and BASIX | Ancillary/residential tables describe habitable studios as Class 10a and omit pool BASIX, while ancillary prose distinguishes new dwellings from alteration thresholds | Classification and BASIX applicability require primary review; no table inferred |
| Class 4 and housing scope | Apartment seed attributes Class 4 amenity/energy to Volume Two, while NCC references differ | Verify volume and building-part applicability before selecting rules |
| Fire compartment limits | NCC seed 5,500 m² versus AS sprinkler seed 3,500 m² | Cross-cluster contradiction owned by fire/determinants, not used in envelope calculation |

## Pending lookup tables

All existing derives contracts below now have pending: true and a reason. Code must return unknown until primary verification and complete applicability are available.

| Table | Pending purpose / missing applicability |
|---|---|
| roof_ceiling_min_r_value | Assembly, roof form, colour, climate, building part and pathway; product R-value is not total R-value |
| wall_min_r_value | Wall assembly and thermal bridges, climate, glazing interactions and pathway |
| nathers_minimum_stars | Jurisdiction adoption date, class, apartment average/individual rules, exclusions |
| basix_applicability | NSW planning instrument, work scope, exempt cases and current alteration thresholds |
| sound_insulation_requirements | Class, adjacent room use, separating element, airborne versus impact metric and tested construction |

Additional primary tables needed before numerical checking: wet-area membrane extent and floor falls; external membrane upstands and drainage design; safety glazing locations; BAL element requirements; room heights and natural light applicability; acoustic design sound levels/reverberation by room use. None was manufactured from seed prose.

## Cross-cluster merge

Most outside endpoints use existing top-level IDs; the new landscape edge uses the verified existing site.landscape child ID. No unresolved system IDs exist in this cluster. The lead may refine these endpoints to stable child systems after matching the other side. All such edges carry cross_cluster: true.

| Interface | Outside systems |
|---|---|
| if.envelope-frame-before-roof | structure |
| if.envelope-roof-service-penetrations | electrical, hydraulic, mechanical |
| if.envelope-rough-in-before-linings | electrical, hydraulic, mechanical |
| if.envelope-joinery-service-positions | electrical, hydraulic |
| if.envelope-dpc-slab-membrane | substructure |
| if.envelope-external-levels-moisture-bridge | site |
| if.envelope-wet-waste-before-membrane | hydraulic |
| if.envelope-hobless-shower-threshold | access-egress |
| if.envelope-external-membrane-drainage | hydraulic |
| if.envelope-membrane-structural-joints | structure |
| if.envelope-facade-frame-movement | structure |
| if.fire-facade-slab-edge | fire-passive |
| if.envelope-facade-services-weatherproofing | comms-security, electrical, hydraulic, mechanical |
| if.envelope-insulation-downlights | electrical |
| if.envelope-envelope-thermal-bridges | structure |
| if.envelope-sealed-envelope-ventilation | mechanical |
| if.envelope-energy-services-commitments | electrical, hydraulic, mechanical |
| if.envelope-basix-water-system | hydraulic |
| if.fire-and-acoustic-separation | fire-passive |
| if.envelope-services-acoustic-penetrations | comms-security, electrical, hydraulic, mechanical |
| if.envelope-plant-room-noise | electrical, hydraulic, mechanical, vertical-transport |
| if.envelope-services-ceiling-space | electrical, fire-active, hydraulic, mechanical |
| if.envelope-grab-rail-backing-before-lining | access-egress |
| if.envelope-bushfire-site-basis | site |
| if.envelope-bushfire-service-entries | comms-security, electrical, hydraulic, mechanical |
| if.envelope-landscaped-roof-protection | site.landscape |
| if.envelope-mixed-use-structure-borne-sound | structure |


Lead reconciliation already retained canonical IDs (see knowledge/MERGE.md), including the two fire-owned edges listed above. Remaining interface topics for review: services before lining; structural frame before roofing; slab membrane/DPC continuity; service penetrations and weatherproofing; ceiling-space coordination; fire/acoustic assembly; downlight/insulation clearance; whole-of-home energy commitments. The wet-air agent removed its duplicate rule.ncc.whole-of-home-energy and references the canonical envelope rule. Electrical's whole-of-home PV offset rule is complementary and can reference it.

## Jev authoring basis

Questions ask whether text explicitly states a detail or reports a defect. A missing statement is not proof of a physical defect. Defect criteria exclude hypothetical warnings. Fire and acoustic evidence are separate questions. Rules, arithmetic, numeric comparison, table selection and document conflicts belong to code. The questions are background work after passage system labelling.

- [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13): literal, atomic questions, explicit text field, relevant state, no arithmetic or generation.
- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): batch independent questions for the same passage state.
- [Confidence](https://docs.typesafe.ai/confidence): noul probability is not Choice confidence; evaluation sets question-specific gates.
- [API](https://docs.typesafe.ai/api): typed question/answer structure; SiteWise additionally requires an exact pinned model.
- [How to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one): the foundation design provides the code-controls-flow principle; direct browser retrieval of this page failed during continuation. Other primary pages above were readable.

Proposed determinant extraction templates with dynamic candidates/criteria require a compiler and evaluation set; these YAML templates are not directly executable API payloads. No runtime calls, confidence calibration or latency benchmark has been implemented by this knowledge-only change.

## Validation

Ran python tools/check_knowledge.py --only knowledge/clusters/envelope --strict: zero errors, zero warnings. Counts below are cluster-local (the checker prints whole-tree totals even with --only).

- 24 child systems
- 65 rules
- 44 interfaces (25 cross-cluster)
- 62 failure modes
- 11 original determinant proposals folded into knowledge/determinants.yaml by the lead; no unmerged proposals
- 7 pending derivation contracts / 5 unique missing tables
- 0 verified numeric claims and 0 verified clause locators in this extraction

## K4 investigation wording, 6 October 2026

Recast 2 draft investigation labels from physical-work instructions into
checks or assessments of their cited source constraints. All other record
fields are preserved, including predicates, signals, sources and draft status.
See `docs/evidence/2026-10-06-k4-investigation-labels.md` and the exact
before/after inventory `docs/unforeseen/k4-investigation-label-review.json`.
This bounded wording pass does not approve content or close owner review.

## K4 flagged batch-1 merges, 6 October 2026

Read all eight source rows for thermal-break, glass-weight and colour/dye-batch
merges. Retained the shared mechanisms for glass capacity and batch appearance,
with reviewer notes limiting what the sources establish. The thermal-break
detector does not cover every interpretation of B1-0239's thermally bridged
window frame; this is an explicit coverage gap despite its ledger association.
Only reviewer notes changed. Questions, routing, predicates, contract wording,
severities and draft status are preserved. Exact rows and open decisions are in
`docs/unforeseen/k4-envelope-merge-review.json`; evidence is in
`docs/evidence/2026-10-06-k4-envelope-merge-review.md`.
