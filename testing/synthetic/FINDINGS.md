# Findings

From pilot runs. Each entry: what was expected, what the app did, the
suspected cause, and where to look. A key is changed only when it was
wrong, with the reason in that scenario's `notes.md`. "Suspected" means not
yet traced in the code.

Run `runs/20261031-pilot` (T01 base, CDC and trees variants; app at
`e233aa6` plus the owner's uncommitted working tree), 31 October 2026.

## App defects

**F1. Update profile response cut off while the app is busy.** *(2 Nov: improved by an owner fix, still 3 of 12 waves, only the first Update on a new project; see `runs/20261102-fixes-1-4-6`.)* In 8 of 12
waves, `POST /projects/{id}/profile/read` returned 200 and then the
connection was reset partway through the body (about 217 KB). The read was
still queued, because that happens before the response is written. Idle,
the same call succeeds every time. No panic or error appears in the server
log. Suspected: the profile view is assembled while a profile event
rewrites it, and something fails after the headers are sent
(`internal/httpapi/profile.go` `writeProfileQueued`; `writeJSON` drops the
encode error). A user would see the Update button fail intermittently.

**F2. Council area read with a trailing word.** `proj.lga` came back as a
conflict between "City of Ryde Section" and "City of Ryde Duty" (base,
trees), and as the single wrong value "City of Ryde Section" (CDC). The
planning certificate says "City of Ryde" and then a section heading or
"Duty". Suspected: the LGA reading takes the words after the name.
`knowledge/profile/project_details.yaml` (lga), `internal/profile/details.go`.

**F3. GST-inclusive fee proposal kept inc GST.** The civil fee proposal
(d07) states $9,900 inc GST. The app's stated total is $9,900; the key is
$9,000 ex GST, with `gst_inc_normalised` expected. All three scenarios.
`internal/proposals/read.go`.

**F4. Procurement route not read.** *(2 Nov: cause found; still open.)*
"Architect-led traditional lump-sum tender to three or four residential
builders" in the owner's brief leaves `hdr.cond.procurement_route` blank.
Knock-on: the program uses `sk.generic` instead of `sk.new-traditional-da`,
in all three scenarios, which accounts for much of the program's missed and
out-of-order activity.

*Cause.* The route question is never asked for that passage. Header
questions (class, type, work type, every `hdr.cond.*`) go only to passages
that `HeaderRouted` accepts (`internal/profile/questions.go`): ordinal ≤ 12,
a section matching `headerSection` (description, outline, scope,
introduction, project details, contract…), or a "the project comprises…"
sentence. The brief's sentence is passage 15 under "## How we want to
build", so it fails all three. Checked with `cmd/profile-eval -live`: the
same text under "Contract and procurement" gets the route question, and
Jev answers `traditional` with confidence 1.0. Under its real heading, the
question is not in the request.

*Suggested fix (owner's call, lane A).* Route the procurement-route
question on its own cue words, not only on position: a passage matching
words like tender, lump sum, design and construct, D&C, ECI, managing
contractor or novat(ed/ion) gets `hdr.cond.procurement_route`, wherever it
sits (the intent-routing pattern the code already cites). Adding
"procurement|tender" to `headerSection` alone would not catch this brief,
whose heading has neither word.

**F5. Invoice with an arithmetic error flagged as a GST-rate error.** d15
states ex $3,800, GST $380 and a total of $4,280 (it should be $4,180).
The app read ex GST as $3,900 (total minus GST) and raised `gst_rate`, not
`arithmetic`. In the trees run the same invoice raised `amount_unknown`
and left the amounts blank. Run-to-run variation on the same bytes.
`internal/invoices/ledger.go` `Check`, `internal/invoices/read.go`.

**F6. The BASIX assessor is not recognised as a discipline.** *(2 Nov: fixed; verified in all three scenarios.)* The
BASIX/NatHERS fee proposal (d09) gets no discipline (Jev 0.59, blank). So
no BASIX package is created from it, and it misses both procurement and
the cost plan. The knowledge has `consultant.basix`, but the filing
discipline question does not pick it.

**F7. The planning certificate is filed as a town planner's document.** d02,
a council s10.7 certificate, gets discipline `consultant.town_planner`
(amber 0.73). The key is `none`: council is not a consultant.

**F8. The supplier's name is never read from an invoice.** All nine
invoices (three per scenario) have a blank `supplier`: Jev answers none at
confidence 1.0, though each invoice names the firm in its heading and
beside its ABN. So no invoice links to its package, even where a package
for that firm exists from its fee proposal. `internal/invoices/read.go`
(supplier question). Found 2 Nov after the scorer stopped counting state
flags as false alarms.

**F9. The certifier RFP has no CDC pathway.** On the CDC variant, every
principal-certifier stage (B stages) is excluded and the RFP blocks on
`pathway_unsupported`: `knowledge/rfp/bca-certification.yaml` lists cdc in
`unsupported_pathways`, and the B stages are gated to da and ssd. A CDC
house needs exactly this appointment. (RFP triage A5.)

**F10. The certifier RFP leaves out the Class 1 and 10 critical-stage
inspections and the BASIX check at the OC** (all scenarios; owner ruling
R7). Related: a certifier package still opens as BCA advisory (a1 to a3),
because `bca_certification` is a client scope decision. Whether a house
should default to certification is an owner call. (RFP triage A1.)

**F11. `flag.mixed_use` is yes for a house with a garage.** Class 1a plus
10a counts as "parts with different classes" (`internal/rfp/drivers.go`
around line 304), so the architect RFP of an ordinary house asks for a
"Class and use map by part". Class 10 alongside one other class should
not make a project mixed use. (RFP triage A3.)

**F12. The structural RFP has no row for footings that avoid a tree's
structural root zone** (trees variant, owner ruling R9). (RFP triage A4.)

## Known gaps confirmed (already named in the shape)

- Electrical, hydraulic and mechanical consultants are suggested for a
  house (`package_defaults.yaml` baselines[0], scope_additions[2]), with
  matching cost lines plus "Active fire protection".
- No BASIX, landscape or QS package, no head contract (needs accepted work
  items). Missing cost lines: long service levy, HBCF insurance, s138 road
  fees, preliminaries and margin, PC and provisional sums, owner supply,
  balustrades.
- CDC: the town planner, "DA fees and contributions" and "Construction
  certificate and certifier fees" stay in.
- Trees: no "Tree protection" cost line.
- Supersession is never written (A-101 Rev A → Rev B), as DISCOVERY.md
  §4.5 predicted. Lifecycle is mostly blank, as Jev lifecycle answers are
  never applied.
- Title blocks laid out as a table (the messy tier) give no number or date,
  by design.

## Key or harness issues (not app defects)

- **Invoice flags `no_supplier`, `unmapped` and `period`.** *(Fixed 2
  Nov, in the scorer.)* They follow from the project's state, not the
  document: nothing is allocated or confirmed in a harness run, and the
  upload month is after the invoice date. `harness/score.py` `state_flags`
  now expects them, and brief 03 says keys list only document-caused
  flags. No key changed.
- **Conflicts on site area (650 vs 650.2 m²), GFA (300 vs 298 m²), access
  and authority assets.** The documents genuinely disagree, by design in
  some cases. The app shows them as conflicts that include the true value,
  which is acceptable (Provisional Q4).
- **Missed "no" flags** (`sustainability_target`, `groundwater`, `staged`,
  `multi_building`), `det.existing_building`, `det.gas_supply`,
  `fact.contract_form` and `proj.regulated_design`: to be checked whether
  the documents' wording matches the app's reading triggers before calling
  them app misses.
- **RFPs.** *(Triaged 2 Nov: `runs/20261102-fixes-1-4-6/RFP_TRIAGE.md`.)*
  The ~100 "new failures" per scenario were mostly not app defects. The
  scorer now reads a position the app marks "not required by the current
  brief" as absent (18 per scenario), and reports blocks that wait on user
  input (allowances, issue fields, terms, client scope decisions) as
  "awaiting user input", not failures (about 50). The T01 shape's phrases are
  now keyed to template rows where one exists, and its structural
  inspection schedule is I, not O. Base new failures: 105 to 34, all
  knock-ons of blank profile values (14 from F4). The real RFP defects are
  F9 to F12.

## What worked

- Kind and title: 100% across 43 filed documents. Status 100%.
- The trigger values each variant changes were all read correctly:
  planning `da` and `cdc`, both tree flags `no`/`yes`, and site class M.
- Profile: 44 or 45 values correct out of 60 per scenario.
- Invoices: number, total and GST read correctly on every clean invoice.
- The arborist package was proposed in the trees variant.
