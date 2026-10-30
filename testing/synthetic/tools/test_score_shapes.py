"""Checks for score_shapes.py. Run from the repo root:

    python -m unittest discover -s testing/synthetic/tools

Fixtures are built from the T01 shape: `app_from_shape` writes the API
responses a perfect app would return for it (GET /projects/{id}/packages,
/cost-plan, /program and GET /reports/{r}), and each test breaks one thing.
"""
import copy
import json
import tempfile
import unittest
from pathlib import Path

import yaml

import score_shapes as s

T01 = Path(__file__).resolve().parents[1] / "types" / "T01-new-house" / "shape.yaml"

# How the app titles the T01 consultants (discipline labels in data/intake/disciplines.json).
CONSULTANT_LINES = {
    "Architect": "consultant.architect",
    "Structural engineer": "consultant.structural",
    "Civil (stormwater) engineer": "consultant.civil",
    "Geotechnical engineer": "consultant.geotechnical",
    "Surveyor": "consultant.surveyor",
    "Certifier": "consultant.certifier",
    "BASIX and NatHERS assessor": "consultant.basix",
    "Landscape designer": "consultant.landscape",
    "Town planner (optional)": "consultant.town_planner",
    "Quantity surveyor (optional)": "consultant.quantity_surveyor",
}
TITLES = {
    "consultant.architect": "Architect", "consultant.structural": "Structural", "consultant.civil": "Civil",
    "consultant.geotechnical": "Geotechnical", "consultant.surveyor": "Surveyor", "consultant.certifier": "Certifier",
    "consultant.basix": "BASIX", "consultant.landscape": "Landscape", "consultant.town_planner": "Town Planner",
    "consultant.quantity_surveyor": "Quantity Surveyor", "trade.main_works": "Main Works",
    "consultant.mechanical": "Mechanical", "consultant.electrical": "Electrical", "consultant.hydraulic": "Hydraulic",
}


def shape():
    return yaml.safe_load(T01.read_text(encoding="utf8"))


