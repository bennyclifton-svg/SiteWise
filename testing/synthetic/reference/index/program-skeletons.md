# Program skeletons, stages, durations and calendars
Generated on 28 October 2026 from the knowledge files on `main` (8a6a6c8 plus the working tree) by a read-only script. Regenerate rather than hand-edit. Back to [the index](../KNOWLEDGE_INDEX.md).

## Delivery stages (`knowledge/works/stages.yaml`)

| id | label | parent | novation side |
|---|---|---|---|
| `investigation` | Investigation | — | — |
| `design` | Design | — | — |
| `concept_design` | Concept design | `design` | pre |
| `approval_documentation` | Approval documentation | `design` | pre |
| `construction_documentation` | Construction documentation | `design` | post |
| `approvals` | Authorities and approvals | — | — |
| `procurement` | Procurement | — | — |
| `construction` | Construction | — | — |
| `construction_stage_inspections` | Construction-stage inspections | `construction` | post |
| `completion` | Completion | — | — |
| `defects` | Defects liability | — | — |
| `other` | Other stage | — | — |

## Skeletons (`knowledge/program/skeletons.yaml`)

`Catalog.MatchSkeleton` (`internal/knowledge/program.go`): a skeleton applies only when its `applies_when` is **true** (unknown never selects). Among those, the one with most predicate terms wins; equal terms is a tie, noted on the plan (`CheckSkeletonTie`). No match → `sk.generic`. `det: work_type` reads the applied `hdr.work_type` values; `cond:` reads `hdr.cond.<key>` (`readProgramGenInput`, `internal/store/program_prepare.go`).

| id | applies_when | terms | stages in order |
|---|---|---|---|
| `sk.generic` | `fallback` | 0 | investigation → concept_design → approval_documentation → construction_documentation → approvals → procurement → construction → completion → defects |
| `sk.new-traditional-da` | `{all:[{det:work_type,eq:new},{cond:procurement_route,eq:traditional},{cond:planning,any_of:[da,planning_permit]}]}` | 3 | investigation → concept_design → approval_documentation → approvals → construction_documentation → procurement → construction → completion → defects |
| `sk.new-traditional-cdc` | `{all:[{det:work_type,eq:new},{cond:procurement_route,eq:traditional},{cond:planning,any_of:[cdc,exempt]}]}` | 3 | investigation → concept_design → approval_documentation → approvals → procurement → construction → completion → defects |
| `sk.new-dc-da` | `{all:[{det:work_type,eq:new},{cond:procurement_route,eq:design_construct},{cond:planning,any_of:[da,planning_permit]}]}` | 3 | investigation → concept_design → approval_documentation → approvals → procurement → construction_documentation → construction → completion → defects |
| `sk.new-eci-da` | `{all:[{det:work_type,eq:new},{cond:procurement_route,any_of:[eci,managing_contractor]},{cond:planning,any_of:[da,planning_permit]}]}` | 3 | investigation → concept_design → procurement → approval_documentation → approvals → construction_documentation → construction → completion → defects |
| `sk.refurb-traditional-da` | `{all:[{det:work_type,eq:refurb},{cond:procurement_route,eq:traditional},{cond:planning,any_of:[da,planning_permit]}]}` | 3 | investigation → concept_design → approval_documentation → approvals → construction_documentation → procurement → construction → completion → defects |
| `sk.refurb-traditional-cdc` | `{all:[{det:work_type,eq:refurb},{cond:procurement_route,eq:traditional},{cond:planning,any_of:[cdc,exempt]}]}` | 3 | investigation → concept_design → construction_documentation → approvals → procurement → construction → completion → defects |
| `sk.extend-traditional` | `{all:[{det:work_type,eq:extend},{cond:procurement_route,eq:traditional},{cond:planning,any_of:[da,cdc,exempt,planning_permit]}]}` | 3 | investigation → concept_design → approval_documentation → approvals → construction_documentation → procurement → construction → completion → defects |
| `sk.ssd-traditional` | `{all:[{det:work_type,any_of:[new,extend]},{cond:procurement_route,eq:traditional},{cond:planning,eq:ssd}]}` | 3 | investigation → concept_design → approval_documentation → approvals → construction_documentation → procurement → construction → completion → defects |
| `sk.ssd-dc` | `{all:[{det:work_type,any_of:[new,extend]},{cond:procurement_route,any_of:[design_construct,eci,managing_contractor]},{cond:planning,eq:ssd}]}` | 3 | investigation → concept_design → approval_documentation → approvals → procurement → construction_documentation → construction → completion → defects |

