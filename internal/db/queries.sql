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
    org_id, id, project_id, file_id, filename, status, document_number, revision, reason
) VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(project_id)::uuid,
    sqlc.arg(file_id)::uuid,
    sqlc.arg(filename),
    sqlc.arg(status),
    sqlc.narg(document_number),
    sqlc.narg(revision),
    sqlc.arg(reason)
);

-- name: GetDocument :one
SELECT
    d.id::text AS id,
    d.project_id::text AS project_id,
    d.file_id::text AS file_id,
    d.filename,
    d.status,
    COALESCE(d.document_number, '') AS document_number,
    COALESCE(d.revision, '') AS revision,
    d.reason
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
INSERT INTO jobs (org_id, id, document_id, kind, status, priority)
SELECT
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(kind),
    'queued',
    CASE WHEN sqlc.arg(kind) IN ('intake', 'jev_retry') THEN 100 ELSE 0 END
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

-- name: InsertFile :one
INSERT INTO files (org_id, id, project_id, sha256, byte_size, media_type)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(project_id)::uuid,
    sqlc.arg(sha256),
    sqlc.arg(byte_size),
    sqlc.arg(media_type)
)
ON CONFLICT (org_id, project_id, sha256) DO NOTHING
RETURNING id::text AS id;

-- name: LockFileByHash :one
SELECT id::text AS id, project_id::text AS project_id, sha256, byte_size, media_type
FROM files
WHERE org_id = sqlc.arg(org_id)::uuid
  AND project_id = sqlc.arg(project_id)::uuid
  AND sha256 = sqlc.arg(sha256)
FOR UPDATE;

-- name: DocumentByFile :one
SELECT
    d.id::text AS id,
    d.project_id::text AS project_id,
    d.file_id::text AS file_id,
    d.filename,
    d.status,
    d.reason
FROM documents d
WHERE d.org_id = sqlc.arg(org_id)::uuid
  AND d.file_id = sqlc.arg(file_id)::uuid;

-- name: JobByDocumentKind :one
SELECT id::text AS id, document_id::text AS document_id, kind, status
FROM jobs
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid
  AND kind = sqlc.arg(kind);

-- name: ListContentHashes :many
SELECT DISTINCT sha256
FROM files
ORDER BY sha256;

-- name: NextEventID :one
INSERT INTO event_counters (org_id, last_id)
VALUES (sqlc.arg(org_id)::uuid, 1)
ON CONFLICT (org_id) DO UPDATE
SET last_id = event_counters.last_id + 1
RETURNING last_id;

-- name: InsertEvent :exec
INSERT INTO events (org_id, id, kind, document_id, payload)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id),
    sqlc.arg(kind),
    sqlc.narg(document_id)::uuid,
    sqlc.arg(payload)
);

-- name: ListEventsAfter :many
SELECT
    id,
    kind,
    CAST(COALESCE(document_id::text, '') AS text) AS document_id,
    payload
FROM events
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(row_limit);

-- name: FailExhaustedJobs :exec
UPDATE jobs
SET
    status = 'failed',
    last_error = CASE WHEN last_error = '' THEN 'attempts exhausted' ELSE last_error END,
    locked_until = NULL,
    lease_token = NULL
WHERE org_id = sqlc.arg(org_id)::uuid
  AND status = 'leased'
  AND locked_until < now()
  AND attempts >= max_attempts;

