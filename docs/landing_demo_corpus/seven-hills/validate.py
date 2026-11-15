"""Validate the generated Seven Hills demonstration corpus.

This is intentionally stdlib-only. It checks the cross-generator contracts that are easy
to miss when the commercial, design and narrative sets are regenerated independently.

    python docs/demo-corpus/seven-hills/validate.py
"""

from __future__ import annotations

import re
from collections import Counter
from datetime import date, timedelta
from pathlib import Path


ROOT = Path(__file__).resolve().parent
PROJECT = "14–18 Wianamatta Avenue, Seven Hills NSW 2147"
CLIENT = "Wianamatta Developments Pty Ltd"
PM = "Ridgeline Project Management Pty Ltd"


def md_files(relative: str) -> list[Path]:
    return sorted((ROOT / relative).rglob("*.md"))


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def require_count(label: str, paths: list[Path], expected: int) -> None:
    assert len(paths) == expected, f"{label}: expected {expected}, found {len(paths)}"


def project_evidence() -> list[Path]:
    folders = (
        "01-briefing-and-planning",
        "02-consultant-procurement",
        "03-consultant-invoices",
        "05-planning-and-approvals",
        "06-builder-procurement",
        "07-construction-commercial",
        "08-project-controls",
    )
    evidence = [path for folder in folders for path in md_files(folder)]
    evidence.extend(md_files("04-design-documents/drawings"))
    evidence.extend(md_files("04-design-documents/reports"))
    evidence.extend(md_files("04-design-documents/staged-revisions"))
    return sorted(evidence)


def check_counts() -> None:
    require_count("obsolete 08-change-advice outputs", md_files("08-change-advice"), 0)

    expected = {
        "01-briefing-and-planning": 8,
        "02-consultant-procurement": 20,
        "03-consultant-invoices": 25,
        "05-planning-and-approvals": 7,
        "06-builder-procurement": 5,
        "07-construction-commercial": 4,
        "08-project-controls": 5,
        "00-prompts": 7,
        "00-answer-keys": 6,
        "09-email-scenarios": 5,
    }
    for folder, count in expected.items():
        require_count(folder, md_files(folder), count)

    drawing_counts = {
        "architectural": 20,
        "structural": 7,
        "civil": 5,
        "hydraulic": 5,
        "electrical": 5,
        "mechanical": 5,
        "landscape": 5,
    }
    for discipline, count in drawing_counts.items():
        require_count(
            f"drawings/{discipline}",
            md_files(f"04-design-documents/drawings/{discipline}"),
            count,
        )
    require_count("current drawings", md_files("04-design-documents/drawings"), 52)
    require_count("reports", md_files("04-design-documents/reports"), 13)
    require_count("staged revisions", md_files("04-design-documents/staged-revisions"), 1)
    require_count("ingestible evidence", project_evidence(), 140)

    disciplines = (
        "architectural-services",
        "town-planning",
        "structural-engineering",
        "civil-stormwater-engineering",
        "building-services-engineering",
    )
    for discipline in disciplines:
        require_count(
            f"{discipline} proposals",
            md_files(f"02-consultant-procurement/{discipline}/proposals"),
            3,
        )
        require_count(
            f"{discipline} appointment",
            md_files(f"02-consultant-procurement/{discipline}/appointment"),
            1,
        )
        require_count(
            f"{discipline} invoices",
            md_files(f"03-consultant-invoices/{discipline}"),
            5,
        )


