# Chat that can see and move the screen: implementation plan (plan 0)

Date 18 November 2026. Lane A where it touches chat's rules (what applies at
once, what waits for a click); lane B for the screen wiring. Status: built
18 November 2026 and merged to local main (3994871); what was built,
measured and left open is in `docs/evidence/2026-11-18-chat-agency.md`.
Owner decision 1 was settled the same day: chat may prepare a read.

It comes first in the next wave (`2026-11-17-next-wave-index.md`). Plans 1 to
6 each fill in the contract it defines ("From chat" in each plan). Source: a
review of chat's reach supplied by the owner on 18 November, with each claim
checked against the code the same day.

## In one page (product requirements)

**Outcome.** The PM types "select the basement documents" and the
repository selects them; chat says "Selected 14 documents: 9 by title, 5 by
what they say. 3 scans could be matched by title only." Then "send those to
the structural engineer": Send opens with the documents and the firm filled
in, and the PM presses Send. "Show the cost lines with no budget" shows only
those. With three rows selected by hand, "exclude these" acts on those
three. "Undo that" undoes the last change.

**Why.** Time-to-decision. Chat changes records (about 60 operations) and
answers questions, but it has four gaps, each confirmed in the code:

| Gap | Evidence (18 Nov) |
|---|---|
| It cannot move the screen | The only screen event is `go`, sent for 15 exact phrases (`internal/httpapi/chat.go:88`). `Chat.tsx` is given `onShow` and `onOpenDocument` only (`web/src/Project.tsx:800`); the selection lives beside it (`Project.tsx:263`). |
| It cannot see the screen | The request is `conversation_id` and `text` (`chat.go:130`). "These" and "this document" mean nothing. |
| It cannot find a set by meaning | `documents` matches words in the title or number, kind and discipline (`chat_docs.go:148`). `passages` returns quotations, not a list of documents. |
| It forgets ids between turns | `saidBefore` replays the answer and the cards' lines as text (`chat.go:283`). "Those" has nothing to point at. |

A fifth limit was chosen, not missed: nine routes are classed "chat starts
no read" and chat takes no file. Owner decisions 1 and 2 revisit them.

**Where it lives.** The strip, as it is. No new screen, button or panel. The
PM sees the workbench move and one line in chat saying what it did and why.

**The rule for what chat may do.** One table, used by every plan in the wave.

| The act | From chat |
|---|---|
| Moves the screen (show, open, select, filter) | Done at once. It changes no record, so there is no Undo and no click, even after the turn read a document. |
| A record change that can be undone | Applied at once with Undo (as today). |
| A vouch, or a change with no Undo | Prepared; the PM clicks Apply on the card (as today). |
| Leaves the building: send, issue, share | Chat opens that screen's own dialog, filled in. The PM presses Send, Issue or Create link there, looking at the thing itself. Chat never presses it. |
| Starts a Jev read | Decision 1. Recommended: a prepared card with Start. The turn still never calls Jev; the click queues the background job. |
| Sign-in, billing, team, the model switch, deleting or archiving a project | Never. |

The fourth row is what lets chat run a whole job without weakening design
4.3: `POST /reports/{r}/issue` and the send route keep their `noOperation`
reasons word for word.

**What reads what.** Jev: none. LLM: the chat model, as now; one more tool.
Go: ranks documents, says why each matched, checks every id against the
project, computes named filters, undoes. No embeddings.

**Speed budgets.**

| Path | p50 / p90 ms | Bench row |
|---|---|---|
| A screen command: named filter, select by `about`, open, show | 1,500 / 3,000, one model round | new `chat_screen_command` (the `chat_simple_command` numbers; the bench fails when the median takes more than one round) |
| `documents` with `about`: 2,000 documents, 200,000 passages | 100 / 250 | `chat_tool_read` (existing) |
| "Undo that", answered by Go with no model call | 100 / 250 | `change_undo` (existing) |
| Chat request carrying screen context | unchanged | `chat_first_response`, `chat_simple_command` |

**Done means.**
1. On the Seven Hills fixture "select the basement documents" selects the
   key's set in one model round and says why; a scan with "basement" only in
   its title is in the set and in the count of title-only matches.
2. Three cost lines selected by hand, then "exclude these": exactly those
   three are excluded, with Undo.
3. "Undo that" undoes the change with no model call.
4. After (1), "now show only those" works from the stored ids; the request
   carries no titles.
