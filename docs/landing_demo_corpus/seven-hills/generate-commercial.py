"""Generate the Seven Hills synthetic commercial corpus.

This script owns only the four commercial output folders and two commercial
answer keys. It overwrites deterministic filenames without deleting unrelated
files.
"""

from __future__ import annotations

from datetime import date
from pathlib import Path


ROOT = Path(__file__).resolve().parent
PROJECT = "14–18 Wianamatta Avenue, Seven Hills NSW 2147"
CLIENT = "Wianamatta Developments Pty Ltd"
SCOPE = "Eleven two-storey attached townhouses and Torrens-title subdivision"
PROCUREMENT_SCOPE = (
    "twelve two-storey attached townhouses and Torrens-title subdivision"
)
PROCUREMENT_QUALIFICATION = (
    "This is the then-current planning scheme and remains subject to RFI "
    "responses and ongoing design development."
)
BUDGET = 9_800_000
GST_RATE = 10
OUTPUT_ROOTS = (
    "00-answer-keys",
    "02-consultant-procurement",
    "03-consultant-invoices",
    "06-builder-procurement",
    "07-construction-commercial",
)


def money(amount: int) -> str:
    return "$" + f"{amount:,.2f}"


def gst(amount: int) -> int:
    assert amount % GST_RATE == 0
    return amount // GST_RATE


def slug(value: str) -> str:
    result = "".join(char.lower() if char.isalnum() else "-" for char in value)
    while "--" in result:
        result = result.replace("--", "-")
    return result.strip("-")


def service_description(discipline_name: str) -> str:
    description = discipline_name.lower()
    if description.endswith(" services"):
        return description
    return f"{description} services"