def check_identity_and_marking() -> None:
    for path in project_evidence():
        body = read(path)
        assert PROJECT in body, f"project identity missing: {path.relative_to(ROOT)}"
        assert "synthetic" in body.lower(), f"synthetic marker missing: {path.relative_to(ROOT)}"

    all_source = "\n".join(
        read(path)
        for path in sorted(ROOT.rglob("*"))
        if path.is_file() and path != Path(__file__) and path.suffix in {".md", ".py"}
    )
    forbidden = {
        "old client identity": r"Wianamatta Living",
        "superseded real-address seed": r"Chelmsford|South Wentworthville",
        "non-canonical variation number": r"VO-07(?!7)",
        "non-existent civil revision": r"C-201\s+Rev\s+D",
        "superseded agent runtime": r"\bHermes\b",
        "non-canonical firm alias": (
            r"Axis\s+Studio\s+Architects|Ironbark Projects|Flux Building Services|Redgum Build Co\."
        ),
        "legacy unrelated VO-007 scope": r"unsuitable[- ]material",
        "nonexistent drawing reference": r"\bE-300\b",
        "non-canonical QS identity": r"Measureline Cost Planning|MCP-25055",
    }
    for label, pattern in forbidden.items():
        assert not re.search(pattern, all_source, re.IGNORECASE), f"found {label}"

    assert CLIENT in read(ROOT / "01-briefing-and-planning/01-client-development-brief.md")
    assert CLIENT in read(ROOT / "04-design-documents/document-register.md")
    assert CLIENT in read(ROOT / "00-answer-keys/commercial-register.md")
    assert PM in read(ROOT / "00-answer-keys/commercial-register.md")


def check_design_register() -> None:
    drawing_number = re.compile(r"\| Drawing number \| \*\*([^*]+)\*\* \|")
    numbers: list[str] = []
    for path in md_files("04-design-documents/drawings"):
        match = drawing_number.search(read(path))
        assert match, f"drawing number missing: {path.relative_to(ROOT)}"
        numbers.append(match.group(1))
    duplicates = [number for number, count in Counter(numbers).items() if count > 1]
    assert not duplicates, f"duplicate current drawing numbers: {duplicates}"

    baseline = read(
        ROOT
        / "04-design-documents/staged-revisions/01-baseline/"
        "S-202-rev-b-osd-tank-base-and-wall-reinforcement.md"
    )
    current = read(
        ROOT
        / "04-design-documents/drawings/structural/"
        "S-202-osd-tank-base-and-wall-reinforcement.md"
    )
    civil = read(
        ROOT
        / "04-design-documents/drawings/civil/"
        "C-201-below-ground-osd-tank-plan-sections-and-outlet-details.md"
    )
    assert "| Revision | **B** |" in baseline
    assert "| Revision | **C** |" in current
    assert "120 m³" in civil or "120 cubic metre" in civil
    qs_report = read(ROOT / "04-design-documents/reports/QS-001-pre-tender-cost-plan-03.md")
    assert "Measureline Quantity Surveying" in qs_report
    assert "MQS-250221" in qs_report
    see_report = read(ROOT / "04-design-documents/reports/SEE-001-statement-of-environmental-effects.md")
    assert "CPP-26019" in see_report


def check_commercial_truth() -> None:
    proposals = [
        path
        for path in md_files("02-consultant-procurement")
        if "proposals" in path.parts
    ]
    for path in proposals:
        body = read(path)
        assert not re.search(
            r"\b(?:appointed|successful tenderer|selected consultant)\b",
            body,
            re.IGNORECASE,
        ), f"proposal contains outcome knowledge: {path.relative_to(ROOT)}"

    appointments = [
        path
        for path in md_files("02-consultant-procurement")
        if "appointment" in path.parts
    ]
    historical_procurement = proposals + appointments
    assert len(historical_procurement) == 20
    for path in historical_procurement:
        body = read(path)
        assert "twelve two-storey attached townhouses" in body.lower()
        assert "RFI responses and ongoing design development" in body
        assert not re.search(r"\beleven\b|120\s*(?:cubic|m³)", body, re.IGNORECASE), (
            f"future fact leaked into historical procurement: {path.relative_to(ROOT)}"
        )

    invoice_fives = [
        path for path in md_files("03-consultant-invoices") if path.stem.endswith("INV-05")
    ]
    require_count("consultant INV-05 files", invoice_fives, 5)
    for path in invoice_fives:
        body = read(path)
        assert "| Issue date | 2026-05-15 |" in body
        assert "| Due date | 2026-05-29 |" in body

    commercial = read(ROOT / "00-answer-keys/commercial-register.md")
    tender = read(ROOT / "00-answer-keys/tender-comparison-answer-key.md")
    change = read(ROOT / "00-answer-keys/live-change-loop.md")
    for required in ("$720,000", "$3,408,500", "$9,500,000", "VO-007"):
        assert required in commercial, f"commercial reconciliation missing {required}"
    for required in ("$9,080,000", "+$420,000", "$9,500,000", "$9,340,000"):
        assert required in tender, f"tender reconciliation missing {required}"
    for required in ("$68,500", "10-calendar-day", "Draft — not sent"):
        assert required in change, f"live-change reconciliation missing {required}"


