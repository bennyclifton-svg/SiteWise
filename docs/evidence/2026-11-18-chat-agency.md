# Chat that sees and moves the screen: what was measured

Date: 18 November 2026. Branch `chat/agency-key`, on `chat/agency` (not
merged, not deployed). Plan: `docs/plans/2026-11-18-chat-agency.md`.
Everything here was run on the development machine against a private
PostgreSQL 17 schema with a stand-in model; the live server was not
contacted, and no live model or Jev call was made. The command key
(`data/eval/chat/`) is a draft the owner has not reviewed, and its new
`screen` group is draft too.

## 1. What was built, by task

| Task | What | Where |
|---|---|---|
| 1. Screen context | The strip sends `screen` (view, open document, selection) with each turn. Go drops any id not in this org and project and writes one line after the cached prefix, with ids and counts only. A selection over 200 rows is sent as a count; an operation on "these" is then refused, never applied to a part. | `internal/httpapi/chat_screen.go`, `web/src/chatView.ts` |
| 2. The `view` tool | show, select, only, clear and open on four tables (documents, cost lines, Program rows, packages), and the award dialog opened on one firm's column. Go checks the table, every id and the dialog. A move is a `view` event stored with the turn; it writes nothing, so it has no Undo and no card, even after a document read. | `internal/httpapi/chat_view.go`, `internal/agent/agent.go` |
| 3. Documents by subject | `documents` takes `about` and `select`. Go ranks by title words and their synonyms, passage hits, and passages the profile labelled with the system the words name; each row says why. A scan is matched by its title only and counted. With `select` the set is selected and the turn ends in one round. No embeddings. | `internal/httpapi/chat_docs.go` |
| 4. Named filters | cost lines `no_budget`, `over_budget`, `excluded`; documents `not_filed`, `not_read`, `superseded`; Program `no_duration`, `critical`; packages `no_quote`. Each calls the rule the screen or the attention read already uses. | `internal/httpapi/chat_filters.go` |
| 5. Ids in the history, and undo | Earlier turns replay as "[Selected 14 documents: …]", "[Changed …: …]". "Undo", "undo that" and "undo the last change" are answered by Go with no model call. | `internal/httpapi/chat_undo.go` |
| 6. Prepared reads | Nine Jev reads chat prepares as a card; the PM's click runs the screen's own route, which only queues the read. | `internal/httpapi/ops_reads.go` |
| 7. The key and the measure | A `screen` group of 27 draft cases; the scorer's `selects`, `opens` and `no_model`; cases that carry earlier messages and the screen sent with the turn. `chat_turns.declined` (migration 065) and `chat-eval -declined`. | `data/eval/chat/`, `internal/chateval`, `cmd/chat-eval`, `internal/store/chat_declined.go` |
| 8. Bench and documents | `chat_screen_command` in `bench/budgets.json`, held by `TestChatScreenCommandBudget` and sampled by the intake bench; AGENTS.md, PRODUCT.md and this note. | `bench/budgets.json`, `internal/httpapi/chat_screen_budget_test.go`, `cmd/intake-bench/chat.go` |

Tasks 1 to 6 were built before this note on the branches merged into
`chat/agency`; their commits say what each measured. Tasks 7 and 8 are
this branch.

### Task 7 in more detail

**The key.** Rows are named the way the fixtures name them, never by id:
a cost line by its label, a document by its title, a Program row by its
label, a package by its discipline id, the award dialog's column by the
firm's name. The runner (`cmd/chat-eval/run.go`) turns names into the
run's ids for the screen it sends, and turns the turn's `view` and `go`
events back into names for the scorer. A case with earlier messages is
judged against the fixture as it was before the first message. The
apartments fixture gained four basement documents: one with basement in
its title only, one in its text only, one labelled with the basement
retention system only, and a scan.

| Case | What it holds |
|---|---|
| s1 to s5 | A named filter on each table: no budget, over budget (none, so the table is shown and the sentence says so), superseded, no duration (none), no quote |
| s6, s7 | Documents selected by subject: the basement set (title, text, label, scan) and the trees set |
| s8 | "Exclude these" with three cost lines selected: exactly those three are excluded |
| s9 | "Exclude these" with nothing selected: chat asks, nothing changes |
| s10 | "Show only these" with two documents selected |
| s11, s12 | "Now show only those" after a selection, from the history's ids, with no screen sent |
| s13, s14, s15 | "Undo that" and "undo the last change" after a change, and with nothing to undo: no model call |
| s16, s17 | The award dialog opened on a named firm's column; nothing is awarded |
| s18, s19 | A screen move after a document read: nothing is prepared |
| s20, s21 | A prepared read (profile update, a tender read again): a card that waits for the click |
| s22, s23 | A table not built yet (messages, RFIs): the fixed words, nothing changes |
| s24 | An exact phrase ("Open the program"): Go shows the screen, no model call |
| s25, s26 | A package selected by name; "show all the cost lines again" clears a filter |
| s27 | "This document" with a document open on screen: quotes that document |

