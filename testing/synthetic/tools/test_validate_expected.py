"""Checks for validate_expected.py. Run from the repo root:

    python -m unittest discover -s testing/synthetic/tools
"""
import copy
import json
import tempfile
import unittest
from pathlib import Path

import yaml

import validate_expected as v

REPO = Path(__file__).resolve().parents[3]
IDS = v.load_ids(REPO)

SHAPE = {
    "type_id": "T01",
    "profile": {"hdr.work_type": "new", "hdr.cond.planning": "da", "flag.basement": "no",
                "det.ncc_class": ["1a", "10a"], "det.bushfire_prone_land": False},
    "scenario_inputs": [{"key": "proj.budget_stated"}],
    "triggers": [{"id": "cdc-pathway", "profile": ["hdr.cond.planning"]}],
}

ABN = "51824753556"  # passes the checksum (the ATO's published example)
POINTER = {"schema": "sitewise-synthetic-from-shape/1", "shape": "types/T01-new-house/shape.yaml", "triggers": []}


def doc(i, drop="repository", **kw):
    d = {"id": i, "filename": f"Document {i}.md", "title": f"Document {i}", "author_role": "owner",
         "firm": None, "date": "2026-02-03", "wave": 1, "drop": drop, "tier": "clean",
         "title_block": "label lines", "carries": [], "traps": [], "content": "x"}
    d.update(kw)
    return d


def base_files():
    manifest = {
        "schema": "sitewise-synthetic-manifest/1", "scenario": "ST01-base", "type_id": "T01",
        "base": None, "triggers": [],
        "project": {"name": "P", "address": "1 Example Rise, Kellyville NSW 2155", "lga": "The Hills Shire", "summary": "s"},
        "invented": {"lots": ["Lot 7 DP 1288841"], "abns": [ABN], "firms": ["Acme Engineers"], "emails": ["jo@example.com"]},
        "documents": [
            doc("d01", carries=[{"key": "proj.budget_stated", "says": "Budget: $1,000,000 ex GST"}]),
            doc("d02", author_role="consultant.structural", firm="Acme Engineers", wave=2),
            doc("d03", drop="invoices", author_role="consultant.structural", firm="Acme Engineers", wave=4),
        ],
        "changed_from_base": [],
    }
    reg = {"schema": "sitewise-synthetic-register/1", "documents": [
        {"doc": "d01", "status": "filed", "kind": "design_brief", "discipline": "none", "lifecycle": "brief",
         "number": None, "revision": None, "title": "Document d01", "date": "2026-02-03", "supersedes": None},
        {"doc": "d02", "status": "filed", "kind": "commercial", "discipline": "consultant.structural",
         "lifecycle": {"accept": ["consultant", "cost"]}, "number": "FP-1", "revision": None,
         "title": "Document d02", "date": "2026-02-03", "supersedes": None},
    ]}
    val = lambda value: {"value": value, "sources": ["d01"], "read_by_design": True}
    profile = {"schema": "sitewise-synthetic-profile/1", "values": {
        "hdr.work_type": val("new"), "hdr.cond.planning": val("da"), "flag.basement": val("no"),
        "det.ncc_class": val(["10a", "1a"]), "det.bushfire_prone_land": val(False),
        "proj.budget_stated": val(100000000),
    }, "conflicts": [], "derived": []}
    proc = {"schema": "sitewise-synthetic-procurement/1", "shape": "types/T01-new-house/shape.yaml",
            "triggers": [], "from_proposals": [{"doc": "d02", "discipline": "consultant.structural", "kind": "services"}],
            "route": "traditional", "novation": {}}
    firm = {"name": "Acme Engineers", "abn": ABN, "contact": "Jo", "email": "jo@example.com", "phone": "02 9000 0000", "address": "a"}
    fees = {"schema": "sitewise-synthetic-fees/1", "submissions": [{
        "doc": "d02", "kind": "fee_proposal", "discipline": "consultant.structural", "firm": firm,
        "stated_total_cents": 500000, "gst_basis": "ex", "flags": [],
        "prices": [{"label": "Design", "kind": "stage", "state": "priced", "amount_cents": 300000},
                   {"label": "Construction", "kind": "stage", "state": "priced", "amount_cents": 200000},
                   {"label": "Extra visit", "kind": "option", "state": "priced", "amount_cents": 50000}]}]}
    inv = {"schema": "sitewise-synthetic-invoices/1", "invoices": [{
        "doc": "d03", "kind": "invoice", "number": "INV-1", "invoice_date": "2026-05-29", "issuer_abn": ABN,
        "supplier_discipline": "consultant.structural", "ex_gst_cents": 100000, "gst_cents": 10000,
        "total_cents": 110000, "flags": [], "unsupported": {}}]}
    return {
        "manifest.json": manifest,
        "expected/document_register.json": reg,
        "expected/profile.json": profile,
        "expected/procurement.json": proc,
        "expected/fees.json": fees,
        "expected/invoices.json": inv,
        "expected/rfis.json": {"schema": "sitewise-synthetic-rfis/1", "supported": False, "rfis": []},
        "expected/cost_plan.json": dict(POINTER),
        "expected/program.json": dict(POINTER),
        "expected/rfps.json": dict(POINTER),
    }


