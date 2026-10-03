-- Project profile. Facts are Jev or rule readings per passage; user values are
-- final; profile_rows is the precomputed view the page reads in one query.
CREATE TABLE project_parts (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    label text NOT NULL,
    kind text NOT NULL,
    ncc_class text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, project_id, label),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    CHECK (kind IN ('whole', 'building', 'part', 'storey', 'compartment', 'tenancy', 'outbuilding')),
    CHECK (char_length(label) BETWEEN 1 AND 80)
);

-- Replaced per document and question prefix when a stage reruns; never edited.
CREATE TABLE profile_facts (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    document_id uuid NOT NULL,
    passage_id uuid,
    question_id text NOT NULL,
    value text NOT NULL,
    unit text NOT NULL DEFAULT '',
    basis text NOT NULL DEFAULT '',
    part_label text NOT NULL DEFAULT '',
    excerpt text NOT NULL DEFAULT '',
    confidence double precision,
    decided_by text NOT NULL,
    question_version text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE,
    CHECK (decided_by IN ('jev', 'rule')),
    CHECK (char_length(excerpt) <= 120)
);
CREATE INDEX profile_facts_by_project ON profile_facts (org_id, project_id);
CREATE INDEX profile_facts_by_document ON profile_facts (org_id, document_id, question_id);

-- The user's word. No background stage writes here.
CREATE TABLE profile_user_values (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    part_id uuid NOT NULL,
    key text NOT NULL,
    value text,
    note text NOT NULL DEFAULT '',
    user_id uuid NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, project_id, part_id, key),
    FOREIGN KEY (org_id, part_id) REFERENCES project_parts (org_id, id) ON DELETE CASCADE,
    CHECK (char_length(note) <= 120)
);

CREATE TABLE profile_rows (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    part_id uuid NOT NULL,
    key text NOT NULL,
    value text NOT NULL DEFAULT '',
    band text NOT NULL,
    assertion text NOT NULL DEFAULT '',
    note text NOT NULL DEFAULT '',
    tenders text NOT NULL DEFAULT '',
    sources jsonb NOT NULL DEFAULT '[]',
    alternatives jsonb NOT NULL DEFAULT '[]',
    derived jsonb,
    PRIMARY KEY (org_id, project_id, part_id, key),
    FOREIGN KEY (org_id, part_id) REFERENCES project_parts (org_id, id) ON DELETE CASCADE,
    CHECK (band IN ('green', 'amber', 'red', 'blank', 'suggested', 'user', 'unchecked'))
);

CREATE TABLE profile_builds (
    org_id uuid NOT NULL,
    project_id uuid NOT NULL,
    built_at timestamptz NOT NULL DEFAULT now(),
    thresholds_version text NOT NULL DEFAULT '',
    PRIMARY KEY (org_id, project_id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE
);
