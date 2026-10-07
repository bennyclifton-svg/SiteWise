# WP-K3 asbestos source check

Date: 6 October 2026. Status: partial research; no instrument verification.

Outcome: prevent the illustrative pre-2004 consequence from being promoted
to a legal requirement on the strength of a superficially matching date.
This is Lane A knowledge research, confined to
`cq.pre-2004-fabric-hazardous-materials-survey`. No runtime, schema, predicate,
source hash, or dependency changed. Existing filing budgets remain p50
1,000 ms / p90 2,000 ms; no timing claim follows from this documentation.

## Evidence read

[SafeWork NSW's asbestos registers and management plans fact sheet](https://www.safework.nsw.gov.au/resource-library/asbestos-publications/asbestos-registers-and-management-plans-fact-sheet)
was successfully read. Its register exemption combines a building constructed
after 31 December 2003 with no identified asbestos and none likely to be present.
This is asbestos-specific workplace guidance about a register. It does not
establish this catalogue record's general hazardous-materials survey trigger.

The official search index located the NSW Work Health and Safety Regulation
2025 and its asbestos provisions. Direct access to both the HTML instrument and
[the indexed dated PDF](https://legislation.nsw.gov.au/view/whole/pdf/inforce/2025-11-05/sl-2025-0440)
returned HTTP 403. The fact sheet's legislation link also failed. Search
snippets, the Commonwealth instrument, and regulator guidance were not
substituted for reading the NSW instrument. Current NSW text and amendment
currency therefore remain unverified in this check.

## Catalogue decision

Preserve `status: draft`, `clause_verified: false`, and empty `governed_by`.
The existing predicate covers alter, replace, upgrade, repair and remove,
combined only with a building year below 2004. It does not encode jurisdiction,
workplace applicability, asbestos presence, work extent, or statutory exceptions.
Its proposal also covers hazardous materials generally. A matching year alone
cannot justify marking that whole consequence verified.

Before a statutory rule is authored, read the applicable NSW instrument and
separately resolve register duties, demolition/refurbishment duties, residential
premises, minor maintenance, and unknown construction dates. Preserve the
distinction between a year-level fact and any exact statutory date boundary.
Determine whether the agreed predicate vocabulary can express applicability;
do not silently extend it or infer missing facts. Owner review remains separate
from primary-source verification. Investigation acceptance still requires the
unresolved explicit physical-target semantics; this research supplies none.

This check does not complete WP-K3. It changes no executable knowledge and
does not justify another performance run. The existing failing full-fixture
latency evidence remains in force.
