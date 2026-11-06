# ST02-traditional-procurement: render review

Reviewer: independent read (brief 04r), 2026-11-06. Scope: the five rewritten documents (d01, d04, d05,
d06, d07) in full, plus a sweep of the thirteen copied base documents for anything that still implies
design and construct or novation. The ST02-base corpus is final and already reviewed.

## Verdict: PASS

Three minor findings, none of which changes a keyed value. Findings 1 and 2 are realism. Finding 3 is an
optional consistency tidy-up.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST02-traditional-procurement`
returned `ok`.

## 2. What a PM concludes from the documents (before opening `expected/`)

- New six-storey Class 2 residential flat building of 64 apartments over two basement levels (Class 7a),
  on three amalgamated R4 lots. Three 1962 dwellings are demolished, the site is vacant, and there is one
  owner.
- DA to the City of Ryde, determined by the local planning panel. The town planner lodges it. The
  building is a regulated design under the DBP Act.
- **Procurement: traditional.** The developer's consultants prepare full construction documentation,
  then a lump-sum tender under AS 4000-1997 with the developer's amendments. Nobody is novated, and the
  consultants stay on the developer's side through construction (d01 L66-72; d05 L21-25; d06 L19-20;
  d07 L19-20). The construction certificate is issued before award (d04 L13).
- Budget is $27.6M ex GST. Target DA lodgement is 31 Aug 2026 and target builder start is 1 Nov 2027
  (d04, an email).
- Fees: architect $1,100,000 ex GST plus a $260,000 option. Structural $309,000 ex GST plus a $60,000
  option and $420 per visit. Civil $116,600 inc GST ($106,000 ex). The geotechnical and planner fees
  are as in the base.

The architect's fee is about 4.0% of the construction budget (4.9% with the option). Structural is about
1.1%. Both are plausible for full documentation on a Class 2 job of this size.

## 3. Comparison with `expected/`

- `profile.json`: `hdr.cond.procurement_route` = traditional (d01, d05) and `fact.contract_form` = as4000
  (d01, d05) both match what the documents say. Nothing in the corpus pulls towards design_construct,
  eci, as4902 or as2124.
- **Contract-form wording against the app's reading.** `knowledge/profile/project_facts.yaml`
  `contract_form` uses the trigger `\bas\s?4000`. Triggers are compiled as case-insensitive RE2
  (`internal/knowledge/profile.go` `compile`). That trigger matches all of these:
  - d01 L66 "lump-sum tender under AS 4000"
  - d01 L71 "AS 4000-1997 General conditions of contract, lump sum"
  - d05 L24 and L75

  The option text "AS 4000 general conditions of contract" is close to d01 L71 word for word. No other
  contract form is named anywhere in the corpus: there are no hits for 4902, 2124, ABIC, D&C, "design and
  construct" or ECI. **The variant's `as4000` key is reachable.**
- `fees.json`: every stage figure and total in d05, d06 and d07 matches. The civil GST-inclusive figures
  divide exactly by 1.1 to the keyed ex-GST cents (14,000, 24,000, 36,000, 20,000, 12,000; total
  106,000). The option and rate lines match.
- `invoices.json`: d16 (Arden Vale, Stage 1 $185,000, agreement AVA-FP-2611) is still consistent with the
  rewritten d05. d17 and d18 do not change.
- `procurement.json`: route is traditional and every package's novation is `no`. This agrees with the
  documents.
- **Fee stages against a traditional route.** The architect's stages run concept, DA, DA assessment,
  tender documents (a full construction set, addenda, tender assessment input), then for-construction
  documents (tender clarifications, CC comments, DBP declarations). Construction-phase services are an
  option. The structural stages follow the same pattern. Civil's "detailed design for tender" and "final
  design for construction" fit too. No D&C tender stage or post-novation stage survives in any of the
  three proposals. The trigger's `must_include` and `must_not_include` RFP texts line up with these
  stage names.

## 4. Stray triggers

Every `triggers[].evidence` and `negation` phrase from `types/T02-apartments/shape.yaml` was searched
across the corpus.

- **Evidence hits:** only `traditional-procurement`, both phrases in d01. This is planned.
- **Negation hits:** groundwater (d10), cl4.6 (d09), flood and acid sulfate (d02), trees, gas and facade
  (d11), and single building (d01). All of these are base negations.

A sweep of all thirteen copied documents found nothing that still implies D&C or novation. The search
covered novat, D&C, design and construct, 4902, contractor, builder, tenderer, "engaged by", "by others"
and shop drawing. None of these appears outside d01 and d04-d07.

d11 L75-76 ("the construction certificate is issued in two parts: Stage 1 CC ... Stage 2 CC") is common
under a traditional route too. It does not conflict with d04's "construction certificate issued before
award", because both parts can precede award. No change is needed.

The DOMAIN.md §10 "novation" negation ("no novation", d01 L67, d05 L25) is planned (notes.md Traps).

## 5. Title blocks

- d01 (clean): the Title, Date, Client and Site label lines come before the first `##`, and the one `#`
  heading is identical to Title. The register title and date match.
