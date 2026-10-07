# K1 interface obligation audit — 7 October 2026

Original scope: architecture/schema K1 row and fit-out research list; requirements
NW-REQ-158/159/306. Knowledge-only audit of the current interface endpoints,
consequence directions and local seed evidence. No new edge, question, determinant
or runtime rule added in this pass. Existing fixture tests are identified below,
not claimed rerun here. Catalogue inventory cannot establish complete source or
owner review.

| Required topic | Existing implementation/evidence | Remaining action |
| --- | --- | --- |
| Base-building plant to tenancy air conditioning | `if.tenant-fitout-base-building-hvac` is now `supplies`, from central plant/ventilation to air distribution/conditioning. `ic.supplies-investigate-supply` touches `to` for new/upgrade/alter and labels capacity as supply, not structural capacity. `internal/works/tenant_fitout_test.go` covers this actual edge, part semantics and existing targets. Mechanical seed `### Existing buildings and live environments` supports checking existing capacity rather than assuming operating plant is adequate. | Treat this specific D-29/AT-15 issue as implemented, subject to final integrated tests. Do not describe the current edge as still typed `loads` from historical requirements cells. |
| Tenant heat loads, outdoor air and operating hours | Same edge summary/question explicitly covers these and existing failure modes describe capacity and extended-hours problems. | This is evidence/coordination coverage, not a load calculation. Separate lighting/equipment/occupant loads are not automatically calculated. No need for a duplicate generic load edge. |
| Partitions and ceilings changing distribution/zoning/return air | Services-to-ceilings `shares_space` edge covers physical congestion/set-out, not room-layout change. `interiors.walls-linings` mixes partition framing with lining-only work. | Explicit changed-layout fact or carefully designed taxonomy split required; broad lining work cannot prove changed zoning. Existing partition preflight source/design note retained. Reversing a shares-space edge would not establish airflow effects. |
| Partition/layout effects on smoke control | Existing fire-detection→fire-mode-air controls edge covers control commands, not partition geometry. | No faithful layout-change trigger or smoke-zone geometry currently encoded. Keep an exact gap; don't label partitions as controllers. |
| Partition/layout effects on thermal performance | Existing energy-geometry edge connects thermal performance to windows/doors and external shading; envelope thermal bridge boundary covers fabric structure/insulation. | Internal zoning/partition changes are not captured by those relationships. Missing explicit changed-layout/use/load facts. Do not infer performance changes from generic wall-lining alteration. |
| Supplementary cooling on condenser water | Broad central-plant supply and capacity coverage exists, but central plant does not prove a condenser-water service available to a tenancy. | No separately evidenced condenser-water tenant connection target found in inspected catalogue. Requires source-specific plant/service semantics; do not fabricate capacity or availability. |
| BMS and fire-mode integration | `if.bms-fire-mode-priority` controls from detection to controls-BMS; `if.fire-detection-controls-smoke-plant` controls from detection to fire-mode-air systems. Existing controls consequence tests affected either-side links. | Preserve existing edges; project scripts/override hierarchy and actual integrated testing still require evidence. |
| Glazing films/blinds, solar gain and thermal stress | External shading and windows energy relationships exist. Inspected mechanical/commercial seeds provide generic facade/energy coordination, not a precise film/blind thermal-stress trigger. | No adequate source/element-specific trigger found in this bounded local sweep. Research a primary technical source before adding a film/blind obligation; do not substitute a generic shading edge for glass thermal stress. |
| Energy provisions for changed work | Envelope energy-services/geometry relationships and existing rules are present. | Applicable edition, jurisdiction, work extent and performance pathway cannot be concluded from a changed window alone. K3 remains separate, with unverified rule numbers unchanged. |
| Base-building ratings and lease obligations | Mechanical class-2/mixed-use overlay calls for tenancy/base-building/strata coordination. | Rating commitment, lease terms and landlord nominated contractor are contract/project facts, not universal physical consequences. No adequate project-independent obligation can be inferred from current input semantics. |
| Landlord approvals/nominated contractors | Existing boundary and capacity concerns can prompt review. | Need explicit tenancy/ownership and actual lease/approval conditions. Do not declare landlord approval granted or appoint a contractor. |
| New-to-existing ties | `if.envelope-old-new-weather-barrier` explicitly connects flashings/seals with walls/roof/external waterproofing, source renovation `## Waterproofing and old-to-new tie-ins`; generic penetrations and supplies interfaces cover related work. | This implements weather-barrier tie-in coverage, not every structural, drainage, movement, acoustic or services tie-in. Same-system old/new instances and junction location need explicit modelling before reliable universal tie-in generation. |

Bounded safe additions were considered. None of the remaining named physical
relationships can be faithfully added by a one-line endpoint change without
making unsupported layout, plant-type, tenancy or junction assumptions. Parent
has retained the partition/jurisdiction schema choices as explicit remaining
work. The audit therefore documents gaps rather than inventing edges to satisfy
a count. K1 as a whole remains partially implemented; current HVAC and weather
barrier coverage should not be lost in a blanket “not started” status.

## Later bounded layout implementation

The lead subsequently chose and implemented explicit per-work unknown/yes/no
layout context; see [layout implementation and tests](2026-10-07-k1-layout-input-proposal.md).
This resolves the user-input/same-item predicate blocker for a bounded draft
room-layout air-distribution review CQ. It does not yet create all missing graph
edges or establish smoke-control/thermal performance. Other topic-specific
plant, tenancy and junction input gaps above remain applicable. The earlier
“no new determinant/runtime rule” statement describes this audit pass, not the
subsequent implementation.
