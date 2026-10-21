# Visual document identity

Outcome: reduce the work of checking drawing and report titles and revision labels, using source images beside literal proposed values. Lane A, with authenticated evidence access (lane C). This extends the successful Hale experiment; its accuracy is not a general calibration.

One background Jev fan-out selects source candidates for title, number and revision. Native filing remains first. Drawing evidence includes overlapping bottom/right regions; other PDFs use the first three pages. No consultant templates. A suggestion is never an accepted decision: the existing correction action is the only application path. No Box, automatic supersession, specification interpretation, financial extraction, or new AI provider.

Budgets: foreground filing p50 1000 ms / p90 2000 ms. Visual preparation and read target p50 10000 ms / p90 20000 ms, measured excluding queue delay. One worker serializes this with OCR. Bounded three images, 32 MiB PDF, 20 second job deadline. Reuse installed PDFium, Pillow and Tesseract; no added dependency or service. Store bounded review evidence with its document, deleted by cascade, so authenticated previews do not need a second file lifecycle.

Done: new PDF filings queue review; existing PDFs can request it; saved corrections remain unchanged; suggestions show their source and can be explicitly used. Test malformed/missing files, unavailable renderer/provider, wrong org, repeated requests, rotated pages, multiline source text and report revision tables. Unresolved values stay unresolved; first-three-page coverage is explicit.

Provider contract: [images](https://docs.typesafe.ai/images), [fan-out](https://docs.typesafe.ai/patterns/fan-out), [confidence](https://docs.typesafe.ai/confidence), [API](https://docs.typesafe.ai/api). Image calls use pinned jev-1.14.0 and the experimental header, while existing text calls retain their model and admission reserve. The failed [Box](https://docs.typesafe.ai/primitives/box) trial does not justify using it here.
