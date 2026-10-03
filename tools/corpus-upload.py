"""Upload the deterministic corpus sample through the running local SiteWise API.

This invokes the app's configured Jev provider. Use only after approving live
classification of these private files. Creates a fresh QA project per corpus;
never deletes, corrects, or retries existing user documents.
"""
import argparse
import hashlib
import http.cookiejar
import json
from pathlib import Path
import time
import urllib.parse
import urllib.request
import urllib.error


def run():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--selected", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--base", default="http://127.0.0.1:8080")
    parser.add_argument("--allow-live-jev", action="store_true", required=True)
    parser.add_argument("--timeout", type=float, default=60)
    parser.add_argument("--resume", action="store_true", help="resume the same saved selection and QA projects")
    args = parser.parse_args()
    url = urllib.parse.urlparse(args.base)
    if url.scheme != "http" or url.hostname not in ("localhost", "127.0.0.1"):
        parser.error("this harness is restricted to the local development server")
    selected = json.loads(Path(args.selected).read_text(encoding="utf8"))
    # Verify every input before creating a project or sending a byte.
    for entry in selected:
        if hashlib.sha256(Path(entry["Path"]).read_bytes()).hexdigest() != entry["SHA256"]:
            raise ValueError(f"Corpus changed: {entry['Relative']}")
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    if (out / "results.json").exists() and not args.resume:
        parser.error("output already contains results; use a new directory or --resume")
    opener = urllib.request.build_opener(
        urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def request(path, body=None, content_type="application/json"):
        req = urllib.request.Request(args.base + path, data=body,
                                     headers={"Content-Type": content_type, "Origin": args.base})
        with opener.open(req, timeout=args.timeout) as response:
            payload = response.read()
            return json.loads(payload) if response.headers.get_content_type() == "application/json" else None

    request("/dev/login")
    projects = json.loads((out / "projects.json").read_text(encoding="utf8")) if args.resume else {}
    results = json.loads((out / "results.json").read_text(encoding="utf8")) if args.resume else []
    if [r["Entry"] for r in results] != selected[:len(results)]:
        raise ValueError("saved results do not match this selection")
    stamp = time.strftime("%Y%m%d-%H%M%S", time.gmtime())
    for entry in selected[len(results):]:
        corpus = entry["Corpus"]
        if corpus not in projects:
            projects[corpus] = request("/api/projects", json.dumps({"name": f"Intake QA {stamp} {corpus}"}).encode())["id"]
            (out / "projects.json").write_text(json.dumps(projects, indent=2), encoding="utf8")
        start = time.monotonic()
        rejection = ""
        try:
            doc = request(f"/api/projects/{projects[corpus]}/files?name=" + urllib.parse.quote(Path(entry["Path"]).name),
                          Path(entry["Path"]).read_bytes(), "application/pdf")
        except urllib.error.HTTPError as exc:
            if exc.code != 413:
                raise
            rejection = "HTTP 413: upload exceeds application size limit"
            doc = {"status": "upload_rejected", "fields": []}
        upload_end = time.monotonic()
        while doc["status"] == "pending" and time.monotonic() - upload_end < args.timeout:
            time.sleep(.1)
            doc = request("/api/documents/" + doc["id"])
        filing_ms = (time.monotonic()-upload_end)*1000
        source = doc
        sheets = []
        # Legacy gold labels a pack's first page only. Wait for expansion and
        # compare that page's actual child, never the hidden container fields.
        # Whole-pack correctness needs check-drawing-sheets.py and page gold.
        while doc.get("expansion", {}).get("status") == "pending" and time.monotonic() - upload_end < args.timeout:
            time.sleep(.1)
            doc = request("/api/documents/" + doc["id"])
        source = doc
        scope = "first_page_only" if doc.get("expansion") else "document_identity"
        if doc["status"] == "split":
            listing = request(f"/api/projects/{projects[corpus]}/documents")
            sheets = sorted((d for d in listing["documents"] if d.get("source_id") == source["id"]), key=lambda d: d["sheet_page"])
            scope = "first_page_only"
            if sheets and sheets[0]["sheet_page"] == 1:
                doc = sheets[0]
            else:
                rejection = "split source has no first sheet"
        elif doc.get("expansion", {}).get("status") == "pending":
            rejection = "drawing expansion timeout"
        result = {"Entry": entry, "Document": doc, "UploadMS": (upload_end-start)*1000,
                  "ObservedFilingMS": filing_ms, "ObservedExpansionMS": (time.monotonic()-upload_end)*1000,
                  "EvaluationScope": scope, "SourceDocument": source, "SheetDocuments": sheets, "Error": rejection}
        results.append(result)
        (out / "results.json").write_text(json.dumps(results, indent=2), encoding="utf8")
        print(f"{len(results)}/{len(selected)} {corpus}: {doc['status']}", flush=True)
    if any(r["Document"]["status"] == "pending" for r in results):
        raise RuntimeError("filing timeout; saved partial results")


if __name__ == "__main__":
    run()
