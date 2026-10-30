"""Drop scenario corpora into a local SiteWise and save what it builds.

For each scenario: create a fresh project, drop each wave's documents on the
manifest's drop target, wait until filing and background reading are idle,
press profile Update, and wait again. After the last wave, press the user's
clicks that bring the cost plan, program and RFPs into being (SCORING.md §7)
and save every API response unmodified, for score.py and score_shapes.py.

Usage, from the repo root, with the separate instance from serve.sh running:

    python testing/synthetic/harness/run.py --allow-live-jev \
        testing/synthetic/scenarios/ST01-base [more scenario folders]

Safety: loopback only, never the owner's :8080 dev organisation, a new
project per scenario, never corrects, deletes or accepts anything. Raw Jev
exchanges are not recorded (the server has no recording switch, and this
work does not change app code); every decision the app stored, with its
band and confidence, is saved instead.
"""
import argparse
import http.client
import http.cookiejar
import json
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
RUNS = ROOT / "testing" / "synthetic" / "runs"
CONTENT_TYPE = "text/markdown; charset=utf-8"


class Client:
    def __init__(self, base, timeout=60):
        self.base = base.rstrip("/")
        self.timeout = timeout
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def call(self, method, path, body=None, content_type="application/json"):
        """One request, retried twice on a dropped connection (a reused
        keep-alive socket the server already closed). A retried upload is
        safe: the same bytes return the same document."""
        for attempt in range(3):
            try:
                return self._call(method, path, body, content_type)
            except (ConnectionError, http.client.HTTPException, urllib.error.URLError) as e:
                if isinstance(e, urllib.error.HTTPError) or attempt == 2:
                    raise
                time.sleep(1 + attempt)

    def _call(self, method, path, body=None, content_type="application/json"):
        if isinstance(body, (dict, list)):
            body = json.dumps(body).encode()
        req = urllib.request.Request(self.base + path, data=body, method=method,
                                     headers={"Content-Type": content_type, "Origin": self.base})
        try:
            with self.opener.open(req, timeout=self.timeout) as r:
                raw = r.read()
                return r.status, (json.loads(raw) if raw and r.headers.get_content_type() == "application/json" else None)
        except urllib.error.HTTPError as e:
            raw = e.read()
            try:
                return e.code, json.loads(raw)
            except ValueError:
                return e.code, {"error": raw.decode("utf8", "replace")[:500]}

    def ok(self, method, path, body=None, content_type="application/json"):
        status, data = self.call(method, path, body, content_type)
        if status >= 400:
            raise RuntimeError(f"{method} {path} -> {status}: {data}")
        return data


def check_base(base):
    url = urllib.parse.urlparse(base)
    if url.scheme != "http" or url.hostname not in ("127.0.0.1", "localhost"):
        raise SystemExit("the harness runs only against a loopback server")
    if url.port in (None, 8080):
        raise SystemExit("refusing :8080, the owner's dev organisation; start harness/serve.sh (:8091)")


def busy(c, project):
    """What is still being read, or [] when idle."""
    out = []
    docs = c.ok("GET", f"/api/projects/{project}/documents").get("documents", [])
    pending = [d for d in docs if d.get("status") == "pending"]
    if pending:
        out.append(f"{len(pending)} documents filing")
    invoices = c.ok("GET", f"/api/projects/{project}/invoices")
    rows = invoices.get("invoices", []) if isinstance(invoices, dict) else invoices or []
    reading = [i for i in rows if i.get("status") == "reading"]
    if reading:
        out.append(f"{len(reading)} invoices reading")
    board = c.ok("GET", f"/api/projects/{project}/procurement/board")
    cards = [x for cs in board.get("packages", {}).values() for x in cs] + board.get("unassigned", [])
    if any(x.get("status") == "reading" for x in cards):
        out.append("proposals reading")
    profile = c.ok("GET", f"/api/projects/{project}/profile")
    reading = (profile.get("pending_documents") or 0) + (profile.get("active_documents") or 0)
    if reading:
        out.append(f"{reading} documents in profile reading")
    status, health = c.call("GET", "/api/health")
    if status == 200:
        queued = sum(b.get("queued", 0) for b in health.get("backlog", []))
        if queued:
            out.append(f"{queued} jobs queued")
    return out


