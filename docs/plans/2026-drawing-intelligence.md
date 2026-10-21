# SiteWise drawing intelligence: Jev Images and Box handoff

Oct 19, 2026 · @Ben clifton

## Purpose and status

This handoff carries the context from a review of using TypeSafe Jev's Images and Box features to read construction drawings in SiteWise. Nothing here is built yet, and the owner has not approved any of it.

- **Owner-stated facts:** the owner has a large drawing corpus across every discipline. They believe most consultants use similar, largely universal symbols. They are interested in counting symbols, extracting dimensions and fill types, and eventually turning a sheet's graphics into a structured schema.
- **Everything else below is a recommendation** from the review, not an owner decision. Section "Open decisions for the owner" lists what still needs a yes or no.
- **Repo reviewed:** `bennyclifton-svg/SiteWise`, branch `main`, commit `d12897f` (17 October 2026). The GitHub forks page could not be fetched, so forks were not reviewed.

## Repo context

SiteWise is a Go + PostgreSQL + React workbench for one building on Australian construction projects. Jev (TypeSafe System One) is its only AI. Read `PRODUCT.md`, `AGENTS.md` and `docs/design/2026-09-29-foundation-design.md` before changing anything.

**Non-negotiables that shape this work**

- Jev only, and Jev only picks. Code parses, computes, counts and controls flow. Images and Box are Jev, so they do not breach this rule.
- Speed budgets fail the build. Filing is p50 ≤ 1 s, p90 ≤ 2 s. One Jev fan-out per state, and no serial round trips on a hot path.
- The building is the model (`knowledge/SCHEMA.md`). Disciplines are a view over systems.
- Never invent a rate, price, rule number or product. Unknown stays unknown.
- Every field shows how it was decided and how sure it is. Bands are green, amber, blank and grey. Green auto-acceptance is currently withheld.
- Cite the TypeSafe docs page for every Jev use.

**Current intake pipeline**

| Area | What it does today | Where |
| --- | --- | --- |
| Native text | PDFium text runs, capped at 400 per page. When over the cap it keeps the first and last 200, hoping the title block is in one end. | `internal/identity/pdf.go` |
| CAD text | Reads `AutoCAD SHX Text` annotations as extra text. | `internal/identity/pdf_shx.go` |
| Title-block finding | About 3,600 lines of geometry heuristics plus about 30 quirk tests (rotated, right-aligned, uncaptioned, CAD, unit plans). | `internal/intake/candidates.go`, `layout.go`, `sheet_grid.go`, `uncaptioned_sheet.go`, `revision.go` |
| Classification | Code harvests candidates; one Jev fan-out picks kind, discipline and identity fields. | `internal/intake/questions.go`, `internal/jev/` |
| Drawing sets | Multi-page drawing PDFs split into one-page children in the background. | `docs/design/2026-10-03-drawing-set-intake.md` |
| OCR | Textless PDFs get a background Tesseract pass at 150 DPI over generic top, right and bottom bands. OCR fields are capped at amber. | `docs/design/2026-10-03-upload-ocr.md` |
| Profile reading | Drawings feed the profile only when a user opts a document in. | `internal/profile/reading.go`, `questions.go` |

**Known gaps:** Bankstown CC-01 to CC-05 have lettering drawn as vector paths, with no text layer (CC-02 has about 591k line commands). The Jev client is JSON-only and pinned to `jev-1.13.0`. Specifications still lists designed PDF tables and images as ahead. Eval tooling lives in `cmd/corpus-eval`, `cmd/intake-eval` and `data/eval/`.

## TypeSafe Images and Box

Images can go anywhere text goes in a request: the state, a question's instructions, or its criteria. Box is a question type that returns one bounding box as four edges, each a fraction of image width or height. Both are labelled "Dreamer-ready", meaning TypeSafe does not yet back them for production.

