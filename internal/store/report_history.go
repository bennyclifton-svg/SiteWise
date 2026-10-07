package store

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

type ReportIssueSummary struct {
	ID              string `json:"id"`
	Number          int    `json:"number"`
	ReportingDate   string `json:"reporting_date"`
	SnapshotSHA256  string `json:"snapshot_sha256"`
	BudgetDisclosed bool   `json:"budget_disclosed"`
}

func reportHistory(ctx context.Context, tx pgx.Tx, org, id string) ([]ReportIssueSummary, []reports.Section, error) {
	rows, err := tx.Query(ctx, `SELECT id::text,number,reporting_date::text,snapshot_sha256,budget_disclosed,sections FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND status='issued' ORDER BY number DESC`, org, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []ReportIssueSummary{}
	var last []reports.Section
	for rows.Next() {
		var v ReportIssueSummary
		var raw []byte
		if err := rows.Scan(&v.ID, &v.Number, &v.ReportingDate, &v.SnapshotSHA256, &v.BudgetDisclosed, &raw); err != nil {
			return nil, nil, err
		}
		if len(out) == 0 {
			if err := json.Unmarshal(raw, &last); err != nil {
				return nil, nil, err
			}
		}
		out = append(out, v)
	}
	return out, last, rows.Err()
}
