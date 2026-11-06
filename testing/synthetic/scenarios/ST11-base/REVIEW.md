# ST11 review (T11 dual-occupancy-townhouses)

Scenarios: ST11-base, ST11-cdc-low-rise-code, ST11-three-plus-townhouses.

## Verdict: FIX

Checker: `python testing/synthetic/tools/validate_expected.py testing/synthetic/scenarios/ST11-*` gives `ok` for
all three. It passes the three-plus variant only because a key was held at a value the variant's own documents
contradict (finding 1).

What this review covered: all seven brief steps (steps 3 and 7 are written up below the findings). Earlier pass: the checker, the base and CDC profile diffs against the base, the three-plus variant's
profile, notes and new fee proposals (d20 adds up: $1,800 + $3,200 + $1,500 = $6,500 ex GST), and the author's two
shape comments. The CDC variant moves only `hdr.cond.planning` (da -> cdc); the other diffs are the d16 SEE dropping
out of `sources`, which is right because the CDC path has no SEE.

## Findings

### 1. ST11-three-plus-townhouses `expected/profile.json`: `hdr.scale.avg_dwelling_size_sqm` is wrong (serious)

- What is wrong: the key says 230 with `sources: []`. The variant's manifest summary and its d14/d15 `why` say
  "Townhouses 1 to 4 at 115 m2 each". With 4 dwellings in the same 460 m2 GFA, the true average is 115. A key that
  contradicts the documents is wrong, even if the app never reads this field. "The scorer should ignore it" is not
  a fix; any later reader or scorer change would take 230 as the truth.
- Ruling on the shape: **change the shape**, as was done for T05 (whose trigger lists
  `hdr.scale.added_gfa_sqm` and `hdr.scale.gfa_sqm` beside `hdr.scale.storeys`). The trigger changes the dwelling
  count while GFA stays the same, so the average moves with it. That is one trigger, not two.
- Exact changes (fix round, by the author):
  1. `testing/synthetic/types/T11-dual-occupancy-townhouses/shape.yaml`, trigger `three-plus-townhouses`:
     `profile: [hdr.scale.dwellings]` -> `profile: [hdr.scale.dwellings, hdr.scale.avg_dwelling_size_sqm]`.
  2. ST11-three-plus-townhouses `manifest.json`, d14 and d15: keep the base's area-schedule wording with the new
     numbers: "Area schedule: Townhouse 1 115 m2, Townhouse 2 115 m2, Townhouse 3 115 m2, Townhouse 4 115 m2;
     average dwelling size 115 m2". Add the `carries` entry for the average as the base has it. The base states the
     average on d14/d15, so dropping it was a second, unasked change.
  3. ST11-three-plus-townhouses `expected/profile.json`: `hdr.scale.avg_dwelling_size_sqm` ->
     `{"value": 115, "sources": ["d14", "d15"], "read_by_design": false}`. It is false because the field has no
     triggers, as in the base.
  4. ST11-three-plus-townhouses `expected/notes.md`: remove the "held at the base's 230" paragraph under "Missed by
     design". Instead say the average comes only from the drawings and the field has no triggers, as the base says.
     Remove the first shape comment once the shape is changed.

### 2. Shape trigger `three-plus-townhouses`: the "R3 Medium Density Residential zone" evidence (shape)

- Ruling: **do not add `proj.zone`** to the trigger. Rezoning is a separate fact from the number of dwellings. If
  the zone moved, the authority path, the SEE and the LEP checks would all move too, so one variant would change two
  triggers. The author's choice to keep R2 is consistent with the rest of the set: the variant puts multi dwelling
  housing under the Housing SEPP low-rise housing provisions and marks that `verified: false`.
- Exact change (fix round, by the author): in the shape, delete `"R3 Medium Density Residential zone"` from the
  trigger's `evidence`, so that no future author writes a zone sentence that moves `proj.zone` without listing it.
  Leave `profile` without `proj.zone`. The variant's keys stay as they are: `proj.zone` accepts R2 / R2 Low Density
  Residential, sources as the base.
- Then remove the `proj.zone` half of the variant's shape comment in `notes.md`.

### 3. ST11-three-plus-townhouses `expected/notes.md`: budget and fee plausibility (minor, note only)