5. A turn that read a document and then only moves the screen prepares
   nothing and changes no table (snapshot test).
6. An id from another project or org in the screen context or a `view` call
   is dropped or refused; it is never selected and never reaches the model.
7. `TestOnlyChatReachesTheModel` passes; the bench rows pass; the key's
   `screen` group meets the floor the owner sets.

**Out of scope.** Embeddings. Reading scans that have no text. Chat speaking
before it is spoken to. Chat on the guest page. More than one project in a
conversation. Refiling many documents at once with Undo (it stays prepared:
one click for the lot).

**Owner decisions** (recommended default first).
1. **Starting a read from chat.** Allow it as a prepared card with Start:
   "Read the conditions from DA-2025-0733?" The click spends Jev credits and
   cannot be undone, so it is never automatic. This rewords nine
   `noOperation` reasons and one sentence of AGENTS.md. Alternative: keep
   "chat starts no read" and have chat open the screen that starts it.
2. **A file dropped on the strip.** Not in this wave. Later: the file goes
   through the ordinary upload (never through the model) and its id joins the
   screen context, so "this is Acme's roofing quote" can place it.
3. **The decline log.** Counts for every org; the message text is read only
   for your own org. Other orgs' words stay in their rows.

## What we reuse

| Need | Existing thing | File |
|---|---|---|
| Selection the transmittal already sets from outside | `setDocumentSelection(selectDocumentRows(ids))` | `web/src/Project.tsx:817` |
| One selection model on nine tables | `Selection`, `selectAll`, `prune` | `web/src/table/selection.ts` |
| "Showing … Clear" line above a table | `reg-filter` | `web/src/Register.tsx:343` |
| Passage hits counted per document | `ChatSearchPassages` (`count(*) OVER (PARTITION BY p.document_id)`) | `internal/store/chat_docs.go:30` |
| Subject to building system | `SystemTerms` ("Basement" → `substructure.basement-retention`), `systemRoute` | `knowledge/works/system_terms.yaml`, `internal/httpapi/chat_docs.go:513` |
| Builders' synonyms | `Synonyms`, `alternatives` | `chat_docs.go:387` |
| Document list with kind, discipline, scan or not | `projectDocs` | `chat_docs.go:105` |
| A tool that ends the turn in one round | `Output.Final` | `internal/agent/agent.go:95` |
| Stored events per turn | `chat_turns.answer.events` | `chat.go:195` |
| Undo by the actor who made the change | `UndoChange` | `internal/store/change_undo.go:56` |
| What needs attention, as Go rules | `readAttention` (moved to `reports.Attention` by plan 5) | `internal/httpapi/chat_more.go:165` |
| Key, scorer, runner | `data/eval/chat/`, `internal/chateval`, `cmd/chat-eval` | — |

## What is new

| Thing | Why an existing one cannot carry it | Est. lines |
|---|---|---|
| Tool `view` and event `view` | `go` names a screen and nothing else | 120 Go |
| `web/src/chatView.ts`: one registry where a table offers `select(ids)`, `only(ids)`, `open(id)` and a dialog offers `open(fields)` | Wiring each table into `Chat.tsx` by hand would grow with every plan | 150 TSX |
| `screen` on the chat request | The request has no place for it | 60 Go, 60 TSX |
| `about` and `select` on `documents` | Title words cannot answer "about the basement" | 200 Go |
| Named filters per table, computed in Go | A set the model picks row by row costs a round and can be wrong | 80 Go, 40 TSX |
| Ids in `saidBefore`; "undo that" routed by Go | History is text only | 70 Go |
| `chat_turns.declined`; key group `screen`; expectations `selects`, `opens` | Nothing measures what chat could not do | 120 Go, 5 SQL |
| Prepared reads (decision 1) | The nine routes are refused today | 60 Go |

About 710 Go and 250 TSX, tests extra. No table, no screen, no dependency.

## Tasks

**1. Screen context.** `Chat.tsx` sends `screen: {view, open: {kind, id},
selected: {table, ids}}` with each turn (ids capped at 200, with the full
count). Go drops any id not in this project, turns the rest into the short
ids the reads use, and writes one line into the turn's own message, after
the cached prefix: "On screen: Cost Plan. Selected: 3 cost lines (c12, c14,
c19)." Ids and counts only, never titles, so the line is not outside text
and does not make a change wait. Test: foreign ids dropped; the prefix hash
is unchanged; "exclude these" on three selected lines.

