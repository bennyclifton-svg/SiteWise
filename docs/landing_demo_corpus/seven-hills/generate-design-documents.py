"""Generate the synthetic Seven Hills design-document demo corpus.

The drawing files are deliberately lean title-block records. They carry enough
structured evidence to build a credible drawing register without pretending to
be construction drawings. Three reports carry a few explicit findings so the
OSD evidence chain can be demonstrated across disciplines.

Re-runnable: only the managed outputs below ``04-design-documents`` are replaced.

    python docs/demo-corpus/seven-hills/generate-design-documents.py
"""

from __future__ import annotations

import shutil
from pathlib import Path
from typing import TypedDict


SOURCE_ROOT = Path(__file__).resolve().parent
OUTPUT_ROOT = SOURCE_ROOT / "04-design-documents"

PROJECT = "14–18 Wianamatta Avenue, Seven Hills NSW 2147"
CLIENT = "Wianamatta Developments Pty Ltd"
SCOPE = "11 attached two-storey Class 1a townhouses"
CONTRACT = "AS 4000 construct-only"
REGISTER_DATE = "2026-08-21"


class Firm(TypedDict):
    name: str
    job: str
    address: str
    drawn: str
    checked: str


class Drawing(TypedDict, total=False):
    number: str
    title: str
    revision: str
    scale: str
    date: str
    purpose: str
    notes: list[str]


class Report(TypedDict, total=False):
    number: str
    title: str
    revision: str
    date: str
    purpose: str
    author: str
    job: str
    prepared_by: str
    pages: int
    scope: str
    contents: list[str]
    findings: list[str]


FIRMS: dict[str, Firm] = {
    "architectural": {
        "name": "Axis Studio",
        "job": "AXS-26017",
        "address": "Level 2, 17 Foundry Road, Parramatta NSW 2150",
        "drawn": "M. Chen",
        "checked": "R. Iqbal",
    },
    "structural": {
        "name": "Northline Structures",
        "job": "NS-26041",
        "address": "Suite 5, 82 George Street, Parramatta NSW 2150",
        "drawn": "L. Duarte",
        "checked": "P. Nair",
    },
    "civil": {
        "name": "Catchment Works",
        "job": "CW-26012",
        "address": "Unit 8, 14 Tucks Road, Seven Hills NSW 2147",
        "drawn": "E. Haddad",
        "checked": "S. Moyo",
    },
    "hydraulic": {
        "name": "Flux Services",
        "job": "FS-26028",
        "address": "Level 1, 33 Phillip Street, Parramatta NSW 2150",
        "drawn": "J. Park",
        "checked": "N. Foster",
    },
    "electrical": {
        "name": "Flux Services",
        "job": "FS-26028",
        "address": "Level 1, 33 Phillip Street, Parramatta NSW 2150",
        "drawn": "A. Singh",
        "checked": "N. Foster",
    },
    "mechanical": {
        "name": "Flux Services",
        "job": "FS-26028",
        "address": "Level 1, 33 Phillip Street, Parramatta NSW 2150",
        "drawn": "K. Osei",
        "checked": "N. Foster",
    },
    "landscape": {
        "name": "Fieldwork Landscape",
        "job": "FL-26009",
        "address": "Studio 3, 46 Grose Street, North Parramatta NSW 2151",
        "drawn": "T. Bell",
        "checked": "A. Farah",
    },
}

DISCIPLINE_LABELS = {
    "architectural": "Architectural",
    "structural": "Structural",
    "civil": "Civil / Stormwater",
    "hydraulic": "Hydraulic Services",
    "electrical": "Electrical Services",
    "mechanical": "Mechanical Services",
    "landscape": "Landscape Architecture",
}


def sheet(
    number: str,
    title: str,
    revision: str,
    scale: str,
    date: str,
    purpose: str = "For Construction",
    notes: list[str] | None = None,
) -> Drawing:
    result: Drawing = {
        "number": number,
        "title": title,
        "revision": revision,
        "scale": scale,
        "date": date,
        "purpose": purpose,
    }
    if notes:
        result["notes"] = notes
    return result


