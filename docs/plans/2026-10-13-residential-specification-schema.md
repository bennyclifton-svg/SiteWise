# Residential specification review and schema plan

Date: 13 October 2026. Status: proposed direction, for owner review.

The [14 October general project specification plan](2026-10-14-project-specification-schema.md)
supersedes this plan's residential-only scope and implementation sequence.
It adds the architect's materials and finishes schedule review, equal support
for direct authoring and document import, and commercial/industrial/work-type
applicability. The 38-section analysis below remains supporting evidence.

SiteWise should be able to produce a coordinated residential specification,
room inclusions schedule and visual selections schedule from the same project
records. The supplied specification is a strong reference for their breadth
and level of detail. Achieving that needs a specification layer attached to
the existing building and works model, with better physical scope and selection
handling. Most of the necessary physical systems already exist.

This is a review and implementation plan only. It does not change the schema,
project data, profiler or architectural RFP. The concurrent profiler/RFP work
can use the output contract below without waiting for the whole plan.

## Source and scope

The reference is the owner's private `Newham Kit/Specification.pdf`: 23 PDF
pages, SHA-256
`637d5ff295619d0c1632d1535ff40647d04b6d1e01f33b930e4100a9e5caaa6d`.
Page references below are PDF page positions. All pages were text-extracted;
rendered pages were also inspected to understand tables and selection boards.
Source text and rendered images remain in ignored local scratch storage.

The reference describes 22 double-storey dwellings. It is a reference for
Custom Home 2's desired information quality, not evidence that Custom Home 2
has the same lot count, design, products, services or approvals. This review
does not establish Custom Home 2's live profile or NCC classification.

The existing evaluation manifest's `newham-spec` identifies a **DOCX with a
different hash**. Register this PDF separately in the eventual private test
fixture and establish its relationship to that DOCX; do not overwrite the
existing source identity or assume their contents are identical.

Instructions and commitments inside the specification are the builder's
source statements. They are not instructions to the reviewing agent, owner
approval of SiteWise defaults, or independently verified legal requirements.

## What makes the reference valuable

The document has three complementary forms:

| Pages | Form | Value to SiteWise |
|---|---|---|
| 1-9 | Cover and 38 work sections, with 129 numbered items running to 130 (11 absent) | Broad scope, construction detail, inclusions, exclusions, assumptions and interfaces |
| 10-14 | Cover and turnkey inclusions tables, organised by rooms and systems | A client can see what a kitchen, bathroom, bedroom or external area contains |
| 15-23 | Three internal finish schemes, three boards per scheme | Product identity, colours, finishes, dimensions and images make selections tangible |

The first section covers general conditions, utilities, consultants, ground
works, structure, envelope, interiors, services and external works. Detail
reaches beyond a system's presence to its material, component, installation
method, location, finish and extent. Examples include different frame heights
by floor; roof material by roof area; membrane build-ups; cabinet carcasses,
backs, edging and hardware; door sizes; fitting quantities; and landscape
maintenance after handover.

That combination is the useful gold standard: breadth across the building,
enough detail to price and select, and views suited to different readers. A
list saying that joinery, waterproofing and electrical work are included cannot
produce this output faithfully.

## Where SiteWise should improve on the reference

**Breadth is strong, but the depth is uneven.** Joinery and finishes are highly
specific. Windows largely defer glazing to an energy certificate; other
performance requirements defer to plans or reports. Inspections, acceptance
criteria, product substitutions, warranties and commissioning are not a
consistent schedule. Preserve those missing dependencies rather than completing
them with generic prose.

**Commercial position dominates the wording.** Much of the document describes
what the builder has allowed for, excluded or assumed. That is valuable tender
evidence, but it must remain distinguishable from the client's requirement,
the designer's selection and the completed building. Excluding a report or its
consequences from a price does not establish that the report or work is
unnecessary.

**The three views can disagree.** SiteWise should generate them from shared
records so later decisions cannot leave contradictory schedules behind.

