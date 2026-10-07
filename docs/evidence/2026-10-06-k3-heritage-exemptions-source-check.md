# K3 NSW heritage exemptions: primary-source check

6 October 2026. Partial legal-source research; no executable rule or catalogue
status changed. No new dependency or runtime call.

Primary instrument read: [NSW Government Gazette 461, 7 November 2025,
notice NSWGG-2025-461-2](https://gazette.nsw.gov.au/gazette/2025/11/2025-11_461-gazette.pdf).
The order begins on PDF page 3, activity standards on page 7, and general
conditions on pages 22–23. The order revokes the specified 2022 order and
takes effect on publication. It grants conditional exemptions for described
activities, rather than a blanket exemption for repairs or alterations.
Its scope includes State Heritage Register items and interim heritage orders;
underwater items are excluded. Activity-specific standards and general
conditions apply together. The utility/services activity excludes several
categories dealt with separately, including fire/security systems. The order
does not supply approvals under other legislation. These are findings about
the text read, not a determination that a particular project qualifies.

[Heritage NSW's current standard-exemptions page](https://www.environment.nsw.gov.au/topics/heritage/works-approvals-state-heritage-register-items/standard-exemptions)
links this Gazette and explains the need to check listing, significance,
standard and site-specific exemptions, and other approvals. The
[current Heritage Act page](https://legislation.nsw.gov.au/view/html/inforce/current/act-1977-136)
returned HTTP 403. The complete current Act and amendment history were not
verified; no Act clause is marked verified on this evidence alone.

## Consequence for SiteWise modelling

The current `heritage_status` choices distinguish conservation areas, local
items and the State register. They do not independently encode an interim
heritage order, affected fabric significance, detailed exemption conditions,
site-specific exemptions, owner authority or underwater status. Therefore a
simple predicate combining heritage status and work action cannot establish
approval or exemption. A record should request review of the applicable
pathway, not mark an approval obtained or allocate statutory responsibility.

The existing delivery unforeseen record's contract text allocating heritage
approvals to the owner is not established by the source read. Contractual
risk allocation and statutory approval pathways are different questions.
No contract, consequence, determinant or detector has been rewritten here.

Next implementation requires a scoped approval-review proposal with explicit
unknowns and jurisdiction, or richer approved input semantics if the product
is to evaluate exemption eligibility. Keep any resulting knowledge draft;
do not infer missing facts from a broad heritage flag. This completes a source
check for one K3 topic, not K3, the legal model or owner content review.