- d04 (clean): an email header thread with the reply on top, as the manifest asks. The register title
  "RE: Program for the Calloway Street apartments" and the date 3 March 2026 match.
- d05 (adversarial): the label lines (Title, Proposal no AVA-FP-2611, Date, Client) and the `#` heading
  match the register. The planned trap "The town planner lodges the DA" is present at L39.
- d06 (clean): Title, Quote no KBS-Q-2604, Date and the `#` heading match the register.
- d07 (messy): a Markdown table title block (ref NCE-P-0388, 13/03/2026, attention Priya Raman), then
  `# Fee proposal: civil engineering`, as planned.

## 6. Realism

d01, d04, d06 and d07 read like the real thing. d05 reads well apart from finding 1.

## Findings

1. **d05, L24-25: the sentence reads as written for a test.** "Stage 4: tender documents for a
   lump-sum tender under AS 4000; Stage 5: for-construction documents. We remain engaged by the
   developer through construction; no novation." It is a bare carry sentence restating two stage names
   in the introduction, before the scope has been described. It works, but no architect writes an
   introduction this way.

   **Change:** replace L21-25 with the paragraph below. It keeps the carry substring word for word, so
   the manifest is unaffected.

   > This proposal covers the design from concept through full construction documentation for a
   > traditional lump-sum procurement. Our documentation stages are Stage 4: tender documents for a
   > lump-sum tender under AS 4000; Stage 5: for-construction documents. We remain engaged by the
   > developer through construction; no novation. Design support during construction is priced as an
   > option.

2. **d05, L73-75: two different services share one name.** The option is headed "Design support during
   construction" but is described as "(contract administration under AS 4000)". Under AS 4000, contract
   administration is the Superintendent's function. That covers assessing variations, which this scope
   includes, as well as claims, EOTs and certificates. Design support covers RFIs, shop drawings and
   site visits. A PM would ask which one the architect is offering, and whether it acts as Superintendent.

   **Change:** at L75, change "(contract administration under AS 4000)" to "(including contract
   administration as the Superintendent's representative under AS 4000)". Make the same change to the
   d05 `content` quote in `manifest.json`. The fees.json label "Option: design support during
   construction" and the amount stay as they are.

3. **d07, L52-53: the civil scope stops at the construction certificate.** It excludes "site inspections
   during construction" and offers no construction-phase option. Under a traditional route, the
   developer's civil engineer usually inspects the OSD and stormwater and certifies them for the
   occupation certificate. The architect (d05) and the structural engineer (d06) both price construction
   services, so civil is the odd one out. This does not affect scoring.

   **Optional change** (adds no figure, so no key change): at L52-53, change "and site inspections during
   construction" to "and site inspections and OSD certification during construction, which we can price
   on request".
