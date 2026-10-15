package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/files"
	"sitewise/internal/specifications"
)

func (s *Store) IssueSpecification(ctx context.Context, org, project, actor, title, purpose string, revision int64, blobs *files.Store) (specifications.Issue, error) {
	var issue specifications.Issue
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 200 || !slices.Contains([]string{"preliminary", "coordination", "pricing"}, purpose) {
		return issue, specificationInvalid(errors.New("provide a title and preliminary, coordination or pricing purpose"))
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return issue, err
	}
	defer tx.Rollback(ctx)
	snap, err := readSpecifications(ctx, tx, org, project)
	if err != nil {
		return issue, err
	}
	if snap.Revision != revision {
		return issue, ErrVersionConflict
	}
	if len(snap.Types) == 0 && len(snap.Coverage) == 0 {
		return issue, specificationInvalid(errors.New("add specification items before issuing a schedule"))
	}
	if blobs == nil {
		return issue, specificationInvalid(errors.New("export storage is unavailable"))
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(number),0)+1 FROM specification_issues WHERE org_id=$1 AND project_id=$2`, org, project).Scan(&issue.Number); err != nil {
		return issue, err
	}
	pdf, err := specifications.PDF(snap, title, fmt.Sprintf("%s; Issue %d; specification revision %d", purpose, issue.Number, revision), time.Now().UTC())
	if err != nil {
		return issue, specificationInvalid(err)
	}
	blob, err := blobs.Put(ctx, bytes.NewReader(pdf))
	if err != nil {
		return issue, err
	}
	issue.ExportSHA256 = blob.SHA256
	_, err = tx.Exec(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'application/pdf') ON CONFLICT(org_id,project_id,sha256) DO NOTHING`, org, newID(), project, blob.SHA256, blob.Size)
	if err != nil {
		return issue, err
	}
	issue.ID = newID()
	issue.Title = title
	issue.Purpose = purpose
	issue.CreatedBy = actor
	issue.Snapshot = snap
	raw, _ := json.Marshal(snap)
	err = tx.QueryRow(ctx, `INSERT INTO specification_issues(org_id,project_id,id,number,title,purpose,snapshot,created_by,export_sha256) SELECT $1,$2,$3,COALESCE(max(number),0)+1,$4,$5,$6,$7,$8 FROM specification_issues WHERE org_id=$1 AND project_id=$2 RETURNING number,created_at`, org, project, issue.ID, title, purpose, raw, actor, blob.SHA256).Scan(&issue.Number, &issue.CreatedAt)
	if err != nil {
		return issue, err
	}
	return issue, s.finishSpecification(ctx, tx, org, project, actor, "issue", issue.ID, map[string]any{"number": issue.Number, "title": title, "purpose": purpose, "revision": revision})
}

func (s *Store) ListSpecificationIssues(ctx context.Context, org, project string) ([]specifications.Issue, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,number,title,purpose,created_by::text,created_at FROM specification_issues WHERE org_id=$1 AND project_id=$2 ORDER BY number DESC`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []specifications.Issue{}
	for rows.Next() {
		var i specifications.Issue
		if err = rows.Scan(&i.ID, &i.Number, &i.Title, &i.Purpose, &i.CreatedBy, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) GetSpecificationIssue(ctx context.Context, org, project, id string) (specifications.Issue, error) {
	var i specifications.Issue
	var raw []byte
	if !specificationID(id, false) {
		return i, ErrNotFound
	}
	err := s.pool.QueryRow(ctx, `SELECT id::text,number,title,purpose,created_by::text,created_at,snapshot,export_sha256 FROM specification_issues WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id).Scan(&i.ID, &i.Number, &i.Title, &i.Purpose, &i.CreatedBy, &i.CreatedAt, &raw, &i.ExportSHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, ErrNotFound
	}
	if err != nil {
		return i, err
	}
	err = json.Unmarshal(raw, &i.Snapshot)
	return i, err
}
