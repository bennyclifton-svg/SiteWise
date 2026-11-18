# Consent conditions answer key

The key for `docs/plans/2026-11-17-consent-conditions.md` lives in
`private/` beside this file. That folder is ignored by git: the notices are
real councils' determinations on real sites, and this repository is public.

The owner supplies and reviews the key. Nothing here is drafted from an
invented case, and until a file says `reviewed: true` (the owner's mark) no
Jev pick shows green or is accepted: picks stay amber "to check".

## One file per notice: `private/<document>.yaml`

```yaml
document: petersham-2016          # a name; never the notice's words
source: Petersham/03 ANX Q PPR/…  # where the notice is on the owner's machine
reviewed: false                   # only the owner sets true
modification: false               # true for an s4.55 modification
conditions:                       # every numbered condition, in printed order
  - number: "46"                  # as printed: "46", "A1"
    page: 12                      # the page it starts on
    stage: before_cc              # before_cc | before_work | during_works |
                                  # before_occupation | before_subdivision |
                                  # ongoing | throughout
    party: consultant.landscape   # a discipline id, trade.main_works or applicant
    system: site                  # a top-level system id, or none
    evidence: true                # something must be handed over to show it is met
    marker: ""                    # a modification's "deleted" or "changed"
evidence:                         # later documents and what they evidence (Task 8)
  - document: landscape-plan-l100
    conditions: ["46"]
```

The condition's text is never copied into the key.

## Running it

```
go run ./cmd/consent-eval -pages "<notice.pdf>" -name petersham-2016 -list
go run ./cmd/consent-eval
```

`-pages` writes `private/<name>.pages.json` (the notice's text page by
page, as filing extracts it) and lists each condition's number, page, gate
and heading to help draft the key; it makes no Jev call. Without flags the
tool splits every keyed notice and scores the split (it must be exact before
any pick is scored); with a recording (`private/jev.jsonl`, written by
`-live`) it replays Jev's answers and reports agreement per field. It only
reports. `go test ./internal/consent` runs the same split check and skips
when there is no key.
