# RFP scope by stage

Owner direction, 2–3 November 2026. Lane A for the scope rules and answer
keys, lane B for the table layout. Every consultant RFP (all 18 disciplines)
now prints its scope as one table under **2. Scope of services**.

## What the consultant sees

| Stage | Scope | Position | Fee $ |
|---|---|---|---|
| Concept |  | Included |  |
|  | Site inspection | Included |  |
|  | Area schedule | Excluded |  |
|  | (a service homed in this stage) | Included |  |
| Across all stages | Services through the whole appointment |  | In stage fees |
| Reimbursables | … |  |  |
| Total fee |  |  |  |

On screen every stage starts collapsed (name and "8 items · 1 excluded");
opened, each line, element and service has an **In** and an **Excluded** tick
box, and the user can add, change and remove lines of their own (saved as
`custom:<stage>`; a built-in line can only be excluded). One fee per stage.
Weeks and hours are no longer asked for (R7 still carries durations).

## Rules

1. **Only included or excluded.** An optional or unknown stage, element or
   service is included; work done by others is excluded. Options totals and
   per-service option prices are gone.
2. **Appointment range.** A stage after a range the *user* set is excluded, so
   a planning-only appointment is not priced for the whole job. A default
   range excludes nothing.
3. **A service follows its stage.** Each service has home stages in
   preference order (`stage:` on each position in `knowledge/rfp/*.yaml`);
   it sits in the first that applies to the project, or across all stages.
   - A stage the user excludes excludes everything under it.
   - A stage left out by the range or a rule: an included service moves to
     its next listed stage in the appointment (or across all stages); an
     optional or unknown service follows the stage out.
4. **Lines are the deliverable.** Each stage's deliverable is a keyed list
   (`lines:`), split word for word at its semicolons and sentences; the stage
   text priced is its included lines. Line keys keep a saved tick on the
   right line if wording changes.
5. **Elements** (civil) appear as lines inside each by-element stage, and the
   stage is priced once, not element by element.
6. **Excluded rows name no party.**
7. Conflicting evidence about a fact still blocks issue.

Settings: `line:<stage>.<line>` (I or X) and `custom:<stage>` (an array of
up to 40 lines), allowed by migration `044_rfp_scope_lines.sql`.

## Answer keys and synthetic shapes

`data/eval/rfp/answer-keys/` was updated only where rules 1–3 and 6 change
the outcome; the update script refused any other change, and each changed
block carries a comment. The synthetic shapes (`testing/synthetic/types/`)
were not rewritten: `score_shapes.py` reads an expected option or unknown as
included when the RFP is by stage.

## For owner review

Home stages for 17 disciplines were drafted from the stage deliverables and
are `status: draft` with the rest of each rules file. Points to look at:

- **access**: `peer` has no natural stage (placed in S8 advisory); `occupied`
  mixes design review with audits (placed in S7); on advisory-only jobs
  `solutions` has no stage.
- **arborist**: `pruning` placed in S2 impact (S3 protection is arguable); no
  service sits in S7 advisory or S7 peer.
- **bca-certification**: every B stage needs a DA or SSD pathway, so a CDC
  certification job has no B stage for the statutory items. A CDC stage set
  looks to be missing.
- **civil**: no tender stage for ECI, alliance or PPP (services fall back to
  for-construction); `osd_new`/`osd_replacement` placed at schematic;
  `civil_declarations` under D&C without novation falls back to the D&C
  tender stage.
- **cost-planning**: no service sits in S3 authority estimate or S7 final
  account; `construction_plan` placed at S4 design.
- **electrical-services**: most services sit in S5 design; `utility` could
  be [planning, design]; four "later services" repeat their stage lines.
- **fire-engineering**: `remediation` placed in advisory; `brief`, `fer`,
  `interim`, `peer` repeat their stages; nothing sits in S2 early approval.
- **fire-services**: `declarations` covers design and installation
  certification (tender, then handover) and may need splitting; `staging`
  placed at design.
- **geotechnical**: `monitoring` placed in G2 field (G6 if it means
  construction dewatering); `environment` and `ass` placed in G3.
- **hydraulic-services**: its seven stage-named services are carried by the
  stage rows and not listed again; `fire` may be a separate appointment.
- **land-surveying**: `titling` spans S5 and S7 and may want its own stage;
  `interior` has no measured-building stage.
- **landscape-architecture**: `design` reads as a stage summary (placed in
  S4); `environment` mixes contamination and failed planting.
- **mechanical-services**: `energy`, `acoustic`, `execution` read as "by
  others" boundaries; several services repeat their stage lines.
- **structural**: no house documentation stage (`residential_framing` placed
  at construction documents); `dilapidation` placed at construction phase.
- **town-planning**: `compliance_matrix` overlaps the pre-lodgement summary;
  `court` placed at determination review.
- **traffic**: `road_safety_audit` and `public_road_design` placed at design
  review; `court` at conditions review.
- **waste-management**: several services copy their stage lines; advisory-only
  jobs have no stage for `completion` or `targets`.

Where a service repeats a stage line word for word, it shows twice; trimming
those is content work for the owner's finessing pass.

## Not changed

No Jev call, no new dependency, no new service. Report read and setting
budgets are unchanged (p50 100 ms / p90 250 ms); filing is untouched. The
old two-table code path remains for drafts saved before this change until
they are refreshed.
