package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// The one send path's records (docs/plans/2026-11-17-correspondence-
// transmittals.md, Task 3). A transmittal is an Out message: numbered once
// under the project lock, its parts the sheet then the documents as they
// were, queued for the outbox. A sent message is frozen like any other.

// Send limits (owner decisions 3 and 6).
const (
	MailSendMaxBytes     = 10 << 20
	MailSendPerDay       = 100
	MailRecipientsPerDay = 500
	// A free plan's day (security review H3; the owner's to change).
	MailFreeSendPerDay       = 10
	MailFreeRecipientsPerDay = 50
	// MailGlobalPerDay is every org's sends together in a day, unless the
	// setting names another (security review H3).
	MailGlobalPerDay      = 2000
	MailMaxRecipients     = 20
	MailSendAttempts      = 5
	mailSendErrorMaxBytes = 300
)

var (
	// ErrMailTooLarge: the documents and files add up to more than 10 MB.
	ErrMailTooLarge = errors.New("the documents are larger than 10 MB")
	// ErrMailCap: the org sent its messages or recipients for today.
	ErrMailCap = errors.New("daily send cap reached")
	// ErrMailGlobalCap: SiteWise sent all it sends in a day, every org together.
	ErrMailGlobalCap = errors.New("global daily send cap reached")
	// ErrProjectArchived: an archived project sends nothing.
	ErrProjectArchived = errors.New("project is archived")
	// ErrMailNotRetryable: retry named a message that is not failed.
	ErrMailNotRetryable = errors.New("only a message that was not sent can be sent again")
)

// MailTooLarge carries the size the PM is told.
type MailTooLarge struct{ Bytes int64 }

func (e MailTooLarge) Error() string { return ErrMailTooLarge.Error() }
func (e MailTooLarge) Unwrap() error { return ErrMailTooLarge }

// OutboundFile is bytes already stored that go with a message: the sheet,
// or an issued report's frozen PDF.
type OutboundFile struct {
	Name, ContentType string
	SHA256            []byte
	Size              int64
}

// SentDocument is one document as it is sent: its identity as the
// repository shows it (corrections win), its bytes by hash.
type SentDocument struct {
	ID, Filename, Number, Revision, Title, MediaType string
	SHA256                                           []byte
	Size                                             int64
}

// Outbound is one message to send. Callers (the transmittal route and plans
// 3, 4 and 5) keep the returned message's id on their own record.
type Outbound struct {
	ProjectID, SentBy string
	From              string // the From address as written
	To, Cc, ReplyTo   []string
	Subject, Body     string
	DocumentIDs       []string
	Files             []OutboundFile
	InReplyTo         string // a message of this project, by its id
	PackageID         string
	MessageIDDomain   string
	// Limits for this org's day; zero values are MailSendPerDay and
	// MailRecipientsPerDay. Global is every org's sends together.
	PerDay, RecipientsPerDay, Global int
	// Sheet renders the transmittal sheet once the number is known, inside
	// the project lock, and returns its stored bytes. Nil sends no sheet.
	Sheet func(number int, docs []SentDocument) (OutboundFile, error)
}

