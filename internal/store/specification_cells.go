package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/specifications"
)

type SpecificationCell struct {
	TypeID             string               `json:"type_id"`
	Key                string               `json:"key"`
	Value              specifications.Value `json:"value"`
	Meaning            string               `json:"meaning"`
	ExpectedAcceptedID string               `json:"expected_accepted_id"`
}

// A direct edit is one atomic user decision. External values must use the
// proposal/review path; this endpoint never disguises source text as user input.
func (s *Store) SaveSpecificationCell(ctx context.Context, org, project, actor string, in SpecificationCell) (specifications.Assertion, error) {
	a := specifications.Assertion{TypeID: in.TypeID, Key: in.Key, Meaning: in.Meaning, Value: in.Value}
	if err := specifications.ValidateAssertion(a); err != nil {
		return a, specificationInvalid(err)
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	if err = specificationTypeExists(ctx, tx, org, project, a.TypeID); err != nil {
		return a, err
	}
	external, err := specificationExternal(ctx, tx, org, project, a.TypeID)
	if err != nil {
		return a, err
	}
	if external {
		return a, specificationInvalid(errors.New("maintained externally; propose a change or adopt maintenance first"))
	}
	var prior, origin string
	err = tx.QueryRow(ctx, `SELECT id::text,origin FROM specification_assertions WHERE org_id=$1 AND project_id=$2 AND type_id=$3 AND application_id IS NULL AND attribute_key=$4 AND meaning=$5 AND status='accepted'`, org, project, a.TypeID, a.Key, a.Meaning).Scan(&prior, &origin)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return a, err
	}
	if prior != in.ExpectedAcceptedID {
		return a, ErrVersionConflict
	}
	if origin == "document" {
		return a, specificationInvalid(errors.New("this cell is a confirmed source reading; propose a separate project decision in Details"))
	}
	if prior != "" {
		_, err = tx.Exec(ctx, `UPDATE specification_assertions SET status='superseded',version=version+1 WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, prior)
		if err != nil {
			return a, err
		}
	}
	a.ID = newID()
	a.Actor = actor
	a.Origin = "user"
	a.Status = "accepted"
	a.Version = 1
	a.Reason = "Edited in the specification table"
	value, _ := json.Marshal(a.Value)
	err = tx.QueryRow(ctx, `INSERT INTO specification_assertions(org_id,project_id,id,type_id,attribute_key,meaning,value,source,origin,status,actor,reason) VALUES($1,$2,$3,$4,$5,$6,$7,'{}','user','accepted',$8,$9) RETURNING created_at`, org, project, a.ID, a.TypeID, a.Key, a.Meaning, value, actor, a.Reason).Scan(&a.CreatedAt)
	if err != nil {
		return a, err
	}
	return a, s.finishSpecification(ctx, tx, org, project, actor, "cell", a.ID, map[string]any{"assertion": a, "supersedes": prior})
}

func (s *Store) LinkSpecificationCandidate(ctx context.Context, org, project, actor, id, target string, version int64) error {
	if id == target || !specificationID(id, false) || !specificationID(target, false) {
		return specificationInvalid(errors.New("choose another item"))
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = specificationTypeExists(ctx, tx, org, project, target); err != nil {
		return err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT content FROM specification_types WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id).Scan(&raw); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	var t specifications.Type
	if err = json.Unmarshal(raw, &t); err != nil {
		return err
	}
	if t.Version != version || t.ReviewStatus != "proposed" || t.Status != "active" {
		return ErrVersionConflict
	}
	var used bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM specification_assertions WHERE org_id=$1 AND project_id=$2 AND type_id=$3 AND status<>'proposed') OR EXISTS(SELECT 1 FROM specification_applications WHERE org_id=$1 AND project_id=$2 AND type_id=$3) OR EXISTS(SELECT 1 FROM specification_relationships WHERE org_id=$1 AND project_id=$2 AND (from_id=$3 OR to_id=$3)) OR EXISTS(SELECT 1 FROM specification_coverage WHERE org_id=$1 AND project_id=$2 AND type_id=$3)`, org, project, id).Scan(&used); err != nil {
		return err
	}
	if used {
		return specificationInvalid(errors.New("reviewed or assigned items cannot be merged; retain their references separately"))
	}
	if _, err = tx.Exec(ctx, `UPDATE specification_assertions SET type_id=$4,version=version+1 WHERE org_id=$1 AND project_id=$2 AND type_id=$3`, org, project, id, target); err != nil {
		return err
	}
	t.Status = "retired"
	t.Version++
	raw, _ = json.Marshal(t)
	if _, err = tx.Exec(ctx, `UPDATE specification_types SET content=$4,version=version+1 WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id, raw); err != nil {
		return err
	}
	return s.finishSpecification(ctx, tx, org, project, actor, "candidate_mapping", id, map[string]string{"target_type_id": target})
}
