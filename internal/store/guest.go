package store

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

// Guest access by link (docs/plans/2026-11-17-guest-access.md). A grant lets
// one person outside the org see one shared thing: a set of documents, the
// issued versions of a report, or a package's issued RFP or RFT, where an
// invited firm may also send its quote. A grant is not a user, a membership
// or a session, so no member route ever admits it. Its token is kept only as
// a SHA-256.

// Grant kinds.
const (
	GuestDocuments = "documents"
	GuestReport    = "report"
	GuestPackage   = "package"
)

// Link life in days (owner decision 2, the plan's default).
const (
	GuestDefaultDays        = 30
	GuestDefaultPackageDays = 21
	GuestMaxDays            = 180
	GuestMaxDocuments       = 200
)

var (
	// ErrGuestGone is every reason a link no longer works: unknown, revoked,
	// expired, its project archived or its package retired. One error, so
	// the answer is the same for all of them.
	ErrGuestGone = errors.New("guest link expired or withdrawn")
	// ErrGuestAwarded refuses a quote on a package already awarded.
	ErrGuestAwarded = errors.New("this tender has been awarded")
)

// GuestGrantInput is what the PM's Share dialog sends.
type GuestGrantInput struct {
	Kind        string   `json:"kind"`
	SharedWith  string   `json:"shared_with"`
	DocumentIDs []string `json:"document_ids"`
	ReportID    string   `json:"report_id"`
	PackageID   string   `json:"package_id"`
	FirmID      string   `json:"firm_id"`
	Days        int      `json:"days"`
	// Replaces is an older link to revoke in the same transaction ("New link").
	Replaces string `json:"replaces"`
}

// GuestGrant is one link as the guest routes check it. It never holds the token.
type GuestGrant struct {
	ID          string
	OrgID       string
	ProjectID   string
	Kind        string
	SharedWith  string
	DocumentIDs []string
	ReportID    string
	PackageID   string
	FirmID      string
	CreatedBy   string
	ExpiresAt   time.Time
}

