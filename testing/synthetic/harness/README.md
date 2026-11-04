# Harness

Drops a scenario's documents into a local SiteWise, wave by wave, as a user
would, then saves what the app built and scores it against the scenario's
answer keys.

## Run

```
bash testing/synthetic/harness/serve.sh          # separate SiteWise on 127.0.0.1:8091
python testing/synthetic/harness/run.py --allow-live-jev --wave-snapshots \
    testing/synthetic/scenarios/ST01-base [more scenarios]
bash testing/synthetic/harness/serve.sh stop
```

`serve.sh` runs its own database (`sitewise_synth`) and file store, so test
projects never mix with the owner's real projects in the `:8080` dev
organisation. `run.py` refuses `:8080` and anything not on loopback.

Each run writes `testing/synthetic/runs/<stamp>/<scenario>/`:

| Path | What |
|---|---|
| `run.json` | project id, git head, waves, every drop and its HTTP result |
| `wave-<n>/` | documents and profile after each wave (with `--wave-snapshots`) |
| `snapshot/` | unmodified API bodies after the last wave: documents, document details, profile, profile sources, invoices, procurement board, packages, price tables, cost plan, program, RFP reports |

## What the harness presses

Per wave: the drops, then **Update project profile**, waiting for filing,
proposal and invoice reading, and profile reading to go idle. After the last
wave, the clicks in `SCORING.md` §7: prepare the cost plan, prepare the
program, create one RFP per services package and open it (which assembles
its draft). It never corrects, accepts, dismisses or deletes anything.

Raw Jev exchanges are not recorded: the server has no recording switch,
and this work does not change app code. Every value the app stored is
saved with its band, confidence and who decided it.

## Score

```
R=testing/synthetic/runs/<stamp>/ST01-base
python testing/synthetic/harness/score.py --scenario testing/synthetic/scenarios/ST01-base \
    --snapshot $R/snapshot --json $R/field_score.json
python testing/synthetic/tools/score_shapes.py --shape testing/synthetic/types/T01-new-house/shape.yaml \
    --snapshot $R/snapshot --json $R/shape_score.json [--trigger cdc-pathway]
```

`score.py` covers the register, profile, invoices and fees; `score_shapes.py`
covers procurement, cost plan, program and RFPs. Outcomes are listed in
each file's docstring. There is no blended score.

## Tests

```
python -m unittest discover -s testing/synthetic/harness
```
