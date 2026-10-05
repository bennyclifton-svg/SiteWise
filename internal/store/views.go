package store

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/db"
)

const (
	// EventFiling is written by the filing transaction.
	EventFiling = "filing"
	// EventCorrection is written with a user correction.
	EventCorrection = "correction"
	// EventNotFiled is written when stored bytes cannot be filed.
	EventNotFiled = "not_filed"

	maxPendingIntake = 1000
)

// FieldView is one field as the UI shows it.
type FieldView struct {
	Field           string   `json:"field"`
	Value           string   `json:"value"`
	Band            string   `json:"band"`
	DecidedBy       string   `json:"decided_by"`
	QuestionVersion string   `json:"question_version,omitempty"`
	Confidence      *float64 `json:"confidence,omitempty"`
}

// DocumentView is one filing with its fields, for the project list.
type DocumentView struct {
	ID                string `json:"id"`
	ProjectID         string `json:"project_id"`
	Filename          string `json:"filename"`
	Status            string `json:"status"`
	TextPages         int32  `json:"text_pages"`
	TextEmptyPages    int32  `json:"text_empty_pages"`
	TextSourceVersion string `json:"text_source_version"`
	TextStatus        string `json:"text_status,omitempty"`
	// ProfileRead is the user's profile reading setting: auto, read or skip.
	ProfileRead    string            `json:"profile_read"`
	Reason         string            `json:"reason,omitempty"`
	Number         string            `json:"number,omitempty"`
	Revision       string            `json:"revision,omitempty"`
	SupersedesID   string            `json:"supersedes_id,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Fields         []FieldView       `json:"fields"`
	SourceID       string            `json:"source_id,omitempty"`
	SourceFilename string            `json:"source_filename,omitempty"`
	SheetPage      int               `json:"sheet_page,omitempty"`
	SheetTotal     int               `json:"sheet_total,omitempty"`
	Expansion      *DrawingExpansion `json:"expansion,omitempty"`
}

// PendingIntake is a filing to resume after a restart.
type PendingIntake struct {
	OrgID      string
	DocumentID string
}

// ListProjects lists orgID's projects, newest first.
func (s *Store) ListProjects(ctx context.Context, orgID string) ([]Project, error) {
	rows, err := s.q.ListOrgProjects(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = Project{ID: row.ID, Name: row.Name}
	}
	return out, nil
}

// ProjectDocumentViews lists a project's filings with their fields in two
// queries, newest first. Another org's project returns an empty list.
func (s *Store) ProjectDocumentViews(ctx context.Context, orgID, projectID string) ([]DocumentView, error) {
	docs, err := s.q.ListProjectDocumentViews(ctx, db.ListProjectDocumentViewsParams{OrgID: orgID, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(docs))
	for i, d := range docs {
		ids[i] = d.ID
	}
	fields, err := s.q.ListDocumentDecisionViews(ctx, db.ListDocumentDecisionViewsParams{OrgID: orgID, DocumentIds: ids})
	if err != nil {
		return nil, err
	}
	byDoc := make(map[string][]FieldView, len(docs))
	for _, row := range fields {
		byDoc[row.DocumentID] = append(byDoc[row.DocumentID], FieldView{
			Field:           row.Field,
			Value:           row.Value,
			Band:            row.Band,
			DecidedBy:       row.DecidedBy,
			QuestionVersion: row.QuestionVersion,
			Confidence:      confidencePtr(row.Confidence),
		})
	}
	out := make([]DocumentView, len(docs))
	for i, row := range docs {
		out[i] = DocumentView{
			ID:         row.ID,
			ProjectID:  projectID,
			Filename:   row.Filename,
			Status:     row.Status,
			TextStatus: row.TextStatus, ProfileRead: row.ProfileRead,
			TextPages: row.TextPages, TextEmptyPages: row.TextEmptyPages, TextSourceVersion: row.TextSourceVersion,
			Reason:       row.Reason,
			Number:       row.DocumentNumber,
			Revision:     row.Revision,
			SupersedesID: row.SupersedesID,
			CreatedAt:    row.CreatedAt,
			Fields:       nonNilFields(byDoc[row.ID]),
		}
	}
	if err := s.sheetViews(ctx, orgID, projectID, out); err != nil {
		return nil, err
	}
	byID := make(map[string]DocumentView, len(out))
	for _, d := range out {
		byID[d.ID] = d
	}
	group := func(d DocumentView) DocumentView {
		if source, ok := byID[d.SourceID]; ok {
			return source
		}
		return d
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := group(out[i]), group(out[j])
		if a.ID == b.ID {
			return out[i].SheetPage < out[j].SheetPage
		}
		if a.CreatedAt.Equal(b.CreatedAt) {
			return a.ID < b.ID
		}
		return a.CreatedAt.After(b.CreatedAt)
	})
	return out, nil
}

// DocumentView loads one filing visible to orgID.
func (s *Store) DocumentView(ctx context.Context, orgID, documentID string) (DocumentView, error) {
	row, err := s.q.GetDocumentView(ctx, db.GetDocumentViewParams{OrgID: orgID, ID: documentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentView{}, ErrNotFound
	}
	if err != nil {
		return DocumentView{}, err
	}
	rows, err := s.q.ListFilingDecisions(ctx, db.ListFilingDecisionsParams{OrgID: orgID, DocumentID: documentID})
	if err != nil {
		return DocumentView{}, err
	}
	out := DocumentView{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Filename:   row.Filename,
		Status:     row.Status,
		TextStatus: row.TextStatus, ProfileRead: row.ProfileRead,
		TextPages: row.TextPages, TextEmptyPages: row.TextEmptyPages, TextSourceVersion: row.TextSourceVersion,
		Reason:       row.Reason,
		Number:       row.DocumentNumber,
		Revision:     row.Revision,
		SupersedesID: row.SupersedesID,
		CreatedAt:    row.CreatedAt,
		Fields:       fieldViews(rows),
	}
	views := []DocumentView{out}
	if err := s.sheetViews(ctx, orgID, row.ProjectID, views); err != nil {
		return DocumentView{}, err
	}
	return views[0], nil
}

// LatestEventID is orgID's newest event id, or zero. A client that reads it
// before loading a list and resumes from it misses nothing committed later.
func (s *Store) LatestEventID(ctx context.Context, orgID string) (int64, error) {
	return s.q.LatestEventID(ctx, orgID)
}

// MarkNotFiled keeps the stored bytes, records why they are not filed, closes
// the intake job and writes the event in one transaction. A document that is
// no longer pending is left as it is and ErrNotFound is returned.
func (s *Store) MarkNotFiled(ctx context.Context, orgID, documentID, reason string) error {
	if reason == "" || len(reason) > 64 {
		return errors.New("invalid not_filed reason")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)
	n, err := q.MarkPendingNotFiled(ctx, db.MarkPendingNotFiledParams{Reason: reason, OrgID: orgID, ID: documentID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if _, err := q.FinishIntakeJob(ctx, db.FinishIntakeJobParams{OrgID: orgID, DocumentID: documentID}); err != nil {
		return err
	}
	payload, err := documentEventPayload(documentID, StatusNotFiled, reason, nil)
	if err != nil {
		return err
	}
	if _, err := appendEvent(ctx, q, orgID, EventNotFiled, documentID, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PendingIntake lists filings, across every org, that stopped before their
// filing committed. Each row carries its own org for the resumed filing.
func (s *Store) PendingIntake(ctx context.Context) ([]PendingIntake, error) {
	rows, err := s.q.ListPendingIntake(ctx, maxPendingIntake)
	if err != nil {
		return nil, err
	}
	out := make([]PendingIntake, len(rows))
	for i, row := range rows {
		out[i] = PendingIntake{OrgID: row.OrgID, DocumentID: row.DocumentID}
	}
	return out, nil
}

// documentEventPayload is the body of filing, correction and not_filed
// events. Fields carry decided_by so a client never shows a correction as a
// rule or Jev value.
func documentEventPayload(documentID, status, reason string, rows []db.ListFilingDecisionsRow) (string, error) {
	body, err := json.Marshal(struct {
		DocumentID string      `json:"document_id"`
		Status     string      `json:"status"`
		Reason     string      `json:"reason,omitempty"`
		Fields     []FieldView `json:"fields"`
	}{documentID, status, reason, fieldViews(rows)})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func fieldViews(rows []db.ListFilingDecisionsRow) []FieldView {
	out := make([]FieldView, len(rows))
	for i, row := range rows {
		out[i] = FieldView{
			Field:           row.Field,
			Value:           row.Value,
			Band:            row.Band,
			DecidedBy:       row.DecidedBy,
			QuestionVersion: row.QuestionVersion,
			Confidence:      confidencePtr(row.Confidence),
		}
	}
	return out
}

func nonNilFields(list []FieldView) []FieldView {
	if list == nil {
		return []FieldView{}
	}
	return list
}
