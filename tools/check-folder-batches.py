"""Rerun every independently labelled folder batch through the local app.

Manifest: [{"name": "batch-name", "roots": "path.json", "gold": "path.json"}].
Each gold file must label all six requested fields (and lifecycle for the
existing evaluator). Never edit gold from predictions. Keep private manifests,
PDF copies and results under data/eval/intake/private.
"""
import argparse
import json
import hashlib
from pathlib import Path
import shutil
import subprocess
import sys


def freeze_run(out, evaluator_source, data_source, batches):
    evaluator = out / 'evaluator.exe'
    shutil.copyfile(evaluator_source, evaluator)
    config = out / 'config'
    config.mkdir()
    captured = [evaluator]
    for name in ('thresholds.json', 'kinds.json', 'disciplines.json', 'lifecycle.json'):
        target = config / name
        shutil.copyfile(Path(data_source) / name, target)
        captured.append(target)
    frozen = []
    for index, batch in enumerate(batches, 1):
        folder = out / 'inputs' / f'{index:03}'
        folder.mkdir(parents=True)
        entry = {'name': batch['name']}
        for key in ('roots', 'gold'):
            target = folder / (key + '.json')
            shutil.copyfile(batch[key], target)
            captured.append(target)
            entry[key] = str(target.resolve())
        frozen.append(entry)
    hashes = {str(p.relative_to(out)): hashlib.sha256(p.read_bytes()).hexdigest() for p in captured}
    (out / 'source-manifest.json').write_text(json.dumps(batches, indent=2), encoding='utf8')
    (out / 'manifest.json').write_text(json.dumps(frozen, indent=2), encoding='utf8')
    (out / 'snapshot.json').write_text(json.dumps(hashes, indent=2), encoding='utf8')
    return evaluator, config, frozen


def verify_classifier_version(results, expected):
    observed = set()
    for entry in results:
        docs = [entry.get('Document'), entry.get('SourceDocument'), *(entry.get('SheetDocuments') or [])]
        for doc in docs:
            for field in (doc or {}).get('fields') or []:
                if field.get('decided_by') != 'jev':
                    continue
                version = field.get('question_version')
                if version != expected:
                    raise RuntimeError(f'Live classifier version {version!r}, expected {expected!r}; run rejected, not an accuracy result')
                observed.add(version)
    return sorted(observed)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--manifest', required=True)
    parser.add_argument('--out', required=True, help='new evidence directory')
    parser.add_argument('--evaluator', default='.tools/corpus-eval.exe')
    parser.add_argument('--data', default='data/intake', help='configuration to freeze for this run')
    parser.add_argument('--base', default='http://127.0.0.1:8080')
    parser.add_argument('--allow-live-jev', action='store_true', required=True)
    args = parser.parse_args()
    batches = json.loads(Path(args.manifest).read_text(encoding='utf8'))
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=False)
    # A long run must not pick up a newly built evaluator or experimental
    # thresholds between folders. Retain exactly what this run started with.
    evaluator, config, batches = freeze_run(out, args.evaluator, args.data, batches)
    expected_version = json.loads((config / 'thresholds.json').read_text(encoding='utf8'))['question_version']
    results = []
    for index, batch in enumerate(batches, 1):
        target = out / f'{index:03}'
        target.mkdir()
        common = [str(evaluator.resolve()), '-data', str(config.resolve()), '-roots', batch['roots'], '-gold', batch['gold'], '-per-folder', '0']
        # Capture fresh extraction evidence; rules-only deliberately cannot
        # pass kind/discipline, which require the app's configured Jev provider.
        with (target / 'rules.log').open('w', encoding='utf8') as log:
            subprocess.run(common + ['-mode', 'rules', '-out', str(target / 'rules')], stdout=log, stderr=log, check=True)
        subprocess.run([sys.executable, 'tools/corpus-upload.py', '--selected', str(target / 'rules/selected.json'),
                        '--out', str(target / 'app'), '--base', args.base, '--allow-live-jev'], check=True)
        observed = verify_classifier_version(json.loads((target / 'app/results.json').read_text(encoding='utf8')), expected_version)
        (target / 'classifier-version.json').write_text(json.dumps({'expected': expected_version, 'observed': observed}), encoding='utf8')
        with (target / 'score.log').open('w', encoding='utf8') as log:
            score = subprocess.run(common + ['-mode', 'app-score', '-app-results', str(target / 'app/results.json'),
                                            '-out', str(target / 'score'), '-gate'], stdout=log, stderr=log)
        summary_path = target / 'score/summary.json'
        if not summary_path.exists():
            raise RuntimeError(f"Evaluator failed before scoring {batch['name']}; see {target / 'score.log'}")
        summary = json.loads(summary_path.read_text(encoding='utf8'))
        results.append({'batch': batch['name'], 'passed': score.returncode == 0,
                        'summary': summary, 'evidence': str(target)})
        (out / 'results.json').write_text(json.dumps(results, indent=2), encoding='utf8')
        print(f"{batch['name']}: {'PASS' if score.returncode == 0 else 'FAIL'}", flush=True)
    # An unsupported date or scan remains a failure; known limitations never
    # disappear from the denominator or silently become an accepted baseline.
    return 0 if results and all(r['passed'] for r in results) else 1


if __name__ == '__main__':
    sys.exit(main())