def wait_idle(c, project, timeout, log, settle=2.0):
    """Idle twice in a row, `settle` seconds apart, so a job queued by a
    finishing job is not missed."""
    deadline = time.monotonic() + timeout
    calm = 0
    last = None
    while time.monotonic() < deadline:
        state = busy(c, project)
        calm = calm + 1 if not state else 0
        if calm >= 2:
            return True
        if state != last:
            log(f"    waiting: {', '.join(state) or 'settling'}")
            last = state
        time.sleep(settle)
    log(f"    TIMEOUT after {timeout}s: {', '.join(busy(c, project))}")
    return False


def package_for(c, project, discipline):
    data = c.ok("GET", f"/api/projects/{project}/packages")
    for p in data.get("items", []):
        if p.get("discipline_id") == discipline and p.get("lifecycle_status") != "cancelled" and not p.get("retired_at"):
            return p["id"], False
    p = c.ok("POST", f"/api/projects/{project}/procurement/rows", {"discipline_id": discipline})
    return p["id"], True


def drop(c, project, doc, data, log):
    name = urllib.parse.quote(doc["filename"])
    target = doc["drop"]
    if target == "repository":
        path = f"/api/projects/{project}/files?name={name}"
    elif target == "invoices":
        path = f"/api/projects/{project}/invoices?name={name}"
    elif target.startswith("row:"):
        pkg, created = package_for(c, project, target[4:])
        if created:
            log(f"    created the {target[4:]} row to drop on, as a user would")
        path = f"/api/projects/{project}/procurement/packages/{pkg}/submissions?name={name}"
    else:
        raise ValueError(f"unknown drop target {target}")
    status, body = c.call("POST", path, data, CONTENT_TYPE)
    return {"doc": doc["id"], "filename": doc["filename"], "drop": target, "http": status, "response": body}


def press_update(c, project):
    """The "Update project profile" button. The server queues the read before
    it writes the response, so a response cut off mid-body (seen while the
    app is busy) still counts as pressed; the cut is reported, not retried."""
    try:
        status, body = c._call("POST", f"/api/projects/{project}/profile/read", {})
    except (ConnectionError, http.client.HTTPException) as e:
        return None, type(e).__name__
    if status >= 400:
        raise RuntimeError(f"profile read -> {status}: {body}")
    return body.get("queued"), None


def save(folder, name, data):
    p = folder / name
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(data, indent=2, ensure_ascii=False), encoding="utf8", newline="\n")


def snapshot(c, project, folder, prepare, log):
    get = lambda path: c.ok("GET", f"/api/projects/{project}{path}")
    save(folder, "documents.json", get("/documents"))
    docs = get("/documents").get("documents", [])
    save(folder, "document_details.json", [c.ok("GET", f"/api/documents/{d['id']}") for d in docs])
    save(folder, "profile.json", get("/profile"))
    save(folder, "profile_sources.json", get("/profile/sources"))
    save(folder, "invoices.json", get("/invoices"))
    save(folder, "board.json", get("/procurement/board"))
    if not prepare:
        save(folder, "packages.json", get("/packages"))
        return
    # The user's clicks that create the cost plan and program (SCORING.md §7).
    save(folder, "cost_plan.json", c.ok("POST", f"/api/projects/{project}/cost-plan/prepare", {}))
    status, body = c.call("POST", f"/api/projects/{project}/program/prepare", {"surface": "program"})
    log(f"    program prepare -> {status}")
    packages = get("/packages")
    save(folder, "packages.json", packages)
    prices = {}
    for p in packages.get("items", []):
        st, body = c.call("GET", f"/api/projects/{project}/procurement/packages/{p['id']}/prices")
        if st == 200:
            prices[p["id"]] = body
    save(folder, "prices.json", prices)
    # One RFP per services package, as a user would create from the table.
    reports = get("/reports")
    rows = reports.get("reports", []) if isinstance(reports, dict) else reports or []
    existing = {r.get("package_id") for r in rows}
    for p in packages.get("items", []):
        if p.get("kind") == "services" and p.get("lifecycle_status") != "cancelled" and p["id"] not in existing:
            st, body = c.call("POST", f"/api/projects/{project}/reports", {"kind": "rfp", "package_id": p["id"]})
            log(f"    RFP for {p.get('title')} -> {st}")
    reports = get("/reports")
    rows = reports.get("reports", []) if isinstance(reports, dict) else reports
    for r in rows:
        if r.get("kind") == "rfp":
            # Opening an RFP assembles its draft (web/src/reportApi.ts ensure).
            st, _ = c.call("POST", f"/api/reports/{r['id']}/draft", {"only_if_missing": True})
            if st >= 400:
                log(f"    RFP draft {r.get('title')} -> {st}")
            save(folder, f"reports/{r['id']}.json", c.ok("GET", f"/api/reports/{r['id']}"))
    status, program = c.call("GET", f"/api/projects/{project}/program")
    save(folder, "program.json", program)


