# Fire, life safety and access extraction

## K4 vehicle-clearance ownership correction, 6 October 2026

Moved `fm.vehicle-clearance-lost-under-structural-beam` and coverage entries
B2-0916/B2-0917 to structure. The complete record is unchanged; its note about
keyword-based fire placement is retained as historical provenance. Detector
routing remains explicit and independent of its file location. This resolves
the flagged file ownership issue without approving its engineering content.
See `docs/evidence/2026-10-06-k4-vehicle-clearance.md` for validation.

Full relevant-seed reading is complete for the source set below. The cluster now contains 28 systems, 30 rules, 21 interfaces and 23 failure modes. This is draft extraction coverage, not a claim that all regulatory requirements or specialist designs have been verified.

The follow-up added six rules for lining reaction-to-fire evidence, hose reels, livable/adaptable design basis, impairments/restoration, integrated testing and specialist suppression hazards. Four edges cover lining selection, party-wall roof continuity, staged occupation of shared fire systems and brigade access. Seven explicit failure detectors support those gaps. The existing late fire-water assessment detector now requires an explicit dependency by issued design, tender or installation; an early plan with a future flow test is insufficient.

## Full-file source coverage

The continuation read the following files end-to-end. Large files were read in bounded chunks, and truncated output gaps were reread. Full reading is a discovery sweep, not primary verification of seed assertions.

| Files read in full | Technical coverage and disposition |
|---|---|
| `fire-life-safety-guide.md`; `access-consultant-guide.md` | Fire strategy, passive/active systems, egress, access, impairments, integrated testing and access audits |
| `ncc-reference-guide.md`; `as-standards-reference.md` | Entire references, including material outside this cluster; applicable technical content extracted or linked to canonical records, numeric prose retained only as unverified claims |
| `mechanical-services-guide.md`; `hydraulic-services-guide.md`; `mep-residential.md` | HVAC, ventilation, fire mode, water, drainage, gas, scope boundaries, testing and existing-building services |
| `trade-interfaces-coordination-guide.md` | Concealment, penetration, membrane, structure and services interfaces; merged canonical edges retained |
| `non-residential-sustainability-energy-guide.md`; `sustainability-energy-guide.md` | Services energy and certificate commitments; envelope retains canonical whole-of-home energy rule |
| `commercial-construction-guide.md`; `small-commercial-guide.md` | Base-building/tenant service boundaries, altered use, egress, HVAC, hydraulic, BMS and fit-out interfaces |
| `residential-construction-guide.md`; `multi-residential-apartments-guide.md` | House/apartment fire and acoustic separation, roof termination, risers, metering, ventilation, livable housing, commissioning and repeated details |
| `new-dwelling-guide.md`; `multi-dwelling-guide.md`; `ancillary-guide.md`; `renovation-guide.md` | Legacy archetype technical facts, retained to catch details absent in successors; classification assumptions, party walls, shared services, staged occupation and live cutovers |
| `industrial-construction-guide.md`; `institution-construction-guide.md`; `mixed-use-construction-guide.md`; `infrastructure-construction-guide.md` | Actual process/clinical/operator design basis, shared systems, emergency/degraded operation and staged acceptance; network assets are not assigned generic building NCC obligations |
| `building-remediation-rectification-guide.md` | Investigation before repair, physical evidence, temporary protection, impairments, occupied work and verification after rectification |
| `civil-residential.md` | Building-to-site drainage, sewer, pump-out failure alarms and authority connection boundaries |
| `defects-and-dlp-guide.md` | Technical defect symptoms and concealed-work interfaces; legal liability and DLP machinery excluded |
| `electrical-services-guide.md`; `ict-av-security-guide.md` | Essential power, controls, fire/security release and system commissioning boundaries; canonical electrical/comms ownership retained |

### Intentionally excluded material

PM doctrine, role workflows, fee/RFP prose, generic contract administration, registers, durations, cost rates, procurement process and legal warranty advice were not converted into physical-building rules. Technical requirements embedded in those sections were still considered. No source instructions were adopted as agent instructions.

`setup-and-commission-guide.md` was inspected at its introduction and complete heading list: “commission” means project mobilisation between award and site start, not technical services commissioning. Its role/setup machinery is outside this extraction; technical commissioning is covered by the full discipline and archetype reads above.

