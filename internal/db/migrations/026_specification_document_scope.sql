-- A request's source belongs to this project, not merely to its organisation.
ALTER TABLE specification_imports DROP CONSTRAINT specification_imports_org_id_document_id_fkey;
ALTER TABLE specification_imports ADD CONSTRAINT specification_imports_document_scope_fkey
 FOREIGN KEY(org_id,project_id,document_id) REFERENCES documents(org_id,project_id,id) ON DELETE CASCADE;
