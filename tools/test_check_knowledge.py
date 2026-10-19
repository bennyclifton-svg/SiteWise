"""Regression checks for the knowledge publication boundary; no network needed."""
import datetime
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import yaml

import check_knowledge as checker


class TableValidationTests(unittest.TestCase):
    def test_predicate_is_rejects_choice_strings_and_other_nonbooleans(self):
        for value in ['NSW', 'true', 1, 0, None, [], {}]:
            with self.subTest(value=value):
                report = checker.Report()
                checker.check_predicate('test', {'det': 'state', 'is': value}, report, [])
                self.assertTrue(any('requires a boolean' in e for e in report.errors))
        for pred in [{'det': 'existing_building', 'is': True},
                     {'det': 'existing_building', 'is': False},
                     {'det': 'state', 'any_of': ['NSW']}]:
            report = checker.Report()
            checker.check_predicate('test', pred, report, [])
            self.assertEqual(report.errors, [])

    def test_proposal_flow_label_cannot_silently_truncate_at_comma(self):
        broken = yaml.safe_load('{kind: investigation, label: Supply (water, cooling)}')
        report = checker.Report()
        checker.check_proposal('test', broken, {'investigation'}, report)
        self.assertTrue(any('unknown fields' in e for e in report.errors))
        correct = yaml.safe_load('{kind: investigation, label: "Supply (water, cooling)"}')
        report = checker.Report()
        checker.check_proposal('test', correct, {'investigation'}, report)
        self.assertEqual(report.errors, [])
        self.assertEqual(correct['label'], 'Supply (water, cooling)')

    def test_package_complexity_values_use_catalogue_options(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            knowledge = root / 'knowledge'
            (knowledge / 'works').mkdir(parents=True)
            (knowledge / 'profile').mkdir()
            (root / 'seed.md').write_text('# Test\n', encoding='utf-8')
            (knowledge / 'profile/taxonomy.yaml').write_text(yaml.safe_dump({
                'conditions': [{'key': 'planning', 'options': [{'id': 'da'}]}]}), encoding='utf-8')
            path = knowledge / 'works/package_defaults.yaml'
            def check(additions):
                path.write_text(yaml.safe_dump({'version': 2, 'status': 'draft', 'baselines': [],
                    'source': {'seed': 'seed.md', 'anchor': '# Test'},
                    'complexity_additions': additions}), encoding='utf-8')
                report = checker.Report()
                with patch.object(checker, 'ROOT', root), patch.object(checker, 'KNOWLEDGE', knowledge):
                    checker.check_catalogues(report, root, {}, {
                        'heritage_status': {'value': 'choice', 'options': [{'id': 'local_item'}]},
                        'existing_building': {'value': 'boolean'}})
                return report
            def addition(field, values):
                return {'field': field, 'values': values, 'consultants': ['Consultant']}
            valid = check([addition('heritage_status', ['local_item']), addition('planning', ['da']),
                           addition('existing_building', ['true', 'false'])])
            self.assertEqual(valid.errors, [])
            self.assertEqual(valid.warnings, [])
            invalid = check([addition('heritage_status', ['local_heritage_item']),
                             addition('planning', ['legacy_da']), addition('existing_building', ['yes']),
                             addition('legacy_field', ['legacy_value'])])
            self.assertEqual(invalid.errors, [])
            self.assertEqual(len(invalid.warnings), 4)
            self.assertTrue(any('local_heritage_item' in w and 'heritage_status' in w for w in invalid.warnings))
            self.assertTrue(any('legacy_field' in w for w in invalid.warnings))
            malformed = check([addition('planning', 'da'), addition('planning', [{}]), addition({}, ['da'])])
            self.assertEqual(len(malformed.errors), 3)

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


class CatalogueTests(unittest.TestCase):
    """Delivery and commercial catalogues (SCHEMA.md, 2026-10-05)."""

    def run_catalogues(self, files):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for rel, body in files.items():
                path = root / 'knowledge' / rel
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(yaml.safe_dump(body))
            report = checker.Report()
            with patch.object(checker, 'ROOT', root), patch.object(checker, 'KNOWLEDGE', root / 'knowledge'):
                checker.check_catalogues(report, root, {}, {})
        return report

    def test_money_totals_cannot_be_planning_values(self):
        report = self.run_catalogues({'profile/planning_keys.yaml': {'version': 1, 'keys': [
            {'key': 'cost.total', 'label': 'Total', 'value': 'number', 'scope': 'project'},
            {'key': 'budget', 'label': 'Budget', 'value': 'money', 'scope': 'project'}]}})
        self.assertEqual(sum('money totals' in e for e in report.errors), 2)

    def test_planning_favourable_values_fit_the_key(self):
        report = self.run_catalogues({'profile/planning_keys.yaml': {'version': 1, 'keys': [
            {'key': 'site_classification', 'label': 'Site class', 'value': 'choice', 'scope': 'site',
             'options': [{'id': 'A'}, {'id': 'P'}], 'favourable': ['A', 'Z'], 'starting_value': 'A'},
            {'key': 'Bad.Key', 'label': 'Bad', 'value': 'text', 'scope': 'site'}]}})
        self.assertTrue(any("'Z'" in e for e in report.errors))
        self.assertTrue(any('starting value may not be favourable' in e for e in report.errors))
        self.assertTrue(any('plan.<key>' in e for e in report.errors))

    def test_benchmark_amount_must_be_a_decimal_string(self):
        report = self.run_catalogues({'costs/benchmarks.yaml': {'version': 1, 'benchmarks': [
            {'id': 'bm.test', 'version': 1, 'basis': 'lump_sum', 'amount': 1250.5, 'currency': 'AUD',
             'tax_basis': 'ex_tax', 'price_date': '2026-10-01', 'geography': 'NSW', 'quality': 'standard',
             'inclusions': [], 'exclusions': [], 'status': 'draft', 'sources': []}]}})
        self.assertTrue(any('decimal string' in e for e in report.errors))

    def test_stages_must_be_the_unforeseen_stage_targets(self):
        report = self.run_catalogues({'works/stages.yaml': {'version': 1, 'status': 'draft', 'stages': [
            {'id': 'design', 'label': 'Design', 'substages': [{'id': 'concept_design', 'label': 'Concept', 'novation': 'later'}]}]}})
        self.assertTrue(any('top-level stages' in e for e in report.errors))
        self.assertTrue(any('novation' in e for e in report.errors))

    def test_clause_needs_outputs_and_a_version(self):
        report = self.run_catalogues({'reports/clauses.yaml': {'version': 1, 'clauses': [
            {'id': 'cl.x', 'version': 0, 'outputs': ['memo'], 'section': 's', 'text': 't', 'status': 'draft'}]}})
        self.assertTrue(any('positive integer' in e for e in report.errors))
        self.assertTrue(any('outputs' in e for e in report.errors))

    def test_repository_catalogues_pass(self):
        report = checker.Report()
        checker.check_catalogues(report, checker.DEFAULT_SEED_DIR, {}, {})
        self.assertEqual(report.errors, [])

    def test_template_rejects_duplicate_sections_and_missing_essential(self):
        report = self.run_catalogues({'reports/templates.yaml': {'version': 1, 'templates': [
            {'id': 'tpl.test', 'version': 1, 'kind': 'rfp', 'status': 'draft', 'sources': [],
             'sections': [{'id': 'brief', 'title': 'Brief', 'clauses': []},
                          {'id': 'brief', 'title': 'Duplicate', 'essential': True, 'clauses': []}]}]}})
        self.assertTrue(any('essential boolean' in e for e in report.errors))
        self.assertTrue(any('duplicate section' in e for e in report.errors))

    def test_template_rejects_wrong_clause_version_and_output(self):
        report = self.run_catalogues({
            'reports/clauses.yaml': {'version': 1, 'clauses': [
                {'id': 'cl.test', 'version': 1, 'outputs': ['rft'], 'section': 'brief',
                 'text': 'Fixture', 'status': 'draft', 'sources': []}]},
            'reports/templates.yaml': {'version': 1, 'templates': [
                {'id': 'tpl.test', 'version': 1, 'kind': 'rfp', 'status': 'draft', 'sources': [],
                 'sections': [{'id': 'brief', 'title': 'Brief', 'essential': True,
                               'clauses': [{'id': 'cl.test', 'version': 2}]}]}]}})
        self.assertTrue(any('incompatible clause reference' in e for e in report.errors))


class RFPCheckTests(unittest.TestCase):
    def run_check(self, discipline: dict, answer_key: dict | None = None):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            knowledge = root / 'knowledge'
            (knowledge / 'rfp').mkdir(parents=True)
            (root / 'docs').mkdir()
            (root / 'docs' / 'golden.md').write_text('# Golden\n', encoding='utf-8')
            (knowledge / 'rfp' / 'architecture.yaml').write_text(yaml.safe_dump(discipline), encoding='utf-8')
            keys = root / 'keys'
            keys.mkdir()
            if answer_key is not None:
                (keys / 'scenario.yaml').write_text(yaml.safe_dump(answer_key), encoding='utf-8')
            report = checker.Report()
            with patch.object(checker, 'ROOT', root), patch.object(checker, 'KNOWLEDGE', knowledge), patch.object(checker, 'RFP_ANSWER_KEYS', keys):
                checker.check_rfp(report)
            return report

    def base(self, **extra):
        doc = {'id': 'architecture', 'status': 'draft', 'golden': 'docs/golden.md',
               'thresholds': [{'key': 't', 'kind': 'client_default', 'text': 't', 'source': 's', 'verified': False}],
               'stages': [{'key': 'concept', 'provenance': [{'kind': 'evidence', 'ref': 'architecture#1'}]}],
               'positions': [{'key': 'bim', 'provenance': [{'kind': 'evidence', 'ref': 'architecture#2'}]}]}
        doc.update(extra)
        return doc

    def test_valid_draft_passes(self):
        self.assertEqual(self.run_check(self.base()).errors, [])

    def test_reviewed_needs_the_owner_review_record(self):
        report = self.run_check(self.base(status='reviewed'))
        self.assertTrue(any('reviewed_by' in e for e in report.errors), report.errors)
        report = self.run_check(self.base(status='reviewed', reviewed_by='Owner', reviewed_on='2026-10-10'))
        self.assertEqual(report.errors, [])

    def test_threshold_needs_kind_verified_and_source(self):
        report = self.run_check(self.base(thresholds=[{'key': 't', 'text': 't', 'source': 's'}]))
        self.assertTrue(any('kind and verified' in e for e in report.errors), report.errors)
        report = self.run_check(self.base(thresholds=[{'key': 't', 'kind': 'client_default', 'text': 't', 'source': '', 'verified': False}]))
        self.assertTrue(any('needs a source' in e for e in report.errors), report.errors)

    def test_golden_path_must_exist(self):
        report = self.run_check(self.base(golden='docs/missing.md'))
        self.assertTrue(any('golden file not found' in e for e in report.errors), report.errors)

    def test_client_decision_needs_a_why(self):
        report = self.run_check(self.base(positions=[{'key': 'bim', 'provenance': [{'kind': 'client_decision', 'ref': 'x'}]}]))
        self.assertTrue(any('needs a why' in e for e in report.errors), report.errors)

    def test_answer_keys_name_real_rows_and_stay_drafts(self):
        key = {'reviewed': True, 'label_source': 'agent-draft',
               'expect': {'architecture': {'stages_include': ['concept', 'gone'], 'positions': {'bim': 'I', 'nope': 'X'}},
                          'acoustics': {}}}
        report = self.run_check(self.base(), key)
        joined = ' '.join(report.errors)
        for needle in ('cannot be marked reviewed', 'unknown stage gone', 'unknown position nope', 'unknown discipline acoustics'):
            self.assertIn(needle, joined)


# Test fixtures only: every number, label and clause is invented for the test.
_PROGRAM_SOURCE = [{'seed': 'seed.md', 'anchor': '# Test'}]
_GENERIC = ['investigation', 'concept_design', 'approval_documentation', 'construction_documentation',
            'approvals', 'procurement', 'construction', 'completion', 'defects']


def _program_fixture():
    def common(**item):
        item.update(status='draft', sources=_PROGRAM_SOURCE)
        return item
    return {
        'durations': [
            common(id='du.fixed', version=1, basis='fixed', unit_days='working', low='1', typical='2.5',
                   high='4', geography='Test', limitations=[]),
            common(id='du.stat', version=1, basis='statutory', unit_days='calendar', low='10', typical='10',
                   high='10', instrument='Test instrument', clause='T1', verified=False, geography='Test',
                   limitations=[]),
            common(id='du.tc', version=1, basis='time_cost', unit_days='calendar', k='100', b='0.5',
                   cost_base_date='2020-01-01', index_basis='Test index', geography='Test', limitations=[]),
        ],
        'fragments': [
            common(id='fr.a', label='A', stage='construction', systems=['structure'],
                   applies_when={'work_action': {'system': 'structure', 'any_of': ['new', 'replace']}},
                   activities=[{'key': 'one', 'label': 'One', 'systems': ['structure'], 'duration': 'du.fixed'},
                               {'key': 'two', 'label': 'Two', 'systems': ['structure'], 'duration': None}],
                   links=[{'from': 'one', 'to': 'two', 'type': 'FS', 'lag': None}],
                   attach={'after': 'stage_start', 'before': 'fr.b'}),
            common(id='fr.b', label='B', stage='construction', systems=['structure'],
                   activities=[{'key': 'one', 'label': 'One', 'systems': ['structure'], 'duration': None}]),
        ],
        'skeletons': [
            common(id='sk.generic', label='Generic', stages=list(_GENERIC),
                   links=[{'from': a, 'to': b, 'type': 'FS', 'lag': None} for a, b in zip(_GENERIC, _GENERIC[1:])]),
            common(id='sk.new-da', label='New with consent',
                   applies_when={'all': [{'det': 'work_type', 'eq': 'new'},
                                         {'cond': 'planning', 'any_of': ['da', 'cdc']}]},
                   stages=['design', 'approvals'],
                   links=[{'from': 'design', 'to': 'approvals', 'type': 'FS', 'lag': 'du.stat'}]),
        ],
        'calendars': [
            common(id='cal.test-2027', state='NSW', year=2027, working_week=['mon', 'tue', 'wed', 'thu', 'fri'],
                   days=[{'date': '2027-03-03', 'kind': 'public_holiday', 'label': 'Test day'},
                         {'from': '2027-12-24', 'to': '2027-12-31', 'kind': 'shutdown', 'label': 'Test shutdown'}],
                   verified=False),
        ],
    }


class ProgramLibraryTests(unittest.TestCase):
    """Checker rules 4.7 of docs/plans/2026-10-18-program.md."""

    def run_check(self, mutate=None):
        rows = _program_fixture()
        if mutate:
            mutate(rows)
        real = checker.KNOWLEDGE
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            knowledge = root / 'knowledge'
            for sub in ('profile', 'works', 'program'):
                (knowledge / sub).mkdir(parents=True)
            for rel in ('profile/taxonomy.yaml', 'profile/planning_keys.yaml', 'works/stages.yaml'):
                (knowledge / rel).write_bytes((real / rel).read_bytes())
            (root / 'seed.md').write_text('# Test\n', encoding='utf-8')
            for key, items in rows.items():
                (knowledge / 'program' / f'{key}.yaml').write_text(yaml.safe_dump(
                    {'version': 1, 'status': 'draft', 'sources': _PROGRAM_SOURCE, key: items}), encoding='utf-8')
            report, refs = checker.Report(), []
            actions = {'new', 'replace', 'upgrade', 'alter', 'repair', 'remove', 'retain', 'investigate'}
            with patch.object(checker, 'ROOT', root), patch.object(checker, 'KNOWLEDGE', knowledge), \
                    patch.object(checker, 'ACTIONS', actions):
                checker.check_program(report, root, {}, refs)
            return report

    def test_fixture_passes(self):
        self.assertEqual(self.run_check().errors, [])

    def test_each_rule_rejects_a_bad_case(self):
        def at(key, i, **change):
            def mutate(rows):
                rows[key][i].update(change)
            return mutate

        def calendar_day(i, **change):
            return lambda r: r['calendars'][0]['days'][i].update(change)

        cases = {
            'id does not match': at('skeletons', 1, id='sk_new'),
            'duplicate id': at('fragments', 1, id='fr.a'),
            'catalogue stage other than other': at('fragments', 1, stage='other'),
            'unknown duration': lambda r: r['fragments'][0]['activities'][0].update(duration='du.missing'),
            'never a number': lambda r: r['skeletons'][1]['links'][0].update(lag=5),
            'unknown planning key': at('durations', 0, basis='per_unit',
                                       driver={'planning_key': 'excavation_volume', 'unit': 'm3'}),
            'unknown Profile condition': at('skeletons', 1, applies_when={'cond': 'approval_pathway', 'eq': 'da'}),
            'not a valid option of planning': at('skeletons', 1, applies_when={'cond': 'planning', 'eq': 'council'}),
            'not a valid work type': at('skeletons', 1, applies_when={'det': 'work_type', 'eq': 'demolish'}),
            'unknown predicate key': at('skeletons', 1, applies_when={'work_type': 'new'}),
            'work_action action': at('fragments', 0, applies_when={
                'work_action': {'system': 'structure', 'any_of': ['retain']}}),
            'needs exactly system': at('fragments', 0, applies_when={'work_action': {'system': 'structure'}}),
            'never a float': at('durations', 0, typical=2.5),
            'at most 2 places': at('durations', 0, typical='2.505'),
            'low <= typical <= high': at('durations', 0, low='3'),
            'instrument, clause and verified': lambda r: r['durations'][1].pop('clause'),
            'links form a cycle': lambda r: r['fragments'][0]['links'].append(
                {'from': 'two', 'to': 'one', 'type': 'SS', 'lag': None}),
            'attach links form a cycle': at('fragments', 1, attach={'before': 'fr.a'}),
            'outside 2027': calendar_day(0, date='2028-03-03'),
            'quoted YYYY-MM-DD': calendar_day(0, date=datetime.date(2027, 3, 3)),
            'is after to': calendar_day(1, **{'from': '2027-12-31', 'to': '2027-12-24'}),
            'needs a kind': calendar_day(0, kind=True),
            'exactly one sk.generic': at('skeletons', 0, id='sk.fallback',
                                         applies_when={'det': 'work_type', 'eq': 'refurb'}),
            'catalogue order': lambda r: r['skeletons'][0]['stages'].reverse(),
            'identical predicate': lambda r: r['skeletons'].append(dict(r['skeletons'][1], id='sk.copy', applies_when={
                'all': [{'cond': 'planning', 'any_of': ['cdc', 'da']}, {'det': 'work_type', 'eq': 'new'}]})),
            'unknown fields': at('fragments', 0, applies_whne={}),
        }
        for needle, mutate in cases.items():
            with self.subTest(needle=needle):
                report = self.run_check(mutate)
                self.assertTrue(any(needle in e for e in report.errors), report.errors)

    def test_program_operands_are_rejected_elsewhere(self):
        for pred in ({'cond': 'planning', 'eq': 'da'}, {'work_action': {'system': 'structure', 'any_of': ['new']}}):
            report = checker.Report()
            checker.check_predicate('test', pred, report, [])
            self.assertTrue(report.errors, pred)


if __name__ == '__main__':
    unittest.main()
