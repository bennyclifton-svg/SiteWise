# Knowledge merge — 29 September 2026

All records remain draft. Schema/reference validity, primary transcription and
owner review are separate checks. This model is not an operational compliance
engine; Jev questions still need compilation and field-specific evaluation.

## Canonical interfaces

The following extraction duplicates were consolidated. Sources, governing
rules, failure links and distinct evidence questions were retained; failure
attachments were redirected. These aliases document merge history, not extra
runtime edges.

| Retired extraction ID | Retained ID |
|---|---|
| if.fire-alarm-controls-security-release | if.fire-alarm-releases-security |
| if.fire-alarm-controls-lift-recall | if.fire-alarm-controls-lift |
| if.envelope-facade-slab-edge-fire | if.fire-facade-slab-edge |
| if.envelope-fire-acoustic-separation | if.fire-and-acoustic-separation |
| if.downlights-insulation-interface | if.envelope-insulation-downlights |
| if.waste-setout-before-membrane | if.envelope-wet-waste-before-membrane |
| if.electrical-rough-in-before-lining | if.envelope-rough-in-before-linings |
| if.hydraulic-rough-in-before-linings | if.envelope-rough-in-before-linings |
| if.services-share-ceiling-space | if.envelope-services-ceiling-space |
| if.external-levels-protect-envelope | if.envelope-external-levels-moisture-bridge |

The wet-air cluster owns `if.plant-loads-structure`; its structure duplicate was
removed during extraction. Envelope owns `rule.ncc.whole-of-home-energy`;
the wet-air duplicate was removed. Specific sprinkler obstruction, PV mounting,
roof drainage and electrical fire-water supply edges remain separate where
they describe different mechanisms or evidence requirements.

Security and lift controls now use the corresponding child-system endpoints.
Facade fire and acoustic boundaries use their specific leaf systems. Broad
top-level endpoints elsewhere are deliberate when evidence spans multiple
children; no unresolved placeholder child ID is substituted.

## Determinants and verification

Proposals were folded into the central determinant file. Empty proposal files
remain as cluster extension points. Unsupported computed structural outputs
were converted to stated-value extraction; invented earthquake categories and
unsupported exposure descriptions were removed. Overall structure height is
distinct from wall height and NCC effective height.

Boolean extraction has explicit affirmative, negative and unknown outcomes.
Code retains scope and provenance. Multi-valued extraction compiles into
independent option questions in one fan-out. No absent mention becomes a
negative project fact. Wind-region values use literal candidates while the
conflicting seed taxonomy awaits primary verification.

Two primary-verified baseline tables were written: construction type and
compartment limits. Both retain scope/exclusions and remain draft. Fire rule
clause locators and corrected numbers were updated only where the NCC primary
text was read. The generic sprinkler construction downgrade remains rejected.
See [verification](clusters/determinants/VERIFICATION.md) for the evidence.

Missing lookup tables are explicit pending derivations, which must return
unknown. No placeholder table populated from unverified seed numbers was added.
FRL matrices, detailed concessions, statutory state adoption and Australian
Standard tables remain verification work; the table list below is not the
entire future compliance engine.

## Publication checks

`tools/check_knowledge.py --strict` validates schema, provenance anchors and
resolved references, including active table references. It rejects unverified
published tables and verified claims without primary sources. It does not
certify engineering correctness or implement a rule evaluator.

Regression tests cover rejected unverified tables, pending-table contracts,
explicit unknown boolean evidence, the seed's construction-type mistakes and
the area/volume pair in the compartment table. Local Go compilation also passed.

Cluster reports contain seed coverage and residual domain gaps. Runtime Jev
evaluation, threshold calibration, org isolation and latency measurements are
implementation work in [the intake plan](../docs/plans/2026-09-29-instant-intake.md).

## Final merged inventory

| Cluster | Child systems | Rules | Interfaces | Failure modes |
|---|---:|---:|---:|---:|
| electrical-comms | 26 | 47 | 30 | 34 |
| envelope | 24 | 65 | 44 | 62 |
| fire | 28 | 30 | 21 | 23 |
| services-wet-air | 20 | 54 | 16 | 23 |
| structure | 32 | 30 | 29 | 30 |

Plus 13 top-level systems: **143 systems, 66 determinants, 226 rules, 140 interfaces and 172 failure modes**.

All six clusters have reports, all proposed determinants are folded, and all graph references resolve. The full-file inventories in the domain reports record the completed relevant seed sweeps and explicit exclusions. Specialist engineering details absent from those seeds remain domain gaps.

**2 verified baseline tables; 9 distinct pending lookup tables** (11 pending rule derivations):

- `accessible_lift_requirement`
- `basix_applicability`
- `energy_monitoring_facilities`
- `ev_charging_readiness`
- `nathers_minimum_stars`
- `roof_ceiling_min_r_value`
- `sanitary_facility_ratios`
- `sound_insulation_requirements`
- `wall_min_r_value`

Validation: standard and strict knowledge checks pass with 0 errors and 0 warnings; all 6 checker regression tests pass. Primary verification is limited to the records identified above, and all knowledge stays draft.
