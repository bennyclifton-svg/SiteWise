"""150 DPI Bankstown-template OCR trial. Outputs evidence, never field answers."""
import argparse
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

from bench_ocr_titleblocks import REGIONS, render
import pypdfium2 as pdfium


def extract(path, tesseract, tessdata):
    started = time.perf_counter()
    with pdfium.PdfDocument(path) as doc:
        if len(doc) != 1:
            raise ValueError('trial supports one-page drawings only')
        page = doc[0]
        width, height = page.get_size()
        page.close()
    if abs(width - 2384) > 5 or abs(height - 1684) > 5:
        raise ValueError('drawing does not match trial landscape template')
    with tempfile.TemporaryDirectory(prefix='sitewise-ocr-') as tmp:
        image = Path(tmp) / 'crops.png'
        render(path, 150, image)
        rendered = time.perf_counter()
        proc = subprocess.run([str(tesseract), str(image), 'stdout', '--tessdata-dir',
                               str(tessdata), '-l', 'eng', '--oem', '1', '--psm', '6',
                               '--dpi', '150', '-c', 'tessedit_create_tsv=1'],
                              env=dict(os.environ, OMP_THREAD_LIMIT='1'),
                              capture_output=True, text=True, encoding='utf-8',
                              check=True, timeout=15)
    rows = list(csv.DictReader(io.StringIO(proc.stdout), delimiter='\t', quoting=csv.QUOTE_NONE))
    lines = {}
    for row in rows:
        if row['level'] == '5' and row['text'].strip():
            key = tuple(row[k] for k in ['block_num', 'par_num', 'line_num'])
            lines.setdefault(key, []).append(row)
    runs = []
    scale = 150 / 72
    # Map the stacked bitmap back to the original displayed page coordinates.
    split = 20 + round((REGIONS[0][3] - REGIONS[0][1]) * height * scale)
    for words in lines.values():
        groups = [[]]
        for word in words:
            if groups[-1]:
                last = groups[-1][-1]
                gap = int(word['left']) - int(last['left']) - int(last['width'])
                # This template's title cell is physically right of its caption.
                # Split at the cell boundary even when OCR merges the caption,
                # border marks and value into one line. Never repair its letters.
                page_y = height - REGIONS[1][1]*height - (int(word['top']) - split - 20)/scale
                page_x = REGIONS[1][0]*width + (int(word['left']) - 20)/scale
                last_x = REGIONS[1][0]*width + (int(last['left']) - 20)/scale
                title_boundary = 135 < page_y < 170 and last_x < 2115 <= page_x
                if title_boundary or gap > max(int(word['height']), int(last['height'])) * 1.5:
                    groups.append([])
            groups[-1].append(word)
        for group in groups:
            left = min(int(w['left']) for w in group)
            top = min(int(w['top']) for w in group)
            right = max(int(w['left']) + int(w['width']) for w in group)
            bottom = max(int(w['top']) + int(w['height']) for w in group)
            region = 0 if top < split else 1
            offset = 20 if region == 0 else split + 20
            runs.append({'Text': ' '.join(w['text'] for w in group), 'Source': {
                'Page': 1, 'X': REGIONS[region][0] * width + (left - 20) / scale,
                'Y': height - REGIONS[region][1] * height - (bottom - offset) / scale,
                'Width': (right-left) / scale, 'Height': (bottom-top) / scale,
            }})
    return {'sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
            'model_sha256': hashlib.sha256((tessdata/'eng.traineddata').read_bytes()).hexdigest(),
            'render_ms': (rendered-started)*1000,
            'total_ms': (time.perf_counter()-started)*1000,
            'text': {'Format': 'pdf', 'PageCount': 1, 'TextLayer': False, 'Runs': runs}}


if __name__ == '__main__':
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--pdf', type=Path, required=True)
    ap.add_argument('--tesseract', type=Path, required=True)
    ap.add_argument('--tessdata', type=Path, required=True)
    args = ap.parse_args()
    print(json.dumps(extract(args.pdf, args.tesseract, args.tessdata)))
