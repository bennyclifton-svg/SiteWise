# RFP triage: run 20261102-fixes-1-4-6 (T01)

Scope: every RFP difference in `ST01-*/shape_score.json`. Each one is classed
against the report bodies (`ST01-*/snapshot/reports/*.json`), the templates
(`knowledge/rfp/*.yaml`), the resolver (`internal/rfp/resolve.go`) and the
profile (`ST01-*/snapshot/profile.json`, with misses checked in
`ST01-*/field_score.txt`). No code, knowledge, shape or scorer was changed.

Classes: **A** app defect, **B** knock-on from a profile value, **C** user
input expected, **D** key too strict or wrong, **E** scorer issue.

There are two kinds of difference. One is the `expect` "new failures"
(stages, positions, blocking: 105 in base). The other is the phrase results
(must_include missing 27 plus must_not_include present 5 in found RFPs, and
8 more phrases from the two RFPs that were never made).

## 1. ST01-base: `expect` new failures (105)

### E1. Inactive positions are read as X (E, 18)

- Keys: architect `position.country`, `bim_model`, `tenant_fitout`, `signage`,
  `facade_certification` and `community`; structural `retention_design`,
  `temporary_works_interfaces` and `strengthening`; civil `norbe`,
  `osd_new`, `osd_existing`, `osd_replacement`, `retained_services`,
  `flood_report`, `flood_evacuation`, `public_road_approvals` and
  `civil_declarations`.
- Evidence: `resolvePositions` (`resolve.go:917-921`) gives a row that does
  not apply (`"-"`) `Value "X"`, `Inactive true`, `Party "not required by the
  current brief"`. It keeps the row out of `r.out.Positions`, which is what the
  Go answer key calls `absent` (`answerkey_test.go`: `v == "absent" && got.Key != ""`).
  The controls group is built from `r.Scope` (`controls.go:189-196`). That list
  includes the inactive rows, so the report shows them as `value: "X"`, and
  the `inactive` flag is not carried onto the control. Every one of the 18
  controls has `party: "not required by the current brief"`. That string is
  set only at `resolve.go:919`.
- Fix (scorer, `tools/score_shapes.py` `_controls`, about line 535): for
  `kind == "position"`, treat a control whose `party` is "not required by the
  current brief" as absent (do not put it in `values`). I checked all three
  scenarios: no shape position expected as `X` is inactive in the app, and
  every `absent`-vs-`X` failure is inactive. The fix turns these 18 into ok
  and breaks nothing. A cleaner fix in the app would be to emit
  `inactive: true` on the control (`controls.go:194`), but that is app code.

### B1. Procurement route blank, F4 (B, 12 plus 2 blocking)

`hdr.cond.procurement_route` is blank with no sources (field_score: expected
`traditional`). `fact:procurement` is `null` / `unknown` in every RFP.
- Architect `stage.house_documents` is option. Its rule needs
  `procurement: [traditional, managing_contractor]`. The boundary falls back
  to the default `brief_masterplan..interiors` (`architecture.yaml:59-64`), so
  the stage is priced "After this appointment".
- `stage.dnc_tender` is option or ask: architect, structural (ask), civil.
- Structural `stage.tender_traditional` and `stage.construction_documents`
  are ask. Civil `stage.detailed_tender` is ask.
- `stage.post_novation` is option: architect, structural, civil. Its rule is
  `novation: [option, yes]`. The fact comes from the package (`drivers.go:166`),
  and every package has `novation: option` because
  `DefaultNovation("")` returns option (`procurement/proposed.go:11`
  `BuilderDesigns(route) = route != "traditional"`). This is still F4, but
  check that reading the route later resets novation on packages that
  already exist. I did not verify that.
- `blocking.fact:procurement`: structural and civil.
- The architect's issued positions table also prints `dnc_performance_brief`
  (I) and "Tender evaluation input". The shape does not score these.

### B2. Other profile values blank or in conflict (B, 20)

