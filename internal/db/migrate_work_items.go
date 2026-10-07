package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/works"
)

// backfillWorkItems is part of migration 015, before its version is recorded.
// DDL, UUID generation, assertions and deletion either all commit or all roll
// back. No extension or live knowledge tree is needed to restore a database.
func backfillWorkItems(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `SELECT v.org_id::text,v.project_id::text,v.site_id::text,v.part_id::text,
 substr(v.key,7),v.value,v.scope,COALESCE(v.user_id::text,''),v.note,
 COALESCE((SELECT value FROM profile_user_values p WHERE p.org_id=v.org_id AND p.project_id=v.project_id AND p.part_id=v.part_id AND p.key='hdr.work_type'),
 (SELECT p.value FROM profile_user_values p JOIN project_parts pp ON pp.org_id=p.org_id AND pp.id=p.part_id WHERE p.org_id=v.org_id AND p.project_id=v.project_id AND pp.kind='whole' AND p.key='hdr.work_type'),
 (SELECT r.value FROM profile_rows r JOIN project_parts pp ON pp.org_id=r.org_id AND pp.id=r.part_id WHERE r.org_id=v.org_id AND r.project_id=v.project_id AND pp.kind='whole' AND r.key='hdr.work_type' AND r.band IN ('user','amber','green')), '')
 FROM profile_user_values v WHERE v.key LIKE 'scope.%' ORDER BY v.org_id,v.project_id,v.part_id,v.key`)
	if err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for rows.Next() {
		var org, project, site, part, system, scope, actor, note, kind string
		var value *string
		if err := rows.Scan(&org, &project, &site, &part, &system, &value, &scope, &actor, &note, &kind); err != nil {
			rows.Close()
			return err
		}
		if value == nil || (*value != "in" && *value != "out") || scope != "project" || system == "" {
			rows.Close()
			return fmt.Errorf("migration 015: unexpected legacy scope value; nothing deleted")
		}
		inclusion := "included"
		if *value == "out" {
			inclusion = "excluded"
		}
		batch.Queue(`INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,user_touched,coarse_key,provenance)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$6,'user','accepted_for_planning',true,$5||'|'||$6,jsonb_build_object('actor',$9::text,'rationale',$10::text,'band','user'))`, org, works.CoarseID(project, part, system), project, site, part, system, works.LegacyDefaultAction(kind), inclusion, actor, note)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}
	var mismatch bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM profile_user_values v LEFT JOIN work_items w ON w.org_id=v.org_id AND w.project_id=v.project_id AND w.part_id=v.part_id AND w.system_id=substr(v.key,7)
 WHERE v.key LIKE 'scope.%' AND (w.id IS NULL OR w.site_id<>v.site_id OR w.inclusion<>CASE v.value WHEN 'in' THEN 'included' ELSE 'excluded' END OR NOT w.user_touched))
 OR (SELECT count(*) FROM profile_user_values WHERE key LIKE 'scope.%')<>(SELECT count(*) FROM work_items)`).Scan(&mismatch)
	if err != nil {
		return err
	}
	if mismatch {
		return fmt.Errorf("migration 015: scope backfill assertion failed")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO project_revisions(org_id,project_id,works) SELECT DISTINCT org_id,project_id,1 FROM work_items
 ON CONFLICT(org_id,project_id) DO UPDATE SET works=project_revisions.works+1,version=project_revisions.version+1,updated_at=now()`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM profile_user_values WHERE key LIKE 'scope.%'`); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `ALTER TABLE profile_user_values ADD CONSTRAINT scope_lives_in_work_items CHECK (key NOT LIKE 'scope.%')`)
	return err
}
