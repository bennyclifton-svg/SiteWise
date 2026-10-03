"""Exercise real drawing-set uploads against independently authored page labels.

Requires pypdf for independent download validation. This script calls the local
development app, which may call Jev. Originals and results stay in private QA
projects. Gold JSON: {source: absolute PDF path, pages: [{page: 1, number: ...,
revision: ..., title: ..., date: ..., kind: ..., discipline: ..., lifecycle: ...}]}.
An empty pages list expects an intact report. Never derive labels from results.
"""
import argparse
import hashlib
import io
import json
import time
import urllib.parse
import urllib.request
import http.cookiejar
from pathlib import Path

from pypdf import PdfReader


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--gold', required=True)
    parser.add_argument('--out', required=True)
    parser.add_argument('--base', default='http://127.0.0.1:8080')
    parser.add_argument('--allow-live-jev', action='store_true', required=True)
    args = parser.parse_args()
    if urllib.parse.urlparse(args.base).hostname not in ('127.0.0.1', 'localhost'):
        parser.error('development localhost only')
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=False)
    gold = json.loads(Path(args.gold).read_text(encoding='utf-8'))
    source = Path(gold['source'])
    original = source.read_bytes()
    page_count = len(PdfReader(io.BytesIO(original)).pages)
    if gold['pages'] and [p['page'] for p in gold['pages']] != list(range(1, page_count + 1)):
        parser.error('gold must independently label every physical sheet in order')
    (out / 'gold.json').write_text(json.dumps(gold, indent=2), encoding='utf-8')
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def request(path, data=None):
        return opener.open(urllib.request.Request(args.base + path, data=data,
                           headers={'Origin': args.base, 'Content-Type': 'application/json'}), timeout=60)

    request('/dev/login').close()
    project = json.load(request('/api/projects', json.dumps({'name': 'Sheet QA ' + source.stem}).encode()))['id']
    route = '/api/projects/' + project + '/files?name=' + urllib.parse.quote(source.name)
    start = time.monotonic()
    uploaded = json.load(request(route, original))
    (out / 'upload.json').write_text(json.dumps({'project': project, 'document': uploaded}, indent=2), encoding='utf-8')
    deadline = time.monotonic() + 600
    while True:
        listing = json.load(request('/api/projects/' + project + '/documents'))
        docs = listing['documents']
        if docs and all(d['status'] != 'pending' and d.get('expansion', {}).get('status') != 'pending' for d in docs):
            break
        if time.monotonic() > deadline:
            break
        time.sleep(.25)
    elapsed = time.monotonic() - start
    (out / 'documents.json').write_text(json.dumps(listing, indent=2), encoding='utf-8')
    failures = []
    parent = next((d for d in docs if d['id'] == uploaded['id']), None)
    children = sorted((d for d in docs if d.get('source_id') == uploaded['id']), key=lambda d: d['sheet_page'])
    if gold['pages']:
        if not parent or parent['status'] != 'split' or len(children) != page_count:
            failures.append('drawing set not completely split')
    elif len(docs) != 1 or children or not parent or parent.get('expansion'):
        failures.append('report did not remain one artifact')
    for doc in docs:
        body = request('/api/documents/' + doc['id'] + '/file').read()
        if doc['id'] == uploaded['id']:
            if hashlib.sha256(body).digest() != hashlib.sha256(original).digest():
                failures.append('original bytes changed')
        elif len(PdfReader(io.BytesIO(body)).pages) != 1:
            failures.append('child download is not one page')
    mismatches = []
    for expected in gold['pages']:
        child = next((d for d in children if d['sheet_page'] == expected['page']), {})
        fields = {f['field']: f['value'] for f in child.get('fields', [])}
        for field, want in expected.items():
            if field == 'page':
                continue
            # Literal comparison is deliberately conservative. Record equivalent
            # date/revision forms as aliases in gold instead of hiding mismatches.
            accepted = want if isinstance(want, list) else [want]
            if fields.get(field, '') not in accepted:
                mismatches.append({'page': expected['page'], 'field': field, 'expected': want, 'actual': fields.get(field, '')})
    duplicate = json.load(request(route, original))
    repeated = json.load(request('/api/projects/' + project + '/documents'))
    if duplicate['id'] != uploaded['id'] or {d['id'] for d in repeated['documents']} != {d['id'] for d in docs}:
        failures.append('duplicate upload changed document identities')
    result = {'project': project, 'source_pages': page_count, 'children': len(children),
              'elapsed_seconds': elapsed, 'flow_failures': failures, 'field_mismatches': mismatches}
    (out / 'result.json').write_text(json.dumps(result, indent=2), encoding='utf-8')
    print(json.dumps(result, indent=2))
    return 1 if failures or mismatches else 0


if __name__ == '__main__':
    raise SystemExit(main())
