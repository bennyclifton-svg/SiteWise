# Extended corpus intake evaluation - 2 October 2026

## Manual-testing handoff — intake-75, 3 October 2026

Corpus expansion stopped at the owner's request before the separate project
profiler merge. Intake-75 is running on port8080. Fresh final uploads pass
**35/35 vertical-transport fields** (2.969 seconds) and **70/70 Petersham
mechanical fields** (4.281 seconds), including original integrity, single-page
downloads and duplicate checks. Existing projects are retained; fresh uploads
in a new test project exercise the current classifier.

The latest complete broad upload result remains **36/99** from round71 below.
Four additional folder batches bring the private manifest to 53 batches / 106
files; these additions have targeted results, not a new combined accuracy claim.
Printed matrix, unit-plan and standalone schedule headings now challenge wrong
filename titles. Their unresolved titles remain blank pending calibration.
Image-only stamped DA drawings remain unsupported. One broad schedule keyword
attempt regressed a structural title; it was narrowed before deployment.
All 497 previously correct rule-derived fields then passed again using fresh
538-PDF extraction traces. Extraction p90 was 381ms against a 250ms budget:
that performance limitation remains open.

Full Go tests75, frontend build and three Python harness tests pass. The browser
drag/drop, correction, reconnection and tenant-isolation test also passes. No confidence cutoffs were lowered
and no new dependencies were introduced. The profiler branch is separate.


## Latest verified continuation — intake-71/72

The complete frozen **real-application upload run is 36/99 documents correct
across all six requested fields** (30 fully populated, six legitimate absences),
covering 49 folder batches. This remains a development regression set, not an
unseen holdout. Evidence: `private/folder-batches/round-intake71`. Compared with
the first 46 batches of round67, three discipline failures recovered and one
existing failure changed; no previously correct field became incorrect.
The extra three files remain failures. This is not 90–95% accuracy.

