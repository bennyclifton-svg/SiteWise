# Render review: ST12-da-pathway

Reviewer: independent read (brief 04r), 2026-11-06. Variant of ST12-base (see its
RENDER_REVIEW.md). Corpus and manifest not edited.

## Verdict: FIX

No leftover CDC wording, and fees and invoices agree with the key. Two items are records only: d08
was changed but the manifest does not list it, and the notes say d08 never named the pathway. Three
are text edits that protect `flag.community` and `flag.panel` and fix one implausible SEE claim.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST12-da-pathway`, run from
the repository root, gives `ok`. Run from inside the scenario folder, it fails with "base ST12-base:
manifest.json unreadable" and reports every real lot and ABN as undeclared. That failure comes from
the tool resolving paths relative to the current directory. It is not a corpus fault. 18 documents.

## 2. What a PM concludes from the documents alone

- Same project as the base: a new, detached, single-storey Class 1a granny flat. It is 60 m2, with
  2 bedrooms and 1 bathroom and no garage. It sits behind an occupied house on a 600 m2 R2 lot at
  Lot 32 DP 237114, Gymea, in Sutherland Shire. The main house is not altered.
- **Pathway: DA to Sutherland Shire Council**, because the flat is 1.5 m from the rear boundary,
  which exceeds the CDC rear setback standard (d01, d12, d18). A private certifier then issues the
  construction certificate after consent (d10). The designer lodges the DA and writes the SEE
  (d05, d18). There is no clause 4.6 variation and no panel. Council notifies the neighbours and
  there is no exhibition. d02 item 3 says complying development may be carried out on the land.
  That is land eligibility, not the chosen pathway. I would not read it as cdc.
- Site, services, consultants, budget, target start and invoices are as in the base.
- Fees: the designer's fee is $10,400 ex GST (Stage 2 is $5,600 and includes the SEE). The
  certifier's fee is $4,300 ex GST (CC $2,000, PC and inspections $1,800, OC $500, plus $260 per
  visit, plus the CC registration fee at cost). BASIX is $650 + $150. The structural, stormwater
  and geotechnical fees are unchanged.

## 3. Comparison with expected/

`profile.json` (`hdr.cond.planning` da from d01, d05, d12, d18; cl46, panel and community no),
`fees.json` and `invoices.json` agree with the conclusions above. d18 appears in
`document_register.json` as SBD-SEE-01, 14 May 2026, for DA, with a matching title block.

## 4. Leftover CDC wording and stray triggers

A search of the corpus for `CDC|complying|Housing Code|determin|registered certifier|neighbour
notice` finds only these:

- the da-pathway evidence sentence, which is planned (d01 l.30-31, d12 l.27-28, d18 l.16-18);
- the d02 item 3 trap, which is declared;
- "Registered Certifier" as Paul Larkin's title (d10 l.10), which is harmless.

No base sentence still implies a CDC. The structural, stormwater, BASIX, geotechnical and certifier
proposals and Rev B all now name the DA or the CC. "two dwellings" (d12 l.26) and "both dwellings"
(d18 l.69) do not match `multi_building`, which looks for `(two|…) buildings`. The other base
traps are unchanged.

## Findings

1. **manifest.json `changed_from_base` and expected/notes.md: d08 changed but not recorded.**
   - Base d08 l.19 reads "by the certifier for the complying development certificate". The variant
     reads "…for the construction certificate". The edit is correct: the base sentence was a stray
     CDC mention, so the variant had to change it.
   - The manifest must add:
     `{"doc": "d08", "change": "changed", "why": "Purpose names the construction certificate instead of the complying development certificate; amounts unchanged"}`.
   - notes.md "What changed" must add d08 to the list of documents that named the CDC and are now
     rewritten. It must also take d08 out of "d03, d08, d11 and the three invoices never named the
     pathway", which becomes "d03, d11 and the three invoices…".

2. **manifest.json d02 entry: listed as `changed`, but the text is byte-identical to the base.**
   This is legitimate, because only the manifest trap was added and the change vocabulary is only
   `changed|added|removed`. Keep `"change": "changed"`; its `why` already says "Text unchanged".
   notes.md should say this outright: "d02 is copied byte for byte from the base; only its manifest
   entry changes (new trap `cdc-land-eligible`)". That stops a later checker or reviewer from
   reading the copy as a missed rewrite.

3. **manifest.json d02 trap `cdc-land-eligible`, `what`: "Item 4 says complying development…".**
   In the certificate this is item 3 ("3. Complying development"); item 4 is Heritage. Change it to
   "Item 3 says complying development may be carried out on the land."

4. **d12 l.80 and d18 l.118: "Neighbour notification under council's community participation plan;
   no public exhibition."**
   - The `community` reading fires on "public exhibition" and asks whether community consultation
     is required. The base said outright "No public exhibition or community consultation". The new
     wording drops that negation and adds "community participation plan", so Jev could answer yes.
     The key says no.
   - Change both lines to: "Council will notify the neighbours by letter under its community
     participation plan; no public exhibition or community consultation is required."

5. **d18 l.91-93 (Overshadowing): the shadow "falls mostly on the subject site and on the rear
   neighbour's shed roof".**
   - d18 l.25 puts the lot on the southern side of the street, so the rear boundary is south. A
     flat 1.5 m from a south boundary throws its midwinter shadow over the fence. That is exactly
     why a DA planner would address it.
   - Change to: "The shadow diagrams show that in midwinter the flat's shadow falls on the strip of
     the rear neighbour's yard behind the fence, mostly on their shed roof. The neighbours' living
     areas and main private open space keep their existing sunlight."
   - No figure is needed.

6. **d18 l.119: "no local planning panel referral is expected".**
   - "planning panel" fires the `panel` reading. The hedge "is expected" is weaker than d12 l.79
     ("will be determined by council staff under delegation, not the local planning panel").
   - Change to: "Determination by council staff under delegation; the application is not referred
     to the local planning panel."

7. **Reflow after edits (realism and format).**
   - d10 l.40-42 breaks a bullet mid-sentence ("Your / designer can apply / for it online"). Reflow
     it as one wrapped bullet.
   - d04 l.10, d05 l.73 and d06 l.18 now run well past the ~104-column wrap the rest of each file
     uses. Rewrap them.
   - Cosmetic only.

## 5. Title blocks

- d18 (clean) has `Title`, `Ref` and `Date` before the first `##`. Its `#` heading equals the
  Title. The number, date and title match the register.
- d10's Title and heading both changed to "construction certificate", matching the register.
- d12 "Issued for: development application" and d14 Rev B "Amended for DA lodgement" agree with the
  register's "for DA".
- The tiers of the others are as in the base.

## 6. Realism

The file reads like a real DA-path granny flat. The SEE is credible in structure (site, proposal,
permissibility, compliance table, s4.15 matters, notification, conclusion), apart from finding 5.
Two passages read as written for a test, and both are forced by carries:

- The evidence sentence "sits closer to the rear boundary than the complying development standards
  allow; the flat exceeds the CDC rear setback standard" appears verbatim in d01, d12 and d18. A
  real owner, designer and SEE would each phrase it differently.
- The SEE compliance table gives the rear setback as "Assessed on merit" without stating the DCP
  control. A real SEE would quote the control. This is deliberate (the setback figure is
  `verified: false`), so leave it.
