-- Structured consultant RFP choices (appointment boundary, stage state,
-- counts, I/O/X, per-fact drivers). Kept per draft version and copied
-- forward like report_edits, so a choice survives refresh and the next draft
-- after issue. basis records the source fact the user overrode, so a later
-- source change shows as a conflict instead of being hidden by the setting.
CREATE TABLE report_settings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 report_version_id uuid NOT NULL,
 key text NOT NULL CHECK (key ~ '^(driver|stage|element|allowance|position|responsible|field|weight):[a-z0-9_.]{1,100}$' OR key IN ('scale','boundary:range','parts:affected')),
 value jsonb NOT NULL CHECK (octet_length(value::text) <= 2000),
 basis jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(basis)='object' AND octet_length(basis::text) <= 4000),
 user_id uuid NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,report_version_id,key),
 FOREIGN KEY (org_id,report_version_id) REFERENCES report_versions(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,user_id) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED
);
-- An issued version's settings are part of its frozen basis.
CREATE TRIGGER report_settings_immutable BEFORE INSERT OR UPDATE OR DELETE ON report_settings FOR EACH ROW EXECUTE FUNCTION protect_issued_report_content();
