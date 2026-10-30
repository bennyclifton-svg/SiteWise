# Brief 03: Scenarios and answer keys for one project type

To dispatch, replace `{TYPE_ID}`, `{SLUG}`, `{NAME}` and `{VARIANTS}` (the
trigger ids to build as variants, from the type's `variants` list). The
orchestrator picks them; the pilot builds the base plus two. Everything else
stays as written.

---

You are a senior Australian construction project manager. You are turning
the owner-approved domain map for **{TYPE_ID} {NAME}** into test scenarios:
one fictional project (the base) and one-change variants of it. For each
you write the **document list** (`manifest.json`) and the **answer keys**
(`expected/`). You do **not** write the documents themselves; a later stage
renders them from your manifest. So the manifest must say precisely what
each document states and how it states it.

Repository: `D:\AI Projects\sitewise`. Write only inside
`testing/synthetic/scenarios/S{TYPE_ID}-base/` and
`testing/synthetic/scenarios/S{TYPE_ID}-<trigger id>/` for each of
{VARIANTS}.

## Read first

1. `testing/synthetic/types/{TYPE_ID}-{SLUG}/shape.yaml` and `DOMAIN.md`.
   The shape is the truth. Your scenario is one concrete instance of it.
2. `testing/synthetic/DISCOVERY.md` §2 to §9: how the app files, reads the
   profile, reads fees and reads invoices. §4.3 decides what a Markdown
   title block must look like for the app to read it.
3. `testing/synthetic/reference/OWNER_RULINGS.md` and `CONVENTIONS.md`.
4. `docs/plans/2026-10-28-synthetic-test-corpus.md`, "Provisional
   decisions". They settle bands, conflicts, tiers, drop targets and
   unsupported fields.
5. `data/intake/kinds.json`, `disciplines.json`, `lifecycle.json` and
   `data/profile/reading.json`.

## Rules

- **Do not change application code, knowledge files or the shape.** If the
  shape looks wrong, say so in `notes.md` and in your reply.
- **Everything is invented.** People, firms, street addresses, lot/DP
  numbers and ABNs. Use a real NSW LGA and suburb so the profile has a
  real place, but an invented street name and number. Emails use
  `example.com`. List every invented lot, ABN, firm and email under
  `invented` in the base manifest. An ABN you list must pass the ABN
  checksum unless a trap needs it to fail; then name the trap.
- **Money is integer cents, ex GST,** in keys. A document may state an
  amount inc GST; the key holds the ex-GST figure and the document's
  `carries` says how it was stated.
- **No rates or prices from the repo's knowledge.** You choose plausible
  fees and the budget yourself; they are scenario facts, not app outputs.
- **Dates** are ISO (`2026-03-14`) in keys. Choose a project calendar that
  runs from brief to the first consultant invoices over a few months.

## The base scenario

The most common version of the job, exactly as the shape's `profile`
describes it, with **every trigger denied or absent**.

- **Size:** 12 to 20 documents in 3 or 4 waves. Wave 1: what the owner has
  before design (brief, planning certificate, survey). Wave 2: fee
  proposals. Wave 3: design reports and drawings, including one drawing
  issued at two revisions. Wave 4: consultant invoices.
- **Every base value must be stated somewhere the app reads.** For each key
  in the shape's `profile` and `scenario_inputs`, at least one document
  whose expected kind is in `reading.json` `read_kinds` must state it, or
  deny it for a `"no"` flag. Drawings, emails and photos are not read. Where
  the only natural home is an email or a drawing, keep it there, mark the
  value `read_by_design: false`, and expect the app to miss it.
- **Tiers** (Provisional decision Q5): most documents `clean` (readable
  `Label: value` lines before the first `##`). At least two `messy` (title
  block as a Markdown table). At least two `adversarial`, each with a
  declared trap from the shape's triggers' `negation` lists or from
  `DOMAIN.md` §10 keyword traps.
- **Drop targets** (Q14): fee proposals to `repository`, with one dropped
  on its procurement row (`row:<discipline id>`); invoices to `invoices`;
  everything else to `repository`.
- **Fee proposals** for at least the architect, structural, civil and
  geotechnical engineers, and one more consultant. Each has a stage-by-stage
  fee table. At least one states GST-inclusive figures. Choose firms whose
  names do not contain "Studio" (Jev reads that as no discipline).
- **Invoices:** 2 to 4. One clean, one with a declared defect the app has a
  flag for (`arithmetic`, `gst_rate`, `abn_invalid` or `duplicate`, see
  DISCOVERY.md §8).

## Variants

One folder per trigger in {VARIANTS}. A variant is the base with **exactly
that trigger true**. Copy the base and change only what the trigger needs:

- the trigger's `evidence` replaces its `negation` in the documents that
  carried it;
- documents the trigger adds (for example an arborist fee proposal and
  report) are new, with new ids;
- documents the trigger `removes` (for example a statement of environmental
  effects under a CDC) are dropped.

List every changed, added or removed document under `changed_from_base` in
the variant's manifest, with the reason. Profile values may differ from the
base only on the trigger's `profile` keys. The checker enforces both.

## Files per scenario

### `manifest.json`

```json
{
  "schema": "sitewise-synthetic-manifest/1",
  "scenario": "ST01-base",
  "type_id": "T01",
  "base": null,
  "triggers": [],
  "project": {"name": "...", "address": "...", "lga": "...", "summary": "two sentences"},
  "invented": {"lots": ["Lot 7 DP 1288841"], "abns": ["..."], "firms": ["..."], "emails": ["..."]},
  "documents": [
    {
      "id": "d01",
      "filename": "Owner brief - 14 Example Rise.md",
      "title": "Owner's brief",
      "author_role": "owner",
      "firm": null,
      "date": "2026-02-03",
      "wave": 1,
      "drop": "repository",
      "tier": "clean",
      "title_block": "label lines: Title, Date",
      "carries": [
        {"key": "proj.budget_stated", "says": "Construction budget: $1,450,000 excluding GST"},
        {"key": "flag.basement", "says": "Single slab on ground; no basement"}
      ],
      "traps": [],
      "content": "What the document is and what its sections say, in 2 to 6 sentences."
    }
  ],
  "changed_from_base": []
}
```

- `id`: `d01`, `d02`, … unique. A variant keeps base ids and adds new ones.
- `filename` ends `.md`. Filenames are realistic and never contain
  "test", "expected" or a key name.
- `author_role`: a discipline id from `disciplines.json`, or `owner`,
  `council`, `authority`, `builder`.
- `drop`: `repository`, `invoices` or `row:<discipline id>`.
- `tier`: `clean`, `messy` or `adversarial`.
- `carries[].key` is a profile key, a fee or invoice field, or
  `register.<field>`. `says` is the wording the document uses.
- `traps[]`: `{"id": "...", "what": "...", "expect": "what a correct app does"}`.
- `changed_from_base[]` (variants only): `{"doc": "d07", "change":
  "changed|added|removed", "why": "..."}`.

### `expected/document_register.json`

One entry per document dropped on `repository` or a row. Invoices dropped
on `invoices` are not in the register.

```json
{
  "schema": "sitewise-synthetic-register/1",
  "documents": [
    {"doc": "d01", "status": "filed", "kind": "design_brief", "discipline": "none",
     "lifecycle": "brief", "number": null, "revision": null,
     "title": "Owner's brief", "date": "2026-02-03", "supersedes": null,
     "unsupported": {"issue_status": "for information", "originating_org": "owner"}}
  ]
}
```

A field may be a value, `null` (the app should leave it blank) or
`{"accept": [..]}` for several right answers. `supersedes` is a doc id.
Follow DISCOVERY.md §4.2 for kinds and Provisional decisions Q13 for
disciplines of parties with no catalogue entry.

### `expected/profile.json`

```json
{
  "schema": "sitewise-synthetic-profile/1",
  "values": {
    "hdr.work_type": {"value": "new", "sources": ["d01"], "read_by_design": true},
    "flag.basement": {"value": "no", "sources": ["d01", "d06"], "read_by_design": true},
    "proj.budget_stated": {"value": 145000000, "sources": ["d01"], "read_by_design": true}
  },
  "conflicts": [
    {"key": "...", "values": ["...", "..."], "sources": ["d03", "d09"], "true_value": "...", "why": "..."}
  ],
  "derived": [{"key": "det.basix_required", "value": true}]
}
```

- Keys: every key in the shape's `profile` and `scenario_inputs`, plus any
  other value your documents state.
- `value` uses the shape's value form, or `{"accept": [..]}`.
- `sources`: the doc ids that state it. `read_by_design` is true when at
  least one source has a read kind.
- `conflicts`: only where your documents genuinely disagree (Q4). The key
  holds the true value. Use at most one in the base.
- `derived`: copy the shape's `expected_derived`.

### `expected/procurement.json`

```json
{
  "schema": "sitewise-synthetic-procurement/1",
  "shape": "types/T01-new-house/shape.yaml",
  "triggers": [],
  "from_proposals": [{"doc": "d05", "discipline": "consultant.structural", "kind": "services"}],
  "route": "traditional",
  "novation": {}
}
```

The shape and triggers give the board the scorer expects. `from_proposals`
lists the packages your fee proposals should create.

### `expected/fees.json`

One entry per fee proposal or quote.

```json
{
  "schema": "sitewise-synthetic-fees/1",
  "submissions": [
    {"doc": "d05", "kind": "fee_proposal", "discipline": "consultant.structural",
     "firm": {"name": "...", "abn": "...", "contact": "...", "email": "...@example.com", "phone": "...", "address": "..."},
     "stated_total_cents": 1850000, "gst_basis": "ex", "flags": [],
     "prices": [
       {"label": "Concept and DA structural advice", "kind": "stage", "state": "priced", "amount_cents": 350000}
     ]}
  ]
}
```

`gst_basis`: `ex`, `inc` or `unknown`. `kind` of a price row: `stage`,
`element`, `option`, `reimbursable` or `rate`. `state`: `priced`,
`not_priced`, `included` or `excluded`. When the document states GST-inc
figures, `gst_basis` is `inc` and the amounts are the ex-GST values, with
`gst_inc_normalised` in `flags`. The sum of priced `stage` rows equals
`stated_total_cents` unless a trap says otherwise (then flag `total_gap`).

### `expected/invoices.json`

```json
{
  "schema": "sitewise-synthetic-invoices/1",
  "invoices": [
    {"doc": "d16", "kind": "invoice", "number": "INV-1042", "invoice_date": "2026-05-29",
     "issuer_abn": "...", "supplier_discipline": "consultant.architect",
     "ex_gst_cents": 600000, "gst_cents": 60000, "total_cents": 660000,
     "flags": [], "unsupported": {"claim_stage": "Concept design", "retention": null}}
  ]
}
```

`ex_gst_cents + gst_cents = total_cents` unless the invoice's trap is an
arithmetic error, which must then appear in `flags` as `arithmetic`. Flags
come from DISCOVERY.md §8. Put anything the app has no field for under
`unsupported`.

### `expected/rfis.json`

`{"schema": "sitewise-synthetic-rfis/1", "supported": false, "rfis": []}`.
List RFI documents if you wrote any.

### `expected/cost_plan.json`, `program.json`, `rfps.json`

These come from the shape, so they only point to it:

```json
{"schema": "sitewise-synthetic-from-shape/1", "shape": "types/T01-new-house/shape.yaml", "triggers": []}
```

Add `"scenario_facts"` only where your documents fix something the scorer
should check beyond the shape, for example
`{"program": {"target_construction_start": "2026-11-02"}}`.

### `expected/notes.md`

Short. The project in two sentences; the traps and what each tests; values
the app will miss by design and why; anything in the shape you think is
wrong.

## Check

```
python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/S{TYPE_ID}-*
```

Fix every error. Do not change the checker.

## Reply

**150 words or fewer:** the checker result; per scenario the document
count, waves and traps; the values the app will miss by design; anything
you think is wrong in the shape. Do not paste the files.
