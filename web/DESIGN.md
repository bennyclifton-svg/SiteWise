---
name: SiteWise workbench
description: Warm paper, compact controls and inspectable provenance for construction project work.
colors:
  paper: "oklch(0.993 0.003 92)"
  canvas: "oklch(0.977 0.004 92)"
  ground: "oklch(0.955 0.005 92)"
  header: "#fefcf8"
  line: "oklch(0.908 0.006 92)"
  line-strong: "oklch(0.852 0.007 92)"
  frame: "oklch(0.33 0.007 86)"
  ink: "oklch(0.196 0.006 86)"
  ink-body: "oklch(0.278 0.007 86)"
  ink-muted: "oklch(0.47 0.008 90)"
  ember: "oklch(0.55 0.12 52)"
  ember-hot: "oklch(0.48 0.11 51.5)"
  ember-soft: "oklch(0.972 0.012 54)"
  azure: "oklch(0.5 0.12 245)"
  azure-soft: "oklch(0.965 0.014 245)"
  ok: "oklch(0.47 0.1 150)"
  warn: "oklch(0.47 0.1 65)"
  warn-soft: "oklch(0.962 0.035 82)"
  alert: "oklch(0.5 0.15 28)"
  alert-soft: "oklch(0.965 0.02 28)"
  citation-evidence: "#185c96"
  citation-user: "#14665f"
  citation-calculation: "#694091"
  citation-assumption: "#79510e"
typography:
  body:
    fontFamily: '"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif'
    fontSize: "13.5px"
    lineHeight: 1.5
  title:
    fontFamily: '"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif'
    fontSize: "22px"
    fontWeight: 600
  field-title:
    fontFamily: '"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif'
    fontSize: "14px"
  metadata:
    fontFamily: '"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif'
    fontSize: "12px"
  identifier:
    fontFamily: '"IBM Plex Mono", ui-monospace, "Cascadia Mono", monospace'
    fontSize: "11px"
rounded:
  sm: "3px"
  control: "0.4375rem"
  pill: "999px"
spacing:
  compact: "6px"
  control-gap: "8px"
  field-gap: "10px"
  group-gap: "12px"
  section-inset: "16px"
  panel-inset: "20px"
  page-inset: "24px"
components:
  button:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    height: "36px"
    padding: "0 14px"
  button-primary:
    backgroundColor: "{colors.ember}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    height: "36px"
    padding: "0 14px"
  button-primary-hover:
    backgroundColor: "{colors.ember-hot}"
  input:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    height: "36px"
    padding: "0 10px"
  assertion:
    rounded: "{rounded.pill}"
    padding: "1px 6px"
  title-block:
    backgroundColor: "{colors.paper}"
    rounded: "{rounded.sm}"
---

# Design System: SiteWise workbench

## Overview

**Creative North Star: "The warm paper workbench"**

The owner-confirmed product register is a working surface for construction documents: warm paper, IBM Plex, tight controls and clear rules between records. Keep the blue S mark and Sitewise wordmark. Density serves comparison and correction; decoration does not compete with the work.

Provenance stays beside the value it qualifies. Reports, Works, Packages, Delivery and Proposals extend this incumbent world with readable wording, inspectable sources and visible handling of user edits. This is a source-derived record of the current app, not a redesign or a claim that all accessibility targets have been verified.

**Key Characteristics:**

- Warm tonal surfaces with explicit borders.
- IBM Plex Sans for reading and controls; IBM Plex Mono for identifiers.
- Compact controls and locally inspectable provenance.
- User wording and changed sources remain visibly distinct.

Evidence: `src/styles.css`, `src/reports.css`, `src/Reports.tsx`, `src/Works.tsx`, `src/Packages.tsx`, `src/Delivery.tsx`, `src/Proposals.tsx`, `src/proposalApi.ts`, `src/Project.tsx`, `src/Profile.tsx`, `src/Register.tsx`; brand authority: `../PRODUCT.md`. Frontmatter values preserve source CSS. Named spacing roles describe repeated literal values; they are not new runtime variables.

## Colors

Warm neutrals carry the work; restrained semantic colour identifies ownership, inference, state and report reference type.

### Primary

- **Ember:** primary actions and evidenced or user-owned values; its deeper and softer companions provide hover and selected emphasis.

### Secondary

