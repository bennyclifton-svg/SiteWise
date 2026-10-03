package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"sitewise/internal/db"
)

const (
	fieldNumber     = "number"
	fieldRevision   = "revision"
	fieldSupersedes = "supersedes"
)

// StoredDecision is one field row visible to the caller's org.
type StoredDecision struct {
	Field           string
	Value           string
	Band            string
	DecidedBy       string
	QuestionVersion string
	Confidence      *float64
	Version         int64
}

// NumberedDocument is a project document that can sit in a number series.
type NumberedDocument struct {
	ID        string
	ProjectID string
	Number    string
	Revision  string
	Status    string
}

// DecisionWrite is one field to store. ExpectedVersion is the version read
// before the Jev call; 0 means no row was seen. A user decision is left as it
// is when this write is stale or the row is already a correction.
type DecisionWrite struct {
	Field           string
	Value           string
	Band            string
	DecidedBy       string
	QuestionVersion string
	Confidence      *float64
	ExpectedVersion int64
}

// CommitFiling is the atomic result of one intake pass.
type CommitFiling struct {
	OCR       bool
	Decisions []DecisionWrite
	PDFPages  int
	// PriorID is the document this filing supersedes. Empty stores no link.
	// The link is refused unless both rows share an org, project and number,
	// the ids differ, and the existing chain does not already reach this one.
	PriorID  string
	RetryJev bool
}

// FilingOutcome is what the transaction committed, or the filing already stored.
type FilingOutcome struct {
	Status       string
	Number       string
	Revision     string
	SupersedesID string
	Decisions    []StoredDecision
	AlreadyFiled bool
}

// DocumentDecisions lists field judgments for one document in orgID.
func (s *Store) DocumentDecisions(ctx context.Context, orgID, documentID string) ([]StoredDecision, error) {
	rows, err := s.q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return nil, err
	}
	return storedDecisions(rows), nil
}

// ProjectDocuments lists documents in one project. Callers scope number series
// themselves; another org's rows are not in the result.
func (s *Store) ProjectDocuments(ctx context.Context, orgID, projectID string) ([]NumberedDocument, error) {
	rows, err := s.q.ListProjectDocuments(ctx, db.ListProjectDocumentsParams{OrgID: orgID, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]NumberedDocument, len(rows))
	for i, row := range rows {
		out[i] = NumberedDocument{
			ID:        row.ID,
			ProjectID: row.ProjectID,
			Number:    row.DocumentNumber,
			Revision:  row.Revision,
			Status:    row.Status,
		}
	}
	return out, nil
}

