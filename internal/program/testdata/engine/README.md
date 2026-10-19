# Shared engine fixtures

Each `*.json` file here is read by the Go engine (`engine_test.go`) and must be
reproduced exactly by the TypeScript engine (`web/src/program/engine.ts`).
The file name equals `name`.

## Input (same JSON as `program.Input`)

```json
{
  "name": "…", "why": "what the fixture proves",
  "calendar": {"start": "2027-03-01" | "", "working_week": 31, "days": [{"day": "YYYY-MM-DD", "kind": "public_holiday|rdo|shutdown|working_override", "label": "…"}]},
  "activities": [{"id": "a", "parent": "s", "ordinal": 0, "kind": "stage|activity|milestone|hold_point", "label": "…",
                  "duration": 5 | null, "low": 4, "high": 7, "constraint": "asap|snet|mso", "constraint_date": "YYYY-MM-DD"}],
  "links": [{"id": "ab", "pred": "a", "succ": "b", "type": "FS|SS|FF|SF", "lag": -2, "lag_unit": "working|calendar"}]
}
```

Defaults: missing `kind` = activity, `type` = FS, `lag_unit` = working,
`constraint` = asap, `working_week` 0 = 31 (Mon–Fri). `duration` null or
missing = unknown (a milestone is always 0; a stage with children ignores
its own duration). `low`/`high` absent = the typical duration in all three runs.
Empty `start` = relative program: index 0 is Monday of Week 1, no dates are
returned and `days` are ignored.

## Expect (only the keys present are compared)

| key | compared with |
| --- | --- |
| `finish`, `finish_low`, `finish_high` | `computed.finish.typical_index` / `low_index` / `high_index` (exclusive PF) |
| `finish_dates` | `{typical, low, high}` date strings or null |
| `complete` | `computed.finish.complete` |
| `status` | `{known, total, words}` |
| `driving` | `computed.driving_links`, exact list in input link order |
| `conflicts` | `computed.conflicts`, exact list in input row order |
| `cycle` | `computed.cycle` (closed id list) |
| `cycle_words` | labels of the cycle joined with " → " |
| `activities` | per id, any of `es ef ls lf tf ff critical start finish unknown`; `why` is prose. An empty object means nothing is computed. |

A null in an expectation means the output value is null.

## Rules the expectations rely on (plan section 6)

- Half-open intervals: `ef = es + duration`; shown finish = date(ef − 1);
  a zero-length row shows its start only (finish null).
- Calendar-day lag uses instants: an activity starts at the beginning of
  date(es) and finishes at the beginning of the day after date(ef − 1); a
  zero-length row finishes at its start. Forward pass takes the first index
  on or after the shifted instant, backward pass the last index on or before.
- Stage with children: hidden Start(S) (the stage's own node, zero length)
  and Finish(S). Links into the stage anchor on Start(S); FS/FF out of it on
  Finish(S), SS/SF on Start(S). Row = es min / ef max / ls min / lf max of
  children, tf = least child float, ff = Finish(S)'s free float.
- Free float = min over successors of (successor ES − that link's unclamped
  forward result), or PF − EF with no known successor.
- Cycle: walk back from the least outline key `(stage ordinal, child ordinal
  or −1, id)` among unordered nodes, always to the least-key unordered
  predecessor; keep the repeated stretch, read it forwards, rotate to its
  least key, show hidden nodes as their stage and fold Start(S) → child →
  Finish(S) into the stage.

All expectations except `perf-2000.json` were worked by hand; the `why`
fields show the arithmetic. `perf-2000.json` is written by
`go run ./internal/program/cmd/genfixture -seed 1`; its expected finish is
the Go engine's own answer (a regression and cross-language check).
