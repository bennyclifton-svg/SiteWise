# Construction cost split by element: research

Date: 21 October 2026. Compiled from four web-research passes for
`docs/plans/2026-10-21-source-to-cost-plan-design.md` (Part 2, budget
spread). Not reviewed by the owner or a QS. Every split here is a planning
default for an editable suggestion, never an estimate, and stays
`verified: false` until checked against Rawlinsons, Cordell, AIQS or BCIS
(all paywalled) or real Australian cost plans.

Lines (cost plan construction elements): 00 Preliminaries and margin;
01 Site works and external works; 02 Substructure; 03 Structure;
04 Building envelope; 05 Interiors and finishes; 06 Passive fire; 07 Active
fire; 08 Mechanical; 09 Hydraulic; 10 Electrical; 11 Communications and
security; 12 Vertical transport; 13 Access and egress. Contingency, fees,
landscaping, land and GST are outside the split.

Mapping conventions shared by the sources: passive fire is rarely priced on
its own (it sits in walls and doors); AIQS puts internal walls in 05, so
03 holds frame, columns, upper floors; stairs and balustrades are 13.

## Confidence by sector

| Sector | Basis | Confidence |
|---|---|---|
| Residential | 12 registered-QS elemental estimates on the NSW Planning Portal, recomputed | Medium: real Australian estimates, small sample, DA estimates not tenders |
| Industrial | Two data-centre splits (second-hand), US cold-store vendor ranges, NSW rates; the rest judgement | Low |
| Institutional | UK NHS elemental models (2010), one NSW school estimate (2018), UK school benchmark (2025); the rest judgement | Low |
| Infrastructure | TfNSW strategic rates (roads, 2025); AEMO/Aurecon energy shares; rail from Transit Costs Project | Medium for roads, low otherwise; building lines fit poorly |
| Commercial / mixed | UK Building/AECOM cost models, AIQS 2020 high-rise excerpts, Australian fit-out guides | Low to medium; fit-out sources disagree 2.5× on services |

## Residential (% for 00/01/02/03/04/05/06/07/08/09/10/11/12/13)

- **House, new:** 18 / 5 / 11 / 6 / 24 / 24 / 0 / 0.2 / 2.5 / 4.5 / 3.8 / 0.5 / 0 / 0.5
  (H1 Earlwood 21.6/11.1*/9.3/4.6/22.8/20.5/0/0.1/1.6/4.6/3.8/0/0/0, *01 includes some landscaping;
  H2 Panania 16.2/4.5/11.2/4.8/23.3/29.3/0/0.1/1.7/3.0/3.4/0/0/2.7; A East Hills with basement 02 = 22.7;
  C Bankstown duplex 00 = 23.6). 11 is judgement.
- **Townhouses / attached:** borrow house. B Condell Park dual occupancy
  5.5/7.6/18.4/11.1/23.0/21.7/0/0.3/2.7/4.5/4.1/0/0/1.1 (developer-builder, low 00).
- **Apartments, high-rise with basement:** 19 / 2 / 9 / 18 / 11 / 22.3 / 0 / 2.8 / 2.7 / 5.5 / 4.5 / 0.2 / 2.2 / 0.8
  (R1 Belmore 184 units 19.6/2.8/10.2/12.3/11.7/22.6/0/2.9/2.6/7.9/6.0/0/1.5/0;
  R2 St Leonards 245 units 18.2/1.0/7.7/27.9/10.2/19.8/0/2.7/2.8/2.9/3.2/0.2/2.8/0.8).
- **Apartments, low-rise (≤ 4 storeys):** no source; interpolate house ↔ high-rise.
- **Build-to-rent:** no source; borrow high-rise.
- **Student housing / co-living:** S1 Todman 13.3/2.2/3.8/9.9/25.5/21.6/0/1.6/4.2/7.6/3.9/0.3/5.2/0.7;
  S2 Rockdale 157 rooms 23.4/0.8/2.9/10.8/10.6/30.7/0/3.0/2.2/8.0/6.5/0/1.2/0.
