# ST04 scenario review (brief 03r)

Reviewer: independent (senior NSW PM who reads JSON and Go). Date: 7 November 2026.
Scenarios: `ST04-base`, `ST04-cdc-industrial-code`, `ST04-flood-affected`. None was edited.

**Verdict: FIX.** The job, dates, fees, invoices, register keys and both variants are sound. But four
profile keys are marked `read_by_design: true` although the app cannot read them as the documents
are worded. The flood variant's `not_read` gives the wrong reason. Seven findings; the changes are
listed under each one.

## How I checked

1. **Checker.** `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST04-*`
   gave `ok` for all three scenarios.
2. **Read-only Go run.** I ran a scratch `package main` with
   `go run -overlay overlay.json ./internal/zzst04r`. The overlay maps that virtual package to a file in
   my scratchpad folder `st04r`, so nothing was written to the repo. It loads `knowledge.Load("knowledge")`.
   For every `carries.says` sentence in all three manifests (381 sentences) it runs `profile.Harvest`,
   then `profile.LabelQuestions`, the code path the worker's `labelCall` uses
   (`internal/jobs/worker.go`). It records which profile questions would be asked and with which
   candidates. There was no database and no Jev call. Then I tried the reworded sentences proposed
   below.
3. **Code read** for each claim below: `internal/profile/harvest.go`, `questions.go`, `works.go`,
   `internal/knowledge/details.go`, `profile.go`, `knowledge/profile/taxonomy.yaml`,
   `project_details.yaml`, `flag_readings.yaml`, `knowledge/determinants.yaml`,
   `internal/intake/markdown_identity.go` (through DISCOVERY.md §4.3), `internal/invoices/ledger.go` and
   `internal/proposals/read.go`.

## The author's "missed by design" claims

| Claim | Ruling | Evidence |
|---|---|---|
| `proj.target_da_lodgement` and `proj.target_construction_start` are only in the d04 email | **Confirmed.** No read-kind document states either date. d04 starts with an email header, so it is filed as `correspondence` and is not read. The sentence "have the DA in with council by 30 June 2026" would not fire the `target_da_lodgement` trigger anyway (it needs `lodg`). The only key it fires is `proj.program`. | Go run; `knowledge/profile/project_details.yaml` triggers; `data/profile/reading.json` |
| `hdr.scale.hardstand_sqm` and `office_percent` have no triggers | **Confirmed.** Neither has `triggers` in `taxonomy.yaml` warehouse `scale_fields`, and `ScaleField.TriggeredIn` returns false with no triggers. **But** the office line fires other questions: see finding 6. | `knowledge/profile/taxonomy.yaml:645-652`; `internal/knowledge/profile.go anyMatch` |
| `proj.regulated_design` has no extraction | **Confirmed.** The detail is a one-line entry with no `extraction`, `triggers` or `asks`, so `compileReading` gives it no reading. **The key contradicts the claim**, though: see finding 1. | `knowledge/profile/project_details.yaml:154`; `internal/knowledge/details.go:173` |
| d02 and d07 title blocks are tables, so number and date are missed | **Confirmed.** Cells are not read (§4.3). Both `# headings` sit outside the table, so each title still settles. d18's table invoice is read by the invoice harvest, which reads cells. | DISCOVERY.md §4.3; `internal/invoices/harvest.go` (cell labels) |
| Flood variant: `det.flood_planning_level` is not asked because its question `runs_on` `site.flooding`, which the app does not add | **Outcome right, reason wrong.** The profile reading never checks `runs_on`. A determinant is asked whenever a trigger fires and code finds a candidate. For a `number` determinant, `candidatesFor` offers only m² areas. "31.2 m AHD" gives no candidate, so the key is never asked. **Would the flood evidence add `site.flooding`? It can.** Whether it does depends on Jev, not on code. See finding 5. | `internal/profile/harvest.go:96-99, 150-180`; `questions.go LabelQuestions` (no `runs_on` filter); Go run |

## Findings

1. **All three `expected/profile.json` files: `proj.regulated_design` is keyed `read_by_design: true`,
   but the app has no reading for it.**
   - *Evidence:* `project_details.yaml:154` gives the key no `extraction` or `triggers`. In the Go run,
     the d01 and d12 sentences ask only `det.state`. Each notes.md says "expect a miss", which disagrees
     with the key. `validate_expected.py` now accepts a `not_read` override (lines 233-238), and ST03
     already uses it for this key.
   - *Change:* in all three files, set
     `"read_by_design": false, "not_read": "proj.regulated_design has no extraction (knowledge/profile/project_details.yaml; internal/knowledge/details.go compileReading); the app never reads it"`.
     The RFP drivers still derive `no` (no Class 2, 3 or 9c part, `internal/rfp/drivers.go:358-369`),
     so the RFP keys are unaffected.

