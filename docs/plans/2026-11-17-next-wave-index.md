# Next wave: one agent, four records

17 November 2026; revised 18 November after a review against Aconex and
Procore and a review of what chat can reach. Plans only; nothing is built.
Read this first. Each plan holds the detail. Where this page and a plan
differ, this page wins. Each plan was revised on 18 November to match and
says so at its head; the text before that is at git ref
`refs/backup/next-wave-plans-2026-11-18`.

## What this wave is

After award a desk PM still keeps four things in email and spreadsheets.
This wave moves them into SiteWise, and makes chat able to run all of it.

1. **What was sent and received** (plan 1), and a way to hand things to
   people outside the org (plan 2).
2. **What is owed, by whom, by when**: RFIs, notices and consent conditions
   in one register (plans 4 and 6).
3. **The money after award**: variations, progress claims, payment
   schedules (plan 3).
4. **The monthly telling**: one report from the screens' own figures, with
   what moved (plan 5).

Plan 0 comes first: chat that can see and move the screen. Every other plan
states what chat reads, changes, opens and never does.

## Five rules for the wave

1. **The record writes itself.** A register filled by hand is Aconex with a
   nicer font. Mail threads itself, replies tick off RFIs, a dropped consent
   becomes its conditions. The PM corrects; the PM does not type.
2. **A message is a record; what is owed is a register row.** One table
   says what is open, with whom and when it is due. One Go function
   (`Attention`) says what needs a look, and the left nav's marks, chat and
   the report all read it. No screen grows its own list of due things.
