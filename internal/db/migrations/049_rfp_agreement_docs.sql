-- The client agreement's documents are attached in Terms (owner direction,
-- 8 November 2026), saved like the transmittal: one list of document rows.
ALTER TABLE report_settings DROP CONSTRAINT report_settings_key_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_key_check CHECK (
 key ~ '^(driver|stage|element|allowance|position|responsible|field|weight|line|wording|custom):[a-z0-9_.]{1,100}$'
 OR key IN ('scale','boundary:range','parts:affected','transmittal','agreement_docs')
);
ALTER TABLE report_settings DROP CONSTRAINT report_settings_value_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_value_check CHECK (
 (key IN ('transmittal','agreement_docs') AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 2000000)
 OR (key NOT IN ('transmittal','agreement_docs') AND octet_length(value::text) <= 2000)
);
