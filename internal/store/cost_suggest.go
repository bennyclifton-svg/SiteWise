package store

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"regexp"
	"sitewise/internal/costs"
	"sort"
	"strings"
)

func (s *Store) costSuggestInput(ctx context.Context, tx pgx.Tx, org, project string, p costs.Plan) (costs.SuggestInput, error) {
	in := costs.SuggestInput{Catalog: s.workCatalog(), Existing: p.Items, Values: map[string]string{}, Dismissed: map[string]bool{}}
	fields := []string{}
	if in.Catalog != nil {
		fields = in.Catalog.CostStarterFields()
	}
	b := &pgx.Batch{}
	b.Queue(`SELECT id::text,title,kind,COALESCE(discipline_id,''),COALESCE(works_scope,''),retired_at IS NOT NULL FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY title,id`, org, project)
	b.Queue(`SELECT id::text,system_id,title,part_id::text FROM work_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND inclusion='included' AND retired_at IS NULL AND NOT is_group ORDER BY system_id,id`, org, project)
	b.Queue(`SELECT source_key FROM cost_suggestion_dismissals WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY source_key`, org, project)
	b.Queue(`SELECT key,CASE WHEN count(DISTINCT value)=1 AND bool_and(value_state='known') THEN min(value) ELSE '' END FROM profile_rows WHERE org_id=$1::uuid AND project_id=$2::uuid AND key=ANY($3::text[]) GROUP BY key`, org, project, fields)
	br := tx.SendBatch(ctx, b)
	defer br.Close()
	rows, e := br.Query()
	if e != nil {
		return in, e
	}
	for rows.Next() {
		var p costs.SuggestPackage
		if e = rows.Scan(&p.ID, &p.Title, &p.Kind, &p.DisciplineID, &p.WorksScope, &p.Retired); e != nil {
			rows.Close()
			return in, e
		}
		in.Packages = append(in.Packages, p)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return in, e
	}
	rows, e = br.Query()
	if e != nil {
		return in, e
	}
	for rows.Next() {
		var w costs.ScopeWork
		if e = rows.Scan(&w.ID, &w.SystemID, &w.Title, &w.PartID); e != nil {
			rows.Close()
			return in, e
		}
		in.Works = append(in.Works, w)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return in, e
	}
	rows, e = br.Query()
	if e != nil {
		return in, e
	}
	for rows.Next() {
		var k string
		if e = rows.Scan(&k); e != nil {
			rows.Close()
			return in, e
		}
		in.Dismissed[k] = true
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return in, e
	}
	rows, e = br.Query()
	if e != nil {
		return in, e
	}
	for rows.Next() {
		var k, v string
		if e = rows.Scan(&k, &v); e != nil {
			rows.Close()
			return in, e
		}
		if v != "" {
			in.Values[k] = v
		}
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return in, e
	}
	return in, br.Close()
}
func (s *Store) decorateCostPlan(ctx context.Context, tx pgx.Tx, org, project string, p *costs.Plan) error {
	p.Suggestions = []costs.Suggestion{}
	p.Dismissed = []costs.Suggestion{}
	p.Systems = []costs.CostSystem{}
	if p.Status != "draft" && p.Status != "unavailable" {
		return nil
	}
	if e := s.attachConstructionBudget(ctx, tx, org, project, p); e != nil {
		return e
	}
	cat := s.workCatalog()
	if cat == nil {
		return nil
	}
	for _, sys := range cat.BuildingOrderSystems() {
		p.Systems = append(p.Systems, costs.CostSystem{ID: sys.ID, Label: sys.Label})
	}
	in, e := s.costSuggestInput(ctx, tx, org, project, *p)
	if e != nil {
		return e
	}
	p.Suggestions = costs.Suggest(in)
	dismissed := in.Dismissed
	in.Dismissed = nil
	all := map[string]costs.Suggestion{}
	for _, sg := range costs.Suggest(in) {
		all[sg.Key] = sg
	}
	keys := []string{}
	for k := range dismissed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sg, ok := all[k]
		if !ok {
			sg = costs.Suggestion{Key: k, Content: costs.Content{Label: k, Rationale: "Source no longer applies. Restoring permits it to be suggested if it becomes applicable again."}}
		}
		p.Dismissed = append(p.Dismissed, sg)
	}
	for n := range p.Items {
		i := &p.Items[n]
		i.ResolvedPackageID, i.ResolvedCategory = costs.ResolveCostPackage(cat, *i, p.Items, in.Packages)
		i.ScopeWorkItems = []costs.ScopeWork{}
		if i.LineKind == "works" {
			i.ScopeWorkItems = costs.ScopeFor(cat, i.SystemID, in.Works)
			if strings.HasSuffix(i.SourceKey, ":remaining") {
				remaining := []costs.ScopeWork{}
				parentKey := ""
				for _, parent := range p.Items {
					if parent.ID == i.ParentItemID {
						parentKey = parent.SourceKey
					}
				}
				for _, w := range i.ScopeWorkItems {
					separate := false
					for _, sibling := range p.Items {
						if sibling.ParentItemID == i.ParentItemID && sibling.ID != i.ID && sibling.SystemID != i.SystemID && cat.SystemWithin(w.SystemID, sibling.SystemID) {
							separate = true
						}
					}
					for _, proposed := range p.Suggestions {
						if proposed.ParentKey == parentKey && proposed.SystemID != i.SystemID && cat.SystemWithin(w.SystemID, proposed.SystemID) {
							separate = true
						}
					}
					if !separate {
						remaining = append(remaining, w)
					}
				}
				i.ScopeWorkItems = remaining
			}

			if i.WorkItemID != "" {
				ws := []costs.ScopeWork{}
				for _, w := range i.ScopeWorkItems {
					if w.ID == i.WorkItemID {
						ws = append(ws, w)
					}
				}
				i.ScopeWorkItems = ws
			}
		}
	}
	return nil
}

