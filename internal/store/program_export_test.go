package store

import (
	"context"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// SetProgramOutlineForTest builds detailed programs (trade rows from work
// items) for the rest of a test, so review mechanics can be driven by work
// items. Store tests do not run in parallel.
func SetProgramOutlineForTest(t interface{ Cleanup(func()) }, on bool) {
	was := programOutline
	programOutline = on
	t.Cleanup(func() { programOutline = was })
}

// ProgramGenInputForTest exposes the generation input reader to the
// external store tests.
func ProgramGenInputForTest(ctx context.Context, s *Store, org, project string, cat *knowledge.Catalog) (engine.GenInput, error) {
	in, err := readProgramGenInput(ctx, s.pool, org, project, cat)
	return in.gi, err
}
