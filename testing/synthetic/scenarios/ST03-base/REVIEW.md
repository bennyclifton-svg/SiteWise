# ST03 review (T03 office-fitout)

Scenarios: ST03-base, ST03-da-pathway, ST03-existing-fitout-strip-out.

## Verdict: FIX

Checker: `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST03-*` gives `ok` for
all three.

What this review covered: all seven brief steps. I diffed each variant's manifest and every expected file against
the base. I re-added every fee table and invoice, divided every GST-inclusive figure by 1.1, checked all eight ABN
checksums (all valid), and read every `title_block` against DISCOVERY.md §4.3. I checked every flag sentence against
`knowledge/profile/flag_readings.yaml` and every scale and determinant sentence against `knowledge/profile/taxonomy.yaml`,
`knowledge/determinants.yaml` and `internal/profile/harvest.go`. I compared the keys with earlier run outcomes in
`testing/synthetic/runs/`.

Two problems would score a correct app wrong or make a variant change more than its trigger (findings 1 and 2).
The rest are small.

## The author's two questions

**A. Fee proposals other than architect, structural, civil and geotechnical: accepted.** Brief 03's list was written
for a house. The T03 shape's `procurement` has no structural, civil or geotechnical engineer. It puts
`consultant.architect` in `not_expected_procurement`, because the interior designer leads the design. An architect
proposal would open a package that the shape says is wrong. A structural, civil or geotechnical proposal would open
packages that only the `internal-stair`, `slab-core-drilling` and `heavy-floor-loads` triggers add. The substitutes
cover every `always` discipline (interior design, BCA, certifier) and the `usually` ones that a proposal can open
(services, access, QS). `consultant.services` is a real id in `data/intake/disciplines.json`. Its alias "Building
services engineer" matches the Meridale firm name. The shape lists it as the alias for mechanical, electrical, fire
and hydraulic. The departure is stated in the base `notes.md`. No change is needed. Brief 03 should say "the
type's core consultants" and not name house disciplines, but that is an edit to the brief, not to these scenarios.

**B. Four keys keyed `not_read`: three are right; the fourth is right in outcome but gives the wrong reason.**

- `hdr.scale.floor_plate_sqm` and `hdr.scale.grade`: right. The office `scale_fields` give neither of them
  `triggers` (`taxonomy.yaml` lines 414 to 428). `harvest.go` lines 121 to 124 skip any scale field with no
  trigger, so they are never asked.
- `proj.regulated_design`: right. `project_details.yaml` line 154 has no `extraction`, so `addDetail` returns early
  (`harvest.go` line 131). Every run in `runs/` leaves it blank (ST01, ST02, ST05, ST12 and ST13 score `missed`, the
  rest `missed_by_design`).
- `det.type_of_construction`: never asked, but it is not unreachable. It is `derived: true` by
  `rule.ncc.type-of-construction`, which looks up `knowledge/tables/type_of_construction.yaml` (Class 5, rise 4 or
  more, gives A). In run 20261107 ST02-base the derivation stays `state: unknown, reason: unreviewed`, because the
  class and rise readings are only `proposed`. So it is blank on a harness run, and `not_read` scores correctly
  (finding 3 fixes the reason). The T02 shape puts this key in `expected_derived`. T03 lists it under `profile`.

## Findings

### 1. All three scenarios, `expected/profile.json`: `hdr.scale.tenancies` 1 cannot be read (serious)

