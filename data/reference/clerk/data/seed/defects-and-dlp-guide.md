---
tier: topic
seed_type: construction
loaded_by: task-loaded
applies_to_roles: [owner-builder, architect-pm, builder, d-and-c]
applies_to_archetypes: [new-dwelling, renovation, multi-dwelling, ancillary]
applies_to_classes: [residential]
applies_to_work_types: [new, refurb, extend]
state_default: NSW
topics: [defects, dlp, warranty]
summary: "Defects and DLP posture: DLP mechanics after practical completion, common residential defect types, defect tracking, and role-aware statutory warranty obligations (HOW/HBCF). Surfaces issues and routes them to the right professional; never assesses warranty liability."
doctrine_anchors: [§evidence-discipline, §seed-consultation-discipline, §escalation-triggers]
agents_anchors: [§1, §2, §8]
---

# Defects, DLP, and warranty guide

This seed gives SiteWise its residential defects and Defects Liability Period (DLP) posture. Load it when the task signal is defects walk, DLP, practical completion, PC checklist, defect rectification, HOW claim, HBCF claim, statutory warranty, owner-builder warranty obligations, or builder warranty.

Authority stack reminder (per `../AGENTS.md §1`): the executed head contract, the PC certificate, the HOW/HBCF certificate, and any relevant Home Building Act notices are the source of truth for DLP obligations. Where evidence is missing, label the point as Assumption per `../00-doctrine/doctrine.md §evidence-discipline`. Warranty disputes and HOW claims require legal or licensed advice — the agent surfaces issues and routes to the appropriate professional; it does not give legal opinions or assess warranty liability.

## 1. DLP overview

The **Defects Liability Period (DLP)** is a contractual period after Practical Completion during which the builder has the right and obligation to return to the site and rectify defects. The DLP is a contract term, not a statutory right — confirm the period and mechanics in the executed contract.

**Common DLP periods for residential:**
- HIA Lump Sum contracts: typically 13 weeks from the date of PC (confirm in Particulars).
- Master Builders contracts: typically 52 weeks; confirm Particulars.
- Owner-drafted or developer contracts: may specify different periods — read the contract.

**DLP is not the same as statutory warranty.** The DLP is a short contractual period. Statutory warranties under the *Home Building Act 1989* (NSW) run for longer periods and survive the end of the DLP, the sale of the property, and (for major defects) the insolvency of the builder.

**Statutory warranty periods under the Home Building Act 1989 (NSW) s. 18B (current as at mid-2025 — confirm current legislation):**

| Defect type | Warranty period | Start date |
|---|---|---|
| Major defects (structural element or waterproofing) | 6 years | Date of completion (OC or practical completion) |
| Other defects (non-major) | 2 years | Date of completion |

A **major defect** under the HBA is a defect in a major element of a building (structural element, fire safety system, or waterproofing) that is attributable to defective design, defective or faulty workmanship, or defective materials, and that causes or is likely to cause the inability of the building to be inhabited or used, or the destruction or threat to the collapse of the whole or part of the building.

**When DLP starts:** The DLP runs from the date of Practical Completion (PC). If the contract uses a PC certificate issued by the superintendent or certifier, that certificate date is the DLP commencement. If no superintendent or certifier exists (common in HIA owner-builder or simple lump-sum contracts), PC is the date the works are complete and the owner takes possession.

**DLP end date** should be recorded in the project register and programme as a milestone — defect notices must be issued before the DLP expires to preserve contractual rights.

## 2. Common residential defect types

### 2.1 Efflorescence

**What it is:** White or grey crystalline salt deposits forming on masonry, brickwork, concrete, or render surfaces. Most common on lower courses of brickwork, retaining walls, and concrete block work exposed to moisture.

**Common cause:** Soluble salts migrate to the surface as moisture moves through masonry and evaporates. Primary causes: rising damp from inadequate DPC (damp-proof course), water ingress at sills or copings, inadequate drainage at masonry base, or wet-season moisture movement in new construction.

**Rectification:** Dry brushing and proprietary efflorescence cleaner for superficial deposits. For recurring efflorescence, the cause must be identified and eliminated — not just the symptom cleaned. Investigate: DPC continuity at brick base, flashings and sill details, ground levels and drainage. If rising damp is the source, a damp-proof injection or drainage remediation may be required.

**Professional routing:** Recurring efflorescence on new construction warrants a waterproofing or building defects consultant's assessment before specifying rectification. The agent surfaces the issue; the licensed inspector or waterproofing contractor determines the cause.

### 2.2 Cracks — hairline vs structural

**Hairline cracks (cosmetic):** Fine surface cracks in render, plaster, paint, or plasterboard joints. Width typically less than 1 mm. Common in new construction as the building moves and materials dry and shrink. Generally cosmetic, not structural.

