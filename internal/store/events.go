package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"sitewise/internal/db"
)

const maxEventPage = 500

// StoredEvent is one durable cursor entry for an org. Ids start at 1 and are
// not shared with any other org.
type StoredEvent struct {
	ID         int64
	Kind       string
	DocumentID string
	Payload    string
}

// EventWrite is appended in the same transaction as the change it describes.
type EventWrite struct {
	Kind       string
	DocumentID string
	Payload    string
}

// AppendEvent commits one event. A crash after this returns still leaves the
// row for a reconnecting client.
func (s *Store) AppendEvent(ctx context.Context, orgID, kind, documentID, payload string) (StoredEvent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StoredEvent{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ev, err := appendEvent(ctx, s.q.WithTx(tx), orgID, kind, documentID, payload)
	if err != nil {
		return StoredEvent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredEvent{}, err
	}
	return ev, nil
}

// EventsAfter returns events with id greater than after, oldest first.
func (s *Store) EventsAfter(ctx context.Context, orgID string, after int64, limit int) ([]StoredEvent, error) {
	if limit <= 0 || limit > maxEventPage {
		limit = 100
	}
	rows, err := s.q.ListEventsAfter(ctx, db.ListEventsAfterParams{
		OrgID:    orgID,
		AfterID:  after,
		RowLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]StoredEvent, len(rows))
	for i, row := range rows {
		out[i] = StoredEvent{
			ID:         row.ID,
			Kind:       row.Kind,
			DocumentID: row.DocumentID,
			Payload:    row.Payload,
		}
	}
	return out, nil
}

func appendEvent(ctx context.Context, q *db.Queries, orgID, kind, documentID, payload string) (StoredEvent, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return StoredEvent{}, errors.New("event kind is required")
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		payload = "{}"
	}
	if !json.Valid([]byte(payload)) {
		return StoredEvent{}, errors.New("event payload must be json")
	}
	id, err := q.NextEventID(ctx, orgID)
	if err != nil {
		return StoredEvent{}, err
	}
	doc, err := optionalUUID(documentID)
	if err != nil {
		return StoredEvent{}, err
	}
	if err := q.InsertEvent(ctx, db.InsertEventParams{
		OrgID:      orgID,
		ID:         id,
		Kind:       kind,
		DocumentID: doc,
		Payload:    payload,
	}); err != nil {
		return StoredEvent{}, err
	}
	return StoredEvent{ID: id, Kind: kind, DocumentID: documentID, Payload: payload}, nil
}

func optionalUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if s == "" {
		return u, nil
	}
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}

func filingEventPayload(documentID string, rows []db.ListFilingDecisionsRow) (string, error) {
	decisions := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		decisions = append(decisions, map[string]string{
			"field": row.Field,
			"value": row.Value,
			"band":  row.Band,
		})
	}
	body, err := json.Marshal(map[string]any{
		"document_id": documentID,
		"status":      StatusFiled,
		"decisions":   decisions,
	})
	if err != nil {
		return "", err
	}
	return string(body), nil
}
