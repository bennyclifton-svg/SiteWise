-- name: ListOrgProjects :many
SELECT id::text AS id, name
FROM projects
WHERE org_id = sqlc.arg(org_id)::uuid
ORDER BY created_at DESC, id;

-- name: ListProjectDocumentViews :many
SELECT
    d.id::text AS id,
    d.filename,
    d.status,
    d.reason,
    COALESCE(d.document_number, '') AS document_number,
    COALESCE(d.revision, '') AS revision,
    COALESCE(s.prior_document_id::text, '')::text AS supersedes_id,
    d.created_at,
    COALESCE((SELECT j.status FROM jobs j WHERE j.org_id = d.org_id AND j.document_id = d.id AND j.kind = 'full_text'), '')::text AS text_status,
    COALESCE((SELECT ds.pages FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),0)::integer AS text_pages,
    COALESCE((SELECT cardinality(ds.empty_pages) FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),0)::integer AS text_empty_pages,
    COALESCE((SELECT ds.version FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),'')::text AS text_source_version,
    d.profile_read
FROM documents d
LEFT JOIN supersessions s
    ON s.org_id = d.org_id
   AND s.document_id = d.id
WHERE d.org_id = sqlc.arg(org_id)::uuid
  AND d.project_id = sqlc.arg(project_id)::uuid
ORDER BY d.created_at DESC, d.id;

-- name: GetDocumentView :one
SELECT
    d.id::text AS id,
    d.project_id::text AS project_id,
    d.filename,
    d.status,
    d.reason,
    COALESCE(d.document_number, '') AS document_number,
    COALESCE(d.revision, '') AS revision,
    COALESCE(s.prior_document_id::text, '')::text AS supersedes_id,
    d.created_at,
    COALESCE((SELECT j.status FROM jobs j WHERE j.org_id = d.org_id AND j.document_id = d.id AND j.kind = 'full_text'), '')::text AS text_status,
    COALESCE((SELECT ds.pages FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),0)::integer AS text_pages,
    COALESCE((SELECT cardinality(ds.empty_pages) FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),0)::integer AS text_empty_pages,
    COALESCE((SELECT ds.version FROM document_sources ds WHERE ds.org_id=d.org_id AND ds.document_id=d.id),'')::text AS text_source_version,
    d.profile_read
FROM documents d
LEFT JOIN supersessions s
    ON s.org_id = d.org_id
   AND s.document_id = d.id
WHERE d.org_id = sqlc.arg(org_id)::uuid
  AND d.id = sqlc.arg(id)::uuid;

-- name: ListProjectDecisionViews :many
SELECT
    dc.document_id::text AS document_id,
    dc.field,
    COALESCE(dc.value, '') AS value,
    dc.band,
    dc.decided_by,
    COALESCE(dc.question_version, '') AS question_version,
    dc.confidence
FROM decisions dc
JOIN documents d
    ON d.org_id = dc.org_id
   AND d.id = dc.document_id
WHERE dc.org_id = sqlc.arg(org_id)::uuid
  AND d.project_id = sqlc.arg(project_id)::uuid
ORDER BY dc.document_id, dc.field;

-- name: LatestEventID :one
SELECT COALESCE(
    (SELECT last_id FROM event_counters WHERE org_id = sqlc.arg(org_id)::uuid),
    0
)::bigint AS id;

-- name: MarkPendingNotFiled :execrows
UPDATE documents
SET status = 'not_filed', reason = sqlc.arg(reason)
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND status = 'pending';

-- Every org's intake that has not been filed. Startup resumes these; the
-- caller then works inside each row's org.
-- name: ListPendingIntake :many
SELECT j.org_id::text AS org_id, j.document_id::text AS document_id
FROM jobs j
JOIN documents d
    ON d.org_id = j.org_id
   AND d.id = j.document_id
WHERE j.kind = 'intake'
  AND j.status = 'queued'
  AND d.status = 'pending'
ORDER BY j.created_at
LIMIT sqlc.arg(row_limit)::int;
