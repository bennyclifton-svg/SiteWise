"""Validate a scenario's manifest.json and expected/ answer keys.

A scenario is one fictional project for one project type: the base (every
trigger absent) or a variant (exactly one trigger true). This checks that
its keys use the app's vocabulary, agree with the type's shape.yaml, and,
for a variant, differ from the base only where the trigger allows. That
last check is what makes a base/variant pair a usable metamorphic test.

Usage, from the repo root:

    python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST01-*

Exit code 1 on any error. The format is described in briefs/03-scenarios.md.
"""
import json
import re
import sys
from pathlib import Path

import yaml

import validate_shape

FILES = ["document_register", "profile", "procurement", "fees", "invoices", "rfis",
         "cost_plan", "program", "rfps"]
SCHEMAS = {
    "manifest": "sitewise-synthetic-manifest/1",
    "document_register": "sitewise-synthetic-register/1",
    "profile": "sitewise-synthetic-profile/1",
    "procurement": "sitewise-synthetic-procurement/1",
    "fees": "sitewise-synthetic-fees/1",
    "invoices": "sitewise-synthetic-invoices/1",
    "rfis": "sitewise-synthetic-rfis/1",
    "cost_plan": "sitewise-synthetic-from-shape/1",
    "program": "sitewise-synthetic-from-shape/1",
    "rfps": "sitewise-synthetic-from-shape/1",
}
POINTERS = ["procurement", "cost_plan", "program", "rfps"]
ROLES = {"owner", "council", "authority", "builder"}
DROPS = {"repository", "invoices"}
TIERS = {"clean", "messy", "adversarial"}
CHANGES = {"changed", "added", "removed"}
STATUSES = {"filed", "not_filed"}
REGISTER_FIELDS = ["kind", "discipline", "lifecycle", "number", "revision", "title", "date", "supersedes"]
SUBMISSION_KINDS = {"fee_proposal", "tender"}
GST_BASIS = {"ex", "inc", "unknown"}
PRICE_KINDS = {"stage", "element", "option", "reimbursable", "rate", "criterion"}
PRICE_STATES = {"priced", "not_priced", "included", "excluded"}
# internal/proposals/read.go flag constants.
FEE_FLAGS = {"rates_only", "options_ab", "gst_not_stated", "gst_inc_normalised", "total_gap",
             "no_text", "no_schedule", "response_form", "abn_invalid", "monthly_fee"}
INVOICE_KINDS = {"invoice", "credit_note", "not_invoice"}
# internal/invoices/ledger.go Check.
INVOICE_FLAGS = {"duplicate", "look_alike", "arithmetic", "gst_rate", "gst_free", "abn_invalid",
                 "abn_differs", "split_total", "period", "odd_date", "credit_note", "not_invoice",
                 "currency", "many", "amount_unknown", "unmapped", "no_supplier"}
LEAK = re.compile(r"\b(test|expected|answer|key)\b", re.I)
EMAIL = re.compile(r"[A-Za-z0-9._%+-]+@([A-Za-z0-9.-]+\.[A-Za-z]{2,})")
DATE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
DOC_ID = re.compile(r"^d\d{2,3}$")
ABN_WEIGHTS = [10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19]


def abn_ok(abn):
    digits = [int(c) for c in str(abn) if c.isdigit()]
    if len(digits) != 11 or len(re.sub(r"[\d ]", "", str(abn))) > 0:
        return False
    digits[0] -= 1
    return sum(d * w for d, w in zip(digits, ABN_WEIGHTS)) % 89 == 0


def load_ids(repo):
    ids = validate_shape.load_ids(repo)
    data = Path(repo) / "data"
    ids["kinds"] = {k["id"] for k in _json(data / "intake" / "kinds.json")["kinds"]}
    ids["lifecycles"] = {a["id"] for a in _json(data / "intake" / "lifecycle.json")["areas"]}
    ids["read_kinds"] = set(_json(data / "profile" / "reading.json")["read_kinds"])
    return ids


def _json(path):
    return json.loads(Path(path).read_text(encoding="utf8"))


def _accepts(value):
    """The values a key accepts: {"accept": [..]} or a single value."""
    if isinstance(value, dict) and "accept" in value:
        return list(value["accept"])
    return [value]


def _norm(value):
    """Compare shape (YAML) and key (JSON) values: lists ignore order, scalars compare as text."""
    if isinstance(value, list):
        return tuple(sorted(_norm(v) for v in value))
    if isinstance(value, bool):
        return "true" if value else "false"
    return str(value).strip().lower()


def _same(a, b):
    return {_norm(x) for x in _accepts(a)} == {_norm(x) for x in _accepts(b)}


def _allows(key_value, shape_value):
    return _norm(shape_value) in {_norm(x) for x in _accepts(key_value)}


