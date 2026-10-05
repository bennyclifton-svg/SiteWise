# Clerk content inventory for SiteWise report drafting

Status: draft editorial inventory, 5 October 2026. This batch maps existing material to planned consumers; it does not approve wording, legal statements, benchmarks or runtime rules.

## Local source archive

The sources inventoried here are now preserved inside SiteWise at `data/reference/clerk/data/`. Use that archive for drafting; its manifest and checksum gate are authoritative for source identity. Original Clerk paths below describe provenance, not an external checkout dependency. Existing SiteWise knowledge clusters remain the first stop for technical concepts and IDs; draft only missing report wording rather than re-extracting the same rules or risks.

## Findings

The inspected Clerk data directories contain **51 Markdown seed guides and 10 JSON taxonomy files**. **23 guides have both staged services and RFP lead sentences.** These are useful starting material, not an owner-approved clause library. Inventory coverage is structural across all files; detailed passage review in this batch covers fire services, the hydraulic RFP/trade sections, and the PMP source-map structure. Other entries below are candidates for later review, not validated scope.

Clerk's guides mix practical scope, discipline-specific services, broader professional roles, legal assertions and agent instructions. Extract the domain content only. Do not transfer its PM doctrine, agent execution instructions, application code, claimed approval status or unverified rule numbers. Source hashes and headings are in `source-manifest.json`.

## Priority source map

| Source in Clerk | Reusable material | SiteWise consumer | Disposition |
| - | - | - | - |
| `data/seed/fire-life-safety-guide.md` §§ Services by stage, Consultant RFP minimum returnables, Construction verification and integrated testing, Handover and operations | Stage leads, explicit appointment boundary, evidence review, design/tender support and close-out | WP-40a; K2; later WP-50/51 | Read relevant passages; redraft as bounded wet-fire services, not automatically a fire-engineering appointment |
| `data/seed/hydraulic-services-guide.md` §§ Consultant request for fee proposal, Consultant deliverables, Hydraulic trade tender | Required client inputs, interfaces, deliverable types, fee allowances; separate construction obligations | WP-40a/b; WP-31; later WP-50 | Read these sections; use only relevant fire-water boundary content |
| `data/seed/trade-interfaces-coordination-guide.md` | Interface sequences, hold points and failure modes | K1/K2; later RFT attachments | Headings inventoried; residential examples are not general industrial requirements |
| `data/seed/procurement-tendering-guide.md` | Procurement-route vocabulary and subjects to resolve | WP-40b/50 | Sample read; legal/contract assertions excluded pending primary-source review; no contract-form default imported |
| `data/seed/setup-and-commission-guide.md` | Brief inputs, role boundaries, design responsibility topics | WP-40b/51 | Headings inventoried; this is project setup, not a substitute for technical systems commissioning; do not import agent behaviour |
| `data/seed/mechanical-services-guide.md`, `electrical-services-guide.md`, `architectural-services-guide.md`, `access-consultant-guide.md` | Further staged service modules | 0777 content batch; WP-40b | Candidates; detailed drafting/review still required |
| `data/seed/program-scheduling-guide.md`, `cost-management-principles.md`, `contract-administration-guide.md`, `defects-and-dlp-guide.md` | Programme, cost, decisions and close-out content topics | PMP / WP-51 | Candidates; avoid duplicating authoritative project records or importing contractual conclusions |
| `data/taxonomy/pmp-section-seed-map.json` | Links report sections to seed headings with class/work-type conditions | WP-40b coverage map | Navigation aid; do not copy Clerk's required-stage logic as SiteWise policy |
| `data/taxonomy/consultant-rosters.json` | Default consultant suggestions | WP-K0/30 | Already has a SiteWise draft counterpart; reconcile, do not create a second catalogue |
| `data/taxonomy/work-scopes.json`, `typical-work-scopes.json`, `asset-register.json` | Scope vocabulary, defaults and condition/action inputs | WP-20/21/30; applicability review | Map to SiteWise systems/actions/parts; defaults never prove actual scope |
| Other taxonomy files in manifest | Class, discipline, complexity, risk and emphasis vocabularies | Later gap analysis | Inventory only; no blanket import |

## Coverage by content family

