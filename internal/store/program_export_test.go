package store

import (
	"context"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// ProgramGenInputForTest exposes the generation input reader to the
// external store tests.
func ProgramGenInputForTest(ctx context.Context, s *Store, org, project string, cat *knowledge.Catalog) (engine.GenInput, error) {
	in, err := readProgramGenInput(ctx, s.pool, org, project, cat)
	return in.gi, err
}
