-- name: CreateOrg :exec
INSERT INTO orgs (id, name)
VALUES (sqlc.arg(id)::uuid, sqlc.arg(name));

-- name: DeleteOrg :exec
DELETE FROM orgs
WHERE id = sqlc.arg(id)::uuid;

-- name: CreateUser :exec
INSERT INTO users (org_id, id, email)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(id)::uuid, sqlc.arg(email));

-- name: CreateMembership :exec
INSERT INTO memberships (org_id, user_id, role)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(user_id)::uuid, sqlc.arg(role))
ON CONFLICT (org_id, user_id) DO NOTHING;

-- name: CreateInvite :exec
INSERT INTO invites (org_id, id, email, token_hash, role, expires_at)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(email),
    sqlc.arg(token_hash),
    sqlc.arg(role),
    sqlc.arg(expires_at)
);

-- name: LockInviteByHash :one
SELECT
    org_id::text AS org_id,
    id::text AS id,
    email,
    role,
    expires_at,
    (consumed_at IS NOT NULL)::boolean AS consumed
FROM invites
WHERE token_hash = sqlc.arg(token_hash)
FOR UPDATE;

-- name: MarkInviteConsumed :execrows
UPDATE invites
SET consumed_at = now()
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND consumed_at IS NULL;

-- name: FindUserByEmail :one
SELECT id::text AS id
FROM users
WHERE org_id = sqlc.arg(org_id)::uuid
  AND email = sqlc.arg(email);

-- name: MembershipExists :one
SELECT true AS member
FROM memberships
WHERE org_id = sqlc.arg(org_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid;

-- name: LookupSession :one
SELECT org_id::text AS org_id, user_id::text AS user_id, expires_at
FROM sessions
WHERE id = sqlc.arg(id)::uuid;

-- name: GetProject :one
SELECT id::text AS id, name
FROM projects
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: GetInvite :one
SELECT id::text AS id, email
FROM invites
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: CreateSession :exec
INSERT INTO sessions (org_id, id, user_id)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(id)::uuid, sqlc.arg(user_id)::uuid);

-- name: GetSession :one
SELECT id::text AS id, user_id::text AS user_id
FROM sessions
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: CreateProject :exec
INSERT INTO projects (org_id, id, name)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(id)::uuid, sqlc.arg(name));

-- name: CreateFile :exec
INSERT INTO files (org_id, id, project_id, sha256, byte_size, media_type)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(project_id)::uuid,
    sqlc.arg(sha256),
    sqlc.arg(byte_size),
    sqlc.arg(media_type)
);

-- name: GetFile :one
SELECT id::text AS id, project_id::text AS project_id, sha256, byte_size, media_type
FROM files
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: FindFileByHash :one
SELECT id::text AS id, project_id::text AS project_id, sha256, byte_size, media_type
FROM files
WHERE org_id = sqlc.arg(org_id)::uuid
  AND project_id = sqlc.arg(project_id)::uuid
  AND sha256 = sqlc.arg(sha256);

-- name: CreateDocument :exec
INSERT INTO documents (
    org_id, id, project_id, file_id, filename, status, document_number, revision
) VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(project_id)::uuid,
    sqlc.arg(file_id)::uuid,
    sqlc.arg(filename),
    sqlc.arg(status),
    sqlc.narg(document_number),
    sqlc.narg(revision)
);

-- name: GetDocument :one
SELECT
    d.id::text AS id,
    d.project_id::text AS project_id,
    d.file_id::text AS file_id,
    d.filename,
    d.status,
    COALESCE(d.document_number, '') AS document_number,
    COALESCE(d.revision, '') AS revision
FROM documents d
WHERE d.org_id = sqlc.arg(org_id)::uuid
  AND d.id = sqlc.arg(id)::uuid;

-- name: GetSupersession :one
SELECT prior_document_id::text AS prior_document_id
FROM supersessions
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid;

-- name: UpdateDocumentStatus :execrows
UPDATE documents
SET status = sqlc.arg(status)
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: SupersedeDocument :execrows
INSERT INTO supersessions (org_id, document_id, prior_document_id)
SELECT sqlc.arg(org_id)::uuid, sqlc.arg(document_id)::uuid, sqlc.arg(prior_document_id)::uuid
WHERE EXISTS (
    SELECT 1 FROM documents
    WHERE org_id = sqlc.arg(org_id)::uuid
      AND id = sqlc.arg(document_id)::uuid
) AND EXISTS (
    SELECT 1 FROM documents
    WHERE org_id = sqlc.arg(org_id)::uuid
      AND id = sqlc.arg(prior_document_id)::uuid
) AND sqlc.arg(document_id)::uuid <> sqlc.arg(prior_document_id)::uuid;

-- name: CreateDecision :exec
INSERT INTO decisions (org_id, id, document_id, field, value, band, decided_by)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(field),
    sqlc.arg(value),
    sqlc.arg(band),
    sqlc.arg(decided_by)
);

-- name: ListStream :many
SELECT
    d.id::text AS id,
    d.document_id::text AS document_id,
    d.field,
    COALESCE(d.value, '') AS value,
    d.band,
    d.decided_by
FROM decisions d
JOIN documents doc
    ON doc.org_id = d.org_id
   AND doc.id = d.document_id
WHERE d.org_id = sqlc.arg(org_id)::uuid
  AND doc.project_id = sqlc.arg(project_id)::uuid
ORDER BY d.id;

-- name: AddPassage :exec
INSERT INTO passages (org_id, id, document_id, ordinal, body)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(ordinal),
    sqlc.arg(body)
);

-- name: GetPassage :one
SELECT id::text AS id, document_id::text AS document_id, ordinal, body
FROM passages
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: EnqueueJob :execrows
INSERT INTO jobs (org_id, id, document_id, kind, status)
SELECT sqlc.arg(org_id)::uuid, sqlc.arg(id)::uuid, sqlc.arg(document_id)::uuid, sqlc.arg(kind), 'queued'
WHERE EXISTS (
    SELECT 1 FROM documents
    WHERE org_id = sqlc.arg(org_id)::uuid
      AND id = sqlc.arg(document_id)::uuid
);

-- name: ListJobs :many
SELECT id::text AS id, document_id::text AS document_id, kind, status
FROM jobs
WHERE org_id = sqlc.arg(org_id)::uuid
ORDER BY id;
