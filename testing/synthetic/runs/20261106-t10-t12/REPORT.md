# Run 20261106-t10-t12: knock-down rebuild, dual occupancy and granny flat

First live run of three new types: T10 knock-down rebuild, T11 dual
occupancy (two townhouses on a Torrens split) and T12 granny flat. Each has
a base and two variants: nine scenarios. App: local main `0ee5b22` plus
another session's uncommitted, display-only changes (`profile.go`
`scope_decision`, `layout.yaml`). Live Jev. ST10-da, ST11-cdc and
ST12-sewer ran partly at the same time, as separate projects in one
organisation. Scores are after the scorer, shape and key fixes listed at the
end; none of those fixes changes what the app did.

## Results

| Scenario | Profile correct | Conflicts | Wrong / missed | Supplier | Fee totals | Skeleton | Program matched | Links violated | Procurement | Cost plan | RFP new failures |
|---|---|---|---|---|---|---|---|---|---|---|---|
| T10 base | 52 | 4 | 3 / 0 | 3/3 | 6/6 | cdc ✓ | 56/104 | 8 | 7/11 | 23/37 | 16 |
| T10 DA | 51 | 5 | 3 / 0 | 3/3 | 8/8 | da ✓ | 56/104 | 5 | 9/13 | 26/40 | 19 |
| T10 separate demolition | 50 | 6 | 3 / 0 | 3/3 | 6/6 | cdc ✓ | 54/107 | 6 | 7/11 | 23/39 | 20 |
| T11 base | 50 | 3 | 1 / 2 | 3/3 | 6/6 | da ✓ | 49/107 | 5 | 8/12 | 27/44 | 4 |
| T11 CDC | 52 | 2 | 1 / 1 | 3/3 | 5/5 | cdc ✓ | 47/106 | 4 | 7/11 | 26/45 | 6 |
| T11 three or more | 50 | 3 | 1 / 2 | 3/3 | 8/8 | da ✓ | 50/109 | 5 | 10/14 | 29/47 | 4 |
| T12 base | 54 | 2 | 0 / 5 | 1/3 | 5/6 | cdc ✓ | 46/73 | 4 | 7/8 | 22/28 | 10 |
| T12 DA | 54 | 2 | 0 / 5 | 2/3 | 5/6 | da ✓ | 47/74 | 5 | 8/9 | 24/30 | 12 |
| T12 sewer main | 55 | 1 | 0 / 5 | 2/3 | 5/6 | cdc ✓ | 46/75 | 4 | 7/8 | 22/30 | 10 |

Also:
- No fabricated amounts.
- On all 27 invoices, number, date, ABN and amounts were right, and the
  `abn_invalid` and `arithmetic` traps were flagged.
- Kind, title and status were 100 % right on 139 documents.
- Firm names: 54 of 57 (F23).

## What repeated

- **F6:** the T10 BASIX fee proposal filed blank in all three runs (Jev
  0.31 to 0.35). There is no BASIX package or RFP. In the pilots this was
  borderline in one run of nine; here it fails every time.
- **Unneeded consultants on houses:** town planner, electrical, hydraulic
  and mechanical are still suggested, plus an "Active fire protection" cost
  line. These are known gaps.
- **T01 open items still open:** missed "no" flags (T12 groundwater,
  multi_building, excavation_adjoining), unread `proj.regulated_design`,
  and `flag.community` read from "neighbour notification".
- **Program known gaps:** the construction certificate comes before
  construction documentation (T11). On CDC jobs, client sign-off for
  lodgement comes before the construction documents (T10, T12). The shapes
  name both.
- **The architect's email (d04)** was filed as architect, where the key
  says none. This is the same question as T05 d04.

## What is new (F17 to F30)

The documents describe two buildings, the house knocked down and the one
built. Several readings take the old house as the project:

- **F17:** `det.existing_building` is read false for a demolished house.
- **F19:** in T11, `hdr.subclass` gets 10 "house" votes from hazmat and
  survey passages about the old house, so it becomes a conflict.