| Content family | RFP | RFT | PMP | Next bounded action |
| - | - | - | - | - |
| Project definition and evidence references | Brief and exclusions | Works brief and issued documents | Current definition and change | Project fields plus common fragments; no copied project values in library |
| Services by stage | Main appointment scope | Design responsibility where expressly allocated | Appointment status and decisions | Fire-services first; 0777 differences next |
| Physical works and interfaces | Design/coordination deliverables | Supply, installation, protection, testing and handover | Material gaps/risks | K1/K2 records drive scope; do not paste every guide into every report |
| Existing-building risks | Investigations and design inputs | Staging, impairments and conditional work | Risk, owner, action and date | Confirm triggers and role boundaries before adopting wording |
| Commercial return | Fees, allowances, exclusions, departures | Pricing schedule and tender return | Cost summaries from ledger | No prices, weightings, commitments or terms invented from seeds |
| Time and approvals | Proposed service dates and dependencies | Access/staging dates and approvals | Saved baseline/current/forecast | Project records remain authoritative; no upload-based progress inference |
| Provenance and uncertainty | Basis, assumptions and open inputs | Exact revisions and conditional scope | Changes and unresolved decisions | WP-42; visible provisional status |

## Guides with staged services

The following is a discovery index, not a completeness or accuracy certificate. Open the source at the listed line to inspect its stage modules.

| Guide | Services section | Reuse status |
| - | - | - |
| [access-consultant-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/access-consultant-guide.md:42>) | Line 42 | Draft source; owner review required |
| [advisory-services-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/advisory-services-guide.md:160>) | Line 160 | Draft source; owner review required |
| [arborist-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/arborist-guide.md:42>) | Line 42 | Draft source; owner review required |
| [architectural-services-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/architectural-services-guide.md:95>) | Line 95 | Draft source; owner review required |
| [building-remediation-rectification-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/building-remediation-rectification-guide.md:163>) | Line 163 | Draft source; owner review required |
| [civil-commercial-industrial-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/civil-commercial-industrial-guide.md:45>) | Line 45 | Draft source; owner review required |
| [civil-residential.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/civil-residential.md:188>) | Line 188 | Draft source; owner review required |
| [electrical-services-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/electrical-services-guide.md:165>) | Line 165 | Draft source; owner review required |
| [fire-life-safety-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/fire-life-safety-guide.md:156>) | Line 156 | Draft source; owner review required |
| [geotechnical-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/geotechnical-guide.md:44>) | Line 44 | Draft source; owner review required |
| [hydraulic-services-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/hydraulic-services-guide.md:176>) | Line 176 | Draft source; owner review required |
| [ict-av-security-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/ict-av-security-guide.md:157>) | Line 157 | Draft source; owner review required |
| [landscape-architecture-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/landscape-architecture-guide.md:42>) | Line 42 | Draft source; owner review required |
| [mechanical-services-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/mechanical-services-guide.md:293>) | Line 293 | Draft source; owner review required |
| [non-residential-sustainability-energy-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/non-residential-sustainability-energy-guide.md:165>) | Line 165 | Draft source; owner review required |
| [remediation-due-diligence-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/remediation-due-diligence-guide.md:551>) | Line 551 | Draft source; owner review required |
| [structural-commercial-industrial-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/structural-commercial-industrial-guide.md:46>) | Line 46 | Draft source; owner review required |
| [structural-residential.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/structural-residential.md:201>) | Line 201 | Draft source; owner review required |
| [surveying-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/surveying-guide.md:40>) | Line 40 | Draft source; owner review required |
| [sustainability-energy-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/sustainability-energy-guide.md:188>) | Line 188 | Draft source; owner review required |
| [town-planning-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/town-planning-guide.md:48>) | Line 48 | Draft source; owner review required |
| [traffic-transport-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/traffic-transport-guide.md:44>) | Line 44 | Draft source; owner review required |
| [waste-management-guide.md](<D:/AI Projects/sitewise/data/reference/clerk/data/seed/waste-management-guide.md:43>) | Line 43 | Draft source; owner review required |

## What is still missing

An explicit report-template contract for placeholders, optional blocks and unknown-input behaviour; a field-to-trigger mapping; owner-reviewed appointment boundaries; independent expected-obligation lists; verified regulatory references; and coverage evidence for contrasting projects. The current clause shape alone does not establish all of these.

Do not commission a full classification × work-type matrix of documents. First review one assembled appointment, extract its reusable parts, test a contrasting case, then commission only uncovered cells. Track legal/regulatory checks as K3 work; this editorial batch has not performed those checks.
