# Materials and finishes schedule review

Date: 14 October 2026. Status: proposed schema requirements for owner review.

The supplied architect's schedule is a strong reference for the detail and
presentation of selections. It extends the Newham review in a significant
way: a specification needs reusable item definitions, their applications in
the building, and relationships between products, finishes, components and
choices. A flat list of product attributes would lose much of its meaning.

The [general project specification plan](../plans/2026-10-14-project-specification-schema.md)
incorporates these findings. The [earlier residential review](../plans/2026-10-13-residential-specification-schema.md)
retains the 38-section analysis of Newham's broader construction specification.
Together, the references describe complementary outputs: construction scope
and execution detail, plus a precise materials, finishes and fittings schedule.

## Source and issue status

Private source: `Golden/FFE SPEC/S2-A2 [K] MATERIAL & FINISHES SCHEDULE.pdf`,
by Neeson Murcutt Architects. The PDF has 22 pages and SHA-256
`ea86fe3ee1080dac0d9fcef211f38a41b44f083666379f6f2d14350884e9b1a4`.
Page references below use PDF page positions. Text from every page and rendered
pages were reviewed, including the visual revision and deletion markings.
Raw extracts, page images and contact details remain outside tracked documents.

The source identifies issue K, dated 30 January 2020, **For 75% Preliminary
Design**. It is a reference for information quality and structure; its products,
prices and design choices are not defaults for Custom Home 2. Instructions
inside it are source requirements to record, not instructions to this agent.
Its source-stated ratings and standards have not been independently verified.

| Source section | Pages | Coded row occurrences | Information shape |
|---|---|---:|---|
| 2.1 Internal and external finishes | 2–11 | 71 | Materials, coatings, assemblies, components and references to other schedules |
| 2.2 Finish upgrade options | 12 | 2 | Alternative island benchtops with restricted application |
| 2.3 Lighting | 13–14 | 9 | Product, finish, performance, controls, mounting, location and price evidence |
| 2.4 Appliances | 15 | 6 | Base selections and integration requirements |
| 2.5 Appliance upgrade options | 16–17 | 10 | Alternatives, handed products, accessories and conditional cabinetry |
| 2.6 Fixtures and fittings | 18–22 | 33 | Sanitary fittings, hardware, dimensions, finishes and supplier information |

There are **131 coded row occurrences and 128 distinct displayed item labels**.
This counts the two struck-through rows, repeated CF, and FG1/FG2 reference and
definition rows. It excludes the uncoded switchplate note and unlabelled child
rows. It is a structural inventory, not a count of selected or installed products.

## What SiteWise should adopt

**A compact, consistent item grammar.** Codes, descriptions, technical
specifications, suppliers, comments and images sit together. Later sections
make finish/colour and location more explicit. A reader can connect the
selection to its appearance and purchasing information without reconstructing
it from prose. SiteWise should support configurable views over shared fields,
rather than permanently fixing every category to one wide table.

**Reusable codes and explicit relationships.** Paint, powdercoat and protective
coating definitions are referenced by other items. The value lies in the link:
changing a finish should identify every affected application. Displayed codes
must remain recognisable, while stable internal identities handle collisions,
references and revisions.

**Specific installation and coordination detail.** The schedule records
substrates, coatings, fixing arrangements, prototypes, installer requirements,
setting-out dimensions and appliance clearances. These details turn a selection
list into useful instructions for pricing and execution. They need their own
structured fields or linked requirements, rather than one undifferentiated note.

**Alternatives with context.** Light, dark and concrete schemes, island-only
upgrades and location-dependent inclusions show that selection is conditional.
SiteWise must make the resulting specification for each room or area explicit.
Showing all options on a board does not establish which option has been chosen.

**Images beside specifications.** Image references help users recognise and
compare items. Preserve a source crop or authorised product image with a
caption and its item association; an image is illustrative evidence and cannot
prove dimensions, colour accuracy or compliance.

## Where SiteWise should improve the shape

The schedule mixes materials, whole products, assemblies, finishes and
accessories in one code list. That is convenient for a printed document, but
the underlying records should distinguish these kinds. Physical systems remain
the organising model; the architect's schedule sections become output views.

An item can apply at several locations and be included at one location but an
upgrade at another. Keep its reusable definition separate from each application.
P1 also specifies different sheen by surface: walls, ceilings and doors cannot
inherit one undifferentiated finish value (p6).

