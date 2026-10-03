"""Bounded native identity-page rendering for OCR. Embedded in Go."""
import json
import math
import sys
from pathlib import Path
from PIL import Image
import pypdfium2 as pdfium

source, target = sys.argv[1:]
if Path(source).stat().st_size > 32 * 1024 * 1024:
    sys.exit(3)
scale = 150 / 72
with pdfium.PdfDocument(source) as doc:
    page = doc[0]
    width, height = page.get_size()
    if not all(math.isfinite(x) and x > 0 for x in (width, height)) or width * height * scale**2 > 25_000_000:
        sys.exit(3)
    # Large-format drawings: read the perimeter where identity blocks live.
    # These are generic page bands, not a consultant/template-specific crop.
    regions = [(0, 0, 1, 1)] if max(width, height) <= 1000 else [
        (0, 0, .8, .12), (.8, 0, 1, 1), (0, .8, .8, 1)]
    pieces, mapping = [], []
    top_pixel = 20
    for left, top, right, bottom in regions:
        bitmap = page.render(scale=scale, grayscale=True, crop=(
            left*width, (1-bottom)*height, (1-right)*width, top*height))
        image = bitmap.to_pil().copy()
        bitmap.close()
        pieces.append(image)
        mapping.append(dict(X=left*width, Y=(1-bottom)*height,
            Width=(right-left)*width, Height=(bottom-top)*height,
            Top=top_pixel, PixelHeight=image.height))
        top_pixel += image.height + 20
    canvas = Image.new('L', (max(p.width for p in pieces)+40, top_pixel), 255)
    for image, region in zip(pieces, mapping):
        canvas.paste(image, (20, region['Top']))
        image.close()
    canvas.save(target, compress_level=1)
    print(json.dumps(dict(Width=width, Height=height, Pages=len(doc),
        CanvasWidth=canvas.width/scale, CanvasHeight=canvas.height/scale, Regions=mapping)))
    canvas.close()
    page.close()