- Holding the $1,650,000 budget and every base fee for 4 kitchens and 4 sets of services is light, but it is
  possible for two-bedroom townhouses at 460 m2 (about $3,600/m2). The checker requires those figures to stay equal
  to the base. No change; the existing note is enough.

### 4. ST11-* `expected/document_register.json` d12: lifecycle is stricter than for the matching reports (minor)

- What is wrong: d12 (geotechnical report) expects `lifecycle: "design"` only. d03 (survey report) and d11 (hazmat
  report) are the same kind: consultant reports with label lines, read before design. Both accept
  `["design", "consultant"]`. Lifecycle is a Jev pick for a report. No rule settles it: d12 has no
  `Drawing number` line and no lifecycle caption. A strict key would mark a sensible `consultant` pick as wrong in
  one place and accept it in the other two.
- Exact change: in all three scenarios, d12 `lifecycle` -> `{"accept": ["design", "consultant"]}`.

## Step 3: register keys (all three scenarios, DISCOVERY.md §4.2 and §4.3)

Every document was checked against its `title_block`. No other register key is wrong.

- Label-line documents (d01, d03, d05, d06, d08 to d13, d16, and d20 and d21 in the three-plus variant): every
  number label is one the rules accept (`Ref`, `Proposal no/ref`, `Quote no/number`, `Report no`,
  `Document number`). Every number matches the number pattern, every date is in an accepted format, and there is
  one `#` heading identical to `Title:`. The rules settle number, revision, title and date as the key expects.
  d11 and d12 `Revision: 0` gives rev 0, and d13 gives rev A.
- d14 and d15 (drawings): `Drawing number`, `Revision`, `Title` and `Date` label lines with no `#` heading, so
  the title has one candidate. The `Drawing number` line makes lifecycle `design` by rule. The key expects d15 to
  supersede d14, which the app cannot write; the notes already say so (§4.5).
- d02 and d07 (Markdown table title blocks): the rules get no number or date. `Certificate no` and `Our ref` are
  not labels in any case. The `#` heading gives the title. The base notes ("Missed by design") already say this,
  and the variants carry it over ("As the base"). That meets the brief.
- d04 (email): the title is the first `Subject` ("RE: Timing for the build"). The date is the latest message,
  3 March 2026, which matches the key. Number and revision are null, as they must be.
- d17 to d19 are invoices. They take the invoice path, and `invoices.json` gives the identity. d19's table header
  is the deliberate messy case.
- Variants: in the CDC variant, d10 and d16 are removed and nothing else in the register changes. In the three-plus
  variant, only the d01 title changes ("townhouses"; its `#` still matches `Title:`), and d20 and d21 are added
  with clean label lines.

## Step 7: realism

- Documents and dates: an s10.7 certificate (9 Feb), a survey (23 Feb), and fee proposals from 9 to 19 March.
  Then the hazmat report (13 Apr), the geotech report (20 Apr), the stage 1 invoices (17 to 30 Apr), drawings
  Rev A then Rev B in May, the design statement, and the SEE on 27 May. That is a DA lodgement in early June. The
  tenancy ends on 31 Oct 2026 and the target start is 1 Feb 2027. A PM would recognise this as a real Parramatta
  dual-occupancy job. The February start is tight, given a council determination and a tender over Christmas,
  but it is possible.
- Fees (ex GST): the architect's fee to tender is $85,000, about 5.2% of $1.65M. Structural is $9,800,
  civil/OSD $13,000 ($14,300 inc GST divided by 1.1), geotech $4,100, hazmat $2,600 and the town planner $11,000.
  All are plausible for Sydney in 2026. The invoices match the stage amounts, and the d18 `arithmetic` flag is
  deliberate. The base budget, $1.65M for 460 m2 (about $3,590/m2 including demolition), is plausible for mid-range.
- The three-plus variant's traffic fee ($6,500) and waste fee ($2,500) are plausible. Its fee and budget
  lightness is finding 3. Multi dwelling housing in R2 relies on the Housing SEPP low-rise provisions near a
  town centre or station. That is plausible for Carlingford and correctly left `verified: false`.
- CDC variant: the subdivision by its own CDC is marked `verified: false`, which is right.

## After the fix

Re-run the checker on `ST11-*`. The three-plus variant must still pass with exactly
`hdr.scale.dwellings` (2 -> 4) and `hdr.scale.avg_dwelling_size_sqm` (230 -> 115) differing from the base.
