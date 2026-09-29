ALTER TABLE documents ADD COLUMN reason text NOT NULL DEFAULT '';

ALTER TABLE documents ADD CONSTRAINT documents_reason_chk
    CHECK (status <> 'not_filed' OR length(reason) > 0);

-- One filing per blob association. A re-drop returns that filing.
CREATE UNIQUE INDEX documents_file_uq ON documents (org_id, file_id);

-- One durable job per document and kind. Retries update the same row.
CREATE UNIQUE INDEX jobs_document_kind_uq ON jobs (org_id, document_id, kind);
