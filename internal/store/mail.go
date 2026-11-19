package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
	// MailDuplicate: the Message-ID of a message already recorded, with
	// other bytes. Held for the PM, never a silent drop (review HIGH-1).
	MailDuplicate = "duplicate"
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
	// Direction is "out" for a team member's own message copied to the
	// project address (correspondence decision 7), else "in".
	Direction string
	// RawSHA256 is the hash of the raw message (or of the parsed post), so
	// a later message with the same Message-ID and other bytes is held.
	RawSHA256 []byte
	// NoRuleLinks: link no firm or package by rule (a rewritten forward;
	// review LOW-1). The thread still links.
	NoRuleLinks bool
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
	// What a sent document was when it went (correspondence, Task 5).
	Number   string `json:"number,omitempty"`
	Revision string `json:"revision,omitempty"`
	Title    string `json:"title,omitempty"`
	SHA256   []byte `json:"-"`
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
	ReplyTo    string     `json:"-"`
	Body       string     `json:"body,omitempty"`
	ReceivedAt time.Time  `json:"received_at"`
	Status     string     `json:"status"`
	HeldReason string     `json:"reason"`
	Parts      []MailPart `json:"parts"`
	// The record (correspondence plan, Task 4). With is the sender for In
	// and the recipients for Out. Replied is computed, never stored.
	Direction  string     `json:"direction"`
	Number     *int       `json:"number,omitempty"`
	With       string     `json:"with"`
	ThreadID   string     `json:"thread_id"`
	FirmID     string     `json:"firm_id,omitempty"`
	Firm       string     `json:"firm,omitempty"`
	PackageID  string     `json:"package_id,omitempty"`
	Package    string     `json:"package,omitempty"`
	LinkedBy   string     `json:"linked_by,omitempty"`
	Replied    bool       `json:"replied"`
	SendStatus string     `json:"send_status,omitempty"`
	SendError  string     `json:"send_error,omitempty"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
	Version    int64      `json:"version"`
	// Notice is Jev's pick shown as a chip: set only when the pick is amber
	// past an approved cut-off and the PM has not acted on it. NoticeRead,
	// NoticeBand and NoticeP are the raw pick, for chat's reads and the key.
	Notice     string   `json:"notice,omitempty"`
	PickState  string   `json:"pick_state,omitempty"`
	NoticeRead string   `json:"-"`
	NoticeBand string   `json:"-"`
	NoticeP    *float64 `json:"-"`
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
	var existingRaw []byte
	err = tx.QueryRow(ctx, `SELECT id::text, raw_sha256 FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND message_id=$3`, org, in.ProjectID, in.MessageID).Scan(&existing, &existingRaw)
	switch {
	case err == nil && (in.RawSHA256 == nil || bytes.Equal(existingRaw, in.RawSHA256)):
		out.Repeat, out.ID = true, existing
		return out, tx.Commit(ctx)
	case err == nil:
		// The same Message-ID with other bytes: held for the PM under its
		// own id, never a silent drop of the later, real one.
		in.HeldReason = MailDuplicate
		in.MessageID = cut(in.MessageID, 900) + " (repeat " + cut(in.Token, 24) + ")"
	case !errors.Is(err, pgx.ErrNoRows):
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
	direction := in.Direction
	if direction != "out" {
		direction = "in"
	}
	link, err := recordLinks(ctx, tx, org, in.ProjectID, out.ID, direction, in.Sender, in.To+","+in.Cc, in.InReplyTo, in.References)
	if err != nil {
		return out, err
	}
	if in.NoRuleLinks {
		link.Firm, link.Package, link.By = "", "", ""
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,sender_name,subject,to_addrs,cc_addrs,in_reply_to,refs,body,received_at,status,held_reason,
 direction,thread_id,firm_id,package_id,linked_by,raw_sha256)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::uuid,NULLIF($19,'')::uuid,NULLIF($20,'')::uuid,$21,$22)`,
		org, out.ID, in.ProjectID, in.MessageID, in.Token, in.Sender, in.SenderName, in.Subject, in.To, in.Cc,
		in.InReplyTo, in.References, in.Body, received, status, reason, direction, link.Thread, link.Firm, link.Package, link.By, in.RawSHA256); err != nil {
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
	// Chat's named filters and ?unanswered=1 (correspondence, Task 4).
	Unanswered bool   // no reply yet
	NotSent    bool   // queued or failed
	FirmID     string // from or to a firm
	PackageID  string // on a package
}

