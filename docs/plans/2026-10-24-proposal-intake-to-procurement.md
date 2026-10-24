# Fee proposals into Procurement, and a cleaner profile from them

Date: 2026-10-24. Branch `intake/proposal-procurement` (worktree
`../sitewise-proposals`). Source: the critique of the Kavanagh Residence
intake (four Markdown fee proposals: architect, structural, hydraulic, QS).

## Outcome

A PM drops consultant fee proposals anywhere. Each one appears on the
Procurement page under its consultancy (Architect, Structural, Hydraulic,
Quantity Surveyor) with the firm's name and its fee, without the PM
choosing a package. The profile reads the same files with less noise and
the facts they plainly state (pool, client, address).

Serves time-to-decision (the comparison is ready when the files land) and
answer trust (no Markdown in firm names, no "false" signals from silence).

## What the Kavanagh files showed

| # | Finding | Root cause in code |
|---|---|---|
| 1 | Firm names saved as `**Quoin Architecture Pty Ltd**`; addresses blank; Vertex has no firm | Harvest keeps Markdown emphasis; firm and address questions are uncalibrated (5–11 samples), so Jev's answers are dropped; Vertex's email domain `vertexadvisory` does not corroborate "Vertex Cost Advisory" |
| 1 | GST basis "unknown" on all four | Harvest counts "Total fee including GST" as inclusive wording, so the ex-GST rule does not fire; the GST question is uncalibrated |
| 1 | No stage fees | No package at read time, so no fee schedule to price against |
| 11 | Not linked to any package | Proposals were read at 13:33:04; the discipline packages were created at 13:37:05 by profile adoption. The reader only links an existing package |
| 11 | No QS package | The profile does not suggest a QS for a house; nothing creates one from a received proposal |
| 2 | Number blank ("Proposal reference: QUA-KAV-2601") | No label rule for proposal/quote references |
| 6 | Lifecycle always blank | No lifecycle threshold; no rule for fee proposals |
| 7 | Pool missed (0.85 not_stated) | Presence asks what the works commit to; "pool interface" in a consultant scope reads as not stated |
| 3 | Client never asked | No trigger for the addressee line "To: …" |
| 5 | State cited from the consultant's office address | The letterhead and the "To / Project / Site" block are one passage |
| 9 | 27 signals shown as amber "false" | A signal is "X is stated"; false only means silence. Downstream only `true` is used |
| 10 | Four agreeing documents still need four looks | Nothing shows agreement |
| 12 | Amber rows spread across the profile | No single list of what to confirm |

Two earlier recommendations are withdrawn after reading the code:

- **Merging the two retaining-wall keys (item 8).** They are deliberately
  separate: `site.retaining-walls` (free-standing) and
  `substructure.retaining-walls` (support the building) have different
  rules and failure modes. The building is the model; the split stays.
  The Kavanagh miss is a low-confidence reading, not a duplicate key.
- **"About 50 s queue delay".** Reading waits for the PM's "Update project
  profile" click by design.

## Overlap with work in progress

Another agent has uncommitted work in the main tree on: submission-to-package
matching when packages change (`internal/store/submission_matching.go`),
Markdown filing identity (`internal/intake/markdown_identity.go`), Markdown
kind rules in `settleVocabulary`, the client cue regex, and profile-9 question
wording (consultant exclusions are not project exclusions). This branch does
not duplicate those. At merge:

- `markdownIdentity` gets the proposal-reference keys this branch adds to
  `labelAt` (PDF and Word).
- After their matcher assigns a package, a submission with no priced cells
  is re-read once so its stage fees fill.
- The presence wording below is added to their profile-9 text, and the
  question version becomes the next number after theirs.

## Changes

Lane A unless marked. Budgets from `bench/budgets.json`; nothing here adds
work to the filing request.

### A. Proposal reading (`internal/proposals`)

1. Strip Markdown emphasis (`**`, `__`, `*`, `_`) from harvested lines
   before names, addresses and labels are read.
