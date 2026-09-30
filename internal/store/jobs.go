package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/db"
)

// ClaimedJob is a leased background job. Token is the only key that can
// complete or fail this attempt.
type ClaimedJob struct {
	OrgID      string
	ID         string
	DocumentID string
	Kind       string
	Token      string
	Attempts   int
}

// JobRecord is the durable state of one job.
type JobRecord struct {
	ID          string
	DocumentID  string
	Kind        string
	Status      string
	Attempts    int
	MaxAttempts int
	Priority    int
	LastError   string
}

// ClaimJob leases the next matching job in orgID. Interactive priorities are
// ordered ahead of background work. An expired lease can be claimed again;
// the previous token then loses the race.
func (s *Store) ClaimJob(ctx context.Context, orgID string, lease time.Duration, kinds []string) (ClaimedJob, error) {
	if orgID == "" {
		return ClaimedJob{}, errors.New("org is required")
	}
	if len(kinds) == 0 {
		return ClaimedJob{}, errors.New("job kind is required")
	}
	for _, kind := range kinds {
		if kind == "" || kind == JobKindIntake {
			return ClaimedJob{}, errors.New("job kind is required")
		}
	}
	if lease < time.Second {
		lease = time.Second
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ClaimedJob{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	if err := q.FailExhaustedJobs(ctx, orgID); err != nil {
		return ClaimedJob{}, err
	}
	row, err := q.ClaimJob(ctx, db.ClaimJobParams{
		LeaseToken:   newID(),
		LeaseSeconds: lease.Seconds(),
		OrgID:        orgID,
		Kinds:        kinds,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return ClaimedJob{}, err
		}
		return ClaimedJob{}, ErrIdle
	}
	if err != nil {
		return ClaimedJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ClaimedJob{}, err
	}
	return ClaimedJob{
		OrgID:      row.OrgID,
		ID:         row.ID,
		DocumentID: row.DocumentID,
		Kind:       row.Kind,
		Token:      row.LeaseToken,
		Attempts:   int(row.Attempts),
	}, nil
}

// CompleteJob finishes a lease and writes its event in the same transaction.
func (s *Store) CompleteJob(ctx context.Context, orgID, jobID, token string, event EventWrite) error {
	return s.finishJob(ctx, orgID, jobID, token, event, true, 0, "")
}

// FailJob returns the job to the queue, or marks it failed when the attempt
// budget is spent. The event is written in the same transaction.
func (s *Store) FailJob(ctx context.Context, orgID, jobID, token, reason string, backoff time.Duration, event EventWrite) error {
	if backoff < 0 {
		backoff = 0
	}
	return s.finishJob(ctx, orgID, jobID, token, event, false, backoff.Seconds(), trimError(reason))
}

func (s *Store) finishJob(ctx context.Context, orgID, jobID, token string, event EventWrite, done bool, backoff float64, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	var n int64
	if done {
		n, err = q.CompleteJob(ctx, db.CompleteJobParams{OrgID: orgID, ID: jobID, LeaseToken: token})
	} else {
		n, err = q.FailJob(ctx, db.FailJobParams{
			BackoffSeconds: backoff,
			LastError:      reason,
			OrgID:          orgID,
			ID:             jobID,
			LeaseToken:     token,
		})
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLeaseLost
	}
	if event.Kind != "" {
		if _, err := appendEvent(ctx, q, orgID, event.Kind, event.DocumentID, event.Payload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ExpireLease makes a leased job claimable without waiting out the clock.
func (s *Store) ExpireLease(ctx context.Context, orgID, jobID string) error {
	n, err := s.q.ExpireJobLease(ctx, db.ExpireJobLeaseParams{OrgID: orgID, ID: jobID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetJob loads one job in orgID.
func (s *Store) GetJob(ctx context.Context, orgID, jobID string) (JobRecord, error) {
	row, err := s.q.GetJob(ctx, db.GetJobParams{OrgID: orgID, ID: jobID})
	if errors.Is(err, pgx.ErrNoRows) {
		return JobRecord{}, ErrNotFound
	}
	if err != nil {
		return JobRecord{}, err
	}
	return JobRecord{
		ID:          row.ID,
		DocumentID:  row.DocumentID,
		Kind:        row.Kind,
		Status:      row.Status,
		Attempts:    int(row.Attempts),
		MaxAttempts: int(row.MaxAttempts),
		Priority:    int(row.Priority),
		LastError:   row.LastError,
	}, nil
}

// EnqueueStage queues a background stage. A stage already queued or finished
// for this document is left as it is.
func (s *Store) EnqueueStage(ctx context.Context, orgID, documentID, kind string) error {
	if kind == "" || kind == JobKindIntake {
		return errors.New("job kind is required")
	}
	n, err := s.q.EnqueueStage(ctx, db.EnqueueStageParams{
		OrgID:      orgID,
		ID:         newID(),
		DocumentID: documentID,
		Kind:       kind,
	})
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = s.q.JobByDocumentKind(ctx, db.JobByDocumentKindParams{
		OrgID:      orgID,
		DocumentID: documentID,
		Kind:       kind,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// ReplacePassages replaces the document's passages. Repeating the same bodies
// leaves one copy, so a job delivered twice does not double the index.
func (s *Store) ReplacePassages(ctx context.Context, orgID, documentID string, bodies []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	if err := q.DeleteDocumentPassages(ctx, db.DeleteDocumentPassagesParams{OrgID: orgID, DocumentID: documentID}); err != nil {
		return err
	}
	for i, body := range bodies {
		if err := q.AddPassage(ctx, db.AddPassageParams{
			OrgID:      orgID,
			ID:         newID(),
			DocumentID: documentID,
			Ordinal:    int32(i + 1),
			Body:       body,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DocumentPassages lists passages of one document in orgID, in order.
func (s *Store) DocumentPassages(ctx context.Context, orgID, documentID string) ([]Passage, error) {
	rows, err := s.q.ListDocumentPassages(ctx, db.ListDocumentPassagesParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return nil, err
	}
	out := make([]Passage, len(rows))
	for i, row := range rows {
		out[i] = Passage{ID: row.ID, DocumentID: documentID, Ordinal: row.Ordinal, Body: row.Body}
	}
	return out, nil
}

// SearchPassages runs the full-text index for one project. Another org's
// passages are not in the result.
func (s *Store) SearchPassages(ctx context.Context, orgID, projectID, query string, limit int) ([]Passage, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.q.SearchPassages(ctx, db.SearchPassagesParams{
		OrgID:     orgID,
		ProjectID: projectID,
		Query:     query,
		RowLimit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Passage, len(rows))
	for i, row := range rows {
		out[i] = Passage{ID: row.ID, DocumentID: row.DocumentID, Ordinal: row.Ordinal, Body: row.Body}
	}
	return out, nil
}

// SetPassageSystems replaces the system labels of one passage.
func (s *Store) SetPassageSystems(ctx context.Context, orgID, passageID string, systems []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	if err := q.DeletePassageSystems(ctx, db.DeletePassageSystemsParams{OrgID: orgID, PassageID: passageID}); err != nil {
		return err
	}
	for _, id := range systems {
		if id == "" {
			continue
		}
		if err := q.InsertPassageSystem(ctx, db.InsertPassageSystemParams{
			OrgID:     orgID,
			PassageID: passageID,
			SystemID:  id,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// PassageSystems lists system ids stored for one passage.
func (s *Store) PassageSystems(ctx context.Context, orgID, passageID string) ([]string, error) {
	return s.q.ListPassageSystems(ctx, db.ListPassageSystemsParams{OrgID: orgID, PassageID: passageID})
}

// SetPassageEvidence replaces evidence states for one passage.
// state is addressed, not_addressed or unknown.
func (s *Store) SetPassageEvidence(ctx context.Context, orgID, passageID string, states map[string]string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	if err := q.DeletePassageEvidence(ctx, db.DeletePassageEvidenceParams{OrgID: orgID, PassageID: passageID}); err != nil {
		return err
	}
	for id, state := range states {
		switch state {
		case "addressed", "not_addressed", "unknown":
		default:
			return errors.New("evidence state")
		}
		if err := q.UpsertPassageEvidence(ctx, db.UpsertPassageEvidenceParams{
			OrgID:      orgID,
			PassageID:  passageID,
			QuestionID: id,
			State:      state,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// PassageEvidence lists question states for one passage.
func (s *Store) PassageEvidence(ctx context.Context, orgID, passageID string) (map[string]string, error) {
	rows, err := s.q.ListPassageEvidence(ctx, db.ListPassageEvidenceParams{OrgID: orgID, PassageID: passageID})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.QuestionID] = row.State
	}
	return out, nil
}

func trimError(s string) string {
	if len(s) <= 400 {
		return s
	}
	return s[:400]
}
