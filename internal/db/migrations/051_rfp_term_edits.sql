-- Terms are edited in the terms table as scope lines are (owner direction,
-- 10 November 2026): term:<key> deletes a term, wording:terms.<key>
-- rewords it and custom:terms holds the terms the user adds, up to 20 of
-- 600 characters each.
ALTER TABLE report_settings DROP CONSTRAINT report_settings_key_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_key_check CHECK (
 key ~ '^(driver|stage|element|allowance|position|responsible|field|weight|line|wording|custom|term):[a-z0-9_.]{1,100}$'
 OR key IN ('scale','boundary:range','parts:affected','transmittal','agreement_docs')
);
ALTER TABLE report_settings DROP CONSTRAINT report_settings_value_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_value_check CHECK (
 (key IN ('transmittal','agreement_docs') AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 2000000)
 OR (key = 'custom:terms' AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 20000)
 OR (key NOT IN ('transmittal','agreement_docs','custom:terms') AND octet_length(value::text) <= 2000)
);
