-- The Accommodation schedule's edits (owner, 11 Nov 2026;
-- docs/plans/2026-11-11-accommodation-schedule-lines.md). The schedule's
-- lines are built in code on every profile read from the building type,
-- the documents' readings and the card; only the user's word is kept here:
-- what they changed on a generated line (line_key, such as "offices#3"), a
-- line they added (no key), its place in the table, a split, a removal.
CREATE TABLE accommodation_lines (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 line_key text,
 name text NOT NULL DEFAULT '',
 level text NOT NULL DEFAULT '',
 area_sqm numeric,
 comment text NOT NULL DEFAULT '',
 -- The fields the user set; a field set blank stays blank.
 set_fields text[] NOT NULL DEFAULT '{}',
 position double precision,
 split boolean NOT NULL DEFAULT false,
 removed boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, line_key),
 FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
 CHECK (line_key IS NULL OR line_key ~ '^[a-z][a-z0-9_]*#[1-9][0-9]{0,2}$'),
 CHECK (area_sqm IS NULL OR area_sqm >= 0),
 CHECK (set_fields <@ ARRAY['name', 'level', 'area', 'comment']::text[]),
 CHECK (length(name) <= 120 AND length(level) <= 80 AND length(comment) <= 500)
);
CREATE INDEX accommodation_lines_project_idx ON accommodation_lines (org_id, project_id, created_at);