- **Retirement living:** RV Fern Bay uses coarse groups (civil 5.7, structure-envelope 26.9, roof 6.4,
  windows/internal 11.5, finishes 11.2, fitments 14.8, services 22.4); use house for villas, low-rise
  for apartment buildings.
- **Residential aged care:** AC Denham Court 16.8/3.9/5.3/7.2/17.6/28.1/0/3.2/5.9/5.5/4.5/0.8/0.9/0.2.
- **Social / affordable:** no source; borrow by building form.
- **Extension / addition:** X1 Ashbury 10.0/7.2/20.8/0/29.4/23.7/0/0.4/1.6/3.9/2.4/0/0/0.7.
- **Refurbishment:** no source.

Drivers (from the reports): a basement roughly doubles 02 (house 9–11 → 22.7);
storeys move share from 01/04 to 03, and add 07 (~3) and 12 (1.5–3); crane and
hoist add ~5.6 to 00; developer-builder pricing puts 00 at ~5 against 16–24 for
full contract pricing; an extension raises 02 and 04 and drops 03. Reactive or
rock ground, slope, quality, BAL and heritage: no numbers found (several reports
exclude rock).

Sources: NSW Planning Portal public documents, prefix
`https://apps.planningportal.nsw.gov.au/prweb/PRRestService/DocMgmt/v1/PublicDocuments/DATA-WORKATTACH-FILE%20PEC-DPE-EP-WORK%20`
+ ID + `%20GMT`: A East Hills, PBA, Feb 2024 `PAN-517417!20250310T042653.445`;
B Condell Park, PBA, Jul 2022 `PAN-246083!20220721T004430.575`; C Bankstown,
Duo QS, Apr 2025 `PAN-546522!20250620T071453.357`; H1 Earlwood, QPC&C, 2024
`PAN-518374!20250313T004534.885`; H2 Panania, CPP QS, 2025
`PAN-525221!20250404T002034.270`; X1 Ashbury, PBA, Feb 2025
`PAN-516714!20250306T221850.697`; R1 Belmore, QPC&C, Mar 2019
`PPS-2019SSH007!20190805T014219.479`; R2 St Leonards South, Quantex, Apr 2022
`PPSSNH-316!20230206T024906.152`; S1 Scape Todman, Feb 2023
`PAN-270199!20230326T215140.792`; S2 Rockdale, QPC&C, Sep 2024
`PPSSEC-341!20250625T061834.178`; RV Fern Bay, Corbett Scott, Dec 2024
`PPSHCC-274!20250325T062528.970`; AC Denham Court, Napier & Blakeley, Nov 2018
`PPS-2019WCI038!20200224T230308.253`. Cross-check only: NAHB, "Cost of
Constructing a Home in 2024", eyeonhousing.org, Jan 2025.

## Commercial and mixed use (mostly UK models, mapped)

| Line | Office high-rise | Office low-rise | Shopping centre | Standalone retail | Hotel (+ residential) | Office fit-out |
|---|---|---|---|---|---|---|
| 00 | 18 | 16 | 20 | 16 | 25 | 14 |
| 01 | 1 | 8 | 3 | 10 | 1 | 0 |
| 02 | 5 | 5 | 11 | 6 | 5 | 0 |
| 03 | 19 | 14 | 16 | 6 | 6 | 0 |
| 04 | 14 | 15 | 27 | 20 | 13 | 0 |
| 05 | 14 | 14 | 6 | 14 | 26 | 50 |
| 06 | 1 | 1 | 1 | 0 | 0 | 1 |
| 07 | 2 | 3 | 2 | 3 | 1 | 4 |
| 08 | 8 | 11 | 6 | 12 | 8 | 12 |
| 09 | 3 | 2 | 1 | 1 | 2 | 2 |
| 10 | 7 | 7 | 4 | 8 | 7 | 11 |
| 11 | 2 | 2 | 1 | 3 | 3 | 5 |
| 12 | 5 | 1 | 1 | 0 | 2 | 0 |
| 13 | 1 | 1 | 1 | 1 | 1 | 1 |

