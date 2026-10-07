package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

type CostBenchmarkSuggestion struct {
	Benchmark     knowledge.CostBenchmark `json:"benchmark"`
	Amount        costs.Money             `json:"amount"`
	Quantity      *costs.Money            `json:"quantity"`
	PlanVersionID string                  `json:"plan_version_id"`
	Version       int64                   `json:"version"`
	Limitations   string                  `json:"limitations"`
}
type CostBenchmarkInput struct {
	CostWrite
	BenchmarkID      string `json:"benchmark_id"`
	BenchmarkVersion int    `json:"benchmark_version"`
	Geography        string `json:"geography"`
	Quality          string `json:"quality"`
	AcknowledgeBasis bool   `json:"acknowledge_basis"`
}

func (s *Store) ReadCostBenchmarks(ctx context.Context, org, project, item, geography, quality string) ([]CostBenchmarkSuggestion, error) {
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	p, e := readCostPlan(ctx, tx, org, project, "")
	if e != nil {
		return nil, e
	}
	out, e := s.costBenchmarks(ctx, tx, org, project, item, p, geography, quality)
	if e != nil {
		return nil, e
	}
	return out, tx.Commit(ctx)
}
func (s *Store) costBenchmarks(ctx context.Context, tx pgx.Tx, org, project, item string, p costs.Plan, geography, quality string) ([]CostBenchmarkSuggestion, error) {
	out := []CostBenchmarkSuggestion{}
	var line costs.Item
	for _, i := range p.Items {
		if i.ID == item {
			line = i
			break
		}
	}
	if line.ID == "" {
		return nil, ErrNotFound
	}
	if len(geography) > 200 || len(quality) > 200 {
		return nil, costs.ErrInvalid
	}
	if !line.Posting || line.Excluded || p.PriceDate == nil || s.profileBuild == nil {
		return out, nil
	}
	cat := s.workCatalog()
	if cat == nil || len(cat.CostBenchmarks()) == 0 {
		return out, nil
	}
	view, e := readProfileTx(ctx, tx, org, project, s.profileBuild.ReadKinds)
	if e != nil {
		return nil, e
	}
	if len(view.StaleFor(*s.profileBuild)) > 0 || view.PendingDocuments > 0 || view.FailedDocuments > 0 {
		return out, nil
	}
	items, e := readWorkItems(ctx, tx, org, project)
	if e != nil {
		return nil, e
	}
	inputs, e := profile.ProposalInputs(view.Rows, view.Parts, items, cat)
	if e != nil {
		return nil, e
	}
	part := line.PartID
	if part == "" {
		for _, p := range view.Parts {
			if p.Kind == "whole" {
				part = p.ID
				break
			}
		}
	}
	effective := inputs.Parts[part]
	env := knowledge.WorksEnv{Values: map[string]string{}, WorkTypes: effective.WorkTypes, Present: effective.Present, PresentState: effective.PresentState}
	for key, v := range effective.Values {
		env.Values[key] = v.Value
	}
	for _, w := range items {
		if w.PartID == part && (line.WorkItemID == "" || w.ID == line.WorkItemID) && w.Inclusion == "included" && w.RetiredAt == nil {
			env.Items = append(env.Items, knowledge.WorkItem{System: w.SystemID, Action: w.Action, LayoutChange: w.LayoutChange})
		}
	}
	if inputs.Existing != nil {
		env.Existing = func(system string) knowledge.Truth { return inputs.Existing(part, system) }
	}
	for _, b := range cat.CostBenchmarks() {
		if !cat.EligibleCostBenchmark(b, env, p.Currency, p.TaxBasis, *p.PriceDate, geography, quality) {
			continue
		}
		amount, e := costs.Normalize(costs.Money(b.Amount))
		if b.Basis == "rate" {
			if line.Quantity == nil || line.Unit != b.Unit {
				continue
			}
			amount, e = costs.Multiply(string(*line.Quantity), b.Amount)
		}
		if e != nil {
			continue
		}
		out = append(out, CostBenchmarkSuggestion{Benchmark: b, Amount: amount, Quantity: line.Quantity, PlanVersionID: p.ID, Version: p.Version, Limitations: "Planning estimate only. Exact benchmark geography, price date, tax basis, quality and units must match; no escalation or tax conversion is assumed. Review inclusions and exclusions before applying."})
	}
	return out, nil
}
func (s *Store) ApplyCostBenchmark(ctx context.Context, org, project, item, actor string, in CostBenchmarkInput) (costs.Plan, error) {
	if !in.AcknowledgeBasis {
		return costs.Plan{}, fmt.Errorf("%w: acknowledge benchmark inclusions, exclusions and limitations", costs.ErrInvalid)
	}
	tx, p, e := s.costTx(ctx, org, project, actor, in.CostWrite)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(ctx)
	suggestions, e := s.costBenchmarks(ctx, tx, org, project, item, p, in.Geography, in.Quality)
	if e != nil {
		return p, e
	}
	var selected *CostBenchmarkSuggestion
	for i := range suggestions {
		b := suggestions[i].Benchmark
		if b.ID == in.BenchmarkID && b.Version == in.BenchmarkVersion {
			selected = &suggestions[i]
			break
		}
	}
	if selected == nil {
		return p, fmt.Errorf("%w: no reviewed applicable benchmark; enter an estimate or leave unknown", costs.ErrInvalid)
	}
	value := costs.Value{Amount: &selected.Amount, ValueState: "known", Origin: "user", Meaning: "allowance", Rationale: "Reviewed benchmark applied after basis acknowledgement"}
	if e = writeCostValue(ctx, tx, org, project, p.ID, item, actor, "estimate", value); e != nil {
		return p, e
	}
	raw, _ := json.Marshal(map[string]any{"method": "reviewed_benchmark", "method_version": 1, "catalogue_version": s.workCatalog().Version(), "benchmark": selected.Benchmark, "quantity": selected.Quantity, "limitations": selected.Limitations, "basis_acknowledged": true})
	if _, e = tx.Exec(ctx, `UPDATE cost_values SET origin='calculation',provenance=provenance||$4::jsonb WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric='estimate'`, org, p.ID, item, raw); e != nil {
		return p, e
	}
	return s.finishCost(ctx, tx, org, project, p.ID)
}
