# ST13-base render review

Reviewer: independent read of `corpus/` (18 documents) against `manifest.json` and `expected/`.

## Verdict: FIX

All fixes are small. No scored value is at risk: every carried value is stated and read where the key says, and every
trap fires as the manifest plans it. The fixes are one key-versus-document mismatch (firm phone and address), one
inconsistency between documents (where the boreholes and test pits were dug), and five places that read as written for
a test or that a PM would query.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST13-base` gives `ok  testing/synthetic/scenarios/ST13-base` (exit 0).

## 2. A PM's reading, before opening `expected/`

- **Project.** Halcyon Eastwood Pty Ltd (sole owner, single decision maker) is building a new six-storey mixed-use
  building at 40-44 Halloran Street, Eastwood NSW 2122 (Lots 4, 5 and 6 in DP 18273, City of Ryde, E1 Local Centre).
  It has three ground-floor shell shops (600 m2 GLA; Rev B splits tenancy 2 into 2A and 2B) under 42 apartments on
  five levels, with two basement levels holding 55 spaces. GFA is about 5,500 m2. A 1965 dwelling and two 1965 shops
  are demolished. The site is vacant during the works, access is constrained, and it is one building in one stage.
- **Class.** Class 2, 6 and 7a. Rise in storeys 6, effective height 18.9 m, Type A construction, sprinklered, climate
  zone 5. The DBP Act applies. BASIX covers the apartments and Section J the retail. Fire safety is DTS throughout,
  with no vulnerable occupants and no special hazards.
- **Planning.** A DA to Ryde under the Housing SEPP Chapter 4 and the ADG, determined by the Local Planning Panel.
  No clause 4.6 request. The planner lodges, not the architect. The UDRP saw the scheme at pre-lodgement. The site is
  not heritage, flood, bushfire, acid sulfate, coastal or rail affected. It is a local road with no TfNSW referral,
  no heavy vehicles, no SIDRA and no green travel plan. Traffic noise affects the site. Street trees are not
  affected. Strata subdivision comes later.
- **Procurement.** D&C, AS 4902, lump sum, with the design team novated.
- **Consultants.** Architect, structural, civil, geotechnical, town planner, surveyor, environmental (PSI) and BCA
  appear. Acoustic (the SEE cites "the acoustic report") and services engineers (d06) are implied. The roof communal
  space suggests a landscape architect. A hazmat survey is expected before demolition.
- **Site.** 1,802.4 m2 by survey. The ground falls 2.3 m to the street. Fill over clay over shale. No groundwater,
  only seepage, so a drained basement. Excavation adjoins No. 38, with no underpinning. Anchors need the neighbours'
  consent. OSD and WSUD are required, with no existing OSD. Sewer, water and natural gas are reticulated. No Sydney
  Water assets in the site. No contamination (d02, d05, d13).
- **Budget.** $24,500,000 ex GST, construction only.
- **Fees (ex GST).** Architect $424,000 (option $264,000, $260/h). Structural $103,000 (option $92,000, $240/h).
  Civil $52,800 inc GST, which is $48,000 ex (option $27,500 inc, $25,000 ex). Geotechnical $26,000 (option $3,900;
  waste classification excluded). Planner $54,500 ($250/h).
- **Invoices.** CRA-INV-1027: $68,000 + $6,800 = $74,800. Ashgrove 2214: $23,500 + $2,350 = $25,850.
  BRG-INV-0388: $20,200 + $2,020, but the total is printed as $22,420; the correct sum is $22,220.

## 3. Comparison with `expected/`

Profile: every value in `profile.json` matches my reading, with the right sources. The fees amounts, GST bases,
options, rates and the excluded element match `fees.json`. All three invoices match `invoices.json`, including the
`arithmetic` flag on d18. The one difference is finding 1.

## 4. Stray triggers

I ran every `flag_readings.yaml` trigger over the corpus, sentence by sentence, and searched for the shape's
`triggers[].evidence` and the DOMAIN.md §10 phrases. Every hit is planned and denied: the contamination denials (d02,
d05), the sewer main in the lane, street trees, no existing OSD, the planner lodging, "pre-lodgement", no
groundwater, no underpinning, the strata bond and strata subdivision, public exhibition, the design review panel,
classified road, clause 4.6, "multiple buildings" and "staged". None of the shape's trigger evidence appears.

Unplanned hits, none scored (both need no change):

- `flag.arborist_offsite_access` fires on d01 "access is constrained ... neighbours", d07 "Anchors under neighbouring
  land will need the owners' consent" and d11 "Anchors extending under neighbouring land require the owners'
  consent". The key does not hold the flag, so it is not scored. The question asks for confirmed permission, so the
  answer should be no.
- `flag.arborist_tree_impacts` fires on d09 and d11 "excavation to about 7 m below the street", because the pattern
  `tree` has no word boundary and matches inside "s**tree**t". The key is no, from d13. This is an app regex gap, not
  a corpus fault. Record it for the shape's gap list.

## 5. Title blocks

All clean label-line documents (d01, d05, d06, d07, d09, d12, d14, d15, d16) and the adversarial label-line documents
(d03, d10, d11, d13, d18) have readable `Label: value` lines before the first `##`, one `#` heading identical to
`Title:`, and numbers, revisions and dates that match `document_register.json`. d04 has the From, To, Subject and
Date header, with the reply on top: the title is "RE: Programme and DA timing" and the date 3 March. The messy
documents d02, d08 and d17 use table title blocks with a `#` heading, as their tier says. Each adversarial document
carries its planned trap.