Quotes, prices and contact details are useful, but their bases differ. The
source contains unit rates, a whole-project quote, supply-only amounts, retail
prices and both explicit and unstated GST treatment. They must remain commercial
evidence until a user deliberately adopts a compatible amount into a cost plan.

The preliminary issue contains open decisions, referenced documents and visible
changes. SiteWise should expose these as an actionable list with responsible
people and dependencies. A beautiful schedule can still be incomplete; an
output's issue purpose should state how far its information has been resolved.

## Coverage and attribute gap analysis

Most families already have a physical system home. The gaps below are primarily
in project specification records and relationships, not missing building systems.
The listed homes are routing destinations, with interfaces where an item spans
systems; they are not instructions to create a new system for every item code.

| Family and source codes | Pages | Existing model home | Attributes or relationships to capture |
|---|---|---|---|
| Battens and balustrades: AB, BAL1, HR1 | 2, 5 | Envelope, joinery and relevant barrier/stair systems | Profile, material, dimensions, spacing, finish reference, mounting and detailed drawing dependency |
| Masonry: BR1–BR3 | 2–3 | Envelope external walls | Multiple constituent products, blend proportions and basis, mortar, joint profile, support, sample panel and approval |
| Benchtops: BT1(A/B), BT2, BT1(C/D) | 3, 12 | Interiors joinery | Material/product, thickness, edge, colour, installer credential, scheme and island-only upgrade scope |
| Fibre cement and sheet linings: CFC, LIN1, PB1/2 | 3, 6 | Envelope and interiors walls/ceilings | Substrate, sheet thickness, board property, finish level, suspension/build-up and linked paint |
| Concrete finishes: CONC1–CONC4 | 3–4 | Structure concrete and floor finishes | Substrate versus finish, surface treatment, flatness tolerance with measurement span, slip classification and test-method reference |
| Carpet and timber flooring: CPT1, TF1(A/B) | 4, 10 | Interiors floor finishes | Product, colour, board sizes, grade, underlay candidates, installation, finish and scheme |
| Drainage and flashings: DP, GT1, FL1 | 4–5 | Hydraulic roof drainage and envelope flashings | Profile, size, material/finish and hydraulic or detail-drawing dependency |
| Ventilation components: EH COWL, GD1, EH | 4, 19 | Mechanical local exhaust and air distribution | Product, size, location, compatible finish, mounting, linked grille/fan/cowl and service dependency |
| Access and vehicle doors: AP1, RD1 | 2, 7 | Relevant ceilings/walls and envelope vehicle doors | Clear opening, panel/door type, frame, coating, access location and reflected-ceiling-plan dependency |
| Joinery accessories: DWR, GS, HNG1/2, PBK1, BIN1 | 4–5, 7, 18 | Interiors joinery | Accessory model, extension/movement, door limits, body size versus minimum cabinet size, mounting and deletion status |
| Decorative surfaces: LAM1, TV1(A/B), SK1/2 | 5, 8, 11 | Interiors joinery and floor/wall finishes | Laminate/veneer species or grade, grain direction, cut, edge, substrate, profile and scheme |
| Paint and powdercoat: P1–P3, PC1/2, PO1/2 | 6–7 | Interiors painting and applicable envelope finishes | Manufacturer/system, colour, sheen by surface, preparation, application and links from finished components |
| Steel and screens: POS1, PS1, SP1 | 7, 9 | Structure steel and envelope components | Structural dependency, ordered coating recipe, per-coat thickness, preparation, applicator qualification, samples and requested warranty |
| Render: RE1/2 | 8 | Envelope external walls | System layers, substrate, texture, colour and application requirements |
| Roof systems: RF1–RF3 | 8 | Envelope roof coverings and membranes | Covering, fixings, membrane, colour, assembly relationships and prototype requirement |
| Signage: S1 | 8 | Site signage | Sign type, material, lettering/detail, mounting, finish and drawing reference |
| Glazing and mirror: MIR, SHS | 6, 9 | Interiors glazing and relevant wet-area components | Glass type, thickness, edges, backing, support and application |
| Decks and stairs: TDK1(A/B), TIM1(A/B) | 9–11 | Site/deck structure and timber/interior finishes | Timber selection, treatment, coating, joists/pedestals, fixing, profile, dimensions and scheme |
| Tiles: TIL1(A/B), TIL2, TIL3(A/B) | 10–11 | Interiors tiling | Stock versus cut size, laying direction/pattern, application, grout, scheme and linked visual evidence |
| Lighting: LF1, LF2(A/B), LF3–LF8 | 13–14 | Electrical lighting | Product/finish, lamp or strip, power with basis, colour temperature, IP claim, control requirement, driver and mounting datum |
| Switchplates: uncoded note | 14 | Electrical wiring and accessories | Product family and finish, without requiring a source code to retain a valid item |
| Base appliances: CF, CK1, DW1, MW1, OV1, RH1 | 15 | FFE, mechanical and electrical interfaces | Manufacturer/model, product dimensions, integration, controls, finish and location |
| Optional appliances: BBQ, CD1/2, CF, HTR, REF1/2, VIN, WM1/2 | 16–17 | FFE and service interfaces | Alternative groups, handing, integration kit, cavity/clearances, power/plumbing dependencies and fallback cabinetry |
| Baths, basins and sanitary fittings: BA1, BAS1/2, BM1/2, BTR, HS, SHM, SI1/2, SM1, SR, WC, WT1 | 18–22 | Hydraulic supply/sanitary and interiors wet areas | Product dimensions, finish, mounting, spout/reach, waste or trap, capacity, WELS claim/registration and selected flow option |
| Floor wastes: FG1/2 references and definitions, FG3 | 4, 19 | Hydraulic sanitary and wet-area waterproofing | Grate/insert, length/width/depth, outlet, site-measured size and waterproofing interface |
| Handles and stops: DS, H1–H4 including H2(A/B) | 19–20 | Internal doors and joinery | Profile, length, finish reference, fixing, application and scheme |
| Wet-area accessories: FP, GP, SD, SSH, TBH1, TR, TRH, TW, WMT | 19–22 | Interiors FFE/joinery and wet-area interfaces | Product, finish, dimensions, attachment, placement and source revision/deletion status |

