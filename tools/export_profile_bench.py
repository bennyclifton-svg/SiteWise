"""Export a private profile benchmark fixture using read-only PostgreSQL queries.

No model calls or database writes. Output must stay in the ignored private folder.
"""
import argparse
import json
import pathlib
import subprocess
import uuid


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--psql", default="psql")
    parser.add_argument("--database", required=True)
    parser.add_argument("--project", required=True, type=uuid.UUID)
    args = parser.parse_args()
    project = str(args.project)
    owner = f"(SELECT org_id FROM projects WHERE id='{project}'::uuid)"
    site = f"(SELECT site_id FROM projects WHERE id='{project}'::uuid)"
    docs = f"(SELECT id FROM documents WHERE org_id={owner} AND project_id='{project}'::uuid)"
    passages = f"(SELECT id FROM passages WHERE org_id={owner} AND document_id IN {docs})"
    filters = {
        "sites": f"id={site}",
        "project_parts": f"site_id={site}",
        "files": f"project_id='{project}'::uuid",
        "documents": f"project_id='{project}'::uuid",
        "decisions": f"document_id IN {docs}",
        "supersessions": f"document_id IN {docs}",
        "passages": f"document_id IN {docs}",
        "passage_sources": f"passage_id IN {passages}",
        "document_sources": f"document_id IN {docs}",
        "profile_facts": f"project_id='{project}'::uuid",
        "profile_user_values": f"(project_id='{project}'::uuid OR site_id={site})",
        "profile_planning_values": f"(project_id='{project}'::uuid OR site_id={site})",
    }
    fields = []
    for table, predicate in filters.items():
        # Generated search columns are recalculated on import.
        fields.append(f"'{table}', (SELECT COALESCE(jsonb_agg(to_jsonb(t)-'body_tsv'), '[]') FROM {table} t WHERE org_id={owner} AND {predicate})")
    sql = "BEGIN READ ONLY; SELECT jsonb_build_object(" + ",".join(fields) + "); COMMIT;"
    raw = subprocess.check_output([args.psql, args.database, "-X", "-qAt", "-v", "ON_ERROR_STOP=1", "-c", sql], text=True, encoding="utf-8")
    tables = json.loads(raw)
    if len(tables["sites"]) != 1 or not tables["profile_facts"]:
        raise SystemExit("Expected one site and nonempty profile facts")
    out = pathlib.Path(__file__).resolve().parents[1] / "data/eval/profile/private/spec-home-bench.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(tables, ensure_ascii=False), encoding="utf-8")
    print("Exported private fixture:", {name: len(rows) for name, rows in tables.items()})


if __name__ == "__main__":
    main()
