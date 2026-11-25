# Correspondence between parties: items, responses and outside organisations

Date: 23 November 2026. Status: direction from an owner interview on 23
November, revised the same evening after the owner read the first draft
(section 12 records each answer; decisions 31 to 35 came from that review).
**Update 24 November 2026:** the peer review ran (section 18 records its
answers), and slices 1 and 2 are built for one organisation: EOT claims,
notices of delay and letters on the register, the result on an EOT claim,
the optional terms and the agreed completion date. Evidence:
`docs/evidence/2026-11-24-correspondence-slice-1.md`. Slices 3 to 5 are
not built. This note was first written outside the repo, from a read-only clone of `main` at
`9472e1a`. A peer review against the tree comes before any plan
(section 16).

Read first, in full. This note leans on all three and reverses parts of
the third:

- `AGENTS.md`: the non-negotiables.
- `PRODUCT.md`: the Register as it stands.
- `docs/plans/2026-11-17-next-wave-index.md`: the five rules, the
  agreements across plans, and what was cut.

Read only the named part. These plans are built, so the tree is the truth
and each plan is needed for one thing:

- `docs/plans/2026-11-17-guest-access.md`, "In one page", the link model:
  why a person from outside must never be a member of the host's
  organisation. The largest change here rests on it.
- `docs/plans/2026-11-17-variations-progress-claims.md`, "In one page",
  owner decision 3: the behaviour that decision 17 below reverses.
- `docs/design/2026-11-07-billing.md`, "Plans" and "Shape": the Free
  plan's caps, which outside organisations will meet.

Open only if a point is in doubt: the "In one page" sections of
`docs/plans/2026-11-17-rfis-review-comments.md` and
`docs/plans/2026-11-17-correspondence-transmittals.md`. They say why the
register and the message tables are shaped as they are; the code says
what they are.

## 1. Outcome

A builder opens SiteWise and lodges an extension of time claim. The form is
already filled in with who it goes to, who must answer and a subject, so
they add the cause, the dates and the days claimed, and press Send.
SiteWise numbers it EOT-4, stamps the time it was received and tells
everyone it was addressed to. One of them has never used SiteWise: the
email they get opens the claim itself, with no sign-up form in the way.

The person who assesses claims on the project sees it in the one list of
what is owed, with the date a response is due and how that date was worked
out. They reply "2 days granted" with their reasons and a document. The
result is kept as a figure as well as in words, the agreed completion date
moves by two days, and the builder sees the response at once. Nobody
retyped anything, nobody was asked for a contract, and both sides hold the
same record.

The same path carries RFIs, notices of delay, variation claims, progress
claims and ordinary letters, in either direction.

Why it earns its place:

- **Time to decision.** One list says what is owed, by whom and by when,
  for every kind of item. A response is one act, not a letter plus three
  register edits.
- **Answer trust.** The record is made by the party who raises or answers
  it, at the time they do it, and it is frozen. Dates and totals are worked
  out by code from that record.
- **Filing speed** is unchanged and must stay so: documents attached to an
  item go through the same filing as any other document.

Lanes: A for items, results, the completion date and anything that feeds
the Cost Plan. C for accounts, access across organisations and mail. B for
lists, labels and the party's page.

## 2. The owner's brief

Every later choice is held to these four tests.

1. **Fit for purpose, to best practice.** A well-rounded product a
   principal, a PM, an architect and a builder would each choose to run a
   job on. Not a copy of Procore, and not a register only a PM uses.
2. **Use what is built.** The register table, the message tables, the send
   path, filing, operations and Undo, accounts and guest grants already
   exist. New tables and new screens need a stated reason.
3. **Jev first, the model last.** Code computes. Jev picks where something
   unstructured must be read. The LLM drafts words in chat when asked, and
   does nothing else here.
4. **Tasteful and delightful.** Section 10. Above all, no friction: nobody
   is made to set something up before they can ask a question.

## 3. What exists today

Checked by reading `main` at `9472e1a`. The app was not run.

- **One register table.** `project_delivery_items` holds ten kinds, with
  what is particular to a kind in `details`. Only RFIs are numbered
  (RFI-007, never reused). A row can sit on a building system.
- **Only an RFI takes an answer.** An RFI has one answer text and a list of
  answer documents; accepting a possible answer appends the document
  (`internal/store/delivery_edits.go`). The matcher looks only at Open RFIs
  (`internal/store/delivery_match.go`). A notice is Open, Closed or
  Withdrawn, with a due date the PM types. Only an RFI's due date is worked
  out (`internal/delivery/due.go`).
