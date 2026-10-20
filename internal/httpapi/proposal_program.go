package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/programread"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

// Reading B: the R7 Program returnable in a proposal (plan
// docs/plans/2026-10-18-program.md section 10.3). When the proposal read
// commits, the proposal_program read starts on the same submission as its
// own background job: code settles exact R7 table rows, and at most one
// Jev fan-out at background priority picks a stated duration for each
// stage still open. Its readings are stored hidden; they show in Program
// as proposals only once that firm's proposal is awarded. Nothing here
// changes the program.

// proposalProgramReader reads proposals' R7 programmes: one goroutine per
// submission, bounded by its own slots, like programImporter.
type proposalProgramReader struct {
	store      *store.Store
	blobs      *files.Store
	ask        programread.Asker
	thresholds programread.Thresholds
	broker     *events.Broker
	log        *log.Logger
	timeout    time.Duration
	slots      chan struct{}
	now        func() time.Time
	wg         sync.WaitGroup

	mu       sync.Mutex
	inflight map[string]struct{}
}

// Queue is the proposal read's hook: after a committed read, the
// submission's program read starts. It returns at once.
func (r *proposalProgramReader) Queue(orgID, submissionID string) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ok, err := r.store.QueueProposalProgram(ctx, orgID, submissionID)
		if err != nil {
			r.log.Printf("proposal program queue org=%s submission=%s: %v", orgID, submissionID, err)
			return
		}
		if ok {
			r.Start(orgID, submissionID, false)
		}
	}()
}

// Start reads one submission's programme unless it is already being read.
func (r *proposalProgramReader) Start(orgID, id string, recheck bool) {
	key := orgID + "/" + id
	r.mu.Lock()
	if _, busy := r.inflight[key]; busy {
		r.mu.Unlock()
		return
	}
	r.inflight[key] = struct{}{}
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.mu.Lock()
			delete(r.inflight, key)
			r.mu.Unlock()
		}()
		r.slots <- struct{}{}
		defer func() { <-r.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()
		done, committed, err := r.read(ctx, orgID, id, recheck)
		if err != nil {
			r.log.Printf("proposal program read stopped org=%s submission=%s: %v", orgID, id, err)
			return
		}
		if !committed || done.Readings == 0 || r.broker == nil {
			return
		}
		// Open Program views refetch; readings of a proposal not yet
		// awarded stay hidden there.
		payload, _ := json.Marshal(map[string]any{"project_id": done.ProjectID, "program_id": done.ProgramID, "document_id": done.DocumentID})
		if _, err := r.broker.Commit(ctx, orgID, eventProgramReadings, "", string(payload)); err != nil {
			r.log.Printf("program readings event: %v", err)
		}
	}()
}

func (r *proposalProgramReader) read(ctx context.Context, org, id string, recheck bool) (store.ProposalProgramCommitted, bool, error) {
	in, err := r.store.ProposalProgramForReading(ctx, org, id)
	if err != nil {
		return store.ProposalProgramCommitted{}, false, err
	}
	if !recheck && in.Status != store.ProposalProgramReading {
		return store.ProposalProgramCommitted{}, false, nil
	}
	var out programread.StageOutcome
	if in.NotRead == "" {
		out, in.NotRead, err = r.outcome(ctx, in)
		if err != nil {
			return store.ProposalProgramCommitted{}, false, err
		}
	}
	return r.store.CommitProposalProgram(ctx, org, id, in, out, recheck, r.now())
}

func (r *proposalProgramReader) outcome(ctx context.Context, in store.ProposalProgramInput) (programread.StageOutcome, string, error) {
	format := proposalFormat(in.Filename)
	if format == "" {
		return programread.StageOutcome{}, importUnsupported, nil
	}
	path, err := r.blobs.Path(in.SHA256)
	if err != nil {
		return programread.StageOutcome{}, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return programread.StageOutcome{}, "", err
	}
	text, err := identity.Extract(ctx, format, f, in.Size, proposals.ExtractLimits())
	f.Close()
	switch {
	case errors.Is(err, identity.ErrTooLarge), errors.Is(err, identity.ErrMalformed):
		return programread.StageOutcome{}, importUnreadable, nil
	case err != nil:
		return programread.StageOutcome{}, "", err
	case format == "pdf" && !text.TextLayer:
		return programread.StageOutcome{}, importNoText, nil
	}
	// Rules first, then one background fan-out at most, inside
	// programread.ReadStageWeeks (https://docs.typesafe.ai/patterns/fan-out).
	out, err := programread.ReadStageWeeks(ctx, r.ask, r.thresholds, programread.StageWeeksInput{Filename: in.Filename, Text: text, PerWeek: in.PerWeek, Stages: in.Stages})
	return out, "", err
}

// RunProposalProgramRechecks re-asks grey proposal programme reads in the
// background with the proposal read's spacing: 30 seconds, at most three
// times.
func (s *Server) RunProposalProgramRechecks(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		due, err := s.store.DueProposalProgramRechecks(ctx, now)
		if err != nil {
			continue
		}
		for _, ref := range due {
			if ok, err := s.store.ClaimProposalProgramRecheck(ctx, ref, now); err == nil && ok {
				s.proposalPrograms.Start(ref.OrgID, ref.SubmissionID, true)
			}
		}
	}
}
