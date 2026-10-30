"""Score SiteWise's cost plan, procurement, program and RFPs against a shape.yaml.

The input is a snapshot of the app's own API responses, saved by the harness
(see testing/synthetic/SCORING.md for the format and the endpoints), plus a
domain map's shape.yaml. The output is a JSON score and a short summary. Every
failure category is counted on its own; there is no blended number.

Usage, from the repo root:

    python testing/synthetic/tools/score_shapes.py --shape testing/synthetic/types/T01-new-house/shape.yaml \
        --snapshot <run>/snapshot [--trigger cdc-pathway] [--json <run>/score.json]

A variant scenario passes its trigger id: the base expectations plus the
trigger's `adds`, minus its `removes` (removed items become not expected).
"""
import argparse
import copy
import json
import re
import sys
from pathlib import Path

import yaml

SCORE_SCHEMA = "sitewise-synthetic-score/1"
ROOT = Path(__file__).resolve().parents[3]
DISCIPLINES = ROOT / "data" / "intake" / "disciplines.json"

# A phrase is found in a control label or a block of RFP text when this share
# of its words is there. Prose phrasing in shapes is approximate; the score
# shows where each phrase was found so an owner can check it.
PHRASE_COVERAGE = 0.75
# Two activity labels are the same task when this share of their words agrees.
SIMILAR_LABEL = 0.75
STOP = {"a", "an", "the", "and", "or", "of", "for", "to", "in", "on", "with", "by", "at", "from", "as", "incl",
        "including", "into", "its", "their", "is", "be", "per", "via", "any", "all", "each", "through"}
PRESENT = {"include", "option", "I", "O"}
ABSENT = {"exclude", "done", "X", None}
EXCLUDED_CELLS = {"x", "excluded", "not in this rfp", "done by others", "by others"}
STAGE_STATES = (("stages_include", "include"), ("stages_option", "option"), ("stages_absent", "exclude"), ("stages_ask", "ask"))
ELEMENT_STATES = (("elements_include", "include"), ("elements_option", "option"), ("elements_absent", "exclude"))
CODE = re.compile(r"^\s*[A-Z]?\d+(?:\.\d+)*\s+")


# ---------------------------------------------------------------- words

def norm(text):
    """Lower case, no leading line code ("F04 ", "13.1 "), no bracketed asides, words only."""
    t = CODE.sub("", str(text or ""))
    t = re.sub(r"\([^)]*\)", " ", t).lower().replace("&", " and ").replace("_", " ")
    return " ".join(re.sub(r"[^a-z0-9]+", " ", t).split())


def _stem(word):
    return word[:-1] if len(word) > 3 and word.endswith("s") and not word.endswith("ss") else word


def tokens(text):
    return {_stem(w) for w in norm(text).split() if w not in STOP}


def coverage(phrase, unit):
    return len(phrase & unit) / len(phrase) if phrase else 0.0


def jaccard(a, b):
    return len(a & b) / len(a | b) if a | b else 0.0


def _claim(value):
    """PyYAML reads a bare yes/no as a boolean; treat it as the word."""
    if value is True:
        return "yes"
    if value is False:
        return "no"
    return value if value in ("yes", "no", "partial") else "unknown"


# ---------------------------------------------------------------- inputs

class Disciplines:
    """Discipline ids by label and alias, from the app's data/intake/disciplines.json."""

    def __init__(self, path=DISCIPLINES):
        data = json.loads(Path(path).read_text(encoding="utf8"))["disciplines"]
        self.names = []
        for d in data:
            for name in [d["label"], *d.get("aliases", [])]:
                if norm(name):
                    self.names.append((norm(name), d["id"]))

    def by_title(self, title):
        """The discipline a package title names exactly, or None."""
        t = norm(title)
        return next((i for n, i in self.names if n == t), None)

    def in_text(self, text):
        """The discipline whose longest label or alias is inside text ("Quantity surveyor (optional)")."""
        words = set(norm(text).split())
        best, size = None, 0
        for name, ident in self.names:
            parts = name.split()
            if set(parts) <= words and len(parts) > size:
                best, size = ident, len(parts)
        return best


def load_snapshot(path):
    """A snapshot folder (packages.json, cost_plan.json, program.json, reports/*.json) or one combined JSON file."""
    p = Path(path)
    if p.is_file():
        return json.loads(p.read_text(encoding="utf8"))
    out = {}
    for key in ("packages", "cost_plan", "program"):
        f = p / f"{key}.json"
        if f.exists():
            out[key] = json.loads(f.read_text(encoding="utf8"))
    if (p / "reports").is_dir():
        out["reports"] = [json.loads(f.read_text(encoding="utf8")) for f in sorted((p / "reports").glob("*.json"))]
    elif (p / "reports.json").exists():
        out["reports"] = json.loads((p / "reports.json").read_text(encoding="utf8"))
    return out


# ---------------------------------------------------------------- expectations

