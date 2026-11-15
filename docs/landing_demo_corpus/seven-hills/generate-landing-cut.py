#!/usr/bin/env python3
"""Write the landing cut of the Seven Hills corpus.

The landing page shows one project being built up in three drops. This
script writes those drops into ``landing-cut/``:

    1-project-folder/   the folder a PM already has: brief, planning, reports,
                        approvals, a specification and a drawing set
    2-fee-proposals/    three fee proposals for each of five consultants
    3-invoices/         three tax invoices from each appointed consultant

It keeps the project, firms and fees of the full corpus, and differs from
it in three ways, each because of how SiteWise reads Markdown
(testing/synthetic/DISCOVERY.md and briefs/04-render.md):

* Number, revision, title and date are ``Label: value`` lines before the
  first ``##`` heading. The app reads those and skips a title block laid
  out as a table.
* Every document states the approved scheme (11 townhouses). The full
  corpus carries the older 12-dwelling position on purpose; here it would
  only put a conflict on the profile.
* Reports say what a report says. The full corpus's reports are
  document-control stubs, which give the profile nothing to read.

Everything is invented: the address, land, firms, people, ABNs, numbers and
advice. Run from anywhere:  python generate-landing-cut.py
"""

from __future__ import annotations

import shutil
from pathlib import Path

ROOT = Path(__file__).resolve().parent / "landing-cut"

ADDRESS = "14–18 Wianamatta Avenue, Seven Hills NSW 2147"
PROJECT = f"11 townhouses and Torrens subdivision, {ADDRESS}"
CLIENT = "Wianamatta Developments Pty Ltd"
PM = "Ridgeline Project Management Pty Ltd"
LOTS = "Lots 41, 42 and 43 DP 1187426"
DA = "DA/2025/0733"


def abn(seed: int) -> str:
    """An invented ABN that passes the checksum, so no invoice is flagged for it."""
    weights = (10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19)
    body = 11_000_000_000 + (seed * 736_842_101) % 88_000_000_000
    for candidate in range(body, body + 20000):
        digits = [int(c) for c in f"{candidate:011d}"]
        digits[0] -= 1
        if digits[0] >= 0 and sum(d * w for d, w in zip(digits, weights)) % 89 == 0:
            s = f"{candidate:011d}"
            return f"{s[:2]} {s[2:5]} {s[5:8]} {s[8:]}"
    raise AssertionError("no ABN found")


def money(amount: int) -> str:
    return f"${amount:,}"


def cents(amount: int) -> str:
    return f"${amount:,.2f}"


def write(folder: str, name: str, text: str) -> None:
    path = ROOT / folder / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text.strip() + "\n", encoding="utf-8", newline="\n")


def head(title: str, number_label: str, number: str, revision: str, date: str) -> str:
    lines = [f"Title: {title}"]
    if number:
        lines.append(f"{number_label}: {number}")
    if revision:
        lines.append(f"Revision: {revision}")
    lines.append(f"Date: {date}")
    return "\n".join(lines) + f"\n\n# {title}\n"


# ---------------------------------------------------------------------------
# Firms
# ---------------------------------------------------------------------------

FIRMS = {
    "axis": ("Axis Studio Pty Ltd", "Studio 3, 18 Railway Parade, Blacktown NSW 2148", "Mei Chen, Director", "mei.chen@axisstudio.example.com", "02 9831 4470"),
    "civic": ("Civic Pattern Planning Pty Ltd", "Suite 8, 44 Flushcombe Road, Blacktown NSW 2148", "Ana Serrano, Director", "ana.serrano@civicpattern.example.com", "02 9622 1185"),
    "northline": ("Northline Structures Pty Ltd", "Suite 2, 12 Main Street, Blacktown NSW 2148", "Daniel Okafor, Principal", "daniel.okafor@northline.example.com", "02 9831 6620"),
    "catchment": ("Catchment Works Pty Ltd", "Unit 5, 9 Sunnyholt Road, Blacktown NSW 2148", "Sibusiso Moyo, Director", "s.moyo@catchmentworks.example.com", "02 9674 3312"),
    "flux": ("Flux Services Pty Ltd", "Suite 1, 72 Third Avenue, Blacktown NSW 2148", "Priya Raman, Director", "priya.raman@fluxservices.example.com", "02 9674 8054"),
    "terraforma": ("TerraForma Geotechnics Pty Ltd", "Unit 2, 31 Garling Road, Kings Park NSW 2148", "Isaac Mensah, Principal Geotechnical Engineer", "i.mensah@terraforma.example.com", "02 9674 2290"),
    "certus": ("Certus Building Surveying Pty Ltd", "Suite 4, 60 Kildare Road, Blacktown NSW 2148", "Rachel Wallace, Registered Certifier", "r.wallace@certus.example.com", "02 9622 7741"),
    "measureline": ("Measureline Quantity Surveying Pty Ltd", "Suite 30, 110 George Street, Parramatta NSW 2150", "Balwinder Kaur, Associate", "b.kaur@measureline.example.com", "02 9891 5520"),
    "greenfold": ("Greenfold Landscape Architects Pty Ltd", "Shop 2, 5 Windsor Road, Baulkham Hills NSW 2153", "Tom Halloran, Director", "tom@greenfold.example.com", "02 9674 6103"),
    "datum": ("Datum Line Surveyors Pty Ltd", "12 Campbell Street, Blacktown NSW 2148", "Grace Lindqvist, Registered Surveyor", "grace@datumline.example.com", "02 9622 3390"),
    "wayline": ("Wayline Traffic Engineering Pty Ltd", "Suite 50, 20 Smith Street, Parramatta NSW 2150", "Omar Haddad, Director", "omar@wayline.example.com", "02 9891 2274"),
    "stillwater": ("Stillwater Acoustics Pty Ltd", "Suite 11, 2 Sentry Drive, Parklea NSW 2768", "Helen Park, Senior Acoustic Engineer", "helen.park@stillwater.example.com", "02 9629 4418"),
    "canopy": ("Canopy Tree Consulting Pty Ltd", "7 Orchard Road, Baulkham Hills NSW 2153", "Rob Maddern, Consulting Arborist (AQF 5)", "rob@canopytree.example.com", "02 9624 0917"),
    "binley": ("Binley Waste Planning Pty Ltd", "Unit 9, 14 Forge Street, Blacktown NSW 2148", "Kate Binley, Director", "kate@binley.example.com", "02 9674 1166"),
    "openpath": ("Open Path Access Consulting Pty Ltd", "Suite 2, 88 Phillip Street, Parramatta NSW 2150", "Jo Tran, Accredited Access Consultant", "jo.tran@openpath.example.com", "02 9891 7702"),
    "kelvin": ("Kelvin Energy Assessors Pty Ltd", "Suite 1, 3 Patrick Street, Blacktown NSW 2148", "Nadia Farouk, Accredited Assessor", "nadia@kelvinenergy.example.com", "02 9622 5529"),
}
ABNS = {key: abn(i + 11) for i, key in enumerate(FIRMS)}


def letterhead(key: str) -> str:
    name, address, contact, email, phone = FIRMS[key]
    return f"{name}\nABN {ABNS[key]}\n{address}\nContact: {contact}\n{email} | {phone}"


# ---------------------------------------------------------------------------
# Drop 1: the project folder
# ---------------------------------------------------------------------------