def app_from_shape(sh):
    """The four saved responses a perfect app would give for this shape."""
    packages = {"project_id": "p1", "items": [], "suggestions": [], "route": {}, "proposed_works": None, "gaps": []}
    pkg_of = {}
    for n, p in enumerate(sh["procurement"]):
        pid = f"pkg-{n}"
        pkg_of[p["discipline"]] = pid
        packages["items"].append({"id": pid, "kind": p["kind"], "discipline_id": p["discipline"],
                                  "title": TITLES[p["discipline"]], "lifecycle_status": "active", "stages": []})

    items = []
    for n, line in enumerate(sh["cost_plan"]):
        item = {"cost_item_id": f"ci-{n}", "label": line["line"], "code": "", "line_kind": "works", "category": "",
                "source_key": "", "package_id": "", "parent_item_id": "", "posting": True, "values": {}}
        src = line.get("source", "")
        if line["section"] == "consultants":
            disc = CONSULTANT_LINES[line["line"]]
            item.update(line_kind="fee", package_id=pkg_of.get(disc, f"pkg-x{n}"), label=TITLES[disc], code=f"F{n:02d}")
        elif line["section"] == "fees_and_charges":
            item.update(line_kind="project_wide", category="authority_fees")
        elif line["section"] == "contingency":
            item.update(line_kind="project_wide", category="contingency")
        if "starter_lines.yaml" in src:
            item["source_key"] = "pw:" + src.split()[-1]
        if "elements.yaml code" in src:
            item["code"] = src.split()[-1]
        items.append(item)
    cost_plan = {"id": "plan", "status": "draft", "items": items, "suggestions": [], "dismissed": [], "links": []}

    acts, links, computed = [], [], {}
    stage_ids = {}
    for a in sh["program"]["activities"]:
        if a["stage"] not in stage_ids:
            sid = f"st-{a['stage']}"
            stage_ids[a["stage"]] = sid
            acts.append({"id": sid, "parent_id": None, "kind": "stage", "label": a["stage"], "catalogue_stage_id": a["stage"],
                         "fragment_id": None, "interface_id": None})
    by_label = {}
    for n, a in enumerate(sh["program"]["activities"]):
        aid = f"a-{n}"
        by_label[a["label"]] = aid
        iface = None
        src = a.get("source", "") or ""
        if "interfaces.yaml" in src:
            iface = src.split()[-1]
        acts.append({"id": aid, "parent_id": stage_ids[a["stage"]], "kind": "activity", "label": a["label"],
                     "catalogue_stage_id": None, "fragment_id": a.get("fragment"), "interface_id": iface})
    for n, a in enumerate(sh["program"]["activities"]):
        for dep in a.get("after") or []:
            links.append({"id": f"l-{len(links)}", "pred_id": by_label[dep], "succ_id": by_label[a["label"]], "type": "FS", "lag_days": 0})
    # As the app's chainLinks (internal/program/generate.go step 7): inside a stage, a row
    # with no predecessor is chained behind the row before it.
    prev = {}
    for a in sh["program"]["activities"]:
        aid = by_label[a["label"]]
        if not a.get("after") and a["stage"] in prev:
            links.append({"id": f"l-{len(links)}", "pred_id": prev[a["stage"]], "succ_id": aid, "type": "FS", "lag_days": 0})
        prev[a["stage"]] = aid
    # Forward pass: activities are listed in a topological order in the T01 shape.
    for a in sh["program"]["activities"]:
        aid = by_label[a["label"]]
        es = max([computed[l["pred_id"]]["ef"] for l in links if l["succ_id"] == aid] or [0])
        computed[aid] = {"es": es, "ef": es + 5, "unknown": False, "start": None, "finish": None}
    for sid in stage_ids.values():
        computed[sid] = {"es": 0, "ef": 0, "unknown": False}
    program = {"program": {"id": "prog", "skeleton_id": sh["program"]["skeleton"]}, "activities": acts, "links": links,
               "computed": {"activities": computed, "cycle": None},
               "skeleton": {"id": sh["program"]["skeleton"], "label": "", "reason": ""}}

    reports = []
    for n, r in enumerate(sh["rfps"]):
        if not r.get("template"):
            continue
        expect = r.get("expect") or {}
        differs = {d["key"]: d["app"] for d in r.get("app_differs") or []}
        stages = []
        for lst, state in (("stages_include", "include"), ("stages_option", "option"), ("stages_absent", "exclude")):
            for k in expect.get(lst) or []:
                value = differs.get("stage." + k, state)
                stages.append({"key": "stage:" + k, "label": k, "kind": "stage", "value": "exclude" if value == "absent" else value})
        elements = []
        for lst, state in (("elements_include", "include"), ("elements_option", "option"), ("elements_absent", "exclude")):
            for k in expect.get(lst) or []:
                elements.append({"key": "element:" + k, "label": k, "kind": "element", "value": differs.get("element." + k, state)})
        positions = []
        for k, v in (expect.get("positions") or {}).items():
            v = differs.get("position." + k, v)
            if v != "absent":
                positions.append({"key": "position:" + k, "label": k, "kind": "position", "value": v})
        groups = [{"id": "stages", "controls": stages}, {"id": "positions", "controls": positions}]
        if elements:
            groups.append({"id": "elements", "controls": elements})
        checks = [{"id": c, "label": c, "blocking": True} for c in expect.get("blocking") or []]
        blocks = [{"id": f"b{i}", "label": "", "text": p} for i, p in enumerate(r["must_include"])]
        reports.append({
            "report": {"id": f"r{n}", "kind": "rfp", "package_id": pkg_of.get(r["discipline"], "")},
            "rfp": {"discipline": r["template"], "status": "draft", "checks": checks, "decisions": checks, "groups": groups},
            "draft": {"sections": [{"id": "scope", "title": "Scope", "blocks": blocks}]},
        })
    return {"packages": packages, "cost_plan": cost_plan, "program": program, "reports": reports}


def find(items, **want):
    for item in items:
        if all(item.get(k) == v for k, v in want.items()):
            return item
    raise AssertionError(f"no entry with {want} in {items}")


