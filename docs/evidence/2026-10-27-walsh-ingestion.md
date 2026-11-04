# Walsh ingestion fixes: evidence

27 October 2026. Plan: `docs/plans/2026-10-27-walsh-ingestion-fixes.md`.
Owner away; decisions D1–D10 are in the plan.

## What was measured

1. Walsh Renovation (dev project `e8a99013…`) re-read with live Jev after
   the change (question version profile-11, read-13): five documents, no
   failures.
2. A new dev project "Walsh check" (`e0281c28…`) with the same files dropped
   on Architect, Heritage and Head contract rows, the letter first, then
   awarded by stage. The owner's Walsh procurement history was not rewritten.

## Profile, before and after (Walsh, whole project)

| Item | Before | After |
|---|---|---|
| Scope rows | 16, all the house/extension default | 32: the 16 defaults plus stairs and sanitary (amber, from action readings), cold and hot water, wet-area waterproofing, tiling, joinery, internal doors, local exhaust, demolition, masonry, adjoining structures, fire compartments, acoustic separation, thermal performance (suggested, implied by the documents' words), and site services connections (amber, see below) |
| Unknowns | — | hazardous materials, switchboards, landscape, termite management, underpinning stay out of scope |
| GFA | `gfa_sqm` 95 (the addition, read as the building) | `added_gfa_sqm` 95; existing and proposed unknown |
| LGA | City of Sydney LEP | City of Sydney |
| Architect lodges | blank (0.37) | yes (amber) |
| Party wall | no question | `sig.party-wall-stated` true |
| Owner targets | none | construction start "early 2027", maximum "18 months"; DA lodgement red: the letter says July 2026, the brief mid-2026, so the user picks |
| Builder estimate | none | "$880,000 – $980,000" and "14–16 months" |

Site services connections came from Jev reading the builder's "standard
hydraulic upgrade" as an upgrade of the service connections (0.94). It is a
sourced reading under rule D2; the user can remove it.

## Cost Plan (Walsh)

- Spread lines say: "The stated budget includes a contingency the document
  does not quantify, so these shares include it. The document does not say
  whether the budget includes GST."
- Construction contingency $85,000: "The document holds it in addition to
  the construction budget."
- Builder's preliminary estimate row: $880,000 – $980,000, ex GST, marked
  as advice, not a budget or a price.
- Opening the plan ran its normal prepare, which added lines for the newly
  scoped systems (site works, mechanical) and re-spread the $920,000.
- Plan read 6–7 ms (budget p50 100 ms, p90 250 ms); prepare 109 ms single
  sample.

## Program (Walsh)

Targets strip: construction start early 2027 (by 30 April 2027), program
27 April 2027, met. Maximum 18 months and builder's 14–16 months against a
construction stage of about 1.3 months: the stored program (generated
before this change, with the 250 m² assumption and library durations) is
flagged "shorter than the builder's estimate". The added 95 m² now reaches
the Program's floor-area driver for an extension; the stored program
changes only through its Review. Program read 12 ms.

## Procurement ("Walsh check")

| Document | Result |
|---|---|
| Engagement letter | Atelier North Pty Ltd, Lucia Ferraro; dated 2026-03-03; PI $5,000,000 to 2026-06-30 (expired); flags `abn_invalid`, `monthly_fee` |
| Fee proposal (read after the letter) | firm Atelier North (sign-off), dated 2026-02-18, **superseded** by the letter |
| Builder email | Deluca Building Pty Ltd, Marco Deluca, dated 2026-03-22 (first read, before the sign-off-first fix: Atelier North from the forwarder's address, date missed) |
| Heritage advice | Heritage Insight Pty Ltd, dated 2026-03-28 |

Jev left the kind of both emails unanswered, so they show as unconfirmed
submissions on their rows rather than "Named in a document"; a document
read as not a proposal shows as named.

Appointment rows keep the letter's words and map Construction documentation
to For-construction documents and Construction administration to Contract
administration. Award by stage wrote six cost lines in the letter's words
(A.1–A.6, $96,500), 83 ms (budget 200/500 ms). The proposal stayed
superseded. Profile `appointed`: Architect, Atelier North Pty Ltd (Lucia
Ferraro), studio@…, PI to 2026-06-30. Profile read 8 ms.

## Tests

- New: `internal/jobs/routes_test.go`; `internal/profile/{scope,gfa,targets}_test.go`
  additions; `internal/proposals/walsh_test.go` (7 tests on the Walsh files in
  `internal/proposals/testdata/walsh`); `internal/costs/framing_test.go`;
  store tests for supersession, PI, named firms, appointed firms, estimate,
  stage mapping, owner targets, floor area and the budget note.
- Changed: `TestActionThresholdAndScopeAreIndependent` — an applied action
  above its calibrated floor is now scope (D2).
- Meridian answer key: 38 → 39 of 46 on stored readings. Its pre-existing
  failure (`sig.acoustic-separation-stated` wrong value) is unchanged.
- Pre-existing failures, unchanged: four hold-point tests (CRLF YAML),
  `TestProfilePlacesInputsInBuildOrder` (RFP flags with no home),
  `TestProgramRoutes` and `TestProgramImportUploadReadAcceptUndo`
  (library durations, leaf stage).
- Web: typecheck clean, build passes, 234 unit tests pass.

## Not done

- The owner's Walsh submissions keep their earlier reading (proposal
  "unsuccessful", letter dated the upload day, no PI). Code sets a date only
  on a first read; a direct database correction was refused by the session's
  safety check. Detaching and re-dropping the two files on the Architect row
  re-reads them; the award would need to be undone and redone.
- The 52-file golden fee-proposal corpus check could not be run here (the
  corpus is not on this machine's test path).