// ShareView is one link as the PM's Share list and chat's read show it: who
// it was shared with, by whom, how often it was opened, when it ends. No token.
type ShareView struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	SharedWith   string     `json:"shared_with"`
	SharedBy     string     `json:"shared_by"`
	What         string     `json:"what"`
	DocumentIDs  []string   `json:"document_ids"`
	ReportID     string     `json:"report_id,omitempty"`
	PackageID    string     `json:"package_id,omitempty"`
	FirmID       string     `json:"firm_id,omitempty"`
	FirmName     string     `json:"firm_name,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	Opens        int        `json:"opens"`
	LastOpenedAt *time.Time `json:"last_opened_at,omitempty"`
	Live         bool       `json:"live"`
	ExpiresOn    string     `json:"expires_on"`
	LastOpenedOn string     `json:"last_opened_on,omitempty"`
}

// CreateGuestGrant makes one link in one transaction. Every target is
// checked against this org and project: a document of another project, a
// report with no issue, an uninvited firm or a package with no issued RFP
// or RFT is ErrNotFound, without saying which, and nothing is written.
func (s *Store) CreateGuestGrant(ctx context.Context, org, project, actor string, in GuestGrantInput, tokenHash []byte, now time.Time) (GuestGrant, error) {
	in.SharedWith = strings.TrimSpace(in.SharedWith)
	if len(tokenHash) != 32 || in.SharedWith == "" || len(in.SharedWith) > 254 || strings.ContainsAny(in.SharedWith, "\r\n") {
		return GuestGrant{}, ErrInvalidInput
	}
	days := in.Days
	if days == 0 {
		days = GuestDefaultDays
		if in.Kind == GuestPackage {
			days = GuestDefaultPackageDays
		}
	}
	if days < 1 || days > GuestMaxDays {
		return GuestGrant{}, ErrInvalidInput
	}
	if !validUUID(project) || in.Replaces != "" && !validUUID(in.Replaces) {
		return GuestGrant{}, ErrNotFound
	}
	g := GuestGrant{ID: newID(), OrgID: org, ProjectID: project, Kind: in.Kind, SharedWith: in.SharedWith, CreatedBy: actor, ExpiresAt: now.Add(time.Duration(days) * 24 * time.Hour), DocumentIDs: []string{}}
	switch in.Kind {
	case GuestDocuments:
		g.DocumentIDs = unique(in.DocumentIDs)
		if len(g.DocumentIDs) == 0 || len(g.DocumentIDs) > GuestMaxDocuments || in.ReportID != "" || in.PackageID != "" || in.FirmID != "" {
			return GuestGrant{}, ErrInvalidInput
		}
		for _, id := range g.DocumentIDs {
			if !validUUID(id) {
				return GuestGrant{}, ErrNotFound
			}
		}
	case GuestReport:
		if len(in.DocumentIDs) > 0 || in.PackageID != "" || in.FirmID != "" {
			return GuestGrant{}, ErrInvalidInput
		}
		if !validUUID(in.ReportID) {
			return GuestGrant{}, ErrNotFound
		}
		g.ReportID = in.ReportID
	case GuestPackage:
		if len(in.DocumentIDs) > 0 || in.ReportID != "" {
			return GuestGrant{}, ErrInvalidInput
		}
		if !validUUID(in.PackageID) || !validUUID(in.FirmID) {
			return GuestGrant{}, ErrNotFound
		}
		g.PackageID, g.FirmID = in.PackageID, in.FirmID
	default:
		return GuestGrant{}, ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GuestGrant{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return GuestGrant{}, err
	}
	var live bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid AND archived_at IS NULL)`, org, project).Scan(&live); err != nil {
		return GuestGrant{}, err
	}
	if !live {
		return GuestGrant{}, ErrNotFound
	}
	var ok bool
	switch g.Kind {
	case GuestDocuments:
		// As transmittalSelection: all filed in this project, or none.
		var n int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND status='filed' AND id::text=ANY($3::text[])`, org, project, g.DocumentIDs).Scan(&n)
		ok = n == len(g.DocumentIDs)
	case GuestReport:
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.id=$3::uuid)`, org, project, g.ReportID).Scan(&ok)
	case GuestPackage:
		// As InviteFirm checks: a live package with the firm invited on it,
		// and an issued RFP or RFT to send.
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_invitations i
JOIN packages p ON p.org_id=i.org_id AND p.project_id=i.project_id AND p.id=i.package_id AND p.retired_at IS NULL
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.package_id=$3::uuid AND i.firm_id=$4::uuid)`, org, project, g.PackageID, g.FirmID).Scan(&ok)
		if err == nil && ok {
			var report string
			report, err = packageTenderReport(ctx, tx, org, project, g.PackageID)
			ok = report != ""
		}
	}
	if err != nil {
		return GuestGrant{}, err
	}
	if !ok {
		return GuestGrant{}, ErrNotFound
	}
	if in.Replaces != "" {
		tag, err := tx.Exec(ctx, `UPDATE guest_grants SET revoked_at=$4 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND revoked_at IS NULL`, org, project, in.Replaces, now)
		if err != nil {
			return GuestGrant{}, err
		}
		if tag.RowsAffected() == 0 {
			return GuestGrant{}, ErrNotFound
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO guest_grants (org_id,id,project_id,token_hash,kind,shared_with,document_ids,report_id,package_id,firm_id,created_by,created_at,expires_at)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7::uuid[],NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11::uuid,$12,$13)`,
		org, g.ID, project, tokenHash, g.Kind, g.SharedWith, g.DocumentIDs, g.ReportID, g.PackageID, g.FirmID, actor, now, g.ExpiresAt)
	if err != nil {
		return GuestGrant{}, err
	}
	if g.Kind == GuestPackage {
		// The firm's cell shows the link at once.
		if err := invitationEvent(ctx, tx, s, org, project, g.PackageID, "linked"); err != nil {
			return GuestGrant{}, err
		}
	}
	return g, tx.Commit(ctx)
}