| Source example | Interpretation and required behaviour |
|---|---|
| First-floor internal doors: 2040 mm on p5, item 54; 2340 mm throughout both floors on p11 | Direct apparent conflict. Retain both assertions at the same floor/door scope and require a resolution. |
| Laundry entry door included on p5, item 54; secondary external hinged door marked N/A on p11 | Potential conflict needing door identity/plan confirmation. Do not assume the labels identify different doors. |
| Main benchtop allowance names a limited range on p7; scheme 1 names another selection on pp15-16 | Possible selection override or conflict. An appendix does not establish precedence merely by appearing later. |
| Laundry cabinetry on p6/p12 and the wet-area boards including laundry on pp16, 19 and 22 | Resolve whether scheme-specific materials replace the base cabinetry finish. Preserve each scheme independently. |
| Two floor-specific frame heights on p4/p11 | Valid scoped differences, not a conflict; frame height is not automatically finished clear ceiling height. |
| Rainwater excluded on p8; rainwater and OSD bundled as excluded on p11 | Separate the two systems and retain the source's stated reason. Do not derive an OSD requirement from recycled-water availability. |
| Specific lots excluded from new fencing on p9 because the developer has completed it | Existing physical presence, builder exclusion and project scope are different facts. |
| Electrical allowances on pp13-14 | Preserve count and extent. Do not multiply by 22 until the quantity basis is established. |

The reference also lacks a clear consolidated issue/revision/precedence
schedule, contains a numbering gap at item 11, and uses conditional wording
such as design-specific, where shown, or equivalent. Its product and standards
references need their own evidence and review. In particular, the claim about a
silica-free benchtop substitution on p7 is a source claim, not compliance
verification or an identified approved replacement product.

Visually, trade headings, room tables and finish boards are useful. The dense
small type and limited navigation are less useful. A SiteWise output should
add a contents list, stable item references, issue details, explicit exceptions
and links back to decisions. A compact RFP and a detailed specification need
different page budgets.

## Fit with the current schema

The [foundation design](../design/2026-09-29-foundation-design.md) and
[knowledge schema](../../knowledge/SCHEMA.md) already provide the right
backbone: systems, determinants, rules, interfaces and failure modes. The works,
delivery and reporting additions are substantial reuse opportunities.

| Existing capability | What it already gives us | Specific gap |
|---|---|---|
| Systems and cluster catalogues | Most trades and building components in the PDF have a home | A system label does not retain the specification of an individual component |
| Determinants and project facts | Class, energy pathway, BAL, ground class, gas, water, contractual facts | Product colours, thicknesses and room finishes should not become project determinants |
| `document_sources`, `passage_sources`, profile readings | Original text, section/context, source locations and unresolved readings | Reusable component attributes need stable identity and evidence per assertion |
| Site-owned `project_parts` | Buildings, storeys, compartments, roofs and other parts | The inspected model is flat, with site-wide unique labels; it lacks explicit room/element hierarchy and scope sets |
| `work_items` | System, action, parent/child work, target, quantity, provenance and review | One quantity and registered scalar targets do not represent a full assembly or selection schedule |
| `Target.Values`, `Target.ClauseRefs`, `Target.Text` | A useful target contract | Accepted target keys currently come from planning, determinants, facts and scale; product attributes lack a registry and field-level source/decision handling |
| `package_scope_items` | Design/document/supply/install/test/certify roles, obligations, stages and source references | Qualifiers and boundaries need links to the particular specification entry and external dependency |
| Delivery items and dependencies | Decisions, risks, approvals, activities and sequencing | Specification decisions and required inputs need stable links into these existing records |
| Report templates, clauses, edits and issue snapshots | Deterministic wording and immutable issued records | Supported report kinds are RFP/RFT/PMP; a full specification with visual selections needs an explicit output contract |

These findings come from migrations 008, 009, 011, 012, 015, 018, 019 and 020,
[`workitems.go`](../../internal/works/workitems.go),
[`target.go`](../../internal/works/target.go),
[`location.go`](../../internal/profile/location.go), and the schema. Full source
retention already exists; this plan must not replace it with short summaries.

The useful additions fall into three groups: populate missing knowledge within
existing systems; add typed specification records; refine physical scope and
resolution. Replacing the system hierarchy or adding hundreds of global
profiler fields would make the model harder to use and slower to read.

## Coverage and attributes to capture

The following mapping covers all 38 work sections, the turnkey tables and the
selection boards. System names identify existing homes; the proposed
attributes are not current implemented fields. Project-specific values belong
in project records, never as universal knowledge defaults.

