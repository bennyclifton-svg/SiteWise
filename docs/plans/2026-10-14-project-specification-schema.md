# Project specification schema and implementation plan

Date: 14 October 2026. Direction: shared across project and work types.
Implementation authorised by the owner on 14 October 2026.

Implementation status: the first working vertical slice now exists in the
`Specifications` project view. See the [implementation and verification record](../evidence/2026-10-14-specification-implementation.md)
for what is implemented, measured and still outstanding. This document remains
the broader target; its five slices are not all complete.

SiteWise should let a project team build and maintain its own specification,
whether it receives a finished architect's document or starts with scattered
notes, product sheets, quotations and direct choices. Both routes should produce
the same structured project records and the same materials, finishes, fittings
and work specifications.

This general plan supersedes the implementation sequence and residential-only
scope of the [13 October plan](2026-10-13-residential-specification-schema.md).
Its 38-section Newham gap analysis remains supporting evidence. The additional
[materials and finishes review](../evidence/2026-10-14-materials-finishes-schema-review.md)
maps all six sections of the architect's schedule, including 131 coded row
occurrences, and records the new gaps and import acceptance cases.

## What the schema does in simple terms

The schema is the application's organised record of **what is required, what
has been chosen, where it applies, how it is to be installed and what remains
undecided**. Each value can point back to a document or a person's decision.

For example, a user chooses a tile, attaches its product sheet, assigns it to
two bathrooms and adds a supplier quote. SiteWise keeps the tile definition
once, records each bathroom's application, and captures the size, finish, grout,
laying pattern and installation requirements. If grout has not been chosen,
the draft says so. A later architect's schedule can add evidence or propose a
different choice without erasing the user's decision.

The same records can generate a code schedule, room schedule, visual selection
board, work-section specification or procurement appendix. Editing the accepted
selection updates the next draft of each relevant view. Previously issued
documents remain immutable records of what was issued at the time.

This supports a user without an architect. SiteWise organises evidence, asks
the relevant questions and assembles approved wording. It must distinguish a
user's choice from verified technical adequacy and leave specialist dependencies
visible when they have not been resolved.

## The two equal input routes

| Route | User experience | Result |
|---|---|---|
| Import a completed specification | File the document; review extracted items, their locations, choices, references and unresolved readings; accept changes | Traceable proposed and accepted project records |
| Build a specification progressively | Add an item directly; choose where it applies; enter known values; attach notes, sheets, images or quotes; resolve gaps over time | The same project records, with user decisions and supporting evidence |
| Combine both routes | Import revised information into an existing authored schedule; inspect differences and conflicts | Existing accepted choices remain intact until changes are deliberately accepted |

Direct entry requires no document and no consultant package. A product sheet
usually supplies capabilities and identity, a quote supplies offered commercial
terms, and a user's instruction supplies a decision. Uploading any of these does
not automatically select the product for every matching location.

The authoring interface should offer an item list and a location view, with
compact item details, supporting evidence, alternatives and missing decisions.
Relevant fields depend on the item and work context. Users can save partial
records and issue preliminary drafts without answering every possible question.
An unresolved construction dependency remains visible in an issued preliminary
schedule; it cannot silently become an approved construction requirement.

## Domain boundaries

Keep systems, determinants, rules, interfaces and failure modes as the building
model described in [the foundation design](../design/2026-09-29-foundation-design.md)
and [knowledge schema](../../knowledge/SCHEMA.md). Specification records attach
detail to that model. Consultant disciplines, NATSPEC, ASMM and printed schedule
sections remain views or mappings.

Building use, legal building class, project work type and an individual item's
work action are separate concepts. An optional product upgrade is also distinct
from the `upgrade` work action. A factory refurbishment can contain retained
equipment, new partitions, repaired coatings and removed services in one project.

Do not add a determinant for every colour, SKU or finish. Determinants answer
questions that alter applicable rules or scope; project specification attributes
describe selected products and execution requirements. Source project choices
must not become global knowledge or defaults for other organisations.

## What exists and what needs extending

