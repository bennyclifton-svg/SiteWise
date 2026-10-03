"""Validate knowledge/ against knowledge/SCHEMA.md.

Interim dev tool for the extraction phase; the Go knowledge loader replaces it.
Usage: python tools/check_knowledge.py [--seed-dir PATH] [--strict] [--only PATH_PREFIX]
Errors fail the run. Unresolved cross-references are warnings (another cluster
may define them) unless --strict.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
KNOWLEDGE = ROOT / "knowledge"
DEFAULT_SEED_DIR = ROOT.parent / "clerk" / "data" / "seed"

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
        if path.parent.name in ("tables", PROFILE_DIR):
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
        if isinstance(src, dict) and "clerk_file" in src:
            # Clerk data (taxonomy JSON) is copied as data; cite the file it came from.
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
        if not isinstance(criteria, dict) or set(criteria) != {True, False}:
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


def check_predicate(where: str, pred, report: Report, refs: list) -> None:
    if not isinstance(pred, dict):
        report.error(where, f"predicate must be a mapping: {pred}")
        return
    for key, val in pred.items():
        if key in ("all", "any"):
            if not isinstance(val, list):
                report.error(where, f"`{key}` must be a list")
                continue
            for sub in val:
                check_predicate(where, sub, report, refs)
        elif key == "not":
            check_predicate(where, val, report, refs)
        elif key == "system_present":
            refs.append(("system", where, val))
        elif key == "det":
            refs.append(("determinant", where, val))
            ops = set(pred) - {"det"}
            if len(ops) != 1 or not ops <= PREDICATE_OPS:
                report.error(where, f"determinant condition needs exactly one of {sorted(PREDICATE_OPS)}: {pred}")
        elif key in PREDICATE_OPS:
            if "det" not in pred:
                report.error(where, f"`{key}` without `det`: {pred}")
        else:
            report.error(where, f"unknown predicate key `{key}`")


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


def check_typical(rel: str, doc: dict, taxonomy: dict, systems: dict, deprecated: dict, report: Report) -> None:
    subclasses = {s.get("id") for c in taxonomy.get("building_classes") or [] for s in c.get("subclasses") or []}
    work_types = {w.get("id") for w in taxonomy.get("work_types") or []}
    for entry in doc.get("typical") or []:
        where = f"{rel} [{entry.get('subclass')} x {entry.get('work_type')}]"
        if entry.get("subclass") not in subclasses:
            report.error(where, f"unknown subclass {entry.get('subclass')}")
        if entry.get("work_type") not in work_types:
            report.error(where, f"unknown work type {entry.get('work_type')}")
        for sid in entry.get("systems") or []:
            if sid not in systems:
                report.error(where, f"unknown system {sid}")
            elif "." not in sid:
                report.error(where, f"`{sid}` is not a leaf system")
            elif sid in deprecated:
                report.error(where, f"{sid} is deprecated")


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


def check_profile(report: Report, seed_dir: Path, known: dict, deprecated: dict, refs: list, cache: dict) -> None:
    folder = KNOWLEDGE / PROFILE_DIR
    if not folder.is_dir():
        return
    docs = {}
    for name in ("taxonomy", "project_facts", "typical_systems"):
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
    if "typical_systems" in docs and "taxonomy" in docs:
        rel, doc = docs["typical_systems"]
        if doc.get("status") not in STATUSES:
            report.error(rel, "typical systems need a status")
        check_sources(rel, doc.get("sources"), seed_dir, report, cache)
        check_typical(rel, doc, docs["taxonomy"][1], known["system"], deprecated.get("system", {}), report)


def check_derivation_output(where: str, rule: dict, determinants: dict, report: Report) -> None:
    derives = rule.get("derives")
    if not isinstance(derives, dict):
        return
    output = determinants.get(derives.get("gives"))
    if output is not None and (output.get("derived") is not True or output.get("by") != rule.get("id")):
        report.error(where, "derivation output must be a derived determinant owned by this rule, not an extracted fact")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed-dir", type=Path, default=DEFAULT_SEED_DIR)
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--only", default="", help="report only problems in paths starting with this prefix, e.g. knowledge/clusters/fire")
    args = parser.parse_args()
    if not args.seed_dir.is_dir():
        print(f"seed dir not found: {args.seed_dir}", file=sys.stderr)
        return 2

    global DOCUMENT_IDS
    DOCUMENT_IDS = load_document_ids()
    report = Report()
    refs: list[tuple[str, str, str]] = []
    deprecated: dict = {}
    known = {k: {} for k in ("system", "determinant", "rule", "interface", "failure_mode")}
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
    check_profile(report, args.seed_dir, known, deprecated, refs, cache)
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
    counts = ", ".join(f"{len(v)} {k}s" for k, v in known.items())
    print(f"\n{counts}; {len(report.errors)} errors, {len(report.warnings)} warnings")
    return 1 if report.errors else 0


if __name__ == "__main__":
    sys.exit(main())
