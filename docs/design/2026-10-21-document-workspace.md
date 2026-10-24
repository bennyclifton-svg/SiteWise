# Document workspace and repository interactions

The owner requested moving document viewing out of the repository's expanded
rows and into the centre workbench. This makes the source readable without
mixing its preview with filing suggestions. This slice is layout lane B, with
a narrowly scoped lane C change allowing authenticated PDF responses to be
embedded by the same origin. No new dependency or Jev call is introduced.

The row's Open button opens one document in the centre. A normal row click
selects only that row; Ctrl/Cmd toggles, Shift replaces the current range while
keeping curated selections, and Ctrl/Cmd+Shift adds a range. Checkboxes toggle
without needing Ctrl. Opening a document does not change that selection.
Closing returns to the prior workbench view, whose
component remains mounted. Choosing a navigation item closes the document.

The Document view shows the whole PDF through the browser's PDF viewer, or
Markdown as literal text (never executable HTML). Word and Excel currently
offer download rather than a misleading preview. The existing field editors,
OCR recovery, provenance and image-assisted suggestions move unchanged into a
separate Filing details view. Nothing is accepted automatically. PDF downloads
remain attachments; only PDF requests with `view=1` allow same-origin framing.
Organisation checks and file lookup remain the existing download path.

Done means opening, switching and closing documents leaves repository columns
stable; the previous workbench and bulk selection survive; field corrections
still save; wrong-org, malformed and missing document requests fail; failed
Markdown loads are reported; narrow screens do not overflow.

The existing document-list budget is p50 50 ms / p90 150 ms; field correction
is 50 / 150 ms and filing stays 1,000 / 2,000 ms. The browser regression measures
20 document-list reads. Native PDF rendering depends on document size and the
browser and is not represented by that list-read measurement.

## Inline editing and selection

- Double-click, F2 or Enter on a field starts editing. Kind, Discipline and
  Lifecycle use the existing vocabulary; other fields use text. Enter saves,
  Escape cancels, Tab saves and focuses the next visible field, and a changed
  text value saves on blur. A refused save retains the draft and shows an error.
- Program and Documents import the same selection functions and inline editor
  from `web/src/table/`. Persistence stays with each caller: document corrections
  use the existing API and retain user provenance. There are no new Jev calls.
- This is lane A. Filing, org isolation, Program editing and bulk operations
  remain covered by the existing tests. The register test covers mixed modifier
  selection, sorting, cancellation, retries, vocabulary edits and persistence,
  and measures 20 corrections against the existing 50 / 150 ms budget.
- Lifecycle is shown in the wide register. Responsive widths still hide the
  less-used columns; the Filing details view keeps every field accessible.

## Follow-on direction, not implemented here

- Simple rectangles, arrows and text are feasible annotations. Store them in
  page coordinates against an immutable document revision, separately from
  original bytes. Snips and annotated export need a controllable page renderer;
  the native PDF iframe is not that editing surface. Word previews need a
  deliberate local conversion/rendering solution before being promised.

## Bulk classification and shared table presentation

Selected filed documents can receive one Discipline, Kind or Lifecycle value
(including None) through the selection toolbar. Apply snapshots the selection
and reuses existing validated corrections, with at most four requests in flight.
Each successful value retains user provenance. Partial failures are reported;
only failed documents remain selected for retry. This is not an atomic batch.
Mixed selections containing unfinished or unfiled documents cannot be corrected.
No new service, dependency or Jev call is introduced.

Documents and Program use one selection-colour token and one borderless inline
editor treatment. Focus uses a subtle background without changing row geometry.
Supporting columns are compact; recognized dates display DD/MM/YY while stored
values and editor contents stay unchanged. Routine review triangles are removed
from rows; field evidence remains in Filing details, and stopped/stored statuses
remain visible. Titles retain over 180px at a 650px panel width.

Validation: production build and document/Program editing browser tests pass,
including bulk vocabulary edits, clearing, partial failure and retry, user
provenance, title width and editor geometry. Twenty correction requests measured
p50 13.8ms / p90 15.2ms against the existing 50ms / 150ms per-correction budget.
