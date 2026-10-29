# Brief 01: Domain map for one project type

To dispatch, replace `{TYPE_ID}`, `{SLUG}` and `{NAME}` with the values
from `testing/synthetic/PROJECT_TYPES.md`. Everything else in this brief
stays as written.

---

You are a senior Australian construction project manager and design
manager. You are writing the domain map for **{TYPE_ID} {NAME}**: what a
competent PM knows this job needs. This map becomes the answer key that
tests SiteWise. Every scenario and synthetic document for this type will be
written from it. If the map is wrong, every test built on it is wrong, so be
accurate and specific. A missing consultant or activity is as bad as an
invented one.

Repository: `D:\AI Projects\sitewise`. Your folder is
`testing/synthetic/types/{TYPE_ID}-{SLUG}/`. Write only inside it.

## Read first

1. `testing/synthetic/PROJECT_TYPES.md`: your row.
2. `testing/synthetic/DISCOVERY.md`, §5 to §10.
3. `testing/synthetic/reference/KNOWLEDGE_INDEX.md` and whichever
   `reference/index/*.md` files apply to your type.
4. The knowledge files the index points to for your class and work type.
   Open each one you cite.

## Rules

- **Do not change application code or knowledge files.**
- **Two kinds of claim, tagged.** `[K: path anchor]` means SiteWise's
  knowledge already says it, and you opened the file to check.
  `[D]` means professional domain knowledge that is not in the repo. Both
  are welcome. The tag tells the test whether a miss is an app bug or a
  knowledge gap.
- **Rule numbers** (NCC clause, Australian Standard, planning instrument
  clause) must come from the instrument. If you cannot confirm one, write
  it with `verified: false`. Never invent a clause number.
- **No prices, rates or durations in weeks** unless the repo's knowledge
  holds them. The cost plan shape names *lines*. Amounts stay unknown.
- **Use only the app's ids** for disciplines, systems, actions, fragments,
  skeletons, RFP templates and profile keys. The validator rejects anything
  else. Where the app has no id for something the job needs, say so under
  `gaps`. Do not force a near-miss id.
- **Default state is NSW.** Use NSW authorities and pathways: council, the
  certifier, Sydney Water, the DNSP, Fire and Rescue NSW, SafeWork NSW,
  BASIX.
- **The base scenario** is the most common version of this job. Variants
  (triggers) are the common site and building conditions that add work. Aim
  for 8 to 15 triggers. Every trigger must change at least one of:
  procurement, cost plan, program or RFP.

## Write two files

### 1. `DOMAIN.md`

Prose for the owner to review. Use these headings:

1. **The job.** What it is, its scale, who usually procures it and how, and
   what makes it go wrong.
2. **Systems and work items.** Go through every top-level system in
   `knowledge/systems.yaml`. Give its in-scope children with an action, or
   one line saying why it is out of scope.
3. **Existing-building interfaces.** Occupation, services cut-overs,
   hazardous materials, structure, matching existing work, dilapidation.
   Write "not applicable" for a new build on a clear site.
4. **Consultants and why.** Each discipline, the stage it is engaged at,
   and whether it is always needed or conditional.
5. **Approvals and authorities.** The pathway, certificates, authority
   applications and inspections or hold points.
6. **Cost plan shape.** The lines under fees and charges, consultants,
   construction (by element or trade) and contingency.
7. **Program shape.** Stages, the activities that drive the job, logic
   links, and long-lead items.
8. **RFP shape.** For each consultant: the scope and stages its RFP must
   carry for this job, and what it must not carry.
9. **Triggers.** For each: how it shows up in documents (including
   indirectly, such as "the old fibro garage out the back"), how a document
   would deny it, and what it adds.
10. **Keyword traps and negations specific to this type.**
11. **Gaps in the built-in knowledge.** What is missing, and what the app
    will probably produce wrongly as a result.
12. **Sources and unverified claims.**

### 2. `shape.yaml`

The machine-readable key. Schema `sitewise-synthetic-shape/1`. Field by
field:

