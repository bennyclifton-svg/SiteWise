# Editable fee comparison

Lane A. Outcome: align differently structured consultant offers quickly without
inventing a breakdown. RFP rows remain the starting point; comparison labels and
base/option placement are explicit user overrides, never edits to an issued RFP.
Program stages will feed RFP suggestions through its planned Update/Keep workflow;
comparison overrides must survive that refresh. No competing stage catalogue.

Build: editable labels and base/option placement, custom fee rows, and a per-firm
move/combine operation with source passages and undo. A bundled fee is counted
once; other covered rows can be marked Included. Never divide a bundle by guess.
Unknown amounts cannot be combined. Rates cannot be combined into lump sums.
All arithmetic stays in Go. Existing award, report and export consume the same
stored result. Edits are locked after award and scoped to organisation/project.

Budgets: price table p50/p90 150/300 ms; writes 100/250 ms. No Jev on edits.
Done: persisted edits survive reload/RFP sync, one firm's grouping does not change
another's, source amounts are counted once, undo restores the prior mapping,
isolation and award checks pass, and timings are measured.

Do not lower acceptance thresholds to reach an arbitrary accuracy percentage.
Measure fee capture, mapping accuracy and correction time separately. Source
coverage, quantities/allowances, exclusions and unit rates need visible review;
an apparently low price is not proof of equivalent scope. No new dependencies.

Out of scope: changing Program implementation, automatic scope equivalence,
invented quantities or allocations, reissuing RFPs, or automatically appointing
a consultant. The four private PDFs are review evidence, not repository fixtures.

## Direct table controls (user feedback, 19 October)

Replace the expanded arrangement form with row-end up/down buttons and fee-level
move/combine actions. Select a source, choose a destination within that firm, then
confirm; an existing destination fee is included automatically. Keep undo in the
firm header. Row labels open the description editor; Add row preserves custom
services and unit rates. Show review triangles without repeated Check/Jev text;
source wording and scope notes stay available when opening the fee. Narrow the
item column from 34% to 22% and give firms equal width.

Lane A because these controls invoke price changes. Existing Go operations and
budgets remain: read 150/300 ms, write 100/250 ms p50/p90. No new runtime AI,
services or dependencies. Done: arrows persist section changes, combination
counts both prices once, undo restores them, rates have no placement arrow,
keyboard labels work, and desktop/mobile remain usable. Do not change actual
Hale fees while verifying. Program and extraction are outside this refinement.
