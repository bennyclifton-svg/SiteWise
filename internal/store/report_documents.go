package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

func reportDocumentSchedule(ctx context.Context, tx pgx.Tx, org, project string) ([]reports.BriefValue, error) {
	rows, err := tx.Query(ctx, `SELECT d.id::text,d.filename,COALESCE(d.document_number,''),COALESCE(d.revision,''),encode(f.sha256,'hex') FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND NOT EXISTS(SELECT 1 FROM supersessions s WHERE s.org_id=d.org_id AND s.prior_document_id=d.id) ORDER BY d.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reports.BriefValue{}
	for rows.Next() {
		var id, name, number, revision, hash string
		if err := rows.Scan(&id, &name, &number, &revision, &hash); err != nil {
			return nil, err
		}
		if revision == "" {
			revision = "Not stated"
		}
		if number == "" {
			number = "Not stated"
		}
		basis, _ := json.Marshal(map[string]any{"sources": []any{map[string]any{"document_id": id, "filename": name, "document_number": number, "revision": revision, "file_sha256": hash, "location": "Document identity"}}})
		out = append(out, reports.BriefValue{ID: id, Label: "Document revision", Text: fmt.Sprintf("%s; document ID %s; number %s; revision %s; SHA-256 %s", name, id, number, revision, hash), Origin: "document", ReviewStatus: "proposed", Meaning: "stated", Basis: basis})
	}
	return out, rows.Err()
}