// mailColumns reads a message with its firm and package names and whether
// it was answered: a later filed message from the other direction in its
// thread. One query, no per-row lookups.
const mailColumns = `m.id::text,m.message_id,m.sender,m.sender_name,m.subject,m.to_addrs,m.cc_addrs,m.in_reply_to,m.refs,m.received_at,m.status,m.held_reason,
 m.direction,m.number,m.thread_id::text,COALESCE(m.firm_id::text,'') AS firm_ref,COALESCE(f.name,'') AS firm_name,COALESCE(m.package_id::text,'') AS package_ref,COALESCE(k.title,'') AS package_title,m.linked_by,
 m.send_status,m.send_error,m.sent_at,m.version,m.reply_to,m.notice_read,m.notice_band,m.notice_confidence::float8 AS notice_p,m.pick_state,
 EXISTS(SELECT 1 FROM mail_messages r WHERE r.org_id=m.org_id AND r.project_id=m.project_id AND r.thread_id=m.thread_id
  AND r.direction<>m.direction AND r.status='filed' AND r.received_at>m.received_at AND r.id<>m.id) AS replied`

const mailFrom = ` FROM mail_messages m
LEFT JOIN firms f ON f.org_id=m.org_id AND f.id=m.firm_id
LEFT JOIN packages k ON k.org_id=m.org_id AND k.project_id=m.project_id AND k.id=m.package_id`

func mailDest(m *MailMessage) []any {
	return []any{&m.ID, &m.MessageID, &m.Sender, &m.SenderName, &m.Subject, &m.To, &m.Cc, &m.InReplyTo, &m.References, &m.ReceivedAt, &m.Status, &m.HeldReason,
		&m.Direction, &m.Number, &m.ThreadID, &m.FirmID, &m.Firm, &m.PackageID, &m.Package, &m.LinkedBy,
		&m.SendStatus, &m.SendError, &m.SentAt, &m.Version, &m.ReplyTo, &m.NoticeRead, &m.NoticeBand, &m.NoticeP, &m.PickState, &m.Replied}
}

// finish fills what the screen shows from the raw row.
func (m *MailMessage) finish() {
	m.With = m.Sender
	if m.Direction == "out" {
		m.With = m.To
	}
	if m.NoticeBand == "amber" && m.PickState == "" && m.NoticeRead != "" && m.NoticeRead != "none" {
		m.Notice = m.NoticeRead
	}
}