def expectations(shape, triggers=()):
    """The base expectations, plus each trigger's adds, minus its removes."""
    by_id = {t["id"]: t for t in shape.get("triggers") or []}
    for tid in triggers:
        if tid not in by_id:
            raise ValueError(f"unknown trigger: {tid}")
    e = {
        "procurement": copy.deepcopy(shape.get("procurement") or []),
        "not_procurement": copy.deepcopy(shape.get("not_expected_procurement") or []),
        "cost_plan": copy.deepcopy(shape.get("cost_plan") or []),
        "not_cost_plan": copy.deepcopy(shape.get("not_expected_cost_plan") or []),
        "activities": copy.deepcopy((shape.get("program") or {}).get("activities") or []),
        "not_activities": [],
        "rfps": copy.deepcopy(shape.get("rfps") or []),
        "not_rfps": [],
        "skeleton": (shape.get("program") or {}).get("skeleton"),
    }
    for tid in triggers:
        t = by_id[tid]
        adds, removes = t.get("adds") or {}, t.get("removes") or {}
        claim = {k: _claim(v) for k, v in (t.get("built_in") or {}).items()}
        why = f"removed by trigger {tid}"

        gone = set(removes.get("procurement") or [])
        e["not_procurement"] += [dict(p, why=why) for p in e["procurement"] if p["discipline"] in gone]
        e["procurement"] = [p for p in e["procurement"] if p["discipline"] not in gone]
        added = set(adds.get("procurement") or [])
        for d in sorted(added - {p["discipline"] for p in e["procurement"]}):
            e["procurement"].append({"discipline": d, "kind": None, "built_in": claim.get("procurement", "unknown"), "trigger": tid})
        e["not_procurement"] = [p for p in e["not_procurement"] if p["discipline"] not in added]

        gone = {norm(x) for x in removes.get("cost_plan") or []}
        e["not_cost_plan"] += [dict(c, why=why) for c in e["cost_plan"] if norm(c["line"]) in gone]
        e["cost_plan"] = [c for c in e["cost_plan"] if norm(c["line"]) not in gone]
        added = {norm(x): x for x in adds.get("cost_plan") or []}
        have = {norm(c["line"]) for c in e["cost_plan"]}
        e["cost_plan"] += [{"line": x, "section": None, "built_in": claim.get("cost_plan", "unknown"), "trigger": tid}
                           for k, x in added.items() if k not in have]
        e["not_cost_plan"] = [c for c in e["not_cost_plan"] if norm(c["line"]) not in added]

        gone = set(removes.get("program") or [])
        e["not_activities"] += [dict(a, why=why) for a in e["activities"] if a["label"] in gone]
        e["activities"] = [a for a in e["activities"] if a["label"] not in gone]
        for a in e["activities"]:
            a["after"] = [d for d in a.get("after") or [] if d not in gone]
        have = {a["label"] for a in e["activities"]}
        e["activities"] += [{"label": x, "stage": None, "fragment": None, "after": [], "built_in": claim.get("program", "unknown"), "trigger": tid}
                            for x in adds.get("program") or [] if x not in have]
        e["not_activities"] = [a for a in e["not_activities"] if a["label"] not in set(adds.get("program") or [])]

        gone = set(removes.get("rfps") or [])
        e["not_rfps"] += [dict(r, why=why) for r in e["rfps"] if r["discipline"] in gone]
        e["rfps"] = [r for r in e["rfps"] if r["discipline"] not in gone]
        for add in adds.get("rfps") or []:
            target = next((r for r in e["rfps"] if r["discipline"] == add["discipline"]), None)
            if target is None:
                target = {"discipline": add["discipline"], "template": add.get("template"), "built_in": claim.get("rfps", "unknown"),
                          "must_include": [], "must_not_include": [], "trigger": tid}
                e["rfps"].append(target)
            target["must_include"] = list(target.get("must_include") or []) + list(add.get("must_include") or [])
            target["must_not_include"] = list(target.get("must_not_include") or []) + list(add.get("must_not_include") or [])
        e["not_rfps"] = [r for r in e["not_rfps"] if r["discipline"] not in {a["discipline"] for a in adds.get("rfps") or []}]
    return e


# ---------------------------------------------------------------- matching

def pair(expected, app, rules):
    """One-to-one matches. Rules run in order over every expected item; each rule
    scores an (expected, app) pair or returns None, and the best free app item wins."""
    pairs, used = {}, set()
    for name, rule in rules:
        for ei, e in enumerate(expected):
            if ei in pairs:
                continue
            best, best_score = None, None
            for ai, a in enumerate(app):
                if ai in used:
                    continue
                s = rule(e, a)
                if s is not None and (best_score is None or s > best_score):
                    best, best_score = ai, s
            if best is not None:
                pairs[ei] = (best, name)
                used.add(best)
    return pairs, used


def _anchor(source):
    """The anchor after a knowledge path: 'knowledge/costs/starter_lines.yaml da-fees' -> ('starter_lines.yaml', 'da-fees')."""
    parts = str(source or "").split()
    if len(parts) < 2:
        return None, None
    return parts[0].rsplit("/", 1)[-1], parts[-1]


def _counts(result, expected_key="expected"):
    missed = result["missed"]
    return {
        "expected": len(result["matched"]) + len(missed),
        "matched": len(result["matched"]),
        "missed": len(missed),
        "missed_built_in": sum(1 for m in missed if m.get("built_in") == "yes"),
        "missed_known_gap": sum(1 for m in missed if m.get("built_in") in ("no", "partial")),
        "extra": len(result["extra"]),
        "false_positive": len(result["false_positive"]),
    }