## 6. Contamination and the PSI (extra check)

- **Supports the carried value.** d05's Conclusions state the carried sentence verbatim: "The PSI found a low risk
  of contamination and no further investigation is required." d13 repeats it under SEPP (Resilience and Hazards)
  Chapter 4, and d02 denies it on council's records. All three agree with `det.potentially_contaminated_land: false`.
- **No contradiction in the base.** "Service station", "dry cleaner" and "underground tanks" appear only in d05's
  denial. No other document names a former use beyond shops, a newsagent, a bakery, a hairdresser and a dwelling.
  Spoil classification in d05, d09 and d11 is routine, and d11 expects natural shale to be VENM. That is consistent.
- **Does it read like a real PSI?** Mostly. It has a site history from titles, aerials and council records, an
  inspection, limited sampling against NEPM criteria, and a suitability conclusion. Without the asbestos text, a
  contamination reviewer would query two things. Fill containing brick fragments, under buildings from about 1965,
  is normally screened for asbestos. A PSI for a demolition site normally points to a hazardous building materials
  survey. The removal also left a double space at line 49 and a short line at line 58. Finding 4 restores both points
  safely.

## Findings

1. **`expected/fees.json` (all five submissions): firm phone and address disagree with the documents.** Every fee
   proposal, and the invoices and reports from the same firm, use one consistent set of contact details. The key holds
   a different set: Calder Rowe "02 9874 3310, Level 2, 18 Rowe Street" against d06/d16 "02 9874 3360, Level 4, 120
   Rowe Street", and likewise for Ironbark, Larkspur, Basalt Ridge and Ashgrove. The manifest invents neither. The
   harness scores only `stated_total_cents` and `firm_name` today, but the app stores phone and address (DISCOVERY.md
   §4), so the key is wrong for the day those fields are scored. The documents are consistent across ten files, so
   fix the key. In `expected/fees.json` set:
   - d06 `phone` "02 9874 3360", `address` "Level 4, 120 Rowe Street, Eastwood NSW 2122"
   - d07 `phone` "02 9411 7085", `address` "Suite 12, 3 Victoria Avenue, Chatswood NSW 2067"
   - d08 `phone` "02 9477 2251", `address` "Level 1, 27 Hunter Street, Hornsby NSW 2077"
   - d09 `phone` "02 9868 5512", `address` "Unit 8, 44 Carlingford Road, Epping NSW 2121"
   - d10 `phone` "02 9635 1840", `address` "Suite 5, 60 Church Street, Parramatta NSW 2150"

