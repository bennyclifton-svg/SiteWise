# Brief 03r: Independent review of one type's scenarios

To dispatch, replace `{TYPE_ID}` and `{SLUG}`.

---

You are an independent reviewer: a senior Australian PM who also reads JSON
and Go. Another agent wrote the scenarios in
`testing/synthetic/scenarios/S{TYPE_ID}-*/` from
`testing/synthetic/briefs/03-scenarios.md`. Each scenario's `manifest.json`
says what each fictional document will state, and `expected/` holds the
answers the app should give. Your job is to find keys that are wrong,
because a wrong key makes every later test result wrong. Don't polish.
Repository: `D:\AI Projects\sitewise`.

Do not edit the scenarios. Write
`testing/synthetic/scenarios/S{TYPE_ID}-base/REVIEW.md` covering all of
this type's scenarios.

## Check, in this order

1. **The checker.** Run
   `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/S{TYPE_ID}-*`.
   Record the result.
2. **Faithful to the shape.** The base is
   `testing/synthetic/types/{TYPE_ID}-{SLUG}/shape.yaml`'s base job with
   every trigger denied or absent. Look for a document that would
   accidentally fire a trigger or a keyword trap (`DOMAIN.md` §10), which
   would make the base answers wrong.
3. **Register keys.** For each document, apply DISCOVERY.md §4.2 and §4.3
   to what the manifest says the title block holds. Would the app's rules
   settle the number, revision, title and date that the key expects? A
   Markdown `# Heading` that differs from `Title:` gives no title; a table
   title block gives no identity values; a filename is never read. Where
   the key expects a value the app cannot get by design, the manifest or
   notes must say so.
4. **Profile keys.** Is each value supported by the documents' `carries`?
   Is `read_by_design` right (drawings, emails and photos are not read)?
   Is each flag stated in words the app's profile reading would catch
   (`knowledge/profile/flag_readings.yaml` and `knowledge/determinants.yaml`
   triggers)?
5. **Fees and invoices.** Money ex GST in whole cents, GST-inc stated
   figures divided by 1.1, totals that add up, flags the app really raises
   (`internal/proposals/read.go`, `internal/invoices/ledger.go`).
6. **Variants.** Each changes exactly its trigger. The trigger's
   `evidence` is phrased the way the shape gives it, and nothing else
   moves.
7. **Realism.** Would a PM recognise this set of documents and dates as a
   real house job in NSW? Are the fees plausible for the scale?

## Write REVIEW.md

- Verdict: **PASS**, **FIX** (specific changes listed) or **REDO**.
- Numbered findings: scenario and file, what is wrong, the evidence, the
  exact change.

## Reply

**100 words or fewer**: the verdict, the number of findings, and the two
most serious.