**2. The `view` tool.** Input `{do: show|open|select|only|clear, table,
ids, filter, dialog, fields, final}`. Go checks the table, the ids and the
filter name, emits a `view` event and stores it with the turn. `Chat.tsx`
looks the table up in `chatView.ts` and calls its setter; a table not on
screen is shown first. `only` hides the other rows under the existing
"Showing 6 lines with no budget · Clear" line. It never takes focus from an
open cell editor. Day one: documents, cost lines, Program rows, Procurement
rows, and the award dialog (chat opens it on the named firm, where today it
only tells the PM to go there). Test: each verb on each table (Playwright);
a view after a document read prepares nothing.

**3. Finding documents by what they are about.** `documents` takes `about`
and `select`. Go ranks by: the words and their synonyms in the title or
number; passage hits per document; passages the profile labelled with the
system the words name. Each row says why ("title", "12 passages", "labelled
Basement and retention"). The coverage line counts documents with no text,
matched by title only. `select: true` selects the result and ends the turn.
Cap 200, said when reached. Test: the Seven Hills key set; a superseded
revision left out unless asked for; another project's documents never in it.

**4. Named filters.** For a set that is a rule, not a judgement, the model
names it and Go computes the ids: cost lines with no budget, over budget,
excluded; documents not filed, not read, superseded; Program rows with no
duration, critical; packages with no quote. The rules are the ones
`Attention` already holds, so chat, the report and the filter cannot
disagree. Plans 1 to 6 add theirs (awaiting a reply, overdue, to check, open
before a gate). Any other set the model picks by id from a `read`, in two
rounds.

**5. Memory of ids, and undo.** `saidBefore` adds "[Selected 14 documents:
d3 d7 …]" (50 ids, then "and N more"), "[Changed c81: 3 cost lines]" and
"[Opened Send with 3 documents]". The message "undo", "undo that" or "undo
the last change" is answered by Go: the newest change this user made in this
conversation is undone, or "Can't undo: this has changed since." No model
call.

**6. Prepared reads (decision 1).** One operation per read route, each
`Click: true`, with the card saying what will be read and that it cannot be
undone. `ops_routes.go` reasons and the AGENTS.md sentence change with it.

**7. The key and the measure.** `gen_cases.py` gains a `screen` group of at
least 20 cases and the expectations `selects` (the table and its ids) and
`opens` (the dialog and its fields). When an answer carries the fixed words
"I can't do that from chat yet", Go sets `chat_turns.declined`.
`cmd/chat-eval -declined` reports declined turns over all turns by week.
That share, not the key, is the measure of "95% of what a PM asks".

**8. Bench and documents.** `chat_screen_command` in `bench/budgets.json`
and the gate test. AGENTS.md and PRODUCT.md: chat moves the screen, sees
the selection, and opens Send, Issue and Share filled in but never presses
them.

Order: 1, 2, 5 (small, and useful on today's screens), then 3, 4, 7, then 6.

## Edge cases

| Case | What happens |
|---|---|
| The PM changes the selection after pressing Enter | The turn uses what was selected when it was sent |
| "These" with nothing selected | Chat asks which; nothing changes |
| 600 rows selected | 200 ids and the count are sent; an operation on "these" is refused with "narrow the selection", never applied to a part |
| An id deleted since the last turn | Dropped; the line says "2 of the 14 are no longer there" |
| `about` matches nothing | "No document mentions basement", with the words tried and the coverage line |
| A scan with no text | Matched by title only, and counted as such |
| The table is not built yet (messages before plan 1) | "I can't do that from chat yet", logged as declined |
| Chat selects while a cell is being edited | The editor keeps focus; the selection changes under it |
| A view turn after reading a document | Done at once: nothing is written |
| Undo of a change another screen has since altered | "Can't undo: this has changed since." |
| Wrong org or project | 404 on the route; ids dropped before the model sees them |

## Risks

- **The tool list changes the cached prefix** once, at the deploy.
- **A moving screen can surprise.** Every view says what it did in one line,
  and Clear or Escape puts the table back.
- **`about` is words and labels, not understanding.** It says why each
  document is in the set, so a wrong one can be seen and unticked. The key
  measures it; no claim is made beyond the key.
- **Two rounds for an ad hoc set.** Named filters cover the common ones; the
  decline log and the round count show which to add next.
- **Decision 1 widens what a click on a card can start.** Each prepared read
  names the document and the cost in plain words.
