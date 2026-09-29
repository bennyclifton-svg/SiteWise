"""Regression checks for the knowledge publication boundary; no network needed."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import yaml

import check_knowledge as checker


class TableValidationTests(unittest.TestCase):
    def test_computed_requirement_cannot_overwrite_observed_fact(self):
        rule = {'id': 'rule.ncc.fixtures', 'derives': {'gives': 'fixture_count'}}
        report = checker.Report()
        checker.check_derivation_output('test', rule, {'fixture_count': {'derived': False}}, report)
        self.assertTrue(any('not an extracted fact' in error for error in report.errors))
        report = checker.Report()
        checker.check_derivation_output('test', rule, {
            'fixture_count': {'derived': True, 'by': 'rule.ncc.fixtures'}}, report)
        self.assertEqual(report.errors, [])

    def test_boolean_evidence_cannot_treat_silence_as_false(self):
        item = {'id': 'sprinklered', 'label': 'Sprinklered', 'value': 'boolean',
                'status': 'draft', 'sources': [{'seed': 'seed.md', 'anchor': '# Test'}],
                'question': {'type': 'noul', 'instructions': 'Using `text`, are sprinklers present?',
                             'criteria': {True: 'Present', False: 'Not mentioned'},
                             'runs_on': ['fire-active']}}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'seed.md').write_text('# Test\n')
            report = checker.Report()
            checker.check_item('determinants', 'test', item, root, report, [], {})
        self.assertTrue(any('not_stated choices' in error for error in report.errors))

    def test_unverified_table_is_rejected(self):
        table = yaml.safe_load((checker.KNOWLEDGE / 'tables/type_of_construction.yaml').read_text())
        table['verified'] = False
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'knowledge/tables').mkdir(parents=True)
            (root / 'knowledge/tables/type_of_construction.yaml').write_text(yaml.safe_dump(table))
            report = checker.Report()
            with patch.object(checker, 'ROOT', root), patch.object(checker, 'KNOWLEDGE', root / 'knowledge'):
                checker.check_tables(report, [])
        self.assertTrue(any('primary-verified' in error for error in report.errors))

    def test_active_derivation_registers_table_reference(self):
        item = {'id': 'rule.ncc.test', 'title': 'Test', 'instrument': 'NCC',
                'clause': 'Unverified', 'clause_verified': False, 'systems': ['structure'],
                'requires': 'Test', 'status': 'draft',
                'sources': [{'seed': 'seed.md', 'anchor': '# Test'}],
                'derives': {'table': 'missing_table', 'inputs': ['ncc_class'], 'gives': 'result'}}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'seed.md').write_text('# Test\n')
            report, refs = checker.Report(), []
            checker.check_item('rules', 'test', item, root, report, refs, {})
            self.assertIn(('table', 'test [rule.ncc.test]', 'missing_table'), refs)
            item['derives']['pending'] = True
            report, refs = checker.Report(), []
            checker.check_item('rules', 'test', item, root, report, refs, {})
            self.assertTrue(any('needs a reason' in error for error in report.errors))
            self.assertFalse(any(kind == 'table' for kind, _, _ in refs))

    def test_known_seed_construction_errors_do_not_return(self):
        table = yaml.safe_load((checker.KNOWLEDGE / 'tables/type_of_construction.yaml').read_text())
        def lookup(classification, rise):
            return [r['type_of_construction'] for r in table['rows']
                    if classification in r['classes'] and r['rise_min'] <= rise <= r.get('rise_max', float('inf'))]
        self.assertEqual(lookup('2', 3), ['A'])
        self.assertEqual(lookup('5', 2), ['C'])
        self.assertEqual(lookup('9a', 3), ['A'])
        self.assertEqual(lookup('1a', 2), [])
        self.assertEqual(lookup('5', 0), [])

    def test_compartment_table_preserves_area_and_volume(self):
        table = yaml.safe_load((checker.KNOWLEDGE / 'tables/compartment_limits.yaml').read_text())
        class5_b = [r for r in table['rows'] if '5' in r['classes'] and r['type_of_construction'] == 'B']
        self.assertEqual(len(class5_b), 1)
        self.assertEqual(class5_b[0]['max_floor_area_m2'], 5500)
        self.assertEqual(class5_b[0]['max_volume_m3'], 33000)


if __name__ == '__main__':
    unittest.main()
