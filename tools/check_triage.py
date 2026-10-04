#!/usr/bin/env python
"""Validate a K4 pass A triage file against the dataset and the knowledge base.

Checks data/unforeseen/<dataset>_triage.csv: every dataset row appears exactly
once, enum values are valid, system ids exist and are not deprecated, anchors
come from the fixed list, clusters exist (or `delivery`), duplicate rows refer
to real rows other than the row itself, and similar_existing ids exist.
Also checks knowledge/works/system_terms.yaml when it is present.

    python tools/check_triage.py [--dataset batch1]

Exits 1 when any error is found.
"""
import argparse
import csv
import glob
import os
import sys

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
COLUMNS = ['row_id', 'kind', 'category', 'side_a', 'side_b', 'categories', 'work_types',
           'existing_building', 'cluster', 'dup_rows', 'similar_existing', 'note']
KINDS = {'unforeseen_condition', 'design_or_coordination_error', 'workmanship_defect',
         'process_authority_supply_weather'}
CATEGORIES = {'ground_and_site', 'existing_structure', 'hazardous_materials',
              'concealed_services_and_earlier_work', 'existing_systems_on_test',
              'authorities_and_utilities', 'third_parties_and_occupation', 'design_and_scope',
              'supply_and_site_operations'}
ANCHORS = {'programme', 'contract', 'procurement', 'design', 'authority', 'utility', 'weather',
           'neighbour', 'occupant', 'commissioning', 'handover', 'budget', 'insurance',
           'site_operations'}
BUILDING_CATEGORIES = {'residential', 'commercial', 'industrial'}
WORK_TYPES = {'new', 'extend', 'refurb', 'remediation'}


def load_yaml(path):
    with open(path, encoding='utf-8') as f:
        return yaml.safe_load(f)


def active_systems():
    """Return the ids of non-deprecated systems; deprecated ones are tracked separately."""
    paths = [os.path.join(ROOT, 'knowledge', 'systems.yaml')]
    paths += sorted(glob.glob(os.path.join(ROOT, 'knowledge', 'clusters', '*', 'systems.yaml')))
    active, deprecated = set(), set()
    for p in paths:
        for s in load_yaml(p).get('systems') or []:
            (deprecated if s.get('status') == 'deprecated' else active).add(s['id'])
    return active, deprecated


def record_ids():
    """Existing failure mode and interface ids from every cluster."""
    ids = set()
    for kind in ('failure_modes', 'interfaces'):
        for p in glob.glob(os.path.join(ROOT, 'knowledge', 'clusters', '*', kind + '.yaml')):
            for r in load_yaml(p).get(kind) or []:
                ids.add(r['id'])
    return ids


def cluster_dirs():
    base = os.path.join(ROOT, 'knowledge', 'clusters')
    return {d for d in os.listdir(base) if os.path.isdir(os.path.join(base, d))}


def split(value):
    return [v for v in value.split(';') if v] if value else []


def check_enum_list(errors, rid, field, value, allowed):
    items = split(value)
    if not items:
        errors.append(f'{rid}: {field} is empty')
    for v in items:
        if v not in allowed:
            errors.append(f'{rid}: {field} has invalid value {v!r}')
    if len(items) != len(set(items)):
        errors.append(f'{rid}: {field} repeats a value')


def check_triage(dataset, errors):
    src = os.path.join(ROOT, 'data', 'unforeseen', dataset + '.csv')
    tri = os.path.join(ROOT, 'data', 'unforeseen', dataset + '_triage.csv')
    with open(src, encoding='utf-8', newline='') as f:
        source_ids = [r['row_id'] for r in csv.DictReader(f)]
    with open(tri, encoding='utf-8', newline='') as f:
        reader = csv.DictReader(f)
        if reader.fieldnames != COLUMNS:
            errors.append(f'triage columns are {reader.fieldnames}, expected {COLUMNS}')
            return
        rows = list(reader)
    seen = {}
    for r in rows:
        if r['row_id'] in seen:
            errors.append(f"{r['row_id']}: appears more than once")
        seen[r['row_id']] = r
    for rid in source_ids:
        if rid not in seen:
            errors.append(f'{rid}: missing from triage')
    for rid in seen:
        if rid not in set(source_ids):
            errors.append(f'{rid}: not a row of {dataset}.csv')

    active, deprecated = active_systems()
    records = record_ids()
    clusters = cluster_dirs()
    all_rows = set(source_ids)
    for rid, r in seen.items():
        if r['kind'] not in KINDS:
            errors.append(f"{rid}: invalid kind {r['kind']!r}")
        if r['category'] not in CATEGORIES:
            errors.append(f"{rid}: invalid category {r['category']!r}")
        for side in ('side_a', 'side_b'):
            v = r[side]
            if v in ANCHORS or v in active:
                continue
            if v in deprecated:
                errors.append(f'{rid}: {side} {v!r} is deprecated')
            else:
                errors.append(f'{rid}: {side} {v!r} is not a system id or anchor')
        check_enum_list(errors, rid, 'categories', r['categories'], BUILDING_CATEGORIES)
        check_enum_list(errors, rid, 'work_types', r['work_types'], WORK_TYPES)
        if r['existing_building'] not in ('yes', 'no'):
            errors.append(f"{rid}: existing_building must be yes or no")
        if r['cluster'] != 'delivery' and r['cluster'] not in clusters:
            errors.append(f"{rid}: cluster {r['cluster']!r} is not a cluster directory")
        both_anchors = r['side_a'] in ANCHORS and r['side_b'] in ANCHORS
        if both_anchors and r['cluster'] != 'delivery':
            errors.append(f'{rid}: both sides are anchors so cluster must be delivery')
        if not both_anchors and r['cluster'] == 'delivery':
            errors.append(f'{rid}: a side is physical so cluster cannot be delivery')
        for d in split(r['dup_rows']):
            if d == rid:
                errors.append(f'{rid}: dup_rows contains itself')
            elif d not in all_rows:
                errors.append(f'{rid}: dup_rows {d} is not a row of {dataset}.csv')
            elif rid not in split(seen[d]['dup_rows']):
                errors.append(f'{rid}: dup_rows {d} does not list {rid} back')
        for s in split(r['similar_existing']):
            if s not in records:
                errors.append(f'{rid}: similar_existing {s!r} is not an existing fm or if id')


def check_terms(errors):
    path = os.path.join(ROOT, 'knowledge', 'works', 'system_terms.yaml')
    if not os.path.exists(path):
        return
    active, _ = active_systems()
    data = load_yaml(path)
    if data.get('version') != 1:
        errors.append('system_terms.yaml: version must be 1')
    seen = set()
    for t in data.get('terms') or []:
        name = t.get('term')
        if name in seen:
            errors.append(f'system_terms.yaml: term {name!r} listed twice')
        seen.add(name)
        if t.get('status') != 'draft':
            errors.append(f'system_terms.yaml: {name!r} status must be draft')
        ids = t.get('ids') or []
        if not ids:
            errors.append(f'system_terms.yaml: {name!r} has no ids')
        for i in ids:
            if i not in ANCHORS and i not in active:
                errors.append(f'system_terms.yaml: {name!r} maps to unknown id {i!r}')


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('--dataset', default='batch1')
    args = ap.parse_args()
    errors = []
    check_triage(args.dataset, errors)
    check_terms(errors)
    for e in errors:
        print('ERROR', e)
    print(f'check_triage: {len(errors)} errors')
    return 1 if errors else 0


if __name__ == '__main__':
    sys.exit(main())
