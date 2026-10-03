# Small-batch intake hardening

Work one consultant layout at a time. Select two or three PDFs per folder;
exclude DWGs and AppleDouble sidecars. A folder containing drawings and reports
has multiple layouts: a drawing pass does not validate its reports.

1. Copy selected files into a private batch input directory. Preserve originals.
2. Independently read/render the identity page and record all six requested
   fields, plus lifecycle, in the existing gold format. Record source conflicts.
   Never populate the reference labels from SiteWise's predictions.
3. Save the baseline through the running app. Use a fresh project: same-project
   hash deduplication returns old results and does not test changed code.
4. Minimise a failure into a test, observe it fail, fix the supported pattern,
   and repeat the actual PDFs. Do not add consultant names or file hashes to
   production extraction rules.
5. Add the batch to the cumulative manifest, including unresolved failures.
   Rebuild the evaluator and app from the same version, restart the local app,
   and rerun all earlier labelled batches before advancing.
6. Verify representative uploads with the browser file chooser and rendered
   fields. API uploads test the production pipeline but not the browser input
   interaction. Both are necessary; neither justifies unlabelled accuracy claims.

Run from the repository root, using a new output directory each time:

```powershell
python tools/check-folder-batches.py `
  --manifest data/eval/intake/private/folder-batches/manifest.json `
  --out data/eval/intake/private/folder-batches/round-UNIQUE `
  --allow-live-jev
```

This sends identity text to the locally configured Jev provider and creates QA
projects. It never corrects or deletes existing user documents. The manifest is
an array of objects with `name`, `roots` and `gold` paths. Only add independently
labelled batches. Results preserve per-file predictions and aggregate failures.

Report complete documents as well as per-field counts. Missing values and wrong
amber suggestions are failures. Keep unextractable text, conflicting source
revisions and speed-budget failures explicit. Do not reinterpret a CAD export
timestamp as an issue date. Current drawing-pack tests inspect the first-page
identity only, as the production intake does; they do not verify every sheet.

When changing the PDF reader or its glyph grouping, re-extract the older labelled
corpus before comparing previously correct fields. Replaying saved text checks
the candidate parser only and cannot detect a reader regression. Keep the fresh
traces and the comparison result alongside the live upload round.

The parser changes do not alter Jev's one-call fan-out or enable other runtime
AI. Literal candidates and conservative confidence routing follow
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)
and [confidence routing](https://docs.typesafe.ai/patterns/confidence-routing).