| Item | Fact |
| --- | --- |
| Opt-in | Send `TypeSafe-Danger-Zone-Images: true` on every request with an image or a Box question. |
| Models | Image questions work on Jev 1.13 and 1.14. Box needs `jev-1.14.0` or `jev-preview`. |
| Formats | PNG or JPEG stills only. Rotate pixels upright; EXIF orientation is ignored. |
| Size | Each side 280 to 1,568 px; total 78,400 to 2,458,624 px; 16 MiB per file. Out-of-range images are rejected, never resized. |
| Request | 32 MiB body, 32 MiB for all images, 32 MiB expanded. Each use of a reference adds the image again. |
| Upload | Multipart recommended: first part `request` (JSON), then one file part per asset. Base64 in JSON also works. |
| Box answer | `top`, `bottom`, `left`, `right`, `confidence`. Always returns a box, even when the target is absent. One target per question. |
| Mixing | A request can mix Box with Noul, Choice and Score against the same state. |
| Criteria | Choice options, Score levels and Noul true/false can be defined by example images. |
| Tokens | Billed as input tokens. One third-party measurement on jev-1.14.0: 2 × ceil(w/28) × ceil(h/28), about 6.3k tokens for 1568 × 1568. TypeSafe has not published guidance yet. |
| Model lifecycle | Jev 1.14 released 14 October 2026; `jev-latest` was to move to 1.14 about a week later. |

**Patterns the docs name:** compare against a known picture, define options by example, what changed between frames, look closer when needed, search a large image by tiling into Choice options, and find where it is with Box. To check a box, crop to it and ask one Noul per edge ("does the target continue past the left edge?"). In TypeSafe's test, per-edge questions caught 30 of 36 cut-off crops.

## Division of work

Code finds, reads and counts; Jev names and locates. Jev never transcribes text, measures or counts, so every number still comes from code.

| Task | Code | Jev |
| --- | --- | --- |
| Title block, revision table, legend, schedule location | Renders the sheet, crops, keeps only text runs inside the box | Box, paired with a Noul for presence |
| Identity fields | Harvests candidates from runs inside the title-block box | Existing Choice fan-out picks among candidates |
| Outlined lettering | Renders the boxed region at 300+ DPI and runs Tesseract | Box to aim the OCR |
| Symbol counting | Groups repeated vector shapes or reused CAD objects; counts instances exactly | Choice names each group once, with legend or library crops as example criteria |
| Dimensions and scale | Parses dimension text and the printed scale | None |
| Fills and hatches | Groups closed shapes by fill colour or pattern; computes areas from scale | Choice matches each group to a legend swatch |
| Sheet content | None | Choice or Noul on a downscaled sheet: sheet type, level, systems shown |

**How Box and image questions combine:** one request per tile can ask a Noul (is it there), a Choice (which legend symbol is it) and a Box (where). A second request on a higher-resolution crop runs only when confidence is low. Because that is serial, image work belongs in the background, never the 1-second filing path.

**A full graphical schema of every element is a direction, not a first deliverable.**

## Opportunities by surface

The strongest product case is counts and drawn content tied to the specification, procurement and revision history, each with provenance. Takeoff tools already count symbols, mostly by hand; they do not link counts to the building model.

| Surface | Opportunity | Depends on accurate counts? |
| --- | --- | --- |
| Filing | Box locates the title block and revision table; code harvests only runs inside. Could retire much of the layout heuristics and fix the 400-run truncation. Show the crop beside each field. | No |
| OCR | Box aims a high-DPI OCR crop at the title block instead of generic bands. Helps Bankstown-style outlined lettering. | No |
| Specifications | Gap check: a fitting type drawn but not covered by any schedule row shows "Needs attention". Legends and finish schedules located for the import. | Presence only |
| Revisions | Rev D shows a different count or new revision clouds compared with rev C; affected spec rows and cost lines are flagged for review. Gives Variations evidence. | Roughly |
| Profile | Systems found on sheets become evidenced profile facts with crops, without per-document opt-in. | Presence only |
| Procurement | Packages proposed from what is drawn (sprinkler heads suggest fire services). RFTs carry drawn quantities marked "from drawings, check". Tender comparison flags bids well below the drawn count. | Yes |
| Cost Plan | Counts become quantity evidence on lines the user prices. No rate is invented. | Yes |

All image-derived values start as amber proposals, the same as OCR fields.

## Using the drawing corpus

Jev is not trained or fine-tuned by SiteWise. The owner's corpus becomes three assets instead.

1. **Symbol library.** Cropped examples of each symbol, used as example pictures in Choice criteria. Link each symbol to a system or component in the knowledge catalogue so drawings feed the building model. A project's own legend takes priority over the library when present.
2. **Gold set.** Hand-labelled sheets per discipline: title-block boxes, symbol counts per level, legend entries. Used to measure accuracy and set per-question confidence thresholds, as the banding rules require. Existing page-level identity gold already has run coordinates, so title-block gold boxes can be derived from it.
3. **Shape signatures.** Normalised vector signatures of each symbol, so code can find and count instances without Jev.

