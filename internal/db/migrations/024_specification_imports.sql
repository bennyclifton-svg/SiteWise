CREATE TABLE specification_imports (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 document_id uuid NOT NULL,
 requested_by uuid NOT NULL,
 target_type_id uuid,
 version text NOT NULL,
 imported_count integer NOT NULL DEFAULT 0,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,document_id),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,document_id) REFERENCES documents(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,requested_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(org_id,project_id,target_type_id) REFERENCES specification_types(org_id,project_id,id)
);
CREATE TABLE specification_import_readings (
 org_id uuid NOT NULL,
 document_id uuid NOT NULL,
 candidate_key text NOT NULL,
 fingerprint text NOT NULL,
 result jsonb NOT NULL,
 PRIMARY KEY(org_id,document_id,candidate_key),
 FOREIGN KEY(org_id,document_id) REFERENCES specification_imports(org_id,document_id) ON DELETE CASCADE
);
ALTER TABLE specification_assertions DROP CONSTRAINT specification_assertions_meaning_check;
ALTER TABLE specification_assertions ADD CONSTRAINT specification_assertions_meaning_check CHECK (meaning IN ('requirement','selection','capability','allowance','exclusion','observed','unclassified'));
