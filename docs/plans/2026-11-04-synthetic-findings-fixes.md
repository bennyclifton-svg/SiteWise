# Fixes for the synthetic-corpus findings (F1 to F16)

Date: 4 November 2026. Owner direction: "turn all the findings into fixes".
Findings: `testing/synthetic/FINDINGS.md`. Evidence: `testing/synthetic/runs/`.

All fixes are lane A. Each is fixed only as far as its finding, test first,
with existing tests, answer keys and speed budgets still passing. After the
merge, the T01, T05 and T09 scenarios are re-run unchanged to measure each
fix.

## Direction choices (decided before code)

**F15: a lopsided choice settles.** A choice field takes its leading value
when that value has at least 3 sources and at least 3 times the sources of
all other values combined. The other values stay visible as alternatives,
and the band is amber, so the user still sees the dissent. Anything closer
stays a conflict, as today. Choice fields only; numbers and money keep
today's rule. Why: one passage about internal alterations blanked the work
type of every extension job (42 to 5), which sent the program to
`sk.generic`; one sentence about a lean-to blanked a heritage conservation
area stated 14 times.

**F10: a house's certifier is appointed, not advisory.** For a Class 1
project the certifier package defaults to certification (principal
certifier), because a house cannot proceed without one. Other classes keep
the advisory default. The certifier RFP gains the Class 1 and 10
critical-stage inspections and the BASIX check at the occupation
certificate, cited to the EP&A Regulation 2021 where the clause is
confirmed from the instrument, otherwise `verified: false`.

**F4: the procurement route is asked on its own cue words.** A passage
naming a tender, lump sum, design and construct, D&C, ECI, managing
contractor, construction management, novation or head contract gets the
procurement-route question wherever it sits. It rides the passage's
existing fan-out; no new round trip.

## Work split

| Group | Findings | Area |
|---|---|---|
| Invoices | F5, F8, F13, table date | `internal/invoices` |
| Fee proposals | F3, F14, firm on hazmat letterhead | `internal/proposals` |
| Profile details | F2 (LGA and address), F16, client care-of | `knowledge/profile`, profile details |
| Profile routing | F4, F15 | `internal/profile` questions and resolution |
| RFPs | F9, F10, F11, F12 | `knowledge/rfp`, `internal/rfp` |
| Filing and Update | F7, F1 | filing discipline, `internal/httpapi/profile.go` |

Out of scope: lifecycle answers never applied, supersession never written,
and the knowledge gaps the shapes list as `built_in: no`. These are known
gaps, not findings.
