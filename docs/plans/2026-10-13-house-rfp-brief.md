# House-aware architect RFP and the brief it reads

Date: 13 October 2026. Lane A (rule numbers, filing of facts, anything a user
trusts). Source: owner critique of the Custom Home 2 architect RFP (Class 1a
house, CDC, traditional, no source documents).

## Why

Sent to three residential architects, the Custom Home 2 RFP would return
three incomparable fees and a page of questions. Commercial defaults leaked
into a house (tenant fit-out, signage, façade certification, BIM, PI $20M,
ISO systems, "unallocated façade engineer" checks), while the two things a
house fee is priced on, the brief and the budget, were missing. The profile
already held 4 bedrooms and 1 garage space; the RFP printed neither.

Every attribute added here lands in an extraction shape the profile already
has, so "Update project profile" fills it from a brief or specification
with no new reading machinery. The RFP stays pure code over the profile.

## Owner decisions (13 October 2026)

1. The construction budget is printed in the RFP (disclosed by default).
2. The architect carries professional indemnity and public liability cover.
   The owner gave no amounts: a house falls back to the shared client default
   (PI $10 million, PL $20 million, `insurance_limits`, unverified).
3. Who engages each other consultant is a question per package: the client
   directly, or the architect as a subconsultant. Either is common.
4. A Class 1 house gets one documentation set for tender and construction.
5. Who lodges the planning application is a question: the architect or the
   owner. The certifier never lodges a CDC; it assesses it.
6. The Newham on York builder's project specification (22 spec homes,
   Kellyville) is the golden end product for the next pass, which walks the
   specification discipline by discipline into the profile scope. Not here.

## Slices

| # | Slice | Where |
|---|---|---|
| 0 | RFP grid prints the house facts the profile holds: garage spaces as car parks, bedrooms, bathrooms | `rfp/drivers.go`, `knowledge/rfp.go` grid |
| 1 | House and non-house modules; commercial rows leave a house RFP; house rows (kitchen, bathroom and joinery design; finishes and fixtures schedule; specification included); one documentation stage; water-asset approval row; lodgement question; house insurance and QA wording | `knowledge/rfp/*.yaml`, two new condition drivers `only_class` and `module` |
| 2 | Brief attributes read by Jev: bathrooms (class field), quality level (condition), stated budget (project detail), architect lodges (flag reading); "Information provided" listed from the filed register by kind and discipline | `knowledge/profile/*.yaml`, `profile/details.go`, `profile/harvest.go`, `store/report_documents.go` |
| 3 | Budget required by the architect RFP; printed by default at issue; a stated budget stands in until the cost plan holds one | `rfp/resolve.go`, `web/src/Reports.tsx` |
| 4 | `packages.engaged_by` (client / lead consultant / unset); Procurement column; the RFP lists client-engaged consultants and the subconsultants priced in the fee; unset blocks issue | migration 021r, store, API, `web/src/Procurement.tsx`, `rfp` |
| 5 | Optional "Send questions to ... by ..." line and its two controls | `knowledge/rfp/skeleton.yaml`, `rfp/controls.go` |

Out of scope: evaluation weights (removed on 11 October by the alignment
plan and kept out); the 103 planning-and-approvals questions a house shows
in Profile (separate relevance defect); suggested consultants for a CDC
house (Mechanical, Electrical, Civil, Town Planner are over-suggested by
`package_defaults`); reading consultant engagement from documents.

## Budgets and checks

No change to filing or Jev fan-out. New questions run only on passages their
triggers match, inside the existing Update fan-out; `ReadRevision` rises so
finished documents are read once more from the call cache. RFP resolution is
still pure code: p50 ≤ 1 s, p90 ≤ 2 s for a draft refresh, and the existing
three-page gate holds for every scenario.

Done means: answer keys for a new-build CDC house (`house-cdc-new`) and the
updated small Class 1 extension pass; every other answer key and wording
snapshot passes or changes only where a house row moved; `check_knowledge.py`
passes; the speed bench passes; Custom Home 2 refreshed in the browser shows
the house RFP.

Edge cases: unknown class keeps the commercial rows (module `if_unknown:
include`) and leaves the house rows out with one non-blocking check; a house
with a Class 10 garage is still a house; a mixed Class 1a + 5 building is
not a house; an unset engagement blocks issue only for live consultant
packages; a stated budget never overrides the cost plan.
