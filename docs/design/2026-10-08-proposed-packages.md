# Proposed packages

Status: draft direction, agreed with the owner 8 October 2026. No code yet.

## Problem

The Packages tab asks the user to author every package from blank forms:
title, kind, scope type (work responsibility or general obligation), stage,
wording, deliverable and inclusion. The profile already holds most of the
answers. Typing them again is friction, and blank forms invite gaps.

## Outcome

Opening Packages shows a complete proposed set of packages derived from the
profile and works. The user reviews it: accept, rename, merge, split or move
items between packages. Forms remain only for exceptions.

Serves time-to-decision (review instead of authoring) and answer trust (the
gap list shows every work item nobody is contracted to do).

Lane A: packages are records a user trusts and RFPs/RFTs read them.

## Decisions

1. **Works default to one head contract.** Capex jobs and new builds get a
   single works package covering every work item. The user can split it into
   trade packages; work items move with their responsibilities.
2. **Proposals are shown, not saved.** Proposed packages appear as distinct
   "proposed" rows. Nothing becomes a package record until the user accepts
   it. Accepting uses the existing planning acceptance path (provenance,
   project lock, undo rules).

3. **The procurement route drives design responsibility, defaulting to D&C.**
   Packages read the profile field `hdr.cond.procurement_route`. When it is
   unanswered, they assume Design & Construct (most projects): the head
   contract carries design responsibility for work items, and services
   packages are proposed for novation (pre/post stages). The assumption is
   labelled on the Packages tab and links to the profile question.

   | Route | Design responsibility on work items |
   | - | - |
   | Design & Construct (default), ECI | Head contract; consultants novated |
   | Traditional (Lump Sum) | Services packages; head contract constructs only |
   | Managing Contractor, Alliance, PPP | Proposed as D&C, flagged for the user to confirm |

4. **Procurement route becomes an elevated profile question.** It joins the
   rows read first on the profile (`GLANCE` in `web/src/Profile.tsx`), next to
   work type, because it changes the shape of every package. Changing the
   answer recomputes proposals; accepted packages whose design roles no longer
   fit the route are flagged, not changed.

## What is proposed

| Kind | Proposal | Source |
| - | - | - |
| Services | One package per discipline the job needs, with catalogue stages (novation-aware) | Existing `SuggestedPackages` baseline + complexity catalogue |
| Works | One head contract listing every live work item with roles filled (install, test, certify; design where the item is D&C) | Work items |
| Supply | Only where the profile records owner-supplied items | Profile provider facts |
| Obligations | `retain` items become obligations on the works package (e.g. keep hydrants live); consequences add their required obligations | Work items, `knowledge/` consequences |

Services also take the design responsibility for work items in their
discipline, so each item has a designer and an installer by default.

## Gap list

Above the packages: every live work item missing a design or install
responsibility, and every role claimed by two packages. Empty list means the
proposal covers the works. This is the existing gap check, shown as a to-do.

## Speed

Proposals are computed from profile rows and catalogues in code. No Jev call
on this path. Reads stay inside the existing package budget, p50 100 ms /
p90 250 ms, measured by the existing package benchmark extended with
proposal generation.

## Out of scope

- **Nominating firms.** Later step: each package gets a list of firms who
  receive the RFP/RFT, then their returned offers. The schema already
  anticipates an offers table keyed by package, tenderer and cost item
  (architecture schema, "Returned tender offers"). Proposed packages must keep
  stable identities once accepted so firms can attach to them.
- Pricing, tender issue, builder-side subcontract procurement.
- Changing the Works tab, or the Profile tab beyond elevating procurement route.

## Edge cases

- Empty or early profile: propose only what is known; the gap list says what
  is missing rather than guessing.
- Work item added after packages are accepted: it appears in the gap list and
  as a proposed addition to the head contract, not silently added.
- Work item marked out of scope or unknown: its proposed responsibilities
  disappear; accepted ones are flagged, never deleted.
- User rejects a proposal: it stays dismissed until the inputs that produced
  it change.
- Existing manually created packages: matched by kind and discipline so they
  are not proposed twice.

## Done when

- A capex fixture (single-system replacement) shows one services package,
  one works package, and an empty gap list with no typing.
- A new-build fixture shows the discipline set, the head contract and
  obligations from consequences.
- Accept, split and merge round-trip through existing acceptance and undo.
- Package read benchmark passes the 100/250 ms budget.
- No proposal is persisted until accepted.
