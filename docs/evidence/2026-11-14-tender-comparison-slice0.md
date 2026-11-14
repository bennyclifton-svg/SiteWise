# Tender comparison slice 0: the tender pass

Date: 14 November 2026. Branch `tender/comparison`. No firm names or real
prices appear here.

## What was built

- `internal/config`: `TenderKey` (`SITEWISE_ANTHROPIC_HAIKU_KEY`, blank is
  off) and `TenderModel` (`SITEWISE_TENDER_MODEL`, default
  `claude-haiku-5-5`, must be a pinned Claude id).
- `internal/llm`: the one model client. `Asker` interface, four calls in
  flight, a breaker that opens for a minute after five failures, a two
  minute deadline, and a call log that never carries the request or answer.
- `internal/tender`: the types (contract C2), the prompt `t1` and its
  schema, page selection by rule, the request, the parser (keeps what
  arrived when the answer is cut off) and `Pass`.
- `internal/identity/pages.go` and `pages_render.py`: whole pages as JPEG,
  long edge 1568, through the OCR helper's Python.
- `cmd/tender-eval -one <file> -schedule <id>`: one file end to end, printed
  to the terminal only.

## What calls what

`tender-eval` reads the file, `identity.Extract` gives text, pages with no
text go to the existing OCR, `tender.PagesOf` builds pages,
`tender.SelectPages` picks them, `OCR.Pages` renders the image pages,
`tender.BuildRequest` builds one request, `llm.Client.Ask` streams it to
the provider, `tender.ParseAnswer` reads the JSON.

## What could fail

No key (the client is off); no Python renderer (`ErrNoRenderer`, no call is
made); the provider is down, slow or rate limited (typed error, the breaker
opens after five); the model declines (`ErrRefused`); the answer is cut
off (complete lines are kept, `Cut` is true).

## Timings (live, one run each)

| Tender | Pages sent | Lines | Seconds | Tokens in | Tokens out | Stop |
|---|---|---|---|---|---|---|
| Consultant proposal, 12 pages | 3 images, 7 text | 10 | 8.4 | 15,092 | 1,633 | end_turn |
| Builder quote, 4 pages | 3 images, 1 text | 11 | 7.0 | 13,098 | 1,373 | end_turn |
| Large camp schedule, 12 pages | 11 images, 1 text | 50 | 20.2 | 41,215 | 5,530 | end_turn |

The first two are inside the 15 s p50. The small proposal was checked by
eye: every printed fee is listed once with the right amount, the rates are
rates, and the printed total equals the lines. The large schedule came back
with far fewer lines than it prints (it priced at pack level); slice 1
measures that against the key.

## New dependency

`github.com/anthropics/anthropic-sdk-go/v2` v2.9.0, the provider's official
SDK: streaming, retries and typed errors. It brings indirect modules
(`go.mod`).

## Baseline failures on `main` (c9eb78e) before any change

- `web`: `npm test`, 3 cases in the Program engine fixture test.
- `internal/httpapi`: `TestProgramRoutes`, and
  `TestQuotedComparisonFiveFirmsBudget` (a timing budget; it ran while
  other suites were running).
- Everything else named in task 0 passed.

## Where the plan was wrong

- The plan's `client.go` listing left out the `sync` import ordering only;
  it compiled once formatted.
- `rowsFor` uses `Catalog.RFPSkeletonFor`, not `RFPSkeleton`: a works RFT
  resolves against the works skeleton.
- `invoices.ParseCents` returns three values (cents, ok, error).
- The worktree got a private Postgres on port 5436 so that its migrations
  do not touch the shared test database other sessions use.