def stages(fee: int) -> list[tuple[str, int, int]]:
    shares = (
        ("Due diligence and project definition", 10),
        ("Concept and authority coordination", 20),
        ("DA and tender documentation", 25),
        ("Construction documentation and IFC issue", 25),
        ("Tender clarification and construction handover", 20),
    )
    result = [(label, share, fee * share // 100) for label, share in shares]
    assert sum(item[2] for item in result) == fee
    return result


DISCIPLINES = [
    {
        "key": "architectural-services",
        "name": "Architectural Services",
        "rfp": "RFP-ARCH-01",
        "selected": "Axis Studio",
        "fee": 286_000,
        "job": "AXS-26017",
        "abn": "62 714 930 185",
        "address": "Studio 3, 18 Railway Parade, Blacktown NSW 2148",
        "bank": "BSB 062-914 · Account 2601 7004",
        "scope": [
            "Lead-consultant architectural service from concept through IFC",
            "Coordination of planning, structural, civil, services and landscape inputs",
            "Waste-room, collection-path and deep-soil coordination",
            "Tender addenda, clarification responses and monthly site attendance",
        ],
        "exclusions": [
            "Subconsultant fees and statutory charges",
            "Builder temporary works and proprietary shop drawings",
        ],
        "bidders": [
            ("Lantern Workshop Architects", "LW-26031", 248_000,
             "Six site inspections; excludes detailed waste-vehicle coordination."),
            ("Axis Studio", "AXS-26017", 286_000,
             "Full stated scope, including multidisciplinary coordination."),
            ("Commonline Architects", "CLA-26044", 332_000,
             "Adds interiors, sales plans and fortnightly site attendance."),
        ],
    },
    {
        "key": "town-planning",
        "name": "Town Planning",
        "rfp": "RFP-PLAN-01",
        "selected": "Civic Pattern Planning",
        "fee": 48_000,
        "job": "CPP-26019",
        "abn": "18 537 204 961",
        "address": "Suite 8, 44 Flushcombe Road, Blacktown NSW 2148",
        "bank": "BSB 082-401 · Account 2601 9018",
        "scope": [
            "Planning due diligence, pre-DA meeting and written strategy",
            "Statement of Environmental Effects and compliance schedule",
            "Torrens-subdivision and section 7.11 contribution coordination",
            "Council RFIs through determination and consent-condition review",
        ],
        "exclusions": [
            "Land and Environment Court proceedings",
            "Modification applications after determination",
        ],
        "bidders": [
            ("Parcel Planning", "PP-26008", 38_000,
             "One Council response; excludes pre-DA and subdivision advice."),
            ("Civic Pattern Planning", "CPP-26019", 48_000,
             "Full stated scope through determination."),
            ("Borough Urban Planning", "BUP-26027", 62_000,
             "Adds neighbour engagement and a panel briefing allowance."),
        ],
    },
    {
        "key": "structural-engineering",
        "name": "Structural Engineering",
        "rfp": "RFP-STR-01",
        "selected": "Northline Structures",
        "fee": 132_000,
        "job": "NS-26041",
        "abn": "29 658 407 133",
        "address": "Level 2, 12 Main Street, Blacktown NSW 2148",
        "bank": "BSB 032-180 · Account 2604 1029",
        "scope": [
            "Structural design for twelve attached dwellings and garages",
            "Bored-pier and ground-beam design responding to geotechnical evidence",
            "Fire-separated party-wall coordination",
            "Tender schedule of rates and eight hold-point inspections",
        ],
        "exclusions": [
            "Geotechnical investigation",
            "Builder temporary works and craneage design",
        ],
        "bidders": [
            ("Span Theory Engineers", "ST-26015", 112_000,
             "Excludes redesign for uncontrolled fill deeper than one metre."),
            ("Northline Structures", "NS-26041", 132_000,
             "Full stated scope, including bored-pier response."),
            ("Hardpoint Engineering", "HE-26062", 155_000,
             "Adds peer review, slab modelling and ten inspections."),
        ],
    },
    {
        "key": "civil-stormwater-engineering",
        "name": "Civil and Stormwater Engineering",
        "rfp": "RFP-CIV-01",
        "selected": "Catchment Works",
        "fee": 96_000,
        "job": "CW-26012",
        "abn": "68 315 742 906",
        "address": "Unit 5, 9 Foundry Road, Seven Hills NSW 2147",
        "bank": "BSB 012-620 · Account 2601 2068",
        "scope": [
            "Earthworks, grading, road, drainage and service-coordination design",
            "OSD investigation and design, with the required storage volume to be confirmed",
            "Drainage-easement and lawful-discharge coordination",
            "Tender clarifications, inspections and work-as-executed review",
        ],
        "exclusions": [
            "Survey, geotechnical and authority fees",
            "Contractor-alternative redesign after award",
        ],
        "bidders": [
            ("Flowmark Civil", "FM-26005", 78_000,
             "Excludes detailed OSD design and construction certification."),
            ("Catchment Works", "CW-26012", 96_000,
             "Full stated scope, including OSD investigation and design with storage volume to be confirmed."),
            ("Basinworks Engineering", "BW-26030", 118_000,
             "Adds WSUD modelling and monthly site attendance."),
        ],
    },
    {
        "key": "building-services-engineering",
        "name": "Building Services Engineering",
        "rfp": "RFP-BSE-01",
        "selected": "Flux Services",
        "fee": 158_000,
        "job": "FS-26028",
        "abn": "25 740 186 932",
        "address": "Level 1, 72 Powers Road, Seven Hills NSW 2147",
        "bank": "BSB 062-771 · Account 2602 8025",
        "scope": [
            "Integrated hydraulic, electrical and mechanical services design",
            "Dwelling metering, common services and communications",
            "Heat-pump hot water, ducted air conditioning and EV-ready infrastructure",
            "Tender clarifications and six coordinated inspections",
        ],
        "exclusions": [
            "Utility augmentation and connection charges",
            "Fire-hydrant design if later required by the certifier",
        ],
        "bidders": [
            ("Circuit and Air Consulting", "CA-26018", 132_000,
             "Excludes utility coordination, active EV provision and acoustic input."),
            ("Flux Services", "FS-26028", 158_000,
             "Full integrated scope across three services disciplines."),
            ("Gridworks Engineering", "GE-26052", 186_000,
             "Adds detailed point schedules and commissioning review."),
        ],
    },
]

INVOICE_DATES = (
    ("2025-07-18", "2025-08-01"),
    ("2025-09-26", "2025-10-10"),
    ("2025-12-19", "2026-01-02"),
    ("2026-02-27", "2026-03-13"),
    ("2026-05-15", "2026-05-29"),
)

TENDERS = [
    {
        "key": "redgum",
        "builder": "Redgum Constructions Pty Ltd",
        "abn": "53 801 264 719",
        "reference": "RG-26031",
        "date": "2026-04-17",
        "total": 9_080_000,
        "osd": "excluded",
        "lines": [
            ("Preliminaries and site establishment", 850_000),
            ("Demolition and remediation allowance", 450_000),
            ("Earthworks, subdivision and below-ground services", 1_350_000),
            ("Structure and framing", 1_650_000),
            ("Roofing and external envelope", 1_200_000),
            ("Windows and external doors", 550_000),
            ("Internal finishes and fitout", 1_150_000),
            ("Hydraulic, electrical and mechanical services", 1_200_000),
            ("External works and landscaping", 680_000),
        ],
    },
    {
        "key": "ironbark",
        "builder": "Ironbark Building Group Pty Ltd",
        "abn": "47 629 418 053",
        "reference": "IBG-T26-044",
        "date": "2026-04-17",
        "total": 9_340_000,
        "osd": "included",
        "lines": [
            ("Preliminaries and site establishment", 820_000),
            ("Demolition and remediation allowance", 420_000),
            ("Earthworks, subdivision and below-ground services", 1_500_000),
            ("Structure and framing", 1_620_000),
            ("Roofing and external envelope", 1_150_000),
            ("Windows and external doors", 500_000),
            ("Internal finishes and fitout", 1_100_000),
            ("Hydraulic, electrical and mechanical services", 1_150_000),
            ("External works and landscaping", 690_000),
            ("120 cubic metre OSD tank", 390_000),
        ],
    },
    {
        "key": "calderline",
        "builder": "Calderline Projects Pty Ltd",
        "abn": "31 956 740 128",
        "reference": "CP-26-118",
        "date": "2026-04-17",
        "total": 9_460_000,
        "osd": "included",
        "lines": [
            ("Preliminaries and site establishment", 870_000),
            ("Demolition and remediation allowance", 430_000),
            ("Earthworks, subdivision and below-ground services", 1_550_000),
            ("Structure and framing", 1_680_000),
            ("Roofing and external envelope", 1_180_000),
            ("Windows and external doors", 520_000),
            ("Internal finishes and fitout", 1_130_000),
            ("Hydraulic, electrical and mechanical services", 1_170_000),
            ("External works and landscaping", 570_000),
            ("120 cubic metre OSD tank", 360_000),
        ],
    },
]

PROGRESS_CLAIMS = [
    {
        "number": "PC-01",
        "invoice": "IBG-PC-01",
        "date": "2026-05-20",
        "period": "5 May to 20 May 2026",
        "total": 560_000,
        "lines": [
            ("Preliminaries and establishment", 320_000),
            ("Mobilisation and temporary facilities", 140_000),
            ("Design review and subcontract coordination", 100_000),
        ],
    },
    {
        "number": "PC-02",
        "invoice": "IBG-PC-02",
        "date": "2026-06-20",
        "period": "21 May to 20 June 2026",
        "total": 680_000,
        "lines": [
            ("Demolition", 220_000),
            ("Bulk earthworks", 260_000),
            ("Temporary drainage", 80_000),
            ("Preliminaries", 120_000),
        ],
    },
    {
        "number": "PC-03",
        "invoice": "IBG-PC-03",
        "date": "2026-07-20",
        "period": "21 June to 20 July 2026",
        "total": 1_120_000,
        "lines": [
            ("Bored piers", 410_000),
            ("Ground beams", 310_000),
            ("Ground-floor slabs", 300_000),
            ("Preliminaries", 100_000),
        ],
    },
    {
        "number": "PC-04",
        "invoice": "IBG-PC-04",
        "date": "2026-08-18",
        "period": "21 July to 18 August 2026",
        "total": 1_048_500,
        "lines": [
            ("Timber framing", 440_000),
            ("Structural steel", 210_000),
            ("Roof trusses", 160_000),
            ("Preliminaries", 170_000),
            ("VO-007 S-202 Rev C OSD structural reinforcement change — unapproved", 68_500),
        ],
    },
]


def synthetic_header(kind: str) -> str:
    return (
        f"> **SYNTHETIC DEMO DOCUMENT — {kind.upper()}**  \n"
        "> Fictional organisations, people, contact details and project. "
        "Prepared solely for the SiteWise product demonstration.\n"
    )


def metadata_table(rows: list[tuple[str, str]]) -> str:
    body = "\n".join(f"| {label} | {value} |" for label, value in rows)
    return f"| Field | Detail |\n|---|---|\n{body}"


def proposal_text(discipline: dict, bidder: tuple[str, str, int, str]) -> str:
    firm, reference, fee, qualification = bidder
    stage_rows = stages(fee)
    fee_table = "\n".join(
        f"| {name} | {percent}% | {money(amount)} |"
        for name, percent, amount in stage_rows
    )
    scope = "\n".join(f"- {item}" for item in discipline["scope"])
    exclusions = "\n".join(f"- {item}" for item in discipline["exclusions"])
    return f"""# Fee Proposal — {discipline["name"]}

{synthetic_header("Consultant fee proposal")}

{metadata_table([
    ("Project", PROJECT),
    ("Client", CLIENT),
    ("RFP", discipline["rfp"]),
    ("Proponent", firm),
    ("Proposal reference", reference),
    ("Proposal date", "26 June 2025"),
    ("Validity", "60 days"),
])}

## Offer

{firm} offers to provide the stated {service_description(discipline["name"])} for a lump-sum professional fee of **{money(fee)} excluding GST**. This proposal is an offer subject to execution of a written consultancy agreement.

## Project understanding

The commission concerns {PROCUREMENT_SCOPE} at {PROJECT}. {PROCUREMENT_QUALIFICATION} Design is to support development approval, coordinated construction documentation, whole-of-works tendering and construction-phase services.

## Included services

{scope}

## Fee schedule

| Stage | Share | Fee excluding GST |
|---|---:|---:|
{fee_table}
| **Total** | **100%** | **{money(fee)}** |

## Proposal-specific qualification

- {qualification}

## General exclusions

{exclusions}

## Commercial assumptions

- GST is additional to the stated fee.
- Client and authority fees, specialist investigations and contractor costs are excluded unless expressly listed.
- Additional services require written scope and fee agreement before commencement.
- Payment terms: 14 days from a valid tax invoice.

---

Synthetic demo document. Not a real offer or project record.
"""


def appointment_text(discipline: dict, number: int) -> str:
    stage_rows = stages(discipline["fee"])
    schedule = "\n".join(
        f"| {name} | {percent}% | {money(amount)} |"
        for name, percent, amount in stage_rows
    )
    scope = "\n".join(f"- {item}" for item in discipline["scope"])
    return f"""# Letter of Appointment — {discipline["name"]}

{synthetic_header("Consultant appointment")}

{metadata_table([
    ("Project", PROJECT),
    ("Client", CLIENT),
    ("Consultant", discipline["selected"]),
    ("Consultant job reference", discipline["job"]),
    ("Appointment reference", f"WD-APPT-{number:02d}"),
    ("Date", "7 July 2025"),
])}

Dear Sir/Madam,

The Client appoints **{discipline["selected"]}** to provide {service_description(discipline["name"])} for {PROJECT}.

The appointment is based on the **{PROCUREMENT_SCOPE}**. {PROCUREMENT_QUALIFICATION}

## Accepted scope

{scope}

## Accepted fee

The lump-sum fee is **{money(discipline["fee"])} excluding GST**, payable against five stage invoices:

| Stage | Share | Fee excluding GST |
|---|---:|---:|
{schedule}
| **Total** | **100%** | **{money(discipline["fee"])}** |

## Administration and boundaries

- Ridgeline Project Management Pty Ltd is the Client's project manager and correspondence administrator.
- The intended building contract is **AS 4000–1997, construct-only**, with Ridgeline Project Management Pty Ltd acting as the Client's Superintendent.
- The consultant remains engaged directly by the Client. **No novation to the builder applies.**
- Changes to scope or fee require prior written Client approval.
- Invoices must quote {discipline["job"]}, state GST separately and be addressed to the Client.

Please countersign the execution copy to confirm acceptance.

Yours faithfully,  
Development Director  
Wianamatta Developments Pty Ltd

---

Synthetic demo document. Not a real appointment or contract.
"""


def consultant_invoice_text(
    discipline: dict,
    invoice_index: int,
    stage_row: tuple[str, int, int],
) -> str:
    stage_name, percent, amount = stage_row
    issue_date, due_date = INVOICE_DATES[invoice_index - 1]
    prior = sum(row[2] for row in stages(discipline["fee"])[: invoice_index - 1])
    cumulative = prior + amount
    invoice_number = f'{discipline["job"]}-INV-{invoice_index:02d}'
    return f"""# Tax Invoice — {invoice_number}

{synthetic_header("Consultant tax invoice")}

**{discipline["selected"]}**  
ABN {discipline["abn"]}  
{discipline["address"]}

{metadata_table([
    ("Invoice number", invoice_number),
    ("Issue date", issue_date),
    ("Due date", due_date),
    ("Bill to", CLIENT),
    ("Deliver to", "Ridgeline Project Management Pty Ltd"),
    ("Project", PROJECT),
    ("Consultant job", discipline["job"]),
    ("Appointment", f'{discipline["name"]} — {money(discipline["fee"])} excluding GST'),
])}

## Current instalment

| Description | Stage share | Amount excluding GST |
|---|---:|---:|
| {stage_name} professional services | {percent}% | {money(amount)} |
| **Subtotal** |  | **{money(amount)}** |
| GST |  | **{money(gst(amount))}** |
| **Amount due including GST** |  | **{money(amount + gst(amount))}** |

## Fee drawdown

| Reconciliation | Excluding GST |
|---|---:|
| Prior invoices | {money(prior)} |
| This invoice | {money(amount)} |
| Cumulative invoiced | {money(cumulative)} |
| Remaining appointment fee | {money(discipline["fee"] - cumulative)} |
| **Appointment fee** | **{money(discipline["fee"])}** |

Payment terms: 14 days. Remittance: {discipline["bank"]}. This is instalment {invoice_index} of five.

---

Synthetic demo document. Not a real tax invoice or demand for payment.
"""


def builder_tender_text(tender: dict) -> str:
    price_rows = "\n".join(
        f"| {description} | {money(amount)} |"
        for description, amount in tender["lines"]
    )
    if tender["key"] == "redgum":
        osd_statement = """- **Excluded:** design, supply, excavation, installation, connection and commissioning of the specified **120 cubic metre OSD tank**.
- The submitted tender total remains $9,080,000 excluding GST before any accepted clarification or addendum."""
    else:
        osd_statement = """- **Included:** complete 120 cubic metre OSD tank, including excavation, structure, fittings, connection, testing and commissioning.
- The inclusion is contained within the submitted tender total."""
    return f"""# Whole-Builder Tender — {tender["builder"]}

{synthetic_header("Builder tender")}

{metadata_table([
    ("Project", PROJECT),
    ("Client", CLIENT),
    ("Tenderer", tender["builder"]),
    ("ABN", tender["abn"]),
    ("Tender reference", tender["reference"]),
    ("Tender date", tender["date"]),
    ("Contract basis", "AS 4000–1997 construct-only"),
    ("Tender validity", "90 days"),
])}

## Tender offer

{tender["builder"]} offers to execute and complete the whole of the building works described by the issued tender documents for **{money(tender["total"])} excluding GST**, subject to the qualifications expressly stated in this tender.

## Price schedule

| Trade or work package | Amount excluding GST |
|---|---:|
{price_rows}
| **Submitted tender total** | **{money(tender["total"])}** |

## OSD tank scope

{osd_statement}

## Contract and delivery assumptions

- AS 4000–1997 construct-only, without contractor design novation.
- Ridgeline Project Management Pty Ltd is identified as the Client's project manager and intended Superintendent.
- Consultant appointments remain directly with the Client; no consultant novation is included.
- Tender pricing is exclusive of GST and includes builder preliminaries, margin and ordinary coordination for the stated scope.
- Authority charges, latent conditions and Client-directed changes are excluded unless expressly priced.

## Programme

- Mobilisation: within 15 business days after contract execution and site possession.
- Construction duration: 58 calendar weeks from site possession, excluding approved extensions of time.

---

Synthetic demo document. Not a real tender or contractual offer.
"""


def redgum_addendum_text() -> str:
    return f"""# Tender Clarification / Addendum 01 — 120 m³ OSD Tank

{synthetic_header("Builder tender clarification")}

{metadata_table([
    ("Project", PROJECT),
    ("Tenderer", "Redgum Constructions Pty Ltd"),
    ("Tender reference", "RG-26031"),
    ("Clarification reference", "RG-26031-ADD-01"),
    ("Date", "24 April 2026"),
    ("Contract basis", "AS 4000–1997 construct-only"),
])}

In response to clarification TCQ-04, Redgum Constructions Pty Ltd confirms that its submitted tender of **$9,080,000 excluding GST** omitted the complete 120 cubic metre OSD tank.

## Priced scope addition

| Addition | Amount excluding GST |
|---|---:|
| Design coordination, excavation, supply, construction, connection, testing and commissioning of the specified 120 cubic metre OSD tank | $420,000 |
| **Clarified addition** | **$420,000** |

If this addition is incorporated, the arithmetically reconciled tender amount is:

| Reconciliation | Amount excluding GST |
|---|---:|
| Submitted Redgum tender | $9,080,000 |
| Clarified OSD tank addition | $420,000 |
| **Comparable tender amount** | **$9,500,000** |

All other qualifications in RG-26031 remain unchanged. This document is a tender clarification only and does not amend a contract.

---

Synthetic demo document. Not a real tender addendum or contractual variation.
"""


def builder_acceptance_text() -> str:
    return f"""# Letter of Acceptance — Ironbark Building Group Pty Ltd

{synthetic_header("Builder appointment")}

{metadata_table([
    ("Project", PROJECT),
    ("Client", CLIENT),
    ("Builder", "Ironbark Building Group Pty Ltd"),
    ("Tender reference", "IBG-T26-044"),
    ("Acceptance reference", "WD-LOA-001"),
    ("Date", "4 May 2026"),
    ("Construction budget", "$9,800,000 excluding GST"),
])}

Dear Sir/Madam,

The Client accepts Ironbark Building Group Pty Ltd's whole-of-works tender dated 17 April 2026 for a contract sum of **$9,340,000 excluding GST**, subject to execution of the formal contract documents.

## Accepted commercial basis

- **Contract form:** AS 4000–1997, construct-only.
- **Contract sum:** $9,340,000 excluding GST.
- **OSD scope:** the complete 120 cubic metre OSD tank is included.
- **Superintendent:** Ridgeline Project Management Pty Ltd.
- **Client project manager:** Ridgeline Project Management Pty Ltd.
- **Consultants:** retained by the Client; no novation to the Builder.
- **Time for completion:** 58 calendar weeks from site possession, subject to the contract.

The Builder is authorised to prepare contract execution documents, insurance evidence, security and the baseline construction programme. No work causing expenditure beyond the accepted contract sum is authorised without the Client's written approval under the contract.

Yours faithfully,  
Development Director  
Wianamatta Developments Pty Ltd

---

Synthetic demo document. Not a real letter of acceptance or building contract.
"""


def progress_claim_text(claim: dict, cumulative: int) -> str:
    rows = "\n".join(
        f"| {description} | {money(amount)} |"
        for description, amount in claim["lines"]
    )
    vo_notice = ""
    if claim["number"] == "PC-04":
        vo_notice = """
## Variation-status notice

**VO-007 — S-202 Rev C OSD structural reinforcement change, $68,500 excluding GST — is unapproved.** It is included in the Builder's claimed amount below while awaiting the Superintendent's determination. Its appearance in this invoice and progress claim is not Client approval and does not change the contract sum.
"""
    amount = claim["total"]
    return f"""# Tax Invoice and Progress Claim — {claim["number"]}

{synthetic_header("Builder tax invoice and progress claim")}

**Ironbark Building Group Pty Ltd**  
ABN 47 629 418 053  
18 Artisan Circuit, Seven Hills NSW 2147

{metadata_table([
    ("Invoice number", claim["invoice"]),
    ("Progress claim", claim["number"]),
    ("Issue date", claim["date"]),
    ("Claim period", claim["period"]),
    ("Bill to", CLIENT),
    ("Project", PROJECT),
    ("Contract", "AS 4000–1997 construct-only"),
    ("Contract sum", "$9,340,000 excluding GST"),
    ("Superintendent", "Ridgeline Project Management Pty Ltd"),
])}
{vo_notice}
## Amount claimed this period

| Work item | Amount excluding GST |
|---|---:|
{rows}
| **Progress claim excluding GST** | **{money(amount)}** |
| GST | **{money(gst(amount))}** |
| **Tax invoice total including GST** | **{money(amount + gst(amount))}** |

## Claim reconciliation

| Reconciliation | Amount excluding GST |
|---|---:|
| Prior gross claims | {money(cumulative - amount)} |
| This gross claim | {money(amount)} |
| **Cumulative gross claims** | **{money(cumulative)}** |
| Original contract sum | {money(9_340_000)} |

Submitted to the Superintendent for assessment under the contract. Payment details: BSB 082-114 · Account 2606 4401.

---

Synthetic demo document. Not a real tax invoice, payment claim or demand for payment.
"""


def commercial_register_text() -> str:
    consultant_rows = []
    for discipline in DISCIPLINES:
        offers = "<br>".join(
            f"{firm}: {money(fee)}"
            for firm, _reference, fee, _qualification in discipline["bidders"]
        )
        invoice_total = sum(row[2] for row in stages(discipline["fee"]))
        consultant_rows.append(
            f'| {discipline["name"]} | {offers} | '
            f'**{discipline["selected"]} — {money(discipline["fee"])}** | '
            f'{discipline["job"]}-INV-01 to -05 | **{money(invoice_total)}** |'
        )
    consultants = "\n".join(consultant_rows)

    tender_rows = []
    for tender in TENDERS:
        addition = "$420,000" if tender["key"] == "redgum" else "$0"
        comparable = tender["total"] + (420_000 if tender["key"] == "redgum" else 0)
        tender_rows.append(
            f'| {tender["builder"]} | {tender["reference"]} | '
            f'{money(tender["total"])} | {tender["osd"].title()} | '
            f'{addition} | **{money(comparable)}** |'
        )
    tenders = "\n".join(tender_rows)

    claim_rows = []
    cumulative = 0
    for claim in PROGRESS_CLAIMS:
        cumulative += claim["total"]
        note = (
            "Includes unapproved VO-007 — S-202 Rev C OSD structural reinforcement change — at $68,500"
            if claim["number"] == "PC-04"
            else "No variation line"
        )
        claim_rows.append(
            f'| {claim["number"]} | {claim["invoice"]} | {claim["date"]} | '
            f'{money(claim["total"])} | {money(gst(claim["total"]))} | '
            f'{money(claim["total"] + gst(claim["total"]))} | '
            f'{money(cumulative)} | {note} |'
        )
    claims = "\n".join(claim_rows)

    return f"""# Commercial Register — Answer Key

{synthetic_header("Commercial reconciliation answer key")}

This file states the expected commercial facts for deterministic demo testing. Source documents remain the evidence; this key is not intended for project ingestion.

## Governing project facts

{metadata_table([
    ("Project", PROJECT),
    ("Client", CLIENT),
    ("Development", SCOPE),
    ("Construction budget", "$9,800,000 excluding GST"),
    ("Building contract", "AS 4000–1997 construct-only"),
    ("Project manager and Superintendent", "Ridgeline Project Management Pty Ltd"),
    ("Consultant novation", "None; all five consultants remain Client-appointed"),
])}

## Consultant procurement and invoice reconciliation

| Discipline | Three submitted fees, excluding GST | Appointed consultant and fee | Five invoice references | Invoices 01–05 total |
|---|---|---|---|---:|
{consultants}
| **Portfolio total** | **15 fee proposals** | **5 appointments — $720,000** | **25 separate invoices** | **$720,000** |

Required count and arithmetic checks:

- 3 proposals × 5 disciplines = **15 proposals**.
- One separate appointment letter per discipline = **5 appointments**.
- 5 invoices × 5 appointed consultants = **25 invoices**.
- $286,000 + $48,000 + $132,000 + $96,000 + $158,000 = **$720,000 excluding GST**.
- Every consultant's five invoice subtotals equal its accepted appointment fee exactly.

## Builder tender register

| Tenderer | Reference | Submitted total excl. GST | 120 m³ OSD | Explicit clarification | Comparable total excl. GST |
|---|---|---:|---|---:|---:|
{tenders}

Appointment evidence: letter WD-LOA-001 records **Ironbark Building Group Pty Ltd**, **$9,340,000 excluding GST**, AS 4000–1997 construct-only. The accepted contract sum is $460,000 below the stated $9,800,000 construction budget.

## Builder invoice and progress-claim register

| Claim | Invoice | Date | Excl. GST | GST | Incl. GST | Cumulative excl. GST | Control note |
|---|---|---|---:|---:|---:|---:|---|
{claims}

The four gross claims total **$3,408,500 excluding GST**. PC-04's printed total of **$1,048,500 excluding GST** includes **VO-007 — S-202 Rev C OSD structural reinforcement change — at $68,500**, which is expressly marked unapproved; invoice inclusion does not establish Client or Superintendent approval.

---

Synthetic demo answer key. Not a real project or financial record.
"""


def tender_comparison_answer_key_text() -> str:
    return f"""# Tender Comparison — Answer Key

{synthetic_header("Tender comparison answer key")}

This key defines the expected extraction and deterministic arithmetic for the three whole-builder tenders. Comparability changes must be tied to explicit tender evidence.

**Project:** {PROJECT}  
**Client:** {CLIENT}

## Input set

| Bidder | Tender evidence | Submitted total excl. GST | 120 m³ OSD evidence |
|---|---|---:|---|
| Redgum Constructions Pty Ltd | RG-26031 | $9,080,000 | Explicitly excluded in base tender |
| Ironbark Building Group Pty Ltd | IBG-T26-044 | $9,340,000 | Explicitly included |
| Calderline Projects Pty Ltd | CP-26-118 | $9,460,000 | Explicitly included |

Each document is a whole-of-works builder tender under an AS 4000–1997 construct-only basis. No tender includes consultant novation.

## Explicit scope reconciliation

Only one comparability change is permitted by the evidence: Redgum clarification RG-26031-ADD-01 prices the omitted 120 cubic metre OSD tank at **$420,000 excluding GST**.

| Bidder | Submitted total | Evidence-backed OSD addition | Comparable total | Difference from Ironbark |
|---|---:|---:|---:|---:|
| Redgum Constructions Pty Ltd | $9,080,000 | +$420,000 | **$9,500,000** | +$160,000 |
| Ironbark Building Group Pty Ltd | $9,340,000 | $0 | **$9,340,000** | $0 |
| Calderline Projects Pty Ltd | $9,460,000 | $0 | **$9,460,000** | +$120,000 |

Arithmetic checks:

- Redgum: $9,080,000 + $420,000 = **$9,500,000**.
- Ironbark: $9,340,000 + $0 = **$9,340,000**.
- Calderline: $9,460,000 + $0 = **$9,460,000**.
- The three submitted totals are $9,080,000, $9,340,000 and $9,460,000.
- The three comparable totals are $9,500,000, $9,340,000 and $9,460,000.

## Outcome evidence boundary

Tender documents and arithmetic do not establish an appointment. The separate letter WD-LOA-001 is the source that records Ironbark Building Group Pty Ltd as appointed at **$9,340,000 excluding GST**, including the OSD tank.

No contingency, escalation, location factor, market allowance, risk premium or other inferred adjustment is applied.

---

Synthetic demo answer key. Not a real tender assessment or project record.
"""


def validate_source_model() -> None:
    expected_keys = {
        "architectural-services",
        "town-planning",
        "structural-engineering",
        "civil-stormwater-engineering",
        "building-services-engineering",
    }
    assert len(DISCIPLINES) == 5
    assert {item["key"] for item in DISCIPLINES} == expected_keys
    assert sum(item["fee"] for item in DISCIPLINES) == 720_000
    assert INVOICE_DATES[-1] == ("2026-05-15", "2026-05-29")
    assert all(tender["date"] == "2026-04-17" for tender in TENDERS)
    assert "2026-04-17" < "2026-05-04" < INVOICE_DATES[-1][0]

    for discipline in DISCIPLINES:
        assert len(discipline["bidders"]) == 3
        selected_matches = [
            bidder
            for bidder in discipline["bidders"]
            if bidder[0] == discipline["selected"]
        ]
        assert len(selected_matches) == 1
        assert selected_matches[0][2] == discipline["fee"]
        assert len(stages(discipline["fee"])) == 5
        assert sum(row[2] for row in stages(discipline["fee"])) == discipline["fee"]

    assert len(TENDERS) == 3
    for tender in TENDERS:
        assert sum(amount for _description, amount in tender["lines"]) == tender["total"]
    tender_by_key = {tender["key"]: tender for tender in TENDERS}
    assert tender_by_key["redgum"]["total"] == 9_080_000
    assert tender_by_key["redgum"]["osd"] == "excluded"
    assert tender_by_key["redgum"]["total"] + 420_000 == 9_500_000
    assert tender_by_key["ironbark"]["total"] == 9_340_000
    assert tender_by_key["ironbark"]["osd"] == "included"
    assert tender_by_key["calderline"]["total"] == 9_460_000
    assert tender_by_key["calderline"]["osd"] == "included"

    assert len(PROGRESS_CLAIMS) == 4
    assert PROGRESS_CLAIMS[0]["period"] == "5 May to 20 May 2026"
    assert "| Date | 4 May 2026 |" in builder_acceptance_text()
    assert [claim["total"] for claim in PROGRESS_CLAIMS] == [
        560_000,
        680_000,
        1_120_000,
        1_048_500,
    ]
    for claim in PROGRESS_CLAIMS:
        assert sum(amount for _description, amount in claim["lines"]) == claim["total"]
    pc04 = PROGRESS_CLAIMS[-1]
    assert pc04["number"] == "PC-04"
    assert (
        "VO-007 S-202 Rev C OSD structural reinforcement change — unapproved",
        68_500,
    ) in pc04["lines"]
    pc04_document = progress_claim_text(pc04, 3_408_500)
    assert pc04_document.count("VO-007") == 2
    assert pc04_document.count("S-202 Rev C OSD structural reinforcement change") == 2
    assert commercial_register_text().count(
        "S-202 Rev C OSD structural reinforcement change"
    ) == 2
    assert sum(claim["total"] for claim in PROGRESS_CLAIMS) == 3_408_500


def write_document(relative_path: Path, content: str) -> Path:
    if relative_path.parts[0] not in OUTPUT_ROOTS:
        raise ValueError(f"Commercial generator cannot write to {relative_path}")
    path = ROOT / relative_path
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    return path


def main() -> None:
    validate_source_model()
    generated: list[Path] = []

    for number, discipline in enumerate(DISCIPLINES, start=1):
        for bidder in discipline["bidders"]:
            generated.append(
                write_document(
                    Path(
                        "02-consultant-procurement",
                        discipline["key"],
                        "proposals",
                        f"{slug(bidder[0])}-fee-proposal.md",
                    ),
                    proposal_text(discipline, bidder),
                )
            )
        generated.append(
            write_document(
                Path(
                    "02-consultant-procurement",
                    discipline["key"],
                    "appointment",
                    f'{slug(discipline["selected"])}-appointment.md',
                ),
                appointment_text(discipline, number),
            )
        )
        for invoice_index, stage_row in enumerate(stages(discipline["fee"]), start=1):
            generated.append(
                write_document(
                    Path(
                        "03-consultant-invoices",
                        discipline["key"],
                        f'{discipline["job"]}-INV-{invoice_index:02d}.md',
                    ),
                    consultant_invoice_text(discipline, invoice_index, stage_row),
                )
            )

    for tender in TENDERS:
        generated.append(
            write_document(
                Path(
                    "06-builder-procurement",
                    "tenders",
                    f'{tender["key"]}-whole-builder-tender.md',
                ),
                builder_tender_text(tender),
            )
        )
    generated.append(
        write_document(
            Path(
                "06-builder-procurement",
                "clarifications",
                "redgum-clarification-addendum-01.md",
            ),
            redgum_addendum_text(),
        )
    )
    generated.append(
        write_document(
            Path(
                "06-builder-procurement",
                "award",
                "ironbark-letter-of-acceptance.md",
            ),
            builder_acceptance_text(),
        )
    )

    cumulative = 0
    for claim in PROGRESS_CLAIMS:
        cumulative += claim["total"]
        generated.append(
            write_document(
                Path(
                    "07-construction-commercial",
                    "progress-claims",
                    f'{claim["number"]}-{claim["invoice"]}.md',
                ),
                progress_claim_text(claim, cumulative),
            )
        )

    generated.append(
        write_document(
            Path("00-answer-keys", "commercial-register.md"),
            commercial_register_text(),
        )
    )
    generated.append(
        write_document(
            Path("00-answer-keys", "tender-comparison-answer-key.md"),
            tender_comparison_answer_key_text(),
        )
    )

    assert len(generated) == 56
    assert len(set(generated)) == 56
    for path in generated:
        content = path.read_text(encoding="utf-8")
        assert "SYNTHETIC DEMO" in content
        assert "services services" not in content.lower()
        assert "torrens-title" not in content

    proposal_root = ROOT / "02-consultant-procurement"
    proposals = list(proposal_root.glob("*/proposals/*.md"))
    appointments = list(proposal_root.glob("*/appointment/*.md"))
    invoices = list((ROOT / "03-consultant-invoices").glob("*/*.md"))
    tenders = list((ROOT / "06-builder-procurement" / "tenders").glob("*.md"))
    claims = list(
        (ROOT / "07-construction-commercial" / "progress-claims").glob("*.md")
    )
    assert len(proposals) == 15
    assert len(appointments) == 5
    assert len(invoices) == 25
    assert len(tenders) == 3
    assert len(claims) == 4
    assert all(
        (date.fromisoformat(due) - date.fromisoformat(issue)).days == 14
        for issue, due in INVOICE_DATES
    )
    assert all(
        "The Client appoints" not in path.read_text(encoding="utf-8")
        for path in proposals
    )

    for historical_document in proposals + appointments:
        content = historical_document.read_text(encoding="utf-8")
        normalised = content.lower()
        assert "eleven" not in normalised
        assert "120 cubic" not in normalised
        assert "120 m³" not in normalised
        assert "twelve" in normalised
        assert "rfi" in normalised
        assert "design development" in normalised
        assert "Torrens-title subdivision" in content

    commercial_key = (ROOT / "00-answer-keys" / "commercial-register.md").read_text(
        encoding="utf-8"
    )
    assert SCOPE in commercial_key
    assert all(
        "120 cubic metre OSD" in path.read_text(encoding="utf-8")
        for path in tenders
    )

    print("Generated and validated 56 synthetic commercial documents.")
    print("15 consultant proposals; 5 appointments; 25 consultant invoices.")
    print("3 builder tenders; 1 clarification; 1 acceptance; 4 progress claims.")
    print("Consultant fee reconciliation: $720,000 excluding GST.")
    print("Builder claim reconciliation: $3,408,500 excluding GST.")


if __name__ == "__main__":
    main()
