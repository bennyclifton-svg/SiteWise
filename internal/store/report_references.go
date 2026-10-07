package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

func saveReportReferences(ctx context.Context, tx pgx.Tx, org, id string, refs []reports.Reference) error {
	if _, err := tx.Exec(ctx, `DELETE FROM report_references WHERE org_id=$1::uuid AND report_version_id=$2::uuid`, org, id); err != nil {
		return err
	}
	raw, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO report_references(org_id,report_version_id,citation_id,label,anchor_id,basis)
 SELECT $1::uuid,$2::uuid,r.citation_id,r.label,r.anchor_id,r.basis FROM jsonb_to_recordset($3::jsonb) AS r(citation_id text,label text,anchor_id text,basis jsonb)`, org, id, raw)
	return err
}

func readReportReferences(ctx context.Context, tx pgx.Tx, org, id string) ([]reports.Reference, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('citation_id',citation_id,'label',label,'anchor_id',anchor_id,'basis',basis) ORDER BY label,substring(citation_id FROM 2)::int),'[]'::jsonb) FROM report_references WHERE org_id=$1::uuid AND report_version_id=$2::uuid`, org, id).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var refs []reports.Reference
	err = json.Unmarshal(raw, &refs)
	return refs, err
}
