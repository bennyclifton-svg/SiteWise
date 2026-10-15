# Specification workspace

Date: 14 October 2026. Status: implemented surface record; final finish review **ship**.

## Overview

Specifications opens directly to an organised, editable six-column table. A
project owner, manager or builder can start with a building-section prompt,
add a line, enter known requirements, or review statements from filed documents.
The route is prompts or documents, then review or authoring, then an immutable
issue that keeps its references and unresolved decisions.

This extends the existing SiteWise workbench. The approved composition is in
`tmp/specification-wireframe-desktop.png` and
`tmp/specification-wireframe-mobile.png`; the later owner-approved shell and
style adaptation governs its presentation. [The coordinated workspace
plan](../plans/2026-10-14-coordinated-specification-workspace.md) records the
ownership contract. [PRODUCT.md](../../PRODUCT.md), the
[foundation design](2026-09-29-foundation-design.md) and
[web/DESIGN.md](../../web/DESIGN.md) remain the product, foundation and shared
visual authorities. This surface record introduces no separate identity or
duplicate root design system.

The shared Profiler building catalogue supplies section IDs, labels and order.
Profiler work supplies scope, location, action and provider by read-through;
Specifications owns detailed materials, products, finishes and requirements.
Starter prompts neither establish project scope nor select a product. External
consultant documents retain authority until the user explicitly changes
coverage and adopts maintenance with a reason.

Implementation evidence is `web/src/SpecificationSchedule.tsx`,
`SpecificationCoverage.tsx`, `Specifications.tsx`, `specificationApi.ts` and
`specifications.css`, with the narrow wide-workspace integration in
`CockpitFrame.tsx`. The table is the landing surface; the existing detailed
authoring and issue components are embedded where needed, with their library
hidden.

Current capture set:

- `tmp/specification-framework-desktop.png` and
  `tmp/specification-framework-mobile.png`: section framework, starter prompts,
  table actions and the current shell.
- `tmp/specification-filled-desktop.png` and
  `tmp/specification-filled-mobile.png`: authored rows in that framework.
- `tmp/specifications-mobile-form.png`: native item-authoring form evidence.

These use synthetic data, not customer proof or performance measurements. The
first finish review returned **fix** for mobile navigation and stale surface
documentation. Mobile navigation has been repaired and the framework and filled
captures refreshed; this record replaces the obsolete library/reader brief.
The final independent reviewer returned **ship**: mobile navigation and documentation findings are both resolved. The earlier brief's **ship**
verdict does not apply to this replacement surface.

## Colors

The table consumes the current Clerk shell's semantic roles through the shared
workbench aliases. Paper, canvas and ground separate working surfaces; ink,
body ink and muted ink establish reading hierarchy; line and strong line divide
rows and sections. Links and lightweight actions use the existing link role.
Source problems use the shared alert role. Light and dark appearance belongs
to the shell, not a specification palette.

Coverage, ownership and review status are written in words: Needs attention,
Managed here, Covered by document, Partially covered and Not applicable.
Proposed readings, unknown values and source-review warnings remain visible
independently of colour.

## Typography

The current shell uses Instrument Sans through `--sans`; identifiers retain
the shared `--mono` stack. Do not restore the superseded IBM Plex body
description from the earlier brief or freeze shared token values here while
the global visual record is being reconciled.

The specification heading is a modest title above compact table text. Section
names are stronger than row metadata; item names, values and locations carry
the reading hierarchy. Codes are monospaced. Supporting source and scope text
is smaller and muted. Long values and filenames wrap, and prose in expanded
details preserves line breaks. All controls inherit the current shell fonts.

## Layout

Specifications sits immediately after Procurement in project navigation. On
desktop the existing left navigation and project header frame the working
table. The document rail is collapsed initially to leave room for comparison;
**Show documents** opens the existing repository and **Hide documents** returns
that width to the workspace.

The first viewport contains navigation, the specification title and actions,
and the start of the table. The header actions are **Update specification**,
**Import**, **Preview / issue** and **Add item**. A compact tools row shows item
and review counts, Expand all, Collapse all and Open Project Profile.

The six columns are **Code**, **Item / work**, **Type / specification**,
**Location / requirements**, **Supplier** and **Reference**. Collapsible
building sections span all columns and show item/prompt counts and coverage.
An item's Details opens directly beneath its row; section coverage opens
beneath its section heading. A section's final Add item action opens a quick
row. Full item creation, import and preview use panels above the table.

At 760px and below, the specification route uses a compact logo/project-switcher
row followed by one horizontally scrolling navigation row. The navigation
spacer and chat footer are hidden on this route so the table remains reachable.
Header actions wrap and coverage forms become one column. The six-column table
keeps its 870px minimum width inside a horizontal scroll region; it does not
turn into stacked cards or force the whole page wider. Coarse-pointer controls
receive larger targets and native entry fields use 16px text.

Opening the embedded item form, item details or preview focuses its heading;
on mobile it also scrolls that content into view. Inline edits focus their
textarea and return focus to the originating cell on save or cancel. Ordinary
refreshes do not deliberately move the active heading.

## Elevation & Depth

The table is a ruled work surface inside the incumbent shell. Section tones,
row borders and the expanded-detail background establish hierarchy. Panels
and controls reuse shared surface and border treatments; Specifications adds
no shadow system.

## Shapes

The table and panels reuse the shared small-radius token; primary and neutral
buttons inherit the shell's control shape. Inline cells are lightly bounded
on hover and become native textarea forms during editing. Section expanders
use a small SVG chevron. These are local interaction patterns, not new global
tokens or logo rules.

