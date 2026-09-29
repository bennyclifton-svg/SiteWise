package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sitewise/internal/db"
)

// ErrNotFound means the row is not visible in the caller's org.
var ErrNotFound = errors.New("not found")

// ErrInviteExpired means the invite can no longer be consumed.
var ErrInviteExpired = errors.New("invite expired")

// ErrInviteUsed means the invite was already consumed.
var ErrInviteUsed = errors.New("invite used")

// File is a content-addressed blob associated with one project.
type File struct {
	ID        string
	ProjectID string
	SHA256    []byte
	ByteSize  int64
	MediaType string
}

// Document is one filing of a file. SupersedesID is empty when the document
// does not replace an earlier filing.
type Document struct {
	ID           string
	ProjectID    string
	FileID       string
	Filename     string
	Status       string
	Number       string
	Revision     string
	SupersedesID string
}

// Decision is one field judgment on a document.
type Decision struct {
	ID         string
	DocumentID string
	Field      string
	Value      string
	Band       string
	DecidedBy  string
}

// Passage is an extracted span of a document.
type Passage struct {
	ID         string
	DocumentID string
	Ordinal    int32
	Body       string
}

// Invite is a single-use membership invitation. Role is owner or member.
// A zero ExpiresAt is stored as 24 hours from creation.
type Invite struct {
	ID        string
	Email     string
	TokenHash []byte
	Role      string
	ExpiresAt time.Time
}

// Project is a filing container owned by one org.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Job is a durable unit of background work.
type Job struct {
	ID         string
	DocumentID string
	Kind       string
	Status     string
}

// Store is the tenant-scoped repository. orgID arguments come from the
// authenticated session, and every query predicates on that org.
type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// Open connects to PostgreSQL. It does not migrate.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, q: db.New(pool)}, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Migrate applies schema migrations.
func (s *Store) Migrate(ctx context.Context) error {
	return db.Migrate(ctx, s.pool)
}

// CreateOrg inserts an organisation.
func (s *Store) CreateOrg(ctx context.Context, orgID, name string) error {
	return s.q.CreateOrg(ctx, db.CreateOrgParams{ID: orgID, Name: name})
}

// DeleteOrg removes one organisation and its tenant rows.
func (s *Store) DeleteOrg(ctx context.Context, orgID string) error {
	return s.q.DeleteOrg(ctx, orgID)
}

// CreateUser inserts a user inside an org.
func (s *Store) CreateUser(ctx context.Context, orgID, userID, email string) error {
	return s.q.CreateUser(ctx, db.CreateUserParams{OrgID: orgID, ID: userID, Email: email})
}

// CreateMembership inserts one membership. The user must already belong to orgID.
func (s *Store) CreateMembership(ctx context.Context, orgID, userID, role string) error {
	return s.q.CreateMembership(ctx, db.CreateMembershipParams{OrgID: orgID, UserID: userID, Role: role})
}

// CreateInvite stores an invitation token hash.
func (s *Store) CreateInvite(ctx context.Context, orgID string, invite Invite) error {
	role := invite.Role
	if role == "" {
		role = "member"
	}
	expires := invite.ExpiresAt
	if expires.IsZero() {
		expires = time.Now().Add(24 * time.Hour)
	}
	return s.q.CreateInvite(ctx, db.CreateInviteParams{
		OrgID:     orgID,
		ID:        invite.ID,
		Email:     invite.Email,
		TokenHash: invite.TokenHash,
		Role:      role,
		ExpiresAt: expires,
	})
}