| Source sections and pages | Existing home | Additional structured detail |
|---|---|---|
| 1 General, p2 | Package obligations, delivery, unforeseen conditions; `site.erosion-sediment-control` | Insurance/levy inclusion, temporary facilities, possession prerequisites, latent-condition exclusion, scope of acoustic/BAL allowances |
| 2 Utilities, pp2-3 | `site.services-connections`, hydraulic/electrical/comms systems | Connection boundary, metering per dwelling, enabling work by others, account activation/closure event, approval and capacity dependencies |
| 3 Consultants, p3 | Packages, stages, scope roles and delivery | Design/certification/inspection deliverable, issuer, input document and revision, included fee, report exclusion and consequences requiring a decision |
| 4 Earthworks, p3 | `site.earthworks`, `site.vehicle-access` | Cut/fill basis, temporary access material and depth, rock exclusion |
| 5 Foundation piers, p4 | `substructure.piles` | Pier type/material, extent, authority-dependent depth; unspecified quantity stays unspecified |
| 6 Concrete, p4 | `substructure.slab-on-ground` | Slab form, geotechnical/exposure report references, wet-area set-down location and intended finish transition |
| 7 Termites, p4 | `substructure.termite-management` | Treatment/system selection and source standard references; verification separate |
| 8 Frames and trusses, p4 | `structure.timber` | Treatment class, prefabrication, frame heights by storey, bracing, beams and wet-area step-downs |
| 9 Brickwork and 12 Render, p4 | `envelope.external-walls` | Exposure designation as stated, brick range, mortar colour, joint finish, render/coating and facade extent |
| 10 Scaffolding, p4 | `structure.temporary-works`, package obligations | Temporary access/protection, coverage and source-stated height trigger |
| 11 Cladding/eaves/soffits, p4 | `envelope.external-walls` | Material, brand/product, thickness, profile, substrate and elevation/location |
| 13 Roofing, p5 | `envelope.roof-coverings`, insulation/membranes, `hydraulic.roof-drainage` | Garage versus main roof, material/profile, insulation thickness, sarking, fascia/gutters/flashings |
| 14 Doors and 15 Hardware, p5; p11 | `envelope.windows-doors`, `interiors.internal-doors` | Door identity, storey, height/width, leaf model, glazing, frame/trim, finish, hardware set and seals |
| 16 Garage doors, p5; p14 | `envelope.vehicle-doors` | Operation/type, finish, glazing exclusion, motor, remote and transmitter quantities |
| 17 Insulation, p5; windows on p11 | `envelope.thermal-insulation`, membranes, thermal performance and windows | Requirement source, certificate version, location, insulation/glazing selection; missing performance values remain unresolved |
| 18 Plasterboard, pp5-6 | `interiors.walls-linings`, `interiors.ceilings` | Board type/thickness, fixing method, junction/cornice by room, expansion-joint conditions and stated finish level |
| 19 Waterproofing, p6 | `interiors.wet-area-waterproofing`, `envelope.external-waterproofing` | Substrate, layer order, membrane/primer/sealant, dry-film thickness, threshold angle dimensions, treatment extent and screed exclusion |
| 20 Tiling, p6; pp12-13, 15-23 | `interiors.tiling` | Room/surface, tile size, finish/colour, laying pattern, grout, height/extent and skirting-only exceptions |
| 21 Joinery and 22 Benchtops, pp6-7; pp12-13, 15-23 | `interiors.joinery` | Component role, carcass/back/door/edge material and thickness, profile, hardware, bench edge versus slab thickness, waterfall, substitution and scheme |
| 23 PC items, p7; pp12-13 | `hydraulic.sanitary`, `interiors.ffe` and service interfaces | Fixture/appliance identity, size/capacity, mounting, finish, service demand and selection status; PC heading alone does not establish an amount |
| 24 Screens and 25 Mirrors, p7 | `interiors.internal-glazing`, `interiors.ffe` | Frame/glass/finish, dimensions or extent, mounting/fixing and applicable room |
| 26 Wardrobes, p7; p13 | `interiors.joinery` | Wardrobe type, rail/shelf/tower/drawer counts and widths, material/finish and bedroom applicability |
| 27 Flooring, pp7-8; p12, pp17/20/23 | `interiors.floor-finishes` | Material/range/colour, underlay type/thickness, room/storey extent, wet-area exclusions and stair inclusion |
| 28 Paint, p8; pp17/20/23 | `interiors.painting`, `envelope.external-walls` | Manufacturer/system, coat count, colour code, gloss, substrate/surface, staining exclusion |
| 29 Hot water, p8/p11 | `hydraulic.hot-water`, `hydraulic.gas` | Unit type, model if stated, flow-rate basis, recess/enclosure, per-dwelling extent and gas availability assumption |
| 30 Rainwater, p8/p11 | `hydraulic.non-potable-water`, `site.stormwater-detention` | Separate rainwater and detention inclusion, recycled-water premise, required civil/approval input |
| 31 Electrical, p8; pp13-14 | `electrical.cabling-wiring`, lighting, smoke alarms and comms systems | Fitting type/count/quantity basis, finish, room/location, wiring category, outlets, conduit and excluded equipment |
| 32 Air conditioning, p8/p11 | `mechanical.air-conditioning`, air distribution and controls | Brand/model verbatim, phase, zones, outlet type and extent; model text is not a calculated capacity |
| 33 Softscape, p8/p14 | `site.landscape`, `site.trees` | Species/product, turf/soil/mulch depth, garden/lot/verge extent, edging, tree exclusions and maintenance duration/start event |
| 34 Hardscape, p9/p14 | `site.vehicle-access`, `site.landscape`, `site.public-domain` | Surface/colour/seal, path/crossover location, lot exceptions, joints and grading exclusions |
| 35 Fences/gates and 36 Retaining, p9 | `site.landscape`, site/substructure retaining walls | Height/material/colour, gates/accessories, boundary side, completed-by-others exceptions, sleeper allowance and survey dependency |
| 37 External fitments, p9/p14 | `interiors.ffe` with external scope; `access-egress.barriers` | Clothesline/letterbox product and mounting, location, porch/alfresco barrier exclusion; review external-fittings routing |
| 38 Window coverings, p9/p11 | `envelope.windows-doors`, `interiors.ffe` | Insect-screen type/mesh/frame/closer, blind range/colour, opening applicability |
| Three schemes, pp15-23 | New project selection sets linked to the same specification entries | Scheme identity, product/code, colour/finish, image-region reference and explicit assignment to dwelling/room |

