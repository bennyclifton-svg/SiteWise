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
shell; `trace-profile.css`, `trace-procurement.css` and `trace-pages.css`
hold the surfaces. They load after the surface stylesheets.

- **Table and sheet.** The page is the table: warm grey with a barely-there
  sandstone grain behind the navigation. The working view and the document
  register share one white sheet; surface cards lose their frames on it.
- **Butter paper.** A white band of butter paper, lighter than the sheet,
  holds the title, the identity line (client · lot · area · address) and the
  page actions. Its bottom edge is torn with a soft fibrous band. Content that
  scrolls up passes under it, blurred and faded, so it never competes with
  the title. Wide screens only; below 1101px the shell stacks as before.
- **Three hand marks, used sparingly, one meaning each.**
  Pencil: provenance underlines in the profile summary (ember from documents,
  dotted blue for Jev, graphite for the user's word), the title-block rule,
  the open Cost Plan tab and the awarded firm. Squared highlighter: a value
  that needs a decision (sources differ, a proposal without a package). Paint
  stroke: the selected navigation item, fainter on hover.
- **Marks are static files** in `src/assets/marks`, made once by
  `scripts/make-marks.mjs` and applied by CSS (masks and backgrounds). No
  runtime drawing, no Jev, no layout cost. Marks turn off for
  `prefers-contrast: more` and print.
- **Profile.** A one-sentence summary built in code, a Sources note beside
  it, the Project as a five-column title block whose choices are dropdowns
  ("Unknown" when unset), and scope as a matrix with one cell per system in
  building order; the open entry's table sits beneath its row.
- **Logo.** The datum lockup in `Wordmark.tsx`, in the current text colour;
  the circle alone is the favicon (`public/brand/sitewise-datum.svg`).

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