2. **d09 and d11: the boreholes are put in "shop yards" that do not exist.** d03 says "The rear of Nos. 42 and 44 is
   a sealed car park; the rear of No. 40 is a grassed yard", so the shops have no yards. A PM reading d03, d09 and
   d11 together would notice.
   - d09 line 26-27: "two through the rear car park and two in the shop yards after access is arranged with the
     tenants" -> "two through the rear car park of Nos. 42 and 44 and two in the rear yard of No. 40 after access is
     arranged with the occupants".
   - d11 line 26-27: "BH1 and BH2 through the rear car park, BH3 and BH4 in the shop yards" -> "BH1 and BH2 through
     the rear car park of Nos. 42 and 44, BH3 and BH4 in the rear yard of No. 40".
   - In the manifest's d09 `content`, change "two in the shop yards" to match.

3. **d05 line 37-38: "no underground tanks are present" overstates what a PSI can say and reads as written for the
   test.** Change "Council holds no record of underground tanks, and none were seen on site; no underground tanks are
   present." to "Council holds no record of underground storage tanks, and no fill points, vent pipes or bowsers were
   seen on site." The denial and the planned trap word stay.

4. **d05: asbestos screening and the hazmat pointer are missing (realism, extra check).** Each new sentence keeps
   "contaminat", "clause 4.6", "groundwater", "water table", "built in", "street" and "excavat" out of the same
   sentence, so no reader changes.
   - Line 49-50: "Soil samples were analysed for heavy metals, hydrocarbons, BTEX, PAHs and OCPs.  All results are
     below ..." -> "Soil samples were analysed for heavy metals, hydrocarbons, BTEX, PAHs and OCPs, and fill samples
     were screened for asbestos. All results are below ..." (this also removes the double space). Then add the
     sentence "No asbestos was detected in the fill."
   - Conclusions: after "The land is suitable for the proposed residential use.", reflow line 58-59 into one
     paragraph and add: "The existing buildings date from the 1960s. A hazardous building materials survey should
     be carried out before demolition; it is outside the scope of this PSI."

5. **d12 line 33: the order of the sentences implies that Type A follows from the effective height.** A certifier
   would tie it to the rise in storeys. Change "The effective height is less than 25 m. Type A construction is
   required." to "The effective height is less than 25 m. With a rise in storeys of 6, Table C2D2 requires Type A
   construction." The carried phrase is unchanged.

6. **d13 line 46-47: the SEPP name is printed twice, as a bold lead and again in the sentence.** Change to:
   "**SEPP (Resilience and Hazards) 2021.** Chapter 4 applies because the proposal introduces residential use. The
   PSI found a low risk of contamination and the land is suitable for the proposed residential use." In the
   manifest's d13 `det.potentially_contaminated_land` `says`, use the second sentence. `contaminat` still matches.

7. **d07 line 11: "RPEQ" is the Queensland register.** A Sydney structural engineer working on a DBP Act project
   would cite NSW registration. Change "(RPEQ, NER Structural)" to "(CPEng, NER Structural; registered professional
   engineer under the Design and Building Practitioners Act 2020)".

8. **d16 against d04: the architect claims 100% of "Concept design and urban design review" on 30 April.** d04 books
   the urban design review meeting for May, so the claim comes before the meeting. A PM would not pass that claim. In
   d04 line 10, change "with council for May" to "with council for April", and change the manifest's d04 `content`
   to match. No key changes, and d13's "at pre-lodgement" still holds.

9. **d01 lines 19-23 (minor realism): the demolition is stated twice in adjacent paragraphs.** The first paragraph
   ends "each existing building on the site is to be demolished"; the next ends "each existing building is to be
   demolished". Change line 22-23 to "The site contains an existing building on each lot (two single-storey shops
   and a dwelling). The existing shops were built in 1965." `\bexisting building\b` still matches, and
   `det.existing_building` stays carried. In the manifest's d01 `det.existing_building` `says`, remove the clause
   "; each existing building is to be demolished". The planned phrasings "This is one building; we are not planning
   multiple buildings" and "one-level retail podium" are stilted but deliberate (notes.md). Leave them.

Notes, no change: d18 is dated 15 May and leaves the $5,800 report unclaimed three weeks after the report issued.
That is plausible, because some firms bill the report later. d06's "before the application is determined" and d13's
"at pre-lodgement" agree.