- What is wrong: the key is `{"value": 1, "sources": ["d01"], "read_by_design": true}`, carried by "Avenor will be
  the single tenancy on the floor". That sentence matches the trigger `\btenanc(y|ies)\b`, but candidates come only
  from `tenancyCountRe`, which is `(\d+)\s+(?:separate\s+)?tenancies\b` (`harvest.go` lines 202 and 210). That pattern
  needs a numeral followed by the plural. `add` drops a pre-parsed key that has no candidates ("Jev cannot pick a
  value code did not find", `harvest.go` lines 138 to 142). No English sentence for one tenancy matches, so a
  correct app scores `missed` and not `missed_by_design`.
- Exact change, in the base and both variants:
  `"hdr.scale.tenancies": {"value": 1, "sources": ["d01"], "read_by_design": false, "not_read": "tenancyCountRe (internal/profile/harvest.go) needs '<n> tenancies'; a single tenancy cannot be stated that way, so no candidate is found and the key is not asked"}`.
  Add the key to "Missed by design" in each `notes.md`. Add one line under the base's "Shape comments": the app
  cannot read a count of one tenancy. Leave the d01 sentence as it is. It is the natural wording, and it records
  the gap.

### 2. ST03-da-pathway `manifest.json` d01: the `flag.community` sentence moved without the trigger needing it (moderate)

- What is wrong: the base sentence "No community consultation is required for an internal fit-out" became "No
  community consultation is required for an internal fit-out beyond council's notification of the DA". The flag's
  question is "does the passage state that community consultation is required?" (`flag_readings.yaml` line 81).
  The added clause names a form of consultation that *is* required, so a careful reader could fairly answer yes.
  The key stays `no`. The trigger's `profile` lists only `hdr.cond.planning`, and nothing in the evidence needs
  this sentence changed. The council's statutory notification is not the consultant community-engagement scope the
  flag drives.
- Exact change: restore the base `says` and the base text for `flag.community` in d01. In `changed_from_base`, d01
  `why`, delete "community" from "clause 4.6, panel and community sentences reworded". d20's "council notification
  under its participation plan" fires no trigger and can stay.

### 3. All three scenarios, `expected/profile.json`: `det.type_of_construction` `not_read` gives the wrong reason (minor)

- What is wrong: "has no triggers ... so no passage is asked it" is true, but it hides the real path. The app derives
  the value from `det.ncc_class` and `det.rise_in_storeys` once both are reviewed (see B above).
- Exact change, in all three: `"not_read": "No trigger asks it. The app derives it from det.ncc_class and det.rise_in_storeys (rule.ncc.type-of-construction, Table C2D2: Class 5, rise 24, gives A) only after both inputs are reviewed; on an unreviewed harness run the derivation stays unknown (run 20261107 ST02-base: reason unreviewed)"`.
  Replace the base `notes.md` line "`det.type_of_construction` has no triggers" with the same point. Shape note for
  the owner: consider moving `det.type_of_construction: A` to `expected_derived`, as T02 does.

### 4. All three scenarios, `expected/document_register.json` d03: `kind` accept list too narrow (minor)

- What is wrong: d03 is the landlord's "Base building information and tenant fit-out guide". The key accepts
  `report` or `specification`. A tenant fit-out guide is a brief from the landlord to the tenant, so `design_brief`
  is just as defensible. All three are read kinds. d03 is the only read source for `det.gas_supply` and
  `flag.authority_asset`. A defensible `design_brief` filing should not score as a register error.
- Exact change: `"kind": {"accept": ["report", "specification", "design_brief"]}` for d03, in all three.

### 5. ST03-existing-fitout-strip-out: the notes and the d01 trap credit the wrong sentences with `site.demolition` (minor)

- What is wrong: the strip-out implied pattern is `\bstrip[- ]?out\b|\bdemoli(?:tion|sh)\b`
  (`knowledge/profile/scope_defaults.yaml`). The trigger sentences in d01, d03 and d13 say "stripped out", "strips
  it out" and "demolishes". None of these matches: "stripped" is not "strip out", and `\b` fails inside
  "demolishes". The pattern does match the fee proposals: d05 "demolition plan", d06 "before the strip-out" and d10
  "Strip-out and make-good costs" (all `commercial`, a read kind). So the implied system comes from the proposals.
  Only Jev's work reading of d13's "remove the existing partitions..." can give the action `remove`. The keys do not
  change: the variant's pointers expect the trigger's board whichever document fires it. The notes are what mislead
  a later reader.
- Exact change: in `expected/notes.md`, replace the bullet beginning "d13's strip-out sentence gives
  `site.demolition`" with: "The implied pattern matches 'demolition plan' (d05) and 'strip-out' (d06, d10), not the
  trigger sentences ('stripped out', 'demolishes' do not match). Those give `site.demolition` with the refurb
  default `alter`. Only a works reading of d13's 'remove the existing partitions...' gives `remove`, so
  `fr.strip-out-fitout` may stay false: a likely miss." In the d01 trap `make-good-at-expiry`, change `expect` to
  "The make-good sentence adds nothing; site.demolition comes from this job's strip-out wording (d05, d06, d10 by
  pattern; d13 for the action remove)".

### 6. ST03-da-pathway `manifest.json` d04: DA determination date not realistic (minor, realism only)

- What is wrong: the 24 February email plans for "council needs to determine the DA by June". The SEE (d20) is
  dated 20 May, after GA Rev B (13 May, "Amended for DA lodgement"). A fit-out DA lodged in late May and determined
  within weeks is not believable for North Sydney. d04 is correspondence and not read, so no key moves.
