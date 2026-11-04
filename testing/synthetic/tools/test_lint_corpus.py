"""Checks for lint_corpus.py. Run from the repo root:

    python -m unittest discover -s testing/synthetic/tools
"""
import json
import tempfile
import unittest
from pathlib import Path

import lint_corpus as lc

ABN = "51824753556"
BRIEF = """# Owner's brief

Title: Owner's brief
Date: 3 February 2026

## The house

Construction budget: $1,450,000 excluding GST. Single slab on ground; no basement.
Lot 7 DP 1288841. Contact jo@example.com. Architect ABN 51 824 753 556.
"""


class Lint(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name) / "ST01-base"
        (self.dir / "corpus").mkdir(parents=True)
        self.manifest = {
            "base": None,
            "invented": {"lots": ["Lot 7 DP 1288841"], "abns": [ABN], "emails": ["jo@example.com"]},
            "documents": [{"id": "d01", "filename": "Owner brief.md", "carries": [
                {"key": "proj.budget_stated", "says": "Construction budget: $1,450,000 excluding GST"}]}],
        }

    def tearDown(self):
        self.tmp.cleanup()

    def run_lint(self, text=BRIEF, manifest=None):
        (self.dir / "manifest.json").write_text(json.dumps(manifest or self.manifest), encoding="utf8")
        (self.dir / "corpus" / "Owner brief.md").write_text(text, encoding="utf8")
        return lc.lint(self.dir)

    def assertError(self, errors, fragment):
        self.assertTrue(any(fragment in e for e in errors), f"no error containing {fragment!r} in {errors}")

    def test_clean_passes(self):
        self.assertEqual(self.run_lint(), [])

    def test_missing_and_extra_files(self):
        (self.dir / "corpus" / "Stray.md").write_text("x", encoding="utf8")
        self.manifest["documents"].append({"id": "d02", "filename": "Survey.md", "carries": []})
        errs = self.run_lint()
        self.assertError(errs, "Stray.md")
        self.assertError(errs, "Survey.md")

    def test_carries_must_be_stated(self):
        self.assertError(self.run_lint(BRIEF.replace("1,450,000", "1,500,000")), "proj.budget_stated")

    def test_carries_ignores_line_breaks_and_emphasis(self):
        self.assertEqual(self.run_lint(BRIEF.replace("budget: $1,450,000", "budget:\n**$1,450,000**")), [])

    def test_leaked_id_and_word(self):
        errs = self.run_lint(BRIEF + "\nflag.basement is no. This synthetic brief.\n")
        self.assertError(errs, "flag.basement")
        self.assertError(errs, "synthetic")

    def test_real_email(self):
        self.assertError(self.run_lint(BRIEF + "\njo@acme.com.au\n"), "acme.com.au")

    def test_undeclared_valid_abn(self):
        self.manifest["invented"]["abns"] = []
        self.assertError(self.run_lint(), "51824753556")

    def test_undeclared_labelled_abn_even_if_invalid(self):
        self.assertError(self.run_lint(BRIEF + "\nABN: 12 345 678 901\n"), "12345678901")

    def test_undeclared_dp(self):
        self.assertError(self.run_lint(BRIEF + "\nLot 3 DP 777777\n"), "777777")

    def test_variant_inherits_base_inventions(self):
        base = Path(self.tmp.name) / "ST01-base"
        (base / "manifest.json").write_text(json.dumps(self.manifest), encoding="utf8")
        var = Path(self.tmp.name) / "ST01-x"
        (var / "corpus").mkdir(parents=True)
        m = dict(self.manifest, base="ST01-base", invented={})
        (var / "manifest.json").write_text(json.dumps(m), encoding="utf8")
        (var / "corpus" / "Owner brief.md").write_text(BRIEF, encoding="utf8")
        self.assertEqual(lc.lint(var), [])


if __name__ == "__main__":
    unittest.main()
