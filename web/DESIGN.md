---
name: SiteWise workbench
description: Instrument Sans cockpit with shared light and dark semantic surfaces and inspectable provenance.
---

# SiteWise workbench design authority

Updated 28 October 2026 from `src/main.tsx`, `src/clerk-colours.css`,
`src/cockpit.css`, and PRODUCT.md. This supersedes the warm-paper/IBM Plex
workbench description. The earlier visual system is historical context, not
an alternative direction for new screens. Preserve the Sitewise mark.

## Trace (owner-approved 3 November 2026)

The workbench is a drawing sheet on a drafting table. `trace.css` holds the
shell; `trace-motion.css` the run-time motion and marks every surface shares
(below); `trace-profile.css`, `trace-procurement.css` and `trace-pages.css`
hold the surfaces. They load after the surface stylesheets.

- **Table and sheet.** The page is the table: warm grey with a barely-there
  sandstone grain behind the navigation. The working view and the document
  register share one white sheet; surface cards lose their frames on it.
- **Butter paper.** A white band of butter paper, lighter than the sheet,
  holds the title, the identity line (client · lot · area · address) and the
  page actions. Its bottom edge is torn with a soft fibrous band. Content that
  scrolls up passes under it, blurred and faded, so it never competes with
  the title. Wide screens only; below 1101px the shell stacks as before.