FOLDER = "1-project-folder"

REPORTS: list[tuple[str, str, str, str, str, str, str, str]] = []  # file, title, label, number, rev, date, firm, body


def report(file: str, title: str, label: str, number: str, rev: str, date: str, firm: str, body: str) -> None:
    REPORTS.append((file, title, label, number, rev, date, firm, body))


report("Client development brief.md", "Client development brief", "Ref", "WD-BRIEF-01", "B", "10 February 2025", "", f"""
{CLIENT}
Prepared by: Michael Kwan, Development Director
Project: {PROJECT}

## Purpose

{CLIENT} will design, approve and build a townhouse project on three adjoining lots at {ADDRESS}. This brief sets out what the client wants built, what it will spend and how the project will be delivered. {PM} is the client's project manager and will act as Superintendent under the building contract.

## The site

- {LOTS}, to be consolidated and then subdivided
- Site area 3,240 m², with a 54 m frontage to Wianamatta Avenue
- Three existing fibro cottages and sheds, all to be demolished
- A 3.0 m wide drainage easement along the rear boundary
- Local government area: Blacktown City Council

## What is to be built

- New build of 11 attached two storey townhouses, with a Torrens title subdivision into 11 lots
- 8 three-bedroom townhouses and 3 four-bedroom townhouses
- A garage to every townhouse and 3 visitor car spaces
- No basement. Private open space to every dwelling
- One townhouse (Townhouse 1) designed as an adaptable dwelling
- Brick veneer to the lower storey, lightweight cladding above, metal roof
- All-electric dwellings: heat pump hot water, ducted air conditioning, solar panels and an EV-ready circuit in every garage. No gas connection

## Budget and delivery

- Construction budget: $9,800,000 excluding GST
- Procurement: traditional lump sum. The design is fully documented and tendered to builders as a construct-only contract
- Form of contract: AS 4000-1997, with {PM} as Superintendent
- The consultants are engaged directly by the client. No novation to the builder
- Quality level: mid-market townhouses for sale, with durable low-maintenance finishes

## Program

- Lodge the development application: August 2025
- Start construction: 5 May 2026
- Practical completion: June 2027
- Defects liability period: 12 months

## Approvals

A development application to Blacktown City Council is required. A registered certifier will issue the construction certificate and act as principal certifier.
""")

report("Planning certificate s10.7 - Lots 41-43 DP 1187426.md", "Planning certificate under section 10.7", "Ref", "PC2025/01882", "", "18 February 2025", "", f"""
Blacktown City Council
Applicant: {CLIENT}
Property: {ADDRESS}
Land: {LOTS}

## Zoning

The land is zoned R3 Medium Density Residential under Blacktown Local Environmental Plan 2015. Townhouses (multi dwelling housing) are permitted with development consent.

## Development controls

- Maximum height of buildings: 9 m
- The land is not within a heritage conservation area and no heritage item is on the land
- The land is not bushfire prone land
- The land is not within the flood planning area
- The land is not affected by acid sulfate soils mapping
- The land is not identified as contaminated land and no notice under the Contaminated Land Management Act 1997 applies
- The land is not affected by a road widening or realignment proposal
- The land is not in a mine subsidence district

## Contributions

A section 7.11 contributions plan applies to the land.

## Complying development

Complying development under the Housing Code may be carried out on the land. Townhouses with Torrens subdivision at this scale require a development application.
""")

report("Title search and deposited plan summary.md", "Title search and deposited plan summary", "Ref", "DP1187426", "", "11 February 2025", "", f"""
Prepared for: {CLIENT}
Property: {ADDRESS}

## Land

| Lot | Deposited plan | Area | Frontage |
|---|---|---|---|
| 41 | DP 1187426 | 1,075 m² | 18.0 m |
| 42 | DP 1187426 | 1,080 m² | 18.0 m |
| 43 | DP 1187426 | 1,085 m² | 18.0 m |
| Total | | 3,240 m² | 54.0 m |

## Registered interests

- Easement to drain water 3.0 m wide along the rear boundary of all three lots, benefiting Blacktown City Council
- No covenant restricts the number of dwellings
- No caveat is recorded

## Notes

The three lots are held in separate titles by the registered proprietor, {CLIENT}. The lots are to be consolidated before the Torrens subdivision into 11 lots.
""")

report("Pre-DA meeting minutes.md", "Pre-DA meeting minutes", "Ref", "PDA-2025-0412", "", "9 April 2025", "", f"""
Blacktown City Council, Development Assessment
Project: {PROJECT}
Attendees: Council planner and development engineer; Michael Kwan ({CLIENT}); Ana Serrano (Civic Pattern Planning); Mei Chen (Axis Studio)

## Matters discussed

1. Council supports attached townhouses on the consolidated site in principle. A development application is required.
2. The scheme tabled showed 11 two storey townhouses in two rows with a central driveway.
3. Deep soil: 15% of the site area (486 m²) is to be provided clear of driveways and the detention tank.
4. Stormwater: on-site detention is required. The tank is to stay clear of the rear drainage easement.
5. Waste: bins are to be kept at each townhouse and presented at the kerb. The kerbside collection point is to be shown on the plans.
6. Parking: one garage per dwelling plus visitor parking, with all vehicles leaving in a forward direction.
7. Trees: an arborist report is required for the two street trees and the trees along the rear boundary.

## Documents Council asked for with the application

Statement of environmental effects, survey, architectural drawings with shadow diagrams, stormwater management report, traffic and parking assessment, arboricultural impact assessment, waste management plan, acoustic assessment, BASIX certificate, landscape plan and a draft subdivision plan.
""")

report("Feasibility cost and program advice.md", "Feasibility cost and program advice", "Ref", "MQS-250221-01", "A", "21 February 2025", "measureline", f"""
Project: {PROJECT}
Client: {CLIENT}

## Basis

An order of cost for 11 two storey townhouses on a 3,240 m² site, from the client development brief and the concept sketch. Total gross floor area 1,700 m².

## Order of cost

| Element | Amount (ex GST) |
|---|---|
| Demolition and site preparation | $310,000 |
| Substructure | $780,000 |
| Structure, envelope and roof | $3,420,000 |
| Internal finishes, joinery and fitments | $1,960,000 |
| Hydraulic, electrical and mechanical services | $1,480,000 |
| External works, stormwater and landscaping | $1,090,000 |
| Preliminaries and margin | $760,000 |
| **Construction cost** | **$9,800,000** |

The construction budget of $9,800,000 excluding GST is achievable for the brief. Consultant fees, authority fees and contributions are outside this figure.

## Program

- Design and development application: 9 months
- Construction documentation and tender: 5 months
- Construction: 58 weeks from 5 May 2026
""")

report("Detail and level survey report.md", "Detail and level survey report", "Document number", "SUR-001", "A", "3 March 2025", "datum", f"""
Project: {ADDRESS}
Client: {CLIENT}
Survey reference: DLS-25-0318

## Scope

A detail and level survey of {LOTS}, to Australian Height Datum, with boundaries defined from the deposited plan.

## Findings

- Site area by survey 3,240 m²
- The site falls about 1.6 m from the street (RL 47.9) to the rear boundary (RL 46.3)
- Three fibro cottages, two sheds and a garage stand on the land
- A 3.0 m wide drainage easement runs along the rear boundary. A 375 mm stormwater pipe was located inside it
- Sewer main along the rear boundary, with an inspection shaft on Lot 42
- Overhead power and NBN along Wianamatta Avenue. Water main in the northern footpath
- 14 trees were located, including 2 street trees

## Drawings

Sheet 1 of 2: detail and levels at 1:200. Sheet 2 of 2: boundary definition and easement.
""")

