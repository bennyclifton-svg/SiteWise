# Profile slowdown: workload isolation

WP-26 speed-gate diagnosis; no runtime change in this experiment. The prior
turn fixed concurrent whole-part creation and retained a failing full-workload
result. This experiment shortens the loop to distinguish workload and cached
connection effects from the already measured pure computation cost.

A temporary benchmark branch stopped after the frozen Spec Home rebuild/edit
loop. Temporary rebuild stage timers logged only operations over 100 ms. All
experiments replayed recorded Jev latency locally; no live call was made.

| Setup | Runs | Rebuild median ms | Edit median ms |
|---|---:|---|---|
| Frozen fixture directly after sign-in | 5 | 28.1–28.9 | 47.5–48.6 |
| One uploaded document, smaller-project profile warm-up, then fixture | 3 | 28.0–29.4 | 48.6–49.4 |
| Full 20-file/two-round upload and profile warm-up, then fixture | 3 | 31.942; 32.846; 28.896 | 58.439; 64.013; 50.979 |

Each run measured 40 frozen-fixture rebuilds and edits. None reproduced the
700 ms tail or a rebuild stage above 100 ms. Full upload history increased
response cost in these repeats without a similarly large rebuild increase.
This does not prove a particular query-plan, JIT, lock or cache explanation.
The earlier failing full-workload measurements remain authoritative failures.

Ignored diagnostic logs are `.tools/profile-short-{1..5}.log`,
`.tools/profile-warm-short-{1..3}.log`, and
`.tools/profile-full-warm-short-{1..3}.log`. Their temporary early-return branch
did not produce normal benchmark gate verdicts and is not release evidence.
The temporary environment switch, early-return branch and stage timers were
removed, restoring the checked implementation and complete benchmark flow.

The next investigation must retain full upload history when examining the
response-read cost. An isolated fixture or small-project warm-up is insufficient
to reproduce that cost, and a passing isolated run cannot close the gate.
Automatic proposal generation remains disabled. No requirement, budget,
prerequisite, owner-review gate or remaining work package has been removed.
