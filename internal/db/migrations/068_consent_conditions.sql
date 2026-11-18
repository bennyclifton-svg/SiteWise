-- Consent conditions (docs/plans/2026-11-17-consent-conditions.md, Task 1).
--
-- A consent condition is a third register kind in project_delivery_items,
-- beside RFIs and notices (066). Its statuses are an obligation's: open,
-- evidence filed, satisfied, not applicable. Its number stays in details
-- ("A1" is a condition number), so delivery_numbered is unchanged. The
-- register's post-filing job stores evidence suggestions in
-- delivery_readings under field 'evidence'. Program gates name their
-- source 'consent'.
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_kind_check,
 ADD CONSTRAINT project_delivery_items_kind_check CHECK (kind IN ('activity','milestone','action','risk','issue','decision','approval','rfi','notice','condition'));
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_status_check,
 ADD CONSTRAINT project_delivery_items_status_check CHECK (status IN ('not_started','in_progress','blocked','complete','cancelled','planned','achieved','open','mitigating','closed','resolved','decided','superseded','not_submitted','submitted','approved','rejected','withdrawn','draft','answered','evidence_filed','satisfied','not_applicable'));
ALTER TABLE delivery_readings DROP CONSTRAINT delivery_readings_field_check,
 ADD CONSTRAINT delivery_readings_field_check CHECK (field IN ('answers','evidence'));
ALTER TABLE program_changes DROP CONSTRAINT program_changes_source_check,
 ADD CONSTRAINT program_changes_source_check CHECK (source IN ('profile','work_items','procurement','library','consent'));
-- Conditions are read per notice and matched by their consent and number.
CREATE INDEX delivery_condition_idx ON project_delivery_items(org_id,project_id) WHERE kind='condition';
-- The consent read is a background job (kind 'consent'). requested_by is
-- the PM who asked for it, so the rows it writes are one change that PM can
-- undo; outcome is what the read found (conditions read, gaps, the consent
-- reference), shown above the register's conditions.
ALTER TABLE jobs ADD COLUMN requested_by uuid, ADD COLUMN outcome jsonb CHECK (outcome IS NULL OR jsonb_typeof(outcome)='object');
