# ST17-base render review

Reviewer: independent read (brief 04r), 6 November 2026.
Corpus: 20 documents in `corpus/`. Lint: `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST17-base` → `ok`.

## Verdict: **FIX** (minor)

The corpus is sound. Every figure the key relies on is printed once and plainly. The fee tables add up and every
stated total matches the key. The only stray hit is one undeclared word. The six findings are realism fixes and one
manifest declaration. None of them changes a key value.

## What a PM concludes from the documents alone (before opening `expected/`)

- **Building:** Tideway Apartments, 27 Tideway Crescent, Meadowbank NSW 2114, City of Ryde, SP 87654. An 8-storey
  residential flat building of 56 apartments, constructed in 2011. Concrete frame, NCC Class 2, rise in storeys 8,
  Type A, not sprinklered. Wholly residential over a basement car park. Built to the street boundary on two
  frontages. Fully occupied throughout.
- **Work type:** remediation. Full removal of the PE-core ACP (balcony spandrels and blades), the EPS render (L7–8
  setbacks) and the combustible sarking, replaced to DTS with a ventilated solid-aluminium rainscreen (Option 2).
  The driver is a council fire safety order (FSO-2026-0031, 19 Jan 2026), to be complied with by 17 Dec 2027.
- **Planning:** DA (council's advice in d02; the SEE in d16). The land is CDC-eligible (d03), but the works are not
  a CDC. Not heritage, not flood, no contamination record, not bush fire prone, R4. Acid sulfate soils class 5.
- **Procurement:** traditional. Full documentation, lump sum tender to remedial builders, AS 4000-1997 with special
  conditions. The PM acts as superintendent.
- **Consultants:** PM, architect, fire engineer, facade engineer, structural engineer, QS and town planner. A BCA
  consultant and a principal certifier come later. No geotech, civil, hydraulic, mechanical, environmental or
  hazmat consultant (the 2020 survey found no asbestos).
- **Site conditions:** a scaffold and an overhead hoarding over the council footpath on both frontages (s138
  approval needed). Scaffold base plates on the podium over the basement. No excavation, no transfer structures,
  no authority assets, no OSD change, no staging, one building.
- **Budget:** $6,850,000 ex GST (d13), adopted as the construction budget. Funded by special levies, with no
  government assistance.
- **Fees:**
  - Corbel $182,000 ex (5 stages).
  - Brightline $234,000 ex (6 stages).
  - Kestrel $48,000 ex, plus about $6,600 testing, with a performance solution excluded.
  - Mullion $76,000 ex.
  - Tallow $33,000 ex, plus about $5,800 pull-out testing.
  - Quoin $49,500 **inc** GST, which is $45,000 ex.
  - Ridgeway $16,500 ex.
- **Invoices:**
  - BLA-INV-2207: $38,000 + $3,800 = $41,800. Correct.
  - K-26-0412: $14,500 + $1,450, printed as $16,850. It should be $15,950, so it is an arithmetic error.

## Comparison with `expected/`

- **profile.json:** every keyed value matches my reading, and each keyed source carries it in the stated words.
  `proj.target_construction_start` is only in the d05 email. `proj.regulated_design`, the type of construction and
  `proj.lot` are keyed as missed by design. No differences.
- **fees.json:** all seven stated totals and stage sums agree (checked row by row). The d11 normalisation is right:
  $49,500 / 1.1 = $45,000 and $198 / 1.1 = $180. The key's labels are trimmed in places (for example d08 item 4
  drops "of the replacement wall"). The harness does not score labels.
- **invoices.json:** both invoices match, and the `arithmetic` flag on d20 is planned.

## Extra checks requested

1. **d07 numbers its stages 1–6, not "Stage 1"–"Stage 6".** The app still reads every row and the total:
   - **The total reads.** The total row `| | **Total fee: $234,000 excluding GST** | **$234,000** |` has the same
     shape as ST09's `| | Total $17,000 excluding GST | $17,000 |` (Karrak, items numbered 1–4). That proposal
     scored `stated_total_cents` correct after the F14 fix
     (`runs/20261104-after-fixes/ST09-base/field_score.json`). The ST10 structural proposal uses the same layout.
   - **Every stage row is harvested.** `harvest.go` takes each row's money cell. Its label is the first filled cell,
     the bare digit. The whole row text, including the description, goes to Jev as context.
   - **What the change costs.** The shown label is weaker than d06's "Stage N" label. `CleanFeeLabel`'s `stageID`
     rule does not prefix a bare digit.
   - **The harness does not score per-stage rows.** `score_fees` scores only `stated_total_cents` and `firm_name`
     (`harness/score.py:255`).
   - **One benefit.** The bare digits keep `\bstages? [1-9a-c]\b` from firing on every row. Only the planned
     "Stage 5 includes" sentence matches.
   - **Verdict:** acceptable. The same layout is already used in d08, d09, d10 and d12.
