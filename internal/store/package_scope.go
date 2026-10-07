package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
)

type ScopeInput struct {
	procurement.ScopeContent
	SourceRefs []profile.Source `json:"source_refs"`
}

type ScopeItem struct {
	ScopeInput
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	PackageID         string          `json:"package_id"`
	Origin            string          `json:"origin"`
	ReviewStatus      string          `json:"review_status"`
	Meaning           string          `json:"meaning"`
	Provenance        json.RawMessage `json:"provenance"`
	SourceProposalKey string          `json:"source_proposal_key,omitempty"`
	RetiredAt         *time.Time      `json:"retired_at,omitempty"`
	Version           int64           `json:"version"`
	Provisional       bool            `json:"provisional"`
}

type ScopePatch struct {
	Version       int64             `json:"version"`
	ItemKind      *string           `json:"item_kind"`
	WorkItemID    *string           `json:"work_item_id"`
	Role          *string           `json:"role"`
	ClauseID      *string           `json:"clause_id"`
	ClauseVersion *int              `json:"clause_version"`
	UserText      *string           `json:"user_text"`
	StageID       *string           `json:"stage_id"`
	Inclusion     *string           `json:"inclusion"`
	Deliverable   *string           `json:"deliverable"`
	InterfaceIDs  *[]string         `json:"interface_ids"`
	SourceRefs    *[]profile.Source `json:"source_refs"`
	Retired       *bool             `json:"retired"`
}

func (p ScopePatch) apply(s *ScopeInput) {
	for dst, src := range map[*string]*string{&s.ItemKind: p.ItemKind, &s.WorkItemID: p.WorkItemID, &s.Role: p.Role, &s.ClauseID: p.ClauseID, &s.UserText: p.UserText, &s.StageID: p.StageID, &s.Inclusion: p.Inclusion, &s.Deliverable: p.Deliverable} {
		if src != nil {
			*dst = *src
		}
	}
	if p.ClauseVersion != nil {
		s.ClauseVersion = *p.ClauseVersion
	}
	if p.InterfaceIDs != nil {
		s.InterfaceIDs = *p.InterfaceIDs
	}
	if p.SourceRefs != nil {
		s.SourceRefs = *p.SourceRefs
	}
}