report("Geotechnical investigation report GEO-001.md", "Geotechnical investigation report", "Report no", "GEO-001", "B", "18 July 2025", "terraforma", f"""
Project: {PROJECT}
Client: {CLIENT}
Job number: TFG-25114

## Investigation

Six boreholes to 4.5 m and four test pits, with laboratory testing of shrink-swell and aggressivity.

## Ground conditions

- Fill: 0.3 m to 0.6 m generally, and 0.6 m to 1.4 m of uncontrolled fill across the rear third of the site
- Residual silty clay, stiff to very stiff, of medium to high plasticity
- Weathered shale from 2.6 m to 3.4 m depth (RL 45.2 to 45.8)
- No groundwater was met to the depth investigated

## Site classification

The site is classified Class M (moderately reactive) under AS 2870. Areas of uncontrolled fill deeper than 0.4 m are to be treated as Class P until the fill is removed or the footings are founded through it.

## Recommendations

- Footings: stiffened raft slabs on bored piers founded in the residual clay or shale, with an allowable bearing pressure of 250 kPa in stiff clay and 700 kPa in shale
- The detention tank is to be founded below the fill in natural material
- Excavation to 2.4 m for the detention tank can be done with a conventional excavator. Batter temporary cuts at 1 to 1
- Soils are non-aggressive to concrete and steel. The soil is not saline
- Earthquake site sub-soil class Ce
""")

report("Statement of environmental effects.md", "Statement of environmental effects", "Report no", "SEE-001", "D", "22 September 2025", "civic", f"""
Project: {PROJECT}
Applicant: {CLIENT}
Consent authority: Blacktown City Council

## The proposal

Demolition of three dwellings and outbuildings, consolidation of {LOTS}, construction of 11 attached two storey townhouses with garages, a common driveway, landscaping and a below-ground on-site detention tank, and Torrens title subdivision into 11 lots. The application is a development application under Part 4 of the Environmental Planning and Assessment Act 1979.

## The site

The site is 3,240 m², zoned R3 Medium Density Residential under Blacktown Local Environmental Plan 2015. Townhouses (multi dwelling housing) are permitted with consent.

## Numerical summary

| Control | Required | Proposed |
|---|---|---|
| Height of buildings | 9 m maximum | 8.4 m |
| Storeys | 2 | 2 |
| Gross floor area | | 1,700 m² |
| Floor space ratio | | 0.52:1 |
| Deep soil | 486 m² (15%) | 512 m² |
| Private open space | 40 m² per dwelling | 42 m² to 68 m² |
| Resident parking | 1 space per dwelling | 11 garages and 11 driveway spaces |
| Visitor parking | 3 spaces | 3 spaces |

## Dwelling mix

8 three-bedroom townhouses of 148 m² each and 3 four-bedroom townhouses of 172 m² each. Townhouse 1 is an adaptable dwelling.

## Assessment

The site is not bushfire prone, not flood affected and not a heritage item or in a conservation area. The proposal complies with the height, deep soil, parking and private open space controls. Stormwater is detained on site in a 120 m³ tank clear of the rear drainage easement. Estimated cost of works $9,800,000 excluding GST.
""")

report("BASIX certificate 1791184M_02.md", "BASIX certificate", "Ref", "1791184M_02", "", "4 February 2026", "kelvin", f"""
Project: {PROJECT}
Local government area: Blacktown City Council
Dwellings: 11 attached townhouses

## Result

| Target | Required | Achieved |
|---|---|---|
| Water | 40 | 43 |
| Thermal performance | Pass | Pass |
| Energy | 72 | 78 |

A BASIX certificate is required for this development. The commitments below must be shown on the construction certificate drawings.

## Commitments for every dwelling

- Rainwater tank of 3,000 L collecting from at least 80 m² of roof, connected to the toilets, laundry and one garden tap
- 4 star showerheads, toilets, kitchen and basin taps
- Electric heat pump hot water system
- Ducted reverse cycle air conditioning with day and night zoning, EER 3.5 or better
- Induction cooktop and electric oven. No gas appliances
- Photovoltaic system of 5.0 kW on each roof
- LED lighting throughout
- Ceiling insulation R5.0, wall insulation R2.5 and sarking to the roof
- Aluminium framed double glazed windows
""")

report("NatHERS and sustainability report ESD-001.md", "NatHERS and sustainability report", "Report no", "ESD-001", "C", "30 January 2026", "kelvin", f"""
Project: {PROJECT}
Client: {CLIENT}

## Thermal performance

All 11 dwellings were modelled in accredited NatHERS software. The site is in NCC climate zone 6 (NatHERS climate zone 28, Richmond).

| Dwelling type | Number | Star rating |
|---|---|---|
| Three-bedroom, mid-row | 6 | 7.4 |
| Three-bedroom, end of row | 2 | 7.1 |
| Four-bedroom, end of row | 3 | 7.0 |

The lowest rating is 7.0 stars and the average is 7.2 stars, which meets the 7 star minimum of NCC 2022.

## Whole-of-home

Each dwelling is all-electric with a heat pump hot water system, ducted reverse cycle air conditioning, an induction cooktop and a 5.0 kW photovoltaic system. Each garage has a dedicated 32 A circuit for a future electric vehicle charger.

## Specification the ratings rely on

R5.0 ceiling batts, R2.5 wall batts, double glazed aluminium windows (U 3.1, SHGC 0.49), light coloured metal roof, and sealed exhaust fans.
""")

report("NCC 2022 compliance report BCA-001.md", "NCC 2022 compliance report", "Report no", "BCA-001", "C", "6 February 2026", "certus", f"""
Project: {PROJECT}
Client: {CLIENT}
Job number: CBS-26008

## Classification

- The 11 townhouses are Class 1a dwellings, each separated from the next by a fire-resisting separating wall
- The attached garages are Class 10a
- Rise in storeys: 2
- The assessment is against NCC 2022 Volume Two and the ABCB Housing Provisions, using the deemed-to-satisfy pathway

## Design inputs

- Wind classification N2
- Site classification M under AS 2870
- Climate zone 6
- The land is not bushfire prone, so no bushfire attack level applies
- Termite management is required: the site is in a termite prone area

## Findings

1. Separating walls between dwellings are to achieve FRL 60/60/60 and run to the underside of the roof covering.
2. Smoke alarms are to be interconnected and hard wired, on both storeys of every dwelling.
3. Wet areas are to be waterproofed to AS 3740.
4. Stair, balustrade and barrier dimensions comply.
5. Energy efficiency is met through the NatHERS certificates (7.0 stars minimum) and the BASIX certificate.
6. Townhouse 1 meets the livable housing design requirements, with a step-free entry and an accessible toilet at entry level.

No performance solutions are required.
""")

