# T09 Waterproofing remediation (balconies, podium): domain map

Draft for owner review, 29 October 2026. The machine key is `shape.yaml` in this folder; it passes
`validate_shape.py`.

Tags: `[K: path anchor]` means SiteWise's knowledge says it and I opened the file to check. `[D]` means
professional domain knowledge that is not in the repo. Any statute or standard clause not confirmed from
the instrument is marked **verified: false**.

---

## 1. The job

**What it is.** The owners corporation (OC) of an occupied NSW strata Class 2 building renews failed
external above-ground waterproofing: the balconies always, and often a ground-floor podium or courtyard
over the basement car park, planter boxes, or a roof terrace. Symptoms are water through the ceilings of the
lots below, efflorescence and drips in the car park, drummy or cracked balcony tiles, and rust stains
and spalling at slab edges. The causes are almost always the same:

- the membrane stops short of, or under, the door sill (no upturn or set-down);
- falls are poor or reversed;
- outlets have no puddle flange or clamping ring, and there is no overflow;
- balustrade posts are core-drilled through the membrane;
- an internal wet-area membrane was used outside;
- tiles are bonded directly to the membrane with no drainage layer.

[K: knowledge/clusters/envelope/failure_modes.yaml fm.envelope-membrane-upstand-low-at-door,
fm.envelope-balustrade-fixed-through-balcony-membrane, fm.envelope-external-internal-membrane-substitution,
fm.envelope-balcony-leak-into-apartment-below; knowledge/clusters/envelope/unforeseen.yaml
uc.balcony-membrane-upturn-short-of-door-sill, uc.terrace-tiles-bonded-to-membrane]

**Base scenario** (used in `shape.yaml`):

- a 6-storey, 42-lot building completed in 2008, reinforced concrete, with tiled balconies and glass
  balustrades;
- out of statutory warranty, no claim;
- all 42 balconies renewed like for like, with no change to the external appearance;
- exempt from consent, traditional procurement, one AS 4000 lump-sum contract with provisional
  quantities and a schedule of rates [D].

The podium is the most common variant and is a trigger.

**Scale.** $0.1–3 M [K: testing/synthetic/PROJECT_TYPES.md T09 row]. Cost is driven by how many balconies
there are, how much access equipment the job needs, and by concealed conditions, not by GFA [D].

**Who procures it and how** [D]:

- **Client.** The OC acts through the strata committee, with the strata manager as its agent. The strata
  manager runs meetings, levies and insurance. It is not a project consultant and is not procured.
- **Lead consultant.** The OC appoints a **remedial building consultant**, also called a building defects
  or waterproofing consultant. It investigates, writes the defect report and options, specifies, runs the
  tender, and administers the contract as superintendent.
- **Remedial structural engineer.** Covers slab and slab-edge condition and the balustrade fixings.
- **Remedial builder.** Licensed, and registered under the DBP Act; uses specialist membrane applicators
  who are accredited by the manufacturer.
- **Funding.** The OC pays from its capital works fund and usually a **special levy** passed at a general
  meeting. That meeting, not design or approvals, is normally the critical path.

**What makes it go wrong** [D]:

- The OC tenders a "scope" before the cause is proven, so the contract is mostly variations.
- No representative opening-up is done, so the extent is unknown and provisional sums blow out.
- Falls are corrected with a thicker screed, which drops the effective balustrade height below the
  minimum and pushes the finished level above the door sill.
- Lot owners' alterations (re-tiled balconies, timber decking, enclosures, planters, AC condensers) are
  discovered late, and it is unclear who pays.
- Nightly temporary weatherproofing fails and floods lots below
  [K: knowledge/clusters/envelope/failure_modes.yaml fm.envelope-temporary-waterproofing-fails-and-floods-lower-levels].
- Hold points are skipped, so the manufacturer warranty is void
  [K: knowledge/clusters/envelope/unforeseen.yaml uc.waterproofing-warranty-void-by-installation].
- Resident access is lost to unanswered notices.
- A defects claim is lost because evidence was destroyed during strip-out.

---

## 2. Systems and work items

Action ids are the app's eight [K: knowledge/works/actions.yaml actions]. The work-type default for
remediation is `repair` [K: knowledge/works/actions.yaml work_type_defaults], but the membrane is
**replaced**: a full strip and renewal is `replace` by the app's own definition.

