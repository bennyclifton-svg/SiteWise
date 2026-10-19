"""Validate knowledge/ against knowledge/SCHEMA.md.

Interim dev tool for the extraction phase; the Go knowledge loader replaces it.
Usage: python tools/check_knowledge.py [--seed-dir PATH] [--strict] [--only PATH_PREFIX]
Errors fail the run. Unresolved cross-references are warnings (another cluster
may define them) unless --strict.
"""

from __future__ import annotations

import argparse
import csv
import datetime
import hashlib
import json
import re
import sys
from decimal import Decimal
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
KNOWLEDGE = ROOT / "knowledge"
SOURCE_ARCHIVE = ROOT / "data" / "reference" / "clerk"
DEFAULT_SEED_DIR = SOURCE_ARCHIVE / "data" / "seed"

LIST_KEYS = {"systems", "rules", "interfaces", "failure_modes", "determinants"}
ID_PATTERNS = {
    "systems": re.compile(r"^[a-z0-9-]+(\.[a-z0-9-]+)*$"),
    "determinants": re.compile(r"^[a-z][a-z0-9_]*$"),
    "rules": re.compile(r"^rule\.[a-z0-9-]+\.[a-z0-9-]+$"),
    "interfaces": re.compile(r"^if\.[a-z0-9-]+$"),
    "failure_modes": re.compile(r"^fm\.[a-z0-9-]+$"),
}
REQUIRED = {
    "systems": {"id", "label", "describes", "status", "sources"},
    "determinants": {"id", "label", "value", "status", "sources"},
    "rules": {"id", "title", "instrument", "clause", "clause_verified", "systems", "requires", "status", "sources"},
    "interfaces": {"id", "type", "from", "to", "summary", "status", "sources"},
    "failure_modes": {"id", "attaches_to", "severity", "detector", "flag_when", "status", "sources"},
}
INTERFACE_TYPES = {"penetrates", "sequences", "loads", "supplies", "controls", "depends_on", "shares_space", "boundary"}
SEVERITIES = {"life-safety", "compliance", "durability", "cost", "programme"}
VALUE_TYPES = {"integer", "number", "boolean", "choice", "multi_choice"}
STATUSES = {"draft", "reviewed", "deprecated"}
PROFILE_GROUPS = {"classification", "site", "services", "fire"}
PROFILE_DIR = "profile"
# Ids of private evaluation documents a record may cite as a source. None
# means the manifest is absent, so a document source cannot be checked.
DOCUMENT_IDS: set | None = None
INTAKE = ROOT / "data" / "intake"
_VOCAB: dict = {}
QUESTION_TYPES = {"noul", "choice", "score"}
PREDICATE_OPS = {"any_of", "eq", "is", "gt", "gte", "lt", "lte"}

# Works layer (SCHEMA.md "Works layer"). Vocabularies are fixed here and in the
# schema together; ACTIONS is read from knowledge/works/actions.yaml.
WORKS_DIR = "works"
CLUSTER_WORKS_FILES = {"consequences.yaml": "consequences", "unforeseen.yaml": "unforeseen", "signals.yaml": "signals"}
ACTIONS: set = set()
DATASETS: dict = {}          # dataset id -> set of row ids, from data/unforeseen/manifest.json
UNFORESEEN_DIR = ROOT / "data" / "unforeseen"
PROPOSAL_KINDS = {"investigation", "discipline", "approval", "hold_point", "obligation"}
INTERFACE_PROPOSAL_KINDS = PROPOSAL_KINDS | {"work_item"}
TOUCHES = {"from", "to", "either"}
UC_KINDS = {"unforeseen_condition", "design_or_coordination_error", "workmanship_defect", "process_authority_supply_weather"}
UC_CATEGORIES = {"ground_and_site", "existing_structure", "hazardous_materials", "concealed_services_and_earlier_work",
                 "existing_systems_on_test", "authorities_and_utilities", "third_parties_and_occupation",
                 "design_and_scope", "supply_and_site_operations"}
UC_EFFECTS = {"cost", "programme", "safety", "compliance", "quality"}
DISCOVERED_AT = {"design", "demolition_strip_out", "excavation", "construction", "testing_commissioning", "handover_defects"}
UC_STAGES = {"investigation", "design", "approvals", "procurement", "construction", "completion", "defects"}
PACKAGE_KINDS = {"services", "works", "supply"}
LEDGER_REJECTIONS = {"duplicate_of", "too_vague", "out_of_scope"}
WORKS_ID = {
    "ic": re.compile(r"^ic\.[a-z0-9-]+$"),
    "cq": re.compile(r"^cq\.[a-z0-9-]+$"),
    "uc": re.compile(r"^uc\.[a-z0-9-]+$"),
    "sig": re.compile(r"^sig\.[a-z0-9-]+$"),
}
ROW_ID = re.compile(r"^[A-Z][A-Z0-9]*-\d{4,}$")

# Program library (SCHEMA.md "Program library (2026-10-18)").
PROGRAM_DIR = "program"
PROGRAM_FILES = {"durations.yaml": "durations", "fragments.yaml": "fragments",
                 "skeletons.yaml": "skeletons", "calendars.yaml": "calendars"}