### Which skeleton a work type, route and planning pathway reach

| work type | route | planning | skeleton |
|---|---|---|---|
| new | traditional | da | `sk.new-traditional-da` |
| new | traditional | cdc | `sk.new-traditional-cdc` |
| new | traditional | exempt | `sk.new-traditional-cdc` |
| new | traditional | ssd | `sk.ssd-traditional` |
| new | traditional | planning_permit | `sk.new-traditional-da` |
| new | traditional | (unknown) | `sk.generic` |
| new | design_construct | da | `sk.new-dc-da` |
| new | design_construct | cdc | `sk.generic` |
| new | design_construct | ssd | `sk.ssd-dc` |
| new | design_construct | planning_permit | `sk.new-dc-da` |
| new | design_construct | (unknown) | `sk.generic` |
| new | eci | da | `sk.new-eci-da` |
| new | eci | ssd | `sk.ssd-dc` |
| new | eci | planning_permit | `sk.new-eci-da` |
| new | managing_contractor | da | `sk.new-eci-da` |
| new | managing_contractor | ssd | `sk.ssd-dc` |
| new | managing_contractor | planning_permit | `sk.new-eci-da` |
| new | (unknown) | da | `sk.generic` |
| new | (unknown) | cdc | `sk.generic` |
| new | (unknown) | (unknown) | `sk.generic` |
| extend | traditional | da | `sk.extend-traditional` |
| extend | traditional | cdc | `sk.extend-traditional` |
| extend | traditional | exempt | `sk.extend-traditional` |
| extend | traditional | ssd | `sk.ssd-traditional` |
| extend | traditional | planning_permit | `sk.extend-traditional` |
| extend | traditional | (unknown) | `sk.generic` |
| extend | design_construct | da | `sk.generic` |
| extend | design_construct | cdc | `sk.generic` |
| extend | design_construct | ssd | `sk.ssd-dc` |
| extend | design_construct | (unknown) | `sk.generic` |
| extend | eci | ssd | `sk.ssd-dc` |
| extend | managing_contractor | ssd | `sk.ssd-dc` |
| extend | (unknown) | da | `sk.generic` |
| extend | (unknown) | cdc | `sk.generic` |
| extend | (unknown) | (unknown) | `sk.generic` |
| refurb | traditional | da | `sk.refurb-traditional-da` |
| refurb | traditional | cdc | `sk.refurb-traditional-cdc` |
| refurb | traditional | exempt | `sk.refurb-traditional-cdc` |
| refurb | traditional | planning_permit | `sk.refurb-traditional-da` |
| refurb | traditional | (unknown) | `sk.generic` |
| refurb | design_construct | da | `sk.generic` |
| refurb | design_construct | cdc | `sk.generic` |
| refurb | design_construct | (unknown) | `sk.generic` |
| refurb | (unknown) | da | `sk.generic` |
| refurb | (unknown) | cdc | `sk.generic` |
| refurb | (unknown) | (unknown) | `sk.generic` |
| remediation | traditional | da | `sk.generic` |
| remediation | traditional | cdc | `sk.generic` |
| remediation | traditional | (unknown) | `sk.generic` |
| remediation | design_construct | da | `sk.generic` |
| remediation | design_construct | cdc | `sk.generic` |
| remediation | design_construct | (unknown) | `sk.generic` |
| remediation | (unknown) | da | `sk.generic` |
| remediation | (unknown) | cdc | `sk.generic` |
| remediation | (unknown) | (unknown) | `sk.generic` |
| advisory | traditional | da | `sk.generic` |
| advisory | traditional | cdc | `sk.generic` |
| advisory | traditional | (unknown) | `sk.generic` |
| advisory | design_construct | da | `sk.generic` |
| advisory | design_construct | cdc | `sk.generic` |
| advisory | design_construct | (unknown) | `sk.generic` |
| advisory | (unknown) | da | `sk.generic` |
| advisory | (unknown) | cdc | `sk.generic` |
| advisory | (unknown) | (unknown) | `sk.generic` |

