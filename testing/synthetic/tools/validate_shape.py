"""Validate a project type's shape.yaml against the app's own vocabulary.

Every id an author writes (discipline, system, action, fragment, skeleton,
RFP template, profile key and value) must exist in the repo's knowledge or
data files, and every "built in" claim must cite a file that exists. This is
what keeps twenty separately written domain maps consistent with each other
and with the app.

Usage, from the repo root:

    python testing/synthetic/tools/validate_shape.py testing/synthetic/types/*/shape.yaml

Exit code 1 on any error. Warnings (thin content) do not fail.
"""
import json
import re
import sys
from pathlib import Path

import yaml

SCHEMA = "sitewise-synthetic-shape/1"
BUILT_IN = {"yes", "no", "partial"}
REQUIRED = ["schema", "type_id", "slug", "name", "reviewed", "state", "profile", "work_items",
            "procurement", "cost_plan", "program", "rfps", "triggers", "gaps", "variants"]
# Adopted after the T01 pilot reviews (29 October 2026). Optional, but checked when present.
OPTIONAL = ["not_expected_procurement", "not_expected_cost_plan", "expected_derived", "scenario_inputs"]
STAGE_LISTS = ["stages_include", "stages_option", "stages_absent"]
STATES = {"NSW", "VIC", "QLD", "SA", "WA", "TAS", "ACT", "NT"}
# Lower bounds that make a domain map worth reviewing. Thin maps are warned, not failed.
MIN = {"triggers": 8, "procurement": 2, "cost_plan": 5, "activities": 8, "rfps": 1}


def _yaml(path):
    return yaml.safe_load(Path(path).read_text(encoding="utf8"))


def _walk(node, key):
    """Every value of `key` in any dict under node."""
    if isinstance(node, dict):
        if key in node and isinstance(node[key], (str, int)):
            yield str(node[key])
        for value in node.values():
            yield from _walk(value, key)
    elif isinstance(node, list):
        for item in node:
            yield from _walk(item, key)


def load_ids(root):
    root = Path(root)
    k = root / "knowledge"
    tax = _yaml(k / "profile" / "taxonomy.yaml")
    classes, subclasses = set(), {}
    for c in tax["building_classes"]:
        classes.add(c["id"])
        subclasses[c["id"]] = {s["id"] for s in c.get("subclasses", [])}
    conditions = {}
    for c in tax["conditions"]:
        opts = {o["id"] for o in c.get("options", [])} | {o["id"] for o in c.get("user_options", [])}
        conditions[c["key"]] = opts
    systems = {s["id"] for s in _yaml(k / "systems.yaml")["systems"]}
    for f in (k / "clusters").glob("*/systems.yaml"):
        systems |= {s["id"] for s in (_yaml(f) or {}).get("systems", [])}
    rfps = set()
    for f in (k / "rfp").glob("*.yaml"):
        data = _yaml(f) or {}
        if data.get("discipline_ids"):
            rfps.add(data["id"])
    disciplines = {d["id"] for d in json.loads((root / "data" / "intake" / "disciplines.json").read_text(encoding="utf8"))["disciplines"]}
    return {
        "root": root,
        "classes": classes,
        "subclasses": subclasses,
        "work_types": {w["id"] for w in tax["work_types"]},
        "conditions": conditions,
        "determinants": {d["id"] for d in _yaml(k / "determinants.yaml")["determinants"]},
        "facts": set(_walk(_yaml(k / "profile" / "project_facts.yaml"), "id")),
        "details": set(_walk(_yaml(k / "profile" / "project_details.yaml"), "key")),
        "flags": set(_walk(_yaml(k / "profile" / "flag_readings.yaml"), "key")),
        "systems": systems,
        "actions": {a["id"] for a in _yaml(k / "works" / "actions.yaml")["actions"]},
        "fragments": {f["id"] for f in _yaml(k / "program" / "fragments.yaml")["fragments"]},
        "skeletons": {s["id"] for s in _yaml(k / "program" / "skeletons.yaml")["skeletons"]},
        "rfps": rfps,
        "disciplines": disciplines | {"none"},
    }


def _claim(value):
    """PyYAML reads a bare yes/no as a boolean; treat it as the word."""
    if value is True:
        return "yes"
    if value is False:
        return "no"
    return value


def _source_ok(source, root):
    """A source is a repo path, optionally followed by a space or # and an anchor, or 'domain'."""
    if source == "domain":
        return True
    path = re.split(r"[ #]", source.strip(), maxsplit=1)[0]
    return bool(path) and (root / path).exists()


def _profile_key_ok(key, ids):
    prefix, _, rest = key.partition(".")
    if prefix == "hdr":
        if key in ("hdr.building_class", "hdr.subclass", "hdr.work_type") or rest.startswith("scale."):
            return True
        return rest.startswith("cond.") and rest[5:] in ids["conditions"]
    if prefix == "det":
        return rest in ids["determinants"]
    if prefix == "fact":
        return rest in ids["facts"]
    if prefix in ("proj", "site"):
        return rest in ids["details"]
    if prefix == "flag":
        return rest in ids["flags"]
    return False