report("Stormwater management report SWM-001.md", "Stormwater management report", "Report no", "SWM-001", "C", "20 February 2026", "catchment", f"""
Project: {PROJECT}
Client: {CLIENT}
Job number: CW-26012

## Existing drainage

The site falls to the rear, where a Council 375 mm pipe runs in a 3.0 m drainage easement. The lawful point of discharge is the Council pit in the easement at the north-east corner.

## On-site detention

- Permissible site discharge 62 L/s
- Required detention storage 120 m³, provided in a below-ground reinforced concrete tank under the common driveway
- The tank is 16.0 m by 5.0 m by 1.8 m deep internally, clear of the easement and the tree protection zones
- Outlet control by an orifice plate in a discharge control pit

## Water quality and reuse

- A 3,000 L rainwater tank to each townhouse, reused for toilets, laundry and garden
- A gross pollutant trap and filter cartridges in the discharge control pit

## Pipes and pits

Roof water drains to the rainwater tanks and overflows to the detention tank. Driveway runoff is collected in grated trench drains and pits. Subsoil drainage is provided behind the retaining walls.

## Erosion and sediment control

A sediment fence, a stabilised site access and a sediment basin are shown on drawing C-300.
""")

report("Traffic and parking assessment TRA-001.md", "Traffic and parking assessment", "Report no", "TRA-001", "B", "12 September 2025", "wayline", f"""
Project: {PROJECT}
Client: {CLIENT}

## Parking

| Use | Required | Provided |
|---|---|---|
| Resident | 11 spaces | 11 garages and 11 driveway spaces |
| Visitor | 3 spaces | 3 spaces |

## Access

One 6.0 m wide vehicle crossing to Wianamatta Avenue serves the common driveway. Swept paths show a B99 car and a small rigid vehicle entering and leaving in a forward direction. Sight distances at the crossing comply with AS 2890.1.

## Traffic generation

The 11 townhouses generate about 6 vehicle trips in the peak hour, a net increase of 4 over the three existing houses. The effect on Wianamatta Avenue is negligible.

## Construction traffic

A construction traffic management plan is to be prepared by the builder before work starts.
""")

report("Acoustic assessment ACO-001.md", "Acoustic assessment", "Report no", "ACO-001", "B", "5 September 2025", "stillwater", f"""
Project: {PROJECT}
Client: {CLIENT}

## Noise environment

Attended and unattended noise measurements were made over seven days. Wianamatta Avenue is a local road. The site is not near a rail corridor or a classified road, and is not a noise affected site.

## Findings

- Standard construction achieves the internal noise goals: 6.38 mm laminated glazing to the street-facing bedrooms and 4 mm float glass elsewhere
- Separating walls between townhouses are to achieve Rw + Ctr 50 or better
- Air conditioning outdoor units are to be located as shown on M-300, with a sound power level no greater than 65 dB(A), to meet the night-time criterion at the nearest neighbour

## Construction noise

Work hours and noise management are to follow the consent conditions.
""")

report("Arboricultural impact assessment ARB-001.md", "Arboricultural impact assessment", "Report no", "ARB-001", "B", "29 August 2025", "canopy", f"""
Project: {PROJECT}
Client: {CLIENT}

## Trees assessed

14 trees were assessed: 2 street trees, 4 trees along the rear boundary and 8 trees within the site.

## Recommendations

- Retain and protect the 2 street trees (Trees 1 and 2, Lophostemon confertus) and the 4 rear boundary trees (Trees 9 to 12)
- Remove 8 trees of low retention value within the building footprint
- Tree protection fencing and trunk protection are to be installed before demolition
- The detention tank and driveway have been located outside the tree protection zones of Trees 9 to 12
- 18 replacement trees are shown on the landscape plan

A project arborist is to inspect at three hold points: before demolition, during excavation near the rear boundary and at practical completion.
""")

report("Waste management plan WMP-001.md", "Waste management plan", "Report no", "WMP-001", "B", "10 September 2025", "binley", f"""
Project: {PROJECT}
Client: {CLIENT}

## Demolition and construction

- Three fibro cottages are to be demolished. A hazardous materials survey before demolition is required, as bonded asbestos cement sheeting is expected
- At least 80% of demolition and construction waste by weight is to be recycled

## Operation

- Each townhouse has a 240 L general waste bin, a 240 L recycling bin and a 240 L garden organics bin, stored in its courtyard or garage
- Bins are presented at the kerb on Wianamatta Avenue. The frontage of 54 m is long enough for 22 bins on collection day
- No communal bin room is required
""")

report("Access and adaptability assessment ACC-001.md", "Access and adaptability assessment", "Report no", "ACC-001", "B", "15 September 2025", "openpath", f"""
Project: {PROJECT}
Client: {CLIENT}

## Scope

A review of the development application drawings against the livable housing design requirements and Council's adaptable housing control.

## Findings

- Townhouse 1 is the adaptable dwelling, to AS 4299 Class C, with a continuous accessible path from the street
- Townhouse 1 has a step-free entry, a bedroom and an accessible bathroom at entry level, and wall reinforcement for grab rails
- The visitor parking includes 1 accessible space
- The Class 1a dwellings do not require a lift. No lift is proposed
""")

report("Notice of determination DA-2025-0733.md", "Notice of determination of a development application", "Ref", DA, "", "27 November 2025", "", f"""
Blacktown City Council
Applicant: {CLIENT}
Property: {ADDRESS} ({LOTS})

## Determination

Development consent is granted, subject to conditions, to development application {DA} for demolition of existing structures, construction of 11 attached two storey townhouses and Torrens title subdivision into 11 lots.

- Date of determination: 27 November 2025
- Consent operates from: 27 November 2025
- Consent lapses on: 27 November 2030

## Conditions (selected)

| No. | When | Requirement |
|---|---|---|
| 6 | Before the construction certificate | A current BASIX certificate for 11 dwellings |
| 9 | Before the construction certificate | Pay the section 7.11 contribution |
| 12 | Before works start | Tree protection installed and certified by the project arborist |
| 18 | Before occupation | Build and certify the 120 m³ on-site detention system |
| 23 | During works | No building work inside the rear drainage easement |
| 27 | During works | Work hours 7 am to 6 pm Monday to Friday and 8 am to 1 pm Saturday |
| 31 | Before the subdivision certificate | Complete all subdivision and authority works |
""")

report("Section 7.11 contribution notice.md", "Section 7.11 contribution notice", "Ref", "S711-2025-0733", "", "4 December 2025", "", f"""
Blacktown City Council
Development application: {DA}
Property: {ADDRESS}

## Contribution

The development of 11 dwellings replaces 3 existing dwellings, a net increase of 8 dwellings.

| Item | Amount |
|---|---|
| Open space and recreation | $98,400 |
| Community facilities | $31,200 |
| Roads and traffic | $46,800 |
| Plan administration | $3,600 |
| **Total contribution** | **$180,000** |

The contribution is payable before the construction certificate is issued and is indexed quarterly.
""")

report("Construction certificate CC-2026-0148.md", "Construction certificate", "Ref", "CC-2026/0148", "", "13 March 2026", "certus", f"""
Project: {PROJECT}
Applicant: {CLIENT}
Development consent: {DA}, Blacktown City Council, 27 November 2025

## Certificate

I certify that work completed in accordance with the documents listed below will comply with the requirements of the Environmental Planning and Assessment Regulation and the consent.

- Building classification: Class 1a (11 dwellings) and Class 10a (garages)
- NCC 2022 Volume Two
- Principal certifier: Rachel Wallace, Certus Building Surveying Pty Ltd

## Documents

Architectural drawings A-000 to A-900 revision D, structural drawings S-001 to S-202, civil drawings C-001 to C-300, the BASIX certificate 1791184M_02 and the NCC 2022 compliance report BCA-001 revision C.

## Critical stage inspections

Footings before concrete, slab reinforcement, framing, wet area waterproofing, stormwater before backfill and final.
""")

