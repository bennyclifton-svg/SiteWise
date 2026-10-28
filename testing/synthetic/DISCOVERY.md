# Synthetic corpus: Phase 0 discovery

Date: 28 October 2026. Read from the code on `main` (8a6a6c8 plus the
uncommitted working tree). Nothing in the application was changed.

Each finding cites the file it came from. Where I could not verify something
by reading, it is marked **unverified** and becomes a pilot check.

## 1. The short version

1. **The "classifier" is Jev (TypeSafe System One), behind deterministic Go
   rules.** Code harvests candidate values. Code settles what it can. Anything
   still open goes to Jev as one multiple-choice call per document. Jev only
   picks among options code offers. It never writes free text. There is no
   keyword model, no trained classifier and no training data in this repo.
2. **There are three drop targets, not one.** They are the repository
   (`POST /projects/{id}/files`), a procurement row
   (`POST /projects/{id}/procurement/packages/{pkg}/submissions`) and the
   Invoices tab (`POST /projects/{id}/invoices`). A fee proposal dropped in
   the repository is detected and gets its own procurement row. An invoice
   dropped in the repository is only filed. It never reaches the ledger.
3. **Markdown is a real input format** (`.md`, `.markdown`), so a Markdown
   corpus goes through the production path. Markdown has its own identity
   reader with strict rules (§4.3). The two most important rules: **the
   filename is never harvested** for number, revision or date, and **only
   `Label: value` lines before the first `##` heading count as a title
   block.** A Markdown table or fixed-width title block gives no identity
   values.
4. **Automatic supersession cannot happen today.** It needs a green Jev
   answer to the `supersedes` question. `data/intake/thresholds.json` has no
   cut-off for that question, so it never goes green and no link is written
   (`internal/intake/decisions.go` `priorToLink`). Every expected
   `superseded_by` will score as missed until that changes. The harness will
   still measure it, because this is a priority target.
5. **Green is withheld everywhere.** Kind can be amber at confidence ≥ 0.8
   and discipline at ≥ 0.6. Number, revision, title, date and lifecycle have
   no Jev cut-off, so only rule-settled values are applied. The profile
   floors are provisional (amber 0.6, green null). Scoring "green only" would
   give zero everywhere, so the scoring has to count amber (question 3).
6. **Several dimensions in the brief have no field in the app.** These are
   document status or purpose of issue, originating organisation, RFIs,
   retention, cumulative-to-date, variation links, and the work types
   demolition and change of use (§9). I propose keying them anyway and
   reporting them as coverage gaps, not as accuracy errors (question 2).

## 2. Ingestion pipeline

### 2.1 Repository drop (filing)

| Step | What happens | Where |
|---|---|---|
| Upload | The filename is cleaned (base name, ≤ 255 bytes). Bytes are stored by SHA-256. One `files` row and one `documents` row (status `pending`) and an `intake` job are committed in one transaction. An empty file gets reason `empty`. | `internal/intake/upload.go` |
| Format | By extension only: `.pdf`, `.docx`, `.xlsx`, `.md`/`.markdown`. Anything else is `not_filed` / `unsupported_format`. | `internal/intake/run.go` `formatOf` |
| Text | Markdown must be valid UTF-8 with no NUL bytes, otherwise `unreadable`. A BOM is stripped. Each non-blank line becomes one run. ATX headings are marked. Tables, links and code are kept verbatim. | `internal/identity/markdown.go` |
| Harvest | Markdown: `markdownIdentity` only (§4.3). PDF, DOCX and XLSX also harvest the filename and spatial title-block cells. | `internal/intake/candidates.go` `Harvest` |
| Rules | `Decide` settles number, revision, title and date when the evidence agrees. `settleVocabulary` settles kind, discipline and lifecycle from explicit captions, email headers or a fee-proposal heading. | `rules.go`, `decisions.go`, `fee_proposal.go` |
| Jev | One fan-out for every field still open: kind, discipline, lifecycle, number, revision, title, date and supersedes. The interactive deadline applies. On a timeout the open fields go grey and a `jev_retry` job is queued. | `file.go`, `draft.go`, `questions.go` |
| Commit | Decisions are upserted, `document_number` and `revision` are set, and a supersession link is written if allowed. Then `full_text`, visual review and `profile_inputs` are queued and procurement submissions are matched. | `internal/store/filing.go` `CommitFiling` |
| After file | `AfterFile` starts the proposal reader (if the text looks like a proposal) and the program importer. | `internal/httpapi/server.go` |
| Background | `full_text` writes passages. `label` and `evidence` jobs read the profile from passages (§5). | `internal/jobs/worker.go` |