**Scored offline.** `TestScreenCasesScoredOffline` (`cmd/chat-eval`) runs
23 of the 27 through the real server on the real fixtures, with a
stand-in model that makes the call a right answer makes, and scores them
with the real scorer. All 23 pass. To check that the scorer bites, four
expected sets were changed by hand in a copy of the key (one row taken
from the basement set, one row added to the no-budget set, another firm
named for the award dialog, the undone line required to be gone); all
four cases then failed with the reason named, and the key was
regenerated. This scores Go's half of a case: the set a filter or a
subject finds, the ids the screen or the history carries, the undo, the
dialog's fields, and that a move after a read prepares nothing.

**Scored only with a live model.** `s9` (ask when nothing is selected),
`s22` and `s23` (say the fixed words) and `s27` (quote the open document)
are the model's judgement alone. In the other 23, whether the model makes
the call the stand-in makes is also the live run's to score. The final
turns of `s13`, `s14`, `s15` and `s24` never reach the model at all,
though `s13` and `s14` need a model for their earlier message in a live
run.

**The decline measure.** The fixed words are one Go constant
(`declineWords`, `internal/httpapi/chat.go`). The operation list, the
view tool's refusal of an unknown table and the detector use it, and
`knowledge/chat/system.md` now says to use exactly those words, also for
a table the view tool does not list. `TestDeclineWordsAreThePrompts`
fails if the prompt and the constant drift. A finished turn whose answer
carries the words, with either apostrophe, is stored with
`chat_turns.declined`. `chat-eval -declined -weeks 12 -org <id>` reads
`SITEWISE_DATABASE_URL` and prints finished turns and declined turns per
week (Monday, Sydney) for every org, and lists declined messages for the
one org named by `-org` (owner decision 3). The count query reads counts
only; the text query is scoped to that org.
`TestDeclinedReportListsOneOrgsWords` holds that another org's words are
never printed, with or without `-org`. A failed turn (chat off, the model
down) is not counted.

## 2. Speed

`TestChatScreenCommandBudget`, 24 samples (the four commands six times
each: a named filter, a select by `about`, an open, a show), stand-in
model answering at once, a 60-line cost plan and 49 documents. So the
figure is Go's share of the turn: the outline, the screen line, the tool,
the view event and storing the turn.

| Budget row | Budget p50 / p90 | Measured p50 / p90 |
|---|---|---|
| `chat_screen_command`, Go share only | 1,500 / 3,000 ms | 22 / 46 ms (a second run: 20 / 48 ms) |

Every sample took one model round. The test fails when the median takes
more than one: with `final` turned off in the stand-in's calls it failed
with "the median screen command took 2 model rounds". The intake bench
(`cmd/intake-bench`) samples the same four commands through the real
server and fails on the same rule; `TestBenchEndToEnd` in that package
passed, but the bench itself was not run over the private corpus. The live
model's time for a screen command was not measured.

## 3. What was not measured

- No live model run. The key's `screen` group is draft and unscored
  against a live model, so no accuracy figure is claimed for it.
- No owner review of the key, the fixtures or the decline report.
- The full `tools/check.ps1` gate was not run (it uses the shared default
  schema, needs the private corpora and runs the Playwright suite). What
  was run: `go build ./...`, `go vet ./...`, gofmt on the changed files,
  `go test -p 1` over `internal/httpapi`, `internal/agent`,
  `internal/store`, `internal/chateval`, `internal/db`, `internal/latency`
  and `cmd/...` in a private schema (one failure,
  `TestRestoreFactsDetectCostContentChanges`, which fails in any schema but
  `public` because `restore_records.go` names `public`), the web unit tests
  (the three known Program engine failures only),
  `python tools/check_knowledge.py` (0 errors, 1 warning) and the key
  generator.

## 4. Known limits

- "Undo that" does not see a change applied by clicking a prepared card:
  that change is made by the click, not stored as a change card of the
  conversation. It looks back over the conversation's last 12 turns.
- A scan has no text, so it is matched by its title only.
- "Over budget" in the filter means claimed to date over budget, per line
  (the invoice ledger's flag the Cost Plan tags). The attention read uses
  contract over budget, per section. Which one the owner means is an open
  question.
- `procurement.read_tender_again` sends the quote to the model's tender
  pass, not to Jev.
- The decline measure counts only answers that use the fixed words. A
  decline in other words is not counted, which is why the prompt and the
  view tool's refusal both give the words.
