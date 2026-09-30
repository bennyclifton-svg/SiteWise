# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Project and design managers on Australian construction projects, working at a
desk on a laptop or large monitor. Their recurring job is filing consultant
packages (drawings, reports, specifications, schedules) into the right project
in batches, then trusting that each file's identity (number, revision, title,
kind, discipline, lifecycle area, what it supersedes) is right.

## Product Purpose

SiteWise is a construction-project app built from first principles on the
physical building. It replaces the old sitewise.au app, which nobody used and
which was too slow. The first slice is instant intake: drop a PDF, DOCX or XLSX
into a chosen project and it is filed within about a second (gate: p50 ≤ 1 s,
p90 ≤ 2 s after upload). Success is a filing the user can trust at a glance and
correct in one move.

## Positioning

Jev (TypeSafe System One) is the only AI, and it only picks. Code parses,
harvests candidates, computes, orders revisions and controls flow. Every field
states how it was decided (read by rule, picked by Jev, or set by the user) and
how sure that decision is. Uncertainty is shown, never hidden, and a user value
is never overwritten.

## Operating Context

A Go single binary serves the API, event stream (SSE) and embedded React SPA
from one host with PostgreSQL 17 and content-addressed files. Access is
invite-only per organisation; every project belongs to one org. Documents are
the working material: consultant drawings with title blocks, reports, specs,
spreadsheets. Scanned PDFs without a text layer are stored but not filed.

## Capabilities and Constraints

- Speed is the first objective. Every user-facing path has a p50/p90 budget and
  a benchmark that fails the build.
- One Jev fan-out per filing; nothing judged while the user waits if it can be
  precomputed. Grey means Jev did not answer in time; rule values stand and the
  rest stays unjudged.
- Bands per field: green (above that question's threshold), amber (applied,
  flagged), blank (below band, not applied), grey (Jev timed out).
- Thresholds are uncalibrated at launch, so Jev answers stay blank until the
  calibration run sets them. The UI must make that honest, not alarming.
- Supersession is a link to an immutable prior filing; nothing is deleted or
  overwritten.
- Stack changes need a stated reason (see AGENTS.md).

## Brand Commitments

- Keep the sitewise.au product look (confirmed by the owner, 30 Sep 2026): the
  blue S mark and "Sitewise" wordmark, the warm paper workbench, tight radii and
  IBM Plex type recorded in `../clerk/DESIGN.md` (product/operate register, not
  the landing).
- Evidence vs inference: ember marks evidenced or user-owned values, azure marks
  model inference. Do not swap those meanings.
- Voice: ordinary Australian built-environment professional; plain, specific,
  no AI theatre.

## Evidence on Hand

- Logo files: `../clerk/Landing/logo light.png`, `../clerk/Landing/logo dark.png`.
- Identity fixtures for demos and tests: `testdata/identity/`.
- No customer proof, benchmarks or testimonials exist; none may be invented.

## Product Principles

1. Fast is the feature. A filing that arrives in a second earns trust a slow
   perfect one never gets.
2. Show how each value was decided. Provenance before polish.
3. Uncertainty is information. Amber, blank and grey are honest states, not
   errors.
4. The user's word is final. Corrections win over any later judgement.
5. The building is the model; disciplines are a filing view.

## Accessibility & Inclusion

Colour never carries meaning alone: every band has text and an icon. Full
keyboard operation with visible focus, and focus kept stable when live updates
arrive. Formal WCAG level not yet set (assumed 2.2 AA as the working target).
