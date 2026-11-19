package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Email intake (docs/plans/2026-11-14-email-intake.md, Tasks 2 and 5). Each
// project has one private address; a message sent to it is recorded once
// (per webhook token and per Message-Id), then filed like dropped files or
// held for a member. A filed message is a record: no route edits or deletes
// it (correspondence plan, A7); the table's trigger refuses it as well.

// Held reasons, in the words the screen turns into plain English.
const (
	MailUnknownSender    = "unknown_sender"
	MailUnverifiedSender = "unverified_sender"
	MailPlanLimit        = "plan_limit"
	MailArchived         = "archived"
	MailHeldFull         = "held_full"
	// MailBusy: more mail for this project in the last hour than the webhook
	// files; the rest is held so a flood cannot spend the org's reads.
	MailBusy = "busy"
)

// Held mail is capped per project and expires (plan, Task 5).
const (
	MailHeldMax   = 50
	MailHeldBytes = 500 << 20
	MailHeldDays  = 30
)

// ErrMailHeldFull: a held message whose bytes were not kept has nothing to file.
var ErrMailHeldFull = errors.New("held mail is full; nothing was kept")

// ErrMailNotHeld: accept or discard named a message that is filed.
var ErrMailNotHeld = errors.New("this message is filed")

// MailTarget is the project an address belongs to.
type MailTarget struct {
	OrgID, ProjectID string
	Archived         bool
}

// MailPartIn is one part of an inbound message, bytes already stored.
// Skipped parts have no hash.
type MailPartIn struct {
	// N is 0 for the body document and 1 on for attachments in order.
	N                     int
	Filename, ContentType string
	SHA256                []byte
	Size                  int64
	Skipped               string
}

// MailIn is one inbound message to record. HeldReason "" means it is to
// be filed.
type MailIn struct {
	ProjectID, MessageID, Token         string
	Sender, SenderName, Subject, To, Cc string
	InReplyTo, References, Body         string
	ReceivedAt                          time.Time
	HeldReason                          string
	Parts                               []MailPartIn
}

// MailRecorded is what RecordMail did. Expired and Unkept are hashes whose
// bytes the caller removes when nothing else references them.
type MailRecorded struct {
	ID         string
	Repeat     bool
	Status     string
	HeldReason string
	Expired    [][]byte
	Unkept     [][]byte
}

// MailPart is one part as the screen lists it.
type MailPart struct {
	N           int    `json:"n"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"byte_size"`
	Skipped     string `json:"skipped"`
	DocumentID  string `json:"document_id,omitempty"`
	SHA256      []byte `json:"-"`
}

// MailMessage is one message as the screen and chat read it. Body is
// filled only when one message is read.
type MailMessage struct {
	ID         string     `json:"id"`
	MessageID  string     `json:"-"`
	Sender     string     `json:"sender"`
	SenderName string     `json:"sender_name"`
	Subject    string     `json:"subject"`
	To         string     `json:"to"`
	Cc         string     `json:"cc"`
	InReplyTo  string     `json:"-"`
	References string     `json:"-"`
	Body       string     `json:"body,omitempty"`
	ReceivedAt time.Time  `json:"received_at"`
	Status     string     `json:"status"`
	HeldReason string     `json:"reason"`
	Parts      []MailPart `json:"parts"`
}

// newMailLocalPart is "p-" and 32 random hex characters. It carries no
// name, so an address never says whose project it is.
func newMailLocalPart() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return "p-" + hex.EncodeToString(b[:])
}

