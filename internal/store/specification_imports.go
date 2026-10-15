package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/jev"
	"sitewise/internal/specifications"
)

const JobKindSpecification = "specification"

type SpecificationImport struct {
	DocumentID   string `json:"document_id"`
	ProjectID    string `json:"project_id"`
	Actor        string `json:"actor"`
	TargetTypeID string `json:"target_type_id"`
	Status       string `json:"status"`
	Count        int    `json:"count"`
	Error        string `json:"error"`
	Hash         string `json:"hash"`
}

func (s *Store) RequestSpecificationImport(ctx context.Context, org, project, actor, doc, target string) error {
	if !specificationID(doc, false) || !specificationID(target, true) {
		return ErrNotFound
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE org_id=$1 AND project_id=$2 AND id=$3)`, org, project, doc).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if target != "" {
		if err = specificationTypeExists(ctx, tx, org, project, target); err != nil {
			return err
		}
	}
	var leased bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE org_id=$1 AND document_id=$2 AND kind='specification' AND status='leased')`, org, doc).Scan(&leased); err != nil {
		return err
	}
	if leased {
		return ErrVersionConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO specification_imports(org_id,project_id,document_id,requested_by,target_type_id,version) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6) ON CONFLICT(org_id,document_id) DO UPDATE SET requested_by=EXCLUDED.requested_by,target_type_id=EXCLUDED.target_type_id,version=EXCLUDED.version,updated_at=now()`, org, project, doc, actor, target, specifications.ImportVersion)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status,priority) VALUES($1,$2,$3,'specification','queued',0) ON CONFLICT(org_id,document_id,kind) DO UPDATE SET status='queued',attempts=0,run_after=now(),created_at=now(),last_error='',lease_token=NULL,locked_until=NULL`, org, newID(), doc)
	if err != nil {
		return err
	}
	return s.finishSpecification(ctx, tx, org, project, actor, "import_request", doc, map[string]string{"document_id": doc, "target_type_id": target})
}

func (s *Store) SpecificationImport(ctx context.Context, org, doc string) (SpecificationImport, error) {
	var i SpecificationImport
	err := s.pool.QueryRow(ctx, `SELECT i.document_id::text,i.project_id::text,i.requested_by::text,COALESCE(i.target_type_id::text,''),j.status,i.imported_count,j.last_error,encode(f.sha256,'hex') FROM specification_imports i JOIN jobs j ON j.org_id=i.org_id AND j.document_id=i.document_id AND j.kind='specification' JOIN documents d ON d.org_id=i.org_id AND d.id=i.document_id JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE i.org_id=$1 AND i.document_id=$2`, org, doc).Scan(&i.DocumentID, &i.ProjectID, &i.Actor, &i.TargetTypeID, &i.Status, &i.Count, &i.Error, &i.Hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return i, ErrNotFound
	}
	return i, err
}

func (s *Store) ListSpecificationImports(ctx context.Context, org, project string) ([]SpecificationImport, error) {
	if _, err := s.GetProject(ctx, org, project); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT i.document_id::text,i.project_id::text,i.requested_by::text,COALESCE(i.target_type_id::text,''),j.status,i.imported_count,j.last_error FROM specification_imports i JOIN jobs j ON j.org_id=i.org_id AND j.document_id=i.document_id AND j.kind='specification' WHERE i.org_id=$1 AND i.project_id=$2 ORDER BY i.updated_at DESC`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SpecificationImport{}
	for rows.Next() {
		var i SpecificationImport
		if err = rows.Scan(&i.DocumentID, &i.ProjectID, &i.Actor, &i.TargetTypeID, &i.Status, &i.Count, &i.Error); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) SpecificationSource(ctx context.Context, org, doc string) ([]SourcePage, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT source FROM document_sources WHERE org_id=$1 AND document_id=$2`, org, doc).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var pages []SourcePage
	err = json.Unmarshal(raw, &pages)
	return pages, err
}

