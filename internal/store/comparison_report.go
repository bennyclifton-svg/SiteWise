package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// Ratings a PM gives a firm on one criterion. Ratings are only ever a
// user's: nothing scores firms.
var ratings = map[string]bool{"strong": true, "acceptable": true, "concern": true}

// ResponsePassage is one section of a proposal that answers a criterion.
type ResponsePassage struct {
	Ordinal int    `json:"ordinal"`
	Heading string `json:"heading"`
	Page    int    `json:"page"`
	Excerpt string `json:"excerpt"`
}

// CriterionResponse is one firm's answer to one non-price criterion.
type CriterionResponse struct {
	Passages  []ResponsePassage `json:"passages"`
	Rating    string            `json:"rating"`
	Band      string            `json:"band"`
	DecidedBy string            `json:"decided_by"`
}

// ReportColumn is one firm in the report.
type ReportColumn struct {
	SubmissionID string                       `json:"submission_id"`
	Responses    map[string]CriterionResponse `json:"responses"`
}

// ComparisonReport is the one-page report: the price table, the non-price
// matrix, and the recommendation the PM edits and signs off.
type ComparisonReport struct {
	Criteria       []rfp.ScheduleItem `json:"criteria"`
	Columns        []ReportColumn     `json:"columns"`
	Recommendation string             `json:"recommendation"`
	Draft          string             `json:"draft"`
	EditedBy       string             `json:"edited_by"`
	SignedOffBy    string             `json:"signed_off_by"`
	SignedOffAt    *time.Time         `json:"signed_off_at"`
	Version        int64              `json:"version"`
}

// ReadComparisonReport assembles the report for a package's live columns.
// draft is the code's factual summary of the price table, offered until
// the PM writes their own recommendation.
func (s *Store) ReadComparisonReport(ctx context.Context, org, project, pkg string, table PriceTable, draft string) (ComparisonReport, error) {
	rep := ComparisonReport{Columns: []ReportColumn{}, Criteria: []rfp.ScheduleItem{}, Draft: draft}
	for _, it := range table.Items {
		if it.Kind == rfp.ItemCriterion {
			rep.Criteria = append(rep.Criteria, it.ScheduleItem)
		}
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return rep, err
	}
	defer tx.Rollback(ctx)
	for _, c := range table.Columns {
		col := ReportColumn{SubmissionID: c.Submission.ID, Responses: map[string]CriterionResponse{}}
		passages := map[int]ResponsePassage{}
		prow, err := tx.Query(ctx, `SELECT ordinal, heading, page, left(body, 700) FROM submission_passages WHERE org_id=$1::uuid AND submission_id::text=$2`, org, c.Submission.ID)
		if err != nil {
			return rep, err
		}
		for prow.Next() {
			var p ResponsePassage
			var ord, page int32
			if err := prow.Scan(&ord, &p.Heading, &page, &p.Excerpt); err != nil {
				prow.Close()
				return rep, err
			}
			p.Ordinal, p.Page = int(ord), int(page)
			passages[p.Ordinal] = p
		}
		prow.Close()
		rrow, err := tx.Query(ctx, `SELECT item_key, passages, COALESCE(rating,''), band, decided_by FROM submission_responses WHERE org_id=$1::uuid AND submission_id::text=$2`, org, c.Submission.ID)
		if err != nil {
			return rep, err
		}
		for rrow.Next() {
			var key, rating, band, by string
			var ords []int32
			if err := rrow.Scan(&key, &ords, &rating, &band, &by); err != nil {
				rrow.Close()
				return rep, err
			}
			r := CriterionResponse{Passages: []ResponsePassage{}, Rating: rating, Band: band, DecidedBy: by}
			// A passage Jev picked below its band is not shown as an answer.
			if band != proposals.BandBlank && band != proposals.BandGrey || by == proposals.ByUser {
				for _, o := range ords {
					if p, ok := passages[int(o)]; ok {
						r.Passages = append(r.Passages, p)
					}
				}
			}
			col.Responses[key] = r
		}
		rrow.Close()
		rep.Columns = append(rep.Columns, col)
	}
	err = tx.QueryRow(ctx, `SELECT recommendation, COALESCE((SELECT email FROM users u WHERE u.org_id=r.org_id AND u.id=r.recommendation_by),''),
 COALESCE((SELECT email FROM users u WHERE u.org_id=r.org_id AND u.id=r.signed_off_by),''), signed_off_at, version
FROM comparison_reports r WHERE org_id=$1::uuid AND package_id::text=$2`, org, pkg).Scan(&rep.Recommendation, &rep.EditedBy, &rep.SignedOffBy, &rep.SignedOffAt, &rep.Version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return rep, err
	}
	return rep, tx.Commit(ctx)
}

// RateResponse records the PM's rating of one firm on one criterion; an
// empty rating clears it.
func (s *Store) RateResponse(ctx context.Context, org, project, sub, key, rating, user string) error {
	if rating != "" && !ratings[rating] || len(key) > 200 {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := lockSubmission(ctx, tx, org, project, sub); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO submission_responses (org_id,project_id,submission_id,item_key,rating,rated_by,decided_by,band)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,NULLIF($5,''),CASE WHEN $5='' THEN NULL ELSE $6::uuid END,'user','green')
ON CONFLICT (org_id,submission_id,item_key) DO UPDATE SET rating=EXCLUDED.rating, rated_by=EXCLUDED.rated_by, updated_at=now()`, org, project, sub, key, rating, user); err != nil {
		return err
	}
	return s.finishSubmissionWrite(ctx, tx, org, project, sub)
}

// SaveRecommendation stores the PM's recommendation. signOff records the
// PM's sign-off; any later edit clears it.
func (s *Store) SaveRecommendation(ctx context.Context, org, project, pkg, text, user string, signOff bool) error {
	if len(text) > 4000 {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := packageInProject(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	var signer *string
	if signOff {
		signer = &user
	}
	if _, err := tx.Exec(ctx, `INSERT INTO comparison_reports (org_id,project_id,package_id,recommendation,recommendation_by,signed_off_by,signed_off_at)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5::uuid,$6::uuid,CASE WHEN $6::uuid IS NULL THEN NULL ELSE now() END)
ON CONFLICT (org_id,package_id) DO UPDATE SET recommendation=EXCLUDED.recommendation, recommendation_by=EXCLUDED.recommendation_by,
 signed_off_by=EXCLUDED.signed_off_by, signed_off_at=EXCLUDED.signed_off_at, updated_at=now(), version=comparison_reports.version+1`, org, project, pkg, text, user, signer); err != nil {
		return err
	}
	if err := procurementEvent(ctx, tx, s, org, project, "", "report"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
