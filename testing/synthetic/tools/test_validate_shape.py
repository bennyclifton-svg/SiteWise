"""Checks for validate_shape.py. Run from the repo root:

    python -m unittest discover -s testing/synthetic/tools
"""
import tempfile
import unittest
from pathlib import Path

import yaml

import validate_shape as v

GOOD = {
    "schema": "sitewise-synthetic-shape/1",
    "type_id": "T01",
    "slug": "new-house",
    "name": "New detached house",
    "reviewed": False,
    "state": "NSW",
    "profile": {
        "hdr.building_class": "residential",
        "hdr.subclass": "house",
        "hdr.work_type": "new",
        "hdr.cond.planning": "da",
        "det.ncc_class": ["1a"],
    },
    "work_items": [{"system": "site", "action": "new", "why": "new site works"}],
    "procurement": [{
        "discipline": "consultant.architect", "kind": "services", "built_in": "yes",
        "source": "knowledge/works/package_defaults.yaml baselines[0]", "why": "designs the house",
    }],
    "cost_plan": [{
        "line": "Construction contingency", "section": "contingency", "built_in": "yes",
        "source": "knowledge/costs/starter_lines.yaml construction-contingency", "why": "always",
    }],
    "program": {
        "skeleton": "sk.new-traditional-da",
        "activities": [{
            "label": "Install tree protection", "stage": "construction",
            "fragment": "fr.tree-protection", "after": [], "built_in": "yes",
            "source": "knowledge/program/fragments.yaml fr.tree-protection",
        }],
    },
    "rfps": [{
        "discipline": "consultant.structural", "template": "structural", "built_in": "yes",
        "source": "knowledge/rfp/structural.yaml", "must_include": ["footing design"], "must_not_include": [],
    }],
    "triggers": [{
        "id": "tree-near-works",
        "label": "Significant tree near the works",
        "evidence": ["the big gum near the rear boundary", "TPZ encroachment"],
        "negation": ["no trees affected"],
        "profile": ["det.bal"],
        "adds": {"procurement": ["consultant.arborist"]},
        "built_in": {"procurement": "yes"},
        "source": "knowledge/works/package_defaults.yaml",
    }],
    "gaps": [],
    "variants": ["tree-near-works"],
}


def check(shape):
    with tempfile.TemporaryDirectory() as d:
        folder = Path(d) / "T01-new-house"
        folder.mkdir()
        path = folder / "shape.yaml"
        path.write_text(yaml.safe_dump(shape), encoding="utf8")
        return v.validate(path, v.load_ids(Path(".")))


class ValidateShapeTests(unittest.TestCase):
    def test_good_shape_has_no_errors(self):
        errors, _ = check(GOOD)
        self.assertEqual(errors, [])

    def test_unknown_ids_are_errors(self):
        bad = yaml.safe_load(yaml.safe_dump(GOOD))
        bad["procurement"][0]["discipline"] = "consultant.wizard"
        bad["work_items"][0]["system"] = "site.moat"
        bad["program"]["activities"][0]["fragment"] = "fr.moat"
        bad["profile"]["hdr.subclass"] = "castle"
        bad["profile"]["det.dragon_risk"] = "high"
        errors, _ = check(bad)
        joined = "\n".join(errors)
        for word in ["consultant.wizard", "site.moat", "fr.moat", "castle", "det.dragon_risk"]:
            self.assertIn(word, joined)

    def test_built_in_claim_needs_an_existing_source(self):
        bad = yaml.safe_load(yaml.safe_dump(GOOD))
        del bad["cost_plan"][0]["source"]
        bad["rfps"][0]["source"] = "knowledge/rfp/nope.yaml"
        errors, _ = check(bad)
        self.assertTrue(any("cost_plan[0]" in e and "source" in e for e in errors))
        self.assertTrue(any("nope.yaml" in e for e in errors))

    def test_trigger_needs_evidence_negation_and_consequence(self):
        bad = yaml.safe_load(yaml.safe_dump(GOOD))
        bad["triggers"][0]["evidence"] = ["one phrase"]
        bad["triggers"][0]["negation"] = []
        bad["triggers"][0]["adds"] = {}
        bad["variants"] = ["not-a-trigger"]
        errors, _ = check(bad)
        joined = "\n".join(errors)
        for word in ["evidence", "negation", "adds", "not-a-trigger"]:
            self.assertIn(word, joined)

    def test_folder_must_match_type_id(self):
        bad = dict(GOOD, type_id="T02")
        errors, _ = check(bad)
        self.assertTrue(any("folder" in e for e in errors))


if __name__ == "__main__":
    unittest.main()
