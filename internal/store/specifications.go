package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/db"
	"sitewise/internal/specifications"
)

var ErrInvalidSpecification = errors.New("invalid specification")

func specificationInvalid(err error) error { return fmt.Errorf("%w: %v", ErrInvalidSpecification, err) }
func specificationID(id string, optional bool) bool {
	if optional && id == "" {
		return true
	}
	var u pgtype.UUID
	return u.Scan(id) == nil && u.Valid
}

func (s *Store) specificationTx(ctx context.Context, org, project, actor string) (pgx.Tx, error) {
	if !specificationID(project, false) {
		return nil, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if err = lockProject(ctx, tx, org, project); err == nil {
		err = packageActor(ctx, tx, org, actor)
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func (s *Store) finishSpecification(ctx context.Context, tx pgx.Tx, org, project, actor, kind, id string, content any) error {
	raw, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO specification_history(org_id,project_id,entity_id,entity_kind,actor,content) VALUES($1,$2,$3,$4,$5,$6)`, org, project, id, kind, actor, raw)
	if err != nil {
		return err
	}
	if err = bumpRevisionLocked(ctx, tx, org, project, "specifications"); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"project_id": project})
	if _, err = appendEvent(ctx, s.q.WithTx(tx), org, "specification", "", string(payload)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ReadSpecifications(ctx context.Context, org, project string) (specifications.Snapshot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return specifications.Snapshot{}, err
	}
	defer tx.Rollback(ctx)
	out, err := readSpecifications(ctx, tx, org, project)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func readSpecifications(ctx context.Context, q db.DBTX, org, project string) (specifications.Snapshot, error) {
	out := specifications.Snapshot{ProjectID: project, Types: []specifications.Type{}, Applications: []specifications.Application{}, Assertions: []specifications.Assertion{}, Relationships: []specifications.Relationship{}}
	err := q.QueryRow(ctx, `SELECT p.name,COALESCE(r.version,0) FROM projects p LEFT JOIN project_revisions r ON r.org_id=p.org_id AND r.project_id=p.id WHERE p.org_id=$1 AND p.id=$2`, org, project).Scan(&out.ProjectName, &out.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if err = readSpecificationContent(ctx, q, org, project, "specification_types", &out.Types); err != nil {
		return out, err
	}
	if err = readSpecificationContent(ctx, q, org, project, "specification_applications", &out.Applications); err != nil {
		return out, err
	}
	if err = readSpecificationContent(ctx, q, org, project, "specification_relationships", &out.Relationships); err != nil {
		return out, err
	}
	if err = readSpecificationWorkspace(ctx, q, org, project, &out); err != nil {
		return out, err
	}
	rows, err := q.Query(ctx, `SELECT a.id::text,a.type_id::text,COALESCE(a.application_id::text,''),a.attribute_key,a.meaning,a.value,a.source,a.origin,a.status,a.actor::text,a.reason,a.version,a.created_at,
 CASE WHEN a.source->>'document_id' IS NULL OR a.source->>'document_id'='' THEN false ELSE NOT EXISTS(SELECT 1 FROM documents d WHERE d.org_id=a.org_id AND d.project_id=a.project_id AND d.id::text=a.source->>'document_id') END
 FROM specification_assertions a WHERE a.org_id=$1 AND a.project_id=$2 ORDER BY a.created_at,a.id`, org, project)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var a specifications.Assertion
		var value, source []byte
		var unavailable bool
		if err = rows.Scan(&a.ID, &a.TypeID, &a.ApplicationID, &a.Key, &a.Meaning, &value, &source, &a.Origin, &a.Status, &a.Actor, &a.Reason, &a.Version, &a.CreatedAt, &unavailable); err != nil {
			return out, err
		}
		if err = json.Unmarshal(value, &a.Value); err != nil {
			return out, err
		}
		if err = json.Unmarshal(source, &a.Source); err != nil {
			return out, err
		}
		a.Source.Unavailable = unavailable
		out.Assertions = append(out.Assertions, a)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows.Close()
	docs, err := specificationDocuments(ctx, q, org, project)
	if err != nil {
		return out, err
	}
	for i := range out.Assertions {
		a := &out.Assertions[i]
		if a.Source.DocumentID != "" {
			d, ok := docs[a.Source.DocumentID]
			a.Source.Unavailable = !ok
			a.Source.Stale = ok && (d.Stale || (a.Source.Hash != "" && a.Source.Hash != d.Hash))
		}
	}
	return out, nil
}

// Only fixed table names from this file are allowed; client values never form SQL.
func readSpecificationContent[T any](ctx context.Context, q db.DBTX, org, project, table string, out *[]T) error {
	switch table {
	case "specification_types", "specification_applications", "specification_relationships", "specification_coverage":
	default:
		return ErrInvalidSpecification
	}
	rows, err := q.Query(ctx, `SELECT content FROM `+table+` WHERE org_id=$1 AND project_id=$2 ORDER BY id`, org, project)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var value T
		if err = rows.Scan(&raw); err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &value); err != nil {
			return err
		}
		*out = append(*out, value)
	}
	return rows.Err()
}

func (s *Store) SaveSpecificationType(ctx context.Context, org, project, actor string, t specifications.Type) (specifications.Type, error) {
	t.Title = strings.TrimSpace(t.Title)
	t.Code = strings.TrimSpace(t.Code)
	if t.Status == "" {
		t.Status = "active"
	}
	if err := specifications.ValidateType(t); err != nil {
		return t, specificationInvalid(err)
	}
	if t.SystemID != "" {
		if s.workCatalog() == nil {
			return t, specificationInvalid(errors.New("system catalogue unavailable"))
		}
		sys, ok := s.workCatalog().System(t.SystemID)
		if !ok || sys.Status == "deprecated" {
			return t, specificationInvalid(errors.New("unknown or deprecated system"))
		}
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return t, err
	}
	defer tx.Rollback(ctx)
	create := t.ID == ""
	if create {
		t.Origin = "user"
		t.Authority = "sitewise"
		t.SourceDocumentID = ""
		t.SuggestedTypeID = ""
		if t.StarterKey != "" {
			starter, ok := specifications.StarterFor(t.StarterKey)
			if !(ok && starter.SystemID == t.SystemID) && t.StarterKey != "system:"+t.SystemID {
				return t, specificationInvalid(errors.New("unknown starter prompt"))
			}
			var existing []byte
			e := tx.QueryRow(ctx, `SELECT content FROM specification_types WHERE org_id=$1 AND project_id=$2 AND content->>'starter_key'=$3 AND content->>'status'='active' ORDER BY id LIMIT 1`, org, project, t.StarterKey).Scan(&existing)
			if e == nil {
				var old specifications.Type
				if e = json.Unmarshal(existing, &old); e != nil {
					return t, e
				}
				return old, tx.Commit(ctx)
			}
			if !errors.Is(e, pgx.ErrNoRows) {
				return t, e
			}
		}
	} else {
		if !specificationID(t.ID, false) {
			return t, ErrNotFound
		}
		var raw []byte
		if err = tx.QueryRow(ctx, `SELECT content FROM specification_types WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, t.ID).Scan(&raw); errors.Is(err, pgx.ErrNoRows) {
			return t, ErrNotFound
		} else if err != nil {
			return t, err
		}
		var old specifications.Type
		if err = json.Unmarshal(raw, &old); err != nil {
			return t, err
		}
		external, e := specificationExternal(ctx, tx, org, project, t.ID)
		if e != nil {
			return t, e
		}
		if external && t.Description != old.Description {
			return t, specificationInvalid(errors.New("external specification content is read-only; propose a change or explicitly adopt maintenance"))
		}
		t.Authority = old.Authority
		t.SourceDocumentID = old.SourceDocumentID
		t.StarterKey = old.StarterKey
		t.SuggestedTypeID = old.SuggestedTypeID
		t.Origin = old.Origin
	}
	if create {
		t.ID = newID()
		t.Version = 1
	} else {
		if !specificationID(t.ID, false) {
			return t, ErrNotFound
		}
		t.Version++
	}
	t.ProjectID = project
	if t.Origin == "" {
		t.Origin = "user"
	}
	t.ReviewStatus = "accepted"
	raw, _ := json.Marshal(t)
	if create {
		_, err = tx.Exec(ctx, `INSERT INTO specification_types(org_id,project_id,id,content,created_by) VALUES($1,$2,$3,$4,$5)`, org, project, t.ID, raw, actor)
	} else {
		var n int64
		tag, e := tx.Exec(ctx, `UPDATE specification_types SET content=$4,version=version+1,updated_at=now() WHERE org_id=$1 AND project_id=$2 AND id=$3 AND version=$5`, org, project, t.ID, raw, t.Version-1)
		err = e
		n = tag.RowsAffected()
		if err == nil && n == 0 {
			err = specificationEditMiss(ctx, tx, org, project, "specification_types", t.ID)
		}
	}
	if err != nil {
		return t, err
	}
	return t, s.finishSpecification(ctx, tx, org, project, actor, "type", t.ID, t)
}

func specificationEditMiss(ctx context.Context, q db.DBTX, org, project, table, id string) error {
	switch table {
	case "specification_types", "specification_applications", "specification_assertions":
	default:
		return ErrInvalidSpecification
	}
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE org_id=$1 AND project_id=$2 AND id=$3)`, org, project, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrVersionConflict
}

func specificationTypeExists(ctx context.Context, q db.DBTX, org, project, id string) error {
	if !specificationID(id, false) {
		return ErrNotFound
	}
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM specification_types WHERE org_id=$1 AND project_id=$2 AND id=$3 AND content->>'status'='active')`, org, project, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SaveSpecificationApplication(ctx context.Context, org, project, actor string, a specifications.Application) (specifications.Application, error) {
	if err := specifications.ValidateApplication(a); err != nil {
		return a, specificationInvalid(err)
	}
	if !specificationID(a.PartID, true) || !specificationID(a.WorkItemID, true) {
		return a, ErrNotFound
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	if err = specificationTypeExists(ctx, tx, org, project, a.TypeID); err != nil {
		return a, err
	}
	if a.PartID != "" {
		if _, err = userValueOwner(ctx, tx, org, project, a.PartID); err != nil {
			return a, err
		}
	}
	if a.WorkItemID != "" {
		var part string
		err = tx.QueryRow(ctx, `SELECT w.part_id::text,w.action,w.inclusion,p.label FROM work_items w JOIN project_parts p ON p.org_id=w.org_id AND p.id=w.part_id WHERE w.org_id=$1 AND w.project_id=$2 AND w.id=$3 AND w.retired_at IS NULL`, org, project, a.WorkItemID).Scan(&part, &a.Action, &a.Inclusion, &a.Location)
		if errors.Is(err, pgx.ErrNoRows) {
			return a, ErrNotFound
		}
		if err != nil {
			return a, err
		}
		if a.PartID != "" && part != a.PartID {
			return a, specificationInvalid(errors.New("work and application parts differ"))
		}
		a.PartID = part
	}
	create := a.ID == ""
	if create {
		a.ID = newID()
		a.Version = 1
	} else {
		if !specificationID(a.ID, false) {
			return a, ErrNotFound
		}
		a.Version++
	}
	a.ProjectID = project
	raw, _ := json.Marshal(a)
	if create {
		_, err = tx.Exec(ctx, `INSERT INTO specification_applications(org_id,project_id,id,type_id,part_id,work_item_id,content) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,$7)`, org, project, a.ID, a.TypeID, a.PartID, a.WorkItemID, raw)
	} else {
		tag, e := tx.Exec(ctx, `UPDATE specification_applications SET content=$5,part_id=NULLIF($6,'')::uuid,work_item_id=NULLIF($7,'')::uuid,version=version+1 WHERE org_id=$1 AND project_id=$2 AND id=$3 AND type_id=$4 AND version=$8`, org, project, a.ID, a.TypeID, raw, a.PartID, a.WorkItemID, a.Version-1)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			err = specificationEditMiss(ctx, tx, org, project, "specification_applications", a.ID)
		}
	}
	if err != nil {
		return a, err
	}
	return a, s.finishSpecification(ctx, tx, org, project, actor, "application", a.ID, a)
}