- **A project setting already exists.** `delivery_settings` holds the
  project's RFI response time (five working days by default), set once and
  undoable.
- **"With"** says who holds a row now, worked out at read.
- **Messages.** `mail_messages` holds mail in and out, threaded, with To
  and Cc, linked to a firm and a package. `mail_parts` ties each attachment
  to a filed document and keeps its number, revision and title as sent. A
  filed or sent message is frozen by a trigger. Mail leaves as a numbered
  transmittal through one send route, on the PM's press. Messages has
  Reply.
- **Jev's notice pick** at filing has eight options
  (`internal/correspondence/notice.go`). It stays hidden until the owner
  approves a cut-off; `data/mail/thresholds.json` does not exist yet.
- **Variations and progress claims** have their own tables (index,
  agreement 3). An awarded package (`awards`) records its own payment
  periods, each in business or calendar days. Nothing records a completion
  date, an assessor or an approval limit.
- **The Program** computes a forecast completion. No agreed date is held
  anywhere.
- **Silence on a progress claim.** After a missed payment schedule date the
  claim counts at the claimed amounts, in red (variations plan, decision 3).
- **Accounts.** A user belongs to one organisation. Memberships carry a
  role. Sign-up, email confirmation, password reset and team invites all
  work by an emailed link. A Free plan exists.
- **Guests.** `guest_grants` gives one outside person one shared thing by
  link, through `/guest` routes that read nothing else. The guest plan
  rejected guest accounts inside the host's organisation, because a member
  session opens every project in that organisation. Guest grants for RFI
  answers were cut on 18 November.
- **Writes** go through the operations call with one Undo, and
  `project_changes` records who acted and by which path.
- **`Attention`** is the one function that says what needs a look.
- **Row links.** `delivery_dependencies` holds Program-style
  finish-to-start links only.

## 4. Where Procore is ahead

Read from four of the owner's screenshots of a live project in Procore (an
RFI, its Create and Export menus, and the Correspondence list). These are
inferences from screens, not from Procore's documentation.

| Procore shows | SiteWise today |
|---|---|
| Fourteen or more correspondence types, grouped by party, each numbered in its own run (EOT-4, NOD-7) | One `notice` kind, unnumbered; eight mail picks |
| Any number of responses on any item, some marked official | One answer on an RFI; none on a notice |
| Every party signs in, raises items and responds | One organisation; others reply by email or open a link |
| Assignees, a distribution list and "action required by" | To, Cc and With |
| Cost impact and program impact on each RFI | Neither |
| "Create related item" from an RFI | No general link between rows |
| Saved views, group by, column settings, last activity | Kind and System filters; three named filters |
| One item exported as PDF or zip | No item export |

On that project, 13 of the 17 rows visible in the list were notices of
delay or extension of time claims. SiteWise treats both as one mail pick
and computes no due date for either. That is where the register is thinnest
against real traffic.

SiteWise is ahead in other places and should stay so: rows sit on the
building, replies are matched to the rows they answer, every value shows
how it was decided, and statutory dates are worked out by code.

## 5. The model in one page

Six words carry the design.

| Word | Meaning |
|---|---|
| **Item** | One thing raised by one party that is owed a response: an RFI, a notice of delay, an EOT claim, a variation claim, a progress claim, a letter |
| **Type** | What kind of item it is. A short list held as data in `knowledge/`, not a new table kind per type |
| **Terms** | Optional. How this project works: who assesses claims, the agreed completion date, response periods, the approval limit. Set once per project, with defaults. Never needed to raise an item |
| **Responder** | The one party named on an item whose reply counts |
| **Message** | Anything written on an item, addressed like email with To and Cc. Its attachments are filed documents |
| **Result** | What a formal response decides, kept as data: an outcome, and days or dollars where they apply |

Rules that follow:

- **No contract is required, and there is no register of contracts.** An
  architect can ask a consultant a question with nothing signed between
  them. A small job run on a handshake behaves as though there is a
  contract, and SiteWise lets it. Where a package has been awarded in
  Procurement, an item may name it, filled in when it can be known and
  never asked for.
- An item is a row. A message is a record. What is owed is still read from
  one place (index, rule 2).
- A response is a message. A **formal** response is a message from the
  item's responder that carries a result. Everything else is
  correspondence.
