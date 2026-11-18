-- The register and RFIs on the building
-- (docs/plans/2026-11-17-rfis-review-comments.md, Task 2).
--
-- RFIs and notices are kinds of project_delivery_items. Consent conditions
-- (plan 6) add a third kind to the same table: that migration drops and
-- re-adds the two named checks below (and delivery_numbered when conditions
-- carry numbers), listing every value here plus its own.
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_kind_check,
 ADD CONSTRAINT project_delivery_items_kind_check CHECK (kind IN ('activity','milestone','action','risk','issue','decision','approval','rfi','notice'));
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_status_check,
 ADD CONSTRAINT project_delivery_items_status_check CHECK (status IN ('not_started','in_progress','blocked','complete','cancelled','planned','achieved','open','mitigating','closed','resolved','decided','superseded','not_submitted','submitted','approved','rejected','withdrawn','draft','answered'));
ALTER TABLE project_delivery_items
 ADD COLUMN number integer CHECK (number > 0),
 ADD COLUMN system_id text CHECK (length(system_id) BETWEEN 1 AND 200),
 ADD COLUMN determinant_id text CHECK (length(determinant_id) BETWEEN 1 AND 200),
 ADD CONSTRAINT delivery_numbered CHECK ((number IS NOT NULL) = (kind = 'rfi'));
-- An RFI number is never reused, even after the row is retired.
CREATE UNIQUE INDEX delivery_number_uq ON project_delivery_items(org_id,project_id,kind,number) WHERE number IS NOT NULL;
CREATE INDEX delivery_system_idx ON project_delivery_items(org_id,project_id,system_id) WHERE system_id IS NOT NULL;

-- The project's RFI response time. Its own small table rather than a
-- column on projects, so the capture trigger can make setting it undoable
-- (chat applies it at once with Undo) without capturing every project write.
CREATE TABLE delivery_settings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 rfi_response_days smallint NOT NULL CHECK (rfi_response_days BETWEEN 1 AND 60),
 PRIMARY KEY (org_id,project_id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);

-- A possible answer: a filed document that may answer an open register
-- row. A proposal only; the PM's edit accepts it. Deleting the document
-- deletes its readings.
CREATE TABLE delivery_readings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 item_id uuid NOT NULL,
 document_id uuid NOT NULL,
 field text NOT NULL CHECK (field IN ('answers')),
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev')),
 probability double precision CHECK (probability BETWEEN 0 AND 1),
 band text NOT NULL CHECK (band IN ('amber','blank','grey')),
 basis text NOT NULL DEFAULT '' CHECK (length(basis) <= 200),
 question_version text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed','accepted','dismissed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,item_id,document_id,field),
 FOREIGN KEY (org_id,project_id,item_id) REFERENCES project_delivery_items(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,document_id) REFERENCES documents(org_id,id) ON DELETE CASCADE
);
CREATE INDEX delivery_readings_project ON delivery_readings(org_id,project_id) WHERE status='proposed';

-- Register writes go through the operations call, so each screen action
-- and each chat change has one Undo.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check,
 ADD CONSTRAINT project_changes_area_check CHECK (area IN ('cost_plan','profile','procurement','delivery'));
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON project_delivery_items
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON delivery_readings
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON delivery_settings
 FOR EACH ROW EXECUTE FUNCTION capture_change();
