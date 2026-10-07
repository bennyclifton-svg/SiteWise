"""Build an ignored, manually annotated 0991 rebuild timing fixture.

This is not an extraction result and must never be used as actual eval output.
No network, model calls, database writes, or owner-review changes are made.
"""
import hashlib
import json
import os
import subprocess
from pathlib import Path
import uuid

from pypdf import PdfReader


def main():
    repo = Path(__file__).resolve().parents[1]
    manifest = json.loads((repo / "data/eval/profile/manifest.json").read_text())
    key_bytes = (repo / "data/eval/profile/answer-keys/0991.yaml").read_bytes()
    validation = json.loads(subprocess.check_output(["go", "run", "./cmd/works-eval", "-key", "data/eval/profile/answer-keys/0991.yaml", "-validate-only"], cwd=repo, text=True, encoding="utf-8"))
    key = validation["key"]
    root = Path(os.environ.get(manifest["root_env"], manifest["root_default"])).resolve(strict=True)
    project_docs = next(p["documents"] for p in manifest["projects"] if p["id"] == "0991")
    if set(project_docs) != set(key["documents"]) or not key["work_items"]:
        raise SystemExit("0991 key and manifest disagree or key is empty")
    namespace = uuid.UUID("766d6d4d-c4b1-478b-9c5c-d53ac4c3c090")
    identity = lambda name: str(uuid.uuid5(namespace, name))
    project, site = identity("project"), identity("site")
    tables = {name: [] for name in ["sites", "project_parts", "files", "documents", "passages", "profile_facts", "work_items"]}
    tables["sites"] = [{"id": site, "label": "0991 draft benchmark"}]
    for label in ["Whole project", *key["parts"]]:
        tables["project_parts"].append({"id": identity("part:" + label), "site_id": site, "created_by_project_id": project, "kind": "whole" if label == "Whole project" else "building", "label": label})
    corpus = {}
    for doc_id in project_docs:
        doc = next(d for d in manifest["documents"] if d["id"] == doc_id)
        path = (root / doc["path"]).resolve(strict=True)
        if not path.is_relative_to(root):
            raise SystemExit("Corpus path escapes root")
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        if digest != doc["sha256"]:
            raise SystemExit(f"Changed source: {doc_id}")
        corpus[doc_id] = digest
        file_id, document_id = identity("file:" + doc_id), identity("document:" + doc_id)
        tables["files"].append({"id": file_id, "project_id": project, "sha256": "\\x" + digest, "byte_size": path.stat().st_size, "media_type": "application/pdf"})
        tables["documents"].append({"id": document_id, "project_id": project, "file_id": file_id, "filename": doc_id + ".pdf", "status": "filed"})
        for page, source in enumerate(PdfReader(path).pages, 1):
            text = source.extract_text() or ""
            if not text.strip():
                raise SystemExit(f"No extracted text for {doc_id}, page {page}")
            tables["passages"].append({"id": identity(f"passage:{doc_id}:{page}"), "document_id": document_id, "ordinal": page, "body": text})
    for item in key["work_items"]:
        tables["work_items"].append({"id": identity("work:" + item["id"]), "project_id": project, "site_id": site, "part_id": identity("part:" + item["part"]), "system_id": item["system"], "action": item["action"], "inclusion": "included", "title": item["id"], "origin": "document", "review_status": "proposed", "meaning": "stated", "provenance": {"rationale": "Draft manual source annotation for timing only; not Jev extraction or owner-reviewed scope", "sources": [{"document_id": identity("document:" + item["document"]), "page": item["page"]}]}})
    fixture = {"kind": "manual-source-timing", "project": "0991", "key_sha256": hashlib.sha256(key_bytes).hexdigest(), "corpus": corpus, "notes": "Two pinned drawings; whole-page text via pypdf; manually proposed work items; zero machine-read facts. Not an extraction accuracy fixture or complete project corpus.", "tables": tables}
    out = repo / "data/eval/profile/private/0991-bench.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(fixture, ensure_ascii=False), encoding="utf-8")
    print("Created private manual timing fixture:", {name: len(rows) for name, rows in tables.items()})


if __name__ == "__main__":
    main()
