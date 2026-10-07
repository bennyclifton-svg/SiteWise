# Structure cluster extraction report

## K4 vehicle-clearance ownership correction, 6 October 2026

`fm.vehicle-clearance-lost-under-structural-beam` now lives in this cluster.
Batch 2 rows B2-0916 and B2-0917 describe clearance lost below structural beams;
the previous fire placement was explicitly attributed to keyword triage.
The complete draft record, including its ID, attachment, question, routing,
sources and unverified-standard note, is preserved. The two coverage entries
move with it. Its note about placement in fire records historical provenance.
This is an ownership correction, not engineering verification or owner approval.
See `docs/evidence/2026-10-06-k4-vehicle-clearance.md` for validation.

Status: draft; completed missing graph files on 2026-09-29. This is a draft extraction checkpoint, not an exhaustive or instrument-verified knowledge base. Existing child systems were preserved. No seed numerical claim was promoted to verified.

## Coverage

| Area | Coverage |
|---|---|
| Ground and foundations | Site investigation, residential footing route, controlled fill, soil testing, groundwater, piling, retaining pressure, adjoining foundations |
| Structural materials | Concrete exposure/detailing/testing, reinforcement traceability, steel stability/fabrication/welding/corrosion, timber framing/modifications/durability, masonry movement |
| Structural actions | Design basis, wind, seismic, plant loads (canonical wet-air interface), flood actions, temporary works and precast stability |
| Site interfaces | Survey datum, tree roots and moisture, drainage, podium loads/waterproofing, contamination validation, acid sulfate soils, underground services |
| Concealment gates | Reinforcement before pour, framing before roofing, underground services before paving |

32 systems; 30 rules; 29 interfaces; 30 failure modes. Lead merged 23 determinants into the canonical catalogue; the proposal file is empty.


## Full-file coverage matrix (continuation and follow-up)

Each file below was read in full during this continuation, including the follow-up sweep. Long files were read in bounded chunks. Reading coverage does not imply that every seed assertion was copied or verified.

| Source files in `../clerk/data/seed/` | Extraction treatment |
|---|---|
| `ncc-reference-guide.md`; `as-standards-reference.md` | Full reference sweep; applicable physical requirements retained as draft/unverified; no executable table created from seed prose |
| `structural-residential.md`; `structural-commercial-industrial-guide.md`; `geotechnical-guide.md` | Structural and ground design, construction and failure interfaces |
| `civil-residential.md`; `civil-commercial-industrial-guide.md`; `surveying-guide.md` | Levels, fill, drainage, ground and infrastructure interfaces |
| `arborist-guide.md`; `landscape-architecture-guide.md`; `traffic-transport-guide.md`; `waste-management-guide.md` | Root zones, podium load/drainage, site access and collection interfaces; specialist design remains external |
| `electrical-services-guide.md`; `ict-av-security-guide.md`; `mep-residential.md` | Electrical, communications, security and integrated domestic services |
| `mechanical-services-guide.md`; `hydraulic-services-guide.md`; `architectural-services-guide.md`; `access-consultant-guide.md` | Cross-system coordination; detailed wet-air/access content remains with the owning clusters |
| `trade-interfaces-coordination-guide.md` | Concealment, sequence and installation boundaries; lead canonical aliases preserved |
| `remediation-due-diligence-guide.md`; `building-remediation-rectification-guide.md`; `defects-and-dlp-guide.md` | Contamination, cap integrity, vapour pathways, retained structure and investigation before repair |
| `residential-construction-guide.md`; `new-dwelling-guide.md`; `renovation-guide.md`; `ancillary-guide.md`; `multi-dwelling-guide.md`; `multi-residential-apartments-guide.md` | Full residential/archetype sweep; extracted physical dependencies, not legacy role or approval doctrine |
| `commercial-construction-guide.md`; `small-commercial-guide.md`; `industrial-construction-guide.md`; `institution-construction-guide.md`; `mixed-use-construction-guide.md`; `infrastructure-construction-guide.md` | Full nonresidential/archetype sweep; use boundaries, live services and structural interfaces; no assumed universal classification |
| `sustainability-energy-guide.md`; `non-residential-sustainability-energy-guide.md` | Certificate commitments, supply impacts, metering and operational verification; no rating prediction |

Intentionally excluded from extraction: fees and RFP wording, consultant appointment stages, commercial contracts, warranties/liability, cost rates, durations, procurement methods, role-specific agent behaviour and all legacy PM doctrine. Technical facts embedded in those sections were considered. Role overlays, dedicated contract/cost/procurement/program guides, `setup-and-commission-guide.md`, and the frozen clerk PM doctrine were not adopted as technical authorities. Detailed fire/access, envelope/finishes and wet-air extraction is owned by the other clusters; this report does not claim a full reread of their dedicated specialist seed inventories.

Residual source gaps: none in the full-file inventory above. Primary NCC/AS/state instruments, manufacturer instructions, network rules, project-specific engineering and test data are not comprehensively verified by this extraction. Seed completeness is limited by the source itself: specialist healthcare, process hazardous areas, linear infrastructure and complex remedial design are signposts, not detailed design knowledge. Full seed reading is complete for the listed cluster sources; engineering completeness and primary-instrument verification remain open.

## Contradictions and unsupported seed claims

