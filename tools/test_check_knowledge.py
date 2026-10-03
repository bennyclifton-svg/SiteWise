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


def _det(**extra):
    item = {'id': 'bal', 'label': 'BAL', 'value': 'choice', 'status': 'draft',
            'sources': [{'seed': 'seed.md', 'anchor': '# Test'}],
            'question': {'type': 'choice', 'instructions': 'Using `text`, which BAL?',
                         'criteria': {}, 'runs_on': ['envelope']}}
    item.update(extra)
    return item


def _check(kind, item, docs=None):
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        (root / 'seed.md').write_text('# Test\n')
        report, refs = checker.Report(), []
        with patch.object(checker, 'DOCUMENT_IDS', docs):
            checker.check_item(kind, 'test', item, root, report, refs, {})
        return report, refs


class ProfileSchemaTests(unittest.TestCase):
    def test_deprecated_requires_replaced_by(self):
        report, _ = _check('determinants', _det(status='deprecated'))
        self.assertTrue(any('replaced_by' in e for e in report.errors))
        report, refs = _check('determinants', _det(status='deprecated', replaced_by='bal_new'))
        self.assertEqual(report.errors, [])
        self.assertIn(('determinant', 'test [bal]', 'bal_new'), refs)

    def test_reference_to_deprecated_id_is_an_error(self):
        report = checker.Report()
        checker.check_deprecated_refs(
            [('system', 'rule x', 'fire-passive.old'), ('system', 'rule y', 'envelope.new')],
            {'system': {'fire-passive.old': 'envelope.new'}}, report)
        self.assertEqual(len(report.errors), 1)
        self.assertIn('deprecated', report.errors[0])

    def test_replaced_by_cannot_be_deprecated(self):
        report = checker.Report()
        checker.check_replacements({'system': {'a.old': 'a.mid', 'a.mid': 'a.new'}},
                                   {'system': {'a.old': 'f', 'a.mid': 'f', 'a.new': 'f'}}, report)
        self.assertTrue(any('a.mid' in e for e in report.errors))

    def test_document_source_must_be_in_manifest(self):
        src = [{'document': 'hale-brief', 'anchor': 'Recessed Docks'}]
        report, _ = _check('determinants', _det(sources=src), docs={'hale-brief'})
        self.assertEqual(report.errors, [])
        report, _ = _check('determinants', _det(sources=src), docs={'other'})
        self.assertTrue(any('hale-brief' in e for e in report.errors))
        report, _ = _check('determinants', _det(sources=src), docs=None)
        self.assertTrue(any('manifest' in e for e in report.errors))

    def test_clerk_file_source_must_exist(self):
        report, _ = _check('determinants', _det(sources=[{'clerk_file': 'no/such.json'}]))
        self.assertTrue(any('clerk file not found' in e for e in report.errors))

    def test_triggers_must_compile_without_re2_gaps(self):
        report, _ = _check('determinants', _det(triggers=[r'\bBAL[- ]?40\b']))
        self.assertEqual(report.errors, [])
        for bad in ['(unclosed', r'(?<=x)y', r'(a)\1']:
            report, _ = _check('determinants', _det(triggers=[bad]))
            self.assertTrue(report.errors, bad)

    def test_stated_in_and_profile_group(self):
        ok = [{'kind': 'report', 'discipline': 'consultant.bushfire', 'label': 'Bushfire report'}]
        report, _ = _check('determinants', _det(stated_in=ok, profile_group='site'))
        self.assertEqual(report.errors, [])
        bad = [{'kind': 'nope', 'discipline': 'consultant.bushfire', 'label': 'x'}]
        report, _ = _check('determinants', _det(stated_in=bad, profile_group='weather'))
        self.assertEqual(len(report.errors), 2)

    def test_scope_defaults_must_name_live_leaves_and_known_types(self):
        taxonomy = {'building_classes': [{'id': 'residential', 'label': 'Residential', 'subclasses': [
            {'id': 'house', 'label': 'House', 'ncc_class': '1a', 'scale_fields': []}]}],
            'work_types': [{'id': 'new', 'label': 'New build'}], 'conditions': []}
        doc = {'always_shown': ['state', 'nope'],
               'empty_work_types': ['refurb'],
               'presets': [{'id': 'fit', 'label': 'Fit-out', 'systems': ['hydraulic.gas']}, {'id': 'fit', 'systems': []}],
               'classes': [{'class': 'house', 'work_type': 'new', 'systems': ['hydraulic.gas', 'fire-passive.old', 'hydraulic']},
                           {'class': 'shed', 'work_type': 'new', 'systems': ['hydraulic.gas']}],
               'categories': [{'category': 'castles', 'systems': ['hydraulic.gas']}]}
        report = checker.Report()
        checker.check_scope_defaults('scope', doc, taxonomy,
                                     {'hydraulic.gas': 'f', 'fire-passive.old': 'f', 'hydraulic': 'f'},
                                     {'fire-passive.old': 'x'}, {'state': {'profile_group': 'site'}}, report)
        joined = ' '.join(report.errors)
        for expected in ('fire-passive.old', 'hydraulic`', 'shed', 'castles', 'refurb', 'nope', 'duplicate preset',
                         'preset needs id and label'):
            self.assertIn(expected, joined)

    def test_a_profile_determinant_must_be_able_to_become_relevant(self):
        determinants = {
            'by_rule': {'profile_group': 'site'},
            'by_derivation': {'profile_group': 'fire'},
            'by_systems': {'profile_group': 'site', 'systems': ['substructure']},
            'always': {'profile_group': 'classification'},
            'orphan': {'profile_group': 'site'},
            'not_profile': {},
        }
        rules = [('r1', {'applies_when': {'all': [{'det': 'by_rule', 'eq': 'x'}]}}),
                 ('r2', {'derives': {'table': 't', 'inputs': ['by_derivation'], 'gives': 'out'}})]
        report = checker.Report()
        checker.check_relevance('scope', {'always_shown': ['always']}, determinants, rules, report)
        self.assertEqual(len(report.warnings), 1)
        self.assertIn('orphan', report.warnings[0])

    def test_determinant_systems_are_references(self):
        report, refs = _check('determinants', _det(systems=['substructure', 'hydraulic.gas']))
        self.assertEqual(report.errors, [])
        self.assertIn(('system', 'substructure'), [(k, t) for k, _, t in refs])
        report, _ = _check('determinants', _det(systems=[]))
        self.assertTrue(report.errors)
