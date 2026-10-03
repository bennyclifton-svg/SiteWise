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
  `conditions` (`key`, `label`, `options`). Copied from Clerk's taxonomy as
  data, without cost-uplift text. Site conditions that determinants already
  cover (bushfire, flood, heritage, contamination, water source) are left out.
- `project_facts.yaml`: `facts:` list, with the same shape as determinants
  (consent number and dates, contract form and basis, defects period, design
  life). Ids must not collide with determinants.
- `typical_systems.yaml`: `status`, `sources`, `typical:` list of
  `{subclass, work_type, systems: [leaf ids]}`. These are suggestions for
  the thin-brief workflow, never evidence.
