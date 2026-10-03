"""Exercise the upload/poll harness against a local HTTP fixture, without Jev."""
import hashlib
import http.server
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest


class UploadHarnessTest(unittest.TestCase):
    def test_upload_polls_and_saves_persisted_fields(self):
        requests = []

        class Handler(http.server.BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def reply(self, value):
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Set-Cookie", "testsession=local; Path=/")
                self.end_headers()
                self.wfile.write(json.dumps(value).encode())

            def do_GET(self):
                requests.append((self.path, b""))
                if self.path == "/dev/login":
                    self.reply({})
                else:
                    self.reply({"id": "doc", "status": "filed",
                                "fields": [{"field": "number", "value": "S101", "band": "green"}]})

            def do_POST(self):
                body = self.rfile.read(int(self.headers["Content-Length"]))
                requests.append((self.path, body))
                if self.path == "/api/projects":
                    self.reply({"id": "project"})
                elif "oversize" in self.path:
                    self.send_error(413)
                else:
                    self.reply({"id": "doc", "status": "pending"})

        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                pdf = root / "S101 plan.pdf"
                pdf.write_bytes(b"%PDF-fixture")
                selected = root / "selected.json"
                entries = [{
                    "Corpus": "fixture", "Path": str(pdf), "Relative": pdf.name,
                    "SHA256": hashlib.sha256(pdf.read_bytes()).hexdigest()}]
                oversize = root / "oversize.pdf"
                oversize.write_bytes(b"%PDF-too-large-fixture")
                entries.append({"Corpus": "fixture", "Path": str(oversize), "Relative": oversize.name,
                                "SHA256": hashlib.sha256(oversize.read_bytes()).hexdigest()})
                selected.write_text(json.dumps(entries))
                command = [
                    sys.executable, str(Path(__file__).with_name("corpus-upload.py")),
                    "--selected", str(selected), "--out", str(root / "out"),
                    "--base", f"http://127.0.0.1:{server.server_port}", "--allow-live-jev"
                ]
                subprocess.run(command, check=True, capture_output=True, timeout=10)
                saved = json.loads((root / "out/results.json").read_text())
                self.assertEqual(saved[0]["Document"]["fields"][0]["value"], "S101")
                self.assertIn("413", saved[1]["Error"])
                self.assertIn(("/api/projects/project/files?name=S101%20plan.pdf", b"%PDF-fixture"), requests)
                self.assertIn(("/api/documents/doc", b""), requests)
                count = len([r for r in requests if "/files?" in r[0]])
                subprocess.run(command + ["--resume"], check=True, capture_output=True, timeout=10)
                self.assertEqual(count, len([r for r in requests if "/files?" in r[0]]))
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


if __name__ == "__main__":
    unittest.main()