class Scenario:
    def __init__(self, folder):
        self.folder = Path(folder)
        self.errors = []
        self.files = {}

    def err(self, msg):
        self.errors.append(msg)

    def load(self):
        path = self.folder / "manifest.json"
        names = [("manifest", path)] + [(n, self.folder / "expected" / f"{n}.json") for n in FILES]
        for name, p in names:
            if not p.exists():
                self.err(f"missing {p.relative_to(self.folder).as_posix()}")
                continue
            try:
                self.files[name] = _json(p)
            except ValueError as e:
                self.err(f"{p.name}: not JSON ({e})")
                continue
            if self.files[name].get("schema") != SCHEMAS[name]:
                self.err(f"{p.name}: schema must be {SCHEMAS[name]}")
        if not (self.folder / "expected" / "notes.md").exists():
            self.err("missing expected/notes.md")

    def get(self, name, default=None):
        return self.files.get(name, default if default is not None else {})


def _manifest(s, ids):
    m = s.get("manifest")
    docs = m.get("documents", [])
    seen, names = set(), set()
    roles = ids["disciplines"] | ROLES
    for d in docs:
        i = d.get("id", "?")
        if not DOC_ID.match(str(i)) or i in seen:
            s.err(f"manifest: document id {i} must be d01.. and unique")
        seen.add(i)
        name = d.get("filename", "")
        if not name.endswith(".md") or name in names:
            s.err(f"manifest {i}: filename {name!r} must end .md and be unique")
        if LEAK.search(name):
            s.err(f"manifest {i}: filename {name!r} gives the test away")
        names.add(name)
        if d.get("author_role") not in roles:
            s.err(f"manifest {i}: author_role {d.get('author_role')!r} is not a discipline id or {sorted(ROLES)}")
        drop = d.get("drop", "")
        if drop not in DROPS and not (drop.startswith("row:") and drop[4:] in ids["disciplines"]):
            s.err(f"manifest {i}: drop {drop!r} must be repository, invoices or row:<discipline>")
        if d.get("tier") not in TIERS:
            s.err(f"manifest {i}: tier {d.get('tier')!r} must be one of {sorted(TIERS)}")
        if not isinstance(d.get("wave"), int) or d["wave"] < 1:
            s.err(f"manifest {i}: wave must be a whole number from 1")
        if not DATE.match(str(d.get("date", ""))):
            s.err(f"manifest {i}: date must be YYYY-MM-DD")
        for c in d.get("carries", []):
            if not c.get("key") or not c.get("says"):
                s.err(f"manifest {i}: each carries entry needs key and says")
        for t in d.get("traps", []):
            if not all(t.get(k) for k in ("id", "what", "expect")):
                s.err(f"manifest {i}: each trap needs id, what and expect")
    for domain in EMAIL.findall(json.dumps(m)) + EMAIL.findall(json.dumps(s.get("fees"))):
        if domain.lower() != "example.com":
            s.err(f"email domain {domain} is not example.com")
    return {d.get("id"): d for d in docs}


def _register(s, ids, docs):
    entries = s.get("document_register").get("documents", [])
    by_doc = {}
    for e in entries:
        i = e.get("doc")
        if i in by_doc:
            s.err(f"document_register: {i} appears twice")
        by_doc[i] = e
        d = docs.get(i)
        if d is None:
            s.err(f"document_register: {i} is not in the manifest")
            continue
        if d.get("drop") == "invoices":
            s.err(f"document_register: {i} is dropped on invoices, so it is never in the register")
        if e.get("status") not in STATUSES:
            s.err(f"document_register {i}: status must be one of {sorted(STATUSES)}")
        for field in REGISTER_FIELDS:
            if field not in e:
                s.err(f"document_register {i}: missing {field} (use null for blank)")
        vocab = {"kind": ids["kinds"], "discipline": ids["disciplines"], "lifecycle": ids["lifecycles"]}
        for field, allowed in vocab.items():
            for value in _accepts(e.get(field)):
                if value is not None and value not in allowed:
                    s.err(f"document_register {i}: {field} {value!r} is not in the app's list")
        sup = e.get("supersedes")
        if sup is not None and sup not in docs:
            s.err(f"document_register {i}: supersedes {sup} is not in the manifest")
        if e.get("date") is not None and not all(DATE.match(str(x)) for x in _accepts(e["date"])):
            s.err(f"document_register {i}: date must be YYYY-MM-DD")
    for i, d in docs.items():
        if d.get("drop") != "invoices" and i not in by_doc:
            s.err(f"document_register: {i} is dropped on {d.get('drop')} but has no entry")
    return by_doc


def _read_by_design(sources, register, ids):
    for i in sources:
        e = register.get(i)
        if e and any(k in ids["read_kinds"] for k in _accepts(e.get("kind"))):
            return True
    return False