| system | in scope (base) | action | note |
|---|---|---|---|
| **site** | none in base | — | No ground works. `site.demolition` is a **trap**: removing tiles and membrane is part of the membrane replacement, not demolition. It becomes real only for licensed asbestos removal (trigger `hazardous-materials`). `site.landscape` is affected only with planters on a podium (trigger). `site.contamination` is never in scope; see §10. |
| **substructure** | none | — | `substructure.damp-waterproofing` is basement tanking and below-ground work, explicitly not balconies [K: knowledge/clusters/structure/systems.yaml substructure.damp-waterproofing excludes]. Car park leaks under a podium come from the podium membrane (envelope), not the basement walls. |
| **structure** | `structure.concrete` | investigate | Balcony slabs and slab edges are assessed for cracking, cover and reinforcement corrosion. Repairs are provisional, and become `repair` under trigger `concrete-spalling`. `structure.temporary-works` does not cover scaffolding [K: clusters/structure/systems.yaml excludes]. |
| **envelope** | `envelope.external-waterproofing` | replace | The core: membrane classified for external trafficable use, upturns, terminations, outlet clamping, drainage layer [K: knowledge/clusters/envelope/systems.yaml envelope.external-waterproofing]. |
| | `envelope.flashings-seals` | repair | Door sill and threshold flashings, sealant and movement joints, the slab-edge and wall junction, weepholes. |
| | `envelope.windows-doors` | retain | Sliding doors stay and are protected. Becomes `replace` if an upturn cannot otherwise be achieved (trigger). |
| | others | — | Walls, roof coverings, curtain wall, insulation and thermal performance are untouched. A roof terrace membrane is a trigger. |
| **interiors** | `interiors.tiling` | replace | Balcony tiles and bedding removed and relaid. The app files external tiling under interiors [K: clusters/envelope/systems.yaml interiors.tiling]. |
| | `interiors.wet-area-waterproofing` | — | **Out of scope, and a trap.** Internal bathrooms are a different job. Ceilings and paint in lots below are a trigger (`internal-water-damage`). |
| **fire-passive** | none | — | No new penetrations through rated elements are expected. |
| **fire-active** | none | — | Smoke alarms and sprinklers are untouched; protect any sprinkler heads at balcony soffits [D]. |
| **mechanical** | none in base | — | AC condensers and condensate drains on balconies are a trigger (`mechanical.refrigerant-condensate`, alter). They belong to the lot owners, are moved by a licensed refrigeration tradesperson, and need no consultant [D]. |
| **hydraulic** | `hydraulic.roof-drainage` | upgrade | Balcony outlets are re-flanged or replaced, and overflows added. The app's system covers balconies and overflows [K: clusters/services-wet-air/systems.yaml hydraulic.roof-drainage]. |
| **electrical** | none in base | — | Balcony lights and GPOs are disconnected and reinstated by the builder's electrician (trigger `balcony-services-relocation`). |
| **comms-security** | none | — | Not affected. |
| **vertical-transport** | none | — | The lift is used for material movement only. Protect the car and set booking rules [D]. |
| **access-egress** | `access-egress.barriers` | alter | Balustrades are removed and re-fixed outside the membrane. Height is measured from the **new** finished level (NCC 2022 Vol One Part D3 barrier provisions, verified: false; the app's own rule cites NCC 2019 numbering, `clause_verified: false` [K: knowledge/clusters/fire/rules.yaml rule.ncc.stairs-barriers]). Replacement is a trigger. |

---

## 3. Existing-building interfaces

The whole job is an existing-building interface [D].

- **Occupation.** The building stays fully occupied. Residents lose their balconies for days to weeks
  each, and lots must be cleared of furniture, plants, BBQs and AC condensers. The OC's power to enter lots
  needs written notice (SSMA 2015, verified: false). Plan the work balcony by balcony or stack by stack,
  with a resident liaison, a notice regime, privacy screening and daily make-safe. The
  `existing_building` and `live_site` RFP modules switch on from the work type and from
  `hdr.cond.operational_constraints` [K: knowledge/rfp/modules.yaml; internal/rfp/drivers.go readFlags,
  deriveFlags].
- **Common property versus lot property.** In most NSW strata plans the membrane, slab and outlets are
  common property. Lot-owner add-ons (decking, tiles laid over the original, planters, enclosures) may be
  lot property or covered by a by-law. Settle who pays before tender (verified: false; depends on the plan
  and its by-laws).
- **Services cut-overs.** Disconnect and pump down AC condensers, isolate balcony electrical circuits, and
  cap hose taps. No authority cut-overs. Keep stormwater flowing through the outlets during the works.
- **Hazardous materials.** Not in the base (built 2008, after the asbestos ban). In older stock,
  fibre-cement substrates, bituminous membranes and mastics may contain asbestos. The app raises a hazmat
  survey only as an Implication [K: knowledge/clusters/structure/consequences.yaml
  cq.pre-2004-fabric-hazardous-materials-survey]. Silica dust from breaking out tile beds and concrete
  needs controls under the WHS Regulation (verified: false).
- **Structure.** Balustrade fixings, slab-edge corrosion, the added screed load, and movement joints that
  must carry through the membrane [K: knowledge/clusters/envelope/interfaces.yaml
  if.envelope-membrane-structural-joints].
- **Matching existing work.** Tiles rarely match, so plan whole-balcony re-tiling. Use one membrane system
  per balcony; mixing systems at a junction voids warranties [K: clusters/envelope/unforeseen.yaml
  uc.membrane-junction-between-two-systems-unwarranted]. New clamping rings may not fit existing outlets
  [K: uc.membrane-clamping-ring-does-not-fit-existing-outlet]. The membrane must stay continuous with the
  facade weatherproofing [K: clusters/envelope/interfaces.yaml if.envelope-balcony-facade-continuity].
- **Dilapidation.** Make an internal condition record of every lot below and beside the works before
  starting: ceilings, finishes, existing water damage. This is not a dilapidation of adjoining
  properties [D].

---

## 4. Consultants and why

| discipline (app id) | stage engaged | need | why |
|---|---|---|---|
| Remedial building consultant (`consultant.building`; a doc may say "waterproofing consultant" → `consultant.waterproofing`, accepted as the same package) | Investigation, then through to defects | **Always** | Leads the job: diagnosis, defect report, options and order-of-cost estimate, specification, regulated design for waterproofing, tender, superintendent, hold points. |
| Remedial structural engineer (`consultant.structural`) | Investigation (slab assessment), then documentation and construction | **Always in base** | Balustrade re-fixing design and certification, slab and slab-edge condition, added load, concrete repair specification. |
| Remedial builder (`trade.main_works`) | Procurement | **Always** | Head contract. |
| Hazmat (`consultant.hazmat`) | Investigation | Conditional: built before about 2004, or asbestos register entries | Survey of affected areas; clearance after licensed removal. |
| Landscape (`consultant.landscape`) | Documentation | Conditional: podium planters | Reinstatement of soil build-up and planting over the new membrane. |
| Hydraulic (`consultant.hydraulic`) | Documentation | Conditional: podium or roof drainage capacity | Outlet and overflow sizing, drainage through the car park. **Not** needed for re-flanging balcony outlets. |
| Quantity surveyor (`consultant.quantity_surveyor`) | Investigation or tender | Conditional: large jobs, a levy budget, or a defects claim | Independent estimate. Usually the building consultant gives the order of cost [D]. |
| Expert witness (`consultant.expert_witness`) | Before strip-out | Conditional: a defects claim | Joint inspections, evidence preservation, expert report. The OC's solicitor has no app id. |
| Heritage (`consultant.heritage`) | Investigation | Conditional: heritage listing | Confirms whether the work stays exempt. |
| Town planner and certifier (`consultant.town_planner`, `consultant.certifier`) | Approvals | Conditional: consent required | Only if the external appearance changes or the work is otherwise not exempt. |
| Roof access (`consultant.roof_access`) | Construction | Conditional: swing stage or roof terrace | Anchor inspection and certification [D]. |

**Never needed for the base:** architect, civil, geotechnical, surveyor, BASIX or ESD, traffic,
mechanical, electrical and fire consultants. The app suggests **none** of the consultants this job does
need [K: testing/synthetic/reference/KNOWLEDGE_INDEX.md §2.1 and §9 check 5; knowledge/works/package_defaults.yaml baselines].
Once documents add hydraulic, mechanical or electrical scope, it suggests the wrong consultants
[K: package_defaults.yaml scope_additions[2]–[4]]. The base outlet item alone brings Hydraulic. These
are keyed in `shape.yaml not_expected_procurement`.

---

## 5. Approvals and authorities

- **Planning.** Like-for-like repair of waterproofing with no change to the external appearance is
  normally treated in NSW as maintenance that needs no consent, or as exempt development. Confirm it in
  writing with a certifier or planner. Codes SEPP 2008 provisions: verified: false. With exempt planning
  the app adds only `fr.exempt-confirmation` [K: knowledge/program/fragments.yaml fr.exempt-confirmation].
  A changed balustrade design, a balcony enclosure, or a heritage listing can move it to DA or CDC
  (trigger).
- **No construction certificate and no occupation certificate** on the exempt path [D]. The app still
  suggests the "Construction certificate and certifier fees" line [K: knowledge/costs/starter_lines.yaml
  lines[4]].
- **Design and Building Practitioners Act 2020 (NSW).** "Building work" includes the repair, renovation
  or protective treatment of a Class 2 building, and **waterproofing** is a building element. So:
  - registered design practitioners prepare **regulated designs** and give **design compliance
    declarations**;
  - a registered building practitioner does the work and lodges the declarations on the NSW Planning
    Portal before work starts;
  - the declarations are varied when the design changes;
  - at the end, the building practitioner gives a **building compliance declaration** (s 17), which
    the consultant reviews together with the varied regulated designs;
  - exemptions exist in the Regulation.

  All of this is verified: false: the s 4 and s 6 definitions, s 17, the Regulation exemptions, and the
  lodgement timing for work that needs no CC or OC. A balustrade fixing is probably **not** a regulated
  building element ("load-bearing component essential to stability"); a slab repair is. So structural
  regulated design is a trigger item (`concrete-spalling`), not a base certainty. The app knows this only as an unverified RFP threshold
  [K: knowledge/rfp/structural.yaml thresholds dbp_regime] and the `proj.regulated_design` key
  [K: knowledge/profile/project_details.yaml regulated_design].
- **Home Building Act 1989.** The contractor licence covers residential building work above the
  licensing threshold. Home building compensation (HBCF) cover is **not** required for multi-storey
  buildings with a rise of more than three storeys, so it applies in the `low-rise-hbcf` trigger.
  Statutory warranties (s 18B) run 6 years for a major defect and 2 years otherwise (s 18E);
  waterproofing is a "major element". All verified: false.

  Further HBA obligations, also verified: false:
  - Residential building work on a Class 2 building is HBA work. An AS 4000 contract therefore needs
    HBA-compliant special conditions: a written contract, the deposit cap, the statutory warranty wording
    and the major-work contract requirements.
  - The remedial builder's own s 18B warranties then run to the OC.
  - The membrane applicator needs a NSW waterproofing licence. Manufacturer accreditation is not enough.
  - HBCF cover, where it applies, starts above a contract price of about $20,000.
- **Strata Schemes Management Act 2015.** All verified: false:
  - the OC's duty to maintain and repair common property (s 106);
  - general meeting resolutions for special levies;
  - the strata building bond and defects scheme (Part 11), which covers only contracts for new buildings
    from 2018;
  - powers of entry to lots;
  - a **special resolution** where common property is *altered* rather than repaired (s 108): replacing
    balustrades, raising sills, replacing doors, or enclosing balconies.
- **Security of Payment Act 1999 (NSW).** Applies to the OC contract; the superintendent manages payment
  claims and schedules (verified: false).
- **SafeWork NSW.** Work at height and edge protection while balustrades are off. Scaffold above the
  licence height needs a licensed scaffolder. Licensed asbestos removal is notified in advance. All
  verified: false.
- **Council.** A hoarding or scaffold over the footpath, or a road occupancy, needs approval under the
  Roads Act 1993 s 138 (verified: false). Trigger `facade-access-scaffold`.
- **No Sydney Water, DNSP, Fire and Rescue NSW or BASIX involvement** on the base.
- **Hold points** (the contract's, set by the consultant) [D]:
  1. substrate, falls and outlet set-down before priming;
  2. membrane inspection (thickness, laps, upturns, terminations, cure) before cover;
  3. flood test (typically 24 h; check the duration against the specification);
  4. balustrade fixing certification before handover of each balcony.

  The app has no external-membrane hold point [K: knowledge/clusters/envelope/interfaces.yaml
  if.envelope-waterproofing-before-tiles is wet-area only].

---

## 6. Cost plan shape

Lines only; no amounts (see `shape.yaml cost_plan`).

- **Fees and charges.** Long service levy (Long Service Corporation; threshold and rate verified: false).
  There is **no** DA fee, CC fee or service-authority fee on the base.
- **Consultants.**
  - remedial building consultant, in stages: investigation and report, specification and tender,
    contract administration;
  - remedial structural engineer;
  - investigation-stage disbursements of the remedial building consultant: opening-up, water testing
    and make-good. The consultant usually engages the opening-up contractor [D]. The app has no
    early-works package before the head contract.
- **Construction**, by trade or element [D]:
  - preliminaries and occupied-building controls;
  - pre-construction condition surveys of the lots;
  - access equipment and edge protection;
  - strip-out of tiles, bedding, screed and membrane;
  - substrate preparation and falls screed;
  - membrane system with upturns, terminations and outlets;
  - flashings, sealants and movement joints;
  - outlets, puddle flanges and overflows;
  - balustrades removed, re-fixed and certified;
  - tiling reinstatement;
  - provisional sums for concealed conditions (concrete, substrate, door sills).
- **Contingency and other.** Design contingency, construction contingency (higher than for new work),
  escalation, exclusions (lot-owner items), and OC costs (strata manager meetings and levy administration,
  legal review of the contract).

App behaviour:

- The app gives only the four contingency lines plus the CC and service-authority fee lines
  [K: testing/synthetic/reference/index/type-predictions.md T09].
- Construction lines appear only as generic element codes once work items exist (04 envelope, 05
  interiors, 09 hydraulic, 13.x balustrades) [K: knowledge/costs/elements.yaml].
- No budget split exists for remediation [K: knowledge/costs/budget_splits.yaml].
- **Lines the app produces that this job should not carry** (`shape.yaml not_expected_cost_plan`):
  P05 construction certificate fees and P06 service authority fees (both always suggested); the
  Hydraulic consultant fee (from the outlet item); and, conditionally, Mechanical and Electrical
  consultant fees.

---

## 7. Program shape

The base has 32 activities; see `shape.yaml program`. Stages and the activities that drive each:

1. **Investigation.** Committee approves the brief and fee → records and leak history → visual and water
   testing → intrusive opening-up → structural assessment → defect report with options and order of
   cost.
2. **Decision.** The OC selects an option and approves the design fee. This is a committee or general
   meeting gate, filed under `concept_design`.
3. **Documentation.** Specification and details → balustrade fixing design → regulated designs and
   declarations.
4. **Approvals.** Confirm the exempt path (in parallel with documentation).
5. **Procurement.**
   - Tender documents → tender with site inspection → evaluation (HBA licence, DBP registration, the
     applicator's NSW waterproofing licence, warranty) → **general meeting resolves award and special
     levy** → contract award.
   - Levy collection timing can hold the start for months. That is the OC's call, and its duration is
     unknown [D].
6. **Construction.**
   - DBP lodgement, resident notices, site establishment and access, a pre-commencement condition
     survey of the affected lots, and payment of the long service levy (paid directly, because there is
     no CC; verified: false).
   - Then repeat **per balcony or stack**: strip → substrate and outlets → hold point → membrane → hold
     point and flood test → re-fix balustrades, tile, seal → certification.
   - The work is weather-sensitive, because membranes need dry substrates and cure time [D].
7. **Completion.** DBP building compliance declaration (verified: false) → PC inspection → handover of
   warranties, records and the maintenance-plan update.
8. **Defects.** Defects liability period and final inspection.

**Logic.** Opening-up comes before the specification. The specification and regulated design come before
tender. The levy resolution comes before award. Each balcony's membrane hold point comes before its tiling.
Tiling comes before the balustrade is re-fixed.

**Long-lead items** [D]:

- replacement balustrades (shop drawings and fabrication), under trigger `balustrade-replacement`;
- replacement high-sill doors, under trigger `door-threshold-replacement`;
- the membrane system if it is imported or specified as a proprietary product;
- scaffold and hoarding approval.

**App behaviour.** No remediation skeleton exists, so the app uses `sk.generic`. The only true fragment is
`fr.exempt-confirmation`. `fr.balcony-waterproof-2` exists but needs work type new or extend, and action
new or replace. Its duration scales with building GFA. Contract award, PC, handover and defects fragments
are all gated off for remediation [K: knowledge/program/fragments.yaml; knowledge/program/durations.yaml
du.balcony-waterproof-2; testing/synthetic/reference/index/type-predictions.md T09].

---

## 8. RFP shape

**Remedial building consultant** (no template; the app uses generic `tpl.rfp-capex`).

- **Must carry:**
  - record review;
  - inspection, leak mapping, water testing and representative opening-up with make-good;
  - defect report with cause, extent, options and order of cost;
  - specification, details and balcony schedule;
  - waterproofing regulated design and declaration (verified: false);
  - tender management and recommendation;
  - committee and general meeting attendance (allowance count);
  - AS 4000 special conditions compliant with the Home Building Act (verified: false);
  - superintendent role, including SOP Act payment schedules;
  - review of the builder's building compliance declaration and the varied regulated designs;
  - hold-point and flood-test inspections;
  - PC, defects and warranty collation;
  - returnables: PI insurance, design practitioner registration, and occupied-strata remedial
    references.
- **Must not carry:** DA or planning services, architectural concept or schematic stages, BASIX or
  Section J, services design, or any contaminated-land scope.

**Remedial structural engineer** (template `structural`).

- **Must carry:**
  - existing structure assessment (the app includes it because the work type is remediation
    [K: knowledge/rfp/structural.yaml stages existing_assessment]);
  - balustrade fixing design and certification (the `metalwork` row is only an option);
  - screed load check;
  - provisional concrete repair specification (`strengthening` row, option);
  - confirmation of whether any structural work is a regulated building element (DBP Act, verified:
    false). Structural regulated design for slab repairs belongs to trigger `concrete-spalling`; the
    template suggests the stage for any NSW Class 2;
  - construction-phase inspection of fixings.
- **Must not carry:** geotechnical brief, shoring or retention, dilapidation of adjoining properties,
  planning approval stage, shared Revit model, footing or framing design, super-flat floors, and
  certification for a construction certificate.
- **What the app gets wrong in this RFP:**
  - Any Class 2 part makes structural scale complex, which wrongly brings in the shared Revit model.
  - The concept stage and the geotechnical brief are always included.
  - The construction phase is *needed*, but the app prices it only as an option. It falls after the
    boundary end (`construction_documents`), and `resolveStages` applies the boundary before
    `include_when scale complex` [K: internal/rfp/resolve.go resolveStages]. For the same reason the
    inspection schedule comes out as an option.
  - `metalwork` (the balustrade fixing design, which is core scope here) is always only an option.
  - The template's regulated-design and construction-documents wording assumes a construction
    certificate that the exempt path does not have.

  [K: knowledge/rfp/structural.yaml scale.complex, stages regulated_design, construction_documents,
  positions shared_model, geotech_brief]

  The expected state, in the template's own stage and position keys, and each predicted app difference
  are in `shape.yaml rfps[consultant.structural].expect` and `app_differs` (14 differences). The remedial
  building consultant has no template, so it has no `expect`; one `app_differs` entry records the
  generic RFP.

**Conditional:**

- **Landscape** (`landscape-architecture`; its `s9_advisory` "Advice / rectification" stage fits).
- **Hydraulic** (`hydraulic-services`; its advisory stage becomes an option for remediation
  [K: knowledge/rfp/hydraulic-services.yaml]).
- **QS** (`cost-planning`). Its remediation row reads "Defect/remediation/contamination costing", which
  needs the client flag `cost_remediation` [K: knowledge/rfp/cost-planning.yaml positions remediation].
  It must not drift into contamination.
- **Town planning and certifier**, only under trigger `development-consent-required`.
- **Hazmat** and **expert witness** have no templates.

---

## 9. Triggers

There are 14 triggers; full keys are in `shape.yaml`. Each changes procurement, cost, program or RFP.

| id | how documents show it (including indirectly) | how a document denies it | adds |
|---|---|---|---|
| `podium-over-basement` | "water entering the car park beneath the courtyard", "calcite stalactites on the B1 soffit under the planters" | "no podium; at-grade car park", "podium renewed 2021, excluded" | Landscape and hydraulic consultants; podium membrane and drainage cell; planter reinstatement; car park protection; podium flood test |
| `hazardous-materials` | "completed in 1986", "asbestos register lists fibre cement soffits", "original bituminous membrane" | "2019 survey found no asbestos", "built 2012" | Hazmat survey; licensed removal and clearance; SafeWork notice |
| `concrete-spalling` | "concrete cancer", "rust staining on soffits", "coastal exposure" | "no spalling or exposed reinforcement" | Concrete testing; provisional repairs; structural repair specification |
| `balustrade-replacement` | "960 mm above the tiles", "no safety-glass marking", "corroded posts" | "side-fixed, retained and protected" | New balustrades (long lead); temporary edge protection; possible consent check; SSMA s 108 special resolution if common property is altered (verified: false) |
| `door-threshold-replacement` | "sill flush with the tiles", "membrane cannot turn up behind the frame" | "adequate set-down; frames retained" | Doors supplied and installed under the head contract (long lead; no separate package); doors replaced before the upturn; s 108 special resolution |
| `statutory-warranty-defects-claim` | "proceedings against the developer", "building bond inspector's report", "warranty expires in March" | "out of warranty, no claim" | Expert witness; evidence preservation before strip-out; legal costs; possible Building Commission involvement. Needs a later building year than the base: within 6 years for the warranty; a building contract from 2018 or later for the bond |
| `development-consent-required` | "changes the appearance of the facade", "council advises a DA is required", "balcony enclosure" | "certifier confirmed like-for-like, no consent" | Town planner and certifier; DA, CC and OC; DA and CC fee lines; s 108 special resolution |
| `heritage-listed` | "within a Heritage Conservation Area", "Schedule 5 item" | "not a heritage item" | Heritage consultant and referral; the LEP cl 5.10(3) minor-works notice may avoid consent for a local item; Codes SEPP exempt development generally unavailable (both verified: false) |
| `facade-access-scaffold` | "scaffold encroaches on the footpath, Class A hoarding", "swing stage from the roof anchors" | "all work from within the balconies" | Scaffold or swing stage; council s 138 approval; roof anchor certification |
| `internal-water-damage` | "collapsed ceiling in Unit 304", "mould below balcony 502", "insurer accepted the resultant damage claim" | "no internal damage reported" | Make-good in the lots below; mould assessment |
| `balcony-services-relocation` | "condensers on 30 balconies", "condensate onto the tiles", "lights in the upturn zone" | "no services on the balconies" | Pump down, store and recommission condensers; electrical reinstatement |
| `low-rise-hbcf` | "three-storey walk-up", "HBCF certificate before deposit" | "rise of more than three storeys, so not required" | HBCF premium (above about $20,000, verified: false); a returnable; a pre-start check |
| `staged-delivery-funding` | "Stage 2 subject to the next special levy", "separable portions" | "one continuous stage" | A levy gate per stage; escalation; per-stage contract administration |
| `roof-terrace-in-scope` | "roof terrace membrane leaking into level 6" | "pitched metal roof, out of scope" | Roof membrane; temporary plant relocation; anchor recertification |

**Variants** (the seven most common): podium, hazardous materials, spalling, door thresholds, defects
claim, balustrade replacement, balcony services.

---

## 10. Keyword traps and negations specific to this type

| trap | right reading |
|---|---|
| **"Remediation"** | Means defective-building rectification here, never contaminated land. `site.contamination` is labelled "Contaminated land and remediation" [K: clusters/structure/systems.yaml]. The seed separates the two explicitly [K: data/reference/clerk/data/seed/building-remediation-rectification-guide.md "Purpose and boundary"]. A "Remediation Action Plan", NEPM or "validation" is a different job; no Environmental consultant. |
| **"Flood test" / "flood testing"** | A membrane water test, **not** a flood-prone site. It matches the `flood_hazard` trigger `\bflood` [K: knowledge/determinants.yaml flood_hazard]. A trigger only routes the passage to the Jev question, so the risk is low: one wasted question and a small chance of a wrong reading. Expected: `det.flood_hazard` is not true. "Ponding" is not flood either. |
| **"Basement"** | "Podium over the basement car park" matches the `basement` trigger, which routes the passage to the question "does the project include a basement or deep excavation?" [K: knowledge/profile/flag_readings.yaml basement]. **High risk:** the basement really exists, so Jev may answer yes. That switches on structural retention design, temporary works, dilapidation and civil shoring. The basement is existing and nothing is excavated. Expected: `proj.flag.basement` is not yes. |
| **"Bathroom" / "ensuite" below** | The worst trap. The implied-scope reader runs **in code, without Jev and with no work-type gate**, on every read passage [K: internal/profile/implied.go; internal/profile/build.go withImpliedScope]. "Leaks into the bathroom of the unit below" suggests wet-area waterproofing, tiling, sanitary, cold and hot water and local exhaust [K: knowledge/profile/scope_defaults.yaml implied_by wet-room]. Those scope rows then suggest **Hydraulic and Mechanical consultants** and 08/09 cost lines. Internal wet areas are not in scope. |
| **Other implied-scope words** | Ordinary remediation wording fires the same code path [K: scope_defaults.yaml implied_by]. Use these words in negative-control documents. |
| | "the Unit 304 **bedroom** ceiling" → internal doors |
| | "**kitchen**" → joinery, hydraulic, exhaust (Hydraulic and Mechanical again) |
| | "the **common wall** between balconies" → adjoining structures, compartments, acoustic separation |
| | "materials carried via the fire **stairs**" → stairs and ramps |
| | "**brickwork** upstands", "**repointing**", "**masonry** planter walls" → structure.masonry |
| **"Waterproofing"** | Can mean basement tanking (substructure), internal wet areas (AS 3740) or external membranes (AS 4654). Only the last is this job. |
| **"Membrane"** | Not sarking or wall wrap (`envelope.pliable-membranes`), and not an under-slab vapour barrier. |
| **"Strip-out" / "demolish the tiles"** | Matches implied `site.demolition` [K: scope_defaults.yaml implied_by strip-out]. It is membrane-replacement work, not demolition. |
| **"Cladding" / "facade"** | Belongs to T17 (combustible cladding). A balcony job's "facade continuity" detail is a termination, not façade replacement. `cladding system` matches the `facade_type` trigger. |
| **"Defects liability period"** | The contract DLP. It is not the HBA statutory warranty or the strata building bond. |
| **"Exempt"** | Can also mean "GST exempt" or "exempt from HBCF". The planning sense is "exempt development". |
| **"Strata"** | "Hard strata" or "strata subdivision" in a development document is not an occupied strata scheme. |
| **"PC"** | Practical completion versus a PC (prime cost) sum. |
| **"Repair" versus "replace"** | "Repair the balcony waterproofing" usually means full membrane renewal. A patch repair is a different, lower-value scope. |
| **Common negations** | "Balustrades to remain", "no asbestos detected", "podium excluded", "lot-owner decking by owners", "no internal make-good (by insurer)". |

---

## 11. Gaps in the built-in knowledge, and what the app will get wrong

1. **Nothing is produced for the work itself.** Remediation has no scope defaults, so no membrane work
   item, no construction lines and no construction activities appear until a document or the user adds
   them [K: knowledge/profile/scope_defaults.yaml empty_work_types]. The default action `repair` also
   misreads a full renewal.
2. **Zero consultants.** There is no remediation baseline. The lead remedial building or waterproofing
   consultant and the remedial engineer are never suggested, and neither has an RFP template. Filing
   aliases lack "remedial building consultant" and "remedial engineer"
   [K: knowledge/works/package_defaults.yaml baselines; data/intake/disciplines.json]. Once minor
   services scope appears, the app instead suggests Hydraulic, Mechanical or Electrical consultants
   [K: scope_additions].
3. **Program.** It uses `sk.generic`, with empty approval stages and no general meeting or levy gate.
   - The balcony fragment is gated to new work.
   - There are no contract award, PC, handover or defects rows.
   - There is no external-membrane hold point or flood test.
   - Occupied protection is gated to refurb.
   - Default package stages (concept, approval, construction documentation) do not fit a remedial
     consultant [K: knowledge/program/skeletons.yaml; fragments.yaml; knowledge/works/stages.yaml].
4. **Cost plan.** The app suggests CC and service-authority fee lines that do not apply, and has no
   preliminaries, access equipment, provisional sums, long service levy or OC cost lines. There is no
   budget split [K: knowledge/costs/starter_lines.yaml; budget_splits.yaml].
5. **RFPs.** The structural RFP is sized as a new Class 2 commission (Revit model, geotechnical brief,
   concept stage).
6. **Profile.** The strata condition affects nothing.
   - The implied-scope words (bathroom, bedroom, kitchen, common wall, stairs, brickwork) add wrong scope
     **in code**, and with it Hydraulic and Mechanical consultants and cost lines.
   - Basement and flood only route a Jev question. Basement is the risky one, because the basement
     exists.
   - Condenser and balcony-light items suggest Mechanical and Electrical consultants the job does not
     need.
   - The structural RFP's DBP wording assumes a construction certificate.
7. **NSW law.** DBP, HBA, SSMA, the building bond, the RAB Act and SOP are not encoded, apart from one
   unverified RFP threshold.

The full machine list is in `shape.yaml gaps` (24 entries).

---

## 12. Sources and unverified claims

**Opened and cited:**

- `knowledge/clusters/envelope/{systems,rules,interfaces,failure_modes,unforeseen}.yaml`
- `knowledge/clusters/structure/{systems,consequences}.yaml`, `knowledge/clusters/fire/{systems,rules}.yaml`,
  `knowledge/clusters/services-wet-air/systems.yaml`, `knowledge/clusters/delivery/unforeseen.yaml`
- `knowledge/works/{actions,package_defaults,stages}.yaml`, `knowledge/costs/{elements,starter_lines,budget_splits}.yaml`
- `knowledge/program/{skeletons,fragments,durations}.yaml`
- `knowledge/rfp/{modules,structural,hydraulic-services,cost-planning,landscape-architecture,bca-certification}.yaml`
- `knowledge/profile/{taxonomy,flag_readings,scope_defaults,project_details}.yaml`, `knowledge/determinants.yaml`
- `data/intake/disciplines.json`, `internal/rfp/drivers.go`
- `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and `index/*.md`
- the seed `data/reference/clerk/data/seed/building-remediation-rectification-guide.md` (a seed, not
  loaded by the generators)

**Unverified (verified: false), to confirm from the instruments:**

- AS 4654.1 and .2 (2012) membrane, falls, upstand and overflow requirements; the app's own numbers are
  unverified and its seeds disagree [K: knowledge/clusters/envelope/rules.yaml rule.as4654.*].
- NCC 2022 Vol One Part F1 external waterproofing (I believe F1D5 cites AS 4654) and Part D3 barrier
  height and climbability.
- DBP Act 2020 and DBP Regulation 2021: the building work and building element definitions, exemptions,
  and lodgement timing without a CC.
- Home Building Act 1989: s 18B and s 18E warranty periods, waterproofing as a major element, the
  licensing threshold, and the HBCF multi-storey exemption.
- SSMA 2015: s 106, special levies, Part 11 building bond, and entry to lots.
- The RAB Act 2020 powers; the SOP Act 1999 application to OC contracts.
- Exempt or maintenance status under the EP&A Act and Codes SEPP 2008.
- Roads Act 1993 s 138; SafeWork asbestos notification and scaffold licensing; the long service levy
  threshold and rate.

**Judgement calls the owner may want to change:**

- the structural engineer as a base package rather than a trigger;
- AS 4000 as the base contract (AS 4902 D&C, or minor-works forms, are also seen);
- `consultant.building` as the lead id, with `consultant.waterproofing` accepted;
- `hdr.cond.operational_constraints: 24_7_occupied`. Jev's criterion for that option is "explicitly
  occupied or operating 24 hours a day" [K: internal/profile/questions.go]. Strata documents usually say
  only "the building remains occupied", which fits `live_environment` or `partial_occupation` just as
  well. All three set `live_operations: yes` [K: internal/rfp/drivers.go readFlags]. **The scorer
  should accept any of the three and must not score this key on exact value.**

Unverified claims added after review: the DBP Act s 17 building compliance declaration and its timing
without an OC; SSMA s 108 special resolution for altering common property; HBA contract requirements;
the NSW waterproofing licence for applicators; the HBCF threshold of about $20,000; LEP cl 5.10(3)
minor works; Codes SEPP limits on heritage items; the building bond applying to contracts from 2018;
direct payment of the long service levy when there is no CC. All verified: false.

Schema keys added after the T01 pilot reviews:

- `expected_derived`: `det.ncc_class` from the subclass, and `det.existing_building` from the work type.
- `scenario_inputs`: the 10 facts the documents must state, with phrasing. Use "units", not "lots", so
  the units trigger matches.
- RFP flags are set explicitly in `profile`: basement, excavation_adjoining, authority_asset, rail,
  transfer_structures, osd_existing and staged all "no"; flood and contamination false.
- `removes` on three triggers:
  - `balustrade-replacement` drops the base re-fix item, its cost line and its design activity;
  - `door-threshold-replacement` drops the retained doors;
  - `development-consent-required` drops the exempt confirmation.
- While writing the structural `expect`, I corrected an earlier statement: `resolveStages` applies the
  boundary first, so the construction phase is an option in the app, not included through complex
  scale (§8 and the rfp gap).

## Disputed review findings

None. All 20 findings in REVIEW.md were applied.