- Residential structural seed gives balcony imposed load 3.0 kPa; AS reference gives 2.0 kPa. Neither is a design default; occupancy and primary table verification are needed.
- Wind region descriptions disagree: structural residential/NCC treat B separately from C/D cyclonic, while AS reference calls B cyclonic and calls A0–A7 plus B/C/D eight regions. Do not derive wind class from these descriptions.
- Structural residential wrongly points to AS 3000 in its earthquake discussion; this wiring reference is not adopted as a structural rule.
- AS 2870 classifications vary between seed tables, including deep variants. Existing finite site class proposals are not proof of an exhaustive primary-standard option set.
- AS 3798 prose contradicts itself about the effect of standard versus modified compactive effort; no density acceptance table was created.
- AS 5131 source calls CC4 an addition in practice while other seed text presents CC1–CC4 uniformly. Category applicability and inspection obligations remain unverified.
- Concrete cover examples omit important dimensions of the actual standard. The copied numbers remain unverified and no executable cover table was invented.
- NCC flood and structural clauses are copied with the seed numbering NCC 2019 annotation; current numbering requires primary text.
- Generic seed statements making every inspection a mandatory NCC hold point are not treated as jurisdiction-independent statutory rules. New hold-point rules are explicitly project specification requirements.

## Determinants and tables

Converted unsupported proposed derivations to stated-value extraction: earthquake_design_category, min_concrete_cover, as2870_design_route, timber_framing_part, nepm_hil_category, termite_prone_area. These are not operational computed outputs. Removed invented earthquake category I/II/III choices; stripped unsupported soil definitions from Ae–Ee options. State alone cannot establish termite risk.

Pending primary-instrument work: wind-region/classification and scope mapping; importance-level actions; earthquake method/category; concrete exposure/cover with all required inputs; residential footing scope; timber framing eligibility; termite locality rules; soil and concrete test acceptance. No tables written. Numeric option descriptions inherited from seed proposals remain drafts, not verified boundaries. structure_height must not be used interchangeably for overall height and framing wall height; the lead clarified this distinction in the merged determinant notes.

## Cross-cluster reconciliation

Cross edges connect structure to hydraulic drainage, wet-air/electrical plant loads, services penetrations, fire protection, envelope concealment and podium waterproofing. Removed duplicate if.plant-loads-structure and fm.plant-loads-structure in favour of services-wet-air ownership. Top-level cross-cluster IDs were deliberately used where child ownership is not yet reconciled. No unresolved system IDs in the cluster checker. Lead canonical aliases in `knowledge/MERGE.md` were preserved. The external-levels interface now lives in envelope; its retained failure detector references the canonical edge. Follow-up adds cap/service-trench continuity, soil-vapour/building protection, and crane-outrigger ground bearing. Soil vapour connects site contamination and substructure to mechanical ventilation; no universal ventilation system is assumed.

## Questions and validation

Questions use explicit `text`, ask for stated evidence, and route to the relevant systems. Failures require affirmative evidence of the defect; an absent passage is not proof of a defect. Each new question checks one fact. Structural calculation, thresholds and compliance outcomes stay in code or primary technical design.

Reviewed TypeSafe [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13): literal criteria, narrow state and no arithmetic. Foundation/schema also cite [AI-powered software](https://docs.typesafe.ai/concepts/how-to-build-with-system-one); the direct tool fetch failed in this continuation, so no claim of a successful reread. Runtime fan-out and per-question calibration remain implementation/evaluation work, not validated by this YAML checker.

Validation: python tools/check_knowledge.py --only knowledge/clusters/structure: 0 errors, 0 warnings after the full-file sweep and seven cross-cluster additions across these two folders. This validates schema and references, not engineering correctness or extraction completeness.

Additional source conflicts from the full sweep: civil fill-density examples differ from AS-reference compaction examples; no percentage is selected as a universal acceptance criterion. Remediation prose sometimes treats screening levels as cleanup/pass-fail criteria but elsewhere correctly calls them investigation screens. HIL-A is not inferred solely from a residential label. The remediation guide gives inconsistent site-audit statement descriptions and speculative PFAS vapour wording; neither was adopted. Defects-guide crack widths do not classify structural safety, and its balcony AS 3740 reference conflicts with external-waterproofing coverage. No repair prescription was extracted from those claims.

## K4 investigation wording, 6 October 2026

Recast 16 draft investigation labels from physical-work instructions into
checks or assessments of their cited source constraints. All other record
fields are preserved, including predicates, signals, sources and draft status.
See `docs/evidence/2026-10-06-k4-investigation-labels.md` and the exact
before/after inventory `docs/unforeseen/k4-investigation-label-review.json`.
This bounded wording pass does not approve content or close owner review.

K4 follow-up: recast 24 further investigation labels that selected remedies
or design solutions before the investigation. Only labels changed. The shared
review inventory now records 82 corrections across the catalogue; this remains
a bounded, unapproved wording pass. See
`docs/evidence/2026-10-06-k4-investigation-labels.md` and
`docs/unforeseen/k4-investigation-label-review.json`.

## K2 draft structural review — 7 October 2026

Added `cq.existing-structural-intervention-sequence-review` from renovation-guide
`## Structural intervention`: existing structural/substructural alteration,
replacement, upgrade or removal raises an undated planning review of load paths,
sequence and applicable temporary support. No propping design or mandatory
support inferred. Distinct from new-load capacity interface investigations.
Existing hazardous-material illustrative CQ remains unchanged and unverified.
Source coverage, exclusions and merge review: docs/evidence/2026-10-07-k2-seed-coverage.md.
All content remains draft; owner usefulness review is pending.