- Several responses stand on one item. For the result, the latest stands
  and earlier ones show as superseded.
- Only results change anything else: an EOT result moves the agreed
  completion date, a variation result moves the Cost Plan, a payment
  schedule sets what is certified.
- Variation claims and progress claims keep their own tables (index,
  agreement 3). An item of those types is the same idea over those tables,
  not a second copy.

## 6. Parties and access

- **A project has one host**, the organisation that created it: most often
  a principal or their PM, sometimes a builder. There is one project per
  building.
- **Every person from outside has their own account, and behind it their
  own organisation.** Nobody from outside is ever a member of the host's
  organisation.
- **Four levels of sight.**

| Level | Who | Sees |
|---|---|---|
| Host | The creating organisation | Everything, including every message on the project |
| Partner, whole project | An organisation on the host's side, such as a principal and their PM | Everything the host sees |
| Partner, register | An organisation that runs the job day to day but sits across the table, such as a D&C contractor | Items and messages across the project, and its design documents. Never cost, tenders, invoices, contract or commercial documents, or messages kept within the host's side |
| Party | Everyone else | Only what was addressed or copied to them |

- **A party's record is theirs to keep.** Read-only access and export
  remain for good, whatever happens to the relationship.
- **The containment rule.** Access across organisations is granted to an
  organisation for one project, and is honoured only by a short list of
  routes written for it, as `/guest` routes are today. No existing member
  route learns to cross organisations.

**Handing a project to a D&C contractor or a builder.** The owner asked
whether the host should be able to pass hosting on, or whether naming the
contractor a partner is enough. Evaluated: a partner grant, not a transfer.

1. Every row of a project belongs to the organisation that created it.
   Moving the host means moving the ownership of every row, the riskiest
   write the system could make. A grant is one row.
2. The principal's cost plan and tender comparison must never pass to the
   contractor. A grant has a level. A transfer has none.
3. A grant can be changed if the contractor is replaced. A transfer is hard
   to take back.
4. What a D&C contractor needs after novation is to run the register: see
   the items, address the consultants, answer and be answered. "Partner,
   register" gives exactly that.

One limit to say plainly. With one project and one host, the host still
sees every message, including the contractor's exchanges with its own
consultants. A contractor who needs those kept private keeps them out of
this project until a linked project per side exists (later). A true
change of owner, such as a PM firm leaving a principal's project, is a
separate question and is also later.

**Joining.** Anyone raising an item can type an email address that
SiteWise has never seen. That person moves through three stages, with one
account throughout.

| Stage | What they can do | What it cost them |
|---|---|---|
| Addressed | Read the email and reply to it. The reply lands on the item | Nothing. No account |
| Signed in | Open the item from the email's link, see everything addressed to them on the project, reply and raise items of their own | One click. The account and its organisation are made behind that click; a password is asked for only when they first come back without a link |
| Host | Create a project of their own | Naming their organisation. Paying when they outgrow the Free plan |

Some people will only ever answer one RFI and never go past the second
stage. Others will become hosts. Neither should notice a step.

## 7. Items

- **Raising is one screen and one Send.** The raiser picks a type and
  everything that can be known is already filled in: recipients and
  responder from their last item of that type on this project, or from the
  project's terms; a subject in the type's pattern; the due date, shown
  before they send. Each type carries its own few fields with a line of
  plain guidance in each, and standard wording they can keep or type over.
  "Another like NOD-6" opens a copy with only the dates blank. All of this
  is filled by code from data. No model writes it.
- **The responder.** On a claim it is pre-filled with whoever the project
  records as assessing claims, or with the host if nobody is recorded. The
  raiser may change it and the host is told.
- **Numbering.** SiteWise numbers an item per type when it is submitted.
  Numbers run across the project and are never reused.
- **Received and frozen.** The submit time is the received time. A
  submitted item is not edited; a mistake is fixed by withdrawing it and
  lodging another.
- **Due date.** Each type has a default response period (below). A period
  the project records replaces it. Code works out the date and shows how,
  as payment schedule dates are shown today.
- **Late.** An item lodged after the period the project works to is
  accepted and flagged with how late it was. Whether it is barred is the
  assessor's decision, never SiteWise's.
- **Silence.** When a due date passes SiteWise warns ahead and marks the
  item overdue. It changes no figure, for any type.
- **Settling.** A click settles an item. A follow-up from the raiser
  reopens it, and the history shows both.