- **Azure:** inference and selected scope in the filing/profile workbench, with a soft companion for selected surfaces.
- **State colours:** green for accepted states, amber for review or uncertainty, and alert red for failure or conflict. State text explains the meaning.
- **Report references:** blue E / Evidence, teal U / User input, purple C / Calculation and amber A / Assumption. These names and letters are displayed with the colour.

### Neutral

- **Paper, canvas and ground:** the surface hierarchy, from readable sheet to workbench to hover or recessed region.
- **Header:** the logo artwork's matching ground.
- **Ink, body ink and muted ink:** primary text, reading text and supporting metadata.
- **Line, strong line and frame:** internal separators, stronger boundaries and title-block outlines.

**The Scoped Semantics Rule.** Preserve ember/azure ownership meanings in the workbench; the E/U/C/A citation palette belongs specifically to report references.

## Typography

**Body Font:** IBM Plex Sans with the source fallback stack.
**Label/Mono Font:** IBM Plex Mono for document numbers, revisions, dates and derived identifiers.

The hierarchy is compact and practical, without a promotional display face. Body text uses the base role; titles are modest steps above it. Project-list headings use the recorded title weight, while report title weights follow their heading defaults in CSS rather than defining a separate shared weight token.

### Hierarchy

- **Title:** project-list and report titles use the recorded title size; the project header also has a larger local treatment.
- **Field title:** small section and block headings distinguish records without overwhelming values.
- **Body:** the recorded body role carries reading and controls. Report wording and expanded reference prose are limited to 75ch and preserve line breaks.
- **Metadata:** the recorded metadata role supports state, reference labels and notes.
- **Identifier:** compact monospaced numbers, revisions and dates support scanning across register rows.

**The Reading-and-Identity Rule.** Use Sans for prose and controls; use Mono where the content is an identifier or a structured value.

## Layout

The desktop shell has a fixed left navigation column (232px) and a flexible work area. Profile and register occupy the project workspace; the existing project view controls also expose Works, Proposals, Packages, Delivery and Reports. Shared controls commonly use the recorded small gaps, with larger insets separating working regions.

Reports, Works, Packages, Delivery and Proposals share a list at left (210–260px) and a reading/editing area at right. The reader has a maximum width of 1000px; prose and planning forms are constrained to 75ch. Sections use ruled headings and vertically separated blocks. This arrangement describes these bounded workbench surfaces, not a required layout for every future screen.

At 1100px the filing workspace becomes one column. At 760px the main navigation becomes horizontal, profile rows stack, and each Reports, Works, Packages, Delivery or Proposals list precedes its reader. The reader changes to narrower side insets and its toolbar wraps. The older title-block surface has an additional 720px adjustment. Preserve those distinct breakpoint responsibilities. The proposal list alone has bounded vertical scrolling (55vh on desktop, 240px at the 760px breakpoint), keeping the reader reachable beneath the list on narrow screens.

## Elevation & Depth

Depth comes mainly from tonal paper layers and borders. A small shadow supports controls and title blocks; a slightly stronger shadow supports entry panels, source disclosures and confirmation dialogs. These are soft shadows, not hard offset decoration. Exact shadow values and the shared easing curve live in the sidecar.

**The Paper-and-Rule Rule.** Establish grouping with surface tone and separators before adding elevation.

## Shapes

Containers use tight small corners and controls use the slightly softer control radius. Pills already exist for compact assertions, scope origins and source counters; keep them for those small labels rather than enlarging them into panel silhouettes. Lines define document-like sections and register rows.

## Components

### Buttons

Compact, bordered and explicit. Shared buttons use the recorded control height and padding; small actions use a shorter treatment (28px). Neutral buttons darken to ground on hover and line on active. Primary buttons use ember and deepen on hover. Disabled buttons reduce opacity. Shared focus is an ember outline (2px) with an offset (2px); some register controls use azure inset focus instead.

### Chips

Assertions, scope-origin labels and source counts are compact pills with text. Required/allowance assertions use warning colour; user scope origins use solid ember boundaries and default origins use dashed azure boundaries. Do not infer that every pill shares one semantic role.

### Cards / Containers

Title blocks are paper containers with a dark frame, small corners and the smallest shadow. Their headers and internal rows separate metadata from values. Report sections use simple horizontal rules instead of wrapping every paragraph in a card.

### Inputs / Fields

Shared inputs and selects are paper controls with a strong border and small shadow. Focus changes the border and places the ember outline directly against it. Invalid inputs use the alert border. Report editing uses a labelled, full-width, vertically resizable textarea with small corners; it inherits the reading font.

