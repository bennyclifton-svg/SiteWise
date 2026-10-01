package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/db"
)

// Backlog is the unfinished (queued or leased) work of one job kind.
type Backlog struct {
	Kind   string
	Queued int64
	// Oldest is the age of the oldest unfinished job by the database clock.
	Oldest time.Duration
}

// Ping checks that the database answers.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Role returns the user's membership role in orgID, or ErrNotFound.
func (s *Store) Role(ctx context.Context, orgID, userID string) (string, error) {
	role, err := s.q.MembershipRole(ctx, db.MembershipRoleParams{OrgID: orgID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

// Backlog is the process-wide unfinished work by kind. It is not a tenant
// read: it returns counts and ages only, for the public health status.
func (s *Store) Backlog(ctx context.Context) ([]Backlog, error) {
	rows, err := s.q.Backlog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Backlog, len(rows))
	for i, r := range rows {
		out[i] = Backlog{Kind: r.Kind, Queued: r.Queued, Oldest: time.Duration(r.OldestMs) * time.Millisecond}
	}
	return out, nil
}

// OrgBacklog is orgID's unfinished work by kind.
func (s *Store) OrgBacklog(ctx context.Context, orgID string) ([]Backlog, error) {
	rows, err := s.q.OrgBacklog(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Backlog, len(rows))
	for i, r := range rows {
		out[i] = Backlog{Kind: r.Kind, Queued: r.Queued, Oldest: time.Duration(r.OldestMs) * time.Millisecond}
	}
	return out, nil
}