- **Links.** An EOT claim points at its notice of delay. The time part of a
  variation claim is its own linked EOT claim.

**Default periods.** The owner's reference form is AS 4902-2000, general
conditions for design and construct, not AS 4000. Defaults come from it
where it gives a period. The form itself was not read for this note: every
row is unverified until the owner reads their own copy, as the Security of
Payment rows were. Only the number of days is recorded, never clause text.

| Type | Default | From | Status |
|---|---|---|---|
| RFI, letter | Respond in 5 working days | SiteWise's existing default; no AS 4902 period found | SiteWise's own |
| Notice of delay | No response owed; it is recorded | cl 34.2 asks the builder for prompt notice | Unverified |
| EOT claim, lodging | Within 28 days of becoming aware of the cause; later is flagged | cl 34.3 | Unverified |
| EOT claim, response | 28 days | cl 34.5 | Unverified |
| Progress claim, response | 14 days to certify. Where a Security of Payment Act sets an earlier date, the earlier date wins, as today | cl 37 | Unverified |
| Payment | 7 days after the certificate | cl 37; the figure found is AS 4000's | To check |
| Variation claim, other claims, practical completion | None recorded yet | cl 36, cl 41, cl 34.6 | To read |

On a handshake job these are SiteWise's convention, not the parties'
agreement. The screen says "default", never "the contract requires".

## 8. Responses and results

- **Addressed like email.** The author picks To and Cc. An email reply
  lands on the item as the same kind of message as one written in SiteWise.
- **Formal means responder plus result.** The result list is short and per
  type: answered, more information needed, granted, granted in part,
  rejected, approved at an amount.
- **Standard wording.** Each result offers its usual words, to keep or type
  over, from the type list.
- **Reasons.** A result less than what was claimed needs a reason, as an
  assessed claim row does today.
- **What a result moves.**

| Result on | Moves | Through |
|---|---|---|
| EOT claim | The agreed completion date, where one is recorded | Days granted. The Program's forecast is untouched |
| Variation claim | Approved Variations and Forecast Final Cost | The existing variation decision |
| Progress claim | Certified to date | The existing payment schedule |
| RFI, notice, letter | Nothing | The record only |

- **No date recorded.** If the project records no agreed completion date,
  an EOT result still stands and the register totals the days granted.
  Nothing moves, and the item says why.
- **Approval.** A response that commits the principal's money above a limit
  set on the project waits for the principal's approval. An assessment of
  time or of a payment claim is never held: it is the assessor's own
  judgement.
- **A person writes every response.** Chat drafts when asked.

## 9. What reads what

| Step | Who |
|---|---|
| Type, recipients, responder and figures on an item raised in SiteWise | The person raising it. No Jev |
| Pre-filled fields and standard wording | Go, from the type list, the project's terms and the raiser's last item of that type |
| Numbers, due dates, lateness, the completion date after an EOT, totals | Go |
| Which item an emailed reply belongs to | Go's thread and cited-number rules first, then Jev's existing register question |
| The type of an emailed item | Jev's existing notice pick, its options drawn from the type list |
| Days and dollars in an emailed or uploaded claim | Go harvests, Jev picks among what Go found, amber until checked |
| Whether a reply states a cost or time impact | Jev, yes or no, as a proposal |
| The words of a response | The person. The LLM in chat, only when asked |

Three things to hold:

- A signed-in party needs Jev less than email does, because they state the
  type and type the figures themselves. Sign-in is the cheaper path as well
  as the cleaner record.
- No read or write on any party screen calls Jev or the model. Existing
  budget rows for the register apply; the party's page gets its own.
- Every Jev pick here stays hidden until its answer key is approved, as
  today.

## 10. Taste

The owner asked for this to be tasteful and delightful. In this product
that means the following.

1. **Nobody sets anything up first.** No contract, no settings page and no
   sign-up form stands between a person and their first item or reply.
2. **One list.** No new left-nav entry. A type, a column or a filter
   appears only when the project has rows for it (index, rule 5).
3. **Raising is already half done.** One screen, pre-filled, with a line of
   guidance in each empty field and one Send. "Another like the last one"
   for the builder lodging their seventh weather notice.
4. **An item reads like a letter thread.** Who wrote, when, to whom, what
   was decided. The result is a chip a person can read at a glance:
   "2 of 5 days granted".
5. **Replying feels like email, deciding feels deliberate.** A reply is
   quick. A formal response asks for a result before Send, so nobody
   decides a claim by accident.