## Components

### Framework, quick rows and inline cells

An empty project still receives the building sections and starter prompts.
**Specify [item]** creates an item from a prompt. **Add item** inside a section
asks for Code, Item name and Kind, with Add line item and Cancel actions.
The full item form additionally supports a building system, description,
scheme and alternative group. Items outside the recognised hierarchy appear
under **Items to classify**.

Code and item name are editable in place. Locally managed material/product,
finish, colour, performance requirement and supplier fields support inline
requirements. Enter saves, Shift+Enter inserts a line break and Escape cancels;
Save and Cancel are also visible. Clearing a value records unknown. Source
readings, externally maintained values and values needing richer review are
read-only in these cells and lead to Details. An active inline edit disables
competing table actions so it is not silently discarded.

### Scope and detailed authoring

Rows show application locations and their choice state. Linked applications
display work action and inclusion from Profiler. Where no application exists,
matching Profiler scope can supply location, action, inclusion and provider for
context. **Open Project Profile** is the route for changing work scope and
responsibility.

Details embeds identity editing, locations, assertions, source evidence,
decision history and related items. A location can link to an existing Profiler
work item; its linked location, work action and inclusion remain read-through.
An unlinked location is an application only, with project scope unconfirmed.

Detailed values retain their meaning: selection, requirement, capability,
allowance, exclusion or observed condition. Known, unknown, deferred, not
applicable and not required are distinct states. Measurement qualifier, unit,
basis, test method and datum remain explicit fields. Native textareas support
descriptions and notes with line breaks; this is not a formatted rich-text
editor. Relationships retain the existing finish, component, dependency,
matching and replacement forms.

### Coverage and consultant authority

Section and item coverage records state extent, gaps, maintainer, reviewer
and referenced document locations. Confirm coverage and revisions pins the
reference metadata through the existing API. Changing an existing coverage mode
requires a reason. References show their revision and flag unavailable or
changed sources for review; uploading a newer file does not appoint it as the
new authority.

Externally maintained items say so in their row and Details. A confirmed
document reading remains a source statement; it does not select a product or
transfer maintenance. Local decisions retained for coordination are excluded
from issued requirements while the item remains external. **Manage this item
here** requires a reason, the current item version and resolution of any
governing external coverage before transfer. Source readings and history remain
attached after adoption.

### Jev update and source review

**Update specification** queues current project documents in the existing
background worker, reuses unchanged readings and announces whether work was
queued. **Import** reads one selected filed document. Document readings lists
completion, pending work and errors; queued or leased imports refresh every
three seconds. Both paths produce candidates for review, not accepted choices.

Code harvests candidates and controls flow. Jev is the only AI and picks through
the bounded [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [one-fan-out](https://docs.typesafe.ai/patterns/fan-out) patterns recorded in
the coordinated plan. Navigation, typing, direct edits, review and issue do not
call Jev. No specification image generation, prose generation or chat workflow
is implemented.

Proposed item identity can be confirmed or its readings linked to an existing
item. Source review exposes page, excerpt and revision markings. **Confirm
reading** requires the source-check acknowledgement; replacing an accepted
value also requires a reason. Stale, unavailable and withdrawn sources cannot
be accepted. The request carries the assertion version and expected prior
decision, keeping review tied to the state the user saw.

### Preview and immutable issues

**Preview / issue** shows the assembled draft, consultant references and
coverage, accepted values and unresolved decisions. Export draft CSV uses the
current draft. An issue records its title and preliminary, coordination or
pricing purpose, and uses the current snapshot revision. A schedule with rows
or coverage can be issued.

Issues freeze local rows, scope and reference metadata. Earlier PDFs and CSVs
remain available in Issued schedules; later edits or document revisions do not
rewrite them. The PDF uses the existing document renderer and remains a basic
text schedule. The browser's table composition is implemented; a designed PDF
table, image/crop evidence and automatic application assignment are not.

### Loading, saving and limits

Loading and background progress use status text. Load failures offer **Reload
specifications**, retain entries and block mutations until recovery. Save
errors remain visible; successful writes refresh the view and announce the
save. Conflicts refresh current data, and a stale full-item editor requires
cancelling and reopening. Signed-out responses use the existing sign-out
handler. Busy states disable affected actions.

This is lane A. Specification reads, writes, issue and export retain their
recorded p50 <= 100 ms / p90 <= 250 ms budgets; inline cell, coverage and update
requests use the write budget. Background reading completion is separate from
request latency. Filing retains p50 <= 1000 ms / p90 <= 2000 ms. This document
does not replace the measured evidence recorded in the implementation report. No new runtime
dependency or AI provider is introduced by the surface.

## Do's and Don'ts

- **Do** preserve the approved table composition, shared building hierarchy
  and current shell authority.
- **Do** keep Profiler scope/action/provider distinct from detailed
  specification requirements and product choices.
- **Do** keep proposal status, unknowns, source revisions, coverage and
  maintenance responsibility visible before issuing.
- **Do** preserve compact mobile navigation, table scrolling, inline keyboard
  actions and heading focus when changing this surface.
- **Don't** infer coverage, product selection or maintenance transfer from an
  upload or confirmed source reading.
- **Don't** treat starter prompts as scope, source capabilities as project
  requirements, or a later document as permission to overwrite an issue.
- **Don't** canonise the superseded library landing layout, historical IBM
  Plex body styling, synthetic captures or a pending finish review as the
  current product contract.
