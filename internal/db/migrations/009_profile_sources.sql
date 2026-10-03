-- Preserve source text separately from its reading units. Existing documents
-- are refreshed by the next explicit profile update, never by a migration.
CREATE TABLE document_sources (
 org_id uuid NOT NULL, document_id uuid NOT NULL,
 version text NOT NULL, pages integer NOT NULL DEFAULT 0,
 empty_pages integer[] NOT NULL DEFAULT '{}',
 source jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, document_id),
 FOREIGN KEY (org_id, document_id) REFERENCES documents(org_id,id) ON DELETE CASCADE
);
CREATE TABLE passage_sources (
 org_id uuid NOT NULL, passage_id uuid NOT NULL,
 page integer NOT NULL DEFAULT 0, location text NOT NULL DEFAULT '',
 section text NOT NULL DEFAULT '', context text NOT NULL DEFAULT '',
 start_offset integer NOT NULL DEFAULT 0, end_offset integer NOT NULL DEFAULT 0,
 category text NOT NULL DEFAULT 'unresolved', provider text NOT NULL DEFAULT '',
 scope text NOT NULL DEFAULT '', outcome text NOT NULL DEFAULT 'pending',
 confidence double precision, mapped_keys text[] NOT NULL DEFAULT '{}',
 unresolved text[] NOT NULL DEFAULT '{}',
 PRIMARY KEY (org_id, passage_id),
 FOREIGN KEY (org_id,passage_id) REFERENCES passages(org_id,id) ON DELETE CASCADE,
 CHECK (outcome IN ('pending','mapped','background','needs_mapping'))
);
CREATE TABLE passage_calls (
 org_id uuid NOT NULL, passage_id uuid NOT NULL,
 stage text NOT NULL, fingerprint text NOT NULL, result jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,passage_id,stage),
 FOREIGN KEY (org_id,passage_id) REFERENCES passages(org_id,id) ON DELETE CASCADE
);
