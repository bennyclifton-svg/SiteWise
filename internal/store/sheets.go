package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/db"
)

type DrawingExpansion struct {
	SourceID  string `json:"source_id"`
	PageCount int    `json:"page_count"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

type SheetWrite struct {
	SHA256    []byte
	ByteSize  int64
	Decisions []DecisionWrite
	PriorID   string
}

// DrawingSetPageKind admits schedules as sheets of an already confirmed drawing
// set. It does not make standalone schedules or reports eligible for splitting.
func DrawingSetPageKind(kind, band string) bool {
	return (kind == "drawing" || kind == "schedule") && (band == "amber" || band == "green")
}

func (s *Store) RetryDrawingExpansion(ctx context.Context, orgID, sourceID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE drawing_expansions SET status='pending',reason='' WHERE org_id=$1::uuid AND source_id=$2::uuid AND status='review'`, orgID, sourceID)
	return err
}

func (s *Store) DrawingExpansion(ctx context.Context, orgID, sourceID string) (DrawingExpansion, error) {
	var out DrawingExpansion
	err := s.pool.QueryRow(ctx, `SELECT source_id::text,page_count,status,reason FROM drawing_expansions WHERE org_id=$1::uuid AND source_id=$2::uuid`, orgID, sourceID).Scan(&out.SourceID, &out.PageCount, &out.Status, &out.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *Store) PendingDrawingExpansions(ctx context.Context) ([]PendingIntake, error) {
	rows, err := s.pool.Query(ctx, `SELECT org_id::text,source_id::text FROM drawing_expansions WHERE status='pending' ORDER BY org_id,source_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingIntake
	for rows.Next() {
		var p PendingIntake
		if err := rows.Scan(&p.OrgID, &p.DocumentID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ReviewDrawingExpansion(ctx context.Context, orgID, sourceID, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE drawing_expansions SET status='review',reason=$3 WHERE org_id=$1::uuid AND source_id=$2::uuid AND status='pending'`, orgID, sourceID, reason)
	if err != nil {
		return err
	}
	if _, err = appendEvent(ctx, db.New(tx), orgID, "sheets", sourceID, `{}`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) sheetViews(ctx context.Context, orgID, projectID string, views []DocumentView) error {
	byID := make(map[string]*DocumentView, len(views))
	for i := range views {
		byID[views[i].ID] = &views[i]
	}
	rows, err := s.pool.Query(ctx, `SELECT e.source_id::text,e.page_count,e.status,e.reason,d.filename,
	COALESCE(sh.document_id::text,''),COALESCE(sh.page_number,0)
	FROM drawing_expansions e JOIN documents d ON d.org_id=e.org_id AND d.id=e.source_id
	LEFT JOIN drawing_sheets sh ON sh.org_id=e.org_id AND sh.source_id=e.source_id
	WHERE e.org_id=$1::uuid AND d.project_id=$2::uuid`, orgID, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e DrawingExpansion
		var filename, child string
		var page int
		if err := rows.Scan(&e.SourceID, &e.PageCount, &e.Status, &e.Reason, &filename, &child, &page); err != nil {
			return err
		}
		if v := byID[e.SourceID]; v != nil {
			v.Expansion = &e
		}
		if v := byID[child]; v != nil {
			v.SourceID = e.SourceID
			v.SourceFilename = filename
			v.SheetPage = page
			v.SheetTotal = e.PageCount
		}
	}
	return rows.Err()
}

// PublishDrawingSheets commits all pages together. The source lock serialises
// duplicate workers. A failure leaves the source intact and no partial schedule.
func (s *Store) PublishDrawingSheets(ctx context.Context, orgID, sourceID string, sheets []SheetWrite) error {
	if len(sheets) < 2 || len(sheets) > 200 {
		return errors.New("invalid sheet count")
	}
	for _, sheet := range sheets {
		if len(sheet.SHA256) != 32 || sheet.ByteSize <= 0 {
			return ErrMetadata
		}
		drawing := false
		for _, d := range sheet.Decisions {
			if d.Field == "kind" && DrawingSetPageKind(d.Value, d.Band) {
				drawing = true
			}
		}
		if !drawing {
			return errors.New("sheet kind is not admitted within a drawing set")
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	doc, err := q.LockFilingDocument(ctx, db.LockFilingDocumentParams{OrgID: orgID, ID: sourceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var count int
	var status string
	err = tx.QueryRow(ctx, `SELECT page_count,status FROM drawing_expansions WHERE org_id=$1::uuid AND source_id=$2::uuid FOR UPDATE`, orgID, sourceID).Scan(&count, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == "complete" {
		return nil
	}
	if status != "pending" || count != len(sheets) {
		return errors.New("expansion state changed")
	}
	var admitted bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM decisions WHERE org_id=$1::uuid AND document_id=$2::uuid AND field='kind' AND value='drawing' AND band IN ('green','amber'))`, orgID, sourceID).Scan(&admitted); err != nil {
		return err
	}
	if !admitted {
		return errors.New("source kind changed during expansion")
	}
	// A container is not a sheet identity. Release its first-page identity in
	// the same transaction that publishes the independently classified children.
	if _, err = tx.Exec(ctx, `UPDATE documents SET status='split',document_number=NULL,revision=NULL WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, sourceID); err != nil {
		return err
	}
	for page, sheet := range sheets {
		fileID := newID()
		err = tx.QueryRow(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'application/pdf')
		ON CONFLICT(org_id,project_id,sha256) DO UPDATE SET sha256=EXCLUDED.sha256 RETURNING id::text`, orgID, fileID, doc.ProjectID, sheet.SHA256, sheet.ByteSize).Scan(&fileID)
		if err != nil {
			return err
		}
		child, err := insertFiling(ctx, q, orgID, fileID, CommitIntake{ProjectID: doc.ProjectID, Filename: fmt.Sprintf("Sheet %d.pdf", page+1), SHA256: sheet.SHA256, ByteSize: sheet.ByteSize, MediaType: "application/pdf"})
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE documents SET identity_page=$3 WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, child.DocumentID, page+1); err != nil {
			return err
		}
		if _, err = commitFilingTx(ctx, tx, orgID, child.DocumentID, CommitFiling{Decisions: sheet.Decisions, PriorID: sheet.PriorID}); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO drawing_sheets(org_id,source_id,page_number,document_id) VALUES($1::uuid,$2::uuid,$3,$4::uuid)`, orgID, sourceID, page+1, child.DocumentID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE drawing_expansions SET status='complete',reason='' WHERE org_id=$1::uuid AND source_id=$2::uuid`, orgID, sourceID); err != nil {
		return err
	}
	if _, err = appendEvent(ctx, q, orgID, "sheets", sourceID, `{}`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
