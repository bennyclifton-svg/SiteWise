# T08 Roof remediation: independent review

Reviewer: independent senior PM review, 30 October 2026. I did not edit `DOMAIN.md` or `shape.yaml`.

## Verdict: **FIX**

The map gets the core of the job right: a bonded AC re-roof on a live 7b warehouse, hazmat and structural
in the base, the keyword traps, and an honest list of what the app gets wrong. Five things need fixing before
it becomes a key. It cites a WHS instrument that has been replaced. Nobody is named to write the roof
specification. One trigger's reasoning about procurement is wrong. One `built_in` is overstated. The
friable-asbestos evidence would teach the wrong reading. The rest are smaller corrections and additions.

## 1. Validator

`python testing/synthetic/tools/validate_shape.py testing/synthetic/types/T08-roof-remediation/shape.yaml`
→ `ok  testing/synthetic/types/T08-roof-remediation/shape.yaml` (exit 0).

## 2. Rulings on the author's three open questions

**A. WHS Regulation clause numbers and the long service levy threshold.**

- *WHS.* The instrument name is out of date. The NSW legislation site lists the **Work Health and Safety
  Regulation 2025** (`sl-2025-0440`) as repealing and replacing the 2017 Regulation, which was due for
  repeal on 1 September 2025 ([NSW legislation](https://legislation.nsw.gov.au/view/whole/html/inforce/2025-10-03/sl-2025-0440)).
  The site refused my automated fetch (HTTP 403), so I could not read the 2025 numbering.
- The author's numbers match the 2017 (model WHS) numbering as I know it:
  - r 425 asbestos register; r 429 asbestos management plan;
  - r 464 asbestos removal control plan;
  - r 466 notice to the regulator, at least 5 days before work starts;
  - r 473 clearance inspection (independent); r 474 clearance certificate;
  - r 475 air monitoring, Class A.
- The author left out three provisions the map relies on:
  - r 458, the 10 m² non-friable exception to licensed removal;
  - r 78–79, managing the risk of falls;
  - r 291 and r 299, high-risk construction work and SWMS.
- **Ruling:** cite the 2025 Regulation. Label every clause number "2017 numbering". Keep each one
  `verified: false` until it is read from the 2025 instrument (finding 1).
- *Air monitoring.* r 475 makes air monitoring mandatory for **Class A** only. For the Class B base, air
  monitoring is a specification choice, a good one, and should be keyed as such.
- *Long service levy.* The rate is **0.25 % of the cost of the work, on work of $250,000 or more including
  GST**. It must be paid **before building or construction work begins**. This applies to work starting on
  or after 1 January 2023; before that it was 0.35 % from $25,000
  ([NSW Government, long service levy](https://www.nsw.gov.au/employment/rights-responsibilities/portable-long-service/bci-long-service-payments-scheme/long-service-levy)).
- **Ruling:** record the figure with that source. Keep it `verified: false` until the owner rules, as R4
  was handled. At the bottom of the $0.1–2 M range the levy line is not expected (finding 6).

**B. Roof safety as a design-and-install item in the head contract, with roof access consultant only as a
trigger.**

- **Ruling: correct for the base.** On a single-tenant warehouse re-roof, the anchors, static lines,
  walkways and ladder are normally a specialist height-safety subcontract under the roofer. That specialist
  designs, installs, load-tests and certifies them (AS/NZS 1891.4, AS/NZS 5532, AS 1657, verified: false).
  An independent roof access consultant is the exception: portfolio owners, roof plant, or an owner who
  wants an independent layout.
- Two fixes follow:
  - Under the traditional route, the app cannot represent that design-and-install element (finding 7a).
  - The consultant's only trigger is roof plant. Give it its own evidence (finding 7b).

**C. Hydraulic engineer in the base.**

- **Ruling: keep it in the base**, with roof-drainage scope only, as mapped.
- In this base, the box gutters are replaced and resized to current intensities, and gutter overflow into
  an occupied warehouse is often why the job exists. That needs catchment, sump and overflow design with PI
  behind it (AS/NZS 3500.3, verified: false).
- This differs from T09 (R2), where the remedial building consultant details balcony outlets. Here the
  4,000 m² roof drainage is a real design commission.
- The "eaves gutters only" exception in DOMAIN §12 is right. It belongs in the scenario author's notes. No
  change to `shape.yaml`.

## 3. Findings

1. **shape.yaml `program[0].why`, `[16].why`, `[18].why`, `[23].why`; `triggers[friable-asbestos].adds.authority`
   and `.rfps`; DOMAIN.md §1, §3, §5, §12. Wrong instrument.**
   - *What:* the map cites "WHS Regulation 2017" throughout. It has been repealed and replaced by the WHS
     Regulation 2025 (see ruling A).
   - *Change:* replace "WHS Regulation 2017 r NNN" with "Work Health and Safety Regulation 2025 (NSW)
     (2017 numbering r NNN; 2025 clause verified: false)".
   - *Add:* r 458 (10 m² exception), r 78–79 (falls) and r 291/r 299 (HRCW, SWMS) to the DOMAIN §12
     unverified list, under the same label.
   - *Change:* in `rfps[consultant.hazmat].must_include` item 6, add "(specified; mandatory only for
     Class A, r 475, verified: false)".

2. **shape.yaml `procurement`; `program` "Roof replacement specification and drawings", "Tender documents";
   DOMAIN §4, §7. Nobody writes the roof specification or runs the contract.**
   - *What:* the base names hazmat, structural, hydraulic and the roofer. The documentation row (sheet
     profile and coating, fixing pattern for the wind actions, flashings, skylights, safety mesh, roof
     safety layout, sequence, hold points) and the AS 4000 superintendent role have no package. "The PM is
     the SiteWise user" does not cover a technical roofing specification. A PM rarely writes one.
   - *Evidence:* T09 under owner ruling R2 makes the remedial building consultant (`consultant.building`)
     the lead who specifies, tenders and administers. T08 has the same shape of job.
   - *Change:* add a base procurement entry
     `{discipline: consultant.building, kind: services, built_in: "no", accept: [consultant.building, consultant.structural], why: "Roofing or building consultant: writes the roof replacement specification and drawings, runs the tender, acts as superintendent and holds the sheeting and roof safety hold points. Where the structural engineer's commission carries the roof sheeting and fixing specification, that satisfies it. No package rule names it; remediation has no baseline."}`.
     Add a matching fee line (`consultants`, `built_in: "no"`).
   - Put this to the owner as a question if it is not accepted as written.

3. **shape.yaml `triggers[sprinklered-building].why`. The procurement reasoning is wrong.**
   - *What:* the why says that, as `retain`, no Fire Services consultant is suggested, and that only
     alter or upgrade would bring one.
   - *Evidence:* `profile.ProjectWorkScope` (`internal/profile/works.go` lines 76–92) writes
     `scope.<system> = in` for **every** included work item, whatever its action.
     `procurement.SuggestedPackages` and `PackageBaselineFacts` never read the action (no `action` or
     `retain` in `internal/procurement/defaults.go` or `internal/profile/package_defaults.go`). So
     `fire-active.sprinklers retain` fires `scope_additions[1]`, and the app suggests Fire Services.
   - *Change:* rewrite the why to "A retain item still puts the system in scope, so scope_additions[1]
     suggests a Fire Services consultant the job does not need; the fire services contractor manages the
     impairment."
   - *Add* a `gaps` entry: "Scope additions fire on any action, including retain, so a protected or
     retained service brings a consultant." Other maps with `retain` items should check this too.

4. **shape.yaml `triggers[cdc-pathway].built_in.program: "yes"`. Overstated.**
   - *What:* `fr.cdc-application` gives two of the four listed activities (prepare and lodge; certifier
     determination). No fragment gives notice of commencement or the principal certifier appointment.
     `fr.occupation-certificate` is gated to work type new, extend or refurb.
   - *Change:* `program: partial`.

5. **shape.yaml `triggers[friable-asbestos].evidence[0]`, `[2]`. Wrong evidence.**
   - *What:* weathering, moss or broken sheets do not make AC friable. Friable means powder, or material
     that can be crumbled, pulverised or reduced to powder by hand pressure when dry (WHS Regulation
     definition, verified: false). A scenario that says "severely weathered … moss" or "cannot be removed
     in whole sheets" would teach Jev that weathered AC is Class A. Hygienists' reports state the
     friability finding explicitly.
   - *Change:* evidence[0] → "Sampling and inspection found the fire-damaged sheeting in bays 4 to 6 can be
     crumbled by hand pressure; the hygienist classifies it as friable asbestos." evidence[2] →
     "Removal of the damaged sheeting must be carried out by a Class A licensed removalist."
   - *Add a negation:* "Although heavily weathered and moss-covered, the AC sheeting remains bonded
     (non-friable)."

6. **shape.yaml `cost_plan` "Long service levy" `why`; `program` "Pay long service levy before work starts"
   `why`; DOMAIN §5, §12. Threshold and rate.**
   - *Change:* replace "threshold, rate and timing verified: false" with "0.25 % of the cost of the work
     where it is $250,000 or more including GST, paid before work begins (nsw.gov.au long service levy
     page; verified: false until the owner rules)".
   - *Add to `scenario_inputs`, `proj.budget_stated` note:* "Below $250,000 including GST no levy line or
     activity is expected."

7. **Roof safety (ruling B), two changes.**
   - **7a. shape.yaml `procurement[trade.main_works].why` and `gaps`.**
     - *What:* `access-egress.maintenance-access new` has `needs_design: true`
       (`knowledge/works/actions.yaml`). `CheckGaps` reports a design gap for it, and for every new,
       replace or upgrade item, because no package carries design.
     - `ProposeHeadContract` drops design lines when `BuilderDesigns(route)` is false
       (`internal/procurement/proposed.go` line 69), and traditional gives false. So the design-and-install
       roof safety element can never appear in the head contract proposal.
     - *Change:* add a `gaps` entry (area procurement) saying exactly this. Add to the trade.main_works why:
       "the roof safety design line is not proposed under the traditional route".
   - **7b. shape.yaml `triggers`.**
     - *What:* `consultant.roof_access` comes only through `roof-plant-relocation`, but the usual reason
       for one is an owner who wants an independent layout.
     - *Change:* add a small trigger `independent-height-safety-design`:
       - evidence: "The owner has engaged a height safety consultant to design the roof access and fall
         protection layout and certify the installation." and "The roof access system is to comply with
         the owner's portfolio height safety standard and be audited by the owner's consultant."
       - negation: "The roofing contractor is to design, install and certify the roof safety system."
       - adds: `procurement: [consultant.roof_access]` (built_in no; no template).

8. **shape.yaml `program`, `after` of three documentation rows. Logic in the wrong order.**
   - *What:* "Purlin strengthening and fixing design", "Box gutter, sump and overflow design" and "Asbestos
     removal specification…" follow only the investigation rows, not the owner's scope decision. The scope
     report offers overlay versus removal, and the owner's choice can delete the removal specification.
   - *Change:* add "Owner approves scope and budget; tenant consulted on access and sequencing" to each of
     their `after` lists.

9. **shape.yaml `program` "Install insulation blanket, safety mesh and metal roof sheeting with flashings"
   `after`.**
   - *What:* sheets discharge into the box gutter, so the gutter in each bay is installed first.
   - *Change:* add "Install box gutters, sumps, overflows and downpipe connections" to its `after`.

10. **shape.yaml `program` and `cost_plan`. Dilapidation is missing.**
    - *What:* DOMAIN §3 says the tenant areas, office ceilings and stock below are recorded before work
      starts, but neither key carries it. On a live warehouse it is how damage claims are settled.
    - *Change:* add the activity "Dilapidation record of the tenant's areas, office ceilings and stock below
      the works" (stage construction, after Contract award, before "Protect stock and plant below…",
      `built_in: "no"`).
    - *Add* the cost line "Dilapidation survey before and after the works" (construction, `built_in: "no"`).

11. **shape.yaml `triggers`. A common trigger is missing: overlay without removal.**
    - *What:* DOMAIN §7 offers "Overlay versus full removal", but there is no trigger. NSW owners often
      sheet over bonded AC on a spacer or purlin-extension system to avoid licensed removal over live stock.
      It removes the `site.demolition remove` item, the removal specification, the SafeWork notification,
      removal and clearance rows, and the removal cost line. It keeps the AC on the asbestos register and
      management plan, and adds a spacer system and uplift check.
    - *Change:* add trigger `overlay-over-ac`:
      - evidence: "The new metal roof will be installed over the existing AC sheeting on a spacer system;
        the AC sheeting is left in place and encapsulated." and "Option B: overlay the existing asbestos
        roof rather than remove it."
      - negation: "The AC sheeting will be removed in full before re-sheeting."
      - `removes` the removal work item, activities and cost line named above.
      - adds the activity "Update the asbestos register and management plan for the encapsulated roof" and
        a structural must_include for spacer fixings into the existing purlins.

12. **DOMAIN.md §12, "AS 4000 as the base contract … AS 4902 or a minor-works form". Wrong standard.**
    - *What:* AS 4902 is the design-and-construct general conditions, not a lower-end form.
    - *Change:* "AS 4906 (minor works) or AS 4905 (minor works, design and construct) at the lower end of
      the range."

13. **shape.yaml `triggers[cdc-pathway].evidence[1]`, `.adds.authority[0]`; DOMAIN §5. Possibly the wrong
    code.**
    - *What:* raising the roof 600 mm is an addition, not just an alteration. It likely falls under Part 5A
      of the Codes SEPP, the Commercial and Industrial (New Buildings and Additions) Code, not Part 5, the
      Commercial and Industrial Alterations Code.
    - *Change:* "Codes SEPP Part 5 or Part 5A, depending on whether the work is an alteration or an addition
      (verified: false)". Keep evidence[1] wording neutral: "…under the Codes SEPP commercial and
      industrial codes".

14. **DOMAIN.md §5, Planning. A general requirement is missing.**
    - *What:* exempt development must meet the relevant deemed-to-satisfy provisions of the BCA (Codes SEPP
      general requirements, verified: false). That is why an exempt re-roof still needs AS 1562.1 sheeting
      and AS/NZS 3500.3 drainage.
    - *Change:* add one bullet.

15. **DOMAIN.md §5 hold point 3 ("sheeting fixing inspection … warranty"). Not in the key.**
    - *Change:* either add "Hold point: sheeting fixing inspection against the manufacturer's fixing pattern"
      to `program` (after the sheeting row, before Practical completion, `built_in: "no"`), or drop it from
      the prose.

16. **shape.yaml `work_items[mechanical.ventilation]`. Low priority.**
    - *What:* the id is defensible: whirlybirds and ridge vents are natural ventilation, and keying it is
      what exposes the false Mechanical suggestion. But documents usually list ventilators inside the
      roofing scope ("supply and fit new whirlybirds"), so a reading may file them under
      `envelope.roof-coverings`.
    - *Change:* add to the why: "Scorer: a ventilator filed under envelope.roof-coverings is acceptable."
      Or use an `accept` alternative if the scorer supports one on work items.

Checked and found right:

- **Owner rulings.** R3 (no DBP Act for 7b) and R4 (no HBCF) are applied. R1 (civil only when the roof area
  or hard surfaces change) is respected, because civil is only a trigger. R7 and R8 are not misapplied.
- **Systems.** All 13 top-level systems are in or out, with reasons (checked against `knowledge/systems.yaml`).
- **Prices and durations.** None appear that the repo does not hold.
- **Keyword traps.** The asbestos contamination, flooding, low-pitch membrane, stage, lodge and
  "completed in" traps are realistic and well chosen.

## 4. Built-in claims checked

I checked these by reading the cited files and the `KNOWLEDGE_INDEX.md` selection rules. I did not re-run
`rfp.Resolve`, so the per-position `app` values in the two RFP `app_differs` lists rest on the author's run.
The stage and scale rules behind them check out.

**Procurement**

- ✔ `consultant.hydraulic` yes, `scope_additions[3]`. It fires only if the gutter item is in scope through
  an action or an "included" presence. On remediation, a document that only describes the gutters gives
  `existing`, not scope (KI §1.1).
- ✔ `trade.main_works` partial, `ProposeHeadContract`. Only accepted items count, and design lines are
  dropped under traditional (finding 7a).
- ✔ not expected: Mechanical, yes, `scope_additions[2]`.
- ✔ not expected: Electrical, conditional, `scope_additions[4]`.

**Cost plan**

- ✔ Hazmat fee, partial, `suggest.go`.
- ✔ Structural fee, partial.
- ✔ Hydraulic fee, yes (adopted on prepare, KI §3.2).
- ✔ Asbestos removal, partial, element 01. `site` → "Site works and external works", `trade.civil_works`
  (`elements.yaml`).
- ✔ Purlin provisional sum, partial, 03.
- ✔ Sheeting, insulation and skylights, partial, 04.
- ✔ Box gutters, partial, 09, `trade.hydraulic_plumbing`.
- ✔ Roof ventilators, partial, 08.
- ✔ Roof safety, partial, 13 child "Plant and maintenance access". The label comes from
  `clusters/fire/systems.yaml`, and the system is not in the `items` list.
- ✔ P01, P02, P03 and P07 = `starter_lines.yaml lines[0]`, `[1]`, `[2]`, `[6]`.
- ✔ not expected: P05 `lines[4]`, P06 `lines[5]`, F02 Mechanical.

**Program**

- ✔ `skeleton: missing`, `app_produces: sk.generic`. No skeleton names remediation.
- ✔ Asbestos survey, partial, `fr.existing-conditions` (refurb or extend only).
- ✔ Structural assessment, partial, `investigationRows`. `structure.alterations investigate` has no
  `fr.investigate-*`.
- ✔ `fr.exempt-confirmation`, yes (`cond planning eq exempt`, no work-type gate).
- ✔ `fr.construction-docs-7b`, `fr.tender-docs-7b`, `fr.tender-period-7b`, `fr.tender-evaluation-7b`,
  `fr.site-establishment-7b` and `fr.roof-7b`, partial. Each needs work type new or extend.
- ✔ `fr.contract-award`, `fr.practical-completion`, `fr.handover` and `fr.defects-liability-end`, partial.
  Each needs work type new, extend or refurb.

**RFPs**

- ✔ structural, partial. `existing_assessment` when work type is remediation. `construction_phase` has
  `default: option`, `include_when` complex, and `scale.complex` omits 7b.
- ✔ hydraulic-services, partial. Positions roof; stage rules as described in the index.

**Triggers**

- ✔ purlin-replacement: cost, program and rfps partial. `fr.steel-7b` is gated new or extend; the steel
  child line needs a `trade.structural_steel` package, which the trigger adds.
- ✔ drainage-capacity-upgrade: procurement no, cost partial, program no, rfps partial (`civil.yaml`).
- ✔ solar-pv: procurement no, cost partial (10), rfps partial.
- ✔ sprinklered-building: procurement no, for the trade. **✘ its why** (finding 3).
- ✔ roof-plant-relocation: cost partial (08); procurement and rfps no.
- **✘ cdc-pathway program yes** (should be partial, finding 4).
- ✔ cdc-pathway: cost_plan yes (P05 is always suggested); procurement no; rfps no (`bca-certification`
  blocks on cdc).
- ✔ heritage-listed: procurement yes (`complexity_additions[1]`, no work-type gate), program yes
  (`fr.heritage-referral`, plus the DA fragments once planning is da), cost_plan yes.

**Claims marked `no`, spot-checked (all confirmed)**

- ✔ No package rule names Hazmat, and no baseline covers remediation (`package_defaults.yaml`).
- ✔ Structural is not suggested for remediation (`baselines[0]` is new or extend only).
- ✔ No long service levy starter (`starter_lines.yaml` has seven lines, none for the levy).
- ✔ No fragment names asbestos or a hygienist (grep of `fragments.yaml`: zero hits).
- ✔ No RFP template lists `consultant.hazmat` or `consultant.roof_access` (grep of `knowledge/rfp/`: zero
  hits).
- ✔ The Hazmat aliases are only "Hazmat consultant" and "Hazardous materials consultant"
  (`data/intake/disciplines.json`).
- ✔ No budget split names remediation (`budget_splits.yaml`).
- ✔ `uc.asbestos-found-in-existing-building` attaches to `structure.alterations` and needs new, alter,
  upgrade or repair, so it never fires on remove or investigate.
- ✔ No 7b roof-drainage fragment. The 1a and 2 ones also need new or replace, not upgrade.
- ✔ All 15 cited `uc.`, `cq.`, `fm.`, `if.` and `rule.` ids exist at the cited files.

Sources used for the rulings:

- [NSW legislation, Work Health and Safety Regulation 2025](https://legislation.nsw.gov.au/view/whole/html/inforce/2025-10-03/sl-2025-0440)
- [NSW Government, long service levy](https://www.nsw.gov.au/employment/rights-responsibilities/portable-long-service/bci-long-service-payments-scheme/long-service-levy)
- [SafeWork NSW, asbestos notifications](https://www.safework.nsw.gov.au/notify-safework/asbestos-notifications)

---

## Round 2 (30 October 2026)

### Verdict: **FIX**

All 16 round 1 findings are applied correctly. The validator passes (`ok`, exit 0). The counts check out:
16 triggers, 5 packages, 25 cost lines, 35 activities and 4 RFPs. One problem is left. The new
`overlay-over-ac` trigger removes too little, so the variant built from it would expect a program that
does not hang together and a hazmat brief for removal work that never happens. The four changes in
finding R2-1 fix it. The other two findings are wording.

### Round 1 findings, checked

| # | Fix | Result |
|---|---|---|
| 1 | Cites WHS Regulation 2025 with "2017 numbering … verified: false" in program rows 1, 17, 19 and 25, the friable trigger and DOMAIN §1, §3, §5, §12; r 458, r 78–79 and r 291/299 added | ✔ (hazmat RFP item 6 still lacks the "2017 numbering" label, R2-3) |
| 2 | `consultant.building` base package with `accept: [consultant.building, consultant.structural]`, fee line, RFP (null template, app_differs generic), named as author of the specification row; DOMAIN §4 | ✔ |
| 3 | Sprinkler why corrected; new procurement gap citing `ProjectWorkScope` and `SuggestedPackages` | ✔ |
| 4 | cdc-pathway `program: partial`, why explains the two fragment rows | ✔ |
| 5 | Friable evidence rewritten to the hand-pressure test; weathered-but-bonded negation added | ✔ (label still says "badly weathered", R2-2) |
| 6 | Long service levy figure, source and threshold note in the cost line, the activity and `proj.budget_stated`; CONVENTIONS C4 | ✔ |
| 7a | Head contract why and gap for dropped design lines under traditional | ✔ |
| 7b | `independent-height-safety-design` trigger | ✔ (see below) |
| 8 | The three documentation rows now follow the owner's approval | ✔ |
| 9 | Sheeting follows box gutters | ✔ |
| 10 | Dilapidation activity (after award, before protection) and cost line | ✔ |
| 11 | `overlay-over-ac` trigger | ✘ incomplete (R2-1) |
| 12 | AS 4906 and AS 4905 replace AS 4902 | ✔ |
| 13 | Codes SEPP Part 5 or 5A; evidence wording neutral | ✔ |
| 14 | BCA deemed-to-satisfy general requirement in DOMAIN §5 and §12 | ✔ |
| 15 | Sheeting fixing hold point added and feeding practical completion | ✔ |
| 16 | Scorer note on `mechanical.ventilation` | ✔ |

**`independent-height-safety-design`** is sound.

- The evidence reads like a real owner instruction or portfolio standard, and the negation is the base
  case.
- The adds are visible in the app as a missing package, fee line and RFP.
- All four `built_in: "no"` values are right: there is no package rule or template for
  `consultant.roof_access` (`data/intake/disciplines.json`; no `knowledge/rfp` hit).

### Round 2 findings

1. **shape.yaml `triggers[overlay-over-ac].removes`, `.adds`, `.why`. The overlay variant is incoherent.**
   - *Program rows left in place:*
     - "Purlin repairs and strengthening, then structural inspection": purlins are never exposed under an
       overlay.
     - "Final asbestos clearance, waste disposal records and updated asbestos register": there is no
       removal to clear.
   - *Dangling links:* those rows, and "Install box gutters…", depend on rows the trigger removes ("Remove
     AC roof sheeting…" and the clearance hold point).
   - *Contradiction:* the why says drilling through AC "is still asbestos work, controlled by the
     hygienist's specification", yet the trigger removes the only hygienist specification row.
   - *Hazmat brief:* the hazmat RFP keeps its removal-only items (removal specification, control plan and
     notification review, air monitoring during removal, clearance per area), and the per-visit monitoring
     and clearance cost line stays.
   - *Change:*
     - **removes.program:** add "Purlin repairs and strengthening, then structural inspection" and "Final
       asbestos clearance, waste disposal records and updated asbestos register".
     - **adds.program:**
       - "Hygienist's work method for drilling and fixing through the AC sheeting", after "Asbestos survey
         and sampling of the roof, gutters and high-level surfaces";
       - "Inspect spacer fixings into the purlins before sheeting".
       - State in the why that "Install box gutters…" and the sheeting row then follow "Install spacer
         system over the existing sheeting".
     - **removes.cost_plan:** add "Asbestos air monitoring and clearance inspections during removal (per
       visit)".
     - **adds.cost_plan:** "Hygienist inspections during fixing through AC".
     - **removes.rfps:** for `consultant.hazmat`, remove the removal specification, control plan and
       notification review, air monitoring and per-area clearance items.
     - **adds.rfps:** for `consultant.hazmat`, "Work method and supervision for drilling and fixing through
       AC sheeting; register and management plan update for the encapsulated roof".
     - If `removes` has no `rfps` key in the schema, put the hazmat changes in the why so the scenario
       author applies them.

2. **shape.yaml `triggers[friable-asbestos].label`. Contradicts the fixed evidence.**
   - *What:* the label still reads "AC sheeting badly weathered, damaged or fire-affected, so removal is
     friable", which is the reading finding 5 removed.
   - *Change:* "AC sheeting found friable (fire- or hail-damaged until it crumbles by hand pressure), so
     removal is Class A".

3. **shape.yaml `rfps[consultant.hazmat].must_include[5]`. Missing label.**
   - *Change:* "(specified; mandatory only for Class A, 2017 numbering r 475, verified: false)".

## Round 1 response

Author, 30 October 2026. All 16 findings applied; none disputed. Validator: `ok`.

1. WHS: every citation now reads "Work Health and Safety Regulation 2025 (NSW) (2017 numbering r NNN; 2025 clause verified: false)". r 458, r 78–79 and r 291/r 299 added to DOMAIN §5 and §12. The hazmat air-monitoring item now says "specified; mandatory only for Class A, r 475".
2. `consultant.building` added to the base as specification author, tender manager and superintendent, with `accept: [consultant.building, consultant.structural]`. Added a fee line, an RFP entry (no template, generic `tpl.rfp-capex`) and DOMAIN §1, §4, §8 text.
3. sprinklered-building why rewritten (retain still puts the system in scope). Added a gap.
4. cdc-pathway `program: partial`, with the why updated.
5. friable-asbestos evidence replaced with the friability finding and the Class A statement. Added the "weathered but bonded" negation.
6. Long service levy: 0.25% at $250,000 or more including GST, before work begins, citing the nsw.gov.au source, verified: false until the owner rules. Added a note on `proj.budget_stated`.
7a. Added a gap and a trade.main_works note: the traditional route drops the roof safety design line.
7b. Added trigger `independent-height-safety-design`.
8. The three documentation rows now follow the owner's scope decision.
9. Sheeting now follows the box gutter install.
10. Added a dilapidation activity and a cost line.
11. Added trigger `overlay-over-ac` (with removes), and made it a variant.
12. Contract forms changed to AS 4906 and AS 4905.
13. CDC: Codes SEPP Part 5 or Part 5A; evidence wording made neutral.
14. Added the BCA DTS general requirement for exempt work to §5.
15. Added the program row "Hold point: sheeting fixing inspection against the manufacturer's fixing pattern".
16. Added to the ventilator why: a ventilator filed under roof-coverings is acceptable.

New counts: 16 triggers, 5 packages, 25 cost lines, 35 activities, 4 RFPs, 25 gaps, 8 variants.

## Round 2 response

Author, 30 October 2026. All three applied. Validator: `ok`.

- **R2-1.** `overlay-over-ac` changes:
  - It also removes the purlin repair row, the final clearance row and the per-visit monitoring cost.
  - It adds the hygienist's work method, the spacer fixing inspection, a hygienist inspection cost line and a hazmat must_include.
  - The new dependencies are stated in the why. The removal-only hazmat items are listed there as well, because the schema's `removes.rfps` takes only whole RFPs.
- **R2-2.** The friable label is reworded.
- **R2-3.** Added the "2017 numbering" label to hazmat item 6.
- **Update.** The schema now supports partial RFP removes. `overlay-over-ac` uses `removes.rfps` to drop the four hazmat items that only apply to removal, and the workaround in the why is deleted. Validator: `ok`.
