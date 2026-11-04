-- A transmittal is one saved document schedule, larger than a scalar RFP setting.
-- Keep the existing limits for every other setting and the issued-content trigger.
ALTER TABLE report_settings DROP CONSTRAINT report_settings_key_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_key_check CHECK (
 key ~ '^(driver|stage|element|allowance|position|responsible|field|weight):[a-z0-9_.]{1,100}$'
 OR key IN ('scale','boundary:range','parts:affected','transmittal')
);
ALTER TABLE report_settings DROP CONSTRAINT report_settings_value_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_value_check CHECK (
 (key = 'transmittal' AND jsonb_typeof(value) = 'array' AND octet_length(value::text) <= 2000000)
 OR (key <> 'transmittal' AND octet_length(value::text) <= 2000)
);
