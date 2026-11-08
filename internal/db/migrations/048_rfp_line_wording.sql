-- Scope lines edited in place (owner direction, 8 November 2026): a standard
-- line's wording can be the user's own for this RFP (wording:<stage>.<line>),
-- and line:<stage>.<line> may also be "D", deleted from this RFP.
ALTER TABLE report_settings DROP CONSTRAINT report_settings_key_check;
ALTER TABLE report_settings ADD CONSTRAINT report_settings_key_check CHECK (
 key ~ '^(driver|stage|element|allowance|position|responsible|field|weight|line|wording|custom):[a-z0-9_.]{1,100}$'
 OR key IN ('scale','boundary:range','parts:affected','transmittal')
);
