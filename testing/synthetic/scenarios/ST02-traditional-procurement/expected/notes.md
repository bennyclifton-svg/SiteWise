# ST02-traditional-procurement notes

The ST02 base with one change: the developer's consultants prepare full construction documentation for a lump-sum
tender under AS 4000, stay engaged by the developer through construction, and nobody is novated. Only
`hdr.cond.procurement_route` (design_construct -> traditional) and `fact.contract_form` (as4902 -> as4000) differ
from the base. 18 documents in 4 waves: base
d01-d18 with d01, d04, d05, d06 and d07 changed. Every package's novation is `no`.

## Passing mentions checked

Searched every base `content`, `carries` and title block for D&C, design and construct, AS 4902, novat and
post-novation. d01 (route and contract form), d04 (tender timing; correspondence, not read), d05, d06 and d07 (stage
labels, options and totals) named them and were changed. d02, d03, d08-d18 never named the procurement route. The
d11 "Stage 1 CC / Stage 2 CC" sentence stays; under a traditional route the CC is issued before award (d04).

## Fees

- d05 architect: Stage 4 tender documents $420,000 and Stage 5 for-construction documents $140,000 replace the D&C
  tender stage; total $1,100,000 ex GST; option: design support during construction $260,000.
- d06 structural: tender documents $150,000 and construction documents $95,000; total $309,000 ex GST.
- d07 civil (GST-inclusive): detailed design for tender $39,600 and final design for construction $22,000; total
  $116,600 inc GST ($106,000 ex GST).

## Traps

As the base. The brief's "Consultants remain engaged by the developer through construction; no novation" is the
DOMAIN.md §10 "novation" negation.

## What the scorer should see

Shape trigger `traditional-procurement`: program rows "Prepare construction documentation before tender" and
"Construction certificate before award", "Novate the design consultants" removed; architect, structural and civil
RFP stages as the trigger lists (tender and for-construction stages in, D&C and post-novation stages out).

## Missed by design

As the base.

## Shape change (review 2026-11-06)

The trigger's `profile` now lists `fact.contract_form`, so this variant keys `as4000` (d01 "AS 4000-1997 General
conditions of contract, lump sum", d05 "lump-sum tender under AS 4000") and the base keys `as4902`.
