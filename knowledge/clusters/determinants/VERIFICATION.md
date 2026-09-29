# Primary verification — 29 September 2026

NCC 2022 adopted Volume One primary HTML was readable without authentication.
This verifies the identified national clauses/tables, not project applicability,
state adoption, later editions or all statements in the seed corpus.

| Item | Primary finding | Merge action |
|---|---|---|
| Type of construction | C2D2 / Table C2D2; the seed's C1.2 locator is wrong | Publish `type_of_construction.yaml`; correct the fire rule |
| Rise calculation | C2D3 corresponds to 2019 C1.2 | Keep rise as an explicitly stated determinant; no simplified counting algorithm |
| Mixed classifications | C2D4 requires attention to top-storey classification | Exclude mixed classification from the baseline lookup |
| Sprinkler concession | 2019 C1.6 maps to C2D7, Class 4 parts, not a general sprinkler downgrade | Reject the generic Class 5 A-to-B inference; retain unverified seed claim for review |
| Compartment limits | C3D3 / Table C3D3 depends on class and construction type, with both area and volume | Publish `compartment_limits.yaml`; do not use either seed figure as a universal Class 5 limit |
| Service openings | C4D15 corresponds to 2019 C3.15 | Correct the seed's erroneous C4.5 locator; keep exact protection routes subject to clause scope |

Sources: [C2](https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c2-fire-resistance-and-stability),
[C3](https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c3-compartmentation-and-separation),
[C4](https://ncc.abcb.gov.au/editions/ncc-2022/adopted/volume-one/c-fire-resistance/part-c4-protection-openings).

## Contradictions resolved

Table C2D2: three-storey Class 2/3/9 requires A, not the seed's B.
Two-storey Class 5/6/7/8 requires C, not the seed's B. Class 9 belongs
with 2/3, not 5–8. These are baseline table results, before exceptions.

The competing Class 5 compartment claims confuse different table columns
and classifications. For Type B, Class 5 is 5,500 m² / 33,000 m³; 3,500 m²
belongs to the other classification group. Neither number alone answers
whether sprinklers are required. Table files retain the other rows and exclusions.

## Still unverified

No licensed Australian Standard was available in the supplied material.
Its clause numbers, thresholds and tables stay unverified. No FRL matrix,
sprinkler trigger, egress-distance, sanitary fixture, insulation, acoustic,
BASIX, EV, submetering or accessible-lift lookup has been published as verified.
Missing derivations are explicitly `pending: true`; the code must return unknown.

The national adopted 2022 text is a versioned knowledge source, not a claim
that it governs every project today. Project edition, jurisdiction, scope and
approved compliance pathway must be established separately.
