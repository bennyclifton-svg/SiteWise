import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('folder_batches', Path(__file__).with_name('check-folder-batches.py'))
runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(runner)


class FrozenRunTests(unittest.TestCase):
    def test_later_edits_cannot_change_captured_run(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            data = root / 'data'
            data.mkdir()
            originals = []
            for name in ('thresholds.json', 'kinds.json', 'disciplines.json', 'lifecycle.json'):
                p = data / name
                p.write_text('{}', encoding='utf8')
                originals.append(p)
            for name in ('evaluator.exe', 'roots.json', 'gold.json'):
                p = root / name
                p.write_text('original', encoding='utf8')
                originals.append(p)
            out = root / 'run'
            out.mkdir()
            exe, config, batches = runner.freeze_run(out, root / 'evaluator.exe', data,
                [{'name': 'test', 'roots': str(root / 'roots.json'), 'gold': str(root / 'gold.json')}])
            for p in originals:
                p.write_text('changed', encoding='utf8')
            self.assertEqual(exe.read_text(), 'original')
            self.assertEqual((config / 'thresholds.json').read_text(), '{}')
            self.assertEqual(Path(batches[0]['gold']).read_text(), 'original')
            self.assertEqual(Path(batches[0]['roots']).read_text(), 'original')

    def test_wrong_or_missing_live_version_fails_even_on_child_pages(self):
        field = {'decided_by': 'jev', 'question_version': 'intake-70'}
        result = [{'Document': {'fields': [{'decided_by': 'rule'}]}, 'SheetDocuments': [{'fields': [field]}]}]
        self.assertEqual(runner.verify_classifier_version(result, 'intake-70'), ['intake-70'])
        for version in ('intake-69', None):
            field['question_version'] = version
            with self.assertRaises(RuntimeError):
                runner.verify_classifier_version(result, 'intake-70')


if __name__ == '__main__':
    unittest.main()