| Existing capability | Retain and reuse | Required addition |
|---|---|---|
| Building systems, interfaces and work actions | Physical organisation and work semantics | Attribute profiles for relevant components, systems, actions and stages |
| Parts and work items | Project scope and actual work | Part hierarchy and room/zone scope; type applications linked to parts/work |
| Work target values | Existing plan/determinant/fact targets | Separate typed specification attributes; do not turn target key validation into an arbitrary property bag |
| Source pages/passages and profile readings | Evidence, uncertainty and review conventions | Stable item/attribute anchors, row/visual context and per-assertion meaning |
| Package scope and delivery records | Responsibilities, approvals, decisions and dependencies | Links from individual specification requirements and applications |
| Cost plans and cost references | Accepted budgets, commitments, claims and their audit | Quoted product/rate evidence with explicit scope; deliberate transfer into existing cost workflows |
| Report templates, edits and issue snapshots | Deterministic assembly and immutable issuance | Standalone specification/schedule kinds, typed rows, images and revision comparison |

Relevant code entry points include `internal/works/workitems.go`,
`internal/works/target.go`, `internal/store/sources.go`,
`internal/jobs/source.go`, `internal/identity/pdf.go`,
`internal/store/package_scope.go`, `internal/store/cost_references.go` and
`internal/reports/`. The current report kinds cover RFP/RFT/PMP; standalone
specification output needs an explicit contract.

## Proposed record model

Use a small set of explicit project-owned records. Names below describe the
contract; final migration names should follow repository conventions.

| Record | Purpose and essential fields |
|---|---|
| Specification type | Stable ID, org/project, version, human code/title, component kind, system reference, description and lifecycle. Represents a material, product, finish or assembly before it is assigned anywhere. |
| Application | Type, location/surface, optional work item, quantity and measurement basis, work action, inclusion and local overrides. Several applications can use one type. |
| Attribute definition | Stable key/version, label, data type, unit family, dimension/measurement meaning, allowed contexts, validation and authoring guidance. |
| Assertion | Type/application, attribute, raw value, typed value, qualifier/basis, meaning, source/user origin, evidence locator, reading version and review state. |
| Decision | Selected assertion or authored value, scope, actor, reason, version and supersession. Keeps acceptance separate from extraction. |
| Relationship | Explicit link such as component-of, finished-with, requires, matches or replaces; scope, sequence/proportion where relevant and provenance. |
| Selection group and option | Scheme or alternative group, candidate type/application, applicability and decision; limited conditional consequences where needed. |
| Requirement | Performance, execution or quality requirement linked to a type/application, responsible role, timing and evidence/dependency. Reuse delivery approvals and package roles. |
| Commercial evidence | Supplier/manufacturer/installer roles, quote/source reference, amount/rate, currency, tax state, unit/basis, covered scope, dates and exclusions. Link to existing costs only through explicit acceptance. |
| Schedule issue | Project-owned output scope, selected records and versions, issue purpose/date, evidence and unresolved decisions; immutable snapshot when issued. |

Every project row and relationship must be scoped to `org_id` and project.
Cross-project IDs, cross-org links, missing referenced records and stale edit
versions fail at the boundary. Stable IDs identify records across revisions;
display codes and supplier SKUs do not.

### Types and their applications

One type may be used in many rooms, on different surfaces or in several work
items. It can exist without an application while the user evaluates a choice.
Do not create fictitious work items solely to hold an unselected product.

Extend the existing part model with bounded hierarchy and room/zone support.
Repeated room names are valid under different parents. Validate parent project,
cycles and deletion impacts. A surface such as wall, ceiling or floor can be an
application qualifier; creating a physical part for every surface is unnecessary.

Initially support explicit part IDs and named groups with explicit membership.
Unresolved source locations remain text plus a pending mapping, not a silent
whole-project scope. Snapshot group membership at issue time. Avoid a general
query language for scope.

Type values are defaults for their applications. An explicit local decision
can override a type default, with its provenance retained. Contradictory imported
assertions do not participate in this precedence until accepted. A change to a
default previews affected applications, including those with local exceptions.

### Attribute meanings and value shapes

Start with registered attributes covering identity, material, colour/finish,
dimensions, performance, installation, quality, maintenance and commercial basis.
Each profile selects the relevant subset and stage-specific questions. Preserve
unmapped source notes as evidence pending classification instead of inventing
unbounded production keys.