// MailAddress is the project's local part. A project made before the
// address table, or by a path that skipped CreateProject, gets one now.
func (s *Store) MailAddress(ctx context.Context, org, project string) (string, error) {
	if !validUUID(org) || !validUUID(project) {
		return "", ErrNotFound
	}
	var local string
	err := s.pool.QueryRow(ctx, `SELECT local_part FROM project_mail_addresses WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project).Scan(&local)
	if err == nil {
		return local, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO project_mail_addresses(org_id,project_id,local_part)
SELECT org_id,id,$3 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid
ON CONFLICT (org_id,project_id) DO NOTHING`, org, project, newMailLocalPart())
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		err = s.pool.QueryRow(ctx, `SELECT local_part FROM project_mail_addresses WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project).Scan(&local)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return local, err
	}
	return s.MailAddress(ctx, org, project)
}

// ProjectByMailAddress finds the project an address belongs to, in any
// letter case. An unknown address is ErrNotFound.
func (s *Store) ProjectByMailAddress(ctx context.Context, local string) (MailTarget, error) {
	local = strings.ToLower(strings.TrimSpace(local))
	var t MailTarget
	err := s.pool.QueryRow(ctx, `SELECT a.org_id::text,a.project_id::text,p.archived_at IS NOT NULL
FROM project_mail_addresses a JOIN projects p ON p.org_id=a.org_id AND p.id=a.project_id
WHERE a.local_part=$1`, local).Scan(&t.OrgID, &t.ProjectID, &t.Archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return MailTarget{}, ErrNotFound
	}
	return t, err
}

// RotateMailAddress gives the project a new address. The old one stops at
// once: mail to it, including replies to anything sent from it, is dropped.
func (s *Store) RotateMailAddress(ctx context.Context, org, project string) (string, error) {
	if _, err := s.MailAddress(ctx, org, project); err != nil {
		return "", err
	}
	local := newMailLocalPart()
	tag, err := s.pool.Exec(ctx, `UPDATE project_mail_addresses SET local_part=$3, created_at=now() WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project, local)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", ErrNotFound
	}
	return local, nil
}

// MailMember is the user of org whose confirmed email is address, or "".
// An unconfirmed address is not a member here: anyone can sign up with
// someone else's address and never confirm it.
func (s *Store) MailMember(ctx context.Context, org, address string) (string, error) {
	var user string
	err := s.pool.QueryRow(ctx, `SELECT u.id::text FROM users u
JOIN memberships m ON m.org_id=u.org_id AND m.user_id=u.id
WHERE u.org_id=$1::uuid AND lower(u.email)=lower($2) AND u.email_verified
LIMIT 1`, org, address).Scan(&user)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return user, err
}

