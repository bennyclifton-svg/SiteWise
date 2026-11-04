package store

import "context"

// EndSession revokes only the authenticated session in its organisation.
func (s *Store) EndSession(ctx context.Context, session Session) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1 AND org_id=$2 AND user_id=$3`, session.ID, session.OrgID, session.UserID)
	return err
}