6. **A date says how it was worked out.** "Response due Monday 21
   December: 28 days, the default." Unknown is said plainly: "No completion
   date is recorded, so nothing has moved."
7. **A receipt for every submission.** "Received 3:46 pm, Monday 23
   November. EOT-4." On screen and by email, to both sides.
8. **A first visit is the item itself.** The email's link opens what was
   sent. No empty workbench, no tour.
9. **Warn before, not after.** Amber two business days out, as payment
   schedules do.
10. **Quiet.** One email per event, only to those addressed.
11. **Plain words.** Responder, Result, Settled, Late. No "ball in court".
12. **Fast.** Speed is part of the feel. Nothing here waits on a model.

## 11. Reuse, and what is new

| Already built | Used for |
|---|---|
| `project_delivery_items`, its kinds and `details` | Items |
| RFI numbering, never reused | Numbers per type |
| `delivery_settings` | The project's optional terms |
| `delivery.DueDate`, `program.BusinessDayAfter`, the period columns on `awards` | Response periods and due dates |
| `mail_messages`, `mail_parts`, the frozen-record trigger | Messages, attachments, the record |
| The send route and Reply | Sending a response |
| Filing and intake | Attached and uploaded documents |
| `delivery_readings` and the register's Jev question | Matching emailed replies to items |
| The notice pick | Typing emailed items |
| `variations`, `progress_claim_lines`, the payment schedule | Variation and progress claims raised by a party |
| The operations call, `project_changes`, Undo | Every party write, with its actor |
| `Attention` | Warn-only overdue marks |
| Organisations, users, sessions, the Free plan | Outside organisations |
| Emailed-link confirmation, reset and team invites | First sign-in from an item's email |
| `guest_grants` and the `/guest` route pattern, with opens and last opened | Access across organisations; when a party first opened an item |
| Named filters and chat's screen operations | List views |
| The report PDF route | One-item export |
| Chat | Drafting a response on request |

What is new, as few things as the review can make it:

1. A list of types in `knowledge/`: each type's fields, guidance, standard
   wording, default period and result options.
2. Optional project terms: the assessor, the agreed completion date, the
   project's own periods and the approval limit.
3. A grant between an organisation and a project, with a level.
4. A result on a formal response.
5. The party's page: their list and an item's sheet.
6. Notification mail, and first sign-in from its link.
7. A hold for the principal's approval.

## 12. Owner decisions (settled 23 November)

*The record and the parties*

1. SiteWise is the record. It never runs beside Procore.
2. Builders and consultants sign in, in the first build.
3. A person from outside gets their own account, and behind it their own
   organisation, free until it chooses to pay.
4. Any organisation can host a project, a builder included.
5. One project per building with one host. Items record who raised them
   and who they are addressed to, so that a linked project per side can
   come later.
6. A partner has one of two levels: whole project, or register.
7. A party sees only what is addressed or copied to them.
8. The host sees every message on its project.
9. A party keeps read-only access and export for good.

*Items*

10. Types in the first build: RFI, general correspondence, notice of
    delay, EOT claim, variation claim, instruction and progress claim. A
    signed-in builder or consultant may raise RFIs, letters, delay
    notices, EOT claims, variation claims and progress claims.
11. The raiser picks who receives an item.
12. Each item names one responder. A project may record who assesses
    claims; claims then pre-fill with them, the raiser can change it and
    the host is told.
13. SiteWise numbers items per type on submission, across the project.
14. Submission is the received time, and the item is then frozen.
15. Each type has a default response period; one the project records
    replaces it.
16. A late claim is accepted and flagged, never refused.
17. Silence warns only, for every type, progress claims included.

*Responses*

18. A response is a message on the item. Its documents are filed like any
    other.
19. Replies are addressed like email.
20. An item can hold several responses, all standing.
21. A reply is formal only when the named responder sends it with a result.
22. A formal response stores its result as well as its words.
23. The latest result stands. An ongoing delay is further claims.
24. A click settles an item. The raiser's follow-up reopens it.
25. Responses that commit money above a set limit wait for the principal.
    Assessments never wait.
26. A person writes each response. Chat drafts only when asked.
27. Email notifies each party with a link, and an email reply still lands
    on the item.

*Time and money*

28. A granted EOT moves the agreed completion date. The Program's forecast
    is separate.
