# Golden RFP preparation review

Date: 10 October 2026. Status: draft; owner review required. This records preparation quality assurance, not owner approval, current legal verification or SiteWise implementation.

## Delivered scope

Thirteen additional disciplines each have an evidence ledger and reusable RFP. The [library index](README.md) links all pairs: BCA advice/certification, geotechnical, land surveying, cost planning, access, waste management, arborist, fire engineering, landscape architecture, fire services, mechanical, electrical and hydraulic services.

The team reviewed all **33 supplied PDFs / 567 physical pages** for the remaining source folders. Exact versions are identified by filename, page count and SHA-256 in the [source inventory](source-inventory-2026-10-10.md). Shared services PDFs are counted once, even though several ledgers use them. The two landscape PDFs are execution versions of one agreement, not independent consultant bids. Existing architecture, civil, structural, town-planning and traffic pairs were preserved unchanged.

Meaningful scope, stages, deliverables, quantities, exclusions, dependencies, fees and commercial terms were extracted. Marketing and unrelated capability statements were discarded. Optional ESD/energy, utility/ASP3 and AV evidence is retained in the [specialist companion](services-specialist-evidence.md); it is not silently included in core appointments. No standalone RFP was added for a specialty without its own remaining source discipline folder.

## Review method and findings

Focused authors prepared disciplines sequentially; the parent reviewed each ledger/RFP before the next discipline began. Shared services source reviewers read the combined proposals and supplied detailed handovers; discipline authors checked relevant original scope, stage and fee pages. Parent review included original text, relevant seeds and selected legible page images. Ledger coverage distinguishes team review from independent repeat review, and records image-only content, duplicates and unavailable incorporated documents.

A separate reviewer checked the full new RFPs and selected ledger evidence for cross-discipline consistency. No material unresolved inconsistency was identified. The final services review prompted an explicit mechanical-condensate handover in hydraulics: receiving connection/level/capacity, falls/traps/tundish/discharge and upstream/downstream design and testing owners. Its ledger identifies this as proposed interface policy. Preparation also corrected R9 disbursement semantics and two electrical-only details carried into the hydraulic ledger.

| Interface or convention | Result |
|---|---|
| Fire disciplines | Fire services designs selected physical systems, including sprinklers, pumps and tanks. Fire engineering separately defines strategy/performance solutions. Electrical/fire services allocate dry fire per system and interface; mechanical designs selected smoke-control functions. |
| Services and tests | Internal versus utility design, passive versus active ICT/AV, controls/programming, installation, test execution, witnessing and statutory decisions are separately allocated. Common supply does not transfer wet-fire design into hydraulics. |
| Hydraulic interfaces | Domestic/fire water, roof/civil drainage, sewer/treatment, trade waste/process, landscape connections and mechanical condensate have explicit boundaries. |
| Other consultant roles | BCA advice versus statutory certification; geotechnical versus environmental work; access advice versus specialist/architectural design; survey types; arborist advice versus tree execution; landscape versus civil/structural work remain distinct. |
| Appointment and price | Start/end, purchased stages, later options, D&C handover, quantities/duration and fee ceilings are explicit. A programme entry does not purchase a service. |
| Unknown facts | Yes/no/not known/conflicting states; resolve uncertainty or authorise a priced assumption with owner/date/consequence/review point, otherwise hold affected scope. Unknown is not an exclusion. |
| Building and work coverage | Actual uses, systems, existing conditions and interfaces activate scope across new work, fit-out, refurbishment, extensions, remediation and advice. Historical industrial/camp assumptions are not universal defaults. |
| Shared procurement format | Sections 1–7 and R1–R9, including R9 expenses. Full proposed terms, agreement precedence, variation rules, project-use rights, insurance and departures are written out. Deliberate BCA procurement exceptions are identified as policy and subject to current verification. |
| Evidence and policy | Historical offers, seed-derived coverage and proposed client positions are separated. Accepted/performed services are not inferred from offers; missing documents and material conflicts remain visible. |

## Validation

Final document checks passed for **13 discipline pairs, 585 ledger rows and 31 new Markdown files**:

- Draft status, sections 1–7 and R1–R9 are present in every new RFP.
- Numeric ledger references resolve, with no duplicate evidence IDs.
- Relative document links resolve; UTF-8 text and trailing-whitespace checks pass.
- All 33 inventory source identities are represented in the ledgers; the inventory totals 567 physical pages.
- Git status shows only new documentation under `docs/rfp-golden/`; the existing five discipline pairs have no diff. `git diff --check` passes; untracked files were covered by the document checks above.

The final independent services review confirmed the condensate clarification and its policy provenance; no outstanding material findings or clarity fixes remain within that review's scope.

No runtime, knowledge data, dependencies or user-facing execution paths changed. Runtime benchmarks and `tools/check_knowledge.py` were not run because this is documentation-only preparation. Existing filing budgets remain p50 <= 1,000 ms and p90 <= 2,000 ms; these are unchanged requirements, not measured results from this task.

## Limits and decisions before use

The independent consistency review was not a fresh page-by-page verification of all PDFs, legal review or current technical-instrument validation. Preparation checks do not establish that every historical transcription is free from error. Source technical/regulatory references remain `verified: false`; fees, insurance and historical project assumptions are not current requirements or benchmarks.

Before encoding or issuing the standards, the owner should review the drafts and proposed common commercial defaults. For a project issue, complete the actual building/system facts, appointment boundary, quantities, evidence and interface owners; obtain missing incorporated documents where relevant; verify current instrument and practitioner applicability; settle agreement/version, precedence, insurance, liability and departures. The documents remain drafts and have not been coded into SiteWise or sent to consultants.

## Peer review follow-up: section 5 (10 October 2026)

A second review compared the thirteen new RFPs with the first-pass disciplines. Sections 1-4, 6, 7 and R1-R9 were found aligned in structure. Section 5 was not: the new drafts used long prose rows ("trigger" and "required decision"), which cannot become the discrete included / option / excluded / to-confirm lines the first-pass disciplines encode (`positions` in `knowledge/rfp/*.yaml`). Section 5 in all thirteen was rewritten as one-line items, each with a position (I / O / X / ?) and a condition or named party. No ledger rows or references changed; every `#n` still resolves. The electrical dry-fire allocation matrix and the hydraulic responsibility schedule were kept after the table.
