-- Explicit domain counters; no dependency graph. Existing builds are stale
-- until rebuilt with the startup knowledge and question versions.
CREATE TABLE project_revisions (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    profile_inputs bigint NOT NULL DEFAULT 0 CHECK (profile_inputs >= 0),
    works bigint NOT NULL DEFAULT 0 CHECK (works >= 0),
    packages bigint NOT NULL DEFAULT 0 CHECK (packages >= 0),
    delivery bigint NOT NULL DEFAULT 0 CHECK (delivery >= 0),
    costs bigint NOT NULL DEFAULT 0 CHECK (costs >= 0),
    reports bigint NOT NULL DEFAULT 0 CHECK (reports >= 0),
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, project_id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE
);
INSERT INTO project_revisions (org_id, project_id) SELECT org_id, id FROM projects;

ALTER TABLE profile_builds
    ADD COLUMN revision bigint NOT NULL DEFAULT 0 CHECK (revision >= 0),
    ADD COLUMN input_fingerprint text NOT NULL DEFAULT '',
    ADD COLUMN knowledge_version text NOT NULL DEFAULT '',
    ADD COLUMN question_version text NOT NULL DEFAULT '',
    ADD COLUMN inputs jsonb NOT NULL DEFAULT '{}';