def validate(path, ids):
    path = Path(path)
    errors, warnings = [], []
    err = errors.append
    shape = _yaml(path) or {}
    root = ids["root"]

    for key in REQUIRED:
        if key not in shape:
            err(f"missing top-level key: {key}")
    for key in shape:
        if key not in REQUIRED and key not in OPTIONAL:
            err(f"unknown top-level key: {key} (add it to the schema, or remove it)")
    if any(e.startswith("missing") for e in errors):
        return errors, warnings
    if shape["schema"] != SCHEMA:
        err(f"schema must be {SCHEMA}")
    if not re.fullmatch(r"T\d\d", str(shape["type_id"])):
        err("type_id must look like T01")
    if path.parent.name != f"{shape['type_id']}-{shape['slug']}":
        err(f"folder {path.parent.name} must be {shape['type_id']}-{shape['slug']}")
    if shape["state"] not in STATES:
        err(f"state {shape['state']} is not an Australian state or territory")

    def built_in(where, item, source_key="source"):
        claim = _claim(item.get("built_in"))
        if claim not in BUILT_IN:
            err(f"{where}: built_in must be one of yes, no, partial")
        elif claim in ("yes", "partial"):
            src = item.get(source_key)
            if not src:
                err(f"{where}: built_in {claim} needs a source")
            elif not _source_ok(str(src), root):
                err(f"{where}: source does not exist: {src}")

    profile = shape["profile"]
    cls = profile.get("hdr.building_class")
    if cls not in ids["classes"]:
        err(f"profile hdr.building_class {cls} is not in taxonomy.yaml")
    sub = profile.get("hdr.subclass")
    if sub is not None and sub not in ids["subclasses"].get(cls, set()):
        err(f"profile hdr.subclass {sub} is not a subclass of {cls}")
    if profile.get("hdr.work_type") not in ids["work_types"]:
        err(f"profile hdr.work_type {profile.get('hdr.work_type')} is not in taxonomy.yaml")
    for key, value in profile.items():
        if not _profile_key_ok(key, ids):
            err(f"profile key {key} is not an app row key")
        cond = key[len("hdr.cond."):] if key.startswith("hdr.cond.") else None
        if cond in ids["conditions"] and value is not None:
            for one in value if isinstance(value, list) else [value]:
                if one not in ids["conditions"][cond]:
                    err(f"profile {key} value {one} is not an option")

    def work_item(where, item):
        if item.get("system") not in ids["systems"]:
            err(f"{where}: system {item.get('system')} is not in systems.yaml or a cluster")
        if item.get("action") not in ids["actions"]:
            err(f"{where}: action {item.get('action')} is not in actions.yaml")

    for i, item in enumerate(shape["work_items"]):
        work_item(f"work_items[{i}]", item)
    for i, item in enumerate(shape["procurement"]):
        if item.get("discipline") not in ids["disciplines"]:
            err(f"procurement[{i}]: discipline {item.get('discipline')} is not in disciplines.json")
        if item.get("kind") not in ("services", "works", "supply"):
            err(f"procurement[{i}]: kind must be services, works or supply")
        built_in(f"procurement[{i}]", item)
    for i, item in enumerate(shape["cost_plan"]):
        if not item.get("line"):
            err(f"cost_plan[{i}]: line is required")
        if item.get("section") not in ("fees_and_charges", "consultants", "construction", "contingency"):
            err(f"cost_plan[{i}]: section must be fees_and_charges, consultants, construction or contingency")
        if any(k in item for k in ("amount", "rate", "price", "budget")):
            err(f"cost_plan[{i}]: a shape names lines, never prices")
        built_in(f"cost_plan[{i}]", item)

    program = shape["program"] or {}
    skeleton = program.get("skeleton")
    if skeleton != "missing" and skeleton not in ids["skeletons"]:
        err(f"program.skeleton {skeleton} is not in skeletons.yaml (use 'missing')")
    activities = program.get("activities") or []
    labels = {a.get("label") for a in activities}
    for i, a in enumerate(activities):
        where = f"program.activities[{i}]"
        if not a.get("label") or not a.get("stage"):
            err(f"{where}: label and stage are required")
        frag = a.get("fragment")
        if frag is not None and frag not in ids["fragments"]:
            err(f"{where}: fragment {frag} is not in fragments.yaml")
        for dep in a.get("after") or []:
            if dep not in labels:
                err(f"{where}: after names an unknown activity: {dep}")
        built_in(where, a)

    for i, r in enumerate(shape["rfps"]):
        where = f"rfps[{i}]"
        if r.get("discipline") not in ids["disciplines"]:
            err(f"{where}: discipline {r.get('discipline')} is not in disciplines.json")
        if r.get("template") is not None and r.get("template") not in ids["rfps"]:
            err(f"{where}: template {r.get('template')} is not a knowledge/rfp id")
        if not r.get("must_include"):
            err(f"{where}: must_include is required")
        built_in(where, r)
        expect = r.get("expect")
        if expect is not None:
            if not isinstance(expect, dict):
                err(f"{where}: expect must be a mapping")
            else:
                seen = {}
                for lst in STAGE_LISTS:
                    for stage in expect.get(lst) or []:
                        if stage in seen:
                            err(f"{where}: expect stage {stage} is in both {seen[stage]} and {lst} (both include/option/absent)")
                        seen[stage] = lst
        for j, d in enumerate(r.get("app_differs") or []):
            if not re.match(r"^(stage|position|element|text)\.", str(d.get("key", ""))):
                err(f"{where} app_differs[{j}]: key {d.get('key')} must start with stage., position., element. or text.")
            if not d.get("why"):
                err(f"{where} app_differs[{j}]: why is required")

    trigger_ids = set()
    for i, t in enumerate(shape["triggers"]):
        where = f"triggers[{i}] {t.get('id')}"
        if not t.get("id") or t["id"] in trigger_ids:
            err(f"{where}: id missing or repeated")
        trigger_ids.add(t.get("id"))
        if len(t.get("evidence") or []) < 2:
            err(f"{where}: needs at least two evidence phrasings")
        if len(t.get("negation") or []) < 1:
            err(f"{where}: needs at least one negation phrasing")
        adds = t.get("adds") or {}
        if not any(adds.get(k) for k in ("procurement", "work_items", "cost_plan", "program", "rfps", "authority", "profile")):
            err(f"{where}: adds must name at least one consequence")
        for key in t.get("profile") or []:
            if not _profile_key_ok(key, ids):
                err(f"{where}: profile key {key} is not an app row key")
        for d in adds.get("procurement") or []:
            if d not in ids["disciplines"]:
                err(f"{where}: adds discipline {d} is not in disciplines.json")
        for j, w in enumerate(adds.get("work_items") or []):
            work_item(f"{where} adds.work_items[{j}]", w)
        removes = t.get("removes") or {}
        for area in ("procurement", "rfps"):
            for d in removes.get(area) or []:
                if d not in ids["disciplines"]:
                    err(f"{where}: removes.{area} discipline {d} is not in disciplines.json")
        claims = {area: _claim(c) for area, c in (t.get("built_in") or {}).items()}
        for area, claim in claims.items():
            if claim not in BUILT_IN:
                err(f"{where}: built_in.{area} must be yes, no or partial")
        if any(c in ("yes", "partial") for c in claims.values()) and not _source_ok(str(t.get("source", "")), root):
            err(f"{where}: built_in claims need an existing source")
    for name in shape["variants"]:
        if name not in trigger_ids:
            err(f"variants: {name} is not a trigger id")
    for i, item in enumerate(shape.get("not_expected_procurement") or []):
        if item.get("discipline") not in ids["disciplines"]:
            err(f"not_expected_procurement[{i}]: discipline {item.get('discipline')} is not in disciplines.json")
        if not item.get("why"):
            err(f"not_expected_procurement[{i}]: why is required")
    for i, item in enumerate(shape.get("not_expected_cost_plan") or []):
        if not item.get("line") or not item.get("why"):
            err(f"not_expected_cost_plan[{i}]: line and why are required")
        if item.get("section") not in ("fees_and_charges", "consultants", "construction", "contingency"):
            err(f"not_expected_cost_plan[{i}]: section must be fees_and_charges, consultants, construction or contingency")
    for name in ("expected_derived", "scenario_inputs"):
        for i, item in enumerate(shape.get(name) or []):
            if not _profile_key_ok(str(item.get("key", "")), ids):
                err(f"{name}[{i}]: key {item.get('key')} is not an app row key")
            if not item.get("note"):
                err(f"{name}[{i}]: note is required")
    for i, item in enumerate(shape.get("scenario_inputs") or []):
        if not item.get("phrasing"):
            err(f"scenario_inputs[{i}]: phrasing is required")
    for i, g in enumerate(shape["gaps"]):
        if not g.get("area") or not g.get("what") or not g.get("evidence"):
            err(f"gaps[{i}]: area, what and evidence are required")

    counts = {"triggers": len(shape["triggers"]), "procurement": len(shape["procurement"]),
              "cost_plan": len(shape["cost_plan"]), "activities": len(activities), "rfps": len(shape["rfps"])}
    for key, n in counts.items():
        if n < MIN[key]:
            warnings.append(f"thin: {n} {key} (expected at least {MIN[key]})")
    if shape["reviewed"] is not False:
        warnings.append("reviewed is not false: only the owner marks a shape reviewed")
    return errors, warnings


def main(argv):
    if not argv or argv[0] in ("-h", "--help"):
        print(__doc__)
        return 0 if argv else 2
    ids = load_ids(Path("."))
    failed = False
    for name in argv:
        errors, warnings = validate(name, ids)
        status = "FAIL" if errors else "ok"
        print(f"{status}  {name}")
        for e in errors:
            print(f"  error: {e}")
        for w in warnings:
            print(f"  warning: {w}")
        failed = failed or bool(errors)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