def _same_discipline(e, a):
    return 1.0 if a["discipline"] and a["discipline"] == e["discipline"] else None


def _alias_discipline(e, a):
    """Other discipline ids the shape says also count (CONVENTIONS.md C2)."""
    return 1.0 if a["discipline"] and a["discipline"] in (e.get("aliases") or []) else None


def _package_disciplines(packages):
    out = {}
    for p in (packages or {}).get("items") or []:
        d = p.get("discipline_id") or ("trade.main_works" if p.get("kind") == "works" and p.get("works_scope") == "head_contract" else "")
        out[p.get("id")] = d
    return out


# ---------------------------------------------------------------- procurement

def score_procurement(exp, packages, disc):
    out = {"status": "scored", "matched": [], "missed": [], "extra": [], "false_positive": [], "kind_mismatch": []}
    if packages is None:
        out["status"] = "not_captured"
    app = []
    for p in (packages or {}).get("items") or []:
        if p.get("retired_at") or p.get("lifecycle_status") == "cancelled":
            continue
        d = _package_disciplines({"items": [p]}).get(p.get("id")) or disc.by_title(p.get("title"))
        app.append({"discipline": d, "app": p.get("title"), "kind": p.get("kind"), "state": "package"})
    for s in (packages or {}).get("suggestions") or []:
        app.append({"discipline": disc.by_title(s.get("title")), "app": s.get("title"), "kind": s.get("kind"), "state": "suggested"})
    works = (packages or {}).get("proposed_works")
    if works and not works.get("package_id"):
        app.append({"discipline": "trade.main_works", "app": works.get("title"), "kind": "works", "state": "proposed_works"})

    same = [("discipline", _same_discipline), ("alias", _alias_discipline)]
    pairs, used = pair(exp["procurement"], app, same)
    for ei, e in enumerate(exp["procurement"]):
        if ei in pairs:
            a, how = app[pairs[ei][0]], pairs[ei][1]
            out["matched"].append({"expected": e["discipline"], "app": a["app"], "state": a["state"], "matched_by": how})
            if e.get("kind") and a["kind"] and e["kind"] != a["kind"]:
                out["kind_mismatch"].append({"expected": e["discipline"], "want": e["kind"], "app": a["kind"]})
        else:
            out["missed"].append({"expected": e["discipline"], "built_in": _claim(e.get("built_in")), "need": e.get("need"), "trigger": e.get("trigger")})
    rest = [a for i, a in enumerate(app) if i not in used]
    fp_pairs, fp_used = pair(exp["not_procurement"], rest, same)
    for ei, (ai, _) in fp_pairs.items():
        a = rest[ai]
        out["false_positive"].append({"discipline": a["discipline"], "app": a["app"], "state": a["state"], "why": exp["not_procurement"][ei].get("why")})
    out["extra"] = [{"discipline": a["discipline"], "app": a["app"], "state": a["state"]} for i, a in enumerate(rest) if i not in fp_used]
    return out


# ---------------------------------------------------------------- cost plan

def _section(line):
    kind = line.get("line_kind")
    if kind == "fee":
        return "consultants"
    if kind == "works":
        return "construction"
    return "fees_and_charges" if line.get("category") == "authority_fees" else "contingency"


def classify_amount(item, value):
    """Where a figure comes from: 'document', 'user' or 'fabricated' (no source document, nobody typed it)."""
    prov = value.get("provenance") or {}
    if isinstance(prov, str):
        try:
            prov = json.loads(prov)
        except ValueError:
            prov = {}
    if value.get("origin") == "user":
        return "document" if str(value.get("rationale") or "").startswith("Awarded to ") else "user"
    if "spread" in prov:
        return {"document": "document", "user": "user"}.get(prov.get("budget_origin"), "fabricated")
    if "brief" in prov:
        return {"document": "document", "user": "user"}.get(prov.get("origin"), "fabricated")
    return "fabricated"


def _cost_lines(plan, packages, disc):
    pkg = _package_disciplines(packages)
    lines = []
    for state, rows in (("item", (plan or {}).get("items") or []), ("suggestion", (plan or {}).get("suggestions") or [])):
        for r in rows:
            d = None
            if r.get("line_kind") == "fee":
                d = pkg.get(r.get("package_id")) or disc.by_title(r.get("label"))
            lines.append({"row": r, "label": r.get("label", ""), "code": str(r.get("code") or ""), "state": state,
                          "section": _section(r), "source_key": r.get("source_key") or "", "discipline": d,
                          "posting": r.get("posting", True) is not False})
    return lines