// QueueOutbound records one message to send, all or nothing, under the
// project lock: the project is live, the org is inside its daily cap,
// every document is a filed document of this project, the total is at
// most 10 MB, the reply is to a message of this project. The number is
// max+1, backed by a unique index.
func (s *Store) QueueOutbound(ctx context.Context, org string, in Outbound, now time.Time) (MailMessage, error) {
	if !validUUID(org) || !validUUID(in.ProjectID) {
		return MailMessage{}, ErrNotFound
	}
	if n := len(in.To) + len(in.Cc); len(in.To) == 0 || n > MailMaxRecipients+1 { // the PM is copied in beside the 20
		return MailMessage{}, fmt.Errorf("%w: send to between 1 and %d addresses", ErrInvalidInput, MailMaxRecipients)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MailMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProject(ctx, tx, org, in.ProjectID); err != nil {
		return MailMessage{}, err
	}
	var archived bool
	err = tx.QueryRow(ctx, `SELECT archived_at IS NOT NULL FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, org, in.ProjectID).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return MailMessage{}, ErrNotFound
	}
	if err != nil {
		return MailMessage{}, err
	}
	if archived {
		return MailMessage{}, ErrProjectArchived
	}
	// The cap is per org: take the org's own lock so two projects cannot
	// both pass at the limit.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('mail-send/' || $1, 0))`, org); err != nil {
		return MailMessage{}, err
	}
	var sentToday, recipientsToday int
	if err := tx.QueryRow(ctx, `SELECT count(*), COALESCE(sum(cardinality(string_to_array(NULLIF(to_addrs,''),',')) + COALESCE(cardinality(string_to_array(NULLIF(cc_addrs,''),',')),0)),0)
FROM mail_messages WHERE org_id=$1::uuid AND number IS NOT NULL AND received_at > $2::timestamptz - interval '1 day'`, org, now).Scan(&sentToday, &recipientsToday); err != nil {
		return MailMessage{}, err
	}
	perDay, recipients, global := in.PerDay, in.RecipientsPerDay, in.Global
	if perDay <= 0 {
		perDay = MailSendPerDay
	}
	if recipients <= 0 {
		recipients = MailRecipientsPerDay
	}
	if global <= 0 {
		global = MailGlobalPerDay
	}
	if sentToday >= perDay || recipientsToday+len(in.To)+len(in.Cc) > recipients {
		return MailMessage{}, ErrMailCap
	}
	// Every org together: SiteWise is one sending domain (review H3).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('mail-send/all', 0))`); err != nil {
		return MailMessage{}, err
	}
	var all int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM mail_messages WHERE number IS NOT NULL AND received_at > $1::timestamptz - interval '1 day'`, now).Scan(&all); err != nil {
		return MailMessage{}, err
	}
	if all >= global {
		return MailMessage{}, ErrMailGlobalCap
	}
	docs, err := sentDocuments(ctx, tx, org, in.ProjectID, in.DocumentIDs)
	if err != nil {
		return MailMessage{}, err
	}
	var total int64
	for _, d := range docs {
		total += d.Size
	}
	for _, f := range in.Files {
		total += f.Size
	}
	if total > MailSendMaxBytes {
		return MailMessage{}, MailTooLarge{Bytes: total}
	}
	inReplyTo, refs := "", ""
	if in.InReplyTo != "" {
		if !validUUID(in.InReplyTo) {
			return MailMessage{}, ErrNotFound
		}
		var parentID, parentRefs string
		err := tx.QueryRow(ctx, `SELECT message_id, refs FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND status='filed'`,
			org, in.ProjectID, in.InReplyTo).Scan(&parentID, &parentRefs)
		if errors.Is(err, pgx.ErrNoRows) {
			return MailMessage{}, ErrNotFound
		}
		if err != nil {
			return MailMessage{}, err
		}
		inReplyTo = parentID
		refs = strings.TrimSpace(parentRefs + " " + parentID)
		if f := strings.Fields(refs); len(f) > 40 {
			refs = strings.Join(f[len(f)-40:], " ")
		}
	}
	if in.PackageID != "" {
		var ok bool
		if !validUUID(in.PackageID) {
			return MailMessage{}, ErrNotFound
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid)`, org, in.ProjectID, in.PackageID).Scan(&ok); err != nil {
			return MailMessage{}, err
		}
		if !ok {
			return MailMessage{}, ErrNotFound
		}
	}
	var number int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(number),0)+1 FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, in.ProjectID).Scan(&number); err != nil {
		return MailMessage{}, err
	}
	files := in.Files
	if in.Sheet != nil {
		sheet, err := in.Sheet(number, docs)
		if err != nil {
			return MailMessage{}, err
		}
		files = append([]OutboundFile{sheet}, files...)
	}
	id := newID()
	domain := in.MessageIDDomain
	if domain == "" {
		domain = "sitewise.invalid"
	}
	messageID := "<" + id + "@" + domain + ">"
	to, cc := strings.Join(in.To, ", "), strings.Join(in.Cc, ", ")
	link, err := recordLinks(ctx, tx, org, in.ProjectID, id, "out", in.From, to+","+cc, inReplyTo, refs)
	if err != nil {
		return MailMessage{}, err
	}
	pkg, by := link.Package, link.By
	if in.PackageID != "" {
		pkg, by = in.PackageID, "user"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mail_messages(org_id,id,project_id,message_id,webhook_token,sender,subject,to_addrs,cc_addrs,in_reply_to,refs,body,received_at,status,
 direction,thread_id,firm_id,package_id,linked_by,number,send_status,decided_by,send_tried_at,reply_to)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULL,$5,$6,$7,$8,$9,$10,$11,$12,'filed','out',$13::uuid,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,$16,$17,'queued',NULLIF($18,'')::uuid,$12,$19)`,
		org, id, in.ProjectID, messageID, in.From, in.Subject, to, cc, inReplyTo, refs, in.Body, now,
		link.Thread, link.Firm, pkg, by, number, in.SentBy, strings.Join(in.ReplyTo, ", ")); err != nil {
		return MailMessage{}, err
	}
	n := 0
	for _, f := range files {
		if _, err := tx.Exec(ctx, `INSERT INTO mail_parts(org_id,message_id,n,filename,content_type,sha256,byte_size) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7)`,
			org, id, n, f.Name, f.ContentType, f.SHA256, f.Size); err != nil {
			return MailMessage{}, err
		}
		n++
	}
	if n == 0 {
		n = 1 // part 0 is the sheet's place even when a caller sends none
	}
	for _, d := range docs {
		if _, err := tx.Exec(ctx, `INSERT INTO mail_parts(org_id,message_id,n,filename,content_type,sha256,byte_size,document_id,doc_number,doc_revision,doc_title)
VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8::uuid,$9,$10,$11)`,
			org, id, n, d.Filename, d.MediaType, d.SHA256, d.Size, d.ID, cut(d.Number, 200), cut(d.Revision, 50), cut(d.Title, 500)); err != nil {
			return MailMessage{}, err
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return MailMessage{}, err
	}
	return s.MailMessageByID(ctx, org, in.ProjectID, id)
}

func cut(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

// sentDocuments reads the documents to send with the repository's identity
// (corrections win). Every id must be a filed document of this project,
// once; anything else is ErrNotFound without saying which, as the report
// transmittal refuses foreign ids.
func sentDocuments(ctx context.Context, tx pgx.Tx, org, project string, ids []string) ([]SentDocument, error) {
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.ToLower(id)
		if !validUUID(id) {
			return nil, ErrNotFound
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: a document is listed twice", ErrInvalidInput)
		}
		seen[id] = true
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `SELECT d.id::text,d.filename,
 COALESCE(NULLIF((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='number'),''),d.document_number,''),
 COALESCE(NULLIF((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='revision'),''),d.revision,''),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='title'),''),
 f.media_type,f.sha256,f.byte_size
FROM unnest($3::text[]) WITH ORDINALITY AS w(id,ord)
JOIN documents d ON d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id::text=w.id AND d.status='filed'
JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
ORDER BY w.ord`, org, project, lower(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SentDocument
	for rows.Next() {
		var d SentDocument
		if err := rows.Scan(&d.ID, &d.Filename, &d.Number, &d.Revision, &d.Title, &d.MediaType, &d.SHA256, &d.Size); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) != len(ids) {
		return nil, ErrNotFound
	}
	return out, nil
}

func lower(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strings.ToLower(id)
	}
	return out
}

// OutboundRef names one queued message.
type OutboundRef struct{ OrgID, ProjectID, ID string }

// QueuedOutbound lists queued messages last tried before cutoff (zero: all
// queued), oldest first, for the outbox's sweep. Not a tenant read: each
// ref carries its own org.
func (s *Store) QueuedOutbound(ctx context.Context, cutoff time.Time) ([]OutboundRef, error) {
	var before *time.Time
	if !cutoff.IsZero() {
		before = &cutoff
	}
	rows, err := s.pool.Query(ctx, `SELECT org_id::text,project_id::text,id::text FROM mail_messages
WHERE send_status='queued' AND ($1::timestamptz IS NULL OR COALESCE(send_tried_at,received_at) < $1) ORDER BY received_at LIMIT 200`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboundRef
	for rows.Next() {
		var r OutboundRef
		if err := rows.Scan(&r.OrgID, &r.ProjectID, &r.ID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RecordSendAttempt writes one attempt's outcome: sent, still queued after
// a failure, or failed after the last attempt. errText is the service's
// short reason, never shown beyond the PM's own project.
func (s *Store) RecordSendAttempt(ctx context.Context, org, id string, sent bool, errText string, now time.Time) (string, error) {
	if len(errText) > mailSendErrorMaxBytes {
		errText = errText[:mailSendErrorMaxBytes]
	}
	var status string
	err := s.pool.QueryRow(ctx, `UPDATE mail_messages SET
 send_attempts = send_attempts + 1,
 send_tried_at = $3,
 send_status = CASE WHEN $4 THEN 'sent' WHEN send_attempts + 1 >= $6 THEN 'failed' ELSE 'queued' END,
 sent_at = CASE WHEN $4 THEN $3 ELSE sent_at END,
 send_error = CASE WHEN $4 THEN '' ELSE $5 END
