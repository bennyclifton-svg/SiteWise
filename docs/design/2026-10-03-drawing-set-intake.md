# Bundled drawing intake

Status: implemented and deployed to the local port-8080 development app; broader corpus hardening continues.

The owner requests that a PDF containing separate drawing sheets produces one
scheduled document per physical page. Reports, including reports with drawing
appendices, remain one document. The original PDF remains downloadable.

## Boundaries

Page count, page copying, hashes and source-page links are deterministic code.
Whether a document is a drawing is a Jev decision using the existing admitted
kind. A filename, landscape page or large paper size alone is not permission to
split. An unresolved kind retains the original and a visible unresolved state.
Technical drawings retain the owner's Design lifecycle policy.

Use the existing PDFium dependency to copy pages, preserving text, graphics,
rotation and page boxes. Do not rasterise sheets or introduce another runtime
AI. Clerk's splitting and provenance mechanism was inspected as reference;
none of its code is copied.

## Delivery design

1. Keep ordinary first-page intake fast. A multi-page admitted drawing becomes
   a candidate for background expansion. Reports do not enter expansion.
2. Before publishing children, check page identities independently. Admitted schedules
   inside an admitted drawing set are valid sheets and retain Schedule kind.
   Standalone schedules do not enter expansion. A report
   page or unresolved sheet keeps the source intact for review; no partial
   successful split is claimed. Mixed bundles need explicit handling rather
   than blindly splitting every multi-page PDF.
3. Persist expansion state so a process restart can retry it. Publish children
   and source-page links atomically. Source/page uniqueness prevents duplicate
   rows on retries; org and project boundaries apply to every reference.
4. Each child has its own one-page PDF and six-field decisions. The original
   filename must not leak its first-sheet number/revision into later sheets.
   Preserve the source filename separately from classification input.
5. Show individual sheets in the schedule, with page index, total pages and a
   link to the retained source. The container must not collide with the first
   child's document number/revision. Expansion errors remain visible and
   retryable; do not silently present a pack as one completed drawing.