report("Pre-tender cost plan QS-001.md", "Pre-tender cost plan", "Report no", "QS-001", "C", "3 April 2026", "measureline", f"""
Project: {PROJECT}
Client: {CLIENT}
Job number: MQS-250221

## Summary

The pre-tender estimate for the construct-only contract is $9,800,000 excluding GST, measured from the construction certificate drawings. Gross floor area 1,700 m².

| Element | Amount (ex GST) |
|---|---|
| Preliminaries | $690,000 |
| Demolition and site preparation | $295,000 |
| Substructure: piers, slabs and retaining walls | $815,000 |
| Frame, upper floors and stairs | $1,240,000 |
| Roof | $545,000 |
| External walls, windows and doors | $1,610,000 |
| Internal walls, finishes, joinery and fitments | $1,985,000 |
| Hydraulic services | $560,000 |
| Electrical services and solar | $585,000 |
| Mechanical services | $330,000 |
| Stormwater and the detention tank | $285,000 |
| External works, driveway and landscaping | $520,000 |
| Margin | $340,000 |
| **Total** | **$9,800,000** |

## Exclusions

Consultant fees, authority fees and contributions, rock excavation, loose furniture and GST.

## Contract

AS 4000-1997 lump sum, construct only. Contract period 58 weeks. Defects liability period 12 months. Liquidated damages $4,500 per day.
""")

SPEC_SECTIONS = [
    ("Demolition", "Demolish the three cottages, sheds, slabs and paving. Remove asbestos cement sheeting by a licensed removalist under the hazardous materials survey."),
    ("Earthworks", "Strip topsoil, remove uncontrolled fill under the buildings and tank, and trim to the levels on C-100."),
    ("Piers and footings", "Bored concrete piers to the engineer's schedule, founded in stiff clay or shale."),
    ("Ground slabs", "Reinforced concrete stiffened raft slabs, 32 MPa, on a 0.2 mm vapour barrier, with termite collars to penetrations."),
    ("Wall framing", "Prefabricated MGP10 timber wall frames, H2 termite treated. Timber I-joist upper floor with 19 mm particleboard flooring."),
    ("Roof framing", "Prefabricated timber roof trusses at 600 mm centres."),
    ("Separating walls", "Twin timber stud fire and sound rated separating walls, FRL 60/60/60, carried to the underside of the roof sheeting."),
    ("Masonry", "Face brick veneer to the lower storey, 230 by 110 by 76 mm, with stainless steel ties."),
    ("Cladding", "Fibre cement weatherboard cladding to the upper storey on a ventilated cavity, paint finish."),
    ("Roofing", "Prefinished corrugated steel roof sheeting at 5 degrees, with matching gutters, downpipes and flashings, over R1.3 anticon blanket."),
    ("Windows and external doors", "Powdercoated aluminium framed double glazed windows and sliding doors. Solid core entry doors. Panel lift garage doors with motors."),
    ("Insulation", "R5.0 ceiling batts and R2.5 wall batts."),
    ("Waterproofing", "Liquid applied membranes to bathrooms, ensuites, laundries and balconies, to AS 3740 and AS 4654."),
    ("Linings", "10 mm plasterboard to walls and ceilings, water resistant board to wet areas, 90 mm cove cornice."),
    ("Joinery", "Laminate kitchens with 20 mm engineered stone benchtops, vanities, built-in wardrobes and laundry cupboards."),
    ("Floor finishes", "Porcelain tiles to the living areas and wet areas. Carpet on underlay to bedrooms and stairs. Sealed concrete to garages."),
    ("Painting", "Three coat low-VOC acrylic paint system to walls, ceilings and trim."),
    ("Hydraulic services", "Sanitary plumbing and drainage, cold and hot water, a 3,000 L rainwater tank with pump to each dwelling, and a 315 L heat pump hot water unit. No gas."),
    ("Electrical services", "A three phase supply and meter to each dwelling, switchboards, LED lighting, power, hard wired smoke alarms, a 5.0 kW photovoltaic system with inverter, and a 32 A circuit for a future electric vehicle charger."),
    ("Communications", "NBN fibre to each dwelling, with a home distributor and data points to living areas and bedrooms. Video intercom to each entry."),
    ("Mechanical services", "Ducted reverse cycle air conditioning with day and night zones. Ducted exhaust to bathrooms, laundries and rangehoods."),
    ("Appliances", "Induction cooktop, electric oven, rangehood and dishwasher to each kitchen."),
    ("Stormwater", "Pits, pipes, trench drains and a 120 m³ below-ground reinforced concrete on-site detention tank."),
    ("Driveway and paving", "Reinforced concrete common driveway with a broom finish, exposed aggregate crossing and paved courtyards."),
    ("Retaining walls and fencing", "Concrete sleeper retaining walls to 1.2 m. 1.8 m lapped and capped timber boundary fences and powdercoated aluminium front fences."),
    ("Landscaping", "Turf, mass planting, 18 trees, garden edging, drip irrigation and letterboxes."),
]

report("Outline specification SP-001.md", "Outline specification", "Document number", "SP-001", "C", "30 January 2026", "axis", f"""
Project: {PROJECT}
Client: {CLIENT}

This outline specification is read with the drawings. It lists the work in each section for all 11 townhouses.

""" + "\n\n".join(f"## {i:02d} {name}\n\n{text}" for i, (name, text) in enumerate(SPEC_SECTIONS, start=1)))

AREAS = [
    ("Townhouse 1", "4 bedroom, adaptable", 172, 212, 68),
    ("Townhouse 2", "3 bedroom", 148, 151, 42),
    ("Townhouse 3", "3 bedroom", 148, 151, 42),
    ("Townhouse 4", "3 bedroom", 148, 151, 42),
    ("Townhouse 5", "3 bedroom", 148, 151, 42),
    ("Townhouse 6", "4 bedroom", 172, 208, 61),
    ("Townhouse 7", "3 bedroom", 148, 163, 48),
    ("Townhouse 8", "3 bedroom", 148, 158, 46),
    ("Townhouse 9", "3 bedroom", 148, 158, 46),
    ("Townhouse 10", "3 bedroom", 148, 163, 48),
    ("Townhouse 11", "4 bedroom", 172, 224, 66),
]
assert sum(row[2] for row in AREAS) == 1700

report("Area and accommodation schedule A-SCH-01.md", "Area and accommodation schedule", "Document number", "A-SCH-01", "D", "30 January 2026", "axis", f"""
Project: {PROJECT}
Client: {CLIENT}

## Summary

- 11 attached two storey townhouses: 8 three-bedroom and 3 four-bedroom
- Total gross floor area 1,700 m² on a site of 3,240 m² (floor space ratio 0.52:1)
- 11 garages, 11 driveway spaces and 3 visitor spaces

## Schedule

| Dwelling | Type | Gross floor area | Lot area | Private open space |
|---|---|---|---|---|
""" + "\n".join(f"| {n} | {t} | {g} m² | {l} m² | {p} m² |" for n, t, g, l, p in AREAS) + """
| **Total** | | **1,700 m²** | | |

## Rooms in each townhouse

Lower storey: entry, living, dining, kitchen, laundry, powder room and garage. Upper storey: main bedroom with ensuite and robe, two further bedrooms (three in the four-bedroom type), bathroom and linen store. Townhouse 1 also has a bedroom and accessible bathroom at entry level.
""")

