package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/knowledge"
)

// budgetKeys are the profile rows the construction budget and its split are
// read from (docs/plans/2026-10-21-source-to-cost-plan-design.md, Part 2).
var budgetKeys = []string{"proj.budget_stated", "hdr.building_class", "hdr.subclass", "hdr.work_type", "hdr.scale.storeys", "proj.flag.basement", "proj.contingency_stated", "proj.owner_supply_stated", "proj.exclusions_stated"}

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

// briefValue is what a brief figure writes into a line's provenance, so a
// later prepare knows the value is still its own.
type briefValue struct {
	Key    string `json:"key"`
	Stated string `json:"stated"`
}

func ownBrief(v costs.Value) (briefValue, bool) {
	var p struct {
		Brief *briefValue `json:"brief"`
	}
	if v.Origin != "assumption" || json.Unmarshal(v.Provenance, &p) != nil || p.Brief == nil {
		return briefValue{}, false
	}
	return *p.Brief, true
}

// applyBriefFigures puts the brief's other figures in the plan: the stated
// contingency on Construction contingency, one line per owner-supplied item
// (outside the construction budget), and the budget's exclusions on the
// Exclusions line. Like the spread, it writes only a budget that is unknown
// or still its own, and relabels Exclusions only while the label is unedited.
func (s *Store) applyBriefFigures(ctx context.Context, tx pgx.Tx, org, project, actor string, p costs.Plan) (int, error) {
	if p.Status != "draft" {
		return 0, nil
	}
	values, origins, err := readBudgetProfile(ctx, tx, org, project)
	if err != nil {
		return 0, err
	}
	bySource := map[string]costs.Item{}
	for _, i := range p.Items {
		if i.SourceKey != "" {
			bySource[i.SourceKey] = i
		}
	}
	n := 0
	write := func(i costs.Item, key, stated string, amount costs.Money, rationale string) error {
		v, has := i.Values["budget"]
		prior, own := ownBrief(v)
		if has && !(v.ValueState == "unknown" && v.Origin != "user") && !own {
			return nil
		}
		if own && prior.Stated == stated && v.Amount != nil && *v.Amount == amount {
			return nil
		}
		provenance, _ := json.Marshal(map[string]any{"actor": actor, "brief": briefValue{Key: key, Stated: stated}, "origin": origins[key]})
		a := amount
		n++
		return writeCostValueWith(ctx, tx, org, project, p.ID, i.ID, "budget", costs.Value{Amount: &a, ValueState: "known", Origin: "assumption", Meaning: "allowance", Rationale: rationale}, provenance)
	}
	if text := values["proj.contingency_stated"]; text != "" {
		if c, ok := costs.ParseStatedBudget(text); ok {
			if i, ok := bySource["pw:construction-contingency"]; ok && i.Posting && !i.Excluded {
				if err := write(i, "proj.contingency_stated", text, c.Amount, fmt.Sprintf("Contingency as a source document states it: %q.", text)); err != nil {
					return n, err
				}
			}
		}
	}
	if text := values["proj.owner_supply_stated"]; text != "" {
		for _, item := range costs.ParseOwnerSupply(text) {
			key := "brief:owner-supply:" + slug(item.Label)
			i, ok := bySource[key]
			if !ok {
				c := costs.Content{Label: "Owner supply: " + item.Label, LineKind: "project_wide", Category: "other", Posting: true, Origin: "assumption", Meaning: "allowance",
					Rationale: "The owner supplies this, outside the construction budget, as a source document states: " + clipText(text, 300)}
				id := newID()
				if err := s.insertCostItem(ctx, tx, org, project, p.ID, actor, id, c); err != nil {
					return n, err
				}
				if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET source_key=$4 WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, id, key); err != nil {
					return n, err
				}
				i = costs.Item{ID: id, Content: c, Values: map[string]costs.Value{}}
				n++
			}
			if item.Amount != "" && i.Posting && !i.Excluded {
				if err := write(i, "proj.owner_supply_stated", text, item.Amount, fmt.Sprintf("Owner allowance for %s as a source document states it; outside the construction budget.", item.Label)); err != nil {
					return n, err
				}
			}
		}
	}
	if text := values["proj.exclusions_stated"]; text != "" {
		if i, ok := bySource["pw:exclusions"]; ok && (i.Label == "Exclusions" || strings.HasPrefix(i.Label, "Exclusions: ") && i.Origin == "assumption") {
			label := "Exclusions: " + clipText(text, 400)
			if i.Label != label {
				c := i.Content
				c.Label, c.Origin = label, "assumption"
				c.Rationale = "The construction budget excludes these, as a source document states."
				if err := writeCostContent(ctx, tx, org, project, p.ID, actor, i.ID, c, true); err != nil {
					return n, err
				}
				n++
			}
		}
	}
	return n, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
