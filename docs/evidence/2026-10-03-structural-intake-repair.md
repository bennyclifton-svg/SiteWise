# Structural drawing intake repair

The reported local project contained 20 new structural sheets with blank
drawing numbers and incorrect or unresolved disciplines. PDFium and an
independent PDF reader exposed mostly the architect's contact panel and the
copyright line. Most printed lettering, including the discipline and drawing
number, was vector graphics. The engineer's embedded font also lacked a usable
Unicode mapping. No SHX text annotations were present in the inspected sheet.

The complete `job_sheet_title-(issue)` filename convention separately encodes
the job and sheet numbers. Previously both became competing document-number
candidates. The parser now excludes the job token only for this complete
convention, still covering it when extracting the title. Other filenames stay
ambiguous, and conflicting printed number evidence is retained.

Intake-76 asks Jev to distinguish an issuing consultant from a title-block
project contact and to consider descriptive filenames when page text is
incomplete. It explains common formwork and reinforcement abbreviations, while
retaining explicit architectural authorship and an abstention option. This
follows TypeSafe's [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [single fan-out](https://docs.typesafe.ai/patterns/fan-out) patterns. No new
runtime dependency, provider, request or relaxed acceptance threshold was added.
The retained discipline threshold is provisional amber-only, not a new calibration.

## Verification and repair

- The filename regression failed before the change and passes afterward.
  Tests cover multiple sheet prefixes, incomplete conventions and printed conflicts.
- All Go tests pass with the dedicated test database, including HTTP integration,
  user-correction protection, organisation isolation and existing latency tests.
- Live Jev checks classified the descriptive structural filenames correctly;
  generic names still lack reliable evidence. The stricter final question left
  the sampled generic-name results below admission instead of applying them.
  Synthetic controls retain Architectural for an architect's own concrete detail
  and Structural for an engineer's detail with an architect contact.
- Every affected drawing number was checked against its rendered title block.
  With the owner's explicit approval, all 20 disciplines were saved as user
  corrections. Refiling with the fixed parser saved all 20 numbers as rule
  decisions. Existing user corrections are protected; original PDFs are unchanged.
- The local port-8080 app was rebuilt and restarted. Database and authenticated
  application reads verify 20 filed documents, 20 correct numbers and 20
  Structural user corrections. Private before/after evidence stays in `.tools/`.
- Candidate-harvesting benchmark: p50/p90 3.746/3.938 ms (budget 5/10 ms).
  Rule decisions: 0.363/0.628 ms (budget 1/1 ms).
- The unchanged PDF extraction on the 20 affected files measured p50/p90
  101.634/241.821 ms against 80/250 ms: the p50 budget is exceeded. This is an
  outstanding extraction performance issue, not a passing full-path speed claim.

This repair does not add OCR or recover graphics-only text. Future files with
similarly incomplete text and generic names can still need a discipline
correction. The owner's corrections are not evidence of automatic Jev accuracy.
Old exact-request recordings cannot establish intake-76 accuracy; broad
holdout validation and threshold calibration remain outstanding.
The full replay benchmark also fails closed because 220 requests have no
matching recorded document; it requires refreshed recordings before it can
provide a full-path latency verdict for this version.