Prioritise a finite attribute registry around these families. Examples are
material, product name/code, finish, colour/code, thickness, height/width,
installation method, substrate, quantity/basis, extent, performance value,
source document dependency and substitution status. Only define keys with an
actual source case and consumer.

Do not split every current system into new leaves. Joinery already includes
benchtops and wardrobes; FF&E already includes appliances, mirrors and blinds;
landscape includes paths and fences. Correct routing descriptions where needed
(especially external fittings) and prove ambiguous cases before adding a leaf.
System subdivisions are justified by different rules/interfaces or useful
independent scope, not by one manufacturer's catalogue.

## Proposed schema contract

### Physical scope

Extend the existing part identity rather than creating a competing building
tree. Add a parent relationship and the minimum room/element/lot-area kinds
needed by fixtures. Parent references must stay within the same organisation
and site, be acyclic, and allow two buildings to each contain a Kitchen.
Retain existing IDs and references during migration.

Add a bounded scope expression for specifications: explicit part IDs, named
groups, component/surface, included and excluded subsets, original extent
wording and resolution status. For example, all first-floor areas except wet
areas can be retained before the room inventory is complete. Do not fabricate
rooms to make that expression resolve. Store the resolved membership and its
version so an issued schedule remains reproducible.

Keep ownership scope separate from physical scope. Proposed products and
allowances belong to a project; they must not automatically become permanent
site facts or carry to another project as as-built evidence.

### Specification entries and assertions

Use one entry for a specified component or assembly attached to a work item.
Its effective values are a projection over evidence and owner decisions.
Proposed names below describe contracts; migration numbers are deliberately
not allocated while other work is active.

| Record | Minimum contents |
|---|---|
| Attribute definition in `knowledge/specifications/attributes.yaml` | Stable key/version, label, scalar type, unit family, allowed component/system contexts, extraction triggers, validation and source |
| Project specification entry | `org_id`, project, work item, physical scope, component kind, optional parent assembly, stable ID/version |
| Attribute assertion | Entry/key, raw source value, typed value/unit when parseable, basis/qualifier, assertion meaning, selection set, source document/hash/page/region or passage/span, reading/question version and uncertainty |
| Accepted value or decision | Existing provenance/review conventions, selected assertion or user value, actor, reason, version and supersession link |
| Specification projection | Current effective values, alternatives, missing inputs, conflicts and scope completeness; precomputed and paginated |

