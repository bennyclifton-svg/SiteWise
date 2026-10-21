# Hale title-block vision experiment

19 October 2026. Lane A, isolated experiment. Status: completed exploratory
comparison; review-only; no rollout or threshold change. Answer key is draft
and has not been reviewed by the owner.

## Decision

Image-assisted candidate selection is promising for background title review.
Box is not reliable enough for the tested title-cell localisation task. Keep
the source page and deterministic overlapping bottom/right regions available
to the reviewer. Do not replace the existing filing path or auto-accept these
results.

This experiment addresses recurring missing titles, not drawing understanding,
number/revision acceptance, supersession, full-document OCR, or a new screen.
No application records, user corrections, thresholds, production Jev model,
or accepted project values were changed.

## Scope and method

The local Hale architectural upload from 16 October produced 15 split sheets.
All 15 have native text. The stored title snapshot has one correct rule title
and 14 blanks. These are not 15 image-only scans.

An org/project-scoped read-only SQL query created an exact document/hash
manifest. The Go harness checks the original bytes against each hash and calls
the real identity extractor, OCR extractor, candidate harvester and filing
draft. Baseline requests use the existing pinned `jev-1.13.0`. Experimental
requests pin `jev-1.14.0` and use the images opt-in header. There is no database
write path in either harness.

Five Petersham documents were inspected/extracted locally as potential
controls, but no Petersham data was sent to TypeSafe: approval review limited
the live scope to the specifically authorised Hale set. The live manifest now
contains only Hale. Cross-template performance remains untested.

The draft answer key was visually transcribed from the printed cells. The
cover has no own title block: its label is supported by the drawing list and
is scored separately in interpretation. Exact correctness normalises case and
whitespace only. View captions, partial titles and reordered wording fail.
The answer key is never passed to Jev.

Methods:

1. Saved application titles, as a snapshot of the reported symptom.
2. Existing native-text filing request, recording raw selections before title
   acceptance thresholds; settled rule values are retained.
3. Existing OCR forced on the same sheets, then the same filing draft. This is
   a counterfactual comparison, not the normal route for these text PDFs.
4. Matched candidates-only text and image questions. Candidate options are
   literal native/OCR lines plus existing harvested titles; no generated text.
   Images include the full page and generic bottom/right regions.
5. Whole-page Box plus presence, then padded crop OCR and a later candidate
   selection. This deliberately measures a background, dependent pipeline.
6. One bounded refinement: four overlapping bottom/right regions with one
   presence and Box question per region in the same call. Pick the highest
   box-confidence qualifying region, map to page coordinates, then crop OCR
   and candidate selection. The presence gate of 0.8 is provisional, not
   calibrated. Box confidence is not treated as field correctness.
7. An additional matched pair uses the original native-text state and generic
   adjacent multiline candidates, with and without images.

The initial run encountered Windows default-decoding problems. The primary
comparison was rerun in full with explicit UTF-8, preserving the initial
recordings. The regional run did not consume that text, and the contextual
pair used UTF-8 throughout; their recordings were retained unchanged.

## Final results

| Method | Exact / 15 | Wrong | Unresolved | p50 / p90 ms |
|---|---:|---:|---:|---:|
| Saved titles | 1 | 0 | 14 | Not measured |
| Existing native-text filing, raw | 7 | 6 | 2 | 295 / 339 |
| Existing OCR filing, raw | 0 | 15 | 0 | 2,772 / 3,840 |
| Candidate-only text control | 4 | 2 | 9 | 2,973 / 3,922 |
| Same candidates with images | **13** | **1** | **1** | **3,890 / 4,837** |
| Whole-page Box + crop OCR | 0 | 3 | 12 | 2,297 / 2,589 |
| Regional Box + crop OCR | 0 | 1 | 14 | 1,370 / 1,651 |
| Full text + multiline candidates | 5 | 2 | 8 | 2,958 / 3,991 |
| Same full context with images | 12 | 2 | 1 | 3,941 / 4,825 |

Removing the cover, image-assisted selection is 12/14 and the raw native
baseline is 6/14. The first image configuration's 13/15 result also occurred
in the initial run, but this is not an independent held-out evaluation.

All rows are proposals for evaluation, not safe acceptance rates. In
particular, the saved application withheld many raw wrong answers. Forcing
OCR often leaves only the generic filename as a title; its 0/15 does not
mean the production application newly wrote 15 wrong titles.

The image-selection API itself measured p50 530 ms / p90 589 ms. The table
includes native extraction, OCR and harvesting whenever those candidates were
used, plus rendering for image methods, plus live API wall time. Box totals
include rendering, crop OCR and the second selection call where reached.
Short-circuited failures can make Box appear fast.

These are reconstructed component totals on the Windows workstation, one
sample per unique sheet, with OS caches potentially warm. They exclude
upload, database commit, SSE, queue waiting and browser paint. Extraction and
render preparation are shared measurements, not repeated for each arm.
There are only 15 unique sheets, below the budget registry's 20-sample gate.
This is not a release benchmark or proof of live filing SLA compliance.

Compared with the existing budgets:

- Ordinary filing remains p50 1,000 / p90 2,000 ms. The experimental image
  path does not qualify for it; production is unchanged.
- Background OCR target is p50 10,000 / p90 20,000 ms on an idle queue. These
  exploratory image results are below those magnitudes, but the existing
  upload/worker gate was not rerun as a vision-integrated system.