Borrowing: food and beverage ← standalone retail; serviced office/coworking ←
fit-out (refurb) or office (new); retail fit-out ← office fit-out;
residential over retail and hotel + residential ← hotel; other mixes blend
component columns by GFA (method, not sourced).

Drivers: high-rise raises 03 (16 → 18) and lowers 02 (3 → 1) [E]; tower lifts
2–3× standard [A]; basement 02 1–3 → 5–6.4 [E][A][B]; facade 15–25 % and
double-skin ~1.5× unitised [M]; occupied refurbishment 00 12–20 against 8–12
new [L]; after-hours CBD can double 00 [G]; access constraints +10–20 % [L].

Sources: [A] Building/AECOM "Cost model: Commercial office towers", 2016;
[B] Building/AECOM/3DReid "Mixed-use hotel, residential and retail", Dec 2019;
[C] Building retail showroom cost model, 2007; [D] AECOM "Retail Evolution Cost
Model", Jan 2019; [E] AIQS *Built Environment Economist*, "The Hidden Cost of
High-Rise", Mar–May 2020 (search excerpts only); [F] Building "Cost model:
Office fit-out", Nov 2017; [G] Complete Office Fitouts (Apr 2026) and Niche
Advisory (Aug 2026) fit-out breakdowns; [H] Building "Services for tall
buildings", 2018 (MEP sub-split); [I] NSW Major Projects SSD-83069459 EDC
report, Dec 2025 (prelims 16 %, margin 5.5 %); [L] Plexa, Gladison; [M]
Building "Office design", 2004.

## Industrial (judgement anchored on sparse sources)

| Line | Warehouse | High-bay | Light mfg | Heavy mfg | Cold store | Data centre | DG store | Food proc. | GMP |
|---|---|---|---|---|---|---|---|---|---|
| 00 | 12 | 12 | 12 | 13 | 11 | 10 | 12 | 12 | 12 |
| 01 | 20 | 17 | 15 | 13 | 10 | 3 | 20 | 10 | 6 |
| 02 | 14 | 14 | 13 | 16 | 12 | 2 | 15 | 11 | 6 |
| 03 | 14 | 16 | 13 | 17 | 10 | 3 | 12 | 10 | 7 |
| 04 | 14 | 15 | 13 | 11 | 20 | 3 | 12 | 13 | 7 |
| 05 | 6 | 4 | 8 | 5 | 2 | 1 | 3 | 12 | 16 |
| 06 | 1 | 1 | 1 | 1 | 1 | 1 | 3 | 1 | 2 |
| 07 | 6 | 8 | 5 | 4 | 5 | 2 | 9 | 4 | 3 |
| 08 | 3 | 3 | 6 | 6 | 22 | 22 | 4 | 14 | 23 |
| 09 | 2 | 2 | 4 | 3 | 1 | 1 | 3 | 6 | 7 |
| 10 | 6 | 6 | 8 | 9 | 5 | 50 | 5 | 6 | 9 |
| 11 | 1 | 1 | 1 | 1 | 1 | 2 | 1 | 1 | 2 |
| 12 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| 13 | 1 | 1 | 1 | 1 | 0 | 0 | 1 | 0 | 0 |