Keep source assertions separate from accepted design requirements. Reuse the
existing `origin`, `meaning`, review and proposal lifecycle instead of creating
a second approval engine. Add explicit roles where existing meanings are
insufficient: required performance, proposed/provided selection and observed
result. An accepted pricing allowance is still an allowance.

For numbers, retain what the number describes: nominal/frame/clear height,
edge build-up versus slab thickness, per-dwelling versus project quantity,
minimum/maximum/approximate, and required versus proposed performance.
Structured dimensions can be separate registered attributes sharing a source
group; avoid an unrestricted nested JSON property bag.

Existing `Target.Values` remains compatible. An initial reader can project
new accepted attributes alongside targets; do not put every selection into
`det.*`, expand global profile rows, or silently reinterpret old keys. The
profiler shows summaries and open decisions; detailed specification records
live under the relevant scope/work item.

### Selection sets and visual evidence

Represent each finish scheme as an alternative set of selections, with an
explicit assignment to a physical scope. Unassigned schemes are options, not
three conflicting installed finishes. Do not infer assignment from page order
or blend products between schemes.

A product selection can retain verbatim brand, range, product/model code,
colour, finish and dimensions without requiring a central product catalogue.
Record an alternative or substitution with its proposed replacement,
equivalence basis, supporting evidence and decision. Silence does not permit
substitution.

Use the existing content-hash file store for original media. Link a swatch to
its PDF page/rectangle and caption, or to an owner-supplied asset; enforce org
access on every read. A rendered colour sample is illustrative, not measured
colour evidence. Textless or ambiguously associated images need mapping; do
not introduce another AI provider to interpret them.

### Commercial boundaries and dependencies

Preserve separate fields/relations for physical inclusion, pricing inclusion,
responsible party, activity role and outstanding condition. A builder can
exclude a connection while another party provides it; a contractor's exclusion
cannot remove the building's requirement.

Reuse package roles and obligations for design, documentation, supply,
installation, testing and certification. Link them to the relevant entry and
physical boundary. Represent missing energy certificates, civil pits, utility
supply and authority confirmation as required inputs/dependencies tied to
existing delivery decisions or actions. A reference to an absent report is not
evidence of its findings.

Use cost-plan records for money if a source actually states an amount and
basis. This reference's PC heading must not create a zero allowance, provisional
sum or price. Maintain unknown values visibly.

### Resolution and regulatory evidence

Compare assertions only when component, attribute, physical scope, scheme and
meaning overlap. Distinct floors and alternative schemes are valid differences.
Two incompatible values at the same scope remain unresolved unless an explicit
approved override or supersession resolves them. A more specific location can
express an exception; its specificity alone does not grant contractual
precedence between documents.

Rules remain separate from product specifications. A source-stated standard,
clause or regulatory threshold retains its locator and `verified: false` until
the instrument supports that precise claim. A product dimension may be
accurately extracted without being a verified regulatory requirement; retain
that distinction. An owner's selection decision does not verify a rule or
product compliance. Existing
determinants should be reused for BAL, gas supply, site class and energy
pathway, retaining the allowance/requirement distinction.

## Output contract and architectural RFP

The current [architecture catalogue](../../knowledge/rfp/architecture.yaml)
already includes specifications, door/finish schedules, typical interiors,
coordination and an input/decision register. Its stated evidence base is a
camp/masterplan, fit-out and warehouse proposal. Residential detail therefore
needs a deliberate module, rather than more generic coordination wording.

For a custom-home appointment, ask the client to select the following
deliverables and allocate who produces and coordinates them:

| Deliverable | Completion test |
|---|---|
| Room and finish schedule | Every agreed room/surface has a selection or an identified decision; local exceptions are visible |
| Door/window/hardware schedule | Opening identity, dimensions/basis, glazing/finish/hardware and energy-report dependencies coordinate |
| Kitchen, laundry, bathroom and wardrobe documentation | Layout/elevations, joinery, bench, appliance/fixture selections and service interfaces agree |
| Wet-area and threshold details | Set-down, substrate, membrane, screed/tile and adjoining finish are coordinated; required evidence is identified |
| External material and works schedule | Roof/facade/soffit, driveway/path, fences/gates and landscape boundaries are covered, with specialist ownership |
| Selections and substitution register | Scheme/selection assignment, approvals, open choices and alternatives are explicit |
| Specification and coordination review | Inclusions, exclusions, required inputs and apparent contradictions reconcile with drawings and specialist documents |

