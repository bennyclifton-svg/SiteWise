package costs

import (
	"encoding/json"
	"strings"
	"time"
)

// Estimate is retired: budget is the one planning figure. Old baselines may still
// hold estimate rows; they are read, never written.
var Metrics = []string{"budget", "commitment", "claimed_to_date"}

type Settings struct {
	Currency                string          `json:"currency"`
	TaxBasis                string          `json:"tax_basis"`
	TaxRate                 *Money          `json:"tax_rate"`
	PriceDate               *string         `json:"price_date"`
	Coverage                string          `json:"coverage"`
	FundingTarget           *Money          `json:"funding_target"`
	FundingTargetProvenance json.RawMessage `json:"funding_target_provenance"`
}
type Plan struct {
	Totals            *Totals `json:"totals,omitempty"`
	BaselineVersionID string  `json:"baseline_version_id"`
	ID                string  `json:"id"`
	ProjectID         string  `json:"project_id"`
	Revision          int     `json:"revision"`
	Version           int64   `json:"version"`
	Status            string  `json:"status"`
	Settings
	Suggestions []Suggestion `json:"suggestions"`
	Dismissed   []Suggestion `json:"dismissed"`
	Systems     []CostSystem `json:"systems"`
	Items       []Item       `json:"items"`
	Links       []Link       `json:"links"`
	// Budget is the construction budget the profile states, read on every
	// plan read; nil when the profile has none.
	Budget *ConstructionBudget `json:"construction_budget,omitempty"`
}
type CostSystem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Content struct {
	SystemID       string `json:"system_id"`
	ParentItemID   string `json:"parent_item_id"`
	Code           string `json:"code"`
	Label          string `json:"label"`
	LineKind       string `json:"line_kind"`
	Category       string `json:"category"`
	WorkItemID     string `json:"work_item_id"`
	PackageID      string `json:"package_id"`
	PackageStageID string `json:"package_stage_id"`
	Posting        bool   `json:"posting"`
	Quantity       *Money `json:"quantity"`
	Unit           string `json:"unit"`
	Rate           *Money `json:"rate"`
	RateBasis      string `json:"rate_basis"`
	Excluded       bool   `json:"excluded"`
	Origin         string `json:"origin"`
	Meaning        string `json:"meaning"`
	Rationale      string `json:"rationale"`
}
type Item struct {
	DisplayOrder      *int   `json:"display_order,omitempty"`
	ResolvedPackageID string `json:"resolved_package_id"`
	ID                string `json:"cost_item_id"`
	Version           int64  `json:"version"`
	Content
	SourceKey        string           `json:"source_key"`
	ResolvedCategory string           `json:"resolved_category"`
	ScopeWorkItems   []ScopeWork      `json:"scope_work_items"`
	PartID           string           `json:"part_id"`
	Values           map[string]Value `json:"values"`
}
type Value struct {
	Amount     *Money          `json:"amount"`
	Low        *Money          `json:"low"`
	High       *Money          `json:"high"`
	ValueState string          `json:"value_state"`
	AsOf       *string         `json:"as_of"`
	Origin     string          `json:"origin"`
	Meaning    string          `json:"meaning"`
	Rationale  string          `json:"rationale"`
	Provenance json.RawMessage `json:"provenance"`
	Version    int64           `json:"version"`
}
type Link struct {
	ScopeID string `json:"package_scope_item_id"`
	CostID  string `json:"cost_item_id"`
}

