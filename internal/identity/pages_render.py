"""Whole pages as JPEG for the tender pass. One JSON object per line."""
import base64, io, json, math, sys
from pathlib import Path
import pypdfium2 as pdfium

source, long_edge, quality, wanted = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4]
if Path(source).stat().st_size > 128 * 1024 * 1024:
    sys.exit(3)
with pdfium.PdfDocument(source) as doc:
    for number in (int(n) for n in wanted.split(",") if n):
        if number < 1 or number > len(doc):
            continue
        page = doc[number - 1]
        w, h = page.get_size()
        if not all(math.isfinite(v) and v > 0 for v in (w, h)):
            page.close()
            continue
        bitmap = page.render(scale=long_edge / max(w, h))
        image = bitmap.to_pil().convert("RGB")
        bitmap.close()
        page.close()
        out = io.BytesIO()
        image.save(out, format="JPEG", quality=quality)
        print(json.dumps(dict(page=number, jpeg=base64.b64encode(out.getvalue()).decode())), flush=True)