func (s *Store) CachedSpecificationReading(ctx context.Context, org, doc, key, fingerprint string) (jev.Result, bool, error) {
	var result jev.Result
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT result FROM specification_import_readings WHERE org_id=$1 AND document_id=$2 AND candidate_key=$3 AND fingerprint=$4`, org, doc, key, fingerprint).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, false, nil
	}
	if err != nil {
		return result, false, err
	}
	err = json.Unmarshal(raw, &result)
	return result, err == nil, err
}

func (s *Store) SaveSpecificationReading(ctx context.Context, job ClaimedJob, key, fingerprint string, result jev.Result) error {
	raw, _ := json.Marshal(result)
	tag, err := s.pool.Exec(ctx, `INSERT INTO specification_import_readings(org_id,document_id,candidate_key,fingerprint,result) SELECT $1,$2,$3,$4,$5 WHERE EXISTS(SELECT 1 FROM jobs WHERE org_id=$1 AND id=$6 AND lease_token=$7 AND status='leased' AND locked_until>now()) ON CONFLICT(org_id,document_id,candidate_key) DO UPDATE SET fingerprint=EXCLUDED.fingerprint,result=EXCLUDED.result`, job.OrgID, job.DocumentID, key, fingerprint, raw, job.ID, job.Token)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrLeaseLost
	}
	return err
}

func (s *Store) SaveSpecificationCandidate(ctx context.Context, job ClaimedJob, i SpecificationImport, c specifications.Candidate, result jev.Result) error {
	if specifications.CandidateTooLong(c) {
		return specificationInvalid(errors.New("source item exceeds the review size limit; split its source information"))
	}
	tx, err := s.specificationTx(ctx, job.OrgID, i.ProjectID, i.Actor)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var live bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE org_id=$1 AND id=$2 AND lease_token=$3 AND status='leased' AND locked_until>now())`, job.OrgID, job.ID, job.Token).Scan(&live); err != nil {
		return err
	}
	if !live {
		return ErrLeaseLost
	}
	key := i.Hash + "/" + i.DocumentID + "/" + c.Key
	typeID := i.TargetTypeID
	if typeID == "" {
		typeID = specifications.StableID(key)
		kind := result.Answers["kind"].Choice
		if kind == "" || kind == "none" {
			kind = "unspecified"
		}
		t := specifications.Type{ID: typeID, ProjectID: i.ProjectID, Code: c.Code, Title: c.Title, Kind: kind, Status: "active", Version: 1, Origin: "document", ReviewStatus: "proposed"}
		t.Authority = "external"
		t.SourceDocumentID = i.DocumentID
		if cat := s.workCatalog(); cat != nil {
			if system, ok := cat.System(result.Answers["system"].Choice); ok {
				t.SystemID = system.ID
			}
		}
		match := result.Answers["match"].Choice
		if strings.HasPrefix(match, "starter:") {
			if starter, ok := specifications.StarterFor(strings.TrimPrefix(match, "starter:")); ok {
				t.StarterKey = starter.Key
				t.SystemID = starter.SystemID
			}
		} else if specificationID(match, false) {
			var exists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM specification_types WHERE org_id=$1 AND project_id=$2 AND id=$3 AND content->>'status'='active')`, job.OrgID, i.ProjectID, match).Scan(&exists); err != nil {
				return err
			}
			if exists {
				t.SuggestedTypeID = match
			}
		}
		if err = specifications.ValidateType(t); err != nil {
			return specificationInvalid(err)
		}
		raw, _ := json.Marshal(t)
		_, err = tx.Exec(ctx, `INSERT INTO specification_types(org_id,project_id,id,content,created_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(org_id,id) DO NOTHING`, job.OrgID, i.ProjectID, typeID, raw, i.Actor)
		if err != nil {
			return err
		}
	} else {
		if err = specificationTypeExists(ctx, tx, job.OrgID, i.ProjectID, typeID); err != nil {
			return err
		}
	}
	meaning := result.Answers["meaning"].Choice
	// Mixed prose remains a source note. It must not silently become a design
	// selection; the reviewer can create specific assertions after reading it.
	if meaning == "" || meaning == "none" {
		meaning = "unclassified"
	}
	fields := append([]specifications.CandidateField{{Key: "notes", Text: c.Text}}, c.Fields...)
	inserted := int64(0)
	for n, f := range fields {
		source := specifications.Source{DocumentID: i.DocumentID, Hash: i.Hash, Page: c.Page, Location: c.Location, Excerpt: c.Text, Marker: "unreviewed"}
		a := specifications.Assertion{TypeID: typeID, Key: f.Key, Meaning: meaning, Value: specifications.Value{State: "known", Text: f.Text}, Source: source}
		if err = specifications.ValidateAssertion(a); err != nil {
			return specificationInvalid(err)
		}
		value, _ := json.Marshal(a.Value)
		src, _ := json.Marshal(source)
		// Extraction identity stays stable when classification questions change.
		importKey := fmt.Sprintf("specification-import-1/%s/%s/%d", key, typeID, n)
		tag, e := tx.Exec(ctx, `INSERT INTO specification_assertions(org_id,project_id,id,type_id,attribute_key,meaning,value,source,origin,status,actor,import_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'document','proposed',$9,$10) ON CONFLICT(org_id,project_id,import_key) WHERE import_key IS NOT NULL DO NOTHING`, job.OrgID, i.ProjectID, newID(), typeID, a.Key, a.Meaning, value, src, i.Actor, importKey)
		if e != nil {
			return e
		}
		inserted += tag.RowsAffected()
	}
	if inserted == 0 {
		return tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `UPDATE specification_imports SET imported_count=imported_count+1,updated_at=now() WHERE org_id=$1 AND document_id=$2`, job.OrgID, i.DocumentID)
	if err != nil {
		return err
	}
	return s.finishSpecification(ctx, tx, job.OrgID, i.ProjectID, i.Actor, "import_candidate", typeID, map[string]string{"document_id": i.DocumentID, "candidate": c.Key})
}