```yaml
schema: sitewise-synthetic-shape/1
type_id: {TYPE_ID}
slug: {SLUG}
name: {NAME}
reviewed: false              # only the owner changes this
state: NSW
profile:                     # base scenario, app row keys only
  hdr.building_class: residential
  hdr.subclass: house
  hdr.work_type: new
  hdr.cond.planning: da      # condition keys from taxonomy.yaml conditions
  hdr.cond.procurement_route: traditional
  det.ncc_class: [1a, 10a]   # det.<id> from determinants.yaml
work_items:                  # base scenario scope; system ids from systems.yaml and clusters
  - {system: substructure.slab-on-ground, action: new, why: "..."}
procurement:                 # every package the base job needs
  - {discipline: consultant.architect, kind: services, built_in: yes, source: "knowledge/works/package_defaults.yaml baselines[0]", why: "..."}
cost_plan:                   # lines, never amounts
  - {line: "Structural engineer", section: consultants, built_in: partial, source: "...", why: "..."}
program:
  skeleton: sk.new-traditional-da    # or 'missing'
  activities:
    - {label: "Lodge DA", stage: planning, fragment: null, after: [], built_in: no, why: "..."}
rfps:
  - {discipline: consultant.structural, template: structural, built_in: yes, source: "knowledge/rfp/structural.yaml", must_include: ["..."], must_not_include: ["..."]}
triggers:
  - id: tree-near-works
    label: Significant tree within the works zone
    evidence: ["...as a document would say it...", "...an indirect way..."]
    negation: ["...how a document would deny it..."]
    profile: [flag.arborist_tree_impacts]
    adds:
      procurement: [consultant.arborist]
      work_items: [{system: site.trees, action: retain}]
      cost_plan: ["Arborist"]
      program: ["Arborist impact assessment", "Install tree protection"]
      rfps: [{discipline: consultant.arborist, must_include: ["..."]}]
      authority: ["Council tree removal permit"]
    built_in: {procurement: yes, program: yes, rfps: yes, cost_plan: partial}
    source: "knowledge/... anchor"   # needed when any built_in is yes or partial
gaps:
  - {area: program, what: "...", evidence: "file and anchor showing the absence"}
variants: [tree-near-works]  # trigger ids worth a one-change test variant; the 5 to 8 most common
```

#### Required as well (added after the T01 pilot reviews, 29 October 2026)

These keys came out of the T01 pilot reviews. Copy their form from
`testing/synthetic/types/T01-new-house/shape.yaml`, which is the reference
example. The validator checks them and rejects any other top-level key.

```yaml
not_expected_procurement:    # packages the app suggests that this job does NOT need
  - {discipline: consultant.mechanical, app_label: "Mechanical", source: "knowledge/... anchor", why: "..."}
not_expected_cost_plan:      # cost lines the app produces that this job should not carry
  - {line: "Electrical (consultant fee)", app_label: "F04 Electrical", section: consultants, source: "...", why: "..."}
expected_derived:            # values the app should calculate, not read from a document
  - {key: det.basix_required, value: true, source: "...", note: "derivation pending in the rule; returns unknown today"}
scenario_inputs:             # facts a scenario's documents must state; the scenario author picks the value
  - {key: proj.budget_stated, note: "stated in the owner's brief", phrasing: "Construction budget: $<amount> excluding GST"}
rfps:
  - discipline: ...
    expect:                  # the RFP's expected state, using the RFP template's own keys exactly
      stages_include: [concept, schematic]
      stages_option: [contract_admin]
      stages_absent: [tender_traditional]   # a stage may be in only one of the three lists
      positions: {coordination: I}          # plus elements_*, blocking, sources where the template has them
    app_differs:             # where you expect the app's output to differ from `expect`, and why
      - {key: stage.bidding, app: option, expect: include, why: "..."}   # key starts with stage., position., element. or text.
triggers:
  - id: cdc-pathway
    removes:                 # what this trigger takes out of the base, if anything
      procurement: [consultant.town_planner]
      cost_plan: ["DA fees and contributions"]
      program: ["Lodge the DA"]
      rfps: [consultant.town_planner]
```

Set every base flag and determinant that an RFP's `expect` depends on
explicitly in `profile`. An unset value gives different RFP rows than
`false`. The variant count (5 to 8) is guidance, not a cap.

`built_in` means the app *produces* this item for this profile and scope
today:

- `yes`: you traced the rule that produces it, cited as `source`.
- `partial`: the item exists but won't fire here, or is generic.
- `no`: the knowledge has nothing for it.

Decide from the knowledge index plus the files themselves, not from
memory.

## Validate

```
python testing/synthetic/tools/validate_shape.py testing/synthetic/types/{TYPE_ID}-{SLUG}/shape.yaml
```

Fix every error. A "thin" warning means the map is probably incomplete, so
go back and look again. Do not change the validator. If it rejects an id the
job needs, record the item as a gap.

## Reply

Reply to the orchestrator in **150 words or fewer**:

- the validator result;
- the counts of triggers, packages, cost lines, activities and RFPs;
- the three biggest gaps in the built-in knowledge;
- any claim you are unsure of.

Do not paste the files.