DRAWINGS: dict[str, list[Drawing]] = {
    "architectural": [
        sheet("A-000", "Cover Sheet, Locality Plan and Drawing Register", "D", "NTS", "2026-01-30"),
        sheet("A-001", "Site Analysis and Existing Site Plan", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-002", "Demolition and Tree Retention Plan", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-100", "Proposed Site Plan", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-101", "Ground Floor Plans — Townhouses 1–6", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-102", "First Floor Plans — Townhouses 1–6", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-103", "Ground Floor Plans — Townhouses 7–11", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-104", "First Floor Plans — Townhouses 7–11", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-105", "Roof Plan", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-110", "Subdivision and Lot Layout Plan", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-200", "Elevations — North and South", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-201", "Elevations — East and West", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-202", "Wianamatta Avenue Streetscape Elevations", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-300", "Sections A–A and B–B", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-301", "Sections C–C and D–D", "D", "1:100 @ A1", "2026-01-30"),
        sheet("A-400", "External Finishes and Materials Schedule", "D", "NTS", "2026-01-30"),
        sheet("A-401", "Door and Window Schedule", "D", "1:50 @ A1", "2026-01-30"),
        sheet("A-500", "Shadow Diagrams — 21 June, 9 am, Noon and 3 pm", "D", "1:250 @ A1", "2026-01-30"),
        sheet("A-600", "Waste Storage and Collection Plan", "D", "1:200 @ A1", "2026-01-30"),
        sheet("A-900", "Construction Details — Party Walls and Wet Areas", "D", "1:10 @ A1", "2026-01-30"),
    ],
    "structural": [
        sheet("S-001", "General Notes, Legend and Design Criteria", "B", "NTS", "2026-03-27"),
        sheet("S-100", "Footing, Pier and Retaining Wall Set-out", "B", "1:100 @ A1", "2026-03-27"),
        sheet("S-101", "Ground Floor Slab and Framing Plan", "B", "1:100 @ A1", "2026-03-27"),
        sheet("S-102", "First Floor Framing Plan", "B", "1:100 @ A1", "2026-03-27"),
        sheet("S-103", "Roof Framing Plan", "B", "1:100 @ A1", "2026-03-27"),
        sheet("S-200", "Framing, Bracing and Masonry Details", "B", "1:20 @ A1", "2026-03-27"),
        sheet(
            "S-202",
            "OSD Tank Base and Wall Reinforcement",
            "C",
            "1:20 @ A1",
            "2026-08-14",
            notes=["Coordination reference: civil drawing C-201."],
        ),
    ],
    "civil": [
        sheet("C-001", "Civil Notes, Legend and Abbreviations", "C", "NTS", "2026-02-20"),
        sheet("C-100", "Bulk Earthworks and Site Grading Plan", "C", "1:200 @ A1", "2026-02-20"),
        sheet("C-200", "Stormwater Drainage Plan", "C", "1:200 @ A1", "2026-02-20"),
        sheet(
            "C-201",
            "Below-ground OSD Tank Plan, Sections and Outlet Details",
            "C",
            "1:20 @ A1",
            "2026-02-20",
            notes=[
                "Required below-ground on-site detention storage volume: 120 m³.",
                "Tank reference: OSD-01; structural base and wall reinforcement: refer S-202.",
                "Hydraulic control basis: Stormwater Management Report SWM-001 Rev C.",
            ],
        ),
        sheet("C-300", "Erosion and Sediment Control Plan and Details", "C", "1:200 @ A1", "2026-02-20"),
    ],
    "hydraulic": [
        sheet("H-001", "Hydraulic Notes, Legend and Fixture Schedule", "B", "NTS", "2026-03-06"),
        sheet("H-100", "In-ground Sanitary Drainage and Sewer Plan", "B", "1:200 @ A1", "2026-03-06"),
        sheet("H-101", "Ground Floor Hydraulic Services Plan", "B", "1:100 @ A1", "2026-03-06"),
        sheet("H-102", "First Floor Hydraulic Services Plan", "B", "1:100 @ A1", "2026-03-06"),
        sheet("H-200", "Cold Water, Hot Water and Gas Schematics", "B", "NTS", "2026-03-06"),
    ],
    "electrical": [
        sheet("E-001", "Electrical Notes, Legend and Equipment Schedule", "B", "NTS", "2026-03-13"),
        sheet("E-100", "Site Reticulation and External Lighting Plan", "B", "1:200 @ A1", "2026-03-13"),
        sheet("E-101", "Ground Floor Power, Lighting and Communications Plan", "B", "1:100 @ A1", "2026-03-13"),
        sheet("E-102", "First Floor Power, Lighting and Communications Plan", "B", "1:100 @ A1", "2026-03-13"),
        sheet("E-200", "Main Switchboard Schedule and Single-line Diagram", "B", "NTS", "2026-03-13"),
    ],
    "mechanical": [
        sheet("M-001", "Mechanical Notes, Legend and Equipment Schedule", "B", "NTS", "2026-03-20"),
        sheet("M-100", "Ground Floor Mechanical Ventilation Plan", "B", "1:100 @ A1", "2026-03-20"),
        sheet("M-101", "First Floor Mechanical Services Plan", "B", "1:100 @ A1", "2026-03-20"),
        sheet("M-200", "Air-conditioning Zoning and Refrigerant Schematic", "B", "NTS", "2026-03-20"),
        sheet("M-300", "Outdoor Unit Locations and Acoustic Treatment", "B", "1:100 @ A1", "2026-03-20"),
    ],
    "landscape": [
        sheet("L-001", "Landscape Notes and Plant Schedule", "C", "NTS", "2026-01-30"),
        sheet("L-100", "Landscape Masterplan", "C", "1:200 @ A1", "2026-01-30"),
        sheet("L-101", "Deep Soil, Tree Canopy and Planting Plan", "C", "1:200 @ A1", "2026-01-30"),
        sheet("L-200", "Hardscape, Fencing and Retaining Wall Details", "C", "1:20 @ A1", "2026-01-30"),
        sheet("L-300", "Landscape Establishment and Maintenance Plan", "C", "NTS", "2026-01-30"),
    ],
}


PREVIOUS_S_202 = sheet(
    "S-202",
    "OSD Tank Base and Wall Reinforcement",
    "B",
    "1:20 @ A1",
    "2026-03-27",
    notes=["Coordination reference: civil drawing C-201."],
)


REPORTS: list[Report] = [
    {
        "number": "SUR-001",
        "title": "Detail and Level Survey",
        "revision": "A",
        "date": "2025-06-06",
        "purpose": "For Information",
        "author": "WestGrid Surveying",
        "job": "WGS-25071",
        "prepared_by": "D. Lin",
        "pages": 8,
        "scope": "12 attached two-storey Class 1a townhouses — pre-RFI design basis",
        "contents": [
            "Boundary definition and deposited-plan references",
            "Existing levels, structures, services and significant vegetation",
            "Wianamatta Avenue frontage, kerb levels and drainage assets",
        ],
    },
    {
        "number": "GEO-001",
        "title": "Geotechnical Investigation Report",
        "revision": "B",
        "date": "2025-07-18",
        "purpose": "For Design",
        "author": "TerraForma Geotechnics",
        "job": "TFG-25114",
        "prepared_by": "I. Mensah",
        "pages": 34,
        "scope": "12 attached two-storey Class 1a townhouses — pre-RFI design basis",
        "contents": [
            "Field investigation, borehole locations and laboratory testing",
            "Subsurface profile and groundwater observations",
            "Site classification and footing recommendations",
            "Excavation and founding recommendations for OSD-01",
        ],
        "findings": [
            "Uncontrolled fill was encountered to depths between 0.6 m and 1.4 m across the proposed OSD-01 footprint.",
            "The OSD tank support system is to bypass fill and found in competent natural material; weathered shale was anticipated near RL 45.2–45.8 m AHD at the investigation locations.",
            "Final founding levels are to be confirmed by the structural engineer during excavation because local variation between boreholes is possible.",
        ],
    },
    {
        "number": "SEE-001",
        "title": "Statement of Environmental Effects",
        "revision": "D",
        "date": "2025-10-24",
        "purpose": "For Development Application",
        "author": "Civic Pattern Planning",
        "job": "CPP-26019",
        "prepared_by": "C. Alvarez",
        "pages": 58,
        "contents": [
            "Site and locality",
            "Description of the 11-townhouse development",
            "Statutory planning assessment",
            "Blacktown development-control assessment",
            "Environmental effects and mitigation measures",
        ],
    },
    {
        "number": "TRA-001",
        "title": "Traffic and Parking Assessment",
        "revision": "C",
        "date": "2025-10-17",
        "purpose": "For Development Application",
        "author": "Traverse Mobility",
        "job": "TM-25044",
        "prepared_by": "S. Rahman",
        "pages": 29,
        "contents": [
            "Existing road network and traffic conditions",
            "Parking demand and swept-path assessment",
            "Driveway operation, sight distance and service access",
        ],
    },
    {
        "number": "ARB-001",
        "title": "Arboricultural Impact Assessment",
        "revision": "B",
        "date": "2025-09-26",
        "purpose": "For Development Application",
        "author": "Canopy Logic Arborists",
        "job": "CLA-25037",
        "prepared_by": "J. Okoro",
        "pages": 23,
        "contents": [
            "Tree inventory and retention values",
            "Development impacts and tree-protection zones",
            "Protection measures and supervision hold points",
        ],
    },
    {
        "number": "SWM-001",
        "title": "Stormwater Management Report",
        "revision": "C",
        "date": "2026-02-20",
        "purpose": "For Construction",
        "author": "Catchment Works",
        "job": "CW-26012",
        "prepared_by": "S. Moyo",
        "pages": 31,
        "contents": [
            "Existing catchment and lawful point of discharge",
            "Post-development hydrology and permissible site discharge",
            "On-site detention sizing and outlet-control calculations",
            "Water-quality treatment and maintenance requirements",
        ],
        "findings": [
            "The required below-ground on-site detention volume is 120 m³ for the 11-townhouse scheme.",
            "The governing tank layout, sections and outlet-control details are documented on C-201 Rev C.",
            "Structural design of the OSD-01 base and walls is documented separately on S-202.",
        ],
    },
    {
        "number": "WMP-001",
        "title": "Construction and Operational Waste Management Plan",
        "revision": "C",
        "date": "2025-10-24",
        "purpose": "For Development Application",
        "author": "Axis Studio",
        "job": "AXS-26017",
        "prepared_by": "M. Chen",
        "pages": 18,
        "contents": [
            "Demolition and construction waste streams",
            "Operational bin numbers and storage arrangement",
            "Collection route, presentation and caretaker responsibilities",
        ],
    },
    {
        "number": "BAS-001",
        "title": "BASIX Certificate — Certificate SH-2025-11842M",
        "revision": "B",
        "date": "2025-10-20",
        "purpose": "For Development Application",
        "author": "Envelope Performance",
        "job": "EP-25029-B",
        "prepared_by": "V. Tran",
        "pages": 22,
        "contents": [
            "Water commitments",
            "Thermal comfort commitments",
            "Energy commitments for 11 Class 1a dwellings",
        ],
    },
    {
        "number": "ESD-001",
        "title": "NatHERS and Sustainability Design Report",
        "revision": "B",
        "date": "2025-10-20",
        "purpose": "For Development Application",
        "author": "Envelope Performance",
        "job": "EP-25029-E",
        "prepared_by": "V. Tran",
        "pages": 41,
        "contents": [
            "Modelling methodology and dwelling assumptions",
            "Thermal-performance results by townhouse",
            "Glazing, insulation and sealing specifications",
        ],
    },
    {
        "number": "ACO-001",
        "title": "Environmental and Building Acoustic Assessment",
        "revision": "B",
        "date": "2025-10-10",
        "purpose": "For Development Application",
        "author": "Resonance Acoustics",
        "job": "RA-25036",
        "prepared_by": "F. Adeyemi",
        "pages": 27,
        "contents": [
            "Measured ambient noise environment",
            "Road-traffic intrusion assessment",
            "Mechanical plant and party-wall criteria",
        ],
    },
    {
        "number": "BCA-001",
        "title": "NCC 2022 Volume Two Compliance Report",
        "revision": "C",
        "date": "2026-02-06",
        "purpose": "For Construction Certificate",
        "author": "Certus Building Surveying",
        "job": "CBS-26008",
        "prepared_by": "R. Wallace",
        "pages": 46,
        "contents": [
            "Class 1a and associated Class 10 classification",
            "Fire separation, egress and smoke-alarm provisions",
            "Health, amenity and energy-efficiency assessment",
            "Construction-certificate information schedule",
        ],
    },
    {
        "number": "ACC-001",
        "title": "Access and Adaptability Assessment",
        "revision": "B",
        "date": "2025-10-03",
        "purpose": "For Development Application",
        "author": "Equal Path Access",
        "job": "EPA-25022",
        "prepared_by": "H. Baxter",
        "pages": 19,
        "contents": [
            "Pedestrian access from the site boundary",
            "Common-area accessibility",
            "Adaptable-housing provisions and post-adaptation layouts",
        ],
    },
    {
        "number": "QS-001",
        "title": "Pre-tender Cost Plan 03",
        "revision": "C",
        "date": "2026-04-03",
        "purpose": "For Tender",
        "author": "Measureline Quantity Surveying",
        "job": "MQS-250221",
        "prepared_by": "B. Kaur",
        "pages": 37,
        "contents": [
            "Executive cost summary",
            "Elemental estimate and measured quantities",
            "Design allowances, exclusions and risk items",
            "Escalation, contingency and tender reconciliation basis",
        ],
        "findings": [
            "The pre-tender construction estimate is $9,800,000 excluding GST.",
            "The estimate includes $285,000 for OSD-01 excavation, reinforced-concrete tank construction and hydraulic controls based on C-201 Rev C and S-202 Rev B.",
            "Abnormal rock excavation and changes arising from founding levels confirmed after excavation are excluded and retained as a project risk.",
        ],
    },
]


EXPECTED_COUNTS = {
    "architectural": 20,
    "structural": 7,
    "civil": 5,
    "hydraulic": 5,
    "electrical": 5,
    "mechanical": 5,
    "landscape": 5,
}


def slug(text: str) -> str:
    result = "".join(character.lower() if character.isalnum() else "-" for character in text)
    while "--" in result:
        result = result.replace("--", "-")
    return result.strip("-")


def assert_source_data() -> None:
    actual_counts = {discipline: len(sheets) for discipline, sheets in DRAWINGS.items()}
    assert actual_counts == EXPECTED_COUNTS, (actual_counts, EXPECTED_COUNTS)
    assert sum(actual_counts.values()) == 52
    assert len(REPORTS) == 13

    drawing_numbers = [sheet_data["number"] for sheets in DRAWINGS.values() for sheet_data in sheets]
    assert len(drawing_numbers) == len(set(drawing_numbers)), "Current drawing numbers must be unique"
    report_numbers = [report["number"] for report in REPORTS]
    assert len(report_numbers) == len(set(report_numbers)), "Report numbers must be unique"

    current_s_202 = next(item for item in DRAWINGS["structural"] if item["number"] == "S-202")
    c_201 = next(item for item in DRAWINGS["civil"] if item["number"] == "C-201")
    reports_by_number = {report["number"]: report for report in REPORTS}
    assert current_s_202["revision"] == "C" and current_s_202["date"] == "2026-08-14"
    assert PREVIOUS_S_202["revision"] == "B" and PREVIOUS_S_202["date"] == "2026-03-27"
    assert any("120 m³" in note for note in c_201.get("notes", []))
    assert any("120 m³" in finding for finding in reports_by_number["SWM-001"]["findings"])
    assert any("S-202 Rev B" in finding for finding in reports_by_number["QS-001"]["findings"])
    assert any("Uncontrolled fill" in finding for finding in reports_by_number["GEO-001"]["findings"])
    for historical_number in ("SUR-001", "GEO-001"):
        assert reports_by_number[historical_number]["scope"].startswith("12 attached")


def revision_history(drawing: Drawing) -> list[tuple[str, str, str]]:
    if drawing["number"] != "S-202":
        return [(drawing["revision"], drawing["date"], drawing["purpose"])]
    if drawing["revision"] == "B":
        return [
            ("A", "2026-02-27", "Issued for structural coordination"),
            ("B", "2026-03-27", "Issued for construction"),
        ]
    return [
        ("A", "2026-02-27", "Issued for structural coordination"),
        ("B", "2026-03-27", "Issued for construction"),
        ("C", "2026-08-14", "Revised issue for construction"),
    ]


def drawing_document(discipline: str, drawing: Drawing, *, historical: bool = False) -> str:
    firm = FIRMS[discipline]
    label = DISCIPLINE_LABELS[discipline]
    status = "Current at baseline upload" if historical else "Current"
    parts = [
        f"# {drawing['number']} — {drawing['title']}",
        "",
        "> **Synthetic demo artefact.** No real site, company, person or identifier is represented.",
        "",
        f"**{label} drawing · {firm['name']}**",
        "",
        "---",
        "",
        "## Title block",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Project | {PROJECT} |",
        f"| Development | {SCOPE} |",
        f"| Client | {CLIENT} |",
        f"| Contract basis | {CONTRACT} |",
        f"| Drawing title | {drawing['title']} |",
        f"| Drawing number | **{drawing['number']}** |",
        f"| Revision | **{drawing['revision']}** |",
        f"| Revision status | {status} |",
        f"| Discipline | {label} |",
        f"| Scale | {drawing['scale']} |",
        "| Sheet size | A1 |",
        f"| Date | {drawing['date']} |",
        f"| Issue purpose | {drawing['purpose']} |",
        f"| Job number | {firm['job']} |",
        f"| Drawn by | {firm['drawn']} |",
        f"| Checked by | {firm['checked']} |",
        "",
        "## Consultant",
        "",
        f"**{firm['name']}**  ",
        f"{firm['address']}",
        "",
        "## Revision history",
        "",
        "| Rev | Date | Description |",
        "| :---: | --- | --- |",
    ]
    parts.extend(f"| {rev} | {date} | {description} |" for rev, date, description in revision_history(drawing))

    if drawing.get("notes"):
        parts.extend(["", "## Indexed coordination notes", ""])
        parts.extend(f"- {note}" for note in drawing["notes"])

    parts.extend(
        [
            "",
            "---",
            "",
            "*Drawing geometry and construction detail are intentionally not reproduced. This record exists",
            "only to populate and test document control, revision tracking and evidence retrieval.*",
            "",
            f"© {firm['name']} — synthetic demonstration record only.",
            "",
        ]
    )
    return "\n".join(parts)


def report_document(report: Report) -> str:
    parts = [
        f"# {report['number']} — {report['title']}",
        "",
        "> **Synthetic demo artefact.** No real site, company, person or identifier is represented.",
        "",
        f"**{report['author']}**",
        "",
        "---",
        "",
        "## Document control",
        "",
        "| Field | Value |",
        "| --- | --- |",
        f"| Project | {PROJECT} |",
        f"| Development | {report.get('scope', SCOPE)} |",
        f"| Client | {CLIENT} |",
        f"| Contract basis | {CONTRACT} |",
        f"| Document title | {report['title']} |",
        f"| Document number | **{report['number']}** |",
        f"| Revision | **{report['revision']}** |",
        "| Revision status | Current |",
        f"| Date | {report['date']} |",
        f"| Issue purpose | {report['purpose']} |",
        f"| Author | {report['author']} |",
        f"| Job number | {report['job']} |",
        f"| Prepared by | {report['prepared_by']} |",
        f"| Pages in source report | {report['pages']} |",
        "",
        "## Contents represented",
        "",
    ]
    parts.extend(f"{index}. {item}" for index, item in enumerate(report["contents"], start=1))

    if report.get("findings"):
        parts.extend(["", "## Findings carried into the demo evidence graph", ""])
        parts.extend(f"- {finding}" for finding in report["findings"])

    parts.extend(
        [
            "",
            "---",
            "",
            "*The full report body is intentionally not reproduced. Except for findings explicitly stated",
            "above, this record carries document-control information only.*",
            "",
            f"© {report['author']} — synthetic demonstration record only.",
            "",
        ]
    )
    return "\n".join(parts)


def register_document() -> str:
    parts = [
        "# Seven Hills Design Document Register — Answer Key",
        "",
        "> **Entirely synthetic demonstration corpus.** The address, project, firms, people, job",
        "> numbers, certificate number and technical particulars are fictional test data.",
        "",
        f"**Project:** {PROJECT}  ",
        f"**Development:** {SCOPE}  ",
        f"**Client:** {CLIENT}  ",
        f"**Contract basis:** {CONTRACT}  ",
        f"**Register date:** {REGISTER_DATE}",
        "",
        "Generated by `../generate-design-documents.py`. Do not hand-edit generated files.",
        "Change the source data and rerun the generator.",
        "",
        "This register is the answer key, not project evidence. Do not upload it into the demo",
        "project. The repository should reproduce its current drawing and report rows after the",
        "staged upload sequence below.",
        "",
        "---",
        "",
        "## Control totals",
        "",
        "| Document group | Current records | Historical staged records |",
        "| --- | ---: | ---: |",
    ]
    for discipline, expected in EXPECTED_COUNTS.items():
        parts.append(f"| {DISCIPLINE_LABELS[discipline]} drawings | {expected} | {'1' if discipline == 'structural' else '0'} |")
    parts.extend(
        [
            f"| Reports and assessments | {len(REPORTS)} | 0 |",
            "| **Total** | **65** | **1** |",
            "",
            "There are exactly **52 current drawing sheets**, **13 current reports** and one staged",
            "historical drawing revision. Uploading S-202 Rev C supersedes Rev B; it does not create",
            "a 53rd current sheet.",
            "",
            "---",
            "",
            "## Current drawings",
        ]
    )

    for discipline, drawings in DRAWINGS.items():
        firm = FIRMS[discipline]
        label = DISCIPLINE_LABELS[discipline]
        parts.extend(
            [
                "",
                f"### {label} — {firm['name']}",
                "",
                f"Job number `{firm['job']}` · {len(drawings)} current sheets",
                "",
                "| Number | Title | Rev | Date | Scale | Issue purpose |",
                "| --- | --- | :---: | --- | --- | --- |",
            ]
        )
        parts.extend(
            f"| `{drawing['number']}` | {drawing['title']} | **{drawing['revision']}** | "
            f"{drawing['date']} | {drawing['scale']} | {drawing['purpose']} |"
            for drawing in drawings
        )

    parts.extend(
        [
            "",
            "---",
            "",
            "## Reports and assessments",
            "",
            "| Number | Title | Rev | Date | Author / job | Issue purpose |",
            "| --- | --- | :---: | --- | --- | --- |",
        ]
    )
    parts.extend(
        f"| `{report['number']}` | {report['title']} | **{report['revision']}** | "
        f"{report['date']} | {report['author']} / `{report['job']}` | {report['purpose']} |"
        for report in REPORTS
    )

    parts.extend(
        [
            "",
            "---",
            "",
            "## OSD evidence chain — expected retrieval answer",
            "",
            "The same risk must remain traceable rather than being repeated as disconnected facts:",
            "",
            "1. `GEO-001` records variable fill and requires OSD-01 to found in competent natural",
            "   material, with final levels confirmed during excavation.",
            "2. `SWM-001` calculates **120 m³** of below-ground detention for the current 11-townhouse",
            "   scheme and identifies `C-201` as the governing civil detail.",
            "3. `C-201` defines OSD-01 as a **120 m³ below-ground tank** and refers its base and wall",
            "   reinforcement to `S-202`.",
            "4. `S-202` is the structural base-and-wall reinforcement sheet. Rev B dated 2026-03-27",
            "   is the baseline issue; Rev C dated 2026-08-14 is current.",
            "5. `QS-001` carries a **$285,000** OSD allowance and excludes abnormal rock excavation",
            "   or changes arising from founding levels confirmed after excavation.",
            "",
            "The reason for the S-202 Rev C change belongs to the separately staged Northline advice",
            "and Pulse email, not to this title-block record. This keeps source attribution honest.",
            "",
            "---",
            "",
            "## Staged upload / demo run sequence",
            "",
            "### Run 1 — establish the baseline register",
            "",
            "1. Upload all 13 files under `reports/`.",
            "2. Upload every file under `drawings/` **except** the current",
            "   `drawings/structural/S-202-osd-tank-base-and-wall-reinforcement.md`.",
            "3. Upload `staged-revisions/01-baseline/S-202-rev-b-osd-tank-base-and-wall-reinforcement.md`.",
            "4. Wait for ingestion, then ask for the drawing register and the OSD evidence chain.",
            "",
            "Expected baseline: 52 current drawing sheets, S-202 at Rev B, 13 current reports and",
            "no superseded drawing revision yet.",
            "",
            "### Run 2 — trigger the Pulse revision event",
            "",
            "1. Deliver the staged Pulse email with the current",
            "   `drawings/structural/S-202-osd-tank-base-and-wall-reinforcement.md` attached.",
            "   Canonical email-attachment intake must create the document; do not upload it again.",
            "2. Ask what changed and what downstream cost, programme and construction items require",
            "   review. Do not upload this answer key.",
            "",
            "Expected current state: 52 current sheets, S-202 Rev C current, Rev B superseded, 13",
            "current reports. The revision event should cite the drawing and the separate advice",
            "before proposing any cost or programme update.",
            "",
            "---",
            "",
            "## Revision distribution at current state",
            "",
            "| Current revision | Sheets |",
            "| :---: | ---: |",
        ]
    )
    revision_counts: dict[str, int] = {}
    for drawings in DRAWINGS.values():
        for drawing in drawings:
            revision = drawing["revision"]
            revision_counts[revision] = revision_counts.get(revision, 0) + 1
    parts.extend(f"| **{revision}** | {count} |" for revision, count in sorted(revision_counts.items()))
    parts.extend(
        [
            "",
            "## Corpus boundaries",
            "",
            "- Drawing records are title blocks, not usable design drawings.",
            "- Only GEO-001, SWM-001, C-201 and QS-001 carry the short technical or commercial",
            "  findings needed for the OSD demonstration. Other records are document-control stubs.",
            "- All arithmetic and counts in this answer key are generated from the same source lists",
            "  as the files, so the register cannot drift from the corpus silently.",
            "- Every name, company, address, identifier and technical particular is synthetic.",
            "",
        ]
    )
    return "\n".join(parts)


def reset_managed_outputs() -> None:
    OUTPUT_ROOT.mkdir(parents=True, exist_ok=True)
    managed_directories = [
        OUTPUT_ROOT / "drawings",
        OUTPUT_ROOT / "reports",
        OUTPUT_ROOT / "staged-revisions",
    ]
    for directory in managed_directories:
        assert directory.parent == OUTPUT_ROOT
        if directory.exists():
            shutil.rmtree(directory)
    register_path = OUTPUT_ROOT / "document-register.md"
    if register_path.exists():
        register_path.unlink()


def write_outputs() -> None:
    for discipline, drawings in DRAWINGS.items():
        discipline_dir = OUTPUT_ROOT / "drawings" / discipline
        discipline_dir.mkdir(parents=True, exist_ok=True)
        for drawing in drawings:
            path = discipline_dir / f"{drawing['number']}-{slug(drawing['title'])}.md"
            path.write_text(drawing_document(discipline, drawing), encoding="utf-8")

    reports_dir = OUTPUT_ROOT / "reports"
    reports_dir.mkdir(parents=True, exist_ok=True)
    for report in REPORTS:
        path = reports_dir / f"{report['number']}-{slug(report['title'])}.md"
        path.write_text(report_document(report), encoding="utf-8")

    staged_dir = OUTPUT_ROOT / "staged-revisions" / "01-baseline"
    staged_dir.mkdir(parents=True, exist_ok=True)
    staged_path = staged_dir / "S-202-rev-b-osd-tank-base-and-wall-reinforcement.md"
    staged_path.write_text(
        drawing_document("structural", PREVIOUS_S_202, historical=True),
        encoding="utf-8",
    )

    (OUTPUT_ROOT / "document-register.md").write_text(register_document(), encoding="utf-8")


def assert_generated_outputs() -> None:
    drawing_files = list((OUTPUT_ROOT / "drawings").rglob("*.md"))
    report_files = list((OUTPUT_ROOT / "reports").glob("*.md"))
    staged_files = list((OUTPUT_ROOT / "staged-revisions").rglob("*.md"))
    assert len(drawing_files) == 52, len(drawing_files)
    assert len(report_files) == 13, len(report_files)
    assert len(staged_files) == 1, len(staged_files)

    current_s_202 = (OUTPUT_ROOT / "drawings" / "structural" / "S-202-osd-tank-base-and-wall-reinforcement.md").read_text(encoding="utf-8")
    previous_s_202 = staged_files[0].read_text(encoding="utf-8")
    c_201 = (OUTPUT_ROOT / "drawings" / "civil" / "C-201-below-ground-osd-tank-plan-sections-and-outlet-details.md").read_text(encoding="utf-8")
    survey = (OUTPUT_ROOT / "reports" / "SUR-001-detail-and-level-survey.md").read_text(encoding="utf-8")
    geotechnical = (OUTPUT_ROOT / "reports" / "GEO-001-geotechnical-investigation-report.md").read_text(encoding="utf-8")
    assert "| Revision | **C** |" in current_s_202 and "2026-08-14" in current_s_202
    assert "| Revision | **B** |" in previous_s_202 and "2026-03-27" in previous_s_202
    assert "120 m³" in c_201
    for historical_report in (survey, geotechnical):
        assert "12 attached two-storey Class 1a townhouses" in historical_report
        assert "11 attached two-storey Class 1a townhouses" not in historical_report


def main() -> None:
    assert_source_data()
    reset_managed_outputs()
    write_outputs()
    assert_generated_outputs()
    print("Generated Seven Hills design corpus")
    print("  current drawings: 52")
    print("  current reports:  13")
    print("  staged revisions: 1 (S-202 Rev B)")


if __name__ == "__main__":
    main()
