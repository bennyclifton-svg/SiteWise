# Bankstown mechanical title repair

M-203, M-204 and M-205 were filed with the rule-selected title `NIC`.
Their printed titles are MECHANICAL SERVICES LEVEL 1, LEVEL 2 and LEVEL 3
respectively. PDFium returned the rotated title text as letter fragments;
`NIC` was a fragment of MECHANICAL. The existing glyph join only handled
horizontal text, and the title candidate rule accepted the fragment.

The PDF reader now compares fragment adjacency in the glyphs' reading
orientation. It retains word-sized gaps as spaces, checks matching text
orientation, and keeps original page coordinates for provenance. There is
no new dependency, Jev call, prompt change or acceptance-threshold change.

Verification:

- The new fragmentation regression failed at 90, 180 and 270 degrees before
  the fix. It now passes at all four orientations, including word boundaries
  and rejection of different orientations, separate rows/cells and reverse order.
- All Go tests pass with the dedicated test database. The identity speed gate
  passed at p50/p90 5.536/8.026 ms over 20 fixture samples (80/250 ms budget).
- Direct extraction of all seven original mechanical PDFs recovers the three
  corrected titles and preserves the first four titles.
- Seven real-sheet samples measured extraction p50/p90 35.466/47.940 ms and
  candidate harvesting 2.519/9.919 ms. These are small-sample diagnostics,
  not a whole-intake or broad-corpus accuracy claim.
- The local port-8080 app was rebuilt and restarted with the verified binary.
  The three titles were recomputed from the original PDFs and saved as rule
  decisions through CommitFiling, with existing decision-version and user
  correction protections. Before/after checks cover all seven documents:
  only the three title values changed. Authenticated API reads confirm the
  corrected titles and filed status. Private backups remain in `.tools/`.

Broad corpus replay was not rerun for this focused repair; the previously
documented missing-recording limitation still applies.
