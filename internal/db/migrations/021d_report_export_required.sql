-- A committed issue must have a frozen, content-addressed export as well as
-- its snapshot. Issue writes both atomically; draft rows remain unrestricted.
ALTER TABLE report_versions ADD CONSTRAINT issued_report_has_export
 CHECK (status <> 'issued' OR export_file_sha256 IS NOT NULL);
