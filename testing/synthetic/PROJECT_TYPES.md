# Project types (locked 28 October 2026)

The owner agreed these 20 types on 28 October 2026. They cover the most common
Australian jobs for project managers, architects and D&C builders. Each
type is one folder: `types/<id>-<slug>/`.

"App profile" gives the starting values. The domain map for each type
confirms them against `knowledge/profile/taxonomy.yaml`. "Built-in program
content" was counted from `knowledge/program/fragments.yaml` on 28 October
2026.

| ID | Slug | Project type | App profile (class / subclass / work type) | NCC | Typical size | Usual approval | Built-in program content |
|---|---|---|---|---|---|---|---|
| T01 | new-house | New detached house | residential / house / new | 1a, 10a | $0.5–3M | DA or CDC | Yes (1a) |
| T02 | apartments | Multi-residential apartments, 4–8 storeys | residential / apartments / new | 2, 7a | $10–60M | DA | Yes (2) |
| T03 | office-fitout | Commercial office fit-out | commercial / office / refurb | 5 | $0.3–5M | CDC or DA | Yes (fitout) |
| T04 | warehouse | Industrial warehouse | industrial / warehouse / new | 7b, 5 | $5–30M | DA | Yes (7b) |
| T05 | house-alterations-additions | House alterations and additions | residential / house / extend | 1a | $0.3–1.5M | DA or CDC | Partly (reno) |
| T06 | sprinkler-pump-tank-upgrade | Sprinkler pump and tank upgrade, with fit-out | commercial / office / refurb | 5 (existing) | $0.2–2M | CDC or exempt | No |
| T07 | mechanical-plant-upgrade | Mechanical plant upgrade (chillers, AHUs) | commercial / office / refurb | 5 (existing) | $0.5–5M | exempt or CDC | No |
| T08 | roof-remediation | Roof remediation, including asbestos roof replacement | industrial / warehouse / remediation | 7b (existing) | $0.1–2M | exempt or CDC | No |
| T09 | waterproofing-remediation | Waterproofing remediation (balconies, podium) | residential / apartments / remediation | 2 (strata) | $0.1–3M | exempt | No |
| T10 | knockdown-rebuild | Knock-down rebuild house | residential / house / new | 1a | $0.6–2.5M | CDC or DA | Partly (demolition) |
| T11 | dual-occupancy-townhouses | Dual occupancy or townhouses | residential / townhouses / new | 1a attached | $1–5M | DA or CDC | Mostly (1a) |
| T12 | granny-flat | Secondary dwelling (granny flat) | residential / house / new | 1a | $0.15–0.4M | CDC | Mostly (1a) |
| T13 | mixed-use-shop-top | Mixed-use shop-top housing | mixed / residential_retail / new | 2, 6, 7a | $15–80M | DA | Class 2 only |
| T14 | retail-cafe-fitout | Retail or café fit-out (kitchen, grease trap) | commercial / food_beverage / refurb | 6 | $0.1–1M | DA or CDC | Fit-out only |
| T15 | childcare-centre | Childcare centre | institution / education_early_childhood / new | 9b | $3–8M | DA | Yes (9b) |
| T16 | office-base-building-refurb | Office base-building refurbishment (lobby, amenities, end-of-trip) | commercial / office / refurb | 5 | $1–10M | CDC or DA | Partly |
| T17 | cladding-remediation | Façade or combustible cladding remediation | residential / apartments / remediation | 2 (strata) | $1–20M | DA or CDC | No |
| T18 | warehouse-extension-mezzanine | Warehouse extension or office/mezzanine fit-out | industrial / warehouse / extend | 7b, 5 (existing) | $1–8M | DA or CDC | Partly |
| T19 | lift-access-upgrade | Access upgrade: new lift and accessible amenities | commercial / office / refurb | 5 (existing) | $0.5–3M | CDC or DA | No |
| T20 | school-classroom-block | School classroom block (government) | institution / education_primary_secondary / new | 9b | $5–30M | REF or SSD | Yes (9b) |

The pilot is T01, T05 and T09:

- T01 is the clean baseline.
- T05 tests interfaces with an existing building.
- T09 tests remediation, where the built-in templates have gaps, along with
  its keyword traps.

The default state is NSW. Other states appear only as variants.