`remediation-due-diligence-guide.md` was inspected at its introduction and complete heading list. It explicitly covers contaminated land and directs building/fire/facade rectification to `building-remediation-rectification-guide.md`, which was read fully. Detailed contaminated-land methods remain with site scope. These two files are intentionally scoped exclusions, not claimed full reads. Other disciplines' detailed architecture, finishes, structure, site, acoustic, facade and vertical-transport seeds remain with their owning clusters; this report does not claim an independent full reread of them.

## Contradictions and suspect seed claims

- Construction-type and compartment-limit prose conflicts between NCC and AS seeds. Lead primary-backed C2D2/C3D3 tables and scope corrections are already present; they were preserved. Exceptions and mixed classifications still need separate assessment.
- Generic sprinkler Type A-to-B reduction is quarantined. A project needs the exact adopted provision or approved performance solution; sprinkler presence alone is insufficient.
- Historic NCC clause numbering and faulty mappings recur throughout the seeds, including access, egress, smoke detection and penetration clauses. Seed-number locators are not current verified locators. Lead C4D15 correction remains intact.
- AS 1530 seed mixes lining group classification, combustibility and fire resistance. Those are separate evidence questions; an FRL is not lining reaction-to-fire evidence.
- Apartment seed says three-storey Class 2 is Type B, promotes generic sprinkler concessions and describes an intra-SOU travel limit. Those are suspect seed claims and not executable calculations here.
- Smoke-alarm bedroom locations, power/interconnection rules, hydrant/hose-reel triggers, exit counts, travel distances and fire-isolation thresholds are overgeneralised. Jurisdiction, class and exceptions require primary instruments.
- Residential/ancillary prose treats a sleepout without a kitchen as Class 10a, while classification depends on actual use; do not adopt that shortcut. Livable/adaptable/accessible provisions and voluntary levels are not interchangeable; jurisdictional adoption and dates remain unresolved.
- Bushfire prose contains BAL-FMP versus BAL-FZ and blanket material/screening claims. Use the canonical BAL-FZ determinant and envelope rules; no duplicate bushfire rules were added.
- Annual fire statements are state obligations, while service schedules are standard-specific. Seed frequency and licensing statements require primary verification.

## Unresolved IDs and pending technical sources

Owned-folder validation passes with no unresolved IDs. No new determinants are proposed by fire. Merged aliases in `knowledge/MERGE.md` were read and no retired alias was recreated.

The full seed-reading gaps disclosed in the earlier checkpoint are closed for the listed source set. Remaining source gaps are authoritative detail, not hidden unread portions of those files: complete adopted FRL/lining hazard/egress/accessibility/hose-reel/sprinkler applicability tables; exception and concession conditions; standard test and installation scope; jurisdiction-specific livable housing, impairment, servicing and occupation obligations; project-specific special-suppression, medical/secure/assisted-evacuation and fire brigade acceptance criteria. No new tables were fabricated from seeds.

Bushfire construction/APZ/service-entry/water coverage is canonical in envelope (`rule.ncc.bushfire-prone-construction`, AS 3959 rules, `if.envelope-bushfire-site-basis`, `if.envelope-bushfire-service-entries`). Grab-rail backing is `if.envelope-grab-rail-backing-before-lining`. Emergency lighting remains canonical in electrical. A source sweep does not verify these other-cluster claims.

## Cross-cluster edges

- `if.services-penetrate-fire-rated-construction`: `hydraulic`, `mechanical`, `electrical`, `comms-security` → `fire-passive.penetrations`.
- `if.fire-detection-controls-smoke-plant`: `fire-active.detection` → `mechanical.fire-mode-air-systems`.
- `if.fire-alarm-releases-security`: `fire-active.detection` → `comms-security.access-control`, `access-egress.egress-doors`.
- `if.fire-alarm-controls-lift`: `fire-active.detection` → `vertical-transport.lift-fire-service`.
- `if.essential-power-supplies-fire-water`: `electrical` → `fire-active.fire-water`.
- `if.fire-tanks-load-structure`: `fire-active.fire-water` → `structure`.
- `if.sprinklers-share-ceiling-services`: `fire-active.sprinklers` → `mechanical.air-distribution`, `interiors`, `electrical`.
- `if.fire-stopping-before-concealment`: `fire-passive.penetrations` → `interiors`.
- `if.fire-facade-slab-edge`: `envelope.curtain-wall` → `fire-passive.compartments`, `structure`.
- `if.fire-and-acoustic-separation`: `fire-passive.compartments` → `interiors.acoustic-separation`.
- `if.accessible-fixtures-clear-circulation`: `hydraulic.sanitary` → `access-egress.accessible-facilities`.
- `if.accessible-threshold-weatherproofing`: `access-egress.accessible-access` → `envelope`.
- `if.fire-warning-acoustic-environment`: `fire-active.occupant-warning` → `interiors`.
- `if.damper-access-maintained`: `fire-passive.openings` → `access-egress.maintenance-access`, `interiors`.
- `if.lining-selection-fire-classification`: `interiors` → `fire-passive.linings-fire-hazard`.
- `if.party-wall-roof-continuity`: `fire-passive.compartments` → `envelope`.
- `if.brigade-access-to-fire-facilities`: `fire-active.hydrants`, `fire-active.fire-control` → `site`.

