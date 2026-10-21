# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Project and design managers, and builders, on Australian construction projects,
working at a desk on a laptop or large monitor. They file consultant packages,
read the building, set a cost plan, appoint consultants and trades, and keep a
specification they can issue. Filing stays a core job: each file's identity
(number, revision, title, kind, discipline, lifecycle area, what it supersedes)
has to be right.

## Product Purpose

SiteWise is the workbench for one building: what it is, what it is expected to
cost, who is appointed, and what is specified, with the document repository
beside that work. It replaces the old sitewise.au app, which nobody used and
which was too slow.

A dropped PDF, DOCX, XLSX or Markdown (.md or .markdown) is filed within about a second (p50 ≤ 1 s, p90 ≤ 2 s
after upload). Saved reads and writes on the cost plan, procurement and
specifications stay within p50 ≤ 100 ms and p90 ≤ 250 ms, except where a named
path is already tighter. Success is a project a manager can read and decide
from, and a filing they can trust at a glance and correct in one move.

PDF filing also queues an image-assisted identity review. The expanded document
row shows literal suggestions for title, number and revision beside source
images, with enlargement and explicit Use actions. Drawings use overlapping
bottom/right regions; reports use the first three pages. Existing PDFs can
request this reading from their row. Suggestions never replace saved values
automatically, and edited fields remain protected. This background state uses
one Jev fan-out and does not delay filing. Box is not used.

The workbench, in navigation order:

- **Project Profile.** The determinants and project facts.
- **Project Plan.** A navigation placeholder.
- **Cost Plan.** Four sections: fees and charges, consultants, construction,
  contingency. Budget is the planning figure. The user enters the approved
  contract and the amount claimed. Forecast, variance and remaining are
  computed. An unpriced line stays unknown. Lines are prepared from the
  profile, the work items and the procurement packages. No price is invented.
  An Invoices tab beside the plan takes dropped invoices into one ledger:
  each is read in about a second, mapped to a supplier package and an
  activity line, and checked in code (duplicates, arithmetic, GST, ABN,
  splits). The plan's invoiced to date, budget remaining and % spent come
  from that ledger, with an Unallocated row so the totals always add up.
  Both views export to Excel and CSV and print. Variations is a placeholder.
- **Program.** A navigation placeholder.
- **Procurement.** One table of consultant and trade packages. Choosing a
  discipline saves the package. Create writes an RFP for a consultant and an
  RFT for a trade. The table proposes the disciplines the profile and scope
  imply. Removing a package can remove its draft cost lines in the same
  confirmation. Each RFP carries a returnable schedule (stages, options,
  rates, criteria) and a response form workbook. A proposal dropped on a
  row is filed and read in a few seconds: firm, prices against the schedule,
  rates, exclusions and departures. The price table compares firms line by
  line on the RFP's own rows, flags what needs a look, and takes a
  correction in one click. Award writes the chosen firm's lines under the
  package in the cost plan as the contract, and can be undone until an
  invoice is allocated. A one-page comparison report holds price, non-price
  ratings the user sets, a recommendation and sign-off, and prints A4.
  An existing project management plan stays reachable.
- **Specifications.** An editable six-column schedule of materials, products,
  finishes, assemblies and equipment, organised by the Profiler's building
  hierarchy. Scope, work action and provider read through from Profiler; the
  schedule owns detailed requirements and choices. The user edits cells, adds
  rows, records explicit unknowns, marks coverage, and issues a frozen PDF or
  CSV. Update specification reads filed documents in the background; readings
  remain proposals for review. Consultant documents retain authority until an
  explicit maintenance transfer with a reason. The browser table is built;
  designed PDF tables, images and automatic application assignment remain ahead.
  The document repository stays beside every view.

Delivery records and implication proposals stay on the project. They are not
in the left nav. Consultant suggestions still appear on Procurement.

## Positioning

Jev (TypeSafe System One) is the only AI, and it only picks. It is used where
a document must be read: filing, profile evidence, invoices, proposals (one
fan-out after filing), and the background specification import (one fan-out
per candidate). Code parses, harvests
candidates, computes, orders revisions and controls flow. Cost arithmetic,
package suggestions, procurement creates and specification acceptance are code
and user acts. Typing and navigation do not call Jev.

Every field states how it was decided (read by rule, picked by Jev, or set by
the user) and how sure that decision is. Uncertainty is shown, and a user
value is never overwritten.

## Operating Context

A Go single binary serves the API, event stream (SSE) and embedded React SPA
from one host with PostgreSQL 17 and content-addressed files. Access is
invite-only per organisation; every project belongs to one org. Documents are
the working material: consultant drawings with title blocks, reports, specs,
spreadsheets. Scanned PDFs without a text layer are stored but not filed.

## Capabilities and Constraints

- Speed is the first objective. Every user-facing path has a p50/p90 budget and
  a benchmark that fails the build.
- One Jev fan-out per filing, and one background fan-out per specification
  import candidate. Nothing is judged while the user waits if it can be
  precomputed. Grey means Jev did not answer in time; rule values stand and the
  rest stays unjudged.
- Bands per field: green (above that question's threshold), amber (applied,
  flagged), blank (below band, not applied), grey (Jev timed out).
- Green automatic acceptance is withheld. Kind and discipline may show a
  provisional amber review. Other Jev identity fields stay blank until their
  own cut-offs exist. Rule-settled fields show as settled. Profile bands are
  provisional and await owner approval; profile green stays withheld. The UI
  keeps that honest.
- Cost and specification status is written in words. An empty money cell, an
  unstated specification and a missing date stay unknown.
- Supersession is a link to an immutable prior filing. A user value, an
  accepted specification decision, an issued specification and a frozen cost
  baseline stay as they were. A later upload can flag them for review.
- Stack changes need a stated reason (see AGENTS.md).

## Brand Commitments

- Keep the Sitewise mark and wordmark and the owner-approved Clerk cockpit
  adaptation: Instrument Sans, shared semantic surfaces and compact working
  controls. `web/DESIGN.md` is the shared visual authority; current shell tokens
  govern light and dark appearance. The earlier warm-paper/IBM Plex register
  in `../clerk/DESIGN.md` records the product's origin, not a separate identity
  for new workbench surfaces.
- Evidence vs inference: on filing and profile, ember marks evidenced or
  user-owned values, azure marks model inference. Keep those meanings. Cost
  and specification status is written in words as well as colour.
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
6. Unknown is a value. An empty money cell, an unstated specification and a
   missing date stay unknown.
7. An issue freezes the record. Later uploads and edits can flag review; the
   issued package and the frozen baseline stay as they were.
8. A filed document is evidence until someone accepts it. Coverage and adoption
   are user decisions.

## Accessibility & Inclusion

Colour never carries meaning alone: every band has text and an icon. Full
keyboard operation with visible focus, and focus kept stable when live updates
arrive. Formal WCAG level not yet set (assumed 2.2 AA as the working target).