def _cost_rules(disc):
    def aliases(e, a):
        names = [e.get("app_label"), *(e.get("aliases") or [])]
        full = norm(f"{a['code']} {a['label']}")
        return 1.0 if any(n and norm(n) in (full, norm(a["label"])) for n in names) else None

    def label(e, a):
        return 1.0 if norm(e["line"]) == norm(a["label"]) else None

    def source(e, a):
        path, anchor = _anchor(e.get("source"))
        if path == "starter_lines.yaml":
            return 1.0 if a["source_key"] == "pw:" + anchor else None
        if path == "elements.yaml" and str(e.get("source")).split()[-2:-1] == ["code"]:
            return 1.0 if a["posting"] and a["row"].get("line_kind") == "works" and a["code"] == anchor else None
        return None

    def discipline(e, a):
        if e.get("section") not in ("consultants", None) or a["section"] != "consultants" or not a["discipline"]:
            return None
        return 1.0 if disc.in_text(e["line"]) == a["discipline"] else None

    def prefix(e, a):
        if e.get("section") and e["section"] != a["section"]:
            return None
        return 1.0 if norm(e["line"]) and norm(a["label"]).startswith(norm(e["line"]) + " ") else None

    return [("app_label", aliases), ("label", label), ("source", source), ("discipline", discipline), ("prefix", prefix)]


def score_cost_plan(exp, plan, packages, disc):
    out = {"status": "scored", "matched": [], "missed": [], "extra": [], "false_positive": [], "section_mismatch": [],
           "groups": 0, "fabricated": [], "amounts": {"document": 0, "user": 0, "fabricated": 0}}
    if plan is None:
        out["status"] = "not_captured"
    elif "plan" in plan and "items" not in plan:  # POST /cost-plan/prepare wraps the plan
        plan = plan["plan"]
    lines = _cost_lines(plan, packages, disc)
    rules = _cost_rules(disc)
    pairs, used = pair(exp["cost_plan"], lines, rules)
    for ei, e in enumerate(exp["cost_plan"]):
        if ei in pairs:
            a, how = lines[pairs[ei][0]], pairs[ei][1]
            out["matched"].append({"expected": e["line"], "app": a["label"], "code": a["code"], "state": a["state"], "matched_by": how})
            if e.get("section") and e["section"] != a["section"]:
                out["section_mismatch"].append({"expected": e["line"], "want": e["section"], "app": a["section"]})
        else:
            out["missed"].append({"expected": e["line"], "section": e.get("section"), "built_in": _claim(e.get("built_in")), "trigger": e.get("trigger")})
    rest = [a for i, a in enumerate(lines) if i not in used]
    fp_pairs, fp_used = pair(exp["not_cost_plan"], rest, rules)
    for ei, (ai, how) in fp_pairs.items():
        out["false_positive"].append({"expected": exp["not_cost_plan"][ei]["line"], "app": rest[ai]["label"], "code": rest[ai]["code"],
                                      "matched_by": how, "why": exp["not_cost_plan"][ei].get("why")})
    for i, a in enumerate(rest):
        if i in fp_used:
            continue
        if not a["posting"]:
            out["groups"] += 1  # a container line; its children are scored
            continue
        out["extra"].append({"app": a["label"], "code": a["code"], "section": a["section"], "state": a["state"]})

    for a in lines:
        r = a["row"]
        for metric, value in sorted((r.get("values") or {}).items()):
            if not any(value.get(k) is not None for k in ("amount", "low", "high")):
                continue
            kind = classify_amount(r, value)
            out["amounts"][kind] += 1
            if kind == "fabricated":
                out["fabricated"].append({"app": a["label"], "code": a["code"], "metric": metric, "amount": value.get("amount"),
                                          "origin": value.get("origin"), "rationale": value.get("rationale", "")})
        if (r.get("rate") is not None or r.get("quantity") is not None) and r.get("origin") != "user":
            out["amounts"]["fabricated"] += 1
            out["fabricated"].append({"app": a["label"], "code": a["code"], "metric": "rate", "amount": r.get("rate"), "origin": r.get("origin")})
    return out


# ---------------------------------------------------------------- program

def _cycles(ids, links):
    """Loops in the link graph, each as a closed list of activity ids (iterative DFS)."""
    succ = {}
    for l in links:
        succ.setdefault(l.get("pred_id"), []).append(l.get("succ_id"))
    colour, found = {}, []
    for root in ids:
        if colour.get(root):
            continue
        stack, path = [(root, iter(succ.get(root, [])))], [root]
        colour[root] = 1
        while stack:
            node, it = stack[-1]
            nxt = next(it, None)
            if nxt is None:
                colour[node] = 2
                stack.pop()
                path.pop()
            elif colour.get(nxt) == 1:
                found.append(path[path.index(nxt):] + [nxt])
            elif not colour.get(nxt):
                colour[nxt] = 1
                stack.append((nxt, iter(succ.get(nxt, []))))
                path.append(nxt)
    return found


def _program_rules():
    def label(e, a):
        return 1.0 if norm(e["label"]) == norm(a["label"]) else None

    def alias(e, a):
        return 1.0 if any(norm(x) == norm(a["label"]) for x in e.get("aliases") or []) else None

    def interface(e, a):
        _, anchor = _anchor(e.get("source"))
        return 1.0 if anchor and anchor.startswith("if.") and a.get("interface_id") == anchor else None

    def fragment(e, a):
        if not e.get("fragment") or a.get("fragment_id") != e["fragment"]:
            return None
        return jaccard(tokens(e["label"]), tokens(a["label"]))

    def similar(e, a):
        s = jaccard(tokens(e["label"]), tokens(a["label"]))
        return s if s >= SIMILAR_LABEL else None

    return [("label", label), ("alias", alias), ("interface", interface), ("fragment", fragment), ("similar_label", similar)]


