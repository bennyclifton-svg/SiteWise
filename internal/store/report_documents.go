package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

func reportDocumentSchedule(ctx context.Context, tx pgx.Tx, org, project string) ([]reports.BriefValue, error) {
	return reportDocumentScheduleSelection(ctx, tx, org, project, nil)
}

func reportDocumentScheduleSelection(ctx context.Context, tx pgx.Tx, org, project string, ids []string) ([]reports.BriefValue, error) {
	// Use the same saved identity values as the repository; corrections win.
	rows, err := tx.Query(ctx, `SELECT d.id::text,d.filename,COALESCE(NULLIF((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='number'),''),d.document_number,''),COALESCE(NULLIF((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='revision'),''),d.revision,''),encode(f.sha256,'hex'),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='kind'),''),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='discipline'),''),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='title'),''),
 COALESCE((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='date'),'')
 FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND (($3::text[] IS NULL AND NOT EXISTS(SELECT 1 FROM supersessions s WHERE s.org_id=d.org_id AND s.prior_document_id=d.id)) OR d.id::text=ANY($3::text[])) ORDER BY 3,8,d.id`, org, project, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reports.BriefValue{}
	for rows.Next() {
		var id, name, number, revision, hash, kind, discipline, title, date string
		if err := rows.Scan(&id, &name, &number, &revision, &hash, &kind, &discipline, &title, &date); err != nil {
			return nil, err
		}
		if revision == "" {
			revision = "Not stated"
		}
		if number == "" {
			number = "Not stated"
		}
		if title == "" {
			title = name
		}
		basis, _ := json.Marshal(map[string]any{"sources": []any{map[string]any{"document_id": id, "filename": name, "document_number": number, "revision": revision, "file_sha256": hash, "location": "Document identity", "kind": kind, "discipline": discipline, "title": title, "date": date}}})
		out = append(out, reports.BriefValue{ID: id, Label: "Document revision", Text: fmt.Sprintf("%s; document ID %s; number %s; revision %s; SHA-256 %s", name, id, number, revision, hash), Origin: "document", ReviewStatus: "proposed", Meaning: "stated", Basis: basis})
	}
	return out, rows.Err()
}
