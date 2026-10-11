# Consultant audit and trade RFT plan

11 October 2026. Status: draft direction for owner review. Repository audit and initial public-source research; no application changes or tender issue.

Build the trade library progressively, starting with formwork, then steel fixing. Keep the evidence ledger comprehensive and the project RFT to two or three readable A4 pages. Maintain separate package boundaries even when one contractor prices both.

## Consultant audit

The current working tree contains 49 consultant filing labels and 18 dedicated draft RFP templates covering 19 labels. BCA and Certifier share one template. All 18 have a ledger and RFP in `docs/rfp-golden/` and an encoded counterpart in `knowledge/rfp/`. The remaining 30 labels have no dedicated encoded RFP.

| Prepared group | Dedicated ledger and RFP pairs |
|---|---|
| Original five | Architecture, town planning, traffic, structural, civil |
| Further thirteen | BCA advice/certification, access, geotechnical, land surveying, cost planning, waste management, arborist, fire engineering, landscape architecture, fire services, mechanical services, electrical services, hydraulic services |

The original 9 October plan described three tranches. The completed library is better described as five original pairs plus thirteen further pairs: the latter include the original tranche-three investigation/report disciplines. It would duplicate work to commission that original tranche three again.

Façade, heritage and archaeology are present in `data/intake/disciplines.json`, but none has its own ledger/RFP pair or encoded discipline template. Heritage and archaeology also appear in draft consultant-package suggestion data. Presence in the taxonomy or suggestion list does not establish a developed procurement framework.

The 30 labels without dedicated templates are:

| Proposed grouping | Existing labels without dedicated RFP coverage |
|---|---|
| Building performance and specialist design | Acoustic, BASIX, ESD, Facade, Waterproofing, Vertical Transport, Roof Access |
| Site constraints and investigations | Archaeology, Heritage, Bushfire, Ecology, Environmental, Hazmat, Demolition Consultant |
| Specialist services | AV Consultant, BMS Consultant, ICT Consultant, Lighting Designer, Security, Commissioning Agent |
| Interiors and use | Interior Design, Signage Consultant, Workplace Strategist |
| Management and advisory | Building Consultant, Construction Manager, Expert Witness, Programmer, Project Manager |
| Broad filing labels needing a scope decision | Services Engineer, Specialist Consultant |

These groups are prioritisation aids, not a new building hierarchy. Broad labels do not necessarily need independent templates. Multi-services evidence already supports separate mechanical/electrical/hydraulic frameworks; do not create a duplicate appointment by default.

Recommended next consultant work: façade, heritage and archaeology as requested, followed by acoustic, ESD/BASIX and waterproofing according to upcoming projects. Test whether heritage needs separate built-heritage, historical-archaeology and Aboriginal-cultural-heritage scopes, competence and engagement boundaries. Do not assume the existing generic Archaeology label covers all three. Similarly check whether environmental and civil scopes adequately cover contamination, flood and water-authority specialist appointments before adding new labels.

### Quality and implementation status

- The [library index](../rfp-golden/README.md), [preparation review](../rfp-golden/preparation-review-2026-10-10.md) and [encoding register](../rfp-golden/encoding-register.md) record evidence preparation and encoding. The preparation review's statement that the new drafts had not been coded describes an earlier stage; the encoding register supersedes that statement.
- The preparation review reports 33 PDFs, 567 pages and 585 ledger rows for the further thirteen. This audit checked repository inventory and mappings, not a fresh page-by-page transcription of those PDFs.
- The encoding register records 137 populated scenarios and three-page checks. These are historical validation results, not tests rerun in this audit.
- Every encoded discipline remains `status: draft`. Internal preparation review is not owner sign-off or approval to issue.
- Electrical and hydraulic have later working-tree revisions. The [hydraulic pilot](../design/2026-10-11-hydraulic-rfp-pilot.md) now combines scope/stage pricing and removes evaluation from that pilot. The old seven-section skeleton is therefore not a universal description of today's output.
- The hydraulic note records successful local RFP checks but an unrelated Profile performance-gate failure. This audit makes no fresh whole-build or deployment claim.

## Existing trade groundwork

The taxonomy contains Main Works plus seven trades: Civil Works, Electrical Services, Mechanical Services, Hydraulic and Plumbing Services, Structural Steel, Fire Services Trade, Joinery and Kitchens. Windows and Glazing is a supplier label, not a trade label.