func ValidDate(s *string) bool {
	if s == nil {
		return true
	}
	_, e := time.Parse("2006-01-02", *s)
	return e == nil
}
func provenance(origin, meaning string) bool {
	return (origin == "user" || origin == "assumption") && (meaning == "stated" || meaning == "requirement" || meaning == "allowance" || meaning == "forecast")
}
func ValidateContent(c Content) error {
	if len(strings.TrimSpace(c.Label)) == 0 || len(c.Label) > 500 || len(c.Code) > 100 || len(c.Rationale) > 10000 {
		return ErrInvalid
	}
	if !provenance(c.Origin, c.Meaning) {
		return ErrInvalid
	}
	switch c.LineKind {
	case "works":
		if (c.WorkItemID == "") == (c.SystemID == "") || c.PackageStageID != "" || c.Category != "" {
			return ErrInvalid
		}
	case "fee":
		if c.PackageID == "" || c.SystemID != "" || c.WorkItemID != "" || c.Category != "" {
			return ErrInvalid
		}
	case "project_wide":
		if c.SystemID != "" || c.WorkItemID != "" || c.PackageID != "" || c.PackageStageID != "" {
			return ErrInvalid
		}
		switch c.Category {
		case "contingency", "escalation", "authority_fees", "exclusions", "other":
		default:
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	if (c.Quantity == nil) != (c.Rate == nil) {
		return ErrInvalid
	}
	if c.Quantity != nil {
		if c.Unit == "" || c.RateBasis == "" {
			return ErrInvalid
		}
		if _, e := Multiply(string(*c.Quantity), string(*c.Rate)); e != nil {
			return e
		}
		parts := strings.Split(strings.TrimPrefix(string(*c.Rate), "-"), ".")
		if len(strings.TrimLeft(parts[0], "0")) > 14 {
			return ErrInvalid
		}
		if len(parts) > 1 && len(parts[1]) > 4 {
			return ErrInvalid
		}
	}
	return nil
}
func ValidateValue(metric string, v *Value) error {
	found := false
	for _, m := range Metrics {
		found = found || m == metric
	}
	if !found || !provenance(v.Origin, v.Meaning) || len(v.Rationale) > 10000 {
		return ErrInvalid
	}
	if !ValidDate(v.AsOf) || (metric == "claimed_to_date" && (v.AsOf == nil || v.Origin != "user")) {
		return ErrInvalid
	}
	if v.ValueState != "known" && v.ValueState != "unknown" {
		return ErrInvalid
	}
	if v.ValueState == "unknown" && (v.Amount != nil || v.Low != nil || v.High != nil) {
		return ErrInvalid
	}
	if (v.Low == nil) != (v.High == nil) {
		return ErrInvalid
	}
	if v.ValueState == "known" && v.Amount == nil && v.Low == nil {
		return ErrInvalid
	}
	for _, p := range []*Money{v.Amount, v.Low, v.High} {
		if p != nil {
			n, e := Normalize(*p)
			if e != nil {
				return e
			}
			*p = n
		}
	}
	if v.Low != nil && (Compare(*v.Low, *v.High) > 0 || (v.Amount != nil && (Compare(*v.Amount, *v.Low) < 0 || Compare(*v.Amount, *v.High) > 0))) {
		return ErrInvalid
	}
	return nil
}
func ValidateSettings(s *Settings) error {
	if len(s.Currency) != 3 || strings.Trim(s.Currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" || (s.TaxBasis != "ex_tax" && s.TaxBasis != "inc_tax") || !ValidDate(s.PriceDate) || len(s.Coverage) > 10000 {
		return ErrInvalid
	}
	if s.TaxRate != nil {
		if _, _, e := Tax("1.00", s.TaxBasis, string(*s.TaxRate)); e != nil {
			return e
		}
		parts := strings.Split(string(*s.TaxRate), ".")
		if len(parts) > 1 && len(parts[1]) > 4 {
			return ErrInvalid
		}
	}
	if s.FundingTarget != nil {
		v, e := Normalize(*s.FundingTarget)
		if e != nil {
			return e
		}
		*s.FundingTarget = v
	}
	if len(s.FundingTargetProvenance) == 0 {
		s.FundingTargetProvenance = json.RawMessage(`{}`)
	}
	var obj map[string]any
	if json.Unmarshal(s.FundingTargetProvenance, &obj) != nil || obj == nil {
		return ErrInvalid
	}
	return nil
}