The RFP procures these services at the appointed stages and with explicit
allowances for iterations, reviews and meetings. It should not embed all of the
builder specification, prescribe these source products, or appoint excluded
specialists automatically. Building class helps establish regulatory context;
desired documentation depth also depends on the brief and selected services.
Do not make detailed interiors optional solely because a project is a house.

The eventual outputs share the same specification entries:

1. A trade/worksection view for pricing and construction.
2. A room-by-room inclusions and finish schedule for client decisions.
3. Assigned visual selection boards with traceable captions.
4. A compact unresolved-input, exclusion and responsibility schedule.

Use the existing RFT report infrastructure for the first draft specification
schedule, adding an explicit appendix/attachment contract and renderer; the
current inspected code does not establish a complete attachment workflow.
A standalone `specification` report kind is a later bounded change
to report validation, templates, clause outputs, snapshots and export. It has
its own page limits and issue checks; the three-page RFP gate stays intact.

The classification-view plan can supply optional NATSPEC/ANZSMM grouping when
implemented. Those codes remain presentation/crosswalk metadata. They do not
replace system IDs, physical scope, work-item identity or quantity ownership.
This source's section numbers are its own, not asserted NATSPEC numbers.

## Extraction and generation within the Jev constraint

Code preserves clause boundaries, table headers/cells, continuation context,
captions and selection-set boundaries, then over-finds candidate values and
identities. Jev selects among those candidates or not-stated. Code copies the
source value and validates units, scope and references. Unknown product text
can remain an exact source span before it is decomposed into attributes. This
follows [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).

