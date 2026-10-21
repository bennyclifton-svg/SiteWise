package store

import (
	"context"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
)

func builderPackage(ctx context.Context, q querier, org, pkg string) (bool, error) {
	var yes bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND id::text=$2 AND kind='works' AND works_scope='head_contract')`, org, pkg).Scan(&yes)
	return yes, err
}

// Retire only unused, generated consultant defaults. User rows, corrections,
// priced rates, issued schedules and existing awards retain their history.
func ensureBuilderSchedule(ctx context.Context, tx pgx.Tx, org, project, pkg string) error {
	_, err := tx.Exec(ctx, `UPDATE schedule_items i SET retired_at=now() WHERE i.org_id=$1::uuid AND i.package_id=$2::uuid AND i.source='generic'
AND i.retired_at IS NULL AND NOT i.comparison_custom AND i.comparison_label IS NULL AND i.comparison_kind IS NULL
AND (i.kind='rate' OR i.key='reimbursables')
AND NOT EXISTS(SELECT 1 FROM submission_prices p JOIN submissions s ON s.org_id=p.org_id AND s.id=p.submission_id WHERE p.org_id=i.org_id AND s.package_id=i.package_id AND p.item_key=i.key AND (p.decided_by='user' OR p.amount_cents IS NOT NULL))
AND NOT EXISTS(SELECT 1 FROM awards a WHERE a.org_id=i.org_id AND a.package_id=i.package_id AND a.undone_at IS NULL)`, org, pkg)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO schedule_items(org_id,project_id,package_id,key,kind,label,detail,base,ordinal,source)
VALUES($1::uuid,$2::uuid,$3::uuid,'stage:lump_sum','stage','Quoted construction price','Whole quoted scope; allowances and breakdowns are not additional prices',true,0,'generic')
ON CONFLICT(org_id,package_id,key) DO NOTHING`, org, project, pkg)
	return err
}

func builderReview(ctx context.Context, q querier, org, sub string) (*proposals.BuilderReview, error) {
	rows, err := q.Query(ctx, `SELECT ordinal,heading,page,body FROM submission_passages WHERE org_id=$1::uuid AND submission_id=$2::uuid ORDER BY ordinal`, org, sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []proposals.Passage
	for rows.Next() {
		var p proposals.Passage
		if err := rows.Scan(&p.Ordinal, &p.Heading, &p.Page, &p.Body); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r := proposals.ReviewBuilderQuote(ps)
	return &r, nil
}

// Builder breakdowns cannot enter arithmetic by moving raw candidates. A PM
// records a confirmed amount on an explicit schedule row through SetSubmissionCell.
func checkBuilderAlignment(ctx context.Context, q querier, org, pkg string, cells []proposals.Cell, selected map[string]bool) error {
	yes, err := builderPackage(ctx, q, org, pkg)
	if err != nil || !yes {
		return err
	}
	for _, c := range cells {
		if selected[c.ID] && c.Key == "" {
			return ErrInvalidInput
		}
	}
	return nil
}
