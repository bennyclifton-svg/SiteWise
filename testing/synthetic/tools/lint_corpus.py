"""Lint a scenario's rendered corpus against its manifest.

The corpus is what the app will read, so it must hold what the manifest
promises and nothing that gives the answers away or looks like real data
(the repository is public). Checks, per scenario folder:

- every manifest document has a file in corpus/, and there are no others;
- each file is UTF-8 Markdown with no NUL bytes;
- each `carries[].says` wording appears in its document (whitespace and
  case ignored), so the documents state what the keys assume;
- no answer-key leakage: app ids such as `flag.basement` or
  `consultant.civil`, or words like "synthetic" or "answer key";
- every email is @example.com;
- every 11-digit number that passes the ABN checksum, and every number
  labelled ABN, is declared in the manifest's invented.abns;
- every lot/DP or DP number is declared in invented.lots.

Usage, from the repo root:

    python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST01-*

Exit code 1 on any error.
"""
import json
import re
import sys
from pathlib import Path

from validate_expected import abn_ok

LEAK_IDS = re.compile(r"\b(?:hdr|det|flag|proj|fact|consultant|trade|supplier)\.[a-z_]{3,}\b")
LEAK_WORDS = re.compile(r"\b(synthetic|answer key|expected value|test scenario|test corpus|metamorphic|sitewise)\b", re.I)
EMAIL = re.compile(r"[A-Za-z0-9._%+-]+@([A-Za-z0-9.-]+\.[A-Za-z]{2,})")
ELEVEN = re.compile(r"(?<![\d.])(\d{2}[ ]?\d{3}[ ]?\d{3}[ ]?\d{3})(?![\d.])")
ABN_LABEL = re.compile(r"\bABN\b[:\s]*([\d ]{11,16})", re.I)
DP = re.compile(r"\b(DP|SP)[\s.]*(\d{4,7})\b")


def _digits(text):
    return re.sub(r"\D", "", text)


def _flat(text):
    return re.sub(r"[\s*_`|]+", " ", text).strip().lower()


def lint(folder):
    folder = Path(folder)
    errors = []
    try:
        manifest = json.loads((folder / "manifest.json").read_text(encoding="utf8"))
    except (OSError, ValueError) as e:
        return [f"manifest.json: {e}"]
    invented = dict(manifest.get("invented", {}))
    if manifest.get("base"):
        try:
            base = json.loads((folder.parent / manifest["base"] / "manifest.json").read_text(encoding="utf8"))
            for k, vals in base.get("invented", {}).items():
                invented[k] = list(invented.get(k, [])) + list(vals)
        except (OSError, ValueError):
            errors.append(f"base {manifest['base']}: manifest.json unreadable")
    abns = {_digits(a) for a in invented.get("abns", [])}
    dps = {m[1] for lot in invented.get("lots", []) for m in DP.findall(lot)}

    corpus = folder / "corpus"
    if not corpus.is_dir():
        return errors + ["missing corpus/"]
    expected = {d["filename"]: d for d in manifest.get("documents", [])}
    present = {p.name for p in corpus.iterdir() if p.is_file()}
    for name in sorted(set(expected) - present):
        errors.append(f"{name}: in the manifest but not in corpus/")
    for name in sorted(present - set(expected)):
        errors.append(f"{name}: in corpus/ but not in the manifest")

    for name in sorted(set(expected) & present):
        raw = (corpus / name).read_bytes()
        if b"\x00" in raw:
            errors.append(f"{name}: contains NUL bytes")
            continue
        try:
            text = raw.decode("utf8")
        except UnicodeDecodeError:
            errors.append(f"{name}: not UTF-8")
            continue
        flat = _flat(text)
        for c in expected[name].get("carries", []):
            if _flat(c.get("says", "")) not in flat:
                errors.append(f"{name}: does not state {c.get('key')} as the manifest says: {c.get('says')!r}")
        for m in LEAK_IDS.findall(text):
            errors.append(f"{name}: app id {m!r} leaks the answer key")
        for m in LEAK_WORDS.findall(text):
            errors.append(f"{name}: the word {m!r} gives the test away")
        for domain in EMAIL.findall(text):
            if domain.lower() != "example.com":
                errors.append(f"{name}: email domain {domain} is not example.com")
        found = {_digits(m) for m in ELEVEN.findall(text) if abn_ok(_digits(m))}
        found |= {_digits(m)[:11] for m in ABN_LABEL.findall(text) if len(_digits(m)) >= 11}
        for abn in sorted(found - abns):
            errors.append(f"{name}: ABN {abn} is not declared under invented.abns")
        for _, number in DP.findall(text):
            if number not in dps:
                errors.append(f"{name}: DP/SP {number} is not declared under invented.lots")
    return errors


def main(argv):
    if not argv or argv[0] in ("-h", "--help"):
        print(__doc__)
        return 0 if argv else 2
    failed = False
    for name in argv:
        if not Path(name).is_dir():
            continue
        errors = lint(name)
        print(f"{'FAIL' if errors else 'ok'}  {name}")
        for e in errors:
            print(f"  error: {e}")
        failed = failed or bool(errors)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
