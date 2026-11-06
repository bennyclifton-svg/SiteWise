# ST03-da-pathway render review

Reviewer: independent read (brief 04r), 7 November 2026. Variant of ST03-base (base corpus final
and reviewed). Read in full: the ten rewritten documents (d01, d04-d09, d11, d12, d15), the new
d19 and d20. d02, d03, d10, d13, d14, d16-d18 were diffed against base.

## Verdict: PASS

No finding changes a scored value. Findings 1 and 2 are worth taking. Findings 3 to 6 are optional
wording.

## Checks

1. **Lint.** `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST03-da-pathway`
   gives `ok`.
2. **Byte-identical to base.** d02, d03, d10, d13, d14, d16, d17 and d18 are byte-identical to
   ST03-base (`cmp`). d02 and d03 are in `changed_from_base` only because their trap expectations
   change. That holds. The other ten base documents differ only in the pathway wording (diffed line
   by line). Every fee amount, invoice amount and carried phrase is unchanged.
3. **PM read before the key.** This is a Class 5 commercial office tenant fit-out (refurb) of the
   whole of Level 14, 1,200 m2 NLA, in a 24-storey A-grade tower built in 2008. The tower is
   sprinklered, Type A, climate zone 5, with no gas. The pathway is a **DA to North Sydney
   Council**, then a CC, principal certifier and OC from Harbourgate. There is no clause 4.6
   variation and no panel. The route is traditional lump sum under AS 4000. The floor stays
   partially occupied with after-hours works, the quality is high and the budget is $2.4M ex GST,
   with FF&E, AV and IT supplied by the tenant. The land is zoned E2 and is not heritage, bushfire,
   flood, contaminated, acid sulfate or rail land; a heritage item adjoins it. The consultants are
   interior design (lead), building services, BCA, access, certifier, QS and a new town planner.
   - Fees: $130,000 ex, $68,200 inc ($62,000 ex), $10,000, $5,500, $17,000, $16,500 and $12,000 ex.
   - Invoices: $14,000 + $1,400 = $15,400; $6,000 + $600 stated as $6,900 (misadded);
     $3,500 + $350 = $3,850.
   - Start date: 7 September 2026.

   All of this matches `profile.json`, `fees.json` and `invoices.json`.
4. **Pathway wording (hdr.cond.planning).** The acronym "CDC" appears nowhere in the corpus. The
   phrase "complying development" is affirmative only at d02 item 4 ("may be carried out on the
   land"). That is the intended land-eligibility trap. Every other hit is a negation that states a
   DA:
   - d01 lines 61-62 and d19 lines 19-20: "do not meet the complying development standards".
   - d20 lines 47-49: "do not meet all of the standards of the Commercial and Industrial Alterations
     Code ... A development application is therefore made".

   d02 line 30 names "SEPP (Exempt and Complying Development Codes) 2008". That is base text and only
   a policy title. d03's "DA 2004/0617" is the tower's history, the planned trap, and it agrees with
   `da` anyway. Ten sources state DA (d01, d05-d09, d11, d12, d19, d20) against d02's one land
   sentence. `conflicts: []` is safe.
5. **Stray triggers** (`shape.yaml` triggers[].evidence and DOMAIN.md section 10). I searched for:
   stair, asbestos, strip-out, demolition, kitchen, bathroom, toilet, shower, common wall, heritage,
   NABERS, Green Star, coring, slab, compactus, change of use, D&C, staged, decant, exempt,
   performance solution, UPS, cooling, BASIX, public exhibition, community consultation and
   notification. Every hit is planned (the d11 fire stairs, BASIX and performance-solution
   sentences) or negated:
   - d01: no staged construction, no community consultation, no Green Star or NABERS target.
   - d19 and d20: "no change of use".

   d20's "Council will notify the application" matches no community trigger in `flag_readings.yaml`.
   "public exhibition" is absent, as the notes require. "upgrade" appears in d07 and d11, but only as
   "any upgrade the consent requires" and "is not yet known". `fs_upgrade` is a client decision and
   is not read from documents.
6. **Title blocks.** All clean documents are correct, d04-d09, d12, d15, d19 and d20 included. Each
   has `Label: value` lines (or the email header) before the first `##`, one `#` heading equal to
   `Title:`, and number, revision and date matching `document_register.json`. The messy d06 keeps
   its table block. The adversarial d01 and d11 carry their planned traps.

## Findings

1. **manifest.json `project.summary`** (not the corpus) still says "a private certifier issues a
   CDC". That contradicts this scenario's pathway, and any later brief that reads the summary will
   inherit it. Change it to: "...under AS 4000; North Sydney Council determines a development
   application and a private certifier issues the construction certificate; fit-out budget
   $2,400,000 ex GST."
2. **expected/profile.json does not key `flag.architect_lodges`.** Its triggers (`\blodge(?:d|s|ment)?\b`)
   now fire in d01 line 65 ("consent to lodge"), d15 line 35 ("DA lodgement"), d19 lines 24 and 40
   and d20. d19 states plainly that the town planner lodges, so the right answer is `no`. Base had
   the same gap with one hit; the DA version has five. Either key it
   (`{"value": "no", "sources": ["d19"], "read_by_design": true}`) or note in `notes.md` why it is
   left unscored.
3. **d20 line 47** gives no reason why the works fail the Commercial and Industrial Alterations Code.
   d11 says the layout is DTS-compliant with no performance solution, and d02 says the land is
   eligible. A PM reading the set would ask the certifier why. This is optional: name the standard
   in one clause, for example "...do not meet all of the standards of the Commercial and Industrial
   Alterations Code for the alterations to the fire safety measures, so they fall outside...". Mark
   it `verified: false` in the manifest if added. Keep it out of d11 so d11's DTS sentence stands.
4. **d20 line 22.** "A development application will be lodged with council for the fit-out of
   Level 14..." The future tense is odd in the document that goes in with the application. It reads
   as the carry phrase dropped in. This is optional: add a lead sentence above it, "This statement
   accompanies the development application to North Sydney Council.", and keep the carried sentence.
5. **d09 line 26.** "Assessment and determination of the construction certificate after development
   consent is the first item; we cannot determine it until the consent is issued." This reads as
   written for a test, and it is an unwrapped long line. This is optional: "Item 1 is the assessment
   and determination of the construction certificate after development consent; we cannot
   determine it until the consent is issued." It keeps the carried phrase.
6. **Cosmetic line wrapping.** These edited lines run past the documents' ~100-character wrap:
   - d05 line 21
   - d09 line 20
   - d11 line 45
   - d19 line 41

   There is no parse effect. Rewrap if the corpus is touched for 3-5.

## Realism

The rewrites read like the real thing. The certifier's CC-after-consent sequence is right, with
the consent and stamped plans in the CC application. The BCA consultant's point that a DA consent
can bring a fire-safety upgrade of the existing building is apt and is left unknown rather than
asserted. The access consultant reviews the DA set and the CC set separately. d19 is a credible
North Sydney planner's offer, with LEP and DCP review, owner's consent, Portal cost estimate and
council fees excluded. d20 is thin for an SEE, as expected for a purely internal fit-out. It also
leaves out the adjoining heritage item, which the manifest bans as a word because of the d02 trap.
The timeline holds: SEE on 20 May, determination by the end of July, CC by late August, start on
7 September.
