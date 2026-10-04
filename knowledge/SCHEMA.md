# Knowledge schema

`knowledge/` is SiteWise's compiled model of the building. It is data, not
code. The Go app loads it at startup. Jev questions in it run against project
passages; everything else is evaluated by code.

Five layers (see `docs/design/2026-09-29-foundation-design.md` §2):
systems, determinants, rules, interfaces, failure modes.

## Layout

```text
knowledge/
  SCHEMA.md                 this file
  systems.yaml              top-level systems (lead-owned; do not edit in a cluster)
  determinants.yaml         project determinants (owned by the determinants cluster)
  tables/<table_id>.yaml    lookup tables used by rule `derives` (determinants cluster)
  clusters/<cluster>/
    systems.yaml            child systems under the cluster's top-level systems
    rules.yaml
    interfaces.yaml
    failure_modes.yaml
    proposed_determinants.yaml   determinants this cluster needs that are not in determinants.yaml
    REPORT.md               coverage, gaps, contradictions, cross-cluster edges
```

Every file is a YAML mapping with `version: 1` and one list key (`systems`,
`rules`, `interfaces`, `failure_modes`, `determinants`). Run
`python tools/check_knowledge.py` before finishing; it must pass.

## IDs

| Kind | Form | Example |
|---|---|---|
| System | dotted path from a top-level id, lowercase, hyphens inside segments | `fire-active.sprinklers` |
| Determinant | snake_case | `rise_in_storeys` |
| Rule | `rule.<instrument>.<slug>` | `rule.ncc.type-of-construction` |
| Interface | `if.<slug>` | `if.services-penetrate-fire-rated-construction` |
| Failure mode | `fm.<slug>` | `fm.plastic-pipe-without-fire-collar` |
| Table | snake_case | `type_of_construction` |

IDs are permanent once merged. Never reuse one for a different meaning.

## Verified tables and pending derivations

Tables use a separate mapping: `version`, `id` (matching the filename),
`status`, `verified: true`, `instrument`, `clause`, `primary_source` (HTTPS),
`checked_on`, `inputs`, `outputs`, `scope`, `exclusions`, and `rows`.
Every row supplies all outputs. Table inputs reference determinants. A verified
transcription remains `status: draft` until the owner reviews it. Verification
does not implement exceptions or establish applicability to a project.

A rule awaiting an authoritative table keeps its `derives` contract with
`pending: true` and a `reason`. Code must return unknown for that derivation;
it must never execute seed numbers or substitute a default. Active derivations
must reference an existing verified table. The strict checker enforces this.
A `derives.gives` target must be a derived determinant whose `by` names that
rule. Keep computed requirements distinct from extracted design/provided values.

Verified rule clauses and numeric claims include `primary_source` next to their
verification flag. A verified clause locator does not verify every claim in
the rule. Corrections retain the rejected seed claim in notes or the verification
report rather than marking a contradictory claim verified.

## Determinant evidence contract

An extracted fact belongs to a building, part, storey, compartment or element;
retain that scope with its document and passage provenance. Never merge facts
from different physical scopes into one project-wide scalar. Missing evidence
and contradictory evidence both remain unresolved.