Support text, boolean, enumeration, number with unit, range and explicit
comparison/tolerance. Unknown, not applicable, deferred, not required and zero
are distinct states. Preserve approximate, minimum, maximum and site-measured
qualifiers. Dimension roles distinguish product size, stock size, cut size,
opening, clearance, setting-out height and datum.

Performance records retain the property, required or offered value, test method,
conditions, source and verification state. A rating is not useful without its
method or scope. Keep source-stated WELS/slip/fire/acoustic claims separate from
verified evidence and from the project's required performance. No product or
standard statement becomes a verified rule through successful extraction.

Separate requirement, proposed selection, offered capability, allowance,
exclusion and observed installed condition. Reading confidence, acceptance,
source issue purpose and work action are independent states. Ordering and
procurement progress can link to existing workflows later; they do not redefine
what was specified.

### Assemblies and shared finishes

Use typed links for a tile and its grout, a door and its finish, a roof and its
membrane/fixings, or an appliance and its integration kit. Ordered layers retain
position and per-layer thickness. Blends retain constituents and proportions
with a common basis. Cycles in component or finish inheritance are invalid.

The source SP1 coating recipe requires three ordered coats; BR1/BR3 require
product blends; P1 requires sheen by surface. These cannot be modelled safely
as unrelated scalar values or a single prose description.

Require explicit linking when a source code is ambiguous. CF refers to different
base/upgrade selections; FG1/FG2 include both cross-references and definitions.
Retain document family, section and option context for reconciliation. A revision
updates evidence about a stable type rather than automatically creating a new
type for every occurrence of its code.

### Alternatives and their consequences

Distinguish a scheme, a mutually exclusive product choice, an optional extra
and a constituent of an assembly. A scheme can cover multiple coordinated
selections, while an upgrade may affect only one application.

Support small explicit conditions, such as selected-option equals a known ID,
with enumerated consequences. Selecting or declining the wine fridge can change
the associated cabinetry requirement. Preview the consequence and preserve
unresolved choices; do not infer that an unselected option was rejected.

Do not build an unrestricted rules engine. Validate dependencies and cycles;
keep consequences auditable and tied to the affected work/application.

### Execution and evidence

Record preparation, substrate, installation method, setting out, fixing,
interfaces, tolerances, samples, prototypes, inspections, tests, commissioning,
care and handover requirements where relevant. Requirements should link to
existing package responsibilities and delivery evidence rather than creating
a second approval system.

Distinguish a requested warranty from an offered warranty and its actual terms.
Similarly, an installer credential requirement is not proof that the eventual
installer holds it. Preserve a source-required approver role; where no architect
is appointed, identify the authorised person through project decisions.

### Commercial evidence

Store unit price, lump sum, allowance and aggregate quote as different bases.
Tax can be inclusive, exclusive or unspecified. Currency, quantity and valid-to
date may be unknown. A document issue date is not necessarily the quote date.
Supply-only and installation-inclusive amounts must remain distinguishable.

The existing cost model's accepted amount/quantity semantics should remain
intact. An isolated supplier rate can remain evidence without fabricating a
quantity or posting a budget. Choosing a product does not accept a contract.
Private supplier contacts belong in project access controls, not public fixtures
or shared attribute catalogues.

## Import and revision behaviour

Preserve source document identity, content hash, page, row/region, source span,
section and associated image region. Retain visual deletion and change markers
where detectable, with review required for uncertain interpretation. Joined
page text alone is insufficient for this schedule's struck-through items.

Current source replacement recreates passage records. Long-lived evidence
anchors therefore cannot rely only on passage IDs. Retain immutable source
identity and durable page/span/region locators with an extraction version and
reconciliation status. An extraction rerun must not erase accepted decisions.

Use the existing PDF extraction infrastructure; investigate its geometry/style
capabilities before adding a dependency. When a visual marker cannot be read
reliably, keep the row pending and expose its source image for manual review.
Do not silently interpret retained extractable text as an active selection.

