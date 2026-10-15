package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/db"
	"sitewise/internal/specifications"
)

func readSpecificationWorkspace(ctx context.Context, q db.DBTX, org, project string, out *specifications.Snapshot) error {
	out.Coverage = []specifications.Coverage{}
	out.Scope = []specifications.Scope{}
	if err := readSpecificationContent(ctx, q, org, project, "specification_coverage", &out.Coverage); err != nil {
		return err
	}
	docs, err := specificationDocuments(ctx, q, org, project)
	if err != nil {
		return err
	}
	for i := range out.Coverage {
		for n := range out.Coverage[i].References {
			r := &out.Coverage[i].References[n]
			d, ok := docs[r.DocumentID]
			r.Unavailable = !ok
			r.Stale = ok && (d.Stale || r.Hash != d.Hash || r.Revision != d.Revision || r.Number != d.Number)
		}
	}
	rows, err := q.Query(ctx, `SELECT w.id::text,w.system_id,w.part_id::text,p.label,w.title,w.action,w.inclusion,COALESCE(pr.value,''),w.retired_at IS NOT NULL,w.version
 FROM work_items w JOIN project_parts p ON p.org_id=w.org_id AND p.id=w.part_id
 LEFT JOIN profile_rows pr ON pr.org_id=w.org_id AND pr.project_id=w.project_id AND pr.part_id=w.part_id AND pr.scope='project' AND pr.key='sys.'||w.system_id||'.provider'
 WHERE w.org_id=$1 AND w.project_id=$2 ORDER BY w.system_id,p.label,w.id`, org, project)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s specifications.Scope
		if err = rows.Scan(&s.WorkItemID, &s.SystemID, &s.PartID, &s.Location, &s.Title, &s.Action, &s.Inclusion, &s.Provider, &s.Retired, &s.Version); err != nil {
			return err
		}
		out.Scope = append(out.Scope, s)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	rows.Close()
	for i := range out.Applications {
		a := &out.Applications[i]
		if a.WorkItemID == "" {
			continue
		}
		for _, s := range out.Scope {
			if s.WorkItemID == a.WorkItemID {
				a.Action = s.Action
				a.Inclusion = s.Inclusion
				a.Location = s.Location
				if s.Retired {
					a.Inclusion = "excluded"
				}
			}
		}
	}
	return nil
}

func specificationDocuments(ctx context.Context, q db.DBTX, org, project string) (map[string]specifications.DocumentReference, error) {
	rows, err := q.Query(ctx, `SELECT d.id::text,d.filename,COALESCE(d.document_number,''),COALESCE(d.revision,''),encode(f.sha256,'hex'),EXISTS(SELECT 1 FROM supersessions s WHERE s.org_id=d.org_id AND s.prior_document_id=d.id)
 FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE d.org_id=$1 AND d.project_id=$2`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]specifications.DocumentReference{}
	for rows.Next() {
		var r specifications.DocumentReference
		if err = rows.Scan(&r.DocumentID, &r.Filename, &r.Number, &r.Revision, &r.Hash, &r.Stale); err != nil {
			return nil, err
		}
		out[r.DocumentID] = r
	}
	return out, rows.Err()
}