2. **d13 gives no cost per option, so its only figure is $6,850,000 ex GST.**
   - **Realistic.** It is about $122,000 per lot. With roughly 2,500–3,500 m² of spandrel, blade and setback
     cladding, it works out at about $2,000–2,700/m² all-in. That covers removal, a new subframe, mineral wool, the
     rainscreen, an 8-storey full perimeter scaffold for 36 weeks, footpath hoarding on two frontages,
     preliminaries and contingencies on an occupied building. That is the upper-middle of the Sydney range for full
     removal to DTS, and plausible.
   - **Consistent with the key.** It matches `profile.json` `proj.budget_stated` = 685000000 and `rfps.json` =
     685000000. It is stated "excluding GST". No other dollar figure in the corpus could be read as a budget.
   - **One weakness: realism, not the key.** The report says it compares options "on … cost" but shows no cost
     comparison. See finding 2.
3. **d11 says the owners corporation is not registered for GST.**
   - **Harmless for `gst_basis`.** Three explicit cues say inc: the column `Fee (inc GST)`, `Total fee $49,500
     including GST` and `$198 per hour including GST`. All three match `incGSTRe`. "Not registered for GST" matches
     neither `exGSTRe` nor `incGSTRe`. The sentence's own "includes GST" does not match `incGSTRe`, so it adds no
     count either way. Jev's GST question ("Answer from the wording printed with the fee schedule") has one clear
     answer. The key (`inc`, `gst_inc_normalised`, $45,000) stands.
   - **Probably not accurate.** An owners corporation must register once its GST turnover passes the threshold
     ($75,000, or $150,000 if it is treated as non-profit). A 56-lot, 8-storey scheme with lifts and a basement
     would normally collect well over that in levies each year. The special levies here (about $0.7m for fees, then
     about $7.5m for construction) put it far past. The statement is unlikely, though not impossible. It also implies
     that every other ex-GST fee understates the levy by 10%, which no document notices. I have not checked this
     against ATO guidance. See finding 6.

## Findings

