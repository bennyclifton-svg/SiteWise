package store

import "context"

func (s *Store) RenameProject(ctx context.Context, orgID, projectID, name string) error {
	result, err := s.pool.Exec(ctx, `UPDATE projects SET name = $3 WHERE org_id = $1 AND id = $2`, orgID, projectID, name)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