WHERE org_id=$1::uuid AND id=$2::uuid AND send_status='queued'
RETURNING send_status`, org, id, now, sent, errText, MailSendAttempts).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return status, err
}

// RetryOutbound queues a message that was not sent again, from its first
// attempt. Only the PM's press of Retry calls it.
func (s *Store) RetryOutbound(ctx context.Context, org, project, id string) error {
	if !validUUID(org) || !validUUID(project) || !validUUID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE mail_messages SET send_status='queued', send_attempts=0, send_error='', send_tried_at=now()
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND send_status='failed'`, org, project, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid)`, org, project, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrMailNotRetryable
}

// OutboundThread is the In-Reply-To and References a queued message goes
// with, as recorded when it was queued.
func (m MailMessage) OutboundThread() (inReplyTo string, references []string) {
	return m.InReplyTo, strings.Fields(m.References)
}

// MailSender is who a message goes out as: the org's and project's names,
// and the sending member's email and whether it is confirmed.
type MailSender struct {
	OrgName, ProjectName, Email string
	Verified                    bool
	// UserSince is when the sending user's account was made.
	UserSince time.Time
}

// MailSenderContext reads the sender's names in one query.
func (s *Store) MailSenderContext(ctx context.Context, org, project, user string) (MailSender, error) {
	var out MailSender
	if !validUUID(org) || !validUUID(project) || !validUUID(user) {
		return out, ErrNotFound
	}
	err := s.pool.QueryRow(ctx, `SELECT o.name, p.name, u.email, u.email_verified, u.created_at FROM orgs o
JOIN projects p ON p.org_id=o.id AND p.id=$2::uuid
JOIN users u ON u.org_id=o.id AND u.id=$3::uuid
WHERE o.id=$1::uuid`, org, project, user).Scan(&out.OrgName, &out.ProjectName, &out.Email, &out.Verified, &out.UserSince)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

// IssuedVersion is an issued version of one of this project's reports, for
// Send: its frozen PDF goes as it was issued (the report plan's Send issue
// N…). ErrNotFound for anything else.
type IssuedVersion struct {
	ID, ReportID, Title string
	Number              int
	SHA256              []byte
}

func (s *Store) IssuedVersionOf(ctx context.Context, org, project, version string) (IssuedVersion, error) {
	var v IssuedVersion
	if !validUUID(org) || !validUUID(project) || !validUUID(version) {
		return v, ErrNotFound
	}
	err := s.pool.QueryRow(ctx, `SELECT v.id::text, r.id::text, r.title, v.number, v.export_file_sha256 FROM report_versions v
JOIN reports r ON r.org_id=v.org_id AND r.id=v.report_id AND r.project_id=$2::uuid
WHERE v.org_id=$1::uuid AND v.id=$3::uuid AND v.status='issued' AND v.export_file_sha256 IS NOT NULL`, org, project, version).Scan(&v.ID, &v.ReportID, &v.Title, &v.Number, &v.SHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

// MailFirm finds one of the org's firms by its id, an id prefix of six or
// more characters, or its exact name, for chat's Send. Two matches or none
// is ErrNotFound.
func (s *Store) MailFirm(ctx context.Context, org, ref string) (Firm, error) {
	var f Firm
	ref = strings.TrimSpace(ref)
	if !validUUID(org) || ref == "" {
		return f, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,name,email FROM firms WHERE org_id=$1::uuid
 AND (id::text=lower($2) OR (length($2)>=6 AND id::text LIKE lower($2)||'%') OR lower(name)=lower($2)) LIMIT 2`, org, ref)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		if err := rows.Scan(&f.ID, &f.Name, &f.Email); err != nil {
			return f, err
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return f, err
	}
	if n != 1 {
		return Firm{}, ErrNotFound
	}
	return f, nil
}

// RFIToSend is an RFI of this project, for Send: its number and title.
func (s *Store) RFIToSend(ctx context.Context, org, project, item string) (int, string, error) {
	if !validUUID(org) || !validUUID(project) || !validUUID(item) {
		return 0, "", ErrNotFound
	}
	var n int
	var title string
	err := s.pool.QueryRow(ctx, `SELECT number, title FROM project_delivery_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND kind='rfi' AND retired_at IS NULL`, org, project, item).Scan(&n, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	return n, title, err
}

// LinkRFIMessage records the message an RFI went out on, so a reply in its
// thread answers it by rule. The RFI's status stays the PM's.
func (s *Store) LinkRFIMessage(ctx context.Context, org, project, item, message string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE project_delivery_items SET details = details || jsonb_build_object('sent_message_id', $4::text), version = version + 1
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND kind='rfi'
  AND EXISTS(SELECT 1 FROM mail_messages m WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.id=$4::uuid AND m.direction='out')`, org, project, item, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