1. **`Combustible cladding audit report KFE-A-2519.md`, line 65 (## Risk):** "The fire engineer's assessment
   identifies the external wall cladding as a facade fire performance issue requiring rectification."
   - **Problem:** the audit is dated 15 September 2025 and is written by the fire engineer. It cites, in the third
     person, an assessment that only exists from 7 April 2026 (d14, which is "this assessment follows our 2025
     audit"). A real author would not write this.
   - **Change:** "Our assessment identifies the external wall cladding as a facade fire performance issue requiring
     rectification."
   - **Effect:** the `fire_specialist_systems` trigger still matches, because "facade" falls within 100 characters
     of "fire" in the same sentence. The planned source for `flag.fire_specialist_systems` holds.

2. **`Facade remediation options report BLA-R-014-01.md`, lines 17–19 and the options table (lines 53–58):** the
   purpose promises a comparison "on compliance, appearance, cost and disruption", and Revision 1 "adds the
   quantity surveyor's order of cost". The table still has no cost row. The only cost words are "Option 2 costs
   more than the other two".
   - **Problem:** a report put to a general meeting would show the cost comparison.
   - **Change:** add a row after `Time on site`: `| Relative cost | Lowest | Highest | Middle |`.
   - **Effect:** this keeps $6,850,000 as the only dollar figure, so the key is unaffected.

3. **`Facade remediation options report BLA-R-014-01.md`, ## Approvals and ## Delivery (lines 71–106):** this is
   the one document that reads as written for a test. It runs through a string of one-line denials:
   - no Country input;
   - no Green Star or NABERS;
   - "there are no multiple buildings";
   - no sewer or water main;
   - no OSD;
   - HBCF;
   - "No claim is being pursued".

   No architect volunteers most of these in an options report. They are the shape's negations, and the key relies
   on d13 for them, so keep every sentence. Give them the frame a real report uses: insert the line "The following
   matters were checked with the consultant team and do not apply to this project:" before
   "- The site is not a heritage item…", and move the four paragraphs from "The site has a single residential flat
   building…" to "…No claim is being pursued." out of ## Delivery into that list as bullets. Keep each sentence
   word for word.

4. **`Slab edge investigation and anchor pull-out results TCE-R-3381-01.md`, lines 59–61:** "no excavation, basement
   or ground works are involved. No excavation is involved, and nothing is done near…"
   - **Problem:** excavation is denied twice in a row. That reads as padding.
   - **Change:** "The works are confined to the external walls; no basement or ground works are involved. No
     excavation is involved, and nothing is done near the adjoining boundaries or neighbouring structures."
   - **Effect:** the `basement` trigger still matches the first sentence, and `excavation_adjoining` still matches
     the second. The sources for `flag.basement` and `flag.excavation_adjoining` hold.

5. **`Planning certificate s10.7 - SP 87654.md`, line 62:** "The land is mapped as acid sulfate soils class 5".
   - **Problem:** this is a stray "class N" not declared in the manifest. The app asks one Jev question per class
     ("does the passage state that this project's building or part is Class 5?"). Its `not_stated` option covers
     "something else, such as … a material grade", so the risk is low. But it is a new, unplanned class hit; no
     earlier corpus has it. Line 37–38 ("shop top housing; neighbourhood shops", in the permitted uses) is a weaker
     unplanned brush with `mixed-use-podium`.
   - **Change (manifest and notes only; the wording is genuine s10.7 content and should stay):** add to d03 `traps`
     `{"id": "acid-sulfate-class-not-ncc", "what": "'acid sulfate soils class 5' names a class that is not an NCC
     building class", "expect": "det.ncc_class stays [\"2\"]"}`, and add the same line under ## Traps in
     `expected/notes.md`.

6. **`Quoin fee submission - Tideway Crescent.md`, lines 21–22:** "because the owners corporation is not registered
   for GST and pays the GST-inclusive amount".
   - **Problem:** this is unlikely for a 56-lot scheme (see extra check 3). It also casts doubt on the GST basis of
     every other proposal.
   - **Change:** "Every figure in this letter includes GST, so that it can be compared directly with the special
     levy."
   - **Effect:** the three inc-GST cues and the key are unchanged.

## Stray triggers (shape `triggers[].evidence`, DOMAIN.md §10)

The corpus was searched for every T17 trigger's evidence phrases and for the §10 traps (kitchen, bathroom, bedroom,
brick, masonry, stairs, common wall, strip-out, demolish, BASIX, exempt, flood, like-for-like, sprinkler, exhaust,
staged/stage N, levy instalments or loans, heritage, contamination, swing stage, neighbour, boundary, novation,
D&C, CDC, asbestos, retail/shop, Class/Type, AFSS, FR core, HBCF, proceedings, Project Remediate, windows,
balustrades, waterproofing, excavation).

- **Planned:** every hit is planned (manifest traps or shape negations), except d03 "class 5" and "shop top
  housing" (finding 5).
- **Two near misses that do not fire:**
  - d08 "Interim fire safety measures and staged strategy" does not match the `staged` triggers (`staged
    (?:construction|delivery|…)`).
  - d02/d05 "first … second levy" is the fee levy followed by the construction levy, not staged funding of the
    works.

## Title blocks

- **Clean documents** (d01, d07, d08, d09, d10, d12, d15, d17, d18, d19):
  - each has `Label: value` lines before its first `##`;
  - each has one `#` heading identical to `Title:`;
  - number, revision, title and date match `document_register.json`.
- **d05** is an email with a From/To/Subject/Date header, as `04-render.md` allows.
- **Messy documents:** d03, d11 and d20 carry their identity in Markdown tables. d11 and d20 also use
  indented fixed-width schedules.
- **Adversarial documents:** d02, d04, d06, d13, d14 and d16 carry their planned traps.

## Realism

- **Reads like the real thing:**
  - d01 has the notice, representations, order, interim measures, appeal and offence structure.
  - d16 includes the s138 hoarding note.
  - d05, the fee proposals and the invoices are convincing.
- **Weakest:** d13 (findings 2 and 3), d04's anachronism (finding 1) and d15's repeated sentence (finding 4).
- **Manifest note (no corpus change):** the manifest calls the d20 error "digits transposed". $15,950 → $16,850 is
  not a transposition. The `arithmetic` flag is unaffected.
