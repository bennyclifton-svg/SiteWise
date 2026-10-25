# Quoted consultant fees: line items before detail

25 October 2026. Lane A: the review layout and agreed fee additions.

The consultant comparison starts with firm names, each source fee on its own
row, added base-fee rows, then the total and Award. Source labels stay with each
firm when comparing several proposals; source positions do not imply matched
scope. Packages, source scope, exclusions, conditions and coverage follow.
The filename and Open proposal link leave the header; the firm name still
opens the existing proposal panel.

The displayed total uses the confirmed appointment amount, or the firm's stated
ex-GST total until confirmation, plus user-added base-fee rows. Go computes the
sum. Source breakdowns, repeats, alternatives, options and rates are not added
to that total. An unpriced addition leaves the total unknown; marking it
included does not add money. Each firm owns its own cell values.

Add row uses the existing comparison-row and cell APIs. Added base-fee cells
are editable in the main quoted view with an agreement note. Award confirms
the stated base fee through the existing action and offers the additional
priced rows as separate cost lines. The original source-stage award remains
available when there are no extra base rows. No budgets or source prices are
overwritten. Unknown base amounts can be confirmed through the Award entry.

No new dependency, service, migration or Jev call. Existing read/write budgets
remain 150/300 ms for price_table and 100/250 ms for submission_write.
Verification: monetary and award regressions, single- and multi-firm browser
flows, source-reading regression and responsive layout. Five-firm local speed
check: reads p50 6.5 ms / p90 7.2 ms; writes p50 8.3 ms / p90 11.2 ms.