def score_program(exp, program):
    acts_out = {"matched": [], "missed": [], "extra": [], "false_positive": [], "wrong_stage": []}
    out = {"status": "scored", "skeleton": {}, "activities": acts_out,
           "links": {"holds": [], "violated": [], "unknown": [], "unscored": []}, "orphans": [], "cycles": []}
    if program is None:
        out["status"] = "not_captured"
        program = {}
    elif program.get("empty"):
        out["status"] = "empty"
    rows = program.get("activities") or []
    links = program.get("links") or []
    by_id = {r["id"]: r for r in rows}
    stage_of = {r["id"]: (by_id.get(r.get("parent_id")) or {}).get("catalogue_stage_id") for r in rows}
    app = [r for r in rows if r.get("kind") != "stage"]
    want = None if exp["skeleton"] in (None, "missing") else exp["skeleton"]
    got = (program.get("skeleton") or {}).get("id") or (program.get("program") or {}).get("skeleton_id")
    out["skeleton"] = {"expected": exp["skeleton"], "app": got, "match": got == want}

    rules = _program_rules()
    pairs, used = pair(exp["activities"], app, rules)
    matched = {}
    for ei, e in enumerate(exp["activities"]):
        if ei in pairs:
            a, how = app[pairs[ei][0]], pairs[ei][1]
            matched[e["label"]] = a
            acts_out["matched"].append({"expected": e["label"], "app": a["label"], "matched_by": how})
            if e.get("stage") and stage_of.get(a["id"]) and e["stage"] != stage_of[a["id"]]:
                acts_out["wrong_stage"].append({"expected": e["label"], "want_stage": e["stage"], "app_stage": stage_of[a["id"]]})
        else:
            acts_out["missed"].append({"expected": e["label"], "stage": e.get("stage"), "built_in": _claim(e.get("built_in")), "trigger": e.get("trigger")})
    rest = [a for i, a in enumerate(app) if i not in used]
    fp_pairs, fp_used = pair(exp["not_activities"], rest, rules)
    for ei, (ai, how) in fp_pairs.items():
        acts_out["false_positive"].append({"expected": exp["not_activities"][ei]["label"], "app": rest[ai]["label"], "matched_by": how,
                                           "why": exp["not_activities"][ei].get("why")})
    acts_out["extra"] = [{"app": a["label"], "fragment": a.get("fragment_id"), "stage": stage_of.get(a["id"])}
                         for i, a in enumerate(rest) if i not in fp_used]

    times = (program.get("computed") or {}).get("activities") or {}
    for e in exp["activities"]:
        for dep in e.get("after") or []:
            entry = {"activity": e["label"], "after": dep}
            a, b = matched.get(e["label"]), matched.get(dep)
            if a is None or b is None:
                out["links"]["unscored"].append(dict(entry, reason="activity missed" if a is None else "predecessor missed"))
                continue
            ta, tb = times.get(a["id"]) or {}, times.get(b["id"]) or {}
            if ta.get("unknown") or tb.get("unknown") or ta.get("es") is None or tb.get("ef") is None:
                out["links"]["unknown"].append(entry)
            elif ta["es"] >= tb["ef"]:
                out["links"]["holds"].append(entry)
            else:
                out["links"]["violated"].append(dict(entry, starts=ta["es"], predecessor_finishes=tb["ef"]))

    linked = {l.get("pred_id") for l in links} | {l.get("succ_id") for l in links}
    out["orphans"] = [{"app": a["label"], "id": a["id"], "stage": stage_of.get(a["id"])} for a in app if a["id"] not in linked]
    cycles = _cycles([r["id"] for r in rows], links)
    engine = (program.get("computed") or {}).get("cycle")
    if engine and not any(set(engine) == set(c) for c in cycles):
        cycles.append(list(engine))
    out["cycles"] = cycles
    out["cycle_labels"] = [[(by_id.get(i) or {}).get("label", i) for i in c] for c in cycles]
    return out


# ---------------------------------------------------------------- RFPs

def _controls(view):
    """{'stage.k': value, 'element.k': value, 'position.k': value} and each key's label."""
    values, labels = {}, {}
    for g in (view.get("rfp") or {}).get("groups") or []:
        if g.get("id") not in ("stages", "elements", "positions"):
            continue
        for c in g.get("controls") or []:
            kind, _, key = str(c.get("key", "")).partition(":")
            if kind in ("stage", "element", "position") and key:
                values[f"{kind}.{key}"] = c.get("value")
                labels[f"{kind}.{key}"] = c.get("label") or key
    return values, labels


def _same(key, got, want):
    if key.startswith(("stage.", "element.")):
        want = "exclude" if want == "absent" else want
        got = "exclude" if got == "absent" else got
        return got == want
    return (got or "absent") == (want or "absent")