- **Four hand marks, used sparingly, one meaning each.**
  Pencil: provenance underlines in the profile summary, by the origin rule
  (owner, 10 Nov 2026: green from the documents, dotted where Jev leans on
  them; dotted blue typical; rust the user's word), the title-block rule,
  the open Cost Plan tab and the awarded firm. Squared highlighter: a value
  that needs a decision (sources differ, a proposal without a package). Paint
  stroke: the selected navigation item, fainter on hover; a dry brush, solid
  to 68% of its length with streaks throughout, feathered out by 81%. Pen:
  the ground-line button (below).
- **Revision triangle** (owner, 10 Nov 2026). A drawing's revision mark
  beside a value or scope entry an update changed, until the user clears it;
  its tooltip says what it was and what it is. One of the shared run-time
  marks (Run-time motion and marks, below).
- **Marks are static files** in `src/assets/marks`, made once by
  `scripts/make-marks.mjs` and applied by CSS (masks and backgrounds). No
  runtime drawing, no Jev, no layout cost. Marks turn off for
  `prefers-contrast: more` and print.
- **Profile.** A one-sentence summary built in code, a Sources note beside
  it, the Project as a title block whose choices are dropdowns ("Unknown"
  when unset), one line per value (label, value, then how it was decided;
  bedrooms, bathrooms and car parks share one Counts row, as on RFP page
  one), and scope as a matrix with one cell per system in building order;
  the open entry's table sits beneath its row. Update and its menu sit in the
  title band; the strip at the foot rests as one line (owner, 10 Nov 2026).
  Each row's source dot is solid: green from the documents whether they
  agree or Jev leans on them (owner, 11 Nov 2026), blue typical, a rust
  square yours, amber to check. The summary's pencil keeps its dotted green
  where Jev leans.
- **Logo.** The datum lockup in `Wordmark.tsx`, Section version (owner pick
  8 Nov 2026): the structure above the level line is solid, the slipped ground
  below it is drawn in 45° section hatch. In the current text colour. The
  favicon (`public/brand/sitewise-datum.svg`) keeps the plain half-tint,
  which reads better at 16px.
- **Ground-line button** (owner pick 8 Nov 2026). Page-level actions only:
  Update (`UpdateButton`) and back out of a drilled-in view (`BackButton`),
  both `GroundButton.tsx`. A pen-drawn box whose corners cross, the label
  standing on a ground line; the primary button has section hatch below the
  line, full to half its width, gone by 90%. The back button keeps the line
  only. Busy work over 400 ms "sketches": the refresh circle is pencilled
  round while the hatch is shaded in, on a 2.4 s beat, and finishes its cycle
  before settling. Table, menu and inline buttons stay plain `.btn`.

- **Projects and Account** (11 Nov 2026). Outside a project the same grammar
  holds: `TracePage.tsx` and `trace-home.css` draw the table, one sheet and a
  butter-paper band (`TraceBand`: title, one line, page actions) that the
  content scrolls under. The account menu (initials) replaces the Account and
  Sign out pills, opening downward. Projects names and creates in one step in
  the band (ground-line "Create project"), then lists project | documents |
  to check | last filing, sorted by the latest filing; rename, archive and
  delete sit in one row menu. Archive keeps everything; delete stays
  confirmed.

## Run-time motion and marks (shared, 11 Nov 2026)

Any surface shows work as it happens with one vocabulary: the classes and
tokens in `trace-motion.css`, played by `play()` and `inkInOrder()` in
`ink.ts`, and the marks in `RunMarks.tsx`. Profile's update is the first user
(`docs/design/2026-11-10-run-time-feedback.md`).

- **Only a real event moves anything:** a count the server made, a write that
  saved, a passage read. Never a timer standing in for progress, never an
  ETA, never decoration. The value is on the page before its motion plays;
  motion never delays or hides it, and nothing waits for it to finish.
- **One meaning per motion.**

  | Meaning | Use | What it does |
  | --- | --- | --- |
  | Spotted: a passage just spoke to this | `ink-glint` | cyan wash that fades, 800 ms; a table row washes its cells |
  | Inked: a value written | `ink-write` | wipes in left to right, 360 ms; also a text value now to check |
  | Inked: words replaced | `data-ink-was="old words"` | the old words struck above the line, 1.2 s |
  | Inked: a mark or count changed | `ink-pop` | pops in, 320 ms |
  | Inked: a block in or out | `ink-on` / `ink-off` | section hatch, then `--ink-fill`; or rubbed out |
  | Now needs a decision | `ink-check` | a small element (cell, dot) bumps once |
  | Held: the user's word stands | `ink-held` | nothing moves; the row warms in rust, 900 ms |
  | Pointing: a summary shows where | `ink-flash` | the check amber, outlined twice, 1.5 s |
  | Reading now (a state) | `ink-reading`, `ink-ring` | the activity ring while the work runs |

- **Marks stay until the user clears them.** The revision triangle
  (`RevisionMark`, `.rev-mark`) beside what an update changed; "+n" in cyan
  (`PendingCount`, `.pend-count`) for readings not yet applied; a count on a
  nav item (`.nav-count`) for what a run left in another view, until it is
  opened.
- **Tokens:** `--ink-ease` (exponential ease-out), `--ink-spotted` (cyan is
  work in progress), `--ink-fill`, `--rev-mark`. `ink.ts` holds the same
  durations and takes each class off when its motion ends.
- **Order and volume.** `inkInOrder()` plays a change's items in page order
  within 900 ms (all at once over 40) and skips what is out of view, which
  keeps its mark. Check `motionAllowed()`: no motion in a hidden tab or with
  reduced motion.
- **Fallbacks.** Reduced motion keeps every mark and value and gives a
  pointed item a plain outline; `prefers-contrast: more` draws the triangle
  in the text colour; print hides marks and rings.
- **Specialise, don't redefine.** A surface may adapt a motion for its own
  element in its own sheet, loaded after this one: Profile's filled scope
  cells take a cyan ring instead of the wash.
- **Where it fits next:** wherever a background reading lands while the user
  watches it. A proposal read onto the price table, an invoice read into the
  ledger, Program changes arriving for review, a specification import.
  Filing's title-block landing (`styles.css`, `ink-in`) means the same as
  Inked and predates this sheet; move it onto `ink-write` when filing is next
  changed.

## Runtime source of truth

`clerk-colours.css` owns primitive colours and the light/dark semantic roles.
`cockpit.css` maps those roles onto the workbench variables and owns shell
layout. `styles.css` provides shared controls and provenance; surface CSS
extends these without creating another theme. Fonts are self-hosted.

| Workbench role | Runtime semantic token |
| --- | --- |
| paper / header | --sw-surface |
| canvas | --sw-canvas |
| ground | --sw-surface-inset |
| ink | --sw-text-primary |
| ink-body | --sw-text-secondary |
| ink-muted | --sw-text-tertiary |
| line | --sw-border-default |
| line-strong / frame | --sw-border-strong |
| azure / azure-soft | --sw-link / --sw-selection-bg |
| success | --sw-success-text |
| warning | --sw-warning-text / --sw-warning-bg |
| alert | --sw-error-text / --sw-error-bg |
| focus | --sw-action-focus |

Use semantic variables rather than copying theme-specific hex values. Keep
ember for evidenced/user-owned filing values and azure for model inference.
Navigation uses a neutral hover surface with a heavier selected label.
Status always includes wording, icons or a native selected state. Report
E/U/C/A citation colours remain local to their existing meaning.

## Typography and controls

Instrument Sans is the shell, form and reading face, with ui-sans-serif and
system-ui fallbacks. IBM Plex Mono is retained for identifiers and tabular
measurements. Use tabular figures for money. Compact metadata is normally
12px, table/control text 13-14px, normal shell text 16px. Headings follow the
existing surface hierarchy rather than imposing a single size on reports,
financial grids and the project title. Existing print typography stays local.

Buttons are 36px tall; compact actions retain their existing smaller variant.
Primary controls use the current neutral inverse theme treatment. Do not
recolour them ember without a product decision. Inputs have explicit labels;
provenance belongs beside the field and must not collide with its label.
The shell radius is 1.25rem, shared control radius 0.875rem and small radius
8px. Preserve the incumbent pill-shaped buttons and restrained surface depth.
Focus must remain visible in both themes; skip navigation becomes visible
when focused. Do not hide important actions behind hover-only interaction.

## Layout and responsive behavior

Desktop: navigation up to 236px, a flexible workspace, and a resizable document
rail (280-920px, constrained to available space). Header and action rows sit
above their respective panels. The document rail can be hidden explicitly;
workbench navigation preserves mounted view state and local drafts.

At 1100px and below, the shell stacks. A labelled Project navigation disclosure
keeps navigation available without pushing the work offscreen; the project
heading stays visible. Opening the disclosure exposes normal keyboard controls.
Tables scroll inside their workspace; never widen the page to fit their minimum
column widths. Keep scrollbars discoverable.

A field inside a table cell is the cell (owner, 8 Nov 2026): no border, no
rounded box, no second frame inside the table's own lines. Hover shades the
cell; focus draws one line. `sizing.css` sets this globally for every
`td`/`th` field, so a new table needs no field styling of its own; never add a
bordered input inside a table.

Row and cell states look the same in every table (owner, 11 Nov 2026,
"Neutral grey"), set once in `src/states.css`: hover a faint grey; selected a
grey band with a mid-grey line on the row's left edge; editing the whole cell
with a 1px focus line; to check an amber line on the cell's left edge, no
fill; errors in the error colour only. Selection never uses an origin colour
(green, blue, rust, amber). A row marked `aria-selected`, a cell marked
`data-band="red"` and the shared `InlineCellEditor` get these looks with no
rules of their own; change a state there, never in a surface sheet.

Profile fields respond to the workspace container, not just the viewport.
At 560px of pane width and below, identity, place and detail grids become one
column. Labels and provenance can wrap independently; inputs take full width.
Selected class/work choices have a check in addition to colour.

The project switcher is a searchable disclosure with normal buttons, not an
ARIA listbox. Its result list scrolls within the nav; All projects remains
outside that scrolling list. Project search filters the already-loaded names
without additional server calls. Deletion remains explicitly confirmed.

## Honest states

No inactive chat composer or empty Chats section. Project Plan remains visibly
planned and disabled. Missing document titles display the filename plus Needs
title review; that fallback is never written as an accepted title. Suggestions
remain separate until the user accepts them. Unknown prices, specification
values and dates stay unknown. Unverified rules stay unverified.

Load errors must differ from loading and empty states and offer an actionable
retry. Save errors preserve the user's draft. Model source counts are labelled
as source passages. Diagnostics only appear after a dev endpoint is confirmed.
A signed-out user loses the authenticated view and its active event stream.

## Verification limits

Source alignment is not accessibility certification. Exercise desktop and
390px mobile layouts, long names, keyboard focus, both themes, unknown values,
network failures and destructive-action cancellation when changing the shell.
Production readiness also requires the independent gates in docs/operations.md.