# Drawings: number, title, revision, date, issue, scale, shows
DRAWINGS = {
    "axis": ("Architectural", [
        ("A-000", "Cover sheet, locality plan and drawing register", "D", "30 January 2026", "1:1000", ["Locality plan and site address", "Drawing register for 20 architectural sheets", "BASIX commitments table", "General notes"]),
        ("A-100", "Proposed site plan", "D", "30 January 2026", "1:200", ["11 townhouses in two rows either side of the common driveway", "Setbacks: 6.0 m front, 4.5 m rear, 1.5 m sides", "The 3.0 m drainage easement and the detention tank under the driveway", "3 visitor spaces, letterboxes and the kerbside bin presentation area"]),
        ("A-101", "Ground floor plans, townhouses 1 to 6", "D", "30 January 2026", "1:100", ["Living, dining, kitchen, laundry, powder room and garage to each townhouse", "Townhouse 1 ground floor bedroom and accessible bathroom", "Separating walls and party wall types"]),
        ("A-102", "First floor plans, townhouses 1 to 6", "D", "30 January 2026", "1:100", ["Bedrooms, ensuites, bathrooms and linen stores", "Balconies to the street-facing main bedrooms", "Roof over the garages"]),
        ("A-110", "Subdivision and lot layout plan", "C", "22 September 2025", "1:200", ["Proposed Lots 1 to 11 with areas", "Easements for services and support", "The right of carriageway over the common driveway"]),
        ("A-200", "Elevations, north and south", "D", "30 January 2026", "1:100", ["Face brick ground floor and weatherboard cladding above", "Metal roof at 5 degrees behind a parapet to the street", "Maximum height 8.4 m above ground level"]),
        ("A-300", "Sections A-A and B-B", "D", "30 January 2026", "1:100", ["Floor to floor height 3.06 m and ceiling heights of 2.7 m and 2.55 m", "Slab, first floor and roof build-ups", "Detention tank under the driveway in section"]),
        ("A-400", "External finishes and materials schedule", "C", "22 September 2025", "NTS", ["Brick, cladding, roof, window and garage door colours", "Fence and paving types"]),
        ("A-401", "Door and window schedule", "D", "30 January 2026", "1:50", ["Aluminium framed double glazed windows by type and size", "Glazing type for each room, from the acoustic and NatHERS reports"]),
    ]),
    "northline": ("Structural", [
        ("S-001", "General notes, legend and design criteria", "C", "13 February 2026", "NTS", ["Design loads, wind classification N2 and site classification M", "Concrete, reinforcement, timber and steel specifications"]),
        ("S-100", "Footing, pier and retaining wall set-out", "C", "13 February 2026", "1:100", ["Bored piers at 450 mm diameter with founding levels", "Stiffened raft slab edge and internal beams", "Sleeper retaining walls to 1.2 m"]),
        ("S-101", "Ground floor slab and framing plan", "C", "13 February 2026", "1:100", ["Slab thickness, mesh and beam reinforcement", "Steel posts and lintels over garage openings"]),
        ("S-103", "Roof framing plan", "B", "12 December 2025", "1:100", ["Truss layout, bracing and tie-down", "Steel beams over the living room openings"]),
        ("S-202", "Detention tank base and wall reinforcement", "B", "13 February 2026", "1:50", ["300 mm base and 250 mm walls, N16 at 200 each way", "Trafficable lid slab under the driveway", "Founding level below the fill"]),
    ]),
    "catchment": ("Civil", [
        ("C-100", "Bulk earthworks and site grading plan", "C", "20 February 2026", "1:200", ["Cut and fill levels and finished surface levels", "Driveway grades and the crossing to Wianamatta Avenue"]),
        ("C-200", "Stormwater drainage plan", "C", "20 February 2026", "1:200", ["Pits, pipes, trench drains and overland flow path", "Connection to the Council pit in the easement"]),
        ("C-201", "On-site detention tank plan, sections and outlet details", "C", "20 February 2026", "1:50", ["120 m³ tank, 16.0 m by 5.0 m by 1.8 m", "Discharge control pit, orifice plate and filter cartridges"]),
        ("C-300", "Erosion and sediment control plan", "B", "12 December 2025", "1:200", ["Sediment fence, stabilised access and sediment basin"]),
    ]),
    "flux": ("Building services", [
        ("H-100", "In-ground sanitary drainage and sewer plan", "B", "27 February 2026", "1:200", ["Sewer connection to the main along the rear boundary", "Inspection openings and overflow relief gullies"]),
        ("H-200", "Cold water, hot water and rainwater schematics", "B", "27 February 2026", "NTS", ["Water meters, heat pump hot water units and tempering valves", "Rainwater tank, pump and mains top-up to each townhouse"]),
        ("E-100", "Site reticulation and external lighting plan", "B", "27 February 2026", "1:200", ["Underground consumer mains from the pillar to each meter", "Driveway bollard lighting and the NBN pit and pipe"]),
        ("E-200", "Switchboard schedule and single line diagram", "B", "27 February 2026", "NTS", ["Three phase supply, solar inverter and EV charger circuit to each dwelling"]),
        ("M-100", "Mechanical services plans", "B", "27 February 2026", "1:100", ["Ducted air conditioning layout, zones and outdoor unit positions", "Bathroom, laundry and rangehood exhaust"]),
    ]),
    "greenfold": ("Landscape", [
        ("L-100", "Landscape masterplan", "C", "30 January 2026", "1:200", ["512 m² of deep soil", "18 new trees and the 6 retained trees", "Courtyards, turf and planting beds"]),
        ("L-200", "Hardscape, fencing and retaining wall details", "B", "12 December 2025", "1:20", ["Paving, garden edging and letterbox details", "Timber boundary fences and aluminium front fences"]),
    ]),
}

ISSUE = {"D": "For construction", "C": "For construction", "B": "For tender", "A": "For approval"}
EARLIER = {"D": ["A", "B", "C"], "C": ["A", "B"], "B": ["A"], "A": []}
REV_NOTE = {"A": ("14 July 2025", "Issued for coordination"), "B": ("22 September 2025", "Issued for the development application"), "C": ("12 December 2025", "Issued for tender")}


def drawing_text(firm: str, discipline: str, d: tuple) -> str:
    number, title, rev, date, scale, shows = d
    name = FIRMS[firm][0]
    history = [(r, *REV_NOTE[r]) for r in EARLIER[rev]] + [(rev, date, ISSUE[rev])]
    return f"""Drawing number: {number}
Revision: {rev}
Title: {title}
Date: {date}

# {title}

{name}
Project: {PROJECT}
Client: {CLIENT}
Discipline: {discipline}
Scale: {scale} at A1
Issue: {ISSUE[rev]}

## Sheet shows

""" + "\n".join(f"- {s}" for s in shows) + f"""

## Notes

1. Do not scale from this drawing. Use written dimensions.
2. Read with the other consultants' drawings and the outline specification SP-001.
3. All work to the National Construction Code 2022 and the conditions of consent {DA}.

## Revisions

| Rev | Date | Description |
|---|---|---|
""" + "\n".join(f"| {r} | {when} | {what} |" for r, when, what in history)


