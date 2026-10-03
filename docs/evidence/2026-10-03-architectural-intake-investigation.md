# Bankstown architectural intake investigation

The five reported uploads are `1115 CC-01 SETOUT PLAN D.pdf`,
`1115 CC-02 BASEMENT 2 F.pdf`, `1115 CC-03 BASEMENT 1 F.pdf`,
`1115 CC-04 GROUND FLOOR J.pdf` and `1115 CC-05 LEVEL 01 H.pdf`.
Their stored SHA-256 hashes match the corresponding delivery-bankstown corpus
files. All five database rows have status `not_filed`, reason `no_text_layer`.

The current production identity reader returns one page, no error, zero runs
and `TextLayer: false` for each original. Independent inspection with pypdf
also returns no text. Each page has zero font resources, zero text operators
(`BT`, `Tf`, `Tj`, `TJ`, single/double quote), no annotations, no form fields
and no attachments. Their XObjects are images, with additional inline images
on CC-04 and CC-05, but the pages also contain extensive vector paths. Calling
them image-only scans is inaccurate: lettering can be outlined as paths.
CC-02 contains 591,151 line-to commands and 22,887 even-odd path fills, with
no text-showing operations. There are no nested Form XObjects carrying hidden text.
PDF metadata contains a source-file path, which is not printed title-block
evidence and cannot establish the current issue date.

The runner therefore stops before candidate harvesting and Jev classification,
as required by the foundation design. This is not a Jev timeout or an identity
read-limit failure. No OCR, additional provider, prompt change or acceptance
threshold change was introduced. The owner explicitly declined OCR.

The register did make this failure misleading: it silently displayed the
filename as the title and hid the explanatory reason inside the expanded row.
It now distinguishes an absent title from `File: …` and shows the no-readable-text
reason in the collapsed row. The expanded explanation identifies the affected
metadata and describes the selectable-text PDF recovery path.

Verification: the existing browser intake regression failed on the old filename
fallback, then passed with assertions for the missing title, labelled filename
and visible not-filed reason. The TypeScript/Vite build and diff whitespace
check pass. No backend hot-path work or runtime dependency was added.

This fixes the misleading presentation, not automatic recovery of these five
drawings. The owner supplied copied title-block text (`DRAWING NAME BASEMENT 2`,
`1115`, `cc- 02`) from the Windows web browser. Browser-side recognition is a
possible explanation for selectable text despite the lack of stored PDF text;
the browser itself was not accessible through the connected browser tools, so
that mechanism is not confirmed. Do not claim the missing metadata has been
repaired or that the owner cannot copy visible text.

## Local Tesseract installation

The owner subsequently authorised installing Tesseract, prioritising speed.
This authorises local OCR evaluation; the upload pipeline has not been changed
to run OCR automatically.

- Installed `UB-Mannheim.TesseractOCR` version `5.4.0.20240606` through winget,
  using the Windows distribution linked by the
  [official installation documentation](https://tesseract-ocr.github.io/tessdoc/Installation.html).
- Executable: `C:\Program Files\Tesseract-OCR\tesseract.exe`.
- Fast English model: `.tools/tesseract/tessdata/eng.traineddata`, downloaded
  from the official `tessdata_fast` repository at commit
  `87416418657359cb625c412a48b6e1d6d41c29bd`.
- Model SHA-256:
  `7d4322bd2a7749724879683fc3912cb542f19906c83bcc1a52132556427170b2`.
- Select this model explicitly with `--tessdata-dir .tools/tesseract/tessdata
  -l eng --oem 1`; the system installation's default model is separate.

The version and language-list checks passed. A recognition smoke test on the
existing `bank-arch01.png` render completed successfully in 3.05 seconds,
recognising `SITE SETOUT PLAN`, `1115`, `CC- 01` and revision `D`, among noisy
output. The engine warned about the render's low resolution. This single test
excludes PDF rendering and Jev classification and is neither an accuracy
validation nor a p50/p90 benchmark. No claim that Tesseract is faster than the
alternatives is established. The local binaries, model and smoke output remain
outside version control in their installation directory and `.tools/`.
