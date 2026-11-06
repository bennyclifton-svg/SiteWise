# ST17-cdc-pathway notes

The base Tideway Apartments job approved by a complying development certificate that the certifier determines,
instead of a DA. Only `hdr.cond.planning` differs from the base (da -> cdc). Whether a Class 2 recladding can be
complying development is `verified: false` in the shape.

**The CDC premise is doubtful.** NSW Planning's Codes SEPP Fire Safety Code page lists only three
complying-development categories (fire sprinklers in residential aged care, alterations to hydraulic fire safety
systems, fire alarm communication link works); cladding replacement on an 8-storey Class 2 building is not one,
and no other code plainly fits (verified: false). The variant is kept as a test of how the app behaves on a CDC
route that the documents state, pending the owner's decision; it does not claim the route is lawful. The documents
no longer name the Fire Safety Code.

## Changes from the base

- d02 brief and d13 options report carry the trigger's evidence phrases in place of "Council has advised that a
  development application is required" / "A development application is required"; d13's clause 4.6, panel and
  consultation denials are reworded for complying development, so those flags keep a read source after d16 goes.
- d07 architect's CDC drawings stage (with photomontages for the general meeting), d08 fire engineer's support
  letter and d11 QS cost estimate report name the CDC instead of the DA; amounts unchanged. In d07 and d08 the CDC
  stage now follows design development and the DTS design review, because a CDC must show NCC compliance when it
  is determined; `fees.json` follows that order.
- d06 PM proposal allows for the certifier's assessment of the CDC application instead of council's assessment of
  the development application.
- d03: only the `cdc-eligible-land` trap's expectation changes (planning is cdc from d02 and d13).
- d12 town planner fee proposal and d16 statement of environmental effects removed (18 documents).
- Checked: no remaining read document says "DA", "development application" or "development consent". d01 (order),
  d05 (email), d09, d10, d14, d15, the A-201 drawings and both invoices were
  already neutral in the base. d03's "Complying development under the Codes SEPP may be carried out on the land"
  stays; here it agrees with the pathway.

## What the scorer should see

Shape trigger `cdc-pathway`: town planner removed from procurement and RFPs; DA lodgement, notification,
determination and CC rows removed, with the DA-worded general meeting and DA documentation rows replaced by CDC
ones and the pre-CC conditions row removed; CDC preparation and certifier determination added; DA fees line
removed. The certifier RFP's CC assessment line becomes CDC assessment and determination (b2_construction_approval
stands for the CDC); the architect's and fire engineer's DA lines become CDC lines. The scorer cannot re-point
`after` links, so "Pay long service levy", "Report progress to council", "Contract award" and the colours hold
point keep their remaining links and simply lose the removed DA/CC predecessors. The BCA
and certifier RFPs block on cdc (`unsupported_pathways`), and no template drafts a CDC certifier appointment (shape
gaps). The town planner row no longer exists because no proposal creates it, though the app may still suggest it if
it reads planning wrongly.

`flag.community`: d13's "neighbour notice the certifier gives" (and d02's "notice to neighbours") assumes a
neighbour notice applies to this CDC under the EP&A Regulation 2021; which notice applies depends on the code
(verified: false).

## Missed by design

As the base: `proj.target_construction_start` (d05 email), `proj.regulated_design` (no extraction), `proj.lot` (no
range form), `proj.client` cut at " - ", the table title blocks of d03, d11 and d20, and d18 → d17 supersession.
det.type_of_construction (A): derived only, never read; blank while rule.ncc.type-of-construction and its table are
draft (reason unreviewed). The key keeps the true value.

## Traps

As the base, less d16 `exempt-repainting` (removed with the SEE).