## Question design and verification boundary

Passage questions name `text`, ask one literal judgement, and use narrow system routing. Positive defect detectors require an explicit failure statement; silence and a merely planned test are not physical defects. Resolution questions identify documentary evidence, not a compliance certificate issued by Jev. Code handles counts, dimensions, applicability, scope, conflicts and acceptance thresholds.

Primary TypeSafe pages read for question authoring:

- [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13): literal atomic questions, explicit state fields, no arithmetic, narrow context.
- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): same-state questions run together; code resolves relevance and combines results.
- [Confidence](https://docs.typesafe.ai/confidence): noul probability is distinct from Choice confidence; calibration is per question.

No live Jev evaluation or threshold calibration was performed. Seed-derived records remain `status: draft`. New project-specific acceptance rules identify their project design basis rather than inventing a statutory clause. Unverified editions, clauses and numeric seed claims remain unverified. Existing primary-backed lead corrections and tables were preserved.

## Validation and ownership

`python tools/check_knowledge.py --only knowledge/clusters/fire --strict` passes with zero errors and zero warnings. No git writes were made. Parent-owned global determinants, primary tables and verification reports were not edited. Only current owned files were loaded for append/update; merged lead corrections were preserved.

## WP-K2 draft consequence, 6 October 2026

Added `cq.existing-fire-measures-approval-basis-review` from the frozen fire
guide's compliance-strategy and inception sections. Alteration, replacement,
upgrade or removal of included active/passive fire measures in an existing
building raises a draft review of the strategy and approval implications.
This is a planning review, not a statutory approval determination or an
automatic whole-building-upgrade requirement. There is no verified clause,
numeric rule, new signal or Jev question.

The proposal uses the existing approval-record path. Acceptance retains the
user's explicit delivery assignments; no system, package, authority or stage
is guessed from the label. Existing generic delivery risk
`uc.authority-requires-design-change-after-approval` is retained: it concerns
an authority-driven design change and has a different, broad work-type trigger.
This new consequence names affected fire work and its existing-building basis.
Potential overlap and usefulness still need the owner-scored K6 review.

Physical-investigation consequences remain outside this slice because the
current consequence shape cannot encode their target system. No shared schema
was extended. Content remains draft; this is partial WP-K2 progress, not closure
of the fire research, M1 quality gate or broader catalogue work.

## K4 investigation wording, 6 October 2026

Recast 1 draft investigation labels from physical-work instructions into
checks or assessments of their cited source constraints. All other record
fields are preserved, including predicates, signals, sources and draft status.
See `docs/evidence/2026-10-06-k4-investigation-labels.md` and the exact
before/after inventory `docs/unforeseen/k4-investigation-label-review.json`.
This bounded wording pass does not approve content or close owner review.

K4 follow-up: recast 1 further investigation labels that selected remedies
or design solutions before the investigation. Only labels changed. The shared
review inventory now records 82 corrections across the catalogue; this remains
a bounded, unapproved wording pass. See
`docs/evidence/2026-10-06-k4-investigation-labels.md` and
`docs/unforeseen/k4-investigation-label-review.json`.

K4 source assessment, 6 October 2026: B2-0559's layout assumptions and B2-0770's
occupant-management assumptions share a performance-solution applicability
review. The retained predicate remains broader than evidence of an actual
solution or breach; contract allocation and usefulness are still unapproved.
No fire record or question changed. See
`docs/unforeseen/k4-electrical-fire-merge-review.json` and
`docs/evidence/2026-10-06-k4-electrical-fire-merges.md`.
