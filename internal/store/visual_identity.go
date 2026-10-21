package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
)

const JobKindVisualIdentity = "visual_identity"
const VisualIdentityVersion = "visual-identity-v4"

// Only small provisional labels enter the register. Images remain lazy-loaded,
// and an intentional user blank is just as authoritative as a filled title.
func (s *Store) visualTitles(ctx context.Context, org string, docs []DocumentView) error {
	ids := []string{}
	byID := map[string]int{}
	for i, doc := range docs {
		blocked := false
		for _, f := range doc.Fields {
			if f.Field == "title" && (f.Value != "" || f.DecidedBy == "user") {
				blocked = true
			}
		}
		if !blocked && doc.Status == "filed" {
			docs[i].VisualStatus = "not_requested"
			ids = append(ids, doc.ID)
			byID[doc.ID] = i
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `SELECT v.document_id, CASE WHEN v.status='queued' AND j.status='failed' THEN 'unavailable' WHEN v.status='queued' AND j.status='leased' AND j.locked_until>now() THEN 'reading' ELSE v.status END, CASE WHEN v.status='ready' AND jsonb_typeof(v.result #> '{suggestions,title,value}')='string' THEN v.result #>> '{suggestions,title,value}' ELSE '' END FROM visual_identity v LEFT JOIN jobs j ON j.org_id=v.org_id AND j.document_id=v.document_id AND j.kind='visual_identity' WHERE v.org_id=$1::uuid AND v.document_id=ANY($2::uuid[])`, org, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, status, title string
		if err := rows.Scan(&id, &status, &title); err != nil {
			return err
		}
		if len(title) <= 180 {
			docs[byID[id]].VisualTitle = title
		}
		docs[byID[id]].VisualStatus = status
	}
	return rows.Err()
}

type VisualReview struct {
	Status    string          `json:"status"`
	Result    json.RawMessage `json:"result"`
	ElapsedMS int             `json:"elapsed_ms"`
}

// QueueVisualIdentity never edits filing decisions. A completed reading is
// cached; repeated requests cannot reset an active lease or spend another call.
func (s *Store) QueueVisualIdentity(ctx context.Context, orgID, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var eligible bool
	err = tx.QueryRow(ctx, `SELECT status='filed' AND lower(filename) LIKE '%.pdf' FROM documents WHERE org_id=$1::uuid AND id=$2::uuid FOR UPDATE`, orgID, id).Scan(&eligible)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !eligible) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var leased bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='visual_identity' AND status='leased' AND locked_until>now())`, orgID, id).Scan(&leased); err != nil {
		return err
	}
	if leased {
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `UPDATE visual_identity SET status='unavailable' WHERE org_id=$1::uuid AND document_id=$2::uuid AND status='queued' AND EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND document_id=$2::uuid AND kind='visual_identity' AND status='failed')`, orgID, id); err != nil {
		return err
	}
	if err = queueVisual(ctx, tx, orgID, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func queueVisual(ctx context.Context, tx pgx.Tx, orgID, id string) error {
	tag, err := tx.Exec(ctx, `INSERT INTO visual_identity(org_id,document_id) SELECT org_id,id FROM documents WHERE org_id=$1::uuid AND id=$2::uuid AND status='filed' AND lower(filename) LIKE '%.pdf' ON CONFLICT (org_id,document_id) DO UPDATE SET status='queued',result='{}' WHERE visual_identity.status='unavailable' OR (visual_identity.status='ready' AND COALESCE(visual_identity.result->>'version','')<>$3)`, orgID, id, VisualIdentityVersion)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status) VALUES($1::uuid,$2::uuid,$3::uuid,'visual_identity','queued') ON CONFLICT (org_id,document_id,kind) DO UPDATE SET status='queued',attempts=0,run_after=now(),locked_until=NULL,lease_token=NULL,last_error='' WHERE jobs.status IN ('done','failed')`, orgID, newID(), id)
	return err
}

func (s *Store) VisualIdentity(ctx context.Context, orgID, id string) (VisualReview, error) {
	var out VisualReview
	err := s.pool.QueryRow(ctx, `SELECT CASE WHEN v.status='queued' AND j.status='failed' THEN 'unavailable' WHEN v.status='queued' AND j.status='leased' AND j.locked_until>now() THEN 'reading' ELSE v.status END,v.result,v.elapsed_ms FROM visual_identity v LEFT JOIN jobs j ON j.org_id=v.org_id AND j.document_id=v.document_id AND j.kind='visual_identity' WHERE v.org_id=$1::uuid AND v.document_id=$2::uuid`, orgID, id).Scan(&out.Status, &out.Result, &out.ElapsedMS)
	if errors.Is(err, pgx.ErrNoRows) {
		return VisualReview{Status: "not_requested", Result: json.RawMessage(`{}`)}, nil
	}
	return out, err
}

// SaveVisualIdentity commits only while the worker still owns its lease.
func (s *Store) SaveVisualIdentity(ctx context.Context, job ClaimedJob, status string, result any, ms int) error {
	if status != "ready" && status != "unavailable" {
		return errors.New("invalid visual status")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if len(raw) > 4<<20 {
		return errors.New("visual result exceeds limit")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE visual_identity SET status=$4,result=$5,elapsed_ms=$6 WHERE org_id=$1::uuid AND document_id=$2::uuid AND EXISTS(SELECT 1 FROM jobs WHERE org_id=$1::uuid AND id=$3::uuid AND lease_token=$7::uuid AND status='leased' AND locked_until>now())`, job.OrgID, job.DocumentID, job.ID, status, raw, ms, job.Token)
	if err == nil && tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := appendEvent(ctx, s.q.WithTx(tx), job.OrgID, "visual", job.DocumentID, `{}`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
