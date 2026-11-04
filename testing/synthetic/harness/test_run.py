"""Checks for run.py against a fake server. From the repo root:

    python -m unittest discover -s testing/synthetic/harness
"""
import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse

import run

P = "11111111-1111-1111-1111-111111111111"


class Fake(BaseHTTPRequestHandler):
    calls = []
    pending_polls = 0
    packages = []

    def log_message(self, *a):
        pass

    def reply(self, status, body):
        raw = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        path = urlparse(self.path).path
        Fake.calls.append(("GET", self.path))
        if path == f"/api/projects/{P}/documents":
            status = "pending" if Fake.pending_polls > 0 else "filed"
            Fake.pending_polls -= 1
            return self.reply(200, {"documents": [{"id": "d", "status": status}]})
        if path.endswith("/invoices"):
            return self.reply(200, {"invoices": []})
        if path.endswith("/procurement/board"):
            return self.reply(200, {"packages": {}, "unassigned": []})
        if path.endswith("/profile"):
            return self.reply(200, {"pending_documents": 0, "active_documents": 0})
        if path.endswith("/packages"):
            return self.reply(200, {"items": Fake.packages})
        if path == "/api/health":
            return self.reply(200, {"backlog": []})
        return self.reply(404, {})

    def do_POST(self):
        n = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(n)
        Fake.calls.append(("POST", self.path, self.headers.get("Content-Type"), body))
        if self.path.endswith("/procurement/rows"):
            return self.reply(201, {"id": "pkg-new"})
        return self.reply(201, {"id": "x", "status": "pending"})


class Harness(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = HTTPServer(("127.0.0.1", 0), Fake)
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()
        cls.c = run.Client(f"http://127.0.0.1:{cls.server.server_port}")

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        Fake.calls.clear()
        Fake.pending_polls = 0
        Fake.packages = []

    def test_refuses_remote_and_dev_org(self):
        for base in ("http://example.com:8091", "http://127.0.0.1:8080", "https://127.0.0.1:8091"):
            with self.assertRaises(SystemExit):
                run.check_base(base)
        run.check_base("http://127.0.0.1:8091")

    def test_drop_targets(self):
        for target, want in (("repository", f"/api/projects/{P}/files?name=A%20b.md"),
                             ("invoices", f"/api/projects/{P}/invoices?name=A%20b.md")):
            r = run.drop(self.c, P, {"id": "d01", "filename": "A b.md", "drop": target}, b"# A", print)
            self.assertEqual(r["http"], 201)
            self.assertEqual(Fake.calls[-1][1], want)
            self.assertTrue(Fake.calls[-1][2].startswith("text/markdown"))
            self.assertEqual(Fake.calls[-1][3], b"# A")

    def test_row_drop_uses_existing_package(self):
        Fake.packages = [{"id": "pkg-1", "discipline_id": "consultant.structural", "lifecycle_status": "proposed"}]
        run.drop(self.c, P, {"id": "d05", "filename": "f.md", "drop": "row:consultant.structural"}, b"x", print)
        self.assertEqual(Fake.calls[-1][1], f"/api/projects/{P}/procurement/packages/pkg-1/submissions?name=f.md")
        self.assertFalse(any(c[1].endswith("/procurement/rows") for c in Fake.calls if c[0] == "POST"))

    def test_row_drop_creates_row_when_missing(self):
        run.drop(self.c, P, {"id": "d05", "filename": "f.md", "drop": "row:consultant.structural"}, b"x", print)
        posts = [c for c in Fake.calls if c[0] == "POST"]
        self.assertTrue(posts[0][1].endswith("/procurement/rows"))
        self.assertEqual(json.loads(posts[0][3]), {"discipline_id": "consultant.structural"})
        self.assertIn("/packages/pkg-new/submissions", posts[1][1])

    def test_wait_idle_waits_for_filing(self):
        Fake.pending_polls = 2
        self.assertTrue(run.wait_idle(self.c, P, timeout=5, log=lambda m: None, settle=0.01))
        polls = [c for c in Fake.calls if c[1] == f"/api/projects/{P}/documents"]
        self.assertGreaterEqual(len(polls), 4)  # two busy, then idle twice

    def test_wait_idle_times_out(self):
        Fake.pending_polls = 10 ** 6
        msgs = []
        self.assertFalse(run.wait_idle(self.c, P, timeout=0.05, log=msgs.append, settle=0.01))
        self.assertTrue(any("TIMEOUT" in m for m in msgs))


if __name__ == "__main__":
    unittest.main()