Order and chunking. Each document is filed on its own. Filing has no
chunking. The profile reads per passage (`passages` table) and builds the
Jev state per passage. Documents are not reconciled at filing time. The only
cross-document filing link is supersession. The profile is where documents
are reconciled (§5.4).

### 2.2 Proposal drop (fees and consultants)

`proposals.Cue` (`internal/proposals/harvest.go:971`) treats a filed document
as a proposal when the filename or text contains one of: "fee proposal",
"fee submission", "fee estimate", "fee quote", "fee schedule", "tender
submission", "quotation", "response form" or "proposal for". It also accepts
a document whose kind is `commercial` and which contains "fee", "proposal",
"quote" or "tender". A cued document opens a `submission`, and
`EnsureProposalPackage` creates a procurement row for that discipline if
none exists. That row is how "required consultants" get filled from
documents. Proposal reading is a separate fan-out after filing.

Keyword traps follow directly from this. An invoice that says "as per our
quotation" is cued as a proposal. A builder quote headed "Tax Invoice" is
cued only if it also contains one of those words.

### 2.3 Invoice drop

`POST /projects/{id}/invoices` creates an `invoices` row and a `files` row,
**not a `documents` row**. An invoice dropped there does not appear in the
document register, and an invoice dropped in the repository does not appear
in the ledger. I have not confirmed whether the UI drops one file into both.
I think it does not.

### 2.4 Duplicates

- Same bytes in the same project: one filing. A re-drop returns the existing
  document (`documents_file_uq`, `files` unique on `(org, project, sha256)`).
  The same applies to invoices.
- Different bytes with the same number and revision in the same project:
  `documents_identity_uq` is unique on (org, project, number, revision,
  identity_page). `SetFiledIdentity` has no conflict handling. **Unverified,
  suspected:** the second filing's commit fails with a unique violation, the
  job retries, and the document stays `pending`. Near-duplicates in the
  pilot will test this.

## 3. The Jev call

- The client is `internal/jev`, configured by `SITEWISE_JEV_API_KEY` and
  `SITEWISE_JEV_MODEL`. Live calls are paid and need the key. Tests use fakes.
  The eval tools record and replay exchanges.
- The request state is the filename, the labelled title-block text, an
  excerpt of the first page (2,400 characters, head and tail), lines naming
  the author (containing ©, "copyright", "prepared by", "Pty Ltd" or "@"),
  up to 20 headings and up to 64 candidates (`questions.go`
  `filingStateOf`).