29. A variation's time is a linked EOT claim.
30. A builder lodges a progress claim by typing on the contract's lines or
    by uploading a PDF to be read.

*Added on the owner's review of the first draft*

31. A contract is never required and there is no register of contracts.
    Terms are optional, set once per project. This changes decisions 5,
    12, 13, 15 and 28 as now worded above.
32. Raising an item is pre-filled and easy: standard wording, guidance in
    each field, one Send.
33. Anyone raising an item may address it to an email address SiteWise has
    never seen. That person is brought in with an account and no sign-up
    form. Some will only ever reply; some will later host.
34. Default times come from AS 4902-2000, not AS 4000.
35. Handing a project to a D&C contractor or a builder is a partner grant,
    not a transfer of the host (evaluated in section 6).

## 13. Rule changes this direction asks for

Each of these reverses or widens something already written down. The
review should confirm each one is meant.

| Today | Becomes |
|---|---|
| A guest is never a seat; RFI answers by link were cut (18 November) | Outside people sign in, as their own organisations |
| Every row and route is isolated to one organisation | A named set of routes reads a host's rows for a granted organisation |
| All mail leaves on the PM's press of Send (index, agreement 7) | Notification mail goes without a press, when someone else submits or responds |
| Sign-up asks for a password before anything else | The emailed link is the confirmation; a password comes later |
| Only an RFI is numbered and only an RFI takes an answer | Every type is numbered and can hold responses |
| A missed payment schedule date counts the claimed amount in red (variations plan, decision 3) | Warn only |
| EOT assessment and moving dates are out of scope | An EOT result moves the agreed completion date |
| Contract time bars were cut: no knowledge file of clause periods | A short table of default days from AS 4902, each unverified until the owner reads the form. Still no clause text. The project's own period wins |

## 14. Risks and practice notes

This is not legal advice. The contract points below come from secondary
sources (section 17); the standard forms themselves were not read.

1. **Service.** An item lodged in SiteWise only counts as given under a
   contract if that contract allows notices to be given that way. A
   construction lawyer should confirm the wording before claims and notices
   run through it.
2. **The payment clock.** A builder's progress claim starts the Security of
   Payment period at submission. That timestamp needs a receipt to both
   sides.
3. **Warn only on payment claims.** After a missed payment schedule date
   the Cost Plan will understate what is payable. The warning should state
   the claimed amount in words.
4. **Silence can decide a claim.** AS 4902 follows AS 4000 here: an EOT
   claim not answered within 28 days is commonly described as granted as
   claimed, and that clause is often amended or removed. The warning should
   state a consequence only when the project's terms record one.
5. **A default is not a contract.** On a job with nothing signed, SiteWise's
   default periods bind nobody. The words on screen must never suggest they
   do.
6. **The assessor's independence.** The person administering a contract
   acts partly as the principal's agent and partly as an independent
   assessor, and interference in an assessment can see it set aside. This
   is why decision 25 never holds an assessment. What a principal may see
   of an assessment before it is issued is still open.
7. **Informal directions.** With replies addressed like email, an architect
   can still answer a builder directly. Only a formal response binds. Every
   other reply should carry a standing line saying it is not a direction.
8. **Anyone can invite.** Decision 33 lets any user send mail from SiteWise
   to an address they type. It needs a rate limit, a link that opens one
   item and nothing more until the person has confirmed themselves, and a
   plain answer to what a forwarded link opens.
9. **Isolation.** This is the first time one organisation reads another's
   rows. It needs its own route list, its own tests and a security review,
   as guest access did.

## 15. Assumptions and open questions

Assumed without asking. The owner should confirm.

- Drafts and internal notes are private to their author's organisation.
- A firm's own admin adds its people.
- One agreed completion date per project in the first build, with no
  separable portions.
- The host's own items to other parties work the same way in reverse.
- Mail from an unknown address that is not a reply is held, as today.
- A party's page has no chat in the first build.
- "Partner, register" never shows contract or commercial documents, or
  messages exchanged only within the host's side (section 6).

Open, with the edge case that raised each.

1. **Who settles.** The owner said "your click". In the symmetric case, is
   that the responder, the raiser or always the host?
2. **A wrong or absent responder.** Who may reassign an item, and does the
   response period restart?
3. **Deleting a project.** "Read-only for good" conflicts with a host
   deleting or abandoning a project that others rely on.
4. **One firm, many organisations.** Three people from one builder are
   each addressed by email and each gets an organisation. How do they find
   each other and join one, and who approves it?
