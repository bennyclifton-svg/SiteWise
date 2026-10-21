# Builder comparison: totals and expandable work sections

## Compact-table follow-up

The owner's follow-up places contractor names first, six major work sections
next, and one ex-GST total row last. Dates, attachment and award actions move
into contractor menus. Conditions and miscellaneous detail are collapsed.
Repeated source/comparison/lump-sum totals are removed from the default view.
Review total contains source evidence, warnings, manual correction and the
renamed **Confirm total for award** action. Confirmed corrections still take
precedence; no award confirmation is performed automatically.

Section summaries only report whole-section inclusion/exclusion when explicitly
named in the text. Component prices require an explicit GST basis and are
normalised in Go; multiple prices are not summed. Otherwise the section says
Unclear and offers excerpts. This is intentionally conservative: saved text is
not a verified scope matrix. Consultant flow and money-write semantics remain
unchanged. No dependencies or Jev calls were added.

Focused builder/store/export tests and four browser regressions pass. Updated
builder gate: read p50/p90 1.58/2.13 ms against 150/300 ms; write plus reread
2.92/3.49 ms against 100/250 ms. Production frontend build passes. Browser checks
assert a single total beneath the sections and desktop vertical fit; mobile
keeps horizontal table scrolling. This follow-up is Lane A for evidence labels,
with the layout part in Lane B. No wider benchmark rerun was needed for the
unchanged filing, award and write paths; the broader failures below remain open.

User outcome: compare builder quotes without treating totals, stage breakdowns,
allowances and options as independent extra prices. Lane A: money and award trust.
This shortens review by putting quoted totals first and collapsing source detail
into major physical work sections. Consultant comparisons remain unchanged.

The head-contract price table reads saved proposal passages, recognises explicit
total labels, checks builder-margin arithmetic, and computes money in integer
cents in Go. A reconciled printed subtotal takes precedence over back-calculating
GST from an inclusive total. Conflicts and absent GST basis stay unresolved.
There are no new dependencies, services, PDF reads or Jev calls on this path.

The source total is a separate read-only projection. **Use quoted total** uses the
existing user cell-correction endpoint. Existing corrections are preserved.
Raw extracted breakdown amounts cannot be moved into builder schedule arithmetic
or selected for award; selected builder prices must be confirmed by the user.
Unused generic consultant rate/reimbursable rows are retired, preserving user
overrides, known prices and active award history. No budget or baseline is changed.

Major sections expand across firm columns. Literal term lookup groups source
wording for navigation only: it does not infer inclusion or completeness. Detail
labels distinguish allowances, options, stage/component figures and totals, but
do not claim a fully resolved scope hierarchy. Stored passages can span pages, so
links say **Source section starts p. N**. A detached source retains its saved
reading with an explicit warning and no broken source link. Excel exports retain
the separate source total and non-additive source details.

Verified against the saved three-quote comparison: Enmore $207,272.73 ex GST;
Kaposi $200,658.70 ex GST including builder margin; Nexus $191,136.71 ex GST,
retaining the printed subtotal and flagging the one-cent GST/rounding difference.
No comparison price was accepted on the user's behalf.

Validation on the local development host:

- Builder amount/relationship unit tests, store integration and export test pass.
- Existing comparison alignment, submission and actor-isolation focused tests pass.
- Four browser tests pass: builder review, alignment, existing-firm attachment,
  and consultant comparison-to-award/report/undo. Desktop and mobile screenshots
  are in `.tools/builder-comparison-desktop.png` and `builder-comparison-mobile.png`.
- TypeScript/Vite production build passes; existing bundle-size warning remains.
- Builder-specific budget gate (20 samples): price-table p50 1.75 / p90 2.29 ms
  against 150 / 300 ms; cell write plus table reread 3.39 / 4.10 ms against
  100 / 250 ms. These measurements use a small synthetic builder fixture.
- Full replay benchmark: price-table 9.4 / 14.7 ms; submission-write 13.4 /
  18.5 ms; proposal-read 1824.4 / 2026.0 ms; filing 514.0 / 800.8 ms. These pass
  their budgets. This broader benchmark does not use the real three-quote fixture.
- The full replay gate is **not green**: specification_issue p50 120.6 > 100 ms,
  spec_home_profile_edit 55.9 > 50 ms, report_setting 104.0 > 100 ms. No budgets
  were loosened. Raw results: `tmp/builder-bench-results.json` and samples beside it.
- The wider Go run passed proposals, HTTP API and RFP packages but hit a foreign-key
  fixture failure in TestPlanningAndWorkWritesRejectForeignActor. That test passed
  in the subsequent focused serial run. The known profile ordering test
  TestProfilePlacesInputsInBuildOrder was excluded from the wider run.

Limits: grouping depends on saved text quality; no matching wording is not an
exclusion. Unknown totals require manual confirmation. Mobile retains horizontal
table scrolling. The signed comparison-report format was not redesigned here.