func (s *Store) SaveSpecificationCoverage(ctx context.Context, org, project, actor string, c specifications.Coverage) (specifications.Coverage, error) {
	if err := specifications.ValidateCoverage(c); err != nil {
		return c, specificationInvalid(err)
	}
	if !specificationID(c.TypeID, true) || !specificationID(c.ID, true) {
		return c, ErrNotFound
	}
	if cat := s.workCatalog(); cat != nil {
		if _, ok := cat.System(c.SystemID); !ok {
			return c, specificationInvalid(errors.New("unknown building system"))
		}
	} else {
		return c, specificationInvalid(errors.New("system catalogue unavailable"))
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	if c.TypeID != "" {
		if err = specificationTypeExists(ctx, tx, org, project, c.TypeID); err != nil {
			return c, err
		}
	}
	docs, err := specificationDocuments(ctx, tx, org, project)
	if err != nil {
		return c, err
	}
	seen := map[string]bool{}
	for i, r := range c.References {
		d, ok := docs[r.DocumentID]
		if !ok {
			return c, ErrNotFound
		}
		if seen[r.DocumentID] {
			return c, specificationInvalid(errors.New("duplicate document reference"))
		}
		seen[r.DocumentID] = true
		d.Location = r.Location
		c.References[i] = d
	}
	var previous []byte
	err = tx.QueryRow(ctx, `SELECT content FROM specification_coverage WHERE org_id=$1 AND project_id=$2 AND ((NULLIF($4,'')::uuid IS NOT NULL AND type_id=NULLIF($4,'')::uuid) OR ($4='' AND type_id IS NULL AND system_id=$3))`, org, project, c.SystemID, c.TypeID).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return c, err
	}
	create := errors.Is(err, pgx.ErrNoRows)
	if create {
		if c.ID != "" || c.Version != 0 {
			return c, ErrVersionConflict
		}
		c.ID = newID()
		c.Version = 1
	} else {
		var old specifications.Coverage
		if err = json.Unmarshal(previous, &old); err != nil {
			return c, err
		}
		if c.ID != old.ID || c.Version != old.Version {
			return c, ErrVersionConflict
		}
		if old.Mode != c.Mode && strings.TrimSpace(c.Reason) == "" {
			return c, specificationInvalid(errors.New("explain the change in specification responsibility"))
		}
		c.Version++
	}
	raw, _ := json.Marshal(c)
	if create {
		_, err = tx.Exec(ctx, `INSERT INTO specification_coverage(org_id,project_id,id,system_id,type_id,content,updated_by) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7)`, org, project, c.ID, c.SystemID, c.TypeID, raw, actor)
	} else {
		_, err = tx.Exec(ctx, `UPDATE specification_coverage SET content=$4,version=version+1,updated_by=$5,system_id=$6 WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, c.ID, raw, actor, c.SystemID)
	}
	if err != nil {
		return c, err
	}
	return c, s.finishSpecification(ctx, tx, org, project, actor, "coverage", c.ID, c)
}

// Adoption explicitly transfers maintenance; confirming a reading never does.
func (s *Store) AdoptSpecification(ctx context.Context, org, project, actor, id string, version int64, reason string) (specifications.Type, error) {
	var t specifications.Type
	if !specificationID(id, false) {
		return t, ErrNotFound
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return t, specificationInvalid(errors.New("record why maintenance is being transferred to SiteWise"))
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return t, err
	}
	defer tx.Rollback(ctx)
	snap, err := readSpecifications(ctx, tx, org, project)
	if err != nil {
		return t, err
	}
	found := false
	for _, v := range snap.Types {
		if v.ID == id {
			t = v
			found = true
		}
	}
	if !found {
		return t, ErrNotFound
	}
	if t.Version != version {
		return t, ErrVersionConflict
	}
	if c := specifications.EffectiveCoverage(snap, t); c != nil && c.Mode == "external" {
		return t, specificationInvalid(errors.New("update the item's coverage responsibility before adopting it"))
	}
	t.Authority = "sitewise"
	t.Version++
	raw, _ := json.Marshal(t)
	_, err = tx.Exec(ctx, `UPDATE specification_types SET content=$4,version=version+1,updated_at=now() WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id, raw)
	if err != nil {
		return t, err
	}
	return t, s.finishSpecification(ctx, tx, org, project, actor, "adoption", id, map[string]any{"item": t, "reason": reason})
}

func specificationExternal(ctx context.Context, q db.DBTX, org, project, id string) (bool, error) {
	var raw []byte
	if err := q.QueryRow(ctx, `SELECT content FROM specification_types WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id).Scan(&raw); err != nil {
		return false, err
	}
	var t specifications.Type
	if err := json.Unmarshal(raw, &t); err != nil {
		return false, err
	}
	snap := specifications.Snapshot{}
	if err := readSpecificationContent(ctx, q, org, project, "specification_coverage", &snap.Coverage); err != nil {
		return false, err
	}
	return specifications.External(snap, t), nil
}

// Only new/failed/outdated documents are queued. Unchanged completed readings
// and live workers are left alone. This does not establish document authority.
func (s *Store) RequestSpecificationUpdate(ctx context.Context, org, project, actor string) (int64, error) {
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT d.id::text FROM documents d LEFT JOIN specification_imports i ON i.org_id=d.org_id AND i.document_id=d.id LEFT JOIN jobs j ON j.org_id=d.org_id AND j.document_id=d.id AND j.kind='specification'
 WHERE d.org_id=$1 AND d.project_id=$2 AND d.status<>'pending'
 AND NOT EXISTS(SELECT 1 FROM supersessions x WHERE x.org_id=d.org_id AND x.prior_document_id=d.id)
 AND (j.id IS NULL OR j.status='failed' OR (i.version<>$3 AND j.status NOT IN ('queued','leased'))) ORDER BY d.id`, org, project, specifications.ImportVersion)
	if err != nil {
		return 0, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `INSERT INTO specification_imports(org_id,project_id,document_id,requested_by,version) SELECT $1,$2,id,$3,$4 FROM unnest($5::uuid[]) id ON CONFLICT(org_id,document_id) DO UPDATE SET requested_by=EXCLUDED.requested_by,version=EXCLUDED.version,updated_at=now()`, org, project, actor, specifications.ImportVersion, ids)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO jobs(org_id,id,document_id,kind,status,priority) SELECT $1,gen_random_uuid(),id,'specification','queued',0 FROM unnest($2::uuid[]) id ON CONFLICT(org_id,document_id,kind) DO UPDATE SET status='queued',attempts=0,run_after=now(),last_error='',lease_token=NULL,locked_until=NULL`, org, ids)
	if err != nil {
		return 0, err
	}
	return int64(len(ids)), s.finishSpecification(ctx, tx, org, project, actor, "update_request", project, map[string]any{"documents": ids})
}