def git_state():
    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip()
    dirty = subprocess.run(["git", "status", "--porcelain", "--untracked-files=no", "--", "internal", "cmd", "knowledge", "data", "web/src"],
                           cwd=ROOT, capture_output=True, text=True).stdout.strip()
    return {"head": head, "app_files_modified": len(dirty.splitlines()) if dirty else 0}


def run_scenario(c, folder, out, args, log):
    manifest = json.loads((folder / "manifest.json").read_text(encoding="utf8"))
    corpus = folder / "corpus"
    name = manifest["scenario"]
    dest = out / name
    project = c.ok("POST", "/api/projects", {"name": f"Synthetic {name} {out.name}"})["id"]
    log(f"{name}: project {project}")
    drops, timeline, cuts = [], [], []
    waves = sorted({d["wave"] for d in manifest["documents"]})
    for wave in waves:
        start = time.monotonic()
        for doc in [d for d in manifest["documents"] if d["wave"] == wave]:
            r = drop(c, project, doc, (corpus / doc["filename"]).read_bytes(), log)
            drops.append(dict(r, wave=wave))
            if r["http"] >= 400:
                log(f"    {doc['id']} {doc['filename']}: HTTP {r['http']} {r['response']}")
        idle = wait_idle(c, project, args.timeout, log)
        queued, cut = press_update(c, project)
        if cut:
            cuts.append({"wave": wave, "error": cut})
            log(f"    APP FINDING: profile Update response cut off ({cut}); the read was queued, continuing")
        idle_profile = wait_idle(c, project, args.timeout, log)
        timeline.append({"wave": wave, "documents": sum(1 for d in drops if d["wave"] == wave),
                         "idle": idle and idle_profile, "profile_queued": queued,
                         "seconds": round(time.monotonic() - start, 1)})
        log(f"  wave {wave}: {timeline[-1]['documents']} dropped, profile queued {queued}, {timeline[-1]['seconds']} s")
        if args.wave_snapshots:
            snapshot(c, project, dest / f"wave-{wave}", False, log)
    snapshot(c, project, dest / "snapshot", True, log)
    save(dest, "run.json", {"scenario": name, "project_id": project, "base": args.base, "git": git_state(),
                            "started": out.name, "waves": timeline, "drops": drops,
                            "profile_update_cut": cuts})
    return dest


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("scenarios", nargs="+", type=Path)
    ap.add_argument("--base", default="http://127.0.0.1:8091")
    ap.add_argument("--out", type=Path, help="run folder (default testing/synthetic/runs/<UTC stamp>)")
    ap.add_argument("--allow-live-jev", action="store_true", required=True,
                    help="required: dropping documents makes live Jev calls")
    ap.add_argument("--timeout", type=float, default=900, help="seconds to wait for idle per step")
    ap.add_argument("--wave-snapshots", action="store_true", help="also save documents and profile after each wave")
    args = ap.parse_args(argv)
    check_base(args.base)
    out = args.out or RUNS / time.strftime("%Y%m%d-%H%M%S", time.gmtime())
    out.mkdir(parents=True, exist_ok=True)
    logfile = (out / "harness.log").open("a", encoding="utf8")

    def log(msg):
        print(msg, flush=True)
        logfile.write(msg + "\n")
        logfile.flush()

    c = Client(args.base)
    c.ok("GET", "/dev/login")
    for folder in args.scenarios:
        run_scenario(c, folder, out, args, log)
    log(f"done: {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