func (s *Store) ReadPackageScope(ctx context.Context, org, project, pkg string) ([]ScopeItem, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := readPackage(ctx, tx, org, project, pkg); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT to_jsonb(i) FROM package_scope_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND retired_at IS NULL ORDER BY id`, org, project, pkg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScopeItem{}
	for rows.Next() {
		var raw []byte
		var item ScopeItem
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		item.Provisional = s.scopeProvisional(item)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return out, tx.Commit(ctx)
}

func (s *Store) CreatePackageScope(ctx context.Context, org, project, pkg, actor string, input ScopeInput) (ScopeItem, error) {
	return s.writePackageScope(ctx, org, project, pkg, "", actor, input, nil)
}

func (s *Store) PatchPackageScope(ctx context.Context, org, project, pkg, id, actor string, patch ScopePatch) (ScopeItem, error) {
	if patch.Version < 1 {
		return ScopeItem{}, ErrInvalidPackage
	}
	return s.writePackageScope(ctx, org, project, pkg, id, actor, ScopeInput{}, &patch)
}

func (s *Store) writePackageScope(ctx context.Context, org, project, pkg, id, actor string, input ScopeInput, patch *ScopePatch) (ScopeItem, error) {
	var item ScopeItem
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return item, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return item, err
	}
	p, err := readPackage(ctx, tx, org, project, pkg)
	if err != nil {
		return item, err
	}
	retire := false
	if patch != nil {
		var raw []byte
		err = tx.QueryRow(ctx, `SELECT to_jsonb(i) FROM package_scope_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND id=$4::uuid AND retired_at IS NULL`, org, project, pkg, id).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			return item, ErrNotFound
		}
		if err != nil {
			return item, err
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return item, err
		}
		if item.Version != patch.Version {
			return item, ErrVersionConflict
		}
		input = item.ScopeInput
		patch.apply(&input)
		retire = patch.Retired != nil && *patch.Retired
		if err := costScopeChange(ctx, tx, org, project, pkg, id, item, input.ScopeContent, retire); err != nil {
			return item, err
		}
	} else {
		id = newID()
	}
	if err := procurement.ValidateScope(input.ScopeContent, p, s.workCatalog()); err != nil {
		return item, fmt.Errorf("%w: %v", ErrInvalidPackage, err)
	}
	if err := scopeTargets(ctx, tx, org, project, pkg, input.ScopeContent); err != nil {
		return item, err
	}
	// Preserve existing snapshots on unrelated edits, including after extraction
	// replaced passage IDs. New selections are always resolved against this project.
	if patch == nil || patch.SourceRefs != nil {
		input.SourceRefs, err = resolveScopeSources(ctx, tx, org, project, input.SourceRefs)
		if err != nil {
			return item, err
		}
	}
	if input.InterfaceIDs == nil {
		input.InterfaceIDs = []string{}
	}
	if input.SourceRefs == nil {
		input.SourceRefs = []profile.Source{}
	}
	sources, _ := json.Marshal(input.SourceRefs)
	var raw []byte
	err = tx.QueryRow(ctx, `INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,clause_id,clause_version,user_text,stage_id,inclusion,deliverable,interface_ids,source_refs,origin,review_status,provenance)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,NULLIF($6,'')::uuid,NULLIF($7,''),NULLIF($8,''),NULLIF($9,0),NULLIF($10,''),NULLIF($11,'')::uuid,$12,NULLIF($13,''),$14,$15::jsonb,'user','accepted_for_planning',jsonb_build_object('actor',$16::text))
 ON CONFLICT(org_id,id) DO UPDATE SET item_kind=EXCLUDED.item_kind,work_item_id=EXCLUDED.work_item_id,role=EXCLUDED.role,clause_id=EXCLUDED.clause_id,clause_version=EXCLUDED.clause_version,user_text=EXCLUDED.user_text,stage_id=EXCLUDED.stage_id,inclusion=EXCLUDED.inclusion,deliverable=EXCLUDED.deliverable,interface_ids=EXCLUDED.interface_ids,source_refs=EXCLUDED.source_refs,origin='user',review_status='accepted_for_planning',verified_by=NULL,verified_at=NULL,verification_basis=NULL,provenance=package_scope_items.provenance||jsonb_build_object('last_edited_by',$16::text),retired_at=CASE WHEN $17 THEN now() ELSE package_scope_items.retired_at END,version=package_scope_items.version+1 RETURNING to_jsonb(package_scope_items)`, org, id, project, pkg, input.ItemKind, input.WorkItemID, input.Role, input.ClauseID, input.ClauseVersion, input.UserText, input.StageID, input.Inclusion, input.Deliverable, input.InterfaceIDs, sources, actor, retire).Scan(&raw)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return item, err
	}
	item.Provisional = s.scopeProvisional(item)
	return item, s.finishPackagesWrite(ctx, tx, org, project)
}

func scopeTargets(ctx context.Context, tx pgx.Tx, org, project, pkg string, input procurement.ScopeContent) error {
	for _, target := range []struct {
		id, query string
		stage     bool
	}{
		{input.WorkItemID, `SELECT EXISTS(SELECT 1 FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL)`, false},
		{input.StageID, `SELECT EXISTS(SELECT 1 FROM package_stages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND package_id=$4::uuid AND retired_at IS NULL)`, true},
	} {
		if target.id == "" {
			continue
		}
		var uuid pgtype.UUID
		if uuid.Scan(target.id) != nil {
			return ErrInvalidPackage
		}
		args := []any{org, project, target.id}
		var live bool
		if target.stage {
			args = append(args, pkg)
		}
		if err := tx.QueryRow(ctx, target.query, args...).Scan(&live); err != nil {
			return err
		}
		if !live {
			return ErrNotFound
		}
	}
	return nil
}

func resolveScopeSources(ctx context.Context, tx pgx.Tx, org, project string, refs []profile.Source) ([]profile.Source, error) {
	if len(refs) > 100 {
		return nil, ErrInvalidPackage
	}
	out := []profile.Source{}
	for _, ref := range refs {
		var uuid pgtype.UUID
		if uuid.Scan(ref.DocumentID) != nil || (ref.PassageID != "" && uuid.Scan(ref.PassageID) != nil) {
			return nil, ErrInvalidPackage
		}
		var src profile.Source
		err := tx.QueryRow(ctx, `SELECT d.id::text,encode(f.sha256,'hex'),d.filename,COALESCE(d.document_number,''),COALESCE(d.revision,''),COALESCE(p.id::text,''),COALESCE(p.body,''),COALESCE(ps.page,0),COALESCE(ps.location,''),COALESCE(ps.section,''),COALESCE(ps.start_offset,0),COALESCE(ps.end_offset,0)
 FROM documents d JOIN files f ON f.org_id=d.org_id AND f.id=d.file_id
 LEFT JOIN passages p ON p.org_id=d.org_id AND p.document_id=d.id AND p.id=NULLIF($4,'')::uuid
 LEFT JOIN passage_sources ps ON ps.org_id=p.org_id AND ps.passage_id=p.id
 WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id=$3::uuid AND ($4='' OR p.id IS NOT NULL)`, org, project, ref.DocumentID, ref.PassageID).Scan(&src.DocumentID, &src.FileSHA256, &src.Filename, &src.DocumentNumber, &src.Revision, &src.PassageID, &src.Excerpt, &src.Page, &src.Location, &src.Section, &src.StartOffset, &src.EndOffset)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, nil
}