2. **All three: `hdr.scale.dock_doors` = 12 is keyed as read, but no read document gives a candidate.**
   - *Evidence:* the count needs `dockCountRe`, which is a number directly before "dock door(s)" or
     "loading dock door(s)" (`harvest.go`). d01 "Dock doors: 12 recessed docks with levellers and
     shelters" gives no candidate. The key is pre-parsed, so no candidate means no question. d13's
     "12 recessed docks" fails the same way. d15 and d16 ("12 recessed dock doors") would also fail
     because "recessed" sits between, and they are drawings, which are not read. The Go run asks
     nothing for any of them.
   - *Change:* in each manifest, change d01's carry to "Dock doors: 12 dock doors, recessed, with
     levellers and shelters". The Go run then gives candidate 12. In base and flood, also change d13's
     proposal line to "12 dock doors". Keep the key at 12 with sources d01, d13 (base and flood) or d01
     (CDC). Do not leave the wording and flip the key to unread: the brief prefers a stated base value
     that the app reads.

3. **All three: `flag.tfnsw_referral` = no is keyed as read from d10 and d13, but their sentence fires
   no `tfnsw_referral` trigger.**
   - *Evidence:* the triggers are `traffic[- ]generating development` and
     `referred to (?:tfnsw|transport for nsw)` (`flag_readings.yaml:64-65`). "The development is below
     the Schedule 3 thresholds, so referral to TfNSW is not required" matches neither. It asks only
     `state_road` (Go run). The documents are told never to write "traffic generating development",
     so nothing reads this flag.
   - *Change:* reword the d10 and d13 carries (d10 only in CDC) to "The development is below the
     Schedule 3 thresholds, so it will not be referred to TfNSW". The Go run then asks
     `proj.flag.tfnsw_referral`. **Shape comment:** the `traffic-generating-development` negation in
     `shape.yaml:808` uses the unreadable wording and should change the same way.

4. **All three: `flag.fire_specialist_systems` = no is keyed as read from d12, but its sentence fires no
   trigger.**
   - *Evidence:* the trigger needs "performance solution", "fire" or "departure" within 100 characters
     of "facade", "cladding" and similar words (`flag_readings.yaml:112-113`). In "Performance solutions
     are proposed only for warehouse travel distances and the perimeter vehicular access; none is
     proposed for the facade, cladding or structure", "performance solution" is about 105 characters
     before "facade". "Structure" is not "structural". The Go run asks nothing.
   - *Change:* in each manifest, change d12's carry to "No fire performance solution is proposed for
     the facade, cladding or structure; performance solutions cover only warehouse travel distances
     and the perimeter vehicular access". The Go run asks `proj.flag.fire_specialist_systems`, and the
     sentence still keeps "storage" and "structural steel" away from "fire".

5. **ST04-flood-affected `expected/profile.json` and `notes.md`: the `det.flood_planning_level` reason
   is wrong, and the notes wrongly say the app never adds `site.flooding`.**
   - *Evidence:*
     - `HarvestOwn` asks every live determinant whose trigger fires (`harvest.go:95-99`).
       `LabelQuestions` and `valueQuestion` never read `Question.RunsOn`. `runs_on` matters only for
       evidence nouls and signals (`Catalog.EvidenceQuestions`).
     - The real gate is `candidatesFor`: a `number` determinant without its own case gets only
       `areaRe` (m²) candidates. "Flood planning level: 31.2 m AHD …" and "RL 31.20 m AHD, the flood
       planning level" therefore give none and are not asked.
     - A passage that also holds an m² figure *is* asked, with area candidates. In the Go run, "…flood
       planning level of RL 31.20 m AHD; the 6,000 m2 hardstand…" offered 6000. Jev must answer none.
   - *Evidence on `site.flooding`:*
     - The route exists. A flood passage labelled `site` gets the leaf label question for
       `site.flooding`, whose description names "flood planning levels and freeboard, overland flow
       paths, flood impact assessment".
     - If that is accepted, `DetailCall` asks presence and action (`worker.go DetailCall`,
       `questions.go EvidenceQuestions`).
     - A green or amber `included` presence becomes a work item (`works.go ProposedWorks`,
       `appliedWorkRow`).
     - The d07 phase (consultant scope naming the system), d13 and d14 (floor set at the flood planning
       level, flood-compatible materials) can plausibly answer `included` with action `investigate`.
       So the shape's work item may appear. Whether it does is up to Jev.
   - *Change:*
     - Replace `not_read` with: "flood_planning_level is a number determinant with no candidate rule of
       its own; candidatesFor (internal/profile/harvest.go) offers only m² areas, so '31.2 m AHD' is
       never a candidate and the question is not asked".
     - In notes.md "Missed by design", drop "which the app's warehouse scope lacks and does not add for
       this trigger". Add that `site.flooding` can be added through evidence presence and action
       readings. If it appears, the scorer should count it as a match to the shape's work item, not as
       noise.
     - Add a trap note: keep m² figures out of the paragraphs that state the flood planning level (d02
       flood item, d14 Flooding section).