## What caused the repeated failures

There are several layers, not one consultant-specific rule to add:

1. **Title acceptance is disabled without calibration.** The current
   provisional title threshold list is empty. A correct raw selection still
   stays blank. This policy must not be bypassed with an arbitrary cutoff.
2. **Candidate recall is incomplete.** Only 10/15 exact titles occur in the
   existing native title candidates; 13/15 occur as complete native text
   runs. Short labels such as elevations/sections and multiline titles need
   better generic representation before a selection model can succeed.
3. **OCR regions cut through cells.** The existing large-sheet renderer uses
   disjoint perimeter bands at the 80% boundary. On Hale, a title crossing
   that boundary is split into fragments. The OCR title-candidate set has
   no exact complete titles on these 15 sheets.
4. **View captions compete with sheet identity.** The image path still picks
   an internal ground-floor view caption on one sheet. More context also
   changes another selection to an internal caption. Context and prompt
   sensitivity remain material.
5. **Box localisation is weak at this scale.** Whole-page boxes commonly
   point into the neighbouring project/purpose cells. Of 14 actual title
   cells, only three selected padded crops retain at least 95% of the title
   glyph rectangle, derived from visually checked native text geometry.
   None yields an exact final title. The regional refinement has no selected
   crop meeting that coverage criterion. It mainly abstains at the presence
   gate, so do not generalise this into a claim that every raw box is wrong.

The lower-ground title spans two printed lines. Adding a generic multiline
candidate makes the complete title available, but the image model still
abstains. Better candidate recall is necessary, not sufficient.

## Does it make review easier?

The private HTML report puts each proposed title beside the whole sheet and
enlarged, overlapping bottom/right regions. Each region covers about 15.4%
of the original page, so the reviewer has less area to search without losing
access to the original. It explicitly shows incorrect and unresolved cases.
The cover still needs its drawing list inspected. No model box is presented
as verified evidence.

This supports a useful review interaction, but **no human checking-time or
error-rate study has been completed**. The owner was invited to compare three
examples. Reduced area and more correct suggestions are proxies; do not claim
measured time saved or that human review has been validated.

## Recommendation and next boundary

Proceed, if separately requested, with image-assisted **review suggestions**
and deterministic overlapping source regions in the existing filing review.
Keep native text first, avoid forcing OCR on readable drawings merely to make
images available, and use OCR only for missing candidate evidence. Those
optimisations have not been timed here.

Do not ship Box-based title crops or automatic title acceptance from this
experiment. Before either, evaluate held-out consultant/layout families,
right-edge and bottom-edge blocks, cover/index pages, rotation, absent or
multiple title cells, long/multiline titles and sparse/scanned inputs. Use a
labelled held-out set for field thresholds, not these same tuned examples.

## Artifacts and validation

- `cmd/titleblock-experiment/main.go`: real extractor/harvester/draft export.
- `tools/titleblock_experiment.py`: scoped local preparation, recorded API
  comparisons, region refinement, matched context and report generation.
- `tools/test_titleblock_experiment.py`: offline geometry/selection checks.
- Private, git-ignored `tmp/titleblock-experiment/`: exact scope, source
  hashes, draft labels, extracted evidence, initial/final response recordings,
  local timings, metrics, crop coverage and `review.html`.

No new runtime dependency or service. Uses the existing Python, PDFium,
Pillow, Tesseract, Go and direct TypeSafe HTTP API. Each request uses narrow
typed questions; code owns cropping, mapping, candidate copying and flow.
There are no project-name or consultant-template rules in title selection.

Validation passed:

```
go test -p 1 ./cmd/titleblock-experiment ./internal/intake ./internal/identity
python -m unittest discover -s tools -p test_titleblock_experiment.py
```

Go integration tests used the dedicated `sitewise_test` database, not the
project database. A final read-only check still found 15 Hale sheets, 14 blank
titles and no user title changes. Pre-existing working-tree edits were left
intact. Baseline measurements use this checkout, not a claim about the exact
binary that processed the original upload.

For reproduction, export a fresh explicitly authorised org/project/hash
manifest into a new private directory. Run the Go exporter, then Python
`prepare`, `run`, `regions`, `context` and `report` with that directory as
`--root`; the exporter has separate `--manifest` and `--output` flags. Supply
visually checked draft labels for `report`. API actions are live and billable.
Existing per-action response files are replayed; **use a fresh directory when
changing prompts, inputs or preparation**. API timing in replayed reports is
the recorded live timing, not replay speed. Do not use the report's
Hale-specific filename sorting as a general corpus runner without adapting
that presentation step.

## TypeSafe references

- [Images](https://docs.typesafe.ai/images): experimental status, image assets,
  preprocessing and limits.
- [Box](https://docs.typesafe.ai/primitives/box): single-target coordinates,
  presence companion, confidence and small-target limitations.
- [Fan-out](https://docs.typesafe.ai/patterns/fan-out): independent questions
  share a state; dependent crop-reading occurs in a later background state.
- [Pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook):
  literal candidates, selection, copying in code.
- [Confidence](https://docs.typesafe.ai/confidence): per-action calibration.
- [API](https://docs.typesafe.ai/api) and
  [models](https://docs.typesafe.ai/models): pinned models and typed requests.
- [Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13):
  baseline limitations; not a substitute for a 1.14 vision evaluation.