3. **Chat can do what a screen can do.** What can be undone applies at once.
   A vouch waits for a click. What leaves the building (send, issue, share)
   chat opens filled in, and the PM presses the button (plan 0's table).
4. **The model has one door: the strip.** No "draft with AI" buttons. RFI
   wording, a transmittal note and a report comment are drafted in chat,
   figures printed by Go.
5. **Nothing new to look at unless it has rows.** One left-nav slot is
   switched on. No tab, column or banner for a thing the project does not
   have.

## Where Aconex and Procore are strong

Read on 18 November 2026 from their own help pages and announcements, with
review sites for the complaints. Sources at the foot.

| | Their strength | What SiteWise does |
|---|---|---|
| **Aconex** | The shared record nobody can alter: sent mail cannot be deleted, versions are superseded and kept, each organisation holds its own copy, and contracts mandate it. | **Match the part one org can:** a message filed or sent is never edited or deleted, and a transmittal keeps the bytes it sent (plan 1). **Do not claim the rest:** SiteWise is the PM's own record, not the project's neutral one. |
| | A transmittal updates the recipient's register, so everyone knows who holds which revision. | **Match:** each document shows whom it was sent to and when; a superseded revision says who still holds it (plan 1). |
| | Formal review: Document Process, Review Matrix and Test Plans (April 2026). | **Not chased.** Review-comment sheets are deferred (plan 4); review matrices and ITPs belong to large delivery teams. |
| | Weakness: every document's fields typed on upload ("a chore"), fixed mail forms, a dated interface, every party needs an account and training. No assistant is marketed. | **Improve:** filing is already a second and needs no typing; a guest needs no account (plan 2). |
| **Procore** | RFIs done well: the ball in court, required responders, an official response, a reply by email logged on the RFI. | **Match in one column:** "With" says who holds it now. A reply by email threads and ticks the RFI (plans 1 and 4). |
| | Breadth for a head contractor: submittals, change events to change orders, commitments, invoicing, daily logs, punch lists, a phone app. | **Not chased:** site diaries, defects, photos and a phone app. **Match for a client-side PM:** variations and claims (plan 3). |
| | Agents: Deep Search with citations, an RFI agent, submittal and contract review, and Agent Builder with about 150 actions that create or update records after a confirm. | **Improve:** chat applies at once with Undo instead of a confirm per record; every figure is printed by Go; it moves the screen and carries a job across areas (plan 0). Procore's actions are per tool. |
| | Weakness: complexity and a steep learning curve are its most common complaints; priced on construction volume; reviewers call it heavy for small projects and a poor fit for owners. Assist answers in US terms. | **Improve:** one building, one person, Australian words. |
| **Neither** | No Security of Payment dates worked out per state, and no consent-conditions register, were found in either product. Payapps (not Procore) is where Australian builders get claim reminders. | Plan 3's dates err early, never late. Plan 6 reads the notice of determination into rows. These two are the wave's reasons to switch. |

## The plans at a glance

Sizes are the plans' own estimates adjusted for the cuts and for chat's
operations; none is re-measured. The wave is about 550 lines of Go larger
than the 17 November total: plan 0 adds 710 and the cuts take back less.

| # | Plan | What the PM gets | Where it lives | Jev | New tables · routes · screens | Go · TSX |
|---|---|---|---|---|---|---|
| 0 | [Chat sees and moves the screen](2026-11-18-chat-agency.md) | "Select the basement documents." "Send those to the engineer." "Exclude these." "Undo that." | The strip, as it is | None | 0 · 0 · 0 | 710 · 250 |
| 1 | [Correspondence and transmittals](2026-11-17-correspondence-transmittals.md) | Every email in and out in one frozen, threaded list. Send drawings on a numbered transmittal. See who holds which revision. A message that looks like an RFI, a claim or a notice offers its one next step. | Repository's Messages list; "Send…" on right-click | One pick joined to the filing call | 0 · 3 · 0 | 910 · 390 |
| 2 | [Guest access](2026-11-17-guest-access.md) | A link to documents, a report or an RFT. A firm drops its quote on its own cell. Revoke in one click. | "Share…"; one small guest page | None new | 1 · 8 · 1 | 610 · 310 |
| 3 | [Variations and progress claims](2026-11-17-variations-progress-claims.md) | A variation register per contract; a dropped claim read in a second; assess, issue the payment schedule, and a warning when its date is near. | Cost Plan's Variations tab | One fan-out per claim | 3 · 5 · 0 | 1,700 · 500 |
| 4 | [The register and RFIs](2026-11-17-rfis-review-comments.md) | One table of what is owed: RFIs, notices, conditions. Replies matched to RFIs. "What is open on the façade." | The idle left-nav slot | Reply picks, one background call per document | 1 · 1 · 0 | 640 · 320 |
| 5 | [Project report](2026-11-17-monthly-report.md) | A draft in under a second from the screens' figures, with what moved since the last one. Comment, Issue, A4 PDF. | The report reader, opened from the register | None | 0 · 0 · 0 | 650 · 50 |
| 6 | [Consent conditions](2026-11-17-consent-conditions.md) | Drop the notice of determination; every condition is a row with its gate, who acts, the building system and whether evidence is needed. Program gets the gates. | The register, filtered to Conditions | One background call per condition | 0 · 1 · 0 | 1,010 · 200 |
| | **Total** | | **One nav slot switched on; nothing else added to the nav** | | **5 · 18 · 1** | **≈ 6,250 · 2,000** |

**Jev before the LLM.** Jev reads every document in this wave. The LLM
drafts words, in chat only. No screen read or write waits on any model.
Every amount and date is computed in Go.

**Speed.** Screen reads and writes stay at 100 ms p50 / 250 ms p90. Filing
stays at 1 s / 2 s. A screen command in chat is one model round, 1.5 s /
3 s. Background work has its own rows.

## The chat contract

Each plan carries a "From chat" section with these four lists, an operation
for every new write route (none parked as pending), and at least six cases
in the command key: two reads, two changes, one that moves the screen, one
that must wait for a click.

| Plan | Reads | Changes at once, with Undo | Waits for a click | Opened filled in; the PM presses the button |
|---|---|---|---|---|
| 1 | The register of messages, a message's body, who was sent a document | Link a package, dismiss a pick | — | Send and Reply, with documents, firms and a drafted note |
| 2 | Who holds a link to what, and when it was opened | — | Revoke a link | Share |
| 3 | Contract sum to date, variations, claims, next due date | Add or edit a variation, type an assessment | Approve or reject a variation, Accept claimed, a contract's periods | Issue the payment schedule (the claim's panel), then Send |
| 4 | The register by kind, system, who holds it, overdue; the also-open line | Add, edit, close, reopen, set who holds it, the response time | Accept a possible answer | Send an RFI |
| 5 | The draft's blocks and what moved | Create, Update, write or clear a comment | — | Issue (the reader), then Send |
| 6 | Conditions by gate, who and status; possible evidence | Edit, link evidence, add to an RFP | Satisfied, accept possible evidence, read a notice | — |

