# Brief 00: Knowledge index

You are building the shared reference that twenty domain-map authors will
rely on. Each of them maps one project type: which consultants, cost lines,
program activities and RFP scope the job needs, and whether SiteWise's
built-in knowledge already produces them. They need to answer "is this built
in?" quickly and correctly, without each re-reading 10,000 lines of YAML.
Your index is how they do that.

Repository: `D:\AI Projects\sitewise`. Read
`testing/synthetic/DISCOVERY.md` first (about 10 minutes).

## Rules

- Do not change any application code or knowledge files. Write only under
  `testing/synthetic/reference/`.
- Every statement about how the app behaves cites the file and an anchor:
  an id, key or line number. Example: `knowledge/works/package_defaults.yaml
  baselines[1]`. If you could not confirm something in code, write
  **unverified**.
- Explain the *selection logic* (the `applies_when` and condition semantics
  and the Go that evaluates them), not just the lists. Authors must be able
  to predict what the app produces for a given profile and scope.

## What to write

Write `testing/synthetic/reference/KNOWLEDGE_INDEX.md`, with these sections.
Put any large table in `reference/index/<topic>.md` and link it from the
index.

1. **From profile to scope.** How the profile becomes work items (system plus
   action). Cover `knowledge/profile/scope_defaults.yaml`,
   `knowledge/works/actions.yaml` (`work_type_defaults`,
   `existing_conditions`) and the Go that applies them (`internal/profile`,
   `internal/works`). For each building class and work type, list the
   default systems and actions.
2. **Procurement.** `knowledge/works/package_defaults.yaml`: each baseline,
   scope addition and complexity addition, written as "when … then these
   consultants". Also `internal/procurement/defaults.go`, `proposed.go` and
   `discipline.go`: novation, head contract, trade packages, and how a
   package's stages are set.
3. **Cost plan.** `knowledge/costs/starter_lines.yaml`, `elements.yaml` and
   `budget_splits.yaml`, plus the Go that prepares lines (`internal/costs`).
   Cover:
   - the four sections (fees and charges, consultants, construction,
     contingency);
   - which line comes from where (profile, work items or packages);
   - what is never priced.
4. **Program.** `knowledge/program/skeletons.yaml`: each skeleton's
   `applies_when` and stages. `fragments.yaml`: group the 223 fragments by
   suffix (1a, 2, 7b, 9b, fitout, reno and the others), and give each
   fragment's id, label, stage, `applies_when` and activity keys. Also
   `durations.yaml`, `calendars.yaml`, and the Go that selects a skeleton
   and attaches fragments (`internal/program`). Say plainly which classes
   and work types have no specific content.
5. **RFPs.** For each `knowledge/rfp/*.yaml` with `discipline_ids`, give:
   - the discipline ids and stage keys and labels;
   - the scale rules;
   - which `positions` are selected by which site facts or flags;
   - the returnables.

   Also cover `modules.yaml` and `skeleton.yaml`, and list the disciplines in
   `data/intake/disciplines.json` that have **no** RFP template.
6. **Triggers and consequences.** List every "site condition leads to extra
   work" link the knowledge already encodes. Sources:
   - `knowledge/works/signals.yaml`, `interface_consequences.yaml` and
     `coverage/`;
   - `knowledge/clusters/*/signals.yaml`, `consequences.yaml`,
     `interfaces.yaml`, `unforeseen.yaml` and `failure_modes.yaml`;
   - the determinant `triggers` in `knowledge/determinants.yaml`;
   - `knowledge/profile/flag_readings.yaml`.

   Give each link as condition → profile key → consequence (consultant, work
   item, fragment, RFP position or cost line), with its source.
7. **Running the generators without Jev.** Find out whether the app can
   build the cost plan, program, procurement suggestions and RFP drafts from
   a profile entered by hand, with no Jev call. Look for HTTP endpoints, Go
   test helpers and `cmd/*` tools (for example `cmd/works-eval`). This
   decides how the harness compares the app's own shapes with the expected
   ones. Name the exact calls.
8. **Known gaps.** Building classes, work types and common conditions with
   no skeleton, fragments, package rule or RFP template.

## Done when

- Every section is filled with citations, and nothing in the index
  contradicts the code. Check at least 10 claims against the Go and say
  which ones you checked.
- Run `python testing/synthetic/tools/validate_shape.py --help` once, so you
  know the vocabulary it checks. Do not change the validator.

## Reply

Reply to the orchestrator in **150 words or fewer**: the files you wrote,
the three most important gaps you found, and anything you could not verify.
Do not paste the index into your reply.