- Questions are `choice` questions with role instructions ("which candidate
  is this sheet's own number… not a project number…"). Markdown documents get
  a stricter discipline question. It asks for the *issuer's* stated
  profession. An owner email, an unstated profession or "Studio" in a company
  name gives `none` (`draft.go` `open`).
- Confidence routing (`decisions.go` `Thresholds.Band`): a cut-off applies
  only to its question version (`intake-77`) and option-count shape. No
  cut-off means the answer is not applied (blank). Live values are in
  `data/intake/thresholds.json`:

| Question | Shape (options) | Amber | Green | Effect |
|---|---|---|---|---|
| kind | 9–16 | 0.8 | none | amber at most |
| discipline | 33–64 | 0.6 | none | amber at most |
| lifecycle, number, revision, title, date | — | — | — | Jev answer never applied |
| supersedes | — | — | — | never links (§4.5) |

- Low confidence means blank, not applied, with the confidence stored. A
  timeout means grey, rule values stand, and a retry is queued. A field with
  conflicting inputs stays open and goes to Jev. There is no "most frequent
  wins" rule.
- `docs/` evidence shows the owner saw response variation between live runs
  ("Green withheld after response variation produced errors"). **The same
  corpus can score differently on two runs.** The harness must record every
  exchange so a run can be replayed and reproduced.

## 4. Document repository and register

### 4.1 Data model

`documents` (`001_init.sql`, `003`, `007`): `filename`, `status`
(`pending`, `filed`, `not_filed`), `reason`, `document_number`, `revision`
and `identity_page`. Each field is a row in `decisions` with `field`,
`value`, `band` (green, amber, blank or grey), `decided_by` (rule, jev or
user), `question_version`, `confidence` and `version`. Supersession is a
row in `supersessions` (document → prior document). A user correction
(`PUT /documents/{id}/fields/{field}`) is final and is never overwritten.

**Register fields the app has:** `kind`, `discipline`, `lifecycle`,
`number`, `revision`, `title`, `date`, `supersedes`.

**Register fields the brief asks for that the app does not have:**
`status` (purpose of issue: preliminary, for coordination, for DA, for
tender, for construction, as-built) and `originating organisation`. Issue
purpose is deliberately not filed: "Lifecycle is filing purpose, not the
issue stamp. Technical drawings remain Design even when issued for
construction (owner decision)" (`decisions.go` `settleVocabulary`).

### 4.2 Taxonomies

**Kind** (`data/intake/kinds.json`, 12 of a planned 16, `complete: false`):
`design_brief`, `drawing`, `specification`, `report`, `certificate`,
`correspondence`, `contract`, `commercial`, `schedule`,
`statutory_instrument`, `photo`, `unknown`.

The brief's document types map onto these kinds as follows. This mapping is
my proposal; question 15 asks you to confirm it.

| Brief type | Proposed kind | Note |
|---|---|---|
| Drawing, SK sketch | `drawing` | Jev instructions: drawing sets including general-notes sheets are Drawing |
| Specification, product data sheet | `specification` | |
| Door, finishes, lighting, equipment or beam schedule; drawing register | `schedule` | "Tabulated schedules and registers are Schedule" |
| Calculations, geotech, survey, BCA, planning report | `report` | |
| Transmittal, email, letter, RFI, council letter | `correspondence` | An RFI has no kind of its own |
| Meeting minutes | `report` or `correspondence` | not stated anywhere; **needs your call** |
| Fee proposal, builder quote, cost plan, invoice, payment claim, credit note | `commercial` | "Commercial is pricing or payments" |
| Signed contract, deed, appointment | `contract` | |
| Planning certificate, s10.7, BASIX certificate | `certificate` | |
| Legislation, NCC or code extract, LEP clause | `statutory_instrument` | |
| Master programme | `schedule` | rule: heading `# Master Programme` |

**Discipline** (`data/intake/disciplines.json`, 58 ids): 47 consultant ids,
from `consultant.access` to `consultant.workplace`, including civil,
structural, mechanical, electrical, hydraulic, fire_services,
fire_engineer, landscape, surveyor, geotechnical, acoustic, esd, facade,
vertical_transport, ict, security, bca, access, traffic, interior_design,
town_planner, heritage, bushfire, quantity_surveyor, certifier, arborist,
waste, hazmat and project_manager. There are 10 trade or supplier ids:
`trade.main_works`, `trade.civil_works`, `trade.electrical`,
`trade.mechanical`, `trade.hydraulic_plumbing`, `trade.structural_steel`,
`trade.fire_services`, `trade.joinery_kitchens` and
`supplier.windows_glazing`. Plus `none` ("Not established"). There is no
council, authority or client discipline, so those authors are `none`.

**Lifecycle** (`data/intake/lifecycle.json`): `brief`, `cost`,
`consultant`, `design`, `authorities`, `procurement`, `construction`,
`reporting`, `project`. Drawings are always `design` by rule. A fee
proposal heading gives `consultant`.

### 4.3 How identity is parsed from Markdown (`internal/intake/markdown_identity.go`)

**Emails.** An email is detected when the first line is `From:` and the next
up to seven lines are header labels (`To`, `Cc`, `Bcc`, `Subject`, `Date`,
`Sent`) including both `To` and `Subject`. Then:

- kind = `correspondence` (green, by rule),
- title = the first `Subject`,
- date = the **latest** `Date:`/`Sent:` header date across every quoted
  `From:` block in the thread.

Number and revision are never read from an email.

**Other documents.** Only the lines before the first `##`, `---`, code fence
or `Dear …` line are read:

- `# Heading` gives a title candidate.
- `Title:` / `Document title:` give a title.
- `Ref`, `Reference`, `Report no`, `Document number`, `Drawing number`,
  `Drawing no`, `Proposal ref/no/number`, `Quote no/number/ref` and
  `Quotation no/number/reference` give a number, which must match
  `^[A-Za-z0-9][A-Za-z0-9./_-]{1,79}$`. A number with a space ("A 101") is
  rejected.
- `Revision` / `Rev` / `Version` give a revision, from the first token only.
- `Date` / `Issue date` / `Report date` give a date. Accepted formats: `2
  January 2006`, `2 Jan 2006`, `2006-01-02`, `2/1/2006`, `2.1.2006`,
  `2-1-2006`. "28 Oct. 2026", "Oct 28, 2026" and "28th October" are not
  accepted.
- A bare line that is a valid date gives a date candidate.
- Emphasis (`**Rev:** C`) is stripped. Labels are case-insensitive. A
  trailing `.` on the label is dropped.
- Not recognised: `Drawing No.` written as `Dwg No`, `Sheet`, `Job No`,
  `Status`, `Purpose of issue`, `Discipline`, `Client`. Also not recognised:
  anything inside a table (`| Drawing No | A-101 |`), a fixed-width block or
  a code fence, and anything after the first `##`.

**Filename.** Not harvested for Markdown. Jev sees it only as context for
kind and discipline. A Markdown file therefore **cannot** have a filename
that contradicts its title block in a way the rules see. That edge case only
exists for PDF, DOCX and XLSX (question 6).

**Captions that settle vocabulary** (any line, `decisions.go`):
`Kind: Drawing` or a bare line `Drawing`; `Discipline: <label or alias>`,
exact and unique; `Lifecycle: <label>`. Any line *starting* with "Drawing
No", "Drawing number", "Dwg No" or "Drg No" makes lifecycle `design` by
rule. Captions conflict when more than one phrase matches, and then nothing
is settled.

**Rule settlement** (`rules.go` `Decide`). If exactly one distinct value is
harvested for a field, it is settled green by rule (`unique`). Several
distinct values stay open and go to Jev, whose number, revision, title and
date answers are never applied today. A Markdown document with two `#`
headings, or a `# Heading` that differs from `Title:`, therefore gets **no
title**.

### 4.4 Revision ordering (`internal/intake/revision.go`)

- There are three kinds of series: letters (`A`…`ZZ`, ranked A=1, Z=26,
  AA=27), plain numbers (`0`…`999`, leading zeros dropped), and a one-letter
  prefix plus a number (`P1`, `C01`, ranked by the number). `Rev`/`Revision`
  prefixes are stripped.
- **Different series are never comparable.** `P3` → `C1` (the switch from
  preliminary to construction) and `B` → `1` give no supersession.
- `1.2`, `v1.2`, `Draft v1.0` and `Final v2.0` are accepted as literal
  revisions but are never ordered, so reports with versions never supersede.
- Number matching for supersession uses `normalizeNumber`: spaces are
  removed, `_` and dashes become `-`, and the result is upper-cased. `A101`
  and `A-101` are different numbers. `a-101` and `A_101` match `A-101`.

### 4.5 Supersession

`priorToLink` (`decisions.go`) links a new filing to a prior one only when
all of these hold:

1. Jev answered `supersedes` green (impossible today, see §1.4).
2. The number is applied and matches the prior's number after normalising.
3. The new revision is strictly later than the prior's, in the same series.
4. The link would not form a cycle.

Candidate priors are only the project's documents already filed with a
matching number. Two consequences follow:

- **Out-of-order uploads.** When rev C is filed before rev B, C has no prior
  when it is filed. When B arrives, B is earlier than C, so no link is
  written. Nothing re-links C → B afterwards. Even with a supersession
  threshold, out-of-order uploads would leave the older revision looking
  current.
- **Same number, different consultant.** Candidates are matched on number
  only. Author, discipline and title are not compared.

The profile ignores superseded documents (`internal/profile/reconcile.go`).
With no links written, it reads every revision.

## 5. Project profile

### 5.1 What it holds (`knowledge/profile/*.yaml`, `knowledge/determinants.yaml`, migration `008`)

Answer keys will use the row keys the existing profile answer keys already
use (`data/eval/profile/answer-keys/*.yaml`):

**Header**

- `hdr.building_class`: `residential`, `commercial`, `industrial`,
  `institution`, `mixed` or `infrastructure`.
- `hdr.subclass`: depends on the class. Residential has `house`,
  `apartments`, `townhouses`, `btr`, `student_housing`, `retirement_living`,
  `residential_aged_care`, `social_affordable_housing` and `other`.
  Commercial has `office`, `retail_shopping_centre`, `retail_standalone`,
  `hotel`, `food_beverage`, `serviced_office_coworking` and `other`.
  Industrial has `warehouse`, `logistics_ecommerce`, `manufacturing`,
  `cold_storage`, `data_centre` and 7 others. Institution has education,
  healthcare, `government_civic` and `religious`. Mixed has
  `residential_retail`, `residential_commercial`, `hotel_residential`,
  `retail_office`, `btr_retail` and `vertical_village`. Infrastructure has 7.
- `hdr.work_type`: `new`, `refurb` (shown as "Refurbishment / fit-out /
  upgrade"), `extend`, `remediation` or `advisory`. **There is no
  demolition, change-of-use or combination type.** Demolition appears as a
  `remove` action on work items (`knowledge/works/actions.yaml`).
- `hdr.scale.*`: per subclass, for example `site_sqm`, `gfa_sqm`,
  `existing_gfa_sqm`, `added_gfa_sqm` and storeys.

**Determinants** (`det.*`, 75 in `determinants.yaml`)

- Site conditions: `ncc_class` (1a…10c, several values allowed), `state`,
  `bal`, `bushfire_prone_land`, `flood_hazard`, `flood_planning_level`,
  `heritage_status` (`none`, `conservation_area`, `local_item`,
  `state_register`), `acid_sulfate_soils_mapped`,
  `potentially_contaminated_land`, `site_class`, `wind_class`,
  `climate_zone`, `existing_building`, `existing_building_year`,
  `groundwater_above_excavation`, `max_excavation_depth`, `saline_soil`,
  `noise_affected_site`, `termite_prone_area`.
- Building: `rise_in_storeys`, `effective_height`, `sprinklered`,
  `type_of_construction`, `commercial_kitchen` (a keyword trap in its own
  right), `basix_required`, `nathers_min_stars`, `estimated_works_cost`,
  `compliance_pathway`, and others.

**Facts:** `fact.contract_form` (`as4902`, `as4000`, `as2124`, `abic`,
`domestic_hia`, `domestic_mba`, `other`), `contract_basis`,
`consent_number`, `consent_date`, `consent_lapse_date`,
`defects_liability_period`, `design_life`.

**Details** (`proj.*`, `site.*`): `client`, `lga`, `zone`, `address`, `lot`,
`program`, `target_da_lodgement`, `target_construction_start`,
`preliminary_estimate`, `budget_stated`, `sustainability`, and others. There
are also 30 flag readings (`groundwater`, `basement`,
`excavation_adjoining`, `rail`, `state_road`, `cl46`, `panel`,
`architect_lodges`, arborist and fire flags).

**Conditions** (`taxonomy.yaml`):

- `planning`: `exempt`, `cdc`, `da`, `ssd`, `planning_permit` (VIC). REF,
  modification, planning proposal and advice-only are user-only values and
  are never offered to Jev.
- `procurement_route`: `traditional`, `design_construct`, `eci`,
  `managing_contractor`, `alliance`, `ppp`. **There is no construction
  management route and no novated D&C route.** Novation is a per-package
  flag (no, option or yes).
- Also `access_constraints`, `operational_constraints`, `quality_level`,
  `stakeholder_complexity`, `environmental_sensitivity`, and two
  industrial-only conditions.

**Systems:** a presence value (`included`, `not_included`, `not_stated`)
and a provider value for each building system id in `systems.yaml`.

### 5.2 Which documents the profile reads (`data/profile/reading.json`)

Only documents whose filed `kind` is one of `design_brief`,
`specification`, `report`, `contract`, `commercial`, `certificate`,
`statutory_instrument` or `schedule` are read. **Drawings, photos,
correspondence and documents with no kind are skipped** unless the user sets
the document to `read`. This has three effects:

- A fact that lives only in a client email or a forwarded chain is never
  read.
- A document whose kind came back blank (Jev below 0.8) is silently skipped,
  so a filing miss turns into a profile miss.
- An owner brief written as an email is skipped.

### 5.3 How it reads

Determinants have regex `triggers`. A passage is asked a question only when
a trigger matches (for example `heritage`, `bal[- ]?(low|12\.5|…)`,
`\b(nsw|vic|…)\s+\d{4}\b`). One fan-out per passage. "Not stated" is not
evidence. An enumerated `none` (for example "not a heritage item") is
evidence. Floors are provisional (amber 0.6, action 0.85), with
`approved_by_owner: false`.

### 5.4 Conflicts (`internal/profile/reconcile.go` `Reconcile`)

- A user value is final.
- Superseded documents are ignored.
- Quotes and tenders count as one source together.
- **Two different values on one part is a conflict.** It is shown with its
  alternatives and support counts. **There is no "later wins" and no
  "council beats client" rule.**
- Allowances and requirements are shown but never derived from.

The brief's resolution rules (authority, recency) are therefore not app
behaviour. Question 4 asks how to score them.

### 5.5 When it runs

Filing queues `full_text`, which leads to `label` and `evidence` jobs. A user
"Update" (`POST /projects/{id}/profile/read`) re-reads stale documents.
Memory notes show the dev server is often run with
`-background-backlog=false`. **Unverified:** whether new uploads are read
without an Update click on the server the harness will use.

## 6. Procurement and required consultants

`packages` (`017`, `021q`, `021r`, `031`): `kind` (`services`, `works`,
`supply`), `works_scope` (`head_contract`, `trade`), `discipline_id`,
`title`, `novation`, `lifecycle_status` (`proposed`, `planned`,
`procuring`, `appointed`, `closed`, `cancelled`), `origin`,
`review_status`, `engaged_by` and `supplier_abn`. A package's
`package_stages` hold its fee stages.

There are two ways packages appear:

1. **Suggested by code from the profile.**
   `knowledge/works/package_defaults.yaml` holds baselines by work type and
   building class. For example, new and extend outside infrastructure
   suggest Architect, Structural, Civil, Town Planner, Geotechnical,
   Surveyor, Certifier, Electrical and Hydraulic. Commercial, industrial,
   institution and mixed add Traffic, BCA, Access, Mechanical and Fire
   Services. The file also adds packages for systems in scope. These are
   transient suggestions on the board until a user accepts them. No Jev call
   is made. Expected consultants therefore follow from the expected profile,
   so a profile error carries through to them.
2. **Created by a fee proposal** filed in the repository (§2.2).

The head contract is proposed from gaps. `BuilderDesigns(route)` treats any
route other than `traditional`, including an unanswered one, as D&C.

## 7. Fees

`submissions` (`032`, `034`, `041`): `kind` (`fee_proposal`, `tender`),
`status` (`reading`, `received`, `needs_review`, `awarded`, `unsuccessful`,
`superseded`, `withdrawn`, `not_proposal`), `stated_total_cents`,
`gst_basis` (`ex`, `inc`, `unknown`), `flags`, `superseded_by`, and the
firm (`firms`: name, ABN, address, contact, email, phone).

`submission_prices` hold one cell per schedule row (`schedule_items.kind`:
`stage`, `element`, `option`, `reimbursable`, `rate`, `criterion`). Each
cell has `amount_cents`, `state` (`priced`, `not_priced`, `included`,
`excluded`) and `basis` (for example "6% of the construction budget" or
"quoted inc GST, ÷ 1.1"). Money is integer cents, AUD, **ex GST**. Unknown
is null. Builder tenders are read by `internal/proposals/builder.go`. Award
copies priced cells into the cost plan, and `cost_line_sources` records
which stage each line holds.

**Unverified:** the exact field list the proposal reader asks Jev for. I
will read `internal/proposals/read.go` and `offer.go` before writing
`fees.json`.

## 8. Invoices

`invoices` (`031`): `kind` (`invoice`, `credit_note`, `not_invoice`),
`number`, `invoice_date`, `period` (the certification month, set to the
upload month by rule), `package_id`, `issuer_abn`, `currency`,
`ex_gst_cents`, `gst_cents`, `total_cents`, `status` (`reading`, `filed`,
`needs_review`, `not_read`, `void`, `superseded`), `invoice_count` and
`superseded_by`. `invoice_allocations` link an invoice to cost lines, with
splits and an Unallocated row. The Jev and rule fields are `kind`,
`number`, `date`, `issuer_abn`, `total`, `gst`, `ex_gst`, `supplier`,
`line`, `period` and `currency` (`internal/invoices/read.go`).

**How invoices link.** An invoice links to its supplier through the package
(an ABN match is set only after a user confirms it). It links to a fee stage
only through its allocation to an awarded cost line whose
`cost_line_sources.item_key` is that stage. There is no link to a
variation, because Variations is a placeholder. There is no contract entity
apart from the award.

**Flags code raises** (`internal/invoices/ledger.go` `Check`): `duplicate`
(same supplier and number), `look_alike` (same supplier, total and date
under a different number), `arithmetic` (ex + GST ≠ total by more than 1
cent), `gst_rate` (GST not 10%), `gst_free` (information only),
`abn_invalid` (checksum), `abn_differs` (from the package's confirmed ABN),
`split_total`, `period`, `odd_date` (future-dated), `credit_note`,
`not_invoice`, `currency`, `many` (more than one invoice number in the
file), `amount_unknown`, `unmapped` and `no_supplier`.

**Anomalies in the brief with no flag:** an invoice over the agreed fee or
fee stage, an unapproved or non-existent variation, the wrong project or
client, retention, cumulative versus this-period, and claim-period wording
under security-of-payment legislation. `no_supplier` partly covers "a
consultant never engaged".

## 9. What the brief asks for and the app does not model

| Brief item | App | Proposed treatment |
|---|---|---|
| Register `status` / purpose of issue | not a field; owner decision | key it, `supported: false` |
| Register originating organisation | not a field (Jev reads author cues for discipline only) | key it, `supported: false` |
| RFIs and the RFI register | no model; RFIs file as `correspondence` | key RFI documents in the register; `rfis.json` all `supported: false` |
| Work type demolition, change of use, combinations | 5 single work types; demolition is a work-item action | key `hdr.work_type` with the app's value, and the brief's dimension in `notes.md` |
| Procurement: construction management, novated D&C | not routes; novation is a package flag | map: novated D&C = `design_construct` plus package novation `yes`; CM = `supported: false` |
| Retention, cumulative-to-date, claim period, variation link | not fields | key them, `supported: false` |
| Invoice over fee, wrong project, never-engaged consultant | no flag except `no_supplier` | key the expected flag, `supported: false` where no code exists |
| Supersession by Jev | no threshold, never links | key it; it will score as missed, recorded in FINDINGS |

## 10. Jurisdiction encoded in the code

- **NCC 2022** classes and DTS tables (`knowledge/tables/*.yaml`).
  Standards: AS 3959 (BAL), AS 2870 (site class), AS 4055 (wind),
  AS 3660 (termites).
- **NSW-centric planning.** LGA, LEP zone, CDC/DA/SSD, cl 4.6
  (`flag.cl46`), local planning panel, TfNSW referral, BASIX, s10.7
  certificates and NorBE. VIC appears only as `planning_permit`.
- `det.state` reads all eight states and territories from address patterns
  like "NSW 2150".
- Money is AUD, GST at 10%, with the ABN checksum enforced.
- Contract forms are AS 4000, AS 4902, AS 2124, ABIC, HIA and MBA.

## 11. Existing tests and harnesses

- **Go `testing`.** Database tests need `SITEWISE_TEST_DATABASE_URL`
  (Postgres 17). Fixtures are in `testdata/identity/` and the packages'
  `testdata/` folders. `tools/check.ps1` runs the gates.
- **Web.** Vitest for unit tests and Playwright (`web/tests`) for browser
  tests.
- **Eval harnesses to reuse.**
  - `cmd/intake-eval`: record and replay Jev, fit thresholds, gate against a
    baseline.
  - `cmd/corpus-eval` plus `tools/corpus-upload.py`: **uploads through the
    running local API** (`127.0.0.1:8080` only, needs `--allow-live-jev`,
    one fresh QA project per corpus, never deletes or corrects). This is the
    closest match to drag-and-drop.
  - `cmd/profile-eval`, `cmd/invoice-eval` and `cmd/proposal-eval`, plus
    `cmd/intake-bench` for latency against `bench/budgets.json`.
- **Answer-key conventions to keep.**
  - Profile YAML: `accept: [...]`, `blank_ok`, `assertion`, `evidence`,
    `reviewed: false`.
  - Intake scoring: byte-identical files count as one filing, and splits
    are by document family.
- Real-project data lives in `private/` folders, which Git ignores, because
  **the repository is public**. The synthetic corpus is fictional, so it can
  be committed. I will use reserved domains (`example.com`), ABNs that fail
  the checksum on purpose only where a test needs that, and invented
  lot/DP numbers.

## 12. Proposed harness shape (for comment, not built)

1. Start the local dev server against a scratch database, with live Jev.
2. For each scenario and each wave:
   - Create a QA project.
   - POST every corpus file of that wave to its drop target from
     `manifest.json`: repository, procurement row or Invoices tab.
   - Wait on SSE or poll until filing and background jobs are idle.
   - If allowed, call profile Update.
   - Snapshot `GET /documents`, `/profile`, `/procurement/board`,
     `/invoices` and the submissions.
3. Record every Jev exchange so the run can be replayed.
4. Score the snapshots against `expected/`:
   - Error classes: wrong value, missed value, hallucinated value (filled
     where null was expected) and wrong conflict resolution.
   - Report by field, dimension, tier and scenario, with confusion matrices
     for kind and discipline.
5. Write `REPORT.md` and `report.json`.

The only gap from a real drag-and-drop is that the browser's drop handler
is replaced by the same HTTP POST it makes. I will confirm the request shape
from `web/src` before building.

## 13. Questions for you

1. **Live Jev cost.** The pilot needs live, paid Jev calls through the local
   server. That is about 3 × 15 filings, plus the profile passage calls and
   the proposal and invoice reads. The full 30 scenarios are roughly ten
   times that. Is that authorised, and which server should it use: a fresh
   dev instance on a scratch database, or `:8080`?
2. **Fields the app does not model** (§9). Should I key them with
   `supported: false` and report them as coverage gaps, kept apart from
   accuracy? Or leave them out of `expected/`?
3. **Bands.** Green is withheld everywhere. Should an amber value count as
   the app's answer? I propose yes, with the band reported next to it so
   "right but amber" and "right and green" can be told apart later.
4. **Conflicts.** The app shows a conflict rather than picking a winner.
   Should the answer key hold the true value, with three outcomes: correct
   single value, conflict that includes the true value, wrong? Or should a
   shown conflict count as the correct behaviour when the documents really
   disagree? This decides the "wrong conflict resolution" category.
5. **Title blocks in Markdown.** The app reads only `Label: value` lines
   before the first `##`. I propose a mix:
   - Clean tier: label lines the app can read.
   - Messy tier: tables and fixed-width blocks, which is what a PDF-to-text
     conversion produces, and which the app will mostly miss.
   - Adversarial tier: label lines with traps.

   Is that the test you want, or should every title block be written in the
   readable form so the corpus measures judgement rather than format?
6. **Filename versus title block.** For Markdown the filename is never
   harvested, so that edge case tests nothing on this path. Should I
   (a) keep a few cases and expect the title block to win, or (b) also
   render some drawings as PDF? Option (b) needs a PDF generator, which
   means a new dependency.
7. **Expected profile scope.** Should the expected profile be what a correct
   app *should* conclude from the whole corpus, emails included? Or only
   what the kinds the app reads (§5.2) can support? I propose the first,
   with each value tagged by its source document, so the report can separate
   "missed by design (email not read)" from "missed".
8. **Profile Update.** May the harness press Update
   (`POST /profile/read`) after each wave, as a user would? Or should it
   measure only what arrives without a click?
9. **Procurement expectations.** Should `procurement.json` score:
   - the suggested board,
   - the packages created from fee proposals,
   - `conditions.procurement_route`, and
   - the novation flags?

   Suggestions are code driven by the profile. Should they be scored against
   the *true* profile or against what the app's own profile said? I propose
   the true profile, with the knock-on errors labelled as such.
10. **Jurisdiction mix.** I propose about 80% NSW, plus a few VIC and QLD
    scenarios to test `det.state` and `planning_permit`. Agreed?
11. **Scale extremes.** Do you want one 100-plus document scenario in the
    full set, given the Jev cost? I would also keep one irrelevant-only
    corpus and two very small ones.
12. **Kind mapping** (§4.2). Please confirm the brief-to-kind table. In
    particular: meeting minutes, transmittals, drawing registers,
    calculations and payment claims.
13. **Discipline for parties with no catalogue entry.** Council, client,
    builder and supplier. I propose:
    - council → `none`,
    - client or owner → `none`,
    - head contractor → `trade.main_works`,
    - glazing supplier → `supplier.windows_glazing`.
14. **Drop targets.** Should invoices go only to the Invoices tab, so they
    are absent from the register? Or also into the repository? Fee
    proposals: mostly the repository, with a few dropped on a row?
15. **Lifecycle in the register.** Key it per document (it is a register
    field), with the mapping in §4.2?
16. **Near-duplicates with the same number and revision.** I suspect these
    leave the second filing stuck as `pending` (§2.4). Should I include
    them in the pilot to confirm, knowing they may leave a QA project with
    a failing job?
17. **Branch and commits.** The working tree has many uncommitted changes
    from other sessions. I will only add files under `testing/synthetic/`.
    Should I commit the corpus on a new branch (for example
    `testing/synthetic-corpus`), or leave it uncommitted for you?
18. **Pilot choice.** I propose:
    - **S001, clean:** an NSW alterations and additions house, DA,
      traditional lump sum, about 12 documents.
    - **S002, messy:** an NSW mixed-use apartments-over-retail project, DA,
      novated D&C, with out-of-order revisions, superseded fee proposals,
      consultant invoices and a builder progress claim, about 25 documents.
    - **S003, adversarial:** a QLD warehouse fit-out with negations, keyword
      traps, distractor projects, duplicates and an irrelevant-only side
      folder, about 20 documents.

    Does that suit you?

## 14. Assumed or not verified

- **Unverified:** that new uploads are profile-read without an Update click
  on the harness server (§5.5).
- **Unverified:** the second-filing failure on a duplicate number and
  revision (§2.4).
- **Unverified:** whether the web UI ever drops one file on two targets
  (§2.3).
- **Not yet read:** the proposal reader's Jev field list (§7).
- **Not yet read:** `data/eval/rfp` and the `web/src` upload request shape.
- **Assumed from memory notes, not re-checked:** that the owner has a
  standing authorisation for live Jev calls. It was given for a different
  plan, so I am asking again in question 1.