| Profile key (field_score) | RFP effect | Keys |
|---|---|---|
| `proj.client`, conflict_true (6 sources, 2 alternatives) | `blocking.details` ("Client. Set once in Profile") | 7: all seven found RFPs |
| `proj.address` / `site.address`, missed | `blocking.slot:project:intro` ("Site address not set") | 7 |
| `flag.staged`, `flag.multi_building`, missed | architect `stage.brief_masterplan` option (`if_unknown: option`); surveyor `position.access` ask; surveyor `blocking.fact:flag.staged` | 3 |
| `flag.authority_asset`, conflict_true (trap d03) | architect `position.water_asset` O; structural `stage.authority_asset` option | 2 |
| `flag.sustainability_target`, missed | architect `position.sustainability_docs` O | 1 |
| `proj.regulated_design`, missed (the architect's fee proposal HLA-FP-2611 says "This is a Class 1a house, so the Design and Building Practitioners Act registered-design obligations do no[t apply]") | architect `stage.regulated_design` option | 1 |
| `flag.groundwater`, missed | civil `position.earthworks_suitability` I (`if_unknown: include`; contamination and acid_sulfate are `no`) | 1 |

In ST01-cdc-pathway and ST01-trees-affected the client was read, and
`blocking.details` is gone. That confirms the details block is B.

### C1. Allowances, issue fields and terms the user types (C, 25)

- `blocking.fields` (issue and close dates) and `blocking.terms` (client
  agreement): 7 RFPs × 2 = 14.
- Blank client allowances (`source: none`): architect `client_presentations`
  and `planning_cap`; geotechnical `groundwater` and `reports`; surveyor
  `control`, `features`, `issues` and `visits`; certifier `duration_months`,
  `meetings`, `plan_sets`, `reports`, `rfi_hours`, `solution_versions`,
  `visits` and `witness_events`; town planner `authority_meetings`. That is 17.
- **Is `expect.blocking: []` wrong?** Yes, for a harness run. These checks
  correctly hold issue until the user types a value, and the harness never
  types. The architect `[]` came from SCORING §7.5, whose intent was "fact:budget
  must not block", and that still holds: fact:budget is not blocking. List the
  C ids (section 5). Leave B ids out, so that they keep failing until the
  profile is fixed.

### C2. Client scope decisions (C, 21 blocking plus 2 stages)

- `blocking.fact:flag.geotech_*` (6), `fact:flag.survey_*` (6) and
  `fact:flag.bca_*` (7). The skeleton marks them `scope_decision: true`,
  "not inferred from building type or silence" (`knowledge/rfp/skeleton.yaml:99-112`).
  The shape already records their stage and position effects as `app_differs`.
- Geotechnical `stage.g4_planning` is option. It needs
  `flag.geotech_planning` (`if_unknown: ask`), and it sits after the
  `g1..g3` boundary, so `resolveStages` prices it as an option
  (`resolve.go:697-703`).
- Certifier `stage.a4_construction` is option. It comes after the default
  a1..a3 boundary, which applies without `bca_certification`. The shape lists
  a1 to a3 and b2 to b4 but not a4.
- Fix: list the blocks in `expect.blocking`, and add the two stages to
  `app_differs` (section 5).

### D1. Structural inspection schedule (D, 1)

`position.inspection_schedule` is I in the app, and the shape wants O. The
template value is `{value: I, when: [{stage: [construction_phase]}], else: "-"}`
(`structural.yaml:239-243`). The row prices inside the construction-phase
stage, and that stage is an option (expected `stages_option`). Its optionality
is already checked through `stage.construction_phase`. Change the shape to `I`.

### Base `expect` tally

| Class | Architect | Structural | Civil | Geotech | Surveyor | Certifier | Town planner | Total |
|---|---|---|---|---|---|---|---|---|
| A | 0 | 0 | 0 | 0 | 0 | 0 | 0 | **0** |
| B | 9 | 8 | 7 | 2 | 4 | 2 | 2 | **34** |
| C | 4 | 2 | 2 | 11 | 12 | 18 | 3 | **52** |
| D | 0 | 1 | 0 | 0 | 0 | 0 | 0 | **1** |
| E | 6 | 3 | 9 | 0 | 0 | 0 | 0 | **18** |

## 2. ST01-base: phrase differences (32, plus 8 in missing RFPs)

### D2. Unkeyed phrases that do not reach 75 % of a control label (D, 22 missing plus 2 present)

The app has the row, and its state is right or already known. The prose
phrase's word coverage on the label is 0.14 to 0.67, so it falls through to
the text. I took the coverages from `find_phrase`.
- Architect (5): "Concept and schematic design". The template splits this
  into `stage.concept` and `stage.schematic`, both include, and the coverage
  is 0.67. Also "DA planning approval documentation" (`stage.planning_approval`
  include), "Tender (bidding) support" (`stage.bidding` option, a known
  difference), "Lodging the DA through the NSW Planning Portal"
  (`position.lodgement` I) and "construction-phase support"
  (`stage.construction_support` option).
- Structural (2): "Design basis …" (`position.design_basis` I, expect ok) and
  "AS 2870 footing and slab design" (`position.residential_framing` I).
- Civil (5): discharge (`position.discharge` I), detailed design
  (`stage.detailed_tender`, ask through B1), driveway (`element.roads`
  include), erosion (`element.earthworks` "Earthworks and sediment control"
  include) and WAE (`stage.as_built` option, known).
- Geotechnical (4): `stage.g2_field` (ask, known) twice, `stage.g3_report`
  and `stage.g6_construction`.
- Surveyor (6): `stage.s2_detail`, `position.adjoining`, `position.records`,
  `position.setout`, `stage.s6_construction` and `stage.s7_completion`.
- must_not_include present: structural "Authority asset stage" matches the
  fixed R1 boilerplate "Existing-structure and authority-asset stages are
  their own lines" (block `stages/rfp:stages`, coverage 1.0). It would still
  match after B2 is fixed. Geotechnical "Groundwater monitoring wells"
  matches the allowance text for the optional monitoring that
  `position.monitoring` O already marks as known.
- Fix: key each phrase (section 5). Three of them carry content the templates
  do not hold: the driveway long section and crossing levels, the rainwater
  tank overflow (`civil.yaml` has neither), and DCP and reactivity tests
  (`geotechnical.yaml`). Keep those as `text.*` app_differs, so that the
  owner still reads them.

### B3. Prohibited phrases present because of B1 or B2 (B, 3)

All three are on the architect RFP: "Green Star or NABERS"
(`flag.sustainability_target`), "Regulated-design declarations"
(`proj.regulated_design`) and "D&C tender documents" (F4).

### A1. Certifier content gaps (A, 2)

- "Every NSW critical-stage inspection … (R7)". `bca-certification.yaml` b3
  (`:104-106`) says only "scheduled inspections". There is no list for
  Class 1 and 10 (pre-commencement, footings, slab, frame, wet areas,
  stormwater, final). The best coverage is 0.17.
- "Occupation certificate with BASIX completion check". b4 (`:112-114`) does
  not name BASIX. The only BASIX row is `residential_evidence`, and it is a
  scope decision.

### A2. Known gaps the shape already names (A-known, 3 plus 4)

- BASIX (3): the RFP has no template (`template: null`, `built_in: "no"`),
  and its draft carries only generic scope.
- Landscape (4 phrases; the RFP is missing): no landscape package is proposed
  (procurement `built_in: "no"`, need "usually"; the SEE and the design
  statement have landscape sections).

### C3. Quantity surveyor RFP missing (C, 4 phrases)

The QS is `need: optional`, no base document names one, and the harness never
adds packages. No change is needed. Treat this as missed by design.

### Base phrase tally

A 9 (2 plus 3 plus 4), B 3, C 4, D 24. That is 40 phrases: the 32 the
summary counts, plus 8 in missing RFPs.

### Other observations (not scored)

- **A3, `flag.mixed_use` derived yes for a 1a house with a 10a garage.**
  `drivers.go:304-306` sets it when there is more than one class. Every RFP
  shows `fact:flag.mixed_use yes`, and the architect issues
  `position.mixed_use_map` I ("Class and use map by part"), which the firm
  sees in the positions table. This contradicts the `house` module
  (`modules.yaml`: "a Class 10 garage … is still a house").
- The electrical, hydraulic and mechanical RFPs score as `extra`, not
  `not_expected`. The scorer seeds `not_rfps` only from trigger removals
  (`expectations`). The cause is the known `app_suggests: yes` services
  baselines (procurement false positives). This is E-minor and changes no
  count.

## 3. ST01-trees-affected (98 new failures)

Only the trigger and run-to-run variation differ:
- `blocking.details` is gone from all seven RFPs, because the client was read
  this time (B −7).
- The arborist RFP is found. "Tree protection plan" and "Construction
  supervision hold points" pass. "AS 4970 impact assessment" is missing:
  `stage.s2_impact` is option by default (include_when
  `flag.arborist_impact_report`, unset), and the phrase reaches only 0.33 of
  a label. That is D: key it.
- **A4.** Structural "Footing design and location that avoid the structural
  root zone" (R9) is missing. `structural.yaml` has no tree, TPZ or SRZ row.
- Tally: `expect` A 0, B 27, C 52, D 1, E 18. Phrases A 10, B 3, C 4, D 25.

## 4. ST01-cdc-pathway (99 new failures)

The trigger is applied correctly (the town planner is removed and its RFP is
`not_expected`). The differences:
- The town planner's 5 are gone (B 2, C 3). `blocking.details` is gone from
  six RFPs (B −6).
- Architect `allowance:planning_cap` no longer blocks (C −1), and
  `stage.planning_response` is `exclude`, against a want of include. The app
  is right for a CDC. The trigger format cannot change `expect` keys, only
  phrases (SCORING §2). Class D: this stays as one standing difference until
  the scorer lets triggers override `expect`. Remove the phrase
  "Planning response to council requests" under the trigger.
- **A5, top.** The certifier RFP on a CDC has b1 to b4 `exclude` (4 new
  failures) and blocks on `pathway_unsupported` (1). `bca-certification.yaml:26`
  has `unsupported_pathways: [cdc, …]`, and every B stage is gated
  `pathway: [da, ssd]` (`:92-116`). As a result "Package B: CC assessment",
  "Principal certifier services" and "CDC assessment and determination" are
  missing (A, 3 phrases). The trigger records `built_in.rfps: "no"`.
- `stage.a2_planning` now shows as "fixed" only because the pathway gates it.
  This is a coincidence: keep the base app_differs.
- The town planner RFP is still made (not_expected_present 1). The trigger
  already says `procurement: "no"`.
- Tally: `expect` A 5, B 26, C 48, D 2, E 18. Phrases A 12, B 3, C 4, D 25.

## 5. Summary

| Class | ST01-base | ST01-cdc-pathway | ST01-trees-affected |
|---|---|---|---|
| A, expect | 0 | 5 | 0 |
| B, expect | 34 | 26 | 27 |
| C, expect | 52 | 48 | 52 |
| D, expect | 1 | 2 | 1 |
| E, expect | 18 | 18 | 18 |
| **expect new failures** | **105** | **99** | **98** |
| A, phrases | 9 | 12 | 10 |
| B, phrases | 3 | 3 | 3 |
| C, phrases | 4 | 4 | 4 |
| D, phrases | 24 | 25 | 25 |

### A items, in priority order

1. **A5. The certifier RFP has no CDC pathway.** `bca-certification.yaml`
   `unsupported_pathways` includes cdc, and the B stages are gated to da/ssd.
   On a CDC house the certifier RFP excludes every principal-certifier stage
   and blocks.
2. **A1. The certifier RFP omits the R7 critical-stage inspections for Class
   1 and 10, and the BASIX check at the OC** (all scenarios). This is also an
   owner direction question: a `consultant.certifier` package still opens as
   BCA advisory a1..a3, because `bca_certification` is a scope decision
   (already an app_differs in the shape).
3. **A3. `flag.mixed_use` is derived yes for Class 1a with 10a**
   (`internal/rfp/drivers.go:304`). It puts "Class and use map by part" into
   the architect RFP of an ordinary house.
4. **A4. The structural RFP has no SRZ or footing-avoidance row** (R9,
   trees).
5. **A2. Known gaps:** no landscape package, no BASIX template, the services
   engineers are suggested, and the town planner is still made under a CDC.
   There are also text gaps: the civil driveway long section, crossing levels
   and rainwater tank overflow, and the geotechnical DCP and reactivity tests.

The B items to fix upstream are F4 `hdr.cond.procurement_route`,
`proj.client`, `proj.address`, `flag.staged`, `flag.multi_building`,
`flag.authority_asset`, `flag.sustainability_target`,
`proj.regulated_design` and `flag.groundwater`. Fixing F4 alone clears 14.

### Recommended shape changes (D and C)

These YAML snippets apply to `types/T01-new-house/shape.yaml` `rfps:`. Each
one replaces the named key in that entry.

```yaml
# consultant.architect
    must_include:
      - {text: "Concept design", key: stage.concept}
      - {text: "Schematic design", key: stage.schematic}
      - {text: "DA planning approval documentation", key: stage.planning_approval}
      - "Planning response to council requests"
      - "Design development"
      - "House tender and construction documents (house_documents)"
      - "Kitchen, bathroom, laundry and joinery design"
      - "Finishes, fixtures and appliances selections schedule"
      - "Specification"
      - {text: "Tender (bidding) support", key: stage.bidding}
      - "Certifier liaison"
      - {text: "Lodging the DA through the NSW Planning Portal", key: position.lodgement}
      - {text: "Option: construction-phase support", key: stage.construction_support}
      - "Option: contract administration"
    expect:
      blocking: [allowance:client_presentations, allowance:planning_cap, fields, terms]
      # sources: keep the note that fact:budget must not block

# consultant.structural
    must_include:
      - {text: "Design basis with site class and AS 4055 wind class", key: position.design_basis}
      - {text: "AS 2870 footing and slab design", key: position.residential_framing}
      - "Framing, bracing and tie-down incl. review of supplier truss and frame design (covers beams and lintels)"
      - "Construction documents and design certificate for the CC"
      - "Option: construction-phase inspections with an inspection schedule"
    must_not_include:   # replace only this item
      - {text: "Authority asset stage", key: stage.authority_asset}
    expect:
      positions: {inspection_schedule: I}   # was O; optionality is stage.construction_phase
      blocking: [fields, terms]

# consultant.civil
    must_include:
      - {text: "Stormwater concept plan for the DA with discharge point confirmed", key: position.discharge}
      - {text: "Detailed stormwater design for the CC incl. rainwater tank overflow", key: stage.detailed_tender}
      - {text: "Driveway long section and boundary levels for the vehicle crossing", key: element.roads}
      - {text: "Erosion and sediment control plan", key: element.earthworks}
      - {text: "Works-as-executed review and stormwater certificate for the OC", key: stage.as_built}
    expect:
      blocking: [fields, terms]
    app_differs:   # add
      - {key: text.roads, app: "no driveway long section or crossing levels named", expect: "named", why: "civil.yaml roads element is generic"}
      - {key: text.detailed_tender, app: "no rainwater tank overflow named", expect: "named", why: "domain-only deliverable"}

# consultant.geotechnical
    must_include:
      - "Investigation plan"
      - {text: "Field work: boreholes or test pits with DCP tests", key: stage.g2_field}
      - {text: "Laboratory reactivity testing", key: stage.g2_field}
      - {text: "Report with AS 2870 site classification, founding depth and footing recommendations", key: stage.g3_report}
      - {text: "Option: pier and footing inspections", key: stage.g6_construction}
    must_not_include:   # replace only this item
      - {text: "Groundwater monitoring wells", key: position.monitoring}
    expect:
      blocking: [allowance:groundwater, allowance:reports, fact:flag.geotech_ass, fact:flag.geotech_asset_exposure, fact:flag.geotech_civil_parameters, fact:flag.geotech_foundation, fact:flag.geotech_handback, fact:flag.geotech_intrusive, fields, terms]
    app_differs:   # add
      - {key: stage.g4_planning, app: option, expect: absent, why: "client flag geotech_planning unset; after the g3 boundary it is priced as an option"}
      - {key: text.g2_field, app: "no DCP or reactivity tests named", expect: "named", why: "house-specific tests the template does not list"}

# consultant.surveyor
    must_include:
      - {text: "Detail and contour survey on AHD with trees and street features", key: stage.s2_detail}
      - "Boundary identification with easements"
      - {text: "Adjoining windows, ridge heights and trees for the DA", key: position.adjoining}
      - {text: "Sewer location from records", key: position.records}
      - {text: "Option: building set-out", key: position.setout}
      - {text: "Option: identification surveys at slab and frame or roof stage", key: stage.s6_construction}
      - {text: "Option: final survey for the OC", key: stage.s7_completion}
    expect:
      blocking: [allowance:control, allowance:features, allowance:issues, allowance:visits, fact:flag.survey_adjoining, fact:flag.survey_boundary, fact:flag.survey_buried, fact:flag.survey_detail, fact:flag.survey_services, fact:flag.survey_sewer, fields, terms]
      # fact:flag.staged is left out on purpose: it is B (flag.staged expected no)

# consultant.certifier
    expect:
      blocking: [allowance:duration_months, allowance:meetings, allowance:plan_sets, allowance:reports, allowance:rfi_hours, allowance:solution_versions, allowance:visits, allowance:witness_events, fact:flag.bca_advisory, fact:flag.bca_certification, fact:flag.bca_classification_map, fact:flag.bca_existing_assessment, fact:flag.bca_fire_referrals, fact:flag.bca_residential_evidence, fact:flag.bca_staged_occupation, fields, terms]
    app_differs:   # add
      - {key: stage.a4_construction, app: option, expect: absent, why: "after the a1..a3 boundary that applies without the client flag bca_certification"}

# consultant.town_planner
    expect:
      blocking: [allowance:authority_meetings, fields, terms]

# triggers: cdc-pathway, removes.rfps (add the dict item)
      rfps: [consultant.town_planner, {discipline: consultant.architect, must_include: ["Planning response to council requests"]}]

# triggers: trees-affected, adds.rfps arborist item
      {discipline: consultant.arborist, must_include: [{text: "AS 4970 impact assessment", key: stage.s2_impact}, "Tree protection plan", "Construction supervision hold points"]}
```

These are safe to apply. Every key and check id above appears in the base
reports or the templates. The blocking lists hold only C ids, so B blocks
(`details`, `slot:project:intro`, `fact:procurement`, `fact:flag.staged`)
still fail until the profile is fixed. Keying phrases loses word-level
checks; the `text.*` notes keep those for the owner's read. Run
`validate_shape.py` after editing.

Expected effect on base `expect` new failures: 105 falls to 34 (B only)
with the scorer fix E1 plus these changes. Phrase differences in found RFPs fall from 32 to 8 (A1 2, BASIX 3, B3 3);
the keyed `ask` rows (g2_field, s2_detail, adjoining, detailed_tender) move to
`unknown`.