class PerfectAppTests(unittest.TestCase):
    def test_perfect_app_has_no_failures(self):
        sh = shape()
        result = s.score(sh, app_from_shape(sh))
        for area in ("procurement", "cost_plan"):
            self.assertEqual(result[area]["missed"], [], area)
            self.assertEqual(result[area]["false_positive"], [], area)
        self.assertEqual(result["cost_plan"]["fabricated"], [])
        prog = result["program"]
        self.assertEqual(prog["activities"]["missed"], [])
        self.assertEqual(prog["links"]["violated"], [])
        self.assertEqual(prog["orphans"], [])
        self.assertEqual(prog["cycles"], [])
        self.assertTrue(prog["skeleton"]["match"])
        for r in result["rfps"]:
            if r["template"]:
                self.assertTrue(r["found"], r["discipline"])
                self.assertEqual(r["expect"]["new_failure"], [], r["discipline"])
                self.assertEqual(r["must_include"]["missing"], [], r["discipline"])
                self.assertEqual(r["must_not_include"]["present"], [], r["discipline"])

    def test_app_differs_is_a_known_difference_not_a_failure(self):
        sh = shape()
        result = s.score(sh, app_from_shape(sh))
        arch = find(result["rfps"], discipline="consultant.architect")
        self.assertIn("stage.bidding", [d["key"] for d in arch["expect"]["known_difference"]])
        self.assertEqual(arch["expect"]["new_failure"], [])
        self.assertIn("text.planning_approval", [d["key"] for d in arch["expect"]["not_checked"]])
        # A must_not_include on a control the app is known to price is a known difference too.
        st = find(result["rfps"], discipline="consultant.structural")
        self.assertIn("Super-flat floor", [k["phrase"] for k in st["must_not_include"]["known"]])


class CostPlanTests(unittest.TestCase):
    def setUp(self):
        self.sh = shape()
        self.app = app_from_shape(self.sh)
        self.items = self.app["cost_plan"]["items"]

    def test_consultant_fee_line_matches_by_package_discipline(self):
        result = s.score(self.sh, self.app)
        hit = find(result["cost_plan"]["matched"], expected="Structural engineer")
        self.assertEqual(hit["app"], "Structural")
        self.assertEqual(hit["matched_by"], "discipline")

    def test_missing_line_is_missed_with_its_built_in_claim(self):
        self.items[:] = [i for i in self.items if i["label"] != "Long service levy" and i["source_key"] != "pw:da-fees"]
        result = s.score(self.sh, self.app)
        missed = {m["expected"]: m["built_in"] for m in result["cost_plan"]["missed"]}
        self.assertEqual(missed, {"Long service levy": "no", "DA fees and contributions": "yes"})
        counts = result["summary"]["cost_plan"]
        self.assertEqual(counts["missed_built_in"], 1)
        self.assertEqual(counts["missed_known_gap"], 1)

    def test_unknown_line_is_extra_and_not_expected_line_is_false_positive(self):
        self.items.append({"cost_item_id": "x1", "label": "Mechanical", "code": "F09", "line_kind": "fee", "package_id": "pkg-mech",
                           "source_key": "fee:pkg-mech", "posting": True, "values": {}})
        self.items.append({"cost_item_id": "x2", "label": "Active fire protection", "code": "07", "line_kind": "works",
                           "source_key": "works:fire-active", "posting": True, "values": {}})
        self.items.append({"cost_item_id": "x3", "label": "Feature lighting allowance", "code": "", "line_kind": "project_wide",
                           "category": "other", "posting": True, "values": {}})
        self.app["packages"]["items"].append({"id": "pkg-mech", "kind": "services", "discipline_id": "consultant.mechanical",
                                              "title": "Mechanical", "lifecycle_status": "active"})
        result = s.score(self.sh, self.app)
        fps = {f["app"] for f in result["cost_plan"]["false_positive"]}
        self.assertEqual(fps, {"Mechanical", "Active fire protection"})
        self.assertEqual([e["app"] for e in result["cost_plan"]["extra"]], ["Feature lighting allowance"])

    def test_construction_line_does_not_match_a_consultant_fee_of_the_same_trade(self):
        # "Electrical services" (construction) is expected; "F04 Electrical" (consultant fee) is not.
        result = s.score(self.sh, self.app)
        self.assertEqual(result["cost_plan"]["false_positive"], [])

    def test_amount_without_a_source_document_is_a_fabrication(self):
        spread = {"amount": "120000.00", "value_state": "known", "origin": "assumption", "meaning": "allowance",
                  "provenance": {"actor": "u", "spread": {"budget": "1000000.00", "split": "residential_house_new"}, "budget_origin": "document"}}
        invented = {"amount": "45000.00", "value_state": "known", "origin": "assumption", "meaning": "allowance",
                    "provenance": {"actor": "u"}}
        typed = {"amount": "9000.00", "value_state": "known", "origin": "user", "meaning": "allowance", "provenance": {"actor": "u"}}
        award = {"amount": "30000.00", "value_state": "known", "origin": "user", "meaning": "stated",
                 "rationale": "Awarded to Example Pty Ltd (fee.pdf); Stage 1 $30,000 p.2", "provenance": {"actor": "u"}}
        unknown = {"amount": None, "value_state": "unknown", "origin": "assumption", "meaning": "allowance", "provenance": {}}
        find(self.items, code="02")["values"] = {"budget": spread}
        find(self.items, code="03")["values"] = {"budget": invented}
        find(self.items, code="04")["values"] = {"budget": typed}
        find(self.items, label="Architect")["values"] = {"commitment": award, "budget": unknown}
        result = s.score(self.sh, self.app)
        cp = result["cost_plan"]
        self.assertEqual([(f["app"], f["metric"]) for f in cp["fabricated"]], [("Structure", "budget")])
        self.assertEqual(cp["amounts"], {"document": 2, "user": 1, "fabricated": 1})

    def test_pending_suggestions_count_as_app_lines(self):
        line = self.items.pop()  # Owner supply items
        line = dict(line, source_key="brief:owner-supply")
        self.app["cost_plan"]["suggestions"].append(line)
        result = s.score(self.sh, self.app)
        hit = find(result["cost_plan"]["matched"], expected=line["label"])
        self.assertEqual(hit["state"], "suggestion")


