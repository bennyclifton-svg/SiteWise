-- Per-org event ids. A cursor from one org cannot walk another org's log.
CREATE TABLE event_counters (
    org_id uuid PRIMARY KEY REFERENCES orgs (id) ON DELETE CASCADE,
    last_id bigint NOT NULL
);

CREATE TABLE events (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    id bigint NOT NULL,
    kind text NOT NULL,
    document_id uuid,
    payload text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    CHECK (id > 0),
    CHECK (length(kind) > 0)
);

ALTER TABLE jobs ADD COLUMN priority integer NOT NULL DEFAULT 0;
ALTER TABLE jobs ADD COLUMN lease_token uuid;
ALTER TABLE jobs ADD COLUMN max_attempts integer NOT NULL DEFAULT 5;
ALTER TABLE jobs ADD COLUMN last_error text NOT NULL DEFAULT '';

ALTER TABLE jobs ADD CONSTRAINT jobs_status_chk
    CHECK (status IN ('queued', 'leased', 'done', 'failed'));

ALTER TABLE jobs ADD CONSTRAINT jobs_priority_chk CHECK (priority >= 0);
ALTER TABLE jobs ADD CONSTRAINT jobs_attempts_chk
    CHECK (attempts >= 0 AND attempts <= max_attempts);

-- Full-text index is maintained with the passage body. Writers do not set it.
ALTER TABLE passages ADD COLUMN body_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('english', body)) STORED;

CREATE INDEX passages_body_tsv_idx ON passages USING gin (body_tsv);

CREATE TABLE passage_systems (
    org_id uuid NOT NULL,
    passage_id uuid NOT NULL,
    system_id text NOT NULL,
    PRIMARY KEY (org_id, passage_id, system_id),
    FOREIGN KEY (org_id, passage_id) REFERENCES passages (org_id, id) ON DELETE CASCADE
);

CREATE TABLE passage_evidence (
    org_id uuid NOT NULL,
    passage_id uuid NOT NULL,
    question_id text NOT NULL,
    state text NOT NULL,
    PRIMARY KEY (org_id, passage_id, question_id),
    FOREIGN KEY (org_id, passage_id) REFERENCES passages (org_id, id) ON DELETE CASCADE,
    CHECK (state IN ('addressed', 'not_addressed', 'unknown'))
);
