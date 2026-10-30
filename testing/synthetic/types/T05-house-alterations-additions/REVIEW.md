# T05 House alterations and additions: independent review

Reviewer brief: `testing/synthetic/briefs/02-review.md`. Read 29 October 2026 against the working tree.
Nothing but this file was written. No Jev, no network.

## Verdict: **FIX**

The map is sound at the root: the base scenario, the skeleton, the existing-building story and most
trigger evidence are right for a Sydney rear addition. It is not yet a reliable answer key. It mixes two
baselines when it marks `built_in`, it overstates the "alter gets no program row" gap, it leaves out
activities the app really produces, it contradicts itself on Civil, and it misses the heritage
conservation area exclusion from CDC. All of these are listed below with the exact change.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T05-house-alterations-additions/shape.yaml`
→ `ok`, exit 0. No warnings.

## 2. Findings

### F1. Civil is both a false positive and a base package (internal inconsistency). Severity: high

- **Where:** `shape.yaml procurement[3]` (`consultant.civil`), `cost_plan` "Civil", `rfps[5]` (civil);
  `DOMAIN.md` §1 ("structural/stormwater engineer"), §4 table row and the paragraph after it, §5 DA
  package ("stormwater concept"), §6 consultant list, §7 ("The CC needs structural and stormwater
  design"), §11 gap 3; `shape.yaml gaps[2]`.
- **What is wrong:** §4 lists Civil as "Usually" needed and the shape puts it in the base procurement,
  cost plan and RFPs. Yet §11 gap 3 and `gaps[2]` say the app "over-suggests Civil", and gap 3 counts
  "three wrong fee lines", which only works if Civil is *not* counted. One key cannot score the same
  suggestion as both right and wrong.
- **Decision: Civil is a trigger, not base.** The base scenario (DOMAIN §1, shape header) says the
  55 m² roof drains to the existing system or a kerb outlet, and OSD applies only if the DCP threshold
  is met. On that job:
  - the architect shows downpipes, pits and the discharge point on the site and stormwater plan;
  - the licensed plumber designs and certifies the connection to AS/NZS 3500.3 [D];
  - the certifier accepts that at CC.

  A civil or stormwater engineer is engaged when any of these applies:
  - the DCP requires OSD or on-site retention for the added impervious area;
  - the lot falls away from the street, so drainage needs an inter-allotment easement, a charged line, a
    pump-out or an absorption trench;
  - a council drainage pipe or easement crosses the lot;
  - there is overland flow;
  - the council's lodgement checklist asks for an engineer-prepared stormwater management plan.

  The repo's civil template supports this call. It is written from a warehouse and a camp RFT, has no
  `house` module, and always includes "Roads pavements and levels" and a DRAINS model
  (`knowledge/rfp/civil.yaml elements`, `drainage_model`). That scope is heavy for a gravity connection
  of one roof.
- **Exact change:**
  1. `shape.yaml`: delete `procurement` entry `consultant.civil`, cost line `"Civil"` and the
     `rfps` block for `consultant.civil`.
  2. Add a trigger:
     ```yaml
     - id: stormwater-engineering-required
       label: Engineer-designed stormwater (OSD, charged line, easement or fall to the rear)
       evidence:
         - "On-site detention is required as the development increases the impervious area above the DCP threshold."
         - "The site falls to the rear; roof water to be drained through an inter-allotment drainage easement over No. 9."
         - "Charged stormwater line to the kerb outlet."
         - "Council's 375 mm stormwater pipe and easement run through the rear yard."
         - "Submit a stormwater management plan prepared by a suitably qualified engineer."
       negation:
         - "Roof water from the addition connects to the existing gravity system discharging to the kerb; OSD is not required."
       profile: [flag.osd_required, flag.authority_asset]
       adds:
         procurement: [consultant.civil]
         cost_plan: ["Civil"]
         program: []
         rfps: [{discipline: consultant.civil, must_include: <the current civil must_include>, must_not_include: <the current civil must_not_include>}]
         authority: ["Council stormwater approval; inter-allotment easement with the downstream owner where needed"]
       built_in: {procurement: "yes", cost_plan: "yes", rfps: partial}
       source: "knowledge/works/package_defaults.yaml baselines[0]"
     ```
     (`375 mm` is illustrative document wording, not a rule number.) Add it to `variants`.
  3. `DOMAIN.md` §4: change the Civil row's Need to "Trigger". Make the false-positive line read
     "**Civil, Electrical, Hydraulic, Mechanical and Town Planner are false positives for the base job**"
     (Mechanical: see F6). In §1, §5, §6 and §7, write "stormwater drainage on the architect's site plan;
     engineer's plan only under `stormwater-engineering-required`".
  4. §11 gap 3: "Expect five wrong fee lines (Civil, Electrical, Hydraulic, Mechanical, Town Planner) and
     two missing ones (BASIX, Hazmat). Arborist and Dilapidation are missed only under their triggers."

### F2. "Every altered item gets no construction row" is overstated. Severity: high

- **Where:** `DOMAIN.md` §7 "What the app produces" and §11 gap 1; `shape.yaml gaps[0]`.
- **Evidence:** I checked `knowledge/program/fragments.yaml` and the evaluator:
  - `gen.fragmentRows` in `internal/program/generate.go` evaluates each fragment once per work item,
    with only that item in the environment;
  - `work_action` maps to `works` in `evalPredicate` (`internal/knowledge/relevance.go`);
  - `det: work_type` reads `Catalog.workTypeValue` (`internal/knowledge/works_eval.go`), which joins
    every part's work type with commas, and `eq` or `any_of` matches any part;
  - nothing in the generator falls back to a generic row for an uncovered item (only `investigate`
    items get one, in `investigationRows`).

  The claim holds for most altered items on `extend`. Each of these fragments needs `new` or `replace`
  (the 1a family), or `refurb` (the 1a-reno family and `fr.structure-alterations`), so they produce no
  row:

  | Item and action | Fragment that does not fire |
  |---|---|
  | `structure.alterations` alter | `fr.structure-alterations` |
  | `interiors.walls-linings` alter | `fr.linings-1a`, `fr.linings-1a-reno` |
  | `interiors.painting` alter | `fr.painting-1a` |
  | `interiors.floor-finishes` alter | `fr.floors-1a` |
  | `hydraulic.sanitary` alter | `fr.hydraulic-sanitary-1a`, `fr.hydraulic-1a-reno` |
  | `hydraulic.cold-water` alter | `fr.hydraulic-cold-1a`, `fr.water-1a-reno` |
  | `electrical.cabling-wiring` alter | `fr.electrical-1a`, `fr.electrical-1a-reno` |
  | `electrical.switchboards` upgrade | `fr.switchboard-1a` |
  | `site.landscape` alter | `fr.landscape-1a` |

  These alter, upgrade and repair items **do** produce rows on `extend`:

  | Item and action | Fragment | Gate |
  |---|---|---|
  | `site.services-connections` alter (in this shape's scope) | `fr.services-connections-1a`: "Lodge services applications", "Lead-in connections" | new, alter or upgrade; no work-type gate |
  | `fire-active.smoke-alarms` upgrade | `fr.smoke-alarms-1a` | accepts upgrade |
  | `substructure.underpinning` alter or repair | `fr.underpinning-1a` | extend or refurb |
  | `electrical.utility-supply` alter or upgrade | `fr.utility-1a` | |
  | `site.erosion-sediment-control` alter | `fr.erosion-control` | |
  | `site.trees` alter or repair | `fr.tree-protection` | |

  With the **default** extend scope, linings, painting, floors and cabling are action `new`, so
  `fr.linings-1a`, `fr.painting-1a`, `fr.floors-1a` and `fr.electrical-1a` fire
  (`index/type-predictions.md` T05). The gap appears only once a document or user sets those items to
  `alter`.
- **Exact change:** in §7, replace "Every altered item gets no construction row" with "Alter and upgrade
  items on interiors, hydraulic, electrical, landscape and structure.alterations get no construction row
  on `extend` (list). Services connections, smoke alarms, underpinning, utility supply, erosion control
  and tree protection still fire." Reword §11 gap 1 and `gaps[0].what` the same way. Keep the multi-part
  note in §12. I confirmed in code that "extend,refurb" makes both families true; it is still untested
  end to end.

### F3. `built_in` uses two baselines (default scope vs this shape's scope). Severity: high (testability)

- **Where:** `shape.yaml program.activities` and `cost_plan`.
- **What is wrong:** some rows are judged against the shape's own `work_items`:
  - linings, painting and floors are `partial` because the shape sets them to alter.

  Other rows are judged against the 16-item default scope:
  - "Connect temporary builders power" is `partial`, "Needs an electrical.temporary-power item, not in
    the default scope", but `work_items` has `electrical.temporary-power: new`;
  - so are erosion, demolition, termite, slab, wet-area waterproofing, tiling and joinery, the survey
    and geotechnical investigations, and the cost lines "Site works and external works", "Mechanical
    services" and "Access and egress".

  Brief 01 defines `built_in` "for this profile and scope". I emulated `gen.fragmentRows` over the
  shape's `work_items` and profile (scratch script, not kept). Every one of those fragments returns
  **true**.
- **Exact change:** judge everything against the shape's `work_items`. Set `built_in: "yes"` on:
  - activities `fr.investigate-survey`, `fr.investigate-geotech`, `fr.temporary-power`,
    `fr.erosion-control`, `fr.demolition-1a`, `fr.termite-1a`, `fr.slab-on-ground-1a`,
    `fr.wet-area-1a`, `fr.tiling-1a` and `fr.joinery-1a`;
  - cost lines "Site works and external works", "Mechanical services" and "Access and egress".

  Rewrite each `why` to say what fires. Keep the shortfall of the default scope in `gaps[1]`, where it
  already is.

### F4. Activities the app produces are missing from the key. Severity: medium

- **Where:** `shape.yaml program.activities`.
- **Evidence:** the same emulation also fires these fragments for the shape's own items, and the key
  has no row for any of them:
  - `fr.services-connections-1a` (the sewer and water alteration that defines this job);
  - `fr.earthworks-1a`, `fr.damp-1a`, `fr.stormwater-1a`, `fr.ceilings-1a`, `fr.internal-doors-1a`;
  - `fr.hydraulic-hot-1a` (replace), `fr.lighting-1a`, `fr.ac-1a`, `fr.roof-drainage-1a`.

  A scorer would count them as extras or misses.
- **Exact change:** add one activity per fragment above, with `built_in: "yes"` and logic. For example,
  "Lodge services applications" and "Lead-in connections" go after "Isolate, demolish and clear", and
  "Install roof drainage" goes after "Install roof cover". Or state in the header comment that the
  activity list is not exhaustive and the scorer must ignore unlisted fragment rows. The first is
  better.

### F5. Program logic contradicts DOMAIN §7. Severity: medium

- **Where:** `shape.yaml` activity "Prop and form openings in the existing structure"
  (`after: ["Isolate, demolish and clear"]`).
- **What is wrong:** DOMAIN §7, step 11, and the "Break through" activity both say openings in the
  existing rear wall are formed only once the addition is weathertight. The shape forms them before the
  footings, which would open a vacant house to the weather for the whole build.
- **Exact change:** split it into two activities:
  - "Remove internal loadbearing walls on props", after "Isolate, demolish and clear" (allowed because
    the house is vacant);
  - "Form the opening in the existing rear wall", after "Break through into the existing house".

  Both stay `fragment: fr.structure-alterations`, `built_in: partial`.

### F6. The Mechanical consultant is omitted from the predicted false positives. Severity: medium

- **Where:** `DOMAIN.md` §4 ("Mechanical would also appear once `mechanical.*` is in scope") and §6.
- **Evidence:** the shape's `work_items` include `mechanical.local-exhaust` and
  `mechanical.air-conditioning`. `scope_additions[2]` fires on any `mechanical.*` scope row
  (`knowledge/works/package_defaults.yaml`; KNOWLEDGE_INDEX §2.1). For this scope the app therefore
  suggests ten consultants, Mechanical among them, and the cost plan gets a Mechanical fee line.
- **Exact change:** state that Mechanical is suggested for this scope and is a false positive, because
  the installer designs the system. Add it to §6, to §11 gap 3 and to `gaps[2]`.

### F7. CDC is unavailable in a heritage conservation area, not only for a heritage item. Severity: high (domain)

- **Where:** `DOMAIN.md` §5 CDC paragraph ("not available for a heritage item"); §9 heritage row;
  `shape.yaml` trigger `heritage-conservation-area` `adds.authority`.
- **Evidence:**
  - Under the Housing Code's land exclusions, a heritage conservation area (or a draft one) excludes
    complying development except detached outbuildings, some detached development and pools [D].
    Alterations and additions to the house itself are not complying development there.
  - The repo's own seed says the same: `data/reference/clerk/data/seed/ancillary-guide.md` lines 72 and
    134 ("CDC pathway is blocked … heritage conservation area").
  - The clause number is not verified.
- **Exact change:** set `authority` to "Council heritage assessment within the DA; CDC not available for a
  heritage item or in a heritage conservation area (Housing Code land exclusions, clause `verified:
  false`)". Make the same edit in §5 and §9. In `cdc-pathway`, add the negation "The site is within a
  heritage conservation area, so complying development is not available."

### F8. Unverified statutory items (task item 3). Severity: low; none is invented

| Item | Status in map | Settled? |
|---|---|---|
| BASIX threshold for alterations and additions | "believed $50,000", `verified: false` | **Not settled against the instrument.** The figure appears only in seed prose (`renovation-guide.md:160`, `sustainability-energy-guide.md:281`, which itself says "confirm"). `rule.basix.certificate-required` is `clause_verified: false` and derives `pending`. Rule 5 lets the map keep the number only as `verified: false`. Correct as is. |
| BASIX rainwater tank for alterations and additions | `verified: false`, no number; `non-potable-water` is only conditional | Not settled. Correct as is: there is no tank in base scope and no number. |
| Housing Code part and exclusions | `verified: false` | Partly settled (see F7). The HCA exclusion is missing; no clause number should be added. |
| Long service levy threshold | `verified: false`, no number | Not settled; correct. Add no figure. Both the threshold and the rate have changed over time. |
| HBCF threshold | `verified: false`, no number | The seed cites $20,000 (`ancillary-guide.md:154-158`) and marks it an assumption. The map is right to carry no number. |
| NCC Housing Provisions 9.5, 9.2, 10.2; H3D6 | `verified: false` | Not settled. The repo holds no NCC 2022 Housing Provisions text. The only seed (`ncc-reference-guide.md`) quotes pre-2022 Volume Two numbering (3.7.2, 3.8). Keep them false. |
| WHS asbestos licensing, notification, 10 m² | `verified: false` | Not settled; correct. See F11 for the year threshold. |
| AS 4361.2 lead paint | `verified: false` | Correct standard; keep false until read. |

No price, rate or duration appears anywhere in the map. ✔

### F9. Existing-building interfaces are incomplete (task item 4). Severity: medium

`DOMAIN.md` §3 covers occupation, cut-overs, asbestos, lead paint, structure, matching work,
dilapidation, approval history and (as triggers) the HCA and party wall. These are missing for an
occupied or vacated older Sydney house:

1. **Subfloor ventilation and the slab-to-suspended-floor junction.** A 1950s brick house is usually on
   a ventilated timber subfloor. A slab-on-ground addition along the rear wall blocks the vents and
   breaks the termite inspection zone, and the floor levels step. Add this to §3 Structure and to the
   `substructure.damp-waterproofing` and `termite-management` `why` text [D].
2. **Ceiling-void lead dust.** In older inner-ring houses it is disturbed whenever ceilings are removed
   at the removed walls. Add it to the hazmat survey's must_include ("ceiling dust sampling") and to the
   `hazardous-materials-found` evidence ("lead dust in the ceiling void") [D].
3. **Stormwater wrongly connected to the sewer.** In older houses, rear downpipes are often found
   discharging to the sewer. Sydney Water requires the disconnection once it is found. Add it to the §3
   services cut-overs [D].
4. **Existing rooftop solar PV, NBN lead-in and gas meter.** These are removed or relocated at the roof
   tie-in or the addition footprint. Add them to the services cut-overs. Solar needs CEC-accredited
   reinstatement [D].
5. **A vacant house's insurance and security.** Home and contents cover on a vacant house under
   construction is usually voided or conditional. The builder's contract works insurance and the owner's
   policy must be lined up. Add this to §3 Occupation and as an owner cost line [D].
6. **Active termites or termite damage found on opening up.** It appears in §1 but not as an interface
   or trigger. Add it to §3 Structure, or as a trigger `termite-damage-found` [D].
7. **Temporary living arrangements.** Covered: base vacate, the variant, the cost line and the
   activities. ✔
8. **Heritage conservation area.** Covered as a trigger, apart from F7. ✔
9. **Underpinning and neighbour dilapidation.** Covered by the `first-floor-addition` and
   `neighbour-dilapidation` triggers. ✔

### F10. Missing critical-stage inspection and DA documents. Severity: medium (domain)

- **Where:** `DOMAIN.md` §5 inspections and DA package; `shape.yaml program`.
- **What is wrong:**
  - For Class 1 work, the principal certifier's critical-stage inspections also include "before covering
    stormwater drainage connections" [D, `verified: false`]. It is missing from the list.
  - The DA package omits a demolition waste management plan and an erosion and sediment control plan.
    Both are routine council lodgement items for a demolition-plus-addition DA [D].
- **Exact change:**
  - add "stormwater drainage connections before backfill" to the §5 inspection list (still `verified:
    false`);
  - add a `built_in: "no"` activity "Hold point: stormwater drainage inspection before backfill" after
    "Install roof drainage" (F4);
  - add both plans to the §5 DA package.

### F11. Hazmat trigger year is inconsistent with the app and WHS practice. Severity: low

- **Where:** `DOMAIN.md` §3 ("before about 1990"), §4 Hazmat row ("Always for a pre-1990 house").
- **Evidence:** the app's survey consequence uses `existing_building_year < 2004`
  (`cq.pre-2004-fabric-hazardous-materials-survey`). The WHS duty to identify asbestos before demolition
  or refurbishment follows the 31 December 2003 ban [D, `verified: false`].
- **Exact change:** in the §4 Hazmat row, write "Always for a house built before 2004 (WHS, `verified:
  false`); asbestos is likely before about 1990". Change the `hazardous-materials-found` negation "The
  house was built in 2009" to a realistic survey denial on a 1958 house: "Samples 1–6 of the eaves and
  wet-area linings: no asbestos detected; the fibre-cement eaves date from the 1996 re-roof."

### F12. RFP and trigger built-in claims marked too high. Severity: medium (testability)

1. **`rfps` structural `must_include` "… with propping sequence".** In `knowledge/rfp/structural.yaml`,
   `temporary_works_design` is `X` (the contractor's temporary works designer). As written, the key
   expects scope that the template correctly excludes. Change it to "Beams, posts and bearing pads over
   new openings and removed loadbearing walls". Add to `must_not_include`: "Temporary propping design
   (builder's temporary works designer)".
2. **The `first-floor-addition` RFP claim is `rfps: "yes"`, but the structural template has no
   underpinning design row.** Underpinning appears only in `temporary_works_interfaces`, which is gated
   on `basement` or `excavation_adjoining`. Change to `rfps: partial`. "Capacity of the existing walls"
   is `existing_assessment` and fires on extend.
3. **`significant-tree` claims `rfps: "yes"`, but no package rule suggests an Arborist**
   (KNOWLEDGE_INDEX §2.1, §6), so the RFP exists only if the user creates the package. Change to
   `rfps: partial`.
4. **`significant-tree` adds `site.trees` with action `retain`.** `retain` items bring no fragment
   (`fragmentRows`), and `fr.tree-protection` needs new, alter or repair. Change the program claim to
   `program: "no"`, and add a gap: "Retained trees get no tree-protection activity; the fragment needs
   new, alter or repair."
5. **`semi-detached-party-wall` has `profile: [det.ncc_class]`.** A semi-detached house is still Class
   1a, so the key changes nothing. Set `profile: []`, unless you mean a Class 10a structure.

### F13. Profile keys the app cannot set. Severity: low (harness)

- **Where:** `shape.yaml profile` `det.basix_required: true` and `det.energy_compliance_pathway: basix`.
- **Evidence:** in `knowledge/determinants.yaml`, neither has a `profile_group`, and `basix_required`
  is `derived: true` by `rule.basix.certificate-required`, whose derivation is `pending`. The API sets
  only determinants that have a `profile_group` (KNOWLEDGE_INDEX §7.2), so an API-driven harness cannot
  enter either key. No generator reads them.
- **Exact change:** move both keys to a comment, for example "expected derived values: basix_required
  true, energy pathway basix", or record them under `gaps` as "BASIX applicability is pending in the
  knowledge".

### F14. Missing common triggers. Severity: medium (completeness)

Add these to `shape.yaml triggers`. The first two belong in `variants` too:

- `stormwater-engineering-required` (F1).
- `sloping-site`:
  - evidence: "The rear yard falls 1.8 m to the back boundary; the addition is on a suspended floor
    with retaining to the cut."
  - adds: `site.retaining-walls` new, `substructure.piles` or piers, `access-egress.barriers` above
    1 m, and a geotechnical slope and retaining scope.
  - §2 now puts `site.retaining-walls` out of scope as "not part of a rear addition". That is too strong
    for Sydney.
- `termite-damage-found` (F9 item 6).
- `loose-fill-asbestos`:
  - evidence: "The property is listed on the NSW loose-fill asbestos insulation register."
  - consequence: no ordinary alterations; the owner must deal with the register process first
    (`verified: false`).
  - It is rare, but if missed the job cannot go ahead.
- `existing-solar-pv` (F9 item 4).

### F15. Small accuracy points. Severity: low

1. §4 BASIX row "Always". The BASIX *certificate* is always needed above the threshold. For alterations
   and additions, the architect often generates it with the DIY thermal method, so write "Usually; the
   architect may generate it". The base procurement entry can stay.
2. §5 "Neighbour notification" is fine. Keep the note that `fr.da-exhibition` is labelled "Public
   exhibition".
3. `cost_plan` "Construction certificate and certifier fees" and the "Certifier" fee line: the double
   count is a real app defect. Keep both rows; the `why` already says so. ✔

## 3. Built-in claims checked

Procurement:

- ✔ Architect, Surveyor, Structural, Geotechnical, Certifier: `baselines[0]` (extend, residential).
- ✔ Civil `yes`: the app does suggest it (`baselines[0]`). It is wrong only as a base *need* (F1).
- ✔ `trade.main_works` partial: `ProposeHeadContract` runs only on accepted items.
- ✔ (no) BASIX: no package rule names it.
- ✔ (no) Hazmat: no package rule names it.

Cost plan:

- ✔ DA fees (`starter_lines` da only), CC and certifier fees, service authority fees, design
  contingency, construction contingency, escalation, exclusions.
- ✔ Fee lines per adopted package (`costs.Suggest`, adopted on prepare).
- ✔ Substructure, Structure, Envelope, Interiors, Active fire, Hydraulic, Electrical (`elements.yaml`).
- ✘ Site works, Mechanical and Access and egress are `partial`; they should be `yes` for this scope (F3).
- ✔ Owner supply partial (`applyBriefFigures`).
- ✔ `residential_extension` has no structure share and no preliminaries line (`budget_splits.yaml:61`).
- ✔ (no) long service levy, HBCF, make good, latent-conditions provisional sum, temporary
  accommodation: none is a starter or an element.

Program:

- ✔ `sk.extend-traditional` (extend, traditional, da).
- ✔ All project fragments: concept, design development, approval documents, `fr.da-lodgement`,
  `fr.da-exhibition`, `fr.da-assessment`, construction certificate, construction documents, tender,
  award, site establishment, PC, OC, handover, defects; and `fr.existing-conditions` (refurb or
  extend).
- ✔ `fr.footings-1a`, `fr.frame-timber-1a`, `fr.roof-1a`, `fr.windows-1a`, `fr.external-walls-1a`,
  `fr.insulation-1a`, `fr.smoke-alarms-1a` (accepts upgrade).
- ✔ Hold point `if.frame-inspected-before-roof`: `after: structure.timber`, `before: envelope`, both
  tagged, so it fires.
- ✘ `fr.investigate-survey`, `fr.investigate-geotech`, `fr.temporary-power`, `fr.erosion-control`,
  `fr.demolition-1a`, `fr.termite-1a`, `fr.slab-on-ground-1a`, `fr.wet-area-1a`, `fr.tiling-1a`,
  `fr.joinery-1a` are marked partial but fire for this scope (F3).
- ✔ `fr.structure-alterations`, `fr.hydraulic-sanitary-1a`, `fr.electrical-1a`, `fr.linings-1a`,
  `fr.painting-1a`, `fr.floors-1a`, `fr.landscape-1a` partial: false for alter on extend.
- ✘ Gap claim "every alter or upgrade item gets no row" (F2).
- ✔ (no) waterproofing hold point: `if.envelope-waterproofing-before-tiles` uses free text.
- ✔ (no) break-through, owners vacate, notice of commencement, planning certificate: no fragment.
- ✔ (no) reinforcement hold point: needs `structure.concrete`.

RFPs:

- ✔ Architecture and structural match `data/eval/rfp/answer-keys/small-class1-extension.yaml`.
  Architecture `required_facts: [budget]` and structural `existing_assessment` on extend both confirmed.
- ✘ Structural "propping sequence" conflicts with `temporary_works_design: X` (F12.1).
- ✔ Certifier partial: `bca-certification` boundary and CDC block.
- ✔ Civil partial: no house module; roads element always shown.
- ✔ Surveyor and geotechnical partial.
- ✔ (no) BASIX and Hazmat templates (`index/rfp-templates.md`).

Triggers:

- ✔ Heritage: procurement `complexity_additions[1]`, program `fr.heritage-referral`.
- ✔ Bushfire: procurement via `det.bal`, program `fr.bushfire-1a` with an item.
- ✔ CDC program `fr.cdc-application`; RFPs block.
- ✔ Flood: `fr.flood-1a`, civil `flood_report`.
- ✔ Electrical supply upgrade `fr.utility-1a` (alter).
- ✔ Building over sewer: structural `authority_asset`.
- ✔ Neighbour dilapidation: structural `dilapidation` O under `excavation_adjoining`.
- ✔ Occupied: `live_site` partial; `fr.occupied-protection` false on extend.
- ✔ Hazmat: `fr.existing-conditions` partial.
- ✘ First-floor `rfps: yes` (F12.2).
- ✘ Significant tree `rfps: yes` and `program: partial` (F12.3, F12.4).

Knowledge gaps:

- ✔ The 16 default extend systems are all `new`.
- ✔ `det.existing_building` triggers only on "existing building".
- ✔ Lead paint: 1970 in the determinant notes vs `uc.lead-paint-on-old-windows`.
- ✔ No asbestos or lead rule in any `rules.yaml` (grep found none).
- ✔ No dilapidation discipline id.

---

## Round 2 (re-review after the author applied round 1)

Read 29 October 2026. The validator still passes (`ok`, exit 0). Because the folder is untracked, there
is no diff to read. I read the whole of `shape.yaml` (672 lines) again and checked `DOMAIN.md` by
searching it for each fix. I re-ran the fragment emulation for the new trigger work items. The variant
count of 9 is the orchestrator's decision and is not a finding.

### Round 1 findings

| # | Finding | Applied | Evidence |
|---|---|---|---|
| F1 | Civil becomes a trigger | ✔ | Civil is gone from the base `procurement`, `cost_plan` and `rfps`. A new trigger `stormwater-engineering-required` uses `flag.osd_required` and `flag.authority_asset` and carries the civil RFP; it is also in `variants`. DOMAIN §1, §4 (row "Trigger"), §5, §6, §7 and §11 gap 3 ("five wrong fee lines") are consistent. |
| F2 | "Every altered item" claim corrected | ✔ | DOMAIN §7 lists both the alter items that get no row and those that still fire. `gaps[0]` and §11 gap 1 are rewritten. |
| F3 | One baseline for `built_in` | ✔ | All ten activities and the three cost lines are now `yes`, each with a "fires for this scope" reason. |
| F4 | Missing fragment rows | ✔ | Added: services connections (2 rows), earthworks, damp, stormwater, gutters and downpipes, hot water, ceilings, internal doors, luminaires, air-conditioning. Every one fires in my emulation. |
| F5 | Split the propping activity | ✔ | "Remove internal loadbearing walls on props" comes after demolition. "Form the opening in the existing rear wall" comes after "Break through into the existing house". |
| F6 | Mechanical is a false positive | ✔ | DOMAIN §4 and `gaps[2]` both include it. |
| F7 | CDC unavailable in an HCA | ✔ | The heritage trigger's `authority`, DOMAIN §5 line 355, and a new `cdc-pathway` negation. |
| F8 | Unverified items stay `verified: false` | ✔ | They are unchanged and still carry no invented number. |
| F9 | Existing-building interfaces | ✔ | DOMAIN §3 now covers subfloor vents and the junction (also in the `damp-waterproofing` and `termite-management` reasons), ceiling-void lead dust, downpipes to the sewer, solar, NBN and gas, vacant-house insurance (and a cost line) and termite damage. |
| F10 | Stormwater inspection; DA plans | ✔ | DOMAIN §5 has the inspection and the waste and erosion plans. The activity "Hold point: stormwater drainage inspection before backfill" is `no`. |
| F11 | Hazmat year; realistic negation | ✔ | §4 row says "before 2004 (WHS, `verified: false`)". The negation now quotes the samples and the 1996 re-roof. |
| F12 | Five RFP and trigger corrections | ✔ | Structural propping text is reworded and propping moved to `must_not_include`. First-floor `rfps` is partial. Tree `rfps` is partial and `program` is `no`, with a gap added. Party wall `profile: []`. |
| F13 | Profile keys the API cannot set | ✔ | They are now a comment plus a gap entry. |
| F14 | Missing triggers | ✔ | Added `stormwater-engineering-required`, `sloping-site`, `termite-damage-found`, `loose-fill-asbestos` and `existing-solar-pv`. |
| F15 | Small accuracy points | ✔ | BASIX is "Usually; the architect may generate it". The other points needed no change. |

**Not applied: 0 of 15.**

### New findings (introduced by the changes)

**R2-1. `flood-affected` has wrong `built_in` values. Severity: low-medium (testability).**
- **Where:** `shape.yaml` trigger `flood-affected`.
- **What is wrong:** it now adds `procurement: [consultant.civil]` and the cost line "Civil", but its
  `built_in` has no `procurement` key and says `cost_plan: "no"`.
- **Evidence:** `baselines[0]` always suggests Civil, so the app produces both the package and the
  Civil fee line. The dead `flood_exposure` rule is not what produces them.
- **Exact change:** `built_in: {procurement: "yes", program: "yes", rfps: "yes", cost_plan: partial}`.
  Here `partial` means the Civil fee line is produced and "Flood report" is not. Keep the existing
  `source`.

**R2-2. `termite-damage-found` adds a work item the base scope already holds. Severity: low.**
- **Where:** `shape.yaml` trigger `termite-damage-found`.
- **What is wrong:**
  - It adds `structure.timber` with action `repair`, but the base scope already has
    `structure.timber: new`. Work items are one per (part, system) (`ProposedWorks`), so on the
    single whole-house part a one-change variant would *replace* the addition's framing item, and
    `fr.frame-timber-1a`'s two rows would disappear.
  - It also adds `substructure.termite-management: new`, which is already in base, so that addition
    does nothing.
  - Its `profile: [det.termite_prone_area]` describes the area, not active termites, and no generator
    reads it (KNOWLEDGE_INDEX §6).
- **Exact change:**
  - drop both work items from `adds` and put the repair in the cost line and the program activities
    only; or state that the variant needs a second part (the existing house) to carry
    `structure.timber: repair`;
  - set `profile: []`.

**R2-3. `existing-solar-pv` leaves out a cost line it causes. Severity: low.**
- **Where:** `shape.yaml` trigger `existing-solar-pv`.
- **What is wrong:** `comms-security.carrier-lead-in: alter` creates a new top-level construction line
  (element code 11, `comms-security`; `knowledge/costs/elements.yaml`) that the base plan does not have.
- **Exact change:** add the comms-security line (its system label) to `adds.cost_plan`.

**R2-4 (note, no change needed).** `stormwater-engineering-required` reuses `flag.authority_asset`,
which the `building-over-sewer` trigger also uses. The flag also makes the structural scale complex.
That is acceptable, because a council pipe through the lot is an authority asset. A scorer should not
treat the two triggers as independent on that flag.

No new rule number, price, rate or duration was introduced. The new triggers' numbers ("1.8 m",
"2.1 m", "375 mm") are document wording, not rules.

### Round 2 verdict: **FIX (minor)**

The map is usable as an answer key once R2-1 is corrected. R2-2 and R2-3 are small and can be fixed in
the same pass. Nothing is wrong at the root.

## Owner rulings applied (30 October 2026)

From `testing/synthetic/reference/OWNER_RULINGS.md`:

- **R1 reverses the round 1 civil finding.** The civil (stormwater) engineer is now in the base.
  - It is a procurement package (`built_in: yes`, `baselines[0]` fires for `extend`), with a Civil fee
    line.
  - It has a civil RFP with an `expect` block and `app_differs`. `retained_services` is I through the
    `existing_building` module. `for_construction` and `as_built` are options after the boundary in the
    app, and the `roads` element is always shown.
  - New program activities: "Stormwater concept plan", "Stormwater design for the CC" and "Stormwater
    works-as-executed and engineer's certificate".
  - Civil is taken out of `not_expected_procurement` and `not_expected_cost_plan`.
- **Triggers.** `stormwater-engineering-required` is replaced by two variants:
  - `internal-only-alteration` (roof footprint and hard surfaces unchanged). Its `removes` takes out
    civil, its fee line and RFP, the stormwater item and its five activities. The app has no rule for
    this, so it keeps Civil.
  - `osd-or-drainage-easement` (OSD, easement, charged line or council pipe). It adds civil `osd_new`
    scope only.
  - `flood-affected` no longer adds the civil package.
  - DOMAIN.md §1, §2, §4, §5, §6, §7, §8, §9 and §11 now say the same.
- **R3.** "Class 1 is not regulated" is now cited as the ruling.
- **R4.** HBCF: over $20,000 including GST, as stated in §5, the HBCF line and the award activity.
  Section numbers stay `verified: false`.
- **R5.** BASIX for alterations: $50,000 or more, or a pool or spa over 40,000 litres. Confirmed in §4,
  the BASIX package and RFP, and `expected_derived`. The "believed to be $50,000" wording is gone.
- **R6.** Livable housing is optional. The "NSW did not adopt" claim is removed.
- **R7.** The §5 inspection list now has eight items.
  - Two activities added: "Pre-commencement inspection by the principal certifier" and "Final
    inspection by the principal certifier".
  - A reinforcement hold point is split out of the slab pour (`partial`; it needs `structure.concrete`).
  - The footings, frame, wet-area and stormwater hold points cite R7.
  - Clause 61 stays `verified: false`.
- **R8.** A DLP is carried. Its length stays `verified: false`.
- DOMAIN.md §12 has a new "Confirmed by the owner's rulings" list, with the source URLs.

Counts:

| | before | after |
|---|---|---|
| triggers | 19 | 20 |
| packages | 8 | 9 |
| cost lines | 34 | 35 |
| activities | 73 | 79 |
| RFPs | 7 | 8 |
| variants | 9 | 10 |

The validator passes.
