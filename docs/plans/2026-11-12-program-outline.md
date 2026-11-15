# Program opens as an outline

Date: 12 November 2026. Lane A (program dates are figures a user trusts).
Supersedes the "every work item brings its trade activities" part of
`docs/plans/2026-10-18-program.md` 7.2 for the first program a user sees.

## Why

Opening Program on Merricks (NCC class not yet on the Profile) built 205
rows. 182 were "If NCC building classification(s): …" copies, one per class
the library knows, each 0 days with a range. Every design stage collapsed
onto the start date and the user would have deleted the lot. With the class
known it was still 84 rows, 56 of them trade activities in Delivery.

## Owner direction (12 November 2026)

1. NCC class unknown: a generic set of activities that ignores the class.
2. A first program is 20 to 30 rows, stages included.
3. The Update program button stays visible.
4. The program opens zoomed to its full extent, bars filling the width.

The aim is 80 percent right with a logical stage flow the user edits.

## What changes

- `GenInput.Outline` (the store always sets it) builds the outline:
  stages plus the project activities. No work-item trade rows, no
  investigation rows, no hold points, and no conditional "If …" rows: a
  project activity whose condition is unknown is left out.
- `fallback_for` on a class-free fragment: it is included when none of the
  class-specific fragments it lists holds outright (class unknown, or a
  class the library has no variant for). Its activities read the Class 1a
  library lengths and say so as an assumption.
- `outline: false` on an activity drops it from the outline: the
  coordinate-consultants rows that run alongside the documentation, the two
  defects inspections, tracing existing services, the client sign-off for
  lodgement, and establishing the site (inside the Construction bar). A new
  build opens at 28 rows, an extension or refurbishment at 30.
- `outline_only` fragments: the class-free fallbacks, one Construction bar,
  and the for-construction issue, so Design finalisation is not empty.
- `stage_fixed` keeps the for-construction issue in Design finalisation when
  a traditional skeleton moves the rest of that stage into Design
  development.
- Construction length, first match: the user's value, an accepted reading,
  the Profile's construction duration (weeks), else the span of the
  library's own trade chain for the included work, computed by the engine.
  With the class unknown or uncovered the chain is a Class 1a house's,
  marked assumed. Nothing new is invented: every length is an existing
  reviewed library row or a computation over them.
- The detailed generation (`Outline` false) is kept: it computes the
  Construction length and stays tested for a later "break Delivery into
  trades" action.
- Web: on first open the chart fits the program's full extent to the width.

## Update program applies at once (owner, same day)

Clicking Update program is the user committing to every update, so nothing
is asked: no review panel, no Adopt or Keep. `POST /program/update`
regenerates and applies the whole difference in one version-pinned batch
(one undo), in up to three rounds so links between rows added together
land. Chain links an add placed between two library rows (no library key)
are retired, because the plan's keyed links carry the order and such a link
blocked a library link as a loop. The button still shows the pending count
from the background review. Accept proposed values now only offers readings
from a dropped program file.

## Not in scope

Toolbar changes beyond the fit (owner kept Update program), the trade
breakdown action, re-reading existing programs. An existing program sees
the outline through Update program (review lists the removals and adds).

## Budget

Program prepare keeps its recorded budget in `bench/budgets.json`. The
outline adds one detailed generation and one engine pass in memory.

## Fragments

### fr.concept-design-generic

Brief, concept and client review for any class; Class 1a lengths assumed.

### fr.design-development-generic

Schematic design for any class; Class 1a length assumed.

### fr.approval-docs-generic

Approval documentation and sign-off for any class; Class 1a lengths assumed.

### fr.design-development-dc-generic

Design for the contractor's tender on a contractor-led route, any class.

### fr.construction-docs-generic

Construction documentation and sign-off for any class; Class 1a lengths.

### fr.tender-docs-generic

### fr.tender-period-generic

### fr.tender-evaluation-generic

### fr.construction

One bar for the building works. Length from the Profile's construction
duration, else the library trade chain (see What changes).

### fr.for-construction-issue

Issue the documents for construction, in Design finalisation. No library
length yet: the default activity length, marked assumed.
