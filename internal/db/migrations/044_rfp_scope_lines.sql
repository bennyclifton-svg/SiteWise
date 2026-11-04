-- Scope by stage (owner direction, 3 November 2026): each stage's scope
-- lines are included or excluded on their own (line:<stage>.<line>), and a
-- user may add their own lines to a stage (custom:<stage>, a short array).
ALTER TABLE report_settings DROP CONSTRAINT report_settings_key_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_key_check CHECK (
 key ~ '^(driver|stage|element|allowance|position|responsible|field|weight|line|custom):[a-z0-9_.]{1,100}$'
 OR key IN ('scale','boundary:range','parts:affected','transmittal')
);
ALTER TABLE report_settings DROP CONSTRAINT report_settings_value_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_value_check CHECK (
 (key = 'transmittal' AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 2000000)
 OR (key LIKE 'custom:%' AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 20000)
 OR (key <> 'transmittal' AND key NOT LIKE 'custom:%' AND octet_length(value::text) <= 2000)
);
