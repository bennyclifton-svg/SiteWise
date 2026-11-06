# ST17-base notes

Tideway Apartments, 27 Tideway Crescent, Meadowbank (City of Ryde): a fully occupied 8-storey, 56-apartment Class 2
strata building constructed in 2011, not sprinkler protected. Under a council fire safety order (compliance date
17 December 2027) the owners corporation, funded by special levies, removes PE-core ACP and an EPS render system and
replaces them with a ventilated rainscreen of solid aluminium panels: DA, then one AS 4000 lump sum contract;
construction budget $6,850,000 ex GST. 20 documents in 4 waves; every T17 trigger denied or absent.

## Where this departs from brief 03, and why

- **Fee proposals.** The brief's new-build defaults are architect, structural, civil and geotechnical. Civil and
  geotechnical engineers have no role in a recladding (no excavation, ground works or stormwater change; DOMAIN.md §4
  "Never needed for the base"), so they are replaced by the shape's remediation team: project manager (d06), fire
  safety engineer (d08), facade engineer (d09), quantity surveyor (d11, GST-inclusive) and town planner (d12, dropped
  on `row:consultant.town_planner`, which the app suggests on the DA path). Architect (d07) and structural (d10)
  stay. The BCA consultant and principal certifier are appointed later, so they have no proposal; the shape still
  expects their packages.
- **Wave 1** is what an owners corporation holds before design: the fire safety order (d01), the strata manager's
  brief (d02), the s10.7 certificate (d03), the 2025 cladding audit (d04) and a committee email (d05). There is no
  site survey on a recladding job.
- **Wave 3**: options report with the QS order of cost (d13), fire engineering risk assessment (d14), slab-edge and
  pull-out report (d15), SEE (d16) and elevation A-201 at Rev A and Rev B (d17, d18).

## Traps

- d01 `balcony-storage-not-special-hazard`: "no storage or barbecues on balconies" near "fire safety measures"
  matches the fire_special_hazards storage pattern; the flag stays no (d14).
- d02 `remediation-not-contamination`: no contamination scope or environmental consultant.
- d02 `rfp-cued-as-proposal`: "submit a fee proposal" must not open a submission.
- d03 `cdc-eligible-land`: land eligibility for complying development; planning stays da.
- d03 `acid-sulfate-class-not-ncc`: 'acid sulfate soils class 5' names a class that is not an NCC building class;
  det.ncc_class stays ["2"].
- d04 `bedroom-window`: "bedroom" must not imply internal doors.
- d04 `basement-exists-not-excavated`: "wholly residential above the basement car park"; flag.basement stays no.
- d06 `fee-stages-not-staged-works`: "Stage 1" to "Stage 5" fee stages; flag.staged stays no.
- d07 `stage-5-fee-stage`: "Stage 5 includes the architectural regulated designs"; flag.staged stays no.
- d13 `basement-scaffold`: scaffold base plates on the podium over the basement; flag.basement stays no.
- d13 `basix-does-not-apply`: "BASIX" fires energy-rating implied scope; no thermal-performance item or BASIX line.
- d14 `type-a-not-a-class`: "Type A construction" beside "NCC building classification: Class 2"; class stays ["2"].
- d14 `not-sprinklered-is-evidence`: det.sprinklered false; no sprinkler scope.
- d16 `exempt-repainting`: exempt repainting is not the works' pathway; planning stays da.
- d20 `total-arithmetic`: total printed $16,850.00 for $14,500 + $1,450 (`arithmetic`).
- d11 states GST-inclusive figures ($49,500 inc = $45,000 ex): `gst_inc_normalised`.

Every other trigger's negation is in a read document in the shape's wording: performance solution and fire upgrades
(d14), hazardous materials, balustrades, services and mixed-use podium (d04), concealed damage and balcony upturns
(d15), windows, boundary, swing stage, D&C, staged funding, defects claim and low-rise HBCF (d13), Project Remediate,
CDC and order-without-consent (d02).

## Missed by design

- `proj.target_construction_start` (2027-03-01): only in the d05 email (correspondence is not read).
- `proj.regulated_design` (stated in d07, d13, d15): no extraction (`details.go compileReading`); it stays unknown
  rather than derived "no", because a Class 2 part is present. Keyed `sources: []` with `stated_in`.
- `det.type_of_construction` (A): derived only, never read; blank while rule.ncc.type-of-construction and its table
  are draft (reason unreviewed). The key keeps the true value.
- `proj.lot`: d03 says "Strata Plan 87654 (Lots 1 to 56 and common property)"; `lotRe` has no range form.
- `proj.client`: documents state "The Owners - Strata Plan No. 87654"; `clientCandidates` stops at " - ", so the app
  offers only "The Owners" (probable app defect, as in ST09).
- d03, d11 and d20: Markdown-table title blocks, so number and date are missed and d03/d11 titles come from the
  heading. Keys keep the true values.
- d18 supersedes d17 (A-201 Rev B over Rev A): the app cannot link it (DISCOVERY.md §4.5).
- The fire safety order has no profile key, fragment or cost line (shape gaps); its compliance date is recorded in
  `program.json` `scenario_facts` for reference only.

## Wording chosen to meet the app's triggers

- "56 apartments", "8-storey", "constructed in 2011", "a rise in storeys of 8", "not sprinkler protected",
  "ventilated rainscreen cladding system", "AS 4000-1997 General conditions of contract", "lump sum" (no hyphen),
  "NCC building classification: Class 2". Each flag denial uses its trigger's own words.

## Shape (findings applied with the orchestrator's authority, 6 Nov)

- `proj.client` now "The Owners - Strata Plan No. 87654" (no "(fictional)").
- Year phrasing "constructed in <year>" (scenario input, hazmat evidence and negation, defects-claim negation);
  units phrasing "<n> apartments".
- `det.type_of_construction` note: derived by rule.ncc.type-of-construction, never read, blank while draft.
- `project-remediate` profile is now `[]`: the client stays the owners corporation.
