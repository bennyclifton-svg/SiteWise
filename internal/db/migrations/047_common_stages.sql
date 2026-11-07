-- Seven project stages (owner direction, 7 November 2026; plan
-- docs/plans/2026-11-07-common-project-stages.md section 5). Saved catalogue
-- stage ids move to the stage they now belong to. Labels the user saw, cost
-- lines, invoices and frozen baselines are untouched: they point at
-- package_stages rows, not at catalogue ids. A program keeps its rows; the
-- next Generate offers the seven stages for review.
--
-- Construction documentation is design finalisation after a novation or on a
-- contractor-led program (the contractor's documents), and design
-- development otherwise (the developed set a traditional tender prices).

UPDATE package_stages SET stage_id = CASE
  WHEN stage_id IN ('investigation','design','concept_design') THEN 'brief_concept'
  WHEN stage_id IN ('approval_documentation','approvals') THEN 'planning'
  WHEN stage_id = 'construction_documentation' AND novation_phase = 'post' THEN 'design_finalisation'
  WHEN stage_id = 'construction_documentation' THEN 'design_development'
  WHEN stage_id = 'procurement' THEN 'tender_documentation'
  ELSE 'delivery' END
WHERE stage_id IN ('investigation','design','concept_design','approval_documentation','approvals',
  'construction_documentation','procurement','construction','construction_stage_inspections','completion','defects');

CREATE FUNCTION pg_temp.common_stage(old text, skeleton text) RETURNS text LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE
    WHEN old IN ('investigation','design','concept_design') THEN 'brief_concept'
    WHEN old IN ('approval_documentation','approvals') THEN 'planning'
    WHEN old = 'construction_documentation'
      AND skeleton IN ('sk.new-dc-da','sk.new-dc-cdc','sk.new-eci-da','sk.ssd-dc') THEN 'design_finalisation'
    WHEN old = 'construction_documentation' THEN 'design_development'
    WHEN old = 'procurement' THEN 'tender_documentation'
    WHEN old IN ('construction','construction_stage_inspections','completion','defects') THEN 'delivery'
    ELSE old END
$$;

UPDATE program_activities a SET catalogue_stage_id = pg_temp.common_stage(a.catalogue_stage_id, p.skeleton_id)
FROM programs p
WHERE p.org_id = a.org_id AND p.id = a.program_id AND a.catalogue_stage_id IS NOT NULL
  AND a.catalogue_stage_id <> pg_temp.common_stage(a.catalogue_stage_id, p.skeleton_id);

UPDATE program_readings r SET catalogue_stage_id = pg_temp.common_stage(r.catalogue_stage_id, p.skeleton_id)
FROM programs p
WHERE p.org_id = r.org_id AND p.id = r.program_id AND r.catalogue_stage_id IS NOT NULL
  AND r.catalogue_stage_id <> pg_temp.common_stage(r.catalogue_stage_id, p.skeleton_id);
