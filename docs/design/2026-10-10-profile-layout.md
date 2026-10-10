# Profile layout: one page, in build order

Date: 10 October 2026. Lane B (layout), with one knowledge file.

The owner reviewed Profile on Petersham and found no heading hierarchy,
facts split across several places, wasted width, and a separate Works tab
that repeated Scope of works under another name. Decisions:

1. **Header.** One identity line (address, client, council), the update
   button, and one status line: reading coverage, passages needing review,
   update time, thresholds, and a "to check" count that jumps to the first
   conflict. Source requirements open as a panel from that line.
2. **Project card.** Three columns (the building, site and planning,
   delivery), one line per value, the source count and reset beside the
   value. Fields that count the same thing (storeys and total storeys, GFA
   and total GFA, units and dwellings, car parks and garage spaces) show
   once unless they disagree. Empty non-essential fields wait behind
   "+ n more".
3. **Scope and works.** A rail in build order, then the chosen entry: one
   row per system with the scope tick, the works action, who provides it
   (By) and the evidence. One expand per row shows the note, sources,
   requirements and work items. Build order lives in
   `knowledge/profile/layout.yaml`, screen order only: Jev questions,
   fingerprints and work items keep the catalogue's order.
4. **Risks and compliance inputs sit under the system they affect.** The
   layout file places each RFP detail flag and each determinant without a
   system; a determinant otherwise goes under its first `systems:` entry.
   Approvals and stakeholder flags form "Planning and approvals", the first
   rail entry. The separate Compliance section is gone.
5. **Works tab removed.** The scope tick already writes the system's own
   (coarse) work item, so it is that item's inclusion; there is no second
   Included box for it. Other items (another part, a split, one added by
   hand without the coarse key) keep their own. Correct, split and retire
   open inside the system row. Provider and inclusion stay two stored
   fields: By says who provides, the tick says whether the works touch it.
6. **Kept.** The document register stays beside Profile.

What must stay true: every RFP detail flag has a home (checked at load, in
`tools/check_knowledge.py` and by `TestProfilePlacesInputsInBuildOrder`);
the layout file never changes what is read or stored.