### Navigation

The project view controls wrap across rows as space narrows, reuse small buttons and expose the current state with `aria-pressed` plus a ground surface. Reports, Works, Packages, Delivery and Proposals lists use full-width text buttons, ground hover/selection and heavier selected text; `aria-current` identifies the selected record. Narrow screens stack the list above its reader. Once opened, these project views remain mounted while hidden, retaining local edits when switching views within the open project.

### Report references and protected wording

Each report block keeps its reference disclosures beside its wording. A disclosure shows the citation ID and reference type; opening it reveals source basis text beneath the block. Missing references have explicit text. Material assumptions and review-needed states are surfaced in warm notice regions.

Creation, assembly and refresh are separate labelled actions. Editing reveals the textarea and Save/Cancel actions, while competing report actions are disabled. Saved edits receive a Protected wording label. Concurrent changes offer the latest saved wording for comparison; refresh keeps protected wording and flags changed or missing sources. These are visible interaction patterns in the current component, not a guarantee about backend behaviour.

### Planning corrections and scope

Works and Packages reuse labelled paper fields, native selects, multiline wording and wrapped Save/Cancel actions. Saved values remain above the editor so a correction can be compared in place. Work records expose the latest editor, edit time when present, rationale and original source excerpts; the disclosure does not present a complete correction history.

When a newer saved version is loaded, a warm notice separates it from the local correction. An explicit keep-my-wording/correction action rebases the local edit onto that version. Loading and save failures offer a labelled reload action. Planning status and provisional wording stay in plain text beside the affected record; saving is not presented as verification or appointment. These are observed UI behaviours, not backend guarantees.

### Delivery records and dates

Delivery extends the same saved-record list, ruled reader and explicit editor. The saved record stays above unsaved entries; record selection and adding another record are disabled while editing. Source and latest-correction details sit behind a native disclosure beside the saved values. Missing dates and assignments have explicit text.

Date entry uses labelled native date inputs. Baseline, target, forecast, actual and as-of dates share an initially open Dates disclosure; approval submission and determination have separate fields. Unknown dates remain blank, and clearing a date removes it. Saving records the team's position without presenting an approval as verified or a hold point as released.

A changed saved version receives a warm comparison notice and blocks saving until the user explicitly keeps their edits against that version. If its record type changed or the record disappeared, the notice directs the user to cancel and reopen as appropriate. Loading failures have an alert and a labelled reload action, distinct from the initial loading message and the successfully loaded empty list. Reloads retain the local editor. These are observed UI behaviours, not backend guarantees.

### Proposal review and decisions

Proposals reuse the saved-record list, ruled reader, warm notices and labelled native fields. List rows show kind, state, critical status and draft knowledge in text. The reader keeps the knowledge record, triggering work, determinant values and rule references beside the decision; work evidence and recorded signals expand through native disclosures. Rule references explicitly remain unverified here. Draft knowledge, unaccepted triggers and inputs changed since a decision have separate notices.

Review acceptance opens an explicit planning step. Obligation acceptance requires a chosen package; delivery requirements offer optional package and work assignments. Role and stage choices follow the selected work and package, with no assignment selected by default. Acceptance wording distinguishes planning from appointment, approval, hold-point release and compliance verification. Dismissal offers an optional reason. While reviewing, selection and list-scope changes are disabled; a changed proposal preserves the choices but blocks submission until the user cancels and reviews the saved reason again.

Saved decisions show their actor, time, optional rationale and created record type. Undo opens a confirmation describing the guard against retiring edited or referenced records; refusal is reported in an alert. Loading, empty results, save errors and success messages remain distinct. These are observed frontend interactions and request shapes, not claims that the production evaluator is connected or that backend guards have been verified by this design record.

**The Planning-Decision Rule.** Keep the reason and provisional status visible beside an explicit acceptance or dismissal; a planning decision does not imply verification.

## Do's and Don'ts

### Do:

- **Do** preserve the paper workbench, IBM Plex and established logo assets.
- **Do** keep provenance, uncertainty and protected wording next to the affected value.
- **Do** retain report citation letters and names alongside colour.
- **Do** reuse existing compact controls and inspectable disclosures.

### Don't:

- **Don't** swap the workbench's ember and azure ownership meanings.
- **Don't** spread the report citation palette into unrelated filing states.
- **Don't** hide uncertainty or present an assumption as evidence.
- **Don't** treat this source inspection as verified contrast, keyboard or screen-reader compliance.