// OrgSupersessions lists supersession links in one org.
func (s *Store) OrgSupersessions(ctx context.Context, orgID string) ([][2]string, error) {
	rows, err := s.q.ListOrgSupersessions(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([][2]string, len(rows))
	for i, row := range rows {
		out[i] = [2]string{row.DocumentID, row.PriorDocumentID}
	}
	return out, nil
}

// CorrectDecision stores a user value. It replaces a rule or Jev value and
// bumps the version so an in-flight filing cannot write over it. The
// correction event commits with it, so a reconnecting client sees it.
func (s *Store) CorrectDecision(ctx context.Context, orgID, documentID, field, value string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: documentID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	n, err := q.CorrectFilingDecision(ctx, db.CorrectFilingDecisionParams{
		OrgID:      orgID,
		ID:         newID(),
		DocumentID: documentID,
		Field:      field,
		Value:      strPtr(value),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	current, err := q.GetDocument(ctx, db.GetDocumentParams{OrgID: orgID, ID: documentID})
	if err != nil {
		return err
	}
	payload, err := documentEventPayload(documentID, doc.Status, current.Reason, rows)
	if err != nil {
		return err
	}
	if _, err := appendEvent(ctx, q, orgID, EventCorrection, documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CommitFiling stores decisions, document identity, a supersession link and
// the background jobs in one transaction.
func (s *Store) CommitFiling(ctx context.Context, orgID, documentID string, in CommitFiling) (FilingOutcome, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FilingOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	out, err := commitFilingTx(ctx, tx, orgID, documentID, in)
	if err != nil {
		return FilingOutcome{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FilingOutcome{}, err
	}
	return out, nil
}

func commitFilingTx(ctx context.Context, tx pgx.Tx, orgID, documentID string, in CommitFiling) (FilingOutcome, error) {
	q := db.New(tx)

	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return FilingOutcome{}, ErrNotFound
	}
	if err != nil {
		return FilingOutcome{}, err
	}
	if doc.Status != StatusPending {
		out, err := readOutcome(ctx, q, orgID, documentID)
		if err != nil {
			return FilingOutcome{}, err
		}
		out.AlreadyFiled = true

		return out, nil
	}

	for _, d := range in.Decisions {
		conf := 0.0
		set := false
		if d.Confidence != nil {
			conf = *d.Confidence
			set = true
		}
		if _, err := q.UpsertFilingDecision(ctx, db.UpsertFilingDecisionParams{
			OrgID:           orgID,
			ID:              newID(),
			DocumentID:      documentID,
			Field:           d.Field,
			Value:           strPtr(d.Value),
			Band:            d.Band,
			DecidedBy:       d.DecidedBy,
			QuestionVersion: strPtr(d.QuestionVersion),
			ConfidenceSet:   set,
			Confidence:      conf,
			ExpectedVersion: d.ExpectedVersion,
		}); err != nil {
			return FilingOutcome{}, err
		}
	}

	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	number, revision := identityOf(rows)
	n, err := q.SetFiledIdentity(ctx, db.SetFiledIdentityParams{
		DocumentNumber: strPtr(number),
		Revision:       strPtr(revision),
		OrgID:          orgID,
		ID:             documentID,
	})
	if err != nil {
		return FilingOutcome{}, err
	}
	if n == 0 {
		return FilingOutcome{}, ErrNotFound
	}

	linked := ""
	if in.PriorID != "" && number != "" && in.PriorID != documentID {
		if _, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: in.PriorID}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return FilingOutcome{}, err
		}
		links, err := orgLinks(ctx, q, orgID)
		if err != nil {
			return FilingOutcome{}, err
		}
		if !supersessionCycle(links, documentID, in.PriorID) {
			n, err := q.InsertScopedSupersession(ctx, db.InsertScopedSupersessionParams{
				OrgID:           orgID,
				DocumentID:      documentID,
				PriorDocumentID: in.PriorID,
			})
			if err != nil {
				return FilingOutcome{}, err
			}
			if n > 0 {
				linked = in.PriorID
			}
		}
	}
	if in.PriorID != "" && linked == "" {
		if err := blankSupersession(ctx, q, orgID, documentID, in.PriorID); err != nil {
			return FilingOutcome{}, err
		}
	}

	n, err = q.FinishIntakeJob(ctx, db.FinishIntakeJobParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	if n == 0 {
		return FilingOutcome{}, errors.New("intake job missing")
	}
	if err := enqueueKind(ctx, q, orgID, documentID, JobKindFullText); err != nil {
		return FilingOutcome{}, err
	}
	if in.RetryJev {
		if err := enqueueKind(ctx, q, orgID, documentID, JobKindJevRetry); err != nil {
			return FilingOutcome{}, err
		}
	}
	rows, err = q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	payload, err := filingEventPayload(documentID, rows)
	if in.OCR {
		if _, err := tx.Exec(ctx, `UPDATE documents SET reason='ocr_review' WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, documentID); err != nil {
			return FilingOutcome{}, err
		}
		payload, err = documentEventPayload(documentID, StatusFiled, "ocr_review", rows)
	}
	if err != nil {
		return FilingOutcome{}, err
	}
	if _, err := appendEvent(ctx, q, orgID, EventFiling, documentID, payload); err != nil {
		return FilingOutcome{}, err
	}
	if in.PDFPages > 1 {
		if _, err := tx.Exec(ctx, `INSERT INTO drawing_expansions (org_id, source_id, page_count)
		SELECT $1::uuid, $2::uuid, $3 WHERE EXISTS (SELECT 1 FROM decisions
		WHERE org_id=$1::uuid AND document_id=$2::uuid AND field='kind' AND value='drawing' AND band IN ('amber','green'))
		ON CONFLICT DO NOTHING`, orgID, documentID, in.PDFPages); err != nil {
			return FilingOutcome{}, err
		}
		if in.OCR {
			if _, err := tx.Exec(ctx, `UPDATE drawing_expansions SET status='review',reason='OCR read first-page identity only. Separate scanned sheets have not been filed.' WHERE org_id=$1::uuid AND source_id=$2::uuid AND status='pending'`, orgID, documentID); err != nil {
				return FilingOutcome{}, err
			}
		}
	}
	out, err := readOutcome(ctx, q, orgID, documentID)
	if err != nil {
		return FilingOutcome{}, err
	}
	return out, nil
}

func enqueueKind(ctx context.Context, q *db.Queries, orgID, documentID, kind string) error {
	_, err := q.EnqueueJob(ctx, db.EnqueueJobParams{
		OrgID:      orgID,
		ID:         newID(),
		DocumentID: documentID,
		Kind:       kind,
	})
	if err != nil {
		return err
	}
	// The caller holds the document lock. A previously completed background
	// stage must not prevent a fresh filing of these same stored bytes.
	return nil
}

func blankSupersession(ctx context.Context, q *db.Queries, orgID, documentID, priorID string) error {
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Field != fieldSupersedes || row.DecidedBy == "user" || row.Band != "green" {
			continue
		}
		value := row.Value
		if value == "" {
			value = priorID
		}
		_, err := q.UpsertFilingDecision(ctx, db.UpsertFilingDecisionParams{
			OrgID:           orgID,
			ID:              newID(),
			DocumentID:      documentID,
			Field:           fieldSupersedes,
			Value:           strPtr(value),
			Band:            "blank",
			DecidedBy:       row.DecidedBy,
			QuestionVersion: strPtr(row.QuestionVersion),
			ConfidenceSet:   row.Confidence.Valid,
			Confidence:      row.Confidence.Float64,
			ExpectedVersion: row.Version,
		})
		return err
	}
	return nil
}

func orgLinks(ctx context.Context, q *db.Queries, orgID string) ([][2]string, error) {
	rows, err := q.ListOrgSupersessions(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([][2]string, len(rows))
	for i, row := range rows {
		out[i] = [2]string{row.DocumentID, row.PriorDocumentID}
	}
	return out, nil
}

func supersessionCycle(links [][2]string, documentID, priorID string) bool {
	if documentID == "" || priorID == "" || documentID == priorID {
		return true
	}
	next := make(map[string]string, len(links))
	for _, link := range links {
		if link[0] == documentID {
			return true
		}
		next[link[0]] = link[1]
	}
	seen := map[string]struct{}{}
	cur := priorID
	for {
		if cur == documentID {
			return true
		}
		if _, ok := seen[cur]; ok {
			return true
		}
		seen[cur] = struct{}{}
		n, ok := next[cur]
		if !ok {
			return false
		}
		cur = n
	}
}

func identityOf(rows []db.ListFilingDecisionsRow) (number, revision string) {
	for _, row := range rows {
		if row.Value == "" {
			continue
		}
		switch row.Field {
		case fieldNumber:
			number = row.Value
		case fieldRevision:
			revision = row.Value
		}
	}
	return number, revision
}

func readOutcome(ctx context.Context, q *db.Queries, orgID, documentID string) (FilingOutcome, error) {
	doc, err := q.GetDocument(ctx, db.GetDocumentParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return FilingOutcome{}, ErrNotFound
	}
	if err != nil {
		return FilingOutcome{}, err
	}
	prior, err := q.GetSupersession(ctx, db.GetSupersessionParams{OrgID: orgID, DocumentID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		prior = ""
	} else if err != nil {
		return FilingOutcome{}, err
	}
	rows, err := q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return FilingOutcome{}, err
	}
	return FilingOutcome{
		Status:       doc.Status,
		Number:       doc.DocumentNumber,
		Revision:     doc.Revision,
		SupersedesID: prior,
		Decisions:    storedDecisions(rows),
	}, nil
}

func storedDecisions(rows []db.ListFilingDecisionsRow) []StoredDecision {
	out := make([]StoredDecision, len(rows))
	for i, row := range rows {
		out[i] = StoredDecision{
			Field:           row.Field,
			Value:           row.Value,
			Band:            row.Band,
			DecidedBy:       row.DecidedBy,
			QuestionVersion: row.QuestionVersion,
			Confidence:      confidencePtr(row.Confidence),
			Version:         row.Version,
		}
	}
	return out
}

func confidencePtr(v pgtype.Float8) *float64 {
	if !v.Valid {
		return nil
	}
	c := v.Float64
	return &c
}