type CostSuggestionInput struct {
	CostWrite
	Budget *costs.Money `json:"budget"`
}

var costKeyPattern = regexp.MustCompile(`^(fee:[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}|works:[a-z0-9.-]+(?::remaining)?|pw:[a-z][a-z0-9-]*)$`)

func validCostKey(key string) bool { return len(key) <= 200 && costKeyPattern.MatchString(key) }

func (s *Store) AcceptCostSuggestions(ctx context.Context, org, project, actor, key string, in CostSuggestionInput) (costs.Plan, error) {
	if key != "" && !validCostKey(key) {
		return costs.Plan{}, costs.ErrInvalid
	}
	if key == "" && in.Budget != nil {
		return costs.Plan{}, costs.ErrInvalid
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	input, e := s.costSuggestInput(ctx, tx, org, project, p)
	if e != nil {
		return p, e
	}
	suggestions := costs.Suggest(input)
	ids := map[string]string{}
	for _, i := range p.Items {
		if i.SourceKey != "" {
			ids[i.SourceKey] = i.ID
		}
	}
	// Current-version repeats never replace an already saved budget.
	if key != "" && ids[key] != "" {
		if e = s.decorateCostPlan(ctx, tx, org, project, &p); e != nil {
			return p, e
		}
		return p, tx.Commit(ctx)
	}
	selected := map[string]bool{}
	if key != "" {
		for _, sg := range suggestions {
			if sg.Key == key {
				selected[key] = true
				if sg.ParentKey != "" {
					selected[sg.ParentKey] = true
				}
			}
		}
		if !selected[key] {
			return p, ErrNotFound
		}
	}
	count, e := s.insertCostSuggestions(ctx, tx, org, project, actor, p, suggestions, key, selected, ids, in, input.Catalog.Version())
	if e != nil {
		return p, e
	}

	if count == 0 {
		if e = s.decorateCostPlan(ctx, tx, org, project, &p); e != nil {
			return p, e
		}
		return p, tx.Commit(ctx)
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
func dismissCostKey(ctx context.Context, tx pgx.Tx, org, project, actor, key string) error {
	_, e := tx.Exec(ctx, `INSERT INTO cost_suggestion_dismissals(org_id,project_id,source_key,actor) VALUES($1::uuid,$2::uuid,$3,$4::uuid) ON CONFLICT DO NOTHING`, org, project, key, actor)
	return e
}
func (s *Store) DismissCostSuggestion(ctx context.Context, org, project, actor, key string, restore bool, in CostWrite) (costs.Plan, error) {
	if !validCostKey(key) {
		return costs.Plan{}, costs.ErrInvalid
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	if restore {
		_, e = tx.Exec(ctx, `DELETE FROM cost_suggestion_dismissals WHERE org_id=$1::uuid AND project_id=$2::uuid AND source_key=$3`, org, project, key)
	} else {
		e = dismissCostKey(ctx, tx, org, project, actor, key)
	}
	if e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}

func calculateItemBudget(ctx context.Context, tx pgx.Tx, org, project, version, id, actor string, c costs.Content) error {
	if !c.Posting || c.Quantity == nil || c.Rate == nil {
		return costs.ErrInvalid
	}
	amount, e := costs.Multiply(string(*c.Quantity), string(*c.Rate))
	if e != nil {
		return e
	}
	if e = writeCostValue(ctx, tx, org, project, version, id, actor, "budget", costs.Value{Amount: &amount, ValueState: "known", Origin: "user", Meaning: "allowance"}); e != nil {
		return e
	}
	return recordCostCalculation(ctx, tx, org, version, id, "budget", costs.Item{Content: c})
}

func (s *Store) insertCostSuggestions(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan, suggestions []costs.Suggestion, key string, selected map[string]bool, ids map[string]string, in CostSuggestionInput, knowledgeVersion string) (int, error) {
	count := 0
	batch := &pgx.Batch{}
	var budgetItem string
	for _, sg := range suggestions {
		if key != "" && !selected[sg.Key] {
			continue
		}
		c := sg.Content
		if sg.ParentKey != "" {
			c.ParentItemID = ids[sg.ParentKey]
			if c.ParentItemID == "" {
				return 0, costs.ErrInvalid
			}
		}
		id := newID()

		// These targets came from the tenant-scoped snapshot under the project lock.
		// Batch the inserts while retaining database FKs and the source uniqueness gate.
		if e := costs.ValidateContent(c); e != nil {
			return 0, e
		}
		raw, _ := json.Marshal(c)
		batch.Queue(`INSERT INTO cost_items(org_id,project_id,id,created_in_version_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid)`, org, project, id, p.ID)
		batch.Queue(costContentSQL, org, project, p.ID, id, raw, actor)
		batch.Queue(`UPDATE cost_item_revisions SET source_key=$4,provenance=provenance||jsonb_build_object('source_key',$4::text,'knowledge_version',$5::text) WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, id, sg.Key, knowledgeVersion)

		if sg.Key == key && in.Budget != nil {
			if !c.Posting {
				return 0, costs.ErrInvalid
			}
			budgetItem = id
		}
		ids[sg.Key] = id
		count++
	}
	if count > 0 {
		if e := tx.SendBatch(ctx, batch).Close(); e != nil {
			return 0, e
		}
	}
	if budgetItem != "" {
		if e := writeCostValue(ctx, tx, org, project, p.ID, budgetItem, actor, "budget", costs.Value{Amount: in.Budget, ValueState: "known", Origin: "user", Meaning: "allowance"}); e != nil {
			return 0, e
		}
	}

	return count, nil
}

// prepareCostLines shares the same derivation and insertion rules as legacy acceptance.
func (s *Store) prepareCostLines(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan) (int, error) {
	input, e := s.costSuggestInput(ctx, tx, org, project, p)
	if e != nil {
		return 0, e
	}
	if input.Catalog == nil {
		return 0, nil
	}
	ids := map[string]string{}
	for _, i := range p.Items {
		if i.SourceKey != "" {
			ids[i.SourceKey] = i.ID
		}
	}
	return s.insertCostSuggestions(ctx, tx, org, project, actor, p, costs.Suggest(input), "", nil, ids, CostSuggestionInput{}, input.Catalog.Version())
}

type PreparedCostPlan struct {
	Plan    costs.Plan   `json:"plan"`
	Totals  costs.Totals `json:"totals"`
	Changed bool         `json:"-"`
}

func (s *Store) PrepareCostPlan(ctx context.Context, org, project, actor string) (PreparedCostPlan, error) {
	tx, p, e := s.costTxMode(ctx, org, project, actor, CostWrite{}, true)
	if e != nil {
		return PreparedCostPlan{}, e
	}
	defer tx.Rollback(ctx)
	count, e := s.prepareCostLines(ctx, tx, org, project, actor, p)
	if e != nil {
		return PreparedCostPlan{}, e
	}
	// The stated budget is spread over the lines just prepared, so the
	// plan is read again inside the transaction first.
	if count > 0 {
		if p, e = readCostPlan(ctx, tx, org, project, p.ID); e != nil {
			return PreparedCostPlan{}, e
		}
	}
	spread, e := s.spreadConstructionBudget(ctx, tx, org, project, actor, p)
	if e != nil {
		return PreparedCostPlan{}, e
	}
	count += spread
	brief, e := s.applyBriefFigures(ctx, tx, org, project, actor, p)
	if e != nil {
		return PreparedCostPlan{}, e
	}
	count += brief
	if count > 0 {
		p, e = s.finishCost(ctx, tx, org, project, p.ID)
	} else {
		e = s.decorateCostPlan(ctx, tx, org, project, &p)
		if e == nil {
			e = tx.Commit(ctx)
		}
	}
	if e != nil {
		return PreparedCostPlan{}, e
	}
	totals, e := costs.Summarize(p, "overall")
	return PreparedCostPlan{Plan: p, Totals: totals, Changed: count > 0}, e
}
