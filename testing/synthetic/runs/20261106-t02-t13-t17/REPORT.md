# Run 20261106-t02-t13-t17: T02, T13 and T17, first live run

The first live run of three more types: T02 apartments, T13 mixed-use shop-top housing and T17 cladding remediation. There are three scenarios per type.

- **App:** git `6284fa1`, plus another session's uncommitted edits in the working tree. These are `internal/httpapi/profile.go`, `knowledge/profile/layout.yaml`, and the web Profile display files (`web/src/Profile.tsx`, `ProfileFields.tsx`, `profileApi.ts`, the trace CSS, `RfpEditor.tsx`, `Reports.tsx`). The Go change adds one display field (`scope_decision`), and the layout file sets where a field shows. Neither changes a reading, a package, a cost line, a program row or an RFP position.
- **How it ran:** on the harness server (:8091) with live Jev, one scenario at a time, in one organisation. No HTTP errors occurred. Every RFP draft returned 201.
- **Scores:** made with the final shapes and keys.

## Outcome

- **Filing and invoices:** sound. Kind and title were right on every document except one T17 council order. Every variant's trigger value was read correctly. T13 invoices were all right.
- **Profile:** good on T02 and T17, but weaker on T13. In every T13 run, five scale values are blank, because the mixed-use subclass fields are never asked (F32). The app does read the same facts under generic keys: units 42, GLA 600 m², 6 storeys.
- **RFP new failures:** T02 base 4, T13 28 to 29, T17 9. Few of these are app defects:
  - Most T13 failures are shape gaps: client questions and options keyed as absent without `app_differs`.
  - 16 of the 21 in ST02-traditional come from the scorer: a trigger cannot change `rfps[].expect`.
  - 5 of T17's 9 come from F35: the owners corporation's name is read in two forms, so the Client is a conflict.
- **New app defects:** F31 to F42.
- **No fabricated amounts.** No program links were violated in any run.

## Results by type

Profile values per scenario: T02 74, T13 76, T17 47. "Conflict" means a conflict that includes the true value.

### T02 apartments

| Scenario | Correct | Wrong | Missed | Conflict | Program | Skeleton | Cost plan | RFP new failures |
|---|---|---|---|---|---|---|---|---|
| base | 66 | 1 | 4 (+1 by design) | 2 | 65/103 | sk.new-dc-da ✓ | 38/59 | 4 |
| groundwater | 66 | 1 | 4 (+1) | 2 | 65/106 | ✓ | 38/61 | 5 |
| traditional | 66 | 1 | 4 (+1) | 2 | 64/104 | sk.new-traditional-da ✓ (scored ✗) | 38/59 | 21 |

- **Procurement:** 15/23 in all three; the misses are known gaps.
- **Invoices:** all amounts right. The town planner's invoice has no supplier (F40).
- **Fees:** 4 of 5 totals in base and groundwater; the structural total was lost (F38). 5 of 5 in traditional.
- **Firm names:** the civil firm is wrong in all three runs (F37).
- **The one wrong profile value:** noise, from the d13 trap at 0.73 to 0.82 (F34).
- **Misses:**
  - `quality_level` and `environmental_sensitivity` (not traced);
  - `avg_unit_size_sqm` (F32);
  - `proj.regulated_design` (known).
- **Planned conflicts:** GFA 6,200 against 6,184 m², and `multi_building`.

### T13 mixed-use shop-top housing

| Scenario | Correct | Wrong | Missed | Program | Skeleton | Cost plan | Procurement | RFP new failures |
|---|---|---|---|---|---|---|---|---|
| base | 66 | 0 | 8 (+2) | 77/113 | sk.new-dc-da ✓ | 38/55 | 15/25 | 28 |
| rise to three storeys | 66 | 0 | 8 (+2) | 74/109 | ✓ | 38/55 | 15/25 | 28 |
| contaminated land | 66 | 0 | 8 (+2) | 77/116 | ✓ | 39/57 | 16/25 | 29 |

- **Invoices, fees and firms:** all right.
- **Misses in every run:**
  - `site_sqm`, `podium_levels`, `tower_levels`, `residential_units`, `retail_gla_sqm` (F32);
  - `flag.cl46` (F33);
  - `flag.architect_lodges` (a missed "no");
  - `proj.regulated_design`.