def check_email_change_pack() -> None:
    inbound = read(ROOT / "09-email-scenarios/01-inbound-structural-transmittal.md")
    required_paths = (
        "04-design-documents/drawings/structural/"
        "S-202-osd-tank-base-and-wall-reinforcement.md",
        "08-project-controls/01-structural-design-change-notice-dcn-007.md",
        "08-project-controls/02-qs-cost-advice-ca-014.md",
        "08-project-controls/03-architect-programme-note-pn-006.md",
    )
    for relative in required_paths:
        assert relative in inbound, f"inbound email missing attachment reference: {relative}"
        assert (ROOT / relative).is_file(), f"email attachment does not exist: {relative}"
    assert "2026-08-15 16:42 AEST" in inbound

    claim_email = read(ROOT / "09-email-scenarios/03-inbound-builder-progress-claim-04.md")
    claim_path = "07-construction-commercial/progress-claims/PC-04-IBG-PC-04.md"
    assert claim_path in claim_email, "Progress Claim 04 email has the wrong attachment"
    assert (ROOT / claim_path).is_file(), "Progress Claim 04 attachment does not exist"

    tender_wave = read(ROOT / "09-email-scenarios/04-builder-tender-return-wave.md")
    for bidder in (
        "Redgum Constructions Pty Ltd",
        "Ironbark Building Group Pty Ltd",
        "Calderline Projects Pty Ltd",
    ):
        assert bidder in tender_wave, f"tender-return email uses the wrong bidder name: {bidder}"

    unanswered = read(ROOT / "09-email-scenarios/05-unanswered-consultant-action.md")
    assert "E-200" in unanswered and "E-300" not in unanswered

    handover = read(ROOT / "01-briefing-and-planning/02-acquisition-handover-email.md")
    assert "2025-02-19 07:18 AEDT" in handover
    handover_attachments = (
        "03-deposited-plan-extract.md",
        "04-title-search-summary.md",
        "05-preliminary-planning-advice.md",
        "06-desktop-geotechnical-advice.md",
    )
    for name in handover_attachments:
        assert name in handover, f"acquisition handover missing {name}"
        assert (ROOT / "01-briefing-and-planning" / name).is_file()


