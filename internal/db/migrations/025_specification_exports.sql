ALTER TABLE specification_issues ADD COLUMN export_sha256 bytea CHECK(export_sha256 IS NULL OR octet_length(export_sha256)=32);
