"""Checks for score.py. From the repo root:

    python -m unittest discover -s testing/synthetic/harness
"""
import unittest

import score as s


class Rules(unittest.TestCase):
    def test_outcomes(self):
        eq = lambda a, b: s.norm(a) == s.norm(b)
        self.assertEqual(s.judge("Report", "report", eq), "correct")
        self.assertEqual(s.judge("Report", "Drawing", eq), "wrong")
        self.assertEqual(s.judge("Report", "", eq), "missed")
        self.assertEqual(s.judge("Report", "", eq, by_design=True), "missed_by_design")
        self.assertEqual(s.judge(None, "", eq), "correct_blank")
        self.assertEqual(s.judge(None, "A", eq), "hallucinated")
        self.assertEqual(s.judge({"accept": ["consultant", "cost"]}, "cost", eq), "correct")

    def test_dates_and_money(self):
        self.assertEqual(s.iso_date("2 March 2026"), "2026-03-02")
        self.assertEqual(s.iso_date("2026-03-02"), "2026-03-02")
        self.assertEqual(s.money_cents("$1,450,000 excluding GST"), 145000000)
        self.assertEqual(s.money_cents("$1.45 million"), 145000000)
        self.assertIsNone(s.money_cents("not stated"))

    def test_profile_values(self):
        self.assertEqual(s.app_key("flag.basement"), "proj.flag.basement")
        self.assertEqual(s.app_key("det.state"), "det.state")
        self.assertEqual(s.app_key("proj.lot"), "site.lot")
        self.assertTrue(s._profile_same("det.flood_hazard")(False, "stated_false"))
        self.assertFalse(s._profile_same("det.flood_hazard")(False, "stated_true"))
        self.assertTrue(s._profile_same("proj.budget_stated")(145000000, "$1,450,000 excluding GST"))
        self.assertTrue(s._profile_same("det.ncc_class")(["1a", "10a"], "10a, 1a"))
        self.assertFalse(s._profile_same("det.ncc_class")(["1a", "10a"], "1a"))
        self.assertTrue(s._profile_same("hdr.scale.storeys")(2, "2"))

    def test_state_flags(self):
        self.assertEqual(s.state_flags({"invoice_date": "2026-05-29"}, {"period": "2026-10-01", "date": "2026-05-29"}),
                         {"unmapped", "no_supplier", "period"})
        self.assertEqual(s.state_flags({"invoice_date": "2026-05-29"}, {"period": "2026-10-01", "date": ""}),
                         {"unmapped", "no_supplier"})
        self.assertEqual(s.state_flags({"invoice_date": "2026-10-03"}, {"period": "2026-10-01"}),
                         {"unmapped", "no_supplier"})

    def test_profile_conflict(self):
        profile = {"header": [{"key": "hdr.cond.planning", "value": "", "alternatives": [{"value": "da"}, {"value": "cdc"}]}]}
        rows = s.score_profile({"values": {"hdr.cond.planning": {"value": "da", "read_by_design": True}}}, profile)
        self.assertEqual(rows[0]["outcome"], "conflict_true")


if __name__ == "__main__":
    unittest.main()
