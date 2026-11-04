# Brief 04r: Independent read of one scenario's documents

To dispatch, replace `{SCENARIO}` and `{TYPE_DIR}` (for example
`ST01-base` and `T01-new-house`).

---

You are an independent reviewer: a senior Australian PM. Another agent
wrote the documents in `testing/synthetic/scenarios/{SCENARIO}/corpus/`
from that scenario's `manifest.json`, using
`testing/synthetic/briefs/04-render.md`. They will be dropped into a
document app and its output scored against
`testing/synthetic/scenarios/{SCENARIO}/expected/`. Repository:
`D:\AI Projects\sitewise`.

Do not edit the corpus. Write
`testing/synthetic/scenarios/{SCENARIO}/RENDER_REVIEW.md`.

## Check

1. Run `python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/{SCENARIO}`.
   Record the result.
2. **Read every document first, as a PM would, before opening
   `expected/`.** Write down what you would conclude about the project:
   class, work type, planning pathway, consultants needed, site
   conditions, budget, fees, invoices.
3. Then compare your conclusions with `expected/profile.json`, `fees.json`
   and `invoices.json`. Every difference is a finding: either a document
   says something it should not (a stray condition, an ambiguous figure)
   or it fails to say something the key relies on.
4. **Stray triggers.** Search the corpus for the conditions in
   `testing/synthetic/types/{TYPE_DIR}/shape.yaml` `triggers[].evidence`
   and the keyword traps in `DOMAIN.md` §10. Any hit the manifest does not
   plan is a finding.
5. **Title blocks.** Each `clean` document has readable `Label: value`
   lines before its first `##`, one `#` heading matching `Title:`, and
   values matching `expected/document_register.json`. Each `messy` and
   `adversarial` document is what its tier says.
6. **Realism.** Does each document read like the real thing? Name any that
   read as written for a test.

## Write RENDER_REVIEW.md

Verdict **PASS** or **FIX**, then numbered findings: the file, the line or
phrase, what is wrong, the exact change.

## Reply

**100 words or fewer**: the verdict, the number of findings, and the two
most serious.
