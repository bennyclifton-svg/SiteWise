# Quoted fees and scope coverage

## Decision and scope

User approved 20 October 2026: show consultant offers before their mapping
to an RFP, with scope coverage separately. Lane A: faster decisions without
inventing allocations or counting a package total together with components.
Applies to consultant disciplines through their existing schedules, with no
geotechnical-specific runtime branches. Builder comparison remains separate.
No new service or dependency.

Existing budgets: price_table 150/300 ms p50/p90; submission_write 100/250 ms;
proposal_read 3000/6000 ms. No model calls during navigation, correction,
comparison, reports or award.

## Implementation

Harvested source fees are stored in `submissions.quote_review` independently
of requested schedule prices. The screen leads with packages and totals,
retains page links, and exposes source lines, conditions and alternatives.
An amount without established GST basis stays unknown. Repeated bare figures
can be folded from the headline when a named offer has that same figure;
all source occurrences remain available.

Coverage selects a source bundle and can reference that bundle from several
services. It never divides money between stages. New Jev questions share the
existing fan-out and are explicitly uncalibrated suggestions. Their answers
cannot create accepted prices. User coverage corrections take precedence.

Confirmation records a separate amount and scope note. Go computes totals
and award values from that confirmed appointment only. Legacy manual columns
also exclude component prices once an appointment is confirmed. Existing
historical schedule-only awards retain their breakdown behavior. Alignment
cannot move or combine the reserved appointment row. Reports and Excel
exports distinguish quotes, coverage and confirmed fees.

Reader repairs retain equal-valued distinct source rows, clean prose from
firm-name candidates, preserve revision context, read subtotal section
headings, recognize “plus 10% GST”, reject liability clauses and an area-plan
OCR artefact, and require tax evidence before treating a 10% difference as
a GST twin. Coverage questions use the exact candidate set sent in state.

TypeSafe references:
[fan-out](https://docs.typesafe.ai/patterns/fan-out),
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[confidence](https://docs.typesafe.ai/confidence),
[model limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).

## Hale validation

With explicit permission to send these four sources to TypeSafe, reran their
native text plus missing-page OCR through Jev 1.13.0. Refreshed machine
readings only, retaining user corrections. No actual comparison fees were
confirmed and no actual awards were made.

Assertions against the original PDFs:

- JK: revised auger 11,700; cored alternative 15,500 includes auger;
  test pits 11,300. Older and conditional source figures retained separately.
- Geotechnique: quoted subtotal 21,395.
- EI: investigation/report 11,950; OSD work 12,250; conditional waste 4,000;
  printed estimate 28,200 retained without adding it to its components.
- Douglas: geotechnical subtotal 47,000 and contamination subtotal 16,050.
  GST twins, liability cap and drawing OCR false fee do not become quotes.

All values above exclude GST. Source IDs, extracted evidence and recorded
responses are in ignored `tmp/hale-quote-check`; source PDFs remain in the
local project store. Latest extraction plus single live fan-out timings:
JK 2357 ms, Geotechnique 462 ms, EI 635 ms, Douglas 1197 ms. Request sizes
58,425 / 57,913 / 75,300 / 73,826 bytes, below the existing 90,000-byte bound.
These four checks are not a statistically sufficient upload-to-read latency
benchmark or a calibration of the new scope questions.

## Tests and limits

- Full proposals, store and RFP suites pass.
- Targeted HTTP upload/read/confirm/export/award/report/undo and isolation
  regression passes. Five-firm architecture comparison budget test uses
  the actual HTTP paths and gates against `bench/budgets.json` (20 samples):
  read p50 7.062 ms / p90 7.587 ms; confirmation p50 8.605 / p90 10.330 ms.
- Bundle/coverage regressions cover geotechnical, architectural, traffic and
  town-planning services. These are synthetic deterministic tests; only the
  four Hale geotechnical proposals were live-validated.
- Three Chromium tests pass: manual comparison alignment, attachment to an
  existing firm, and complete quoted-comparison/award/report/undo workflow.
- TypeScript and production Vite build pass. Existing bundle-size warning
  remains.
- Broader HTTP suite still reports the pre-existing
  `TestProfilePlacesInputsInBuildOrder` missing-home assertions. A transient
  profile-write failure in `TestProcurementRowsAndCreate` passed on isolated
  rerun. These are outside this change.

Source relationships can still be ambiguous, especially revised attachments
and coverage inferred from narrative rather than a fee table. The interface
shows suggestions as suggestions, keeps unclear coverage unknown, and
requires the user to confirm the appointment amount and scope.