class ProcurementTests(unittest.TestCase):
    def test_suggestions_match_by_title_and_over_suggestions_are_false_positives(self):
        sh = shape()
        app = app_from_shape(sh)
        pk = app["packages"]
        pk["items"] = [p for p in pk["items"] if p["discipline_id"] not in ("consultant.surveyor", "consultant.basix")]
        pk["suggestions"] = [{"key": "default:services:Surveyor", "kind": "services", "title": "Surveyor"},
                             {"key": "default:services:Mechanical", "kind": "services", "title": "Mechanical"},
                             {"key": "default:services:Acoustic", "kind": "services", "title": "Acoustic"}]
        result = s.score(sh, app)
        pr = result["procurement"]
        self.assertEqual(find(pr["matched"], expected="consultant.surveyor")["state"], "suggested")
        self.assertEqual([m["expected"] for m in pr["missed"]], ["consultant.basix"])
        self.assertEqual([f["discipline"] for f in pr["false_positive"]], ["consultant.mechanical"])
        self.assertEqual([e["discipline"] for e in pr["extra"]], ["consultant.acoustic"])
        self.assertEqual(result["summary"]["procurement"]["missed_known_gap"], 1)

    def test_discipline_aliases_count_as_a_match(self):
        sh = shape()
        app = app_from_shape(sh)  # the app holds a consultant.basix package
        basix = next(p for p in sh["procurement"] if p["discipline"] == "consultant.basix")
        basix.update(discipline="consultant.esd", aliases=["consultant.basix"])  # CONVENTIONS.md C2
        sh["not_expected_procurement"] = [{"discipline": "consultant.acoustic", "aliases": ["consultant.mechanical"], "why": "x"}]
        app["packages"]["suggestions"] = [{"key": "default:services:Mechanical", "kind": "services", "title": "Mechanical"}]
        result = s.score(sh, app)
        pr = result["procurement"]
        hit = find(pr["matched"], expected="consultant.esd")
        self.assertEqual((hit["app"], hit["matched_by"]), ("BASIX", "alias"))
        self.assertEqual(pr["missed"], [])
        self.assertEqual([f["discipline"] for f in pr["false_positive"]], ["consultant.mechanical"])

    def test_head_contract_proposal_counts_as_main_works(self):
        sh = shape()
        app = app_from_shape(sh)
        pk = app["packages"]
        pk["items"] = [p for p in pk["items"] if p["discipline_id"] != "trade.main_works"]
        pk["proposed_works"] = {"key": "head", "title": "Head contract", "lines": []}
        result = s.score(sh, app)
        self.assertEqual(find(result["procurement"]["matched"], expected="trade.main_works")["state"], "proposed_works")