Import creates proposals, and comparison occurs at matching attribute,
application, scheme, assertion meaning and issue context. A product width and
its required opening are not conflicting measurements. Newer document dates
alone do not override accepted user decisions or prove higher contractual
precedence. A deletion marker proposes withdrawal; it does not retire an
accepted record without review.

Use idempotency based on source hash, extraction version and reconciled source
identity. Retain both sides of conflicts and record the actor/reason that resolves
them. Removing a source marks affected evidence unavailable; it does not silently
convert sourced values into user-authored facts.

## Applicability beyond residential

The common model is shared. Attribute profiles and output templates vary by
system, component, exposure, work action, project use and issue stage. Avoid a
separate residential database model or a universal form containing every field.

| Context | Example to prove | Same model, different relevant detail |
|---|---|---|
| Residential new work | Bathroom tile and sanitary fittings | Scheme, finish, installation, location and unresolved selections |
| Commercial fit-out | Acoustic partition with door and ceiling interfaces | Required performance, assembly evidence, finish, tenancy/room scope and coordination |
| Industrial work | Protective floor coating and equipment installation | Substrate preparation, exposure, coating layers, curing, loads, service interfaces and acceptance evidence |
| Extension | New work connected to retained fabric | Existing/new boundary, matching requirement, interface investigation and local scope |
| Refurbishment | Replace a fitting while retaining adjacent finishes | Existing condition, protection, removal, make-good, compatibility and proposed versus observed product |
| Repair or remediation | Repair a damaged coating or waterproofing area | Investigation, cause/dependency, extent, preparation, repair system, testing and verification |
| Maintenance | Planned servicing or component replacement | Access, isolation, intervals, consumables, service requirements and completion evidence |
| Advisory or investigation | Performance requirement with no selected product | Requirements, findings and open design dependencies without fabricated procurement items |
| Removal and retention | Remove one service, retain another | Work actions, protection, disposal/isolation requirements and existing-condition evidence |

Commercial/industrial examples are proposed validation scenarios, not findings
from the two residential PDFs. Register additional profiles only with suitable
source evidence and owner review. Rule applicability still comes from verified
instruments and the building profile. A sector label alone cannot establish it.

## Outputs and the profiler

The first useful release must provide a standalone materials and finishes
schedule, independent of an RFP/RFT package. Rows include code, description,
location/application, selected properties, installation notes, reference image,
responsibility/dependencies and decision state. Prices and contact details are
optional audience-specific columns.

Provide a room/zone view and work-section view over the same records. Preliminary
output identifies alternatives and missing decisions. An issued output freezes
the selected record versions, scope membership, evidence and issue purpose.
Revision comparison identifies changed values and affected applications.

The profiler summarises coverage and unresolved decisions. It should link to
specification details rather than accumulate every finish as a profile field.
Relevant rule-driving facts can enter the existing profile review path, preserving
their meaning and evidence. Specification acceptance must not bypass that path.

The architectural RFP can state what information exists, its maturity and what
the appointed designer must complete, coordinate or verify. It must distinguish
owner selections from source references and unresolved requirements. Where no
architect is engaged, the specification workspace remains independently usable.

Schedule edits must update the underlying project record through an explicit
operation. Free report prose must not silently change a technical requirement,
and a structured edit must not rewrite an already issued document.

## Jev and deterministic assembly

Code discovers and parses candidate rows, values, units and source context. Jev
chooses classifications and candidate interpretations from bounded choices,
including none/uncertain. Code copies and validates the chosen values. Follow
[pre-parsed value extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).