def write_folder() -> int:
    count = 0
    for file, title, label, number, rev, date, firm, body in REPORTS:
        text = head(title, label, number, rev, date) + ("\n" + letterhead(firm) + "\n" if firm else "") + "\n" + body.strip()
        write(FOLDER, file, text)
        count += 1
    for firm, (discipline, sheets) in DRAWINGS.items():
        for d in sheets:
            write(FOLDER, f"{d[0]} {d[1]} Rev {d[2]}.md".replace(",", ""), drawing_text(firm, discipline, d))
            count += 1
    return count


# ---------------------------------------------------------------------------
# Drop 2: fee proposals. Drop 3: invoices.
# ---------------------------------------------------------------------------

BIDDER_FIRMS = {
    "lantern": ("Lantern Workshop Architects Pty Ltd", "Suite 10, 41 Church Street, Parramatta NSW 2150", "Hugo Lantern, Director", "hugo@lanternworkshop.example.com", "02 9891 3305"),
    "commonline": ("Commonline Architects Pty Ltd", "Suite 14, 3 Columbia Court, Norwest NSW 2153", "Farah Aziz, Principal", "farah@commonline.example.com", "02 8824 6610"),
    "parcel": ("Parcel Planning Pty Ltd", "8 Alpha Street, Blacktown NSW 2148", "Leo Marchetti, Director", "leo@parcelplanning.example.com", "02 9622 9043"),
    "borough": ("Borough Urban Planning Pty Ltd", "Suite 70, 91 Phillip Street, Parramatta NSW 2150", "Ingrid Solberg, Director", "ingrid@boroughurban.example.com", "02 9891 4487"),
    "spantheory": ("Span Theory Engineers Pty Ltd", "Unit 3, 26 Holbeche Road, Arndell Park NSW 2148", "Wei Zhang, Director", "wei@spantheory.example.com", "02 9674 5521"),
    "hardpoint": ("Hardpoint Engineering Pty Ltd", "Suite 40, 10 Valentine Avenue, Parramatta NSW 2150", "Sara Lindholm, Associate Director", "sara@hardpoint.example.com", "02 9891 8830"),
    "flowmark": ("Flowmark Civil Pty Ltd", "5 Vardys Road, Kings Park NSW 2148", "Ben Achterberg, Director", "ben@flowmarkcivil.example.com", "02 9674 7719"),
    "basinworks": ("Basinworks Engineering Pty Ltd", "Suite 6, 15 Lexington Drive, Bella Vista NSW 2153", "Thandi Nkosi, Principal", "thandi@basinworks.example.com", "02 8824 1906"),
    "circuit": ("Circuit and Air Consulting Pty Ltd", "Unit 12, 4 Tattersall Road, Kings Park NSW 2148", "Marco Ferri, Director", "marco@circuitandair.example.com", "02 9674 0238"),
    "gridworks": ("Gridworks Engineering Pty Ltd", "Suite 20, 150 George Street, Parramatta NSW 2150", "Alinta Brown, Director", "alinta@gridworks.example.com", "02 9891 6652"),
}
FIRMS.update(BIDDER_FIRMS)
ABNS.update({key: abn(i + 61) for i, key in enumerate(BIDDER_FIRMS)})

# Each discipline: file label, title wording, the stages its comparison has
# in SiteWise (knowledge/rfp/*.yaml) with the services each buys, exclusions,
# and three bidders (firm, reference, fee, stage shares, note). The
# appointed firm is listed second, as in the full corpus.
DISCIPLINES = [
    {
        "label": "Architectural services", "title": "Fee proposal: architectural services", "invoice_prefix": "AXS-INV",
        "stages": [
            ("Brief and concept", "Site analysis, brief review and concept options"),
            ("Schematic design", "Schematic design and area schedule"),
            ("Planning", "Development application drawings and lodgement"),
            ("Design development", "Developed design and consultant coordination"),
            ("Tender documentation", "Construction drawings, specification and tender set"),
            ("Delivery", "Site visits, contractor queries and defects inspections"),
        ],
        "exclusions": ["Fees of the other consultants", "Council fees, contributions and certifier fees", "Marketing images and sales plans"],
        "bidders": [
            ("lantern", "LW-26031", 248_000, (8, 14, 18, 18, 30, 12), "Six site visits are allowed during construction. Coordination of the waste collection vehicle path is excluded."),
            ("axis", "AXS-26017", 286_000, (8, 12, 18, 18, 30, 14), "Monthly site visits for the 58 week construction period are included."),
            ("commonline", "CLA-26044", 332_000, (7, 12, 16, 17, 30, 18), "Interior design and fortnightly site visits are included."),
        ],
    },
    {
        "label": "Town planning", "title": "Fee proposal: town planning services", "invoice_prefix": "CPP-INV",
        "stages": [
            ("Brief and concept", "Planning due diligence, pre-DA meeting and written strategy"),
            ("Planning", "Statement of environmental effects, lodgement and Council requests to determination"),
            ("Design development", "Consent condition schedule and subdivision certificate advice"),
        ],
        "exclusions": ["Land and Environment Court proceedings", "Modification applications after determination", "Council application fees"],
        "bidders": [
            ("parcel", "PP-26008", 38_000, (20, 62, 18), "One response to a Council request for information is allowed. Attendance at the pre-DA meeting is excluded."),
            ("civic", "CPP-26019", 48_000, (22, 58, 20), "All Council requests through to determination are included."),
            ("borough", "BUP-26027", 62_000, (20, 58, 22), "Neighbour consultation and a planning panel briefing are included."),
        ],
    },
    {
        "label": "Structural engineering", "title": "Fee proposal: structural engineering services", "invoice_prefix": "NS-INV",
        "stages": [
            ("Brief and concept", "Review of the geotechnical report and footing options"),
            ("Schematic design", "Schematic structure and framing concept"),
            ("Planning", "Structural statement for the development application"),
            ("Design development", "Developed design of footings, slabs and framing"),
            ("Tender documentation", "Structural drawings and specification for tender"),
            ("Design finalisation", "Construction certificate drawings and design certificate"),
            ("Delivery", "Hold point inspections and contractor queries"),
        ],
        "exclusions": ["Geotechnical investigation", "Temporary works design", "Shop drawings for trusses and frames"],
        "bidders": [
            ("spantheory", "ST-26015", 112_000, (6, 12, 6, 24, 26, 12, 14), "Redesign for uncontrolled fill deeper than 1.0 m is excluded. Six inspections are allowed."),
            ("northline", "NS-26041", 132_000, (6, 12, 6, 24, 26, 12, 14), "Bored pier design through the fill and eight hold point inspections are included."),
            ("hardpoint", "HE-26062", 155_000, (6, 12, 6, 22, 26, 12, 16), "A peer review and ten inspections are included."),
        ],
    },
    {
        "label": "Civil engineering", "title": "Fee proposal: civil and stormwater engineering services", "invoice_prefix": "CW-INV",
        "stages": [
            ("Schematic design", "Stormwater concept, detention sizing and the stormwater management report"),
            ("Tender documentation", "Civil drawings and specification for tender"),
            ("Design finalisation", "Construction certificate drawings and design certificate"),
            ("Delivery", "Inspections and review of the work-as-executed survey"),
        ],
        "exclusions": ["Survey and geotechnical investigation", "Authority fees", "Redesign for a contractor's alternative after award"],
        "bidders": [
            ("flowmark", "FM-26005", 78_000, (34, 40, 14, 12), "Detailed design of the detention tank and its certification are excluded."),
            ("catchment", "CW-26012", 96_000, (32, 38, 16, 14), "Detention tank design and certification are included."),
            ("basinworks", "BW-26030", 118_000, (30, 38, 16, 16), "Water quality modelling and monthly site visits are included."),
        ],
    },
    {
        "label": "Building services engineering", "title": "Fee proposal: building services engineering", "invoice_prefix": "FS-INV",
        "stages": [
            ("Brief and concept", "Utility enquiries and services brief"),
            ("Schematic design", "Schematic hydraulic, electrical and mechanical design"),
            ("Planning", "Services statements for the development application"),
            ("Design development", "Developed design and spatial coordination"),
            ("Tender documentation", "Services drawings and specification for tender"),
            ("Design finalisation", "Construction certificate drawings and design certificates"),
            ("Delivery", "Inspections, commissioning review and contractor queries"),
        ],
        "exclusions": ["Utility connection and augmentation charges", "Fire hydrant design, if the certifier later requires it", "Shop drawings"],
        "bidders": [
            ("circuit", "CA-26018", 132_000, (6, 14, 4, 24, 28, 12, 12), "Coordination with the utilities and the electric vehicle charging design are excluded."),
            ("flux", "FS-26028", 158_000, (6, 14, 4, 24, 28, 12, 12), "Hydraulic, electrical and mechanical services are all included, with six inspections."),
            ("gridworks", "GE-26052", 186_000, (6, 12, 4, 24, 28, 12, 14), "Detailed point schedules and a commissioning review are included."),
        ],
    },
]