One typed line can then cross all of it: "Raise an RFI to the structural
engineer about the basement wall waterproofing, attach the drawings that
show it, and open it to send." Chat adds the RFI as a draft, finds and
selects the drawings, and opens Send. Neither competitor's agent does that
across its tools.

## Cut or deferred by the review

| What | Why | Plan |
|---|---|---|
| Contract time bars (a knowledge file of clause periods, a loader, a checker) | The periods sit in copyrighted forms that special conditions amend, and nothing computes until you review about 16 rows. Day one the PM types the date or says it in chat. Statutory dates (plan 3) stay: they can be checked against public Acts. | 1 |
| Notice, Due and Status as columns on Messages, and "Messages · 2 due" | Rule 2. A confirmed notice becomes a register row. | 1 |
| Guest grants for RFI answers and comments | Built ahead of any design for them. A consultant answers by replying to the email. | 2 |
| Review-comment sheets, PDF markups, a Jev call per comment | The least certain item in the wave, with its own parser and answer key. Revisit after RFIs and conditions have run on two projects. | 4 |
| Eight kind tabs (RFIs, Comments, Actions, Risks, Issues, Decisions, Approvals, All) | Rule 5. One table, a Kind filter that lists only kinds with rows. | 4 |
| The LLM summary button on the report | Rule 4. Chat drafts the comment. | 5 |
| The report in Procurement's three-dot menu | A project report is not a procurement action. It opens from the register and from chat. | 5 |
| "Chat never sends", "never shares", "does not change settings" | Rule 3. Chat opens Send and Share filled in; settings are operations. | all |

## Build order

- **First: plan 0, Tasks 1, 2 and 5** (screen context, `view`, ids and
  undo). Small, and useful on today's screens. Then its Tasks 3, 4 and 7.
- **Then `Attention` as one Go function** (plan 5, Task 5, lifted out). Plans
  1, 3, 4 and 6 each add rules to it.
- **Track A, needs nothing from outside: 4 → 6 → 5.**
  - Plan 4 is now the register and RFIs only.
  - Plan 6 is the feature neither competitor has, and its test documents
    are already on hand.
  - Plan 5 reads everything above.
- **Track B, starts with you: email intake → 1 → 2.**
  - Email intake waits on its Task 0, a Mailgun check that needs your login.
  - Plan 2 is lane C and gets a security review before it ships.
- **Then plan 3, in two slices:** 3a variations, 3b claims and payment
  schedules. 3b waits on decisions 6 to 8 below.

## Agreements across plans

1. **One business-day helper.** `program.BusinessDayAfter` in
   `internal/program/calendar.go`, written by whichever of plans 3 and 4
   lands first. RFIs use the project's calendar. Security of Payment dates
   pass public holidays only: a builder's rostered day off must never push a
   statutory date later.
2. **One post-filing job for the register.** `register` is queued beside
   `queueVisual`, only when the project has an open RFI or condition. It
   carries plan 4's reply picks and plan 6's evidence picks in one Jev call.
3. **One register table.** RFIs, notices and conditions are kinds in
   `project_delivery_items`. Variations are not: they hold money that feeds
   the Cost Plan and Undo, so they have their own table.
4. **One read route into the register.** `POST /projects/{id}/delivery/imports`
   is built by plan 6 (comment sheets, its other caller, are deferred).
5. **Shared migration checks.** Plans 4 and 6 widen the delivery kind and
   status checks; plans 3 and 5 widen the report kind check. Whichever lands
   second lists both.
