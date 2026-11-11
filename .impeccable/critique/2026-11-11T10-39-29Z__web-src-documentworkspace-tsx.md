---
target: the document viewer opened from the register
total_score: 20
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
timestamp: 2026-11-11T10-39-29Z
slug: web-src-documentworkspace-tsx
---
⚠️ DEGRADED: single-context (sub-agents are not spawned in this session unless the owner asks; the design read was finished before the detector ran)

Target: the document viewer (`web/src/DocumentWorkspace.tsx`, `web/src/document-workspace.css`, with `VisualIdentityReview.tsx` and the `FieldCell` grid from `DocumentRow.tsx`). Reviewed on the dev server with a 13-page fee proposal PDF at 2237×1167 and 1440×900, light and dark, in Chromium 153 and Edge 157 (headless, through Playwright; the Chrome extension was not connected).

## Design health score

| # | Heuristic | Score | Key issue |
|---|-----------|-------|-----------|
| 1 | Visibility of system status | 2 | No loading state for the page; the nav still shows the previous surface; to-check filing is invisible while reading |
| 2 | Match system / real world | 3 | Plain wording; number and revision are absent from the reading view |
| 3 | User control and freedom | 2 | Esc does not close; switching views resets the PDF |
| 4 | Consistency and standards | 1 | Close cross, pills, mono field cells and the browser toolbar are used nowhere else in the workbench |
| 5 | Error prevention | 2 | The browser's markup tools invite work SiteWise does not keep |
| 6 | Recognition rather than recall | 2 | A field and its source are compared across two views |
| 7 | Flexibility and efficiency | 2 | Enter opens from the register; no Esc, no next document, no page links |
| 8 | Aesthetic and minimalist design | 2 | About 250px of chrome above the page; two scrollbars |
| 9 | Error recovery | 2 | PDF failure is a permanent hedge line; the Markdown error has no Retry |
| 10 | Help and documentation | 2 | Tooltips only; adequate for this surface |
| **Total** | | **20/40** | **Acceptable (low end)** |

## Design specificity verdict

LLM assessment: the viewer is the one workbench surface that could belong to any product. It uses none of the incumbent grammar (ground-line Back button, three-dot menu, section-band heads, label | value | origin rows, pencil-underlined tabs). The only product-specific content, per-field provenance in Filing details, sits behind a pill.

Deterministic scan: `detect.mjs` over the three markup files returned 0 findings. The in-page detector (Chromium, both themes, both views) reported nothing inside the viewer. One page-level finding, `clipped-overflow-container` on `#main`, agrees with the nested-scroll issue below. Its other findings sit outside the viewer (register title cells: tight leading; hidden mounted surfaces; shell-level heuristics; the detector's own labels).

Visual overlays: none; the run was headless.

## Overall impression

The architecture is right (centre pane, register beside it, previous surface kept mounted) but the frame is unfinished: it spends the screen on chrome, borrows the browser's toolbar, and separates a filing from its source. Biggest opportunity: one header row in the workbench's grammar, then the page edge to edge, with filing beside it.

## What is working

- Opening keeps the previous surface mounted and bulk selection intact; closing returns exactly there. Enter opens from the register.
- Filing details states who decided each field and never applies a suggestion without an explicit Use.
- Sound semantics: labelled section, real buttons with `aria-pressed`, logical tab order, Markdown as literal text, PDF response with a tight CSP.

## Priority issues

1. **[P1] The frame takes the screen from the document.** At 2237×1167 the page starts about 255px below the title band (stacked Download and close 72px, padding, title, pills, native toolbar). At 1440×900 with the register open the title wraps to three lines, the page starts near y=420 of 900 and the frame's bottom is 62px below the fold. The outer panel scrolls (88px and 171px) around the PDF's own scrollbar, and when it scrolls the page actions ride over the title band (`.panel-actions` has z-index 5). Causes: the action target is an unstyled block (`CockpitFrame.tsx:220`) so two portalled children stack; `min-height: 65vh` (`document-workspace.css:32`); a permanent help line. Fix: one header row (Back, title, actions), a meta line, then a viewer that fills the remaining height with no outer scroll. Command: /impeccable layout.
2. **[P1] The toolbar is the browser's.** Edge shows Draw, Highlight, Erase, Add text, Read aloud, Save; Chromium shows a dark bar titled "file" (the URL's last segment) with the thumbnail column open, which leaves the page about 190px wide in the laptop pane. Markup made there is not stored by SiteWise and is discarded when the frame unmounts, including on a switch to Filing details. The browser's Save duplicates the header's Download. Verified: `#toolbar=0&navpanes=0` removes both bar and column in Chromium 153 and Edge 157 and the two then render alike; Ctrl+wheel zoom still works; `#page=N` opens at a page in both. Fix now: hide the bar, put Print and Open in new tab in a three-dot menu. Later decision: own the renderer (PDF.js) for a Trace toolbar, passage-level provenance and kept markup; that is a dependency with its own speed budget. Command: /impeccable shape.
3. **[P1] It uses none of the workbench's parts.** Close cross instead of `BackButton` (RFP, price table); pills instead of the tab or toggle grammar; mono-caps two-column cells instead of the Profile's label | value | origin rows under a section band; a 16px/700 caps title in secondary ink; the nav still highlights the previous surface with no cue that this is a drill-in. Fix: adopt the existing components. Command: /impeccable polish.
4. **[P2] Checking a filing means leaving the document.** The two views are exclusive; the switch unmounts the iframe (`DocumentWorkspace.tsx:39-91`), so the PDF reloads at page 1. Filing details then shows page thumbnails of the document just left, plus an Open original PDF link to a new tab. The to-check count is not visible from the Document view. Fix: a Filing column beside the page (count on its toggle; suggestions link to their page); below about 880px of pane width it replaces the page as today. Minimum: keep the frame mounted and show the count. Command: /impeccable shape.
5. **[P2] States and small craft.** Esc does not close (tested). No loading state for the frame. "If the PDF does not display" is always shown. The Markdown error asks the user to close and reopen instead of offering Retry. Word and Excel open on a near-empty Document view. The open register row is marked only by the hover grey. `VisualIdentityReview.tsx:70` is an unstyled default link. Command: /impeccable harden.

## Persona red flags

- **PM checking a price against a proposal (project-specific):** opens the letter from the price table and it replaces the table; after finding the fee on page 3, a look at Filing details and back reloads the PDF at page 1.
- **Alex (power user):** Enter opens, Esc does not close; no next or previous document; no shortcut between views.
- **Sam (keyboard and screen reader):** opening announces nothing and leaves focus on the register row; the document frame is the fifth stop in the workspace.

## Minor observations

- The invoice panel embeds its PDF the same way (`Invoices.tsx:252`); one shared frame should carry whatever is decided.
- The file is served whole on every open: `Cache-Control: no-store`, no `Content-Length`, range requests ignored (checked on the dev server). Harmless for a 1.2 MB letter; worth a separate lane C look for large drawing sets.
- At 390px the view stays on the register after opening; the document is above, off screen.
- The viewer's backdrop is the browser's dark grey in both themes; only an owned renderer can change it.

## Questions to consider

- Hide the browser toolbar now, keep it, or go straight to an owned viewer?
- Filing beside the page, or two views restyled as tabs?
- Should a document opened from the price table open beside the table rather than replace it, as the invoice panel already does for an invoice?
