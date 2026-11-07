package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrEmailTaken means a password account already uses that email.
var ErrEmailTaken = errors.New("email taken")

// ErrResetUsed means a reset link was already used or has expired.
var ErrResetUsed = errors.New("reset used or expired")

// Account is the sign-in identity: one owner user in their own org.
type Account struct {
	OrgID        string
	UserID       string
	Email        string
	PasswordHash []byte
}

// CreateAccount makes a new org with one owner who signs in by password, and
// opens their first session. The email must already be normalised. The owner
// starts unverified and records the terms version they accepted.
func (s *Store) CreateAccount(ctx context.Context, email string, hash []byte, terms string) (Session, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	orgID, userID, sessionID := newID(), newID(), newID()
	if _, err := tx.Exec(ctx, `INSERT INTO orgs (id, name) VALUES ($1, $2)`, orgID, email); err != nil {
		return Session{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO users (org_id, id, email, password_hash, email_verified, terms_accepted_at, terms_version)
		VALUES ($1, $2, $3, $4, false, now(), $5)`, orgID, userID, email, hash, terms)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Session{}, ErrEmailTaken
	}
	if err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, 'owner')`, orgID, userID); err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO sessions (org_id, id, user_id) VALUES ($1, $2, $3)`, orgID, sessionID, userID); err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{ID: sessionID, OrgID: orgID, UserID: userID}, nil
}

// AccountByEmail finds the password account for a normalised email.
func (s *Store) AccountByEmail(ctx context.Context, email string) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `
		SELECT org_id::text, id::text, email, password_hash FROM users
		WHERE lower(email) = $1 AND password_hash IS NOT NULL`, email).
		Scan(&a.OrgID, &a.UserID, &a.Email, &a.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

// StartSession opens a new session for an account that just proved its password.
func (s *Store) StartSession(ctx context.Context, orgID, userID string) (Session, error) {
	id := newID()
	if _, err := s.pool.Exec(ctx, `INSERT INTO sessions (org_id, id, user_id) VALUES ($1, $2, $3)`, orgID, id, userID); err != nil {
		return Session{}, err
	}
	return Session{ID: id, OrgID: orgID, UserID: userID}, nil
}

// CreatePasswordReset stores a hashed single-use reset token.
func (s *Store) CreatePasswordReset(ctx context.Context, a Account, tokenHash []byte, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO password_resets (org_id, id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, a.OrgID, newID(), a.UserID, tokenHash, expires)
	return err
}

// ResetPassword uses a reset token once: it sets the new hash, ends every
// other session of that user and opens a fresh one, in one transaction.
func (s *Store) ResetPassword(ctx context.Context, tokenHash, newHash []byte, now time.Time) (Session, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var orgID, userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_resets SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING org_id::text, user_id::text`, tokenHash, now).Scan(&orgID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrResetUsed
	}
	if err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $3 WHERE org_id = $1 AND id = $2`, orgID, userID, newHash); err != nil {
		return Session{}, err
	}
	// Any other outstanding link for this user is spent too.
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = $3 WHERE org_id = $1 AND user_id = $2 AND used_at IS NULL`, orgID, userID, now); err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE org_id = $1 AND user_id = $2`, orgID, userID); err != nil {
		return Session{}, err
	}
	sessionID := newID()
	if _, err := tx.Exec(ctx, `INSERT INTO sessions (org_id, id, user_id) VALUES ($1, $2, $3)`, orgID, sessionID, userID); err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{ID: sessionID, OrgID: orgID, UserID: userID}, nil
}

// SetPassword gives an existing user a password so they can sign in by
// email. It is the operator path for invite-only users made before accounts.
// The email must be unique among users when orgID is empty.
func (s *Store) SetPassword(ctx context.Context, orgID, email string, hash []byte) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET password_hash = $3
		WHERE lower(email) = $2 AND ($1 = '' OR org_id::text = $1)
		  AND (SELECT count(*) FROM users u WHERE lower(u.email) = $2 AND ($1 = '' OR u.org_id::text = $1)) = 1`,
		orgID, email, hash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailTaken
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// ErrLinkUsed means an emailed link was already used or has expired.
var ErrLinkUsed = errors.New("link used or expired")

// Profile is the signed-in person as the account page shows them.
type Profile struct {
	Email    string `json:"email"`
	Verified bool   `json:"email_verified"`
	Role     string `json:"role"`
}

// Profile reads the signed-in user's email, verification and role.
func (s *Store) Profile(ctx context.Context, orgID, userID string) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT u.email, u.email_verified, coalesce(m.role, '')
		FROM users u LEFT JOIN memberships m ON m.org_id = u.org_id AND m.user_id = u.id
		WHERE u.org_id = $1 AND u.id = $2`, orgID, userID).Scan(&p.Email, &p.Verified, &p.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

// CreateEmailVerification stores a hashed single-use verification token.
func (s *Store) CreateEmailVerification(ctx context.Context, orgID, userID string, tokenHash []byte, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO email_verifications (org_id, id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, orgID, newID(), userID, tokenHash, expires)
	return err
}

// VerifyEmail uses a verification token once and marks its user verified.
func (s *Store) VerifyEmail(ctx context.Context, tokenHash []byte, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var orgID, userID string
	err = tx.QueryRow(ctx, `
		UPDATE email_verifications SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING org_id::text, user_id::text`, tokenHash, now).Scan(&orgID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLinkUsed
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET email_verified = true WHERE org_id = $1 AND id = $2`, orgID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