Keep library images small and reuse them carefully: each question that references an example image sends it again.

## Risks and constraints

| Risk | Detail | Mitigation |
| --- | --- | --- |
| Resolution | A full A1 sheet at 1,568 px wide is about 1.9 px/mm; 2.5 mm lettering is about 5 px tall. Fine for layout, too small for text or small symbols. | Tile and narrow in; crop at higher resolution for detail. |
| Latency | Rendering heavy vector sheets plus serial tile requests will not fit the 1-second filing budget. | Background worker only, with its own measured budget, like OCR. |
| Accuracy unknown | Images and Box are not production-backed; no measured accuracy on drawings. | Gold set first; cap at amber; no green until thresholds are calibrated. |
| Absent targets | Box always returns a box. | Pair every Box with a Noul and ignore low-presence boxes. |
| Several targets | One box per question. | Choice of none/one/several, then tile. |
| Cost | Image tokens add up; repeated example images are re-sent per question. | Measure tokens per sheet in the pilot; put shared images in state once. |
| Geometry varies | Symbols differ across consultants and CAD exports; outlined text mixes with symbols; scans have no vectors. | Legend-first matching; exclude scans from counting at first. |
| Model change | Box needs 1.14; current thresholds were calibrated on 1.13. | Keep text filing on 1.13 until re-evaluated, or re-run the full intake eval before switching. |
| Client | `internal/jev` is JSON-only. | Add asset support (multipart preferred) and the opt-in header. |
| Product rules | Counts must not become invented quantities or prices. | Counts are evidence with crops; users price and accept. |

## Recommended sequence

Start with work that pays off even when counts are approximate, and gate each step on measured accuracy.

1. **Jev client support.** Add image assets (multipart) and the opt-in header to `internal/jev`, with tests. No behaviour change to filing.
2. **Title-block locator eval.** Offline: render sheets, ask Box for the title block and revision table, compare with gold boxes derived from existing identity gold on Camp Y, Block B and Petersham. Report overlap, confidence, latency and tokens.
   - Gate: the box contains every gold title-block run on a high share of sheets the owner agrees on.
3. **Use the box, no question changes.** In the background path, filter runs to the box and aim OCR at it. Keep existing Jev questions and thresholds. Re-run the intake eval and speed gates.
4. **One-discipline symbol pilot.** One electrical lighting or services plan with a clear legend. Code groups shapes and counts; Jev labels groups against legend crops. Compare per-level counts with a hand count.
   - Gate: counts match the hand count closely enough for the owner to trust a flag.
5. **Spec gap check and revision flags.** Surface presence and count changes as review flags with crops.
6. **Profile evidence, then procurement and cost quantities.** Only after counting accuracy is measured per discipline.
7. **Fills, hatches and other disciplines.** Extend the library and gold set one discipline at a time.

Each step needs its own design note in `docs/design/`, cited TypeSafe pages, and a benchmark in `bench/budgets.json` per `AGENTS.md`.

## Open decisions for the owner

- [ ] Approve using Dreamer-ready Images and Box features at all, given TypeSafe does not back them for production.
- [ ] Choose model strategy: keep filing on `jev-1.13.0` and use 1.14 only for Box work, or move everything to 1.14 after re-evaluation.
- [ ] Set a background latency and token budget for drawing reading.
- [ ] Pick the pilot discipline and sheets for the symbol count.
- [ ] Decide who labels the gold set and how many sheets per discipline.
- [ ] Agree the gate thresholds for steps 2 and 4.
- [ ] Decide whether drawings should feed the profile automatically, replacing per-document opt-in.
- [ ] Confirm whether OCR should stay Tesseract with Box-aimed crops.

## References

- [TypeSafe: Box](https://docs.typesafe.ai/primitives/box)
- [TypeSafe: Images](https://docs.typesafe.ai/images)
- [Third-party jevclient: image token measurement](https://pypi.org/project/jevclient/1.3.1/)
- [Jev 1.14 release coverage](https://www.remio.ai/post/jev-1-14-0-release-expands-what-jev-can-see-but-typesafe-wont-call-it-production)
- [SiteWise repo](https://github.com/bennyclifton-svg/SiteWise), commit `d12897f`
- In repo: `PRODUCT.md`, `AGENTS.md`, `docs/design/2026-09-29-foundation-design.md`, `docs/design/2026-10-03-drawing-set-intake.md`, `docs/design/2026-10-03-upload-ocr.md`, `docs/evidence/2026-10-03-architectural-intake-investigation.md`
