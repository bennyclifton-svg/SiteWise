# Reprocess missing drawing details

Lane A. Recover missing filing details from a previously OCR-read PDF without
deleting or uploading it again. This improves answer trust and removes a manual
typing detour. Existing values, including deliberate user blanks, are protected.
No new dependency, service, model, threshold changes, supersession decisions or
Project Profile re-reading are included.

## Flow and budgets

- The document's expanded OCR panel lists missing title, number, discipline,
  revision and kind. **Reprocess missing details** queues one durable OCR job.
- The document stays filed and editable. The button disables during submission;
  active work shows Queued / Read lettering / Check details. SSE updates the row,
  with a two-second document-read fallback while recovery is active.
- The existing bounded first-page OCR reader runs once. The parser handles a
  labelled drawing-number prefix split from an adjacent numeric suffix, and
  differently sized title text beside its explicit caption. Geometry is relative
  to glyph size, never a Bankstown coordinate or filename rule.
- Existing saved fields leave the Jev questions. Open questions share at most
  one background fan-out; existing confidence thresholds remain unchanged.
  This follows TypeSafe's [fan-out](https://docs.typesafe.ai/patterns/fan-out)
  and [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
  patterns. Code copies source fragments; Jev does not generate values.
- A transaction locks the document and rechecks each field. Nonempty values,
  user decisions and supersession links cannot be replaced. New OCR values stay
  amber for review. A collision with another document's identity rolls back.
- Completion explicitly distinguishes new details, no new details, and failure.
  An unsuccessful pass leaves the saved document intact and offers an explicit
  retry. No automatic retry loop is added. Worker-crash recovery uses the existing
  lease mechanism.

Queue endpoint budget: p50 <= 50ms, p90 <= 150ms, enforced by
`TestReprocessDetailsEndpointAndLatencyBudget`. Background single-file recovery:
p50 <= 10000ms, p90 <= 20000ms, enforced alongside upload OCR in
`tools/check-ocr.ps1` using real OCR and replayed 350ms provider latency, including
2000ms worker polling allowance. Burst queue waiting is additional. Normal
filing retains 1000/2000ms; the new geometry pass returns immediately for non-OCR
text.

## Verification

Measured CC-06/CC-07 OCR geometry reproduces the missing fields before the fix.
Both reduced tests and complete captured OCR outputs now settle CC-06 / LEVEL 2
and CC-07 / LEVEL 3. Unrelated, distant, cross-page and ambiguous number fragments
remain unjoined. Tests exercise the actual queue, worker and commit with a
concurrent correction, an intentional blank, duplicate clicks, an expired lease,
no-change output, missing-file failure, malformed IDs and org isolation.

Browser coverage exercises queue submission failure, all progress states,
completion without SSE, partial completion, no-change completion and failed
recovery. Desktop and mobile rendering were inspected at 1440px and 390px.

The repository-wide recorded intake accuracy/speed replay was already blocked
by missing recordings/documents before this change. Do not treat passing unit
and OCR gates as a substitute for that unresolved corpus gate.
