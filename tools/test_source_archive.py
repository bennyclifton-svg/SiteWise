"""Source integrity and standalone knowledge validation regressions."""
import hashlib
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import check_knowledge as checker


class SourceArchiveTests(unittest.TestCase):
    def test_missing_modified_and_escaping_sources_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'data/seed/example.md'
            source.parent.mkdir(parents=True)
            source.write_bytes(b'# Original\n')
            entry = {'path': 'data/seed/example.md',
                     'sha256': hashlib.sha256(source.read_bytes()).hexdigest()}
            manifest = root / 'manifest.json'
            manifest.write_text(json.dumps({'version': 1, 'files': [entry]}))
            report = checker.Report()
            checker.check_source_archive(root, report)
            self.assertEqual(report.errors, [])
            source.write_bytes(b'# Changed\n')
            report = checker.Report()
            checker.check_source_archive(root, report)
            self.assertTrue(any('checksum mismatch' in e for e in report.errors))
            source.unlink()
            report = checker.Report()
            checker.check_source_archive(root, report)
            self.assertTrue(any('not found' in e for e in report.errors))
            entry['path'] = '../outside.md'
            manifest.write_text(json.dumps({'version': 1, 'files': [entry]}))
            report = checker.Report()
            checker.check_source_archive(root, report)
            self.assertTrue(any('out-of-archive' in e for e in report.errors))

    def test_invalid_manifest_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for content in ['{', '[]', '{"version": 1, "files": []}']:
                (root / 'manifest.json').write_text(content)
                report = checker.Report()
                checker.check_source_archive(root, report)
                self.assertTrue(report.errors)

    def test_checkout_without_sibling_clerk_validates(self):
        with tempfile.TemporaryDirectory() as directory:
            isolated = Path(directory) / 'sitewise'
            isolated.mkdir()
            for name in ['knowledge', 'docs']:
                shutil.copytree(checker.ROOT / name, isolated / name)
            # Copy only metadata/source data needed by the checker, never private corpora.
            for name in ['reference', 'unforeseen', 'intake']:
                shutil.copytree(checker.ROOT / 'data' / name, isolated / 'data' / name)
            manifest_dir = isolated / 'data/eval/profile'
            manifest_dir.mkdir(parents=True)
            shutil.copyfile(checker.ROOT / 'data/eval/profile/manifest.json',
                            manifest_dir / 'manifest.json')
            (isolated / 'tools').mkdir()
            shutil.copyfile(checker.ROOT / 'tools/check_knowledge.py',
                            isolated / 'tools/check_knowledge.py')
            self.assertFalse((isolated.parent / 'clerk').exists())
            result = subprocess.run([sys.executable, 'tools/check_knowledge.py', '--strict'],
                                    cwd=isolated, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