Batch judgements for one state in one fan-out. Full-document work runs in the
background after filing; interactive editing and output rendering use prepared
records. Follow [fan-out](https://docs.typesafe.ai/patterns/fan-out). Selection,
arithmetic, unit conversion, rendering and dependency evaluation remain explicit
code paths; Jev does not generate specification prose or determine engineering
adequacy. Respect [jev-1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).

Use calibrated thresholds per question/option set; Noul probability is not
Choice confidence. See [confidence](https://docs.typesafe.ai/confidence).
Pin the model according to the repository contract and use the documented
[API](https://docs.typesafe.ai/api). No runtime LLM, embedding service or new AI
provider is part of this plan. Controlled templates and approved clauses turn
accepted records into readable documents.

## Implementation sequence

This is lane A: users rely on the recorded requirements and selected scope.
The outcome is a traceable specification that a non-specialist can author and
revise, reducing the time spent finding selections and resolving inconsistencies.
Follow [developing](../developing.md) and keep changes additive. Concurrent
front-end styling and colour work must remain independent.

| Slice | Deliverable | Completion evidence |
|---|---|---|
| 1 Shared records and standalone authoring | Validated type/application/assertion records, direct entry, evidence attachments, partial-save states, list/detail views and a standalone draft schedule | A user with no architect or uploaded file can define a tile, assign two locations, record an unknown grout and export the same selection consistently |
| 2 Import into the same records | Background candidates, code/context reconciliation, review and conflict handling; ingest fragments as well as finished schedules | Source values and user choices can coexist; accepting one proposal preserves rejected alternatives and provenance; repeated import is idempotent |
| 3 Rich selections and execution | Schemes, local overrides, assemblies, ordered coatings, fit constraints, conditional alternatives, quality requirements and commercial evidence | Golden cases for P1, SP1, BT1, LF4, VIN, quotes and approvals survive authoring, import and output |
| 4 Revision and visual output | Stable source anchors, deletion/highlight review, image associations, immutable issue snapshots and changes by application | GS/TBH1 remain withdrawn proposals; an issued schedule retains its original images/values; source reruns do not break traceability |
| 5 General applicability and integration | Commercial/industrial/existing-work fixtures, relevant profiles, profiler summaries and RFP/RFT projections | Same records support the cross-sector matrix without a residential fork or copying project selections into global knowledge |

Slices are vertical: each adds useful authoring, persistence, review and output
behaviour. Do not finish an extraction pipeline while leaving users unable to
create a schedule themselves. Attribute catalogue breadth can grow after the
shared contract is proven, while preserving unknown source information.

Out of scope: autonomous engineering design, unrestricted rule scripting,
universal product catalogue scraping, live price procurement, replacement of
cost/approval workflows, changes to the runtime stack, and a styling redesign.
Any new dependency needs a stated reason; none is assumed by this plan.

## Validation and speed

Use both private PDFs as golden fixtures and redacted synthetic public fixtures.
Preserve the original Newham 38-section answer key; register its PDF separately
from the existing DOCX fixture. The detailed evidence review supplies the second
document's expected semantic outcomes.

Required boundary cases include missing files, wrong org/project, dangling
references, duplicate codes, stale edits, cyclic scope/assembly links, unknown
units and locations, blank versus zero, repeated imports, continued rows,
unresolved options, conflicting finish names, withdrawn source rows, local
exceptions and deletion of referenced evidence. Tests should assert observable
workflow outcomes and evidence preservation.

Cross-sector fixtures must exercise at least one commercial assembly, industrial
performance requirement and repair/retain case. Source claims remain unverified
until their instrument/evidence is checked. Changes under `knowledge/` require
`python tools/check_knowledge.py`; new knowledge remains draft until owner review.

| Path | p50 ms | p90 ms | Gate |
|---|---:|---:|---|
| Filing | 1000 | 2000 | Existing whole-intake budget; no full-spec extraction while waiting |
| Profile read/edit | 50 | 150 | Existing budgets remain intact |
| Deterministic profile rebuild | 100 | 300 | Existing budget; detailed specification work stays outside this path |
| Specification list/detail/save/review | 100 | 250 | Proposed user-path budgets, measured with representative projects |
| Background preparation acknowledgement | 100 | 250 | Enqueue only; progress reported separately |
| Structured schedule assembly | 300 | 1000 | Reuse report-assembly budget for the bounded first output |
| Issue / prepared export retrieval | 100 | 250 | Reuse issue/export budgets; full PDF preparation measured separately |

Use at least the benchmark harness's 20 samples. Define document-size tiers and
numeric background completion/render budgets before enabling those paths in
production; the user-path budget must not conceal synchronous PDF or full-source
work. Tests fail when budgets are broken. Record measured timings, environment,
failures and dependencies when each implemented slice is delivered.