// GetInvite loads an invite visible to orgID.
func (s *Store) GetInvite(ctx context.Context, orgID, inviteID string) (Invite, error) {
	row, err := s.q.GetInvite(ctx, db.GetInviteParams{OrgID: orgID, ID: inviteID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Invite{}, ErrNotFound
	}
	if err != nil {
		return Invite{}, err
	}
	return Invite{ID: row.ID, Email: row.Email}, nil
}

// CreateSession inserts a session for a user in orgID.
func (s *Store) CreateSession(ctx context.Context, orgID, sessionID, userID string) error {
	return s.q.CreateSession(ctx, db.CreateSessionParams{OrgID: orgID, ID: sessionID, UserID: userID})
}

// GetSession loads a session only when it belongs to orgID.
func (s *Store) GetSession(ctx context.Context, orgID, sessionID string) (string, error) {
	row, err := s.q.GetSession(ctx, db.GetSessionParams{OrgID: orgID, ID: sessionID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

// CreateProject inserts a project owned by orgID.
func (s *Store) CreateProject(ctx context.Context, orgID, projectID, name string) error {
	return s.q.CreateProject(ctx, db.CreateProjectParams{OrgID: orgID, ID: projectID, Name: name})
}

// CreateFile associates a blob with a project in orgID.
func (s *Store) CreateFile(ctx context.Context, orgID string, file File) error {
	return s.q.CreateFile(ctx, db.CreateFileParams{
		OrgID:     orgID,
		ID:        file.ID,
		ProjectID: file.ProjectID,
		Sha256:    file.SHA256,
		ByteSize:  file.ByteSize,
		MediaType: file.MediaType,
	})
}

// GetFile loads file metadata visible to orgID.
func (s *Store) GetFile(ctx context.Context, orgID, fileID string) (File, error) {
	row, err := s.q.GetFile(ctx, db.GetFileParams{OrgID: orgID, ID: fileID})
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, err
	}
	return fileFromRow(row.ID, row.ProjectID, row.Sha256, row.ByteSize, row.MediaType), nil
}

// FindFileByHash loads the project association for a hash. Another org's copy
// is not visible.
func (s *Store) FindFileByHash(ctx context.Context, orgID, projectID string, sha256 []byte) (File, error) {
	row, err := s.q.FindFileByHash(ctx, db.FindFileByHashParams{
		OrgID:     orgID,
		ProjectID: projectID,
		Sha256:    sha256,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, err
	}
	return fileFromRow(row.ID, row.ProjectID, row.Sha256, row.ByteSize, row.MediaType), nil
}

// CreateDocument files a document in orgID.
func (s *Store) CreateDocument(ctx context.Context, orgID string, doc Document) error {
	return s.q.CreateDocument(ctx, db.CreateDocumentParams{
		OrgID:          orgID,
		ID:             doc.ID,
		ProjectID:      doc.ProjectID,
		FileID:         doc.FileID,
		Filename:       doc.Filename,
		Status:         doc.Status,
		DocumentNumber: strPtr(doc.Number),
		Revision:       strPtr(doc.Revision),
	})
}

// GetDocument loads a document visible to orgID.
func (s *Store) GetDocument(ctx context.Context, orgID, documentID string) (Document, error) {
	row, err := s.q.GetDocument(ctx, db.GetDocumentParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, err
	}
	prior, err := s.q.GetSupersession(ctx, db.GetSupersessionParams{OrgID: orgID, DocumentID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		prior = ""
	} else if err != nil {
		return Document{}, err
	}
	return Document{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		FileID:       row.FileID,
		Filename:     row.Filename,
		Status:       row.Status,
		Number:       row.DocumentNumber,
		Revision:     row.Revision,
		SupersedesID: prior,
	}, nil
}

// UpdateDocumentStatus changes status inside orgID.
func (s *Store) UpdateDocumentStatus(ctx context.Context, orgID, documentID, status string) error {
	n, err := s.q.UpdateDocumentStatus(ctx, db.UpdateDocumentStatusParams{
		OrgID:  orgID,
		ID:     documentID,
		Status: status,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Supersede links a document to an earlier document in the same org.
func (s *Store) Supersede(ctx context.Context, orgID, documentID, priorID string) error {
	n, err := s.q.SupersedeDocument(ctx, db.SupersedeDocumentParams{
		OrgID:           orgID,
		DocumentID:      documentID,
		PriorDocumentID: priorID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateDecision records one field decision.
func (s *Store) CreateDecision(ctx context.Context, orgID string, decision Decision) error {
	return s.q.CreateDecision(ctx, db.CreateDecisionParams{
		OrgID:      orgID,
		ID:         decision.ID,
		DocumentID: decision.DocumentID,
		Field:      decision.Field,
		Value:      strPtr(decision.Value),
		Band:       decision.Band,
		DecidedBy:  decision.DecidedBy,
	})
}

// ListStream returns decision events for a project in orgID.
func (s *Store) ListStream(ctx context.Context, orgID, projectID string) ([]Decision, error) {
	rows, err := s.q.ListStream(ctx, db.ListStreamParams{OrgID: orgID, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]Decision, 0, len(rows))
	for _, row := range rows {
		out = append(out, Decision{
			ID:         row.ID,
			DocumentID: row.DocumentID,
			Field:      row.Field,
			Value:      row.Value,
			Band:       row.Band,
			DecidedBy:  row.DecidedBy,
		})
	}
	return out, nil
}

// AddPassage stores a passage of a document in orgID.
func (s *Store) AddPassage(ctx context.Context, orgID string, passage Passage) error {
	return s.q.AddPassage(ctx, db.AddPassageParams{
		OrgID:      orgID,
		ID:         passage.ID,
		DocumentID: passage.DocumentID,
		Ordinal:    passage.Ordinal,
		Body:       passage.Body,
	})
}

// GetPassage loads a passage visible to orgID.
func (s *Store) GetPassage(ctx context.Context, orgID, passageID string) (Passage, error) {
	row, err := s.q.GetPassage(ctx, db.GetPassageParams{OrgID: orgID, ID: passageID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Passage{}, ErrNotFound
	}
	if err != nil {
		return Passage{}, err
	}
	return Passage{ID: row.ID, DocumentID: row.DocumentID, Ordinal: row.Ordinal, Body: row.Body}, nil
}

// EnqueueJob queues work for a document in orgID.
func (s *Store) EnqueueJob(ctx context.Context, orgID, jobID, documentID, kind string) error {
	n, err := s.q.EnqueueJob(ctx, db.EnqueueJobParams{
		OrgID:      orgID,
		ID:         jobID,
		DocumentID: documentID,
		Kind:       kind,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListJobs lists jobs visible to orgID.
func (s *Store) ListJobs(ctx context.Context, orgID string) ([]Job, error) {
	rows, err := s.q.ListJobs(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Job, 0, len(rows))
	for _, row := range rows {
		out = append(out, Job{ID: row.ID, DocumentID: row.DocumentID, Kind: row.Kind, Status: row.Status})
	}
	return out, nil
}

func fileFromRow(id, projectID string, sha []byte, size int64, mediaType string) File {
	return File{ID: id, ProjectID: projectID, SHA256: sha, ByteSize: size, MediaType: mediaType}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Session is an authenticated browser session.
type Session struct {
	ID     string
	OrgID  string
	UserID string
}

// ConsumeInvite marks an invite used and creates the user, membership and
// session in one transaction. The org is taken from the invite row.
func (s *Store) ConsumeInvite(ctx context.Context, tokenHash []byte, now time.Time) (Session, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	inv, err := q.LockInviteByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if inv.Consumed {
		return Session{}, ErrInviteUsed
	}
	if !inv.ExpiresAt.After(now) {
		return Session{}, ErrInviteExpired
	}
	userID, err := q.FindUserByEmail(ctx, db.FindUserByEmailParams{OrgID: inv.OrgID, Email: inv.Email})
	if errors.Is(err, pgx.ErrNoRows) {
		userID = newID()
		err = q.CreateUser(ctx, db.CreateUserParams{OrgID: inv.OrgID, ID: userID, Email: inv.Email})
	}
	if err != nil {
		return Session{}, err
	}
	if err := q.CreateMembership(ctx, db.CreateMembershipParams{OrgID: inv.OrgID, UserID: userID, Role: inv.Role}); err != nil {
		return Session{}, err
	}
	sessionID := newID()
	if err := q.CreateSession(ctx, db.CreateSessionParams{OrgID: inv.OrgID, ID: sessionID, UserID: userID}); err != nil {
		return Session{}, err
	}
	n, err := q.MarkInviteConsumed(ctx, db.MarkInviteConsumedParams{OrgID: inv.OrgID, ID: inv.ID})
	if err != nil {
		return Session{}, err
	}
	if n == 0 {
		return Session{}, ErrInviteUsed
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{ID: sessionID, OrgID: inv.OrgID, UserID: userID}, nil
}

// LookupSession resolves a cookie session id to its org. The caller does not
// supply the org.
func (s *Store) LookupSession(ctx context.Context, sessionID string) (Session, time.Time, error) {
	row, err := s.q.LookupSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, time.Time{}, ErrNotFound
	}
	if err != nil {
		return Session{}, time.Time{}, err
	}
	return Session{ID: sessionID, OrgID: row.OrgID, UserID: row.UserID}, row.ExpiresAt, nil
}

// Member reports whether the user has a membership in orgID.
func (s *Store) Member(ctx context.Context, orgID, userID string) (bool, error) {
	ok, err := s.q.MembershipExists(ctx, db.MembershipExistsParams{OrgID: orgID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ok, nil
}

// GetProject loads a project visible to orgID.
func (s *Store) GetProject(ctx context.Context, orgID, projectID string) (Project, error) {
	row, err := s.q.GetProject(ctx, db.GetProjectParams{OrgID: orgID, ID: projectID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	if err != nil {
		return Project{}, err
	}
	return Project{ID: row.ID, Name: row.Name}, nil
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
