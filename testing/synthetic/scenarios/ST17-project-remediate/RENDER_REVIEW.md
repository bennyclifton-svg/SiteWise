# Render review: ST17-project-remediate

Reviewer: independent read (brief 04r), 6 November 2026. Corpus: 19 documents. The base corpus
(ST17-base) is final and reviewed; 15 documents are byte-identical copies. This review concentrates
on the four rewritten documents (d02, d05, d11, d13), the removal of d06, and whether anything still
reads as a self-funded, self-managed job.

## Verdict: FIX

Small text fixes only. Nothing changes a scored value: the profile, fees and invoices keys all hold.
The fixes are about whether the Project Remediate story hangs together for a PM reading it.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST17-project-remediate`
→ `ok`.

## 2. What a PM concludes from the documents alone

- **Project:** Tideway Apartments, 27 Tideway Crescent, Meadowbank NSW 2114, City of Ryde. An
  8-storey Class 2 residential flat building of 56 apartments, built in 2011. Concrete frame, no
  sprinklers, rise in storeys 8, Type A construction. Strata scheme SP 87654. Harbourline is the
  strata managing agent. The building stays fully occupied.
- **Work type:** remediation of combustible external wall cladding on an existing building, under
  council's fire safety order FSO-2026-0031 (compliance date 17 December 2027). PE-core ACP, EPS
  render and foil sarking all come off. The replacement is a ventilated rainscreen of solid aluminium
  panels that meets the deemed-to-satisfy provisions. Windows and balustrades are retained.
- **Planning pathway:** a DA for the change of external materials and appearance, Zone R4,
  determined under delegation. The site is not heritage and needs no cl 4.6 variation. A separate
  hoarding approval is needed.
- **Site conditions:** the building sits on the street boundary on both frontages. The scaffold and
  overhead hoarding stand over the council footpath. Base plates sit on the podium over the existing
  basement, with no excavation. The land is not flood-prone or contaminated, and has ASS class 5.
- **Funding and management:** a Project Remediate interest-free loan, repaid through levies over ten
  years. The program's project manager is already appointed and leads the team. Consultants come
  from the program panels.
- **Consultants:** architect, fire engineer, facade engineer, structural engineer, QS and town
  planner. A BCA consultant and principal certifier come later. There is **no PM package**.
- **Contract:** traditional, fully documented, one lump sum AS 4000-1997 contract with special
  conditions, not staged.
- **Budget:** $6,850,000 excluding GST (d13).
- **Fees:**
  - Brightline: $234,000 ex GST.
  - Kestrel: $48,000 ex GST, plus a testing estimate of $6,600.
  - Mullion: $76,000 ex GST.
  - Tallow: $33,000 ex GST, plus a pull-out testing estimate of $5,800.
  - Quoin: $49,500 inc GST, which is $45,000 ex GST.
  - Ridgeway: $16,500 ex GST.
- **Invoices:**
  - BLA-INV-2207: $38,000 + $3,800 = $41,800.
  - K-26-0412: $14,500 + $1,450, printed as $16,850. The correct total is $15,950, so this is an
    arithmetic error.

## 3. Comparison with expected/

- `profile.json` matches the reading above. The only difference from the base is that d06 drops out
  of the sources for `proj.client` and `fact.contract_form`. That is correct.
- `fees.json`: six submissions. Totals, GST bases, stage prices and rates match the documents. Quoin
  is normalised to ex GST ($45,000; hourly rate $180). There is no PM submission, which is correct.
- `invoices.json`: both invoices match. The Kestrel arithmetic flag is planned
  (d20 `total-arithmetic`).
- `document_register.json`: d02's title "... design team" matches the document. d06 is absent.

There is no difference between the key and the documents.

## 4. Stray triggers

I checked every `triggers[].evidence` and `negation` phrase in `T17-cladding-remediation/shape.yaml`
against the corpus:

- The only evidence hits are the two planned `project-remediate` phrases in d02.
- The negation phrase "funding the works through special levies without government assistance" is
  gone from every document.
- No `staged-funding` evidence appears. There is no "strata loan" and no "levy instalments".
- The DOMAIN.md §10 traps that are present are the planned base traps: remediation, basement, BASIX,
  bedroom, EPS/PE, Type A and audit.
- "Corbel", "special levy", "superintendent" and "without government assistance" do not appear
  anywhere in the corpus.

## 5. Title blocks

- d02 and d13: label lines before the first `##`, and one `#` heading identical to `Title:`.
- d05: a clean email header.
- d11: a messy table block, as its tier says.

