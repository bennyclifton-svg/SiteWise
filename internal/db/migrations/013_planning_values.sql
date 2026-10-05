-- Planning values: assumptions and calculated values with a range and
-- limitations, kept apart from document-only profile_facts (PRG L329). An
-- assumption stays an assumption however it is reviewed (L276), and a value
-- may be explicitly unknown. Keys come from knowledge/profile/planning_keys.yaml,
-- which has no money totals: those belong to the cost plan.
-- Plan: docs/plans/2026-10-04-next-wave-implementation-plan.md §4.3 (WP-13).
CREATE TABLE profile_planning_values (
    org_id uuid NOT NULL,
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    site_id uuid NOT NULL,
    project_id uuid,
    scope text NOT NULL CHECK (scope IN ('site', 'project')),
    part_id uuid NOT NULL,
    key text NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*$' AND key NOT LIKE 'cost%'),
    value_state text NOT NULL CHECK (value_state IN ('set', 'unknown')),
    value_text text,
    value_numeric numeric,
    value_bool boolean,
    range_low numeric,
    range_high numeric,
    unit text NOT NULL DEFAULT '',
    origin text NOT NULL CHECK (origin IN ('user', 'assumption', 'calculation')),
    -- A planning value is never verified: verifying it would make it a fact.
    review_status text NOT NULL CHECK (review_status IN ('proposed', 'accepted_for_planning', 'superseded')),
    meaning text NOT NULL CHECK (meaning IN ('stated', 'requirement', 'allowance', 'forecast')),
    rationale text NOT NULL DEFAULT '' CHECK (char_length(rationale) <= 500),
    limitations text NOT NULL DEFAULT '' CHECK (char_length(limitations) <= 500),
    -- How a calculation was made and the origins of its inputs (NW-REQ-176, 183).
    provenance jsonb NOT NULL DEFAULT '{}',
    user_id uuid,
    version bigint NOT NULL CHECK (version > 0),
    superseded_by uuid,
    superseded_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    CONSTRAINT profile_planning_values_scope_owner CHECK ((scope = 'site') = (project_id IS NULL)),
    -- Exactly one typed value when set; none when unknown (never zero or false).
    CONSTRAINT profile_planning_values_typed CHECK (
        (value_state = 'set') = (num_nonnulls(value_text, value_numeric, value_bool) = 1)
        AND num_nonnulls(value_text, value_numeric, value_bool) <= 1),
    CONSTRAINT profile_planning_values_range CHECK (
        (range_low IS NULL OR range_high IS NULL OR range_low <= range_high)
        AND ((range_low IS NULL AND range_high IS NULL) OR value_text IS NULL AND value_bool IS NULL)),
    CONSTRAINT profile_planning_values_superseded CHECK ((review_status = 'superseded') = (superseded_at IS NOT NULL)),
    FOREIGN KEY (org_id, site_id, part_id) REFERENCES project_parts (org_id, site_id, id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, project_id, site_id) REFERENCES projects (org_id, id, site_id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, superseded_by) REFERENCES profile_planning_values (org_id, id)
);

-- One live value per key and part, per site or per project; superseded rows
-- are history and never deleted.
CREATE UNIQUE INDEX profile_planning_values_site_key ON profile_planning_values (org_id, site_id, part_id, key)
    WHERE scope = 'site' AND review_status <> 'superseded';
CREATE UNIQUE INDEX profile_planning_values_project_key ON profile_planning_values (org_id, project_id, part_id, key)
    WHERE scope = 'project' AND review_status <> 'superseded';
CREATE INDEX profile_planning_values_by_site ON profile_planning_values (org_id, site_id);

-- The profile shows a planning value as its own band, never as evidence.
ALTER TABLE profile_rows DROP CONSTRAINT profile_rows_band_check;
ALTER TABLE profile_rows ADD CONSTRAINT profile_rows_band_check
    CHECK (band IN ('green', 'amber', 'red', 'blank', 'suggested', 'user', 'planning', 'unchecked'));
