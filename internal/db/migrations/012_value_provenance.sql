-- Profile values record origin, review status and meaning, and belong to the
-- site (what exists) or the project (what the works are), per the key-scope
-- registry knowledge/profile/key_scope.yaml (owner decision D-04). A site
-- value has no project, so the next project on the same site sees it.
-- Plan: docs/plans/2026-10-04-next-wave-implementation-plan.md §4.1, §4.3.

-- User values: the user's word, now with provenance and an owner.
ALTER TABLE profile_user_values ADD COLUMN id uuid;
UPDATE profile_user_values SET id = gen_random_uuid();
ALTER TABLE profile_user_values ALTER COLUMN id SET NOT NULL;
-- A surrogate key: the natural key now depends on scope (partial indexes below).
ALTER TABLE profile_user_values DROP CONSTRAINT profile_user_values_pkey;
ALTER TABLE profile_user_values ADD PRIMARY KEY (org_id, id);
ALTER TABLE profile_user_values ADD COLUMN site_id uuid;
UPDATE profile_user_values v SET site_id = pp.site_id
FROM project_parts pp WHERE pp.org_id = v.org_id AND pp.id = v.part_id;
ALTER TABLE profile_user_values ALTER COLUMN site_id SET NOT NULL;
ALTER TABLE profile_user_values ADD COLUMN scope text NOT NULL DEFAULT 'project'
    CHECK (scope IN ('site', 'project'));
-- One-time copy of the registry's classification for existing rows. New
-- writes take the scope from the registry itself.
UPDATE profile_user_values SET scope = 'site'
WHERE (key LIKE 'det.%' AND key <> 'det.existing_building')
   OR key IN ('hdr.building_class', 'hdr.subclass')
   OR key LIKE 'hdr.scale.%'
   OR key ~ '^sys\..*\.(condition|existing)$';
ALTER TABLE profile_user_values ALTER COLUMN project_id DROP NOT NULL;
UPDATE profile_user_values SET project_id = NULL WHERE scope = 'site';
ALTER TABLE profile_user_values ADD CONSTRAINT profile_user_values_scope_owner
    CHECK ((scope = 'site') = (project_id IS NULL));

-- A value is set, cleared by the user (the old null), or explicitly unknown.
ALTER TABLE profile_user_values ADD COLUMN value_state text NOT NULL DEFAULT 'set'
    CHECK (value_state IN ('set', 'cleared', 'unknown'));
UPDATE profile_user_values SET value_state = 'cleared' WHERE value IS NULL;
ALTER TABLE profile_user_values ADD CONSTRAINT profile_user_values_value_state
    CHECK ((value IS NOT NULL) = (value_state = 'set'));

-- Provenance (plan §4.1). User input is accepted for planning, never verified
-- by default; verifying needs an actor and a basis.
ALTER TABLE profile_user_values ADD COLUMN origin text NOT NULL DEFAULT 'user'
    CHECK (origin IN ('user', 'assumption'));
ALTER TABLE profile_user_values ADD COLUMN review_status text NOT NULL DEFAULT 'accepted_for_planning'
    CHECK (review_status IN ('proposed', 'accepted_for_planning', 'verified', 'superseded'));
ALTER TABLE profile_user_values ADD COLUMN meaning text NOT NULL DEFAULT 'stated'
    CHECK (meaning IN ('stated', 'requirement', 'allowance', 'forecast'));
ALTER TABLE profile_user_values ADD COLUMN provenance jsonb NOT NULL DEFAULT '{}';
ALTER TABLE profile_user_values ADD COLUMN verified_by uuid;
ALTER TABLE profile_user_values ADD COLUMN verified_at timestamptz;
ALTER TABLE profile_user_values ADD COLUMN verification_basis text NOT NULL DEFAULT '';
ALTER TABLE profile_user_values ADD CONSTRAINT profile_user_values_verified_basis
    CHECK (review_status <> 'verified' OR (verified_by IS NOT NULL AND verification_basis <> ''));

-- Identity and ownership: one value per key and part, per site or per project,
-- enforced by the database on the owner the row belongs to.
CREATE UNIQUE INDEX profile_user_values_site_key
    ON profile_user_values (org_id, site_id, part_id, key) WHERE scope = 'site';
CREATE UNIQUE INDEX profile_user_values_project_key
    ON profile_user_values (org_id, project_id, part_id, key) WHERE scope = 'project';
ALTER TABLE profile_user_values DROP CONSTRAINT profile_user_values_org_id_part_id_fkey;
ALTER TABLE profile_user_values ADD FOREIGN KEY (org_id, site_id, part_id)
    REFERENCES project_parts (org_id, site_id, id) ON DELETE CASCADE;
ALTER TABLE profile_user_values ADD FOREIGN KEY (org_id, project_id, site_id)
    REFERENCES projects (org_id, id, site_id) ON DELETE CASCADE;

-- Rows: the per-project projection, rebuilt in code. They carry the same
-- provenance columns so a reader never re-derives them.
ALTER TABLE profile_rows ADD COLUMN site_id uuid;
UPDATE profile_rows r SET site_id = pp.site_id
FROM project_parts pp WHERE pp.org_id = r.org_id AND pp.id = r.part_id;
ALTER TABLE profile_rows ALTER COLUMN site_id SET NOT NULL;
ALTER TABLE profile_rows ADD COLUMN scope text NOT NULL DEFAULT 'project' CHECK (scope IN ('site', 'project'));
ALTER TABLE profile_rows ADD COLUMN origin text NOT NULL DEFAULT 'document'
    CHECK (origin IN ('document', 'user', 'calculation', 'assumption'));
ALTER TABLE profile_rows ADD COLUMN review_status text NOT NULL DEFAULT 'proposed'
    CHECK (review_status IN ('proposed', 'accepted_for_planning', 'verified', 'superseded'));
ALTER TABLE profile_rows ADD COLUMN meaning text NOT NULL DEFAULT 'stated'
    CHECK (meaning IN ('stated', 'requirement', 'allowance', 'forecast'));
ALTER TABLE profile_rows ADD COLUMN value_state text NOT NULL DEFAULT 'set'
    CHECK (value_state IN ('set', 'cleared', 'unknown'));
-- The user value's version, so a client can edit optimistically from a read.
ALTER TABLE profile_rows ADD COLUMN user_version bigint NOT NULL DEFAULT 0;
ALTER TABLE profile_rows DROP CONSTRAINT profile_rows_org_id_part_id_fkey;
ALTER TABLE profile_rows ADD FOREIGN KEY (org_id, site_id, part_id)
    REFERENCES project_parts (org_id, site_id, id) ON DELETE CASCADE;
ALTER TABLE profile_rows ADD FOREIGN KEY (org_id, project_id, site_id)
    REFERENCES projects (org_id, id, site_id) ON DELETE CASCADE;

-- Backfill assertions (named checks; the runner splits on semicolons).
CREATE TEMP TABLE migration_012_assert (
    user_values_on_their_site boolean CONSTRAINT user_values_on_their_site CHECK (user_values_on_their_site),
    project_values_keep_project boolean CONSTRAINT project_values_keep_project CHECK (project_values_keep_project)
) ON COMMIT DROP;
INSERT INTO migration_012_assert
SELECT NOT EXISTS (SELECT 1 FROM profile_user_values v JOIN project_parts pp
                   ON pp.org_id = v.org_id AND pp.id = v.part_id WHERE pp.site_id <> v.site_id),
       NOT EXISTS (SELECT 1 FROM profile_user_values WHERE scope = 'project' AND project_id IS NULL);
DROP TABLE migration_012_assert;