For boolean determinants, use Choice with `stated_true`, `stated_false` and
`not_stated`; code maps only the first two to booleans. Silence is not false.
For `multi_choice`, compile one independent option-presence question per option
in the same fan-out, using the question as a wording template; one Choice cannot
return a set. An option not mentioned remains unknown, not explicitly excluded.
For `pre_parsed`, include the candidates for that determinant in state; code
builds Choice criteria from those candidates plus `none`, and copies the selected
source value. The empty criteria maps in these templates are not API requests.
Scope, units, conflicts, thresholds and option generation are code responsibilities.
See [pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[primitives](https://docs.typesafe.ai/primitives), and
[fan-out](https://docs.typesafe.ai/patterns/fan-out).

## Common fields

- `sources` (required): list of `{seed: <file in clerk/data/seed>, anchor: "<exact heading line, including the #s>"}`. The checker verifies the anchor exists.
- `status` (required): `draft` for everything written from the seeds. Only the owner promotes to `reviewed`.
- `notes` (optional): short, for reviewers. Never read at runtime.

YAML trap: never use `on`, `off`, `yes`, `no`, `y` or `n` as keys or unquoted
option ids. YAML 1.1 reads them as booleans. Quote option ids that look like
numbers or booleans (`"5"`, `"1a"`).

## Systems

```yaml
- id: fire-active.sprinklers
  parent: fire-active
  label: Automatic fire sprinklers
  describes: >-            # Jev criterion text: what a passage about this system talks about
    Sprinkler heads, pipework, control and alarm valves, and sprinkler water
    supply design (wet, dry, pre-action or deluge).
  excludes: >-             # boundary cases: similar things that are NOT this system
    Fire hydrants, hose reels and domestic water supply.
  status: draft
  sources: [...]
```

`describes` and `excludes` are read by Jev when it labels passages, so write
them literally and include the boundary cases. Keep a parent's children
mutually exclusive and at most 40 per parent.

## Determinants

```yaml
- id: rise_in_storeys
  label: Rise in storeys (NCC definition)
  value: integer            # integer | number | boolean | choice | multi_choice
  unit: storeys             # for numbers
  options: [...]            # for choice: [{id: "5", describes: "..."}]
  derived: false            # true = computed by code from other determinants (then `by: <rule id>`)
  extraction: pre_parsed    # pre_parsed = code finds candidate values, Jev picks one or none
                            # choice = Jev chooses among options directly
  question:                 # see Questions; omitted when derived
    ...
  status: draft
  sources: [...]
```

## Rules

```yaml
- id: rule.ncc.type-of-construction
  title: Type of Construction required
  instrument: NCC 2022 Volume One      # or "AS 2118.1:2017", "EP&A Regulation 2021 (NSW)"
  clause: "C2D2, Table C2D2"
  clause_verified: false               # set true only after checking the instrument's own text
  systems: [structure, fire-passive]
  applies_when: <predicate>            # optional; omitted = always
  derives:                             # optional; a code lookup, never Jev
    table: type_of_construction
    inputs: [ncc_class, rise_in_storeys]
    gives: type_of_construction        # a derived determinant
  requires: >-                         # one plain statement of the requirement
    ...
  evidence: <question>                 # optional; Jev checks whether a passage addresses it
  numbers:                             # every numeric claim, copied from the seed
    - {claim: "Type A for Class 5 with a rise in storeys of 4 or more", verified: false}
  related_rules: [rule id, ...]
  status: draft
  sources: [...]
```

## Interfaces

```yaml
- id: if.services-penetrate-fire-rated-construction
  type: penetrates   # penetrates | sequences | loads | supplies | controls | depends_on | shares_space | boundary
  from: [hydraulic, mechanical, electrical, comms-security]   # systems (any level)
  to: [fire-passive.rated-elements]
  applies_when: <predicate>
  summary: >-
    ...
  governed_by: [rule ids]
  hold_point:                 # only for `sequences`
    before: <system or activity>
    after: <system or activity>
    evidence: what proves the hold point was released
  responsibility: >-          # optional: who designs / installs / certifies each side
    ...
  resolved_when: [<question>, ...]   # Jev nouls; each true answer is evidence the interface is addressed
  failure_modes: [fm ids]
  cross_cluster: true         # true when `from` and `to` sit in different clusters
  status: draft
  sources: [...]
```

Direction: `from` acts on `to` (services penetrate construction; fire water
supplies sprinklers; detection controls fans; sprinkler concession: FRLs
depend_on sprinklers).

## Failure modes

```yaml
- id: fm.plastic-pipe-without-fire-collar
  attaches_to: {interface: if.services-penetrate-fire-rated-construction}   # or {rule: ...} or {system: ...}
  severity: life-safety      # life-safety | compliance | durability | cost | programme
  detector: <question>       # a noul
  flag_when: true            # the detector answer that means the failure is present
  status: draft
  sources: [...]
```

## Predicates (`applies_when`)

Evaluated by code over determinants and present systems:

```yaml
applies_when:
  all:
    - {det: ncc_class, any_of: ["5", "6", "7a", "7b", "8", "9a", "9b"]}
    - {det: rise_in_storeys, gte: 4}
    - {system_present: fire-active.sprinklers}
  any: [...]
  not: {det: sprinklered, is: true}
```

Operators: `any_of`, `eq`, `is` (boolean), `gt`, `gte`, `lt`, `lte`,
`system_present`. Nest `all`, `any`, `not`.

## Questions (Jev)

A question mirrors the TypeSafe API
([primitives](https://docs.typesafe.ai/primitives)) plus routing fields:

```yaml
question:
  type: noul                  # noul | choice | score
  instructions: >-
    Using `text`, does this passage specify how service penetrations through
    fire-resisting walls or floors are protected?
  criteria:                   # noul: {true, false}; choice: {option_id: description}; score: [levels, 2-10]
    true: >-
      It names a protection method for penetrations (fire collar, wrap,
      sealant, damper or a tested system) or requires one.
    false: >-
      It does not address penetration protection, or only mentions fire
      rating of the wall without saying how penetrations are protected.
  runs_on: [fire-passive.rated-elements, hydraulic, mechanical]   # only passages labelled with these systems
```

State for passage questions is always:

```json
{"document": {"kind": "...", "discipline": "...", "title": "..."},
 "section": "<heading path>", "text": "<passage>"}
```

### Authoring rules (from TypeSafe's docs; the checker cannot enforce these)

1. **Atomic.** One judgement per question. Split "A and B" into two questions
   and combine in code ([how to build](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)).
2. **Literal.** Jev answers the written question, not the intent. Name the
   state field in backticks, state exact conditions, and put boundary cases in
   the criteria ([jaggedness §1](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
3. **No maths, counting, dates or unit conversion.** Those go to code. Ask
   whether a value is *stated*, never whether it is *enough*
   ([jaggedness §2, §3](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
4. **Positive phrasing.** `true` means the thing is present. No double
   negatives ([jaggedness §4](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
5. **Criteria agree with instructions** ([jaggedness §7](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
6. **Never ask Jev to write text** ([jaggedness §9](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
7. **Narrow `runs_on`.** Every question runs only on passages labelled with
   its systems. Speed and accuracy both depend on it
   ([jaggedness §5](https://docs.typesafe.ai/model-jaggedness/jev-1.13)).
8. **No thresholds in questions.** Code applies thresholds per question.
9. **Ask what a document can show.** "Does this passage state / specify / show
   X?" A document cannot prove a building complies.

## Project profile additions (2026-10-03)

See `docs/design/2026-10-03-project-profile.md`.

**Deprecation.** `status` may also be `deprecated`, with `replaced_by: <id of
the same kind>`. The replacement must exist and must not itself be
deprecated. Nothing may refer to a deprecated id (`systems`, `runs_on`,
`from`, `to`, `attaches_to`, `parent`, predicates, derivations). Deprecated
systems are never offered to Jev. IDs stay permanent.

**More source forms.** Besides `{seed, anchor}`:

```yaml
sources:
  - {document: hale-brief, anchor: "Recessed Docks"}           # id from data/eval/profile/manifest.json
  - {clerk_file: data/taxonomy/building-classes.json}         # Clerk data copied as data
```

Use `document` only when no seed heading covers the record. The anchor is
recorded but cannot be checked, because the corpus is private.

**Determinant routing and display.**

```yaml
- id: bal
  triggers:                      # RE2 patterns, matched case-insensitively by code
    - '\bBAL[- ]?(LOW|12\.5|19|29|40|FZ)\b'   # the question is asked for a passage only
    - 'bush ?fire attack level'               # when a trigger matches it
  stated_in:                     # where it is usually stated; ids from data/intake
    - {kind: report, discipline: consultant.bushfire, label: Bushfire assessment report}
  profile_group: site            # classification | site | services | fire
  systems: [envelope, site]      # optional: systems that make it relevant to a project's scope
```

A determinant with `triggers` may omit `question.runs_on`. Triggers must be
narrow: `rise_in_storeys` triggers on "rise in storeys", never on "5 storey".

**Profile questions** (built by code from these templates; same authoring
rules as above):

| Id | Options | Meaning |
|---|---|---|
| `det.<id>.assertion` | `stated`, `required`, `allowance`, `not_stated` | How the passage presents the value. Only `stated` (and user) values feed derivations. |
| `sys.<leaf>.presence` | `included`, `not_included`, `not_stated` | Whether the completed project will have the system. "Not applicable" is `not_included`; silence is `not_stated`. |
| `sys.<leaf>.provider` | `contractor`, `owner`, `others`, `not_stated` | Who provides it. |

The exact wording is in `docs/plans/2026-10-03-project-profile.md` §2.2.

**Profile files** in `knowledge/profile/`, each with `version: 1`:

- `taxonomy.yaml`: `status`, `sources`, `building_classes` (with
  `subclasses`, each with `ncc_class` and `scale_fields`: `key`, `label`,
  `type`, optional `unit`, `basis_required`, `triggers`), `work_types`, and
  `conditions` (`key`, `label`, `options`). A work type may add
  `display_label`, shown to people; `label` stays the wording Jev reads, so a
  screen rename never changes a recorded question. Copied from Clerk's taxonomy as
  data, without cost-uplift text. Site conditions that determinants already
  cover (bushfire, flood, heritage, contamination, water source) are left out.
- `project_facts.yaml`: `facts:` list, with the same shape as determinants
  (consent number and dates, contract form and basis, defects period, design
  life). Ids must not collide with determinants.
- `scope_defaults.yaml`: `status`, `sources`, `always_shown` (profile
  determinants every scope shows), `empty_work_types` (work types that start
  with nothing in scope), `presets` (`id`, `label`, `systems`), `classes`
  (`{class, work_type, systems}`, where class is a taxonomy subclass id) and
  `categories` (`{category, systems}`). New build and extension use the
  class's list for the work type, else its new-build list, else its
  category's list. Systems are live leaf ids. Defaults are suggestions,
  never evidence.

**Scope and relevance** (`docs/design/2026-10-03-profile-scope.md`). A
project's scope is its defaults, plus systems a read document includes,
plus the user's choices, which are final. Code then decides which profile
determinants are relevant: those read by a rule (its `applies_when` and
`derives`) whose `systems` touch the scope and whose `applies_when` is not
false for the values the profile holds (an unknown value keeps the rule),
those whose own `systems` touch the scope, and `always_shown`. The checker
warns about a profile determinant that can never become relevant: read by
no rule, no `systems`, not always shown. Fix a profile that shows too much
or too little here, in the schema, not in code.

## Works layer (2026-10-04)

See `docs/plans/2026-10-04-next-wave-architecture-schema.md` ("Works model",
"Building logic for works"). The building layers say what exists and what
rules apply; the works layer says what a kind of work does to a system and
what that raises. Evaluated by code during the profile rebuild; the only Jev
parts are the signal questions. Everything here is `status: draft` until the
owner reviews it. Nothing in this layer states a clause number or a numeric
legal claim; a regulatory trigger carries `clause_verified: false` until the
instrument itself is read.

```text
knowledge/works/
  actions.yaml                 the eight actions, work-type defaults, existing conditions
  interface_consequences.yaml  what work on one side of an interface raises for the other
  signals.yaml                 shared Jev signal questions, cited by id
  coverage/<dataset>.yaml      ledger: every source row accounted for
knowledge/clusters/<cluster>/
  consequences.yaml            cq.* records (list key `consequences`)
  unforeseen.yaml              uc.* records (list key `unforeseen`)
data/unforeseen/
  manifest.json                dataset manifest
  <dataset>.csv                owner-supplied rows with stable row ids
```

| Kind | Form | Example |
|---|---|---|
| Interface consequence | `ic.<slug>` | `ic.loads-investigate-supported` |
| Consequence | `cq.<slug>` | `cq.pre-2004-fabric-hazardous-materials-survey` |
| Unforeseen condition | `uc.<slug>` | `uc.existing-fire-water-fails-flow-test` |
| Signal | `sig.<slug>` | `sig.existing-fire-water-test-results-stated` |
| Dataset row | `<prefix>-<4+ digits>` | `B1-0042` |

### Actions (`works/actions.yaml`)

Jev reads `describes` and `excludes`, so write them literally with the
boundary cases, as for systems. Top-level keys: `actions` (the eight: new,
replace, upgrade, alter, repair, remove, retain, investigate), `answers`,
`work_type_defaults` and `existing_conditions`.

```yaml
version: 1
status: draft
sources: [{design: docs/plans/2026-10-04-next-wave-architecture-schema.md, anchor: "### Actions"}]
actions:
  - id: upgrade
    describes: The works increase the capacity or performance of an existing one, ...
    excludes: Like-for-like renewal is replace.
answers:                    # extra answers of the Jev choice sys.<leaf>.action
  several: ...              # more than one action stated: goes to Needs mapping
  not_stated: ...           # falls back to the work-type default
work_type_defaults: {new: new, extend: new, refurb: alter, remediation: repair, advisory: investigate}
existing_conditions:
  source: {clerk_file: data/taxonomy/asset-register.json}
  values: [{id: serviceable, label: Serviceable}, ...]   # 7 Clerk conditions
```

`work_type_defaults` must cover every work type in `profile/taxonomy.yaml`.
The user's choice of action is final.

### Interface consequences (`works/interface_consequences.yaml`)

One entry per row of the plan's table. List key `interface_consequences`.
Applies when the works touch `touches` of an interface of `type` with one of
`actions`, and the other side exists on the site and is not being replaced.
Direction follows Interfaces: `from` acts on `to`.

```yaml
- id: ic.supplies-investigate-supply
  type: supplies              # an interface type
  touches: to                 # from | to | either
  actions: [new, upgrade, alter]   # action ids, or `any` for either-side rows
  propose: {kind: investigation, label: Capacity of the existing supply}
  status: draft
  sources: [...]
```

Proposal `kind`: `investigation` (a work item), `discipline` (a package
suggestion), `approval` or `hold_point` (a delivery item), `obligation`
(package scope). Interface consequences also allow `work_item` (physical
make-good work).

### Consequences (`cq.*`, `clusters/<cluster>/consequences.yaml`)

For what interfaces cannot express, chiefly regulatory triggers on existing
buildings. List key `consequences`.

```yaml
- id: cq.pre-2004-fabric-hazardous-materials-survey
  when:                       # a predicate, may use `works`
    all:
      - works: {action: [alter, replace, upgrade, repair, remove]}
      - {det: existing_building_year, lt: 2004}
  propose:                    # one or more proposals
    - {kind: investigation, label: Hazardous materials survey of areas affected by the works}
  signals: [sig.hazardous-materials-survey-stated]   # optional: evidence it is already addressed
  governed_by: []             # rule ids; may be empty until a rule is verified
  clause_verified: false      # required when governed_by is non-empty
  severity: life-safety       # same values as failure modes
  status: draft
  sources: [...]
  notes: ...
```

### Unforeseen conditions (`uc.*`, `clusters/<cluster>/unforeseen.yaml`)

A state of the site or building commonly discovered during works, raised by
the work items that make it likely. A failure mode is a pitfall a document
can show; an unforeseen condition is discovered during the works. List key
`unforeseen`. Never record a frequency or cost percentage unless a source
states it, and then mark it unverified.

```yaml
- id: uc.existing-fire-water-fails-flow-test
  kind: unforeseen_condition  # unforeseen_condition | design_or_coordination_error | workmanship_defect | process_authority_supply_weather
  category: existing_systems_on_test
  attaches_to: {system: fire-active.fire-water}   # exactly one of system | interface | stage | package_kind
  when:
    all:
      - works: {action: [new, upgrade, alter], system: [fire-active.sprinklers, fire-active.hydrants]}
      - {det: existing_building, is: true}
  signals: [sig.existing-fire-water-test-results-stated]
  de_risk: {kind: investigation, label: Flow and pressure test of the existing fire water supply before design}
  contract: Provisional sum or separable portion for any supply upgrade.
  effect: [cost, programme, compliance]   # cost | programme | safety | compliance | quality
  severity: life-safety
  discovered_at: [design, testing_commissioning]  # one or a list
  status: draft
  sources: [...]
```

`category` is one of the plan's nine: `ground_and_site`, `existing_structure`,
`hazardous_materials`, `concealed_services_and_earlier_work`,
`existing_systems_on_test`, `authorities_and_utilities`,
`third_parties_and_occupation`, `design_and_scope`,
`supply_and_site_operations`. `discovered_at` is one of `design`,
`demolition_strip_out`, `excavation`, `construction`,
`testing_commissioning`, `handover_defects`. A `stage` target is one of
`investigation`, `design`, `approvals`, `procurement`, `construction`,
`completion`, `defects`; a `package_kind` target is `services`, `works` or
`supply`.

### Signals (`works/signals.yaml`)

One shared catalogue, so many records share one Jev question
([patterns](https://docs.typesafe.ai/patterns)). A signal is a noul in the
Questions format, written with the same authoring rules, at the top level of
the entry. `true` means the thing is stated, so a signal says what a passage
shows, never whether the condition exists. Criteria keys are quoted
(`"true"`, `"false"`); the checker accepts either form. Records reference
signals by id in `signals`.

```yaml
- id: sig.existing-fire-water-test-results-stated
  type: noul
  instructions: Using `text`, does the passage state measured flow or pressure results from a test of the existing fire water supply?
  criteria:
    "true": It states measured flow or pressure results for the existing supply.
    "false": It does not state test results; a requirement or intention to test is not a result.
  runs_on: [fire-active.fire-water]
  status: draft
  sources: [...]
```

### Predicates added

- `works: {action: [...], system: [...]}`: an in-scope work item with one of
  these actions on one of these systems. Either list may be omitted; actions
  must exist in `actions.yaml`, systems must exist.
- `system_existing: <system id>`: the system is on the site and is not being
  replaced.
- `system_present` keeps its meaning: the system is in the completed building.

Valid in `applies_when`, consequence `when` and unforeseen `when`.

### Building age: `existing_building_year`

Consequences and unforeseen conditions read the existing
`existing_building_year` determinant (integer, `pre_parsed`, narrow triggers
such as `year built` and `built in 1985`) and compare it in code. A separate
`construction_year` was drafted in K0 and merged into it on 4 October 2026, so
one fact has one determinant.

### Dataset sources and the coverage ledger

Owner-supplied lists (for example `docs/unforeseen/construction_interfaces_1000.xlsx`)
are exported to `data/unforeseen/<dataset>.csv` with a stable `row_id` first
column and registered in `data/unforeseen/manifest.json`. They are
AI-generated and unverified, and the manifest must say so:

```json
{"datasets": [{"id": "batch1", "file": "batch1.csv", "row_id_column": "row_id", "rows": 1000,
               "ai_generated": true, "status": "draft", "origin": "..."}]}
```

A record cites a row with a new source form, which the checker resolves:

```yaml
sources:
  - {dataset: batch1, row: B1-0042}
  - {design: docs/plans/2026-10-04-next-wave-architecture-schema.md, anchor: "### Actions"}
```

`design` cites a document in this repo by a heading line. Plans can be
untracked in a worktree, so a missing file warns; a present file must contain
the anchor. A dataset row alone does not justify a number: it is a lead, not
a source of fact.

Every dataset has a ledger `knowledge/works/coverage/<dataset>.yaml` naming
every row exactly once:

```yaml
version: 1
dataset: batch1
status: draft
rows:
  B1-0001: pending                                  # not yet processed
  B1-0002: {records: [uc.rock-at-footing-depth]}    # enriched into these records
  B1-0003: {rejected: duplicate_of, ref: B1-0001}   # a row or a record
  B1-0004: {rejected: too_vague}
  B1-0005: {rejected: out_of_scope}
```

The checker fails on a missing, extra or repeated row, an unknown record id
and an invalid disposition. Pending rows are allowed and counted in its
summary.
