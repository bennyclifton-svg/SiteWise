# T01 New detached house: independent review

Reviewer: independent agent acting as a senior NSW residential PM. Brief: `testing/synthetic/briefs/02-review.md`.
Read 29 October 2026 against `main` plus the working tree. No file other than this one was written.
Nothing was run except the validator and read-only Python over the YAML.

## Verdict: **FIX**

The map covers every top-level system and the NSW house approvals chain well, and most fragment and package
claims check out. It should not become an answer key yet, for four reasons:

- several RFP claims marked `yes` would fail against the app as the base profile is written;
- one trigger (`builder-designs`) takes the app's novation behaviour as the right answer;
- the evidence for one variant (`trees-affected`) would correctly be read as *not* establishing the flag;
- a handful of `built_in` labels are wrong in both directions.

All of these can be fixed in place. Nothing is wrong at the root.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T01-new-house/shape.yaml` →
`ok`, exit 0, no warnings. Counts: 15 triggers, 11 packages, 32 cost lines, 76 activities, 10 RFPs.

YAML booleans: every `built_in`, `app_suggests` and `flag.*` value is quoted. The four bare booleans
(`det.existing_building`, `det.bushfire_prone_land`, `det.flood_hazard`, `det.basix_required`) are boolean
determinants, so they are correct. The validator does **not** check `det.*` values. I checked them against
`knowledge/determinants.yaml`, and all are valid options.

## 2. Findings

1. **`shape.yaml profile`: RFP flags left unset turn `must_not_include` items into priced options.** An unset flag
   is unknown, and `if_unknown: option` then prices the row (`unknownState`, `internal/rfp/resolve.go`).
   - Civil: the `public_road` element, `road_safety_audit` and `norbe` are all `if_unknown: option`
     (`knowledge/rfp/civil.yaml`).
   - Architecture: `sustainability_docs` (Green Star or NABERS) is `if_unknown: option`
     (`knowledge/rfp/architecture.yaml`).
   - Town planning: `panel_attendance` is `if_unknown: option`.

   All of these keys are settable flags in `knowledge/profile/flag_readings.yaml`. **Change:** add
   `flag.public_road: "no"`, `flag.norbe: "no"`, `flag.stormwater_quality: "no"`, `flag.sustainability_target: "no"`,
   `flag.panel: "no"`, `flag.community: "no"`, `flag.groundwater: "no"`, `flag.rail: "no"`,
   `flag.transfer_structures: "no"` and `flag.coastal: "no"` to `profile`. Decide `flag.architect_lodges`. Unset,
   it makes the architect's `lodgement` row "ask".

2. **`shape.yaml rfps[*]`: `built_in: "yes"` on RFPs whose `must_not_include` the app always prices as options,
   whatever the flags say.**
   - Civil: `water_quality` (MUSIC) is "I when flags, else **O**"; `civil_model` (3D model) is simple O.
   - Structural: `shared_model` (Revit) is simple O; `super_flat_floor` is `if_unknown: option`, and its flag is a
     client-decision flag the validator rejects.
   - Surveyor: `titling`, `interior` and `monitoring` are O always.
   - Geotechnical: `monitoring` is O; `g7_transfer` is option and `transfer` is O.
   - Town planning: `panel_attendance` is O even when `panel` is no.

   Evidence: the position tables in `testing/synthetic/reference/index/rfp-templates.md` for civil, structural,
   land-surveying, geotechnical and town-planning. **Change:**
   - (a) Give each RFP item an expected state (I / O / X / ask / absent), in the format of
     `data/eval/rfp/answer-keys/*.yaml`, so "option" can be told apart from "include".
   - (b) Until then, set `built_in: "partial"` on the civil, structural, surveyor, geotechnical and town-planner
     RFPs.
   - (c) Add the gap `{area: rfp, what: "House RFPs always price commercial options (MUSIC, 3D civil model, Revit
     model, super-flat floor, strata titling, interior scan, movement monitoring, groundwater wells, novation
     transfer, panel attendance)", evidence: "knowledge/rfp/civil.yaml water_quality, civil_model; structural.yaml
     shared_model, super_flat_floor; land-surveying.yaml titling, interior, monitoring; geotechnical.yaml
     monitoring, transfer; town-planning.yaml panel_attendance"}`.

3. **`shape.yaml rfps[architect]`: the RFP blocks in the base scenario.** `architecture.yaml` has
   `required_facts: [budget]`. The profile has no `proj.budget_stated`, and the cost plan writes no amount, so
   `checkRequiredFacts` raises the blocking check `fact:budget`. **Change:** the scenario documents state a
   construction budget (this is a scenario input, not an app price), and the shape records the expected block.
   Add to `why` "blocks on fact:budget until the budget is stated", and add a gap.

4. **`rfps[architect].must_include` claims text the template does not hold.** "incl. shadow diagrams and BASIX
   commitments" does not appear. The `planning_approval` deliverable reads "Architectural drawings and input to the
   {planning_report} …" (`architecture.yaml` line 94). **Change:** make it `"DA planning approval documentation"`,
   and record "shadow diagrams and BASIX commitments not named in the architect's RFP" as a domain-only gap.

5. **`rfps[structural].must_include`: "Inspection schedule …" is an option, not an include.** `inspection_schedule`
   is I only when `construction_phase` is in the RFP. That stage is `default: option`, and it is included only at
   complex scale, which a house is not. "Beams and lintels" has no position of its own; it is covered by
   `residential_framing`. **Change:** write `"Option: construction-phase inspections with an inspection schedule"`,
   and drop "Beams and lintels" or tag it as domain-only.

6. **`triggers[builder-designs]`: the app's behaviour is taken as the answer.** A project-home design and
   construct job has no owner-engaged design team to novate. The builder's in-house drafting and engineers produce
   the DA and CC set. The owner appoints only the principal certifier, and optionally an independent inspector [D].
   The trigger's own evidence ("the builder's drafting team will prepare the CC drawings") contradicts "Novate the
   design consultants". **Change:**
   - `adds.procurement`: remove the architect, structural, civil, geotechnical, surveyor, BASIX and landscape
     packages from the owner's list; keep the certifier.
   - `adds.program`: "Sign the builder's preliminary (design) agreement", "Builder prepares DA/CC set", "Sign the
     new-homes contract"; remove the tender rows.
   - `adds.rfps`: none for the design disciplines.
   - `built_in: {procurement: "no", program: "no", rfps: "no"}`.
   - Add the gap "D&C route gives every consultant `novation: option` and adds `fr.novation` and post-novation
     stages; wrong for a project home". Evidence: `internal/procurement/proposed.go BuilderDesigns`,
     `knowledge/program/fragments.yaml fr.novation`.

7. **`triggers[trees-affected].evidence`: two of the three phrasings do not establish the flag.**
   `flag_readings.yaml arborist_tree_impacts` asks whether works affect trees, and says "tree presence … alone does
   not establish an impact". "A large blue gum near the rear boundary" does not even match the trigger regex.
   "The neighbour's jacaranda overhangs the driveway" matches only `arborist_offsite_trees`. A correct Jev answer to
   either is "no". **Change:** use "the driveway cut encroaches into the tree protection zone of the neighbour's
   jacaranda" and "the rear slab edge is within the structural root zone of the blue gum". Keep "a large blue gum
   near the rear boundary" as a **negation** (present but unaffected).

8. **`triggers[cdc-pathway]`.**
   - (a) `built_in.cost_plan: "yes"` is wrong. The `da-fees` starter is dropped entirely for cdc, which also drops
     the s7.11/s7.12 contributions a CDC still pays. `cc-fees` (`applies_when: true`) stays, although no CC issues
     under a CDC. No CDC fee line exists (`knowledge/costs/starter_lines.yaml`). Set `"partial"` and add the gap.
   - (b) `built_in.procurement: "partial"` sits on an empty list. Town Planner is still suggested through
     `baselines[0]`, which is unconditional, and its RFP then blocks on cdc. Write `adds.procurement: []` with a
     why of "town planner not needed; app still suggests it", and set `"no"`.
   - (c) The negation "CDC is not available because the lot is in a heritage conservation area" is positive
     evidence for `heritage-conservation-area`, so a one-change variant test would fire two triggers. Replace it
     with "The design exceeds the Housing Code side-setback standard, so a DA will be lodged".
   - (d) Add the gap: `sk.new-traditional-cdc` has no `construction_documentation` stage, so
     `fr.construction-docs-1a` is placed in the nearest earlier stage (`resolveStage`,
     `internal/program/generate.go`), yet a CDC needs CC-level documents before lodgement.

9. **`triggers[reactive-site-class]`: H1 or H2 does not mean piers.** Under AS 2870, an H1 or H2 site normally gets
   a stiffened raft or waffle slab with deeper beams. Piers or engineered footings are for Class P (uncontrolled
   fill), E, or founding to rock [D]. **Change:** add `substructure.piles` only for the Class P evidence; give
   H1/H2 "deeper edge beams; engineer's footing inspection". Set `built_in.rfps: "no"`, because the geotechnical
   G6 inspection option is already in the base RFP, so the trigger adds no delta.

10. **`triggers[old-structure-on-lot]`: `built_in.program: "yes"` overclaims.**
    - `fr.demolition-1a` needs `site.demolition` with action `remove`. `implied_by strip-out` adds the scope row,
      and its action is the work-type default `new` (`works.DefaultAction`), so the fragment is false unless an
      action reading or the user sets `remove`.
    - "Asbestos survey" and "Asbestos clearance certificate" have no new-build fragment. `fr.existing-conditions`
      needs refurb or extend.

    **Change:** set `"partial"`, and add "No demolition is required" to `negation`, because it is keyword trap 1.

11. **`cost_plan` "Stairs, balustrades and barriers" `built_in: "no"` is wrong.** `elements.yaml` code 13 is
    itemised. `implied_by stair` adds `access-egress.stairs-and-ramps` from any "stair" in a read document, and
    every two-storey house set says "stair". That item yields the line "Stairs, ramps, handrails and nosings".
    Barriers never arrive. **Change:** split the line in two:
    - "Stairs, ramps, handrails and nosings": `partial`, source `knowledge/costs/elements.yaml code 13`;
    - "Compliant balustrades and barriers": `no`.

12. **`cost_plan` omits lines the app produces.** These are F-lines Electrical, Hydraulic and Mechanical (adopted
    on prepare, `prepareCostPlan`) and "07 Active fire protection" (type-predictions T01). A scorer will count them
    as unexplained extras. **Change:** add `not_expected_cost_plan` (mirroring `not_expected_procurement`) with
    those four lines. Also, DOMAIN §6 lists an HBCF row that shape.yaml lacks. Either add it to `cost_plan` with
    `built_in: "no"`, or drop it from DOMAIN.

13. **`program.activities` frame inspection order.** "Critical-stage frame inspection before linings" follows
    "Install wall and ceiling insulation". Wall batts conceal the bracing, straps and tie-downs the certifier
    inspects, so the inspection comes first [D]. **Change:** frame inspection
    `after: ["Rough-in sanitary, water and electrical services", "Install windows and external doors"]`;
    insulation `after: ["Critical-stage frame inspection before linings"]`; linings
    `after: ["Install wall and ceiling insulation"]`.

14. **`program.activities` lacks rows that the app produces or the map itself names.**
    - `fr.ceilings-1a` "Install ceilings" and `fr.internal-doors-1a` "Hang internal doors" are both true for T01.
    - "Stormwater drainage inspection before covering" (critical stage 5) is missing.
    - "Building set-out survey" and "Identification survey at slab" are named in DOMAIN §5 and gap 5 but are not
      activities.

    **Change:** add the two fragment rows as `yes`, and the other three as `no`.

15. **`program.activities` "Install air-conditioning": the app gets the logic wrong.** `fr.ac-1a` attaches
    `{after: fr.linings-1a}`. Ducted air needs its ducts in before the ceilings. The row exists, so `yes` stands
    for the item. **Change:** add the gap "`fr.ac-1a` follows linings; ducted rough-in must precede ceilings".

16. **`shape.yaml profile` and trigger profiles use keys the app cannot set.** `det.energy_compliance_pathway`,
    `det.basix_required`, `det.max_excavation_depth` and `det.max_retaining_wall_height` have no `profile_group`.
    `PUT /profile/det.<id>` takes only determinants that have one (KNOWLEDGE_INDEX §7.2). `basix_required` is
    derived by `rule.basix.certificate-required`. **Change:** drop the two base keys, or move them to an
    `expected_derived` note. Set `sloping-site.profile: []`, and add the gap "no settable key for slope, cut or
    retaining height".

17. **`work_items` `electrical.solar-pv` is in the base yet described as "conditional on the BASIX design".** That
    makes the base key ambiguous. **Change:** remove it from the base, or make it a trigger ("PV array to meet the
    BASIX energy target").

18. **Trigger `built_in` maps do not match `adds`.**
    - `gas-connection` claims `cost_plan` but adds none.
    - `flood-affected`, `builder-designs` and `cdc-pathway` claim `procurement` on empty lists.
    - `bushfire-prone`, `flood-affected`, `trees-affected`, `sewer-main-in-lot`, `basement-or-boundary-excavation`
      and `old-structure-on-lot` add `work_items` with no claim.

    **Change:** give every non-empty `adds` area a claim and remove claims on empty areas. The added work items are
    all `no`: no reading or rule produces them.

19. **Some triggers have no app-visible delta today: `stormwater-easement`, `reactive-site-class` and
    `gas-connection`.** No settable key, or no generator reads the key (`site_class` and `gas_supply`: triggers.md
    §1). The civil `discharge` position is already I in the base. **Change:** mark them in `gaps` as "zero-delta
    triggers: the harness expects no change". Set `stormwater-easement.built_in.rfps: "no"`.

20. **`heritage-conservation-area` `built_in` labels.**
    - `cost_plan: "partial"` should be `"yes"`: the Heritage package from `complexity_additions[1]` is adopted on
      prepare and gets a fee line.
    - `program: "yes"` should be `"partial"`: "Heritage impact statement" has no fragment, and only
      `fr.heritage-referral` exists.

21. **`sewer-main-in-lot` `built_in.cost_plan: "no"` should be `"partial"`.** Piers fall in the existing "02
    Substructure" line. Only the asset-assessment line is missing.

22. **Missing common triggers.** Name these, and add at least the first two:
    - acid sulfate soils mapped on the LEP (`det.acid_sulfate_soils_mapped` → civil `earthworks_suitability`,
      the only reacting row);
    - coastal or salt-spray exposure (`flag.coastal` exists; durability, wind class, Coastal Management SEPP);
    - upper-floor balcony or deck over a habitable room (`envelope.external-waterproofing`, a top defect source);
    - an overhead power line or a DNSP easement over the frontage.

    The less common variants that DOMAIN §9 leaves out are reasonably excluded.

23. **DOMAIN §5 critical-stage inspections.** The six stages match my recollection of the NSW list for Class 1
   and 10:
   1. after excavation and before footings;
   2. before pouring in-situ reinforced concrete;
   3. before covering the framework;
   4. before covering wet-area waterproofing;
   5. before covering stormwater connections;
   6. after completion, before the OC.

   A pool adds a seventh: after the barrier is erected and before the pool is filled. I believe the 2021 provisions
   sit in the *EP&A (Development Certification and Fire Safety) Regulation 2021*, not the general EP&A Regulation
   2021. I cannot confirm that offline. **Change:** rename the instrument, keep `verified: false`, and add the pool
   stage to `swimming-pool.adds.program`.

24. **DOMAIN §2 and §5 NCC references.** In NCC 2022, house DTS detail sits in the ABCB Housing Provisions Standard
   (Parts 9.2, 9.5, 11.2, 11.3), called up from Volume Two. **Change:** write "ABCB Housing Provisions Part 9.5
   (verified: false)" and so on.

25. **`shape.yaml` strings carry clause numbers without `verified: false`.** These are s138 (`work_items[7].why`,
   activity "Driveway…"), s7.11/s7.12 (`cost_plan[0].why`) and s10.7 (activity 1). I believe they are correct, but
   DOMAIN admits they are from memory. **Change:** append "(verified: false)".

26. **Uncertainties the author flagged:**
    - **Livable housing:** consistent with my understanding that NSW did not call up the NCC 2022 livable-housing
      provisions for Class 1a. Not confirmable offline, so keep `verified: false`. Out of scope stands.
    - **Critical stages:** see 23.
    - **HIA and MBA NSW defects period:** not settled; nothing in the repo holds it. **Change:**
      `fr.defects-liability-end` stays `yes`, because the app produces it. Add to its `why` "the domestic contract
      may carry no DLP beyond the HBA s18E warranties (verified: false); the key does not require a contract DLP".
    - **Certifier RFP defaults to advisory: confirmed.** `boundary` falls back to `a1_inception … a3_tender`, A
      stages resolve to ask, and B1–B4 are priced as options after the boundary (`resolveBoundary`,
      `resolveStages`, `internal/rfp/resolve.go`). No code sets `bca_certification` from `consultant.certifier`.
    - **CDC blocking: confirmed.** `cdc` is in `unsupported_pathways` for both `bca-certification.yaml` and
      `town-planning.yaml`, and `checkPathway` raises the blocking check `pathway_unsupported`.

## 3. Built-in claims checked

**Procurement**
- architect, surveyor, geotechnical, structural, civil, certifier (`baselines[0]`) ✔
- town_planner (`complexity_additions[0]`; also `baselines[0]`) ✔
- trade.main_works partial (`ProposeHeadContract` exists; gaps only on accepted items) ✔
- basix no ✔ · landscape no ✔ · quantity_surveyor no ✔
- not_expected: electrical, hydraulic (`baselines[0]`), mechanical (`scope_additions[2]`) ✔

**Cost plan**
- da-fees ✔ · cc-fees ✔ · service-authority partial ✔
- Architect, Structural, Civil, Geotech, Surveyor, Certifier, Town planner F-lines ✔
- elements 01, 02, 03, 04, 05, 08, 09, 10, 11 ✔
- design contingency, construction contingency, escalation, exclusions ✔
- owner supply partial (`applyBriefFigures` in `internal/store/cost_budget.go`) ✔
- spot-checked `no`: long service levy ✔ · road-reserve fees ✔ · BASIX ✔ · landscape ✔ · preliminaries ✔ ·
  provisional sums ✔ · **stairs/balustrades ✘** (finding 11)

**Program**
- skeleton `sk.new-traditional-da` ✔
- fragment rows, gates and labels (`program-fragments.md`, type-predictions T01) ✔: concept-design-1a,
  design-development-1a, approval-docs-1a, da-lodgement, da-exhibition, da-assessment, construction-docs-1a,
  tender-docs-1a, tender-period-1a, tender-evaluation-1a, contract-award, site-establishment-1a, erosion-control,
  services-connections-1a, utility-1a, earthworks-1a, footings-1a, slab-on-ground-1a, termite-1a, frame-timber-1a,
  roof-1a, roof-drainage-1a, windows-1a, external-walls-1a, hydraulic-sanitary-1a, ac-1a (logic ✘, finding 15),
  insulation-1a, linings-1a, wet-area-1a, tiling-1a, joinery-1a, electrical-1a, painting-1a, floors-1a,
  stormwater-1a, landscape-1a, practical-completion, occupation-certificate, handover, defects-inspection,
  defects-liability-end
- partial: construction-certificate (approvals before construction documentation) ✔ · investigate-services,
  investigate-survey, investigate-geotech (need investigate) ✔ · temporary-power ✔ · masonry-1a ("brick veneer"
  does not match `implied_by masonry`) ✔ · reinforcement hold point (needs `structure.concrete`) ✔
- frame hold point `if.frame-inspected-before-roof` (structure.timber → envelope) ✔
- spot-checked `no`: rainwater tank ✔ · stair ✔ · driveway ✔ · waterproofing inspection ✔ · BASIX ✔ · Tap in ✔ ·
  HBCF ✔

**RFPs**
- architect yes: **✘** (findings 1–4)
- structural yes: **✘** (2, 5)
- civil yes: **✘** (1, 2)
- geotechnical yes: **✘** (2)
- surveyor yes: **✘** (2)
- certifier partial ✔
- landscape partial ✔
- town planner yes: **✘** (1, 2)
- QS partial ✔
- basix no ✔

**Triggers**
- cdc: program ✔, cost_plan ✘, procurement ✘, rfps no ✔
- bushfire: procurement ✔ (any `det.bal`), program partial ✔, rfps no ✔, cost partial ✔
- flood: rfps ✔ (`flood_report` via `det.flood_hazard`), program partial ✔, procurement ✘ (empty list)
- heritage: procurement ✔, program ✘, rfps partial ✔, cost ✘
- trees: procurement no ✔, program partial ✔, rfps ✔, cost no ✔
- sloping: work_items no ✔, cost partial ✔, program no ✔, rfps partial ✔
- stormwater-easement: rfps ✘ (no delta)
- osd: rfps ✔ (`osd_new`), cost partial ✔, program no ✔
- sewer main: rfps ✔ (structural `authority_asset`, architecture `water_asset` O), program no ✔, cost ✘
- reactive: rfps ✘, the rest no ✔
- pool: all no ✔
- gas: work_items no ✔, program partial ✔, cost ✘ (no adds)
- basement: rfps ✔ (`retention_design`, `dilapidation`, civil `shoring`), cost partial ✔ (budget split), program
  no ✔ (`fr.basement-2` is class 2 only)
- builder-designs: app claims ✔ as app behaviour, ✘ as answer key (finding 6)
- old structure: program ✘, cost partial ✔, procurement no ✔

---

# Round 2

Re-read 29 October 2026, after the author applied round 1. Round 1 above is unchanged.

Validator: `ok`, exit 0, no warnings. The counts are now 17 triggers, 11 packages, 33 cost lines, 81 activities
and 10 RFPs. I also checked, with read-only Python, that every key in each `rfps[].expect` block (stages,
positions, elements) exists in its `knowledge/rfp/*.yaml` template. None is unknown, and every template stage is
covered by `stages_include`, `stages_option` or `stages_absent`.

## Round 1 findings: applied?

| # | Finding | Applied | Note |
|---|---|---|---|
| 1 | Unset RFP flags | ✔ | 11 flags set; `architect_lodges: "yes"`. Some still missing: see R2-1 |
| 2 | Options priced against `must_not_include` | ✔ | RFPs `partial`, gap added, `expect` and `app_differs` blocks added |
| 3 | Architect blocks on budget | ✔ | `scenario_inputs`, `expect.blocking`, gap |
| 4 | Shadow diagrams and BASIX text | ✔ | Moved to `app_differs` and a domain-only gap |
| 5 | Structural inspection schedule | ✔ | Now "Option: …"; beams covered by framing |
| 6 | `builder-designs` | ✔ (partly) | Consequences corrected, but the removals live only in `why` prose: R2-5 |
| 7 | Tree evidence | ✔ | Two impact phrasings; presence-only phrasing moved to `negation` |
| 8 | CDC cost, procurement, negation, skeleton gap | ✔ | All four parts applied |
| 9 | H1 is not piers | ✔ | `why` limits piles to Class P; `rfps` claim dropped |
| 10 | Demolition overclaim | ✔ | `partial`; "No demolition is required" added to `negation`; gap added |
| 11 | Stairs cost line | ✔ | Split into two lines: `partial` and `no` |
| 12 | Unexpected cost lines | ✔ | `not_expected_cost_plan`; HBCF line added |
| 13 | Frame inspection before insulation | ✔ | Logic links fixed |
| 14 | Missing activities | ✔ | Ceilings, doors, stormwater inspection, set-out and identification survey added |
| 15 | AC after linings | ✔ | Gap added |
| 16 | Unsettable keys | ✔ | Moved to `expected_derived`; sloping `profile: []`; gap added |
| 17 | Solar PV in base | ✔ | Removed from base and named in prose |
| 18 | Claims vs adds | ✔ | Every non-empty area has a claim; empty-area claims removed |
| 19 | Zero-delta triggers | ✔ | Harness gap added. `coastal-exposure` should join it: R2-6 |
| 20 | Heritage labels | ✔ | — |
| 21 | Sewer cost | ✔ | — |
| 22 | Missing triggers | ✔ | Acid sulfate and coastal added; balcony and power line in prose |
| 23 | Critical stages | ✔ | Instrument renamed; pool barrier stage added |
| 24 | Housing Provisions | ✔ | — |
| 25 | Clause markers | ✔ | — |
| 26 | Flagged uncertainties | ✔ | DLP `why` added; livable housing stays `verified: false` |

**Not applied: 0.** One (6) is only partly machine-readable.

## The author's choices

- **`flag.architect_lodges: "yes"`: sound, if the documents say so.** The flag's question needs an explicit
  statement, such as "design through to DA lodgement" (`flag_readings.yaml architect_lodges`). Add that wording
  to `scenario_inputs` so the scenario author writes it.
- **Budget stated in the scenario, and the RFP keyed to block without it: sound.** It keeps prices out of the
  shape and still tests the block.
- **Solar PV out of the base: sound.**
- **Acid sulfate and coastal triggers: sound, with two fixes.** Acid sulfate needs a base value (R2-1) and a
  corrected clause note (R2-7). Coastal changes nothing the app produces (R2-6).
- **`expect` and `app_differs` per RFP: sound in concept.** `expect` is the right half; `app_differs` is
  incomplete and free text (R2-3, R2-4).

## New findings

R2-1. **`profile`: six more unknowns resolve to rows the `expect` blocks call absent.** The base profile does not
set them, so these rows appear in the app's output:

| Unset value | Row it produces | Evidence |
|---|---|---|
| `flag.osd_existing` | civil `osd_existing` included (feasibility of keeping existing OSD on a vacant lot); structural `existing_assessment` priced as an option | `knowledge/rfp/civil.yaml` `osd_existing` `if_unknown: include` (line 170); `structural.yaml` `existing_assessment`, `if_unknown: option` |
| `det.potentially_contaminated_land`, `det.acid_sulfate_soils_mapped` | civil `earthworks_suitability` included | `civil.yaml` line 201 `if_unknown: include`; the `contamination` and `acid_sulfate` flags are read from these determinants (`internal/rfp/drivers.go` line 250) |
| `flag.country` | architecture "Connecting with Country" priced as an option | `architecture.yaml` line 329 `if_unknown: option` |
| `flag.staged`, `flag.multi_building` | architecture `brief_masterplan` priced as an option; surveyor `access` asked | both stages are `if_unknown` option or ask |

**Change:** add `flag.osd_existing: "no"`, `flag.country: "no"`, `flag.staged: "no"`, `flag.multi_building: "no"`,
`det.potentially_contaminated_land: false` and `det.acid_sulfate_soils_mapped: false`. The acid-sulfate trigger
then flips the last of these to true.

R2-2. **`rfps[architect].expect`: `bidding` is in `stages_include`, but the app prices it as an option.** The house
boundary ends at `house_documents`. `bidding` comes after it in `architecture.yaml`, so `resolveStages` gives
"After this appointment: priced as an option". Include is the right domain answer for an architect-led select
tender. **Change:** keep the expectation and add `bidding` to `app_differs`.

R2-3. **`app_differs` is incomplete.** For this profile the app also differs on:

| RFP | Where the app differs |
|---|---|
| Geotechnical | `defects` priced O; `ass`, `asset_exposure` and `clearance` asked (client flags) |
| Surveyor | `s5_support` priced O, after the boundary |
| Certifier | `existing_assessment`, `classification_map`, `staged_occupation` and `fire_referrals` asked (client flags) |
| Landscape | `s8_establishment` and `s9_advisory` priced O |
| Town planner | `pre_lodgement` included where the key says option |
| QS | `s3_authority` asked; `s6_construction` and `s7_completion` priced O where the key says absent |

For the QS, `stages_option` for s7 fits "Final account unless selected" better than `absent`. **Change:** add
each of these, and R2-2.

R2-4. **`expect` drifts from the answer-key format** (`data/eval/rfp/answer-keys/README.md`):
- civil uses `elements: {…}`; the format is `elements_include` / `elements_option` / `elements_absent` lists;
- `blocking` holds prose; the format is a list of check ids, so write `blocking: [fact:budget]` and put the
  condition in a `why` or `sources`.

**Change:** use the README's keys exactly, so the harness can feed `expect` to the same comparer
`TestSelectionAnswerKeys` uses.

R2-5. **`triggers[builder-designs]`: the dropped packages and tender rows exist only in `why`.** A scorer cannot
test a drop it cannot read. `adds.procurement: [consultant.certifier]` is not an addition either, because the
certifier is in the base. **Change:** set `adds.procurement: []` and add a machine-readable
`removes: {procurement: [consultant.architect, consultant.structural, consultant.civil, consultant.geotechnical,
consultant.surveyor, consultant.basix, consultant.landscape], program: ["Assemble the tender package", "Tender
period", "Evaluate tenders"]}`. The `cdc-pathway` trigger could use the same key to remove
`consultant.town_planner` and "DA fees and contributions".

R2-6. **`triggers[coastal-exposure]` changes nothing the app produces today.** No generator reads `flag.coastal`
(triggers.md, `coastal` → none) or `det.wind_class` (§1). Its claims are already `no`. **Change:** add it to the
zero-delta harness gap.

R2-7. **`triggers[acid-sulfate-soils].adds.authority`: "Standard Instrument cl 6.1" is not right.** Acid sulfate
soils is a model local clause, and its number varies by LEP. **Change:** write "the LEP's acid sulfate soils
clause (number varies by LEP; verified: false)". The trigger is otherwise sound:
- the evidence matches `acid sulf(ate|ide)`;
- the civil `earthworks_suitability` claim is correct (`partial`);
- the geotechnical `ass` position needs the client flag `geotech_ass`, so it is asked (`partial`).

## The new keys: well formed, and worth adopting?

All four belong in the shared schema, with small changes. The validator should check them.

| Key | Shape | Verdict |
|---|---|---|
| `not_expected_cost_plan` | list of `{line, section, source, why}` | **Adopt.** Mirrors `not_expected_procurement`. Add an optional `app_label` (for example "F04 Electrical") so a scorer matches the app's exact label. The validator should check `section` and `source` as it does for `cost_plan`. |
| `expected_derived` | list of `{key, value, source?, note}` | **Adopt.** These are values the app derives or cannot set, and they are checked, never PUT. The validator should check `key` with `_profile_key_ok`. |
| `scenario_inputs` | list of `{key, note}` | **Adopt.** These are facts the synthetic documents must state, and that the shape keeps out of `profile` (budget). Add an optional `phrasing` naming the wording the documents must carry, such as the `architect_lodges` statement. |
| `rfps[].expect` | the answer-key expectation keys: `stages_include`, `stages_option`, `stages_absent`, `stages_ask`; `elements_include`, `elements_option`, `elements_absent`; `positions` (key → I / O / X / ask / absent); `blocking` (check ids) | **Adopt**, using the README's keys exactly (R2-4). It states the **correct** answer for the job. The validator should check every key against the template's stages, positions and elements; my one-off check found T01 clean. |
| `rfps[].app_differs` | today, free-text strings | **Adopt only in structured form**, for example `- {key: stage.bidding, app: option, expect: include, why: "after the house_documents boundary"}`. Then the scorer can derive the app's expected output as `expect` overridden by `app_differs`, and count each difference as a known gap rather than a failure. |
| `triggers[].removes` (proposed, R2-5) | `{procurement: [...], cost_plan: [...], program: [...]}` | **Adopt** for every type whose variants take work away: CDC, D&C, owner-builder, exempt. |

## Round 2 verdict: **FIX** (minor)

The round 1 problems are fixed. What remains is mechanical:
- six missing base values (R2-1), the same class of problem as round 1 finding 1;
- completing and structuring `app_differs` (R2-2, R2-3);
- answer-key key names (R2-4);
- a `removes` block for `builder-designs` (R2-5);
- two notes (R2-6, R2-7).

None changes the domain content. After these the map is usable as an answer key, pending the owner's review
(`reviewed: false`).

## Owner rulings applied (30 October 2026)

From `testing/synthetic/reference/OWNER_RULINGS.md`:

- **R3.** The DBP Act applies to Class 2 and 9c, not to a Class 1a house. The architect RFP note now says
  so as a ruling, not `verified: false`.
- **R4.** HBCF is required where the contract price is over $20,000 including GST (SIRA). The principal
  contractor takes it out before starting work or taking a deposit. This is stated in DOMAIN.md §1 and
  §5, and in the HBCF cost line and program activity. Home Building Act section numbers stay
  `verified: false`.
- **R6.** Livable housing is optional for NSW houses, not part of the base scope. The claim that NSW
  "did not adopt" the provisions is gone. NCC clause references stay `verified: false`.
- **R7.** The DOMAIN.md §5 inspection list now has eight items: the pre-commencement inspection is added,
  and the pool inspection is now before the pool's in-situ concrete is poured and once the barrier is
  erected. The list cites Hornsby Council; clause 61 of the 2021 Regulation stays `verified: false`.
  In `shape.yaml`:
  - two activities added: "Pre-commencement inspection by the principal certifier" and "Final
    inspection by the principal certifier", with the site-start and OC links re-pointed;
  - the certifier RFP names the inspections;
  - `swimming-pool` gets two pool inspections and a certifier RFP item.
- **R8.** HIA and MBA contracts carry a defects liability period, so the line "may carry no DLP" is
  removed. The length stays `verified: false`.
- DOMAIN.md §12 has a new "Confirmed by the owner's rulings" list, and the resolved items are taken out
  of "least sure".

Counts: activities 81 → 83. Triggers 17, packages 11, cost lines 34 and RFPs 10 are unchanged. The
validator passes.