Anchors: external works 20–35 % of building cost (Blackark, "Warehouse
Construction Costs NSW 2026"); data centre electrical 54 / mechanical 22 /
shell 14 / fees 10, liquid-cooled 48 / 33 / 9 / 10 (Turner & Townsend via Axis
Intelligence, second-hand); cold store refrigeration 18–40, panels 12–25, slab
6–12 (US vendor guides: Kafafab, National Steel Buildings, ColdMatch); NSW
cold-store rates (Blackark; TMX Transform via eCommerceNews, Jul 2026); regional
+5–10, remote +20–40 (Asset Building 2026); GMP cleanroom and validation 35–55
(iFactory, vendor). Refurbishment: no source. Drivers: hardstand ratio → 01;
clear height → 03/04/07; office share → 05; temperature → 08/04/02; MW → 10.

## Institutional (judgement anchored on UK models and one NSW school)

| Line | Early childhood | School | Tertiary / lab | Hospital | Medical centre | Allied health | Civic | Religious |
|---|---|---|---|---|---|---|---|---|
| 00 | 13 | 12 | 13 | 15 | 15 | 15 | 13 | 13 |
| 01 | 14 | 10 | 4 | 4 | 5 | 6 | 7 | 8 |
| 02 | 7 | 6 | 4 | 4 | 5 | 6 | 5 | 6 |
| 03 | 5 | 12 | 9 | 8 | 8 | 6 | 10 | 12 |
| 04 | 17 | 18 | 11 | 12 | 15 | 15 | 18 | 24 |
| 05 | 16 | 15 | 12 | 15 | 16 | 20 | 18 | 17 |
| 06 | 1 | 1 | 1.5 | 1.5 | 1 | 1 | 1 | 1 |
| 07 | 1.5 | 1.5 | 2.5 | 2.5 | 2 | 1.5 | 2 | 1.5 |
| 08 | 9 | 8 | 18 | 12 | 11 | 10 | 10 | 7 |
| 09 | 7 | 4 | 7 | 10 | 7 | 6 | 4 | 2 |
| 10 | 6 | 7 | 10 | 9 | 9 | 8 | 7 | 6 |
| 11 | 2.5 | 3.5 | 4 | 4 | 3 | 3 | 3 | 1.5 |
| 12 | 0 | 1 | 2 | 2 | 2 | 1 | 1 | 0 |
| 13 | 1 | 1 | 2 | 1 | 1 | 1.5 | 1 | 1 |

Sources: NHS *Healthcare Premises Cost Guides* 2nd ed., DH Estates &
Facilities, Aug 2010 (inpatient and primary care elemental models,
https://www.england.nhs.uk/wp-content/uploads/2021/05/Healthcare-premises-cost-guides.pdf);
Wilde and Woollard, DA Budget Estimate, Good Samaritan College, Mar 2018, NSW
Planning Portal `PPS-2018WCI002!20190619T014914.394`; EBDOG/DfE *National
School Delivery Cost Benchmarking*, Jun 2025; lab MEP 40–55 % (BHDP;
Consulting-Specifying Engineer). Early childhood, civic and religious are
judgement.

## Infrastructure (own work breakdown)

Roads, % of infrastructure construction cost (TfNSW IP-0033-GD03 v3.0, Mar
2025, June 2024 rates): prelims and establishment 6 / 6 (urban / rural);
traffic management and temporary works 13 / 10; environment 4 / 4; drainage
10 / 8; earthworks 17 / 22; pavements 18 / 18; bridges, walls, ITS, lighting,
landscape (by difference) 32 / 32. On top: utilities +15–30 % urban, +5–10 %
rural; design 5–15 %; project management 5–20 %. Energy (Aurecon for AEMO,
2023): solar EPC equipment 60 / installation 40; onshore wind 75 / 25. Rail
(Transit Costs Project): stations 30–77 % of hard cost; systems ~22 %. Water,
substations, ports, airside, telecoms: no credible split. Contingency examples
(DITRDCA Guidance Note 3B, 2023) are client costs, outside the split.
Infrastructure needs its own line set; mapping it onto building elements is
judgement.

## Owner direction (21 October 2026)

"It doesn't have to be perfect, it just needs to be near enough." The split
must be informed by the profile: at minimum residential, commercial,
industrial, institutional and infrastructure, with the profile attributes
that drive it chosen by the build.