2. Letterhead rule: when the proposal prints exactly one valid ABN, the
   company name printed within three lines above it, and the one street
   address within three lines of it, are the firm's. Lines labelled
   To / Client / Attention / Project / Site / Prepared for are never the
   firm's. Rule-decided, green.
3. GST: inclusive wording on a line whose amount is the inclusive twin of
   an ex-GST total ("Total fee including GST" under "$96,000 + GST $9,600")
   does not count as inclusive basis. Then "exclusive of GST" settles ex.
4. The proposal's own date ("Date:" in the header block, not in the future)
   becomes the received date. The upload date stays when none is printed.

Done: the four Kavanagh files read with clean firm name, ABN, address,
email, phone, GST ex and stated total, by rule, without a Jev answer.

### B. Proposal to Procurement (`internal/store`, `internal/httpapi`, web)

1. Before a filed proposal is read, if its filed discipline is a consultant
   or trade discipline and no live package holds it, create that package
   (origin `document`, provenance names the document), with the default
   stages. A discipline whose package the PM removed is not recreated.
   One live package per discipline still holds (existing guard), so later
   profile adoption skips it.
2. The read then links that package by the existing discipline rule and
   prices the package's fee schedule (price questions are calibrated).
3. The Procurement firm chip shows the stated total when no schedule cell is
   priced, marked "stated", instead of "—".

Edge cases: discipline blank or supplier → no package, stays unassigned;
two packages with one discipline (legacy) → no link; user dropped the file
on a row → the user's package wins; package created but read fails → the
row exists with the firm chip in its failed state.

Budget: proposal read is background; `proposal_read` and `price_table`
budgets unchanged. Package creation is one transaction under the project
lock.

### C. Filing (`internal/intake`)

1. `labelAt`: "Proposal reference/ref/no/number", "Quote/Quotation
   no/number/ref/reference", "Our ref" are number labels.
2. A document whose own first heading or filename says "fee proposal" or
   "fee submission": kind Commercial and lifecycle Consultant by rule.
   Owner check: consultant fee proposals file under 02-consultant; a
   contractor's tender or quotation is not covered by this rule.

### D. Profile reading (`internal/jobs`, `internal/profile`, knowledge)

1. Passage split: a labelled addressee block (To / Attention / Client /
   Project / Site / Re) starts its own passage, so the letterhead's address
   is not read as the project's.
2. Skip the label fan-out for signature and acceptance blocks
   ("Accepted for client: ____") and billing or payment-terms sections:
   nothing in them describes the building.
3. State from the site address postcode, computed in Go from the
   Australia Post state ranges (data file, `verified: false`). Shown as a
   calculation when no document states the state; flagged when a document
   disagrees. LGA lookup needs a postcode-to-council dataset (ABS
   correspondence); not downloaded without the owner's approval.
4. Client: "To:" on an addressee block is a client cue (candidates only;
   Jev picks or says none; "select none when a consultant or contractor").
5. Presence wording: a consultant scope that coordinates with, interfaces
   with, or excludes its own design of a system shows the system is part of
   the project ("pool interface", "pool engineering excluded" → pools
   included, provider not stated). Question version bumped.
6. Signals: a signal row whose only value is "false" shows as not stated
   (blank), not amber "false". Downstream already treats false as unknown.
7. Agreement: a row records how many distinct documents gave its value.

### E. Profile review list (lane B, web)

One "To confirm" list at the top of Profile after a reading: amber
document-backed rows in importance order (site and client first, then
building, then scope), each with its value, "N documents agree" and
Accept. Accept writes the user value through the existing route; the site
address accepts onto the site record. Collapses when empty.

## Out of scope

Calibrating firm, GST or lifecycle thresholds (needs labelled answers);
merging retaining-wall systems; LGA data; changing how awards write
contract values.

## Verification

- Unit tests for every rule above with the Kavanagh files as fixtures
  (synthetic, marked so).
- Store test: proposals filed before packages exist land under the right
  package with firm and total; a removed discipline is not recreated.
- Live Jev re-read of the four files (owner allows live calls) and a
  Kavanagh profile answer key: pool, client, address, storeys, work type.
- Filing benchmark and `python tools/check_knowledge.py` pass.