Rows omitted above all fall back to `sk.generic`. In particular: every **remediation** and **advisory** project; refurb under anything but traditional; extend under anything but traditional (SSD aside); any project whose route or pathway is unknown.

## Durations (`knowledge/program/durations.yaml`, file status `reviewed`)

301 rows, all `status: reviewed` (owner accepted 24 October 2026; file notes). Bases: fixed 207, statutory 4, per_unit 64, lead_time 20, floor_cycle 6. Statutory rows keep `verified: false`. Assumption: `plan.gross_floor_area` = 250 m2 until the Profile states it. `Generate` uses a reviewed row only; an activity with no usable duration takes `du.activity-default`, an uncovered investigation `du.investigate-default` (`internal/program/generate.go` constants).

| id | basis | days | low / typical / high | driver | applies_when |
|---|---|---|---|---|---|
| `du.concrete-cure` | fixed | calendar | 7 / 7 / 14 | — | — |
| `du.closein-1a` | fixed | working | 8 / 12 / 20 | — | — |
| `du.closein-2` | fixed | working | 15 / 30 / 50 | — | — |
| `du.closein-fitout` | fixed | working | 5 / 10 / 15 | — | — |
| `du.closein-9b` | fixed | working | 10 / 20 / 35 | — | — |
| `du.closein-7b` | fixed | working | 5 / 8 / 15 | — | — |
| `du.closein-reno` | fixed | working | 5 / 8 / 12 | — | — |
| `du.dc-site-overlap` | fixed | working | 10 / 15 / 25 | — | — |
| `du.eci-overlap` | fixed | working | 10 / 15 / 20 | — | — |
| `du.da-assessment-period` | statutory | calendar | 40 / 60 / 90 | — | — |
| `du.da-exhibition-period` | statutory | calendar | 14 / 14 / 28 | — | — |
| `du.cdc-determination-period` | statutory | calendar | 10 / 20 / 20 | — | — |
| `du.ssd-exhibition-period` | statutory | calendar | 14 / 28 / 28 | — | — |
| `du.da-program-allowance` | fixed | calendar | 40 / 90 / 180 | — | — |
| `du.ssd-program-allowance` | fixed | calendar | 90 / 180 / 365 | — | — |
| `du.defects-liability` | fixed | calendar | 90 / 365 / 730 | — | — |
| `du.temporary-power` | fixed | working | 1 / 3 / 5 | — | — |
| `du.erosion-control` | fixed | working | 1 / 2 / 5 | — | — |
| `du.tree-protection` | fixed | working | 1 / 2 / 5 | — | — |
| `du.occupied-protection` | fixed | working | 2 / 5 / 10 | — | — |
| `du.demolition-1a` | fixed | working | 5 / 10 / 20 | — | — |
| `du.strip-out-1a` | fixed | working | 3 / 8 / 15 | — | — |
| `du.earthworks-1a` | fixed | working | 3 / 5 / 10 | — | — |
| `du.underpinning-1a` | fixed | working | 5 / 15 / 30 | — | — |
| `du.footings-excavate-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.footings-pour-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.termite-1a` | fixed | working | 1 / 1 / 2 | — | — |
| `du.damp-1a` | fixed | working | 1 / 1 / 2 | — | — |
| `du.stormwater-1a` | fixed | working | 2 / 4 / 8 | — | — |
| `du.flood-1a` | fixed | working | 3 / 8 / 15 | — | — |
| `du.slab-setout-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.under-slab-services-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.slab-pour-1a` | per_unit | working | 0.02 / 0.03 / 0.05 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.frame-walls-1a` | per_unit | working | 0.02 / 0.03 / 0.05 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.frame-roof-1a` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.masonry-1a` | per_unit | working | 0.03 / 0.05 / 0.08 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.roof-sark-1a` | fixed | working | 1 / 2 / 3 | — | — |
| `du.roof-cover-1a` | fixed | working | 3 / 5 / 10 | — | — |
| `du.external-walls-1a` | per_unit | working | 0.02 / 0.03 / 0.05 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.insulation-1a` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.windows-lead-1a` | lead_time | working | 10 / 15 / 25 | — | — |
| `du.windows-install-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.bushfire-1a` | fixed | working | 3 / 8 / 15 | — | — |
| `du.roof-drainage-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.partitions-1a` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sheet-1a` | per_unit | working | 0.03 / 0.04 / 0.06 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ceilings-1a` | per_unit | working | 0.02 / 0.03 / 0.04 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.wet-area-1a` | fixed | working | 2 / 4 / 8 | — | — |
| `du.internal-doors-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.sanitary-rough-1a` | fixed | working | 3 / 5 / 8 | — | — |
| `du.sanitary-fit-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.cold-rough-1a` | fixed | working | 2 / 3 / 5 | — | — |
| `du.cold-fit-1a` | fixed | working | 1 / 2 / 3 | — | — |
| `du.hot-rough-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.hot-fit-1a` | fixed | working | 1 / 1 / 2 | — | — |
| `du.gas-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.electrical-rough-1a` | per_unit | working | 0.02 / 0.03 / 0.05 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.electrical-fit-1a` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.lighting-1a` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.switchboard-1a` | fixed | working | 1 / 2 / 4 | — | — |
| `du.smoke-alarms-1a` | fixed | working | 1 / 1 / 2 | — | — |
| `du.ac-1a` | fixed | working | 2 / 4 / 8 | — | — |
| `du.tiling-1a` | fixed | working | 3 / 6 / 12 | — | — |
| `du.joinery-lead-1a` | lead_time | working | 10 / 15 / 25 | — | — |
| `du.joinery-install-1a` | fixed | working | 3 / 5 / 10 | — | — |
| `du.painting-1a` | per_unit | working | 0.03 / 0.05 / 0.08 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.floors-1a` | per_unit | working | 0.01 / 0.02 / 0.04 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.landscape-1a` | fixed | working | 3 / 8 / 15 | — | — |
| `du.linings-1a-reno` | fixed | working | 5 / 10 / 20 | — | — |
| `du.wet-area-1a-reno` | fixed | working | 2 / 4 / 8 | — | — |
| `du.hydraulic-rough-reno` | fixed | working | 3 / 5 / 10 | — | — |
| `du.hydraulic-fit-reno` | fixed | working | 2 / 3 / 5 | — | — |
| `du.water-1a-reno` | fixed | working | 2 / 3 / 6 | — | — |
| `du.electrical-rough-reno` | fixed | working | 3 / 5 / 10 | — | — |
| `du.electrical-fit-reno` | fixed | working | 2 / 3 / 5 | — | — |
| `du.joinery-1a-reno` | fixed | working | 10 / 15 / 25 | — | — |
| `du.painting-1a-reno` | fixed | working | 4 / 8 / 15 | — | — |
| `du.floors-1a-reno` | fixed | working | 3 / 5 / 10 | — | — |
| `du.demolition-2` | fixed | working | 10 / 20 / 40 | — | — |
| `du.earthworks-2` | fixed | working | 5 / 15 / 30 | — | — |
| `du.underpinning-2` | fixed | working | 10 / 20 / 40 | — | — |
| `du.piles-platform-2` | fixed | working | 3 / 5 / 10 | — | — |
| `du.piles-install-2` | fixed | working | 10 / 20 / 40 | — | — |
| `du.basement-shoring-2` | fixed | working | 15 / 25 / 45 | — | — |
| `du.basement-excavate-2` | fixed | working | 10 / 20 / 40 | — | — |
| `du.basement-anchors-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.basement-slab-2` | fixed | working | 8 / 15 / 25 | — | — |
| `du.footings-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.slab-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.lift-pits-2` | fixed | working | 5 / 10 / 15 | — | — |
| `du.lift-shafts-2` | floor_cycle | working | 1 / 2 / 3 | det.rise_in_storeys (storeys) | — |
| `du.floor-cycle-2` | floor_cycle | working | 6 / 9 / 12 | det.rise_in_storeys (storeys) | — |
| `du.structure-roof-2` | fixed | working | 8 / 12 / 20 | — | — |
| `du.precast-lead-2` | lead_time | working | 20 / 40 / 70 | — | — |
| `du.precast-install-2` | floor_cycle | working | 1 / 2 / 3 | det.rise_in_storeys (storeys) | — |
| `du.stairs-2` | floor_cycle | working | 1 / 2 / 4 | det.rise_in_storeys (storeys) | — |
| `du.roof-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.facade-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.windows-lead-2` | lead_time | working | 15 / 25 / 40 | — | — |
| `du.windows-install-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.balcony-waterproof-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.insulation-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.roof-drainage-2` | fixed | working | 3 / 5 / 10 | — | — |
| `du.linings-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ceilings-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.wet-area-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.internal-doors-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-rough-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-fit-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-commission-2` | fixed | working | 3 / 5 / 10 | — | — |
| `du.cold-rough-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.cold-fit-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.hot-plant-lead-2` | lead_time | working | 15 / 25 / 40 | — | — |
| `du.hot-fit-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.electrical-rough-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.electrical-fit-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.lighting-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sprinklers-rough-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sprinklers-fit-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.hydrants-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.detection-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.smoke-alarms-2` | fixed | working | 2 / 4 / 8 | — | — |
| `du.ac-plant-lead-2` | lead_time | working | 40 / 80 / 140 | — | — |
| `du.ac-install-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ac-commission-2` | fixed | working | 5 / 10 / 15 | — | — |
| `du.carpark-vent-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.comms-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.tiling-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.joinery-lead-2` | lead_time | working | 15 / 25 / 40 | — | — |
| `du.joinery-install-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.painting-2` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.floors-2` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.landscape-2` | fixed | working | 5 / 15 / 30 | — | — |
| `du.stormwater-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.flood-2` | fixed | working | 5 / 15 / 30 | — | — |
| `du.demolition-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.earthworks-7b` | fixed | working | 5 / 10 / 25 | — | — |
| `du.footings-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.steel-lead-7b` | lead_time | working | 20 / 40 / 70 | — | — |
| `du.steel-erect-7b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.roof-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.cladding-7b` | fixed | working | 8 / 15 / 30 | — | — |
| `du.slab-prepare-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.slab-pour-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.vehicle-doors-lead-7b` | lead_time | working | 10 / 20 / 40 | — | — |
| `du.vehicle-doors-install-7b` | fixed | working | 2 / 5 / 10 | — | — |
| `du.sanitary-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.electrical-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.lighting-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.sprinklers-7b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.hydrants-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.stormwater-7b` | fixed | working | 3 / 8 / 15 | — | — |
| `du.landscape-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.painting-7b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.demolition-9b` | fixed | working | 5 / 15 / 30 | — | — |
| `du.earthworks-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.piles-9b` | fixed | working | 5 / 15 / 30 | — | — |
| `du.footings-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.slab-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.structure-9b` | floor_cycle | working | 8 / 12 / 18 | det.rise_in_storeys (storeys) | — |
| `du.steel-lead-9b` | lead_time | working | 20 / 40 / 60 | — | — |
| `du.steel-erect-9b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.roof-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.external-walls-9b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.windows-lead-9b` | lead_time | working | 15 / 30 / 50 | — | — |
| `du.windows-install-9b` | fixed | working | 5 / 10 / 15 | — | — |
| `du.bushfire-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.stairs-9b` | floor_cycle | working | 2 / 3 / 5 | det.rise_in_storeys (storeys) | — |
| `du.linings-9b` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ceilings-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.wet-area-9b` | fixed | working | 3 / 6 / 12 | — | — |
| `du.joinery-9b` | fixed | working | 10 / 20 / 35 | — | — |
| `du.painting-9b` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.floors-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-rough-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-fit-9b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.electrical-rough-9b` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.electrical-fit-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.lighting-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ac-lead-9b` | lead_time | working | 20 / 40 / 60 | — | — |
| `du.ac-install-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ac-commission-9b` | fixed | working | 3 / 5 / 10 | — | — |
| `du.sprinklers-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.detection-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.comms-9b` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.landscape-9b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.stormwater-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.strip-isolate-fitout` | fixed | working | 1 / 2 / 5 | — | — |
| `du.strip-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.makegood-fitout` | fixed | working | 2 / 5 / 10 | — | — |
| `du.structure-alterations` | fixed | working | 5 / 15 / 30 | — | — |
| `du.partitions-fitout` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.ceilings-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.wet-area-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.internal-doors-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.sanitary-rough-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.sanitary-fit-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.cold-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.hot-fitout` | fixed | working | 1 / 2 / 4 | — | — |
| `du.electrical-rough-fitout` | per_unit | working | 0.01 / 0.02 / 0.03 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.electrical-fit-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.lighting-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.mechanical-rough-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.mechanical-commission-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.ventilation-fitout` | fixed | working | 2 / 4 / 8 | — | — |
| `du.sprinklers-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.detection-fitout` | fixed | working | 3 / 5 / 10 | — | — |
| `du.comms-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.tiling-fitout` | fixed | working | 3 / 5 / 10 | — | — |
| `du.joinery-lead-fitout` | lead_time | working | 10 / 20 / 30 | — | — |
| `du.joinery-install-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.painting-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.floors-fitout` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.fire-stopping` | per_unit | working | 0.01 / 0.01 / 0.02 per unit (unit_days working) | plan.gross_floor_area (m2) | — |
| `du.gas-commercial` | fixed | working | 3 / 5 / 10 | — | — |
| `du.lift-lead` | lead_time | working | 60 / 100 / 160 | — | — |
| `du.lift-install` | fixed | working | 10 / 20 / 40 | — | — |
| `du.lift-commission` | fixed | working | 5 / 10 / 15 | — | — |
| `du.switchboard-lead-2` | lead_time | working | 20 / 40 / 70 | — | — |
| `du.switchboard-install-2` | fixed | working | 5 / 10 / 20 | — | — |
| `du.switchboard-lead-major` | lead_time | working | 15 / 30 / 60 | — | — |
| `du.switchboard-install-major` | fixed | working | 3 / 8 / 15 | — | — |
| `du.utility-apply-1a` | fixed | working | 5 / 10 / 15 | — | — |
| `du.utility-lead-1a` | lead_time | working | 10 / 20 / 40 | — | — |
| `du.utility-apply-major` | fixed | working | 10 / 15 / 25 | — | — |
| `du.utility-lead-major` | lead_time | working | 20 / 40 / 80 | — | — |
| `du.services-apply-1a` | fixed | working | 5 / 10 / 20 | — | — |
| `du.services-lead-1a` | lead_time | working | 10 / 20 / 40 | — | — |
| `du.services-apply-major` | fixed | working | 10 / 20 / 30 | — | — |
| `du.services-lead-major` | lead_time | working | 20 / 40 / 80 | — | — |
| `du.concept-design-1a` | fixed | working | 10 / 20 / 40 | — | — |
| `du.concept-design-2` | fixed | working | 20 / 40 / 80 | — | — |
| `du.concept-design-fitout` | fixed | working | 5 / 10 / 20 | — | — |
| `du.concept-design-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.concept-design-9b` | fixed | working | 15 / 30 / 50 | — | — |
| `du.design-development-1a` | fixed | working | 10 / 20 / 30 | — | — |
| `du.design-development-2` | fixed | working | 20 / 45 / 80 | — | — |
| `du.design-development-fitout` | fixed | working | 5 / 15 / 25 | — | — |
| `du.design-development-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.design-development-9b` | fixed | working | 15 / 30 / 45 | — | — |
| `du.approval-docs-1a` | fixed | working | 15 / 25 / 40 | — | — |
| `du.approval-docs-2` | fixed | working | 30 / 50 / 80 | — | — |
| `du.approval-docs-7b` | fixed | working | 10 / 20 / 30 | — | — |
| `du.approval-docs-9b` | fixed | working | 20 / 40 / 60 | — | — |
| `du.fitout-documentation` | fixed | working | 10 / 20 / 40 | — | — |
| `du.construction-docs-1a` | fixed | working | 15 / 30 / 50 | — | — |
| `du.construction-docs-2` | fixed | working | 40 / 70 / 110 | — | — |
| `du.construction-docs-7b` | fixed | working | 10 / 20 / 40 | — | — |
| `du.construction-docs-9b` | fixed | working | 25 / 40 / 70 | — | — |
| `du.measure-existing` | fixed | working | 3 / 5 / 10 | — | — |
| `du.hazmat-survey` | fixed | working | 5 / 10 / 20 | — | — |
| `du.trace-services` | fixed | working | 3 / 5 / 10 | — | — |
| `du.da-lodgement` | fixed | working | 5 / 10 / 20 | — | — |
| `du.cdc-prepare` | fixed | working | 10 / 20 / 40 | — | — |
| `du.exempt-confirmation` | fixed | working | 1 / 5 / 10 | — | — |
| `du.ssd-sears` | fixed | working | 15 / 30 / 45 | — | — |
| `du.ssd-eis` | fixed | working | 40 / 80 / 140 | — | — |
| `du.construction-certificate` | fixed | working | 10 / 15 / 30 | — | — |
| `du.heritage-referral` | fixed | working | 10 / 20 / 40 | — | — |
| `du.tender-docs-1a` | fixed | working | 10 / 15 / 25 | — | — |
| `du.tender-docs-2` | fixed | working | 15 / 25 / 40 | — | — |
| `du.tender-docs-fitout` | fixed | working | 5 / 10 / 20 | — | — |
| `du.tender-docs-7b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.tender-docs-9b` | fixed | working | 10 / 20 / 30 | — | — |
| `du.tender-period-1a` | fixed | working | 10 / 15 / 25 | — | — |
| `du.tender-period-2` | fixed | working | 15 / 20 / 30 | — | — |
| `du.tender-period-fitout` | fixed | working | 10 / 15 / 20 | — | — |
| `du.tender-period-7b` | fixed | working | 10 / 15 / 25 | — | — |
| `du.tender-period-9b` | fixed | working | 15 / 20 / 30 | — | — |
| `du.tender-evaluation-1a` | fixed | working | 5 / 10 / 15 | — | — |
| `du.tender-evaluation-2` | fixed | working | 10 / 15 / 25 | — | — |
| `du.tender-evaluation-fitout` | fixed | working | 5 / 10 / 15 | — | — |
| `du.tender-evaluation-7b` | fixed | working | 5 / 10 / 15 | — | — |
| `du.tender-evaluation-9b` | fixed | working | 10 / 15 / 20 | — | — |
| `du.contract-award` | fixed | working | 5 / 10 / 20 | — | — |
| `du.novation` | fixed | working | 10 / 15 / 30 | — | — |
| `du.site-establishment-1a` | fixed | working | 2 / 5 / 10 | — | — |
| `du.site-establishment-2` | fixed | working | 10 / 15 / 25 | — | — |
| `du.site-establishment-fitout` | fixed | working | 3 / 5 / 10 | — | — |
| `du.site-establishment-7b` | fixed | working | 3 / 8 / 15 | — | — |
| `du.site-establishment-9b` | fixed | working | 5 / 10 / 20 | — | — |
| `du.practical-completion` | fixed | working | 1 / 3 / 5 | — | — |
| `du.occupation-certificate` | fixed | working | 5 / 10 / 20 | — | — |
| `du.handover` | fixed | working | 2 / 5 / 10 | — | — |
| `du.defects-inspection` | fixed | working | 1 / 2 / 5 | — | — |
| `du.final-defects` | fixed | working | 1 / 2 / 5 | — | — |
| `du.brief-confirm-1a` | fixed | working | 3 / 5 / 10 | — | — |
| `du.brief-confirm` | fixed | working | 5 / 10 / 20 | — | — |
| `du.client-review-1a` | fixed | working | 5 / 10 / 15 | — | — |
| `du.client-review` | fixed | working | 10 / 15 / 25 | — | — |
| `du.client-signoff-1a` | fixed | working | 3 / 5 / 10 | — | — |
| `du.client-signoff` | fixed | working | 5 / 10 / 15 | — | — |
| `du.investigate-geotech` | fixed | working | 10 / 20 / 30 | — | — |
| `du.investigate-survey` | fixed | working | 5 / 10 / 15 | — | — |
| `du.investigate-contamination` | fixed | working | 10 / 15 / 30 | — | — |
| `du.investigate-acid-sulfate` | fixed | working | 10 / 15 / 25 | — | — |
| `du.investigate-trees` | fixed | working | 5 / 10 / 15 | — | — |
| `du.investigate-flooding` | fixed | working | 10 / 20 / 40 | — | — |
| `du.investigate-services` | fixed | working | 5 / 10 / 15 | — | — |
| `du.investigate-default` | fixed | working | 5 / 10 / 20 | — | — |
| `du.activity-default` | fixed | working | 3 / 5 / 10 | — | — |

## Calendars (`knowledge/program/calendars.yaml`)

`calendars: []`. No state has public holidays; a program uses weekends only and says so (file header).