PROGRAM_ID = {
    "skeletons": re.compile(r"^sk\.[a-z0-9-]+$"),
    "fragments": re.compile(r"^fr\.[a-z0-9-]+$"),
    "durations": re.compile(r"^du\.[a-z0-9-]+$"),
    "calendars": re.compile(r"^cal\.[a-z0-9-]+$"),
}
PROGRAM_STATUSES = {"draft", "reviewed"}
GENERIC_SKELETON = "sk.generic"
ACTIVITY_KEY = re.compile(r"^[a-z0-9-]+$")
LINK_TYPES = {"FS", "SS", "FF", "SF"}
ATTACH_ENDS = {"stage_start", "stage_finish"}
FRAGMENT_ACTIONS = {"new", "replace", "upgrade", "alter", "repair", "remove"}
DURATION_BASES = {"fixed", "per_unit", "floor_cycle", "time_cost", "statutory", "lead_time"}
DAY_DECIMAL = re.compile(r"^[0-9]+(\.[0-9]{1,2})?$")
# Time-cost K and B keep up to four places so a published set is not re-rounded.
PARAM_DECIMAL = re.compile(r"^[0-9]+(\.[0-9]{1,4})?$")
ISO_DATE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
CALENDAR_KINDS = {"public_holiday", "rdo", "shutdown"}
CALENDAR_STATES = {"NSW", "VIC", "QLD", "SA", "WA", "TAS", "ACT", "NT"}
WEEKDAYS = {"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
PROGRAM_KEYS = {
    "skeletons": {"id", "label", "applies_when", "stages", "links", "status", "sources", "notes"},
    "fragments": {"id", "label", "stage", "systems", "applies_when", "activities", "links", "attach",
                  "status", "sources", "notes"},
    "durations": {"id", "version", "basis", "driver", "unit_days", "low", "typical", "high", "k", "b",
                  "cost_base_date", "index_basis", "instrument", "clause", "verified", "primary_source",
                  "applies_when", "geography", "limitations", "status", "sources", "notes"},
    "calendars": {"id", "state", "year", "working_week", "days", "verified", "status", "sources", "notes"},
}


class Report:
    def __init__(self) -> None:
        self.errors: list[str] = []
        self.warnings: list[str] = []

    def error(self, where: str, msg: str) -> None:
        self.errors.append(f"{where}: {msg}")

    def warn(self, where: str, msg: str) -> None:
        self.warnings.append(f"{where}: {msg}")


def load_files(report: Report) -> list[tuple[Path, str, list[dict]]]:
    loaded = []
    for path in sorted(KNOWLEDGE.rglob("*.yaml")):
        rel = path.relative_to(ROOT).as_posix()
        parts = path.relative_to(KNOWLEDGE).parts[:-1]
        # RFP discipline files have their own shape and check (check_rfp).
        if path.parent.name in ("tables", PROFILE_DIR) or WORKS_DIR in parts or set(CATALOGUE_DIRS) & set(parts) or "rfp" in parts \
                or PROGRAM_DIR in parts:
            continue
        if path.name in CLUSTER_WORKS_FILES and path.parent.parent.name == "clusters":
            continue
        try:
            doc = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            report.error(rel, f"invalid YAML: {exc}")
            continue
        if not isinstance(doc, dict) or doc.get("version") != 1:
            report.error(rel, "must be a mapping with version: 1")
            continue
        keys = LIST_KEYS & doc.keys()
        if len(keys) != 1:
            report.error(rel, f"must have exactly one of {sorted(LIST_KEYS)}")
            continue
        kind = keys.pop()
        items = doc[kind] or []
        if not isinstance(items, list):
            report.error(rel, f"{kind} must be a list")
            continue
        if path.name == "proposed_determinants.yaml" and kind != "determinants":
            report.error(rel, "proposed_determinants.yaml must hold `determinants`")
        loaded.append((path, kind, items))
    return loaded


def load_document_ids() -> set | None:
    path = ROOT / "data" / "eval" / "profile" / "manifest.json"
    if not path.is_file():
        return None
    return {d["id"] for d in json.loads(path.read_text(encoding="utf-8")).get("documents", [])}


def vocab(name: str) -> set:
    """Kind or discipline ids from data/intake, so stated_in cannot drift from intake."""
    if name not in _VOCAB:
        doc = json.loads((INTAKE / f"{name}.json").read_text(encoding="utf-8"))
        _VOCAB[name] = {entry["id"] for entry in doc[name]}
    return _VOCAB[name]


def check_detail_reading(where: str, d: dict, report: Report) -> None:
    """A project or site detail is read from documents only when it says how:
    pre_parsed text, a keyword gate and the question Jev is asked."""
    if not d.get("extraction"):
        if d.get("triggers") or d.get("asks"):
            report.error(where, "triggers or asks without extraction")
        return
    if d["extraction"] != "pre_parsed" or d.get("value") != "text":
        report.error(where, "a read detail is text with extraction pre_parsed")
    if not str(d.get("asks", "")).strip():
        report.error(where, "a read detail needs the question it asks")
    check_triggers(where, d.get("triggers"), report)


def check_triggers(where: str, triggers, report: Report) -> None:
    # Go's regexp is RE2: no look-around and no backreferences.
    if not isinstance(triggers, list) or not triggers:
        report.error(where, "triggers must be a non-empty list of patterns")
        return
    for pattern in triggers:
        if not isinstance(pattern, str) or re.search(r"\(\?<?[=!]|\\[1-9]", pattern):
            report.error(where, f"trigger is not valid RE2: {pattern!r}")
            continue
        try:
            re.compile(pattern)
        except re.error as exc:
            report.error(where, f"trigger does not compile: {pattern!r}: {exc}")


def check_profile_fields(where: str, item: dict, report: Report) -> None:
    if "triggers" in item:
        check_triggers(where, item["triggers"], report)
    if "profile_group" in item and item["profile_group"] not in PROFILE_GROUPS:
        report.error(where, f"profile_group must be one of {sorted(PROFILE_GROUPS)}")
    for entry in item.get("stated_in") or []:
        if not isinstance(entry, dict) or not str(entry.get("label", "")).strip():
            report.error(where, f"stated_in entries need kind and label: {entry}")
            continue
        if entry.get("kind") not in vocab("kinds"):
            report.error(where, f"stated_in kind is not an intake kind: {entry.get('kind')}")
        if "discipline" in entry and entry["discipline"] not in vocab("disciplines"):
            report.error(where, f"stated_in discipline is not an intake discipline: {entry['discipline']}")


def check_sources(where: str, sources, seed_dir: Path, report: Report, heading_cache: dict) -> None:
    if not isinstance(sources, list) or not sources:
        report.error(where, "sources must be a non-empty list")
        return
    for src in sources:
        if isinstance(src, dict) and "document" in src:
            if DOCUMENT_IDS is None:
                report.error(where, "document source needs data/eval/profile/manifest.json")
            elif src["document"] not in DOCUMENT_IDS:
                report.error(where, f"document not in the profile manifest: {src['document']}")
            if not str(src.get("anchor", "")).strip():
                report.error(where, "document source needs an anchor line")
            continue
        if isinstance(src, dict) and "dataset" in src:
            # Owner-supplied, AI-generated list: cite the dataset and a row id.
            rows = DATASETS.get(src["dataset"])
            if rows is None:
                report.error(where, f"dataset not in data/unforeseen/manifest.json: {src['dataset']}")
            elif src.get("row") not in rows:
                report.error(where, f"row not in dataset {src['dataset']}: {src.get('row')}")
            continue
        if isinstance(src, dict) and "design" in src:
            # A design or plan document in this repo. Plans can be untracked in a
            # worktree, so a missing file warns; a present file must hold the anchor.
            path = ROOT / str(src["design"])
            if not path.is_file():
                report.warn(where, f"design file not found: {src['design']}")
            elif str(src.get("anchor", "")).rstrip() not in {
                    line.rstrip() for line in path.read_text(encoding="utf-8").splitlines()}:
                report.error(where, f"anchor not found in {src['design']}: {src.get('anchor')!r}")
            continue
        if isinstance(src, dict) and "clerk_file" in src:
            # Historical provenance key; resolves inside the local archived source tree.
            if not (seed_dir.parent.parent / src["clerk_file"]).is_file():
                report.error(where, f"clerk file not found: {src['clerk_file']}")
            continue
        if not isinstance(src, dict) or not {"seed", "anchor"} <= src.keys():
            report.error(where, f"source needs seed and anchor: {src}")
            continue
        seed_path = seed_dir / src["seed"]
        if not seed_path.is_file():
            report.error(where, f"seed file not found: {src['seed']}")
            continue
        if seed_path not in heading_cache:
            lines = seed_path.read_text(encoding="utf-8").splitlines()
            heading_cache[seed_path] = {line.rstrip() for line in lines if line.startswith("#")}
        if src["anchor"].rstrip() not in heading_cache[seed_path]:
            report.error(where, f"anchor not found in {src['seed']}: {src['anchor']!r}")


def check_question(where: str, q, report: Report, refs: list, runs_on_required: bool = True) -> None:
    if not isinstance(q, dict):
        report.error(where, "question must be a mapping")
        return
    qtype = q.get("type")
    if qtype not in QUESTION_TYPES:
        report.error(where, f"question type must be one of {sorted(QUESTION_TYPES)}")
    if not str(q.get("instructions", "")).strip():
        report.error(where, "question needs instructions")
    criteria = q.get("criteria")
    if qtype == "noul" and criteria is not None:
        # Existing files use bare booleans; works files quote the keys ("true").
        if not isinstance(criteria, dict) or {str(k).lower() for k in criteria} != {"true", "false"} or len(criteria) != 2:
            report.error(where, "noul criteria must have exactly `true` and `false`")
    if qtype == "choice" and criteria is not None and not isinstance(criteria, dict):
        report.error(where, "choice criteria must be a mapping")
    if qtype == "score" and not (isinstance(criteria, list) and 2 <= len(criteria) <= 10):
        report.error(where, "score criteria must be a list of 2-10 levels")
    runs_on = q.get("runs_on")
    if runs_on is None and not runs_on_required:
        return
    if not isinstance(runs_on, list) or not runs_on:
        report.error(where, "question needs a non-empty runs_on list")
    else:
        refs.extend(("system", where, s) for s in runs_on)


def check_works_predicate(where: str, val, report: Report, refs: list) -> None:
    """`works`: an in-scope work item with one of these actions on one of these systems."""
    if not isinstance(val, dict) or not val or not val.keys() <= {"action", "system", "layout_change"}:
        report.error(where, f"`works` needs action and/or system lists: {val}")
        return
    if "layout_change" in val and val["layout_change"] not in ("yes", "no"):
        report.error(where, "works.layout_change must be quoted yes or no")
    for field in ("action", "system"):
        if field in val and (not isinstance(val[field], list) or not val[field]):
            report.error(where, f"works.{field} must be a non-empty list")
    for action in val.get("action") or []:
        if action not in ACTIONS:
            report.error(where, f"unknown action in works predicate: {action}")
    refs.extend(("system", where, s) for s in val.get("system") or [])


def check_predicate(where: str, pred, report: Report, refs: list, program: dict | None = None) -> None:
    """`program` is the program library context; only there are `cond` and
    `work_action` valid, and `det: work_type` literals checked."""
    if not isinstance(pred, dict):
        report.error(where, f"predicate must be a mapping: {pred}")
        return
    for key, val in pred.items():
        if key in ("all", "any"):
            if not isinstance(val, list):
                report.error(where, f"`{key}` must be a list")
                continue
            for sub in val:
                check_predicate(where, sub, report, refs, program)
        elif key == "not":
            check_predicate(where, val, report, refs, program)
        elif key == "system_present":
            refs.append(("system", where, val))
        elif key == "system_existing":
            # On the site and not being replaced; system_present is the completed building.
            refs.append(("system", where, val))
        elif key == "works":
            check_works_predicate(where, val, report, refs)
        elif key == "det":
            refs.append(("determinant", where, val))
            ops = set(pred) - {"det"}
            if len(ops) != 1 or not ops <= PREDICATE_OPS:
                report.error(where, f"determinant condition needs exactly one of {sorted(PREDICATE_OPS)}: {pred}")
            elif program is not None and val == "work_type":
                # Code feeds work_type from the Profile's work types.
                check_literals(where, pred, "det", program["work_types"], "work type", report)
        elif key == "cond" and program is not None:
            options = program["conditions"].get(val)
            ops = set(pred) - {"cond"}
            if options is None:
                report.error(where, f"unknown Profile condition `{val}`")
            elif len(ops) != 1 or not ops <= {"any_of", "eq"}:
                report.error(where, f"condition needs exactly one of any_of or eq: {pred}")
            else:
                check_literals(where, pred, "cond", options, f"option of {val}", report)
        elif key == "work_action" and program is not None:
            check_work_action(where, val, report, refs)
        elif key in PREDICATE_OPS:
            if "det" not in pred and not (program is not None and "cond" in pred):
                report.error(where, f"`{key}` without `det`: {pred}")
            if key == "is" and not isinstance(val, bool):
                report.error(where, "`is` requires a boolean; use any_of for choice values")
        else:
            report.error(where, f"unknown predicate key `{key}`")


def check_literals(where: str, pred: dict, operand: str, allowed: set, what: str, report: Report) -> None:
    op = next(k for k in pred if k != operand)
    values = pred[op] if op == "any_of" else [pred[op]]
    if op not in ("any_of", "eq") or not isinstance(values, list) or not values:
        report.error(where, f"{what} compares with any_of (a non-empty list) or eq: {pred}")
        return
    for v in values:
        if str(v) not in allowed:
            report.error(where, f"{v!r} is not a valid {what}")


def check_work_action(where: str, val, report: Report, refs: list) -> None:
    """`work_action`: an included work item on the system (or below it) with one
    of the actions. retain and investigate never select a fragment."""
    if not isinstance(val, dict) or set(val) != {"system", "any_of"} or not isinstance(val.get("system"), str):
        report.error(where, f"`work_action` needs exactly system (one id) and any_of: {val}")
        return
    refs.append(("system", where, val["system"]))
    actions = val["any_of"]
    if not isinstance(actions, list) or not actions:
        report.error(where, "work_action.any_of must be a non-empty list")
        return
    for action in actions:
        if action not in FRAGMENT_ACTIONS or action not in ACTIONS:
            report.error(where, f"work_action action {action!r} must be one of {sorted(FRAGMENT_ACTIONS)}")


def check_item(kind: str, rel: str, item, seed_dir: Path, report: Report, refs: list, cache: dict) -> None:
    if not isinstance(item, dict):
        report.error(rel, f"{kind} entry must be a mapping")
        return
    where = f"{rel} [{item.get('id', '?')}]"
    missing = REQUIRED[kind] - item.keys()
    if missing:
        report.error(where, f"missing fields: {sorted(missing)}")
    if "id" in item and not ID_PATTERNS[kind].match(str(item["id"])):
        report.error(where, f"id does not match {ID_PATTERNS[kind].pattern}")
    if item.get("status") not in STATUSES:
        report.error(where, f"status must be one of {sorted(STATUSES)}")
    if item.get("status") == "deprecated":
        if not item.get("replaced_by"):
            report.error(where, "deprecated record needs replaced_by")
        else:
            refs.append(({"systems": "system", "determinants": "determinant", "rules": "rule",
                          "interfaces": "interface", "failure_modes": "failure_mode"}[kind], where, item["replaced_by"]))
    check_sources(where, item.get("sources"), seed_dir, report, cache)
    if "applies_when" in item:
        check_predicate(where, item["applies_when"], report, refs)

    if kind == "systems":
        parent = item.get("parent")
        is_top = rel == "knowledge/systems.yaml"
        if is_top and parent:
            report.error(where, "top-level systems have no parent")
        if not is_top:
            if not parent:
                report.error(where, "child systems need a parent")
            elif not str(item.get("id", "")).startswith(f"{parent}."):
                report.error(where, f"id must start with parent `{parent}.`")
            else:
                refs.append(("system", where, parent))
    elif kind == "determinants":
        check_profile_fields(where, item, report)
        if "systems" in item:
            if not isinstance(item["systems"], list) or not item["systems"]:
                report.error(where, "determinant systems must be a non-empty list")
            else:
                refs.extend(("system", where, s) for s in item["systems"])
        if item.get("value") not in VALUE_TYPES:
            report.error(where, f"value must be one of {sorted(VALUE_TYPES)}")
        if item.get("derived"):
            refs.append(("rule", where, item.get("by")))
        elif "question" not in item:
            report.error(where, "extracted determinant needs a question")
        elif item.get("value") == "boolean":
            q = item["question"]
            if (not isinstance(q, dict) or q.get("type") != "choice"
                    or set(q.get("criteria") or {}) != {"stated_true", "stated_false", "not_stated"}):
                report.error(where, "boolean evidence needs explicit true, false and not_stated choices")
        if "question" in item:
            # A triggered question is routed by code patterns, not system labels.
            check_question(where, item["question"], report, refs, runs_on_required="triggers" not in item)
    elif kind == "rules":
        for s in item.get("systems") or []:
            refs.append(("system", where, s))
        derives = item.get("derives")
        if derives is not None:
            if not isinstance(derives, dict) or not {"table", "inputs", "gives"} <= derives.keys():
                report.error(where, "derives needs table, inputs and gives")
            else:
                if "pending" in derives and not isinstance(derives["pending"], bool):
                    report.error(where, "derivation pending must be a boolean")
                for d in derives["inputs"] + [derives["gives"]]:
                    refs.append(("determinant", where, d))
                if derives.get("pending") is True:
                    if not str(derives.get("reason", "")).strip():
                        report.error(where, "pending derivation needs a reason")
                else:
                    refs.append(("table", where, derives["table"]))
        if not isinstance(item.get("clause_verified"), bool):
            report.error(where, "clause_verified must be a boolean")
        if item.get("clause_verified") is True and not item.get("primary_source"):
            report.error(where, "verified clause needs primary_source")
        if "evidence" in item:
            check_question(where, item["evidence"], report, refs)
        for r in item.get("related_rules") or []:
            refs.append(("rule", where, r))
        for n in item.get("numbers") or []:
            if not isinstance(n, dict) or "claim" not in n or "verified" not in n:
                report.error(where, f"numbers entries need claim and verified: {n}")
            elif not isinstance(n["verified"], bool):
                report.error(where, "number verified must be a boolean")
            elif n["verified"] and not n.get("primary_source"):
                report.error(where, "verified number needs primary_source")
    elif kind == "interfaces":
        if item.get("type") not in INTERFACE_TYPES:
            report.error(where, f"type must be one of {sorted(INTERFACE_TYPES)}")
        for side in ("from", "to"):
            vals = item.get(side)
            if not isinstance(vals, list) or not vals:
                report.error(where, f"`{side}` must be a non-empty list of systems")
            else:
                refs.extend(("system", where, s) for s in vals)
        if item.get("type") == "sequences" and "hold_point" not in item:
            report.warn(where, "sequences interface without hold_point")
        hold = item.get("hold_point")
        release = hold.get("release") if isinstance(hold, dict) else None
        if release is not None:
            # Optional Jev noul for hold-point release evidence (program plan 10.4).
            check_question(where, release, report, refs)
            if not isinstance(release, dict) or release.get("type") != "noul":
                report.error(where, "hold_point.release must be a noul")
            elif release.get("status") not in STATUSES:
                report.error(where, f"hold_point.release needs a status in {sorted(STATUSES)}")
        for q in item.get("resolved_when") or []:
            check_question(where, q, report, refs)
        for r in item.get("governed_by") or []:
            refs.append(("rule", where, r))
        for f in item.get("failure_modes") or []:
            refs.append(("failure_mode", where, f))
    elif kind == "failure_modes":
        if item.get("severity") not in SEVERITIES:
            report.error(where, f"severity must be one of {sorted(SEVERITIES)}")
        on = item.get("attaches_to")
        if not isinstance(on, dict) or len(on) != 1 or not on.keys() <= {"interface", "rule", "system"}:
            report.error(where, "`attaches_to` must be exactly one of interface, rule or system")
        else:
            (target_kind, target), = on.items()
            refs.append(({"interface": "interface", "rule": "rule", "system": "system"}[target_kind], where, target))
        detector = item.get("detector")
        check_question(where, detector, report, refs)
        if isinstance(detector, dict) and detector.get("type") != "noul":
            report.error(where, "detector must be a noul")
        if not isinstance(item.get("flag_when"), bool):
            report.error(where, "flag_when must be true or false")


def check_tables(report: Report, refs: list) -> dict:
    known = {}
    required = {"version", "id", "status", "verified", "instrument", "clause",
                "primary_source", "checked_on", "inputs", "outputs", "scope", "exclusions", "rows"}
    for path in sorted((KNOWLEDGE / "tables").glob("*.yaml")):
        rel = path.relative_to(ROOT).as_posix()
        try:
            table = yaml.safe_load(path.read_text(encoding="utf-8"))
        except yaml.YAMLError as exc:
            report.error(rel, f"invalid YAML: {exc}")
            continue
        if not isinstance(table, dict) or required - table.keys():
            report.error(rel, "table needs provenance, scope, inputs, outputs and rows")
            continue
        if table["version"] != 1 or table["verified"] is not True:
            report.error(rel, "only version 1 primary-verified tables may be published")
        if table["status"] not in STATUSES:
            report.error(rel, "invalid table status")
        if table["id"] != path.stem or not ID_PATTERNS["determinants"].fullmatch(str(table["id"])):
            report.error(rel, "table id must match its snake_case filename")
        if not str(table["primary_source"]).startswith("https://"):
            report.error(rel, "table needs an HTTPS primary source")
        for field in ("inputs", "outputs", "exclusions", "rows"):
            if not isinstance(table[field], list) or not table[field]:
                report.error(rel, f"table {field} must be a non-empty list")
        if isinstance(table["inputs"], list):
            refs.extend(("determinant", rel, d) for d in table["inputs"])
        if isinstance(table["rows"], list) and isinstance(table["outputs"], list):
            for row in table["rows"]:
                if not isinstance(row, dict) or not set(table["outputs"]) <= row.keys():
                    report.error(rel, "every table row must define each output")
        known[table["id"]] = rel
    return known


def check_deprecated_refs(refs: list, deprecated: dict, report: Report) -> None:
    for ref_kind, where, target in refs:
        if target in deprecated.get(ref_kind, {}):
            report.error(where, f"refers to deprecated {ref_kind} {target}; use {deprecated[ref_kind][target]}")


def check_replacements(deprecated: dict, known: dict, report: Report) -> None:
    for kind, mapping in deprecated.items():
        for old, new in mapping.items():
            if new in mapping:
                report.error(known[kind].get(old, old), f"{old} is replaced by {new}, which is itself deprecated")


def check_scope_defaults(rel: str, doc: dict, taxonomy: dict, systems: dict, deprecated: dict,
                         determinants: dict, report: Report) -> None:
    categories = {c.get("id") for c in taxonomy.get("building_classes") or []}
    classes = {s.get("id") for c in taxonomy.get("building_classes") or [] for s in c.get("subclasses") or []}
    work_types = {w.get("id") for w in taxonomy.get("work_types") or []}

    def leaves(where: str, ids) -> None:
        if not isinstance(ids, list) or not ids:
            report.error(where, "systems must be a non-empty list of leaf ids")
            return
        for sid in ids:
            if sid not in systems:
                report.error(where, f"unknown system {sid}")
            elif "." not in sid:
                report.error(where, f"`{sid}` is not a leaf system")
            elif sid in deprecated:
                report.error(where, f"{sid} is deprecated")

    for w in doc.get("empty_work_types") or []:
        if w not in work_types:
            report.error(rel, f"unknown work type {w}")
    for d in doc.get("always_shown") or []:
        if d not in determinants or not determinants[d].get("profile_group"):
            report.error(rel, f"always_shown must name a profile determinant: {d}")
    seen = set()
    for preset in doc.get("presets") or []:
        where = f"{rel} [preset {preset.get('id')}]"
        if not preset.get("id") or not str(preset.get("label", "")).strip():
            report.error(where, "preset needs id and label")
        if preset.get("id") in seen:
            report.error(where, "duplicate preset id")
        seen.add(preset.get("id"))
        leaves(where, preset.get("systems"))
    for entry in doc.get("classes") or []:
        where = f"{rel} [{entry.get('class')} x {entry.get('work_type')}]"
        if entry.get("class") not in classes:
            report.error(where, f"unknown building class {entry.get('class')}")
        if entry.get("work_type") not in work_types:
            report.error(where, f"unknown work type {entry.get('work_type')}")
        leaves(where, entry.get("systems"))
    for entry in doc.get("categories") or []:
        where = f"{rel} [{entry.get('category')}]"
        if entry.get("category") not in categories:
            report.error(where, f"unknown building category {entry.get('category')}")
        leaves(where, entry.get("systems"))


def predicate_determinants(pred) -> set:
    """Determinant ids a predicate reads."""
    out = set()
    if isinstance(pred, dict):
        if "det" in pred:
            out.add(pred["det"])
        for val in pred.values():
            out |= predicate_determinants(val)
    elif isinstance(pred, list):
        for val in pred:
            out |= predicate_determinants(val)
    return out


def check_relevance(rel: str, doc: dict, determinants: dict, rules: list, report: Report) -> None:
    """A profile determinant must be able to become relevant: a rule reads it,
    it names its own systems, or it is always shown. Otherwise the scope-led
    profile can never show it (docs/design/2026-10-03-profile-scope.md)."""
    read = set()
    for _, rule in rules:
        read |= predicate_determinants(rule.get("applies_when"))
        derives = rule.get("derives")
        if isinstance(derives, dict):
            read |= set(derives.get("inputs") or []) | {derives.get("gives")}
    always = set(doc.get("always_shown") or [])
    for did, item in sorted(determinants.items()):
        if not item.get("profile_group") or item.get("status") == "deprecated":
            continue
        if did not in read and did not in always and not item.get("systems"):
            report.warn(rel, f"profile determinant {did} is read by no rule and has no systems; the scoped profile never shows it")


def check_taxonomy(rel: str, doc: dict, report: Report) -> None:
    for cls in doc.get("building_classes") or []:
        where = f"{rel} [{cls.get('id')}]"
        if not cls.get("id") or not cls.get("label") or not cls.get("subclasses"):
            report.error(where, "building class needs id, label and subclasses")
        for sub in cls.get("subclasses") or []:
            if not sub.get("id") or not sub.get("label"):
                report.error(where, f"subclass needs id and label: {sub}")
            for field in sub.get("scale_fields") or []:
                if not {"key", "label", "type"} <= field.keys():
                    report.error(where, f"scale field needs key, label and type: {field}")
                if "triggers" in field:
                    check_triggers(f"{where} {field.get('key')}", field["triggers"], report)
    for wt in doc.get("work_types") or []:
        if not wt.get("id") or not wt.get("label"):
            report.error(rel, f"work type needs id and label: {wt}")
    for cond in doc.get("conditions") or []:
        if not cond.get("key") or not cond.get("options"):
            report.error(rel, f"condition needs key and options: {cond}")
        for opt in cond.get("options") or []:
            if re.search(r"\(\+\d", str(opt.get("label", ""))):
                report.error(rel, f"condition option label carries cost-uplift text: {opt.get('label')}")


def check_profile(report: Report, seed_dir: Path, known: dict, deprecated: dict, refs: list, cache: dict,
                  determinants: dict | None = None, rules: list | None = None) -> None:
    folder = KNOWLEDGE / PROFILE_DIR
    if not folder.is_dir():
        return
    docs = {}
    for name in ("taxonomy", "project_facts", "scope_defaults"):
        path = folder / f"{name}.yaml"
        rel = path.relative_to(ROOT).as_posix()
        if not path.is_file():
            report.error(rel, "profile file is missing")
            continue
        doc = yaml.safe_load(path.read_text(encoding="utf-8"))
        if not isinstance(doc, dict) or doc.get("version") != 1:
            report.error(rel, "must be a mapping with version: 1")
            continue
        docs[name] = (rel, doc)
    if "taxonomy" in docs:
        rel, doc = docs["taxonomy"]
        if doc.get("status") not in STATUSES:
            report.error(rel, "taxonomy needs a status")
        check_sources(rel, doc.get("sources"), seed_dir, report, cache)
        check_taxonomy(rel, doc, report)
    if "project_facts" in docs:
        rel, doc = docs["project_facts"]
        for item in doc.get("facts") or []:
            check_item("determinants", rel, item, seed_dir, report, refs, cache)
            if isinstance(item, dict) and item.get("id") in known["determinant"]:
                report.error(rel, f"project fact id collides with a determinant: {item['id']}")
    if "scope_defaults" in docs and "taxonomy" in docs:
        rel, doc = docs["scope_defaults"]
        if doc.get("status") not in STATUSES:
            report.error(rel, "scope defaults need a status")
        check_sources(rel, doc.get("sources"), seed_dir, report, cache)
        check_scope_defaults(rel, doc, docs["taxonomy"][1], known["system"], deprecated.get("system", {}),
                             determinants or {}, report)
        check_relevance(rel, doc, determinants or {}, rules or [], report)


def check_layout(report: Report, known: dict, determinants: dict) -> None:
    """profile/layout.yaml: screen order only, but a bad entry hides an input."""
    path = KNOWLEDGE / PROFILE_DIR / "layout.yaml"
    if not path.is_file():
        return
    rel = path.relative_to(ROOT).as_posix()
    doc = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(doc, dict) or doc.get("version") != 1:
        report.error(rel, "must be a mapping with version: 1")
        return
    if doc.get("status") not in STATUSES:
        report.error(rel, "layout needs a status")
    top = {sid for sid in known["system"] if "." not in sid}
    order = doc.get("build_order") or []
    if sorted(order) != sorted(top):
        report.error(rel, "build_order must list every top-level system once")
    skeleton = yaml.safe_load((KNOWLEDGE / "rfp" / "skeleton.yaml").read_text(encoding="utf-8")) or {}
    flags = {f.get("key") for f in skeleton.get("flags") or [] if isinstance(f, dict)}
    seen = set()
    for home, keys in (doc.get("homes") or {}).items():
        if home not in ("project", "planning") and home not in top:
            report.error(rel, f"home {home} is not project, planning or a top-level system")
        for key in keys or []:
            if key in seen:
                report.error(rel, f"{key} has two homes")
            seen.add(key)
            if key.startswith("det.") and key[4:] not in determinants:
                report.error(rel, f"{key}: no such determinant")
            elif key.startswith("proj.flag.") and key[10:] not in flags:
                report.error(rel, f"{key}: no such RFP flag")
            elif not key.startswith(("det.", "proj.flag.")):
                report.error(rel, f"{key}: only det.<id> and proj.flag.<key> are placed")


def check_derivation_output(where: str, rule: dict, determinants: dict, report: Report) -> None:
    derives = rule.get("derives")
    if not isinstance(derives, dict):
        return
    output = determinants.get(derives.get("gives"))
    if output is not None and (output.get("derived") is not True or output.get("by") != rule.get("id")):
        report.error(where, "derivation output must be a derived determinant owned by this rule, not an extracted fact")


class _UniqueKeyLoader(yaml.SafeLoader):
    """Safe loader that rejects a repeated mapping key, which would otherwise
    silently drop a coverage-ledger row."""

    def construct_mapping(self, node, deep=False):
        seen = set()
        for key_node, _ in node.value:
            key = self.construct_object(key_node, deep=True)
            if key in seen:
                raise yaml.constructor.ConstructorError(None, None, f"duplicate key {key!r}", key_node.start_mark)
            seen.add(key)
        return super().construct_mapping(node, deep)


def load_datasets(report: Report) -> dict:
    """Dataset id -> row ids, from data/unforeseen/manifest.json and each CSV."""
    path = UNFORESEEN_DIR / "manifest.json"
    if not path.is_file():
        return {}
    rel = path.relative_to(ROOT).as_posix()
    out = {}
    for entry in json.loads(path.read_text(encoding="utf-8")).get("datasets", []):
        did = entry.get("id")
        if not did or entry.get("ai_generated") is not True or entry.get("status") not in STATUSES \
                or not str(entry.get("origin", "")).strip():
            report.error(rel, f"dataset needs id, origin, status and ai_generated: true: {did}")
            continue
        csv_path = UNFORESEEN_DIR / str(entry.get("file", ""))
        if not csv_path.is_file():
            report.error(rel, f"dataset file not found: {entry.get('file')}")
            continue
        with csv_path.open(encoding="utf-8", newline="") as fh:
            ids = [row.get(entry.get("row_id_column", "row_id"), "") for row in csv.DictReader(fh)]
        bad = [i for i in ids if not ROW_ID.match(i)]
        if bad or len(set(ids)) != len(ids):
            report.error(rel, f"dataset {did} row ids must be unique and well-formed (e.g. B1-0001): {bad[:3]}")
        if entry.get("rows") != len(ids):
            report.error(rel, f"dataset {did} declares {entry.get('rows')} rows but the file has {len(ids)}")
        out[did] = set(ids)
    return out


def load_works_doc(path: Path, key: str, report: Report, version: int = 1):
    rel = path.relative_to(ROOT).as_posix()
    try:
        doc = yaml.safe_load(path.read_text(encoding="utf-8"))
    except yaml.YAMLError as exc:
        report.error(rel, f"invalid YAML: {exc}")
        return rel, None
    if not isinstance(doc, dict) or doc.get("version") != version:
        report.error(rel, f"must be a mapping with version: {version}")
        return rel, None
    if key and not isinstance(doc.get(key) or [], list):
        report.error(rel, f"{key} must be a list")
        return rel, None
    return rel, doc


def load_actions(report: Report) -> dict | None:
    """Fills ACTIONS before any predicate is checked."""
    path = KNOWLEDGE / WORKS_DIR / "actions.yaml"
    if not path.is_file():
        return None
    rel, doc = load_works_doc(path, "actions", report)
    if doc is None:
        return None
    for action in doc.get("actions") or []:
        where = f"{rel} [{action.get('id') if isinstance(action, dict) else '?'}]"
        if not isinstance(action, dict) or not str(action.get("id", "")).strip() \
                or not str(action.get("describes", "")).strip() or not str(action.get("excludes", "")).strip():
            report.error(where, "action needs id, describes and excludes")
            continue
        if action["id"] in ACTIONS:
            report.error(where, "duplicate action id")
        ACTIONS.add(action["id"])
    return doc


def check_actions(rel: str, doc: dict, seed_dir: Path, report: Report, cache: dict) -> None:
    for action in doc.get("actions") or []:
        if isinstance(action, dict) and type(action.get("needs_design")) is not bool:
            report.error(rel, f"action {action.get('id')} needs explicit boolean needs_design")
    if doc.get("status") not in STATUSES:
        report.error(rel, "actions file needs a status")
    check_sources(rel, doc.get("sources"), seed_dir, report, cache)
    if len(ACTIONS) != 8:
        report.warn(rel, f"expected the 8 actions of the plan, found {len(ACTIONS)}")
    for ans in ("several", "not_stated"):
        if not str((doc.get("answers") or {}).get(ans, "")).strip():
            report.error(rel, f"answers.{ans} is missing")
    taxonomy = KNOWLEDGE / PROFILE_DIR / "taxonomy.yaml"
    work_types = set()
    if taxonomy.is_file():
        work_types = {w.get("id") for w in (yaml.safe_load(taxonomy.read_text(encoding="utf-8")) or {}).get("work_types") or []}
    defaults = doc.get("work_type_defaults")
    if not isinstance(defaults, dict) or not defaults:
        report.error(rel, "work_type_defaults is missing")
        defaults = {}
    for wt, action in defaults.items():
        if work_types and wt not in work_types:
            report.error(rel, f"work_type_defaults: unknown work type {wt}")
        if action not in ACTIONS:
            report.error(rel, f"work_type_defaults: unknown action {action}")
    if work_types and work_types - defaults.keys():
        report.error(rel, f"work_type_defaults missing work types: {sorted(work_types - defaults.keys())}")
    cond = doc.get("existing_conditions") or {}
    values = cond.get("values")
    if not isinstance(values, list) or not values:
        report.error(rel, "existing_conditions.values is missing")
    else:
        ids = [v.get("id") if isinstance(v, dict) else None for v in values]
        if len(set(ids)) != len(ids) or not all(isinstance(i, str) and ID_PATTERNS["determinants"].match(i) for i in ids):
            report.error(rel, "existing_conditions ids must be unique snake_case")
    check_sources(rel, [cond.get("source")] if cond.get("source") else None, seed_dir, report, cache)


def check_proposal(where: str, prop, kinds: set, report: Report, field: str = "propose") -> None:
    if not isinstance(prop, dict) or prop.get("kind") not in kinds or not str(prop.get("label", "")).strip():
        report.error(where, f"{field} needs a kind from {sorted(kinds)} and a label: {prop}")
    if isinstance(prop, dict) and set(prop) - {"kind", "label"}:
        report.error(where, f"{field} has unknown fields {sorted(set(prop) - {'kind', 'label'}, key=str)}; quote labels containing commas in flow mappings")


def check_common(where: str, kind: str, item: dict, required: set, seed_dir: Path, report: Report, cache: dict) -> None:
    missing = required - item.keys()
    if missing:
        report.error(where, f"missing fields: {sorted(missing)}")
    if "id" in item and not WORKS_ID[kind].match(str(item["id"])):
        report.error(where, f"id does not match {WORKS_ID[kind].pattern}")
    if item.get("status") not in STATUSES:
        report.error(where, f"status must be one of {sorted(STATUSES)}")
    check_sources(where, item.get("sources"), seed_dir, report, cache)


def check_signal_refs(where: str, ids, signals: dict, report: Report) -> None:
    if not isinstance(ids, list):
        report.error(where, "signals must be a list of signal ids")
        return
    for sid in ids:
        if sid not in signals:
            report.error(where, f"unknown signal {sid}")


def check_interface_consequence(rel: str, item, seed_dir: Path, report: Report, cache: dict) -> None:
    where = f"{rel} [{item.get('id', '?') if isinstance(item, dict) else '?'}]"
    if not isinstance(item, dict):
        report.error(rel, "entry must be a mapping")
        return
    check_common(where, "ic", item, {"id", "type", "touches", "actions", "propose", "status", "sources"}, seed_dir, report, cache)
    if item.get("type") not in INTERFACE_TYPES:
        report.error(where, f"type must be one of {sorted(INTERFACE_TYPES)}")
    if item.get("touches") not in TOUCHES:
        report.error(where, f"touches must be one of {sorted(TOUCHES)}")
    actions = item.get("actions")
    if actions != "any":
        if not isinstance(actions, list) or not actions:
            report.error(where, "actions must be a list or `any`")
        else:
            for a in actions:
                if a not in ACTIONS:
                    report.error(where, f"unknown action {a}")
    check_proposal(where, item.get("propose"), INTERFACE_PROPOSAL_KINDS, report)


def check_consequence(rel: str, item, seed_dir: Path, report: Report, refs: list, cache: dict, signals: dict) -> None:
    where = f"{rel} [{item.get('id', '?') if isinstance(item, dict) else '?'}]"
    if not isinstance(item, dict):
        report.error(rel, "entry must be a mapping")
        return
    check_common(where, "cq", item, {"id", "when", "propose", "governed_by", "severity", "status", "sources"},
                 seed_dir, report, cache)
    if "when" in item:
        check_predicate(where, item["when"], report, refs)
    props = item.get("propose")
    if not isinstance(props, list) or not props:
        report.error(where, "propose must be a non-empty list")
    else:
        for prop in props:
            check_proposal(where, prop, PROPOSAL_KINDS, report)
    if item.get("severity") not in SEVERITIES:
        report.error(where, f"severity must be one of {sorted(SEVERITIES)}")
    if not isinstance(item.get("governed_by"), list):
        report.error(where, "governed_by must be a list of rule ids (may be empty)")
    else:
        refs.extend(("rule", where, r) for r in item["governed_by"])
    # Regulatory triggers stay unverified until the instrument is read.
    if "clause_verified" in item and not isinstance(item["clause_verified"], bool):
        report.error(where, "clause_verified must be a boolean")
    if item.get("clause_verified") is True and not item.get("primary_source"):
        report.error(where, "verified clause needs primary_source")
    if item.get("governed_by") and "clause_verified" not in item:
        report.error(where, "a consequence governed by a rule needs clause_verified")
    check_signal_refs(where, item.get("signals", []), signals, report)


def check_unforeseen(rel: str, item, seed_dir: Path, report: Report, refs: list, cache: dict, signals: dict) -> None:
    where = f"{rel} [{item.get('id', '?') if isinstance(item, dict) else '?'}]"
    if not isinstance(item, dict):
        report.error(rel, "entry must be a mapping")
        return
    check_common(where, "uc", item, {"id", "kind", "category", "attaches_to", "when", "signals", "de_risk", "contract",
                                     "effect", "severity", "discovered_at", "status", "sources"}, seed_dir, report, cache)
    if item.get("kind") not in UC_KINDS:
        report.error(where, f"kind must be one of {sorted(UC_KINDS)}")
    if item.get("category") not in UC_CATEGORIES:
        report.error(where, f"category must be one of {sorted(UC_CATEGORIES)}")
    on = item.get("attaches_to")
    if not isinstance(on, dict) or len(on) != 1 or not on.keys() <= {"system", "interface", "stage", "package_kind"}:
        report.error(where, "`attaches_to` must be exactly one of system, interface, stage or package_kind")
    else:
        (target_kind, target), = on.items()
        if target_kind in ("system", "interface"):
            refs.append((target_kind, where, target))
        elif target not in (UC_STAGES if target_kind == "stage" else PACKAGE_KINDS):
            report.error(where, f"unknown {target_kind}: {target}")
    if "when" in item:
        check_predicate(where, item["when"], report, refs)
    check_signal_refs(where, item.get("signals"), signals, report)
    check_proposal(where, item.get("de_risk"), PROPOSAL_KINDS, report, "de_risk")
    if not str(item.get("contract", "")).strip():
        report.error(where, "contract needs a plain statement")
    effect = item.get("effect")
    if not isinstance(effect, list) or not effect or not set(effect) <= UC_EFFECTS:
        report.error(where, f"effect must be a non-empty list from {sorted(UC_EFFECTS)}")
    if item.get("severity") not in SEVERITIES:
        report.error(where, f"severity must be one of {sorted(SEVERITIES)}")
    found = item.get("discovered_at")
    found = [found] if isinstance(found, str) else found
    if not isinstance(found, list) or not found or not set(found) <= DISCOVERED_AT:
        report.error(where, f"discovered_at must be one or more of {sorted(DISCOVERED_AT)}")


def check_ledger(path: Path, records: set, report: Report) -> int:
    """Every dataset row appears once with a disposition. Returns the pending count."""
    rel = path.relative_to(ROOT).as_posix()
    try:
        doc = yaml.load(path.read_text(encoding="utf-8"), Loader=_UniqueKeyLoader)
    except yaml.YAMLError as exc:
        report.error(rel, f"invalid YAML: {exc}")
        return 0
    if not isinstance(doc, dict) or doc.get("version") != 1 or not isinstance(doc.get("rows"), dict):
        report.error(rel, "ledger must be a mapping with version: 1 and a rows mapping")
        return 0
    dataset = doc.get("dataset")
    if dataset != path.stem or dataset not in DATASETS:
        report.error(rel, f"ledger dataset must match its filename and the manifest: {dataset}")
        return 0
    if doc.get("status") not in STATUSES:
        report.error(rel, "ledger needs a status")
    rows = doc["rows"]
    expected = DATASETS[dataset]
    missing = sorted(expected - rows.keys())
    for rid in missing[:5]:
        report.error(rel, f"dataset row missing from the ledger: {rid}")
    if len(missing) > 5:
        report.error(rel, f"... {len(missing)} dataset rows missing in all")
    for rid in sorted(rows.keys() - expected)[:5]:
        report.error(rel, f"ledger row is not in the dataset: {rid}")
    # Per-author overlays (coverage/<dataset>/<name>.yaml) decide rows that are
    # pending in the base ledger, so parallel authors never edit one file.
    overlay_dir = path.parent / dataset
    decided_in: dict = {}
    for overlay in sorted(overlay_dir.glob("*.yaml")) if overlay_dir.is_dir() else []:
        orel = overlay.relative_to(ROOT).as_posix()
        try:
            odoc = yaml.load(overlay.read_text(encoding="utf-8"), Loader=_UniqueKeyLoader)
        except yaml.YAMLError as exc:
            report.error(orel, f"invalid YAML: {exc}")
            continue
        if not isinstance(odoc, dict) or odoc.get("version") != 1 or not isinstance(odoc.get("rows"), dict):
            report.error(orel, "overlay must be a mapping with version: 1 and a rows mapping")
            continue
        for rid, disp in odoc["rows"].items():
            if rid not in expected:
                report.error(orel, f"overlay row is not in the dataset: {rid}")
            elif rows.get(rid) != "pending":
                report.error(orel, f"overlay row is already decided in the base ledger: {rid}")
            elif rid in decided_in:
                report.error(orel, f"overlay row also decided in {decided_in[rid]}: {rid}")
            elif disp == "pending":
                report.error(orel, f"overlay rows must be decided, not pending: {rid}")
            else:
                decided_in[rid] = orel
                rows = dict(rows)
                rows[rid] = disp
    pending = 0
    for rid, disp in rows.items():
        where = f"{decided_in.get(rid, rel)} [{rid}]"
        if disp == "pending":
            pending += 1
        elif isinstance(disp, dict) and set(disp) == {"records"} and isinstance(disp["records"], list) and disp["records"]:
            for rec in disp["records"]:
                if rec not in records:
                    report.error(where, f"unknown record {rec}")
        elif isinstance(disp, dict) and disp.get("rejected") in LEDGER_REJECTIONS:
            if disp["rejected"] == "duplicate_of":
                if disp.get("ref") not in expected and disp.get("ref") not in records:
                    report.error(where, f"duplicate_of needs a ref to a dataset row or record: {disp.get('ref')}")
                elif disp.get("ref") == rid:
                    report.error(where, "a row cannot duplicate itself")
            elif set(disp) != {"rejected"}:
                report.error(where, f"unexpected fields: {sorted(set(disp) - {'rejected'})}")
        else:
            report.error(where, f"disposition must be pending, a records list or a rejection ({sorted(LEDGER_REJECTIONS)}): {disp}")
    return pending


def check_works(report: Report, seed_dir: Path, refs: list, cache: dict, actions_doc: dict | None, known: dict) -> int:
    """Validates knowledge/works/ and the per-cluster consequences and unforeseen
    files. Returns the number of pending ledger rows."""
    folder = KNOWLEDGE / WORKS_DIR
    signals: dict = {}
    signal_files = [folder / "signals.yaml"] + sorted((KNOWLEDGE / "clusters").glob("*/signals.yaml"))
    for path in signal_files:
        if not path.is_file():
            continue
        rel, doc = load_works_doc(path, "signals", report)
        for sig in (doc or {}).get("signals") or []:
            where = f"{rel} [{sig.get('id', '?') if isinstance(sig, dict) else '?'}]"
            if not isinstance(sig, dict):
                report.error(rel, "signal must be a mapping")
                continue
            check_common(where, "sig", sig, {"id", "type", "instructions", "criteria", "runs_on", "status", "sources"},
                         seed_dir, report, cache)
            check_question(where, sig, report, refs)
            if sig.get("type") != "noul":
                report.error(where, "a signal must be a noul")
            if sig.get("id") in signals:
                report.error(where, f"duplicate signal id (also in {signals[sig.get('id')]})")
            signals[sig.get("id")] = rel
    known["signal"] = signals
    if actions_doc is not None:
        check_actions("knowledge/works/actions.yaml", actions_doc, seed_dir, report, cache)
    path = folder / "interface_consequences.yaml"
    if path.is_file():
        rel, doc = load_works_doc(path, "interface_consequences", report)
        for item in (doc or {}).get("interface_consequences") or []:
            check_interface_consequence(rel, item, seed_dir, report, cache)
            if isinstance(item, dict) and "id" in item:
                if item["id"] in known["interface_consequence"]:
                    report.error(rel, f"duplicate id {item['id']}")
                known["interface_consequence"][item["id"]] = rel
    for cluster in sorted((KNOWLEDGE / "clusters").iterdir()):
        for name, key in CLUSTER_WORKS_FILES.items():
            if key == "signals":
                continue
            path = cluster / name
            if not path.is_file():
                continue
            rel, doc = load_works_doc(path, key, report)
            for item in (doc or {}).get(key) or []:
                if key == "consequences":
                    check_consequence(rel, item, seed_dir, report, refs, cache, signals)
                    bucket = known["consequence"]
                else:
                    check_unforeseen(rel, item, seed_dir, report, refs, cache, signals)
                    bucket = known["unforeseen"]
                if isinstance(item, dict) and "id" in item:
                    if item["id"] in bucket:
                        report.error(rel, f"duplicate id {item['id']} (also in {bucket[item['id']]})")
                    bucket[item["id"]] = rel
    # A ledger row may be covered by any record kind it was folded into.
    records = (set(known["consequence"]) | set(known["unforeseen"])
               | set(known["failure_mode"]) | set(known["interface"]))
    pending = 0
    ledger_dir = folder / "coverage"
    ledger_files = sorted(ledger_dir.glob("*.yaml")) if ledger_dir.is_dir() else []
    for dataset in sorted(DATASETS.keys() - {p.stem for p in ledger_files}):
        report.error("knowledge/works/coverage", f"dataset {dataset} has no coverage ledger")
    for ledger in ledger_files:
        pending += check_ledger(ledger, records, report)
    return pending



# Delivery and commercial catalogues (SCHEMA.md, 2026-10-05). Each file is
# optional; a present file must match its documented shape.
CATALOGUE_DIRS = ("reports", "costs")
STAGE_OTHER = "other"
PLANNING_VALUES = {"integer", "number", "boolean", "choice", "text"}
PLANNING_KEY = re.compile(r"^[a-z][a-z0-9_]*$")


def planning_value_ok(key: dict, value) -> bool:
    """A favourable or starting value, as text, fits the key's type."""
    v = str(value)
    kind = key.get("value")
    if kind == "integer":
        return re.fullmatch(r"-?[0-9]+", v) is not None
    if kind == "number":
        return re.fullmatch(r"-?[0-9]+(\.[0-9]+)?", v) is not None
    if kind == "boolean":
        return v in ("true", "false")
    if kind == "choice":
        return v in {o.get("id") for o in key.get("options") or [] if isinstance(o, dict)}
    return True
SCOPES = {"site", "project"}
NOVATION = {"pre", "post"}
REPORT_OUTPUTS = {"rfp", "rft", "pmp"}
CLAUSE_ID = re.compile(r"^cl\.[a-z0-9-]+$")
BENCHMARK_ID = re.compile(r"^bm\.[a-z0-9-]+$")
DECIMAL = re.compile(r"^-?\d+(\.\d+)?$")


def load_catalogue(path: Path, key: str, report: Report, version: int = 1):
    if not path.is_file():
        return None, None
    rel, doc = load_works_doc(path, key, report, version)
    return rel, doc


def check_catalogues(report: Report, seed_dir: Path, cache: dict, determinants: dict) -> None:
    taxonomy = {}
    tax_path = KNOWLEDGE / PROFILE_DIR / "taxonomy.yaml"
    if tax_path.is_file():
        taxonomy = yaml.safe_load(tax_path.read_text(encoding="utf-8")) or {}
    classes = {c.get("id") for c in taxonomy.get("building_classes") or []}
    work_types = {w.get("id") for w in taxonomy.get("work_types") or []}
    conditions = {c.get("key"): c for c in taxonomy.get("conditions") or []}

    rel, doc = load_catalogue(KNOWLEDGE / WORKS_DIR / "stages.yaml", "stages", report)
    if doc is not None:
        if doc.get("status") not in STATUSES:
            report.error(rel, "stages file needs a status")
        check_sources(rel, doc.get("sources"), seed_dir, report, cache)
        ids = [s.get("id") for s in doc.get("stages") or [] if isinstance(s, dict)]
        # `other` is the one extra: where a user-made Program stage maps.
        if set(ids) - {STAGE_OTHER} != UC_STAGES or len(ids) != len(set(ids)):
            report.error(rel, f"top-level stages must be exactly {sorted(UC_STAGES)}, plus optional {STAGE_OTHER}")
        seen = set(ids)
        for stage in doc.get("stages") or []:
            if not isinstance(stage, dict) or not str(stage.get("label", "")).strip():
                report.error(rel, f"stage needs id and label: {stage}")
                continue
            for sub in stage.get("substages") or []:
                where = f"{rel} [{stage.get('id')}]"
                if not isinstance(sub, dict) or not ID_PATTERNS["determinants"].match(str(sub.get("id", ""))) \
                        or not str(sub.get("label", "")).strip():
                    report.error(where, f"sub-stage needs a snake_case id and a label: {sub}")
                    continue
                if sub.get("novation") is not None and sub.get("novation") not in NOVATION:
                    report.error(where, f"novation must be one of {sorted(NOVATION)}")
                if sub["id"] in seen:
                    report.error(where, f"duplicate stage id {sub['id']}")
                seen.add(sub["id"])

    rel, doc = load_catalogue(KNOWLEDGE / WORKS_DIR / "package_defaults.yaml", "baselines", report, version=2)
    if doc is not None:
        if doc.get("status") not in STATUSES:
            report.error(rel, "package defaults need a status")
        check_sources(rel, [doc.get("source")] if doc.get("source") else None, seed_dir, report, cache)
        for i, b in enumerate(doc.get("baselines") or []):
            where = f"{rel} [baselines {i}]"
            if not isinstance(b, dict) or not b.get("consultants"):
                report.error(where, "baseline needs consultants")
                continue
            for c in (b.get("building_classes") or []) + (b.get("except_building_classes") or []):
                if classes and c not in classes:
                    report.error(where, f"unknown building class {c}")
            for w in b.get("work_types") or []:
                if work_types and w not in work_types:
                    report.error(where, f"unknown work type {w}")
        for i, a in enumerate(doc.get("scope_additions") or []):
            where = f"{rel} [scope_additions {i}]"
            if not isinstance(a, dict) or not isinstance(a.get("systems"), list) or not a["systems"]                     or any(not isinstance(v, str) or not v.strip() for v in a["systems"]) or not a.get("consultants"):
                report.error(where, "scope addition needs systems and consultants")
        for i, a in enumerate(doc.get("complexity_additions") or []):
            where = f"{rel} [complexity_additions {i}]"
            if not isinstance(a, dict) or not isinstance(a.get("field"), str) or not a["field"].strip() \
                    or not isinstance(a.get("values"), list) or not a["values"] \
                    or any(not isinstance(v, str) or not v.strip() for v in a["values"]) \
                    or not a.get("consultants"):
                report.error(where, "addition needs field, values and consultants")
                continue
            if a["field"] not in determinants and a["field"] not in conditions:
                report.warn(where, f"field {a['field']} is not a SiteWise determinant or condition; map it before use")
                continue
            field = determinants.get(a["field"]) or conditions[a["field"]]
            options = {o["id"] for o in field.get("options") or [] if isinstance(o, dict) and "id" in o}
            if field.get("value") == "boolean":
                options = {"true", "false"}
            if options:
                for value in a["values"]:
                    if value not in options:
                        report.warn(where, f"value {value!r} is not an option of {a['field']}; map it before use")

    rel, doc = load_catalogue(KNOWLEDGE / PROFILE_DIR / "planning_keys.yaml", "keys", report)
    if doc is not None:
        seen = set()
        for k in doc.get("keys") or []:
            where = f"{rel} [{k.get('key', '?') if isinstance(k, dict) else '?'}]"
            if not isinstance(k, dict) or not str(k.get("key", "")).strip() or not str(k.get("label", "")).strip():
                report.error(where, "planning key needs key and label")
                continue
            if k["key"].startswith("cost") or k.get("value") not in PLANNING_VALUES:
                report.error(where, f"value must be one of {sorted(PLANNING_VALUES)}; money totals belong to the cost plan")
            elif not PLANNING_KEY.match(k["key"]):
                report.error(where, f"key does not match {PLANNING_KEY.pattern}; the profile shows it as plan.<key>")
            if k.get("value") == "choice" and not k.get("options"):
                report.error(where, "a choice key needs options")
            for v in list(k.get("favourable") or []) + ([k["starting_value"]] if k.get("starting_value") else []):
                if not planning_value_ok(k, v):
                    report.error(where, f"{v!r} is not a {k.get('value')} value of this key")
            if k.get("starting_value") and k["starting_value"] in (k.get("favourable") or []):
                report.error(where, "a starting value may not be favourable (L276)")
            if k.get("scope") not in SCOPES:
                report.error(where, f"scope must be one of {sorted(SCOPES)}")
            if k["key"] in seen:
                report.error(where, "duplicate key")
            seen.add(k["key"])

    rel, doc = load_catalogue(KNOWLEDGE / PROFILE_DIR / "project_details.yaml", "details", report)
    if doc is not None:
        seen = set()
        for d in doc.get("details") or []:
            where = f"{rel} [{d.get('key', '?') if isinstance(d, dict) else '?'}]"
            if not isinstance(d, dict) or not str(d.get("key", "")).strip() or not str(d.get("label", "")).strip():
                report.error(where, "project detail needs key and label")
                continue
            if not PLANNING_KEY.match(d["key"]) or d["key"] == "flag":
                report.error(where, f"key does not match {PLANNING_KEY.pattern} or is 'flag'; the profile shows it as proj.<key>")
            if d.get("value") not in ("text", "yes_no"):
                report.error(where, "value must be text or yes_no")
            if d.get("scope") not in SCOPES:
                report.error(where, f"scope must be one of {sorted(SCOPES)}")
            check_detail_reading(where, d, report)
            if d["key"] in seen:
                report.error(where, "duplicate key")
            seen.add(d["key"])
        seen = set()
        for d in doc.get("site_details") or []:
            where = f"{rel} [site {d.get('key', '?') if isinstance(d, dict) else '?'}]"
            if not isinstance(d, dict) or not PLANNING_KEY.match(str(d.get("key", ""))) or not str(d.get("label", "")).strip():
                report.error(where, "site detail needs a key matching " + PLANNING_KEY.pattern + " and a label")
                continue
            if d.get("value") != "text":
                report.error(where, "a site detail is text")
            check_detail_reading(where, d, report)
            if d["key"] in seen:
                report.error(where, "duplicate key")
            seen.add(d["key"])

    # Flags a document can settle: each must be a skeleton flag that has no
    # determinant, so a flag is never read twice and left to disagree.
    skeleton = load_catalogue(KNOWLEDGE / "rfp" / "skeleton.yaml", "flags", report)[1]
    rel, doc = load_catalogue(KNOWLEDGE / PROFILE_DIR / "flag_readings.yaml", "flags", report)
    if doc is not None and skeleton is not None:
        skeleton_flags = {f.get("key") for f in skeleton.get("flags") or [] if isinstance(f, dict)}
        has_home = {"bushfire", "flood", "contamination", "acid_sulfate", "existing_building", "heritage", "live_operations", "mixed_use"}
        seen = set()
        for f in doc.get("flags") or []:
            where = f"{rel} [{f.get('key', '?') if isinstance(f, dict) else '?'}]"
            if not isinstance(f, dict) or f.get("key") not in skeleton_flags:
                report.error(where, "not a flag of rfp/skeleton.yaml")
                continue
            if f["key"] in has_home:
                report.error(where, "this flag already has a determinant or derived home in the profile")
            if not str(f.get("asks", "")).strip():
                report.error(where, "a flag reading needs the question it asks")
            check_triggers(where, f.get("triggers"), report)
            if f["key"] in seen:
                report.error(where, "duplicate key")
            seen.add(f["key"])

    rel, doc = load_catalogue(KNOWLEDGE / PROFILE_DIR / "key_scope.yaml", "families", report)
    if doc is not None:
        patterns = set()
        for f in doc.get("families") or []:
            if not isinstance(f, dict) or not str(f.get("match", "")).strip() or f.get("scope") not in SCOPES:
                report.error(rel, f"family needs a match and a scope in {sorted(SCOPES)}: {f}")
                continue
            if f["match"] in patterns:
                report.error(rel, f"duplicate match {f['match']}")
            patterns.add(f["match"])

    rel, doc = load_catalogue(KNOWLEDGE / "reports" / "clauses.yaml", "clauses", report)
    clauses_by_id = {}
    if doc is not None:
        seen = set()
        for c in doc.get("clauses") or []:
            where = f"{rel} [{c.get('id', '?') if isinstance(c, dict) else '?'}]"
            if not isinstance(c, dict):
                report.error(rel, "clause must be a mapping")
                continue
            if not CLAUSE_ID.match(str(c.get("id", ""))):
                report.error(where, f"id does not match {CLAUSE_ID.pattern}")
            if not isinstance(c.get("version"), int) or c.get("version") < 1:
                report.error(where, "version must be a positive integer")
            if not c.get("outputs") or set(c.get("outputs")) - REPORT_OUTPUTS:
                report.error(where, f"outputs must be a non-empty subset of {sorted(REPORT_OUTPUTS)}")
            if not str(c.get("text", "")).strip() or not str(c.get("section", "")).strip():
                report.error(where, "clause needs section and text")
            if c.get("status") not in STATUSES:
                report.error(where, f"status must be one of {sorted(STATUSES)}")
            check_sources(where, c.get("sources"), seed_dir, report, cache)
            if c.get("id") in seen:
                report.error(where, "duplicate clause id")
            seen.add(c.get("id"))
            clauses_by_id[c.get("id")] = c

    rel, doc = load_catalogue(KNOWLEDGE / "reports" / "templates.yaml", "templates", report)
    if doc is not None:
        seen = set()
        for template in doc.get("templates") or []:
            where = f"{rel} [{template.get('id', '?') if isinstance(template, dict) else '?'}]"
            if not isinstance(template, dict):
                report.error(rel, "template must be a mapping")
                continue
            template_id = str(template.get("id", ""))
            if not re.fullmatch(r"tpl\.[a-z0-9-]+", template_id) or template_id in seen:
                report.error(where, "invalid or duplicate template id")
            seen.add(template_id)
            if type(template.get("version")) is not int or template["version"] < 1:
                report.error(where, "template version must be a positive integer")
            if template.get("kind") not in REPORT_OUTPUTS or template.get("status") not in STATUSES:
                report.error(where, "invalid template kind or status")
            check_sources(where, template.get("sources"), seed_dir, report, cache)
            sections = template.get("sections")
            if not isinstance(sections, list) or not sections:
                report.error(where, "template needs non-empty sections")
                continue
            section_ids = set()
            for section in sections:
                if not isinstance(section, dict):
                    report.error(where, "section must be a mapping")
                    continue
                section_id = str(section.get("id", ""))
                if not re.fullmatch(r"[a-z][a-z0-9_]*", section_id) or section_id in section_ids:
                    report.error(where, "invalid or duplicate section id")
                section_ids.add(section_id)
                if not str(section.get("title", "")).strip() or type(section.get("essential")) is not bool:
                    report.error(where, "section requires title and essential boolean")
                refs = section.get("clauses")
                if not isinstance(refs, list):
                    report.error(where, "section clauses must be a list")
                    continue
                used = set()
                for ref in refs:
                    if not isinstance(ref, dict):
                        report.error(where, "clause reference must be a mapping")
                        continue
                    ref_id = str(ref.get("id", ""))
                    clause = clauses_by_id.get(ref_id)
                    if (clause is None or ref_id in used or type(ref.get("version")) is not int
                            or ref.get("version") != clause.get("version")
                            or clause.get("section") != section_id
                            or template.get("kind") not in (clause.get("outputs") or [])):
                        report.error(where, f"incompatible clause reference {ref_id}")
                    used.add(ref_id)

    rel, doc = load_catalogue(KNOWLEDGE / "costs" / "benchmarks.yaml", "benchmarks", report)
    if doc is not None:
        seen = set()
        required = {"id", "version", "basis", "amount", "currency", "tax_basis", "price_date", "geography",
                    "quality", "inclusions", "exclusions", "status", "sources"}
        for b in doc.get("benchmarks") or []:
            where = f"{rel} [{b.get('id', '?') if isinstance(b, dict) else '?'}]"
            if not isinstance(b, dict):
                report.error(rel, "benchmark must be a mapping")
                continue
            missing = required - b.keys()
            if missing:
                report.error(where, f"missing fields: {sorted(missing)}")
            if not BENCHMARK_ID.match(str(b.get("id", ""))):
                report.error(where, f"id does not match {BENCHMARK_ID.pattern}")
            if b.get("basis") not in ("lump_sum", "rate") or (b.get("basis") == "rate" and not b.get("unit")):
                report.error(where, "basis must be lump_sum or rate (rate needs a unit)")
            if not isinstance(b.get("amount"), str) or not DECIMAL.match(b.get("amount", "")):
                report.error(where, "amount must be a decimal string, never a float")
            if b.get("tax_basis") not in ("ex_tax", "inc_tax"):
                report.error(where, "tax_basis must be ex_tax or inc_tax")
            if b.get("status") not in STATUSES:
                report.error(where, f"status must be one of {sorted(STATUSES)}")
            check_sources(where, b.get("sources"), seed_dir, report, cache)
            if b.get("id") in seen:
                report.error(where, "duplicate benchmark id")
            seen.add(b.get("id"))


def check_cost_structure(report: Report, seed_dir: Path, cache: dict) -> None:
    systems = {}
    for _, kind, items in load_files(Report()):
        if kind == "systems":
            systems.update({i["id"]: i for i in items if isinstance(i, dict) and "id" in i})
    path = KNOWLEDGE / "costs" / "elements.yaml"
    rel, doc = load_catalogue(path, "elements", report)
    if doc is None:
        return
    for name, key in (("elements", "elements"), ("starter_lines", "lines")):
        rel, data = load_catalogue(KNOWLEDGE / "costs" / (name + ".yaml"), key, report)
        if data is None:
            report.error(str(path), "cost structure needs both files")
            continue
        if data.get("status") not in {"draft", "reviewed"}:
            report.error(rel, "cost structure needs draft or reviewed status")
        check_sources(rel, data.get("sources"), seed_dir, report, cache)
        seen, codes = set(), set()
        for row in data.get(key) or []:
            if not isinstance(row, dict):
                report.error(rel, "cost structure row must be a mapping")
                continue
            id = row.get("system" if name == "elements" else "id")
            if not id or id in seen:
                report.error(rel, f"missing or duplicate cost identity {id}")
            seen.add(id)
            if name == "elements":
                if id not in systems or systems[id].get("parent"):
                    report.error(rel, f"not a top-level system: {id}")
                code = row.get("code")
                if not isinstance(code, str) or not code or code in codes:
                    report.error(rel, f"missing, non-string or duplicate code: {code}")
                codes.add(code)
                for trade in [row.get("trade")] + [o.get("trade") for o in row.get("overrides") or []]:
                    if trade not in vocab("disciplines") or not str(trade).startswith("trade."):
                        report.error(rel, f"unknown cost trade {trade}")
                for section in ("items", "overrides"):
                    subs = set()
                    for child in row.get(section) or []:
                        sub = child.get("sub")
                        if sub not in systems or systems[sub].get("parent") != id or sub in subs:
                            report.error(rel, f"invalid or duplicate cost subsystem {sub}")
                        subs.add(sub)
                        if section == "items" and (not isinstance(child.get("costed"), bool) or (child.get("costed") is False and not child.get("why"))):
                            report.error(rel, f"subsystem needs explicit costed and reason when omitted: {sub}")
            else:
                if not re.fullmatch(r"[a-z][a-z0-9-]*", str(id)) or not row.get("label") or row.get("category") not in {"contingency", "escalation", "authority_fees", "exclusions", "other"}:
                    report.error(rel, f"invalid cost starter {id}")
                pred = row.get("applies_when")
                if not isinstance(pred, bool):
                    registry = yaml.safe_load((KNOWLEDGE / "profile" / "taxonomy.yaml").read_text(encoding="utf-8"))
                    fields = {"hdr.cond." + r["key"] for r in registry.get("conditions", [])}
                    if not isinstance(pred, dict) or set(pred) != {"field", "in"} or pred.get("field") not in fields or not isinstance(pred.get("in"), list) or not pred["in"] or any(not isinstance(v, str) for v in pred["in"]):
                        report.error(rel, f"invalid cost starter predicate {id}")
        if name == "elements" and seen != {id for id, sys in systems.items() if not sys.get("parent")}:
            report.error(rel, "cost elements must cover every top-level system exactly once")


def link_cycle(nodes, links) -> bool:
    """Kahn's algorithm: a node never peeled off sits on a cycle."""
    indeg = {n: 0 for n in nodes}
    out: dict = {}
    for a, b in links:
        out.setdefault(a, []).append(b)
        indeg[b] = indeg.get(b, 0) + 1
        indeg.setdefault(a, 0)
    queue = [n for n, d in indeg.items() if d == 0]
    done = 0
    while queue:
        n = queue.pop()
        done += 1
        for m in out.get(n, []):
            indeg[m] -= 1
            if indeg[m] == 0:
                queue.append(m)
    return done != len(indeg)


def canonical_predicate(pred) -> str:
    """Order inside all, any and any_of does not change a predicate."""
    def norm(v):
        if isinstance(v, dict):
            return {str(k): norm(x) for k, x in v.items()}
        if isinstance(v, list):
            return sorted((norm(x) for x in v), key=lambda x: json.dumps(x, sort_keys=True, default=str))
        return v
    return json.dumps(norm(pred), sort_keys=True, default=str)


def check_program_links(where: str, nodes: list, links, durations: set, report: Report) -> None:
    if not isinstance(links, list):
        report.error(where, "links must be a list")
        return
    pairs = []
    for link in links:
        if not isinstance(link, dict) or set(link) - {"from", "to", "type", "lag"}:
            report.error(where, f"link needs from, to, type and lag: {link}")
            continue
        if link.get("from") not in nodes or link.get("to") not in nodes or link.get("from") == link.get("to"):
            report.error(where, f"link must join two different listed ids: {link}")
            continue
        if link.get("type") not in LINK_TYPES:
            report.error(where, f"link type must be one of {sorted(LINK_TYPES)}")
        lag = link.get("lag")
        if lag is not None and lag not in durations:
            report.error(where, f"lag must be null or a duration id in durations.yaml, never a number: {lag!r}")
        pairs.append((link["from"], link["to"]))
    if link_cycle(nodes, pairs):
        report.error(where, "links form a cycle")


def program_date(where: str, value, year: int, report: Report):
    # Quoted, so YAML 1.1 and Go's YAML 1.2 read the same string.
    if not isinstance(value, str) or not ISO_DATE.match(value):
        report.error(where, f"date must be a quoted YYYY-MM-DD string: {value!r}")
        return None
    try:
        day = datetime.date.fromisoformat(value)
    except ValueError:
        report.error(where, f"not a calendar date: {value}")
        return None
    if year is not None and day.year != year:
        report.error(where, f"date {value} is outside {year}")
    return day


def check_program_duration(where: str, d: dict, planning: set, context: dict, report: Report, refs: list) -> None:
    for field in ("id", "version", "basis", "unit_days", "geography", "limitations", "status", "sources"):
        if field not in d:
            report.error(where, f"missing field {field}")
    if not isinstance(d.get("version"), int) or isinstance(d.get("version"), bool) or d.get("version", 0) < 1:
        report.error(where, "version must be a positive integer")
    basis = d.get("basis")
    if basis not in DURATION_BASES:
        report.error(where, f"basis must be one of {sorted(DURATION_BASES)}")
    if d.get("unit_days") not in ("working", "calendar"):
        report.error(where, "unit_days must be working or calendar")
    if not str(d.get("geography") or "").strip() or not isinstance(d.get("limitations"), list):
        report.error(where, "needs geography text and a limitations list")
    driver = d.get("driver")
    if (basis in ("per_unit", "floor_cycle")) != (driver is not None):
        report.error(where, "driver is required for per_unit and floor_cycle and omitted otherwise")
    if driver is not None:
        if not isinstance(driver, dict) or set(driver) - {"planning_key", "determinant", "unit"} \
                or ("planning_key" in driver) == ("determinant" in driver) or not str(driver.get("unit") or "").strip():
            report.error(where, f"driver needs exactly one of planning_key or determinant, and a unit: {driver}")
        elif "planning_key" in driver and driver["planning_key"] not in planning:
            report.error(where, f"unknown planning key {driver['planning_key']}")
        elif "determinant" in driver:
            refs.append(("determinant", where, driver["determinant"]))
    cost_fields = {"k", "b", "cost_base_date", "index_basis"} & d.keys()
    if basis == "time_cost":
        if {"low", "typical", "high"} & d.keys():
            report.error(where, "time_cost rows carry k and b, not low, typical and high")
        for field in ("k", "b"):
            if not isinstance(d.get(field), str) or not PARAM_DECIMAL.match(d[field]):
                report.error(where, f"{field} must be a non-negative decimal string with at most 4 places")
        program_date(where, d.get("cost_base_date"), None, report)
        if not str(d.get("index_basis") or "").strip():
            report.error(where, "time_cost needs index_basis")
    else:
        if cost_fields:
            report.error(where, f"{sorted(cost_fields)} belong to time_cost rows")
        values = []
        for field in ("low", "typical", "high"):
            v = d.get(field)
            if not isinstance(v, str) or not DAY_DECIMAL.match(v):
                report.error(where, f"{field} must be a non-negative decimal string with at most 2 places, never a float")
            else:
                values.append(Decimal(v))
        if len(values) == 3 and not values[0] <= values[1] <= values[2]:
            report.error(where, "needs low <= typical <= high")
    if basis == "statutory":
        if not str(d.get("instrument") or "").strip() or not str(d.get("clause") or "").strip() \
                or not isinstance(d.get("verified"), bool):
            report.error(where, "statutory rows carry instrument, clause and verified")
        elif d["verified"] and not d.get("primary_source"):
            report.error(where, "a verified statutory row needs primary_source")
    if d.get("applies_when") is not None:
        check_predicate(where, d["applies_when"], report, refs, context)


def check_program(report: Report, seed_dir: Path, cache: dict, refs: list) -> None:
    """The program library: plan section 4 and checker rules 4.7."""
    directory = KNOWLEDGE / PROGRAM_DIR
    if not directory.is_dir():
        return
    taxonomy = yaml.safe_load((KNOWLEDGE / PROFILE_DIR / "taxonomy.yaml").read_text(encoding="utf-8")) or {}
    context = {
        "work_types": {str(w.get("id")) for w in taxonomy.get("work_types") or []},
        "conditions": {c.get("key"): {str(o.get("id")) for o in (c.get("options") or []) + (c.get("user_options") or [])}
                       for c in taxonomy.get("conditions") or []},
    }
    planning = {k.get("key") for k in (yaml.safe_load((KNOWLEDGE / PROFILE_DIR / "planning_keys.yaml")
                                                      .read_text(encoding="utf-8")) or {}).get("keys") or []}
    catalogue = (yaml.safe_load((KNOWLEDGE / WORKS_DIR / "stages.yaml").read_text(encoding="utf-8")) or {}).get("stages") or []
    stages, generic = set(), []
    for top in catalogue:
        stages.add(top.get("id"))
        subs = [s.get("id") for s in top.get("substages") or []]
        stages.update(subs)
        if top.get("id") != STAGE_OTHER:
            # Only a stage with several sub-stages is replaced by them; one
            # sub-stage is a fee item inside its stage (GenericStages).
            generic.extend(subs if len(subs) > 1 else [top.get("id")])

    rows: dict = {}
    for name, key in PROGRAM_FILES.items():
        path = directory / name
        rel = path.relative_to(ROOT).as_posix()
        if not path.is_file():
            report.error(rel, "missing program library file")
            rows[key] = (rel, [])
            continue
        rel, doc = load_works_doc(path, key, report)
        if doc is None:
            rows[key] = (rel, [])
            continue
        if doc.get("status") not in PROGRAM_STATUSES:
            report.error(rel, f"status must be one of {sorted(PROGRAM_STATUSES)}")
        check_sources(rel, doc.get("sources"), seed_dir, report, cache)
        items = []
        seen = set()
        for item in doc.get(key) or []:
            if not isinstance(item, dict):
                report.error(rel, f"{key} entry must be a mapping")
                continue
            where = f"{rel} [{item.get('id', '?')}]"
            if not PROGRAM_ID[key].match(str(item.get("id", ""))):
                report.error(where, f"id does not match {PROGRAM_ID[key].pattern}")
            if item.get("id") in seen:
                report.error(where, "duplicate id")
            seen.add(item.get("id"))
            unknown = set(item) - PROGRAM_KEYS[key]
            if unknown:
                report.error(where, f"unknown fields {sorted(unknown, key=str)}")
            if item.get("status") not in PROGRAM_STATUSES:
                report.error(where, f"status must be one of {sorted(PROGRAM_STATUSES)}")
            check_sources(where, item.get("sources"), seed_dir, report, cache)
            items.append((where, item))
        rows[key] = (rel, items)

    durations = {item.get("id") for _, item in rows["durations"][1]}
    fragments = {item.get("id") for _, item in rows["fragments"][1]}

    for where, d in rows["durations"][1]:
        check_program_duration(where, d, planning, context, report, refs)

    attach_links = []
    for where, f in rows["fragments"][1]:
        if not str(f.get("label") or "").strip():
            report.error(where, "fragment needs a label")
        if f.get("stage") not in stages or f.get("stage") == STAGE_OTHER:
            report.error(where, f"stage must be a catalogue stage other than {STAGE_OTHER}: {f.get('stage')}")
        if not isinstance(f.get("systems"), list) or not f["systems"]:
            report.error(where, "fragment needs systems")
        else:
            refs.extend(("system", where, s) for s in f["systems"])
        if f.get("applies_when") is not None:
            check_predicate(where, f["applies_when"], report, refs, context)
        keys = []
        activities = f.get("activities")
        if not isinstance(activities, list) or not activities:
            report.error(where, "fragment needs activities")
            activities = []
        for a in activities:
            if not isinstance(a, dict) or set(a) - {"key", "label", "systems", "duration"} \
                    or not ACTIVITY_KEY.match(str(a.get("key", ""))) or not str(a.get("label") or "").strip():
                report.error(where, f"activity needs key, label, systems and duration: {a}")
                continue
            if a["key"] in keys:
                report.error(where, f"duplicate activity key {a['key']}")
            keys.append(a["key"])
            if not isinstance(a.get("systems"), list) or not a["systems"]:
                report.error(where, f"activity {a['key']} needs systems")
            else:
                refs.extend(("system", where, s) for s in a["systems"])
            if a.get("duration") is not None and a["duration"] not in durations:
                report.error(where, f"activity {a['key']} names unknown duration {a['duration']}")
        check_program_links(where, keys, f.get("links") or [], durations, report)
        attach = f.get("attach")
        if attach is not None:
            if not isinstance(attach, dict) or not attach or set(attach) - {"after", "before"}:
                report.error(where, f"attach needs after and/or before: {attach}")
                continue
            for side, end in attach.items():
                if end in ATTACH_ENDS:
                    continue
                if end not in fragments or end == f.get("id"):
                    report.error(where, f"attach.{side} must be stage_start, stage_finish or another fragment: {end}")
                    continue
                attach_links.append((end, f["id"]) if side == "after" else (f["id"], end))
    if link_cycle(sorted(fragments, key=str), attach_links):
        report.error(rows["fragments"][0], "fragment attach links form a cycle")

    generic_count = 0
    predicates: dict = {}
    for where, sk in rows["skeletons"][1]:
        if not str(sk.get("label") or "").strip():
            report.error(where, "skeleton needs a label")
        listed = sk.get("stages")
        if not isinstance(listed, list) or not listed or len(set(map(str, listed))) != len(listed):
            report.error(where, "stages must be a non-empty list without repeats")
            listed = []
        for stage in listed:
            if stage not in stages or stage == STAGE_OTHER:
                report.error(where, f"stage must be a catalogue stage other than {STAGE_OTHER}: {stage}")
        check_program_links(where, listed, sk.get("links") or [], durations, report)
        if sk.get("id") == GENERIC_SKELETON:
            generic_count += 1
            if "applies_when" in sk:
                report.error(where, "sk.generic has no applies_when")
            if listed != generic:
                report.error(where, f"sk.generic stages must be the catalogue order {generic}")
            chain = [{"from": a, "to": b, "type": "FS", "lag": None} for a, b in zip(generic, generic[1:])]
            if sk.get("links") != chain:
                report.error(where, "sk.generic links must chain its stages finish-to-start with no lag")
            continue
        if sk.get("applies_when") is None:
            report.error(where, "only sk.generic may omit applies_when")
            continue
        check_predicate(where, sk["applies_when"], report, refs, context)
        canon = canonical_predicate(sk["applies_when"])
        if canon in predicates:
            report.error(where, f"identical predicate to {predicates[canon]}; generation could not tell them apart")
        predicates[canon] = sk.get("id")
    if generic_count != 1:
        report.error(rows["skeletons"][0], f"needs exactly one {GENERIC_SKELETON}, found {generic_count}")

    covered = set()
    for where, cal in rows["calendars"][1]:
        year = cal.get("year")
        if cal.get("state") not in CALENDAR_STATES or not isinstance(year, int) or isinstance(year, bool):
            report.error(where, f"needs a state in {sorted(CALENDAR_STATES)} and an integer year")
            year = None
        if (cal.get("state"), year) in covered:
            report.error(where, "another calendar covers this state and year")
        covered.add((cal.get("state"), year))
        week = cal.get("working_week")
        if not isinstance(week, list) or not week or set(week) - WEEKDAYS or len(set(week)) != len(week):
            report.error(where, f"working_week must list distinct days from {sorted(WEEKDAYS)}")
        if not isinstance(cal.get("verified"), bool):
            report.error(where, "calendar needs verified true or false")
        for day in cal.get("days") or []:
            if not isinstance(day, dict) or set(day) - {"date", "from", "to", "kind", "label"}:
                report.error(where, f"day needs date (or from and to), kind and label: {day}")
                continue
            # A bare `on` or `no` would arrive here as a boolean, never a kind.
            if day.get("kind") not in CALENDAR_KINDS or not str(day.get("label") or "").strip():
                report.error(where, f"day needs a kind in {sorted(CALENDAR_KINDS)} and a label: {day}")
            if ("date" in day) == ("from" in day or "to" in day):
                report.error(where, f"day needs either date or from and to: {day}")
            elif "date" in day:
                program_date(where, day["date"], year, report)
            else:
                start = program_date(where, day.get("from"), year, report)
                end = program_date(where, day.get("to"), year, report)
                if start and end and start > end:
                    report.error(where, f"from {day['from']} is after to {day['to']}")


def check_source_archive(archive: Path, report: Report) -> None:
    """Keep cited source bytes reproducible without the original repository."""
    where = "source archive"
    try:
        manifest = json.loads((archive / "manifest.json").read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        report.error(where, f"cannot read manifest: {exc}")
        return
    if (not isinstance(manifest, dict) or manifest.get("version") != 1
            or not isinstance(manifest.get("files"), list) or not manifest["files"]):
        report.error(where, "expected version 1 and a non-empty files list")
        return
    seen = set()
    for entry in manifest["files"]:
        if not isinstance(entry, dict):
            report.error(where, "file entry must be a mapping")
            continue
        name, digest = entry.get("path"), entry.get("sha256")
        if (not isinstance(name, str) or not isinstance(digest, str)
                or not re.fullmatch(r"[0-9a-f]{64}", digest)):
            report.error(where, "file entry needs path and SHA-256")
            continue
        path = (archive / name).resolve()
        if not path.is_relative_to(archive.resolve()) or name in seen:
            report.error(where, f"duplicate or out-of-archive path: {name}")
            continue
        seen.add(name)
        try:
            actual = hashlib.sha256(path.read_bytes()).hexdigest()
        except OSError:
            report.error(where, f"source file not found or unreadable: {name}")
            continue
        if actual != digest:
            report.error(where, f"source checksum mismatch: {name}")
    actual_files = {p.relative_to(archive).as_posix()
                    for p in (archive / "data").rglob("*") if p.is_file()}
    for name in sorted(actual_files - seen):
        report.error(where, f"source missing from manifest: {name}")


RFP_DIR = "rfp"
RFP_ANSWER_KEYS = ROOT / "data" / "eval" / "rfp" / "answer-keys"
RFP_THRESHOLD_KINDS = {"statutory", "client_default", "evidence_estimate"}
RFP_PROVENANCE_KINDS = {"evidence", "seed", "practice", "client_decision"}


def rfp_provenance(where: str, rows, report: Report, ledgers: dict) -> None:
    for row in rows or []:
        for p in row.get("provenance") or [] if isinstance(row, dict) else []:
            if not isinstance(p, dict) or p.get("kind") not in RFP_PROVENANCE_KINDS or not p.get("ref"):
                report.error(where, f"{row.get('key')}: provenance needs a kind and a ref")
            elif p.get("kind") == "client_decision" and not str(p.get("why") or "").strip():
                report.error(where, f"{row.get('key')}: a client_decision provenance needs a why")
            if not isinstance(p, dict):
                continue
            # Check numeric ledger citations without interpreting source-page or
            # seed-heading references as ledger row numbers.
            for discipline, refs in re.findall(r"([a-z][a-z-]+)#(\d+(?:[-–]\d+)?(?:,\d+(?:[-–]\d+)?)*)", str(p.get("ref", ""))):
                if discipline not in ledgers:
                    report.error(where, f"{row.get('key')}: unknown ledger {discipline}")
                    continue
                for ref in refs.split(","):
                    bounds = [int(n) for n in re.split("[-–]", ref)]
                    missing = set(range(bounds[0], bounds[-1] + 1)) - ledgers[discipline]
                    if bounds[-1] < bounds[0] or missing:
                        report.error(where, f"{row.get('key')}: invalid ledger reference {discipline}#{ref}")


def check_rfp(report: Report) -> None:
    """Consultant RFP discipline files (knowledge/rfp) and their selection
    answer keys. The Go loader enforces the condition vocabulary; this gate
    adds review records, threshold provenance, golden paths and answer-key
    references."""
    folder = KNOWLEDGE / RFP_DIR
    if not folder.is_dir():
        return
    ledgers = {
        path.parent.name: {int(n) for n in re.findall(r"^\|\s*(\d+)\s*\|", path.read_text(encoding="utf-8-sig"), re.M)}
        for path in (ROOT / "docs" / "rfp-golden").glob("*/ledger.md")
    }
    disciplines: dict = {}
    for path in sorted(folder.glob("*.yaml")):
        rel = path.relative_to(ROOT).as_posix()
        try:
            doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        except yaml.YAMLError as e:
            report.error(rel, f"invalid YAML: {e}")
            continue
        status = doc.get("status")
        if status not in {"draft", "reviewed"}:
            report.error(rel, "status must be draft or reviewed")
        if status == "reviewed" and not (doc.get("reviewed_by") and doc.get("reviewed_on")):
            report.error(rel, "reviewed needs reviewed_by and reviewed_on; only the owner marks a file reviewed")
        golden = doc.get("golden")
        if golden and not (ROOT / golden).is_file():
            report.error(rel, f"golden file not found: {golden}")
        for t in doc.get("thresholds") or []:
            if not isinstance(t, dict) or t.get("kind") not in RFP_THRESHOLD_KINDS or "verified" not in t:
                report.error(rel, f"threshold {t.get('key') if isinstance(t, dict) else t}: needs kind and verified")
            elif not str(t.get("source") or "").strip():
                report.error(rel, f"threshold {t.get('key')}: needs a source")
        for section in ("stages", "elements", "allowances", "positions", "scope_schedule", "role", "terms", "project_rows", "returnable_notes", "evaluation", "flags"):
            rfp_provenance(f"{rel} {section}", doc.get(section), report, ledgers)
        if path.name not in {"skeleton.yaml", "modules.yaml"} and doc.get("id"):
            disciplines[doc["id"]] = doc
    if not RFP_ANSWER_KEYS.is_dir():
        return
    for path in sorted(RFP_ANSWER_KEYS.glob("*.yaml")):
        rel = path.relative_to(ROOT).as_posix()
        key = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        if key.get("reviewed") and key.get("label_source") == "agent-draft":
            report.error(rel, "an agent draft cannot be marked reviewed")
        for discipline, expect in (key.get("expect") or {}).items():
            doc = disciplines.get(discipline)
            if doc is None:
                report.error(rel, f"unknown discipline {discipline}")
                continue
            names = {
                "stage": {s.get("key") for s in doc.get("stages") or []},
                "element": {e.get("key") for e in doc.get("elements") or []},
                "allowance": {a.get("key") for a in doc.get("allowances") or []},
                "position": {p.get("key") for p in doc.get("positions") or []},
            }
            refs = [("stage", k) for f in ("stages_include", "stages_option", "stages_absent", "stages_ask") for k in expect.get(f) or []]
            refs += [("element", k) for f in ("elements_include", "elements_option", "elements_absent") for k in expect.get(f) or []]
            refs += [("allowance", k) for f in ("allowances_present", "allowances_absent") for k in expect.get(f) or []]
            refs += [("position", k) for f in ("positions", "party") for k in (expect.get(f) or {})]
            for kind, name in refs:
                if name not in names[kind]:
                    report.error(rel, f"{discipline}: unknown {kind} {name}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed-dir", type=Path, default=DEFAULT_SEED_DIR)
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--only", default="", help="report only problems in paths starting with this prefix, e.g. knowledge/clusters/fire")
    args = parser.parse_args()
    if not args.seed_dir.is_dir():
        print(f"seed dir not found: {args.seed_dir}", file=sys.stderr)
        return 2

    global DOCUMENT_IDS, DATASETS
    DOCUMENT_IDS = load_document_ids()
    report = Report()
    if args.seed_dir.resolve() == DEFAULT_SEED_DIR.resolve():
        check_source_archive(SOURCE_ARCHIVE, report)
    DATASETS = load_datasets(report)
    actions_doc = load_actions(report)
    refs: list[tuple[str, str, str]] = []
    deprecated: dict = {}
    known = {k: {} for k in ("system", "determinant", "rule", "interface", "failure_mode",
                             "consequence", "unforeseen", "interface_consequence", "signal")}
    kind_key = {"systems": "system", "determinants": "determinant", "rules": "rule",
                "interfaces": "interface", "failure_modes": "failure_mode"}
    cache: dict = {}
    known["table"] = check_tables(report, refs)
    determinants = {}
    rules = []

    for path, kind, items in load_files(report):
        rel = path.relative_to(ROOT).as_posix()
        for item in items:
            check_item(kind, rel, item, args.seed_dir, report, refs, cache)
            if isinstance(item, dict) and "id" in item:
                if kind == "determinants":
                    determinants[item["id"]] = item
                elif kind == "rules":
                    rules.append((f"{rel} [{item['id']}]", item))
                if item.get("status") == "deprecated":
                    deprecated.setdefault(kind_key[kind], {})[item["id"]] = item.get("replaced_by")
                bucket = known[kind_key[kind]]
                if item["id"] in bucket:
                    report.error(rel, f"duplicate id {item['id']} (also in {bucket[item['id']]})")
                bucket[item["id"]] = rel

    for where, rule in rules:
        check_derivation_output(where, rule, determinants, report)
    check_profile(report, args.seed_dir, known, deprecated, refs, cache, determinants, rules)
    check_layout(report, known, determinants)
    pending = check_works(report, args.seed_dir, refs, cache, actions_doc, known)
    check_catalogues(report, args.seed_dir, cache, determinants)
    check_rfp(report)
    check_cost_structure(report, args.seed_dir, cache)
    check_program(report, args.seed_dir, cache, refs)
    check_replacements(deprecated, known, report)
    check_deprecated_refs(refs, deprecated, report)

    for ref_kind, where, target in refs:
        if target not in known[ref_kind]:
            msg = f"unresolved {ref_kind} reference: {target}"
            if ref_kind == "system" and str(target).split(".")[0] not in known["system"]:
                report.error(where, f"{msg} (unknown top-level system)")
            elif args.strict:
                report.error(where, msg)
            else:
                report.warn(where, msg)

    if args.only:
        report.errors = [e for e in report.errors if e.startswith(args.only)]
        report.warnings = [w for w in report.warnings if w.startswith(args.only)]
    for w in report.warnings:
        print(f"WARN  {w}")
    for e in report.errors:
        print(f"ERROR {e}")
    counts = ", ".join(f"{len(v)} {k.replace('_', ' ')}s" for k, v in known.items())
    rows = sum(len(r) for r in DATASETS.values())
    print(f"\n{counts}; {len(ACTIONS)} actions; {rows} dataset rows, {pending} pending in the coverage ledger")
    print(f"{len(report.errors)} errors, {len(report.warnings)} warnings")
    return 1 if report.errors else 0


if __name__ == "__main__":
    sys.exit(main())