def split(fee: int, shares: tuple[int, ...]) -> list[int]:
    assert sum(shares) == 100, shares
    parts = [round(fee * s / 100 / 500) * 500 for s in shares]
    parts[-1] += fee - sum(parts)
    assert sum(parts) == fee and min(parts) > 0
    return parts


def proposal_text(d: dict, bidder: tuple) -> str:
    firm, ref, fee, shares, note = bidder
    name, _, contact, _, _ = FIRMS[firm]
    person, role = contact.split(", ", 1)
    parts = split(fee, shares)
    rows = "\n".join(f"| {i}. {stage} | {service} | {money(amount)} |" for i, ((stage, service), amount) in enumerate(zip(d["stages"], parts), start=1))
    return f"""Title: {d["title"]}
Proposal no: {ref}
Date: 26 June 2025
Client: {CLIENT}

# {d["title"]}

{letterhead(firm)}

Project: {PROJECT}

## Introduction

Dear Michael,

Thank you for the invitation of 12 June 2025 and the client development brief. {name} is pleased to offer its {d["label"].lower()} for the 11 townhouses at {ADDRESS}, from the brief through to completion on site.

## Scope and fees

| Stage | Services | Fee (ex GST) |
|---|---|---|
{rows}
| **Total** | | **{money(fee)}** |

Total fee {money(fee)} excluding GST. All fees exclude GST.

{note}

## Exclusions

""" + "\n".join(f"- {e}" for e in d["exclusions"]) + f"""

## Additional services

Work outside this scope is charged at $240 per hour excluding GST for a director and $185 per hour excluding GST for an engineer or graduate, with your approval before we start.

## Terms

Each stage is invoiced on completion. Payment is due within 14 days. Professional indemnity insurance of $10 million and public liability insurance of $20 million are held. This offer is valid for 60 days.

Kind regards,

{person}
{role}, {name}
"""


def invoice_text(d: dict, bidder: tuple, n: int, date: str) -> tuple[str, str]:
    firm, ref, fee, shares, _ = bidder
    name, address, _, email, phone = FIRMS[firm]
    amount = split(fee, shares)[n - 1]
    number = f'{d["invoice_prefix"]}-{n:03d}'
    tax = amount // 10
    stage, service = d["stages"][n - 1]
    return number, f"""{name.upper()}
{address}
{email} | {phone}

# Tax Invoice

Invoice number: {number}
Invoice date: {date}
ABN: {ABNS[firm]}
Our reference: {ref}
Terms: 14 days

Bill to: {CLIENT}
Attention: {PM}

Project: {ADDRESS}

| Item | Description | Amount |
|---|---|---|
| 1 | {stage}: {service[0].lower() + service[1:]}. Stage {n} of {len(d["stages"])}, 100% complete | {cents(amount)} |
| | Subtotal ex GST | {cents(amount)} |
| | GST | {cents(tax)} |
| | **Total** | **{cents(amount + tax)}** |

Please pay by EFT to {name}
BSB 062-{914 + n} Account 2601 {7000 + len(name)}
"""


INVOICE_DATES = ["18/07/2025", "26/09/2025", "19/12/2025"]


def write_commercial() -> tuple[int, int]:
    proposals = invoices = 0
    for d in DISCIPLINES:
        for bidder in d["bidders"]:
            write("2-fee-proposals", f'{d["label"]} fee proposal - {FIRMS[bidder[0]][0].replace(" Pty Ltd", "")}.md', proposal_text(d, bidder))
            proposals += 1
        appointed = d["bidders"][1]
        # Every stage but the last has been invoiced, three at most.
        for n, date in enumerate(INVOICE_DATES[: len(d["stages"]) - 1], start=1):
            number, text = invoice_text(d, appointed, n, date)
            write("3-invoices", f"Tax invoice {number}.md", text)
            invoices += 1
    return proposals, invoices


README = """# Seven Hills: the landing cut

Written by `../generate-landing-cut.py`. Do not edit by hand; change the
script and run it again.

Everything here is invented: the address, land, firms, people, ABNs,
numbers and advice.

| Drop | Folder | Files | What the landing page shows |
|---|---|---|---|
| 1 | `1-project-folder/` | {folder} | The folder is dropped on an empty project and filed. Update fills the profile. |
| 2 | `2-fee-proposals/` | {proposals} | Fee proposals land on their consultant rows, three firms a row. One firm per row is awarded and the fees reach the cost plan. |
| 3 | `3-invoices/` | {invoices} | Invoices are dropped on Cost Plan, read and mapped. Claimed to date fills in. |

The appointed firm on each row is the middle-priced one: Axis Studio
($286,000), Civic Pattern Planning ($48,000), Northline Structures
($132,000), Catchment Works ($96,000) and Flux Services ($158,000), which
is $720,000 excluding GST in all. Each has invoiced its first stages (two for the planner, three for the others).
"""


def main() -> None:
    if ROOT.exists():
        shutil.rmtree(ROOT)
    folder = write_folder()
    proposals, invoices = write_commercial()
    (ROOT / "README.md").write_text(README.format(folder=folder, proposals=proposals, invoices=invoices), encoding="utf-8", newline="\n")
    print(f"landing-cut: {folder} folder documents, {proposals} fee proposals, {invoices} invoices")


if __name__ == "__main__":
    main()
