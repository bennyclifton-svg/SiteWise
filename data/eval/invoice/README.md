# Invoice reading evaluation

`cmd/invoice-eval` scores invoice reading against an answer key of real
invoices and fits the cut-offs in `data/invoice/thresholds.json`.

The answer key, the corpus paths and the Jev recording are in `private/`,
which Git ignores: they name real projects, suppliers and fees, and this
repository is public. `results/latest.json` holds aggregate counts only.

- `private/answer-key.json`: 90 files from the local test corpus
  (78 invoices; the rest are receipts, a quote, claim statements and two
  PDFs whose text layer is unreadable). Labels were drafted by the agent from
  the extracted PDF text on 16 October 2026 and are `reviewed: false` until
  the owner checks them.
- `private/recording.jsonl`: every live System One exchange of the last
  `-live` run. `-live` needs `SITEWISE_JEV_API_KEY`.

```
go run ./cmd/invoice-eval -live   # record a live run (asks every question on every case)
go run ./cmd/invoice-eval -fit    # replay and fit cut-offs on the calibration split
go run ./cmd/invoice-eval         # replay and score; fails on any wrong green value
go run ./cmd/invoice-eval -show   # also list each wrong or blank field (terminal only)
```

The fit policy (Wilson lower bounds at z = 1.645: green 0.95, amber 0.70,
at least 20 answers per question shape and 10 per band) is the agent's
proposal and needs the owner's sign-off. Any change to a question's wording
or options is a new `QuestionVersion` and needs a new live run.
