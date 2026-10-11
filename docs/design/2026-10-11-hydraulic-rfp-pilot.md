# Hydraulic RFP pilot

Owner direction: 11 October 2026. Implementation pilot for Hydraulic only;
discipline knowledge remains draft pending owner review.

Outcome: a manager can review a mostly resolved appointment, change a service
or stage in place, and give every consultant one consistent fee schedule.
This serves time-to-decision and answer trust. Lane A selection/report work
and lane B presentation; no new dependency, service or AI call.

## Agreed document

- Release date and proposals close sit immediately below the title. The old
  reference, questions contact/deadline and start-by inputs are not required,
  displayed or printed for Hydraulic. Closing before release blocks issue.
- Project, then Scope of services. Role obligations and shared-system
  boundaries belong here. One stable, logically grouped table offers mutually
  exclusive Included / Optional price / Excluded choices. Exclusions retain
  a responsible party; uncertainty stays visible beside the affected item.
- Scope by stage and fee proposal combines the stage summary and R1 pricing.
  Checked stages form the base fee. Unchecked stages are separately priced
  options, with explicit alternatives for not in this RFP or done by others.
  The initial design appointment ends at design, before tender. Existing saved
  appointment ranges and individual stage choices continue to win.
- Fee, weeks and hours columns are for the consultant to complete. Base and
  option totals stay separate. Other optional services have pricing rows;
  stage services are not duplicated in R1. Remaining returnables start at R2.
- Allowances, Terms, remaining returnables, then the internal issue action.
  Evaluation and its weight controls/checks are removed from this pilot.

## Resolution and preservation

Rainwater reuse and river/carted/other sources now have separate facts and
scope items. Standard rainwater, domestic pumping, roof drainage and design
coordination use explicit pricing assumptions when unknown; they do not
assert that equipment exists or that a source is suitable. Unusual source,
trade-waste, gas, utility and staging work is separately priced when unknown.
Explicit absence still hides an irrelevant item. Conflicting facts remain
blocking. Treatment, civil and declaration boundaries still require a decision
where material facts are missing. Selected stages bound pricing assumptions.

A legacy combined-source fact can establish absence, but a positive answer
cannot identify either new source type. Such positive/conflicting legacy
answers require clarification. Old combined-source overrides remain visible
as blocking orphan settings until reviewed; they are never silently remapped.
Stable report block IDs preserve custom wording during refresh. Removing a
generated block does not discard protected wording; the normal orphan-review
flow handles it. Other disciplines retain their existing structure.

## Verification and budgets

RFP setting/read budgets remain p50 100 ms / p90 250 ms. Assembly remains
300 / 1,000 ms; filing remains 1,000 / 2,000 ms. The populated RFP test gates
resolve+sections at p90 20 ms and PDF render at p90 100 ms. All Hydraulic
scenarios must fit the existing three-page limit at normal type size.

Regression coverage includes stage changes and persistence, source conflicts,
unknown pricing assumptions, explicit exclusions, old combined-source choices,
date order, removal of evaluation/extra issue fields, one pricing schedule,
and desktop/mobile layout. Existing RFP answer keys cover the other disciplines.
Knowledge validation must run after changing the discipline data.

Verified locally on 11 October 2026: backend RFP/report/knowledge tests and
Hydraulic desktop/mobile persistence test pass; existing Architecture, Civil
and Electrical editor tests pass. All eight Hydraulic PDF scenarios fit three
pages. Knowledge validation: zero errors, two existing works-default warnings.
Populated RFP assembly p50/p90: 1.00/1.00 ms; PDF: 38.52/44.70 ms.
The 24-file recorded-latency filing run measured 432.3/990.5 ms against
1,000/2,000 ms. Report setting measured 66.4/68.6 ms against 100/250 ms.
This is local replay evidence, not a live Jev or target VPS benchmark. Its
broader gate failed on the separate spec_home_profile_edit p50 (51.6 ms vs
50 ms); no profile performance changes were made in this pilot. Detailed
samples and results are in ignored .tools/hydraulic-bench-{samples,results}.json.


## Owner refinement: one scope table

The second review removes the separate appointment-fact questionnaire and
Scope position editor. Every visible service has four exclusive choices:
Included, Optional price, Excluded, Unknown. Unknown is saved explicitly and
blocks issue; it is a scope decision, not a factual claim about the building.
Conflicting source facts remain blocking and can be corrected inline. Parts
covered move to Project. No Rule badges appear in the scope table.

Water supply/storage, sewerage/trade waste, stormwater, gas and coordination
are distinct groups with action-led descriptions. Water Services Coordinator
is written in full. Investigation and authority rows edit the same stage
settings as the fee schedule; they print only in that schedule to avoid
repeating scope. Later-stage deliverables remain in the stage schedule.

The former combined domestic/sanitary choice is inherited by the new sanitary
row until it is set separately. Editing domestic freezes the sanitary choice
seen on screen in the same transaction, so the toggles remain independent.
Protected introductory wording no longer disables scope or stage controls.
No new dependency or service; the scope remains lanes A and B with the same
budgets. Regression tests cover Unknown persistence and blocking, independent
water/sanitary changes, stage synchronisation and protected wording. Four
browser checks and the backend tests pass. PDF scenarios remain within three
pages; assembly p50/p90 1.00/1.00 ms, rendering 36.37/42.01 ms on this host.