Use one fan-out per page state, per TypeSafe's
[fan-out pattern](https://docs.typesafe.ai/patterns/fan-out), and existing
[confidence routing](https://docs.typesafe.ai/patterns/confidence-routing).
Sheet expansion is background work and must yield PDFium between pages.
Measure expansion latency separately from the existing 1/2-second intake
budget; include page count and errors in evidence, never omit slow packs.

## Required verification

- Synthetic two-page drawings: distinct text and rotated title blocks survive
  one-page exports; invalid page, malformed PDF, limits and cancellation fail.
- Real Camp Y structural packs and samples from the added Block B/Newham
  corpora: independently label every published sheet's six fields.
- Reports with appendices stay one artifact; textless/mixed packs are retained
  without inventing identities or claiming successful expansion.
- Repeated/concurrent upload and crash/restart do not duplicate children.
- Tenant/project isolation, source download, individual downloads, schedule
  display and event refresh are covered through the real application.
- Rerun previous extraction/rule checks and full-app samples. First-page
  success from earlier pack tests is not whole-pack accuracy.

The additional corpus roots are recorded in the private evaluation inventory.
They supplement the existing Bankstown/Camp Y folder queue.

## Implemented verification, 3 October 2026

The background worker checks every page with one independent filing fan-out,
then publishes all children in one database transaction. An unresolved or
page not admitted as Drawing or Schedule leaves the original intact with a review explanation. Failures
are retryable; pending work is resumed at startup. Limits are 200 pages, 512 MiB
source reads and 128 MiB per exported page. No new runtime dependency was added.

Full Go tests and frontend build pass. Integration tests cover drawing versus
report handling, mixed-page rejection, duplicate upload, source-byte retention,
one-page downloads, page provenance and tenant isolation. The sheet-copy build
gate is p50/p90 80/250 ms over 20 warm samples; the small fixture measured 0/0.999
ms (clock resolution limits the p50). This is not a real-corpus latency claim.
First-page filing retains its existing 1/2-second budget; later pages run in the
background using Jev background priority and releasing PDFium between pages.

A real three-sheet Camp Y structural set split in 1.407 seconds including upload
on the deployed app. All number/revision/title/date values match independent
page labels; page 2's discipline remains withheld, so only 2/3 sheets pass all
six fields. A two-sheet Block B architectural bundle also splits. Intake-62 recovers both
drawing numbers and full titles; revision and date remain unresolved on both. A 52-page waterproofing report remains one intact artifact.
These examples demonstrate the flow, not 90–95% classification accuracy.

`tools/check-drawing-sheets.py` tests complete packs against page-level gold and
validates downloads and duplicate handling. The older corpus-upload harness now
waits for splitting and evaluates the first child against its existing first-page
gold; it records `EvaluationScope: first_page_only`, the source and every child.
Those legacy pack results must never be counted as whole-pack correctness.


Schedule regression: the actual Test Multi L09 CC Plans PDF now publishes all
20 pages, including the window schedule on page 14. Browser verification and
download validation confirm 20 one-page PDFs and an unchanged original. The
HTTP regression covers drawing-plus-schedule, standalone schedule, report and
drawing-plus-report retention; the full Go suite passes. No Jev prompt or
acceptance threshold changed.

## CAD text annotations (intake-65)

The Petersham mechanical example has ten physical sheets. AutoCAD exported
part of their printed lettering as paths and stored searchable strings in
annotations named `AutoCAD SHX Text`. PDFium's normal text API omitted those
strings, including the title block. The first-page kind consequently remained
unaccepted, so the drawing expansion never started.

The identity reader now supplements native text with these explicitly named
CAD annotations, normalises reversed rectangle coordinates and records their
annotation provenance. Ordinary reviewer comments are excluded. Traversal is
capped at 4,096 annotations, each string at 4,096 bytes, and the combined result
retains the existing 400-run limit. Native text is never displaced: annotations
fill remaining space, retaining both ends of their sequence. Pages already at
the native limit do not read annotations. This limitation is deliberate until
we have a measured region-selection alternative; it is not OCR support.

The parser recognises a TITLE cell immediately above a drawing-number/revision
row and prefers that complete title block over abbreviated regulated-design
records. Revision-history columns can grow upwards or downwards; dates are
paired by column geometry and the independently read current revision, without
guessing the latest date. In this example the issue date comes from revision C's
row (01/12/2023), not an invented day for the month-only `2023/12` cell.

No kind threshold or Jev question was relaxed. Jev still selects kind and
discipline in the existing fan-out; code copies literal metadata and keeps
technical sheets in Design. This retains TypeSafe's
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [fan-out](https://docs.typesafe.ai/patterns/fan-out) division of work.

Re-filing an existing source also exposed a duplicate background-stage insert.
Job insertion is now idempotent by organisation/document/kind, preserving the
existing job instead of rolling back the new filing. User field corrections
remain protected by the existing optimistic decision writes.

The independently transcribed ten-sheet gold checks all six requested fields
plus lifecycle, source-byte preservation, one-page downloads and duplicate
upload identity. Synthetic tests cover SHX-only text, reversed rectangles,
ordinary-comment exclusion, annotation survival after splitting, native text
at its run limit, both revision-table directions and cross-page rejection.

## Repeated printed sheet numbers and stamp appearances (intake-66)

A drawing can span several physical sheets with the same printed document
number and revision. `documents.identity_page` is zero for ordinary artifacts
and the physical source page for split children. The identity index includes
this position. Atomic publication assigns it before filing; migration 007
backfills existing children. Numbers are never decorated with invented suffixes.

A complete Drg No./REV./Page row identifies the manufacturer's drawing block,
including a nearby Project Title cell. Revision and page columns remain distinct.
For PDFs whose returned text length differs from PDFium's character count,
per-character Unicode lookup preserves coordinate indexing. Printed stamp
appearances are extracted from a temporary single-page copy flattened for
printing and reopened; the original is never changed. Serialization is capped
at 32 MiB, annotations at 4096, and native runs retain priority within 400 runs.
Ordinary annotation comments are not treated as printed text. This adds no OCR.

The five-sheet vertical transport development regression passes all 35 fields
in a fresh upload and the existing project. Repeated live classification remains
probabilistic: a subsequent mechanical regression retains every metadata field
but leaves one discipline blank. Threshold calibration and broad holdout accuracy
remain unfinished; no acceptance bands were relaxed for this repair.


Supersession guard (intake-68): persisted links additionally require equal
physical identity_page. An integration test supplies the wrong prior page with
the same drawing number and confirms it is refused while the matching page link
is retained. Physical order changes remain a review concern; shared printed
numbers alone do not establish which page replaces which.
