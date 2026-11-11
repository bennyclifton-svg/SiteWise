# Procurement: name firms before their quotes arrive

Owner request and approval, 11 November 2026. Lane A (procurement packages).

## Outcome

Before any quote arrives, the PM clicks an empty Firm 1-3 cell on a
Procurement row and types the firm's name: the two or three firms the RFP or
RFT goes to. The cell then shows the firm as **Invited**. When that firm's
quote is dropped on its cell, it fills that cell. The row shows at a glance
who has and has not responded (time-to-decision).

## Owner decisions (11 Nov 2026)

1. A firm keeps the cell it first took on the row, invited or first
   received. On rows with no nominations, cards order by when each firm's
   first proposal was uploaded rather than by the printed date (usually the
   same order).
2. An invited firm that never sends a quote stays visible as Invited after
   the award: the row records who did not respond.

## Model

A nomination is its own record, not a proposal: `package_invitations`
(migration 052), one row per firm per package, the org's directory firm,
deleted with its package. Nothing is priced, compared or awarded from it.
The submissions table, price table, comparison report, award, Program and
invoices are unchanged.

- **Name typed.** A directory firm whose name key matches is used (a name
  key drops case, spaces and "Pty Ltd"); otherwise a new confirmed directory
  firm is made, as Add firm in Prices does. Suggestions come from the
  existing directory search while typing.
- **Combined engagements.** A nomination made on a grouped member goes to
  its group, as proposals do. On the board, a group shows its own
  nominations and those made on its members before they were combined; a
  grouped member shows none. Combine and uncombine move nothing.
- **Board.** `ProcurementBoard` adds the invited firms with no card on the
  row as `status: "invited"` cards (firm id and name, invited date, no
  submission, no figures). A firm with a proposal on the row shows its
  proposal card instead. Cards order by when the firm joined the row: its
  nomination, else its first proposal on that package (any status, so a
  revised quote keeps its cell). A card whose firm is not yet read orders
  by its own upload.

## Quote arrives

- **Dropped on the invited cell** (file or register document): the upload
  carries `firm_id`; the new submission is opened with that firm and a
  user firm-name decision, so the read keeps the user's firm (the same rule
  as attaching to an existing column). One Jev fan-out after filing, as
  today.
- **Dropped elsewhere on the row:** unchanged. If the read matches the same
  directory firm (ABN, then name key) the card takes that firm's cell. No
  fuzzy matching: "Smith Arch" and "Smith Architects Pty Ltd" stay two firms,
  and the PM picks the firm in the proposal's firm panel (existing) or
  removes the nomination.
- **Proposal detached:** the cell shows Invited again.

## API

- `POST /projects/{id}/procurement/packages/{pkg}/invitations` with
  `{"firm_id": …}` or `{"name": …}`: 201 with `{package_id, firm_id,
  firm_name}`. Nominating twice is one nomination. Name 1-200 characters.
  A retired package, another project's package or another org's firm is 404.
- `DELETE /projects/{id}/procurement/packages/{pkg}/invitations/{firm}`:
  204; 404 when there is no such nomination.
- `POST …/submissions?firm_id=…` and `POST …/documents` with `firm_id`:
  the upload is that firm's.
- Speed path `submission_write` for both writes; the board stays
  `price_table`.

## Screen and style (web/DESIGN.md, cost-tables.css)

- Empty cell: the ghosted drop icon stays; beside it a plain button,
  "Add firm", in tertiary text that shows on row hover and on focus. The
  whole cell is its click target. Its accessible name says the slot and
  the row ("Name firm 2 for Town Planner").
- Editing: the cell becomes the field (`inline-cell-editor field-plain`,
  same row height, no border, focus changes colour, never geometry). Enter
  or a suggestion saves, Esc cancels, leaving with typed text saves, as in
  every inline editor. Suggestions float under the cell in the raised
  menu style (`page-action-menu`): name, then ABN muted. Plain buttons, not
  an ARIA listbox (DESIGN.md, project switcher).
- Invited firm: name over the status word "Invited" in the user's origin
  colour (`--origin-user`: rust, the user's word), no figure, the usual
  detach X labelled "Remove invitation". Not a button to Prices: there is
  nothing to price yet.
- Dragging a file over an invited cell outlines it dashed, as the Invoices
  drop zone does.
- Prices / Compare counts proposals only, never invitations.

## Budgets

- Nominate and remove: p50 ≤ 100 ms, p90 ≤ 250 ms (`procurement_invite`,
  the `submission_write` budget), new benchmark samples.
- Board read with nominations: p50 ≤ 150 ms, p90 ≤ 300 ms
  (`procurement_board`, the `price_table` budget), new benchmark samples.
- No Jev on typing, search, nominate or remove.

## Out of scope

More than three nominations from the row; sending or addressing the RFP to
nominees; a "declined to tender" status; filling nominations from the
profile's invited-builders detail (2026-11-11 accommodation plan); editing
firm details from the cell; fuzzy name matching; showing nominations in
Prices.

## Done

- Store tests: nominate by name and by id, twice is once, name key reuses
  the directory firm, bad names refused, other org / other project / retired
  package not found, grouped member routes to the group, board pairs a
  proposal with its nomination in the nomination's place, detach shows
  Invited again, remove, upload with a firm keeps that firm through the read.
- API tests: the routes above, 404 from another org, 400/422 on bad input.
- FK sweep fixture and inventory for the new table.
- Browser test: click a cell, type, Enter, Invited shows and survives a
  reload; drop a quote on it and its card takes the same cell.
- Existing procurement, award and price-table tests pass.
- Benchmarks within budget.

## Follow-up, 11 Nov 2026 (owner feedback)

**Correcting an invited name.** A click on an invited firm opens its cell as
the field with the name in it; Enter or a suggestion saves, Esc or an
unchanged name leaves it. `PUT .../invitations/{firm}` with `{name}` or
`{firm_id}` (`Store.CorrectInvitation`, no Jev, `procurement_invite`
budget). The invitation keeps its time, so the firm keeps its cell. A firm
that is only a name typed for this row (no ABN or contact details, no
quotes, no other rows) is renamed, or dropped when a directory firm is
chosen, so the misspelling leaves the directory. Any other firm keeps its
name and the cell takes the corrected firm. A firm already on the row is
refused (409, "That firm is already on this row."). X still removes.

**A proposal waiting for a package** is the table's first row on both tabs,
not a strip above it: "Choose package" in the to-check colour where the
discipline goes (packages listed under Consultants and Trades), the firm's
card in Firm 1, shown only. A combined proposal offers "Combine for this
proposal" in Prices; its tick list opens in a full-width row beneath.

**PI expiry** is no longer shown on the board's firm cards (owner: the
board is a summary of who, price and status). The date stays in the card's
hover description and in the stored submission.