def _compare_expect(r, values, checks):
    out = {"ok": 0, "new_failure": [], "known_difference": [], "fixed": [], "not_checked": []}
    expect = r.get("expect") or {}
    differs = {d["key"]: d for d in r.get("app_differs") or []}
    wanted = {}
    for lst, state in STAGE_STATES:
        for k in expect.get(lst) or []:
            wanted["stage." + k] = state
    for lst, state in ELEMENT_STATES:
        for k in expect.get(lst) or []:
            wanted["element." + k] = state
    for k, v in (expect.get("positions") or {}).items():
        wanted["position." + k] = v
    for key, d in differs.items():
        if key.startswith("text."):
            out["not_checked"].append({"key": key, "why": d.get("why")})
        elif key not in wanted:
            wanted[key] = d.get("expect")
    for key, want in wanted.items():
        present = key in values
        got = values.get(key)
        if key.startswith(("stage.", "element.")) and not present:
            out["new_failure"].append({"key": key, "want": want, "app": None, "why": "the app has no such key"})
            continue
        d = differs.get(key)
        if _same(key, got, want):
            if d:
                out["fixed"].append({"key": key, "want": want, "app": got, "was": d.get("app")})
            else:
                out["ok"] += 1
        elif d and _same(key, got, d.get("app")):
            out["known_difference"].append({"key": key, "want": want, "app": got, "why": d.get("why")})
        else:
            out["new_failure"].append({"key": key, "want": want, "app": got if present else "absent", "app_differs_said": d.get("app") if d else None})
    if "blocking" in expect:
        want_b = set(expect.get("blocking") or [])
        for c in sorted(want_b - checks):
            out["new_failure"].append({"key": "blocking." + c, "want": "blocking", "app": "not blocking"})
        for c in sorted(checks - want_b):
            out["new_failure"].append({"key": "blocking." + c, "want": "not blocking", "app": "blocking"})
    return out


def _text_units(view):
    draft = view.get("draft")
    if not draft:
        return None
    units = []
    for sec in draft.get("sections") or []:
        if sec.get("internal"):
            continue  # PM review lists are never issued
        for b in sec.get("blocks") or []:
            units.append((f"{sec.get('id')}/{b.get('id')}", f"{b.get('label', '')} {b.get('text', '')}"))
            for n, row in enumerate(((b.get("table") or {}).get("rows")) or []):
                if any(norm(cell) in EXCLUDED_CELLS for cell in row):
                    continue  # a printed exclusion is not the item being in scope
                units.append((f"{sec.get('id')}/{b.get('id')}#{n}", " ".join(row)))
    return units


def phrase_text(phrase):
    """A must_include item is a string or {text, key}."""
    return phrase.get("text", "") if isinstance(phrase, dict) else str(phrase)


def _state(value):
    return "present" if value in PRESENT else "absent" if value in ABSENT else "unknown"


def find_phrase(phrase, values, labels, units):
    """Where an RFP carries a phrase: on its keyed control, on a control whose label
    holds its words (by the control's state), or in issued text. An exact key beats
    word overlap; a keyed row the app does not have is absent.
    Returns (state, where, coverage, control key) with state present, absent or unknown."""
    if isinstance(phrase, dict) and phrase.get("key"):
        key = phrase["key"]
        return _state(values.get(key)), "key:" + key, 1.0, key
    text = re.sub(r"^\s*option:\s*", "", phrase_text(phrase), flags=re.I)
    want = tokens(text)
    if not want:
        return "unknown", None, 0.0, None
    best = None
    for key, label in labels.items():
        lt = tokens(label)
        cov = coverage(want, lt)
        if cov >= PHRASE_COVERAGE and (best is None or (cov, jaccard(want, lt)) > best[0]):
            best = ((cov, jaccard(want, lt)), key)
    if best:
        key = best[1]
        return _state(values.get(key)), "control:" + key, best[0][0], key
    if units is None:
        return "unknown", "no draft", 0.0, None
    scored = [(coverage(want, tokens(t)), where) for where, t in units]
    cov, where = max(scored, default=(0.0, None))
    return ("present" if cov >= PHRASE_COVERAGE else "absent"), (where if cov >= PHRASE_COVERAGE else None), round(cov, 2), None


def _phrases(r, view, values, labels):
    units = _text_units(view)
    differs = {d["key"]: d for d in r.get("app_differs") or []}
    inc = {"present": 0, "missing": [], "unknown": [], "known": []}
    for item in r.get("must_include") or []:
        p = phrase_text(item)
        state, where, cov, key = find_phrase(item, values, labels, units)
        if state == "present":
            inc["present"] += 1
        elif state == "unknown":
            inc["unknown"].append({"phrase": p, "where": where})
        elif key and key in differs and _same(key, values.get(key), differs[key].get("app")):
            inc["known"].append({"phrase": p, "where": where})
        else:
            inc["missing"].append(p)
    exc = {"absent": 0, "present": [], "unknown": [], "known": []}
    for item in r.get("must_not_include") or []:
        p = phrase_text(item)
        state, where, cov, key = find_phrase(item, values, labels, units)
        if state == "absent":
            exc["absent"] += 1
        elif state == "unknown":
            exc["unknown"].append({"phrase": p, "where": where})
        elif key and key in differs and _same(key, values.get(key), differs[key].get("app")):
            exc["known"].append({"phrase": p, "where": where})
        else:
            exc["present"].append({"phrase": p, "where": where, "coverage": cov})
    return inc, exc


