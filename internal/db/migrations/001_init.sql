CREATE TABLE orgs (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Users belong to one org in v1. Identity is (org_id, id), not a global user.
CREATE TABLE users (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    id uuid NOT NULL,
    email text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, email)
);

CREATE TABLE memberships (
    org_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id),
    FOREIGN KEY (org_id, user_id) REFERENCES users (org_id, id) ON DELETE CASCADE
);

CREATE TABLE invites (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    id uuid NOT NULL,
    email text NOT NULL,
    token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL DEFAULT now() + interval '7 days',
    consumed_at timestamptz,
    PRIMARY KEY (org_id, id),
    CHECK (octet_length(token_hash) = 32)
);

-- Session ids are globally unique so a cookie can be resolved without a
-- client-supplied org. Tenant checks still use (org_id, id).
CREATE TABLE sessions (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT now() + interval '14 days',
    PRIMARY KEY (org_id, id),
    UNIQUE (id),
    FOREIGN KEY (org_id, user_id) REFERENCES users (org_id, id) ON DELETE CASCADE
);

CREATE TABLE projects (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    id uuid NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id)
);

-- The same blob may be filed in another project or org. Uniqueness is the
-- association, not the hash.
CREATE TABLE files (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    sha256 bytea NOT NULL,
    byte_size bigint NOT NULL,
    media_type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, project_id, sha256),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    CHECK (octet_length(sha256) = 32)
);

CREATE TABLE documents (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    project_id uuid NOT NULL,
    file_id uuid NOT NULL,
    filename text NOT NULL,
    status text NOT NULL,
    document_number text,
    revision text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    FOREIGN KEY (org_id, project_id) REFERENCES projects (org_id, id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, file_id) REFERENCES files (org_id, id) ON DELETE CASCADE
);

-- Number and revision identify a document inside one project. Rows still
-- being filed have a null identity and are not unique.
CREATE UNIQUE INDEX documents_identity_uq
    ON documents (org_id, project_id, document_number, revision)
    WHERE document_number IS NOT NULL AND revision IS NOT NULL;

CREATE TABLE supersessions (
    org_id uuid NOT NULL,
    document_id uuid NOT NULL,
    prior_document_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, document_id),
    CHECK (document_id <> prior_document_id),
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, prior_document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE
);

CREATE TABLE decisions (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    field text NOT NULL,
    value text,
    band text NOT NULL,
    decided_by text NOT NULL,
    question_version text,
    confidence double precision,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, document_id, field),
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE
);

CREATE TABLE passages (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    ordinal integer NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (org_id, document_id, ordinal),
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE
);

CREATE TABLE jobs (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    document_id uuid NOT NULL,
    kind text NOT NULL,
    status text NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    run_after timestamptz NOT NULL DEFAULT now(),
    locked_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    FOREIGN KEY (org_id, document_id) REFERENCES documents (org_id, id) ON DELETE CASCADE
);
