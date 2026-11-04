"""Score a harness snapshot against a scenario's expected/ keys.

Covers the fields score_shapes.py does not: the document register, the
profile, invoices and fees. Every comparison gets one outcome:

- correct           the app's value is right (amber counts, Provisional Q3)
- correct_blank     the key expects blank and the app left it blank
- wrong             a different value
- missed            the app left blank what the key expects
- missed_by_design  missed, and the key says no read document states it
- hallucinated      the app filled what the key expects blank
- conflict_true     the app shows a conflict that includes the true value
- conflict_false    the app shows a conflict without the true value
- not_captured      the snapshot does not hold the item at all

There is no blended score; counts are reported per area and per field.

    python testing/synthetic/harness/score.py \
        --scenario testing/synthetic/scenarios/ST01-base \
        --snapshot testing/synthetic/runs/<run>/ST01-base/snapshot [--json out.json]
"""
import argparse
import json
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime
from pathlib import Path

DATE_FORMATS = ["%Y-%m-%d", "%d %B %Y", "%d %b %Y", "%d/%m/%Y", "%d.%m.%Y", "%d-%m-%Y"]
REGISTER_FIELDS = ["kind", "discipline", "lifecycle", "number", "revision", "title", "date"]


def _json(path):
    return json.loads(Path(path).read_text(encoding="utf8"))


def accepts(value):
    if isinstance(value, dict) and "accept" in value:
        return list(value["accept"])
    return [value]


def iso_date(text):
    text = str(text).strip()
    for f in DATE_FORMATS:
        try:
            return datetime.strptime(text, f).strftime("%Y-%m-%d")
        except ValueError:
            pass
    return text


def money_cents(text):
    """'$1,450,000 excluding GST' -> 145000000. None when no single amount."""
    amounts = re.findall(r"\$?\s*(\d[\d,]*(?:\.\d{1,2})?)", str(text))
    if not amounts:
        return None
    value = float(amounts[0].replace(",", ""))
    if re.search(r"\b(m|million)\b", str(text), re.I) and value < 1000:
        value *= 1_000_000
    return round(value * 100)


def norm(value):
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, list):
        return ",".join(sorted(norm(v) for v in value))
    text = re.sub(r"\s+", " ", str(value)).strip().lower()
    return text


def judge(expected, actual, same, by_design=False):
    """One outcome from an expected value (None = blank) and the app's value ('' = blank)."""
    want_blank = all(v is None for v in accepts(expected))
    got_blank = actual in (None, "", [])
    if want_blank:
        return "correct_blank" if got_blank else "hallucinated"
    if got_blank:
        return "missed_by_design" if by_design else "missed"
    return "correct" if any(v is not None and same(v, actual) for v in accepts(expected)) else "wrong"


# ---- register ---------------------------------------------------------------

def _register_same(field):
    if field == "date":
        return lambda e, a: iso_date(e) == iso_date(a)
    if field in ("number", "revision"):
        return lambda e, a: norm(e).replace(" ", "") == norm(a).replace(" ", "")
    return lambda e, a: norm(e) == norm(a)


def score_register(expected, manifest, details):
    by_name = {d["filename"]: d for d in details}
    names = {d["id"]: d["filename"] for d in manifest["documents"]}
    tiers = {d["id"]: d.get("tier") for d in manifest["documents"]}
    rows = []
    for e in expected.get("documents", []):
        app = by_name.get(names.get(e["doc"]))
        if app is None:
            rows.append({"doc": e["doc"], "field": "status", "outcome": "not_captured"})
            continue
        fields = {f["field"]: f for f in app.get("fields", [])}
        st = "correct" if app.get("status") == e.get("status") else "wrong"
        rows.append({"doc": e["doc"], "field": "status", "expected": e.get("status"), "app": app.get("status"), "outcome": st})
        for field in REGISTER_FIELDS:
            f = fields.get(field, {})
            value = f.get("value", "")
            # Jev's "Not established" discipline is stored as a blank with a band.
            if field == "discipline" and value == "" and f.get("band") in ("green", "amber"):
                value = "none"
            rows.append({"doc": e["doc"], "tier": tiers.get(e["doc"]), "field": field, "expected": e.get(field), "app": value,
                         "band": f.get("band"), "by": f.get("decided_by"), "confidence": f.get("confidence"),
                         "outcome": judge(e.get(field), value, _register_same(field))})
        if e.get("supersedes") is not None:
            rows.append({"doc": e["doc"], "field": "supersedes", "expected": e["supersedes"],
                         "app": app.get("supersedes") or app.get("prior_id"), "outcome": "missed"
                         if not (app.get("supersedes") or app.get("prior_id")) else "correct"})
    return rows


# ---- profile ----------------------------------------------------------------

