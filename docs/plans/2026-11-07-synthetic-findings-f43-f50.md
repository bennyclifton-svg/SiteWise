# Fixes for the synthetic-corpus findings F43 to F50, the T03, T04 and T06 recommendations, and the combined services design

Date: 7 November 2026. Owner direction: "proceed with 1, 2 and 3": fix the
app findings F43 to F50, write the combined services design (R17), and
apply the recommended T03, T04 and T06 shape and key edits. Findings:
`testing/synthetic/FINDINGS.md`, section "From run
`runs/20261107-t03-t04-t06`". Evidence:
`testing/synthetic/runs/20261107-t03-t04-t06/REPORT.md`.

App fixes are lane A. Each is fixed only as far as its finding, test first.
Existing tests, answer keys and speed budgets must still pass. No new
dependency and no new Jev round trip. After the merge, the nine T03, T04
and T06 scenarios are re-run unchanged to measure each fix.

## Owner answers (rulings R15 to R19)

- **R15:** a vehicle crossing is work in the public road (applied to T04).
- **R16:** on a refurbishment, an included system whose works action is
  unsettled stays in scope (F43).
- **R17:** a combined services engineer is often engaged under one
  agreement and one invoice. The user chooses, per project, one
  multi-services package or separate discipline packages. Design first.
- **R18:** over height means a clause 4.6 variation and a DA (applied to
  ST04).
- **R19:** an empty floor in an occupied tower is both partial occupation
  and a live environment, so the T03 keys accept either. T06 site access
  likewise accepts restricted hours or urban constrained.

## Direction choices (decided before code)

**F43 (R16), refurb scope defaults to in.** On a refurbishment, a system
whose presence is included is filed as existing only when its works action
settles on a non-works value (retain, existing, no work). When the action
is unsettled (every candidate below the action floor, or a conflict that
includes a works action: new, alter, upgrade, replace, remove), the system
goes into scope with its action shown as unsettled (the existing `several`
path where works actions conflict). A settled action keeps today's
behaviour.

**F44, refurb program order.** Fix the three T03 REVIEW defects:
- fire stopping gets a predecessor (after the services rough-in it seals)
  and its hold point gates concealment (ceilings and linings), not every
  interiors row;
- services rough-in comes before ceilings when there is no strip-out
  anchor;
- the sprinkler pipework pressure test comes before ceilings without the
  link being dropped as a loop.

**F45, implied scope ignores denials.** An `implied_by` pattern does not
fire on a sentence that denies the match ("not in scope", "no …", "not
required", "excluded", "are not"). Use the repo's existing negation
handling if one exists, rather than a second list.

**F46, "Name, Role, Company" addressee.** When an addressee line splits on
commas and its last part is a company name (Pty Ltd, Ltd, Limited, Group,
Holdings, Council, Corporation, Trust, "The Owners"), offer that last part
alone as the client.

**F47, empty stages.** A skeleton stage with no rows is an empty summary
with zero duration that dates from its predecessor. It is not an unknown
leaf, so it no longer spreads "unknown" to every later row. The skeleton's
stage list is unchanged (no stage is dropped).

**F48, LGA label form.** "Local government area: North Sydney" offers the
name after the label as an LGA candidate, and a council heading in capitals
("NORTH SYDNEY COUNCIL") matches like "North Sydney Council".

**F49, fire services drawings.** The filing discipline description makes
fire services drawings (hydrant, sprinkler and fire-water layouts, FS-
sheets) fire services, and keeps fire engineer for fire engineering reports
and performance solutions. The intake answer keys must still pass.

**F50, Interior Design on a refurb.** The commercial refurb baseline
suggests Interior Design only when the project's scope has interiors work.
A services-only upgrade gets none.

## Work split

| Group | Findings | Area | Worktree / branch |
|---|---|---|---|
| Profile scope | F43, F45 | `internal/profile` (`existing.go`, `build.go`), `knowledge/profile/scope_defaults.yaml` | `sitewise-fx2-scope`, `fix/f43-scope` |
| Profile details | F46, F48 | `internal/profile` (`addressee.go`, `details.go`) | `sitewise-fx2-details`, `fix/f43-details` |
| Program | F44, F47 | `internal/program`, `knowledge/program/fragments.yaml`, `knowledge/clusters/*/interfaces.yaml` | `sitewise-fx2-program`, `fix/f43-program` |
| Filing and procurement knowledge | F49, F50 | `data/intake/disciplines.json`, `knowledge/works/package_defaults.yaml` | `sitewise-fx2-knowledge`, `fix/f43-knowledge` |
| Testing (no app code) | shape and key edits, R19 | `testing/synthetic/` | main tree |
| Design (no code) | R17 | `docs/design/2026-11-07-combined-services-package.md` | main tree |

Each worktree starts from local main `0b7d4a9`. Nobody edits
`knowledge/profile/layout.yaml`, `internal/httpapi/profile.go` or
`internal/knowledge/layout_test.go`, which hold another session's
uncommitted work. Only the group that changes a profile question bumps the
question version.

## Testing changes

Every recommendation under "Shape and key gaps" in the F43–F50 section of
FINDINGS:
- T03 and T06 certifier and BCA keys re-keyed for F9;
- variant `skeleton:` lines (T03 da-pathway, T06 tank-needs-consent, T04
  cdc-industrial-code);
- triggers that restate `expect` (T03 da-pathway, T06 tank-needs-consent,
  T04 cdc-industrial-code and flood-affected);
- T06 `position.existing` absent;
- T04 cdc "CDC fee" alias;
- T06 tank-needs-consent adds a town planner cost line;
- T06 pump-tank-only keeps the generic design rows;
- T03 and T06 `hdr.cond.quality_level` not read by design.

Also: R19 in the T03 and T06 keys, `det.site_class` and `det.saline_soil`
in the T04 base profile. After the app fixes, `built_in` claims that change
are updated.

## Budgets

Filing stays p50 ≤ 1 s and p90 ≤ 2 s; every other path keeps its
`bench/budgets.json` budget, including profile edit and program open. The
intake bench gate is run after the merge.

## Done

- Every group merged to local main with its tests.
- The Go suite passes apart from the failures that pre-date this work
  (TestMeridianAnswerKey, TestProgramRoutes).
- The bench gate passes.
- The design note is written for the owner.
- The nine T03, T04 and T06 scenarios are re-run into
  `runs/20261107-t03-t04-t06-after-fixes/`, and each fix is confirmed or
  reported.
