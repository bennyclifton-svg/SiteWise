# Brief 04: Write the documents for one scenario

To dispatch, replace `{SCENARIO}` (for example `ST01-base`).

---

You are a senior Australian construction PM who has read thousands of real
project documents: owner briefs, planning certificates, survey and
geotechnical reports, consultant fee proposals, drawings, emails and tax
invoices. You are writing the documents of one fictional project so that
they read like the real thing. They will be dropped into a document app as
a test, so they must state exactly what the plan says, and nothing that
gives the test away.

Repository: `D:\AI Projects\sitewise`. Your scenario is
`testing/synthetic/scenarios/{SCENARIO}/`. Write only its `corpus/` folder.

## Read first

1. `testing/synthetic/scenarios/{SCENARIO}/manifest.json`: your plan. One
   file per document, named exactly as its `filename`.
2. `testing/synthetic/scenarios/{SCENARIO}/expected/`: the answers. Your
   documents must support them. Do not copy key names or ids into the
   documents.
3. `testing/synthetic/DISCOVERY.md` §2 to §4 and §7 to §8: how the app reads
   Markdown, fee proposals and invoices.
4. If the manifest has a `base`, that scenario's `corpus/`. Copy unchanged
   documents byte for byte, and rewrite only those listed in
   `changed_from_base`.

## How to write each document

- **Every `carries[].says` appears word for word** in its document. Line
  breaks and `**bold**` around it are fine. Otherwise use your own words.
- **Say nothing else that matters.** Do not mention trees, heritage, flood,
  bushfire, pools, basements, easements, gas, slope or any other site or
  building condition unless the manifest has the document say it. A stray
  mention fires a trigger and makes the answers wrong. The same goes for
  the keyword traps in the type's `DOMAIN.md` §10.
- **Title blocks follow the tier** (DISCOVERY.md §4.3):
  - `clean`: `Label: value` lines **before the first `##` heading**, using
    labels the app reads: `Title:`, `Document number:` or `Drawing number:`
    or `Ref:` or `Quote no:`, `Revision:`, `Date:`. Dates as `3 February
    2026` or `2026-02-03`. Numbers with no spaces. One `# Heading` only, and
    make it the same text as `Title:` (two different titles give none).
  - `messy`: the title block as a Markdown table or a fixed-width block,
    the way a PDF-to-text conversion lays it out. The app reads none of it;
    that is the test.
  - `adversarial`: a clean title block plus the declared trap, written
    exactly as `traps[].what` describes.
- **Emails** start with header lines: `From:`, `To:`, `Subject:`, `Date:`
  (`To` and `Subject` both needed). Quoted replies below repeat the header
  block.
- **Fee proposals** have "Fee proposal" in the heading, the firm's name,
  ABN, contact, email, phone and address, a stage-by-stage fee table, the
  total, and the GST basis (ex or inc) stated as the manifest says.
- **Invoices** say "Tax Invoice", with the supplier, ABN, invoice number,
  date, the project, a line for the stage claimed, subtotal ex GST, GST
  and total. Do not use the word "quotation" or "proposal" on an invoice.
- **Drawings** are Markdown stand-ins for a drawing sheet: a title block,
  a short list of what the sheet shows, general notes, and a revision
  table. Keep them short.
- **Realistic length.** An email 50 to 250 words; a brief, report or
  proposal 300 to 1,200 words; a drawing 100 to 300 words.
- **Invented only.** Use the manifest's invented people, firms, ABNs,
  emails and lot/DP numbers and nothing else. Every email is
  `@example.com`. Do not write any other 11-digit number.
- **No test words.** Never write "test", "synthetic", "expected", "answer",
  "scenario", "trap" or an app id such as `flag.basement`. (Real phrases
  such as "test pit" or "flood test" are fine where a real document would
  use them.)
- Write UTF-8 with LF line endings. With Python on Windows, open files with
  `newline=''`.

## Check

```
python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/{SCENARIO}
```

Fix every error. Do not change the lint.

## Reply

**100 words or fewer:** the lint result, the document count, and any
manifest instruction you could not follow and why.
