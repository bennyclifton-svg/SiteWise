-- Read-only psql inventory for before/after upgrade comparison.
-- psql -X --csv --file internal/db/migrations/checks/pre_m2_preservation.sql DB
-- Compare rows for pre-existing tables. New cost tables are expected additions.
-- Normalizes only declared changes: new layout/document/hash columns and the
-- old cached proposal rank (now independently checked through ranked_proposals).
-- Output contains counts and digests, never source/provenance text. SHA-256 is
-- PostgreSQL's built-in bytea hash, not an extension. Large tables can require
-- substantial sort/aggregate memory: use the streaming Go rehearsal for fixtures.
\set ON_ERROR_STOP on
BEGIN READ ONLY;
SET LOCAL TIME ZONE 'UTC';
SELECT format(
 'SELECT %L AS table_name,count(*) AS rows,encode(sha256(convert_to(COALESCE(string_agg(content || chr(10),'''' ORDER BY content COLLATE "C"),''''),''UTF8'')),''hex'') AS sha256 FROM (SELECT (to_jsonb(r)-%L::text[])::text AS content FROM %I.%I r) saved_rows',
 tablename,
 CASE tablename WHEN 'work_items' THEN '{layout_change}' WHEN 'proposals' THEN '{rank,projection_hash}'
 WHEN 'profile_rows' THEN '{projection_hash}' WHEN 'passage_sources' THEN '{document_id}' WHEN 'passage_calls' THEN '{document_id}' ELSE '{}' END,
 schemaname,tablename)
FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations' ORDER BY tablename
\gexec
ROLLBACK;