def check_story_choreography() -> None:
    programme = read(ROOT / "08-project-controls/03-architect-programme-note-pn-006.md")
    assert "Finish-to-start predecessor" in programme
    assert "Approved civil OSD design — C-201 Rev C" in programme

    timeline = read(ROOT / "00-answer-keys/timeline.md")
    ordered_events = (
        "2026-03-27 | Construction Certificate and S-202 Rev B issued",
        "2026-04-17 | Three builder tenders returned",
        "2026-04-24 | Redgum prices its stated OSD exclusion",
        "2026-05-04 | Ironbark accepted at $9.340m excl GST",
        "2026-05-05 | Construction starts",
    )
    positions = [timeline.index(event) for event in ordered_events]
    assert positions == sorted(positions), "master timeline is not chronological"

    run_sheet = read(ROOT / "00-storyboard/run-sheet.md")
    assert re.search(r"structural engineering and\s+building services engineering", run_sheet)
    assert re.search(r"5 each: hydraulic, electrical, mechanical and landscape", run_sheet)
    assert "05-planning-and-approvals/" in run_sheet
    assert "12-dwelling acquisition target" in run_sheet
    assert "Do not separately upload those four attachments again" in run_sheet
    assert "Do **not** ingest the Ironbark letter of acceptance yet" in run_sheet
    assert "S-202 Rev B visible as current at the baseline" in run_sheet
    story = read(ROOT / "00-storyboard/README.md")
    assert "the other two include it" in story
    assert "one carries a stated provisional sum" not in story
    assert "text extraction/OCR" not in story
    assert "mixed filenames as received" not in run_sheet
    assert "drawing splitting" not in run_sheet

    prompt_three = read(ROOT / "00-prompts/03-consultant-procurement.md")
    assert "02-consultant-procurement/proposals/" not in prompt_three
    assert "02-consultant-procurement/<discipline>/proposals/" in prompt_three
    prompt_four = read(ROOT / "00-prompts/04-document-register.md")
    assert "substitute the staged S-202 Rev B baseline" in prompt_four

    design_register = read(ROOT / "04-design-documents/document-register.md")
    assert "Canonical email-attachment intake must create the document" in design_register
    assert "do not upload it again" in design_register

    root_readme = read(ROOT / "README.md")
    assert "Planning: ingest folder 05 in date order" in root_readme
    assert "Keep WD-LOA-001 out" in root_readme

    baseline_completion = date(2026, 5, 5) + timedelta(weeks=58)
    changed_completion = baseline_completion + timedelta(days=10)
    assert baseline_completion == date(2027, 6, 15)
    assert changed_completion == date(2027, 6, 25)
    programme = read(ROOT / "08-project-controls/03-architect-programme-note-pn-006.md")
    live_key = read(ROOT / "00-answer-keys/live-change-loop.md")
    for body in (programme, live_key):
        assert "15 June 2027" in body
        assert "25 June 2027" in body
        assert "10 May 2027" not in body
        assert "practical completion moves" not in body.lower()
    assert "forecast completion" in programme
    assert re.search(r"contractual\s+practical\s+completion\s+remains", programme, re.I)
    assert "Forecast completion moves" in live_key
    assert re.search(r"contractual\s+practical\s+completion\s+remains", live_key, re.I)

    programme_baseline = read(
        ROOT / "08-project-controls/00-IBG-PROG-B01-reviewed-construction-programme.md"
    )
    for required in (
        "IBG-PROG-B01",
        "PRG-210",
        "PRG-220",
        "58 calendar weeks",
        "2026-08-24",
        "2026-09-20",
        "2027-06-15",
    ):
        assert required in programme_baseline
    assert "2026-08-16" not in programme_baseline
    assert "PRG-210" in programme and "PRG-220" in programme


def check_markdown_links() -> None:
    link_pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
    broken: list[str] = []
    for source in ROOT.rglob("*.md"):
        for raw_target in link_pattern.findall(read(source)):
            target = raw_target.strip().strip("<>")
            if target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            target = target.split("#", 1)[0]
            if not target:
                continue
            resolved = (source.parent / target).resolve()
            if not resolved.exists():
                broken.append(f"{source.relative_to(ROOT)} -> {raw_target}")
    assert not broken, "broken local Markdown links:\n" + "\n".join(broken)


def main() -> None:
    check_counts()
    check_identity_and_marking()
    check_design_register()
    check_commercial_truth()
    check_email_change_pack()
    check_story_choreography()
    check_markdown_links()
    print("Seven Hills corpus validation passed")
    print("  ingestible evidence: 140")
    print("  current drawings:    52")
    print("  reports:             13")
    print("  consultant invoices: 25")
    print("  builder tenders:     3")


if __name__ == "__main__":
    main()
