package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Team errors. Each maps to one plain message at the HTTP boundary.
var (
	ErrSeatLimit     = errors.New("seat limit reached")
	ErrInvitePending = errors.New("invite already open")
	ErrLastOwner     = errors.New("cannot remove the owner")
)

// TeamMember is one person who can sign in to the org.
type TeamMember struct {
	UserID string    `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Since  time.Time `json:"since"`
}

// TeamInvite is an invite not yet accepted.
type TeamInvite struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Team lists the org's members and open invites.
func (s *Store) Team(ctx context.Context, orgID string, now time.Time) ([]TeamMember, []TeamInvite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.email, m.role, m.created_at
		FROM memberships m JOIN users u ON u.org_id = m.org_id AND u.id = m.user_id
		WHERE m.org_id = $1 ORDER BY m.created_at, u.email`, orgID)
	if err != nil {
		return nil, nil, err
	}
	members, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (TeamMember, error) {
		var m TeamMember
		return m, r.Scan(&m.UserID, &m.Email, &m.Role, &m.Since)
	})
	if err != nil {
		return nil, nil, err
	}
	rows, err = s.pool.Query(ctx, `
		SELECT id::text, email, expires_at FROM invites
		WHERE org_id = $1 AND consumed_at IS NULL AND expires_at > $2 ORDER BY expires_at`, orgID, now)
	if err != nil {
		return nil, nil, err
	}
	invites, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (TeamInvite, error) {
		var i TeamInvite
		return i, r.Scan(&i.ID, &i.Email, &i.ExpiresAt)
	})
	return members, invites, err
}

// CreateTeamInvite opens a member invite when a seat is free. Seats are
// members plus open invites; the org row is locked so two invites cannot
// both take the last seat. An email that already signs in anywhere is
// refused: in v1 one person belongs to one org.
func (s *Store) CreateTeamInvite(ctx context.Context, orgID, email string, tokenHash []byte, now, expires time.Time, seats int) (TeamInvite, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TeamInvite{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM orgs WHERE id = $1 FOR UPDATE`, orgID); err != nil {
		return TeamInvite{}, err
	}
	var used int
	var taken, pending bool
	err = tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM memberships WHERE org_id = $1)
		     + (SELECT count(*) FROM invites WHERE org_id = $1 AND consumed_at IS NULL AND expires_at > $3),
		       EXISTS (SELECT 1 FROM users WHERE lower(email) = $2 AND password_hash IS NOT NULL),
		       EXISTS (SELECT 1 FROM invites WHERE org_id = $1 AND lower(email) = $2 AND consumed_at IS NULL AND expires_at > $3)`,
		orgID, email, now).Scan(&used, &taken, &pending)
	if err != nil {
		return TeamInvite{}, err
	}
	switch {
	case taken:
		return TeamInvite{}, ErrEmailTaken
	case pending:
		return TeamInvite{}, ErrInvitePending
	case used >= seats:
		return TeamInvite{}, ErrSeatLimit
	}
	inv := TeamInvite{ID: newID(), Email: email, ExpiresAt: expires}
	if _, err := tx.Exec(ctx, `
		INSERT INTO invites (org_id, id, email, token_hash, expires_at, role)
		VALUES ($1, $2, $3, $4, $5, 'member')`, orgID, inv.ID, email, tokenHash, expires); err != nil {
		return TeamInvite{}, err
	}
	return inv, tx.Commit(ctx)
}

// RevokeInvite ends an open invite so its link no longer works.
func (s *Store) RevokeInvite(ctx context.Context, orgID, inviteID string, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE invites SET expires_at = $3
		WHERE org_id = $1 AND id::text = $2 AND consumed_at IS NULL AND expires_at > $3`, orgID, inviteID, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// RemoveMember takes a member out of the org: membership and sessions end and
// the password is cleared, so they cannot sign in and their email is free.
// The user row stays because records they made point at it.
func (s *Store) RemoveMember(ctx context.Context, orgID, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var role string
	err = tx.QueryRow(ctx, `DELETE FROM memberships WHERE org_id = $1 AND user_id::text = $2 AND role <> 'owner' RETURNING role`, orgID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		var owner bool
		_ = tx.QueryRow(ctx, `SELECT true FROM memberships WHERE org_id = $1 AND user_id::text = $2 AND role = 'owner'`, orgID, userID).Scan(&owner)
		if owner {
			return ErrLastOwner
		}
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE org_id = $1 AND user_id::text = $2`, orgID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = NULL WHERE org_id = $1 AND id::text = $2`, orgID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AcceptTeamInvite uses an invite once: it gives the invited email a password
// in the inviting org, makes them a member and opens a session. The link
// proves the address, so the user starts verified.
func (s *Store) AcceptTeamInvite(ctx context.Context, tokenHash, passwordHash []byte, terms string, now time.Time) (Session, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var orgID, inviteID, email, role string
	err = tx.QueryRow(ctx, `
		SELECT org_id::text, id::text, lower(email), role FROM invites
		WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $2
		FOR UPDATE`, tokenHash, now).Scan(&orgID, &inviteID, &email, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrLinkUsed
	}
	if err != nil {
		return Session{}, err
	}
	var taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = $1 AND password_hash IS NOT NULL)`, email).Scan(&taken); err != nil {
		return Session{}, err
	}
	if taken {
		return Session{}, ErrEmailTaken
	}
	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE users SET password_hash = $3, email_verified = true, terms_accepted_at = $4, terms_version = $5
		WHERE org_id = $1 AND lower(email) = $2 RETURNING id::text`, orgID, email, passwordHash, now, terms).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		userID = newID()
		_, err = tx.Exec(ctx, `
			INSERT INTO users (org_id, id, email, password_hash, email_verified, terms_accepted_at, terms_version)
			VALUES ($1, $2, $3, $4, true, $5, $6)`, orgID, userID, email, passwordHash, now, terms)
	}
	if err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, $3)
		ON CONFLICT (org_id, user_id) DO NOTHING`, orgID, userID, role); err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE invites SET consumed_at = $3 WHERE org_id = $1 AND id = $2`, orgID, inviteID, now); err != nil {
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