6. **What a guest sees of a report.** Every issued version, newest first,
   never a draft.
7. **One way out.** All mail leaves through plan 1's send path, and only on
   the PM's press of Send.
8. **Frozen means frozen.** A sent or filed message, an issued report and
   an issued payment schedule are never edited or deleted.
9. **The message offers the next step.** Jev's pick on an email body routes
   it: possible RFI or notice → Add to register (plan 4); possible payment
   claim → Read as progress claim (plan 3); possible variation claim → Add
   variation (plan 3). One chip, one button.

## Decide these first

The plans held 61 decisions. The plans now take their own defaults for the
rest; these nine change what gets built.

1. **Chat and reads.** Settled 18 November: yes. Chat may prepare a Jev
   read as a card; the PM's click starts it (plan 0, decision 1).
2. **The register's home and name.** Settled 18 November (the owner left
   the choice to the plans): the idle nav slot (labelled "Report" today)
   becomes "Register", one table with no kind tabs.
3. **The cuts.** Settled 18 November (same): review comments and contract
   time bars are deferred as above.
4. **Mailgun.** Run email intake's Task 0, and one DNS step so SiteWise can
   send from the project's address.
5. **Outside senders.** File, rather than hold, mail that replies to
   something SiteWise sent or that comes from a firm already on the project?
   Recommended: yes.
6. **NSW Security of Payment Act.** Someone with a browser reads ss 4, 11,
   13 and 14 on legislation.nsw.gov.au. Queensland's sections are verified.
   Most projects are in NSW, so this gates plan 3b.
7. **Public holidays.** Use the NSW and QLD government lists for 2026–2028,
   shared with Program?
8. **Answer keys (real documents from you).** About 50 emails, about 60 RFI
   replies, 20 or more head-contract claims, and a review of the consent key
   built from the Petersham, Camp Y and Bankstown notices. Until a key is
   reviewed, its Jev picks stay withheld.
9. **Condition picks before their key is reviewed.** Amber "to check" from
   day one (recommended, as the profile does), or blank?

## Facts the planning turned up

- No screen opens the project management plan today
  (`web/src/Procurement.tsx:317` filters it out), although PRODUCT.md says
  it stays reachable.
- The Cost Plan's "Remaining" is computed two ways:
  `costs.Derived.Remaining` is forecast less claimed; the grid shows
  approved contract less invoiced. The report uses the grid's figure, and
  plan 3 changes both to include variations.
- Four existing queries find a package's report without checking its kind.
  Plan 3 fixes them before any new report kind lands.
- Consultant invoices can be payment claims under the Acts. Plan 3 leaves
  them as invoices with no dates. For a client-side PM before construction
  they are most of the claims received; it is the first follow-up.

## Sources (read 18 November 2026)

- Aconex record and audit trail: https://help.aconex.com/documents/how-do-version-control-and-the-aconex-audit-trail-work/
- Aconex, April 2026 capabilities: https://www.prnewswire.com/news-releases/new-oracle-aconex-capabilities-improve-project-transparency-and-control-302739966.html
- Aconex reviews: https://www.capterra.com/p/220249/Oracle-Aconex/reviews/ and https://www.g2.com/products/oracle-aconex/reviews
- Procore agents: https://www.procore.com/ai
- Procore Agent Builder actions: https://support.procore.com/products/online/user-guide/project-level/agent-builder/faq/what-actions-are-available-in-agent-builder
- Procore Assist, what it can read: https://support.procore.com/products/online/user-guide/project-level/assist/faq/what-data-can-assist-access
- Procore RFIs: https://support.procore.com/products/online/user-guide/project-level/rfi/tutorials/respond-to-an-rfi
- Procore and Datagrid, May 2026: https://www.enr.com/articles/63042-procore-releases-new-ai-agients-after-datagrid-integration
- Procore reviews: https://www.g2.com/products/procore/reviews
- Progress-claim software in Australia: https://planyard.com/blog/best-progress-claim-software-australia
