# ST17-cdc-pathway: independent render review

Reviewer: senior PM read, 2026-11-06. Brief: `testing/synthetic/briefs/04r-render-review.md`.
Base corpus ST17-base is final and reviewed. This review covers what changed (d02, d06, d07, d08, d11,
d13 rewritten; d12 and d16 removed) and whether the CDC pathway holds up.

## Verdict: **FIX**

The corpus is clean and consistent with the key: lint passes, every profile, fee and invoice value is
read where the key says, and no document still implies a DA. The problem is the premise. d02 names the
**Codes SEPP Fire Safety Code** as the instrument. NSW Planning's own description of that code does not
cover cladding replacement. The shape's certifier fee stages and several program rows also still assume
a construction certificate (CC) issued after a DA.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST17-cdc-pathway` →
`ok` (exit 0).

## 2. What a PM concludes from the documents alone

- **Building:** residential flat building, Class 2, 8 storeys (rise in storeys 8), 56 apartments, built
  in 2011. Concrete frame, not sprinklered, Type A construction. Stays fully occupied. Strata scheme
  (SP 87654) run through a committee and general meetings.
- **Work type:** remediation under a council fire safety order (FSO-2026-0031). PE-core ACP and the EPS
  render come off; a ventilated rainscreen of solid aluminium panels goes on. Windows and balustrades are
  retained. Compliance date 17 Dec 2027; target start 1 Mar 2027.
- **Planning pathway:** a CDC that the certifier determines (d02, d13). d03 says complying development
  may be carried out on the land, which agrees. No heritage, flood, contamination or rail constraint.
  Zone R4, City of Ryde.
- **Procurement:** traditional route, fully documented, lump sum, AS 4000-1997 with special conditions.
- **Consultants:** PM, architect, fire engineer, facade engineer, structural engineer and QS have
  proposals. A BCA consultant and the certifier come later and have no proposal. No town planner is
  sought, and no SEE exists.
- **Site:** on the street boundary, with scaffold and hoarding over the council footpath. The basement
  exists but nothing is excavated. No transfer structures. No work near the boundaries.
- **Budget:** $6,850,000 ex GST (d13).
- **Fees:**

  | Consultant | Fee | GST basis |
  |---|---|---|
  | PM | $182,000 | ex GST |
  | Architect | $234,000 | ex GST |
  | Fire engineer | $48,000, plus $6,600 lab testing | ex GST |
  | Facade engineer | $76,000 | ex GST |
  | Structural engineer | $33,000, plus $5,800 pull-out testing | ex GST |
  | QS | $49,500 inc GST, which is $45,000 ex GST | inc GST |

- **Invoices:**
  - BLA-INV-2207: $38,000 + $3,800 = $41,800.
  - K-26-0412: $14,500 + $1,450 should be $15,950, but the invoice states $16,850, an arithmetic error.

## 3. Comparison with expected/

`profile.json`, `fees.json` and `invoices.json` all agree with the read above, with no gaps. The
d12/d16 removal leaves no orphan: d13 still carries the flags for clause 4.6, panel and community
consultation. `document_register.json` lists 16 documents with no d12 or d16. The changed labels match
fees.json: "CDC drawings and photomontages", "Fire engineering support letter for the CDC" and
"CDC cost estimate report".

## 4. Stray triggers and DA residue

- **DA residue:** a corpus grep for `development application`, `\bDA\b`, `consent`, `SEE`,
  `statement of environmental`, `planner`, `Ridgeway`, `exhibit`, `4.55` and `construction certificate`
  finds nothing in the works context. The remaining hits are all legitimate:
  - d03's fixed council wording ("Permitted with consent", "Duty planner").
  - d01's citation of the Act.
- **Diff against base:** confirms only the planned lines changed.
- **Ridgeway Planning:** its ABN and the Hartigan email appear in no document.
- **`order-compliance-no-consent`:** no "without a development application" wording. No hit.
- **Other variants:** no hits.
- **DOMAIN.md §10 traps:** same as the base, and planned.

## 5. Title blocks

The changed documents keep their base title blocks unchanged and match the register: d02 and d13 are
adversarial with clean label lines, d06 adversarial, d07 and d08 clean, d11 messy with a table block.
No issue.

## 6. Findings

1. **d02 line 43: the instrument named is wrong (most serious).**
   - **Phrase:** "a complying development certificate under the Codes SEPP Fire Safety Code".
   - **What NSW Planning says:** its Fire Safety Code page lists three complying-development
     categories:
     - fire sprinklers in residential aged care;
     - alterations to hydraulic fire safety systems;
     - fire alarm communication link works.

     Cladding replacement is not among them
     (https://www.planning.nsw.gov.au/the-planning-system/development-assessment/planning-approval-pathways/development-types/fire-safety).
   - **Exempt route also closed:** the 2018 Codes SEPP cladding amendments bar exempt re-cladding on
     residential apartment buildings
     (https://www.planning.nsw.gov.au/policy-and-legislation/buildings/combustible-cladding/exempt-development-for-cladding).
   - **Other codes:** none plainly fits an 8-storey Class 2 recladding. The Housing Alterations Code is
     for dwelling houses. The Industrial and Business Alterations Code is for commercial and industrial
     buildings.
   - **Effect:** a NSW PM reading d02 would question the CDC at once. So the variant is not credible as
     written. Its own shape already marks the CDC `verified: false`, and the evidence now points against
     it.
   - **Change (corpus):**
     - Delete " under the Codes SEPP Fire Safety Code" so the sentence reads exactly as the trigger
       evidence: "The works will be approved by a complying development certificate; the certifier will
       determine the CDC for the cladding replacement."
     - Update manifest d02 `carries` `hdr.cond.planning.says` to match.
   - **Change (owner decision):**
     - In `expected/notes.md`, the shape's `cdc-pathway.why` and DOMAIN.md §5, record that NSW Planning
       lists only three Fire Safety Code categories, and cladding is not one.
     - The owner then decides between two options:
       - (a) keep the variant as a test of reading a pathway a document states, while saying in
         `notes.md` that it is probably not a real NSW route for this building; or
       - (b) retire it, since `order-compliance-no-consent` and the DA base already cover the real
         options.

     Do not mark the pathway verified.

2. **Shape: the certifier's fee stages still assume a CC after a DA (serious, affects expected/rfps
   via the shape).**
   - **What the shape says:** `rfps` for `consultant.certifier` lists in `must_include` "Construction
     certificate assessment, including regulated designs lodged". Its `stages_include` has
     `b2_construction_approval` as a CC stage.
   - **What the trigger does:** `cdc-pathway` adds "CDC assessment and determination" but does not
     remove the CC line.
   - **Why it is wrong:** a CDC is a combined planning and building approval, so no CC follows it.
   - **Change:** in `cdc-pathway`, add `rfps.removes_must_include` (or the equivalent edit) for the CC
     line, so the certifier's CDC appointment reads:
     - CDC assessment and determination, including regulated designs lodged;
     - principal certifier inspections;
     - final fire safety certificate and OC for the altered part;
     - updated fire safety schedule.

     `b2_construction_approval` then stands for the CDC.
   - **BCA consultant stages:** these fit if `a2_planning` ("planning-stage compliance report") is read
     as the BCA report lodged with the CDC application. Add that note to the shape. It also means the BCA
     consultant has to be appointed **before** the CDC drawings are lodged. d02's "once the design is
     settled" fits that timing only if the CDC goes in after design development (see 3).

3. **Shape program and cost plan: rows still name the DA or the CC (serious for expected/program).**
   `cdc-pathway.removes.program` drops only "Lodge DA", "DA notification", "DA assessment and
   determination" and "Construction certificate". It leaves these rows, which reference them:

   | Row | Problem | Change |
   |---|---|---|
   | "General meeting resolves option, design fees and first levy, and owners' consent to lodge the DA" | Names the DA | Read "… owners' consent to lodge the CDC" |
   | "DA documentation: drawings, photomontages, SEE, BCA and fire letters, QS cost report" | Names the DA and the SEE | Replace with "CDC documentation: drawings, BCA report, fire engineer's DTS confirmation, structural fixing details, QS cost report". Place it after "Design development", not after schematic design |
   | "Satisfy pre-CC conditions of consent" | Has no meaning on a CDC | Remove |
   | "Pay long service levy" | Runs after "DA assessment and determination" | The levy is paid before the CDC issues. Hang it before "Certifier determination" |
   | "Report progress to council…" | Runs after "Lodge DA" | Run it after "Prepare and lodge the CDC" |
   | "Contract award" | Runs after "Construction certificate" | Run it after "Certifier determination" |
   | "Hold point: panel samples…" | Its why cites "the DA colours condition" | Change to "the CDC colours condition" |

   In the cost plan, "Construction certificate and certifier fees" stays suggested. Note in the shape
   that on this path it means CDC and principal certifier fees.

4. **d07 and d08: the stage order is still DA-shaped (moderate, realism).**
   - **d07:** stage 3 "CDC drawings and photomontages" ($32,000) comes before stage 4 "Design
     development" ($44,000).
   - **d08:** item 3 "Fire engineering support letter for the CDC" comes before item 4 "Design review
     and confirmation of deemed-to-satisfy compliance".
   - **Why it matters:** a CDC must show NCC compliance at the point of determination, so the CDC set
     follows design development and the fire engineer's DTS confirmation. It does not precede them. As
     written it reads like the base DA order with "DA" swapped for "CDC".
   - **Change:**
     - In d07, swap rows 3 and 4: 3 Design development $44,000, then 4 CDC drawings and photomontages
       $32,000.
     - In d08, swap items 3 and 4: 3 Design review and confirmation of DTS compliance $9,000, then
       4 Fire engineering support letter for the CDC $3,000.
     - Reorder the matching `prices` entries in `expected/fees.json`.
     - Amounts and totals are unchanged. d19 and d20 bill stage 1 and item 1 only, so they are
       unaffected.

5. **d07 line 36: "photomontages" have no CDC audience (minor).**
   - **Problem:** under a CDC nobody assesses visual impact publicly. The photomontages serve the owners'
     general meeting.
   - **Change:**
     - Label: "CDC drawings, with photomontages for the general meeting".
     - Match `fees.json` and the manifest d07 content.
     - Optional; take it with 4 if that is done.

6. **d13 line 81: the clause 4.6 negation reads as written for a test (minor).**
   - **Problem:** "Complying development does not change the height or floor space ratio and needs no
     clause 4.6 variation" is a general statement, not a fact about this project.
   - **Change:**
     - New wording: "The works do not change the height or floor space ratio, and as complying
       development need no clause 4.6 variation."
     - Update manifest d13 `carries` `flag.cl46.says` to match.

7. **d13 line 82: the planning-panel negation is moot (minor).**
   - **Problem:** "It is not regionally significant development and will not be determined by a planning
     panel" is redundant once the certifier determines.
   - **Change:**
     - New wording: "As complying development determined by the certifier, it does not go to a planning
       panel."
     - Update manifest `flag.panel.says` to match.

8. **d02 line 43 and d13 line 74: unwrapped lines (minor, a splice tell).**
   - **Problem:** each changed sentence runs as one long line, while the rest of both documents wraps at
     about 105 columns.
   - **Change:** rewrap both to match the surrounding prose.

9. **Neighbour notice (note, no corpus change).**
   - **What the documents say:** d02 line 45 has "the notice to neighbours"; d13 line 83 has "the
     neighbour notice the certifier gives".
   - **Status:** whether a pre-determination or post-determination neighbour notice applies to this
     CDC under the EP&A Regulation 2021 depends on which code applies (see 1).
   - **Change:** add "(verified: false)" to `notes.md` against `flag.community`.

10. **Manifest housekeeping (minor).**
    - **Problem:** `invented.firms` still lists "Ridgeway Planning Pty Ltd". `invented.abns` and
      `emails` still carry its ABN 14 270 817 793 and ben.hartigan@example.com. No document uses them.
    - **Change:** remove them from the ST17-cdc-pathway manifest.

## Checked and found sound

- **d06:** "the certifier's assessment of the CDC application" is right. Stage 2 "Design and approvals
  management" is neutral. The exclusions ("Council, certifier and authority fees") fit.
- **d11:** a registered QS's CDC cost estimate report is credible. The certifier needs a cost of works
  for fees, long service levy and contributions, and d03 says a s7.11 plan applies.
- **d08:** item 6, "Verification for the occupation certificate", still fits. Under a CDC the principal
  certifier issues the OC.
- **d03:** the `cdc-eligible-land` trap now agrees with the pathway, as the manifest intends.
- **Realism:** apart from findings 4 to 8, the changed documents read like the real thing.