`internal/procurement/discipline.go` already distinguishes consultant services, trade works and the head contract. `knowledge/reports/templates.yaml` contains the generic `tpl.rft`, with scope, responsibilities, investigations, interfaces, dates, pricing and tender-return sections. The report assembler scopes RFT content to the package. This is useful groundwork, but it is not an authored library of trade-specific ledgers and RFT frameworks. Formwork and steel fixing do not yet have taxonomy entries or dedicated frameworks.

The [proposed trade master list](../rft-library/master-list.md) is a documentation backlog, not a bulk change to the filing taxonomy. It separates trades, procurement variants and common site services, and preserves existing IDs for later reconciliation.

## Evidence without subcontractor proposals

Use LLM knowledge offline to draft candidate scope lines and search questions. Treat those lines as hypotheses until checked. Jev remains the only runtime AI; this authoring process adds no runtime model or research agent.

For each package, combine:

1. Public Australian specifications and accessible tender schedules: evidence of scope and how work is bought. Record the project type and jurisdiction; a bridge specification does not become a building default.
2. Regulator guidance: risks, responsibilities and inspection topics. Verify applicability before expressing a current legal obligation.
3. Industry and manufacturer technical material: practical supply, installation, detailing, handling and interface questions. A manufacturer's offering is not a universal minimum or mandatory brand.
4. Existing structural, construction and trade-interface seeds: useful gap prompts, labelled as seed evidence. Do not import their process instructions or unverified legal assertions.
5. Explicit client choices: who buys, supplies, designs, inspects, pays and bears a defined interface. Record these as proposed procurement decisions, not discoveries from sources.

Fee proposals are particularly useful for commercial exceptions; they are not essential to create a first draft. The initial limitation is weaker evidence of customary exclusions, negotiated risk allocation and real subcontractor pricing. Leave rates, productivity, quantities and commercial periods unfilled rather than inventing market norms. Later quotations or builder review should challenge the draft.

### Initial source register

