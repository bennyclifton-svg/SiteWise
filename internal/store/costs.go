package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sitewise/internal/costs"
)

var ErrFrozenCostPlan = errors.New("frozen")

type CostWrite struct {
	PlanVersionID string `json:"plan_version_id"`
	Version       int64  `json:"version"`
}
type CostItemInput struct {
	CostWrite
	costs.Content
	Values map[string]costs.Value `json:"values"`
}
type CostValueInput struct {
	PlanVersionID string `json:"plan_version_id"`
	PlanVersion   int64  `json:"plan_version"`
	Calculate     bool   `json:"calculate"`
	costs.Value
}
type CostSettingsInput struct {
	CostWrite
	costs.Settings
}

func (s *Store) ReadCostPlan(ctx context.Context, org, project, version string) (costs.Plan, error) {
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return costs.Plan{}, e
	}
	defer tx.Rollback(ctx)
	p, e := readCostPlan(ctx, tx, org, project, version)
	if e != nil {
		return p, e
	}
	return p, tx.Commit(ctx)
}
func readCostPlan(ctx context.Context, tx pgx.Tx, org, project, version string) (costs.Plan, error) {
	p := costs.Plan{ProjectID: project, Status: "unavailable", Items: []costs.Item{}, Links: []costs.Link{}}
	var exists bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); e != nil {
		return p, e
	}
	if !exists {
		return p, ErrNotFound
	}
	if version != "" {
		var u pgtype.UUID
		if u.Scan(version) != nil {
			return p, costs.ErrInvalid
		}
	}
	var raw []byte
	e := tx.QueryRow(ctx, `SELECT to_jsonb(v)||jsonb_build_object('baseline_version_id',(SELECT baseline_version_id FROM cost_plans WHERE org_id=v.org_id AND project_id=v.project_id)) FROM cost_plan_versions v WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=COALESCE(NULLIF($3,'')::uuid,(SELECT draft_version_id FROM cost_plans WHERE org_id=$1::uuid AND project_id=$2::uuid))`, org, project, version).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		if version != "" {
			return p, ErrNotFound
		}
		return p, nil
	}
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(raw, &p); e != nil {
		return p, e
	}
	rows, e := tx.Query(ctx, `SELECT to_jsonb(r)||jsonb_build_object('system_id',CASE WHEN $4 THEN r.system_id ELSE w.system_id END,'part_id',CASE WHEN $4 THEN r.part_id ELSE w.part_id END) FROM cost_item_revisions r LEFT JOIN work_items w ON w.org_id=r.org_id AND w.project_id=r.project_id AND w.id=r.work_item_id WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.plan_version_id=$3::uuid ORDER BY r.code,r.label,r.cost_item_id`, org, project, p.ID, p.Status != "draft")
	if e != nil {
		return p, e
	}
	index := map[string]int{}
	for rows.Next() {
		var i costs.Item
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return p, e
		}
		if e = json.Unmarshal(raw, &i); e != nil {
			rows.Close()
			return p, e
		}
		i.Values = map[string]costs.Value{}
		index[i.ID] = len(p.Items)
		p.Items = append(p.Items, i)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, e
	}
	rows, e = tx.Query(ctx, `SELECT cost_item_id::text,metric,to_jsonb(v) FROM cost_values v WHERE org_id=$1::uuid AND project_id=$2::uuid AND plan_version_id=$3::uuid`, org, project, p.ID)
	if e != nil {
		return p, e
	}
	for rows.Next() {
		var id, m string
		var v costs.Value
		if e = rows.Scan(&id, &m, &raw); e != nil {
			rows.Close()
			return p, e
		}
		if e = json.Unmarshal(raw, &v); e != nil {
			rows.Close()
			return p, e
		}
		p.Items[index[id]].Values[m] = v
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return p, e
	}
	rows, e = tx.Query(ctx, `SELECT package_scope_item_id::text,cost_item_id::text FROM scope_cost_links WHERE org_id=$1::uuid AND project_id=$2::uuid AND plan_version_id=$3::uuid ORDER BY package_scope_item_id,cost_item_id`, org, project, p.ID)
	if e != nil {
		return p, e
	}
	defer rows.Close()
	for rows.Next() {
		var l costs.Link
		if e = rows.Scan(&l.ScopeID, &l.CostID); e != nil {
			return p, e
		}
		p.Links = append(p.Links, l)
	}
	return p, rows.Err()
}
func (s *Store) costTx(ctx context.Context, org, project, actor string, w CostWrite) (pgx.Tx, costs.Plan, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, costs.Plan{}, e
	}
	fail := func(e error) (pgx.Tx, costs.Plan, error) { tx.Rollback(ctx); return nil, costs.Plan{}, e }
	if e = lockProject(ctx, tx, org, project); e != nil {
		return fail(e)
	}
	if e = packageActor(ctx, tx, org, actor); e != nil {
		return fail(e)
	}
	p, e := readCostPlan(ctx, tx, org, project, w.PlanVersionID)
	if e != nil {
		return fail(e)
	}
	if p.Status == "unavailable" {
		if w.Version != 0 {
			return fail(ErrVersionConflict)
		}
		p.ID = newID()
		p.Version = 1
		p.Revision = 1
		p.Status = "draft"
		p.Currency = "AUD"
		p.TaxBasis = "ex_tax"
		_, e = tx.Exec(ctx, `INSERT INTO cost_plan_versions(org_id,project_id,id,revision,status,tax_basis) VALUES($1::uuid,$2::uuid,$3::uuid,1,'draft','ex_tax');`, org, project, p.ID)
		if e != nil {
			return fail(e)
		}
		_, e = tx.Exec(ctx, `INSERT INTO cost_plans(org_id,project_id,draft_version_id) VALUES($1::uuid,$2::uuid,$3::uuid)`, org, project, p.ID)
		if e != nil {
			return fail(e)
		}
	} else {
		if p.Status != "draft" {
			return fail(ErrFrozenCostPlan)
		}
		if w.PlanVersionID == "" || p.Version != w.Version {
			return fail(ErrVersionConflict)
		}
	}
	return tx, p, nil
}
func (s *Store) finishCost(ctx context.Context, tx pgx.Tx, org, project, id string) (costs.Plan, error) {
	if _, e := tx.Exec(ctx, `UPDATE cost_plan_versions SET version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, id); e != nil {
		return costs.Plan{}, e
	}
	if e := BumpRevision(ctx, tx, org, project, "costs"); e != nil {
		return costs.Plan{}, e
	}
	p, e := readCostPlan(ctx, tx, org, project, id)
	if e != nil {
		return p, e
	}
	raw, _ := json.Marshal(map[string]any{"project_id": project, "plan_version_id": id, "revision": p.Version})
	if _, e = appendEvent(ctx, s.q.WithTx(tx), org, "costs", "", string(raw)); e != nil {
		return p, e
	}
	return p, tx.Commit(ctx)
}
func (s *Store) SaveCostSettings(ctx context.Context, org, project, actor string, in CostSettingsInput) (costs.Plan, error) {
	if e := costs.ValidateSettings(&in.Settings); e != nil {
		return costs.Plan{}, e
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	if len(p.Items) > 0 && (p.Currency != in.Currency || p.TaxBasis != in.TaxBasis) {
		return p, fmt.Errorf("%w: currency and tax basis cannot change after items exist", costs.ErrInvalid)
	}
	_, e = tx.Exec(ctx, `UPDATE cost_plan_versions SET currency=$3,tax_basis=$4,tax_rate=$5::text::numeric,price_date=$6::text::date,coverage=$7,funding_target=$8::text::numeric,funding_target_provenance=$9::jsonb||jsonb_build_object('actor',$10::text,'at',now()) WHERE org_id=$1::uuid AND id=$2::uuid`, org, p.ID, in.Currency, in.TaxBasis, in.TaxRate, in.PriceDate, in.Coverage, in.FundingTarget, in.FundingTargetProvenance, actor)
	if e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func costTargets(ctx context.Context, tx pgx.Tx, org, project, version, id string, c costs.Content) error {
	var u pgtype.UUID
	for _, v := range []string{id, c.ParentItemID, c.WorkItemID, c.PackageID, c.PackageStageID} {
		if v != "" && u.Scan(v) != nil {
			return costs.ErrInvalid
		}
	}
	checks := []struct{ id, sql string }{
		{c.WorkItemID, `SELECT EXISTS(SELECT 1 FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL)`},
		{c.PackageID, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND retired_at IS NULL)`},
	}
	for _, c := range checks {
		if c.id == "" {
			continue
		}
		var ok bool
		if e := tx.QueryRow(ctx, c.sql, org, project, c.id).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return ErrNotFound
		}
	}
	if c.PackageStageID != "" {
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_stages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND package_id=$4::uuid AND retired_at IS NULL)`, org, project, c.PackageStageID, c.PackageID).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return ErrNotFound
		}
	}
	if c.WorkItemID != "" && c.PackageID != "" {
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM package_scope_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND work_item_id=$4::uuid AND item_kind='responsibility' AND inclusion='included' AND retired_at IS NULL)`, org, project, c.PackageID, c.WorkItemID).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return fmt.Errorf("%w: package needs a live responsibility for the work item", costs.ErrInvalid)
		}
	}
	if c.ParentItemID != "" {
		var ok bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND NOT posting)`, org, version, c.ParentItemID).Scan(&ok); e != nil {
			return e
		}
		if !ok {
			return costs.ErrInvalid
		}
		var cycle bool
		if e := tx.QueryRow(ctx, `WITH RECURSIVE parents AS (SELECT cost_item_id,parent_item_id FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid UNION SELECT r.cost_item_id,r.parent_item_id FROM cost_item_revisions r JOIN parents p ON r.cost_item_id=p.parent_item_id WHERE r.org_id=$1::uuid AND r.plan_version_id=$2::uuid) SELECT EXISTS(SELECT 1 FROM parents WHERE cost_item_id=$4::uuid)`, org, version, c.ParentItemID, id).Scan(&cycle); e != nil {
			return e
		}
		if cycle {
			return costs.ErrInvalid
		}
	}
	return nil
}
func insertCostItem(ctx context.Context, tx pgx.Tx, org, project, version, actor, id string, c costs.Content) error {
	if e := costs.ValidateContent(c); e != nil {
		return e
	}
	if e := costTargets(ctx, tx, org, project, version, id, c); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, org, project, id, version); e != nil {
		return e
	}
	return writeCostContent(ctx, tx, org, project, version, actor, id, c, false)
}
func writeCostContent(ctx context.Context, tx pgx.Tx, org, project, version, actor, id string, c costs.Content, update bool) error {
	raw, _ := json.Marshal(c)
	// jsonb_populate_record preserves decimal inputs without a binary-number hop.
	sql := `INSERT INTO cost_item_revisions(org_id,project_id,plan_version_id,cost_item_id,parent_item_id,code,label,line_kind,category,work_item_id,package_id,package_stage_id,posting,quantity,unit,rate,rate_basis,excluded,origin,meaning,rationale,provenance) SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,NULLIF(x.parent_item_id,'')::uuid,x.code,x.label,x.line_kind,NULLIF(x.category,''),NULLIF(x.work_item_id,'')::uuid,NULLIF(x.package_id,'')::uuid,NULLIF(x.package_stage_id,'')::uuid,x.posting,x.quantity::numeric,NULLIF(x.unit,''),x.rate::numeric,NULLIF(x.rate_basis,''),x.excluded,x.origin,x.meaning,x.rationale,jsonb_build_object('actor',$6::text,'at',now()) FROM jsonb_to_record($5::jsonb) AS x(parent_item_id text,code text,label text,line_kind text,category text,work_item_id text,package_id text,package_stage_id text,posting boolean,quantity text,unit text,rate text,rate_basis text,excluded boolean,origin text,meaning text,rationale text)`
	if update {
		sql += ` ON CONFLICT(org_id,plan_version_id,cost_item_id) DO UPDATE SET parent_item_id=excluded.parent_item_id,code=excluded.code,label=excluded.label,line_kind=excluded.line_kind,category=excluded.category,work_item_id=excluded.work_item_id,package_id=excluded.package_id,package_stage_id=excluded.package_stage_id,posting=excluded.posting,quantity=excluded.quantity,unit=excluded.unit,rate=excluded.rate,rate_basis=excluded.rate_basis,excluded=excluded.excluded,origin=excluded.origin,meaning=excluded.meaning,rationale=excluded.rationale,provenance=cost_item_revisions.provenance||excluded.provenance,version=cost_item_revisions.version+1`
	}
	_, e := tx.Exec(ctx, sql, org, project, version, id, raw, actor)
	return e
}
func (s *Store) CreateCostItem(ctx context.Context, org, project, actor string, in CostItemInput) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	id := newID()
	if e = insertCostItem(ctx, tx, org, project, p.ID, actor, id, in.Content); e != nil {
		return p, e
	}
	for m, v := range in.Values {
		if !in.Posting {
			return p, costs.ErrInvalid
		}
		if e = writeCostValue(ctx, tx, org, project, p.ID, id, actor, m, v); e != nil {
			return p, e
		}
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func (s *Store) PatchCostItem(ctx context.Context, org, project, id, actor string, in CostItemInput) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	found := false
	for _, i := range p.Items {
		if i.ID == id {
			found = true
			if i.Posting != in.Posting {
				return p, fmt.Errorf("%w: use subdivision to convert a posting item to a group", costs.ErrInvalid)
			}
		}
	}
	if !found {
		return p, ErrNotFound
	}
	if len(in.Values) > 0 {
		return p, costs.ErrInvalid
	}
	if e = costs.ValidateContent(in.Content); e != nil {
		return p, e
	}
	if e = costTargets(ctx, tx, org, project, p.ID, id, in.Content); e != nil {
		return p, e
	}
	if e = writeCostContent(ctx, tx, org, project, p.ID, actor, id, in.Content, true); e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func writeCostValue(ctx context.Context, tx pgx.Tx, org, project, version, item, actor, metric string, v costs.Value) error {
	if e := costs.ValidateValue(metric, &v); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, `INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,meaning,rationale,provenance) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::text::numeric,$8::text::numeric,$9::text::numeric,$10::text::date,$11,$12,$13,jsonb_build_object('actor',$14::text,'at',now())) ON CONFLICT(org_id,plan_version_id,cost_item_id,metric) DO UPDATE SET value_state=excluded.value_state,amount=excluded.amount,low=excluded.low,high=excluded.high,as_of=excluded.as_of,origin=excluded.origin,meaning=excluded.meaning,rationale=excluded.rationale,provenance=excluded.provenance,version=cost_values.version+1`, org, project, version, item, metric, v.ValueState, v.Amount, v.Low, v.High, v.AsOf, v.Origin, v.Meaning, v.Rationale, actor)
	return e
}
func (s *Store) PutCostValue(ctx context.Context, org, project, item, actor, metric string, in CostValueInput) (costs.Plan, error) {
	tx, p, e := s.costTx(ctx, org, project, actor, CostWrite{in.PlanVersionID, in.PlanVersion})
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	found := false
	var chosen costs.Item
	for _, i := range p.Items {
		if i.ID == item {
			found = true
			chosen = i
			if e = calculateCostInput(&in, metric, i); e != nil {
				return p, e
			}
			if !i.Posting {
				return p, costs.ErrInvalid
			}
			if i.Values[metric].Version != in.Version {
				return p, ErrVersionConflict
			}
		}
	}
	if !found {
		return p, ErrNotFound
	}
	if e = writeCostValue(ctx, tx, org, project, p.ID, item, actor, metric, in.Value); e != nil {
		return p, e
	}
	if in.Calculate {
		if e = recordCostCalculation(ctx, tx, org, p.ID, item, metric, chosen); e != nil {
			return p, e
		}
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func (s *Store) CostTotals(ctx context.Context, org, project, version, by string) (costs.Totals, error) {
	p, e := s.ReadCostPlan(ctx, org, project, version)
	if e != nil {
		return costs.Totals{}, e
	}
	return costs.Summarize(p, by)
}

// readCostPlanTx keeps report assembly and issue snapshots on the same MVCC view.
func readCostPlanTx(ctx context.Context, tx pgx.Tx, org, project, version string) (costs.Plan, error) {
	return readCostPlan(ctx, tx, org, project, version)
}