6. **All three, d01: the office line fires the GFA and floor-area questions with a 500 m² candidate.
   This incidental is not declared.**
   - *Evidence:* "Office: 500 m2, about 6% of the gross floor area" asks `hdr.scale.gfa_sqm`,
     `hdr.scale.total_gfa_sqm` and `det.floor_area`, each with candidate 500 (Go run). If Jev picks it,
     8,000 against 500 becomes a conflict. The GFA line already gives the office's 500 m².
   - *Change:* reword the carry to "Office share: about 6% of the gross floor area". The Go run asks
     nothing for it, and the key is unread anyway. Otherwise, add it to the notes' incidentals.

7. **ST04-base and ST04-flood-affected notes, d14 `hydrocarbon-contaminants`: the trap does not fire.**
   - *Evidence:* the trigger is the literal `contaminat`, and "contaminants" does not contain it
     (contamina-n-ts). The Go run asks nothing for the d14 sentence. The same error is in
     `types/T04-warehouse/DOMAIN.md` §10 item 8 and in the shape's keyword-trap gap.
   - *Change:* no key change. Remove the d14 trap from the notes, or use the word "contamination" in
     the d14 water-quality sentence if a live trap is wanted. Correct DOMAIN.md §10 item 8 and the
     `gaps` profile entry ("contaminants") to say that only "contaminated" and "contamination" fire.

## Checked and right

- **Register.**
  - Every clean title block settles number, revision, title and date under §4.3. Labels used: `Ref`,
    `Report no`, `Proposal no`, `Drawing number`, `Revision` 0, A, 1, B, `Report date`, and `Date` in
    long form.
  - `# headings` match `Title:`.
  - The d04 email gives the first `Subject` ("RE: …") and the latest date (4 March).
  - A-1100 Rev B supersedes Rev A in the key, and the notes record that the app cannot link them today.
- **Base profile.** The other 70-odd keys each have a read-kind source whose sentence fires the right
  question in the Go run: flags, determinants, lot, address, LGA, zone, client and budget. The
  16,000 against 16,012 m² site area is a planned conflict that includes the true value (FINDINGS.md
  precedent). The declared traps fire as stated: floodlights, bore logs, excavated to a depth,
  "lodge", panel, public exhibition and the d02 CDC sentence.
- **Fees and invoices.**
  - Every table adds up. d07 base: $88,000 inc GST = $80,000 ex. d07 flood: $103,400 = $94,000 ex,
    with the flood phase $15,400 = $14,000 ex. The post-novation option is $49,500 = $45,000 ex.
  - `gst_inc_normalised` is a real flag (`read.go:68`).
  - d19's 15% GST gives `gst_rate`, not `arithmetic` (`ledger.go:305-313`).
  - All nine ABNs pass the checksum.
  - Invoice amounts match the proposals' stage fees.
- **Variants.**
  - CDC changes only `hdr.cond.planning`. Its evidence is verbatim. d09, d13 and d18 go with the town
    planner.
  - Flood changes only `det.flood_hazard` and `det.flood_planning_level`. The three evidence sentences
    are verbatim from the shape. The d07 flood phase follows from the trigger's civil RFP item.
    30.7 + 0.5 = 31.2 m AHD agrees across d02, d13 and d14.
- **Realism.**
  - Erskine Park sits in Penrith LGA, zone E4 General Industrial, with Endeavour Energy, Bringelly
    Shale, the Sydney Western City Planning Panel, and staff delegation under the regional threshold.
  - $14.8 m is about $1,850/m² of GFA including 6,000 m² of hardstand.
  - Architect $240k to D&C tender (1.6%), structural $76k, civil $80k, geotech $24k, planner $38k and
    traffic $19k suit an 8,000 m² D&C warehouse.
  - The dates run in order, from the February brief to the 30 June DA target.
  - A PM would recognise the set.