| Ref | Source inspected on 11 October 2026 | Use and limit |
|---|---|---|
| S1 | [SafeWork NSW Formwork Code landing page](https://www.safework.nsw.gov.au/resource-library/codes-of-practice/codes-of-practice/formwork) | Organises formwork/falsework planning, systems, hazards and erection/removal. The detailed PDF must be read and page-cited during the pilot. |
| S2 | [SafeWork NSW installation design and certification alert, 29 November 2024](https://www.safework.nsw.gov.au/safety-alerts/safety-alerts/formwork-installation-design-and-certification-safety-alert) | Identifies site-specific design, documented changes and inspection before loading and pouring. Preserve its context; do not copy its regulatory references as current verified law without checking the instruments. |
| S3 | [NATSPEC National Classification System](https://natspec.com.au/resources/national-classification-system) and [Building Structure worksection list](https://www.natspec.com.au/products/national-master-building-specification/building/building-structure) | Useful scope cross-check: formwork, reinforcement, post-tensioning and concrete placement are distinct worksections. Classification is not a mandated subcontract packaging model. Full licensed specifications have not been inspected or reproduced. |
| S4 | [InfraBuild reinforcing contracting options](https://www.infrabuild.com/reinforcing/products/reinforcing_service/project-management-and-contracting-options/) | Actual supplier offering distinguishes pricing arrangements and supply/install scope; supports making the procurement basis explicit. No universal rate or allocation follows from it. |
| S5 | [ACRS Traceability Certification Scheme](https://steelcertification.com/traceability-certification) | Describes certification scope and material identification. Helps frame evidence questions; supplier certification alone is not proof that every delivery or the installed works comply. |

TfNSW B80 was found as a possible comparative source, but its PDF fetch was blocked. It is not supporting evidence in this plan. No actual trade tender, accepted subcontract or full technical standard has yet been audited. During the pilot, seek an accessible building tender/specification and record a gap if none is available.

## Trade ledger framework

Use `docs/rft-library/<trade>/ledger.md` and `rft.md`. Add `sources.md`, `checks.md` and `handover.md` to keep evidence and progress reusable between sessions. Keep all authored outputs draft until the owner marks them reviewed.

The ledger has two linked tables:

| Evidence table field | Meaning |
|---|---|
| Stable evidence ID | Never renumber merely to tidy a document |
| Source and exact locator | URL/file, publisher, title, edition/date, access date and page/heading; file hash where a source is retained |
| Evidence class | Real tender/offer; specification; regulator; manufacturer/industry; seed; proposed client decision; unsupported hypothesis |
| Observation | Faithful paraphrase; preserve source-specific exclusions, quantities and conflicts |
| Applicability and limits | Jurisdiction, building/work type, procurement basis and what the source does not establish |
| Verification | Technical rule identifiers remain `verified: false` until checked against the instrument itself |

| Canonical scope table field | Meaning |
|---|---|
| Stable scope ID and wording | One independently priceable or allocatable item |
| Supporting evidence IDs | Trace the line back to observations; distinguish evidence from proposed policy |
| Trigger | Physical system, work extent, existing condition, procurement variant or explicit client choice |
| Scope position | Included, separately priced option, excluded with named other party, or unresolved |
| Allocation | Design, supply, install, inspect/release, protect and rectify, as relevant |
| Price and quantity basis | Unit, measurement definition, extent, allowances and variation treatment; no invented amounts |
| Interface and release evidence | Named adjoining package, handover condition and evidence needed |
| RFT destination and open question | Where the item appears; what prevents a settled position |

Keep opposing source positions separate. Absence from a source is not an exclusion. A package described as labour-only does not settle consumables, accessories, plant or supervision. Every exclusion must point to an actual responsible party or remain unresolved.

## Short RFT framework

The two-to-three-page target applies to the project-specific scope and tender return. Drawings, specifications and the identified subcontract terms are controlled referenced documents, not squeezed into these pages. State this boundary prominently so a short RFT is not mistaken for a complete technical specification or agreement.

| Page | Content |
|---|---|
| 1 | Project/package, issue and close dates, exact document revisions, work extent by location/element, procurement basis, concise included scope and separately priced options |
| 2 | Responsibility table, builder-provided facilities, access/lifting/sequencing, inspection and release evidence, quantified attendance/allowances, unresolved information or express pricing assumptions |
| 3 | One pricing schedule with separate base/options/GST, defined quantity/rate basis, lead time and duration, exclusions/departures, subcontract reference, required competence/insurance/design evidence where applicable |

Target roughly 900–1,200 words as an editing guide, then render and inspect. A word count does not prove page fit. Use normal readable type, allow space for tender responses, and test populated values. Remove duplication and irrelevant options before reducing font size. Do not hide critical package boundaries in an appendix to claim three pages. A genuinely complex package may require an expressly identified scope schedule and an honest exception to the target.

Use work areas, elements, pours and milestones where useful; consultant design stages are not the default trade pricing structure. Keep evaluation methodology internal unless the tender process specifically needs it in the issued document. Commercial details refer to the selected agreement and recorded departures; this plan does not invent payment, retention, indemnity or delay terms.

## Formwork and steel fixing pilot

Prepare formwork first, then use steel fixing to test whether the skeleton works for an adjacent trade. Model separate packages even if the same subcontractor is invited for both. A future combined concrete package would allocate the same scope once; do not build combined-package functionality in this pilot.

| Package | Decisions to resolve in its ledger |
|---|---|
| Formwork | Designed supply/erect/strip package versus erection labour only; temporary-works designer and checker; formwork/falsework/backpropping extent; formed faces and finishes; permanent decking boundary; openings, rebates and cast-ins; lifting and access; inspections, pour attendance, stripping/repropping authorisation; hire period, extended standing time and make-good |
| Steel fixing | Fix-only versus supply-and-fix; scheduling/detailing versus permanent structural design; bar/mesh supply and processing; accessories, chairs, ties, spacers and couplers; delivery/unloading/storage/lifting; prefabricated cages and temporary stability; starters and construction joints; PT and cast-in interfaces; pre-pour inspection, traceability and rectification |

These are proposed research questions, not settled universal inclusions. Structural steel is a different trade from reinforcing steel fixing.

The shared interface checklist must allocate survey/set-out; design information; formwork design and release for loading; crane and unloading; reinforcement supply/scheduling/fixing; penetrations and cast-ins; PT; concrete supply/pump/place/finish/cure; pre-pour inspections and final release coordination; protection; stripping/backpropping; and defects/make-good. Name the party for every activity. Do not turn an inspection or a supplier certificate into acceptance of the whole structure.

### Pilot steps and acceptance

1. Formwork: read selected primary sources and relevant seeds, record source coverage, write the full ledger, then normalise scope and unresolved allocations.
2. Draft `formwork/rft.md` and populate a clearly fictional, documented low-rise suspended-concrete example. No real project facts or engineering values are invented.
3. Render the populated example to two or three A4 pages at normal type. Review scope, pricing completeness and readability together. Preserve the full ledger regardless of page target.
4. Test slab-on-ground edges, suspended construction, labour-only appointment, existing-building work and missing/conflicting documents. Include a permanent-decking/PT interface case. Test whether a contractor can price without guessing scope ownership.
5. Owner reviews the actual formwork output. Revise the common skeleton once, then prepare steel fixing against it, including fix-only and supply-and-fix cases.
6. Cross-check both packages: no unassigned activity, duplicate price, assumed lifting provision or contradictory inspection/release duty. Every important issued clause has a source or identified client decision. Material unknowns remain visible.
7. Freeze the documentation pattern and record remaining evidence gaps. Encoding and application UI are a subsequent scoped task after the pilot direction is settled.

Delivering this plan does not claim that steps 1–7 or a three-page trade render have already been completed.

## Incremental research runs

No recurring automation or bulk agent run is started. The owner initiates each batch as capacity allows. Initially use one trade per run. Once the pilot is stable, a pair of neighbouring trades can be researched in parallel, followed by a separate interface review. Each agent writes only its assigned trade folder; one integrator owns shared files.

Use these recorded states: not started, sources inventoried, ledger drafted, RFT drafted, checked, owner reviewed, encoded. An agent can mark its checks complete but cannot mark owner reviewed. Keep the master list as a backlog; put actual progress in a preparation register.

Reusable future prompt:

```text
Prepare only [trade] to [ledger drafted / RFT drafted / checked].
Read AGENTS.md, the foundation design, this plan and the approved trade skeleton.
Read that trade's handover first and reuse existing sources and stable IDs.
Write only docs/rft-library/[trade]/. Do not change runtime code or knowledge YAML.
Research Australian primary sources relevant to [jurisdiction/project types].
Use seed and model knowledge as labelled candidate coverage, never invented evidence.
Record source locators, limits, conflicting allocations and missing paid standards.
Separate labour-only, supply-only and supply/install responsibilities where relevant.
Keep all work draft; unverified technical references stay verified: false.
Keep the ledger complete and the populated RFT within the agreed readable page target.
Check named parties, quantity/rate bases, interfaces, departures and unanswered facts.
Research cap: [owner-selected time/token budget, if supplied].
Default bounded pass: inspect up to six substantive primary documents, then report gaps
rather than expanding the trade or pretending evidence is complete.
Checkpoint sources.md and ledger.md as you work. End with handover.md naming the
completed steps, unanswered questions and the exact next action. Do not start another trade.
```

Six documents is a resource-control default, not a sufficiency standard. A short-source pass with unresolved evidence remains unfinished research. A later independent review should challenge high-risk boundaries rather than simply paraphrase the author's draft.

## Later application integration

Future work is Lane A: comparable, traceable tender scope shortens procurement decisions and preserves answer trust. Keep trades as views over physical systems/interfaces; do not reorganise building knowledge around packages. Reconcile stable trade IDs with the existing taxonomy, distinguish supplier-only from installing trades, and reuse deterministic assembly where appropriate without forcing consultant-specific semantics onto works.

Out of scope now: new UI, bulk taxonomy changes, runtime AI, services/dependencies, current pricing, full contracts, issued tenders, deployment and unsolicited batch research.

For a later implementation, preserve filing p50/p90 <= 1,000/2,000 ms, report read/settings <= 100/250 ms and assembly <= 300/1,000 ms. Proposed trade gates should match existing RFP gates: resolve/sections p90 < 20 ms and PDF p90 < 100 ms. Measure before claiming compliance. Keep organisation isolation and user edits; test unknown/conflicting facts, duplicate supply/install allocation, wrong organisation, stale revisions and removed/changed scope.

This task changes documentation only. It adds no dependency or runtime path. Runtime timings were not measured, and knowledge validation is not required because `knowledge/` was not edited.
