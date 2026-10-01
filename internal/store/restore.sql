-- Restore rehearsal facts. Counts and catalog state only; no row content.

-- name: OrgCounts :many
SELECT o.id::text AS org_id,
       (SELECT count(*) FROM users u WHERE u.org_id = o.id)::bigint AS users,
       (SELECT count(*) FROM memberships m WHERE m.org_id = o.id)::bigint AS memberships,
       (SELECT count(*) FROM projects p WHERE p.org_id = o.id)::bigint AS projects,
       (SELECT count(*) FROM files f WHERE f.org_id = o.id)::bigint AS files,
       (SELECT count(*) FROM documents d WHERE d.org_id = o.id)::bigint AS documents,
       (SELECT count(*) FROM decisions c WHERE c.org_id = o.id)::bigint AS decisions,
       (SELECT count(*) FROM passages s WHERE s.org_id = o.id)::bigint AS passages,
       (SELECT count(*) FROM events e WHERE e.org_id = o.id)::bigint AS events
FROM orgs o
ORDER BY o.id;

-- name: ForeignKeyState :one
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE NOT convalidated)::bigint AS unvalidated
FROM pg_catalog.pg_constraint
WHERE contype = 'f'
  AND connamespace = 'public'::regnamespace;

-- name: CrossOrgRows :many
-- Rows whose parent is not in the same org. Composite keys make most of
-- these impossible while the constraints hold; the checks prove the restored
-- data, not the schema. events.document_id has no foreign key at all.
SELECT 'events.document_id' AS reference, count(*)::bigint AS rows
FROM events e
WHERE e.document_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM documents d WHERE d.org_id = e.org_id AND d.id = e.document_id)
UNION ALL
SELECT 'documents.project_id', count(*)::bigint
FROM documents d
WHERE NOT EXISTS (SELECT 1 FROM projects p WHERE p.org_id = d.org_id AND p.id = d.project_id)
UNION ALL
SELECT 'documents.file_id', count(*)::bigint
FROM documents d
WHERE NOT EXISTS (SELECT 1 FROM files f WHERE f.org_id = d.org_id AND f.id = d.file_id)
UNION ALL
SELECT 'decisions.document_id', count(*)::bigint
FROM decisions c
WHERE NOT EXISTS (SELECT 1 FROM documents d WHERE d.org_id = c.org_id AND d.id = c.document_id)
UNION ALL
SELECT 'sessions.user_id', count(*)::bigint
FROM sessions s
WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.org_id = s.org_id AND u.id = s.user_id)
UNION ALL
SELECT 'memberships.user_id', count(*)::bigint
FROM memberships m
WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.org_id = m.org_id AND u.id = m.user_id);