def _profile(s, ids, shape, docs, register, trigger_keys, base_profile):
    p = s.get("profile")
    values = p.get("values", {})
    for key, entry in values.items():
        if not validate_shape._profile_key_ok(key, ids):
            s.err(f"profile: {key} is not a profile key the app has")
        if "value" not in entry:
            s.err(f"profile {key}: needs value")
        for i in entry.get("sources", []):
            if i not in docs:
                s.err(f"profile {key}: source {i} is not in the manifest")
        want = _read_by_design(entry.get("sources", []), register, ids)
        # A key the app never asks stays unread whatever its source; not_read says why.
        if "not_read" in entry:
            if not str(entry["not_read"] or "").strip():
                s.err(f"profile {key}: not_read needs a reason")
            elif entry.get("read_by_design") is not False:
                s.err(f"profile {key}: not_read needs read_by_design false")
        elif entry.get("read_by_design") is not want:
            s.err(f"profile {key}: read_by_design should be {str(want).lower()} from its sources' kinds")
    needed = list(shape.get("profile", {})) + [x["key"] for x in shape.get("scenario_inputs", [])]
    for key in needed:
        if key not in values:
            s.err(f"profile: shape key {key} has no expected value")
    if base_profile is None:
        for key, want in shape.get("profile", {}).items():
            if key in values and not _allows(values[key].get("value"), want):
                s.err(f"profile {key}: base value {values[key].get('value')!r} differs from the shape's {want!r}")
    else:
        old = base_profile.get("values", {})
        for key in set(old) | set(values):
            if key in trigger_keys:
                continue
            a, b = old.get(key, {}).get("value"), values.get(key, {}).get("value")
            if (key in old) != (key in values) or not _same(a, b):
                s.err(f"profile {key}: differs from the base but is not one of the trigger's profile keys")
    for c in p.get("conflicts", []):
        if not validate_shape._profile_key_ok(c.get("key", ""), ids):
            s.err(f"profile conflict: {c.get('key')} is not a profile key")
        if c.get("true_value") is None or len(c.get("values", [])) < 2:
            s.err(f"profile conflict {c.get('key')}: needs two values and a true_value")


def _abn(s, where, abn, flags, declared):
    if not abn:
        return
    if abn not in declared:
        s.err(f"{where}: ABN {abn} is not declared under manifest invented.abns")
    if not abn_ok(abn) and "abn_invalid" not in flags:
        s.err(f"{where}: ABN {abn} fails the checksum, so flags must include abn_invalid")


def _fees(s, ids, docs, declared):
    for sub in s.get("fees").get("submissions", []):
        i = sub.get("doc")
        where = f"fees {i}"
        if i not in docs:
            s.err(f"{where}: not in the manifest")
        if sub.get("kind") not in SUBMISSION_KINDS:
            s.err(f"{where}: kind must be one of {sorted(SUBMISSION_KINDS)}")
        if sub.get("discipline") not in ids["disciplines"]:
            s.err(f"{where}: discipline {sub.get('discipline')!r} is not in disciplines.json")
        if sub.get("gst_basis") not in GST_BASIS:
            s.err(f"{where}: gst_basis must be one of {sorted(GST_BASIS)}")
        flags = sub.get("flags", [])
        for f in flags:
            if f not in FEE_FLAGS:
                s.err(f"{where}: flag {f!r} is not a proposal flag the app raises")
        _abn(s, where, sub.get("firm", {}).get("abn"), flags, declared)
        stages = 0
        for row in sub.get("prices", []):
            if row.get("kind") not in PRICE_KINDS or row.get("state") not in PRICE_STATES:
                s.err(f"{where}: price row {row.get('label')!r} has a bad kind or state")
            amount = row.get("amount_cents")
            if row.get("state") == "priced" and not isinstance(amount, int):
                s.err(f"{where}: priced row {row.get('label')!r} needs whole-cent amount_cents")
            if row.get("kind") == "stage" and row.get("state") == "priced" and isinstance(amount, int):
                stages += amount
        total = sub.get("stated_total_cents")
        if total is not None and total != stages and "total_gap" not in flags:
            s.err(f"{where}: priced stages sum to {stages}, not stated_total_cents {total}; flag total_gap if intended")