def score_rfps(exp, reports, packages):
    pkg = _package_disciplines(packages)
    app = []
    for v in reports or []:
        rep = v.get("report") or {}
        if rep.get("kind") not in (None, "rfp"):
            continue
        app.append({"view": v, "template": (v.get("rfp") or {}).get("discipline"), "discipline": pkg.get(rep.get("package_id")), "id": rep.get("id")})
    # Discipline first: two disciplines can share a template (bca-certification;
    # reference/CONVENTIONS.md C3), so a template match is only a fallback.
    rule = [("discipline", _same_discipline), ("alias", _alias_discipline),
            ("template", lambda e, a: 1.0 if e.get("template") and a["template"] == e["template"] else None)]
    pairs, used = pair(exp["rfps"], app, rule)
    out = []
    for ei, r in enumerate(exp["rfps"]):
        entry = {"discipline": r["discipline"], "template": r.get("template"), "built_in": _claim(r.get("built_in")), "trigger": r.get("trigger")}
        if ei not in pairs:
            entry.update(status="missing", found=False)
            entry["must_include"] = {"present": 0, "missing": [phrase_text(p) for p in r.get("must_include") or []], "unknown": [], "known": []}
            entry["must_not_include"] = {"absent": 0, "present": [], "unknown": [], "known": []}
            entry["expect"] = {"ok": 0, "new_failure": [], "known_difference": [], "fixed": [], "not_checked": []}
            out.append(entry)
            continue
        a = app[pairs[ei][0]]
        values, labels = _controls(a["view"])
        checks = {c.get("id") for c in (a["view"].get("rfp") or {}).get("checks") or [] if c.get("blocking")}
        entry.update(status="found", found=True, report=a["id"], app_template=a["template"])
        entry["expect"] = _compare_expect(r, values, checks) if r.get("expect") or r.get("app_differs") else \
            {"ok": 0, "new_failure": [], "known_difference": [], "fixed": [], "not_checked": []}
        entry["must_include"], entry["must_not_include"] = _phrases(r, a["view"], values, labels)
        out.append(entry)
    rest = [a for i, a in enumerate(app) if i not in used]
    fp_pairs, fp_used = pair(exp["not_rfps"], rest, rule)
    for ei, (ai, _) in fp_pairs.items():
        out.append({"discipline": exp["not_rfps"][ei]["discipline"], "template": rest[ai]["template"], "status": "not_expected",
                    "found": True, "report": rest[ai]["id"], "why": exp["not_rfps"][ei].get("why")})
    for i, a in enumerate(rest):
        if i not in fp_used:
            out.append({"discipline": a["discipline"], "template": a["template"], "status": "extra", "found": True, "report": a["id"]})
    return out


# ---------------------------------------------------------------- score and summary

def score(shape, outputs, triggers=None, disciplines=None):
    triggers = list(triggers or [])
    disc = disciplines or Disciplines()
    exp = expectations(shape, triggers)
    result = {
        "schema": SCORE_SCHEMA,
        "type_id": shape.get("type_id"),
        "name": shape.get("name"),
        "scenario": {"triggers": triggers},
        "procurement": score_procurement(exp, outputs.get("packages"), disc),
        "cost_plan": score_cost_plan(exp, outputs.get("cost_plan"), outputs.get("packages"), disc),
        "program": score_program(exp, outputs.get("program")),
        "rfps": score_rfps(exp, outputs.get("reports"), outputs.get("packages")),
    }
    if outputs.get("reports") is None:
        result["rfps_status"] = "not_captured"
    result["summary"] = _summary_counts(result)
    return result


def _summary_counts(result):
    cp = result["cost_plan"]
    prog = result["program"]
    rfps = [r for r in result["rfps"] if r["status"] in ("found", "missing")]
    found = [r for r in rfps if r["found"]]
    return {
        "procurement": dict(_counts(result["procurement"]), kind_mismatch=len(result["procurement"]["kind_mismatch"])),
        "cost_plan": dict(_counts(cp), section_mismatch=len(cp["section_mismatch"]), fabricated=len(cp["fabricated"]),
                          amounts_document=cp["amounts"]["document"], amounts_user=cp["amounts"]["user"]),
        "program": dict(_counts(prog["activities"]), wrong_stage=len(prog["activities"]["wrong_stage"]),
                        skeleton_match=prog["skeleton"]["match"],
                        **{f"links_{k}": len(v) for k, v in prog["links"].items()},
                        orphans=len(prog["orphans"]), cycles=len(prog["cycles"])),
        "rfps": {
            "expected": len(rfps),
            "found": len(found),
            "missing": len(rfps) - len(found),
            "missing_built_in": sum(1 for r in rfps if not r["found"] and r["built_in"] == "yes"),
            "not_expected_present": sum(1 for r in result["rfps"] if r["status"] == "not_expected"),
            "extra": sum(1 for r in result["rfps"] if r["status"] == "extra"),
            "expect_ok": sum(r["expect"]["ok"] for r in found),
            "expect_new_failure": sum(len(r["expect"]["new_failure"]) for r in found),
            "expect_known_difference": sum(len(r["expect"]["known_difference"]) for r in found),
            "expect_fixed": sum(len(r["expect"]["fixed"]) for r in found),
            "must_include_missing": sum(len(r["must_include"]["missing"]) for r in found),
            "must_include_unknown": sum(len(r["must_include"]["unknown"]) for r in found),
            "must_not_include_present": sum(len(r["must_not_include"]["present"]) for r in found),
            "must_not_include_known": sum(len(r["must_not_include"]["known"]) for r in found),
        },
    }


