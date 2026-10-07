package store

import (
	"context"
	"sitewise/internal/profile"
)

// DocumentParts reads only the site's parts, rather than the full profile
// snapshot, once per background stage. Document access is scoped to the org.
func (s *Store) DocumentParts(ctx context.Context, org, document string) ([]profile.Part, error) {
	doc, err := s.GetDocument(ctx, org, document)
	if err != nil {
		return nil, err
	}
	return readParts(ctx, s.pool, org, doc.ProjectID)
}
