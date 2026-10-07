"""Read-only K4 review inventory. This does not approve or change knowledge."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess

import yaml


ROOT = Path(__file__).resolve().parents[1]
PREFIXES = ("fm.", "uc.", "sig.", "cq.")


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, encoding="utf-8")


def records(text):
    for rows in (yaml.safe_load(text) or {}).values():
        if isinstance(rows, list):
            for row in rows:
                if isinstance(row, dict) and str(row.get("id", "")).startswith(PREFIXES):
                    yield row


def changes(before, after, prefix=""):
    if isinstance(before, dict) and isinstance(after, dict):
        return [field for key in sorted(before.keys() | after.keys(), key=str)
                for field in changes(before.get(key), after.get(key), prefix + str(key) + ".")]
    return [] if before == after else [prefix.rstrip(".")]


def has_work_type(value):
    if isinstance(value, dict):
        return value.get("det") == "work_type" or any(has_work_type(v) for v in value.values())
    return isinstance(value, list) and any(has_work_type(v) for v in value)


def audit(baseline):
    baseline = git("rev-parse", "--verify", baseline + "^{commit}").strip()
    current, hashes = {}, {}
    for path in sorted((ROOT / "knowledge").rglob("*.yaml")):
        body = path.read_bytes()
        name = path.relative_to(ROOT).as_posix()
        hashes[name] = hashlib.sha256(body).hexdigest()
        for row in records(body.decode("utf-8")):
            if row["id"] in current:
                raise ValueError(f"duplicate record: {row['id']}")
            current[row["id"]] = (name, row)
    old = {}
    for path in git("ls-tree", "-r", "--name-only", baseline, "knowledge").splitlines():
        if path.endswith("/failure_modes.yaml"):
            for row in records(git("show", f"{baseline}:{path}")):
                old[row["id"]] = (path, row)
    deltas = []
    for identifier, (old_path, before) in sorted(old.items()):
        if identifier not in current:
            deltas.append({"id": identifier, "removed": True, "baseline_path": old_path})
            continue
        path, after = current[identifier]
        fields = changes(before, after)
        if fields or old_path != path:
            delta = {"id": identifier, "path": path, "changed_fields": fields,
                     "sources_only": fields == ["sources"], "moved": old_path != path}
            if "detector.runs_on" in fields:
                delta["runs_on_before"] = before["detector"]["runs_on"]
                delta["runs_on_after"] = after["detector"]["runs_on"]
            deltas.append(delta)
    unforeseen = {k: row for k, (_, row) in current.items() if k.startswith("uc.")}
    attachments = {}
    for row in unforeseen.values():
        for kind in row.get("attaches_to", {}):
            attachments[kind] = attachments.get(kind, 0) + 1
    return {
        "schema_version": 1, "baseline_commit": baseline,
        "knowledge_sha256": hashes,
        "record_counts": {p: sum(k.startswith(p) for k in current) for p in PREFIXES},
        "baseline_failure_modes": len(old), "failure_mode_deltas": deltas,
        "work_type_predicate_records": sorted(k for k, row in unforeseen.items() if has_work_type(row.get("when"))),
        "unforeseen_attachment_counts": attachments,
        "non_draft_records": [{"id": k, "status": row.get("status")}
                              for k, (_, row) in sorted(current.items()) if row.get("status") != "draft"],
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", default="02094db", help="pre-K4 commit")
    args = parser.parse_args()
    print(json.dumps(audit(args.baseline), indent=2, sort_keys=True))
