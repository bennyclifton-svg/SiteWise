-- One stage plan shared with Procurement and the Cost Plan (plan
-- docs/plans/2026-10-18-program.md section 7.6, WP-P10).
--
-- A downstream document records the program's stage plan as it was when the
-- document was built: stage_plan is every live program stage at that
-- stage_revision, in program order, as [{"id","label"}]. Differences are
-- then the program's own changes since, keyed by the stable stage id, so a
-- rename is never mistaken for a delete and an add. Null means the document
-- was never built from a program: nothing is offered.
--
-- stage_acknowledged is the revision the user chose Keep at; the offer does
-- not repeat until the program's stage revision passes it.

-- A package's stage set. Each package_stages row keeps its own
-- program_stage_id and stage_revision (033).
ALTER TABLE packages ADD COLUMN stage_plan jsonb CHECK (stage_plan IS NULL OR jsonb_typeof(stage_plan)='array');
ALTER TABLE packages ADD COLUMN stage_revision bigint CHECK (stage_revision IS NULL OR stage_revision>0);
ALTER TABLE packages ADD COLUMN stage_acknowledged bigint CHECK (stage_acknowledged IS NULL OR stage_acknowledged>0);
ALTER TABLE packages ADD CONSTRAINT packages_stage_plan_revision CHECK ((stage_plan IS NULL) = (stage_revision IS NULL));

-- A package stage whose program stage was deleted while money (invoices, an
-- award, live cost lines) is attached is kept and marked, never retired.
ALTER TABLE package_stages ADD COLUMN removed_from_program timestamptz;

-- An RFP version's stage rows: the package's stages as program stages,
-- [{"program_stage_id","label","removed_from_program"}], built with the
-- version and rewritten only by Update. Issued versions stay as issued.
-- report_versions.stage_revision is from 033.
ALTER TABLE report_versions ADD COLUMN stage_plan jsonb CHECK (stage_plan IS NULL OR jsonb_typeof(stage_plan)='array');
ALTER TABLE report_versions ADD COLUMN stage_rows jsonb CHECK (stage_rows IS NULL OR jsonb_typeof(stage_rows)='array');
ALTER TABLE report_versions ADD CONSTRAINT report_versions_stage_plan_revision CHECK ((stage_plan IS NULL) = (stage_revision IS NULL));

-- Keep on an RFP lives on the report, so it holds whether the latest
-- version is a draft or an issued one (issued versions cannot change).
ALTER TABLE reports ADD COLUMN stage_acknowledged bigint CHECK (stage_acknowledged IS NULL OR stage_acknowledged>0);