**Indicative of structural movement (refer to engineer):** Cracks wider than 2–3 mm; stepped diagonal cracks in brickwork following mortar joints; cracks running diagonally from door or window corners; cracks that are wider at one end than the other (indicating differential settlement); cracks accompanied by sticking doors or windows; cracks through structural elements (beams, columns, slabs).

**The agent must not assess whether a crack is structural.** When a crack exhibits any of the structural indicators above, the agent's response is: "This crack pattern requires assessment by a structural engineer before rectification proceeds."

**Rectification of cosmetic cracks:** Flexible filler, prime, and repaint. For render cracks, raking out and repointing with a compatible render, followed by painting. Hairline cracks in new plasterboard typically warrant a revisit at DLP expiry, by which time initial shrinkage is complete.

**Rectification of structural cracks:** Determined by the structural engineer after diagnosis of cause. May involve underpinning, drainage remediation, subfloor treatment, or structural repair. The agent does not prescribe.

### 2.3 Balcony waterproofing failure

**What it is:** Water ingress through balcony or deck substrate, typically manifesting as staining on the soffit below, bubbling or delamination of tiles, wet substrate at the balcony-to-wall junction, or corrosion of embedded framing or steel.

**Common causes:** Inadequate or deteriorated waterproofing membrane; membrane not turned up the wall to the required height (minimum 100 mm above finished floor level per NCC); inadequate or missing falls to outlet (minimum 1:80 for paved surfaces per AS 3740); puncture of membrane during tiling or installation of fixtures; inadequate perimeter detailing at door thresholds; failure of expansion joints.

**Consequence if undetected:** Moisture infiltration can degrade structural framing (timber or steel), corrode reinforcement in concrete slabs, cause mould, and result in major defect classification — triggering the 6-year statutory warranty period.

**Rectification:** Strip tiles, investigate membrane integrity, reapply waterproofing membrane to AS 3740 standard, restore falls, reinstate tiles. Where structural framing has been wet, structural assessment before reinstatement.

**Professional routing:** Waterproofing failure to a balcony is a major defect risk. Any rectification beyond a very localised repair should be directed through a waterproofing specialist. Confirm with the project certifier whether a compliance certificate is required after rectification.

### 2.4 Tile drumming (debonded tiles)

**What it is:** Hollow or drum-like sound when tiles are tapped, indicating the tile adhesive bond has failed and a void exists between the tile and substrate. The tile is still in place but unbonded — risk of cracking, displacement, or injury from loose tiles underfoot or overhead.

**Common causes:** Inadequate adhesive coverage (bed coverage less than 95% required by AS 3958.1); substrate movement (deflection-sensitive substrates not fully restrained); incorrect adhesive type for substrate or tile size; adhesive skinning before tile placement; wet substrate at time of installation; thermal expansion without adequate movement joints.

**Rectification:** Individual tiles: carefully remove, investigate adhesive coverage, clean substrate and tile back, re-bed with correct adhesive and coverage. Widespread drumming: strip the tile field, inspect substrate, address substrate movement or deflection, relay tiles with movement joints per AS 3958.1. The agent does not prescribe minimum coverage figures without confirming the current standard version.

**Professional routing:** For floor tiles in wet areas (showers, ensuites), removal and reinstatement involves the waterproofing layer — a waterproofing specialist must be involved.

### 2.5 Paint defects

**What it is:** Visible defects in paint finish including blistering, peeling, roller marks, lap marks, colour variation (patchy appearance), sagging or runs, micro-cracking (alligatoring), and poor opacity.

**Common causes:**
- **Blistering/peeling:** Paint applied over damp substrate; incompatible primer or sealer; paint over contaminants (oil, silicone residue, chalky surface).
- **Roller marks / lap marks:** Inconsistent rolling technique, paint drying too quickly in hot conditions, rolling over dried edges.
- **Colour variation:** Insufficient coats; inconsistent paint application; using touch-up paint from a different batch.
- **Micro-cracking (alligatoring):** Hard topcoat over flexible substrate; oil-based topcoat over oil-based undercoat not fully cured.

**Rectification:** Blistering and peeling — strip back to sound surface, identify and treat the cause (moisture, contamination), reprime, repaint. Roller marks and lap marks — sand back and repaint affected area with consistent technique; on large areas, full recoat is more reliable than spot repairs. Alligatoring — strip back to bare surface, apply flexible primer, repaint.

**Warranty note:** Internal paint on new construction is typically a 12-month cosmetic warranty item (confirm contract terms). External paint is more variable — check the paint manufacturer's warranty and the contract clause. The builder's obligation to rectify cosmetic paint defects is time-limited by the DLP; statutory warranty does not generally extend to cosmetic paint except where it is evidence of structural or waterproofing failure.

## 3. Defect tracking