// packageTenderReport is the id of a package's RFP or RFT report that has an
// issue, or "". packageRFPReport looks for kind rfp only; a trade package's
// RFT is kind rft.
func packageTenderReport(ctx context.Context, tx pgx.Tx, org, project, pkg string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT r.id::text FROM reports r
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.package_id=$3::uuid AND r.kind IN ('rfp','rft')
  AND EXISTS(SELECT 1 FROM report_versions v WHERE v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued')
ORDER BY r.created_at, r.id LIMIT 1`, org, project, pkg).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// liveGrantSQL is what makes a grant work at time $2: not revoked, not
// expired, its project not archived, a package link's package not retired.
const liveGrantSQL = `g.revoked_at IS NULL AND g.expires_at > $2
 AND EXISTS(SELECT 1 FROM projects p WHERE p.org_id=g.org_id AND p.id=g.project_id AND p.archived_at IS NULL)
 AND (g.kind<>'package' OR EXISTS(SELECT 1 FROM packages k WHERE k.org_id=g.org_id AND k.project_id=g.project_id AND k.id=g.package_id AND k.retired_at IS NULL))`

const grantColumns = `g.id::text,g.org_id::text,g.project_id::text,g.kind,g.shared_with,g.document_ids::text[],COALESCE(g.report_id::text,''),COALESCE(g.package_id::text,''),COALESCE(g.firm_id::text,''),g.created_by::text,g.expires_at,g.token_hash`

// GuestGrantByToken is the live grant whose token hashes to hash, in one
// query. Unknown, revoked, expired, archived and retired are all
// ErrGuestGone, so the answer reveals nothing about which.
func (s *Store) GuestGrantByToken(ctx context.Context, hash []byte, now time.Time) (GuestGrant, error) {
	if len(hash) != 32 {
		return GuestGrant{}, ErrGuestGone
	}
	var g GuestGrant
	var stored []byte
	err := s.pool.QueryRow(ctx, `SELECT `+grantColumns+` FROM guest_grants g WHERE g.token_hash=$1 AND `+liveGrantSQL, hash, now).
		Scan(&g.ID, &g.OrgID, &g.ProjectID, &g.Kind, &g.SharedWith, &g.DocumentIDs, &g.ReportID, &g.PackageID, &g.FirmID, &g.CreatedBy, &g.ExpiresAt, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuestGrant{}, ErrGuestGone
	}
	if err != nil {
		return GuestGrant{}, err
	}
	// The index finds the row; the comparison that admits it takes the same
	// time whatever the bytes.
	if subtle.ConstantTimeCompare(stored, hash) != 1 {
		return GuestGrant{}, ErrGuestGone
	}
	return g, nil
}

// TouchGuestGrant records an open: opened counts a link exchange (a click on
// the link); every page read moves last opened.
func (s *Store) TouchGuestGrant(ctx context.Context, g GuestGrant, opened bool, now time.Time) error {
	add := 0
	if opened {
		add = 1
	}
	_, err := s.pool.Exec(ctx, `UPDATE guest_grants SET opens=opens+$3, last_opened_at=$4 WHERE org_id=$1::uuid AND id=$2::uuid`, g.OrgID, g.ID, add, now)
	if err != nil || !opened || g.Kind != GuestPackage {
		return err
	}
	// The PM's cell moves to "link opened" without a reload.
	payload, _ := json.Marshal(map[string]string{"project_id": g.ProjectID, "package_id": g.PackageID, "state": "link_opened"})
	_, err = appendEvent(ctx, s.q, g.OrgID, EventProcurement, "", string(payload))
	return err
}

// ShareFilter narrows the list to one shared thing; empty lists every link.
type ShareFilter struct {
	Kind, ReportID, PackageID, FirmID, DocumentID string
}

// ListGuestGrants is the project's links, newest first, with who made each.
func (s *Store) ListGuestGrants(ctx context.Context, org, project string, f ShareFilter, now time.Time) ([]ShareView, error) {
	if !validUUID(project) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT g.id::text,g.kind,g.shared_with,COALESCE(u.email,''),g.document_ids::text[],COALESCE(g.report_id::text,''),COALESCE(g.package_id::text,''),COALESCE(g.firm_id::text,''),
 COALESCE(f.name,''),COALESCE(r.title,''),COALESCE(k.title,''),g.created_at,g.expires_at,g.revoked_at,g.opens,g.last_opened_at,
 (`+liveGrantSQL+`)
FROM guest_grants g
LEFT JOIN users u ON u.org_id=g.org_id AND u.id=g.created_by
LEFT JOIN firms f ON f.org_id=g.org_id AND f.id=g.firm_id
LEFT JOIN reports r ON r.org_id=g.org_id AND r.id=g.report_id
LEFT JOIN packages k ON k.org_id=g.org_id AND k.id=g.package_id
WHERE g.org_id=$1::uuid AND g.project_id=$3::uuid
  AND ($4='' OR g.kind=$4) AND ($5='' OR g.report_id::text=$5) AND ($6='' OR g.package_id::text=$6)
  AND ($7='' OR g.firm_id::text=$7) AND ($8='' OR $8=ANY(g.document_ids::text[]))
ORDER BY g.created_at DESC, g.id LIMIT 500`, org, now, project, f.Kind, f.ReportID, f.PackageID, f.FirmID, f.DocumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ShareView{}
	for rows.Next() {
		var v ShareView
		var report, pkg string
		if err := rows.Scan(&v.ID, &v.Kind, &v.SharedWith, &v.SharedBy, &v.DocumentIDs, &v.ReportID, &v.PackageID, &v.FirmID, &v.FirmName, &report, &pkg, &v.CreatedAt, &v.ExpiresAt, &v.RevokedAt, &v.Opens, &v.LastOpenedAt, &v.Live); err != nil {
			return nil, err
		}
		switch v.Kind {
		case GuestDocuments:
			v.What = countWords(len(v.DocumentIDs), "document")
		case GuestReport:
			v.What = report
		default:
			v.What = pkg
		}
		v.ExpiresOn = SydneyDate(v.ExpiresAt)
		if v.LastOpenedAt != nil {
			v.LastOpenedOn = SydneyDate(*v.LastOpenedAt)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func countWords(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}

// RevokeGuestGrant withdraws a link at once; the next request on it is
// refused. Revoking a revoked link changes nothing. A link cannot be put back.
func (s *Store) RevokeGuestGrant(ctx context.Context, org, project, id string, now time.Time) (ShareView, error) {
	if !validUUID(project) || !validUUID(id) {
		return ShareView{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ShareView{}, err
	}
	defer tx.Rollback(ctx)
	var revoked *time.Time
	var pkg string
	err = tx.QueryRow(ctx, `SELECT revoked_at, COALESCE(package_id::text,'') FROM guest_grants WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, org, project, id).Scan(&revoked, &pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShareView{}, ErrNotFound
	}
	if err != nil {
		return ShareView{}, err
	}
	if revoked == nil {
		if _, err := tx.Exec(ctx, `UPDATE guest_grants SET revoked_at=$3 WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, now); err != nil {
			return ShareView{}, err
		}
		if pkg != "" {
			if err := invitationEvent(ctx, tx, s, org, project, pkg, "unlinked"); err != nil {
				return ShareView{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ShareView{}, err
	}
	views, err := s.ListGuestGrants(ctx, org, project, ShareFilter{}, now)
	if err != nil {
		return ShareView{}, err
	}
	for _, v := range views {
		if v.ID == id {
			return v, nil
		}
	}
	return ShareView{}, ErrNotFound
}

// GuestPageView is everything the guest page shows. It names the project,
// the sharer and the guest, and only the shared thing: no price, no other
// firm, no draft.
type GuestPageView struct {
	Kind       string          `json:"kind"`
	Title      string          `json:"title"`
	Project    string          `json:"project"`
	From       string          `json:"from"`
	To         string          `json:"to"`
	WorksUntil string          `json:"works_until"`
	Documents  []GuestDocument `json:"documents"`
	Issues     []GuestIssue    `json:"issues"`
	// Quote is a firm's place to send its quote; nil on other links.
	Quote *GuestQuoteBox `json:"quote,omitempty"`
}

type GuestDocument struct {
	ID       string `json:"id"`
	Number   string `json:"number"`
	Title    string `json:"title"`
	Revision string `json:"revision"`
}

type GuestIssue struct {
	ID            string `json:"id"`
	Number        int    `json:"number"`
	ReportingDate string `json:"reporting_date"`
}

type GuestQuoteBox struct {
	// Closed says why no quote is taken ("" when one is).
	Closed   string          `json:"closed,omitempty"`
	Received []GuestReceived `json:"received"`
}

type GuestReceived struct {
	Filename   string `json:"filename"`
	ReceivedOn string `json:"received_on"`
}

// GuestPage reads the page in one read-only transaction.
func (s *Store) GuestPage(ctx context.Context, g GuestGrant) (GuestPageView, error) {
	out := GuestPageView{Kind: g.Kind, To: g.SharedWith, WorksUntil: SydneyDate(g.ExpiresAt), Documents: []GuestDocument{}, Issues: []GuestIssue{}}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `SELECT p.name, COALESCE(u.email,'') FROM projects p LEFT JOIN users u ON u.org_id=p.org_id AND u.id=$3::uuid
WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, g.OrgID, g.ProjectID, g.CreatedBy).Scan(&out.Project, &out.From); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, ErrGuestGone
		}
		return out, err
	}
	ids := g.DocumentIDs
	switch g.Kind {
	case GuestDocuments:
		out.Title = countWords(len(ids), "document")
	case GuestReport, GuestPackage:
		report, title, err := grantReport(ctx, tx, g)
		if err != nil {
			return out, err
		}
		out.Title = title
		history, sections, err := reportHistory(ctx, tx, g.OrgID, report)
		if err != nil {
			return out, err
		}
		for _, h := range history {
			out.Issues = append(out.Issues, GuestIssue{ID: h.ID, Number: h.Number, ReportingDate: h.ReportingDate})
		}
		ids = nil
		if g.Kind == GuestPackage {
			// The latest issue's transmittal is what the firm downloads.
			ids = transmittalIDs(sections)
			box, err := guestQuoteBox(ctx, tx, g)
			if err != nil {
				return out, err
			}
			out.Quote = &box
		}
	}
	if len(ids) > 0 {
		values, err := reportDocumentScheduleSelection(ctx, tx, g.OrgID, g.ProjectID, ids)
		if err != nil {
			return out, err
		}
		for _, v := range values {
			out.Documents = append(out.Documents, guestDocumentRow(v))
		}
	}
	return out, nil
}

// grantReport is the report a report or package link shows, and its title.
func grantReport(ctx context.Context, tx pgx.Tx, g GuestGrant) (string, string, error) {
	report := g.ReportID
	if g.Kind == GuestPackage {
		var err error
		if report, err = packageTenderReport(ctx, tx, g.OrgID, g.ProjectID, g.PackageID); err != nil {
			return "", "", err
		}
	}
	var title string
	err := tx.QueryRow(ctx, `SELECT title FROM reports WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, g.OrgID, g.ProjectID, report).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrGuestGone
	}
	return report, title, err
}

func guestDocumentRow(v reports.BriefValue) GuestDocument {
	var basis struct {
		Sources []map[string]string `json:"sources"`
	}
	row := GuestDocument{ID: v.ID}
	if json.Unmarshal(v.Basis, &basis) == nil && len(basis.Sources) > 0 {
		src := basis.Sources[0]
		row.Number, row.Title, row.Revision = src["document_number"], src["title"], src["revision"]
	}
	return row
}

// transmittalIDs are the document ids an issue's transmittal block froze.
func transmittalIDs(sections []reports.Section) []string {
	var ids []string
	for _, s := range sections {
		for _, b := range s.Blocks {
			var basis struct {
				DocumentIDs []string `json:"document_ids"`
			}
			if json.Unmarshal(b.Basis, &basis) == nil {
				ids = append(ids, basis.DocumentIDs...)
			}
		}
	}
	return unique(ids)
}

// guestQuoteBox is this firm's own quotes on this package, filename and date
// only, and whether one more is taken.
func guestQuoteBox(ctx context.Context, tx pgx.Tx, g GuestGrant) (GuestQuoteBox, error) {
	box := GuestQuoteBox{Received: []GuestReceived{}}
	var awarded bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM awards WHERE org_id=$1::uuid AND package_id=$2::uuid AND undone_at IS NULL)`, g.OrgID, g.PackageID).Scan(&awarded); err != nil {
		return box, err
	}
	if awarded {
		box.Closed = "This tender has been awarded."
	}
	rows, err := tx.Query(ctx, `SELECT filename, received_on::text FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND firm_id=$4::uuid AND document_id IS NOT NULL ORDER BY created_at DESC, id`, g.OrgID, g.ProjectID, g.PackageID, g.FirmID)
	if err != nil {
		return box, err
	}
	defer rows.Close()
	for rows.Next() {
		var r GuestReceived
		if err := rows.Scan(&r.Filename, &r.ReceivedOn); err != nil {
			return box, err
		}
		box.Received = append(box.Received, r)
	}
	return box, rows.Err()
}

// GuestFile is what a guest download streams.
type GuestFile struct {
	Filename  string
	MediaType string
	SHA256    []byte
}

// GuestDocumentFile admits a documents link's own ids, or a package link's
// transmittal ids from any issue of its RFP or RFT, and nothing else. A
// document deleted since drops off: ErrNotFound.
func (s *Store) GuestDocumentFile(ctx context.Context, g GuestGrant, doc string) (GuestFile, error) {
	if !validUUID(doc) {
		return GuestFile{}, ErrNotFound
	}
	doc = strings.ToLower(doc)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return GuestFile{}, err
	}
	defer tx.Rollback(ctx)
	allowed := false
	switch g.Kind {
	case GuestDocuments:
		allowed = containsString(g.DocumentIDs, doc)
	case GuestPackage:
		report, err := packageTenderReport(ctx, tx, g.OrgID, g.ProjectID, g.PackageID)
		if err != nil {
			return GuestFile{}, err
		}
		rows, err := tx.Query(ctx, `SELECT sections FROM report_versions WHERE org_id=$1::uuid AND report_id::text=$2 AND status='issued'`, g.OrgID, report)
		if err != nil {
			return GuestFile{}, err
		}
		for rows.Next() && !allowed {
			var raw []byte
			var sections []reports.Section
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return GuestFile{}, err
			}
			if json.Unmarshal(raw, &sections) == nil {
				allowed = containsString(transmittalIDs(sections), doc)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return GuestFile{}, err
		}
	}
	if !allowed {
		return GuestFile{}, ErrNotFound
	}
	var f GuestFile
	err = tx.QueryRow(ctx, `SELECT d.filename, f.media_type, f.sha256 FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id=$3::uuid`, g.OrgID, g.ProjectID, doc).Scan(&f.Filename, &f.MediaType, &f.SHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuestFile{}, ErrNotFound
	}
	return f, err
}

// GuestIssueFile admits only an issued version of the link's report (or its
// package's RFP or RFT). A draft is ErrNotFound.
func (s *Store) GuestIssueFile(ctx context.Context, g GuestGrant, version string) (IssuedReport, error) {
	if g.Kind == GuestDocuments || !validUUID(version) {
		return IssuedReport{}, ErrNotFound
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return IssuedReport{}, err
	}
	report, _, err := grantReport(ctx, tx, g)
	tx.Rollback(ctx)
	if errors.Is(err, ErrGuestGone) {
		return IssuedReport{}, ErrNotFound
	}
	if err != nil {
		return IssuedReport{}, err
	}
	return s.ReadIssuedReport(ctx, g.OrgID, report, version)
}

// GuestQuote is what the guest is told about a quote it sent.
type GuestQuote struct {
	SubmissionID string `json:"-"`
	Filename     string `json:"filename"`
	ReceivedOn   string `json:"received_on"`
}

// OpenGuestQuote places a firm's quote on its own invited cell, as a PM drop
// there does, in one transaction under the project lock. The grant is
// checked again under that lock (ErrGuestGone), so a link revoked while the
// bytes were landing places nothing. created says the upload made this
// document; one that existed before is placed only when it is already this
// firm's quote here, never re-pointed from another firm. The write is
// recorded as a change with the grant as actor, so no member can undo it.
func (s *Store) OpenGuestQuote(ctx context.Context, g GuestGrant, documentID string, created bool, now time.Time) (GuestQuote, error) {
	if g.Kind != GuestPackage {
		return GuestQuote{}, ErrNotFound
	}
	tx, err := s.pool.Pool.Begin(ctx)
	if err != nil {
		return GuestQuote{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, g.OrgID, g.ProjectID); err != nil {
		return GuestQuote{}, err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT g.id::text FROM guest_grants g WHERE g.org_id=$1::uuid AND g.id=$3::uuid AND `+liveGrantSQL+` FOR SHARE OF g`, g.OrgID, now, g.ID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuestQuote{}, ErrGuestGone
	}
	if err != nil {
		return GuestQuote{}, err
	}
	var awarded bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM awards WHERE org_id=$1::uuid AND package_id=$2::uuid AND undone_at IS NULL)`, g.OrgID, g.PackageID).Scan(&awarded); err != nil {
		return GuestQuote{}, err
	}
	if awarded {
		return GuestQuote{}, ErrGuestAwarded
	}
	var q GuestQuote
	var subPkg, subFirm string
	err = tx.QueryRow(ctx, `SELECT d.filename, COALESCE(s.id::text,''), COALESCE(s.package_id::text,''), COALESCE(s.firm_id::text,'') FROM documents d
LEFT JOIN submissions s ON s.org_id=d.org_id AND s.document_id=d.id
WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id::text=$3`, g.OrgID, g.ProjectID, documentID).Scan(&q.Filename, &q.SubmissionID, &subPkg, &subFirm)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuestQuote{}, ErrNotFound
	}
	if err != nil {
		return GuestQuote{}, err
	}
	mine := q.SubmissionID != "" && subPkg == g.PackageID && subFirm == g.FirmID
	if !created && !mine {
		// The same bytes are already in the project as something else: another
		// firm's quote or the PM's own file. Never re-point it.
		return GuestQuote{}, ErrConflict
	}
	if !mine {
		if q.SubmissionID, err = s.openSubmissionTx(ctx, tx, g.OrgID, g.ProjectID, documentID, g.PackageID, g.FirmID, now); err != nil {
			return GuestQuote{}, err
		}
		ops, _ := json.Marshal([]map[string]string{{"op": "guest_quote", "document_id": documentID, "submission_id": q.SubmissionID, "filename": q.Filename}})
		if _, err := tx.Exec(ctx, `INSERT INTO project_changes (org_id,project_id,id,actor,via,area,ops,at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,'guest','procurement',$5::jsonb,$6)`,
			g.OrgID, g.ProjectID, newID(), g.ID, string(ops), now); err != nil {
			return GuestQuote{}, err
		}
	}
	q.ReceivedOn = SydneyDate(now)
	return q, tx.Commit(ctx)
}

// markGuestLinks marks each card whose firm holds a live link on its row,
// with the date the newest one was last opened. A grouped member's link
// shows on its group's row, as its invitation does.
func markGuestLinks(ctx context.Context, tx pgx.Tx, org, project string, board map[string][]FirmCard) error {
	issued, err := tx.Query(ctx, `SELECT DISTINCT r.package_id::text FROM reports r
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.kind IN ('rfp','rft') AND r.package_id IS NOT NULL
  AND EXISTS(SELECT 1 FROM report_versions v WHERE v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued')`, org, project)
	if err != nil {
		return err
	}
	for issued.Next() {
		var pkg string
		if err := issued.Scan(&pkg); err != nil {
			issued.Close()
			return err
		}
		for i := range board[pkg] {
			board[pkg][i].TenderIssued = true
		}
	}
	issued.Close()
	if err := issued.Err(); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT ON (COALESCE(k.group_id,g.package_id), g.firm_id) COALESCE(k.group_id,g.package_id)::text, g.firm_id::text, g.last_opened_at
FROM guest_grants g JOIN packages k ON k.org_id=g.org_id AND k.project_id=g.project_id AND k.id=g.package_id AND k.retired_at IS NULL
WHERE g.org_id=$1::uuid AND g.project_id=$3::uuid AND g.kind='package' AND g.revoked_at IS NULL AND g.expires_at > $2
ORDER BY COALESCE(k.group_id,g.package_id), g.firm_id, g.created_at DESC`, org, time.Now(), project)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var pkg, firm string
		var opened *time.Time
		if err := rows.Scan(&pkg, &firm, &opened); err != nil {
			return err
		}
		for i := range board[pkg] {
			if c := &board[pkg][i]; c.FirmID == firm {
				c.Linked = true
				if opened != nil {
					c.LinkOpenedOn = SydneyDate(*opened)
				}
			}
		}
	}
	return rows.Err()
}

// ShareContext is what the Share dialog needs to know about one thing before
// a link is made: whether it has an issue, whether the newest issue prints
// the budget, and an invited firm's directory email.
type ShareContext struct {
	Issued          bool
	BudgetDisclosed bool
	SuggestTo       string
}

func (s *Store) ShareContext(ctx context.Context, org, project string, f ShareFilter) (ShareContext, error) {
	var out ShareContext
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	report := f.ReportID
	if f.PackageID != "" {
		if report, err = packageTenderReport(ctx, tx, org, project, f.PackageID); err != nil {
			return out, err
		}
	}
	if report != "" {
		err := tx.QueryRow(ctx, `SELECT v.budget_disclosed FROM report_versions v JOIN reports r ON r.org_id=v.org_id AND r.id=v.report_id
WHERE v.org_id=$1::uuid AND r.project_id=$2::uuid AND v.report_id::text=$3 AND v.status='issued' ORDER BY v.number DESC LIMIT 1`, org, project, report).Scan(&out.BudgetDisclosed)
		switch {
		case err == nil:
			out.Issued = true
		case !errors.Is(err, pgx.ErrNoRows):
			return out, err
		}
	}
	if f.FirmID != "" {
		err := tx.QueryRow(ctx, `SELECT email FROM firms WHERE org_id=$1::uuid AND id::text=$2`, org, f.FirmID).Scan(&out.SuggestTo)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
	}
	return out, nil
}