def profile_cells(profile):
    cells = {}
    for section in ("header", "facts", "details", "site_details"):
        for f in profile.get(section, []):
            cells.setdefault(f["key"], f)
    for group in profile.get("compliance", []):
        for f in group.get("rows", []):
            cells.setdefault(f["key"], f)
    return cells


SITE_KEYS = {"proj.address": "site.address", "proj.lot": "site.lot"}


def app_key(key):
    """Answer-key names to the app's: flag.x is proj.flag.x; the address and
    lot are site readings."""
    if key.startswith("flag."):
        return "proj." + key
    return SITE_KEYS.get(key, key)


def _profile_same(key):
    def same(expected, actual):
        if isinstance(expected, bool):
            return norm(actual) in ({"stated_true", "true", "yes"} if expected else {"stated_false", "false", "no"})
        if isinstance(expected, int) and key.endswith(("_stated", "estimate")):
            return money_cents(actual) == expected
        if isinstance(expected, list):
            return set(norm(expected).split(",")) == {x.strip() for x in norm(actual).split(",") if x.strip()}
        if isinstance(expected, (int, float)):
            try:
                return float(str(actual).replace(",", "")) == float(expected)
            except ValueError:
                return False
        if key.startswith("proj.target") or key.endswith("_date"):
            return iso_date(expected) == iso_date(actual)
        return norm(expected) == norm(actual)
    return same


def score_profile(expected, profile):
    cells = profile_cells(profile)
    rows = []
    for key, e in expected.get("values", {}).items():
        cell = cells.get(app_key(key))
        if cell is None:
            rows.append({"key": key, "expected": e.get("value"), "outcome": "not_captured"})
            continue
        value = cell.get("value", "")
        alts = [a.get("value") for a in (cell.get("alternatives") or []) if isinstance(a, dict)]
        same = _profile_same(key)
        if len(alts) > 1:
            outcome = "conflict_true" if any(same(v, a) for v in accepts(e.get("value")) for a in alts) else "conflict_false"
        else:
            outcome = judge(e.get("value"), value, same, by_design=not e.get("read_by_design", True))
        rows.append({"key": key, "expected": e.get("value"), "app": value, "band": cell.get("band"),
                     "alternatives": alts or None, "read_by_design": e.get("read_by_design"), "outcome": outcome})
    return rows


# ---- invoices and fees --------------------------------------------------------

def _find(items, filename):
    for item in items:
        if item.get("filename") == filename or item.get("file_name") == filename:
            return item
    return None


def state_flags(expected, app):
    """Flags the ledger raises from the project's state, not the document.

    The keys list only what the document itself causes. These follow from
    what the harness does (internal/invoices/ledger.go Check): it never
    allocates an invoice to a cost line (unmapped) or confirms a supplier
    (no_supplier), and `period` fires when the invoice is dated in a month
    before the upload month, which a run made later than the invoice date
    always is."""
    out = {"unmapped", "no_supplier"}
    # Period needs a date the app read; a missed date is scored once, as a date.
    read = ((app.get("fields") or {}).get("date") or {}).get("value") or app.get("date")
    date = str(expected.get("invoice_date") or "") if read else ""
    period = str(app.get("period") or "")
    if date and period and date[:7] < period[:7]:
        out.add("period")
    return out


def score_invoices(expected, manifest, invoices):
    names = {d["id"]: d["filename"] for d in manifest["documents"]}
    rows = []
    items = invoices.get("invoices", []) if isinstance(invoices, dict) else invoices or []
    for e in expected.get("invoices", []):
        app = _find(items, names.get(e["doc"]))
        if app is None:
            rows.append({"doc": e["doc"], "field": "invoice", "outcome": "not_captured"})
            continue
        fields = {k: (v or {}).get("value", "") for k, v in (app.get("fields") or {}).items()}
        cents = lambda x, y: str(y).strip() != "" and int(x) == int(str(y).replace(",", ""))
        checks = {
            "kind": (e.get("kind"), fields.get("kind", app.get("kind")), lambda x, y: norm(x) == norm(y)),
            "number": (e.get("number"), fields.get("number", app.get("number")), lambda x, y: norm(x) == norm(y)),
            "invoice_date": (e.get("invoice_date"), fields.get("date", app.get("date")), lambda x, y: iso_date(x) == iso_date(str(y)[:10])),
            "issuer_abn": (e.get("issuer_abn"), fields.get("issuer_abn", app.get("abn")), lambda x, y: re.sub(r"\D", "", str(x)) == re.sub(r"\D", "", str(y))),
            "ex_gst_cents": (e.get("ex_gst_cents"), fields.get("ex_gst"), cents),
            "gst_cents": (e.get("gst_cents"), fields.get("gst"), cents),
            "total_cents": (e.get("total_cents"), fields.get("total"), cents),
            "supplier": (e.get("supplier_discipline"), app.get("supplier") or fields.get("supplier"), lambda x, y: True),
        }
        for field, (want, got, same) in checks.items():
            rows.append({"doc": e["doc"], "field": field, "expected": want, "app": got,
                         "outcome": judge(want, got, same)})
        flags = {f.get("code") if isinstance(f, dict) else f for f in app.get("flags") or []}
        wanted = set(e.get("flags", [])) | state_flags(e, app)
        for flag in sorted(wanted):
            rows.append({"doc": e["doc"], "field": f"flag.{flag}", "expected": True, "app": flag in flags,
                         "outcome": "correct" if flag in flags else "missed"})
        for flag in sorted(flags - wanted):
            rows.append({"doc": e["doc"], "field": f"flag.{flag}", "expected": False, "app": True, "outcome": "hallucinated"})
    return rows