Extend relevant existing background reading states with atomic, narrowly
routed questions. Send all questions for a state in one
[fan-out](https://docs.typesafe.ai/patterns/fan-out); avoid a model call per
attribute or a serial selection/validation loop. Keep candidate sets within
the API limit by deterministic source-unit boundaries, with explicit overflow
and Needs mapping rather than silent truncation. Do not send every part and
every product in the project to every question.

Calibrate thresholds by question and option shape; Noul probability is not
Choice confidence. See [confidence](https://docs.typesafe.ai/confidence).
Code performs counts, conversions, comparisons and template assembly; Jev
does not write specifications or infer engineering adequacy. See
[Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
Use the existing pinned model and client contract with the
[API](https://docs.typesafe.ai/api). Existing building logic and
[foundation responsibilities](../design/2026-09-29-foundation-design.md)
remain binding.

All full-specification reading runs after filing, with existing checkpoints,
versioned fingerprints and reserved intake capacity. Opening the profiler,
selecting a scheme and assembling a report use saved data. Approved template
wording and typed values compose the output; source wording can be retained
with attribution for review. No new service, runtime AI or dependency is
proposed.

## Implementation sequence

All implementation is lane A because scope and specification statements are
information a user trusts. The outcome is a reproducible residential
specification with explicit unresolved choices. It improves answer trust and
time-to-decision while retaining filing speed. Each phase is a separate,
reviewable slice; recheck the other agent's landed changes before implementation.

| Phase | Work and likely files | Done means |
|---|---|---|
| 1 Establish the reference contract | Add a PDF-specific private fixture/manifest record, source-section coverage ledger and hand-labelled examples; document new shapes in `knowledge/SCHEMA.md` | All 23 pages, 38 sections, table rows and scheme panels are accounted for; labels distinguish extraction, uncertainty and accepted scope |
| 2 Prove physical scope and attribute storage | Additive migrations; extend part handling, `internal/works` target integration and attribute catalogue/validation | Doors at different floors/rooms keep different values; assertions retain source and basis; repeated room names and unresolved scope work |
| 3 Extract and reconcile four representative cases | Extend source parsing/reading and saved projections for doors, wet areas, joinery/selections and utility boundaries | Source values survive, known conflicts are raised, alternatives stay separate, builder exclusions do not erase project need |
| 4 Complete residential attribute coverage | Expand the finite catalogue using the coverage table; connect package roles, required inputs and delivery decisions | Every substantive source entry maps, is deliberately out of scope with reason, or remains visibly unresolved; no generic fallback claims full understanding |
| 5 Render one specification schedule and its room view | Extend report snapshots/templates/assembly with an explicit RFT draft appendix/attachment contract; render and visually inspect | Both views use the same accepted values, list exclusions/dependencies, and retain source references and issue basis |
| 6 Add visual selections and standalone output | Media-region associations, set assignment, substitution review and explicit report kind | Three alternative source schemes remain distinct; selected boards and full specification export from the same snapshot |

The architectural RFP module can proceed alongside phases 1-2. Give that work
the deliverables table and the four representative cases; coordinate shared
schema changes after its current work lands. No other chat is messaged and no
project is edited as part of this review.

Out of scope: a complete BIM authoring model, geometric take-off, manufacturer
catalogue sync, automatic compliant design, a new legal rules engine, copying
all historical selections into global knowledge, or replacing the profiler.
All new knowledge authored from the reference remains `status: draft`; only
the owner promotes it.

## Validation and speed gates

The golden reference needs two answer keys: what each source states, including
contradictions, and what an approved output is allowed to say. No output answer
key should resolve an owner decision by guessing.

Required cases:

- The door-height conflict remains open until an explicit decision; different
  floor heights remain separate and preserve frame-height basis.
- Scheme 1/2/3 differences are alternatives; an unassigned scheme never becomes
  the installed specification. A scope change flags affected draft selections.
- Quantity basis is retained: a fitting allowance is not multiplied by dwelling
  count; a model number is not a capacity; bench edge thickness is not slab thickness.
- Existing fencing by others, no price allowance, not required, unknown and
  not in the completed project remain distinguishable.
- OSD and rainwater remain separate. An absent referenced energy/geotechnical
  report leaves its input unresolved. A BAL allowance cannot establish assessed BAL.
- Page/table continuation and image captions survive. Blank-text or unmapped
  regions remain visible; page processing is not claimed as semantic accuracy.
- Re-reading is idempotent, source revisions invalidate only dependent draft
  projections, user decisions survive refresh, and issued snapshots stay immutable.
- Wrong-org/wrong-site references, missing files, deleted sources, invalid units,
  cycles, ambiguous locations, oversized inputs and Jev timeout fail visibly and
  safely without blocking filing.
- Residential fixtures plus existing Hale/Petersham/Newham and non-residential
  cases still pass. The pilot has zero unsupported automatic selections, lost
  exclusions or missed labelled conflicts; unresolved cases are counted separately.

Keep private source text, product schedules and recordings outside tracked
fixtures. Public tests can use synthetic equivalents. The corpus manifest
already records that this repository is public.

The current [budgets](../../bench/budgets.json) provide these gates, in
milliseconds. New-path numbers are proposed budgets, not measurements.

| Path | p50 | p90 | Basis |
|---|---:|---:|---|
| Filing from upload complete | 1000 | 2000 | Existing whole-intake gate; unchanged during background extraction |
| Profile read/edit | 50 | 150 | Existing profile gates |
| Profile rebuild | 100 | 300 | Existing deterministic rebuild gate, not full document reading |
| Specification page read / single selection edit | 100 | 250 | Proposed, aligned with works/report reads and writes |
| Draft schedule assembly | 300 | 1000 | Existing report-assemble gate; use a bounded representative fixture |
| Fetch already prepared export | 100 | 250 | Existing report-export gate |
| Start background specification preparation | 100 | 250 | Proposed enqueue/acknowledgement gate |

Benchmark with at least the existing minimum of 20 samples, both a single-home
fixture and the 22-dwelling/scoped-exception reference shape, with concurrent
intake. Add new paths to the build-failing budget harness. Full background
extraction and visual PDF preparation need separate workload completion
measurements and a proposed gate in their build slice; they must not be hidden
inside the fast read/export acknowledgement timing.

After knowledge changes run `python tools/check_knowledge.py` (and the strict
gate required by the relevant build slice). Run the affected Go tests, existing
answer keys, report/page-fit checks, org-isolation tests and speed harness.
This review makes no runtime change, so no new latency measurement or passing
implementation test is claimed.

## Recommended first build

Start with scoped doors, wet areas, joinery/finish schemes and utility
boundaries. Together they exercise the important new shapes: measurements,
assemblies, alternatives, contradictions, exclusions and dependencies. Produce
one draft schedule and its room view from those records before extending the
catalogue across the remaining trades.

This keeps the building model intact, gives the architectural RFP a concrete
deliverable to commission, and proves that SiteWise can preserve the detail
that makes the reference useful without importing its ambiguities as facts.
