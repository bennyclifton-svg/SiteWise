package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	engine "sitewise/internal/program"
)

// Update program, one click (owner, 12 November 2026): pressing the button
// is the user committing to every update, so nothing is asked. The program
// is regenerated from the Profile and Procurement and the whole difference
// is applied in one version-pinned batch; one undo reverses it. No Jev.
//
// The diff is applied in rounds inside the one write: a library link
// between two rows added in the same round can be refused (a chain link the
// add placed is already there, or a row still being removed makes it look
// like a loop) and only lands once those rows exist. Three rounds settle
// every case seen; a link still refused after them is a real loop and is
// left out, as an Adopt would leave it.
const updateRounds = 3

// UpdateProgram applies the regenerated program's difference.
func (s *Store) UpdateProgram(ctx context.Context, org, project, actor string, version int64, cat *knowledge.Catalog) (ProgramResult, error) {
	if cat == nil {
		return ProgramResult{}, errors.New("update program needs the knowledge catalogue")
	}
	var applied []engine.Op
	res, err := s.programWrite(ctx, org, project, "", version, func(ctx context.Context, tx pgx.Tx, w *programWork) error {
		in, err := readProgramGenInput(ctx, tx, org, project, cat)
		if err != nil {
			return err
		}
		w.genReason = "Removed by Update program."
		for round := 0; round < updateRounds; round++ {
			st := w.state()
			plan, err := engine.Generate(cat, genInputFor(in.gi, w.prog, st.Activities))
			if err != nil {
				return err
			}
			diff, err := reviewDiff(ctx, tx, org, w.prog.ID, st.Activities, st.Links, plan)
			if err != nil {
				return err
			}
			keyIdx := map[string]string{}
			for _, id := range w.order() {
				if k := w.acts[id].SourceKey; k != "" {
					keyIdx[string(k)] = id
				}
			}
			did := retirePlacementChains(w) > 0
			for _, c := range diff {
				for _, op := range c.Ops {
					if op = adoptRefs(op, w, keyIdx); op == nil {
						continue
					}
					ok, err := w.applyGenerated(len(applied), op)
					if err != nil {
						return err
					}
					if ok {
						did = true
						applied = append(applied, op)
					}
				}
			}
			if !did {
				break
			}
		}
		// The last round's adds placed chain links of their own; the plan's
		// keyed links already sequence those rows.
		retired := retirePlacementChains(w)
		if len(applied) == 0 && retired == 0 {
			return errNoProgramChange
		}
		if err := closePendingChanges(ctx, tx, org, w.prog.ID); err != nil {
			return err
		}
		w.fb.Toasts = append(w.fb.Toasts, fmt.Sprintf("Program updated: %d change%s. Undo reverses it.", len(applied), plural(len(applied))))
		return nil
	}, func(ctx context.Context, tx pgx.Tx, w *programWork, c programChanges) error {
		// Logged as a review adoption so the review reads these rows as
		// generation's values, not the user's own (generatedBatch).
		ops := []any{map[string]any{"op": "review_adopt", "changes": []string{}, "applied": applied}}
		return logProgramBatch(ctx, tx, w, actor, ops, c)
	})
	if !errors.Is(err, errNoProgramChange) {
		return res, err
	}
	return s.programUpToDate(ctx, org, project, version)
}

// retirePlacementChains retires chain links with no library key between two
// library rows. Adding a generated row chains it behind its neighbour the
// way a user add does; the regenerated plan already has its own keyed link
// for every sequence, so such a link only adds an order the library does
// not have and can block a library link as a loop (Construction certificate
// before the documentation it certifies). Links the user drew, and chains
// to the user's own rows, are left alone.
func retirePlacementChains(w *programWork) int {
	n := 0
	for _, l := range w.liveLinks() {
		if l.Origin != engine.LinkOriginChain || l.SourceKey != "" {
			continue
		}
		pred, succ := w.live(l.PredID), w.live(l.SuccID)
		if pred == nil || succ == nil || pred.SourceKey == "" || succ.SourceKey == "" {
			continue
		}
		w.retireLink(l.ID)
		n++
	}
	return n
}

// closePendingChanges retires the stored review items: the update applied
// whatever they described, or it no longer applies.
func closePendingChanges(ctx context.Context, tx pgx.Tx, org, programID string) error {
	_, err := tx.Exec(ctx, `UPDATE program_changes SET status='superseded' WHERE org_id=$1::uuid AND program_id=$2::uuid AND status='pending'`, org, programID)
	return err
}

// programUpToDate answers an update with nothing to apply: the pending
// items are closed and the program is returned unchanged.
func (s *Store) programUpToDate(ctx context.Context, org, project string, version int64) (ProgramResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProgramResult{}, err
	}
	defer tx.Rollback(ctx)
	p, acts, links, days, err := readProgramInputs(ctx, tx, org, project, true)
	if err != nil {
		return ProgramResult{}, err
	}
	st := computeState(p, acts, links, days)
	if p.Version != version {
		return ProgramResult{ProgramState: st, ProgramFeedback: emptyFeedback()}, ErrVersionConflict
	}
	if err := closePendingChanges(ctx, tx, org, p.ID); err != nil {
		return ProgramResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramResult{}, err
	}
	fb := emptyFeedback()
	fb.Toasts = append(fb.Toasts, "The program is up to date with the Profile and Procurement.")
	return ProgramResult{ProgramState: st, ProgramFeedback: fb}, nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