All values match the register.

## 6. Self-funding and self-management residue

None found in the rewritten documents.

The copied fee proposals (d07–d10, d12) and the invoices (d19, d20) are addressed to the owners
corporation c/- Harbourline. That is still right: the owners corporation remains the client and the
borrower, even though the program's PM processes payments. Brightline's line "We will lead the design
team" is a lead-consultant role under a PM and does not conflict. No change needed.

## Findings

1. **d05 contradicts d02 on whether the loan is already accepted.**
   - d02 (3 February) says the owners corporation "has accepted an interest-free loan" and the PM
     "is already appointed".
   - d13 says the owners corporation "accepted [the loan] when the building was found eligible".
   - d05 (12 February) says the first general meeting "must resolve to enter the Project Remediate
     remediation and loan agreements".

   A PM reads this as two different stories. The d02 sentence is the trigger's fixed evidence, so
   the fix goes in d05. In `RE Timing of the cladding works.md`, line 11–12, replace "The first must
   resolve to enter the Project Remediate remediation and loan agreements, as both are well above the
   committee's spending limit." with: "The first must pass the resolution to sign the Project
   Remediate remediation and loan agreements, which formalise the loan the committee accepted; only
   a general meeting can commit the owners corporation to a loan of that size." This also fixes
   finding 3.

2. **The enrolment timeline is not explained.**
   - shape.yaml's `why` says registration for Project Remediate closed years ago, so a 2026 scenario
     needs an owners corporation that was already enrolled.
   - Yet every document dates the problem from the owners corporation's own 2025 audit and council's
     November 2025 notice.
   - Read alone, the documents show a building found in 2025 and in a closed program by February
     2026.

   Fix: in `Owners brief - Tideway Apartments cladding remediation.md`, ## Funding, after the
   evidence sentence (line 37), add: "The owners corporation registered the building with the
   program in 2022; the program confirmed its eligibility once the 2025 audit and council's order
   were in hand." Leave the evidence sentence unchanged. Program dates and eligibility stay
   unverified, as shape.yaml says.

3. **d05 keeps leftover levy reasoning.** "as both are well above the committee's spending limit"
   was written for the special levy and design fees. A loan agreement is not spending against a
   committee limit. Fixed by the replacement text in finding 1.

4. **d13 funding sentence reads as if written to carry a fact.** In `Facade remediation options
   report BLA-R-014-01.md`, lines 70–71, "as the owners corporation accepted when the building was
   found eligible for Project Remediate" is stilted. It also leaves out the ten-year term that d02
   states. Replace the paragraph with: "The works will be funded through the owners corporation's
   Project Remediate interest-free loan, repaid through levies over ten years."

5. **The manifest still names the removed d06 author.** `manifest.json` → `invented` still lists
   "Corbel Project Services Pty Ltd", `daniel.okafor@example.com` and ABN `80 848 775 257`. None of
   them appears in the corpus. Remove all three so that nothing downstream expects them.

6. **Line wrap (cosmetic).** The rewritten lines run past the corpus's wrap width:
   - d02, lines 37 and 41
   - d11, line 18
   - d13, line 97

   Re-wrap them to match the surrounding text. This has no effect on reading.

## Realism

The rewritten documents read like the real thing:

- The strata manager's brief, the chair's email thread, the QS letter with its GST-inclusive
  comparison against the loan, and the architect's options report for a general meeting all ring
  true.
- The QS's exclusion of "advice on how loan repayments are apportioned between lots" is a good
  touch.
- Removing the PM fee proposal leaves no gap: d02 says why there is no PM proposal, and d13 says who
  administers the contract.

Once findings 1 and 2 are fixed, the Project Remediate details are accurate and consistent across
d02, d05, d11 and d13: NSW program, interest-free loan repaid through levies, and a
program-appointed PM coordinating panel consultants.
