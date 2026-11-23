-- Correspondence between parties, first slice
-- (docs/plans/2026-11-23-correspondence-between-parties.md, section 18).
--
-- A notice of delay, an extension of time claim and a letter are register
-- kinds, each numbered in its own run (NOD-1, EOT-1, LTR-1); the unique
-- index on (org, project, kind, number) already keeps the runs apart. An
-- EOT claim adds one status, assessed. Every kind and status allowed before
-- stays (066, 068).
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_kind_check,
 ADD CONSTRAINT project_delivery_items_kind_check CHECK (kind IN ('activity','milestone','action','risk','issue','decision','approval','rfi','notice','condition','delay_notice','eot_claim','letter'));
ALTER TABLE project_delivery_items DROP CONSTRAINT project_delivery_items_status_check,
 ADD CONSTRAINT project_delivery_items_status_check CHECK (status IN ('not_started','in_progress','blocked','complete','cancelled','planned','achieved','open','mitigating','closed','resolved','decided','superseded','not_submitted','submitted','approved','rejected','withdrawn','draft','answered','evidence_filed','satisfied','not_applicable','assessed'));
ALTER TABLE project_delivery_items DROP CONSTRAINT delivery_numbered,
 ADD CONSTRAINT delivery_numbered CHECK ((number IS NOT NULL) = (kind IN ('rfi','delay_notice','eot_claim','letter')));

-- The project's optional terms. Each is null until the PM sets it, and Go
-- falls back to SiteWise's default (or to unknown for the completion date)
-- in one place. The agreed completion date is its own term, never the
-- Profile's target completion (which a document may state and the user
-- owns); the date after EOTs is computed at read, so nothing stored moves.
ALTER TABLE delivery_settings ALTER COLUMN rfi_response_days DROP NOT NULL,
 ADD COLUMN eot_response_days smallint CHECK (eot_response_days BETWEEN 1 AND 120),
 ADD COLUMN agreed_completion date;