**Every defect identified during the DLP or at the PC walk must be logged as a register row.** Use the `register-row-draft` atomic skill for every defect entry.

**Register type:** `Defects` (ID convention `Df-<seq>`, status vocabulary: `identified` → `notified` → `in-rectification` → `closed` / `disputed`)

**Output paths:**
- During construction (pre-PC defects): `07-construction/11-defects/`
- During DLP (post-PC defects): `09-handover-dlp/`

**Required row fields per §register-discipline:**

| Field | Example |
|---|---|
| ID | `Df-003` |
| Description | Hollow-sounding tiles to ensuite floor (drumming evident on tap test). Area approximately 0.8 m². |
| Owner | Builder |
| Status | `notified` |
| Due date | `2026-06-15` |
| Source | PC walk site visit 2026-06-01; photo Df-003.jpg |
| Next action | Builder to investigate tile adhesive coverage and provide rectification plan by 2026-06-15. |

**Additional register columns for defects rows** (per `register-row-draft` §register-type-specific columns): `Trade`, `Location`, `Severity`, `Date identified`, `Date notified`, `Date closed`, `Photo reference`.

**Severity guidance (for the agent to apply):**
- `Critical` — structural, waterproofing, fire/life-safety, or access issue that prevents PC or safe occupation. Rectify before PC.
- `Significant` — material defect affecting habitability or BASIX compliance. Rectify within DLP period.
- `Minor` — cosmetic or finish defect. Rectify within DLP period.

The agent does not downgrade a potential structural or waterproofing defect to `Minor` without a professional assessment. When in doubt, flag as `Significant` and note "professional assessment required before severity classification confirmed."