class ProgramTests(unittest.TestCase):
    def setUp(self):
        self.sh = shape()
        self.app = app_from_shape(self.sh)
        self.prog = self.app["program"]

    def act(self, label):
        return find(self.prog["activities"], label=label)

    def test_fragment_matches_when_the_app_labels_differently(self):
        self.act("Neighbour notification")["label"] = "Public exhibition"
        result = s.score(self.sh, self.app)
        hit = find(result["program"]["activities"]["matched"], expected="Neighbour notification")
        self.assertEqual((hit["app"], hit["matched_by"]), ("Public exhibition", "fragment"))

    def test_activity_alias_matches(self):
        a = next(x for x in self.sh["program"]["activities"] if x["label"] == "Neighbour notification")
        a["aliases"] = ["Notify the neighbours"]
        row = self.act("Neighbour notification")
        row.update(label="Notify the neighbours", fragment_id=None)
        result = s.score(self.sh, self.app)
        hit = find(result["program"]["activities"]["matched"], expected="Neighbour notification")
        self.assertEqual(hit["matched_by"], "alias")

    def test_after_link_violated_by_computed_dates(self):
        tender = self.act("Tender period")["id"]
        self.prog["computed"]["activities"][tender]["es"] = 0
        result = s.score(self.sh, self.app)
        bad = result["program"]["links"]["violated"]
        self.assertIn(("Tender period", "Assemble the tender package"), [(v["activity"], v["after"]) for v in bad])

    def test_unknown_dates_are_not_a_pass(self):
        tender = self.act("Tender period")["id"]
        self.prog["computed"]["activities"][tender] = {"es": None, "ef": None, "unknown": True}
        result = s.score(self.sh, self.app)
        self.assertIn("Tender period", [u["activity"] for u in result["program"]["links"]["unknown"]])

    def test_orphan_cycle_wrong_stage_and_skeleton(self):
        self.prog["activities"].append({"id": "lonely", "parent_id": "st-construction", "kind": "activity",
                                        "label": "Lonely row", "fragment_id": None})
        a, b = self.act("Tender period")["id"], self.act("Evaluate tenders")["id"]
        self.prog["links"].append({"id": "back", "pred_id": b, "succ_id": a, "type": "FS"})
        self.act("Construction certificate")["parent_id"] = "st-construction_documentation"
        self.prog["skeleton"]["id"] = "sk.new-dc-da"
        result = s.score(self.sh, self.app)
        prog = result["program"]
        self.assertEqual([o["app"] for o in prog["orphans"]], ["Lonely row"])
        self.assertTrue(any(a in c and b in c for c in prog["cycles"]))
        self.assertEqual(find(prog["activities"]["wrong_stage"], expected="Construction certificate")["app_stage"], "construction_documentation")
        self.assertFalse(prog["skeleton"]["match"])
        self.assertIn("Lonely row", [e["app"] for e in prog["activities"]["extra"]])

    def test_no_program_yet(self):
        self.app["program"] = {"empty": True}
        result = s.score(self.sh, self.app)
        self.assertEqual(result["program"]["status"], "empty")
        self.assertEqual(len(result["program"]["activities"]["missed"]), len(self.sh["program"]["activities"]))


