# K1 partition and air-distribution relationship

6 October 2026. Design/source assessment before implementation. Draft only.

K1 explicitly asks for missing fit-out partition relationships to air
distribution and zoning. Current services interfaces include base-building
capacity but no partition-to-air-distribution edge. The mechanical seed's
`### Brief and design basis` and `### Coordination` sections cover room use,
zoning and services routes without specifying this relationship directly.

Primary publisher source checked:
[CIBSE Journal CPD Module 227, November 2023](https://www.cibsejournal.com/cpd/modules/2023-12-fcui/).
It explains that subdividing an open layout changes heating/cooling demands
and can make existing fan-coil locations or sizes unsuitable. It also describes
duct/diffuser adaptation to newly partitioned rooms. This supports a layout
coordination relationship, not a universal equipment-replacement instruction.
The article discusses particular equipment solutions and draws on an industry
white paper; those product choices are not adopted here. It is not an
Australian regulatory instrument or proof of ventilation compliance.

Proposed bounded implementation: a draft `shares_space` edge from the existing
internal-wall/partition system to mechanical air distribution. New/altered
partitions should produce the existing coordination obligation when the
receiving system exists or is unknown, while absent/excluded targets and
unrelated work remain suppressed under the current evaluator rules. Preserve
part identity and existing-system reason provenance. Validate the exact system
identifier, direction and target behaviour before adding the record.

Use the existing consequence kind and label; do not invent a control-link
test, force an equipment replacement, or assert that partitions are HVAC
controllers. Smoke control, thermal-performance and new-to-existing tie-ins
are separate K1 relationships and are not closed by this edge.

Lane A knowledge change supporting time-to-decision and physical coordination.
No new service/dependency. Existing profile edit/rebuild budgets remain
50/150 and 100/300 ms; their intermittent failure remains open. Before calling
the edge implemented, check the schema's evidence-question requirements,
strict validation, positive/negative evaluator behaviour and question routing
impact. Any added Jev question must follow the TypeSafe guidance and remain
within the existing single fan-out; no live/private call is authorized by this
design note. Owner review and K6 usefulness remain outstanding.

## Implementation preflight: taxonomy decision required

The exact current identifier is `interiors.walls-linings`. Its definition
includes non-load-bearing partition framing but also plasterboard, internal
sheet lining, joints and other lining work. An action on this combined system
does not establish a changed room layout. The planned edge would therefore
raise the same coordination obligation for lining-only alterations. The
current interface consequence operates on system/action, not a proven
partition-layout predicate. Inferring layout changes from titles would add
unsupported semantic heuristics.

No edge has been added. The owner has been asked whether to split partition
work from wall linings, accept coordination for all new/alter work on the
combined system, or defer the edge. A split would require taxonomy, candidate
selection and compatibility analysis before implementation; it is not a
one-line endpoint substitution. This correction supersedes the assumption
above that an existing partition-specific system identifier was available.

The checker accepts an empty `resolved_when` list, but that alone does not
make an imprecisely triggered proposal acceptable. No new evidence question,
runtime call or dependency was introduced during preflight.