def score_fees(expected, manifest, board, prices):
    names = {d["id"]: d["filename"] for d in manifest["documents"]}
    cards = [c for cs in board.get("packages", {}).values() for c in cs] + board.get("unassigned", [])
    rows = []
    for e in expected.get("submissions", []):
        card = _find(cards, names.get(e["doc"]))
        if card is None:
            rows.append({"doc": e["doc"], "field": "submission", "outcome": "not_captured"})
            continue
        rows.append({"doc": e["doc"], "field": "stated_total_cents", "expected": e.get("stated_total_cents"),
                     "app": card.get("stated_total"), "outcome": judge(
                         e.get("stated_total_cents"), card.get("stated_total"),
                         lambda x, y: round(float(y) * 100) == int(x) if isinstance(y, (int, float, str)) and str(y) else False)})
        rows.append({"doc": e["doc"], "field": "firm_name", "expected": e.get("firm", {}).get("name"),
                     "app": card.get("firm_name"), "outcome": judge(e.get("firm", {}).get("name"), card.get("firm_name"),
                                                                    lambda x, y: norm(x) == norm(y))})
    return rows


# ---- report -----------------------------------------------------------------

def summarise(rows, by):
    out = defaultdict(Counter)
    for r in rows:
        out[r[by]][r["outcome"]] += 1
    return {k: dict(v) for k, v in out.items()}


def score(scenario, snapshot):
    scenario, snapshot = Path(scenario), Path(snapshot)
    manifest = _json(scenario / "manifest.json")
    exp = lambda n: _json(scenario / "expected" / f"{n}.json")
    snap = lambda n, default=None: _json(snapshot / n) if (snapshot / n).exists() else default
    result = {
        "schema": "sitewise-synthetic-field-score/1",
        "scenario": manifest["scenario"],
        "register": score_register(exp("document_register"), manifest, snap("document_details.json", [])),
        "profile": score_profile(exp("profile"), snap("profile.json", {})),
        "invoices": score_invoices(exp("invoices"), manifest, snap("invoices.json", {})),
        "fees": score_fees(exp("fees"), manifest, snap("board.json", {}), snap("prices.json", {})),
    }
    result["summary"] = {
        "register_by_field": summarise(result["register"], "field"),
        "profile": dict(Counter(r["outcome"] for r in result["profile"])),
        "invoices_by_field": summarise(result["invoices"], "field"),
        "fees_by_field": summarise(result["fees"], "field"),
    }
    return result


def text_summary(result):
    lines = [f"{result['scenario']}"]
    s = result["summary"]
    lines.append("  Register (per field):")
    for field, c in s["register_by_field"].items():
        lines.append(f"    {field:<11} " + ", ".join(f"{k} {v}" for k, v in sorted(c.items())))
    lines.append("  Profile: " + ", ".join(f"{k} {v}" for k, v in sorted(s["profile"].items())))
    bad = [r for r in result["profile"] if r["outcome"] not in ("correct", "correct_blank")]
    for r in bad:
        lines.append(f"    {r['outcome']:<16} {r['key']}: expected {r.get('expected')!r}, app {r.get('app')!r}")
    for area in ("invoices", "fees"):
        lines.append(f"  {area.capitalize()} (per field):")
        for field, c in s[f"{area}_by_field"].items():
            lines.append(f"    {field:<18} " + ", ".join(f"{k} {v}" for k, v in sorted(c.items())))
    return "\n".join(lines)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--scenario", required=True)
    ap.add_argument("--snapshot", required=True)
    ap.add_argument("--json")
    args = ap.parse_args(argv)
    result = score(args.scenario, args.snapshot)
    if args.json:
        Path(args.json).write_text(json.dumps(result, indent=2, ensure_ascii=False), encoding="utf8")
    print(text_summary(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