class RFPTests(unittest.TestCase):
    def setUp(self):
        self.sh = shape()
        self.app = app_from_shape(self.sh)

    def report(self, template):
        return next(r for r in self.app["reports"] if r["rfp"]["discipline"] == template)

    def control(self, template, key):
        for g in self.report(template)["rfp"]["groups"]:
            for c in g["controls"]:
                if c["key"] == key:
                    return c
        raise AssertionError(key)

    def test_stage_and_position_differences(self):
        self.control("structural", "stage:tender_traditional")["value"] = "option"   # new failure
        self.control("structural", "position:shared_model")["value"] = "O"         # app_differs says O: known
        self.control("structural", "stage:planning_approval")["value"] = "option"  # app_differs said include: now fixed
        groups = self.report("structural")["rfp"]["groups"]
        positions = next(g for g in groups if g["id"] == "positions")
        positions["controls"].append({"key": "position:retention_design", "label": "Retention", "value": "O"})  # want absent
        result = s.score(self.sh, self.app)
        st = find(result["rfps"], discipline="consultant.structural")["expect"]
        self.assertEqual(sorted(f["key"] for f in st["new_failure"]), ["position.retention_design", "stage.tender_traditional"])
        self.assertIn("position.shared_model", [d["key"] for d in st["known_difference"]])
        self.assertIn("stage.planning_approval", [d["key"] for d in st["fixed"]])

    def test_must_include_and_must_not_include(self):
        draft = self.report("geotechnical")["draft"]["sections"][0]["blocks"]
        draft[:] = [b for b in draft if "Laboratory" not in b["text"]]
        draft.append({"id": "bad", "label": "", "text": "Contamination or waste classification of excavated material."})
        result = s.score(self.sh, self.app)
        geo = find(result["rfps"], discipline="consultant.geotechnical")
        self.assertEqual(geo["must_include"]["missing"], ["Laboratory reactivity testing"])
        self.assertEqual([p["phrase"] for p in geo["must_not_include"]["present"]], ["Contamination or waste classification"])

    def test_an_excluded_scope_row_is_not_present(self):
        # The RFP prints excluded positions; a must_not_include that names one is absent while it is X.
        blocks = self.report("structural")["draft"]["sections"][0]["blocks"]
        blocks.append({"id": "scope-table", "label": "Scope", "text": "",
                       "table": {"columns": ["Item", "Position"], "rows": [["Dilapidation surveys", "X"]]}})
        result = s.score(self.sh, self.app)
        st = find(result["rfps"], discipline="consultant.structural")
        self.assertNotIn("Dilapidation surveys", [p["phrase"] for p in st["must_not_include"]["present"]])

    def test_keyed_phrase_is_decided_by_its_control_not_by_words(self):
        st = next(r for r in self.sh["rfps"] if r["discipline"] == "consultant.structural")
        st["must_include"].append({"text": "Words found nowhere in the RFP", "key": "stage.design_development"})
        st["must_include"].append({"text": "Concept and schematic design", "key": "stage.existing_assessment"})
        st["must_not_include"].append({"text": "Dilapidation surveys", "key": "position.dilapidation"})
        self.control("structural", "position:dilapidation")["value"] = "O"
        result = s.score(self.sh, self.app)
        got = find(result["rfps"], discipline="consultant.structural")
        self.assertEqual(got["must_include"]["missing"], ["Concept and schematic design"])  # that stage is excluded
        self.assertEqual([(p["phrase"], p["where"]) for p in got["must_not_include"]["present"]],
                         [("Dilapidation surveys", "key:position.dilapidation")])

    def test_shared_template_pairs_by_discipline_first(self):
        # consultant.bca and consultant.certifier share bca-certification (CONVENTIONS.md C3).
        self.sh["rfps"].append({"discipline": "consultant.bca", "template": "bca-certification", "built_in": "partial",
                                "must_include": ["BCA advisory"], "must_not_include": []})
        self.app["packages"]["items"].append({"id": "pkg-bca", "kind": "services", "discipline_id": "consultant.bca",
                                              "title": "BCA", "lifecycle_status": "active"})
        cert = self.report("bca-certification")
        bca = copy.deepcopy(cert)
        bca["report"].update(id="r-bca", package_id="pkg-bca")
        self.app["reports"].insert(0, bca)  # the BCA report comes first
        result = s.score(self.sh, self.app)
        self.assertEqual(find(result["rfps"], discipline="consultant.certifier")["report"], cert["report"]["id"])
        self.assertEqual(find(result["rfps"], discipline="consultant.bca")["report"], "r-bca")

    def test_rfp_discipline_alias_beats_template(self):
        cert = next(r for r in self.sh["rfps"] if r["discipline"] == "consultant.certifier")
        # A wrong template would claim the landscape report; the alias must win first.
        cert.update(discipline="consultant.bca", aliases=["consultant.certifier"], template="landscape-architecture")
        result = s.score(self.sh, self.app)
        got = find(result["rfps"], discipline="consultant.bca")
        self.assertEqual(got["report"], self.report("bca-certification")["report"]["id"])
        self.assertTrue(find(result["rfps"], discipline="consultant.landscape")["found"])

    def test_rfp_without_template_is_reported_missing(self):
        result = s.score(self.sh, self.app)
        basix = find(result["rfps"], discipline="consultant.basix")
        self.assertFalse(basix["found"])
        self.assertEqual(basix["built_in"], "no")


