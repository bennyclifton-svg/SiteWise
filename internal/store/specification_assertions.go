package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/specifications"
)

func (s *Store) AddSpecificationAssertion(ctx context.Context, org, project, actor string, a specifications.Assertion) (specifications.Assertion, error) {
	if err := specifications.ValidateAssertion(a); err != nil {
		return a, specificationInvalid(err)
	}
	if !specificationID(a.ApplicationID, true) || !specificationID(a.Source.DocumentID, true) {
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
	if a.ApplicationID != "" {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM specification_applications WHERE org_id=$1 AND project_id=$2 AND type_id=$3 AND id=$4)`, org, project, a.TypeID, a.ApplicationID).Scan(&exists); err != nil {
			return a, err
		}
		if !exists {
			return a, ErrNotFound
		}
	}
	a.ID = newID()
	a.Actor = actor
	a.Status = "proposed"
	a.Version = 1
	a.Origin = "user"
	a.Source.Unavailable = false
	if a.Source.DocumentID != "" {
		err = tx.QueryRow(ctx, `SELECT encode(f.sha256,'hex') FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id WHERE d.org_id=$1 AND d.project_id=$2 AND d.id=$3`, org, project, a.Source.DocumentID).Scan(&a.Source.Hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return a, ErrNotFound
		}
		if err != nil {
			return a, err
		}
		a.Origin = "document"
		if a.Source.Marker == "" {
			a.Source.Marker = "unreviewed"
		}
	} else {
		a.Source = specifications.Source{}
	}
	value, _ := json.Marshal(a.Value)
	source, _ := json.Marshal(a.Source)
	err = tx.QueryRow(ctx, `INSERT INTO specification_assertions(org_id,project_id,id,type_id,application_id,attribute_key,meaning,value,source,origin,status,actor,reason) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,'proposed',$11,$12) RETURNING created_at`, org, project, a.ID, a.TypeID, a.ApplicationID, a.Key, a.Meaning, value, source, a.Origin, actor, a.Reason).Scan(&a.CreatedAt)
	if err != nil {
		return a, err
	}
	return a, s.finishSpecification(ctx, tx, org, project, actor, "assertion", a.ID, a)
}

type SpecificationReview struct {
	Version            int64  `json:"version"`
	Status             string `json:"status"`
	ExpectedAcceptedID string `json:"expected_accepted_id"`
	SourceReviewed     bool   `json:"source_reviewed"`
	Reason             string `json:"reason"`
}

func (s *Store) ReviewSpecificationAssertion(ctx context.Context, org, project, actor, id string, in SpecificationReview) (specifications.Assertion, error) {
	var a specifications.Assertion
	if !specificationID(id, false) {
		return a, ErrNotFound
	}
	if (in.Status != "accepted" && in.Status != "rejected") || in.Version < 1 || len(in.Reason) > 1000 {
		return a, specificationInvalid(errors.New("invalid review"))
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	var value, source []byte
	err = tx.QueryRow(ctx, `SELECT id::text,type_id::text,COALESCE(application_id::text,''),attribute_key,meaning,value,source,origin,status,actor::text,reason,version,created_at FROM specification_assertions WHERE org_id=$1 AND project_id=$2 AND id=$3 FOR UPDATE`, org, project, id).Scan(&a.ID, &a.TypeID, &a.ApplicationID, &a.Key, &a.Meaning, &value, &source, &a.Origin, &a.Status, &a.Actor, &a.Reason, &a.Version, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	if err = json.Unmarshal(value, &a.Value); err != nil {
		return a, err
	}
	if err = json.Unmarshal(source, &a.Source); err != nil {
		return a, err
	}
	if a.Version != in.Version || a.Status != "proposed" {
		return a, ErrVersionConflict
	}
	if err = specificationTypeExists(ctx, tx, org, project, a.TypeID); err != nil {
		return a, err
	}
	if in.Status == "accepted" {
		if a.Origin == "user" {
			external, e := specificationExternal(ctx, tx, org, project, a.TypeID)
			if e != nil {
				return a, e
			}
			if external {
				return a, specificationInvalid(errors.New("this item is maintained externally; keep the change proposed or explicitly adopt maintenance"))
			}
		}
		if a.Source.DocumentID != "" {
			docs, e := specificationDocuments(ctx, tx, org, project)
			if e != nil {
				return a, e
			}
			doc, ok := docs[a.Source.DocumentID]
			if !ok || doc.Stale || (a.Source.Hash != "" && a.Source.Hash != doc.Hash) {
				return a, specificationInvalid(errors.New("review the current source revision before confirming this reading"))
			}
			var exists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE org_id=$1 AND project_id=$2 AND id=$3)`, org, project, a.Source.DocumentID).Scan(&exists); err != nil {
				return a, err
			}
			if !exists {
				return a, specificationInvalid(errors.New("source document is unavailable"))
			}
			if a.Source.Marker == "withdrawn" {
				return a, specificationInvalid(errors.New("withdrawn source items cannot become active choices"))
			}
			if !in.SourceReviewed {
				return a, specificationInvalid(errors.New("review the source page and its visual markings before accepting"))
			}
		}
		var prior string
		err = tx.QueryRow(ctx, `SELECT id::text FROM specification_assertions WHERE org_id=$1 AND project_id=$2 AND type_id=$3 AND application_id IS NOT DISTINCT FROM NULLIF($4,'')::uuid AND attribute_key=$5 AND meaning=$6 AND status='accepted'`, org, project, a.TypeID, a.ApplicationID, a.Key, a.Meaning).Scan(&prior)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return a, err
		}
		if prior != in.ExpectedAcceptedID {
			return a, ErrVersionConflict
		}
		if prior != "" {
			if strings.TrimSpace(in.Reason) == "" {
				return a, specificationInvalid(errors.New("explain why the accepted value is being replaced"))
			}
			_, err = tx.Exec(ctx, `UPDATE specification_assertions SET status='superseded',version=version+1 WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, prior)
			if err != nil {
				return a, err
			}
			_, err = tx.Exec(ctx, `INSERT INTO specification_history(org_id,project_id,entity_id,entity_kind,actor,content) VALUES($1,$2,$3,'superseded',$4,jsonb_build_object('replaced_by',$5::text,'reason',$6::text))`, org, project, prior, actor, a.ID, in.Reason)
			if err != nil {
				return a, err
			}
		}
	}
	a.Status = in.Status
	a.Reason = in.Reason
	a.Version++
	_, err = tx.Exec(ctx, `UPDATE specification_assertions SET status=$4,reason=$5,version=version+1 WHERE org_id=$1 AND project_id=$2 AND id=$3`, org, project, id, a.Status, a.Reason)
	if err != nil {
		return a, err
	}
	return a, s.finishSpecification(ctx, tx, org, project, actor, "review", a.ID, a)
}

func (s *Store) AddSpecificationRelationship(ctx context.Context, org, project, actor string, r specifications.Relationship) (specifications.Relationship, error) {
	if err := specifications.ValidateRelationship(r); err != nil {
		return r, specificationInvalid(err)
	}
	tx, err := s.specificationTx(ctx, org, project, actor)
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	for _, id := range []string{r.FromID, r.ToID} {
		if err = specificationTypeExists(ctx, tx, org, project, id); err != nil {
			return r, err
		}
	}
	owners := []string{r.FromID}
	if r.Kind == "component_of" {
		owners = append(owners, r.ToID)
	}
	for _, id := range owners {
		external, e := specificationExternal(ctx, tx, org, project, id)
		if e != nil {
			return r, e
		}
		if external {
			return r, specificationInvalid(errors.New("adopt maintenance before changing an externally maintained assembly or requirement"))
		}
	}
	if r.Kind != "matches" {
		var cycle bool
		err = tx.QueryRow(ctx, `WITH RECURSIVE linked(id) AS (SELECT $3::uuid UNION SELECT l.to_id FROM specification_relationships l JOIN linked p ON l.from_id=p.id WHERE l.org_id=$1 AND l.project_id=$2 AND l.kind<>'matches') SELECT EXISTS(SELECT 1 FROM linked WHERE id=$4::uuid)`, org, project, r.ToID, r.FromID).Scan(&cycle)
		if err != nil {
			return r, err
		}
		if cycle {
			return r, specificationInvalid(errors.New("relationship would create a dependency cycle"))
		}
	}
	r.ID = newID()
	r.Version = 1
	raw, _ := json.Marshal(r)
	_, err = tx.Exec(ctx, `INSERT INTO specification_relationships(org_id,project_id,id,from_id,to_id,kind,content) VALUES($1,$2,$3,$4,$5,$6,$7)`, org, project, r.ID, r.FromID, r.ToID, r.Kind, raw)
	if err != nil {
		return r, specificationInvalid(err)
	}
	return r, s.finishSpecification(ctx, tx, org, project, actor, "relationship", r.ID, r)
}