-- name: ClaimJob :one
-- The pick is a materialized CTE so it locks once. As a FROM subquery the
-- planner rescanned it inside a nested loop, and a concurrent claimer could
-- come back empty while a second job was still queued.
WITH picked AS MATERIALIZED (
    SELECT c.org_id, c.id
    FROM jobs c
    WHERE c.org_id = sqlc.arg(org_id)::uuid
      AND c.kind = ANY(sqlc.arg(kinds)::text[])
      AND c.kind <> 'intake'
      AND c.attempts < c.max_attempts
      AND c.run_after <= now()
      AND (
          c.status = 'queued'
          OR (c.status = 'leased' AND c.locked_until < now())
      )
    ORDER BY c.priority DESC, c.run_after, c.created_at, c.id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE jobs AS j
SET
    status = 'leased',
    attempts = j.attempts + 1,
    lease_token = sqlc.arg(lease_token)::uuid,
    locked_until = now() + make_interval(secs => sqlc.arg(lease_seconds)::double precision),
    last_error = ''
FROM picked
WHERE j.org_id = picked.org_id
  AND j.id = picked.id
RETURNING
    j.org_id::text AS org_id,
    j.id::text AS id,
    j.document_id::text AS document_id,
    j.kind,
    j.attempts,
    j.lease_token::text AS lease_token;

-- name: CompleteJob :execrows
UPDATE jobs
SET status = 'done', locked_until = NULL, lease_token = NULL
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'leased';

-- name: FailJob :execrows
UPDATE jobs
SET
    status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
    run_after = CASE
        WHEN attempts >= max_attempts THEN run_after
        ELSE now() + make_interval(secs => sqlc.arg(backoff_seconds)::double precision)
    END,
    last_error = sqlc.arg(last_error),
    locked_until = NULL,
    lease_token = NULL
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'leased';

-- name: ExpireJobLease :execrows
UPDATE jobs
SET locked_until = now() - interval '1 second'
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid
  AND status = 'leased';

-- name: GetJob :one
SELECT
    id::text AS id,
    document_id::text AS document_id,
    kind,
    status,
    attempts,
    max_attempts,
    priority,
    COALESCE(last_error, '') AS last_error
FROM jobs
WHERE org_id = sqlc.arg(org_id)::uuid
  AND id = sqlc.arg(id)::uuid;

-- name: EnqueueStage :execrows
INSERT INTO jobs (org_id, id, document_id, kind, status, priority)
SELECT
    sqlc.arg(org_id)::uuid,
    sqlc.arg(id)::uuid,
    sqlc.arg(document_id)::uuid,
    sqlc.arg(kind),
    'queued',
    0
WHERE EXISTS (
    SELECT 1 FROM documents
    WHERE org_id = sqlc.arg(org_id)::uuid
      AND id = sqlc.arg(document_id)::uuid
)
ON CONFLICT (org_id, document_id, kind) DO NOTHING;

-- name: ListDocumentPassages :many
SELECT id::text AS id, ordinal, body
FROM passages
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid
ORDER BY ordinal;

-- name: DeleteDocumentPassages :exec
DELETE FROM passages
WHERE org_id = sqlc.arg(org_id)::uuid
  AND document_id = sqlc.arg(document_id)::uuid;

-- name: DeletePassageSystems :exec
DELETE FROM passage_systems
WHERE org_id = sqlc.arg(org_id)::uuid
  AND passage_id = sqlc.arg(passage_id)::uuid;

-- name: DeletePassageEvidence :exec
DELETE FROM passage_evidence
WHERE org_id = sqlc.arg(org_id)::uuid
  AND passage_id = sqlc.arg(passage_id)::uuid;

-- name: InsertPassageSystem :exec
INSERT INTO passage_systems (org_id, passage_id, system_id)
VALUES (sqlc.arg(org_id)::uuid, sqlc.arg(passage_id)::uuid, sqlc.arg(system_id))
ON CONFLICT (org_id, passage_id, system_id) DO NOTHING;

-- name: ListPassageSystems :many
SELECT system_id
FROM passage_systems
WHERE org_id = sqlc.arg(org_id)::uuid
  AND passage_id = sqlc.arg(passage_id)::uuid
ORDER BY system_id;

-- name: UpsertPassageEvidence :exec
INSERT INTO passage_evidence (org_id, passage_id, question_id, state)
VALUES (
    sqlc.arg(org_id)::uuid,
    sqlc.arg(passage_id)::uuid,
    sqlc.arg(question_id),
    sqlc.arg(state)
)
ON CONFLICT (org_id, passage_id, question_id) DO UPDATE
SET state = EXCLUDED.state;

-- name: ListPassageEvidence :many
SELECT question_id, state
FROM passage_evidence
WHERE org_id = sqlc.arg(org_id)::uuid
  AND passage_id = sqlc.arg(passage_id)::uuid
ORDER BY question_id;

-- name: SearchPassages :many
SELECT p.id::text AS id, p.document_id::text AS document_id, p.ordinal, p.body
FROM passages p
JOIN documents d
    ON d.org_id = p.org_id
   AND d.id = p.document_id
WHERE p.org_id = sqlc.arg(org_id)::uuid
  AND d.project_id = sqlc.arg(project_id)::uuid
  AND p.body_tsv @@ websearch_to_tsquery('english', sqlc.arg(query))
ORDER BY ts_rank(p.body_tsv, websearch_to_tsquery('english', sqlc.arg(query))) DESC, p.ordinal
LIMIT sqlc.arg(row_limit);

-- name: MembershipRole :one
SELECT role
FROM memberships
WHERE org_id = sqlc.arg(org_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid;

-- name: Backlog :many
-- Process-wide unfinished work for health. Counts and ages only; no row of
-- any org leaves this query. Age is by the database clock.
SELECT kind,
       count(*)::bigint AS queued,
       floor(extract(epoch FROM now() - min(created_at)) * 1000)::bigint AS oldest_ms
FROM jobs
WHERE status IN ('queued', 'leased')
GROUP BY kind
ORDER BY kind;

-- name: OrgBacklog :many
SELECT kind,
       count(*)::bigint AS queued,
       floor(extract(epoch FROM now() - min(created_at)) * 1000)::bigint AS oldest_ms
FROM jobs
WHERE org_id = sqlc.arg(org_id)::uuid
  AND status IN ('queued', 'leased')
GROUP BY kind
ORDER BY kind;