// SeenWebhookToken records a signed token and reports whether it was seen
// before. Every verified post records its token, whatever happens to it, so
// a captured post cannot be sent again with another body. Tokens older than
// two days are pruned here: the webhook refuses timestamps older than one.
func (s *Store) SeenWebhookToken(ctx context.Context, token string) (bool, error) {
	if _, err := s.pool.Exec(ctx, `DELETE FROM mail_webhook_tokens WHERE seen_at < now() - interval '2 days'`); err != nil {
		return false, err
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO mail_webhook_tokens(token) VALUES ($1) ON CONFLICT DO NOTHING`, token)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 0, nil
}

// RecordMail records one inbound message in one transaction under the
// project lock: expire old held mail, record the token, refuse a repeat
// token or Message-Id, cap held mail, insert the message and its parts.
func (s *Store) RecordMail(ctx context.Context, org string, in MailIn) (MailRecorded, error) {
	var out MailRecorded
	if !validUUID(org) || !validUUID(in.ProjectID) {
		return out, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, org, in.ProjectID); err != nil {
		return out, err
	}
	expired, err := expireHeldMail(ctx, tx, org, in.ProjectID, in.ReceivedAt)
	if err != nil {
		return out, err
	}
	out.Expired = expired
	tag, err := tx.Exec(ctx, `INSERT INTO mail_webhook_tokens(token) VALUES ($1) ON CONFLICT DO NOTHING`, in.Token)
	if err != nil {
		return out, err
	}
	if tag.RowsAffected() == 0 {
		out.Repeat = true
		return out, tx.Commit(ctx)
	}
	var existing string
	err = tx.QueryRow(ctx, `SELECT id::text FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND message_id=$3`, org, in.ProjectID, in.MessageID).Scan(&existing)
	if err == nil {
		out.Repeat, out.ID = true, existing
		return out, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	status, reason := "filed", in.HeldReason
	if reason != "" {
		status = "held"
		var count int
		var held int64
		if err := tx.QueryRow(ctx, `SELECT count(DISTINCT m.id), COALESCE(sum(p.byte_size),0)::bigint FROM mail_messages m
LEFT JOIN mail_parts p ON p.org_id=m.org_id AND p.message_id=m.id
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.status='held'`, org, in.ProjectID).Scan(&count, &held); err != nil {
			return out, err
		}
		var adding int64
		for _, p := range in.Parts {
			adding += p.Size
		}
		if count >= MailHeldMax || held+adding > MailHeldBytes {
			reason = MailHeldFull
		}
	}
	out.ID, out.Status, out.HeldReason = newID(), status, reason
	received := in.ReceivedAt
	if received.IsZero() {
		received = time.Now()
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,sender_name,subject,to_addrs,cc_addrs,in_reply_to,refs,body,received_at,status,held_reason)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		org, out.ID, in.ProjectID, in.MessageID, in.Token, in.Sender, in.SenderName, in.Subject, in.To, in.Cc,
		in.InReplyTo, in.References, in.Body, received, status, reason); err != nil {
		return out, err
	}
	for _, p := range in.Parts {
		sum, skipped := p.SHA256, p.Skipped
		if reason == MailHeldFull && skipped == "" {
			out.Unkept = append(out.Unkept, sum)
			sum, skipped = nil, "not_kept"
		}
		if skipped != "" {
			sum = nil
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mail_parts(org_id,message_id,n,filename,content_type,sha256,byte_size,skipped)
VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8)`, org, out.ID, p.N, p.Filename, p.ContentType, sum, p.Size, skipped); err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

// expireHeldMail deletes this project's held messages older than the
// limit and returns their hashes.
func expireHeldMail(ctx context.Context, tx pgx.Tx, org, project string, now time.Time) ([][]byte, error) {
	if now.IsZero() {
		now = time.Now()
	}
	rows, err := tx.Query(ctx, `WITH gone AS (
 DELETE FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND status='held'
  AND received_at < $3::timestamptz - make_interval(days => $4) RETURNING org_id,id)
SELECT p.sha256 FROM mail_parts p JOIN gone g ON g.org_id=p.org_id AND g.id=p.message_id WHERE p.sha256 IS NOT NULL`,
		org, project, now, MailHeldDays)
	if err != nil {
		return nil, err
	}
	return scanHashes(rows)
}

func scanHashes(rows pgx.Rows) ([][]byte, error) {
	defer rows.Close()
	var out [][]byte
	for rows.Next() {
		var sum []byte
		if err := rows.Scan(&sum); err != nil {
			return nil, err
		}
		out = append(out, sum)
	}
	return out, rows.Err()
}

// MailFilter narrows the message list. Held lists only held mail.
type MailFilter struct {
	Held   bool
	Before time.Time // paging: messages received before this
	Limit  int
}

const mailColumns = `m.id::text,m.message_id,m.sender,m.sender_name,m.subject,m.to_addrs,m.cc_addrs,m.in_reply_to,m.refs,m.received_at,m.status,m.held_reason`

func scanMailMessage(row pgx.Row, m *MailMessage) error {
	return row.Scan(&m.ID, &m.MessageID, &m.Sender, &m.SenderName, &m.Subject, &m.To, &m.Cc, &m.InReplyTo, &m.References, &m.ReceivedAt, &m.Status, &m.HeldReason)
}

// MailMessages lists the project's messages, filed and held, newest first,
// each with its parts, in two queries.
func (s *Store) MailMessages(ctx context.Context, org, project string, f MailFilter) ([]MailMessage, error) {
	if !validUUID(org) || !validUUID(project) {
		return nil, ErrNotFound
	}
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 1000
	}
	var before *time.Time
	if !f.Before.IsZero() {
		before = &f.Before
	}
	rows, err := s.pool.Query(ctx, `SELECT `+mailColumns+` FROM mail_messages m
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND (NOT $3 OR m.status='held')
  AND ($4::timestamptz IS NULL OR m.received_at < $4)
ORDER BY m.received_at DESC, m.id LIMIT $5`, org, project, f.Held, before, f.Limit)
	if err != nil {
		return nil, err
	}
	out := []MailMessage{}
	index := map[string]int{}
	for rows.Next() {
		var m MailMessage
		if err := scanMailMessage(rows, &m); err != nil {
			rows.Close()
			return nil, err
		}
		m.Parts = []MailPart{}
		index[m.ID] = len(out)
		out = append(out, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(out) == 0 {
		return out, err
	}
	ids := make([]string, len(out))
	for i, m := range out {
		ids[i] = m.ID
	}
	parts, err := s.mailParts(ctx, org, ids)
	if err != nil {
		return nil, err
	}
	for id, ps := range parts {
		out[index[id]].Parts = ps
	}
	return out, nil
}

func (s *Store) mailParts(ctx context.Context, org string, ids []string) (map[string][]MailPart, error) {
	rows, err := s.pool.Query(ctx, `SELECT message_id::text,n,filename,content_type,byte_size,skipped,COALESCE(document_id::text,''),sha256
FROM mail_parts WHERE org_id=$1::uuid AND message_id::text = ANY($2::text[]) ORDER BY message_id,n`, org, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]MailPart{}
	for rows.Next() {
		var id string
		var p MailPart
		if err := rows.Scan(&id, &p.N, &p.Filename, &p.ContentType, &p.Size, &p.Skipped, &p.DocumentID, &p.SHA256); err != nil {
			return nil, err
		}
		out[id] = append(out[id], p)
	}
	return out, rows.Err()
}

// MailMessageByID reads one message of this project with its body and parts.
func (s *Store) MailMessageByID(ctx context.Context, org, project, id string) (MailMessage, error) {
	var m MailMessage
	if !validUUID(org) || !validUUID(project) || !validUUID(id) {
		return m, ErrNotFound
	}
	err := s.pool.QueryRow(ctx, `SELECT `+mailColumns+`,m.body FROM mail_messages m
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.id=$3::uuid`, org, project, id).Scan(
		&m.ID, &m.MessageID, &m.Sender, &m.SenderName, &m.Subject, &m.To, &m.Cc, &m.InReplyTo, &m.References, &m.ReceivedAt, &m.Status, &m.HeldReason, &m.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	parts, err := s.mailParts(ctx, org, []string{m.ID})
	if err != nil {
		return m, err
	}
	m.Parts = parts[m.ID]
	if m.Parts == nil {
		m.Parts = []MailPart{}
	}
	return m, nil
}

// SetMailPartDocument records the document a part became, as soon as it
// exists and before filing starts (correspondence plan A3), so the filing
// finds its message.
func (s *Store) SetMailPartDocument(ctx context.Context, org, message string, n int, document string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE mail_parts SET document_id=$4::uuid
WHERE org_id=$1::uuid AND message_id=$2::uuid AND n=$3 AND sha256 IS NOT NULL`, org, message, n, document)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// MarkMailFiled marks a message filed. by is the member who accepted held
// mail, or "" for mail the webhook filed.
func (s *Store) MarkMailFiled(ctx context.Context, org, project, id, by string) error {
	var decided any
	if by != "" {
		decided = by
	}
	tag, err := s.pool.Exec(ctx, `UPDATE mail_messages SET status='filed', held_reason='', decided_by=COALESCE($4::uuid,decided_by)
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id, decided)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// HeldForDecision is a held message ready to accept: ErrNotFound when it is
// not this project's, ErrMailNotHeld when it is filed.
func (s *Store) HeldForDecision(ctx context.Context, org, project, id string) (MailMessage, error) {
	m, err := s.MailMessageByID(ctx, org, project, id)
	if err != nil {
		return m, err
	}
	if m.Status != "held" {
		return m, ErrMailNotHeld
	}
	return m, nil
}

// DiscardMail deletes a held message and returns its hashes. A filed
// message is never discarded (A7).
func (s *Store) DiscardMail(ctx context.Context, org, project, id string) ([][]byte, error) {
	if !validUUID(org) || !validUUID(project) || !validUUID(id) {
		return nil, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, org, project, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "held" {
		return nil, ErrMailNotHeld
	}
	rows, err := tx.Query(ctx, `SELECT sha256 FROM mail_parts WHERE org_id=$1::uuid AND message_id=$2::uuid AND sha256 IS NOT NULL AND document_id IS NULL`, org, id)
	if err != nil {
		return nil, err
	}
	sums, err := scanHashes(rows)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM mail_messages WHERE org_id=$1::uuid AND id=$2::uuid`, org, id); err != nil {
		return nil, err
	}
	return sums, tx.Commit(ctx)
}

// ProjectArchived reports whether the project is archived; ErrNotFound when
// it is not this org's.
func (s *Store) ProjectArchived(ctx context.Context, org, project string) (bool, error) {
	if !validUUID(org) || !validUUID(project) {
		return false, ErrNotFound
	}
	var archived bool
	err := s.pool.QueryRow(ctx, `SELECT archived_at IS NOT NULL FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, project).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return archived, err
}