- Exact change: "council needs to determine the DA by the end of July and the certifier to issue the construction
  certificate by late August". This still fits the 7 September access date.

## Checked and correct

- **Shape fidelity (base).** Every shape `profile` key is keyed. Every trigger is denied or absent, and the
  negations follow the shape: "The fit-out is complying development; the certifier will issue a CDC", "handed over
  in base-building condition", "the layout complies with the DTS provisions", "The building was completed/
  constructed in 2008", and no Green Star or NABERS tenancy target. The declared traps are real. "fire stairs"
  matches the stair pattern, "BASIX" the energy-rating pattern, "bathrooms" the wet-room pattern, and "No
  strip-out or demolition" the strip-out pattern. "kitchenette" does not match `\bkitchens?\b`. The tower's DA,
  the NABERS base-building rating, the heritage item next door, "24-storey" and "sprinklered throughout" are
  correctly keyed as not changing the job.
- **Flags.** All 22 `no` flag sentences match their triggers in `flag_readings.yaml`. That includes
  `osd_existing` ("on-site detention is required and no existing"), `excavation_adjoining`
  ("excavation, so no adjoining"), `authority_asset` ("Sydney Water assets") and `panel` (d20 "local planning
  panel").
- **Scale and determinants.** `nla_sqm` ("1,200 m2" with "NLA"), `storeys` ("24-storey", "24 storeys"),
  `existing_building_year` ("constructed in 2008", which matches the `(?:built|constructed) in \d{4}` trigger),
  `climate_zone`, `gas_supply` and `ncc_class` are stated in read kinds, in wording that earlier runs read
  correctly (ST09, ST02, ST10).
- **Register.** The label-line title blocks give number, revision, title and date, using `Ref`, `Proposal no`,
  `Quote no`, `Report no`, `Drawing number` and `Revision`. Each `# Heading` is identical to its `Title:`. The d04
  email title is the first Subject ("RE: ..."), and its date is the latest header (24 February). The d02 and d06
  tables give no number or date, and the notes declare it. d15 supersedes d14 is declared unlinkable today (every
  supersedes key in run 20261107 scores `missed`). Invoices are kept out of the register.
- **Fees.** Every stage table adds to its stated total. d06's figures are GST-inclusive: 6,600, 19,800, 26,400,
  3,300 and 12,100, each divided by 1.1, give 6,000, 18,000, 24,000, 3,000 and 11,000, which sum to $62,000 ex
  GST, with `gst_inc_normalised` (`read.go` `FlagGSTInc`). Options and the hourly rate sit outside the totals. d19
  (DA variant) adds 2,800 + 7,200 + 2,000 = $12,000. d07 is dropped on `row:consultant.bca`, a row the commercial
  refurb baseline suggests.
- **Invoices.** d16 and d18 add up at 10%. d17 is $6,000 + $600, printed as $6,900: `arithmetic` only (the GST is
  exactly 10%, so no `gst_rate`). No state flags are listed.
- **Variants.** da-pathway changes only `hdr.cond.planning` (`cdc` to `da`), and its evidence is the shape's own
  phrasing. Every read document that named the CDC now names the DA or the CC. d02's land-eligibility sentence is
  kept as a trap, and R12 majority settling still gives `da`. The town planner proposal and the SEE are added, with
  new ids. Apart from finding 2, nothing else moves. existing-fitout-strip-out has no profile keys and changes none.
  It replaces both base negations and the d13 `no-strip-out` trap, and adds the demolition plan drawing. The notes
  declare that "kitchen joinery" in d13 fires the kitchen pattern. That is harmless, because joinery, sanitary,
  water and local exhaust are all in the base scope.
- **Realism.** An A-grade North Sydney floor fitted out under a CDC (Commercial and Industrial Alterations Code) by
  an interior designer and a multi-services engineer is the standard Sydney pattern. So is the landlord's fit-out
  guide standing in for a survey, and a QS and access consultant. Zone E2 Commercial Centre is current. The budget
  is $2,000/m2 for a high-quality fit-out. The fees are interior design 5.4%, services 2.6%, certifier $17,000,
  BCA $10,000, QS $16,500 and access $5,500. All are plausible at this scale. The calendar runs from the brief in
  February, through proposals in March, reports in April and May and invoices in April and May, to access on
  7 September. That is coherent.