5. **The first link.** When is a password first asked for, when are the
   terms of service accepted, and what does a forwarded link open?
6. **Reply defaults.** Who is pre-selected on a reply, given the risk of an
   accidental reply to all.
7. **Where claims show.** Variations and progress claims live in their own
   tables and on the Cost Plan. Do they also appear in the one list of
   what is owed?
8. **Free plan caps.** Being addressed must not use a person's one active
   project. Which organisation's allowance pays to read a document a party
   uploads?
9. **The party's own "what is due".** What a builder sees as waiting on
   them.
10. **Weather.** Most delay claims on the reference project were for
    weather. Contracts often include an allowance of wet days.
11. **Later.** Separable portions, concurrent delay, delay costs, a linked
    project per side, and a true change of a project's owner.

## 16. For the reviewer

Before any plan is written, check this note against the tree.

- **What was verified.** The statements in section 3, by reading files on
  `main` at `9472e1a`. Nothing was run and no test was executed.
- **What was not.** Line numbers, branches not on `main`, the deployed
  state, anything about Procore beyond four screenshots, and AS 4902
  itself.
- **Test the big choices.**
  1. Is `mail_messages` the right home for a response written in SiteWise,
     given its frozen-record trigger and its sent-message shape check?
  2. Does a grant between organisations stay contained to its own routes,
     and what does the guest plan's finding about member sessions imply
     for a granted organisation's users?
  3. Does "one item idea over three stores" (register, variations, claims)
     hold, or does it leak?
  4. Does the type list belong in `knowledge/` as data, and what must be a
     real column so the list can sort, filter and number by it?
  5. Are the project's terms at home in `delivery_settings`, and can every
     one of them stay optional without a branch in each date calculation?
  6. Can first sign-in from an item's link reuse the invite and reset
     token pattern, and where must it be stricter?
  7. Which existing tests and budget rows must still pass, and which new
     rows are needed.
  8. Can the first slice below ship with no outside organisation at all?
- **Say where this note is wrong.** It was written from one long
  conversation. Where the tree disagrees, the tree wins.

Slices, in the order they depend on each other:

1. Types, easy raising, optional terms and due dates, for one organisation.
2. Responses and results on every type, and the agreed completion date.
3. Outside people: addressed by email, first sign-in, the party's page.
4. Partners at both levels, and a builder as host.
5. List views and one-item export.

Out of scope here: site diaries, defects, photos, review-comment sheets, a
phone app, subcontractors under a builder, a register of contracts, and any
import from Procore.

## 17. Sources (read 23 November 2026)

Secondary sources for sections 7 and 14. None is the contract form itself.

- Standards Australia, AS 4902-2000 listing and contents (the form follows
  AS 4000 and covers design and construct, design development and
  construct, and design, novate and construct):
  https://store.standards.org.au/product/as-4902-2000
- Construction Law Made Easy, chapter 11, extension of time procedures (the
  28 days to claim under AS 4000 and AS 4902):
  https://clme.komosion.com/?p=8903
- Plexa, EOT calculator notes (28 days to respond and the effect of silence
  under AS 4000 and AS 4902; prompt notice of delay):
  https://www.plexapro.com/tools/eot-calculator
- Legal 500, "Navigating an AS 4902-2000 contract" (14 days to certify a
  progress claim):
  https://www.legal500.com/intelligence/australia/real-estate-and-construction/navigating-an-as-4902-2000-contract-legal-traps-and-safeguards
- Mastt, introduction to AS 4000 (7 days to pay after the certificate; an
  AS 4000 figure):
  https://www.mastt.com/blogs/what-is-as4000-introduction-for-client-side-project-managers
- Law Insider, a common amendment to clause 34:
  https://www.lawinsider.com/clause/time-and-progress/_1
- HopgoodGanim, "The superintendent's dual role in construction contracts":
  https://www.hopgoodganim.com.au/news-insights/the-superintendents-dual-role-in-construction-contracts-balancing-agency-and-independence/
- Allens, "Best practice approach to communications between principals and
  superintendents":
  https://www.allens.com.au/insights-news/insights/2023/09/Best-practice-communications/

## 18. Peer review answers (23 November 2026)

The owner accepted the peer review's recommendations and asked the
reviewer to answer the open questions. These answers stand until the owner
changes one.

Rule changes in section 13:

- **Decision 17 is narrowed.** Silence warns only on contract periods. A
  progress claim whose payment schedule date passes unserved still counts at
  the claimed amount under a verified Security of Payment row, as today: the
  NSW Act makes the client liable for it, and the Cost Plan must not show
  less than that.
- **The agreed completion date is its own project term**, never the
  Profile's `proj.target_completion` (which Jev may read and the user owns).
  The date after EOTs is computed at read: agreed date plus the calendar days
  granted. Nothing stored moves.
- **Notification mail goes out as account mail** (the `SendTeamInvite`
  path: a link only, no attachments), never through `queueOutbound`, so it
  takes no TR number and its own limits. Built with slice 3.
- **No AS 4902 table yet.** SiteWise's own defaults are shown as "default,
  check your contract": RFI and letter 5 working days, EOT claim 28 calendar
  days, notice of delay none. No clause numbers on screen until the owner has
  read the form.

Open questions in section 15:

1. **Who settles.** The host, in slices 1 and 2 (only the host is signed in).
   In slice 3: the raiser or the host; never the responder alone.
2. **Reassigning.** The host may change the responder. The response period
   does not restart; the history shows the change.
3. **Deleting a project.** A project with an outside party is archived, not
   deleted, and each party is offered its export first. Slice 3.
4. **One firm, many organisations.** An address whose domain matches an
   existing organisation's verified members is offered to join it, approved
   by that organisation's owner. Slice 3.
5. **The first link.** Opens one item read-only. Terms are accepted before
   the first reply; a password is asked for on the first visit without a
   link. A forwarded link opens the same item read-only and nothing else.
6. **Reply defaults.** To the item's raiser only; Cc is empty.
7. **Where claims show.** Yes, every claim is a register row. A variation or
   progress claim row links to the Cost Plan's record only once an award
   exists; before that it is an item and moves nothing.
8. **Free plan caps.** Being addressed uses no project allowance. Filing a
   party's upload counts against the host, as a guest's quote does today.
9. **The party's own "what is due".** Items where they are the responder,
   by due date. Slice 3.
10. **Weather.** Later: a wet-day allowance becomes a project term.
11. **Later**, as listed.

The first slice built: notice of delay, EOT claim and letter as register
kinds, numbered per kind (NOD-1, EOT-1, LTR-1); due dates from the
project's period or SiteWise's default; an EOT result (outcome, days
granted, reasons) recorded on the item by the host; days granted totalled;
the optional agreed completion date and the date after EOTs; overdue
warnings in `Attention`; chat through the existing register operations.

## 19. Raised by us, and Correspondence (owner, 25 November 2026)

**Direction.** An EOT claim, a notice of delay and a letter are each either
received from another party or raised by the host and lodged with one. A
builder hosting the project lodges its EOT claims with the principal; a
principal or PM receives them. The row keeps `details.direction`: `in` (or
blank, every row before this) or `out`. Nothing else changes in the data, so
there is no migration.

- A raised item is with whoever the row names (the party who owes the
  reply), never "You" by default. Once their result or reply is recorded, it
  is with the host.
- Its date is the day it was lodged (a letter: sent), in `received_on`; its
  due date is the other party's period from that day, worked out exactly as
  a received item's. Lateness then means they are late.
- The result form reads "Their result", recorded as it came back. The days
  granted and the date after EOTs count the same either way.
- Chat takes `direction` (in, out, or the words received and raised) on
  `delivery.add` and `delivery.edit`; its register read says "raised by you".

Not in this step: the builder's own time bar for lodging a claim, counted
from when the delay began. That period is the contract's, and like the
others it would show as SiteWise's default until the owner reads the form.
Sending a raised item through mail is also later.

**Correspondence.** The register and the project's messages leave the
right-hand panel, which is the documents only. Correspondence is on the left
nav in Specifications' place (off the nav until it is ready), with tabs
Register (the default) and Messages. The register is one list: Type, No.,
Subject, Raised by, With, Issued, Due and Status. A heading click sorts; a
drag moves the column, and the order is kept in the browser for that viewer.
A row drills in to its sheet, with a breadcrumb (Correspondence › EOT-1)
back. The sheet is laid out in an RFP's bands: particulars, the message,
the answer or result, and history. The project report is Report on the left
nav, under Correspondence, with an RFP's section bands.

**Breadcrumbs.** Every drilled-in view's Back is now a breadcrumb trail:
each step above is a link back, and the last names what is open (a document,
a price table, the comparison report, an RFP, a register row).
