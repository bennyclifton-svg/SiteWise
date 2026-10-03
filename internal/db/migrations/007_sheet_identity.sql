-- A drawing number may span several separately scheduled physical pages.
-- Keep the printed number/revision literal; source-page position distinguishes
-- those rows while ordinary single-document identities retain position zero.
ALTER TABLE documents ADD COLUMN identity_page integer NOT NULL DEFAULT 0
    CHECK (identity_page >= 0 AND identity_page <= 200);
UPDATE documents d SET identity_page=s.page_number
FROM drawing_sheets s WHERE s.org_id=d.org_id AND s.document_id=d.id;
DROP INDEX documents_identity_uq;
CREATE UNIQUE INDEX documents_identity_uq
    ON documents(org_id, project_id, document_number, revision, identity_page)
    WHERE document_number IS NOT NULL AND revision IS NOT NULL;