**Defect notice timing:** The owner (or owner's representative) must issue a written defect notice to the builder before the DLP expires. The agent should flag DLP expiry as a programme milestone and prompt for a defects review at 80% of the DLP period — earlier than the last week.

## 4. Role-aware warranty obligations

### 4.1 Owner-builder

An owner-builder who later sells the property has statutory obligations to future buyers under the *Home Building Act 1989* (NSW).

**Disclosure obligations:** If the property is sold within 7 years and 6 months of the owner-builder permit issue date, the owner-builder must disclose the owner-builder permit on the contract of sale and provide a home warranty insurance certificate (for works over the statutory threshold — confirm current threshold with NSW Fair Trading).

**Statutory warranties pass to the buyer:** The s. 18B warranties (major defects: 6 years; other defects: 2 years) run with the land. A buyer who discovers major defects within the warranty period can pursue the owner-builder under the Act.

**The agent's role:** When a project has `user_role: owner-builder` and a sale is contemplated, surface this obligation proactively. Do not assess the owner-builder's warranty exposure — route to a construction lawyer or conveyancer familiar with HBA obligations.

**Home warranty insurance for owner-builders:** Owner-builders are generally required to obtain owner-builder home warranty insurance for works over the threshold before starting — confirm current requirements with NSW Fair Trading. This is not the same as HOW/HBCF (which is builder-obtained insurance); it is a separate product for owner-builder projects.

### 4.2 Builder — HOW claim mechanics and warranty on sale

**HOW / HBCF background:** The Home Owner Warranty (HOW) scheme, administered through icare under the *Home Building Compensation Fund* (HBCF), is mandatory insurance for licensed residential builders in NSW doing work over the statutory threshold. The builder obtains the certificate before commencing work; the certificate is issued to the owner.

**When can the owner make an HOW claim?** The owner (or a subsequent buyer) can make a claim against the HOW/HBCF insurer only if the builder:
- is insolvent (in liquidation, administration, or bankrupt);
- has died;
- has disappeared (cannot be located after reasonable attempts); or
- has had their contractor licence suspended or cancelled and cannot be contacted.

**HOW is not first resort:** The owner cannot make a HOW claim simply because the builder refuses to rectify. The owner must first pursue the builder directly (and through the tribunal or court if necessary). HOW is the insurer of last resort — it pays when the builder cannot pay.

**Builder's own statutory warranty obligation:** The builder has a direct obligation to the owner under HBA s. 18B for the warranty periods regardless of HOW. If the builder is solvent and reachable, the owner's claim is against the builder, not the insurer.

**When the home is sold during DLP:** The statutory warranties under s. 18B transfer to the new owner. The new owner has the same warranty rights against the original builder as the first owner had. The DLP (a contractual right to require rectification) runs between the original contracting parties — it does not automatically transfer. If the property is sold during the DLP period, the new owner's remedy is under the statutory warranty, not the original contract.

**Builder's rectification obligation:** During the DLP, the builder must rectify defects notified in writing within the DLP period. The owner should issue a written notice (email or letter) identifying each defect and requesting rectification within a reasonable time. The agent assists in preparing this notice through `markdown-draft-for-review` but does not draft legal correspondence directly — for formal demand letters, route to the owner's solicitor.

**Agent trigger for escalation:** If the builder does not respond to a defect notice within 14 days, or does not complete rectification before the DLP expires, flag as `§escalation-triggers` breach and surface options: NSW Fair Trading complaint, NSW Civil and Administrative Tribunal (NCAT) application, or legal advice.

## 5. State callouts

**NSW default:** The DLP, statutory warranty, and HOW/HBCF framework described above is the NSW regime under the *Home Building Act 1989* (NSW) and HBCF. All guidance above is NSW-specific unless noted.

**VIC:** Victoria uses **Domestic Building Insurance (DBI)**, administered by the Victorian Managed Insurance Authority (VMIA) or approved insurers under the *Domestic Building Contracts Act 1995* (Vic). The trigger conditions for claiming (insolvency, death, disappearance) are similar but the insurer and claim process differ from NSW HBCF. Statutory warranty periods under the *Domestic Building Contracts Act* differ from HBA — confirm current VIC periods with a VIC construction lawyer. The DLP period and mechanics are governed by the executed contract (typically a Master Builders Victoria or HIA Vic contract).

**QLD:** Queensland uses the **QBCC Home Warranty Insurance** scheme, administered by the Queensland Building and Construction Commission (QBCC). The QBCC scheme provides structural defect cover for 6 years and 6 months, non-structural for 12 months (confirm current periods with QBCC). The claim pathway differs from NSW HOW.

**Other states and territories:** Each jurisdiction has its own home warranty insurance scheme, statutory warranty periods, and licensing authority. For any non-NSW project, the agent should flag: "Confirm applicable home warranty insurer, statutory warranty periods, and licensing authority for the project state before advising the owner."

## Coverage depth

| Topic | Depth |
|---|---|
| DLP overview and contract mechanics | Deep |
| Statutory warranty periods (NSW HBA) | Deep |
| Common residential defect types and rectification | Moderate |
| Defect tracking schema and register-row-draft usage | Deep |
| Owner-builder warranty obligations to buyers | Moderate |
| Builder HOW/HBCF claim mechanics | Moderate |
| Builder statutory warranty on sale during DLP | Moderate |
| VIC / QLD home warranty schemes | Shallow (signpost + state callouts) |
| Legal demand and tribunal processes | Minimal (surface and route to solicitor) |

## Low-confidence flags

- **Statutory warranty periods:** The HBA 1989 s. 18B periods cited (6 years major defects, 2 years other defects) reflect the law as at mid-2025. Confirm current periods with the NSW Fair Trading website or a construction lawyer before advising on any live warranty claim.
- **HOW/HBCF claim thresholds and eligibility:** The eligible trigger conditions and minimum work value for HOW coverage are subject to legislative change. Confirm current thresholds with icare / HBCF before advising an owner on whether HOW coverage applies to a specific project.
- **Owner-builder home warranty insurance threshold:** The value threshold triggering the obligation to obtain owner-builder warranty insurance is subject to change. Confirm current threshold with NSW Fair Trading.
- **Major defect definition:** What constitutes a "major defect" and a "major element" under the HBA is a legal question that may require tribunal or court interpretation for a specific defect. The agent applies the broad definition from s. 18B but does not make legal determinations — route to a construction lawyer for any contested case.
- **Rectification standards and costs:** This seed describes rectification approaches in general terms. Actual rectification scope, method, and cost should be determined by a licensed tradesperson or building inspector who has assessed the specific defect.
- **Crack assessment:** The guidance on hairline vs structural cracks is indicative only. The agent must always refer suspected structural cracks to a structural engineer — it does not classify crack severity as structural or non-structural on the evidence available in correspondence or photos.

## See also

- `../02-skills/systems/handover-pc-system.md` — PC checklist orchestrating the defects walk, HOW handover, and DLP tracking
- `../02-skills/atomic/register-row-draft.md` — used for every defect row (Df-seq schema)
- `../01-seed/sustainability-energy-guide.md` — BASIX final certificate as a PC hold point (cross-reference at PC walk)
- `../01-seed/contract-administration-guide.md` — written notice obligations, defect notice clause references, final claim mechanics
- `../01-seed/program-scheduling-guide.md` — DLP end date as a programme milestone; defect review at 80% of DLP period
- `../00-doctrine/doctrine.md §evidence-discipline` — labelling defect descriptions and photos as Fact, Assumption, or Judgement
- `../00-doctrine/doctrine.md §register-discipline` — the schema every defect row must satisfy
- `../00-doctrine/doctrine.md §escalation-triggers` — when a defect notice goes unanswered or the DLP expires with open items
- `../AGENTS.md §1` — authority stack; executed contract and PC certificate beat seed guidance
- `../AGENTS.md §8` — state callouts; confirm VIC/QLD home warranty insurer and warranty periods before advising