- **F18:** passages about old footings, lead paint on old window frames,
  and consultants' inspections set work actions to `investigate`. The
  footing, slab, frame, window and lining fragments then do not fire. That
  causes most of the program's built-in misses, and adds a seven-item
  Investigations list to every RFP.

Other new defects:

- **F20 (traced):** "DA fees and contributions" stays on every CDC job. The
  query checks `value_state='known'`, but profile rows store 'set'.
- **F19, continued:** with the subclass blank, the T11 scope falls back to
  the apartment list. That adds Vertical Transport, Fire Services and
  Access consultants and RFPs in all three T11 runs, not only the CDC one.
- **F21 (traced):** "Caringbah South NSW 2229" is cut to "Caringbah".
- **F22 (traced):** proposals addressed "To: architect, for client" leave
  the client blank, so five T10 RFPs block on Client.
- **F23 (traced):** the firm name is taken from the letterhead's descriptor
  line: "Civil and stormwater engineers".
- **F24 (traced offline):** the stated fee total is lost when a proposal
  quotes the cost of work for the long service levy.
- **F25:** "Building designer" is not an architect alias, so most T12
  design documents file blank.
- **F26:** a house connected to the street water main is read as an
  authority asset. It is wrong once and a conflict five times, and drives
  16 RFP new failures.
- **F27:** the F15 majority rule counts passages, not documents. Five
  geotechnical passages about rig access outvote the survey's
  "unrestricted".
- **F28 (traced):** the reinforcement hold point comes after the pour.
- **F29:** house trades are not linked in order. Fit-offs finish before
  plasterboard, and site establishment links to nothing.
- **F30:** "Dangerous goods" is read as a hazardous process on houses, from
  asbestos passages. It shows in 7 RFP briefs.

Most RFP new failures are knock-ons:

| Failure | Cause |
|---|---|
| `existing_interfaces`, `retained_services` | F17 |
| `water_asset`, `authority_asset` | F26 |
| Dilapidation, retention, shoring | `excavation_adjoining` unresolved |
| Client block | F22 |
| Surveyor access, live operations | F27 and the occupancy conflict |

Details, causes and code locations are in `../../FINDINGS.md`.

## Compared with the pilots

- **Profile:** as good or better than the pilots (50 to 55 correct,
  against 43 to 55).
- **Skeleton:** right first time in every run.
- **Program:** weaker (45 to 63 % matched, against 61 % for T01), mostly
  because of F18 and F29.
- **RFPs:** 4 to 20 new failures, against 4 to 15. T10 is highest.
- **Supplier:** 24 of 27, against 27 of 27. Two of the misses are by
  design: no surveyor package exists when the invoice is read. One is F25.

## Harness

Two RFP drafts returned 409: Hazmat in ST10-da and BASIX in ST12-sewer.
Both came while three projects were drafting at once. It was not a reading
problem: no jobs were queued. Suspected cause: serialisation retries ran
out (`refreshReport`, then `ErrVersionConflict`). Next time, run scenarios
one at a time or log the 409 body.

## Scorer, shape and key changes

Before these, RFP new failures per run were 21, 24, 25, 5, 7, 5, 14, 16 and
14.

- **Scorer:** `score_shapes.py` lets a trigger set `skeleton:`;
  `harness/score.py` expects `no_supplier` only when no supplier was read.
  Both have tests (83 tool and 12 harness tests pass).
- **T10 to T12 shapes** (reasons in each DOMAIN.md):
  - pathway skeletons;
  - architect `fact:budget` block removed, because the budget is stated;
  - CC fee and town planner lines on a DA;
  - "Building designer" aliased to "Architect";
  - T12 certifier no longer keyed to the pre-F9 template;
  - T10 certifier client-flag positions keyed as in T11.
- **ST10 keys:** `proj.lga` also accepts "Sutherland", as ST12 does. This
  is a convention call for the owner to confirm.
