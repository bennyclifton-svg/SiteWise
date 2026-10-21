"""Read bounded image pages for a proposal; native text stays in Go."""
import concurrent.futures
import json
import math
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

import pypdfium2 as pdfium
from PIL import Image

source, executable, tessdata, selected = sys.argv[1:]
pages = json.loads(selected)
if Path(source).stat().st_size > 32 * 1024 * 1024 or len(pages) > 4:
    sys.exit(3)
scale = 200 / 72
env = dict(os.environ, OMP_THREAD_LIMIT="1")
osd_data = next((p for p in [Path(tessdata), Path(executable).parent / "tessdata"] if (p / "osd.traineddata").is_file()), None)

def recognise(job):
    number, path, width, height = job
    if osd_data:
        orientation = subprocess.run([executable, path, "stdout", "--psm", "0", "--tessdata-dir", str(osd_data)], capture_output=True, timeout=5, env=env)
        rotate = re.search(rb"Rotate: (90|180|270)\b", orientation.stdout)
        if rotate:
            degrees = int(rotate[1])
            with Image.open(path) as original:
                original.rotate(-degrees, expand=True).save(path)
            if degrees != 180:
                width, height = height, width
    args = [executable, path, "stdout", "-l", "eng", "--psm", "11", "--dpi", "200"]
    if tessdata:
        args += ["--tessdata-dir", tessdata]
    args += ["-c", "tessedit_create_tsv=1", "-c", "tessedit_create_txt=0"]
    result = subprocess.run(args, capture_output=True, timeout=12, env=env, check=True)
    if len(result.stdout) > 500_000:
        raise ValueError("OCR output exceeds page limit")
    return dict(page=number, width=width, height=height, tsv=result.stdout.decode("utf-8"))

with tempfile.TemporaryDirectory(prefix="sitewise-proposal-") as temp:
    jobs = []
    with pdfium.PdfDocument(source) as doc:
        for number in pages:
            if number < 1 or number > len(doc):
                raise ValueError("page outside PDF")
            page = doc[number-1]
            width, height = page.get_size()
            if not all(math.isfinite(v) and v > 0 for v in (width, height)) or width*height*scale**2 > 12_000_000:
                raise ValueError("page exceeds render limit")
            bitmap = page.render(scale=scale, grayscale=True)
            target = str(Path(temp) / f"{number}.png")
            bitmap.to_pil().save(target, compress_level=1)
            bitmap.close()
            page.close()
            jobs.append((number, target, width, height))
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
        print(json.dumps(list(pool.map(recognise, jobs))))
