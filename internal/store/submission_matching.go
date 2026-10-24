package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/db"
	"sitewise/internal/proposals"
)

// matchUnassignedSubmissions uses current filing decisions, not a stale AI
// reading. The caller holds the project lock. A user's explicit blank is a
// choice too; completed/withdrawn submissions and existing assignments stand.
func matchUnassignedSubmissions(ctx context.Context, tx pgx.Tx, org, project, document string) error {
	rows, err := tx.Query(ctx, `SELECT s.id::text, p.id::text
FROM submissions s
JOIN decisions d ON d.org_id=s.org_id AND d.document_id=s.document_id AND d.field='discipline'
JOIN packages p ON p.org_id=s.org_id AND p.project_id=s.project_id AND p.discipline_id=d.value
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid
 AND ($3='' OR s.document_id::text=$3)
 AND s.package_id IS NULL AND s.status IN ('received','needs_review')
 AND p.retired_at IS NULL AND p.lifecycle_status <> 'cancelled'
 AND NOT EXISTS (SELECT 1 FROM packages other WHERE other.org_id=p.org_id AND other.project_id=p.project_id
   AND other.discipline_id=p.discipline_id AND other.id<>p.id AND other.retired_at IS NULL AND other.lifecycle_status <> 'cancelled')
 AND NOT EXISTS (SELECT 1 FROM submission_decisions choice WHERE choice.org_id=s.org_id
   AND choice.submission_id=s.id AND choice.field='package' AND choice.decided_by='user')
ORDER BY s.id FOR UPDATE OF s`, org, project, document)
	if err != nil {
		return err
	}
	type match struct{ id, pkg string }
	var matches []match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.pkg); err != nil {
			rows.Close()
			return err
		}
		matches = append(matches, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range matches {
		if err := upsertSubmissionDecision(ctx, tx, org, project, m.id, proposals.Decision{
			Field: proposals.FieldPackage, Value: m.pkg, Band: proposals.BandGreen, DecidedBy: proposals.ByRule,
		}, false); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE submissions SET package_id=$3::uuid,version=version+1,updated_at=now()
WHERE org_id=$1::uuid AND id=$2::uuid`, org, m.id, m.pkg); err != nil {
			return err
		}
	}
	if len(matches) > 0 {
		payload, _ := json.Marshal(map[string]string{"project_id": project, "state": "matched"})
		_, err = appendEvent(ctx, db.New(tx), org, EventProcurement, "", string(payload))
	}
	return err
}
