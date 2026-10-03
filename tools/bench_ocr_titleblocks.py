"""Offline Bankstown crop benchmark; never changes PDFs or application data.

Requires Pillow and pypdfium2, plus a Tesseract executable and eng.traineddata.
The fixed, visually inspected crop regions are specific to this drawing template.
"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import re
import subprocess
import time

from PIL import Image
import pypdfium2 as pdfium


# Normalised displayed-page coordinates: left, top, right, bottom.
REGIONS = [(0.855, 0.031, 0.987, 0.085), (0.855, 0.735, 0.987, 0.975)]

# Transcribed from rendered printed cells, not inferred from filenames.
# CC-05 really prints September 7; do not silently swap it to July 9.
EXPECTED = {
    '01': ('SITE SETOUT PLAN', 'D', '09/07/2015'),
    '02': ('BASEMENT 2', 'F', '09/07/2015'),
    '03': ('BASEMENT 1', 'F', '09/07/2015'),
    '04': ('GROUND FLOOR PLAN', 'J', '14/07/2015'),
    '05': ('LEVEL 1', 'H', '07/09/2015'),
}


def evidence_checks(filename, text):
    number = re.search(r'CC-(\d{2})', filename).group(1)
    title, revision, date = EXPECTED[number]
    return {
        'title_present': title in ' '.join(text.split()),
        'number_present': bool(re.search(r'\bCC\s*-\s*' + number + r'\b', text)),
        'revision_label_and_value': bool(re.search(r'\bREVISION\s+' + revision + r'\b', text, re.I)),
        'current_revision_date_row': bool(re.search(r'^' + revision + r'\s+' + re.escape(date) + r'\b', text, re.M)),
    }


def render(path, dpi, target):
    with pdfium.PdfDocument(path) as doc:
        page = doc[0]
        width, height = page.get_size()
        pieces = []
        for left, top, right, bottom in REGIONS:
            bitmap = page.render(
                scale=dpi / 72, grayscale=True,
                crop=(left * width, (1 - bottom) * height,
                      (1 - right) * width, top * height),
            )
            pieces.append(bitmap.to_pil().copy())
            bitmap.close()
        page.close()
    # White gutters separate the issue table from the lower identity block.
    combined = Image.new('L', (max(p.width for p in pieces) + 40,
                               sum(p.height for p in pieces) + 60), 255)
    y = 20
    for piece in pieces:
        combined.paste(piece, (20, y))
        y += piece.height + 20
    combined.save(target, dpi=(dpi, dpi))
    return combined.size


def percentile(values, p):
    return sorted(values)[max(0, math.ceil(len(values) * p) - 1)]


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--corpus', type=Path, required=True)
    ap.add_argument('--tesseract', type=Path, required=True)
    ap.add_argument('--tessdata', type=Path, required=True)
    ap.add_argument('--output', type=Path, required=True)
    ap.add_argument('--dpi', type=int, nargs='+', default=[150, 200, 300])
    ap.add_argument('--psm', type=int, default=6)
    ap.add_argument('--repeats', type=int, default=3)
    args = ap.parse_args()
    if args.repeats < 1 or any(d < 72 or d > 600 for d in args.dpi):
        ap.error('repeats must be positive and DPI must be between 72 and 600')
    paths = sorted(args.corpus.glob('1115 CC-0[1-5] *.pdf'))
    if len(paths) != 5:
        ap.error(f'expected five Bankstown originals, found {len(paths)}')
    args.output.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, OMP_THREAD_LIMIT='1')
    result = {
        'platform': platform.platform(), 'processor': platform.processor(),
        'pdfium': str(pdfium.PDFIUM_INFO), 'pypdfium2': str(pdfium.PYPDFIUM_INFO),
        'tesseract': subprocess.check_output([str(args.tesseract), '--version'], text=True),
        'model_sha256': hashlib.sha256((args.tessdata / 'eng.traineddata').read_bytes()).hexdigest(),
        'input_sha256': {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in paths},
        'regions': REGIONS, 'psm': args.psm, 'omp_thread_limit': 1,
        'method': 'Sequential; reopen PDF and launch OCR per sample; no warm-up discarded. OS cache may be warm. Includes PNG encoding and OCR startup; excludes Jev, database and upload.',
        'samples': [], 'summary': {},
    }
    for repeat in range(args.repeats):
        for path in paths:
            # Rotate resolution order to avoid always favouring one warm-cache setting.
            dpis = args.dpi[repeat % len(args.dpi):] + args.dpi[:repeat % len(args.dpi)]
            for dpi in dpis:
                stem = f'{path.name.split()[1]}-{dpi}-{repeat}'
                crop = args.output / f'{stem}.png'
                started = time.perf_counter()
                size = render(path, dpi, crop)
                rendered = time.perf_counter()
                proc = subprocess.run([
                    str(args.tesseract), str(crop), 'stdout',
                    '--tessdata-dir', str(args.tessdata), '-l', 'eng',
                    '--oem', '1', '--psm', str(args.psm), '--dpi', str(dpi),
                ], capture_output=True, text=True, encoding='utf-8', env=env,
                    check=True, timeout=60)
                ended = time.perf_counter()
                (args.output / f'{stem}.txt').write_text(proc.stdout, encoding='utf-8')
                sample = dict(file=path.name, dpi=dpi, repeat=repeat, size=size,
                              render_ms=(rendered-started)*1000,
                              ocr_ms=(ended-rendered)*1000, total_ms=(ended-started)*1000,
                              text=proc.stdout, stderr=proc.stderr)
                sample['evidence_checks'] = evidence_checks(path.name, proc.stdout)
                result['samples'].append(sample)
                print(f'{stem}: render={sample["render_ms"]:.0f} OCR={sample["ocr_ms"]:.0f} total={sample["total_ms"]:.0f} ms', flush=True)
                (args.output / 'results.json').write_text(json.dumps(result, indent=2), encoding='utf-8')
    for dpi in args.dpi:
        samples = [s for s in result['samples'] if s['dpi'] == dpi]
        result['summary'][str(dpi)] = {
            metric: {f'p{int(p*100)}': round(percentile([s[metric] for s in samples], p), 1)
                     for p in [0.5, 0.9]}
            for metric in ['render_ms', 'ocr_ms', 'total_ms']
        }
        result['summary'][str(dpi)]['samples'] = len(samples)
        result['summary'][str(dpi)]['all_four_evidence_checks_pass'] = sum(
            all(s['evidence_checks'].values()) for s in samples)
    (args.output / 'results.json').write_text(json.dumps(result, indent=2), encoding='utf-8')
    print(json.dumps(result['summary'], indent=2))


if __name__ == '__main__':
    main()