class TriggerTests(unittest.TestCase):
    def test_cdc_variant_adds_and_removes(self):
        sh = shape()
        app = app_from_shape(sh)  # the base app: still DA, still a town planner
        result = s.score(sh, app, triggers=["cdc-pathway"])
        self.assertIn("consultant.town_planner", [f["discipline"] for f in result["procurement"]["false_positive"]])
        self.assertIn("Lodge the DA", [f["expected"] for f in result["program"]["activities"]["false_positive"]])
        self.assertIn("Prepare and lodge the complying development certificate",
                      [m["expected"] for m in result["program"]["activities"]["missed"]])
        self.assertIn("CDC fee", [m["expected"] for m in result["cost_plan"]["missed"]])
        self.assertIn("DA fees and contributions", [f["expected"] for f in result["cost_plan"]["false_positive"]])
        planner = find(result["rfps"], discipline="consultant.town_planner")
        self.assertEqual(planner["status"], "not_expected")
        certifier = find(result["rfps"], discipline="consultant.certifier")
        self.assertIn("CDC assessment and determination", certifier["must_include"]["missing"])
        self.assertEqual(result["scenario"]["triggers"], ["cdc-pathway"])

    def test_unknown_trigger_is_an_error(self):
        with self.assertRaises(ValueError):
            s.score(shape(), app_from_shape(shape()), triggers=["moat"])


class OutputTests(unittest.TestCase):
    def test_summary_names_every_category_and_no_blended_score(self):
        sh = shape()
        result = s.score(sh, app_from_shape(sh))
        text = s.summary(result)
        for word in ["Procurement", "Cost plan", "fabricated", "Program", "links", "orphans", "cycles", "RFPs", "must_include"]:
            self.assertIn(word, text)
        self.assertNotIn("score", result["summary"])
        self.assertNotIn("total", result["summary"])

    def test_cli_reads_a_saved_snapshot_folder(self):
        sh = shape()
        app = app_from_shape(sh)
        with tempfile.TemporaryDirectory() as d:
            snap = Path(d) / "snapshot"
            (snap / "reports").mkdir(parents=True)
            (snap / "packages.json").write_text(json.dumps(app["packages"]), encoding="utf8")
            (snap / "cost_plan.json").write_text(json.dumps({"plan": app["cost_plan"], "totals": {}}), encoding="utf8")
            (snap / "program.json").write_text(json.dumps(app["program"]), encoding="utf8")
            for r in app["reports"]:
                (snap / "reports" / f"{r['report']['id']}.json").write_text(json.dumps(r), encoding="utf8")
            out = Path(d) / "score.json"
            code = s.main(["--shape", str(T01), "--snapshot", str(snap), "--json", str(out)])
            self.assertEqual(code, 0)
            saved = json.loads(out.read_text(encoding="utf8"))
            self.assertEqual(saved["type_id"], "T01")
            self.assertEqual(saved["cost_plan"]["missed"], [])

    def test_area_not_captured_is_said_not_passed(self):
        sh = shape()
        app = app_from_shape(sh)
        del app["cost_plan"]
        result = s.score(sh, app)
        self.assertEqual(result["cost_plan"]["status"], "not_captured")
        self.assertEqual(len(result["cost_plan"]["missed"]), len(sh["cost_plan"]))


if __name__ == "__main__":
    unittest.main()
