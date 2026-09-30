-- name: LockFilingDocument :one
SELECT
    id::text AS id,
    project_id::text AS project_id,
    filename,
    status,
    COALESCE(document_number, '') AS document_number,
    COALESCE(revision, '') AS revision
FROM documents
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: ListFilingDecisions :many
SELECT
    field,
    COALESCE(value, '') AS value,
    band,
    decided_by,
    COALESCE(question_version, '') AS question_version,
    confidence,
    version
FROM decisions
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid
ORDER BY field;

-- name: UpsertFilingDecision :execrows
INSERT INTO decisions (
    org_id, id, document_id, field, value, band, decided_by, question_version, confidence, version
) VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(field),
    sqlc.narg(value),
    sqlc.arg(band),
    sqlc.arg(decided_by),
    sqlc.narg(question_version),
    CASE
        WHEN sqlc.arg(confidence_set)::boolean THEN sqlc.arg(confidence)::float8
        ELSE NULL
    END,
    1
)
ON CONFLICT (org_id, document_id, field) DO UPDATE
SET
    value = EXCLUDED.value,
    band = EXCLUDED.band,
    decided_by = EXCLUDED.decided_by,
    question_version = EXCLUDED.question_version,
    confidence = EXCLUDED.confidence,
    version = decisions.version + 1
WHERE decisions.decided_by <> 'user'
  AND decisions.version = sqlc.arg(expected_version);

-- name: CorrectFilingDecision :execrows
INSERT INTO decisions (
    org_id, id, document_id, field, value, band, decided_by, question_version, confidence, version
) VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(field),
    sqlc.narg(value),
    'green',
    'user',
    NULL,
    NULL,
    1
)
ON CONFLICT (org_id, document_id, field) DO UPDATE
SET
    value = EXCLUDED.value,
    band = 'green',
    decided_by = 'user',
    question_version = NULL,
    confidence = NULL,
    version = decisions.version + 1;

-- name: ListProjectDocuments :many
SELECT
    id::text AS id,
    project_id::text AS project_id,
    COALESCE(document_number, '') AS document_number,
    COALESCE(revision, '') AS revision,
    status
FROM documents
WHERE org_id = sqlc.arg(org_id)::uuid
  AND project_id = sqlc.arg(project_id)::uuid
ORDER BY id;

-- name: ListOrgSupersessions :many
SELECT
    document_id::text AS document_id,
    prior_document_id::text AS prior_document_id
FROM supersessions
WHERE org_id = sqlc.arg(org_id)::uuid;

-- name: InsertScopedSupersession :execrows
INSERT INTO supersessions (org_id, document_id, prior_document_id)
SELECT sqlc.arg(org_id)::uuid, sqlc.arg(document_id)::uuid, sqlc.arg(prior_document_id)::uuid
WHERE sqlc.arg(document_id)::uuid <> sqlc.arg(prior_document_id)::uuid
  AND EXISTS (
    SELECT 1
    FROM documents newer
    JOIN documents prior
        ON prior.org_id = newer.org_id
       AND prior.project_id = newer.project_id
       AND upper(prior.document_number) = upper(newer.document_number)
       AND prior.document_number IS NOT NULL
       AND prior.id <> newer.id
    WHERE newer.org_id = sqlc.arg(org_id)::uuid
      AND newer.id = sqlc.arg(document_id)::uuid
      AND prior.id = sqlc.arg(prior_document_id)::uuid
  );

-- name: SetFiledIdentity :execrows
UPDATE documents
SET
    status = 'filed',
    document_number = sqlc.narg(document_number),
    revision = sqlc.narg(revision)
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND status = 'pending';

-- name: FinishIntakeJob :execrows
UPDATE jobs
SET status = 'done'
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid
  AND kind = 'intake';
