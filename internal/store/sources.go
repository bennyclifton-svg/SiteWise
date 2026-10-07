package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/jev"
)

const SourceVersion = "source-3"

type SourcePage struct {
	Page     int    `json:"page"`
	Location string `json:"location"`
	Text     string `json:"text"`
	Context  string `json:"context,omitempty"`
}
type SourceUnit struct {
	Category                   string
	Body                       string
	Page                       int
	Location, Section, Context string
	Start, End                 int
}
type DocumentSource struct {
	Pages      int
	EmptyPages []int
	Source     []SourcePage
	Units      []SourceUnit
}

// ReplaceSource commits complete extraction and its units together. No old
// reading can survive a changed source; user edits are stored independently.
func (s *Store) ReplaceSource(ctx context.Context, orgID, docID string, src DocumentSource) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	projectID, err := lockDocumentProject(ctx, tx, orgID, docID)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(src.Source)
	if err != nil {
		return err
	}
	if src.EmptyPages == nil {
		src.EmptyPages = []int{}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM passages WHERE org_id=$1::uuid AND document_id=$2::uuid`, orgID, docID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM profile_facts WHERE org_id=$1::uuid AND document_id=$2::uuid`, orgID, docID); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for i, u := range src.Units {
		id := newID()
		batch.Queue(`INSERT INTO passages(org_id,id,document_id,ordinal,body) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5)`, orgID, id, docID, i+1, u.Body)
		batch.Queue(`INSERT INTO passage_sources(org_id,passage_id,page,location,section,context,start_offset,end_offset) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8)`, orgID, id, u.Page, u.Location, u.Section, u.Context, u.Start, u.End)
	}
	if batch.Len() > 0 {
		if err = tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_sources(org_id,document_id,version,pages,empty_pages,source) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6)
 ON CONFLICT(org_id,document_id) DO UPDATE SET version=EXCLUDED.version,pages=EXCLUDED.pages,empty_pages=EXCLUDED.empty_pages,source=EXCLUDED.source,created_at=now()`, orgID, docID, SourceVersion, src.Pages, src.EmptyPages, raw)
	if err != nil {
		return err
	}
	if err := BumpRevision(ctx, tx, orgID, projectID, "profile_inputs"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SourceUnits(ctx context.Context, orgID, docID string) (map[string]SourceUnit, error) {
	rows, err := s.pool.Query(ctx, `SELECT p.id::text,ps.page,ps.location,ps.section,ps.context,ps.start_offset,ps.end_offset,ps.category FROM passages p JOIN passage_sources ps ON ps.org_id=p.org_id AND ps.passage_id=p.id WHERE p.org_id=$1::uuid AND p.document_id=$2::uuid`, orgID, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]SourceUnit{}
	for rows.Next() {
		var id string
		var u SourceUnit
		if err = rows.Scan(&id, &u.Page, &u.Location, &u.Section, &u.Context, &u.Start, &u.End, &u.Category); err != nil {
			return nil, err
		}
		out[id] = u
	}
	return out, rows.Err()
}

// CachedPassageCall includes the entire state and questions in its fingerprint.
// A changed question/context cannot accidentally reuse an old judgement.
func (s *Store) CachedPassageCall(ctx context.Context, orgID, id, stage, fingerprint string) (jev.Result, bool, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT result FROM passage_calls WHERE org_id=$1::uuid AND passage_id=$2::uuid AND stage=$3 AND fingerprint=$4`, orgID, id, stage, fingerprint).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return jev.Result{}, false, nil
	}
	if err != nil {
		return jev.Result{}, false, err
	}
	var r jev.Result
	err = json.Unmarshal(raw, &r)
	return r, err == nil, err
}
func (s *Store) SavePassageCall(ctx context.Context, orgID, id, stage, fingerprint string, r jev.Result) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO passage_calls(org_id,passage_id,stage,fingerprint,result) VALUES($1::uuid,$2::uuid,$3,$4,$5) ON CONFLICT(org_id,passage_id,stage) DO UPDATE SET fingerprint=EXCLUDED.fingerprint,result=EXCLUDED.result,created_at=now()`, orgID, id, stage, fingerprint, raw)
	return err
}

type SourceReading struct {
	Category, Provider, Scope, Outcome string
	Confidence                         *float64
	Keys, Unresolved                   []string
}

func (s *Store) SetSourceReading(ctx context.Context, orgID, id string, r SourceReading) error {
	if r.Keys == nil {
		r.Keys = []string{}
	}
	if r.Unresolved == nil {
		r.Unresolved = []string{}
	}
	_, err := s.pool.Exec(ctx, `UPDATE passage_sources SET category=$3,provider=$4,scope=$5,outcome=$6,confidence=$7,mapped_keys=$8,unresolved=$9 WHERE org_id=$1::uuid AND passage_id=$2::uuid`, orgID, id, r.Category, r.Provider, r.Scope, r.Outcome, r.Confidence, r.Keys, r.Unresolved)
	return err
}