Intake-72 is running on port 8080. Fresh uploads pass **35/35 fields on the
five-sheet vertical pack** (2.703 seconds) and **70/70 on the ten-sheet Petersham
mechanical pack** (4.407 seconds), including splitting, original-file integrity,
and duplicate-upload checks. Earlier vertical70 34/35 and mechanical66/70/71
69/70 results are retained. The mechanical installation-detail drawing had
near-even consultant/trade probabilities; explicit wording distinguishing
design details from records of completed installation raised page9 confidence
from 0.49 to 0.96. This follows TypeSafe's guidance on
[explicit questions and boundary cases](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
No confidence thresholds were lowered. A separate direct-classifier 98-file
check remains 36/98 complete; an unchanged bushfire kind question fell below
its threshold in that run. It is retained as a regression, not retried away.

Other verified changes:

- Captured Jev distributions sometimes sum to 0.99 from two-decimal rounding.
  The client accepts a one-cent discrepancy only for cent-precision values,
  while retaining all option, bound and finite-number checks. Raw confidence
  and probabilities remain unchanged. This is observed provider compatibility;
  the [API contract](https://docs.typesafe.ai/api) specifies a total of one.
  Exact replay of 98 recordings recovered one BASIX discipline and changed no
  other accepted values; complete-document accuracy remained 36/98.
- Discipline can explicitly abstain when authorship is unsupported. Contract
  parties no longer force an issuing trade. The agreement retest confirms this.
- The PPR amendment schedule now pairs Version3 with 07 October 2015. Its title
  and other classification failures remain unresolved and counted.
- The harness freezes evaluator, configuration and labels, records hashes, and
  rejects persisted Jev fields with a mismatched classifier version. The earlier
  interrupted round69 is not used as a complete accuracy result.
- Concurrent duplicate uploads exposed a Windows hash-file replacement race.
  Publication is serialised by hash stripe after streaming. Both existing
  duplicate tests passed 100 repetitions; full Go suites71 and72 pass.

Image-only DA consent remains not filed and counts as a failure. Metadata Jev
thresholds remain uncalibrated; the calibration tooling now supports an amber
fit independently of green, but no experimental thresholds were promoted.
Extraction latency has not consistently met its p90 budget. Further corpus
work is required; neither an accuracy ceiling nor completion is claimed.


## Current broad result — intake-67/68 continuation

**The latest complete application run is 35/96 documents correct across all six
requested fields**, including 29 with every field populated and six with a
legitimate absent field. This is a development regression, not an unseen release
holdout. Per-field correct counts (including correct absences) are number 79,
revision 71, title 45, date 69, kind 78 and discipline 71. The 46-folder-batch
run is `private/folder-batches/round-intake67`. Earlier larger numbers below
score different, partially labelled subsets and must not be read as a 95% live
application success rate.

The five-sheet vertical transport pack additionally passes 35/35 again on
intake-68 (2.703 seconds). Its original upload remains repaired. Full Go tests
pass; a new integration regression prevents cross-page supersession when pages
share a drawing number. Native reader changes retain all 497 previously correct
rule fields using fresh extraction evidence from 538 PDFs.

Two newly sampled Bankstown contract folders exposed confident wrong metadata:
a source standard was taken as the document number and a shortened filename as
the title. Intake-67/68 excludes explicitly “amended from” standard references
and offers the printed contract/agreement headings. Retests remove those false
green values but remain incomplete: legal references and titles are blank;
the agreement still has a wrong amber discipline. These failures stay in the
evidence. No contract was split into drawings.

A separate 538-PDF raw Jev diagnostic (`full-live-fit67b`) completed with no
provider errors. It is **not** 538 fully labelled documents: 206 lack gold,
86 are expected abstentions, and only 84 have all six labels. Its 59/84 complete
result must not replace the 35/96 application score. Extraction p50/p90 was
72.5/409.672 ms, failing the 80/250 ms budget; timing overlapped other local tests.

Calibration tooling now supports an amber-only review threshold independently
of a green automatic threshold, including export; regression tests cover both
sample and error limits. See [TypeSafe confidence guidance](https://docs.typesafe.ai/confidence).
This changes the testing machinery, **not production acceptance thresholds**.
The current diagnostic has too few independently labelled ambiguous metadata
answers to fit those bands. Its experimental category thresholds are not promoted.
More labelled failures and separate validation are still required.


## Vertical transport five-sheet repair — 3 October, intake-66

The live app on port 8080 and the owner's existing Test Multi upload now pass
**35/35 page-level checks** across all five sheets (six requested fields plus
lifecycle). Pages 1–3 share printed number N647414-01 and title LIFT LAYOUT
DRAWING; pages 4–5 share N647414-02 and CAR FINISH DRAWINGS. All five print
revision 1 and date 10/05/23. The incoming filename's Rev 02 is not substituted
for those printed revisions. Physical page identity keeps the five entries
separate without inventing document-number suffixes.

Two PDF-reader gaps caused the missing fields: unmapped characters invalidated
text geometry, and visible regulated-design stamp text was outside the normal
page text stream. Character-index recovery restores geometry; a bounded,
in-memory single-page copy exposes printed stamp appearances. The stored PDFs
are unchanged. A complete Drg No./REV./Page title-block parser then copies the
literal values. No Jev thresholds or questions changed.

A fresh real HTTP upload completed in 2.672 seconds with five single-page
children, unchanged original bytes and no flow failures. The existing upload
was refreshed using the parser and verified through the API and browser.
Private evidence: `drawing-sheets/vertical-live66`, `vertical-existing66.json`.

Regression evidence is deliberately separated:

- Full Go suite passes, including printed stamps and duplicate printed sheet
  identities with atomic publication.
- Fresh 538-PDF extraction followed by replay retains **497/497** previously
  correct rule fields (before the final native-run-preservation refinement).
- New 20-sheet upload retains **80/80** metadata checks; 52-page report remains
  one intact artifact.
- New mechanical upload splits all ten pages and retains **40/40** metadata
  fields, but scores **69/70** overall: Jev leaves page 9 discipline blank at
  confidence 0.54. This failure is retained, not retried away or reported as
  stable perfect accuracy. The title/author cues are present in Jev's input.
- The earlier broad 95-source score below remains the latest broad live score;
  these targeted repairs do not establish 90–95% accuracy on unseen files.


## Petersham mechanical bundle repair — 3 October, intake-65

The owner's `Mechanical Design & Spec [C].pdf` now splits into all ten sheets
in the existing Test Multi project. All **70/70** page-level checks pass:
document number, revision, full title, issue date, kind, discipline and lifecycle
on every sheet. This is a complete-pack development regression, not a claim
of 100% accuracy across unseen documents.

The missing title-block strings were AutoCAD SHX annotations, absent from the
ordinary PDF text API. The reader now recovers those strings with geometry and
provenance. The parser reads the complete title block and pairs revision C to
its history date, 01/12/2023. M01–M10 retain their own titles, page 2 is a Schedule,
the other pages are Drawings, all are Mechanical and remain under Design.
Number/revision/title/date and lifecycle are rule-derived; kind and discipline
are real Jev results with provisional amber review flags. Prompts and acceptance
thresholds were not relaxed to force this example through.

The existing original's hash and all ten one-page downloads were checked. Fresh
uploads and a repeat run pass metadata, splitting, duplicate identity and
original preservation. The final intake-65 build completed the whole background
pack in 4.234 seconds, again with 70/70 checks. Existing source and page evidence is private under
`data/eval/intake/private/drawing-sheets/petersham-*`.

Regression checks:

- Fresh extraction of 538 PDFs, then replay of 497 previously correct metadata
  rule fields: **497 retained, zero losses**. An intermediate implementation
  displaced native text; that regression was fixed before the final build.
- Earlier 20-sheet example: **80/80** number/revision/title/date checks retained
  through a new live upload (other classification fields are outside that score).
- 52-page waterproofing report: still one intact artifact, no split.
- Full Go suite passes, including CAD annotations, native-text budget protection,
  upward/downward revision tables, cross-page rejection and repeat job insertion.
- Candidate harvesting p50/p90 3.931/4.458 ms; rules 0.265/0.575 ms: pass.
- Full-corpus extraction p50/p90 80.538/433.289 ms: **fails** 80/250 ms budget.
  This run overlapped local live testing; the previously recorded extraction
  p90 failure also remains unresolved. Do not interpret unit-budget success as
  whole-corpus speed compliance.

See [drawing-set design](../../design/2026-10-03-drawing-set-intake.md) for limits
and the TypeSafe references. This repair adds no runtime OCR or alternative AI.

The additional 45-batch / 95-source live run is **not a passing broad accuracy
result**: 32/95 six-field-correct outcomes (26 fully populated, six legitimate
absences), compared with 34/95 in intake-62. Raw correct field totals were
number 78, revision 69, title 45, date 66, kind 75, discipline 67. Jev category
changes include losses on identical extracted text, as well as changed CAD
evidence on some fire-services files. Do not describe these as zero end-to-end
regressions. An intermediate downward revision-table date regression was found
in the facade report and repaired; a focused final live rerun recovered its
date (1/1), but that report still fails other fields. This focused rerun is not
silently substituted into the recorded 95-source aggregate. Evidence:
`folder-batches/round-intake65` and `folder-batches/facade-final65`.

## Current folder-by-folder hardening (intake-62)

This supersedes the manual-test status below, not the historical measurements.
The live development server on port 8080 now runs intake-65. The historical 95-source aggregate below describes intake-62. Existing documents generally retain their prior results; the owner's Test Multi examples were explicitly refreshed after validation.

**Why many fields stay blank:** the live threshold configuration currently has no acceptance bands for Jev-selected title, number, revision, date or lifecycle values. Deterministic rules can populate these fields, but ambiguous metadata choices remain blank even when Jev returns a selection and confidence. Kind and discipline have provisional amber-only bands. Missing candidates, incorrect raw choices, and correct-but-withheld choices must be measured separately before calibrating additional bands; a blank alone does not prove Jev chose incorrectly. Broad calibration and held-out validation remain unfinished.

The Bankstown Final Contract Set and Camp Y inventory contains 576 PDFs in 73
PDF-bearing folders. DWGs and AppleDouble sidecars are ignored. The folder queue,
independent labels, source conflicts and per-file predictions are private local
evidence under `data/eval/intake/private/folder-batches/`.

| Small batch | PDFs | All six populated correctly | Remaining issue |
|---|---:|---:|---|
| Bankstown mechanical drawings | 3 | 3 | None in this sample |
| Camp Y mechanical drawings | 3 | 3 | Extraction p50 exceeded budget in a diagnostic run |
| Bankstown electrical drawings | 3 | 0 | All three issue dates absent from extracted text; intermittent discipline failure retained |
| Camp Y electrical drawings | 3 | 2 | One source conflicts: filename/history T2, own revision cell T1; revision/date remain blank |
| Bankstown structural drawings | 3 | 0 | Sparse usable text: wrong discipline suggestions, missing numbers/dates, incomplete titles |
| Camp Y structural drawing packs | 3 | 3 | First-page identity only |
| Bankstown hydraulic drawings | 3 | 2 | Wrong Civil suggestion; earlier intermittent discipline blanks retained |
| Camp Y hydraulic drawings | 3 | 1 | One correct blank source date; earlier intermittent discipline blanks retained |
| Bankstown architectural drawings | 3 | 0 | Image-only PDFs; all six target fields remain failures |
| Camp Y architectural drawings | 3 | 3 | None in this sample |
| Bankstown BCA report | 1 | 0 | Image-only identity pages |
| Camp Y BCA reports | 2 | 0 | One correct six-field outcome with absent number/revision; other title/kind unresolved |
| Bankstown acoustic report | 1 | 0 | Revision absent in source; intermittent discipline failures retained |
| Camp Y acoustic documents | 2 | 0 | Specification correct with absent number; DA report title/revision unresolved |
| Bankstown access report | 1 | 0 | Wrapped title unresolved; source has no own number, revision or date |
| Camp Y access documents | 3 | 0 | Cover titles and two prefixed revision conflicts unresolved |
| Bankstown fire engineering report | 1 | 0 | Correct six-field outcome with absent document number; lifecycle unresolved |
| Camp Y bushfire documents | 2 | 0 | Letter title and conflicting assessment revision unresolved |
| Camp Y fire-services documents | 3 | 1 | Earlier discipline blanks retained; specification title unresolved |
| Camp Y fire schematic | 1 | 0 | No extractable text; stays in the failure denominator |
| Camp Y civil drawings | 3 | 2 | All titles/numbers/revisions/dates correct; cover kind unresolved; all three lifecycle values now Design |
| Camp Y civil DA | 2 | 1 | Both six-field outcomes correct, report number legitimately absent; report lifecycle unresolved |
| Camp Y civil WAD | 1 | 1 | First-sheet identity correct; other pages not individually validated |
| Bankstown strata plan | 1 | 0 | Revision recovered; visible filled values missing from usable PDF text |
| Bankstown consolidation plan | 1 | 0 | False form-caption title removed; visible identity values missing from usable PDF text |
| Bankstown landscape drawings | 3 | 0 | No usable text; two own-revision cells conflict with filenames/history |
| Camp Y landscape documents | 3 | 3 | Both drawings and specification correct on all six requested fields; specification lifecycle remains blank |
| Camp Y landscape DA | 3 | 0 | Drawing title recovered and revision conflicts detected; one textless PDF; report metadata unresolved despite reading its history |
| Bankstown arborist | 1 | 0 | Reference/date recovered, revision absent in source, title and lifecycle unresolved |
| Camp Y arborist | 3 | 0 | Both report references recovered; complete title candidates available but unselected; addendum date and tree-plan identity unresolved |
| Bankstown BASIX package | 1 | 0 | Broken font-to-text mapping; BASIX discipline correct, primary report identity missing |
| Camp Y ESD reports | 3 | 0 | False project-number assignments removed and cover revision recovered; titles and Section J discipline unresolved |
| Bankstown traffic | 1 | 0 | Full printed title candidate recovered, but title remains withheld; source has month/year only |
| Camp Y traffic | 2 | 0 | Full references and assessment revision recovered; titles and response-letter kind unresolved |
| Camp Y ASP3 | 3 | 0 | False project-code number removed; drawing identity and report title/date unresolved; one discipline label provisional |
| Bankstown airspace approvals | 2 | 0 | Both scanned; stored but not filed |
| Camp Y audio visual | 3 | 2 | Specification title candidate present but withheld; lifecycle blank |
| Camp Y facade | 1 | 0 | Issue date recovered; title withheld; discipline varies |
| Camp Y flood | 2 | 0 | Full council reference recovered; title and classification limitations remain |
| Camp Y geotechnical | 1 | 1 | Full reference and equivalent dates recovered; lifecycle blank |
| Camp Y hazardous materials | 1 | 0 | False filename revision removed; title, revision and lifecycle unresolved |
| Camp Y ecology | 3 | 0 | Full metadata candidates recovered; titles, versions and two issue dates remain blank |
| Camp Y wastewater | 2 | 0 | Soil report date recovered; title limitations and unsupported manual taxonomy remain |
| Camp Y survey | 1 | 0 | Outlined lettering has no readable text layer; stored but not filed |
| Camp Y waterproofing | 1 | 0 | Own reference recovered; revision, date and lifecycle remain blank |

Across forty-five batches, the latest intake-62 full-app run (`round-intake62`) has 28/95 files with all six fields populated correctly and 34/95 correct six-field outcomes. Six legitimate source-absence cases are separate from populated success; 61 files fail. These are document-identity results, with first-page-only scope for drawing packs, not all-sheet accuracy. Twelve PDFs have no readable identity text layer. All 497 previously correct rule fields remain correct. Compared with intake-61 (36/95 correct outcomes), four kind/discipline fields regress, one kind recovers and an already-failed discipline changes. Candidate lists are identical across all 95 sources between those versions; these observed differences are Jev classification variation, not recovered Block B metadata. The latest lower result is retained.

Waterproofing testing recovers the printed document number from an explicit page-three control section referenced by the contents page. It excludes the following product certificates and footer, retains both revision-history rows, and removes a falsely confirmed filename revision. The actual app still leaves revision, date and lifecycle blank. Browser evidence is retained. Full Go tests pass. Quiet harvest p50/p90 is 3.403/3.539 ms and rules 0.218/0.417 ms; fresh corpus extraction is 70/379.147 ms, still failing the 250 ms p90 budget.

Bundled drawing-sheet intake is implemented and running on port 8080. Each admitted drawing page is separately filed, with source/page provenance and downloads; reports remain whole. The original stays available, publication is atomic, and failed or ambiguous expansion is reviewable/retryable. Full Go tests and frontend build pass. Synthetic flow tests cover report/mixed-page retention, duplicate upload, download integrity and tenant isolation. Real Camp Y external works: three sheets split, 2/3 six-field correct (page 2 discipline withheld). Block B architectural (intake-62): two sheets split, both drawing numbers and full titles correct; revision/date still blank. The page-2 gold title was corrected only after viewing the source render: it includes /Sections. Historical gold and results remain retained. Waterproofing: 52-page report retained whole. Details and limits are in `docs/design/2026-10-03-drawing-set-intake.md`.

A new complete-pack harness, `tools/check-drawing-sheets.py`, compares independently labelled pages and validates downloads and duplicates. Legacy corpus tests now explicitly record first-page-only scope when a pack splits and score the first child's visible fields. The post-feature 95-source regression is complete. Four packs published 50 individual sheets; three packs remain intact for review because a page did not meet Drawing acceptance. Those 50 sheets have not all been independently labelled, so no whole-pack accuracy percentage is claimed.

Survey testing distinguishes outlined lettering from a scan: the first page has extensive vector paths, one image, and no font or text operators. The visible title block was independently labelled, but the app cannot extract its identity and stores it without filing. It remains a failed file. The browser explanation now says lettering may be scanned or saved as shapes; it no longer assumes that every textless PDF is scanned. Frontend typecheck/build and browser verification pass. No extraction or classification behavior changed for this batch, so the prior 93-file regression remains applicable.

Wastewater testing recovers literal month-first issue dates, including uppercase month names, while rejecting impossible dates and incomplete month/year values. Sparse manual covers now offer their printed capital heading as a candidate even at body-text size. The browser verifies the report date and removal of a falsely confirmed abbreviated manual title. Both PDFs still fail: titles remain withheld, the soil report title includes a defective extracted glyph, and the taxonomy has no manual kind or operations lifecycle. Provisional mappings are now explicitly excluded from threshold fitting by a validated `CalibrationExcluded` map; they remain in scored evidence. Tests verify that the exclusion does not silently remove the case from evaluation.

Ecology testing now retains explicit report metadata rows behind a sparse cover without retaining surrounding disclaimer prose, then probes the next page for explicit document control. The parser recognises `Report:` and separate `Version:` captions and keeps Draft and Final versions distinct. Focused tests reject unrelated values and cross-page binding. All three real uploads now show Report and Ecology in the browser; false filename revisions and incomplete filename titles are no longer confirmed. Full title and version candidates are available, but those fields remain withheld under the empty metadata acceptance bands. Two issue dates are also unresolved; the third source prints only a month/year. All three remain failures, not successful extractions.

Hazardous-materials testing reaches an explicit third-page control table only after two sparse covers sharing repeated text. Ordinary body pages and drawing sheets do not enable that extra probe; only explicit control content is retained. Parsing the merged Revision/Issued To heading exposes the printed revision conflict and removes a falsely confirmed filename revision. The corroborated cover date remains correct despite older dates in its issue history; an unrelated date still prevents that rule from settling. App and browser verification agree. Revision, title and lifecycle remain withheld, so this PDF still fails.

Geotechnical testing recovers the full dotted report reference with its underscore/type suffix. Complete ordinal dates now compare as the same calendar day as equivalent numeric dates, while keeping the displayed source literal. Focused tests preserve conflicting-day and abbreviated-year ambiguity. The app and browser show all six requested fields correctly; lifecycle remains blank, so the broader seven-field gate still fails.

Flood testing adds complete letter-and-digit references with a slash/year suffix. A regression test checks the full literal and rejects partial fragments of malformed references. The app and browser now show the complete council reference. Both issue dates and the assessment's decimal revision remain correct. The advice title is withheld; the assessment title remains an incomplete filename title. Council issuer discipline and the nearest available category for flood modelling are taxonomy limitations, with provisional labels excluded from calibration decisions. Both PDFs remain failures.

The facade report exposed revision-table Date captions containing a format hint. The parser now accepts bounded dd.mm.yy/yyyy and dd/mm/yy/yyyy captions and aligns their values by the column's left edge. A focused regression test first failed and now passes, including a separate print date. App and browser confirm the recovered issue date. Revision 0 and printed 000 already compare as the same numeric revision; no revision accuracy fix is claimed. The complete title remains withheld, so this PDF still fails six-field evaluation.

Both airspace approvals are image-only scans: independent extraction returns no text on either page, and the app/browser report stored but not filed. Their visible file reference and signed date were independently labelled before upload. Both remain failures. A government issuer has no matching consultant/trade discipline in the current taxonomy; that label limitation is recorded rather than inventing a discipline.

ASP3 testing removes an unlabelled project-code repetition from document-number candidates when the cover explicitly identifies that code in Project Name. An explicitly labelled document number remains eligible. Drawing lifecycle now follows an accepted Drawing kind as Design, inherits its review band, and respects user corrections. The browser distinguishes “Check · Rule” from “Check · Jev”; amber does not itself prove an AI call. All three ASP3 PDFs still fail six-field evaluation. One report's electrical-versus-environmental discipline label is provisional and excluded from calibration decisions; another has conflicting printed dates. These source ambiguities remain documented rather than silently resolved to match predictions.

Camp Y traffic testing replaces a falsely confirmed engineering-registration number with each document's full dotted reference. Compact centred capital headings are now literal title candidates. A bounded second-page approval-history read recovers revision B; matching the printed cover date to its row handles reused draft/final revision letters without assuming the last row is current. Browser verification confirms references, report revision and dates. Titles remain blank under the uncalibrated metadata policy; the response-letter kind remains unresolved. Both PDFs still fail the six-field test.

Full Go tests pass; the unchanged frontend previously passed typecheck/build and its UI detector. Fresh extraction with the intake-55 reader (unchanged in intake-56) has p50/p90 of 70.524/370.604 ms and fails the 250 ms p90 budget. Current parser harvesting p50/p90 is 3.293/3.417 ms; rule p50/p90 is 0.218/0.408 ms. Both parser benchmarks pass. An initial full-suite rule-budget failure prompted allocation profiling: preallocating field candidate lists reduced allocation from about 1,018 KB to 405 KB per decision run. The subsequent full suite passes; the original failed measurement remains recorded. Earlier measurements below are historical.

The Bankstown traffic report exposed a heading-size estimate dominated by large cover controls. Using the lower quartile of text sizes recovers its two-line descriptive title as a candidate while retaining the existing prominence requirement. The wrong abbreviated filename title is no longer confirmed; the title remains blank under the current uncalibrated Jev metadata policy. Reference, revision, kind and discipline are correct; only a month/year is printed, so a complete date is legitimately absent. Full Go tests and the 497 earlier rule checks pass. Latest harvesting p50/p90 is 3.009/3.135 ms; rule p50/p90 is 0.342/1.128 ms and still fails the 1 ms budget.

ESD testing exposed separately positioned Project No and Rev cover cells. Project references and their unlabelled header repetitions no longer become document numbers; a short Rev is read as current report revision only in the bounded Project No/Date cover column. Tests include an ordinary-table negative case. Browser checks confirm the number and revision changes. The titles are available as candidates but remain unselected; two Section J disciplines remain blank. The BASIX PDF uses custom embedded font encodings without Unicode maps and remains unreadable to both independent and application text extraction. Its initial general ESD gold label was corrected to the taxonomy's specific BASIX assessor category, with the original label retained for audit; this did not make the document pass.

The full Go suite passes. Latest parser harvesting p50/p90 is 3.007/3.041 ms; rule p50/p90 is 0.315/1.317 ms and fails the 1 ms budget. The earlier whole-corpus extraction p90 remains 396 ms against 250 ms. Detailed earlier iteration measurements below are historical.

Camp Y arborist testing recovered inline and separate-cell report references, four single-word cover heading lines, and the full Report Type title without its revision suffix. The live app and browser agree: both report numbers are correct, but titles remain blank after Jev selection. The addendum date remains blank. The tree plan lacks usable extracted identity text and still receives an incomplete filename title. All three remain failures. The full Go suite passes. Latest harvesting p50/p90 is 3.048/3.137 ms; rule p50/p90 is 0.406/1.225 ms and fails the 1 ms budget. The tree-plan extraction takes about three seconds. The prior whole-corpus extraction p90 of 396 ms also remains above its 250 ms budget. No timing gate is claimed green by repeating until it passes.

The following paragraphs describe earlier iterations and their measurements; the intake-57 results above are the current status.

Arborist testing recovers an explicitly labelled spaced report reference and ordinal month date. Centred assessment headings now retain all three printed lines as a candidate. The browser confirms number/date, kind and discipline; title remains blank, and this PDF still fails. The full Go suite and 497 earlier rule checks pass. The latest parser benchmark passes harvesting (2.884/2.978 ms), but rule p90 is 1.095 ms, above its 1 ms budget. That failed gate remains recorded.

Landscape DA now reads the report's explicit second-page issue history after its sparse application cover. Repeated revision/date/authorisation labels allow the bounded control-page read; ordinary body pages are excluded. The printed cover heading becomes a literal candidate. Conflicting filename revision/title values are no longer confirmed. The browser shows unresolved report revision, title and date; this remains a failed filing. The drawing's own Issue cell similarly prevents a conflicting filename revision being confirmed, while its title is recovered. One PDF has no usable text. All three remain in the failure denominator.

Full Go tests pass. Fresh extraction of 538 PDFs preserves all 497 earlier correct rule fields; extraction p50/p90 is 71/396 ms and still fails the 250 ms p90 budget. Parser benchmarks pass at 2.539/2.601 ms for harvesting and 0.252/0.283 ms for rules. The prior benchmark failure during concurrent uploads remains recorded in the private checkpoint.

Landscape testing recovered Drawing Name captions, merged number/title cells, full composite specification references and dates paired to the current revision in a bottom-caption issue table. A drawing schedule column no longer becomes the current sheet title. Recognising the specification number initially exposed false neighbouring title candidates from remote headers and labelled revision/date values; geometry and caption checks remove those candidates. The final three-file Camp Y batch is correct on all six requested fields, verified in the browser. Its specification lifecycle remains unresolved, so the broader seven-field gate still fails. Bankstown's three textless landscape PDFs remain failures; no OCR or other AI was introduced.

The full Go suite and parser benchmarks pass. All 497 older correct rule fields remain correct; the accumulated live batches were rerun after both landscape iterations. The latest evidence is `private/folder-batches/round-intake50/results.json`. Neither these small development samples nor the historical partial-label checks establish 90–95% population accuracy.

Survey-plan testing recovered numeric ISSUE tags and removed unsupported confirmed filename dates, bare job numbers and short acronym titles. A land-registry Title System caption no longer becomes a document title. Both survey plans remain failures: their visible filled-in identity values are largely absent from usable extracted text. Print and survey-completion dates are not silently treated as issue dates.

Civil DA now has two correct six-field outcomes. The report reads an explicit second-page document-control table after its sparse cover. The cover date corroborates revision 2, the repeated Document Subject corroborates the descriptive title, and the project number is excluded. Browser verification confirms the saved fields. The report lifecycle remains blank. The large drawing pack continues to pass using bounded range reads.

Fresh extraction of 538 earlier PDFs preserves all 497 previously correct rule fields. Its latest p90 is about 396 ms, above the 250 ms budget.

These are development samples, **not a population accuracy estimate**. Samples
currently cover drawings, three BCA reports and three acoustic documents and four access documents and three fire/bushfire documents and four fire-services PDFs and three civil drawings; specifications and schedules in those folders
are different layouts and are not validated by these successes. Every blank
remains in the denominator. The ambiguous Camp Y source is not silently counted
as a complete filing. Its provisional latest-issue target is recorded alongside
the conflict and must not be used to force a guessed revision into production.

Civil baseline intake-28 had no correct full numbers or dates. Intake-29 recovered all three full numbers and revisions by binding merged values to explicit drawing-number/revision columns. Intake-30 pairs merged revision/date rows above bottom captions, recovering all three current issue dates without choosing the historical first-issue date. Full Go tests pass and all 497 previously correct rule fields remain correct. Intake-33 also recovers all three titles: bounded drawing-title cells include sheet subtitles, and the cover schedule row matches the independently bound own number. Browser verification agrees with saved fields. Two civil drawings now have all six requested fields correct; cover kind remains unresolved, so the batch is not complete.

The original mechanical pair was uploaded through the browser file chooser and
all six rendered fields checked. Three-file repeats use fresh projects through
the running application's upload endpoint, including real Jev calls and saved
decisions. Kind and discipline remain amber suggestions. Numbers, titles,
revisions and dates use literal document evidence where deterministic rules
can establish it. No threshold was relaxed to conceal missing metadata.

Fixes cover right-angle title blocks, wrapped titles, issue/date row pairing,
multi-part drawing numbers, job-prefix filenames, merged initials/date cells
and overlapping glyph boxes. Later fixes add dotted drawing numbers, current
revision cells, bottom-caption issue tables and merged/stacked captions. Camp Y architecture
adds four-part numbers, right-aligned identity cells and revision-register
heading exclusion.
CAD plot timestamps are excluded from issue-date
candidates; a coinciding calendar date is not proof of the issue date. Initial changes disturbed 17 earlier correct
fields; narrower rules restored them. All 497 previously correct labelled rule
fields still pass. The full Go suite passed after the current changes. A Windows
concurrent-upload file-lock test failed once and passed on the full rerun; that
intermittent failure is recorded, not claimed fixed.

`tools/check-folder-batches.py` reruns all labelled batches in its manifest,
captures fresh extraction, uploads to fresh local projects, scores persisted
fields and returns failure when any batch fails. The cumulative intake-28 round
passed both mechanical batches and Camp Y structural, hydraulic and architecture,
and correctly failed the incomplete electrical, Bankstown structural, hydraulic
and image-only architecture batches. Structural discipline suggestions were wrong
on all three files (Architecture from readable architect details), not merely
blank; these failures remain visible in the private results.
The BCA batch caught bracketed month/year filename tags being parsed as document
numbers and equivalent calendar dates being treated as conflicting. Both are
fixed and verified through fresh app uploads. Prominent report headings now enter title candidates, so conflicting filenames
no longer win uniquely. That conflict remains unresolved; no threshold was lowered.
Report-number syntax and explicitly labelled project references are now separated.
The Bankstown acoustic report was also checked in the browser against its cover. Full Go tests, all 497 older
rule-field regressions and candidate/rule performance benchmarks pass. The
current cumulative evidence is `private/folder-batches/round-intake28/results.json`.
Access testing adds decimal revision harvesting and excludes explicitly identified
job references and copyright template codes from document numbers. Wrapped cover
headings are now captured as literal candidates, including headings beside a
large badge. Conflicting filename titles stay blank pending validated selection. Decimal revisions do not enable automatic
supersession ordering.
The reader change was checked by freshly extracting all 538 older PDFs, then
confirming that 497 previously correct labelled rule fields were preserved.
This is a regression check, not a 538-file accuracy claim. Extraction p90 was
374 ms, still above the 250 ms budget.
Fire testing adds literal version labels, explicit letter subjects and merged
revision/date history rows. An overbroad subject-caption rule briefly disturbed
a waterproofing title; the regression check caught it and the narrowed rule
restored all 497 prior correct rule fields. The fire-report title now resolves through agreement between its explicit
document-control title and its separate cover heading.
Sparse-cover heading detection now handles equal-sized groups of smaller control
text and larger title/address text. The fire-services specification exposes a
real cover-title versus filename conflict and remains blank rather than falsely
accepting the filename. Both sampled fire-services drawings pass all six fields.
The broader release gate remains failing, including coverage and performance.

**All 538 unique PDFs were exercised through extraction, live Jev classification and the actual SiteWise API. The overall 95% accuracy release gate remains failing.**

## Manual testing clarification

**Bankstown coverage audit:** all 179 PDFs in `Bankstown Final Contract Set`
are byte-identical to the previously tested Bankstown PDFs. However, Bankstown
had reference labels for only 9 discipline fields, 15 kind fields and 4 lifecycle
fields, with **zero reference labels for number, title, revision or date**.
Consequently the headline identity accuracy ratios do not validate Bankstown
identity extraction. This is a validation coverage failure, not evidence that
the manually encountered errors are new. The next evaluation must fully label
a folder-balanced Bankstown set before making further accuracy claims.

The current reader finds no identity-page text in 61/179 Bankstown PDFs. Direct
content inspection of the two architectural files reported by the user found
one image object per page and no PDF text-showing operators. Selectable text in
a viewer could come from viewer OCR; that behaviour has not been verified.

The accuracy table below covers only fields with independent reference labels;
it is **not the population rate of complete metadata across 538 PDFs**. Unlabelled
fields are excluded from those ratios, so poorly covered layouts can perform
much worse in manual testing. Do not read 145/145 numbers as all drawing numbers
being recovered, or 183/190 disciplines as all documents having a discipline.

The user's normal server on port 8080 was still running `intake-2` when they
compared these results with manual uploads. Their project had 69 blank Jev
discipline decisions, 67 with returned confidence: Jev had answered, but values
were not applied. On 2 October the normal development server was rebuilt and
restarted with `intake-12` and the final amber-only configuration.

A fresh six-file check on port 8080 verified six successful HTTP 200 filing
calls to `jev-1.13.0`. Kind populated on 6/6, discipline on 5/6, drawing number
on 4/6 and title on 2/6. These are population counts, not independently scored
accuracy. Mechanical titles and stormwater metadata remain real gaps after the
restart. Private evidence is in `private/manual-8080-verification/`.

For a fresh manual test, refresh the browser, create a **new project**, and
drop the PDFs there. Restarting does not reclassify existing documents;
re-dropping the same bytes in the same project returns the existing result,
even if the filename changes. This behaviour must not be confused with a fresh
classifier run.

The legend "Check: Jev's pick, flagged" is static explanatory text. A populated
field marked "Check Â· Jev" is an applied amber suggestion; "From document"
means a deterministic rule. "Not set" alone does not say whether Jev ran:
it can mean missing candidates, a none selection or a disabled/below-cutoff
choice. Server logs and persisted `decided_by`/`confidence` provide the audit.

## Scope and evidence

The owner authorised extended testing across all three corpora. Seven full-corpus live draft rounds and four full-corpus persisted-app rounds are retained locally. Only extracted identity text went to TypeSafe Jev; full PDFs stayed local. App rounds used fresh QA projects, an isolated database and only one worker build at a time.

Inventory: 544 PDFs, 538 unique after six duplicate copies, and 179 excluded AppleDouble sidecars. The final full app pass produced 452 filed documents, 85 not filed and one HTTP 413 upload rejection. Across the corpus, 83 PDFs lack first-page text and three exceed the 32 MiB identity limit. Independent extraction with a second reader completed on 536 PDFs; two had font-encoding failures in that reader. All 83 textless identity pages were independently confirmed textless. No OCR was added.

Gold v10 has 332 cases: 246 partially annotated documents and 86 expected extraction abstentions. Another 206 PDFs have no gold. Labels are draft agent annotations, not owner-reviewed. They come from independent page text, visual inspections or frozen register entries. Source corrections and explicitly evidenced literal title alternatives are preserved alongside earlier gold versions; predictions are not automatically accepted as truth.

## Results

Correct/scored includes blanks as failures. The baseline is the **start of extended testing**, after the earlier sample fixes, not the original parser. Both comparisons use the same frozen gold v10. The app column checks persisted values with fresh provider responses.

| Field | Correct before extended | 0 | Correct full app | App blank | App wrong |
|---|---:|---:|---:|---:|---:|
| number | 145/145 | 145/145 | 145/145 | 0 | 0 |
| discipline | 0/190 | 183/190 | 183/190 | 6 | 1 |
| kind | 19/185 | 165/185 | 165/185 | 19 | 1 |
| lifecycle | 167/181 | 167/181 | 167/181 | 14 | 0 |
| title | 1/145 | 134/145 | 134/145 | 10 | 1 |
| revision | 108/148 | 122/148 | 122/148 | 26 | 0 |
| date | 65/131 | 96/131 | 96/131 | 35 | 0 |

These denominators are labelled fields, not all 538 PDFs. Per-corpus, expected-discipline and calibration/validation-family metrics are in [summary.json](summary.json). An unreadable PDF is an expected abstention, not a correct classification.

The final full app run used amber kind and the then-candidate green discipline cutoff. It exposed one wrong green discipline decision; repeated live responses also exposed that error. **Final production config has both kind and discipline amber-only.** Exact-request replay rechecked all 538 PDFs with that final config, with no provider errors or replay misses. A subsequent fresh app check on 12 representative/problem PDFs verified all 13 populated kind/discipline fields saved as amber. The full app's earlier green band is preserved in evidence, not rewritten.

## Changes

- Bind drawing numbers, current Revision cells and sheet titles using PDF geometry; join tightly aligned multiline titles.
- Exclude short REV history-table headings from current revision binding. A full-corpus regression caught and tested this distinction before the final run.
- Prefer explicit sheet titles over CAD Layout export suffixes, while leaving conflicting descriptive filenames unresolved.
- Settle dates from explicit revision/date pairings; do not ask Jev to order dates.
- Distinguish main documents from attached compliance declarations and distinguish drawings, specifications, schedules and certificates. Filename kind words no longer settle kind.
- Preserve bounded author evidence for discipline selection and separate the author from other consultants mentioned on the page.
- Keep technical drawings in Design, including construction-issued drawings, as directed by the owner.
- Freeze request recordings, gold snapshots, run/config hashes and private review queues for reproducibility.

Final cutoffs are discipline 0.60 for 33-64 options and kind 0.80 for 9-16 options, **amber only**, with 79 and 93 calibration answers respectively. They retain conservative previously tested cutoffs rather than lowering them after validation. Fresh response variation caused errors, so no green cutoff is promoted for either field. Other unsupported question shapes remain disabled. Deterministic rules still operate.

## Remaining blockers

The release gate stays red for incomplete reference coverage, blank fields, one title/reference disagreement and extraction latency. The title disagreement is a descriptive filename versus a shorter register title on a product-sheet compilation; it remains visible for adjudication. Kind and discipline suggestions can still be wrong and require review.

Live identity extraction measured 70.1 ms p50 / 373.8 ms p90 against the 80/250 ms budget. Live draft latency was 311.4/507.6 ms. App polling-observed filing was 344.0/562.0 ms, excluding upload time. These are local Windows measurements, not a VPS release benchmark.

The gate also exposes the difference between the app's HTTP 413 rejection and the identity reader's size-limit abstention. Neither is counted as a successful filing.

Validation families have been inspected repeatedly during development and are **not an untouched release holdout**. Related templates and sparse Newham/Bankstown labels limit generalisation. Completing independent labels, acquiring a fresh reviewed holdout, resolving remaining identity candidates and reducing extraction p90 are required before claiming very high overall accuracy.

## Reproduce and inspect

See [CORPUS.md](../../../data/eval/intake/CORPUS.md). The combined offline gate runs exact-request replay and persisted-app scoring and returns nonzero for remaining failures:

```powershell
./tools/check-corpus.ps1 -PerFolder 0 -Gold data/eval/intake/private/corpus-gold-v10.json -Recordings data/eval/intake/private/full-live-20261002-07/recordings.jsonl -AppResults data/eval/intake/private/app-12-20261002/results.json -Out data/eval/intake/private/release-check-next
```

Final evidence is in `private/release-check-20261002/`; the post-routing app probe is `private/app-bands-final/`. Each scoring run writes `review.json` with blanks, wrong values, wrong green values, status mismatches and unlabelled documents, linked by SHA256 to immutable traces. PDF text, filenames, annotations, recordings, QA projects and document IDs stay git-ignored. Public evidence contains aggregates only.

QA is available at http://127.0.0.1:8082/dev/login, database `sitewise_qa_20261002`, with final routing loaded. The normal development server on port 8080 was subsequently rebuilt and restarted with the same final code/config; its existing database and files were preserved. See the manual testing clarification above.

Validation: full Go suite, `go vet ./...`, Python upload/poll/413/resume integration test and whitespace checks. One intermediate duplicate-upload test hit a Windows file-sharing error; five targeted repeats and later full suites passed. This remains a recorded intermittent risk, not a claimed fix.


### Latest sheet and raw-choice diagnostics

Intake-59 rejects merged adjacent reference captions such as `C.A.P. No FILE No`
as titles. A focused test failed before the fix and passes after it; the actual
split ASP3 sheet now leaves title blank instead of confirming a caption. Full
Go tests pass, and quiet harvest/rule p50/p90 are 3.456/3.601 ms and 0.226/0.504
ms. The previously reported real-corpus extraction p90 failure remains unresolved.
A forced local-server interruption during pending expansion resumed to exactly
three children plus the original. Concurrent publication and transaction rollback
after a mid-batch identity collision also pass store tests.

A separate intake-59 raw Jev diagnostic over the 95 development samples records
12 extraction failures and no provider errors. Of questions actually asked and
answered, raw choices matched labels for number 4/8, revision 11/12, title 15/30,
date 7/10, kind 78/81, discipline 71/82, and lifecycle 29/45. These denominators
exclude fields already settled by rules; they are not whole-document accuracy.
This shows that simply lowering acceptance bands would also admit wrong values.
The samples were repeatedly inspected during development, so these measurements
are not untouched holdout validation and no new thresholds were promoted.


Intake-60/61 title diagnostics use literal printed-title wording and preserve the
PDF heading cue in candidate descriptions, following TypeSafe's [jev-1.13
limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13). On the same 30
unresolved development cases, raw title matches rose from 15 to 18; eight still
lack a candidate matching the independent label. Repeated development samples
are not held-out validation, and metadata acceptance bands remain unchanged.

Intake-62 adds bounded parsing for uncaptioned drawing cells alongside a Project
Number / Print Date / Drawn By / Scale control column. It does not mark these as
explicitly captioned fields. Both Block B drawing numbers and titles now match
the rendered source. Tests reject missing anchors and unrelated page, rotation,
position and font-size references. All 497 earlier correct rule fields remain
correct. Full Go tests pass; quiet harvest p50/p90 is 3.403/3.539 ms and rules
0.218/0.417 ms. Complete-pack retests retain the prior Camp Y 2/3 six-field result
and the intact 52-page report. Block B remains 0/2 fully correct because revision
and date are blank. The two drawing-pack checks also verify each downloaded
sheet has one page, source bytes are unchanged, and duplicate uploads add no rows.


## Test Multi metadata repair (intake-63)

The exact 20-page L09 CC Plans upload now passes 80/80 **printed title-block
metadata comparisons** (number, revision, title, date), on two fresh full-app
uploads and the refreshed original project. Both fresh uploads split all 20
pages in 11.36/11.844 seconds end to end; that is background pack completion,
not the initial upload's latency. Original retention, one-page downloads and
duplicate checks pass. Browser verification confirms corrected existing rows.

The parser recognises the bounded Sheet/Scale, Drawn By/Date and
Version/Construction grid; it reads the literal sheet index, current version,
uncaptioned title cell and date. It excludes approval-stamp prose from title
candidates. No consultant name, project ID, expected title list, page count of
20 or file hash is encoded in runtime rules. Unit tests reject cross-page
anchors and incomplete blocks. Refreshing existing sheets preserves user edits
and stale-version protection, tested at the store boundary.

**Source conflict:** all outer title blocks state version 3 and DATE 07.03.24;
the cover's revision history dates version 3 as 12.04.24. The 80-field result
measures faithful transcription of the title blocks, not resolution of that
issue-date conflict. Drawing numbers use the printed SHEET indices 1–20;
no job-number prefix is invented. This is not a six-field classification score
or an accuracy claim for other consultant layouts.

Full Go tests pass, as do all 497 previously correct rule fields. Additional
stamp/grid boundary tests pass. Harvest p50/p90 3.966/4.203 ms and rules
0.235/0.852 ms pass their budgets. Reader unchanged; the previously reported
corpus extraction p90 failure remains. Jev prompts and acceptance bands were
not relaxed. Private source text, labels, traces and before/after evidence are
under `data/eval/intake/private/drawing-sheets/test-multi-*`.
