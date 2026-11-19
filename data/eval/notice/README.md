# Notice pick answer key

The key for Jev's one `notice` question on an email body
(`docs/plans/2026-11-17-correspondence-transmittals.md`, Task 2, owner
decision 5). It lives in `private/` beside this file, which git ignores: the
messages are the owner's real mail, and this repository is public.

Status: **empty, by design.** The owner supplies about 50 real messages,
half of them notices or RFIs. Nothing here is drafted from an invented case.
Until the key is reviewed and `data/mail/thresholds.json` is approved, every
pick is stored blank and never shown; no chip appears and nothing is created.

## One file per message: `private/<name>.yaml`

```yaml
message: camp-y-2026-03-delay      # a name; never the message's words
source: "Camp Y/Mail/…"            # where the .eml is on the owner's machine
reviewed: false                    # only the owner sets true
notice: eot_delay                  # none | rfi | direction | variation_claim |
                                   # eot_delay | payment_claim | dispute | other_notice
traps: [forwarded]                 # optional: quoted, forwarded, in_passing,
                                   # quote_request, already_answered
```

The body is never copied into the key; the scorer files the message through
the ordinary body document (`mailin.BodyDocument`) and reads the pick.

## The cut-off

`data/mail/thresholds.json`, written only after the owner reviews a run:

```json
{"question_version": "notice-1", "amber": 0.8}
```

Run the key twice, the options in their order and reversed, because Jev
leans toward the first option
(https://docs.typesafe.ai/model-jaggedness/jev-1.13); a cut-off is the
probability of the action taken (https://docs.typesafe.ai/confidence).
