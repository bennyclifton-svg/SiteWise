# Determinants and tables report

Status: draft, 29 September 2026. Lead reconciliation owns this file and the
central determinant list. Cluster-specific technical coverage is in each
cluster report; shared NCC/AS coverage is recorded there rather than claimed
as a separate exhaustive primary-standard review.

## Coverage

| Area | Result |
|---|---|
| Core project facts | Completed extraction questions/options; added stated edition, use, work type, compliance pathway, compartment volume, patient-care scope and top-storey classification |
| Cluster proposals | Folded into the central file with stable IDs and original seed anchors |
| Unknown evidence | Explicit three-way boolean choices; missing/conflicting evidence stays unknown |
| Physical scope | Building/part/storey/compartment/element provenance required; no substitution of area or height meanings |
| Structural derivations | Unsupported derivations changed to stated evidence rather than inventing code tables |
| NCC primary access | Adopted 2022 Volume One C2, C3 and C4 HTML readable |
| Verified tables | Baseline construction type and compartment area/volume |
| Pending work | Primary Australian Standard tables, state applicability and scoped exceptions; pending rules cannot compute |

## Reconciled contradictions

See [VERIFICATION.md](VERIFICATION.md) for the construction-type table,
compartment-limit conflict, false generic sprinkler downgrade and penetration
clause correction. Each verified record links the instrument itself.

Wind region subdivisions, earthquake category descriptions and exposure
thresholds were not resolved by guessing. Literal extraction preserves the
stated evidence pending authoritative definitions. Numeric identifiers such
as a classification label are distinct from an asserted numeric design limit.

## Jev contract

Code builds candidate criteria and copies selected values. Jev does not
calculate a determinant or decide a numerical threshold. Multi-choice facts
require one option question per option in a single request, not an invented
multi-answer Choice primitive. Classification descriptions remain draft aids;
questions extract the stated classification rather than assigning a legal class.

References: [primitives](https://docs.typesafe.ai/primitives),
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[fan-out](https://docs.typesafe.ai/patterns/fan-out),
[limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
[confidence](https://docs.typesafe.ai/confidence).

## Remaining boundaries

No threshold calibration or runtime compiler has been implemented. A checked
table remains an owner-unreviewed draft; its exceptions require separate
implementation. Project edition and jurisdiction are evidence, not inferred
from today's date. A clean checker is not certification of a building.