def variant_files():
    f = copy.deepcopy(base_files())
    m = f["manifest.json"]
    m.update(scenario="ST01-cdc-pathway", base="ST01-base", triggers=["cdc-pathway"], invented={})
    m["documents"][0]["carries"].append({"key": "hdr.cond.planning", "says": "Complying development"})
    m["changed_from_base"] = [{"doc": "d01", "change": "changed", "why": "states the CDC"}]
    f["expected/profile.json"]["values"]["hdr.cond.planning"]["value"] = "cdc"
    for name in ("expected/procurement.json", "expected/cost_plan.json", "expected/program.json", "expected/rfps.json"):
        f[name]["triggers"] = ["cdc-pathway"]
    return f


class Case(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        t = self.root / "types" / "T01-new-house"
        t.mkdir(parents=True)
        (t / "shape.yaml").write_text(yaml.safe_dump(SHAPE), encoding="utf8")

    def tearDown(self):
        self.tmp.cleanup()

    def write(self, name, files):
        d = self.root / "scenarios" / name
        for rel, body in files.items():
            p = d / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(json.dumps(body, indent=2), encoding="utf8")
        (d / "expected" / "notes.md").write_text("notes", encoding="utf8")
        return d

    def errors(self, name, files):
        return v.validate(self.write(name, files), IDS, self.root)

    def assertError(self, errors, fragment):
        self.assertTrue(any(fragment in e for e in errors), f"no error containing {fragment!r} in {errors}")


class BaseScenario(Case):
    def test_good_base_passes(self):
        self.assertEqual(self.errors("ST01-base", base_files()), [])

    def test_missing_expected_file(self):
        f = base_files()
        del f["expected/fees.json"]
        self.assertError(self.errors("ST01-base", f), "fees.json")

    def test_register_must_cover_every_repository_drop(self):
        f = base_files()
        f["expected/document_register.json"]["documents"].pop()
        self.assertError(self.errors("ST01-base", f), "d02")

    def test_invoice_drop_is_not_in_register(self):
        f = base_files()
        f["expected/document_register.json"]["documents"].append(
            dict(f["expected/document_register.json"]["documents"][0], doc="d03"))
        self.assertError(self.errors("ST01-base", f), "d03")

    def test_unknown_kind_discipline_lifecycle(self):
        f = base_files()
        r = f["expected/document_register.json"]["documents"][0]
        r.update(kind="minutes", discipline="consultant.wizard", lifecycle="design-ish")
        errs = self.errors("ST01-base", f)
        for word in ("minutes", "consultant.wizard", "design-ish"):
            self.assertError(errs, word)

    def test_filename_leaks_answer(self):
        f = base_files()
        f["manifest.json"]["documents"][0]["filename"] = "expected brief.md"
        self.assertError(self.errors("ST01-base", f), "filename")

    def test_bad_drop_and_role(self):
        f = base_files()
        f["manifest.json"]["documents"][0].update(drop="desk", author_role="wizard")
        errs = self.errors("ST01-base", f)
        self.assertError(errs, "desk")
        self.assertError(errs, "wizard")

    def test_profile_must_cover_shape_and_inputs(self):
        f = base_files()
        del f["expected/profile.json"]["values"]["flag.basement"]
        del f["expected/profile.json"]["values"]["proj.budget_stated"]
        errs = self.errors("ST01-base", f)
        self.assertError(errs, "flag.basement")
        self.assertError(errs, "proj.budget_stated")

    def test_base_profile_must_match_shape(self):
        f = base_files()
        f["expected/profile.json"]["values"]["hdr.cond.planning"]["value"] = "cdc"
        self.assertError(self.errors("ST01-base", f), "hdr.cond.planning")

    def test_unknown_profile_key(self):
        f = base_files()
        f["expected/profile.json"]["values"]["det.colour_of_door"] = {"value": "red", "sources": ["d01"], "read_by_design": True}
        self.assertError(self.errors("ST01-base", f), "det.colour_of_door")

    def test_read_by_design_follows_source_kind(self):
        f = base_files()
        f["expected/document_register.json"]["documents"][0]["kind"] = "drawing"
        self.assertError(self.errors("ST01-base", f), "read_by_design")

    def test_fee_stages_must_sum_to_total(self):
        f = base_files()
        f["expected/fees.json"]["submissions"][0]["stated_total_cents"] = 999
        self.assertError(self.errors("ST01-base", f), "stated_total")
        f["expected/fees.json"]["submissions"][0]["flags"] = ["total_gap"]
        self.assertEqual(self.errors("ST01-base", f), [])

    def test_invoice_arithmetic_needs_flag(self):
        f = base_files()
        f["expected/invoices.json"]["invoices"][0]["total_cents"] = 110500
        self.assertError(self.errors("ST01-base", f), "arithmetic")
        f["expected/invoices.json"]["invoices"][0]["flags"] = ["arithmetic"]
        self.assertEqual(self.errors("ST01-base", f), [])

    def test_abn_must_be_declared_and_valid(self):
        f = base_files()
        f["expected/invoices.json"]["invoices"][0]["issuer_abn"] = "51824753557"
        errs = self.errors("ST01-base", f)
        self.assertError(errs, "not declared")
        self.assertError(errs, "abn_invalid")

    def test_email_must_be_example_com(self):
        f = base_files()
        f["manifest.json"]["documents"][0]["content"] = "Write to jo@acme.com.au"
        self.assertError(self.errors("ST01-base", f), "example.com")

    def test_pointer_triggers_match_manifest(self):
        f = base_files()
        f["expected/program.json"]["triggers"] = ["cdc-pathway"]
        self.assertError(self.errors("ST01-base", f), "program.json")


class Variant(Case):
    def setUp(self):
        super().setUp()
        self.write("ST01-base", base_files())

    def test_good_variant_passes(self):
        self.assertEqual(self.errors("ST01-cdc-pathway", variant_files()), [])

    def test_variant_needs_one_known_trigger(self):
        f = variant_files()
        f["manifest.json"]["triggers"] = ["pool"]
        self.assertError(self.errors("ST01-cdc-pathway", f), "pool")

    def test_profile_may_change_only_on_trigger_keys(self):
        f = variant_files()
        f["expected/profile.json"]["values"]["flag.basement"]["value"] = "yes"
        self.assertError(self.errors("ST01-cdc-pathway", f), "flag.basement")

    def test_undeclared_document_change(self):
        f = variant_files()
        f["manifest.json"]["documents"][1]["title"] = "Something else"
        self.assertError(self.errors("ST01-cdc-pathway", f), "d02 is changed")

    def test_undeclared_removal(self):
        f = variant_files()
        f["manifest.json"]["documents"].pop(1)
        f["expected/document_register.json"]["documents"].pop(1)
        f["expected/procurement.json"]["from_proposals"] = []
        f["expected/fees.json"]["submissions"] = []
        self.assertError(self.errors("ST01-cdc-pathway", f), "d02 is removed")

    def test_register_unchanged_for_unchanged_doc(self):
        f = variant_files()
        f["expected/document_register.json"]["documents"][1]["number"] = "FP-2"
        self.assertError(self.errors("ST01-cdc-pathway", f), "d02 is changed")


class Checksum(unittest.TestCase):
    def test_abn(self):
        self.assertTrue(v.abn_ok("51 824 753 556"))
        self.assertFalse(v.abn_ok("51824753557"))
        self.assertFalse(v.abn_ok("123"))


if __name__ == "__main__":
    unittest.main()
