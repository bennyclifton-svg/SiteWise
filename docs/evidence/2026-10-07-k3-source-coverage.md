# K3 NSW source coverage and bounded draft runtime reviews — 7 October 2026

Lane A knowledge research. Original scope: architecture/schema K3 row and
NW-REQ-163/308. This pass reads public primary government guidance; it does not
verify the complete current legal instruments or licensed Australian Standards.
The initial source pass changed no runtime records. The later corrected decision below implements six draft review consequences without new determinants, questions, numeric rules or owner-review state. The initial pass incorrectly treated jurisdiction as absent. The correction below supersedes that conclusion. Inferring NSW from an address or applying NSW rules
to every class-2 building would be incorrect.

## Primary source coverage

| Original topic | Source read and bounded conclusion | Remaining inputs/verification |
| --- | --- | --- |
| Alterations and fire-safety upgrade | [Building Commission NSW: upgrading fire safety in strata buildings](https://www.nsw.gov.au/departments-and-agencies/building-commission/current-and-future-homeowners/fire-safety-measures-strata) distinguishes orders, development/use changes and maintenance. A planning review should identify the actual approval/order pathway; routine maintenance alone does not imply wholesale upgrade to current standards. | Jurisdiction, actual order/approval, change of use, work extent and existing fire performance. Current EP&A instrument open returned an internal error; indexed historical text was not treated as current verification. |
| Essential fire-safety measures | [NSW Planning: fire safety certification](https://www.planning.nsw.gov.au/the-planning-system/buildings/fire-safety-in-buildings/fire-safety-certification) and [Building Commission: owner responsibilities](https://www.nsw.gov.au/departments-and-agencies/building-commission/industry-changes/as1851-2012/responsibilities-of-building-owners-under-as-1851-2012) distinguish the building's scheduled measures and their maintenance basis. | Actual fire safety schedule, measure-specific standards, statement applicability, responsible practitioner and current instrument. No standard intervals or frequencies authored; no licensed standard read. |
| Class 2 practitioner regime | [Building Commission: remedial work in regulated buildings](https://www.nsw.gov.au/housing-and-construction/compliance-and-regulation/professionals-working-on-regulated-buildings/design-and-building-practitioners/remedial-building-work), updated July 2026, explains that remedial class-2 work can fall within DBP obligations, with exclusions/emergency distinctions. | Jurisdiction, class incl mixed-use, building-work exclusion, urgency, affected elements and specific design/declaration obligations. Direct current DBP Act access returned an internal error. Guidance is enough to draft an applicability review, not declare regulated designs legally required for every repair. |
| Hazardous materials in older buildings | [SafeWork NSW registers fact sheet](https://www.safework.nsw.gov.au/resource-library/asbestos-publications/asbestos-registers-and-management-plans-fact-sheet) distinguishes an asbestos workplace register from a generic hazardous-materials survey. [SafeWork demolition guidance](https://www.safework.nsw.gov.au/your-industry/construction/demolition) links demolition planning to asbestos information/inspection. | Workplace/PCBU context, actual demolition/refurbishment scope, asbestos evidence, inaccessible areas and applicable exceptions. Building age alone cannot establish every survey/removal obligation. Existing illustrative pre-2004 CQ remains unverified; no broader seed claim is promoted. |
| Home-building thresholds | [Building Commission licensing categories](https://www.nsw.gov.au/business-and-economy/licences-and-credentials/building-and-trade-licences-and-registrations/categories-of-work), [NSW home-building contracts](https://www.nsw.gov.au/housing-and-construction/building-or-renovating-a-home/preparing/contracts), and [SIRA insurance obligations](https://www.sira.nsw.gov.au/resources-library/home-building-compensation-resources/publications/insurance-obligations-for-residential-building-works) show different licensing, contract and insurance applicability, including specialist-work treatment and insurance exemptions. | Jurisdiction, statutory residential/specialist work classification, contracting entity, aggregated contract/market value incl tax, exemptions and timing. Cost-plan budget/estimate is not automatically the statutory contract price. No executable dollar threshold or deposit percentage added. |
| Heritage | [Heritage NSW standard exemptions](https://www.environment.nsw.gov.au/topics/heritage/works-approvals-state-heritage-register-items/standard-exemptions) directs checking listing, significant fabric, activity and general conditions. Existing Gazette reading is recorded in the 6 October K3 heritage note. | Jurisdiction; State-register versus local-area distinction; interim order; significant fabric; site-specific conditions; authority. A broad heritage flag does not establish an exemption, approval obtained or which party bears contractual responsibility. |

## Source-pass candidates (later runtime disposition below)

These are review actions rather than legal determinations. All six candidates now have bounded runtime review CQs; richer eligibility decisions require the missing inputs below; all would remain draft with
`clause_verified: false` and no governed-by rule until instrument work supports it.

1. **NSW alteration approval-basis review**: explicit NSW + existing building +
   included alteration/replacement/upgrade; approval review to identify orders,
   proposed use and authority/certifier requirements. Do not infer an upgrade.
2. **NSW essential-measures continuity review**: explicit NSW + affected active/
   passive fire scope + project schedule evidence; hold-point review of scheduled
   performance, impairment/restoration and records. Existing general fire CQ
   should be merged or refined rather than duplicated once semantics are settled.
3. **NSW class-2 remedial-design applicability review**: explicit NSW + class-2
   part + remediation work; approval review to check DBP exclusions/emergency
   treatment and required practitioner/design/declaration steps. Never infer that
   all emergency work is exempt, or that planning exemption implies DBP exemption.
4. **NSW asbestos disturbance-information review**: explicit NSW + actual
   demolition/refurbishment disturbance; hold-point review of asbestos information,
   access limitations and applicable competent-person assessment. Replace the
   illustrative age-only legal inference only after backward-compatibility and
   source semantics are reviewed; do not impose a generic hazardous survey.
5. **NSW residential contracting applicability review**: explicit NSW + statutory
   work category + contract/market value and entity; review licensing, contract
   form and insurance evidence with actual exemptions. No automatic action from
   a project-wide forecast total.
6. **NSW heritage pathway review**: explicit NSW + confirmed relevant listing/order
   + affected fabric/work; approval review to resolve standard/site-specific
   exemption conditions and other approvals. Do not label local conservation-area
   work as State-register work.

Typed jurisdiction is already available; the initial contrary conclusion was corrected by tracing the adapter. Further inputs are needed only for stronger legal eligibility outcomes. No new document question or runtime API call follows from this note.
K3 has source coverage of all six topics and bounded runtime review coverage;
owner review and complete current-instrument verification remain outstanding.

## Corrected runtime decision — 7 October 2026

Read-only tracing found the existing `state` determinant, NSW enum, site-profile
editor and `ProposalInputs` adapter already provide explicit jurisdiction. Applied
`det.state` becomes provenance-bearing `ProposalPart.Values[state]`, then WorksEnv.
An explicit part unknown shadows project state; another part cannot supply it.
No schema or Jev question is needed. The earlier absence claim was an audit error.

Decision: implement draft review-only CQs for NSW alteration approval basis,
Class-2 remediation applicability and State-register heritage pathway using
existing typed inputs. Unknown jurisdiction remains an uncertainty-bearing
proposal under the existing three-valued evaluator; it must not become a known
NSW obligation. VIC/other known states do not match. No numeric/legal eligibility,
whole-building upgrade, approved permit or practitioner appointment is asserted.
The reviewed government guidance above supports checking applicability; all
records remain draft, unverified and without governed-by rules. Source records
reference this ledger and its direct official links. This is Lane A, existing
proposal edit/rebuild budgets 50/150 and 100/300 ms, no runtime call/dependency.

Validation: NSW/VIC/unknown and part-specific provenance regression plus authored-state inheritance/unknown-shadow/assumption rejection passed (works1.018s; profile0.768s combined with K1). No DB or organisation access path changed. Existing store actor/organisation boundary remains the input gate.

## Final three-topic runtime decision

Re-read the official fire-certification, SafeWork demolition, NSW residential
contracts and SIRA insurance guidance on7October2026. Add review-only prompts:
(1) NSW existing fire-system alterations/repair require reviewing the actual
schedule, affected measures, maintenance/certification responsibilities and
continuity arrangements; (2) NSW existing building fabric alteration/replacement/
removal requires reviewing whether materials are disturbed and available asbestos
information/access limitations, with competent assessment where needed; (3) NSW
residential-class work requires reviewing the statutory work category, actual
contract value/entity, licensing, contract and insurance applicability/exemptions.
These are preparation reviews, not a finding that an approval, register, survey,
insurance policy or upgrade is legally mandatory. NCCclass is a screening context,
not a substitute for statutory residential-work classification. No amount or
threshold is derived from the cost plan. No age cutoff or assumed hazard added.
Existing boolean/choice/work predicates suffice; no schema/question added.

Final validation: all six K3 topic families now have executable draft review CQs.
The final33-case added-family test covers known/unknown/false jurisdiction,
class, existing condition and action, excluded work and unrelated systems.
Together with the original9cases:42pass in0.906s. Strict checker reports
16CQs,0errors and2existing warnings. No legal thresholds, verified clauses,
new questions or automatic physical work were introduced.
