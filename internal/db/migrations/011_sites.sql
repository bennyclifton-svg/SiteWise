-- A site is the lasting record of what exists: one address or campus. A
-- project is an intervention on one site. Parts belong to the site, so a
-- later project on the same building starts from what is known. Version 1
-- has one project per site; nothing visible changes.
-- Plan: docs/plans/2026-10-04-next-wave-implementation-plan.md §4.2.
CREATE TABLE sites (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    id uuid NOT NULL,
    label text NOT NULL,
    address text NOT NULL DEFAULT '',
    lot text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    CHECK (char_length(label) BETWEEN 1 AND 120),
    CHECK (char_length(address) <= 200),
    CHECK (char_length(lot) <= 80)
);

-- One site per existing project. The id is derived from the project id so a
-- rerun on a restored database gives the same site.
INSERT INTO sites (org_id, id, label)
SELECT org_id, md5('site:' || id::text)::uuid, COALESCE(NULLIF(left(name, 120), ''), 'Site')
FROM projects;

ALTER TABLE projects ADD COLUMN site_id uuid;
UPDATE projects SET site_id = md5('site:' || id::text)::uuid;
ALTER TABLE projects ALTER COLUMN site_id SET NOT NULL;
ALTER TABLE projects ADD FOREIGN KEY (org_id, site_id) REFERENCES sites (org_id, id);
-- Anchor for composite foreign keys that need a project and its site together.
ALTER TABLE projects ADD CONSTRAINT projects_org_id_site_uq UNIQUE (org_id, id, site_id);
-- Version 1: one project per site. Drop when several projects share a site.
ALTER TABLE projects ADD CONSTRAINT projects_one_per_site_v1 UNIQUE (org_id, site_id);

-- Parts become site-owned. The creating project is kept for provenance only;
-- deleting it no longer deletes the site's parts.
ALTER TABLE project_parts ADD COLUMN site_id uuid;
UPDATE project_parts pp SET site_id = p.site_id
FROM projects p WHERE p.org_id = pp.org_id AND p.id = pp.project_id;
ALTER TABLE project_parts ALTER COLUMN site_id SET NOT NULL;
ALTER TABLE project_parts RENAME COLUMN project_id TO created_by_project_id;
ALTER TABLE project_parts ALTER COLUMN created_by_project_id DROP NOT NULL;
ALTER TABLE project_parts DROP CONSTRAINT project_parts_org_id_project_id_fkey;
-- The creating project must be on the part's own site. MATCH SIMPLE skips
-- the check once the project is gone (NULL).
ALTER TABLE project_parts ADD FOREIGN KEY (org_id, created_by_project_id, site_id)
    REFERENCES projects (org_id, id, site_id) ON DELETE SET NULL (created_by_project_id);
ALTER TABLE project_parts ADD FOREIGN KEY (org_id, site_id) REFERENCES sites (org_id, id) ON DELETE CASCADE;
ALTER TABLE project_parts DROP CONSTRAINT project_parts_org_id_project_id_label_key;
ALTER TABLE project_parts ADD CONSTRAINT project_parts_site_label_uq UNIQUE (org_id, site_id, label);
ALTER TABLE project_parts ADD CONSTRAINT project_parts_site_id_uq UNIQUE (org_id, site_id, id);
-- Before 011 a renamed whole part let a second one be created. Keep the
-- oldest as the whole part; later ones become ordinary parts, so the user
-- values on them survive.
UPDATE project_parts pp SET kind = 'part'
WHERE pp.kind = 'whole' AND EXISTS (
    SELECT 1 FROM project_parts older
    WHERE older.org_id = pp.org_id AND older.site_id = pp.site_id AND older.kind = 'whole'
      AND (older.created_at, older.id) < (pp.created_at, pp.id));
CREATE UNIQUE INDEX project_parts_one_whole_per_site ON project_parts (org_id, site_id) WHERE kind = 'whole';
ALTER TABLE project_parts DROP CONSTRAINT project_parts_kind_check;
ALTER TABLE project_parts ADD CONSTRAINT project_parts_kind_check CHECK (kind IN
    ('whole', 'building', 'part', 'storey', 'compartment', 'tenancy', 'outbuilding', 'roof', 'plant_area'));

-- Backfill assertions: a violated named check fails the whole migration
-- (one transaction) rather than leaving a mismatch. The migration runner
-- splits on semicolons, so no DO block.
CREATE TEMP TABLE migration_011_assert (
    one_site_per_project boolean CONSTRAINT one_site_per_project CHECK (one_site_per_project),
    site_ids_derived_from_projects boolean CONSTRAINT site_ids_derived_from_projects CHECK (site_ids_derived_from_projects),
    parts_on_their_project_site boolean CONSTRAINT parts_on_their_project_site CHECK (parts_on_their_project_site)
) ON COMMIT DROP;
INSERT INTO migration_011_assert
SELECT (SELECT count(*) FROM projects) = (SELECT count(*) FROM sites),
       NOT EXISTS (SELECT 1 FROM projects WHERE site_id <> md5('site:' || id::text)::uuid),
       NOT EXISTS (SELECT 1 FROM project_parts pp JOIN projects p
                   ON p.org_id = pp.org_id AND p.id = pp.created_by_project_id
                   WHERE pp.site_id <> p.site_id);
DROP TABLE migration_011_assert;