func scanMailMessage(row pgx.Row, m *MailMessage) error {
	if err := row.Scan(mailDest(m)...); err != nil {
		return err
	}
	m.finish()
	return nil
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
	rows, err := s.pool.Query(ctx, `SELECT * FROM (SELECT `+mailColumns+mailFrom+`
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND (NOT $3 OR m.status='held')
  AND ($4::timestamptz IS NULL OR m.received_at < $4)
  AND ($6='' OR m.firm_id::text=$6) AND ($7='' OR m.package_id::text=$7)
  AND (NOT $8 OR m.send_status IN ('queued','failed'))) x
WHERE NOT $9 OR (NOT x.replied AND x.status='filed')
ORDER BY x.received_at DESC, x.id LIMIT $5`, org, project, f.Held, before, f.Limit, f.FirmID, f.PackageID, f.NotSent, f.Unanswered)
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
	rows, err := s.pool.Query(ctx, `SELECT message_id::text,n,filename,content_type,byte_size,skipped,COALESCE(document_id::text,''),doc_number,doc_revision,doc_title,sha256
FROM mail_parts WHERE org_id=$1::uuid AND message_id::text = ANY($2::text[]) ORDER BY message_id,n`, org, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]MailPart{}
	for rows.Next() {
		var id string
		var p MailPart
		if err := rows.Scan(&id, &p.N, &p.Filename, &p.ContentType, &p.Size, &p.Skipped, &p.DocumentID, &p.Number, &p.Revision, &p.Title, &p.SHA256); err != nil {
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
	err := s.pool.QueryRow(ctx, `SELECT `+mailColumns+`,m.body`+mailFrom+`
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.id=$3::uuid`, org, project, id).Scan(append(mailDest(&m), &m.Body)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	m.finish()
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

// useMessagePick checks a register row's message is a filed message of
// this project, and marks its pick used. A message whose pick was used or
// dismissed can still be added again from its menu (plan, edge cases).
func useMessagePick(ctx context.Context, tx pgx.Tx, org, project, message string) error {
	if !validUUID(message) {
		return fmt.Errorf("%w: the message is not a filed message of this project", ErrInvalidDelivery)
	}
	tag, err := tx.Exec(ctx, `UPDATE mail_messages SET pick_state = CASE WHEN pick_state='' THEN 'used' ELSE pick_state END
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND status='filed'`, org, project, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: the message is not a filed message of this project", ErrInvalidDelivery)
	}
	return nil
}

// MailEdit is the PM's two edits to a message (correspondence plan, Task
// 4): its package and what was done with Jev's pick. Nothing else on a
// message can change. Version is the message's version as read.
type MailEdit struct {
	Version   int64   `json:"version"`
	PackageID *string `json:"package_id"`
	PickState *string `json:"pick_state"`
}

// ErrMailHeld: a held message is not part of the record yet.
var ErrMailHeld = errors.New("a held message can't be edited; accept or discard it")

// EditMail applies one edit at the message's version. A package edit is the
// user's (linked_by user) and survives later replies. A held message is not
// editable. With a change mark on ctx it is captured for Undo.
func (s *Store) EditMail(ctx context.Context, org, project, id string, e MailEdit) (MailMessage, error) {
	if !validUUID(org) || !validUUID(project) || !validUUID(id) {
		return MailMessage{}, ErrNotFound
	}
	if e.PackageID == nil && e.PickState == nil {
		return MailMessage{}, fmt.Errorf("%w: name the package or the pick", ErrInvalidInput)
	}
	if e.PickState != nil && *e.PickState != "" && *e.PickState != "used" && *e.PickState != "dismissed" {
		return MailMessage{}, fmt.Errorf("%w: the pick is used, dismissed or blank", ErrInvalidInput)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MailMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, org, project); err != nil {
		return MailMessage{}, err
	}
	var status string
	var version int64
	err = tx.QueryRow(ctx, `SELECT status, version FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, org, project, id).Scan(&status, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return MailMessage{}, ErrNotFound
	}
	if err != nil {
		return MailMessage{}, err
	}
	if status != "filed" {
		return MailMessage{}, ErrMailHeld
	}
	if e.Version != version {
		return MailMessage{}, ErrVersionConflict
	}
	if e.PackageID != nil && *e.PackageID != "" {
		var ok bool
		if !validUUID(*e.PackageID) {
			return MailMessage{}, ErrNotFound
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid)`, org, project, *e.PackageID).Scan(&ok); err != nil {
			return MailMessage{}, err
		}
		if !ok {
			return MailMessage{}, ErrNotFound
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE mail_messages SET
 package_id = CASE WHEN $4::boolean THEN NULLIF($5,'')::uuid ELSE package_id END,
 linked_by = CASE WHEN $4::boolean THEN (CASE WHEN $5='' THEN '' ELSE 'user' END) ELSE linked_by END,
 pick_state = CASE WHEN $6::boolean THEN $7 ELSE pick_state END,
 version = version + 1
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid`, org, project, id,
		e.PackageID != nil, deref(e.PackageID), e.PickState != nil, deref(e.PickState)); err != nil {
		return MailMessage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MailMessage{}, err
	}
	return s.MailMessageByID(ctx, org, project, id)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// MailPrecheck is what RecordMail would refuse or cap, read before any byte
// is stored: a repeat token or Message-Id, and held mail already full. The
// lock is RecordMail's; this read only spares the bytes (review H2).
type MailPrecheck struct{ Repeat, HeldFull bool }

func (s *Store) MailPrecheck(ctx context.Context, org, project, token, messageID string, raw []byte) (MailPrecheck, error) {
	var out MailPrecheck
	if !validUUID(org) || !validUUID(project) {
		return out, ErrNotFound
	}
	var count int
	var held int64
	// A repeat is the same token, or the same Message-ID with the same
	// bytes; the same Message-ID with other bytes is recorded and held.
	err := s.pool.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM mail_webhook_tokens WHERE token=$3)
  OR EXISTS(SELECT 1 FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND message_id=$4 AND ($5::bytea IS NULL OR raw_sha256=$5)),
 (SELECT count(*) FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND status='held'),
 (SELECT COALESCE(sum(p.byte_size),0)::bigint FROM mail_messages m JOIN mail_parts p ON p.org_id=m.org_id AND p.message_id=m.id
   WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.status='held')`, org, project, token, messageID, raw).Scan(&out.Repeat, &count, &held)
	out.HeldFull = count >= MailHeldMax || held >= MailHeldBytes
	return out, err
}

// StageMailBlob notes bytes one request stored before a part names them.
func (s *Store) StageMailBlob(ctx context.Context, sum []byte, request string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO mail_blob_stages(sha256,request) VALUES ($1,$2) ON CONFLICT (sha256,request) DO UPDATE SET staged_at=now()`, sum, request)
	return err
}

// UnstageMailBlob clears one request's staging of a hash, never another's.
func (s *Store) UnstageMailBlob(ctx context.Context, sum []byte, request string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM mail_blob_stages WHERE sha256=$1 AND request=$2`, sum, request)
	return err
}

// StagedBlob is one staging row.
type StagedBlob struct {
	SHA256  []byte
	Request string
}

// StaleMailBlobs are stagings older than cutoff. Not a tenant read: a hash
// names no org.
func (s *Store) StaleMailBlobs(ctx context.Context, cutoff time.Time) ([]StagedBlob, error) {
	rows, err := s.pool.Query(ctx, `SELECT sha256, request FROM mail_blob_stages WHERE staged_at < $1 ORDER BY staged_at LIMIT 500`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StagedBlob
	for rows.Next() {
		var b StagedBlob
		if err := rows.Scan(&b.SHA256, &b.Request); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// MailBlobStagedByOthers reports whether a request other than this one
// still stages a hash.
func (s *Store) MailBlobStagedByOthers(ctx context.Context, sum []byte, request string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mail_blob_stages WHERE sha256=$1 AND request<>$2)`, sum, request).Scan(&ok)
	return ok, err
}

// MailAttachment reports whether the document came attached to an email
// (a part after the body).
func (s *Store) MailAttachment(ctx context.Context, org, document string) (bool, error) {
	if !validUUID(org) || !validUUID(document) {
		return false, nil
	}
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mail_parts WHERE org_id=$1::uuid AND document_id=$2::uuid AND n>0)`, org, document).Scan(&ok)
	return ok, err
}

// MailMessageByRef finds one message of this project by its id, or a short
// id of six or more characters that names exactly one, in SQL (review L7).
func (s *Store) MailMessageByRef(ctx context.Context, org, project, ref string) (MailMessage, error) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	if !validUUID(org) || !validUUID(project) || len(ref) < 6 || len(ref) > 36 || strings.Trim(ref, "0123456789abcdef-") != "" {
		return MailMessage{}, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid
 AND (id::text=$3 OR id::text LIKE $3 || '%') LIMIT 2`, org, project, ref)
	if err != nil {
		return MailMessage{}, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return MailMessage{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return MailMessage{}, err
	}
	if len(ids) != 1 {
		return MailMessage{}, ErrNotFound
	}
	return s.MailMessageByID(ctx, org, project, ids[0])
}