def _invoices(s, ids, docs, declared):
    for inv in s.get("invoices").get("invoices", []):
        i = inv.get("doc")
        where = f"invoices {i}"
        if docs.get(i, {}).get("drop") != "invoices":
            s.err(f"{where}: must be a manifest document dropped on invoices")
        if inv.get("kind") not in INVOICE_KINDS:
            s.err(f"{where}: kind must be one of {sorted(INVOICE_KINDS)}")
        flags = inv.get("flags", [])
        for f in flags:
            if f not in INVOICE_FLAGS:
                s.err(f"{where}: flag {f!r} is not an invoice flag the app raises")
        if inv.get("supplier_discipline") not in ids["disciplines"]:
            s.err(f"{where}: supplier_discipline {inv.get('supplier_discipline')!r} is not in disciplines.json")
        if not DATE.match(str(inv.get("invoice_date", ""))):
            s.err(f"{where}: invoice_date must be YYYY-MM-DD")
        ex, gst, total = inv.get("ex_gst_cents"), inv.get("gst_cents"), inv.get("total_cents")
        if all(isinstance(x, int) for x in (ex, gst, total)):
            if abs(ex + gst - total) > 1 and "arithmetic" not in flags:
                s.err(f"{where}: ex + GST != total, so flags must include arithmetic")
        _abn(s, where, inv.get("issuer_abn"), flags, declared)


def _pointers(s, triggers, shape_rel):
    for name in POINTERS:
        f = s.get(name)
        if f.get("shape") != shape_rel:
            s.err(f"{name}.json: shape must be {shape_rel}")
        if sorted(f.get("triggers", [])) != sorted(triggers):
            s.err(f"{name}.json: triggers {f.get('triggers')} differ from the manifest's {triggers}")


def _variant(s, docs, register, base):
    declared = {c.get("doc"): c for c in s.get("manifest").get("changed_from_base", [])}
    for c in declared.values():
        if c.get("change") not in CHANGES or not c.get("why"):
            s.err(f"changed_from_base {c.get('doc')}: needs change ({'/'.join(sorted(CHANGES))}) and why")
    old_docs = {d["id"]: d for d in base.get("manifest").get("documents", [])}
    old_reg = {e["doc"]: e for e in base.get("document_register").get("documents", [])}
    for i in set(old_docs) | set(docs):
        if i in old_docs and i in docs:
            if old_docs[i] == docs[i] and old_reg.get(i) == register.get(i):
                continue
            kind = "changed"
        else:
            kind = "added" if i in docs else "removed"
        if declared.get(i, {}).get("change") != kind:
            s.err(f"variant: {i} is {kind} from the base but changed_from_base does not say so")


def validate(folder, ids, synth_root):
    s = Scenario(folder)
    s.load()
    m = s.get("manifest")
    type_id = m.get("type_id", "")
    shapes = sorted(Path(synth_root).glob(f"types/{type_id}-*/shape.yaml")) if type_id else []
    if not shapes:
        s.err(f"manifest: no shape.yaml for type {type_id!r}")
        return s.errors
    shape = yaml.safe_load(shapes[0].read_text(encoding="utf8"))
    shape_rel = shapes[0].relative_to(synth_root).as_posix()
    triggers = m.get("triggers", [])
    trigger_ids = {t["id"]: t for t in shape.get("triggers", [])}

    base = None
    declared_abns = set(m.get("invented", {}).get("abns", []))
    if m.get("base") is None:
        if triggers:
            s.err("manifest: a base scenario has no triggers")
    else:
        if len(triggers) != 1 or triggers[0] not in trigger_ids:
            s.err(f"manifest: a variant needs exactly one trigger id from the shape, got {triggers}")
        base_dir = Path(folder).parent / m["base"]
        base = Scenario(base_dir)
        base.load()
        if base.errors:
            s.err(f"manifest: base {m['base']} is missing or invalid")
            base = None
        else:
            declared_abns |= set(base.get("manifest").get("invented", {}).get("abns", []))

    docs = _manifest(s, ids)
    register = _register(s, ids, docs)
    trigger_keys = set()
    for t in triggers:
        trigger_keys |= set(trigger_ids.get(t, {}).get("profile", []))
    _profile(s, ids, shape, docs, register, trigger_keys, base.get("profile") if base else None)
    _fees(s, ids, docs, declared_abns)
    _invoices(s, ids, docs, declared_abns)
    _pointers(s, triggers, shape_rel)
    for p in s.get("procurement").get("from_proposals", []):
        if p.get("doc") not in docs or p.get("discipline") not in ids["disciplines"]:
            s.err(f"procurement from_proposals {p.get('doc')}: unknown document or discipline")
    if base is not None:
        _variant(s, docs, register, base)
    return s.errors


def main(argv):
    if not argv or argv[0] in ("-h", "--help"):
        print(__doc__)
        return 0 if argv else 2
    here = Path(__file__).resolve()
    ids = load_ids(here.parents[3])
    synth_root = here.parents[1]
    failed = False
    for name in argv:
        if not Path(name).is_dir():
            continue
        errors = validate(name, ids, synth_root)
        print(f"{'FAIL' if errors else 'ok'}  {name}")
        for e in errors:
            print(f"  error: {e}")
        failed = failed or bool(errors)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