Every coded family in the six sections is represented above. A displayed
NATSPEC reference is a classification reference distinct from an item code or
manufacturer model. Its source version and verification state must be retained;
it does not establish a regulatory requirement.

## What changes from the first plan

The first plan already proposed scoped attributes, source assertions, accepted
decisions, selection sets, image references and commercial exclusions. Those
remain useful. The schedule shows where those ideas need a more precise contract.

| Topic | Earlier proposal | Required refinement |
|---|---|---|
| Record identity | Entry attached to a work item | Reusable type can exist before work is assigned; applications link it to locations and work items |
| Selection sets | Schemes and alternatives | Separate schemes from optional upgrades; support local inclusion and explicit fallback work |
| Assemblies | Optional parent assembly | Typed component/finish links, ordered layers, blend proportions and required accessories |
| Value structure | Scalar, range, units and qualifiers | Dimension role/datum, rate basis, performance method and scoped surface exceptions |
| Commercial evidence | Allowances, exclusions and basis | Supplier roles, quote lines, aggregate scope, tax state and no automatic budget posting |
| Quality obligations | Dependencies and responsibilities | Samples, prototypes, installer credentials, hold points and requested versus offered warranties |
| Source trace | Hash, page, region/span | Row continuation, image associations, visual deletion/change markings and durable anchors across re-extraction |
| Output | RFT appendix first; standalone later | Standalone authoring and schedule output in the first useful release, independent of consultant procurement |
| Input workflow | Primarily document extraction | Direct entry and fragmented evidence must be equally capable, including projects with no architect |
| Applicability | Residential specification | Shared system/component/work-action records with category-specific attribute profiles |

## Findings that must survive import

