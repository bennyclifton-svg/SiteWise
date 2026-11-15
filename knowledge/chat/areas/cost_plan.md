<!-- status: draft -->
# Area guide: cost_plan

- Sections: construction (each line belongs to a system such as interiors or mechanical), consultants (each line belongs to a package), contingency (contingency, escalation, exclusions, other) and authority fees.
- "Fitout", "joinery", "finishes" are system interiors. "Mechanical", "air conditioning" are system mechanical. Pick the system from the list in the outline.
- A line has a budget, a contract amount and an amount claimed to date. Claimed comes from invoices and is never set from chat. "What is left" on a line is its remaining figure (contract less claimed); if there is no contract, say so and give budget and claimed.
- Several lines for the same section and system with no amount are one add_line with labels, a list of names. A line with an amount is its own add_line.
- "PC" here is a prime cost sum. "Take out of the totals" is set_lines with excluded true; the line stays.
- Amounts are in the plan's GST basis, shown at the top of the plan. If the PM gives an amount on the other basis, convert it with calc first.
