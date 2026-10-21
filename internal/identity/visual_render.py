"""Bounded identity evidence. PDF display rotation precedes all crops."""
import base64, io, json, math, os, subprocess, sys, tempfile
from pathlib import Path
import pypdfium2 as pdfium
from PIL import Image

source, drawing, ocr, executable, tessdata = sys.argv[1:]
if Path(source).stat().st_size > 128 * 1024 * 1024:
    sys.exit(3)
images, lines = [], []
with pdfium.PdfDocument(source) as doc, tempfile.TemporaryDirectory(prefix='sitewise-visual-') as temp:
    first = doc[0]
    is_drawing = drawing == 'true' or max(first.get_size()) > 1000
    first.close()
    count = 1 if is_drawing else min(3, len(doc))
    for index in range(count):
        page = doc[index]
        w, h = page.get_size()
        if not all(math.isfinite(v) and v > 0 for v in (w, h)) or w*h*2.25 > 25_000_000:
            sys.exit(3)
        bitmap = page.render(scale=1.5)
        im = bitmap.to_pil().convert('RGB')
        bitmap.close()
        regions = [('Whole page', (0,0,1,1))]
        if is_drawing:
            regions += [('Bottom title region', (0,.84,1,1)), ('Right title region', (.72,0,1,1))]
        for label, bounds in regions:
            crop = im.crop(tuple(round(v*n) for v,n in zip(bounds,(im.width,im.height,im.width,im.height))))
            textpage = page.get_textpage()
            sparse = textpage.count_chars() < 30
            textpage.close()
            if ocr == 'true' or sparse or (is_drawing and label != 'Whole page'):
                path = str(Path(temp)/'region.png')
                crop.save(path)
                args = [executable,path,'stdout','-l','eng','--psm','11']
                if tessdata: args += ['--tessdata-dir',tessdata]
                result = subprocess.run(args, capture_output=True, timeout=5, check=True, env=dict(os.environ,OMP_THREAD_LIMIT='1'))
                if len(result.stdout)>100_000: raise ValueError('OCR output limit')
                region_lines=[]
                for line in result.stdout.decode('utf-8').splitlines():
                    value=' '.join(line.split())
                    if value and len(value)<=180: region_lines.append(dict(text=value,page=index+1))
                if is_drawing and label != 'Whole page': lines=region_lines+lines
                else: lines+=region_lines
            # Keep both ends of a wide title strip legible. The two overlapping
            # halves are stacked, left above right; never discard an edge.
            if label == 'Bottom title region' and crop.width > crop.height * 2:
                left=crop.crop((0,0,round(crop.width*.55),crop.height))
                right=crop.crop((round(crop.width*.45),0,crop.width,crop.height))
                stacked=Image.new('RGB',(max(left.width,right.width),crop.height*2+12),'white')
                stacked.paste(left,(0,0));stacked.paste(right,(0,crop.height+12))
                crop=stacked
                label='Bottom edge: left half above right half'
            crop.thumbnail((1568,1568))
            padded=Image.new('RGB',(max(280,crop.width),max(280,crop.height)),'white')
            padded.paste(crop,(0,0))
            out=io.BytesIO(); padded.save(out,format='JPEG',quality=92)
            images.append(dict(label=label,page=index+1,data=base64.b64encode(out.getvalue()).decode()))
        page.close()
    print(json.dumps(dict(images=images,lines=lines[:400],pages=count,drawing=is_drawing)))
