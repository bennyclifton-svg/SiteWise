-- A fact's project and source document must agree. Separate tenant-scoped
-- references permit a document from another project in the same organisation.
ALTER TABLE documents
 ADD CONSTRAINT documents_org_project_id_uq UNIQUE (org_id,project_id,id);

ALTER TABLE profile_facts
 DROP CONSTRAINT profile_facts_org_id_document_id_fkey,
 ADD CONSTRAINT profile_facts_document_project_fkey
 FOREIGN KEY (org_id,project_id,document_id)
 REFERENCES documents(org_id,project_id,id) ON DELETE CASCADE;

-- passage_id remains a historical locator: snapshot reads explicitly retain
-- facts whose old passage is unavailable. This does not relax document scope.
