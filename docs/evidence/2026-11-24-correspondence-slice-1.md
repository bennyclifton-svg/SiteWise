# Correspondence between parties: slices 1 and 2, one organisation

Date: 24 November 2026. Branch `register/eot-items`, merged to `main`.
Plan: `docs/plans/2026-11-23-correspondence-between-parties.md`, sections 16
to 18 (the peer review and the answers the owner accepted). Lane A (items,
results, the completion date); lane B for the register's labels and layout.

## What was built

- **Three register kinds**: `eot_claim`, `delay_notice` and `letter` in
  `project_delivery_items` (migration 079). Each is numbered in its own
  run: EOT-1, NOD-1, LTR-1 beside RFI-001. A numbered row keeps its kind.
- **Due dates in Go.** An EOT claim is due 28 calendar days after it was
  received, a letter five working days on the Program's calendar, a notice
  of delay never (no response is owed). Each is SiteWise's default and is
  labelled so ("check your contract") until the PM sets the project's own.
  A typed due date survives a later change to the received date, as an
  RFI's does. Lateness on an EOT claim is calendar days.
- **The result on an EOT claim**: granted, granted in part, rejected or
  more information needed, with the days granted and the reasons. Go
  refuses an assessed claim with no decided result, granted in part with
  no reasons or with as many days as claimed, granted with fewer days than
  claimed, and a rejection that grants days. The assessed date is filled
  when none is given. With moves to the raiser once assessed.
- **Optional terms** in `delivery_settings` (now all nullable): the RFI
  and letter period, the EOT period and the agreed completion date. One
  function (`store.DeliveryTerms.Resolve`) falls back to the defaults; no
  date calculation has a branch of its own. `PUT /delivery/settings`
  takes any of the three, and a null clears one. Setting terms is one
  undoable change.
- **Completion.** The register read totals the days granted on assessed or
  closed EOT claims (never a withdrawn one) and, when the agreed date is
  set, adds them as calendar days. Computed at read: nothing stored moves.
  With no agreed date the screen says "No agreed completion date is
  recorded, so nothing has moved". The Profile's target completion is
  untouched.
- **Attention** (warn only): `register.eot_overdue`, `register.eot_due`
  (within two business days) and `register.letter_overdue`. The register's
  overdue filter now covers all three kinds.
- **Chat**: `delivery.add`, `delivery.edit` and `delivery.settings` carry
  the new fields; dates and day counts are figures, so chat refuses one the
  PM did not give. Rows are found by EOT-4, NOD-2 or LTR-1. This fixed a
  latent fault: `registerItem` matched "RFI-1" by number alone and would
  have found EOT-1. The read prints each claim's result and the completion
  line.
- **Screens**: the grey row has an "Add as" picker (RFI, EOT claim, notice
  of delay, letter); a received item opens a sheet with its fields, and an
  EOT claim's sheet has a Result form with the standard words for each
  outcome. The three-dot menu's "Project terms…" replaces the RFI response
  time form. A completion line shows only when the project has an EOT
  claim or an agreed date (rule 5). A message whose Jev pick is a delay
  records an EOT claim, or a notice of delay, in one click.
- **Project report**: EOT claims, delay notices and letters are listed
  when the project has rows, with the days granted; they never join the
  "No … are recorded" line.

## What calls what

No path here calls Jev or the model. The register read, every register
write, terms, results and the totals are Go and PostgreSQL. Chat reaches
them only through the existing operations. No new route, table or
dependency: one migration widens three checks and adds two nullable
columns.

## What could fail

- Rollback to the previous release after a PM has cleared the RFI period
  to null: the old code reads `rfi_response_days` as a number and would
  fail on that row. Restore from the pre-deploy dump instead of rolling
  back once the new terms are in use.
- The 28-day EOT default and the 5-day letter default are SiteWise's
  convention. They are not AS 4902 periods; none is shown until the owner
  reads the form.

## Tests and timing

Run on the owner's PC, 24 November 2026, local PostgreSQL 17 on port 5433.

| Check | Result |
|---|---|
| `go test -p 1 ./...` | All packages pass except `internal/profile` `TestMeridianAnswerKey` (73/78), which fails the same way on `main` at `9472e1a`; not touched here |
| New store tests (`internal/store/register_claims_test.go`) | Pass: numbering per kind, due dates, result rules, completion with and without an agreed date, withdrawal, optional terms, Undo of a terms change, foreign org refused, calendar-day lateness |
| Chat command key, offline (`cmd/chat-eval`) | 200 cases, all pass; six new (`rg11` to `rg16`): record a claim, record its result, set the agreed date, two reads, and an invented figure refused |
| Browser (`web/tests/delivery.spec.ts`) | 4 of 4 pass, one new: record an EOT claim, the default due date, a result granted in part with reasons, the agreed date and the date after EOTs, NOD-1 in its own run |
| Full browser suite | 112 pass, 58 fail. Every one of the 58 also fails on `main` at `9472e1a` (checked file by file in a clean worktree): Program, RFP, profile, procurement award, navigation, reports and the correspondence address test (which expects no "mail" in a project named "Mail house", stale since migration 078) |

Timing, `TestRegisterClaimsBudget` (1,000 rows, a quarter assessed EOT
claims; store level, 20 samples):

| Path | Budget p50 / p90 | Measured p50 / p90 |
|---|---|---|
| `delivery_read` | 100 / 250 ms | 28.8 / 29.6 ms |
| `delivery_write` (EOT claim recorded) | 100 / 250 ms | 1.9 / 2.3 ms |

`cmd/intake-bench` now seeds EOT claims into its 1,000-row register and
times an EOT create and a result write under `delivery_write`; the full
HTTP bench was not run for this change.

## Not built (slices 3 to 5)

Outside organisations signing in, partner grants, a builder as host,
notification mail, the party's page, list views and one-item export. Each
needs the decisions in section 18 put into a plan, and outside access needs
its own security review before it ships (lane C), as guest access had.