- **Contaminated land:** the environmental package came only from the d19 proposal (`origin: document`). There is no suggestion (F41).
- **Program:** the reinforcement hold and the ground slab are missed (F42; the shape's built-in claim is wrong).

### T17 cladding remediation

| Scenario | Correct | Wrong | Conflict | Program | Skeleton | Cost plan | Procurement | RFP new failures |
|---|---|---|---|---|---|---|---|---|
| base | 40 | 0 | 3 (+4 by design) | 4/44 | sk.generic (known) | 15/33, 1 false positive (P06, known) | 7/10 | 9 |
| Project Remediate | 40 | 0 | 3 (+4) | 4/48 | sk.generic | 14/33, 1 FP | 6/9 | 9 |
| CDC | 41 | 0 | 2 (+4) | 1/41 | sk.generic | 13/32, 1 FP | 6/9 | 9 |

- **Invoices:** the Kestrel invoice is wrong in all three runs (F31).
- **Fees:** the QS total is lost in all three runs (F39).
- **Conflicts:**
  - the client (F35);
  - `flag.basement` (F36);
  - access constraints: urban against restricted hours, both reasonable.
- **CDC variant:**
  - The app correctly dropped the town planner cost line. The shape doesn't remove it, so it scored as a missed built-in.
  - The app's "Prepare and lodge the complying development certificate" did not match the shape's "…the CDC".

## RFP new failures, by cause

| Cause | T02 base / gw / trad | T13 (each run) | T17 base / PR / CDC |
|---|---|---|---|
| App defect | 0 | 1 (F33) | 8 (F35 5 or 4, F36 3) |
| Known (`proj.regulated_design` unread) | 1 / 1 / 2 | 1 | 1 |
| Shape gap | 2 | 25 | 0 |
| Scorer: a trigger can't change `expect` | 0 / 1 / 16 | 0 (1 in contaminated) | 0 / 0 / 1 |
| Scorer: design vehicle is user input | 1 | 1 | 0 |

**ST02-traditional:**
- The skeleton is right. The trigger lacks `skeleton:`.
- The 16 trigger failures are correct app behaviour, as the trigger's own `why` predicts:
  - architect: the D&C and post-novation stages are out; the tender, bidding, for-construction and construction-support stages are in;
  - structural: the tender and construction-documents stages are in; `specialist_element_design` is O, shown as included;
  - civil: detailed tender and for-construction are in.
- The other 5:
  - 2 shape gaps from base: architect `brief_masterplan` (`architecture.yaml:62-66`, where every boundary starts) and certifier `a4_construction` (an option);
  - 1 design-vehicle block (scorer);
  - 2 regulated-design blocks (known).
- The 3 "built-in" program misses are reworded base rows (and Landscape, not traced).

## Answers to the questions

1. **T13 scale.**
   - The subclass fields have no `triggers:` (`taxonomy.yaml:1426-1440`). `HarvestOwn` asks a scale question only when its trigger matches (`internal/profile/harvest.go:121-125`), so these fields are never asked.
   - T02 uses triggered keys.
   - T13 `site_sqm`: "consolidated site … about 1,800 m2" misses the "site area" and "lot size" triggers (`taxonomy.yaml:40-42`). The survey's figure is cut into the next passage at the colon (`internal/jobs/source.go:207`). No survey reading appeared in T02 or T13 (not traced further).
2. **T17 invoice (F31).**
   - `exish` (`internal/invoices/read.go:314`) doesn't know "Fee", so `settlePrinted` fails.
   - `Resolve` sets ex = total − GST = $15,400 (`read.go:755`).
   - The ledger then raises `gst_rate` and no `arithmetic` (`ledger.go:305-313`).
3. **T02 noise:** yes, the d13 trap. The passage is "Road traffic noise from the local street is low…". The criteria don't treat a low level as a denial (`determinants.yaml:809-830`).
4. **ST02-traditional:** see the section above.
5. **ST13 contamination:** confirmed. `contamination_level` (`package_defaults.yaml:135-141`) is not a field the profile has, so `packageFieldValue` (`internal/procurement/defaults.go:160-186`) rejects it.
6. **T17 skeleton and P06:** known gaps, named in the shape (lines 142, 148, 821, 826) and in FINDINGS.
7. **Other items:**
   - the fee totals: F38 and F39;
   - the firm: F37;
   - register lifecycle and supersession: known gaps;
   - discipline: F40;
   - the d01 kind: a key question.

## Shape and key gaps (recommended, not applied)

These are as listed in FINDINGS.md:

- T02 traditional: add `skeleton: sk.new-traditional-da`, and remove the two reworded program adds.
- T02 RFPs: architect `app_differs` `stage.brief_masterplan` (app include); certifier `stage.a4_construction` (app option).
- T13 architect: `blocking: []`.
- T13 `app_differs` entries, listed in FINDINGS (24 rows).
- T13 program: the reinforcement-hold and ground-slab rows become `built_in: "no"`.
- T13 keys: podium and tower levels become `read_by_design: false`; units and GLA are the owner's choice.
- T17 cdc-pathway: add "Town planner" to `removes.cost_plan`, rename the CDC row, and set `built_in.program` to `"partial"`.
- ST17 d01: also accept `statutory_instrument`.

## Scorer limits

- A trigger can't change `rfps[].expect` (known).
- The design-vehicle block should count as user input (`tools/score_shapes.py:46-52`).
- `proj.regulated_design` scores as missed (known).
- Trigger program labels are matched at 0.75 of their words.

## Harness

- The firm directory is organisation-wide and accumulates firms across runs (F37).
- Basalt Ridge Geotechnical is reused in T02, T11 and T13 with three ABNs.

## Owner questions

1. Is a CDC lawful for an 8-storey Class 2 recladding? The ST17-cdc premise is doubtful and is kept as an app test.
2. Should free text and flags settle on a strong majority (F35, F36, F27)?
3. Should the architect RFP always price "Brief and masterplan"?
4. Should the T13 subclass fields get triggers, or should the keys accept the generic keys?
