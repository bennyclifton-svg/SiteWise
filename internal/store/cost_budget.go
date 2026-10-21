package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/knowledge"
)

// budgetKeys are the profile rows the construction budget and its split are
// read from (docs/plans/2026-10-21-source-to-cost-plan-design.md, Part 2).
var budgetKeys = []string{"proj.budget_stated", "hdr.building_class", "hdr.subclass", "hdr.work_type", "hdr.scale.storeys", "proj.flag.basement"}

// readBudgetProfile reads the applied value of each budget key, preferring
// the whole-project part. A red or blank row is not applied.
func readBudgetProfile(ctx context.Context, tx pgx.Tx, org, project string) (map[string]string, map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT r.key, r.value, r.origin FROM profile_rows r JOIN project_parts p ON p.org_id=r.org_id AND p.site_id=r.site_id AND p.id=r.part_id
 WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.key=ANY($3) AND r.value_state='set' AND r.value<>'' AND r.band NOT IN ('red','blank','unchecked')
 ORDER BY (p.kind='whole') DESC`, org, project, budgetKeys)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	values, origins := map[string]string{}, map[string]string{}
	for rows.Next() {
		var key, value, origin string
		if err := rows.Scan(&key, &value, &origin); err != nil {
			return nil, nil, err
		}
		if _, seen := values[key]; !seen {
			values[key], origins[key] = value, origin
		}
	}
	return values, origins, rows.Err()
}

// statedConstructionBudget reads the stated budget and chooses its split from the
// profile. It is nil when the profile states no budget.
func statedConstructionBudget(cat *knowledge.Catalog, values, origins map[string]string) (*costs.ConstructionBudget, map[string]*big.Rat) {
	text := values["proj.budget_stated"]
	if text == "" {
		return nil, nil
	}
	stated, known := costs.ParseStatedBudget(text)
	b := &costs.ConstructionBudget{Stated: stated, Known: known, Origin: "document"}
	if origins["proj.budget_stated"] == "user" {
		b.Origin = "user"
	}
	if cat == nil {
		return b, nil
	}
	storeys, _ := strconv.Atoi(strings.TrimSpace(values["hdr.scale.storeys"]))
	sp := knowledge.SplitProfile{Class: values["hdr.building_class"], Subclass: values["hdr.subclass"], Storeys: storeys, Basement: values["proj.flag.basement"] == "yes"}
	for _, w := range strings.Split(values["hdr.work_type"], ",") {
		if w = strings.TrimSpace(w); w != "" {
			sp.WorkTypes = append(sp.WorkTypes, w)
		}
	}
	split, weights, ok := cat.ChooseBudgetSplit(sp)
	switch {
	case ok:
		b.Split, b.SplitConfidence, b.SplitBasis = split.Label, split.Confidence, split.Basis
	case sp.Class == "" || len(sp.WorkTypes) == 0:
		b.NoSplit = "Set the building class and work type in Profile to spread the budget."
	default:
		b.NoSplit = "No split is held for this building type and work type yet."
	}
	return b, weights
}

// spreadValue is what a spread writes into a line's budget provenance, so a
// later spread knows the value is still its own and may refresh it.
type spreadValue struct {
	Budget costs.Money `json:"budget"`
	Split  string      `json:"split"`
}

// ownSpread reports whether a budget value is an unedited spread.
func ownSpread(v costs.Value) (spreadValue, bool) {
	var p struct {
		Spread *spreadValue `json:"spread"`
	}
	if v.Origin != "assumption" || json.Unmarshal(v.Provenance, &p) != nil || p.Spread == nil {
		return spreadValue{}, false
	}
	return *p.Spread, true
}

// spreadConstructionBudget writes each construction line's share of the
// stated budget into lines whose budget is unknown or still an unedited
// spread. A typed figure, a cleared one, an awarded group and an excluded
// line are never written, and no other line moves when one is edited:
// each share is of the whole budget.
func (s *Store) spreadConstructionBudget(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan) (int, error) {
	cat := s.workCatalog()
	if p.Status != "draft" || cat == nil {
		return 0, nil
	}
	values, origins, err := readBudgetProfile(ctx, tx, org, project)
	if err != nil {
		return 0, err
	}
	b, weights := statedConstructionBudget(cat, values, origins)
	if b == nil || !b.Known || weights == nil {
		return 0, nil
	}
	splitID := ""
	for _, sp := range cat.BudgetSplits() {
		if sp.Label == b.Split {
			splitID = sp.ID
		}
	}
	var lines []costs.SpreadLine
	byID := map[string]costs.Item{}
	for _, el := range cat.CostElements() {
		w, ok := weights[el.System]
		if !ok || w.Sign() == 0 {
			continue
		}
		for _, i := range p.Items {
			if i.SourceKey == "works:"+el.System && i.ParentItemID == "" && i.Posting && !i.Excluded {
				lines = append(lines, costs.SpreadLine{ID: i.ID, Weight: w})
				byID[i.ID] = i
			}
		}
	}
	shares := costs.Spread(b.Stated.Amount, lines)
	total, _ := costs.Parse(string(b.Stated.Amount))
	n := 0
	for _, l := range lines {
		i, amount := byID[l.ID], shares[l.ID]
		v, has := i.Values["budget"]
		prior, own := ownSpread(v)
		writable := !has || v.ValueState == "unknown" && v.Origin != "user" || own
		if !writable || own && prior.Budget == b.Stated.Amount && prior.Split == splitID && v.Amount != nil && *v.Amount == amount {
			continue
		}
		pct, _ := new(big.Rat).Quo(new(big.Rat).Mul(mustRat(amount), big.NewRat(100, 1)), total).Float64()
		rationale := fmt.Sprintf("%.1f %% of the $%s construction budget, by the draft %s split (%s confidence); includes preliminaries and margin pro rata. %s", pct, groupDigits(b.Stated.Amount), strings.ToLower(b.Split), b.SplitConfidence, b.SplitBasis)
		provenance, _ := json.Marshal(map[string]any{"actor": actor, "spread": spreadValue{Budget: b.Stated.Amount, Split: splitID}, "budget_text": b.Stated.Text, "budget_origin": b.Origin})
		a := amount
		if err := writeCostValueWith(ctx, tx, org, project, p.ID, i.ID, "budget", costs.Value{Amount: &a, ValueState: "known", Origin: "assumption", Meaning: "allowance", Rationale: rationale}, provenance); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func mustRat(m costs.Money) *big.Rat {
	r, _ := costs.Parse(string(m))
	return r
}

// groupDigits writes whole dollars with thousands separators.
func groupDigits(m costs.Money) string {
	whole := strings.Split(string(m), ".")[0]
	var b strings.Builder
	for n, r := range whole {
		if n > 0 && (len(whole)-n)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// attachConstructionBudget puts the stated budget on a plan read, so the
// totals can compare it with the construction lines.
func (s *Store) attachConstructionBudget(ctx context.Context, tx pgx.Tx, org, project string, p *costs.Plan) error {
	values, origins, err := readBudgetProfile(ctx, tx, org, project)
	if err != nil {
		return err
	}
	p.Budget, _ = statedConstructionBudget(s.workCatalog(), values, origins)
	return nil
}