| Source example | Meaning to preserve | Expected application behaviour |
|---|---|---|
| P1, p6: different sheen for walls, ceilings and doors | One finish family with application-specific properties | Show the correct sheen in each application; do not flatten to one value |
| BR1, p2 and BR3, pp2–3: 90/10 and 40/60 brick blends | Constituent proportions, not competing alternatives | Preserve both products and the stated basis; calculate totals only when the basis is known |
| SP1, p9: three coatings at 75/200/100 microns | Ordered layers with per-layer thickness and preparation | Preserve order and each thickness; do not replace the recipe with a single sum |
| TDK1, pp9–10: decking, finish, joists and pedestals continue over a page | One assembly with several component rows | Attach continued rows to their parent with reviewable page anchors |
| TIL1(B), p10: dark bathroom and concrete-scheme living applications | Scheme and location both affect meaning | Preserve the concrete scheme as well as light/dark choices |
| BT1(C/D), p12: island upgrades | Limited application | Do not change every benchtop when the upgrade is selected |
| LF4, p13: kitchen inclusion and robe-interior upgrade | Commercial inclusion belongs to the application | One product definition can have different inclusion states by location |
| VIN, p17: cupboard if wine fridge is not selected | An option changes the physical work | Preview the cabinetry consequence and require an explicit resolved choice |
| VIN, p17: 595 mm product width and 620 mm opening | Product size differs from installation envelope | Store distinct dimension roles; do not flag these values as a direct conflict |
| BIN1, p18: 346 mm body and minimum 400 mm cabinet | Fit constraint | Preserve a minimum constraint separately from product dimensions |
| LF5, p14: mounting at 1600 above finished floor | Mounting height has a datum | Do not treat it as product height or a storey height |
| TIL1/3, pp10–11: stock and cut sizes, laying patterns | Procurement size differs from installed module | Keep both dimensions and the cutting/laying instruction |
| CONC1, p3: flatness over a measurement span; slip classes | Tolerance and performance method matter | Preserve ±3 mm over 3 m, and distinct source-stated slip classifications without equating methods |
| BM2, p18 and SR, p21: alternative rating/flow combinations | Paired options | Do not combine one option's rating with another option's flow |
| LF1, p13: unit price and whole-project quotation | Different price scopes | Keep the quoted total separate from the unit rate; infer neither quantity nor current validity |
| PS1, p7: sample approval, applicator requirement and 20-year warranty request | Quality obligations have actors, timing and evidence | Record what is required separately from submitted proof or an offered warranty |
| GS, p4 and TBH1, p21: struck-through rows | Text remains extractable although visually withdrawn | Exclude from a proposed current selection pending markup review; retain source history |
| Green revision highlights and issue K legend | Visual change indicator without prior-issue comparison | Preserve the indicator; do not invent an issue-to-issue difference |
| CF, pp15–16; FG1/2, pp4 and 19 | Reused option code versus repeated reference to one type | Resolve with section/option context; do not merge solely by code or create a product per mention |

## Open questions in the reference

Some ambiguity is expected at 75% preliminary design. The importer should make
it visible rather than treating the polished layout as evidence of completion.

- TF1(A/B), p10, mentions 4 mm R44 underlay and also a separate 5 mm Regupol
  entry. These may be alternatives, an inconsistency or a build-up needing
  explanation. Do not choose one or automatically create a 9 mm assembly.
- BTR, p19, and HS, p20, show Brushed Platinum in the finish column while the
  detail includes Brushed Nickel. Preserve both assertions for review.
- FG1, p19, leaves depth blank. Blank is unknown, not zero or not applicable.
- Site-measured lengths, custom sizes, TBC prices and referenced consultant
  information need explicit unresolved states.
- RF1/2, p8, calls for a prototype with dimensional wording that should remain
  in its source form until its intended size is confirmed.
- Doors, windows and hardware expressly reference other schedules (p1).
  Referenced drawings, structural information and service design remain
  dependencies until the relevant document and revision are available.

The source's request for architect approval does not appoint an architect to a
project that has none. Retain the required role and ask the project to identify
the authorised approver; do not silently substitute the owner or PM.

## Acceptance evidence for implementation

Use this private PDF as a golden fixture for the cases above, with the exact hash
and page anchors. Add synthetic equivalents to the public test suite so private
products, project details and contact information are not committed.

The acceptance target is semantic fidelity: correct types and applications,
relationships, unresolved choices and traceable output. Row-count agreement is
only a coverage check. Tests must also prove that no price becomes a budget, no
optional product becomes selected, no struck-through item becomes active, and
no source rating becomes verified merely because it was extracted successfully.

These two residential references establish a strong residential baseline.
Commercial and industrial applicability must also be demonstrated with separate
fixtures covering their performance, equipment and existing-work requirements,
as set out in the general plan.