def _list(label, items, show=6):
    if not items:
        return []
    more = f" (+{len(items) - show} more)" if len(items) > show else ""
    return [f"    {label}: " + "; ".join(str(i) for i in items[:show]) + more]


def summary(result):
    s = result["summary"]
    variant = ", ".join(result["scenario"]["triggers"]) or "base"
    p, c, g, r = s["procurement"], s["cost_plan"], s["program"], s["rfps"]
    lines = [f"{result['type_id']} {result['name']} ({variant})"]
    lines.append(f"  Procurement [{result['procurement']['status']}]: matched {p['matched']}/{p['expected']}, missed {p['missed']} "
                 f"({p['missed_built_in']} built in, {p['missed_known_gap']} known gaps), extra {p['extra']}, false positives {p['false_positive']}")
    lines += _list("missed built in", [m["expected"] for m in result["procurement"]["missed"] if m["built_in"] == "yes"])
    lines += _list("false positives", [f["discipline"] for f in result["procurement"]["false_positive"]])
    lines.append(f"  Cost plan [{result['cost_plan']['status']}]: matched {c['matched']}/{c['expected']}, missed {c['missed']} "
                 f"({c['missed_built_in']} built in, {c['missed_known_gap']} known gaps), extra {c['extra']}, false positives {c['false_positive']}, "
                 f"fabricated amounts {c['fabricated']} (from documents {c['amounts_document']}, typed {c['amounts_user']})")
    lines += _list("missed built in", [m["expected"] for m in result["cost_plan"]["missed"] if m["built_in"] == "yes"])
    lines += _list("false positives", [f["app"] for f in result["cost_plan"]["false_positive"]])
    lines += _list("fabricated", [f"{f['app']} {f['metric']} {f['amount']}" for f in result["cost_plan"]["fabricated"]])
    sk = result["program"]["skeleton"]
    lines.append(f"  Program [{result['program']['status']}]: skeleton {sk['app']} ({'match' if sk['match'] else 'expected ' + str(sk['expected'])}), "
                 f"matched {g['matched']}/{g['expected']}, missed {g['missed']} ({g['missed_built_in']} built in, {g['missed_known_gap']} known gaps), "
                 f"extra {g['extra']}, false positives {g['false_positive']}, wrong stage {g['wrong_stage']}; "
                 f"links holds {g['links_holds']}, violated {g['links_violated']}, unknown {g['links_unknown']}, unscored {g['links_unscored']}; "
                 f"orphans {g['orphans']}, cycles {g['cycles']}")
    lines += _list("missed built in", [m["expected"] for m in result["program"]["activities"]["missed"] if m["built_in"] == "yes"])
    lines += _list("violated", [f"{v['activity']} after {v['after']}" for v in result["program"]["links"]["violated"]])
    lines += _list("cycles", [" > ".join(c) for c in result["program"]["cycle_labels"]], show=3)
    lines.append(f"  RFPs: found {r['found']}/{r['expected']} ({r['missing_built_in']} built-in missing), not expected present {r['not_expected_present']}, "
                 f"extra {r['extra']}; expect ok {r['expect_ok']}, new failures {r['expect_new_failure']}, known differences "
                 f"{r['expect_known_difference']}, fixed {r['expect_fixed']}; must_include missing {r['must_include_missing']} "
                 f"(unknown {r['must_include_unknown']}); must_not_include present {r['must_not_include_present']} (known {r['must_not_include_known']})")
    for e in result["rfps"]:
        if e["status"] == "found" and (e["expect"]["new_failure"] or e["must_include"]["missing"] or e["must_not_include"]["present"]):
            parts = [f["key"] for f in e["expect"]["new_failure"]] + [f"missing '{m}'" for m in e["must_include"]["missing"]] + \
                    [f"has '{m['phrase']}'" for m in e["must_not_include"]["present"]]
            lines += _list(e["discipline"], parts)
        elif e["status"] in ("missing", "not_expected"):
            lines.append(f"    {e['discipline']}: {e['status']}")
    return "\n".join(lines)


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--shape", required=True)
    ap.add_argument("--snapshot", required=True, help="snapshot folder or combined JSON file")
    ap.add_argument("--trigger", action="append", default=[], help="variant trigger id (repeatable)")
    ap.add_argument("--json", help="write the JSON score here")
    ap.add_argument("--disciplines", default=str(DISCIPLINES))
    args = ap.parse_args(argv)
    try:
        shape = yaml.safe_load(Path(args.shape).read_text(encoding="utf8"))
        result = score(shape, load_snapshot(args.snapshot), args.trigger, Disciplines(args.disciplines))
    except (OSError, ValueError, KeyError) as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    if args.json:
        Path(args.json).write_text(json.dumps(result, indent=2) + "\n", encoding="utf8", newline="\n")
    print(summary(result))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
